package session

import "errors"

// 세션 관련 에러들
var (
	// ErrSessionNotFound는 세션을 찾을 수 없을 때 발생합니다.
	ErrSessionNotFound = errors.New("세션을 찾을 수 없습니다")
	
	// ErrSessionExpired는 세션이 만료되었을 때 발생합니다.
	ErrSessionExpired = errors.New("세션이 만료되었습니다")
	
	// ErrSessionInactive는 세션이 비활성 상태일 때 발생합니다.
	ErrSessionInactive = errors.New("세션이 비활성 상태입니다")
	
	// ErrConcurrentSessionLimitExceeded는 동시 세션 제한을 초과했을 때 발생합니다.
	ErrConcurrentSessionLimitExceeded = errors.New("동시 세션 제한을 초과했습니다")
	
	// ErrSuspiciousActivity는 의심스러운 활동이 감지되었을 때 발생합니다.
	ErrSuspiciousActivity = errors.New("의심스러운 활동이 감지되었습니다")
	
	// ErrDeviceNotRecognized는 인식되지 않은 디바이스에서 접근할 때 발생합니다.
	ErrDeviceNotRecognized = errors.New("인식되지 않은 디바이스입니다")
	
	// ErrLocationChanged는 위치가 변경되었을 때 발생합니다.
	ErrLocationChanged = errors.New("접속 위치가 변경되었습니다")
	
	// GeoIP 관련 에러들
	// ErrGeoIPDatabaseNotFound는 GeoIP 데이터베이스를 찾을 수 없을 때 발생합니다.
	ErrGeoIPDatabaseNotFound = errors.New("GeoIP 데이터베이스를 찾을 수 없습니다")
	
	// ErrInvalidIPAddress는 잘못된 IP 주소 형식일 때 발생합니다.
	ErrInvalidIPAddress = errors.New("잘못된 IP 주소 형식입니다")
	
	// ErrGeoIPLookupFailed는 GeoIP 조회가 실패했을 때 발생합니다.
	ErrGeoIPLookupFailed = errors.New("GeoIP 조회에 실패했습니다")
)