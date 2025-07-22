package security

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewResourceManager(t *testing.T) {
	t.Run("with config", func(t *testing.T) {
		config := &IsolationConfig{
			DefaultCPULimit:    2.0,
			DefaultMemoryLimit: 1024 * 1024 * 1024,
		}
		
		rm := NewResourceManager(config)
		
		assert.NotNil(t, rm)
		assert.Equal(t, config, rm.config)
	})
	
	t.Run("with nil config", func(t *testing.T) {
		rm := NewResourceManager(nil)
		
		assert.NotNil(t, rm)
		assert.NotNil(t, rm.config)
		// 기본 설정으로 초기화되어야 함
		assert.Equal(t, 1.0, rm.config.DefaultCPULimit)
	})
}

func TestResourceManagerCreateResourceLimits(t *testing.T) {
	config := &IsolationConfig{
		DefaultCPULimit:    2.0,
		DefaultMemoryLimit: 1024 * 1024 * 1024, // 1GB
	}
	rm := NewResourceManager(config)
	
	limits := rm.CreateResourceLimits()
	
	assert.NotNil(t, limits)
	assert.Equal(t, int64(1024), limits.CPUShares)
	assert.Equal(t, int64(200000), limits.CPUQuota) // 2.0 * 100000
	assert.Equal(t, int64(100000), limits.CPUPeriod)
	assert.Equal(t, int64(1024*1024*1024), limits.Memory)
	assert.Equal(t, int64(1024*1024*1024), limits.MemorySwap)
	assert.Equal(t, int64(100), limits.PidsLimit)
	assert.Equal(t, "100m", limits.IOMaxBandwidth)
	assert.Equal(t, int64(1000), limits.IOMaxIOps)
}

func TestCreateCustomResourceLimits(t *testing.T) {
	rm := NewResourceManager(nil)
	req := &ResourceLimitRequest{
		CPULimit:    1.5,
		MemoryLimit: 2048 * 1024 * 1024, // 2GB
		PidsLimit:   150,
		IOBandwidth: "200m",
		IOOps:       2000,
	}
	
	limits := rm.CreateCustomResourceLimits(req)
	
	assert.NotNil(t, limits)
	assert.Equal(t, int64(150000), limits.CPUQuota) // 1.5 * 100000
	assert.Equal(t, int64(2048*1024*1024), limits.Memory)
	assert.Equal(t, int64(2048*1024*1024), limits.MemorySwap)
	assert.Equal(t, int64(150), limits.PidsLimit)
	assert.Equal(t, "200m", limits.IOMaxBandwidth)
	assert.Equal(t, int64(2000), limits.IOMaxIOps)
}

func TestCreateCustomResourceLimits_DefaultValues(t *testing.T) {
	rm := NewResourceManager(nil)
	req := &ResourceLimitRequest{
		CPULimit: 0, // 기본값 사용
		IOOps:    0, // 기본값 사용
	}
	
	limits := rm.CreateCustomResourceLimits(req)
	
	// 기본값이 유지되어야 함
	assert.Equal(t, int64(100000), limits.CPUQuota) // 기본 1.0 * 100000
	assert.Equal(t, int64(1000), limits.IOMaxIOps)  // 기본값
}

