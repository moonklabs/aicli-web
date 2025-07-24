package claude

import (
	"time"
)

// Process는 테스트용 프로세스 구조체입니다.
type Process struct {
	ID        string
	Config    ProcessConfig
	State     ProcessState
	StartTime time.Time
	PID       int
}

// ProcessState는 프로세스 상태를 나타냅니다.
type ProcessState string

const (
	ProcessStateRunning    ProcessState = "running"
	ProcessStateTerminated ProcessState = "terminated"
)

// ProcessHealth는 프로세스 헬스 정보입니다.
type ProcessHealth struct {
	ProcessID    string
	Healthy      bool
	LastCheck    time.Time
	ResponseTime time.Duration
}

// ErrProcessNotFound는 프로세스를 찾을 수 없을 때 반환되는 오류입니다.
var ErrProcessNotFound = error(nil)