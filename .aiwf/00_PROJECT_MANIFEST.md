# PROJECT MANIFEST - AICode Manager

## 프로젝트 정보
- **프로젝트명**: AICode Manager (aicli-web)
- **버전**: 0.1.0 (초기 개발)
- **생성일**: 2025-07-20
- **프로젝트 타입**: CLI 기반 웹 플랫폼
- **주요 기술**: Go, Docker, Claude CLI

## 프로젝트 개요
AICode Manager는 Claude CLI를 웹 플랫폼으로 관리하는 시스템입니다. Terragon에서 영감을 받아 개발되며, 로컬 환경에서 여러 프로젝트의 AI 코딩 작업을 병렬로 실행하고 모니터링할 수 있는 플랫폼을 제공합니다.

## 현재 상태
- **단계**: 설계 완료, 구현 시작
- **진행률**: 10%
- **다음 마일스톤**: M01 - 기본 CLI 도구 구현

## 주요 목표
1. Go 기반의 네이티브 CLI 도구 개발
2. Docker를 통한 격리된 실행 환경 제공
3. 병렬 Claude 인스턴스 실행 지원
4. 웹 기반 실시간 모니터링 인터페이스
5. Git 워크플로우 자동화

## 아키텍처 선택
- **언어**: Go (성능, 동시성, Docker 친화성)
- **웹 프레임워크**: Gin 또는 Echo
- **데이터베이스**: SQLite (로컬) / PostgreSQL (프로덕션)
- **컨테이너**: Docker SDK 직접 통합
- **프론트엔드**: Vue.js 또는 React (추후 결정)

## 프로젝트 구조
```
.aiwf/
├── 00_PROJECT_MANIFEST.md (현재 파일)
├── 01_PROJECT_DOCS/       # 프로젝트 문서
├── 02_REQUIREMENTS/       # 요구사항 및 마일스톤
├── 03_SPRINTS/           # 스프린트 관리
├── 04_GENERAL_TASKS/     # 일반 태스크
├── 05_MEETINGS/          # 회의록
├── 06_RETROSPECTIVES/    # 회고
└── 07_REPORTS/           # 보고서
```

## 설계 문서
- `/docs/cli-design/`: CLI 기반 설계 문서
- `/docs/python-design/`: Python 기반 설계 문서 (참고용)
- `/CLAUDE.md`: 프로젝트 가이드라인

## 팀 구성
- **개발자**: 1명 (AI 어시스턴트 지원)
- **역할**: 풀스택 개발

## 위험 요소
1. Claude CLI와의 통합 복잡도
2. Docker 컨테이너 관리의 안정성
3. 실시간 로그 스트리밍 성능
4. 멀티 프로젝트 동시 실행 시 리소스 관리

## 업데이트 로그
- 2025-07-20: AIWF 구조 생성 및 PROJECT_MANIFEST 작성
- 2025-07-20: 설계 문서 완료 (CLI 및 Python 버전)