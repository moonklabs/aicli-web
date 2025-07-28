/**
 * 성능 모니터링 API 서비스
 */

import { apiClient } from './api'

export interface PerformanceMetrics {
  timestamp: number
  requestCount: number
  errorCount: number
  requestDuration: number
  activeRequests: number
  responseCodeCounts: Record<string, number>
  endpointMetrics: Record<string, EndpointMetric>
  cpuUsage: number
  memoryUsage: number
  goroutineCount: number
  heapSize: number
  gcCount: number
  uptime: number
}

export interface EndpointMetric {
  method: string
  path: string
  count: number
  errorCount: number
  totalDuration: number
  averageDuration: number
  minDuration: number
  maxDuration: number
  lastAccess: string
  statusCodes: Record<number, number>
}

export interface PerformanceScore {
  score: number
  grade: string
  timestamp: number
}

export interface ErrorInfo {
  id: string
  level: 'debug' | 'info' | 'warn' | 'error' | 'fatal'
  message: string
  component?: string
  action?: string
  url?: string
  timestamp: number
  stackTrace?: string
  userAgent?: string
  userId?: string
  sessionId?: string
  metadata?: Record<string, any>
}

export interface ErrorSummary {
  totalErrors: number
  errorsToday: number
  errorsByLevel: Record<string, number>
  errorsByComponent: Record<string, number>
  recentErrors: ErrorInfo[]
  topErrors: ErrorInfo[]
}

export interface AlertInfo {
  id: string
  level: 'info' | 'warning' | 'error' | 'critical'
  title: string
  message: string
  source: string
  timestamp: number
  resolved: boolean
  resolvedAt?: number
}

export interface AlertingMetrics {
  totalAlerts: number
  activeAlerts: number
  alertsByLevel: Record<string, number>
  recentAlerts: AlertInfo[]
}

export interface LoadTestConfig {
  concurrent: number
  duration: number
  rampUp: number
  endpoint: string
  method: string
  payload?: any
}

export interface LoadTestReport {
  config: LoadTestConfig
  totalRequests: number
  successfulRequests: number
  failedRequests: number
  averageResponseTime: number
  minResponseTime: number
  maxResponseTime: number
  requestsPerSecond: number
  percentiles: Record<string, number>
  errors: Array<{ message: string; count: number }>
  startTime: number
  endTime: number
}

export interface TestResult {
  id: string
  type: 'load' | 'stress' | 'performance'
  status: 'running' | 'completed' | 'failed'
  startTime: number
  endTime?: number
  duration?: number
  results?: any
  error?: string
}

/**
 * 성능 모니터링 API 클라이언트
 */
export class PerformanceAPIService {
  private baseUrl = '/api/monitoring'

  /**
   * 현재 성능 메트릭 조회
   */
  async getPerformanceMetrics(): Promise<PerformanceMetrics> {
    const response = await apiClient.get(`${this.baseUrl}/performance`)
    return response.data
  }

  /**
   * 성능 점수 조회
   */
  async getPerformanceScore(): Promise<PerformanceScore> {
    const response = await apiClient.get(`${this.baseUrl}/performance/score`)
    return response.data
  }

  /**
   * 상위 엔드포인트 조회
   */
  async getTopEndpoints(limit: number = 10): Promise<EndpointMetric[]> {
    const response = await apiClient.get(`${this.baseUrl}/endpoints/top`, {
      params: { limit }
    })
    return response.data
  }

  /**
   * 느린 엔드포인트 조회
   */
  async getSlowEndpoints(limit: number = 10): Promise<EndpointMetric[]> {
    const response = await apiClient.get(`${this.baseUrl}/endpoints/slow`, {
      params: { limit }
    })
    return response.data
  }

  /**
   * 성능 메트릭 초기화
   */
  async resetMetrics(): Promise<void> {
    await apiClient.post(`${this.baseUrl}/performance/reset`)
  }

  /**
   * 에러 목록 조회
   */
  async getErrors(): Promise<ErrorInfo[]> {
    const response = await apiClient.get(`${this.baseUrl}/errors`)
    return response.data
  }

