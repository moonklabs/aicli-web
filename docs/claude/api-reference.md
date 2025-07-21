# Claude CLI API 레퍼런스

## 개요

이 문서는 AICode Manager의 Claude CLI 통합을 위한 REST API 및 WebSocket API 레퍼런스입니다.

## 기본 정보

- **Base URL**: `http://localhost:8080/api/v1`
- **Content-Type**: `application/json`
- **인증**: JWT Bearer 토큰
- **API 버전**: v1

## 인증

모든 API 요청은 JWT 토큰을 통한 인증이 필요합니다.

```http
Authorization: Bearer <jwt-token>
```

### 토큰 획득
```bash
curl -X POST /api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "user", "password": "pass"}'
```

## Claude 세션 API

### 세션 생성

**POST** `/api/v1/claude/sessions`

세션을 생성합니다.

```json
{
  "workspace_id": "my-project",
  "system_prompt": "You are a helpful coding assistant",
  "max_turns": 10,
  "allowed_tools": ["Read", "Write", "Bash"],
  "timeout": "5m",
  "config": {
    "max_retries": 3,
    "backpressure_policy": "drop_oldest"
  }
}
```

**응답**:
```json
{
  "id": "session-123",
  "workspace_id": "my-project",
  "status": "idle",
  "created_at": "2025-07-22T01:20:00Z",
  "config": {
    "system_prompt": "You are a helpful coding assistant",
    "max_turns": 10,
    "allowed_tools": ["Read", "Write", "Bash"],
    "timeout": "5m"
  }
}
```

### 세션 목록 조회

**GET** `/api/v1/claude/sessions`

세션 목록을 조회합니다.

**쿼리 매개변수**:
- `workspace_id` (string, optional): 워크스페이스 ID 필터
- `status` (string, optional): 세션 상태 필터 (idle, running, closed, error)
- `limit` (int, optional): 결과 제한 (기본값: 50)
- `offset` (int, optional): 오프셋 (기본값: 0)

**응답**:
```json
{
  "sessions": [
    {
      "id": "session-123",
      "workspace_id": "my-project",
      "status": "idle",
      "created_at": "2025-07-22T01:20:00Z",
      "last_activity": "2025-07-22T01:25:00Z"
    }
  ],
  "total": 1,
  "limit": 50,
  "offset": 0
}
```

### 세션 상세 조회

**GET** `/api/v1/claude/sessions/{session-id}`

특정 세션의 상세 정보를 조회합니다.

**응답**:
```json
{
  "id": "session-123",
  "workspace_id": "my-project",
  "status": "idle",
  "created_at": "2025-07-22T01:20:00Z",
  "last_activity": "2025-07-22T01:25:00Z",
  "config": {
    "system_prompt": "You are a helpful coding assistant",
    "max_turns": 10,
    "allowed_tools": ["Read", "Write", "Bash"]
  },
  "statistics": {
    "total_requests": 5,
    "successful_requests": 4,
    "failed_requests": 1,
    "average_response_time": "2.5s",
    "total_tokens": 1250
  }
}
```

### 세션 삭제

**DELETE** `/api/v1/claude/sessions/{session-id}`

세션을 삭제합니다.

**응답**:
```json
{
  "message": "Session deleted successfully"
}
```

## Claude 실행 API

### 프롬프트 실행

**POST** `/api/v1/claude/execute`

Claude 프롬프트를 실행합니다.

```json
{
  "session_id": "session-123",
  "prompt": "Write a Go function to reverse a string",
  "stream": true,
  "tools": ["Read", "Write"],
  "context": {
    "workspace_path": "/workspace/my-project",
    "files": [
      {
        "path": "main.go",
        "content": "package main\n\nfunc main() {\n}"
      }
    ]
  }
}
```

