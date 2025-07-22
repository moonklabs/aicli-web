package claude

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// ErrorRecovery 에러 복구 인터페이스
type ErrorRecovery interface {
	HandleError(err error) RecoveryAction
	ShouldRestart(err error) bool
	Restart(ctx context.Context) error
	GetRecoveryStats() *RecoveryStats
	SetRecoveryPolicy(policy *RecoveryPolicy)
	Start(ctx context.Context) error
	Stop() error
	IsEnabled() bool
}

// RecoveryAction 복구 액션 타입
type RecoveryAction int

const (
	// ActionIgnore 에러를 무시하고 계속 진행
	ActionIgnore RecoveryAction = iota
	// ActionRetry 재시도
	ActionRetry
	// ActionRestart 프로세스 재시작
	ActionRestart
	// ActionFail 실패 처리
	ActionFail
	// ActionCircuitBreak 회로 차단기 활성화
	ActionCircuitBreak
)

// String RecoveryAction을 문자열로 변환
func (a RecoveryAction) String() string {
	switch a {
	case ActionIgnore:
		return "ignore"
	case ActionRetry:
		return "retry"
	case ActionRestart:
		return "restart"
	case ActionFail:
		return "fail"
	case ActionCircuitBreak:
		return "circuit_break"
	default:
		return "unknown"
	}
}

// RecoveryErrorType 에러 타입 분류 (복구 전략용)
type RecoveryErrorType int

const (
	// RecoveryErrorTypeUnknown 알 수 없는 에러
	RecoveryErrorTypeUnknown RecoveryErrorType = iota
	// RecoveryErrorTypeTransient 일시적 오류 (네트워크, 타임아웃)
	RecoveryErrorTypeTransient
	// RecoveryErrorTypePermanent 영구적 오류 (설정, 권한)
	RecoveryErrorTypePermanent
	// RecoveryErrorTypeProcess 프로세스 관련 오류
	RecoveryErrorTypeProcess
	// RecoveryErrorTypeResource 리소스 부족
	RecoveryErrorTypeResource
	// RecoveryErrorTypeAPI API 오류
	RecoveryErrorTypeAPI
)

// String RecoveryErrorType을 문자열로 변환
func (e RecoveryErrorType) String() string {
	switch e {
	case RecoveryErrorTypeTransient:
		return "transient"
	case RecoveryErrorTypePermanent:
		return "permanent"
	case RecoveryErrorTypeProcess:
		return "process"
	case RecoveryErrorTypeResource:
		return "resource"
	case RecoveryErrorTypeAPI:
		return "api"
	default:
		return "unknown"
	}
}

// RecoveryPolicy 복구 정책 설정
type RecoveryPolicy struct {
	// MaxRestarts 최대 재시작 횟수
	MaxRestarts int `yaml:"max_restarts" json:"max_restarts"`
	// RestartInterval 재시작 간격
	RestartInterval time.Duration `yaml:"restart_interval" json:"restart_interval"`
	// BackoffMultiplier 백오프 배수
	BackoffMultiplier float64 `yaml:"backoff_multiplier" json:"backoff_multiplier"`
	// MaxBackoff 최대 백오프 시간
	MaxBackoff time.Duration `yaml:"max_backoff" json:"max_backoff"`
	// CircuitBreakerConfig 회로 차단기 설정
	CircuitBreakerConfig *CircuitBreakerConfig `yaml:"circuit_breaker" json:"circuit_breaker"`
	// Enabled 복구 시스템 활성화 여부
	Enabled bool `yaml:"enabled" json:"enabled"`
}

// RecoveryStats 복구 통계 정보
type RecoveryStats struct {
	// TotalErrors 총 에러 수
	TotalErrors int64 `json:"total_errors"`
	// RestartCount 재시작 횟수
	RestartCount int64 `json:"restart_count"`
	// LastError 마지막 에러
	LastError error `json:"last_error"`
	// LastRestart 마지막 재시작 시간
	LastRestart time.Time `json:"last_restart"`
	// SuccessfulRuns 성공적인 실행 횟수
	SuccessfulRuns int64 `json:"successful_runs"`
	// AverageUptime 평균 가동 시간
	AverageUptime time.Duration `json:"average_uptime"`
	// ErrorsByType 에러 타입별 통계
	ErrorsByType map[RecoveryErrorType]int64 `json:"errors_by_type"`
	// ActionsByType 액션별 통계
	ActionsByType map[RecoveryAction]int64 `json:"actions_by_type"`
}

