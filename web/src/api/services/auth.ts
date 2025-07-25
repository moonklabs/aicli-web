import { apiGet, apiPost } from '@/api'
import type {
  AuditLog,
  LinkOAuthRequest,
  LogExportRequest,
  LogExportResponse,
  LoginHistory,
  LoginRequest,
  LoginResponse,
  OAuthAccount,
  OAuthAuthUrlRequest,
  OAuthAuthUrlResponse,
  OAuthCallbackRequest,
  PaginatedResponse,
  RefreshTokenRequest,
  SecurityEventFilter,
  SecurityStats,
  SessionSecurityEvent,
  SuspiciousActivity,
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

  // 보안 모니터링 및 감사 로그 관련 API

  /**
   * 감사 로그 조회
   */
  getAuditLogs: async (filters?: SecurityEventFilter): Promise<PaginatedResponse<AuditLog>> => {
    const params = new URLSearchParams()
    if (filters) {
      Object.entries(filters).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          if (Array.isArray(value)) {
            value.forEach(v => params.append(key, v.toString()))
          } else {
            params.append(key, value.toString())
          }
        }
      })
    }
    const response = await apiGet<PaginatedResponse<AuditLog>>(`/auth/audit/logs?${params.toString()}`)
    return response.data.data
  },

  /**
   * 로그인 이력 조회
   */
  getLoginHistory: async (filters?: SecurityEventFilter): Promise<PaginatedResponse<LoginHistory>> => {
    const params = new URLSearchParams()
    if (filters) {
      Object.entries(filters).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          if (Array.isArray(value)) {
            value.forEach(v => params.append(key, v.toString()))
          } else {
            params.append(key, value.toString())
          }
        }
      })
    }
    const response = await apiGet<PaginatedResponse<LoginHistory>>(`/auth/security/login-history?${params.toString()}`)
    return response.data.data
  },

  /**
   * 보안 이벤트 조회
   */
  getSecurityEvents: async (filters?: SecurityEventFilter): Promise<PaginatedResponse<SessionSecurityEvent>> => {
    const params = new URLSearchParams()
    if (filters) {
      Object.entries(filters).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          if (Array.isArray(value)) {
            value.forEach(v => params.append(key, v.toString()))
          } else {
            params.append(key, value.toString())
          }
        }
      })
    }
    const response = await apiGet<PaginatedResponse<SessionSecurityEvent>>(`/auth/security/events?${params.toString()}`)
    return response.data.data
  },

  /**
   * 의심스러운 활동 조회
   */
  getSuspiciousActivities: async (filters?: SecurityEventFilter): Promise<PaginatedResponse<SuspiciousActivity>> => {
    const params = new URLSearchParams()
    if (filters) {
      Object.entries(filters).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          if (Array.isArray(value)) {
            value.forEach(v => params.append(key, v.toString()))
          } else {
            params.append(key, value.toString())
          }
        }
      })
    }
    const response = await apiGet<PaginatedResponse<SuspiciousActivity>>(`/auth/security/suspicious-activities?${params.toString()}`)
    return response.data.data
  },

  /**
   * 보안 통계 조회
   */
  getSecurityStats: async (period?: string): Promise<SecurityStats> => {
    const params = period ? `?period=${period}` : ''
    const response = await apiGet<SecurityStats>(`/auth/security/stats${params}`)
    return response.data.data
  },

  /**
   * 의심스러운 활동 해결 처리
   */
  resolveSuspiciousActivity: async (activityId: string, resolution: string): Promise<void> => {
    await apiPost(`/auth/security/suspicious-activities/${activityId}/resolve`, { resolution })
  },

  /**
   * 보안 로그 내보내기 요청
   */
  exportSecurityLogs: async (request: LogExportRequest): Promise<LogExportResponse> => {
    const response = await apiPost<LogExportResponse>('/auth/security/export', request)
    return response.data.data
  },

  /**
   * 실시간 보안 알림 설정 조회
   */
  getSecurityAlertSettings: async () => {
    const response = await apiGet('/auth/security/alert-settings')
    return response.data.data
  },

  /**
   * 실시간 보안 알림 설정 업데이트
   */
  updateSecurityAlertSettings: async (settings: {
    enableRealTimeAlerts?: boolean
    notifyOnSuspiciousLogin?: boolean
    notifyOnNewDevice?: boolean
    notifyOnLocationChange?: boolean
    notifyOnHighRiskActivity?: boolean
    alertThreshold?: 'low' | 'medium' | 'high' | 'critical'
  }): Promise<void> => {
    await apiPost('/auth/security/alert-settings', settings)
  },
}