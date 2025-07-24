package session

import (
	"math"
	"net"
	"os"
	"sync"

	"github.com/aicli/aicli-web/internal/models"
	"github.com/oschwald/geoip2-golang"
)

// GeoIPService는 GeoIP 기능을 제공하는 서비스입니다.
type GeoIPService struct {
	db     *geoip2.Reader
	mutex  sync.RWMutex
	closed bool
}

// GeoIPConfig는 GeoIP 서비스 설정입니다.
type GeoIPConfig struct {
	DatabasePath string `json:"database_path" yaml:"database_path"`
	MaxMindAPIKey string `json:"maxmind_api_key" yaml:"maxmind_api_key"`
}

// NewGeoIPService는 새로운 GeoIP 서비스를 생성합니다.
func NewGeoIPService(config *GeoIPConfig) (*GeoIPService, error) {
	if config == nil || config.DatabasePath == "" {
		return nil, ErrGeoIPDatabaseNotFound
	}

	// GeoIP 데이터베이스 파일 확인
	if _, err := os.Stat(config.DatabasePath); os.IsNotExist(err) {
		return nil, ErrGeoIPDatabaseNotFound
	}

	// GeoIP 데이터베이스 열기
	db, err := geoip2.Open(config.DatabasePath)
	if err != nil {
		return nil, ErrGeoIPDatabaseNotFound
	}

	return &GeoIPService{
		db:     db,
		closed: false,
	}, nil
}

// NewGeoIPServiceWithFallback는 fallback 메커니즘을 포함하는 GeoIP 서비스를 생성합니다.
func NewGeoIPServiceWithFallback(config *GeoIPConfig) *GeoIPService {
	service, err := NewGeoIPService(config)
	if err != nil {
		// 데이터베이스를 찾을 수 없는 경우 nil 서비스를 반환하여 fallback 처리
		return &GeoIPService{
			db:     nil,
			closed: true,
		}
	}
	return service
}

// Close는 GeoIP 데이터베이스를 닫습니다.
func (g *GeoIPService) Close() error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if g.db != nil && !g.closed {
		err := g.db.Close()
		g.closed = true
		return err
	}
	return nil
}

// GetLocationInfo는 IP 주소로부터 위치 정보를 조회합니다.
func (g *GeoIPService) GetLocationInfo(ipAddress string) (*models.LocationInfo, error) {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	if g.db == nil || g.closed {
		return nil, ErrGeoIPDatabaseNotFound
	}

	// IP 주소 파싱
	ip := net.ParseIP(ipAddress)
	if ip == nil {
		return nil, ErrInvalidIPAddress
	}

	// 로컬 IP 주소 처리 (GeoIP에서 조회할 수 없음)
	if ip.IsLoopback() || ip.IsPrivate() {
		return &models.LocationInfo{
			Country:     "Local",
			CountryCode: "LO",
			City:        "Local",
			Latitude:    0.0,
			Longitude:   0.0,
			TimeZone:    "UTC",
		}, nil
	}

	// GeoIP 조회
	record, err := g.db.City(ip)
	if err != nil {
		return nil, ErrGeoIPLookupFailed
	}

	// 언어 우선순위 설정 (한국어 > 영어)
	nameMap := record.City.Names
	cityName := ""
	if val, exists := nameMap["ko"]; exists && val != "" {
		cityName = val
	} else if val, exists := nameMap["en"]; exists && val != "" {
		cityName = val
	} else {
		cityName = "Unknown"
	}

	countryNameMap := record.Country.Names
	countryName := ""
	if val, exists := countryNameMap["ko"]; exists && val != "" {
		countryName = val
	} else if val, exists := countryNameMap["en"]; exists && val != "" {
		countryName = val
	} else {
		countryName = "Unknown"
	}

	// 시간대 정보 처리
	timezone := ""
	if record.Location.TimeZone != "" {
		timezone = record.Location.TimeZone
	} else {
		timezone = "UTC"
	}

	locationInfo := &models.LocationInfo{
		Country:     countryName,
		CountryCode: record.Country.IsoCode,
		City:        cityName,
		Latitude:    float64(record.Location.Latitude),
		Longitude:   float64(record.Location.Longitude),
		TimeZone:    timezone,
	}

	return locationInfo, nil
}

// isLocalIP은 IP가 로컬 IP인지 확인합니다.
func (g *GeoIPService) isLocalIP(ip net.IP) bool {
	// 로컬 IP 범위 체크
	privateIPBlocks := []string{
		"127.0.0.0/8",    // Loopback
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"::1/128",        // IPv6 loopback
		"fc00::/7",       // IPv6 unique local
		"fe80::/10",      // IPv6 link local
	}
	
	for _, block := range privateIPBlocks {
		_, cidr, err := net.ParseCIDR(block)
		if err == nil && cidr.Contains(ip) {
			return true
		}
	}
	
	return false
}

