<template>
  <div
    class="mobile-card"
    :class="cardClasses"
    @click="handleClick"
    @touchstart="handleTouchStart"
    @touchend="handleTouchEnd"
  >
    <!-- 카드 헤더 -->
    <header v-if="$slots.header || title || subtitle" class="mobile-card__header">
      <slot name="header">
        <div class="mobile-card__header-content">
          <div class="mobile-card__header-text">
            <h3 v-if="title" class="mobile-card__title">{{ title }}</h3>
            <p v-if="subtitle" class="mobile-card__subtitle">{{ subtitle }}</p>
          </div>

          <div v-if="$slots.actions" class="mobile-card__header-actions">
            <slot name="actions" />
          </div>
        </div>
      </slot>
    </header>

    <!-- 카드 이미지 -->
    <div v-if="$slots.image || image" class="mobile-card__image">
      <slot name="image">
        <img
          v-if="image"
          :src="image"
          :alt="imageAlt || title"
          class="mobile-card__img"
          loading="lazy"
        />
      </slot>
    </div>

    <!-- 카드 콘텐츠 -->
    <div v-if="$slots.default" class="mobile-card__content">
      <slot />
    </div>

    <!-- 카드 푸터 -->
    <footer v-if="$slots.footer" class="mobile-card__footer">
      <slot name="footer" />
    </footer>

    <!-- 리플 효과 -->
    <div
      v-if="ripple && showRipple"
      class="mobile-card__ripple"
      :style="rippleStyle"
    ></div>

    <!-- 로딩 오버레이 -->
    <div v-if="loading" class="mobile-card__loading">
      <AppSpinner size="medium" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import AppSpinner from '../feedback/AppSpinner.vue'

interface Props {
  title?: string
  subtitle?: string
  image?: string
  imageAlt?: string
  variant?: 'default' | 'outlined' | 'elevated' | 'filled'
  size?: 'small' | 'medium' | 'large'
  clickable?: boolean
  disabled?: boolean
  loading?: boolean
  ripple?: boolean
  rounded?: boolean
  flat?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'default',
  size: 'medium',
  clickable: false,
  disabled: false,
  loading: false,
  ripple: true,
  rounded: false,
  flat: false,
})

const emit = defineEmits<{
  click: [event: Event]
  'touch-start': [event: TouchEvent]
  'touch-end': [event: TouchEvent]
}>()

// 리플 효과 상태
const showRipple = ref(false)
const rippleX = ref(0)
const rippleY = ref(0)

// 카드 클래스
const cardClasses = computed(() => ({
  [`mobile-card--${props.variant}`]: true,
  [`mobile-card--${props.size}`]: true,
  'mobile-card--clickable': props.clickable,
  'mobile-card--disabled': props.disabled,
  'mobile-card--loading': props.loading,
  'mobile-card--rounded': props.rounded,
  'mobile-card--flat': props.flat,
}))

// 리플 스타일
const rippleStyle = computed(() => ({
  left: `${rippleX.value}px`,
  top: `${rippleY.value}px`,
}))

// 이벤트 핸들러
const handleClick = (event: Event) => {
  if (props.disabled || props.loading) return

  emit('click', event)
}

const handleTouchStart = (event: TouchEvent) => {
  if (props.disabled || props.loading) return

  emit('touch-start', event)

  // 리플 효과 생성
  if (props.ripple && props.clickable) {
    const rect = (event.currentTarget as HTMLElement).getBoundingClientRect()
    const touch = event.touches[0]

    rippleX.value = touch.clientX - rect.left
    rippleY.value = touch.clientY - rect.top
    showRipple.value = true

    // 햅틱 피드백
    if ('vibrate' in navigator) {
      navigator.vibrate(10)
    }

    setTimeout(() => {
      showRipple.value = false
    }, 600)
  }
}

const handleTouchEnd = (event: TouchEvent) => {
  if (props.disabled || props.loading) return

  emit('touch-end', event)
}
</script>

<style lang="scss" scoped>
@use '@/styles/variables' as *;
@use '@/styles/mixins' as *;

