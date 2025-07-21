package claude

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sijinhui/aicli-web/internal/models"
	"github.com/sijinhui/aicli-web/internal/storage"
)

// SessionManager 인터페이스는 Claude CLI 세션을 관리합니다
type SessionManager interface {
	CreateSession(ctx context.Context, config SessionConfig) (*Session, error)
	GetSession(sessionID string) (*Session, error)
	UpdateSession(sessionID string, updates SessionUpdate) error
	CloseSession(sessionID string) error
	ListSessions(filter SessionFilter) ([]*Session, error)
}

// Session은 Claude CLI 세션을 나타냅니다
type Session struct {
	ID          string                 `json:"id"`
	WorkspaceID string                 `json:"workspace_id"`
	UserID      string                 `json:"user_id"`
	Config      SessionConfig          `json:"config"`
	State       SessionState           `json:"state"`
	Process     *Process               `json:"-"` // 프로세스는 직렬화하지 않음
	Created     time.Time              `json:"created"`
	LastActive  time.Time              `json:"last_active"`
	Metadata    map[string]interface{} `json:"metadata"`
	mu          sync.RWMutex           // 동시성 제어
}

// SessionConfig는 세션 설정을 정의합니다
type SessionConfig struct {
	// 기본 설정
	WorkingDir   string  `json:"working_dir" validate:"required,dir"`
	SystemPrompt string  `json:"system_prompt"`
	MaxTurns     int     `json:"max_turns" validate:"min=1,max=1000"`
	Temperature  float64 `json:"temperature" validate:"min=0,max=2"`

	// 도구 설정
	AllowedTools []string      `json:"allowed_tools"`
	ToolTimeout  time.Duration `json:"tool_timeout" validate:"min=1s,max=5m"`

	// 환경 설정
	Environment map[string]string `json:"environment"`
	OAuthToken  string            `json:"-"` // 보안상 직렬화하지 않음

	// 리소스 제한
	MaxMemory   int64         `json:"max_memory" validate:"min=0"`   // bytes
	MaxCPU      float64       `json:"max_cpu" validate:"min=0,max=1"` // 0-1 범위
	MaxDuration time.Duration `json:"max_duration" validate:"min=1m,max=24h"`
}

// Validate는 설정의 유효성을 검증합니다
func (c SessionConfig) Validate() error {
	// 필수 필드 검증
	if c.WorkingDir == "" {
		return errors.New("working directory is required")
	}

	// 값 범위 검증
	if c.MaxTurns < 1 || c.MaxTurns > 1000 {
		return fmt.Errorf("max_turns must be between 1 and 1000, got %d", c.MaxTurns)
	}

	if c.Temperature < 0 || c.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2, got %f", c.Temperature)
	}

	if c.ToolTimeout > 0 && (c.ToolTimeout < time.Second || c.ToolTimeout > 5*time.Minute) {
		return fmt.Errorf("tool_timeout must be between 1s and 5m, got %v", c.ToolTimeout)
	}

	if c.MaxCPU < 0 || c.MaxCPU > 1 {
		return fmt.Errorf("max_cpu must be between 0 and 1, got %f", c.MaxCPU)
	}

	if c.MaxDuration > 0 && (c.MaxDuration < time.Minute || c.MaxDuration > 24*time.Hour) {
		return fmt.Errorf("max_duration must be between 1m and 24h, got %v", c.MaxDuration)
	}

	// 도구 권한 검증
	validTools := map[string]bool{
		"code_interpreter": true,
		"file_search":      true,
		"function":         true,
	}
	for _, tool := range c.AllowedTools {
		if !validTools[tool] {
			return fmt.Errorf("invalid tool: %s", tool)
		}
	}

	return nil
}

// SessionState는 세션 상태를 나타냅니다
type SessionState int

const (
	SessionStateCreated SessionState = iota
	SessionStateInitializing
	SessionStateReady
	SessionStateActive
	SessionStateIdle
	SessionStateSuspended
	SessionStateClosing
	SessionStateClosed
	SessionStateError
)

// String은 SessionState의 문자열 표현을 반환합니다
func (s SessionState) String() string {
	states := []string{
		"created",
		"initializing",
		"ready",
		"active",
		"idle",
		"suspended",
		"closing",
		"closed",
		"error",
	}
	if int(s) < len(states) {
		return states[s]
	}
	return "unknown"
}

// SessionUpdate는 세션 업데이트 정보를 담습니다
type SessionUpdate struct {
	State    *SessionState                  `json:"state,omitempty"`
	Config   *SessionConfig                 `json:"config,omitempty"`
	Metadata map[string]interface{}         `json:"metadata,omitempty"`
	UpdateFn func(*Session) error          `json:"-"` // 커스텀 업데이트 함수
}

// SessionFilter는 세션 필터링 옵션을 정의합니다
type SessionFilter struct {
	WorkspaceID string
	UserID      string
	State       *SessionState
	Active      *bool // true: 활성 세션만, false: 비활성 세션만, nil: 전체
}

