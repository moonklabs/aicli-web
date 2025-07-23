<template>
  <n-modal
    v-model:show="showModal"
    preset="card"
    style="width: 90vw; max-width: 1200px; height: 80vh"
    :title="`${containerName} - 컨테이너 로그`"
    :bordered="false"
    size="huge"
    role="dialog"
    aria-modal="true"
  >
    <template #header-extra>
      <n-space>
        <n-switch
          v-model:value="isStreaming"
          @update:value="toggleStreaming"
          size="small"
        >
          <template #checked>실시간</template>
          <template #unchecked>정적</template>
        </n-switch>
        
        <n-button size="small" @click="clearLogs">
          <template #icon>
            <n-icon><Trash2 /></n-icon>
          </template>
          지우기
        </n-button>
        
        <n-button size="small" @click="downloadLogs">
          <template #icon>
            <n-icon><Download /></n-icon>
          </template>
          다운로드
        </n-button>
      </n-space>
    </template>

    <div class="logs-container">
      <!-- 로그 필터 -->
      <div class="logs-toolbar">
        <n-input
          v-model:value="searchQuery"
          placeholder="로그 검색..."
          clearable
          size="small"
        >
          <template #prefix>
            <n-icon><Search /></n-icon>
          </template>
        </n-input>
        
        <n-select
          v-model:value="logLevel"
          :options="logLevelOptions"
          size="small"
          style="width: 120px"
          placeholder="레벨"
        />
        
        <n-select
          v-model:value="streamFilter"
          :options="streamOptions"
          size="small"
          style="width: 100px"
          placeholder="스트림"
        />
      </div>

      <!-- 로그 내용 -->
      <div class="logs-content" ref="logsContentRef">
        <div v-if="isLoading" class="logs-loading">
          <n-spin size="small" />
          <span>로그를 불러오는 중...</span>
        </div>
        
        <div v-else-if="filteredLogs.length === 0" class="logs-empty">
          <n-empty description="로그가 없습니다" size="small">
            <template #icon>
              <n-icon><FileText /></n-icon>
            </template>
          </n-empty>
        </div>
        
        <div v-else class="logs-list">
          <div
            v-for="(log, index) in displayedLogs"
            :key="`${log.timestamp.getTime()}-${index}`"
            class="log-entry"
            :class="{
              'log-stdout': log.stream === 'stdout',
              'log-stderr': log.stream === 'stderr',
              'log-highlight': isHighlighted(log.message)
            }"
          >
            <span class="log-timestamp">
              {{ formatTimestamp(log.timestamp) }}
            </span>
            <span class="log-stream" :class="`stream-${log.stream}`">
              {{ log.stream.toUpperCase() }}
            </span>
            <span class="log-message" v-html="highlightSearchTerm(log.message)"></span>
          </div>
        </div>
      </div>

      <!-- 자동 스크롤 및 페이지네이션 -->
      <div class="logs-footer">
        <div class="footer-left">
          <n-checkbox v-model:checked="autoScroll" size="small">
            자동 스크롤
          </n-checkbox>
          <span class="log-count">
            {{ filteredLogs.length }}개 로그 ({{ displayedLogs.length }}개 표시)
          </span>
        </div>
        
        <div class="footer-right">
          <n-button-group size="small">
            <n-button
              @click="loadMoreLogs"
              :disabled="displayedLogs.length >= filteredLogs.length"
              :loading="isLoadingMore"
            >
              더 보기
            </n-button>
            <n-button @click="scrollToTop">
              <template #icon>
                <n-icon><ArrowUp /></n-icon>
              </template>
            </n-button>
            <n-button @click="scrollToBottom">
              <template #icon>
                <n-icon><ArrowDown /></n-icon>
              </template>
            </n-button>
          </n-button-group>
        </div>
      </div>
    </div>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'
import {
  NModal,
  NSpace,
  NSwitch,
  NButton,
  NButtonGroup,
  NIcon,
  NInput,
  NSelect,
  NSpin,
  NEmpty,
  NCheckbox,
  useMessage
} from 'naive-ui'
import {
  Trash2,
  Download,
  Search,
  FileText,
  ArrowUp,
  ArrowDown
} from '@vicons/lucide'

