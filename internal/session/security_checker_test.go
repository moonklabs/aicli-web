package session

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockStore는 테스트용 Store 인터페이스 모킹입니다.
type mockStore struct {
	mock.Mock
}

func (m *mockStore) Create(ctx context.Context, session *models.AuthSession) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *mockStore) Get(ctx context.Context, sessionID string) (*models.AuthSession, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AuthSession), args.Error(1)
}

func (m *mockStore) Update(ctx context.Context, session *models.AuthSession) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *mockStore) Delete(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *mockStore) GetUserSessions(ctx context.Context, userID string) ([]*models.AuthSession, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.AuthSession), args.Error(1)
}

func (m *mockStore) GetDeviceSessions(ctx context.Context, fingerprint string) ([]*models.AuthSession, error) {
	args := m.Called(ctx, fingerprint)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.AuthSession), args.Error(1)
}

func (m *mockStore) CleanupExpiredSessions(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockStore) ExtendSession(ctx context.Context, sessionID string, duration time.Duration) error {
	args := m.Called(ctx, sessionID, duration)
	return args.Error(0)
}

func (m *mockStore) CountUserActiveSessions(ctx context.Context, userID string) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

// mockMonitor는 테스트용 Monitor 인터페이스 모킹입니다.
type mockMonitor struct {
	mock.Mock
}

func (m *mockMonitor) GetActiveSessions(ctx context.Context) ([]*models.AuthSession, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.AuthSession), args.Error(1)
}

func (m *mockMonitor) GetSessionMetrics(ctx context.Context) (*SessionMetrics, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SessionMetrics), args.Error(1)
}

func (m *mockMonitor) GetSessionHistory(ctx context.Context, userID string, limit int) ([]*SessionEvent, error) {
	args := m.Called(ctx, userID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*SessionEvent), args.Error(1)
}

