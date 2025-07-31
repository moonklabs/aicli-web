<template>
  <div class="main-layout">
    <!-- 모바일 네비게이션 -->
    <MobileNav
      v-if="isMobileView"
      :menu-items="navigationItems"
      @drawer-open="handleDrawerOpen"
      @drawer-close="handleDrawerClose"
    />

    <!-- 데스크톱 사이드바 -->
    <aside
      v-if="!isMobileView"
      class="main-layout__sidebar"
      :class="{ 'main-layout__sidebar--collapsed': sidebarCollapsed }"
    >
      <nav class="sidebar-nav" role="navigation" aria-label="주 네비게이션">
        <div class="sidebar-nav__brand">
          <router-link to="/" class="brand-link">
            <img src="/favicon.ico" alt="AICLI" class="brand-logo" />
            <span v-if="!sidebarCollapsed" class="brand-title">AICLI Web</span>
          </router-link>
        </div>

        <ul class="sidebar-nav__menu">
          <li
            v-for="item in navigationItems"
            :key="item.path"
            class="menu-item"
          >
            <router-link
              :to="item.path"
              class="menu-link"
              :class="{ 'menu-link--active': isActiveRoute(item.path) }"
              :title="sidebarCollapsed ? item.label : undefined"
            >
              <Icon v-if="item.icon" :name="item.icon" class="menu-icon" />
              <span v-if="!sidebarCollapsed" class="menu-text">{{ item.label }}</span>
              <span
                v-if="item.badge && !sidebarCollapsed"
                class="menu-badge"
              >
                {{ item.badge }}
              </span>
            </router-link>
          </li>
        </ul>

        <!-- 사이드바 토글 버튼 -->
        <button
          class="sidebar-toggle"
          @click="toggleSidebar"
          :aria-label="sidebarCollapsed ? '사이드바 펼치기' : '사이드바 접기'"
        >
          <Icon :name="sidebarCollapsed ? 'ChevronRight' : 'ChevronLeft'" />
        </button>
      </nav>
    </aside>

    <!-- 메인 콘텐츠 영역 -->
    <div class="main-layout__content">
      <!-- 모바일 헤더 -->
      <header v-if="isMobileView" class="mobile-header">
        <MobileNav />
        <h1 class="mobile-header__title">{{ currentPageTitle }}</h1>
        <div class="mobile-header__actions">
          <slot name="header-actions" />
        </div>
      </header>

      <!-- 콘텐츠 래퍼 -->
      <main
        id="main-content"
        class="content-wrapper"
        role="main"
        :class="{ 'content-wrapper--mobile': isMobileView }"
      >
        <slot />
      </main>
    </div>

    <!-- 모바일 하단 네비게이션 -->
    <MobileTabBar
      v-if="isMobileView"
      :items="bottomNavItems"
      :current-route="$route.path"
    />

    <!-- 백 투 탑 버튼 -->
    <Teleport to="body">
      <button
        v-show="showBackToTop"
        class="back-to-top"
        @click="scrollToTop"
        aria-label="맨 위로 이동"
      >
        <Icon name="ChevronUp" />
      </button>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useMobileOptimization } from '@/composables/useMobileOptimization'
import { useTouchGestures } from '@/composables/useTouchGestures'
import MobileNav from '@/components/ui/navigation/MobileNav.vue'
import MobileTabBar from '@/components/ui/navigation/MobileTabBar.vue'
import Icon from '@/components/common/Icon.vue'

interface NavigationItem {
  path: string
  label: string
  icon?: string
  badge?: string | number
  children?: NavigationItem[]
}

interface Props {
  showSidebar?: boolean
  sidebarCollapsible?: boolean
  enableMobileOptimization?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  showSidebar: true,
  sidebarCollapsible: true,
  enableMobileOptimization: true,
})

const route = useRoute()

// 반응형 브레이크포인트 (Vue 3 API 사용)
const windowWidth = ref(0)

const updateWindowWidth = () => {
  windowWidth.value = window.innerWidth
}

const isMobileView = computed(() => windowWidth.value < 1024)
const isTabletView = computed(() => windowWidth.value >= 768 && windowWidth.value < 1280)

