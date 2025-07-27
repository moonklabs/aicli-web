# PRD: Multi Agent Platform

**참조**: 상세한 제품 요구사항은 `/docs/prd/multi-agent-platform-prd.md`를 참조하세요.

## 개요

이 문서는 `/docs/prd/multi-agent-platform-prd.md`에 작성된 상세 PRD를 AIWF 구조 내에서 참조하기 위한 메타 문서입니다.

## 핵심 참조 문서

- **상세 PRD**: `/docs/prd/multi-agent-platform-prd.md`
- **구현 계획**: `/docs/prd/implementation-plan.md`
- **마일스톤 로드맵**: `/docs/prd/milestone-roadmap.md`
- **실행 요약**: `/docs/prd/execution-summary.md`

## 요약

### 목표
웹 브라우저에서 여러 AI 에이전트를 동시에 실행하고 관리할 수 있는 플랫폼 구축

### 핵심 기능
- 멀티 에이전트 동시 실행
- Git worktree 기반 독립 작업 공간
- PTY 기반 실시간 터미널
- tmux 스타일 세션 영속성
- 모바일 반응형 UI

### 성공 지표
- 100개 이상 동시 에이전트 지원
- PTY 응답 시간 < 50ms
- 95% 이상 터미널 명령어 호환성

## 스프린트 분해

1. **S01_M06_Agent_Backend** (2주) - 백엔드 기반 구축
2. **S02_M06_PTY_Streaming** (1주) - 실시간 스트리밍
3. **S03_M06_Frontend_UI** (1주) - 프론트엔드 구현
4. **S04_M06_Integration** (1주) - 통합 및 최적화