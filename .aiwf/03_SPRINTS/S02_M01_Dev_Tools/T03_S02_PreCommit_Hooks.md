---
task_id: T03_S02
sprint_sequence_id: S02
status: open
complexity: Low
estimated_hours: 3
assigned_to: TBD
created_date: 2025-07-20
last_updated: 2025-07-20T04:00:00Z
---

# Task: Pre-commit Hooks 설정

## Description
코드 품질을 보장하고 일관된 코딩 스타일을 유지하기 위한 pre-commit hooks를 설정합니다. 커밋 전에 자동으로 코드 포맷팅, 린팅, 기본 테스트를 실행하여 저품질 코드가 저장소에 커밋되는 것을 방지합니다.

## Goal / Objectives
- pre-commit 프레임워크 설정
- Go 코드 포맷팅 자동화 (gofmt, goimports)
- 린팅 자동 실행 (golangci-lint)
- 기본 테스트 자동 실행
- 커밋 메시지 검증
- 팀원들의 일관된 개발 환경 보장

## Acceptance Criteria
- [ ] .pre-commit-config.yaml 설정 파일 생성
- [ ] pre-commit hooks 설치 스크립트 작성
- [ ] 코드 포맷팅 자동 적용 (gofmt, goimports)
- [ ] 린팅 오류 시 커밋 차단
- [ ] 기본 테스트 실패 시 커밋 차단
- [ ] 커밋 메시지 컨벤션 검증
- [ ] 바이너리 파일 커밋 방지
- [ ] 개발자 가이드 문서 업데이트

## Subtasks
- [ ] pre-commit 프레임워크 설치 및 설정
- [ ] .pre-commit-config.yaml 파일 작성
- [ ] Go 관련 hooks 설정 (format, lint, test)
- [ ] 커밋 메시지 validation hooks 추가
- [ ] 보안 관련 hooks 설정 (secrets 검사)
- [ ] hooks 설치 자동화 스크립트 작성
- [ ] Makefile에 pre-commit 관련 타겟 추가
- [ ] 팀원 온보딩을 위한 설정 가이드 작성

## Technical Guide

### Pre-commit Framework 설정

#### 설치 방법
```bash
# Python pip을 통한 설치
pip install pre-commit

# Homebrew를 통한 설치 (macOS)
brew install pre-commit

# 프로젝트에 hooks 설치
pre-commit install
```

#### .pre-commit-config.yaml 기본 구조
```yaml
# See https://pre-commit.com for more information
repos:
  # Go 관련 hooks
  - repo: https://github.com/dnephin/pre-commit-golang
    rev: v0.5.1
    hooks:
      - id: go-fmt
        name: 'Go 코드 포맷팅 (gofmt)'
        description: 'Go 코드를 표준 형식으로 포맷팅'
      - id: go-imports
        name: 'Go import 정리 (goimports)'
        description: 'Go import 구문 정리 및 포맷팅'
        args: [-local, github.com/drumcap/aicli-web]
      - id: go-vet-mod
        name: 'Go 정적 분석 (go vet)'
        description: 'Go 코드 정적 분석 실행'
      - id: go-mod-tidy
        name: 'Go 모듈 정리 (go mod tidy)'
        description: 'Go 모듈 의존성 정리'
      - id: go-unit-tests
        name: 'Go 단위 테스트'
        description: '단위 테스트 실행 (빠른 테스트만)'
        args: [-short]
      - id: golangci-lint
        name: 'Go 린팅 (golangci-lint)'
        description: 'Go 코드 품질 검사'

  # 일반적인 파일 검사
  - repo: https://github.com/pre-commit/pre-commit-hooks
    rev: v4.4.0
    hooks:
      - id: trailing-whitespace
        name: '공백 문자 정리'
        description: '줄 끝 공백 문자 제거'
      - id: end-of-file-fixer
        name: '파일 끝 개행 확인'
        description: '파일 마지막에 개행 문자 추가'
      - id: check-yaml
        name: 'YAML 파일 검증'
        description: 'YAML 파일 구문 검사'
      - id: check-json
        name: 'JSON 파일 검증'
        description: 'JSON 파일 구문 검사'
      - id: check-toml
        name: 'TOML 파일 검증'
        description: 'TOML 파일 구문 검사'
      - id: check-merge-conflict
        name: '병합 충돌 확인'
        description: '병합 충돌 마커 검사'
      - id: check-added-large-files
        name: '큰 파일 검사'
        description: '큰 파일 커밋 방지 (기본 500KB)'
        args: ['--maxkb=1024']
      - id: detect-private-key
        name: '개인키 검사'
        description: '개인키 파일 커밋 방지'
      - id: mixed-line-ending
        name: '개행 문자 통일'
        description: '일관된 개행 문자 사용 확인'

  # 커밋 메시지 검증
  - repo: https://github.com/compilerla/conventional-pre-commit
    rev: v3.0.0
    hooks:
      - id: conventional-pre-commit
        name: '커밋 메시지 컨벤션 검사'
        description: 'Conventional Commits 형식 검증'
        stages: [commit-msg]
        args: [optional-scope]

  # 보안 검사
  - repo: https://github.com/Yelp/detect-secrets
    rev: v1.4.0
    hooks:
      - id: detect-secrets
        name: '시크릿 키 검사'
        description: 'API 키, 패스워드 등 민감 정보 검사'
        args: ['--baseline', '.secrets.baseline']
        exclude: package.sum

  # 마크다운 검사
  - repo: https://github.com/igorshubovych/markdownlint-cli
    rev: v0.37.0
    hooks:
      - id: markdownlint
        name: '마크다운 문법 검사'
        description: '마크다운 파일 스타일 및 문법 검사'
        args: [--fix]
```

