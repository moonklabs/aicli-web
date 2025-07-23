/**
 * 터미널 성능 최적화를 위한 컴포저블
 */

import { computed, onUnmounted, reactive, ref } from 'vue'
import { debounce, throttle } from '@/utils/terminal-utils'

export interface TerminalOptimizationOptions {
  maxLines?: number
  updateInterval?: number
  batchSize?: number
  enableVirtualization?: boolean
  memoryThreshold?: number
  renderDelay?: number
}

export interface TerminalPerformanceMetrics {
  renderCount: number
  avgRenderTime: number
  memoryUsage: number
  scrollEvents: number
  lastUpdateTime: number
  batchedUpdates: number
}

export interface OptimizedTerminalLine {
  id: string
  content: string
  rendered?: string
  timestamp: number
  type: 'input' | 'output' | 'error' | 'system'
  visible?: boolean
  height?: number
}

export function useTerminalOptimization(options: TerminalOptimizationOptions = {}) {
  const {
    maxLines = 1000,
    updateInterval = 50,
    batchSize = 10,
    enableVirtualization = true,
    memoryThreshold = 50 * 1024 * 1024, // 50MB
    renderDelay = 16, // 60fps
  } = options

  // 상태 관리
  const lines = ref<OptimizedTerminalLine[]>([])
  const pendingLines = ref<OptimizedTerminalLine[]>([])
  const isProcessing = ref(false)
  const lastRenderTime = ref(0)

  // 성능 메트릭스
  const metrics = reactive<TerminalPerformanceMetrics>({
    renderCount: 0,
    avgRenderTime: 0,
    memoryUsage: 0,
    scrollEvents: 0,
    lastUpdateTime: 0,
    batchedUpdates: 0,
  })

  // 메모리 사용량 추정
  const estimatedMemoryUsage = computed(() => {
    const linesMemory = lines.value.reduce((total, line) => {
      return total + (line.content?.length || 0) + (line.rendered?.length || 0) + 100 // 기본 오버헤드
    }, 0)
    return linesMemory * 2 // 문자열은 UTF-16이므로 2배
  })

  // 메모리 압박 상태 감지
  const isMemoryPressure = computed(() => {
    return estimatedMemoryUsage.value > memoryThreshold
  })

  // 가시 영역 관리
  const visibleRange = ref({ start: 0, end: 100 })
  const visibleLines = computed(() => {
    if (!enableVirtualization) return lines.value

    const { start, end } = visibleRange.value
    return lines.value.slice(start, Math.min(end, lines.value.length))
  })

  // 배치 처리를 위한 디바운스 함수
  const processPendingLines = debounce(() => {
    if (pendingLines.value.length === 0) return

    const startTime = performance.now()
    isProcessing.value = true

    // 배치 크기만큼 처리
    const batchToProcess = pendingLines.value.splice(0, batchSize)

    // 라인 추가 및 전처리
    const processedLines = batchToProcess.map(line => ({
      ...line,
      visible: isLineInVisibleRange(lines.value.length),
    }))

    lines.value.push(...processedLines)

    // 최대 라인 수 제한
    if (lines.value.length > maxLines) {
      const excessLines = lines.value.length - maxLines
      lines.value.splice(0, excessLines)
    }

    // 메모리 압박 시 렌더링된 콘텐츠 정리
    if (isMemoryPressure.value) {
      cleanupRenderedContent()
    }

    // 메트릭스 업데이트
    const renderTime = performance.now() - startTime
    updateMetrics(renderTime)

    isProcessing.value = false
    lastRenderTime.value = Date.now()

    // 더 처리할 라인이 있다면 계속 처리
    if (pendingLines.value.length > 0) {
      processPendingLines()
    }
  }, updateInterval)

  // 스로틀된 업데이트 함수
  const throttledUpdate = throttle(() => {
    if (pendingLines.value.length > 0) {
      processPendingLines()
    }
  }, renderDelay)

  // 라인이 가시 영역에 있는지 확인
  const isLineInVisibleRange = (index: number): boolean => {
    const { start, end } = visibleRange.value
    return index >= start && index <= end
  }

  // 렌더링된 콘텐츠 정리 (메모리 절약)
  const cleanupRenderedContent = () => {
    const { start, end } = visibleRange.value

    lines.value.forEach((line, index) => {
      if (index < start - 50 || index > end + 50) {
        // 가시 영역에서 충분히 멀리 떨어진 라인의 렌더링된 콘텐츠 제거
        if (line.rendered) {
          delete line.rendered
        }
      }
    })
  }

  // 성능 메트릭스 업데이트
  const updateMetrics = (renderTime: number) => {
    metrics.renderCount++
    metrics.avgRenderTime = (metrics.avgRenderTime * (metrics.renderCount - 1) + renderTime) / metrics.renderCount
    metrics.memoryUsage = estimatedMemoryUsage.value
    metrics.lastUpdateTime = Date.now()
    metrics.batchedUpdates++
  }

  // 가시 영역 업데이트
  const updateVisibleRange = (start: number, end: number) => {
    visibleRange.value = { start, end }

    // 가시 영역 변경 시 해당 라인들의 visible 플래그 업데이트
    lines.value.forEach((line, index) => {
      line.visible = isLineInVisibleRange(index)
    })
  }

  // 스크롤 이벤트 처리 (스로틀링)
  const handleScroll = throttle((scrollInfo: { top: number; isAtBottom: boolean }) => {
    metrics.scrollEvents++

    if (enableVirtualization) {
      // 스크롤 위치에 따라 가시 영역 계산
      const lineHeight = 20 // 기본 라인 높이
      const containerHeight = 400 // 기본 컨테이너 높이

      const start = Math.floor(scrollInfo.top / lineHeight)
      const visibleCount = Math.ceil(containerHeight / lineHeight)
      const end = start + visibleCount + 10 // 여유분

      updateVisibleRange(start, end)
    }
  }, 16) // 60fps

  // 라인 추가 (최적화된)
  const addLine = (line: Omit<OptimizedTerminalLine, 'id' | 'timestamp'>): void => {
    const optimizedLine: OptimizedTerminalLine = {
      ...line,
      id: `line-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
      timestamp: Date.now(),
      visible: false,
    }

    pendingLines.value.push(optimizedLine)
    throttledUpdate()
  }

  // 여러 라인 추가 (배치)
  const addLines = (newLines: Omit<OptimizedTerminalLine, 'id' | 'timestamp'>[]): void => {
    const optimizedLines = newLines.map(line => ({
      ...line,
      id: `line-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
      timestamp: Date.now(),
      visible: false,
    }))

    pendingLines.value.push(...optimizedLines)
    throttledUpdate()
  }

  // 라인 클리어
  const clearLines = (): void => {
    lines.value = []
    pendingLines.value = []

    // 메트릭스 리셋
    Object.assign(metrics, {
      renderCount: 0,
      avgRenderTime: 0,
      memoryUsage: 0,
      scrollEvents: 0,
      lastUpdateTime: 0,
      batchedUpdates: 0,
    })
  }

  // 라인 검색 (최적화된)
  const searchLines = (query: string, caseSensitive = false): OptimizedTerminalLine[] => {
    if (!query.trim()) return lines.value

    const searchTerm = caseSensitive ? query : query.toLowerCase()

    return lines.value.filter(line => {
      const content = caseSensitive ? line.content : line.content.toLowerCase()
      return content.includes(searchTerm)
    })
  }

  // 메모리 사용량 최적화
  const optimizeMemoryUsage = (): void => {
    // 오래된 라인 제거
    const now = Date.now()
    const maxAge = 30 * 60 * 1000 // 30분

    lines.value = lines.value.filter(line => {
      return now - line.timestamp < maxAge
    })

    // 렌더링된 콘텐츠 정리
    cleanupRenderedContent()

    // 가비지 컬렉션 힌트
    if (typeof window !== 'undefined' && 'gc' in window) {
      // @ts-ignore - 개발자 도구에서만 사용 가능
      window.gc?.()
    }
  }

  // 성능 보고서 생성
  const generatePerformanceReport = () => {
    return {
      ...metrics,
      linesCount: lines.value.length,
      pendingLinesCount: pendingLines.value.length,
      estimatedMemoryUsage: estimatedMemoryUsage.value,
      isMemoryPressure: isMemoryPressure.value,
      visibleRange: visibleRange.value,
      virtualizationEnabled: enableVirtualization,
      timestamp: Date.now(),
    }
  }

  // 자동 최적화 인터벌 설정
  let optimizationInterval: NodeJS.Timeout | null = null

  const startAutoOptimization = () => {
    if (optimizationInterval) return

    optimizationInterval = setInterval(() => {
      if (isMemoryPressure.value) {
        optimizeMemoryUsage()
      }
    }, 30000) // 30초마다 실행
  }

  const stopAutoOptimization = () => {
    if (optimizationInterval) {
      clearInterval(optimizationInterval)
      optimizationInterval = null
    }
  }

  // 컴포넌트 언마운트 시 정리
  onUnmounted(() => {
    stopAutoOptimization()
  })

  // 자동 최적화 시작
  startAutoOptimization()

  return {
    // 상태
    lines: computed(() => lines.value),
    visibleLines,
    isProcessing,
    metrics: computed(() => metrics),

    // 계산된 속성
    estimatedMemoryUsage,
    isMemoryPressure,

    // 메서드
    addLine,
    addLines,
    clearLines,
    searchLines,
    updateVisibleRange,
    handleScroll,
    optimizeMemoryUsage,
    generatePerformanceReport,

    // 최적화 제어
    startAutoOptimization,
    stopAutoOptimization,
  }
}