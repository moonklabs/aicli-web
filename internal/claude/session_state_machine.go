package claude

import (
	"fmt"
	"sync"
)

// SessionStateMachine은 세션 상태 전이를 관리합니다
type SessionStateMachine struct {
	transitions map[SessionState][]SessionState
	mu          sync.RWMutex
}

// NewSessionStateMachine은 새로운 세션 상태 머신을 생성합니다
func NewSessionStateMachine() *SessionStateMachine {
	sm := &SessionStateMachine{
		transitions: make(map[SessionState][]SessionState),
	}

	// 상태 전이 규칙 정의
	sm.transitions[SessionStateCreated] = []SessionState{
		SessionStateInitializing,
		SessionStateError,
		SessionStateClosed,
	}

	sm.transitions[SessionStateInitializing] = []SessionState{
		SessionStateReady,
		SessionStateError,
		SessionStateClosed,
	}

	sm.transitions[SessionStateReady] = []SessionState{
		SessionStateActive,
		SessionStateIdle,
		SessionStateSuspended,
		SessionStateClosing,
		SessionStateError,
	}

	sm.transitions[SessionStateActive] = []SessionState{
		SessionStateIdle,
		SessionStateSuspended,
		SessionStateClosing,
		SessionStateError,
	}

	sm.transitions[SessionStateIdle] = []SessionState{
		SessionStateActive,
		SessionStateSuspended,
		SessionStateClosing,
		SessionStateError,
	}

	sm.transitions[SessionStateSuspended] = []SessionState{
		SessionStateReady,
		SessionStateClosing,
		SessionStateError,
	}

	sm.transitions[SessionStateClosing] = []SessionState{
		SessionStateClosed,
		SessionStateError,
	}

	sm.transitions[SessionStateClosed] = []SessionState{
		// 종료 상태에서는 전이 불가
	}

	sm.transitions[SessionStateError] = []SessionState{
		SessionStateClosing,
		SessionStateClosed,
	}

	return sm
}

// CanTransition은 상태 전이가 가능한지 확인합니다
func (sm *SessionStateMachine) CanTransition(from, to SessionState) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	allowedTransitions, exists := sm.transitions[from]
	if !exists {
		return fmt.Errorf("unknown state: %s", from)
	}

	for _, allowed := range allowedTransitions {
		if allowed == to {
			return nil
		}
	}

	return fmt.Errorf("invalid transition from %s to %s", from, to)
}

// GetAllowedTransitions은 현재 상태에서 가능한 전이 목록을 반환합니다
func (sm *SessionStateMachine) GetAllowedTransitions(from SessionState) []SessionState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if transitions, exists := sm.transitions[from]; exists {
		// 복사본 반환
		result := make([]SessionState, len(transitions))
		copy(result, transitions)
		return result
	}

	return []SessionState{}
}

// IsTerminalState는 종료 상태인지 확인합니다
func (sm *SessionStateMachine) IsTerminalState(state SessionState) bool {
	return state == SessionStateClosed
}

// IsActiveState는 활성 상태인지 확인합니다
func (sm *SessionStateMachine) IsActiveState(state SessionState) bool {
	return state == SessionStateActive || state == SessionStateIdle
}

// IsErrorState는 에러 상태인지 확인합니다
func (sm *SessionStateMachine) IsErrorState(state SessionState) bool {
	return state == SessionStateError
}

// CanRecover는 복구 가능한 상태인지 확인합니다
func (sm *SessionStateMachine) CanRecover(state SessionState) bool {
	// 에러 상태나 중단된 상태는 복구 가능
	return state == SessionStateError || state == SessionStateSuspended
}

// ValidateTransitionPath는 상태 전이 경로가 유효한지 검증합니다
func (sm *SessionStateMachine) ValidateTransitionPath(path []SessionState) error {
	if len(path) < 2 {
		return fmt.Errorf("path must contain at least 2 states")
	}

	for i := 0; i < len(path)-1; i++ {
		if err := sm.CanTransition(path[i], path[i+1]); err != nil {
			return fmt.Errorf("invalid transition at step %d: %w", i, err)
		}
	}

	return nil
}

// GetShortestPath는 두 상태 간의 최단 경로를 찾습니다 (BFS 알고리즘)
func (sm *SessionStateMachine) GetShortestPath(from, to SessionState) ([]SessionState, error) {
	if from == to {
		return []SessionState{from}, nil
	}

	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// BFS를 위한 큐와 방문 기록
	type node struct {
		state SessionState
		path  []SessionState
	}

	queue := []node{{state: from, path: []SessionState{from}}}
	visited := make(map[SessionState]bool)
	visited[from] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		// 현재 상태에서 가능한 전이 확인
		if transitions, exists := sm.transitions[current.state]; exists {
			for _, next := range transitions {
				if next == to {
					// 목표 상태에 도달
					path := append(current.path, next)
					return path, nil
				}

				if !visited[next] {
					visited[next] = true
					newPath := append([]SessionState{}, current.path...)
					newPath = append(newPath, next)
					queue = append(queue, node{state: next, path: newPath})
				}
			}
		}
	}

	return nil, fmt.Errorf("no path from %s to %s", from, to)
}

// AddTransition은 새로운 상태 전이를 추가합니다 (확장성을 위해)
func (sm *SessionStateMachine) AddTransition(from, to SessionState) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.transitions[from]; !exists {
		sm.transitions[from] = []SessionState{}
	}

	// 중복 확인
	for _, existing := range sm.transitions[from] {
		if existing == to {
			return // 이미 존재
		}
	}

	sm.transitions[from] = append(sm.transitions[from], to)
}

// RemoveTransition은 상태 전이를 제거합니다
func (sm *SessionStateMachine) RemoveTransition(from, to SessionState) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if transitions, exists := sm.transitions[from]; exists {
		filtered := []SessionState{}
		for _, t := range transitions {
			if t != to {
				filtered = append(filtered, t)
			}
		}
		sm.transitions[from] = filtered
	}
}

// GetStateDiagram은 상태 다이어그램을 문자열로 반환합니다 (디버깅용)
func (sm *SessionStateMachine) GetStateDiagram() string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	diagram := "Session State Diagram:\n"
	diagram += "=====================\n"

	for from, transitions := range sm.transitions {
		if len(transitions) > 0 {
			diagram += fmt.Sprintf("%s -> ", from)
			for i, to := range transitions {
				if i > 0 {
					diagram += ", "
				}
				diagram += to.String()
			}
			diagram += "\n"
		}
	}

	return diagram
}