import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

export interface User {
  id: string
  username: string
  email: string
  displayName?: string
  avatar?: string
  roles?: string[]
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

  // 계산된 속성
  const isAuthenticated = computed(() => {
    if (!authState.value.token || !authState.value.expiresAt) {
      return false
    }
    return Date.now() < authState.value.expiresAt
  })

  const currentUser = computed(() => user.value)

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

    // 계산된 속성
    isAuthenticated,
    currentUser,

    // 액션
    setUser,
    setAuth,
    clearAuth,
    setLoading,
    setError,
    refreshToken,
    initializeAuth,
  }
})