# Docker 개발 환경 가이드

이 문서는 AICode Manager 프로젝트의 Docker 기반 개발 환경 설정 및 사용법을 설명합니다.

## 목차

1. [사전 요구사항](#사전-요구사항)
2. [빠른 시작](#빠른-시작)
3. [개발 환경 구성](#개발-환경-구성)
4. [사용 가능한 명령어](#사용-가능한-명령어)
5. [개발 워크플로우](#개발-워크플로우)
6. [디버깅](#디버깅)
7. [문제 해결](#문제-해결)

## 사전 요구사항

- Docker 20.10 이상
- Docker Compose 2.0 이상
- Git
- (선택사항) VS Code with Remote-Containers 확장

## 빠른 시작

```bash
# 1. 환경 변수 설정
cp .env.example .env
# .env 파일을 편집하여 필요한 값 설정

# 2. Docker 개발 환경 시작
make docker-dev

# 3. API 서버 로그 확인
make docker-dev-logs

# 4. 개발 환경 종료
make docker-dev-down
```

## 개발 환경 구성

### Docker 이미지 구성

개발 환경은 다음 컴포넌트로 구성됩니다:

- **Base Image**: `golang:1.21-alpine`
- **개발 도구**: air (hot reload), delve (디버거), golangci-lint
- **서비스**:
  - `aicli-dev`: CLI 개발 환경
  - `api-dev`: API 서버 개발 환경
  - `workspace-prep`: 초기화 서비스

### 볼륨 마운트

- **소스 코드**: 로컬 디렉토리가 `/workspace`로 마운트
- **Go 모듈 캐시**: 재사용을 위한 별도 볼륨
- **빌드 캐시**: 컴파일 속도 향상을 위한 캐시

### 포트 설정

- **8080**: API 서버
- **2345**: Delve 디버거
- **6060**: pprof 프로파일러

## 사용 가능한 명령어

### 기본 명령어

```bash
# Docker 개발 환경 시작
make docker-dev

# API 서버만 실행 (hot reload 포함)
make docker-dev-api

# CLI 도구 실행
make docker-dev-cli

# 테스트 실행
make docker-dev-test

# 린팅 실행
make docker-dev-lint

# 개발 환경 종료
make docker-dev-down
```

### 고급 명령어

```bash
# 디버그 모드로 API 서버 실행
make docker-dev-debug

# 실시간 로그 확인
make docker-dev-logs

# 특정 서비스의 로그만 확인
docker-compose logs -f api-dev

# 컨테이너 내부 접속
docker-compose exec api-dev /bin/bash

# 환경 변수 확인
docker-compose config
```

## 개발 워크플로우

### 1. API 서버 개발

```bash
# API 서버 시작 (hot reload 활성화)
make docker-dev-api

# 다른 터미널에서 코드 수정
# 파일 저장 시 자동으로 재컴파일 및 재시작

# API 테스트
curl http://localhost:8080/api/v1/health
```

### 2. CLI 도구 개발

```bash
# CLI 개발 컨테이너 시작
make docker-dev-cli

# 컨테이너 내부에서 CLI 명령어 테스트
./tmp/aicli --help
./tmp/aicli workspace list
```

### 3. 테스트 주도 개발

```bash
# 테스트 파일 수정 후
make docker-dev-test

# 특정 패키지만 테스트
docker-compose run --rm test go test -v ./internal/cli/...

# 커버리지 확인
docker-compose run --rm test make test-coverage
```

## 디버깅

### VS Code에서 원격 디버깅

1. API 서버를 디버그 모드로 시작:
   ```bash
   make docker-dev-debug
   ```

2. VS Code에서 디버깅 구성 추가 (`.vscode/launch.json`):
   ```json
   {
     "version": "0.2.0",
     "configurations": [
       {
         "name": "Attach to Docker",
         "type": "go",
         "request": "attach",
         "mode": "remote",
         "remotePath": "/workspace",
         "port": 2345,
         "host": "localhost",
         "trace": "verbose"
       }
     ]
   }
   ```

3. 브레이크포인트 설정 후 디버깅 시작

### 로그 레벨 조정

```bash
# 환경 변수로 로그 레벨 설정
docker-compose run -e LOG_LEVEL=trace api-dev

# .env 파일에서 영구 설정
LOG_LEVEL=debug
```

## Hot Reload 설정

### Air 설정 파일

- **`.air.cli.toml`**: CLI 도구용 설정
- **`.air.api.toml`**: API 서버용 설정
- **`.air.debug.toml`**: 디버그 모드용 설정

### 파일 감시 패턴 수정

```toml
# .air.api.toml 예시
[build]
  # 감시할 디렉토리
  include_dir = ["cmd/api", "internal", "pkg"]
  # 감시할 확장자
  include_ext = ["go", "tpl", "tmpl", "html"]
  # 제외할 디렉토리
  exclude_dir = ["tmp", "vendor", ".git"]
```

## 문제 해결

### 포트 충돌

```bash
# 사용 중인 포트 확인
lsof -i :8080
lsof -i :2345

# 다른 포트로 변경 (.env 파일)
API_PORT=8081
DEBUG_PORT=2346
```

### 권한 문제

```bash
# Docker 소켓 권한 문제
sudo chmod 666 /var/run/docker.sock

# 또는 사용자를 docker 그룹에 추가
sudo usermod -aG docker $USER
```

### 느린 빌드 속도

```bash
# 볼륨 정리
docker volume prune

# 빌드 캐시 정리
docker builder prune

# 특정 볼륨만 정리
docker volume rm aicli-web_go-mod-cache
```

### 메모리 부족

```bash
# Docker Desktop 메모리 할당 증가
# Settings > Resources > Memory

# 또는 docker-compose.yml에서 제한 설정
services:
  api-dev:
    mem_limit: 2g
```

## 성능 최적화 팁

1. **파일 시스템 성능** (macOS):
   ```yaml
   volumes:
     - .:/workspace:cached  # 읽기 성능 향상
   ```

2. **Go 모듈 캐시 활용**:
   - 별도 볼륨으로 관리하여 재다운로드 방지
   - 정기적으로 `go mod tidy` 실행

3. **불필요한 재빌드 방지**:
   - `.dockerignore` 파일 최적화
   - Air 설정에서 불필요한 파일 제외

## 다음 단계

- [VS Code 개발 환경 설정](./vscode-setup.md)
- [CI/CD 파이프라인 구성](./ci-cd-setup.md)
- [프로덕션 배포 가이드](../deployment.md)