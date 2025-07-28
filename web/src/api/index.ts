import axios, { type AxiosError, type AxiosInstance, type AxiosRequestConfig, type AxiosResponse } from 'axios'
import { useUserStore } from '@/stores/user'
import type { ApiError, ApiResponse } from '@/types/api'
import { mockApiMatcher } from '@/mocks/mockApi'
import { setupMockApi } from '@/mocks/mockAdapter'

// 재시도 및 캐싱 관련 타입
interface RetryConfig {
  maxRetries: number
  retryDelay: number
  exponentialBackoff: boolean
  retryCondition?: (error: AxiosError) => boolean
}

interface CacheConfig {
  enabled: boolean
  maxAge: number
  excludeHeaders?: string[]
}

interface ApiClientConfig {
  retry?: Partial<RetryConfig>
  cache?: Partial<CacheConfig>
  timeout?: number
}

// 기본 설정
const DEFAULT_RETRY_CONFIG: RetryConfig = {
  maxRetries: 3,
  retryDelay: 1000,
  exponentialBackoff: true,
  retryCondition: (error: AxiosError) => {
    return !error.response ||
           error.response.status >= 500 ||
           error.response.status === 408 || // Request Timeout
           error.response.status === 429 || // Too Many Requests
           error.code === 'ECONNABORTED' || // Timeout
           error.code === 'NETWORK_ERROR'
  },
}

const DEFAULT_CACHE_CONFIG: CacheConfig = {
  enabled: true,
  maxAge: 5 * 60 * 1000, // 5분
  excludeHeaders: ['authorization', 'cookie'],
}

// 전역 상태
const pendingRequests = new Map<string, Promise<AxiosResponse>>()
const requestCache = new Map<string, { data: unknown; timestamp: number; maxAge: number }>()
let isOnline = navigator.onLine
let globalRetryConfig = DEFAULT_RETRY_CONFIG
let globalCacheConfig = DEFAULT_CACHE_CONFIG

// 네트워크 상태 감지
window.addEventListener('online', () => {
  isOnline = true
  console.log('🌐 네트워크 연결됨')
})

window.addEventListener('offline', () => {
  isOnline = false
  console.log('🔌 네트워크 연결 끊어짐')
})

// 유틸리티 함수들
const generateCacheKey = (config: AxiosRequestConfig): string => {
  const { method, url, params, data } = config
  return `${method?.toUpperCase()}_${url}_${JSON.stringify(params)}_${JSON.stringify(data)}`
}

const generateRequestKey = (config: AxiosRequestConfig): string => {
  const { method, url, params } = config
  return `${method?.toUpperCase()}_${url}_${JSON.stringify(params)}`
}

const shouldCache = (config: AxiosRequestConfig): boolean => {
  return globalCacheConfig.enabled &&
         config.method?.toLowerCase() === 'get' &&
         !config.headers?.Authorization?.includes('Bearer')
}

const getCachedResponse = (cacheKey: string): unknown | null => {
  const cached = requestCache.get(cacheKey)
  if (cached && Date.now() - cached.timestamp < cached.maxAge) {
    return cached.data
  }
  if (cached) {
    requestCache.delete(cacheKey)
  }
  return null
}

const setCachedResponse = (cacheKey: string, data: unknown, maxAge: number = globalCacheConfig.maxAge): void => {
  requestCache.set(cacheKey, {
    data,
    timestamp: Date.now(),
    maxAge,
  })
}

const sleep = (ms: number): Promise<void> => new Promise(resolve => setTimeout(resolve, ms))

const calculateRetryDelay = (attempt: number, baseDelay: number, exponential: boolean): number => {
  if (!exponential) return baseDelay
  return baseDelay * Math.pow(2, attempt - 1)
}

// Axios 인스턴스 생성
const api: AxiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api',
  timeout: 30000, // 30초 타임아웃
  headers: {
    'Content-Type': 'application/json',
  },
})

// Mock API 설정 (개발 환경에서만)
if (import.meta.env.DEV) {
  setupMockApi(api)
}

