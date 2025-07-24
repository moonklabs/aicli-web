import { apiGet, apiPost, apiPut } from '@/api'
import type {
  UserSession,
  SessionSecurityEvent,
  SessionSecuritySettings,
  SessionStatsResponse,
  TerminateSessionRequest,
  TerminateAllSessionsRequest,
  UpdateSessionSettingsRequest,
  PaginatedResponse
} from '@/types/api'

export const sessionApi = {
  /**
   * 현재 사용자의 모든 활성 세션 조회
   */
  getActiveSessions: async (): Promise<UserSession[]> => {
    const response = await apiGet<UserSession[]>('/auth/sessions')
    return response.data.data
  },

  /**
   * 특정 세션 정보 조회
   */
  getSession: async (sessionId: string): Promise<UserSession> => {
    const response = await apiGet<UserSession>(`/auth/sessions/${sessionId}`)
    return response.data.data
  },

  /**
   * 현재 사용자의 세션 통계 조회
   */
  getSessionStats: async (): Promise<SessionStatsResponse> => {
    const response = await apiGet<SessionStatsResponse>('/auth/sessions/stats')
    return response.data.data
  },

  /**
   * 특정 세션 강제 종료
   */
  terminateSession: async (request: TerminateSessionRequest): Promise<void> => {
    await apiPost(`/auth/sessions/${request.sessionId}/terminate`, {
      reason: request.reason
    })
  },

  /**
   * 현재 세션을 제외한 모든 세션 종료
   */
  terminateAllSessions: async (request: TerminateAllSessionsRequest): Promise<void> => {
    await apiPost('/auth/sessions/terminate-all', request)
  },

  /**
   * 세션 보안 설정 조회
   */
  getSecuritySettings: async (): Promise<SessionSecuritySettings> => {
    const response = await apiGet<SessionSecuritySettings>('/auth/settings/security')
    return response.data.data
  },

  /**
   * 세션 보안 설정 업데이트
   */
  updateSecuritySettings: async (request: UpdateSessionSettingsRequest): Promise<SessionSecuritySettings> => {
    const response = await apiPut<SessionSecuritySettings>('/auth/settings/security', request)
    return response.data.data
  },

  /**
   * 세션 보안 이벤트 히스토리 조회
   */
  getSecurityEvents: async (params?: {
    page?: number
    limit?: number
    eventType?: string
    severity?: string
    startDate?: string
    endDate?: string
  }): Promise<PaginatedResponse<SessionSecurityEvent>> => {
    const queryParams = new URLSearchParams()
    
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined) {
          queryParams.append(key, value.toString())
        }
      })
    }
    
    const response = await apiGet<PaginatedResponse<SessionSecurityEvent>>(
      `/auth/security-events?${queryParams.toString()}`
    )
    return response.data.data
  },

  /**
   * 특정 세션의 보안 이벤트 조회
   */
  getSessionSecurityEvents: async (sessionId: string): Promise<SessionSecurityEvent[]> => {
    const response = await apiGet<SessionSecurityEvent[]>(`/auth/sessions/${sessionId}/security-events`)
    return response.data.data
  },

  /**
   * 의심스러운 활동 보고
   */
  reportSuspiciousActivity: async (sessionId: string, reason: string): Promise<void> => {
    await apiPost(`/auth/sessions/${sessionId}/report-suspicious`, {
      reason
    })
  },

  /**
   * 세션 활동 강제 업데이트 (활성상태 유지)
   */
  refreshSession: async (): Promise<void> => {
    await apiPost('/auth/sessions/refresh')
  },

  /**
   * 현재 세션 정보 조회
   */
  getCurrentSession: async (): Promise<UserSession> => {
    const response = await apiGet<UserSession>('/auth/sessions/current')
    return response.data.data
  }
}