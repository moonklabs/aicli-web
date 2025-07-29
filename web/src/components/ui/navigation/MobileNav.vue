<template>
  <div class="mobile-nav">
    <!-- 햄버거 메뉴 버튼 -->
    <button
      v-if="!isDrawerOpen"
      class="mobile-nav__hamburger"
      @click="toggleDrawer"
      aria-label="메뉴 열기"
    >
      <span class="mobile-nav__hamburger-line"></span>
      <span class="mobile-nav__hamburger-line"></span>
      <span class="mobile-nav__hamburger-line"></span>
    </button>

    <!-- 모바일 네비게이션 드로어 -->
    <Teleport to="body">
      <div
        v-if="isDrawerOpen"
        class="mobile-nav__overlay"
        @click="closeDrawer"
      ></div>

      <nav
        ref="drawerRef"
        :class="[
          'mobile-nav__drawer',
          { 'mobile-nav__drawer--open': isDrawerOpen }
        ]"
        role="navigation"
        aria-label="주 네비게이션"
      >
        <!-- 드로어 헤더 -->
        <div class="mobile-nav__header">
          <div class="mobile-nav__brand">
            <img src="/favicon.ico" alt="AICLI" class="mobile-nav__logo" />
            <span class="mobile-nav__title">AICLI Web</span>
          </div>

          <button
            class="mobile-nav__close"
            @click="closeDrawer"
            aria-label="메뉴 닫기"
          >
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
              <path
                d="M18 6L6 18M6 6l12 12"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
            </svg>
          </button>
        </div>

        <!-- 네비게이션 메뉴 -->
        <div class="mobile-nav__content">
          <ul class="mobile-nav__menu">
            <li
              v-for="item in menuItems"
              :key="item.path"
              class="mobile-nav__menu-item"
            >
              <router-link
                :to="item.path"
                class="mobile-nav__menu-link"
                :class="{ 'mobile-nav__menu-link--active': isActiveRoute(item.path) }"
                @click="handleMenuItemClick"
              >
                <component
                  v-if="item.icon"
                  :is="item.icon"
                  class="mobile-nav__menu-icon"
                />
                <span class="mobile-nav__menu-text">{{ item.label }}</span>
                <span
                  v-if="item.badge"
                  class="mobile-nav__menu-badge"
                >
                  {{ item.badge }}
                </span>
              </router-link>

              <!-- 서브메뉴 -->
              <div
                v-if="item.children && item.children.length > 0"
                class="mobile-nav__submenu"
              >
                <button
                  class="mobile-nav__submenu-toggle"
                  @click="toggleSubmenu(item.path)"
                >
                  <svg
                    :class="[
                      'mobile-nav__submenu-arrow',
                      { 'mobile-nav__submenu-arrow--open': openSubmenus.includes(item.path) }
                    ]"
                    width="16"
                    height="16"
                    viewBox="0 0 16 16"
                    fill="none"
                  >
                    <path
                      d="M6 4l4 4-4 4"
                      stroke="currentColor"
                      stroke-width="2"
                      stroke-linecap="round"
                      stroke-linejoin="round"
                    />
                  </svg>
                </button>

                <ul
                  v-show="openSubmenus.includes(item.path)"
                  class="mobile-nav__submenu-list"
                >
                  <li
                    v-for="child in item.children"
                    :key="child.path"
                    class="mobile-nav__submenu-item"
                  >
                    <router-link
                      :to="child.path"
                      class="mobile-nav__submenu-link"
                      :class="{ 'mobile-nav__submenu-link--active': isActiveRoute(child.path) }"
                      @click="handleMenuItemClick"
                    >
                      {{ child.label }}
                    </router-link>
                  </li>
                </ul>
              </div>
            </li>
          </ul>

          <!-- 사용자 정보 섹션 -->
          <div v-if="user" class="mobile-nav__user">
            <div class="mobile-nav__user-info">
              <div class="mobile-nav__user-avatar">
                <img :src="user.avatar || '/default-avatar.png'" :alt="user.displayName || user.username" />
              </div>
              <div class="mobile-nav__user-details">
                <span class="mobile-nav__user-name">{{ user.displayName || user.username }}</span>
                <span class="mobile-nav__user-email">{{ user.email }}</span>
              </div>
            </div>

            <div class="mobile-nav__user-actions">
              <button
                class="mobile-nav__user-action"
                @click="handleProfile"
              >
                프로필
              </button>
              <button
                class="mobile-nav__user-action mobile-nav__user-action--logout"
                @click="handleLogout"
              >
                로그아웃
              </button>
            </div>
          </div>
        </div>
      </nav>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useTouchGestures } from '@/composables/useTouchGestures'
