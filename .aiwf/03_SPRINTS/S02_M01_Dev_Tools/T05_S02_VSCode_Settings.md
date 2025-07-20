---
task_id: T05_S02
sprint_sequence_id: S02
status: completed
complexity: Low
estimated_hours: 3
actual_hours: 2
assigned_to: Claude
created_date: 2025-07-20
completed_date: 2025-07-21
last_updated: 2025-07-21T03:30:00Z
---

# Task: VS Code 개발 환경 설정 확장 및 최적화

## Description
기존에 구성된 VS Code 설정을 확장하여 팀 개발 생산성을 극대화합니다. Go 개발에 최적화된 확장 프로그램, 디버깅 설정, 태스크 실행, 코드 스니펫 등을 추가로 구성하여 완전한 IDE 환경을 제공합니다.

## Goal / Objectives
- 기존 VS Code 설정 파일 확장 및 최적화
- Go 개발 특화 확장 프로그램 추가 구성
- 프로젝트별 디버깅 설정 고도화
- 빌드/테스트 태스크 자동화 설정
- 코드 스니펫 및 템플릿 제공
- 팀 간 일관된 개발 환경 보장

## Acceptance Criteria
- [ ] 기존 .vscode/settings.json 확장 및 최적화
- [ ] .vscode/extensions.json에 필수 확장 프로그램 추가
- [ ] .vscode/launch.json 디버깅 구성 고도화
- [ ] .vscode/tasks.json 빌드/테스트 태스크 확장
- [ ] .vscode/snippets/ Go 코드 스니펫 추가
- [ ] 워크스페이스 설정 최적화
- [ ] 팀 사용 가이드 문서 작성
- [ ] 설정 동기화 방법 안내

## Subtasks
- [ ] 기존 .vscode 설정 파일 분석 및 개선점 파악
- [ ] Go 개발 필수 확장 프로그램 선별
- [ ] 디버깅 시나리오별 launch.json 구성
- [ ] Makefile 타겟 연동 tasks.json 작성
- [ ] 프로젝트 특화 코드 스니펫 개발
- [ ] 설정 파일 주석 및 문서화
- [ ] VS Code 워크스페이스 파일 생성
- [ ] 팀 온보딩 가이드 작성

## Technical Guide

### 기존 VS Code 설정 확장

#### 기존 설정 파일 분석
현재 구성된 VS Code 설정:
- **.vscode/settings.json**: golangci-lint 통합, 기본 Go 설정
- **.vscode/extensions.json**: 권장 확장 프로그램 목록
- **.vscode/launch.json**: 디버깅 설정
- **.vscode/tasks.json**: 빌드/테스트 태스크

#### settings.json 확장 필요 영역
기존 설정을 기반으로 다음 영역 확장:
1. **코드 포맷팅**: goimports 설정 최적화
2. **린팅**: golangci-lint 세부 설정
3. **테스트**: 테스트 실행 최적화
4. **파일 관리**: 자동 저장, 제외 패턴
5. **터미널**: 기본 셸 및 설정

### 확장 프로그램 구성

#### 필수 Go 개발 확장
.vscode/extensions.json 확장:
```json
{
  "recommendations": [
    "golang.go",                    // Go 언어 지원
    "ms-vscode.vscode-go",         // Go 팀 공식 확장
    "github.copilot",              // AI 코드 어시스턴트
    "ms-python.python",            // Python (pre-commit용)
    "redhat.vscode-yaml",          // YAML 지원
    "ms-vscode.makefile-tools",    // Makefile 지원
    "ms-vscode-remote.remote-containers", // Docker 개발
    "eamodio.gitlens",             // Git 확장
    "ms-vscode.test-adapter-converter", // 테스트 실행기
    "humao.rest-client"            // API 테스트
  ]
}
```

#### 선택적 유용한 확장
개발 생산성 향상을 위한 추가 확장:
- **Error Lens**: 인라인 에러 표시
- **Todo Tree**: TODO 주석 관리
- **Better Comments**: 주석 하이라이팅
- **Bracket Pair Colorizer**: 괄호 매칭

### 디버깅 설정 고도화

