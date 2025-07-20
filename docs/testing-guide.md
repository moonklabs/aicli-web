# 테스트 가이드

## 개요

이 문서는 AICode Manager 프로젝트의 테스트 프레임워크와 테스트 작성 가이드를 제공합니다.

## 테스트 구조

```
aicli-web/
├── internal/
│   ├── cli/
│   │   ├── cli_test.go      # CLI 단위 테스트
│   │   └── root_test.go     # Root 명령어 테스트
│   ├── server/
│   │   ├── server_test.go   # 서버 단위 테스트
│   │   ├── router_test.go   # 라우터 테스트
│   │   └── server_bench_test.go # 벤치마크 테스트
│   └── testutil/            # 테스트 유틸리티
│       ├── helpers.go       # 헬퍼 함수
│       ├── mocks.go        # 목(Mock) 구현
│       ├── fixtures.go     # 테스트 데이터
│       └── database_mock.go # DB 모킹
├── pkg/
│   └── version/
│       ├── version_test.go      # 버전 패키지 테스트
│       └── version_bench_test.go # 벤치마크 테스트
└── test/
    └── integration_test.go  # 통합 테스트
```

## 테스트 유형

### 1. 단위 테스트 (Unit Tests)

개별 함수나 메서드를 테스트합니다.

```go
func TestNewCompletionCmd(t *testing.T) {
    cmd := newCompletionCmd()
    
    testutil.AssertNotNil(t, cmd)
    testutil.AssertEqual(t, "completion [bash|zsh|fish|powershell]", cmd.Use)
}
```

### 2. 통합 테스트 (Integration Tests)

여러 컴포넌트가 함께 작동하는 것을 테스트합니다.

```go
//go:build integration
// +build integration

func TestAPIServerIntegration(t *testing.T) {
    srv := server.New()
    ts := httptest.NewServer(srv.Router())
    defer ts.Close()
    
    // 실제 HTTP 요청을 보내 테스트
}
```

### 3. 벤치마크 테스트 (Benchmark Tests)

성능을 측정합니다.

```go
func BenchmarkHealthCheck(b *testing.B) {
    s := New()
    
    b.ResetTimer()
    b.ReportAllocs()
    
    for i := 0; i < b.N; i++ {
        // 벤치마크 코드
    }
}
```

## 테스트 실행

### 모든 테스트 실행
```bash
make test
```

### 단위 테스트만 실행
```bash
make test-unit
```

### 통합 테스트만 실행
```bash
make test-integration
```

### 테스트 커버리지 측정
```bash
make test-coverage
```

### 벤치마크 테스트 실행
```bash
make test-bench
```

### 특정 패키지 테스트
```bash
go test -v ./internal/cli
```

### 특정 테스트 함수 실행
```bash
go test -v -run TestNewCompletionCmd ./internal/cli
```

## 테스트 작성 가이드

### 1. 테이블 드리븐 테스트

여러 케이스를 효율적으로 테스트합니다.

```go
func TestCompletionCmd_Execute(t *testing.T) {
    tests := []struct {
        name      string
        args      []string
        wantErr   bool
        errMsg    string
    }{
        {
            name:    "bash completion",
            args:    []string{"bash"},
            wantErr: false,
        },
        {
            name:    "invalid shell",
            args:    []string{"invalid"},
            wantErr: true,
            errMsg:  "지원하지 않는 셸: invalid",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // 테스트 실행
        })
    }
}
```

### 2. 테스트 헬퍼 사용

`internal/testutil` 패키지의 헬퍼 함수를 활용합니다.

```go
// 임시 디렉토리 생성
tmpDir := testutil.TempDir(t, "test-prefix")

// 값 비교
testutil.AssertEqual(t, expected, actual)

// nil 체크
testutil.AssertNotNil(t, value)

// 문자열 포함 체크
testutil.AssertContains(t, str, substr)
```

### 3. 목(Mock) 사용

외부 의존성을 모킹합니다.

```go
// HTTP 클라이언트 모킹
mockClient := &testutil.MockHTTPClient{
    DoFunc: func(req *http.Request) (*http.Response, error) {
        return &http.Response{
            StatusCode: 200,
            Body:       io.NopCloser(strings.NewReader("response")),
        }, nil
    },
}

// 파일시스템 모킹
mockFS := &testutil.MockFileSystem{
    ReadFunc: func(name string) ([]byte, error) {
        return []byte("file content"), nil
    },
}
```

