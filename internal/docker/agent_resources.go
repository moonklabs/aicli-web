package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-units"
)

// Docker API 호환성을 위한 간단한 구조체들
type WeightDevice struct {
	Path   string `json:"Path"`
	Weight uint16 `json:"Weight"`
}

type ThrottleDevice struct {
	Path string `json:"Path"`
	Rate uint64 `json:"Rate"`
}

// AgentResourceManager 에이전트 리소스 제한 관리자
type AgentResourceManager struct {
	client *Client
}

// AgentResourceConfig 에이전트 리소스 설정
type AgentResourceConfig struct {
	// CPU 제한
	CPUQuota     int64   `json:"cpu_quota"`      // CPU 할당량 (100000 = 1 CPU)
	CPUPeriod    uint64  `json:"cpu_period"`     // CPU 기간 (기본값: 100000)
	CPUShares    int64   `json:"cpu_shares"`     // CPU 우선순위 (기본값: 1024)
	CPUSetCpus   string  `json:"cpuset_cpus"`    // 사용할 CPU 코어 지정
	CPUSetMems   string  `json:"cpuset_mems"`    // 사용할 메모리 노드 지정

	// 메모리 제한
	Memory           int64 `json:"memory"`              // 메모리 제한 (bytes)
	MemorySwap       int64 `json:"memory_swap"`         // 스왑 제한 (bytes)
	MemoryReservation int64 `json:"memory_reservation"` // 메모리 예약 (bytes)
	KernelMemory     int64 `json:"kernel_memory"`       // 커널 메모리 제한

	// 디스크 I/O 제한
	BlkioWeight              uint16                           `json:"blkio_weight"`                // I/O 우선순위 (10-1000)
	BlkioWeightDevice        []*WeightDevice                  `json:"blkio_weight_device"`         // 디바이스별 가중치
	BlkioDeviceReadBps       []*ThrottleDevice                `json:"blkio_device_read_bps"`       // 읽기 대역폭 제한
	BlkioDeviceWriteBps      []*ThrottleDevice                `json:"blkio_device_write_bps"`      // 쓰기 대역폭 제한
	BlkioDeviceReadIOps      []*ThrottleDevice                `json:"blkio_device_read_iops"`      // 읽기 IOPS 제한
	BlkioDeviceWriteIOps     []*ThrottleDevice                `json:"blkio_device_write_iops"`     // 쓰기 IOPS 제한

	// 프로세스 제한
	PidsLimit *int64 `json:"pids_limit"` // 프로세스 수 제한

	// Ulimits
	Ulimits []units.Ulimit `json:"ulimits"`

	// OOM 제어
	OomKillDisable *bool `json:"oom_kill_disable"` // OOM Killer 비활성화
	OomScoreAdj    int   `json:"oom_score_adj"`    // OOM Score 조정
}

// ResourceTier 리소스 등급
type ResourceTier string

const (
	ResourceTierMicro  ResourceTier = "micro"  // 0.25 CPU, 512MB RAM
	ResourceTierSmall  ResourceTier = "small"  // 0.5 CPU, 1GB RAM
	ResourceTierMedium ResourceTier = "medium" // 1 CPU, 2GB RAM
	ResourceTierLarge  ResourceTier = "large"  // 2 CPU, 4GB RAM
	ResourceTierXLarge ResourceTier = "xlarge" // 4 CPU, 8GB RAM
)

// ResourceUsage 현재 리소스 사용량
type ResourceUsage struct {
	ContainerID string                 `json:"container_id"`
	CPU         CPUUsage               `json:"cpu"`
	Memory      MemoryUsage            `json:"memory"`
	BlkIO       BlkIOUsage             `json:"blkio"`
	Network     NetworkUsage           `json:"network"`
	PIDs        PIDsUsage              `json:"pids"`
	Timestamp   time.Time              `json:"timestamp"`
	Limits      AgentResourceConfig    `json:"limits"`
}

type CPUUsage struct {
	UsagePercent     float64 `json:"usage_percent"`
	TotalUsage       uint64  `json:"total_usage"`
	SystemUsage      uint64  `json:"system_usage"`
	OnlineCPUs       uint32  `json:"online_cpus"`
	ThrottlingData   ThrottlingData `json:"throttling_data"`
}


type MemoryUsage struct {
	Usage     uint64  `json:"usage"`
	MaxUsage  uint64  `json:"max_usage"`
	Limit     uint64  `json:"limit"`
	UsagePercent float64 `json:"usage_percent"`
	Cache     uint64  `json:"cache"`
	RSS       uint64  `json:"rss"`
	Swap      uint64  `json:"swap"`
}

type BlkIOUsage struct {
	ReadBytes  uint64 `json:"read_bytes"`
	WriteBytes uint64 `json:"write_bytes"`
	ReadIOPS   uint64 `json:"read_iops"`
	WriteIOPS  uint64 `json:"write_iops"`
}