func (m *mockMonitor) RecordSessionEvent(ctx context.Context, event *SessionEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *mockMonitor) RecordSuspiciousActivity(ctx context.Context, event *SessionEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func TestNewRedisSecurityChecker(t *testing.T) {
	store := &mockStore{}
	monitor := &mockMonitor{}
	
	checker := NewRedisSecurityChecker(store, monitor)
	
	assert.NotNil(t, checker)
	assert.Equal(t, store, checker.store)
	assert.Equal(t, monitor, checker.monitor)
	assert.NotNil(t, checker.deviceGenerator)
	assert.Equal(t, 1000.0, checker.maxLocationDistance)
	assert.Equal(t, 0.5, checker.suspiciousThreshold)
}

func TestRedisSecurityChecker_SetThresholds(t *testing.T) {
	checker := NewRedisSecurityChecker(&mockStore{}, &mockMonitor{})
	
	checker.SetLocationDistanceThreshold(500.0)
	assert.Equal(t, 500.0, checker.maxLocationDistance)
	
	checker.SetSuspiciousThreshold(0.7)
	assert.Equal(t, 0.7, checker.suspiciousThreshold)
}

func TestRedisSecurityChecker_CheckDeviceFingerprint(t *testing.T) {
	store := &mockStore{}
	monitor := &mockMonitor{}
	checker := NewRedisSecurityChecker(store, monitor)
	ctx := context.Background()
	
	tests := []struct {
		name           string
		userID         string
		deviceInfo     *models.DeviceFingerprint
		existingSessions []*models.AuthSession
		expectError    error
		shouldRecord   bool
	}{
		{
			name:             "nil device info",
			userID:           "user1",
			deviceInfo:       nil,
			existingSessions: nil,
			expectError:      assert.AnError,
			shouldRecord:     false,
		},
		{
			name:   "first session - no existing sessions",
			userID: "user1",
			deviceInfo: &models.DeviceFingerprint{
				Fingerprint: "new_device",
				Browser:     "Chrome",
				OS:          "Windows",
				Device:      "Desktop",
			},
			existingSessions: []*models.AuthSession{},
			expectError:      nil,
			shouldRecord:     false,
		},
		{
			name:   "similar device - should pass",
			userID: "user1",
			deviceInfo: &models.DeviceFingerprint{
				UserAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0",
				IPAddress:   "192.168.1.100",
				Browser:     "Chrome",
				OS:          "Windows",
				Device:      "Desktop",
				Fingerprint: "similar_device",
			},
			existingSessions: []*models.AuthSession{
				{
					ID:     "existing_session",
					UserID: "user1",
					DeviceInfo: &models.DeviceFingerprint{
						UserAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0",
						IPAddress:   "192.168.1.101",
						Browser:     "Chrome", 
						OS:          "Windows",
						Device:      "Desktop",
						Fingerprint: "existing_device",
					},
				},
			},
			expectError:  nil,
			shouldRecord: false,
		},
		{
			name:   "completely different device - should fail",
			userID: "user1",
			deviceInfo: &models.DeviceFingerprint{
				UserAgent:   "Mozilla/5.0 (iPhone; CPU iPhone OS 14_6 like Mac OS X) Safari/604.1",
				IPAddress:   "10.0.0.1",
				Browser:     "Safari",
				OS:          "iOS",
				Device:      "Mobile",
				Fingerprint: "different_device",
			},
			existingSessions: []*models.AuthSession{
				{
					ID:     "existing_session",
					UserID: "user1",
					DeviceInfo: &models.DeviceFingerprint{
						UserAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0",
						IPAddress:   "192.168.1.100",
						Browser:     "Chrome",
						OS:          "Windows", 
						Device:      "Desktop",
						Fingerprint: "existing_device",
					},
				},
			},
			expectError:  ErrDeviceNotRecognized,
			shouldRecord: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			store.ExpectedCalls = nil
			monitor.ExpectedCalls = nil
			
			store.On("GetUserSessions", ctx, tt.userID).Return(tt.existingSessions, nil)
			
			if tt.shouldRecord {
				monitor.On("RecordSuspiciousActivity", ctx, mock.AnythingOfType("*session.SessionEvent")).Return(nil)
			}
			
			// Execute
			err := checker.CheckDeviceFingerprint(ctx, tt.userID, tt.deviceInfo)
			
			// Verify
			if tt.expectError != nil {
				if tt.expectError == assert.AnError {
					assert.Error(t, err)
				} else {
					assert.Equal(t, tt.expectError, err)
				}
			} else {
				assert.NoError(t, err)
			}
			
			// Verify mock calls
			store.AssertExpectations(t)
			monitor.AssertExpectations(t)
		})
	}
}

