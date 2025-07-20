# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 프로젝트 개요

AICode Manager는 Claude CLI를 웹 플랫폼으로 관리하는 시스템입니다. Go 언어로 개발된 네이티브 CLI 도구를 중심으로 구현됩니다.

**기술 스택**: Go + Gin/Echo + SQLite/BoltDB + Docker SDK

## 주요 개발 명령어

```bash
# 빌드
make build              # 로컬 플랫폼용 빌드
make build-all          # 모든 플랫폼용 빌드

# 테스트
make test               # 단위 테스트 실행
make test-integration   # 통합 테스트 실행
make test-coverage      # 커버리지 리포트 생성

# 개발
make run                # 로컬 실행
make dev                # 개발 모드 (자동 재시작)
make lint               # 코드 린팅
make fmt                # 코드 포맷팅

# Docker
make docker             # Docker 이미지 빌드
make docker-push        # Docker Hub에 푸시

# 정리
make clean              # 빌드 아티팩트 제거
```

## 아키텍처 구조

### 핵심 컴포넌트

1. **Claude CLI 래퍼**
   - 각 프로젝트별 격리된 Docker 컨테이너에서 Claude CLI 실행
   - 프로세스 생명주기 관리 및 로그 스트리밍

2. **워크스페이스 관리**
   - 로컬 프로젝트 디렉토리를 Docker 볼륨으로 마운트
   - 병렬 작업 실행 및 상태 추적

3. **API 서버**
   - RESTful API + WebSocket for 실시간 통신
   - 사용자 인증 (Supabase Auth 또는 자체 구현)

4. **프론트엔드**
   - 실시간 로그 뷰어
   - 멀티 프로젝트 대시보드
   - Git 워크플로우 통합

### 설계 원칙

- **격리성**: 각 Claude 인스턴스는 독립된 환경에서 실행
- **병렬성**: 여러 프로젝트 동시 작업 가능
- **확장성**: 수평적 확장 가능한 아키텍처
- **보안성**: 프로젝트 간 격리 및 사용자 권한 관리

## AIWF 프레임워크

이 프로젝트는 AIWF(AI Workflow) 프레임워크를 사용합니다:

- **마일스톤 기반 개발**: `M##_Milestone_Name/`
- **스프린트 관리**: `S##_M##_Sprint_Name/`  
- **태스크 추적**: `T##_Task_Name.md`
- **프로젝트 매니페스트**: `.aiwf/00_PROJECT_MANIFEST.md`

## 프로젝트 구조

```
aicli-web/
├── cmd/
│   ├── aicli/          # CLI 도구 엔트리포인트
│   └── api/            # API 서버 엔트리포인트
├── internal/           # 내부 패키지 (외부 노출 X)
│   ├── cli/            # CLI 명령어 구현
│   ├── server/         # API 서버 구현
│   ├── claude/         # Claude CLI 래퍼
│   ├── docker/         # Docker 통합
│   ├── storage/        # 데이터 저장소
│   └── models/         # 데이터 모델
├── pkg/                # 외부 패키지 (재사용 가능)
│   ├── version/        # 버전 정보
│   └── utils/          # 유틸리티 함수
├── web/                # 웹 프론트엔드
├── docker/             # Docker 관련 파일
├── docs/               # 문서
├── scripts/            # 빌드/배포 스크립트
├── Makefile           # 빌드 자동화
├── go.mod             # Go 모듈 정의
└── go.sum             # 의존성 잠금
```

## 현재 프로젝트 상태

- **단계**: CLI 기반 구현 시작
- **설계 문서**: `/docs/cli-design/`에 상세 설계 완료

## 개발 시 주의사항

1. **명명 규칙**: 
   - CLI 도구명은 `aicli`로 통일 (이전 `terry`에서 변경됨)
   - Docker 이미지명: `aicli-web`, `aicli-api`, `aicli-workspace`

2. **보안**:
   - Claude API 키는 환경 변수로 관리
   - 각 워크스페이스는 격리된 네트워크에서 실행

3. **Git 워크플로우**:
   - 자동 브랜치 생성 시 `aicli/` 프리픽스 사용
   - 커밋 메시지는 한글로 작성

4. **문서화**:
   - 모든 문서는 `/docs` 폴더에 저장
   - README.md는 루트와 각 설계 폴더에 위치

## 한국어 커뮤니케이션

모든 커뮤니케이션과 코드 주석은 한국어로 작성합니다. 단, 다음은 예외:
- 변수명, 함수명: 영어
- 기술 용어, 라이브러리명: 원문 유지
- API 엔드포인트: 영어