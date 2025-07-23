<template>
  <div class="terminal-view">
    <header class="terminal-header">
      <div class="terminal-tabs">
        <div
          v-for="session in sessions"
          :key="session.id"
          class="terminal-tab"
          :class="{ active: session.id === activeSessionId }"
          @click="selectSession(session.id)"
        >
          <span class="tab-title">{{ session.title }}</span>
          <NBadge
            :type="getStatusType(session.status)"
            dot
            class="tab-status"
          />
          <NButton
            text
            size="tiny"
            @click.stop="closeSession(session.id)"
            class="tab-close"
          >
            ×
          </NButton>
        </div>
        <NButton
          text
          size="small"
          @click="createNewSession"
          class="new-tab-btn"
        >
          +
        </NButton>
      </div>

      <div class="terminal-actions">
        <NButton
          text
          size="small"
          @click="clearTerminal"
          :disabled="!activeSession"
        >
          Clear
        </NButton>
        <NButton
          text
          size="small"
          @click="reconnectSession"
          :disabled="!activeSession || activeSession.status === 'connected'"
        >
          Reconnect
        </NButton>
      </div>
    </header>

    <div class="terminal-content">
      <div v-if="activeSession" class="terminal-container">
        <TerminalEmulator
          :session-id="activeSession.id"
          :session-name="activeSession.title"
          :logs="activeSession.logs"
          :is-connected="activeSession.status === 'connected'"
          :is-executing="activeSession.claudeStream?.isClaudeRunning || false"
          :last-activity="activeSession.lastActivity"
          :use-virtual-scrolling="true"
          :line-height="24"
          :max-lines="1000"
          :auto-scroll="true"
          @execute-command="handleExecuteCommand"
          @stop-execution="handleStopExecution"
          @clear-logs="clearTerminal"
          @export-logs="handleExportLogs"
        />
      </div>

      <div v-else class="empty-terminal">
        <NEmpty description="터미널 세션이 없습니다">
          <template #extra>
            <NButton type="primary" @click="createNewSession">
              새 터미널 세션 생성
            </NButton>
          </template>
        </NEmpty>
      </div>
    </div>

    <!-- 세션 생성 모달 -->
    <NModal v-model:show="showCreateModal" preset="dialog" title="새 터미널 세션">
      <NForm ref="createFormRef" :model="createForm" label-placement="top">
        <NFormItem label="워크스페이스">
          <NSelect
            v-model:value="createForm.workspaceId"
            :options="workspaceOptions"
            placeholder="워크스페이스를 선택하세요"
          />
        </NFormItem>
        <NFormItem label="세션 이름 (선택사항)">
          <NInput
            v-model:value="createForm.title"
            placeholder="터미널 세션 이름"
          />
        </NFormItem>
      </NForm>
      <template #action>
        <NSpace>
          <NButton @click="showCreateModal = false">취소</NButton>
          <NButton type="primary" @click="handleCreateSession">생성</NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import {
  NBadge,
  NButton,
  NEmpty,
  NForm,
  NFormItem,
  NInput,
  NModal,
  NSelect,
  NSpace,
} from 'naive-ui'
import { useTerminalStore } from '@/stores/terminal'
import { useWorkspaceStore } from '@/stores/workspace'
import type { TerminalSession } from '@/stores/terminal'
import TerminalEmulator from '@/components/Terminal/TerminalEmulator.vue'

const route = useRoute()
const router = useRouter()
const message = useMessage()
const terminalStore = useTerminalStore()
const workspaceStore = useWorkspaceStore()

// 상태
const showCreateModal = ref(false)
const createForm = ref({
  workspaceId: '',
  title: '',
})

const sessionId = computed(() => route.params.sessionId as string | undefined)
const sessions = computed(() => terminalStore.sessions)
const activeSession = computed(() =>
  sessionId.value
    ? terminalStore.sessionById(sessionId.value)
    : sessions.value[0] || null,
)
const activeSessionId = computed(() => activeSession.value?.id || null)

const workspaceOptions = computed(() =>
  workspaceStore.activeWorkspaces.map(ws => ({
    label: ws.name,
    value: ws.id,
  })),
)

// 메서드 (제거됨 - TerminalEmulator 내부에서 처리)

const getStatusType = (status: TerminalSession['status']) => {
  switch (status) {
    case 'connected':
      return 'success'
    case 'disconnected':
      return 'default'
    case 'error':
      return 'error'
    case 'connecting':
      return 'warning'
    default:
      return 'default'
  }
}