import { useUserStore } from '@/stores/user'

interface MenuItem {
  path: string
  label: string
  icon?: string
  badge?: string | number
  children?: MenuItem[]
}

interface Props {
  menuItems?: MenuItem[]
  autoClose?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  menuItems: () => [
    { path: '/', label: '홈', icon: 'HomeIcon' },
    { path: '/workspace', label: '워크스페이스', icon: 'WorkspaceIcon' },
    { path: '/terminal', label: '터미널', icon: 'TerminalIcon' },
    { path: '/docker', label: 'Docker', icon: 'DockerIcon' },
    { path: '/profile', label: '프로필', icon: 'UserIcon' },
    { path: '/settings', label: '설정', icon: 'SettingsIcon' },
  ],
  autoClose: true,
})

const emit = defineEmits<{
  'drawer-open': []
  'drawer-close': []
  'menu-item-click': [item: MenuItem]
}>()

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

const user = computed(() => userStore.user)

// 드로어 상태
const isDrawerOpen = ref(false)
const drawerRef = ref<HTMLElement>()
const openSubmenus = ref<string[]>([])

// 터치 제스처 설정
const { on: onGesture } = useTouchGestures(drawerRef, {
  enableSwipe: true,
  enablePan: false,
  enablePinch: false,
  swipeThreshold: 100,
})

// 스와이프로 드로어 닫기
onGesture('swipeleft', () => {
  if (isDrawerOpen.value) {
    closeDrawer()
  }
})

// 드로어 토글
const toggleDrawer = () => {
  if (isDrawerOpen.value) {
    closeDrawer()
  } else {
    openDrawer()
  }
}

// 드로어 열기
const openDrawer = () => {
  isDrawerOpen.value = true
  emit('drawer-open')

  // 포커스 트랩
  nextTick(() => {
    if (drawerRef.value) {
      const firstFocusable = drawerRef.value.querySelector(
        'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])',
      ) as HTMLElement
      firstFocusable?.focus()
    }
  })
}

// 드로어 닫기
const closeDrawer = () => {
  isDrawerOpen.value = false
  emit('drawer-close')
}

// 활성 라우트 확인
const isActiveRoute = (path: string) => {
  return route.path === path || route.path.startsWith(`${path}/`)
}

// 서브메뉴 토글
const toggleSubmenu = (path: string) => {
  const index = openSubmenus.value.indexOf(path)
  if (index > -1) {
    openSubmenus.value.splice(index, 1)
  } else {
    openSubmenus.value.push(path)
  }
}

// 메뉴 아이템 클릭 핸들러
const handleMenuItemClick = (event: Event) => {
  const target = event.target as HTMLElement
  const link = target.closest('a') as HTMLAnchorElement

  if (link && props.autoClose) {
    closeDrawer()
  }

  emit('menu-item-click', props.menuItems.find(item =>
    link?.getAttribute('href')?.includes(item.path),
  ) as MenuItem)
}

// 프로필 페이지로 이동
const handleProfile = () => {
  router.push('/profile')
  if (props.autoClose) {
    closeDrawer()
  }
}

