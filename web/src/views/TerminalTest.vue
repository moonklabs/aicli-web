<template>
  <div class="terminal-test">
    <div class="test-header">
      <h1>터미널 인터페이스 테스트</h1>
      <div class="test-controls">
        <NButton @click="generateTestData" type="primary">테스트 데이터 생성</NButton>
        <NButton @click="clearTestData">데이터 클리어</NButton>
        <NButton @click="simulateError" type="error">에러 시뮬레이션</NButton>
        <NButton @click="simulateHeavyLoad">대용량 데이터 테스트</NButton>
      </div>
    </div>

    <div class="test-layout">
      <!-- 터미널 인터페이스 -->
      <div class="terminal-section">
        <TerminalInterface
          :session="testSession"
          :logs="testLogs"
          :auto-scroll="settings.autoScroll"
          :show-timestamp="settings.showTimestamp"
          :font-size="settings.fontSize"
          :line-height="settings.lineHeight"
          :max-lines="settings.maxLines"
          :show-performance-info="settings.showPerformanceInfo"
          @command="handleCommand"
          @clear="handleClear"
          @stop="handleStop"
          @connect="handleConnect"
          @reconnect="handleReconnect"
          @disconnect="handleDisconnect"
          @settings-change="handleSettingsChange"
        />
      </div>

      <!-- 테스트 정보 패널 -->
      <div class="info-panel">
        <NCard title="테스트 정보" size="small">
          <div class="info-grid">
            <div class="info-item">
              <span class="info-label">세션 상태:</span>
              <NTag :type="getSessionStatusType(testSession.status)" size="small">
                {{ testSession.status }}
              </NTag>
            </div>
            <div class="info-item">
              <span class="info-label">로그 수:</span>
              <span class="info-value">{{ testLogs.length }}</span>
            </div>
            <div class="info-item">
              <span class="info-label">마지막 명령:</span>
              <span class="info-value">{{ lastCommand || 'None' }}</span>
            </div>
            <div class="info-item">
              <span class="info-label">실행 시간:</span>
              <span class="info-value">{{ formatDuration(sessionDuration) }}</span>
            </div>
          </div>
        </NCard>

        <NCard title="설정" size="small" style="margin-top: 16px;">
          <div class="settings-grid">
            <div class="setting-item">
              <NCheckbox
                v-model:checked="settings.autoScroll"
                @update:checked="updateSetting('autoScroll', $event)"
              >
                자동 스크롤
              </NCheckbox>
            </div>
            <div class="setting-item">
              <NCheckbox
                v-model:checked="settings.showTimestamp"
                @update:checked="updateSetting('showTimestamp', $event)"
              >
                타임스탬프 표시
              </NCheckbox>
            </div>
            <div class="setting-item">
              <NCheckbox
                v-model:checked="settings.showPerformanceInfo"
                @update:checked="updateSetting('showPerformanceInfo', $event)"
              >
                성능 정보 표시
              </NCheckbox>
            </div>
            <div class="setting-item">
              <label>폰트 크기: {{ settings.fontSize }}px</label>
              <NSlider
                v-model:value="settings.fontSize"
                :min="10"
                :max="20"
                :step="1"
                @update:value="updateSetting('fontSize', $event)"
              />
            </div>
            <div class="setting-item">
              <label>라인 높이: {{ settings.lineHeight }}</label>
              <NSlider
                v-model:value="settings.lineHeight"
                :min="1.0"
                :max="2.0"
                :step="0.1"
                @update:value="updateSetting('lineHeight', $event)"
              />
            </div>
            <div class="setting-item">
              <label>최대 라인: {{ settings.maxLines }}</label>
              <NSlider
                v-model:value="settings.maxLines"
                :min="100"
                :max="5000"
                :step="100"
                @update:value="updateSetting('maxLines', $event)"
              />
            </div>
          </div>
        </NCard>

        <NCard title="테스트 시나리오" size="small" style="margin-top: 16px;">
          <div class="scenario-buttons">
            <NButton @click="runScenario('basic')" size="small" block>
              기본 명령 테스트
            </NButton>
            <NButton @click="runScenario('ansi')" size="small" block>
              ANSI 색상 테스트
            </NButton>
            <NButton @click="runScenario('long-output')" size="small" block>
              긴 출력 테스트
            </NButton>
            <NButton @click="runScenario('error-handling')" size="small" block>
              에러 처리 테스트
            </NButton>
            <NButton @click="runScenario('performance')" size="small" block>
              성능 테스트
            </NButton>
            <NButton @click="runScenario('interactive')" size="small" block>
              대화형 명령 테스트
            </NButton>
          </div>
        </NCard>
      </div>
    </div>

    <!-- 디버그 로그 -->
    <div v-if="showDebugLog" class="debug-log">
      <NCard title="디버그 로그" size="small">
        <div class="debug-entries">
          <div
            v-for="(entry, index) in debugLog"
            :key="index"
            class="debug-entry"
            :class="[`debug-${entry.type}`]"
          >
            <span class="debug-time">{{ formatTime(entry.timestamp) }}</span>
            <span class="debug-message">{{ entry.message }}</span>
          </div>
        </div>
      </NCard>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, reactive, ref } from 'vue'