func TestRedisSecurityChecker_CheckLocationChange(t *testing.T) {
	store := &mockStore{}
	monitor := &mockMonitor{}
	checker := NewRedisSecurityChecker(store, monitor)
	checker.SetLocationDistanceThreshold(100.0) // 100km threshold
	ctx := context.Background()
	
	tests := []struct {
		name         string
		sessionID    string
		newLocation  *models.LocationInfo
		existingSession *models.AuthSession
		expectError  error
		shouldRecord bool
	}{
		{
			name:        "nil location - skip check",
			sessionID:   "session1",
			newLocation: nil,
			existingSession: &models.AuthSession{
				ID:     "session1",
				UserID: "user1",
			},
			expectError:  nil,
			shouldRecord: false,
		},
		{
			name:      "first location - should set",
			sessionID: "session1",
			newLocation: &models.LocationInfo{
				Country:   "US",
				City:      "New York",
				Latitude:  40.7128,
				Longitude: -74.0060,
			},
			existingSession: &models.AuthSession{
				ID:           "session1",
				UserID:       "user1",
				LocationInfo: nil,
			},
			expectError:  nil,
			shouldRecord: false,
		},
		{
			name:      "close location - should pass",
			sessionID: "session1",
			newLocation: &models.LocationInfo{
				Country:   "US",
				City:      "Jersey City",
				Latitude:  40.7282,
				Longitude: -74.0776,
			},
			existingSession: &models.AuthSession{
				ID:     "session1",
				UserID: "user1",
				LocationInfo: &models.LocationInfo{
					Country:   "US",
					City:      "New York",
					Latitude:  40.7128,
					Longitude: -74.0060,
				},
			},
			expectError:  nil,
			shouldRecord: false,
		},
		{
			name:      "far location - should fail",
			sessionID: "session1",
			newLocation: &models.LocationInfo{
				Country:   "UK",
				City:      "London",
				Latitude:  51.5074,
				Longitude: -0.1278,
			},
			existingSession: &models.AuthSession{
				ID:     "session1",
				UserID: "user1",
				LocationInfo: &models.LocationInfo{
					Country:   "US",
					City:      "New York",
					Latitude:  40.7128,
					Longitude: -74.0060,
				},
			},
			expectError:  ErrLocationChanged,
			shouldRecord: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			store.ExpectedCalls = nil
			monitor.ExpectedCalls = nil
			
			if tt.newLocation != nil {
				store.On("Get", ctx, tt.sessionID).Return(tt.existingSession, nil)
				store.On("Update", ctx, mock.AnythingOfType("*models.AuthSession")).Return(nil)
			}
			
			if tt.shouldRecord {
				monitor.On("RecordSuspiciousActivity", ctx, mock.AnythingOfType("*session.SessionEvent")).Return(nil)
			}
			
			// Execute
			err := checker.CheckLocationChange(ctx, tt.sessionID, tt.newLocation)
			
			// Verify
			if tt.expectError != nil {
				assert.Equal(t, tt.expectError, err)
			} else {
				assert.NoError(t, err)
			}
			
			// Verify mock calls
			store.AssertExpectations(t)
			monitor.AssertExpectations(t)
		})
	}
}

func TestRedisSecurityChecker_ValidateConcurrentSessions(t *testing.T) {
	store := &mockStore{}
	monitor := &mockMonitor{}
	checker := NewRedisSecurityChecker(store, monitor)
	ctx := context.Background()
	
	tests := []struct {
		name        string
		userID      string
		maxSessions int
		activeCount int
		expectError error
	}{
		{
			name:        "within limit",
			userID:      "user1",
			maxSessions: 5,
			activeCount: 3,
			expectError: nil,
		},
		{
			name:        "at limit",
			userID:      "user1", 
			maxSessions: 3,
			activeCount: 3,
			expectError: ErrConcurrentSessionLimitExceeded,
		},
		{
			name:        "over limit",
			userID:      "user1",
			maxSessions: 2,
			activeCount: 5,
			expectError: ErrConcurrentSessionLimitExceeded,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			store.ExpectedCalls = nil
			store.On("CountUserActiveSessions", ctx, tt.userID).Return(tt.activeCount, nil)
			
			// Execute
			err := checker.ValidateConcurrentSessions(ctx, tt.userID, tt.maxSessions)
			
			// Verify
			if tt.expectError != nil {
				assert.Equal(t, tt.expectError, err)
			} else {
				assert.NoError(t, err)
			}
			
			store.AssertExpectations(t)
		})
	}
}

