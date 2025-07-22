package docker

import (
	"fmt"
	"strings"
	"time"
)


// GenerateImageTag 워크스페이스용 이미지 태그를 생성합니다.
func (c *Client) GenerateImageTag(workspaceID string) string {
	return fmt.Sprintf("aicli-workspace:%s", workspaceID)
}


// GenerateNetworkName 네트워크 이름을 생성합니다.
func (c *Client) GenerateNetworkName(suffix string) string {
	if suffix == "" {
		return c.config.NetworkName
	}
	return fmt.Sprintf("%s_%s", c.config.NetworkName, suffix)
}

// SanitizeName Docker 네이밍 규칙에 맞게 이름을 정리합니다.
func SanitizeName(name string) string {
	// Docker 네이밍 규칙에 맞게 정리
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")

	// 허용된 문자만 유지 (영문자, 숫자, 하이픈)
	var result strings.Builder
	for _, char := range name {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' {
			result.WriteRune(char)
		}
	}

	sanitized := result.String()

	// 시작과 끝의 하이픈 제거
	sanitized = strings.Trim(sanitized, "-")

	// 빈 문자열인 경우 기본값 반환
	if sanitized == "" {
		return "default"
	}

	return sanitized
}

// ValidateContainerName 컨테이너 이름의 유효성을 검사합니다.
func ValidateContainerName(name string) error {
	if name == "" {
		return fmt.Errorf("container name cannot be empty")
	}

	// Docker 컨테이너 이름 규칙 검사
	if len(name) > 253 {
		return fmt.Errorf("container name too long (max 253 characters)")
	}

	// 유효한 문자 검사
	for i, char := range name {
		if i == 0 && char == '/' {
			continue // 첫 번째 문자가 /인 경우는 허용
		}

		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-' || char == '_' || char == '.') {
			return fmt.Errorf("invalid character '%c' in container name", char)
		}
	}

	return nil
}

// ValidateImageName 이미지 이름의 유효성을 검사합니다.
func ValidateImageName(image string) error {
	if image == "" {
		return fmt.Errorf("image name cannot be empty")
	}

	// 기본적인 이미지 이름 형식 검사
	parts := strings.Split(image, ":")
	if len(parts) > 2 {
		return fmt.Errorf("invalid image name format")
	}

	imageName := parts[0]
	if imageName == "" {
		return fmt.Errorf("image name cannot be empty")
	}

	// 태그 검사 (있는 경우)
	if len(parts) == 2 {
		tag := parts[1]
		if tag == "" {
			return fmt.Errorf("image tag cannot be empty")
		}
	}

	return nil
}

// FormatBytes 바이트 단위를 사람이 읽기 쉬운 형태로 변환합니다.
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatDuration 시간을 사람이 읽기 쉬운 형태로 변환합니다.
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.0fm%.0fs", d.Minutes(), d.Seconds()-60*d.Minutes())
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%.0fh%.0fm", d.Hours(), d.Minutes()-60*d.Hours())
	}
	return fmt.Sprintf("%.0fd%.0fh", d.Hours()/24, d.Hours()-24*(d.Hours()/24))
}

// MergeLabels 여러 레이블 맵을 병합합니다.
func MergeLabels(labelMaps ...map[string]string) map[string]string {
	result := make(map[string]string)
	for _, labels := range labelMaps {
		for k, v := range labels {
			result[k] = v
		}
	}
	return result
}

// FilterByLabels 레이블로 필터링하는 필터를 생성합니다.
func (c *Client) FilterByLabels(labels map[string]string) map[string][]string {
	filters := make(map[string][]string)
	for k, v := range labels {
		key := fmt.Sprintf("label=%s=%s", k, v)
		filters["label"] = append(filters["label"], key)
	}
	return filters
}

// GetManagedResourceFilter aicli에서 관리하는 리소스 필터를 반환합니다.
func (c *Client) GetManagedResourceFilter() map[string][]string {
	return map[string][]string{
		"label": {fmt.Sprintf("%s.managed=true", c.labelPrefix)},
	}
}

// ParseResourceLimits 리소스 제한 문자열을 파싱합니다.
func ParseResourceLimits(cpu string, memory string) (float64, int64, error) {
	var cpuLimit float64 = 0
	var memoryLimit int64 = 0

	// CPU 제한 파싱
	if cpu != "" {
		if _, err := fmt.Sscanf(cpu, "%f", &cpuLimit); err != nil {
			return 0, 0, fmt.Errorf("invalid CPU limit format: %s", cpu)
		}
		if cpuLimit <= 0 {
			return 0, 0, fmt.Errorf("CPU limit must be positive: %f", cpuLimit)
		}
	}

	// 메모리 제한 파싱
	if memory != "" {
		memoryStr := strings.ToUpper(memory)
		var multiplier int64 = 1

		if strings.HasSuffix(memoryStr, "K") || strings.HasSuffix(memoryStr, "KB") {
			multiplier = 1024
			memoryStr = strings.TrimSuffix(strings.TrimSuffix(memoryStr, "B"), "K")
		} else if strings.HasSuffix(memoryStr, "M") || strings.HasSuffix(memoryStr, "MB") {
			multiplier = 1024 * 1024
			memoryStr = strings.TrimSuffix(strings.TrimSuffix(memoryStr, "B"), "M")
		} else if strings.HasSuffix(memoryStr, "G") || strings.HasSuffix(memoryStr, "GB") {
			multiplier = 1024 * 1024 * 1024
			memoryStr = strings.TrimSuffix(strings.TrimSuffix(memoryStr, "B"), "G")
		}

		var memValue int64
		if _, err := fmt.Sscanf(memoryStr, "%d", &memValue); err != nil {
			return 0, 0, fmt.Errorf("invalid memory limit format: %s", memory)
		}
		if memValue <= 0 {
			return 0, 0, fmt.Errorf("memory limit must be positive: %s", memory)
		}

		memoryLimit = memValue * multiplier
	}

	return cpuLimit, memoryLimit, nil
}