import { type Ref, onBeforeUnmount, onMounted, ref } from 'vue'

interface TouchGestureConfig {
  // 제스처 활성화 옵션
  enableSwipe?: boolean
  enablePinch?: boolean
  enablePan?: boolean
  enableTap?: boolean
  enableLongPress?: boolean

  // 임계값 설정
  swipeThreshold?: number
  pinchThreshold?: number
  panThreshold?: number
  longPressThreshold?: number

  // 감도 설정
  swipeSensitivity?: number
  pinchSensitivity?: number
  panSensitivity?: number
}

interface TouchPoint {
  x: number
  y: number
  id: number
  timestamp: number
}

interface GestureEvent {
  type: string
  startPoint: TouchPoint
  currentPoint: TouchPoint
  deltaX: number
  deltaY: number
  distance: number
  direction: string
  velocity: number
  scale?: number
  angle?: number
}

export function useTouchGestures(
  element: Ref<HTMLElement | undefined>,
  config: TouchGestureConfig = {},
) {
  const defaultConfig: Required<TouchGestureConfig> = {
    enableSwipe: true,
    enablePinch: true,
    enablePan: true,
    enableTap: true,
    enableLongPress: true,
    swipeThreshold: 50,
    pinchThreshold: 0.1,
    panThreshold: 10,
    longPressThreshold: 500,
    swipeSensitivity: 1,
    pinchSensitivity: 1,
    panSensitivity: 1,
  }

  const settings = { ...defaultConfig, ...config }

  // 터치 상태 관리
  const touches = ref<Map<number, TouchPoint>>(new Map())
  const isGesturing = ref(false)
  const gestureStartTime = ref(0)
  const lastGestureEvent = ref<GestureEvent | null>(null)

  // 현재 제스처 타입
  const currentGesture = ref<string>('')

  // 이벤트 콜백들
  const callbacks = ref<{
    [key: string]: ((event: GestureEvent) => void)[]
  }>({})

  // 이벤트 리스너 등록
  const on = (eventType: string, callback: (event: GestureEvent) => void) => {
    if (!callbacks.value[eventType]) {
      callbacks.value[eventType] = []
    }
    callbacks.value[eventType].push(callback)
  }

  // 이벤트 리스너 제거
  const off = (eventType: string, callback: (event: GestureEvent) => void) => {
    if (callbacks.value[eventType]) {
      const index = callbacks.value[eventType].indexOf(callback)
      if (index > -1) {
        callbacks.value[eventType].splice(index, 1)
      }
    }
  }

  // 이벤트 발생
  const emit = (eventType: string, event: GestureEvent) => {
    if (callbacks.value[eventType]) {
      callbacks.value[eventType].forEach(callback => callback(event))
    }
  }

  // 터치 포인트를 TouchPoint로 변환
  const createTouchPoint = (touch: Touch): TouchPoint => ({
    x: touch.clientX,
    y: touch.clientY,
    id: touch.identifier,
    timestamp: Date.now(),
  })

  // 두 점 사이의 거리 계산
  const getDistance = (point1: TouchPoint, point2: TouchPoint): number => {
    const dx = point2.x - point1.x
    const dy = point2.y - point1.y
    return Math.sqrt(dx * dx + dy * dy)
  }

  // 각도 계산 (라디안)
  const getAngle = (point1: TouchPoint, point2: TouchPoint): number => {
    const dx = point2.x - point1.x
    const dy = point2.y - point1.y
    return Math.atan2(dy, dx)
  }

  // 방향 문자열 반환
  const getDirection = (deltaX: number, deltaY: number): string => {
    const absX = Math.abs(deltaX)
    const absY = Math.abs(deltaY)

    if (absX > absY) {
      return deltaX > 0 ? 'right' : 'left'
    } else {
      return deltaY > 0 ? 'down' : 'up'
    }
  }

  // 속도 계산
  const getVelocity = (startPoint: TouchPoint, currentPoint: TouchPoint): number => {
    const distance = getDistance(startPoint, currentPoint)
    const time = currentPoint.timestamp - startPoint.timestamp
    return time > 0 ? distance / time : 0
  }

  // 제스처 이벤트 생성
  const createGestureEvent = (
    type: string,
    startPoint: TouchPoint,
    currentPoint: TouchPoint,
  ): GestureEvent => {
    const deltaX = currentPoint.x - startPoint.x
    const deltaY = currentPoint.y - startPoint.y
    const distance = getDistance(startPoint, currentPoint)
    const direction = getDirection(deltaX, deltaY)
    const velocity = getVelocity(startPoint, currentPoint)

    return {
      type,
      startPoint,
      currentPoint,
      deltaX,
      deltaY,
      distance,
      direction,
      velocity,
    }
  }

  // 터치 시작 핸들러
  const handleTouchStart = (event: TouchEvent) => {
    event.preventDefault()

    Array.from(event.changedTouches).forEach(touch => {
      const touchPoint = createTouchPoint(touch)
      touches.value.set(touch.identifier, touchPoint)
    })

    gestureStartTime.value = Date.now()
    isGesturing.value = true

    // 싱글 터치 시작
    if (touches.value.size === 1 && settings.enableTap) {
      const touchPoint = Array.from(touches.value.values())[0]
      currentGesture.value = 'tap'

      // 롱 프레스 타이머 설정
      if (settings.enableLongPress) {
        setTimeout(() => {
          if (isGesturing.value && touches.value.size === 1 && currentGesture.value === 'tap') {
            const currentTouchPoint = Array.from(touches.value.values())[0]
            const gestureEvent = createGestureEvent('longpress', touchPoint, currentTouchPoint)
            emit('longpress', gestureEvent)
            currentGesture.value = 'longpress'
          }
        }, settings.longPressThreshold)
      }
    }

    // 멀티 터치 시작 (핀치)
    if (touches.value.size === 2 && settings.enablePinch) {
      currentGesture.value = 'pinch'
    }
  }

  // 터치 이동 핸들러
  const handleTouchMove = (event: TouchEvent) => {
    event.preventDefault()

    if (!isGesturing.value) return

    Array.from(event.changedTouches).forEach(touch => {
      const touchPoint = createTouchPoint(touch)
      const startTouchPoint = touches.value.get(touch.identifier)

      if (!startTouchPoint) return

      const deltaX = touchPoint.x - startTouchPoint.x
      const deltaY = touchPoint.y - startTouchPoint.y
      const distance = getDistance(startTouchPoint, touchPoint)

      // 싱글 터치 제스처 처리
      if (touches.value.size === 1) {
        // 팬 제스처
        if (settings.enablePan && distance > settings.panThreshold) {
          if (currentGesture.value === 'tap') {
            currentGesture.value = 'pan'
          }

          if (currentGesture.value === 'pan') {
            const gestureEvent = createGestureEvent('pan', startTouchPoint, touchPoint)
            emit('pan', gestureEvent)
            emit('panmove', gestureEvent)
          }
        }

        // 스와이프 제스처 (빠른 이동)
        if (settings.enableSwipe && distance > settings.swipeThreshold) {
          const velocity = getVelocity(startTouchPoint, touchPoint)

          if (velocity > 0.5) { // 속도 임계값
            const gestureEvent = createGestureEvent('swipe', startTouchPoint, touchPoint)
            emit('swipe', gestureEvent)
            emit(`swipe${gestureEvent.direction}`, gestureEvent)
          }
        }
      }

      // 멀티 터치 제스처 처리 (핀치/줌)
      if (touches.value.size === 2 && settings.enablePinch) {
        const touchPoints = Array.from(touches.value.values())

        if (touchPoints.length === 2) {
          const distance1 = getDistance(touchPoints[0], touchPoints[1])

          // 초기 거리 계산 (첫 번째 이동에서)
          if (!lastGestureEvent.value || lastGestureEvent.value.type !== 'pinch') {
            lastGestureEvent.value = {
              ...createGestureEvent('pinch', touchPoints[0], touchPoints[1]),
              scale: 1,
            }
          }

          const initialDistance = lastGestureEvent.value.distance
          const scale = distance1 / initialDistance

          if (Math.abs(scale - 1) > settings.pinchThreshold) {
            const gestureEvent = {
              ...createGestureEvent('pinch', touchPoints[0], touchPoints[1]),
              scale,
            }

            emit('pinch', gestureEvent)

            if (scale > 1) {
              emit('pinchout', gestureEvent) // 줌 아웃
            } else {
              emit('pinchin', gestureEvent) // 줌 인
            }

            lastGestureEvent.value = gestureEvent
          }
        }
      }
    })
  }

  // 터치 종료 핸들러
  const handleTouchEnd = (event: TouchEvent) => {
    Array.from(event.changedTouches).forEach(touch => {
      const startTouchPoint = touches.value.get(touch.identifier)

      if (startTouchPoint) {
        const endTouchPoint = createTouchPoint(touch)
        const gestureTime = Date.now() - gestureStartTime.value

        // 탭 제스처 처리
        if (
          settings.enableTap &&
          currentGesture.value === 'tap' &&
          gestureTime < settings.longPressThreshold
        ) {
          const distance = getDistance(startTouchPoint, endTouchPoint)

          if (distance < settings.panThreshold) {
            const gestureEvent = createGestureEvent('tap', startTouchPoint, endTouchPoint)
            emit('tap', gestureEvent)
          }
        }

        // 팬 종료
        if (currentGesture.value === 'pan') {
          const gestureEvent = createGestureEvent('panend', startTouchPoint, endTouchPoint)
          emit('panend', gestureEvent)
        }

        touches.value.delete(touch.identifier)
      }
    })

    // 모든 터치 포인트가 제거되면 제스처 종료
    if (touches.value.size === 0) {
      isGesturing.value = false
      currentGesture.value = ''
      lastGestureEvent.value = null

      emit('gestureend', {} as GestureEvent)
    }
  }

  // 터치 취소 핸들러
  const handleTouchCancel = (event: TouchEvent) => {
    handleTouchEnd(event)
  }

  // 이벤트 리스너 등록
  const setupEventListeners = () => {
    if (!element.value) return

    const el = element.value

    // 터치 이벤트
    el.addEventListener('touchstart', handleTouchStart, { passive: false })
    el.addEventListener('touchmove', handleTouchMove, { passive: false })
    el.addEventListener('touchend', handleTouchEnd, { passive: false })
    el.addEventListener('touchcancel', handleTouchCancel, { passive: false })

    // CSS 터치 액션 설정 (브라우저 기본 제스처 방지)
    el.style.touchAction = 'none'
  }

  // 이벤트 리스너 제거
  const removeEventListeners = () => {
    if (!element.value) return

    const el = element.value

    el.removeEventListener('touchstart', handleTouchStart)
    el.removeEventListener('touchmove', handleTouchMove)
    el.removeEventListener('touchend', handleTouchEnd)
    el.removeEventListener('touchcancel', handleTouchCancel)

    // CSS 터치 액션 복원
    el.style.touchAction = ''
  }

  // 제스처 비활성화
  const disable = () => {
    removeEventListeners()
  }

  // 제스처 활성화
  const enable = () => {
    setupEventListeners()
  }

  // 현재 터치 포인트 수 반환
  const getTouchCount = () => touches.value.size

  // 현재 제스처 상태 반환
  const getGestureState = () => ({
    isGesturing: isGesturing.value,
    currentGesture: currentGesture.value,
    touchCount: getTouchCount(),
    gestureTime: isGesturing.value ? Date.now() - gestureStartTime.value : 0,
  })

  // 라이프사이클 관리
  onMounted(() => {
    setupEventListeners()
  })

  onBeforeUnmount(() => {
    removeEventListeners()
  })

  return {
    // 상태
    isGesturing,
    currentGesture,

    // 메서드
    on,
    off,
    disable,
    enable,
    getTouchCount,
    getGestureState,

    // 유틸리티
    getDistance,
    getAngle,
    getDirection,
    getVelocity,
  }
}