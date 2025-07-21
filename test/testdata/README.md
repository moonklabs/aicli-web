# 테스트 데이터 디렉토리

이 디렉토리는 통합 테스트와 E2E 테스트에서 사용되는 테스트 데이터를 포함합니다.

## 디렉토리 구조

```
testdata/
├── README.md                    # 이 파일
├── streams/                     # 스트림 테스트 데이터
│   ├── simple_response.jsonl   # 단순 응답 스트림
│   ├── complex_response.jsonl  # 복합 응답 스트림
│   ├── tool_use_response.jsonl # 도구 사용 응답 스트림
│   └── error_response.jsonl    # 에러 처리 응답 스트림
├── sessions/                   # 세션 테스트 데이터
│   ├── session_configs.json   # 세션 설정 예제들
│   └── session_workflows.json # 워크플로우 테스트 시나리오
├── processes/                  # 프로세스 테스트 데이터
│   ├── mock_claude.sh         # Claude CLI 모킹 스크립트
│   └── test_commands.json     # 테스트 명령어들
└── benchmarks/                # 벤치마크 테스트 데이터
    ├── large_stream.jsonl     # 대용량 스트림 데이터
    └── performance_configs.json # 성능 테스트 설정
```

## 사용법

테스트 데이터는 `TestDataProvider`를 통해 로드됩니다:

```go
env := helpers.NewTestEnvironment(t)
data := env.TestData.LoadStreamData("complex_response.jsonl")
```

## 데이터 형식

### 스트림 데이터 (.jsonl)

각 줄은 하나의 JSON 메시지를 나타냅니다:

```jsonl
{"type":"text","content":"응답 메시지"}
{"type":"tool_use","tool_name":"Write","input":{"file_path":"/tmp/test.go","content":"package main"}}
{"type":"completion","final":true}
```

### 설정 파일 (.json)

JSON 형식의 설정 데이터:

```json
{
  "test_sessions": [
    {
      "name": "simple_session",
      "system_prompt": "You are a helpful assistant",
      "tools": ["Write", "Read"]
    }
  ]
}
```

## 테스트 데이터 생성

필요한 경우 `TestDataProvider`가 동적으로 테스트 데이터를 생성합니다. 
실제 파일이 없으면 기본 데이터나 생성된 데이터를 사용합니다.