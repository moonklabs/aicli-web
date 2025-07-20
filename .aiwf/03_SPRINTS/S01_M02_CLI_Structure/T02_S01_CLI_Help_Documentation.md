---
task_id: T02_S01
sprint_sequence_id: S01_M02
status: open
complexity: Low
last_updated: 2025-07-21T06:16:00Z
github_issue: # Optional: GitHub issue number
---

# Task: CLI 도움말 시스템 완성

## Description
AICode Manager CLI의 모든 명령어에 대한 포괄적이고 사용자 친화적인 도움말 시스템을 구현합니다. 각 명령어의 용도, 사용법, 예시를 명확히 제공하여 사용자가 CLI를 효율적으로 활용할 수 있도록 합니다.

## Goal / Objectives
- 모든 CLI 명령어에 대한 상세한 도움말 제공
- 실제 사용 예시와 시나리오 포함
- 일관된 도움말 형식 및 스타일 적용
- 에러 메시지 표준화 및 개선

## Acceptance Criteria
- [ ] 모든 명령어에 Short 및 Long 설명 완성
- [ ] 각 명령어에 대한 실제 사용 예시 포함
- [ ] 플래그 및 인자에 대한 상세 설명 제공
- [ ] 에러 메시지 표준화 및 해결 방법 제시
- [ ] `aicli help` 및 `aicli --help` 명령어 완성도 100%

## Subtasks
- [ ] 기존 명령어 도움말 검토 및 개선
- [ ] 사용 예시 (Examples) 작성
- [ ] 플래그 설명 표준화
- [ ] 에러 메시지 개선
- [ ] 도움말 형식 일관성 검증
- [ ] 문서화 품질 검증

## 기술 가이드

### 주요 인터페이스 및 통합 지점
- **기존 파일**: `internal/cli/commands/*.go` 파일들
- **Cobra 필드**: `Use`, `Short`, `Long`, `Example`, `Args`
- **에러 처리**: 기존 에러 핸들링 패턴 개선

### 따라야 할 기존 패턴
- Cobra Command 구조체의 표준 필드 활용
- 기존 명령어 그룹화 및 계층 구조
- 에러 메시지 및 로깅 방식

### 작업할 주요 파일들
- `cmd/aicli/main.go` - 루트 명령어 도움말
- `internal/cli/commands/workspace.go` - 워크스페이스 관리 명령어
- `internal/cli/commands/logs.go` - 로그 조회 명령어
- `internal/cli/commands/config.go` - 설정 관리 명령어

### 구현 노트

#### 단계별 구현 접근법
1. **도움말 내용 표준화**
   ```go
   var exampleCmd = &cobra.Command{
       Use:   "example [flags]",
       Short: "한 줄 요약 설명",
       Long:  `상세한 설명...`,
       Example: `  aicli example --flag value
     aicli example --help`,
   }
   ```

2. **에러 메시지 개선**
   - 구체적인 문제 설명
   - 해결 방법 제시
   - 관련 도움말 링크 제공

3. **사용 예시 작성**
   - 일반적인 사용 시나리오
   - 고급 사용법
   - 실제 워크플로우 예시

#### 기존 테스트 패턴 기반 테스트 접근법
- 도움말 출력 내용 검증
- 에러 메시지 표준 준수 검증
- 명령어 파싱 및 검증 테스트

### 도움말 작성 가이드라인
- **명확성**: 기술적 전문용어 최소화
- **실용성**: 실제 사용 사례 중심
- **일관성**: 동일한 형식 및 용어 사용
- **완성도**: 모든 옵션 및 플래그 설명

## Output Log
*(This section is populated as work progresses on the task)*