import { NButton, NCard, NCheckbox, NSlider, NTag } from 'naive-ui'
import TerminalInterface from '@/components/Terminal/TerminalInterface.vue'
import type { TerminalLog, TerminalSession } from '@/stores/terminal'

// 상태
const testSession = reactive<TerminalSession>({
  id: 'test-session-001',
  workspaceId: 'test-workspace',
  title: 'Terminal Test Session',
  status: 'disconnected',
  logs: [],
  createdAt: new Date().toISOString(),
  lastActivity: new Date().toISOString(),
})

const testLogs = ref<TerminalLog[]>([])
const lastCommand = ref('')
const sessionStartTime = ref(Date.now())
const sessionDuration = ref(0)
const showDebugLog = ref(true)

const settings = reactive({
  autoScroll: true,
  showTimestamp: true,
  showPerformanceInfo: true,
  fontSize: 14,
  lineHeight: 1.4,
  maxLines: 1000,
})

const debugLog = ref<Array<{
  timestamp: number
  type: 'info' | 'warn' | 'error' | 'success'
  message: string
}>>([])

// 타이머
let durationTimer: NodeJS.Timeout | null = null

// 메서드
const addLog = (content: string, type: TerminalLog['type'] = 'output', level: TerminalLog['level'] = 'info') => {
  const log: TerminalLog = {
    id: `log-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
    timestamp: new Date().toISOString(),
    type,
    content,
    level,
  }

  testLogs.value.push(log)
  testSession.lastActivity = log.timestamp

  addDebugLog('info', `Added ${type} log: ${content.slice(0, 50)}...`)
}

const addDebugLog = (type: 'info' | 'warn' | 'error' | 'success', message: string) => {
  debugLog.value.push({
    timestamp: Date.now(),
    type,
    message,
  })

  // 최대 100개 항목 유지
  if (debugLog.value.length > 100) {
    debugLog.value = debugLog.value.slice(-100)
  }
}

const generateTestData = () => {
  addDebugLog('info', 'Generating test data...')

  const commands = [
    'ls -la',
    'pwd',
    'echo "Hello, World!"',
    'cat /etc/passwd | head -5',
    'ps aux | grep node',
    'npm --version',
    'git status',
    'docker ps',
  ]

  const outputs = [
    'total 42\ndrwxr-xr-x  5 user user 4096 Jan 20 10:30 .\ndrwxr-xr-x  3 user user 4096 Jan 20 10:00 ..',
    '/home/user/workspace/aicli-web',
    'Hello, World!',
    'root:x:0:0:root:/root:/bin/bash\ndaemon:x:1:1:daemon:/usr/sbin:/usr/sbin/nologin',
    'user      1234  0.1  2.3 123456 45678 ?        Ssl  10:30   0:01 node server.js',
    '9.5.0',
    'On branch main\nnothing to commit, working tree clean',
    'CONTAINER ID   IMAGE     COMMAND   CREATED   STATUS   PORTS   NAMES',
  ]

  commands.forEach((cmd, index) => {
    setTimeout(() => {
      addLog(`$ ${cmd}`, 'input')
      setTimeout(() => {
        addLog(outputs[index] || 'Command executed successfully', 'output')
      }, 100 + Math.random() * 300)
    }, index * 500)
  })

  addDebugLog('success', 'Test data generation completed')
}

const clearTestData = () => {
  testLogs.value = []
  lastCommand.value = ''
  addDebugLog('info', 'Test data cleared')
}

const simulateError = () => {
  addDebugLog('warn', 'Simulating error scenario...')
  addLog('$ invalid-command', 'input')
  setTimeout(() => {
    addLog('bash: invalid-command: command not found', 'error')
    addLog('Exit code: 127', 'system', 'error')
  }, 200)
}

const simulateHeavyLoad = () => {
  addDebugLog('warn', 'Starting heavy load test...')

  for (let i = 0; i < 500; i++) {
    setTimeout(() => {
      const logTypes: TerminalLog['type'][] = ['output', 'error', 'system']
      const type = logTypes[Math.floor(Math.random() * logTypes.length)]
      const content = `Heavy load test line ${i + 1}: ${'Lorem ipsum '.repeat(Math.floor(Math.random() * 10) + 1)}`
      addLog(content, type)
    }, i * 10)
  }

  setTimeout(() => {
    addDebugLog('success', 'Heavy load test completed')
  }, 5000)
}

const runScenario = (scenario: string) => {
  addDebugLog('info', `Running scenario: ${scenario}`)

  switch (scenario) {
    case 'basic':
      runBasicScenario()
      break
    case 'ansi':
      runAnsiScenario()
      break
    case 'long-output':
      runLongOutputScenario()
      break
    case 'error-handling':
      runErrorHandlingScenario()
      break
    case 'performance':
      runPerformanceScenario()
      break
    case 'interactive':
      runInteractiveScenario()
      break
  }
}

const runBasicScenario = () => {
  const steps = [
    () => addLog('$ echo "Starting basic test"', 'input'),
    () => addLog('Starting basic test', 'output'),
    () => addLog('$ date', 'input'),
    () => addLog(new Date().toString(), 'output'),
    () => addLog('$ whoami', 'input'),
    () => addLog('testuser', 'output'),
    () => addLog('Basic test completed', 'system', 'info'),
  ]

  steps.forEach((step, index) => {
    setTimeout(step, index * 300)
  })
}

const runAnsiScenario = () => {
  const ansiCommands = [
    '$ echo -e "\\033[31mRed text\\033[0m"',
    '\x1b[31mRed text\x1b[0m',
    '$ echo -e "\\033[32mGreen text\\033[0m"',
    '\x1b[32mGreen text\x1b[0m',
    '$ echo -e "\\033[33mYellow text\\033[0m"',
    '\x1b[33mYellow text\x1b[0m',
    '$ echo -e "\\033[1mBold text\\033[0m"',
    '\x1b[1mBold text\x1b[0m',
    '$ echo -e "\\033[4mUnderlined text\\033[0m"',
    '\x1b[4mUnderlined text\x1b[0m',
  ]

  ansiCommands.forEach((cmd, index) => {
    setTimeout(() => {
      addLog(cmd, index % 2 === 0 ? 'input' : 'output')
    }, index * 400)
  })
}

const runLongOutputScenario = () => {
  addLog('$ cat large-file.txt', 'input')

  for (let i = 0; i < 100; i++) {
    setTimeout(() => {
      const lineNum = String(i + 1).padStart(3, '0')
      const content = `${lineNum}: This is a very long line of text that demonstrates how the terminal handles long output content. Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.`
      addLog(content, 'output')
    }, i * 50)
  }
}

const runErrorHandlingScenario = () => {
  const errorSteps = [
    () => addLog('$ command-that-fails', 'input'),
    () => addLog('command-that-fails: command not found', 'error'),
    () => addLog('$ python script-with-error.py', 'input'),
    () => addLog('Traceback (most recent call last):', 'error'),
    () => addLog('  File "script-with-error.py", line 1, in <module>', 'error'),
    () => addLog('    undefined_variable', 'error'),
    () => addLog('NameError: name \'undefined_variable\' is not defined', 'error'),
    () => addLog('Process exited with code 1', 'system', 'error'),
  ]

  errorSteps.forEach((step, index) => {
    setTimeout(step, index * 400)
  })
}

const runPerformanceScenario = () => {
  addLog('$ performance-test --iterations=1000', 'input')
  addLog('Starting performance test...', 'system', 'info')

  for (let i = 0; i < 200; i++) {
    setTimeout(() => {
      addLog(`Iteration ${i + 1}/200: Processing data... ${Math.random().toFixed(6)}ms`, 'output')
    }, i * 25)
  }

  setTimeout(() => {
    addLog('Performance test completed successfully', 'system', 'info')
    addLog('Average processing time: 0.123ms', 'system', 'info')
  }, 200 * 25 + 500)
}

const runInteractiveScenario = () => {
  const steps = [
    () => addLog('$ interactive-command', 'input'),
    () => addLog('Starting interactive mode...', 'system', 'info'),
    () => addLog('Please enter your name: ', 'output'),
    () => addLog('John Doe', 'input'),
    () => addLog('Hello, John Doe!', 'output'),
    () => addLog('Enter a number (1-10): ', 'output'),
    () => addLog('7', 'input'),
    () => addLog('You entered: 7', 'output'),
    () => addLog('Interactive session completed.', 'system', 'info'),
  ]

  steps.forEach((step, index) => {
    setTimeout(step, index * 800)
  })
}

const handleCommand = (command: string) => {
  lastCommand.value = command
  addDebugLog('info', `Command executed: ${command}`)
  addLog(`$ ${command}`, 'input')

  setTimeout(() => {
    if (command.includes('help')) {
      addLog('Available commands: help, echo, date, ls, clear', 'output')
    } else if (command.includes('echo')) {
      const text = command.replace('echo', '').trim()
      addLog(text || 'echo', 'output')
    } else if (command.includes('date')) {
      addLog(new Date().toString(), 'output')
    } else if (command.includes('ls')) {
      addLog('index.html  style.css  script.js  README.md', 'output')
    } else {
      addLog(`Executed: ${command}`, 'output')
    }
  }, 200 + Math.random() * 300)
}

const handleClear = () => {
  clearTestData()
  addDebugLog('info', 'Terminal cleared')
}

const handleStop = () => {
  addDebugLog('warn', 'Command execution stopped')
  addLog('Process interrupted by user', 'system', 'warn')
}

const handleConnect = () => {
  testSession.status = 'connecting'
  addDebugLog('info', 'Connecting to terminal session...')

  setTimeout(() => {
    testSession.status = 'connected'
    addDebugLog('success', 'Connected to terminal session')
    addLog('Terminal session connected', 'system', 'info')
  }, 1000 + Math.random() * 2000)
}

const handleReconnect = () => {
  testSession.status = 'connecting'
  addDebugLog('info', 'Reconnecting to terminal session...')

  setTimeout(() => {
    testSession.status = 'connected'
    addDebugLog('success', 'Reconnected to terminal session')
    addLog('Terminal session reconnected', 'system', 'info')
  }, 800 + Math.random() * 1200)
}

const handleDisconnect = () => {
  testSession.status = 'disconnected'
  addDebugLog('info', 'Disconnected from terminal session')
  addLog('Terminal session disconnected', 'system', 'warn')
}

const handleSettingsChange = (newSettings: Record<string, any>) => {
  Object.assign(settings, newSettings)
  addDebugLog('info', `Settings updated: ${JSON.stringify(newSettings)}`)
}

const updateSetting = (key: string, value: any) => {
  ;(settings as any)[key] = value
  addDebugLog('info', `Setting ${key} updated to: ${value}`)
}

const getSessionStatusType = (status: string) => {
  switch (status) {
    case 'connected':
      return 'success'
    case 'connecting':
      return 'warning'
    case 'error':
      return 'error'
    default:
      return 'default'
  }
}

const formatDuration = (ms: number): string => {
  const seconds = Math.floor(ms / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)

  if (hours > 0) {
    return `${hours}h ${minutes % 60}m ${seconds % 60}s`
  } else if (minutes > 0) {
    return `${minutes}m ${seconds % 60}s`
  } else {
    return `${seconds}s`
  }
}

const formatTime = (timestamp: number): string => {
  return new Date(timestamp).toLocaleTimeString()
}

// 생명주기
onMounted(() => {
  sessionStartTime.value = Date.now()

  durationTimer = setInterval(() => {
    sessionDuration.value = Date.now() - sessionStartTime.value
  }, 1000)

  addDebugLog('success', 'Terminal test page initialized')

  // 초기 연결 시뮬레이션
  setTimeout(() => {
    handleConnect()
  }, 1000)
})

onUnmounted(() => {
  if (durationTimer) {
    clearInterval(durationTimer)
  }
})
</script>

<style scoped>
.terminal-test {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background: #f5f5f5;
  padding: 16px;
  gap: 16px;
}

.test-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px;
  background: white;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.test-header h1 {
  margin: 0;
  color: #333;
}

.test-controls {
  display: flex;
  gap: 8px;
}

.test-layout {
  display: flex;
  gap: 16px;
  flex: 1;
  min-height: 0;
}

.terminal-section {
  flex: 1;
  background: white;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  overflow: hidden;
}

.info-panel {
  width: 300px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.info-grid {
  display: grid;
  gap: 8px;
}

.info-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.info-label {
  font-weight: 500;
  color: #666;
}

.info-value {
  font-family: monospace;
  background: #f5f5f5;
  padding: 2px 6px;
  border-radius: 3px;
  font-size: 12px;
}

.settings-grid {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.setting-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.setting-item label {
  font-size: 12px;
  color: #666;
}

.scenario-buttons {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.debug-log {
  height: 200px;
  background: white;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.debug-entries {
  max-height: 160px;
  overflow-y: auto;
  font-family: monospace;
  font-size: 11px;
}

.debug-entry {
  display: flex;
  gap: 8px;
  padding: 2px 4px;
  border-bottom: 1px solid #f0f0f0;
}

.debug-time {
  color: #999;
  flex-shrink: 0;
  width: 80px;
}

.debug-message {
  color: #333;
}

.debug-info .debug-message {
  color: #1890ff;
}

.debug-warn .debug-message {
  color: #fa8c16;
}

.debug-error .debug-message {
  color: #f5222d;
}

.debug-success .debug-message {
  color: #52c41a;
}

/* 반응형 */
@media (max-width: 1200px) {
  .test-layout {
    flex-direction: column;
  }

  .info-panel {
    width: 100%;
    flex-direction: row;
    overflow-x: auto;
  }

  .info-panel > * {
    min-width: 250px;
  }
}

@media (max-width: 768px) {
  .terminal-test {
    padding: 8px;
  }

  .test-header {
    flex-direction: column;
    gap: 12px;
    align-items: stretch;
  }

  .test-header h1 {
    text-align: center;
  }

  .test-controls {
    justify-content: center;
    flex-wrap: wrap;
  }

  .info-panel {
    flex-direction: column;
  }
}
</style>