func TestToDockerResources(t *testing.T) {
	rm := NewResourceManager(nil)
	limits := &ResourceLimits{
		CPUShares:  2048,
		CPUQuota:   200000,
		CPUPeriod:  100000,
		Memory:     1024 * 1024 * 1024,
		MemorySwap: 1024 * 1024 * 1024,
		PidsLimit:  200,
	}
	
	resources := rm.ToDockerResources(limits)
	
	assert.Equal(t, limits.CPUShares, resources.CPUShares)
	assert.Equal(t, limits.CPUQuota, resources.CPUQuota)
	assert.Equal(t, limits.CPUPeriod, resources.CPUPeriod)
	assert.Equal(t, limits.Memory, resources.Memory)
	assert.Equal(t, limits.MemorySwap, resources.MemorySwap)
	assert.Equal(t, limits.PidsLimit, *resources.PidsLimit)
	assert.Equal(t, uint16(500), resources.BlkioWeight)
	assert.NotEmpty(t, resources.DeviceCgroupRules)
	assert.Contains(t, resources.DeviceCgroupRules, "c 1:3 rmw")   // /dev/null
	assert.Contains(t, resources.DeviceCgroupRules, "c 1:9 rmw")   // /dev/urandom
	assert.Contains(t, resources.DeviceCgroupRules, "c 136:* rmw") // PTY
}

func TestToDockerResources_ZeroPidsLimit(t *testing.T) {
	rm := NewResourceManager(nil)
	limits := &ResourceLimits{
		PidsLimit: 0,
	}
	
	resources := rm.ToDockerResources(limits)
	
	assert.Nil(t, resources.PidsLimit) // 0이면 포인터가 설정되지 않음
}

