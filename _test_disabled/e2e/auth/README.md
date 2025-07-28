# Authentication E2E Tests

인증 시스템의 End-to-End 테스트 스위트입니다.

## 테스트 구조

### 1. 기본 인증 플로우 테스트 (`auth_flow_test.go`)
- 완전한 로그인-로그아웃 플로우
- JWT 토큰 관리 (생성, 갱신, 만료)
- 세션 관리 (타임아웃, 블랙리스트, 동시 세션)
- 인증 실패 시나리오

### 2. 보안 테스트 (`auth_security_test.go`)
- 보안 헤더 검증
- CSRF 보호 테스트
- Rate Limiting 동작 확인
- 브루트포스 공격 방지

### 3. 헬퍼 함수 (`auth_helpers.go`)
- Mock 핸들러 구현
- API 호출 헬퍼 메서드
- 테스트 유틸리티 함수

### 4. JWT 테스트 확장 (`internal/auth/jwt_test_extensions.go`)
- 테스트용 JWT Manager 확장 기능
- 사용자 정의 만료 시간 토큰 생성
- 토큰 검증 유틸리티

### 5. 프론트엔드 테스트 (`web/src/test/e2e/auth.test.ts`)
- Vue.js 컴포넌트 테스트
- 상태 관리 테스트
- OAuth 플로우 테스트
- 접근성 테스트

## 테스트 실행

### 백엔드 테스트 실행
```bash
# 개별 테스트 파일 실행
go test -v ./_test_disabled/e2e/auth/

# 특정 테스트 함수 실행
go test -v ./_test_disabled/e2e/auth/ -run TestBasicAuthenticationFlow

# 보안 테스트만 실행
go test -v ./_test_disabled/e2e/auth/ -run TestSecurity
```

### 프론트엔드 테스트 실행
```bash
cd web
npm run test:run -- auth.test.ts
```

### 통합 테스트 실행 (CI/CD에서)
```bash
# 모든 E2E 테스트 실행
make test-e2e

# 커버리지 포함 실행
make test-e2e-coverage
```

## 테스트 커버리지

현재 구현된 테스트는 다음 시나리오를 커버합니다:

### ✅ 완료된 테스트
- [x] 기본 로그인/로그아웃 플로우
- [x] JWT 토큰 갱신
- [x] 세션 타임아웃 및 만료 처리
- [x] 인증 실패 시나리오 (잘못된 credentials, 만료된 토큰)
- [x] 보안 헤더 및 CSRF 보호
- [x] Rate Limiting 동작
- [x] 다중 로그인 세션
- [x] 토큰 블랙리스트
- [x] 동시 세션 격리

### 🔄 진행 중인 테스트
- [ ] OAuth 소셜 로그인 E2E
- [ ] RBAC 권한 시스템 통합

### 📋 계획된 테스트
- [ ] 모바일 웹 인증 플로우
- [ ] 2FA (Two-Factor Authentication)
- [ ] SSO (Single Sign-On) 통합

## 테스트 환경 요구사항

### 최소 요구사항
- Go 1.21+
- Node.js 18+
- 메모리: 최소 2GB (동시 테스트 실행 시)

### 선택적 요구사항
- Docker (컨테이너 기반 테스트 시)
- Redis (실제 세션 관리 테스트 시)

## 테스트 데이터

### 테스트 사용자 계정
```go
testUsers := map[string]string{
    "admin": "admin123",  // 관리자 계정
    "user":  "user123",   // 일반 사용자 계정
    "test":  "test123",   // 테스트 전용 계정
}
```

### Mock JWT 설정
```go
jwtConfig := &config.Config{
    JWT: config.JWTConfig{
        Secret:               "test-secret-key-for-e2e-testing",
        AccessTokenExpiry:    15 * time.Minute,
        RefreshTokenExpiry:   24 * time.Hour,
    },
}
```

## 디버깅

### 테스트 실패 시 확인사항
1. JWT 설정이 올바른지 확인
2. 테스트 데이터베이스가 초기화되었는지 확인
3. Rate Limit 설정이 테스트에 적합한지 확인
4. 네트워크 타임아웃 설정 확인

### 로그 확인
```bash
# 상세 로그와 함께 테스트 실행
go test -v -args -test.v ./_test_disabled/e2e/auth/

# 특정 패턴의 로그 필터링
go test -v ./_test_disabled/e2e/auth/ 2>&1 | grep "AUTH"
```

### 테스트 격리
각 테스트는 독립적으로 실행되며, 다음과 같은 격리 메커니즘을 사용합니다:
- 테스트별 고유한 사용자 ID
- 메모리 기반 블랙리스트 (테스트 간 공유되지 않음)
- HTTP 테스트 서버 (실제 포트 사용하지 않음)

## 성능 최적화

### 테스트 실행 시간 최적화
- 병렬 테스트 실행 활성화
- 타임아웃 설정 최적화 (개발: 짧게, CI: 여유있게)
- Mock 응답 시간 최소화

### 리소스 사용 최적화
- 테스트 완료 후 리소스 정리
- 메모리 기반 저장소 사용 (실제 DB 대신)
- 네트워크 호출 최소화 (Mock 활용)

## 문제 해결

### 일반적인 문제
1. **토큰 만료 테스트 실패**
   - 시스템 시계 확인
   - 타임아웃 설정 조정

2. **Rate Limit 테스트 불안정**
   - 테스트 간 충분한 간격 확보
   - Rate Limit 카운터 초기화 확인

3. **병렬 테스트 간섭**
   - 테스트 격리 메커니즘 확인
   - 공유 리소스 사용 최소화

### 도움말
- 테스트 관련 이슈: GitHub Issues 참조
- 문서 업데이트: `/docs/testing/` 디렉토리 확인