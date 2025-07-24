package claude

import (
	"context"
	"fmt"
	"testing"
	"time"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAdvancedSessionPool은 고급 세션 풀의 기본 기능을 테스트합니다
func TestAdvancedSessionPool(t *testing.T) {
	// 테스트용 세션 매니저 (Mock)
	mockSessionManager := &MockSessionManager{}
	
	// 고급 풀 설정
	config := DefaultAdvancedPoolConfig()
	config.AutoScaling.MinSessions = 2
	config.AutoScaling.MaxSessions = 10
	config.AutoScaling.Enabled = true
	
	// 고급 세션 풀 생성
	pool := NewAdvancedSessionPool(mockSessionManager, config)
	defer pool.Shutdown()
	
	t.Run("Pool Initialization", func(t *testing.T) {
		assert.NotNil(t, pool)
		assert.NotNil(t, pool.scaler)
		assert.NotNil(t, pool.monitor)
		assert.NotNil(t, pool.loadBalancer)
		assert.NotNil(t, pool.healthChecker)
		assert.NotNil(t, pool.metrics)
	})
	
	t.Run("Session Acquisition", func(t *testing.T) {
		ctx := context.Background()
		config := SessionConfig{
			WorkspaceID:  "test-workspace",
			SystemPrompt: "Test assistant",
			MaxTurns:     10,
		}
		
		// 세션 획득 시도 (Mock이므로 실제 세션 없음)
		_, err := pool.AcquireSession(ctx, config)
		// Mock 구현에서는 에러가 예상됨
		assert.Error(t, err)
	})
	
	t.Run("Pool Statistics", func(t *testing.T) {
		stats := pool.GetPoolStats()
		assert.NotNil(t, stats)
		assert.GreaterOrEqual(t, stats.Size, 0)
		assert.GreaterOrEqual(t, stats.ActiveSessions, 0)
		assert.GreaterOrEqual(t, stats.IdleSessions, 0)
	})
	
	t.Run("Auto Scaling", func(t *testing.T) {
		// 자동 스케일링 활성화 확인
		err := pool.AutoScale(true)
		assert.NoError(t, err)
		
		// 수동 스케일링 테스트
		err = pool.Scale(5)
		assert.NoError(t, err)
	})
}

// TestAutoScaler는 자동 스케일러 기능을 테스트합니다
func TestAutoScaler(t *testing.T) {
	// 실제 풀 생성 (MockAdvancedSessionPool 대신)
	mockSessionManager := &MockSessionManager{}
	poolConfig := DefaultAdvancedPoolConfig()
	realPool := NewAdvancedSessionPool(mockSessionManager, poolConfig)
	defer realPool.Shutdown()
	
	config := AutoScalingConfig{
		Enabled:            true,
		MinSessions:        2,
		MaxSessions:        10,
		TargetUtilization:  0.7,
		ScaleUpThreshold:   0.8,
		ScaleDownThreshold: 0.3,
		ScaleUpCooldown:    time.Minute,
		ScaleDownCooldown:  2 * time.Minute,
		ScaleFactor:        1.5,
	}
	
	scaler := NewAutoScaler(realPool, config)
	defer scaler.Stop()
	
	t.Run("Scaler Initialization", func(t *testing.T) {
		assert.NotNil(t, scaler)
		assert.Equal(t, "none", scaler.GetLastAction())
	})
	
	t.Run("Manual Scale Up", func(t *testing.T) {
		err := scaler.ScaleUp()
		// Mock 구현에서 에러 발생 예상
		assert.Error(t, err)
	})
	
	t.Run("Scaling History", func(t *testing.T) {
		history := scaler.GetScalingHistory()
		assert.NotNil(t, history)
	})
}

// TestPoolMonitor는 풀 모니터 기능을 테스트합니다
func TestPoolMonitor(t *testing.T) {
	// 실제 풀 생성 (MockAdvancedSessionPool 대신)
	mockSessionManager := &MockSessionManager{}
	poolConfig := DefaultAdvancedPoolConfig()
	realPool := NewAdvancedSessionPool(mockSessionManager, poolConfig)
	defer realPool.Shutdown()
	
	config := MonitoringConfig{
		MetricsInterval:      30 * time.Second,
		EnableCPUTracking:    true,
		EnableMemoryTracking: true,
		AlertThresholds: AlertThresholds{
			HighCPUUsage:    0.8,
			HighMemoryUsage: 1024 * 1024 * 1024,
			HighErrorRate:   0.05,
			LowAvailability: 0.95,
		},
	}
	
	monitor := NewPoolMonitor(realPool, config)
	defer monitor.Stop()
	
	t.Run("Monitor Initialization", func(t *testing.T) {
		assert.NotNil(t, monitor)
	})
	
	t.Run("Session Metrics", func(t *testing.T) {
		// 테스트 세션 메트릭 설정
		testMetrics := SessionMetrics{
			SessionID:    "test-session",
			StartTime:    time.Now(),
			LastUsed:     time.Now(),
			RequestCount: 10,
			Status:       SessionStatusActive,
		}
		
		monitor.SetSessionMetrics("test-session", testMetrics)
		
		// 메트릭 조회
		metrics := monitor.GetSessionMetrics()
		assert.Len(t, metrics, 1)
		assert.Equal(t, "test-session", metrics[0].SessionID)
	})
	
	t.Run("System Metrics", func(t *testing.T) {
		systemMetrics := monitor.GetSystemMetrics()
		assert.NotNil(t, systemMetrics)
		assert.Greater(t, systemMetrics.CPUCores, 0)
	})
}

// TestLoadBalancer는 로드 밸런서 기능을 테스트합니다
func TestLoadBalancer(t *testing.T) {
	// 실제 풀 생성 (MockAdvancedSessionPool 대신)
	mockSessionManager := &MockSessionManager{}
	poolConfig := DefaultAdvancedPoolConfig()
	realPool := NewAdvancedSessionPool(mockSessionManager, poolConfig)
	defer realPool.Shutdown()
	
	config := LoadBalancingConfig{
		Strategy:        WeightedRoundRobin,
		SessionAffinity: true,
		WeightedRouting: true,
		HealthAware:     true,
		StickyDuration:  30 * time.Minute,
	}
	
	lb := NewLoadBalancer(realPool, config)
	
	t.Run("LoadBalancer Initialization", func(t *testing.T) {
		assert.NotNil(t, lb)
		assert.NotNil(t, lb.responseTimeTracker)
	})
	
	t.Run("Session Selection", func(t *testing.T) {
		ctx := context.Background()
		sessionConfig := SessionConfig{
			WorkspaceID: "test-workspace",
		}
		
		// 세션 선택 시도 (Mock이므로 실제 세션 없음)
		_, err := lb.SelectSession(ctx, sessionConfig)
		assert.Error(t, err) // Mock에서는 에러 예상
	})
	
	t.Run("Response Time Recording", func(t *testing.T) {
		lb.RecordResponseTime("test-session", 100*time.Millisecond)
		
		// 응답 시간 확인 (실제 구현에서는 더 정확한 테스트 필요)
		avgTime := lb.responseTimeTracker.GetAverageTime("test-session")
		assert.Equal(t, 100*time.Millisecond, avgTime)
	})
	
	t.Run("Session Weights", func(t *testing.T) {
		weights := lb.GetSessionWeights()
		assert.NotNil(t, weights)
	})
}

// TestHealthChecker는 헬스 체커 기능을 테스트합니다
func TestHealthChecker(t *testing.T) {
	// 실제 풀 생성 (MockAdvancedSessionPool 대신)
	mockSessionManager := &MockSessionManager{}
	poolConfig := DefaultAdvancedPoolConfig()
	realPool := NewAdvancedSessionPool(mockSessionManager, poolConfig)
	defer realPool.Shutdown()
	
	config := HealthCheckConfig{
		Interval:         30 * time.Second,
		Timeout:          5 * time.Second,
		FailureThreshold: 3,
		SuccessThreshold: 2,
	}
	
	hc := NewHealthChecker(realPool, config)
	defer hc.Stop()
	
	t.Run("HealthChecker Initialization", func(t *testing.T) {
		assert.NotNil(t, hc)
	})
	
	t.Run("Overall Health", func(t *testing.T) {
		health := hc.GetOverallHealth()
		assert.NotNil(t, health)
		assert.Equal(t, HealthUnknown, health.Status)
	})
	
	t.Run("Session Health Check", func(t *testing.T) {
		result := hc.CheckSessionHealth("test-session")
		assert.NotNil(t, result)
		assert.False(t, result.Success) // Mock에서는 실패 예상
	})
}

// TestPoolMetrics는 풀 메트릭 기능을 테스트합니다
func TestPoolMetrics(t *testing.T) {
	metrics := NewPoolMetrics()
	defer metrics.Stop()
	
	t.Run("Metrics Initialization", func(t *testing.T) {
		assert.NotNil(t, metrics)
	})
	
	t.Run("Action Recording", func(t *testing.T) {
		metrics.RecordAction("acquired", nil)
		metrics.RecordAction("released", nil)
		
		summary := metrics.GetMetricsSummary()
		assert.Greater(t, summary.TotalRequests, int64(0))
	})
	
	t.Run("Latency Recording", func(t *testing.T) {
		metrics.RecordLatency(100 * time.Millisecond)
		metrics.RecordLatency(200 * time.Millisecond)
		
		avgLatency := metrics.GetAverageLatency()
		assert.Equal(t, 150*time.Millisecond, avgLatency)
	})
	
	t.Run("Metrics Summary", func(t *testing.T) {
		summary := metrics.GetMetricsSummary()
		assert.NotNil(t, summary)
		assert.NotZero(t, summary.StartTime)
		assert.Greater(t, summary.Uptime, time.Duration(0))
	})
	
	t.Run("Time Series Data", func(t *testing.T) {
		timeSeriesData := metrics.GetTimeSeriesData()
		assert.NotNil(t, timeSeriesData)
	})
}

// TestLatencyTracker는 지연시간 추적기를 테스트합니다
func TestLatencyTracker(t *testing.T) {
	tracker := NewLatencyTracker(100)
	
	// 샘플 데이터 추가
	latencies := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		150 * time.Millisecond,
		300 * time.Millisecond,
		250 * time.Millisecond,
	}
	
	for _, latency := range latencies {
		tracker.AddSample(latency)
	}
	
	t.Run("Average Calculation", func(t *testing.T) {
		avg := tracker.GetAverage()
		expected := (100 + 200 + 150 + 300 + 250) / 5
		assert.Equal(t, time.Duration(expected)*time.Millisecond, avg)
	})
	
	t.Run("Percentile Calculation", func(t *testing.T) {
		p50 := tracker.GetPercentile(50)
		p95 := tracker.GetPercentile(95)
		
		assert.Greater(t, p50, time.Duration(0))
		assert.Greater(t, p95, p50)
	})
	
	t.Run("Min/Max Values", func(t *testing.T) {
		min := tracker.GetMin()
		max := tracker.GetMax()
		
		assert.Equal(t, 100*time.Millisecond, min)
		assert.Equal(t, 300*time.Millisecond, max)
	})
}

