# 빠른 시작 가이드

AICode Manager 멀티 에이전트 플랫폼을 5분 내에 실행해보세요! 🚀

## 📋 전제 조건

### 시스템 요구사항
- **운영체제**: Linux, macOS, Windows (WSL2)
- **Docker**: 20.10+ 설치 및 실행 중
- **Git**: 2.28+ (worktree 지원)
- **메모리**: 최소 4GB RAM 사용 가능
- **디스크**: 최소 10GB 여유 공간

### 사전 확인
```bash
# Docker 설치 확인
docker --version
# Docker Compose 설치 확인  
docker-compose --version
# Git 설치 확인
git --version
```

## 🚀 1단계: 서버 설치 및 실행

### 방법 1: 바이너리 다운로드 (추천)
```bash
# 최신 릴리즈 다운로드
curl -L https://github.com/aicli/aicli-web/releases/latest/download/aicli-api-linux-amd64 -o aicli-api
chmod +x aicli-api

# 서버 시작
./aicli-api serve --port 8080
```

### 방법 2: 소스 코드 빌드
```bash
# 저장소 클론
git clone https://github.com/aicli/aicli-web.git
cd aicli-web

# 의존성 설치 및 빌드
make deps
make build

# 서버 시작
./build/aicli-api serve --port 8080
```

### 방법 3: Docker Compose (개발용)
```bash
# 개발 환경 시작
docker-compose -f docker-compose.test.yml up -d

# 로그 확인
docker-compose logs -f api
```

### 서버 실행 확인
```bash
# 헬스체크
curl http://localhost:8080/health

# 예상 응답
{
  "status": "healthy",
  "timestamp": "2025-07-30T23:50:00Z"
}
```

## 🎯 2단계: 첫 번째 에이전트 생성

### 에이전트 생성
```bash
curl -X POST http://localhost:8080/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-first-agent",
    "project_id": "quickstart-demo",
    "agent_type": "standard", 
    "description": "My first AI agent for testing"
  }'
```

### 성공 응답 예시
```json
{
  "id": "agent-abc123",
  "name": "my-first-agent", 
  "project_id": "quickstart-demo",
  "agent_type": "standard",
  "status": "created",
  "description": "My first AI agent for testing",
  "created_at": "2025-07-30T23:51:00Z"
}
```

**📝 에이전트 ID 저장**: 응답에서 `id` 값을 복사해두세요. 다음 단계에서 사용합니다.

## ▶️ 3단계: 에이전트 시작

```bash
# 에이전트 시작 (위에서 받은 ID 사용)
export AGENT_ID="agent-abc123"  # 실제 ID로 변경
curl -X POST http://localhost:8080/api/v1/agents/$AGENT_ID/start
```

### 성공 응답
```json
{
  "id": "agent-abc123",
  "status": "starting",
  "message": "Agent start initiated",
  "started_at": "2025-07-30T23:52:00Z"
}
```

## 📊 4단계: 에이전트 상태 확인

### 상태 조회
```bash
curl http://localhost:8080/api/v1/agents/$AGENT_ID/status
```

### 응답 예시 (시작 완료 시)
```json
{
  "agent_id": "agent-abc123",
  "status": "running",
  "container_status": "running", 
  "health_status": "healthy",
  "uptime_seconds": 120,
  "last_seen": "2025-07-30T23:54:00Z",
  "resource_usage": {
    "cpu_percent": 5.2,
    "memory_usage": "128MB",
    "memory_percent": 12.5
  }
}
```

## 📜 5단계: 로그 확인

### 로그 조회
```bash
curl http://localhost:8080/api/v1/agents/$AGENT_ID/logs
```

### 실시간 로그 스트리밍 (선택사항)
```bash
# WebSocket 연결 (wscat 필요: npm install -g wscat)
wscat -c ws://localhost:8080/api/v1/agents/$AGENT_ID/logs/stream
```

## 🛑 6단계: 에이전트 정리

### 에이전트 중지
```bash
curl -X POST http://localhost:8080/api/v1/agents/$AGENT_ID/stop
```

### 에이전트 삭제
```bash
curl -X DELETE http://localhost:8080/api/v1/agents/$AGENT_ID
```

## 🎉 축하합니다!

첫 번째 에이전트를 성공적으로 실행했습니다! 이제 다음과 같은 기능들을 탐색해보세요:

## 🔄 다음 단계

### 1. 웹 UI 사용 (선택사항)
```bash
# 웹 인터페이스 접속
open http://localhost:8080
```

### 2. 여러 에이전트 동시 실행
```bash
# 두 번째 에이전트 생성
curl -X POST http://localhost:8080/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "agent-2",
    "project_id": "multi-agent-demo",
    "agent_type": "standard"
  }'

# 세 번째 에이전트 생성
curl -X POST http://localhost:8080/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "agent-3", 
    "project_id": "multi-agent-demo",
    "agent_type": "standard"
  }'

# 모든 에이전트 조회
curl http://localhost:8080/api/v1/agents
```

