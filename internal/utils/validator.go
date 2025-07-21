package utils

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// RegisterCustomValidators 커스텀 validator 등록
func RegisterCustomValidators() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		// 디렉토리 존재 여부 검증
		v.RegisterValidation("dir", validateDirectory)
		
		// 안전한 경로 검증 (상위 디렉토리 접근 방지)
		v.RegisterValidation("safepath", validateSafePath)
	}
}

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