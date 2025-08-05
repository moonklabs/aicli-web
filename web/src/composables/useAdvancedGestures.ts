import { type Ref, computed, ref, watch } from 'vue'
import { useTouchGestures } from './useTouchGestures'

interface AdvancedGestureConfig {
  // 고급 제스처 활성화
  enableRotation?: boolean
  enableMultiFingerGestures?: boolean
  enableVelocityGestures?: boolean
  enableInertia?: boolean
  enableCustomPatterns?: boolean
  
  // 임계값 설정
  rotationThreshold?: number
  velocityThreshold?: number
  inertiaThreshold?: number
  
  // 감도 설정
  rotationSensitivity?: number
  velocitySensitivity?: number
  inertiaSensitivity?: number
  
  // 고급 설정
  enableHapticFeedback?: boolean
  enableSoundFeedback?: boolean
  maxFingers?: number
}

interface AdvancedGestureEvent {
  type: string
  fingerCount: number
  rotation?: number
  velocity: { x: number; y: number; magnitude: number }
  centroid: { x: number; y: number }
  momentum?: { x: number; y: number }
  inertia?: { x: number; y: number; decay: number }
  pattern?: string
  timestamp: number
}

interface TouchPoint {
  x: number
  y: number
  id: number
  timestamp: number
}

