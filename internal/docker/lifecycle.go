package docker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
)

// LifecycleManager 컨테이너 생명주기 이벤트를 관리합니다.
type LifecycleManager struct {
	client       *Client
	eventChan    chan ContainerEvent
	subscribers  map[string][]ContainerEventHandler
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
}

// ContainerEvent 컨테이너 이벤트 구조체입니다.
type ContainerEvent struct {
	ContainerID   string            `json:"container_id"`
	WorkspaceID   string            `json:"workspace_id"`
	Type          ContainerEventType `json:"type"`
	Status        ContainerState    `json:"status"`
	Message       string            `json:"message"`
	Timestamp     time.Time         `json:"timestamp"`
	Attributes    map[string]string `json:"attributes,omitempty"`
}

// ContainerEventType 컨테이너 이벤트 타입입니다.
type ContainerEventType string

const (
	EventTypeCreate     ContainerEventType = "create"
	EventTypeStart      ContainerEventType = "start"
	EventTypeStop       ContainerEventType = "stop"
	EventTypeRestart    ContainerEventType = "restart"
	EventTypeDestroy    ContainerEventType = "destroy"
	EventTypeDie        ContainerEventType = "die"
	EventTypePause      ContainerEventType = "pause"
	EventTypeUnpause    ContainerEventType = "unpause"
	EventTypeHealthcheck ContainerEventType = "health_status"
)

// ContainerEventHandler 컨테이너 이벤트 핸들러 함수 타입입니다.
type ContainerEventHandler func(event ContainerEvent)

// NewLifecycleManager 새로운 생명주기 매니저를 생성합니다.
func NewLifecycleManager(client *Client) *LifecycleManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	lm := &LifecycleManager{
		client:      client,
		eventChan:   make(chan ContainerEvent, 100),
		subscribers: make(map[string][]ContainerEventHandler),
		ctx:         ctx,
		cancel:      cancel,
	}
	
	// 이벤트 모니터링 시작
	go lm.monitorEvents()
	go lm.processEvents()
	
	return lm
}

// Subscribe 워크스페이스의 컨테이너 이벤트를 구독합니다.
func (lm *LifecycleManager) Subscribe(workspaceID string, handler ContainerEventHandler) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	
	lm.subscribers[workspaceID] = append(lm.subscribers[workspaceID], handler)
}

// Unsubscribe 워크스페이스의 이벤트 구독을 해제합니다.
func (lm *LifecycleManager) Unsubscribe(workspaceID string) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	
	delete(lm.subscribers, workspaceID)
}

// Close 생명주기 매니저를 종료합니다.
func (lm *LifecycleManager) Close() {
	lm.cancel()
	close(lm.eventChan)
}

// monitorEvents Docker 이벤트를 모니터링합니다.
func (lm *LifecycleManager) monitorEvents() {
	// aicli 관리 컨테이너만 필터링
	eventFilters := filters.NewArgs()
	eventFilters.Add("type", "container")
	eventFilters.Add("label", fmt.Sprintf("%s.managed=true", lm.client.labelPrefix))
	
	eventOptions := types.EventsOptions{
		Filters: eventFilters,
	}
	
	eventChan, errChan := lm.client.cli.Events(lm.ctx, eventOptions)
	
	for {
		select {
		case <-lm.ctx.Done():
			return
		case err := <-errChan:
			if err != nil {
				// 에러 발생시 잠시 대기 후 재시도
				time.Sleep(5 * time.Second)
				go lm.monitorEvents()
				return
			}
		case event := <-eventChan:
			lm.handleDockerEvent(event)
		}
	}
}

// handleDockerEvent Docker 이벤트를 처리합니다.
func (lm *LifecycleManager) handleDockerEvent(dockerEvent events.Message) {
	// 워크스페이스 ID 추출
	workspaceID, exists := dockerEvent.Actor.Attributes[lm.client.labelKey("workspace.id")]
	if !exists {
		return
	}
	
	// 컨테이너 이벤트로 변환
	containerEvent := ContainerEvent{
		ContainerID: dockerEvent.Actor.ID,
		WorkspaceID: workspaceID,
		Type:        ContainerEventType(dockerEvent.Action),
		Status:      lm.mapDockerStatusToContainerState(dockerEvent.Status),
		Message:     fmt.Sprintf("Container %s: %s", dockerEvent.Action, dockerEvent.Status),
		Timestamp:   time.Unix(dockerEvent.Time, dockerEvent.TimeNano),
		Attributes:  dockerEvent.Actor.Attributes,
	}
	
	select {
	case lm.eventChan <- containerEvent:
	case <-lm.ctx.Done():
		return
	default:
		// 채널이 가득 찬 경우 가장 오래된 이벤트를 버림
		select {
		case <-lm.eventChan:
		default:
		}
		select {
		case lm.eventChan <- containerEvent:
		default:
		}
	}
}

