# 통합 테스트 가이드

AICode Manager 멀티 에이전트 플랫폼의 통합 테스트 작성 및 실행 방법을 설명합니다.

## 🧪 테스트 개요

### 테스트 구조 (T07_S01 구현 결과)

우리의 통합 테스트 시스템은 다음과 같은 구조로 구성되어 있습니다:

```
test/integration/
├── agent_integration_test_suite.go    # 에이전트 생명주기 E2E 테스트
├── git_integration_test.go            # Git worktree 통합 테스트
├── api_integration_test.go            # API 엔드포인트 통합 테스트
├── performance_load_test.go           # 성능 및 부하 테스트
├── common/                            # 공통 테스트 유틸리티
│   ├── test_infrastructure.go         # 테스트 인프라 설정
│   ├── test_runners.go                # 테스트 실행기
│   └── test_reporting.go              # 테스트 보고서 생성
└── fixtures/                          # 테스트 데이터 및 Mock
    ├── docker/                        # Docker 테스트 이미지
    ├── git/                           # Git 테스트 저장소
    └── api/                           # API 테스트 페이로드
```

### 테스트 카테고리

#### 1. 단위 테스트 (Unit Tests)
- 개별 함수/메서드 테스트
- Mock을 사용한 의존성 격리
- 빠른 실행 시간

#### 2. 통합 테스트 (Integration Tests)
- 컴포넌트 간 상호작용 테스트
- 실제 Docker 및 Git 사용
- 실제 환경과 유사한 조건

#### 3. E2E 테스트 (End-to-End Tests)
- 전체 워크플로우 테스트
- 사용자 시나리오 기반
- 프로덕션 환경 시뮬레이션

#### 4. 성능 테스트 (Performance Tests)
- 부하 테스트 및 스트레스 테스트
- 성능 지표 검증
- T06_S01 최적화 목표 달성 확인

## 🚀 테스트 환경 설정

### 1. 테스트 의존성 설치

```bash
# Go 테스트 도구
go install github.com/onsi/ginkgo/v2/ginkgo@latest
go install github.com/onsi/gomega/...@latest

# Docker 테스트 환경
docker network create aicli-test-network || true

# 테스트용 Git 저장소 준비
mkdir -p test/fixtures/git
cd test/fixtures/git
git init --bare test-repo.git
```

### 2. 테스트 환경 변수

```bash
# .env.test 파일 생성
cat << EOF > .env.test
# 테스트 환경 설정
AICLI_TEST_MODE=true
AICLI_TEST_DB_PATH=./test/data/test.db
AICLI_TEST_WORKSPACE_ROOT=./test/workspaces
AICLI_TEST_DOCKER_NETWORK=aicli-test-network

# Docker 테스트 설정
DOCKER_API_VERSION=1.41
DOCKER_TEST_TIMEOUT=300s

# Git 테스트 설정
GIT_TEST_REPO_PATH=./test/fixtures/git/test-repo.git
GIT_TEST_CLEANUP_ENABLED=true

# 성능 테스트 설정
PERF_TEST_CONCURRENT_AGENTS=100
PERF_TEST_TARGET_CREATION_TIME=5000
PERF_TEST_DURATION=300s
EOF
```

### 3. 테스트 데이터베이스 설정

```bash
# 테스트용 데이터베이스 스키마 생성
mkdir -p test/data
sqlite3 test/data/test.db < scripts/schema.sql
```

## 🔧 테스트 작성 가이드

### 1. 기본 테스트 구조

#### 테스트 스위트 기본 형태
```go
package integration

import (
    "testing"
    
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
    
    "github.com/aicli/aicli-web/internal/claude"
    "github.com/aicli/aicli-web/internal/docker"
    "github.com/aicli/aicli-web/internal/models"
)

func TestIntegration(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "Integration Test Suite")
}

var _ = Describe("Agent Integration Tests", func() {
    var (
        testSuite *AgentIntegrationTestSuite
        agent     *models.Agent
    )
    
    BeforeEach(func() {
        testSuite = NewAgentIntegrationTestSuite()
        Expect(testSuite.Setup()).To(Succeed())
    })
    
    AfterEach(func() {
        if agent != nil {
            testSuite.CleanupAgent(agent.ID)
        }
        testSuite.Teardown()
    })
    
    Context("Agent Lifecycle", func() {
        It("should create, start, and stop an agent successfully", func() {
            // 테스트 구현
        })
    })
})
```

### 2. 에이전트 생명주기 테스트