// DefaultRecoveryPolicy 기본 복구 정책을 반환합니다
func DefaultRecoveryPolicy() *RecoveryPolicy {
	return &RecoveryPolicy{
		MaxRestarts:       5,
		RestartInterval:   30 * time.Second,
		BackoffMultiplier: 2.0,
		MaxBackoff:        5 * time.Minute,
		CircuitBreakerConfig: &CircuitBreakerConfig{
			FailureThreshold: 5,
			RecoveryTimeout:  1 * time.Minute,
			SuccessThreshold: 3,
		},
		Enabled: true,
	}
}

// NewRecoveryStats 새로운 복구 통계를 생성합니다
func NewRecoveryStats() *RecoveryStats {
	return &RecoveryStats{
		ErrorsByType:  make(map[RecoveryErrorType]int64),
		ActionsByType: make(map[RecoveryAction]int64),
	}
}

// IncrementError 에러 수를 증가시킵니다
func (rs *RecoveryStats) IncrementError(errorType RecoveryErrorType) {
	rs.TotalErrors++
	rs.ErrorsByType[errorType]++
}

// IncrementAction 액션 수를 증가시킵니다
func (rs *RecoveryStats) IncrementAction(action RecoveryAction) {
	rs.ActionsByType[action]++
}

// IncrementRestart 재시작 수를 증가시킵니다
func (rs *RecoveryStats) IncrementRestart() {
	rs.RestartCount++
	rs.LastRestart = time.Now()
}

// IncrementSuccessfulRun 성공적인 실행 수를 증가시킵니다
func (rs *RecoveryStats) IncrementSuccessfulRun() {
	rs.SuccessfulRuns++
}

// RecoveryRecoveryClassificationRule 에러 분류 규칙 (복구용)
type RecoveryClassificationRule struct {
	// ErrorPattern 에러 패턴
	ErrorPattern string `yaml:"error_pattern" json:"error_pattern"`
	// Action 수행할 액션
	Action RecoveryAction `yaml:"action" json:"action"`
	// Retryable 재시도 가능 여부
	Retryable bool `yaml:"retryable" json:"retryable"`
	// RecoveryBackoffType 백오프 타입
	RecoveryBackoffType RecoveryBackoffType `yaml:"backoff_type" json:"backoff_type"`
}

// RecoveryBackoffType 백오프 타입 (복구용)
type RecoveryBackoffType int

const (
	// BackoffFixed 고정 백오프
	BackoffFixed RecoveryBackoffType = iota
	// BackoffExponential 지수 백오프
	BackoffExponential
	// BackoffLinear 선형 백오프
	BackoffLinear
)

// String RecoveryBackoffType을 문자열로 변환
func (b RecoveryBackoffType) String() string {
	switch b {
	case BackoffFixed:
		return "fixed"
	case BackoffExponential:
		return "exponential"
	case BackoffLinear:
		return "linear"
	default:
		return "unknown"
	}
}

// RecoveryRecoveryErrorClassifier 에러 분류기 (복구용)
type RecoveryRecoveryErrorClassifier struct {
	rules map[RecoveryErrorType][]RecoveryClassificationRule
	mutex sync.RWMutex
}

// NewRecoveryErrorClassifier 새로운 에러 분류기를 생성합니다
func NewRecoveryErrorClassifier() *RecoveryRecoveryErrorClassifier {
	classifier := &RecoveryErrorClassifier{
		rules: make(map[RecoveryErrorType][]RecoveryClassificationRule),
	}
	
	// 기본 분류 규칙 추가
	classifier.addDefaultRules()
	
	return classifier
}

