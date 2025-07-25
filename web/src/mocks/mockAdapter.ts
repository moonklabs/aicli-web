import type { AxiosAdapter, AxiosRequestConfig, AxiosResponse } from 'axios'
import { mockApiMatcher } from './mockApi'

// Mock Axios 어댑터 생성
export function createMockAdapter(defaultAdapter: AxiosAdapter): AxiosAdapter {
  return async (config: AxiosRequestConfig): Promise<AxiosResponse> => {
    // 개발 환경이고 Mock API가 활성화된 경우
    if (import.meta.env.DEV && mockApiMatcher.isActive()) {
      const mockResponse = await mockApiMatcher.matchRequest(config)

      if (mockResponse) {
        console.log(`🎭 Mock API Response: ${config.method?.toUpperCase()} ${config.url}`, mockResponse)
        return mockResponse
      }
    }

    // Mock이 매치되지 않으면 기본 어댑터 사용
    return defaultAdapter(config)
  }
}

// Mock API 설정 함수
export function setupMockApi(axios: any): void {
  if (import.meta.env.DEV) {
    const originalAdapter = axios.defaults.adapter
    axios.defaults.adapter = createMockAdapter(originalAdapter)

    console.log('🎭 Mock API adapter installed')
  }
}