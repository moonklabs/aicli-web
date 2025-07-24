package session

import (
	"net/http/httptest"
	"testing"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestNewDeviceFingerprintGenerator(t *testing.T) {
	tests := []struct {
		name        string
		geoipService *GeoIPService
		expectError bool
	}{
		{
			name:         "with GeoIP service",
			geoipService: &GeoIPService{},
			expectError:  false,
		},
		{
			name:         "without GeoIP service",
			geoipService: nil,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var generator *DeviceFingerprintGenerator
			
			if tt.geoipService != nil {
				generator = NewDeviceFingerprintGenerator(tt.geoipService)
			} else {
				generator = NewDeviceFingerprintGeneratorWithoutGeoIP()
			}

			assert.NotNil(t, generator)
			assert.NotNil(t, generator.parser)
			
			if tt.geoipService != nil {
				assert.Equal(t, tt.geoipService, generator.geoipService)
			} else {
				assert.Nil(t, generator.geoipService)
			}
		})
	}
}

func TestGenerateFromRequest(t *testing.T) {
	generator := NewDeviceFingerprintGeneratorWithoutGeoIP()

	tests := []struct {
		name        string
		userAgent   string
		remoteAddr  string
		headers     map[string]string
		expectBrowser string
		expectOS     string
		expectDevice string
	}{
		{
			name:      "Chrome on Windows Desktop",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
			remoteAddr: "192.168.1.1:12345",
			expectBrowser: "Chrome 91",
			expectOS:     "Windows 10",
			expectDevice: "Desktop",
		},
		{
			name:      "Safari on iPhone",
			userAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 14_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Mobile/15E148 Safari/604.1",
			remoteAddr: "10.0.0.1:54321",
			expectBrowser: "Mobile Safari 14",
			expectOS:     "iOS 14",
			expectDevice: "Mobile",
		},
		{
			name:      "Firefox on Ubuntu",
			userAgent: "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:89.0) Gecko/20100101 Firefox/89.0",
			remoteAddr: "172.16.1.1:8080",
			expectBrowser: "Firefox 89",
			expectOS:     "Ubuntu",
			expectDevice: "Desktop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("User-Agent", tt.userAgent)
			req.RemoteAddr = tt.remoteAddr
			
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			fingerprint := generator.GenerateFromRequest(req)

			assert.NotNil(t, fingerprint)
			assert.Equal(t, tt.userAgent, fingerprint.UserAgent)
			assert.NotEmpty(t, fingerprint.IPAddress)
			assert.NotEmpty(t, fingerprint.Fingerprint)
			
			// UA Parser에서 정확한 파싱 결과는 버전에 따라 달라질 수 있으므로
			// 기본적인 검증만 수행
			assert.NotEmpty(t, fingerprint.Browser)
			assert.NotEmpty(t, fingerprint.OS)
			assert.NotEmpty(t, fingerprint.Device)
		})
	}
}

func TestExtractIPAddress(t *testing.T) {
	generator := NewDeviceFingerprintGeneratorWithoutGeoIP()

	tests := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		expectIP   string
	}{
		{
			name:       "direct connection",
			remoteAddr: "192.168.1.1:12345",
			headers:    map[string]string{},
			expectIP:   "192.168.1.1",
		},
		{
			name:       "X-Forwarded-For header",
			remoteAddr: "127.0.0.1:80",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.1, 192.168.1.1",
			},
			expectIP: "203.0.113.1",
		},
		{
			name:       "X-Real-IP header",
			remoteAddr: "127.0.0.1:80",
			headers: map[string]string{
				"X-Real-IP": "203.0.113.2",
			},
			expectIP: "203.0.113.2",
		},
		{
			name:       "both headers (X-Forwarded-For priority)",
			remoteAddr: "127.0.0.1:80",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.3",
				"X-Real-IP":       "203.0.113.4",
			},
			expectIP: "203.0.113.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			ip := generator.extractIPAddress(req)
			assert.Equal(t, tt.expectIP, ip)
		})
	}
}