// addDefaultRules 기본 분류 규칙을 추가합니다
func (ec *RecoveryRecoveryErrorClassifier) addDefaultRules() {
	// 일시적 에러 규칙
	ec.rules[RecoveryErrorTypeTransient] = []RecoveryClassificationRule{
		{
			ErrorPattern: "connection refused",
			Action:       ActionRetry,
			Retryable:    true,
			RecoveryBackoffType:  BackoffExponential,
		},
		{
			ErrorPattern: "timeout",
			Action:       ActionRetry,
			Retryable:    true,
			RecoveryBackoffType:  BackoffExponential,
		},
		{
			ErrorPattern: "temporary failure",
			Action:       ActionRetry,
			Retryable:    true,
			RecoveryBackoffType:  BackoffLinear,
		},
	}

	// 영구적 에러 규칙
	ec.rules[RecoveryErrorTypePermanent] = []RecoveryClassificationRule{
		{
			ErrorPattern: "permission denied",
			Action:       ActionFail,
			Retryable:    false,
			RecoveryBackoffType:  BackoffFixed,
		},
		{
			ErrorPattern: "invalid api key",
			Action:       ActionFail,
			Retryable:    false,
			RecoveryBackoffType:  BackoffFixed,
		},
		{
			ErrorPattern: "authentication failed",
			Action:       ActionFail,
			Retryable:    false,
			RecoveryBackoffType:  BackoffFixed,
		},
	}

	// 프로세스 에러 규칙
	ec.rules[RecoveryErrorTypeProcess] = []RecoveryClassificationRule{
		{
			ErrorPattern: "process exited",
			Action:       ActionRestart,
			Retryable:    true,
			RecoveryBackoffType:  BackoffExponential,
		},
		{
			ErrorPattern: "signal",
			Action:       ActionRestart,
			Retryable:    true,
			RecoveryBackoffType:  BackoffExponential,
		},
		{
			ErrorPattern: "unexpected exit",
			Action:       ActionRestart,
			Retryable:    true,
			RecoveryBackoffType:  BackoffExponential,
		},
	}

	// 리소스 에러 규칙
	ec.rules[RecoveryErrorTypeResource] = []RecoveryClassificationRule{
		{
			ErrorPattern: "out of memory",
			Action:       ActionCircuitBreak,
			Retryable:    false,
			RecoveryBackoffType:  BackoffFixed,
		},
		{
			ErrorPattern: "resource limit",
			Action:       ActionCircuitBreak,
			Retryable:    false,
			RecoveryBackoffType:  BackoffFixed,
		},
		{
			ErrorPattern: "disk full",
			Action:       ActionFail,
			Retryable:    false,
			RecoveryBackoffType:  BackoffFixed,
		},
	}

	// API 에러 규칙
	ec.rules[RecoveryErrorTypeAPI] = []RecoveryClassificationRule{
		{
			ErrorPattern: "rate limit",
			Action:       ActionRetry,
			Retryable:    true,
			RecoveryBackoffType:  BackoffExponential,
		},
		{
			ErrorPattern: "service unavailable",
			Action:       ActionRetry,
			Retryable:    true,
			RecoveryBackoffType:  BackoffExponential,
		},
		{
			ErrorPattern: "bad request",
			Action:       ActionFail,
			Retryable:    false,
			RecoveryBackoffType:  BackoffFixed,
		},
	}
}

// ClassifyError 에러를 분류하고 적절한 액션을 반환합니다
func (ec *RecoveryRecoveryErrorClassifier) ClassifyError(err error) (RecoveryErrorType, RecoveryAction) {
	if err == nil {
		return RecoveryErrorTypeUnknown, ActionIgnore
	}

	ec.mutex.RLock()
	defer ec.mutex.RUnlock()
	
	errStr := strings.ToLower(err.Error())
	
	// 각 에러 타입의 규칙을 확인
	for errorType, rules := range ec.rules {
		for _, rule := range rules {
			if strings.Contains(errStr, strings.ToLower(rule.ErrorPattern)) {
				return errorType, rule.Action
			}
		}
	}
	
	// 기본 처리
	return RecoveryErrorTypeUnknown, ActionIgnore
}

// AddRule 새로운 분류 규칙을 추가합니다
func (ec *RecoveryRecoveryErrorClassifier) AddRule(errorType RecoveryErrorType, rule RecoveryClassificationRule) {
	ec.mutex.Lock()
	defer ec.mutex.Unlock()
	
	if ec.rules[errorType] == nil {
		ec.rules[errorType] = []RecoveryClassificationRule{}
	}
	
	ec.rules[errorType] = append(ec.rules[errorType], rule)
}

// GetRules 특정 에러 타입의 규칙들을 반환합니다
func (ec *RecoveryRecoveryErrorClassifier) GetRules(errorType RecoveryErrorType) []RecoveryClassificationRule {
	ec.mutex.RLock()
	defer ec.mutex.RUnlock()
	
	return ec.rules[errorType]
}