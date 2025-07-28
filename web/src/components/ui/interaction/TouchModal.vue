<template>
  <Teleport to="body">
    <div
      v-if="visible"
      class="touch-modal"
      :class="modalClasses"
      @click="handleBackdropClick"
    >
      <!-- 배경 오버레이 -->
      <div class="touch-modal__backdrop" />

      <!-- 모달 컨테이너 -->
      <div
        ref="modalRef"
        class="touch-modal__container"
        :style="containerStyle"
        @click.stop
        @touchstart="handleTouchStart"
        @touchmove="handleTouchMove"
        @touchend="handleTouchEnd"
      >
        <!-- 드래그 핸들 -->
        <div v-if="draggable" class="touch-modal__handle">
          <div class="touch-modal__handle-bar"></div>
        </div>

        <!-- 모달 헤더 -->
        <header v-if="$slots.header || title" class="touch-modal__header">
          <slot name="header">
            <div class="touch-modal__header-content">
              <h2 v-if="title" class="touch-modal__title">{{ title }}</h2>
              <button
                v-if="showCloseButton"
                class="touch-modal__close-button"
                @click="handleClose"
                aria-label="모달 닫기"
              >
                <svg viewBox="0 0 24 24" fill="none">
                  <path
                    d="M18 6L6 18M6 6l12 12"
                    stroke="currentColor"
                    stroke-width="2"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  />
                </svg>
              </button>
            </div>
          </slot>
        </header>

        <!-- 모달 콘텐츠 -->
        <div class="touch-modal__content">
          <slot />
        </div>

        <!-- 모달 푸터 -->
        <footer v-if="$slots.footer" class="touch-modal__footer">
          <slot name="footer" />
        </footer>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'

interface Props {
  visible: boolean
  title?: string
  position?: 'center' | 'bottom' | 'top' | 'fullscreen'
  size?: 'small' | 'medium' | 'large' | 'auto'
  draggable?: boolean
  closeOnBackdrop?: boolean
  showCloseButton?: boolean
  maxHeight?: string
  preventScroll?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  position: 'bottom',
  size: 'medium',
  draggable: true,
  closeOnBackdrop: true,
  showCloseButton: true,
  maxHeight: '80vh',
  preventScroll: true,
})

const emit = defineEmits<{
  'update:visible': [visible: boolean]
  close: []
  'before-close': []
  'after-open': []
  'after-close': []
}>()

const modalRef = ref<HTMLElement>()

// 드래그 상태
const isDragging = ref(false)
const startY = ref(0)
const currentY = ref(0)
const translateY = ref(0)
const isClosing = ref(false)

// 모달 클래스
const modalClasses = computed(() => ({
  [`touch-modal--${props.position}`]: true,
  [`touch-modal--${props.size}`]: true,
  'touch-modal--draggable': props.draggable,
  'touch-modal--closing': isClosing.value,
}))

// 컨테이너 스타일
const containerStyle = computed(() => {
  const style: Record<string, string> = {
    maxHeight: props.maxHeight,
  }

  if (isDragging.value && translateY.value > 0) {
    style.transform = `translateY(${translateY.value}px)`
    style.transition = 'none'
  }

  return style
})

// 터치 이벤트 핸들러
const handleTouchStart = (event: TouchEvent) => {
  if (!props.draggable || props.position !== 'bottom') return

  const touch = event.touches[0]
  startY.value = touch.clientY
  currentY.value = touch.clientY
  isDragging.value = true
}

const handleTouchMove = (event: TouchEvent) => {
  if (!isDragging.value || !props.draggable) return

  const touch = event.touches[0]
  currentY.value = touch.clientY

  const deltaY = currentY.value - startY.value

  // 아래쪽으로만 드래그 허용
  if (deltaY > 0) {
    translateY.value = deltaY

    // 스크롤 방지
    event.preventDefault()
  }
}

const handleTouchEnd = () => {
  if (!isDragging.value || !props.draggable) return

  isDragging.value = false

  // 임계값을 넘으면 모달 닫기
  const threshold = 100
  if (translateY.value > threshold) {
    handleClose()
  } else {
    // 원래 위치로 복귀
    translateY.value = 0
  }
}

