# AICode Manager - 마일스톤 로드맵

## 🎯 전체 로드맵 개요

### 완료된 마일스톤
- ✅ **M01**: Foundation Setup - 프로젝트 초기화
- ✅ **M02**: Core Implementation - 핵심 기능 구현
- ✅ **M03**: Claude CLI Integration - Claude CLI 통합
- ✅ **M04**: Workspace Foundation - 워크스페이스 기반 구축

### 진행 중 마일스톤
- 🚧 **M05**: Advanced Auth System - 고급 인증 시스템 (90% 완료)

### 계획된 마일스톤
- 📋 **M06**: Multi Agent Platform - 멀티 에이전트 플랫폼

---

## 📌 M05: Advanced Auth System 마무리

### 현재 상태
- ✅ OAuth2.0 통합 완료
- ✅ JWT 토큰 인증 구현
- ✅ RBAC 권한 관리 시스템 구현
- ✅ 세션 보안 강화 완료
- ✅ 통합 테스트 완료 (T07_S01)

### 남은 작업 - S02_M05_Auth_Finalization (1주)

#### Week 1 (즉시 시작)
1. **T01_Frontend_Auth_Integration**
   - Vue.js 로그인/회원가입 UI
   - JWT 토큰 관리 (Pinia store)
   - 인증 가드 및 라우터 통합
   - OAuth2 소셜 로그인 버튼

2. **T02_E2E_Auth_Tests**
   - 로그인/로그아웃 플로우 테스트
   - 권한별 접근 제어 테스트
   - 토큰 갱신 시나리오 테스트
   - 세션 보안 정책 검증

3. **T03_Auth_Documentation**
   - API 인증 가이드 작성
   - RBAC 권한 설정 매뉴얼
   - 보안 베스트 프랙티스 문서
   - 관리자 가이드

---

## 🚀 M06: Multi Agent Platform

### 목표
웹 브라우저에서 여러 AI 에이전트(Claude Code, Gemini CLI)를 동시에 실행하고 관리할 수 있는 플랫폼 구축

### 핵심 기능
- 프로젝트당 N개의 독립 에이전트 실행
- Git worktree 기반 격리된 작업 공간
- PTY 기반 실제 터미널 환경 제공
- tmux 스타일 세션 영속성
- 실시간 터미널 스냅샷 미리보기
- 모바일 반응형 터미널 UI

### 전제조건
- ✅ M05 인증 시스템 (에이전트별 권한 관리)
- ✅ Docker 통합 (컨테이너 격리)
- ✅ WebSocket 인프라 (실시간 통신)

---

## 📅 M06 스프린트 계획

### S01_M06_Agent_Backend (2주) - 백엔드 기반 구축

#### Week 1: 에이전트 모델 및 Git 통합
1. **T01_Agent_Model_Definition**
   - `internal/models/agent.go` - 에이전트 모델
   - AgentType (claude-code, gemini-cli, custom)
   - AgentStatus 상태 관리
   - AgentConfig 설정 구조

2. **T02_Git_Worktree_Manager**
   - `internal/git/worktree_manager.go`
   - go-git/go-git v5 통합
   - Worktree 생성/삭제/관리
   - 브랜치별 독립 작업 공간

3. **T03_Agent_Storage_Implementation**
   - `internal/storage/sqlite/agent.go`
   - AgentStorage 인터페이스
   - SQLite 스키마 마이그레이션
   - CRUD 작업 구현

#### Week 2: 에이전트 서비스 및 PTY 통합
4. **T04_Agent_Service_Logic**
   - `internal/services/agent.go`
   - 에이전트 생명주기 관리
   - Docker 컨테이너 통합
   - 상태 모니터링

5. **T05_PTY_Session_Manager**
   - `internal/pty/session.go`
   - Docker PTY 어태치
   - 터미널 크기 동기화
   - 세션 영속성 관리

