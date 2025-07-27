# SPECS: Frontend UI

## 개요

멀티 에이전트 플랫폼의 프론트엔드 UI 기술 사양

## 컴포넌트 아키텍처

### 주요 뷰
```vue
<!-- 프로젝트 대시보드 -->
<ProjectDashboardView>
  <ProjectInfo />
  <AgentGrid>
    <AgentCard v-for="agent in agents">
      <TerminalSnapshot />
      <AgentStatus />
      <QuickActions />
    </AgentCard>
  </AgentGrid>
</ProjectDashboardView>

<!-- 에이전트 터미널 -->
<AgentTerminalView>
  <PTYTerminal />
  <CustomPromptBar />
  <AgentControls />
</AgentTerminalView>
```

### 에이전트 관리 컴포넌트
- `AgentCard.vue` - 에이전트 카드 (스냅샷 포함)
- `AgentGrid.vue` - 에이전트 그리드 레이아웃
- `CreateAgentModal.vue` - 에이전트 생성 모달
- `AgentControls.vue` - 에이전트 제어 UI

### 터미널 인터페이스
- `AgentTerminal.vue` - 메인 터미널 (xterm.js 기반)
- `TerminalSnapshot.vue` - 스냅샷 렌더러
- `PTYTerminal.vue` - PTY 통합 터미널

## 상태 관리

### Agent Store
```typescript
interface AgentState {
  agents: Map<string, Agent>
  activeAgentId: string | null
  snapshots: Map<string, TerminalSnapshot>
  connectionStatus: Map<string, ConnectionStatus>
}
```

### API 통합
```typescript
// 에이전트 API 클라이언트
class AgentAPI {
  createAgent(projectId: string, config: AgentConfig): Promise<Agent>
  listAgents(projectId: string): Promise<Agent[]>
  getAgent(agentId: string): Promise<Agent>
  startAgent(agentId: string): Promise<void>
  stopAgent(agentId: string): Promise<void>
}
```

## 실시간 통신

### WebSocket 통합
```typescript
// PTY 스트리밍 훅
const usePTYStream = (agentId: string) => {
  const { connect, disconnect, send } = useWebSocket()
  const terminalRef = ref<Terminal>()
  
  const connectToAgent = () => {
    connect(`/api/agents/${agentId}/pty/stream`)
  }
  
  return { connectToAgent, terminalRef }
}
```

### xterm.js 통합
- PTY 지원 추가
- ANSI 코드 완벽 지원
- 터미널 크기 자동 조정
- 모바일 최적화

## 모바일 UI

### 반응형 디자인
- 터치 제스처 지원
- 가상 키보드 최적화
- 스와이프 네비게이션
- 적응형 레이아웃

### 모바일 컴포넌트
- `MobileAgentView.vue` - 모바일 에이전트 뷰
- `MobileTerminal.vue` - 모바일 터미널
- `TouchKeyboard.vue` - 터치 키보드

## 커스텀 프롬프트

### 프롬프트 관리
```typescript
interface CustomPrompt {
  id: string
  name: string
  template: string
  variables: Variable[]
  shortcut?: string
}
```

### 컴포넌트
- `CustomPromptManager.vue` - 프롬프트 관리
- `PromptExecutor.vue` - 프롬프트 실행
- `PromptEditor.vue` - 프롬프트 편집

## 성능 요구사항

- 터미널 연결 시간 < 2초
- 모바일 터치 응답 < 100ms
- 스냅샷 렌더링 < 50ms
- UI 반응 시간 < 16ms (60fps)

## 파일 구조
```
web/src/
├── views/
│   ├── ProjectDashboardView.vue
│   └── AgentTerminalView.vue
├── components/
│   ├── Agent/
│   │   ├── AgentCard.vue
│   │   ├── AgentGrid.vue
│   │   └── AgentTerminal.vue
│   └── Mobile/
│       ├── MobileAgentView.vue
│       └── MobileTerminal.vue
├── stores/
│   └── agent.ts
├── composables/
│   └── usePTYStream.ts
└── api/
    └── services/agent.ts
```