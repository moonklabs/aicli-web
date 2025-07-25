<template>
  <button
    :class="[
      'app-button',
      sizeClasses,
      variantClasses,
      typeClasses,
      {
        'app-button--loading': loading,
        'app-button--disabled': disabled || loading,
        'app-button--block': block,
        'app-button--round': round,
        'app-button--circle': circle,
        'app-button--icon-only': iconOnly
      }
    ]"
    :disabled="disabled || loading"
    :type="htmlType"
    :aria-label="ariaLabel"
    :aria-describedby="ariaDescribedby"
    :aria-expanded="ariaExpanded"
    :aria-pressed="ariaPressed"
    :tabindex="disabled ? -1 : tabindex"
    v-bind="$attrs"
    @click="handleClick"
    @keydown="handleKeydown"
    @focus="handleFocus"
    @blur="handleBlur"
  >
    <!-- 로딩 스피너 -->
    <span v-if="loading" class="app-button__loading">
      <AppSpinner :size="spinnerSize" :variant="spinnerVariant" />
    </span>

    <!-- 아이콘 -->
    <span v-if="$slots.icon && !loading" class="app-button__icon">
      <slot name="icon" />
    </span>

    <!-- 텍스트 내용 -->
    <span
      v-if="$slots.default && !iconOnly"
      class="app-button__content"
      :class="{ 'app-button__content--hidden': loading }"
    >
      <slot />
    </span>

    <!-- 접미사 아이콘 -->
    <span v-if="$slots.suffix && !loading" class="app-button__suffix">
      <slot name="suffix" />
    </span>
  </button>
</template>

<script setup lang="ts">
import { computed, useSlots } from 'vue'
import type { ButtonProps, ColorVariant, Size } from '@/types/ui'
import AppSpinner from '../feedback/AppSpinner.vue'

interface Props extends ButtonProps {
  htmlType?: 'button' | 'submit' | 'reset';
  ariaLabel?: string;
  ariaDescribedby?: string;
  ariaExpanded?: boolean;
  ariaPressed?: boolean;
  tabindex?: number;
}

const props = withDefaults(defineProps<Props>(), {
  type: 'default',
  size: 'medium',
  variant: 'solid',
  htmlType: 'button',
  disabled: false,
  loading: false,
  block: false,
  round: false,
  circle: false,
  tabindex: 0,
})

const emit = defineEmits<{
  click: [event: Event];
  focus: [event: FocusEvent];
  blur: [event: FocusEvent];
  keydown: [event: KeyboardEvent];
}>()

const slots = useSlots()

// 아이콘만 있는 버튼인지 확인
const iconOnly = computed(() => {
  return (slots.icon && !slots.default) || props.circle
})

// 사이즈별 클래스
const sizeClasses = computed(() => ({
  'app-button--small': props.size === 'small',
  'app-button--medium': props.size === 'medium',
  'app-button--large': props.size === 'large',
}))

// 변형별 클래스
const variantClasses = computed(() => ({
  'app-button--solid': props.variant === 'solid',
  'app-button--outline': props.variant === 'outline',
  'app-button--ghost': props.variant === 'ghost',
  'app-button--text': props.variant === 'text',
}))

// 타입별 클래스
const typeClasses = computed(() => ({
  'app-button--default': props.type === 'default',
  'app-button--primary': props.type === 'primary',
  'app-button--success': props.type === 'success',
  'app-button--warning': props.type === 'warning',
  'app-button--error': props.type === 'error',
  'app-button--info': props.type === 'info',
}))

// 스피너 크기 계산
const spinnerSize = computed((): Size => {
  const sizeMap: Record<Size, Size> = {
    small: 'small',
    medium: 'small',
    large: 'medium',
  }
  return sizeMap[props.size]
})

// 스피너 색상 계산
const spinnerVariant = computed((): ColorVariant => {
  if (props.variant === 'solid') {
    return props.type === 'default' ? 'default' : 'default'
  }
  return props.type as ColorVariant
})

// 이벤트 핸들러
const handleClick = (event: Event) => {
  if (props.disabled || props.loading) {
    event.preventDefault()
    return
  }

  emit('click', event)
  props.onClick?.(event)
}

const handleKeydown = (event: KeyboardEvent) => {
  emit('keydown', event)

  // Enter 또는 Space 키로 버튼 활성화
  if (event.key === 'Enter' || event.key === ' ') {
    event.preventDefault()
    if (!props.disabled && !props.loading) {
      handleClick(event)
    }
  }
}

const handleFocus = (event: FocusEvent) => {
  emit('focus', event)
}

const handleBlur = (event: FocusEvent) => {
  emit('blur', event)
}
</script>

