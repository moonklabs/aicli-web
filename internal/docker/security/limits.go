package security

import (
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"
)

// ResourceManager 리소스 제한 관리자
type ResourceManager struct {
	config *IsolationConfig
}

// NewResourceManager 새로운 리소스 관리자 생성
func NewResourceManager(config *IsolationConfig) *ResourceManager {
	if config == nil {
		config = DefaultIsolationConfig()
	}
	return &ResourceManager{config: config}
}

// CreateResourceLimits 리소스 제한 설정 생성
func (rm *ResourceManager) CreateResourceLimits() *ResourceLimits {
	return &ResourceLimits{
		CPUShares:      1024, // 기본 가중치
		CPUQuota:       int64(rm.config.DefaultCPULimit * 100000),
		CPUPeriod:      100000,
		Memory:         rm.config.DefaultMemoryLimit,
		MemorySwap:     rm.config.DefaultMemoryLimit, // Swap 비활성화
		PidsLimit:      100,    // 최대 100개 프로세스
		IOMaxBandwidth: "100m", // 100MB/s
		IOMaxIOps:      1000,   // 1000 IOPS
	}
}

// CreateCustomResourceLimits 사용자 정의 리소스 제한 생성
func (rm *ResourceManager) CreateCustomResourceLimits(req *ResourceLimitRequest) *ResourceLimits {
	limits := rm.CreateResourceLimits()

	// 사용자 정의 값 적용
	if req.CPULimit > 0 {
		limits.CPUQuota = int64(req.CPULimit * 100000)
	}
	if req.MemoryLimit > 0 {
		limits.Memory = req.MemoryLimit
		limits.MemorySwap = req.MemoryLimit
	}
	if req.PidsLimit > 0 {
		limits.PidsLimit = req.PidsLimit
	}
	if req.IOBandwidth != "" {
		limits.IOMaxBandwidth = req.IOBandwidth
	}
	if req.IOOps > 0 {
		limits.IOMaxIOps = req.IOOps
	}

	return limits
}

// ToDockerResources Docker container.Resources로 변환
func (rm *ResourceManager) ToDockerResources(limits *ResourceLimits) container.Resources {
	resources := container.Resources{
		CPUShares:  limits.CPUShares,
		CPUQuota:   limits.CPUQuota,
		CPUPeriod:  limits.CPUPeriod,
		Memory:     limits.Memory,
		MemorySwap: limits.MemorySwap,
		
		// Block I/O 제한
		BlkioWeight: 500, // 기본 I/O 가중치
		
		// Device cgroup rules
		DeviceCgroupRules: []string{
			"c 1:3 rmw",   // /dev/null 접근 허용
			"c 1:5 rmw",   // /dev/zero 접근 허용
			"c 1:9 rmw",   // /dev/urandom 접근 허용
			"c 136:* rmw", // PTY 접근 허용
		},
	}

	// PidsLimit 설정 (포인터 사용)
	if limits.PidsLimit > 0 {
		resources.PidsLimit = &limits.PidsLimit
	}

	return resources
}

// ValidateResourceLimits 리소스 제한 검증
func (rm *ResourceManager) ValidateResourceLimits(limits *ResourceLimits) error {
	if limits == nil {
		return fmt.Errorf("resource limits cannot be nil")
	}

	// CPU 제한 검증
	if limits.CPUQuota < 0 {
		return fmt.Errorf("CPU quota cannot be negative")
	}
	if limits.CPUPeriod <= 0 {
		return fmt.Errorf("CPU period must be positive")
	}
	if limits.CPUShares < 0 {
		return fmt.Errorf("CPU shares cannot be negative")
	}

	// 메모리 제한 검증
	if limits.Memory < 0 {
		return fmt.Errorf("memory limit cannot be negative")
	}
	if limits.Memory > 0 && limits.Memory < 4*1024*1024 { // 최소 4MB
		return fmt.Errorf("memory limit too small (minimum 4MB)")
	}

	// Swap 제한 검증
	if limits.MemorySwap != -1 && limits.MemorySwap < limits.Memory {
		return fmt.Errorf("memory+swap limit must be >= memory limit")
	}

	// 프로세스 수 제한 검증
	if limits.PidsLimit < 0 {
		return fmt.Errorf("PIDs limit cannot be negative")
	}
	if limits.PidsLimit > 0 && limits.PidsLimit < 1 {
		return fmt.Errorf("PIDs limit too small (minimum 1)")
	}

	return nil
}

