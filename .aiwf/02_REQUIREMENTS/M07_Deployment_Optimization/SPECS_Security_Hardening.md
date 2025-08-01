---
title: Security Hardening Technical Specifications
document_type: SPECS
milestone: M07
status: draft
last_updated: 2025-08-01 07:15
---

# Technical Specifications: Security Hardening

## Overview

AICode Manager의 보안을 프로덕션 수준으로 강화합니다. OWASP Top 10 대응, 보안 스캐닝 자동화, 침투 테스트, 컴플라이언스 준수를 포함한 포괄적인 보안 강화 작업을 수행합니다.

## Security Architecture

### 보안 계층 아키텍처

```
┌─────────────────────────────────────────────────────────────┐
│                     Edge Security                            │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │
│  │    WAF      │  │   DDoS      │  │     Rate            │ │
│  │ (ModSec)    │  │ Protection  │  │    Limiting         │ │
│  └──────┬──────┘  └──────┬──────┘  └──────────┬──────────┘ │
└─────────┴─────────────────┴───────────────────┴────────────┘
          │                 │                     │
          ▼                 ▼                     ▼
┌─────────────────────────────────────────────────────────────┐
│                  Network Security                            │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │
│  │    TLS      │  │   mTLS      │  │    Network          │ │
│  │   1.3+      │  │   Auth      │  │   Policies          │ │
│  └──────┬──────┘  └──────┬──────┘  └──────────┬──────────┘ │
└─────────┴─────────────────┴───────────────────┴────────────┘
          │                 │                     │
          ▼                 ▼                     ▼
┌─────────────────────────────────────────────────────────────┐
│                Application Security                          │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │
│  │   Input     │  │   Auth &    │  │    Session          │ │
│  │ Validation  │  │    Authz    │  │   Security          │ │
│  └──────┬──────┘  └──────┬──────┘  └──────────┬──────────┘ │
└─────────┴─────────────────┴───────────────────┴────────────┘
          │                 │                     │
          ▼                 ▼                     ▼
┌─────────────────────────────────────────────────────────────┐
│                    Data Security                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │
│  │ Encryption  │  │   Key       │  │     Data            │ │
│  │  at Rest    │  │ Management  │  │   Masking           │ │
│  └──────┬──────┘  └──────┬──────┘  └──────────┬──────────┘ │
└─────────┴─────────────────┴───────────────────┴────────────┘
```

## Detailed Specifications

### 1. Application Security

#### 1.1 Input Validation and Sanitization

```go
// security/validation.go
package security

import (
    "regexp"
    "strings"
    "net/url"
    "github.com/microcosm-cc/bluemonday"
)

type Validator struct {
    policy *bluemonday.Policy
    rules  map[string]ValidationRule
}

type ValidationRule struct {
    Pattern    *regexp.Regexp
    MaxLength  int
    Required   bool
    Sanitizer  func(string) string
    Validator  func(string) error
}

func NewValidator() *Validator {
    // XSS 방지 정책
    policy := bluemonday.StrictPolicy()
    
    return &Validator{
        policy: policy,
        rules: map[string]ValidationRule{
            "username": {
                Pattern:   regexp.MustCompile(`^[a-zA-Z0-9_-]{3,32}$`),
                MaxLength: 32,
                Required:  true,
                Sanitizer: sanitizeUsername,
            },
            "email": {
                Pattern:   regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`),
                MaxLength: 254,
                Required:  true,
                Sanitizer: strings.ToLower,
            },
            "workspace_name": {
                Pattern:   regexp.MustCompile(`^[a-zA-Z0-9\s_-]{1,100}$`),
                MaxLength: 100,
                Required:  true,
                Sanitizer: sanitizeWorkspaceName,
            },
            "file_path": {
                MaxLength: 4096,
                Required:  true,
                Validator: validateFilePath,
                Sanitizer: sanitizeFilePath,
            },
            "url": {
                MaxLength: 2048,
                Required:  false,
                Validator: validateURL,
                Sanitizer: sanitizeURL,
            },
        },
    }
}

