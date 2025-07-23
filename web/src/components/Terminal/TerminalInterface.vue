<template>
  <div class="terminal-interface" :class="{ 'terminal-interface--fullscreen': isFullscreen }">
    <!-- 터미널 헤더 -->
    <div class="terminal-header">
      <div class="header-info">
        <div class="session-status" :class="[`status-${session?.status || 'disconnected'}`]">
          <div class="status-indicator"></div>
          <span class="status-text">{{ statusText }}</span>
        </div>

        <div v-if="session" class="session-details">
          <span class="session-id">{{ session.id.slice(0, 8) }}...</span>
          <span class="workspace-id">{{ session.workspaceId }}</span>
        </div>
      </div>

      <div class="header-controls">
        <!-- 기본 제어 버튼들 -->
        <NButtonGroup size="small">
          <NButton @click="handleClear" :disabled="!hasLogs" quaternary>
            <template #icon>
              <NIcon>
                <svg viewBox="0 0 24 24">
                  <path d="M19 13v6a2 2 0 01-2 2H5a2 2 0 01-2-2V7a2 2 0 012-2h6" stroke="currentColor" stroke-width="2" fill="none"/>
                  <path d="M15 3h4v4M19 3l-6 6" stroke="currentColor" stroke-width="2" fill="none"/>
                </svg>
              </NIcon>
            </template>
          </NButton>

          <NButton @click="handleStop" :disabled="!canStop" quaternary>
            <template #icon>
              <NIcon>
                <svg viewBox="0 0 24 24">
                  <rect x="6" y="6" width="12" height="12" fill="currentColor"/>
                </svg>
              </NIcon>
            </template>
          </NButton>

          <NButton @click="toggleSearch" :type="showSearch ? 'primary' : 'default'" quaternary>
            <template #icon>
              <NIcon>
                <svg viewBox="0 0 24 24">
                  <circle cx="11" cy="11" r="8" stroke="currentColor" stroke-width="2" fill="none"/>
                  <path d="m21 21-4.35-4.35" stroke="currentColor" stroke-width="2"/>
                </svg>
              </NIcon>
            </template>
          </NButton>

          <NButton @click="toggleFullscreen" quaternary>
            <template #icon>
              <NIcon>
                <svg viewBox="0 0 24 24">
                  <path v-if="!isFullscreen" d="M8 3H5a2 2 0 00-2 2v3m6 0V4m0 4h4m0-4v4m0 0h4a2 2 0 012 2v3" stroke="currentColor" stroke-width="2" fill="none"/>
                  <path v-else d="M8 3v3a2 2 0 002 2h3M16 3v3a2 2 0 01-2 2h-3" stroke="currentColor" stroke-width="2" fill="none"/>
                </svg>
              </NIcon>
            </template>
          </NButton>
        </NButtonGroup>

        <!-- 설정 드롭다운 -->
        <NDropdown :options="settingsOptions" @select="handleSettingsSelect" trigger="click">
          <NButton size="small" quaternary>
            <template #icon>
              <NIcon>
                <svg viewBox="0 0 24 24">
                  <circle cx="12" cy="12" r="3" stroke="currentColor" stroke-width="2" fill="none"/>
                  <path d="M12 1v6m0 6v6M9 4.5l3 3 3-3M9 19.5l3-3 3 3" stroke="currentColor" stroke-width="2" fill="none"/>
                </svg>
              </NIcon>
            </template>
          </NButton>
        </NDropdown>
      </div>
    </div>

    <!-- 검색 바 -->
    <div v-if="showSearch" class="search-bar">
      <NInput
        ref="searchInput"
        v-model:value="searchQuery"
        size="small"
        placeholder="터미널에서 검색... (Enter: 다음, Shift+Enter: 이전)"
        clearable
        @keyup.enter="handleSearchNext"
        @keyup.shift.enter="handleSearchPrevious"
        @keyup.escape="toggleSearch"
      >
        <template #prefix>
          <NIcon>
            <svg viewBox="0 0 24 24">
              <circle cx="11" cy="11" r="8" stroke="currentColor" stroke-width="2" fill="none"/>
              <path d="m21 21-4.35-4.35" stroke="currentColor" stroke-width="2"/>
            </svg>
          </NIcon>
        </template>
        <template #suffix>
          <div class="search-controls">
            <span class="search-results">{{ searchResultText }}</span>
            <NButton size="tiny" quaternary @click="handleSearchPrevious" :disabled="!hasSearchResults">
              <NIcon size="12">
                <svg viewBox="0 0 24 24">
                  <polyline points="15,18 9,12 15,6" stroke="currentColor" stroke-width="2" fill="none"/>
                </svg>
              </NIcon>
            </NButton>
            <NButton size="tiny" quaternary @click="handleSearchNext" :disabled="!hasSearchResults">
              <NIcon size="12">
                <svg viewBox="0 0 24 24">
                  <polyline points="9,18 15,12 9,6" stroke="currentColor" stroke-width="2" fill="none"/>
                </svg>
              </NIcon>
            </NButton>
          </div>
        </template>
      </NInput>
    </div>

    <!-- 터미널 내용 영역 -->
    <div class="terminal-content" ref="terminalContent">
      <VirtualScroller
        ref="virtualScroller"
        :items="optimizedLines"
        :item-height="20"
        :height="terminalHeight"
        :auto-scroll="autoScroll"
        :show-scroll-indicator="true"
        :enable-selection="true"
        @scroll="handleScroll"
        @selection-change="handleSelectionChange"
      >
        <template #default="{ item, index }">
          <div
            class="terminal-line"
            :class="[
              `line-type-${item.type}`,
              { 'line-highlighted': isHighlighted(index) }
            ]"
          >
            <span v-if="showTimestamp" class="line-timestamp">
              {{ formatTimestamp(item.timestamp) }}
            </span>
            <AnsiRenderer
              :text="item.content"
              :preserve-whitespace="true"
            />
          </div>
        </template>
      </VirtualScroller>
    </div>

    <!-- 명령 입력 영역 -->
    <div class="terminal-input" v-if="session?.status === 'connected'">
      <div class="input-line">
        <span class="input-prompt">$</span>
        <NInput
          ref="commandInput"
          v-model:value="currentCommand"
          type="text"
          placeholder="명령을 입력하세요..."
          @keyup.enter="handleCommandSubmit"
          @keyup.tab.prevent="handleTabCompletion"
          @keyup.up="handleHistoryUp"
          @keyup.down="handleHistoryDown"
          @keyup.ctrl.c="handleInterrupt"
          :disabled="isExecuting"
        />
        <NButton
          @click="handleCommandSubmit"
          :disabled="!currentCommand.trim() || isExecuting"
          :loading="isExecuting"
          type="primary"
          size="small"
        >
          실행
        </NButton>
      </div>
    </div>

    <!-- 성능 정보 (개발 모드) -->
    <div v-if="showPerformanceInfo" class="performance-overlay">
      <div class="perf-stats">
        <div class="perf-item">
          <span class="perf-label">Lines:</span>
          <span class="perf-value">{{ performanceMetrics.linesCount || 0 }}</span>
        </div>
        <div class="perf-item">
          <span class="perf-label">Memory:</span>
          <span class="perf-value">{{ formatBytes(performanceMetrics.estimatedMemoryUsage || 0) }}</span>
        </div>
        <div class="perf-item">
          <span class="perf-label">FPS:</span>
          <span class="perf-value">{{ Math.round(1000 / (performanceMetrics.avgRenderTime || 16)) }}</span>
        </div>
        <div class="perf-item">
          <span class="perf-label">Batches:</span>
          <span class="perf-value">{{ performanceMetrics.batchedUpdates || 0 }}</span>
        </div>
      </div>
    </div>

    <!-- 연결 상태 오버레이 -->
    <div v-if="session?.status !== 'connected'" class="connection-overlay">
      <div class="overlay-content">
        <NIcon size="48" :color="overlayIconColor">
          <svg viewBox="0 0 24 24">
            <circle v-if="session?.status === 'connecting'" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="2" fill="none"/>
            <path v-else-if="session?.status === 'error'" d="M12 2L2 7v10c0 5.55 3.84 10 9 11 1.16-.21 2.31-.54 3.42-1.01" stroke="currentColor" stroke-width="2" fill="none"/>
            <circle v-else cx="12" cy="12" r="10" stroke="currentColor" stroke-width="2" fill="none"/>
          </svg>
        </NIcon>
        <h3 class="overlay-title">{{ overlayTitle }}</h3>
        <p class="overlay-message">{{ overlayMessage }}</p>
        <div class="overlay-actions">
          <NButton v-if="session?.status === 'error'" @click="handleReconnect" type="primary">
            재연결
          </NButton>
          <NButton v-else-if="!session" @click="handleConnect" type="primary">
            연결
          </NButton>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { NButton, NButtonGroup, NDropdown, NIcon, NInput } from 'naive-ui'