// 요청 인터셉터 설정
api.interceptors.request.use(
  async (config: AxiosRequestConfig): Promise<AxiosRequestConfig> => {
    // 네트워크 연결 확인 (Mock API가 활성화되지 않은 경우에만)
    if (!isOnline && !(import.meta.env.DEV && mockApiMatcher.isActive())) {
      const error = new Error('Network is offline') as Error & { code: string }
      error.code = 'NETWORK_OFFLINE'
      throw error
    }

    // 인증 토큰 추가
    const userStore = useUserStore()
    if (userStore.authState.token) {
      if (config.headers) {
        config.headers.Authorization = `Bearer ${userStore.authState.token}`
      }
    }

    // 요청 중복 제거 (GET 요청만)
    if (config.method?.toLowerCase() === 'get') {
      const requestKey = generateRequestKey(config)

      if (pendingRequests.has(requestKey)) {
        console.log('🔄 Deduplicating request:', config.url)
        return pendingRequests.get(requestKey)
      }
    }

    // 캐시 확인 (GET 요청이고 캐시 가능한 경우)
    if (shouldCache(config)) {
      const cacheKey = generateCacheKey(config)
      const cachedResponse = getCachedResponse(cacheKey)

      if (cachedResponse) {
        console.log('💾 Using cached response:', config.url)
        return Promise.resolve({
          data: cachedResponse,
          status: 200,
          statusText: 'OK',
          headers: {},
          config,
          fromCache: true,
        })
      }
    }

    // 요청 로깅 (개발 환경에서만)
    if (import.meta.env.DEV) {
      console.log('🚀 API Request:', {
        method: config.method?.toUpperCase(),
        url: config.url,
        data: config.data,
        params: config.params,
      })
    }

    // 재시도 설정 추가
    config.metadata = { retryCount: 0 }

    return config
  },
  (error: AxiosError) => {
    console.error('❌ Request Error:', error)
    return Promise.reject(error)
  },
)

// 응답 인터셉터 설정
api.interceptors.response.use(
  (response: AxiosResponse<ApiResponse>): AxiosResponse<ApiResponse> => {
    // pending requests에서 제거
    if (response.config.method?.toLowerCase() === 'get') {
      const requestKey = generateRequestKey(response.config)
      pendingRequests.delete(requestKey)
    }

    // 캐시 가능한 응답이면 캐시에 저장
    if (shouldCache(response.config) && !response.fromCache) {
      const cacheKey = generateCacheKey(response.config)
      setCachedResponse(cacheKey, response.data)
    }

    // 응답 로깅 (개발 환경에서만)
    if (import.meta.env.DEV) {
      console.log('✅ API Response:', {
        status: response.status,
        url: response.config.url,
        cached: !!response.fromCache,
        data: response.data,
      })
    }

    return response
  },
  async (error: AxiosError<ApiError>) => {
    const config = error.config
    const userStore = useUserStore()

    // pending requests에서 제거
    if (config?.method?.toLowerCase() === 'get') {
      const requestKey = generateRequestKey(config)
      pendingRequests.delete(requestKey)
    }

    // 응답 에러 로깅
    console.error('❌ Response Error:', {
      status: error.response?.status,
      url: config?.url,
      message: error.response?.data?.message || error.message,
      data: error.response?.data,
      retryCount: config?.metadata?.retryCount || 0,
    })

    // 재시도 로직 (401 에러 제외하고 먼저 처리)
    if (config && error.response?.status !== 401) {
      const retryCount = config.metadata?.retryCount || 0
      const shouldRetry = globalRetryConfig.retryCondition ?
        globalRetryConfig.retryCondition(error) :
        (DEFAULT_RETRY_CONFIG.retryCondition?.(error) ?? false)

      if (shouldRetry && retryCount < globalRetryConfig.maxRetries) {
        const delay = calculateRetryDelay(
          retryCount + 1,
          globalRetryConfig.retryDelay,
          globalRetryConfig.exponentialBackoff,
        )

        console.log(`🔄 Retrying request (${retryCount + 1}/${globalRetryConfig.maxRetries}) after ${delay}ms:`, config.url)

        await sleep(delay)

        // 재시도 횟수 증가
        config.metadata.retryCount = retryCount + 1

        return api.request(config)
      }
    }

    // 401 에러 처리 (인증 실패) - 재시도 후에 처리
    if (error.response?.status === 401) {
      // 토큰 갱신 시도
      if (userStore.authState.refreshToken && !config?.url?.includes('/auth/refresh')) {
        try {
          const refreshSuccess = await userStore.refreshToken()
          if (refreshSuccess && config) {
            // 토큰 갱신 성공 시 원래 요청 재시도
            if (config.headers) {
              config.headers.Authorization = `Bearer ${userStore.authState.token}`
            }
            // 재시도 카운트 리셋 (토큰 갱신 후)
            config.metadata = { retryCount: 0 }
            return api.request(config)
          }
        } catch (refreshError) {
          console.error('Token refresh failed:', refreshError)
        }
      }

      // 토큰 갱신 실패 또는 불가능한 경우 로그아웃
      userStore.clearAuth()

      // 로그인 페이지로 리다이렉트 (라우터를 통해)
      if (typeof window !== 'undefined' && window.location.pathname !== '/login') {
        window.location.href = '/login'
      }
    }

    // 403 에러 처리 (권한 없음)
    if (error.response?.status === 403) {
      // 권한 없음 알림
      if (typeof window !== 'undefined') {
        // TODO: 전역 알림 시스템 구현 후 사용
        console.warn('Access denied: Insufficient permissions')
      }
    }

    // 네트워크 오프라인 에러 처리
    if (error.code === 'NETWORK_OFFLINE') {
      console.warn('🔌 Request failed: Network is offline')
      // TODO: 오프라인 상태 UI 표시
    }

    // 5xx 서버 에러 처리
    if (error.response?.status && error.response.status >= 500) {
      // 서버 에러 알림
      if (typeof window !== 'undefined') {
        // TODO: 전역 알림 시스템 구현 후 사용
        console.error('Server error occurred')
      }
    }

    return Promise.reject(error)
  },
)

