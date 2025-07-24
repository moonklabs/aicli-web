package claude

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// HealthStatus 헬스체크 상태
type HealthStatus struct {
	// Healthy 프로세스가 정상인지 여부
	Healthy bool
	// LastCheck 마지막 체크 시간
	LastCheck time.Time
	// Message 상태 메시지
	Message string
	// Metrics 프로세스 메트릭
	Metrics *ProcessMetrics
}

// ProcessMetrics 프로세스 메트릭 정보
type ProcessMetrics struct {
	// CPUUsage CPU 사용률 (0-100)
	CPUUsage float64
	// MemoryUsage 메모리 사용량 (바이트)
	MemoryUsage int64
	// Uptime 실행 시간
	Uptime time.Duration
	// ResponseTime 응답 시간
	ResponseTime time.Duration
}

// HealthChecker 프로세스 헬스체크 인터페이스
type HealthChecker interface {
	// Start 헬스체크 시작
	Start(ctx context.Context, process ProcessManager, interval time.Duration)
	// Stop 헬스체크 중지
	Stop()
	// CheckHealth 즉시 헬스체크 수행
	CheckHealth(ctx context.Context, process ProcessManager) error
	// GetHealthStatus 현재 헬스 상태 반환
	GetHealthStatus() HealthStatus
	// RegisterHealthHandler 헬스 핸들러 등록
	RegisterHealthHandler(handler HealthHandler)
	// GetOverallHealth 풀 기반 전체 헬스 상태 반환 (PoolHealthChecker만 구현)
	GetOverallHealth() interface{}
}

// HealthHandler 헬스 상태 변경 핸들러
type HealthHandler func(status HealthStatus)

// healthChecker 헬스체크 구현
type healthChecker struct {
	mu          sync.RWMutex
	status      HealthStatus
	handlers    []HealthHandler
	stopCh      chan struct{}
	stopped     bool
	logger      *logrus.Logger
	checkFunc   HealthCheckFunc
}

// HealthCheckFunc 헬스체크 함수 타입
type HealthCheckFunc func(ctx context.Context, process ProcessManager) error

// NewHealthChecker 새로운 헬스체커를 생성합니다
func NewHealthChecker(logger *logrus.Logger) HealthChecker {
	if logger == nil {
		logger = logrus.StandardLogger()
	}
	
	return &healthChecker{
		logger:    logger,
		stopCh:    make(chan struct{}),
		status: HealthStatus{
			Healthy:   false,
			LastCheck: time.Now(),
			Message:   "헬스체크가 아직 수행되지 않았습니다",
		},
		checkFunc: defaultHealthCheck,
	}
}

// Start 주기적인 헬스체크를 시작합니다
func (hc *healthChecker) Start(ctx context.Context, process ProcessManager, interval time.Duration) {
	hc.mu.Lock()
	if hc.stopped {
		hc.stopCh = make(chan struct{})
		hc.stopped = false
	}
	hc.mu.Unlock()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// 초기 헬스체크
	if err := hc.CheckHealth(ctx, process); err != nil {
		hc.logger.WithError(err).Warn("초기 헬스체크 실패")
	}

	for {
		select {
		case <-ctx.Done():
			hc.logger.Info("헬스체크 중지: 컨텍스트 취소")
			return
		case <-hc.stopCh:
			hc.logger.Info("헬스체크 중지: 명시적 중지 요청")
			return
		case <-ticker.C:
			if err := hc.CheckHealth(ctx, process); err != nil {
				hc.logger.WithError(err).Warn("헬스체크 실패")
			}
		}
	}
}

// Stop 헬스체크를 중지합니다
func (hc *healthChecker) Stop() {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	if !hc.stopped {
		close(hc.stopCh)
		hc.stopped = true
	}
}

// CheckHealth 즉시 헬스체크를 수행합니다
func (hc *healthChecker) CheckHealth(ctx context.Context, process ProcessManager) error {
	// 프로세스 상태 확인
	if !process.IsRunning() {
		hc.updateStatus(HealthStatus{
			Healthy:   false,
			LastCheck: time.Now(),
			Message:   "프로세스가 실행 중이 아닙니다",
		})
		return fmt.Errorf("프로세스가 실행 중이 아닙니다")
	}

	// 커스텀 헬스체크 함수 실행
	startTime := time.Now()
	err := hc.checkFunc(ctx, process)
	responseTime := time.Since(startTime)

	if err != nil {
		hc.updateStatus(HealthStatus{
			Healthy:   false,
			LastCheck: time.Now(),
			Message:   fmt.Sprintf("헬스체크 실패: %v", err),
			Metrics: &ProcessMetrics{
				ResponseTime: responseTime,
			},
		})
		return err
	}

	// 프로세스 메트릭 수집 (실제 구현에서는 시스템 정보 수집)
	metrics := &ProcessMetrics{
		ResponseTime: responseTime,
		Uptime:       time.Since(startTime), // 실제로는 프로세스 시작 시간 사용
		CPUUsage:     0.0,                   // TODO: 실제 CPU 사용률 수집
		MemoryUsage:  0,                      // TODO: 실제 메모리 사용량 수집
	}

	hc.updateStatus(HealthStatus{
		Healthy:   true,
		LastCheck: time.Now(),
		Message:   "프로세스가 정상적으로 작동 중입니다",
		Metrics:   metrics,
	})

	return nil
}

// GetHealthStatus 현재 헬스 상태를 반환합니다
func (hc *healthChecker) GetHealthStatus() HealthStatus {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	return hc.status
}

// RegisterHealthHandler 헬스 핸들러를 등록합니다
func (hc *healthChecker) RegisterHealthHandler(handler HealthHandler) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.handlers = append(hc.handlers, handler)
}

// updateStatus 상태를 업데이트하고 핸들러를 호출합니다
func (hc *healthChecker) updateStatus(status HealthStatus) {
	hc.mu.Lock()
	oldHealthy := hc.status.Healthy
	hc.status = status
	handlers := make([]HealthHandler, len(hc.handlers))
	copy(handlers, hc.handlers)
	hc.mu.Unlock()

	// 상태 변경 로깅
	if oldHealthy != status.Healthy {
		if status.Healthy {
			hc.logger.Info("프로세스 헬스 상태: 정상")
		} else {
			hc.logger.Warn("프로세스 헬스 상태: 비정상")
		}
	}

	// 핸들러 호출
	for _, handler := range handlers {
		handler(status)
	}
}

// defaultHealthCheck 기본 헬스체크 함수
func defaultHealthCheck(ctx context.Context, process ProcessManager) error {
	// 기본 헬스체크는 ProcessManager의 HealthCheck 메서드 사용
	return process.HealthCheck()
}

// PingHealthCheck ping/pong 방식의 헬스체크 함수를 생성합니다
func PingHealthCheck(pingCmd string) HealthCheckFunc {
	return func(ctx context.Context, process ProcessManager) error {
		// TODO: 실제 ping/pong 구현
		// 현재는 기본 헬스체크 사용
		return process.HealthCheck()
	}
}

func (hc *healthChecker) GetOverallHealth() interface{} {
	// PoolHealthChecker가 아닌 경우 지원하지 않음
	return nil // 또는 panic("not implemented")
}