// TestIntegration은 통합 테스트를 수행합니다
func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	t.Run("Full System Integration", func(t *testing.T) {
		// Mock 세션 매니저
		mockSessionManager := &MockSessionManager{}
		
		// 고급 풀 생성
		config := DefaultAdvancedPoolConfig()
		config.AutoScaling.Enabled = false // 테스트에서는 비활성화
		
		pool := NewAdvancedSessionPool(mockSessionManager, config)
		defer pool.Shutdown()
		
		// 메트릭 수집 시작
		pool.metrics.Start()
		
		// 기본 동작 테스트
		ctx := context.Background()
		sessionConfig := SessionConfig{
			WorkspaceID: "integration-test",
		}
		
		// 세션 획득 시도 (실패 예상)
		_, err := pool.AcquireSession(ctx, sessionConfig)
		assert.Error(t, err)
		
		// 메트릭 확인
		summary := pool.metrics.GetMetricsSummary()
		assert.NotNil(t, summary)
		
		// 풀 통계 확인
		stats := pool.GetPoolStats()
		assert.NotNil(t, stats)
	})
}

// Mock 구현들

type MockSessionManager struct{}

func (m *MockSessionManager) CreateSession(ctx context.Context, config SessionConfig) (*Session, error) {
	return nil, fmt.Errorf("mock: session creation not implemented")
}

func (m *MockSessionManager) GetSession(sessionID string) (*Session, error) {
	return nil, fmt.Errorf("mock: session not found")
}

func (m *MockSessionManager) UpdateSession(sessionID string, update SessionUpdate) error {
	return fmt.Errorf("mock: session update not implemented")
}

func (m *MockSessionManager) CloseSession(sessionID string) error {
	return nil
}

func (m *MockSessionManager) ListSessions(filter SessionFilter) ([]*Session, error) {
	return []*Session{}, nil
}

