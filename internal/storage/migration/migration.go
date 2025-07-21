package migration

import (
	"context"
	"fmt"
	"time"
)

// Direction 마이그레이션 방향
type Direction string

const (
	// DirectionUp 업그레이드 마이그레이션
	DirectionUp Direction = "up"
	
	// DirectionDown 다운그레이드 마이그레이션
	DirectionDown Direction = "down"
)

// MigrationStatus 마이그레이션 상태
type MigrationStatus string

const (
	// MigrationStatusPending 대기 중
	MigrationStatusPending MigrationStatus = "pending"
	
	// MigrationStatusRunning 실행 중
	MigrationStatusRunning MigrationStatus = "running"
	
	// MigrationStatusCompleted 완료됨
	MigrationStatusCompleted MigrationStatus = "completed"
	
	// MigrationStatusFailed 실패함
	MigrationStatusFailed MigrationStatus = "failed"
	
	// MigrationStatusRolledBack 롤백됨
	MigrationStatusRolledBack MigrationStatus = "rolled_back"
)

// Migration 마이그레이션 인터페이스
type Migration interface {
	// Version 마이그레이션 버전 반환
	Version() string
	
	// Description 마이그레이션 설명 반환
	Description() string
	
	// Up 업그레이드 마이그레이션 실행
	Up(ctx context.Context, db interface{}) error
	
	// Down 다운그레이드 마이그레이션 실행
	Down(ctx context.Context, db interface{}) error
	
	// CanRollback 롤백 가능 여부
	CanRollback() bool
}

// MigrationInfo 마이그레이션 정보
type MigrationInfo struct {
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Status      MigrationStatus   `json:"status"`
	AppliedAt   *time.Time        `json:"applied_at,omitempty"`
	RolledBackAt *time.Time       `json:"rolled_back_at,omitempty"`
	Duration    time.Duration     `json:"duration"`
	Error       string            `json:"error,omitempty"`
	CanRollback bool              `json:"can_rollback"`
}

// MigrationRecord 마이그레이션 실행 기록
type MigrationRecord struct {
	Version     string            `json:"version"`
	Direction   Direction         `json:"direction"`
	Status      MigrationStatus   `json:"status"`
	StartedAt   time.Time         `json:"started_at"`
	CompletedAt *time.Time        `json:"completed_at,omitempty"`
	Duration    time.Duration     `json:"duration"`
	Error       string            `json:"error,omitempty"`
	Checksum    string            `json:"checksum,omitempty"`
}

// Migrator 마이그레이션 실행기 인터페이스
type Migrator interface {
	// Current 현재 스키마 버전 반환
	Current() (string, error)
	
	// Migrate 특정 버전으로 마이그레이션
	Migrate(ctx context.Context, target string) error
	
	// MigrateUp 최신 버전으로 업그레이드
	MigrateUp(ctx context.Context) error
	
	// Rollback N단계 롤백
	Rollback(ctx context.Context, steps int) error
	
	// RollbackTo 특정 버전으로 롤백
	RollbackTo(ctx context.Context, target string) error
	
	// List 모든 마이그레이션 목록 반환
	List() ([]MigrationInfo, error)
	
	// Status 현재 마이그레이션 상태 반환
	Status() (*MigrationStatus, []MigrationInfo, error)
	
	// Validate 마이그레이션 파일 검증
	Validate() error
	
	// Close 마이그레이션 리소스 정리
	Close() error
}

// MigrationSource 마이그레이션 소스 인터페이스
type MigrationSource interface {
	// Load 마이그레이션 파일들을 로드
	Load() ([]Migration, error)
	
	// Get 특정 버전의 마이그레이션 반환
	Get(version string) (Migration, error)
	
	// List 사용 가능한 모든 마이그레이션 버전 나열
	List() ([]string, error)
}

// MigrationTracker 마이그레이션 상태 추적기 인터페이스
type MigrationTracker interface {
	// EnsureTable 마이그레이션 테이블/버킷 생성
	EnsureTable(ctx context.Context) error
	
	// GetApplied 적용된 마이그레이션 목록 반환
	GetApplied(ctx context.Context) ([]MigrationRecord, error)
	
	// IsApplied 특정 버전이 적용되었는지 확인
	IsApplied(ctx context.Context, version string) (bool, error)
	
	// RecordUp 업그레이드 마이그레이션 기록
	RecordUp(ctx context.Context, version string, duration time.Duration) error
	
	// RecordDown 다운그레이드 마이그레이션 기록
	RecordDown(ctx context.Context, version string, duration time.Duration) error
	
	// RecordFailed 실패한 마이그레이션 기록
	RecordFailed(ctx context.Context, version string, direction Direction, err error) error
	
	// RemoveRecord 마이그레이션 기록 제거
	RemoveRecord(ctx context.Context, version string) error
}

// BaseMigration 기본 마이그레이션 구조체
type BaseMigration struct {
	version     string
	description string
	canRollback bool
}

