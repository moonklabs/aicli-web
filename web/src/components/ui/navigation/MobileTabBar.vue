<template>
  <nav
    class="mobile-tab-bar"
    role="navigation"
    aria-label="하단 네비게이션"
  >
    <div class="mobile-tab-bar__container">
      <router-link
        v-for="item in items"
        :key="item.path"
        :to="item.path"
        class="mobile-tab-bar__item"
        :class="{
          'mobile-tab-bar__item--active': isActive(item.path),
          'mobile-tab-bar__item--disabled': item.disabled
        }"
        :aria-label="item.label"
        @click="handleItemClick(item)"
      >
        <!-- 아이콘 -->
        <div class="mobile-tab-bar__icon-wrapper">
          <Icon
            v-if="item.icon"
            :name="item.icon"
            class="mobile-tab-bar__icon"
          />
          <!-- 배지 -->
          <span
            v-if="item.badge"
            class="mobile-tab-bar__badge"
            :aria-label="`${item.badge}개의 알림`"
          >
            {{ item.badge }}
          </span>
        </div>

        <!-- 라벨 -->
        <span class="mobile-tab-bar__label">{{ item.label }}</span>

        <!-- 활성 인디케이터 -->
        <div
          v-if="isActive(item.path)"
          class="mobile-tab-bar__indicator"
          aria-hidden="true"
        ></div>
      </router-link>
    </div>
  </nav>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import Icon from '@/components/common/Icon.vue'

interface TabItem {
  path: string
  label: string
  icon?: string
  badge?: string | number
  disabled?: boolean
}

interface Props {
  items: TabItem[]
  currentRoute?: string
}

const props = withDefaults(defineProps<Props>(), {
  currentRoute: '',
})

const emit = defineEmits<{
  'item-click': [item: TabItem]
  'active-change': [path: string]
}>()

const route = useRoute()

// 활성 상태 확인
const isActive = (path: string) => {
  const currentPath = props.currentRoute || route.path
  return currentPath === path || (path !== '/' && currentPath.startsWith(path))
}

// 아이템 클릭 핸들러
const handleItemClick = (item: TabItem) => {
  if (item.disabled) return

  emit('item-click', item)

  if (!isActive(item.path)) {
    emit('active-change', item.path)
  }
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
  z-index: $z-sticky;
  background: $light-bg-primary;
  border-top: 1px solid map-get($gray-colors, 200);
  @include safe-area-padding(padding, bottom);

  .dark & {
    background: $dark-bg-primary;
    border-top-color: $dark-bg-tertiary;
  }

  &__container {
    display: flex;
    height: 60px;
    max-width: 500px;
    margin: 0 auto;
  }

  &__item {
    @include flex-column-center;
    @include touch-target(44px);
    flex: 1;
    padding: $spacing-2 $spacing-1;
    color: $light-text-secondary;
    text-decoration: none;
    transition: $transition-fast;
    position: relative;
    min-width: 0;

    .dark & {
      color: $dark-text-secondary;
    }

    &--active {
      color: map-get($primary-colors, 600);

      .dark & {
        color: map-get($primary-colors, 400);
      }

      .mobile-tab-bar__icon {
        transform: scale(1.1);
      }

      .mobile-tab-bar__label {
        font-weight: $font-weight-semibold;
      }
    }

    &--disabled {
      opacity: 0.4;
      cursor: not-allowed;
    }
  }

  &__icon-wrapper {
    position: relative;
    @include flex-center;
    margin-bottom: 2px;
  }

  &__icon {
    width: 24px;
    height: 24px;
    transition: transform $transition-fast;
  }

  &__badge {
    position: absolute;
    top: -6px;
    right: -8px;
    min-width: 16px;
    height: 16px;
    padding: 0 4px;
    background: $error;
    color: white;
    font-size: 10px;
    font-weight: $font-weight-bold;
    line-height: 16px;
    text-align: center;
    border-radius: $border-radius-full;
    border: 2px solid $light-bg-primary;

    .dark & {
      border-color: $dark-bg-primary;
    }
  }

  &__label {
    font-size: 11px;
    line-height: 1.2;
    text-align: center;
    @include text-clamp(1);
  }

  &__indicator {
    position: absolute;
    top: 4px;
    left: 50%;
    transform: translateX(-50%);
    width: 4px;
    height: 4px;
    background: map-get($primary-colors, 600);
    border-radius: $border-radius-full;

    .dark & {
      background: map-get($primary-colors, 400);
    }
  }
}

@include tablet-up {
  .mobile-tab-bar {
    display: none;
  }
}
</style>