import VirtualScroller from './VirtualScroller.vue'
import AnsiRenderer from './AnsiRenderer.vue'
import { useTerminalOptimization } from '@/composables/useTerminalOptimization'
import { useTerminalKeyboardShortcuts } from '@/composables/useKeyboardShortcuts'
import type { TerminalLog, TerminalSession } from '@/stores/terminal'

interface Props {
  session?: TerminalSession | null
  logs?: TerminalLog[]
  autoScroll?: boolean
  showTimestamp?: boolean
  fontSize?: number
  lineHeight?: number
  maxLines?: number
  showPerformanceInfo?: boolean
}

interface Emits {
  (e: 'command', command: string): void
  (e: 'clear'): void
  (e: 'stop'): void
  (e: 'connect'): void
  (e: 'reconnect'): void
  (e: 'disconnect'): void
  (e: 'settings-change', settings: Record<string, any>): void
}

const props = withDefaults(defineProps<Props>(), {
  logs: () => [],
  autoScroll: true,
  showTimestamp: true,
  fontSize: 14,
  lineHeight: 1.4,
  maxLines: 1000,
  showPerformanceInfo: false,
})

const emit = defineEmits<Emits>()

// 상태
const isFullscreen = ref(false)
const showSearch = ref(false)
const searchQuery = ref('')
const searchInput = ref()
const currentCommand = ref('')
const commandInput = ref()
const commandHistory = ref<string[]>([])
const historyIndex = ref(-1)
const isExecuting = ref(false)
const terminalContent = ref()
const virtualScroller = ref()
const terminalHeight = ref(400)
const searchResults = ref<number[]>([])
const currentSearchIndex = ref(-1)

