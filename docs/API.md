# API 문서

## 🌐 API 개요

AICode Manager API는 RESTful 설계 원칙을 따르며, JSON 형식으로 데이터를 주고받습니다.

### 기본 정보

- **Base URL**: `http://localhost:8080/api/v1`
- **인증 방식**: JWT Bearer Token
- **Content-Type**: `application/json`
- **API 버전**: v1

### 인증 헤더

```http
Authorization: Bearer <your-jwt-token>
```

## 🔐 인증 API

### 로그인

```http
POST /api/v1/auth/login
```

**요청 본문:**
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**응답:**
```json
{
  "success": true,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "user": {
      "id": "user-123",
      "email": "user@example.com",
      "name": "John Doe"
    },
    "expires_at": "2025-08-11T12:00:00Z"
  }
}
```

### 회원가입

```http
POST /api/v1/auth/register
```

**요청 본문:**
```json
{
  "email": "user@example.com",
  "password": "password123",
  "name": "John Doe"
}
```

### 토큰 갱신

```http
POST /api/v1/auth/refresh
```

**요청 헤더:**
```http
Authorization: Bearer <expired-token>
```

## 📊 세션 관리 API

### 세션 생성

```http
POST /api/v1/sessions
```

**요청 본문:**
```json
{
  "workspace_id": "ws-123",
  "command": "claude",
  "args": ["--model", "claude-3-sonnet"],
  "config": {
    "rows": 24,
    "cols": 80,
    "term": "xterm-256color"
  }
}
```

**응답:**
```json
{
  "success": true,
  "data": {
    "session_id": "sess-456",
    "workspace_id": "ws-123",
    "status": "running",
    "created_at": "2025-08-10T10:00:00Z",
    "pty": {
      "rows": 24,
      "cols": 80
    },
    "websocket_url": "ws://localhost:8080/api/v1/ws/terminal/sess-456"
  }
}
```

### 세션 목록 조회

```http
GET /api/v1/sessions
```

**쿼리 파라미터:**
- `workspace_id` (선택): 특정 워크스페이스의 세션만 조회
- `status` (선택): active, idle, terminated
- `page` (선택): 페이지 번호 (기본값: 1)
- `limit` (선택): 페이지당 항목 수 (기본값: 20)

### 세션 상세 조회

```http
GET /api/v1/sessions/{session_id}
```

### 세션 종료

```http
DELETE /api/v1/sessions/{session_id}
```

### 세션 재시작

```http
POST /api/v1/sessions/{session_id}/restart
```

## 📁 워크스페이스 API

### 워크스페이스 생성

```http
POST /api/v1/workspaces
```

**요청 본문:**
```json
{
  "name": "My Project",
  "description": "Project description",
  "project_path": "/home/user/projects/my-project",
  "config": {
    "docker_image": "aicli/claude:latest",
    "environment": {
      "NODE_ENV": "development"
    }
  }
}
```

### 워크스페이스 목록 조회

```http
GET /api/v1/workspaces
```

### 워크스페이스 상세 조회

```http
GET /api/v1/workspaces/{workspace_id}
```

### 워크스페이스 업데이트

```http
PUT /api/v1/workspaces/{workspace_id}
```

### 워크스페이스 삭제

```http
DELETE /api/v1/workspaces/{workspace_id}
```

## 🔌 WebSocket API

### 터미널 스트리밍

```javascript
// WebSocket 연결
const ws = new WebSocket('ws://localhost:8080/api/v1/ws/terminal/{session_id}');

// 인증
ws.onopen = () => {
  ws.send(JSON.stringify({
    type: 'auth',
    token: 'your-jwt-token'
  }));
};

// 메시지 수신
ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  
  switch(message.type) {
    case 'output':
      // 터미널 출력
      console.log(message.data);
      break;
    case 'error':
      // 에러 처리
      console.error(message.error);
      break;
    case 'status':
      // 상태 업데이트
      console.log('Status:', message.status);
      break;
  }
};

// 입력 전송
ws.send(JSON.stringify({
  type: 'input',
  data: 'ls -la\n'
}));

// 터미널 크기 조정
ws.send(JSON.stringify({
  type: 'resize',
  rows: 30,
  cols: 100
}));
```

### WebSocket 메시지 타입

#### 클라이언트 → 서버

| 타입 | 설명 | 데이터 |
|-----|------|--------|
| `auth` | 인증 | `{ token: string }` |
| `input` | 터미널 입력 | `{ data: string }` |
| `resize` | 터미널 크기 조정 | `{ rows: number, cols: number }` |
| `ping` | 연결 유지 | `{}` |

#### 서버 → 클라이언트

