# 변경 로그 (Changelog)

이 프로젝트의 모든 주목할 만한 변경사항은 이 파일에 문서화됩니다.

이 형식은 [Keep a Changelog](https://keepachangelog.com/ko/1.0.0/)를 기반으로 하며,
이 프로젝트는 [Semantic Versioning](https://semver.org/spec/v2.0.0.html)을 따릅니다.

## [Unreleased]

### 추가됨 (Added)
- 초기 프로젝트 구조 설정
- Go 기반 CLI 도구 아키텍처
- Gin 프레임워크 기반 API 서버
- 포괄적인 미들웨어 시스템
- 멀티플랫폼 빌드 시스템

### 변경됨 (Changed)
- 프로젝트명을 terry에서 aicli로 변경

### 개선됨 (Improved)
- 빌드 시스템 자동화
- 코드 품질 검사 도구 통합
- 문서화 표준 수립

## [0.1.0] - 2025-01-20

### 추가됨 (Added)
- **프로젝트 초기화**
  - Go 모듈 설정 및 표준 디렉토리 구조
  - 기본 Makefile 및 빌드 시스템
  - .gitignore 설정

- **CLI 도구 기반 구현**
  - Cobra 프레임워크 기반 CLI 구조
  - workspace, task, logs, config 명령어 그룹
  - 자동 완성 및 도움말 시스템
  - 설정 관리 (Viper)
  - 출력 포맷터 (table, json, yaml)

- **API 서버 프레임워크**
  - Gin 기반 웹 서버
  - RESTful API 구조 (/api/v1)
  - 헬스체크 엔드포인트
  - 우아한 종료 지원

- **미들웨어 시스템**
  - 요청 ID 추적
  - 구조화된 로깅
  - CORS 정책 관리
  - 보안 헤더 (OWASP 권장)
  - 패닉 복구
  - 표준화된 에러 처리

- **API 엔드포인트**
  - 시스템 정보 (/api/v1/system/info)
  - 버전 정보 (/version)
  - 워크스페이스 CRUD 스텁
  - 실시간 시스템 메트릭

- **빌드 시스템**
  - 멀티플랫폼 빌드 (Linux, macOS, Windows)
  - 컬러 출력이 있는 포괄적인 Makefile
  - 버전 정보 임베딩
  - 코드 품질 도구 통합 (lint, vet, security)
  - 테스트 및 커버리지 리포트
  - Docker 이미지 빌드

- **개발 도구**
  - Hot reload 개발 모드
  - 종합 품질 검사 (`make check`)
  - 벤치마크 테스트
  - 보안 스캔 (gosec)

- **문서화**
  - 포괄적인 README.md
  - 기여 가이드라인 (CONTRIBUTING.md)
  - 변경 로그 (CHANGELOG.md)
  - 프로젝트 구조 문서

### 기술적 세부사항
- **버전**: 0.1.0
- **Go 버전**: 1.21+
- **의존성**:
  - github.com/gin-gonic/gin v1.9.1
  - github.com/spf13/cobra v1.8.0
  - github.com/spf13/viper v1.18.2

### 아키텍처
- 격리된 Docker 컨테이너 기반 실행 환경
- RESTful API + WebSocket 실시간 통신
- 프로젝트별 워크스페이스 관리
- 병렬 태스크 실행 지원

### 품질 보증
- 단위 테스트 프레임워크
- 통합 테스트 구조
- 코드 커버리지 추적
- 정적 분석 도구
- 보안 검사 자동화

---

## 변경사항 분류

- **추가됨 (Added)**: 새로운 기능
- **변경됨 (Changed)**: 기존 기능의 변경
- **개선됨 (Improved)**: 기존 기능의 개선
- **수정됨 (Fixed)**: 버그 수정
- **제거됨 (Removed)**: 제거된 기능
- **보안 (Security)**: 보안 관련 변경

## 링크

- [Keep a Changelog](https://keepachangelog.com/ko/1.0.0/)
- [Semantic Versioning](https://semver.org/spec/v2.0.0.html)
- [GitHub Releases](https://github.com/drumcap/aicli-web/releases)