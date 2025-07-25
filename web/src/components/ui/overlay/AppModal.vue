<template>
  <!-- 모달 오버레이 -->
  <Teleport to="body">
    <transition
      name="app-modal"
      @before-enter="onBeforeEnter"
      @after-enter="onAfterEnter"
      @before-leave="onBeforeLeave"
      @after-leave="onAfterLeave"
    >
      <div
        v-if="modelValue"
        ref="modalRef"
        class="app-modal"
        :class="modalClasses"
        role="dialog"
        :aria-modal="true"
        :aria-labelledby="titleId"
        :aria-describedby="descriptionId"
        :aria-label="ariaLabel"
        @keydown="handleKeydown"
        @click="handleBackdropClick"
      >
        <!-- 배경 오버레이 -->
        <div class="app-modal__backdrop" :class="backdropClasses" />

        <!-- 모달 컨테이너 -->
        <div
          ref="contentRef"
          class="app-modal__container"
          :class="containerClasses"
          @click.stop
        >
          <!-- 모달 헤더 -->
          <header v-if="showHeader" class="app-modal__header">
            <div class="app-modal__title-area">
              <h2
                v-if="title || $slots.title"
                :id="titleId"
                class="app-modal__title"
              >
                <slot name="title">{{ title }}</slot>
              </h2>
            </div>

            <!-- 닫기 버튼 -->
            <button
              v-if="closable"
              type="button"
              class="app-modal__close"
              :aria-label="closeButtonLabel"
              @click="handleClose"
            >
              <svg viewBox="0 0 16 16" fill="currentColor">
                <path d="M4.646 4.646a.5.5 0 0 1 .708 0L8 7.293l2.646-2.647a.5.5 0 0 1 .708.708L8.707 8l2.647 2.646a.5.5 0 0 1-.708.708L8 8.707l-2.646 2.647a.5.5 0 0 1-.708-.708L7.293 8 4.646 5.354a.5.5 0 0 1 0-.708z"/>
              </svg>
            </button>
          </header>

          <!-- 모달 본문 -->
          <main class="app-modal__body" :class="bodyClasses">
            <!-- 설명 텍스트 -->
            <p
              v-if="description"
              :id="descriptionId"
              class="app-modal__description"
            >
              {{ description }}
            </p>

            <!-- 로딩 상태 -->
            <div v-if="loading" class="app-modal__loading">
              <AppSpinner size="medium" />
              <span>{{ loadingText }}</span>
            </div>

            <!-- 본문 내용 -->
            <div v-else class="app-modal__content">
              <slot />
            </div>
          </main>

          <!-- 모달 푸터 -->
          <footer v-if="$slots.footer || showDefaultFooter" class="app-modal__footer">
            <slot name="footer">
              <div v-if="showDefaultFooter" class="app-modal__actions">
                <AppButton
                  v-if="showCancel"
                  variant="outline"
                  @click="handleCancel"
                >
                  {{ cancelText }}
                </AppButton>
                <AppButton
                  v-if="showConfirm"
                  type="primary"
                  :loading="confirmLoading"
                  @click="handleConfirm"
                >
                  {{ confirmText }}
                </AppButton>
              </div>
            </slot>
          </footer>
        </div>
      </div>
    </transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, onUnmounted, ref, useId, watch } from 'vue'
import { useFocusTrap } from '@/composables/useFocusTrap'
import { useAriaLive } from '@/composables/useAriaLive'
import AppSpinner from '../feedback/AppSpinner.vue'
import AppButton from '../form/AppButton.vue'
import type { ModalProps } from '@/types/ui'

interface Props extends ModalProps {
  modelValue?: boolean;
  ariaLabel?: string;
  description?: string;
  loading?: boolean;
  loadingText?: string;
  showDefaultFooter?: boolean;
  showCancel?: boolean;
  showConfirm?: boolean;
  cancelText?: string;
  confirmText?: string;
  confirmLoading?: boolean;
  closeButtonLabel?: string;
  persistent?: boolean; // 외부 클릭으로 닫히지 않음
  fullscreen?: boolean;
  centered?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  size: 'medium',
  closable: true,
  maskClosable: true,
  loading: false,
  autoFocus: true,
  trapFocus: true,
  blockScroll: true,
  loadingText: '로딩 중...',
  showDefaultFooter: false,
  showCancel: true,
  showConfirm: true,
  cancelText: '취소',
  confirmText: '확인',
  confirmLoading: false,
  closeButtonLabel: '모달 닫기',
  persistent: false,
  fullscreen: false,
  centered: true,
})