// SQL Injection 방지
func (v *Validator) PreventSQLInjection(input string) string {
    // 위험한 SQL 키워드 제거
    dangerous := []string{
        "SELECT", "INSERT", "UPDATE", "DELETE", "DROP",
        "UNION", "JOIN", "WHERE", "HAVING", "ORDER BY",
        "--", "/*", "*/", ";", "'", "\"",
    }
    
    sanitized := input
    for _, keyword := range dangerous {
        sanitized = strings.ReplaceAll(sanitized, keyword, "")
        sanitized = strings.ReplaceAll(sanitized, strings.ToLower(keyword), "")
    }
    
    return sanitized
}

// Path Traversal 방지
func validateFilePath(path string) error {
    // 상대 경로 요소 확인
    if strings.Contains(path, "..") {
        return ErrPathTraversal
    }
    
    // 절대 경로 확인
    if strings.HasPrefix(path, "/") {
        return ErrAbsolutePath
    }
    
    // 특수 문자 확인
    if strings.ContainsAny(path, "\x00\n\r") {
        return ErrInvalidCharacters
    }
    
    return nil
}

// Command Injection 방지
func (v *Validator) SanitizeCommand(cmd string) string {
    // 위험한 문자 제거
    dangerous := []string{";", "|", "&", "$", "`", "(", ")", "<", ">", "\n", "\r"}
    sanitized := cmd
    
    for _, char := range dangerous {
        sanitized = strings.ReplaceAll(sanitized, char, "")
    }
    
    return sanitized
}
```

#### 1.2 CSRF Protection

```go
// security/csrf.go
package security

import (
    "crypto/rand"
    "encoding/base64"
    "net/http"
    "time"
)

type CSRFProtection struct {
    tokenStore TokenStore
    config     CSRFConfig
}

type CSRFConfig struct {
    TokenLength    int
    TokenLifetime  time.Duration
    CookieName     string
    HeaderName     string
    SameSite       http.SameSite
    Secure         bool
}

func (c *CSRFProtection) GenerateToken(w http.ResponseWriter, r *http.Request) (string, error) {
    // 토큰 생성
    b := make([]byte, c.config.TokenLength)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    token := base64.URLEncoding.EncodeToString(b)
    
    // 세션에 저장
    sessionID := getSessionID(r)
    if err := c.tokenStore.Set(sessionID, token, c.config.TokenLifetime); err != nil {
        return "", err
    }
    
    // 쿠키 설정
    http.SetCookie(w, &http.Cookie{
        Name:     c.config.CookieName,
        Value:    token,
        Path:     "/",
        Expires:  time.Now().Add(c.config.TokenLifetime),
        HttpOnly: true,
        Secure:   c.config.Secure,
        SameSite: c.config.SameSite,
    })
    
    return token, nil
}

func (c *CSRFProtection) ValidateToken(r *http.Request) error {
    // 헤더에서 토큰 가져오기
    headerToken := r.Header.Get(c.config.HeaderName)
    if headerToken == "" {
        return ErrMissingCSRFToken
    }
    
    // 세션에서 토큰 가져오기
    sessionID := getSessionID(r)
    storedToken, err := c.tokenStore.Get(sessionID)
    if err != nil {
        return ErrInvalidCSRFToken
    }
    
    // 토큰 비교 (타이밍 공격 방지)
    if !secureCompare(headerToken, storedToken) {
        return ErrInvalidCSRFToken
    }
    
    return nil
}
```

#### 1.3 Security Headers

```go
// security/headers.go
package security

import "net/http"

type SecurityHeaders struct {
    config HeaderConfig
}

type HeaderConfig struct {
    HSTS                  HSTSConfig
    CSP                   string
    XFrameOptions         string
    XContentTypeOptions   string
    ReferrerPolicy        string
    PermissionsPolicy     string
}

type HSTSConfig struct {
    MaxAge            int
    IncludeSubdomains bool
    Preload           bool
}

func (sh *SecurityHeaders) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // HSTS
        hstsValue := fmt.Sprintf("max-age=%d", sh.config.HSTS.MaxAge)
        if sh.config.HSTS.IncludeSubdomains {
            hstsValue += "; includeSubDomains"
        }
        if sh.config.HSTS.Preload {
            hstsValue += "; preload"
        }
        w.Header().Set("Strict-Transport-Security", hstsValue)
        
        // Content Security Policy
        w.Header().Set("Content-Security-Policy", sh.config.CSP)
        
        // 기타 보안 헤더
        w.Header().Set("X-Frame-Options", sh.config.XFrameOptions)
        w.Header().Set("X-Content-Type-Options", sh.config.XContentTypeOptions)
        w.Header().Set("Referrer-Policy", sh.config.ReferrerPolicy)
        w.Header().Set("Permissions-Policy", sh.config.PermissionsPolicy)
        
        // 서버 정보 숨기기
        w.Header().Del("Server")
        w.Header().Del("X-Powered-By")
        
        next.ServeHTTP(w, r)
    })
}

