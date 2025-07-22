package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/aicli/aicli-web/internal/server/handlers"
	apiHandlers "github.com/aicli/aicli-web/internal/api/handlers"
	"github.com/aicli/aicli-web/internal/api/controllers"
	"github.com/aicli/aicli-web/internal/middleware"
	"github.com/aicli/aicli-web/pkg/version"
	"github.com/aicli/aicli-web/internal/docs"
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
		authHandler := apiHandlers.NewAuthHandler(s.jwtManager, s.blacklist, s.oauthManager)
		
		// 인증 엔드포인트 (인증 불필요)
		auth := v1.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.Refresh)
			auth.POST("/logout", authHandler.Logout)
			
			// OAuth 엔드포인트
			oauth := auth.Group("/oauth")
			{
				oauth.GET("/:provider", authHandler.OAuthLogin)
				oauth.GET("/:provider/callback", authHandler.OAuthCallback)
			}
		}
		
		// 시스템 정보 엔드포인트
		system := v1.Group("/system")
		{
			system.GET("/info", apiHandlers.GetSystemInfo)
			system.GET("/status", apiHandlers.GetSystemStatus)
		}

		// 워크스페이스 컨트롤러 인스턴스 생성
		workspaceController := controllers.NewWorkspaceController(s.workspaceService)
		
		// 프로젝트 컨트롤러 인스턴스 생성
		projectController := controllers.NewProjectController(s.storage)
		
		// 세션 컨트롤러 인스턴스 생성
		sessionController := controllers.NewSessionController(s.sessionService)
		
		// 태스크 컨트롤러 인스턴스 생성
		taskController := controllers.NewTaskController(s.taskService)
		
		// RBAC 컨트롤러 인스턴스 생성
		rbacController := controllers.NewRBACController(s.rbacManager, s.storage)

		// Claude 핸들러 인스턴스 생성 (Claude wrapper가 있다고 가정)
		// TODO: s.claudeWrapper가 Server 구조체에 추가되어야 함
		claudeHandler := handlers.NewClaudeHandler(s.claudeWrapper, s.storage.Session(), s.wsHub)

		// Claude 관련 엔드포인트 (인증 필요)
		claude := v1.Group("/claude")
		claude.Use(middleware.RequireAuth(s.jwtManager, s.blacklist))
		claude.Use(middleware.ClaudeErrorHandler())
		{
			// Claude 실행 및 세션 관리
			claude.POST("/execute", claudeHandler.Execute)
			claude.GET("/sessions", claudeHandler.ListSessions)
			claude.GET("/sessions/:id", claudeHandler.GetSession)
			claude.DELETE("/sessions/:id", claudeHandler.CloseSession)
			claude.GET("/sessions/:id/logs", claudeHandler.GetSessionLogs)
		}

		// 워크스페이스 관련 엔드포인트 (인증 필요)
		workspaces := v1.Group("/workspaces")
		workspaces.Use(middleware.RequireAuth(s.jwtManager, s.blacklist))
		{
			workspaces.GET("", workspaceController.ListWorkspaces)
			workspaces.POST("", workspaceController.CreateWorkspace)
			workspaces.GET("/:id", workspaceController.GetWorkspace)
			workspaces.PUT("/:id", workspaceController.UpdateWorkspace)
			workspaces.DELETE("/:id", workspaceController.DeleteWorkspace)
			
			// 워크스페이스 내 프로젝트 엔드포인트
			workspaces.POST("/:workspace_id/projects", projectController.CreateProject)
			workspaces.GET("/:workspace_id/projects", projectController.ListProjects)
		}
		
		// 프로젝트 관련 엔드포인트 (인증 필요)
		projects := v1.Group("/projects")
		projects.Use(middleware.RequireAuth(s.jwtManager, s.blacklist))
		{
			projects.GET("/:id", projectController.GetProject)
			projects.PUT("/:id", projectController.UpdateProject)
			projects.DELETE("/:id", projectController.DeleteProject)
			
			// 프로젝트별 세션 생성
			projects.POST("/:id/sessions", sessionController.Create)
		}
		
		// 세션 관련 엔드포인트 (인증 필요)
		sessions := v1.Group("/sessions")
		sessions.Use(middleware.RequireAuth(s.jwtManager, s.blacklist))
		{
			sessions.GET("", sessionController.List)
			sessions.GET("/active", sessionController.GetActiveSessions)
			sessions.GET("/:id", sessionController.GetByID)
			sessions.DELETE("/:id", sessionController.Terminate)
			sessions.PUT("/:id/activity", sessionController.UpdateActivity)
			
			// 세션별 태스크 생성
			sessions.POST("/:sessionId/tasks", taskController.Create)
		}

		// 태스크 관련 엔드포인트 (인증 필요)
		tasks := v1.Group("/tasks")
		tasks.Use(middleware.RequireAuth(s.jwtManager, s.blacklist))
		{
			tasks.GET("", taskController.List)
			tasks.GET("/active", taskController.GetActiveTasks)
			tasks.GET("/stats", taskController.GetStats)
			tasks.GET("/:id", taskController.GetByID)
			tasks.DELETE("/:id", taskController.Cancel)
		}

		// 로그 관련 엔드포인트 (인증 필요)
		logs := v1.Group("/logs")
		logs.Use(middleware.RequireAuth(s.jwtManager, s.blacklist))
		{
			logs.GET("/workspaces/:id", handlers.GetWorkspaceLogs)
			logs.GET("/tasks/:id", handlers.GetTaskLogs)
			// TODO: WebSocket 엔드포인트는 나중에 추가
		}

		// RBAC 관련 엔드포인트 (인증 필요)
		rbac := v1.Group("/rbac")
		rbac.Use(middleware.RequireAuth(s.jwtManager, s.blacklist))
		{
			// 역할 관리 API
			roles := rbac.Group("/roles")
			{
				roles.GET("", rbacController.ListRoles)
				roles.POST("", middleware.RequirePermission(s.rbacManager, models.ResourceTypeSystem, models.ActionManage), rbacController.CreateRole)
				roles.GET("/:id", rbacController.GetRole)
				roles.PUT("/:id", middleware.RequirePermission(s.rbacManager, models.ResourceTypeSystem, models.ActionManage), rbacController.UpdateRole)
				roles.DELETE("/:id", middleware.RequirePermission(s.rbacManager, models.ResourceTypeSystem, models.ActionManage), rbacController.DeleteRole)
			}
			
			// 권한 관리 API
			permissions := rbac.Group("/permissions")
			{
				permissions.GET("", rbacController.ListPermissions)
				permissions.POST("", middleware.RequirePermission(s.rbacManager, models.ResourceTypeSystem, models.ActionManage), rbacController.CreatePermission)
			}
			
			// 사용자 역할 관리 API
			userRoles := rbac.Group("/user-roles")
			{
				userRoles.POST("", middleware.RequirePermission(s.rbacManager, models.ResourceTypeUser, models.ActionManage), rbacController.AssignRoleToUser)
			}
			
			// 권한 확인 API
			rbac.POST("/check-permission", rbacController.CheckPermission)
			
			// 사용자 권한 조회 API
			rbac.GET("/users/:user_id/permissions", rbacController.GetUserPermissions)
			
			// 캐시 관리 API (관리자만)
			cache := rbac.Group("/cache")
			cache.Use(middleware.RequirePermission(s.rbacManager, models.ResourceTypeSystem, models.ActionManage))
			{
				cache.POST("/invalidate", rbacController.InvalidateCache)
			}
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
	
	// WebSocket 엔드포인트
	s.router.GET("/ws", s.wsHandler.HandleConnection)
	
	// Claude WebSocket 스트림 엔드포인트
	s.router.GET("/ws/executions/:executionID", s.claudeStreamHandler.HandleConnection)

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