package mount

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

// Validator 프로젝트 경로 및 마운트 검증을 수행합니다.
type Validator struct{}

// NewValidator 새로운 검증기를 생성합니다.
func NewValidator() *Validator {
	return &Validator{}
}

// ValidateProjectPath 프로젝트 경로의 유효성을 검사합니다.
func (v *Validator) ValidateProjectPath(path string) error {
	if path == "" {
		return fmt.Errorf("project path is required")
	}
	
	// 절대 경로로 변환
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve absolute path: %w", err)
	}
	
	// 디렉토리 존재 확인
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("project directory does not exist: %s", absPath)
		}
		return fmt.Errorf("stat project directory: %w", err)
	}
	
	if !info.IsDir() {
		return fmt.Errorf("project path is not a directory: %s", absPath)
	}
	
	// 접근 권한 확인
	if err := v.checkAccess(absPath); err != nil {
		return fmt.Errorf("access check failed: %w", err)
	}
	
	// 보안 검사
	if err := v.checkSecurity(absPath); err != nil {
		return fmt.Errorf("security check failed: %w", err)
	}
	
	return nil
}

// checkAccess 디렉토리 접근 권한을 확인합니다.
func (v *Validator) checkAccess(path string) error {
	if runtime.GOOS == "windows" {
		// Windows에서는 간단한 접근성 테스트
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("no read permission: %s", path)
		}
		file.Close()
		
		// 쓰기 권한 테스트 (임시 파일 생성 시도)
		tempFile := filepath.Join(path, ".aicli_write_test")
		file, err = os.Create(tempFile)
		if err == nil {
			file.Close()
			os.Remove(tempFile) // 정리
		}
		// 쓰기 권한이 없어도 경고만, 에러는 아님
		
		return nil
	}
	
	// 읽기 권한 확인 (크로스 플랫폼 호환)
	if file, err := os.Open(path); err != nil {
		return fmt.Errorf("no read permission: %s", path)
	} else {
		file.Close()
	}
	
	// 쓰기 권한 확인 (선택적)
	if file, err := os.OpenFile(path, os.O_WRONLY, 0); err != nil {
		// 경고만, 읽기 전용으로 마운트될 가능성이 있음
		_ = err // 무시
	} else {
		file.Close()
	}
	
	return nil
}

// checkSecurity 보안상 위험한 경로를 차단합니다.
func (v *Validator) checkSecurity(path string) error {
	absPath, _ := filepath.Abs(path)
	
	// 플랫폼별 시스템 디렉토리 확인
	var systemPaths []string
	
	if runtime.GOOS == "windows" {
		systemPaths = []string{
			"C:\\Windows",
			"C:\\Program Files",
			"C:\\Program Files (x86)",
			"C:\\System Volume Information",
		}
	} else {
		systemPaths = []string{
			"/",
			"/etc",
			"/usr",
			"/bin",
			"/sbin",
			"/boot",
			"/sys",
			"/proc",
			"/dev",
			"/root",
		}
	}
	
	// 시스템 루트 디렉토리 마운트 방지
	for _, sysPath := range systemPaths {
		if strings.HasPrefix(strings.ToLower(absPath), strings.ToLower(sysPath)) {
			return fmt.Errorf("cannot mount system directory: %s", absPath)
		}
	}
	
	// Docker 소켓 마운트 방지 (Unix 계열만)
	if runtime.GOOS != "windows" {
		dockerSocket := "/var/run/docker.sock"
		if strings.Contains(absPath, dockerSocket) {
			return fmt.Errorf("cannot mount docker socket path: %s", absPath)
		}
	}
	
	// 심볼릭 링크 검사
	if err := v.checkSymlinks(absPath); err != nil {
		return err
	}
	
	return nil
}

// checkSymlinks 심볼릭 링크의 실제 경로를 검사합니다.
func (v *Validator) checkSymlinks(path string) error {
	// 심볼릭 링크인 경우 실제 경로 확인
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return fmt.Errorf("eval symlinks: %w", err)
	}
	
	// 실제 경로가 예상 경로와 다른 경우 보안 검사 수행
	if realPath != path {
		// 심볼릭 링크를 허용하지만 보안 검사는 수행
		return v.checkSecurity(realPath)
	}
	
	return nil
}

