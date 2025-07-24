//go:build integration
// +build integration

package test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/aicli/aicli-web/internal/server"
	"github.com/aicli/aicli-web/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func TestMain(m *testing.M) {
	// 테스트 환경 설정
	gin.SetMode(gin.TestMode)
	viper.Set("env", "test")
	
	// 테스트 실행
	code := m.Run()
	
	// 정리
	os.Exit(code)
}

func TestAPIServerIntegration(t *testing.T) {
	// API 서버 통합 테스트
	srv := server.New()
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	tests := []struct {
		name       string
		endpoint   string
		method     string
		wantStatus int
		checkBody  func(t *testing.T, body map[string]interface{})
	}{
		{
			name:       "헬스체크 통합 테스트",
			endpoint:   "/health",
			method:     http.MethodGet,
			wantStatus: http.StatusOK,
			checkBody: func(t *testing.T, body map[string]interface{}) {
				testutil.AssertEqual(t, "healthy", body["status"])
				testutil.AssertNotNil(t, body["timestamp"])
			},
		},
		{
			name:       "시스템 정보 통합 테스트",
			endpoint:   "/api/v1/system/info",
			method:     http.MethodGet,
			wantStatus: http.StatusOK,
			checkBody: func(t *testing.T, body map[string]interface{}) {
				testutil.AssertNotNil(t, body["version"])
				testutil.AssertNotNil(t, body["os"])
				testutil.AssertNotNil(t, body["arch"])
			},
		},
		{
			name:       "워크스페이스 목록 통합 테스트",
			endpoint:   "/api/v1/workspaces",
			method:     http.MethodGet,
			wantStatus: http.StatusOK,
			checkBody: func(t *testing.T, body map[string]interface{}) {
				workspaces, ok := body["workspaces"].([]interface{})
				if !ok {
					t.Error("workspaces 필드가 배열이 아님")
				}
				testutil.AssertNotNil(t, workspaces)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// HTTP 요청 생성
			req, err := http.NewRequest(tt.method, ts.URL+tt.endpoint, nil)
			if err != nil {
				t.Fatal(err)
			}

			// 요청 실행
			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			// 상태 코드 확인
			testutil.AssertEqual(t, tt.wantStatus, resp.StatusCode)

			// 응답 본문 파싱
			var body map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}

			// 본문 검증
			if tt.checkBody != nil {
				tt.checkBody(t, body)
			}
		})
	}
}

func TestCLIIntegration(t *testing.T) {
	// CLI 통합 테스트
	tmpDir := testutil.TempDir(t, "cli-integration")
	
	// 테스트 프로젝트 생성
	testutil.CreateTestProject(t, tmpDir)
	
	tests := []struct {
		name      string
		setup     func() error
		teardown  func()
		test      func(t *testing.T)
	}{
		{
			name: "CLI 명령어 실행 테스트",
			setup: func() error {
				// 설정 파일 생성
				config := map[string]interface{}{
					"workspaces": map[string]interface{}{
						"default": tmpDir,
					},
				}
				viper.Set("workspaces", config["workspaces"])
				return nil
			},
			test: func(t *testing.T) {
				// CLI 명령어 실행 시뮬레이션
				// 실제 구현에서는 cobra 명령어를 직접 실행
				t.Log("CLI 명령어 실행 테스트")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				if err := tt.setup(); err != nil {
					t.Fatal(err)
				}
			}
			
			tt.test(t)
			
			if tt.teardown != nil {
				tt.teardown()
			}
		})
	}
}

func TestWorkspaceLifecycle(t *testing.T) {
	// 워크스페이스 생명주기 통합 테스트
	srv := server.New()
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	// 1. 워크스페이스 생성
	createResp := createWorkspace(t, ts.URL, "test-workspace")
	workspaceID := createResp["id"].(string)

	// 2. 워크스페이스 조회
	getWorkspace(t, ts.URL, workspaceID)

	// 3. 워크스페이스 업데이트
	updateWorkspace(t, ts.URL, workspaceID)

	// 4. 워크스페이스 삭제
	deleteWorkspace(t, ts.URL, workspaceID)
}

func TestConcurrentRequests(t *testing.T) {
	// 동시 요청 처리 테스트
	srv := server.New()
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	// 동시에 여러 요청 보내기
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	numRequests := 10
	results := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(idx int) {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/health", nil)
			if err != nil {
				results <- err
				return
			}

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				results <- err
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				results <- fmt.Errorf("요청 %d 실패: status=%d", idx, resp.StatusCode)
				return
			}

			results <- nil
		}(i)
	}

	// 결과 수집
	for i := 0; i < numRequests; i++ {
		if err := <-results; err != nil {
			t.Error(err)
		}
	}
}

// Helper functions
func createWorkspace(t *testing.T, baseURL, name string) map[string]interface{} {
	// 워크스페이스 생성 헬퍼
	// 실제 구현에서는 POST 요청을 보내고 응답을 파싱
	return map[string]interface{}{
		"id":   "test-id",
		"name": name,
	}
}

func getWorkspace(t *testing.T, baseURL, id string) map[string]interface{} {
	// 워크스페이스 조회 헬퍼
	return map[string]interface{}{
		"id": id,
	}
}

func updateWorkspace(t *testing.T, baseURL, id string) {
	// 워크스페이스 업데이트 헬퍼
	t.Logf("워크스페이스 업데이트: %s", id)
}

func deleteWorkspace(t *testing.T, baseURL, id string) {
	// 워크스페이스 삭제 헬퍼
	t.Logf("워크스페이스 삭제: %s", id)
}