import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'

interface VirtualScrollOptions {
  itemHeight: number
  containerHeight?: number
  overscan?: number
  enabled?: boolean
}

interface VirtualScrollItem {
  index: number
  top: number
  height: number
}

export function useVirtualScroll<T>(
  items: T[],
  containerRef: { value: HTMLElement | undefined },
  options: VirtualScrollOptions,
) {
  const {
    itemHeight,
    containerHeight: fixedContainerHeight,
    overscan = 3,
    enabled = true,
  } = options

  // 상태 관리
  const scrollTop = ref(0)
  const containerHeight = ref(fixedContainerHeight || 300)
  const isScrolling = ref(false)
  const scrollTimeout = ref<NodeJS.Timeout>()

  // 가상화 계산
  const totalHeight = computed(() => items.length * itemHeight)

  const visibleRange = computed(() => {
    if (!enabled || !containerHeight.value) {
      return {
        start: 0,
        end: items.length,
      }
    }

    const start = Math.floor(scrollTop.value / itemHeight)
    const visibleCount = Math.ceil(containerHeight.value / itemHeight)
    const end = Math.min(start + visibleCount, items.length)

    return {
      start: Math.max(0, start - overscan),
      end: Math.min(items.length, end + overscan),
    }
  })

  const visibleItems = computed(() => {
    const { start, end } = visibleRange.value
    return items.slice(start, end).map((item, index) => ({
      item,
      index: start + index,
      top: (start + index) * itemHeight,
    }))
  })

  const offsetY = computed(() => {
    return visibleRange.value.start * itemHeight
  })

  // 스크롤 이벤트 핸들러
  const handleScroll = (event: Event) => {
    const target = event.target as HTMLElement
    scrollTop.value = target.scrollTop

    // 스크롤 상태 추적
    isScrolling.value = true
    if (scrollTimeout.value) {
      clearTimeout(scrollTimeout.value)
    }
    scrollTimeout.value = setTimeout(() => {
      isScrolling.value = false
    }, 150)
  }

  // 컨테이너 크기 업데이트
  const updateContainerHeight = () => {
    if (containerRef.value && !fixedContainerHeight) {
      containerHeight.value = containerRef.value.clientHeight
    }
  }

  // 특정 인덱스로 스크롤
  const scrollToIndex = (index: number, behavior: ScrollBehavior = 'smooth') => {
    if (!containerRef.value) return

    const targetScrollTop = index * itemHeight
    containerRef.value.scrollTo({
      top: targetScrollTop,
      behavior,
    })
  }

  // 맨 위로 스크롤
  const scrollToTop = (behavior: ScrollBehavior = 'smooth') => {
    scrollToIndex(0, behavior)
  }

  // 맨 아래로 스크롤
  const scrollToBottom = (behavior: ScrollBehavior = 'smooth') => {
    if (!containerRef.value) return

    containerRef.value.scrollTo({
      top: totalHeight.value,
      behavior,
    })
  }

  // 리사이즈 옵저버
  let resizeObserver: ResizeObserver | null = null

  const setupResizeObserver = () => {
    if (!window.ResizeObserver || !containerRef.value) return

    resizeObserver = new ResizeObserver(() => {
      updateContainerHeight()
    })

    resizeObserver.observe(containerRef.value)
  }

  const cleanupResizeObserver = () => {
    if (resizeObserver) {
      resizeObserver.disconnect()
      resizeObserver = null
    }
  }

  // 이벤트 리스너 설정
  const setupEventListeners = () => {
    if (!containerRef.value) return

    containerRef.value.addEventListener('scroll', handleScroll, { passive: true })
    setupResizeObserver()
  }

  const cleanupEventListeners = () => {
    if (!containerRef.value) return

    containerRef.value.removeEventListener('scroll', handleScroll)
    cleanupResizeObserver()
  }

  // 생명주기 관리
  onMounted(() => {
    nextTick(() => {
      updateContainerHeight()
      setupEventListeners()
    })
  })

  onBeforeUnmount(() => {
    cleanupEventListeners()
    if (scrollTimeout.value) {
      clearTimeout(scrollTimeout.value)
    }
  })

  // containerRef 변화 감지
  watch(
    () => containerRef.value,
    (newContainer, oldContainer) => {
      if (oldContainer) {
        cleanupEventListeners()
      }
      if (newContainer) {
        nextTick(() => {
          updateContainerHeight()
          setupEventListeners()
        })
      }
    },
  )

  // 아이템 변화 감지
  watch(
    () => items.length,
    () => {
      // 아이템이 변경되면 스크롤 위치 조정
      nextTick(() => {
        if (scrollTop.value > totalHeight.value) {
          scrollToBottom('auto')
        }
      })
    },
  )

  return {
    // 상태
    scrollTop,
    containerHeight,
    isScrolling,
    totalHeight,
    visibleRange,
    visibleItems,
    offsetY,

    // 메서드
    scrollToIndex,
    scrollToTop,
    scrollToBottom,
    updateContainerHeight,

    // 이벤트 핸들러
    handleScroll,
  }
}

// 가상 스크롤 아이템 컴포넌트용 헬퍼
export interface VirtualScrollItemProps {
  index: number
  top: number
  height: number
}

export function createVirtualScrollItem(
  index: number,
  top: number,
  height: number,
): VirtualScrollItemProps {
  return {
    index,
    top,
    height,
  }
}

// 성능 최적화를 위한 유틸리티
export function useScrollPerformance() {
  const isScrolling = ref(false)
  const scrollVelocity = ref(0)
  let lastScrollTop = 0
  let lastScrollTime = Date.now()

  const updateScrollState = (scrollTop: number) => {
    const now = Date.now()
    const timeDiff = now - lastScrollTime
    const scrollDiff = scrollTop - lastScrollTop

    // 스크롤 속도 계산
    if (timeDiff > 0) {
      scrollVelocity.value = Math.abs(scrollDiff / timeDiff)
    }

    lastScrollTop = scrollTop
    lastScrollTime = now
    isScrolling.value = true

    // 스크롤 종료 감지
    setTimeout(() => {
      if (Date.now() - lastScrollTime > 100) {
        isScrolling.value = false
        scrollVelocity.value = 0
      }
    }, 100)
  }

  return {
    isScrolling,
    scrollVelocity,
    updateScrollState,
  }
}