// 기본 보안 헤더 설정
func DefaultSecurityHeaders() HeaderConfig {
    return HeaderConfig{
        HSTS: HSTSConfig{
            MaxAge:            31536000, // 1년
            IncludeSubdomains: true,
            Preload:           true,
        },
        CSP: "default-src 'self'; " +
            "script-src 'self' 'unsafe-inline' 'unsafe-eval'; " +
            "style-src 'self' 'unsafe-inline'; " +
            "img-src 'self' data: https:; " +
            "font-src 'self'; " +
            "connect-src 'self' wss:; " +
            "frame-ancestors 'none'; " +
            "base-uri 'self'; " +
            "form-action 'self'",
        XFrameOptions:       "DENY",
        XContentTypeOptions: "nosniff",
        ReferrerPolicy:      "strict-origin-when-cross-origin",
        PermissionsPolicy:   "geolocation=(), microphone=(), camera=()",
    }
}
```

### 2. Authentication & Authorization

#### 2.1 Advanced Authentication

```go
// security/auth_advanced.go
package security

import (
    "context"
    "crypto/subtle"
    "golang.org/x/crypto/argon2"
)

type AdvancedAuthenticator struct {
    userStore    UserStore
    tokenManager *TokenManager
    mfaProvider  MFAProvider
    config       AuthConfig
}

type AuthConfig struct {
    PasswordPolicy    PasswordPolicy
    SessionPolicy     SessionPolicy
    MFAPolicy         MFAPolicy
    LockoutPolicy     LockoutPolicy
}

type PasswordPolicy struct {
    MinLength          int
    RequireUppercase   bool
    RequireLowercase   bool
    RequireNumbers     bool
    RequireSpecial     bool
    PreventCommon      bool
    HistoryCount       int
    ExpirationDays     int
}

// 비밀번호 해싱 (Argon2id)
func (a *AdvancedAuthenticator) HashPassword(password string) (string, error) {
    // 비밀번호 정책 검증
    if err := a.validatePasswordPolicy(password); err != nil {
        return "", err
    }
    
    // Salt 생성
    salt := make([]byte, 16)
    if _, err := rand.Read(salt); err != nil {
        return "", err
    }
    
    // Argon2id 해싱
    hash := argon2.IDKey(
        []byte(password),
        salt,
        1,        // iterations
        64*1024,  // memory (64MB)
        4,        // parallelism
        32,       // key length
    )
    
    // 인코딩
    return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
        argon2.Version,
        64*1024,
        1,
        4,
        base64.RawStdEncoding.EncodeToString(salt),
        base64.RawStdEncoding.EncodeToString(hash),
    ), nil
}

// MFA 구현
type MFAProvider struct {
    totpSecret string
    config     MFAConfig
}

func (m *MFAProvider) GenerateTOTP(user *User) (string, error) {
    key, err := totp.Generate(totp.GenerateOpts{
        Issuer:      "AICode Manager",
        AccountName: user.Email,
        Period:      30,
        SecretSize:  32,
        Digits:      6,
        Algorithm:   otp.AlgorithmSHA256,
    })
    if err != nil {
        return "", err
    }
    
    return key.Secret(), nil
}

func (m *MFAProvider) ValidateTOTP(secret, code string) bool {
    return totp.Validate(code, secret)
}

// 계정 잠금 관리
type LockoutManager struct {
    store  LockoutStore
    config LockoutPolicy
}

type LockoutPolicy struct {
    MaxAttempts      int
    LockoutDuration  time.Duration
    ResetAfter       time.Duration
}

func (lm *LockoutManager) RecordFailedAttempt(userID string) error {
    attempts, err := lm.store.IncrementAttempts(userID)
    if err != nil {
        return err
    }
    
    if attempts >= lm.config.MaxAttempts {
        return lm.store.LockAccount(userID, lm.config.LockoutDuration)
    }
    
    return nil
}
```

#### 2.2 Fine-Grained Authorization

```go
// security/authorization.go
package security