// 모바일 최적화 훅
const mobileOptimization = useMobileOptimization({
  enableVirtualKeyboardOptimization: props.enableMobileOptimization,
  enableImageLazyLoading: props.enableMobileOptimization,
  enableMemoryOptimization: props.enableMobileOptimization,
})

// 사이드바 상태
const sidebarCollapsed = ref(false)
const showBackToTop = ref(false)

// 네비게이션 아이템
const navigationItems: NavigationItem[] = [
  { path: '/', label: '홈', icon: 'Home' },
  { path: '/workspace', label: '워크스페이스', icon: 'FolderOpen' },
  { path: '/terminal', label: '터미널', icon: 'Terminal' },
  { path: '/docker', label: 'Docker', icon: 'Container' },
  { path: '/monitoring', label: '모니터링', icon: 'Activity', badge: 3 },
  { path: '/profile', label: '프로필', icon: 'User' },
  { path: '/settings', label: '설정', icon: 'Settings' },
]

// 하단 네비게이션 아이템 (모바일용)
const bottomNavItems = computed(() => [
  { path: '/', label: '홈', icon: 'Home' },
  { path: '/workspace', label: '워크스페이스', icon: 'FolderOpen' },
  { path: '/terminal', label: '터미널', icon: 'Terminal' },
  { path: '/profile', label: '프로필', icon: 'User' },
])

// 현재 페이지 제목
const currentPageTitle = computed(() => {
  return (route.meta?.title as string) || '페이지'
})

// 활성 라우트 확인
const isActiveRoute = (path: string) => {
  return route.path === path || route.path.startsWith(`${path}/`)
}

// 사이드바 토글
const toggleSidebar = () => {
  sidebarCollapsed.value = !sidebarCollapsed.value
  localStorage.setItem('sidebar-collapsed', String(sidebarCollapsed.value))
}

// 드로어 이벤트 핸들러
const handleDrawerOpen = () => {
  document.body.style.overflow = 'hidden'
}

const handleDrawerClose = () => {
  document.body.style.overflow = ''
}

// 스크롤 관련
let scrollElement: HTMLElement | null = null

const handleScroll = () => {
  if (!scrollElement) return

  showBackToTop.value = scrollElement.scrollTop > 500
}

const scrollToTop = () => {
  if (!scrollElement) return

  scrollElement.scrollTo({
    top: 0,
    behavior: 'smooth',
  })
}

// 터치 제스처 (사이드바 스와이프)
const setupTouchGestures = () => {
  if (!isMobileView.value) return

  const contentElement = document.querySelector('.main-layout__content')
  if (!contentElement) return

  const { on: onGesture } = useTouchGestures(contentElement as HTMLElement, {
    enableSwipe: true,
    swipeThreshold: 50,
  })

  // 우측에서 좌측으로 스와이프시 드로어 열기
  onGesture('swiperight', (event) => {
    if (event.startX < 50) { // 화면 왼쪽 가장자리에서 시작한 스와이프만
      // MobileNav 컴포넌트의 드로어 열기 트리거
      // 이 부분은 MobileNav 컴포넌트와 연결 필요
    }
  })
}

// 키보드 네비게이션
const handleKeydown = (event: KeyboardEvent) => {
  // ESC로 모바일 드로어 닫기는 MobileNav 컴포넌트에서 처리

  // Alt + S로 사이드바 토글 (데스크톱에서만)
  if (event.altKey && event.key === 's' && !isMobileView.value) {
    event.preventDefault()
    toggleSidebar()
  }

  // Alt + T로 맨 위로 이동
  if (event.altKey && event.key === 't') {
    event.preventDefault()
    scrollToTop()
  }
}

// 반응형 사이드바 상태 관리
watch(isMobileView, (mobile) => {
  if (mobile) {
    // 모바일로 전환시 사이드바 접기
    sidebarCollapsed.value = true
  } else {
    // 데스크톱으로 전환시 저장된 상태 복원
    const saved = localStorage.getItem('sidebar-collapsed')
    sidebarCollapsed.value = saved === 'true'
  }
})