#### 기본 생명주기 테스트
```go
var _ = Describe("Agent Lifecycle Tests", func() {
    It("should complete full agent lifecycle", func() {
        By("Creating a new agent")
        createReq := &models.CreateAgentRequest{
            Name:        "test-agent",
            ProjectID:   "test-project",
            AgentType:   models.AgentTypeStandard,
            Description: "Integration test agent",
        }
        
        agent, err := testSuite.agentService.CreateAgent(createReq)
        Expect(err).ToNot(HaveOccurred())
        Expect(agent.ID).ToNot(BeEmpty())
        Expect(agent.Status).To(Equal(models.AgentStatusCreated))
        
        By("Starting the agent")
        err = testSuite.agentService.StartAgent(agent.ID)
        Expect(err).ToNot(HaveOccurred())
        
        // 에이전트가 실행될 때까지 대기
        Eventually(func() models.AgentStatus {
            status, _ := testSuite.agentService.GetAgentStatus(agent.ID)
            return status.Status
        }, "30s", "1s").Should(Equal(models.AgentStatusRunning))
        
        By("Verifying agent is healthy")
        status, err := testSuite.agentService.GetAgentStatus(agent.ID)
        Expect(err).ToNot(HaveOccurred())
        Expect(status.HealthStatus).To(Equal("healthy"))
        Expect(status.ContainerStatus).To(Equal("running"))
        
        By("Stopping the agent")
        err = testSuite.agentService.StopAgent(agent.ID)
        Expect(err).ToNot(HaveOccurred())
        
        Eventually(func() models.AgentStatus {
            status, _ := testSuite.agentService.GetAgentStatus(agent.ID)
            return status.Status
        }, "30s", "1s").Should(Equal(models.AgentStatusStopped))
        
        By("Deleting the agent")
        err = testSuite.agentService.DeleteAgent(agent.ID)
        Expect(err).ToNot(HaveOccurred())
    })
})
```

#### 동시성 테스트
```go
var _ = Describe("Concurrent Agent Operations", func() {
    It("should handle multiple concurrent agent operations", func() {
        const numAgents = 10
        agentIDs := make([]string, numAgents)
        
        By("Creating multiple agents concurrently")
        createWG := sync.WaitGroup{}
        createWG.Add(numAgents)
        
        for i := 0; i < numAgents; i++ {
            go func(index int) {
                defer createWG.Done()
                defer GinkgoRecover()
                
                req := &models.CreateAgentRequest{
                    Name:      fmt.Sprintf("concurrent-agent-%d", index),
                    ProjectID: "concurrent-test",
                    AgentType: models.AgentTypeStandard,
                }
                
                agent, err := testSuite.agentService.CreateAgent(req)
                Expect(err).ToNot(HaveOccurred())
                agentIDs[index] = agent.ID
            }(i)
        }
        
        createWG.Wait()
        
        By("Starting all agents concurrently")
        startWG := sync.WaitGroup{}
        startWG.Add(numAgents)
        
        for _, agentID := range agentIDs {
            go func(id string) {
                defer startWG.Done()
                defer GinkgoRecover()
                
                err := testSuite.agentService.StartAgent(id)
                Expect(err).ToNot(HaveOccurred())
            }(agentID)
        }
        
        startWG.Wait()
        
        By("Verifying all agents are running")
        Eventually(func() int {
            runningCount := 0
            for _, agentID := range agentIDs {
                if status, err := testSuite.agentService.GetAgentStatus(agentID); err == nil {
                    if status.Status == models.AgentStatusRunning {
                        runningCount++
                    }
                }
            }
            return runningCount
        }, "60s", "2s").Should(Equal(numAgents))
    })
})
```

### 3. API 통합 테스트

