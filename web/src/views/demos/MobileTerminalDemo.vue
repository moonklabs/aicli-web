<template>
  <div class="mobile-terminal-demo">
    <div class="demo-header">
      <h1>모바일 터미널 데모</h1>
      <p>모바일 최적화된 터미널 인터페이스를 테스트해보세요</p>
    </div>

    <div class="demo-controls">
      <NSpace>
        <NButton @click="openTerminal">터미널 열기</NButton>
        <NButton @click="addLog('output')">로그 추가</NButton>
        <NButton @click="addLog('error')">에러 추가</NButton>
        <NButton @click="toggleConnection">{{ isConnected ? '연결 끊기' : '연결하기' }}</NButton>
        <NButton @click="clearLogs">로그 지우기</NButton>
      </NSpace>
    </div>

    <!-- 로그 미리보기 -->
    <div class="log-preview">
      <h3>로그 미리보기 ({{ logs.length }}개)</h3>
      <div class="log-list">
        <div
          v-for="log in logs.slice(-10)"
          :key="log.id"
          class="log-item"
          :class="`log-${log.type}`"
        >
          <span class="log-time">{{ formatTime(log.timestamp) }}</span>
          <span class="log-content">{{ log.content }}</span>
        </div>
      </div>
    </div>

    <!-- 모바일 터미널 모달 -->
    <NModal
      v-model:show="showTerminal"
      :mask-closable="false"
      :close-on-esc="false"
      :style="{
        width: '100%',
        height: '100%',
        maxWidth: '100vw',
        maxHeight: '100vh',
        margin: 0,
        borderRadius: 0,
      }"
      :body-style="{
        padding: 0,
        height: '100%',
      }"
    >
      <AdaptiveTerminal
        session-id="demo"
        session-name="Demo Terminal"
        :logs="logs"
        :is-connected="isConnected"
        :is-executing="isExecuting"
        :command-history="commandHistory"
        @close="showTerminal = false"
        @execute-command="handleExecuteCommand"
        @stop-execution="handleStopExecution"
        @clear-logs="clearLogs"
        @export-logs="handleExportLogs"
      />
    </NModal>

    <!-- 통계 -->
    <div class="demo-stats">
      <NStatistic label="총 로그" :value="logs.length" />
      <NStatistic label="명령어 실행" :value="commandCount" />
      <NStatistic label="세션 시간" :value="sessionTime" suffix="초" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { NButton, NSpace, NModal, NStatistic } from 'naive-ui'
import { AdaptiveTerminal } from '@/components/Terminal'

interface TerminalLog {
  id: string
  content: string
  type: 'input' | 'output' | 'error' | 'system'
  timestamp: string
}

// State
const showTerminal = ref(false)
const isConnected = ref(true)
const isExecuting = ref(false)
const logs = ref<TerminalLog[]>([])
const commandHistory = ref<string[]>(['ls -la', 'git status', 'npm run dev'])
const commandCount = ref(0)
const sessionTime = ref(0)

// 샘플 로그 메시지
const sampleOutputs = [
  'Building application...',
  'Compiling TypeScript files...',
  'Running tests...',
  'Starting development server...',
  'Server running at http://localhost:3000',
  'Watching for file changes...',
  'Hot module replacement enabled',
  'Ready in 1234ms',
]

const sampleErrors = [
  'Error: Module not found',
  'TypeError: Cannot read property of undefined',
  'SyntaxError: Unexpected token',
  'Warning: Deprecation notice',
  'Error: Connection timeout',
]

// Timer
let sessionTimer: number | null = null

// Methods
const openTerminal = () => {
  showTerminal.value = true
  
  // 초기 시스템 메시지 추가
  if (logs.value.length === 0) {
    addSystemLog('터미널에 연결되었습니다.')
    addSystemLog('도움말을 보려면 "help"를 입력하세요.')
  }
}

const addLog = (type: 'output' | 'error') => {
  const messages = type === 'error' ? sampleErrors : sampleOutputs
  const content = messages[Math.floor(Math.random() * messages.length)]
  
  logs.value.push({
    id: Date.now().toString(),
    content,
    type,
    timestamp: new Date().toISOString(),
  })
}

const addSystemLog = (content: string) => {
  logs.value.push({
    id: Date.now().toString(),
    content,
    type: 'system',
    timestamp: new Date().toISOString(),
  })
}

const toggleConnection = () => {
  isConnected.value = !isConnected.value
  addSystemLog(isConnected.value ? '연결되었습니다.' : '연결이 끊어졌습니다.')
}

const clearLogs = () => {
  logs.value = []
  addSystemLog('터미널이 지워졌습니다.')
}

