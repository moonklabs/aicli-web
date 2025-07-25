import { apiGet, apiPost, apiPut, apiDelete } from '@/api'
import type {
  UserProfile,
  UpdateProfileRequest,
  ChangePasswordRequest,
  PasswordStrength,
  TwoFactorAuthSettings,
  EnableTwoFactorRequest,
  VerifyTwoFactorRequest,
  NotificationSettings,
  UpdateNotificationSettingsRequest,
  PrivacySettings,
  UpdatePrivacySettingsRequest,
  ProfileImageUploadResponse,
  AccountDeletionRequest,
  AccountDeactivationRequest,
  AccountReactivationRequest
} from '@/types/api'

export const profileApi = {
  /**
   * 사용자 프로파일 정보 조회
   */
  getProfile: async (): Promise<UserProfile> => {
    const response = await apiGet<UserProfile>('/auth/profile')
    return response.data.data
  },

  /**
   * 사용자 프로파일 정보 업데이트
   */
  updateProfile: async (request: UpdateProfileRequest): Promise<UserProfile> => {
    const response = await apiPut<UserProfile>('/auth/profile', request)
    return response.data.data
  },

  /**
   * 프로파일 이미지 업로드
   */
  uploadProfileImage: async (file: File, cropData?: {
    x: number
    y: number
    width: number
    height: number
    rotate?: number
    scaleX?: number
    scaleY?: number
  }): Promise<ProfileImageUploadResponse> => {
    const formData = new FormData()
    formData.append('image', file)
    
    if (cropData) {
      formData.append('cropData', JSON.stringify(cropData))
    }

    const response = await apiPost<ProfileImageUploadResponse>('/auth/profile/image', formData, {
      headers: {
        'Content-Type': 'multipart/form-data'
      }
    })
    return response.data.data
  },

  /**
   * 프로파일 이미지 삭제
   */
  deleteProfileImage: async (): Promise<void> => {
    await apiDelete('/auth/profile/image')
  },

  /**
   * 비밀번호 변경
   */
  changePassword: async (request: ChangePasswordRequest): Promise<void> => {
    await apiPost('/auth/profile/password', request)
  },

  /**
   * 비밀번호 강도 체크
   */
  checkPasswordStrength: async (password: string): Promise<PasswordStrength> => {
    const response = await apiPost<PasswordStrength>('/auth/profile/password/strength', { password })
    return response.data.data
  },

  /**
   * 2FA 설정 정보 조회
   */
  getTwoFactorSettings: async (): Promise<TwoFactorAuthSettings> => {
    const response = await apiGet<TwoFactorAuthSettings>('/auth/profile/2fa')
    return response.data.data
  },

  /**
   * 2FA 설정 시작 (시크릿 키 및 QR 코드 생성)
   */
  setupTwoFactor: async (): Promise<TwoFactorAuthSettings> => {
    const response = await apiPost<TwoFactorAuthSettings>('/auth/profile/2fa/setup')
    return response.data.data
  },

  /**
   * 2FA 활성화
   */
  enableTwoFactor: async (request: EnableTwoFactorRequest): Promise<TwoFactorAuthSettings> => {
    const response = await apiPost<TwoFactorAuthSettings>('/auth/profile/2fa/enable', request)
    return response.data.data
  },

  /**
   * 2FA 비활성화
   */
  disableTwoFactor: async (token: string): Promise<void> => {
    await apiPost('/auth/profile/2fa/disable', { token })
  },

  /**
   * 2FA 백업 코드 재생성
   */
  regenerateBackupCodes: async (): Promise<string[]> => {
    const response = await apiPost<{ backupCodes: string[] }>('/auth/profile/2fa/backup-codes')
    return response.data.data.backupCodes
  },

  /**
   * 알림 설정 조회
   */
  getNotificationSettings: async (): Promise<NotificationSettings> => {
    const response = await apiGet<NotificationSettings>('/auth/profile/notifications')
    return response.data.data
  },

  /**
   * 알림 설정 업데이트
   */
  updateNotificationSettings: async (request: UpdateNotificationSettingsRequest): Promise<NotificationSettings> => {
    const response = await apiPut<NotificationSettings>('/auth/profile/notifications', request)
    return response.data.data
  },

  /**
   * 개인정보 설정 조회
   */
  getPrivacySettings: async (): Promise<PrivacySettings> => {
    const response = await apiGet<PrivacySettings>('/auth/profile/privacy')
    return response.data.data
  },

  /**
   * 개인정보 설정 업데이트
   */
  updatePrivacySettings: async (request: UpdatePrivacySettingsRequest): Promise<PrivacySettings> => {
    const response = await apiPut<PrivacySettings>('/auth/profile/privacy', request)
    return response.data.data
  },

  /**
   * 계정 비활성화
   */
  deactivateAccount: async (request: AccountDeactivationRequest): Promise<void> => {
    await apiPost('/auth/profile/deactivate', request)
  },

  /**
   * 계정 활성화
   */
  reactivateAccount: async (request: AccountReactivationRequest): Promise<void> => {
    await apiPost('/auth/profile/reactivate', request)
  },

  /**
   * 계정 삭제 요청
   */
  requestAccountDeletion: async (request: AccountDeletionRequest): Promise<void> => {
    await apiPost('/auth/profile/delete', request)
  },

  /**
   * 계정 삭제 취소
   */
  cancelAccountDeletion: async (): Promise<void> => {
    await apiPost('/auth/profile/delete/cancel')
  },

  /**
   * 이메일 변경 요청
   */
  requestEmailChange: async (newEmail: string, password: string): Promise<void> => {
    await apiPost('/auth/profile/email/change-request', { newEmail, password })
  },

  /**
   * 이메일 변경 확인
   */
  confirmEmailChange: async (token: string): Promise<void> => {
    await apiPost('/auth/profile/email/change-confirm', { token })
  },

  /**
   * 이메일 인증 요청
   */
  requestEmailVerification: async (): Promise<void> => {
    await apiPost('/auth/profile/email/verify-request')
  },

  /**
   * 이메일 인증 확인
   */
  confirmEmailVerification: async (token: string): Promise<void> => {
    await apiPost('/auth/profile/email/verify-confirm', { token })
  },

  /**
   * 전화번호 인증 요청
   */
  requestPhoneVerification: async (phone: string): Promise<void> => {
    await apiPost('/auth/profile/phone/verify-request', { phone })
  },

  /**
   * 전화번호 인증 확인
   */
  confirmPhoneVerification: async (phone: string, code: string): Promise<void> => {
    await apiPost('/auth/profile/phone/verify-confirm', { phone, code })
  },

  /**
   * 활동 로그 조회
   */
  getActivityLog: async (params?: {
    page?: number
    limit?: number
    startDate?: string
    endDate?: string
  }) => {
    const queryParams = new URLSearchParams()
    
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined) {
          queryParams.append(key, value.toString())
        }
      })
    }
    
    const response = await apiGet(`/auth/profile/activity?${queryParams.toString()}`)
    return response.data.data
  },

  /**
   * 로그인 기록 조회
   */
  getLoginHistory: async (params?: {
    page?: number
    limit?: number
    startDate?: string
    endDate?: string
  }) => {
    const queryParams = new URLSearchParams()
    
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined) {
          queryParams.append(key, value.toString())
        }
      })
    }
    
    const response = await apiGet(`/auth/profile/login-history?${queryParams.toString()}`)
    return response.data.data
  }
}