// 배경 클릭 핸들러
const handleBackdropClick = () => {
  if (props.closeOnBackdrop) {
    handleClose()
  }
}

// 모달 닫기
const handleClose = () => {
  emit('before-close')

  isClosing.value = true
  translateY.value = 0

  setTimeout(() => {
    emit('update:visible', false)
    emit('close')
    isClosing.value = false

    nextTick(() => {
      emit('after-close')
    })
  }, 300)
}

// 모달 열기 애니메이션
const handleOpen = () => {
  nextTick(() => {
    emit('after-open')
  })
}

// 스크롤 방지
const preventBodyScroll = () => {
  if (!props.preventScroll) return

  document.body.style.overflow = 'hidden'
  document.body.style.paddingRight = '0px' // 스크롤바 공간 보정
}

const restoreBodyScroll = () => {
  if (!props.preventScroll) return

  document.body.style.overflow = ''
  document.body.style.paddingRight = ''
}

// 키보드 이벤트 핸들러
const handleKeydown = (event: KeyboardEvent) => {
  if (event.key === 'Escape' && props.visible) {
    handleClose()
  }
}

// 모달 가시성 변화 감지
watch(
  () => props.visible,
  (visible) => {
    if (visible) {
      preventBodyScroll()
      handleOpen()
      document.addEventListener('keydown', handleKeydown)
    } else {
      restoreBodyScroll()
      document.removeEventListener('keydown', handleKeydown)
    }
  },
  { immediate: true },
)

// 컴포넌트 해제 시 정리
onBeforeUnmount(() => {
  restoreBodyScroll()
  document.removeEventListener('keydown', handleKeydown)
})

// 공개 메서드
const close = () => {
  handleClose()
}

defineExpose({
  close,
})
</script>

<style lang="scss" scoped>
@use '@/styles/variables' as *;
@use '@/styles/mixins' as *;

