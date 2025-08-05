import { type Ref, onBeforeUnmount, onMounted, ref, computed } from 'vue'
import { useTouchGestures } from './useTouchGestures'

interface TouchInteractionConfig {
  // 기본 제스처 설정
  enableSwipe?: boolean
  enablePinch?: boolean
  enablePan?: boolean
  enableTap?: boolean
  enableLongPress?: boolean
  
  // 고급 터치 기능
  enableHapticFeedback?: boolean
  enableContextMenu?: boolean
  enableQuickActions?: boolean
  enableDragAndDrop?: boolean
  
  // 터치 타겟 최적화
  minTouchTargetSize?: number
  touchTargetExpansion?: boolean
  
  // 스크롤 및 오버스크롤
  enableOverscroll?: boolean
  enablePullToRefresh?: boolean
  enableInfiniteScroll?: boolean
  
  // 성능 최적화
  enableTouchOptimization?: boolean
  preventDefaultBehavior?: boolean
  usePassiveListeners?: boolean
}

interface TouchInteractionState {
  isTouch: boolean
  touchCount: number
  lastTouchEvent: TouchEvent | null
  gestureInProgress: boolean
  touchStartTime: number
  touchPosition: { x: number; y: number }
  scrollDirection: 'up' | 'down' | 'left' | 'right' | null
}

