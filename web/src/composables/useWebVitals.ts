import { ref, onMounted, onUnmounted } from 'vue'
import type { Metric } from 'web-vitals'

export interface WebVitalsMetrics {
  // Core Web Vitals
  lcp?: number   // Largest Contentful Paint
  fid?: number   // First Input Delay
  cls?: number   // Cumulative Layout Shift
  
  // Other Web Vitals
  fcp?: number   // First Contentful Paint
  inp?: number   // Interaction to Next Paint
  ttfb?: number  // Time to First Byte
}

export interface WebVitalsThresholds {
  lcp: { good: number; poor: number }
  fid: { good: number; poor: number }
  cls: { good: number; poor: number }
  fcp: { good: number; poor: number }
  inp: { good: number; poor: number }
  ttfb: { good: number; poor: number }
}

// Google의 권장 임계값
const DEFAULT_THRESHOLDS: WebVitalsThresholds = {
  lcp: { good: 2500, poor: 4000 },
  fid: { good: 100, poor: 300 },
  cls: { good: 0.1, poor: 0.25 },
  fcp: { good: 1800, poor: 3000 },
  inp: { good: 200, poor: 500 },
  ttfb: { good: 800, poor: 1800 },
}

export function useWebVitals(reportCallback?: (metrics: WebVitalsMetrics) => void) {
  const metrics = ref<WebVitalsMetrics>({})
  const isSupported = ref(true)
  
  // 메트릭 평가
  const evaluateMetric = (
    value: number,
    thresholds: { good: number; poor: number }
  ): 'good' | 'needs-improvement' | 'poor' => {
    if (value <= thresholds.good) return 'good'
    if (value <= thresholds.poor) return 'needs-improvement'
    return 'poor'
  }
  
  // Core Web Vitals 점수 계산
  const getCoreWebVitalsScore = (): number => {
    const { lcp, fid, cls } = metrics.value
    if (!lcp || !fid || !cls) return 0
    
    let score = 0
    const lcpEval = evaluateMetric(lcp, DEFAULT_THRESHOLDS.lcp)
    const fidEval = evaluateMetric(fid, DEFAULT_THRESHOLDS.fid)
    const clsEval = evaluateMetric(cls, DEFAULT_THRESHOLDS.cls)
    
    // 각 메트릭별 점수 계산
    if (lcpEval === 'good') score += 33.33
    else if (lcpEval === 'needs-improvement') score += 16.67
    
    if (fidEval === 'good') score += 33.33
    else if (fidEval === 'needs-improvement') score += 16.67
    
    if (clsEval === 'good') score += 33.33
    else if (clsEval === 'needs-improvement') score += 16.67
    
    return Math.round(score)
  }
  
  // 메트릭 리포트 핸들러
  const handleMetricReport = (metric: Metric) => {
    const value = metric.value
    
    switch (metric.name) {
      case 'LCP':
        metrics.value.lcp = value
        break
      case 'FID':
        metrics.value.fid = value
        break
      case 'CLS':
        metrics.value.cls = value
        break
      case 'FCP':
        metrics.value.fcp = value
        break
      case 'INP':
        metrics.value.inp = value
        break
      case 'TTFB':
        metrics.value.ttfb = value
        break
    }
    
    // 콜백 실행
    reportCallback?.(metrics.value)
  }
  
  // Web Vitals 초기화
  const initWebVitals = async () => {
    try {
      const { onLCP, onFID, onCLS, onFCP, onINP, onTTFB } = await import('web-vitals')
      
      // Core Web Vitals
      onLCP(handleMetricReport)
      onFID(handleMetricReport)
      onCLS(handleMetricReport)
      
      // Other Web Vitals
      onFCP(handleMetricReport)
      onINP(handleMetricReport)
      onTTFB(handleMetricReport)
    } catch (error) {
      console.warn('Web Vitals not supported:', error)
      isSupported.value = false
    }
  }
  
  // 메트릭 리셋
  const reset = () => {
    metrics.value = {}
  }
  
  // 메트릭을 서버로 전송
  const sendToAnalytics = async (endpoint: string) => {
    if (Object.keys(metrics.value).length === 0) return
    
    try {
      await fetch(endpoint, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          metrics: metrics.value,
          score: getCoreWebVitalsScore(),
          timestamp: new Date().toISOString(),
          url: window.location.href,
          userAgent: navigator.userAgent,
        }),
      })
    } catch (error) {
      console.error('Failed to send metrics:', error)
    }
  }
  
  onMounted(() => {
    if (typeof window !== 'undefined') {
      initWebVitals()
    }
  })
  
  return {
    metrics,
    isSupported,
    getCoreWebVitalsScore,
    evaluateMetric,
    reset,
    sendToAnalytics,
    thresholds: DEFAULT_THRESHOLDS,
  }
}