<template>
  <Teleport to="body">
    <Transition
      :name="`drawer-slide-${placement}`"
      @after-enter="onAfterEnter"
      @after-leave="onAfterLeave"
    >
      <div
        v-if="visible"
        ref="drawerRef"
        class="drawer-container"
        :style="{ zIndex }"
        @click="handleBackdropClick"
      >
        <!-- 백드롭 -->
        <div class="drawer-backdrop" :class="{ 'drawer-backdrop--transparent': !mask }" />
        
        <!-- 드로어 본체 -->
        <div
          class="drawer"
          :class="[
            `drawer--${placement}`,
            `drawer--${size}`
          ]"
          :style="drawerStyle"
          @click.stop
        >
          <!-- 헤더 -->
          <div v-if="!hideHeader" class="drawer__header">
            <slot name="header">
              <h2 class="drawer__title">{{ title }}</h2>
            </slot>
            <button
              v-if="closable"
              type="button"
              class="drawer__close"
              @click="handleClose"
              aria-label="닫기"
            >
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                <path d="M18 6L6 18M6 6l12 12" stroke-width="2" stroke-linecap="round"/>
              </svg>
            </button>
          </div>
          
          <!-- 본문 -->
          <div class="drawer__body" :style="bodyStyle">
            <slot />
          </div>
          
          <!-- 푸터 -->
          <div v-if="!hideFooter && $slots.footer" class="drawer__footer">
            <slot name="footer" />
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'
import { useOverlay } from '../composables/useOverlayManager'
import { useFocusTrap } from '../composables/useFocusTrap'

export type DrawerPlacement = 'top' | 'right' | 'bottom' | 'left'
export type DrawerSize = 'small' | 'medium' | 'large'

export interface DrawerProps {
  // 표시 상태
  visible: boolean
  // 제목
  title?: string
  // 위치
  placement?: DrawerPlacement
  // 크기
  size?: DrawerSize
  // 커스텀 너비/높이
  width?: string | number
  height?: string | number
  // 닫기 버튼 표시
  closable?: boolean
  // 마스크 표시
  mask?: boolean
  // ESC로 닫기
  closeOnEsc?: boolean
  // 배경 클릭으로 닫기
  closeOnClickOutside?: boolean
  // 헤더 숨기기
  hideHeader?: boolean
  // 푸터 숨기기
  hideFooter?: boolean
  // 바디 스타일
  bodyStyle?: Record<string, any>
}

const props = withDefaults(defineProps<DrawerProps>(), {
  placement: 'right',
  size: 'medium',
  closable: true,
  mask: true,
  closeOnEsc: true,
  closeOnClickOutside: true,
  hideHeader: false,
  hideFooter: false,
})

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'close': []
  'after-enter': []
  'after-leave': []
}>()

// 드로어 레퍼런스
const drawerRef = ref<HTMLElement>()

// 고유 ID 생성
const drawerId = `drawer-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`

// 오버레이 관리
const { zIndex } = useOverlay({
  id: drawerId,
  type: 'drawer',
  closeOnEsc: props.closeOnEsc,
  closeOnClickOutside: props.closeOnClickOutside,
  onClose: () => handleClose(),
})

// 포커스 트랩
useFocusTrap(drawerRef, {
  autoFocus: true,
  restoreFocus: true,
})

// 크기별 기본값
const sizeMap = {
  top: { small: '200px', medium: '300px', large: '400px' },
  bottom: { small: '200px', medium: '300px', large: '400px' },
  left: { small: '300px', medium: '400px', large: '600px' },
  right: { small: '300px', medium: '400px', large: '600px' },
}

// 드로어 스타일 계산
const drawerStyle = computed(() => {
  const style: Record<string, any> = {}
  const isHorizontal = props.placement === 'left' || props.placement === 'right'
  
  if (isHorizontal && props.width) {
    style.width = typeof props.width === 'number' ? `${props.width}px` : props.width
  } else if (!isHorizontal && props.height) {
    style.height = typeof props.height === 'number' ? `${props.height}px` : props.height
  }
  
  return style
})

// 닫기 처리
function handleClose() {
  emit('update:visible', false)
  emit('close')
}

// 배경 클릭 처리
function handleBackdropClick(event: MouseEvent) {
  if (props.closeOnClickOutside && event.target === event.currentTarget) {
    handleClose()
  }
}

// 트랜지션 후 이벤트
function onAfterEnter() {
  emit('after-enter')
}

function onAfterLeave() {
  emit('after-leave')
}

// body 스크롤 방지
watch(() => props.visible, (visible) => {
  if (visible) {
    document.body.style.overflow = 'hidden'
  } else {
    nextTick(() => {
      const hasOtherDrawers = document.querySelectorAll('.drawer-container').length > 0
      if (!hasOtherDrawers) {
        document.body.style.overflow = ''
      }
    })
  }
})
</script>

<style scoped>
/* 컨테이너 */
.drawer-container {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
}

/* 백드롭 */
.drawer-backdrop {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.5);
  transition: opacity 0.3s ease;
}

.drawer-backdrop--transparent {
  background-color: transparent;
}