func TestValidateResourceLimits(t *testing.T) {
	rm := NewResourceManager(nil)
	
	tests := []struct {
		name        string
		limits      *ResourceLimits
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil limits",
			limits:      nil,
			expectError: true,
			errorMsg:    "resource limits cannot be nil",
		},
		{
			name: "negative CPU quota",
			limits: &ResourceLimits{
				CPUQuota:  -1000,
				CPUPeriod: 100000,
			},
			expectError: true,
			errorMsg:    "CPU quota cannot be negative",
		},
		{
			name: "zero CPU period",
			limits: &ResourceLimits{
				CPUQuota:  100000,
				CPUPeriod: 0,
			},
			expectError: true,
			errorMsg:    "CPU period must be positive",
		},
		{
			name: "negative CPU shares",
			limits: &ResourceLimits{
				CPUShares: -1,
				CPUPeriod: 100000,
			},
			expectError: true,
			errorMsg:    "CPU shares cannot be negative",
		},
		{
			name: "negative memory limit",
			limits: &ResourceLimits{
				Memory:    -1,
				CPUPeriod: 100000,
			},
			expectError: true,
			errorMsg:    "memory limit cannot be negative",
		},
		{
			name: "memory limit too small",
			limits: &ResourceLimits{
				Memory:    1024 * 1024, // 1MB < 4MB minimum
				CPUPeriod: 100000,
			},
			expectError: true,
			errorMsg:    "memory limit too small",
		},
		{
			name: "invalid memory swap",
			limits: &ResourceLimits{
				Memory:     512 * 1024 * 1024, // 512MB
				MemorySwap: 256 * 1024 * 1024, // 256MB < Memory
				CPUPeriod:  100000,
			},
			expectError: true,
			errorMsg:    "memory+swap limit must be >= memory limit",
		},
		{
			name: "negative PIDs limit",
			limits: &ResourceLimits{
				PidsLimit: -1,
				CPUPeriod: 100000,
			},
			expectError: true,
			errorMsg:    "PIDs limit cannot be negative",
		},
		{
			name: "PIDs limit too small",
			limits: &ResourceLimits{
				PidsLimit: 0, // 0은 허용되지만 양수인 경우 최소 1
				CPUPeriod: 100000,
			},
			expectError: false,
		},
		{
			name: "valid limits",
			limits: &ResourceLimits{
				CPUShares:  1024,
				CPUQuota:   100000,
				CPUPeriod:  100000,
				Memory:     512 * 1024 * 1024,
				MemorySwap: 512 * 1024 * 1024,
				PidsLimit:  100,
			},
			expectError: false,
		},
		{
			name: "memory swap -1 (unlimited)",
			limits: &ResourceLimits{
				Memory:     512 * 1024 * 1024,
				MemorySwap: -1, // unlimited
				CPUPeriod:  100000,
			},
			expectError: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rm.ValidateResourceLimits(tt.limits)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateResourceUsage(t *testing.T) {
	rm := NewResourceManager(nil)
	
	tests := []struct {
		name               string
		metrics            *WorkspaceMetrics
		expectedViolations []string
	}{
		{
			name: "high CPU usage",
			metrics: &WorkspaceMetrics{
				CPUPercent:   95.0, // > 90%
				MemoryUsage:  400 * 1024 * 1024,
				MemoryLimit:  1024 * 1024 * 1024,
				NetworkRx:    10 * 1024 * 1024,
				NetworkTx:    10 * 1024 * 1024,
			},
			expectedViolations: []string{"cpu_high_usage"},
		},
		{
			name: "high memory usage",
			metrics: &WorkspaceMetrics{
				CPUPercent:   50.0,
				MemoryUsage:  900 * 1024 * 1024, // 90% of 1GB > 85%
				MemoryLimit:  1024 * 1024 * 1024,
				NetworkRx:    10 * 1024 * 1024,
				NetworkTx:    10 * 1024 * 1024,
			},
			expectedViolations: []string{"memory_high_usage"},
		},
		{
			name: "high network IO",
			metrics: &WorkspaceMetrics{
				CPUPercent:   50.0,
				MemoryUsage:  400 * 1024 * 1024,
				MemoryLimit:  1024 * 1024 * 1024,
				NetworkRx:    60 * 1024 * 1024, // 60MB/s
				NetworkTx:    50 * 1024 * 1024, // 50MB/s -> total 110MB/s > 100MB/s
			},
			expectedViolations: []string{"network_high_io"},
		},
		{
			name: "high disk IO",
			metrics: &WorkspaceMetrics{
				CPUPercent:  50.0,
				MemoryUsage: 400 * 1024 * 1024,
				MemoryLimit: 1024 * 1024 * 1024,
				DiskRead:    30 * 1024 * 1024, // 30MB/s
				DiskWrite:   25 * 1024 * 1024, // 25MB/s -> total 55MB/s > 50MB/s
			},
			expectedViolations: []string{"disk_high_io"},
		},
		{
			name: "multiple violations",
			metrics: &WorkspaceMetrics{
				CPUPercent:   95.0, // CPU violation
				MemoryUsage:  900 * 1024 * 1024, // Memory violation
				MemoryLimit:  1024 * 1024 * 1024,
				NetworkRx:    110 * 1024 * 1024, // Network violation
				NetworkTx:    10 * 1024 * 1024,
				DiskRead:     60 * 1024 * 1024, // Disk violation
				DiskWrite:    10 * 1024 * 1024,
			},
			expectedViolations: []string{"cpu_high_usage", "memory_high_usage", "network_high_io", "disk_high_io"},
		},
		{
			name: "no violations",
			metrics: &WorkspaceMetrics{
				CPUPercent:  50.0,
				MemoryUsage: 400 * 1024 * 1024,
				MemoryLimit: 1024 * 1024 * 1024,
				NetworkRx:   10 * 1024 * 1024,
				NetworkTx:   10 * 1024 * 1024,
				DiskRead:    10 * 1024 * 1024,
				DiskWrite:   10 * 1024 * 1024,
			},
			expectedViolations: []string{},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations := rm.ValidateResourceUsage(tt.metrics)
			
			assert.Len(t, violations, len(tt.expectedViolations))
			
			actualTypes := make([]string, len(violations))
			for i, v := range violations {
				actualTypes[i] = v.Type
				assert.NotEmpty(t, v.Description)
				assert.NotEmpty(t, v.Severity)
				assert.WithinDuration(t, time.Now(), v.Timestamp, time.Second)
			}
			
			for _, expectedType := range tt.expectedViolations {
				assert.Contains(t, actualTypes, expectedType)
			}
		})
	}
}

func TestCalculateOptimalLimits(t *testing.T) {
	rm := NewResourceManager(nil)
	
	// 히스토리 메트릭 없이 워크로드 타입별 테스트
	t.Run("development workload", func(t *testing.T) {
		limits := rm.CalculateOptimalLimits(WorkloadTypeDevelopment, nil)
		
		assert.Equal(t, int64(200000), limits.CPUQuota) // 2 CPU
		assert.Equal(t, int64(1024*1024*1024), limits.Memory) // 1GB
		assert.Equal(t, int64(200), limits.PidsLimit)
	})
	
	t.Run("build workload", func(t *testing.T) {
		limits := rm.CalculateOptimalLimits(WorkloadTypeBuild, nil)
		
		assert.Equal(t, int64(400000), limits.CPUQuota) // 4 CPU
		assert.Equal(t, int64(2048*1024*1024), limits.Memory) // 2GB
		assert.Equal(t, int64(500), limits.PidsLimit)
		assert.Equal(t, int64(2000), limits.IOMaxIOps)
	})
	
	t.Run("test workload", func(t *testing.T) {
		limits := rm.CalculateOptimalLimits(WorkloadTypeTest, nil)
		
		assert.Equal(t, int64(100000), limits.CPUQuota) // 1 CPU
		assert.Equal(t, int64(512*1024*1024), limits.Memory) // 512MB
		assert.Equal(t, int64(100), limits.PidsLimit)
	})
	
	t.Run("production workload", func(t *testing.T) {
		limits := rm.CalculateOptimalLimits(WorkloadTypeProduction, nil)
		
		assert.Equal(t, int64(150000), limits.CPUQuota) // 1.5 CPU
		assert.Equal(t, int64(1024*1024*1024), limits.Memory) // 1GB
		assert.Equal(t, int64(300), limits.PidsLimit)
		assert.Equal(t, int64(1500), limits.IOMaxIOps)
	})
	
	// 히스토리 메트릭과 함께 테스트
	t.Run("with history metrics", func(t *testing.T) {
		historyMetrics := []WorkspaceMetrics{
			{
				CPUPercent:  120.0, // 1.2 CPU cores
				MemoryUsage: 1500 * 1024 * 1024, // 1.5GB
			},
			{
				CPUPercent:  180.0, // 1.8 CPU cores
				MemoryUsage: 1300 * 1024 * 1024, // 1.3GB
			},
		}
		
		limits := rm.CalculateOptimalLimits(WorkloadTypeDevelopment, historyMetrics)
		
		// 평균 CPU 사용량: (1.2 + 1.8) / 2 = 1.5 cores
		// 권장 CPU: 1.5 * 1.5 = 2.25 cores > 기본 2.0 cores
		assert.True(t, limits.CPUQuota > 200000) // > 2 CPU
		
		// 평균 메모리 사용량: (1.5GB + 1.3GB) / 2 = 1.4GB
		// 권장 메모리: 1.4GB * 1.2 = 1.68GB > 기본 1GB
		assert.True(t, limits.Memory > 1024*1024*1024) // > 1GB
	})
}

func TestGetResourceLimitPreset(t *testing.T) {
	rm := NewResourceManager(nil)
	
	tests := []struct {
		name             string
		preset           ResourcePreset
		expectedCPUQuota int64
		expectedMemory   int64
		expectedPids     int64
	}{
		{
			name:             "minimal preset",
			preset:           ResourcePresetMinimal,
			expectedCPUQuota: 50000, // 0.5 CPU
			expectedMemory:   256 * 1024 * 1024, // 256MB
			expectedPids:     50,
		},
		{
			name:             "small preset",
			preset:           ResourcePresetSmall,
			expectedCPUQuota: 100000, // 1 CPU
			expectedMemory:   512 * 1024 * 1024, // 512MB
			expectedPids:     100,
		},
		{
			name:             "medium preset",
			preset:           ResourcePresetMedium,
			expectedCPUQuota: 200000, // 2 CPU
			expectedMemory:   1024 * 1024 * 1024, // 1GB
			expectedPids:     200,
		},
		{
			name:             "large preset",
			preset:           ResourcePresetLarge,
			expectedCPUQuota: 400000, // 4 CPU
			expectedMemory:   2048 * 1024 * 1024, // 2GB
			expectedPids:     500,
		},
		{
			name:             "invalid preset",
			preset:           ResourcePreset("invalid"),
			expectedCPUQuota: 100000, // 기본값
			expectedMemory:   512 * 1024 * 1024, // 기본값
			expectedPids:     100,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limits := rm.GetResourceLimitPreset(tt.preset)
			
			assert.NotNil(t, limits)
			assert.Equal(t, tt.expectedCPUQuota, limits.CPUQuota)
			assert.Equal(t, tt.expectedMemory, limits.Memory)
			assert.Equal(t, tt.expectedPids, limits.PidsLimit)
		})
	}
}

