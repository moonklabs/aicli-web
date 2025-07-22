package server

import (
	"context"
	
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/aicli/aicli-web/internal/auth"
	"github.com/aicli/aicli-web/internal/config"
	"github.com/aicli/aicli-web/internal/services"
	"github.com/aicli/aicli-web/internal/storage"
	"github.com/aicli/aicli-web/internal/storage/memory"
	"github.com/aicli/aicli-web/internal/utils"
	"github.com/aicli/aicli-web/internal/middleware"
	"github.com/aicli/aicli-web/internal/websocket"
	"github.com/aicli/aicli-web/internal/claude"
	"github.com/aicli/aicli-web/internal/docker"
	"github.com/aicli/aicli-web/internal/api/controllers"
)

// Server는 API 서버의 핵심 구조체입니다.
type Server struct {
	router         *gin.Engine
	jwtManager     *auth.JWTManager
	blacklist      *auth.Blacklist
	oauthManager   auth.OAuthManager
	storage          storage.Storage
	workspaceService services.WorkspaceService
	dockerWorkspaceService *services.DockerWorkspaceService // Docker 통합 워크스페이스 서비스 추가
	sessionService   *services.SessionService
	taskService      *services.TaskService
	
	// Claude 관련
	claudeWrapper        claude.Wrapper
	claudeStreamHandler  *websocket.ClaudeStreamHandler
	executionTracker     *claude.ExecutionTracker
	
	// WebSocket 관련
	wsHub     *websocket.Hub
	wsHandler *websocket.WebSocketHandler
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
	
	// OAuth 매니저 초기화
	oauthConfigs := make(map[auth.OAuthProvider]*auth.OAuthConfig)
	
	// Google OAuth 설정
	if cfg.API.OAuth.Google.Enabled {
		oauthConfigs[auth.ProviderGoogle] = &auth.OAuthConfig{
			Provider:     auth.ProviderGoogle,
			ClientID:     cfg.API.OAuth.Google.ClientID,
			ClientSecret: cfg.API.OAuth.Google.ClientSecret,
			RedirectURL:  cfg.API.OAuth.BaseRedirectURL + "/google/callback",
			Scopes:       cfg.API.OAuth.Google.Scopes,
			Enabled:      true,
		}
	}
	
	// GitHub OAuth 설정
	if cfg.API.OAuth.GitHub.Enabled {
		oauthConfigs[auth.ProviderGitHub] = &auth.OAuthConfig{
			Provider:     auth.ProviderGitHub,
			ClientID:     cfg.API.OAuth.GitHub.ClientID,
			ClientSecret: cfg.API.OAuth.GitHub.ClientSecret,
			RedirectURL:  cfg.API.OAuth.BaseRedirectURL + "/github/callback",
			Scopes:       cfg.API.OAuth.GitHub.Scopes,
			Enabled:      true,
		}
	}
	
	oauthManager := auth.NewOAuthManager(oauthConfigs, jwtManager)
	
	// 스토리지 초기화 (개발 환경에서는 메모리 스토리지 사용)
	storage := memory.New()
	
	// 워크스페이스 서비스 초기화
	workspaceService := services.NewWorkspaceService(storage)
	
	// Docker 매니저 초기화 (선택적)
	var dockerWorkspaceService *services.DockerWorkspaceService
	dockerManager, err := docker.NewManagerWithDefaults()
	if err != nil {
		// Docker를 사용할 수 없는 경우 로깅만 하고 계속 진행
		// TODO: 로거 추가 시 로깅
		dockerWorkspaceService = nil
	} else {
		// Docker 통합 워크스페이스 서비스 초기화
		dockerWorkspaceService = services.NewDockerWorkspaceService(workspaceService, storage, dockerManager)
	}
	
	// 프로젝트 서비스 초기화
	projectService := services.NewProjectService(storage)
	
	// 세션 서비스 초기화
	sessionService := services.NewSessionService(storage, projectService, nil)
	
	// 태스크 서비스 초기화
	taskService := services.NewTaskService(storage, sessionService, nil)
	
	// WebSocket 허브 초기화
	wsHub := websocket.NewHub(nil)
	
	// WebSocket 핸들러 초기화
	wsHandler := websocket.NewWebSocketHandler(wsHub, jwtManager, blacklist, nil)
	
	// Claude 세션 매니저 초기화
	sessionManager := claude.NewSessionManager(storage.Session())
	
	// Claude 프로세스 매니저 초기화 (기존 구현체 사용)
	processManager := claude.NewProcessManager()
	
	// Claude 래퍼 초기화
	claudeWrapper := claude.NewWrapper(sessionManager, processManager)
	
	// Claude 스트림 핸들러 초기화
	claudeStreamHandler := websocket.NewClaudeStreamHandler(wsHub, claudeWrapper)
	
	// 실행 추적기 초기화
	executionTracker := claude.NewExecutionTracker(wsHub)
	
	s := &Server{
		jwtManager:           jwtManager,
		blacklist:            blacklist,
		oauthManager:         oauthManager,
		storage:              storage,
		workspaceService:     workspaceService,
		dockerWorkspaceService: dockerWorkspaceService,
		sessionService:       sessionService,
		taskService:          taskService,
		claudeWrapper:        claudeWrapper,
		claudeStreamHandler:  claudeStreamHandler,
		executionTracker:     executionTracker,
		wsHub:                wsHub,
		wsHandler:            wsHandler,
	}
	
	// 태스크 서비스 시작
	if err := taskService.Start(context.Background()); err != nil {
		// 에러 로깅하지만 서버는 계속 시작
		// TODO: 로거 추가 시 로깅
	}
	
	// WebSocket 허브 시작
	if err := wsHub.Start(); err != nil {
		// 에러 로깅하지만 서버는 계속 시작
		// TODO: 로거 추가 시 로깅
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
	s.router.Use(middleware.RateLimit(middleware.DefaultRateLimitConfig())) // Rate Limiting
	s.router.Use(middleware.Logger())       // 기본 로깅
	s.router.Use(middleware.RequestLogger()) // 상세 요청 로깅
	s.router.Use(middleware.GracefulRecovery()) // 패닉 복구
	s.router.Use(middleware.ErrorHandler())  // 에러 처리 (마지막)

	// 라우터 설정
	s.setupRoutes()
}

// setupRoutes는 API 엔드포인트를 설정합니다.
func (s *Server) setupRoutes() {
	// API 컨트롤러들 초기화
	s.setupControllers()
}

// setupControllers는 컨트롤러들을 초기화하고 라우트를 설정합니다.
func (s *Server) setupControllers() {
	// 컨트롤러들 초기화
	workspaceController := controllers.NewWorkspaceController(s.workspaceService, s.dockerWorkspaceService)
	
	// 헬스체크 엔드포인트
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "aicli-web"})
	})
	
	// API v1 그룹
	v1 := s.router.Group("/api/v1")
	{
		// 인증 없이 접근 가능한 엔드포인트
		public := v1.Group("/")
		{
			public.GET("/health", func(c *gin.Context) {
				c.JSON(200, gin.H{"status": "ok", "version": "v1"})
			})
		}
		
		// 인증이 필요한 엔드포인트
		protected := v1.Group("/")
		protected.Use(middleware.AuthMiddleware(s.jwtManager, s.blacklist))
		{
			// 워크스페이스 라우트
			workspaces := protected.Group("/workspaces")
			{
				workspaces.GET("/", workspaceController.ListWorkspaces)
				workspaces.POST("/", workspaceController.CreateWorkspace)
				workspaces.GET("/:id", workspaceController.GetWorkspace)
				workspaces.PUT("/:id", workspaceController.UpdateWorkspace)
				workspaces.DELETE("/:id", workspaceController.DeleteWorkspace)
				
				// Docker 통합 엔드포인트 (Docker 서비스가 활성화된 경우에만)
				if s.dockerWorkspaceService != nil {
					workspaces.GET("/:id/status", workspaceController.GetWorkspaceStatus)
					workspaces.POST("/batch", workspaceController.BatchWorkspaceOperation)
					workspaces.GET("/batch/:batch_id/status", workspaceController.GetBatchOperationStatus)
					workspaces.POST("/batch/:batch_id/cancel", workspaceController.CancelBatchOperation)
				}
			}
		}
	}
	
	// WebSocket 엔드포인트
	s.router.GET("/ws", middleware.AuthMiddleware(s.jwtManager, s.blacklist), s.wsHandler.HandleWebSocket)
}