// 터미널 최적화 컴포저블
const {
  lines: optimizedLines,
  addLine,
  addLines,
  clearLines,
  searchLines,
  handleScroll: optimizationScroll,
  metrics: performanceMetrics,
} = useTerminalOptimization({
  maxLines: props.maxLines,
  enableVirtualization: true,
  updateInterval: 50,
  batchSize: 10,
})

// 계산된 속성
const hasLogs = computed(() => props.logs && props.logs.length > 0)
const canStop = computed(() => isExecuting.value)

const statusText = computed(() => {
  switch (props.session?.status) {
    case 'connected':
      return '연결됨'
    case 'connecting':
      return '연결 중...'
    case 'error':
      return '오류'
    default:
      return '연결 안됨'
  }
})

const hasSearchResults = computed(() => searchResults.value.length > 0)
const searchResultText = computed(() => {
  if (!hasSearchResults.value) return '결과 없음'
  return `${currentSearchIndex.value + 1}/${searchResults.value.length}`
})

const overlayIconColor = computed(() => {
  switch (props.session?.status) {
    case 'connecting':
      return '#FFA500'
    case 'error':
      return '#F44336'
    default:
      return '#757575'
  }
})

const overlayTitle = computed(() => {
  switch (props.session?.status) {
    case 'connecting':
      return '연결 중...'
    case 'error':
      return '연결 오류'
    default:
      return '연결되지 않음'
  }
})