type PolicyEngine struct {
    policies map[string]Policy
    enforcer *casbin.Enforcer
}

type Policy struct {
    ID          string
    Name        string
    Effect      Effect
    Actions     []string
    Resources   []string
    Conditions  []Condition
}

type Condition struct {
    Type     string
    Operator string
    Value    interface{}
}

func (pe *PolicyEngine) Authorize(ctx context.Context, subject Subject, action string, resource string) (bool, error) {
    // 컨텍스트에서 속성 추출
    attrs := extractAttributes(ctx)
    
    // IP 기반 제한
    if !pe.checkIPRestrictions(attrs.IP, subject) {
        return false, ErrIPRestricted
    }
    
    // 시간 기반 제한
    if !pe.checkTimeRestrictions(time.Now(), subject) {
        return false, ErrTimeRestricted
    }
    
    // RBAC 확인
    allowed, err := pe.enforcer.Enforce(subject.ID, resource, action)
    if err != nil {
        return false, err
    }
    
    if !allowed {
        return false, ErrUnauthorized
    }
    
    // 추가 조건 확인
    for _, policy := range pe.getPoliciesForSubject(subject) {
        if pe.evaluatePolicy(policy, attrs) {
            if policy.Effect == Deny {
                return false, ErrPolicyDenied
            }
        }
    }
    
    return true, nil
}
```

### 3. Data Security

#### 3.1 Encryption at Rest

```go
// security/encryption.go
package security

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "io"
)

type EncryptionService struct {
    keyManager KeyManager
    config     EncryptionConfig
}

type EncryptionConfig struct {
    Algorithm      string
    KeyRotation    time.Duration
    KeyDerivation  KDFConfig
}

// 필드 레벨 암호화
func (es *EncryptionService) EncryptField(plaintext []byte, context string) ([]byte, error) {
    // 컨텍스트별 키 유도
    key, err := es.keyManager.DeriveKey(context)
    if err != nil {
        return nil, err
    }
    
    // AES-GCM 암호화
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }
    
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }
    
    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, err
    }
    
    ciphertext := gcm.Seal(nonce, nonce, plaintext, []byte(context))
    return ciphertext, nil
}

// 투명한 암호화 레이어
type EncryptedStorage struct {
    underlying Storage
    encryption *EncryptionService
}

func (es *EncryptedStorage) Set(key string, value interface{}) error {
    // 직렬화
    data, err := serialize(value)
    if err != nil {
        return err
    }
    
    // 암호화
    encrypted, err := es.encryption.EncryptField(data, key)
    if err != nil {
        return err
    }
    
    // 저장
    return es.underlying.Set(key, encrypted)
}

func (es *EncryptedStorage) Get(key string, dest interface{}) error {
    // 조회
    var encrypted []byte
    if err := es.underlying.Get(key, &encrypted); err != nil {
        return err
    }
    
    // 복호화
    decrypted, err := es.encryption.DecryptField(encrypted, key)
    if err != nil {
        return err
    }
    
    // 역직렬화
    return deserialize(decrypted, dest)
}
```

#### 3.2 Key Management

```go
// security/key_management.go
package security

import (
    "cloud.google.com/go/kms/apiv1"
    "github.com/hashicorp/vault/api"
)

type KeyManager struct {
    provider   KeyProvider
    cache      *KeyCache
    rotator    *KeyRotator
}

type KeyProvider interface {
    GetKey(ctx context.Context, keyID string) ([]byte, error)
    CreateKey(ctx context.Context, keyID string) ([]byte, error)
    RotateKey(ctx context.Context, keyID string) ([]byte, error)
    DeleteKey(ctx context.Context, keyID string) error
}

// HashiCorp Vault 통합
type VaultKeyProvider struct {
    client *api.Client
    mount  string
}

func (vkp *VaultKeyProvider) GetKey(ctx context.Context, keyID string) ([]byte, error) {
    secret, err := vkp.client.Logical().Read(fmt.Sprintf("%s/keys/%s", vkp.mount, keyID))
    if err != nil {
        return nil, err
    }
    
    if secret == nil || secret.Data == nil {
        return nil, ErrKeyNotFound
    }
    
    key, ok := secret.Data["key"].(string)
    if !ok {
        return nil, ErrInvalidKeyFormat
    }
    
    return base64.StdEncoding.DecodeString(key)
}