  /**
   * 특정 에러 조회
   */
  async getError(id: string): Promise<ErrorInfo> {
    const response = await apiClient.get(`${this.baseUrl}/errors/${id}`)
    return response.data
  }

  /**
   * 상위 에러 조회
   */
  async getTopErrors(limit: number = 10): Promise<ErrorInfo[]> {
    const response = await apiClient.get(`${this.baseUrl}/errors/top`, {
      params: { limit }
    })
    return response.data
  }

  /**
   * 에러 요약 조회
   */
  async getErrorSummary(): Promise<ErrorSummary> {
    const response = await apiClient.get(`${this.baseUrl}/errors/summary`)
    return response.data
  }

  /**
   * 에러 해결 처리
   */
  async resolveError(id: string): Promise<void> {
    await apiClient.post(`${this.baseUrl}/errors/${id}/resolve`)
  }

  /**
   * 에러 무시 처리
   */
  async ignoreError(id: string): Promise<void> {
    await apiClient.post(`${this.baseUrl}/errors/${id}/ignore`)
  }

  /**
   * 모든 에러 삭제
   */
  async clearErrors(): Promise<void> {
    await apiClient.post(`${this.baseUrl}/errors/clear`)
  }

  /**
   * 알림 목록 조회
   */
  async getAlerts(activeOnly: boolean = false): Promise<AlertInfo[]> {
    const response = await apiClient.get(`${this.baseUrl}/alerts`, {
      params: { active: activeOnly }
    })
    return response.data
  }

  /**
   * 알림 시스템 메트릭 조회
   */
  async getAlertingMetrics(): Promise<AlertingMetrics> {
    const response = await apiClient.get(`${this.baseUrl}/alerts/metrics`)
    return response.data
  }

  /**
   * 알림 음소거
   */
  async silenceAlert(id: string): Promise<void> {
    await apiClient.post(`${this.baseUrl}/alerts/${id}/silence`)
  }

  /**
   * 알림 해결
   */
  async resolveAlert(id: string): Promise<void> {
    await apiClient.post(`${this.baseUrl}/alerts/${id}/resolve`)
  }

  /**
   * 부하 테스트 실행
   */
  async runLoadTest(config?: LoadTestConfig): Promise<LoadTestReport> {
    const response = await apiClient.post(`${this.baseUrl}/test/load`, config)
    return response.data
  }

  /**
   * 테스트 결과 조회
   */
  async getTestResults(): Promise<TestResult[]> {
    const response = await apiClient.get(`${this.baseUrl}/test/results`)
    return response.data
  }

  /**
   * 테스트 결과 삭제
   */
  async clearTestResults(): Promise<void> {
    await apiClient.post(`${this.baseUrl}/test/results/clear`)
  }

  /**
   * 실행 중인 테스트 중지
   */
  async stopTest(): Promise<void> {
    await apiClient.post(`${this.baseUrl}/test/stop`)
  }

  /**
   * 실시간 메트릭 스트리밍 (WebSocket)
   */
  subscribeToMetrics(callback: (metrics: PerformanceMetrics) => void): () => void {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${window.location.host}/api/monitoring/ws/metrics`
    
    const ws = new WebSocket(wsUrl)
    
    ws.onmessage = (event) => {
      try {
        const metrics = JSON.parse(event.data)
        callback(metrics)
      } catch (error) {
        console.error('메트릭 파싱 오류:', error)
      }
    }
    
    ws.onerror = (error) => {
      console.error('WebSocket 연결 오류:', error)
    }
    
    ws.onclose = () => {
      console.log('WebSocket 연결 종료')
    }
    
    // 연결 해제 함수 반환
    return () => {
      ws.close()
    }
  }

  /**
   * 실시간 에러 스트리밍 (WebSocket)
   */
  subscribeToErrors(callback: (error: ErrorInfo) => void): () => void {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${window.location.host}/api/monitoring/ws/errors`
    
    const ws = new WebSocket(wsUrl)
    
    ws.onmessage = (event) => {
      try {
        const error = JSON.parse(event.data)
        callback(error)
      } catch (parseError) {
        console.error('에러 파싱 오류:', parseError)
      }
    }
    
    ws.onerror = (error) => {
      console.error('WebSocket 연결 오류:', error)
    }
    
    ws.onclose = () => {
      console.log('WebSocket 연결 종료')
    }
    
    return () => {
      ws.close()
    }
  }
}

