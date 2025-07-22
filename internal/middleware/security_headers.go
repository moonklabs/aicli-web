package middleware

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SecurityHeadersConfig는 보안 헤더 설정입니다.
type SecurityHeadersConfig struct {
	// HSTS (HTTP Strict Transport Security) 설정
	HSTSMaxAge            int    // HSTS Max-Age (초)
	HSTSIncludeSubdomains bool   // 서브도메인 포함 여부
	HSTSPreload           bool   // Preload 목록 등록 여부
	
	// Content Security Policy 설정
	CSPDefaultSrc    []string // default-src 지시문
	CSPScriptSrc     []string // script-src 지시문
	CSPStyleSrc      []string // style-src 지시문
	CSPImgSrc        []string // img-src 지시문
	CSPConnectSrc    []string // connect-src 지시문
	CSPFontSrc       []string // font-src 지시문
	CSPObjectSrc     []string // object-src 지시문
	CSPMediaSrc      []string // media-src 지시문
	CSPFrameSrc      []string // frame-src 지시문
	CSPSandbox       []string // sandbox 지시문
	CSPReportURI     string   // report-uri 지시문
	CSPReportOnly    bool     // Report-Only 모드 활성화
	
	// X-Frame-Options 설정
	FrameOptions string // DENY, SAMEORIGIN, ALLOW-FROM uri
	
	// X-Content-Type-Options 설정
	ContentTypeNosniff bool // nosniff 활성화
	
	// X-XSS-Protection 설정
	XSSProtection string // 0, 1, 1; mode=block
	
	// Referrer-Policy 설정
	ReferrerPolicy string // no-referrer, strict-origin-when-cross-origin 등
	
	// Permissions-Policy 설정
	PermissionsPolicy map[string][]string // 기능별 허용 도메인 목록
	
	// Cross-Origin 설정
	CrossOriginEmbedderPolicy string // require-corp, unsafe-none
	CrossOriginOpenerPolicy   string // same-origin, same-origin-allow-popups, unsafe-none
	CrossOriginResourcePolicy string // same-site, same-origin, cross-origin
	
	// Custom 헤더
	CustomHeaders map[string]string // 추가 보안 헤더
	
	// 개발 모드 설정
	DevelopmentMode bool // 개발 모드 (일부 보안 헤더 완화)
	
	Logger *zap.Logger
}

// SecurityHeaders는 보안 헤더 미들웨어입니다.
type SecurityHeaders struct {
	config *SecurityHeadersConfig
	logger *zap.Logger
}

// DefaultSecurityHeadersConfig는 기본 보안 헤더 설정을 반환합니다.
func DefaultSecurityHeadersConfig() *SecurityHeadersConfig {
	return &SecurityHeadersConfig{
		// HSTS 설정 (1년)
		HSTSMaxAge:            31536000,
		HSTSIncludeSubdomains: true,
		HSTSPreload:           false,
		
		// CSP 기본 설정
		CSPDefaultSrc: []string{"'self'"},
		CSPScriptSrc:  []string{"'self'", "'unsafe-inline'"},
		CSPStyleSrc:   []string{"'self'", "'unsafe-inline'"},
		CSPImgSrc:     []string{"'self'", "data:", "https:"},
		CSPConnectSrc: []string{"'self'"},
		CSPFontSrc:    []string{"'self'"},
		CSPObjectSrc:  []string{"'none'"},
		CSPMediaSrc:   []string{"'self'"},
		CSPFrameSrc:   []string{"'none'"},
		CSPReportOnly: false,
		
		// 기본 보안 헤더
		FrameOptions:       "SAMEORIGIN",
		ContentTypeNosniff: true,
		XSSProtection:      "1; mode=block",
		ReferrerPolicy:     "strict-origin-when-cross-origin",
		
		// Cross-Origin 정책
		CrossOriginEmbedderPolicy: "require-corp",
		CrossOriginOpenerPolicy:   "same-origin",
		CrossOriginResourcePolicy: "same-origin",
		
		// Permissions Policy 기본 설정
		PermissionsPolicy: map[string][]string{
			"geolocation":    {"'none'"},
			"microphone":     {"'none'"},
			"camera":         {"'none'"},
			"payment":        {"'none'"},
			"usb":           {"'none'"},
			"accelerometer": {"'none'"},
			"gyroscope":     {"'none'"},
			"magnetometer":  {"'none'"},
		},
		
		CustomHeaders: make(map[string]string),
		DevelopmentMode: false,
	}
}

