package session

import (
	"context"
	"fmt"
	"math"
	"net"
	"strings"
	"time"

	"github.com/aicli/aicli-web/internal/models"
)

// RedisSecurityChecker는 Redis 기반 세션 보안 검사기입니다.
type RedisSecurityChecker struct {
	store               Store
	monitor             Monitor
	deviceGenerator     *DeviceFingerprintGenerator
	maxLocationDistance float64 // 최대 허용 거리 (km)
	suspiciousThreshold float64 // 의심 활동 임계값
}

// NewRedisSecurityChecker는 새로운 보안 검사기를 생성합니다.
func NewRedisSecurityChecker(store Store, monitor Monitor) *RedisSecurityChecker {
	return &RedisSecurityChecker{
		store:               store,
		monitor:             monitor,
		deviceGenerator:     NewDeviceFingerprintGeneratorWithoutGeoIP(),
		maxLocationDistance: 1000.0, // 기본 1000km
		suspiciousThreshold: 0.5,    // 50% 미만 유사도면 의심
	}
}

// SetLocationDistanceThreshold는 위치 변경 허용 거리를 설정합니다.
func (s *RedisSecurityChecker) SetLocationDistanceThreshold(distance float64) {
	s.maxLocationDistance = distance
}

// SetSuspiciousThreshold는 의심 활동 임계값을 설정합니다.
func (s *RedisSecurityChecker) SetSuspiciousThreshold(threshold float64) {
	s.suspiciousThreshold = threshold
}

// CheckDeviceFingerprint는 디바이스 핑거프린트를 검증합니다.
func (s *RedisSecurityChecker) CheckDeviceFingerprint(ctx context.Context, userID string, deviceInfo *models.DeviceFingerprint) error {
	if deviceInfo == nil {
		return fmt.Errorf("디바이스 정보가 제공되지 않았습니다")
	}
	
	// 사용자의 기존 세션들 조회
	existingSessions, err := s.store.GetUserSessions(ctx, userID)
	if err != nil {
		return fmt.Errorf("사용자 세션 조회 실패: %w", err)
	}
	
	// 첫 번째 세션인 경우 통과
	if len(existingSessions) == 0 {
		return nil
	}
	
	// 기존 디바이스들과 비교
	var maxSimilarity float64
	var mostSimilarDevice *models.DeviceFingerprint
	
	for _, session := range existingSessions {
		if session.DeviceInfo != nil {
			similarity := s.deviceGenerator.CompareFingerprints(deviceInfo, session.DeviceInfo)
			if similarity > maxSimilarity {
				maxSimilarity = similarity
				mostSimilarDevice = session.DeviceInfo
			}
		}
	}
	
	// 유사도가 임계값 미만이면 의심스러운 디바이스로 판단
	if maxSimilarity < s.suspiciousThreshold {
		// 의심스러운 활동 기록
		event := &SessionEvent{
			UserID:    userID,
			EventType: EventDeviceChanged,
			Timestamp: time.Now(),
			Severity:  SeverityWarning,
			Description: fmt.Sprintf("새로운 디바이스 감지 (유사도: %.2f%%)", maxSimilarity*100),
			EventData: map[string]interface{}{
				"new_device":        deviceInfo,
				"similar_device":    mostSimilarDevice,
				"similarity_score":  maxSimilarity,
				"threshold":         s.suspiciousThreshold,
			},
		}
		
		if s.monitor != nil {
			s.monitor.RecordSuspiciousActivity(ctx, event)
		}
		
		return ErrDeviceNotRecognized
	}
	
	return nil
}

// CheckLocationChange는 위치 변경을 감지합니다.
func (s *RedisSecurityChecker) CheckLocationChange(ctx context.Context, sessionID string, newLocation *models.LocationInfo) error {
	if newLocation == nil {
		return nil // 위치 정보가 없으면 검사하지 않음
	}
	
	// 현재 세션 정보 조회
	session, err := s.store.Get(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("세션 조회 실패: %w", err)
	}
	
	// 기존 위치 정보가 없으면 새 위치로 설정
	if session.LocationInfo == nil {
		session.LocationInfo = newLocation
		return s.store.Update(ctx, session)
	}
	
	// 위치 간 거리 계산
	distance := s.calculateDistance(session.LocationInfo, newLocation)
	
	// 허용 거리를 초과하면 의심스러운 활동으로 기록
	if distance > s.maxLocationDistance {
		event := &SessionEvent{
			SessionID: sessionID,
			UserID:    session.UserID,
			EventType: EventLocationChanged,
			Timestamp: time.Now(),
			Severity:  SeverityWarning,
			Description: fmt.Sprintf("비정상적인 위치 변경 감지 (거리: %.2fkm)", distance),
			EventData: map[string]interface{}{
				"old_location":   session.LocationInfo,
				"new_location":   newLocation,
				"distance_km":    distance,
				"threshold_km":   s.maxLocationDistance,
			},
		}
		
		if s.monitor != nil {
			s.monitor.RecordSuspiciousActivity(ctx, event)
		}
		
		return ErrLocationChanged
	}
	
	// 위치 정보 업데이트
	session.LocationInfo = newLocation
	return s.store.Update(ctx, session)
}

