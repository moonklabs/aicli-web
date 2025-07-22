# 보안 가이드

## 개요

AICode Manager는 다층 보안 아키텍처를 통해 포괄적인 보안을 제공합니다. 이 문서는 구현된 보안 시스템의 사용법과 설정 방법을 설명합니다.

## 보안 아키텍처

### 1. 인증 및 권한 관리

#### OAuth2 인증
- **지원 공급자**: Google, GitHub
- **보안 기능**: PKCE (Proof Key for Code Exchange)
- **설정 위치**: `internal/auth/oauth2.go`

```go
// OAuth2 설정 예시
config := &OAuth2Config{
    Google: &OAuth2Provider{
        ClientID:     "your-google-client-id",
        ClientSecret: "your-google-client-secret",
        RedirectURL:  "https://your-domain.com/auth/callback/google",
    },
    GitHub: &OAuth2Provider{
        ClientID:     "your-github-client-id",
        ClientSecret: "your-github-client-secret",
        RedirectURL:  "https://your-domain.com/auth/callback/github",
    },
}
```

#### RBAC (역할 기반 접근 제어)
- **계층적 역할 시스템**: 상위 역할이 하위 역할의 권한을 상속
- **세분화된 권한**: 리소스별, 작업별 권한 관리
- **캐싱**: Redis를 통한 고성능 권한 조회

```go
// 권한 확인 예시
err := rbacService.CheckPermission(userID, "projects", "create")
if err != nil {
    // 권한 없음
    return errors.NewForbiddenError("insufficient permissions")
}
```

### 2. 세션 관리

#### Redis 기반 분산 세션
- **확장성**: 다중 서버 환경에서 세션 공유
- **보안성**: 디바이스 핑거프린팅 및 위치 기반 검증
- **관리**: 동시 로그인 제한 및 세션 모니터링

```go
// 세션 설정 예시
sessionConfig := &SessionConfig{
    MaxConcurrentSessions: 3,
    SessionTimeout:        30 * time.Minute,
    DeviceTrackingEnabled: true,
    GeoLocationEnabled:    true,
}
```

### 3. 고급 Rate Limiting

#### 다층 제한
- **전역 제한**: 시스템 전체 요청 제한
- **IP별 제한**: 특정 IP의 요청 제한
- **사용자별 제한**: 인증된 사용자의 요청 제한
- **엔드포인트별 제한**: 민감한 API 엔드포인트 보호

```go
// Rate Limiting 설정 예시
config := &AdvancedRateLimitConfig{
    GlobalRateLimit: 1000,  // 초당 1000 요청
    IPRateLimit:     100,   // IP당 초당 100 요청
    UserRateLimit:   500,   // 사용자당 초당 500 요청
    EndpointRateLimit: map[string]int{
        "/api/v1/auth/login": 5,  // 로그인은 분당 5회
    },
}
```

### 4. CSRF 보호

#### 토큰 기반 보호
- **동적 토큰**: 요청마다 새로운 토큰 생성
- **다중 전송**: 헤더, 쿠키, 폼 필드 지원
- **Origin 검증**: 요청 출처 확인

```go
// CSRF 설정 예시
csrfConfig := &CSRFConfig{
    TokenLifetime: 24 * time.Hour,
    SecureCookie:  true,
    SameSite:      http.SameSiteStrictMode,
    TrustedOrigins: []string{
        "https://yourdomain.com",
        "https://www.yourdomain.com",
    },
}
```

### 5. 보안 헤더

#### 포괄적인 보안 헤더
- **HSTS**: HTTPS 강제 사용
- **CSP**: 콘텐츠 보안 정책
- **Frame Options**: 클릭재킹 방지
- **Content Type Options**: MIME 타입 스니핑 방지

```go
// 보안 헤더 설정 예시
headers := &SecurityHeadersConfig{
    HSTSMaxAge:            31536000, // 1년
    HSTSIncludeSubdomains: true,
    CSPDefaultSrc:         []string{"'self'"},
    CSPScriptSrc:          []string{"'self'", "'unsafe-inline'"},
    FrameOptions:          "SAMEORIGIN",
    ContentTypeNosniff:    true,
}
```

### 6. 공격 탐지

#### 다양한 공격 패턴 탐지
- **SQL Injection**: 데이터베이스 공격 탐지
- **XSS**: 크로스사이트 스크립팅 공격 탐지
- **Command Injection**: 시스템 명령 실행 공격 탐지
- **브루트포스**: 무차별 대입 공격 탐지

