package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/drumcap/aicli-web/internal/server/handlers"
	apiHandlers "github.com/drumcap/aicli-web/internal/api/handlers"
	"github.com/drumcap/aicli-web/internal/api/controllers"
	"github.com/drumcap/aicli-web/internal/middleware"
	"github.com/drumcap/aicli-web/pkg/version"
	"aicli-web/internal/docs"
)

// setupRoutes는 모든 API 라우트를 설정합니다.
func (s *Server) setupRoutes() {
	// Swagger UI 설정
	docs.SetupSwagger(s.router)
	
	// 루트 경로 - 기본 정보
	s.router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "AICode Manager API",
			"version": version.Version,
			"status":  "running",
		})
	})

	// 헬스체크 엔드포인트
	s.router.GET("/health", handlers.HealthCheck)

	// 버전 정보 엔드포인트
	s.router.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, version.Get())
	})

	// API v1 그룹
	v1 := s.router.Group("/api/v1")
	{
		// 인증 핸들러 생성
		authHandler := apiHandlers.NewAuthHandler(s.jwtManager, s.blacklist)
		
		// 인증 엔드포인트 (인증 불필요)
		auth := v1.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.Refresh)
			auth.POST("/logout", authHandler.Logout)
		}
		
		// 시스템 정보 엔드포인트
		system := v1.Group("/system")
		{
			system.GET("/info", apiHandlers.GetSystemInfo)
			system.GET("/status", apiHandlers.GetSystemStatus)
		}

		// 워크스페이스 컨트롤러 인스턴스 생성
		workspaceController := controllers.NewWorkspaceController(s.storage)

		// 워크스페이스 관련 엔드포인트 (인증 필요)
		workspaces := v1.Group("/workspaces")
		workspaces.Use(middleware.RequireAuth(s.jwtManager, s.blacklist))
		{
			workspaces.GET("", workspaceController.ListWorkspaces)
			workspaces.POST("", workspaceController.CreateWorkspace)
			workspaces.GET("/:id", workspaceController.GetWorkspace)
			workspaces.PUT("/:id", workspaceController.UpdateWorkspace)
			workspaces.DELETE("/:id", workspaceController.DeleteWorkspace)
		}

		// 태스크 관련 엔드포인트 (인증 필요)
		tasks := v1.Group("/tasks")
		tasks.Use(middleware.RequireAuth(s.jwtManager, s.blacklist))
		{
			tasks.GET("", handlers.ListTasks)
			tasks.POST("", handlers.CreateTask)
			tasks.GET("/:id", handlers.GetTask)
			tasks.DELETE("/:id", handlers.CancelTask)
		}

		// 로그 관련 엔드포인트 (인증 필요)
		logs := v1.Group("/logs")
		logs.Use(middleware.RequireAuth(s.jwtManager, s.blacklist))
		{
			logs.GET("/workspaces/:id", handlers.GetWorkspaceLogs)
			logs.GET("/tasks/:id", handlers.GetTaskLogs)
			// TODO: WebSocket 엔드포인트는 나중에 추가
		}

		// 설정 관련 엔드포인트 (인증 필요 + 관리자 권한)
		config := v1.Group("/config")
		config.Use(middleware.RequireAuth(s.jwtManager, s.blacklist))
		config.Use(middleware.RequireRole("admin"))
		{
			config.GET("", handlers.GetConfig)
			config.PUT("", handlers.UpdateConfig)
		}
	}

	// 개발 환경용 디버그 라우트
	if gin.Mode() == gin.DebugMode {
		debug := s.router.Group("/debug")
		{
			debug.GET("/routes", func(c *gin.Context) {
				routes := s.router.Routes()
				c.JSON(http.StatusOK, gin.H{
					"routes": routes,
					"count":  len(routes),
				})
			})
		}
	}

	// 404 핸들러
	s.router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Not Found",
			"message": "요청한 엔드포인트를 찾을 수 없습니다",
			"path":    c.Request.URL.Path,
		})
	})
}