const emit = defineEmits<{
  'update:modelValue': [value: boolean];
  'close': [];
  'cancel': [];
  'confirm': [];
  'afterEnter': [];
  'afterLeave': [];
  'beforeEnter': [];
  'beforeLeave': [];
}>()

// DOM 참조
const modalRef = ref<HTMLDivElement>()
const contentRef = ref<HTMLDivElement>()

// 고유 ID 생성
const titleId = useId()
const descriptionId = useId()

// 이전 포커스 요소
const previousActiveElement = ref<HTMLElement | null>(null)

// 컴포저블
const { announce } = useAriaLive('modal')

// 포커스 트랩 설정
const { isActive: isFocusTrapped } = useFocusTrap(
  contentRef,
  computed(() => props.modelValue && props.trapFocus),
  {
    initialFocus: () => {
      if (!props.autoFocus) return null

      // 첫 번째 포커스 가능한 요소 찾기
      const firstFocusable = contentRef.value?.querySelector(
        'button, input, textarea, select, a[href], [tabindex]:not([tabindex="-1"])',
      ) as HTMLElement

      return firstFocusable || null
    },
    returnFocus: true,
    escapeKeyDeactivates: props.closable,
    clickOutsideDeactivates: false,
    onActivate: () => {
      // 스크린 리더에 모달 열림 알림
      if (props.title) {
        announce(`${props.title} 모달이 열렸습니다`)
      } else {
        announce('모달이 열렸습니다')
      }
    },
    onDeactivate: () => {
      // 스크린 리더에 모달 닫힘 알림
      announce('모달이 닫혔습니다')
    },
  },
)

// 계산된 속성들
const showHeader = computed(() => {
  return props.title || props.$slots?.title || props.closable
})

const modalClasses = computed(() => ({
  'app-modal--fullscreen': props.fullscreen,
  'app-modal--centered': props.centered && !props.fullscreen,
  'app-modal--loading': props.loading,
}))

const backdropClasses = computed(() => ({
  'app-modal__backdrop--persistent': props.persistent,
}))

const containerClasses = computed(() => ({
  'app-modal__container--small': props.size === 'small',
  'app-modal__container--medium': props.size === 'medium',
  'app-modal__container--large': props.size === 'large',
  'app-modal__container--huge': props.size === 'huge',
  'app-modal__container--fullscreen': props.fullscreen,
}))

const bodyClasses = computed(() => ({
  'app-modal__body--loading': props.loading,
  'app-modal__body--no-header': !showHeader.value,
  'app-modal__body--no-footer': !props.$slots?.footer && !props.showDefaultFooter,
}))

// 스크롤 차단 관리
const manageBodyScroll = (block: boolean) => {
  if (!props.blockScroll) return

  if (block) {
    document.body.style.overflow = 'hidden'
    document.body.style.paddingRight = `${window.innerWidth - document.documentElement.clientWidth}px`
  } else {
    document.body.style.overflow = ''
    document.body.style.paddingRight = ''
  }
}

// 이벤트 핸들러들
const handleClose = () => {
  if (!props.closable) return

  emit('update:modelValue', false)
  emit('close')
}

const handleCancel = () => {
  emit('update:modelValue', false)
  emit('cancel')
}

const handleConfirm = () => {
  emit('confirm')
  // 부모 컴포넌트에서 명시적으로 닫아야 함
}

const handleBackdropClick = (event: MouseEvent) => {
  if (event.target === modalRef.value && props.maskClosable && !props.persistent) {
    handleClose()
  }
}

const handleKeydown = (event: KeyboardEvent) => {
  if (event.key === 'Escape' && props.closable) {
    handleClose()
  }
}

// 트랜지션 이벤트 핸들러들
const onBeforeEnter = () => {
  previousActiveElement.value = document.activeElement as HTMLElement
  manageBodyScroll(true)
  emit('beforeEnter')
}

const onAfterEnter = () => {
  emit('afterEnter')
  props.onAfterEnter?.()
}

const onBeforeLeave = () => {
  emit('beforeLeave')
}

const onAfterLeave = () => {
  manageBodyScroll(false)

  // 포커스 복귀
  if (previousActiveElement.value) {
    nextTick(() => {
      previousActiveElement.value?.focus()
      previousActiveElement.value = null
    })
  }

  emit('afterLeave')
  props.onAfterLeave?.()
}

// 컴포넌트 언마운트 시 정리
onUnmounted(() => {
  manageBodyScroll(false)
})

// 외부에서 접근 가능한 메서드들
defineExpose({
  close: handleClose,
  modalRef,
  contentRef,
})
</script>