/**
 * 성능 데이터 분석 유틸리티
 */
export class PerformanceAnalyzer {
  /**
   * 성능 점수 계산
   */
  static calculatePerformanceScore(metrics: PerformanceMetrics): number {
    let score = 100

    // 에러율 감점 (최대 40점)
    if (metrics.requestCount > 0) {
      const errorRate = metrics.errorCount / metrics.requestCount
      score -= errorRate * 40
    }

    // 평균 응답 시간 감점 (최대 30점)
    const avgDuration = metrics.requestDuration / Math.max(metrics.requestCount, 1)
    if (avgDuration > 1000) {
      score -= 30
    } else if (avgDuration > 500) {
      score -= 15
    } else if (avgDuration > 200) {
      score -= 5
    }

    // 메모리 사용률 감점 (최대 20점)
    const memoryMB = metrics.memoryUsage / (1024 * 1024)
    if (memoryMB > 100) {
      score -= 20
    } else if (memoryMB > 50) {
      score -= 10
    }

    // 고루틴 수 감점 (최대 10점)
    if (metrics.goroutineCount > 1000) {
      score -= 10
    } else if (metrics.goroutineCount > 500) {
      score -= 5
    }

    return Math.max(0, Math.min(100, score))
  }

  /**
   * 성능 등급 결정
   */
  static getPerformanceGrade(score: number): string {
    if (score >= 90) return 'A'
    if (score >= 80) return 'B'
    if (score >= 70) return 'C'
    if (score >= 60) return 'D'
    return 'F'
  }

  /**
   * 성능 트렌드 분석
   */
  static analyzePerformanceTrend(scores: number[]): 'improving' | 'declining' | 'stable' {
    if (scores.length < 2) return 'stable'
    
    const recent = scores.slice(-5) // 최근 5개 데이터 포인트
    if (recent.length < 2) return 'stable'
    
    const firstHalf = recent.slice(0, Math.floor(recent.length / 2))
    const secondHalf = recent.slice(Math.floor(recent.length / 2))
    
    const firstAvg = firstHalf.reduce((sum, score) => sum + score, 0) / firstHalf.length
    const secondAvg = secondHalf.reduce((sum, score) => sum + score, 0) / secondHalf.length
    
    const threshold = 2 // 2점 이상 차이가 나야 트렌드로 인정
    
    if (secondAvg - firstAvg > threshold) return 'improving'
    if (firstAvg - secondAvg > threshold) return 'declining'
    return 'stable'
  }

  /**
   * 메트릭 비교
   */
  static compareMetrics(current: PerformanceMetrics, previous: PerformanceMetrics): {
    requestCount: { value: number, change: number, percentage: number }
    errorRate: { value: number, change: number, percentage: number }
    avgResponseTime: { value: number, change: number, percentage: number }
    memoryUsage: { value: number, change: number, percentage: number }
  } {
    const currentErrorRate = current.requestCount > 0 ? current.errorCount / current.requestCount : 0
    const previousErrorRate = previous.requestCount > 0 ? previous.errorCount / previous.requestCount : 0
    
    const currentAvgResponseTime = current.requestDuration / Math.max(current.requestCount, 1)
    const previousAvgResponseTime = previous.requestDuration / Math.max(previous.requestCount, 1)

    return {
      requestCount: {
        value: current.requestCount,
        change: current.requestCount - previous.requestCount,
        percentage: previous.requestCount > 0 ? ((current.requestCount - previous.requestCount) / previous.requestCount) * 100 : 0
      },
      errorRate: {
        value: currentErrorRate,
        change: currentErrorRate - previousErrorRate,
        percentage: previousErrorRate > 0 ? ((currentErrorRate - previousErrorRate) / previousErrorRate) * 100 : 0
      },
      avgResponseTime: {
        value: currentAvgResponseTime,
        change: currentAvgResponseTime - previousAvgResponseTime,
        percentage: previousAvgResponseTime > 0 ? ((currentAvgResponseTime - previousAvgResponseTime) / previousAvgResponseTime) * 100 : 0
      },
      memoryUsage: {
        value: current.memoryUsage,
        change: current.memoryUsage - previous.memoryUsage,
        percentage: previous.memoryUsage > 0 ? ((current.memoryUsage - previous.memoryUsage) / previous.memoryUsage) * 100 : 0
      }
    }
  }

