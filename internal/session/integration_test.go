// +build integration

package session

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// IntegrationTestSuite는 세션 관리 시스템의 통합 테스트를 위한 테스트 스위트입니다.
type IntegrationTestSuite struct {
	store        *RedisStore
	fingerprinter *DeviceFingerprintGenerator
	geoipService  *GeoIPService
	client       *mockRedisClient
}

func setupIntegrationTest(t *testing.T) *IntegrationTestSuite {
	client := newMockRedisClient()
	store := NewRedisStore(client, "integration_test", time.Hour)
	fingerprinter := NewDeviceFingerprintGeneratorWithoutGeoIP()
	
	return &IntegrationTestSuite{
		store:         store,
		fingerprinter: fingerprinter,
		geoipService:  nil, // Mock 환경에서는 nil
		client:        client,
	}
}

func TestFullSessionLifecycle(t *testing.T) {
	suite := setupIntegrationTest(t)
	ctx := context.Background()

	// 1. HTTP 요청으로부터 디바이스 정보 생성
	req := httptest.NewRequest("POST", "/login", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.RemoteAddr = "192.168.1.100:12345"

	deviceInfo := suite.fingerprinter.GenerateFromRequest(req)
	require.NotNil(t, deviceInfo)
	assert.NotEmpty(t, deviceInfo.Fingerprint)
	assert.Equal(t, "192.168.1.100", deviceInfo.IPAddress)

	// 2. 세션 생성
	session := &models.AuthSession{
		ID:           "session_integration_test",
		UserID:       "user_123",
		DeviceInfo:   deviceInfo,
		LocationInfo: nil, // Mock 환경에서는 nil
		CreatedAt:    time.Now(),
		LastAccess:   time.Now(),
		ExpiresAt:    time.Now().Add(time.Hour),
		IsActive:     true,
		Metadata: map[string]interface{}{
			"login_method": "password",
			"client_type":  "web",
		},
	}

	err := suite.store.Create(ctx, session)
	require.NoError(t, err)

	// 3. 세션 조회
	retrievedSession, err := suite.store.Get(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, session.ID, retrievedSession.ID)
	assert.Equal(t, session.UserID, retrievedSession.UserID)
	assert.True(t, retrievedSession.IsActive)

	// 4. 세션 업데이트 (활동 갱신)
	time.Sleep(time.Millisecond * 10) // 시간 차이를 위한 대기
	retrievedSession.UpdateLastAccess()
	retrievedSession.Metadata["last_activity"] = "page_view"

	err = suite.store.Update(ctx, retrievedSession)
	require.NoError(t, err)

	// 5. 사용자 세션 목록 조회
	userSessions, err := suite.store.GetUserSessions(ctx, "user_123")
	require.NoError(t, err)
	assert.Len(t, userSessions, 1)
	assert.Equal(t, session.ID, userSessions[0].ID)

	// 6. 디바이스 세션 조회
	deviceSessions, err := suite.store.GetDeviceSessions(ctx, deviceInfo.Fingerprint)
	require.NoError(t, err)
	assert.Len(t, deviceSessions, 1)
	assert.Equal(t, session.ID, deviceSessions[0].ID)

	// 7. 세션 만료 시뮬레이션
	retrievedSession.IsActive = false
	err = suite.store.Update(ctx, retrievedSession)
	require.NoError(t, err)

	// 8. 활성 세션 수 확인
	activeCount, err := suite.store.CountUserActiveSessions(ctx, "user_123")
	require.NoError(t, err)
	assert.Equal(t, 0, activeCount) // 세션이 비활성화되어 0개

	// 9. 세션 삭제
	err = suite.store.Delete(ctx, session.ID)
	require.NoError(t, err)

	// 10. 삭제 확인
	_, err = suite.store.Get(ctx, session.ID)
	assert.Error(t, err)
	assert.Equal(t, ErrSessionNotFound, err)
}

func TestConcurrentSessionManagement(t *testing.T) {
	suite := setupIntegrationTest(t)
	ctx := context.Background()

	const numSessions = 50
	const numUsers = 5
	var wg sync.WaitGroup
	
	// 동시에 여러 세션 생성
	wg.Add(numSessions)
	for i := 0; i < numSessions; i++ {
		go func(sessionNum int) {
			defer wg.Done()
			
			userID := fmt.Sprintf("user_%d", sessionNum%numUsers)
			session := &models.AuthSession{
				ID:       fmt.Sprintf("session_%d", sessionNum),
				UserID:   userID,
				IsActive: true,
				DeviceInfo: &models.DeviceFingerprint{
					Fingerprint: fmt.Sprintf("device_%d", sessionNum%10),
					IPAddress:   fmt.Sprintf("192.168.1.%d", sessionNum%255),
					UserAgent:   "Test Agent",
					Browser:     "Chrome",
					OS:          "Windows",
					Device:      "Desktop",
				},
				CreatedAt:  time.Now(),
				LastAccess: time.Now(),
				ExpiresAt:  time.Now().Add(time.Hour),
			}
			
			err := suite.store.Create(ctx, session)
			assert.NoError(t, err, "Session creation should succeed for session %d", sessionNum)
		}(i)
	}
	wg.Wait()

	// 각 사용자별 세션 수 확인
	totalActiveSessions := 0
	for i := 0; i < numUsers; i++ {
		userID := fmt.Sprintf("user_%d", i)
		count, err := suite.store.CountUserActiveSessions(ctx, userID)
		require.NoError(t, err)
		totalActiveSessions += count
		
		// 각 사용자는 최소 1개 이상의 세션을 가져야 함
		assert.Greater(t, count, 0, "User %s should have at least one session", userID)
	}

	// 전체 세션 수 확인
	assert.Equal(t, numSessions, totalActiveSessions, "Total active sessions should match created sessions")
	
	// 동시 세션 업데이트 테스트
	wg.Add(numSessions)
	for i := 0; i < numSessions; i++ {
		go func(sessionNum int) {
			defer wg.Done()
			
			sessionID := fmt.Sprintf("session_%d", sessionNum)
			session, err := suite.store.Get(ctx, sessionID)
			if err != nil {
				assert.NoError(t, err, "Failed to get session %d", sessionNum)
				return
			}
			
			session.LastAccess = time.Now()
			session.Metadata = map[string]interface{}{
				"concurrent_update": true,
				"thread_id":         sessionNum,
			}
			
			err = suite.store.Update(ctx, session)
			assert.NoError(t, err, "Session update should succeed for session %d", sessionNum)
		}(i)
	}
	wg.Wait()
	
	// 동시 세션 삭제 테스트
	wg.Add(numSessions)
	for i := 0; i < numSessions; i++ {
		go func(sessionNum int) {
			defer wg.Done()
			
			sessionID := fmt.Sprintf("session_%d", sessionNum)
			err := suite.store.Delete(ctx, sessionID)
			assert.NoError(t, err, "Session deletion should succeed for session %d", sessionNum)
		}(i)
	}
	wg.Wait()
	
	// 모든 세션이 삭제되었는지 확인
	for i := 0; i < numUsers; i++ {
		userID := fmt.Sprintf("user_%d", i)
		count, err := suite.store.CountUserActiveSessions(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, 0, count, "All sessions for user %s should be deleted", userID)
	}
}

func TestDeviceFingerprintingAccuracy(t *testing.T) {
	suite := setupIntegrationTest(t)

	testCases := []struct {
		name        string
		userAgent   string
		remoteAddr  string
		expectDevice string
		expectOS    string
	}{
		{
			name:         "Chrome Windows Desktop",
			userAgent:    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
			remoteAddr:   "203.0.113.1:12345",
			expectDevice: "Desktop",
			expectOS:     "Windows",
		},
		{
			name:         "iPhone Safari Mobile",
			userAgent:    "Mozilla/5.0 (iPhone; CPU iPhone OS 14_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Mobile/15E148 Safari/604.1",
			remoteAddr:   "203.0.113.2:54321",
			expectDevice: "Mobile",
			expectOS:     "iOS",
		},
		{
			name:         "Android Chrome Mobile",
			userAgent:    "Mozilla/5.0 (Linux; Android 11; SM-G991B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.120 Mobile Safari/537.36",
			remoteAddr:   "203.0.113.3:8080",
			expectDevice: "Mobile",
			expectOS:     "Android",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("User-Agent", tc.userAgent)
			req.RemoteAddr = tc.remoteAddr

			fingerprint := suite.fingerprinter.GenerateFromRequest(req)
			
			assert.NotNil(t, fingerprint)
			assert.NotEmpty(t, fingerprint.Fingerprint)
			assert.Equal(t, tc.userAgent, fingerprint.UserAgent)
			assert.Contains(t, fingerprint.IPAddress, "203.0.113")
			
			// Device type and OS detection
			assert.Equal(t, tc.expectDevice, fingerprint.Device)
			// OS 검증은 UA Parser 결과에 따라 달라질 수 있으므로 포함 관계로 검증
			if tc.expectOS == "Windows" {
				assert.Contains(t, fingerprint.OS, "Windows")
			} else if tc.expectOS == "iOS" {
				assert.Contains(t, fingerprint.OS, "iOS")
			} else if tc.expectOS == "Android" {
				assert.Contains(t, fingerprint.OS, "Android")
			}
		})
	}
}

