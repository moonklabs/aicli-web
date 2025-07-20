# API 명세서

## 🌐 API 개요

AICode Manager API는 RESTful 원칙을 따르며, WebSocket을 통한 실시간 통신을 지원합니다.

### 기본 정보
- **Base URL**: `https://api.aicli.local` (프로덕션)
- **개발 URL**: `http://localhost:8000`
- **인증**: Bearer Token (Supabase JWT)
- **응답 형식**: JSON

### 공통 헤더
```http
Authorization: Bearer <token>
Content-Type: application/json
X-Workspace-ID: <workspace-id>
```

## 🔐 인증 API

### 로그인
```http
POST /api/auth/login
```

**요청 본문:**
```json
{
  "email": "user@example.com",
  "password": "secure_password"
}
```

**응답:**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": "uuid",
    "email": "user@example.com",
    "name": "John Doe",
    "role": "user"
  }
}
```

### 회원가입
```http
POST /api/auth/register
```

**요청 본문:**
```json
{
  "email": "user@example.com",
  "password": "secure_password",
  "name": "John Doe"
}
```

### 토큰 갱신
```http
POST /api/auth/refresh
```

**요청 본문:**
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

### 로그아웃
```http
POST /api/auth/logout
```

## 📁 워크스페이스 API

### 워크스페이스 목록 조회
```http
GET /api/workspaces
```

**쿼리 파라미터:**
- `page` (기본값: 1)
- `limit` (기본값: 10)
- `search` (검색어)
- `status` (active|archived)

**응답:**
```json
{
  "data": [
    {
      "id": "workspace_uuid",
      "name": "My Project",
      "description": "프로젝트 설명",
      "path": "/Users/drumcap/workspace/my-project",
      "status": "active",
      "created_at": "2025-07-20T10:00:00Z",
      "updated_at": "2025-07-20T10:00:00Z",
      "stats": {
        "total_tasks": 15,
        "completed_tasks": 10,
        "running_tasks": 2
      }
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 10,
    "total": 25,
    "pages": 3
  }
}
```

### 워크스페이스 생성
```http
POST /api/workspaces
```

**요청 본문:**
```json
{
  "name": "New Project",
  "description": "새 프로젝트 설명",
  "path": "/Users/drumcap/workspace/new-project",
  "git_url": "https://github.com/user/repo.git",
  "branch": "main"
}
```

### 워크스페이스 상세 조회
```http
GET /api/workspaces/{workspace_id}
```

### 워크스페이스 수정
```http
PUT /api/workspaces/{workspace_id}
```

### 워크스페이스 삭제
```http
DELETE /api/workspaces/{workspace_id}
```

## 🚀 작업(Task) API

### 작업 생성
```http
POST /api/workspaces/{workspace_id}/tasks
```

**요청 본문:**
```json
{
  "type": "claude_chat",
  "prompt": "버그를 수정하고 테스트를 작성해주세요",
  "config": {
    "system_prompt": "당신은 전문 개발자입니다",
    "max_turns": 10,
    "allowed_tools": ["Read", "Write", "Bash", "Edit"],
    "working_directory": "/workspace",
    "environment": {
      "NODE_ENV": "development"
    }
  },
  "metadata": {
    "priority": "high",
    "tags": ["bugfix", "testing"]
  }
}
```

**응답:**
```json
{
  "task_id": "task_uuid",
  "status": "queued",
  "created_at": "2025-07-20T10:00:00Z",
  "estimated_start_time": "2025-07-20T10:01:00Z"
}
```

### 작업 상태 조회
```http
GET /api/tasks/{task_id}
```

**응답:**
```json
{
  "id": "task_uuid",
  "workspace_id": "workspace_uuid",
  "type": "claude_chat",
  "status": "running",
  "progress": {
    "current_turn": 3,
    "max_turns": 10,
    "percentage": 30
  },
  "started_at": "2025-07-20T10:01:00Z",
  "updated_at": "2025-07-20T10:05:00Z",
  "logs": [
    {
      "timestamp": "2025-07-20T10:01:00Z",
      "level": "info",
      "message": "Task started"
    }
  ]
}
```

### 작업 목록 조회
```http
GET /api/workspaces/{workspace_id}/tasks
```

**쿼리 파라미터:**
- `status` (queued|running|completed|failed|cancelled)
- `type` (claude_chat|git_operation|docker_build)
- `start_date` (ISO 8601)
- `end_date` (ISO 8601)

### 작업 취소
```http
POST /api/tasks/{task_id}/cancel
```

### 작업 재시작
```http
POST /api/tasks/{task_id}/restart
```

### 작업 로그 조회
```http
GET /api/tasks/{task_id}/logs
```

**쿼리 파라미터:**
- `level` (debug|info|warning|error)
- `since` (timestamp)
- `limit` (기본값: 100)

## 🔄 실시간 통신 (WebSocket)

### WebSocket 연결
```javascript
const ws = new WebSocket('ws://localhost:8000/ws/{workspace_id}');

ws.onopen = () => {
  // 인증
  ws.send(JSON.stringify({
    type: 'auth',
    token: 'Bearer <token>'
  }));
};
```

### 메시지 타입

#### 1. 명령 실행
```json
{
  "type": "execute",
  "command": {
    "prompt": "파일을 읽어주세요",
    "context": {
      "file_path": "/workspace/main.py"
    }
  }
}
```

#### 2. 실시간 출력
```json
{
  "type": "output",
  "data": {
    "stream": "stdout",
    "content": "파일 내용입니다...",
    "timestamp": "2025-07-20T10:00:00Z"
  }
}
```

#### 3. 상태 업데이트
```json
{
  "type": "status",
  "data": {
    "task_id": "task_uuid",
    "status": "completed",
    "result": {
      "success": true,
      "summary": "작업이 완료되었습니다"
    }
  }
}
```

#### 4. 에러
```json
{
  "type": "error",
  "data": {
    "code": "TASK_FAILED",
    "message": "작업 실행 중 오류가 발생했습니다",
    "details": {
      "reason": "timeout"
    }
  }
}
```

## 📊 모니터링 API

### 시스템 상태
```http
GET /api/health
```

**응답:**
```json
{
  "status": "healthy",
  "timestamp": "2025-07-20T10:00:00Z",
  "services": {
    "database": "healthy",
    "redis": "healthy",
    "docker": "healthy",
    "claude": "healthy"
  }
}
```

### 리소스 사용량
```http
GET /api/metrics
```

**응답:**
```json
{
  "system": {
    "cpu_percent": 45.2,
    "memory_mb": 2048,
    "disk_gb": 50.5
  },
  "workspaces": {
    "active": 5,
    "total_containers": 8,
    "cpu_usage": {
      "workspace_1": 20.5,
      "workspace_2": 15.3
    }
  },
  "tasks": {
    "queued": 3,
    "running": 2,
    "completed_today": 25,
    "failed_today": 2
  }
}
```

### 작업 통계
```http
GET /api/stats/tasks
```

**쿼리 파라미터:**
- `period` (hour|day|week|month)
- `workspace_id` (선택사항)

## 🛠️ 관리자 API

### 사용자 관리
```http
GET /api/admin/users
POST /api/admin/users/{user_id}/suspend
DELETE /api/admin/users/{user_id}
```

### 시스템 설정
```http
GET /api/admin/settings
PUT /api/admin/settings
```

**설정 예시:**
```json
{
  "max_concurrent_tasks": 10,
  "task_timeout_minutes": 30,
  "allowed_workspace_paths": [
    "/home/*/workspace",
    "/opt/projects"
  ],
  "claude_config": {
    "max_turns_default": 10,
    "allowed_tools": ["Read", "Write", "Bash", "Edit"]
  }
}
```

## 📡 Server-Sent Events (SSE)

### 작업 진행상황 스트리밍
```http
GET /api/tasks/{task_id}/stream
```

**응답 형식:**
```
data: {"type": "progress", "percentage": 25, "message": "파일 분석 중..."}

data: {"type": "output", "content": "main.py 파일을 읽었습니다."}

data: {"type": "complete", "result": {"success": true}}
```

**클라이언트 예제:**
```javascript
const eventSource = new EventSource('/api/tasks/task_uuid/stream');

eventSource.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('진행상황:', data);
};

