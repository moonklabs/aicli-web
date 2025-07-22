package session

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewGeoIPService(t *testing.T) {
	tests := []struct {
		name        string
		geoipDataDir string
		expectError bool
		description string
	}{
		{
			name:        "nonexistent directory",
			geoipDataDir: "/nonexistent/path",
			expectError: true,
			description: "should return error when GeoIP database files are not found",
		},
		{
			name:        "empty directory path",
			geoipDataDir: "",
			expectError: true,
			description: "should return error when directory path is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewGeoIPService(tt.geoipDataDir)
			
			if tt.expectError {
				assert.Error(t, err, tt.description)
				assert.Nil(t, service, "service should be nil when error occurs")
			} else {
				assert.NoError(t, err, tt.description)
				assert.NotNil(t, service, "service should not be nil when no error")
			}
		})
	}
}

func TestIsLocalIP(t *testing.T) {
	// Mock service for testing private methods
	service := &GeoIPService{}

	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{
			name:     "IPv4 loopback",
			ip:       "127.0.0.1",
			expected: true,
		},
		{
			name:     "IPv6 loopback",
			ip:       "::1",
			expected: true,
		},
		{
			name:     "private IP 192.168.x.x",
			ip:       "192.168.1.1",
			expected: true,
		},
		{
			name:     "private IP 10.x.x.x",
			ip:       "10.0.0.1",
			expected: true,
		},
		{
			name:     "private IP 172.16.x.x",
			ip:       "172.16.0.1",
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
			ip := net.ParseIP(tt.ip)
			require.NotNil(t, ip, "IP should be valid")
			
			result := service.isLocalIP(ip)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetLocationInfoLocalIP(t *testing.T) {
	// Mock service without actual database files
	service := &GeoIPService{}

	tests := []struct {
		name     string
		ip       string
		expected *LocationInfo
	}{
		{
			name: "loopback IP",
			ip:   "127.0.0.1",
			expected: &LocationInfo{
				Country:     "Local",
				CountryCode: "LOCAL",
				City:        "Local",
				Latitude:    0,
				Longitude:   0,
				TimeZone:    "Local",
			},
		},
		{
			name: "private IP",
			ip:   "192.168.1.1",
			expected: &LocationInfo{
				Country:     "Local",
				CountryCode: "LOCAL",
				City:        "Local",
				Latitude:    0,
				Longitude:   0,
				TimeZone:    "Local",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.GetLocationInfo(tt.ip)
			
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetLocationInfoInvalidIP(t *testing.T) {
	service := &GeoIPService{}

	tests := []string{
		"invalid-ip",
		"256.256.256.256",
		"",
		"not-an-ip",
	}

	for _, ip := range tests {
		t.Run("invalid IP: "+ip, func(t *testing.T) {
			result, err := service.GetLocationInfo(ip)
			
			assert.Error(t, err)
			assert.Equal(t, ErrInvalidIPAddress, err)
			assert.Nil(t, result)
		})
	}
}

func TestCalculateDistance(t *testing.T) {
	service := &GeoIPService{}

	tests := []struct {
		name      string
		lat1, lon1 float64
		lat2, lon2 float64
		expected  float64
		tolerance float64
	}{
		{
			name:      "same location",
			lat1:      37.7749, lon1: -122.4194, // San Francisco
			lat2:      37.7749, lon2: -122.4194, // San Francisco
			expected:  0.0,
			tolerance: 0.1,
		},
		{
			name:      "SF to LA approximate",
			lat1:      37.7749, lon1: -122.4194, // San Francisco
			lat2:      34.0522, lon2: -118.2437, // Los Angeles
			expected:  559.0, // ~559km
			tolerance: 50.0,  // 50km tolerance for approximation
		},
		{
			name:      "zero coordinates",
			lat1:      0, lon1: 0,
			lat2:      0, lon2: 0,
			expected:  0.0,
			tolerance: 0.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			distance := service.CalculateDistance(tt.lat1, tt.lon1, tt.lat2, tt.lon2)
			assert.InDelta(t, tt.expected, distance, tt.tolerance)
		})
	}
}

func TestMathHelperFunctions(t *testing.T) {
	tests := []struct {
		name     string
		function func() float64
		expected float64
		tolerance float64
	}{
		{
			name: "cosine(0)",
			function: func() float64 { return cosine(0) },
			expected: 1.0,
			tolerance: 0.01,
		},
		{
			name: "sqrt(4)",
			function: func() float64 { return sqrt(4) },
			expected: 2.0,
			tolerance: 0.01,
		},
		{
			name: "sqrt(0)",
			function: func() float64 { return sqrt(0) },
			expected: 0.0,
			tolerance: 0.01,
		},
		{
			name: "sqrt negative",
			function: func() float64 { return sqrt(-1) },
			expected: 0.0,
			tolerance: 0.01,
		},
		{
			name: "arcsine(0)",
			function: func() float64 { return arcsine(0) },
			expected: 0.0,
			tolerance: 0.01,
		},
		{
			name: "arcsine out of range",
			function: func() float64 { return arcsine(2) },
			expected: 0.0,
			tolerance: 0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.function()
			assert.InDelta(t, tt.expected, result, tt.tolerance)
		})
	}
}

func TestClose(t *testing.T) {
	service := &GeoIPService{}
	
	// Close should not panic even with nil databases
	err := service.Close()
	assert.NoError(t, err, "Close should not return error with nil databases")
}

// Benchmark tests
func BenchmarkCalculateDistance(b *testing.B) {
	service := &GeoIPService{}
	lat1, lon1 := 37.7749, -122.4194 // San Francisco
	lat2, lon2 := 34.0522, -118.2437  // Los Angeles

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.CalculateDistance(lat1, lon1, lat2, lon2)
	}
}

func BenchmarkIsLocalIP(b *testing.B) {
	service := &GeoIPService{}
	ip := net.ParseIP("192.168.1.1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.isLocalIP(ip)
	}
}

func BenchmarkMathFunctions(b *testing.B) {
	b.Run("cosine", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cosine(0.5)
		}
	})

	b.Run("sqrt", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			sqrt(16.0)
		}
	})

	b.Run("arcsine", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			arcsine(0.5)
		}
	})
}