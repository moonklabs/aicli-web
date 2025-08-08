package flow

import (
	"time"
)

// FlowControlConfig 플로우 제어 전체 설정
type FlowControlConfig struct {
	// 백프레셔 임계값 설정
	Thresholds BackpressureThresholds
	
	// 스로틀링 설정
	Throttle ThrottleConfig
	
	// 버퍼 관리 설정
	Buffer BufferConfig
	
	// 모니터링 설정
	Monitor MonitorConfig
	
	// 메시지 드롭 정책
	DropPolicy DropPolicy
	
	// 전역 메모리 제한
	GlobalMemoryLimit int64
	
	// 시스템 부하 임계값
	SystemLoadThreshold float64
}

// BackpressureThresholds 백프레셔 감지 임계값
type BackpressureThresholds struct {
	BufferUtilizationLow    float64       // 낮은 백프레셔 시작점 (기본: 0.6)
	BufferUtilizationMedium float64       // 중간 백프레셔 시작점 (기본: 0.75) 
	BufferUtilizationHigh   float64       // 높은 백프레셔 시작점 (기본: 0.9)
	ProcessingRateMin       float64       // 최소 처리 속도 (bytes/sec)
	LatencyThresholdMs      int64         // 지연 시간 임계값 (ms)
	MemoryPressureMax       float64       // 최대 메모리 압박 (기본: 0.8)
	QueueDepthMax           int           // 최대 큐 깊이
	MonitorInterval         time.Duration // 모니터링 간격
}

// ThrottleConfig 동적 스로틀링 설정
type ThrottleConfig struct {
	MinRate            float64       // 최소 전송률 (messages/sec)
	MaxRate            float64       // 최대 전송률 (messages/sec)
	AdjustmentFactor   float64       // 조정 비율 (기본: 0.2)
	AdjustmentInterval time.Duration // 조정 간격
	RecoveryRate       float64       // 복구 속도 (기본: 0.1)
	SystemLoadThreshold float64      // 시스템 부하 임계값
	AdaptiveMode       bool          // 적응형 모드 활성화
}

// BufferConfig 적응형 버퍼 설정
type BufferConfig struct {
	InitialSize       int           // 초기 버퍼 크기 (bytes)
	MinSize           int           // 최소 버퍼 크기
	MaxSize           int           // 최대 버퍼 크기
	GrowthFactor      float64       // 증가 비율 (기본: 1.5)
	ShrinkFactor      float64       // 감소 비율 (기본: 0.8) 
	ResizeInterval    time.Duration // 크기 조정 간격
	UtilizationWindow int           // 사용률 히스토리 윈도우
	MemoryLimit       int64         // 버퍼별 메모리 제한
}

// MonitorConfig 모니터링 설정
type MonitorConfig struct {
	MonitorInterval    time.Duration // 모니터링 간격
	MetricsWindow      int           // 메트릭 유지 윈도우
	AlertThreshold     float64       // 경고 임계값
	EnablePrometheus   bool          // Prometheus 메트릭 활성화
	EnableLogging      bool          // 상세 로깅 활성화
	StatsReportInterval time.Duration // 통계 보고 간격
}

// DefaultFlowControlConfig 기본 플로우 제어 설정 반환
func DefaultFlowControlConfig() *FlowControlConfig {
	return &FlowControlConfig{
		Thresholds: BackpressureThresholds{
			BufferUtilizationLow:    0.6,
			BufferUtilizationMedium: 0.75,
			BufferUtilizationHigh:   0.9,
			ProcessingRateMin:       1024 * 100, // 100KB/s
			LatencyThresholdMs:      500,
			MemoryPressureMax:       0.8,
			QueueDepthMax:           1000,
			MonitorInterval:         100 * time.Millisecond,
		},
		Throttle: ThrottleConfig{
			MinRate:             10,    // 10 msg/sec
			MaxRate:             10000, // 10000 msg/sec  
			AdjustmentFactor:    0.2,
			AdjustmentInterval:  500 * time.Millisecond,
			RecoveryRate:        0.1,
			SystemLoadThreshold: 0.8,
			AdaptiveMode:        true,
		},
		Buffer: BufferConfig{
			InitialSize:       1024 * 64,  // 64KB
			MinSize:           1024 * 16,  // 16KB
			MaxSize:           1024 * 1024, // 1MB
			GrowthFactor:      1.5,
			ShrinkFactor:      0.8,
			ResizeInterval:    2 * time.Second,
			UtilizationWindow: 10,
			MemoryLimit:       1024 * 1024 * 10, // 10MB per buffer
		},
		Monitor: MonitorConfig{
			MonitorInterval:     100 * time.Millisecond,
			MetricsWindow:       100,
			AlertThreshold:      0.9,
			EnablePrometheus:    false,
			EnableLogging:       true,
			StatsReportInterval: 10 * time.Second,
		},
		DropPolicy:          DropLowestPriority,
		GlobalMemoryLimit:   1024 * 1024 * 100, // 100MB
		SystemLoadThreshold: 0.8,
	}
}

// ProductionFlowControlConfig 프로덕션 환경 설정
func ProductionFlowControlConfig() *FlowControlConfig {
	config := DefaultFlowControlConfig()
	
	// 프로덕션 환경을 위한 조정
	config.Thresholds.ProcessingRateMin = 1024 * 50     // 50KB/s
	config.Throttle.AdjustmentInterval = 1 * time.Second
	config.Buffer.ResizeInterval = 5 * time.Second
	config.Monitor.EnablePrometheus = true
	config.GlobalMemoryLimit = 1024 * 1024 * 500 // 500MB
	
	return config
}

// DevelopmentFlowControlConfig 개발 환경 설정
func DevelopmentFlowControlConfig() *FlowControlConfig {
	config := DefaultFlowControlConfig()
	
	// 개발 환경을 위한 조정
	config.Monitor.EnableLogging = true
	config.Monitor.MonitorInterval = 50 * time.Millisecond
	config.Throttle.AdjustmentInterval = 200 * time.Millisecond
	
	return config
}