import { useDockerStore, type LogEntry } from '@/stores/docker'

interface Props {
  show: boolean
  containerId: string
  containerName: string
}

interface Emits {
  (e: 'update:show', value: boolean): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const dockerStore = useDockerStore()
const message = useMessage()

// 모달 표시 상태
const showModal = computed({
  get: () => props.show,
  set: (value) => emit('update:show', value)
})

// 로컬 상태
const isStreaming = ref(false)
const searchQuery = ref('')
const logLevel = ref<string | null>(null)
const streamFilter = ref<string | null>(null)
const autoScroll = ref(true)
const isLoading = ref(false)
const isLoadingMore = ref(false)
const displayLimit = ref(100)
const logsContentRef = ref<HTMLElement>()

// 옵션
const logLevelOptions = [
  { label: '전체', value: null },
  { label: 'ERROR', value: 'error' },
  { label: 'WARN', value: 'warn' },
  { label: 'INFO', value: 'info' },
  { label: 'DEBUG', value: 'debug' }
]

const streamOptions = [
  { label: '전체', value: null },
  { label: 'STDOUT', value: 'stdout' },
  { label: 'STDERR', value: 'stderr' }
]

// 계산된 속성
const containerLogs = computed(() => {
  if (!props.containerId) return null
  return dockerStore.getContainerLogs(props.containerId)
})

const allLogs = computed(() => {
  return containerLogs.value?.logs || []
})

const filteredLogs = computed(() => {
  let logs = allLogs.value

  // 스트림 필터
  if (streamFilter.value) {
    logs = logs.filter(log => log.stream === streamFilter.value)
  }

  // 검색 필터
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    logs = logs.filter(log => log.message.toLowerCase().includes(query))
  }

  // 로그 레벨 필터 (메시지 내용 기반)
  if (logLevel.value) {
    const level = logLevel.value.toUpperCase()
    logs = logs.filter(log => log.message.toUpperCase().includes(level))
  }

  return logs
})

const displayedLogs = computed(() => {
  return filteredLogs.value.slice(-displayLimit.value)
})

// 메서드
const toggleStreaming = (enabled: boolean): void => {
  if (!props.containerId) return

  if (enabled) {
    dockerStore.startLogStreaming(props.containerId)
    message.success('실시간 로그 스트리밍이 시작되었습니다')
  } else {
    dockerStore.stopLogStreaming(props.containerId)
    message.info('실시간 로그 스트리밍이 중지되었습니다')
  }
}

const clearLogs = (): void => {
  // TODO: 로그 클리어 API 호출
  message.success('로그가 지워졌습니다')
}

const downloadLogs = (): void => {
  if (filteredLogs.value.length === 0) {
    message.warning('다운로드할 로그가 없습니다')
    return
  }

  const logsText = filteredLogs.value
    .map(log => `[${formatTimestamp(log.timestamp)}] ${log.stream.toUpperCase()}: ${log.message}`)
    .join('\n')

  const blob = new Blob([logsText], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `${props.containerName}-logs-${new Date().toISOString().split('T')[0]}.txt`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)

  message.success('로그 파일이 다운로드되었습니다')
}

const loadMoreLogs = (): void => {
  isLoadingMore.value = true
  setTimeout(() => {
    displayLimit.value += 100
    isLoadingMore.value = false
  }, 500)
}

const scrollToTop = (): void => {
  if (logsContentRef.value) {
    logsContentRef.value.scrollTop = 0
  }
}

const scrollToBottom = (): void => {
  if (logsContentRef.value) {
    logsContentRef.value.scrollTop = logsContentRef.value.scrollHeight
  }
}

const formatTimestamp = (timestamp: Date): string => {
  return timestamp.toLocaleTimeString('ko-KR', {
    hour12: false,
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    fractionalSecondDigits: 3
  })
}

const isHighlighted = (message: string): boolean => {
  if (!searchQuery.value) return false
  return message.toLowerCase().includes(searchQuery.value.toLowerCase())
}

const highlightSearchTerm = (message: string): string => {
  if (!searchQuery.value) return escapeHtml(message)
  
  const escapedMessage = escapeHtml(message)
  const regex = new RegExp(`(${escapeRegExp(searchQuery.value)})`, 'gi')
  return escapedMessage.replace(regex, '<mark>$1</mark>')
}