#### 자동 대응
- **IP 차단**: 공격 IP 자동 차단
- **알림 발송**: 관리자에게 즉시 알림
- **패턴 학습**: 새로운 공격 패턴 학습 및 추가

### 7. 감사 로깅

#### 포괄적인 로그 수집
- **요청/응답**: 모든 HTTP 트래픽 로깅
- **인증 이벤트**: 로그인, 로그아웃, 권한 변경
- **보안 이벤트**: 공격 시도, 위반 사항
- **시스템 이벤트**: 설정 변경, 시스템 오류

```go
// 감사 로깅 설정 예시
auditConfig := &AuditConfig{
    EnableRequestLogging:  true,
    EnableResponseLogging: true,
    EnableBodyLogging:     true,
    MaxBodySize:          1024 * 1024, // 1MB
    SensitiveHeaders: []string{
        "Authorization",
        "Cookie",
        "X-CSRF-Token",
    },
    SensitiveFields: []string{
        "password",
        "token",
        "secret",
    },
}
```

## 설정 가이드

### 1. 기본 보안 설정

#### 환경 변수
```bash
# OAuth2 설정
GOOGLE_CLIENT_ID=your-google-client-id
GOOGLE_CLIENT_SECRET=your-google-client-secret
GITHUB_CLIENT_ID=your-github-client-id
GITHUB_CLIENT_SECRET=your-github-client-secret

# Redis 설정
REDIS_URL=redis://localhost:6379
REDIS_PASSWORD=your-redis-password

# 보안 설정
CSRF_SECRET_KEY=your-32-character-secret-key
SESSION_SECRET_KEY=your-session-secret-key
ENCRYPTION_KEY=your-encryption-key
```

#### 설정 파일 (config.yml)
```yaml
security:
  oauth2:
    google:
      client_id: ${GOOGLE_CLIENT_ID}
      client_secret: ${GOOGLE_CLIENT_SECRET}
      redirect_url: "https://yourdomain.com/auth/callback/google"
    github:
      client_id: ${GITHUB_CLIENT_ID}
      client_secret: ${GITHUB_CLIENT_SECRET}
      redirect_url: "https://yourdomain.com/auth/callback/github"
  
  session:
    timeout: "30m"
    max_concurrent: 3
    secure_cookie: true
    same_site: "strict"
  
  rate_limit:
    global: 1000
    ip: 100
    user: 500
    endpoints:
      "/api/v1/auth/login": 5
      "/api/v1/auth/register": 3
  
  csrf:
    token_lifetime: "24h"
    secure_cookie: true
    trusted_origins:
      - "https://yourdomain.com"
      - "https://www.yourdomain.com"
```

### 2. 미들웨어 설정

#### Gin 프레임워크 통합
```go
func setupSecurityMiddlewares(r *gin.Engine, deps *Dependencies) {
    // 보안 헤더
    r.Use(middleware.SecurityHeadersMiddleware(deps.SecurityHeadersConfig))
    
    // CSRF 보호
    r.Use(middleware.CSRF(deps.CSRFConfig))
    
    // Rate Limiting
    r.Use(middleware.AdvancedRateLimit(deps.RateLimitConfig))
    
    // 감사 로깅
    r.Use(middleware.AuditMiddleware(deps.AuditConfig))
    
    // 인증 미들웨어
    r.Use(middleware.JWTAuth(deps.JWTConfig))
    
    // RBAC 미들웨어
    r.Use(middleware.RBACMiddleware(deps.RBACConfig))
}
```

### 3. API 라우팅 보안

#### 보호된 라우트 설정
```go
func setupSecureRoutes(r *gin.Engine, deps *Dependencies) {
    // 공개 라우트
    public := r.Group("/api/v1/public")
    {
        public.GET("/health", healthHandler)
        public.GET("/status", statusHandler)
    }
    
    // 인증 필요 라우트
    auth := r.Group("/api/v1/auth")
    auth.Use(middleware.JWTAuth(deps.JWTConfig))
    {
        auth.GET("/profile", profileHandler)
        auth.POST("/logout", logoutHandler)
    }
    
    // 관리자 전용 라우트
    admin := r.Group("/api/v1/admin")
    admin.Use(middleware.JWTAuth(deps.JWTConfig))
    admin.Use(middleware.RequireRole("admin"))
    {
        admin.GET("/users", listUsersHandler)
        admin.POST("/users", createUserHandler)
        admin.DELETE("/users/:id", deleteUserHandler)
    }
    
    // 보안 관리 라우트
    security := r.Group("/api/v1/security")
    security.Use(middleware.JWTAuth(deps.JWTConfig))
    security.Use(middleware.RequirePermission("security", "read"))
    {
        security.GET("/events", deps.SecurityController.GetSecurityEvents)
        security.GET("/statistics", deps.SecurityController.GetSecurityStatistics)
        security.POST("/detect-attack", deps.SecurityController.DetectAttack)
    }
}
```

