import { computed, ref, watch, onUnmounted } from 'vue'
import { useUserStore } from '@/stores/user'
import { PermissionUtils } from '@/utils/permission'
import type { ResourceType, ActionType } from '@/types/api'

/**
 * 권한 관리 컴포저블
 */
export function usePermission() {
  const userStore = useUserStore()
  
  // 권한 체크 함수
  const hasPermission = (
    resourceType: ResourceType,
    action: ActionType,
    resourceId?: string
  ) => {
    return computed(() => 
      userStore.checkPermission(resourceType, action, resourceId)
    )
  }
  
  // 역할 체크 함수
  const hasRole = (roleName: string) => {
    return computed(() => userStore.hasRole(roleName))
  }
  
  // 관리자 권한 체크
  const isAdmin = computed(() => userStore.isAdmin)
  const isSuperAdmin = computed(() => userStore.isSuperAdmin)
  
  // 권한 정보 로딩 상태
  const permissionsLoading = computed(() => userStore.permissionsLoading)
  const permissionsError = computed(() => userStore.permissionsError)
  const hasPermissions = computed(() => userStore.hasPermissions)
  
  // 현재 사용자 역할
  const currentUserRoles = computed(() => userStore.currentUserRoles)
  
  // 권한 요약 정보
  const permissionsSummary = computed(() => userStore.permissionsSummary)
  
  // 권한 새로고침
  const refreshPermissions = () => userStore.refreshPermissions()
  
  return {
    hasPermission,
    hasRole,
    isAdmin,
    isSuperAdmin,
    permissionsLoading,
    permissionsError,
    hasPermissions,
    currentUserRoles,
    permissionsSummary,
    refreshPermissions
  }
}

/**
 * 특정 권한에 대한 반응형 체크
 */
export function usePermissionCheck(
  resourceType: ResourceType,
  action: ActionType,
  resourceId?: string
) {
  const userStore = useUserStore()
  
  const hasPermission = computed(() => 
    userStore.checkPermission(resourceType, action, resourceId)
  )
  
  const permissionError = computed(() => {
    if (hasPermission.value) return null
    return PermissionUtils.getPermissionErrorMessage(resourceType, action, resourceId)
  })
  
  const debugInfo = computed(() => {
    if (import.meta.env.PROD) return null
    return PermissionUtils.getPermissionDebugInfo(resourceType, action, resourceId)
  })
  
  return {
    hasPermission,
    permissionError,
    debugInfo
  }
}

/**
 * 역할 기반 반응형 체크
 */
export function useRoleCheck(roles: string | string[]) {
  const userStore = useUserStore()
  const rolesList = Array.isArray(roles) ? roles : [roles]
  
  const hasAnyRole = computed(() => 
    rolesList.some(role => userStore.hasRole(role))
  )
  
  const hasAllRoles = computed(() => 
    rolesList.every(role => userStore.hasRole(role))
  )
  
  const missingRoles = computed(() => 
    rolesList.filter(role => !userStore.hasRole(role))
  )
  
  return {
    hasAnyRole,
    hasAllRoles,
    missingRoles
  }
}

/**
 * 권한 변경 감지 및 콜백 실행
 */
export function usePermissionWatcher(
  callback: () => void,
  options?: {
    immediate?: boolean
    deep?: boolean
  }
) {
  const userStore = useUserStore()
  
  // 권한 정보 변경 감지
  const stopWatcher = watch(
    () => [
      userStore.userPermissions,
      userStore.userRoles,
      userStore.lastPermissionUpdate
    ],
    callback,
    {
      immediate: options?.immediate ?? false,
      deep: options?.deep ?? true
    }
  )
  
  onUnmounted(() => {
    stopWatcher()
  })
  
  return {
    stop: stopWatcher
  }
}

/**
 * 권한 기반 네비게이션 필터링
 */