<style lang="scss" scoped>
.app-modal {
  @apply fixed inset-0 z-50 flex items-center justify-center;

  &--fullscreen {
    @apply items-stretch justify-stretch;
  }

  &--centered {
    @apply items-center justify-center;
  }

  &--loading {
    pointer-events: none;
  }

  &__backdrop {
    @apply absolute inset-0 bg-black bg-opacity-50;
    @apply transition-opacity duration-300;

    &--persistent {
      @apply cursor-not-allowed;
    }
  }

  &__container {
    @apply relative bg-white rounded-lg shadow-xl;
    @apply max-h-full overflow-hidden;
    @apply transform transition-all duration-300;
    @apply flex flex-col;

    &--small {
      @apply w-full max-w-sm mx-4;
    }

    &--medium {
      @apply w-full max-w-md mx-4;
    }

    &--large {
      @apply w-full max-w-2xl mx-4;
    }

    &--huge {
      @apply w-full max-w-4xl mx-4;
    }

    &--fullscreen {
      @apply w-full h-full max-w-none mx-0 rounded-none;
    }
  }

  &__header {
    @apply flex items-center justify-between;
    @apply px-6 py-4 border-b border-gray-200;
    @apply flex-shrink-0;
  }

  &__title-area {
    @apply flex-1 min-w-0;
  }

  &__title {
    @apply text-lg font-semibold text-gray-900;
    @apply m-0 truncate;
  }

  &__close {
    @apply ml-4 p-2 -mr-2 text-gray-400 hover:text-gray-600;
    @apply rounded-md transition-colors duration-200;
    @apply focus:outline-none focus:ring-2 focus:ring-blue-500;

    svg {
      @apply w-5 h-5;
    }
  }

  &__body {
    @apply flex-1 overflow-y-auto;
    @apply px-6 py-4;

    &--loading {
      @apply flex items-center justify-center;
    }

    &--no-header {
      @apply pt-6;
    }

    &--no-footer {
      @apply pb-6;
    }
  }

  &__description {
    @apply text-gray-600 mb-4;
    @apply m-0;
  }

  &__loading {
    @apply flex flex-col items-center gap-3;
    @apply text-gray-600;
  }

  &__content {
    @apply w-full;
  }

  &__footer {
    @apply px-6 py-4 border-t border-gray-200;
    @apply flex-shrink-0;
  }

  &__actions {
    @apply flex items-center justify-end gap-3;
  }
}

// 모달 애니메이션
.app-modal-enter-active,
.app-modal-leave-active {
  .app-modal__backdrop {
    @apply transition-opacity duration-300;
  }

  .app-modal__container {
    @apply transition-all duration-300;
  }
}

.app-modal-enter-from,
.app-modal-leave-to {
  .app-modal__backdrop {
    @apply opacity-0;
  }

  .app-modal__container {
    @apply opacity-0 scale-95 translate-y-4;
  }
}

// 다크 모드
.dark .app-modal {
  &__backdrop {
    @apply bg-gray-900 bg-opacity-75;
  }

  &__container {
    @apply bg-gray-800 border-gray-700;
  }

  &__header {
    @apply border-gray-700;
  }

  &__title {
    @apply text-gray-100;
  }

  &__close {
    @apply text-gray-500 hover:text-gray-300;
  }

  &__description {
    @apply text-gray-400;
  }

  &__footer {
    @apply border-gray-700;
  }

  &__loading {
    @apply text-gray-400;
  }
}

// 접근성: 고대비 모드
@media (prefers-contrast: high) {
  .app-modal {
    &__backdrop {
      @apply bg-black bg-opacity-80;
    }

    &__container {
      @apply border-2 border-gray-900;
    }

    &__close {
      &:focus {
        @apply ring-4;
      }
    }
  }
}

// 접근성: 움직임 감소
@media (prefers-reduced-motion: reduce) {
  .app-modal-enter-active,
  .app-modal-leave-active {
    .app-modal__backdrop,
    .app-modal__container {
      @apply transition-none;
    }
  }

  .app-modal-enter-from,
  .app-modal-leave-to {
    .app-modal__container {
      transform: none;
    }
  }
}

// 모바일 최적화
@media (max-width: 640px) {
  .app-modal {
    &__container {
      &--small,
      &--medium,
      &--large,
      &--huge {
        @apply mx-2 max-h-[calc(100vh-1rem)];
      }
    }

    &__header {
      @apply px-4 py-3;
    }

    &__body {
      @apply px-4 py-3;

      &--no-header {
        @apply pt-4;
      }

      &--no-footer {
        @apply pb-4;
      }
    }

    &__footer {
      @apply px-4 py-3;
    }

    &__actions {
      @apply flex-col-reverse gap-2;

      :deep(.app-button) {
        @apply w-full;
      }
    }
  }
}
</style>