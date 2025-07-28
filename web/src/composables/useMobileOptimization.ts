import { ref, onMounted, onBeforeUnmount, watch, nextTick } from 'vue'
import { useTouchPerformance, useDeviceDetection } from './useTouchPerformance'

interface MobileOptimizationOptions {
  enableVirtualKeyboardOptimization?: boolean
  enableImageLazyLoading?: boolean
  enableIntersectionObserver?: boolean
  enablePrefetching?: boolean
  enableMemoryOptimization?: boolean
  enableBatteryOptimization?: boolean
}

export function useMobileOptimization(options: MobileOptimizationOptions = {}) {
  const {
    enableVirtualKeyboardOptimization = true,
    enableImageLazyLoading = true,
    enableIntersectionObserver = true,
    enablePrefetching = true,
    enableMemoryOptimization = true,
    enableBatteryOptimization = true,
  } = options

  const { isHighPerformanceMode, scheduleIdleWork } = useTouchPerformance()
  const { isMobile, isIOS } = useDeviceDetection()

  // 가상 키보드 상태
  const isVirtualKeyboardOpen = ref(false)
  const initialViewportHeight = ref(0)
  const currentViewportHeight = ref(0)

  // 이미지 지연 로딩
  const lazyImageObserver = ref<IntersectionObserver | null>(null)
  const lazyImages = ref<Set<HTMLImageElement>>(new Set())

  // 프리페칭 큐
  const prefetchQueue = ref<string[]>([])
  const prefetchedResources = ref<Set<string>>(new Set())

  // 메모리 정리 상태
  const memoryPressure = ref(false)
  const lastCleanupTime = ref(0)

  // 가상 키보드 최적화
  const setupVirtualKeyboardOptimization = () => {
    if (!enableVirtualKeyboardOptimization || !isMobile.value) return

    initialViewportHeight.value = window.innerHeight

    const handleViewportChange = () => {
      currentViewportHeight.value = window.innerHeight
      const heightDiff = initialViewportHeight.value - currentViewportHeight.value
      
      // 높이가 150px 이상 줄어들면 가상 키보드가 열린 것으로 간주
      isVirtualKeyboardOpen.value = heightDiff > 150

      if (isVirtualKeyboardOpen.value) {
        // 가상 키보드가 열렸을 때의 최적화
        document.body.style.height = `${currentViewportHeight.value}px`
        document.body.style.overflow = 'hidden'
      } else {
        // 가상 키보드가 닫혔을 때 복원
        document.body.style.height = ''
        document.body.style.overflow = ''
      }
    }

    // iOS의 경우 visualViewport API 사용
    if (isIOS.value && window.visualViewport) {
      window.visualViewport.addEventListener('resize', handleViewportChange)
    } else {
      // Android의 경우 window resize 이벤트 사용
      window.addEventListener('resize', handleViewportChange)
    }

    return () => {
      if (isIOS.value && window.visualViewport) {
        window.visualViewport.removeEventListener('resize', handleViewportChange)
      } else {
        window.removeEventListener('resize', handleViewportChange)
      }
    }
  }

  // 이미지 지연 로딩 설정
  const setupImageLazyLoading = () => {
    if (!enableImageLazyLoading || !enableIntersectionObserver) return

    const imageObserverCallback = (entries: IntersectionObserverEntry[]) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting) {
          const img = entry.target as HTMLImageElement
          const dataSrc = img.dataset.src

          if (dataSrc) {
            // 고해상도 이미지 지원
            const devicePixelRatio = window.devicePixelRatio || 1
            const src = devicePixelRatio > 1 && img.dataset.srcHd ? 
              img.dataset.srcHd : dataSrc

            img.src = src
            img.classList.remove('lazy')
            img.classList.add('loaded')
            
            lazyImageObserver.value?.unobserve(img)
            lazyImages.value.delete(img)
          }
        }
      })
    }

    lazyImageObserver.value = new IntersectionObserver(imageObserverCallback, {
      rootMargin: '50px',
      threshold: 0.1,
    })

    return () => {
      lazyImageObserver.value?.disconnect()
    }
  }

  // 리소스 프리페칭
  const setupPrefetching = () => {
    if (!enablePrefetching) return

    const prefetchResource = (url: string, type: 'script' | 'style' | 'image' = 'script') => {
      if (prefetchedResources.value.has(url)) return

      scheduleIdleWork(() => {
        const link = document.createElement('link')
        link.rel = 'prefetch'
        link.href = url
        link.as = type
        document.head.appendChild(link)
        
        prefetchedResources.value.add(url)
      })
    }

    const processPrefetchQueue = () => {
      if (prefetchQueue.value.length === 0 || !isHighPerformanceMode.value) return

      const url = prefetchQueue.value.shift()
      if (url) {
        prefetchResource(url)
      }

      // 다음 프리페치 스케줄링
      if (prefetchQueue.value.length > 0) {
        setTimeout(processPrefetchQueue, 100)
      }
    }

    return { prefetchResource, processPrefetchQueue }
  }

  // 메모리 최적화
  const setupMemoryOptimization = () => {
    if (!enableMemoryOptimization) return

    const checkMemoryPressure = () => {
      // @ts-ignore
      const memory = (performance as any).memory
      
      if (memory) {
        const usageRatio = memory.usedJSHeapSize / memory.jsHeapSizeLimit
        memoryPressure.value = usageRatio > 0.8
      }
    }

    const cleanupUnusedResources = () => {
      const now = Date.now()
      if (now - lastCleanupTime.value < 30000) return // 30초마다 실행

      scheduleIdleWork(() => {
        // 사용하지 않는 이미지 정리
        document.querySelectorAll('img').forEach((img) => {
          const rect = img.getBoundingClientRect()
          const isVisible = rect.top < window.innerHeight && rect.bottom > 0

          if (!isVisible && img.src && !img.dataset.keepLoaded) {
            img.src = ''
          }
        })

        // 캐시 정리
        if ('caches' in window) {
          caches.keys().then((names) => {
            names.forEach((name) => {
              if (name.includes('old') || name.includes('temp')) {
                caches.delete(name)
              }
            })
          })
        }

        lastCleanupTime.value = now
      })
    }

    // 메모리 압박 상황에서 자동 정리
    watch(memoryPressure, (pressure) => {
      if (pressure) {
        cleanupUnusedResources()
      }
    })

    // 정기적인 메모리 체크
    const memoryInterval = setInterval(checkMemoryPressure, 10000) // 10초마다

    return () => {
      clearInterval(memoryInterval)
    }
  }

  // 배터리 최적화
  const setupBatteryOptimization = async () => {
    if (!enableBatteryOptimization) return

    try {
      // @ts-ignore
      const battery = await navigator.getBattery?.()
      
      if (battery) {
        const optimizeForBattery = () => {
          const lowBattery = battery.level < 0.2 && !battery.charging
          
          if (lowBattery) {
            // 저전력 모드 활성화
            document.body.classList.add('low-battery-mode')
            
            // 애니메이션 비활성화
            const style = document.createElement('style')
            style.textContent = `
              .low-battery-mode * {
                animation-duration: 0s !important;
                transition-duration: 0s !important;
              }
            `
            document.head.appendChild(style)
          } else {
            document.body.classList.remove('low-battery-mode')
          }
        }

        battery.addEventListener('levelchange', optimizeForBattery)
        battery.addEventListener('chargingchange', optimizeForBattery)
        optimizeForBattery()
      }
    } catch (error) {
      console.warn('Battery API not supported')
    }
  }

  // 터치 스크롤링 최적화
  const optimizeTouchScrolling = () => {
    const optimizeScrollElements = () => {
      document.querySelectorAll('[data-scroll-optimize]').forEach((element) => {
        const el = element as HTMLElement
        el.style.WebkitOverflowScrolling = 'touch'
        el.style.transform = 'translateZ(0)'
        el.style.willChange = 'scroll-position'
      })
    }

    scheduleIdleWork(optimizeScrollElements)
  }

  // 이미지 등록 및 해제
  const registerLazyImage = (img: HTMLImageElement) => {
    if (!lazyImageObserver.value) return

    lazyImages.value.add(img)
    lazyImageObserver.value.observe(img)
  }

  const unregisterLazyImage = (img: HTMLImageElement) => {
    if (!lazyImageObserver.value) return

    lazyImages.value.delete(img)
    lazyImageObserver.value.unobserve(img)
  }

  // 프리페치 큐에 리소스 추가
  const addToPrefetchQueue = (url: string) => {
    if (!prefetchedResources.value.has(url)) {
      prefetchQueue.value.push(url)
    }
  }

  // 초기화 및 정리
  let cleanupFunctions: (() => void)[] = []

  onMounted(() => {
    nextTick(() => {
      const keyboardCleanup = setupVirtualKeyboardOptimization()
      const imageCleanup = setupImageLazyLoading()
      const memoryCleanup = setupMemoryOptimization()
      const prefetching = setupPrefetching()

      if (keyboardCleanup) cleanupFunctions.push(keyboardCleanup)
      if (imageCleanup) cleanupFunctions.push(imageCleanup)
      if (memoryCleanup) cleanupFunctions.push(memoryCleanup)

      setupBatteryOptimization()
      optimizeTouchScrolling()

      // 프리페치 큐 처리 시작
      if (prefetching) {
        setTimeout(prefetching.processPrefetchQueue, 1000)
      }
    })
  })

  onBeforeUnmount(() => {
    cleanupFunctions.forEach(cleanup => cleanup())
    cleanupFunctions = []
  })

  return {
    // 상태
    isVirtualKeyboardOpen,
    memoryPressure,
    prefetchQueue,

    // 메서드
    registerLazyImage,
    unregisterLazyImage,
    addToPrefetchQueue,
    optimizeTouchScrolling,

    // 유틸리티
    currentViewportHeight,
    initialViewportHeight,
  }
}

