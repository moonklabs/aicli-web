# 예제 코드 및 튜토리얼

AICode Manager 멀티 에이전트 플랫폼의 다양한 사용 사례와 실제 구현 예제들을 제공합니다.

## 📂 예제 구조

```
examples/
├── README.md                    # 이 파일
├── basic-usage/                 # 기본 사용법
│   ├── create-agent.md         # 에이전트 생성
│   ├── manage-lifecycle.md     # 생명주기 관리
│   └── monitor-agent.md        # 모니터링
├── advanced/                    # 고급 사용법
│   ├── batch-operations.md     # 배치 작업
│   ├── custom-workflows.md     # 커스텀 워크플로우
│   └── performance-tuning.md   # 성능 튜닝
├── integrations/                # 외부 시스템 연동
│   ├── ci-cd-pipeline.md       # CI/CD 파이프라인
│   ├── monitoring-setup.md     # 모니터링 연동
│   └── webhook-integration.md  # 웹훅 연동
├── use-cases/                   # 실제 사용 사례
│   ├── code-review-bot.md      # 코드 리뷰 봇
│   ├── multi-env-testing.md    # 다중 환경 테스트
│   └── feature-development.md  # 기능 개발 지원
├── sdk-examples/                # SDK 사용 예제
│   ├── go-client.go            # Go 클라이언트
│   ├── python-client.py        # Python 클라이언트
│   └── javascript-client.js    # JavaScript 클라이언트
└── troubleshooting/             # 문제 해결 예제
    ├── common-issues.md        # 일반적인 문제
    ├── debugging-guide.md      # 디버깅 가이드
    └── performance-issues.md   # 성능 문제
```

## 🚀 빠른 시작 예제

### 첫 번째 에이전트 생성

가장 기본적인 에이전트 생성과 실행 예제입니다.

```bash
# 1. 에이전트 생성
curl -X POST http://localhost:8080/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-first-agent",
    "project_id": "hello-world",
    "agent_type": "standard",
    "description": "첫 번째 에이전트 테스트"
  }'

# 2. 응답에서 agent_id 확인
export AGENT_ID="agent-abc123"

# 3. 에이전트 시작
curl -X POST http://localhost:8080/api/v1/agents/$AGENT_ID/start

# 4. 상태 확인
curl http://localhost:8080/api/v1/agents/$AGENT_ID/status

# 5. 로그 확인
curl http://localhost:8080/api/v1/agents/$AGENT_ID/logs
```

### WebSocket으로 실시간 로그 확인

```javascript
// JavaScript WebSocket 예제
const ws = new WebSocket('ws://localhost:8080/api/v1/agents/agent-abc123/logs/stream');

ws.onopen = function() {
    console.log('로그 스트림 연결됨');
};

ws.onmessage = function(event) {
    const logEntry = JSON.parse(event.data);
    console.log(`[${logEntry.timestamp}] ${logEntry.level}: ${logEntry.message}`);
};

ws.onerror = function(error) {
    console.error('WebSocket 에러:', error);
};
```

## 📖 상세 예제 목록

### 🔰 기본 사용법

1. **[에이전트 생성](basic-usage/create-agent.md)**
   - 다양한 타입의 에이전트 생성
   - 리소스 제한 설정
   - 환경 변수 구성

2. **[생명주기 관리](basic-usage/manage-lifecycle.md)**
   - 에이전트 시작/중지/재시작
   - 상태 모니터링
   - 에러 처리

3. **[모니터링](basic-usage/monitor-agent.md)**
   - 성능 메트릭 수집
   - 로그 분석
   - 알람 설정

### 🚀 고급 사용법

1. **[배치 작업](advanced/batch-operations.md)**
   - 여러 에이전트 동시 관리
   - 병렬 작업 실행
   - 결과 취합

2. **[커스텀 워크플로우](advanced/custom-workflows.md)**
   - 복잡한 작업 체인 구성
   - 조건부 작업 실행
   - 에러 복구 전략

3. **[성능 튜닝](advanced/performance-tuning.md)**
   - 풀링 최적화
   - 리소스 튜닝
   - 병목점 해결

### 🔗 외부 시스템 연동

1. **[CI/CD 파이프라인](integrations/ci-cd-pipeline.md)**
   - GitHub Actions 연동
   - Jenkins 플러그인
   - 빌드 자동화

2. **[모니터링 연동](integrations/monitoring-setup.md)**
   - Prometheus/Grafana 설정
   - 알람 규칙 구성
   - 대시보드 생성

3. **[웹훅 연동](integrations/webhook-integration.md)**
   - 이벤트 기반 자동화
   - Slack/Teams 알림
   - 외부 시스템 트리거

### 💼 실제 사용 사례

1. **[코드 리뷰 봇](use-cases/code-review-bot.md)**
   - Pull Request 자동 분석
   - 코드 품질 검사
   - 리뷰 코멘트 자동 생성

