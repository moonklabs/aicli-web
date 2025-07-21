---
task_id: T04_S01
sprint_sequence_id: S01_M02
status: open
complexity: Medium
last_updated: 2025-07-21T06:18:00Z
github_issue: # Optional: GitHub issue number
---

# Task: CLI 출력 포맷팅 시스템 구현

## Description
AICode Manager CLI의 다양한 출력 형식을 지원하는 포맷팅 시스템을 구현합니다. Table, JSON, YAML 형식을 지원하며, 색상 및 가독성을 고려한 사용자 친화적인 출력을 제공합니다.

## Goal / Objectives
- 다중 출력 형식 지원 (table, json, yaml)
- 색상 지원 및 터미널 호환성 확보
- 일관된 출력 인터페이스 제공
- 파이프라인 및 스크립팅 지원

## Acceptance Criteria
- [x] Table 형식 출력 (기본값) 구현
- [x] JSON 형식 출력 구현
- [x] YAML 형식 출력 구현
- [x] 색상 지원 및 터미널 감지 구현
- [x] `--output` 또는 `-o` 플래그로 형식 선택 가능
- [x] 파이프 환경에서 색상 자동 비활성화

## Subtasks
- [x] 출력 포맷터 인터페이스 설계
- [x] Table 포맷터 구현
- [x] JSON 포맷터 구현
- [x] YAML 포맷터 구현
- [x] 색상 지원 시스템 구현
- [x] CLI 플래그 통합
- [x] 출력 형식 테스트 작성

## 기술 가이드

### 주요 인터페이스 및 통합 지점
- **기존 파일**: `internal/cli/output/formatter.go` 완성
- **외부 라이브러리**: 
  - `github.com/olekukonko/tablewriter` (테이블)
  - `github.com/fatih/color` (색상)
  - `gopkg.in/yaml.v3` (YAML)
- **CLI 통합**: 모든 명령어에 `--output` 플래그 추가

### 포맷터 인터페이스 설계
```go
type Formatter interface {
    Format(data interface{}) (string, error)
    SetHeaders(headers []string)
    SetColorEnabled(enabled bool)
}

type TableFormatter struct {
    headers      []string
    colorEnabled bool
}

type JSONFormatter struct {
    indent       bool
    colorEnabled bool
}

type YAMLFormatter struct {
    colorEnabled bool
}
```

### 따라야 할 기존 패턴
- 기존 CLI 명령어 구조
- 에러 처리 및 로깅 방식
- 설정 관리 통합 (색상 모드 설정)

### 구현 노트

#### 단계별 구현 접근법
1. **포맷터 인터페이스 정의**
   - 공통 인터페이스 설계
   - 각 포맷터 구조체 구현
   - 포맷터 팩토리 패턴 적용

2. **Table 포맷터 구현**
   - 동적 컬럼 크기 조정
   - 헤더 및 행 구분선
   - 색상 지원 (헤더, 데이터 구분)

3. **JSON/YAML 포맷터**
   - 구조화된 데이터 직렬화
   - Pretty printing 지원
   - 문법 하이라이팅 (색상)

4. **CLI 통합**
   - 글로벌 `--output` 플래그
   - 설정 파일 기본값 지원
   - 환경 변수 오버라이드

#### 색상 지원 전략
```go
func detectColorSupport() bool {
    if os.Getenv("NO_COLOR") != "" {
        return false
    }
    
    if !isatty.IsTerminal(os.Stdout.Fd()) {
        return false
    }
    
    return true
}
```

### 지원할 데이터 타입
- **리스트 데이터**: 워크스페이스 목록, 태스크 목록
- **키-값 데이터**: 설정 정보, 시스템 상태
- **중첩 구조**: 상세 정보, 계층적 데이터

### 기존 테스트 패턴 기반 테스트 접근법
- 각 포맷터별 출력 검증
- 색상 코드 포함/제외 테스트
- 터미널 환경 시뮬레이션
- 대용량 데이터 성능 테스트

### 성능 고려사항
- 대용량 데이터 스트리밍 처리
- 메모리 사용량 최적화
- 터미널 크기 감지 및 동적 조정

### 사용 예시
```bash
# 기본 테이블 형식
aicli workspace list

# JSON 형식으로 출력
aicli workspace list --output json

# YAML 형식으로 출력 (색상 비활성화)
NO_COLOR=1 aicli workspace list --output yaml
```

## Output Log

### 2025-07-21 - CLI 출력 포맷팅 시스템 구현 완료

#### 구현된 기능

1. **출력 포맷터 인터페이스 및 구현체**
   - `/internal/cli/output/formatter.go` - 메인 포맷터 구현
   - `Formatter` 인터페이스와 `FormatterManager` 구조체
   - `TableFormatter`, `JSONFormatter`, `YAMLFormatter` 구현체

2. **주요 기능**
   - 테이블 형식 출력 (tablewriter 라이브러리 사용)
   - JSON 형식 출력 (들여쓰기 지원)
   - YAML 형식 출력
   - 색상 지원 (fatih/color 라이브러리)
   - 터미널 감지 (mattn/go-isatty)
   - NO_COLOR 환경 변수 지원

3. **CLI 통합**
   - 전역 `--output` 플래그 추가 (root.go)
   - workspace 명령어 통합 (list, info)
   - task 명령어 통합 (list)
   - config 명령어 통합 (list)

4. **테스트 및 예제**
   - `/internal/cli/output/formatter_test.go` - 단위 테스트
   - `/examples/output_formatter_example.go` - 사용 예제

5. **추가된 의존성**
   - `github.com/olekukonko/tablewriter` - 테이블 렌더링
   - `github.com/fatih/color` - 터미널 색상 지원
   - `github.com/mattn/go-isatty` - 터미널 감지

#### 구현 특징

- 리플렉션을 사용한 구조체 자동 테이블 변환
- 맵, 구조체, 슬라이스 등 다양한 데이터 타입 지원
- JSON 태그를 활용한 필드명 매핑
- 색상 문법 하이라이팅 (JSON, YAML)
- 파이프라인 환경 자동 감지

#### 사용 방법

```bash
# 기본 테이블 형식
aicli workspace list

# JSON 형식
aicli workspace list --output json
aicli workspace list -o json

# YAML 형식
aicli workspace list --output yaml

# 색상 비활성화
NO_COLOR=1 aicli workspace list
```

#### 향후 개선 사항

1. CSV 형식 추가 (필요시)
2. 커스텀 템플릿 지원
3. 페이지네이션 지원
4. 더 정교한 문법 하이라이팅