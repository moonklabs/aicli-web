<template>
  <nav class="mobile-tab-bar" role="navigation" aria-label="하단 탭 네비게이션">
    <div class="mobile-tab-bar__container">
      <router-link
        v-for="tab in tabs"
        :key="tab.path"
        :to="tab.path"
        class="mobile-tab-bar__tab"
        :class="{ 'mobile-tab-bar__tab--active': isActiveTab(tab.path) }"
        :aria-label="tab.label"
      >
        <div class="mobile-tab-bar__icon-wrapper">
          <component
            v-if="tab.icon"
            :is="tab.icon"
            class="mobile-tab-bar__icon"
          />
          <div
            v-if="tab.badge"
            class="mobile-tab-bar__badge"
            :class="`mobile-tab-bar__badge--${tab.badgeType || 'primary'}`"
          >
            {{ tab.badge }}
          </div>
        </div>
        
        <span class="mobile-tab-bar__label">{{ tab.label }}</span>
        
        <!-- 활성 상태 인디케이터 -->
        <div
          v-if="isActiveTab(tab.path)"
          class="mobile-tab-bar__indicator"
        ></div>
      </router-link>
    </div>
    
    <!-- 플로팅 액션 버튼 (선택사항) -->
    <button
      v-if="showFab"
      class="mobile-tab-bar__fab"
      :class="{ 'mobile-tab-bar__fab--expanded': fabExpanded }"
      @click="handleFabClick"
      :aria-label="fabLabel"
    >
      <component
        :is="fabIcon"
        class="mobile-tab-bar__fab-icon"
        :class="{ 'mobile-tab-bar__fab-icon--rotated': fabExpanded }"
      />
    </button>
  </nav>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'

interface TabItem {
  path: string
  label: string
  icon?: string
  badge?: string | number
  badgeType?: 'primary' | 'success' | 'warning' | 'error' | 'info'
}

interface Props {
  tabs?: TabItem[]
  showFab?: boolean
  fabIcon?: string
  fabLabel?: string
}

const props = withDefaults(defineProps<Props>(), {
  tabs: () => [
    { path: '/', label: '홈', icon: 'HomeIcon' },
    { path: '/workspace', label: '워크스페이스', icon: 'WorkspaceIcon' },
    { path: '/terminal', label: '터미널', icon: 'TerminalIcon' },
    { path: '/profile', label: '프로필', icon: 'UserIcon' },
  ],
  showFab: false,
  fabIcon: 'PlusIcon',
  fabLabel: '새로 만들기',
})

const emit = defineEmits<{
  'tab-change': [path: string]
  'fab-click': []
}>()

const route = useRoute()
const fabExpanded = ref(false)

// 활성 탭 확인
const isActiveTab = (path: string) => {
  if (path === '/') {
    return route.path === '/'
  }
  return route.path.startsWith(path)
}

// FAB 클릭 핸들러
const handleFabClick = () => {
  fabExpanded.value = !fabExpanded.value
  emit('fab-click')
}
</script>

<style lang="scss" scoped>
@use '@/styles/variables' as *;
@use '@/styles/mixins' as *;

.mobile-tab-bar {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  z-index: $z-fixed;
  background: $light-bg-primary;
  border-top: 1px solid map-get($gray-colors, 200);
  backdrop-filter: blur(10px);
  
  .dark & {
    background: rgba($dark-bg-secondary, 0.95);
    border-top-color: $dark-bg-tertiary;
  }
  
  // 안전 영역 처리 (iPhone X 이상)
  padding-bottom: env(safe-area-inset-bottom);

  &__container {
    display: flex;
    justify-content: space-around;
    align-items: center;
    height: 60px;
    padding: 0 $spacing-2;
  }

  &__tab {
    @include flex-column-center;
    gap: 2px;
    flex: 1;
    max-width: 80px;
    padding: $spacing-2;
    color: $light-text-secondary;
    text-decoration: none;
    transition: $transition-base;
    position: relative;
    border-radius: $border-radius-lg;
    
    // 터치 친화적 크기
    min-height: 48px;
    min-width: 48px;
    
    .dark & {
      color: $dark-text-secondary;
    }
    
    &:hover {
      background: map-get($gray-colors, 50);
      
      .dark & {
        background: rgba(255, 255, 255, 0.05);
      }
    }
    
    &--active {
      color: map-get($primary-colors, 600);
      
      .dark & {
        color: map-get($primary-colors, 400);
      }
      
      .mobile-tab-bar__icon {
        transform: scale(1.1);
      }
    }
  }

  &__icon-wrapper {
    position: relative;
    @include flex-center;
  }

  &__icon {
    width: 24px;
    height: 24px;
    transition: transform 0.2s ease;
  }

  &__badge {
    position: absolute;
    top: -6px;
    right: -6px;
    min-width: 16px;
    height: 16px;
    padding: 0 4px;
    border-radius: $border-radius-full;
    font-size: 10px;
    font-weight: $font-weight-bold;
    line-height: 16px;
    text-align: center;
    
    &--primary {
      background: map-get($primary-colors, 500);
      color: white;
    }
    
    &--success {
      background: $success;
      color: white;
    }
    
    &--warning {
      background: $warning;
      color: white;
    }
    
    &--error {
      background: $error;
      color: white;
    }
    
    &--info {
      background: $info;
      color: white;
    }
  }

  &__label {
    font-size: 10px;
    font-weight: $font-weight-medium;
    line-height: 1;
    @include text-ellipsis;
    max-width: 100%;
  }

  &__indicator {
    position: absolute;
    top: 4px;
    left: 50%;
    transform: translateX(-50%);
    width: 4px;
    height: 4px;
    background: map-get($primary-colors, 500);
    border-radius: $border-radius-full;
    
    .dark & {
      background: map-get($primary-colors, 400);
    }
  }

  // 플로팅 액션 버튼
  &__fab {
    position: absolute;
    right: $spacing-4;
    bottom: calc(60px + $spacing-4 + env(safe-area-inset-bottom));
    width: 56px;
    height: 56px;
    border-radius: $border-radius-full;
    background: map-get($primary-colors, 500);
    color: white;
    border: none;
    box-shadow: $shadow-lg;
    @include flex-center;
    transition: all 0.3s ease;
    cursor: pointer;
    
    &:hover {
      transform: scale(1.05);
      box-shadow: $shadow-xl;
    }
    
    &:active {
      transform: scale(0.95);
    }
    
    &--expanded {
      transform: scale(1.1) rotate(45deg);
    }
  }

  &__fab-icon {
    width: 24px;
    height: 24px;
    transition: transform 0.3s ease;
    
    &--rotated {
      transform: rotate(45deg);
    }
  }
}

// 진동 피드백 (지원하는 기기에서)
.mobile-tab-bar__tab:active {
  animation: tap-feedback 0.1s ease;
}

@keyframes tap-feedback {
  0% { transform: scale(1); }
  50% { transform: scale(0.95); }
  100% { transform: scale(1); }
}

// 모바일에서만 표시
@include tablet-up {
  .mobile-tab-bar {
    display: none;
  }
}

// 스크롤 시 숨김 효과 (선택사항)
.mobile-tab-bar--hidden {
  transform: translateY(100%);
}

// 알림 애니메이션
@keyframes badge-pulse {
  0% { transform: scale(1); }
  50% { transform: scale(1.2); }
  100% { transform: scale(1); }
}

.mobile-tab-bar__badge {
  animation: badge-pulse 2s infinite;
}
</style>