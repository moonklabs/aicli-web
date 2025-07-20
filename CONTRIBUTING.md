# 기여 가이드 (Contributing Guide)

AICode Manager 프로젝트에 기여해주셔서 감사합니다! 이 문서는 프로젝트에 기여하는 방법을 상세히 안내합니다.

## 목차

- [시작하기](#시작하기)
- [개발 환경 설정](#개발-환경-설정)
- [기여 프로세스](#기여-프로세스)
- [코딩 스타일 가이드](#코딩-스타일-가이드)
- [커밋 메시지 규칙](#커밋-메시지-규칙)
- [Pull Request 가이드라인](#pull-request-가이드라인)
- [이슈 보고](#이슈-보고)
- [테스트 가이드라인](#테스트-가이드라인)
- [개발 워크플로우](#개발-워크플로우)
- [행동 강령](#행동-강령)

## 시작하기

### 사전 요구사항

기여하기 전에 다음 도구들이 설치되어 있는지 확인하세요:

- **Go**: 1.21 이상
- **Git**: 2.0 이상
- **Make**: 빌드 자동화용
- **Docker**: 20.10 이상 (테스트 환경용)
- **pre-commit**: 코드 품질 자동화
- **golangci-lint**: Go 린터

### 추가 도구 설치

```bash
# pre-commit 설치
pip install pre-commit

# golangci-lint 설치
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# goimports 설치
go install golang.org/x/tools/cmd/goimports@latest

# air (hot reload) 설치 (선택사항)
go install github.com/cosmtrek/air@latest
```

## 개발 환경 설정

### 1. 저장소 복제

```bash
# 원본 저장소를 fork한 후 복제
git clone https://github.com/[your-username]/aicli-web.git
cd aicli-web

# upstream 원격 저장소 추가
git remote add upstream https://github.com/drumcap/aicli-web.git
```

### 2. 의존성 설치

```bash
# Go 모듈 다운로드
make deps

# 또는 직접 실행
go mod download
go mod tidy
```

### 3. pre-commit 훅 설치

```bash
# pre-commit 훅 설치
make pre-commit-install

# 또는 직접 실행
pre-commit install
pre-commit install --hook-type commit-msg
```

### 4. 개발 환경 검증

```bash
# 빌드 테스트
make build

# 테스트 실행
make test

# 코드 품질 검사
make check
```

### 5. Docker 개발 환경 (선택사항)

```bash
# Docker 개발 환경 시작
make docker-dev

# 개발 환경 중지
make docker-dev-down
```

## 기여 프로세스

### 1. 이슈 생성 및 논의

새로운 기능이나 버그 수정을 시작하기 전에:

1. [GitHub Issues](https://github.com/drumcap/aicli-web/issues)에서 기존 이슈 확인
2. 새로운 이슈가 필요한 경우 이슈 템플릿을 사용하여 생성
3. 이슈에서 작업 내용과 접근 방법을 논의

### 2. 브랜치 생성

```bash
# 최신 main 브랜치로 업데이트
git checkout main
git pull upstream main

# 새 기능 브랜치 생성
git checkout -b feature/새기능-이름

# 또는 버그 수정 브랜치
git checkout -b fix/버그-설명
```

**브랜치 네이밍 규칙:**
- `feature/기능-이름`: 새로운 기능
- `fix/버그-설명`: 버그 수정
- `docs/문서-내용`: 문서 업데이트
- `refactor/리팩토링-내용`: 코드 리팩토링
- `test/테스트-내용`: 테스트 추가/수정

### 3. 개발 진행

```bash
# 개발 모드 실행 (hot reload)
make dev

# 또는 개별 실행
make run-cli    # CLI 도구 실행
make run-api    # API 서버 실행
```

### 4. 코드 품질 검사

변경사항을 커밋하기 전에 항상 다음 검사를 실행하세요:

```bash
# 코드 포맷팅
make fmt

# 코드 분석
make vet

# 린팅
make lint

# 보안 검사
make security

# 모든 품질 검사
make check
```

### 5. 테스트 실행

```bash
# 모든 테스트 실행
make test

# 단위 테스트만
make test-unit

# 통합 테스트만
make test-integration

# 커버리지 리포트 생성
make test-coverage
```

### 6. 커밋 및 푸시

```bash
# 스테이징
git add .

# 커밋 (pre-commit 훅이 자동 실행됨)
git commit -m "feat(api): 워크스페이스 생성 API 추가"

# 푸시
git push origin feature/새기능-이름
```

### 7. Pull Request 생성

1. GitHub에서 Pull Request 생성
2. PR 템플릿을 사용하여 상세한 설명 작성
3. 관련 이슈를 링크
4. 리뷰어 요청

## 코딩 스타일 가이드

### Go 코드 스타일

1. **표준 포맷팅**: `gofmt`와 `goimports` 사용
2. **린터 규칙**: `.golangci.yml` 설정 준수
3. **패키지 구조**: 기존 프로젝트 구조 따르기

#### 코드 예제

```go
// 좋은 예제
package models

import (
    "context"
    "fmt"
    
    "github.com/drumcap/aicli-web/internal/storage"
)

// Workspace는 작업 공간을 나타내는 모델입니다.
type Workspace struct {
    ID          string `json:"id" db:"id"`
    Name        string `json:"name" db:"name"`
    Path        string `json:"path" db:"path"`
    Description string `json:"description" db:"description"`
    CreatedAt   int64  `json:"created_at" db:"created_at"`
    UpdatedAt   int64  `json:"updated_at" db:"updated_at"`
}

// CreateWorkspace는 새로운 워크스페이스를 생성합니다.
// name은 워크스페이스 이름이고, path는 프로젝트 경로입니다.
func CreateWorkspace(ctx context.Context, name, path string) (*Workspace, error) {
    if name == "" {
        return nil, fmt.Errorf("워크스페이스 이름은 필수입니다")
    }
    
    workspace := &Workspace{
        Name: name,
        Path: path,
    }
    
    // 워크스페이스 저장 로직...
    return workspace, nil
}
```

### 명명 규칙

- **변수/함수명**: camelCase (예: `userName`, `getUserInfo`)
- **상수**: UPPER_SNAKE_CASE (예: `MAX_RETRY_COUNT`)
- **인터페이스**: -er 접미사 (예: `WorkspaceManager`, `TaskRunner`)
- **파일명**: snake_case (예: `workspace_manager.go`)
- **패키지명**: 소문자, 단수형 (예: `models`, `storage`)

### 주석 작성 규칙

- **패키지 주석**: 한국어로 작성
- **공개 함수/타입**: GoDoc 형식으로 한국어 주석 작성
- **복잡한 로직**: 인라인 주석으로 설명

```go
// GetWorkspaceByID는 주어진 ID로 워크스페이스를 조회합니다.
// 워크스페이스가 존재하지 않으면 ErrNotFound를 반환합니다.
func GetWorkspaceByID(ctx context.Context, id string) (*Workspace, error) {
    // 입력값 검증
    if id == "" {
        return nil, fmt.Errorf("워크스페이스 ID는 필수입니다")
    }
    
    // 캐시에서 먼저 확인
    if cached := cache.Get(id); cached != nil {
        return cached.(*Workspace), nil
    }
    
    // 데이터베이스에서 조회
    workspace := &Workspace{}
    err := db.GetByID(ctx, id, workspace)
    if err != nil {
        return nil, fmt.Errorf("워크스페이스 조회 실패: %w", err)
    }
    
    return workspace, nil
}
```

### 에러 처리

```go
// 에러 정의 (패키지 레벨)
var (
    ErrWorkspaceNotFound = errors.New("워크스페이스를 찾을 수 없습니다")
    ErrInvalidPath      = errors.New("유효하지 않은 경로입니다")
    ErrDuplicateName    = errors.New("이미 존재하는 워크스페이스 이름입니다")
)

// 에러 래핑 예제
func processWorkspace(ctx context.Context, id string) error {
    workspace, err := GetWorkspaceByID(ctx, id)
    if err != nil {
        return fmt.Errorf("워크스페이스 처리 실패: %w", err)
    }
    
    // 처리 로직...
    if err := workspace.Process(); err != nil {
        return fmt.Errorf("워크스페이스 처리 중 오류 발생: %w", err)
    }
    
    return nil
}
```

### 디렉토리 구조 규칙

새로운 패키지나 파일을 추가할 때 다음 구조를 따르세요:

```
aicli-web/
├── cmd/                    # 실행 가능한 프로그램
│   ├── aicli/             # CLI 도구
│   └── api/               # API 서버
├── internal/              # 내부 패키지
│   ├── cli/               # CLI 명령어
│   │   └── commands/      # 개별 명령어
│   ├── server/            # API 서버
│   ├── models/            # 도메인 모델
│   ├── storage/           # 데이터 저장소
│   └── config/            # 설정 관리
├── pkg/                   # 외부 공개 패키지
│   ├── version/           # 버전 정보
│   └── utils/             # 공용 유틸리티
└── test/                  # 테스트 파일
```

## 커밋 메시지 규칙

### 커밋 메시지 형식

```
<타입>(<범위>): <제목>

<본문>

<꼬리말>
```

### 타입

- `feat`: 새로운 기능
- `fix`: 버그 수정
- `docs`: 문서 변경
- `style`: 코드 포맷팅, 세미콜론 누락 등 (기능 변경 없음)
- `refactor`: 코드 리팩토링
- `test`: 테스트 추가 또는 수정
- `chore`: 빌드 프로세스 또는 보조 도구 변경
- `perf`: 성능 개선
- `ci`: CI 설정 변경
- `build`: 빌드 시스템 또는 외부 의존성 변경
- `revert`: 이전 커밋 되돌리기

### 범위 (선택사항)

- `cli`: CLI 관련 변경사항
- `api`: API 서버 관련 변경사항
- `models`: 데이터 모델 관련 변경사항
- `storage`: 저장소 관련 변경사항
- `config`: 설정 관련 변경사항

### 커밋 메시지 예시

```bash
# 기본 커밋
feat(cli): 워크스페이스 생성 명령어 추가

# 상세 설명이 있는 커밋
feat(api): 실시간 로그 스트리밍 기능 추가

WebSocket을 사용하여 태스크 실행 중 실시간으로 로그를 
스트리밍할 수 있는 기능을 추가했습니다.

- /api/v1/logs/stream/:id 엔드포인트 추가
- WebSocket 연결 관리 로직 구현
- 클라이언트 연결 해제 시 정리 로직 추가

Closes #123

# 버그 수정
fix(storage): 워크스페이스 삭제 시 발생하는 데이터 레이스 수정

# 문서 업데이트
docs: API 사용 예제 추가
```

## Pull Request 가이드라인

### PR 제목

커밋 메시지와 동일한 형식을 사용합니다:

```
feat(api): 워크스페이스 관리 API 추가
```

### PR 템플릿

```markdown
## 변경사항 요약
이 PR에서 변경된 내용을 간단히 설명해주세요.

## 동기 및 배경
이 변경이 필요한 이유를 설명해주세요.

## 변경사항 세부내용
- 변경된 파일과 주요 내용
- 새로 추가된 기능
- 수정된 버그

## 테스트 방법
이 변경사항을 어떻게 테스트했는지 설명해주세요.

```bash
# 테스트 명령어 예제
make test
make test-integration
```

## 스크린샷 (UI 변경 시)
UI 변경사항이 있다면 스크린샷을 첨부해주세요.

## 체크리스트
- [ ] 코드가 올바르게 빌드되는가?
- [ ] 모든 테스트가 통과하는가?
- [ ] 린터 검사를 통과하는가?
- [ ] 새로운 기능에 대한 테스트를 작성했는가?
- [ ] 문서를 업데이트했는가?
- [ ] Breaking change가 있다면 명시했는가?

## 관련 이슈
Closes #123
```

### 리뷰 프로세스

1. **자동 검사**: GitHub Actions CI가 자동으로 실행됩니다
2. **코드 리뷰**: 최소 1명의 리뷰어 승인이 필요합니다
3. **승인 후 머지**: squash merge 또는 rebase merge를 사용합니다

## 이슈 보고

### 버그 리포트

버그를 발견했다면 다음 정보를 포함하여 이슈를 생성하세요:

```markdown
## 버그 설명
무엇이 잘못되었는지 명확하고 간결하게 설명해주세요.

## 재현 단계
버그를 재현하는 단계:
1. '...'로 이동
2. '...'를 클릭
3. 아래로 스크롤
4. 오류 확인

## 예상 동작
어떻게 동작해야 하는지 명확하고 간결하게 설명해주세요.

## 실제 동작
실제로 어떻게 동작하는지 설명해주세요.

## 환경 정보
- OS: [예: macOS 13.0]
- Go 버전: [예: 1.21.0]
- aicli 버전: [예: 0.1.0]
- Docker 버전: [예: 20.10.21]

## 추가 정보
스크린샷, 로그, 또는 기타 관련 정보를 첨부해주세요.
```

### 기능 요청

새로운 기능을 제안하려면:

```markdown
## 기능 설명
원하는 기능에 대한 명확하고 간결한 설명

## 동기
이 기능이 필요한 이유를 설명해주세요. 어떤 문제를 해결하나요?

## 상세 설명
기능이 어떻게 작동해야 하는지 자세히 설명해주세요.

## 사용 사례
이 기능을 어떻게 사용할지 구체적인 예시를 제공해주세요.

## 대안
고려된 다른 해결책이나 기능이 있다면 설명해주세요.

## 추가 컨텍스트
기타 관련 정보나 스크린샷을 첨부해주세요.
```

## 테스트 가이드라인

### 테스트 작성 원칙

1. **단위 테스트**: 모든 공개 함수에 대해 작성
2. **통합 테스트**: API 엔드포인트에 대해 작성
3. **테스트 커버리지**: 최소 80% 유지
4. **테스트 네이밍**: 명확하고 설명적인 이름 사용

### 테스트 구조

```go
func TestWorkspaceCreate(t *testing.T) {
    // Given - 테스트 준비
    workspace := &models.Workspace{
        Name: "test-workspace",
        Path: "/tmp/test",
    }
    
    // When - 테스트 실행
    err := workspace.Create(context.Background())
    
    // Then - 결과 검증
    assert.NoError(t, err)
    assert.NotEmpty(t, workspace.ID)
    assert.Equal(t, "test-workspace", workspace.Name)
}

func TestWorkspaceCreate_InvalidPath(t *testing.T) {
    // Given
    workspace := &models.Workspace{
        Name: "test-workspace",
        Path: "", // 잘못된 경로
    }
    
    // When
    err := workspace.Create(context.Background())
    
    // Then
    assert.Error(t, err)
    assert.Equal(t, models.ErrInvalidPath, err)
}
```

### 테이블 드리븐 테스트

복잡한 테스트의 경우 테이블 드리븐 테스트를 사용하세요:

```go
func TestValidateWorkspaceName(t *testing.T) {
    tests := []struct {
        name      string
        input     string
        wantError bool
        errorType error
    }{
        {
            name:      "유효한 이름",
            input:     "valid-name",
            wantError: false,
        },
        {
            name:      "빈 이름",
            input:     "",
            wantError: true,
            errorType: ErrEmptyName,
        },
        {
            name:      "너무 긴 이름",
            input:     strings.Repeat("a", 256),
            wantError: true,
            errorType: ErrNameTooLong,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateWorkspaceName(tt.input)
            
            if tt.wantError {
                assert.Error(t, err)
                if tt.errorType != nil {
                    assert.Equal(t, tt.errorType, err)
                }
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### 모킹 사용

외부 의존성을 모킹할 때 `internal/testutil` 패키지의 도구를 사용하세요:

```go
func TestWorkspaceService_Create(t *testing.T) {
    // Given
    mockDB := testutil.NewMockDB()
    service := NewWorkspaceService(mockDB)
    
    // When
    workspace, err := service.Create(context.Background(), "test", "/tmp")
    
    // Then
    assert.NoError(t, err)
    assert.Equal(t, "test", workspace.Name)
    mockDB.AssertCalled(t, "Insert", mock.Anything)
}
```

### 테스트 실행

```bash
# 모든 테스트 실행
make test

# 단위 테스트만 실행
make test-unit

# 통합 테스트만 실행
make test-integration

# 커버리지 리포트 생성
make test-coverage

# 특정 패키지 테스트
go test -v ./internal/models/...

# 특정 테스트 함수 실행
go test -v -run TestWorkspaceCreate ./internal/models/

# 벤치마크 테스트
make test-bench
```

## 개발 워크플로우

### 개발 명령어

```bash
# 개발 서버 실행 (hot reload)
make dev

# 개별 실행
make run-cli        # CLI 도구 실행
make run-api        # API 서버 실행

# 빌드
make build          # 현재 플랫폼용 빌드
make build-all      # 모든 플랫폼용 빌드

# 코드 품질
make fmt            # 코드 포맷팅
make vet            # 정적 분석
make lint           # 린팅
make security       # 보안 검사
make check          # 모든 품질 검사

# 정리
make clean          # 빌드 파일 정리
make clean-all      # 모든 캐시 정리
```

### Docker 개발 환경

```bash
# Docker 개발 환경 시작
make docker-dev

# 개별 서비스 실행
make docker-dev-cli     # CLI 개발 컨테이너
make docker-dev-api     # API 서버 (hot reload)

# 테스트 및 린팅 (Docker)
make docker-dev-test    # Docker에서 테스트 실행
make docker-dev-lint    # Docker에서 린팅 실행

# 디버그 모드
make docker-dev-debug   # 디버거 포트 노출

# 로그 확인
make docker-dev-logs    # 모든 서비스 로그

# 환경 정리
make docker-dev-down    # 개발 환경 중지
```

### IDE 설정 (VS Code)

권장 VS Code 확장:

- **Go**: Go 언어 지원
- **Go Outliner**: Go 코드 구조 표시
- **REST Client**: API 테스트
- **GitLens**: Git 기능 향상
- **Todo Tree**: TODO 주석 표시
- **Docker**: Docker 지원

`.vscode/settings.json` 파일이 이미 구성되어 있습니다.

## 성능 및 보안

### 성능 고려사항

1. **메모리 할당**: 불필요한 메모리 할당 피하기
2. **고루틴 관리**: 고루틴 리크 방지
3. **데이터베이스**: N+1 쿼리 문제 주의
4. **로깅**: 적절한 로깅 레벨 사용

### 보안 체크리스트

- [ ] 사용자 입력 검증
- [ ] SQL 인젝션 방지
- [ ] 인증/인가 확인
- [ ] 민감한 정보 로깅 금지
- [ ] 하드코딩된 비밀번호 없음

### 보안 도구

```bash
# 보안 검사 실행
make security

# 또는 직접 실행
gosec ./...
```

## 행동 강령

### 우리의 약속

우리는 모든 참여자에게 괴롭힘 없는 경험을 제공하기 위해 다음을 약속합니다:

- **포용적인 환경**: 나이, 성별, 국적, 종교 등에 관계없이 모든 사람을 환영
- **건설적인 소통**: 비판은 코드에 대해서만, 개인 공격 금지
- **서로 배우는 자세**: 초보자도 전문가도 모두 배울 것이 있다는 마음가짐
- **친근하고 전문적인 태도**: 도움을 요청하고 제공하는 것을 환영

### 금지 행동

- 공격적이거나 모욕적인 언어 사용
- 개인정보 무단 공개
- 괴롭힘이나 trolling 행위
- 정치적/종교적 논쟁
- 스팸이나 무관한 홍보

### 신고

부적절한 행동을 목격하면 maintainer에게 연락해주세요.

## 문서화

### 문서 작성 규칙

- **언어**: 한국어로 작성 (기술 용어는 영어 유지)
- **형식**: Markdown 사용
- **예제**: 실행 가능하고 테스트된 예제만 포함
- **업데이트**: 코드 변경 시 관련 문서도 함께 업데이트

### 문서 구조

```
docs/
├── api/              # API 문서
├── cli/              # CLI 사용법
├── development/      # 개발 가이드
├── architecture/     # 아키텍처 문서
└── examples/         # 사용 예제
```

## 릴리스 프로세스

### 버전 관리

- **Semantic Versioning**: MAJOR.MINOR.PATCH 형식
- **Git 태그**: 릴리스 시 태그 생성
- **CHANGELOG**: 변경 사항 문서화

### 릴리스 준비

```bash
# 릴리스 빌드
make release

# 모든 플랫폼용 빌드 검증
make build-all

# Docker 이미지 빌드
make docker
```

## 도움이 필요할 때

### 지원 채널

1. **GitHub Discussions**: 일반적인 질문과 토론
2. **GitHub Issues**: 버그 리포트와 기능 요청
3. **이메일**: 민감한 문제나 보안 이슈

### 자주 묻는 질문

**Q: Go 초보자도 기여할 수 있나요?**
A: 네! 문서 개선, 번역, 테스트 작성 등 다양한 방법으로 기여할 수 있습니다.

**Q: Windows에서 개발할 수 있나요?**
A: WSL2를 사용하면 Windows에서도 개발 가능합니다.

**Q: 어떤 기능부터 시작하면 좋을까요?**
A: "good first issue" 라벨이 있는 이슈들을 확인해보세요.

## 감사합니다!

여러분의 기여로 AICode Manager가 더 나은 프로젝트가 됩니다. 궁금한 점이 있으면 언제든 문의해주세요!

---

**마지막 업데이트**: 2025-07-21  
**버전**: 1.0

> 이 가이드는 지속적으로 업데이트됩니다. 개선 사항이나 수정이 필요한 부분이 있다면 언제든 알려주세요.