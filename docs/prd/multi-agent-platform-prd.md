# AICode Manager - 웹 기반 멀티 에이전트 플랫폼 PRD
*Product Requirements Document*

## 1. 제품 개요

### 1.1 제품명
AICode Manager - Web-based Multi-Agent Platform

### 1.2 제품 비전
개발자가 여러 AI 코딩 에이전트(Claude Code, Gemini CLI 등)를 웹 브라우저에서 동시에 실행하고 관리할 수 있는 통합 플랫폼을 제공하여, 언제 어디서나 효율적인 AI 기반 개발 환경을 구축한다.

### 1.3 핵심 가치
- **접근성**: 웹 브라우저만 있으면 어디서든 개발 환경 접근
- **지속성**: tmux 스타일의 세션 유지로 작업 연속성 보장
- **확장성**: 다양한 AI 에이전트 통합 지원
- **효율성**: 멀티 에이전트 병렬 작업으로 개발 생산성 향상

## 2. 핵심 요구사항

### 2.1 시스템 아키텍처
```
┌─────────────────────────────────────────────────────┐
│                   웹 브라우저                         │
├─────────────────────────────────────────────────────┤
│                  프론트엔드 (Vue.js)                  │
├─────────────────────────────────────────────────────┤
│              WebSocket / REST API                    │
├─────────────────────────────────────────────────────┤
│                백엔드 (Go + Gin)                     │
├─────────────────────────────────────────────────────┤
│   에이전트 관리 │ PTY 세션 │ Git Worktree │ Docker   │
└─────────────────────────────────────────────────────┘
```

### 2.2 데이터 모델
```
프로젝트 (Project)
├── Git Repository URL
├── 메인 브랜치
├── 공통 설정 (API 키, 환경변수)
└── 에이전트들 (Agents) [1:N]
    ├── 에이전트 ID
    ├── 에이전트 타입 (claude-code, gemini-cli, custom)
    ├── Git Worktree 경로
    ├── Docker 컨테이너 ID
    ├── PTY 세션 ID
    ├── 상태 (active, idle, stopped)
    ├── 초기 실행 스크립트
    └── 커스텀 프롬프트 목록
```

### 2.3 핵심 기능 요구사항

#### 2.3.1 에이전트 관리
- **생성**: 프로젝트별 여러 에이전트 생성
- **타입 선택**: Claude Code, Gemini CLI, 또는 커스텀 CLI
- **Git Worktree**: 각 에이전트별 독립된 작업 공간
- **격리 실행**: Docker 컨테이너 기반 격리 환경

#### 2.3.2 세션 영속성
- **tmux 스타일**: 브라우저 종료 후에도 프로세스 유지
- **재연결**: 언제든지 실행 중인 세션에 재접속
- **세션 타임아웃**: 설정 가능한 유휴 시간 제한

#### 2.3.3 실시간 터미널
- **PTY (Pseudo Terminal)**: 실제 터미널 환경 제공
- **WebSocket 스트리밍**: 실시간 입출력 스트리밍
- **ANSI 코드 지원**: 컬러 및 커서 제어 완벽 지원

#### 2.3.4 UI/UX 요구사항
- **프로젝트 대시보드**: 프로젝트 목록 및 상태 표시
- **에이전트 그리드**: 실시간 스냅샷과 함께 에이전트 표시
- **터미널 뷰**: 전체 화면 터미널 인터페이스
- **모바일 지원**: 반응형 디자인 및 터치 최적화

#### 2.3.5 커스텀 프롬프트
- **프롬프트 저장**: 자주 사용하는 명령어 템플릿 저장
- **변수 지원**: 동적 값 입력을 위한 변수 시스템
- **원클릭 실행**: 저장된 프롬프트 즉시 실행

## 3. 상세 기능 명세

### 3.1 에이전트 생명주기
```
생성 → 초기화 → 실행 → 유휴 → 정지 → 삭제
         ↓         ↓      ↓
      Worktree  Container PTY
      생성       시작    연결
```

### 3.2 터미널 스냅샷 시스템
- **캡처 주기**: 1초마다 터미널 상태 캡처
- **저장 내용**: 마지막 50줄 + 커서 위치
- **미리보기**: 에이전트 카드에 실시간 표시

### 3.3 WebSocket 프로토콜
```javascript
// 클라이언트 → 서버
{
  type: "pty_input",
  agentId: "agent-123",
  data: "ls -la\n"
}

// 서버 → 클라이언트
{
  type: "pty_output",
  agentId: "agent-123",
  data: "\x1b[34mfile.txt\x1b[0m\n"
}

// 터미널 크기 조정
{
  type: "pty_resize",
  agentId: "agent-123",
  cols: 120,
  rows: 40
}
```

### 3.4 보안 요구사항
- **인증**: JWT 기반 사용자 인증
- **권한**: RBAC 기반 프로젝트/에이전트 접근 제어
- **격리**: Docker 컨테이너별 리소스 제한
- **암호화**: API 키 및 민감 정보 암호화 저장

## 4. 기술 스택