### S02_M06_PTY_Streaming (1주) - 실시간 스트리밍

#### Week 3: WebSocket 및 스냅샷
6. **T06_WebSocket_PTY_Streaming**
   - `internal/websocket/pty_handler.go`
   - PTY 입출력 스트리밍
   - 터미널 크기 조정 프로토콜
   - 에러 처리 및 재연결

7. **T07_Terminal_Snapshot_System**
   - `internal/snapshot/manager.go`
   - 주기적 터미널 캡처 (1초)
   - 순환 버퍼 최적화
   - ANSI 코드 파싱

8. **T08_Agent_API_Handlers**
   - `internal/server/handlers/agent.go`
   - RESTful API 엔드포인트
   - WebSocket 연결 핸들링
   - 권한 검증 통합

### S03_M06_Frontend_UI (1주) - 프론트엔드 구현

#### Week 4: Vue.js UI 컴포넌트
9. **T09_Agent_Management_UI**
   - `web/src/views/ProjectDashboardView.vue`
   - `web/src/components/Agent/AgentCard.vue`
   - 에이전트 그리드 레이아웃
   - 실시간 상태 표시

10. **T10_Terminal_Interface**
    - `web/src/components/Agent/AgentTerminal.vue`
    - xterm.js 업그레이드
    - PTY 스트리밍 통합
    - 터미널 컨트롤 UI

11. **T11_Mobile_Optimization**
    - 반응형 터미널 디자인
    - 터치 제스처 지원
    - 모바일 키보드 최적화
    - 스와이프 네비게이션

### S04_M06_Integration (1주) - 통합 및 최적화

#### Week 5: 최종 통합
12. **T12_Custom_Prompt_System**
    - 사용자 정의 프롬프트 관리
    - 템플릿 변수 시스템
    - 빠른 실행 단축키

13. **T13_Performance_Testing**
    - 100개 동시 에이전트 테스트
    - PTY 지연시간 측정
    - 메모리 사용량 최적화
    - 스냅샷 성능 튜닝

14. **T14_Documentation**
    - 사용자 가이드
    - API 문서 업데이트
    - 배포 가이드
    - 트러블슈팅 매뉴얼

---

## 📊 성공 지표

### 기술적 지표
- ✅ 동시 에이전트 100개 이상 지원
- ✅ PTY 응답 시간 < 50ms (P95)
- ✅ 스냅샷 생성 시간 < 10ms
- ✅ WebSocket 재연결 < 1초

### 사용자 경험 지표
- ✅ 에이전트 생성 시간 < 5초
- ✅ 터미널 연결 시간 < 2초
- ✅ 모바일 터치 응답 < 100ms
- ✅ 95% 이상의 터미널 명령어 호환성

---

## 🔄 적응적 관리 전략

### 스프린트 검토 포인트
- 각 스프린트 완료 시 다음 스프린트 범위 재조정
- 기술적 발견사항 즉시 반영
- 사용자 피드백 기반 우선순위 변경

### 리스크 관리
| 리스크 | 영향도 | 대응 방안 |
|--------|--------|-----------|
| PTY 호환성 문제 | 높음 | xterm.js 폴백 메커니즘 |
| 성능 병목 | 중간 | 점진적 최적화, 캐싱 전략 |
| 모바일 UX | 중간 | 단계별 개선, A/B 테스트 |

### 품질 게이트
- 각 스프린트 완료 조건 명확화
- 자동화된 테스트 커버리지 80% 이상
- 성능 벤치마크 통과 필수

---

## 📝 다음 단계 액션

### 즉시 시작 (Week 1)
1. M05 마무리 작업 시작
2. AIWF 디렉토리 구조 생성
3. M06 마일스톤 문서 작성

### Week 2부터
1. M06 S01 스프린트 시작
2. 에이전트 모델 설계 리뷰
3. Git worktree 프로토타입

---

*마지막 업데이트: 2024년 12월*
*버전: 1.0*