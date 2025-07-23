<template>
  <div class="terminal-emulator" :class="{ 'terminal-emulator--fullscreen': isFullscreen }">
    <!-- 터미널 헤더 -->
    <div class="terminal-header">
      <div class="terminal-title">
        <span class="session-name">{{ sessionName }}</span>
        <span class="session-status" :class="`status-${connectionStatus.type}`">
          {{ connectionStatus.icon }} {{ connectionStatus.text }}
        </span>
      </div>

      <div class="terminal-controls">
        <NButton
          text
          size="small"
          @click="searchVisible = !searchVisible"
          :type="searchVisible ? 'primary' : 'default'"
        >
          <template #icon>
            <Icon name="search" />
          </template>
        </NButton>

        <NButton
          text
          size="small"
          @click="clearTerminal"
          :disabled="!hasLogs"
        >
          <template #icon>
            <Icon name="clear" />
          </template>
          Clear
        </NButton>

        <NButton
          text
          size="small"
          @click="exportLogs"
          :disabled="!hasLogs"
        >
          <template #icon>
            <Icon name="download" />
          </template>
          Export
        </NButton>

        <NButton
          text
          size="small"
          @click="toggleAutoScroll"
          :type="autoScrollEnabled ? 'primary' : 'default'"
        >
          <template #icon>
            <Icon :name="autoScrollEnabled ? 'auto-scroll-on' : 'auto-scroll-off'" />
          </template>
        </NButton>

        <NButton
          text
          size="small"
          @click="toggleFullscreen"
        >
          <template #icon>
            <Icon :name="isFullscreen ? 'fullscreen-exit' : 'fullscreen'" />
          </template>
        </NButton>
      </div>
    </div>

    <!-- 검색 바 -->
    <div v-if="searchVisible" class="terminal-search">
      <NInput
        v-model:value="searchQuery"
        placeholder="로그 검색..."
        clearable
        @update:value="handleSearch"
      >
        <template #prefix>
          <Icon name="search" />
        </template>
        <template #suffix>
          <span v-if="searchResults.length > 0" class="search-count">
            {{ currentSearchIndex + 1 }} / {{ searchResults.length }}
          </span>
        </template>
      </NInput>

      <div class="search-navigation">
        <NButton
          text
          size="small"
          @click="previousSearchResult"
          :disabled="searchResults.length === 0"
        >
          <Icon name="arrow-up" />
        </NButton>
        <NButton
          text
          size="small"
          @click="nextSearchResult"
          :disabled="searchResults.length === 0"
        >
          <Icon name="arrow-down" />
        </NButton>
      </div>
    </div>

    <!-- 터미널 출력 영역 -->
    <div class="terminal-content" ref="terminalContentRef">
      <!-- 가상 스크롤링 사용 -->
      <VirtualScroller
        v-if="useVirtualScrolling"
        ref="virtualScrollerRef"
        :items="optimizedLogs"
        :item-height="lineHeight"
        :container-height="contentHeight"
        :auto-scroll="autoScrollEnabled"
        :scroll-key="sessionId"
        @scroll="handleScroll"
      >
        <template #default="{ item, index }">
          <div
            class="terminal-line"
            :class="getLogLineClasses(item)"
            :data-log-id="item.id"
            :data-search-match="isSearchMatch(index)"
          >
            <span class="log-timestamp">{{ formatTime(item.timestamp) }}</span>
            <div class="log-content">
              <AnsiRenderer
                v-if="item.parsed?.hasAnsi"
                :content="highlightSearchTerm(item.content)"
                :allow-unsafe-html="false"
              />
              <span v-else v-html="highlightSearchTerm(item.content)" />
            </div>
          </div>
        </template>
      </VirtualScroller>

      <!-- 일반 스크롤링 (fallback) -->
      <div v-else class="terminal-output" ref="terminalOutputRef">
        <div
          v-for="(log, index) in optimizedLogs"
          :key="log.id"
          class="terminal-line"
          :class="getLogLineClasses(log)"
          :data-log-id="log.id"
          :data-search-match="isSearchMatch(index)"
        >
          <span class="log-timestamp">{{ formatTime(log.timestamp) }}</span>
          <div class="log-content">
            <AnsiRenderer
              v-if="log.parsed?.hasAnsi"
              :content="highlightSearchTerm(log.content)"
              :allow-unsafe-html="false"
            />
            <span v-else v-html="highlightSearchTerm(log.content)" />
          </div>
        </div>
      </div>

      <!-- 자동 스크롤 버튼 -->
      <Transition name="fade">
        <NButton
          v-show="!autoScrollEnabled && hasNewContent"
          class="scroll-to-bottom-btn"
          type="primary"
          size="small"
          circle
          @click="scrollToBottom"
        >
          <template #icon>
            <Icon name="arrow-down" />
          </template>
        </NButton>
      </Transition>
    </div>

    <!-- 입력 영역 -->
    <div class="terminal-input">
      <div class="input-prompt">
        <span class="prompt-symbol">$</span>
      </div>

      <NInput
        ref="commandInputRef"
        v-model:value="currentCommand"
        type="text"
        placeholder="명령어를 입력하세요... (Enter로 실행, Ctrl+C로 중단)"
        :disabled="!isConnected"
        @keydown="handleKeyDown"
        @focus="inputFocused = true"
        @blur="inputFocused = false"
        class="command-input"
      />

      <div class="input-actions">
        <NButton
          text
          size="small"
          @click="executeCurrentCommand"
          :disabled="!currentCommand.trim() || !isConnected"
        >
          <template #icon>
            <Icon name="play" />
          </template>
        </NButton>

        <NButton
          text
          size="small"
          @click="stopExecution"
          :disabled="!isExecuting"
        >
          <template #icon>
            <Icon name="stop" />
          </template>
        </NButton>
      </div>
    </div>

    <!-- 상태 바 -->
    <div class="terminal-status">
      <div class="status-left">
        <span class="connection-info">
          연결 상태: {{ connectionStatus.text }}
        </span>
        <span v-if="lastActivity" class="last-activity">
          마지막 활동: {{ formatRelativeTime(lastActivity) }}
        </span>
      </div>

      <div class="status-right">
        <span v-if="performanceStats" class="performance-info">
          {{ performanceStats.totalLogs }}줄 | {{ formatFileSize(performanceStats.memoryUsage) }}
        </span>
        <span v-if="searchResults.length > 0" class="search-info">
          검색 결과: {{ searchResults.length }}개
        </span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { NButton, NInput } from 'naive-ui'