### 4.1 백엔드
- **언어**: Go 1.23+
- **웹 프레임워크**: Gin
- **데이터베이스**: SQLite (임베디드)
- **컨테이너**: Docker SDK
- **PTY**: github.com/creack/pty
- **Git**: go-git/go-git v5

### 4.2 프론트엔드
- **프레임워크**: Vue.js 3 + TypeScript
- **터미널**: xterm.js
- **상태관리**: Pinia
- **UI 라이브러리**: Naive UI
- **빌드 도구**: Vite

### 4.3 인프라
- **컨테이너화**: Docker
- **리버스 프록시**: Nginx
- **모니터링**: Prometheus + Grafana

## 5. API 명세

### 5.1 에이전트 관리 API
```
POST   /api/projects/:projectId/agents
GET    /api/projects/:projectId/agents
GET    /api/agents/:agentId
PUT    /api/agents/:agentId
DELETE /api/agents/:agentId
POST   /api/agents/:agentId/start
POST   /api/agents/:agentId/stop
```

### 5.2 PTY 세션 API
```
POST   /api/agents/:agentId/pty/start
WS     /api/agents/:agentId/pty/stream
POST   /api/agents/:agentId/pty/resize
GET    /api/agents/:agentId/pty/snapshot
```

### 5.3 커스텀 프롬프트 API
```
GET    /api/agents/:agentId/prompts
POST   /api/agents/:agentId/prompts
PUT    /api/prompts/:promptId
DELETE /api/prompts/:promptId
POST   /api/agents/:agentId/execute-prompt
```

## 6. 사용자 시나리오

### 6.1 새 프로젝트 시작
1. 사용자가 프로젝트 생성 (Git URL 입력)
2. Claude Code 에이전트 생성
3. 초기 스크립트: `claude --resume`
4. 터미널에서 작업 시작

### 6.2 멀티 에이전트 작업
1. 메인 브랜치에서 기능 개발 (Agent 1)
2. 버그 수정을 위한 새 에이전트 생성 (Agent 2)
3. 두 에이전트 동시 작업
4. 각각 다른 Git worktree에서 독립적 작업

### 6.3 모바일 접속
1. 스마트폰으로 웹 접속
2. 실행 중인 에이전트 목록 확인
3. 스냅샷으로 상태 파악
4. 터미널 접속하여 작업 계속

## 7. 성능 요구사항

### 7.1 확장성
- 동시 에이전트: 100개 이상
- 동시 사용자: 1000명 이상
- WebSocket 연결: 10000개 이상

### 7.2 응답 시간
- 터미널 입력 지연: < 50ms
- 스냅샷 업데이트: 1초 간격
- API 응답: < 200ms (95 percentile)

### 7.3 가용성
- 시스템 가동률: 99.9%
- 자동 복구: 컨테이너 실패 시 재시작
- 세션 복구: 서버 재시작 후 세션 복원

## 8. 마일스톤 및 일정

### M06: 멀티 에이전트 플랫폼 구축 (4주)

#### Sprint 1 (Week 1-2): 백엔드 구현
- Week 1: 에이전트 모델, Git worktree, 스토리지
- Week 2: Docker PTY 통합, WebSocket 스트리밍

#### Sprint 2 (Week 3-4): 프론트엔드 구현
- Week 3: 에이전트 UI, 터미널 인터페이스
- Week 4: 모바일 최적화, 커스텀 프롬프트

## 9. 리스크 및 대응 방안

### 9.1 기술적 리스크
- **PTY 호환성**: 다양한 CLI 도구의 터미널 제어 시퀀스 지원
  - 대응: xterm.js의 광범위한 ANSI 지원 활용
  
- **WebSocket 안정성**: 네트워크 불안정 시 연결 유지
  - 대응: 자동 재연결 및 버퍼링 메커니즘

- **리소스 관리**: 많은 Docker 컨테이너 실행 시 리소스 부족
  - 대응: 컨테이너별 리소스 제한 및 자동 정리

### 9.2 사용성 리스크
- **복잡한 UI**: 많은 에이전트 관리의 복잡성
  - 대응: 직관적인 그리드 뷰와 필터링 기능

- **모바일 터미널**: 작은 화면에서의 터미널 사용 어려움
  - 대응: 가상 키보드 최적화 및 제스처 지원

## 10. 향후 확장 계획

### 10.1 Phase 2 (M07)
- 협업 기능: 실시간 공동 작업
- AI 에이전트 간 통신
- 작업 자동화 및 파이프라인

### 10.2 Phase 3 (M08)
- 플러그인 시스템
- 커스텀 AI 에이전트 통합
- 엔터프라이즈 기능 (SSO, 감사 로그)

## 11. 성공 지표

### 11.1 기술적 지표
- 모든 주요 CLI 도구 정상 작동
- 95% 이상의 터미널 명령어 호환성
- 평균 응답 시간 < 100ms

### 11.2 사용자 지표
- 일일 활성 사용자 100명 이상
- 평균 세션 시간 30분 이상
- 사용자 만족도 4.5/5 이상

---

*작성일: 2024년 12월*
*버전: 1.0*