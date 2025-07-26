package testutil

import (
	"encoding/json"
	"testing"
)

// TestData 테스트 데이터 구조체
type TestData struct {
	Projects []TestProject `json:"projects"`
	Users    []TestUser    `json:"users"`
	Configs  []TestConfig  `json:"configs"`
}

// TestProject 테스트용 프로젝트 데이터
type TestProject struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Path        string   `json:"path"`
	Status      string   `json:"status"`
	ClaudeAPIKey string  `json:"claude_api_key,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// TestUser 테스트용 사용자 데이터
type TestUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

// TestConfig 테스트용 설정 데이터
type TestConfig struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// LoadTestData 테스트 데이터 로드
func LoadTestData(t *testing.T) *TestData {
	t.Helper()
	
	return &TestData{
		Projects: []TestProject{
			{
				ID:     "proj-1",
				Name:   "test-project-1",
				Path:   "/workspace/test-project-1",
				Status: "active",
				Tags:   []string{"go", "web"},
			},
			{
				ID:     "proj-2",
				Name:   "test-project-2",
				Path:   "/workspace/test-project-2",
				Status: "inactive",
				Tags:   []string{"cli", "tool"},
			},
		},
		Users: []TestUser{
			{
				ID:       "user-1",
				Username: "testuser1",
				Email:    "test1@example.com",
				Role:     "admin",
			},
			{
				ID:       "user-2",
				Username: "testuser2",
				Email:    "test2@example.com",
				Role:     "user",
			},
		},
		Configs: []TestConfig{
			{
				Key:   "api.port",
				Value: "8080",
			},
			{
				Key:   "api.timeout",
				Value: "30s",
			},
		},
	}
}

// CreateTestDataFile 테스트 데이터 파일 생성
func CreateTestDataFile(t *testing.T, dir string, data interface{}) string {
	t.Helper()
	
	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatalf("테스트 데이터 마샬링 실패: %v", err)
	}
	
	return TempFile(t, dir, "testdata-*.json", string(content))
}

// GetTestProject 테스트용 프로젝트 생성
func GetTestProject(name string) TestProject {
	return TestProject{
		ID:     "test-" + name,
		Name:   name,
		Path:   "/workspace/" + name,
		Status: "active",
		Tags:   []string{"test"},
	}
}

// GetTestUser 테스트용 사용자 생성
func GetTestUser(username string) TestUser {
	return TestUser{
		ID:       "user-" + username,
		Username: username,
		Email:    username + "@test.com",
		Role:     "user",
	}
}

// GetTestConfig 테스트용 설정 생성
func GetTestConfig(key, value string) TestConfig {
	return TestConfig{
		Key:   key,
		Value: value,
	}
}

// SampleAPIResponse 샘플 API 응답 데이터
func SampleAPIResponse() map[string]interface{} {
	return map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"id":      "123",
			"message": "테스트 성공",
		},
		"timestamp": "2025-01-20T12:00:00Z",
	}
}

// SampleErrorResponse 샘플 에러 응답 데이터
func SampleErrorResponse(code int, message string) map[string]interface{} {
	return map[string]interface{}{
		"success": false,
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	}
}