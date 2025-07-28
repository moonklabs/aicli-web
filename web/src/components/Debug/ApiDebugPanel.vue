<template>
  <Teleport to="body">
    <div
      v-if="isDev && isVisible"
      class="fixed bottom-4 left-4 z-50 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg shadow-xl max-w-sm w-full max-h-96 overflow-hidden"
    >
      <!-- 헤더 -->
      <div class="bg-gray-50 dark:bg-gray-700 px-4 py-2 border-b border-gray-200 dark:border-gray-600 flex items-center justify-between">
        <div class="flex items-center gap-2">
          <span class="text-blue-500">🔗</span>
          <h3 class="font-medium text-sm text-gray-900 dark:text-gray-100">API 디버거</h3>
          <div class="flex items-center gap-1">
            <!-- 네트워크 상태 표시 -->
            <div
              :class="[
                'w-2 h-2 rounded-full',
                networkStatus.isOnline ? 'bg-green-500' : 'bg-red-500'
              ]"
            />
            <span class="text-xs text-gray-600 dark:text-gray-400">
              {{ networkStatus.isOnline ? 'Online' : 'Offline' }}
            </span>
          </div>
        </div>

        <div class="flex items-center gap-1">
          <!-- 자동 스크롤 토글 -->
          <button
            @click="autoScroll = !autoScroll"
            :class="[
              'p-1 rounded text-xs transition-colors',
              autoScroll
                ? 'bg-blue-500 text-white'
                : 'bg-gray-200 dark:bg-gray-600 text-gray-600 dark:text-gray-400'
            ]"
            title="자동 스크롤"
          >
            <span>⬇️</span>
          </button>

          <!-- 일시정지/재개 -->
          <button
            @click="isPaused = !isPaused"
            :class="[
              'p-1 rounded text-xs transition-colors',
              isPaused
                ? 'bg-yellow-500 text-white'
                : 'bg-gray-200 dark:bg-gray-600 text-gray-600 dark:text-gray-400'
            ]"
            :title="isPaused ? '재개' : '일시정지'"
          >
            <span>{{ isPaused ? '▶️' : '⏸️' }}</span>
          </button>

          <!-- 로그 지우기 -->
          <button
            @click="clearLogs"
            class="p-1 rounded text-xs bg-gray-200 dark:bg-gray-600 text-gray-600 dark:text-gray-400 hover:bg-red-500 hover:text-white transition-colors"
            title="로그 지우기"
          >
            <span>🗑️</span>
          </button>

          <!-- 최소화/닫기 -->
          <button
            @click="minimize"
            class="p-1 rounded text-xs bg-gray-200 dark:bg-gray-600 text-gray-600 dark:text-gray-400 hover:bg-gray-300 dark:hover:bg-gray-500 transition-colors"
            title="최소화"
          >
            <span>➖</span>
          </button>

          <button
            @click="isVisible = false"
            class="p-1 rounded text-xs bg-gray-200 dark:bg-gray-600 text-gray-600 dark:text-gray-400 hover:bg-red-500 hover:text-white transition-colors"
            title="닫기"
          >
            <span>✖️</span>
          </button>
        </div>
      </div>

      <!-- 통계 정보 -->
      <div v-if="!isMinimized" class="px-4 py-2 bg-gray-50 dark:bg-gray-700 border-b border-gray-200 dark:border-gray-600">
        <div class="grid grid-cols-3 gap-2 text-xs">
          <div class="text-center">
            <div class="font-medium text-gray-900 dark:text-gray-100">{{ stats.total }}</div>
            <div class="text-gray-600 dark:text-gray-400">총 요청</div>
          </div>
          <div class="text-center">
            <div class="font-medium text-gray-900 dark:text-gray-100">{{ stats.pending }}</div>
            <div class="text-gray-600 dark:text-gray-400">진행 중</div>
          </div>
          <div class="text-center">
            <div class="font-medium text-gray-900 dark:text-gray-100">{{ stats.cached }}</div>
            <div class="text-gray-600 dark:text-gray-400">캐시됨</div>
          </div>
        </div>

        <!-- 필터 버튼들 -->
        <div class="flex gap-1 mt-2">
          <button
            v-for="filter in filters"
            :key="filter.key"
            @click="activeFilter = activeFilter === filter.key ? 'all' : filter.key"
            :class="[
              'px-2 py-1 text-xs rounded transition-colors',
              activeFilter === filter.key
                ? filter.activeClass
                : 'bg-gray-200 dark:bg-gray-600 text-gray-600 dark:text-gray-400 hover:bg-gray-300 dark:hover:bg-gray-500'
            ]"
          >
            {{ filter.label }}
          </button>
        </div>
      </div>

      <!-- API 로그 -->
      <div
        v-if="!isMinimized"
        ref="logContainer"
        class="max-h-64 overflow-y-auto p-2 space-y-1"
      >
        <div
          v-for="log in filteredLogs"
          :key="log.id"
          :class="[
            'text-xs p-2 rounded border-l-2 cursor-pointer transition-colors',
            getLogStyles(log),
            selectedLog?.id === log.id ? 'ring-2 ring-blue-500' : ''
          ]"
          @click="selectLog(log)"
        >
          <div class="flex items-center justify-between mb-1">
            <div class="flex items-center gap-2">
              <span>{{ getLogIcon(log) }}</span>
              <span class="font-medium">{{ log.method }}</span>
              <span class="text-gray-600 dark:text-gray-400">{{ getUrlPath(log.url) }}</span>
            </div>
            <div class="flex items-center gap-1">
              <span :class="getStatusStyles(log.status)">{{ log.status || 'pending' }}</span>
              <span class="text-gray-500">{{ formatDuration(log.duration) }}</span>
            </div>
          </div>

          <div v-if="log.error" class="text-red-600 dark:text-red-400 text-xs">
            {{ log.error }}
          </div>

          <div class="text-gray-500 text-xs">
            {{ formatTime(log.timestamp) }}
            <span v-if="log.cached" class="ml-2 text-yellow-600">📋 캐시됨</span>
            <span v-if="log.retryCount > 0" class="ml-2 text-blue-600">🔄 재시도 {{ log.retryCount }}회</span>
          </div>
        </div>

        <div v-if="filteredLogs.length === 0" class="text-center text-gray-500 py-4">
          {{ isPaused ? '일시정지됨' : '로그가 없습니다' }}
        </div>
      </div>
    </div>

    <!-- 로그 상세 모달 -->
    <div
      v-if="selectedLog && showDetails"
      class="fixed inset-0 z-60 bg-black bg-opacity-50 flex items-center justify-center p-4"
      @click="closeDetails"
    >
      <div
        class="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-2xl w-full max-h-96 overflow-hidden"
        @click.stop
      >
        <div class="bg-gray-50 dark:bg-gray-700 px-4 py-2 border-b border-gray-200 dark:border-gray-600 flex items-center justify-between">
          <h3 class="font-medium text-gray-900 dark:text-gray-100">API 요청 상세정보</h3>
          <button
            @click="closeDetails"
            class="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
          >
            <span>✖️</span>
          </button>
        </div>

        <div class="p-4 overflow-y-auto max-h-80">
          <pre class="text-xs bg-gray-100 dark:bg-gray-900 p-3 rounded overflow-auto">{{ JSON.stringify(selectedLog, null, 2) }}</pre>
        </div>
      </div>
    </div>

    <!-- 최소화된 버튼 -->
    <button
      v-if="isDev && !isVisible"
      @click="isVisible = true"
      class="fixed bottom-4 left-4 z-50 bg-blue-500 hover:bg-blue-600 text-white p-2 rounded-full shadow-lg transition-colors"
      title="API 디버거 열기"
    >
      <span>🔗</span>
    </button>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
