# 통합 테스트 가이드

## 개요

이 문서는 AICLI-Web 프로젝트의 통합 테스트, E2E 테스트, 그리고 성능 벤치마크 실행 방법을 설명합니다.

## 테스트 구조

### 디렉토리 구조

```
test/
├── helpers/                     # 테스트 헬퍼 및 유틸리티
│   ├── environment.go          # 테스트 환경 설정
│   └── assertions.go           # 커스텀 assertion 함수
├── integration/                 # 통합 테스트
│   ├── process_test.go         # 프로세스 관리 통합 테스트
│   └── stream_test.go          # 스트림 처리 통합 테스트
├── e2e/                        # E2E 테스트
│   └── claude_workflow_test.go # Claude 워크플로우 E2E 테스트
├── benchmark/                  # 성능 벤치마크
│   └── performance_test.go     # 성능 및 스트레스 테스트
└── testdata/                   # 테스트 데이터
    ├── streams/                # 스트림 테스트 데이터
    └── sessions/               # 세션 테스트 데이터
```

## 테스트 태그

각 테스트 유형은 빌드 태그로 구분됩니다:

- `integration`: 통합 테스트
- `e2e`: End-to-End 테스트  
- `benchmark`: 성능 벤치마크 및 스트레스 테스트

## 테스트 실행

### 1. 통합 테스트

통합 테스트는 실제 프로세스와 모킹된 Claude CLI를 사용하여 컴포넌트 간 상호작용을 검증합니다.

```bash
# 통합 테스트만 실행
make test-integration

# 또는 직접 실행
go test -v -race -tags=integration ./test/integration/...
```

#### 테스트 환경 변수

- `TEST_REAL_CLAUDE=true`: 실제 Claude CLI 사용 (기본값: false, 모킹 사용)
- `VERBOSE_TESTS=true`: 상세한 테스트 로그 출력

### 2. E2E 테스트

E2E 테스트는 전체 시스템의 워크플로우를 검증합니다.

```bash
# E2E 테스트 실행
make test-e2e

# 또는 직접 실행
go test -v -race -tags=e2e ./test/e2e/...
```

**주의**: E2E 테스트는 실행 시간이 오래 걸릴 수 있습니다.

### 3. 성능 벤치마크

```bash
# 성능 벤치마크 실행
make test-benchmark

# 스트레스 테스트 실행
make test-stress

# 또는 직접 실행
go test -v -race -tags=benchmark -bench=. -benchmem ./test/benchmark/...
```

### 4. 전체 테스트 실행

```bash
# 모든 테스트 실행 (단위 + 통합 + E2E + 벤치마크)
make test-all
```

## 테스트 환경 설정

### TestEnvironment

`TestEnvironment`는 통합 테스트를 위한 격리된 환경을 제공합니다:

```go
func TestExample(t *testing.T) {
    env := helpers.NewTestEnvironment(t)
    // env.TempDir        - 임시 디렉토리
    // env.MockClaude     - 모킹된 Claude 서버
    // env.TestData       - 테스트 데이터 프로바이더
    // env.APIServer      - 테스트용 API 서버
    
    // 테스트 로직...
}
```

### Mock Claude CLI

실제 Claude CLI 대신 모킹된 서버를 사용하여 테스트 속도를 향상시키고 일관성을 보장합니다:

```go
// 모킹된 응답 설정
env.MockClaude.SetResponse("code_generation", []byte(`{"type":"text","content":"Code generated"}`))

// 모킹된 프로세스 스크립트 생성
scriptPath := env.MockClaude.SimulateClaudeProcess()
```

## 테스트 데이터

### 스트림 데이터

테스트용 스트림 데이터는 JSONL 형식으로 제공됩니다:

```jsonl
{"type":"text","content":"응답 메시지"}
{"type":"tool_use","tool_name":"Write","input":{"file_path":"/tmp/test.go"}}
{"type":"completion","final":true}
```

### 동적 데이터 생성

`TestDataProvider`는 필요시 동적으로 테스트 데이터를 생성합니다:

```go
// 복잡한 스트림 데이터 로드
data := env.TestData.LoadStreamData("complex_response.jsonl")

// 대용량 스트림 데이터 생성
largeData := env.TestData.GenerateLargeStreamData(1024 * 1024) // 1MB
```

