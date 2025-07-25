<template>
  <!-- 토스트 컨테이너 -->
  <Teleport to="body">
    <div
      v-if="toasts.length > 0"
      class="app-toast-container"
      :class="containerClasses"
      role="region"
      aria-label="알림 메시지"
      aria-live="polite"
    >
      <TransitionGroup
        name="app-toast"
        tag="div"
        class="app-toast-list"
        @before-enter="onBeforeEnter"
        @after-leave="onAfterLeave"
      >
        <div
          v-for="toast in toasts"
          :key="toast.id"
          class="app-toast"
          :class="getToastClasses(toast)"
          role="alert"
          :aria-live="toast.type === 'error' ? 'assertive' : 'polite'"
          :aria-atomic="true"
        >
          <!-- 토스트 아이콘 -->
          <div v-if="toast.showIcon" class="app-toast__icon">
            <svg viewBox="0 0 16 16" fill="currentColor">
              <!-- 성공 아이콘 -->
              <path
                v-if="toast.type === 'success'"
                d="M13.854 3.646a.5.5 0 0 1 0 .708l-7 7a.5.5 0 0 1-.708 0l-3.5-3.5a.5.5 0 1 1 .708-.708L6.5 10.293l6.646-6.647a.5.5 0 0 1 .708 0z"
              />
              <!-- 에러 아이콘 -->
              <path
                v-else-if="toast.type === 'error'"
                d="M8 1a7 7 0 1 0 0 14A7 7 0 0 0 8 1zM6.646 5.646a.5.5 0 1 1 .708.708L8 7.293l.646-.647a.5.5 0 0 1 .708.708L8.707 8l.647.646a.5.5 0 0 1-.708.708L8 8.707l-.646.647a.5.5 0 0 1-.708-.708L7.293 8l-.647-.646z"
              />
              <!-- 경고 아이콘 -->
              <path
                v-else-if="toast.type === 'warning'"
                d="M8 1a7 7 0 1 0 0 14A7 7 0 0 0 8 1zM7.5 4a.5.5 0 0 1 1 0v3a.5.5 0 0 1-1 0V4zM8 10a.75.75 0 1 1 0 1.5.75.75 0 0 1 0-1.5z"
              />
              <!-- 정보 아이콘 -->
              <path
                v-else-if="toast.type === 'info'"
                d="M8 1a7 7 0 1 0 0 14A7 7 0 0 0 8 1zm-.5 4a.5.5 0 0 1 1 0v3a.5.5 0 0 1-1 0V5zM8 3.5a.75.75 0 1 1 0 1.5.75.75 0 0 1 0-1.5z"
              />
              <!-- 로딩 아이콘 -->
              <g v-else-if="toast.type === 'loading'">
                <circle cx="8" cy="8" r="6" fill="none" stroke="currentColor" stroke-width="2" class="app-toast__loading-circle"/>
              </g>
            </svg>
          </div>

          <!-- 토스트 내용 -->
          <div class="app-toast__content">
            <div v-if="toast.title" class="app-toast__title">
              {{ toast.title }}
            </div>
            <div class="app-toast__message">
              {{ toast.message }}
            </div>
          </div>

          <!-- 액션 버튼들 -->
          <div v-if="toast.actions && toast.actions.length > 0" class="app-toast__actions">
            <button
              v-for="action in toast.actions"
              :key="action.label"
              type="button"
              class="app-toast__action-btn"
              @click="handleActionClick(toast, action)"
            >
              {{ action.label }}
            </button>
          </div>

          <!-- 닫기 버튼 -->
          <button
            v-if="toast.closable"
            type="button"
            class="app-toast__close"
            :aria-label="`${toast.message} 알림 닫기`"
            @click="removeToast(toast.id)"
          >
            <svg viewBox="0 0 16 16" fill="currentColor">
              <path d="M4.646 4.646a.5.5 0 0 1 .708 0L8 7.293l2.646-2.647a.5.5 0 0 1 .708.708L8.707 8l2.647 2.646a.5.5 0 0 1-.708.708L8 8.707l-2.646 2.647a.5.5 0 0 1-.708-.708L7.293 8 4.646 5.354a.5.5 0 0 1 0-.708z"/>
            </svg>
          </button>

          <!-- 진행 바 -->
          <div
            v-if="toast.showProgress && toast.duration && toast.duration > 0"
            class="app-toast__progress"
          >
            <div
              class="app-toast__progress-bar"
              :style="{ animationDuration: `${toast.duration}ms` }"
            />
          </div>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useAriaLive } from '@/composables/useAriaLive'