// import { Icon } from '@iconify/vue'
import { useNetworkStatus } from '@/composables/useNetworkStatus'

interface ApiLog {
  id: string
  method: string
  url: string
  status?: number
  duration?: number
  error?: string
  cached?: boolean
  retryCount: number
  timestamp: Date
  requestData?: any
  responseData?: any
  headers?: any
}

const isDev = computed(() => import.meta.env.DEV)
const { networkStatus } = useNetworkStatus()

// 상태
const isVisible = ref(false)
const isMinimized = ref(false)
const isPaused = ref(false)
const autoScroll = ref(true)
const activeFilter = ref<string>('all')
const selectedLog = ref<ApiLog | null>(null)
const showDetails = ref(false)

// 로그 데이터
const logs = ref<ApiLog[]>([])
const logContainer = ref<HTMLElement>()

// 필터 설정
const filters = [
  { key: 'success', label: '성공', activeClass: 'bg-green-500 text-white' },
  { key: 'error', label: '오류', activeClass: 'bg-red-500 text-white' },
  { key: 'pending', label: '진행중', activeClass: 'bg-yellow-500 text-white' },
  { key: 'cached', label: '캐시', activeClass: 'bg-blue-500 text-white' },
]

// 통계 계산
const stats = computed(() => {
  const total = logs.value.length
  const pending = logs.value.filter(log => !log.status).length
  const cached = logs.value.filter(log => log.cached).length

  return { total, pending, cached }
})

