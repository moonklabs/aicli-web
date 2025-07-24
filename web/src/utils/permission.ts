import type {
  Role,
  Permission,
  UserRole,
  PermissionCheck,
  PermissionCheckResponse,
  UserPermissions,
  ResourceType,
  ActionType,
  PermissionEffect
} from '@/types/api'

/**
 * 권한 체크 유틸리티 클래스
 */
export class PermissionUtils {
  private static permissions: UserPermissions | null = null
  private static roles: Role[] = []
  private static permissionCache = new Map<string, boolean>()
  private static cacheTimeout = 5 * 60 * 1000 // 5분 캐시
  
  /**
   * 사용자 권한 정보 설정
   */
  static setUserPermissions(permissions: UserPermissions) {
    this.permissions = permissions
    this.clearCache()
  }
  
  /**
   * 사용자 역할 정보 설정
   */
  static setUserRoles(roles: Role[]) {
    this.roles = roles
    this.clearCache()
  }
  
  /**
   * 권한 캐시 초기화
   */
  static clearCache() {
    this.permissionCache.clear()
  }
  
  /**
   * 권한 체크 (캐시 사용)
   */
  static hasPermission(
    resourceType: ResourceType,
    action: ActionType,
    resourceId?: string
  ): boolean {
    if (!this.permissions) {
      console.warn('User permissions not loaded')
      return false
    }
    
    const cacheKey = `${resourceType}:${action}:${resourceId || '*'}`
    
    // 캐시에서 확인
    if (this.permissionCache.has(cacheKey)) {
      return this.permissionCache.get(cacheKey)!
    }
    
    const result = this.checkPermission(resourceType, action, resourceId)
    
    // 캐시에 저장
    this.permissionCache.set(cacheKey, result)
    
    // 캐시 자동 삭제 타이머 설정
    setTimeout(() => {
      this.permissionCache.delete(cacheKey)
    }, this.cacheTimeout)
    
    return result
  }
  
  /**
   * 실제 권한 체크 로직
   */
  private static checkPermission(
    resourceType: ResourceType,
    action: ActionType,
    resourceId?: string
  ): boolean {
    if (!this.permissions) return false
    
    const permissionKey = resourceId 
      ? `${resourceType}:${resourceId}:${action}`
      : `${resourceType}:*:${action}`
    
    // 정확한 리소스 ID 매칭 우선
    if (resourceId && this.permissions.finalPermissions[permissionKey]) {
      const permission = this.permissions.finalPermissions[permissionKey]
      return permission.effect === 'allow'
    }
    
    // 와일드카드 매칭
    const wildcardKey = `${resourceType}:*:${action}`
    if (this.permissions.finalPermissions[wildcardKey]) {
      const permission = this.permissions.finalPermissions[wildcardKey]
      return permission.effect === 'allow'
    }
    
    // 관리자 권한 체크 (manage 액션은 모든 하위 액션 포함)
    const manageKey = resourceId 
      ? `${resourceType}:${resourceId}:manage`
      : `${resourceType}:*:manage`
    
    if (this.permissions.finalPermissions[manageKey]) {
      const permission = this.permissions.finalPermissions[manageKey]
      return permission.effect === 'allow'
    }
    
    return false
  }
  
  /**
   * 역할별 권한 체크
   */
  static hasRole(roleName: string): boolean {
    if (!this.permissions) return false
    
    return this.roles.some(role => 
      role.name === roleName && 
      (this.permissions!.directRoles.includes(role.id) ||
       this.permissions!.inheritedRoles.includes(role.id) ||
       this.permissions!.groupRoles.includes(role.id))
    )
  }
  
  /**
   * 여러 권한 중 하나라도 있는지 체크 (OR 조건)
   */
  static hasAnyPermission(checks: Array<{
    resourceType: ResourceType
    action: ActionType
    resourceId?: string
  }>): boolean {
    return checks.some(check => 
      this.hasPermission(check.resourceType, check.action, check.resourceId)
    )
  }
  
  /**
   * 모든 권한이 있는지 체크 (AND 조건)
   */
  static hasAllPermissions(checks: Array<{
    resourceType: ResourceType
    action: ActionType
    resourceId?: string
  }>): boolean {
    return checks.every(check => 
      this.hasPermission(check.resourceType, check.action, check.resourceId)
    )
  }
  
  /**
   * 관리자 권한 체크
   */
  static isAdmin(): boolean {
    return this.hasRole('admin') || this.hasRole('super_admin')
  }
  