### 4. 병렬 테스트

독립적인 테스트는 병렬로 실행합니다.

```go
func TestParallel(t *testing.T) {
    t.Parallel() // 이 테스트를 병렬로 실행
    
    // 테스트 코드
}
```

### 5. 테스트 전후 처리

```go
func TestWithSetup(t *testing.T) {
    // Setup
    oldValue := someGlobalVar
    someGlobalVar = "test"
    
    // Cleanup
    t.Cleanup(func() {
        someGlobalVar = oldValue
    })
    
    // 테스트 코드
}
```

## 테스트 모범 사례

### 1. 명확한 테스트 이름

테스트 이름은 무엇을 테스트하는지 명확하게 표현해야 합니다.

```go
// Good
func TestNewCompletionCmd_ReturnsValidCommand(t *testing.T)
func TestHealthCheck_Returns200OK(t *testing.T)

// Bad
func TestCmd(t *testing.T)
func Test1(t *testing.T)
```

### 2. 한글 주석 사용

프로젝트 규칙에 따라 한글 주석을 사용합니다.

```go
func TestExample(t *testing.T) {
    // 임시 디렉토리 생성
    tmpDir := testutil.TempDir(t, "example")
    
    // 설정 파일 생성
    config := createTestConfig()
    
    // 서버 시작
    srv := startTestServer(config)
}
```

### 3. 에러 케이스 테스트

정상 케이스뿐만 아니라 에러 케이스도 테스트합니다.

```go
// 정상 케이스
t.Run("valid input", func(t *testing.T) {
    result, err := doSomething("valid")
    testutil.AssertNil(t, err)
    testutil.AssertEqual(t, "expected", result)
})

// 에러 케이스
t.Run("invalid input", func(t *testing.T) {
    _, err := doSomething("")
    testutil.AssertNotNil(t, err)
    testutil.AssertContains(t, err.Error(), "invalid input")
})
```

### 4. 테스트 격리

각 테스트는 독립적으로 실행될 수 있어야 합니다.

```go
func TestIsolated(t *testing.T) {
    // 각 테스트마다 새로운 인스턴스 생성
    srv := server.New()
    
    // 테스트별 임시 데이터 사용
    tmpDir := testutil.TempDir(t, "isolated")
    
    // 전역 상태 변경 시 복원
    defer resetGlobalState()
}
```

### 5. 테스트 데이터 관리

테스트 데이터는 `testutil/fixtures.go`를 활용합니다.

```go
// 테스트 데이터 로드
testData := testutil.LoadTestData(t)

// 특정 테스트 데이터 생성
project := testutil.GetTestProject("test-project")
user := testutil.GetTestUser("testuser")
```

## 테스트 커버리지

### 목표
- 전체 커버리지: 70% 이상
- 핵심 비즈니스 로직: 90% 이상
- 유틸리티 함수: 80% 이상

### 커버리지 확인
```bash
# 커버리지 측정 및 HTML 리포트 생성
make test-coverage

# 터미널에서 커버리지 확인
go test -cover ./...

# 상세 커버리지 확인
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

### 커버리지 제외
일부 코드는 테스트에서 제외할 수 있습니다.

```go
// 코드 생성 파일
//go:generate ...

// 메인 함수 (통합 테스트에서 커버)
func main() {
    // ...
}
```

## CI/CD 통합

GitHub Actions에서 자동으로 테스트가 실행됩니다.

```yaml
- name: Run tests
  run: |
    make test
    make test-coverage
    
- name: Upload coverage
  uses: codecov/codecov-action@v3
  with:
    file: ./coverage.out
```

## 문제 해결

### 테스트 실패 시
1. 에러 메시지를 자세히 읽어봅니다
2. `-v` 플래그로 상세 출력을 확인합니다
3. 특정 테스트만 실행하여 문제를 격리합니다
4. 테스트 데이터나 환경 설정을 확인합니다

### 느린 테스트
1. 병렬 실행을 활용합니다 (`t.Parallel()`)
2. 무거운 초기화는 `TestMain`에서 한 번만 수행합니다
3. 통합 테스트는 별도로 분리합니다
4. 벤치마크로 성능 병목을 찾습니다

### 플레이키 테스트 (Flaky Tests)
1. 시간 의존성을 제거합니다 (MockTimeProvider 사용)
2. 동시성 문제를 확인합니다
3. 외부 의존성을 모킹합니다
4. 테스트 격리를 확인합니다