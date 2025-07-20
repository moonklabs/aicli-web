# Pre-commit Hooks 가이드

이 문서는 AICode Manager 프로젝트의 pre-commit hooks 설정 및 사용 방법을 설명합니다.

## 목차

1. [개요](#개요)
2. [설치 방법](#설치-방법)
3. [사용 방법](#사용-방법)
4. [훅 설명](#훅-설명)
5. [커밋 메시지 규칙](#커밋-메시지-규칙)
6. [문제 해결](#문제-해결)
7. [훅 우회 방법](#훅-우회-방법)

## 개요

Pre-commit hooks는 Git 커밋 전에 자동으로 코드 품질 검사를 수행하는 도구입니다. 이를 통해:

- 일관된 코드 스타일 유지
- 잠재적인 버그 사전 방지
- CI/CD에서의 실패 최소화
- 코드 리뷰 시간 단축

## 설치 방법

### 1. 빠른 설치 (권장)

```bash
# Makefile을 사용한 자동 설치
make pre-commit-install
```

### 2. 수동 설치

```bash
# Python pip를 사용한 pre-commit 설치
pip install pre-commit

# 또는 Homebrew (macOS)
brew install pre-commit

# hooks 설치
pre-commit install
pre-commit install --hook-type commit-msg
```

### 3. 개발 환경 설정에 포함

새로운 개발자가 프로젝트를 시작할 때:

```bash
# 전체 개발 환경 설정
make deps
make pre-commit-install
```

## 사용 방법

### 자동 실행

Git 커밋 시 자동으로 실행됩니다:

```bash
git add .
git commit -m "feat(api): 새로운 워크스페이스 API 추가"
```

### 수동 실행

커밋 전에 미리 확인하고 싶을 때:

```bash
# 모든 파일 검사
make pre-commit-run

# 또는
pre-commit run --all-files

# 특정 훅만 실행
pre-commit run go-fmt --all-files
```

### 훅 업데이트

최신 버전으로 업데이트:

```bash
make pre-commit-update
```

## 훅 설명

### 1. 일반 파일 검사

- **trailing-whitespace**: 줄 끝 공백 제거
- **end-of-file-fixer**: 파일이 개행으로 끝나도록 수정
- **check-yaml**: YAML 파일 문법 검증
- **check-added-large-files**: 500KB 이상 파일 방지
- **check-merge-conflict**: 병합 충돌 마커 검사
- **mixed-line-ending**: LF 줄바꿈 문자로 통일

### 2. Go 언어 특화 훅

#### 코드 포맷팅
- **go-fmt**: Go 표준 포맷팅 적용
- **go-imports**: import 문 정리 및 그룹화

#### 정적 분석
- **go-vet**: Go 공식 정적 분석
- **golangci-lint**: 빠른 린팅 검사 (30초 제한)

#### 의존성 관리
- **go-mod-tidy**: go.mod 파일 정리

#### 테스트 (push 시에만)
- **go-unit-tests**: 빠른 단위 테스트 실행 (30초 제한)

### 3. 커밋 메시지 검증

커밋 메시지가 정해진 형식을 따르는지 검사합니다.

## 커밋 메시지 규칙

### 형식

```
<타입>(<범위>): <제목>

<본문> (선택사항)

<푸터> (선택사항)
```

### 타입

- `feat`: 새로운 기능
- `fix`: 버그 수정
- `docs`: 문서 변경
- `style`: 코드 포맷팅, 세미콜론 누락 등
- `refactor`: 코드 리팩토링
- `test`: 테스트 추가 또는 수정
- `chore`: 빌드 프로세스 또는 보조 도구 변경
- `perf`: 성능 개선
- `ci`: CI 설정 변경
- `build`: 빌드 시스템 또는 외부 의존성 변경
- `revert`: 이전 커밋 되돌리기

### 예시

```bash
# 좋은 예시
git commit -m "feat(api): 워크스페이스 생성 API 추가"
git commit -m "fix(cli): 로그인 시 발생하는 null 포인터 오류 수정"
git commit -m "docs: README에 설치 가이드 추가"
git commit -m "refactor(server): 미들웨어 구조 개선"

# 나쁜 예시
git commit -m "업데이트"
git commit -m "버그 수정"
git commit -m "WIP"
```

### 범위 (선택사항)

변경 사항이 영향을 미치는 범위:
- `api`: API 서버 관련
- `cli`: CLI 도구 관련
- `docker`: Docker 관련
- `docs`: 문서
- `build`: 빌드 시스템

## 문제 해결

### 1. pre-commit이 설치되지 않음

```bash
# Python이 설치되어 있는지 확인
python --version

# pip 업그레이드
pip install --upgrade pip

# pre-commit 재설치
pip install --user pre-commit
```

### 2. golangci-lint가 없음

```bash
# golangci-lint 설치
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# PATH에 추가 확인
export PATH=$PATH:$(go env GOPATH)/bin
```

### 3. 훅이 너무 느림

빠른 검사만 실행하도록 설정되어 있지만, 여전히 느리다면:

```bash
# 특정 훅 건너뛰기
SKIP=go-unit-tests git commit -m "fix: 긴급 수정"
```

### 4. Windows에서 줄바꿈 문제

Git 설정 확인:

```bash
git config core.autocrlf input
```

## 훅 우회 방법

### 1. 긴급 상황에서 전체 훅 건너뛰기

```bash
git commit --no-verify -m "fix: 긴급 핫픽스"
```

⚠️ **주의**: 가능한 한 사용하지 마세요. CI에서 실패할 수 있습니다.

### 2. 특정 훅만 건너뛰기

```bash
# golangci-lint만 건너뛰기
SKIP=golangci-lint git commit -m "feat: 새 기능"

# 여러 훅 건너뛰기
SKIP=go-unit-tests,golangci-lint git commit -m "feat: 새 기능"
```

### 3. 임시로 pre-commit 비활성화

```bash
# 비활성화
pre-commit uninstall

# 커밋 작업...

# 다시 활성화
pre-commit install
```

## 성능 최적화 팁

1. **변경된 파일만 검사**: pre-commit은 기본적으로 스테이징된 파일만 검사합니다.

2. **병렬 실행**: pre-commit은 가능한 경우 훅을 병렬로 실행합니다.

3. **캐시 활용**: golangci-lint와 go build는 캐시를 사용합니다.

4. **선택적 실행**: push 할 때만 테스트를 실행하도록 설정되어 있습니다.

## CI/CD 통합

GitHub Actions에서도 동일한 검사를 수행합니다:

```yaml
- name: Run pre-commit
  uses: pre-commit/action@v3.0.0
```

이를 통해 로컬과 CI 환경에서 일관된 품질 검사를 보장합니다.

## 추가 자료

- [Pre-commit 공식 문서](https://pre-commit.com/)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [golangci-lint 설정](https://golangci-lint.run/usage/configuration/)

## 기여하기

pre-commit 설정을 개선하고 싶다면:

1. `.pre-commit-config.yaml` 파일 수정
2. 로컬에서 테스트
3. PR 생성 시 변경 사항 설명

---

문의사항이 있으면 프로젝트 메인테이너에게 연락하세요.