#### HTTP API 테스트
```go
var _ = Describe("API Integration Tests", func() {
    var (
        server *httptest.Server
        client *http.Client
    )
    
    BeforeEach(func() {
        // 테스트 서버 시작
        server = testSuite.StartTestServer()
        client = &http.Client{Timeout: 30 * time.Second}
    })
    
    AfterEach(func() {
        server.Close()
    })
    
    Context("Agent CRUD Operations", func() {
        It("should perform complete CRUD operations via API", func() {
            By("Creating an agent via POST /api/v1/agents")
            createPayload := map[string]interface{}{
                "name":        "api-test-agent",
                "project_id":  "api-test",
                "agent_type":  "standard",
                "description": "API integration test",
            }
            
            body, _ := json.Marshal(createPayload)
            resp, err := client.Post(
                server.URL+"/api/v1/agents",
                "application/json",
                bytes.NewBuffer(body),
            )
            Expect(err).ToNot(HaveOccurred())
            Expect(resp.StatusCode).To(Equal(201))
            
            var agent models.Agent
            err = json.NewDecoder(resp.Body).Decode(&agent)
            Expect(err).ToNot(HaveOccurred())
            Expect(agent.Name).To(Equal("api-test-agent"))
            
            By("Retrieving the agent via GET /api/v1/agents/{id}")
            resp, err = client.Get(server.URL + "/api/v1/agents/" + agent.ID)
            Expect(err).ToNot(HaveOccurred())
            Expect(resp.StatusCode).To(Equal(200))
            
            By("Starting the agent via POST /api/v1/agents/{id}/start")
            resp, err = client.Post(
                server.URL+"/api/v1/agents/"+agent.ID+"/start",
                "application/json",
                nil,
            )
            Expect(err).ToNot(HaveOccurred())
            Expect(resp.StatusCode).To(Equal(200))
            
            By("Checking agent status via GET /api/v1/agents/{id}/status")
            Eventually(func() int {
                resp, err := client.Get(server.URL + "/api/v1/agents/" + agent.ID + "/status")
                if err != nil {
                    return 0
                }
                return resp.StatusCode
            }, "30s", "1s").Should(Equal(200))
        })
    })
})
```

#### WebSocket 테스트
```go
var _ = Describe("WebSocket Integration Tests", func() {
    It("should stream agent logs via WebSocket", func() {
        // 에이전트 생성 및 시작
        agent := testSuite.CreateAndStartAgent("ws-test-agent")
        
        By("Connecting to WebSocket endpoint")
        wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + 
                "/api/v1/agents/" + agent.ID + "/logs/stream"
        
        conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
        Expect(err).ToNot(HaveOccurred())
        defer conn.Close()
        
        By("Receiving log messages")
        messageReceived := make(chan bool, 1)
        go func() {
            defer GinkgoRecover()
            for {
                _, message, err := conn.ReadMessage()
                if err != nil {
                    return
                }
                
                var logEntry models.LogEntry
                if json.Unmarshal(message, &logEntry) == nil {
                    if logEntry.AgentID == agent.ID {
                        messageReceived <- true
                        return
                    }
                }
            }
        }()
        
        // 로그가 수신될 때까지 대기
        Eventually(messageReceived, "30s").Should(Receive())
    })
})
```

### 4. Git 통합 테스트

#### Git Worktree 테스트
```go
var _ = Describe("Git Integration Tests", func() {
    It("should manage Git worktrees correctly", func() {
        By("Creating a Git worktree for agent")
        repoURL := testSuite.GetTestGitRepo()
        branch := "main"
        
        worktreePath, err := testSuite.gitManager.CreateWorktree(repoURL, branch)
        Expect(err).ToNot(HaveOccurred())
        Expect(worktreePath).ToNot(BeEmpty())
        
        By("Verifying worktree exists and is on correct branch")
        Expect(worktreePath).To(BeADirectory())
        
        currentBranch, err := testSuite.gitManager.GetCurrentBranch(worktreePath)
        Expect(err).ToNot(HaveOccurred())
        Expect(currentBranch).To(Equal(branch))
        
        By("Switching to a different branch")
        newBranch := "feature/test-branch"
        err = testSuite.gitManager.CreateAndSwitchBranch(worktreePath, newBranch)
        Expect(err).ToNot(HaveOccurred())
        
        currentBranch, err = testSuite.gitManager.GetCurrentBranch(worktreePath)
        Expect(err).ToNot(HaveOccurred())
        Expect(currentBranch).To(Equal(newBranch))
        
        By("Cleaning up worktree")
        err = testSuite.gitManager.RemoveWorktree(worktreePath)
        Expect(err).ToNot(HaveOccurred())
        Expect(worktreePath).ToNot(BeADirectory())
    })
})
```

### 5. 성능 테스트