## 모니터링 및 대응

### 1. 보안 이벤트 모니터링

#### 대시보드 메트릭
- **실시간 공격 시도 수**: 초당/분당 공격 시도 건수
- **차단된 IP 수**: 현재 차단된 IP 주소 목록
- **인증 실패율**: 성공/실패 비율
- **Rate Limit 위반**: 제한 초과 요청 수

#### 알림 설정
```go
// 알림 임계값 설정
alertConfig := &AlertConfig{
    AttackThreshold:     100, // 시간당 100회 이상 공격 시도
    AuthFailureRate:     0.3, // 30% 이상 인증 실패율
    RateLimitViolation: 1000, // 시간당 1000회 이상 제한 위반
    BlockedIPThreshold:   50, // 50개 이상 IP 차단 시
}
```

### 2. 로그 분석

#### 로그 형식
```json
{
  "timestamp": "2023-07-22T10:30:00Z",
  "level": "WARN",
  "message": "Security attack detected",
  "fields": {
    "attack_type": "sql_injection",
    "ip_address": "192.168.1.100",
    "user_agent": "sqlmap/1.0",
    "confidence": 0.95,
    "blocked": true,
    "patterns": ["UNION SELECT", "DROP TABLE"]
  }
}
```

#### 로그 분석 쿼리 예시
```bash
# 최근 1시간 공격 시도
grep "Security attack detected" /var/log/aicli/security.log | \
  jq 'select(.timestamp > (now - 3600)) | .fields.attack_type' | \
  sort | uniq -c

# IP별 공격 시도 수
grep "Security attack detected" /var/log/aicli/security.log | \
  jq -r '.fields.ip_address' | sort | uniq -c | sort -nr
```

### 3. 자동 대응

#### 자동 차단 규칙
- **브루트포스**: 5분 내 10회 이상 로그인 실패 시 30분 차단
- **SQL Injection**: 탐지 즉시 1시간 차단
- **Rate Limit**: 제한 초과 시 15분 차단
- **의심스러운 활동**: 점수 기반 자동 차단

#### 수동 대응 절차
1. **알림 수신**: 보안 이벤트 발생 시 즉시 알림
2. **로그 분석**: 공격 패턴 및 영향 범위 분석
3. **차단 조치**: 필요 시 수동 IP 차단
4. **패치 적용**: 취약점 발견 시 긴급 패치
5. **사후 분석**: 공격 원인 분석 및 대응 개선

## API 참조

### 보안 이벤트 API

#### 보안 이벤트 목록 조회
```http
GET /api/v1/security/events
Authorization: Bearer {token}

Query Parameters:
- types: 이벤트 타입 (쉼표로 구분)
- severities: 심각도 (low,medium,high,critical)
- start_time: 시작 시간 (RFC3339)
- end_time: 종료 시간 (RFC3339)
- limit: 제한 수 (기본: 100)
- offset: 오프셋 (기본: 0)
```

#### 공격 탐지 API
```http
POST /api/v1/security/detect-attack
Authorization: Bearer {token}
Content-Type: application/json

{
  "ip_address": "192.168.1.100",
  "user_agent": "Mozilla/5.0...",
  "method": "GET",
  "url": "https://example.com/api/users?id=1",
  "path": "/api/users",
  "query": "id=1",
  "headers": {
    "Accept": "application/json"
  }
}
```

#### 보안 통계 API
```http
GET /api/v1/security/statistics?period=24h
Authorization: Bearer {token}

Response:
{
  "period": "24h",
  "events": {
    "total_events": 1250,
    "events_by_type": {
      "sql_injection": 45,
      "xss": 23,
      "brute_force": 12
    }
  },
  "attacks": {
    "blocked_ips": 15,
    "active_patterns": 25
  }
}
```

## 보안 모범 사례

### 1. 배포 환경 보안

#### HTTPS 설정
- **TLS 1.2 이상**: 구형 프로토콜 비활성화
- **강력한 암호화**: AES-256, ECDHE 등 사용
- **HSTS 적용**: Strict-Transport-Security 헤더 설정

