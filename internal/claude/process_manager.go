package claude

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

// ProcessStatus 프로세스 상태
type ProcessStatus int

const (
	// StatusStopped 프로세스가 중지된 상태
	StatusStopped ProcessStatus = iota
	// StatusStarting 프로세스가 시작 중인 상태
	StatusStarting
	// StatusRunning 프로세스가 실행 중인 상태
	StatusRunning
	// StatusStopping 프로세스가 중지 중인 상태
	StatusStopping
	// StatusError 프로세스 오류 상태
	StatusError
	// StatusUnknown 알 수 없는 상태
	StatusUnknown
)

// String ProcessStatus를 문자열로 변환
func (s ProcessStatus) String() string {
	switch s {
	case StatusStopped:
		return "stopped"
	case StatusStarting:
		return "starting"
	case StatusRunning:
		return "running"
	case StatusStopping:
		return "stopping"
	case StatusError:
		return "error"
	default:
		return "unknown"
	}
}

// ProcessManager Claude CLI 프로세스 관리 인터페이스
type ProcessManager interface {
	// Start 프로세스를 시작합니다
	Start(ctx context.Context, config *ProcessConfig) error
	// Stop 프로세스를 정상적으로 중지합니다
	Stop(timeout time.Duration) error
	// Kill 프로세스를 강제로 종료합니다
	Kill() error
	// IsRunning 프로세스가 실행 중인지 확인합니다
	IsRunning() bool
	// GetStatus 현재 프로세스 상태를 반환합니다
	GetStatus() ProcessStatus
	// GetPID 프로세스 ID를 반환합니다
	GetPID() int
	// Wait 프로세스가 종료될 때까지 대기합니다
	Wait() error
	// HealthCheck 프로세스 상태를 확인합니다
	HealthCheck() error
	// RestartProcess 프로세스를 재시작합니다
	RestartProcess(identifier string) error
}

// ProcessConfig 프로세스 실행 설정
type ProcessConfig struct {
	// Command 실행할 명령어
	Command string
	// Args 명령어 인자
	Args []string
	// WorkingDir 작업 디렉토리
	WorkingDir string
	// Environment 환경 변수
	Environment map[string]string
	// Timeout 실행 타임아웃
	Timeout time.Duration
	// OAuthToken OAuth 인증 토큰
	OAuthToken string
	// APIKey Claude API 키 (OAuth 토큰이 없는 경우 fallback)
	APIKey string
	// ResourceLimits 리소스 제한 설정
	ResourceLimits *ResourceLimits
	// HealthCheckInterval 헬스체크 주기
	HealthCheckInterval time.Duration
}

// ResourceLimits 프로세스 리소스 제한 설정
type ResourceLimits struct {
	// MaxCPU 최대 CPU 사용량 (CPU 코어 수, 예: 1.5 = 1.5 코어)
	MaxCPU float64
	// MaxMemory 최대 메모리 사용량 (바이트)
	MaxMemory int64
	// MaxDiskIO 최대 디스크 I/O (바이트/초)
	MaxDiskIO int64
	// Timeout 최대 실행 시간
	Timeout time.Duration
}

// claudeProcessManager Claude CLI 프로세스 관리자 구현
type claudeProcessManager struct {
	cmd           *exec.Cmd
	config        *ProcessConfig
	status        ProcessStatus
	pid           int
	startTime     time.Time
	mutex         sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	done          chan error
	logger        *logrus.Logger
	tokenManager  TokenManager
	healthChecker HealthChecker
	healthCancel  context.CancelFunc
}

// NewProcessManager 새로운 프로세스 관리자를 생성합니다
func NewProcessManager(logger *logrus.Logger) ProcessManager {
	if logger == nil {
		logger = logrus.StandardLogger()
	}
	return &claudeProcessManager{
		status: StatusStopped,
		logger: logger,
		done:   make(chan error, 1),
	}
}

