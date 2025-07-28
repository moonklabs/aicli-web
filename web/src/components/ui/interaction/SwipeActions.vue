<template>
  <div class="swipe-actions" ref="containerRef">
    <!-- 왼쪽 액션 배경 -->
    <div
      v-if="leftActions.length > 0"
      class="swipe-actions__background swipe-actions__background--left"
      :style="leftBackgroundStyle"
    >
      <div class="swipe-actions__actions swipe-actions__actions--left">
        <button
          v-for="action in leftActions"
          :key="action.id"
          class="swipe-actions__action"
          :class="`swipe-actions__action--${action.type || 'default'}`"
          @click="handleActionClick(action)"
          :aria-label="action.label"
        >
          <component v-if="action.icon" :is="action.icon" />
          <span v-if="action.showLabel !== false">{{ action.label }}</span>
        </button>
      </div>
    </div>

    <!-- 오른쪽 액션 배경 -->
    <div
      v-if="rightActions.length > 0"
      class="swipe-actions__background swipe-actions__background--right"
      :style="rightBackgroundStyle"
    >
      <div class="swipe-actions__actions swipe-actions__actions--right">
        <button
          v-for="action in rightActions"
          :key="action.id"
          class="swipe-actions__action"
          :class="`swipe-actions__action--${action.type || 'default'}`"
          @click="handleActionClick(action)"
          :aria-label="action.label"
        >
          <component v-if="action.icon" :is="action.icon" />
          <span v-if="action.showLabel !== false">{{ action.label }}</span>
        </button>
      </div>
    </div>

    <!-- 메인 콘텐츠 -->
    <div
      class="swipe-actions__content"
      :style="contentStyle"
      @touchstart="handleTouchStart"
      @touchmove="handleTouchMove"
      @touchend="handleTouchEnd"
      @touchcancel="handleTouchCancel"
    >
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'

interface SwipeAction {
  id: string
  label: string
  icon?: any
  type?: 'default' | 'primary' | 'danger' | 'warning' | 'success'
  showLabel?: boolean
  handler: () => void
}

interface Props {
  leftActions?: SwipeAction[]
  rightActions?: SwipeAction[]
  threshold?: number
  disabled?: boolean
  allowBounce?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  leftActions: () => [],
  rightActions: () => [],
  threshold: 80,
  disabled: false,
  allowBounce: true,
})

const emit = defineEmits<{
  'swipe-start': []
  'swipe-move': [offset: number]
  'swipe-end': [offset: number]
  'action-trigger': [action: SwipeAction]
}>()

const containerRef = ref<HTMLElement>()

// 스와이프 상태
const isDragging = ref(false)
const startX = ref(0)
const currentX = ref(0)
const offset = ref(0)
const isAnimating = ref(false)

// 터치 이벤트 핸들러
const handleTouchStart = (event: TouchEvent) => {
  if (props.disabled || isAnimating.value) return

  const touch = event.touches[0]
  startX.value = touch.clientX
  currentX.value = touch.clientX
  isDragging.value = true

  emit('swipe-start')
}

const handleTouchMove = (event: TouchEvent) => {
  if (!isDragging.value || props.disabled) return

  event.preventDefault()

  const touch = event.touches[0]
  currentX.value = touch.clientX

  let newOffset = currentX.value - startX.value

  // 액션이 없는 방향으로의 스와이프 제한
  if (newOffset > 0 && props.leftActions.length === 0) {
    newOffset = props.allowBounce ? newOffset * 0.3 : 0
  }
  if (newOffset < 0 && props.rightActions.length === 0) {
    newOffset = props.allowBounce ? newOffset * 0.3 : 0
  }

  // 최대 스와이프 거리 제한
  const maxOffset = 120
  if (Math.abs(newOffset) > maxOffset) {
    newOffset = newOffset > 0 ? maxOffset : -maxOffset
  }

  offset.value = newOffset
  emit('swipe-move', offset.value)
}

const handleTouchEnd = () => {
  if (!isDragging.value || props.disabled) return

  isDragging.value = false

  const absOffset = Math.abs(offset.value)

  // 임계값을 넘으면 액션 실행
  if (absOffset >= props.threshold) {
    if (offset.value > 0 && props.leftActions.length > 0) {
      // 왼쪽 스와이프 - 첫 번째 액션 실행
      handleActionClick(props.leftActions[0])
    } else if (offset.value < 0 && props.rightActions.length > 0) {
      // 오른쪽 스와이프 - 첫 번째 액션 실행
      handleActionClick(props.rightActions[0])
    }
  }

  // 원래 위치로 복귀
  resetPosition()
}