### 3. 리소스 제한 설정
```bash
# 제한된 리소스로 에이전트 생성
curl -X POST http://localhost:8080/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "limited-agent",
    "project_id": "resource-demo",
    "agent_type": "standard",
    "config": {
      "resources": {
        "cpu": "0.5",
        "memory": "512Mi"
      }
    }
  }'
```

### 4. 환경 변수 설정
```bash
# 환경 변수와 함께 에이전트 생성
curl -X POST http://localhost:8080/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "env-agent",
    "project_id": "env-demo", 
    "agent_type": "standard",
    "config": {
      "environment": {
        "NODE_ENV": "production",
        "DEBUG": "true",
        "APP_VERSION": "1.0.0"
      }
    }
  }'
```

## 🧪 실전 예제

### CI/CD 파이프라인 시뮬레이션
```bash
#!/bin/bash
# build-pipeline.sh

# 빌드 에이전트 생성
BUILD_AGENT=$(curl -s -X POST http://localhost:8080/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "build-'$(date +%s)'",
    "project_id": "ci-cd-demo",
    "agent_type": "standard",
    "description": "CI/CD Build Agent"
  }' | jq -r '.id')

echo "Build Agent Created: $BUILD_AGENT"

# 에이전트 시작
curl -X POST http://localhost:8080/api/v1/agents/$BUILD_AGENT/start

# 상태 확인 (실행될 때까지 대기)
while true; do
  STATUS=$(curl -s http://localhost:8080/api/v1/agents/$BUILD_AGENT/status | jq -r '.status')
  echo "Agent Status: $STATUS"
  
  if [ "$STATUS" = "running" ]; then
    echo "Agent is ready!"
    break
  elif [ "$STATUS" = "failed" ]; then
    echo "Agent failed to start"
    exit 1
  fi
  
  sleep 2
done

# 빌드 완료 후 정리
echo "Build completed. Cleaning up..."
curl -X POST http://localhost:8080/api/v1/agents/$BUILD_AGENT/stop
curl -X DELETE http://localhost:8080/api/v1/agents/$BUILD_AGENT

echo "Pipeline completed successfully!"
```

### 실행 방법
```bash
chmod +x build-pipeline.sh
./build-pipeline.sh
```

## 🚨 문제 해결

### 서버가 시작되지 않는 경우
```bash
# 포트 사용 확인
lsof -i :8080

# Docker 상태 확인
docker ps
docker system df

# 로그 확인
./aicli-api serve --port 8080 --log-level debug
```

### 에이전트 생성 실패
```bash
# Docker 이미지 확인
docker images | grep aicli

# 시스템 리소스 확인
free -h
df -h

# 에러 로그 확인
curl http://localhost:8080/api/v1/agents/$AGENT_ID/logs
```

### 메모리 부족 에러
```bash
# 메모리 사용량 확인
docker stats

# 사용하지 않는 컨테이너 정리
docker container prune
docker image prune
```

## 📚 추가 학습 자료

### 핵심 문서
- **[API 레퍼런스](api-reference.md)** - 전체 API 문서
- **[아키텍처 가이드](architecture.md)** - 시스템 구조 이해
- **[성능 가이드](performance.md)** - 성능 최적화 방법

### 고급 주제
- **[배포 가이드](deployment.md)** - 프로덕션 배포
- **[보안 가이드](security.md)** - 보안 설정
- **[모니터링 가이드](monitoring.md)** - 운영 모니터링

### 개발자 도구
- **[SDK 문서](developer-guide.md)** - Go/Python/Node.js SDK
- **[예제 코드](examples/)** - 실제 사용 예제
- **[문제 해결](troubleshooting.md)** - 일반적인 문제 해결

## 💡 팁 & 베스트 프랙티스

### 🏷️ 명명 규칙
```bash
# 좋은 에이전트 이름 예시
"build-main-$(date +%Y%m%d)"
"test-pr-123"
"deploy-staging-v1.2.3"

# 피해야 할 이름
"agent1", "test", "temp"
```

### ⚡ 성능 최적화
1. **적절한 리소스 할당**: 작업에 맞는 CPU/메모리 설정
2. **에이전트 재사용**: 가능한 경우 기존 에이전트 재사용
3. **정리 습관**: 작업 완료 후 즉시 에이전트 삭제

### 🔒 보안 고려사항
1. **최소 권한 원칙**: 필요한 최소 권한만 부여
2. **네트워크 격리**: 기본 제공되는 네트워크 격리 활용
3. **로그 관리**: 민감한 정보 로그에 출력 금지

---

**🎯 완료!** 이제 멀티 에이전트 플랫폼을 사용할 준비가 되었습니다. 

**다음**: [API 레퍼런스](api-reference.md)에서 모든 API 기능을 탐색해보세요!