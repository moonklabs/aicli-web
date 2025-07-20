---
task_id: T02_S02
sprint_sequence_id: S02
status: open
complexity: Medium
estimated_hours: 6
assigned_to: TBD
created_date: 2025-07-20
last_updated: 2025-07-20T04:00:00Z
---

# Task: 테스트 프레임워크 및 기본 테스트 설정

## Description
Go 프로젝트의 테스트 프레임워크를 설정하고 기본 테스트 코드를 작성합니다. 단위 테스트, 통합 테스트, 커버리지 리포트 생성을 포함한 포괄적인 테스트 환경을 구축합니다.

## Goal / Objectives
- Go 표준 테스트 패키지 + Testify 프레임워크 설정
- 단위 테스트 및 통합 테스트 구조 설계
- 테스트 커버리지 리포트 생성 시스템
- Makefile에 테스트 관련 명령어 통합
- CI/CD 파이프라인 테스트 자동화 준비

## Acceptance Criteria
- [ ] testify 의존성 추가 및 설정
- [ ] 단위 테스트 템플릿 및 예제 작성
- [ ] 통합 테스트 구조 설계
- [ ] 테스트 데이터 및 픽스처 관리 시스템
- [ ] `make test` 명령어로 모든 테스트 실행
- [ ] `make test-coverage` 명령어로 커버리지 리포트 생성
- [ ] 테스트 결과 HTML 리포트 생성
- [ ] 최소 70% 코드 커버리지 달성

## Subtasks
- [ ] testify/assert, testify/mock 의존성 추가
- [ ] 테스트 디렉토리 구조 설계 및 생성
- [ ] 기존 코드에 대한 단위 테스트 작성
- [ ] 모킹 시스템 설정 (API 호출, 데이터베이스 등)
- [ ] 테스트 헬퍼 함수 및 유틸리티 작성
- [ ] 통합 테스트 프레임워크 설정
- [ ] Makefile 테스트 타겟 추가
- [ ] 테스트 환경 설정 (.env.test 등)
- [ ] 벤치마크 테스트 기본 구조

## Technical Guide

### 테스트 프레임워크 선택

#### 기본 스택
1. **Go 표준 testing 패키지**: 기본 테스트 러너
2. **testify/assert**: 풍부한 assertion 함수
3. **testify/mock**: 모킹 및 스텁 기능
4. **testify/suite**: 테스트 스위트 구성

#### 의존성 추가
```bash
go get github.com/stretchr/testify/assert
go get github.com/stretchr/testify/mock
go get github.com/stretchr/testify/suite
go get github.com/DATA-DOG/go-sqlmock  # 데이터베이스 모킹
```

### 테스트 디렉토리 구조

#### 권장 구조
```
aicli-web/
├── internal/
│   ├── cli/
│   │   ├── commands/
│   │   │   ├── workspace.go
│   │   │   └── workspace_test.go    # 단위 테스트
│   │   └── cli_test.go
│   ├── server/
│   │   ├── handlers/
│   │   │   ├── health.go
│   │   │   └── health_test.go
│   │   └── server_test.go
│   └── models/
│       ├── workspace.go
│       └── workspace_test.go
├── test/
│   ├── integration/                 # 통합 테스트
│   │   ├── api_test.go
│   │   └── cli_test.go
│   ├── fixtures/                    # 테스트 데이터
│   │   ├── config.yaml
│   │   └── sample_projects/
│   ├── mocks/                       # 생성된 모킹 코드
│   │   ├── claude_client.go
│   │   └── docker_client.go
│   └── testutils/                   # 테스트 유틸리티
│       ├── helpers.go
│       └── assertions.go
```

### 단위 테스트 예제

#### 기본 테스트 구조
```go
package commands

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

func TestWorkspaceList(t *testing.T) {
    tests := []struct {
        name     string
        input    []string
        expected []string
        wantErr  bool
    }{
        {
            name:     "성공적인 워크스페이스 목록 조회",
            input:    []string{},
            expected: []string{"project1", "project2"},
            wantErr:  false,
        },
        {
            name:     "빈 워크스페이스 목록",
            input:    []string{},
            expected: []string{},
            wantErr:  false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Given
            cmd := NewWorkspaceCommand()
            
            // When
            result, err := cmd.List(tt.input)
            
            // Then
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.expected, result)
            }
        })
    }
}
```

