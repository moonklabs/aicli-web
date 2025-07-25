<template>
  <div
    :class="[
      'app-spinner',
      sizeClasses,
      variantClasses,
      {
        'app-spinner--center': center,
        'app-spinner--overlay': overlay
      }
    ]"
    v-bind="$attrs"
  >
    <div class="app-spinner__wrapper">
      <div class="app-spinner__circle" :style="spinnerStyle">
        <svg
          class="app-spinner__svg"
          :width="svgSize"
          :height="svgSize"
          viewBox="0 0 50 50"
          :aria-label="description || '로딩 중'"
          role="img"
          aria-live="polite"
        >
          <circle
            class="app-spinner__track"
            cx="25"
            cy="25"
            r="20"
            fill="none"
            :stroke-width="strokeWidth"
          />
          <circle
            class="app-spinner__fill"
            cx="25"
            cy="25"
            r="20"
            fill="none"
            :stroke-width="strokeWidth"
            stroke-linecap="round"
          />
        </svg>
      </div>
      
      <div
        v-if="description"
        class="app-spinner__description"
        :class="textSizeClasses"
      >
        {{ description }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, type CSSProperties } from 'vue';
import type { Size, ColorVariant } from '@/types/ui';

interface Props {
  size?: Size;
  variant?: ColorVariant;
  description?: string;
  center?: boolean;
  overlay?: boolean;
  color?: string;
  strokeWidth?: number;
}

const props = withDefaults(defineProps<Props>(), {
  size: 'medium',
  variant: 'primary',
  center: false,
  overlay: false,
  strokeWidth: 3
});

// 크기별 설정
const sizeConfig = {
  small: { size: 24, textSize: 'text-sm' },
  medium: { size: 32, textSize: 'text-base' },
  large: { size: 48, textSize: 'text-lg' }
};

const currentSizeConfig = computed(() => sizeConfig[props.size]);

const svgSize = computed(() => currentSizeConfig.value.size);

const sizeClasses = computed(() => ({
  'app-spinner--small': props.size === 'small',
  'app-spinner--medium': props.size === 'medium',
  'app-spinner--large': props.size === 'large'
}));

const textSizeClasses = computed(() => currentSizeConfig.value.textSize);

const variantClasses = computed(() => ({
  'app-spinner--default': props.variant === 'default',
  'app-spinner--primary': props.variant === 'primary',
  'app-spinner--secondary': props.variant === 'secondary',
  'app-spinner--success': props.variant === 'success',
  'app-spinner--warning': props.variant === 'warning',
  'app-spinner--error': props.variant === 'error',
  'app-spinner--info': props.variant === 'info'
}));

const spinnerStyle = computed((): CSSProperties => {
  const style: CSSProperties = {};
  
  if (props.color) {
    style['--spinner-color'] = props.color;
  }
  
  return style;
});
</script>

<style lang="scss" scoped>
.app-spinner {
  @apply inline-flex;
  
  &--center {
    @apply flex items-center justify-center w-full h-full;
  }
  
  &--overlay {
    @apply fixed inset-0 bg-white/80 dark:bg-gray-900/80 backdrop-blur-sm;
    z-index: var(--z-overlay);
    
    .app-spinner__wrapper {
      @apply flex flex-col items-center justify-center h-full;
    }
  }
  
  &__wrapper {
    @apply flex flex-col items-center gap-3;
  }
  
  &__circle {
    @apply relative inline-block;
  }
  
  &__svg {
    @apply block;
    animation: spin 1.5s linear infinite;
  }
  
  &__track {
    stroke: rgb(var(--gray-300) / 0.3);
  }
  
  &__fill {
    stroke: var(--spinner-color, var(--primary-500));
    stroke-dasharray: 90, 150;
    stroke-dashoffset: 0;
    animation: stroke 1.5s ease-in-out infinite;
  }
  
  &__description {
    @apply text-center font-medium;
    color: var(--text-secondary);
  }
  
  // 크기별 스타일
  &--small {
    .app-spinner__wrapper {
      @apply gap-2;
    }
  }
  
  &--large {
    .app-spinner__wrapper {
      @apply gap-4;
    }
  }
  
  // 색상 변형
  &--primary .app-spinner__fill {
    stroke: var(--primary-500);
  }
  
  &--secondary .app-spinner__fill {
    stroke: var(--gray-500);
  }
  
  &--success .app-spinner__fill {
    stroke: var(--success-500);
  }
  
  &--warning .app-spinner__fill {
    stroke: var(--warning-500);
  }
  
  &--error .app-spinner__fill {
    stroke: var(--error-500);
  }
  
  &--info .app-spinner__fill {
    stroke: var(--info-500);
  }
  
  &--default .app-spinner__fill {
    stroke: var(--gray-600);
  }
}

// 애니메이션
@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

@keyframes stroke {
  0% {
    stroke-dasharray: 1, 150;
    stroke-dashoffset: 0;
  }
  50% {
    stroke-dasharray: 90, 150;
    stroke-dashoffset: -35;
  }
  100% {
    stroke-dasharray: 90, 150;
    stroke-dashoffset: -124;
  }
}

// 접근성: 애니메이션 감소 설정
@media (prefers-reduced-motion: reduce) {
  .app-spinner__svg {
    animation: none;
  }
  
  .app-spinner__fill {
    animation: none;
    stroke-dasharray: none;
  }
}
</style>