// Start 프로세스를 시작합니다
func (pm *claudeProcessManager) Start(ctx context.Context, config *ProcessConfig) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if pm.status != StatusStopped {
		return fmt.Errorf("프로세스가 이미 실행 중이거나 시작 중입니다 (현재 상태: %s)", pm.status)
	}

	if config == nil {
		return fmt.Errorf("프로세스 설정이 nil입니다")
	}

	if config.Command == "" {
		return fmt.Errorf("실행할 명령어가 지정되지 않았습니다")
	}

	pm.config = config
	pm.status = StatusStarting

	// 컨텍스트 설정
	pm.ctx, pm.cancel = context.WithCancel(ctx)

	// 명령어 준비
	pm.cmd = exec.CommandContext(pm.ctx, config.Command, config.Args...)
	
	// 작업 디렉토리 설정
	if config.WorkingDir != "" {
		pm.cmd.Dir = config.WorkingDir
	}

	// 환경 변수 설정
	env := os.Environ()
	
	// 기본 환경 변수 설정
	if len(config.Environment) > 0 {
		for key, value := range config.Environment {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}
	}
	
	// OAuth 토큰 또는 API 키 설정
	if config.OAuthToken != "" {
		env = append(env, fmt.Sprintf("CLAUDE_CODE_OAUTH_TOKEN=%s", config.OAuthToken))
		// 토큰 관리자 초기화
		pm.tokenManager = NewTokenManager(config.OAuthToken, config.APIKey, nil)
	} else if config.APIKey != "" {
		env = append(env, fmt.Sprintf("CLAUDE_API_KEY=%s", config.APIKey))
		// API 키만으로 토큰 관리자 초기화
		pm.tokenManager = NewTokenManager("", config.APIKey, nil)
	}
	
	pm.cmd.Env = env

	// 리소스 제한 설정
	if config.ResourceLimits != nil {
		if err := pm.applyResourceLimits(config.ResourceLimits); err != nil {
			pm.logger.WithError(err).Warn("리소스 제한 설정 실패")
			// 리소스 제한 실패는 치명적이지 않으므로 계속 진행
		}
	}

	// 프로세스 시작
	if err := pm.cmd.Start(); err != nil {
		pm.status = StatusError
		return fmt.Errorf("프로세스 시작 실패: %w", err)
	}

	pm.pid = pm.cmd.Process.Pid
	pm.startTime = time.Now()
	pm.status = StatusRunning

	pm.logger.WithFields(logrus.Fields{
		"pid":        pm.pid,
		"command":    config.Command,
		"args":       config.Args,
		"workingDir": config.WorkingDir,
	}).Info("프로세스가 성공적으로 시작되었습니다")

	// 헬스체커 초기화 및 시작
	if config.HealthCheckInterval > 0 {
		pm.healthChecker = NewHealthChecker(pm.logger)
		healthCtx, cancel := context.WithCancel(ctx)
		pm.healthCancel = cancel
		go pm.healthChecker.Start(healthCtx, pm, config.HealthCheckInterval)
	}

	// 비동기 프로세스 모니터링
	go pm.monitor()

	return nil
}

// monitor 프로세스를 모니터링합니다
func (pm *claudeProcessManager) monitor() {
	defer close(pm.done)

	err := pm.cmd.Wait()

	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if pm.status == StatusStopping {
		pm.status = StatusStopped
		pm.logger.WithFields(logrus.Fields{
			"pid":      pm.pid,
			"duration": time.Since(pm.startTime),
		}).Info("프로세스가 정상적으로 중지되었습니다")
	} else if err != nil {
		pm.status = StatusError
		pm.logger.WithFields(logrus.Fields{
			"pid":      pm.pid,
			"duration": time.Since(pm.startTime),
			"error":    err,
		}).Error("프로세스가 예기치 않게 종료되었습니다")
	} else {
		pm.status = StatusStopped
		pm.logger.WithFields(logrus.Fields{
			"pid":      pm.pid,
			"duration": time.Since(pm.startTime),
		}).Info("프로세스가 종료되었습니다")
	}

	pm.done <- err
}

// Stop 프로세스를 정상적으로 중지합니다
func (pm *claudeProcessManager) Stop(timeout time.Duration) error {
	pm.mutex.Lock()
	
	if pm.status != StatusRunning {
		pm.mutex.Unlock()
		return fmt.Errorf("프로세스가 실행 중이 아닙니다 (현재 상태: %s)", pm.status)
	}

	pm.status = StatusStopping
	pm.mutex.Unlock()

	// 헬스체커 중지
	if pm.healthChecker != nil && pm.healthCancel != nil {
		pm.healthCancel()
		pm.healthChecker.Stop()
	}

	pm.logger.WithFields(logrus.Fields{
		"pid":     pm.pid,
		"timeout": timeout,
	}).Info("프로세스 정상 종료를 시작합니다")

	// SIGTERM 전송
	if err := pm.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("인터럽트 시그널 전송 실패: %w", err)
	}

	// 타임아웃 기다리기
	select {
	case err := <-pm.done:
		if err != nil && err.Error() != "signal: terminated" {
			return fmt.Errorf("프로세스 종료 중 오류: %w", err)
		}
		return nil
	case <-time.After(timeout):
		pm.logger.WithField("pid", pm.pid).Warn("정상 종료 타임아웃, 강제 종료를 시도합니다")
		return pm.Kill()
	}
}

