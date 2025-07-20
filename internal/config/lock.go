package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// FileLock 은 프로세스 간 파일 잠금을 관리합니다
type FileLock struct {
	path     string
	lockFile string
}

// NewFileLock 은 새로운 FileLock을 생성합니다
func NewFileLock(path string) *FileLock {
	return &FileLock{
		path:     path,
		lockFile: path + ".lock",
	}
}

// Lock 은 파일 잠금을 획득합니다
func (fl *FileLock) Lock() error {
	// 잠금 파일이 이미 존재하는지 확인
	if _, err := os.Stat(fl.lockFile); err == nil {
		// 잠금 파일이 오래된 경우 제거 (5분 이상)
		info, _ := os.Stat(fl.lockFile)
		if time.Since(info.ModTime()) > 5*time.Minute {
			os.Remove(fl.lockFile)
		} else {
			return fmt.Errorf("설정 파일이 다른 프로세스에 의해 잠겨 있습니다")
		}
	}

	// 잠금 파일 생성
	lockFile, err := os.OpenFile(fl.lockFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("잠금 파일 생성 실패: %w", err)
	}
	
	// PID 기록
	fmt.Fprintf(lockFile, "%d\n", os.Getpid())
	lockFile.Close()
	
	return nil
}

// Unlock 은 파일 잠금을 해제합니다
func (fl *FileLock) Unlock() error {
	return os.Remove(fl.lockFile)
}

// TryLock 은 잠금 획득을 시도하고, timeout 동안 재시도합니다
func (fl *FileLock) TryLock(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		if err := fl.Lock(); err == nil {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	
	return fmt.Errorf("잠금 획득 시간 초과")
}

// WithLock 은 잠금을 획득한 상태에서 함수를 실행합니다
func (fl *FileLock) WithLock(fn func() error) error {
	if err := fl.Lock(); err != nil {
		return err
	}
	defer fl.Unlock()
	
	return fn()
}

// ensureParentDir 은 파일의 부모 디렉토리가 존재하는지 확인하고 생성합니다
func ensureParentDir(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, 0700)
}