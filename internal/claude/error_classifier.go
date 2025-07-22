package claude

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

// ErrorClassifier는 에러 분류 및 분석을 담당합니다
type ErrorClassifier interface {
	// 에러 분류 및 심각도 평가
	ClassifyError(err error) ErrorClass
	
	// 재시도 가능 여부 판단
	IsRetryable(err error) bool
	
	// 에러 우선순위 계산
	GetPriority(err error) ErrorPriority
	
	// 복구 전략 추천
	SuggestRecoveryStrategy(err error) RecoveryStrategy
	
	// 분류 규칙 추가
	AddClassificationRule(rule ClassificationRule) error
	
	// 학습된 패턴 추가
	LearnFromError(err error, actualClass ErrorClass) error
	
	// 통계 조회
	GetErrorStatistics() *ErrorStatistics
}

// ErrorClass는 에러 분류 정보입니다
type ErrorClass struct {
	Type        ErrorType     `json:"type"`
	Severity    ErrorSeverity `json:"severity"`
	Category    string        `json:"category"`
	Description string        `json:"description"`
	RetryAfter  time.Duration `json:"retry_after"`
	Confidence  float64       `json:"confidence"`
	Tags        []string      `json:"tags"`
	Context     map[string]interface{} `json:"context"`
}

// ErrorType은 에러 유형입니다
type ErrorType int

const (
	NetworkError ErrorType = iota
	ProcessError
	AuthError
	ResourceError
	TimeoutError
	ValidationError
	InternalError
	ConfigError
	DependencyError
	QuotaError
	UnknownError
)

// ErrorSeverity는 에러 심각도입니다
type ErrorSeverity int

const (
	SeverityLow ErrorSeverity = iota
	SeverityMedium
	SeverityHigh
	SeverityCritical
	SeverityFatal
)

// ErrorPriority는 에러 처리 우선순위입니다
type ErrorPriority int

const (
	PriorityLow ErrorPriority = iota
	PriorityMedium
	PriorityHigh
	PriorityUrgent
	PriorityCritical
)

// RecoveryStrategy는 복구 전략 인터페이스입니다
type RecoveryStrategy interface {
	CanRecover(ctx context.Context, err error) bool
	Execute(ctx context.Context, target RecoveryTarget) error
	GetEstimatedTime() time.Duration
	GetSuccessRate() float64
	GetName() string
}

// RecoveryTarget은 복구 대상입니다
type RecoveryTarget struct {
	Type       string                 `json:"type"`
	Identifier string                 `json:"identifier"`
	Context    map[string]interface{} `json:"context"`
}