#### launch.json 확장
기존 디버깅 설정 확장:
```json
{
  "configurations": [
    {
      "name": "Launch CLI",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/cmd/aicli",
      "args": ["--help"],
      "env": {
        "GO_ENV": "development"
      }
    },
    {
      "name": "Launch API Server", 
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/cmd/api",
      "env": {
        "PORT": "8080",
        "GO_ENV": "development"
      }
    },
    {
      "name": "Attach to Docker",
      "type": "go", 
      "request": "attach",
      "mode": "remote",
      "remotePath": "/workspace",
      "port": 2345,
      "host": "127.0.0.1"
    }
  ]
}
```

#### 디버깅 시나리오
1. **로컬 CLI 디버깅**: 명령행 인자 테스트
2. **API 서버 디버깅**: HTTP 요청 처리
3. **원격 디버깅**: Docker 컨테이너 연결
4. **테스트 디버깅**: 단위 테스트 디버깅

### 태스크 자동화

#### tasks.json 확장
기존 Makefile 타겟과 연동:
```json
{
  "tasks": [
    {
      "label": "Build All",
      "type": "shell",
      "command": "make build",
      "group": {
        "kind": "build",
        "isDefault": true
      }
    },
    {
      "label": "Run Tests",
      "type": "shell", 
      "command": "make test",
      "group": "test"
    },
    {
      "label": "Lint Code",
      "type": "shell",
      "command": "make lint",
      "group": "test"
    },
    {
      "label": "Start Dev Environment",
      "type": "shell",
      "command": "make docker-dev",
      "group": "build"
    }
  ]
}
```

#### 태스크 바인딩
- **F5**: 디버그 실행
- **Ctrl+Shift+P**: 태스크 실행
- **Ctrl+Shift+B**: 빌드 실행
- **Ctrl+Shift+T**: 테스트 실행

### 코드 스니펫 개발

#### Go 코드 스니펫
.vscode/snippets/go.json 생성:
```json
{
  "Go Test Function": {
    "prefix": "gotest",
    "body": [
      "func Test${1:FunctionName}(t *testing.T) {",
      "\t// ${2:테스트 설명}",
      "\t${3:// 테스트 코드}",
      "}"
    ]
  },
  "HTTP Handler": {
    "prefix": "gohandler", 
    "body": [
      "func ${1:handlerName}(c *gin.Context) {",
      "\t// ${2:핸들러 설명}",
      "\tc.JSON(http.StatusOK, gin.H{",
      "\t\t\"message\": \"${3:success}\",",
      "\t})",
      "}"
    ]
  }
}
```

#### 템플릿 스니펫
프로젝트 특화 스니펫:
- **테스트 함수**: 테이블 드리븐 테스트
- **HTTP 핸들러**: Gin 핸들러 템플릿
- **CLI 명령어**: Cobra 명령어 템플릿
- **에러 처리**: 표준 에러 처리 패턴

### 워크스페이스 설정

#### .vscode/workspace.json
프로젝트 워크스페이스 설정:
```json
{
  "folders": [
    {
      "name": "aicli-web",
      "path": "."
    }
  ],
  "settings": {
    "go.testTimeout": "30s",
    "go.buildOnSave": "package",
    "go.lintOnSave": "package"
  }
}
```

### 기존 설정과의 통합

#### 기존 파일 패턴 유지
- **한글 주석**: 기존 프로젝트의 한글 주석 패턴 유지
- **파일 구조**: 기존 디렉토리 구조 존중
- **Makefile 연동**: 기존 Makefile 타겟 최대한 활용

#### 설정 충돌 방지
- **글로벌 vs 프로젝트**: 프로젝트 설정 우선
- **확장 충돌**: 불필요한 확장 비활성화
- **성능 최적화**: 파일 인덱싱 최적화

### 팀 협업 최적화

#### 설정 동기화
- **Settings Sync**: VS Code 내장 동기화 활용
- **프로젝트 설정**: .vscode 폴더 Git 포함
- **개인 설정**: 개인별 커스터마이징 가이드

#### 온보딩 지원
- **README 섹션**: VS Code 설정 가이드 추가
- **확장 설치**: 자동 설치 스크립트
- **문제 해결**: FAQ 및 트러블슈팅

## Implementation Notes
- 기존 VS Code 설정을 완전히 대체하지 않고 점진적 확장
- 팀원별 선호도를 고려한 선택적 설정 제공
- 성능 영향을 최소화하는 확장 프로그램 선별
- 다른 IDE 사용자도 고려한 범용적 설정
- 정기적인 설정 업데이트 및 개선

