package claude

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"
)

// SimpleErrorClassifier는 ErrorClassifier의 간단한 구현체입니다
type SimpleErrorClassifier struct {
	rules  []ClassificationRule
	stats  *ErrorStatistics
	mutex  sync.RWMutex
}

// NewSimpleErrorClassifier는 새로운 SimpleErrorClassifier를 생성합니다
func NewSimpleErrorClassifier() *SimpleErrorClassifier {
	return &SimpleErrorClassifier{
		rules: make([]ClassificationRule, 0),
		stats: NewErrorStatistics(),
	}
}

// ClassifyError는 에러를 분류합니다
func (s *SimpleErrorClassifier) ClassifyError(err error) ErrorClass {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if err == nil {
		return ErrorClass{
			Type:     ErrorTypeNone,
			Severity: SeverityLow,
		}
	}

	// 규칙에 따라 분류
	for _, rule := range s.rules {
		if rule.ErrorClass.Type != ErrorTypeNone && strings.Contains(err.Error(), rule.Pattern) {
			return rule.ErrorClass
		}
	}

	// 기본 분류
	return ErrorClass{
		Type:        UnknownError,
		Severity:    SeverityMedium,
		RetryAfter:  5 * time.Second,
	}
}

// IsRetryable은 에러가 재시도 가능한지 판단합니다
func (s *SimpleErrorClassifier) IsRetryable(err error) bool {
	class := s.ClassifyError(err)
	return class.Type == NetworkError || class.Type == TimeoutError || class.Type == ResourceError
}

// GetPriority는 에러의 우선순위를 반환합니다
func (s *SimpleErrorClassifier) GetPriority(err error) ErrorPriority {
	class := s.ClassifyError(err)
	
	switch class.Severity {
	case SeverityCritical:
		return PriorityHigh
	case SeverityHigh:
		return PriorityHigh
	case SeverityMedium:
		return PriorityMedium
	default:
		return PriorityLow
	}
}

// SuggestRecoveryStrategy는 복구 전략을 제안합니다
func (s *SimpleErrorClassifier) SuggestRecoveryStrategy(err error) RecoveryStrategy {
	class := s.ClassifyError(err)
	isRetryable := s.IsRetryable(err)
	
	if !isRetryable {
		return &SimpleRecoveryStrategy{
			StrategyType: "fail_fast",
		}
	}

	switch class.Type {
	case NetworkError:
		return &SimpleRecoveryStrategy{
			StrategyType: "exponential_backoff",
			MaxRetries:   5,
		}
	case TimeoutError:
		return &SimpleRecoveryStrategy{
			StrategyType: "exponential_backoff",
			MaxRetries:   3,
		}
	case ResourceError:
		return &SimpleRecoveryStrategy{
			StrategyType: "linear_backoff",
			MaxRetries:   2,
		}
	default:
		return &SimpleRecoveryStrategy{
			StrategyType: "exponential_backoff",
			MaxRetries:   3,
		}
	}
}

// AddClassificationRule은 분류 규칙을 추가합니다
func (s *SimpleErrorClassifier) AddClassificationRule(rule ClassificationRule) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if rule.Name == "" {
		return errors.New("rule name cannot be empty")
	}

	s.rules = append(s.rules, rule)
	return nil
}

// LearnFromError는 에러로부터 학습합니다 (현재는 단순 구현)
func (s *SimpleErrorClassifier) LearnFromError(err error, actualClass ErrorClass) error {
	// TODO: 실제 머신러닝 구현
	s.stats.RecordError(err, actualClass)
	return nil
}

// GetErrorStatistics는 에러 통계를 반환합니다
func (s *SimpleErrorClassifier) GetErrorStatistics() *ErrorStatistics {
	return s.stats
}

// SimpleRecoveryStrategy는 RecoveryStrategy의 간단한 구현체입니다
type SimpleRecoveryStrategy struct {
	StrategyType string
	MaxRetries   int
}

// CanRecover는 복구 가능 여부를 확인합니다
func (s *SimpleRecoveryStrategy) CanRecover(ctx context.Context, err error) bool {
	return s.StrategyType != "fail_fast"
}

// Execute는 복구를 실행합니다
func (s *SimpleRecoveryStrategy) Execute(ctx context.Context, target RecoveryTarget) error {
	// 간단한 구현 - 실제로는 복구 로직이 들어가야 함
	return nil
}

// GetEstimatedTime는 예상 소요 시간을 반환합니다
func (s *SimpleRecoveryStrategy) GetEstimatedTime() time.Duration {
	if s.StrategyType == "exponential_backoff" {
		return time.Duration(s.MaxRetries) * time.Second * 10
	}
	return time.Duration(s.MaxRetries) * time.Second * 5
}

// GetSuccessRate는 성공률을 반환합니다
func (s *SimpleRecoveryStrategy) GetSuccessRate() float64 {
	switch s.StrategyType {
	case "exponential_backoff":
		return 0.8
	case "linear_backoff":
		return 0.7
	default:
		return 0.5
	}
}

// GetName는 전략명을 반환합니다
func (s *SimpleRecoveryStrategy) GetName() string {
	return s.StrategyType
}