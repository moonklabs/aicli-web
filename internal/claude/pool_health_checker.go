package claude

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// PoolPoolHealthChecker는 세션 풀의 헬스를 체크합니다
type PoolHealthChecker struct {
	pool   *AdvancedSessionPool
	config HealthCheckConfig
	
	// 헬스 상태 추적
	sessionHealth map[string]*SessionHealth
	healthMutex   sync.RWMutex
	
	// 전체 헬스 상태
	overallHealth atomic.Value // *OverallHealth
	
	// 상태 관리
	running atomic.Bool
	
	// 생명주기 관리
	ctx    context.Context
	cancel context.CancelFunc
	ticker *time.Ticker
}

// SessionHealth는 개별 세션의 헬스 상태입니다
type SessionHealth struct {
	SessionID         string                `json:"session_id"`
	Status            PoolHealthStatus          `json:"status"`
	LastCheckTime     time.Time             `json:"last_check_time"`
	ConsecutiveFailures int                 `json:"consecutive_failures"`
	ConsecutiveSuccess  int                 `json:"consecutive_success"`
	TotalChecks       int64                 `json:"total_checks"`
	SuccessfulChecks  int64                 `json:"successful_checks"`
	FailedChecks      int64                 `json:"failed_checks"`
	AverageResponseTime time.Duration       `json:"average_response_time"`
	LastError         string                `json:"last_error,omitempty"`
	HealthScore       float64               `json:"health_score"`
	Checks            []HealthCheckResult   `json:"checks"`
}

// OverallHealth는 전체 풀의 헬스 상태입니다
type OverallHealth struct {
	Status            PoolHealthStatus    `json:"status"`
	HealthySessions   int             `json:"healthy_sessions"`
	UnhealthySessions int             `json:"unhealthy_sessions"`
	TotalSessions     int             `json:"total_sessions"`
	HealthScore       float64         `json:"health_score"`
	LastUpdate        time.Time       `json:"last_update"`
	Issues            []HealthIssue   `json:"issues"`
}

// HealthCheckResult는 개별 헬스 체크 결과입니다
type HealthCheckResult struct {
	Timestamp    time.Time     `json:"timestamp"`
	Success      bool          `json:"success"`
	ResponseTime time.Duration `json:"response_time"`
	ErrorMessage string        `json:"error_message,omitempty"`
	CheckType    CheckType     `json:"check_type"`
}

// HealthIssue는 헬스 이슈 정보입니다
type HealthIssue struct {
	Type        IssueType     `json:"type"`
	Severity    IssueSeverity `json:"severity"`
	Description string        `json:"description"`
	SessionID   string        `json:"session_id,omitempty"`
	Timestamp   time.Time     `json:"timestamp"`
	Count       int           `json:"count"`
}

// PoolHealthStatus는 헬스 상태입니다
type PoolHealthStatus int

const (
	HealthUnknown PoolHealthStatus = iota
	HealthHealthy
	HealthWarning
	HealthCritical
	HealthFailed
)

// CheckType은 체크 타입입니다
type CheckType int

const (
	CheckPing CheckType = iota
	CheckProcess
	CheckMemory
	CheckResponse
	CheckLoad
)

// IssueType은 이슈 타입입니다
type IssueType int

const (
	IssueHighLatency IssueType = iota
	IssueMemoryLeak
	IssueProcessDead
	IssueHighErrorRate
	IssueResourceExhaustion
)

// IssueSeverity는 이슈 심각도입니다
type IssueSeverity int

const (
	PoolPoolSeverityLow IssueSeverity = iota
	PoolPoolSeverityMedium
	PoolPoolSeverityHigh
	PoolPoolSeverityCritical
)

// NewPoolHealthChecker는 새로운 헬스 체커를 생성합니다
func NewPoolHealthChecker(pool *AdvancedSessionPool, config HealthCheckConfig) *PoolHealthChecker {
	ctx, cancel := context.WithCancel(context.Background())
	
	hc := &PoolHealthChecker{
		pool:          pool,
		config:        config,
		sessionHealth: make(map[string]*SessionHealth),
		ctx:           ctx,
		cancel:        cancel,
		ticker:        time.NewTicker(config.Interval),
	}
	
	// 초기 전체 헬스 상태 설정
	hc.overallHealth.Store(&OverallHealth{
		Status:     HealthUnknown,
		LastUpdate: time.Now(),
	})
	
	return hc
}

// Start는 헬스 체커를 시작합니다
func (hc *PoolHealthChecker) Start() {
	if !hc.running.CompareAndSwap(false, true) {
		return // 이미 실행 중
	}
	
	go hc.healthCheckLoop()
}

