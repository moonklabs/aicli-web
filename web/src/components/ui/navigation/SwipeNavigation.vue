<template>
  <div class="swipe-navigation" ref="containerRef">
    <!-- 현재 페이지 인디케이터 -->
    <div v-if="showIndicator" class="swipe-navigation__indicator">
      <div
        v-for="(page, index) in pages"
        :key="index"
        class="swipe-navigation__dot"
        :class="{ 'swipe-navigation__dot--active': currentIndex === index }"
        @click="goToPage(index)"
      ></div>
    </div>

    <!-- 스와이프 컨테이너 -->
    <div
      class="swipe-navigation__container"
      :style="containerStyle"
      @touchstart="handleTouchStart"
      @touchmove="handleTouchMove"
      @touchend="handleTouchEnd"
    >
      <div
        v-for="(page, index) in pages"
        :key="index"
        class="swipe-navigation__page"
        :class="{
          'swipe-navigation__page--active': currentIndex === index,
          'swipe-navigation__page--prev': currentIndex === index - 1,
          'swipe-navigation__page--next': currentIndex === index + 1,
        }"
      >
        <component
          v-if="page.component"
          :is="page.component"
          v-bind="page.props"
        />
        <div v-else-if="page.content" v-html="page.content"></div>
        <slot v-else :name="page.slot" :page="page" :index="index"></slot>
      </div>
    </div>

    <!-- 네비게이션 화살표 (데스크톱용) -->
    <button
      v-if="showArrows && currentIndex > 0"
      class="swipe-navigation__arrow swipe-navigation__arrow--left"
      @click="goToPrevPage"
      aria-label="이전 페이지"
    >
      <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
        <path
          d="M15 18l-6-6 6-6"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
      </svg>
    </button>

    <button
      v-if="showArrows && currentIndex < pages.length - 1"
      class="swipe-navigation__arrow swipe-navigation__arrow--right"
      @click="goToNextPage"
      aria-label="다음 페이지"
    >
      <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
        <path
          d="M9 18l6-6-6-6"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
      </svg>
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useTouchGestures } from '@/composables/useTouchGestures'

interface Page {
  id: string
  title?: string
  component?: any
  props?: Record<string, any>
  content?: string
  slot?: string
}

interface Props {
  pages: Page[]
  initialIndex?: number
  showIndicator?: boolean
  showArrows?: boolean
  autoPlay?: boolean
  autoPlayDelay?: number
  loop?: boolean
  swipeThreshold?: number
  animationDuration?: number
}

const props = withDefaults(defineProps<Props>(), {
  initialIndex: 0,
  showIndicator: true,
  showArrows: false,
  autoPlay: false,
  autoPlayDelay: 5000,
  loop: false,
  swipeThreshold: 50,
  animationDuration: 300,
})

const emit = defineEmits<{
  'page-change': [index: number, page: Page]
  'swipe-start': [index: number]
  'swipe-end': [index: number]
}>()

const containerRef = ref<HTMLElement>()
const currentIndex = ref(props.initialIndex)
const isTransitioning = ref(false)
const translateX = ref(0)
const autoPlayTimer = ref<NodeJS.Timeout>()

// 터치 제스처 설정
const { on: onGesture } = useTouchGestures(containerRef, {
  enableSwipe: true,
  enablePan: true,
  swipeThreshold: props.swipeThreshold,
})

// 스와이프 제스처 처리
onGesture('swipeLeft', () => {
  if (!isTransitioning.value) {
    goToNextPage()
  }
})

onGesture('swipeRight', () => {
  if (!isTransitioning.value) {
    goToPrevPage()
  }
})

onGesture('pan', (event) => {
  if (!isTransitioning.value) {
    const maxTranslate = containerRef.value?.clientWidth || 0
    const translate = Math.max(-maxTranslate, Math.min(maxTranslate, event.deltaX))
    translateX.value = translate
  }
})

onGesture('panend', (event) => {
  if (!isTransitioning.value) {
    const threshold = (containerRef.value?.clientWidth || 0) * 0.3

    if (Math.abs(event.deltaX) > threshold) {
      if (event.deltaX > 0) {
        goToPrevPage()
      } else {
        goToNextPage()
      }
    } else {
      // 원래 위치로 복귀
      translateX.value = 0
    }
  }
})

// 컨테이너 스타일
const containerStyle = computed(() => ({
  transform: `translateX(calc(-${currentIndex.value * 100}% + ${translateX.value}px))`,
  transition: isTransitioning.value ? `transform ${props.animationDuration}ms ease` : 'none',
}))

// 페이지 이동 함수들
const goToPage = async (index: number) => {
  if (index === currentIndex.value || isTransitioning.value) return

  if (index < 0 || index >= props.pages.length) {
    if (!props.loop) return
    index = index < 0 ? props.pages.length - 1 : 0
  }

  isTransitioning.value = true
  translateX.value = 0

  const prevIndex = currentIndex.value
  currentIndex.value = index

  emit('page-change', index, props.pages[index])

  await nextTick()

  // 애니메이션 완료 후 상태 초기화
  setTimeout(() => {
    isTransitioning.value = false
  }, props.animationDuration)

  resetAutoPlay()
}

