package session

import (
	"crypto/sha256"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/ua-parser/uap-go/uaparser"
)

// DeviceFingerprintGenerator는 디바이스 핑거프린트를 생성합니다.
type DeviceFingerprintGenerator struct {
	parser       *uaparser.Parser
	geoipService *GeoIPService
}

// NewDeviceFingerprintGenerator는 새로운 디바이스 핑거프린트 생성기를 생성합니다.
func NewDeviceFingerprintGenerator(geoipService *GeoIPService) *DeviceFingerprintGenerator {
	parser := uaparser.NewFromSaved()
	return &DeviceFingerprintGenerator{
		parser:       parser,
		geoipService: geoipService,
	}
}

// NewDeviceFingerprintGeneratorWithoutGeoIP는 GeoIP 없이 생성기를 생성합니다.
func NewDeviceFingerprintGeneratorWithoutGeoIP() *DeviceFingerprintGenerator {
	parser := uaparser.NewFromSaved()
	return &DeviceFingerprintGenerator{
		parser:       parser,
		geoipService: nil,
	}
}

// GenerateFromRequest는 HTTP 요청으로부터 디바이스 핑거프린트를 생성합니다.
func (g *DeviceFingerprintGenerator) GenerateFromRequest(r *http.Request) *models.DeviceFingerprint {
	userAgent := r.Header.Get("User-Agent")
	ipAddress := g.extractIPAddress(r)
	
	// User-Agent 파싱
	browser, os, device := g.parseUserAgent(userAgent)
	
	// 핑거프린트 생성
	fingerprint := g.generateFingerprint(userAgent, ipAddress, browser, os, device)
	
	return &models.DeviceFingerprint{
		UserAgent:   userAgent,
		IPAddress:   ipAddress,
		Browser:     browser,
		OS:          os,
		Device:      device,
		Fingerprint: fingerprint,
	}
}

// GetLocationFromRequest는 HTTP 요청으로부터 위치 정보를 조회합니다.
func (g *DeviceFingerprintGenerator) GetLocationFromRequest(r *http.Request) (*models.LocationInfo, error) {
	if g.geoipService == nil {
		return nil, ErrGeoIPDatabaseNotFound
	}
	
	ipAddress := g.extractIPAddress(r)
	return g.geoipService.GetLocationInfo(ipAddress)
}

// GenerateFullSessionInfo는 HTTP 요청으로부터 디바이스 핑거프린트와 위치 정보를 모두 생성합니다.
func (g *DeviceFingerprintGenerator) GenerateFullSessionInfo(r *http.Request) (*models.DeviceFingerprint, *models.LocationInfo, error) {
	deviceInfo := g.GenerateFromRequest(r)
	
	var locationInfo *models.LocationInfo
	var err error
	
	if g.geoipService != nil {
		locationInfo, err = g.geoipService.GetLocationInfo(deviceInfo.IPAddress)
		if err != nil {
			// 위치 정보 조회 실패는 치명적이지 않으므로 로그만 남기고 계속 진행
			locationInfo = nil
		}
	}
	
	return deviceInfo, locationInfo, nil
}

// extractIPAddress는 요청에서 실제 IP 주소를 추출합니다.
func (g *DeviceFingerprintGenerator) extractIPAddress(r *http.Request) string {
	// X-Forwarded-For 헤더 확인 (프록시/로드밸런서 뒤에 있을 경우)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// 첫 번째 IP가 실제 클라이언트 IP
		if ips := strings.Split(xff, ","); len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	
	// X-Real-IP 헤더 확인
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	
	// RemoteAddr에서 IP 추출
	if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return ip
	}
	
	return r.RemoteAddr
}

// parseUserAgent는 User-Agent 문자열을 파싱하여 브라우저, OS, 디바이스 정보를 추출합니다.
func (g *DeviceFingerprintGenerator) parseUserAgent(userAgent string) (browser, os, device string) {
	client := g.parser.Parse(userAgent)
	
	// 브라우저 정보 추출
	browser = fmt.Sprintf("%s %s", client.UserAgent.Family, client.UserAgent.Major)
	if client.UserAgent.Minor != "" {
		browser = fmt.Sprintf("%s.%s", browser, client.UserAgent.Minor)
	}
	
	// OS 정보 추출
	os = client.Os.Family
	if client.Os.Major != "" {
		os = fmt.Sprintf("%s %s", os, client.Os.Major)
		if client.Os.Minor != "" {
			os = fmt.Sprintf("%s.%s", os, client.Os.Minor)
		}
	}
	
	// 디바이스 정보 추출
	device = client.Device.Family
	if device == "" || device == "Other" {
		// UA Parser에서 디바이스를 감지하지 못한 경우 fallback 로직 사용
		device = g.detectDeviceFallback(userAgent)
	}
	
	return browser, os, device
}

