<template>
  <div class="adaptive-layout" :class="layoutClasses">
    <!-- 모바일 레이아웃 -->
    <MobileLayout
      v-if="isMobile"
      :title="currentPageTitle"
      :show-header="true"
      :show-back-button="showBackButton"
      :show-menu-button="true"
      :show-bottom-bar="showBottomNavigation"
      :tab-items="bottomTabItems"
      :pull-to-refresh="enablePullToRefresh"
      :safe-area="true"
      @back="handleBack"
      @menu-toggle="handleMenuToggle"
      @refresh="handleRefresh"
    >
      <!-- 모바일 네비게이션 드로어 -->
      <template #header-left>
        <MobileNav
          :menu-items="navigationItems"
          @drawer-open="handleDrawerOpen"
          @drawer-close="handleDrawerClose"
        />
      </template>

      <!-- 메인 콘텐츠 -->
      <slot />

      <!-- 모바일 하단 네비게이션 -->
      <template #bottom-bar>
        <MobileTabBar
          v-if="showBottomNavigation"
          :tabs="bottomTabItems"
          :active-tab="currentRoute"
          @tab-change="handleTabChange"
        />
      </template>
    </MobileLayout>

    <!-- 데스크톱 레이아웃 -->
    <div v-else class="desktop-layout">
      <!-- 데스크톱 헤더 -->
      <header class="desktop-header">
        <div class="header-content">
          <div class="header-brand">
            <img src="/favicon.ico" alt="AICLI" class="brand-logo" />
            <span class="brand-title">AICLI Web</span>
          </div>
          
          <nav class="desktop-nav">
            <router-link
              v-for="item in navigationItems"
              :key="item.path"
              :to="item.path"
              class="nav-link"
              :class="{ active: isActiveRoute(item.path) }"
            >
              <component v-if="item.icon" :is="item.icon" />
              {{ item.label }}
            </router-link>
          </nav>

          <div class="header-actions">
            <ThemeToggle />
            <UserMenu v-if="user" :user="user" />
          </div>
        </div>
      </header>

      <!-- 데스크톱 메인 콘텐츠 -->
      <main class="desktop-main">
        <slot />
      </main>
    </div>

    <!-- 모바일 워크플로우 매니저 -->
    <MobileWorkflowManager
      v-if="isMobile"
      :auto-detect-usage="true"
      :adapt-to-context="true"
      :learn-from-behavior="true"
      @workflow-change="handleWorkflowChange"
      @action-execute="handleMobileAction"
    />

    <!-- 전역 오버레이들 -->
    <Teleport to="body">
      <!-- 모바일 네비게이션 오버레이 -->
      <div
        v-if="isMobile && isDrawerOpen"
        class="mobile-nav-overlay"
        @click="handleDrawerClose"
      />
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { useTheme } from '@/composables/useTheme'
import { useMobileOptimization } from '@/composables/useMobileOptimization'
import { useOrientationAdaptation } from '@/composables/useOrientationAdaptation'

import MobileLayout from './MobileLayout.vue'
import MobileNav from '../navigation/MobileNav.vue'
import MobileTabBar from '../navigation/MobileTabBar.vue'
import ThemeToggle from '../accessibility/ThemeToggle.vue'
import UserMenu from './UserMenu.vue'
import MobileWorkflowManager from '../mobile/MobileWorkflowManager.vue'
import type { MenuItem, TabItem } from '../navigation'

interface NavigationItem extends MenuItem {
  roles?: string[]
}

interface BottomTabItem extends TabItem {
  id: string
}

interface Props {
  enablePullToRefresh?: boolean
  showBottomNavigation?: boolean
  showBackButton?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  enablePullToRefresh: false,
  showBottomNavigation: true,
  showBackButton: false,
})

const emit = defineEmits<{
  'layout-change': [layout: 'mobile' | 'desktop']
  'navigation-toggle': [isOpen: boolean]
}>()

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const { isDark } = useTheme()
const {
  isMobile,
  isTablet,
  isDesktop,
  orientation,
  setupViewportMeta,
  handleOrientationChange,
} = useMobileOptimization()

// 방향 적응 기능
const {
  orientationState,
  isPortrait,
  isLandscape,
  addOrientationChangeListener,
  removeOrientationChangeListener,
} = useOrientationAdaptation({
  enableAutoRotation: true,
  enableLayoutSwitching: true,
  enableContentReflow: true,
  enableTransitions: true,
  transitionDuration: 300,
  portraitPreferences: {
    navigationPosition: 'bottom',
    contentPadding: 16,
    enableFullHeight: true,
    keyboardBehavior: 'resize',
  },
  landscapePreferences: {
    navigationPosition: 'left',
    enableSidePanel: false,
    contentColumns: 1,
    compactMode: true,
  },
})