type NetworkUsage struct {
	RxBytes   uint64 `json:"rx_bytes"`
	RxPackets uint64 `json:"rx_packets"`
	RxErrors  uint64 `json:"rx_errors"`
	RxDropped uint64 `json:"rx_dropped"`
	TxBytes   uint64 `json:"tx_bytes"`
	TxPackets uint64 `json:"tx_packets"`
	TxErrors  uint64 `json:"tx_errors"`
	TxDropped uint64 `json:"tx_dropped"`
}

type PIDsUsage struct {
	Current uint64 `json:"current"`
	Limit   uint64 `json:"limit"`
}

// NewAgentResourceManager 새로운 에이전트 리소스 매니저 생성
func NewAgentResourceManager(client *Client) *AgentResourceManager {
	return &AgentResourceManager{
		client: client,
	}
}

// GetPresetConfig 사전 정의된 리소스 설정 조회
func (arm *AgentResourceManager) GetPresetConfig(tier ResourceTier) AgentResourceConfig {
	var config AgentResourceConfig

	switch tier {
	case ResourceTierMicro:
		config = AgentResourceConfig{
			CPUQuota:          25000,   // 0.25 CPU
			CPUPeriod:         100000,
			CPUShares:         256,     // 기본값의 1/4
			Memory:            512 * 1024 * 1024, // 512MB
			MemorySwap:        512 * 1024 * 1024, // 스왑 = 메모리
			MemoryReservation: 256 * 1024 * 1024, // 예약 메모리
			BlkioWeight:       100,     // 낮은 I/O 우선순위
			PidsLimit:         int64Ptr(500),
		}
	case ResourceTierSmall:
		config = AgentResourceConfig{
			CPUQuota:          50000,   // 0.5 CPU
			CPUPeriod:         100000,
			CPUShares:         512,     // 기본값의 1/2
			Memory:            1024 * 1024 * 1024, // 1GB
			MemorySwap:        1024 * 1024 * 1024,
			MemoryReservation: 512 * 1024 * 1024,
			BlkioWeight:       200,
			PidsLimit:         int64Ptr(1000),
		}
	case ResourceTierMedium:
		config = AgentResourceConfig{
			CPUQuota:          100000,  // 1 CPU
			CPUPeriod:         100000,
			CPUShares:         1024,    // 기본값
			Memory:            2 * 1024 * 1024 * 1024, // 2GB
			MemorySwap:        2 * 1024 * 1024 * 1024,
			MemoryReservation: 1024 * 1024 * 1024,
			BlkioWeight:       500,     // 기본 I/O 우선순위
			PidsLimit:         int64Ptr(2000),
		}
	case ResourceTierLarge:
		config = AgentResourceConfig{
			CPUQuota:          200000,  // 2 CPU
			CPUPeriod:         100000,
			CPUShares:         2048,    // 기본값의 2배
			Memory:            4 * 1024 * 1024 * 1024, // 4GB
			MemorySwap:        4 * 1024 * 1024 * 1024,
			MemoryReservation: 2 * 1024 * 1024 * 1024,
			BlkioWeight:       750,
			PidsLimit:         int64Ptr(4000),
		}
	case ResourceTierXLarge:
		config = AgentResourceConfig{
			CPUQuota:          400000,  // 4 CPU
			CPUPeriod:         100000,
			CPUShares:         4096,    // 기본값의 4배
			Memory:            8 * 1024 * 1024 * 1024, // 8GB
			MemorySwap:        8 * 1024 * 1024 * 1024,
			MemoryReservation: 4 * 1024 * 1024 * 1024,
			BlkioWeight:       1000,    // 최고 I/O 우선순위
			PidsLimit:         int64Ptr(8000),
		}
	default:
		// 기본값은 Medium
		return arm.GetPresetConfig(ResourceTierMedium)
	}

	// 공통 설정
	config.Ulimits = []units.Ulimit{
		{Name: "nofile", Soft: 65536, Hard: 65536}, // 파일 디스크립터
		{Name: "nproc", Soft: 2048, Hard: 2048},    // 프로세스 수
		{Name: "memlock", Soft: -1, Hard: -1},      // 메모리 락
	}

	config.OomKillDisable = boolPtr(false)
	config.OomScoreAdj = 0

	return config
}

