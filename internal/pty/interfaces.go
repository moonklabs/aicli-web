package pty

import (
	"context"
	"os"
	"time"
)

// PTYSessionInterface PTY 세션 관리 인터페이스
type PTYSessionInterface interface {
	// 세션 생성 및 관리
	CreateSession(ctx context.Context, containerID string, config *PTYConfig) (*PTYSession, error)
	GetSession(sessionID string) (*PTYSession, error)
	CloseSession(sessionID string) error
	ListSessions() map[string]*PTYSession
	
	// 세션 정리
	CleanupIdleSessions(timeout time.Duration) int
	RequestCleanup(sessionID string)
	
	// 세션 활동 관리
	UpdateSessionActivity(sessionID string) error
	
	// 통계 및 모니터링
	GetStats() map[string]interface{}
	
	// 생명주기 관리
	Shutdown() error
}

// ContainerConnector Docker 컨테이너 연결 인터페이스
type ContainerConnector interface {
	// PTY 연결 관리
	AttachPTY(ctx context.Context, containerID string, config *PTYConfig) (*os.File, error)
	DetachPTY(sessionID string) error
	
	// PTY 크기 조정
	ResizePTY(sessionID string, rows, cols int) error
	
	// 컨테이너 상태 확인
	IsContainerRunning(containerID string) (bool, error)
	GetContainerInfo(containerID string) (map[string]interface{}, error)
}

// PTYReader PTY 읽기 인터페이스
type PTYReader interface {
	// 데이터 읽기
	Read(sessionID string, buffer []byte) (int, error)
	
	// 스트림 읽기
	ReadStream(sessionID string) (<-chan []byte, error)
	
	// 읽기 중단
	StopReading(sessionID string) error
}

// PTYWriter PTY 쓰기 인터페이스
type PTYWriter interface {
	// 데이터 쓰기
	Write(sessionID string, data []byte) (int, error)
	
	// 명령 실행
	ExecuteCommand(sessionID string, command string) error
	
	// 특수 키 전송
	SendSignal(sessionID string, signal os.Signal) error
}

// SessionMonitor 세션 모니터링 인터페이스
type SessionMonitor interface {
	// 세션 상태 모니터링
	GetSessionStatus(sessionID string) (SessionStatus, error)
	GetSessionMetrics(sessionID string) (map[string]interface{}, error)
	
	// 세션 이벤트 구독
	SubscribeToEvents(sessionID string) (<-chan SessionEvent, error)
	UnsubscribeFromEvents(sessionID string) error
	
	// 알림 설정
	SetIdleAlert(sessionID string, duration time.Duration) error
	SetResourceAlert(sessionID string, threshold ResourceThreshold) error
}

// SessionEvent 세션 이벤트
type SessionEvent struct {
	SessionID string                 // 세션 ID
	Type      SessionEventType       // 이벤트 타입
	Timestamp time.Time              // 발생 시간
	Data      map[string]interface{} // 이벤트 데이터
}

// SessionEventType 세션 이벤트 타입
type SessionEventType int

const (
	EventSessionCreated SessionEventType = iota
	EventSessionActive
	EventSessionIdle
	EventSessionTerminated
	EventSessionError
	EventDataReceived
	EventDataSent
	EventResized
)

// String SessionEventType 문자열 변환
func (e SessionEventType) String() string {
	switch e {
	case EventSessionCreated:
		return "session_created"
	case EventSessionActive:
		return "session_active"
	case EventSessionIdle:
		return "session_idle"
	case EventSessionTerminated:
		return "session_terminated"
	case EventSessionError:
		return "session_error"
	case EventDataReceived:
		return "data_received"
	case EventDataSent:
		return "data_sent"
	case EventResized:
		return "resized"
	default:
		return "unknown"
	}
}

// ResourceThreshold 리소스 임계값
type ResourceThreshold struct {
	MaxMemory   int64   // 최대 메모리 사용량 (bytes)
	MaxCPU      float64 // 최대 CPU 사용률 (0-100)
	MaxDiskIO   int64   // 최대 디스크 I/O (bytes/sec)
	MaxNetworkIO int64  // 최대 네트워크 I/O (bytes/sec)
}

// SessionSerializer 세션 직렬화 인터페이스
type SessionSerializer interface {
	// 세션 직렬화
	SerializeSession(session *PTYSession) ([]byte, error)
	DeserializeSession(data []byte) (*PTYSession, error)
	
	// 세션 스냅샷
	CreateSnapshot(sessionID string) ([]byte, error)
	RestoreSnapshot(sessionID string, snapshot []byte) error
	
	// 세션 마이그레이션
	ExportSession(sessionID string) ([]byte, error)
	ImportSession(data []byte) (*PTYSession, error)
}

// SessionPersistence 세션 영속성 인터페이스
type SessionPersistence interface {
	// 세션 저장
	SaveSession(session *PTYSession) error
	LoadSession(sessionID string) (*PTYSession, error)
	DeleteSession(sessionID string) error
	
	// 세션 목록
	ListPersistedSessions() ([]string, error)
	
	// 세션 메타데이터
	SaveMetadata(sessionID string, metadata map[string]interface{}) error
	LoadMetadata(sessionID string) (map[string]interface{}, error)
}

// SessionReplicator 세션 복제 인터페이스 (클러스터링용)
type SessionReplicator interface {
	// 세션 복제
	ReplicateSession(session *PTYSession, targetNode string) error
	SyncSession(sessionID string) error
	
	// 세션 마이그레이션
	MigrateSession(sessionID string, targetNode string) error
	
	// 노드 간 동기화
	BroadcastSessionUpdate(sessionID string, update SessionUpdate) error
	SubscribeToUpdates() (<-chan SessionUpdate, error)
}

// SessionUpdate 세션 업데이트
type SessionUpdate struct {
	SessionID   string                 // 세션 ID
	UpdateType  UpdateType             // 업데이트 타입
	SourceNode  string                 // 소스 노드
	Timestamp   time.Time              // 타임스탬프
	Data        map[string]interface{} // 업데이트 데이터
}

// UpdateType 업데이트 타입
type UpdateType int

const (
	UpdateCreated UpdateType = iota
	UpdateModified
	UpdateDeleted
	UpdateActivity
	UpdateMetadata
)