// 상태
const isDrawerOpen = ref(false)
const currentBreakpoint = ref<'mobile' | 'tablet' | 'desktop'>('desktop')

// 사용자 정보
const user = computed(() => userStore.user)

// 현재 페이지 제목
const currentPageTitle = computed(() => {
  return (route.meta?.title as string) || 'AICLI Web'
})

// 현재 라우트
const currentRoute = computed(() => route.path)

// 레이아웃 클래스
const layoutClasses = computed(() => ({
  'adaptive-layout--mobile': isMobile.value,
  'adaptive-layout--tablet': isTablet.value,
  'adaptive-layout--desktop': isDesktop.value,
  'adaptive-layout--portrait': isPortrait.value,
  'adaptive-layout--landscape': isLandscape.value,
  'adaptive-layout--dark': isDark.value,
  'adaptive-layout--orientation-changing': orientationState.value.isChanging,
}))

// 네비게이션 아이템
const navigationItems = computed<NavigationItem[]>(() => [
  { path: '/', label: '대시보드', icon: 'HomeIcon' },
  { path: '/workspaces', label: '워크스페이스', icon: 'WorkspaceIcon' },
  { path: '/terminal', label: '터미널', icon: 'TerminalIcon' },
  { path: '/docker', label: 'Docker', icon: 'DockerIcon', roles: ['admin'] },
  { path: '/profile', label: '프로필', icon: 'UserIcon' },
])

// 하단 탭 아이템 (모바일 전용)
const bottomTabItems = computed<BottomTabItem[]>(() => [
  { id: 'dashboard', label: '홈', icon: 'HomeIcon', path: '/' },
  { id: 'workspaces', label: '워크스페이스', icon: 'WorkspaceIcon', path: '/workspaces' },
  { id: 'terminal', label: '터미널', icon: 'TerminalIcon', path: '/terminal' },
  { id: 'profile', label: '프로필', icon: 'UserIcon', path: '/profile' },
])

// 메서드
const handleBack = () => {
  router.back()
}

const handleMenuToggle = () => {
  isDrawerOpen.value = !isDrawerOpen.value
  emit('navigation-toggle', isDrawerOpen.value)
}

const handleDrawerOpen = () => {
  isDrawerOpen.value = true
  emit('navigation-toggle', true)
}

const handleDrawerClose = () => {
  isDrawerOpen.value = false
  emit('navigation-toggle', false)
}

const handleRefresh = () => {
  // 페이지별 새로고침 로직
  window.location.reload()
}

const handleTabChange = (tabId: string) => {
  const tab = bottomTabItems.value.find(t => t.id === tabId)
  if (tab) {
    router.push(tab.path)
  }
}

const isActiveRoute = (path: string) => {
  return route.path === path || route.path.startsWith(`${path}/`)
}

// 모바일 워크플로우 이벤트 핸들러
const handleWorkflowChange = (settings: any) => {
  console.log('Mobile workflow settings changed:', settings)
  // 워크플로우 설정 변경 시 레이아웃 조정
}

const handleMobileAction = (actionId: string) => {
  console.log('Mobile action executed:', actionId)
  
  // 액션별 처리
  switch (actionId) {
    case 'new-workspace':
      router.push('/workspaces/new')
      break
    case 'open-terminal':
      router.push('/terminal')
      break
    case 'search':
      // 검색 모달 열기
      break
    case 'notifications':
      // 알림 패널 열기
      break
    default:
      console.log('Unknown mobile action:', actionId)
  }
}

// 브레이크포인트 변경 감지
const updateBreakpoint = () => {
  if (isMobile.value) {
    currentBreakpoint.value = 'mobile'
  } else if (isTablet.value) {
    currentBreakpoint.value = 'tablet'
  } else {
    currentBreakpoint.value = 'desktop'
  }
  
  emit('layout-change', isMobile.value ? 'mobile' : 'desktop')
}

// 뷰포트 메타태그 설정
const setupMobileViewport = () => {
  setupViewportMeta()
  
  // iOS 관련 메타태그 추가
  const meta = document.createElement('meta')
  meta.name = 'apple-mobile-web-app-capable'
  meta.content = 'yes'
  document.head.appendChild(meta)

  const statusBar = document.createElement('meta')
  statusBar.name = 'apple-mobile-web-app-status-bar-style'
  statusBar.content = 'default'
  document.head.appendChild(statusBar)
}

// 키보드 이벤트 처리
const handleKeyboardEvents = (event: KeyboardEvent) => {
  // 모바일에서 ESC 키로 드로어 닫기
  if (event.key === 'Escape' && isDrawerOpen.value) {
    handleDrawerClose()
  }
}

