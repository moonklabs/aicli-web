package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aicli/aicli-web/internal/server"
	"github.com/spf13/viper"
	// Swagger docs 자동 생성을 위한 임포트 (docs 패키지 생성 필요)
	// _ "github.com/aicli/aicli-web/docs"
)

// @title AICode Manager API
// @version 1.0
// @description Claude CLI를 웹 플랫폼으로 관리하는 로컬 우선 시스템의 API
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url https://github.com/aicli/aicli-web
// @contact.email support@aicli.dev

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description JWT 인증 토큰. "Bearer {token}" 형식으로 입력하세요.

func main() {
	// 설정 초기화
	initConfig()

	// 서버 생성
	srv := server.New()

	// 서버 설정
	port := viper.GetString("port")
	if port == "" {
		port = "8080"
	}

	httpServer := &http.Server{
		Addr:    ":" + port,
		Handler: srv.Router(),
	}

	// 고루틴에서 서버 시작
	go func() {
		log.Printf("🚀 AICode Manager API 서버가 포트 %s에서 시작됩니다", port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("서버 시작 실패: %v", err)
		}
	}()

	// 우아한 종료를 위한 시그널 대기
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("서버를 종료합니다...")

	// 30초 타임아웃으로 서버 종료
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatal("서버 강제 종료:", err)
	}

	log.Println("서버가 정상적으로 종료되었습니다")
}

// initConfig는 환경 변수 및 설정을 초기화합니다.
func initConfig() {
	// 환경 변수 자동 읽기
	viper.AutomaticEnv()
	viper.SetEnvPrefix("AICLI")

	// 기본값 설정
	viper.SetDefault("port", "8080")
	viper.SetDefault("env", "development")
	viper.SetDefault("log_level", "info")

	// 환경별 설정
	env := viper.GetString("env")
	fmt.Printf("환경: %s\n", env)

	if env == "development" {
		viper.SetDefault("log_level", "debug")
	}
}