import type { TerminalLog } from '@/stores/terminal'
import VirtualScroller from './VirtualScroller.vue'
import AnsiRenderer from './AnsiRenderer.vue'
import { useTerminalOptimization } from '@/composables/useTerminalOptimization'
import {
  CommandHistory,
  formatFileSize,
  formatTerminalTimestamp,
  getTerminalStatusInfo,
  highlightSearchTerm as highlightSearchTermUtil,
} from '@/utils/terminal-utils'

// Props 인터페이스
interface Props {
  sessionId: string
  sessionName?: string
  logs: TerminalLog[]
  isConnected?: boolean
  isExecuting?: boolean
  lastActivity?: string
  useVirtualScrolling?: boolean
  lineHeight?: number
  maxLines?: number
  autoScroll?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  sessionName: 'Terminal',
  isConnected: false,
  isExecuting: false,
  useVirtualScrolling: true,
  lineHeight: 24,
  maxLines: 1000,
  autoScroll: true,
})

// Emits
const emit = defineEmits<{
  executeCommand: [command: string]
  stopExecution: []
  clearLogs: []
  exportLogs: [format: 'text' | 'html' | 'json']
}>()

// 반응형 참조
const terminalContentRef = ref<HTMLElement>()
const terminalOutputRef = ref<HTMLElement>()
const virtualScrollerRef = ref<InstanceType<typeof VirtualScroller>>()
const commandInputRef = ref<InstanceType<typeof NInput>>()

// 상태 관리
const currentCommand = ref('')
const inputFocused = ref(false)
const isFullscreen = ref(false)
const autoScrollEnabled = ref(props.autoScroll)
const hasNewContent = ref(false)
const contentHeight = ref(400)