// API 래퍼 함수들 (고도화된 버전)
export const apiGet = <T = unknown>(
  url: string,
  config?: AxiosRequestConfig & { cache?: Partial<CacheConfig> },
): Promise<AxiosResponse<ApiResponse<T>>> => {
  const requestKey = generateRequestKey({ method: 'GET', url, ...config })

  // 중복 요청 방지
  if (pendingRequests.has(requestKey)) {
    return pendingRequests.get(requestKey) as Promise<AxiosResponse<ApiResponse<T>>>
  }

  const request = api.get(url, config)
  pendingRequests.set(requestKey, request)

  // 요청 완료 후 pending에서 제거
  request.finally(() => {
    pendingRequests.delete(requestKey)
  })

  return request
}

export const apiPost = <T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<AxiosResponse<ApiResponse<T>>> => {
  return api.post(url, data, config)
}

export const apiPut = <T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<AxiosResponse<ApiResponse<T>>> => {
  return api.put(url, data, config)
}

export const apiPatch = <T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<AxiosResponse<ApiResponse<T>>> => {
  return api.patch(url, data, config)
}

export const apiDelete = <T = unknown>(url: string, config?: AxiosRequestConfig): Promise<AxiosResponse<ApiResponse<T>>> => {
  return api.delete(url, config)
}

// 파일 업로드용 API
export const apiUpload = <T = unknown>(
  url: string,
  formData: FormData,
  onUploadProgress?: (progressEvent: { loaded: number; total: number }) => void,
): Promise<AxiosResponse<ApiResponse<T>>> => {
  return api.post(url, formData, {
    headers: {
      'Content-Type': 'multipart/form-data',
    },
    onUploadProgress,
  })
}

// 스트림 다운로드용 API
export const apiDownload = (url: string, filename?: string): Promise<void> => {
  return api.get(url, {
    responseType: 'blob',
  }).then(response => {
    const blob = new Blob([response.data])
    const downloadUrl = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = downloadUrl
    link.download = filename || 'download'
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(downloadUrl)
  })
}

// 설정 함수들
export const configureApiClient = (config: ApiClientConfig): void => {
  if (config.retry) {
    globalRetryConfig = { ...globalRetryConfig, ...config.retry }
  }

  if (config.cache) {
    globalCacheConfig = { ...globalCacheConfig, ...config.cache }
  }

  if (config.timeout) {
    api.defaults.timeout = config.timeout
  }
}

// 캐시 관리 함수들
export const clearApiCache = (): void => {
  requestCache.clear()
  console.log('🗑️ API cache cleared')
}

export const removeFromCache = (url: string, method = 'GET', params?: unknown): void => {
  const cacheKey = generateCacheKey({ method, url, params })
  requestCache.delete(cacheKey)
}

export const getCacheSize = (): number => {
  return requestCache.size
}

export const getCacheStats = (): {
  size: number;
  entries: Array<{ key: string; age: number; maxAge: number }>
} => {
  const now = Date.now()
  const entries = Array.from(requestCache.entries()).map(([key, value]) => ({
    key,
    age: now - value.timestamp,
    maxAge: value.maxAge,
  }))

  return {
    size: requestCache.size,
    entries,
  }
}

// 네트워크 상태 관련
export const getNetworkStatus = (): {
  isOnline: boolean;
  pendingRequests: number;
  cachedResponses: number;
} => {
  return {
    isOnline,
    pendingRequests: pendingRequests.size,
    cachedResponses: requestCache.size,
  }
}

// 모든 대기 중인 요청 취소
export const cancelAllPendingRequests = (): void => {
  pendingRequests.clear()
  console.log('❌ All pending requests cancelled')
}

// 타입 정의 export
export type { RetryConfig, CacheConfig, ApiClientConfig }

export default api