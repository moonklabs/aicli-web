package claude

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// LoadBalancer는 부하 분산을 담당합니다
type LoadBalancer struct {
	pool   *AdvancedSessionPool
	config LoadBalancingConfig
	
	// 라운드 로빈 카운터
	roundRobinCounter atomic.Uint64
	
	// 세션 어피니티 매핑
	affinityMap map[string]string // key -> sessionID
	affinityMutex sync.RWMutex
	
	// 가중치 캐시
	weightCache map[string]float64
	weightMutex sync.RWMutex
	
	// 응답 시간 추적
	responseTimeTracker *ResponseTimeTracker
	
	// 커넥션 카운터
	connectionCounter map[string]int64
	connectionMutex   sync.RWMutex
}

// ResponseTimeTracker는 응답 시간을 추적합니다
type ResponseTimeTracker struct {
	sessions map[string]*SessionResponseTime
	mutex    sync.RWMutex
}

// SessionResponseTime은 세션별 응답 시간 정보입니다
type SessionResponseTime struct {
	SessionID       string        `json:"session_id"`
	AverageTime     time.Duration `json:"average_time"`
	RecentTimes     []time.Duration `json:"recent_times"`
	TotalRequests   int64         `json:"total_requests"`
	LastUpdate      time.Time     `json:"last_update"`
	WindowSize      int           `json:"window_size"`
}

// SessionWeight는 세션 가중치 정보입니다
type SessionWeight struct {
	SessionID      string  `json:"session_id"`
	Weight         float64 `json:"weight"`
	LoadScore      float64 `json:"load_score"`
	HealthScore    float64 `json:"health_score"`
	ResponseScore  float64 `json:"response_score"`
	LastCalculated time.Time `json:"last_calculated"`
}

// AffinityKey는 어피니티 키 생성기입니다
type AffinityKey struct {
	UserID      string `json:"user_id"`
	WorkspaceID string `json:"workspace_id"`
	ProjectID   string `json:"project_id"`
	Custom      string `json:"custom"`
}

// NewLoadBalancer는 새로운 로드 밸런서를 생성합니다
func NewLoadBalancer(pool *AdvancedSessionPool, config LoadBalancingConfig) *LoadBalancer {
	lb := &LoadBalancer{
		pool:                pool,
		config:              config,
		affinityMap:         make(map[string]string),
		weightCache:         make(map[string]float64),
		connectionCounter:   make(map[string]int64),
		responseTimeTracker: NewResponseTimeTracker(),
	}
	
	return lb
}

// NewResponseTimeTracker는 새로운 응답 시간 추적기를 생성합니다
func NewResponseTimeTracker() *ResponseTimeTracker {
	return &ResponseTimeTracker{
		sessions: make(map[string]*SessionResponseTime),
	}
}

// SelectSession은 최적의 세션을 선택합니다
func (lb *LoadBalancer) SelectSession(ctx context.Context, config SessionConfig) (*PooledSession, error) {
	// 1. 어피니티 세션 확인
	if lb.config.SessionAffinity {
		if session := lb.FindAffinitySession(config); session != nil {
			return session, nil
		}
	}
	
	// 2. 전략에 따른 세션 선택
	switch lb.config.Strategy {
	case RoundRobin:
		return lb.selectRoundRobin(ctx, config)
	case LeastConnections:
		return lb.selectLeastConnections(ctx, config)
	case WeightedRoundRobin:
		return lb.selectWeightedRoundRobin(ctx, config)
	case ResourceBased:
		return lb.selectResourceBased(ctx, config)
	case ResponseTimeBased:
		return lb.selectResponseTimeBased(ctx, config)
	default:
		return lb.selectRoundRobin(ctx, config)
	}
}

// FindAffinitySession은 어피니티 세션을 찾습니다
func (lb *LoadBalancer) FindAffinitySession(config SessionConfig) *PooledSession {
	if !lb.config.SessionAffinity {
		return nil
	}
	
	affinityKey := lb.generateAffinityKey(config)
	
	lb.affinityMutex.RLock()
	sessionID, exists := lb.affinityMap[affinityKey]
	lb.affinityMutex.RUnlock()
	
	if !exists {
		return nil
	}
	
	// 세션이 여전히 유효한지 확인
	if session := lb.getAvailableSession(sessionID); session != nil {
		// 어피니티 만료 시간 확인
		if lb.isAffinityValid(affinityKey) {
			return session
		} else {
			// 만료된 어피니티 제거
			lb.RemoveAffinity(affinityKey)
		}
	}
	
	return nil
}