/* 드로어 본체 */
.drawer {
  position: absolute;
  background-color: var(--bg-color, #ffffff);
  box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -2px rgba(0, 0, 0, 0.05);
  display: flex;
  flex-direction: column;
  transition: transform 0.3s ease;
}

/* 위치별 스타일 */
.drawer--top {
  top: 0;
  left: 0;
  right: 0;
}

.drawer--right {
  top: 0;
  right: 0;
  bottom: 0;
}

.drawer--bottom {
  bottom: 0;
  left: 0;
  right: 0;
}

.drawer--left {
  top: 0;
  left: 0;
  bottom: 0;
}

/* 크기별 스타일 */
.drawer--top.drawer--small { height: 200px; }
.drawer--top.drawer--medium { height: 300px; }
.drawer--top.drawer--large { height: 400px; }

.drawer--bottom.drawer--small { height: 200px; }
.drawer--bottom.drawer--medium { height: 300px; }
.drawer--bottom.drawer--large { height: 400px; }

.drawer--left.drawer--small { width: 300px; }
.drawer--left.drawer--medium { width: 400px; }
.drawer--left.drawer--large { width: 600px; }

.drawer--right.drawer--small { width: 300px; }
.drawer--right.drawer--medium { width: 400px; }
.drawer--right.drawer--large { width: 600px; }

/* 헤더 */
.drawer__header {
  padding: 20px 24px;
  border-bottom: 1px solid var(--border-color, #e5e7eb);
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
}

.drawer__title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--text-color, #1f2937);
}

.drawer__close {
  background: none;
  border: none;
  padding: 4px;
  cursor: pointer;
  color: var(--text-color-secondary, #6b7280);
  transition: color 0.2s ease;
  border-radius: 4px;
}

.drawer__close:hover {
  color: var(--text-color, #1f2937);
  background-color: var(--bg-color-hover, #f3f4f6);
}

.drawer__close:focus {
  outline: 2px solid var(--primary-color, #3b82f6);
  outline-offset: 2px;
}

/* 본문 */
.drawer__body {
  padding: 24px;
  overflow-y: auto;
  flex: 1;
}

/* 푸터 */
.drawer__footer {
  padding: 16px 24px;
  border-top: 1px solid var(--border-color, #e5e7eb);
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
  flex-shrink: 0;
}

/* 트랜지션 - 위쪽 */
.drawer-slide-top-enter-active,
.drawer-slide-top-leave-active {
  transition: opacity 0.3s ease;
}

.drawer-slide-top-enter-from,
.drawer-slide-top-leave-to {
  opacity: 0;
}

.drawer-slide-top-enter-active .drawer,
.drawer-slide-top-leave-active .drawer {
  transition: transform 0.3s ease;
}

.drawer-slide-top-enter-from .drawer {
  transform: translateY(-100%);
}

.drawer-slide-top-leave-to .drawer {
  transform: translateY(-100%);
}

/* 트랜지션 - 오른쪽 */
.drawer-slide-right-enter-active,
.drawer-slide-right-leave-active {
  transition: opacity 0.3s ease;
}

.drawer-slide-right-enter-from,
.drawer-slide-right-leave-to {
  opacity: 0;
}

.drawer-slide-right-enter-active .drawer,
.drawer-slide-right-leave-active .drawer {
  transition: transform 0.3s ease;
}

.drawer-slide-right-enter-from .drawer {
  transform: translateX(100%);
}

.drawer-slide-right-leave-to .drawer {
  transform: translateX(100%);
}

/* 트랜지션 - 아래쪽 */
.drawer-slide-bottom-enter-active,
.drawer-slide-bottom-leave-active {
  transition: opacity 0.3s ease;
}

.drawer-slide-bottom-enter-from,
.drawer-slide-bottom-leave-to {
  opacity: 0;
}

.drawer-slide-bottom-enter-active .drawer,
.drawer-slide-bottom-leave-active .drawer {
  transition: transform 0.3s ease;
}

.drawer-slide-bottom-enter-from .drawer {
  transform: translateY(100%);
}

.drawer-slide-bottom-leave-to .drawer {
  transform: translateY(100%);
}

/* 트랜지션 - 왼쪽 */
.drawer-slide-left-enter-active,
.drawer-slide-left-leave-active {
  transition: opacity 0.3s ease;
}

.drawer-slide-left-enter-from,
.drawer-slide-left-leave-to {
  opacity: 0;
}

.drawer-slide-left-enter-active .drawer,
.drawer-slide-left-leave-active .drawer {
  transition: transform 0.3s ease;
}

.drawer-slide-left-enter-from .drawer {
  transform: translateX(-100%);
}

.drawer-slide-left-leave-to .drawer {
  transform: translateX(-100%);
}

/* 다크 모드 */
@media (prefers-color-scheme: dark) {
  .drawer {
    --bg-color: #1f2937;
    --text-color: #f3f4f6;
    --text-color-secondary: #9ca3af;
    --border-color: #374151;
    --bg-color-hover: #374151;
  }
}

/* 모바일 대응 */
@media (max-width: 640px) {
  .drawer--left,
  .drawer--right {
    width: 80vw !important;
    max-width: 400px;
  }
  
  .drawer--top,
  .drawer--bottom {
    height: 50vh !important;
    max-height: 400px;
  }
}
</style>