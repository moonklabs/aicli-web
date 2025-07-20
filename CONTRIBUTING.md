# 기여 가이드 (Contributing Guide)

AICode Manager 프로젝트에 기여해주셔서 감사합니다! 이 문서는 프로젝트에 기여하는 방법을 안내합니다.

## 🎯 기여 방법

### 1. 이슈 생성
- 버그 리포트나 기능 요청은 [GitHub Issues](https://github.com/drumcap/aicli-web/issues)에 생성해주세요
- 이슈 템플릿을 사용하여 상세한 정보를 제공해주세요
- 기존 이슈를 먼저 검색하여 중복을 피해주세요

### 2. 개발 환경 설정

#### 사전 요구사항
- Go 1.21 이상
- Docker 20.10 이상
- Make
- Git

#### 저장소 클론 및 설정
```bash
# 저장소 포크 후 클론
git clone https://github.com/YOUR_USERNAME/aicli-web.git
cd aicli-web

# 의존성 설치
make deps

# 개발 환경 확인
make help
```

### 3. 브랜치 전략
- `main` 브랜치에서 새 브랜치 생성
- 브랜치 명명 규칙: `feature/기능명`, `fix/버그명`, `docs/문서명`

```bash
git checkout -b feature/your-feature-name
```

### 4. 개발 워크플로우

#### 코드 작성
```bash
# 개발 서버 실행 (hot reload)
make dev

# 또는 개별 실행
make run-cli    # CLI 도구 실행
make run-api    # API 서버 실행
```

#### 코드 품질 확인
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

#### 테스트
```bash
# 모든 테스트 실행
make test

# 단위 테스트만
make test-unit

# 커버리지 리포트
make test-coverage
```

#### 빌드
```bash
# 현재 플랫폼용 빌드
make build

# 모든 플랫폼용 빌드
make build-all
```

### 5. 커밋 규칙

#### 커밋 메시지 형식
```
type(scope): 간단한 설명

상세한 설명 (필요시)

관련 이슈: #123
```

#### 커밋 타입
- `feat`: 새로운 기능
- `fix`: 버그 수정
- `docs`: 문서 변경
- `style`: 코드 포맷팅 (기능 변경 없음)
- `refactor`: 리팩토링
- `test`: 테스트 추가/수정
- `chore`: 빌드 프로세스나 도구 변경

#### 예시
```bash
feat(cli): add workspace create command

워크스페이스 생성 명령어를 추가합니다.
- 이름과 경로 유효성 검사
- 설정 파일 자동 생성
- 도움말 및 자동완성 지원

관련 이슈: #42
```

### 6. Pull Request

#### PR 생성 전 체크리스트
- [ ] 모든 테스트 통과 (`make test`)
- [ ] 코드 품질 검사 통과 (`make check`)
- [ ] 문서 업데이트 (필요시)
- [ ] 변경 로그 업데이트 (필요시)

#### PR 템플릿
```markdown
## 변경사항 요약
- 변경된 내용을 간단히 설명

## 테스트
- 테스트 방법과 결과

## 체크리스트
- [ ] 모든 테스트 통과
- [ ] 문서 업데이트
- [ ] 코드 품질 검사 통과

## 관련 이슈
Closes #123
```

## 📋 코딩 스타일

### Go 코딩 규칙
- `gofmt`과 `golangci-lint` 규칙 준수
- 패키지 이름은 소문자, 단수형
- 인터페이스 이름은 `-er` 접미사 사용
- 에러 처리를 명시적으로 수행

### 주석 규칙
- 공개 함수/타입은 반드시 주석 작성
- 주석은 한국어로 작성 (기술 용어는 영어 유지)
- 예시:
```go
// CreateWorkspace는 새로운 워크스페이스를 생성합니다.
// name은 워크스페이스 이름이고, path는 프로젝트 경로입니다.
func CreateWorkspace(name, path string) error {
    // 구현...
}
```

### 디렉토리 구조
```
aicli-web/
├── cmd/           # 실행 가능한 프로그램
├── internal/      # 내부 패키지 (외부에서 import 불가)
├── pkg/          # 외부 공개 패키지
├── docs/         # 문서
├── test/         # 테스트 파일
└── build/        # 빌드 산출물
```

## 🐛 버그 리포트

### 버그 리포트 템플릿
```markdown
## 버그 설명
버그에 대한 명확한 설명

## 재현 단계
1. ...
2. ...
3. ...

## 예상 동작
정상적으로 동작해야 하는 방식

## 실제 동작
실제로 발생하는 현상

## 환경
- OS: [예: macOS 12.6]
- Go 버전: [예: 1.21.0]
- aicli 버전: [예: 0.1.0]

## 추가 정보
스크린샷, 로그 등
```

## 🚀 기능 요청

### 기능 요청 템플릿
```markdown
## 기능 설명
원하는 기능에 대한 명확한 설명

## 동기
이 기능이 필요한 이유

## 상세 설명
기능의 작동 방식

## 대안
고려된 다른 해결책들

## 추가 컨텍스트
기타 관련 정보
```

## 📚 문서 기여

- 문서는 한국어로 작성
- 기술 용어나 명령어는 영어 유지
- 예제 코드는 실행 가능하고 테스트된 것만 포함
- 이미지는 `docs/images/` 디렉토리에 저장

## 🤝 커뮤니티

- 질문이나 토론은 [GitHub Discussions](https://github.com/drumcap/aicli-web/discussions) 활용
- 코드 리뷰 시 건설적이고 친근한 톤 유지
- 다양성과 포용성을 존중

## 📄 라이선스

이 프로젝트에 기여함으로써 귀하의 기여가 프로젝트와 동일한 MIT 라이선스 하에 있음에 동의합니다.

---

**감사합니다!** 🎉

여러분의 기여가 AICode Manager를 더 나은 도구로 만듭니다.