// SetAffinity는 세션 어피니티를 설정합니다
func (lb *LoadBalancer) SetAffinity(config SessionConfig, sessionID string) {
	if !lb.config.SessionAffinity {
		return
	}
	
	affinityKey := lb.generateAffinityKey(config)
	
	lb.affinityMutex.Lock()
	defer lb.affinityMutex.Unlock()
	
	lb.affinityMap[affinityKey] = sessionID
}

// RemoveAffinity는 세션 어피니티를 제거합니다
func (lb *LoadBalancer) RemoveAffinity(affinityKey string) {
	lb.affinityMutex.Lock()
	defer lb.affinityMutex.Unlock()
	
	delete(lb.affinityMap, affinityKey)
}

// RecordResponseTime은 응답 시간을 기록합니다
func (lb *LoadBalancer) RecordResponseTime(sessionID string, responseTime time.Duration) {
	lb.responseTimeTracker.RecordTime(sessionID, responseTime)
}

// UpdateConnectionCount는 연결 수를 업데이트합니다
func (lb *LoadBalancer) UpdateConnectionCount(sessionID string, delta int64) {
	lb.connectionMutex.Lock()
	defer lb.connectionMutex.Unlock()
	
	lb.connectionCounter[sessionID] += delta
	if lb.connectionCounter[sessionID] < 0 {
		lb.connectionCounter[sessionID] = 0
	}
}

// GetSessionWeights는 모든 세션의 가중치를 반환합니다
func (lb *LoadBalancer) GetSessionWeights() []SessionWeight {
	availableSessions := lb.getAvailableSessions()
	weights := make([]SessionWeight, 0, len(availableSessions))
	
	for _, session := range availableSessions {
		weight := lb.calculateSessionWeight(session)
		weights = append(weights, weight)
	}
	
	// 가중치순으로 정렬
	sort.Slice(weights, func(i, j int) bool {
		return weights[i].Weight > weights[j].Weight
	})
	
	return weights
}

// 전략별 세션 선택 메서드들

func (lb *LoadBalancer) selectRoundRobin(ctx context.Context, config SessionConfig) (*PooledSession, error) {
	availableSessions := lb.getAvailableSessions()
	if len(availableSessions) == 0 {
		return nil, fmt.Errorf("no available sessions")
	}
	
	// 라운드 로빈 카운터 증가
	counter := lb.roundRobinCounter.Add(1)
	index := (counter - 1) % uint64(len(availableSessions))
	
	session := availableSessions[index]
	lb.updateSessionUsage(session)
	
	return session, nil
}

func (lb *LoadBalancer) selectLeastConnections(ctx context.Context, config SessionConfig) (*PooledSession, error) {
	availableSessions := lb.getAvailableSessions()
	if len(availableSessions) == 0 {
		return nil, fmt.Errorf("no available sessions")
	}
	
	var bestSession *PooledSession
	minConnections := int64(math.MaxInt64)
	
	lb.connectionMutex.RLock()
	for _, session := range availableSessions {
		connections := lb.connectionCounter[session.ID]
		if connections < minConnections {
			minConnections = connections
			bestSession = session
		}
	}
	lb.connectionMutex.RUnlock()
	
	if bestSession == nil {
		bestSession = availableSessions[0]
	}
	
	lb.updateSessionUsage(bestSession)
	return bestSession, nil
}

func (lb *LoadBalancer) selectWeightedRoundRobin(ctx context.Context, config SessionConfig) (*PooledSession, error) {
	availableSessions := lb.getAvailableSessions()
	if len(availableSessions) == 0 {
		return nil, fmt.Errorf("no available sessions")
	}
	
	// 가중치 기반 선택
	weightedSessions := lb.buildWeightedSessionList(availableSessions)
	if len(weightedSessions) == 0 {
		return lb.selectRoundRobin(ctx, config) // fallback
	}
	
	// 가중치 기반 라운드 로빈
	counter := lb.roundRobinCounter.Add(1)
	index := (counter - 1) % uint64(len(weightedSessions))
	
	session := weightedSessions[index]
	lb.updateSessionUsage(session)
	
	return session, nil
}

func (lb *LoadBalancer) selectResourceBased(ctx context.Context, config SessionConfig) (*PooledSession, error) {
	availableSessions := lb.getAvailableSessions()
	if len(availableSessions) == 0 {
		return nil, fmt.Errorf("no available sessions")
	}
	
	var bestSession *PooledSession
	bestScore := float64(-1)
	
	for _, session := range availableSessions {
		score := lb.calculateResourceScore(session)
		if score > bestScore {
			bestScore = score
			bestSession = session
		}
	}
	
	if bestSession == nil {
		bestSession = availableSessions[0]
	}
	
	lb.updateSessionUsage(bestSession)
	return bestSession, nil
}