// ApplyResourceLimits 컨테이너에 리소스 제한 적용
func (arm *AgentResourceManager) ApplyResourceLimits(config *container.Config, hostConfig *container.HostConfig, resourceConfig AgentResourceConfig) error {
	if hostConfig.Resources.CPUQuota != 0 || hostConfig.Resources.CPUPeriod != 0 {
		return fmt.Errorf("resource limits already applied")
	}

	// CPU 제한 설정
	hostConfig.Resources.CPUQuota = resourceConfig.CPUQuota
	hostConfig.Resources.CPUPeriod = int64(resourceConfig.CPUPeriod)
	hostConfig.Resources.CPUShares = resourceConfig.CPUShares
	hostConfig.Resources.CpusetCpus = resourceConfig.CPUSetCpus
	hostConfig.Resources.CpusetMems = resourceConfig.CPUSetMems

	// 메모리 제한 설정
	hostConfig.Resources.Memory = resourceConfig.Memory
	hostConfig.Resources.MemorySwap = resourceConfig.MemorySwap
	hostConfig.Resources.MemoryReservation = resourceConfig.MemoryReservation
	hostConfig.Resources.KernelMemory = resourceConfig.KernelMemory

	// 블록 I/O 제한 설정
	hostConfig.Resources.BlkioWeight = resourceConfig.BlkioWeight
	
	// Docker API 호환성 문제로 Blkio 디바이스 설정은 임시로 생략
	// TODO: Docker API 업데이트 후 블록 I/O 디바이스 설정 구현
	_ = resourceConfig.BlkioWeightDevice
	_ = resourceConfig.BlkioDeviceReadBps
	_ = resourceConfig.BlkioDeviceWriteBps
	_ = resourceConfig.BlkioDeviceReadIOps
	_ = resourceConfig.BlkioDeviceWriteIOps

	// 프로세스 제한 설정
	hostConfig.Resources.PidsLimit = resourceConfig.PidsLimit

	// Ulimits 설정
	if len(resourceConfig.Ulimits) > 0 {
		ulimits := make([]*units.Ulimit, len(resourceConfig.Ulimits))
		for i, ulimit := range resourceConfig.Ulimits {
			ulimits[i] = &ulimit
		}
		hostConfig.Resources.Ulimits = ulimits
	}

	// OOM 제어 설정
	hostConfig.Resources.OomKillDisable = resourceConfig.OomKillDisable
	// TODO: Docker API 호환성 문제로 OomScoreAdj 설정 임시 생략
	_ = resourceConfig.OomScoreAdj

	return nil
}

// GetContainerResources 컨테이너의 현재 리소스 사용량 조회
func (arm *AgentResourceManager) GetContainerResources(ctx context.Context, containerID string) (*ResourceUsage, error) {
	// 컨테이너 통계 조회
	stats, err := arm.client.cli.ContainerStats(ctx, containerID, false)
	if err != nil {
		return nil, fmt.Errorf("get container stats: %w", err)
	}
	defer stats.Body.Close()

	// 통계 데이터 파싱
	var containerStats ContainerStatsData
	if err := json.NewDecoder(stats.Body).Decode(&containerStats); err != nil {
		return nil, fmt.Errorf("decode stats: %w", err)
	}

	// 컨테이너 정보 조회 (리소스 제한 정보)
	inspect, err := arm.client.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("inspect container: %w", err)
	}

	usage := &ResourceUsage{
		ContainerID: containerID,
		Timestamp:   time.Now(),
		Limits:      arm.extractResourceConfig(inspect.HostConfig.Resources),
	}

	// CPU 사용량 계산
	usage.CPU = arm.calculateCPUUsage(containerStats.CPUStats, containerStats.PreCPUStats)

	// 메모리 사용량 계산
	usage.Memory = arm.calculateMemoryUsage(containerStats.MemoryStats)

	// 블록 I/O 사용량 계산
	usage.BlkIO = arm.calculateBlkIOUsage(containerStats.BlkioStats)

	// 네트워크 사용량 계산
	usage.Network = arm.calculateNetworkUsage(containerStats.Networks)

	// PIDs 사용량 계산
	usage.PIDs = arm.calculatePIDsUsage(containerStats.PidsStats)

	return usage, nil
}

// calculateCPUUsage CPU 사용량 계산 - 임시로 기본 구현
func (arm *AgentResourceManager) calculateCPUUsage(cpuStats, preCPUStats interface{}) CPUUsage {
	// TODO: Docker API 호환성 문제로 임시 구현
	return CPUUsage{
		UsagePercent: 0.0,
		TotalUsage:   0,
		SystemUsage:  0,
		OnlineCPUs:   1,
	}
}

// calculateMemoryUsage 메모리 사용량 계산 - 임시로 기본 구현
func (arm *AgentResourceManager) calculateMemoryUsage(memStats interface{}) MemoryUsage {
	// TODO: Docker API 호환성 문제로 임시 구현
	return MemoryUsage{
		Usage:        0,
		MaxUsage:     0,
		Limit:        0,
		UsagePercent: 0.0,
		Cache:        0,
		RSS:          0,
		Swap:         0,
	}
}

