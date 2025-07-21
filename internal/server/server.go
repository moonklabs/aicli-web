package server

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"aicli-web/internal/auth"
	"aicli-web/internal/config"
	"aicli-web/internal/storage"
	"aicli-web/internal/storage/memory"
	"aicli-web/internal/utils"
	"github.com/drumcap/aicli-web/internal/middleware"
)

// Server는 API 서버의 핵심 구조체입니다.
type Server struct {
	router      *gin.Engine
	jwtManager  *auth.JWTManager
	blacklist   *auth.Blacklist
	storage     storage.Storage
}

// New는 새로운 서버 인스턴스를 생성합니다.
func New() *Server {
	// 커스텀 validator 등록
	utils.RegisterCustomValidators()
	
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
	
	// 스토리지 초기화 (개발 환경에서는 메모리 스토리지 사용)
	storage := memory.New()
	
	s := &Server{
		jwtManager: jwtManager,
		blacklist:  blacklist,
		storage:    storage,
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