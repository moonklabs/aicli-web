<template>
  <div ref="triggerRef" class="tooltip-trigger">
    <slot />
  </div>

  <Teleport to="body">
    <Transition name="tooltip-fade">
      <div
        v-if="visible"
        ref="tooltipRef"
        role="tooltip"
        class="tooltip"
        :class="[
          `tooltip--${theme}`,
          { 'tooltip--nowrap': !wrap }
        ]"
        :style="{
          ...floatingStyles,
          zIndex,
          maxWidth: maxWidth ? (typeof maxWidth === 'number' ? `${maxWidth}px` : maxWidth) : undefined,
        }"
      >
        <!-- 화살표 -->
        <div
          v-if="showArrow"
          class="tooltip__arrow"
          :class="`tooltip__arrow--${arrowPlacement}`"
        />

        <!-- 내용 -->
        <div class="tooltip__content">
          <slot name="content">
            {{ content }}
          </slot>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useOverlay } from '../composables/useOverlayManager'
import { type Placement, usePositioning } from '../composables/usePositioning'

export interface TooltipProps {
  // 툴팁 내용
  content?: string
  // 위치
  placement?: Placement
  // 테마
  theme?: 'dark' | 'light'
  // 오프셋
  offset?: number
  // 화살표 표시
  showArrow?: boolean
  // 지연 시간
  showDelay?: number
  hideDelay?: number
  // 최대 너비
  maxWidth?: string | number
  // 텍스트 줄바꿈
  wrap?: boolean
  // 비활성화
  disabled?: boolean
}

const props = withDefaults(defineProps<TooltipProps>(), {
  placement: 'top',
  theme: 'dark',
  offset: 8,
  showArrow: true,
  showDelay: 500,
  hideDelay: 0,
  wrap: true,
  disabled: false,
})

// 레퍼런스
const triggerRef = ref<HTMLElement>()
const tooltipRef = ref<HTMLElement>()

// 상태
const visible = ref(false)

// 타이머
let showTimer: ReturnType<typeof setTimeout> | null = null
let hideTimer: ReturnType<typeof setTimeout> | null = null

// 고유 ID
const tooltipId = `tooltip-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`

// 오버레이 관리
const { zIndex } = useOverlay({
  id: tooltipId,
  type: 'tooltip',
  closeOnEsc: false,
  closeOnClickOutside: false,
})

// 포지셔닝
const { floatingStyles, position } = usePositioning(
  triggerRef,
  tooltipRef,
  {
    placement: props.placement,
    offset: props.offset,
    flip: true,
  },
)

// 화살표 위치 계산
const arrowPlacement = computed(() => {
  const placement = position.value.placement
  const [side] = placement.split('-')

  // 반대 방향 반환
  const opposite = {
    top: 'bottom',
    right: 'left',
    bottom: 'top',
    left: 'right',
  }

  return opposite[side as keyof typeof opposite]
})

// 표시
function show() {
  if (props.disabled || (!props.content && !tooltipRef.value?.querySelector('.tooltip__content')?.textContent)) {
    return
  }

  clearTimeout(hideTimer!)

  showTimer = setTimeout(() => {
    visible.value = true
  }, props.showDelay)
}

// 숨기기
function hide() {
  clearTimeout(showTimer!)

  hideTimer = setTimeout(() => {
    visible.value = false
  }, props.hideDelay)
}

// 이벤트 핸들러
function handleMouseEnter() {
  show()
}

function handleMouseLeave() {
  hide()
}

function handleFocus() {
  show()
}

function handleBlur() {
  hide()
}

// 이벤트 리스너 설정
function setupEventListeners() {
  if (!triggerRef.value) return

  // 마우스 이벤트
  triggerRef.value.addEventListener('mouseenter', handleMouseEnter)
  triggerRef.value.addEventListener('mouseleave', handleMouseLeave)

  // 포커스 이벤트 (키보드 접근성)
  triggerRef.value.addEventListener('focus', handleFocus, true)
  triggerRef.value.addEventListener('blur', handleBlur, true)

  // 툴팁에도 마우스 이벤트 추가 (hover 유지)
  if (tooltipRef.value) {
    tooltipRef.value.addEventListener('mouseenter', () => clearTimeout(hideTimer!))
    tooltipRef.value.addEventListener('mouseleave', hide)
  }
}

// 이벤트 리스너 정리
function cleanupEventListeners() {
  if (triggerRef.value) {
    triggerRef.value.removeEventListener('mouseenter', handleMouseEnter)
    triggerRef.value.removeEventListener('mouseleave', handleMouseLeave)
    triggerRef.value.removeEventListener('focus', handleFocus, true)
    triggerRef.value.removeEventListener('blur', handleBlur, true)
  }

  if (tooltipRef.value) {
    tooltipRef.value.removeEventListener('mouseenter', () => clearTimeout(hideTimer!))
    tooltipRef.value.removeEventListener('mouseleave', hide)
  }
}

// 마운트 시 설정
onMounted(() => {
  setupEventListeners()
})

// 언마운트 시 정리
onUnmounted(() => {
  cleanupEventListeners()
  clearTimeout(showTimer!)
  clearTimeout(hideTimer!)
})
</script>

<style scoped>
/* 트리거 */
.tooltip-trigger {
  display: inline-block;
}

/* 툴팁 본체 */
.tooltip {
  position: absolute;
  padding: 8px 12px;
  border-radius: 6px;
  font-size: 12px;
  line-height: 1.5;
  pointer-events: none;
  word-wrap: break-word;
}

/* 테마 - 다크 */
.tooltip--dark {
  background-color: rgba(0, 0, 0, 0.9);
  color: #ffffff;
}

.tooltip--dark .tooltip__arrow {
  background-color: rgba(0, 0, 0, 0.9);
}

/* 테마 - 라이트 */
.tooltip--light {
  background-color: #ffffff;
  color: #1f2937;
  box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
}

.tooltip--light .tooltip__arrow {
  background-color: #ffffff;
}

/* 줄바꿈 없음 */
.tooltip--nowrap {
  white-space: nowrap;
}

/* 화살표 */
.tooltip__arrow {
  position: absolute;
  width: 6px;
  height: 6px;
  transform: rotate(45deg);
}

.tooltip__arrow--top {
  top: -3px;
  left: 50%;
  transform: translateX(-50%) rotate(45deg);
}

.tooltip__arrow--right {
  right: -3px;
  top: 50%;
  transform: translateY(-50%) rotate(45deg);
}

.tooltip__arrow--bottom {
  bottom: -3px;
  left: 50%;
  transform: translateX(-50%) rotate(45deg);
}

.tooltip__arrow--left {
  left: -3px;
  top: 50%;
  transform: translateY(-50%) rotate(45deg);
}

/* 내용 */
.tooltip__content {
  position: relative;
  z-index: 1;
}

/* 트랜지션 */
.tooltip-fade-enter-active,
.tooltip-fade-leave-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
}

.tooltip-fade-enter-from,
.tooltip-fade-leave-to {
  opacity: 0;
  transform: scale(0.9);
}
</style>