const overlayMessage = computed(() => {
  switch (props.session?.status) {
    case 'connecting':
      return '터미널 세션에 연결하고 있습니다.'
    case 'error':
      return '터미널 세션 연결에 실패했습니다.'
    default:
      return '터미널 세션을 시작하려면 연결 버튼을 클릭하세요.'
  }
})

const settingsOptions = computed(() => [
  {
    label: '표시 설정',
    key: 'display',
    children: [
      { label: '타임스탬프 토글', key: 'toggle-timestamp' },
      { label: '자동 스크롤 토글', key: 'toggle-autoscroll' },
      { label: '성능 정보 토글', key: 'toggle-performance' },
    ],
  },
  { type: 'divider' },
  { label: '내보내기', key: 'export' },
  { label: '설정', key: 'settings' },
])

// 메서드
const handleClear = () => {
  clearLines()
  emit('clear')
}

const handleStop = () => {
  isExecuting.value = false
  emit('stop')
}

const handleConnect = () => {
  emit('connect')
}

const handleReconnect = () => {
  emit('reconnect')
}

const handleDisconnect = () => {
  emit('disconnect')
}

const toggleFullscreen = () => {
  if (!isFullscreen.value) {
    document.documentElement.requestFullscreen?.()
  } else {
    document.exitFullscreen?.()
  }
}

const toggleSearch = () => {
  showSearch.value = !showSearch.value
  if (showSearch.value) {
    nextTick(() => {
      searchInput.value?.focus()
    })
  } else {
    searchQuery.value = ''
    searchResults.value = []
    currentSearchIndex.value = -1
  }
}

const handleSearchNext = () => {
  if (!searchQuery.value.trim()) return
  performSearch('next')
}

const handleSearchPrevious = () => {
  if (!searchQuery.value.trim()) return
  performSearch('previous')
}

const performSearch = (direction: 'next' | 'previous') => {
  const results = searchLines(searchQuery.value)
  searchResults.value = results.map((_, index) => index)

  if (hasSearchResults.value) {
    if (direction === 'next') {
      currentSearchIndex.value = (currentSearchIndex.value + 1) % searchResults.value.length
    } else {
      currentSearchIndex.value = currentSearchIndex.value === 0
        ? searchResults.value.length - 1
        : currentSearchIndex.value - 1
    }

    // 검색 결과로 스크롤
    const targetIndex = searchResults.value[currentSearchIndex.value]
    virtualScroller.value?.scrollToIndex(targetIndex)
  }
}

const handleCommandSubmit = () => {
  const command = currentCommand.value.trim()
  if (!command) return

  // 명령 히스토리에 추가
  commandHistory.value.push(command)
  historyIndex.value = -1

  // 실행 상태 설정
  isExecuting.value = true

  // 명령 emit
  emit('command', command)

  // 입력 필드 클리어
  currentCommand.value = ''
}

const handleTabCompletion = () => {
  // TODO: 자동완성 구현
  console.log('Tab completion requested')
}

const handleHistoryUp = () => {
  if (commandHistory.value.length === 0) return

  if (historyIndex.value === -1) {
    historyIndex.value = commandHistory.value.length - 1
  } else if (historyIndex.value > 0) {
    historyIndex.value--
  }

  currentCommand.value = commandHistory.value[historyIndex.value] || ''
}

const handleHistoryDown = () => {
  if (historyIndex.value === -1) return

  if (historyIndex.value < commandHistory.value.length - 1) {
    historyIndex.value++
    currentCommand.value = commandHistory.value[historyIndex.value] || ''
  } else {
    historyIndex.value = -1
    currentCommand.value = ''
  }
}

