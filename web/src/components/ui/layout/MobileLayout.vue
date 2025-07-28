<template>
  <div class="mobile-layout" :class="layoutClasses">
    <!-- 상단 헤더 -->
    <header v-if="showHeader" class="mobile-layout__header" ref="headerRef">
      <div class="mobile-layout__header-content">
        <!-- 왼쪽 액션 -->
        <div class="mobile-layout__header-left">
          <slot name="header-left">
            <button
              v-if="showBackButton"
              class="mobile-layout__back-button"
              @click="handleBack"
              aria-label="뒤로 가기"
            >
              <svg viewBox="0 0 24 24" fill="none">
                <path
                  d="M15 18l-6-6 6-6"
                  stroke="currentColor"
                  stroke-width="2"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
              </svg>
            </button>
            
            <button
              v-else-if="showMenuButton"
              class="mobile-layout__menu-button"
              @click="handleMenuToggle"
              aria-label="메뉴 열기"
            >
              <svg viewBox="0 0 24 24" fill="none">
                <path
                  d="M3 12h18M3 6h18M3 18h18"
                  stroke="currentColor"
                  stroke-width="2"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
              </svg>
            </button>
          </slot>
        </div>

        <!-- 중앙 타이틀 -->
        <div class="mobile-layout__header-center">
          <slot name="header-center">
            <h1 v-if="title" class="mobile-layout__title">{{ title }}</h1>
          </slot>
        </div>

        <!-- 오른쪽 액션 -->
        <div class="mobile-layout__header-right">
          <slot name="header-right">
            <button
              v-if="showSearchButton"
              class="mobile-layout__search-button"
              @click="handleSearch"
              aria-label="검색"
            >
              <svg viewBox="0 0 24 24" fill="none">
                <circle cx="11" cy="11" r="8" stroke="currentColor" stroke-width="2"/>
                <path d="m21 21-4.35-4.35" stroke="currentColor" stroke-width="2"/>
              </svg>
            </button>
            
            <button
              v-if="showOptionsButton"
              class="mobile-layout__options-button"
              @click="handleOptions"
              aria-label="옵션"
            >
              <svg viewBox="0 0 24 24" fill="none">
                <circle cx="12" cy="12" r="1" fill="currentColor"/>
                <circle cx="12" cy="5" r="1" fill="currentColor"/>
                <circle cx="12" cy="19" r="1" fill="currentColor"/>
              </svg>
            </button>
          </slot>
        </div>
      </div>
    </header>

    <!-- 메인 콘텐츠 영역 -->
    <main class="mobile-layout__main" ref="mainRef">
      <!-- 풀 리프레시 인디케이터 -->
      <div
        v-if="pullToRefresh && isPulling"
        class="mobile-layout__pull-indicator"
        :class="{ 'mobile-layout__pull-indicator--active': shouldRefresh }"
      >
        <div class="mobile-layout__pull-icon">
          <svg
            v-if="shouldRefresh"
            class="mobile-layout__pull-spinner"
            viewBox="0 0 24 24"
            fill="none"
          >
            <path
              d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
            />
          </svg>
          <svg v-else viewBox="0 0 24 24" fill="none">
            <path
              d="M12 5v14M19 12l-7 7-7-7"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
          </svg>
        </div>
        <span class="mobile-layout__pull-text">
          {{ shouldRefresh ? '놓아서 새로고침' : '당겨서 새로고침' }}
        </span>
      </div>

      <!-- 스크롤 컨테이너 -->
      <div
        class="mobile-layout__content"
        :style="contentStyle"
        @scroll="handleScroll"
        @touchstart="handleTouchStart"
        @touchmove="handleTouchMove"
        @touchend="handleTouchEnd"
      >
        <slot />
      </div>
    </main>

    <!-- 하단 바 -->
    <footer v-if="showBottomBar" class="mobile-layout__footer">
      <slot name="bottom-bar">
        <!-- 기본 탭 바 -->
        <MobileTabBar v-if="tabItems.length > 0" :tabs="tabItems" />
      </slot>
    </footer>

    <!-- 플로팅 액션 버튼 -->
    <div v-if="$slots.fab || showFab" class="mobile-layout__fab">
      <slot name="fab">
        <button
          class="mobile-layout__fab-button"
          @click="handleFab"
          :aria-label="fabLabel"
        >
          <component :is="fabIcon" />
        </button>
      </slot>
    </div>

    <!-- 오버레이 -->
    <div
      v-if="showOverlay"
      class="mobile-layout__overlay"
      @click="handleOverlayClick"
    ></div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch, onMounted, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import MobileTabBar from '../navigation/MobileTabBar.vue'
import type { TabItem } from '../navigation'

