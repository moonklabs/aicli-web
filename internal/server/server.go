package server

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"aicli-web/internal/auth"
	"aicli-web/internal/config"
	"github.com/drumcap/aicli-web/internal/middleware"
)

// Server는 API 서버의 핵심 구조체입니다.
type Server struct {
	router      *gin.Engine
	jwtManager  *auth.JWTManager
	blacklist   *auth.Blacklist
	// TODO: 데이터베이스, 클라이언트 등 의존성은 나중에 추가
}

// New는 새로운 서버 인스턴스를 생성합니다.
func New() *Server {
	// 설정 로드
	cfg := config.DefaultConfig()
	
	// JWT 매니저 초기화
	jwtManager := auth.NewJWTManager(
		cfg.API.JWTSecret,
		cfg.API.AccessTokenExpiry,
		cfg.API.RefreshTokenExpiry,
	)
	
	// 블랙리스트 초기화
	blacklist := auth.NewBlacklist()
	
	s := &Server{
		jwtManager: jwtManager,
		blacklist:  blacklist,
	}
	s.setupRouter()
	return s
}

// Router는 Gin 라우터를 반환합니다.
func (s *Server) Router() *gin.Engine {
	return s.router
}

// setupRouter는 라우터를 설정합니다.
func (s *Server) setupRouter() {
	// 환경에 따른 Gin 모드 설정
	env := viper.GetString("env")
	if env == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// 라우터 생성
	s.router = gin.New()

	// 미들웨어 설정 (순서 중요!)
	s.router.Use(middleware.RequestID())    // 요청 ID 생성 (가장 먼저)
	s.router.Use(middleware.Security())     // 보안 헤더
	s.router.Use(middleware.CORS())         // CORS 설정
	s.router.Use(middleware.Logger())       // 기본 로깅
	s.router.Use(middleware.RequestLogger()) // 상세 요청 로깅
	s.router.Use(middleware.GracefulRecovery()) // 패닉 복구
	s.router.Use(middleware.ErrorHandler())  // 에러 처리 (마지막)

	// 라우터 설정
	s.setupRoutes()
}