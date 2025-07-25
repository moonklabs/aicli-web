import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { Ref } from 'vue'

interface PerformanceConfig {
  // 디바운스/스로틀 설정
  debounceMs?: number
  throttleMs?: number

  // 가상화 설정
  virtualScrollEnabled?: boolean
  itemHeight?: number
  overscan?: number

  // 지연 로딩 설정
  lazyLoading?: boolean
  loadingThreshold?: number
  batchSize?: number

  // 메모이제이션 설정
  enableMemoization?: boolean
  maxCacheSize?: number

  // 성능 모니터링
  enableProfiling?: boolean
  logPerformance?: boolean
}

interface PerformanceMetrics {
  renderTime: number
  updateTime: number
  memoryUsage: number
  domNodes: number
  lastMeasurement: number
}

export function usePerformanceOptimization(config: PerformanceConfig = {}) {
  const defaultConfig: Required<PerformanceConfig> = {
    debounceMs: 300,
    throttleMs: 100,
    virtualScrollEnabled: false,
    itemHeight: 40,
    overscan: 5,
    lazyLoading: false,
    loadingThreshold: 100,
    batchSize: 50,
    enableMemoization: true,
    maxCacheSize: 100,
    enableProfiling: false,
    logPerformance: false,
  }

  const settings = { ...defaultConfig, ...config }

  // 성능 메트릭스
  const metrics = ref<PerformanceMetrics>({
    renderTime: 0,
    updateTime: 0,
    memoryUsage: 0,
    domNodes: 0,
    lastMeasurement: Date.now(),
  })

  // 캐시 관리
  const memoCache = ref(new Map<string, any>())
  const cacheHits = ref(0)
  const cacheMisses = ref(0)

  // 성능 관찰자
  const performanceObserver = ref<PerformanceObserver | null>(null)
  const mutationObserver = ref<MutationObserver | null>(null)

  // 지연 로딩 상태
  const loadedItems = ref(new Set<number>())
  const loadingItems = ref(new Set<number>())

  // 디바운스 함수
  const debounce = <T extends (...args: any[]) => any>(
    func: T,
    delay: number = settings.debounceMs,
  ): ((...args: Parameters<T>) => void) => {
    let timeoutId: number

    return (...args: Parameters<T>) => {
      clearTimeout(timeoutId)
      timeoutId = setTimeout(() => func(...args), delay)
    }
  }

  // 스로틀 함수
  const throttle = <T extends (...args: any[]) => any>(
    func: T,
    delay: number = settings.throttleMs,
  ): ((...args: Parameters<T>) => void) => {
    let lastCall = 0

    return (...args: Parameters<T>) => {
      const now = Date.now()
      if (now - lastCall >= delay) {
        lastCall = now
        func(...args)
      }
    }
  }

  // 메모이제이션 함수
  const memoize = <T extends (...args: any[]) => any>(
    func: T,
    keyGenerator?: (...args: Parameters<T>) => string,
  ): T => {
    if (!settings.enableMemoization) {
      return func
    }

    return ((...args: Parameters<T>) => {
      const key = keyGenerator ? keyGenerator(...args) : JSON.stringify(args)

      if (memoCache.value.has(key)) {
        cacheHits.value++
        return memoCache.value.get(key)
      }

      cacheMisses.value++
      const result = func(...args)

      // 캐시 크기 제한
      if (memoCache.value.size >= settings.maxCacheSize) {
        const firstKey = memoCache.value.keys().next().value
        memoCache.value.delete(firstKey)
      }

      memoCache.value.set(key, result)
      return result
    }) as T
  }

  // 배치 처리 함수
  const batchProcess = <T>(
    items: T[],
    processor: (batch: T[]) => Promise<void>,
    batchSize: number = settings.batchSize,
  ): Promise<void> => {
    return new Promise(async (resolve, reject) => {
      try {
        for (let i = 0; i < items.length; i += batchSize) {
          const batch = items.slice(i, i + batchSize)
          await processor(batch)

          // 다음 배치 처리 전 잠시 대기 (UI 블로킹 방지)
          await new Promise(resolve => setTimeout(resolve, 1))
        }
        resolve()
      } catch (error) {
        reject(error)
      }
    })
  }

  // 가상 스크롤링 계산
  const calculateVirtualItems = (
    scrollTop: number,
    containerHeight: number,
    totalItems: number,
  ) => {
    if (!settings.virtualScrollEnabled) {
      return { startIndex: 0, endIndex: totalItems, offsetY: 0 }
    }

    const itemHeight = settings.itemHeight
    const overscan = settings.overscan

    const startIndex = Math.max(0, Math.floor(scrollTop / itemHeight) - overscan)
    const visibleCount = Math.ceil(containerHeight / itemHeight)
    const endIndex = Math.min(totalItems, startIndex + visibleCount + overscan * 2)
    const offsetY = startIndex * itemHeight

    return { startIndex, endIndex, offsetY }
  }

  // 지연 로딩 관리
  const shouldLoadItem = (index: number, threshold: number = settings.loadingThreshold): boolean => {
    if (!settings.lazyLoading) return true
    if (loadedItems.value.has(index)) return true
    if (loadingItems.value.has(index)) return false

    return index < threshold
  }

  const loadItem = async (index: number, loader: () => Promise<any>): Promise<any> => {
    if (loadedItems.value.has(index)) {
      return // 이미 로드됨
    }

    if (loadingItems.value.has(index)) {
      return // 이미 로딩 중
    }

    try {
      loadingItems.value.add(index)
      const result = await loader()
      loadedItems.value.add(index)
      return result
    } finally {
      loadingItems.value.delete(index)
    }
  }

  // 성능 측정 시작
  const startPerformanceMeasurement = (name: string) => {
    if (settings.enableProfiling && performance.mark) {
      performance.mark(`${name}-start`)
    }
  }

  // 성능 측정 종료
  const endPerformanceMeasurement = (name: string) => {
    if (settings.enableProfiling && performance.mark && performance.measure) {
      performance.mark(`${name}-end`)
      performance.measure(name, `${name}-start`, `${name}-end`)

      const measurement = performance.getEntriesByName(name, 'measure')[0]
      if (measurement && settings.logPerformance) {
        console.log(`[Performance] ${name}: ${measurement.duration.toFixed(2)}ms`)
      }

      return measurement.duration
    }
    return 0
  }

  // DOM 노드 수 계산
  const countDOMNodes = (element?: Element): number => {
    if (!element) {
      return document.querySelectorAll('*').length
    }

    return element.querySelectorAll('*').length + 1 // +1 for the element itself
  }

  // 메모리 사용량 측정 (근사치)
  const measureMemoryUsage = (): number => {
    if ('memory' in performance) {
      return (performance as any).memory.usedJSHeapSize / 1024 / 1024 // MB 단위
    }
    return 0
  }

  // 성능 메트릭스 업데이트
  const updateMetrics = (element?: Element) => {
    const now = Date.now()
    const timeDiff = now - metrics.value.lastMeasurement

    metrics.value = {
      renderTime: endPerformanceMeasurement('render'),
      updateTime: timeDiff,
      memoryUsage: measureMemoryUsage(),
      domNodes: countDOMNodes(element),
      lastMeasurement: now,
    }
  }

  // 성능 최적화된 배열 정렬
  const optimizedSort = <T>(
    array: T[],
    compareFn: (a: T, b: T) => number,
    useWorker = false,
  ): Promise<T[]> => {
    return new Promise((resolve) => {
      if (useWorker && array.length > 10000 && typeof Worker !== 'undefined') {
        // Web Worker를 사용한 정렬 (실제 구현 시 별도 워커 파일 필요)
        const sortedArray = [...array].sort(compareFn)
        resolve(sortedArray)
      } else {
        // 메인 스레드에서 배치 정렬
        const batchSize = 1000
        let sortedArray = [...array]

        if (array.length > batchSize) {
          // 큰 배열의 경우 배치로 나누어 정렬
          const batches: T[][] = []
          for (let i = 0; i < array.length; i += batchSize) {
            batches.push(array.slice(i, i + batchSize))
          }

          // 각 배치를 정렬하고 병합
          const sortedBatches = batches.map(batch => batch.sort(compareFn))
          sortedArray = mergeSortedArrays(sortedBatches, compareFn)
        } else {
          sortedArray.sort(compareFn)
        }

        resolve(sortedArray)
      }
    })
  }

  // 정렬된 배열들을 병합
  const mergeSortedArrays = <T>(
    arrays: T[][],
    compareFn: (a: T, b: T) => number,
  ): T[] => {
    if (arrays.length === 0) return []
    if (arrays.length === 1) return arrays[0]

    const result: T[] = []
    const pointers = new Array(arrays.length).fill(0)

    while (true) {
      let minIndex = -1
      let minValue: T | null = null

      // 각 배열의 현재 포인터에서 최소값 찾기
      for (let i = 0; i < arrays.length; i++) {
        if (pointers[i] < arrays[i].length) {
          const value = arrays[i][pointers[i]]
          if (minValue === null || compareFn(value, minValue) < 0) {
            minValue = value
            minIndex = i
          }
        }
      }

      if (minIndex === -1) break // 모든 배열이 처리됨

      result.push(minValue!)
      pointers[minIndex]++
    }

    return result
  }

  // 성능 프로파일링 설정
  const setupPerformanceObserver = () => {
    if (!settings.enableProfiling || !PerformanceObserver) return

    try {
      performanceObserver.value = new PerformanceObserver((list) => {
        const entries = list.getEntries()

        entries.forEach((entry) => {
          if (settings.logPerformance) {
            console.log(`[Performance] ${entry.name}: ${entry.duration.toFixed(2)}ms`)
          }
        })
      })

      performanceObserver.value.observe({ entryTypes: ['measure'] })
    } catch (error) {
      console.warn('Performance Observer not supported', error)
    }
  }

  // DOM 변화 감지
  const setupMutationObserver = (element: Element) => {
    if (!MutationObserver) return

    mutationObserver.value = new MutationObserver((mutations) => {
      let significantChange = false

      mutations.forEach((mutation) => {
        if (mutation.type === 'childList' && mutation.addedNodes.length > 10) {
          significantChange = true
        }
      })

      if (significantChange) {
        throttledUpdateMetrics()
      }
    })

    mutationObserver.value.observe(element, {
      childList: true,
      subtree: true,
      attributes: false,
      characterData: false,
    })
  }

  // 스로틀된 메트릭스 업데이트
  const throttledUpdateMetrics = throttle(updateMetrics, 1000)

  // 캐시 통계
  const getCacheStats = () => ({
    size: memoCache.value.size,
    hits: cacheHits.value,
    misses: cacheMisses.value,
    hitRate: cacheHits.value / (cacheHits.value + cacheMisses.value) || 0,
  })

  // 캐시 지우기
  const clearCache = () => {
    memoCache.value.clear()
    cacheHits.value = 0
    cacheMisses.value = 0
  }

  // 성능 리포트 생성
  const generatePerformanceReport = () => ({
    metrics: metrics.value,
    cache: getCacheStats(),
    virtualScroll: {
      enabled: settings.virtualScrollEnabled,
      itemHeight: settings.itemHeight,
      overscan: settings.overscan,
    },
    lazyLoading: {
      enabled: settings.lazyLoading,
      loadedItems: loadedItems.value.size,
      loadingItems: loadingItems.value.size,
    },
    timestamp: new Date().toISOString(),
  })

  // 초기화
  onMounted(() => {
    setupPerformanceObserver()
  })

  // 정리
  onBeforeUnmount(() => {
    if (performanceObserver.value) {
      performanceObserver.value.disconnect()
    }

    if (mutationObserver.value) {
      mutationObserver.value.disconnect()
    }

    clearCache()
  })

  return {
    // 상태
    metrics,
    loadedItems,
    loadingItems,

    // 최적화 함수들
    debounce,
    throttle,
    memoize,
    batchProcess,

    // 가상화
    calculateVirtualItems,

    // 지연 로딩
    shouldLoadItem,
    loadItem,

    // 성능 측정
    startPerformanceMeasurement,
    endPerformanceMeasurement,
    updateMetrics,

    // 정렬 최적화
    optimizedSort,

    // 캐시 관리
    getCacheStats,
    clearCache,

    // 옵저버 설정
    setupMutationObserver,

    // 리포트
    generatePerformanceReport,

    // 유틸리티
    countDOMNodes,
    measureMemoryUsage,
  }
}