export type ToastType = 'success' | 'error' | 'warning' | 'info' | 'loading';
export type ToastPosition = 'top-left' | 'top-center' | 'top-right' | 'bottom-left' | 'bottom-center' | 'bottom-right';

export interface ToastAction {
  label: string;
  action: () => void;
  style?: 'primary' | 'secondary';
}

export interface ToastOptions {
  id?: string;
  type?: ToastType;
  title?: string;
  message: string;
  duration?: number;
  closable?: boolean;
  showIcon?: boolean;
  showProgress?: boolean;
  actions?: ToastAction[];
  onClose?: () => void;
  onClick?: () => void;
}

interface ToastItem extends Required<Omit<ToastOptions, 'onClose' | 'onClick'>> {
  id: string;
  createdAt: number;
  onClose?: () => void;
  onClick?: () => void;
  timeoutId?: NodeJS.Timeout;
}

interface Props {
  position?: ToastPosition;
  maxToasts?: number;
  defaultDuration?: number;
  pauseOnHover?: boolean;
  pauseOnFocus?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  position: 'top-right',
  maxToasts: 5,
  defaultDuration: 4000,
  pauseOnHover: true,
  pauseOnFocus: true,
})

const emit = defineEmits<{
  'toast-added': [toast: ToastItem];
  'toast-removed': [toastId: string];
}>()

// 토스트 목록
const toasts = ref<ToastItem[]>([])
let toastIdCounter = 0

// ARIA 라이브 리전
const { announce } = useAriaLive('toast')

// 컨테이너 클래스 계산
const containerClasses = computed(() => ({
  'app-toast-container--top-left': props.position === 'top-left',
  'app-toast-container--top-center': props.position === 'top-center',
  'app-toast-container--top-right': props.position === 'top-right',
  'app-toast-container--bottom-left': props.position === 'bottom-left',
  'app-toast-container--bottom-center': props.position === 'bottom-center',
  'app-toast-container--bottom-right': props.position === 'bottom-right',
}))

// 토스트 클래스 계산
const getToastClasses = (toast: ToastItem) => ({
  'app-toast--success': toast.type === 'success',
  'app-toast--error': toast.type === 'error',
  'app-toast--warning': toast.type === 'warning',
  'app-toast--info': toast.type === 'info',
  'app-toast--loading': toast.type === 'loading',
  'app-toast--with-progress': toast.showProgress && toast.duration > 0,
  'app-toast--with-actions': toast.actions && toast.actions.length > 0,
})

// 토스트 추가
const addToast = (options: ToastOptions): string => {
  const id = options.id || `toast-${++toastIdCounter}`

  // 기존 토스트 중복 제거
  if (options.id) {
    removeToast(options.id)
  }

  const toast: ToastItem = {
    id,
    type: options.type || 'info',
    title: options.title || '',
    message: options.message,
    duration: options.duration ?? props.defaultDuration,
    closable: options.closable ?? true,
    showIcon: options.showIcon ?? true,
    showProgress: options.showProgress ?? false,
    actions: options.actions || [],
    createdAt: Date.now(),
    onClose: options.onClose,
    onClick: options.onClick,
  }

  // 최대 개수 제한
  if (toasts.value.length >= props.maxToasts) {
    const oldestToast = toasts.value[0]
    removeToast(oldestToast.id)
  }

  // 토스트 추가
  toasts.value.push(toast)

  // 자동 제거 타이머 설정
  if (toast.duration > 0) {
    toast.timeoutId = setTimeout(() => {
      removeToast(toast.id)
    }, toast.duration)
  }

  // 스크린 리더에 알림
  const announceText = toast.title
    ? `${toast.title}: ${toast.message}`
    : toast.message

  if (toast.type === 'error') {
    announce(announceText, { politeness: 'assertive' })
  } else {
    announce(announceText, { politeness: 'polite' })
  }

  emit('toast-added', toast)
  return id
}

// 토스트 제거
const removeToast = (id: string): void => {
  const index = toasts.value.findIndex(toast => toast.id === id)
  if (index === -1) return

  const toast = toasts.value[index]

  // 타이머 클리어
  if (toast.timeoutId) {
    clearTimeout(toast.timeoutId)
  }

  // 콜백 실행
  toast.onClose?.()

  // 토스트 제거
  toasts.value.splice(index, 1)

  emit('toast-removed', id)
}

// 모든 토스트 제거
const clearAll = (): void => {
  toasts.value.forEach(toast => {
    if (toast.timeoutId) {
      clearTimeout(toast.timeoutId)
    }
    toast.onClose?.()
  })

  toasts.value = []
}

