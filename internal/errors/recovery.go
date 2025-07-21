package errors

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// RetryPolicy는 재시도 정책을 정의합니다.
type RetryPolicy struct {
	MaxAttempts   int           // 최대 시도 횟수
	BaseDelay     time.Duration // 기본 지연 시간
	MaxDelay      time.Duration // 최대 지연 시간
	Multiplier    float64       // 지연 시간 증가 배수
	Jitter        bool          // 지터(랜덤 변동) 적용 여부
	RetryableFunc RetryableFunc // 재시도 가능 여부 판단 함수
}

// RetryableFunc는 에러가 재시도 가능한지 판단하는 함수 타입입니다.
type RetryableFunc func(error) bool

// DefaultRetryPolicy는 기본 재시도 정책을 반환합니다.
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxAttempts:   3,
		BaseDelay:     time.Second,
		MaxDelay:      30 * time.Second,
		Multiplier:    2.0,
		Jitter:        true,
		RetryableFunc: DefaultRetryableFunc,
	}
}

// DefaultRetryableFunc는 기본 재시도 가능 여부 판단 함수입니다.
func DefaultRetryableFunc(err error) bool {
	if cliErr, ok := err.(*CLIError); ok {
		switch cliErr.Type {
		case ErrorTypeNetwork:
			return true // 네트워크 오류는 재시도 가능
		case ErrorTypeProcess:
			return true // 프로세스 오류는 재시도 가능
		case ErrorTypeInternal:
			return true // 내부 오류는 재시도 가능
		case ErrorTypeConfig:
			return false // 설정 오류는 재시도 불가
		case ErrorTypeValidation:
			return false // 검증 오류는 재시도 불가
		case ErrorTypeAuthentication:
			return false // 인증 오류는 재시도 불가
		case ErrorTypePermission:
			return false // 권한 오류는 재시도 불가
		case ErrorTypeFileSystem:
			return false // 파일 시스템 오류는 재시도 불가
		case ErrorTypeNotFound:
			return false // 미발견 오류는 재시도 불가
		case ErrorTypeConflict:
			return false // 충돌 오류는 재시도 불가
		default:
			return false
		}
	}
	return false
}

// RetryableOperation은 재시도 가능한 작업을 나타내는 함수 타입입니다.
type RetryableOperation func(ctx context.Context, attempt int) error

// RetryWithPolicy는 지정된 정책에 따라 작업을 재시도합니다.
func RetryWithPolicy(ctx context.Context, policy *RetryPolicy, operation RetryableOperation) error {
	if policy == nil {
		policy = DefaultRetryPolicy()
	}
	
	var lastError error
	
	for attempt := 1; attempt <= policy.MaxAttempts; attempt++ {
		// 컨텍스트 취소 확인
		select {
		case <-ctx.Done():
			return NewInternalError("retry", ctx.Err())
		default:
		}
		
		// 작업 실행
		err := operation(ctx, attempt)
		if err == nil {
			return nil // 성공
		}
		
		lastError = err
		
		// 마지막 시도인 경우 바로 반환
		if attempt >= policy.MaxAttempts {
			break
		}
		
		// 재시도 가능 여부 확인
		if policy.RetryableFunc != nil && !policy.RetryableFunc(err) {
			// CLIError에 재시도 불가 정보 추가
			if cliErr, ok := err.(*CLIError); ok {
				cliErr.AddContext("retryable", false)
				cliErr.AddContext("attempt", attempt)
				cliErr.AddSuggestion("이 오류는 재시도해도 해결되지 않습니다")
			}
			return err
		}
		
		// 지연 시간 계산
		delay := policy.calculateDelay(attempt)
		
		// 로그 기록
		LogErrorWithLevel(LogLevelWarn, NewInternalError("retry", 
			fmt.Errorf("작업 실패 (시도 %d/%d), %v 후 재시도: %v", 
				attempt, policy.MaxAttempts, delay, err)))
		
		// 지연
		select {
		case <-ctx.Done():
			return NewInternalError("retry", ctx.Err())
		case <-time.After(delay):
		}
	}
	
	// 모든 시도 실패
	if cliErr, ok := lastError.(*CLIError); ok {
		cliErr.AddContext("max_attempts_reached", true)
		cliErr.AddContext("total_attempts", policy.MaxAttempts)
		cliErr.AddSuggestion(fmt.Sprintf("%d번 재시도했지만 실패했습니다", policy.MaxAttempts))
		return cliErr
	}
	
	return NewInternalError("retry", 
		fmt.Errorf("%d번 재시도 후에도 실패: %w", policy.MaxAttempts, lastError))
}