// DetectSuspiciousActivity는 의심스러운 활동을 감지합니다.  
func (s *RedisSecurityChecker) DetectSuspiciousActivity(ctx context.Context, session *models.AuthSession) (bool, string) {
	var suspiciousReasons []string
	
	// 1. 비정상적인 접속 시간 패턴 검사
	if s.isUnusualAccessTime(session) {
		suspiciousReasons = append(suspiciousReasons, "비정상적인 접속 시간")
	}
	
	// 2. 짧은 시간 내 많은 요청 검사
	if s.isHighFrequencyAccess(ctx, session) {
		suspiciousReasons = append(suspiciousReasons, "짧은 시간 내 과도한 접속")
	}
	
	// 3. 의심스러운 User-Agent 검사
	if s.isSuspiciousUserAgent(session) {
		suspiciousReasons = append(suspiciousReasons, "의심스러운 사용자 에이전트")
	}
	
	// 4. IP 주소 변경 패턴 검사
	if s.isIPAddressHopping(ctx, session) {
		suspiciousReasons = append(suspiciousReasons, "IP 주소 빈번한 변경")
	}
	
	// 5. 봇 활동 패턴 검사
	if s.isBotLikeActivity(session) {
		suspiciousReasons = append(suspiciousReasons, "봇과 유사한 활동 패턴")
	}
	
	if len(suspiciousReasons) > 0 {
		return true, strings.Join(suspiciousReasons, ", ")
	}
	
	return false, ""
}

// ValidateConcurrentSessions는 동시 세션 제한을 검증합니다.
func (s *RedisSecurityChecker) ValidateConcurrentSessions(ctx context.Context, userID string, maxSessions int) error {
	count, err := s.store.CountUserActiveSessions(ctx, userID)
	if err != nil {
		return fmt.Errorf("활성 세션 수 조회 실패: %w", err)
	}
	
	if count >= maxSessions {
		return ErrConcurrentSessionLimitExceeded
	}
	
	return nil
}

// isUnusualAccessTime은 비정상적인 접속 시간을 검사합니다.
func (s *RedisSecurityChecker) isUnusualAccessTime(session *models.AuthSession) bool {
	// 심야 시간대(새벽 2-5시) 접속을 의심스럽게 판단
	hour := session.CreatedAt.Hour()
	return hour >= 2 && hour <= 5
}

// isHighFrequencyAccess는 짧은 시간 내 많은 접속을 검사합니다.
func (s *RedisSecurityChecker) isHighFrequencyAccess(ctx context.Context, session *models.AuthSession) bool {
	// 사용자의 최근 세션들을 조회하여 생성 빈도 확인
	recentSessions, err := s.store.GetUserSessions(ctx, session.UserID)
	if err != nil {
		return false
	}
	
	// 최근 1시간 내 생성된 세션 수 확인
	recentCount := 0
	oneHourAgo := time.Now().Add(-time.Hour)
	
	for _, recentSession := range recentSessions {
		if recentSession.CreatedAt.After(oneHourAgo) {
			recentCount++
		}
	}
	
	// 1시간 내 5개 이상의 세션은 의심스러움
	return recentCount >= 5
}

// isSuspiciousUserAgent는 의심스러운 User-Agent를 검사합니다.
func (s *RedisSecurityChecker) isSuspiciousUserAgent(session *models.AuthSession) bool {
	if session.DeviceInfo == nil || session.DeviceInfo.UserAgent == "" {
		return true // User-Agent가 없는 경우 의심
	}
	
	ua := strings.ToLower(session.DeviceInfo.UserAgent)
	
	// 봇, 크롤러, 자동화 도구 패턴
	suspiciousPatterns := []string{
		"bot", "crawler", "spider", "scraper",
		"curl", "wget", "python", "go-http-client",
		"headless", "phantom", "selenium",
	}
	
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(ua, pattern) {
			return true
		}
	}
	
	// 너무 짧거나 긴 User-Agent
	if len(ua) < 20 || len(ua) > 500 {
		return true
	}
	
	return false
}