// 검색 관련
const searchVisible = ref(false)
const searchQuery = ref('')
const searchResults = ref<number[]>([])
const currentSearchIndex = ref(-1)

// 명령어 히스토리
const commandHistory = new CommandHistory(100)

// 성능 최적화
const optimization = useTerminalOptimization({
  maxLines: props.maxLines,
  batchSize: 50,
  debounceDelay: 100,
  useVirtualScrolling: props.useVirtualScrolling,
  enableMemoryMonitoring: true,
  collectStats: true,
})

// 계산된 속성
const connectionStatus = computed(() => {
  const status = props.isConnected ? 'connected' : 'disconnected'
  return getTerminalStatusInfo(status)
})

const hasLogs = computed(() => props.logs.length > 0)

const optimizedLogs = computed(() => {
  // 성능 최적화 적용
  optimization.addLogs(props.logs)
  return optimization.logs.value
})

const performanceStats = computed(() => optimization.profilePerformance())

const isSearchMatch = (index: number) => searchResults.value.includes(index)

// 메서드
const formatTime = (timestamp: string) => formatTerminalTimestamp(timestamp)

const formatRelativeTime = (timestamp: string) => {
  const diff = Date.now() - new Date(timestamp).getTime()
  const minutes = Math.floor(diff / 60000)
  if (minutes < 1) return '방금 전'
  if (minutes < 60) return `${minutes}분 전`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}시간 전`
  const days = Math.floor(hours / 24)
  return `${days}일 전`
}

const getLogLineClasses = (log: TerminalLog) => {
  return [
    `log-${log.type}`,
    `level-${log.level || 'info'}`,
    {
      'search-highlight': searchQuery.value && log.content.toLowerCase().includes(searchQuery.value.toLowerCase()),
    },
  ]
}

const highlightSearchTerm = (content: string) => {
  if (!searchQuery.value) return content
  return highlightSearchTermUtil(content, searchQuery.value)
}

// 터미널 제어
const clearTerminal = () => {
  emit('clearLogs')
  optimization.clearLogs()
  searchResults.value = []
  currentSearchIndex.value = -1
}

const exportLogs = () => {
  emit('exportLogs', 'text')
}

const toggleAutoScroll = () => {
  autoScrollEnabled.value = !autoScrollEnabled.value
  if (autoScrollEnabled.value) {
    scrollToBottom()
  }
}

const toggleFullscreen = () => {
  isFullscreen.value = !isFullscreen.value

  if (isFullscreen.value) {
    document.documentElement.requestFullscreen?.()
  } else {
    document.exitFullscreen?.()
  }
}

const scrollToBottom = () => {
  if (virtualScrollerRef.value) {
    virtualScrollerRef.value.scrollToBottom()
  } else if (terminalOutputRef.value) {
    terminalOutputRef.value.scrollTop = terminalOutputRef.value.scrollHeight
  }
  hasNewContent.value = false
}

// 명령어 실행
const executeCurrentCommand = () => {
  const command = currentCommand.value.trim()
  if (!command) return

  commandHistory.add(command)
  emit('executeCommand', command)
  currentCommand.value = ''
}

const stopExecution = () => {
  emit('stopExecution')
}

// 키보드 이벤트
const handleKeyDown = (event: KeyboardEvent) => {
  switch (event.key) {
    case 'Enter':
      if (!event.shiftKey) {
        event.preventDefault()
        executeCurrentCommand()
      }
      break

    case 'ArrowUp':
      if (!event.shiftKey) {
        event.preventDefault()
        const prev = commandHistory.getPrevious()
        if (prev !== null) {
          currentCommand.value = prev
        }
      }
      break

    case 'ArrowDown':
      if (!event.shiftKey) {
        event.preventDefault()
        const next = commandHistory.getNext()
        if (next !== null) {
          currentCommand.value = next
        }
      }
      break

    case 'c':
      if (event.ctrlKey) {
        event.preventDefault()
        stopExecution()
      }
      break

    case 'l':
      if (event.ctrlKey) {
        event.preventDefault()
        clearTerminal()
      }
      break

    case 'f':
      if (event.ctrlKey) {
        event.preventDefault()
        searchVisible.value = !searchVisible.value
      }
      break
  }
}

// 검색 기능
const handleSearch = (query: string) => {
  if (!query.trim()) {
    searchResults.value = []
    currentSearchIndex.value = -1
    return
  }

  searchResults.value = optimization.searchLogs(query)
  currentSearchIndex.value = searchResults.value.length > 0 ? 0 : -1

  if (currentSearchIndex.value >= 0) {
    scrollToSearchResult(currentSearchIndex.value)
  }
}

const nextSearchResult = () => {
  if (searchResults.value.length === 0) return

  currentSearchIndex.value = (currentSearchIndex.value + 1) % searchResults.value.length
  scrollToSearchResult(currentSearchIndex.value)
}

const previousSearchResult = () => {
  if (searchResults.value.length === 0) return

  currentSearchIndex.value = currentSearchIndex.value <= 0
    ? searchResults.value.length - 1
    : currentSearchIndex.value - 1
  scrollToSearchResult(currentSearchIndex.value)
}

const scrollToSearchResult = (index: number) => {
  const logIndex = searchResults.value[index]
  if (virtualScrollerRef.value) {
    virtualScrollerRef.value.scrollToIndex(logIndex)
  }
}

// 스크롤 이벤트
const handleScroll = () => {
  // 사용자가 스크롤했을 때 자동 스크롤 비활성화
  if (virtualScrollerRef.value) {
    // const scrollTop = virtualScrollerRef.value.getScrollTop()
    const { end } = virtualScrollerRef.value.getVisibleRange()

    // 맨 아래에 있지 않으면 자동 스크롤 비활성화
    if (end < optimizedLogs.value.length - 1) {
      autoScrollEnabled.value = false
      hasNewContent.value = true
    }
  }
}

// 컨테이너 크기 업데이트
const updateContentHeight = () => {
  if (terminalContentRef.value) {
    const rect = terminalContentRef.value.getBoundingClientRect()
    contentHeight.value = rect.height
  }
}

// 로그 변경 감지
watch(
  () => props.logs.length,
  (newLength, oldLength) => {
    if (newLength > (oldLength || 0)) {
      hasNewContent.value = !autoScrollEnabled.value

      if (autoScrollEnabled.value) {
        nextTick(() => scrollToBottom())
      }
    }
  },
)

// 자동 성능 조정
watch(
  () => performanceStats.value,
  (stats) => {
    if (stats && !stats.isOptimal) {
      optimization.autoTunePerformance()
    }
  },
  { deep: true },
)

// 생명주기
onMounted(() => {
  updateContentHeight()

  // 리사이즈 이벤트 리스너
  window.addEventListener('resize', updateContentHeight)

  // 입력창에 포커스
  nextTick(() => {
    commandInputRef.value?.focus()
  })

  // 초기에 맨 아래로 스크롤
  if (autoScrollEnabled.value && props.logs.length > 0) {
    nextTick(() => scrollToBottom())
  }
})

onUnmounted(() => {
  window.removeEventListener('resize', updateContentHeight)
  optimization.stopOptimization()
})

// 부모 컴포넌트에서 접근 가능한 메서드들
defineExpose({
  scrollToBottom,
  clearTerminal,
  focusInput: () => commandInputRef.value?.focus(),
  executeCommand: executeCurrentCommand,
  getPerformanceStats: () => performanceStats.value,
})
</script>

<style lang="scss" scoped>
.terminal-emulator {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #1a1a1a;
  color: #ffffff;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;

  &--fullscreen {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    z-index: 9999;
    background: #1a1a1a;
  }
}

.terminal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 16px;
  background: #2a2a2a;
  border-bottom: 1px solid #3a3a3a;
  min-height: 40px;

  .terminal-title {
    display: flex;
    align-items: center;
    gap: 12px;

    .session-name {
      font-weight: 600;
      font-size: 14px;
    }

    .session-status {
      font-size: 12px;
      padding: 2px 8px;
      border-radius: 12px;

      &.status-success {
        background: rgba(0, 255, 0, 0.1);
        color: #00ff00;
      }

      &.status-error {
        background: rgba(255, 0, 0, 0.1);
        color: #ff6b6b;
      }

      &.status-warning {
        background: rgba(255, 255, 0, 0.1);
        color: #ffa502;
      }

      &.status-default {
        background: rgba(128, 128, 128, 0.1);
        color: #888;
      }
    }
  }

  .terminal-controls {
    display: flex;
    gap: 4px;
  }
}

.terminal-search {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  background: #2a2a2a;
  border-bottom: 1px solid #3a3a3a;

  .search-count {
    font-size: 12px;
    color: #888;
  }

  .search-navigation {
    display: flex;
    gap: 2px;
  }
}

.terminal-content {
  flex: 1;
  position: relative;
  overflow: hidden;
}

.terminal-output {
  height: 100%;
  overflow-y: auto;
  padding: 8px;

  &::-webkit-scrollbar {
    width: 8px;
  }

  &::-webkit-scrollbar-track {
    background: rgba(255, 255, 255, 0.1);
  }

  &::-webkit-scrollbar-thumb {
    background: rgba(255, 255, 255, 0.3);
    border-radius: 4px;

    &:hover {
      background: rgba(255, 255, 255, 0.5);
    }
  }
}

.terminal-line {
  display: flex;
  align-items: flex-start;
  padding: 2px 0;
  min-height: 20px;

  &:hover {
    background: rgba(255, 255, 255, 0.05);
  }

  &[data-search-match="true"] {
    background: rgba(255, 255, 0, 0.1);
  }

  &.search-highlight {
    animation: highlight-flash 0.5s ease-in-out;
  }

  .log-timestamp {
    color: rgba(255, 255, 255, 0.5);
    font-size: 11px;
    margin-right: 8px;
    min-width: 80px;
    user-select: none;
    flex-shrink: 0;
  }

  .log-content {
    flex: 1;
    word-wrap: break-word;
    white-space: pre-wrap;
  }

  // 로그 타입별 스타일
  &.log-input .log-content {
    color: #ffffff;
    &::before {
      content: '$ ';
      color: #00ff00;
      font-weight: bold;
    }
  }

  &.log-output .log-content {
    color: #e0e0e0;
  }

  &.log-error .log-content {
    color: #ff6b6b;
  }

  &.log-system .log-content {
    color: #74c0fc;
    font-style: italic;
  }
}

.scroll-to-bottom-btn {
  position: absolute;
  bottom: 16px;
  right: 16px;
  z-index: 10;
}

.terminal-input {
  display: flex;
  align-items: center;
  padding: 8px 16px;
  background: #2a2a2a;
  border-top: 1px solid #3a3a3a;
  gap: 8px;

  .input-prompt {
    color: #00ff00;
    font-weight: bold;
    font-size: 14px;
    flex-shrink: 0;
  }

  .command-input {
    flex: 1;

    :deep(.n-input__input-el) {
      background: transparent;
      border: none;
      color: #ffffff;
      font-family: inherit;
      font-size: 14px;

      &::placeholder {
        color: rgba(255, 255, 255, 0.5);
      }
    }

    :deep(.n-input__border),
    :deep(.n-input__state-border) {
      display: none;
    }
  }

  .input-actions {
    display: flex;
    gap: 4px;
  }
}

.terminal-status {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 16px;
  background: #1a1a1a;
  border-top: 1px solid #3a3a3a;
  font-size: 11px;
  color: rgba(255, 255, 255, 0.7);

  .status-left,
  .status-right {
    display: flex;
    gap: 16px;
  }
}

// 애니메이션
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

@keyframes highlight-flash {
  0%, 100% { background: rgba(255, 255, 0, 0.1); }
  50% { background: rgba(255, 255, 0, 0.3); }
}

// 반응형
@media (max-width: 768px) {
  .terminal-header {
    padding: 6px 12px;

    .terminal-controls {
      gap: 2px;
    }
  }

  .terminal-input {
    padding: 6px 12px;
  }

  .terminal-line {
    .log-timestamp {
      min-width: 60px;
      font-size: 10px;
    }
  }
}
</style>