// 라이프사이클
onMounted(() => {
  // 창 크기 초기화 및 이벤트 리스너 설정
  updateWindowWidth()
  window.addEventListener('resize', updateWindowWidth)

  // 스크롤 이벤트 설정
  scrollElement = document.querySelector('.content-wrapper')
  if (scrollElement) {
    scrollElement.addEventListener('scroll', handleScroll)
  }

  // 키보드 이벤트 설정
  document.addEventListener('keydown', handleKeydown)

  // 터치 제스처 설정
  setupTouchGestures()

  // 사이드바 상태 복원
  const saved = localStorage.getItem('sidebar-collapsed')
  if (saved && !isMobileView.value) {
    sidebarCollapsed.value = saved === 'true'
  }
})

onUnmounted(() => {
  // 이벤트 리스너 정리
  window.removeEventListener('resize', updateWindowWidth)

  if (scrollElement) {
    scrollElement.removeEventListener('scroll', handleScroll)
  }
  document.removeEventListener('keydown', handleKeydown)
})

// 접근성: 사이드바 상태 변경 알림
watch(sidebarCollapsed, (collapsed) => {
  const message = collapsed ? '사이드바가 접혔습니다' : '사이드바가 펼쳐졌습니다'
  // 스크린 리더에 알림 (AriaLive 사용)
  const announcer = document.createElement('div')
  announcer.setAttribute('aria-live', 'polite')
  announcer.setAttribute('aria-atomic', 'true')
  announcer.className = 'sr-only'
  announcer.textContent = message
  document.body.appendChild(announcer)

  setTimeout(() => {
    document.body.removeChild(announcer)
  }, 1000)
})
</script>

<style lang="scss" scoped>
@use '@/styles/variables' as *;
@use '@/styles/mixins' as *;

.main-layout {
  display: flex;
  width: 100%;
  height: 100vh;
  overflow: hidden;
  background: $light-bg-primary;

  .dark & {
    background: $dark-bg-primary;
  }

  // 모바일 레이아웃
  @include mobile {
    flex-direction: column;
  }

  // 사이드바
  &__sidebar {
    width: $sidebar-width;
    height: 100%;
    background: $light-bg-secondary;
    border-right: 1px solid map-get($gray-colors, 200);
    transition: width $transition-base;
    flex-shrink: 0;
    position: relative;

    .dark & {
      background: $dark-bg-secondary;
      border-right-color: $dark-bg-tertiary;
    }

    &--collapsed {
      width: $sidebar-collapsed-width;

      .sidebar-nav__menu {
        .menu-link {
          justify-content: center;
          padding: $spacing-3;
        }
      }
    }

    @include mobile {
      display: none;
    }
  }

  // 메인 콘텐츠
  &__content {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    min-width: 0; // flex 아이템 축소 허용
  }
}

// 사이드바 네비게이션
.sidebar-nav {
  height: 100%;
  display: flex;
  flex-direction: column;
  padding: $spacing-4 0;

  &__brand {
    padding: 0 $spacing-4 $spacing-4;
    border-bottom: 1px solid map-get($gray-colors, 200);
    margin-bottom: $spacing-4;

    .dark & {
      border-bottom-color: $dark-bg-tertiary;
    }
  }

  &__menu {
    flex: 1;
    list-style: none;
    margin: 0;
    padding: 0 $spacing-2;
    display: flex;
    flex-direction: column;
    gap: $spacing-1;
  }
}

// 브랜드 영역
.brand-link {
  @include flex-center;
  gap: $spacing-3;
  text-decoration: none;
  color: $light-text-primary;

  .dark & {
    color: $dark-text-primary;
  }
}

.brand-logo {
  width: 32px;
  height: 32px;
  border-radius: $border-radius-base;
}

.brand-title {
  font-size: $font-size-lg;
  font-weight: $font-weight-semibold;
}

// 메뉴 아이템
.menu-item {
  width: 100%;
}

.menu-link {
  @include flex-center;
  justify-content: flex-start;
  gap: $spacing-3;
  padding: $spacing-3 $spacing-4;
  border-radius: $border-radius-md;
  text-decoration: none;
  color: $light-text-secondary;
  transition: $transition-base;
  width: 100%;
  position: relative;

  .dark & {
    color: $dark-text-secondary;
  }

  &:hover {
    background: map-get($gray-colors, 100);
    color: $light-text-primary;

    .dark & {
      background: $dark-bg-tertiary;
      color: $dark-text-primary;
    }
  }

  &--active {
    background: map-get($primary-colors, 50);
    color: map-get($primary-colors, 700);

    .dark & {
      background: rgba(map-get($primary-colors, 500), 0.1);
      color: map-get($primary-colors, 300);
    }

    &::before {
      content: '';
      position: absolute;
      left: 0;
      top: 0;
      bottom: 0;
      width: 3px;
      background: map-get($primary-colors, 500);
      border-radius: 0 2px 2px 0;
    }
  }

  &:focus {
    outline: 2px solid map-get($primary-colors, 500);
    outline-offset: 2px;
  }
}

