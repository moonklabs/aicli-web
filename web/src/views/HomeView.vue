<script setup lang="ts">
import { ref } from 'vue'
import MainLayout from '@/components/layout/MainLayout.vue'
import TheWelcome from '../components/TheWelcome.vue'
import Icon from '@/components/common/Icon.vue'

// 헤더 액션 함수들
const showNotifications = ref(false)

const handleQuickStart = () => {
  console.log('빠른 시작 버튼 클릭')
}

const handleSearch = () => {
  console.log('검색 버튼 클릭')
}

const handleNotifications = () => {
  showNotifications.value = !showNotifications.value
}
</script>

<template>
  <MainLayout>
    <template #header-actions>
      <!-- 모바일 헤더 액션 버튼들 -->
      <button
        class="header-action-btn"
        @click="handleQuickStart"
        aria-label="빠른 시작"
      >
        <Icon name="Zap" />
      </button>
      <button
        class="header-action-btn"
        @click="handleSearch"
        aria-label="검색"
      >
        <Icon name="Search" />
      </button>
      <button
        class="header-action-btn"
        @click="handleNotifications"
        :aria-label="showNotifications ? '알림 닫기' : '알림 열기'"
      >
        <Icon name="Bell" />
      </button>
    </template>

    <!-- 홈 콘텐츠 -->
    <div class="home-view">
      <TheWelcome />
    </div>
  </MainLayout>
</template>

<style lang="scss" scoped>
@use '@/styles/variables' as *;
@use '@/styles/mixins' as *;

.home-view {
  width: 100%;
  max-width: 1200px;
  margin: 0 auto;

  @include mobile {
    padding: 0; // MainLayout에서 이미 패딩 적용
  }
}

// 헤더 액션 버튼 스타일
.header-action-btn {
  @include button-base;
  @include touch-target(44px);
  padding: $spacing-2;
  background: transparent;
  color: $light-text-primary;
  border: 1px solid map-get($gray-colors, 300);
  border-radius: $border-radius-md;

  .dark & {
    color: $dark-text-primary;
    border-color: $dark-bg-tertiary;
  }

  &:hover {
    background: map-get($gray-colors, 100);
    border-color: map-get($primary-colors, 400);

    .dark & {
      background: $dark-bg-tertiary;
    }
  }

  &:active {
    transform: scale(0.95);
  }

  // 모바일에서 아이콘 크기 조정
  @include mobile {
    padding: $spacing-3;

    svg {
      width: 20px;
      height: 20px;
    }
  }
}
</style>
