# Multi-Agent Platform 시작하기

AICode Manager의 Multi-Agent Platform을 사용하여 AI 에이전트를 생성하고 관리하는 방법을 알아보세요.

## 빠른 시작

### 1. 첫 번째 에이전트 생성하기

```bash
# API를 사용한 에이전트 생성
curl -X POST http://localhost:8080/api/v1/agents \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "name": "my-first-agent",
    "repository_url": "https://github.com/user/repo.git",
    "branch": "main",
    "description": "My first AI agent"
  }'
```

### 2. 에이전트 시작하기

```bash
# 생성된 에이전트 시작
curl -X POST http://localhost:8080/api/v1/agents/{agent-id}/start \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### 3. 에이전트 상태 확인하기

```bash
# 에이전트 상태 조회
curl -X GET http://localhost:8080/api/v1/agents/{agent-id}/status \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

## 주요 개념

### 에이전트 (Agent)
- **정의**: 독립적인 Docker 컨테이너에서 실행되는 AI 어시스턴트
- **구성**: Git 저장소, 작업 환경, Claude CLI
- **목적**: 코드 작성, 리뷰, 분석 등 다양한 개발 작업 수행

### 에이전트 생명주기
1. **생성 (Created)**: 에이전트 메타데이터 생성
2. **시작 (Starting)**: Docker 컨테이너 및 Git worktree 준비
3. **실행 (Running)**: 활성 상태에서 작업 수행 가능
4. **중지 (Stopping)**: 작업 완료 후 정리 과정
5. **종료 (Stopped)**: 리소스 해제 완료

## 사용 예제

### JavaScript/Node.js SDK 사용

```javascript
const { AgentClient } = require('@aicli/agent-sdk');

const client = new AgentClient({
  apiUrl: 'http://localhost:8080/api/v1',
  token: 'YOUR_JWT_TOKEN'
});

// 에이전트 생성
const agent = await client.agents.create({
  name: 'code-reviewer',
  repositoryUrl: 'https://github.com/user/repo.git',
  branch: 'main'
});

// 에이전트 시작
await client.agents.start(agent.id);

// 상태 모니터링
const status = await client.agents.getStatus(agent.id);
console.log(`Agent status: ${status.status}`);

// 로그 스트리밍
const logStream = client.agents.streamLogs(agent.id);
logStream.on('data', (log) => {
  console.log(`[${log.timestamp}] ${log.message}`);
});
```

### Python SDK 사용

```python
from aicli_agent import AgentClient

client = AgentClient(
    api_url='http://localhost:8080/api/v1',
    token='YOUR_JWT_TOKEN'
)

# 에이전트 생성
agent = client.agents.create(
    name='data-analyzer',
    repository_url='https://github.com/user/data-repo.git',
    branch='main'
)

# 에이전트 시작
client.agents.start(agent.id)

# 메트릭 조회
metrics = client.agents.get_metrics(agent.id)
print(f"CPU Usage: {metrics.cpu_usage.current_percent}%")
print(f"Memory Usage: {metrics.memory_usage.current_mb}MB")
```

## 웹 인터페이스 사용

1. **대시보드 접속**: http://localhost:3000/agents
2. **에이전트 생성**: "새 에이전트" 버튼 클릭
3. **저장소 연결**: GitHub 저장소 URL 입력
4. **설정 구성**: 브랜치, 환경 변수 등 설정
5. **시작**: 에이전트 활성화

## 고급 기능

### 배치 작업
```bash
# 여러 에이전트 동시 시작
curl -X POST http://localhost:8080/api/v1/agents/batch/start \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "agent_ids": ["agent-1", "agent-2", "agent-3"]
  }'
```

### 실시간 모니터링
```javascript
// WebSocket을 통한 실시간 이벤트 구독
const ws = new WebSocket('ws://localhost:8080/api/v1/agents/events/stream');

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Agent Event:', data);
};
```

### 헬스체크
```bash
# 에이전트 건강 상태 확인
curl -X GET http://localhost:8080/api/v1/agents/{agent-id}/health \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

## 모범 사례

### 1. 에이전트 명명 규칙
- 목적을 명확히 표현: `code-reviewer`, `test-runner`, `docs-generator`
- 프로젝트별 구분: `project-frontend-agent`, `project-backend-agent`

### 2. 리소스 관리
- 사용하지 않는 에이전트는 즉시 중지
- 정기적인 메트릭 모니터링
- 적절한 리소스 제한 설정

### 3. 보안
- JWT 토큰 안전한 보관
- Git 저장소 접근 권한 최소화
- 민감한 정보는 환경 변수로 관리

## 문제 해결

### 에이전트 시작 실패
```bash
# 에이전트 로그 확인
curl -X GET http://localhost:8080/api/v1/agents/{agent-id}/logs?tail=100
```

### 성능 문제
```bash
# 리소스 사용량 확인
curl -X GET http://localhost:8080/api/v1/agents/{agent-id}/metrics
```

### 연결 문제
- Docker 데몬 상태 확인
- 네트워크 연결 상태 확인  
- Git 저장소 접근 권한 확인

## 다음 단계

- [API 레퍼런스](./api-reference.md) - 전체 API 문서
- [아키텍처 가이드](./architecture.md) - 시스템 구조 이해
- [배포 가이드](./deployment.md) - 프로덕션 배포
- [문제 해결](./troubleshooting.md) - 일반적인 문제 해결

## 도움이 필요하신가요?

- 📚 [전체 문서](../README.md)
- 🎯 [예제 코드](../examples/)
- 🐛 [이슈 리포트](https://github.com/aicli/aicli-web/issues)
- 💬 [커뮤니티 토론](https://github.com/aicli/aicli-web/discussions)