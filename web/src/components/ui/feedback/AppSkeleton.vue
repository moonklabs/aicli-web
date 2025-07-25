<template>
  <div
    :class="[
      'app-skeleton',
      {
        'app-skeleton--animated': animated,
        'app-skeleton--text': text,
        'app-skeleton--round': round,
        'app-skeleton--circle': circle
      }
    ]"
    :style="skeletonStyle"
    :aria-label="ariaLabel"
    role="img"
    aria-live="polite"
    v-bind="$attrs"
  >
    <template v-if="repeat > 1">
      <div
        v-for="index in repeat"
        :key="index"
        class="app-skeleton__item"
        :class="{
          'app-skeleton__item--last': index === repeat
        }"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, type CSSProperties } from 'vue';

interface Props {
  width?: string | number;
  height?: string | number;
  text?: boolean;
  round?: boolean;
  circle?: boolean;
  repeat?: number;
  animated?: boolean;
  ariaLabel?: string;
}

const props = withDefaults(defineProps<Props>(), {
  repeat: 1,
  animated: true,
  ariaLabel: '콘텐츠 로딩 중'
});

const skeletonStyle = computed((): CSSProperties => {
  const style: CSSProperties = {};
  
  // 너비 설정
  if (props.width !== undefined) {
    style.width = typeof props.width === 'number' ? `${props.width}px` : props.width;
  }
  
  // 높이 설정
  if (props.height !== undefined) {
    style.height = typeof props.height === 'number' ? `${props.height}px` : props.height;
  }
  
  // 텍스트 스켈레톤의 기본 설정
  if (props.text) {
    if (!props.height) {
      style.height = '1.2em';
    }
    if (!props.width) {
      style.width = '100%';
    }
  }
  
  // 원형 스켈레톤 설정
  if (props.circle) {
    const size = props.width || props.height || '40px';
    style.width = typeof size === 'number' ? `${size}px` : size;
    style.height = style.width;
  }
  
  return style;
});
</script>

<style lang="scss" scoped>
.app-skeleton {
  @apply relative overflow-hidden;
  background: linear-gradient(
    90deg,
    var(--gray-200) 25%,
    var(--gray-100) 50%,
    var(--gray-200) 75%
  );
  background-size: 200% 100%;
  
  // 다크 모드
  [data-theme='dark'] & {
    background: linear-gradient(
      90deg,
      var(--gray-700) 25%,
      var(--gray-600) 50%,
      var(--gray-700) 75%
    );
  }
  
  &--animated {
    animation: skeleton-loading 1.5s ease-in-out infinite;
  }
  
  &--text {
    @apply rounded-sm;
    
    &:not(:last-child) {
      @apply mb-2;
    }
    
    // 텍스트 라인 변형
    &:nth-child(odd) {
      width: 95%;
    }
    
    &:nth-child(even) {
      width: 80%;
    }
    
    &:last-child {
      width: 60%;
    }
  }
  
  &--round {
    @apply rounded-lg;
  }
  
  &--circle {
    @apply rounded-full;
  }
  
  &__item {
    @apply w-full;
    height: inherit;
    background: inherit;
    border-radius: inherit;
    
    &:not(.app-skeleton__item--last) {
      @apply mb-2;
    }
  }
  
  // 다중 스켈레톤일 때의 스타일
  &:has(.app-skeleton__item) {
    @apply space-y-2;
    width: auto;
    height: auto;
    background: none;
    
    .app-skeleton__item {
      background: linear-gradient(
        90deg,
        var(--gray-200) 25%,
        var(--gray-100) 50%,
        var(--gray-200) 75%
      );
      background-size: 200% 100%;
      
      [data-theme='dark'] & {
        background: linear-gradient(
          90deg,
          var(--gray-700) 25%,
          var(--gray-600) 50%,
          var(--gray-700) 75%
        );
      }
      
      &.app-skeleton--animated {
        animation: skeleton-loading 1.5s ease-in-out infinite;
      }
    }
  }
}

// 애니메이션
@keyframes skeleton-loading {
  0% {
    background-position: -200% 0;
  }
  100% {
    background-position: 200% 0;
  }
}

// 접근성: 애니메이션 감소 설정
@media (prefers-reduced-motion: reduce) {
  .app-skeleton--animated {
    animation: none;
  }
}

// 유틸리티 클래스들
.skeleton-text {
  @apply app-skeleton app-skeleton--text app-skeleton--animated;
}

.skeleton-avatar {
  @apply app-skeleton app-skeleton--circle app-skeleton--animated;
  width: 40px;
  height: 40px;
}

.skeleton-button {
  @apply app-skeleton app-skeleton--round app-skeleton--animated;
  width: 80px;
  height: 32px;
}

.skeleton-card {
  @apply app-skeleton app-skeleton--round app-skeleton--animated;
  width: 100%;
  height: 200px;
}

// 프리셋 레이아웃들
.skeleton-post {
  @apply space-y-4;
  
  .skeleton-header {
    @apply flex items-center space-x-3;
    
    .skeleton-avatar {
      @apply flex-shrink-0;
    }
    
    .skeleton-info {
      @apply flex-1 space-y-2;
      
      .skeleton-title {
        height: 16px;
        width: 150px;
      }
      
      .skeleton-subtitle {
        height: 14px;
        width: 100px;
      }
    }
  }
  
  .skeleton-content {
    @apply space-y-2;
    
    .skeleton-line {
      height: 14px;
      
      &:nth-child(1) { width: 100%; }
      &:nth-child(2) { width: 95%; }
      &:nth-child(3) { width: 85%; }
      &:nth-child(4) { width: 70%; }
    }
  }
  
  .skeleton-footer {
    @apply flex justify-between items-center;
    
    .skeleton-actions {
      @apply flex space-x-2;
      
      .skeleton-action {
        width: 60px;
        height: 28px;
      }
    }
    
    .skeleton-meta {
      width: 80px;
      height: 12px;
    }
  }
}

.skeleton-table {
  @apply space-y-2;
  
  .skeleton-header {
    @apply flex space-x-4 pb-2 border-b border-gray-200;
    
    .skeleton-header-cell {
      height: 16px;
      
      &:nth-child(1) { width: 120px; }
      &:nth-child(2) { width: 80px; }
      &:nth-child(3) { width: 100px; }
      &:nth-child(4) { width: 60px; }
    }
  }
  
  .skeleton-row {
    @apply flex space-x-4 py-3;
    
    .skeleton-cell {
      height: 14px;
      
      &:nth-child(1) { width: 120px; }
      &:nth-child(2) { width: 80px; }
      &:nth-child(3) { width: 100px; }
      &:nth-child(4) { width: 60px; }
    }
  }
}
</style>