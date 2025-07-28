# 🚀 AICode Manager 빠른 시작 가이드

이 가이드는 인증 없이 핵심 기능을 빠르게 테스트하기 위한 것입니다.

## 1. 준비사항

- Go 1.21+
- Node.js 18+
- Docker
- Claude API Key (환경변수: `ANTHROPIC_API_KEY`)

## 2. 빠른 시작 (3단계)

### Step 1: 백엔드 빌드
```bash
make build
```

### Step 2: API 서버 실행
```bash
# 터미널 1
./start-test-server.sh
```

### Step 3: 웹 UI 실행
```bash
# 터미널 2
./start-test-web.sh
```

이제 http://localhost:5173 에서 웹 UI에 접속할 수 있습니다!

## 3. 기능 테스트

### A. CLI 도구 테스트
```bash
# 버전 확인
./build/aicli version

# 도움말
./build/aicli help

# 워크스페이스 생성
./build/aicli workspace create test-project --path ./test-workspace

# 워크스페이스 목록
./build/aicli workspace list
```

### B. API 직접 테스트
```bash
# Health check
curl http://localhost:8080/health

# 워크스페이스 목록
curl http://localhost:8080/api/v1/workspaces

# 워크스페이스 생성
curl -X POST http://localhost:8080/api/v1/workspaces \
  -H "Content-Type: application/json" \
  -d '{"name":"test","path":"./test-workspace"}'

# 워크스페이스 상태
curl http://localhost:8080/api/v1/workspaces/{id}/status
```

### C. WebSocket 테스트
```javascript
// 브라우저 콘솔에서 실행
const ws = new WebSocket('ws://localhost:8080/ws');
ws.onmessage = (event) => console.log('Message:', event.data);
ws.onopen = () => {
  console.log('Connected!');
  ws.send(JSON.stringify({type: 'ping'}));
};
```

### D. Docker 테스트
```bash
# Docker Compose로 실행
docker-compose -f docker-compose.test.yml up

# 개별 컨테이너 테스트
docker run -it --rm \
  -v $(pwd):/workspace \
  -e CLAUDE_API_KEY=$ANTHROPIC_API_KEY \
  alpine:latest sh
```

## 4. UI 주요 페이지

- **대시보드**: http://localhost:5173/
- **워크스페이스**: http://localhost:5173/workspaces
- **터미널**: http://localhost:5173/terminal
- **Docker 모니터**: http://localhost:5173/docker

## 5. 문제 해결

### API 서버가 시작되지 않음
```bash
# 포트 확인
lsof -i :8080

# 로그 확인
tail -f ./data/aicli.log
```

### 웹 UI가 로드되지 않음
```bash
# 의존성 재설치
cd web && pnpm install

# 캐시 정리
pnpm store prune
```

### Docker 권한 오류
```bash
# Docker 소켓 권한 확인
ls -la /var/run/docker.sock

# 현재 사용자를 docker 그룹에 추가
sudo usermod -aG docker $USER
```

## 6. 정리

```bash
# 서버 종료
# Ctrl+C 또는
pkill aicli-api

# Docker 컨테이너 정리
docker-compose -f docker-compose.test.yml down

# 테스트 데이터 삭제
rm -rf ./data/test.db ./workspaces/test-*
```

## 7. 다음 단계

- 전체 기능 테스트: `make test`
- 프로덕션 빌드: `make build-all`
- 인증 활성화: `.env`에서 `DISABLE_AUTH=false` 설정

Happy Testing! 🎉