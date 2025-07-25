# M02: 코어 구조 구현 - 마일스톤 요구사항

## 마일스톤 개요

- **ID**: M02
- **이름**: 코어 구조 구현
- **기간**: 2주
- **목표**: CLI 명령어 구조, API 서버 기본 구조, 데이터베이스 스키마 설계

## 비즈니스 목표

1. **CLI 도구 기반 구축**
   - 사용자가 AICode Manager를 명령줄에서 직접 사용할 수 있는 기본 구조 확립
   - Claude CLI 래퍼 메커니즘 설계 및 구현

2. **API 서버 기초 확립**
   - RESTful API 구조 설계
   - 인증 및 권한 관리 시스템 기초 구현

3. **데이터 관리 체계 수립**
   - 프로젝트, 워크스페이스, 세션 관리를 위한 데이터베이스 스키마 설계
   - 효율적인 데이터 저장 및 조회 구조 구현

## 기술 요구사항

### CLI 구조
- Cobra를 활용한 명령어 체계 구현
- 서브커맨드 구조: `aicli [command] [subcommand] [flags]`
- 설정 파일 관리 (YAML/JSON)
- 환경 변수 지원

### API 서버
- Gin 프레임워크 기반 RESTful API
- JWT 기반 인증 시스템
- 미들웨어 체계 (로깅, 인증, 에러 처리)
- OpenAPI 3.0 명세 작성

### 데이터베이스
- SQLite/BoltDB 선택적 사용
- 마이그레이션 시스템 구현
- 모델 정의 및 ORM 설정

## 스프린트 계획

### S01_M02_CLI_Structure (주 1)
- CLI 기본 명령어 구현
- 설정 관리 시스템
- 도움말 및 자동완성
- Claude CLI 래퍼 인터페이스 설계

### S02_M02_API_Foundation (주 1-2)
- API 서버 초기 구조 설정
- 라우팅 및 미들웨어 구현
- 인증/인가 시스템 기초
- API 문서화 시스템

### S03_M02_Data_Model (주 2)
- 데이터베이스 스키마 설계
- 모델 구현 및 마이그레이션
- CRUD 작업 구현
- 데이터 검증 및 제약사항

## 완료 기준

1. **CLI 도구**
   - `aicli` 명령어로 기본 작업 실행 가능
   - 설정 파일 로드 및 저장 가능
   - 도움말 시스템 작동

2. **API 서버**
   - 기본 엔드포인트 작동 (/health, /version)
   - JWT 토큰 발급 및 검증 가능
   - OpenAPI 문서 자동 생성

3. **데이터베이스**
   - 기본 테이블/컬렉션 생성 완료
   - CRUD 작업 테스트 통과
   - 마이그레이션 시스템 작동

## 의존성

- Go 1.21+ 환경 설정 완료 (M01)
- 개발 도구 및 CI/CD 파이프라인 구축 완료 (M01)

## 리스크 및 완화 방안

1. **Claude CLI 통합 복잡성**
   - 리스크: Docker 환경에서의 프로세스 관리 어려움
   - 완화: 초기에는 단순한 래퍼로 시작, 점진적 개선

2. **데이터베이스 선택**
   - 리스크: SQLite vs BoltDB 성능 차이
   - 완화: 추상화 레이어 구현으로 향후 변경 용이하게 설계

3. **API 보안**
   - 리스크: 초기 보안 취약점
   - 완화: 보안 모범 사례 준수, 취약점 스캐닝 도구 활용

## 성공 지표

- CLI 명령어 실행 시간 < 100ms
- API 응답 시간 < 200ms (95 퍼센타일)
- 테스트 커버리지 > 80%
- 문서화 완성도 100%