#### 방화벽 설정
```bash
# 기본 정책: 모든 연결 차단
iptables -P INPUT DROP
iptables -P FORWARD DROP
iptables -P OUTPUT ACCEPT

# 필요한 포트만 허용
iptables -A INPUT -p tcp --dport 80 -j ACCEPT   # HTTP
iptables -A INPUT -p tcp --dport 443 -j ACCEPT  # HTTPS
iptables -A INPUT -p tcp --dport 22 -j ACCEPT   # SSH (관리용)

# 로컬 연결 허용
iptables -A INPUT -i lo -j ACCEPT
iptables -A OUTPUT -o lo -j ACCEPT
```

### 2. 애플리케이션 보안

#### 입력 검증
```go
// 입력 데이터 검증 예시
func validateUserInput(input string) error {
    // 길이 제한
    if len(input) > 1000 {
        return errors.New("입력 데이터가 너무 큽니다")
    }
    
    // SQL Injection 패턴 검사
    sqlPatterns := []string{
        `(?i)(union\s+(all\s+)?select)`,
        `(?i)(\'\s*or\s*\d+\s*=\s*\d+)`,
        `(?i)(drop\s+table)`,
    }
    
    for _, pattern := range sqlPatterns {
        if matched, _ := regexp.MatchString(pattern, input); matched {
            return errors.New("악성 입력이 감지되었습니다")
        }
    }
    
    return nil
}
```

#### 안전한 쿠키 설정
```go
cookie := &http.Cookie{
    Name:     "session",
    Value:    sessionToken,
    Path:     "/",
    Domain:   "yourdomain.com",
    MaxAge:   3600,
    Secure:   true,           // HTTPS에서만 전송
    HttpOnly: true,           // JavaScript 접근 차단
    SameSite: http.SameSiteStrictMode, // CSRF 보호
}
```

### 3. 데이터베이스 보안

#### 접근 제어
- **최소 권한 원칙**: 필요한 권한만 부여
- **전용 계정**: 애플리케이션 전용 DB 계정 사용
- **네트워크 격리**: DB 서버 네트워크 분리

#### 데이터 암호화
```go
// 민감한 데이터 암호화
func encryptSensitiveData(data string, key []byte) (string, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return "", err
    }
    
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }
    
    nonce := make([]byte, gcm.NonceSize())
    if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
        return "", err
    }
    
    ciphertext := gcm.Seal(nonce, nonce, []byte(data), nil)
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}
```

## 문제 해결

### 1. 일반적인 문제

#### 높은 Rate Limit 위반
**증상**: 429 Too Many Requests 오류 빈발
**원인**: Rate Limit 설정이 너무 낮거나 봇 트래픽
**해결책**:
1. Rate Limit 임계값 조정
2. 화이트리스트에 신뢰할 수 있는 IP 추가
3. 봇 트래픽 분석 및 차단

#### CSRF 토큰 오류
**증상**: 403 Forbidden, "Invalid CSRF token"
**원인**: 토큰 만료, Origin 불일치, 쿠키 설정 문제
**해결책**:
1. 토큰 만료 시간 조정
2. TrustedOrigins에 도메인 추가
3. 쿠키 SameSite 설정 확인

#### 세션 이상 감지
**증상**: 정상 사용자가 로그아웃됨
**원인**: 디바이스 변경 감지, IP 위치 변경
**해결책**:
1. 감지 임계값 조정
2. 모바일 사용자 예외 처리
3. VPN 사용자 고려

### 2. 성능 문제

#### Redis 연결 오류
**증상**: "connection refused", "timeout" 오류
**원인**: Redis 서버 다운, 네트워크 문제, 연결 풀 부족
**해결책**:
1. Redis 서버 상태 확인
2. 연결 풀 크기 조정
3. 연결 타임아웃 설정 조정

#### 메모리 사용량 증가
**증상**: 메모리 사용량 지속적 증가
**원인**: 메트릭 데이터 누적, 세션 정리 실패
**해결책**:
1. 메트릭 보관 기간 단축
2. 정리 작업 주기 단축
3. 메모리 프로파일링 수행

## 결론

이 보안 시스템은 현대적인 웹 애플리케이션에 필요한 다층 보안을 제공합니다. 정기적인 모니터링과 업데이트를 통해 새로운 위협에 대응하고, 보안 정책을 지속적으로 개선하는 것이 중요합니다.

보안 관련 문의사항이나 문제가 발생할 경우, 보안 팀에 즉시 연락하여 신속한 대응을 받으시기 바랍니다.