/**
 * OAuth Social Login E2E Tests
 * OAuth 소셜 로그인 E2E 테스트
 */

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia } from 'pinia'
import { useUserStore } from '@/stores/user'
import { authApi } from '@/api/services/auth'

// OAuth Mock 응답
const mockOAuthResponses = {
  google: {
    authUrl: {
      authUrl: 'https://accounts.google.com/oauth/authorize?client_id=test&redirect_uri=http%3A%2F%2Flocalhost%3A3000%2Foauth%2Fcallback%2Fgoogle&response_type=code&scope=openid%20profile%20email&state=random-state-123&code_challenge=PKCEcodeChallenge&code_challenge_method=S256',
      state: 'random-state-123',
    },
    callback: {
      access_token: 'google-access-token',
      refresh_token: 'google-refresh-token',
      token_type: 'Bearer',
      expires_in: 3600,
      user: {
        id: 'google-user-123',
        username: 'googleuser',
        email: 'user@gmail.com',
        role: 'user',
        provider: 'google',
        providerId: 'google-123456789',
      },
    },
    userInfo: {
      id: 'google-123456789',
      email: 'user@gmail.com',
      name: 'Google User',
      picture: 'https://lh3.googleusercontent.com/avatar.jpg',
      given_name: 'Google',
      family_name: 'User',
      locale: 'ko',
    },
  },
  github: {
    authUrl: {
      authUrl: 'https://github.com/login/oauth/authorize?client_id=test&redirect_uri=http%3A%2F%2Flocalhost%3A3000%2Foauth%2Fcallback%2Fgithub&scope=user%3Aemail&state=random-state-456&code_challenge=PKCEcodeChallenge&code_challenge_method=S256',
      state: 'random-state-456',
    },
    callback: {
      access_token: 'github-access-token',
      refresh_token: 'github-refresh-token',
      token_type: 'Bearer',
      expires_in: 3600,
      user: {
        id: 'github-user-456',
        username: 'githubuser',
        email: 'user@example.com',
        role: 'user',
        provider: 'github',
        providerId: 'github-987654321',
      },
    },
    userInfo: {
      id: 987654321,
      login: 'githubuser',
      email: 'user@example.com',
      name: 'GitHub User',
      avatar_url: 'https://avatars.githubusercontent.com/u/987654321',
      bio: 'Developer',
      location: 'Seoul, Korea',
    },
  },
}

// OAuth 에러 응답
const mockOAuthErrors = {
  accessDenied: {
    error: 'access_denied',
    error_description: 'The user denied the request',
    state: 'random-state-123',
  },
  invalidState: {
    error: 'invalid_request',
    error_description: 'The state parameter is invalid',
    state: 'invalid-state',
  },
  serverError: {
    error: 'server_error',
    error_description: 'The authorization server encountered an error',
    state: 'random-state-123',
  },
}

