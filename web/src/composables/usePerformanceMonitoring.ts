import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import {
  type PerformanceListener,
  type PerformanceMetrics,
  calculatePerformanceScore,
  loadPerformanceData,
  performanceMonitor,
  savePerformanceData,
} from '@/utils/performance-monitor'

/**
 * 성능 모니터링 상태 인터페이스
 */
interface PerformanceState {
  isMonitoring: boolean
  currentMetrics: Partial<PerformanceMetrics>
  historicalData: PerformanceMetrics[]
  errors: Error[]
}

/**
 * 성능 모니터링을 위한 Composable
 */
export function usePerformanceMonitoring() {
  // 반응형 상태
  const state = reactive<PerformanceState>({
    isMonitoring: false,
    currentMetrics: {},
    historicalData: [],
    errors: [],
  })

  // 실시간 성능 점수
  const performanceScore = computed(() => {
    if (!state.currentMetrics.timestamp) return 0
    return calculatePerformanceScore(state.currentMetrics as PerformanceMetrics)
  })

  // 성능 등급
  const performanceGrade = computed(() => {
    const score = performanceScore.value
    if (score >= 90) return 'A'
    if (score >= 80) return 'B'
    if (score >= 70) return 'C'
    if (score >= 60) return 'D'
    return 'F'
  })

  // Core Web Vitals 상태
  const webVitalsStatus = computed(() => {
    const metrics = state.currentMetrics
    return {
      lcp: {
        value: metrics.LCP || 0,
        status: !metrics.LCP ? 'pending' :
                metrics.LCP <= 2500 ? 'good' :
                metrics.LCP <= 4000 ? 'needs-improvement' : 'poor',
      },
      fid: {
        value: metrics.FID || 0,
        status: !metrics.FID ? 'pending' :
                metrics.FID <= 100 ? 'good' :
                metrics.FID <= 300 ? 'needs-improvement' : 'poor',
      },
      cls: {
        value: metrics.CLS || 0,
        status: !metrics.CLS ? 'pending' :
                metrics.CLS <= 0.1 ? 'good' :
                metrics.CLS <= 0.25 ? 'needs-improvement' : 'poor',
      },
      fcp: {
        value: metrics.FCP || 0,
        status: !metrics.FCP ? 'pending' :
                metrics.FCP <= 1800 ? 'good' :
                metrics.FCP <= 3000 ? 'needs-improvement' : 'poor',
      },
      ttfb: {
        value: metrics.TTFB || 0,
        status: !metrics.TTFB ? 'pending' :
                metrics.TTFB <= 800 ? 'good' :
                metrics.TTFB <= 1800 ? 'needs-improvement' : 'poor',
      },
    }
  })

  // 성능 트렌드 분석
  const performanceTrend = computed(() => {
    if (state.historicalData.length < 2) return 'stable'

    const recent = state.historicalData.slice(-5)
    const scores = recent.map(calculatePerformanceScore)
    const trend = scores[scores.length - 1] - scores[0]

    if (trend > 5) return 'improving'
    if (trend < -5) return 'declining'
    return 'stable'
  })

  // 성능 리스너
  const listener: PerformanceListener = {
    onMetric: (metrics: PerformanceMetrics) => {
      state.currentMetrics = { ...metrics }

      // 완전한 메트릭일 때만 히스토리에 저장
      if (metrics.LCP && metrics.FID && metrics.CLS) {
        state.historicalData.push(metrics)
        savePerformanceData(metrics)

        // 최근 50개 항목만 유지
        if (state.historicalData.length > 50) {
          state.historicalData.splice(0, state.historicalData.length - 50)
        }
      }
    },
    onError: (error: Error) => {
      state.errors.push(error)
      console.error('Performance monitoring error:', error)
    },
  }

  /**
   * 성능 모니터링 시작
   */
  const startMonitoring = () => {
    if (state.isMonitoring) return

    state.isMonitoring = true
    performanceMonitor.addListener(listener)
    performanceMonitor.start()

    // 히스토리 데이터 로드
    state.historicalData = loadPerformanceData()
  }

  /**
   * 성능 모니터링 중지
   */
  const stopMonitoring = () => {
    if (!state.isMonitoring) return

    state.isMonitoring = false
    performanceMonitor.removeListener(listener)
  }

  /**
   * 메트릭 리셋
   */
  const resetMetrics = () => {
    state.currentMetrics = {}
    state.errors = []
    performanceMonitor.reset()
  }

  /**
   * 성능 리포트 생성
   */
  const generateReport = () => {
    const metrics = state.currentMetrics as PerformanceMetrics
    const score = performanceScore.value
    const grade = performanceGrade.value
    const vitals = webVitalsStatus.value

    return {
      timestamp: Date.now(),
      url: window.location.href,
      score,
      grade,
      metrics,
      webVitals: vitals,
      recommendations: generateRecommendations(vitals),
      deviceInfo: {
        userAgent: navigator.userAgent,
        deviceType: metrics.deviceType,
        connectionType: metrics.connectionType,
        memoryUsage: metrics.memoryUsage,
      },
    }
  }

  /**
   * 성능 개선 권장사항 생성
   */
  const generateRecommendations = (vitals: ReturnType<typeof webVitalsStatus>['value']) => {
    const recommendations: string[] = []

    if (vitals.lcp.status === 'poor') {
      recommendations.push('이미지 최적화 및 지연 로딩을 고려하세요')
      recommendations.push('서버 응답 시간을 개선하세요')
      recommendations.push('중요하지 않은 리소스의 로딩을 지연시키세요')
    }

    if (vitals.fid.status === 'poor') {
      recommendations.push('JavaScript 실행 시간을 줄이세요')
      recommendations.push('코드 스플리팅을 적용하세요')
      recommendations.push('긴 작업을 분할하세요')
    }

    if (vitals.cls.status === 'poor') {
      recommendations.push('이미지와 동영상에 크기 속성을 설정하세요')
      recommendations.push('동적 콘텐츠를 위한 공간을 미리 예약하세요')
      recommendations.push('웹폰트 로딩을 최적화하세요')
    }

    if (vitals.fcp.status === 'poor') {
      recommendations.push('중요 리소스를 우선 로딩하세요')
      recommendations.push('렌더링 차단 리소스를 제거하세요')
      recommendations.push('서버 사이드 렌더링을 고려하세요')
    }

    return recommendations
  }

  /**
   * 메트릭 내보내기
   */
  const exportMetrics = () => {
    const data = {
      current: state.currentMetrics,
      historical: state.historicalData,
      report: generateReport(),
    }

    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `performance-metrics-${Date.now()}.json`
    a.click()
    URL.revokeObjectURL(url)
  }

  // 컴포넌트 마운트 시 자동 시작
  onMounted(() => {
    startMonitoring()
  })

  // 컴포넌트 언마운트 시 정리
  onUnmounted(() => {
    stopMonitoring()
  })

  return {
    // 상태
    state: readonly(state),

    // 계산된 값
    performanceScore,
    performanceGrade,
    webVitalsStatus,
    performanceTrend,

    // 메서드
    startMonitoring,
    stopMonitoring,
    resetMetrics,
    generateReport,
    exportMetrics,

    // 유틸리티
    isMonitoring: computed(() => state.isMonitoring),
    hasErrors: computed(() => state.errors.length > 0),
    isReady: computed(() => Object.keys(state.currentMetrics).length > 0),
  }
}

/**
 * 간단한 성능 측정을 위한 헬퍼
 */
export function measurePerformance<T>(
  name: string,
  fn: () => T | Promise<T>,
): Promise<{ result: T; duration: number }> {
  return new Promise(async (resolve, reject) => {
    const start = performance.now()

    try {
      const result = await fn()
      const duration = performance.now() - start

      // 커스텀 마크 추가
      performance.mark(`${name}-start`)
      performance.mark(`${name}-end`)
      performance.measure(name, `${name}-start`, `${name}-end`)

      resolve({ result, duration })
    } catch (error) {
      reject(error)
    }
  })
}

/**
 * 컴포넌트별 성능 측정 데코레이터
 */
export function withPerformanceTracking<T extends (...args: any[]) => any>(
  componentName: string,
  method: T,
): T {
  return ((...args: any[]) => {
    return measurePerformance(`${componentName}-${method.name}`, () => method(...args))
  }) as T
}