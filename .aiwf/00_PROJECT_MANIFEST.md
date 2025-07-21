# AIWF 프로젝트 매니페스트 - AICode Manager

## 프로젝트 정보
- **프로젝트명**: AICode Manager (aicli-web)
- **설명**: Claude CLI를 웹 플랫폼으로 관리하는 로컬 우선 시스템
- **버전**: 0.1.0 (Pre-Alpha)
- **프레임워크**: AIWF v1.0
- **생성일**: 2025-01-20
- **최종 수정일**: 2025-07-21T06:07:00Z

## 기술 스택
- **언어**: Go 1.21+
- **웹 프레임워크**: Gin/Echo (TBD)
- **데이터베이스**: SQLite (임베디드)
- **컨테이너**: Docker SDK
- **CLI**: Cobra
- **실시간 통신**: WebSocket, SSE

## 프로젝트 구조
```
aicli-web/
├── .aiwf/                      # AIWF 프레임워크 문서
│   ├── 00_PROJECT_MANIFEST.md  # 이 파일
│   ├── 01_PROJECT_DOCS/        # 프로젝트 문서
│   │   └── ARCHITECTURE.md     # 아키텍처 설계
│   └── 02_REQUIREMENTS/        # 요구사항 및 마일스톤
│       └── M01_Foundation_Setup/
├── docs/                       # 기존 설계 문서
│   ├── cli-design/            # Go CLI 설계
│   └── python-design/         # Python 설계 (참고용)
└── CLAUDE.md                  # Claude Code 가이드라인
```

## 마일스톤 현황

### 완료된 마일스톤: M01_Foundation_Setup  
- **상태**: 완료 (2025-01-20 시작, 2025-07-21 04:10 완료)
- **목표**: Go 프로젝트 초기화 및 개발 환경 구축
- **기간**: 1주 (3 스프린트)
- **주요 작업**:
  - Go 프로젝트 초기화 ✅
  - 개발 도구 설정 ✅
  - CI/CD 파이프라인 구축 ✅

### 현재 마일스톤: M02_Core_Implementation
- **상태**: 진행 중 (2025-07-21 06:07 시작)
- **목표**: CLI 명령어 구조, API 서버 기본 구조, 데이터베이스 스키마 설계
- **기간**: 2주 (3 스프린트)
- **주요 작업**:
  - CLI 기본 구조 구현
  - API 서버 초기 설정
  - 데이터 모델 설계

### 스프린트 로드맵 (M01)
- **S01_M01_Project_Init** (COMPLETED - 2025-07-20 03:00 완료) - Go 프로젝트 초기화 및 기본 구조 설정
  - Go 모듈 초기화, 디렉토리 구조, Makefile, 기본 코드
  - 완료된 태스크: 
    - T01_S01_Go_Project_Init (COMPLETED - 2025-01-20 15:30 완료)
    - T02_S01_CLI_Tool_Base (COMPLETED - 2025-07-20 00:30 완료)
    - T03_S01_API_Framework_Setup (COMPLETED - 2025-07-20 01:00 완료)
    - T04_S01_API_Middleware (COMPLETED - 2025-07-20 01:30 완료)
    - T05_S01_API_Basic_Endpoints (COMPLETED - 2025-07-20 02:00 완료)
    - T06_S01_Build_System (COMPLETED - 2025-07-20 02:30 완료)
    - T07_S01_Project_Documentation (COMPLETED - 2025-07-20 03:00 완료)
  - 진행률: 7/7 태스크 완료 (100%)
- **S02_M01_Dev_Tools** (IN_PROGRESS - 2025-07-21 01:36 시작) - 개발 도구 및 환경 설정
  - 린터, 테스트, pre-commit, Docker 환경, IDE 설정
  - 완료된 태스크:
    - TX01_S02_Linting_Setup (COMPLETED - 2025-07-21 01:47 완료)
    - TX02_S02_Test_Framework_Setup (COMPLETED - 2025-07-21 03:19 완료)
    - TX03_S02_PreCommit_Hooks (COMPLETED - 2025-07-21 03:05 완료)
    - TX04_S02_Docker_Dev_Environment (COMPLETED - 2025-07-21 03:03 완료)
  - 대기 중인 태스크:
    - T05_S02_VSCode_Settings (OPEN)
  - 진행률: 4/5 태스크 완료 (80%)