**응답 (스트림 비활성화)**:
```json
{
  "execution_id": "exec-456",
  "session_id": "session-123",
  "status": "completed",
  "response": {
    "type": "text",
    "content": "func reverseString(s string) string {\n  runes := []rune(s)\n  for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {\n    runes[i], runes[j] = runes[j], runes[i]\n  }\n  return string(runes)\n}",
    "tokens": 125,
    "tool_calls": []
  },
  "created_at": "2025-07-22T01:30:00Z",
  "completed_at": "2025-07-22T01:30:15Z"
}
```

**응답 (스트림 활성화)**:
```json
{
  "execution_id": "exec-456",
  "session_id": "session-123", 
  "status": "started",
  "stream_url": "ws://localhost:8080/ws/executions/exec-456",
  "created_at": "2025-07-22T01:30:00Z"
}
```

### 실행 상태 조회

**GET** `/api/v1/claude/executions/{execution-id}`

실행 상태를 조회합니다.

**응답**:
```json
{
  "execution_id": "exec-456",
  "session_id": "session-123",
  "status": "running",
  "progress": {
    "current_step": "processing",
    "total_steps": 3,
    "percentage": 66
  },
  "created_at": "2025-07-22T01:30:00Z",
  "estimated_completion": "2025-07-22T01:30:30Z"
}
```

### 실행 취소

**POST** `/api/v1/claude/executions/{execution-id}/cancel`

실행 중인 프롬프트를 취소합니다.

**응답**:
```json
{
  "execution_id": "exec-456",
  "status": "cancelled",
  "cancelled_at": "2025-07-22T01:30:25Z"
}
```

## WebSocket API

### 실시간 스트림

실행 스트림을 실시간으로 받아볼 수 있습니다.

**연결**: `ws://localhost:8080/ws/executions/{execution-id}`

**인증**: WebSocket 연결 시 토큰을 쿼리 매개변수로 전달
```
ws://localhost:8080/ws/executions/exec-456?token=<jwt-token>
```

### 메시지 형식

#### 텍스트 메시지
```json
{
  "type": "text",
  "execution_id": "exec-456",
  "timestamp": "2025-07-22T01:30:05Z",
  "data": {
    "content": "Here's the Go function you requested:\n\n"
  }
}
```

#### 도구 사용 메시지
```json
{
  "type": "tool_use",
  "execution_id": "exec-456", 
  "timestamp": "2025-07-22T01:30:08Z",
  "data": {
    "tool": "Write",
    "parameters": {
      "file_path": "/workspace/my-project/utils.go",
      "content": "package main\n\nfunc reverseString..."
    }
  }
}
```

#### 에러 메시지
```json
{
  "type": "error",
  "execution_id": "exec-456",
  "timestamp": "2025-07-22T01:30:10Z",
  "data": {
    "code": "TOOL_ERROR",
    "message": "Failed to write file: permission denied",
    "details": {
      "file_path": "/workspace/my-project/utils.go"
    }
  }
}
```

#### 완료 메시지
```json
{
  "type": "complete",
  "execution_id": "exec-456",
  "timestamp": "2025-07-22T01:30:15Z",
  "data": {
    "status": "completed",
    "summary": {
      "tokens_used": 125,
      "tools_called": 1,
      "execution_time": "15s"
    }
  }
}
```

## 시스템 API

### 헬스체크

**GET** `/api/v1/claude/health`

Claude 시스템 상태를 확인합니다.

**응답**:
```json
{
  "status": "healthy",
  "claude_cli_status": "running",
  "active_sessions": 3,
  "system_resources": {
    "cpu_usage": "25%",
    "memory_usage": "512MB",
    "disk_usage": "2GB"
  },
  "uptime": "2h30m45s",
  "last_check": "2025-07-22T01:30:00Z"
}
```

### 메트릭 조회

**GET** `/api/v1/claude/metrics`

시스템 메트릭을 조회합니다.

**응답**:
```json
{
  "sessions": {
    "total": 10,
    "active": 3,
    "idle": 5,
    "closed": 2
  },
  "executions": {
    "total": 150,
    "successful": 142,
    "failed": 8,
    "average_duration": "12.5s"
  },
  "resources": {
    "cpu_usage": 25.5,
    "memory_usage": 536870912,
    "goroutines": 45
  },
  "timestamp": "2025-07-22T01:30:00Z"
}
```

