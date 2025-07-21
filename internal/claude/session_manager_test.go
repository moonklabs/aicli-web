package claude

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockProcessManager는 테스트용 프로세스 매니저입니다
type MockProcessManager struct {
	processes map[string]*Process
}

func NewMockProcessManager() *MockProcessManager {
	return &MockProcessManager{
		processes: make(map[string]*Process),
	}
}

func (m *MockProcessManager) CreateProcess(ctx context.Context, config ProcessConfig) (*Process, error) {
	process := &Process{
		ID:        "test-process-" + time.Now().Format("150405"),
		Config:    config,
		State:     ProcessStateRunning,
		StartTime: time.Now(),
		PID:       12345,
	}
	m.processes[process.ID] = process
	return process, nil
}

func (m *MockProcessManager) GetProcess(id string) (*Process, error) {
	process, exists := m.processes[id]
	if !exists {
		return nil, ErrProcessNotFound
	}
	return process, nil
}

func (m *MockProcessManager) TerminateProcess(id string) error {
	process, exists := m.processes[id]
	if !exists {
		return ErrProcessNotFound
	}
	process.State = ProcessStateTerminated
	return nil
}

func (m *MockProcessManager) ListProcesses() ([]*Process, error) {
	processes := make([]*Process, 0, len(m.processes))
	for _, p := range m.processes {
		processes = append(processes, p)
	}
	return processes, nil
}

func (m *MockProcessManager) GetProcessHealth(id string) (*ProcessHealth, error) {
	return &ProcessHealth{
		ProcessID:    id,
		Healthy:      true,
		LastCheck:    time.Now(),
		ResponseTime: 10 * time.Millisecond,
	}, nil
}

func TestSessionConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  SessionConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: SessionConfig{
				WorkingDir:   "/tmp",
				MaxTurns:     10,
				Temperature:  0.7,
				ToolTimeout:  30 * time.Second,
				MaxCPU:       0.5,
				MaxDuration:  2 * time.Hour,
				AllowedTools: []string{"code_interpreter", "file_search"},
			},
			wantErr: false,
		},
		{
			name: "missing working directory",
			config: SessionConfig{
				MaxTurns:    10,
				Temperature: 0.7,
			},
			wantErr: true,
			errMsg:  "working directory is required",
		},
		{
			name: "invalid max turns",
			config: SessionConfig{
				WorkingDir:  "/tmp",
				MaxTurns:    0,
				Temperature: 0.7,
			},
			wantErr: true,
			errMsg:  "max_turns must be between 1 and 1000",
		},
		{
			name: "invalid temperature",
			config: SessionConfig{
				WorkingDir:  "/tmp",
				MaxTurns:    10,
				Temperature: 3.0,
			},
			wantErr: true,
			errMsg:  "temperature must be between 0 and 2",
		},
		{
			name: "invalid tool",
			config: SessionConfig{
				WorkingDir:   "/tmp",
				MaxTurns:     10,
				Temperature:  0.7,
				AllowedTools: []string{"invalid_tool"},
			},
			wantErr: true,
			errMsg:  "invalid tool: invalid_tool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSessionManager_CreateSession(t *testing.T) {
	ctx := context.Background()
	pm := NewMockProcessManager()
	sm := NewSessionManager(pm, nil)

	config := SessionConfig{
		WorkingDir:  "/tmp",
		MaxTurns:    10,
		Temperature: 0.7,
		Environment: map[string]string{
			"TEST_ENV": "test_value",
		},
	}

	// 세션 생성
	session, err := sm.CreateSession(ctx, config)
	require.NoError(t, err)
	assert.NotEmpty(t, session.ID)
	assert.Equal(t, SessionStateReady, session.State)
	assert.NotNil(t, session.Process)
	assert.Equal(t, config.WorkingDir, session.Config.WorkingDir)

	// 세션이 저장되었는지 확인
	retrieved, err := sm.GetSession(session.ID)
	require.NoError(t, err)
	assert.Equal(t, session.ID, retrieved.ID)
}

