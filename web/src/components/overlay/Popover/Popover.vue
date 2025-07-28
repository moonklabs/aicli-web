<template>
  <div ref="triggerRef" class="popover-trigger" @click="handleTriggerClick">
    <slot name="trigger" />
  </div>
  
  <Teleport to="body">
    <Transition name="popover-fade">
      <div
        v-if="visible"
        ref="popoverRef"
        class="popover"
        :style="{
          ...floatingStyles,
          zIndex,
        }"
        @click.stop
      >
        <!-- 화살표 -->
        <div
          v-if="showArrow"
          class="popover__arrow"
          :class="`popover__arrow--${arrowPlacement}`"
        />
        
        <!-- 헤더 -->
        <div v-if="title || $slots.header" class="popover__header">
          <slot name="header">
            <h3 class="popover__title">{{ title }}</h3>
          </slot>
          <button
            v-if="closable"
            type="button"
            class="popover__close"
            @click="handleClose"
            aria-label="닫기"
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor">
              <path d="M18 6L6 18M6 6l12 12" stroke-width="2" stroke-linecap="round"/>
            </svg>
          </button>
        </div>
        
        <!-- 본문 -->
        <div class="popover__body">
          <slot />
        </div>
        
        <!-- 푸터 -->
        <div v-if="$slots.footer" class="popover__footer">
          <slot name="footer" />
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useOverlay } from '../composables/useOverlayManager'
import { usePositioning, type Placement } from '../composables/usePositioning'

export interface PopoverProps {
  // 표시 상태
  visible?: boolean
  // 제목
  title?: string
  // 트리거 방식
  trigger?: 'click' | 'hover' | 'focus' | 'manual'
  // 위치
  placement?: Placement
  // 오프셋
  offset?: number
  // 화살표 표시
  showArrow?: boolean
  // 닫기 버튼
  closable?: boolean
  // 클릭 외부 영역으로 닫기
  closeOnClickOutside?: boolean
  // 지연 시간 (hover 모드)
  showDelay?: number
  hideDelay?: number
  // 최대 너비
  maxWidth?: string | number
}

const props = withDefaults(defineProps<PopoverProps>(), {
  trigger: 'click',
  placement: 'bottom',
  offset: 8,
  showArrow: true,
  closable: false,
  closeOnClickOutside: true,
  showDelay: 100,
  hideDelay: 100,
})

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'show': []
  'hide': []
}>()

// 레퍼런스
const triggerRef = ref<HTMLElement>()
const popoverRef = ref<HTMLElement>()

// 상태
const internalVisible = ref(false)
const visible = computed({
  get: () => props.visible ?? internalVisible.value,
  set: (value) => {
    internalVisible.value = value
    emit('update:visible', value)
  }
})

// 타이머
let showTimer: ReturnType<typeof setTimeout> | null = null
let hideTimer: ReturnType<typeof setTimeout> | null = null

// 고유 ID
const popoverId = `popover-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`

// 오버레이 관리
const { zIndex } = useOverlay({
  id: popoverId,
  type: 'popover',
  closeOnEsc: true,
  closeOnClickOutside: props.closeOnClickOutside,
  onClose: () => handleClose(),
})

// 포지셔닝
const { floatingStyles, position } = usePositioning(
  triggerRef,
  popoverRef,
  {
    placement: props.placement,
    offset: props.offset,
    flip: true,
  }
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
  clearTimeout(hideTimer!)
  
  if (props.trigger === 'hover') {
    showTimer = setTimeout(() => {
      visible.value = true
      emit('show')
    }, props.showDelay)
  } else {
    visible.value = true
    emit('show')
  }
}

// 숨기기
function hide() {
  clearTimeout(showTimer!)
  
  if (props.trigger === 'hover') {
    hideTimer = setTimeout(() => {
      visible.value = false
      emit('hide')
    }, props.hideDelay)
  } else {
    visible.value = false
    emit('hide')
  }
}

// 트리거 클릭 핸들러
function handleTriggerClick() {
  if (props.trigger === 'click') {
    visible.value = !visible.value
  }
}

// 닫기 핸들러
function handleClose() {
  hide()
}