// CalculateDistance은 두 위치 사이의 거리를 계산합니다 (km 단위).
func (g *GeoIPService) CalculateDistance(loc1, loc2 *models.LocationInfo) float64 {
	if loc1 == nil || loc2 == nil {
		return 0
	}
	
	const R = 6371 // 지구 반지름 (km)
	
	lat1 := loc1.Latitude * math.Pi / 180
	lat2 := loc2.Latitude * math.Pi / 180
	deltaLat := (loc2.Latitude - loc1.Latitude) * math.Pi / 180
	deltaLon := (loc2.Longitude - loc1.Longitude) * math.Pi / 180
	
	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*
		math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
		
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	
	return R * c
}

// IsAvailable은 GeoIP 서비스가 사용 가능한지 확인합니다.
func (g *GeoIPService) IsAvailable() bool {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	return g.db != nil && !g.closed
}

// GetDatabaseInfo는 GeoIP 데이터베이스 정보를 반환합니다.
func (g *GeoIPService) GetDatabaseInfo() map[string]interface{} {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	if g.db == nil || g.closed {
		return map[string]interface{}{
			"available":  false,
			"build_date": nil,
			"version":    nil,
		}
	}

	metadata := g.db.Metadata()
	return map[string]interface{}{
		"available":       true,
		"build_date":      metadata.BuildEpoch,
		"version":         metadata.BinaryFormatMajorVersion,
		"database_type":   metadata.DatabaseType,
		"description":     metadata.Description,
		"record_size":     metadata.RecordSize,
		"node_count":      metadata.NodeCount,
	}
}

// CompareLocations는 두 위치가 얼마나 유사한지 비교합니다.
func (g *GeoIPService) CompareLocations(loc1, loc2 *models.LocationInfo) float64 {
	if loc1 == nil || loc2 == nil {
		return 0.0
	}

	score := 0.0
	totalChecks := 3.0

	// 국가 비교 (50% 가중치)
	if loc1.CountryCode == loc2.CountryCode {
		score += 1.5
	}

	// 도시 비교 (30% 가중치)
	if loc1.City == loc2.City {
		score += 0.9
	}

	// 시간대 비교 (20% 가중치)
	if loc1.TimeZone == loc2.TimeZone {
		score += 0.6
	}

	return score / totalChecks
}

// IsSignificantLocationChange는 위치 변경이 중요한지 판단합니다.
func (g *GeoIPService) IsSignificantLocationChange(oldLoc, newLoc *models.LocationInfo) bool {
	if oldLoc == nil || newLoc == nil {
		return false
	}

	// 국가가 다른 경우는 중요한 변경
	if oldLoc.CountryCode != newLoc.CountryCode {
		return true
	}

	// 시간대가 다른 경우도 중요한 변경으로 간주
	if oldLoc.TimeZone != newLoc.TimeZone {
		return true
	}

	return false
}

// ValidateIPAddress는 IP 주소가 유효한지 검증합니다.
func (g *GeoIPService) ValidateIPAddress(ipAddress string) error {
	ip := net.ParseIP(ipAddress)
	if ip == nil {
		return ErrInvalidIPAddress
	}
	return nil
}

// GetLocationFromMultipleIPs는 여러 IP 주소에서 가장 적합한 위치를 선택합니다.
func (g *GeoIPService) GetLocationFromMultipleIPs(ipAddresses []string) (*models.LocationInfo, error) {
	if len(ipAddresses) == 0 {
		return nil, ErrInvalidIPAddress
	}

	var bestLocation *models.LocationInfo
	var bestScore float64 = 0

	for _, ip := range ipAddresses {
		location, err := g.GetLocationInfo(ip)
		if err != nil {
			continue
		}

		// 로컬 IP는 건너뛰기
		if location.Country == "Local" {
			continue
		}

		// 더 구체적인 정보를 가진 위치를 선호
		score := g.calculateLocationScore(location)
		if score > bestScore {
			bestLocation = location
			bestScore = score
		}
	}

	if bestLocation == nil {
		return nil, ErrGeoIPLookupFailed
	}

	return bestLocation, nil
}

// calculateLocationScore는 위치 정보의 품질 점수를 계산합니다.
func (g *GeoIPService) calculateLocationScore(location *models.LocationInfo) float64 {
	score := 0.0

	// 국가 정보가 있으면 +1
	if location.Country != "" && location.Country != "Unknown" {
		score += 1.0
	}

	// 도시 정보가 있으면 +1
	if location.City != "" && location.City != "Unknown" {
		score += 1.0
	}

	// 좌표 정보가 있으면 +0.5
	if location.Latitude != 0.0 || location.Longitude != 0.0 {
		score += 0.5
	}

	// 시간대 정보가 있으면 +0.5
	if location.TimeZone != "" && location.TimeZone != "UTC" {
		score += 0.5
	}

	return score
}