<style lang="scss" scoped>
.app-button {
  @apply relative inline-flex items-center justify-center;
  @apply font-medium leading-none select-none cursor-pointer;
  @apply border border-solid rounded-md transition-all duration-200;
  @apply focus:outline-none focus-visible:ring-2 focus-visible:ring-offset-2;

  // 기본 상태
  &:active {
    transform: translateY(1px);
  }

  // 비활성화 상태
  &--disabled {
    @apply opacity-50 cursor-not-allowed pointer-events-none;
  }

  // 로딩 상태
  &--loading {
    @apply pointer-events-none;

    .app-button__content--hidden {
      @apply opacity-0;
    }
  }

  // 블록 버튼
  &--block {
    @apply w-full;
  }

  // 둥근 버튼
  &--round {
    @apply rounded-full;
  }

  // 원형 버튼
  &--circle {
    @apply rounded-full aspect-square;
  }

  // 아이콘만 있는 버튼
  &--icon-only {
    .app-button__content {
      @apply sr-only;
    }
  }

  // 사이즈 변형
  &--small {
    @apply px-3 py-1.5 text-sm min-h-[32px];

    &.app-button--circle {
      @apply w-8 h-8 p-0;
    }

    .app-button__icon {
      @apply text-sm;
    }
  }

  &--medium {
    @apply px-4 py-2 text-base min-h-[40px];

    &.app-button--circle {
      @apply w-10 h-10 p-0;
    }

    .app-button__icon {
      @apply text-base;
    }
  }

  &--large {
    @apply px-6 py-3 text-lg min-h-[48px];

    &.app-button--circle {
      @apply w-12 h-12 p-0;
    }

    .app-button__icon {
      @apply text-lg;
    }
  }

  // 기본 색상 (Default)
  &--default {
    &.app-button--solid {
      @apply bg-gray-100 border-gray-300 text-gray-900;
      @apply hover:bg-gray-200 hover:border-gray-400;
      @apply active:bg-gray-300 active:border-gray-500;
      @apply focus-visible:ring-gray-500;
    }

    &.app-button--outline {
      @apply bg-transparent border-gray-300 text-gray-700;
      @apply hover:bg-gray-50 hover:border-gray-400;
      @apply active:bg-gray-100 active:border-gray-500;
      @apply focus-visible:ring-gray-500;
    }

    &.app-button--ghost {
      @apply bg-transparent border-transparent text-gray-700;
      @apply hover:bg-gray-100;
      @apply active:bg-gray-200;
      @apply focus-visible:ring-gray-500;
    }

    &.app-button--text {
      @apply bg-transparent border-transparent text-gray-700;
      @apply hover:text-gray-900 hover:bg-transparent;
      @apply active:text-gray-900;
      @apply focus-visible:ring-gray-500;
    }
  }

  // 주요 색상 (Primary)
  &--primary {
    &.app-button--solid {
      @apply bg-blue-600 border-blue-600 text-white;
      @apply hover:bg-blue-700 hover:border-blue-700;
      @apply active:bg-blue-800 active:border-blue-800;
      @apply focus-visible:ring-blue-500;
    }

    &.app-button--outline {
      @apply bg-transparent border-blue-600 text-blue-600;
      @apply hover:bg-blue-50 hover:border-blue-700;
      @apply active:bg-blue-100 active:border-blue-800;
      @apply focus-visible:ring-blue-500;
    }

    &.app-button--ghost {
      @apply bg-transparent border-transparent text-blue-600;
      @apply hover:bg-blue-50;
      @apply active:bg-blue-100;
      @apply focus-visible:ring-blue-500;
    }

    &.app-button--text {
      @apply bg-transparent border-transparent text-blue-600;
      @apply hover:text-blue-700 hover:bg-transparent;
      @apply active:text-blue-800;
      @apply focus-visible:ring-blue-500;
    }
  }

  // 성공 색상 (Success)
  &--success {
    &.app-button--solid {
      @apply bg-green-600 border-green-600 text-white;
      @apply hover:bg-green-700 hover:border-green-700;
      @apply active:bg-green-800 active:border-green-800;
      @apply focus-visible:ring-green-500;
    }

    &.app-button--outline {
      @apply bg-transparent border-green-600 text-green-600;
      @apply hover:bg-green-50 hover:border-green-700;
      @apply active:bg-green-100 active:border-green-800;
      @apply focus-visible:ring-green-500;
    }

    &.app-button--ghost {
      @apply bg-transparent border-transparent text-green-600;
      @apply hover:bg-green-50;
      @apply active:bg-green-100;
      @apply focus-visible:ring-green-500;
    }

    &.app-button--text {
      @apply bg-transparent border-transparent text-green-600;
      @apply hover:text-green-700 hover:bg-transparent;
      @apply active:text-green-800;
      @apply focus-visible:ring-green-500;
    }
  }

  // 경고 색상 (Warning)
  &--warning {
    &.app-button--solid {
      @apply bg-yellow-500 border-yellow-500 text-white;
      @apply hover:bg-yellow-600 hover:border-yellow-600;
      @apply active:bg-yellow-700 active:border-yellow-700;
      @apply focus-visible:ring-yellow-500;
    }

    &.app-button--outline {
      @apply bg-transparent border-yellow-500 text-yellow-600;
      @apply hover:bg-yellow-50 hover:border-yellow-600;
      @apply active:bg-yellow-100 active:border-yellow-700;
      @apply focus-visible:ring-yellow-500;
    }

    &.app-button--ghost {
      @apply bg-transparent border-transparent text-yellow-600;
      @apply hover:bg-yellow-50;
      @apply active:bg-yellow-100;
      @apply focus-visible:ring-yellow-500;
    }

    &.app-button--text {
      @apply bg-transparent border-transparent text-yellow-600;
      @apply hover:text-yellow-700 hover:bg-transparent;
      @apply active:text-yellow-800;
      @apply focus-visible:ring-yellow-500;
    }
  }

  // 오류 색상 (Error)
  &--error {
    &.app-button--solid {
      @apply bg-red-600 border-red-600 text-white;
      @apply hover:bg-red-700 hover:border-red-700;
      @apply active:bg-red-800 active:border-red-800;
      @apply focus-visible:ring-red-500;
    }

    &.app-button--outline {
      @apply bg-transparent border-red-600 text-red-600;
      @apply hover:bg-red-50 hover:border-red-700;
      @apply active:bg-red-100 active:border-red-800;
      @apply focus-visible:ring-red-500;
    }

    &.app-button--ghost {
      @apply bg-transparent border-transparent text-red-600;
      @apply hover:bg-red-50;
      @apply active:bg-red-100;
      @apply focus-visible:ring-red-500;
    }

    &.app-button--text {
      @apply bg-transparent border-transparent text-red-600;
      @apply hover:text-red-700 hover:bg-transparent;
      @apply active:text-red-800;
      @apply focus-visible:ring-red-500;
    }
  }

  // 정보 색상 (Info)
  &--info {
    &.app-button--solid {
      @apply bg-sky-600 border-sky-600 text-white;
      @apply hover:bg-sky-700 hover:border-sky-700;
      @apply active:bg-sky-800 active:border-sky-800;
      @apply focus-visible:ring-sky-500;
    }

    &.app-button--outline {
      @apply bg-transparent border-sky-600 text-sky-600;
      @apply hover:bg-sky-50 hover:border-sky-700;
      @apply active:bg-sky-100 active:border-sky-800;
      @apply focus-visible:ring-sky-500;
    }

    &.app-button--ghost {
      @apply bg-transparent border-transparent text-sky-600;
      @apply hover:bg-sky-50;
      @apply active:bg-sky-100;
      @apply focus-visible:ring-sky-500;
    }

    &.app-button--text {
      @apply bg-transparent border-transparent text-sky-600;
      @apply hover:text-sky-700 hover:bg-transparent;
      @apply active:text-sky-800;
      @apply focus-visible:ring-sky-500;
    }
  }

  // 컴포넌트 요소들
  &__loading {
    @apply absolute inset-0 flex items-center justify-center;
  }

  &__icon {
    @apply flex items-center justify-center;

    &:not(:last-child) {
      @apply mr-2;
    }
  }

  &__content {
    @apply flex items-center justify-center transition-opacity duration-200;
  }

  &__suffix {
    @apply flex items-center justify-center ml-2;
  }
}

// 다크 모드
.dark .app-button {
  &--default {
    &.app-button--solid {
      @apply bg-gray-700 border-gray-600 text-gray-100;
      @apply hover:bg-gray-600 hover:border-gray-500;
      @apply active:bg-gray-500 active:border-gray-400;
    }

    &.app-button--outline {
      @apply border-gray-600 text-gray-300;
      @apply hover:bg-gray-800 hover:border-gray-500;
      @apply active:bg-gray-700 active:border-gray-400;
    }

    &.app-button--ghost {
      @apply text-gray-300;
      @apply hover:bg-gray-800;
      @apply active:bg-gray-700;
    }

    &.app-button--text {
      @apply text-gray-300;
      @apply hover:text-gray-100;
      @apply active:text-gray-100;
    }
  }
}

// 접근성: 고대비 모드
@media (prefers-contrast: high) {
  .app-button {
    @apply border-2;

    &:focus-visible {
      @apply ring-4;
    }
  }
}

// 접근성: 움직임 감소
@media (prefers-reduced-motion: reduce) {
  .app-button {
    @apply transition-none;

    &:active {
      transform: none;
    }

    .app-button__content {
      @apply transition-none;
    }
  }
}
</style>