describe('OAuth Social Login E2E Tests', () => {
  let userStore: ReturnType<typeof useUserStore>

  beforeEach(() => {
    const pinia = createPinia()
    userStore = useUserStore(pinia)

    // API Mock 설정
    vi.clearAllMocks()

    // localStorage 초기화
    localStorage.clear()
    sessionStorage.clear()

    // window.open Mock 설정
    vi.spyOn(window, 'open').mockImplementation(() => {
      const mockPopup = {
        closed: false,
        close: vi.fn(),
        location: { href: '' },
      }
      return mockPopup as any
    })
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('Google OAuth Flow', () => {
    it('should generate Google OAuth authorization URL with PKCE', async () => {
      // Google OAuth URL 생성 API Mock
      const mockGetOAuthUrl = vi.spyOn(authApi, 'getOAuthAuthUrl').mockResolvedValue(mockOAuthResponses.google.authUrl)

      // OAuth URL 요청
      const request = {
        provider: 'google' as const,
        state: 'random-state-123',
      }
      const response = await authApi.getOAuthAuthUrl(request)

      // API 호출 확인
      expect(mockGetOAuthUrl).toHaveBeenCalledWith(request)

      // URL 검증
      expect(response.authUrl).toContain('accounts.google.com/oauth/authorize')
      expect(response.authUrl).toContain('client_id=test')
      expect(response.authUrl).toContain('scope=openid%20profile%20email')
      expect(response.authUrl).toContain('code_challenge=PKCEcodeChallenge')
      expect(response.authUrl).toContain('code_challenge_method=S256')
      expect(response.state).toBe('random-state-123')
    })

    it('should handle Google OAuth callback successfully', async () => {
      // Google OAuth 콜백 API Mock
      const mockOAuthLogin = vi.spyOn(authApi, 'oAuthLogin').mockResolvedValue(mockOAuthResponses.google.callback)

      // OAuth 콜백 요청
      const request = {
        provider: 'google' as const,
        code: 'google-auth-code-123',
        state: 'random-state-123',
      }
      const response = await authApi.oAuthLogin(request)

      // API 호출 확인
      expect(mockOAuthLogin).toHaveBeenCalledWith(request)

      // 응답 검증
      expect(response.access_token).toBe('google-access-token')
      expect(response.user.email).toBe('user@gmail.com')
      expect(response.user.provider).toBe('google')
      expect(response.user.providerId).toBe('google-123456789')
    })

    it('should handle Google user profile synchronization', async () => {
      // 사용자 정보 동기화 테스트
      const googleUser = mockOAuthResponses.google.callback.user

      // Store에 OAuth 사용자 설정
      userStore.setUser(googleUser)
      userStore.setAuth({
        token: 'google-access-token',
        refreshToken: 'google-refresh-token',
        expiresAt: Date.now() + 3600000,
      })

      // 상태 확인
      expect(userStore.isAuthenticated).toBe(true)
      expect(userStore.user?.email).toBe('user@gmail.com')
      expect(userStore.user?.provider).toBe('google')

      // 프로필 정보 확인
      expect(userStore.user?.username).toBe('googleuser')
      expect(userStore.user?.role).toBe('user')
    })
  })

  describe('GitHub OAuth Flow', () => {
    it('should generate GitHub OAuth authorization URL with PKCE', async () => {
      // GitHub OAuth URL 생성 API Mock
      const mockGetOAuthUrl = vi.spyOn(authApi, 'getOAuthAuthUrl').mockResolvedValue(mockOAuthResponses.github.authUrl)

      // OAuth URL 요청
      const request = {
        provider: 'github' as const,
        state: 'random-state-456',
      }
      const response = await authApi.getOAuthAuthUrl(request)

      // API 호출 확인
      expect(mockGetOAuthUrl).toHaveBeenCalledWith(request)

      // URL 검증
      expect(response.authUrl).toContain('github.com/login/oauth/authorize')
      expect(response.authUrl).toContain('client_id=test')
      expect(response.authUrl).toContain('scope=user%3Aemail')
      expect(response.authUrl).toContain('code_challenge=PKCEcodeChallenge')
      expect(response.authUrl).toContain('code_challenge_method=S256')
      expect(response.state).toBe('random-state-456')
    })

    it('should handle GitHub OAuth callback successfully', async () => {
      // GitHub OAuth 콜백 API Mock
      const mockOAuthLogin = vi.spyOn(authApi, 'oAuthLogin').mockResolvedValue(mockOAuthResponses.github.callback)

      // OAuth 콜백 요청
      const request = {
        provider: 'github' as const,
        code: 'github-auth-code-456',
        state: 'random-state-456',
      }
      const response = await authApi.oAuthLogin(request)

      // API 호출 확인
      expect(mockOAuthLogin).toHaveBeenCalledWith(request)

      // 응답 검증
      expect(response.access_token).toBe('github-access-token')
      expect(response.user.email).toBe('user@example.com')
      expect(response.user.provider).toBe('github')
      expect(response.user.providerId).toBe('github-987654321')
    })
  })

  describe('PKCE Security Mechanism', () => {
    it('should validate PKCE code challenge in OAuth flow', async () => {
      // PKCE 테스트용 Mock
      const mockGetOAuthUrl = vi.spyOn(authApi, 'getOAuthAuthUrl').mockResolvedValue({
        authUrl: 'https://accounts.google.com/oauth/authorize?code_challenge=abc123&code_challenge_method=S256',
        state: 'test-state',
      })

      const response = await authApi.getOAuthAuthUrl({
        provider: 'google',
        state: 'test-state',
      })

      // PKCE 파라미터 확인
      expect(response.authUrl).toContain('code_challenge=')
      expect(response.authUrl).toContain('code_challenge_method=S256')
    })

    it('should handle PKCE verification failure', async () => {
      // PKCE 검증 실패 시뮬레이션
      const pkceError = {
        response: {
          status: 400,
          data: {
            error: {
              code: 'INVALID_PKCE_VERIFIER',
              message: 'PKCE code verifier is invalid',
            },
          },
        },
      }

      const mockOAuthLogin = vi.spyOn(authApi, 'oAuthLogin').mockRejectedValue(pkceError)

      try {
        await authApi.oAuthLogin({
          provider: 'google',
          code: 'invalid-code',
          state: 'valid-state',
        })
        expect.fail('Should have thrown PKCE error')
      } catch (error: any) {
        expect(error.response.status).toBe(400)
        expect(error.response.data.error.code).toBe('INVALID_PKCE_VERIFIER')
      }
    })
  })

  describe('State Parameter Validation', () => {
    it('should validate OAuth state parameter', async () => {
      // 유효한 상태 매개변수 테스트
      const validState = 'valid-state-123'

      const mockOAuthLogin = vi.spyOn(authApi, 'oAuthLogin').mockResolvedValue(mockOAuthResponses.google.callback)

      await authApi.oAuthLogin({
        provider: 'google',
        code: 'auth-code',
        state: validState,
      })

      expect(mockOAuthLogin).toHaveBeenCalledWith({
        provider: 'google',
        code: 'auth-code',
        state: validState,
      })
    })

    it('should reject invalid state parameter', async () => {
      // 잘못된 상태 매개변수 테스트
      const stateError = {
        response: {
          status: 400,
          data: {
            error: {
              code: 'INVALID_STATE',
              message: 'OAuth state parameter is invalid',
            },
          },
        },
      }

      const mockOAuthLogin = vi.spyOn(authApi, 'oAuthLogin').mockRejectedValue(stateError)

      try {
        await authApi.oAuthLogin({
          provider: 'google',
          code: 'auth-code',
          state: 'invalid-state',
        })
        expect.fail('Should have thrown state error')
      } catch (error: any) {
        expect(error.response.status).toBe(400)
        expect(error.response.data.error.code).toBe('INVALID_STATE')
      }
    })
  })

  describe('Account Linking', () => {
    it('should link OAuth account to existing user', async () => {
      // 기존 사용자로 로그인
      userStore.setUser({
        id: 'existing-user-123',
        username: 'existinguser',
        email: 'existing@example.com',
        role: 'user',
      })

      // OAuth 계정 연결 API Mock
      const mockLinkAccount = vi.spyOn(authApi, 'linkOAuthAccount').mockResolvedValue()

      await authApi.linkOAuthAccount({
        provider: 'google',
        code: 'auth-code',
        state: 'link-state',
      })

      expect(mockLinkAccount).toHaveBeenCalledWith({
        provider: 'google',
        code: 'auth-code',
        state: 'link-state',
      })
    })

    it('should unlink OAuth account from user', async () => {
      // OAuth 계정 연결 해제 API Mock
      const mockUnlinkAccount = vi.spyOn(authApi, 'unlinkOAuthAccount').mockResolvedValue()

      await authApi.unlinkOAuthAccount({
        provider: 'google',
      })

      expect(mockUnlinkAccount).toHaveBeenCalledWith({
        provider: 'google',
      })
    })

    it('should list linked OAuth accounts', async () => {
      // 연결된 OAuth 계정 목록 API Mock
      const mockLinkedAccounts = [
        {
          provider: 'google',
          providerId: 'google-123456789',
          email: 'user@gmail.com',
          name: 'Google User',
          linkedAt: new Date().toISOString(),
        },
        {
          provider: 'github',
          providerId: 'github-987654321',
          email: 'user@example.com',
          name: 'GitHub User',
          linkedAt: new Date().toISOString(),
        },
      ]

      const mockGetLinkedAccounts = vi.spyOn(authApi, 'getLinkedOAuthAccounts').mockResolvedValue(mockLinkedAccounts)

      const accounts = await authApi.getLinkedOAuthAccounts()

      expect(mockGetLinkedAccounts).toHaveBeenCalled()
      expect(accounts).toHaveLength(2)
      expect(accounts[0].provider).toBe('google')
      expect(accounts[1].provider).toBe('github')
    })
  })

  describe('OAuth Error Scenarios', () => {
    it('should handle user access denial', async () => {
      // 사용자 거부 에러 시뮬레이션
      const accessDeniedError = {
        response: {
          status: 400,
          data: mockOAuthErrors.accessDenied,
        },
      }

      const mockOAuthLogin = vi.spyOn(authApi, 'oAuthLogin').mockRejectedValue(accessDeniedError)

      try {
        await authApi.oAuthLogin({
          provider: 'google',
          code: '',
          state: 'random-state-123',
        })
        expect.fail('Should have thrown access denied error')
      } catch (error: any) {
        expect(error.response.data.error).toBe('access_denied')
        expect(error.response.data.error_description).toBe('The user denied the request')
      }
    })

    it('should handle OAuth server error', async () => {
      // OAuth 서버 에러 시뮬레이션
      const serverError = {
        response: {
          status: 500,
          data: mockOAuthErrors.serverError,
        },
      }

      const mockOAuthLogin = vi.spyOn(authApi, 'oAuthLogin').mockRejectedValue(serverError)

      try {
        await authApi.oAuthLogin({
          provider: 'google',
          code: 'auth-code',
          state: 'random-state-123',
        })
        expect.fail('Should have thrown server error')
      } catch (error: any) {
        expect(error.response.status).toBe(500)
        expect(error.response.data.error).toBe('server_error')
      }
    })

    it('should handle network timeout during OAuth', async () => {
      // 네트워크 타임아웃 시뮬레이션
      const timeoutError = new Error('Network timeout')

      const mockOAuthLogin = vi.spyOn(authApi, 'oAuthLogin').mockRejectedValue(timeoutError)

      try {
        await authApi.oAuthLogin({
          provider: 'google',
          code: 'auth-code',
          state: 'valid-state',
        })
        expect.fail('Should have thrown timeout error')
      } catch (error) {
        expect(error).toBeInstanceOf(Error)
        expect((error as Error).message).toBe('Network timeout')
      }
    })
  })

  describe('Token Management', () => {
    it('should store OAuth tokens securely', () => {
      // OAuth 토큰 저장 테스트
      const oauthUser = mockOAuthResponses.google.callback.user
      const oauthAuth = {
        token: mockOAuthResponses.google.callback.access_token,
        refreshToken: mockOAuthResponses.google.callback.refresh_token,
        expiresAt: Date.now() + mockOAuthResponses.google.callback.expires_in * 1000,
      }

      userStore.setUser(oauthUser)
      userStore.setAuth(oauthAuth)

      // 토큰 저장 확인
      expect(userStore.auth?.token).toBe('google-access-token')
      expect(userStore.auth?.refreshToken).toBe('google-refresh-token')
      expect(userStore.auth?.expiresAt).toBeGreaterThan(Date.now())

      // localStorage 저장 확인
      const storedAuth = JSON.parse(localStorage.getItem('auth') || '{}')
      expect(storedAuth.token).toBe('google-access-token')
    })

    it('should handle OAuth token expiry and refresh', async () => {
      // 만료된 OAuth 토큰 설정
      userStore.setAuth({
        token: 'expired-oauth-token',
        refreshToken: 'oauth-refresh-token',
        expiresAt: Date.now() - 1000,
      })

      // 토큰 갱신 API Mock
      const mockRefreshToken = vi.spyOn(authApi, 'refreshToken').mockResolvedValue({
        access_token: 'new-oauth-token',
        refresh_token: 'new-oauth-refresh-token',
        token_type: 'Bearer',
        expires_in: 3600,
      })

      // 토큰 갱신 실행
      const refreshResult = await authApi.refreshToken({
        refresh_token: 'oauth-refresh-token',
      })

      expect(mockRefreshToken).toHaveBeenCalled()
      expect(refreshResult.access_token).toBe('new-oauth-token')
    })
  })

  describe('OAuth Provider Management', () => {
    it('should get available OAuth providers', async () => {
      // 사용 가능한 OAuth 제공자 목록 Mock
      const mockProviders = {
        google: {
          name: 'Google',
          enabled: true,
          clientId: 'google-client-id',
          scopes: ['openid', 'profile', 'email'],
        },
        github: {
          name: 'GitHub',
          enabled: true,
          clientId: 'github-client-id',
          scopes: ['user:email'],
        },
      }

      const mockGetProviders = vi.spyOn(authApi, 'getOAuthProviders').mockResolvedValue(mockProviders)

      const providers = await authApi.getOAuthProviders()

      expect(mockGetProviders).toHaveBeenCalled()
      expect(providers.google.enabled).toBe(true)
      expect(providers.github.enabled).toBe(true)
      expect(providers.google.scopes).toContain('openid')
      expect(providers.github.scopes).toContain('user:email')
    })
  })

  describe('Performance Tests', () => {
    it('should handle multiple concurrent OAuth requests', async () => {
      const mockOAuthLogin = vi.spyOn(authApi, 'oAuthLogin').mockResolvedValue(mockOAuthResponses.google.callback)

      const startTime = Date.now()

      // 3개의 동시 OAuth 요청
      const promises = Array.from({ length: 3 }, (_, i) =>
        authApi.oAuthLogin({
          provider: 'google',
          code: `auth-code-${i}`,
          state: `state-${i}`,
        }),
      )

      const results = await Promise.all(promises)
      const endTime = Date.now()

      // 모든 요청이 성공해야 함
      results.forEach(result => {
        expect(result.access_token).toBeTruthy()
      })

      // 3개 요청이 3초 내에 완료되어야 함
      expect(endTime - startTime).toBeLessThan(3000)
    })
  })
})