const handleInterrupt = () => {
  if (isExecuting.value) {
    handleStop()
  }
}

const handleScroll = (scrollInfo: { top: number; isAtBottom: boolean }) => {
  optimizationScroll(scrollInfo)
}

const handleSelectionChange = (selection: { start: number; end: number } | null) => {
  // TODO: 선택 처리
  console.log('Selection changed:', selection)
}

const handleSettingsSelect = (key: string) => {
  switch (key) {
    case 'toggle-timestamp':
      emit('settings-change', { showTimestamp: !props.showTimestamp })
      break
    case 'toggle-autoscroll':
      emit('settings-change', { autoScroll: !props.autoScroll })
      break
    case 'toggle-performance':
      emit('settings-change', { showPerformanceInfo: !props.showPerformanceInfo })
      break
    default:
      console.log('Settings action:', key)
  }
}

const isHighlighted = (index: number): boolean => {
  return searchResults.value.includes(index)
}

const formatTimestamp = (timestamp: string): string => {
  return new Date(timestamp).toLocaleTimeString()
}

const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`
}

const updateTerminalHeight = () => {
  if (terminalContent.value) {
    const rect = terminalContent.value.getBoundingClientRect()
    terminalHeight.value = rect.height
  }
}

// 전체화면 상태 감지
const handleFullscreenChange = () => {
  isFullscreen.value = !!document.fullscreenElement
}

// 로그 변경 감지하여 최적화된 라인에 추가
watch(() => props.logs, (newLogs, oldLogs) => {
  if (!newLogs) return

  const oldLength = oldLogs?.length || 0
  const newLength = newLogs.length

  if (newLength > oldLength) {
    const newItems = newLogs.slice(oldLength).map(log => ({
      content: log.content,
      type: log.type,
      timestamp: log.timestamp,
    }))

    if (newItems.length === 1) {
      addLine(newItems[0])
    } else {
      addLines(newItems)
    }
  }
}, { deep: true, immediate: true })

// 명령 실행 완료 처리
watch(() => props.session?.status, (newStatus) => {
  if (newStatus === 'connected' && isExecuting.value) {
    isExecuting.value = false
  }
})

// 키보드 단축키 설정
useTerminalKeyboardShortcuts({
  onClear: handleClear,
  onStop: handleStop,
  onSearch: toggleSearch,
  onFullscreen: toggleFullscreen,
  onScrollToTop: () => virtualScroller.value?.scrollToIndex(0),
  onScrollToBottom: () => virtualScroller.value?.scrollToBottom(),
  onCopy: () => {
    // TODO: 선택된 텍스트 복사 구현
    console.log('Copy shortcut triggered')
  },
  onPaste: () => {
    // TODO: 텍스트 붙여넣기 구현
    console.log('Paste shortcut triggered')
  },
  onSelectAll: () => {
    // TODO: 전체 선택 구현
    console.log('Select all shortcut triggered')
  },
  onZoomIn: () => {
    const newSize = Math.min(props.fontSize + 2, 24)
    emit('settings-change', { fontSize: newSize })
  },
  onZoomOut: () => {
    const newSize = Math.max(props.fontSize - 2, 10)
    emit('settings-change', { fontSize: newSize })
  },
  onResetZoom: () => {
    emit('settings-change', { fontSize: 14 })
  },
})

// 생명주기
onMounted(() => {
  updateTerminalHeight()
  document.addEventListener('fullscreenchange', handleFullscreenChange)
  window.addEventListener('resize', updateTerminalHeight)

  // 명령 입력에 포커스
  nextTick(() => {
    commandInput.value?.focus()
  })
})

onUnmounted(() => {
  document.removeEventListener('fullscreenchange', handleFullscreenChange)
  window.removeEventListener('resize', updateTerminalHeight)
})
</script>

<style scoped>
.terminal-interface {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #1a1a1a;
  color: #e5e5e5;
  font-family: 'Fira Code', 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  position: relative;
  overflow: hidden;
}

.terminal-interface--fullscreen {
  position: fixed;
  top: 0;
  left: 0;
  width: 100vw;
  height: 100vh;
  z-index: 9999;
}

.terminal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  background: rgba(0, 0, 0, 0.2);
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  min-height: 40px;
}

.header-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.session-status {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.05);
  font-size: 12px;
}

.status-indicator {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #757575;
}

.status-connected .status-indicator {
  background: #4CAF50;
}

.status-connecting .status-indicator {
  background: #FF9800;
  animation: pulse 1.5s infinite;
}

.status-error .status-indicator {
  background: #F44336;
}

.session-details {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 11px;
  color: rgba(255, 255, 255, 0.6);
}

.session-id, .workspace-id {
  padding: 2px 6px;
  background: rgba(255, 255, 255, 0.1);
  border-radius: 3px;
  font-family: monospace;
}

.header-controls {
  display: flex;
  align-items: center;
  gap: 8px;
}

.search-bar {
  padding: 8px 12px;
  background: rgba(0, 0, 0, 0.1);
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}

.search-controls {
  display: flex;
  align-items: center;
  gap: 4px;
}

.search-results {
  font-size: 11px;
  color: rgba(255, 255, 255, 0.6);
  margin-right: 8px;
  min-width: 60px;
  text-align: center;
}

.terminal-content {
  flex: 1;
  position: relative;
  overflow: hidden;
}

.terminal-line {
  display: flex;
  align-items: flex-start;
  padding: 0 8px;
  line-height: 1.4;
  font-size: 14px;
  white-space: pre-wrap;
  word-break: break-all;
}

.line-type-input {
  color: #4A9EFF;
}

.line-type-error {
  color: #F44336;
}

.line-type-system {
  color: #FFA500;
  font-style: italic;
}

.line-highlighted {
  background: rgba(255, 255, 0, 0.2);
}

.line-timestamp {
  color: rgba(255, 255, 255, 0.4);
  font-size: 11px;
  margin-right: 8px;
  flex-shrink: 0;
  width: 80px;
}

.terminal-input {
  padding: 8px 12px;
  background: rgba(0, 0, 0, 0.2);
  border-top: 1px solid rgba(255, 255, 255, 0.1);
}

.input-line {
  display: flex;
  align-items: center;
  gap: 8px;
}

.input-prompt {
  color: #4A9EFF;
  font-weight: bold;
  flex-shrink: 0;
}

.performance-overlay {
  position: absolute;
  top: 50px;
  right: 12px;
  background: rgba(0, 0, 0, 0.8);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 6px;
  padding: 8px;
  font-size: 11px;
  z-index: 10;
}

.perf-stats {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.perf-item {
  display: flex;
  justify-content: space-between;
  gap: 12px;
}

.perf-label {
  color: rgba(255, 255, 255, 0.6);
}

.perf-value {
  color: #4CAF50;
  font-family: monospace;
  font-weight: 500;
}

.connection-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.8);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 20;
}

.overlay-content {
  text-align: center;
  padding: 32px;
  max-width: 400px;
}

.overlay-title {
  font-size: 18px;
  margin: 16px 0 8px;
  color: #e5e5e5;
}

.overlay-message {
  color: rgba(255, 255, 255, 0.7);
  margin-bottom: 24px;
  line-height: 1.5;
}

.overlay-actions {
  display: flex;
  justify-content: center;
  gap: 12px;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

/* 반응형 */
@media (max-width: 768px) {
  .terminal-header {
    flex-direction: column;
    gap: 8px;
    align-items: stretch;
  }

  .header-info,
  .header-controls {
    justify-content: center;
  }

  .session-details {
    display: none;
  }

  .terminal-line {
    padding: 0 4px;
    font-size: 12px;
  }

  .line-timestamp {
    display: none;
  }

  .performance-overlay {
    position: relative;
    top: auto;
    right: auto;
    margin: 8px;
  }
}
</style>