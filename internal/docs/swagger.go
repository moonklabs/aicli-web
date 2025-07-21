package docs

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// SetupSwagger Swagger UI 라우트를 설정합니다
func SetupSwagger(router *gin.Engine) {
	// Swagger UI 엔드포인트 설정
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	
	// /docs로도 접근 가능하도록 리다이렉트
	router.GET("/docs", func(c *gin.Context) {
		c.Redirect(302, "/swagger/index.html")
	})
}

// SwaggerInfo Swagger 정보를 반환합니다
type SwaggerInfo struct {
	Title       string
	Description string
	Version     string
	Host        string
	BasePath    string
}

// GetSwaggerInfo 현재 Swagger 설정 정보를 반환합니다
func GetSwaggerInfo() SwaggerInfo {
	return SwaggerInfo{
		Title:       "AICode Manager API",
		Description: "Claude CLI를 웹 플랫폼으로 관리하는 로컬 우선 시스템의 API",
		Version:     "1.0",
		Host:        "localhost:8080",
		BasePath:    "/api/v1",
	}
}