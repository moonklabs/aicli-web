package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/drumcap/aicli-web/internal/server/handlers"
	apiHandlers "github.com/drumcap/aicli-web/internal/api/handlers"
	"github.com/drumcap/aicli-web/internal/api/controllers"
	"github.com/drumcap/aicli-web/pkg/version"
)

// setupRoutes는 모든 API 라우트를 설정합니다.
func (s *Server) setupRoutes() {
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
		// 시스템 정보 엔드포인트
		system := v1.Group("/system")
		{
			system.GET("/info", apiHandlers.GetSystemInfo)
			system.GET("/status", apiHandlers.GetSystemStatus)
		}

		// 워크스페이스 컨트롤러 인스턴스 생성
		workspaceController := controllers.NewWorkspaceController()

		// 워크스페이스 관련 엔드포인트
		workspaces := v1.Group("/workspaces")
		{
			workspaces.GET("", workspaceController.ListWorkspaces)
			workspaces.POST("", workspaceController.CreateWorkspace)
			workspaces.GET("/:id", workspaceController.GetWorkspace)
			workspaces.PUT("/:id", workspaceController.UpdateWorkspace)
			workspaces.DELETE("/:id", workspaceController.DeleteWorkspace)
		}

		// 태스크 관련 엔드포인트 (기존 스텁 유지)
		tasks := v1.Group("/tasks")
		{
			tasks.GET("", handlers.ListTasks)
			tasks.POST("", handlers.CreateTask)
			tasks.GET("/:id", handlers.GetTask)
			tasks.DELETE("/:id", handlers.CancelTask)
		}

		// 로그 관련 엔드포인트 (기존 스텁 유지)
		logs := v1.Group("/logs")
		{
			logs.GET("/workspaces/:id", handlers.GetWorkspaceLogs)
			logs.GET("/tasks/:id", handlers.GetTaskLogs)
			// TODO: WebSocket 엔드포인트는 나중에 추가
		}

		// 설정 관련 엔드포인트 (기존 스텁 유지)
		config := v1.Group("/config")
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
				routeInfo := make([]gin.H, len(routes))
				for i, route := range routes {
					routeInfo[i] = gin.H{
						"method": route.Method,
						"path":   route.Path,
					}
				}
				c.JSON(http.StatusOK, gin.H{
					"routes": routeInfo,
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