// DevelopmentSecurityHeadersConfig는 개발용 보안 헤더 설정을 반환합니다.
func DevelopmentSecurityHeadersConfig() *SecurityHeadersConfig {
	config := DefaultSecurityHeadersConfig()
	
	// 개발 모드 설정
	config.DevelopmentMode = true
	config.HSTSMaxAge = 0 // HSTS 비활성화
	
	// CSP 완화
	config.CSPScriptSrc = append(config.CSPScriptSrc, "'unsafe-eval'")
	config.CSPConnectSrc = append(config.CSPConnectSrc, "ws:", "wss:")
	
	// 개발 도구 허용
	config.CrossOriginEmbedderPolicy = "unsafe-none"
	
	return config
}

// NewSecurityHeaders는 새로운 보안 헤더 미들웨어를 생성합니다.
func NewSecurityHeaders(config *SecurityHeadersConfig) *SecurityHeaders {
	if config == nil {
		config = DefaultSecurityHeadersConfig()
	}
	
	return &SecurityHeaders{
		config: config,
		logger: config.Logger,
	}
}

// SecurityHeadersMiddleware는 보안 헤더 미들웨어를 생성합니다.
func SecurityHeadersMiddleware(config *SecurityHeadersConfig) gin.HandlerFunc {
	sh := NewSecurityHeaders(config)
	return sh.Handler()
}

// Handler는 보안 헤더 미들웨어 핸들러를 반환합니다.
func (sh *SecurityHeaders) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// HSTS 헤더 설정
		sh.setHSTSHeader(c)
		
		// CSP 헤더 설정
		sh.setCSPHeader(c)
		
		// X-Frame-Options 헤더 설정
		sh.setFrameOptionsHeader(c)
		
		// X-Content-Type-Options 헤더 설정
		sh.setContentTypeOptionsHeader(c)
		
		// X-XSS-Protection 헤더 설정
		sh.setXSSProtectionHeader(c)
		
		// Referrer-Policy 헤더 설정
		sh.setReferrerPolicyHeader(c)
		
		// Permissions-Policy 헤더 설정
		sh.setPermissionsPolicyHeader(c)
		
		// Cross-Origin 헤더 설정
		sh.setCrossOriginHeaders(c)
		
		// Custom 헤더 설정
		sh.setCustomHeaders(c)
		
		// Server 헤더 숨기기
		sh.hideServerHeader(c)
		
		c.Next()
	}
}

// setHSTSHeader는 HSTS 헤더를 설정합니다.
func (sh *SecurityHeaders) setHSTSHeader(c *gin.Context) {
	// 개발 모드이거나 HTTPS가 아닌 경우 HSTS 비활성화
	if sh.config.DevelopmentMode || c.Request.TLS == nil {
		return
	}
	
	if sh.config.HSTSMaxAge > 0 {
		hstsValue := fmt.Sprintf("max-age=%d", sh.config.HSTSMaxAge)
		
		if sh.config.HSTSIncludeSubdomains {
			hstsValue += "; includeSubDomains"
		}
		
		if sh.config.HSTSPreload {
			hstsValue += "; preload"
		}
		
		c.Header("Strict-Transport-Security", hstsValue)
	}
}