- **S03_M01_CI_Setup** (COMPLETED - 2025-07-21 04:10 완료) - CI/CD 파이프라인 및 문서화
  - GitHub Actions, 빌드 자동화, 프로젝트 문서 완성
  - 완료된 태스크:
    - TX01_S03_GitHub_Actions_CI_Pipeline (COMPLETED - 2025-07-20T18:27:29Z 완료) - CI 파이프라인 구축
    - TX02_S03_Multi_Platform_Build (COMPLETED - 2025-07-21T03:37:23+0900 완료) - 멀티 플랫폼 빌드
    - TX03_S03_Release_Automation (COMPLETED - 2025-07-21T03:51:00+0900 완료) - 릴리스 자동화
    - TX04_S03_README_Documentation (COMPLETED - 2025-07-21 03:59) - README 문서화
    - TX05_S03_Contributing_Guide (COMPLETED - 2025-07-21 04:05) - 기여 가이드 작성
    - TX06_S03_Development_Guide (COMPLETED - 2025-07-21 04:10) - 개발 가이드 작성
  - 진행률: 6/6 태스크 완료 (100%)

### 스프린트 로드맵 (M02)
- **S01_M02_CLI_Structure** (IN_PROGRESS - 2025-07-21 07:30 시작) - CLI 기본 구조 구현
  - Cobra 기반 명령어 체계, 설정 관리, Claude CLI 래퍼 설계
  - 태스크 상세화 완료: 11개 태스크 (High: 2개, Medium: 7개, Low: 2개)
  - 핵심 태스크: CLI 자동완성, 설정 관리 (3개), 출력 포맷팅, Claude 래퍼 (3개), 에러 처리, 테스트 프레임워크
  - 완료된 태스크:
    - TX01_S01_CLI_Completion_System (COMPLETED - 2025-07-21 07:51)
    - TX02_S01_CLI_Help_Documentation (COMPLETED - 2025-07-21 07:58)
    - TX03A_S01_Config_Structure_Design (COMPLETED - 2025-07-21 08:16)
    - TX03B_S01_Config_File_Management (COMPLETED - 2025-07-21 08:30)
    - TX03C_S01_Config_Integration (COMPLETED - 2025-07-21 08:52)
    - TX05B_S01_Stream_Handler (COMPLETED - 2025-07-21 09:44)
  - 진행률: 6/11 태스크 완료 (55%)
  - 진행 중인 태스크:
    - T05A_S01_Process_Manager (IN_PROGRESS - 2025-07-21 09:13)
- **S02_M02_API_Foundation** (PLANNED) - API 서버 기초 구축
  - Gin 서버 설정, 라우팅, JWT 인증, OpenAPI 문서화
  - 진행률: 0/X 태스크 (태스크 상세화 필요)
- **S03_M02_Data_Model** (PLANNED) - 데이터 모델 구현
  - 스키마 설계, 모델 구현, CRUD 작업, 마이그레이션
  - 진행률: 0/X 태스크 (태스크 상세화 필요)

### 예정된 마일스톤
1. **M03: Claude CLI 통합** (2주)
2. **M04: 워크스페이스 관리** (3주)
3. **M05: API 서버 구현** (3주)
4. **M06: 웹 인터페이스** (4주)
5. **M07: 배포 및 최적화** (3주)

## 핵심 기능
1. **Claude CLI 래핑**: 프로세스 격리 및 생명주기 관리
2. **워크스페이스 관리**: 멀티 프로젝트 병렬 실행
3. **실시간 로그 스트리밍**: WebSocket 기반
4. **Git 워크플로우 통합**: 자동 브랜치/커밋/PR
5. **사용자 인증 및 권한 관리**

## 프로젝트 원칙
- **로컬 우선**: 클라우드 의존성 최소화
- **성능 최적화**: Go 네이티브 구현
- **보안 격리**: Docker 기반 샌드박스
- **개발자 경험**: 직관적인 CLI 및 웹 UI

