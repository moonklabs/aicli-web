# AICode Manager Agent Docker Image

Claude CLI가 포함된 에이전트 실행 환경을 제공하는 Docker 이미지입니다.

## 개요

이 Docker 이미지는 AICode Manager의 에이전트가 실행되는 격리된 환경을 제공합니다. 각 에이전트는 독립된 컨테이너에서 실행되며, Claude CLI를 통해 AI 기반 코드 작업을 수행합니다.

## 주요 기능

- Ubuntu 22.04 기반
- Claude CLI 사전 설치
- Git, Python, Node.js 등 개발 도구 포함
- 프로젝트 워크스페이스 마운트 지원
- 사용자 권한 관리 (non-root 실행)
- 환경 변수를 통한 설정 관리

## 빌드 방법

### Make 사용 (권장)

```bash
# 이미지 빌드
make build

# 개발 환경 실행
make dev

# 테스트 실행
make test
```

### Docker 명령어 직접 사용

```bash
# 이미지 빌드
docker build -t aicli-agent:latest .

# 컨테이너 실행
docker run -it --rm \
  -v /path/to/project:/workspace \
  -e CLAUDE_API_KEY=your-api-key \
  aicli-agent:latest
```

## 환경 변수

| 변수명 | 설명 | 필수 | 기본값 |
|--------|------|------|--------|
| `CLAUDE_API_KEY` | Claude API 키 | Yes | - |
| `GIT_USER_NAME` | Git 사용자 이름 | No | AICode Agent |
| `GIT_USER_EMAIL` | Git 사용자 이메일 | No | agent@aicode.local |
| `TZ` | 시간대 설정 | No | Asia/Seoul |
| `AGENT_ID` | 에이전트 ID | No | - |
| `PROJECT_ID` | 프로젝트 ID | No | - |

## 볼륨 마운트

- `/workspace`: 프로젝트 작업 디렉토리
- `/home/agent/.claude`: Claude CLI 설정 및 히스토리

## 사용 예시

### 기본 실행

```bash
docker run -it --rm \
  -v $(pwd):/workspace \
  -e CLAUDE_API_KEY=$CLAUDE_API_KEY \
  aicli-agent:latest
```

### Git 설정과 함께 실행

```bash
docker run -it --rm \
  -v $(pwd):/workspace \
  -v ~/.gitconfig:/home/agent/.gitconfig:ro \
  -e CLAUDE_API_KEY=$CLAUDE_API_KEY \
  -e GIT_USER_NAME="Your Name" \
  -e GIT_USER_EMAIL="your.email@example.com" \
  aicli-agent:latest
```

### Docker Compose 사용

```yaml
version: '3.8'
services:
  agent:
    image: aicli-agent:latest
    environment:
      - CLAUDE_API_KEY=${CLAUDE_API_KEY}
    volumes:
      - ./my-project:/workspace
      - agent-data:/home/agent/.claude
    stdin_open: true
    tty: true

volumes:
  agent-data:
```

## 개발 환경

### 로컬 개발

1. 코드 수정
2. `make build`로 이미지 재빌드
3. `make dev`로 개발 환경 실행
4. `make logs`로 로그 확인

### 테스트

```bash
# 단위 테스트
make test

# 보안 스캔
make scan

# 이미지 크기 확인
make size
```

## 보안 고려사항

- 컨테이너는 non-root 사용자(`agent`)로 실행됩니다
- API 키는 환경 변수로만 전달하고 이미지에 포함하지 마세요
- 민감한 파일은 읽기 전용으로 마운트하세요
- 네트워크는 격리된 브리지 네트워크를 사용합니다

## 문제 해결

### Claude CLI가 실행되지 않는 경우

```bash
# 컨테이너 내부에서 확인
docker run -it --rm aicli-agent:latest bash
$ which claude
$ claude --version
```

### 권한 문제

```bash
# 호스트 UID/GID 확인
id -u
id -g

# 동일한 UID로 컨테이너 실행
docker run -it --rm \
  --user $(id -u):$(id -g) \
  -v $(pwd):/workspace \
  aicli-agent:latest
```

### 네트워크 연결 문제

```bash
# 프록시 설정이 필요한 경우
docker build \
  --build-arg HTTP_PROXY=$HTTP_PROXY \
  --build-arg HTTPS_PROXY=$HTTPS_PROXY \
  -t aicli-agent:latest .
```

## 라이선스

이 프로젝트는 AICode Manager의 일부로 제공됩니다.