export function usePermissionNavigation() {
  const userStore = useUserStore()
  
  interface NavigationItem {
    name: string
    path: string
    label: string
    icon?: string
    permissions?: Array<{
      resource: ResourceType
      action: ActionType
      resourceId?: string
    }>
    roles?: string[]
    adminOnly?: boolean
    superAdminOnly?: boolean
    children?: NavigationItem[]
  }
  
  const filterNavigation = (items: NavigationItem[]): NavigationItem[] => {
    return items.filter(item => {
      // 슈퍼 관리자 전용 체크
      if (item.superAdminOnly && !userStore.isSuperAdmin) {
        return false
      }
      
      // 관리자 전용 체크
      if (item.adminOnly && !userStore.isAdmin) {
        return false
      }
      
      // 역할 기반 체크
      if (item.roles && item.roles.length > 0) {
        const hasRequiredRole = item.roles.some(role => userStore.hasRole(role))
        if (!hasRequiredRole) {
          return false
        }
      }
      
      // 권한 기반 체크
      if (item.permissions && item.permissions.length > 0) {
        const hasRequiredPermission = item.permissions.every(permission =>
          userStore.checkPermission(
            permission.resource,
            permission.action,
            permission.resourceId
          )
        )
        if (!hasRequiredPermission) {
          return false
        }
      }
      
      // 하위 항목이 있는 경우 재귀적으로 필터링
      if (item.children) {
        item.children = filterNavigation(item.children)
        // 하위 항목이 모두 필터링되면 상위 항목도 제거
        if (item.children.length === 0) {
          return false
        }
      }
      
      return true
    }).map(item => ({
      ...item,
      children: item.children ? filterNavigation(item.children) : undefined
    }))
  }
  
  return {
    filterNavigation
  }
}

/**
 * 권한 기반 동작 실행
 */
export function usePermissionAction() {
  const userStore = useUserStore()
  
  const executeWithPermission = async <T>(
    resourceType: ResourceType,
    action: ActionType,
    callback: () => Promise<T> | T,
    options?: {
      resourceId?: string
      onError?: (message: string) => void
      showError?: boolean
    }
  ): Promise<T | null> => {
    const hasPermission = userStore.checkPermission(
      resourceType,
      action,
      options?.resourceId
    )
    
    if (!hasPermission) {
      const errorMessage = PermissionUtils.getPermissionErrorMessage(
        resourceType,
        action,
        options?.resourceId
      )
      
      if (options?.onError) {
        options.onError(errorMessage)
      } else if (options?.showError !== false) {
        console.warn(errorMessage)
      }
      
      return null
    }
    
    try {
      return await callback()
    } catch (error) {
      console.error('Action execution failed:', error)
      throw error
    }
  }
  
  return {
    executeWithPermission
  }
}

/**
 * 권한 상태 실시간 모니터링
 */
export function usePermissionMonitor() {
  const userStore = useUserStore()
  const lastCheck = ref<Date>(new Date())
  const permissionChanges = ref<string[]>([])
  
  // 권한 변경 이력 추적
  watch(
    () => userStore.userPermissions,
    (newPermissions, oldPermissions) => {
      if (!oldPermissions || !newPermissions) return
      
      const changes: string[] = []
      const newKeys = Object.keys(newPermissions.finalPermissions)
      const oldKeys = Object.keys(oldPermissions.finalPermissions)
      
      // 새로 추가된 권한
      newKeys.forEach(key => {
        if (!oldKeys.includes(key)) {
          changes.push(`Added: ${key}`)
        }
      })
      
      // 제거된 권한
      oldKeys.forEach(key => {
        if (!newKeys.includes(key)) {
          changes.push(`Removed: ${key}`)
        }
      })
      
      // 변경된 권한
      newKeys.forEach(key => {
        if (oldKeys.includes(key)) {
          const newPermission = newPermissions.finalPermissions[key]
          const oldPermission = oldPermissions.finalPermissions[key]
          
          if (newPermission.effect !== oldPermission.effect) {
            changes.push(`Changed: ${key} (${oldPermission.effect} → ${newPermission.effect})`)
          }
        }
      })
      
      if (changes.length > 0) {
        permissionChanges.value = changes
        lastCheck.value = new Date()
        
        // 개발 모드에서 변경 로그 출력
        if (import.meta.env.DEV) {
          console.log('Permission changes detected:', changes)
        }
      }
    },
    { deep: true }
  )
  
  // 권한 변경 이력 초기화
  const clearChanges = () => {
    permissionChanges.value = []
  }
  
  return {
    lastCheck: computed(() => lastCheck.value),
    permissionChanges: computed(() => permissionChanges.value),
    clearChanges
  }
}

export default usePermission