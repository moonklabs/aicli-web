<template>
  <div 
    class="mobile-terminal"
    :class="{
      'mobile-terminal--fullscreen': isFullscreen,
      'mobile-terminal--landscape': isLandscape,
      'mobile-terminal--keyboard-visible': isKeyboardVisible
    }"
  >
    <!-- 헤더 -->
    <div class="mobile-terminal__header">
      <TouchButton
        text
        size="lg"
        class="mobile-terminal__back"
        @click="emit('close')"
      >
        <Icon name="arrow-left" />
      </TouchButton>

      <div class="mobile-terminal__title">
        <span class="session-name">{{ sessionName }}</span>
        <span class="connection-status" :class="connectionClass">
          <Icon :name="connectionIcon" size="14" />
        </span>
      </div>

      <div class="mobile-terminal__actions">
        <TouchButton
          text
          size="lg"
          @click="toggleFullscreen"
        >
          <Icon :name="isFullscreen ? 'fullscreen-exit' : 'fullscreen'" />
        </TouchButton>

        <TouchButton
          text
          size="lg"
          @click="showQuickActions = true"
        >
          <Icon name="more-vert" />
        </TouchButton>
      </div>
    </div>

    <!-- 터미널 출력 영역 -->
    <div 
      ref="terminalContentRef"
      class="mobile-terminal__content"
      @touchstart="handleTouchStart"
      @touchmove="handleTouchMove"
      @touchend="handleTouchEnd"
    >
      <div 
        ref="scrollContainerRef"
        class="scroll-container"
        :style="scrollStyle"
      >
        <div
          v-for="(log, index) in visibleLogs"
          :key="log.id"
          class="terminal-line"
          :class="getLogClass(log)"
        >
          <span class="line-number">{{ index + 1 }}</span>
          <div class="line-content">
            <span v-if="log.type === 'input'" class="prompt">$ </span>
            <span v-html="highlightAnsi(log.content)" />
          </div>
        </div>

        <!-- 자동 스크롤 표시기 -->
        <div 
          v-if="!isAtBottom && hasNewContent"
          class="new-content-indicator"
          @click="scrollToBottom"
        >
          <Icon name="arrow-down" />
          <span>새 메시지</span>
        </div>
      </div>
    </div>

    <!-- 입력 영역 -->
    <div 
      class="mobile-terminal__input"
      :class="{ 'mobile-terminal__input--focused': isInputFocused }"
    >
      <div class="input-wrapper">
        <span class="prompt">$ </span>
        <input
          ref="commandInputRef"
          v-model="currentCommand"
          type="text"
          class="command-input"
          :placeholder="isConnected ? '명령어 입력...' : '연결되지 않음'"
          :disabled="!isConnected"
          @focus="handleInputFocus"
          @blur="handleInputBlur"
          @keydown="handleKeyDown"
        />
      </div>

      <div class="input-actions">
        <TouchButton
          v-haptic
          type="primary"
          size="lg"
          circle
          :disabled="!currentCommand.trim() || !isConnected"
          @click="executeCommand"
        >
          <Icon name="send" />
        </TouchButton>
      </div>
    </div>

    <!-- 빠른 명령어 바 -->
    <Transition name="slide-up">
      <div v-if="showQuickCommands" class="quick-commands">
        <div class="quick-commands__header">
          <span>자주 사용하는 명령어</span>
          <TouchButton text size="sm" @click="showQuickCommands = false">
            <Icon name="close" />
          </TouchButton>
        </div>
        <div class="quick-commands__list">
          <TouchButton
            v-for="cmd in quickCommands"
            :key="cmd.id"
            text
            size="md"
            class="quick-command"
            @click="selectQuickCommand(cmd.command)"
          >
            {{ cmd.label }}
          </TouchButton>
        </div>
      </div>
    </Transition>

    <!-- 빠른 액션 패널 -->
    <QuickActionPanel
      v-model:visible="showQuickActions"
      title="터미널 옵션"
    >
      <div class="terminal-options">
        <TouchButton
          block
          text
          size="lg"
          @click="clearTerminal"
        >
          <Icon name="clear" /> 터미널 지우기
        </TouchButton>

        <TouchButton
          block
          text
          size="lg"
          @click="toggleAutoScroll"
        >
          <Icon :name="autoScrollEnabled ? 'check-box' : 'check-box-outline'" />
          자동 스크롤
        </TouchButton>

        <TouchButton
          block
          text
          size="lg"
          @click="showSearch = true"
        >
          <Icon name="search" /> 검색
        </TouchButton>

        <TouchButton
          block
          text
          size="lg"
          @click="exportLogs"
        >
          <Icon name="download" /> 로그 내보내기
        </TouchButton>

        <TouchButton
          v-if="isExecuting"
          block
          text
          type="error"
          size="lg"
          @click="stopExecution"
        >
          <Icon name="stop" /> 실행 중지
        </TouchButton>
      </div>
    </QuickActionPanel>

    <!-- 검색 오버레이 -->
    <Transition name="fade">
      <div v-if="showSearch" class="search-overlay">
        <div class="search-header">
          <input
            ref="searchInputRef"
            v-model="searchQuery"
            type="text"
            placeholder="검색어 입력..."
            class="search-input"
            @input="handleSearch"
          />
          <TouchButton text size="lg" @click="closeSearch">
            <Icon name="close" />
          </TouchButton>
        </div>

        <div v-if="searchResults.length > 0" class="search-results">
          <div class="search-info">
            {{ currentSearchIndex + 1 }} / {{ searchResults.length }}
          </div>
          <div class="search-nav">
            <TouchButton
              text
              size="md"
              :disabled="searchResults.length === 0"
              @click="previousSearchResult"
            >
              <Icon name="arrow-up" />
            </TouchButton>
            <TouchButton
              text
              size="md"
              :disabled="searchResults.length === 0"
              @click="nextSearchResult"
            >
              <Icon name="arrow-down" />
            </TouchButton>
          </div>
        </div>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { Icon } from '@iconify/vue'