// sessionManager는 SessionManager 인터페이스의 구현체입니다
type sessionManager struct {
	sessions       map[string]*Session
	processManager ProcessManager
	stateMachine   *SessionStateMachine
	store          storage.Storage
	eventBus       *SessionEventBus
	mu             sync.RWMutex
}

// NewSessionManager는 새로운 SessionManager를 생성합니다
func NewSessionManager(processManager ProcessManager, store storage.Storage) SessionManager {
	sm := &sessionManager{
		sessions:       make(map[string]*Session),
		processManager: processManager,
		stateMachine:   NewSessionStateMachine(),
		store:          store,
		eventBus:       NewSessionEventBus(1000),
	}
	
	// 기본 이벤트 로거 추가
	sm.eventBus.Subscribe("", NewSessionEventLogger(nil))
	
	return sm
}

// CreateSession은 새로운 세션을 생성합니다
func (sm *sessionManager) CreateSession(ctx context.Context, config SessionConfig) (*Session, error) {
	// 설정 검증
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// 세션 ID 생성
	sessionID, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}

	// 세션 생성
	session := &Session{
		ID:         sessionID.String(),
		Config:     config,
		State:      SessionStateCreated,
		Created:    time.Now(),
		LastActive: time.Now(),
		Metadata:   make(map[string]interface{}),
	}

	// 메모리에 저장
	sm.mu.Lock()
	sm.sessions[session.ID] = session
	sm.mu.Unlock()

	// 영구 저장소에 저장
	if sm.store != nil {
		sessionModel := &models.Session{
			ProjectID:  session.WorkspaceID,
			Status:     models.SessionStatus(session.State.String()),
			StartedAt:  &session.Created,
			LastActive: session.LastActive,
			Metadata:   make(map[string]string),
		}
		sessionModel.ID = session.ID
		
		if err := sm.store.Session().Create(ctx, sessionModel); err != nil {
			// 메모리에서 롤백
			sm.mu.Lock()
			delete(sm.sessions, session.ID)
			sm.mu.Unlock()
			return nil, fmt.Errorf("failed to persist session: %w", err)
		}
	}
	
	// 이벤트 발행
	sm.eventBus.Publish(SessionEvent{
		SessionID: session.ID,
		Type:      SessionEventCreated,
		Data:      session,
	})

	// 상태를 Initializing으로 변경
	if err := sm.updateSessionState(session.ID, SessionStateInitializing); err != nil {
		return nil, err
	}

	// 프로세스 생성
	processConfig := ProcessConfig{
		Command:      "claude",
		Args:         []string{},
		WorkingDir:   config.WorkingDir,
		Environment:  config.Environment,
		OAuthToken:   config.OAuthToken,
		SystemPrompt: config.SystemPrompt,
		MaxMemory:    config.MaxMemory,
		MaxCPU:       config.MaxCPU,
	}

	process, err := sm.processManager.CreateProcess(ctx, processConfig)
	if err != nil {
		sm.updateSessionState(session.ID, SessionStateError)
		return nil, fmt.Errorf("failed to create process: %w", err)
	}

	session.Process = process

	// 상태를 Ready로 변경
	if err := sm.updateSessionState(session.ID, SessionStateReady); err != nil {
		return nil, err
	}

	return session, nil
}

// GetSession은 세션을 조회합니다
func (sm *sessionManager) GetSession(sessionID string) (*Session, error) {
	sm.mu.RLock()
	session, exists := sm.sessions[sessionID]
	sm.mu.RUnlock()

	if !exists {
		// 영구 저장소에서 조회
		if sm.store != nil {
			sessionModel, err := sm.store.Session().GetByID(context.Background(), sessionID)
			if err != nil {
				return nil, fmt.Errorf("session not found: %s", sessionID)
			}
			
			// 모델을 세션으로 변환
			session = &Session{
				ID:         sessionModel.ID,
				WorkspaceID: sessionModel.ProjectID,
				State:      SessionStateFromString(string(sessionModel.Status)),
				Created:    sessionModel.CreatedAt,
				LastActive: sessionModel.LastActive,
				Metadata:   make(map[string]interface{}),
			}
			
			// 메타데이터 변환
			for k, v := range sessionModel.Metadata {
				session.Metadata[k] = v
			}
			
			// 메모리에 캐시
			sm.mu.Lock()
			sm.sessions[session.ID] = session
			sm.mu.Unlock()
		} else {
			return nil, fmt.Errorf("session not found: %s", sessionID)
		}
	}

	return session, nil
}

