import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import type { OAuthAccount, OAuthProvider } from '@/types/api'

export interface User {
  id: string
  username: string
  email: string
  displayName?: string
  avatar?: string
  roles?: string[]
  oauthAccounts?: OAuthAccount[]
}

export interface AuthState {
  token?: string
  refreshToken?: string
  expiresAt?: number
}

export const useUserStore = defineStore('user', () => {
  // 상태
  const user = ref<User | null>(null)
  const authState = ref<AuthState>({})
  const isLoading = ref(false)
  const error = ref<string | null>(null)
  
  // OAuth 관련 상태
  const oauthProviders = ref<OAuthProvider[]>([])
  const linkedOAuthAccounts = ref<OAuthAccount[]>([])
  const oauthLoading = ref(false)
  const oauthError = ref<string | null>(null)

  // 계산된 속성
  const isAuthenticated = computed(() => {
    if (!authState.value.token || !authState.value.expiresAt) {
      return false
    }
    return Date.now() < authState.value.expiresAt
  })

  const currentUser = computed(() => user.value)

  // OAuth 관련 계산된 속성
  const availableOAuthProviders = computed(() => 
    oauthProviders.value.filter(provider => provider.enabled)
  )
  
  const isOAuthAccountLinked = computed(() => (providerName: string) => 
    linkedOAuthAccounts.value.some(account => 
      account.provider === providerName && account.connected
    )
  )

  // 액션
  const setUser = (userData: User) => {
    user.value = userData
  }

  const setAuth = (auth: AuthState) => {
    authState.value = auth
    // 토큰을 localStorage에 저장 (실제 구현에서는 보안을 고려해야 함)
    if (auth.token) {
      localStorage.setItem('auth_token', auth.token)
    }
    if (auth.refreshToken) {
      localStorage.setItem('refresh_token', auth.refreshToken)
    }
  }

  const clearAuth = () => {
    user.value = null
    authState.value = {}
    localStorage.removeItem('auth_token')
    localStorage.removeItem('refresh_token')
  }

  const setLoading = (loading: boolean) => {
    isLoading.value = loading
  }

  const setError = (errorMessage: string | null) => {
    error.value = errorMessage
  }

  // OAuth 관련 액션
  const setOAuthProviders = (providers: OAuthProvider[]) => {
    oauthProviders.value = providers
  }

  const setLinkedOAuthAccounts = (accounts: OAuthAccount[]) => {
    linkedOAuthAccounts.value = accounts
    // 사용자 정보에도 동기화
    if (user.value) {
      user.value.oauthAccounts = accounts
    }
  }

  const setOAuthLoading = (loading: boolean) => {
    oauthLoading.value = loading
  }

  const setOAuthError = (errorMessage: string | null) => {
    oauthError.value = errorMessage
  }

  const addLinkedOAuthAccount = (account: OAuthAccount) => {
    const existingIndex = linkedOAuthAccounts.value.findIndex(
      acc => acc.provider === account.provider
    )
    
    if (existingIndex >= 0) {
      linkedOAuthAccounts.value[existingIndex] = account
    } else {
      linkedOAuthAccounts.value.push(account)
    }
    
    // 사용자 정보에도 동기화
    if (user.value) {
      user.value.oauthAccounts = [...linkedOAuthAccounts.value]
    }
  }

  const removeLinkedOAuthAccount = (provider: string) => {
    linkedOAuthAccounts.value = linkedOAuthAccounts.value.filter(
      account => account.provider !== provider
    )
    
    // 사용자 정보에도 동기화
    if (user.value) {
      user.value.oauthAccounts = [...linkedOAuthAccounts.value]
    }
  }

  // 토큰 갱신 함수
  const refreshToken = async (): Promise<boolean> => {
    try {
      setLoading(true)
      // TODO: API 호출로 토큰 갱신 구현
      console.log('Token refresh not implemented yet')
      return false
    } catch (err) {
      console.error('Token refresh failed:', err)
      return false
    } finally {
      setLoading(false)
    }
  }

  // 초기화 (localStorage에서 토큰 복원)
  const initializeAuth = () => {
    const token = localStorage.getItem('auth_token')
    const refreshTokenValue = localStorage.getItem('refresh_token')

    if (token) {
      try {
        // JWT 토큰 디코딩 (실제 구현에서는 라이브러리 사용 권장)
        const payload = JSON.parse(atob(token.split('.')[1]))
        const expiresAt = payload.exp * 1000

        if (Date.now() < expiresAt) {
          authState.value = {
            token,
            refreshToken: refreshTokenValue || undefined,
            expiresAt,
          }
        } else {
          clearAuth()
        }
      } catch (err) {
        console.error('Failed to parse token:', err)
        clearAuth()
      }
    }
  }

  return {
    // 상태
    user,
    authState,
    isLoading,
    error,
    
    // OAuth 관련 상태
    oauthProviders,
    linkedOAuthAccounts,
    oauthLoading,
    oauthError,

    // 계산된 속성
    isAuthenticated,
    currentUser,
    availableOAuthProviders,
    isOAuthAccountLinked,

    // 액션
    setUser,
    setAuth,
    clearAuth,
    setLoading,
    setError,
    refreshToken,
    initializeAuth,
    
    // OAuth 관련 액션
    setOAuthProviders,
    setLinkedOAuthAccounts,
    setOAuthLoading,
    setOAuthError,
    addLinkedOAuthAccount,
    removeLinkedOAuthAccount,
  }
})