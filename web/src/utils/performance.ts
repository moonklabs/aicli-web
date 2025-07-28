/**
 * 성능 모니터링 유틸리티
 * Core Web Vitals 측정 및 성능 지표 추적
 */

import { type Metric, onCLS, onFCP, onLCP, onTTFB } from 'web-vitals'

export interface PerformanceMetrics {
  // Core Web Vitals
  lcp?: number // Largest Contentful Paint
  cls?: number // Cumulative Layout Shift

  // Additional Metrics
  fcp?: number // First Contentful Paint
  ttfb?: number // Time to First Byte

  // Custom Metrics
  pageLoadTime?: number
  resourceLoadTime?: number
  navigationTiming?: PerformanceNavigationTiming
}

export interface PerformanceReport {
  url: string
  timestamp: number
  userAgent: string
  metrics: PerformanceMetrics
  scores: {
    lcp: 'good' | 'needs-improvement' | 'poor'
    cls: 'good' | 'needs-improvement' | 'poor'
    overall: 'good' | 'needs-improvement' | 'poor'
  }
}

class PerformanceMonitor {
  private metrics: PerformanceMetrics = {}
  private callbacks: Array<(report: PerformanceReport) => void> = []
  private isInitialized = false

  /**
   * 성능 모니터링 초기화
   */
  init() {
    if (this.isInitialized) return

    this.isInitialized = true

    // Core Web Vitals 측정
    onLCP((metric: Metric) => {
      this.metrics.lcp = metric.value
      this.checkAndReport()
    })

    onCLS((metric: Metric) => {
      this.metrics.cls = metric.value
      this.checkAndReport()
    })

    onFCP((metric: Metric) => {
      this.metrics.fcp = metric.value
      this.checkAndReport()
    })

    onTTFB((metric: Metric) => {
      this.metrics.ttfb = metric.value
      this.checkAndReport()
    })

    // 페이지 로드 완료 시 추가 메트릭 수집
    this.collectAdditionalMetrics()
  }

  /**
   * 성능 리포트 콜백 등록
   */
  onReport(callback: (report: PerformanceReport) => void) {
    this.callbacks.push(callback)
  }

  /**
   * 현재 메트릭 가져오기
   */
  getMetrics(): PerformanceMetrics {
    return { ...this.metrics }
  }

  /**
   * 성능 점수 계산
   */
  private calculateScores(): PerformanceReport['scores'] {
    const lcpScore = this.getScore(this.metrics.lcp, [2500, 4000])
    const clsScore = this.getScore(this.metrics.cls, [0.1, 0.25])

    // 전체 점수는 가장 낮은 점수로 결정
    const allScores = [lcpScore, clsScore]
    const overallScore = allScores.includes('poor')
      ? 'poor'
      : allScores.includes('needs-improvement')
        ? 'needs-improvement'
        : 'good'

    return {
      lcp: lcpScore,
      cls: clsScore,
      overall: overallScore,
    }
  }

  /**
   * 개별 메트릭 점수 계산
   */
  private getScore(
    value: number | undefined,
    thresholds: [number, number],
  ): 'good' | 'needs-improvement' | 'poor' {
    if (value === undefined) return 'poor'

    const [good, needsImprovement] = thresholds

    if (value <= good) return 'good'
    if (value <= needsImprovement) return 'needs-improvement'
    return 'poor'
  }

  /**
   * 추가 메트릭 수집
   */
  private collectAdditionalMetrics() {
    // 페이지 로드 완료 후 실행
    if (document.readyState === 'complete') {
      this.collectTimingMetrics()
    } else {
      window.addEventListener('load', () => {
        this.collectTimingMetrics()
      })
    }
  }

  /**
   * 타이밍 메트릭 수집
   */
  private collectTimingMetrics() {
    const navigation = performance.getEntriesByType('navigation')[0] as PerformanceNavigationTiming

    if (navigation) {
      this.metrics.navigationTiming = navigation
      this.metrics.pageLoadTime = navigation.loadEventEnd - navigation.navigationStart

      // 리소스 로딩 시간 계산
      const resources = performance.getEntriesByType('resource')
      const totalResourceTime = resources.reduce((total, resource) => {
        return total + (resource.responseEnd - resource.requestStart)
      }, 0)
      this.metrics.resourceLoadTime = totalResourceTime
    }

    this.checkAndReport()
  }