// 토스트 일시정지/재개
const pauseToast = (id: string): void => {
  const toast = toasts.value.find(t => t.id === id)
  if (toast?.timeoutId) {
    clearTimeout(toast.timeoutId)
    toast.timeoutId = undefined
  }
}

const resumeToast = (id: string): void => {
  const toast = toasts.value.find(t => t.id === id)
  if (toast && !toast.timeoutId && toast.duration > 0) {
    const elapsed = Date.now() - toast.createdAt
    const remaining = Math.max(0, toast.duration - elapsed)

    if (remaining > 0) {
      toast.timeoutId = setTimeout(() => {
        removeToast(toast.id)
      }, remaining)
    } else {
      removeToast(toast.id)
    }
  }
}

// 이벤트 핸들러들
const handleActionClick = (toast: ToastItem, action: ToastAction): void => {
  action.action()
  // 액션 실행 후 토스트 제거 (필요에 따라)
  removeToast(toast.id)
}

const handleToastClick = (toast: ToastItem): void => {
  toast.onClick?.()
}

const handleMouseEnter = (toast: ToastItem): void => {
  if (props.pauseOnHover) {
    pauseToast(toast.id)
  }
}

const handleMouseLeave = (toast: ToastItem): void => {
  if (props.pauseOnHover) {
    resumeToast(toast.id)
  }
}

const handleFocus = (toast: ToastItem): void => {
  if (props.pauseOnFocus) {
    pauseToast(toast.id)
  }
}

const handleBlur = (toast: ToastItem): void => {
  if (props.pauseOnFocus) {
    resumeToast(toast.id)
  }
}

// 트랜지션 이벤트
const onBeforeEnter = (el: Element): void => {
  const element = el as HTMLElement
  element.style.height = '0'
  element.style.opacity = '0'
}

const onAfterLeave = (el: Element): void => {
  const element = el as HTMLElement
  element.style.height = ''
  element.style.opacity = ''
}

// 편의 메서드들
const success = (message: string, options?: Omit<ToastOptions, 'message' | 'type'>): string => {
  return addToast({ ...options, message, type: 'success' })
}

const error = (message: string, options?: Omit<ToastOptions, 'message' | 'type'>): string => {
  return addToast({ ...options, message, type: 'error' })
}

const warning = (message: string, options?: Omit<ToastOptions, 'message' | 'type'>): string => {
  return addToast({ ...options, message, type: 'warning' })
}

const info = (message: string, options?: Omit<ToastOptions, 'message' | 'type'>): string => {
  return addToast({ ...options, message, type: 'info' })
}

const loading = (message: string, options?: Omit<ToastOptions, 'message' | 'type'>): string => {
  return addToast({
    ...options,
    message,
    type: 'loading',
    duration: 0, // 로딩 토스트는 수동으로 제거
    closable: false,
  })
}

// 컴포넌트 언마운트 시 정리
onUnmounted(() => {
  clearAll()
})

defineExpose({
  addToast,
  removeToast,
  clearAll,
  success,
  error,
  warning,
  info,
  loading,
  toasts,
})
</script>

<style lang="scss" scoped>
.app-toast-container {
  @apply fixed z-50 flex flex-col gap-2;
  @apply max-w-sm w-full;

  // 위치별 스타일
  &--top-left {
    @apply top-4 left-4;
  }

  &--top-center {
    @apply top-4 left-1/2 transform -translate-x-1/2;
  }

  &--top-right {
    @apply top-4 right-4;
  }

  &--bottom-left {
    @apply bottom-4 left-4;
  }

  &--bottom-center {
    @apply bottom-4 left-1/2 transform -translate-x-1/2;
  }

  &--bottom-right {
    @apply bottom-4 right-4;
  }
}

.app-toast-list {
  @apply flex flex-col gap-2;
}