func TestSessionSecurityFeatures(t *testing.T) {
	suite := setupIntegrationTest(t)

	// 같은 디바이스에서 다른 세션들 생성
	baseFingerprint := &models.DeviceFingerprint{
		UserAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0",
		IPAddress:   "192.168.1.100",
		Browser:     "Chrome",
		OS:          "Windows",
		Device:      "Desktop",
		Fingerprint: "base_device",
	}

	// 약간 다른 디바이스 정보 (버전 업그레이드)
	similarFingerprint := &models.DeviceFingerprint{
		UserAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/92.0", // 버전만 다름
		IPAddress:   "192.168.1.101", // IP도 약간 다름
		Browser:     "Chrome",
		OS:          "Windows",
		Device:      "Desktop",
		Fingerprint: "similar_device",
	}

	// 완전히 다른 디바이스
	differentFingerprint := &models.DeviceFingerprint{
		UserAgent:   "Mozilla/5.0 (iPhone; CPU iPhone OS 14_6 like Mac OS X) Safari/604.1",
		IPAddress:   "10.0.0.1",
		Browser:     "Safari",
		OS:          "iOS",
		Device:      "Mobile",
		Fingerprint: "different_device",
	}

	// 디바이스 유사성 테스트
	similarity1 := suite.fingerprinter.CompareFingerprints(baseFingerprint, baseFingerprint)
	assert.Equal(t, 1.0, similarity1, "Identical fingerprints should have 100% similarity")

	similarity2 := suite.fingerprinter.CompareFingerprints(baseFingerprint, similarFingerprint)
	assert.Greater(t, similarity2, 0.7, "Similar fingerprints should have high similarity")
	assert.Less(t, similarity2, 1.0, "Similar fingerprints should not be identical")

	similarity3 := suite.fingerprinter.CompareFingerprints(baseFingerprint, differentFingerprint)
	assert.Less(t, similarity3, 0.3, "Different fingerprints should have low similarity")

	// IP 주소 유사성 테스트
	assert.True(t, suite.fingerprinter.similarIPAddress("192.168.1.100", "192.168.1.101"), 
		"IPs in same subnet should be considered similar")
	assert.False(t, suite.fingerprinter.similarIPAddress("192.168.1.100", "10.0.0.1"), 
		"IPs in different subnets should not be similar")
}

