---
task_id: T05_S02
sprint_sequence_id: S02
status: open
complexity: Low
estimated_hours: 2
assigned_to: TBD
created_date: 2025-07-20
last_updated: 2025-07-20T04:00:00Z
---

# Task: VS Code 개발 환경 설정

## Description
팀원들이 일관된 개발 환경에서 작업할 수 있도록 VS Code 설정 파일과 권장 확장 프로그램을 구성합니다. Go 개발에 최적화된 설정과 프로젝트별 커스터마이징을 포함합니다.

## Goal / Objectives
- VS Code 워크스페이스 설정 파일 생성
- Go 개발에 필요한 확장 프로그램 목록 정의
- 코드 포맷팅, 린팅, 디버깅 설정 통합
- 일관된 에디터 설정 및 코드 스타일 적용
- 팀 온보딩을 위한 설정 가이드 작성

## Acceptance Criteria
- [ ] .vscode/settings.json 워크스페이스 설정 파일 생성
- [ ] .vscode/extensions.json 권장 확장 프로그램 목록
- [ ] .vscode/launch.json 디버깅 설정
- [ ] .vscode/tasks.json 빌드/테스트 태스크 설정
- [ ] 코드 포맷팅 자동 적용 설정
- [ ] 린팅 실시간 피드백 설정
- [ ] Git 통합 및 커밋 템플릿 설정
- [ ] VS Code 설정 가이드 문서화

## Subtasks
- [ ] 기본 워크스페이스 설정 (.vscode/settings.json)
- [ ] 필수 및 권장 확장 프로그램 목록 작성
- [ ] Go 언어 서버 및 도구 설정
- [ ] 디버깅 설정 (로컬 및 Docker)
- [ ] 빌드/테스트 태스크 자동화 설정
- [ ] 코드 스니펫 및 템플릿 작성
- [ ] Git 관련 설정 및 커밋 템플릿
- [ ] 팀원 온보딩을 위한 설정 가이드

## Technical Guide

### 워크스페이스 설정 (.vscode/settings.json)

#### 기본 에디터 설정
```json
{
  // 에디터 기본 설정
  "editor.fontSize": 14,
  "editor.fontFamily": "'JetBrains Mono', 'Fira Code', Consolas, 'Courier New', monospace",
  "editor.fontLigatures": true,
  "editor.tabSize": 4,
  "editor.insertSpaces": false,
  "editor.detectIndentation": true,
  "editor.trimAutoWhitespace": true,
  "editor.rulers": [80, 120],
  "editor.wordWrap": "off",
  "editor.minimap.enabled": true,
  "editor.bracketPairColorization.enabled": true,
  "editor.guides.bracketPairs": true,

  // 파일 설정
  "files.encoding": "utf8",
  "files.eol": "\n",
  "files.trimTrailingWhitespace": true,
  "files.insertFinalNewline": true,
  "files.trimFinalNewlines": true,
  "files.autoSave": "onFocusChange",
  "files.exclude": {
    "**/tmp": true,
    "**/vendor": true,
    "**/node_modules": true,
    "**/.git": true,
    "**/dist": true,
    "**/*.exe": true
  },

  // Go 언어 설정
  "go.useLanguageServer": true,
  "go.languageServerExperimentalFeatures": {
    "diagnostics": true,
    "documentLink": true
  },
  "go.lintTool": "golangci-lint",
  "go.lintFlags": [
    "--config=.golangci.yml",
    "--fast"
  ],
  "go.lintOnSave": "package",
  "go.vetOnSave": "package",
  "go.buildOnSave": "package",
  "go.formatTool": "goimports",
  "go.formatFlags": [
    "-local",
    "github.com/drumcap/aicli-web"
  ],
  "go.generateTestsFlags": [
    "-exported"
  ],
  "go.coverOnSave": true,
  "go.coverageDecorator": {
    "type": "gutter",
    "coveredHighlightColor": "rgba(64,128,64,0.2)",
    "uncoveredHighlightColor": "rgba(128,64,64,0.2)",
    "coveredGutterStyle": "blockgreen",
    "uncoveredGutterStyle": "blockred"
  },

  // 테스트 설정
  "go.testFlags": [
    "-v",
    "-race"
  ],
  "go.testTimeout": "30s",
  "go.benchmarkFlags": [
    "-benchmem"
  ],
  "go.testOnSave": false,

  // 자동 완성 및 IntelliSense
  "go.autocompleteUnimportedPackages": true,
  "go.gocodeAutoBuild": false,
  "go.useCodeSnippetsOnFunctionSuggest": true,
  "go.includeImports": true,

  // Git 설정
  "git.enableSmartCommit": true,
  "git.confirmSync": false,
  "git.autofetch": true,
  "git.inputValidation": "warn",
  "git.ignoreLimitWarning": true,

  // 검색 설정
  "search.exclude": {
    "**/node_modules": true,
    "**/vendor": true,
    "**/tmp": true,
    "**/dist": true,
    "**/*.log": true
  },

  // 터미널 설정
  "terminal.integrated.shell.linux": "/bin/bash",
  "terminal.integrated.shell.osx": "/bin/zsh",
  "terminal.integrated.fontSize": 13,
  "terminal.integrated.fontFamily": "'JetBrains Mono', monospace",

  // 기타 설정
  "telemetry.telemetryLevel": "off",
  "breadcrumbs.enabled": true,
  "explorer.confirmDelete": false,
  "explorer.confirmDragAndDrop": false,
  "workbench.editor.enablePreview": false,
  "workbench.startupEditor": "newUntitledFile",
  "workbench.colorTheme": "GitHub Dark Default",
  "workbench.iconTheme": "material-icon-theme"
}
```

