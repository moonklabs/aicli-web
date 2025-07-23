package docker

import (
	"context"
	"time"

	"github.com/docker/docker/api/types"
)

// HealthChecker Docker daemon과 컨테이너의 헬스체크를 담당합니다.
type HealthChecker struct {
	client   *Client
	interval time.Duration
}

// NewHealthChecker 새로운 헬스체커를 생성합니다.
func NewHealthChecker(client *Client, interval time.Duration) *HealthChecker {
	if interval == 0 {
		interval = 30 * time.Second
	}

	return &HealthChecker{
		client:   client,
		interval: interval,
	}
}

// CheckDaemon Docker daemon 상태를 확인합니다.
func (h *HealthChecker) CheckDaemon(ctx context.Context) error {
	_, err := h.client.cli.Ping(ctx)
	return err
}

// StartMonitoring Docker daemon 모니터링을 시작합니다.
func (h *HealthChecker) StartMonitoring(ctx context.Context, callback func(error)) {
	ticker := time.NewTicker(h.interval)
	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				err := h.CheckDaemon(ctx)
				if callback != nil {
					callback(err)
				}
			}
		}
	}()
}

// GetSystemInfo Docker 시스템 정보를 조회합니다.
func (h *HealthChecker) GetSystemInfo(ctx context.Context) (*types.Info, error) {
	info, err := h.client.cli.Info(ctx)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

// GetVersion Docker 버전 정보를 조회합니다.
func (h *HealthChecker) GetVersion(ctx context.Context) (types.Version, error) {
	return h.client.cli.ServerVersion(ctx)
}

// CheckContainer 컨테이너의 헬스 상태를 확인합니다.
func (h *HealthChecker) CheckContainer(ctx context.Context, containerID string) (bool, error) {
	inspect, err := h.client.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return false, err
	}

	// 헬스체크가 설정된 경우
	if inspect.State.Health != nil {
		return inspect.State.Health.Status == "healthy", nil
	}

	// 헬스체크가 없으면 실행 상태만 확인
	return inspect.State.Running, nil
}

// WaitHealthy 컨테이너가 healthy 상태가 될 때까지 대기합니다.
func (h *HealthChecker) WaitHealthy(ctx context.Context, containerID string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			healthy, err := h.CheckContainer(ctx, containerID)
			if err != nil {
				return err
			}
			if healthy {
				return nil
			}
		}
	}
}

// HealthStatus 헬스체크 결과 구조체
type HealthStatus struct {
	ContainerID string    `json:"container_id"`
	Healthy     bool      `json:"healthy"`
	Status      string    `json:"status"`
	CheckedAt   time.Time `json:"checked_at"`
	Error       string    `json:"error,omitempty"`
}

// CheckMultipleContainers 여러 컨테이너의 헬스 상태를 확인합니다.
func (h *HealthChecker) CheckMultipleContainers(ctx context.Context, containerIDs []string) ([]HealthStatus, error) {
	results := make([]HealthStatus, len(containerIDs))
	
	for i, containerID := range containerIDs {
		status := HealthStatus{
			ContainerID: containerID,
			CheckedAt:   time.Now(),
		}

		healthy, err := h.CheckContainer(ctx, containerID)
		if err != nil {
			status.Error = err.Error()
			status.Healthy = false
		} else {
			status.Healthy = healthy
			if healthy {
				status.Status = "healthy"
			} else {
				status.Status = "unhealthy"
			}
		}

		results[i] = status
	}

	return results, nil
}