import TouchButton from '../form/TouchButton.vue'
import QuickActionPanel from './QuickActionPanel.vue'
import { useTouchInteractions } from '@/composables/useTouchInteractions'
import { useAdvancedGestures } from '@/composables/useAdvancedGestures'
import { useOrientationAdaptation } from '@/composables/useOrientationAdaptation'
import { useMobileWorkflow } from '@/composables/useMobileWorkflow'

interface TerminalLog {
  id: string
  content: string
  type: 'input' | 'output' | 'error' | 'system'
  timestamp: string
}

interface Props {
  sessionName?: string
  logs?: TerminalLog[]
  isConnected?: boolean
  isExecuting?: boolean
  commandHistory?: string[]
}

const props = withDefaults(defineProps<Props>(), {
  sessionName: 'Terminal',
  logs: () => [],
  isConnected: false,
  isExecuting: false,
  commandHistory: () => []
})

const emit = defineEmits<{
  close: []
  executeCommand: [command: string]
  stopExecution: []
  clearLogs: []
  exportLogs: []
}>()

// Composables
const { triggerHapticFeedback } = useTouchInteractions()
const { setupGestureZone, removeGestureZone } = useAdvancedGestures()
const { isLandscape, orientation } = useOrientationAdaptation()
const { enableOneHandMode, getQuickActions } = useMobileWorkflow()

// Refs
const terminalContentRef = ref<HTMLElement>()
const scrollContainerRef = ref<HTMLElement>()
const commandInputRef = ref<HTMLInputElement>()
const searchInputRef = ref<HTMLInputElement>()

// State
const currentCommand = ref('')
const isInputFocused = ref(false)
const isFullscreen = ref(false)
const showQuickActions = ref(false)
const showQuickCommands = ref(false)
const showSearch = ref(false)
const searchQuery = ref('')
const searchResults = ref<number[]>([])
const currentSearchIndex = ref(-1)
const autoScrollEnabled = ref(true)
const isAtBottom = ref(true)
const hasNewContent = ref(false)
const scrollOffset = ref(0)
const isKeyboardVisible = ref(false)

// Command history
const historyIndex = ref(-1)
const commandHistoryLocal = ref<string[]>([...props.commandHistory])

// Quick commands
const quickCommands = [
  { id: 'ls', label: 'ls', command: 'ls -la' },
  { id: 'cd', label: 'cd ..', command: 'cd ..' },
  { id: 'git', label: 'git status', command: 'git status' },
  { id: 'clear', label: 'clear', command: 'clear' },
  { id: 'pwd', label: 'pwd', command: 'pwd' },
  { id: 'npm', label: 'npm run', command: 'npm run ' }
]

// Computed
const connectionClass = computed(() => {
  return props.isConnected ? 'connected' : 'disconnected'
})

const connectionIcon = computed(() => {
  return props.isConnected ? 'wifi' : 'wifi-off'
})

const visibleLogs = computed(() => {
  // 모바일에서는 최대 500개의 로그만 표시
  const maxLogs = 500
  if (props.logs.length > maxLogs) {
    return props.logs.slice(-maxLogs)
  }
  return props.logs
})

