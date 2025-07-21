package validation

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/go-playground/validator/v10"
)

// validateDirectory 디렉토리 존재 여부 검증
func validateDirectory(fl validator.FieldLevel) bool {
	path := fl.Field().String()
	if path == "" {
		return false
	}

	// 절대 경로로 변환
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	// 디렉토리 존재 여부 확인
	info, err := os.Stat(absPath)
	if err != nil {
		return false
	}

	return info.IsDir()
}

// validateSafePath 안전한 경로 검증
func validateSafePath(fl validator.FieldLevel) bool {
	path := fl.Field().String()
	if path == "" {
		return false
	}

	// 위험한 패턴 확인
	dangerousPatterns := []string{
		"..",
		"~",
		"$",
		"|",
		";",
		"&",
		">",
		"<",
		"`",
		"\\",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(path, pattern) {
			return false
		}
	}

	// 절대 경로로 변환
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	// Clean 경로와 비교
	cleanPath := filepath.Clean(absPath)
	return absPath == cleanPath
}

// IsValidProjectPath 프로젝트 경로 유효성 검사
func IsValidProjectPath(path string) error {
	// 빈 경로 확인
	if path == "" {
		return os.ErrInvalid
	}

	// 절대 경로로 변환
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	// 디렉토리 존재 여부 확인
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			// 디렉토리가 존재하지 않으면 생성 시도
			if err := os.MkdirAll(absPath, 0755); err != nil {
				return err
			}
			return nil
		}
		return err
	}

	// 디렉토리인지 확인
	if !info.IsDir() {
		return os.ErrInvalid
	}

	// 쓰기 권한 확인
	testFile := filepath.Join(absPath, ".aicli_test")
	file, err := os.Create(testFile)
	if err != nil {
		return err
	}
	file.Close()
	os.Remove(testFile)

	return nil
}

// validateDirectoryWritable 디렉토리 쓰기 가능 여부 검증
func validateDirectoryWritable(fl validator.FieldLevel) bool {
	path := fl.Field().String()
	if path == "" {
		return false
	}

	return IsValidProjectPath(path) == nil
}

// validateDirectoryOrCreatable 디렉토리 존재 또는 생성 가능 여부 검증
func validateDirectoryOrCreatable(fl validator.FieldLevel) bool {
	path := fl.Field().String()
	if path == "" {
		return false
	}

	// 절대 경로로 변환
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	// 이미 존재하는 디렉토리인지 확인
	if info, err := os.Stat(absPath); err == nil {
		return info.IsDir()
	}

	// 상위 디렉토리가 존재하고 쓰기 가능한지 확인
	parentDir := filepath.Dir(absPath)
	parentInfo, err := os.Stat(parentDir)
	if err != nil {
		return false
	}

	if !parentInfo.IsDir() {
		return false
	}

	// 상위 디렉토리에 쓰기 권한이 있는지 테스트
	testDir := filepath.Join(parentDir, ".aicli_test_dir")
	if err := os.Mkdir(testDir, 0755); err != nil {
		return false
	}
	os.Remove(testDir)

	return true
}

// PathValidationOptions 경로 검증 옵션
type PathValidationOptions struct {
	MustExist     bool   // 경로가 반드시 존재해야 함
	MustBeDir     bool   // 디렉토리여야 함
	MustBeFile    bool   // 파일이어야 함
	Writable      bool   // 쓰기 가능해야 함
	Readable      bool   // 읽기 가능해야 함
	AllowRelative bool   // 상대 경로 허용
	MaxDepth      int    // 최대 디렉토리 깊이
	AllowedExts   []string // 허용되는 파일 확장자
}