2. **[다중 환경 테스트](use-cases/multi-env-testing.md)**
   - 개발/스테이징/프로덕션 환경 테스트
   - 환경별 설정 관리
   - 테스트 결과 비교

3. **[기능 개발 지원](use-cases/feature-development.md)**
   - 브랜치별 개발 환경
   - 실시간 개발 피드백
   - 협업 워크플로우

## 🛠️ SDK 사용 예제

### Go 클라이언트

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/aicli/aicli-web/pkg/client"
)

func main() {
    // 클라이언트 생성
    client := client.NewClient("http://localhost:8080")
    
    // 에이전트 생성
    agent, err := client.Agents.Create(context.Background(), &client.CreateAgentRequest{
        Name:        "go-sdk-agent",
        ProjectID:   "go-example",
        AgentType:   "standard",
        Description: "Go SDK 사용 예제",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("에이전트 생성됨: %s\n", agent.ID)
    
    // 에이전트 시작
    err = client.Agents.Start(context.Background(), agent.ID)
    if err != nil {
        log.Fatal(err)
    }
    
    // 상태 확인
    status, err := client.Agents.Status(context.Background(), agent.ID)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("에이전트 상태: %s\n", status.Status)
}
```

### Python 클라이언트

```python
import asyncio
import aiohttp
from aicli_client import AICodeManagerClient

async def main():
    # 클라이언트 생성
    client = AICodeManagerClient("http://localhost:8080")
    
    # 에이전트 생성
    agent = await client.agents.create(
        name="python-sdk-agent",
        project_id="python-example",
        agent_type="standard",
        description="Python SDK 사용 예제"
    )
    
    print(f"에이전트 생성됨: {agent.id}")
    
    # 에이전트 시작
    await client.agents.start(agent.id)
    
    # 실시간 로그 스트리밍
    async for log_entry in client.agents.stream_logs(agent.id):
        print(f"[{log_entry.timestamp}] {log_entry.message}")

if __name__ == "__main__":
    asyncio.run(main())
```

### JavaScript/Node.js 클라이언트

```javascript
const { AICodeManagerClient } = require('@aicli/client');

async function main() {
    // 클라이언트 생성
    const client = new AICodeManagerClient('http://localhost:8080');
    
    try {
        // 에이전트 생성
        const agent = await client.agents.create({
            name: 'nodejs-sdk-agent',
            projectId: 'nodejs-example',
            agentType: 'standard',
            description: 'Node.js SDK 사용 예제'
        });
        
        console.log(`에이전트 생성됨: ${agent.id}`);
        
        // 에이전트 시작
        await client.agents.start(agent.id);
        
        // 상태 모니터링
        const status = await client.agents.status(agent.id);
        console.log(`에이전트 상태: ${status.status}`);
        
        // 이벤트 리스너 설정
        client.agents.on('status-change', (event) => {
            console.log(`상태 변경: ${event.agentId} -> ${event.status}`);
        });
        
    } catch (error) {
        console.error('에러:', error);
    }
}

main();
```

## 🔧 문제 해결 예제

### 일반적인 문제 해결

#### 에이전트 생성 실패
```bash
# 문제: 에이전트 생성 시 리소스 부족 에러
# 해결: 리소스 제한 조정

curl -X POST http://localhost:8080/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "resource-limited-agent",
    "project_id": "test",
    "agent_type": "standard",
    "config": {
      "resources": {
        "cpu": "0.5",
        "memory": "512Mi"
      }
    }
  }'
```

#### 성능 최적화
```bash
# 문제: 에이전트 생성이 느림
# 해결: 풀 워밍업 활성화

curl -X POST http://localhost:8080/api/v1/performance/pools/warmup \
  -H "Content-Type: application/json" \
  -d '{
    "pool_type": "standard",
    "target_size": 10
  }'
```

## 📚 추가 리소스

### 학습 자료

1. **[API 레퍼런스](../api-reference.md)** - 전체 API 문서
2. **[아키텍처 가이드](../architecture.md)** - 시스템 구조 이해
3. **[성능 가이드](../performance.md)** - 성능 최적화 방법
4. **[배포 가이드](../deployment.md)** - 프로덕션 배포
5. **[테스트 가이드](../testing.md)** - 테스트 작성 및 실행

### 커뮤니티

- **GitHub**: [aicli/aicli-web](https://github.com/aicli/aicli-web)
- **Discord**: [AICode Manager 커뮤니티](https://discord.gg/aicli)
- **Stack Overflow**: `aicode-manager` 태그

### 지원

- **문서 이슈**: GitHub Issues에 문서 개선 요청
- **기능 요청**: GitHub Discussions에 새 기능 제안
- **버그 리포트**: GitHub Issues에 버그 신고

---

**다음 단계**: 관심 있는 예제를 선택해서 직접 실행해보세요! 각 예제는 단계별로 자세히 설명되어 있습니다.