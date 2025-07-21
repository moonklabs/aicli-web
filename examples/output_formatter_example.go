package main

import (
	"fmt"
	"log"
	"os"
	"time"
	
	"aicli-web/internal/cli/output"
)

// 예시 구조체 정의
type Workspace struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	ProjectPath string    `json:"project_path"`
}

type Task struct {
	ID         string `json:"id"`
	Workspace  string `json:"workspace"`
	Command    string `json:"command"`
	Status     string `json:"status"`
	StartedAt  string `json:"started_at"`
	Duration   string `json:"duration"`
}

func main() {
	// 예시 1: 워크스페이스 목록을 테이블 형식으로 출력
	fmt.Println("=== 1. Table Format (Default) ===")
	workspaceListExample()
	
	// 예시 2: 동일한 데이터를 JSON 형식으로 출력
	fmt.Println("\n=== 2. JSON Format ===")
	workspaceListJSONExample()
	
	// 예시 3: YAML 형식으로 출력
	fmt.Println("\n=== 3. YAML Format ===")
	workspaceListYAMLExample()
	
	// 예시 4: 색상 비활성화 예시
	fmt.Println("\n=== 4. No Color Output ===")
	noColorExample()
	
	// 예시 5: 구조체 슬라이스 출력
	fmt.Println("\n=== 5. Struct Slice Output ===")
	structSliceExample()
	
	// 예시 6: 단일 객체 출력
	fmt.Println("\n=== 6. Single Object Output ===")
	singleObjectExample()
}

func workspaceListExample() {
	// 테이블 포맷터 생성
	formatter := output.NewFormatterManager(output.FormatTable)
	
	// 데이터 준비
	workspaces := []map[string]interface{}{
		{
			"name":       "project-alpha",
			"status":     "active",
			"created_at": "2024-01-15T10:30:00Z",
			"path":       "/home/user/projects/alpha",
		},
		{
			"name":       "project-beta",
			"status":     "inactive",
			"created_at": "2024-01-10T08:15:00Z",
			"path":       "/home/user/projects/beta",
		},
		{
			"name":       "project-gamma",
			"status":     "active",
			"created_at": "2024-01-20T14:45:00Z",
			"path":       "/home/user/projects/gamma",
		},
	}
	
	// 헤더 설정
	formatter.SetHeaders([]string{"name", "status", "created_at", "path"})
	
	// 출력
	if err := formatter.Print(workspaces); err != nil {
		log.Fatal(err)
	}
}

func workspaceListJSONExample() {
	// JSON 포맷터 생성
	formatter := output.NewFormatterManager(output.FormatJSON)
	
	// 동일한 데이터
	workspaces := []map[string]interface{}{
		{
			"name":       "project-alpha",
			"status":     "active",
			"created_at": "2024-01-15T10:30:00Z",
			"path":       "/home/user/projects/alpha",
		},
		{
			"name":       "project-beta",
			"status":     "inactive",
			"created_at": "2024-01-10T08:15:00Z",
			"path":       "/home/user/projects/beta",
		},
	}
	
	// 출력
	if err := formatter.Print(workspaces); err != nil {
		log.Fatal(err)
	}
}

func workspaceListYAMLExample() {
	// YAML 포맷터 생성
	formatter := output.NewFormatterManager(output.FormatYAML)
	
	// 동일한 데이터
	workspaces := []map[string]interface{}{
		{
			"name":       "project-alpha",
			"status":     "active",
			"created_at": "2024-01-15T10:30:00Z",
			"path":       "/home/user/projects/alpha",
		},
	}
	
	// 출력
	if err := formatter.Print(workspaces); err != nil {
		log.Fatal(err)
	}
}

func noColorExample() {
	// 색상 비활성화를 위해 환경 변수 설정
	os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")
	
	// 테이블 포맷터 생성
	formatter := output.NewFormatterManager(output.FormatTable)
	
	// 데이터
	data := map[string]interface{}{
		"name":   "test-workspace",
		"status": "active",
		"tasks":  5,
	}
	
	// 출력
	if err := formatter.Print(data); err != nil {
		log.Fatal(err)
	}
}

func structSliceExample() {
	// 구조체 슬라이스 데이터
	workspaces := []Workspace{
		{
			ID:          "ws-001",
			Name:        "frontend-app",
			Status:      "active",
			CreatedAt:   time.Now().Add(-24 * time.Hour),
			ProjectPath: "/projects/frontend",
		},
		{
			ID:          "ws-002",
			Name:        "backend-api",
			Status:      "inactive",
			CreatedAt:   time.Now().Add(-48 * time.Hour),
			ProjectPath: "/projects/backend",
		},
	}
	
	// 테이블 포맷터로 출력
	formatter := output.NewFormatterManager(output.FormatTable)
	if err := formatter.Print(workspaces); err != nil {
		log.Fatal(err)
	}
}

func singleObjectExample() {
	// 단일 객체 데이터
	config := map[string]interface{}{
		"api_key":     "sk-ant-xxxxx",
		"model":       "claude-3-opus",
		"temperature": 0.7,
		"timeout":     30,
		"debug_mode":  true,
	}
	
	// 테이블 포맷터로 출력
	formatter := output.NewFormatterManager(output.FormatTable)
	if err := formatter.Print(config); err != nil {
		log.Fatal(err)
	}
	
	fmt.Println("\n--- Same data in JSON format ---")
	
	// JSON 포맷터로 동일한 데이터 출력
	jsonFormatter := output.NewFormatterManager(output.FormatJSON)
	if err := jsonFormatter.Print(config); err != nil {
		log.Fatal(err)
	}
}