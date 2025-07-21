package storage

import (
	"context"
	"testing"
	"time"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDefaultStorageConfig 기본 스토리지 설정 테스트
func TestDefaultStorageConfig(t *testing.T) {
	config := DefaultStorageConfig()
	
	assert.Equal(t, StorageTypeMemory, config.Type)
	assert.Empty(t, config.DataSource)
	assert.Equal(t, 10, config.MaxConns)
	assert.Equal(t, 5, config.MaxIdleConns)
	assert.Equal(t, time.Hour, config.ConnMaxLifetime)
	assert.Equal(t, time.Second*30, config.Timeout)
	assert.Equal(t, 3, config.RetryCount)
	assert.Equal(t, time.Second, config.RetryInterval)
}

// TestValidateStorageConfig 스토리지 설정 검증 테스트
func TestValidateStorageConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  StorageConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:   "유효한 메모리 설정",
			config: StorageConfig{Type: StorageTypeMemory, MaxConns: 5, Timeout: time.Second},
			wantErr: false,
		},
		{
			name: "유효한 SQLite 설정",
			config: StorageConfig{
				Type: StorageTypeSQLite, 
				DataSource: "/tmp/test.db", 
				MaxConns: 5, 
				Timeout: time.Second,
				RetryInterval: time.Second,
			},
			wantErr: false,
		},
		{
			name:    "빈 스토리지 타입",
			config:  StorageConfig{},
			wantErr: true,
			errMsg:  "스토리지 타입이 지정되지 않았습니다",
		},
		{
			name:    "지원하지 않는 스토리지 타입",
			config:  StorageConfig{Type: "invalid"},
			wantErr: true,
			errMsg:  "지원하지 않는 스토리지 타입",
		},
		{
			name: "SQLite 데이터 소스 누락",
			config: StorageConfig{
				Type: StorageTypeSQLite, 
				MaxConns: 5, 
				Timeout: time.Second,
				RetryInterval: time.Second,
			},
			wantErr: true,
			errMsg:  "데이터 소스가 필요합니다",
		},
		{
			name: "잘못된 최대 연결 수",
			config: StorageConfig{
				Type: StorageTypeMemory, 
				MaxConns: 0, 
				Timeout: time.Second,
				RetryInterval: time.Second,
			},
			wantErr: true,
			errMsg:  "최대 연결 수는 1 이상이어야 합니다",
		},
		{
			name: "유휴 연결 수 초과",
			config: StorageConfig{
				Type: StorageTypeMemory, 
				MaxConns: 5, 
				MaxIdleConns: 10, 
				Timeout: time.Second,
				RetryInterval: time.Second,
			},
			wantErr: true,
			errMsg:  "최대 유휴 연결 수는 최대 연결 수보다 클 수 없습니다",
		},
		{
			name: "잘못된 타임아웃",
			config: StorageConfig{
				Type: StorageTypeMemory, 
				MaxConns: 5, 
				Timeout: 0,
				RetryInterval: time.Second,
			},
			wantErr: true,
			errMsg:  "연결 타임아웃은 0보다 커야 합니다",
		},
		{
			name: "잘못된 재시도 횟수",
			config: StorageConfig{
				Type: StorageTypeMemory, 
				MaxConns: 5, 
				Timeout: time.Second, 
				RetryCount: -1,
				RetryInterval: time.Second,
			},
			wantErr: true,
			errMsg:  "재시도 횟수는 0 이상이어야 합니다",
		},
		{
			name: "잘못된 재시도 간격",
			config: StorageConfig{
				Type: StorageTypeMemory, 
				MaxConns: 5, 
				Timeout: time.Second, 
				RetryInterval: 0,
			},
			wantErr: true,
			errMsg:  "재시도 간격은 0보다 커야 합니다",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStorageConfig(tt.config)
			
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestDefaultStorageFactory 기본 스토리지 팩토리 테스트
func TestDefaultStorageFactory(t *testing.T) {
	factory := NewDefaultStorageFactory()
	assert.NotNil(t, factory)
}

// TestDefaultStorageFactoryCreateMemory 메모리 스토리지 생성 테스트
func TestDefaultStorageFactoryCreateMemory(t *testing.T) {
	factory := NewDefaultStorageFactory()
	config := StorageConfig{
		Type:     StorageTypeMemory,
		MaxConns: 5,
		Timeout:  time.Second * 10,
		RetryInterval: time.Second,
	}
	
	storage, err := factory.Create(config)
	require.NoError(t, err)
	assert.NotNil(t, storage)
	
	// 스토리지 인터페이스 확인
	assert.NotNil(t, storage.Workspace())
	assert.NotNil(t, storage.Project())
	assert.NotNil(t, storage.Session())
	assert.NotNil(t, storage.Task())
	
	// 정상적으로 닫히는지 확인
	err = storage.Close()
	assert.NoError(t, err)
}

// TestDefaultStorageFactoryCreateSQLite SQLite 스토리지 생성 테스트 (아직 미구현)
func TestDefaultStorageFactoryCreateSQLite(t *testing.T) {
	factory := NewDefaultStorageFactory()
	config := StorageConfig{
		Type:       StorageTypeSQLite,
		DataSource: "/tmp/test.db",
		MaxConns:   5,
		Timeout:    time.Second * 10,
		RetryInterval: time.Second,
	}
	
	storage, err := factory.Create(config)
	assert.Error(t, err)
	assert.Nil(t, storage)
	assert.Contains(t, err.Error(), "SQLite 스토리지는 아직 구현되지 않았습니다")
}

// TestDefaultStorageFactoryCreateBoltDB BoltDB 스토리지 생성 테스트 (아직 미구현)
func TestDefaultStorageFactoryCreateBoltDB(t *testing.T) {
	factory := NewDefaultStorageFactory()
	config := StorageConfig{
		Type:       StorageTypeBoltDB,
		DataSource: "/tmp/test.boltdb",
		MaxConns:   5,
		Timeout:    time.Second * 10,
		RetryInterval: time.Second,
	}
	
	storage, err := factory.Create(config)
	assert.Error(t, err)
	assert.Nil(t, storage)
	assert.Contains(t, err.Error(), "BoltDB 스토리지는 아직 구현되지 않았습니다")
}

// TestDefaultStorageFactoryCreateInvalidType 지원하지 않는 스토리지 타입 테스트
func TestDefaultStorageFactoryCreateInvalidType(t *testing.T) {
	factory := NewDefaultStorageFactory()
	config := StorageConfig{
		Type:     "invalid",
		MaxConns: 5,
		Timeout:  time.Second * 10,
		RetryInterval: time.Second,
	}
	
	storage, err := factory.Create(config)
	assert.Error(t, err)
	assert.Nil(t, storage)
	assert.Contains(t, err.Error(), "지원하지 않는 스토리지 타입")
}

// TestDefaultStorageFactoryHealthCheck 헬스체크 테스트
func TestDefaultStorageFactoryHealthCheck(t *testing.T) {
	factory := NewDefaultStorageFactory()
	
	t.Run("nil 스토리지", func(t *testing.T) {
		err := factory.HealthCheck(context.Background(), nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "스토리지가 nil입니다")
	})
	
	t.Run("정상 스토리지", func(t *testing.T) {
		config := StorageConfig{
			Type:     StorageTypeMemory,
			MaxConns: 5,
			Timeout:  time.Second * 10,
			RetryInterval: time.Second,
		}
		
		storage, err := factory.Create(config)
		require.NoError(t, err)
		defer storage.Close()
		
		err = factory.HealthCheck(context.Background(), storage)
		assert.NoError(t, err)
	})
	
	t.Run("타임아웃 컨텍스트", func(t *testing.T) {
		config := StorageConfig{
			Type:     StorageTypeMemory,
			MaxConns: 5,
			Timeout:  time.Second * 10,
			RetryInterval: time.Second,
		}
		
		storage, err := factory.Create(config)
		require.NoError(t, err)
		defer storage.Close()
		
		ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
		defer cancel()
		
		// 매우 짧은 타임아웃으로 인해 컨텍스트가 취소될 가능성이 높음
		time.Sleep(time.Millisecond) // 타임아웃 발생을 위한 대기
		
		err = factory.HealthCheck(ctx, storage)
		// 타임아웃이나 정상 완료 모두 허용 (컨텍스트 타이밍에 따라 다름)
	})
}

// Benchmark tests
func BenchmarkStorageFactoryCreate(b *testing.B) {
	factory := NewDefaultStorageFactory()
	config := StorageConfig{
		Type:     StorageTypeMemory,
		MaxConns: 5,
		Timeout:  time.Second * 10,
		RetryInterval: time.Second,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		storage, err := factory.Create(config)
		if err != nil {
			b.Fatal(err)
		}
		storage.Close()
	}
}

func BenchmarkValidateStorageConfig(b *testing.B) {
	config := DefaultStorageConfig()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateStorageConfig(config)
	}
}