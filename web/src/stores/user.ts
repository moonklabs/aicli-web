import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import type { 
  OAuthAccount, 
  OAuthProvider, 
  Role, 
  Permission, 
  UserRole, 
  UserPermissions,
  PermissionCheck,
  PermissionCheckResponse 
} from '@/types/api'
import { PermissionUtils } from '@/utils/permission'

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
  
  // RBAC 관련 상태
  const userRoles = ref<UserRole[]>([])
  const availableRoles = ref<Role[]>([])
  const userPermissions = ref<UserPermissions | null>(null)
  const permissionCache = ref<Map<string, boolean>>(new Map())
  const permissionsLoading = ref(false)
  const permissionsError = ref<string | null>(null)
  const lastPermissionUpdate = ref<Date | null>(null)

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
  
  // RBAC 관련 계산된 속성
  const hasPermissions = computed(() => userPermissions.value !== null)
  
  const currentUserRoles = computed(() => 
    userRoles.value.filter(ur => ur.isActive).map(ur => ur.role?.name).filter(Boolean)
  )
  
  const isAdmin = computed(() => 
    currentUserRoles.value.includes('admin') || 
    currentUserRoles.value.includes('super_admin')
  )
  
  const isSuperAdmin = computed(() => 
    currentUserRoles.value.includes('super_admin')
  )
  
  const permissionsSummary = computed(() => {
    if (!userPermissions.value) return null
    
    const permissions = userPermissions.value.finalPermissions
    const summary = {
      total: Object.keys(permissions).length,
      allowed: 0,
      denied: 0,
      byResource: {} as Record<string, number>
    }
    
    Object.values(permissions).forEach(permission => {
      if (permission.effect === 'allow') {
        summary.allowed++
      } else {
        summary.denied++
      }
      
      const resource = permission.resourceType
      summary.byResource[resource] = (summary.byResource[resource] || 0) + 1
    })
    
    return summary
  })

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
    
    // RBAC 정보도 함께 초기화
    clearPermissions()
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
  
  // RBAC 관련 액션
  const setUserRoles = (roles: UserRole[]) => {
    userRoles.value = roles
    // PermissionUtils에도 역할 정보 동기화
    const roleObjects = roles.map(ur => ur.role).filter(Boolean) as Role[]
    PermissionUtils.setUserRoles(roleObjects)
  }
  
  const setAvailableRoles = (roles: Role[]) => {
    availableRoles.value = roles
  }
  
  const setUserPermissions = (permissions: UserPermissions) => {
    userPermissions.value = permissions
    lastPermissionUpdate.value = new Date()
    
    // PermissionUtils에 권한 정보 동기화
    PermissionUtils.setUserPermissions(permissions)
  }
  
  const clearPermissions = () => {
    userPermissions.value = null
    userRoles.value = []
    availableRoles.value = []
    permissionCache.value.clear()
    lastPermissionUpdate.value = null
    
    // PermissionUtils도 초기화
    PermissionUtils.clearCache()
  }
  
  const setPermissionsLoading = (loading: boolean) => {
    permissionsLoading.value = loading
  }
  
  const setPermissionsError = (errorMessage: string | null) => {
    permissionsError.value = errorMessage
  }
  
  const addUserRole = (userRole: UserRole) => {
    const existingIndex = userRoles.value.findIndex(ur => ur.roleId === userRole.roleId)
    
    if (existingIndex >= 0) {
      userRoles.value[existingIndex] = userRole
    } else {
      userRoles.value.push(userRole)
    }
    
    // 역할 변경 시 권한 정보 재로드 트리거
    refreshPermissions()
  }
  
  const removeUserRole = (roleId: string) => {
    userRoles.value = userRoles.value.filter(ur => ur.roleId !== roleId)
    
    // 역할 변경 시 권한 정보 재로드 트리거
    refreshPermissions()
  }
  
  const checkPermission = (resourceType: string, action: string, resourceId?: string): boolean => {
    const cacheKey = `${resourceType}:${action}:${resourceId || '*'}`
    
    // 캐시에서 먼저 확인
    if (permissionCache.value.has(cacheKey)) {
      return permissionCache.value.get(cacheKey)!
    }
    
    // PermissionUtils를 통해 권한 체크
    const hasPermission = PermissionUtils.hasPermission(
      resourceType as any, 
      action as any, 
      resourceId
    )
    
    // 캐시에 저장 (5분 후 자동 삭제)
    permissionCache.value.set(cacheKey, hasPermission)
    setTimeout(() => {
      permissionCache.value.delete(cacheKey)
    }, 5 * 60 * 1000)
    
    return hasPermission
  }
  
  const hasRole = (roleName: string): boolean => {
    return currentUserRoles.value.includes(roleName)
  }
  
  const refreshPermissions = async (): Promise<void> => {
    if (!user.value) return
    
    try {
      setPermissionsLoading(true)
      setPermissionsError(null)
      
      // TODO: 실제 API 호출로 권한 정보 새로고침
      console.log('Refreshing user permissions...')
      
      // 임시로 기본 권한 설정 (실제 구현에서는 API 호출)
      const mockPermissions: UserPermissions = {
        userId: user.value.id,
        directRoles: userRoles.value.map(ur => ur.roleId),
        inheritedRoles: [],
        groupRoles: [],
        finalPermissions: {
          'system:*:read': {
            resourceType: 'system',
            resourceId: '*',
            action: 'read',
            effect: 'allow',
            source: 'role',
            reason: 'Basic user access'
          },
          'workspace:*:read': {
            resourceType: 'workspace',
            resourceId: '*',
            action: 'read',
            effect: 'allow',
            source: 'role',
            reason: 'Workspace access'
          }
        },
        computedAt: new Date().toISOString()
      }
      
      setUserPermissions(mockPermissions)
    } catch (error) {
      console.error('Failed to refresh permissions:', error)
      setPermissionsError('권한 정보를 불러오는데 실패했습니다.')
    } finally {
      setPermissionsLoading(false)
    }
  }
  
  const loadAvailableRoles = async (): Promise<void> => {
    try {
      // TODO: 실제 API 호출로 사용 가능한 역할 목록 로드
      console.log('Loading available roles...')
      
      // 임시로 기본 역할 설정
      const mockRoles: Role[] = [
        {
          id: '1',
          name: 'user',
          description: '일반 사용자',
          level: 1,
          isSystem: true,
          isActive: true,
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString()
        },
        {
          id: '2',
          name: 'admin',
          description: '관리자',
          level: 5,
          isSystem: true,
          isActive: true,
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString()
        }
      ]
      
      setAvailableRoles(mockRoles)
    } catch (error) {
      console.error('Failed to load available roles:', error)
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
  const initializeAuth = async () => {
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
          
          // 토큰이 유효한 경우 권한 정보도 로드
          await refreshPermissions()
          await loadAvailableRoles()
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
    
    // RBAC 관련 상태
    userRoles,
    availableRoles,
    userPermissions,
    permissionCache,
    permissionsLoading,
    permissionsError,
    lastPermissionUpdate,

    // 계산된 속성
    isAuthenticated,
    currentUser,
    availableOAuthProviders,
    isOAuthAccountLinked,
    
    // RBAC 관련 계산된 속성
    hasPermissions,
    currentUserRoles,
    isAdmin,
    isSuperAdmin,
    permissionsSummary,

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
    
    // RBAC 관련 액션
    setUserRoles,
    setAvailableRoles,
    setUserPermissions,
    clearPermissions,
    setPermissionsLoading,
    setPermissionsError,
    addUserRole,
    removeUserRole,
    checkPermission,
    hasRole,
    refreshPermissions,
    loadAvailableRoles,
  }
})