  /**
   * 시스템 관리자 권한 체크
   */
  static isSuperAdmin(): boolean {
    return this.hasRole('super_admin')
  }
  
  /**
   * 리소스별 권한 목록 조회
   */
  static getResourcePermissions(resourceType: ResourceType, resourceId?: string): ActionType[] {
    if (!this.permissions) return []
    
    const permissions: ActionType[] = []
    const actions: ActionType[] = ['create', 'read', 'update', 'delete', 'execute', 'manage']
    
    actions.forEach(action => {
      if (this.hasPermission(resourceType, action, resourceId)) {
        permissions.push(action)
      }
    })
    
    return permissions
  }
  
  /**
   * 권한 디버깅 정보 조회
   */
  static getPermissionDebugInfo(
    resourceType: ResourceType,
    action: ActionType,
    resourceId?: string
  ): {
    allowed: boolean
    reason: string
    source?: string
    conditions?: string
  } {
    if (!this.permissions) {
      return {
        allowed: false,
        reason: 'User permissions not loaded'
      }
    }
    
    const permissionKey = resourceId 
      ? `${resourceType}:${resourceId}:${action}`
      : `${resourceType}:*:${action}`
    
    let permission = this.permissions.finalPermissions[permissionKey]
    
    if (!permission && resourceId) {
      // 와일드카드 확인
      const wildcardKey = `${resourceType}:*:${action}`
      permission = this.permissions.finalPermissions[wildcardKey]
    }
    
    if (!permission) {
      // 관리자 권한 확인
      const manageKey = resourceId 
        ? `${resourceType}:${resourceId}:manage`
        : `${resourceType}:*:manage`
      permission = this.permissions.finalPermissions[manageKey]
    }
    
    if (permission) {
      return {
        allowed: permission.effect === 'allow',
        reason: permission.reason,
        source: permission.source,
        conditions: permission.conditions
      }
    }
    
    return {
      allowed: false,
      reason: 'No matching permission found'
    }
  }
  
  /**
   * 권한 에러 메시지 생성
   */
  static getPermissionErrorMessage(
    resourceType: ResourceType,
    action: ActionType,
    resourceId?: string
  ): string {
    const actionMap: Record<ActionType, string> = {
      create: '생성',
      read: '조회',
      update: '수정',
      delete: '삭제',
      execute: '실행',
      manage: '관리'
    }
    
    const resourceMap: Record<ResourceType, string> = {
      workspace: '워크스페이스',
      project: '프로젝트',
      session: '세션',
      task: '작업',
      user: '사용자',
      system: '시스템'
    }
    
    const actionText = actionMap[action] || action
    const resourceText = resourceMap[resourceType] || resourceType
    
    if (resourceId) {
      return `이 ${resourceText}(${resourceId})을(를) ${actionText}할 권한이 없습니다.`
    } else {
      return `${resourceText}을(를) ${actionText}할 권한이 없습니다.`
    }
  }
}

/**
 * 권한 체크 컴포저블 함수들
 */

/**
 * 권한 체크 헬퍼 함수
 */
export const hasPermission = (
  resourceType: ResourceType,
  action: ActionType,
  resourceId?: string
): boolean => {
  return PermissionUtils.hasPermission(resourceType, action, resourceId)
}

/**
 * 역할 체크 헬퍼 함수
 */
export const hasRole = (roleName: string): boolean => {
  return PermissionUtils.hasRole(roleName)
}

/**
 * 관리자 권한 체크 헬퍼 함수
 */
export const isAdmin = (): boolean => {
  return PermissionUtils.isAdmin()
}

/**
 * 권한 에러 메시지 헬퍼 함수
 */
export const getPermissionError = (
  resourceType: ResourceType,
  action: ActionType,
  resourceId?: string
): string => {
  return PermissionUtils.getPermissionErrorMessage(resourceType, action, resourceId)
}

/**
 * 권한 체크 결과와 함께 함수 실행
 */
export const withPermissionCheck = <T>(
  resourceType: ResourceType,
  action: ActionType,
  callback: () => T,
  onError?: (message: string) => void,
  resourceId?: string
): T | undefined => {
  if (hasPermission(resourceType, action, resourceId)) {
    return callback()
  } else {
    const errorMessage = getPermissionError(resourceType, action, resourceId)
    if (onError) {
      onError(errorMessage)
    } else {
      console.warn(errorMessage)
    }
    return undefined
  }
}

export default PermissionUtils