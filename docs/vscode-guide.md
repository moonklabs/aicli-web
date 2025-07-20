# VS Code 개발 환경 가이드

## 목차
- [개요](#개요)
- [설정 파일 구조](#설정-파일-구조)
- [확장 프로그램 설치](#확장-프로그램-설치)
- [개발 워크플로우](#개발-워크플로우)
- [디버깅 가이드](#디버깅-가이드)
- [태스크 실행](#태스크-실행)
- [코드 스니펫 사용](#코드-스니펫-사용)
- [팁과 트릭](#팁과-트릭)

## 개요

이 가이드는 AICode Manager 프로젝트의 VS Code 개발 환경 설정과 사용법을 설명합니다. 
프로젝트는 Go 언어 개발에 최적화된 설정을 제공하며, 팀 간 일관된 개발 환경을 보장합니다.

## 설정 파일 구조

프로젝트의 VS Code 설정은 다음 파일들로 구성됩니다:

```
.vscode/
├── settings.json       # 에디터 및 확장 프로그램 설정
├── extensions.json     # 권장 확장 프로그램 목록
├── launch.json        # 디버깅 구성
├── tasks.json         # 빌드 및 태스크 자동화
└── snippets/          # 코드 스니펫
    └── go.code-snippets
```

추가로 워크스페이스 파일:
- `aicli-web.code-workspace` - 프로젝트 워크스페이스 설정

## 확장 프로그램 설치

### 필수 확장 프로그램 자동 설치

1. VS Code에서 프로젝트를 열면 권장 확장 프로그램 설치 메시지가 표시됩니다
2. "모두 설치"를 클릭하여 한 번에 설치하거나
3. 확장 프로그램 탭에서 `@recommended` 필터로 권장 목록을 확인할 수 있습니다

### 주요 확장 프로그램

#### Go 개발
- **golang.Go**: Go 언어 공식 지원
- **GitHub Copilot**: AI 코드 어시스턴트

#### 코드 품질
- **ErrorLens**: 인라인 에러 표시
- **Code Spell Checker**: 철자 검사
- **TODO Tree**: TODO 주석 관리
- **Better Comments**: 향상된 주석 하이라이팅

#### 개발 도구
- **GitLens**: Git 히스토리 및 블레임
- **Docker**: Docker 개발 지원
- **REST Client**: API 테스트
- **Makefile Tools**: Makefile 지원

## 개발 워크플로우

### 1. 프로젝트 열기

워크스페이스 파일로 열기를 권장합니다:
```bash
code aicli-web.code-workspace
```

### 2. 개발 환경 준비

```bash
# 도구 설치
make install-tools

# 의존성 다운로드
go mod download

# pre-commit 훅 설치
pre-commit install
```

### 3. 코드 작성

- **자동 포맷팅**: 파일 저장 시 자동으로 `goimports` 실행
- **린팅**: 저장 시 `golangci-lint` 자동 실행
- **자동 완성**: Go 언어 서버(gopls)가 코드 완성 제공

### 4. 빌드 및 테스트

단축키를 사용한 빠른 실행:
- **Ctrl+Shift+B**: 기본 빌드 실행
- **Ctrl+Shift+P** → "Tasks: Run Task": 특정 태스크 선택

## 디버깅 가이드

### CLI 디버깅

1. `launch.json`에서 "Launch CLI" 구성 선택
2. 필요시 `args` 배열에 CLI 인자 추가
3. **F5**로 디버깅 시작

동적 인자 입력:
1. "Launch CLI with Args" 구성 선택
2. 실행 시 입력 프롬프트에서 CLI 인자 입력

### API 서버 디버깅

1. "Launch API Server" 구성 선택
2. **F5**로 디버깅 시작
3. 브레이크포인트 설정 후 API 요청 테스트

### Docker 컨테이너 디버깅

1. Docker 컨테이너를 디버그 모드로 실행:
   ```bash
   make docker-debug
   ```

2. "Attach to Docker Container" 구성 선택
3. **F5**로 원격 디버거 연결

### 테스트 디버깅

현재 파일의 특정 테스트 디버깅:
1. 테스트 함수에 커서 위치
2. "Debug Test at Cursor" 실행
3. 테스트 함수명 입력 (정규식 지원)

## 태스크 실행

### 빌드 태스크

```bash
# VS Code 태스크 실행
Ctrl+Shift+P → "Tasks: Run Build Task"

# 사용 가능한 빌드 태스크:
- go: build          # 기본 빌드
- go: build all      # 모든 플랫폼 빌드
- go: build cli      # CLI만 빌드
- go: build api      # API 서버만 빌드
```

### 테스트 태스크

```bash
# 테스트 실행
Ctrl+Shift+P → "Tasks: Run Test Task"

# 사용 가능한 테스트 태스크:
- go: test           # 기본 테스트
- go: test verbose   # 상세 출력 테스트
- go: test coverage  # 커버리지 리포트 생성
- go: benchmark      # 벤치마크 실행
```

### 개발 태스크

```bash
# 개발 서버 실행
- dev: run cli       # CLI 개발 모드
- dev: run api       # API 서버 개발 모드
- dev: watch         # 파일 변경 감지 자동 재시작
```

### Docker 태스크

```bash
# Docker 관련 태스크
- docker: build             # 이미지 빌드
- docker: start dependencies  # 의존성 컨테이너 시작
- docker: stop dependencies   # 의존성 컨테이너 중지
- docker: logs              # 컨테이너 로그 확인
```

## 코드 스니펫 사용

### 테스트 작성

`gotest` 입력 후 Tab:
```go
func TestFunctionName(t *testing.T) {
    // 테스트 설명
    tests := []struct {
        name string
        // 테스트 케이스 필드
        want    interface{}
        wantErr bool
    }{
        // 테스트 케이스...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // 테스트 로직
        })
    }
}
```

### HTTP 핸들러

`ginhandler` 입력 후 Tab:
```go
// HandlerName 핸들러 설명
func HandlerName(c *gin.Context) {
    var req RequestType
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": err.Error(),
        })
        return
    }
    
    // 비즈니스 로직
    
    c.JSON(http.StatusOK, gin.H{
        "message": "success",
        "data":    responseData,
    })
}
```

### 에러 처리

`iferr` 또는 `errwrap` 입력:
```go
if err != nil {
    return fmt.Errorf("context: %w", err)
}
```

더 많은 스니펫은 `.vscode/snippets/go.code-snippets` 파일을 참조하세요.

## 팁과 트릭

### 1. 멀티 커서 편집

- **Alt+Click**: 여러 위치에 커서 추가
- **Ctrl+D**: 다음 동일 선택 영역 추가
- **Ctrl+Shift+L**: 모든 동일 텍스트 선택

### 2. 빠른 네비게이션

- **Ctrl+P**: 파일 빠른 열기
- **Ctrl+Shift+O**: 심볼 네비게이션
- **F12**: 정의로 이동
- **Alt+F12**: 정의 미리보기

### 3. 리팩토링

- **F2**: 심볼 이름 변경
- **Ctrl+.**: 빠른 수정 및 리팩토링

### 4. 터미널 관리

- **Ctrl+`**: 통합 터미널 토글
- **Ctrl+Shift+`**: 새 터미널 생성
- 터미널에 자동으로 `GO_ENV=development` 설정됨

### 5. Git 통합

- **Ctrl+Shift+G**: 소스 제어 뷰
- GitLens로 인라인 blame 정보 확인
- Git Graph로 브랜치 히스토리 시각화

### 6. 테스트 실행

- 테스트 함수 위의 "run test" 링크 클릭
- Test Explorer 뷰에서 테스트 관리

### 7. 코드 폴딩

- **Ctrl+Shift+[**: 현재 영역 접기
- **Ctrl+Shift+]**: 현재 영역 펼치기
- **Ctrl+K Ctrl+0**: 모두 접기
- **Ctrl+K Ctrl+J**: 모두 펼치기

## 문제 해결

### 확장 프로그램 충돌

만약 Go 관련 기능이 제대로 작동하지 않는다면:
1. 구버전 Go 확장 프로그램 제거 (`ms-vscode.go`, `lukehoban.go`)
2. VS Code 재시작
3. `golang.Go` 확장만 사용

### gopls 서버 문제

```bash
# gopls 재설치
go install golang.org/x/tools/gopls@latest

# VS Code 재시작
```

### 린팅 오류

```bash
# golangci-lint 업데이트
make install-tools

# 캐시 정리
golangci-lint cache clean
```

## 추가 리소스

- [VS Code Go 확장 문서](https://github.com/golang/vscode-go/wiki)
- [golangci-lint 설정](.golangci.yml)
- [Makefile 명령어](../Makefile)
- [프로젝트 README](../README.md)