### 권장 확장 프로그램 (.vscode/extensions.json)

#### 필수 및 권장 확장 프로그램
```json
{
  "recommendations": [
    // Go 개발 필수
    "golang.go",
    "ms-vscode.vscode-go",
    
    // 개발 도구
    "ms-azuretools.vscode-docker",
    "ms-vscode-remote.remote-containers",
    "ms-vscode.makefile-tools",
    
    // Git 관련
    "eamodio.gitlens",
    "mhutchie.git-graph",
    "donjayamanne.githistory",
    
    // 코드 품질
    "davidanson.vscode-markdownlint",
    "streetsidesoftware.code-spell-checker",
    "editorconfig.editorconfig",
    
    // 유틸리티
    "ms-vscode.hexeditor",
    "redhat.vscode-yaml",
    "ms-vscode.vscode-json",
    "bradlc.vscode-tailwindcss",
    
    // 테마 및 UI
    "pkief.material-icon-theme",
    "github.github-vscode-theme",
    "aaron-bond.better-comments",
    
    // 생산성
    "formulahendry.auto-rename-tag",
    "christian-kohler.path-intellisense",
    "visualstudioexptteam.vscodeintellicode",
    "ms-vscode.vscode-typescript-next",
    
    // 테스트 및 디버깅
    "ms-vscode.test-adapter-converter",
    "hbenl.vscode-test-explorer",
    
    // API 개발
    "humao.rest-client",
    "rangav.vscode-thunder-client"
  ],
  "unwantedRecommendations": [
    "ms-vscode.vscode-typescript",
    "hookyqr.beautify"
  ]
}
```

### 디버깅 설정 (.vscode/launch.json)

#### 로컬 및 Docker 디버깅
```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Debug CLI",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "./cmd/aicli",
            "args": ["workspace", "list"],
            "cwd": "${workspaceFolder}",
            "env": {
                "AICLI_ENV": "development",
                "AICLI_LOG_LEVEL": "debug"
            },
            "console": "integratedTerminal",
            "stopOnEntry": false
        },
        {
            "name": "Debug API Server",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "./cmd/api",
            "env": {
                "AICLI_ENV": "development",
                "AICLI_PORT": "8080",
                "AICLI_LOG_LEVEL": "debug"
            },
            "console": "integratedTerminal",
            "stopOnEntry": false
        },
        {
            "name": "Debug Tests",
            "type": "go",
            "request": "launch",
            "mode": "test",
            "program": "${fileDirname}",
            "env": {
                "AICLI_ENV": "test"
            },
            "args": [
                "-test.v"
            ],
            "console": "integratedTerminal"
        },
        {
            "name": "Debug Docker Container",
            "type": "go",
            "request": "attach",
            "mode": "remote",
            "remotePath": "/app",
            "port": 2345,
            "host": "localhost",
            "program": "/app",
            "cwd": "${workspaceFolder}",
            "console": "integratedTerminal"
        },
        {
            "name": "Debug Current File",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${file}",
            "console": "integratedTerminal",
            "env": {
                "AICLI_ENV": "development"
            }
        }
    ]
}
```

### 태스크 설정 (.vscode/tasks.json)