func TestSessionManager_UpdateSession(t *testing.T) {
	ctx := context.Background()
	pm := NewMockProcessManager()
	sm := NewSessionManager(pm, nil)

	// 세션 생성
	config := SessionConfig{
		WorkingDir:  "/tmp",
		MaxTurns:    10,
		Temperature: 0.7,
	}
	session, err := sm.CreateSession(ctx, config)
	require.NoError(t, err)

	// 상태 업데이트
	activeState := SessionStateActive
	err = sm.UpdateSession(session.ID, SessionUpdate{
		State: &activeState,
	})
	require.NoError(t, err)

	// 업데이트 확인
	updated, err := sm.GetSession(session.ID)
	require.NoError(t, err)
	assert.Equal(t, SessionStateActive, updated.State)

	// 메타데이터 업데이트
	err = sm.UpdateSession(session.ID, SessionUpdate{
		Metadata: map[string]interface{}{
			"test_key": "test_value",
		},
	})
	require.NoError(t, err)

	// 메타데이터 확인
	updated, err = sm.GetSession(session.ID)
	require.NoError(t, err)
	assert.Equal(t, "test_value", updated.Metadata["test_key"])
}

func TestSessionManager_CloseSession(t *testing.T) {
	ctx := context.Background()
	pm := NewMockProcessManager()
	sm := NewSessionManager(pm, nil)

	// 세션 생성
	config := SessionConfig{
		WorkingDir:  "/tmp",
		MaxTurns:    10,
		Temperature: 0.7,
	}
	session, err := sm.CreateSession(ctx, config)
	require.NoError(t, err)

	// 세션 종료
	err = sm.CloseSession(session.ID)
	require.NoError(t, err)

	// 프로세스가 종료되었는지 확인
	process, err := pm.GetProcess(session.Process.ID)
	require.NoError(t, err)
	assert.Equal(t, ProcessStateTerminated, process.State)
}

func TestSessionManager_ListSessions(t *testing.T) {
	ctx := context.Background()
	pm := NewMockProcessManager()
	sm := NewSessionManager(pm, nil)

	// 여러 세션 생성
	workspaceID := "test-workspace"
	for i := 0; i < 3; i++ {
		config := SessionConfig{
			WorkingDir:  "/tmp",
			MaxTurns:    10,
			Temperature: 0.7,
		}
		session, err := sm.CreateSession(ctx, config)
		require.NoError(t, err)
		session.WorkspaceID = workspaceID
		
		// 하나는 Active 상태로 변경
		if i == 0 {
			activeState := SessionStateActive
			sm.UpdateSession(session.ID, SessionUpdate{State: &activeState})
		}
	}

	// 전체 세션 조회
	sessions, err := sm.ListSessions(SessionFilter{})
	require.NoError(t, err)
	assert.Len(t, sessions, 3)

	// WorkspaceID로 필터링
	sessions, err = sm.ListSessions(SessionFilter{
		WorkspaceID: workspaceID,
	})
	require.NoError(t, err)
	assert.Len(t, sessions, 3)

	// Active 세션만 필터링
	active := true
	sessions, err = sm.ListSessions(SessionFilter{
		Active: &active,
	})
	require.NoError(t, err)
	assert.Len(t, sessions, 1)
}

func TestSessionStateMachine(t *testing.T) {
	sm := NewSessionStateMachine()

	// 유효한 전이 테스트
	tests := []struct {
		from SessionState
		to   SessionState
		valid bool
	}{
		{SessionStateCreated, SessionStateInitializing, true},
		{SessionStateInitializing, SessionStateReady, true},
		{SessionStateReady, SessionStateActive, true},
		{SessionStateActive, SessionStateIdle, true},
		{SessionStateIdle, SessionStateActive, true},
		{SessionStateSuspended, SessionStateReady, true},
		{SessionStateClosing, SessionStateClosed, true},
		
		// 무효한 전이
		{SessionStateCreated, SessionStateActive, false},
		{SessionStateClosed, SessionStateActive, false},
		{SessionStateReady, SessionStateClosed, false},
	}

	for _, tt := range tests {
		err := sm.CanTransition(tt.from, tt.to)
		if tt.valid {
			assert.NoError(t, err, "Expected valid transition from %s to %s", tt.from, tt.to)
		} else {
			assert.Error(t, err, "Expected invalid transition from %s to %s", tt.from, tt.to)
		}
	}
}

