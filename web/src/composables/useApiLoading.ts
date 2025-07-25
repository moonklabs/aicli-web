import { ref, computed, watch, type Ref } from 'vue'
import type { AxiosResponse, AxiosError } from 'axios'

export interface LoadingState {
  isLoading: boolean
  error: string | null
  progress: number
  startTime: number | null
  duration: number
  retryCount: number
}

export interface ApiCallOptions {
  loadingMessage?: string
  errorMessage?: string
  showProgress?: boolean
  timeout?: number
  retryCount?: number
  onProgress?: (progress: number) => void
  onRetry?: (attempt: number) => void
}

// 전역 로딩 상태 관리
const globalLoadingStates = ref<Map<string, LoadingState>>(new Map())
const globalLoadingCount = computed(() => {
  return Array.from(globalLoadingStates.value.values()).filter(state => state.isLoading).length
})

export function useApiLoading(key?: string) {
  const uniqueKey = key || `api_${Date.now()}_${Math.random()}`
  const loadingMessage = ref<string>('')
  const errorMessage = ref<string>('')

  // 개별 로딩 상태
  const loadingState = ref<LoadingState>({
    isLoading: false,
    error: null,
    progress: 0,
    startTime: null,
    duration: 0,
    retryCount: 0
  })

  // 계산된 속성들
  const isLoading = computed(() => loadingState.value.isLoading)
  const error = computed(() => loadingState.value.error)
  const progress = computed(() => loadingState.value.progress)
  const duration = computed(() => loadingState.value.duration)
  const retryCount = computed(() => loadingState.value.retryCount)

  // 로딩 시작
  const startLoading = (message?: string) => {
    loadingState.value = {
      isLoading: true,
      error: null,
      progress: 0,
      startTime: Date.now(),
      duration: 0,
      retryCount: 0
    }
    
    if (message) {
      loadingMessage.value = message
    }

    // 전역 상태에 추가
    globalLoadingStates.value.set(uniqueKey, { ...loadingState.value })
  }

  // 진행률 업데이트
  const updateProgress = (newProgress: number) => {
    if (loadingState.value.isLoading) {
      loadingState.value.progress = Math.max(0, Math.min(100, newProgress))
      
      // 전역 상태 업데이트
      const globalState = globalLoadingStates.value.get(uniqueKey)
      if (globalState) {
        globalState.progress = loadingState.value.progress
      }
    }
  }

  // 재시도 카운트 증가
  const incrementRetry = () => {
    loadingState.value.retryCount += 1
    
    // 전역 상태 업데이트
    const globalState = globalLoadingStates.value.get(uniqueKey)
    if (globalState) {
      globalState.retryCount = loadingState.value.retryCount
    }
  }

  // 로딩 완료
  const finishLoading = (error?: string | Error | null) => {
    const endTime = Date.now()
    
    loadingState.value.isLoading = false
    loadingState.value.duration = loadingState.value.startTime ? 
      endTime - loadingState.value.startTime : 0
    
    if (error) {
      const errorStr = error instanceof Error ? error.message : error
      loadingState.value.error = errorStr
      errorMessage.value = errorStr
    } else {
      loadingState.value.error = null
      errorMessage.value = ''
      loadingState.value.progress = 100
    }

    // 전역 상태에서 제거
    globalLoadingStates.value.delete(uniqueKey)
  }

  // API 호출 래퍼
  const executeApi = async <T>(
    apiCall: () => Promise<AxiosResponse<T>>,
    options: ApiCallOptions = {}
  ): Promise<T> => {
    const {
      loadingMessage: message,
      errorMessage: customErrorMessage,
      showProgress = false,
      timeout = 30000,
      retryCount: maxRetries = 0,
      onProgress,
      onRetry
    } = options

    startLoading(message)

    try {
      // 진행률 시뮬레이션 (showProgress가 true인 경우)
      let progressInterval: NodeJS.Timeout | null = null
      
      if (showProgress) {
        let currentProgress = 0
        progressInterval = setInterval(() => {
          if (currentProgress < 90 && loadingState.value.isLoading) {
            currentProgress += Math.random() * 10
            updateProgress(currentProgress)
            onProgress?.(currentProgress)
          }
        }, 500)
      }

      // 타임아웃 설정
      const timeoutPromise = new Promise<never>((_, reject) => {
        setTimeout(() => {
          reject(new Error(`Request timeout after ${timeout}ms`))
        }, timeout)
      })

      let attempt = 0
      let lastError: any = null

      while (attempt <= maxRetries) {
        try {
          if (attempt > 0) {
            incrementRetry()
            onRetry?.(attempt)
            
            // 재시도 전 대기 (exponential backoff)
            const delay = Math.min(1000 * Math.pow(2, attempt - 1), 10000)
            await new Promise(resolve => setTimeout(resolve, delay))
          }

          const response = await Promise.race([
            apiCall(),
            timeoutPromise
          ])

          // 진행률 완료
          if (showProgress) {
            updateProgress(100)
            onProgress?.(100)
          }

          if (progressInterval) {
            clearInterval(progressInterval)
          }

          finishLoading()
          return response.data
        } catch (error) {
          lastError = error
          attempt++
          
          if (attempt > maxRetries) {
            throw error
          }
        }
      }

      throw lastError
    } catch (error) {
      if (progressInterval) {
        clearInterval(progressInterval)
      }

      let errorMsg = customErrorMessage || 'API 호출 중 오류가 발생했습니다'
      
      if (error instanceof Error) {
        if (error.message.includes('timeout')) {
          errorMsg = '요청 시간이 초과되었습니다'
        } else if (error.message.includes('Network Error')) {
          errorMsg = '네트워크 연결을 확인해주세요'
        } else {
          errorMsg = error.message
        }
      }

      finishLoading(errorMsg)
      throw error
    }
  }

  // 상태 초기화
  const reset = () => {
    loadingState.value = {
      isLoading: false,
      error: null,
      progress: 0,
      startTime: null,
      duration: 0,
      retryCount: 0
    }
    loadingMessage.value = ''
    errorMessage.value = ''
    
    // 전역 상태에서도 제거
    globalLoadingStates.value.delete(uniqueKey)
  }

  // 여러 API 호출 병렬 처리
  const executeParallel = async <T>(
    apiCalls: Array<() => Promise<AxiosResponse<T>>>,
    options: ApiCallOptions = {}
  ): Promise<T[]> => {
    startLoading(options.loadingMessage || '병렬 API 호출 중...')

    try {
      const promises = apiCalls.map((call, index) => {
        return call().then(response => {
          // 각 API 완료시 진행률 업데이트
          const progress = ((index + 1) / apiCalls.length) * 100
          updateProgress(progress)
          options.onProgress?.(progress)
          return response.data
        })
      })

      const results = await Promise.all(promises)
      finishLoading()
      return results
    } catch (error) {
      finishLoading(error instanceof Error ? error.message : '병렬 API 호출 실패')
      throw error
    }
  }

  // 전역 로딩 상태 감시
  watch(loadingState, (newState) => {
    if (newState.isLoading) {
      globalLoadingStates.value.set(uniqueKey, { ...newState })
    } else {
      globalLoadingStates.value.delete(uniqueKey)
    }
  }, { deep: true })

  return {
    // 상태
    isLoading,
    error,
    progress,
    duration,
    retryCount,
    loadingMessage: computed(() => loadingMessage.value),
    errorMessage: computed(() => errorMessage.value),

    // 메서드
    startLoading,
    finishLoading,
    updateProgress,
    incrementRetry,
    executeApi,
    executeParallel,
    reset,

    // 원시 상태 (필요한 경우)
    loadingState: computed(() => loadingState.value)
  }
}

// 전역 로딩 상태 관리
export function useGlobalLoading() {
  return {
    globalLoadingCount,
    globalLoadingStates: computed(() => globalLoadingStates.value),
    isGlobalLoading: computed(() => globalLoadingCount.value > 0),
    
    clearAllLoading: () => {
      globalLoadingStates.value.clear()
    },
    
    getLoadingByKey: (key: string) => {
      return globalLoadingStates.value.get(key)
    }
  }
}