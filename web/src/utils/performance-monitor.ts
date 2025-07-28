import { type Metric, onCLS, onFCP, onLCP, onTTFB } from 'web-vitals'

/**
 * 성능 메트릭 데이터 인터페이스
 */
export interface PerformanceMetrics {
  // Core Web Vitals
  LCP?: number  // Largest Contentful Paint
  FID?: number  // First Input Delay
  CLS?: number  // Cumulative Layout Shift
  FCP?: number  // First Contentful Paint
  TTFB?: number // Time to First Byte

  // 추가 메트릭
  domContentLoaded?: number
  loadComplete?: number
  memoryUsage?: number
  connectionType?: string

  // 메타데이터
  timestamp: number
  url: string
  userAgent: string
  deviceType: 'desktop' | 'mobile' | 'tablet'
}

/**
 * 성능 이벤트 리스너 인터페이스
 */
export interface PerformanceListener {
  onMetric: (metric: PerformanceMetrics) => void
  onError: (error: Error) => void
}

/**
 * 성능 모니터링 클래스
 */
export class PerformanceMonitor {
  private metrics: Partial<PerformanceMetrics> = {}
  private listeners: PerformanceListener[] = []
  private isStarted = false

  /**
   * 성능 모니터링 시작
   */
  start(): void {
    if (this.isStarted) return

    this.isStarted = true
    this.initializeMetrics()
    this.collectWebVitals()
    this.collectCustomMetrics()
  }

  /**
   * 리스너 추가
   */
  addListener(listener: PerformanceListener): void {
    this.listeners.push(listener)
  }

  /**
   * 리스너 제거
   */
  removeListener(listener: PerformanceListener): void {
    const index = this.listeners.indexOf(listener)
    if (index > -1) {
      this.listeners.splice(index, 1)
    }
  }

  /**
   * 현재 메트릭 반환
   */
  getMetrics(): Partial<PerformanceMetrics> {
    return { ...this.metrics }
  }

  /**
   * 메트릭 리셋
   */
  reset(): void {
    this.metrics = {}
    this.initializeMetrics()
  }

  /**
   * 기본 메트릭 초기화
   */
  private initializeMetrics(): void {
    this.metrics = {
      timestamp: Date.now(),
      url: window.location.href,
      userAgent: navigator.userAgent,
      deviceType: this.getDeviceType(),
      connectionType: this.getConnectionType(),
    }
  }

  /**
   * Core Web Vitals 수집
   */
  private collectWebVitals(): void {
    const handleMetric = (metric: Metric) => {
      this.metrics[metric.name as keyof PerformanceMetrics] = metric.value
      this.notifyListeners()
    }

    try {
      onCLS(handleMetric)
      onFCP(handleMetric)
      onLCP(handleMetric)
      onTTFB(handleMetric)
    } catch (error) {
      this.notifyError(error as Error)
    }
  }

  /**
   * 사용자 정의 메트릭 수집
   */
  private collectCustomMetrics(): void {
    // DOM Content Loaded 시간
    if (document.readyState === 'loading') {
      document.addEventListener('DOMContentLoaded', () => {
        this.metrics.domContentLoaded = performance.now()
        this.notifyListeners()
      })
    } else {
      this.metrics.domContentLoaded = performance.now()
    }

    // 페이지 로드 완료 시간
    if (document.readyState !== 'complete') {
      window.addEventListener('load', () => {
        this.metrics.loadComplete = performance.now()
        this.collectMemoryUsage()
        this.notifyListeners()
      })
    } else {
      this.metrics.loadComplete = performance.now()
      this.collectMemoryUsage()
    }

    // 주기적으로 메모리 사용량 업데이트
    setInterval(() => {
      this.collectMemoryUsage()
      this.notifyListeners()
    }, 30000) // 30초마다
  }

  /**
   * 메모리 사용량 수집
   */
  private collectMemoryUsage(): void {
    if ('memory' in performance) {
      const memory = (performance as any).memory
      this.metrics.memoryUsage = memory.usedJSHeapSize / (1024 * 1024) // MB 단위
    }
  }

  /**
   * 디바이스 타입 감지
   */
  private getDeviceType(): 'desktop' | 'mobile' | 'tablet' {
    const userAgent = navigator.userAgent.toLowerCase()

    if (/tablet|ipad|playbook|silk/.test(userAgent)) {
      return 'tablet'
    }

    if (/mobile|iphone|ipod|android|blackberry|opera|mini|windows\sce|palm|smartphone|iemobile/.test(userAgent)) {
      return 'mobile'
    }

    return 'desktop'
  }

  /**
   * 연결 타입 감지
   */
  private getConnectionType(): string {
    if ('connection' in navigator) {
      const connection = (navigator as any).connection
      return connection.effectiveType || connection.type || 'unknown'
    }
    return 'unknown'
  }

  /**
   * 리스너들에게 메트릭 업데이트 알림
   */
  private notifyListeners(): void {
    const completeMetrics = this.metrics as PerformanceMetrics
    this.listeners.forEach(listener => {
      try {
        listener.onMetric(completeMetrics)
      } catch (error) {
        console.error('Performance listener error:', error)
      }
    })
  }

  /**
   * 리스너들에게 오류 알림
   */
  private notifyError(error: Error): void {
    this.listeners.forEach(listener => {
      try {
        listener.onError(error)
      } catch (listenerError) {
        console.error('Performance listener error handler failed:', listenerError)
      }
    })
  }
}

/**
 * 전역 성능 모니터 인스턴스
 */
export const performanceMonitor = new PerformanceMonitor()

/**
 * 성능 데이터를 로컬 저장소에 저장
 */
export function savePerformanceData(metrics: PerformanceMetrics): void {
  try {
    const existingData = localStorage.getItem('performance_metrics')
    const data = existingData ? JSON.parse(existingData) : []

    data.push(metrics)

    // 최근 100개 항목만 유지
    if (data.length > 100) {
      data.splice(0, data.length - 100)
    }

    localStorage.setItem('performance_metrics', JSON.stringify(data))
  } catch (error) {
    console.error('Failed to save performance data:', error)
  }
}

/**
 * 저장된 성능 데이터 로드
 */
export function loadPerformanceData(): PerformanceMetrics[] {
  try {
    const data = localStorage.getItem('performance_metrics')
    return data ? JSON.parse(data) : []
  } catch (error) {
    console.error('Failed to load performance data:', error)
    return []
  }
}

/**
 * 성능 점수 계산
 */
export function calculatePerformanceScore(metrics: PerformanceMetrics): number {
  let score = 100

  // LCP 점수 (0-40점)
  if (metrics.LCP) {
    if (metrics.LCP > 4000) score -= 40
    else if (metrics.LCP > 2500) score -= 20
    else if (metrics.LCP > 1000) score -= 10
  }

  // FID 점수 (0-30점)
  if (metrics.FID) {
    if (metrics.FID > 300) score -= 30
    else if (metrics.FID > 100) score -= 15
    else if (metrics.FID > 50) score -= 5
  }

  // CLS 점수 (0-30점)
  if (metrics.CLS) {
    if (metrics.CLS > 0.25) score -= 30
    else if (metrics.CLS > 0.1) score -= 15
    else if (metrics.CLS > 0.05) score -= 5
  }

  return Math.max(0, score)
}