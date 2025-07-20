---
task_id: T02_S01
sprint_sequence_id: S01
status: open
complexity: Medium
last_updated: 2025-01-20T09:00:00Z
github_issue: 
---

# Task: CLI 도구 기본 구현 (Cobra 프레임워크)

## Description
AICLI 커맨드라인 도구의 기본 구조를 Cobra 프레임워크를 사용하여 구현합니다. Go 언어로 개발되는 이 도구는 Claude CLI를 웹 플랫폼으로 관리하는 시스템의 핵심 인터페이스 역할을 하게 됩니다.

## Goal / Objectives
Cobra 프레임워크를 활용하여 확장 가능하고 사용자 친화적인 CLI 도구의 기본 구조를 구축합니다.
- Cobra 프레임워크 기반의 표준 CLI 구조 확립
- 계층적 명령 체계와 플래그 시스템 구현
- 설정 관리 및 출력 포맷팅 기능 구현
- 사용자 경험을 고려한 인터랙티브 기능 추가

## Acceptance Criteria
CLI 도구의 기본 구조가 완성되고 핵심 명령어가 작동해야 합니다.
- [ ] Cobra 프레임워크를 사용한 기본 구조 구성 완료
- [ ] workspace, task, logs, config 명령어 골격 구현
- [ ] 전역 플래그 및 설정 시스템 작동
- [ ] 도움말 및 버전 정보 표시 기능 구현
- [ ] 기본적인 에러 처리 및 사용자 피드백 시스템 구축

## Subtasks
CLI 도구 구현을 위한 세부 작업 목록입니다.
- [ ] 프로젝트 구조 설정 (cmd/, internal/, pkg/ 디렉토리 구성)
- [ ] main.go 엔트리포인트 구현
- [ ] root command 구조 및 전역 플래그 구현
- [ ] workspace 명령어 하위 구조 구현 (list, create, delete, info)
- [ ] task 명령어 하위 구조 구현 (create, list, status, cancel)
- [ ] logs 명령어 구현 (정적 로그 조회 및 실시간 스트리밍 준비)
- [ ] config 명령어 구현 (get, set, list)
- [ ] 출력 포맷터 기본 구조 구현 (table, json, yaml)
- [ ] 버전 정보 관리 시스템 구현
- [ ] 자동 완성 스크립트 생성 기능 추가

## Technical Guidelines

### Cobra CLI 패턴 및 베스트 프랙티스

#### 1. 명령어 설계 철학
- **패턴**: `aicli VERB NOUN --ADJECTIVE` 또는 `aicli COMMAND ARG --FLAG`
- **예시**: `aicli create workspace --name=myproject`
- 명령어는 동작(VERB), 인자는 대상(NOUN), 플래그는 수식어(ADJECTIVE) 역할

#### 2. 프로젝트 구조
```
aicli-web/
├── cmd/
│   └── aicli/
│       └── main.go          # 최소한의 진입점
├── internal/
│   ├── cli/
│   │   ├── root.go         # 루트 명령 및 전역 설정
│   │   ├── workspace.go    # workspace 관련 명령
│   │   ├── task.go         # task 관련 명령
│   │   ├── logs.go         # 로그 관련 명령
│   │   └── config.go       # 설정 관련 명령
│   ├── client/             # API 클라이언트 (추후 구현)
│   ├── config/             # 설정 관리 (Viper 통합)
│   └── output/             # 출력 포맷팅
├── pkg/
│   └── version/            # 버전 정보 관리
└── go.mod
```

#### 3. 계층적 명령 구조
- 관련 기능은 부모 명령 아래 그룹화
- 서브커맨드로 특정 동작 구현
- 일관된 명명 규칙 유지

#### 4. 사용자 경험 고려사항
- 명령어는 직관적이고 문장처럼 읽혀야 함
- 별칭(alias) 지원: `ws` → `workspace`, `t` → `task`
- 자동 완성 지원 (bash, zsh, fish, powershell)
- 도움말 플래그 자동 인식 (-h, --help)
- 오타 수정 제안 기능

#### 5. 설정 관리
- Viper 라이브러리 활용
- 설정 파일 위치: `$HOME/.aicli.yaml`
- 환경 변수 자동 바인딩
- 12-factor app 원칙 준수

#### 6. 출력 형식
- 기본: 테이블 형식 (사람이 읽기 쉬움)
- JSON: 프로그래밍 통합용
- YAML: 설정 파일 형식
- 컬러 출력 지원 (터미널 환경 자동 감지)

#### 7. 에러 처리
- 명확한 에러 메시지
- 컨텍스트 정보 포함
- 해결 방법 제안
- 종료 코드 적절히 설정

### 구현 참고사항

#### docs/cli-design/cli-implementation.md 기반 지침:
1. **모듈성**: 각 명령은 독립적인 파일로 관리
2. **테스트 가능성**: 비즈니스 로직은 internal/handlers에 분리
3. **확장성**: 새로운 명령 추가가 쉬운 구조 유지
4. **일관성**: 모든 명령에서 동일한 패턴과 스타일 사용

#### 개발 순서:
1. 기본 구조 설정 (main.go, root.go)
2. 전역 플래그 및 설정 시스템
3. 각 명령어의 골격 구현
4. 출력 포맷팅 시스템
5. 도움말 및 자동 완성
6. 에러 처리 개선

#### 주의사항:
- main.go는 최대한 간결하게 유지 (Cobra 초기화만)
- 명령어 정의와 비즈니스 로직 분리
- 플랫 구조로 시작하여 필요시 모듈화
- 사용자 피드백을 위한 진행 표시기 고려
- 워크스페이스 자동 감지 기능 (.aicli 파일)

## Output Log
*(이 섹션은 작업 진행에 따라 업데이트됩니다)*

[YYYY-MM-DD HH:MM:SS] 태스크 시작
[YYYY-MM-DD HH:MM:SS] 프로젝트 구조 생성 완료
[YYYY-MM-DD HH:MM:SS] 루트 명령 구현 완료
[YYYY-MM-DD HH:MM:SS] 서브커맨드 골격 구현 완료
[YYYY-MM-DD HH:MM:SS] 태스크 완료