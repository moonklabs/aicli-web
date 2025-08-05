<template>
  <button
    ref="buttonRef"
    :class="buttonClasses"
    :disabled="disabled || loading"
    :aria-label="ariaLabel"
    :type="type"
    @click="handleClick"
  >
    <!-- 로딩 스피너 -->
    <div v-if="loading" class="touch-button__loading">
      <div class="touch-button__spinner" />
    </div>

    <!-- 아이콘 -->
    <component
      v-if="icon && !loading"
      :is="icon"
      class="touch-button__icon"
      :class="{ 'touch-button__icon--only': !$slots.default }"
    />

    <!-- 텍스트 컨텐츠 -->
    <span v-if="$slots.default && !loading" class="touch-button__text">
      <slot />
    </span>

    <!-- 뱃지 -->
    <span
      v-if="badge && !loading"
      class="touch-button__badge"
      :class="`touch-button__badge--${badgeType}`"
    >
      {{ badge }}
    </span>

    <!-- 리플 효과 -->
    <div
      v-if="showRipple"
      ref="rippleRef"
      class="touch-button__ripple"
      :style="rippleStyle"
    />
  </button>
</template>

<script setup lang="ts">
import { computed, ref, nextTick } from 'vue'
import { useTouchInteractions } from '@/composables/useTouchInteractions'
import { useAdvancedGestures } from '@/composables/useAdvancedGestures'

type ButtonSize = 'xs' | 'sm' | 'md' | 'lg' | 'xl'
type ButtonType = 'primary' | 'secondary' | 'tertiary' | 'danger' | 'success' | 'warning'
type BadgeType = 'primary' | 'secondary' | 'success' | 'warning' | 'error' | 'info'

interface Props {
  // 기본 속성
  type?: 'button' | 'submit' | 'reset'
  size?: ButtonSize
  variant?: ButtonType
  disabled?: boolean
  loading?: boolean
  
  // 시각적 속성
  icon?: any
  badge?: string | number
  badgeType?: BadgeType
  round?: boolean
  block?: boolean
  
  // 터치 최적화
  hapticFeedback?: boolean
  rippleEffect?: boolean
  longPressAction?: boolean
  
  // 접근성
  ariaLabel?: string
}

const props = withDefaults(defineProps<Props>(), {
  type: 'button',
  size: 'md',
  variant: 'primary',
  disabled: false,
  loading: false,
  badgeType: 'primary',
  round: false,
  block: false,
  hapticFeedback: true,
  rippleEffect: true,
  longPressAction: false,
})

const emit = defineEmits<{
  click: [event: MouseEvent]
  'long-press': [event: TouchEvent]
  'touch-start': [event: TouchEvent]
  'touch-end': [event: TouchEvent]
}>()

// 템플릿 참조
const buttonRef = ref<HTMLButtonElement>()
const rippleRef = ref<HTMLDivElement>()

// 리플 효과 상태
const showRipple = ref(false)
const rippleStyle = ref({})

// 터치 인터랙션 설정
const {
  onGesture,
  triggerHapticFeedback,
  applyTouchOptimizations,
  isTouchDevice,
} = useTouchInteractions(buttonRef, {
  enableTap: true,
  enableLongPress: props.longPressAction,
  enableHapticFeedback: props.hapticFeedback,
  minTouchTargetSize: getSizeValue(props.size),
  touchTargetExpansion: true,
})

// 고급 제스처 설정
const advancedGestures = useAdvancedGestures(buttonRef, {
  enableRotation: false, // 버튼에서는 회전 비활성화
  enableMultiFingerGestures: false, // 버튼에서는 멀티터치 비활성화
  enableVelocityGestures: true, // 빠른 탭 감지 활성화
  enableInertia: false, // 버튼에서는 관성 비활성화
  enableHapticFeedback: props.hapticFeedback,
  velocityThreshold: 0.3,
})

// 크기별 최소 터치 타겟 크기
function getSizeValue(size: ButtonSize): number {
  const sizes = {
    xs: 32,
    sm: 36,
    md: 44,
    lg: 52,
    xl: 60,
  }
  return sizes[size]
}

// 버튼 클래스 계산
const buttonClasses = computed(() => [
  'touch-button',
  `touch-button--${props.variant}`,
  `touch-button--${props.size}`,
  {
    'touch-button--disabled': props.disabled,
    'touch-button--loading': props.loading,
    'touch-button--round': props.round,
    'touch-button--block': props.block,
    'touch-button--icon-only': props.icon && !$slots.default,
    'touch-button--touch-device': isTouchDevice.value,
  },
])