.mobile-card {
  @include card-base;
  position: relative;
  overflow: hidden;
  transition: $transition-base;

  &--clickable {
    @include touch-feedback;
    cursor: pointer;

    &:hover {
      @include hover-lift;
    }
  }

  &--disabled {
    opacity: 0.6;
    cursor: not-allowed;
    pointer-events: none;
  }

  &--loading {
    pointer-events: none;
  }

  &--rounded {
    border-radius: $border-radius-2xl;
  }

  &--flat {
    box-shadow: none;
    border: 1px solid map-get($gray-colors, 200);

    .dark & {
      border-color: $dark-bg-tertiary;
    }
  }

  // 변형 스타일
  &--default {
    // 기본 스타일은 card-base에서 처리
  }

  &--outlined {
    background: transparent;
    border: 2px solid map-get($gray-colors, 200);
    box-shadow: none;

    .dark & {
      border-color: $dark-bg-tertiary;
    }
  }

  &--elevated {
    box-shadow: $shadow-lg;

    &:hover {
      @include no-touch {
        box-shadow: $shadow-xl;
        transform: translateY(-4px);
      }
    }
  }

  &--filled {
    background: map-get($gray-colors, 100);

    .dark & {
      background: lighten($dark-bg-secondary, 5%);
    }
  }

  // 크기 변형
  &--small {
    padding: $spacing-3;

    .mobile-card__header {
      padding-bottom: $spacing-2;
    }

    .mobile-card__title {
      font-size: $font-size-base;
    }

    .mobile-card__subtitle {
      font-size: $font-size-xs;
    }
  }

  &--medium {
    padding: $spacing-4;

    .mobile-card__header {
      padding-bottom: $spacing-3;
    }
  }

  &--large {
    padding: $spacing-6;

    .mobile-card__header {
      padding-bottom: $spacing-4;
    }

    .mobile-card__title {
      font-size: $font-size-xl;
    }

    .mobile-card__subtitle {
      font-size: $font-size-base;
    }
  }

  // 헤더
  &__header {
    border-bottom: 1px solid map-get($gray-colors, 100);
    padding-bottom: $spacing-3;
    margin-bottom: $spacing-4;

    .mobile-card--small & {
      margin-bottom: $spacing-3;
    }

    .mobile-card--large & {
      margin-bottom: $spacing-5;
    }

    .dark & {
      border-bottom-color: $dark-bg-tertiary;
    }
  }

  &__header-content {
    @include flex-between;
    align-items: flex-start;
    gap: $spacing-3;
  }

  &__header-text {
    flex: 1;
    min-width: 0;
  }

  &__header-actions {
    flex-shrink: 0;
  }

  &__title {
    font-size: $font-size-lg;
    font-weight: $font-weight-semibold;
    color: $light-text-primary;
    margin: 0 0 $spacing-1 0;
    @include text-ellipsis;

    .dark & {
      color: $dark-text-primary;
    }
  }

  &__subtitle {
    font-size: $font-size-sm;
    color: $light-text-secondary;
    margin: 0;
    @include text-clamp(2);

    .dark & {
      color: $dark-text-secondary;
    }
  }

  // 이미지
  &__image {
    margin: -#{$spacing-4} -#{$spacing-4} $spacing-4 -#{$spacing-4};
    overflow: hidden;

    .mobile-card--small & {
      margin: -#{$spacing-3} -#{$spacing-3} $spacing-3 -#{$spacing-3};
    }

    .mobile-card--large & {
      margin: -#{$spacing-6} -#{$spacing-6} $spacing-6 -#{$spacing-6};
    }

    .mobile-card--rounded & {
      border-radius: $border-radius-2xl $border-radius-2xl 0 0;
    }
  }

  &__img {
    width: 100%;
    height: 200px;
    object-fit: cover;
    display: block;

    .mobile-card--small & {
      height: 120px;
    }

    .mobile-card--large & {
      height: 240px;
    }
  }

  // 콘텐츠
  &__content {
    color: $light-text-primary;
    line-height: $line-height-relaxed;

    .dark & {
      color: $dark-text-primary;
    }

    p {
      margin: 0 0 $spacing-3 0;

      &:last-child {
        margin-bottom: 0;
      }
    }
  }

  // 푸터
  &__footer {
    border-top: 1px solid map-get($gray-colors, 100);
    padding-top: $spacing-3;
    margin-top: $spacing-4;

    .mobile-card--small & {
      margin-top: $spacing-3;
    }

    .mobile-card--large & {
      margin-top: $spacing-5;
    }

    .dark & {
      border-top-color: $dark-bg-tertiary;
    }
  }

  // 리플 효과
  &__ripple {
    position: absolute;
    width: 40px;
    height: 40px;
    border-radius: $border-radius-full;
    background: radial-gradient(circle, rgba(map-get($primary-colors, 500), 0.3) 0%, transparent 70%);
    transform: translate(-50%, -50%) scale(0);
    animation: ripple-animation 0.6s ease-out;
    pointer-events: none;
  }

  // 로딩 오버레이
  &__loading {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba($light-bg-primary, 0.8);
    @include flex-center;
    z-index: 10;

    .dark & {
      background: rgba($dark-bg-primary, 0.8);
    }
  }
}

// 리플 애니메이션
@keyframes ripple-animation {
  to {
    transform: translate(-50%, -50%) scale(4);
    opacity: 0;
  }
}

// 모바일 최적화
@include mobile {
  .mobile-card {
    margin-bottom: $spacing-4;

    &:last-child {
      margin-bottom: 0;
    }

    // 터치 영역 확대
    &--clickable {
      @include touch-target(auto);
      min-height: 60px;
    }

    &__title {
      @include responsive-text($font-size-base, $font-size-lg);
    }

    &__subtitle {
      @include responsive-text($font-size-xs, $font-size-sm);
    }
  }
}

// 접근성 개선
@include reduce-motion {
  .mobile-card {
    transition: none;

    &__ripple {
      animation: none;
    }

    &--elevated:hover {
      transform: none;
    }
  }
}

// 고대비 모드
@media (prefers-contrast: high) {
  .mobile-card {
    border: 2px solid currentColor;

    &--outlined {
      border-width: 3px;
    }
  }
}
</style>