// UpdateSession은 세션을 업데이트합니다
func (sm *sessionManager) UpdateSession(sessionID string, updates SessionUpdate) error {
	session, err := sm.GetSession(sessionID)
	if err != nil {
		return err
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	// 이전 상태 저장
	oldState := session.State

	// 상태 업데이트
	if updates.State != nil {
		if err := sm.stateMachine.CanTransition(session.State, *updates.State); err != nil {
			return fmt.Errorf("invalid state transition: %w", err)
		}
		session.State = *updates.State
	}

	// 설정 업데이트
	if updates.Config != nil {
		if err := updates.Config.Validate(); err != nil {
			return fmt.Errorf("invalid config update: %w", err)
		}
		session.Config = *updates.Config
	}

	// 메타데이터 업데이트
	if updates.Metadata != nil {
		for k, v := range updates.Metadata {
			session.Metadata[k] = v
		}
	}

	// 커스텀 업데이트 함수 실행
	if updates.UpdateFn != nil {
		if err := updates.UpdateFn(session); err != nil {
			return fmt.Errorf("update function failed: %w", err)
		}
	}

	// 활동 시간 업데이트
	session.LastActive = time.Now()

	// 영구 저장소에 업데이트
	if sm.store != nil {
		sessionModel, err := sm.store.Session().GetByID(context.Background(), sessionID)
		if err != nil {
			return fmt.Errorf("failed to get session from store: %w", err)
		}
		
		sessionModel.Status = models.SessionStatus(session.State.String())
		sessionModel.LastActive = session.LastActive
		
		if err := sm.store.Session().Update(context.Background(), sessionModel); err != nil {
			return fmt.Errorf("failed to persist session update: %w", err)
		}
	}
	
	// 이벤트 발행
	if updates.State != nil {
		sm.eventBus.Publish(SessionEvent{
			SessionID: sessionID,
			Type:      SessionEventStateChanged,
			Data:      StateChangeData{OldState: oldState, NewState: *updates.State},
		})
	}
	if updates.Config != nil {
		sm.eventBus.Publish(SessionEvent{
			SessionID: sessionID,
			Type:      SessionEventConfigUpdated,
			Data:      updates.Config,
		})
	}
	if updates.Metadata != nil {
		sm.eventBus.Publish(SessionEvent{
			SessionID: sessionID,
			Type:      SessionEventMetadataUpdated,
			Data:      updates.Metadata,
		})
	}

	return nil
}

// CloseSession은 세션을 종료합니다
func (sm *sessionManager) CloseSession(sessionID string) error {
	session, err := sm.GetSession(sessionID)
	if err != nil {
		return err
	}

	// 상태를 Closing으로 변경
	if err := sm.updateSessionState(sessionID, SessionStateClosing); err != nil {
		return err
	}

	// 프로세스 종료
	if session.Process != nil {
		if err := sm.processManager.TerminateProcess(session.Process.ID); err != nil {
			// 에러가 발생해도 계속 진행
			fmt.Printf("Failed to terminate process %s: %v\n", session.Process.ID, err)
		}
	}

	// 상태를 Closed로 변경
	if err := sm.updateSessionState(sessionID, SessionStateClosed); err != nil {
		return err
	}

	// 메모리에서 제거
	sm.mu.Lock()
	delete(sm.sessions, sessionID)
	sm.mu.Unlock()

	// 영구 저장소 업데이트
	if sm.store != nil {
		sessionModel, err := sm.store.Session().GetByID(context.Background(), sessionID)
		if err != nil {
			return fmt.Errorf("failed to get session from store: %w", err)
		}
		
		now := time.Now()
		sessionModel.Status = models.SessionEnded
		sessionModel.EndedAt = &now
		
		if err := sm.store.Session().Update(context.Background(), sessionModel); err != nil {
			return fmt.Errorf("failed to persist session close: %w", err)
		}
	}
	
	// 이벤트 발행
	sm.eventBus.Publish(SessionEvent{
		SessionID: sessionID,
		Type:      SessionEventClosed,
	})

	return nil
}

// ListSessions은 필터에 맞는 세션 목록을 반환합니다
func (sm *sessionManager) ListSessions(filter SessionFilter) ([]*Session, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var sessions []*Session

	for _, session := range sm.sessions {
		// WorkspaceID 필터
		if filter.WorkspaceID != "" && session.WorkspaceID != filter.WorkspaceID {
			continue
		}

		// UserID 필터
		if filter.UserID != "" && session.UserID != filter.UserID {
			continue
		}

		// State 필터
		if filter.State != nil && session.State != *filter.State {
			continue
		}

		// Active 필터
		if filter.Active != nil {
			isActive := session.State == SessionStateActive || session.State == SessionStateIdle
			if *filter.Active != isActive {
				continue
			}
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

// updateSessionState는 세션 상태를 업데이트하는 헬퍼 함수입니다
func (sm *sessionManager) updateSessionState(sessionID string, newState SessionState) error {
	return sm.UpdateSession(sessionID, SessionUpdate{
		State: &newState,
	})
}

// SessionStateFromString은 문자열을 SessionState로 변환합니다
func SessionStateFromString(state string) SessionState {
	states := map[string]SessionState{
		"created":      SessionStateCreated,
		"initializing": SessionStateInitializing,
		"ready":        SessionStateReady,
		"active":       SessionStateActive,
		"idle":         SessionStateIdle,
		"suspended":    SessionStateSuspended,
		"closing":      SessionStateClosing,
		"closed":       SessionStateClosed,
		"error":        SessionStateError,
	}
	
	if s, ok := states[state]; ok {
		return s
	}
	return SessionStateError
}