  /**
   * 리포트 생성 및 전송
   */
  private checkAndReport() {
    // 주요 메트릭이 모두 수집되었는지 확인
    if (!this.metrics.lcp || !this.metrics.fcp) return

    const report: PerformanceReport = {
      url: window.location.href,
      timestamp: Date.now(),
      userAgent: navigator.userAgent,
      metrics: this.getMetrics(),
      scores: this.calculateScores(),
    }

    // 콜백 실행
    this.callbacks.forEach(callback => {
      try {
        callback(report)
      } catch (error) {
        console.error('Performance report callback error:', error)
      }
    })
  }

  /**
   * 메모리 사용량 모니터링
   */
  getMemoryInfo() {
    if ('memory' in performance) {
      const memory = (performance as any).memory
      return {
        usedJSHeapSize: memory.usedJSHeapSize,
        totalJSHeapSize: memory.totalJSHeapSize,
        jsHeapSizeLimit: memory.jsHeapSizeLimit,
        usagePercentage: (memory.usedJSHeapSize / memory.jsHeapSizeLimit) * 100,
      }
    }
    return null
  }

  /**
   * 리소스 로딩 성능 분석
   */
  getResourceLoadingAnalysis() {
    const resources = performance.getEntriesByType('resource') as PerformanceResourceTiming[]

    const analysis = {
      totalResources: resources.length,
      slowestResources: resources
        .map(resource => ({
          name: resource.name,
          duration: resource.responseEnd - resource.requestStart,
          size: resource.transferSize || 0,
          type: this.getResourceType(resource.name),
        }))
        .sort((a, b) => b.duration - a.duration)
        .slice(0, 10),

      resourcesByType: resources.reduce((acc, resource) => {
        const type = this.getResourceType(resource.name)
        if (!acc[type]) {
          acc[type] = { count: 0, totalSize: 0, totalDuration: 0 }
        }
        acc[type].count++
        acc[type].totalSize += resource.transferSize || 0
        acc[type].totalDuration += resource.responseEnd - resource.requestStart
        return acc
      }, {} as Record<string, { count: number; totalSize: number; totalDuration: number }>),
    }

    return analysis
  }

  /**
   * 리소스 타입 분류
   */
  private getResourceType(url: string): string {
    if (url.match(/\.(js|ts)(\?|$)/)) return 'script'
    if (url.match(/\.(css|scss|sass)(\?|$)/)) return 'stylesheet'
    if (url.match(/\.(png|jpg|jpeg|gif|webp|svg)(\?|$)/)) return 'image'
    if (url.match(/\.(woff|woff2|ttf|eot)(\?|$)/)) return 'font'
    if (url.includes('/api/')) return 'api'
    return 'other'
  }
}

// 싱글톤 인스턴스
export const performanceMonitor = new PerformanceMonitor()

// 개발 환경에서 성능 로깅
if (import.meta.env.DEV) {
  performanceMonitor.onReport((report) => {
    console.group('🚀 Performance Report')
    console.log('📊 Core Web Vitals:', {
      LCP: `${report.metrics.lcp?.toFixed(1)}ms (${report.scores.lcp})`,
      CLS: `${report.metrics.cls?.toFixed(3)} (${report.scores.cls})`,
    })
    console.log('📈 Additional Metrics:', {
      FCP: `${report.metrics.fcp?.toFixed(1)}ms`,
      TTFB: `${report.metrics.ttfb?.toFixed(1)}ms`,
      'Page Load': `${report.metrics.pageLoadTime?.toFixed(1)}ms`,
    })
    console.log('🎯 Overall Score:', report.scores.overall)
    console.groupEnd()
  })
}

// Vue 플러그인으로 전역 등록
export function createPerformancePlugin() {
  return {
    install(app: any) {
      app.config.globalProperties.$performance = performanceMonitor
      app.provide('performance', performanceMonitor)

      // 앱 시작 시 초기화
      performanceMonitor.init()
    },
  }
}