// calculateDelay는 지연 시간을 계산합니다.
func (p *RetryPolicy) calculateDelay(attempt int) time.Duration {
	// 지수 백오프 계산
	delay := time.Duration(float64(p.BaseDelay) * math.Pow(p.Multiplier, float64(attempt-1)))
	
	// 최대 지연 시간 제한
	if delay > p.MaxDelay {
		delay = p.MaxDelay
	}
	
	// 지터 적용
	if p.Jitter {
		jitter := time.Duration(rand.Float64() * float64(delay) * 0.1) // 10% 지터
		if rand.Intn(2) == 0 {
			delay += jitter
		} else {
			delay -= jitter
		}
	}
	
	// 최소값 보장
	if delay < 0 {
		delay = p.BaseDelay
	}
	
	return delay
}

// RecoveryStrategy는 복구 전략을 정의합니다.
type RecoveryStrategy interface {
	// CanRecover는 에러가 복구 가능한지 확인합니다.
	CanRecover(err error) bool
	
	// Recover는 에러를 복구합니다.
	Recover(ctx context.Context, err error) error
	
	// Name은 복구 전략의 이름을 반환합니다.
	Name() string
}

// ConfigRecoveryStrategy는 설정 관련 에러 복구 전략입니다.
type ConfigRecoveryStrategy struct{}

// CanRecover는 설정 에러가 복구 가능한지 확인합니다.
func (s *ConfigRecoveryStrategy) CanRecover(err error) bool {
	return IsType(err, ErrorTypeConfig)
}

// Recover는 설정 에러를 복구합니다.
func (s *ConfigRecoveryStrategy) Recover(ctx context.Context, err error) error {
	LogErrorWithLevel(LogLevelInfo, NewInternalError("recovery", 
		fmt.Errorf("설정 에러 복구 시도: %v", err)))
	
	// 설정 파일 백업 및 기본값으로 재설정
	// 실제 구현은 config 패키지와 연동 필요
	
	return NewConfigError(fmt.Errorf("설정 복구 시도됨"), "config")
}

// Name은 전략 이름을 반환합니다.
func (s *ConfigRecoveryStrategy) Name() string {
	return "ConfigRecovery"
}

// NetworkRecoveryStrategy는 네트워크 관련 에러 복구 전략입니다.
type NetworkRecoveryStrategy struct{}

// CanRecover는 네트워크 에러가 복구 가능한지 확인합니다.
func (s *NetworkRecoveryStrategy) CanRecover(err error) bool {
	return IsType(err, ErrorTypeNetwork)
}

// Recover는 네트워크 에러를 복구합니다.
func (s *NetworkRecoveryStrategy) Recover(ctx context.Context, err error) error {
	LogErrorWithLevel(LogLevelInfo, NewInternalError("recovery", 
		fmt.Errorf("네트워크 에러 복구 시도: %v", err)))
	
	// 네트워크 연결 상태 확인 및 재연결 시도
	// 실제 구현은 네트워크 모듈과 연동 필요
	
	return nil // 복구 성공으로 가정
}

// Name은 전략 이름을 반환합니다.
func (s *NetworkRecoveryStrategy) Name() string {
	return "NetworkRecovery"
}

// ProcessRecoveryStrategy는 프로세스 관련 에러 복구 전략입니다.
type ProcessRecoveryStrategy struct{}

