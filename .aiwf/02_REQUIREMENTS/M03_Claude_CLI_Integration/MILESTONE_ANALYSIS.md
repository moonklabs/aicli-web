# 마일스톤 M03 분석 결과

## 목표 및 범위

### 마일스톤 목표
M03: Claude CLI Integration은 AICode Manager의 핵심 기능인 Claude CLI와의 완전한 통합을 구현하는 마일스톤입니다. Go 네이티브 프로세스 관리, 실시간 스트림 처리, 세션 관리를 통해 안정적이고 효율적인 Claude CLI 래핑 시스템을 구축합니다.

### 주요 성공 기준
- Claude CLI 프로세스의 안정적인 생명주기 관리
- 실시간 JSON 스트림 파싱 및 이벤트 처리
- 효율적인 세션 풀링 및 재사용 메커니즘
- 포괄적인 에러 처리 및 자동 복구 시스템

### 핵심 기능 영역
1. **프로세스 관리**: exec.Cmd 기반 프로세스 생성, 모니터링, 종료
2. **스트림 처리**: JSON 파싱, 이벤트 분류, 버퍼 관리
3. **세션 관리**: 상태 관리, 풀링, 설정 관리
4. **에러 처리**: 분류, 재시도, 회로 차단, 복구

## 현재 상태 분석

### 코드베이스 현재 상태

#### 완료된 작업 (M01, M02)
1. **프로젝트 기반 구조**
   - Go 프로젝트 초기화 및 디렉토리 구조 완성
   - CLI 도구 (Cobra) 기본 구조 구현
   - API 서버 (Gin) 프레임워크 설정

2. **CLI 구조 및 기능**
   - 자동완성 시스템 (Bash/Zsh/Fish/PowerShell)
   - 설정 관리 시스템 (Viper 통합)
   - 출력 포맷팅 시스템
   - Claude 래퍼 기초 구조 (프로세스 관리, 스트림 처리, 에러 복구)
   - CLI 에러 처리 프레임워크
   - 테스트 프레임워크

3. **API 기반 구조**
   - JWT 인증 시스템
   - OpenAPI 문서화
   - 워크스페이스/프로젝트/세션/태스크 API 엔드포인트
   - WebSocket 기초 구현
   - Rate Limiting

4. **데이터 모델**
   - 데이터베이스 스키마 설계
   - SQLite/BoltDB 듀얼 스토리지 구현
   - 마이그레이션 시스템
   - 트랜잭션 관리
   - 데이터 검증 시스템

#### 기존 Claude 래퍼 구현 상태
`internal/claude/` 디렉토리에 이미 상당 부분 구현됨:
- `process_manager.go`: 프로세스 생명주기 관리
- `stream_handler.go`: 스트림 처리 시스템
- `stream_parser.go`: JSON 파싱 로직
- `error_recovery.go`: 에러 복구 메커니즘
- `circuit_breaker.go`: 회로 차단기 패턴
- `event_bus.go`: 이벤트 시스템
- `state_machine.go`: 상태 관리

### 기술적 준비도
- **높음**: 프로세스 관리, 스트림 처리 기초 코드 이미 구현
- **중간**: 세션 풀링, 고급 에러 처리 부분적 구현
- **낮음**: 통합 테스트, 성능 최적화, 모니터링

### 식별된 갭 및 과제
1. **통합 필요사항**
   - 기존 Claude 래퍼 코드와 API/CLI 통합
   - 스토리지 레이어와 세션 데이터 연동
   - WebSocket을 통한 실시간 스트림 전달

2. **개선 필요사항**
   - 세션 풀 구현 완성
   - 메트릭 수집 시스템 구현
   - 성능 최적화 (버퍼 풀, 고루틴 관리)

3. **신규 구현 필요**
   - OAuth 토큰 관리 시스템
   - 도구 권한 관리
   - 세션 타임아웃 관리

## 첫 번째 스프린트 계획

### 목표
S01_M03: Claude CLI Process Foundation - Claude CLI 프로세스 관리 기반 구축 및 기존 코드 통합

### 포함 작업
1. **프로세스 매니저 통합 및 개선**
   - 기존 process_manager.go 코드 리팩토링
   - OAuth 토큰 및 환경 변수 관리 추가
   - 프로세스 헬스체크 메커니즘 구현

2. **스트림 핸들러 완성**
   - stream_handler.go와 stream_parser.go 통합
   - 백프레셔 처리 로직 추가
   - 메시지 타입별 라우팅 시스템

3. **기본 세션 관리 구현**
   - 세션 생성/종료 API
   - 세션 상태 추적
   - 세션 설정 관리 (SystemPrompt, MaxTurns 등)

4. **CLI/API 통합**
   - CLI 명령어를 통한 Claude 실행
   - API 엔드포인트와 Claude 래퍼 연결
   - 기본적인 에러 처리

### 완료 기준
- Claude CLI 프로세스 생성 및 종료 성공
- 기본 명령어 실행 및 응답 수신
- 스트림 출력을 CLI/API로 전달
- 단위 테스트 80% 이상 커버리지

## 전체 로드맵 (예상)

### 스프린트 1: Process Foundation (1주)
- 프로세스 관리 기반
- 스트림 처리 통합
- 기본 세션 관리
- CLI/API 연동

### 스프린트 2: Advanced Session Management (1주)
- 세션 풀 구현
- 세션 재사용 로직
- 세션 설정 관리
- 타임아웃 처리

### 스프린트 3: Error Handling & Recovery (1주)
- 고급 에러 분류
- 자동 재시도 메커니즘
- 회로 차단기 완성
- 프로세스 복구 시스템

### 스프린트 4: Performance & Monitoring (1주)
- 성능 최적화 (풀링, 버퍼)
- 메트릭 수집 시스템
- 로그 집계 및 분석
- 통합 테스트 및 벤치마크

## 적응적 관리 전략

### 재평가 포인트
- 각 스프린트 완료 시점
- Claude CLI 버전 업데이트 시
- 성능 목표 미달성 시

### 유연성 확보 방안
- 세션 풀 크기 동적 조정 가능
- 에러 처리 정책 설정 가능
- 스트림 버퍼 크기 조정 가능

### 리스크 대응 계획
1. **Claude CLI API 변경**
   - 버전별 어댑터 패턴 적용
   - 통합 테스트로 조기 감지

2. **성능 이슈**
   - 프로파일링 도구 활용
   - 병목 지점 식별 및 최적화

3. **메모리 누수**
   - pprof 활용 정기 점검
   - 고루틴 누수 탐지 도구 적용