// 키 로테이션
type KeyRotator struct {
    manager  *KeyManager
    schedule RotationSchedule
}

func (kr *KeyRotator) Start(ctx context.Context) {
    ticker := time.NewTicker(kr.schedule.CheckInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            kr.checkAndRotate(ctx)
        }
    }
}

func (kr *KeyRotator) checkAndRotate(ctx context.Context) {
    keys, err := kr.manager.ListKeys(ctx)
    if err != nil {
        log.Error("Failed to list keys", "error", err)
        return
    }
    
    for _, key := range keys {
        if kr.shouldRotate(key) {
            if err := kr.rotateKey(ctx, key); err != nil {
                log.Error("Failed to rotate key", "keyID", key.ID, "error", err)
            }
        }
    }
}
```

### 4. Security Scanning

#### 4.1 Vulnerability Scanning

```go
// security/scanner.go
package security

import (
    "github.com/aquasecurity/trivy/pkg/scanner"
)

type VulnerabilityScanner struct {
    scanner  scanner.Scanner
    config   ScannerConfig
    reporter Reporter
}

type ScannerConfig struct {
    ScanTypes      []string
    Severity       []string
    IgnoreFile     string
    CacheDir       string
    UpdateDB       bool
}

func (vs *VulnerabilityScanner) ScanContainer(imageID string) (*ScanResult, error) {
    result, err := vs.scanner.ScanImage(imageID)
    if err != nil {
        return nil, err
    }
    
    // 결과 분석
    scanResult := &ScanResult{
        ImageID:   imageID,
        Timestamp: time.Now(),
        Findings:  []Finding{},
    }
    
    for _, vuln := range result.Vulnerabilities {
        if vs.shouldReport(vuln) {
            scanResult.Findings = append(scanResult.Findings, Finding{
                Type:        "vulnerability",
                Severity:    vuln.Severity,
                Title:       vuln.Title,
                Description: vuln.Description,
                CVE:         vuln.VulnerabilityID,
                Package:     vuln.PkgName,
                Version:     vuln.InstalledVersion,
                FixVersion:  vuln.FixedVersion,
            })
        }
    }
    
    return scanResult, nil
}

// SAST (Static Application Security Testing)
type SASTScanner struct {
    rules    []SecurityRule
    analyzer CodeAnalyzer
}

func (ss *SASTScanner) ScanCode(path string) (*ScanResult, error) {
    // 코드 분석
    ast, err := ss.analyzer.Parse(path)
    if err != nil {
        return nil, err
    }
    
    result := &ScanResult{
        Path:      path,
        Timestamp: time.Now(),
        Findings:  []Finding{},
    }
    
    // 규칙 적용
    for _, rule := range ss.rules {
        findings := rule.Check(ast)
        result.Findings = append(result.Findings, findings...)
    }
    
    return result, nil
}
```

#### 4.2 Dependency Scanning

```go
// security/dependency_scanner.go
package security

type DependencyScanner struct {
    checkers map[string]DependencyChecker
}

type DependencyChecker interface {
    Check(path string) ([]Vulnerability, error)
}

// Go 모듈 스캐너
type GoModScanner struct {
    database VulnDB
}

func (gms *GoModScanner) Check(modPath string) ([]Vulnerability, error) {
    // go.mod 파싱
    modFile, err := modfile.Parse(modPath, nil, nil)
    if err != nil {
        return nil, err
    }
    
    var vulnerabilities []Vulnerability
    
    // 각 의존성 확인
    for _, req := range modFile.Require {
        vulns, err := gms.database.Check(req.Mod.Path, req.Mod.Version)
        if err != nil {
            continue
        }
        vulnerabilities = append(vulnerabilities, vulns...)
    }
    
    return vulnerabilities, nil
}

// NPM 패키지 스캐너
type NPMScanner struct {
    auditAPI string
}

func (ns *NPMScanner) Check(packagePath string) ([]Vulnerability, error) {
    // package-lock.json 읽기
    lockFile, err := os.ReadFile(filepath.Join(packagePath, "package-lock.json"))
    if err != nil {
        return nil, err
    }
    
    // npm audit API 호출
    resp, err := ns.callAuditAPI(lockFile)
    if err != nil {
        return nil, err
    }
    
    return ns.parseAuditResponse(resp)
}
```

### 5. Security Monitoring

#### 5.1 Security Event Monitoring

```go
// security/monitoring.go
package security