func TestRedisSecurityChecker_DetectSuspiciousActivity(t *testing.T) {
	store := &mockStore{}
	monitor := &mockMonitor{}
	checker := NewRedisSecurityChecker(store, monitor)
	ctx := context.Background()
	
	tests := []struct {
		name            string
		session         *models.AuthSession
		existingSessions []*models.AuthSession
		expectSuspicious bool
		expectedReasons  []string
	}{
		{
			name: "normal session",
			session: &models.AuthSession{
				ID:        "normal_session",
				UserID:    "user1",
				CreatedAt: time.Now().Add(-time.Hour), // Normal time
				DeviceInfo: &models.DeviceFingerprint{
					UserAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0",
					Browser:     "Chrome",
					OS:          "Windows",
					Device:      "Desktop",
					IPAddress:   "192.168.1.100",
					Fingerprint: "normal_device",
				},
			},
			existingSessions: []*models.AuthSession{},
			expectSuspicious: false,
			expectedReasons:  nil,
		},
		{
			name: "unusual access time",
			session: &models.AuthSession{
				ID:        "night_session",
				UserID:    "user1",
				CreatedAt: time.Date(2023, 1, 1, 3, 0, 0, 0, time.UTC), // 3 AM
				DeviceInfo: &models.DeviceFingerprint{
					UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0",
					Browser:   "Chrome",
					OS:        "Windows",
					Device:    "Desktop",
				},
			},
			existingSessions: []*models.AuthSession{},
			expectSuspicious: true,
			expectedReasons:  []string{"비정상적인 접속 시간"},
		},
		{
			name: "suspicious user agent",
			session: &models.AuthSession{
				ID:        "bot_session",
				UserID:    "user1",
				CreatedAt: time.Now().Add(-time.Hour),
				DeviceInfo: &models.DeviceFingerprint{
					UserAgent: "python-requests/2.25.1",
					Browser:   "Unknown",
					OS:        "Unknown",
					Device:    "Unknown",
				},
			},
			existingSessions: []*models.AuthSession{},
			expectSuspicious: true,
			expectedReasons:  []string{"의심스러운 사용자 에이전트"},
		},
		{
			name: "short-lived bot session",
			session: &models.AuthSession{
				ID:        "short_session",
				UserID:    "user1",
				CreatedAt: time.Now().Add(-30 * time.Second), // Very recent
				DeviceInfo: &models.DeviceFingerprint{
					UserAgent: "bot-crawler/1.0",
					Browser:   "Unknown",
					OS:        "Unknown",
					Device:    "Unknown",
				},
			},
			existingSessions: []*models.AuthSession{},
			expectSuspicious: true,
			expectedReasons:  []string{"의심스러운 사용자 에이전트", "봇과 유사한 활동 패턴"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks for high frequency access check
			store.ExpectedCalls = nil
			store.On("GetUserSessions", ctx, tt.session.UserID).Return(tt.existingSessions, nil)
			
			if tt.session.DeviceInfo != nil && tt.session.DeviceInfo.Fingerprint != "" {
				store.On("GetDeviceSessions", ctx, tt.session.DeviceInfo.Fingerprint).Return([]*models.AuthSession{tt.session}, nil)
			}
			
			// Execute
			suspicious, reasons := checker.DetectSuspiciousActivity(ctx, tt.session)
			
			// Verify
			assert.Equal(t, tt.expectSuspicious, suspicious)
			if tt.expectSuspicious {
				assert.NotEmpty(t, reasons)
				// Verify that expected reasons are contained in the actual reasons
				for _, expectedReason := range tt.expectedReasons {
					assert.Contains(t, reasons, expectedReason)
				}
			} else {
				assert.Empty(t, reasons)
			}
			
			store.AssertExpectations(t)
		})
	}
}