eventSource.onerror = (error) => {
  console.error('SSE 에러:', error);
  eventSource.close();
};
```

## 🔍 에러 응답

### 표준 에러 형식
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "요청 데이터가 유효하지 않습니다",
    "details": {
      "field": "email",
      "reason": "이메일 형식이 올바르지 않습니다"
    }
  },
  "request_id": "req_uuid",
  "timestamp": "2025-07-20T10:00:00Z"
}
```

### 에러 코드
- `400` - Bad Request
- `401` - Unauthorized
- `403` - Forbidden
- `404` - Not Found
- `409` - Conflict
- `429` - Too Many Requests
- `500` - Internal Server Error
- `503` - Service Unavailable

## 📋 API 사용 예제

### Python 클라이언트
```python
import httpx
import asyncio

class AICliClient:
    def __init__(self, base_url: str, token: str):
        self.base_url = base_url
        self.headers = {"Authorization": f"Bearer {token}"}
    
    async def create_task(self, workspace_id: str, prompt: str):
        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{self.base_url}/api/workspaces/{workspace_id}/tasks",
                headers=self.headers,
                json={"type": "claude_chat", "prompt": prompt}
            )
            return response.json()

# 사용 예
client = AICliClient("http://localhost:8000", "your_token")
task = await client.create_task("workspace_id", "코드를 리팩토링해주세요")
```

### JavaScript/TypeScript 클라이언트
```typescript
class AICliClient {
  constructor(
    private baseUrl: string,
    private token: string
  ) {}

  async createTask(workspaceId: string, prompt: string) {
    const response = await fetch(
      `${this.baseUrl}/api/workspaces/${workspaceId}/tasks`,
      {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${this.token}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          type: 'claude_chat',
          prompt
        })
      }
    );
    
    return response.json();
  }
}

// 사용 예
const client = new AICliClient('http://localhost:8000', 'your_token');
const task = await client.createTask('workspace_id', '코드를 리팩토링해주세요');
```

## 🔒 Rate Limiting

API는 다음과 같은 rate limit을 적용합니다:

- **인증된 사용자**: 분당 100개 요청
- **작업 생성**: 시간당 50개
- **WebSocket 연결**: 사용자당 5개 동시 연결

Rate limit 정보는 응답 헤더에 포함됩니다:
```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1627849200
```