// 이미지 최적화 디렉티브
export const vLazyImage = {
  mounted(el: HTMLImageElement, binding: any) {
    const { registerLazyImage } = binding.instance?.setupState || {}
    
    if (registerLazyImage) {
      // data-src 속성이 있으면 지연 로딩 적용
      if (el.dataset.src) {
        el.classList.add('lazy')
        registerLazyImage(el)
      }
    }
  },
  
  beforeUnmount(el: HTMLImageElement, binding: any) {
    const { unregisterLazyImage } = binding.instance?.setupState || {}
    
    if (unregisterLazyImage) {
      unregisterLazyImage(el)
    }
  },
}

// 성능 모니터링 유틸리티
export function usePerformanceMonitor() {
  const metrics = ref({
    fps: 0,
    memoryUsage: 0,
    loadTime: 0,
    interactionLatency: 0,
  })

  const startMonitoring = () => {
    let lastTime = performance.now()
    let frameCount = 0

    const measureFPS = () => {
      frameCount++
      const currentTime = performance.now()
      
      if (currentTime - lastTime >= 1000) {
        metrics.value.fps = Math.round(frameCount * 1000 / (currentTime - lastTime))
        frameCount = 0
        lastTime = currentTime
      }
      
      requestAnimationFrame(measureFPS)
    }

    measureFPS()

    // 메모리 사용량 모니터링
    const measureMemory = () => {
      // @ts-ignore
      const memory = (performance as any).memory
      if (memory) {
        metrics.value.memoryUsage = Math.round(memory.usedJSHeapSize / 1048576)
      }
    }

    setInterval(measureMemory, 5000)

    // 페이지 로드 시간
    window.addEventListener('load', () => {
      metrics.value.loadTime = performance.now()
    })
  }

  onMounted(() => {
    startMonitoring()
  })

  return {
    metrics,
  }
}