## Output Log

### 2025-07-21 작업 완료

#### 수행된 작업
1. **settings.json 확장 완료**
   - Go 개발 설정 고도화 (커버리지 데코레이터, Delve 디버거 설정)
   - 에디터 설정 확장 (브래킷 페어링, 미니맵, 자동 저장)
   - 파일 관리 설정 추가 (자동 저장, 공백 처리, 파일 연관)
   - 터미널 환경 설정 (플랫폼별 기본 셸, 환경 변수)
   - 검색 및 Git 설정 최적화
   - 언어별 특화 설정 추가

2. **extensions.json 대폭 확장**
   - 필수 Go 개발 확장 프로그램 추가
   - 코드 품질 도구 (ErrorLens, TODO Tree, Better Comments)
   - Docker 및 컨테이너 개발 도구
   - Git 고급 기능 (GitLens, Git Graph)
   - API 개발 도구 (REST Client, OpenAPI)
   - 마크다운 및 문서화 도구
   - 권장하지 않는 확장 목록 추가 (충돌 방지)

3. **launch.json 고도화**
   - CLI 디버깅 구성 확장 (동적 인자 입력 지원)
   - API 서버 디버깅 (개발/프로덕션 모드)
   - Docker 원격 디버깅 구성 추가
   - 테스트 디버깅 고도화 (특정 테스트, 벤치마크)
   - 입력 변수 정의로 동적 디버깅 지원
   - 복합 실행 구성 (API + CLI 동시 디버깅)

4. **tasks.json 대폭 확장**
   - 빌드 태스크 세분화 (전체, CLI, API 개별 빌드)
   - 테스트 태스크 확장 (verbose, integration, coverage, benchmark)
   - 개발 환경 태스크 (watch 모드, 자동 재시작)
   - Docker 관련 태스크 추가
   - 유틸리티 태스크 (mod 관리, pre-commit)
   - 문서화 태스크 추가

5. **Go 코드 스니펫 생성**
   - 테스트 관련 스니펫 (테이블 드리븐, 벤치마크, 서브테스트)
   - HTTP 핸들러 스니펫 (Gin 핸들러, 미들웨어)
   - CLI 스니펫 (Cobra 명령어)
   - 에러 처리 패턴
   - 구조체 및 인터페이스 템플릿
   - 동시성 패턴 (고루틴, 채널, WaitGroup)
   - 컨텍스트 및 로깅 스니펫
   - 데이터베이스 쿼리 템플릿

6. **워크스페이스 파일 생성**
   - 프로젝트 전용 워크스페이스 설정
   - 파일 탐색기 네스팅 패턴 정의
   - 워크스페이스 레벨 디버그 및 태스크 구성

7. **VS Code 사용 가이드 문서 작성**
   - 설정 파일 구조 설명
   - 확장 프로그램 설치 가이드
   - 개발 워크플로우 안내
   - 디버깅 시나리오별 가이드
   - 태스크 실행 방법
   - 코드 스니펫 사용법
   - 팁과 트릭 정리
   - 문제 해결 가이드

#### 생성/수정된 파일
- `.vscode/settings.json` - 186줄로 확장 (기존 52줄)
- `.vscode/extensions.json` - 67줄로 확장 (기존 13줄)
- `.vscode/launch.json` - 213줄로 확장 (기존 46줄)
- `.vscode/tasks.json` - 439줄로 확장 (기존 99줄)
- `.vscode/snippets/go.code-snippets` - 신규 생성 (458줄)
- `aicli-web.code-workspace` - 신규 생성 (125줄)
- `docs/vscode-guide.md` - 신규 생성 (367줄)

#### 주요 개선사항
- Go 개발에 최적화된 완전한 IDE 환경 구성
- 디버깅 시나리오 대폭 확장 (로컬, 원격, 테스트, Docker)
- 태스크 자동화로 개발 생산성 향상
- 풍부한 코드 스니펫으로 반복 작업 최소화
- 팀 간 일관된 개발 환경 보장
- 상세한 사용 가이드로 온보딩 시간 단축

모든 Acceptance Criteria가 성공적으로 완료되었습니다.