// ValidatePathWithOptions 옵션을 사용한 경로 검증
func ValidatePathWithOptions(path string, options PathValidationOptions) error {
	if path == "" {
		return NewBusinessValidationError(
			ErrCodeInvalidConfiguration,
			"경로가 비어있습니다",
			"path",
		)
	}

	// 상대 경로 검사
	if !options.AllowRelative && !filepath.IsAbs(path) {
		return NewBusinessValidationError(
			ErrCodeInvalidConfiguration,
			"절대 경로만 허용됩니다",
			"path",
		)
	}

	// 절대 경로로 변환
	absPath, err := filepath.Abs(path)
	if err != nil {
		return NewBusinessValidationError(
			ErrCodeInvalidConfiguration,
			"경로를 절대 경로로 변환할 수 없습니다",
			"path",
		)
	}

	// 안전성 검사
	if err := validatePathSafety(absPath); err != nil {
		return err
	}

	// 깊이 검사
	if options.MaxDepth > 0 {
		depth := len(strings.Split(filepath.Clean(absPath), string(filepath.Separator)))
		if depth > options.MaxDepth {
			return NewBusinessValidationError(
				ErrCodeInvalidConfiguration,
				"경로 깊이가 최대값을 초과했습니다",
				"path",
			)
		}
	}

	// 존재 여부 검사
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			if options.MustExist {
				return NewBusinessValidationError(
					ErrCodeResourceNotFound,
					"지정된 경로가 존재하지 않습니다",
					"path",
				)
			}
			return nil // 존재하지 않아도 됨
		}
		return NewBusinessValidationError(
			ErrCodePathNotAccessible,
			"경로에 접근할 수 없습니다",
			"path",
		)
	}

	// 타입 검사
	if options.MustBeDir && !info.IsDir() {
		return NewBusinessValidationError(
			ErrCodeInvalidConfiguration,
			"디렉토리여야 합니다",
			"path",
		)
	}

	if options.MustBeFile && info.IsDir() {
		return NewBusinessValidationError(
			ErrCodeInvalidConfiguration,
			"파일이어야 합니다",
			"path",
		)
	}

	// 확장자 검사
	if len(options.AllowedExts) > 0 && !info.IsDir() {
		ext := strings.ToLower(filepath.Ext(absPath))
		allowed := false
		for _, allowedExt := range options.AllowedExts {
			if ext == strings.ToLower(allowedExt) {
				allowed = true
				break
			}
		}
		if !allowed {
			return NewBusinessValidationError(
				ErrCodeInvalidConfiguration,
				"허용되지 않는 파일 확장자입니다",
				"path",
			)
		}
	}

	// 권한 검사
	if options.Readable {
		if err := checkReadablePermission(absPath); err != nil {
			return err
		}
	}

	if options.Writable {
		if err := checkWritablePermission(absPath); err != nil {
			return err
		}
	}

	return nil
}

// validatePathSafety 경로 안전성 검증
func validatePathSafety(path string) error {
	// 위험한 패턴들 검사
	dangerousPatterns := []string{
		"..",
		"~",
		"$",
		"|",
		";",
		"&",
		">",
		"<",
		"`",
		"\\",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(path, pattern) {
			return NewBusinessValidationError(
				ErrCodePathNotAccessible,
				"경로에 위험한 문자가 포함되어 있습니다",
				"path",
			)
		}
	}

	// Clean 경로와 비교
	cleanPath := filepath.Clean(path)
	if path != cleanPath {
		return NewBusinessValidationError(
			ErrCodePathNotAccessible,
			"정규화되지 않은 경로입니다",
			"path",
		)
	}

	return nil
}

// checkReadablePermission 읽기 권한 확인
func checkReadablePermission(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return NewBusinessValidationError(
			ErrCodePermissionDenied,
			"경로에 읽기 권한이 없습니다",
			"path",
		)
	}
	file.Close()
	return nil
}

// checkWritablePermission 쓰기 권한 확인
func checkWritablePermission(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return NewBusinessValidationError(
			ErrCodePermissionDenied,
			"경로 정보를 읽을 수 없습니다",
			"path",
		)
	}

	if info.IsDir() {
		// 디렉토리인 경우 테스트 파일 생성
		testFile := filepath.Join(path, ".aicli_write_test")
		file, err := os.Create(testFile)
		if err != nil {
			return NewBusinessValidationError(
				ErrCodePermissionDenied,
				"디렉토리에 쓰기 권한이 없습니다",
				"path",
			)
		}
		file.Close()
		os.Remove(testFile)
	} else {
		// 파일인 경우 쓰기 권한으로 열기 시도
		file, err := os.OpenFile(path, os.O_WRONLY, 0)
		if err != nil {
			return NewBusinessValidationError(
				ErrCodePermissionDenied,
				"파일에 쓰기 권한이 없습니다",
				"path",
			)
		}
		file.Close()
	}

	return nil
}