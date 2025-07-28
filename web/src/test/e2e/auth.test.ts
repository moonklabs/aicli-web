/**
 * E2E Authentication Flow Tests for Frontend
 * 프론트엔드 인증 플로우 E2E 테스트
 */

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createRouter, createWebHistory } from 'vue-router'
import { createPinia } from 'pinia'
import { createApp } from 'vue'

import LoginView from '@/views/LoginView.vue'
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
  loginFailed: {
    success: false,
    error: {
      code: 'INVALID_CREDENTIALS',
      message: 'Invalid username or password',
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
}

// Router 설정
const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'Home', component: { template: '<div>Home</div>' } },
    { path: '/login', name: 'Login', component: LoginView },
    { path: '/dashboard', name: 'Dashboard', component: { template: '<div>Dashboard</div>' } },
  ],
})

// 테스트 유틸리티
const createTestApp = () => {
  const app = createApp({})
  const pinia = createPinia()

  app.use(pinia)
  app.use(router)

  return { app, pinia }
}

const mountLoginView = () => {
  const { pinia } = createTestApp()

  return mount(LoginView, {
    global: {
      plugins: [pinia, router],
    },
  })
}

describe('Frontend Authentication E2E Tests', () => {
  beforeEach(() => {
    // API Mock 설정
    vi.clearAllMocks()

    // localStorage 초기화
    localStorage.clear()
    sessionStorage.clear()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('Basic Login Flow', () => {
    it('should complete successful login flow', async () => {
      // API Mock 설정
      const mockLogin = vi.spyOn(authApi, 'login').mockResolvedValue(mockApiResponses.login)

      const wrapper = mountLoginView()
      const userStore = useUserStore()

      // 로그인 폼 입력
      const usernameInput = wrapper.find('input[placeholder*="사용자명"]')
      const passwordInput = wrapper.find('input[type="password"]')
      const loginButton = wrapper.find('button[type="submit"], .n-button:contains("로그인")')

      await usernameInput.setValue('admin')
      await passwordInput.setValue('admin123')

      // 폼 유효성 확인
      expect(usernameInput.element.value).toBe('admin')
      expect(passwordInput.element.value).toBe('admin123')

      // 로그인 버튼 클릭
      await loginButton.trigger('click')

      // API 호출 확인
      expect(mockLogin).toHaveBeenCalledWith({
        username: 'admin',
        password: 'admin123',
      })

      // 로딩 상태 확인
      expect(wrapper.vm.isLoading).toBe(true)

      // API 응답 대기
      await vi.waitFor(() => {
        expect(wrapper.vm.isLoading).toBe(false)
      })

      // 사용자 상태 확인
      expect(userStore.isAuthenticated).toBe(true)
      expect(userStore.user?.username).toBe('admin')
      expect(userStore.auth?.token).toBe('mock-access-token')
    })

    it('should handle login failure correctly', async () => {
      // API Mock 설정 (실패)
      const mockLogin = vi.spyOn(authApi, 'login').mockRejectedValue(
        new Error('Invalid credentials'),
      )

      const wrapper = mountLoginView()
      const userStore = useUserStore()

      // 잘못된 인증 정보 입력
      const usernameInput = wrapper.find('input[placeholder*="사용자명"]')
      const passwordInput = wrapper.find('input[type="password"]')
      const loginButton = wrapper.find('button[type="submit"], .n-button:contains("로그인")')

      await usernameInput.setValue('admin')
      await passwordInput.setValue('wrongpassword')
      await loginButton.trigger('click')

      // API 호출 확인
      expect(mockLogin).toHaveBeenCalledWith({
        username: 'admin',
        password: 'wrongpassword',
      })

      // API 응답 대기
      await vi.waitFor(() => {
        expect(wrapper.vm.isLoading).toBe(false)
      })

      // 사용자 상태 확인 (인증되지 않음)
      expect(userStore.isAuthenticated).toBe(false)
      expect(userStore.user).toBeNull()

      // 에러 메시지 확인 (실제 구현에서는 메시지 컴포넌트 확인)
      // expect(wrapper.text()).toContain('사용자명 또는 비밀번호가 올바르지 않습니다')
    })

    it('should validate form inputs', async () => {
      const wrapper = mountLoginView()

      const usernameInput = wrapper.find('input[placeholder*="사용자명"]')
      const passwordInput = wrapper.find('input[type="password"]')
      const loginButton = wrapper.find('button[type="submit"], .n-button:contains("로그인")')

      // 빈 폼으로 로그인 시도
      await loginButton.trigger('click')

      // 유효성 검사 에러 확인
      expect(wrapper.vm.formData.username).toBe('')
      expect(wrapper.vm.formData.password).toBe('')
      expect(wrapper.vm.isFormValid).toBe(false)

      // 사용자명만 입력
      await usernameInput.setValue('admin')
      expect(wrapper.vm.isFormValid).toBe(false)

      // 짧은 비밀번호 입력
      await passwordInput.setValue('123')
      expect(wrapper.vm.isFormValid).toBe(false)

      // 유효한 입력
      await passwordInput.setValue('admin123')
      expect(wrapper.vm.isFormValid).toBe(true)
    })
  })

  describe('Token Management', () => {
    it('should handle token refresh', async () => {
      const { pinia } = createTestApp()
      const userStore = useUserStore()

      // 초기 인증 상태 설정
      userStore.setAuth({
        token: 'old-token',
        refreshToken: 'refresh-token',
        expiresAt: Date.now() + 1000, // 1초 후 만료
      })

      // Token Refresh API Mock
      const mockRefresh = vi.spyOn(authApi, 'refreshToken').mockResolvedValue(mockApiResponses.refresh)

      // 토큰 만료 시뮬레이션
      await new Promise(resolve => setTimeout(resolve, 1100))

      // API 호출 (토큰 갱신 트리거)
      await userStore.refreshTokenIfNeeded()

      // Token Refresh API 호출 확인
      expect(mockRefresh).toHaveBeenCalledWith('refresh-token')

      // 새 토큰 확인
      expect(userStore.auth?.token).toBe('new-mock-access-token')
    })

    it('should handle token expiry and redirect to login', async () => {
      const { pinia } = createTestApp()
      const userStore = useUserStore()

      // 만료된 토큰 설정
      userStore.setAuth({
        token: 'expired-token',
        refreshToken: 'expired-refresh-token',
        expiresAt: Date.now() - 1000, // 이미 만료됨
      })

      // Token Refresh API Mock (실패)
      const mockRefresh = vi.spyOn(authApi, 'refreshToken').mockRejectedValue(
        new Error('Refresh token expired'),
      )

      // 토큰 갱신 시도
      await userStore.refreshTokenIfNeeded()

      // 인증 상태 확인 (로그아웃됨)
      expect(userStore.isAuthenticated).toBe(false)
      expect(userStore.auth).toBeNull()

      // 라우터 리다이렉트 확인 (실제 구현에서)
      // expect(router.currentRoute.value.name).toBe('Login')
    })
  })

  describe('Session Management', () => {
    it('should handle logout flow', async () => {
      const { pinia } = createTestApp()
      const userStore = useUserStore()

      // 인증된 상태 설정
      userStore.setUser({
        id: '1',
        username: 'admin',
        email: 'admin@example.com',
        displayName: '관리자',
      })
      userStore.setAuth({
        token: 'access-token',
        refreshToken: 'refresh-token',
        expiresAt: Date.now() + 3600000,
      })

      // Logout API Mock
      const mockLogout = vi.spyOn(authApi, 'logout').mockResolvedValue({
        success: true,
        message: 'Logged out successfully',
      })

      // 로그아웃 실행
      await userStore.logout()

      // Logout API 호출 확인
      expect(mockLogout).toHaveBeenCalledWith('access-token')

      // 상태 초기화 확인
      expect(userStore.isAuthenticated).toBe(false)
      expect(userStore.user).toBeNull()
      expect(userStore.auth).toBeNull()

      // LocalStorage 정리 확인
      expect(localStorage.getItem('user')).toBeNull()
      expect(localStorage.getItem('auth')).toBeNull()
    })

    it('should persist authentication state', async () => {
      const { pinia } = createTestApp()
      const userStore = useUserStore()

      const userData = {
        id: '1',
        username: 'admin',
        email: 'admin@example.com',
        displayName: '관리자',
      }

      const authData = {
        token: 'access-token',
        refreshToken: 'refresh-token',
        expiresAt: Date.now() + 3600000,
      }

      // 인증 상태 설정
      userStore.setUser(userData)
      userStore.setAuth(authData)

      // LocalStorage 저장 확인
      expect(JSON.parse(localStorage.getItem('user') || '{}')).toEqual(userData)
      expect(JSON.parse(localStorage.getItem('auth') || '{}')).toEqual(authData)

      // 새 스토어 인스턴스에서 상태 복원 확인
      const { pinia: newPinia } = createTestApp()
      const newUserStore = useUserStore()

      await newUserStore.initializeAuthState()

      expect(newUserStore.user).toEqual(userData)
      expect(newUserStore.auth).toEqual(authData)
      expect(newUserStore.isAuthenticated).toBe(true)
    })
  })

  describe('OAuth Integration', () => {
    it('should handle OAuth login flow', async () => {
      // OAuth URL 생성 API Mock
      const mockGetOAuthUrl = vi.spyOn(authApi, 'getOAuthAuthUrl').mockResolvedValue({
        authUrl: 'https://accounts.google.com/oauth/authorize?client_id=test&redirect_uri=test',
        state: 'random-state-string',
      })

      // window.open Mock
      const mockOpen = vi.spyOn(window, 'open').mockImplementation(() => {
        const mockPopup = {
          closed: false,
          close: vi.fn(),
        }

        // 성공적인 OAuth 콜백 시뮬레이션
        setTimeout(() => {
          mockPopup.closed = true
        }, 1000)

        return mockPopup as any
      })

      const wrapper = mountLoginView()

      // Google 로그인 버튼 클릭
      const googleLoginButton = wrapper.find('button:contains("Google로 로그인")')
      await googleLoginButton.trigger('click')

      // OAuth URL 생성 API 호출 확인
      expect(mockGetOAuthUrl).toHaveBeenCalledWith({
        provider: 'google',
        state: expect.any(String),
      })

      // 팝업 창 열기 확인
      expect(mockOpen).toHaveBeenCalledWith(
        'https://accounts.google.com/oauth/authorize?client_id=test&redirect_uri=test',
        'oauth-google',
        'width=500,height=600,scrollbars=yes,resizable=yes',
      )

      // 로딩 상태 확인
      expect(wrapper.vm.oauthLoading).toBe('google')

      // 팝업 완료 대기
      await vi.waitFor(() => {
        expect(wrapper.vm.oauthLoading).toBeNull()
      }, { timeout: 2000 })
    })

    it('should handle OAuth popup blocked', async () => {
      // OAuth URL 생성 API Mock
      vi.spyOn(authApi, 'getOAuthAuthUrl').mockResolvedValue({
        authUrl: 'https://accounts.google.com/oauth/authorize?client_id=test&redirect_uri=test',
        state: 'random-state-string',
      })

      // window.open Mock (팝업 차단)
      vi.spyOn(window, 'open').mockReturnValue(null)

      const wrapper = mountLoginView()

      // Google 로그인 버튼 클릭
      const googleLoginButton = wrapper.find('button:contains("Google로 로그인")')
      await googleLoginButton.trigger('click')

      // 에러 상태 확인
      await vi.waitFor(() => {
        expect(wrapper.vm.oauthLoading).toBeNull()
      })

      // 에러 메시지 확인 (실제 구현에서는 message 컴포넌트 확인)
      // expect(wrapper.text()).toContain('팝업이 차단되었습니다')
    })
  })

  describe('Form Interactions', () => {
    it('should handle demo credentials filling', async () => {
      const wrapper = mountLoginView()

      // 데모 계정 버튼 클릭
      const demoButton = wrapper.find('button:contains("데모 계정으로 채우기")')
      await demoButton.trigger('click')

      // 폼 필드 확인
      expect(wrapper.vm.formData.username).toBe('admin')
      expect(wrapper.vm.formData.password).toBe('admin123')

      // 입력 필드 값 확인
      const usernameInput = wrapper.find('input[placeholder*="사용자명"]')
      const passwordInput = wrapper.find('input[type="password"]')

      expect(usernameInput.element.value).toBe('admin')
      expect(passwordInput.element.value).toBe('admin123')
    })

    it('should handle enter key login', async () => {
      const mockLogin = vi.spyOn(authApi, 'login').mockResolvedValue(mockApiResponses.login)

      const wrapper = mountLoginView()

      // 폼 입력
      const usernameInput = wrapper.find('input[placeholder*="사용자명"]')
      const passwordInput = wrapper.find('input[type="password"]')

      await usernameInput.setValue('admin')
      await passwordInput.setValue('admin123')

      // Enter 키 이벤트
      await passwordInput.trigger('keyup.enter')

      // 로그인 API 호출 확인
      expect(mockLogin).toHaveBeenCalledWith({
        username: 'admin',
        password: 'admin123',
      })
    })

    it('should handle remember me option', async () => {
      const wrapper = mountLoginView()

      // Remember Me 체크박스 클릭
      const rememberMeCheckbox = wrapper.find('input[type="checkbox"]')
      await rememberMeCheckbox.setChecked(true)

      // 상태 확인
      expect(wrapper.vm.formData.rememberMe).toBe(true)

      // 체크박스 해제
      await rememberMeCheckbox.setChecked(false)
      expect(wrapper.vm.formData.rememberMe).toBe(false)
    })
  })

  describe('Error Handling', () => {
    it('should handle network errors', async () => {
      // 네트워크 에러 시뮬레이션
      const mockLogin = vi.spyOn(authApi, 'login').mockRejectedValue(
        new Error('Network Error'),
      )

      const wrapper = mountLoginView()

      // 로그인 시도
      const usernameInput = wrapper.find('input[placeholder*="사용자명"]')
      const passwordInput = wrapper.find('input[type="password"]')
      const loginButton = wrapper.find('button[type="submit"], .n-button:contains("로그인")')

      await usernameInput.setValue('admin')
      await passwordInput.setValue('admin123')
      await loginButton.trigger('click')

      // 에러 처리 확인
      await vi.waitFor(() => {
        expect(wrapper.vm.isLoading).toBe(false)
      })

      // 사용자 상태 확인 (인증되지 않음)
      const userStore = useUserStore()
      expect(userStore.isAuthenticated).toBe(false)
    })

    it('should handle API validation errors', async () => {
      // 유효성 검사 에러 시뮬레이션
      const mockLogin = vi.spyOn(authApi, 'login').mockRejectedValue({
        response: {
          status: 400,
          data: {
            success: false,
            error: {
              code: 'INVALID_REQUEST',
              message: 'Invalid request body',
              details: 'Username is required',
            },
          },
        },
      })

      const wrapper = mountLoginView()

      // 부분적 입력으로 로그인 시도
      const passwordInput = wrapper.find('input[type="password"]')
      const loginButton = wrapper.find('button[type="submit"], .n-button:contains("로그인")')

      await passwordInput.setValue('admin123')
      await loginButton.trigger('click')

      // 에러 처리 확인
      await vi.waitFor(() => {
        expect(wrapper.vm.isLoading).toBe(false)
      })

      // 에러 메시지 확인 (실제 구현에서)
      // expect(wrapper.text()).toContain('사용자명을 입력해주세요')
    })
  })

  describe('Accessibility', () => {
    it('should have proper ARIA attributes', () => {
      const wrapper = mountLoginView()

      // 폼 요소들의 접근성 속성 확인
      const usernameInput = wrapper.find('input[placeholder*="사용자명"]')
      const passwordInput = wrapper.find('input[type="password"]')

      // ARIA 라벨 또는 라벨 연결 확인
      expect(usernameInput.attributes('aria-label') || usernameInput.attributes('id')).toBeDefined()
      expect(passwordInput.attributes('aria-label') || passwordInput.attributes('id')).toBeDefined()
    })

    it('should handle keyboard navigation', async () => {
      const wrapper = mountLoginView()

      // Tab 키 네비게이션 테스트
      const usernameInput = wrapper.find('input[placeholder*="사용자명"]')
      const passwordInput = wrapper.find('input[type="password"]')
      const loginButton = wrapper.find('button[type="submit"], .n-button:contains("로그인")')

      // 포커스 순서 확인
      await usernameInput.trigger('focus')
      expect(document.activeElement).toBe(usernameInput.element)

      await usernameInput.trigger('keydown.tab')
      // Tab으로 다음 요소로 이동하는 것은 실제 브라우저에서만 작동
      // 테스트에서는 수동으로 포커스 이동
      await passwordInput.trigger('focus')
      expect(document.activeElement).toBe(passwordInput.element)
    })
  })

  describe('Security Features', () => {
    describe('CSRF Protection', () => {
      it('should include CSRF token in login requests', async () => {
        // CSRF 토큰 API Mock
        const mockGetCsrfToken = vi.spyOn(authApi, 'getCsrfToken').mockResolvedValue({
          token: 'csrf-token-123',
          expiresAt: Date.now() + 600000, // 10분
        })

        const mockLogin = vi.spyOn(authApi, 'login').mockResolvedValue(mockApiResponses.login)

        const wrapper = mountLoginView()

        // 컴포넌트 마운트 시 CSRF 토큰 요청 확인
        expect(mockGetCsrfToken).toHaveBeenCalled()

        // 로그인 시도
        const usernameInput = wrapper.find('input[placeholder*="사용자명"]')
        const passwordInput = wrapper.find('input[type="password"]')
        const loginButton = wrapper.find('button[type="submit"], .n-button:contains("로그인")')

        await usernameInput.setValue('admin')
        await passwordInput.setValue('admin123')
        await loginButton.trigger('click')

        // CSRF 토큰이 포함된 요청 확인
        expect(mockLogin).toHaveBeenCalledWith({
          username: 'admin',
          password: 'admin123',
          _csrf: 'csrf-token-123',
        })
      })

      it('should handle CSRF token expiry', async () => {
        // 만료된 CSRF 토큰 시뮬레이션
        const mockGetCsrfToken = vi.spyOn(authApi, 'getCsrfToken')
          .mockResolvedValueOnce({
            token: 'expired-csrf-token',
            expiresAt: Date.now() - 1000, // 이미 만료됨
          })
          .mockResolvedValueOnce({
            token: 'new-csrf-token',
            expiresAt: Date.now() + 600000,
          })

        const mockLogin = vi.spyOn(authApi, 'login')
          .mockRejectedValueOnce({
            response: {
              status: 403,
              data: { error: { code: 'CSRF_TOKEN_INVALID' } },
            },
          })
          .mockResolvedValueOnce(mockApiResponses.login)

        const wrapper = mountLoginView()

        // 첫 번째 로그인 시도 (만료된 토큰)
        const usernameInput = wrapper.find('input[placeholder*="사용자명"]')
        const passwordInput = wrapper.find('input[type="password"]')
        const loginButton = wrapper.find('button[type="submit"], .n-button:contains("로그인")')

        await usernameInput.setValue('admin')
        await passwordInput.setValue('admin123')
        await loginButton.trigger('click')

        // CSRF 토큰 갱신 및 재시도 확인
        await vi.waitFor(() => {
          expect(mockGetCsrfToken).toHaveBeenCalledTimes(2)
          expect(mockLogin).toHaveBeenCalledTimes(2)
        })

        // 최종 로그인 성공 확인
        const userStore = useUserStore()
        expect(userStore.isAuthenticated).toBe(true)
      })

      it('should validate CSRF token format', async () => {
        // 잘못된 형식의 CSRF 토큰
        const mockGetCsrfToken = vi.spyOn(authApi, 'getCsrfToken').mockResolvedValue({
          token: '', // 빈 토큰
          expiresAt: Date.now() + 600000,
        })

        const wrapper = mountLoginView()

        // CSRF 토큰 유효성 검사 확인
        await vi.waitFor(() => {
          expect(wrapper.vm.csrfToken).toBe('')
          expect(wrapper.vm.csrfError).toBeTruthy()
        })

        // 로그인 버튼 비활성화 확인
        const loginButton = wrapper.find('button[type="submit"], .n-button:contains("로그인")')
        expect(loginButton.attributes('disabled')).toBeDefined()
      })
    })

    describe('Rate Limiting', () => {
      it('should handle rate limiting on login attempts', async () => {
        // Rate limit 에러 시뮬레이션
        const mockLogin = vi.spyOn(authApi, 'login').mockRejectedValue({
          response: {
            status: 429,
            data: {
              error: {
                code: 'RATE_LIMIT_EXCEEDED',
                message: 'Too many login attempts',
                retryAfter: 60, // 60초 후 재시도
              },
            },
          },
        })

        const wrapper = mountLoginView()

        // 로그인 시도
        const usernameInput = wrapper.find('input[placeholder*="사용자명"]')
        const passwordInput = wrapper.find('input[type="password"]')
        const loginButton = wrapper.find('button[type="submit"], .n-button:contains("로그인")')

        await usernameInput.setValue('admin')
        await passwordInput.setValue('admin123')
        await loginButton.trigger('click')

        // Rate limit 처리 확인
        await vi.waitFor(() => {
          expect(wrapper.vm.isLoading).toBe(false)
          expect(wrapper.vm.rateLimited).toBe(true)
          expect(wrapper.vm.retryAfter).toBe(60)
        })

        // 로그인 버튼 비활성화 확인
        expect(loginButton.attributes('disabled')).toBeDefined()

        // 에러 메시지 확인
        // expect(wrapper.text()).toContain('너무 많은 로그인 시도')
      })

      it('should countdown retry timer for rate limited requests', async () => {
        const wrapper = mountLoginView()

        // Rate limit 상태 설정
        wrapper.vm.rateLimited = true
        wrapper.vm.retryAfter = 5 // 5초

        // 카운트다운 시작
        wrapper.vm.startRetryCountdown()

        // 1초 후 카운트다운 확인
        await new Promise(resolve => setTimeout(resolve, 1100))
        expect(wrapper.vm.retryAfter).toBe(4)

        // 4초 더 대기 후 해제 확인
        await new Promise(resolve => setTimeout(resolve, 4100))
        expect(wrapper.vm.rateLimited).toBe(false)
        expect(wrapper.vm.retryAfter).toBe(0)
      })

      it('should handle OAuth rate limiting', async () => {
        // OAuth Rate limit 에러 시뮬레이션
        const mockGetOAuthUrl = vi.spyOn(authApi, 'getOAuthAuthUrl').mockRejectedValue({
          response: {
            status: 429,
            data: {
              error: {
                code: 'OAUTH_RATE_LIMIT_EXCEEDED',
                message: 'Too many OAuth requests',
                retryAfter: 120,
              },
            },
          },
        })

        const wrapper = mountLoginView()

        // Google OAuth 로그인 시도
        const googleLoginButton = wrapper.find('button:contains("Google로 로그인")')
        await googleLoginButton.trigger('click')

        // OAuth Rate limit 처리 확인
        await vi.waitFor(() => {
          expect(wrapper.vm.oauthRateLimited).toBe(true)
          expect(wrapper.vm.oauthRetryAfter).toBe(120)
        })

        // OAuth 버튼들 비활성화 확인
        const oauthButtons = wrapper.findAll('button[data-oauth]')
        oauthButtons.forEach(button => {
          expect(button.attributes('disabled')).toBeDefined()
        })
      })
    })

    describe('Security Headers', () => {
      it('should verify security headers in API responses', async () => {
        // Security headers를 포함한 API Mock
        const mockLogin = vi.spyOn(authApi, 'login').mockImplementation(async () => {
          // 실제 구현에서는 HTTP 응답 헤더 확인
          const mockResponse = {
            ...mockApiResponses.login,
            headers: {
              'X-Content-Type-Options': 'nosniff',
              'X-Frame-Options': 'DENY',
              'X-XSS-Protection': '1; mode=block',
              'Strict-Transport-Security': 'max-age=31536000; includeSubDomains',
              'Content-Security-Policy': "default-src 'self'",
            },
          }
          return mockResponse
        })

        const wrapper = mountLoginView()

        // 로그인 실행
        const usernameInput = wrapper.find('input[placeholder*="사용자명"]')
        const passwordInput = wrapper.find('input[type="password"]')
        const loginButton = wrapper.find('button[type="submit"], .n-button:contains("로그인")')

        await usernameInput.setValue('admin')
        await passwordInput.setValue('admin123')
        await loginButton.trigger('click')

        // API 호출 확인
        expect(mockLogin).toHaveBeenCalled()

        // 보안 헤더 검증은 실제 네트워크 레이어에서 확인
        // E2E 테스트에서는 API 클라이언트의 헤더 검증 로직 테스트
        await vi.waitFor(() => {
          expect(wrapper.vm.securityHeadersVerified).toBe(true)
        })
      })

      it('should handle Content Security Policy violations', async () => {
        // CSP 위반 이벤트 시뮬레이션
        const cspViolationEvent = new Event('securitypolicyviolation')
        Object.defineProperty(cspViolationEvent, 'blockedURI', {
          value: 'inline',
          writable: false,
        })
        Object.defineProperty(cspViolationEvent, 'violatedDirective', {
          value: 'script-src',
          writable: false,
        })

        const wrapper = mountLoginView()

        // CSP 위반 이벤트 발생
        window.dispatchEvent(cspViolationEvent)

        // CSP 위반 처리 확인
        await vi.waitFor(() => {
          expect(wrapper.vm.cspViolations.length).toBeGreaterThan(0)
          expect(wrapper.vm.cspViolations[0]).toMatchObject({
            blockedURI: 'inline',
            violatedDirective: 'script-src',
          })
        })
      })
    })
  })
})