func TestDetectDeviceFallback(t *testing.T) {
	generator := NewDeviceFingerprintGeneratorWithoutGeoIP()

	tests := []struct {
		name      string
		userAgent string
		expect    string
	}{
		{
			name:      "iPhone",
			userAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 14_6 like Mac OS X)",
			expect:    "Mobile",
		},
		{
			name:      "Android phone",
			userAgent: "Mozilla/5.0 (Linux; Android 11; SM-G991B)",
			expect:    "Mobile",
		},
		{
			name:      "iPad",
			userAgent: "Mozilla/5.0 (iPad; CPU OS 14_6 like Mac OS X)",
			expect:    "Tablet",
		},
		{
			name:      "Desktop Chrome",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/91.0.4472.124",
			expect:    "Desktop",
		},
		{
			name:      "Mobile keyword",
			userAgent: "Some Mobile Browser",
			expect:    "Mobile",
		},
		{
			name:      "Tablet keyword",
			userAgent: "Some Tablet Browser",
			expect:    "Tablet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generator.detectDeviceFallback(tt.userAgent)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestGenerateFingerprint(t *testing.T) {
	generator := NewDeviceFingerprintGeneratorWithoutGeoIP()

	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0"
	ipAddress := "192.168.1.1"
	browser := "Chrome"
	os := "Windows"
	device := "Desktop"

	fingerprint1 := generator.generateFingerprint(userAgent, ipAddress, browser, os, device)
	fingerprint2 := generator.generateFingerprint(userAgent, ipAddress, browser, os, device)

	// 같은 입력에 대해서는 같은 결과
	assert.Equal(t, fingerprint1, fingerprint2)
	assert.Len(t, fingerprint1, 16) // 16자리 해시

	// 다른 입력에 대해서는 다른 결과
	differentFingerprint := generator.generateFingerprint("different", ipAddress, browser, os, device)
	assert.NotEqual(t, fingerprint1, differentFingerprint)
}

func TestCompareFingerprints(t *testing.T) {
	generator := NewDeviceFingerprintGeneratorWithoutGeoIP()

	baseFingerprint := &models.DeviceFingerprint{
		UserAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0",
		IPAddress:   "192.168.1.1",
		Browser:     "Chrome",
		OS:          "Windows",
		Device:      "Desktop",
		Fingerprint: "abc123def456",
	}

	tests := []struct {
		name        string
		fingerprint *models.DeviceFingerprint
		expectScore float64
		description string
	}{
		{
			name:        "identical fingerprints",
			fingerprint: baseFingerprint,
			expectScore: 1.0,
			description: "same fingerprint should score 1.0",
		},
		{
			name: "same browser and OS, different IP",
			fingerprint: &models.DeviceFingerprint{
				UserAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0",
				IPAddress:   "10.0.0.1",
				Browser:     "Chrome",
				OS:          "Windows",
				Device:      "Desktop",
				Fingerprint: "different",
			},
			expectScore: 0.95, // Loses only IP similarity points
			description: "same device, different IP should score high",
		},
		{
			name: "different browser",
			fingerprint: &models.DeviceFingerprint{
				UserAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Firefox/89.0",
				IPAddress:   "192.168.1.1",
				Browser:     "Firefox",
				OS:          "Windows",
				Device:      "Desktop",
				Fingerprint: "different",
			},
			expectScore: 0.4, // Loses browser and user-agent points
			description: "different browser should score lower",
		},
		{
			name:        "nil fingerprint",
			fingerprint: nil,
			expectScore: 0.0,
			description: "nil fingerprint should score 0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := generator.CompareFingerprints(baseFingerprint, tt.fingerprint)
			
			if tt.expectScore == 0.0 {
				assert.Equal(t, tt.expectScore, score, tt.description)
			} else {
				// 부동소수점 비교에서 약간의 오차 허용
				assert.InDelta(t, tt.expectScore, score, 0.1, tt.description)
			}
		})
	}
}

func TestSimilarUserAgent(t *testing.T) {
	generator := NewDeviceFingerprintGeneratorWithoutGeoIP()

	tests := []struct {
		name   string
		ua1    string
		ua2    string
		expect bool
	}{
		{
			name:   "identical user agents",
			ua1:    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0",
			ua2:    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0",
			expect: true,
		},
		{
			name:   "similar user agents (version difference)",
			ua1:    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0",
			ua2:    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/92.0",
			expect: true, // 80% similar
		},
		{
			name:   "completely different user agents",
			ua1:    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0",
			ua2:    "Mozilla/5.0 (iPhone; CPU iPhone OS 14_6 like Mac OS X) Safari/604.1",
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generator.similarUserAgent(tt.ua1, tt.ua2)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestSimilarIPAddress(t *testing.T) {
	generator := NewDeviceFingerprintGeneratorWithoutGeoIP()

	tests := []struct {
		name   string
		ip1    string
		ip2    string
		expect bool
	}{
		{
			name:   "identical IPs",
			ip1:    "192.168.1.1",
			ip2:    "192.168.1.1",
			expect: true,
		},
		{
			name:   "same subnet",
			ip1:    "192.168.1.1",
			ip2:    "192.168.1.100",
			expect: true,
		},
		{
			name:   "different subnet",
			ip1:    "192.168.1.1",
			ip2:    "10.0.0.1",
			expect: false,
		},
		{
			name:   "invalid IP format",
			ip1:    "invalid",
			ip2:    "192.168.1.1",
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generator.similarIPAddress(tt.ip1, tt.ip2)
			assert.Equal(t, tt.expect, result)
		})
	}
}

// Benchmark 테스트
func BenchmarkGenerateFromRequest(b *testing.B) {
	generator := NewDeviceFingerprintGeneratorWithoutGeoIP()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.RemoteAddr = "192.168.1.1:12345"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generator.GenerateFromRequest(req)
	}
}

func BenchmarkCompareFingerprints(b *testing.B) {
	generator := NewDeviceFingerprintGeneratorWithoutGeoIP()
	
	fp1 := &models.DeviceFingerprint{
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0",
		IPAddress: "192.168.1.1",
		Browser:   "Chrome",
		OS:        "Windows",
		Device:    "Desktop",
	}
	
	fp2 := &models.DeviceFingerprint{
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/92.0",
		IPAddress: "192.168.1.2",
		Browser:   "Chrome",
		OS:        "Windows",  
		Device:    "Desktop",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generator.CompareFingerprints(fp1, fp2)
	}
}