func TestCalculateAverageUsage(t *testing.T) {
	rm := NewResourceManager(nil)
	
	t.Run("empty metrics", func(t *testing.T) {
		avgCPU, avgMemory := rm.calculateAverageUsage([]WorkspaceMetrics{})
		
		assert.Equal(t, 0.0, avgCPU)
		assert.Equal(t, 0.0, avgMemory)
	})
	
	t.Run("single metric", func(t *testing.T) {
		metrics := []WorkspaceMetrics{
			{
				CPUPercent:  150.0, // 1.5 cores
				MemoryUsage: 1024 * 1024 * 1024, // 1GB
			},
		}
		
		avgCPU, avgMemory := rm.calculateAverageUsage(metrics)
		
		assert.Equal(t, 1.5, avgCPU)
		assert.Equal(t, float64(1024*1024*1024), avgMemory)
	})
	
	t.Run("multiple metrics", func(t *testing.T) {
		metrics := []WorkspaceMetrics{
			{
				CPUPercent:  100.0, // 1.0 cores
				MemoryUsage: 512 * 1024 * 1024, // 512MB
			},
			{
				CPUPercent:  200.0, // 2.0 cores
				MemoryUsage: 1024 * 1024 * 1024, // 1GB
			},
			{
				CPUPercent:  150.0, // 1.5 cores
				MemoryUsage: 768 * 1024 * 1024, // 768MB
			},
		}
		
		avgCPU, avgMemory := rm.calculateAverageUsage(metrics)
		
		// 평균 CPU: (1.0 + 2.0 + 1.5) / 3 = 1.5 cores
		assert.Equal(t, 1.5, avgCPU)
		
		// 평균 메모리: (512MB + 1024MB + 768MB) / 3 = 768MB
		expectedMemory := float64(512*1024*1024 + 1024*1024*1024 + 768*1024*1024) / 3
		assert.Equal(t, expectedMemory, avgMemory)
	})
}

// 벤치마크 테스트
func BenchmarkCreateResourceLimits(b *testing.B) {
	rm := NewResourceManager(nil)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rm.CreateResourceLimits()
	}
}

func BenchmarkValidateResourceUsage(b *testing.B) {
	rm := NewResourceManager(nil)
	metrics := &WorkspaceMetrics{
		CPUPercent:   75.0,
		MemoryUsage:  800 * 1024 * 1024,
		MemoryLimit:  1024 * 1024 * 1024,
		NetworkRx:    50 * 1024 * 1024,
		NetworkTx:    30 * 1024 * 1024,
		DiskRead:     25 * 1024 * 1024,
		DiskWrite:    20 * 1024 * 1024,
		ProcessCount: 25,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rm.ValidateResourceUsage(metrics)
	}
}

func BenchmarkToDockerResources(b *testing.B) {
	rm := NewResourceManager(nil)
	limits := rm.CreateResourceLimits()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rm.ToDockerResources(limits)
	}
}