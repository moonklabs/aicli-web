package models

import "time"

// CreateAgentRequest 에이전트 생성 요청
type CreateAgentRequest struct {
	Name      string      `json:"name" validate:"required,min=1,max=100"`
	Type      AgentType   `json:"type" validate:"required,oneof=claude gemini custom"`
	ProjectID string      `json:"project_id" validate:"required,uuid"`
	Config    AgentConfig `json:"config" validate:"required"`
}

// UpdateAgentRequest 에이전트 업데이트 요청
type UpdateAgentRequest struct {
	Name   *string      `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Config *AgentConfig `json:"config,omitempty"`
}

// AgentListResponse 에이전트 목록 응답
type AgentListResponse struct {
	Agents     []*Agent            `json:"agents"`
	Pagination *PaginationResponse `json:"pagination,omitempty"`
	Metadata   *AgentListMetadata  `json:"metadata,omitempty"`
}

// AgentListMetadata 에이전트 목록 메타데이터
type AgentListMetadata struct {
	TotalCount   int                 `json:"total_count"`
	ActiveCount  int                 `json:"active_count"`
	StatusCounts map[AgentStatus]int `json:"status_counts"`
	TypeCounts   map[AgentType]int   `json:"type_counts"`
}

// AgentHealth 에이전트 헬스 상태
type AgentHealth struct {
	AgentID   string                 `json:"agent_id"`
	Status    string                 `json:"status"` // healthy, unhealthy, unknown
	LastCheck time.Time              `json:"last_check"`
	Checks    map[string]HealthCheck `json:"checks"`
	Uptime    time.Duration          `json:"uptime"`
}

// HealthCheck 개별 헬스 체크
type HealthCheck struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"` // pass, fail, warn
	Message   string    `json:"message,omitempty"`
	LastCheck time.Time `json:"last_check"`
	Duration  string    `json:"duration,omitempty"`
}

// AgentMetrics 에이전트 메트릭
type AgentMetrics struct {
	AgentID     string    `json:"agent_id"`
	CollectedAt time.Time `json:"collected_at"`

	// 컨테이너 리소스 메트릭
	CPU     *ResourceMetric `json:"cpu,omitempty"`
	Memory  *ResourceMetric `json:"memory,omitempty"`
	Network *NetworkMetric  `json:"network,omitempty"`

	// 에이전트 특화 메트릭
	TaskCount     int     `json:"task_count"`
	ActiveTasks   int     `json:"active_tasks"`
	TotalRequests int64   `json:"total_requests"`
	ErrorRate     float64 `json:"error_rate"`

	// 성능 메트릭
	AvgResponseTime time.Duration `json:"avg_response_time"`
	P95ResponseTime time.Duration `json:"p95_response_time"`
}

// ResourceMetric 리소스 메트릭
type ResourceMetric struct {
	Current float64 `json:"current"` // 현재 사용량
	Limit   float64 `json:"limit"`   // 제한값
	Usage   float64 `json:"usage"`   // 사용률 (%)
	Unit    string  `json:"unit"`    // 단위
}

// NetworkMetric 네트워크 메트릭
type NetworkMetric struct {
	BytesReceived   int64 `json:"bytes_received"`
	BytesSent       int64 `json:"bytes_sent"`
	PacketsReceived int64 `json:"packets_received"`
	PacketsSent     int64 `json:"packets_sent"`
	ErrorsReceived  int64 `json:"errors_received"`
	ErrorsSent      int64 `json:"errors_sent"`
}

// AgentLogRequest 에이전트 로그 요청
type AgentLogRequest struct {
	Since  *time.Time `json:"since,omitempty" form:"since"`
	Until  *time.Time `json:"until,omitempty" form:"until"`
	Follow bool       `json:"follow,omitempty" form:"follow"`
	Tail   int        `json:"tail,omitempty" form:"tail" validate:"omitempty,min=1,max=10000"`
	Level  string     `json:"level,omitempty" form:"level" validate:"omitempty,oneof=debug info warn error"`
}

// AgentEventRequest 에이전트 이벤트 스트림 요청
type AgentEventRequest struct {
	Types []string   `json:"types,omitempty" form:"types"` // 이벤트 타입 필터
	Since *time.Time `json:"since,omitempty" form:"since"`
}

// StartAgentRequest 에이전트 시작 요청
type StartAgentRequest struct {
	Force bool `json:"force,omitempty"` // 강제 시작 여부
}

// StopAgentRequest 에이전트 중지 요청
type StopAgentRequest struct {
	Force   bool          `json:"force,omitempty"`   // 강제 중지 여부
	Timeout time.Duration `json:"timeout,omitempty"` // 중지 타임아웃
}

// RestartAgentRequest 에이전트 재시작 요청
type RestartAgentRequest struct {
	Force   bool          `json:"force,omitempty"`   // 강제 재시작 여부
	Timeout time.Duration `json:"timeout,omitempty"` // 재시작 타임아웃
}
