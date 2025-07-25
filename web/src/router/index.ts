import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { PermissionUtils } from '@/utils/permission'
import type { ActionType, ResourceType } from '@/types/api'

// 라우트 컴포넌트들을 lazy-load로 import
const DashboardView = () => import('@/views/DashboardView.vue')
const WorkspaceView = () => import('@/views/WorkspaceView.vue')
const WorkspaceDetailView = () => import('@/views/WorkspaceDetailView.vue')
const TerminalView = () => import('@/views/TerminalView.vue')
const DockerView = () => import('@/views/DockerView.vue')
const SessionManagementView = () => import('@/views/SessionManagementView.vue')
const SecurityDashboardView = () => import('@/views/SecurityDashboardView.vue')
const ProfileEditView = () => import('@/views/ProfileEditView.vue')
const TerminalTest = () => import('@/views/TerminalTest.vue')
const LoginView = () => import('@/views/LoginView.vue')
const OAuthCallbackView = () => import('@/views/OAuthCallbackView.vue')
const NotFoundView = () => import('@/views/NotFoundView.vue')
const ForbiddenView = () => import('@/views/ForbiddenView.vue')

// 라우트 메타 인터페이스 확장
declare module 'vue-router' {
  interface RouteMeta {
    requiresAuth?: boolean
    title?: string
    roles?: string[]
    permissions?: Array<{
      resource: ResourceType
      action: ActionType
      resourceId?: string
    }>
    adminOnly?: boolean
    superAdminOnly?: boolean
  }
}

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'dashboard',
      component: DashboardView,
      meta: {
        requiresAuth: true,
        title: '대시보드',
        permissions: [
          { resource: 'system', action: 'read' },
        ],
      },
    },
    {
      path: '/login',
      name: 'login',
      component: LoginView,
      meta: {
        requiresAuth: false,
        title: '로그인',
      },
    },
    {
      path: '/auth/callback',
      name: 'oauth-callback',
      component: OAuthCallbackView,
      meta: {
        requiresAuth: false,
        title: 'OAuth 로그인 처리',
      },
    },
    {
      path: '/workspaces',
      name: 'workspaces',
      component: WorkspaceView,
      meta: {
        requiresAuth: true,
        title: '워크스페이스',
        permissions: [
          { resource: 'workspace', action: 'read' },
        ],
      },
    },
    {
      path: '/workspace/:id',
      name: 'workspace-detail',
      component: WorkspaceDetailView,
      meta: {
        requiresAuth: true,
        title: '워크스페이스 상세',
        permissions: [
          { resource: 'workspace', action: 'read' },
        ],
      },
      props: true,
    },
    {
      path: '/terminal/:sessionId?',
      name: 'terminal',
      component: TerminalView,
      meta: {
        requiresAuth: true,
        title: '터미널',
        permissions: [
          { resource: 'session', action: 'execute' },
        ],
      },
      props: true,
    },
    {
      path: '/docker',
      name: 'docker',
      component: DockerView,
      meta: {
        requiresAuth: true,
        title: 'Docker 관리',
        adminOnly: true,
        permissions: [
          { resource: 'system', action: 'manage' },
        ],
      },
    },
    {
      path: '/profile',
      name: 'profile',
      component: ProfileEditView,
      meta: {
        requiresAuth: true,
        title: '프로파일 설정',
        permissions: [
          { resource: 'user', action: 'read' },
        ],
      },
    },
    {
      path: '/sessions',
      name: 'session-management',
      component: SessionManagementView,
      meta: {
        requiresAuth: true,
        title: '세션 관리',
        permissions: [
          { resource: 'user', action: 'read' },
        ],
      },
    },
    {
      path: '/security',
      name: 'security-dashboard',
      component: SecurityDashboardView,
      meta: {
        requiresAuth: true,
        title: '보안 대시보드',
        permissions: [
          { resource: 'user', action: 'read' },
        ],
      },
    },
    {
      path: '/terminal-test',
      name: 'terminal-test',
      component: TerminalTest,
      meta: {
        requiresAuth: false,
        title: '터미널 테스트',
      },
    },
    // 403 Forbidden 페이지
    {
      path: '/forbidden',
      name: 'forbidden',
      component: ForbiddenView,
      meta: {
        requiresAuth: false,
        title: '접근 권한 없음',
      },
    },
    // 리다이렉트 라우트들
    {
      path: '/home',
      redirect: { name: 'dashboard' },
    },
    // 404 페이지
    {
      path: '/:pathMatch(.*)*',
      name: 'not-found',
      component: NotFoundView,
      meta: {
        requiresAuth: false,
        title: '페이지를 찾을 수 없음',
      },
    },
  ],
})