export function useTouchInteractions(
  element: Ref<HTMLElement | undefined>,
  config: TouchInteractionConfig = {},
) {
  const defaultConfig: Required<TouchInteractionConfig> = {
    enableSwipe: true,
    enablePinch: true,
    enablePan: true,
    enableTap: true,
    enableLongPress: true,
    enableHapticFeedback: true,
    enableContextMenu: true,
    enableQuickActions: false,
    enableDragAndDrop: false,
    minTouchTargetSize: 44,
    touchTargetExpansion: true,
    enableOverscroll: true,
    enablePullToRefresh: false,
    enableInfiniteScroll: false,
    enableTouchOptimization: true,
    preventDefaultBehavior: false,
    usePassiveListeners: true,
  }

  const settings = { ...defaultConfig, ...config }

  // 터치 상태 관리
  const touchState = ref<TouchInteractionState>({
    isTouch: false,
    touchCount: 0,
    lastTouchEvent: null,
    gestureInProgress: false,
    touchStartTime: 0,
    touchPosition: { x: 0, y: 0 },
    scrollDirection: null,
  })

  // 터치 제스처 컴포저블 사용
  const {
    on: onGesture,
    off: offGesture,
    isGesturing,
    currentGesture,
    getTouchCount,
  } = useTouchGestures(element, {
    enableSwipe: settings.enableSwipe,
    enablePinch: settings.enablePinch,
    enablePan: settings.enablePan,
    enableTap: settings.enableTap,
    enableLongPress: settings.enableLongPress,
  })

  // 터치 디바이스 감지
  const isTouchDevice = computed(() => {
    return 'ontouchstart' in window || navigator.maxTouchPoints > 0
  })

  // 햅틱 피드백 지원 여부
  const supportsHaptics = computed(() => {
    return 'vibrate' in navigator
  })

  // 터치 타겟 크기 검증
  const validateTouchTarget = (target: HTMLElement): boolean => {
    const rect = target.getBoundingClientRect()
    const size = Math.min(rect.width, rect.height)
    return size >= settings.minTouchTargetSize
  }

  // 터치 타겟 확대 (접근성 개선)
  const expandTouchTarget = (target: HTMLElement) => {
    if (!settings.touchTargetExpansion) return

    const rect = target.getBoundingClientRect()
    const currentSize = Math.min(rect.width, rect.height)
    
    if (currentSize < settings.minTouchTargetSize) {
      const expansion = settings.minTouchTargetSize - currentSize
      target.style.padding = `${expansion / 2}px`
      target.dataset.touchExpanded = 'true'
    }
  }

  // 햅틱 피드백 실행
  const triggerHapticFeedback = (
    pattern: number | number[] = 50,
    force: 'light' | 'medium' | 'heavy' = 'light',
  ) => {
    if (!settings.enableHapticFeedback || !supportsHaptics.value) return

    try {
      // 진동 API 사용
      if (Array.isArray(pattern)) {
        navigator.vibrate(pattern)
      } else {
        // 강도에 따른 진동 패턴
        const patterns = {
          light: [10],
          medium: [20],
          heavy: [50],
        }
        navigator.vibrate(patterns[force])
      }
    } catch (error) {
      console.warn('Haptic feedback not supported:', error)
    }
  }

  // 스크롤 방향 감지
  const detectScrollDirection = (deltaX: number, deltaY: number) => {
    const threshold = 10
    
    if (Math.abs(deltaY) > Math.abs(deltaX)) {
      touchState.value.scrollDirection = deltaY > threshold ? 'down' : deltaY < -threshold ? 'up' : null
    } else {
      touchState.value.scrollDirection = deltaX > threshold ? 'right' : deltaX < -threshold ? 'left' : null
    }
  }

  // 오버스크롤 효과
  const handleOverscroll = (element: HTMLElement, direction: string, amount: number) => {
    if (!settings.enableOverscroll) return

    const maxOverscroll = 50
    const clampedAmount = Math.min(amount, maxOverscroll)
    
    switch (direction) {
      case 'up':
        element.style.transform = `translateY(${clampedAmount}px)`
        break
      case 'down':
        element.style.transform = `translateY(-${clampedAmount}px)`
        break
      case 'left':
        element.style.transform = `translateX(${clampedAmount}px)`
        break
      case 'right':
        element.style.transform = `translateX(-${clampedAmount}px)`
        break
    }

    // 복원 애니메이션
    setTimeout(() => {
      element.style.transition = 'transform 0.3s ease-out'
      element.style.transform = 'translateX(0) translateY(0)'
      
      setTimeout(() => {
        element.style.transition = ''
      }, 300)
    }, 100)
  }

  // 컨텍스트 메뉴 처리
  const handleContextMenu = (event: TouchEvent | MouseEvent) => {
    if (!settings.enableContextMenu) return

    event.preventDefault()
    
    const target = event.target as HTMLElement
    const rect = target.getBoundingClientRect()
    
    // 커스텀 컨텍스트 메뉴 이벤트 발생
    const contextMenuEvent = new CustomEvent('customcontextmenu', {
      detail: {
        x: rect.left + rect.width / 2,
        y: rect.top + rect.height / 2,
        target,
        originalEvent: event,
      },
    })
    
    target.dispatchEvent(contextMenuEvent)
  }

  // 터치 최적화 적용
  const applyTouchOptimizations = (target: HTMLElement) => {
    if (!settings.enableTouchOptimization) return

    // CSS 터치 최적화
    target.style.touchAction = 'manipulation'
    target.style.webkitTapHighlightColor = 'transparent'
    target.style.webkitTouchCallout = 'none'
    target.style.webkitUserSelect = 'none'
    target.style.userSelect = 'none'

    // 터치 타겟 크기 검증 및 확대
    if (!validateTouchTarget(target)) {
      expandTouchTarget(target)
    }
  }

  // 제스처 이벤트 핸들러 설정
  const setupGestureHandlers = () => {
    // 탭 제스처
    onGesture('tap', (event) => {
      triggerHapticFeedback(10, 'light')
      touchState.value.touchPosition = { x: event.currentPoint.x, y: event.currentPoint.y }
    })

    // 롱 프레스 제스처
    onGesture('longpress', (event) => {
      triggerHapticFeedback([50, 50], 'medium')
      if (settings.enableContextMenu) {
        // 가상의 터치 이벤트 생성하여 컨텍스트 메뉴 처리
        const touchEvent = new TouchEvent('touchstart', {
          touches: [{
            clientX: event.currentPoint.x,
            clientY: event.currentPoint.y,
          } as Touch],
        })
        handleContextMenu(touchEvent)
      }
    })

    // 스와이프 제스처
    onGesture('swipe', (event) => {
      triggerHapticFeedback(30, 'light')
      detectScrollDirection(event.deltaX, event.deltaY)
    })

    // 핀치 제스처
    onGesture('pinch', (event) => {
      if (event.scale && event.scale !== 1) {
        triggerHapticFeedback(20, 'light')
      }
    })

    // 팬 제스처
    onGesture('pan', (event) => {
      detectScrollDirection(event.deltaX, event.deltaY)
      
      // 오버스크롤 처리
      if (settings.enableOverscroll && element.value) {
        const scrollElement = element.value
        const isAtEdge = (
          (event.direction === 'up' && scrollElement.scrollTop === 0) ||
          (event.direction === 'down' && 
           scrollElement.scrollTop >= scrollElement.scrollHeight - scrollElement.clientHeight) ||
          (event.direction === 'left' && scrollElement.scrollLeft === 0) ||
          (event.direction === 'right' && 
           scrollElement.scrollLeft >= scrollElement.scrollWidth - scrollElement.clientWidth)
        )
        
        if (isAtEdge) {
          handleOverscroll(scrollElement, event.direction, event.distance * 0.5)
        }
      }
    })
  }

  // 터치 이벤트 핸들러
  const handleTouchStart = (event: TouchEvent) => {
    touchState.value.isTouch = true
    touchState.value.touchCount = event.touches.length
    touchState.value.lastTouchEvent = event
    touchState.value.touchStartTime = Date.now()
    touchState.value.gestureInProgress = true

    if (event.touches.length > 0) {
      touchState.value.touchPosition = {
        x: event.touches[0].clientX,
        y: event.touches[0].clientY,
      }
    }
  }

  const handleTouchEnd = (event: TouchEvent) => {
    touchState.value.isTouch = false
    touchState.value.touchCount = event.touches.length
    touchState.value.gestureInProgress = false
    touchState.value.scrollDirection = null
  }

  // 이벤트 리스너 설정
  const setupEventListeners = () => {
    if (!element.value) return

    const target = element.value
    const options = { 
      passive: settings.usePassiveListeners && !settings.preventDefaultBehavior,
      capture: false,
    }

    // 터치 이벤트
    target.addEventListener('touchstart', handleTouchStart, options)
    target.addEventListener('touchend', handleTouchEnd, options)

    // 컨텍스트 메뉴 (데스크톱)
    if (settings.enableContextMenu) {
      target.addEventListener('contextmenu', handleContextMenu)
    }

    // 터치 최적화 적용
    applyTouchOptimizations(target)

    // 제스처 핸들러 설정
    setupGestureHandlers()
  }

  const removeEventListeners = () => {
    if (!element.value) return

    const target = element.value

    target.removeEventListener('touchstart', handleTouchStart)
    target.removeEventListener('touchend', handleTouchEnd)
    target.removeEventListener('contextmenu', handleContextMenu)
  }

  // 터치 상호작용 활성화/비활성화
  const enable = () => {
    setupEventListeners()
  }

  const disable = () => {
    removeEventListeners()
  }

  // 생명주기 관리
  onMounted(() => {
    setupEventListeners()
  })

  onBeforeUnmount(() => {
    removeEventListeners()
  })

  return {
    // 상태
    touchState: computed(() => touchState.value),
    isTouchDevice,
    supportsHaptics,
    isGesturing,
    currentGesture,
    
    // 메서드
    enable,
    disable,
    triggerHapticFeedback,
    validateTouchTarget,
    expandTouchTarget,
    applyTouchOptimizations,
    
    // 제스처 이벤트
    onGesture,
    offGesture,
    
    // 유틸리티
    getTouchCount,
  }
}