# Agent API Reference

AICode Manager 멀티 에이전트 플랫폼의 완전한 API 레퍼런스입니다.

## 📋 개요

- **Base URL**: `http://localhost:8080/api/v1`
- **Authentication**: Bearer Token (선택사항)
- **Content-Type**: `application/json`
- **API Version**: v1.0

## 🔗 엔드포인트 목록

### 에이전트 관리 (CRUD)
- [POST /agents](#post-agents) - 에이전트 생성
- [GET /agents](#get-agents) - 에이전트 목록 조회
- [GET /agents/{id}](#get-agentsid) - 에이전트 상세 조회
- [PUT /agents/{id}](#put-agentsid) - 에이전트 수정
- [DELETE /agents/{id}](#delete-agentsid) - 에이전트 삭제

### 에이전트 제어
- [POST /agents/{id}/start](#post-agentsidstart) - 에이전트 시작
- [POST /agents/{id}/stop](#post-agentsidstop) - 에이전트 중지
- [POST /agents/{id}/restart](#post-agentsidrestart) - 에이전트 재시작

### 모니터링 & 로깅
- [GET /agents/{id}/status](#get-agentsidstatus) - 상태 조회
- [GET /agents/{id}/health](#get-agentsidhealth) - 헬스체크
- [GET /agents/{id}/metrics](#get-agentsidmetrics) - 메트릭 조회
- [GET /agents/{id}/logs](#get-agentsidlogs) - 로그 조회

### 배치 작업
- [POST /agents/batch/start](#post-agentsbatchstart) - 에이전트 일괄 시작
- [POST /agents/batch/stop](#post-agentsbatchstop) - 에이전트 일괄 중지

### WebSocket 스트리밍
- [WS /agents/{id}/logs/stream](#ws-agentsidlogsstream) - 로그 실시간 스트리밍
- [WS /agents/{id}/events/stream](#ws-agentsideventstream) - 이벤트 실시간 스트리밍
- [WS /agents/events/stream](#ws-agentseventsstream) - 전역 이벤트 스트리밍

---

## 📖 상세 API 문서

### POST /agents

새로운 에이전트를 생성합니다.

**요청**
```http
POST /api/v1/agents
Content-Type: application/json

{
  "name": "my-agent",
  "project_id": "project-123",
  "agent_type": "standard",
  "description": "Agent description",
  "config": {
    "resources": {
      "cpu": "1.0",
      "memory": "1Gi"
    },
    "environment": {
      "NODE_ENV": "production"
    }
  }
}
```

**응답**
```http
HTTP/1.1 201 Created
Content-Type: application/json

{
  "id": "agent-uuid-123",
  "name": "my-agent",
  "project_id": "project-123",
  "agent_type": "standard",
  "status": "created",
  "description": "Agent description",
  "config": {
    "resources": {
      "cpu": "1.0",
      "memory": "1Gi"
    }
  },
  "created_at": "2025-07-30T23:00:00Z",
  "updated_at": "2025-07-30T23:00:00Z"
}
```

**필드 설명**
- `name` (string, required): 에이전트 이름 (고유해야 함)
- `project_id` (string, required): 프로젝트 ID
- `agent_type` (enum, required): `standard`, `gpu`, `memory_optimized`
- `description` (string, optional): 에이전트 설명
- `config.resources.cpu` (string, optional): CPU 할당량 (예: "0.5", "2.0")
- `config.resources.memory` (string, optional): 메모리 할당량 (예: "512Mi", "2Gi")
- `config.environment` (object, optional): 환경 변수

**에러 응답**
```http
HTTP/1.1 400 Bad Request
{
  "error": "invalid_request",
  "message": "Agent name must be unique",
  "code": "DUPLICATE_NAME"
}
```

---

### GET /agents

에이전트 목록을 조회합니다.

**요청**
```http
GET /api/v1/agents?project_id=project-123&status=running&limit=20&offset=0
```

**쿼리 파라미터**
- `project_id` (string, optional): 프로젝트 ID로 필터링
- `status` (enum, optional): 상태로 필터링 (`created`, `running`, `stopped`, `failed`)
- `agent_type` (enum, optional): 타입으로 필터링
- `limit` (int, optional): 결과 수 제한 (기본값: 50, 최대: 100)
- `offset` (int, optional): 페이지네이션 오프셋 (기본값: 0)
- `sort` (string, optional): 정렬 기준 (`created_at`, `updated_at`, `name`)
- `order` (enum, optional): 정렬 순서 (`asc`, `desc`)

**응답**
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "agents": [
    {
      "id": "agent-uuid-123",
      "name": "my-agent",
      "project_id": "project-123",
      "agent_type": "standard",
      "status": "running",
      "container_id": "container-456",
      "created_at": "2025-07-30T23:00:00Z",
      "updated_at": "2025-07-30T23:01:00Z"
    }
  ],
  "pagination": {
    "total": 150,
    "limit": 20,
    "offset": 0,
    "has_next": true,
    "has_prev": false
  }
}
```

---

### GET /agents/{id}

특정 에이전트의 상세 정보를 조회합니다.

**요청**
```http
GET /api/v1/agents/agent-uuid-123
```

**응답**
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "id": "agent-uuid-123",
  "name": "my-agent",
  "project_id": "project-123",
  "agent_type": "standard",
  "status": "running",
  "description": "Agent description",
  "container_id": "container-456",
  "config": {
    "resources": {
      "cpu": "1.0",
      "memory": "1Gi"
    },
    "environment": {
      "NODE_ENV": "production"
    }
  },
  "metadata": {
    "uptime_seconds": 3600,
    "restart_count": 0,
    "last_health_check": "2025-07-30T23:30:00Z"
  },
  "created_at": "2025-07-30T23:00:00Z",
  "updated_at": "2025-07-30T23:01:00Z"
}
```

**에러 응답**
```http
HTTP/1.1 404 Not Found
{
  "error": "not_found",
  "message": "Agent not found",
  "code": "AGENT_NOT_FOUND"
}
```

---

### PUT /agents/{id}

에이전트의 설정을 수정합니다.

**요청**
```http
PUT /api/v1/agents/agent-uuid-123
Content-Type: application/json

{
  "description": "Updated description",
  "config": {
    "resources": {
      "cpu": "2.0",
      "memory": "2Gi"
    }
  }
}
```

**응답**
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "id": "agent-uuid-123",
  "name": "my-agent",
  "description": "Updated description",
  "config": {
    "resources": {
      "cpu": "2.0",
      "memory": "2Gi"
    }
  },
  "updated_at": "2025-07-30T23:35:00Z"
}
```

**주의사항**
- 실행 중인 에이전트의 리소스 변경은 재시작이 필요합니다
- `name`, `project_id`, `agent_type`은 수정할 수 없습니다

---

### DELETE /agents/{id}

에이전트를 삭제합니다.

**요청**
```http
DELETE /api/v1/agents/agent-uuid-123
```

**응답**
```http
HTTP/1.1 204 No Content
```

**주의사항**
- 실행 중인 에이전트는 먼저 중지한 후 삭제됩니다
- 삭제된 에이전트는 복구할 수 없습니다

---

### POST /agents/{id}/start

에이전트를 시작합니다.

**요청**
```http
POST /api/v1/agents/agent-uuid-123/start
```

**응답**
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "id": "agent-uuid-123",
  "status": "starting",
  "message": "Agent start initiated",
  "started_at": "2025-07-30T23:40:00Z"
}
```

**에러 응답**
```http
HTTP/1.1 409 Conflict
{
  "error": "invalid_state",
  "message": "Agent is already running",
  "code": "ALREADY_RUNNING"
}
```

---

### POST /agents/{id}/stop

에이전트를 중지합니다.

**요청**
```http
POST /api/v1/agents/agent-uuid-123/stop
Content-Type: application/json

{
  "force": false,
  "timeout": 30
}
```

**요청 필드**
- `force` (boolean, optional): 강제 중지 여부 (기본값: false)
- `timeout` (int, optional): 중지 대기 시간(초) (기본값: 30)

**응답**
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "id": "agent-uuid-123",
  "status": "stopping",
  "message": "Agent stop initiated",
  "stopped_at": "2025-07-30T23:41:00Z"
}
```

---

### POST /agents/{id}/restart

에이전트를 재시작합니다.

**요청**
```http
POST /api/v1/agents/agent-uuid-123/restart
```

**응답**
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "id": "agent-uuid-123",
  "status": "restarting",
  "message": "Agent restart initiated",
  "restarted_at": "2025-07-30T23:42:00Z"
}
```

---

### GET /agents/{id}/status

에이전트의 현재 상태를 조회합니다.

**요청**
```http
GET /api/v1/agents/agent-uuid-123/status
```

**응답**
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "agent_id": "agent-uuid-123",
  "status": "running",
  "container_status": "running",
  "health_status": "healthy",
  "uptime_seconds": 3600,
  "last_seen": "2025-07-30T23:42:30Z",
  "resource_usage": {
    "cpu_percent": 25.5,
    "memory_usage": "512MB",
    "memory_percent": 50.0
  }
}
```

---

### GET /agents/{id}/metrics

에이전트의 메트릭을 조회합니다.

**요청**
```http
GET /api/v1/agents/agent-uuid-123/metrics?duration=1h
```

**쿼리 파라미터**
- `duration` (string, optional): 메트릭 수집 기간 (기본값: 1h)

**응답**
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "agent_id": "agent-uuid-123",
  "uptime_seconds": 3600,
  "restart_count": 0,
  "resource_usage": {
    "cpu_percent": 25.5,
    "memory_usage": 536870912,
    "memory_percent": 50.0,
    "network_rx_bytes": 1048576,
    "network_tx_bytes": 2097152
  },
  "performance_stats": {
    "avg_cpu_percent": 22.1,
    "max_memory_usage": 671088640,
    "task_completion_count": 45,
    "error_count": 2
  },
  "collected_at": "2025-07-30T23:43:00Z"
}
```

---

### WebSocket API

### WS /agents/{id}/logs/stream

에이전트 로그를 실시간으로 스트리밍합니다.

**연결**
```javascript
const ws = new WebSocket('ws://localhost:8080/api/v1/agents/agent-uuid-123/logs/stream');

ws.onmessage = function(event) {
  const log = JSON.parse(event.data);
  console.log(log);
};
```

**메시지 형식**
```json
{
  "timestamp": "2025-07-30T23:44:00Z",
  "level": "info",
  "message": "Task completed successfully",
  "agent_id": "agent-uuid-123",
  "container_id": "container-456",
  "source": "stdout"
}
```

**쿼리 파라미터**
- `since` (string, optional): 특정 시간 이후 로그만 수신
- `level` (string, optional): 로그 레벨 필터 (`debug`, `info`, `warn`, `error`)
- `follow` (boolean, optional): 실시간 스트리밍 여부 (기본값: true)

---

### WS /agents/{id}/events/stream

에이전트 이벤트를 실시간으로 스트리밍합니다.

**연결**
```javascript
const ws = new WebSocket('ws://localhost:8080/api/v1/agents/agent-uuid-123/events/stream');
```

**이벤트 형식**
```json
{
  "event_type": "agent.started",
  "timestamp": "2025-07-30T23:45:00Z",
  "agent_id": "agent-uuid-123",
  "data": {
    "container_id": "container-456",
    "startup_time_ms": 2500
  }
}
```

**이벤트 타입**
- `agent.created` - 에이전트 생성
- `agent.started` - 에이전트 시작
- `agent.stopped` - 에이전트 중지
- `agent.failed` - 에이전트 실패
- `agent.health_check` - 헬스체크 결과
- `agent.resource_limit` - 리소스 제한 도달

---

## 🚦 HTTP 상태 코드

| 코드 | 설명 | 사용 상황 |
|------|------|-----------|
| 200 | OK | 요청 성공 |
| 201 | Created | 리소스 생성 성공 |
| 204 | No Content | 삭제 성공 |
| 400 | Bad Request | 잘못된 요청 |
| 401 | Unauthorized | 인증 실패 |
| 403 | Forbidden | 권한 없음 |
| 404 | Not Found | 리소스 없음 |
| 409 | Conflict | 상태 충돌 |
| 422 | Unprocessable Entity | 유효성 검증 실패 |
| 429 | Too Many Requests | 요청 제한 초과 |
| 500 | Internal Server Error | 서버 오류 |

## 🔧 에러 응답 형식

모든 에러 응답은 일관된 형식을 따릅니다:

```json
{
  "error": "error_type",
  "message": "Human readable error message",
  "code": "ERROR_CODE",
  "details": {
    "field": "validation error details"
  },
  "timestamp": "2025-07-30T23:46:00Z",
  "request_id": "req-uuid-789"
}
```

## 📊 Rate Limiting

API 요청은 다음과 같이 제한됩니다:

- **일반 API**: 초당 100 요청
- **에이전트 생성**: 분당 50 요청
- **WebSocket 연결**: 동시 100 연결

제한 초과 시 `429 Too Many Requests` 응답을 받습니다.

## 🔒 인증 (선택사항)

Bearer Token을 사용한 인증을 지원합니다:

```http
Authorization: Bearer your-api-token-here
```

---

**다음**: [개발자 가이드](developer-guide.md)에서 SDK 사용법을 확인하세요.