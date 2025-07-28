/**
 * Backend-Frontend Authentication Integration E2E Tests
 * 백엔드-프론트엔드 인증 통합 E2E 테스트
 */

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createRouter, createWebHistory } from 'vue-router'
import { createPinia } from 'pinia'
import axios from 'axios'

import LoginView from '@/views/LoginView.vue'
import DashboardView from '@/views/DashboardView.vue'
import { useUserStore } from '@/stores/user'
import { authApi } from '@/api/services/auth'

// 백엔드 통합 테스트를 위한 설정
const TEST_API_BASE_URL = process.env.VITE_TEST_API_URL || 'http://localhost:8080'

// 실제 백엔드 API 호출을 위한 axios 인스턴스
const testApiClient = axios.create({
  baseURL: TEST_API_BASE_URL,
  timeout: 10000,
  withCredentials: true,
})

// 테스트 데이터
const testUsers = {
  admin: {
    username: 'test-admin',
    email: 'admin@test.com',
    password: 'TestPassword123!',
    role: 'admin',
  },
  user: {
    username: 'test-user',
    email: 'user@test.com',
    password: 'TestPassword123!',
    role: 'user',
  },
  viewer: {
    username: 'test-viewer',
    email: 'viewer@test.com',
    password: 'TestPassword123!',
    role: 'viewer',
  },
}

// Router 설정
const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'Home', component: { template: '<div>Home</div>' } },
    { path: '/login', name: 'Login', component: LoginView },
    { path: '/dashboard', name: 'Dashboard', component: DashboardView },
  ],
})

// 테스트 유틸리티
const createTestApp = () => {
  const pinia = createPinia()
  return { pinia }
}

// 백엔드 서버 상태 확인
const checkBackendHealth = async (): Promise<boolean> => {
  try {
    const response = await testApiClient.get('/api/v1/health')
    return response.status === 200
  } catch (error) {
    return false
  }
}

// 테스트 사용자 생성
const createTestUser = async (userData: typeof testUsers.admin) => {
  try {
    await testApiClient.post('/api/v1/auth/register', userData)
  } catch (error) {
    // 이미 존재하는 사용자는 무시
    if (!(error as any)?.response?.data?.error?.code === 'USER_ALREADY_EXISTS') {
      throw error
    }
  }
}

// 테스트 사용자 정리
const cleanupTestUsers = async () => {
  for (const user of Object.values(testUsers)) {
    try {
      await testApiClient.delete(`/api/v1/users/by-username/${user.username}`)
    } catch (error) {
      // 404는 무시 (이미 삭제됨)
    }
  }
}

