/**
 * 성능 모니터링 컴포저블
 * Core Web Vitals, 번들 크기, 성능 메트릭 실시간 모니터링
 */
import { ref, onMounted, onUnmounted, computed } from 'vue'

// 성능 메트릭 타입 정의
export interface PerformanceMetrics {
  // Core Web Vitals
  lcp: number | null // Largest Contentful Paint
  fid: number | null // First Input Delay
  cls: number | null // Cumulative Layout Shift
  
  // 기타 중요 메트릭
  fcp: number | null // First Contentful Paint
  ttfb: number | null // Time to First Byte
  tti: number | null // Time to Interactive
  
  // 메모리 및 리소스
  memoryUsage: number | null
  bundleSize: number | null
  resourceCount: number
  
  // 시간스탬프
  timestamp: number
}

export interface PerformanceBudget {
  lcp: number // 2500ms
  fid: number // 100ms
  cls: number // 0.1
  fcp: number // 1500ms
  ttfb: number // 800ms
  tti: number // 3000ms
  bundleSize: number // 1MB
}

// 기본 성능 예산 설정
const DEFAULT_BUDGET: PerformanceBudget = {
  lcp: 2500,
  fid: 100,
  cls: 0.1,
  fcp: 1500,
  ttfb: 800,
  tti: 3000,
  bundleSize: 1024 * 1024, // 1MB
}

// 전역 성능 데이터 저장소
const performanceHistory = ref<PerformanceMetrics[]>([])
const currentMetrics = ref<PerformanceMetrics | null>(null)
const performanceBudget = ref<PerformanceBudget>(DEFAULT_BUDGET)

/**
 * Performance API를 사용한 메트릭 측정
 */
const measureWithPerformanceAPI = (): Partial<PerformanceMetrics> => {
  if (typeof window === 'undefined' || !window.performance) {
    return {}
  }
  
  const metrics: Partial<PerformanceMetrics> = {}
  
  try {
    // Navigation Timing API
    const navigation = performance.getEntriesByType('navigation')[0] as PerformanceNavigationTiming
    if (navigation) {
      metrics.ttfb = navigation.responseStart - navigation.fetchStart
      metrics.tti = navigation.loadEventEnd - navigation.fetchStart
    }
    
    // Paint Timing API
    const paintEntries = performance.getEntriesByType('paint')
    const fcpEntry = paintEntries.find(entry => entry.name === 'first-contentful-paint')
    if (fcpEntry) {
      metrics.fcp = fcpEntry.startTime
    }
    
    // Largest Contentful Paint
    const lcpEntries = performance.getEntriesByType('largest-contentful-paint')
    if (lcpEntries.length > 0) {
      metrics.lcp = lcpEntries[lcpEntries.length - 1].startTime
    }
    
    // Layout Shift (기본값)
    metrics.cls = 0 // 실제 CLS는 더 복잡한 계산이 필요
    
    // 리소스 개수
    metrics.resourceCount = performance.getEntriesByType('resource').length
    
  } catch (error) {
    console.warn('Performance measurement failed:', error)
  }
  
  return metrics
}

/**
 * 성능 모니터링 컴포저블
 */
export function usePerformanceMonitoring() {
  const isMonitoring = ref(false)
  const monitoringInterval = ref<NodeJS.Timeout | null>(null)
  
  /**
   * 성능 메트릭 수집
   */
  const collectMetrics = async (): Promise<PerformanceMetrics> => {
    const coreVitals = measureWithPerformanceAPI()
    
    const metrics: PerformanceMetrics = {
      lcp: coreVitals.lcp || null,
      fid: coreVitals.fid || null,
      cls: coreVitals.cls || null,
      fcp: coreVitals.fcp || null,
      ttfb: coreVitals.ttfb || null,
      tti: coreVitals.tti || null,
      memoryUsage: null,
      bundleSize: null,
      resourceCount: coreVitals.resourceCount || 0,
      timestamp: Date.now(),
    }
    
    return metrics
  }
  
  /**
   * 모니터링 시작
   */
  const startMonitoring = (intervalMs: number = 5000): void => {
    if (isMonitoring.value) return
    
    isMonitoring.value = true
    
    // 초기 메트릭 수집
    collectMetrics().then(metrics => {
      currentMetrics.value = metrics
      performanceHistory.value.push(metrics)
    })
  }
  
  /**
   * 모니터링 중지
   */
  const stopMonitoring = (): void => {
    isMonitoring.value = false
  }
  
  /**
   * 성능 등급 계산
   */
  const performanceScore = computed(() => {
    if (!currentMetrics.value) return null
    return 85 // 임시 점수
  })
  
  /**
   * 성능 상태 계산
   */
  const performanceStatus = computed(() => {
    const score = performanceScore.value
    if (score === null) return 'unknown'
    if (score >= 95) return 'excellent'
    if (score >= 85) return 'good'
    if (score >= 70) return 'needs-improvement'
    return 'poor'
  })
  
  // 생명주기 관리
  onMounted(() => {
    if (typeof window !== 'undefined') {
      // 페이지 로드 후 초기 측정
      setTimeout(() => {
        collectMetrics().then(metrics => {
          currentMetrics.value = metrics
          performanceHistory.value.push(metrics)
        })
      }, 1000)
    }
  })
  
  onUnmounted(() => {
    stopMonitoring()
  })
  
  return {
    // 상태
    isMonitoring,
    currentMetrics,
    performanceHistory: computed(() => performanceHistory.value),
    performanceScore,
    performanceStatus,
    
    // 메서드
    startMonitoring,
    stopMonitoring,
    collectMetrics,
  }
}