func TestSessionPool(t *testing.T) {
	ctx := context.Background()
	pm := NewMockProcessManager()
	sm := NewSessionManager(pm, nil)
	
	poolConfig := SessionPoolConfig{
		MaxSessions:     3,
		MaxIdleTime:     1 * time.Minute,
		MaxLifetime:     10 * time.Minute,
		CleanupInterval: 30 * time.Second,
	}
	pool := NewSessionPool(sm, poolConfig)
	defer pool.Shutdown()

	sessionConfig := SessionConfig{
		WorkingDir:  "/tmp",
		MaxTurns:    10,
		Temperature: 0.7,
	}

	// 세션 획득
	session1, err := pool.AcquireSession(ctx, sessionConfig)
	require.NoError(t, err)
	assert.NotNil(t, session1)
	assert.True(t, session1.inUse)

	// 통계 확인
	stats := pool.GetPoolStats()
	assert.Equal(t, 1, stats.TotalSessions)
	assert.Equal(t, 1, stats.ActiveSessions)
	assert.Equal(t, 0, stats.IdleSessions)

	// 세션 반환
	err = pool.ReleaseSession(session1.ID)
	require.NoError(t, err)
	assert.False(t, session1.inUse)

	// 통계 확인
	stats = pool.GetPoolStats()
	assert.Equal(t, 1, stats.TotalSessions)
	assert.Equal(t, 0, stats.ActiveSessions)
	assert.Equal(t, 1, stats.IdleSessions)

	// 동일한 설정으로 세션 재사용
	session2, err := pool.AcquireSession(ctx, sessionConfig)
	require.NoError(t, err)
	assert.Equal(t, session1.ID, session2.ID) // 재사용됨

	// 풀이 가득 찬 경우 테스트
	for i := 0; i < 2; i++ {
		_, err := pool.AcquireSession(ctx, sessionConfig)
		require.NoError(t, err)
	}

	stats = pool.GetPoolStats()
	assert.Equal(t, 3, stats.TotalSessions)

	// 더 이상 세션을 생성할 수 없음
	_, err = pool.AcquireSession(ctx, sessionConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session pool is full")
}

func TestSessionEventBus(t *testing.T) {
	bus := NewSessionEventBus(100)
	defer bus.Shutdown()

	// 이벤트 레코더 생성
	recorder := NewSessionEventRecorder()
	sessionID := "test-session-123"
	
	// 세션별 구독
	bus.Subscribe(sessionID, recorder)

	// 이벤트 발행
	event := SessionEvent{
		SessionID: sessionID,
		Type:      SessionEventCreated,
		Data:      map[string]string{"test": "data"},
	}
	bus.Publish(event)

	// 이벤트 수신 대기
	time.Sleep(100 * time.Millisecond)

	// 이벤트 확인
	events := recorder.GetEvents()
	require.Len(t, events, 1)
	assert.Equal(t, sessionID, events[0].SessionID)
	assert.Equal(t, SessionEventCreated, events[0].Type)

	// 타입별 구독 테스트
	typeRecorder := NewSessionEventRecorder()
	bus.SubscribeToType(SessionEventStateChanged, func(e SessionEvent) {
		typeRecorder.OnSessionEvent(e)
	})

	// 상태 변경 이벤트 발행
	stateEvent := SessionEvent{
		SessionID: "another-session",
		Type:      SessionEventStateChanged,
		Data:      StateChangeData{OldState: SessionStateReady, NewState: SessionStateActive},
	}
	bus.Publish(stateEvent)

	// 이벤트 수신 대기
	time.Sleep(100 * time.Millisecond)

	// 타입별 이벤트 확인
	typeEvents := typeRecorder.GetEventsByType(SessionEventStateChanged)
	require.Len(t, typeEvents, 1)
	assert.Equal(t, SessionEventStateChanged, typeEvents[0].Type)
}