# Agent API 레퍼런스

AICode Manager Agent API의 완전한 레퍼런스 문서입니다.

## 기본 정보

- **Base URL**: `http://localhost:8080/api/v1`
- **인증**: Bearer JWT Token
- **Content-Type**: `application/json`
- **버전**: v1

## 인증

모든 API 요청에는 JWT 토큰이 필요합니다:

```bash
Authorization: Bearer YOUR_JWT_TOKEN
```

## 에이전트 관리

### 에이전트 목록 조회

에이전트 목록을 조회합니다.

```http
GET /agents
```

**Query Parameters:**
- `project_id` (string, optional): 프로젝트 ID로 필터링

**Response:**
```json
{
  "agents": [
    {
      "id": "agent-123",
      "name": "my-agent",
      "project_id": "project-456",
      "repository_url": "https://github.com/user/repo.git",
      "branch": "main",
      "status": "running",
      "docker_container_id": "container-789",
      "git_worktree_path": "/workspaces/agent-123",
      "created_at": "2025-01-01T12:00:00Z",
      "updated_at": "2025-01-01T12:00:00Z",
      "last_activity_at": "2025-01-01T12:00:00Z"
    }
  ],
  "count": 1
}
```

### 에이전트 생성

새로운 에이전트를 생성합니다.

```http
POST /agents
```

**Request Body:**
```json
{
  "name": "my-new-agent",
  "project_id": "project-456",
  "repository_url": "https://github.com/user/repo.git",
  "branch": "main",
  "description": "Agent for processing user requests"
}
```

**Response (201):**
```json
{
  "id": "agent-123",
  "name": "my-new-agent",
  "project_id": "project-456",
  "repository_url": "https://github.com/user/repo.git",
  "branch": "main",
  "status": "idle",
  "created_at": "2025-01-01T12:00:00Z"
}
```

### 에이전트 조회

특정 에이전트의 상세 정보를 조회합니다.

```http
GET /agents/{id}
```

**Path Parameters:**
- `id` (string, required): 에이전트 ID

**Response (200):**
```json
{
  "id": "agent-123",
  "name": "my-agent",
  "project_id": "project-456",
  "repository_url": "https://github.com/user/repo.git",
  "branch": "main",
  "status": "running",
  "docker_container_id": "container-789",
  "git_worktree_path": "/workspaces/agent-123",
  "created_at": "2025-01-01T12:00:00Z",
  "updated_at": "2025-01-01T12:00:00Z",
  "last_activity_at": "2025-01-01T12:00:00Z"
}
```

### 에이전트 수정

에이전트 정보를 수정합니다.

```http
PUT /agents/{id}
```

**Path Parameters:**
- `id` (string, required): 에이전트 ID

**Request Body:**
```json
{
  "name": "updated-agent-name",
  "branch": "develop",
  "description": "Updated description"
}
```

### 에이전트 삭제

에이전트를 삭제합니다.

```http
DELETE /agents/{id}
```

**Path Parameters:**
- `id` (string, required): 에이전트 ID

**Response:** `204 No Content`

## 에이전트 제어

### 에이전트 시작

에이전트를 시작합니다.

```http
POST /agents/{id}/start
```

**Response (200):**
```json
{
  "message": "에이전트가 시작되었습니다",
  "agent_id": "agent-123"
}
```

### 에이전트 중지

에이전트를 중지합니다.

```http
POST /agents/{id}/stop
```

**Response (200):**
```json
{
  "message": "에이전트가 중지되었습니다",
  "agent_id": "agent-123"
}
```

### 에이전트 재시작

에이전트를 재시작합니다.

```http
POST /agents/{id}/restart
```

**Response (200):**
```json
{
  "message": "에이전트가 재시작되었습니다",
  "agent_id": "agent-123"
}
```

## 모니터링

### 에이전트 상태 조회

에이전트의 현재 상태를 조회합니다.

```http
GET /agents/{id}/status
```

**Response (200):**
```json
{
  "agent_id": "agent-123",
  "status": "running",
  "container_status": "running",
  "uptime": "2h 30m 15s",
  "last_heartbeat": "2025-01-01T12:00:00Z",
  "resource_usage": {
    "cpu_percent": 25.5,
    "memory_mb": 512,
    "disk_usage_mb": 1024
  }
}
```

### 에이전트 헬스체크

에이전트의 건강 상태를 확인합니다.

```http
GET /agents/{id}/health
```

**Response (200):**
```json
{
  "status": "healthy",
  "timestamp": "2025-01-01T12:00:00Z",
  "checks": {
    "container": "healthy",
    "git_worktree": "healthy",
    "claude_cli": "healthy"
  }
}
```

### 에이전트 메트릭 조회

에이전트의 상세 메트릭을 조회합니다.