// ValidateResourceUsage 리소스 사용량 검증
func (rm *ResourceManager) ValidateResourceUsage(metrics *WorkspaceMetrics) []ResourceViolation {
	var violations []ResourceViolation

	// CPU 사용률 검사
	if metrics.CPUPercent > 90.0 {
		violations = append(violations, ResourceViolation{
			Type:        "cpu_high_usage",
			Threshold:   90.0,
			Current:     metrics.CPUPercent,
			Description: "CPU usage exceeded 90%",
			Severity:    "warning",
			Timestamp:   time.Now(),
		})
	}

	// 메모리 사용량 검사
	if metrics.MemoryLimit > 0 {
		memoryPercent := float64(metrics.MemoryUsage) / float64(metrics.MemoryLimit) * 100
		if memoryPercent > 85.0 {
			violations = append(violations, ResourceViolation{
				Type:        "memory_high_usage",
				Threshold:   85.0,
				Current:     memoryPercent,
				Description: "Memory usage exceeded 85%",
				Severity:    "warning",
				Timestamp:   time.Now(),
			})
		}
	}

	// 네트워크 I/O 검사
	networkIO := metrics.NetworkRx + metrics.NetworkTx
	networkThresholdMB := float64(100 * 1024 * 1024) // 100MB/s
	if float64(networkIO) > networkThresholdMB {
		violations = append(violations, ResourceViolation{
			Type:        "network_high_io",
			Threshold:   networkThresholdMB,
			Current:     float64(networkIO),
			Description: "Network I/O exceeded 100MB/s",
			Severity:    "info",
			Timestamp:   time.Now(),
		})
	}

	// 디스크 I/O 검사
	diskIO := metrics.DiskRead + metrics.DiskWrite
	diskThresholdMB := float64(50 * 1024 * 1024) // 50MB/s
	if float64(diskIO) > diskThresholdMB {
		violations = append(violations, ResourceViolation{
			Type:        "disk_high_io",
			Threshold:   diskThresholdMB,
			Current:     float64(diskIO),
			Description: "Disk I/O exceeded 50MB/s",
			Severity:    "info",
			Timestamp:   time.Now(),
		})
	}

	return violations
}

// CalculateOptimalLimits 워크로드에 따른 최적 리소스 제한 계산
func (rm *ResourceManager) CalculateOptimalLimits(workloadType WorkloadType, historyMetrics []WorkspaceMetrics) *ResourceLimits {
	limits := rm.CreateResourceLimits()

	// 기본 워크로드 타입별 설정
	switch workloadType {
	case WorkloadTypeDevelopment:
		// 개발 환경: 높은 CPU, 중간 메모리
		limits.CPUQuota = int64(2.0 * 100000) // 2 CPU
		limits.Memory = 1024 * 1024 * 1024    // 1GB
		limits.PidsLimit = 200
		
	case WorkloadTypeBuild:
		// 빌드 환경: 매우 높은 CPU, 높은 메모리
		limits.CPUQuota = int64(4.0 * 100000) // 4 CPU
		limits.Memory = 2048 * 1024 * 1024    // 2GB
		limits.PidsLimit = 500
		limits.IOMaxIOps = 2000
		
	case WorkloadTypeTest:
		// 테스트 환경: 중간 CPU, 낮은 메모리
		limits.CPUQuota = int64(1.0 * 100000) // 1 CPU
		limits.Memory = 512 * 1024 * 1024     // 512MB
		limits.PidsLimit = 100
		
	case WorkloadTypeProduction:
		// 프로덕션: 안정적인 리소스 배분
		limits.CPUQuota = int64(1.5 * 100000) // 1.5 CPU
		limits.Memory = 1024 * 1024 * 1024    // 1GB
		limits.PidsLimit = 300
		limits.IOMaxIOps = 1500
		
	default:
		// 기본값 유지
	}

	// 히스토리 메트릭을 이용한 동적 조정
	if len(historyMetrics) > 0 {
		avgCPU, avgMemory := rm.calculateAverageUsage(historyMetrics)
		
		// CPU 사용량 기반 조정 (평균 사용량의 150% 여유 확보)
		if avgCPU > 0 {
			recommendedCPU := avgCPU * 1.5
			if recommendedCPU > float64(limits.CPUQuota)/100000 {
				limits.CPUQuota = int64(recommendedCPU * 100000)
			}
		}
		
		// 메모리 사용량 기반 조정 (평균 사용량의 120% 여유 확보)
		if avgMemory > 0 {
			recommendedMemory := int64(avgMemory * 1.2)
			if recommendedMemory > limits.Memory {
				limits.Memory = recommendedMemory
				limits.MemorySwap = recommendedMemory
			}
		}
	}

	return limits
}

// calculateAverageUsage 히스토리 메트릭에서 평균 사용량 계산
func (rm *ResourceManager) calculateAverageUsage(metrics []WorkspaceMetrics) (float64, float64) {
	if len(metrics) == 0 {
		return 0, 0
	}

	var totalCPU, totalMemory float64
	for _, m := range metrics {
		totalCPU += m.CPUPercent / 100.0 // CPU 코어 수로 변환
		totalMemory += float64(m.MemoryUsage)
	}

	return totalCPU / float64(len(metrics)), totalMemory / float64(len(metrics))
}