func (lb *LoadBalancer) selectResponseTimeBased(ctx context.Context, config SessionConfig) (*PooledSession, error) {
	availableSessions := lb.getAvailableSessions()
	if len(availableSessions) == 0 {
		return nil, fmt.Errorf("no available sessions")
	}
	
	var bestSession *PooledSession
	bestTime := time.Duration(math.MaxInt64)
	
	for _, session := range availableSessions {
		avgTime := lb.responseTimeTracker.GetAverageTime(session.ID)
		if avgTime > 0 && avgTime < bestTime {
			bestTime = avgTime
			bestSession = session
		}
	}
	
	// 응답 시간 데이터가 없는 경우 라운드 로빈으로 fallback
	if bestSession == nil {
		return lb.selectRoundRobin(ctx, config)
	}
	
	lb.updateSessionUsage(bestSession)
	return bestSession, nil
}

// 유틸리티 메서드들

func (lb *LoadBalancer) getAvailableSessions() []*PooledSession {
	baseStats := lb.pool.basePool.GetPoolStats()
	if baseStats.IdleSessions == 0 {
		return []*PooledSession{}
	}
	
	// 실제 구현에서는 basePool에서 유휴 세션 목록을 가져와야 함
	// 여기서는 시뮬레이션
	return lb.getIdleSessionsFromPool()
}

func (lb *LoadBalancer) getIdleSessionsFromPool() []*PooledSession {
	// 이 메서드는 실제로는 basePool의 내부 구조에 접근해야 함
	// 여기서는 더미 구현
	return []*PooledSession{}
}

func (lb *LoadBalancer) getAvailableSession(sessionID string) *PooledSession {
	// 특정 세션이 사용 가능한지 확인
	// 실제 구현에서는 basePool에서 세션 상태 확인
	return nil
}

func (lb *LoadBalancer) generateAffinityKey(config SessionConfig) string {
	// WorkspaceID를 기본 어피니티 키로 사용
	affinityKey := AffinityKey{
		WorkspaceID: config.WorkspaceID,
	}
	
	// 사용자 정보가 있다면 추가
	if userID, ok := config.Environment["USER_ID"]; ok {
		affinityKey.UserID = userID
	}
	
	// 프로젝트 정보가 있다면 추가
	if projectID, ok := config.Environment["PROJECT_ID"]; ok {
		affinityKey.ProjectID = projectID
	}
	
	// 해시 생성
	h := fnv.New64a()
	h.Write([]byte(affinityKey.UserID + affinityKey.WorkspaceID + affinityKey.ProjectID))
	
	return fmt.Sprintf("affinity_%x", h.Sum64())
}

func (lb *LoadBalancer) isAffinityValid(affinityKey string) bool {
	// 어피니티 유효성 확인 (시간 기반)
	// 실제 구현에서는 어피니티 생성 시간을 추적해야 함
	return true // 임시로 항상 유효
}

func (lb *LoadBalancer) calculateSessionWeight(session *PooledSession) SessionWeight {
	weight := SessionWeight{
		SessionID:      session.ID,
		Weight:         1.0,
		LastCalculated: time.Now(),
	}
	
	// 부하 점수 계산 (낮을수록 좋음)
	weight.LoadScore = lb.calculateLoadScore(session)
	
	// 헬스 점수 계산 (높을수록 좋음)
	weight.HealthScore = lb.calculateHealthScore(session)
	
	// 응답 시간 점수 계산 (낮을수록 좋음)
	weight.ResponseScore = lb.calculateResponseScore(session)
	
	// 전체 가중치 계산
	weight.Weight = weight.HealthScore * (1.0 / (1.0 + weight.LoadScore + weight.ResponseScore))
	
	return weight
}

func (lb *LoadBalancer) calculateLoadScore(session *PooledSession) float64 {
	lb.connectionMutex.RLock()
	connections := lb.connectionCounter[session.ID]
	lb.connectionMutex.RUnlock()
	
	// 연결 수를 기반으로 부하 점수 계산
	return float64(connections) / 10.0 // 정규화
}

func (lb *LoadBalancer) calculateHealthScore(session *PooledSession) float64 {
	// 세션 헬스 점수 계산
	// 실제 구현에서는 더 정교한 헬스 체크 필요
	
	if session.State == SessionStateError {
		return 0.1
	}
	
	if session.State == SessionStateClosed {
		return 0.0
	}
	
	// 사용 횟수 기반 점수 (적당한 사용이 좋음)
	useCount := float64(session.useCount)
	if useCount < 10 {
		return 0.8 + useCount*0.02 // 신규 세션 보너스
	} else if useCount > 100 {
		return math.Max(0.3, 1.0-useCount*0.001) // 과사용 페널티
	}
	
	return 1.0
}