### 로그 조회

**GET** `/api/v1/claude/logs`

Claude 관련 로그를 조회합니다.

**쿼리 매개변수**:
- `level` (string): 로그 레벨 (debug, info, warn, error)
- `session_id` (string): 세션 ID 필터
- `since` (string): 시작 시간 (RFC3339 형식)
- `until` (string): 종료 시간 (RFC3339 형식)
- `limit` (int): 결과 제한 (기본값: 100)

**응답**:
```json
{
  "logs": [
    {
      "timestamp": "2025-07-22T01:30:00Z",
      "level": "info",
      "session_id": "session-123",
      "message": "Session created successfully",
      "data": {
        "workspace_id": "my-project",
        "system_prompt": "You are a helpful coding assistant"
      }
    }
  ],
  "total": 1,
  "limit": 100
}
```

## 에러 코드

### HTTP 상태 코드

- `200`: 성공
- `201`: 생성됨
- `400`: 잘못된 요청
- `401`: 인증 실패
- `403`: 권한 없음
- `404`: 찾을 수 없음
- `409`: 충돌 (세션 중복 등)
- `429`: 요청 한도 초과
- `500`: 서버 내부 오류
- `503`: 서비스 사용 불가

### 에러 응답 형식

```json
{
  "error": {
    "code": "SESSION_NOT_FOUND",
    "message": "Session not found",
    "details": {
      "session_id": "session-123"
    },
    "timestamp": "2025-07-22T01:30:00Z"
  }
}
```

### Claude 특화 에러 코드

- `CLAUDE_CLI_NOT_FOUND`: Claude CLI 실행 파일을 찾을 수 없음
- `TOKEN_EXPIRED`: OAuth 토큰 만료
- `SESSION_LIMIT_EXCEEDED`: 세션 한도 초과
- `STREAM_PARSING_ERROR`: 스트림 파싱 실패
- `TOOL_EXECUTION_ERROR`: 도구 실행 실패
- `PROCESS_CRASH`: Claude 프로세스 크래시
- `TIMEOUT_ERROR`: 실행 타임아웃
- `MEMORY_LIMIT_EXCEEDED`: 메모리 한도 초과

## Rate Limiting

API 요청은 rate limiting이 적용됩니다:

- **기본 한도**: 분당 100 요청
- **세션 생성**: 분당 10 요청
- **프롬프트 실행**: 분당 50 요청

Rate limit 정보는 응답 헤더에 포함됩니다:

```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1642781400
```

## SDK 및 클라이언트 라이브러리

### Go 클라이언트
```go
import "aicli-web/pkg/client"

client := client.New(&client.Config{
    BaseURL: "http://localhost:8080",
    Token:   "your-jwt-token",
})

session, err := client.CreateSession(&client.SessionRequest{
    WorkspaceID:  "my-project",
    SystemPrompt: "You are a helpful assistant",
})
```

### JavaScript/TypeScript 클라이언트
```typescript
import { ClaudeClient } from '@aicli/client';

const client = new ClaudeClient({
  baseUrl: 'http://localhost:8080',
  token: 'your-jwt-token'
});

const session = await client.createSession({
  workspaceId: 'my-project',
  systemPrompt: 'You are a helpful assistant'
});
```

### Python 클라이언트
```python
from aicli_client import ClaudeClient

client = ClaudeClient(
    base_url='http://localhost:8080',
    token='your-jwt-token'
)

session = client.create_session(
    workspace_id='my-project',
    system_prompt='You are a helpful assistant'
)
```

## 예제 및 튜토리얼

자세한 사용 예제는 [사용 가이드](./usage-guide.md)를 참조하세요.

## OpenAPI 스펙

완전한 OpenAPI 3.0 스펙은 다음에서 확인할 수 있습니다:
- Swagger UI: `http://localhost:8080/swagger/`
- JSON 스펙: `http://localhost:8080/swagger/doc.json`
- YAML 스펙: `http://localhost:8080/swagger/doc.yaml`