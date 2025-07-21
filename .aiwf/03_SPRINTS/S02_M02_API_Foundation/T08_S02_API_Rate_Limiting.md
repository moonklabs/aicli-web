---
task_id: T08_S02
task_name: API Rate Limiting and Throttling
status: pending
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

- [ ] Rate Limit 미들웨어가 모든 API 엔드포인트에 적용
- [ ] X-RateLimit-* 헤더가 응답에 포함
- [ ] 429 Too Many Requests 응답 처리
- [ ] 사용자별 차등 제한 적용 가능
- [ ] 화이트리스트 IP 지원
- [ ] 설정 파일로 정책 관리
- [ ] 메트릭 수집 가능
- [ ] 테스트 작성

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

- [ ] Rate Limiter 인터페이스 설계
- [ ] 메모리 기반 구현
- [ ] Rate Limit 미들웨어 구현
- [ ] 설정 구조체 정의
- [ ] 사용자별 제한 로직
- [ ] 응답 헤더 처리
- [ ] 통합 테스트 작성

## 관련 링크

- Go Rate Limiter: https://golang.org/x/time/rate
- Token Bucket Algorithm: https://en.wikipedia.org/wiki/Token_bucket
- Rate Limiting Best Practices: https://cloud.google.com/architecture/rate-limiting-strategies-techniques