// 방향 변경 핸들러
const handleAdaptiveOrientationChange = (newOrientation: string, oldOrientation: string) => {
  console.log(`Adaptive layout orientation change: ${oldOrientation} → ${newOrientation}`)
  
  // 방향 변경 시 드로어 닫기
  if (isMobile.value && isDrawerOpen.value) {
    handleDrawerClose()
  }
  
  // 방향별 추가 처리
  if (newOrientation === 'landscape') {
    // 가로 모드에서는 더 넓은 레이아웃 활용
    emit('layout-change', 'desktop')
  } else {
    // 세로 모드에서는 모바일 레이아웃 유지
    emit('layout-change', 'mobile')
  }
}

// 생명주기
onMounted(() => {
  setupMobileViewport()
  updateBreakpoint()
  
  // 이벤트 리스너 등록
  document.addEventListener('keydown', handleKeyboardEvents)
  window.addEventListener('orientationchange', handleOrientationChange)
  
  // 방향 변경 리스너 등록
  addOrientationChangeListener(handleAdaptiveOrientationChange)
})

onBeforeUnmount(() => {
  // 이벤트 리스너 정리
  document.removeEventListener('keydown', handleKeyboardEvents)
  window.removeEventListener('orientationchange', handleOrientationChange)
  
  // 방향 변경 리스너 제거
  removeOrientationChangeListener(handleAdaptiveOrientationChange)
})

// 반응형 변화 감지
watch([isMobile, isTablet, isDesktop], () => {
  updateBreakpoint()
}, { immediate: true })

// 라우트 변경 시 드로어 닫기
watch(route, () => {
  if (isMobile.value && isDrawerOpen.value) {
    handleDrawerClose()
  }
})
</script>

<style lang="scss" scoped>
@use '@/styles/variables' as *;
@use '@/styles/mixins' as *;

.adaptive-layout {
  width: 100%;
  height: 100vh;
  overflow: hidden;
  background: var(--bg-primary);

  // 레이아웃별 스타일
  &--mobile {
    .desktop-layout {
      display: none;
    }
  }

  &--tablet,
  &--desktop {
    .mobile-layout {
      display: none;
    }
  }

  // 다크 모드
  &--dark {
    background: $dark-bg-primary;
  }
}

// 데스크톱 레이아웃
.desktop-layout {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.desktop-header {
  background: $light-bg-primary;
  border-bottom: 1px solid map-get($gray-colors, 200);
  padding: 0 $spacing-6;
  position: sticky;
  top: 0;
  z-index: $z-sticky;

  .dark & {
    background: $dark-bg-secondary;
    border-bottom-color: $dark-bg-tertiary;
  }

  .header-content {
    @include flex-between;
    max-width: 1400px;
    margin: 0 auto;
    height: $navbar-height;
  }
}

.header-brand {
  @include flex-center;
  gap: $spacing-3;

  .brand-logo {
    width: 32px;
    height: 32px;
    border-radius: $border-radius-base;
  }

  .brand-title {
    font-size: $font-size-xl;
    font-weight: $font-weight-semibold;
    color: $light-text-primary;

    .dark & {
      color: $dark-text-primary;
    }
  }
}

.desktop-nav {
  display: flex;
  align-items: center;
  gap: $spacing-6;

  .nav-link {
    @include flex-center;
    gap: $spacing-2;
    padding: $spacing-2 $spacing-3;
    color: $light-text-secondary;
    text-decoration: none;
    border-radius: $border-radius-md;
    font-weight: $font-weight-medium;
    transition: $transition-base;

    .dark & {
      color: $dark-text-secondary;
    }

    &:hover {
      color: $light-text-primary;
      background: map-get($gray-colors, 100);

      .dark & {
        color: $dark-text-primary;
        background: $dark-bg-tertiary;
      }
    }

    &.active {
      color: map-get($primary-colors, 600);
      background: map-get($primary-colors, 50);

      .dark & {
        color: map-get($primary-colors, 400);
        background: rgba(map-get($primary-colors, 500), 0.1);
      }
    }

    svg {
      width: 20px;
      height: 20px;
    }
  }
}

.header-actions {
  @include flex-center;
  gap: $spacing-3;
}

.desktop-main {
  flex: 1;
  overflow: auto;
  @include scrollbar-thin;
}

// 모바일 네비게이션 오버레이
.mobile-nav-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  z-index: $z-modal-backdrop;
  backdrop-filter: blur(4px);
}

// 반응형 조정
@include tablet-up {
  .adaptive-layout--mobile {
    .desktop-layout {
      display: flex;
    }
    
    .mobile-layout {
      display: none;
    }
  }
}

// 접근성 개선
@include reduce-motion {
  .adaptive-layout {
    .nav-link {
      transition: none;
    }
  }
}

// 인쇄 최적화
@media print {
  .desktop-header,
  .mobile-layout {
    display: none !important;
  }

  .desktop-main {
    overflow: visible;
  }
}
</style>