// 외부 클릭 핸들러
function handleClickOutside(event: MouseEvent) {
  if (!props.closeOnClickOutside || props.trigger === 'manual') return
  
  const target = event.target as Node
  
  if (
    triggerRef.value &&
    popoverRef.value &&
    !triggerRef.value.contains(target) &&
    !popoverRef.value.contains(target)
  ) {
    hide()
  }
}

// 이벤트 리스너 설정
function setupEventListeners() {
  if (!triggerRef.value) return
  
  switch (props.trigger) {
    case 'hover':
      triggerRef.value.addEventListener('mouseenter', show)
      triggerRef.value.addEventListener('mouseleave', hide)
      popoverRef.value?.addEventListener('mouseenter', () => clearTimeout(hideTimer!))
      popoverRef.value?.addEventListener('mouseleave', hide)
      break
      
    case 'focus':
      triggerRef.value.addEventListener('focus', show, true)
      triggerRef.value.addEventListener('blur', hide, true)
      break
  }
  
  // 외부 클릭 리스너
  document.addEventListener('click', handleClickOutside)
}

// 이벤트 리스너 정리
function cleanupEventListeners() {
  if (triggerRef.value) {
    triggerRef.value.removeEventListener('mouseenter', show)
    triggerRef.value.removeEventListener('mouseleave', hide)
    triggerRef.value.removeEventListener('focus', show, true)
    triggerRef.value.removeEventListener('blur', hide, true)
  }
  
  if (popoverRef.value) {
    popoverRef.value.removeEventListener('mouseenter', () => clearTimeout(hideTimer!))
    popoverRef.value.removeEventListener('mouseleave', hide)
  }
  
  document.removeEventListener('click', handleClickOutside)
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

// visible prop 변경 감지
watch(() => props.visible, (newVisible) => {
  internalVisible.value = newVisible ?? false
})
</script>

<style scoped>
/* 트리거 */
.popover-trigger {
  display: inline-block;
}

/* 팝오버 본체 */
.popover {
  position: absolute;
  background-color: var(--bg-color, #ffffff);
  border-radius: 8px;
  box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -2px rgba(0, 0, 0, 0.05);
  min-width: 200px;
  max-width: v-bind('props.maxWidth || "400px"');
}

/* 화살표 */
.popover__arrow {
  position: absolute;
  width: 8px;
  height: 8px;
  background-color: var(--bg-color, #ffffff);
  transform: rotate(45deg);
}

.popover__arrow--top {
  top: -4px;
  left: 50%;
  transform: translateX(-50%) rotate(45deg);
}

.popover__arrow--right {
  right: -4px;
  top: 50%;
  transform: translateY(-50%) rotate(45deg);
}

.popover__arrow--bottom {
  bottom: -4px;
  left: 50%;
  transform: translateX(-50%) rotate(45deg);
}

.popover__arrow--left {
  left: -4px;
  top: 50%;
  transform: translateY(-50%) rotate(45deg);
}

/* 헤더 */
.popover__header {
  padding: 12px 16px;
  border-bottom: 1px solid var(--border-color, #e5e7eb);
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.popover__title {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: var(--text-color, #1f2937);
}

.popover__close {
  background: none;
  border: none;
  padding: 2px;
  cursor: pointer;
  color: var(--text-color-secondary, #6b7280);
  transition: color 0.2s ease;
  border-radius: 3px;
  line-height: 0;
}

.popover__close:hover {
  color: var(--text-color, #1f2937);
  background-color: var(--bg-color-hover, #f3f4f6);
}

/* 본문 */
.popover__body {
  padding: 16px;
  color: var(--text-color, #1f2937);
  font-size: 14px;
  line-height: 1.5;
}

/* 푸터 */
.popover__footer {
  padding: 12px 16px;
  border-top: 1px solid var(--border-color, #e5e7eb);
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
}

/* 트랜지션 */
.popover-fade-enter-active,
.popover-fade-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.popover-fade-enter-from,
.popover-fade-leave-to {
  opacity: 0;
  transform: scale(0.95);
}

/* 다크 모드 */
@media (prefers-color-scheme: dark) {
  .popover {
    --bg-color: #374151;
    --text-color: #f3f4f6;
    --text-color-secondary: #9ca3af;
    --border-color: #4b5563;
    --bg-color-hover: #4b5563;
  }
  
  .popover__arrow {
    background-color: #374151;
  }
}
</style>