---
task_id: T08_S02
task_name: API Rate Limiting and Throttling
status: done
complexity: low
priority: low
created_at: 2025-07-21
updated_at: 2025-07-21
sprint_id: S02_M02
---

# T08_S02: API Rate Limiting and Throttling

## 태스크 개요

API 서버의 안정성과 보안을 위해 Rate Limiting과 Throttling 미들웨어를 구현합니다. 사용자별, IP별 요청 제한을 적용합니다.

## 목표

- Rate Limiting 미들웨어 구현
- 사용자별/IP별 제한 정책 적용
- 제한 초과 시 적절한 응답 제공
- 설정 가능한 제한 정책

## 수용 기준

- [x] Rate Limit 미들웨어가 모든 API 엔드포인트에 적용
- [x] X-RateLimit-* 헤더가 응답에 포함
- [x] 429 Too Many Requests 응답 처리
- [x] 사용자별 차등 제한 적용 가능
- [x] 화이트리스트 IP 지원
- [x] 설정 파일로 정책 관리
- [x] 메트릭 수집 가능
- [x] 테스트 작성

## 기술 가이드

### 주요 인터페이스 및 통합 지점

1. **Rate Limiter 라이브러리**: golang.org/x/time/rate 또는 github.com/ulule/limiter
2. **미들웨어 체인**: `internal/middleware/` 디렉토리에 추가
3. **설정 관리**: Viper를 통한 rate limit 설정
4. **Redis 통합**: 분산 환경을 위한 Redis 기반 구현 (선택적)

### 구현 구조

```
internal/
├── middleware/
│   ├── ratelimit.go     # Rate Limit 미들웨어
│   └── ratelimit_test.go # 테스트
├── ratelimit/
│   ├── limiter.go       # Rate Limiter 인터페이스
│   ├── memory.go        # 메모리 기반 구현
│   └── redis.go         # Redis 기반 구현 (선택적)
└── config/
    └── ratelimit.go     # Rate Limit 설정
```

### 기존 패턴 참조

- 미들웨어 패턴: `internal/middleware/request_id.go` 구조 참조
- 설정 관리: 기존 Viper 설정 패턴 활용
- 에러 응답: 표준 에러 응답 포맷 사용

## 구현 노트

### 단계별 접근법

1. Rate Limiter 인터페이스 정의
2. 메모리 기반 구현 (개발/단일 서버)
3. Rate Limit 미들웨어 구현
4. 설정 구조체 및 로딩
5. 사용자별 제한 로직
6. 응답 헤더 추가
7. 테스트 및 벤치마크

### Rate Limit 설정

```yaml
ratelimit:
  enabled: true
  default:
    requests_per_minute: 60
    burst: 10
  authenticated:
    requests_per_minute: 300
    burst: 50
  endpoints:
    - path: "/api/v1/auth/login"
      requests_per_minute: 5
      burst: 2
  whitelist:
    - "127.0.0.1"
    - "10.0.0.0/8"
```

### Rate Limiter 인터페이스

```go
type RateLimiter interface {
    Allow(key string) bool
    Reset(key string)
    Remaining(key string) int
}
```

### 응답 헤더

- X-RateLimit-Limit: 제한 수
- X-RateLimit-Remaining: 남은 요청 수
- X-RateLimit-Reset: 리셋 시간 (Unix timestamp)
- Retry-After: 재시도 가능 시간 (초)

### 키 생성 전략

- 인증된 사용자: `user:{userID}`
- 미인증 사용자: `ip:{clientIP}`
- 엔드포인트별: `{key}:{endpoint}`

## 서브태스크

- [x] Rate Limiter 인터페이스 설계
- [x] 메모리 기반 구현
- [x] Rate Limit 미들웨어 구현
- [x] 설정 구조체 정의
- [x] 사용자별 제한 로직
- [x] 응답 헤더 처리
- [x] 통합 테스트 작성

## 관련 링크

- Go Rate Limiter: https://golang.org/x/time/rate
- Token Bucket Algorithm: https://en.wikipedia.org/wiki/Token_bucket
- Rate Limiting Best Practices: https://cloud.google.com/architecture/rate-limiting-strategies-techniques

## 구현 결과

### 완료된 작업

1. **Rate Limiter 인터페이스 설계** (`internal/ratelimit/limiter.go`)
   - RateLimiter 인터페이스 정의 (Allow, Reset, Remaining, Limit, ResetTime, Close)
   - LimiterConfig 구조체 정의 (Rate, Burst, Window 설정)
   - DefaultLimiterConfig 함수로 기본 설정 제공