type SecurityMonitor struct {
    detectors  []ThreatDetector
    aggregator EventAggregator
    responder  IncidentResponder
}

type ThreatDetector interface {
    Detect(event Event) *ThreatIndicator
}

// 이상 행동 감지
type AnomalyDetector struct {
    baseline BaselineProfile
    ml       MLModel
}

func (ad *AnomalyDetector) Detect(event Event) *ThreatIndicator {
    // 특성 추출
    features := ad.extractFeatures(event)
    
    // ML 모델 예측
    score := ad.ml.Predict(features)
    
    if score > ad.threshold {
        return &ThreatIndicator{
            Type:        "anomaly",
            Severity:    ad.calculateSeverity(score),
            Description: ad.describeAnomaly(event, features),
            Score:       score,
            Timestamp:   time.Now(),
        }
    }
    
    return nil
}

// 무차별 대입 공격 감지
type BruteForceDetector struct {
    window   time.Duration
    counters map[string]*RateLimiter
}

func (bfd *BruteForceDetector) Detect(event Event) *ThreatIndicator {
    if event.Type != "auth_failed" {
        return nil
    }
    
    key := fmt.Sprintf("%s:%s", event.IP, event.UserID)
    counter := bfd.getOrCreateCounter(key)
    
    if counter.Exceeded() {
        return &ThreatIndicator{
            Type:        "brute_force",
            Severity:    "high",
            Description: fmt.Sprintf("Brute force attempt detected from %s", event.IP),
            Metadata: map[string]interface{}{
                "attempts":  counter.Count(),
                "timeframe": bfd.window,
            },
        }
    }
    
    return nil
}
```

### 6. Incident Response

#### 6.1 Automated Response

```go
// security/incident_response.go
package security

type IncidentResponder struct {
    playbooks map[string]Playbook
    executor  ActionExecutor
}

type Playbook struct {
    Name        string
    Triggers    []TriggerCondition
    Actions     []ResponseAction
    Escalation  EscalationPolicy
}

type ResponseAction struct {
    Type     string
    Target   string
    Params   map[string]interface{}
    Timeout  time.Duration
    Rollback func() error
}

func (ir *IncidentResponder) Respond(incident Incident) error {
    // 적절한 플레이북 선택
    playbook := ir.selectPlaybook(incident)
    if playbook == nil {
        return ir.defaultResponse(incident)
    }
    
    // 액션 실행
    for _, action := range playbook.Actions {
        if err := ir.executor.Execute(action); err != nil {
            // 롤백
            ir.rollbackActions(playbook.Actions[:i])
            return err
        }
    }
    
    // 보고서 생성
    report := ir.generateReport(incident, playbook)
    return ir.notifyStakeholders(report)
}

// 자동 차단
type AutoBlocker struct {
    firewall Firewall
    duration time.Duration
}

func (ab *AutoBlocker) BlockIP(ip string, reason string) error {
    rule := FirewallRule{
        Type:        "DROP",
        Source:      ip,
        Description: reason,
        Expiry:      time.Now().Add(ab.duration),
    }
    
    return ab.firewall.AddRule(rule)
}
```

### 7. Compliance

#### 7.1 Audit Logging

```go
// security/audit.go
package security

type AuditLogger struct {
    storage    AuditStorage
    encryption EncryptionService
    config     AuditConfig
}

type AuditConfig struct {
    RetentionDays   int
    ComplianceLevel string // PCI-DSS, HIPAA, SOC2
    TamperProof     bool
}

type AuditEntry struct {
    ID          string
    Timestamp   time.Time
    Actor       Actor
    Action      string
    Resource    string
    Result      string
    Details     map[string]interface{}
    IPAddress   string
    UserAgent   string
    SessionID   string
    TraceID     string
    Signature   string // 무결성 검증용
}

func (al *AuditLogger) Log(entry AuditEntry) error {
    // 시그니처 생성 (무결성 보장)
    entry.Signature = al.generateSignature(entry)
    
    // 민감 데이터 마스킹
    entry.Details = al.maskSensitiveData(entry.Details)
    
    // 암호화
    encrypted, err := al.encryption.Encrypt(entry)
    if err != nil {
        return err
    }
    
    // 저장
    return al.storage.Store(encrypted)
}

