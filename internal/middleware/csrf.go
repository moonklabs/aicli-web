package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"

	"github.com/aicli/aicli-web/internal/errors"
)

// CSRFConfig는 CSRF 보호 설정입니다.
type CSRFConfig struct {
	// Redis 클라이언트 (토큰 저장용)
	Redis redis.UniversalClient
	
	// 토큰 설정
	TokenLength    int           // 토큰 길이 (바이트)
	TokenHeader    string        // CSRF 토큰 헤더명
	TokenField     string        // 폼 필드명
	CookieName     string        // 쿠키명
	TokenLifetime  time.Duration // 토큰 유효 시간
	
	// 보안 설정
	SecureCookie   bool          // HTTPS 전용 쿠키
	SameSite       http.SameSite // SameSite 설정
	CookieDomain   string        // 쿠키 도메인
	CookiePath     string        // 쿠키 경로
	
	// 검사 설정
	SkipMethods    []string      // 검사를 건너뛸 HTTP 메서드
	TrustedOrigins []string      // 신뢰할 수 있는 Origin 목록
	
	// 에러 핸들러
	ErrorHandler func(*gin.Context, error)
	
	Logger *zap.Logger
}

// CSRFProtection은 CSRF 보호 미들웨어입니다.
type CSRFProtection struct {
	config *CSRFConfig
	redis  redis.UniversalClient
	logger *zap.Logger
}