.app-toast {
  @apply relative flex items-start gap-3;
  @apply bg-white border border-gray-200 rounded-lg shadow-lg;
  @apply p-4 min-h-[60px];
  @apply cursor-pointer transition-all duration-200;
  @apply hover:shadow-xl;

  &:focus-within {
    @apply ring-2 ring-blue-500 ring-opacity-50;
  }

  // 타입별 색상
  &--success {
    @apply border-green-200 bg-green-50;

    .app-toast__icon {
      @apply text-green-600;
    }
  }

  &--error {
    @apply border-red-200 bg-red-50;

    .app-toast__icon {
      @apply text-red-600;
    }
  }

  &--warning {
    @apply border-yellow-200 bg-yellow-50;

    .app-toast__icon {
      @apply text-yellow-600;
    }
  }

  &--info {
    @apply border-blue-200 bg-blue-50;

    .app-toast__icon {
      @apply text-blue-600;
    }
  }

  &--loading {
    @apply border-gray-200 bg-gray-50;

    .app-toast__icon {
      @apply text-gray-600;
    }
  }

  // 진행 바가 있는 경우
  &--with-progress {
    @apply pb-2;
  }

  // 액션이 있는 경우
  &--with-actions {
    @apply flex-col items-stretch;
  }

  // 컴포넌트 요소들
  &__icon {
    @apply flex-shrink-0 flex items-center justify-center;
    @apply w-5 h-5 mt-0.5;

    svg {
      @apply w-full h-full;
    }
  }

  &__loading-circle {
    animation: spin 1s linear infinite;
    transform-origin: center;
  }

  &__content {
    @apply flex-1 min-w-0;
  }

  &__title {
    @apply font-medium text-gray-900 mb-1;
    @apply text-sm leading-5;
  }

  &__message {
    @apply text-gray-700 text-sm leading-5;
    @apply break-words;
  }

  &__actions {
    @apply flex items-center gap-2 mt-3;
  }

  &__action-btn {
    @apply px-3 py-1 text-xs font-medium rounded;
    @apply bg-white border border-gray-300;
    @apply hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500;
    @apply transition-colors duration-200;
  }

  &__close {
    @apply flex-shrink-0 p-1 -mt-1 -mr-1;
    @apply text-gray-400 hover:text-gray-600;
    @apply rounded focus:outline-none focus:ring-2 focus:ring-blue-500;
    @apply transition-colors duration-200;

    svg {
      @apply w-4 h-4;
    }
  }

  &__progress {
    @apply absolute bottom-0 left-0 right-0;
    @apply h-1 bg-gray-200 rounded-b-lg overflow-hidden;
  }

  &__progress-bar {
    @apply h-full bg-blue-500;
    @apply animate-shrink-width;
  }
}

// 토스트 애니메이션
.app-toast-enter-active,
.app-toast-leave-active {
  @apply transition-all duration-300;
}

.app-toast-enter-from {
  @apply opacity-0 transform translate-x-full;
}

.app-toast-leave-to {
  @apply opacity-0 transform scale-95 translate-x-full;
}

.app-toast-move {
  @apply transition-transform duration-300;
}

// 진행 바 애니메이션
@keyframes shrink-width {
  from {
    width: 100%;
  }
  to {
    width: 0%;
  }
}

.animate-shrink-width {
  animation: shrink-width linear forwards;
}

// 로딩 스피너 애니메이션
@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

// 다크 모드
.dark .app-toast {
  @apply bg-gray-800 border-gray-700;

  &--success {
    @apply border-green-800 bg-green-900;
  }

  &--error {
    @apply border-red-800 bg-red-900;
  }

  &--warning {
    @apply border-yellow-800 bg-yellow-900;
  }

  &--info {
    @apply border-blue-800 bg-blue-900;
  }

  &--loading {
    @apply border-gray-600 bg-gray-800;
  }

  .app-toast__title {
    @apply text-gray-100;
  }

  .app-toast__message {
    @apply text-gray-300;
  }

  .app-toast__action-btn {
    @apply bg-gray-700 border-gray-600 text-gray-200;
    @apply hover:bg-gray-600;
  }

  .app-toast__close {
    @apply text-gray-500 hover:text-gray-300;
  }

  .app-toast__progress {
    @apply bg-gray-700;
  }
}

// 접근성: 움직임 감소
@media (prefers-reduced-motion: reduce) {
  .app-toast-enter-active,
  .app-toast-leave-active,
  .app-toast-move {
    @apply transition-none;
  }

  .app-toast-enter-from,
  .app-toast-leave-to {
    transform: none;
  }

  .app-toast__loading-circle {
    animation: none;
  }

  .animate-shrink-width {
    animation: none;
    width: 0%;
  }
}

// 모바일 최적화
@media (max-width: 640px) {
  .app-toast-container {
    @apply max-w-none left-2 right-2;

    &--top-center,
    &--bottom-center {
      @apply left-2 right-2 transform-none;
    }

    &--top-left,
    &--top-right {
      @apply left-2 right-2;
    }

    &--bottom-left,
    &--bottom-right {
      @apply left-2 right-2;
    }
  }

  .app-toast {
    &__actions {
      @apply flex-col gap-1;
    }

    &__action-btn {
      @apply w-full;
    }
  }
}
</style>