// 컴플라이언스 리포트
type ComplianceReporter struct {
    auditor    AuditLogger
    scanner    SecurityScanner
    policies   []CompliancePolicy
}

func (cr *ComplianceReporter) GenerateReport(standard string) (*ComplianceReport, error) {
    report := &ComplianceReport{
        Standard:  standard,
        Timestamp: time.Now(),
        Results:   []ComplianceCheck{},
    }
    
    // 각 정책 확인
    for _, policy := range cr.getPoliciesForStandard(standard) {
        result := cr.checkPolicy(policy)
        report.Results = append(report.Results, result)
    }
    
    report.Score = cr.calculateScore(report.Results)
    report.Recommendations = cr.generateRecommendations(report.Results)
    
    return report, nil
}
```

### 8. Security Configuration

```yaml
# security.yaml
security:
  # 암호화 설정
  encryption:
    algorithm: AES-256-GCM
    key_rotation_days: 90
    key_provider: vault
    vault:
      address: https://vault.aicli.com
      mount: transit
      
  # 인증 설정
  authentication:
    password_policy:
      min_length: 12
      require_uppercase: true
      require_lowercase: true
      require_numbers: true
      require_special: true
      prevent_common: true
      history_count: 5
      expiration_days: 90
    mfa:
      required: true
      methods:
        - totp
        - webauthn
    session:
      timeout: 30m
      max_concurrent: 3
      
  # 권한 설정
  authorization:
    model: rbac
    policy_file: /etc/aicli/policies.yaml
    
  # 보안 헤더
  headers:
    hsts:
      max_age: 31536000
      include_subdomains: true
      preload: true
    csp: "default-src 'self'; script-src 'self' 'unsafe-inline';"
    x_frame_options: DENY
    x_content_type_options: nosniff
    
  # 스캐닝 설정
  scanning:
    vulnerability:
      enabled: true
      schedule: "0 2 * * *"
      severity: [CRITICAL, HIGH, MEDIUM]
    dependency:
      enabled: true
      check_on_build: true
    sast:
      enabled: true
      rules_path: /etc/aicli/sast-rules
      
  # 모니터링 설정
  monitoring:
    anomaly_detection:
      enabled: true
      threshold: 0.95
    brute_force:
      window: 10m
      max_attempts: 5
    rate_limiting:
      enabled: true
      default_limit: 100
      burst: 20
      
  # 감사 설정
  audit:
    enabled: true
    retention_days: 365
    compliance_level: SOC2
    tamper_proof: true
```

## Security Checklist

### Pre-Deployment
- [ ] 모든 의존성 스캔 완료
- [ ] SAST 스캔 통과
- [ ] 보안 헤더 설정 확인
- [ ] TLS 설정 검증
- [ ] 비밀번호 정책 활성화

### Deployment
- [ ] 최소 권한 원칙 적용
- [ ] 네트워크 정책 구성
- [ ] WAF 규칙 활성화
- [ ] 모니터링 알림 설정
- [ ] 백업 암호화 확인

### Post-Deployment
- [ ] 침투 테스트 수행
- [ ] 보안 감사 실행
- [ ] 컴플라이언스 검증
- [ ] 인시던트 대응 훈련
- [ ] 보안 문서 업데이트

## Security Testing

### 침투 테스트 계획
1. **정찰 단계**: 정보 수집
2. **스캐닝 단계**: 취약점 식별
3. **공격 단계**: 취약점 검증
4. **권한 상승**: 시스템 침투
5. **보고서 작성**: 발견사항 문서화

### 보안 벤치마크
- OWASP Top 10 준수
- CIS 벤치마크 적용
- NIST 프레임워크 준수

## Timeline

### Week 1: 기본 보안
- 입력 검증 구현
- 보안 헤더 설정
- 기본 스캐닝 도구 통합

### Week 2: 고급 보안
- 암호화 시스템 구현
- 키 관리 시스템 구축
- 고급 인증 구현

### Week 3: 모니터링 및 컴플라이언스
- 보안 모니터링 시스템
- 감사 로깅 구현
- 컴플라이언스 검증