const handleTouchCancel = () => {
  isDragging.value = false
  resetPosition()
}

// 위치 초기화
const resetPosition = () => {
  isAnimating.value = true
  offset.value = 0

  setTimeout(() => {
    isAnimating.value = false
  }, 300)

  emit('swipe-end', 0)
}

// 액션 클릭 핸들러
const handleActionClick = (action: SwipeAction) => {
  emit('action-trigger', action)
  action.handler()

  // 액션 실행 후 위치 초기화
  setTimeout(() => {
    resetPosition()
  }, 100)
}

// 스타일 계산
const contentStyle = computed(() => ({
  transform: `translateX(${offset.value}px)`,
  transition: isAnimating.value ? 'transform 0.3s ease' : 'none',
}))

const leftBackgroundStyle = computed(() => {
  const opacity = Math.min(Math.abs(offset.value) / props.threshold, 1)
  return {
    opacity: offset.value > 0 ? opacity : 0,
    transform: `translateX(${Math.min(offset.value, 0)}px)`,
  }
})

const rightBackgroundStyle = computed(() => {
  const opacity = Math.min(Math.abs(offset.value) / props.threshold, 1)
  return {
    opacity: offset.value < 0 ? opacity : 0,
    transform: `translateX(${Math.max(offset.value, 0)}px)`,
  }
})

// 외부에서 위치 초기화 호출 가능
const reset = () => {
  resetPosition()
}

defineExpose({
  reset,
})
</script>

<style lang="scss" scoped>
@use '@/styles/variables' as *;
@use '@/styles/mixins' as *;

.swipe-actions {
  position: relative;
  overflow: hidden;
  background: $light-bg-primary;

  .dark & {
    background: $dark-bg-secondary;
  }

  &__background {
    position: absolute;
    top: 0;
    bottom: 0;
    @include flex-center;
    z-index: 1;

    &--left {
      left: 0;
      background: map-get($primary-colors, 500);
      border-radius: 0 $border-radius-lg 0 0;
    }

    &--right {
      right: 0;
      background: $error;
      border-radius: $border-radius-lg 0 0 0;
    }
  }

  &__actions {
    display: flex;
    align-items: center;
    gap: $spacing-2;
    padding: 0 $spacing-4;

    &--left {
      flex-direction: row;
    }

    &--right {
      flex-direction: row-reverse;
    }
  }

  &__action {
    @include touch-target(48px);
    @include flex-center;
    flex-direction: column;
    gap: 2px;
    background: none;
    border: none;
    color: white;
    cursor: pointer;
    border-radius: $border-radius-base;
    padding: $spacing-2;
    transition: $transition-base;

    svg {
      width: 20px;
      height: 20px;
    }

    span {
      font-size: $font-size-xs;
      font-weight: $font-weight-medium;
      white-space: nowrap;
    }

    &:hover {
      @include no-touch {
        background: rgba(255, 255, 255, 0.1);
      }
    }

    &:active {
      background: rgba(255, 255, 255, 0.2);
      transform: scale(0.95);
    }

    &--primary {
      // 기본 프라이머리 색상 사용
    }

    &--danger {
      // 부모의 배경색이 이미 위험 색상
    }

    &--warning {
      color: darken($warning, 20%);
    }

    &--success {
      color: darken($success, 20%);
    }
  }

  &__content {
    position: relative;
    z-index: 2;
    background: inherit;
    will-change: transform;

    // 터치 스크롤링 방지
    touch-action: pan-y;
  }
}

// 햅틱 피드백 (지원하는 기기에서)
.swipe-actions__action:active {
  // 진동 피드백은 JavaScript에서 처리
}

// 접근성 개선
@include reduce-motion {
  .swipe-actions__content {
    transition: none !important;
  }
}

// 모바일 최적화
@include mobile {
  .swipe-actions {
    &__action {
      @include touch-target(56px);
      padding: $spacing-3;

      svg {
        width: 24px;
        height: 24px;
      }

      span {
        font-size: $font-size-sm;
      }
    }

    &__actions {
      padding: 0 $spacing-6;
      gap: $spacing-3;
    }
  }
}
</style>