// 권한 체크 헬퍼 함수
function checkRoutePermissions(route: any): { allowed: boolean; reason?: string } {
  const { meta } = route

  // 권한 체크가 불필요한 경우 (인증이 필요없는 페이지)
  if (meta.requiresAuth === false) {
    return { allowed: true }
  }

  // 슈퍼 관리자 권한 필요
  if (meta.superAdminOnly && !PermissionUtils.isSuperAdmin()) {
    return {
      allowed: false,
      reason: '슈퍼 관리자 권한이 필요합니다.',
    }
  }

  // 관리자 권한 필요
  if (meta.adminOnly && !PermissionUtils.isAdmin()) {
    return {
      allowed: false,
      reason: '관리자 권한이 필요합니다.',
    }
  }

  // 특정 역할 필요
  if (meta.roles && meta.roles.length > 0) {
    const hasRequiredRole = meta.roles.some((role: string) =>
      PermissionUtils.hasRole(role),
    )
    if (!hasRequiredRole) {
      return {
        allowed: false,
        reason: `다음 역할 중 하나가 필요합니다: ${meta.roles.join(', ')}`,
      }
    }
  }

  // 세부 권한 체크
  if (meta.permissions && meta.permissions.length > 0) {
    for (const permission of meta.permissions) {
      const hasPermission = PermissionUtils.hasPermission(
        permission.resource,
        permission.action,
        permission.resourceId,
      )
      if (!hasPermission) {
        return {
          allowed: false,
          reason: `권한이 없습니다: ${permission.resource}:${permission.action}`,
        }
      }
    }
  }

  return { allowed: true }
}

// 글로벌 네비게이션 가드
router.beforeEach(async (to, from, next) => {
  const userStore = useUserStore()

  // 페이지 타이틀 설정
  if (to.meta.title) {
    document.title = `${to.meta.title} - AICLI Web`
  }

  // 인증이 필요없는 페이지는 통과
  if (to.meta.requiresAuth === false) {
    // 이미 로그인된 상태에서 로그인 페이지로 가려면 대시보드로 리다이렉트
    if (to.name === 'login' && userStore.isAuthenticated) {
      return next({ name: 'dashboard' })
    }
    return next()
  }

  // 인증 체크
  if (!userStore.isAuthenticated) {
    // 로그인 페이지로 리다이렉트 (원래 가려던 페이지 정보 저장)
    const query = to.path !== '/' ? { redirect: to.fullPath } : {}
    return next({
      name: 'login',
      query,
    })
  }

  // 사용자 정보가 없으면 가져오기
  if (!userStore.user) {
    try {
      await userStore.fetchCurrentUser()
    } catch (error) {
      console.error('Failed to fetch user info:', error)
      // 사용자 정보를 가져올 수 없으면 로그아웃 처리
      userStore.clearAuth()
      return next({
        name: 'login',
        query: { redirect: to.fullPath },
      })
    }
  }

  // 권한 체크
  const permissionCheck = checkRoutePermissions(to)
  if (!permissionCheck.allowed) {
    console.warn(`Access denied to ${to.path}: ${permissionCheck.reason}`)
    // 권한이 없으면 403 페이지로 리다이렉트
    return next({
      name: 'forbidden',
      query: {
        from: to.fullPath,
        reason: permissionCheck.reason,
      },
    })
  }

  next()
})

// 네비게이션 후 처리
router.afterEach((to) => {
  // 페이지 스크롤 초기화
  window.scrollTo(0, 0)

  // 페이지 추적 (분석 도구 등)
  if (typeof window !== 'undefined' && window.gtag) {
    window.gtag('config', 'GA_MEASUREMENT_ID', {
      page_path: to.fullPath,
    })
  }
})

export default router