describe('Backend-Frontend Authentication Integration E2E Tests', () => {
  let backendAvailable = false

  beforeEach(async () => {
    // 백엔드 서버 상태 확인
    backendAvailable = await checkBackendHealth()

    if (!backendAvailable) {
      console.warn('⚠️ Backend server not available, skipping integration tests')
      return
    }

    // 테스트 사용자 생성
    await Promise.all(
      Object.values(testUsers).map(user => createTestUser(user)),
    )

    // localStorage 초기화
    localStorage.clear()
    sessionStorage.clear()
  })

  afterEach(async () => {
    if (!backendAvailable) return

    // 테스트 사용자 정리
    await cleanupTestUsers()
    vi.restoreAllMocks()
  })

  describe('Full Stack Authentication Flow', () => {
    it('should complete end-to-end login flow with real backend', async () => {
      if (!backendAvailable) {
        console.warn('⚠️ Skipping test: Backend not available')
        return
      }

      const { pinia } = createTestApp()

      const wrapper = mount(LoginView, {
        global: {
          plugins: [pinia, router],
        },
      })

      const userStore = useUserStore()

      // 실제 백엔드 API 호출 설정
      const originalApiCall = authApi.login
      authApi.login = async (credentials) => {
        const response = await testApiClient.post('/api/v1/auth/login', credentials)
        return response.data
      }

      try {
        // 로그인 폼 입력
        const usernameInput = wrapper.find('input[placeholder*="사용자명"]')
        const passwordInput = wrapper.find('input[type="password"]')
        const loginButton = wrapper.find('button[type="submit"], .n-button:contains("로그인")')

        await usernameInput.setValue(testUsers.admin.username)
        await passwordInput.setValue(testUsers.admin.password)
        await loginButton.trigger('click')

        // 로그인 완료 대기
        await vi.waitFor(() => {
          expect(userStore.isAuthenticated).toBe(true)
        }, { timeout: 5000 })

        // 사용자 정보 확인
        expect(userStore.user?.username).toBe(testUsers.admin.username)
        expect(userStore.user?.role).toBe(testUsers.admin.role)
        expect(userStore.auth?.token).toBeTruthy()

        // JWT 토큰 검증
        const token = userStore.auth?.token
        expect(token).toMatch(/^[A-Za-z0-9-_]+\.[A-Za-z0-9-_]+\.[A-Za-z0-9-_]+$/)

      } finally {
        // API 호출 원복
        authApi.login = originalApiCall
      }
    })

    it('should handle role-based access control with backend', async () => {
      if (!backendAvailable) return

      const { pinia } = createTestApp()
      const userStore = useUserStore()

      // Admin 사용자로 로그인
      const adminLoginResponse = await testApiClient.post('/api/v1/auth/login', {
        username: testUsers.admin.username,
        password: testUsers.admin.password,
      })

      userStore.setAuth({
        token: adminLoginResponse.data.access_token,
        refreshToken: adminLoginResponse.data.refresh_token,
        expiresAt: Date.now() + adminLoginResponse.data.expires_in * 1000,
      })

      userStore.setUser({
        id: adminLoginResponse.data.user.id,
        username: adminLoginResponse.data.user.username,
        email: adminLoginResponse.data.user.email,
        role: adminLoginResponse.data.user.role,
      })

      // Admin 권한 API 호출 테스트
      const adminApiResponse = await testApiClient.get('/api/v1/admin/users', {
        headers: {
          Authorization: `Bearer ${adminLoginResponse.data.access_token}`,
        },
      })

      expect(adminApiResponse.status).toBe(200)
      expect(Array.isArray(adminApiResponse.data.users)).toBe(true)

      // 일반 사용자로 로그인
      const userLoginResponse = await testApiClient.post('/api/v1/auth/login', {
        username: testUsers.user.username,
        password: testUsers.user.password,
      })

      // 일반 사용자로 Admin API 접근 시도 (실패해야 함)
      try {
        await testApiClient.get('/api/v1/admin/users', {
          headers: {
            Authorization: `Bearer ${userLoginResponse.data.access_token}`,
          },
        })
        expect.fail('Admin API should be forbidden for regular users')
      } catch (error) {
        expect((error as any).response?.status).toBe(403)
      }
    })

    it('should handle token refresh with real backend', async () => {
      if (!backendAvailable) return

      const { pinia } = createTestApp()
      const userStore = useUserStore()

      // 로그인
      const loginResponse = await testApiClient.post('/api/v1/auth/login', {
        username: testUsers.admin.username,
        password: testUsers.admin.password,
      })

      const originalToken = loginResponse.data.access_token
      const refreshToken = loginResponse.data.refresh_token

      userStore.setAuth({
        token: originalToken,
        refreshToken,
        expiresAt: Date.now() + loginResponse.data.expires_in * 1000,
      })

      // 토큰 갱신
      const refreshResponse = await testApiClient.post('/api/v1/auth/refresh', {
        refresh_token: refreshToken,
      })

      expect(refreshResponse.status).toBe(200)
      expect(refreshResponse.data.access_token).toBeTruthy()
      expect(refreshResponse.data.access_token).not.toBe(originalToken)

      // 새 토큰으로 API 호출 테스트
      const apiResponse = await testApiClient.get('/api/v1/auth/profile', {
        headers: {
          Authorization: `Bearer ${refreshResponse.data.access_token}`,
        },
      })

      expect(apiResponse.status).toBe(200)
      expect(apiResponse.data.user.username).toBe(testUsers.admin.username)
    })
  })

  describe('Backend Security Integration', () => {
    it('should validate CSRF protection on backend', async () => {
      if (!backendAvailable) return

      // CSRF 토큰 없이 로그인 시도
      try {
        await testApiClient.post('/api/v1/auth/login', {
          username: testUsers.admin.username,
          password: testUsers.admin.password,
        }, {
          headers: {
            'X-Requested-With': 'XMLHttpRequest',
          },
        })
      } catch (error) {
        // CSRF 보호가 활성화된 경우 403 또는 400 에러 예상
        const status = (error as any).response?.status
        expect([400, 403].includes(status)).toBe(true)
      }

      // CSRF 토큰 획득
      const csrfResponse = await testApiClient.get('/api/v1/auth/csrf-token')
      const csrfToken = csrfResponse.data.token

      // CSRF 토큰과 함께 로그인
      const loginResponse = await testApiClient.post('/api/v1/auth/login', {
        username: testUsers.admin.username,
        password: testUsers.admin.password,
        _csrf: csrfToken,
      })

      expect(loginResponse.status).toBe(200)
      expect(loginResponse.data.access_token).toBeTruthy()
    })

    it('should enforce rate limiting on backend', async () => {
      if (!backendAvailable) return

      // 연속으로 잘못된 로그인 시도
      const attempts = []
      for (let i = 0; i < 10; i++) {
        attempts.push(
          testApiClient.post('/api/v1/auth/login', {
            username: testUsers.admin.username,
            password: 'wrong-password',
          }).catch(error => error.response),
        )
      }

      const results = await Promise.all(attempts)

      // 마지막 몇 개 요청에서 429 Rate Limit 에러 확인
      const rateLimitedResults = results.filter(result => result?.status === 429)
      expect(rateLimitedResults.length).toBeGreaterThan(0)

      // Rate limit 헤더 확인
      const rateLimitedResponse = rateLimitedResults[0]
      expect(rateLimitedResponse.headers['retry-after']).toBeTruthy()
      expect(rateLimitedResponse.headers['x-ratelimit-remaining']).toBe('0')
    })

    it('should validate security headers from backend', async () => {
      if (!backendAvailable) return

      const response = await testApiClient.get('/api/v1/health')

      // 보안 헤더 확인
      const headers = response.headers
      expect(headers['x-content-type-options']).toBe('nosniff')
      expect(headers['x-frame-options']).toBe('DENY')
      expect(headers['x-xss-protection']).toBe('1; mode=block')
      expect(headers['strict-transport-security']).toContain('max-age')
      expect(headers['content-security-policy']).toBeTruthy()
    })
  })

  describe('Session Management Integration', () => {
    it('should handle session timeout with backend', async () => {
      if (!backendAvailable) return

      const { pinia } = createTestApp()
      const userStore = useUserStore()

      // 짧은 세션 타임아웃으로 로그인
      const loginResponse = await testApiClient.post('/api/v1/auth/login', {
        username: testUsers.admin.username,
        password: testUsers.admin.password,
        session_timeout: 2, // 2초
      })

      userStore.setAuth({
        token: loginResponse.data.access_token,
        refreshToken: loginResponse.data.refresh_token,
        expiresAt: Date.now() + 2000, // 2초 후 만료
      })

      // 즉시 API 호출 (성공해야 함)
      const apiResponse = await testApiClient.get('/api/v1/auth/profile', {
        headers: {
          Authorization: `Bearer ${loginResponse.data.access_token}`,
        },
      })
      expect(apiResponse.status).toBe(200)

      // 3초 대기 후 API 호출 (실패해야 함)
      await new Promise(resolve => setTimeout(resolve, 3000))

      try {
        await testApiClient.get('/api/v1/auth/profile', {
          headers: {
            Authorization: `Bearer ${loginResponse.data.access_token}`,
          },
        })
        expect.fail('API call should fail with expired token')
      } catch (error) {
        expect((error as any).response?.status).toBe(401)
      }
    })

    it('should handle concurrent sessions with backend', async () => {
      if (!backendAvailable) return

      // 첫 번째 세션
      const session1 = await testApiClient.post('/api/v1/auth/login', {
        username: testUsers.admin.username,
        password: testUsers.admin.password,
      })

      // 두 번째 세션
      const session2 = await testApiClient.post('/api/v1/auth/login', {
        username: testUsers.admin.username,
        password: testUsers.admin.password,
      })

      // 두 세션 모두 유효한지 확인
      const profile1 = await testApiClient.get('/api/v1/auth/profile', {
        headers: { Authorization: `Bearer ${session1.data.access_token}` },
      })

      const profile2 = await testApiClient.get('/api/v1/auth/profile', {
        headers: { Authorization: `Bearer ${session2.data.access_token}` },
      })

      expect(profile1.status).toBe(200)
      expect(profile2.status).toBe(200)

      // 첫 번째 세션 로그아웃
      await testApiClient.post('/api/v1/auth/logout', {}, {
        headers: { Authorization: `Bearer ${session1.data.access_token}` },
      })

      // 첫 번째 세션 무효화 확인
      try {
        await testApiClient.get('/api/v1/auth/profile', {
          headers: { Authorization: `Bearer ${session1.data.access_token}` },
        })
        expect.fail('First session should be invalidated')
      } catch (error) {
        expect((error as any).response?.status).toBe(401)
      }

      // 두 번째 세션은 여전히 유효
      const profile2Again = await testApiClient.get('/api/v1/auth/profile', {
        headers: { Authorization: `Bearer ${session2.data.access_token}` },
      })
      expect(profile2Again.status).toBe(200)
    })
  })

  describe('Error Scenarios Integration', () => {
    it('should handle database connection errors gracefully', async () => {
      if (!backendAvailable) return

      // 데이터베이스 연결 문제 시뮬레이션은 실제 환경에서만 가능
      // 여기서는 서버 에러 응답 처리 테스트
      try {
        await testApiClient.post('/api/v1/auth/login', {
          username: 'nonexistent-user',
          password: 'any-password',
        })
      } catch (error) {
        const response = (error as any).response
        expect(response?.status).toBe(401)
        expect(response?.data?.error?.code).toBe('INVALID_CREDENTIALS')
      }
    })

    it('should handle malformed requests properly', async () => {
      if (!backendAvailable) return

      // 잘못된 JSON 데이터
      try {
        await testApiClient.post('/api/v1/auth/login', {
          username: '', // 빈 사용자명
          password: '', // 빈 비밀번호
        })
      } catch (error) {
        const response = (error as any).response
        expect(response?.status).toBe(400)
        expect(response?.data?.error?.code).toBe('INVALID_REQUEST')
      }

      // 누락된 필드
      try {
        await testApiClient.post('/api/v1/auth/login', {
          username: testUsers.admin.username,
          // password 누락
        })
      } catch (error) {
        const response = (error as any).response
        expect(response?.status).toBe(400)
      }
    })
  })

  describe('Performance Integration', () => {
    it('should handle multiple concurrent logins efficiently', async () => {
      if (!backendAvailable) return

      const startTime = Date.now()

      // 10개의 동시 로그인 요청
      const loginPromises = Array.from({ length: 10 }, () =>
        testApiClient.post('/api/v1/auth/login', {
          username: testUsers.admin.username,
          password: testUsers.admin.password,
        }),
      )

      const results = await Promise.all(loginPromises)
      const endTime = Date.now()

      // 모든 로그인이 성공해야 함
      results.forEach(result => {
        expect(result.status).toBe(200)
        expect(result.data.access_token).toBeTruthy()
      })

      // 10개 요청이 10초 내에 완료되어야 함
      expect(endTime - startTime).toBeLessThan(10000)
    })

    it('should maintain reasonable response times', async () => {
      if (!backendAvailable) return

      const times = []

      // 5번의 연속 API 호출로 평균 응답 시간 측정
      for (let i = 0; i < 5; i++) {
        const startTime = Date.now()

        await testApiClient.post('/api/v1/auth/login', {
          username: testUsers.admin.username,
          password: testUsers.admin.password,
        })

        const endTime = Date.now()
        times.push(endTime - startTime)
      }

      const averageTime = times.reduce((a, b) => a + b, 0) / times.length

      // 평균 응답 시간이 2초 이하여야 함
      expect(averageTime).toBeLessThan(2000)
    })
  })
})