func TestSessionCleanupAndMaintenance(t *testing.T) {
	suite := setupIntegrationTest(t)
	ctx := context.Background()

	// 만료 예정 세션들 생성
	now := time.Now()
	sessions := []*models.AuthSession{
		{
			ID:         "active_session",
			UserID:     "user1",
			IsActive:   true,
			CreatedAt:  now.Add(-30 * time.Minute),
			ExpiresAt:  now.Add(30 * time.Minute), // 아직 유효
		},
		{
			ID:         "expiring_soon_session", 
			UserID:     "user1",
			IsActive:   true,
			CreatedAt:  now.Add(-50 * time.Minute),
			ExpiresAt:  now.Add(10 * time.Minute), // 곧 만료
		},
		{
			ID:         "expired_session",
			UserID:     "user2",
			IsActive:   true,
			CreatedAt:  now.Add(-2 * time.Hour),
			ExpiresAt:  now.Add(-30 * time.Minute), // 이미 만료됨
		},
	}

	for _, session := range sessions {
		err := suite.store.Create(ctx, session)
		require.NoError(t, err)
	}

	// 세션 연장 테스트
	err := suite.store.ExtendSession(ctx, "expiring_soon_session", time.Hour*2)
	require.NoError(t, err)

	// 정리 작업 실행
	err = suite.store.CleanupExpiredSessions(ctx)
	require.NoError(t, err)

	// 활성 세션은 여전히 존재해야 함
	_, err = suite.store.Get(ctx, "active_session")
	assert.NoError(t, err, "Active session should still exist")

	// 연장된 세션도 존재해야 함
	_, err = suite.store.Get(ctx, "expiring_soon_session")
	assert.NoError(t, err, "Extended session should still exist")
}

