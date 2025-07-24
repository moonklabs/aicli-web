import { apiGet, apiPost } from '@/api'
import type {
  LoginRequest,
  LoginResponse,
  RefreshTokenRequest,
  OAuthAuthUrlRequest,
  OAuthAuthUrlResponse,
  OAuthCallbackRequest,
  OAuthAccount,
  LinkOAuthRequest,
  UnlinkOAuthRequest,
} from '@/types/api'

export const authApi = {
  /**
   * 로그인
   */
  login: async (credentials: LoginRequest): Promise<LoginResponse> => {
    const response = await apiPost<LoginResponse>('/auth/login', credentials)
    return response.data.data
  },

  /**
   * 로그아웃
   */
  logout: async (): Promise<void> => {
    await apiPost('/auth/logout')
  },

  /**
   * 토큰 갱신
   */
  refreshToken: async (request: RefreshTokenRequest): Promise<LoginResponse> => {
    const response = await apiPost<LoginResponse>('/auth/refresh', request)
    return response.data.data
  },

  /**
   * 현재 사용자 정보 조회
   */
  getCurrentUser: async () => {
    const response = await apiGet('/auth/me')
    return response.data.data
  },

  /**
   * 패스워드 변경
   */
  changePassword: async (data: {
    currentPassword: string
    newPassword: string
    confirmPassword: string
  }): Promise<void> => {
    await apiPost('/auth/change-password', data)
  },

  /**
   * 패스워드 리셋 요청
   */
  requestPasswordReset: async (email: string): Promise<void> => {
    await apiPost('/auth/password-reset-request', { email })
  },

  /**
   * 패스워드 리셋 확인
   */
  resetPassword: async (data: {
    token: string
    newPassword: string
    confirmPassword: string
  }): Promise<void> => {
    await apiPost('/auth/password-reset-confirm', data)
  },

  /**
   * 이메일 인증 요청
   */
  requestEmailVerification: async (): Promise<void> => {
    await apiPost('/auth/email-verification-request')
  },

  /**
   * 이메일 인증 확인
   */
  verifyEmail: async (token: string): Promise<void> => {
    await apiPost('/auth/email-verification-confirm', { token })
  },

  // OAuth 관련 API
  
  /**
   * OAuth 인증 URL 생성
   */
  getOAuthAuthUrl: async (request: OAuthAuthUrlRequest): Promise<OAuthAuthUrlResponse> => {
    const response = await apiPost<OAuthAuthUrlResponse>(`/auth/oauth/${request.provider}/auth-url`, {
      state: request.state,
    })
    return response.data.data
  },

  /**
   * OAuth 로그인 (콜백 처리)
   */
  oAuthLogin: async (request: OAuthCallbackRequest): Promise<LoginResponse> => {
    const response = await apiPost<LoginResponse>(`/auth/oauth/${request.provider}/callback`, {
      code: request.code,
      state: request.state,
    })
    return response.data.data
  },

  /**
   * OAuth 계정 연결
   */
  linkOAuthAccount: async (request: LinkOAuthRequest): Promise<void> => {
    await apiPost(`/auth/oauth/${request.provider}/link`, {
      code: request.code,
      state: request.state,
    })
  },

  /**
   * OAuth 계정 연결 해제
   */
  unlinkOAuthAccount: async (request: UnlinkOAuthRequest): Promise<void> => {
    await apiPost(`/auth/oauth/${request.provider}/unlink`)
  },

  /**
   * 연결된 OAuth 계정 목록 조회
   */
  getLinkedOAuthAccounts: async (): Promise<OAuthAccount[]> => {
    const response = await apiGet<OAuthAccount[]>('/auth/oauth/accounts')
    return response.data.data
  },

  /**
   * 사용 가능한 OAuth 제공자 목록 조회
   */
  getOAuthProviders: async () => {
    const response = await apiGet('/auth/oauth/providers')
    return response.data.data
  },
}