// 리플 효과 생성
const createRipple = (event: MouseEvent | TouchEvent) => {
  if (!props.rippleEffect || !buttonRef.value || !rippleRef.value) return

  const button = buttonRef.value
  const rect = button.getBoundingClientRect()
  
  let clientX: number, clientY: number
  
  if (event instanceof TouchEvent && event.touches.length > 0) {
    clientX = event.touches[0].clientX
    clientY = event.touches[0].clientY
  } else if (event instanceof MouseEvent) {
    clientX = event.clientX
    clientY = event.clientY
  } else {
    return
  }

  const x = clientX - rect.left
  const y = clientY - rect.top
  const size = Math.max(rect.width, rect.height) * 2

  rippleStyle.value = {
    left: `${x - size / 2}px`,
    top: `${y - size / 2}px`,
    width: `${size}px`,
    height: `${size}px`,
  }

  showRipple.value = true

  // 리플 애니메이션 종료 후 제거
  setTimeout(() => {
    showRipple.value = false
  }, 600)
}

// 클릭 핸들러
const handleClick = (event: MouseEvent) => {
  if (props.disabled || props.loading) return

  // 햅틱 피드백
  if (props.hapticFeedback) {
    triggerHapticFeedback(20, 'light')
  }

  // 리플 효과
  createRipple(event)

  emit('click', event)
}

// 제스처 이벤트 핸들러 설정
onGesture('tap', (gestureEvent) => {
  if (props.disabled || props.loading) return
  
  // 터치 탭에 대한 추가 처리
  const touchEvent = new TouchEvent('touchstart')
  emit('touch-start', touchEvent)
})

onGesture('longpress', (gestureEvent) => {
  if (props.disabled || props.loading || !props.longPressAction) return

  // 롱 프레스 햅틱 피드백
  if (props.hapticFeedback) {
    triggerHapticFeedback([50, 50], 'medium')
  }

  const touchEvent = new TouchEvent('touchstart')
  emit('long-press', touchEvent)
})

// 고급 제스처 이벤트 핸들러
advancedGestures.on('fastswipe', (advancedEvent) => {
  if (props.disabled || props.loading) return
  
  // 빠른 스와이프는 즉시 클릭으로 처리
  if (props.hapticFeedback) {
    advancedGestures.triggerHapticFeedback([30])
  }
  
  // 빠른 클릭 이벤트 발생
  const clickEvent = new MouseEvent('click', {
    bubbles: true,
    cancelable: true,
    clientX: advancedEvent.centroid.x,
    clientY: advancedEvent.centroid.y,
  })
  
  emit('click', clickEvent)
})

// 더블 탭 감지
let lastTapTime = 0
const doubleTapThreshold = 300 // 300ms 내 두 번째 탭

onGesture('tap', (gestureEvent) => {
  if (props.disabled || props.loading) return
  
  const now = Date.now()
  const timeSinceLastTap = now - lastTapTime
  
  if (timeSinceLastTap < doubleTapThreshold) {
    // 더블 탭 감지
    if (props.hapticFeedback) {
      triggerHapticFeedback([20, 20], 'light')
    }
    
    // 더블 탭 이벤트 (커스텀)
    const doubleClickEvent = new CustomEvent('double-tap', {
      detail: {
        position: { x: gestureEvent.currentPoint.x, y: gestureEvent.currentPoint.y },
        timestamp: now,
      },
    })
    
    if (buttonRef.value) {
      buttonRef.value.dispatchEvent(doubleClickEvent)
    }
  }
  
  lastTapTime = now
  
  // 터치 탭에 대한 추가 처리
  const touchEvent = new TouchEvent('touchstart')
  emit('touch-start', touchEvent)
})
</script>

<style lang="scss" scoped>
@use '@/styles/variables' as *;
@use '@/styles/mixins' as *;