// setCSPHeader는 Content Security Policy 헤더를 설정합니다.
func (sh *SecurityHeaders) setCSPHeader(c *gin.Context) {
	cspDirectives := make([]string, 0)
	
	// 각 CSP 지시문 설정
	if len(sh.config.CSPDefaultSrc) > 0 {
		cspDirectives = append(cspDirectives, 
			fmt.Sprintf("default-src %s", strings.Join(sh.config.CSPDefaultSrc, " ")))
	}
	
	if len(sh.config.CSPScriptSrc) > 0 {
		cspDirectives = append(cspDirectives,
			fmt.Sprintf("script-src %s", strings.Join(sh.config.CSPScriptSrc, " ")))
	}
	
	if len(sh.config.CSPStyleSrc) > 0 {
		cspDirectives = append(cspDirectives,
			fmt.Sprintf("style-src %s", strings.Join(sh.config.CSPStyleSrc, " ")))
	}
	
	if len(sh.config.CSPImgSrc) > 0 {
		cspDirectives = append(cspDirectives,
			fmt.Sprintf("img-src %s", strings.Join(sh.config.CSPImgSrc, " ")))
	}
	
	if len(sh.config.CSPConnectSrc) > 0 {
		cspDirectives = append(cspDirectives,
			fmt.Sprintf("connect-src %s", strings.Join(sh.config.CSPConnectSrc, " ")))
	}
	
	if len(sh.config.CSPFontSrc) > 0 {
		cspDirectives = append(cspDirectives,
			fmt.Sprintf("font-src %s", strings.Join(sh.config.CSPFontSrc, " ")))
	}
	
	if len(sh.config.CSPObjectSrc) > 0 {
		cspDirectives = append(cspDirectives,
			fmt.Sprintf("object-src %s", strings.Join(sh.config.CSPObjectSrc, " ")))
	}
	
	if len(sh.config.CSPMediaSrc) > 0 {
		cspDirectives = append(cspDirectives,
			fmt.Sprintf("media-src %s", strings.Join(sh.config.CSPMediaSrc, " ")))
	}
	
	if len(sh.config.CSPFrameSrc) > 0 {
		cspDirectives = append(cspDirectives,
			fmt.Sprintf("frame-src %s", strings.Join(sh.config.CSPFrameSrc, " ")))
	}
	
	if len(sh.config.CSPSandbox) > 0 {
		cspDirectives = append(cspDirectives,
			fmt.Sprintf("sandbox %s", strings.Join(sh.config.CSPSandbox, " ")))
	}
	
	if sh.config.CSPReportURI != "" {
		cspDirectives = append(cspDirectives,
			fmt.Sprintf("report-uri %s", sh.config.CSPReportURI))
	}
	
	if len(cspDirectives) > 0 {
		cspValue := strings.Join(cspDirectives, "; ")
		headerName := "Content-Security-Policy"
		
		if sh.config.CSPReportOnly {
			headerName = "Content-Security-Policy-Report-Only"
		}
		
		c.Header(headerName, cspValue)
	}
}

// setFrameOptionsHeader는 X-Frame-Options 헤더를 설정합니다.
func (sh *SecurityHeaders) setFrameOptionsHeader(c *gin.Context) {
	if sh.config.FrameOptions != "" {
		c.Header("X-Frame-Options", sh.config.FrameOptions)
	}
}

// setContentTypeOptionsHeader는 X-Content-Type-Options 헤더를 설정합니다.
func (sh *SecurityHeaders) setContentTypeOptionsHeader(c *gin.Context) {
	if sh.config.ContentTypeNosniff {
		c.Header("X-Content-Type-Options", "nosniff")
	}
}

// setXSSProtectionHeader는 X-XSS-Protection 헤더를 설정합니다.
func (sh *SecurityHeaders) setXSSProtectionHeader(c *gin.Context) {
	if sh.config.XSSProtection != "" {
		c.Header("X-XSS-Protection", sh.config.XSSProtection)
	}
}

// setReferrerPolicyHeader는 Referrer-Policy 헤더를 설정합니다.
func (sh *SecurityHeaders) setReferrerPolicyHeader(c *gin.Context) {
	if sh.config.ReferrerPolicy != "" {
		c.Header("Referrer-Policy", sh.config.ReferrerPolicy)
	}
}