.touch-modal {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: $z-modal;
  @include flex-center;

  &__backdrop {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.5);
    backdrop-filter: blur(4px);
    animation: modal-backdrop-enter 0.3s ease;
  }

  &__container {
    position: relative;
    background: $light-bg-primary;
    border-radius: $border-radius-2xl $border-radius-2xl 0 0;
    box-shadow: $shadow-xl;
    overflow: hidden;
    width: 100%;
    max-width: 480px;
    animation: modal-enter 0.3s ease;

    .dark & {
      background: $dark-bg-secondary;
    }
  }

  // 위치별 스타일
  &--center {
    .touch-modal__container {
      border-radius: $border-radius-2xl;
      margin: $spacing-4;
      max-height: calc(100vh - #{$spacing-8});
    }
  }

  &--bottom {
    align-items: flex-end;

    .touch-modal__container {
      border-radius: $border-radius-2xl $border-radius-2xl 0 0;
      max-height: 90vh;
      @include safe-area-padding(padding, bottom);
    }
  }

  &--top {
    align-items: flex-start;

    .touch-modal__container {
      border-radius: 0 0 $border-radius-2xl $border-radius-2xl;
      margin-top: 0;
      @include safe-area-padding(padding, top);
    }
  }

  &--fullscreen {
    .touch-modal__container {
      width: 100vw;
      height: 100vh;
      max-width: none;
      max-height: none;
      border-radius: 0;
      margin: 0;
    }
  }

  // 크기별 스타일
  &--small {
    .touch-modal__container {
      max-width: 320px;
    }
  }

  &--medium {
    .touch-modal__container {
      max-width: 480px;
    }
  }

  &--large {
    .touch-modal__container {
      max-width: 640px;
    }
  }

  &--auto {
    .touch-modal__container {
      width: auto;
      min-width: 280px;
    }
  }

  // 닫기 애니메이션
  &--closing {
    .touch-modal__backdrop {
      animation: modal-backdrop-exit 0.3s ease;
    }

    .touch-modal__container {
      animation: modal-exit 0.3s ease;
    }
  }

  // 드래그 핸들
  &__handle {
    @include flex-center;
    padding: $spacing-3 0 $spacing-2 0;
    cursor: grab;

    &:active {
      cursor: grabbing;
    }
  }

  &__handle-bar {
    width: 40px;
    height: 4px;
    background: map-get($gray-colors, 300);
    border-radius: $border-radius-full;

    .dark & {
      background: $dark-text-tertiary;
    }
  }

  // 헤더
  &__header {
    border-bottom: 1px solid map-get($gray-colors, 200);
    padding: $spacing-4 $spacing-6;

    .touch-modal--draggable & {
      padding-top: $spacing-2;
    }

    .dark & {
      border-bottom-color: $dark-bg-tertiary;
    }
  }

  &__header-content {
    @include flex-between;
    align-items: center;
    gap: $spacing-4;
  }

  &__title {
    font-size: $font-size-xl;
    font-weight: $font-weight-semibold;
    color: $light-text-primary;
    margin: 0;

    .dark & {
      color: $dark-text-primary;
    }
  }

  &__close-button {
    @include touch-target(40px);
    @include touch-feedback;
    @include flex-center;
    background: none;
    border: none;
    color: map-get($gray-colors, 500);
    cursor: pointer;
    border-radius: $border-radius-full;

    svg {
      width: 20px;
      height: 20px;
    }

    &:hover {
      @include no-touch {
        background: map-get($gray-colors, 100);
        color: map-get($gray-colors, 700);

        .dark & {
          background: $dark-bg-tertiary;
          color: $dark-text-primary;
        }
      }
    }

    .dark & {
      color: $dark-text-secondary;
    }
  }

  // 콘텐츠
  &__content {
    padding: $spacing-6;
    overflow-y: auto;
    @include smooth-scroll;
    @include scrollbar-thin;

    .touch-modal--fullscreen & {
      flex: 1;
      padding: $spacing-4;
    }
  }

  // 푸터
  &__footer {
    border-top: 1px solid map-get($gray-colors, 200);
    padding: $spacing-4 $spacing-6;
    background: map-get($gray-colors, 50);

    .dark & {
      border-top-color: $dark-bg-tertiary;
      background: darken($dark-bg-secondary, 2%);
    }
  }
}

// 애니메이션
@keyframes modal-backdrop-enter {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

@keyframes modal-backdrop-exit {
  from {
    opacity: 1;
  }
  to {
    opacity: 0;
  }
}

@keyframes modal-enter {
  from {
    opacity: 0;
    transform: translateY(100%);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes modal-exit {
  from {
    opacity: 1;
    transform: translateY(0);
  }
  to {
    opacity: 0;
    transform: translateY(100%);
  }
}

// 중앙 정렬 모달 애니메이션
.touch-modal--center {
  @keyframes modal-enter {
    from {
      opacity: 0;
      transform: scale(0.9) translateY(20px);
    }
    to {
      opacity: 1;
      transform: scale(1) translateY(0);
    }
  }

  @keyframes modal-exit {
    from {
      opacity: 1;
      transform: scale(1) translateY(0);
    }
    to {
      opacity: 0;
      transform: scale(0.9) translateY(20px);
    }
  }
}

// 모바일 최적화
@include mobile {
  .touch-modal {
    &__container {
      margin: 0;
      width: 100vw;
      border-radius: $border-radius-2xl $border-radius-2xl 0 0;
    }

    &--center {
      .touch-modal__container {
        margin: $spacing-4;
        width: calc(100vw - #{$spacing-8});
        border-radius: $border-radius-2xl;
      }
    }

    &__content {
      padding: $spacing-4;
    }

    &__header,
    &__footer {
      padding-left: $spacing-4;
      padding-right: $spacing-4;
    }
  }
}

// 접근성 개선
@include reduce-motion {
  .touch-modal {
    &__backdrop,
    &__container {
      animation: none !important;
    }
  }
}

// 포커스 트랩
.touch-modal {
  &:focus {
    outline: none;
  }

  &__container:focus-within {
    outline: 2px solid map-get($primary-colors, 500);
    outline-offset: -2px;
  }
}
</style>