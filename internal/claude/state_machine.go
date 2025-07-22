package claude

import (
	"fmt"
	"sync"
)

// ProcessStateTransition 프로세스 상태 전환 정의
type ProcessStateTransition struct {
	From ProcessStatus
	To   ProcessStatus
}

// StateMachine 프로세스 상태 머신
type StateMachine struct {
	current     ProcessStatus
	transitions map[ProcessStateTransition]bool
	listeners   []StateChangeListener
	mutex       sync.RWMutex
}

// StateChangeListener 상태 변경 리스너 인터페이스
type StateChangeListener interface {
	OnStateChange(from, to ProcessStatus)
}

// StateChangeFunc 함수 타입의 상태 변경 리스너
type StateChangeFunc func(from, to ProcessStatus)

// OnStateChange StateChangeListener 인터페이스 구현
func (f StateChangeFunc) OnStateChange(from, to ProcessStatus) {
	f(from, to)
}

// NewStateMachine 새로운 상태 머신을 생성합니다
func NewStateMachine() *StateMachine {
	sm := &StateMachine{
		current: StatusStopped,
		transitions: map[StateTransition]bool{
			// 정상적인 생명주기 전환
			{StatusStopped, StatusStarting}:  true,
			{StatusStarting, StatusRunning}:  true,
			{StatusRunning, StatusStopping}:  true,
			{StatusStopping, StatusStopped}:  true,
			
			// 에러 상태로의 전환
			{StatusStarting, StatusError}:    true,
			{StatusRunning, StatusError}:     true,
			{StatusStopping, StatusError}:    true,
			
			// 에러 상태에서의 전환
			{StatusError, StatusStopped}:     true,
			{StatusError, StatusStarting}:    true,
			
			// 강제 종료
			{StatusStarting, StatusStopped}:  true,
			{StatusRunning, StatusStopped}:   true,
			{StatusError, StatusStopped}:     true,
		},
		listeners: make([]StateChangeListener, 0),
	}
	return sm
}

// GetState 현재 상태를 반환합니다
func (sm *StateMachine) GetState() ProcessStatus {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return sm.current
}

// TransitionTo 지정된 상태로 전환을 시도합니다
func (sm *StateMachine) TransitionTo(newState ProcessStatus) error {
	sm.mutex.Lock()
	
	transition := StateTransition{
		From: sm.current,
		To:   newState,
	}
	
	// 전환 가능 여부 확인
	if !sm.transitions[transition] {
		sm.mutex.Unlock()
		return fmt.Errorf("잘못된 상태 전환: %s -> %s", sm.current, newState)
	}
	
	oldState := sm.current
	sm.current = newState
	listeners := make([]StateChangeListener, len(sm.listeners))
	copy(listeners, sm.listeners)
	
	sm.mutex.Unlock()
	
	// 리스너들에게 알림 (락 밖에서 실행)
	for _, listener := range listeners {
		listener.OnStateChange(oldState, newState)
	}
	
	return nil
}

// MustTransitionTo 지정된 상태로 강제 전환합니다 (패닉 발생 가능)
func (sm *StateMachine) MustTransitionTo(newState ProcessStatus) {
	if err := sm.TransitionTo(newState); err != nil {
		panic(err)
	}
}

// CanTransitionTo 지정된 상태로 전환 가능한지 확인합니다
func (sm *StateMachine) CanTransitionTo(newState ProcessStatus) bool {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	
	transition := StateTransition{
		From: sm.current,
		To:   newState,
	}
	
	return sm.transitions[transition]
}

// AddListener 상태 변경 리스너를 추가합니다
func (sm *StateMachine) AddListener(listener StateChangeListener) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.listeners = append(sm.listeners, listener)
}

// RemoveListener 상태 변경 리스너를 제거합니다
func (sm *StateMachine) RemoveListener(listener StateChangeListener) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	
	for i, l := range sm.listeners {
		if l == listener {
			sm.listeners = append(sm.listeners[:i], sm.listeners[i+1:]...)
			break
		}
	}
}

// IsTerminal 현재 상태가 종료 상태인지 확인합니다
func (sm *StateMachine) IsTerminal() bool {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	
	return sm.current == StatusStopped || sm.current == StatusError
}

// IsRunning 프로세스가 실행 중인지 확인합니다
func (sm *StateMachine) IsRunning() bool {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	
	return sm.current == StatusRunning
}

// Reset 상태 머신을 초기 상태로 리셋합니다
func (sm *StateMachine) Reset() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	
	sm.current = StatusStopped
}

// ValidateTransitions 모든 상태 전환이 유효한지 검증합니다
func (sm *StateMachine) ValidateTransitions() error {
	// 각 상태에서 최소 하나의 전환이 가능한지 확인
	states := []ProcessStatus{
		StatusStopped, StatusStarting, StatusRunning, 
		StatusStopping, StatusError,
	}
	
	for _, state := range states {
		hasTransition := false
		for transition := range sm.transitions {
			if transition.From == state {
				hasTransition = true
				break
			}
		}
		if !hasTransition && state != StatusStopped {
			return fmt.Errorf("상태 %s에서 전환 가능한 상태가 없습니다", state)
		}
	}
	
	return nil
}