```http
GET /agents/{id}/metrics
```

**Response (200):**
```json
{
  "agent_id": "agent-123",
  "timestamp": "2025-01-01T12:00:00Z",
  "cpu_usage": {
    "current_percent": 25.5,
    "average_percent": 22.3,
    "peak_percent": 45.2
  },
  "memory_usage": {
    "current_mb": 512,
    "limit_mb": 2048,
    "usage_percent": 25.0
  },
  "network_io": {
    "rx_bytes": 1024000,
    "tx_bytes": 512000,
    "rx_rate_bps": 1500.5,
    "tx_rate_bps": 750.2
  }
}
```

## 배치 작업

### 에이전트 일괄 시작

여러 에이전트를 한 번에 시작합니다.

```http
POST /agents/batch/start
```

**Request Body:**
```json
{
  "agent_ids": ["agent-123", "agent-456", "agent-789"]
}
```

**Response (200):**
```json
{
  "results": [
    {
      "agent_id": "agent-123",
      "success": true,
      "message": "에이전트가 성공적으로 시작되었습니다",
      "error": null
    },
    {
      "agent_id": "agent-456",
      "success": false,
      "message": null,
      "error": "에이전트를 찾을 수 없습니다"
    }
  ],
  "count": 2
}
```

### 에이전트 일괄 중지

여러 에이전트를 한 번에 중지합니다.

```http
POST /agents/batch/stop
```

**Request Body:**
```json
{
  "agent_ids": ["agent-123", "agent-456", "agent-789"]
}
```

## 실시간 스트리밍

### 로그 스트리밍

WebSocket을 통해 에이전트 로그를 실시간으로 받습니다.

```http
GET /agents/{id}/logs/stream
```

**Query Parameters:**
- `follow` (boolean, default: true): 실시간 팔로우 여부
- `tail` (integer, default: 100): 마지막 N개 라인

**WebSocket Message:**
```json
{
  "type": "log",
  "data": {
    "agent_id": "agent-123",
    "timestamp": "2025-01-01T12:00:00Z",
    "level": "info",
    "message": "Sample log message from agent"
  }
}
```

### 이벤트 스트리밍

특정 에이전트의 이벤트를 실시간으로 받습니다.

```http
GET /agents/{id}/events/stream
```

**WebSocket Message:**
```json
{
  "type": "event",
  "data": {
    "type": "agent_started",
    "agent_id": "agent-123",
    "timestamp": "2025-01-01T12:00:00Z",
    "message": "Agent started successfully"
  }
}
```

### 전역 이벤트 스트리밍

모든 에이전트의 이벤트를 실시간으로 받습니다.

```http
GET /agents/events/stream
```

## 에러 코드

| 상태 코드 | 설명 |
|----------|------|
| 200 | 성공 |
| 201 | 생성됨 |
| 204 | 성공 (내용 없음) |
| 400 | 잘못된 요청 |
| 401 | 인증 실패 |
| 403 | 권한 없음 |
| 404 | 리소스 없음 |
| 500 | 서버 오류 |

### 에러 응답 형식

```json
{
  "error": {
    "code": "INVALID_REQUEST",
    "message": "요청이 잘못되었습니다.",
    "details": {
      "field": "repository_url",
      "reason": "올바른 URL 형식이 아닙니다"
    }
  }
}
```

## Rate Limiting

API 요청은 다음과 같이 제한됩니다:

- **일반 API**: 분당 100회
- **배치 작업**: 분당 10회
- **WebSocket 연결**: 동시 50개

Rate limit 초과 시 `429 Too Many Requests` 응답을 받습니다.

## SDK 예제

### cURL
```bash
# 에이전트 생성
curl -X POST http://localhost:8080/api/v1/agents \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{"name": "test-agent", "repository_url": "https://github.com/user/repo.git"}'

# 에이전트 시작
curl -X POST http://localhost:8080/api/v1/agents/agent-123/start \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### JavaScript
```javascript
const response = await fetch('http://localhost:8080/api/v1/agents', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer YOUR_TOKEN'
  },
  body: JSON.stringify({
    name: 'test-agent',
    repository_url: 'https://github.com/user/repo.git'
  })
});
const agent = await response.json();
```

### Python
```python
import requests

response = requests.post(
    'http://localhost:8080/api/v1/agents',
    headers={
        'Content-Type': 'application/json',
        'Authorization': 'Bearer YOUR_TOKEN'
    },
    json={
        'name': 'test-agent',
        'repository_url': 'https://github.com/user/repo.git'
    }
)
agent = response.json()
```

## 추가 리소스

- [Getting Started Guide](./getting-started.md)
- [Architecture Documentation](./architecture.md)
- [Deployment Guide](./deployment.md)
- [Troubleshooting Guide](./troubleshooting.md)