interface Props {
  title?: string
  showHeader?: boolean
  showBackButton?: boolean
  showMenuButton?: boolean
  showSearchButton?: boolean
  showOptionsButton?: boolean
  showBottomBar?: boolean
  showFab?: boolean
  fabIcon?: any
  fabLabel?: string
  tabItems?: TabItem[]
  headerFixed?: boolean
  headerTransparent?: boolean
  pullToRefresh?: boolean
  showOverlay?: boolean
  safeArea?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  showHeader: true,
  showBackButton: false,
  showMenuButton: true,
  showSearchButton: false,
  showOptionsButton: false,
  showBottomBar: false,
  showFab: false,
  fabLabel: '액션',
  tabItems: () => [],
  headerFixed: true,
  headerTransparent: false,
  pullToRefresh: false,
  showOverlay: false,
  safeArea: true,
})

const emit = defineEmits<{
  back: []
  'menu-toggle': []
  search: []
  options: []
  fab: []
  'overlay-click': []
  refresh: []
  scroll: [scrollTop: number]
}>()

const router = useRouter()

// 레이아웃 상태
const headerRef = ref<HTMLElement>()
const mainRef = ref<HTMLElement>()
const scrollTop = ref(0)
const headerHeight = ref(0)

// Pull to refresh 상태
const isPulling = ref(false)
const pullDistance = ref(0)
const shouldRefresh = ref(false)
const isRefreshing = ref(false)
const touchStartY = ref(0)

// 레이아웃 클래스
const layoutClasses = computed(() => ({
  'mobile-layout--header-fixed': props.headerFixed,
  'mobile-layout--header-transparent': props.headerTransparent,
  'mobile-layout--safe-area': props.safeArea,
  'mobile-layout--with-bottom-bar': props.showBottomBar,
  'mobile-layout--pulling': isPulling.value,
}))

// 콘텐츠 스타일
const contentStyle = computed(() => {
  const style: Record<string, string> = {}
  
  if (props.pullToRefresh && isPulling.value) {
    style.transform = `translateY(${pullDistance.value}px)`
  }
  
  return style
})

// 이벤트 핸들러
const handleBack = () => {
  emit('back')
  router.back()
}

const handleMenuToggle = () => {
  emit('menu-toggle')
}

const handleSearch = () => {
  emit('search')
}

const handleOptions = () => {
  emit('options')
}

const handleFab = () => {
  emit('fab')
}

const handleOverlayClick = () => {
  emit('overlay-click')
}

const handleScroll = (event: Event) => {
  const target = event.target as HTMLElement
  scrollTop.value = target.scrollTop
  emit('scroll', scrollTop.value)
}

// Pull to refresh 구현
const handleTouchStart = (event: TouchEvent) => {
  if (!props.pullToRefresh || scrollTop.value > 0) return
  
  touchStartY.value = event.touches[0].clientY
}

const handleTouchMove = (event: TouchEvent) => {
  if (!props.pullToRefresh || scrollTop.value > 0 || isRefreshing.value) return
  
  const currentY = event.touches[0].clientY
  const diff = currentY - touchStartY.value
  
  if (diff > 0) {
    event.preventDefault()
    isPulling.value = true
    pullDistance.value = Math.min(diff * 0.5, 100)
    shouldRefresh.value = pullDistance.value >= 60
  }
}

const handleTouchEnd = () => {
  if (!props.pullToRefresh || !isPulling.value) return
  
  if (shouldRefresh.value && !isRefreshing.value) {
    isRefreshing.value = true
    emit('refresh')
    
    // 새로고침 완료 후 상태 초기화
    setTimeout(() => {
      resetPullToRefresh()
    }, 1000)
  } else {
    resetPullToRefresh()
  }
}

const resetPullToRefresh = () => {
  isPulling.value = false
  pullDistance.value = 0
  shouldRefresh.value = false
  isRefreshing.value = false
}

// 헤더 높이 계산
const updateHeaderHeight = () => {
  if (headerRef.value) {
    headerHeight.value = headerRef.value.offsetHeight
  }
}

// 스크롤 위치에 따른 헤더 투명도 조절
const updateHeaderOpacity = () => {
  if (!props.headerTransparent || !headerRef.value) return
  
  const opacity = Math.min(scrollTop.value / 100, 1)
  headerRef.value.style.backgroundColor = `rgba(255, 255, 255, ${opacity})`
}

// 생명주기
onMounted(() => {
  updateHeaderHeight()
  
  // 리사이즈 이벤트 리스너
  const handleResize = () => {
    updateHeaderHeight()
  }
  
  window.addEventListener('resize', handleResize)
  
  onBeforeUnmount(() => {
    window.removeEventListener('resize', handleResize)
  })
})

// 스크롤 변화 감지
watch(scrollTop, () => {
  updateHeaderOpacity()
})

// 공개 메서드
const scrollToTop = () => {
  const content = mainRef.value?.querySelector('.mobile-layout__content')
  if (content) {
    content.scrollTo({ top: 0, behavior: 'smooth' })
  }
}

const scrollToBottom = () => {
  const content = mainRef.value?.querySelector('.mobile-layout__content')
  if (content) {
    content.scrollTo({ top: content.scrollHeight, behavior: 'smooth' })
  }
}

defineExpose({
  scrollToTop,
  scrollToBottom,
  resetPullToRefresh,
})
</script>