const selectSession = (id: string) => {
  router.push({ name: 'terminal', params: { sessionId: id } })
}

const closeSession = async (id: string) => {
  try {
    await terminalStore.disconnectSession(id)
    terminalStore.removeSession(id)

    // 현재 세션이 닫혔으면 다른 세션으로 이동
    if (id === activeSessionId.value) {
      const remainingSessions = sessions.value.filter(s => s.id !== id)
      if (remainingSessions.length > 0) {
        selectSession(remainingSessions[0].id)
      } else {
        router.push({ name: 'terminal' })
      }
    }
  } catch (_error) {
    message.error('터미널 세션 종료에 실패했습니다')
  }
}

const createNewSession = () => {
  if (workspaceOptions.value.length === 0) {
    message.error('활성화된 워크스페이스가 없습니다')
    return
  }

  createForm.value.workspaceId = workspaceOptions.value[0].value
  createForm.value.title = ''
  showCreateModal.value = true
}

const handleCreateSession = async () => {
  try {
    const session = await terminalStore.createSession(
      createForm.value.workspaceId,
      createForm.value.title,
    )

    if (session) {
      showCreateModal.value = false
      selectSession(session.id)
      message.success('터미널 세션이 생성되었습니다')
    }
  } catch (_error) {
    message.error('터미널 세션 생성에 실패했습니다')
  }
}

const clearTerminal = () => {
  if (activeSession.value) {
    terminalStore.clearLogs(activeSession.value.id)
  }
}

const reconnectSession = async () => {
  if (activeSession.value) {
    try {
      const success = await terminalStore.reconnectSession(activeSession.value.id)
      if (success) {
        message.success('터미널이 재연결되었습니다')
      } else {
        message.error('터미널 재연결에 실패했습니다')
      }
    } catch (_error) {
      message.error('터미널 재연결에 실패했습니다')
    }
  }
}

const handleExecuteCommand = async (command: string) => {
  if (!activeSession.value) return

  try {
    await terminalStore.executeCommand(activeSession.value.id, {
      command,
      workingDir: activeSession.value.workspaceId, // 실제로는 워크스페이스 경로
    })
  } catch (_error) {
    message.error('명령 실행에 실패했습니다')
  }
}

const handleStopExecution = () => {
  if (!activeSession.value) return

  try {
    terminalStore.stopExecution(activeSession.value.id)
    message.success('명령 실행이 중단되었습니다')
  } catch (_error) {
    message.error('명령 중단에 실패했습니다')
  }
}

const handleExportLogs = (format: 'text' | 'html' | 'json') => {
  if (!activeSession.value) return

  try {
    const logs = activeSession.value.logs
    let content = ''
    let filename = `terminal-${activeSession.value.id}-${new Date().toISOString().slice(0, 10)}`
    let mimeType = 'text/plain'

    switch (format) {
      case 'text':
        content = logs.map(log => {
          const timestamp = new Date(log.timestamp).toLocaleTimeString()
          const prefix = log.type === 'input' ? '$ ' : ''
          return `[${timestamp}] ${prefix}${log.content}`
        }).join('\n')
        filename += '.txt'
        break

      case 'html':
        content = `<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <title>Terminal Log - ${activeSession.value.title}</title>
  <style>
    body { font-family: 'Monaco', 'Menlo', monospace; background: #1a1a1a; color: #fff; }
    .log-line { margin-bottom: 4px; }
    .log-timestamp { color: #666; margin-right: 8px; }
    .log-input { color: #0f0; }
    .log-error { color: #f66; }
    .log-system { color: #66f; font-style: italic; }
  </style>
</head>
<body>
  <h1>Terminal Log: ${activeSession.value.title}</h1>
  <div class="terminal-content">
    ${logs.map(log => `
      <div class="log-line log-${log.type}">
        <span class="log-timestamp">${new Date(log.timestamp).toLocaleTimeString()}</span>
        <span class="log-content">${log.parsed?.html || log.content}</span>
      </div>
    `).join('')}
  </div>
</body>
</html>`
        filename += '.html'
        mimeType = 'text/html'
        break

      case 'json':
        content = JSON.stringify({
          session: {
            id: activeSession.value.id,
            title: activeSession.value.title,
            workspaceId: activeSession.value.workspaceId,
            createdAt: activeSession.value.createdAt,
          },
          logs,
          exportedAt: new Date().toISOString(),
        }, null, 2)
        filename += '.json'
        mimeType = 'application/json'
        break
    }

    // 파일 다운로드
    const blob = new Blob([content], { type: mimeType })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)

    message.success(`로그가 ${format.toUpperCase()} 형식으로 내보내졌습니다`)
  } catch (_error) {
    message.error('로그 내보내기에 실패했습니다')
  }
}

