package claude

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTokenManager(t *testing.T) {
	t.Run("with OAuth token", func(t *testing.T) {
		tm := NewTokenManager("test-token", "test-api-key", nil)
		assert.NotNil(t, tm)
		
		token, err := tm.GetToken(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "test-token", token)
	})

	t.Run("with API key only", func(t *testing.T) {
		tm := NewTokenManager("", "test-api-key", nil)
		assert.NotNil(t, tm)
		
		token, err := tm.GetToken(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "test-api-key", token)
	})

	t.Run("without credentials", func(t *testing.T) {
		tm := NewTokenManager("", "", nil)
		assert.NotNil(t, tm)
		
		_, err := tm.GetToken(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "유효한 인증 정보가 없습니다")
	})
}

func TestTokenManager_ValidateToken(t *testing.T) {
	t.Run("valid OAuth token", func(t *testing.T) {
		tm := NewTokenManager("test-token", "", nil)
		
		err := tm.ValidateToken("test-token")
		assert.NoError(t, err)
	})

	t.Run("valid API key", func(t *testing.T) {
		tm := NewTokenManager("", "test-api-key", nil)
		
		err := tm.ValidateToken("test-api-key")
		assert.NoError(t, err)
	})

	t.Run("invalid token", func(t *testing.T) {
		tm := NewTokenManager("test-token", "test-api-key", nil)
		
		err := tm.ValidateToken("invalid-token")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "유효하지 않은 토큰입니다")
	})

	t.Run("empty token", func(t *testing.T) {
		tm := NewTokenManager("test-token", "", nil)
		
		err := tm.ValidateToken("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "토큰이 비어있습니다")
	})
}

func TestTokenManager_SetToken(t *testing.T) {
	tm := NewTokenManager("old-token", "", nil)
	
	// 초기 토큰 확인
	token, err := tm.GetToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "old-token", token)
	
	// 새 토큰 설정
	newExpiresAt := time.Now().Add(2 * time.Hour)
	tm.SetToken("new-token", newExpiresAt)
	
	// 변경된 토큰 확인
	token, err = tm.GetToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "new-token", token)
}

func TestTokenManager_RefreshToken(t *testing.T) {
	t.Run("with refresh function", func(t *testing.T) {
		refreshCalled := false
		refreshFunc := func(ctx context.Context) (string, time.Time, error) {
			refreshCalled = true
			return "refreshed-token", time.Now().Add(time.Hour), nil
		}
		
		tm := NewTokenManager("", "", refreshFunc)
		
		err := tm.RefreshToken(context.Background())
		require.NoError(t, err)
		assert.True(t, refreshCalled)
		
		// 갱신된 토큰 확인
		token, err := tm.GetToken(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "refreshed-token", token)
	})

	t.Run("without refresh function", func(t *testing.T) {
		tm := NewTokenManager("", "", nil)
		
		err := tm.RefreshToken(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "토큰 갱신 함수가 설정되지 않았습니다")
	})
}

func TestTokenManager_ExpiredToken(t *testing.T) {
	refreshCalled := false
	refreshFunc := func(ctx context.Context) (string, time.Time, error) {
		refreshCalled = true
		return "new-token", time.Now().Add(time.Hour), nil
	}
	
	tm := NewTokenManager("expired-token", "api-key", refreshFunc)
	
	// 만료된 시간으로 설정
	tm.SetToken("expired-token", time.Now().Add(-time.Hour))
	
	// GetToken은 자동으로 갱신을 시도해야 함
	token, err := tm.GetToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "new-token", token)
	assert.True(t, refreshCalled)
}

func TestTokenManager_Concurrency(t *testing.T) {
	tm := NewTokenManager("test-token", "test-api-key", nil)
	
	// 동시에 여러 고루틴에서 접근
	done := make(chan bool, 3)
	
	// 토큰 읽기
	go func() {
		for i := 0; i < 100; i++ {
			_, _ = tm.GetToken(context.Background())
		}
		done <- true
	}()
	
	// 토큰 검증
	go func() {
		for i := 0; i < 100; i++ {
			_ = tm.ValidateToken("test-token")
		}
		done <- true
	}()
	
	// 토큰 설정
	go func() {
		for i := 0; i < 100; i++ {
			tm.SetToken("new-token", time.Now().Add(time.Hour))
		}
		done <- true
	}()
	
	// 모든 고루틴 완료 대기
	for i := 0; i < 3; i++ {
		<-done
	}
}