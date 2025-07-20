# AICLI 도움말 가이드

AICode Manager CLI(aicli)의 상세 사용 가이드입니다.

## 목차

1. [시작하기](#시작하기)
2. [워크스페이스 관리](#워크스페이스-관리)
3. [태스크 실행](#태스크-실행)
4. [로그 조회](#로그-조회)
5. [설정 관리](#설정-관리)
6. [문제 해결](#문제-해결)

## 시작하기

### 설치 확인

```bash
# 버전 확인
aicli version

# 도움말 보기
aicli help
aicli help [command]
```

### 기본 워크플로우

1. **워크스페이스 생성**
   ```bash
   aicli workspace create --name myproject --path ~/projects/myapp
   ```

2. **태스크 실행**
   ```bash
   aicli task create --workspace myproject --command "implement login feature"
   ```

3. **진행 상황 확인**
   ```bash
   aicli logs --workspace myproject --follow
   ```

## 워크스페이스 관리

워크스페이스는 각 프로젝트를 위한 격리된 환경입니다.

### 워크스페이스 생성

```bash
# 기본 생성
aicli workspace create --name myproject --path /path/to/project

# Claude API 키와 함께 생성
aicli workspace create --name myproject --path /path/to/project --claude-key sk-ant-...

# 환경 변수 사용
export CLAUDE_API_KEY=sk-ant-...
aicli workspace create --name myproject --path /path/to/project
```

### 워크스페이스 목록 조회

```bash
# 기본 테이블 형식
aicli workspace list

# JSON 형식으로 출력
aicli workspace list --output json

# 별칭 사용
aicli ws list
```

### 워크스페이스 정보 확인

```bash
# 상세 정보 조회
aicli workspace info myproject

# JSON 형식으로 출력
aicli workspace info myproject --output json
```

### 워크스페이스 삭제

```bash
# 확인 후 삭제
aicli workspace delete myproject

# 강제 삭제 (확인 없음)
aicli workspace delete myproject --force
```

## 태스크 실행

태스크는 Claude CLI에 전달되는 작업 단위입니다.

### 태스크 생성

```bash
# 명령형 태스크
aicli task create --workspace myproject --command "add user authentication"

# 대화형 모드
aicli task create --workspace myproject --interactive

# 백그라운드 실행
aicli task create -w myproject -c "refactor database layer" --detach

# 복잡한 명령어
aicli task create -w myproject -c "analyze the codebase and suggest performance improvements"
```

### 태스크 목록 조회

```bash
# 실행 중인 태스크만
aicli task list

# 특정 워크스페이스의 태스크
aicli task list --workspace myproject

# 모든 태스크 (완료된 것 포함)
aicli task list --all

# 상태별 필터링
aicli task list --status completed
aicli task list --status running
aicli task list --status failed
```

### 태스크 상태 확인

```bash
# 상태 조회
aicli task status task-001

# JSON 형식으로 출력
aicli task status task-001 --output json
```

### 태스크 취소

```bash
# 정상 취소
aicli task cancel task-001

# 강제 종료
aicli task cancel task-001 --force
```

## 로그 조회

실시간으로 태스크 진행 상황을 모니터링할 수 있습니다.

### 기본 로그 조회

```bash
# 워크스페이스 로그
aicli logs --workspace myproject

# 특정 태스크 로그
aicli logs --task task-001
```

### 실시간 로그 스트리밍

```bash
# 실시간 팔로우
aicli logs -w myproject --follow

# Ctrl+C로 중지
```

### 로그 필터링

```bash
# 최근 10분간의 로그
aicli logs -w myproject --since 10m

# 최근 1시간
aicli logs -w myproject --since 1h

# 마지막 100줄만
aicli logs -t task-001 --tail 100

# 타임스탬프 포함
aicli logs -w myproject --timestamps
```

### 로그 검색

```bash
# grep과 함께 사용
aicli logs -w myproject | grep ERROR

# 특정 패턴 검색
aicli logs -w myproject | grep -i "failed\|error"
```

## 설정 관리

AICLI의 동작을 사용자화할 수 있습니다.

### 설정 조회

```bash
# 모든 설정 표시
aicli config list

# 특정 설정 조회
aicli config get claude.api_key
aicli config get logging.level
```

### 설정 변경

```bash
# 현재 세션만
aicli config set logging.level debug

# 전역 설정 (파일에 저장)
aicli config set claude.model claude-3-opus --global

# 타임아웃 설정
aicli config set api.timeout 30
```

### 주요 설정 항목

| 설정 키 | 설명 | 기본값 |
|---------|------|--------|
| `api.endpoint` | API 서버 주소 | `http://localhost:8080` |
| `api.timeout` | API 타임아웃 (초) | `30` |
| `api.retry_count` | 재시도 횟수 | `3` |
| `claude.api_key` | Claude API 키 | (없음) |
| `claude.model` | Claude 모델 | `claude-3-sonnet` |
| `claude.max_tokens` | 최대 토큰 수 | `4096` |
| `docker.registry` | Docker 레지스트리 | `docker.io` |
| `docker.network` | Docker 네트워크 | `aicli_default` |
| `workspace.default_dir` | 기본 워크스페이스 디렉토리 | `~/.aicli/workspaces` |
| `logging.level` | 로그 레벨 | `info` |
| `logging.format` | 로그 형식 | `text` |

### 환경 변수

모든 설정은 환경 변수로도 지정할 수 있습니다:

```bash
export AICLI_CLAUDE_API_KEY=sk-ant-...
export AICLI_LOGGING_LEVEL=debug
export AICLI_API_ENDPOINT=http://localhost:8080
```

## 문제 해결

### 일반적인 문제

#### 워크스페이스를 찾을 수 없음

```bash
# 문제
Error: 워크스페이스를 찾을 수 없습니다: myproject

# 해결
aicli workspace list  # 존재하는 워크스페이스 확인
aicli workspace create --name myproject --path /path/to/project
```

#### API 연결 실패

```bash
# 문제
Error: API 서버에 연결할 수 없습니다.

# 해결
# 1. API 서버 상태 확인
aicli config get api.endpoint

# 2. 서버 실행 확인
curl http://localhost:8080/health

# 3. 엔드포인트 변경
aicli config set api.endpoint http://localhost:8080 --global
```

#### Claude API 키 오류

```bash
# 문제
Error: Claude API 키가 유효하지 않습니다.

# 해결
# 1. 환경 변수 설정
export CLAUDE_API_KEY=sk-ant-...

# 2. 또는 설정 파일에 저장
aicli config set claude.api_key sk-ant-... --global
```

### 디버깅

```bash
# 상세 로그 활성화
aicli --verbose [command]

# 디버그 레벨 로깅
aicli config set logging.level debug
aicli [command]
```

### 도움 받기

```bash
# 명령어별 도움말
aicli help workspace
aicli help task
aicli help logs

# 온라인 문서
https://github.com/drumcap/aicli-web/docs

# 이슈 리포트
https://github.com/drumcap/aicli-web/issues
```

## 고급 사용법

### 스크립팅

```bash
#!/bin/bash
# 여러 프로젝트 일괄 처리

projects=("project1" "project2" "project3")

for project in "${projects[@]}"; do
    echo "Processing $project..."
    aicli task create -w $project -c "update dependencies" --detach
done

# 모든 태스크 상태 확인
aicli task list --all --output json | jq '.'
```

### 파이프라인 통합

```bash
# CI/CD 파이프라인에서 사용
aicli task create \
    --workspace ci-project \
    --command "analyze code quality and security" \
    --detach

# 태스크 완료 대기
task_id=$(aicli task list -w ci-project --output json | jq -r '.[0].id')
while [ "$(aicli task status $task_id --output json | jq -r '.status')" == "running" ]; do
    sleep 5
done
```

### 별칭 설정

```bash
# ~/.bashrc 또는 ~/.zshrc에 추가
alias aic='aicli'
alias aicw='aicli workspace'
alias aict='aicli task'
alias aicl='aicli logs --follow'
```