#### 모킹 예제
```go
// mocks/claude_client.go
type MockClaudeClient struct {
    mock.Mock
}

func (m *MockClaudeClient) Execute(cmd string) (string, error) {
    args := m.Called(cmd)
    return args.String(0), args.Error(1)
}

// 테스트에서 사용
func TestClaudeIntegration(t *testing.T) {
    // Given
    mockClient := new(MockClaudeClient)
    mockClient.On("Execute", "workspace list").Return("project1\nproject2", nil)
    
    service := NewWorkspaceService(mockClient)
    
    // When
    workspaces, err := service.GetWorkspaces()
    
    // Then
    assert.NoError(t, err)
    assert.Len(t, workspaces, 2)
    mockClient.AssertExpectations(t)
}
```

### 통합 테스트 구조

#### API 통합 테스트
```go
package integration

import (
    "testing"
    "net/http/httptest"
    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/suite"
)

type APITestSuite struct {
    suite.Suite
    router *gin.Engine
    server *httptest.Server
}

func (suite *APITestSuite) SetupSuite() {
    gin.SetMode(gin.TestMode)
    suite.router = setupTestRouter()
    suite.server = httptest.NewServer(suite.router)
}

func (suite *APITestSuite) TearDownSuite() {
    suite.server.Close()
}

func (suite *APITestSuite) TestHealthEndpoint() {
    resp, err := http.Get(suite.server.URL + "/health")
    suite.NoError(err)
    suite.Equal(200, resp.StatusCode)
}

func TestAPITestSuite(t *testing.T) {
    suite.Run(t, new(APITestSuite))
}
```

### Makefile 테스트 타겟

#### 테스트 관련 명령어
```makefile
# 모든 테스트 실행
.PHONY: test
test:
	@printf "${BLUE}Running tests...${NC}\n"
	go test -v -race ./...

# 단위 테스트만 실행
.PHONY: test-unit
test-unit:
	@printf "${BLUE}Running unit tests...${NC}\n"
	go test -v -short ./...

# 통합 테스트 실행
.PHONY: test-integration
test-integration:
	@printf "${BLUE}Running integration tests...${NC}\n"
	go test -v -run Integration ./test/integration/...

# 테스트 커버리지
.PHONY: test-coverage
test-coverage:
	@printf "${BLUE}Generating test coverage...${NC}\n"
	@mkdir -p reports
	go test -coverprofile=reports/coverage.out ./...
	go tool cover -html=reports/coverage.out -o reports/coverage.html
	go tool cover -func=reports/coverage.out

# 벤치마크 테스트
.PHONY: test-bench
test-bench:
	@printf "${BLUE}Running benchmark tests...${NC}\n"
	go test -bench=. -benchmem ./...

# 테스트 캐시 정리
.PHONY: test-clean
test-clean:
	@printf "${BLUE}Cleaning test cache...${NC}\n"
	go clean -testcache
```

### 테스트 환경 설정

#### .env.test 파일
```env
# 테스트 환경 설정
AICLI_ENV=test
AICLI_PORT=0  # 랜덤 포트 사용
AICLI_LOG_LEVEL=warn
AICLI_DB_PATH=:memory:  # 인메모리 데이터베이스

# 테스트용 Claude API (모킹)
CLAUDE_API_MOCK=true
CLAUDE_API_KEY=test_key

# Docker 테스트 설정
DOCKER_HOST=unix:///var/run/docker.sock
DOCKER_TEST_NETWORK=aicli_test
```

### 구현 노트
- 테스트는 독립적이고 반복 가능해야 함
- 외부 의존성은 모킹으로 처리
- 테스트 데이터는 fixtures 디렉토리에서 관리
- 커버리지 70% 이상 목표 설정
- 실행 속도를 위해 단위/통합 테스트 분리

## Output Log

### [날짜 및 시간은 태스크 진행 시 업데이트]

<!-- 작업 진행 로그를 여기에 기록 -->

**상태**: 📋 대기 중