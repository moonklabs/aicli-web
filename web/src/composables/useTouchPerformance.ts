import { ref, onMounted, onBeforeUnmount } from 'vue'

interface TouchPerformanceOptions {
  // 터치 지연 최소화
  fastClick?: boolean
  // 스크롤 최적화
  optimizeScroll?: boolean
  // 디바운스 설정
  debounceMs?: number
  // 쓰로틀 설정
  throttleMs?: number
  // 터치 피드백
  hapticFeedback?: boolean
}

export function useTouchPerformance(options: TouchPerformanceOptions = {}) {
  const {
    fastClick = true,
    optimizeScroll = true,
    debounceMs = 100,
    throttleMs = 16, // 60fps
    hapticFeedback = true,
  } = options

  // 성능 메트릭
  const touchLatency = ref(0)
  const frameRate = ref(60)
  const lastFrameTime = ref(0)
  const isHighPerformanceMode = ref(false)

  // 터치 상태 추적
  const touchStartTime = ref(0)
  const touchEndTime = ref(0)
  const activeTouches = ref(0)

  // 성능 모니터링
  const measureFrameRate = () => {
    const now = performance.now()
    if (lastFrameTime.value > 0) {
      const delta = now - lastFrameTime.value
      frameRate.value = Math.round(1000 / delta)
    }
    lastFrameTime.value = now
    requestAnimationFrame(measureFrameRate)
  }

  // 터치 지연 측정
  const measureTouchLatency = (startTime: number, endTime: number) => {
    touchLatency.value = endTime - startTime
  }

  // 빠른 클릭 최적화
  const optimizeFastClick = () => {
    if (!fastClick) return

    // CSS touch-action 설정
    document.documentElement.style.touchAction = 'manipulation'
    
    // iOS Safari의 300ms 지연 제거
    const meta = document.createElement('meta')
    meta.name = 'viewport'
    meta.content = 'width=device-width, initial-scale=1, user-scalable=no'
    
    if (!document.querySelector('meta[name="viewport"]')) {
      document.head.appendChild(meta)
    }
  }

  // 스크롤 최적화
  const optimizeScrollPerformance = () => {
    if (!optimizeScroll) return

    // CSS 스크롤 최적화
    const style = document.createElement('style')
    style.textContent = `
      * {
        -webkit-overflow-scrolling: touch;
        scroll-behavior: smooth;
      }
      
      @media (prefers-reduced-motion: reduce) {
        * {
          scroll-behavior: auto;
        }
      }
      
      /* GPU 가속 활성화 */
      .optimized-scroll {
        transform: translateZ(0);
        will-change: scroll-position;
      }
      
      /* 모바일 텍스트 렌더링 최적화 */
      @media (max-width: 768px) {
        * {
          -webkit-font-smoothing: antialiased;
          -moz-osx-font-smoothing: grayscale;
          text-rendering: optimizeSpeed;
        }
      }
    `
    document.head.appendChild(style)
  }

  // 디바운스 함수
  const debounce = <T extends (...args: any[]) => any>(
    func: T,
    delay: number = debounceMs
  ): T => {
    let timeoutId: NodeJS.Timeout
    return ((...args: any[]) => {
      clearTimeout(timeoutId)
      timeoutId = setTimeout(() => func(...args), delay)
    }) as T
  }

  // 쓰로틀 함수
  const throttle = <T extends (...args: any[]) => any>(
    func: T,
    delay: number = throttleMs
  ): T => {
    let lastCall = 0
    return ((...args: any[]) => {
      const now = Date.now()
      if (now - lastCall >= delay) {
        lastCall = now
        return func(...args)
      }
    }) as T
  }

  // 햅틱 피드백
  const triggerHaptic = (type: 'light' | 'medium' | 'heavy' = 'light') => {
    if (!hapticFeedback || !('vibrate' in navigator)) return

    const patterns = {
      light: [10],
      medium: [20],
      heavy: [30],
    }

    navigator.vibrate(patterns[type])
  }

  // 배터리 상태 기반 성능 모드 조정
  const adaptPerformanceMode = async () => {
    try {
      // @ts-ignore - Battery API는 실험적 기능
      const battery = await navigator.getBattery?.()
      
      if (battery) {
        const updatePerformanceMode = () => {
          // 배터리가 20% 이하이거나 충전 중이 아닐 때 고성능 모드 비활성화
          isHighPerformanceMode.value = battery.level > 0.2 || battery.charging
        }

        battery.addEventListener('levelchange', updatePerformanceMode)
        battery.addEventListener('chargingchange', updatePerformanceMode)
        updatePerformanceMode()
      }
    } catch (error) {
      // Battery API를 지원하지 않는 경우 기본 고성능 모드 유지
      isHighPerformanceMode.value = true
    }
  }

  // 터치 이벤트 최적화
  const optimizeTouchEvents = () => {
    const handleTouchStart = (event: TouchEvent) => {
      touchStartTime.value = performance.now()
      activeTouches.value = event.touches.length
    }

    const handleTouchEnd = (event: TouchEvent) => {
      touchEndTime.value = performance.now()
      measureTouchLatency(touchStartTime.value, touchEndTime.value)
      activeTouches.value = event.touches.length
    }

    // Passive 이벤트 리스너 사용으로 성능 향상
    document.addEventListener('touchstart', handleTouchStart, { passive: true })
    document.addEventListener('touchend', handleTouchEnd, { passive: true })

    return () => {
      document.removeEventListener('touchstart', handleTouchStart)
      document.removeEventListener('touchend', handleTouchEnd)
    }
  }

  // 메모리 사용량 모니터링
  const getMemoryUsage = () => {
    // @ts-ignore - memory API는 Chrome에서만 지원
    const memory = (performance as any).memory
    
    if (memory) {
      return {
        used: Math.round(memory.usedJSHeapSize / 1048576), // MB
        total: Math.round(memory.totalJSHeapSize / 1048576), // MB
        limit: Math.round(memory.jsHeapSizeLimit / 1048576), // MB
      }
    }
    
    return null
  }

  // 성능 진단
  const getDiagnostics = () => {
    return {
      touchLatency: touchLatency.value,
      frameRate: frameRate.value,
      isHighPerformanceMode: isHighPerformanceMode.value,
      activeTouches: activeTouches.value,
      memoryUsage: getMemoryUsage(),
      userAgent: navigator.userAgent,
      devicePixelRatio: window.devicePixelRatio,
      viewport: {
        width: window.innerWidth,
        height: window.innerHeight,
      },
      connection: (navigator as any).connection ? {
        effectiveType: (navigator as any).connection.effectiveType,
        downlink: (navigator as any).connection.downlink,
        rtt: (navigator as any).connection.rtt,
      } : null,
    }
  }

  // RequestIdleCallback 폴리필
  const requestIdleCallback = window.requestIdleCallback || 
    ((callback: IdleRequestCallback) => {
      const start = Date.now()
      return setTimeout(() => {
        callback({
          didTimeout: false,
          timeRemaining: () => Math.max(0, 50 - (Date.now() - start)),
        })
      }, 1)
    })

  // 아이들 시간에 작업 수행
  const scheduleIdleWork = (work: () => void) => {
    requestIdleCallback(() => {
      if (isHighPerformanceMode.value) {
        work()
      }
    })
  }

  // 초기화
  const initialize = () => {
    optimizeFastClick()
    optimizeScrollPerformance()
    adaptPerformanceMode()
    measureFrameRate()
    
    return optimizeTouchEvents()
  }

  // 정리 함수
  let cleanup: (() => void) | null = null

  onMounted(() => {
    cleanup = initialize()
  })

  onBeforeUnmount(() => {
    cleanup?.()
  })

  return {
    // 상태
    touchLatency,
    frameRate,
    isHighPerformanceMode,
    activeTouches,

    // 유틸리티 함수
    debounce,
    throttle,
    triggerHaptic,
    scheduleIdleWork,
    getDiagnostics,

    // 성능 측정
    measureTouchLatency,
    getMemoryUsage,
  }
}

