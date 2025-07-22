package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aicli/aicli-web/internal/testutil"
	"github.com/gin-gonic/gin"
)

func TestSetupRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := &Server{}
	s.router = gin.New()
	s.setupRoutes()
	
	// 라우트가 설정되었는지 확인
	routes := s.router.Routes()
	if len(routes) == 0 {
		t.Fatal("라우트가 설정되지 않음")
	}
	
	// 주요 라우트 경로 확인
	expectedPaths := []string{
		"/",
		"/health",
		"/version",
		"/api/v1/system/info",
		"/api/v1/system/status",
		"/api/v1/workspaces",
		"/api/v1/tasks",
		"/api/v1/logs/workspaces/:id",
		"/api/v1/logs/tasks/:id",
		"/api/v1/config",
	}
	
	routeMap := make(map[string]bool)
	for _, route := range routes {
		routeMap[route.Path] = true
	}
	
	for _, path := range expectedPaths {
		if !routeMap[path] {
			t.Errorf("예상된 라우트가 없음: %s", path)
		}
	}
}

func TestAPIRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := New()
	
	tests := []struct {
		name         string
		method       string
		path         string
		wantStatus   int
		checkHeaders bool
		checkJSON    bool
	}{
		// 시스템 관련 엔드포인트
		{
			name:       "시스템 정보",
			method:     "GET",
			path:       "/api/v1/system/info",
			wantStatus: http.StatusOK,
			checkJSON:  true,
		},
		{
			name:       "시스템 상태",
			method:     "GET",
			path:       "/api/v1/system/status",
			wantStatus: http.StatusOK,
			checkJSON:  true,
		},
		// 워크스페이스 관련 엔드포인트
		{
			name:       "워크스페이스 목록",
			method:     "GET",
			path:       "/api/v1/workspaces",
			wantStatus: http.StatusOK,
			checkJSON:  true,
		},
		{
			name:       "워크스페이스 생성",
			method:     "POST",
			path:       "/api/v1/workspaces",
			wantStatus: http.StatusBadRequest, // 요청 본문 없이 보내므로 BadRequest 예상
			checkJSON:  true,
		},
		{
			name:       "워크스페이스 조회",
			method:     "GET",
			path:       "/api/v1/workspaces/test-id",
			wantStatus: http.StatusNotFound, // 존재하지 않는 ID
			checkJSON:  true,
		},
		// 태스크 관련 엔드포인트
		{
			name:       "태스크 목록",
			method:     "GET",
			path:       "/api/v1/tasks",
			wantStatus: http.StatusOK,
			checkJSON:  true,
		},
		// 로그 관련 엔드포인트
		{
			name:       "워크스페이스 로그",
			method:     "GET",
			path:       "/api/v1/logs/workspaces/test-id",
			wantStatus: http.StatusOK,
			checkJSON:  true,
		},
		{
			name:       "태스크 로그",
			method:     "GET",
			path:       "/api/v1/logs/tasks/test-id",
			wantStatus: http.StatusOK,
			checkJSON:  true,
		},
		// 설정 관련 엔드포인트
		{
			name:       "설정 조회",
			method:     "GET",
			path:       "/api/v1/config",
			wantStatus: http.StatusOK,
			checkJSON:  true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			
			s.router.ServeHTTP(w, req)
			
			// 상태 코드 확인
			testutil.AssertEqual(t, tt.wantStatus, w.Code)
			
			// JSON 응답 확인
			if tt.checkJSON {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				if err != nil {
					t.Errorf("JSON 파싱 실패: %v", err)
				}
			}
		})
	}
}

func TestNoRouteHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := New()
	
	// 존재하지 않는 경로들 테스트
	invalidPaths := []string{
		"/invalid",
		"/api/invalid",
		"/api/v2/workspaces",
		"/api/v1/invalid/endpoint",
	}
	
	for _, path := range invalidPaths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest("GET", path, nil)
			w := httptest.NewRecorder()
			
			s.router.ServeHTTP(w, req)
			
			// 404 상태 확인
			testutil.AssertEqual(t, http.StatusNotFound, w.Code)
			
			// 에러 메시지 확인
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			if err != nil {
				t.Fatalf("JSON 파싱 실패: %v", err)
			}
			
			testutil.AssertEqual(t, "Not Found", response["error"])
			testutil.AssertEqual(t, "요청한 엔드포인트를 찾을 수 없습니다", response["message"])
			testutil.AssertEqual(t, path, response["path"])
		})
	}
}

func TestDebugMode(t *testing.T) {
	tests := []struct {
		name         string
		mode         string
		expectDebug  bool
	}{
		{
			name:        "Debug 모드에서 디버그 라우트 활성화",
			mode:        gin.DebugMode,
			expectDebug: true,
		},
		{
			name:        "Release 모드에서 디버그 라우트 비활성화",
			mode:        gin.ReleaseMode,
			expectDebug: false,
		},
		{
			name:        "Test 모드에서 디버그 라우트 비활성화",
			mode:        gin.TestMode,
			expectDebug: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(tt.mode)
			s := New()
			
			req := httptest.NewRequest("GET", "/debug/routes", nil)
			w := httptest.NewRecorder()
			
			s.router.ServeHTTP(w, req)
			
			if tt.expectDebug {
				testutil.AssertEqual(t, http.StatusOK, w.Code)
				testutil.AssertContains(t, w.Body.String(), "routes")
			} else {
				testutil.AssertEqual(t, http.StatusNotFound, w.Code)
			}
		})
	}
}

func TestRouteGroups(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := New()
	
	// API v1 그룹 테스트
	v1Endpoints := []string{
		"/api/v1/system/info",
		"/api/v1/workspaces",
		"/api/v1/tasks",
		"/api/v1/logs/workspaces/test",
		"/api/v1/config",
	}
	
	for _, endpoint := range v1Endpoints {
		req := httptest.NewRequest("GET", endpoint, nil)
		w := httptest.NewRecorder()
		
		s.router.ServeHTTP(w, req)
		
		// 404가 아닌지 확인 (실제 핸들러가 없어도 라우트는 등록됨)
		if w.Code == http.StatusNotFound {
			t.Errorf("API v1 엔드포인트가 등록되지 않음: %s", endpoint)
		}
	}
}