// Kill 프로세스를 강제로 종료합니다
func (pm *claudeProcessManager) Kill() error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if pm.status == StatusStopped {
		return nil
	}

	if pm.cmd != nil && pm.cmd.Process != nil {
		if err := pm.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("프로세스 강제 종료 실패: %w", err)
		}
	}

	if pm.cancel != nil {
		pm.cancel()
	}

	pm.status = StatusStopped
	pm.logger.WithField("pid", pm.pid).Info("프로세스가 강제 종료되었습니다")

	return nil
}

// IsRunning 프로세스가 실행 중인지 확인합니다
func (pm *claudeProcessManager) IsRunning() bool {
	return pm.GetStatus() == StatusRunning
}

// GetStatus 현재 프로세스 상태를 반환합니다
func (pm *claudeProcessManager) GetStatus() ProcessStatus {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	return pm.status
}

// GetPID 프로세스 ID를 반환합니다
func (pm *claudeProcessManager) GetPID() int {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	return pm.pid
}

// Wait 프로세스가 종료될 때까지 대기합니다
func (pm *claudeProcessManager) Wait() error {
	select {
	case err := <-pm.done:
		return err
	case <-pm.ctx.Done():
		return pm.ctx.Err()
	}
}

// HealthCheck 프로세스 상태를 확인합니다
func (pm *claudeProcessManager) HealthCheck() error {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	if pm.status != StatusRunning {
		return fmt.Errorf("프로세스가 실행 중이 아닙니다 (상태: %s)", pm.status)
	}

	if pm.cmd == nil || pm.cmd.Process == nil {
		return fmt.Errorf("프로세스 객체가 nil입니다")
	}

	// 프로세스가 실제로 실행 중인지 확인
	if err := pm.cmd.Process.Signal(syscall.Signal(0)); err != nil {
		return fmt.Errorf("프로세스 헬스체크 실패: %w", err)
	}

	return nil
}

// RestartProcess 프로세스를 재시작합니다
func (pm *claudeProcessManager) RestartProcess(identifier string) error {
	pm.logger.WithField("identifier", identifier).Info("프로세스 재시작을 시작합니다")
	
	// 현재 설정 저장
	pm.mutex.RLock()
	config := pm.config
	ctx := pm.ctx
	pm.mutex.RUnlock()
	
	if config == nil {
		return fmt.Errorf("프로세스 설정이 없습니다")
	}
	
	// 프로세스 중지
	if pm.IsRunning() {
		if err := pm.Stop(30 * time.Second); err != nil {
			pm.logger.WithError(err).Warn("프로세스 정상 중지 실패, 강제 종료 시도")
			if killErr := pm.Kill(); killErr != nil {
				return fmt.Errorf("프로세스 중지 실패: %w", killErr)
			}
		}
	}
	
	// 잠시 대기
	time.Sleep(2 * time.Second)
	
	// 프로세스 재시작
	if err := pm.Start(ctx, config); err != nil {
		return fmt.Errorf("프로세스 재시작 실패: %w", err)
	}
	
	pm.logger.WithField("identifier", identifier).Info("프로세스가 성공적으로 재시작되었습니다")
	return nil
}

// applyResourceLimits 프로세스에 리소스 제한을 적용합니다
func (pm *claudeProcessManager) applyResourceLimits(limits *ResourceLimits) error {
	if limits == nil {
		return nil
	}

	// Linux/Unix 시스템에서 rlimit 설정
	if runtime.GOOS != "windows" {
		if pm.cmd.SysProcAttr == nil {
			pm.cmd.SysProcAttr = &syscall.SysProcAttr{}
		}

		// TODO: 실제 리소스 제한 구현
		// - cgroup 또는 rlimit 사용
		// - CPU 제한: cgroup cpu.cfs_quota_us / cpu.cfs_period_us
		// - 메모리 제한: cgroup memory.limit_in_bytes 또는 rlimit RLIMIT_AS
		// - 디스크 I/O 제한: cgroup blkio 컨트롤러
		
		pm.logger.WithFields(logrus.Fields{
			"maxCPU":    limits.MaxCPU,
			"maxMemory": limits.MaxMemory,
			"maxDiskIO": limits.MaxDiskIO,
			"timeout":   limits.Timeout,
		}).Debug("리소스 제한 설정 (구현 예정)")
	}

	// 타임아웃 설정
	if limits.Timeout > 0 && pm.config.Timeout == 0 {
		pm.config.Timeout = limits.Timeout
	}

	return nil
}