### 커밋 메시지 컨벤션

#### Conventional Commits 규칙
```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

#### 타입 종류
- `feat`: 새로운 기능 추가
- `fix`: 버그 수정
- `docs`: 문서 수정
- `style`: 코드 스타일 변경 (포맷팅, 세미콜론 등)
- `refactor`: 코드 리팩토링
- `test`: 테스트 코드 추가/수정
- `chore`: 빌드 프로세스, 도구 설정 등

#### 한글 커밋 메시지 허용 설정
```yaml
# .pre-commit-config.yaml에서 한글 허용
- repo: local
  hooks:
    - id: commit-msg-korean
      name: '한글 커밋 메시지 검증'
      entry: scripts/validate-commit-msg.sh
      language: script
      stages: [commit-msg]
```

### 설치 자동화 스크립트

#### scripts/setup-precommit.sh
```bash
#!/bin/bash
set -e

echo "🔧 Pre-commit hooks 설정 중..."

# pre-commit 설치 확인
if ! command -v pre-commit &> /dev/null; then
    echo "❌ pre-commit이 설치되지 않았습니다."
    echo "다음 명령어로 설치하세요:"
    echo "  pip install pre-commit"
    echo "  또는"
    echo "  brew install pre-commit"
    exit 1
fi

# hooks 설치
echo "📦 Pre-commit hooks 설치 중..."
pre-commit install
pre-commit install --hook-type commit-msg

# 기존 파일에 대해 hooks 실행
echo "🚀 기존 파일에 대해 hooks 실행 중..."
pre-commit run --all-files || echo "⚠️ 일부 오류가 있습니다. 수정 후 다시 실행하세요."

echo "✅ Pre-commit hooks 설정 완료!"
echo ""
echo "이제 커밋할 때마다 자동으로 다음이 실행됩니다:"
echo "  - 코드 포맷팅 (gofmt, goimports)"
echo "  - 린팅 (golangci-lint)"
echo "  - 단위 테스트 (짧은 테스트만)"
echo "  - 보안 검사 및 파일 검증"
echo ""
echo "hooks를 건너뛰려면: git commit --no-verify"
```

### Makefile 통합

#### Pre-commit 관련 타겟
```makefile
# Pre-commit hooks 설정
.PHONY: setup-precommit
setup-precommit:
	@printf "${BLUE}Setting up pre-commit hooks...${NC}\n"
	@./scripts/setup-precommit.sh

# Pre-commit hooks 수동 실행
.PHONY: precommit
precommit:
	@printf "${BLUE}Running pre-commit hooks...${NC}\n"
	pre-commit run --all-files

# Pre-commit hooks 업데이트
.PHONY: precommit-update
precommit-update:
	@printf "${BLUE}Updating pre-commit hooks...${NC}\n"
	pre-commit autoupdate

# Pre-commit hooks 제거
.PHONY: precommit-uninstall
precommit-uninstall:
	@printf "${YELLOW}Uninstalling pre-commit hooks...${NC}\n"
	pre-commit uninstall
```

### 커스텀 Hooks

#### 프로젝트별 검증 스크립트
```bash
#!/bin/bash
# scripts/validate-go-version.sh

# Go 버전 확인
required_version="1.21"
current_version=$(go version | grep -oE '[0-9]+\.[0-9]+' | head -1)

if [ "$(printf '%s\n' "$required_version" "$current_version" | sort -V | head -n1)" != "$required_version" ]; then
    echo "❌ Go $required_version 이상이 필요합니다. 현재: $current_version"
    exit 1
fi

echo "✅ Go 버전 확인 완료: $current_version"
```

### 구현 노트
- 개발 속도를 위해 너무 엄격하지 않게 설정
- 빠른 검사만 pre-commit에 포함 (느린 통합 테스트는 CI에서)
- 팀원들의 로컬 환경 차이 고려
- hooks 우회 방법 및 상황 문서화
- 단계적 도입으로 개발자 저항 최소화

## Output Log

### [날짜 및 시간은 태스크 진행 시 업데이트]

<!-- 작업 진행 로그를 여기에 기록 -->

**상태**: 📋 대기 중