const handleExecuteCommand = async (command: string) => {
  // 입력 로그 추가
  logs.value.push({
    id: Date.now().toString(),
    content: command,
    type: 'input',
    timestamp: new Date().toISOString(),
  })
  
  commandCount.value++
  isExecuting.value = true
  
  // 명령어 처리 시뮬레이션
  await new Promise(resolve => setTimeout(resolve, 500))
  
  // 명령어에 따른 응답
  switch (command.toLowerCase()) {
    case 'help':
      addSystemLog('사용 가능한 명령어:')
      addSystemLog('  help - 도움말 표시')
      addSystemLog('  clear - 터미널 지우기')
      addSystemLog('  date - 현재 시간 표시')
      addSystemLog('  echo <text> - 텍스트 출력')
      addSystemLog('  error - 에러 시뮬레이션')
      break
      
    case 'clear':
      clearLogs()
      break
      
    case 'date':
      logs.value.push({
        id: Date.now().toString(),
        content: new Date().toLocaleString('ko-KR'),
        type: 'output',
        timestamp: new Date().toISOString(),
      })
      break
      
    case 'error':
      logs.value.push({
        id: Date.now().toString(),
        content: 'Error: Simulated error for testing',
        type: 'error',
        timestamp: new Date().toISOString(),
      })
      break
      
    default:
      if (command.startsWith('echo ')) {
        logs.value.push({
          id: Date.now().toString(),
          content: command.substring(5),
          type: 'output',
          timestamp: new Date().toISOString(),
        })
      } else {
        logs.value.push({
          id: Date.now().toString(),
          content: `명령어를 찾을 수 없습니다: ${command}`,
          type: 'error',
          timestamp: new Date().toISOString(),
        })
      }
  }
  
  isExecuting.value = false
}

const handleStopExecution = () => {
  isExecuting.value = false
  addSystemLog('실행이 중단되었습니다.')
}

const handleExportLogs = () => {
  const logText = logs.value
    .map(log => `[${formatTime(log.timestamp)}] ${log.content}`)
    .join('\n')
  
  const blob = new Blob([logText], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `terminal-logs-${new Date().getTime()}.txt`
  a.click()
  URL.revokeObjectURL(url)
  
  addSystemLog('로그가 내보내졌습니다.')
}

const formatTime = (timestamp: string) => {
  const date = new Date(timestamp)
  return date.toLocaleTimeString('ko-KR', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

// Lifecycle
onMounted(() => {
  // 초기 로그 추가
  addSystemLog('모바일 터미널 데모가 시작되었습니다.')
  addSystemLog('터미널 열기 버튼을 클릭하여 시작하세요.')
  
  // 세션 타이머 시작
  sessionTimer = window.setInterval(() => {
    sessionTime.value++
  }, 1000)
})

onUnmounted(() => {
  if (sessionTimer) {
    clearInterval(sessionTimer)
  }
})
</script>

<style lang="scss" scoped>
.mobile-terminal-demo {
  padding: 20px;
  max-width: 1200px;
  margin: 0 auto;
}

.demo-header {
  margin-bottom: 30px;
  
  h1 {
    font-size: 28px;
    margin-bottom: 10px;
  }
  
  p {
    font-size: 16px;
    color: var(--text-secondary);
  }
}

.demo-controls {
  margin-bottom: 30px;
}

.log-preview {
  background: #1a1a1a;
  border-radius: 8px;
  padding: 20px;
  margin-bottom: 30px;
  
  h3 {
    color: white;
    margin-bottom: 15px;
  }
  
  .log-list {
    font-family: monospace;
    font-size: 14px;
  }
  
  .log-item {
    display: flex;
    gap: 12px;
    margin-bottom: 8px;
    
    &.log-input {
      color: #ffffff;
      
      &::before {
        content: '$ ';
        color: #51cf66;
      }
    }
    
    &.log-output {
      color: #e0e0e0;
    }
    
    &.log-error {
      color: #ff6b6b;
    }
    
    &.log-system {
      color: #74c0fc;
      font-style: italic;
    }
  }
  
  .log-time {
    color: rgba(255, 255, 255, 0.5);
    font-size: 12px;
  }
  
  .log-content {
    flex: 1;
  }
}

.demo-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 20px;
  
  :deep(.n-statistic) {
    background: var(--n-color-modal);
    padding: 20px;
    border-radius: 8px;
    text-align: center;
  }
}

// 모바일 반응형
@media (max-width: 768px) {
  .mobile-terminal-demo {
    padding: 16px;
  }
  
  .demo-header {
    h1 {
      font-size: 24px;
    }
  }
  
  .demo-controls {
    :deep(.n-space) {
      flex-wrap: wrap;
    }
  }
  
  .log-preview {
    padding: 16px;
    
    .log-item {
      font-size: 12px;
    }
  }
}
</style>