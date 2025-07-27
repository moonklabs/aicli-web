# AICode Manager - 멀티 에이전트 플랫폼 구현 계획

## 📋 구현 체크리스트

### Week 1: 에이전트 모델 및 Git Worktree 통합

#### 1. 에이전트 모델 정의
- [ ] `internal/models/agent.go` - 에이전트 모델 구조체
  ```go
  type Agent struct {
      ID          string        `json:"id"`
      ProjectID   string        `json:"project_id"`
      Name        string        `json:"name"`
      Type        AgentType     `json:"type"`
      WorktreeID  string        `json:"worktree_id"`
      ContainerID string        `json:"container_id"`
      SessionID   string        `json:"session_id"`
      Status      AgentStatus   `json:"status"`
      Config      AgentConfig   `json:"config"`
      CreatedAt   time.Time     `json:"created_at"`
      UpdatedAt   time.Time     `json:"updated_at"`
  }
  ```

- [ ] `internal/models/agent_types.go` - 에이전트 관련 타입 정의
  - AgentType (claude-code, gemini-cli, custom)
  - AgentStatus (creating, running, idle, stopped, error)
  - AgentConfig 구조체

#### 2. Git Worktree 관리
- [ ] `internal/git/worktree_manager.go` - Worktree 관리자
  - CreateWorktree(projectID, agentID, branch string)
  - DeleteWorktree(worktreeID string)
  - GetWorktreePath(worktreeID string)
  - ListWorktrees(projectID string)

- [ ] `internal/git/git_client.go` - Git 클라이언트
  - go-git/go-git v5 통합
  - 브랜치 생성/체크아웃
  - Worktree 상태 확인

#### 3. 에이전트 스토리지
- [ ] `internal/storage/interface.go` - AgentStorage 인터페이스 추가
  ```go
  type AgentStorage interface {
      Create(ctx context.Context, agent *models.Agent) error
      GetByID(ctx context.Context, id string) (*models.Agent, error)
      GetByProjectID(ctx context.Context, projectID string) ([]*models.Agent, error)
      Update(ctx context.Context, agent *models.Agent) error
      Delete(ctx context.Context, id string) error
  }
  ```

- [ ] `internal/storage/sqlite/agent.go` - SQLite 구현
- [ ] `internal/storage/sqlite/schema/004_agents.sql` - 마이그레이션

#### 4. 에이전트 서비스
- [ ] `internal/services/agent.go` - 에이전트 비즈니스 로직
  - CreateAgent(req CreateAgentRequest)
  - StartAgent(agentID string)
  - StopAgent(agentID string)
  - GetAgentStatus(agentID string)
  - ExecuteCommand(agentID string, command string)

### Week 2: Docker PTY 통합 및 스트리밍

#### 5. PTY 세션 관리
- [ ] `internal/pty/session.go` - PTY 세션 구조체
  ```go
  type PTYSession struct {
      ID          string
      AgentID     string
      ContainerID string
      PTY         *os.File
      Input       io.WriteCloser
      Output      io.ReadCloser
      ErrorOutput io.ReadCloser
      Size        WindowSize
  }
  ```

- [ ] `internal/pty/manager.go` - PTY 세션 매니저
  - CreateSession(agentID, containerID string)
  - AttachToContainer(containerID string)
  - ResizePTY(sessionID string, size WindowSize)
  - CloseSession(sessionID string)

#### 6. Docker PTY 통합
- [ ] `internal/docker/pty_integration.go` - Docker PTY 통합
  - CreateContainerWithPTY(config ContainerConfig)
  - AttachToPTY(containerID string)
  - 기존 docker 클라이언트 확장

- [ ] `internal/docker/exec_pty.go` - PTY 실행 지원
  - ExecWithPTY(containerID, command string)
  - StreamPTYOutput(session *PTYSession)

#### 7. WebSocket PTY 스트리밍
- [ ] `internal/websocket/pty_handler.go` - PTY WebSocket 핸들러
  ```go
  type PTYMessage struct {
      Type    string      `json:"type"`    // input, output, resize, error
      AgentID string      `json:"agent_id"`
      Data    string      `json:"data,omitempty"`
      Cols    int         `json:"cols,omitempty"`
      Rows    int         `json:"rows,omitempty"`
  }
  ```

- [ ] `internal/websocket/pty_stream.go` - 스트리밍 로직
  - HandlePTYConnection(ws *websocket.Conn, agentID string)
  - StreamToClient(session *PTYSession, ws *websocket.Conn)
  - HandleClientInput(ws *websocket.Conn, session *PTYSession)

#### 8. 터미널 스냅샷
- [ ] `internal/snapshot/terminal_snapshot.go` - 스냅샷 생성
  ```go
  type TerminalSnapshot struct {
      AgentID   string
      Content   []string  // 마지막 N줄
      Cursor    CursorPosition
      Timestamp time.Time
      IsActive  bool
  }
  ```

- [ ] `internal/snapshot/manager.go` - 스냅샷 매니저
  - CaptureSnapshot(agentID string)
  - GetSnapshot(agentID string)
  - 주기적 캡처 스케줄러

### Week 3: 프론트엔드 에이전트 UI

#### 9. Vue 컴포넌트 구조
- [ ] `web/src/views/ProjectDashboardView.vue` - 프로젝트 대시보드
- [ ] `web/src/views/AgentTerminalView.vue` - 전체 화면 터미널