// Stop은 헬스 체커를 중지합니다
func (hc *PoolHealthChecker) Stop() {
	if !hc.running.CompareAndSwap(true, false) {
		return // 이미 중지됨
	}
	
	hc.cancel()
	hc.ticker.Stop()
}

// GetOverallHealth는 전체 헬스 상태를 반환합니다
func (hc *PoolHealthChecker) GetOverallHealth() *OverallHealth {
	health := hc.overallHealth.Load().(*OverallHealth)
	
	// 복사본 반환
	copy := *health
	copy.Issues = make([]HealthIssue, len(health.Issues))
	copy(copy.Issues, health.Issues)
	
	return &copy
}

// GetSessionHealth는 세션별 헬스 상태를 반환합니다
func (hc *PoolHealthChecker) GetSessionHealth(sessionID string) *SessionHealth {
	hc.healthMutex.RLock()
	defer hc.healthMutex.RUnlock()
	
	if health, exists := hc.sessionHealth[sessionID]; exists {
		// 복사본 반환
		copy := *health
		copy.Checks = make([]HealthCheckResult, len(health.Checks))
		copy(copy.Checks, health.Checks)
		return &copy
	}
	
	return nil
}

// GetAllSessionHealth는 모든 세션의 헬스 상태를 반환합니다
func (hc *PoolHealthChecker) GetAllSessionHealth() map[string]*SessionHealth {
	hc.healthMutex.RLock()
	defer hc.healthMutex.RUnlock()
	
	result := make(map[string]*SessionHealth)
	for sessionID, health := range hc.sessionHealth {
		copy := *health
		copy.Checks = make([]HealthCheckResult, len(health.Checks))
		copy(copy.Checks, health.Checks)
		result[sessionID] = &copy
	}
	
	return result
}

// CheckSessionHealth는 특정 세션의 헬스를 체크합니다
func (hc *PoolHealthChecker) CheckSessionHealth(sessionID string) *HealthCheckResult {
	startTime := time.Now()
	
	// 세션 존재 여부 확인
	session := hc.getSession(sessionID)
	if session == nil {
		return &HealthCheckResult{
			Timestamp:    startTime,
			Success:      false,
			ResponseTime: time.Since(startTime),
			ErrorMessage: "session not found",
			CheckType:    CheckPing,
		}
	}
	
	// 다양한 헬스 체크 수행
	checks := []HealthCheckResult{
		hc.checkSessionPing(session),
		hc.checkSessionProcess(session),
		hc.checkSessionMemory(session),
		hc.checkSessionResponse(session),
		hc.checkSessionLoad(session),
	}
	
	// 전체 결과 집계
	overallSuccess := true
	var totalResponseTime time.Duration
	var errors []string
	
	for _, check := range checks {
		if !check.Success {
			overallSuccess = false
			if check.ErrorMessage != "" {
				errors = append(errors, check.ErrorMessage)
			}
		}
		totalResponseTime += check.ResponseTime
	}
	
	result := &HealthCheckResult{
		Timestamp:    startTime,
		Success:      overallSuccess,
		ResponseTime: totalResponseTime / time.Duration(len(checks)),
		CheckType:    CheckPing,
	}
	
	if len(errors) > 0 {
		result.ErrorMessage = fmt.Sprintf("%v", errors)
	}
	
	// 헬스 상태 업데이트
	hc.updateSessionHealth(sessionID, result)
	
	return result
}

// 내부 메서드들

func (hc *PoolHealthChecker) healthCheckLoop() {
	for {
		select {
		case <-hc.ctx.Done():
			return
		case <-hc.ticker.C:
			hc.performHealthChecks()
		}
	}
}

func (hc *PoolHealthChecker) performHealthChecks() {
	// 활성 세션 목록 가져오기
	sessions := hc.getActiveSessions()
	
	// 각 세션에 대해 헬스 체크 수행
	var wg sync.WaitGroup
	for _, sessionID := range sessions {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			hc.CheckSessionHealth(id)
		}(sessionID)
	}
	
	wg.Wait()
	
	// 전체 헬스 상태 업데이트
	hc.updateOverallHealth()
	
	// 이슈 감지 및 알림
	hc.detectIssues()
}

func (hc *PoolHealthChecker) getActiveSessions() []string {
	// 실제 구현에서는 풀에서 활성 세션 목록을 가져와야 함
	// 여기서는 더미 구현
	return []string{}
}

func (hc *PoolHealthChecker) getSession(sessionID string) *PooledSession {
	// 실제 구현에서는 풀에서 세션을 가져와야 함
	// 여기서는 더미 구현
	return nil
}

func (hc *PoolHealthChecker) checkSessionPing(session *PooledSession) HealthCheckResult {
	startTime := time.Now()
	
	// 기본 ping 체크 (세션이 응답하는지)
	// 실제 구현에서는 세션에 ping 명령 전송
	
	// 시뮬레이션
	time.Sleep(10 * time.Millisecond)
	
	return HealthCheckResult{
		Timestamp:    startTime,
		Success:      true,
		ResponseTime: time.Since(startTime),
		CheckType:    CheckPing,
	}
}