// CSRFToken은 CSRF 토큰 정보를 담습니다.
type CSRFToken struct {
	Token     string    `json:"token"`
	SessionID string    `json:"session_id"`
	UserID    string    `json:"user_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Used      bool      `json:"used"`
}

// DefaultCSRFConfig는 기본 CSRF 설정을 반환합니다.
func DefaultCSRFConfig() *CSRFConfig {
	return &CSRFConfig{
		TokenLength:   32,
		TokenHeader:   "X-CSRF-Token",
		TokenField:    "_csrf_token",
		CookieName:    "csrf_token",
		TokenLifetime: 24 * time.Hour,
		SecureCookie:  true,
		SameSite:      http.SameSiteStrictMode,
		CookiePath:    "/",
		SkipMethods:   []string{"GET", "HEAD", "OPTIONS"},
		TrustedOrigins: []string{},
		ErrorHandler:  defaultCSRFErrorHandler,
	}
}

// NewCSRFProtection은 새로운 CSRF 보호 미들웨어를 생성합니다.
func NewCSRFProtection(config *CSRFConfig) *CSRFProtection {
	if config == nil {
		config = DefaultCSRFConfig()
	}
	
	// 기본값 설정
	if config.TokenLength == 0 {
		config.TokenLength = 32
	}
	if config.TokenLifetime == 0 {
		config.TokenLifetime = 24 * time.Hour
	}
	if config.TokenHeader == "" {
		config.TokenHeader = "X-CSRF-Token"
	}
	if config.TokenField == "" {
		config.TokenField = "_csrf_token"
	}
	if config.CookieName == "" {
		config.CookieName = "csrf_token"
	}
	if config.CookiePath == "" {
		config.CookiePath = "/"
	}
	if config.SkipMethods == nil {
		config.SkipMethods = []string{"GET", "HEAD", "OPTIONS"}
	}
	if config.ErrorHandler == nil {
		config.ErrorHandler = defaultCSRFErrorHandler
	}

	return &CSRFProtection{
		config: config,
		redis:  config.Redis,
		logger: config.Logger,
	}
}

// CSRF는 CSRF 보호 미들웨어를 생성합니다.
func CSRF(config *CSRFConfig) gin.HandlerFunc {
	protection := NewCSRFProtection(config)
	return protection.Handler()
}

// Handler는 CSRF 보호 미들웨어 핸들러를 반환합니다.
func (cp *CSRFProtection) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 건너뛸 메서드 확인
		if cp.shouldSkipMethod(c.Request.Method) {
			// GET, HEAD, OPTIONS 요청에 대해서는 토큰 생성만
			cp.generateAndSetToken(c)
			c.Next()
			return
		}

		// Origin 검사
		if !cp.checkOrigin(c) {
			cp.config.ErrorHandler(c, errors.NewValidationError("CSRF_INVALID_ORIGIN", "Invalid origin"))
			return
		}

		// CSRF 토큰 검증
		if !cp.validateToken(c) {
			cp.config.ErrorHandler(c, errors.NewValidationError("CSRF_TOKEN_INVALID", "Invalid CSRF token"))
			return
		}

		// 토큰 갱신 (사용된 토큰 처리)
		cp.refreshToken(c)

		c.Next()
	}
}

// shouldSkipMethod는 해당 HTTP 메서드를 건너뛸지 확인합니다.
func (cp *CSRFProtection) shouldSkipMethod(method string) bool {
	for _, skipMethod := range cp.config.SkipMethods {
		if method == skipMethod {
			return true
		}
	}
	return false
}

// checkOrigin은 Origin 헤더를 검사합니다.
func (cp *CSRFProtection) checkOrigin(c *gin.Context) bool {
	origin := c.GetHeader("Origin")
	if origin == "" {
		// Origin 헤더가 없는 경우 Referer로 대체
		referer := c.GetHeader("Referer")
		if referer == "" {
			return false
		}
		
		// Referer에서 origin 추출
		if strings.Contains(referer, "://") {
			parts := strings.SplitN(referer, "://", 2)
			if len(parts) == 2 {
				hostPart := strings.SplitN(parts[1], "/", 2)[0]
				origin = parts[0] + "://" + hostPart
			}
		}
	}

	// 신뢰할 수 있는 Origin 목록 확인
	for _, trustedOrigin := range cp.config.TrustedOrigins {
		if origin == trustedOrigin {
			return true
		}
	}

	// 요청 호스트와 Origin 호스트 비교
	requestHost := c.Request.Host
	if requestHost != "" {
		expectedOrigin := "https://" + requestHost
		if c.Request.TLS == nil {
			expectedOrigin = "http://" + requestHost
		}
		return origin == expectedOrigin
	}

	return false
}

// validateToken은 CSRF 토큰을 검증합니다.
func (cp *CSRFProtection) validateToken(c *gin.Context) bool {
	// 토큰 추출
	token := cp.extractToken(c)
	if token == "" {
		return false
	}

	// 세션 ID 추출
	sessionID := cp.getSessionID(c)
	if sessionID == "" {
		return false
	}

	// Redis에서 토큰 검증
	if cp.redis != nil {
		return cp.validateTokenFromRedis(c, token, sessionID)
	}

	// Redis가 없는 경우 쿠키 기반 검증
	return cp.validateTokenFromCookie(c, token, sessionID)
}

// extractToken은 요청에서 CSRF 토큰을 추출합니다.
func (cp *CSRFProtection) extractToken(c *gin.Context) string {
	// 헤더에서 토큰 추출
	token := c.GetHeader(cp.config.TokenHeader)
	if token != "" {
		return token
	}

	// 폼 필드에서 토큰 추출
	token = c.PostForm(cp.config.TokenField)
	if token != "" {
		return token
	}

	// 쿼리 파라미터에서 토큰 추출
	token = c.Query(cp.config.TokenField)
	return token
}

// getSessionID는 세션 ID를 추출합니다.
func (cp *CSRFProtection) getSessionID(c *gin.Context) string {
	// 컨텍스트에서 세션 ID 추출
	if sessionID, exists := c.Get("session_id"); exists {
		if sid, ok := sessionID.(string); ok {
			return sid
		}
	}

	// 쿠키에서 세션 ID 추출
	if cookie, err := c.Cookie("session_id"); err == nil {
		return cookie
	}

	// 클라이언트 IP를 기본 식별자로 사용
	return c.ClientIP()
}

// validateTokenFromRedis는 Redis에서 토큰을 검증합니다.
func (cp *CSRFProtection) validateTokenFromRedis(c *gin.Context, token, sessionID string) bool {
	ctx := c.Request.Context()
	redisKey := cp.getRedisTokenKey(sessionID, token)

	// 토큰 존재 및 만료 확인
	exists, err := cp.redis.Exists(ctx, redisKey).Result()
	if err != nil {
		cp.logger.Error("CSRF 토큰 Redis 조회 실패", zap.Error(err))
		return false
	}

	if exists == 0 {
		return false
	}

	// 토큰 정보 조회
	tokenData, err := cp.redis.Get(ctx, redisKey).Result()
	if err != nil {
		cp.logger.Error("CSRF 토큰 데이터 조회 실패", zap.Error(err))
		return false
	}

	// 토큰 사용 표시 (일회성 토큰 정책)
	cp.redis.Del(ctx, redisKey)

	cp.logger.Debug("CSRF 토큰 검증 성공",
		zap.String("session_id", sessionID),
		zap.String("token", token[:8]+"..."))

	return tokenData != ""
}

// validateTokenFromCookie는 쿠키 기반으로 토큰을 검증합니다.
func (cp *CSRFProtection) validateTokenFromCookie(c *gin.Context, token, sessionID string) bool {
	// 쿠키에서 저장된 토큰 조회
	storedToken, err := c.Cookie(cp.config.CookieName)
	if err != nil {
		return false
	}

	// 토큰 비교 (타이밍 공격 방지)
	return subtle.ConstantTimeCompare([]byte(token), []byte(storedToken)) == 1
}

// generateAndSetToken은 새로운 CSRF 토큰을 생성하고 설정합니다.
func (cp *CSRFProtection) generateAndSetToken(c *gin.Context) {
	// 기존 토큰이 유효한 경우 재사용
	if cp.hasValidToken(c) {
		return
	}

	// 새 토큰 생성
	token, err := cp.generateToken()
	if err != nil {
		cp.logger.Error("CSRF 토큰 생성 실패", zap.Error(err))
		return
	}

	sessionID := cp.getSessionID(c)
	userID := cp.getUserID(c)

	// Redis에 토큰 저장
	if cp.redis != nil {
		cp.storeTokenInRedis(c, token, sessionID, userID)
	}

	// 쿠키에 토큰 설정
	cp.setTokenCookie(c, token)

	// 응답 헤더에 토큰 설정 (AJAX 요청용)
	c.Header(cp.config.TokenHeader, token)

	// 컨텍스트에 토큰 저장
	c.Set("csrf_token", token)
}

// hasValidToken은 유효한 토큰이 있는지 확인합니다.
func (cp *CSRFProtection) hasValidToken(c *gin.Context) bool {
	sessionID := cp.getSessionID(c)
	if sessionID == "" {
		return false
	}

	// 쿠키에서 토큰 확인
	storedToken, err := c.Cookie(cp.config.CookieName)
	if err != nil || storedToken == "" {
		return false
	}

	// Redis에서 유효성 확인
	if cp.redis != nil {
		ctx := c.Request.Context()
		redisKey := cp.getRedisTokenKey(sessionID, storedToken)
		exists, err := cp.redis.Exists(ctx, redisKey).Result()
		return err == nil && exists > 0
	}

	// Redis가 없으면 쿠키 존재만으로 유효하다고 판단
	return true
}

// generateToken은 새로운 CSRF 토큰을 생성합니다.
func (cp *CSRFProtection) generateToken() (string, error) {
	bytes := make([]byte, cp.config.TokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("토큰 생성 실패: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// storeTokenInRedis는 토큰을 Redis에 저장합니다.
func (cp *CSRFProtection) storeTokenInRedis(c *gin.Context, token, sessionID, userID string) {
	ctx := c.Request.Context()
	redisKey := cp.getRedisTokenKey(sessionID, token)

	// 토큰 정보를 JSON으로 저장
	err := cp.redis.Set(ctx, redisKey, "valid", cp.config.TokenLifetime).Err()
	if err != nil {
		cp.logger.Error("CSRF 토큰 Redis 저장 실패", zap.Error(err))
		return
	}

	cp.logger.Debug("CSRF 토큰 생성됨",
		zap.String("session_id", sessionID),
		zap.String("user_id", userID),
		zap.String("token", token[:8]+"..."))
}

// setTokenCookie는 토큰을 쿠키에 설정합니다.
func (cp *CSRFProtection) setTokenCookie(c *gin.Context, token string) {
	cookie := &http.Cookie{
		Name:     cp.config.CookieName,
		Value:    token,
		Path:     cp.config.CookiePath,
		Domain:   cp.config.CookieDomain,
		MaxAge:   int(cp.config.TokenLifetime.Seconds()),
		Secure:   cp.config.SecureCookie,
		HttpOnly: true,
		SameSite: cp.config.SameSite,
	}

	http.SetCookie(c.Writer, cookie)
}

// refreshToken은 사용된 토큰을 새 토큰으로 갱신합니다.
func (cp *CSRFProtection) refreshToken(c *gin.Context) {
	// 새 토큰 생성 및 설정
	cp.generateAndSetToken(c)
}

// getRedisTokenKey는 Redis 토큰 키를 생성합니다.
func (cp *CSRFProtection) getRedisTokenKey(sessionID, token string) string {
	return fmt.Sprintf("csrf:token:%s:%s", sessionID, token)
}

// getUserID는 사용자 ID를 추출합니다.
func (cp *CSRFProtection) getUserID(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(string); ok {
			return uid
		}
	}
	return ""
}

// GetToken은 현재 유효한 CSRF 토큰을 반환합니다.
func (cp *CSRFProtection) GetToken(c *gin.Context) string {
	if token, exists := c.Get("csrf_token"); exists {
		if t, ok := token.(string); ok {
			return t
		}
	}

	// 쿠키에서 토큰 조회
	if cookie, err := c.Cookie(cp.config.CookieName); err == nil {
		return cookie
	}

	return ""
}

// ClearToken은 CSRF 토큰을 제거합니다.
func (cp *CSRFProtection) ClearToken(c *gin.Context) {
	// 쿠키 제거
	cookie := &http.Cookie{
		Name:     cp.config.CookieName,
		Value:    "",
		Path:     cp.config.CookiePath,
		Domain:   cp.config.CookieDomain,
		MaxAge:   -1,
		Secure:   cp.config.SecureCookie,
		HttpOnly: true,
		SameSite: cp.config.SameSite,
	}
	http.SetCookie(c.Writer, cookie)

	// Redis에서 토큰 제거
	if cp.redis != nil {
		sessionID := cp.getSessionID(c)
		token := cp.GetToken(c)
		if sessionID != "" && token != "" {
			ctx := c.Request.Context()
			redisKey := cp.getRedisTokenKey(sessionID, token)
			cp.redis.Del(ctx, redisKey)
		}
	}
}

// GetStats는 CSRF 보호 통계를 반환합니다.
func (cp *CSRFProtection) GetStats(c *gin.Context) map[string]interface{} {
	stats := make(map[string]interface{})

	if cp.redis != nil {
		ctx := c.Request.Context()
		
		// 활성 토큰 수
		pattern := "csrf:token:*"
		var cursor uint64
		var count int

		for {
			keys, nextCursor, err := cp.redis.Scan(ctx, cursor, pattern, 100).Result()
			if err != nil {
				break
			}
			count += len(keys)
			cursor = nextCursor
			if cursor == 0 {
				break
			}
		}

		stats["active_tokens"] = count
	}

	return stats
}

// defaultCSRFErrorHandler는 기본 CSRF 에러 핸들러입니다.
func defaultCSRFErrorHandler(c *gin.Context, err error) {
	c.JSON(http.StatusForbidden, gin.H{
		"error":   "Forbidden",
		"message": err.Error(),
		"code":    "CSRF_TOKEN_INVALID",
	})
	c.Abort()
}

// CSRFTokenGenerator는 클라이언트에서 사용할 토큰 생성 엔드포인트를 제공합니다.
func CSRFTokenGenerator(cp *CSRFProtection) gin.HandlerFunc {
	return func(c *gin.Context) {
		cp.generateAndSetToken(c)
		token := cp.GetToken(c)
		
		c.JSON(http.StatusOK, gin.H{
			"csrf_token": token,
		})
	}
}