// 필터된 로그
const filteredLogs = computed(() => {
  if (activeFilter.value === 'all') return logs.value

  return logs.value.filter(log => {
    switch (activeFilter.value) {
      case 'success':
        return log.status && log.status >= 200 && log.status < 300
      case 'error':
        return log.status && log.status >= 400
      case 'pending':
        return !log.status
      case 'cached':
        return log.cached
      default:
        return true
    }
  })
})

// 로그 스타일
const getLogStyles = (log: ApiLog): string => {
  if (!log.status) {
    return 'bg-yellow-50 dark:bg-yellow-900/20 border-yellow-400'
  } else if (log.status >= 200 && log.status < 300) {
    return 'bg-green-50 dark:bg-green-900/20 border-green-400'
  } else if (log.status >= 400) {
    return 'bg-red-50 dark:bg-red-900/20 border-red-400'
  }
  return 'bg-gray-50 dark:bg-gray-700 border-gray-400'
}

// 로그 아이콘
const getLogIcon = (log: ApiLog): string => {
  if (!log.status) return '⏳'
  if (log.status >= 200 && log.status < 300) return '✅'
  if (log.status >= 400) return '❌'
  return 'ℹ️'
}

// 상태 코드 스타일
const getStatusStyles = (status?: number): string => {
  if (!status) return 'text-yellow-600'
  if (status >= 200 && status < 300) return 'text-green-600'
  if (status >= 400) return 'text-red-600'
  return 'text-gray-600'
}

// URL 경로 추출
const getUrlPath = (url: string): string => {
  try {
    return new URL(url, window.location.origin).pathname
  } catch {
    return url
  }
}

// 시간 포맷팅
const formatTime = (date: Date): string => {
  return date.toLocaleTimeString()
}

// 지속시간 포맷팅
const formatDuration = (duration?: number): string => {
  if (!duration) return '...'
  if (duration < 1000) return `${duration}ms`
  return `${(duration / 1000).toFixed(2)}s`
}

// 로그 추가
const addLog = (log: Omit<ApiLog, 'id' | 'timestamp'>): void => {
  if (isPaused.value) return

  const newLog: ApiLog = {
    ...log,
    id: `log-${Date.now()}-${Math.random()}`,
    timestamp: new Date(),
  }

  logs.value.unshift(newLog)

  // 최대 100개 로그만 유지
  if (logs.value.length > 100) {
    logs.value = logs.value.slice(0, 100)
  }

  // 자동 스크롤
  if (autoScroll.value) {
    nextTick(() => {
      if (logContainer.value) {
        logContainer.value.scrollTop = 0
      }
    })
  }
}

// 로그 업데이트
const updateLog = (id: string, updates: Partial<ApiLog>): void => {
  const log = logs.value.find(l => l.id === id)
  if (log) {
    Object.assign(log, updates)
  }
}

