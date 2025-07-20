# T01 - Go 모듈 초기화 및 기본 구조 생성

## 태스크 정보
- **ID**: T01
- **제목**: Go 모듈 초기화 및 기본 구조 생성
- **스프린트**: S01_M01
- **상태**: 대기 중
- **우선순위**: 높음
- **예상 시간**: 1시간

## 목표
Go 프로젝트의 기본 구조를 생성하고 모듈을 초기화하여 개발을 시작할 수 있는 기반을 마련합니다.

## 작업 내용

### 1. Go 모듈 초기화
```bash
go mod init github.com/yourusername/aicli-web
```

### 2. 디렉토리 구조 생성
```
aicli-web/
├── cmd/
│   ├── aicli/          # CLI 도구 엔트리포인트
│   │   └── main.go
│   └── api/            # API 서버 엔트리포인트 (추후)
│       └── main.go
├── internal/           # 내부 패키지 (외부 노출 X)
│   ├── cli/            # CLI 명령어 구현
│   │   ├── root.go
│   │   └── version.go
│   ├── config/         # 설정 관리
│   │   └── config.go
│   ├── version/        # 버전 정보
│   │   └── version.go
│   └── logger/         # 로깅
│       └── logger.go
├── pkg/                # 외부 패키지 (재사용 가능)
│   └── utils/          # 유틸리티 함수
│       └── utils.go
├── scripts/            # 빌드/배포 스크립트
├── docs/               # 문서 (기존)
├── .aiwf/              # AIWF 관리 (기존)
├── go.mod
├── go.sum
├── Makefile
├── README.md
└── LICENSE
```

### 3. 기본 파일 내용

#### cmd/aicli/main.go
```go
package main

import (
    "github.com/yourusername/aicli-web/internal/cli"
)

func main() {
    cli.Execute()
}
```

#### internal/version/version.go
```go
package version

var (
    Version   = "dev"
    GitCommit = "unknown"
    BuildTime = "unknown"
)
```

#### internal/cli/root.go
```go
package cli

import (
    "fmt"
    "os"

    "github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
    Use:   "aicli",
    Short: "AICode Manager CLI - Claude CLI 웹 관리 플랫폼",
    Long: `AICode Manager는 Claude CLI를 웹에서 관리할 수 있는 플랫폼입니다.
여러 프로젝트의 AI 코딩 작업을 병렬로 실행하고 모니터링할 수 있습니다.`,
}

func Execute() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

### 4. go.mod 초기 내용
```
module github.com/yourusername/aicli-web

go 1.21

require (
    github.com/spf13/cobra v1.8.0
    github.com/spf13/viper v1.18.2
)
```

## 체크리스트
- [ ] go mod init 실행
- [ ] 디렉토리 구조 생성
- [ ] main.go 파일 작성
- [ ] version 패키지 구현
- [ ] root command 구현
- [ ] go mod tidy 실행
- [ ] 빌드 테스트

## 완료 조건
1. `go build ./cmd/aicli` 성공
2. `./aicli` 실행 시 도움말 출력
3. 모든 디렉토리와 기본 파일 생성 완료

## 참고 사항
- Cobra를 CLI 프레임워크로 선택 (구글, 쿠버네티스 등에서 사용)
- 내부 패키지는 `internal/`에, 외부 공개 패키지는 `pkg/`에 배치
- 버전 정보는 빌드 시 ldflags로 주입 예정