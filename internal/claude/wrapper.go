package claude

// Wrapper는 Claude CLI와의 최상위 통합 인터페이스입니다.
type Wrapper interface {
	// 세션 관리
	CreateSession(config *SessionConfig) (*Session, error)
	GetSession(sessionID string) (*Session, error)
	CloseSession(sessionID string) error
	ListSessions(filter SessionFilter) ([]*Session, error)
	
	// Claude 실행
	Execute(sessionID, prompt string) (interface{}, error)
}

// WrapperImpl은 Wrapper 인터페이스의 구현체입니다.
type WrapperImpl struct {
	sessionManager SessionManager
	processManager ProcessManager
}

// SessionManager는 내부 세션 매니저를 반환합니다.
func (w *WrapperImpl) SessionManager() SessionManager {
	return w.sessionManager
}

// NewWrapper는 새로운 Claude 래퍼를 생성합니다.
func NewWrapper(sessionManager SessionManager, processManager ProcessManager) Wrapper {
	return &WrapperImpl{
		sessionManager: sessionManager,
		processManager: processManager,
	}
}

// CreateSession은 새로운 Claude 세션을 생성합니다.
func (w *WrapperImpl) CreateSession(config *SessionConfig) (*Session, error) {
	if config == nil {
		return nil, &ClaudeError{
			Code:    "INVALID_REQUEST",
			Message: "Session configuration is required",
		}
	}

	// 기본값 설정
	if config.MaxTurns == 0 {
		config.MaxTurns = 10
	}

	ctx := context.Background()
	return w.sessionManager.CreateSession(ctx, *config)
}

// GetSession은 세션 정보를 조회합니다.
func (w *WrapperImpl) GetSession(sessionID string) (*Session, error) {
	if sessionID == "" {
		return nil, &ClaudeError{
			Code:    "INVALID_REQUEST",
			Message: "Session ID is required",
		}
	}

	return w.sessionManager.GetSession(sessionID)
}

// CloseSession은 세션을 종료합니다.
func (w *WrapperImpl) CloseSession(sessionID string) error {
	if sessionID == "" {
		return &ClaudeError{
			Code:    "INVALID_REQUEST",
			Message: "Session ID is required",
		}
	}

	return w.sessionManager.CloseSession(sessionID)
}

// Execute는 Claude 명령을 실행합니다.
func (w *WrapperImpl) Execute(sessionID, prompt string) (interface{}, error) {
	if sessionID == "" {
		return nil, &ClaudeError{
			Code:    "INVALID_REQUEST",
			Message: "Session ID is required",
		}
	}

	if prompt == "" {
		return nil, &ClaudeError{
			Code:    "INVALID_REQUEST",
			Message: "Prompt is required",
		}
	}

	// 세션 조회
	session, err := w.sessionManager.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	// 세션 상태 확인
	if session.State.Status != "idle" && session.State.Status != "ready" {
		return nil, &ClaudeError{
			Code:    "SESSION_BUSY",
			Message: "Session is currently busy",
		}
	}

	// 프로세스 실행
	result, err := w.processManager.Execute(sessionID, prompt)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// ListSessions는 세션 목록을 조회합니다.
func (w *WrapperImpl) ListSessions(filter SessionFilter) ([]*Session, error) {
	return w.sessionManager.ListSessions(filter)
}