// 로그 선택
const selectLog = (log: ApiLog): void => {
  selectedLog.value = log
  showDetails.value = true
}

// 상세 정보 닫기
const closeDetails = (): void => {
  showDetails.value = false
  selectedLog.value = null
}

// 로그 지우기
const clearLogs = (): void => {
  logs.value = []
}

// 최소화 토글
const minimize = (): void => {
  isMinimized.value = !isMinimized.value
}

// Axios 인터셉터 설치
let requestInterceptor: number | null = null
let responseInterceptor: number | null = null

const installInterceptors = () => {
  if (typeof window !== 'undefined' && (window as any).axios) {
    const axios = (window as any).axios

    // 요청 인터셉터
    requestInterceptor = axios.interceptors.request.use((config: any) => {
      const logId = `req-${Date.now()}-${Math.random()}`
      config.logId = logId

      addLog({
        method: config.method?.toUpperCase() || 'UNKNOWN',
        url: config.url || '',
        retryCount: config.metadata?.retryCount || 0,
        requestData: config.data,
        headers: config.headers,
      })

      return config
    })

    // 응답 인터셉터
    responseInterceptor = axios.interceptors.response.use(
      (response: any) => {
        if (response.config.logId) {
          const log = logs.value.find(l =>
            l.url === response.config.url &&
            l.method === response.config.method?.toUpperCase(),
          )

          if (log) {
            updateLog(log.id, {
              status: response.status,
              duration: Date.now() - log.timestamp.getTime(),
              cached: !!response.fromCache,
              responseData: response.data,
            })
          }
        }

        return response
      },
      (error: any) => {
        if (error.config?.logId) {
          const log = logs.value.find(l =>
            l.url === error.config.url &&
            l.method === error.config.method?.toUpperCase(),
          )

          if (log) {
            updateLog(log.id, {
              status: error.response?.status || 0,
              duration: Date.now() - log.timestamp.getTime(),
              error: error.message,
              responseData: error.response?.data,
            })
          }
        }

        return Promise.reject(error)
      },
    )
  }
}

// 키보드 단축키
const handleKeyDown = (event: KeyboardEvent) => {
  if (event.ctrlKey && event.shiftKey && event.key === 'D') {
    event.preventDefault()
    isVisible.value = !isVisible.value
  }
}

onMounted(() => {
  // 개발 환경에서만 활성화
  if (isDev.value) {
    installInterceptors()
    window.addEventListener('keydown', handleKeyDown)

    // 전역 함수로 노출
    ;(window as any).__apiDebugger = {
      show: () => { isVisible.value = true },
      hide: () => { isVisible.value = false },
      toggle: () => { isVisible.value = !isVisible.value },
      addLog,
      clearLogs,
    }
  }
})

onUnmounted(() => {
  if (requestInterceptor !== null && typeof window !== 'undefined' && (window as any).axios) {
    (window as any).axios.interceptors.request.eject(requestInterceptor)
  }

  if (responseInterceptor !== null && typeof window !== 'undefined' && (window as any).axios) {
    (window as any).axios.interceptors.response.eject(responseInterceptor)
  }

  window.removeEventListener('keydown', handleKeyDown)
})

// 개발 환경에서 초기 표시
watch(isDev, (dev) => {
  if (dev) {
    // 5초 후 자동으로 표시 (초기 로딩 후)
    setTimeout(() => {
      isVisible.value = true
    }, 5000)
  }
}, { immediate: true })
</script>

<style scoped>
/* 스크롤바 스타일링 */
.overflow-y-auto::-webkit-scrollbar {
  width: 4px;
}

.overflow-y-auto::-webkit-scrollbar-track {
  background: transparent;
}

.overflow-y-auto::-webkit-scrollbar-thumb {
  background: rgba(156, 163, 175, 0.5);
  border-radius: 2px;
}

.overflow-y-auto::-webkit-scrollbar-thumb:hover {
  background: rgba(156, 163, 175, 0.8);
}
</style>