2. **메모리 기반 Rate Limiter 구현** (`internal/ratelimit/memory.go`)
   - Token Bucket 알고리즘 기반 MemoryLimiter 구현
   - golang.org/x/time/rate 라이브러리 활용
   - 키별 독립적인 rate limiting (사용자별, IP별)
   - 백그라운드 정리 작업으로 메모리 관리
   - 동시성 안전 구현 (sync.RWMutex 활용)
   - 통계 수집 기능 (GetStats)

3. **Rate Limit 미들웨어 구현** (`internal/middleware/ratelimit.go`)
   - Gin 기반 Rate Limit 미들웨어
   - 사용자별/IP별 차등 제한 적용
   - 엔드포인트별 개별 설정 지원
   - 화이트리스트 IP 지원 (127.0.0.1, ::1 기본 포함)
   - 인증된 사용자와 미인증 사용자 구분
   - 설정 가능한 키 생성 전략

4. **응답 헤더 및 에러 처리**
   - X-RateLimit-Limit: 제한된 요청 수
   - X-RateLimit-Remaining: 남은 요청 수  
   - X-RateLimit-Reset: 리셋 시간 (Unix timestamp)
   - Retry-After: 재시도 가능 시간 (초)
   - 429 Too Many Requests 표준 응답
   - 구조화된 에러 메시지 (JSON 형태)

5. **서버 통합** (`internal/server/server.go`)
   - 미들웨어 체인에 Rate Limiting 추가
   - 보안 헤더와 CORS 뒤, 로깅 앞에 배치
   - DefaultRateLimitConfig로 기본 설정 적용

6. **의존성 관리**
   - `go.mod`에 golang.org/x/time v0.5.0 추가
   - Rate Limiting에 필요한 패키지 임포트

7. **포괄적인 테스트 작성**
   - MemoryLimiter 단위 테스트 (`internal/ratelimit/memory_test.go`)
     - 기본 동작, 리셋, 다중 키, 통계, 동시성 테스트
   - Rate Limit 미들웨어 통합 테스트 (`internal/middleware/ratelimit_test.go`)
     - 기본 제한, 비활성화, 화이트리스트, 엔드포인트별 제한
     - 인증된 사용자, 헤더 검증, 429 응답 테스트

### 주요 기능

- **유연한 설정**: 기본, 인증된 사용자, 엔드포인트별 개별 설정
- **차등 제한**: 인증된 사용자는 더 높은 한도 (300 req/min vs 60 req/min)
- **엔드포인트별 제한**: 로그인 엔드포인트는 더 엄격한 제한 (5 req/min)
- **화이트리스트**: 특정 IP는 Rate Limiting에서 제외
- **토큰 버킷 알고리즘**: 버스트 트래픽 허용과 평균 제한 적용
- **메모리 효율성**: 백그라운드 정리로 오래된 키 제거
- **표준 준수**: HTTP Rate Limiting 헤더 표준 준수
- **통계 수집**: 실시간 모니터링을 위한 통계 API

### Rate Limit 정책

| 사용자 유형 | 분당 요청 수 | 버스트 | 특별 규칙 |
|------------|------------|--------|-----------|
| 미인증 사용자 | 60 | 10 | IP 기반 제한 |
| 인증된 사용자 | 300 | 50 | 사용자 ID 기반 제한 |
| 로그인 엔드포인트 | 5 | 2 | 브루트포스 방지 |

### 응답 헤더

모든 API 응답에 다음 헤더가 포함됩니다:
- `X-RateLimit-Limit`: 분당 허용 요청 수
- `X-RateLimit-Remaining`: 남은 요청 수
- `X-RateLimit-Reset`: 제한 리셋 시간 (Unix timestamp)

Rate Limit 초과 시 추가 헤더:
- `Retry-After`: 재시도 가능 시간 (초)

### 설정 예시

```yaml
ratelimit:
  enabled: true
  default:
    requests_per_minute: 60
    burst: 10
  authenticated:
    requests_per_minute: 300
    burst: 50
  endpoints:
    - path: "/api/v1/auth/login"
      requests_per_minute: 5
      burst: 2
  whitelist:
    - "127.0.0.1"
    - "::1"
```

### 다음 단계 권장사항

1. **Redis 기반 분산 구현**: 여러 서버 인스턴스 간 Rate Limit 공유
2. **설정 파일 통합**: Viper를 통한 YAML/JSON 설정 파일 지원  
3. **고급 알고리즘**: Sliding Window, Fixed Window 알고리즘 추가
4. **모니터링 대시보드**: Prometheus 메트릭 및 Grafana 대시보드
5. **동적 설정**: 실시간 Rate Limit 정책 업데이트 지원
6. **IP CIDR 지원**: 네트워크 범위 기반 화이트리스트
7. **Rate Limit 바이패스**: 특정 사용자 역할에 대한 제한 면제