// setPermissionsPolicyHeader는 Permissions-Policy 헤더를 설정합니다.
func (sh *SecurityHeaders) setPermissionsPolicyHeader(c *gin.Context) {
	if len(sh.config.PermissionsPolicy) > 0 {
		policies := make([]string, 0, len(sh.config.PermissionsPolicy))
		
		for feature, allowlist := range sh.config.PermissionsPolicy {
			policy := fmt.Sprintf("%s=(%s)", feature, strings.Join(allowlist, " "))
			policies = append(policies, policy)
		}
		
		if len(policies) > 0 {
			c.Header("Permissions-Policy", strings.Join(policies, ", "))
		}
	}
}

// setCrossOriginHeaders는 Cross-Origin 관련 헤더를 설정합니다.
func (sh *SecurityHeaders) setCrossOriginHeaders(c *gin.Context) {
	if sh.config.CrossOriginEmbedderPolicy != "" {
		c.Header("Cross-Origin-Embedder-Policy", sh.config.CrossOriginEmbedderPolicy)
	}
	
	if sh.config.CrossOriginOpenerPolicy != "" {
		c.Header("Cross-Origin-Opener-Policy", sh.config.CrossOriginOpenerPolicy)
	}
	
	if sh.config.CrossOriginResourcePolicy != "" {
		c.Header("Cross-Origin-Resource-Policy", sh.config.CrossOriginResourcePolicy)
	}
}

// setCustomHeaders는 사용자 정의 헤더를 설정합니다.
func (sh *SecurityHeaders) setCustomHeaders(c *gin.Context) {
	for name, value := range sh.config.CustomHeaders {
		c.Header(name, value)
	}
}

// hideServerHeader는 Server 헤더를 숨깁니다.
func (sh *SecurityHeaders) hideServerHeader(c *gin.Context) {
	c.Header("Server", "")
}

// SecurityHeadersBuilder는 보안 헤더 설정을 빌더 패턴으로 구성할 수 있게 해줍니다.
type SecurityHeadersBuilder struct {
	config *SecurityHeadersConfig
}

// NewSecurityHeadersBuilder는 새로운 보안 헤더 빌더를 생성합니다.
func NewSecurityHeadersBuilder() *SecurityHeadersBuilder {
	return &SecurityHeadersBuilder{
		config: DefaultSecurityHeadersConfig(),
	}
}

// WithHSTS는 HSTS 설정을 추가합니다.
func (b *SecurityHeadersBuilder) WithHSTS(maxAge int, includeSubdomains, preload bool) *SecurityHeadersBuilder {
	b.config.HSTSMaxAge = maxAge
	b.config.HSTSIncludeSubdomains = includeSubdomains
	b.config.HSTSPreload = preload
	return b
}

// WithCSP는 CSP 설정을 추가합니다.
func (b *SecurityHeadersBuilder) WithCSP(defaultSrc, scriptSrc, styleSrc []string) *SecurityHeadersBuilder {
	b.config.CSPDefaultSrc = defaultSrc
	b.config.CSPScriptSrc = scriptSrc
	b.config.CSPStyleSrc = styleSrc
	return b
}

// WithFrameOptions는 X-Frame-Options를 설정합니다.
func (b *SecurityHeadersBuilder) WithFrameOptions(options string) *SecurityHeadersBuilder {
	b.config.FrameOptions = options
	return b
}

// WithContentTypeOptions는 X-Content-Type-Options를 설정합니다.
func (b *SecurityHeadersBuilder) WithContentTypeOptions(nosniff bool) *SecurityHeadersBuilder {
	b.config.ContentTypeNosniff = nosniff
	return b
}

// WithXSSProtection은 X-XSS-Protection을 설정합니다.
func (b *SecurityHeadersBuilder) WithXSSProtection(protection string) *SecurityHeadersBuilder {
	b.config.XSSProtection = protection
	return b
}

// WithReferrerPolicy는 Referrer-Policy를 설정합니다.
func (b *SecurityHeadersBuilder) WithReferrerPolicy(policy string) *SecurityHeadersBuilder {
	b.config.ReferrerPolicy = policy
	return b
}