// 로그아웃 핸들러
const handleLogout = async () => {
  try {
    await userStore.logout()
    router.push('/login')
    closeDrawer()
  } catch (error) {
    console.error('로그아웃 실패:', error)
  }
}

// 라우트 변경 시 드로어 자동 닫기
watch(
  () => route.path,
  () => {
    if (isDrawerOpen.value && props.autoClose) {
      closeDrawer()
    }
  },
)

// ESC 키로 드로어 닫기
watch(isDrawerOpen, (isOpen) => {
  const handleEscape = (event: KeyboardEvent) => {
    if (event.key === 'Escape' && isOpen) {
      closeDrawer()
    }
  }

  if (isOpen) {
    document.addEventListener('keydown', handleEscape)
    document.body.style.overflow = 'hidden' // 스크롤 방지
  } else {
    document.removeEventListener('keydown', handleEscape)
    document.body.style.overflow = ''
  }

  return () => {
    document.removeEventListener('keydown', handleEscape)
    document.body.style.overflow = ''
  }
})
</script>

<style lang="scss" scoped>
@use '@/styles/variables' as *;
@use '@/styles/mixins' as *;

.mobile-nav {
  position: relative;
  z-index: $z-sticky;

  // 햄버거 메뉴 버튼
  &__hamburger {
    @include button-base;
    @include flex-column;
    gap: 4px;
    width: 44px;
    height: 44px;
    padding: 8px;
    background: transparent;
    border: none;

    // 터치 친화적 크기
    min-width: 44px;
    min-height: 44px;

    &:hover {
      background: rgba(0, 0, 0, 0.05);

      .dark & {
        background: rgba(255, 255, 255, 0.05);
      }
    }
  }

  &__hamburger-line {
    width: 20px;
    height: 2px;
    background: currentColor;
    border-radius: 1px;
    transition: $transition-base;
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

  // 드로어
  &__drawer {
    position: fixed;
    top: 0;
    left: 0;
    bottom: 0;
    width: 280px;
    max-width: 80vw;
    background: $light-bg-primary;
    border-right: 1px solid map-get($gray-colors, 200);
    transform: translateX(-100%);
    transition: transform 0.3s ease;
    z-index: $z-modal;
    display: flex;
    flex-direction: column;
    box-shadow: $shadow-xl;

    .dark & {
      background: $dark-bg-secondary;
      border-right-color: $dark-bg-tertiary;
    }

    &--open {
      transform: translateX(0);
    }
  }

  // 드로어 헤더
  &__header {
    @include flex-between;
    padding: $spacing-4;
    border-bottom: 1px solid map-get($gray-colors, 200);

    .dark & {
      border-bottom-color: $dark-bg-tertiary;
    }
  }

  &__brand {
    @include flex-center;
    gap: $spacing-3;
  }

  &__logo {
    width: 32px;
    height: 32px;
    border-radius: $border-radius-base;
  }

  &__title {
    font-size: $font-size-lg;
    font-weight: $font-weight-semibold;
    color: $light-text-primary;

    .dark & {
      color: $dark-text-primary;
    }
  }

  &__close {
    @include button-base;
    @include flex-center;
    width: 40px;
    height: 40px;
    padding: 8px;
    background: transparent;
    border: none;
    color: map-get($gray-colors, 600);

    .dark & {
      color: $dark-text-secondary;
    }

    &:hover {
      background: map-get($gray-colors, 100);

      .dark & {
        background: $dark-bg-tertiary;
      }
    }
  }

  // 드로어 콘텐츠
  &__content {
    flex: 1;
    overflow-y: auto;
    @include scrollbar-thin;
  }

  // 메뉴
  &__menu {
    list-style: none;
    padding: $spacing-2 0;
    margin: 0;
  }

  &__menu-item {
    position: relative;
  }

  &__menu-link {
    @include flex-center;
    justify-content: flex-start;
    gap: $spacing-3;
    padding: $spacing-3 $spacing-4;
    color: $light-text-primary;
    text-decoration: none;
    transition: $transition-base;

    // 터치 친화적 높이
    min-height: 48px;

    .dark & {
      color: $dark-text-primary;
    }

    &:hover {
      background: map-get($gray-colors, 50);

      .dark & {
        background: $dark-bg-tertiary;
      }
    }

    &--active {
      background: map-get($primary-colors, 50);
      color: map-get($primary-colors, 700);
      border-right: 3px solid map-get($primary-colors, 500);

      .dark & {
        background: rgba(map-get($primary-colors, 500), 0.1);
        color: map-get($primary-colors, 300);
      }
    }
  }

  &__menu-icon {
    width: 20px;
    height: 20px;
    flex-shrink: 0;
  }

  &__menu-text {
    flex: 1;
    font-size: $font-size-base;
    font-weight: $font-weight-medium;
  }

  &__menu-badge {
    @include status-badge(map-get($primary-colors, 500));
    font-size: $font-size-xs;
    padding: 2px 6px;
    min-width: 18px;
  }

  // 서브메뉴
  &__submenu {
    position: relative;
  }

  &__submenu-toggle {
    @include button-base;
    position: absolute;
    right: $spacing-4;
    top: 50%;
    transform: translateY(-50%);
    width: 32px;
    height: 32px;
    padding: 4px;
    background: transparent;
    border: none;
  }

  &__submenu-arrow {
    transition: transform 0.2s ease;

    &--open {
      transform: rotate(90deg);
    }
  }

  &__submenu-list {
    list-style: none;
    padding: 0;
    margin: 0;
    background: map-get($gray-colors, 25);

    .dark & {
      background: darken($dark-bg-secondary, 5%);
    }
  }

  &__submenu-item {
    border-left: 2px solid map-get($gray-colors, 200);
    margin-left: $spacing-8;

    .dark & {
      border-left-color: $dark-bg-tertiary;
    }
  }

  &__submenu-link {
    display: block;
    padding: $spacing-2 $spacing-4;
    color: $light-text-secondary;
    text-decoration: none;
    font-size: $font-size-sm;
    transition: $transition-base;

    // 터치 친화적 높이
    min-height: 40px;

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
    }
  }

  // 사용자 정보
  &__user {
    border-top: 1px solid map-get($gray-colors, 200);
    padding: $spacing-4;

    .dark & {
      border-top-color: $dark-bg-tertiary;
    }
  }

  &__user-info {
    @include flex-center;
    gap: $spacing-3;
    margin-bottom: $spacing-3;
  }

  &__user-avatar {
    width: 48px;
    height: 48px;
    border-radius: $border-radius-full;
    overflow: hidden;

    img {
      width: 100%;
      height: 100%;
      object-fit: cover;
    }
  }

  &__user-details {
    @include flex-column;
    gap: 2px;
    flex: 1;
  }

  &__user-name {
    font-size: $font-size-base;
    font-weight: $font-weight-medium;
    color: $light-text-primary;

    .dark & {
      color: $dark-text-primary;
    }
  }

  &__user-email {
    font-size: $font-size-sm;
    color: $light-text-secondary;

    .dark & {
      color: $dark-text-secondary;
    }
  }

  &__user-actions {
    display: flex;
    gap: $spacing-2;
  }

  &__user-action {
    @include button-secondary;
    flex: 1;
    font-size: $font-size-sm;
    padding: $spacing-2 $spacing-3;

    &--logout {
      background: lighten($error, 35%);
      color: darken($error, 10%);

      &:hover:not(:disabled) {
        background: lighten($error, 25%);
      }

      .dark & {
        background: rgba($error, 0.2);
        color: lighten($error, 20%);

        &:hover:not(:disabled) {
          background: rgba($error, 0.3);
        }
      }
    }
  }
}

// 모바일 전용 표시
@include mobile {
  .mobile-nav {
    display: block;
  }
}

@include tablet-up {
  .mobile-nav {
    display: none;
  }
}
</style>