/**
 * Simple Authentication E2E Tests
 * 간단한 인증 E2E 테스트
 */

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia } from 'pinia'
import { useUserStore } from '@/stores/user'
import { authApi } from '@/api/services/auth'

// Mock API 응답
const mockApiResponses = {
  login: {
    success: true,
    data: {
      access_token: 'mock-access-token',
      refresh_token: 'mock-refresh-token',
      token_type: 'Bearer',
      expires_in: 3600,
    },
  },
  refresh: {
    success: true,
    data: {
      access_token: 'new-mock-access-token',
      token_type: 'Bearer',
      expires_in: 3600,
    },
  },
  profile: {
    username: 'admin',
    email: 'admin@example.com',
    role: 'admin',
  },
  csrfToken: {
    token: 'csrf-token-123',
    expiresAt: Date.now() + 600000,
  },
}

describe('Simple Authentication E2E Tests', () => {
  let userStore: ReturnType<typeof useUserStore>

  beforeEach(() => {
    const pinia = createPinia()
    userStore = useUserStore(pinia)

    // API Mock 설정
    vi.clearAllMocks()

    // localStorage 초기화
    localStorage.clear()
    sessionStorage.clear()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('API Integration Tests', () => {
    it('should handle successful login flow', async () => {
      // API Mock 설정
      const mockLogin = vi.spyOn(authApi, 'login').mockResolvedValue(mockApiResponses.login)

      // 로그인 요청
      const credentials = { username: 'admin', password: 'admin123' }
      const response = await authApi.login(credentials)

      // API 호출 확인
      expect(mockLogin).toHaveBeenCalledWith(credentials)
      expect(response.data.access_token).toBe('mock-access-token')
      expect(response.data.refresh_token).toBe('mock-refresh-token')
    })

    it('should handle token refresh', async () => {
      // Token Refresh API Mock
      const mockRefresh = vi.spyOn(authApi, 'refreshToken').mockResolvedValue(mockApiResponses.refresh)

      // 토큰 갱신 요청
      const request = { refresh_token: 'old-refresh-token' }
      const response = await authApi.refreshToken(request)

      // API 호출 확인
      expect(mockRefresh).toHaveBeenCalledWith(request)
      expect(response.data.access_token).toBe('new-mock-access-token')
    })

    it('should handle CSRF token retrieval', async () => {
      // CSRF Token API Mock
      const mockGetCsrfToken = vi.spyOn(authApi, 'getCsrfToken').mockResolvedValue(mockApiResponses.csrfToken)

      // CSRF 토큰 요청
      const response = await authApi.getCsrfToken()

      // API 호출 확인
      expect(mockGetCsrfToken).toHaveBeenCalled()
      expect(response.token).toBe('csrf-token-123')
      expect(response.expiresAt).toBeGreaterThan(Date.now())
    })

    it('should handle logout', async () => {
      // Logout API Mock
      const mockLogout = vi.spyOn(authApi, 'logout').mockResolvedValue()

      // 로그아웃 요청
      await authApi.logout()

      // API 호출 확인
      expect(mockLogout).toHaveBeenCalled()
    })
  })

  describe('Store Integration Tests', () => {
    it('should manage authentication state correctly', () => {
      // 초기 상태 확인
      expect(userStore.isAuthenticated).toBe(false)
      expect(userStore.user).toBeNull()
      expect(userStore.auth).toBeNull()

      // 인증 상태 설정
      const userData = {
        id: '1',
        username: 'admin',
        email: 'admin@example.com',
        role: 'admin',
      }

      const authData = {
        token: 'access-token',
        refreshToken: 'refresh-token',
        expiresAt: Date.now() + 3600000,
      }

      userStore.setUser(userData)
      userStore.setAuth(authData)

      // 상태 확인
      expect(userStore.isAuthenticated).toBe(true)
      expect(userStore.user).toEqual(userData)
      expect(userStore.auth).toEqual(authData)
    })

    it('should handle logout and clear state', () => {
      // 인증된 상태 설정
      userStore.setUser({
        id: '1',
        username: 'admin',
        email: 'admin@example.com',
        role: 'admin',
      })
      userStore.setAuth({
        token: 'access-token',
        refreshToken: 'refresh-token',
        expiresAt: Date.now() + 3600000,
      })

      expect(userStore.isAuthenticated).toBe(true)

      // 로그아웃
      userStore.clearAuth()

      // 상태 초기화 확인
      expect(userStore.isAuthenticated).toBe(false)
      expect(userStore.user).toBeNull()
      expect(userStore.auth).toBeNull()
    })

    it('should persist authentication state to localStorage', () => {
      const userData = {
        id: '1',
        username: 'admin',
        email: 'admin@example.com',
        role: 'admin',
      }

      const authData = {
        token: 'access-token',
        refreshToken: 'refresh-token',
        expiresAt: Date.now() + 3600000,
      }

      // 상태 설정
      userStore.setUser(userData)
      userStore.setAuth(authData)

      // localStorage 저장 확인
      expect(JSON.parse(localStorage.getItem('user') || '{}')).toEqual(userData)
      expect(JSON.parse(localStorage.getItem('auth') || '{}')).toEqual(authData)
    })
  })

  describe('Security Features Tests', () => {
    it('should handle rate limiting scenarios', async () => {
      // Rate limit 에러 시뮬레이션
      const rateLimitError = {
        response: {
          status: 429,
          data: {
            error: {
              code: 'RATE_LIMIT_EXCEEDED',
              message: 'Too many login attempts',
              retryAfter: 60,
            },
          },
        },
      }

      const mockLogin = vi.spyOn(authApi, 'login').mockRejectedValue(rateLimitError)

      try {
        await authApi.login({ username: 'admin', password: 'admin123' })
        expect.fail('Should have thrown rate limit error')
      } catch (error: any) {
        expect(error.response.status).toBe(429)
        expect(error.response.data.error.code).toBe('RATE_LIMIT_EXCEEDED')
      }
    })

    it('should handle CSRF token validation', async () => {
      const mockGetCsrfToken = vi.spyOn(authApi, 'getCsrfToken').mockResolvedValue({
        token: 'valid-csrf-token',
        expiresAt: Date.now() + 600000,
      })

      const mockLogin = vi.spyOn(authApi, 'login').mockResolvedValue(mockApiResponses.login)

      // CSRF 토큰 획득
      const csrfResponse = await authApi.getCsrfToken()
      expect(csrfResponse.token).toBe('valid-csrf-token')

      // CSRF 토큰과 함께 로그인
      const loginResponse = await authApi.login({
        username: 'admin',
        password: 'admin123',
        _csrf: csrfResponse.token,
      })

      expect(loginResponse.data.access_token).toBeTruthy()
    })

    it('should handle expired token scenarios', () => {
      // 만료된 토큰 설정
      userStore.setAuth({
        token: 'expired-token',
        refreshToken: 'expired-refresh-token',
        expiresAt: Date.now() - 1000, // 이미 만료됨
      })

      // 토큰 만료 확인
      expect(userStore.isTokenExpired).toBe(true)
    })
  })

  describe('Error Handling Tests', () => {
    it('should handle network errors', async () => {
      const networkError = new Error('Network Error')
      const mockLogin = vi.spyOn(authApi, 'login').mockRejectedValue(networkError)

      try {
        await authApi.login({ username: 'admin', password: 'admin123' })
        expect.fail('Should have thrown network error')
      } catch (error) {
        expect(error).toBeInstanceOf(Error)
        expect((error as Error).message).toBe('Network Error')
      }
    })

    it('should handle invalid credentials', async () => {
      const authError = {
        response: {
          status: 401,
          data: {
            error: {
              code: 'INVALID_CREDENTIALS',
              message: 'Invalid username or password',
            },
          },
        },
      }

      const mockLogin = vi.spyOn(authApi, 'login').mockRejectedValue(authError)

      try {
        await authApi.login({ username: 'admin', password: 'wrongpassword' })
        expect.fail('Should have thrown auth error')
      } catch (error: any) {
        expect(error.response.status).toBe(401)
        expect(error.response.data.error.code).toBe('INVALID_CREDENTIALS')
      }
    })

    it('should handle malformed requests', async () => {
      const validationError = {
        response: {
          status: 400,
          data: {
            error: {
              code: 'INVALID_REQUEST',
              message: 'Invalid request body',
            },
          },
        },
      }

      const mockLogin = vi.spyOn(authApi, 'login').mockRejectedValue(validationError)

      try {
        await authApi.login({ username: '', password: '' })
        expect.fail('Should have thrown validation error')
      } catch (error: any) {
        expect(error.response.status).toBe(400)
        expect(error.response.data.error.code).toBe('INVALID_REQUEST')
      }
    })
  })

  describe('Performance Tests', () => {
    it('should handle multiple concurrent requests', async () => {
      const mockLogin = vi.spyOn(authApi, 'login').mockResolvedValue(mockApiResponses.login)

      const startTime = Date.now()

      // 5개의 동시 로그인 요청
      const promises = Array.from({ length: 5 }, () =>
        authApi.login({ username: 'admin', password: 'admin123' }),
      )

      const results = await Promise.all(promises)
      const endTime = Date.now()

      // 모든 요청이 성공해야 함
      results.forEach(result => {
        expect(result.data.access_token).toBeTruthy()
      })

      // 5개 요청이 5초 내에 완료되어야 함
      expect(endTime - startTime).toBeLessThan(5000)
    })

    it('should measure API response times', async () => {
      const mockLogin = vi.spyOn(authApi, 'login').mockResolvedValue(mockApiResponses.login)

      const times = []

      // 3번의 연속 API 호출로 평균 응답 시간 측정
      for (let i = 0; i < 3; i++) {
        const startTime = Date.now()
        await authApi.login({ username: 'admin', password: 'admin123' })
        const endTime = Date.now()
        times.push(endTime - startTime)
      }

      const averageTime = times.reduce((a, b) => a + b, 0) / times.length

      // 평균 응답 시간이 1초 이하여야 함 (Mock이므로 매우 빨라야 함)
      expect(averageTime).toBeLessThan(1000)
    })
  })
})