// 터치 디바이스 감지
export function useDeviceDetection() {
  const isTouchDevice = ref(false)
  const isIOS = ref(false)
  const isAndroid = ref(false)
  const isMobile = ref(false)
  const isTablet = ref(false)
  const devicePixelRatio = ref(1)

  const detectDevice = () => {
    // 터치 지원 감지
    isTouchDevice.value = 'ontouchstart' in window || 
      navigator.maxTouchPoints > 0 || 
      (navigator as any).msMaxTouchPoints > 0

    // OS 감지
    const userAgent = navigator.userAgent
    isIOS.value = /iPad|iPhone|iPod/.test(userAgent)
    isAndroid.value = /Android/.test(userAgent)

    // 디바이스 타입 감지
    const width = window.innerWidth
    const height = window.innerHeight
    const minDimension = Math.min(width, height)
    const maxDimension = Math.max(width, height)

    isMobile.value = isTouchDevice.value && minDimension < 768
    isTablet.value = isTouchDevice.value && minDimension >= 768 && maxDimension < 1024

    // 픽셀 비율
    devicePixelRatio.value = window.devicePixelRatio || 1
  }

  onMounted(() => {
    detectDevice()
    
    // 화면 방향 변경 감지
    window.addEventListener('orientationchange', () => {
      setTimeout(detectDevice, 100)
    })
    
    window.addEventListener('resize', detectDevice)
  })

  return {
    isTouchDevice,
    isIOS,
    isAndroid,
    isMobile,
    isTablet,
    devicePixelRatio,
  }
}