.menu-icon {
  width: 20px;
  height: 20px;
  flex-shrink: 0;
}

.menu-text {
  flex: 1;
  font-weight: $font-weight-medium;
}

.menu-badge {
  @include status-badge(map-get($primary-colors, 500));
  font-size: $font-size-xs;
  padding: 2px 6px;
  min-width: 18px;
}

// 사이드바 토글 버튼
.sidebar-toggle {
  @include button-base;
  position: absolute;
  top: 50%;
  right: -12px;
  transform: translateY(-50%);
  width: 24px;
  height: 24px;
  padding: 4px;
  border-radius: $border-radius-full;
  background: $light-bg-primary;
  border: 1px solid map-get($gray-colors, 200);
  box-shadow: $shadow-sm;
  z-index: $z-sticky;

  .dark & {
    background: $dark-bg-secondary;
    border-color: $dark-bg-tertiary;
  }

  &:hover {
    background: map-get($gray-colors, 50);
    border-color: map-get($primary-colors, 300);

    .dark & {
      background: $dark-bg-tertiary;
    }
  }
}

// 모바일 헤더
.mobile-header {
  @include flex-between;
  @include safe-area-padding(padding, top);
  height: $navbar-height;
  padding: 0 $spacing-4;
  background: $light-bg-primary;
  border-bottom: 1px solid map-get($gray-colors, 200);
  z-index: $z-sticky;

  .dark & {
    background: $dark-bg-primary;
    border-bottom-color: $dark-bg-tertiary;
  }

  &__title {
    font-size: $font-size-lg;
    font-weight: $font-weight-semibold;
    color: $light-text-primary;
    margin: 0;

    .dark & {
      color: $dark-text-primary;
    }
  }

  &__actions {
    @include flex-center;
    gap: $spacing-2;
  }
}

// 콘텐츠 래퍼
.content-wrapper {
  flex: 1;
  overflow: auto;
  padding: $spacing-6;
  @include smooth-scroll;

  &--mobile {
    padding: $spacing-4;
    @include safe-area-padding(padding, bottom);
  }

  // 포커스 스타일
  &:focus {
    outline: none;
  }

  &:focus-visible {
    outline: 2px solid map-get($primary-colors, 500);
    outline-offset: -2px;
  }
}

// 백 투 탑 버튼
.back-to-top {
  @include button-primary;
  @include touch-target(56px);
  position: fixed;
  bottom: $spacing-6;
  right: $spacing-6;
  border-radius: $border-radius-full;
  padding: $spacing-4;
  box-shadow: $shadow-lg;
  z-index: $z-sticky;

  @include mobile {
    bottom: calc($spacing-6 + 60px); // 하단 네비게이션 위에 위치
    @include safe-area-padding(margin, bottom);
  }

  &:hover {
    transform: translateY(-2px);
    box-shadow: $shadow-xl;
  }
}

// 접근성 전용 클래스
.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}

// 반응형 조정
@include tablet {
  .main-layout__sidebar {
    width: 200px; // 태블릿에서 조금 더 좁게

    &--collapsed {
      width: $sidebar-collapsed-width;
    }
  }
}

// 고해상도 디스플레이 최적화
@include retina {
  .brand-logo,
  .menu-icon {
    image-rendering: -webkit-optimize-contrast;
    image-rendering: crisp-edges;
  }
}

// 모션 감소 설정
@include reduce-motion {
  .main-layout__sidebar,
  .menu-link,
  .back-to-top {
    transition: none;
  }
}

// 인쇄 스타일
@media print {
  .main-layout__sidebar,
  .mobile-header,
  .back-to-top {
    display: none !important;
  }

  .content-wrapper {
    padding: 0;
    overflow: visible;
  }
}
</style>