// WithPermissionsPolicy는 Permissions-Policy를 추가합니다.
func (b *SecurityHeadersBuilder) WithPermissionsPolicy(feature string, allowlist []string) *SecurityHeadersBuilder {
	if b.config.PermissionsPolicy == nil {
		b.config.PermissionsPolicy = make(map[string][]string)
	}
	b.config.PermissionsPolicy[feature] = allowlist
	return b
}

// WithCustomHeader는 사용자 정의 헤더를 추가합니다.
func (b *SecurityHeadersBuilder) WithCustomHeader(name, value string) *SecurityHeadersBuilder {
	if b.config.CustomHeaders == nil {
		b.config.CustomHeaders = make(map[string]string)
	}
	b.config.CustomHeaders[name] = value
	return b
}

// WithDevelopmentMode는 개발 모드를 설정합니다.
func (b *SecurityHeadersBuilder) WithDevelopmentMode(enabled bool) *SecurityHeadersBuilder {
	b.config.DevelopmentMode = enabled
	return b
}

// WithLogger는 로거를 설정합니다.
func (b *SecurityHeadersBuilder) WithLogger(logger *zap.Logger) *SecurityHeadersBuilder {
	b.config.Logger = logger
	return b
}

// Build는 최종 설정으로 미들웨어를 생성합니다.
func (b *SecurityHeadersBuilder) Build() gin.HandlerFunc {
	return SecurityHeadersMiddleware(b.config)
}

// GetConfig는 현재 설정을 반환합니다.
func (b *SecurityHeadersBuilder) GetConfig() *SecurityHeadersConfig {
	return b.config
}

// SecurityHeadersInfo는 설정된 보안 헤더 정보를 반환합니다.
func (sh *SecurityHeaders) GetSecurityHeadersInfo() map[string]interface{} {
	info := make(map[string]interface{})
	
	info["hsts_enabled"] = sh.config.HSTSMaxAge > 0 && !sh.config.DevelopmentMode
	info["hsts_max_age"] = sh.config.HSTSMaxAge
	info["hsts_include_subdomains"] = sh.config.HSTSIncludeSubdomains
	info["hsts_preload"] = sh.config.HSTSPreload
	
	info["csp_enabled"] = len(sh.config.CSPDefaultSrc) > 0
	info["csp_report_only"] = sh.config.CSPReportOnly
	
	info["frame_options"] = sh.config.FrameOptions
	info["content_type_nosniff"] = sh.config.ContentTypeNosniff
	info["xss_protection"] = sh.config.XSSProtection
	info["referrer_policy"] = sh.config.ReferrerPolicy
	
	info["permissions_policy_count"] = len(sh.config.PermissionsPolicy)
	info["custom_headers_count"] = len(sh.config.CustomHeaders)
	
	info["development_mode"] = sh.config.DevelopmentMode
	
	return info
}

// ValidateConfig는 보안 헤더 설정을 검증합니다.
func (sh *SecurityHeaders) ValidateConfig() []string {
	var warnings []string
	
	// HSTS 검증
	if sh.config.HSTSMaxAge > 0 && sh.config.HSTSMaxAge < 3600 {
		warnings = append(warnings, "HSTS Max-Age가 너무 짧습니다 (최소 1시간 권장)")
	}
	
	// CSP 검증
	if len(sh.config.CSPScriptSrc) > 0 {
		for _, src := range sh.config.CSPScriptSrc {
			if src == "'unsafe-inline'" || src == "'unsafe-eval'" {
				warnings = append(warnings, fmt.Sprintf("CSP script-src에 안전하지 않은 지시문이 포함되어 있습니다: %s", src))
			}
		}
	}
	
	// Frame Options 검증
	validFrameOptions := []string{"DENY", "SAMEORIGIN"}
	if sh.config.FrameOptions != "" {
		valid := false
		for _, validOption := range validFrameOptions {
			if sh.config.FrameOptions == validOption || strings.HasPrefix(sh.config.FrameOptions, "ALLOW-FROM") {
				valid = true
				break
			}
		}
		if !valid {
			warnings = append(warnings, "유효하지 않은 X-Frame-Options 값입니다")
		}
	}
	
	return warnings
}