# AIWF 프로젝트 매니페스트 - AICode Manager

## 프로젝트 정보
- **프로젝트명**: AICode Manager (aicli-web)
- **설명**: Claude CLI를 웹 플랫폼으로 관리하는 로컬 우선 시스템
- **버전**: 0.1.0 (Pre-Alpha)
- **프레임워크**: AIWF v1.0
- **생성일**: 2025-01-20
- **최종 수정일**: 2025-07-20T03:00:00Z

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

### 현재 마일스톤: M01_Foundation_Setup
- **상태**: 부분 완료 (2025-01-20 시작, S01 완료됨)
- **목표**: Go 프로젝트 초기화 및 개발 환경 구축
- **기간**: 1주 (3 스프린트)
- **주요 작업**:
  - Go 프로젝트 초기화 ✅
  - 개발 도구 설정 (S02 예정)
  - CI/CD 파이프라인 구축 (S03 예정)

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
- **S03_M01_CI_Setup** (IN_PROGRESS - 2025-07-20T18:22:06Z 시작) - CI/CD 파이프라인 및 문서화
  - GitHub Actions, 빌드 자동화, 프로젝트 문서 완성
  - 완료된 태스크:
    - TX01_S03_GitHub_Actions_CI_Pipeline (COMPLETED - 2025-07-20T18:27:29Z 완료) - CI 파이프라인 구축
    - TX02_S03_Multi_Platform_Build (COMPLETED - 2025-07-21T03:37:23+0900 완료) - 멀티 플랫폼 빌드
    - TX03_S03_Release_Automation (COMPLETED - 2025-07-21T03:51:00+0900 완료) - 릴리스 자동화
    - T04_S03_README_Documentation (OPEN) - README 문서화
    - T05_S03_Contributing_Guide (OPEN) - 기여 가이드 작성
    - T06_S03_Development_Guide (OPEN) - 개발 가이드 작성
  - 진행률: 3/6 태스크 완료 (50%)

### 예정된 마일스톤
1. **M02: 코어 구조 구현** (2주)
2. **M03: Claude CLI 통합** (2주)
3. **M04: 워크스페이스 관리** (3주)
4. **M05: API 서버 구현** (3주)
5. **M06: 웹 인터페이스** (4주)
6. **M07: 배포 및 최적화** (3주)

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