// CanOptimizeMount 파일 시스템에 따른 마운트 최적화 가능성을 검사합니다.
func (v *Validator) CanOptimizeMount(path string) (bool, SyncMode) {
	if runtime.GOOS == "windows" {
		// Windows에서는 기본적으로 네이티브 모드 사용
		return false, SyncModeNative
	}
	
	// 파일 시스템 타입에 따른 최적화
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return false, SyncModeNative
	}
	
	// 마운트 포인트 분석 등을 통한 성능 최적화 방식 결정
	switch stat.Type {
	case 0x58465342: // XFS
		return true, SyncModeOptimized
	case 0xEF53:     // EXT4
		return true, SyncModeCached
	case 0x6969:     // NFS
		return true, SyncModeDelegated
	default:
		return false, SyncModeNative
	}
}

// ValidateMountPath 마운트 대상 경로를 검증합니다.
func (v *Validator) ValidateMountPath(sourcePath, targetPath string) error {
	// 소스 경로 검증
	if err := v.ValidateProjectPath(sourcePath); err != nil {
		return fmt.Errorf("invalid source path: %w", err)
	}
	
	// 타겟 경로 검증 (컨테이너 내부 경로)
	if targetPath == "" {
		return fmt.Errorf("target path is required")
	}
	
	if !filepath.IsAbs(targetPath) {
		return fmt.Errorf("target path must be absolute: %s", targetPath)
	}
	
	// 컨테이너 내부의 민감한 경로 마운트 방지
	sensitiveTargets := []string{
		"/etc",
		"/usr",
		"/bin",
		"/sbin",
		"/boot",
		"/sys",
		"/proc",
		"/dev",
		"/root",
		"/var/run",
	}
	
	for _, sensitive := range sensitiveTargets {
		if strings.HasPrefix(targetPath, sensitive) {
			return fmt.Errorf("cannot mount to sensitive container path: %s", targetPath)
		}
	}
	
	return nil
}

// GetDiskUsage 디렉토리의 디스크 사용량을 조회합니다.
func (v *Validator) GetDiskUsage(path string) (*DiskUsage, error) {
	var stat syscall.Statfs_t
	
	if runtime.GOOS == "windows" {
		// Windows에서는 간단한 크기 계산
		size, err := v.calculateDirSize(path)
		if err != nil {
			return nil, fmt.Errorf("calculate directory size: %w", err)
		}
		
		return &DiskUsage{
			Total:     size,
			Used:      size,
			Available: 0, // Windows에서는 정확한 계산이 복잡
		}, nil
	}
	
	if err := syscall.Statfs(path, &stat); err != nil {
		return nil, fmt.Errorf("get filesystem stats: %w", err)
	}
	
	usage := &DiskUsage{
		Total:     int64(stat.Blocks) * int64(stat.Bsize),
		Available: int64(stat.Bavail) * int64(stat.Bsize),
	}
	usage.Used = usage.Total - usage.Available
	
	return usage, nil
}

// calculateDirSize 디렉토리 크기를 재귀적으로 계산합니다.
func (v *Validator) calculateDirSize(path string) (int64, error) {
	var size int64
	
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 개별 파일 에러는 무시
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	
	return size, err
}

// DiskUsage 디스크 사용량 정보입니다.
type DiskUsage struct {
	Total     int64 `json:"total"`
	Used      int64 `json:"used"`
	Available int64 `json:"available"`
}

// SyncMode 동기화 모드를 정의합니다.
type SyncMode string

const (
	SyncModeNative     SyncMode = "native"     // 기본 Docker 마운트
	SyncModeOptimized  SyncMode = "optimized"  // 성능 최적화 모드
	SyncModeCached     SyncMode = "cached"     // 캐시 모드
	SyncModeDelegated  SyncMode = "delegated"  // 위임 모드
)