#### 빌드 및 테스트 자동화
```json
{
    "version": "2.0.0",
    "tasks": [
        {
            "label": "Go: Build",
            "type": "shell",
            "command": "make",
            "args": ["build"],
            "group": {
                "kind": "build",
                "isDefault": true
            },
            "presentation": {
                "echo": true,
                "reveal": "silent",
                "focus": false,
                "panel": "shared"
            },
            "problemMatcher": "$go"
        },
        {
            "label": "Go: Test",
            "type": "shell",
            "command": "make",
            "args": ["test"],
            "group": {
                "kind": "test",
                "isDefault": true
            },
            "presentation": {
                "echo": true,
                "reveal": "always",
                "focus": false,
                "panel": "shared"
            },
            "problemMatcher": "$go"
        },
        {
            "label": "Go: Test with Coverage",
            "type": "shell",
            "command": "make",
            "args": ["test-coverage"],
            "group": "test",
            "presentation": {
                "echo": true,
                "reveal": "always",
                "focus": false,
                "panel": "shared"
            },
            "problemMatcher": "$go"
        },
        {
            "label": "Go: Lint",
            "type": "shell",
            "command": "make",
            "args": ["lint"],
            "group": "build",
            "presentation": {
                "echo": true,
                "reveal": "always",
                "focus": false,
                "panel": "shared"
            },
            "problemMatcher": "$go"
        },
        {
            "label": "Go: Format",
            "type": "shell",
            "command": "make",
            "args": ["fmt"],
            "group": "build",
            "presentation": {
                "echo": true,
                "reveal": "silent",
                "focus": false,
                "panel": "shared"
            }
        },
        {
            "label": "Docker: Build Dev",
            "type": "shell",
            "command": "make",
            "args": ["docker-dev"],
            "group": "build",
            "presentation": {
                "echo": true,
                "reveal": "always",
                "focus": false,
                "panel": "shared"
            }
        },
        {
            "label": "Docker: Stop Dev",
            "type": "shell",
            "command": "make",
            "args": ["docker-dev-stop"],
            "group": "build",
            "presentation": {
                "echo": true,
                "reveal": "always",
                "focus": false,
                "panel": "shared"
            }
        }
    ]
}
```

### 코드 스니펫 (.vscode/go.json)

#### Go 개발용 스니펫
```json
{
    "HTTP Handler": {
        "prefix": "httphandler",
        "body": [
            "func ${1:handlerName}(c *gin.Context) {",
            "\t${2:// TODO: implement handler}",
            "\tc.JSON(http.StatusOK, gin.H{",
            "\t\t\"message\": \"${3:success}\",",
            "\t})",
            "}"
        ],
        "description": "Gin HTTP 핸들러 템플릿"
    },
    "Test Function": {
        "prefix": "testfunc",
        "body": [
            "func Test${1:FunctionName}(t *testing.T) {",
            "\ttests := []struct {",
            "\t\tname     string",
            "\t\tinput    ${2:inputType}",
            "\t\texpected ${3:expectedType}",
            "\t\twantErr  bool",
            "\t}{",
            "\t\t{",
            "\t\t\tname:     \"${4:test case description}\",",
            "\t\t\tinput:    ${5:inputValue},",
            "\t\t\texpected: ${6:expectedValue},",
            "\t\t\twantErr:  false,",
            "\t\t},",
            "\t}",
            "",
            "\tfor _, tt := range tests {",
            "\t\tt.Run(tt.name, func(t *testing.T) {",
            "\t\t\t${7:// Given}",
            "\t\t\t",
            "\t\t\t${8:// When}",
            "\t\t\tresult, err := ${9:functionCall}(tt.input)",
            "\t\t\t",
            "\t\t\t${10:// Then}",
            "\t\t\tif tt.wantErr {",
            "\t\t\t\tassert.Error(t, err)",
            "\t\t\t} else {",
            "\t\t\t\tassert.NoError(t, err)",
            "\t\t\t\tassert.Equal(t, tt.expected, result)",
            "\t\t\t}",
            "\t\t})",
            "\t}",
            "}"
        ],
        "description": "테이블 드리븐 테스트 템플릿"
    },
    "Error Check": {
        "prefix": "iferr",
        "body": [
            "if err != nil {",
            "\treturn ${1:nil, }err",
            "}"
        ],
        "description": "에러 체크 템플릿"
    }
}
```

### 설정 가이드 문서

#### docs/development/vscode-setup.md
```markdown
# VS Code 개발 환경 설정 가이드

## 자동 설정
1. 프로젝트 디렉토리에서 VS Code 실행
2. 권장 확장 프로그램 설치 알림에서 "모두 설치" 클릭
3. Go 언어 서버 설치 알림에서 "Install All" 클릭

## 수동 설정
필수 확장 프로그램:
- Go (golang.go)
- Docker (ms-azuretools.vscode-docker)
- GitLens (eamodio.gitlens)

## 키보드 단축키
- `Ctrl+Shift+P`: 명령 팔레트
- `F5`: 디버깅 시작
- `Ctrl+F5`: 디버깅 없이 실행
- `Ctrl+Shift+T`: 테스트 실행
- `Ctrl+Shift+F`: 전체 검색

## 문제 해결
### Go 도구 설치 오류
```bash
go install -a github.com/go-delve/delve/cmd/dlv@latest
go install -a github.com/haya14busa/goplay/cmd/goplay@latest
```
```

### 구현 노트
- 팀원들의 다양한 운영체제 고려
- 최소한의 설정으로 최대 효율 추구
- 프로젝트별 설정과 개인 설정 분리
- 확장 프로그램은 필수/권장 구분
- 정기적인 설정 업데이트 및 리뷰

## Output Log

### [날짜 및 시간은 태스크 진행 시 업데이트]

<!-- 작업 진행 로그를 여기에 기록 -->

**상태**: 📋 대기 중