#### 부하 테스트
```go
var _ = Describe("Performance Load Tests", func() {
    It("should handle 100+ concurrent agents (T06_S01 requirement)", func() {
        const targetAgents = 100
        const creationTimeLimit = 5 * time.Second
        
        By("Creating 100 agents concurrently")
        startTime := time.Now()
        agentIDs := make([]string, targetAgents)
        
        var wg sync.WaitGroup
        var mu sync.Mutex
        var creationTimes []time.Duration
        
        for i := 0; i < targetAgents; i++ {
            wg.Add(1)
            go func(index int) {
                defer wg.Done()
                defer GinkgoRecover()
                
                agentStartTime := time.Now()
                
                req := &models.CreateAgentRequest{
                    Name:      fmt.Sprintf("load-test-agent-%d", index),
                    ProjectID: "load-test",
                    AgentType: models.AgentTypeStandard,
                }
                
                agent, err := testSuite.agentService.CreateAgent(req)
                Expect(err).ToNot(HaveOccurred())
                
                err = testSuite.agentService.StartAgent(agent.ID)
                Expect(err).ToNot(HaveOccurred())
                
                creationTime := time.Since(agentStartTime)
                
                mu.Lock()
                agentIDs[index] = agent.ID
                creationTimes = append(creationTimes, creationTime)
                mu.Unlock()
            }(i)
        }
        
        wg.Wait()
        totalTime := time.Since(startTime)
        
        By("Verifying performance requirements")
        // P95 생성 시간이 5초 이내인지 확인
        sort.Slice(creationTimes, func(i, j int) bool {
            return creationTimes[i] < creationTimes[j]
        })
        p95Index := int(float64(len(creationTimes)) * 0.95)
        p95Time := creationTimes[p95Index]
        
        Expect(p95Time).To(BeNumerically("<", creationTimeLimit))
        
        By("Verifying all agents are running")
        Eventually(func() int {
            runningCount := 0
            for _, agentID := range agentIDs {
                if status, err := testSuite.agentService.GetAgentStatus(agentID); err == nil {
                    if status.Status == models.AgentStatusRunning {
                        runningCount++
                    }
                }
            }
            return runningCount
        }, "120s", "2s").Should(Equal(targetAgents))
        
        fmt.Printf("Performance Results:\n")
        fmt.Printf("- Total agents created: %d\n", targetAgents)
        fmt.Printf("- Total time: %v\n", totalTime)
        fmt.Printf("- Average creation time: %v\n", totalTime/time.Duration(targetAgents))
        fmt.Printf("- P95 creation time: %v\n", p95Time)
    })
})
```

## 🏃‍♂️ 테스트 실행

### 1. 개별 테스트 실행

```bash
# 모든 통합 테스트 실행
go test -v ./test/integration/...

# 특정 테스트 스위트 실행
go test -v ./test/integration/ -run TestAgentIntegrationTestSuite

# Ginkgo를 사용한 실행 (더 자세한 출력)
ginkgo -v ./test/integration/

# 특정 테스트만 실행
ginkgo -v ./test/integration/ --focus="Agent Lifecycle"

# 성능 테스트만 실행
ginkgo -v ./test/integration/ --focus="Performance"
```

### 2. 테스트 환경별 실행

```bash
# 개발 환경에서 실행
TEST_ENV=development go test -v ./test/integration/...

# CI 환경에서 실행
TEST_ENV=ci go test -v ./test/integration/... -timeout=30m

# 성능 테스트만 실행 (긴 시간 소요)
go test -v ./test/integration/ -run TestPerformance -timeout=60m
```

### 3. 커버리지와 함께 실행

```bash
# 테스트 커버리지 측정
go test -v ./test/integration/... -coverprofile=coverage.out

# 커버리지 리포트 생성
go tool cover -html=coverage.out -o coverage.html

# 커버리지 요약 확인
go tool cover -func=coverage.out
```

## 📊 테스트 보고서

### 1. 자동 보고서 생성

#### JUnit XML 보고서
```bash
# Ginkgo JUnit 리포터 사용
ginkgo -v ./test/integration/ --junit-report=test-results.xml
```

#### HTML 보고서
```bash
# 테스트 결과를 HTML로 출력
ginkgo -v ./test/integration/ --json-report=test-results.json
go run scripts/generate-html-report.go test-results.json > test-report.html
```

### 2. 성능 벤치마크 보고서

```go
// 성능 테스트 결과 구조체
type PerformanceReport struct {
    Timestamp           time.Time         `json:"timestamp"`
    TotalAgents         int               `json:"total_agents"`
    ConcurrentAgents    int               `json:"concurrent_agents"`
    AverageCreationTime time.Duration     `json:"average_creation_time"`
    P95CreationTime     time.Duration     `json:"p95_creation_time"`
    P99CreationTime     time.Duration     `json:"p99_creation_time"`
    SuccessRate         float64           `json:"success_rate"`
    ThroughputPerSecond float64           `json:"throughput_per_second"`
    ResourceUsage       ResourceUsage     `json:"resource_usage"`
}

// 보고서 생성 함수
func GeneratePerformanceReport(results []TestResult) *PerformanceReport {
    // 성능 메트릭 계산 및 보고서 생성
}
```