export function useAdvancedGestures(
  element: Ref<HTMLElement | undefined>,
  config: AdvancedGestureConfig = {},
) {
  const defaultConfig: Required<AdvancedGestureConfig> = {
    enableRotation: true,
    enableMultiFingerGestures: true,
    enableVelocityGestures: true,
    enableInertia: true,
    enableCustomPatterns: false,
    rotationThreshold: 0.1,
    velocityThreshold: 0.5,
    inertiaThreshold: 0.3,
    rotationSensitivity: 1,
    velocitySensitivity: 1,
    inertiaSensitivity: 1,
    enableHapticFeedback: true,
    enableSoundFeedback: false,
    maxFingers: 5,
  }

  const settings = { ...defaultConfig, ...config }

  // 기본 제스처 시스템 사용
  const baseGestures = useTouchGestures(element, {
    enableSwipe: true,
    enablePinch: true,
    enablePan: true,
    enableTap: true,
    enableLongPress: true,
  })

  // 고급 제스처 상태
  const currentRotation = ref(0)
  const lastRotationAngle = ref(0)
  const velocityHistory = ref<{ x: number; y: number; timestamp: number }[]>([])
  const gesturePattern = ref<TouchPoint[]>([])
  const isAdvancedGesturing = ref(false)

  // 이벤트 콜백들
  const advancedCallbacks = ref<{
    [key: string]: ((event: AdvancedGestureEvent) => void)[]
  }>({})

  // 이벤트 리스너 등록
  const on = (eventType: string, callback: (event: AdvancedGestureEvent) => void) => {
    if (!advancedCallbacks.value[eventType]) {
      advancedCallbacks.value[eventType] = []
    }
    advancedCallbacks.value[eventType].push(callback)
  }

  // 이벤트 리스너 제거
  const off = (eventType: string, callback: (event: AdvancedGestureEvent) => void) => {
    if (advancedCallbacks.value[eventType]) {
      const index = advancedCallbacks.value[eventType].indexOf(callback)
      if (index > -1) {
        advancedCallbacks.value[eventType].splice(index, 1)
      }
    }
  }

  // 이벤트 발생
  const emit = (eventType: string, event: AdvancedGestureEvent) => {
    if (advancedCallbacks.value[eventType]) {
      advancedCallbacks.value[eventType].forEach(callback => callback(event))
    }
  }

  // 중심점 계산
  const calculateCentroid = (points: TouchPoint[]): { x: number; y: number } => {
    if (points.length === 0) return { x: 0, y: 0 }
    
    const sum = points.reduce(
      (acc, point) => ({ x: acc.x + point.x, y: acc.y + point.y }),
      { x: 0, y: 0 }
    )
    
    return {
      x: sum.x / points.length,
      y: sum.y / points.length,
    }
  }

  // 회전 각도 계산
  const calculateRotation = (touches: TouchPoint[]): number => {
    if (touches.length < 2) return 0

    const [touch1, touch2] = touches
    const angle = Math.atan2(touch2.y - touch1.y, touch2.x - touch1.x)
    
    return angle
  }

  // 고급 속도 계산
  const calculateAdvancedVelocity = (history: { x: number; y: number; timestamp: number }[]) => {
    if (history.length < 2) return { x: 0, y: 0, magnitude: 0 }

    const recent = history.slice(-3) // 최근 3개 샘플
    const timeSpan = recent[recent.length - 1].timestamp - recent[0].timestamp
    
    if (timeSpan === 0) return { x: 0, y: 0, magnitude: 0 }

    const deltaX = recent[recent.length - 1].x - recent[0].x
    const deltaY = recent[recent.length - 1].y - recent[0].y
    
    const velocityX = deltaX / timeSpan
    const velocityY = deltaY / timeSpan
    const magnitude = Math.sqrt(velocityX ** 2 + velocityY ** 2)

    return { x: velocityX, y: velocityY, magnitude }
  }

  // 관성 계산
  const calculateInertia = (velocity: { x: number; y: number; magnitude: number }) => {
    const decay = Math.max(0.95, Math.min(0.99, velocity.magnitude * 0.01))
    
    return {
      x: velocity.x,
      y: velocity.y,
      decay,
    }
  }

  // 멀티 핑거 패턴 감지
  const detectMultiFingerPattern = (fingerCount: number, centroid: { x: number; y: number }): string => {
    switch (fingerCount) {
      case 3:
        return 'threefinger'
      case 4:
        return 'fourfinger'
      case 5:
        return 'fivefinger'
      default:
        return 'multifinger'
    }
  }

  // 커스텀 제스처 패턴 분석
  const analyzeGesturePattern = (pattern: TouchPoint[]): string | null => {
    if (!settings.enableCustomPatterns || pattern.length < 5) return null

    // 원형 제스처 감지
    const centroid = calculateCentroid(pattern)
    let isCircular = true
    let avgRadius = 0

    // 평균 반지름 계산
    for (const point of pattern) {
      const radius = Math.sqrt((point.x - centroid.x) ** 2 + (point.y - centroid.y) ** 2)
      avgRadius += radius
    }
    avgRadius /= pattern.length

    // 원형 여부 검사
    for (const point of pattern) {
      const radius = Math.sqrt((point.x - centroid.x) ** 2 + (point.y - centroid.y) ** 2)
      if (Math.abs(radius - avgRadius) > avgRadius * 0.3) {
        isCircular = false
        break
      }
    }

    if (isCircular && pattern.length >= 8) {
      return 'circle'
    }

    // Z형 제스처 감지 (간단한 지그재그 패턴)
    if (pattern.length >= 6) {
      let directionChanges = 0
      let lastDirection = ''

      for (let i = 1; i < pattern.length; i++) {
        const deltaX = pattern[i].x - pattern[i - 1].x
        const deltaY = pattern[i].y - pattern[i - 1].y
        
        const currentDirection = Math.abs(deltaX) > Math.abs(deltaY) 
          ? (deltaX > 0 ? 'right' : 'left')
          : (deltaY > 0 ? 'down' : 'up')

        if (lastDirection && currentDirection !== lastDirection) {
          directionChanges++
        }
        lastDirection = currentDirection
      }

      if (directionChanges >= 3) {
        return 'zigzag'
      }
    }

    return null
  }

  // 햅틱 피드백 실행
  const triggerHapticFeedback = (pattern: number[] = [50]) => {
    if (!settings.enableHapticFeedback || !('vibrate' in navigator)) return

    try {
      navigator.vibrate(pattern)
    } catch (error) {
      console.warn('Haptic feedback not supported:', error)
    }
  }

  // 사운드 피드백 실행
  const triggerSoundFeedback = (type: 'tap' | 'swipe' | 'rotate' | 'multi') => {
    if (!settings.enableSoundFeedback) return

    // Web Audio API를 사용한 간단한 사운드 생성
    try {
      const audioContext = new (window.AudioContext || (window as any).webkitAudioContext)()
      const oscillator = audioContext.createOscillator()
      const gainNode = audioContext.createGain()

      oscillator.connect(gainNode)
      gainNode.connect(audioContext.destination)

      // 제스처 타입별 주파수 설정
      const frequencies = {
        tap: 800,
        swipe: 400,
        rotate: 600,
        multi: 1000,
      }

      oscillator.frequency.setValueAtTime(frequencies[type], audioContext.currentTime)
      oscillator.type = 'sine'

      gainNode.gain.setValueAtTime(0.1, audioContext.currentTime)
      gainNode.gain.exponentialRampToValueAtTime(0.01, audioContext.currentTime + 0.1)

      oscillator.start()
      oscillator.stop(audioContext.currentTime + 0.1)
    } catch (error) {
      console.warn('Sound feedback not supported:', error)
    }
  }

  // 기본 제스처 이벤트 확장
  baseGestures.on('pinch', (gestureEvent) => {
    if (!settings.enableRotation) return

    const touchCount = baseGestures.getTouchCount()
    if (touchCount === 2) {
      // 회전 감지를 위한 추가 처리
      const angle = calculateRotation([
        gestureEvent.startPoint,
        gestureEvent.currentPoint,
      ])

      const rotationDelta = angle - lastRotationAngle.value
      
      if (Math.abs(rotationDelta) > settings.rotationThreshold) {
        currentRotation.value += rotationDelta
        lastRotationAngle.value = angle

        const advancedEvent: AdvancedGestureEvent = {
          type: 'rotation',
          fingerCount: 2,
          rotation: rotationDelta,
          velocity: calculateAdvancedVelocity(velocityHistory.value),
          centroid: { x: gestureEvent.currentPoint.x, y: gestureEvent.currentPoint.y },
          timestamp: Date.now(),
        }

        emit('rotation', advancedEvent)
        triggerHapticFeedback([30])
        triggerSoundFeedback('rotate')
      }
    }
  })

  // 멀티 핑거 제스처 감지
  const detectMultiFingerGestures = () => {
    const touchCount = baseGestures.getTouchCount()
    
    if (touchCount >= 3 && settings.enableMultiFingerGestures) {
      isAdvancedGesturing.value = true
      
      // 가상의 터치 포인트들을 생성 (실제 구현에서는 실제 터치 데이터 사용)
      const mockTouches: TouchPoint[] = Array.from({ length: touchCount }, (_, i) => ({
        x: 100 + i * 50,
        y: 100 + i * 50,
        id: i,
        timestamp: Date.now(),
      }))

      const centroid = calculateCentroid(mockTouches)
      const pattern = detectMultiFingerPattern(touchCount, centroid)

      const advancedEvent: AdvancedGestureEvent = {
        type: pattern,
        fingerCount: touchCount,
        velocity: calculateAdvancedVelocity(velocityHistory.value),
        centroid,
        timestamp: Date.now(),
      }

      emit(pattern, advancedEvent)
      triggerHapticFeedback([20, 20, 20])
      triggerSoundFeedback('multi')
    }
  }

  // 고속 제스처 감지
  baseGestures.on('swipe', (gestureEvent) => {
    if (!settings.enableVelocityGestures) return

    const velocity = gestureEvent.velocity
    
    if (velocity > settings.velocityThreshold * 2) {
      const advancedEvent: AdvancedGestureEvent = {
        type: 'fastswipe',
        fingerCount: 1,
        velocity: {
          x: gestureEvent.deltaX,
          y: gestureEvent.deltaY,
          magnitude: velocity,
        },
        centroid: { x: gestureEvent.currentPoint.x, y: gestureEvent.currentPoint.y },
        timestamp: Date.now(),
      }

      emit('fastswipe', advancedEvent)
      emit(`fastswipe${gestureEvent.direction}`, advancedEvent)
      triggerHapticFeedback([40])
    }

    // 관성 계산
    if (settings.enableInertia && velocity > settings.inertiaThreshold) {
      const inertia = calculateInertia({
        x: gestureEvent.deltaX,
        y: gestureEvent.deltaY,
        magnitude: velocity,
      })

      const inertiaEvent: AdvancedGestureEvent = {
        type: 'inertia',
        fingerCount: 1,
        velocity: {
          x: gestureEvent.deltaX,
          y: gestureEvent.deltaY,
          magnitude: velocity,
        },
        centroid: { x: gestureEvent.currentPoint.x, y: gestureEvent.currentPoint.y },
        inertia,
        timestamp: Date.now(),
      }

      // 관성 애니메이션 시작
      startInertiaAnimation(inertiaEvent)
    }
  })

  // 관성 애니메이션
  const startInertiaAnimation = (event: AdvancedGestureEvent) => {
    if (!event.inertia) return

    let { x, y, decay } = event.inertia
    
    const animate = () => {
      if (Math.abs(x) < 0.1 && Math.abs(y) < 0.1) {
        emit('inertiaend', event)
        return
      }

      const inertiaFrame: AdvancedGestureEvent = {
        ...event,
        type: 'inertiaframe',
        inertia: { x, y, decay },
        timestamp: Date.now(),
      }

      emit('inertiaframe', inertiaFrame)

      x *= decay
      y *= decay

      requestAnimationFrame(animate)
    }

    requestAnimationFrame(animate)
  }

  // 제스처 종료 시 패턴 분석
  baseGestures.on('gestureend', () => {
    if (gesturePattern.value.length > 0) {
      const pattern = analyzeGesturePattern(gesturePattern.value)
      
      if (pattern) {
        const patternEvent: AdvancedGestureEvent = {
          type: `pattern_${pattern}`,
          fingerCount: 1,
          velocity: calculateAdvancedVelocity(velocityHistory.value),
          centroid: calculateCentroid(gesturePattern.value),
          pattern,
          timestamp: Date.now(),
        }

        emit(`pattern_${pattern}`, patternEvent)
        triggerHapticFeedback([60, 60])
      }

      gesturePattern.value = []
    }

    isAdvancedGesturing.value = false
    currentRotation.value = 0
    lastRotationAngle.value = 0
    velocityHistory.value = []
  })

  // 터치 카운트 변경 감지
  watch(() => baseGestures.getTouchCount(), (newCount, oldCount) => {
    if (newCount !== oldCount) {
      detectMultiFingerGestures()
    }
  })

  return {
    // 상태
    isAdvancedGesturing: computed(() => isAdvancedGesturing.value),
    currentRotation: computed(() => currentRotation.value),
    
    // 기본 제스처 메서드 노출
    ...baseGestures,
    
    // 고급 제스처 메서드
    on,
    off,
    
    // 유틸리티
    calculateCentroid,
    calculateRotation,
    calculateAdvancedVelocity,
    calculateInertia,
    detectMultiFingerPattern,
    analyzeGesturePattern,
    triggerHapticFeedback,
    triggerSoundFeedback,
  }
}