func TestRedisSecurityChecker_CalculateDistance(t *testing.T) {
	checker := NewRedisSecurityChecker(&mockStore{}, &mockMonitor{})
	
	tests := []struct {
		name      string
		loc1      *models.LocationInfo
		loc2      *models.LocationInfo
		expected  float64
		tolerance float64
	}{
		{
			name: "same location",
			loc1: &models.LocationInfo{
				Latitude:  40.7128,
				Longitude: -74.0060,
			},
			loc2: &models.LocationInfo{
				Latitude:  40.7128,
				Longitude: -74.0060,
			},
			expected:  0.0,
			tolerance: 0.1,
		},
		{
			name: "NYC to Philadelphia",
			loc1: &models.LocationInfo{
				Latitude:  40.7128, // NYC
				Longitude: -74.0060,
			},
			loc2: &models.LocationInfo{
				Latitude:  39.9526, // Philadelphia
				Longitude: -75.1652,
			},
			expected:  130.0, // approximately 130km
			tolerance: 20.0,  // 20km tolerance
		},
		{
			name: "nil locations",
			loc1: nil,
			loc2: &models.LocationInfo{
				Latitude:  40.7128,
				Longitude: -74.0060,
			},
			expected:  0.0,
			tolerance: 0.1,
		},
		{
			name: "zero coordinates",
			loc1: &models.LocationInfo{
				Latitude:  0,
				Longitude: 0,
			},
			loc2: &models.LocationInfo{
				Latitude:  40.7128,
				Longitude: -74.0060,
			},
			expected:  0.0,
			tolerance: 0.1,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			distance := checker.calculateDistance(tt.loc1, tt.loc2)
			assert.InDelta(t, tt.expected, distance, tt.tolerance)
		})
	}
}

func TestRedisSecurityChecker_IsPrivateIP(t *testing.T) {
	checker := NewRedisSecurityChecker(&mockStore{}, &mockMonitor{})
	
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{
			name:     "private 192.168.x.x",
			ip:       "192.168.1.100",
			expected: true,
		},
		{
			name:     "private 10.x.x.x",
			ip:       "10.0.0.1",
			expected: true,
		},
		{
			name:     "private 172.16.x.x",
			ip:       "172.16.0.1",
			expected: true,
		},
		{
			name:     "localhost",
			ip:       "127.0.0.1",
			expected: true,
		},
		{
			name:     "public IP",
			ip:       "8.8.8.8",
			expected: false,
		},
		{
			name:     "public IP 2",
			ip:       "1.1.1.1",
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := parseIP(tt.ip)
			require.NotNil(t, ip)
			
			result := checker.isPrivateIP(ip)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// parseIP is a helper function for testing
func parseIP(s string) net.IP {
	return net.ParseIP(s)
}

// Benchmark tests
func BenchmarkSecurityChecker_CheckDeviceFingerprint(b *testing.B) {
	store := &mockStore{}
	monitor := &mockMonitor{}
	checker := NewRedisSecurityChecker(store, monitor)
	ctx := context.Background()
	
	deviceInfo := &models.DeviceFingerprint{
		UserAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0",
		IPAddress:   "192.168.1.100",
		Browser:     "Chrome",
		OS:          "Windows",
		Device:      "Desktop",
		Fingerprint: "test_device",
	}
	
	existingSessions := []*models.AuthSession{
		{
			ID:         "existing",
			UserID:     "user1",
			DeviceInfo: deviceInfo,
		},
	}
	
	store.On("GetUserSessions", ctx, "user1").Return(existingSessions, nil)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker.CheckDeviceFingerprint(ctx, "user1", deviceInfo)
	}
}

func BenchmarkSecurityChecker_DetectSuspiciousActivity(b *testing.B) {
	store := &mockStore{}
	monitor := &mockMonitor{}
	checker := NewRedisSecurityChecker(store, monitor)
	ctx := context.Background()
	
	session := &models.AuthSession{
		ID:        "test_session",
		UserID:    "user1",
		CreatedAt: time.Now().Add(-time.Hour),
		DeviceInfo: &models.DeviceFingerprint{
			UserAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0",
			Browser:     "Chrome",
			OS:          "Windows",
			Device:      "Desktop",
			Fingerprint: "test_device",
		},
	}
	
	store.On("GetUserSessions", ctx, "user1").Return([]*models.AuthSession{session}, nil)
	store.On("GetDeviceSessions", ctx, "test_device").Return([]*models.AuthSession{session}, nil)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker.DetectSuspiciousActivity(ctx, session)
	}
}