// detectDeviceFallback는 UA Parser가 디바이스를 감지하지 못한 경우 fallback 로직을 제공합니다.
func (g *DeviceFingerprintGenerator) detectDeviceFallback(userAgent string) string {
	ua := strings.ToLower(userAgent)
	
	if strings.Contains(ua, "mobile") || strings.Contains(ua, "android") || strings.Contains(ua, "iphone") {
		return "Mobile"
	}
	
	if strings.Contains(ua, "tablet") || strings.Contains(ua, "ipad") {
		return "Tablet"
	}
	
	return "Desktop"
}

// generateFingerprint는 디바이스 정보로부터 고유한 핑거프린트를 생성합니다.
func (g *DeviceFingerprintGenerator) generateFingerprint(userAgent, ipAddress, browser, os, device string) string {
	// IP 주소는 개인정보이므로 해시에서 제외하거나 일부만 사용
	ipParts := strings.Split(ipAddress, ".")
	partialIP := ""
	if len(ipParts) >= 3 {
		partialIP = strings.Join(ipParts[:3], ".") // 처음 3옥텟만 사용
	}
	
	// 핑거프린트 생성을 위한 문자열 조합
	fingerprintData := fmt.Sprintf("%s|%s|%s|%s|%s", 
		userAgent, partialIP, browser, os, device)
	
	// SHA-256 해시 생성
	hash := sha256.Sum256([]byte(fingerprintData))
	return fmt.Sprintf("%x", hash)[:16] // 처음 16자리만 사용
}

// CompareFingerprints는 두 핑거프린트가 같은 디바이스인지 비교합니다.
func (g *DeviceFingerprintGenerator) CompareFingerprints(fp1, fp2 *models.DeviceFingerprint) float64 {
	if fp1 == nil || fp2 == nil {
		return 0.0
	}
	
	score := 0.0
	totalChecks := 5.0
	
	// User-Agent 비교 (가중치: 30%)
	if fp1.UserAgent == fp2.UserAgent {
		score += 1.5
	} else if g.similarUserAgent(fp1.UserAgent, fp2.UserAgent) {
		score += 0.5
	}
	
	// 브라우저 비교 (가중치: 25%)
	if fp1.Browser == fp2.Browser {
		score += 1.25
	}
	
	// OS 비교 (가중치: 25%)
	if fp1.OS == fp2.OS {
		score += 1.25
	}
	
	// 디바이스 타입 비교 (가중치: 15%)
	if fp1.Device == fp2.Device {
		score += 0.75
	}
	
	// IP 주소 비교 (가중치: 5% - 네트워크 변경 고려)
	if g.similarIPAddress(fp1.IPAddress, fp2.IPAddress) {
		score += 0.25
	}
	
	return score / totalChecks
}

// similarUserAgent는 User-Agent 문자열의 유사성을 검사합니다.
func (g *DeviceFingerprintGenerator) similarUserAgent(ua1, ua2 string) bool {
	// 간단한 유사성 검사 - 브라우저 버전 차이는 허용
	ua1Parts := strings.Fields(ua1)
	ua2Parts := strings.Fields(ua2)
	
	if len(ua1Parts) != len(ua2Parts) {
		return false
	}
	
	similarCount := 0
	for i := range ua1Parts {
		if ua1Parts[i] == ua2Parts[i] {
			similarCount++
		}
	}
	
	// 80% 이상 유사하면 같은 것으로 판단
	return float64(similarCount)/float64(len(ua1Parts)) >= 0.8
}

// similarIPAddress는 IP 주소의 유사성을 검사합니다.
func (g *DeviceFingerprintGenerator) similarIPAddress(ip1, ip2 string) bool {
	// 같은 서브넷인지 확인 (처음 3옥텟이 같은지)
	parts1 := strings.Split(ip1, ".")
	parts2 := strings.Split(ip2, ".")
	
	if len(parts1) < 3 || len(parts2) < 3 {
		return ip1 == ip2
	}
	
	return parts1[0] == parts2[0] && parts1[1] == parts2[1] && parts1[2] == parts2[2]
}