#### 10. 에이전트 관리 컴포넌트
- [ ] `web/src/components/Agent/AgentCard.vue` - 에이전트 카드
  - 터미널 스냅샷 표시
  - 상태 인디케이터
  - 빠른 액션 버튼

- [ ] `web/src/components/Agent/AgentGrid.vue` - 에이전트 그리드
  - 반응형 레이아웃
  - 필터링/정렬 기능

- [ ] `web/src/components/Agent/CreateAgentModal.vue` - 에이전트 생성
  - 타입 선택 (Claude/Gemini/Custom)
  - 초기 스크립트 설정
  - 환경 변수 설정

#### 11. PTY 터미널 인터페이스
- [ ] `web/src/components/Agent/AgentTerminal.vue` - 메인 터미널
  - xterm.js 통합 업그레이드
  - PTY 지원 추가

- [ ] `web/src/components/Agent/TerminalSnapshot.vue` - 스냅샷 렌더러
  - ANSI 코드 파싱
  - 미니 터미널 뷰

#### 12. 상태 관리 및 API 통합
- [ ] `web/src/stores/agent.ts` - 에이전트 상태 관리
  ```typescript
  interface AgentState {
    agents: Map<string, Agent>
    activeAgentId: string | null
    snapshots: Map<string, TerminalSnapshot>
  }
  ```

- [ ] `web/src/api/services/agent.ts` - 에이전트 API 클라이언트
- [ ] `web/src/composables/usePTYStream.ts` - PTY WebSocket 훅

### Week 4: 모바일 최적화 및 고급 기능

#### 13. 커스텀 프롬프트 시스템
- [ ] `internal/models/prompt.go` - 프롬프트 모델
- [ ] `web/src/components/Agent/CustomPromptManager.vue` - 프롬프트 관리
- [ ] `web/src/components/Agent/PromptExecutor.vue` - 프롬프트 실행

#### 14. 모바일 UI
- [ ] `web/src/components/Mobile/MobileAgentView.vue` - 모바일 뷰
- [ ] `web/src/components/Mobile/MobileTerminal.vue` - 모바일 터미널
- [ ] `web/src/utils/mobile-terminal.ts` - 모바일 최적화 유틸

#### 15. API 핸들러 구현
- [ ] `internal/server/handlers/agent.go` - 에이전트 핸들러
  - CreateAgent
  - ListAgents
  - GetAgent
  - UpdateAgent
  - DeleteAgent
  - StartAgent
  - StopAgent

- [ ] `internal/server/handlers/pty.go` - PTY 핸들러
  - StartPTYSession
  - HandlePTYWebSocket
  - ResizePTY
  - GetSnapshot

#### 16. 통합 테스트
- [ ] `internal/services/agent_test.go` - 서비스 테스트
- [ ] `internal/websocket/pty_handler_test.go` - WebSocket 테스트
- [ ] `web/src/components/Agent/__tests__/` - 컴포넌트 테스트

## 🔧 기술적 구현 세부사항

### Docker 컨테이너 설정
```go
// 에이전트별 Docker 설정
type AgentContainerConfig struct {
    Image        string            // 기본: "ubuntu:22.04"
    WorkingDir   string            // Worktree 경로
    Environment  map[string]string // 환경 변수
    Volumes      []string          // 마운트할 볼륨
    Resources    ResourceLimits    // CPU/메모리 제한
    InitCommand  []string          // 초기 실행 명령
}
```

### PTY 크기 동기화
```javascript
// 프론트엔드 터미널 크기 감지
const fitAddon = new FitAddon();
xterm.loadAddon(fitAddon);

// 크기 변경 시 서버에 전송
xterm.onResize((size) => {
  ws.send(JSON.stringify({
    type: 'pty_resize',
    agent_id: agentId,
    cols: size.cols,
    rows: size.rows
  }));
});
```

### 스냅샷 최적화
```go
// 효율적인 스냅샷 저장
type SnapshotBuffer struct {
    lines    []string
    maxLines int
    cursor   CursorPosition
}

// 순환 버퍼로 메모리 효율성 확보
func (sb *SnapshotBuffer) AddLine(line string) {
    if len(sb.lines) >= sb.maxLines {
        sb.lines = sb.lines[1:]
    }
    sb.lines = append(sb.lines, line)
}
```

## 📊 성공 지표 측정

### 기술적 메트릭
- [ ] PTY 응답 시간 < 50ms (P95)
- [ ] 스냅샷 생성 시간 < 10ms
- [ ] WebSocket 재연결 시간 < 1초
- [ ] 동시 에이전트 100개 테스트 통과

### 사용성 메트릭
- [ ] 에이전트 생성 시간 < 5초
- [ ] 터미널 연결 시간 < 2초
- [ ] 모바일 터치 응답 < 100ms

## 🚀 배포 준비

### 인프라 체크리스트
- [ ] Docker 이미지 빌드 파이프라인
- [ ] 환경 변수 설정 문서화
- [ ] 리소스 제한 가이드라인
- [ ] 모니터링 대시보드 설정

### 문서화
- [ ] API 문서 업데이트
- [ ] 사용자 가이드 작성
- [ ] 관리자 가이드 작성
- [ ] 트러블슈팅 가이드

---

*마지막 업데이트: 2024년 12월*