func (hc *PoolHealthChecker) checkSessionProcess(session *PooledSession) HealthCheckResult {
	startTime := time.Now()
	
	// 프로세스 상태 체크
	// 실제 구현에서는 Claude CLI 프로세스가 살아있는지 확인
	
	return HealthCheckResult{
		Timestamp:    startTime,
		Success:      true,
		ResponseTime: time.Since(startTime),
		CheckType:    CheckProcess,
	}
}

func (hc *PoolHealthChecker) checkSessionMemory(session *PooledSession) HealthCheckResult {
	startTime := time.Now()
	
	// 메모리 사용량 체크
	// 실제 구현에서는 세션의 메모리 사용량 확인
	
	return HealthCheckResult{
		Timestamp:    startTime,
		Success:      true,
		ResponseTime: time.Since(startTime),
		CheckType:    CheckMemory,
	}
}

func (hc *PoolHealthChecker) checkSessionResponse(session *PooledSession) HealthCheckResult {
	startTime := time.Now()
	
	// 응답 시간 체크
	// 실제 구현에서는 간단한 명령을 보내고 응답 시간 측정
	
	return HealthCheckResult{
		Timestamp:    startTime,
		Success:      true,
		ResponseTime: time.Since(startTime),
		CheckType:    CheckResponse,
	}
}

func (hc *PoolHealthChecker) checkSessionLoad(session *PooledSession) HealthCheckResult {
	startTime := time.Now()
	
	// 부하 상태 체크
	// 실제 구현에서는 세션의 CPU 사용률 등 확인
	
	return HealthCheckResult{
		Timestamp:    startTime,
		Success:      true,
		ResponseTime: time.Since(startTime),
		CheckType:    CheckLoad,
	}
}

func (hc *PoolHealthChecker) updateSessionHealth(sessionID string, result *HealthCheckResult) {
	hc.healthMutex.Lock()
	defer hc.healthMutex.Unlock()
	
	health, exists := hc.sessionHealth[sessionID]
	if !exists {
		health = &SessionHealth{
			SessionID: sessionID,
			Status:    HealthUnknown,
			Checks:    make([]HealthCheckResult, 0, 10),
		}
		hc.sessionHealth[sessionID] = health
	}
	
	// 체크 결과 추가
	health.Checks = append(health.Checks, *result)
	
	// 최근 10개만 유지
	if len(health.Checks) > 10 {
		health.Checks = health.Checks[1:]
	}
	
	// 통계 업데이트
	health.TotalChecks++
	health.LastCheckTime = result.Timestamp
	
	if result.Success {
		health.SuccessfulChecks++
		health.ConsecutiveSuccess++
		health.ConsecutiveFailures = 0
	} else {
		health.FailedChecks++
		health.ConsecutiveFailures++
		health.ConsecutiveSuccess = 0
		health.LastError = result.ErrorMessage
	}
	
	// 평균 응답 시간 업데이트
	if len(health.Checks) > 0 {
		var totalTime time.Duration
		for _, check := range health.Checks {
			totalTime += check.ResponseTime
		}
		health.AverageResponseTime = totalTime / time.Duration(len(health.Checks))
	}
	
	// 헬스 상태 및 점수 계산
	health.Status = hc.calculatePoolHealthStatus(health)
	health.HealthScore = hc.calculateHealthScore(health)
}

func (hc *PoolHealthChecker) calculatePoolHealthStatus(health *SessionHealth) PoolHealthStatus {
	// 연속 실패 확인
	if health.ConsecutiveFailures >= hc.config.FailureThreshold {
		return HealthFailed
	}
	
	if health.ConsecutiveFailures > 0 {
		return HealthWarning
	}
	
	// 성공률 확인
	if health.TotalChecks > 0 {
		successRate := float64(health.SuccessfulChecks) / float64(health.TotalChecks)
		if successRate >= 0.95 {
			return HealthHealthy
		} else if successRate >= 0.8 {
			return HealthWarning
		} else {
			return HealthCritical
		}
	}
	
	return HealthUnknown
}

func (hc *PoolHealthChecker) calculateHealthScore(health *SessionHealth) float64 {
	if health.TotalChecks == 0 {
		return 0.5 // 미지수
	}
	
	// 기본 성공률 점수
	successRate := float64(health.SuccessfulChecks) / float64(health.TotalChecks)
	score := successRate
	
	// 연속 실패 페널티
	if health.ConsecutiveFailures > 0 {
		penalty := float64(health.ConsecutiveFailures) * 0.1
		score -= penalty
	}
	
	// 연속 성공 보너스
	if health.ConsecutiveSuccess > hc.config.SuccessThreshold {
		bonus := 0.1
		score += bonus
	}
	
	// 응답 시간 페널티
	if health.AverageResponseTime > hc.config.Timeout {
		score -= 0.2
	}
	
	// 0-1 범위로 제한
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}
	
	return score
}

