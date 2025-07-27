---
milestone_id: M06
title: Multi Agent Platform
status: pending
last_updated: 2025-07-27 19:45
---

## Milestone: Multi Agent Platform

### Goals
- 웹 브라우저에서 여러 AI 에이전트(Claude Code, Gemini CLI)를 동시에 실행하고 관리할 수 있는 플랫폼 구축
- 프로젝트당 N개의 독립 에이전트를 Git worktree 기반 격리된 작업 공간에서 실행
- PTY 기반 실제 터미널 환경 제공 및 tmux 스타일 세션 영속성 구현
- 실시간 터미널 스냅샷 미리보기 및 모바일 반응형 터미널 UI 구축

### Key Documents

- `PRD_Multi_Agent_Platform.md` - 제품 요구사항 문서
- `SPECS_Agent_Backend.md` - 에이전트 백엔드 기술 사양
- `SPECS_PTY_Streaming.md` - PTY 스트리밍 기술 사양
- `SPECS_Frontend_UI.md` - 프론트엔드 UI 기술 사양

### Definition of Done (DoD)

#### 기술적 완료 기준
- [ ] 100개 이상의 동시 에이전트 지원
- [ ] PTY 응답 시간 < 50ms (P95)
- [ ] 터미널 스냅샷 1초 간격 업데이트
- [ ] WebSocket 재연결 시간 < 1초
- [ ] 95% 이상의 터미널 명령어 호환성

#### 기능적 완료 기준
- [ ] 프로젝트당 여러 에이전트 생성/관리 기능
- [ ] Git worktree 자동 생성 및 관리
- [ ] 실시간 터미널 스트리밍 (WebSocket)
- [ ] 터미널 스냅샷 미리보기 시스템
- [ ] tmux 스타일 세션 유지 기능
- [ ] 커스텀 프롬프트 시스템
- [ ] 모바일 반응형 UI 완성

#### 사용자 경험 완료 기준
- [ ] 에이전트 생성 시간 < 5초
- [ ] 터미널 연결 시간 < 2초
- [ ] 모바일에서 원활한 터미널 조작 가능
- [ ] 직관적인 에이전트 그리드 인터페이스

### Notes / Context

**전제조건**:
- M05 고급 인증 시스템 완료 (에이전트별 권한 관리)
- 기존 Docker 통합 및 WebSocket 인프라 활용

**아키텍처 변경사항**:
- 단일 Claude 세션에서 멀티 에이전트 아키텍처로 확장
- PTY 기반 실제 터미널 환경 도입
- Git worktree 기반 독립 작업 공간 구현

**예상 구현 기간**: 4-5주 (4개 스프린트)

**리스크 요소**:
- PTY 호환성 문제 (xterm.js 폴백 메커니즘으로 대응)
- 성능 병목 현상 (점진적 최적화 및 캐싱 전략)
- 모바일 UX 복잡성 (단계별 개선 및 A/B 테스트)

**연관 문서**:
- `/docs/prd/multi-agent-platform-prd.md`
- `/docs/prd/implementation-plan.md`
- `/docs/prd/milestone-roadmap.md`