// 스크롤 함수는 TerminalEmulator 내부에서 처리됨

// 로그 변경 감지는 TerminalEmulator에서 처리됨

// 라이프사이클
onMounted(() => {
  // 세션 ID가 주어졌지만 해당 세션이 없으면 새로 생성하거나 리다이렉트
  if (sessionId.value && !activeSession.value) {
    message.warning('터미널 세션을 찾을 수 없습니다')
    router.push({ name: 'terminal' })
  }

  // 입력창 포커스는 TerminalEmulator에서 처리됨
})

onUnmounted(() => {
  // 정리 작업이 필요하면 여기서 수행
})
</script>

<style lang="scss" scoped>
.terminal-view {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background-color: var(--terminal-bg, #1a1a1a);
  color: var(--terminal-text, #ffffff);
}

.terminal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.5rem 1rem;
  background-color: var(--header-bg, #2a2a2a);
  border-bottom: 1px solid var(--border-color, #3a3a3a);

  .terminal-tabs {
    display: flex;
    align-items: center;
    gap: 0.25rem;
  }

  .terminal-tab {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.5rem 1rem;
    background-color: var(--tab-bg, #333);
    border-radius: 6px 6px 0 0;
    cursor: pointer;
    transition: background-color 0.2s ease;
    border-bottom: 2px solid transparent;

    &.active {
      background-color: var(--terminal-bg, #1a1a1a);
      border-bottom-color: var(--primary-color, #18a058);
    }

    &:hover:not(.active) {
      background-color: var(--tab-hover-bg, #404040);
    }

    .tab-title {
      font-size: 0.875rem;
    }

    .tab-close {
      opacity: 0;
      transition: opacity 0.2s ease;
      &:hover {
        color: var(--error-color, #d03050);
      }
    }

    &:hover .tab-close {
      opacity: 1;
    }
  }

  .new-tab-btn {
    padding: 0.5rem 0.75rem;
    font-size: 1.25rem;
    line-height: 1;
  }

  .terminal-actions {
    display: flex;
    gap: 0.5rem;
  }
}

.terminal-content {
  flex: 1;
  overflow: hidden;
}

.terminal-container {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.terminal-output {
  flex: 1;
  overflow-y: auto;
  padding: 1rem;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 14px;
  line-height: 1.4;

  .terminal-line {
    display: flex;
    margin-bottom: 0.25rem;
    word-break: break-all;

    &.log-input {
      color: var(--terminal-input, #ffffff);
      .log-content::before {
        content: '$ ';
        color: var(--primary-color, #18a058);
        font-weight: bold;
      }
    }

    &.log-output {
      color: var(--terminal-output, #e0e0e0);
    }

    &.log-error {
      color: var(--error-color, #ff6b6b);
    }

    &.log-system {
      color: var(--warning-color, #ffa502);
      font-style: italic;
    }

    .log-timestamp {
      color: var(--text-color-3, #666);
      font-size: 0.8em;
      margin-right: 0.5rem;
      min-width: 60px;
      user-select: none;
    }

    .log-content {
      flex: 1;
      white-space: pre-wrap;
    }
  }
}

.terminal-input {
  display: flex;
  align-items: center;
  padding: 1rem;
  background-color: var(--input-bg, #2a2a2a);
  border-top: 1px solid var(--border-color, #3a3a3a);

  .input-prompt {
    margin-right: 0.5rem;
    .prompt-symbol {
      color: var(--primary-color, #18a058);
      font-weight: bold;
      font-family: monospace;
    }
  }

  .command-input {
    flex: 1;
    :deep(.n-input__input-el) {
      background-color: transparent;
      border: none;
      color: var(--terminal-text, #ffffff);
      font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
      font-size: 14px;

      &::placeholder {
        color: var(--text-color-3, #666);
      }
    }

    :deep(.n-input__border),
    :deep(.n-input__state-border) {
      display: none;
    }
  }
}

.empty-terminal {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}
</style>