const goToNextPage = () => {
  const nextIndex = currentIndex.value + 1
  if (nextIndex < props.pages.length) {
    goToPage(nextIndex)
  } else if (props.loop) {
    goToPage(0)
  }
}

const goToPrevPage = () => {
  const prevIndex = currentIndex.value - 1
  if (prevIndex >= 0) {
    goToPage(prevIndex)
  } else if (props.loop) {
    goToPage(props.pages.length - 1)
  }
}

// 자동 재생 관리
const startAutoPlay = () => {
  if (!props.autoPlay) return

  autoPlayTimer.value = setInterval(() => {
    goToNextPage()
  }, props.autoPlayDelay)
}

const stopAutoPlay = () => {
  if (autoPlayTimer.value) {
    clearInterval(autoPlayTimer.value)
    autoPlayTimer.value = undefined
  }
}

const resetAutoPlay = () => {
  stopAutoPlay()
  startAutoPlay()
}

// 초기화 및 정리
watch(
  () => props.autoPlay,
  (autoPlay) => {
    if (autoPlay) {
      startAutoPlay()
    } else {
      stopAutoPlay()
    }
  },
  { immediate: true },
)

watch(
  () => props.pages.length,
  (newLength) => {
    if (currentIndex.value >= newLength) {
      currentIndex.value = Math.max(0, newLength - 1)
    }
  },
)

// 컴포넌트 해제 시 타이머 정리
watch(containerRef, (el) => {
  if (!el) {
    stopAutoPlay()
  }
})

// 키보드 네비게이션 지원
const handleKeydown = (event: KeyboardEvent) => {
  if (event.key === 'ArrowLeft') {
    event.preventDefault()
    goToPrevPage()
  } else if (event.key === 'ArrowRight') {
    event.preventDefault()
    goToNextPage()
  }
}

// 포커스/블러 시 자동 재생 제어
const handleVisibilityChange = () => {
  if (document.hidden) {
    stopAutoPlay()
  } else {
    resetAutoPlay()
  }
}

// 이벤트 리스너 등록
if (typeof document !== 'undefined') {
  document.addEventListener('keydown', handleKeydown)
  document.addEventListener('visibilitychange', handleVisibilityChange)
}

// 공개 메서드
defineExpose({
  goToPage,
  goToNextPage,
  goToPrevPage,
  currentIndex: () => currentIndex.value,
  isTransitioning: () => isTransitioning.value,
})
</script>

<style lang="scss" scoped>
@use '@/styles/variables' as *;
@use '@/styles/mixins' as *;

.swipe-navigation {
  position: relative;
  width: 100%;
  height: 100%;
  overflow: hidden;

  &__indicator {
    position: absolute;
    bottom: $spacing-4;
    left: 50%;
    transform: translateX(-50%);
    display: flex;
    gap: $spacing-2;
    z-index: 10;
  }

  &__dot {
    width: 8px;
    height: 8px;
    border-radius: $border-radius-full;
    background: rgba(255, 255, 255, 0.5);
    cursor: pointer;
    transition: $transition-base;

    &:hover {
      background: rgba(255, 255, 255, 0.8);
      transform: scale(1.2);
    }

    &--active {
      background: white;
      transform: scale(1.3);
    }
  }

  &__container {
    display: flex;
    width: 100%;
    height: 100%;
    will-change: transform;
  }

  &__page {
    flex: 0 0 100%;
    width: 100%;
    height: 100%;
    display: flex;
    flex-direction: column;
    opacity: 0.7;
    transform: scale(0.9);
    transition: opacity 0.3s ease, transform 0.3s ease;

    &--active {
      opacity: 1;
      transform: scale(1);
    }

    &--prev,
    &--next {
      opacity: 0.5;
      transform: scale(0.85);
    }
  }

  &__arrow {
    position: absolute;
    top: 50%;
    transform: translateY(-50%);
    width: 48px;
    height: 48px;
    border-radius: $border-radius-full;
    background: rgba(255, 255, 255, 0.9);
    border: none;
    @include flex-center;
    cursor: pointer;
    z-index: 10;
    transition: $transition-base;
    box-shadow: $shadow-md;

    &:hover {
      background: white;
      transform: translateY(-50%) scale(1.1);
      box-shadow: $shadow-lg;
    }

    &:active {
      transform: translateY(-50%) scale(0.95);
    }

    &--left {
      left: $spacing-4;
    }

    &--right {
      right: $spacing-4;
    }

    svg {
      color: map-get($gray-colors, 700);
    }
  }
}

// 모바일에서 화살표 숨기기
@include mobile {
  .swipe-navigation__arrow {
    display: none;
  }
}

// 접근성 개선
.swipe-navigation:focus-within {
  .swipe-navigation__arrow {
    opacity: 1;
  }
}

// 스와이프 인디케이터 애니메이션
@keyframes swipe-hint {
  0% { transform: translateX(0); }
  25% { transform: translateX(10px); }
  75% { transform: translateX(-10px); }
  100% { transform: translateX(0); }
}

.swipe-navigation--hint {
  .swipe-navigation__container {
    animation: swipe-hint 1s ease-in-out 2s 3;
  }
}
</style>