// NewBaseMigration 기본 마이그레이션 생성
func NewBaseMigration(version, description string, canRollback bool) *BaseMigration {
	return &BaseMigration{
		version:     version,
		description: description,
		canRollback: canRollback,
	}
}

// Version 버전 반환
func (m *BaseMigration) Version() string {
	return m.version
}

// Description 설명 반환
func (m *BaseMigration) Description() string {
	return m.description
}

// CanRollback 롤백 가능 여부 반환
func (m *BaseMigration) CanRollback() bool {
	return m.canRollback
}

// Up 업그레이드 (기본 구현 - 오버라이드 필요)
func (m *BaseMigration) Up(ctx context.Context, db interface{}) error {
	return fmt.Errorf("Up 메서드가 구현되지 않았습니다: %s", m.version)
}

// Down 다운그레이드 (기본 구현 - 오버라이드 필요)
func (m *BaseMigration) Down(ctx context.Context, db interface{}) error {
	if !m.canRollback {
		return fmt.Errorf("마이그레이션 %s는 롤백을 지원하지 않습니다", m.version)
	}
	return fmt.Errorf("Down 메서드가 구현되지 않았습니다: %s", m.version)
}

// MigrationOptions 마이그레이션 옵션
type MigrationOptions struct {
	// DryRun 실제 실행하지 않고 시뮬레이션만
	DryRun bool
	
	// Force 에러가 있어도 강제 실행
	Force bool
	
	// Verbose 상세 로그 출력
	Verbose bool
	
	// Timeout 마이그레이션 타임아웃
	Timeout time.Duration
	
	// IgnoreUnknown 알 수 없는 마이그레이션 무시
	IgnoreUnknown bool
}

// DefaultMigrationOptions 기본 마이그레이션 옵션
func DefaultMigrationOptions() MigrationOptions {
	return MigrationOptions{
		DryRun:        false,
		Force:         false,
		Verbose:       false,
		Timeout:       time.Minute * 10,
		IgnoreUnknown: false,
	}
}

// MigrationEvent 마이그레이션 이벤트
type MigrationEvent struct {
	Type      string    `json:"type"`      // start, progress, complete, error
	Version   string    `json:"version"`
	Direction Direction `json:"direction"`
	Message   string    `json:"message"`
	Error     error     `json:"error,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// MigrationEventHandler 마이그레이션 이벤트 핸들러
type MigrationEventHandler func(event MigrationEvent)

// MigrationError 마이그레이션 에러
type MigrationError struct {
	Version   string
	Direction Direction
	Cause     error
	Rollback  bool
}

// Error 에러 메시지 반환
func (e *MigrationError) Error() string {
	action := "실행"
	if e.Direction == DirectionDown {
		action = "롤백"
	}
	
	message := fmt.Sprintf("마이그레이션 %s %s 중 에러 발생: %v", e.Version, action, e.Cause)
	
	if e.Rollback {
		message += " (자동 롤백됨)"
	}
	
	return message
}

// Unwrap 원본 에러 반환
func (e *MigrationError) Unwrap() error {
	return e.Cause
}

// NewMigrationError 새 마이그레이션 에러 생성
func NewMigrationError(version string, direction Direction, cause error, rollback bool) *MigrationError {
	return &MigrationError{
		Version:   version,
		Direction: direction,
		Cause:     cause,
		Rollback:  rollback,
	}
}

// ValidateVersion 마이그레이션 버전 형식 검증
func ValidateVersion(version string) error {
	if len(version) == 0 {
		return fmt.Errorf("버전이 비어있습니다")
	}
	
	// 버전 형식 검증 (예: 001, 002, 003)
	if len(version) < 3 {
		return fmt.Errorf("버전 형식이 잘못되었습니다. 최소 3자리 숫자여야 합니다: %s", version)
	}
	
	for _, char := range version {
		if char < '0' || char > '9' {
			return fmt.Errorf("버전은 숫자만 포함해야 합니다: %s", version)
		}
	}
	
	return nil
}

// CompareVersions 버전 비교 (v1 < v2: -1, v1 == v2: 0, v1 > v2: 1)
func CompareVersions(v1, v2 string) int {
	if v1 == v2 {
		return 0
	}
	
	// 숫자 비교
	for i := 0; i < len(v1) && i < len(v2); i++ {
		if v1[i] < v2[i] {
			return -1
		}
		if v1[i] > v2[i] {
			return 1
		}
	}
	
	// 길이 비교
	if len(v1) < len(v2) {
		return -1
	}
	if len(v1) > len(v2) {
		return 1
	}
	
	return 0
}

// SortVersions 버전들을 오름차순으로 정렬
func SortVersions(versions []string) []string {
	sorted := make([]string, len(versions))
	copy(sorted, versions)
	
	// 간단한 버블 정렬
	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if CompareVersions(sorted[j], sorted[j+1]) > 0 {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}
	
	return sorted
}