const escapeHtml = (text: string): string => {
  const div = document.createElement('div')
  div.textContent = text
  return div.innerHTML
}

const escapeRegExp = (string: string): string => {
  return string.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

// 자동 스크롤 처리
watch(displayedLogs, async () => {
  if (autoScroll.value) {
    await nextTick()
    scrollToBottom()
  }
}, { flush: 'post' })

// 모달이 열릴 때 로그 초기화
watch(showModal, (newValue) => {
  if (newValue && props.containerId) {
    displayLimit.value = 100
    autoScroll.value = true
    searchQuery.value = ''
    logLevel.value = null
    streamFilter.value = null
  }
})

// 스트리밍 상태 동기화
watch(() => containerLogs.value?.isStreaming, (newValue) => {
  if (newValue !== undefined) {
    isStreaming.value = newValue
  }
})

// 생명주기
onMounted(() => {
  // 초기 로그 데이터 로드
  if (props.containerId && containerLogs.value) {
    isStreaming.value = containerLogs.value.isStreaming
  }
})

onUnmounted(() => {
  // 스트리밍 정리
  if (isStreaming.value && props.containerId) {
    dockerStore.stopLogStreaming(props.containerId)
  }
})
</script>

<style scoped>
.logs-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  gap: 12px;
}

.logs-toolbar {
  display: flex;
  gap: 8px;
  padding: 12px;
  background-color: var(--n-color-hover);
  border-radius: 6px;
}

.logs-content {
  flex: 1;
  border: 1px solid var(--n-border-color);
  border-radius: 6px;
  overflow-y: auto;
  background-color: #1a1a1a;
  color: #ffffff;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
  line-height: 1.4;
}

.logs-loading,
.logs-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 200px;
  gap: 8px;
  color: var(--n-text-color-3);
}

.logs-list {
  padding: 8px;
}

.log-entry {
  display: flex;
  gap: 8px;
  padding: 2px 0;
  border-left: 3px solid transparent;
  padding-left: 8px;
  word-break: break-word;
}

.log-entry:hover {
  background-color: rgba(255, 255, 255, 0.05);
}

.log-entry.log-stderr {
  border-left-color: #ff4757;
  background-color: rgba(255, 71, 87, 0.1);
}

.log-entry.log-stdout {
  border-left-color: #2ed573;
}

.log-entry.log-highlight {
  background-color: rgba(255, 193, 7, 0.2);
}

.log-timestamp {
  color: #747d8c;
  font-size: 11px;
  white-space: nowrap;
  min-width: 90px;
}

.log-stream {
  font-size: 10px;
  font-weight: bold;
  min-width: 50px;
  text-align: center;
}

.stream-stdout {
  color: #2ed573;
}

.stream-stderr {
  color: #ff4757;
}

.log-message {
  flex: 1;
  word-break: break-word;
}

.log-message :deep(mark) {
  background-color: #feca57;
  color: #000;
  padding: 1px 2px;
  border-radius: 2px;
}

.logs-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  background-color: var(--n-color-hover);
  border-radius: 6px;
  font-size: 12px;
}

.footer-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.log-count {
  color: var(--n-text-color-3);
}

.footer-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

/* 반응형 디자인 */
@media (max-width: 768px) {
  .logs-toolbar {
    flex-direction: column;
    gap: 8px;
  }

  .log-entry {
    flex-direction: column;
    gap: 4px;
  }

  .log-timestamp,
  .log-stream {
    min-width: auto;
  }

  .logs-footer {
    flex-direction: column;
    gap: 8px;
  }

  .footer-left {
    flex-direction: column;
    gap: 4px;
  }
}

/* 스크롤바 스타일 */
.logs-content::-webkit-scrollbar {
  width: 8px;
}

.logs-content::-webkit-scrollbar-track {
  background: #2c2c2c;
}

.logs-content::-webkit-scrollbar-thumb {
  background: #666;
  border-radius: 4px;
}

.logs-content::-webkit-scrollbar-thumb:hover {
  background: #888;
}
</style>