// ClassificationRule은 분류 규칙입니다
type ClassificationRule struct {
	ID          string     `json:"id"`
	Pattern     string     `json:"pattern"`
	ErrorClass  ErrorClass `json:"error_class"`
	Weight      float64    `json:"weight"`
	IsRegex     bool       `json:"is_regex"`
	Enabled     bool       `json:"enabled"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ErrorStatistics는 에러 통계입니다
type ErrorStatistics struct {
	TotalErrors       int64                    `json:"total_errors"`
	ErrorsByType      map[ErrorType]int64      `json:"errors_by_type"`
	ErrorsBySeverity  map[ErrorSeverity]int64  `json:"errors_by_severity"`
	ErrorsByCategory  map[string]int64         `json:"errors_by_category"`
	ClassificationAccuracy float64            `json:"classification_accuracy"`
	TopErrors         []ErrorPattern           `json:"top_errors"`
	RecentErrors      []ClassifiedError        `json:"recent_errors"`
	StartTime         time.Time                `json:"start_time"`
	LastUpdated       time.Time                `json:"last_updated"`
}

// ErrorPattern은 에러 패턴입니다
type ErrorPattern struct {
	Pattern     string    `json:"pattern"`
	Count       int64     `json:"count"`
	LastSeen    time.Time `json:"last_seen"`
	ErrorClass  ErrorClass `json:"error_class"`
}

// ClassifiedError는 분류된 에러입니다
type ClassifiedError struct {
	Error       string     `json:"error"`
	ErrorClass  ErrorClass `json:"error_class"`
	Timestamp   time.Time  `json:"timestamp"`
	Context     map[string]interface{} `json:"context"`
}

// ErrorClassificationEngine은 에러 분류 엔진 구현체입니다
type ErrorClassificationEngine struct {
	// 분류 규칙들
	rules        []ClassificationRule
	rulesMutex   sync.RWMutex
	
	// 패턴 매칭
	patterns     map[string]ErrorClass
	patternsMutex sync.RWMutex
	
	// 학습된 규칙들
	learnedRules map[string]ErrorClass
	learnedMutex sync.RWMutex
	
	// 통계
	statistics   *ErrorStatistics
	statsMutex   sync.RWMutex
	
	// 캐시
	cache        map[string]ErrorClass
	cacheMutex   sync.RWMutex
	cacheSize    int
	
	// 복구 전략 매핑
	recoveryStrategies map[ErrorType][]RecoveryStrategy
	strategiesMutex    sync.RWMutex
}

// NewErrorClassificationEngine은 새로운 에러 분류 엔진을 생성합니다
func NewErrorClassificationEngine() *ErrorClassificationEngine {
	engine := &ErrorClassificationEngine{
		rules:        make([]ClassificationRule, 0),
		patterns:     make(map[string]ErrorClass),
		learnedRules: make(map[string]ErrorClass),
		cache:        make(map[string]ErrorClass),
		cacheSize:    1000,
		recoveryStrategies: make(map[ErrorType][]RecoveryStrategy),
		statistics: &ErrorStatistics{
			ErrorsByType:      make(map[ErrorType]int64),
			ErrorsBySeverity:  make(map[ErrorSeverity]int64),
			ErrorsByCategory:  make(map[string]int64),
			TopErrors:         make([]ErrorPattern, 0),
			RecentErrors:      make([]ClassifiedError, 0, 100),
			StartTime:         time.Now(),
		},
	}
	
	// 기본 분류 규칙 초기화
	engine.initializeDefaultRules()
	
	return engine
}

// ClassifyError는 에러를 분류합니다
func (e *ErrorClassificationEngine) ClassifyError(err error) ErrorClass {
	if err == nil {
		return ErrorClass{
			Type:        UnknownError,
			Severity:    SeverityLow,
			Category:    "none",
			Description: "No error",
			Confidence:  1.0,
		}
	}
	
	errorMsg := err.Error()
	
	// 캐시 확인
	if cached, found := e.getCachedClassification(errorMsg); found {
		return cached
	}
	
	// 분류 수행
	classification := e.performClassification(errorMsg)
	
	// 캐시에 저장
	e.cacheClassification(errorMsg, classification)
	
	// 통계 업데이트
	e.updateStatistics(classification)
	
	// 최근 에러에 추가
	e.addToRecentErrors(errorMsg, classification)
	
	return classification
}

// IsRetryable은 에러가 재시도 가능한지 판단합니다
func (e *ErrorClassificationEngine) IsRetryable(err error) bool {
	classification := e.ClassifyError(err)
	
	switch classification.Type {
	case NetworkError, TimeoutError, ResourceError, QuotaError:
		return true
	case ProcessError:
		// 프로세스 에러는 상황에 따라 재시도 가능
		return classification.Severity <= SeverityMedium
	case AuthError, ValidationError, ConfigError:
		return false
	case InternalError, DependencyError:
		// 심각도에 따라 결정
		return classification.Severity <= SeverityHigh
	default:
		return false
	}
}

// GetPriority는 에러의 처리 우선순위를 계산합니다
func (e *ErrorClassificationEngine) GetPriority(err error) ErrorPriority {
	classification := e.ClassifyError(err)
	
	// 심각도 기반 우선순위 계산
	switch classification.Severity {
	case SeverityFatal:
		return PriorityCritical
	case SeverityCritical:
		return PriorityUrgent
	case SeverityHigh:
		return PriorityHigh
	case SeverityMedium:
		return PriorityMedium
	default:
		return PriorityLow
	}
}

// SuggestRecoveryStrategy는 복구 전략을 추천합니다
func (e *ErrorClassificationEngine) SuggestRecoveryStrategy(err error) RecoveryStrategy {
	classification := e.ClassifyError(err)
	
	e.strategiesMutex.RLock()
	strategies := e.recoveryStrategies[classification.Type]
	e.strategiesMutex.RUnlock()
	
	if len(strategies) == 0 {
		return nil
	}
	
	// 가장 성공률이 높은 전략 선택
	var bestStrategy RecoveryStrategy
	var bestSuccessRate float64
	
	for _, strategy := range strategies {
		if strategy.CanRecover(context.Background(), err) {
			successRate := strategy.GetSuccessRate()
			if successRate > bestSuccessRate {
				bestSuccessRate = successRate
				bestStrategy = strategy
			}
		}
	}
	
	return bestStrategy
}

// AddClassificationRule은 분류 규칙을 추가합니다
func (e *ErrorClassificationEngine) AddClassificationRule(rule ClassificationRule) error {
	if rule.Pattern == "" {
		return fmt.Errorf("rule pattern cannot be empty")
	}
	
	// 정규식 유효성 검사
	if rule.IsRegex {
		if _, err := regexp.Compile(rule.Pattern); err != nil {
			return fmt.Errorf("invalid regex pattern: %w", err)
		}
	}
	
	rule.ID = fmt.Sprintf("rule_%d", time.Now().UnixNano())
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()
	
	e.rulesMutex.Lock()
	e.rules = append(e.rules, rule)
	e.rulesMutex.Unlock()
	
	return nil
}

// LearnFromError는 에러로부터 학습합니다
func (e *ErrorClassificationEngine) LearnFromError(err error, actualClass ErrorClass) error {
	if err == nil {
		return fmt.Errorf("error cannot be nil")
	}
	
	errorMsg := err.Error()
	
	e.learnedMutex.Lock()
	e.learnedRules[errorMsg] = actualClass
	e.learnedMutex.Unlock()
	
	// 캐시 무효화
	e.cacheMutex.Lock()
	delete(e.cache, errorMsg)
	e.cacheMutex.Unlock()
	
	return nil
}

// GetErrorStatistics는 에러 통계를 조회합니다
func (e *ErrorClassificationEngine) GetErrorStatistics() *ErrorStatistics {
	e.statsMutex.RLock()
	defer e.statsMutex.RUnlock()
	
	// 복사본 반환
	stats := &ErrorStatistics{
		TotalErrors:       e.statistics.TotalErrors,
		ErrorsByType:      make(map[ErrorType]int64),
		ErrorsBySeverity:  make(map[ErrorSeverity]int64),
		ErrorsByCategory:  make(map[string]int64),
		ClassificationAccuracy: e.statistics.ClassificationAccuracy,
		TopErrors:         make([]ErrorPattern, len(e.statistics.TopErrors)),
		RecentErrors:      make([]ClassifiedError, len(e.statistics.RecentErrors)),
		StartTime:         e.statistics.StartTime,
		LastUpdated:       e.statistics.LastUpdated,
	}
	
	// 맵 복사
	for k, v := range e.statistics.ErrorsByType {
		stats.ErrorsByType[k] = v
	}
	for k, v := range e.statistics.ErrorsBySeverity {
		stats.ErrorsBySeverity[k] = v
	}
	for k, v := range e.statistics.ErrorsByCategory {
		stats.ErrorsByCategory[k] = v
	}
	
	// 슬라이스 복사
	copy(stats.TopErrors, e.statistics.TopErrors)
	copy(stats.RecentErrors, e.statistics.RecentErrors)
	
	return stats
}

// 내부 메서드들

func (e *ErrorClassificationEngine) initializeDefaultRules() {
	defaultRules := []ClassificationRule{
		{
			Pattern: "connection.*refused",
			ErrorClass: ErrorClass{
				Type:        NetworkError,
				Severity:    SeverityMedium,
				Category:    "network",
				Description: "Connection refused error",
				RetryAfter:  2 * time.Second,
			},
			Weight:  1.0,
			IsRegex: true,
			Enabled: true,
		},
		{
			Pattern: "timeout",
			ErrorClass: ErrorClass{
				Type:        TimeoutError,
				Severity:    SeverityMedium,
				Category:    "timeout",
				Description: "Operation timeout",
				RetryAfter:  5 * time.Second,
			},
			Weight:  0.9,
			IsRegex: false,
			Enabled: true,
		},
		{
			Pattern: "process.*exit.*code",
			ErrorClass: ErrorClass{
				Type:        ProcessError,
				Severity:    SeverityHigh,
				Category:    "process",
				Description: "Process exit with error code",
				RetryAfter:  10 * time.Second,
			},
			Weight:  1.0,
			IsRegex: true,
			Enabled: true,
		},
		{
			Pattern: "unauthorized|authentication.*failed",
			ErrorClass: ErrorClass{
				Type:        AuthError,
				Severity:    SeverityHigh,
				Category:    "auth",
				Description: "Authentication failed",
				RetryAfter:  0, // 재시도 불가
			},
			Weight:  1.0,
			IsRegex: true,
			Enabled: true,
		},
		{
			Pattern: "out of memory|memory allocation failed",
			ErrorClass: ErrorClass{
				Type:        ResourceError,
				Severity:    SeverityCritical,
				Category:    "resource",
				Description: "Memory allocation failed",
				RetryAfter:  30 * time.Second,
			},
			Weight:  1.0,
			IsRegex: true,
			Enabled: true,
		},
		{
			Pattern: "quota.*exceeded|rate.*limit",
			ErrorClass: ErrorClass{
				Type:        QuotaError,
				Severity:    SeverityMedium,
				Category:    "quota",
				Description: "Quota or rate limit exceeded",
				RetryAfter:  60 * time.Second,
			},
			Weight:  1.0,
			IsRegex: true,
			Enabled: true,
		},
	}
	
	for _, rule := range defaultRules {
		e.AddClassificationRule(rule)
	}
}

func (e *ErrorClassificationEngine) performClassification(errorMsg string) ErrorClass {
	// 1. 학습된 규칙에서 정확한 매치 확인
	e.learnedMutex.RLock()
	if learned, found := e.learnedRules[errorMsg]; found {
		e.learnedMutex.RUnlock()
		learned.Confidence = 1.0
		return learned
	}
	e.learnedMutex.RUnlock()
	
	// 2. 분류 규칙 적용
	e.rulesMutex.RLock()
	var bestMatch ErrorClass
	var bestScore float64
	
	for _, rule := range e.rules {
		if !rule.Enabled {
			continue
		}
		
		var match bool
		if rule.IsRegex {
			if regex, err := regexp.Compile(rule.Pattern); err == nil {
				match = regex.MatchString(errorMsg)
			}
		} else {
			match = strings.Contains(strings.ToLower(errorMsg), strings.ToLower(rule.Pattern))
		}
		
		if match && rule.Weight > bestScore {
			bestScore = rule.Weight
			bestMatch = rule.ErrorClass
			bestMatch.Confidence = rule.Weight
		}
	}
	e.rulesMutex.RUnlock()
	
	// 3. 매치된 규칙이 있으면 반환
	if bestScore > 0 {
		return bestMatch
	}
	
	// 4. 기본 분류 (휴리스틱 기반)
	return e.performHeuristicClassification(errorMsg)
}

func (e *ErrorClassificationEngine) performHeuristicClassification(errorMsg string) ErrorClass {
	lowerMsg := strings.ToLower(errorMsg)
	
	// 네트워크 관련
	if strings.Contains(lowerMsg, "network") || strings.Contains(lowerMsg, "connection") ||
		strings.Contains(lowerMsg, "socket") || strings.Contains(lowerMsg, "tcp") ||
		strings.Contains(lowerMsg, "dns") || strings.Contains(lowerMsg, "host") {
		return ErrorClass{
			Type:        NetworkError,
			Severity:    SeverityMedium,
			Category:    "network",
			Description: "Network-related error detected",
			RetryAfter:  3 * time.Second,
			Confidence:  0.7,
		}
	}
	
	// 타임아웃 관련
	if strings.Contains(lowerMsg, "timeout") || strings.Contains(lowerMsg, "deadline") ||
		strings.Contains(lowerMsg, "expired") {
		return ErrorClass{
			Type:        TimeoutError,
			Severity:    SeverityMedium,
			Category:    "timeout",
			Description: "Timeout error detected",
			RetryAfter:  5 * time.Second,
			Confidence:  0.8,
		}
	}
	
	// 리소스 관련
	if strings.Contains(lowerMsg, "memory") || strings.Contains(lowerMsg, "resource") ||
		strings.Contains(lowerMsg, "limit") || strings.Contains(lowerMsg, "capacity") {
		return ErrorClass{
			Type:        ResourceError,
			Severity:    SeverityHigh,
			Category:    "resource",
			Description: "Resource-related error detected",
			RetryAfter:  15 * time.Second,
			Confidence:  0.6,
		}
	}
	
	// 프로세스 관련
	if strings.Contains(lowerMsg, "process") || strings.Contains(lowerMsg, "exit") ||
		strings.Contains(lowerMsg, "signal") || strings.Contains(lowerMsg, "killed") {
		return ErrorClass{
			Type:        ProcessError,
			Severity:    SeverityHigh,
			Category:    "process",
			Description: "Process-related error detected",
			RetryAfter:  10 * time.Second,
			Confidence:  0.7,
		}
	}
	
	// 인증 관련
	if strings.Contains(lowerMsg, "auth") || strings.Contains(lowerMsg, "unauthorized") ||
		strings.Contains(lowerMsg, "forbidden") || strings.Contains(lowerMsg, "permission") {
		return ErrorClass{
			Type:        AuthError,
			Severity:    SeverityHigh,
			Category:    "auth",
			Description: "Authentication/authorization error detected",
			RetryAfter:  0,
			Confidence:  0.8,
		}
	}
	
	// 기본값 (알 수 없는 에러)
	return ErrorClass{
		Type:        UnknownError,
		Severity:    SeverityMedium,
		Category:    "unknown",
		Description: "Unknown error type",
		RetryAfter:  5 * time.Second,
		Confidence:  0.3,
	}
}

func (e *ErrorClassificationEngine) getCachedClassification(errorMsg string) (ErrorClass, bool) {
	e.cacheMutex.RLock()
	defer e.cacheMutex.RUnlock()
	
	classification, found := e.cache[errorMsg]
	return classification, found
}

func (e *ErrorClassificationEngine) cacheClassification(errorMsg string, classification ErrorClass) {
	e.cacheMutex.Lock()
	defer e.cacheMutex.Unlock()
	
	// 캐시 크기 제한
	if len(e.cache) >= e.cacheSize {
		// LRU 방식으로 오래된 항목 제거 (간단한 구현)
		for k := range e.cache {
			delete(e.cache, k)
			break
		}
	}
	
	e.cache[errorMsg] = classification
}

func (e *ErrorClassificationEngine) updateStatistics(classification ErrorClass) {
	e.statsMutex.Lock()
	defer e.statsMutex.Unlock()
	
	e.statistics.TotalErrors++
	e.statistics.ErrorsByType[classification.Type]++
	e.statistics.ErrorsBySeverity[classification.Severity]++
	e.statistics.ErrorsByCategory[classification.Category]++
	e.statistics.LastUpdated = time.Now()
}

func (e *ErrorClassificationEngine) addToRecentErrors(errorMsg string, classification ErrorClass) {
	e.statsMutex.Lock()
	defer e.statsMutex.Unlock()
	
	recentError := ClassifiedError{
		Error:      errorMsg,
		ErrorClass: classification,
		Timestamp:  time.Now(),
		Context:    make(map[string]interface{}),
	}
	
	// 최근 에러 목록에 추가 (최대 100개)
	e.statistics.RecentErrors = append(e.statistics.RecentErrors, recentError)
	if len(e.statistics.RecentErrors) > 100 {
		e.statistics.RecentErrors = e.statistics.RecentErrors[1:]
	}
}