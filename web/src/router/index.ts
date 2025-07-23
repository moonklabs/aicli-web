import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '@/stores/user'

// 라우트 컴포넌트들을 lazy-load로 import
const DashboardView = () => import('@/views/DashboardView.vue')
const WorkspaceView = () => import('@/views/WorkspaceView.vue')
const WorkspaceDetailView = () => import('@/views/WorkspaceDetailView.vue')
const TerminalView = () => import('@/views/TerminalView.vue')
const DockerView = () => import('@/views/DockerView.vue')
const TerminalTest = () => import('@/views/TerminalTest.vue')
const LoginView = () => import('@/views/LoginView.vue')
const NotFoundView = () => import('@/views/NotFoundView.vue')

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
      path: '/workspaces',
      name: 'workspaces',
      component: WorkspaceView,
      meta: {
        requiresAuth: true,
        title: '워크스페이스',
      },
    },
    {
      path: '/workspace/:id',
      name: 'workspace-detail',
      component: WorkspaceDetailView,
      meta: {
        requiresAuth: true,
        title: '워크스페이스 상세',
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

// 라우터 가드 설정
router.beforeEach(async (to, _from, next) => {
  const userStore = useUserStore()

  // 페이지 타이틀 설정
  if (to.meta.title) {
    document.title = `${to.meta.title} - AICLI Web`
  } else {
    document.title = 'AICLI Web'
  }

  // 인증이 필요한 페이지 확인
  const requiresAuth = to.meta.requiresAuth !== false

  if (requiresAuth && !userStore.isAuthenticated) {
    // 인증이 필요하지만 로그인되지 않은 경우
    next({ name: 'login', query: { redirect: to.fullPath } })
  } else if (to.name === 'login' && userStore.isAuthenticated) {
    // 이미 로그인된 상태에서 로그인 페이지 접근 시 대시보드로 리다이렉트
    next({ name: 'dashboard' })
  } else {
    // 정상적인 라우팅
    next()
  }
})

// 라우터 에러 핸들링
router.onError((error) => {
  console.error('Router error:', error)
})

export default router