// CanRecover는 프로세스 에러가 복구 가능한지 확인합니다.
func (s *ProcessRecoveryStrategy) CanRecover(err error) bool {
	return IsType(err, ErrorTypeProcess)
}

// Recover는 프로세스 에러를 복구합니다.
func (s *ProcessRecoveryStrategy) Recover(ctx context.Context, err error) error {
	LogErrorWithLevel(LogLevelInfo, NewInternalError("recovery", 
		fmt.Errorf("프로세스 에러 복구 시도: %v", err)))
	
	// 프로세스 재시작 시도
	// 실제 구현은 프로세스 관리자와 연동 필요
	
	return nil // 복구 성공으로 가정
}

// Name은 전략 이름을 반환합니다.
func (s *ProcessRecoveryStrategy) Name() string {
	return "ProcessRecovery"
}

// RecoveryManager는 에러 복구를 관리합니다.
type RecoveryManager struct {
	strategies []RecoveryStrategy
}

// NewRecoveryManager는 새로운 복구 매니저를 생성합니다.
func NewRecoveryManager() *RecoveryManager {
	return &RecoveryManager{
		strategies: []RecoveryStrategy{
			&ConfigRecoveryStrategy{},
			&NetworkRecoveryStrategy{},
			&ProcessRecoveryStrategy{},
		},
	}
}

// AddStrategy는 복구 전략을 추가합니다.
func (m *RecoveryManager) AddStrategy(strategy RecoveryStrategy) {
	m.strategies = append(m.strategies, strategy)
}

// TryRecover는 에러 복구를 시도합니다.
func (m *RecoveryManager) TryRecover(ctx context.Context, err error) error {
	for _, strategy := range m.strategies {
		if strategy.CanRecover(err) {
			LogErrorWithLevel(LogLevelInfo, NewInternalError("recovery", 
				fmt.Errorf("%s 전략으로 복구 시도", strategy.Name())))
			
			if recoveryErr := strategy.Recover(ctx, err); recoveryErr == nil {
				LogErrorWithLevel(LogLevelInfo, NewInternalError("recovery", 
					fmt.Errorf("%s 전략으로 복구 성공", strategy.Name())))
				return nil
			} else {
				LogErrorWithLevel(LogLevelWarn, NewInternalError("recovery", 
					fmt.Errorf("%s 전략으로 복구 실패: %v", strategy.Name(), recoveryErr)))
			}
		}
	}
	
	// 복구 불가능
	if cliErr, ok := err.(*CLIError); ok {
		cliErr.AddContext("recovery_attempted", true)
		cliErr.AddContext("recovery_success", false)
		cliErr.AddSuggestion("자동 복구가 불가능합니다. 수동으로 문제를 해결해주세요")
		return cliErr
	}
	
	return NewInternalError("recovery", fmt.Errorf("복구 불가능: %w", err))
}

// RetryWithRecovery는 재시도와 복구를 결합한 함수입니다.
func RetryWithRecovery(ctx context.Context, policy *RetryPolicy, manager *RecoveryManager, operation RetryableOperation) error {
	return RetryWithPolicy(ctx, policy, func(ctx context.Context, attempt int) error {
		err := operation(ctx, attempt)
		if err == nil {
			return nil
		}
		
		// 첫 번째 시도에서만 복구 시도
		if attempt == 1 && manager != nil {
			if recoveryErr := manager.TryRecover(ctx, err); recoveryErr == nil {
				// 복구 성공 후 다시 시도
				return operation(ctx, attempt)
			}
		}
		
		return err
	})
}

// GlobalRecoveryManager는 전역 복구 매니저입니다.
var GlobalRecoveryManager *RecoveryManager

// InitializeGlobalRecoveryManager는 전역 복구 매니저를 초기화합니다.
func InitializeGlobalRecoveryManager() {
	GlobalRecoveryManager = NewRecoveryManager()
}