.touch-button {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: $spacing-2;
  border: none;
  border-radius: $border-radius-md;
  font-family: inherit;
  font-weight: $font-weight-medium;
  line-height: 1;
  text-decoration: none;
  cursor: pointer;
  overflow: hidden;
  transition: all 0.2s ease;
  user-select: none;
  -webkit-tap-highlight-color: transparent;

  // 터치 디바이스 최적화
  &--touch-device {
    @include touch-target;
    @include touch-feedback;
  }

  // 크기 변형
  &--xs {
    @include responsive-spacing(padding, $spacing-1 $spacing-2, $spacing-1 $spacing-3);
    @include responsive-text($font-size-xs, $font-size-sm);
    min-height: 32px;
    border-radius: $border-radius-sm;

    .touch-button__icon {
      width: 14px;
      height: 14px;
    }
  }

  &--sm {
    @include responsive-spacing(padding, $spacing-2 $spacing-3, $spacing-2 $spacing-4);
    @include responsive-text($font-size-sm, $font-size-base);
    min-height: 36px;

    .touch-button__icon {
      width: 16px;
      height: 16px;
    }
  }

  &--md {
    @include responsive-spacing(padding, $spacing-3 $spacing-4, $spacing-3 $spacing-5);
    @include responsive-text($font-size-base, $font-size-base);
    min-height: 44px;

    .touch-button__icon {
      width: 18px;
      height: 18px;
    }
  }

  &--lg {
    @include responsive-spacing(padding, $spacing-4 $spacing-6, $spacing-4 $spacing-8);
    @include responsive-text($font-size-lg, $font-size-lg);
    min-height: 52px;
    border-radius: $border-radius-lg;

    .touch-button__icon {
      width: 20px;
      height: 20px;
    }
  }

  &--xl {
    @include responsive-spacing(padding, $spacing-5 $spacing-8, $spacing-6 $spacing-10);
    @include responsive-text($font-size-xl, $font-size-xl);
    min-height: 60px;
    border-radius: $border-radius-xl;

    .touch-button__icon {
      width: 24px;
      height: 24px;
    }
  }

  // 색상 변형
  &--primary {
    background: map-get($primary-colors, 500);
    color: white;

    &:hover:not(&--disabled):not(&--loading) {
      @include no-touch {
        background: map-get($primary-colors, 600);
        transform: translateY(-1px);
        box-shadow: $shadow-md;
      }
    }

    &:active:not(&--disabled):not(&--loading) {
      background: map-get($primary-colors, 700);
      transform: translateY(0);
    }
  }

  &--secondary {
    background: map-get($gray-colors, 100);
    color: map-get($gray-colors, 800);
    border: 1px solid map-get($gray-colors, 300);

    .dark & {
      background: $dark-bg-tertiary;
      color: $dark-text-primary;
      border-color: $dark-bg-tertiary;
    }

    &:hover:not(&--disabled):not(&--loading) {
      @include no-touch {
        background: map-get($gray-colors, 200);
        border-color: map-get($gray-colors, 400);

        .dark & {
          background: lighten($dark-bg-tertiary, 10%);
        }
      }
    }
  }

  &--tertiary {
    background: transparent;
    color: map-get($primary-colors, 600);

    .dark & {
      color: map-get($primary-colors, 400);
    }

    &:hover:not(&--disabled):not(&--loading) {
      @include no-touch {
        background: map-get($primary-colors, 50);

        .dark & {
          background: rgba(map-get($primary-colors, 500), 0.1);
        }
      }
    }
  }

  &--danger {
    background: $error;
    color: white;

    &:hover:not(&--disabled):not(&--loading) {
      @include no-touch {
        background: darken($error, 10%);
      }
    }
  }

  &--success {
    background: $success;
    color: white;

    &:hover:not(&--disabled):not(&--loading) {
      @include no-touch {
        background: darken($success, 10%);
      }
    }
  }

  &--warning {
    background: $warning;
    color: white;

    &:hover:not(&--disabled):not(&--loading) {
      @include no-touch {
        background: darken($warning, 10%);
      }
    }
  }

  // 상태
  &--disabled {
    opacity: 0.6;
    cursor: not-allowed;
    pointer-events: none;
  }

  &--loading {
    pointer-events: none;
  }

  // 모양
  &--round {
    border-radius: $border-radius-full;
  }

  &--block {
    width: 100%;
  }

  &--icon-only {
    aspect-ratio: 1;
    padding: $spacing-2;

    &.touch-button--md {
      width: 44px;
      height: 44px;
    }
  }

  // 컴포넌트
  &__loading {
    @include flex-center;
  }

  &__spinner {
    width: 16px;
    height: 16px;
    border: 2px solid currentColor;
    border-radius: 50%;
    border-top-color: transparent;
    @include loading-spinner;
  }

  &__icon {
    flex-shrink: 0;
    transition: transform 0.2s ease;

    &--only {
      margin: 0;
    }
  }

  &__text {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  &__badge {
    @include status-badge(map-get($primary-colors, 500));
    font-size: 0.75em;
    line-height: 1;
    margin-left: $spacing-1;

    &--primary { @include status-badge(map-get($primary-colors, 500)); }
    &--secondary { @include status-badge(map-get($gray-colors, 500)); }
    &--success { @include status-badge($success); }
    &--warning { @include status-badge($warning); }
    &--error { @include status-badge($error); }
    &--info { @include status-badge($info); }
  }

  &__ripple {
    position: absolute;
    border-radius: 50%;
    background: rgba(255, 255, 255, 0.3);
    pointer-events: none;
    transform: scale(0);
    animation: ripple 0.6s ease-out;
  }

  // 접근성 개선
  &:focus {
    outline: none;
    box-shadow: 0 0 0 3px rgba(map-get($primary-colors, 500), 0.3);
  }

  // 모바일 최적화
  @include mobile {
    min-height: 48px;
    font-size: $font-size-base;

    &--xs { min-height: 36px; }
    &--sm { min-height: 40px; }
    &--lg { min-height: 56px; }
    &--xl { min-height: 64px; }
  }
}

// 리플 애니메이션
@keyframes ripple {
  to {
    transform: scale(1);
    opacity: 0;
  }
}

// 접근성 개선
@include reduce-motion {
  .touch-button {
    transition: none;

    &:hover {
      transform: none;
    }

    &__ripple {
      display: none;
    }

    &__icon {
      transition: none;
    }
  }
}
</style>