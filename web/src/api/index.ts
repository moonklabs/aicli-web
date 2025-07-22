import axios, { type AxiosError, type AxiosInstance, type AxiosRequestConfig, type AxiosResponse } from 'axios'
import { useUserStore } from '@/stores/user'
import type { ApiError, ApiResponse } from '@/types/api'

// Axios 인스턴스 생성
const api: AxiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api',
  timeout: 30000, // 30초 타임아웃
  headers: {
    'Content-Type': 'application/json',
  },
})

// 요청 인터셉터 설정
api.interceptors.request.use(
  (config: AxiosRequestConfig): any => {
    // 인증 토큰 추가
    const userStore = useUserStore()
    if (userStore.authState.token) {
      if (config.headers) {
        config.headers.Authorization = `Bearer ${userStore.authState.token}`
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
    // 응답 로깅 (개발 환경에서만)
    if (import.meta.env.DEV) {
      console.log('✅ API Response:', {
        status: response.status,
        url: response.config.url,
        data: response.data,
      })
    }

    return response
  },
  async (error: AxiosError<ApiError>) => {
    const userStore = useUserStore()

    // 응답 에러 로깅
    console.error('❌ Response Error:', {
      status: error.response?.status,
      url: error.config?.url,
      message: error.response?.data?.message || error.message,
      data: error.response?.data,
    })

    // 401 에러 처리 (인증 실패)
    if (error.response?.status === 401) {
      // 토큰 갱신 시도
      if (userStore.authState.refreshToken && !error.config?.url?.includes('/auth/refresh')) {
        try {
          const refreshSuccess = await userStore.refreshToken()
          if (refreshSuccess && error.config) {
            // 토큰 갱신 성공 시 원래 요청 재시도
            if (error.config.headers) {
              error.config.headers.Authorization = `Bearer ${userStore.authState.token}`
            }
            return api.request(error.config)
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

// API 래퍼 함수들
export const apiGet = <T = any>(url: string, config?: AxiosRequestConfig): Promise<AxiosResponse<ApiResponse<T>>> => {
  return api.get(url, config)
}

export const apiPost = <T = any>(url: string, data?: any, config?: AxiosRequestConfig): Promise<AxiosResponse<ApiResponse<T>>> => {
  return api.post(url, data, config)
}

export const apiPut = <T = any>(url: string, data?: any, config?: AxiosRequestConfig): Promise<AxiosResponse<ApiResponse<T>>> => {
  return api.put(url, data, config)
}

export const apiPatch = <T = any>(url: string, data?: any, config?: AxiosRequestConfig): Promise<AxiosResponse<ApiResponse<T>>> => {
  return api.patch(url, data, config)
}

export const apiDelete = <T = any>(url: string, config?: AxiosRequestConfig): Promise<AxiosResponse<ApiResponse<T>>> => {
  return api.delete(url, config)
}

// 파일 업로드용 API
export const apiUpload = <T = any>(
  url: string,
  formData: FormData,
  onUploadProgress?: (progressEvent: any) => void,
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

export default api