func (hc *PoolHealthChecker) updateOverallHealth() {
	hc.healthMutex.RLock()
	defer hc.healthMutex.RUnlock()
	
	overall := &OverallHealth{
		LastUpdate: time.Now(),
		Issues:     make([]HealthIssue, 0),
	}
	
	// 세션별 헬스 상태 집계
	var totalScore float64
	for _, health := range hc.sessionHealth {
		overall.TotalSessions++
		totalScore += health.HealthScore
		
		switch health.Status {
		case HealthHealthy:
			overall.HealthySessions++
		case HealthWarning, HealthCritical, HealthFailed:
			overall.UnhealthySessions++
		}
	}
	
	// 전체 헬스 점수 계산
	if overall.TotalSessions > 0 {
		overall.HealthScore = totalScore / float64(overall.TotalSessions)
	}
	
	// 전체 상태 결정
	if overall.TotalSessions == 0 {
		overall.Status = HealthUnknown
	} else if overall.HealthySessions == overall.TotalSessions {
		overall.Status = HealthHealthy
	} else if overall.UnhealthySessions == overall.TotalSessions {
		overall.Status = HealthFailed
	} else if float64(overall.UnhealthySessions)/float64(overall.TotalSessions) > 0.5 {
		overall.Status = HealthCritical
	} else {
		overall.Status = HealthWarning
	}
	
	hc.overallHealth.Store(overall)
}

func (hc *PoolHealthChecker) detectIssues() {
	hc.healthMutex.RLock()
	defer hc.healthMutex.RUnlock()
	
	var issues []HealthIssue
	
	// 세션별 이슈 감지
	for sessionID, health := range hc.sessionHealth {
		// 높은 지연시간 감지
		if health.AverageResponseTime > 5*time.Second {
			issues = append(issues, HealthIssue{
				Type:        IssueHighLatency,
				Severity:    PoolSeverityMedium,
				Description: fmt.Sprintf("High average response time: %v", health.AverageResponseTime),
				SessionID:   sessionID,
				Timestamp:   time.Now(),
				Count:       1,
			})
		}
		
		// 높은 에러율 감지
		if health.TotalChecks > 0 {
			errorRate := float64(health.FailedChecks) / float64(health.TotalChecks)
			if errorRate > 0.1 { // 10% 이상
				issues = append(issues, HealthIssue{
					Type:        IssueHighErrorRate,
					Severity:    PoolSeverityHigh,
					Description: fmt.Sprintf("High error rate: %.2f%%", errorRate*100),
					SessionID:   sessionID,
					Timestamp:   time.Now(),
					Count:       int(health.FailedChecks),
				})
			}
		}
		
		// 연속 실패 감지
		if health.ConsecutiveFailures >= hc.config.FailureThreshold {
			issues = append(issues, HealthIssue{
				Type:        IssueProcessDead,
				Severity:    PoolSeverityCritical,
				Description: fmt.Sprintf("Session consecutive failures: %d", health.ConsecutiveFailures),
				SessionID:   sessionID,
				Timestamp:   time.Now(),
				Count:       health.ConsecutiveFailures,
			})
		}
	}
	
	// 이슈가 감지되면 알림
	for _, issue := range issues {
		hc.reportIssue(issue)
	}
}

func (hc *PoolHealthChecker) reportIssue(issue HealthIssue) {
	// 실제 구현에서는 알림 시스템으로 전송
	fmt.Printf("[HEALTH_ISSUE] %s: %s (Session: %s)\n", 
		hc.getIssueTypeString(issue.Type), 
		issue.Description, 
		issue.SessionID)
}

func (hc *PoolHealthChecker) getIssueTypeString(issueType IssueType) string {
	switch issueType {
	case IssueHighLatency:
		return "HIGH_LATENCY"
	case IssueMemoryLeak:
		return "MEMORY_LEAK"
	case IssueProcessDead:
		return "PROCESS_DEAD"
	case IssueHighErrorRate:
		return "HIGH_ERROR_RATE"
	case IssueResourceExhaustion:
		return "RESOURCE_EXHAUSTION"
	default:
		return "UNKNOWN"
	}
}

// RemoveSession은 세션 헬스 정보를 제거합니다
func (hc *PoolHealthChecker) RemoveSession(sessionID string) {
	hc.healthMutex.Lock()
	defer hc.healthMutex.Unlock()
	
	delete(hc.sessionHealth, sessionID)
}