## 🔧 CI/CD 통합

### 1. GitHub Actions 설정

```yaml
# .github/workflows/integration-tests.yml
name: Integration Tests

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  integration-tests:
    runs-on: ubuntu-latest
    
    services:
      docker:
        image: docker:20.10-dind
        options: --privileged
        
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v3
      with:
        go-version: 1.21
        
    - name: Install dependencies
      run: |
        go mod download
        go install github.com/onsi/ginkgo/v2/ginkgo@latest
        
    - name: Set up test environment
      run: |
        docker network create aicli-test-network
        make test-setup
        
    - name: Run integration tests
      run: |
        make test-integration
        
    - name: Run performance tests
      run: |
        make test-performance
        
    - name: Upload test results
      uses: actions/upload-artifact@v3
      if: always()
      with:
        name: test-results
        path: |
          test-results.xml
          coverage.out
          performance-report.json
```

### 2. Makefile 통합

```makefile
# Makefile
.PHONY: test-setup test-integration test-performance test-all

test-setup:
	@echo "Setting up test environment..."
	@mkdir -p test/data test/workspaces test/logs
	@docker network create aicli-test-network 2>/dev/null || true
	@sqlite3 test/data/test.db < scripts/schema.sql

test-integration:
	@echo "Running integration tests..."
	@TEST_ENV=ci go test -v ./test/integration/ -timeout=30m -coverprofile=coverage.out

test-performance:
	@echo "Running performance tests..."
	@TEST_ENV=ci go test -v ./test/integration/ -run TestPerformance -timeout=60m

test-all: test-setup test-integration test-performance
	@echo "All tests completed"

test-cleanup:
	@echo "Cleaning up test environment..."
	@docker network rm aicli-test-network 2>/dev/null || true
	@rm -rf test/data test/workspaces test/logs
```

## 🚨 테스트 문제 해결

### 1. 일반적인 문제

#### Docker 연결 실패
```bash
# Docker 소켓 권한 확인
sudo chmod 666 /var/run/docker.sock

# Docker 서비스 상태 확인
sudo systemctl status docker

# 테스트용 Docker 네트워크 재생성
docker network rm aicli-test-network
docker network create aicli-test-network
```

#### 테스트 데이터베이스 문제
```bash
# 테스트 데이터베이스 재생성
rm -f test/data/test.db
sqlite3 test/data/test.db < scripts/schema.sql
```

#### Git 저장소 문제
```bash
# 테스트 Git 저장소 재생성
rm -rf test/fixtures/git/test-repo.git
mkdir -p test/fixtures/git
cd test/fixtures/git
git init --bare test-repo.git
```

### 2. 성능 테스트 문제

#### 시간 초과 문제
```bash
# 타임아웃 시간 늘리기
go test -v ./test/integration/ -run TestPerformance -timeout=120m

# 동시 에이전트 수 줄이기
PERF_TEST_CONCURRENT_AGENTS=50 go test -v ./test/integration/ -run TestPerformance
```

#### 리소스 부족 문제
```bash
# 시스템 리소스 확인
free -h
df -h

# Docker 정리
docker system prune -f
docker volume prune -f
```

## 📝 테스트 베스트 프랙티스

### 1. 테스트 작성 원칙

- **독립성**: 각 테스트는 독립적으로 실행 가능해야 함
- **반복성**: 동일한 환경에서 동일한 결과를 보장
- **명확성**: 테스트 의도가 명확하게 드러나야 함
- **빠른 피드백**: 가능한 한 빠른 실행 시간 유지

### 2. 테스트 데이터 관리

- **고정된 테스트 데이터**: 예측 가능한 결과를 위해 고정된 데이터 사용
- **데이터 격리**: 테스트 간 데이터 오염 방지
- **정리**: 테스트 후 생성된 리소스 자동 정리

### 3. 성능 테스트 가이드라인

- **기준점 설정**: 명확한 성능 목표 정의
- **환경 일관성**: 동일한 하드웨어/소프트웨어 환경에서 실행
- **메트릭 수집**: 상세한 성능 메트릭 수집 및 분석

---

이 가이드를 통해 AICode Manager의 품질을 보장하는 포괄적인 테스트를 작성하고 실행할 수 있습니다.

**다음**: [예제 코드](examples/)에서 실제 사용 예제를 확인하세요.