  /**
   * 임계값 경고 확인
   */
  static checkThresholds(metrics: PerformanceMetrics): Array<{
    type: 'warning' | 'critical'
    message: string
    value: number
    threshold: number
  }> {
    const warnings = []

    // 에러율 확인
    const errorRate = metrics.requestCount > 0 ? (metrics.errorCount / metrics.requestCount) * 100 : 0
    if (errorRate > 10) {
      warnings.push({
        type: 'critical' as const,
        message: '에러율이 높습니다',
        value: errorRate,
        threshold: 10
      })
    } else if (errorRate > 5) {
      warnings.push({
        type: 'warning' as const,
        message: '에러율이 증가했습니다',
        value: errorRate,
        threshold: 5
      })
    }

    // 평균 응답 시간 확인
    const avgResponseTime = metrics.requestDuration / Math.max(metrics.requestCount, 1)
    if (avgResponseTime > 2000) {
      warnings.push({
        type: 'critical' as const,
        message: '응답 시간이 매우 느립니다',
        value: avgResponseTime,
        threshold: 2000
      })
    } else if (avgResponseTime > 1000) {
      warnings.push({
        type: 'warning' as const,
        message: '응답 시간이 느립니다',
        value: avgResponseTime,
        threshold: 1000
      })
    }

    // 메모리 사용량 확인
    const memoryMB = metrics.memoryUsage / (1024 * 1024)
    if (memoryMB > 500) {
      warnings.push({
        type: 'critical' as const,
        message: '메모리 사용량이 매우 높습니다',
        value: memoryMB,
        threshold: 500
      })
    } else if (memoryMB > 200) {
      warnings.push({
        type: 'warning' as const,
        message: '메모리 사용량이 높습니다',
        value: memoryMB,
        threshold: 200
      })
    }

    return warnings
  }

  /**
   * 성능 권장사항 생성
   */
  static generateRecommendations(metrics: PerformanceMetrics): string[] {
    const recommendations = []
    
    const errorRate = metrics.requestCount > 0 ? (metrics.errorCount / metrics.requestCount) * 100 : 0
    const avgResponseTime = metrics.requestDuration / Math.max(metrics.requestCount, 1)
    const memoryMB = metrics.memoryUsage / (1024 * 1024)

    if (errorRate > 5) {
      recommendations.push('에러 로그를 분석하여 주요 오류 원인을 파악하세요')
      recommendations.push('예외 처리를 강화하고 에러 복구 메커니즘을 구현하세요')
    }

    if (avgResponseTime > 1000) {
      recommendations.push('데이터베이스 쿼리를 최적화하세요')
      recommendations.push('캐싱 전략을 검토하고 개선하세요')
      recommendations.push('비동기 처리를 도입하여 응답성을 향상시키세요')
    }

    if (memoryMB > 200) {
      recommendations.push('메모리 누수를 점검하세요')
      recommendations.push('불필요한 객체 참조를 제거하세요')
      recommendations.push('가비지 컬렉션 설정을 조정하세요')
    }

    if (metrics.goroutineCount > 500) {
      recommendations.push('고루틴 풀을 사용하여 동시성을 제한하세요')
      recommendations.push('고루틴 누수를 점검하세요')
    }

    if (recommendations.length === 0) {
      recommendations.push('현재 성능 상태가 양호합니다')
      recommendations.push('정기적인 모니터링을 통해 성능을 유지하세요')
    }

    return recommendations
  }
}

// 싱글톤 인스턴스 내보내기
export const performanceAPI = new PerformanceAPIService()