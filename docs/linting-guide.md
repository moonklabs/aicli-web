# Go 린팅 가이드

## 개요

이 프로젝트는 `golangci-lint`를 사용하여 코드 품질을 유지합니다. 모든 코드는 커밋 전에 린팅 규칙을 통과해야 합니다.

## 린팅 명령어

### 기본 린팅

```bash
make lint                 # 기본 린팅 실행
make lint-fix            # 자동 수정 가능한 이슈 수정
make lint-all            # 모든 린터 활성화하여 검사
make lint-report         # HTML/XML 리포트 생성
```

### 수동 실행

```bash
./bin/golangci-lint run                    # 기본 실행
./bin/golangci-lint run --fix             # 자동 수정
./bin/golangci-lint run --enable-all      # 모든 린터 활성화
```

## 린팅 규칙

현재 활성화된 주요 린터들:

### 코드 품질
- `errcheck`: 에러 처리 검증
- `gosimple`: 코드 단순화 제안
- `govet`: Go 공식 정적 분석
- `staticcheck`: 고급 정적 분석
- `unused`: 사용되지 않는 코드 감지

### 코드 스타일
- `gofmt`: 코드 포맷팅
- `goimports`: import 문 정리
- `revive`: 코딩 스타일 가이드
- `whitespace`: 공백 문자 검증

### 보안
- `gosec`: 보안 취약점 검사

### 성능
- `ineffassign`: 비효율적 할당 감지

## 린팅 우회 방법

### 특정 라인 무시

```go
// nolint:linter-name
problematicCode()

// 여러 린터 무시
// nolint:gosec,errcheck
dangerousCode()

// 모든 린터 무시 (권장하지 않음)
// nolint
anyCode()
```

### 특정 함수 무시

```go
// nolint:funlen
func veryLongFunction() {
    // 긴 함수 내용...
}
```

### 파일 전체 무시

파일 상단에 추가:

```go
// Package example contains example code
// nolint // 전체 파일 무시 (매우 특별한 경우만)
package example
```

### 특정 디렉토리 무시

`.golangci.yml` 파일에서:

```yaml
run:
  skip-dirs:
    - generated_code/
    - third_party/
```

## 권장 사항

### 1. 린팅 우회는 최소한으로

- 가능한 한 코드를 수정하여 린팅 규칙을 통과시키세요
- 우회는 정말 필요한 경우에만 사용하세요

### 2. 우회 시 주석 추가

```go
// gosec: G104 - HTTP 에러는 상위에서 처리됨
// nolint:gosec
resp, _ := http.Get(url)
```

### 3. 특정 린터만 비활성화

```go
// errcheck만 비활성화 (다른 린터는 여전히 동작)
// nolint:errcheck
file.Close()
```

## 사용 사례별 가이드

### 1. 테스트 코드

테스트 코드에서는 일부 규칙이 완화됩니다:

```yaml
# .golangci.yml
issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - gomnd        # 매직 넘버 허용
        - funlen       # 긴 함수 허용
        - goconst      # 상수화 완화
```

### 2. 생성된 코드

```go
//go:generate protoc --go_out=. example.proto

// 생성된 파일은 린팅에서 제외
// nolint
package generated
```

### 3. 외부 라이브러리 래핑

```go
// 외부 라이브러리의 인터페이스를 맞추기 위해 불가피한 경우
// nolint:golint
func (c *Client) XMLHttpRequest() error {
    // 외부 API와 호환성 유지
}
```

## VS Code 통합

### 실시간 린팅

VS Code에서는 파일 저장 시 자동으로 린팅이 실행됩니다:

1. Go 확장 설치: `golang.Go`
2. 설정이 이미 `.vscode/settings.json`에 구성됨
3. 저장 시 자동 포맷팅 및 import 정리

### 수동 실행

- `Ctrl+Shift+P` → "Go: Lint Current Package"
- `Ctrl+Shift+P` → "Go: Lint Workspace"

## 문제 해결

### 1. golangci-lint가 너무 느림

```bash
# 캐시 사용하여 속도 향상
make lint

# 특정 파일만 검사
./bin/golangci-lint run ./internal/specific/package/
```

### 2. false positive

특정 경우에 대해서는 프로젝트 설정을 조정할 수 있습니다:

```yaml
# .golangci.yml
issues:
  exclude-rules:
    - text: "specific error message"
      linters:
        - specific-linter
```

### 3. 새로운 린터 추가

새로운 린터를 추가할 때는:

1. 로컬에서 테스트
2. 기존 코드가 통과하는지 확인
3. 팀과 논의 후 적용

## 참고 자료

- [golangci-lint 공식 문서](https://golangci-lint.run/)
- [Go 코딩 스타일 가이드](https://golang.org/doc/effective_go.html)
- [Uber Go 스타일 가이드](https://github.com/uber-go/guide)

## 설정 파일 위치

- 프로젝트 설정: `.golangci.yml`
- VS Code 설정: `.vscode/settings.json`
- Make 명령어: `Makefile`