// GetResourceLimitPreset 미리 정의된 리소스 제한 프리셋 반환
func (rm *ResourceManager) GetResourceLimitPreset(preset ResourcePreset) *ResourceLimits {
	switch preset {
	case ResourcePresetMinimal:
		return &ResourceLimits{
			CPUShares:      512,
			CPUQuota:       50000, // 0.5 CPU
			CPUPeriod:      100000,
			Memory:         256 * 1024 * 1024, // 256MB
			MemorySwap:     256 * 1024 * 1024,
			PidsLimit:      50,
			IOMaxBandwidth: "50m",
			IOMaxIOps:      500,
		}
		
	case ResourcePresetSmall:
		return &ResourceLimits{
			CPUShares:      1024,
			CPUQuota:       100000, // 1 CPU
			CPUPeriod:      100000,
			Memory:         512 * 1024 * 1024, // 512MB
			MemorySwap:     512 * 1024 * 1024,
			PidsLimit:      100,
			IOMaxBandwidth: "100m",
			IOMaxIOps:      1000,
		}
		
	case ResourcePresetMedium:
		return &ResourceLimits{
			CPUShares:      2048,
			CPUQuota:       200000, // 2 CPU
			CPUPeriod:      100000,
			Memory:         1024 * 1024 * 1024, // 1GB
			MemorySwap:     1024 * 1024 * 1024,
			PidsLimit:      200,
			IOMaxBandwidth: "200m",
			IOMaxIOps:      2000,
		}
		
	case ResourcePresetLarge:
		return &ResourceLimits{
			CPUShares:      4096,
			CPUQuota:       400000, // 4 CPU
			CPUPeriod:      100000,
			Memory:         2048 * 1024 * 1024, // 2GB
			MemorySwap:     2048 * 1024 * 1024,
			PidsLimit:      500,
			IOMaxBandwidth: "500m",
			IOMaxIOps:      5000,
		}
		
	default:
		return rm.CreateResourceLimits()
	}
}

// ResourceLimitRequest 리소스 제한 요청
type ResourceLimitRequest struct {
	CPULimit    float64 `json:"cpu_limit"`    // CPU 코어 수
	MemoryLimit int64   `json:"memory_limit"` // 바이트 단위 메모리
	PidsLimit   int64   `json:"pids_limit"`   // 프로세스 수 제한
	IOBandwidth string  `json:"io_bandwidth"` // I/O 대역폭 (예: "100m")
	IOOps       int64   `json:"io_ops"`       // IOPS 제한
}

// ResourceViolation 리소스 위반 정보
type ResourceViolation struct {
	Type        string    `json:"type"`
	Threshold   float64   `json:"threshold"`
	Current     float64   `json:"current"`
	Description string    `json:"description"`
	Severity    string    `json:"severity"` // info, warning, error, critical
	Timestamp   time.Time `json:"timestamp"`
}

// WorkspaceMetrics 워크스페이스 메트릭
type WorkspaceMetrics struct {
	WorkspaceID   string    `json:"workspace_id"`
	CPUPercent    float64   `json:"cpu_percent"`    // CPU 사용률 (0-100%)
	MemoryUsage   int64     `json:"memory_usage"`   // 메모리 사용량 (바이트)
	MemoryLimit   int64     `json:"memory_limit"`   // 메모리 제한 (바이트)
	NetworkRx     int64     `json:"network_rx"`     // 네트워크 수신 바이트/초
	NetworkTx     int64     `json:"network_tx"`     // 네트워크 송신 바이트/초
	DiskRead      int64     `json:"disk_read"`      // 디스크 읽기 바이트/초
	DiskWrite     int64     `json:"disk_write"`     // 디스크 쓰기 바이트/초
	ProcessCount  int       `json:"process_count"`  // 프로세스 수
	Timestamp     time.Time `json:"timestamp"`
}

// WorkloadType 워크로드 타입
type WorkloadType string

const (
	WorkloadTypeDevelopment WorkloadType = "development" // 개발 환경
	WorkloadTypeBuild       WorkloadType = "build"       // 빌드 환경
	WorkloadTypeTest        WorkloadType = "test"        // 테스트 환경
	WorkloadTypeProduction  WorkloadType = "production"  // 프로덕션 환경
)

// ResourcePreset 리소스 프리셋
type ResourcePreset string

const (
	ResourcePresetMinimal ResourcePreset = "minimal" // 최소 리소스
	ResourcePresetSmall   ResourcePreset = "small"   // 소형 리소스
	ResourcePresetMedium  ResourcePreset = "medium"  // 중형 리소스
	ResourcePresetLarge   ResourcePreset = "large"   // 대형 리소스
)