// isIPAddressHopping은 IP 주소 빈번한 변경을 검사합니다.
func (s *RedisSecurityChecker) isIPAddressHopping(ctx context.Context, session *models.AuthSession) bool {
	if session.DeviceInfo == nil {
		return false
	}
	
	// 같은 디바이스의 다른 세션들 조회
	deviceSessions, err := s.store.GetDeviceSessions(ctx, session.DeviceInfo.Fingerprint)
	if err != nil {
		return false
	}
	
	// 서로 다른 IP 주소 카운트
	ipSet := make(map[string]bool)
	for _, deviceSession := range deviceSessions {
		if deviceSession.DeviceInfo != nil && deviceSession.DeviceInfo.IPAddress != "" {
			ipSet[deviceSession.DeviceInfo.IPAddress] = true
		}
	}
	
	// 같은 디바이스에서 3개 이상의 다른 IP 사용은 의심
	return len(ipSet) >= 3
}

// isBotLikeActivity는 봇과 유사한 활동 패턴을 검사합니다.
func (s *RedisSecurityChecker) isBotLikeActivity(session *models.AuthSession) bool {
	// 세션 지속 시간이 너무 짧은 경우 (1분 미만)
	if time.Since(session.CreatedAt) < time.Minute {
		return true
	}
	
	// 디바이스 정보가 너무 일반적인 경우
	if session.DeviceInfo != nil {
		if session.DeviceInfo.Browser == "Unknown" && 
		   session.DeviceInfo.OS == "Unknown" && 
		   session.DeviceInfo.Device == "Unknown" {
			return true
		}
	}
	
	return false
}

// calculateDistance는 두 위치 간의 거리를 계산합니다 (Haversine formula).
func (s *RedisSecurityChecker) calculateDistance(loc1, loc2 *models.LocationInfo) float64 {
	if loc1 == nil || loc2 == nil {
		return 0
	}
	
	// 좌표가 0,0인 경우 거리를 0으로 반환
	if (loc1.Latitude == 0 && loc1.Longitude == 0) || 
	   (loc2.Latitude == 0 && loc2.Longitude == 0) {
		return 0
	}
	
	const R = 6371 // 지구 반지름 (km)
	
	lat1Rad := loc1.Latitude * math.Pi / 180
	lat2Rad := loc2.Latitude * math.Pi / 180
	deltaLatRad := (loc2.Latitude - loc1.Latitude) * math.Pi / 180
	deltaLngRad := (loc2.Longitude - loc1.Longitude) * math.Pi / 180
	
	a := math.Sin(deltaLatRad/2)*math.Sin(deltaLatRad/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLngRad/2)*math.Sin(deltaLngRad/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	
	return R * c
}

// GetLocationFromIP는 IP 주소로부터 위치 정보를 추출합니다.
func (s *RedisSecurityChecker) GetLocationFromIP(ipAddress string) (*models.LocationInfo, error) {
	// 실제 구현에서는 MaxMind GeoIP2 또는 다른 IP geolocation 서비스 사용
	// 현재는 간단한 국가 판별만 구현
	
	ip := net.ParseIP(ipAddress)
	if ip == nil {
		return nil, fmt.Errorf("잘못된 IP 주소: %s", ipAddress)
	}
	
	// 사설 IP 대역 체크
	if s.isPrivateIP(ip) {
		return &models.LocationInfo{
			Country: "Local",
			City:    "Private Network",
		}, nil
	}
	
	// 실제로는 GeoIP 데이터베이스를 사용해야 함
	// 여기서는 간단한 예시만 제공
	return &models.LocationInfo{
		Country: "Unknown",
		City:    "Unknown",
	}, nil
}

// isPrivateIP는 사설 IP 주소인지 확인합니다.
func (s *RedisSecurityChecker) isPrivateIP(ip net.IP) bool {
	privateBlocks := []*net.IPNet{
		{IP: net.IPv4(10, 0, 0, 0), Mask: net.CIDRMask(8, 32)},
		{IP: net.IPv4(172, 16, 0, 0), Mask: net.CIDRMask(12, 32)},
		{IP: net.IPv4(192, 168, 0, 0), Mask: net.CIDRMask(16, 32)},
		{IP: net.IPv4(127, 0, 0, 0), Mask: net.CIDRMask(8, 32)},
	}
	
	for _, block := range privateBlocks {
		if block.Contains(ip) {
			return true
		}
	}
	
	return false
}