func (lb *LoadBalancer) calculateResponseScore(session *PooledSession) float64 {
	avgTime := lb.responseTimeTracker.GetAverageTime(session.ID)
	
	if avgTime == 0 {
		return 0.5 // 데이터 없음
	}
	
	// 응답 시간을 점수로 변환 (ms 단위)
	timeMs := float64(avgTime.Milliseconds())
	
	if timeMs < 100 {
		return 0.1 // 매우 빠름
	} else if timeMs < 500 {
		return 0.3 // 빠름
	} else if timeMs < 1000 {
		return 0.5 // 보통
	} else if timeMs < 5000 {
		return 0.8 // 느림
	} else {
		return 1.0 // 매우 느림
	}
}

func (lb *LoadBalancer) calculateResourceScore(session *PooledSession) float64 {
	// 리소스 기반 점수 계산
	weight := lb.calculateSessionWeight(session)
	return weight.Weight
}

func (lb *LoadBalancer) buildWeightedSessionList(sessions []*PooledSession) []*PooledSession {
	var weightedList []*PooledSession
	
	for _, session := range sessions {
		weight := lb.calculateSessionWeight(session)
		
		// 가중치에 비례하여 목록에 추가
		count := int(math.Ceil(weight.Weight * 10))
		for i := 0; i < count; i++ {
			weightedList = append(weightedList, session)
		}
	}
	
	// 랜덤 셔플로 편향 방지
	rand.Shuffle(len(weightedList), func(i, j int) {
		weightedList[i], weightedList[j] = weightedList[j], weightedList[i]
	})
	
	return weightedList
}

func (lb *LoadBalancer) updateSessionUsage(session *PooledSession) {
	// 세션 사용 업데이트
	lb.UpdateConnectionCount(session.ID, 1)
	
	// 어피니티 설정 (필요한 경우)
	if lb.config.SessionAffinity {
		// 세션 설정에서 어피니티 키 생성 및 설정
		// 실제 구현에서는 config 정보 필요
	}
}

// ResponseTimeTracker 메서드들

func (rt *ResponseTimeTracker) RecordTime(sessionID string, responseTime time.Duration) {
	rt.mutex.Lock()
	defer rt.mutex.Unlock()
	
	sessionTime, exists := rt.sessions[sessionID]
	if !exists {
		sessionTime = &SessionResponseTime{
			SessionID:     sessionID,
			RecentTimes:   make([]time.Duration, 0, 10),
			WindowSize:    10,
		}
		rt.sessions[sessionID] = sessionTime
	}
	
	// 최근 시간 목록에 추가
	sessionTime.RecentTimes = append(sessionTime.RecentTimes, responseTime)
	
	// 윈도우 크기 제한
	if len(sessionTime.RecentTimes) > sessionTime.WindowSize {
		sessionTime.RecentTimes = sessionTime.RecentTimes[1:]
	}
	
	// 평균 시간 계산
	var total time.Duration
	for _, t := range sessionTime.RecentTimes {
		total += t
	}
	sessionTime.AverageTime = total / time.Duration(len(sessionTime.RecentTimes))
	
	sessionTime.TotalRequests++
	sessionTime.LastUpdate = time.Now()
}

func (rt *ResponseTimeTracker) GetAverageTime(sessionID string) time.Duration {
	rt.mutex.RLock()
	defer rt.mutex.RUnlock()
	
	if sessionTime, exists := rt.sessions[sessionID]; exists {
		return sessionTime.AverageTime
	}
	
	return 0
}

func (rt *ResponseTimeTracker) GetSessionResponseTime(sessionID string) *SessionResponseTime {
	rt.mutex.RLock()
	defer rt.mutex.RUnlock()
	
	if sessionTime, exists := rt.sessions[sessionID]; exists {
		// 복사본 반환
		copy := *sessionTime
		copy.RecentTimes = make([]time.Duration, len(sessionTime.RecentTimes))
		copy.Copy(sessionTime.RecentTimes, copy.RecentTimes)
		return &copy
	}
	
	return nil
}

func (rt *ResponseTimeTracker) RemoveSession(sessionID string) {
	rt.mutex.Lock()
	defer rt.mutex.Unlock()
	
	delete(rt.sessions, sessionID)
}