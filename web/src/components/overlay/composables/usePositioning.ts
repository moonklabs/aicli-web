import { type Ref, computed, onMounted, onUnmounted, ref, watch } from 'vue'

export type Placement =
  | 'top' | 'top-start' | 'top-end'
  | 'right' | 'right-start' | 'right-end'
  | 'bottom' | 'bottom-start' | 'bottom-end'
  | 'left' | 'left-start' | 'left-end'

export interface PositionOptions {
  placement?: Placement
  offset?: number
  flip?: boolean
  boundary?: 'viewport' | 'scrollParent' | HTMLElement
  arrowPadding?: number
}

export interface Position {
  x: number
  y: number
  placement: Placement
}

/**
 * 요소 포지셔닝 훅
 * 팝오버, 툴팁 등의 위치를 자동으로 계산하고 조정
 */
export function usePositioning(
  referenceRef: Ref<HTMLElement | null>,
  floatingRef: Ref<HTMLElement | null>,
  options: PositionOptions = {},
) {
  const {
    placement = 'bottom',
    offset = 8,
    flip = true,
    boundary = 'viewport',
    arrowPadding = 8,
  } = options

  const position = ref<Position>({
    x: 0,
    y: 0,
    placement,
  })

  const floatingStyles = computed(() => ({
    position: 'absolute' as const,
    top: `${position.value.y}px`,
    left: `${position.value.x}px`,
  }))

  /**
   * 요소의 바운딩 박스 가져오기
   */
  function getBoundingBox(element: HTMLElement) {
    return element.getBoundingClientRect()
  }

  /**
   * 뷰포트 크기 가져오기
   */
  function getViewportSize() {
    return {
      width: window.innerWidth,
      height: window.innerHeight,
    }
  }

  /**
   * 주어진 placement에 따른 위치 계산
   */
  function calculatePosition(
    referenceBounds: DOMRect,
    floatingBounds: DOMRect,
    placement: Placement,
  ): { x: number; y: number } {
    const [side, align] = placement.split('-') as [string, string?]

    let x = 0
    let y = 0

    // 주 축 위치 계산
    switch (side) {
      case 'top':
        x = referenceBounds.left + referenceBounds.width / 2 - floatingBounds.width / 2
        y = referenceBounds.top - floatingBounds.height - offset
        break
      case 'right':
        x = referenceBounds.right + offset
        y = referenceBounds.top + referenceBounds.height / 2 - floatingBounds.height / 2
        break
      case 'bottom':
        x = referenceBounds.left + referenceBounds.width / 2 - floatingBounds.width / 2
        y = referenceBounds.bottom + offset
        break
      case 'left':
        x = referenceBounds.left - floatingBounds.width - offset
        y = referenceBounds.top + referenceBounds.height / 2 - floatingBounds.height / 2
        break
    }

    // 정렬 조정
    if (align) {
      if (side === 'top' || side === 'bottom') {
        if (align === 'start') {
          x = referenceBounds.left
        } else if (align === 'end') {
          x = referenceBounds.right - floatingBounds.width
        }
      } else {
        if (align === 'start') {
          y = referenceBounds.top
        } else if (align === 'end') {
          y = referenceBounds.bottom - floatingBounds.height
        }
      }
    }

    return { x, y }
  }

  /**
   * 뷰포트 내에 위치하는지 확인
   */
  function isInViewport(
    x: number,
    y: number,
    width: number,
    height: number,
  ): boolean {
    const viewport = getViewportSize()
    return (
      x >= 0 &&
      y >= 0 &&
      x + width <= viewport.width &&
      y + height <= viewport.height
    )
  }

  /**
   * 대체 placement 목록 생성
   */
  function getFlipPlacements(placement: Placement): Placement[] {
    const [side, align] = placement.split('-') as [string, string?]
    const oppositeSide = {
      top: 'bottom',
      right: 'left',
      bottom: 'top',
      left: 'right',
    }[side]

    const placements: Placement[] = []

    // 반대편 같은 정렬
    if (oppositeSide) {
      placements.push(
        align ? `${oppositeSide}-${align}` as Placement : oppositeSide as Placement,
      )
    }

    // 다른 정렬들
    if (align) {
      placements.push(side as Placement)
      const oppositeAlign = align === 'start' ? 'end' : 'start'
      placements.push(`${side}-${oppositeAlign}` as Placement)
    }

    return placements
  }

  /**
   * 위치 업데이트
   */
  function updatePosition() {
    if (!referenceRef.value || !floatingRef.value) return

    const referenceBounds = getBoundingBox(referenceRef.value)
    const floatingBounds = getBoundingBox(floatingRef.value)

    let finalPlacement = placement
    let finalPosition = calculatePosition(referenceBounds, floatingBounds, placement)

    // Flip 로직
    if (flip) {
      const isValid = isInViewport(
        finalPosition.x,
        finalPosition.y,
        floatingBounds.width,
        floatingBounds.height,
      )

      if (!isValid) {
        const flipPlacements = getFlipPlacements(placement)

        for (const flipPlacement of flipPlacements) {
          const flipPosition = calculatePosition(referenceBounds, floatingBounds, flipPlacement)

          if (isInViewport(
            flipPosition.x,
            flipPosition.y,
            floatingBounds.width,
            floatingBounds.height,
          )) {
            finalPlacement = flipPlacement
            finalPosition = flipPosition
            break
          }
        }
      }
    }

    position.value = {
      x: finalPosition.x,
      y: finalPosition.y,
      placement: finalPlacement,
    }
  }

  // ResizeObserver로 크기 변경 감지
  let resizeObserver: ResizeObserver | null = null

  function setupObservers() {
    if (!referenceRef.value || !floatingRef.value) return

    resizeObserver = new ResizeObserver(() => {
      updatePosition()
    })

    resizeObserver.observe(referenceRef.value)
    resizeObserver.observe(floatingRef.value)
  }

  function cleanupObservers() {
    if (resizeObserver) {
      resizeObserver.disconnect()
      resizeObserver = null
    }
  }

  // 스크롤 및 리사이즈 이벤트 핸들러
  function handleUpdate() {
    updatePosition()
  }

  // 초기 설정
  onMounted(() => {
    updatePosition()
    setupObservers()

    window.addEventListener('scroll', handleUpdate, true)
    window.addEventListener('resize', handleUpdate)
  })

  // 정리
  onUnmounted(() => {
    cleanupObservers()

    window.removeEventListener('scroll', handleUpdate, true)
    window.removeEventListener('resize', handleUpdate)
  })

  // 참조 요소 변경 감지
  watch([referenceRef, floatingRef], () => {
    cleanupObservers()
    setupObservers()
    updatePosition()
  })

  return {
    position: computed(() => position.value),
    floatingStyles,
    updatePosition,
  }
}