## 팀 구성
- **프로젝트 리드**: TBD
- **백엔드 개발**: Go 개발자
- **프론트엔드 개발**: Vue.js/React 개발자
- **DevOps**: Docker/K8s 전문가

## 리스크 관리
1. **기술적 리스크**:
   - Claude CLI API 변경 가능성
   - Docker 성능 오버헤드
   
2. **프로젝트 리스크**:
   - Go 언어 학습 곡선
   - 복잡한 프로세스 관리

## 품질 기준
- **코드 커버리지**: 최소 70%
- **성능**: CLI 명령 응답 < 100ms
- **가용성**: 99.9% (로컬 환경)
- **문서화**: 모든 API 및 함수 문서화

## 다음 단계
1. Go 프로젝트 초기화 (`go mod init`)
2. 기본 디렉토리 구조 생성
3. Makefile 작성
4. 개발 환경 설정

## 업데이트 로그
- 2025-07-21 09:44: TX05B_S01_Stream_Handler 태스크 완료 (Claude CLI 스트림 처리 시스템 완전 구현 - JSON 파서, 이벤트 버스, 버퍼 관리, 테스트 포함)
- 2025-07-21 09:34: T05B_S01_Stream_Handler 태스크 시작 (Claude CLI 스트림 처리 시스템 구현)
- 2025-07-21 09:13: T05A_S01_Process_Manager 태스크 시작 (Claude CLI 프로세스 생명주기 관리 시스템 구현)
- 2025-07-21 08:52: TX03C_S01_Config_Integration 태스크 완료 (Viper 통합 설정 관리, 우선순위 체계, CLI 명령어, 동적 설정 감지 구현)
- 2025-07-21 08:37: T03C_S01_Config_Integration 태스크 시작 (설정 통합 및 우선순위 시스템 구현 - Viper 통합)
- 2025-07-21 08:30: TX03B_S01_Config_File_Management 태스크 완료 (설정 파일 관리 시스템 구현, YAML 읽기/쓰기, 권한 관리, 백업/복구, 동시성 처리)
- 2025-07-21 08:17: T03B_S01_Config_File_Management 태스크 시작 (설정 파일 관리 시스템 구현)
- 2025-07-21 08:16: TX03A_S01_Config_Structure_Design 태스크 완료 (설정 구조체 설계, 기본값 정의, 환경 변수 매핑, 검증 규칙, 스키마 문서화)
- 2025-07-21 08:03: T03A_S01_Config_Structure_Design 태스크 시작 (설정 구조체 및 스키마 설계)
- 2025-07-21 07:58: TX02_S01_CLI_Help_Documentation 태스크 완료 (모든 CLI 명령어 도움말 시스템 완성, 에러 메시지 표준화)
- 2025-07-21 07:51: TX01_S01_CLI_Completion_System 태스크 완료 (Bash/Zsh/Fish/PowerShell 자동완성 및 동적 자동완성 구현)
- 2025-07-21 07:30: T01_S01_CLI_Completion_System 태스크 시작 (CLI 자동완성 시스템 구현)
- 2025-07-21 06:33: S01_M02_CLI_Structure 스프린트 태스크 상세화 완료 (11개 태스크: High 2개, Medium 7개, Low 2개)
- 2025-07-21 06:33: 복잡성 분석에 따른 태스크 분할 완료 (T03→T03A/B/C, T05→T05A/B/C)
- 2025-07-21 06:07: M02_Core_Implementation 마일스톤 시작 (CLI 구조, API 기초, 데이터 모델 구현 예정)
- 2025-07-21 06:07: M02 스프린트 계획 수립 완료 (S01_M02_CLI_Structure, S02_M02_API_Foundation, S03_M02_Data_Model)
- 2025-07-21 04:10: TX06_S03_Development_Guide 태스크 완료 (포괄적인 개발 가이드 작성, 아키텍처 설명, 디버깅 가이드, 테스트 전략, 성능 최적화, 배포 방법 포함)
- 2025-07-21 04:10: S03_M01_CI_Setup 스프린트 완료 (6/6 태스크, 100%), M01_Foundation_Setup 마일스톤 완료
- 2025-07-21 04:05: TX05_S03_Contributing_Guide 태스크 완료 (상세한 기여 가이드 작성, 개발 환경 설정, 코딩 스타일, PR 프로세스, 테스트 가이드라인 포함)
- 2025-07-21 03:59: TX04_S03_README_Documentation 태스크 완료 (프로젝트 README.md 전면 재작성, 설치 가이드, CLI/API 문서화, 개발 가이드 추가)
- 2025-07-21 03:51: TX03_S03_Release_Automation 태스크 완료 (Git 태그 기반 릴리스 자동화, 체크섬 생성, Docker 이미지 빌드 포함)
- 2025-07-21 03:41: T03_S03_Release_Automation 태스크 시작 (릴리스 자동화 프로세스 구축)
- 2025-07-21 03:37: TX02_S03_Multi_Platform_Build 태스크 완료 (멀티 플랫폼 빌드 자동화, CGO 비활성화, 바이너리 최적화 구현)
- 2025-07-20 18:27: TX01_S03_GitHub_Actions_CI_Pipeline 태스크 완료 (GitHub Actions CI 파이프라인 구축, 보안 스캔 포함)
- 2025-07-20 18:22: T01_S03_GitHub_Actions_CI_Pipeline 태스크 시작, S03 스프린트 진행 중
- 2025-01-20 14:45: T01_S01_Go_Project_Init 태스크 시작, 마일스톤 상태 업데이트
- 2025-01-20 15:30: T01_S01_Go_Project_Init 태스크 완료 (Go 프로젝트 초기화, 디렉토리 구조, Makefile 생성)
- 2025-01-20 15:30: T02_S01_CLI_Tool_Base 태스크 시작, 진행률 업데이트 (1/7, 14%)
- 2025-07-20 00:00: T02_S01_CLI_Tool_Base 태스크 상태를 IN_PROGRESS로 변경
- 2025-07-20 00:30: T02_S01_CLI_Tool_Base 태스크 완료 (Cobra CLI 구조, 기본 명령어, 버전 시스템 구현)
- 2025-07-20 00:30: T03_S01_API_Framework_Setup 태스크 시작, 진행률 업데이트 (2/7, 29%)
- 2025-07-20 01:00: T03_S01_API_Framework_Setup 태스크 완료 (Gin 프레임워크, 라우터, 기본 미들웨어 구현)
- 2025-07-20 01:00: T04_S01_API_Middleware 태스크 시작, 진행률 업데이트 (3/7, 43%)
- 2025-07-20 01:30: T04_S01_API_Middleware 태스크 완료 (CORS, 로깅, 에러 핸들링, 복구 미들웨어 구현)
- 2025-07-20 01:30: T05_S01_API_Basic_Endpoints 태스크 시작, 진행률 업데이트 (4/7, 57%)
- 2025-07-20 02:00: T05_S01_API_Basic_Endpoints 태스크 완료 (헬스체크, 프로젝트 목록, 정보 조회 API 엔드포인트 구현)
- 2025-07-20 02:00: T06_S01_Build_System 태스크 시작, 진행률 업데이트 (5/7, 71%)
- 2025-07-20 02:30: T06_S01_Build_System 태스크 완료 (Makefile 고도화, 빌드 타겟, 테스트, 배포 자동화 구현)
- 2025-07-20 02:30: T07_S01_Project_Documentation 태스크 시작, 진행률 업데이트 (6/7, 86%)
- 2025-07-20 03:00: T07_S01_Project_Documentation 태스크 완료 (README, API 문서, CONTRIBUTING 가이드 작성)
- 2025-07-20 03:00: S01_M01_Project_Init 스프린트 완료 (100%), M01 마일스톤 1/3 스프린트 완료
- 2025-07-21 03:05: TX03_S02_PreCommit_Hooks 태스크 완료 (pre-commit 설정, Makefile 통합, 문서화 완료)
- 2025-07-21 02:47: T04_S02_Docker_Dev_Environment 태스크 시작, 진행률 업데이트
- 2025-07-21 03:03: TX04_S02_Docker_Dev_Environment 태스크 완료 (Docker 개발 환경 구성, hot reload, 디버깅 지원 구현)

---

**참고**: 이 매니페스트는 프로젝트 진행에 따라 지속적으로 업데이트됩니다.