| 타입 | 설명 | 데이터 |
|-----|------|--------|
| `output` | 터미널 출력 | `{ data: string }` |
| `error` | 에러 메시지 | `{ error: string, code: string }` |
| `status` | 세션 상태 | `{ status: string }` |
| `pong` | Ping 응답 | `{}` |

## 🐳 Docker 관리 API

### 컨테이너 목록

```http
GET /api/v1/docker/containers
```

### 컨테이너 생성

```http
POST /api/v1/docker/containers
```

**요청 본문:**
```json
{
  "workspace_id": "ws-123",
  "image": "aicli/claude:latest",
  "name": "project-container",
  "volumes": [
    {
      "host": "/home/user/project",
      "container": "/workspace"
    }
  ],
  "environment": {
    "CLAUDE_API_KEY": "sk-ant-..."
  }
}
```

### 컨테이너 시작/중지

```http
POST /api/v1/docker/containers/{container_id}/start
POST /api/v1/docker/containers/{container_id}/stop
```

### 컨테이너 삭제

```http
DELETE /api/v1/docker/containers/{container_id}
```

## 📈 모니터링 API

### 시스템 상태

```http
GET /api/v1/health
```

**응답:**
```json
{
  "status": "healthy",
  "timestamp": "2025-08-10T10:00:00Z",
  "services": {
    "api": "running",
    "docker": "connected",
    "database": "connected"
  },
  "version": "1.0.0"
}
```

### 메트릭

```http
GET /api/v1/metrics
```

**응답 (Prometheus 형식):**
```
# HELP aicli_sessions_active Number of active sessions
# TYPE aicli_sessions_active gauge
aicli_sessions_active 5

# HELP aicli_containers_running Number of running containers
# TYPE aicli_containers_running gauge
aicli_containers_running 3
```

## 🔧 에이전트 API

### 에이전트 생성

```http
POST /api/v1/agents
```

**요청 본문:**
```json
{
  "name": "Code Review Agent",
  "type": "reviewer",
  "config": {
    "language": "go",
    "rules": ["gofmt", "golint"]
  }
}
```

### 에이전트 목록

```http
GET /api/v1/agents
```

### 에이전트 실행

```http
POST /api/v1/agents/{agent_id}/execute
```

**요청 본문:**
```json
{
  "workspace_id": "ws-123",
  "params": {
    "file_path": "/workspace/main.go"
  }
}
```

## 📝 에러 응답 형식

모든 API 에러는 다음 형식으로 반환됩니다:

```json
{
  "success": false,
  "error": {
    "code": "INVALID_REQUEST",
    "message": "요청이 유효하지 않습니다",
    "details": {
      "field": "email",
      "reason": "이메일 형식이 올바르지 않습니다"
    }
  },
  "timestamp": "2025-08-10T10:00:00Z"
}
```

### 에러 코드

| 코드 | HTTP 상태 | 설명 |
|-----|-----------|------|
| `UNAUTHORIZED` | 401 | 인증 실패 |
| `FORBIDDEN` | 403 | 권한 없음 |
| `NOT_FOUND` | 404 | 리소스를 찾을 수 없음 |
| `INVALID_REQUEST` | 400 | 잘못된 요청 |
| `CONFLICT` | 409 | 리소스 충돌 |
| `INTERNAL_ERROR` | 500 | 서버 내부 오류 |
| `SERVICE_UNAVAILABLE` | 503 | 서비스 일시적 불가 |

## 📊 페이지네이션

목록 조회 API는 다음과 같은 페이지네이션을 지원합니다:

**요청:**
```http
GET /api/v1/sessions?page=2&limit=10
```

**응답:**
```json
{
  "success": true,
  "data": [...],
  "pagination": {
    "page": 2,
    "limit": 10,
    "total": 45,
    "total_pages": 5,
    "has_next": true,
    "has_prev": true
  }
}
```

## 🔒 Rate Limiting

API는 다음과 같은 Rate Limit을 적용합니다:

- **기본**: 분당 60회
- **인증된 사용자**: 분당 600회
- **WebSocket**: 연결당 초당 10메시지

Rate Limit 정보는 응답 헤더에 포함됩니다:

```http
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 45
X-RateLimit-Reset: 1691658000
```

## 🧪 API 테스트

### cURL 예제

```bash
# 로그인
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123"}'

# 세션 생성
curl -X POST http://localhost:8080/api/v1/sessions \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"workspace_id":"ws-123","command":"claude"}'

# WebSocket 연결 (wscat 사용)
wscat -c ws://localhost:8080/api/v1/ws/terminal/sess-456 \
  -H "Authorization: Bearer <token>"
```

### Postman Collection

Postman Collection은 [여기](./postman/aicli-web.postman_collection.json)에서 다운로드할 수 있습니다.

---

**API 버전**: 1.0.0 | **최종 업데이트**: 2025-08-10