const scrollStyle = computed(() => ({
  transform: `translateY(${scrollOffset.value}px)`
}))

// Methods
const getLogClass = (log: TerminalLog) => {
  return `log-${log.type}`
}

const highlightAnsi = (content: string) => {
  // 간단한 ANSI 색상 변환
  return content
    .replace(/\x1b\[31m/g, '<span style="color: #ff6b6b;">')
    .replace(/\x1b\[32m/g, '<span style="color: #51cf66;">')
    .replace(/\x1b\[33m/g, '<span style="color: #ffd93d;">')
    .replace(/\x1b\[34m/g, '<span style="color: #74c0fc;">')
    .replace(/\x1b\[0m/g, '</span>')
}

const executeCommand = () => {
  const command = currentCommand.value.trim()
  if (!command) return

  // 명령어 히스토리에 추가
  commandHistoryLocal.value.push(command)
  historyIndex.value = commandHistoryLocal.value.length

  emit('executeCommand', command)
  currentCommand.value = ''
  
  triggerHapticFeedback('light')
}

const stopExecution = () => {
  emit('stopExecution')
  triggerHapticFeedback('warning')
}

const clearTerminal = () => {
  emit('clearLogs')
  showQuickActions.value = false
  triggerHapticFeedback('light')
}

const exportLogs = () => {
  emit('exportLogs')
  showQuickActions.value = false
}

const toggleFullscreen = () => {
  isFullscreen.value = !isFullscreen.value
  
  if (isFullscreen.value) {
    document.documentElement.requestFullscreen?.()
  } else {
    document.exitFullscreen?.()
  }
  
  triggerHapticFeedback('light')
}

const toggleAutoScroll = () => {
  autoScrollEnabled.value = !autoScrollEnabled.value
  if (autoScrollEnabled.value) {
    scrollToBottom()
  }
}

const scrollToBottom = () => {
  if (scrollContainerRef.value) {
    scrollContainerRef.value.scrollTop = scrollContainerRef.value.scrollHeight
    isAtBottom.value = true
    hasNewContent.value = false
  }
}

const selectQuickCommand = (command: string) => {
  currentCommand.value = command
  showQuickCommands.value = false
  commandInputRef.value?.focus()
  triggerHapticFeedback('selection')
}

// Touch handling
let touchStartY = 0
let touchStartTime = 0

const handleTouchStart = (e: TouchEvent) => {
  touchStartY = e.touches[0].clientY
  touchStartTime = Date.now()
}

const handleTouchMove = (e: TouchEvent) => {
  const touchY = e.touches[0].clientY
  const deltaY = touchY - touchStartY
  
  // 스크롤 처리는 기본 동작에 맡김
  
  // 자동 스크롤 비활성화 체크
  if (scrollContainerRef.value) {
    const { scrollTop, scrollHeight, clientHeight } = scrollContainerRef.value
    isAtBottom.value = scrollTop + clientHeight >= scrollHeight - 10
    
    if (!isAtBottom.value && autoScrollEnabled.value) {
      autoScrollEnabled.value = false
    }
  }
}

const handleTouchEnd = (e: TouchEvent) => {
  const touchEndTime = Date.now()
  const touchDuration = touchEndTime - touchStartTime
  
  // 빠른 탭으로 명령어 입력창 포커스
  if (touchDuration < 200) {
    commandInputRef.value?.focus()
  }
}

// Input handling
const handleInputFocus = () => {
  isInputFocused.value = true
  isKeyboardVisible.value = true
  
  // 한 손 모드 활성화
  if (window.innerHeight < 700) {
    enableOneHandMode(true)
  }
}

const handleInputBlur = () => {
  isInputFocused.value = false
  
  // 키보드가 사라질 때까지 약간의 지연
  setTimeout(() => {
    isKeyboardVisible.value = false
    enableOneHandMode(false)
  }, 300)
}

const handleKeyDown = (e: KeyboardEvent) => {
  switch (e.key) {
    case 'Enter':
      e.preventDefault()
      executeCommand()
      break
      
    case 'ArrowUp':
      e.preventDefault()
      if (historyIndex.value > 0) {
        historyIndex.value--
        currentCommand.value = commandHistoryLocal.value[historyIndex.value]
      }
      break
      
    case 'ArrowDown':
      e.preventDefault()
      if (historyIndex.value < commandHistoryLocal.value.length - 1) {
        historyIndex.value++
        currentCommand.value = commandHistoryLocal.value[historyIndex.value]
      } else {
        historyIndex.value = commandHistoryLocal.value.length
        currentCommand.value = ''
      }
      break
      
    case 'Tab':
      e.preventDefault()
      showQuickCommands.value = !showQuickCommands.value
      break
  }
}

// Search functionality
const handleSearch = () => {
  if (!searchQuery.value) {
    searchResults.value = []
    return
  }
  
  searchResults.value = visibleLogs.value
    .map((log, index) => ({ log, index }))
    .filter(({ log }) => log.content.toLowerCase().includes(searchQuery.value.toLowerCase()))
    .map(({ index }) => index)
    
  currentSearchIndex.value = searchResults.value.length > 0 ? 0 : -1
  
  if (currentSearchIndex.value >= 0) {
    scrollToSearchResult()
  }
}

const nextSearchResult = () => {
  if (searchResults.value.length === 0) return
  
  currentSearchIndex.value = (currentSearchIndex.value + 1) % searchResults.value.length
  scrollToSearchResult()
}

const previousSearchResult = () => {
  if (searchResults.value.length === 0) return
  
  currentSearchIndex.value = currentSearchIndex.value <= 0
    ? searchResults.value.length - 1
    : currentSearchIndex.value - 1
  scrollToSearchResult()
}

const scrollToSearchResult = () => {
  const logIndex = searchResults.value[currentSearchIndex.value]
  const logElement = scrollContainerRef.value?.children[logIndex] as HTMLElement
  
  if (logElement && scrollContainerRef.value) {
    const containerRect = scrollContainerRef.value.getBoundingClientRect()
    const elementRect = logElement.getBoundingClientRect()
    const scrollTop = elementRect.top - containerRect.top + scrollContainerRef.value.scrollTop
    
    scrollContainerRef.value.scrollTo({
      top: scrollTop - 100,
      behavior: 'smooth'
    })
  }
}

const closeSearch = () => {
  showSearch.value = false
  searchQuery.value = ''
  searchResults.value = []
  currentSearchIndex.value = -1
}

// Lifecycle
onMounted(() => {
  // 제스처 설정
  if (terminalContentRef.value) {
    setupGestureZone(terminalContentRef.value, {
      onSwipeDown: () => {
        if (isInputFocused.value) {
          commandInputRef.value?.blur()
        }
      },
      onSwipeUp: () => {
        if (!isInputFocused.value) {
          commandInputRef.value?.focus()
        }
      },
      onDoubleTap: () => {
        showQuickCommands.value = !showQuickCommands.value
      }
    })
  }
  
  // 초기 스크롤
  if (autoScrollEnabled.value) {
    nextTick(() => scrollToBottom())
  }
})

onUnmounted(() => {
  if (terminalContentRef.value) {
    removeGestureZone(terminalContentRef.value)
  }
})

// Watch for new logs
watch(() => props.logs.length, (newLength, oldLength) => {
  if (newLength > oldLength) {
    hasNewContent.value = !isAtBottom.value
    
    if (autoScrollEnabled.value) {
      nextTick(() => scrollToBottom())
    }
  }
})

// Watch search visibility
watch(showSearch, (visible) => {
  if (visible) {
    nextTick(() => searchInputRef.value?.focus())
  }
})
</script>

<style lang="scss" scoped>
@import '@/styles/variables';
@import '@/styles/mixins';

.mobile-terminal {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--bg-primary);
  color: var(--text-primary);
  font-family: var(--font-mono);
  position: relative;
  
  &--fullscreen {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    z-index: 9999;
  }
  
  &--landscape {
    .mobile-terminal__header {
      height: 48px;
    }
    
    .mobile-terminal__input {
      height: 48px;
    }
  }
  
  &--keyboard-visible {
    .mobile-terminal__content {
      // iOS 키보드 대응
      padding-bottom: env(keyboard-inset-height, 0);
    }
  }
}

.mobile-terminal__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 56px;
  padding: 0 8px;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border-color);
  flex-shrink: 0;
  
  @include safe-area-padding-top;
}

.mobile-terminal__back {
  flex-shrink: 0;
}

.mobile-terminal__title {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  font-size: 16px;
  font-weight: 500;
  
  .connection-status {
    display: flex;
    align-items: center;
    
    &.connected {
      color: var(--success-color);
    }
    
    &.disconnected {
      color: var(--text-tertiary);
    }
  }
}

.mobile-terminal__actions {
  display: flex;
  gap: 4px;
  flex-shrink: 0;
}

.mobile-terminal__content {
  flex: 1;
  overflow: hidden;
  position: relative;
  background: #1a1a1a;
  -webkit-overflow-scrolling: touch;
}

.scroll-container {
  height: 100%;
  overflow-y: auto;
  padding: 12px;
  
  &::-webkit-scrollbar {
    width: 4px;
  }
  
  &::-webkit-scrollbar-thumb {
    background: var(--text-tertiary);
    border-radius: 2px;
  }
}

.terminal-line {
  display: flex;
  align-items: flex-start;
  margin-bottom: 4px;
  font-size: 14px;
  line-height: 1.5;
  word-break: break-word;
  
  @include touch-target(32px);
  
  .line-number {
    color: var(--text-tertiary);
    font-size: 12px;
    min-width: 40px;
    flex-shrink: 0;
    user-select: none;
    opacity: 0.5;
  }
  
  .line-content {
    flex: 1;
    white-space: pre-wrap;
  }
  
  .prompt {
    color: #51cf66;
    font-weight: bold;
  }
  
  &.log-input {
    color: #ffffff;
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

.new-content-indicator {
  position: absolute;
  bottom: 16px;
  right: 16px;
  background: var(--primary-color);
  color: white;
  padding: 8px 16px;
  border-radius: 20px;
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
  
  @include touch-target;
  @include touch-feedback;
}

.mobile-terminal__input {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: var(--bg-secondary);
  border-top: 1px solid var(--border-color);
  min-height: 56px;
  
  @include safe-area-padding-bottom;
  
  &--focused {
    background: var(--bg-tertiary);
  }
}

.input-wrapper {
  flex: 1;
  display: flex;
  align-items: center;
  background: var(--bg-primary);
  border-radius: 24px;
  padding: 0 16px;
  height: 40px;
  
  .prompt {
    color: #51cf66;
    font-weight: bold;
    margin-right: 8px;
    flex-shrink: 0;
  }
  
  .command-input {
    flex: 1;
    background: none;
    border: none;
    outline: none;
    color: var(--text-primary);
    font-family: var(--font-mono);
    font-size: 14px;
    
    &::placeholder {
      color: var(--text-tertiary);
    }
    
    &:disabled {
      opacity: 0.5;
    }
  }
}

.input-actions {
  flex-shrink: 0;
}

// Quick commands
.quick-commands {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  background: var(--bg-primary);
  border-top: 1px solid var(--border-color);
  border-radius: 16px 16px 0 0;
  box-shadow: 0 -4px 16px rgba(0, 0, 0, 0.1);
  max-height: 50vh;
  overflow-y: auto;
  
  @include safe-area-padding-bottom;
}

.quick-commands__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border-color);
  font-weight: 500;
  position: sticky;
  top: 0;
  background: var(--bg-primary);
}

.quick-commands__list {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
  gap: 8px;
  padding: 12px;
}

.quick-command {
  justify-content: flex-start;
  font-family: var(--font-mono);
  font-size: 14px;
}

// Terminal options
.terminal-options {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

// Search overlay
.search-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  background: var(--bg-primary);
  border-bottom: 1px solid var(--border-color);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  z-index: 100;
  
  @include safe-area-padding-top;
}

.search-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  height: 56px;
}

.search-input {
  flex: 1;
  height: 40px;
  padding: 0 16px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 20px;
  font-size: 16px;
  outline: none;
  
  &:focus {
    border-color: var(--primary-color);
  }
}

.search-results {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 16px;
  border-top: 1px solid var(--border-color);
}

.search-info {
  font-size: 14px;
  color: var(--text-secondary);
}

.search-nav {
  display: flex;
  gap: 8px;
}

// Animations
.slide-up-enter-active,
.slide-up-leave-active {
  transition: transform 0.3s ease;
}

.slide-up-enter-from,
.slide-up-leave-to {
  transform: translateY(100%);
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

// Landscape mode
@media (orientation: landscape) {
  .mobile-terminal__content {
    .scroll-container {
      padding: 8px;
    }
  }
  
  .terminal-line {
    margin-bottom: 2px;
    font-size: 13px;
  }
  
  .quick-commands {
    max-height: 70vh;
  }
}

// Tablet optimizations
@media (min-width: 768px) {
  .mobile-terminal__header {
    height: 64px;
    padding: 0 16px;
  }
  
  .terminal-line {
    font-size: 15px;
    
    .line-number {
      min-width: 60px;
    }
  }
  
  .mobile-terminal__input {
    padding: 12px 16px;
    
    .input-wrapper {
      height: 48px;
      padding: 0 20px;
    }
    
    .command-input {
      font-size: 16px;
    }
  }
  
  .quick-commands__list {
    grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
    gap: 12px;
    padding: 16px;
  }
}
</style>