<style lang="scss" scoped>
@use '@/styles/variables' as *;
@use '@/styles/mixins' as *;

.mobile-layout {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background: $light-bg-primary;
  overflow: hidden;
  
  .dark & {
    background: $dark-bg-primary;
  }

  &--safe-area {
    @include safe-area-padding(padding, all);
  }

  &--header-fixed {
    .mobile-layout__header {
      position: fixed;
      top: 0;
      left: 0;
      right: 0;
      z-index: $z-sticky;
    }
    
    .mobile-layout__main {
      padding-top: $navbar-height;
    }
  }

  &--header-transparent {
    .mobile-layout__header {
      background: transparent;
      backdrop-filter: blur(10px);
      transition: background-color 0.3s ease;
    }
  }

  &--with-bottom-bar {
    .mobile-layout__main {
      padding-bottom: 60px;
    }
  }

  &--pulling {
    .mobile-layout__content {
      transition: transform 0.2s ease;
    }
  }

  // 헤더
  &__header {
    background: $light-bg-primary;
    border-bottom: 1px solid map-get($gray-colors, 200);
    z-index: $z-sticky;
    
    .dark & {
      background: $dark-bg-secondary;
      border-bottom-color: $dark-bg-tertiary;
    }
  }

  &__header-content {
    @include flex-between;
    align-items: center;
    height: $navbar-height;
    padding: 0 $spacing-4;
  }

  &__header-left,
  &__header-right {
    display: flex;
    align-items: center;
    gap: $spacing-2;
    min-width: 60px;
  }

  &__header-right {
    justify-content: flex-end;
  }

  &__header-center {
    flex: 1;
    text-align: center;
  }

  &__title {
    font-size: $font-size-lg;
    font-weight: $font-weight-semibold;
    color: $light-text-primary;
    margin: 0;
    @include text-ellipsis;
    
    .dark & {
      color: $dark-text-primary;
    }
  }

  &__back-button,
  &__menu-button,
  &__search-button,
  &__options-button {
    @include touch-target(44px);
    @include touch-feedback;
    @include flex-center;
    background: none;
    border: none;
    color: $light-text-primary;
    cursor: pointer;
    border-radius: $border-radius-lg;
    
    .dark & {
      color: $dark-text-primary;
    }
    
    svg {
      width: 24px;
      height: 24px;
    }
    
    &:hover {
      @include no-touch {
        background: rgba(0, 0, 0, 0.05);
        
        .dark & {
          background: rgba(255, 255, 255, 0.05);
        }
      }
    }
  }

  // 메인 콘텐츠
  &__main {
    flex: 1;
    position: relative;
    overflow: hidden;
  }

  &__content {
    height: 100%;
    overflow-y: auto;
    @include smooth-scroll;
    @include scrollbar-thin;
  }

  // Pull to refresh
  &__pull-indicator {
    position: absolute;
    top: -60px;
    left: 0;
    right: 0;
    height: 60px;
    @include flex-column-center;
    gap: $spacing-2;
    background: $light-bg-primary;
    z-index: 10;
    
    .dark & {
      background: $dark-bg-primary;
    }
    
    &--active {
      .mobile-layout__pull-icon svg {
        animation: spin 1s linear infinite;
      }
    }
  }

  &__pull-icon {
    width: 24px;
    height: 24px;
    color: map-get($primary-colors, 500);
    
    .dark & {
      color: map-get($primary-colors, 400);
    }
  }

  &__pull-text {
    font-size: $font-size-sm;
    color: $light-text-secondary;
    
    .dark & {
      color: $dark-text-secondary;
    }
  }

  // 하단 바
  &__footer {
    position: fixed;
    bottom: 0;
    left: 0;
    right: 0;
    z-index: $z-fixed;
    @include safe-area-padding(padding, bottom);
  }

  // 플로팅 액션 버튼
  &__fab {
    position: fixed;
    bottom: calc(80px + env(safe-area-inset-bottom));
    right: $spacing-4;
    z-index: $z-fixed;
  }

  &__fab-button {
    @include touch-target(56px);
    @include touch-feedback;
    @include flex-center;
    width: 56px;
    height: 56px;
    background: map-get($primary-colors, 500);
    color: white;
    border: none;
    border-radius: $border-radius-full;
    box-shadow: $shadow-lg;
    cursor: pointer;
    
    svg {
      width: 24px;
      height: 24px;
    }
    
    &:hover {
      @include no-touch {
        transform: scale(1.05);
        box-shadow: $shadow-xl;
      }
    }
  }

  // 오버레이
  &__overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.5);
    z-index: $z-modal-backdrop;
    backdrop-filter: blur(4px);
  }
}

// 애니메이션
@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

// 모바일에서만 표시
@include tablet-up {
  .mobile-layout {
    display: none;
  }
}

// 접근성 개선
@include reduce-motion {
  .mobile-layout {
    &__content {
      scroll-behavior: auto;
    }
    
    &__fab-button:hover {
      transform: none;
    }
    
    &__pull-indicator--active .mobile-layout__pull-icon svg {
      animation: none;
    }
  }
}
</style>