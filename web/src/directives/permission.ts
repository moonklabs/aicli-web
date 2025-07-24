import type { App, DirectiveBinding, VNode } from 'vue'
import type { ResourceType, ActionType } from '@/types/api'
import { PermissionUtils } from '@/utils/permission'

/**
 * v-permission 디렉티브 바인딩 값 타입
 */
export interface PermissionDirectiveValue {
  resource: ResourceType
  action: ActionType
  resourceId?: string
  fallback?: 'hide' | 'disable' | 'show-message'
  message?: string
}

/**
 * v-permission 디렉티브 구현
 */
export const permissionDirective = {
  name: 'permission',
  
  mounted(el: HTMLElement, binding: DirectiveBinding<PermissionDirectiveValue | string>) {
    checkPermissionAndApply(el, binding)
  },
  
  updated(el: HTMLElement, binding: DirectiveBinding<PermissionDirectiveValue | string>) {
    checkPermissionAndApply(el, binding)
  }
}

/**
 * 권한 체크 및 요소 처리
 */
function checkPermissionAndApply(
  el: HTMLElement, 
  binding: DirectiveBinding<PermissionDirectiveValue | string>
) {
  let config: PermissionDirectiveValue
  
  // 바인딩 값 파싱
  if (typeof binding.value === 'string') {
    // 간단한 문자열 형태: "workspace:read" 또는 "workspace:read:resource-id"
    const parts = binding.value.split(':')
    if (parts.length < 2) {
      console.error('v-permission: Invalid permission format. Expected "resource:action" or "resource:action:resourceId"')
      return
    }
    
    config = {
      resource: parts[0] as ResourceType,
      action: parts[1] as ActionType,
      resourceId: parts[2],
      fallback: 'hide'
    }
  } else {
    config = binding.value
  }
  
  // 권한 체크
  const hasPermission = PermissionUtils.hasPermission(
    config.resource,
    config.action,
    config.resourceId
  )
  
  // 원본 요소 상태 저장 (처음 마운트될 때만)
  if (!el.dataset.originalDisplay) {
    el.dataset.originalDisplay = el.style.display || ''
    el.dataset.originalDisabled = (el as any).disabled || 'false'
    el.dataset.originalTitle = el.title || ''
  }
  
  if (hasPermission) {
    // 권한이 있는 경우 - 원래 상태로 복원
    restoreElement(el)
  } else {
    // 권한이 없는 경우 - 폴백 동작 적용
    applyFallback(el, config)
  }
}

/**
 * 요소를 원래 상태로 복원
 */
function restoreElement(el: HTMLElement) {
  // display 복원
  if (el.dataset.originalDisplay !== undefined) {
    el.style.display = el.dataset.originalDisplay === 'none' ? '' : el.dataset.originalDisplay
  }
  
  // disabled 복원
  if (el.dataset.originalDisabled !== undefined) {
    (el as any).disabled = el.dataset.originalDisabled === 'true'
  }
  
  // title 복원
  if (el.dataset.originalTitle !== undefined) {
    el.title = el.dataset.originalTitle
  }
  
  // 권한 관련 클래스 제거
  el.classList.remove('permission-denied', 'permission-disabled')
  
  // 권한 메시지 제거
  const messageEl = el.querySelector('.permission-message')
  if (messageEl) {
    messageEl.remove()
  }
}

/**
 * 폴백 동작 적용
 */
function applyFallback(el: HTMLElement, config: PermissionDirectiveValue) {
  const fallback = config.fallback || 'hide'
  
  switch (fallback) {
    case 'hide':
      el.style.display = 'none'
      break
      
    case 'disable':
      if ('disabled' in el) {
        (el as any).disabled = true
      }
      el.classList.add('permission-disabled')
      // 툴팁 추가
      el.title = config.message || PermissionUtils.getPermissionErrorMessage(
        config.resource,
        config.action,
        config.resourceId
      )
      break
      
    case 'show-message':
      el.classList.add('permission-denied')
      // 메시지 요소 추가
      const existingMessage = el.querySelector('.permission-message')
      if (!existingMessage) {
        const messageEl = document.createElement('div')
        messageEl.className = 'permission-message'
        messageEl.textContent = config.message || PermissionUtils.getPermissionErrorMessage(
          config.resource,
          config.action,
          config.resourceId
        )
        el.appendChild(messageEl)
      }
      break
  }
}

