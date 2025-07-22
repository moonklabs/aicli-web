package session

import (
	"net"
	"path/filepath"

	"github.com/oschwald/geoip2-golang"
)

// GeoIPService는 지리적 위치 정보를 제공합니다.
type GeoIPService struct {
	cityDB    *geoip2.Reader
	countryDB *geoip2.Reader
}

// LocationInfo는 지리적 위치 정보를 담는 구조체입니다.
type LocationInfo struct {
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	City        string  `json:"city"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	TimeZone    string  `json:"timezone"`
}

// NewGeoIPService는 새로운 GeoIP 서비스를 생성합니다.
func NewGeoIPService(geoipDataDir string) (*GeoIPService, error) {
	service := &GeoIPService{}
	
	// City 데이터베이스 로드 (선택적)
	cityDBPath := filepath.Join(geoipDataDir, "GeoLite2-City.mmdb")
	if cityDB, err := geoip2.Open(cityDBPath); err == nil {
		service.cityDB = cityDB
	}
	
	// Country 데이터베이스 로드 (fallback)
	countryDBPath := filepath.Join(geoipDataDir, "GeoLite2-Country.mmdb")
	if countryDB, err := geoip2.Open(countryDBPath); err == nil {
		service.countryDB = countryDB
	}
	
	// 최소한 Country DB는 있어야 함
	if service.cityDB == nil && service.countryDB == nil {
		return nil, ErrGeoIPDatabaseNotFound
	}
	
	return service, nil
}

// GetLocationInfo는 IP 주소로부터 지리적 위치 정보를 조회합니다.
func (g *GeoIPService) GetLocationInfo(ipAddress string) (*LocationInfo, error) {
	ip := net.ParseIP(ipAddress)
	if ip == nil {
		return nil, ErrInvalidIPAddress
	}
	
	// 로컬 IP 주소인지 확인
	if g.isLocalIP(ip) {
		return &LocationInfo{
			Country:     "Local",
			CountryCode: "LOCAL",
			City:        "Local",
			Latitude:    0,
			Longitude:   0,
			TimeZone:    "Local",
		}, nil
	}
	
	// City 데이터베이스가 있으면 우선 사용
	if g.cityDB != nil {
		if record, err := g.cityDB.City(ip); err == nil {
			return &LocationInfo{
				Country:     record.Country.Names["en"],
				CountryCode: record.Country.IsoCode,
				City:        record.City.Names["en"],
				Latitude:    record.Location.Latitude,
				Longitude:   record.Location.Longitude,
				TimeZone:    record.Location.TimeZone,
			}, nil
		}
	}
	
	// Country 데이터베이스 사용
	if g.countryDB != nil {
		if record, err := g.countryDB.Country(ip); err == nil {
			return &LocationInfo{
				Country:     record.Country.Names["en"],
				CountryCode: record.Country.IsoCode,
				City:        "",
				Latitude:    0,
				Longitude:   0,
				TimeZone:    "",
			}, nil
		}
	}
	
	return nil, ErrGeoIPLookupFailed
}

// isLocalIP는 IP 주소가 로컬 주소인지 확인합니다.
func (g *GeoIPService) isLocalIP(ip net.IP) bool {
	if ip.IsLoopback() {
		return true
	}
	
	// RFC1918 사설 IP 주소 범위
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
	}
	
	for _, cidr := range privateRanges {
		_, private, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if private.Contains(ip) {
			return true
		}
	}
	
	return false
}

// CalculateDistance는 두 지점 간의 거리를 계산합니다 (킬로미터 단위).
func (g *GeoIPService) CalculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371 // 지구 반지름 (킬로미터)
	
	// 위도와 경도를 라디안으로 변환
	lat1Rad := lat1 * 0.017453292519943295 // π/180
	lon1Rad := lon1 * 0.017453292519943295
	lat2Rad := lat2 * 0.017453292519943295
	lon2Rad := lon2 * 0.017453292519943295
	
	// Haversine 공식
	deltaLat := lat2Rad - lat1Rad
	deltaLon := lon2Rad - lon1Rad
	
	a := 0.5 - 0.5*cosine(deltaLat) + cosine(lat1Rad)*cosine(lat2Rad)*(1-cosine(deltaLon))/2
	
	return earthRadius * 2 * arcsine(sqrt(a))
}

// 간단한 삼각함수 함수들
func cosine(x float64) float64 {
	// 간단한 Taylor 급수 근사
	x2 := x * x
	return 1 - x2/2 + x2*x2/24 - x2*x2*x2/720
}

func sqrt(x float64) float64 {
	if x < 0 {
		return 0
	}
	
	// Newton's method
	z := x
	for i := 0; i < 10; i++ {
		z = (z + x/z) / 2
	}
	return z
}

func arcsine(x float64) float64 {
	if x < -1 || x > 1 {
		return 0
	}
	
	// Taylor 급수 근사
	result := x
	term := x
	x2 := x * x
	
	for i := 1; i < 10; i++ {
		term = term * x2 * (2*float64(i)-1) * (2*float64(i)-1) / (2*float64(i)) / (2*float64(i)+1)
		result += term
	}
	
	return result
}

// Close는 GeoIP 데이터베이스를 닫습니다.
func (g *GeoIPService) Close() error {
	var err error
	
	if g.cityDB != nil {
		if closeErr := g.cityDB.Close(); closeErr != nil {
			err = closeErr
		}
	}
	
	if g.countryDB != nil {
		if closeErr := g.countryDB.Close(); closeErr != nil {
			err = closeErr
		}
	}
	
	return err
}