## 커스텀 Assertion

테스트 검증을 위한 헬퍼 함수들:

```go
// 메시지 타입 순서 검증
helpers.AssertMessageTypes(t, messages, []string{"text", "tool_use", "completion"})

// 도구 사용 검증
helpers.AssertContainsToolUse(t, messages, "Write")

// 성능 임계값 검증
helpers.AssertPerformanceThreshold(t, actualTime, maxTime, "response_time")

// 메모리 사용량 검증
helpers.AssertMemoryUsage(t, actualMB, thresholdMB)
```

## 성능 벤치마크

### 벤치마크 테스트

다양한 시나리오에 대한 성능 측정:

1. **스트림 처리 성능**
   - 다양한 크기의 데이터 처리
   - 동시 스트림 처리
   - 백프레셔 처리

2. **세션 관리 성능**
   - 세션 생성/조회 속도
   - 동시 세션 관리
   - 세션 풀 성능

3. **프로세스 관리 성능**
   - 프로세스 시작/종료 속도
   - 다중 프로세스 처리
   - 리소스 사용량

### 스트레스 테스트

장시간 실행 및 대량 부하 상황에서의 안정성 검증:

```go
// 5분간 지속적인 처리
func TestStressTest(t *testing.T) {
    duration := 5 * time.Minute
    // ... 스트레스 테스트 로직
}
```

## CI/CD 통합

### GitHub Actions

CI 파이프라인에서 자동 실행되는 테스트:

1. **단위 테스트**: 모든 PR에서 실행
2. **통합 테스트**: 모든 PR에서 실행  
3. **성능 벤치마크**: PR에서 실행 (실패해도 블록하지 않음)
4. **E2E 테스트**: 릴리스 브랜치에서만 실행

### 테스트 리포트

- 커버리지 리포트: Codecov 업로드
- 성능 벤치마크 결과: 아티팩트로 저장
- 테스트 결과: JUnit XML 형식으로 저장

## 베스트 프랙티스

### 1. 테스트 격리

- 각 테스트는 독립된 임시 디렉토리 사용
- 테스트 간 상태 공유 금지
- 적절한 cleanup 설정

### 2. 타임아웃 설정

```go
// 적절한 타임아웃 설정
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

### 3. 병렬 테스트

```go
// 안전한 병렬 테스트
func TestConcurrent(t *testing.T) {
    t.Parallel() // 병렬 실행 가능한 경우만
    // ...
}
```

### 4. 에러 처리

```go
// 적절한 에러 검증
require.NoError(t, err, "상세한 에러 메시지")
assert.Equal(t, expected, actual, "값 불일치: %v != %v", expected, actual)
```

### 5. 테스트 데이터 관리

- 실제 파일 대신 메모리 데이터 사용
- 테스트 데이터 재사용성 고려
- 적절한 크기의 테스트 데이터 사용

## 문제 해결

### 일반적인 문제들

1. **테스트 타임아웃**
   - 타임아웃 값 조정
   - 리소스 부족 확인
   - 네트워크 연결 상태 확인

2. **메모리 부족**
   - 테스트 데이터 크기 조정
   - 가비지 컬렉션 강제 실행
   - 동시 실행 수 제한

3. **파일 권한 문제**
   - 임시 디렉토리 권한 확인
   - 실행 권한 설정 확인

4. **프로세스 정리 실패**
   - cleanup 함수 등록 확인
   - 프로세스 종료 대기 시간 증가

### 디버깅 팁

```bash
# 상세한 로그로 실행
VERBOSE_TESTS=true go test -v -tags=integration ./test/integration/...

# 특정 테스트만 실행
go test -v -tags=integration -run=TestSpecificTest ./test/integration/...

# Race detector 없이 실행 (성능 테스트 시)
go test -v -tags=benchmark -bench=. ./test/benchmark/...
```

## 결론

이 통합 테스트 시스템은 AICLI-Web 프로젝트의 품질과 안정성을 보장하기 위해 설계되었습니다. 
정기적인 테스트 실행과 지속적인 개선을 통해 높은 코드 품질을 유지할 수 있습니다.