/**
 * v-role 디렉티브 - 역할 기반 표시/숨김
 */
export const roleDirective = {
  name: 'role',
  
  mounted(el: HTMLElement, binding: DirectiveBinding<string | string[]>) {
    checkRoleAndApply(el, binding)
  },
  
  updated(el: HTMLElement, binding: DirectiveBinding<string | string[]>) {
    checkRoleAndApply(el, binding)
  }
}

/**
 * 역할 체크 및 요소 처리
 */
function checkRoleAndApply(
  el: HTMLElement, 
  binding: DirectiveBinding<string | string[]>
) {
  const roles = Array.isArray(binding.value) ? binding.value : [binding.value]
  
  // 하나라도 매칭되는 역할이 있는지 확인 (OR 조건)
  const hasRole = roles.some(role => PermissionUtils.hasRole(role))
  
  // 원본 상태 저장
  if (!el.dataset.originalDisplay) {
    el.dataset.originalDisplay = el.style.display || ''
  }
  
  if (hasRole) {
    // 역할이 있는 경우
    el.style.display = el.dataset.originalDisplay === 'none' ? '' : el.dataset.originalDisplay
  } else {
    // 역할이 없는 경우
    el.style.display = 'none'
  }
}

/**
 * v-admin 디렉티브 - 관리자 전용 표시/숨김
 */
export const adminDirective = {
  name: 'admin',
  
  mounted(el: HTMLElement, binding: DirectiveBinding<boolean>) {
    checkAdminAndApply(el, binding)
  },
  
  updated(el: HTMLElement, binding: DirectiveBinding<boolean>) {
    checkAdminAndApply(el, binding)
  }
}

/**
 * 관리자 권한 체크 및 요소 처리
 */
function checkAdminAndApply(
  el: HTMLElement, 
  binding: DirectiveBinding<boolean>
) {
  const requireSuperAdmin = binding.value === true // true면 슈퍼 관리자만, false면 일반 관리자도 포함
  
  const hasAdminPermission = requireSuperAdmin 
    ? PermissionUtils.isSuperAdmin()
    : PermissionUtils.isAdmin()
  
  // 원본 상태 저장
  if (!el.dataset.originalDisplay) {
    el.dataset.originalDisplay = el.style.display || ''
  }
  
  if (hasAdminPermission) {
    // 관리자 권한이 있는 경우
    el.style.display = el.dataset.originalDisplay === 'none' ? '' : el.dataset.originalDisplay
  } else {
    // 관리자 권한이 없는 경우
    el.style.display = 'none'
  }
}

/**
 * 모든 권한 디렉티브를 Vue 앱에 등록
 */
export function registerPermissionDirectives(app: App) {
  app.directive('permission', permissionDirective)
  app.directive('role', roleDirective)
  app.directive('admin', adminDirective)
}

/**
 * 개별 디렉티브 내보내기
 */
export { permissionDirective as vPermission }
export { roleDirective as vRole }
export { adminDirective as vAdmin }

/**
 * 권한 디렉티브용 CSS 스타일 (전역 스타일에 추가 필요)
 */
export const permissionDirectiveStyles = `
.permission-disabled {
  opacity: 0.5;
  cursor: not-allowed;
  pointer-events: none;
}

.permission-denied {
  position: relative;
  opacity: 0.6;
}

.permission-message {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  background: rgba(0, 0, 0, 0.8);
  color: white;
  padding: 8px 12px;
  border-radius: 4px;
  font-size: 12px;
  white-space: nowrap;
  z-index: 1000;
  pointer-events: none;
}

.permission-message::before {
  content: '';
  position: absolute;
  top: -4px;
  left: 50%;
  transform: translateX(-50%);
  width: 0;
  height: 0;
  border-left: 4px solid transparent;
  border-right: 4px solid transparent;
  border-bottom: 4px solid rgba(0, 0, 0, 0.8);
}
`