package models

import (
	"time"
)

// SessionStatusActive 상수 (SessionStatus 타입과 연동)
const (
	SessionStatusActive = SessionActive
	SessionStatusIdle   = SessionIdle
	SessionStatusEnded  = SessionEnded
	SessionStatusError  = SessionError
)

// SessionStats 세션 통계 정보
type SessionStats struct {
	TotalCount      int64          `json:"total_count"`
	ActiveCount     int64          `json:"active_count"`
	IdleCount       int64          `json:"idle_count"`
	ErrorCount      int64          `json:"error_count"`
	AverageLifetime time.Duration  `json:"average_lifetime"`
	TotalBytesIn    int64          `json:"total_bytes_in"`
	TotalBytesOut   int64          `json:"total_bytes_out"`
	LastActive      *time.Time     `json:"last_active,omitempty"`
}

// TaskStats 태스크 통계 정보
type TaskStats struct {
	TotalCount       int64          `json:"total_count"`
	RunningCount     int64          `json:"running_count"`
	CompletedCount   int64          `json:"completed_count"`
	FailedCount      int64          `json:"failed_count"`
	AverageDuration  time.Duration  `json:"average_duration"`
	SuccessRate      float64        `json:"success_rate"`
	LastExecuted     *time.Time     `json:"last_executed,omitempty"`
}

// WorkspaceStats 워크스페이스 통계 정보
type WorkspaceStats struct {
	TotalWorkspaces  int64          `json:"total_workspaces"`
	ActiveWorkspaces int64          `json:"active_workspaces"`
	TotalProjects    int64          `json:"total_projects"`
	ActiveProjects   int64          `json:"active_projects"`
	TotalSessions    int64          `json:"total_sessions"`
	ActiveSessions   int64          `json:"active_sessions"`
	StorageUsed      int64          `json:"storage_used"`
	LastUpdated      time.Time      `json:"last_updated"`
}

// UserActivityStats 사용자 활동 통계 정보
type UserActivityStats struct {
	UserID           string         `json:"user_id"`
	WorkspaceCount   int64          `json:"workspace_count"`
	ProjectCount     int64          `json:"project_count"`
	SessionCount     int64          `json:"session_count"`
	TaskCount        int64          `json:"task_count"`
	StorageUsed      int64          `json:"storage_used"`
	LastActive       *time.Time     `json:"last_active,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
}

// SystemStats 시스템 전체 통계
type SystemStats struct {
	Uptime           time.Duration  `json:"uptime"`
	TotalUsers       int64          `json:"total_users"`
	ActiveUsers      int64          `json:"active_users"`
	TotalWorkspaces  int64          `json:"total_workspaces"`
	TotalProjects    int64          `json:"total_projects"`
	TotalSessions    int64          `json:"total_sessions"`
	ActiveSessions   int64          `json:"active_sessions"`
	TotalTasks       int64          `json:"total_tasks"`
	RunningTasks     int64          `json:"running_tasks"`
	CPUUsage         float64        `json:"cpu_usage"`
	MemoryUsage      int64          `json:"memory_usage"`
	DiskUsage        int64          `json:"disk_usage"`
	Timestamp        time.Time      `json:"timestamp"`
}