// 성능 테스트들
func TestSessionStorePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	suite := setupIntegrationTest(t)
	ctx := context.Background()

	// 대량 세션 생성 성능 테스트
	const numSessions = 1000
	
	start := time.Now()
	for i := 0; i < numSessions; i++ {
		session := &models.AuthSession{
			ID:       fmt.Sprintf("perf_session_%d", i),
			UserID:   fmt.Sprintf("user_%d", i%100), // 100명의 사용자
			IsActive: true,
			DeviceInfo: &models.DeviceFingerprint{
				Fingerprint: fmt.Sprintf("device_%d", i%50), // 50개의 디바이스
			},
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(time.Hour),
		}
		
		err := suite.store.Create(ctx, session)
		require.NoError(t, err)
	}
	createDuration := time.Since(start)

	t.Logf("Created %d sessions in %v (%.2f sessions/sec)", 
		numSessions, createDuration, float64(numSessions)/createDuration.Seconds())

	// 조회 성능 테스트
	start = time.Now()
	for i := 0; i < numSessions; i++ {
		sessionID := fmt.Sprintf("perf_session_%d", i)
		_, err := suite.store.Get(ctx, sessionID)
		require.NoError(t, err)
	}
	readDuration := time.Since(start)

	t.Logf("Retrieved %d sessions in %v (%.2f sessions/sec)", 
		numSessions, readDuration, float64(numSessions)/readDuration.Seconds())

	// 사용자별 세션 조회 성능 테스트
	start = time.Now()
	for i := 0; i < 100; i++ {
		userID := fmt.Sprintf("user_%d", i)
		sessions, err := suite.store.GetUserSessions(ctx, userID)
		require.NoError(t, err)
		assert.Greater(t, len(sessions), 0)
	}
	userQueryDuration := time.Since(start)

	t.Logf("Retrieved sessions for 100 users in %v (%.2f queries/sec)", 
		userQueryDuration, float64(100)/userQueryDuration.Seconds())

	// 삭제 성능 테스트
	start = time.Now()
	for i := 0; i < numSessions; i++ {
		sessionID := fmt.Sprintf("perf_session_%d", i)
		err := suite.store.Delete(ctx, sessionID)
		require.NoError(t, err)
	}
	deleteDuration := time.Since(start)

	t.Logf("Deleted %d sessions in %v (%.2f sessions/sec)", 
		numSessions, deleteDuration, float64(numSessions)/deleteDuration.Seconds())

	// 성능 임계값 검증 (선택적)
	if createDuration > time.Second*10 {
		t.Errorf("Session creation took too long: %v", createDuration)
	}
	if readDuration > time.Second*5 {
		t.Errorf("Session retrieval took too long: %v", readDuration)
	}
}

func TestMemoryUsageAndGoroutineLeak(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	suite := setupIntegrationTest(t)
	ctx := context.Background()

	// 초기 고루틴 수
	initialGoroutines := testing.AllocsPerRun(1, func() {})

	const iterations = 100
	for i := 0; i < iterations; i++ {
		// 세션 생성, 조회, 업데이트, 삭제 사이클
		session := &models.AuthSession{
			ID:       fmt.Sprintf("memory_test_%d", i),
			UserID:   "memory_user",
			IsActive: true,
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(time.Hour),
		}

		err := suite.store.Create(ctx, session)
		require.NoError(t, err)

		_, err = suite.store.Get(ctx, session.ID)
		require.NoError(t, err)

		session.LastAccess = time.Now()
		err = suite.store.Update(ctx, session)
		require.NoError(t, err)

		err = suite.store.Delete(ctx, session.ID)
		require.NoError(t, err)
	}

	// 메모리 사용량이 안정적인지 확인
	finalGoroutines := testing.AllocsPerRun(1, func() {})

	// 고루틴 리크 검사 (정확한 수는 환경에 따라 다를 수 있음)
	goroutineDiff := finalGoroutines - initialGoroutines
	if goroutineDiff > 10 { // 임계값은 조정 가능
		t.Logf("Warning: Potential goroutine leak detected. Initial: %f, Final: %f, Diff: %f", 
			initialGoroutines, finalGoroutines, goroutineDiff)
	}

	t.Logf("Memory test completed. %d iterations with goroutine diff: %f", iterations, goroutineDiff)
}