package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aicli/aicli-web/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func TestNew(t *testing.T) {
	// 서버 인스턴스 생성 테스트
	s := New()
	
	testutil.AssertNotNil(t, s)
	testutil.AssertNotNil(t, s.router)
}

func TestServer_Router(t *testing.T) {
	s := New()
	router := s.Router()
	
	testutil.AssertNotNil(t, router)
	
	// Gin 엔진 타입 확인
	_, ok := router.(*gin.Engine)
	if !ok {
		t.Error("Router()가 *gin.Engine 타입을 반환해야 함")
	}
}

func TestSetupRouter(t *testing.T) {
	tests := []struct {
		name    string
		env     string
		ginMode string
	}{
		{
			name:    "production 환경",
			env:     "production",
			ginMode: gin.ReleaseMode,
		},
		{
			name:    "development 환경",
			env:     "development",
			ginMode: gin.DebugMode,
		},
		{
			name:    "환경 설정 없음",
			env:     "",
			ginMode: gin.DebugMode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// viper 설정
			viper.Reset()
			viper.Set("env", tt.env)
			
			// 서버 생성
			s := &Server{}
			s.setupRouter()
			
			// Gin 모드 확인
			testutil.AssertEqual(t, tt.ginMode, gin.Mode())
			
			// 라우터가 생성되었는지 확인
			testutil.AssertNotNil(t, s.router)
			
			// 미들웨어가 설정되었는지 확인 (핸들러 개수로 간접 확인)
			routes := s.router.Routes()
			if len(routes) == 0 {
				t.Error("라우트가 설정되지 않음")
			}
		})
	}
}

func TestServerEndpoints(t *testing.T) {
	// 테스트용 서버 생성
	gin.SetMode(gin.TestMode)
	s := New()
	
	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		checkBody  func(t *testing.T, body string)
	}{
		{
			name:       "루트 엔드포인트",
			method:     "GET",
			path:       "/",
			wantStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				testutil.AssertContains(t, body, "AICode Manager API")
				testutil.AssertContains(t, body, "version")
				testutil.AssertContains(t, body, "running")
			},
		},
		{
			name:       "헬스체크 엔드포인트",
			method:     "GET",
			path:       "/health",
			wantStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				testutil.AssertContains(t, body, "status")
			},
		},
		{
			name:       "버전 엔드포인트",
			method:     "GET",
			path:       "/version",
			wantStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				testutil.AssertContains(t, body, "version")
				testutil.AssertContains(t, body, "go_version")
			},
		},
		{
			name:       "시스템 정보 엔드포인트",
			method:     "GET",
			path:       "/api/v1/system/info",
			wantStatus: http.StatusOK,
		},
		{
			name:       "워크스페이스 목록 엔드포인트",
			method:     "GET",
			path:       "/api/v1/workspaces",
			wantStatus: http.StatusOK,
		},
		{
			name:       "404 엔드포인트",
			method:     "GET",
			path:       "/api/v1/invalid",
			wantStatus: http.StatusNotFound,
			checkBody: func(t *testing.T, body string) {
				testutil.AssertContains(t, body, "Not Found")
				testutil.AssertContains(t, body, "요청한 엔드포인트를 찾을 수 없습니다")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 요청 생성
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			
			// 요청 실행
			s.router.ServeHTTP(w, req)
			
			// 상태 코드 확인
			testutil.AssertEqual(t, tt.wantStatus, w.Code)
			
			// 응답 본문 확인
			if tt.checkBody != nil {
				tt.checkBody(t, w.Body.String())
			}
		})
	}
}

func TestDebugRoutes(t *testing.T) {
	// 디버그 모드 설정
	gin.SetMode(gin.DebugMode)
	s := New()
	
	// 디버그 라우트 테스트
	req := httptest.NewRequest("GET", "/debug/routes", nil)
	w := httptest.NewRecorder()
	
	s.router.ServeHTTP(w, req)
	
	testutil.AssertEqual(t, http.StatusOK, w.Code)
	testutil.AssertContains(t, w.Body.String(), "routes")
	testutil.AssertContains(t, w.Body.String(), "count")
}

func TestMiddlewareOrder(t *testing.T) {
	// 미들웨어 순서가 중요하므로 테스트
	s := New()
	
	// 테스트 요청으로 미들웨어 실행 확인
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	
	s.router.ServeHTTP(w, req)
	
	// 응답 헤더로 미들웨어 실행 확인
	// RequestID 미들웨어 확인
	if w.Header().Get("X-Request-ID") == "" {
		t.Error("RequestID 미들웨어가 실행되지 않음")
	}
	
	// CORS 미들웨어 확인
	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("CORS 미들웨어가 실행되지 않음")
	}
	
	// Security 미들웨어 확인
	if w.Header().Get("X-Content-Type-Options") == "" {
		t.Error("Security 미들웨어가 실행되지 않음")
	}
}