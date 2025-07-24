package storage

import (
	"context"
	"fmt"
	"time"
	
	"github.com/aicli/aicli-web/internal/models"
)

// StorageType 스토리지 백엔드 타입 정의
type StorageType string

const (
	// StorageTypeMemory 메모리 기반 스토리지
	StorageTypeMemory StorageType = "memory"
	
	// StorageTypeSQLite SQLite 기반 스토리지
	StorageTypeSQLite StorageType = "sqlite"
	
	// StorageTypeBoltDB BoltDB 기반 스토리지
	StorageTypeBoltDB StorageType = "boltdb"
)

// StorageConfig 스토리지 설정 구조체
type StorageConfig struct {
	// Type 스토리지 타입
	Type StorageType `yaml:"type" mapstructure:"type" json:"type"`
	
	// DataSource 데이터 소스 (파일 경로 또는 연결 문자열)
	DataSource string `yaml:"data_source" mapstructure:"data_source" json:"data_source"`
	
	// MaxConns 최대 연결 수 (SQLite에서 사용)
	MaxConns int `yaml:"max_conns" mapstructure:"max_conns" json:"max_conns"`
	
	// MaxIdleConns 최대 유휴 연결 수
	MaxIdleConns int `yaml:"max_idle_conns" mapstructure:"max_idle_conns" json:"max_idle_conns"`
	
	// ConnMaxLifetime 연결 최대 생명 시간
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" mapstructure:"conn_max_lifetime" json:"conn_max_lifetime"`
	
	// Timeout 연결 타임아웃
	Timeout time.Duration `yaml:"timeout" mapstructure:"timeout" json:"timeout"`
	
	// RetryCount 재시도 횟수
	RetryCount int `yaml:"retry_count" mapstructure:"retry_count" json:"retry_count"`
	
	// RetryInterval 재시도 간격
	RetryInterval time.Duration `yaml:"retry_interval" mapstructure:"retry_interval" json:"retry_interval"`
}

// DefaultStorageConfig 기본 스토리지 설정 반환
func DefaultStorageConfig() StorageConfig {
	return StorageConfig{
		Type:            StorageTypeMemory,
		DataSource:      "",
		MaxConns:        10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
		Timeout:         time.Second * 30,
		RetryCount:      3,
		RetryInterval:   time.Second,
	}
}

// StorageFactory 스토리지 팩토리 인터페이스
type StorageFactory interface {
	// Create 설정에 따른 스토리지 인스턴스 생성
	Create(config StorageConfig) (Storage, error)
	
	// HealthCheck 스토리지 연결 상태 확인
	HealthCheck(ctx context.Context, storage Storage) error
}

// DefaultStorageFactory 기본 스토리지 팩토리
type DefaultStorageFactory struct{}

// NewDefaultStorageFactory 기본 스토리지 팩토리 생성
func NewDefaultStorageFactory() *DefaultStorageFactory {
	return &DefaultStorageFactory{}
}

// Create 설정에 따른 스토리지 인스턴스 생성
func (f *DefaultStorageFactory) Create(config StorageConfig) (Storage, error) {
	switch config.Type {
	case StorageTypeMemory:
		return f.createMemoryStorage(config)
	case StorageTypeSQLite:
		return f.createSQLiteStorage(config)
	case StorageTypeBoltDB:
		return f.createBoltDBStorage(config)
	default:
		return nil, fmt.Errorf("지원하지 않는 스토리지 타입: %s", config.Type)
	}
}

// createMemoryStorage 메모리 스토리지 생성
func (f *DefaultStorageFactory) createMemoryStorage(config StorageConfig) (Storage, error) {
	// 메모리 스토리지는 순환 의존성 회피를 위해 여기서 직접 생성하지 않습니다
	// new.go의 NewMemory() 함수를 사용하세요
	return nil, fmt.Errorf("메모리 스토리지는 storage.NewMemory()를 사용하세요")
}

// createSQLiteStorage SQLite 스토리지 생성
func (f *DefaultStorageFactory) createSQLiteStorage(config StorageConfig) (Storage, error) {
	// SQLite 스토리지는 순환 의존성 회피를 위해 여기서 직접 생성하지 않습니다
	return nil, fmt.Errorf("SQLite 스토리지는 아직 구현되지 않았습니다")
}

// createBoltDBStorage BoltDB 스토리지 생성  
func (f *DefaultStorageFactory) createBoltDBStorage(config StorageConfig) (Storage, error) {
	// BoltDB 스토리지는 순환 의존성 회피를 위해 여기서 직접 생성하지 않습니다
	return nil, fmt.Errorf("BoltDB 스토리지는 아직 구현되지 않았습니다")
}

// HealthCheck 스토리지 연결 상태 확인
func (f *DefaultStorageFactory) HealthCheck(ctx context.Context, storage Storage) error {
	if storage == nil {
		return fmt.Errorf("스토리지가 nil입니다")
	}
	
	// 간단한 헬스체크: 워크스페이스 목록 조회 시도
	_, _, err := storage.Workspace().List(ctx, &models.PaginationRequest{
		Page:  1,
		Limit: 1,
	})
	
	if err != nil {
		return fmt.Errorf("스토리지 헬스체크 실패: %w", err)
	}
	
	return nil
}

// ValidateStorageConfig 스토리지 설정 검증
func ValidateStorageConfig(config StorageConfig) error {
	if config.Type == "" {
		return fmt.Errorf("스토리지 타입이 지정되지 않았습니다")
	}
	
	validTypes := []StorageType{StorageTypeMemory, StorageTypeSQLite, StorageTypeBoltDB}
	isValidType := false
	for _, validType := range validTypes {
		if config.Type == validType {
			isValidType = true
			break
		}
	}
	
	if !isValidType {
		return fmt.Errorf("지원하지 않는 스토리지 타입: %s", config.Type)
	}
	
	// 파일 기반 스토리지는 데이터 소스가 필요
	if (config.Type == StorageTypeSQLite || config.Type == StorageTypeBoltDB) && config.DataSource == "" {
		return fmt.Errorf("%s 스토리지는 데이터 소스가 필요합니다", config.Type)
	}
	
	if config.MaxConns < 1 {
		return fmt.Errorf("최대 연결 수는 1 이상이어야 합니다")
	}
	
	if config.MaxIdleConns > config.MaxConns {
		return fmt.Errorf("최대 유휴 연결 수는 최대 연결 수보다 클 수 없습니다")
	}
	
	if config.Timeout <= 0 {
		return fmt.Errorf("연결 타임아웃은 0보다 커야 합니다")
	}
	
	if config.RetryCount < 0 {
		return fmt.Errorf("재시도 횟수는 0 이상이어야 합니다")
	}
	
	if config.RetryInterval <= 0 {
		return fmt.Errorf("재시도 간격은 0보다 커야 합니다")
	}
	
	return nil
}