// processEvents 이벤트를 처리하고 구독자에게 전달합니다.
func (lm *LifecycleManager) processEvents() {
	for {
		select {
		case <-lm.ctx.Done():
			return
		case event := <-lm.eventChan:
			lm.notifySubscribers(event)
		}
	}
}

// notifySubscribers 구독자들에게 이벤트를 알립니다.
func (lm *LifecycleManager) notifySubscribers(event ContainerEvent) {
	lm.mu.RLock()
	handlers, exists := lm.subscribers[event.WorkspaceID]
	lm.mu.RUnlock()
	
	if !exists {
		return
	}
	
	// 각 핸들러를 별도 고루틴에서 실행
	for _, handler := range handlers {
		go func(h ContainerEventHandler) {
			defer func() {
				if r := recover(); r != nil {
					// 핸들러에서 패닉이 발생해도 다른 핸들러에 영향을 주지 않음
				}
			}()
			h(event)
		}(handler)
	}
}

// mapDockerStatusToContainerState Docker 상태를 컨테이너 상태로 매핑합니다.
func (lm *LifecycleManager) mapDockerStatusToContainerState(dockerStatus string) ContainerState {
	switch dockerStatus {
	case "create":
		return ContainerStateCreated
	case "start":
		return ContainerStateRunning
	case "running":
		return ContainerStateRunning
	case "stop":
		return ContainerStateExited
	case "die":
		return ContainerStateExited
	case "pause":
		return ContainerStatePaused
	case "unpause":
		return ContainerStateRunning
	case "destroy":
		return ContainerStateRemoving
	case "restart":
		return ContainerStateRestarting
	default:
		return ContainerState(dockerStatus)
	}
}

// GetContainerHistory 컨테이너의 이벤트 히스토리를 조회합니다.
func (lm *LifecycleManager) GetContainerHistory(ctx context.Context, containerID string, since time.Time) ([]ContainerEvent, error) {
	// Docker events API를 통해 특정 컨테이너의 이벤트 히스토리 조회
	eventFilters := filters.NewArgs()
	eventFilters.Add("type", "container")
	eventFilters.Add("container", containerID)
	
	eventOptions := types.EventsOptions{
		Since:   fmt.Sprintf("%d", since.Unix()),
		Until:   fmt.Sprintf("%d", time.Now().Unix()),
		Filters: eventFilters,
	}
	
	eventChan, errChan := lm.client.cli.Events(ctx, eventOptions)
	var events []ContainerEvent
	
	for {
		select {
		case dockerEvent := <-eventChan:
			if dockerEvent.Type == "" {
				// 이벤트 스트림 종료
				return events, nil
			}
			
			// 워크스페이스 ID 추출
			workspaceID := dockerEvent.Actor.Attributes[lm.client.labelKey("workspace.id")]
			
			event := ContainerEvent{
				ContainerID: dockerEvent.Actor.ID,
				WorkspaceID: workspaceID,
				Type:        ContainerEventType(dockerEvent.Action),
				Status:      lm.mapDockerStatusToContainerState(dockerEvent.Status),
				Message:     fmt.Sprintf("Container %s: %s", dockerEvent.Action, dockerEvent.Status),
				Timestamp:   time.Unix(dockerEvent.Time, dockerEvent.TimeNano),
				Attributes:  dockerEvent.Actor.Attributes,
			}
			
			events = append(events, event)
			
		case err := <-errChan:
			if err != nil {
				return events, fmt.Errorf("get container history: %w", err)
			}
			
		case <-ctx.Done():
			return events, ctx.Err()
		}
	}
}

// WaitForContainerState 컨테이너가 특정 상태가 될 때까지 대기합니다.
func (lm *LifecycleManager) WaitForContainerState(ctx context.Context, containerID string, targetState ContainerState, timeout time.Duration) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	
	// 현재 상태 확인
	inspect, err := lm.client.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return fmt.Errorf("inspect container: %w", err)
	}
	
	currentState := ContainerState(inspect.State.Status)
	if currentState == targetState {
		return nil
	}
	
	// 상태 변경을 기다림
	eventChan := make(chan ContainerEvent, 10)
	handler := func(event ContainerEvent) {
		if event.ContainerID == containerID {
			eventChan <- event
		}
	}
	
	// 임시 구독
	workspaceID := inspect.Config.Labels[lm.client.labelKey("workspace.id")]
	lm.Subscribe(workspaceID, handler)
	defer lm.Unsubscribe(workspaceID)
	
	for {
		select {
		case <-timeoutCtx.Done():
			return fmt.Errorf("timeout waiting for container state %s", targetState)
		case event := <-eventChan:
			if event.Status == targetState {
				return nil
			}
		}
	}
}