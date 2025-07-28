<template>
  <Teleport to="body">
    <Transition
      :name="transitionName"
      @after-enter="onAfterEnter"
      @after-leave="onAfterLeave"
    >
      <div
        v-if="visible"
        ref="modalRef"
        class="modal-container"
        :style="{ zIndex }"
        @click="handleBackdropClick"
      >
        <!-- 백드롭 -->
        <div class="modal-backdrop" :class="{ 'modal-backdrop--transparent': !mask }" />

        <!-- 모달 본체 -->
        <div
          class="modal"
          :class="[
            `modal--${size}`,
            { 'modal--fullscreen': fullscreen }
          ]"
          :style="modalStyle"
          @click.stop
        >
          <!-- 헤더 -->
          <div v-if="!hideHeader" class="modal__header">
            <slot name="header">
              <h2 class="modal__title">{{ title }}</h2>
            </slot>
            <button
              v-if="closable"
              type="button"
              class="modal__close"
              @click="handleClose"
              aria-label="닫기"
            >
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                <path d="M18 6L6 18M6 6l12 12" stroke-width="2" stroke-linecap="round"/>
              </svg>
            </button>
          </div>

          <!-- 본문 -->
          <div class="modal__body" :style="bodyStyle">
            <slot />
          </div>

          <!-- 푸터 -->
          <div v-if="!hideFooter && $slots.footer" class="modal__footer">
            <slot name="footer" />
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useOverlay } from '../composables/useOverlayManager'
import { useFocusTrap } from '../composables/useFocusTrap'

export interface ModalProps {
  // 표시 상태
  visible: boolean
  // 제목
  title?: string
  // 크기
  size?: 'small' | 'medium' | 'large' | 'xlarge'
  // 전체화면 모드
  fullscreen?: boolean
  // 너비 (px 또는 %)
  width?: string | number
  // 높이 (px 또는 %)
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
  // 트랜지션 이름
  transitionName?: string
  // 바디 스타일
  bodyStyle?: Record<string, any>
}

const props = withDefaults(defineProps<ModalProps>(), {
  size: 'medium',
  fullscreen: false,
  closable: true,
  mask: true,
  closeOnEsc: true,
  closeOnClickOutside: true,
  hideHeader: false,
  hideFooter: false,
  transitionName: 'modal-fade',
})

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'close': []
  'after-enter': []
  'after-leave': []
}>()

// 모달 레퍼런스
const modalRef = ref<HTMLElement>()

// 고유 ID 생성
const modalId = `modal-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`

// 오버레이 관리
const { zIndex } = useOverlay({
  id: modalId,
  type: 'modal',
  closeOnEsc: props.closeOnEsc,
  closeOnClickOutside: props.closeOnClickOutside,
  onClose: () => handleClose(),
})

// 포커스 트랩
useFocusTrap(modalRef, {
  autoFocus: true,
  restoreFocus: true,
})

// 모달 크기 계산
const modalStyle = computed(() => {
  const style: Record<string, any> = {}

  if (props.width) {
    style.width = typeof props.width === 'number' ? `${props.width}px` : props.width
  }

  if (props.height) {
    style.height = typeof props.height === 'number' ? `${props.height}px` : props.height
  }

  return style
})

// 크기별 기본 너비
const sizeWidths = {
  small: '400px',
  medium: '600px',
  large: '800px',
  xlarge: '1200px',
}

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
    // 다른 모달이 열려있을 수 있으므로 체크
    nextTick(() => {
      const hasOtherModals = document.querySelectorAll('.modal-container').length > 0
      if (!hasOtherModals) {
        document.body.style.overflow = ''
      }
    })
  }
})
</script>

<style scoped>
/* 컨테이너 */
.modal-container {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}

/* 백드롭 */
.modal-backdrop {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.5);
  transition: opacity 0.3s ease;
}

.modal-backdrop--transparent {
  background-color: transparent;
}

/* 모달 본체 */
.modal {
  position: relative;
  background-color: var(--bg-color, #ffffff);
  border-radius: 8px;
  box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04);
  display: flex;
  flex-direction: column;
  max-height: calc(100vh - 48px);
  transition: all 0.3s ease;
}

/* 크기 변형 */
.modal--small {
  width: 400px;
}

.modal--medium {
  width: 600px;
}

.modal--large {
  width: 800px;
}

.modal--xlarge {
  width: 1200px;
}

.modal--fullscreen {
  width: 100vw;
  height: 100vh;
  max-height: 100vh;
  border-radius: 0;
}

/* 헤더 */
.modal__header {
  padding: 20px 24px;
  border-bottom: 1px solid var(--border-color, #e5e7eb);
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
}

.modal__title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--text-color, #1f2937);
}

.modal__close {
  background: none;
  border: none;
  padding: 4px;
  cursor: pointer;
  color: var(--text-color-secondary, #6b7280);
  transition: color 0.2s ease;
  border-radius: 4px;
}

.modal__close:hover {
  color: var(--text-color, #1f2937);
  background-color: var(--bg-color-hover, #f3f4f6);
}

.modal__close:focus {
  outline: 2px solid var(--primary-color, #3b82f6);
  outline-offset: 2px;
}

/* 본문 */
.modal__body {
  padding: 24px;
  overflow-y: auto;
  flex: 1;
}

/* 푸터 */
.modal__footer {
  padding: 16px 24px;
  border-top: 1px solid var(--border-color, #e5e7eb);
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
  flex-shrink: 0;
}

/* 트랜지션 */
.modal-fade-enter-active,
.modal-fade-leave-active {
  transition: opacity 0.3s ease;
}

.modal-fade-enter-from,
.modal-fade-leave-to {
  opacity: 0;
}

.modal-fade-enter-active .modal,
.modal-fade-leave-active .modal {
  transition: transform 0.3s ease;
}

.modal-fade-enter-from .modal {
  transform: scale(0.9);
}

.modal-fade-leave-to .modal {
  transform: scale(0.9);
}

/* 다크 모드 */
@media (prefers-color-scheme: dark) {
  .modal {
    --bg-color: #1f2937;
    --text-color: #f3f4f6;
    --text-color-secondary: #9ca3af;
    --border-color: #374151;
    --bg-color-hover: #374151;
  }
}

/* 모바일 대응 */
@media (max-width: 640px) {
  .modal-container {
    padding: 16px;
  }

  .modal {
    width: 100%;
    max-width: calc(100vw - 32px);
  }

  .modal--fullscreen {
    max-width: 100vw;
  }
}
</style>