// calculateBlkIOUsage 블록 I/O 사용량 계산 - 임시로 기본 구현
func (arm *AgentResourceManager) calculateBlkIOUsage(blkioStats interface{}) BlkIOUsage {
	// TODO: Docker API 호환성 문제로 임시 구현
	return BlkIOUsage{
		ReadBytes:  0,
		WriteBytes: 0,
		ReadIOPS:   0,
		WriteIOPS:  0,
	}
}

// calculateNetworkUsage 네트워크 사용량 계산 - 임시로 기본 구현
func (arm *AgentResourceManager) calculateNetworkUsage(networks interface{}) NetworkUsage {
	// TODO: Docker API 호환성 문제로 임시 구현
	return NetworkUsage{
		RxBytes:   0,
		RxPackets: 0,
		RxErrors:  0,
		RxDropped: 0,
		TxBytes:   0,
		TxPackets: 0,
		TxErrors:  0,
		TxDropped: 0,
	}
}

// calculatePIDsUsage PIDs 사용량 계산 - 임시로 기본 구현
func (arm *AgentResourceManager) calculatePIDsUsage(pidsStats interface{}) PIDsUsage {
	// TODO: Docker API 호환성 문제로 임시 구현
	return PIDsUsage{
		Current: 0,
		Limit:   0,
	}
}

// extractResourceConfig 컨테이너 리소스 설정 추출 - 임시로 기본 구현
func (arm *AgentResourceManager) extractResourceConfig(resources interface{}) AgentResourceConfig {
	// TODO: Docker API 호환성 문제로 임시 구현
	return AgentResourceConfig{
		CPUQuota:          0,
		CPUPeriod:         100000,
		CPUShares:         1024,
		Memory:            0,
		MemorySwap:        0,
		MemoryReservation: 0,
		KernelMemory:      0,
		BlkioWeight:       0,
	}
}

// ValidateResourceConfig 리소스 설정 유효성 검사
func (arm *AgentResourceManager) ValidateResourceConfig(config AgentResourceConfig) error {
	// CPU 설정 검증
	if config.CPUQuota > 0 && config.CPUPeriod == 0 {
		return fmt.Errorf("CPU period must be set when CPU quota is specified")
	}

	if config.CPUPeriod > 0 && (config.CPUPeriod < 1000 || config.CPUPeriod > 1000000) {
		return fmt.Errorf("CPU period must be between 1000 and 1000000")
	}

	if config.CPUShares > 0 && (config.CPUShares < 2 || config.CPUShares > 262144) {
		return fmt.Errorf("CPU shares must be between 2 and 262144")
	}

	// 메모리 설정 검증
	if config.Memory > 0 && config.Memory < 4*1024*1024 {
		return fmt.Errorf("memory limit must be at least 4MB")
	}

	if config.MemorySwap > 0 && config.MemorySwap < config.Memory {
		return fmt.Errorf("memory swap must be greater than or equal to memory limit")
	}

	// 블록 I/O 설정 검증
	if config.BlkioWeight > 0 && (config.BlkioWeight < 10 || config.BlkioWeight > 1000) {
		return fmt.Errorf("block I/O weight must be between 10 and 1000")
	}

	return nil
}

// ParseResourceString 리소스 문자열 파싱 (예: "1.5", "2g", "1024m")
func (arm *AgentResourceManager) ParseResourceString(resourceType, value string) (int64, error) {
	switch resourceType {
	case "cpu":
		// CPU 값을 쿼터로 변환 (1.0 = 100000)
		if cpuFloat, err := strconv.ParseFloat(value, 64); err == nil {
			return int64(cpuFloat * 100000), nil
		}
		return 0, fmt.Errorf("invalid CPU format: %s", value)

	case "memory":
		// 메모리 값을 바이트로 변환
		if strings.HasSuffix(value, "g") || strings.HasSuffix(value, "G") {
			if val, err := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSuffix(value, "G"), "g"), 64); err == nil {
				return int64(val * 1024 * 1024 * 1024), nil
			}
		} else if strings.HasSuffix(value, "m") || strings.HasSuffix(value, "M") {
			if val, err := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSuffix(value, "M"), "m"), 64); err == nil {
				return int64(val * 1024 * 1024), nil
			}
		} else if strings.HasSuffix(value, "k") || strings.HasSuffix(value, "K") {
			if val, err := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSuffix(value, "K"), "k"), 64); err == nil {
				return int64(val * 1024), nil
			}
		} else {
			// 바이트 단위
			return strconv.ParseInt(value, 10, 64)
		}
		return 0, fmt.Errorf("invalid memory format: %s", value)

	default:
		return 0, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}

// 헬퍼 함수들
func int64Ptr(v int64) *int64 {
	return &v
}

func boolPtr(v bool) *bool {
	return &v
}