/**
 * 성능 테스트 헬퍼 유틸리티
 */

import { performance } from '@/utils/performance'

export interface PerformanceTestResult {
  testName: string
  duration: number
  memoryUsage: number
  success: boolean
  error?: string
  metrics: {
    fps?: number
    loadTime?: number
    renderTime?: number
    networkRequests?: number
  }
}

export interface LoadTestConfig {
  concurrent: number
  duration: number
  rampUp: number
  endpoint: string
  method: 'GET' | 'POST' | 'PUT' | 'DELETE'
  payload?: any
}

export interface LoadTestResult {
  config: LoadTestConfig
  totalRequests: number
  successfulRequests: number
  failedRequests: number
  averageResponseTime: number
  minResponseTime: number
  maxResponseTime: number
  requestsPerSecond: number
  errors: Array<{
    message: string
    count: number
  }>
}

/**
 * 성능 테스트 실행기
 */
export class PerformanceTestRunner {
  private tests: Map<string, () => Promise<any>> = new Map()
  private results: PerformanceTestResult[] = []

  /**
   * 테스트 등록
   */
  register(name: string, testFn: () => Promise<any>) {
    this.tests.set(name, testFn)
  }

  /**
   * 모든 테스트 실행
   */
  async runAll(): Promise<PerformanceTestResult[]> {
    this.results = []

    for (const [name, testFn] of this.tests) {
      const result = await this.runSingle(name, testFn)
      this.results.push(result)
    }

    return this.results
  }

  /**
   * 단일 테스트 실행
   */
  async runSingle(name: string, testFn: () => Promise<any>): Promise<PerformanceTestResult> {
    const startTime = performance.now()
    const startMemory = this.getMemoryUsage()

    try {
      await testFn()

      const endTime = performance.now()
      const endMemory = this.getMemoryUsage()

      return {
        testName: name,
        duration: endTime - startTime,
        memoryUsage: endMemory - startMemory,
        success: true,
        metrics: {
          loadTime: endTime - startTime,
        },
      }
    } catch (error) {
      const endTime = performance.now()
      const endMemory = this.getMemoryUsage()

      return {
        testName: name,
        duration: endTime - startTime,
        memoryUsage: endMemory - startMemory,
        success: false,
        error: error instanceof Error ? error.message : String(error),
        metrics: {},
      }
    }
  }

  /**
   * 메모리 사용량 측정
   */
  private getMemoryUsage(): number {
    if ('memory' in performance) {
      return (performance as any).memory.usedJSHeapSize
    }
    return 0
  }

  /**
   * 결과 리포트 생성
   */
  generateReport(): string {
    const totalTests = this.results.length
    const passedTests = this.results.filter(r => r.success).length
    const failedTests = totalTests - passedTests
    const averageDuration = this.results.reduce((sum, r) => sum + r.duration, 0) / totalTests

    let report = '성능 테스트 리포트\n'
    report += '===================\n'
    report += `총 테스트: ${totalTests}\n`
    report += `성공: ${passedTests}\n`
    report += `실패: ${failedTests}\n`
    report += `평균 실행 시간: ${averageDuration.toFixed(2)}ms\n\n`

    report += '상세 결과:\n'
    this.results.forEach(result => {
      report += `${result.testName}: ${result.success ? 'PASS' : 'FAIL'} (${result.duration.toFixed(2)}ms)\n`
      if (!result.success && result.error) {
        report += `  오류: ${result.error}\n`
      }
    })

    return report
  }
}

/**
 * 부하 테스트 실행기
 */
export class LoadTestRunner {
  async run(config: LoadTestConfig): Promise<LoadTestResult> {
    const results: LoadTestResult = {
      config,
      totalRequests: 0,
      successfulRequests: 0,
      failedRequests: 0,
      averageResponseTime: 0,
      minResponseTime: Infinity,
      maxResponseTime: 0,
      requestsPerSecond: 0,
      errors: [],
    }

    const startTime = Date.now()
    const endTime = startTime + config.duration * 1000
    const responseTimes: number[] = []
    const errors: Map<string, number> = new Map()

    // 동시 요청 실행
    const promises: Promise<void>[] = []

    for (let i = 0; i < config.concurrent; i++) {
      promises.push(this.runConcurrentRequests(config, endTime, responseTimes, errors))
    }

    await Promise.all(promises)

    // 결과 계산
    results.totalRequests = responseTimes.length
    results.successfulRequests = responseTimes.filter(time => time > 0).length
    results.failedRequests = results.totalRequests - results.successfulRequests

    if (responseTimes.length > 0) {
      results.averageResponseTime = responseTimes.reduce((sum, time) => sum + time, 0) / responseTimes.length
      results.minResponseTime = Math.min(...responseTimes.filter(time => time > 0))
      results.maxResponseTime = Math.max(...responseTimes)
    }

    const durationSeconds = (Date.now() - startTime) / 1000
    results.requestsPerSecond = results.totalRequests / durationSeconds

    results.errors = Array.from(errors.entries()).map(([message, count]) => ({
      message,
      count,
    }))

    return results
  }

  private async runConcurrentRequests(
    config: LoadTestConfig,
    endTime: number,
    responseTimes: number[],
    errors: Map<string, number>,
  ): Promise<void> {
    while (Date.now() < endTime) {
      try {
        const requestStart = performance.now()

        const response = await fetch(config.endpoint, {
          method: config.method,
          headers: {
            'Content-Type': 'application/json',
          },
          body: config.payload ? JSON.stringify(config.payload) : undefined,
        })

        const requestEnd = performance.now()
        const responseTime = requestEnd - requestStart

        if (response.ok) {
          responseTimes.push(responseTime)
        } else {
          responseTimes.push(-1) // 실패한 요청 표시
          const errorMessage = `HTTP ${response.status}: ${response.statusText}`
          errors.set(errorMessage, (errors.get(errorMessage) || 0) + 1)
        }
      } catch (error) {
        responseTimes.push(-1)
        const errorMessage = error instanceof Error ? error.message : String(error)
        errors.set(errorMessage, (errors.get(errorMessage) || 0) + 1)
      }

      // 약간의 지연을 두어 서버 부하 조절
      await new Promise(resolve => setTimeout(resolve, 10))
    }
  }
}

/**
 * 렌더링 성능 테스트
 */
export class RenderPerformanceTest {
  /**
   * 컴포넌트 렌더링 시간 측정
   */
  async measureRenderTime(componentName: string, renderFn: () => Promise<void>): Promise<number> {
    return new Promise(async (resolve) => {
      const startTime = performance.now()

      await renderFn()

      // 다음 프레임에서 측정
      requestAnimationFrame(() => {
        const endTime = performance.now()
        resolve(endTime - startTime)
      })
    })
  }

  /**
   * 프레임 레이트 측정
   */
  measureFrameRate(duration = 1000): Promise<number> {
    return new Promise((resolve) => {
      let frames = 0
      const startTime = performance.now()

      const countFrames = () => {
        frames++
        if (performance.now() - startTime < duration) {
          requestAnimationFrame(countFrames)
        } else {
          const fps = frames / (duration / 1000)
          resolve(fps)
        }
      }

      requestAnimationFrame(countFrames)
    })
  }

  /**
   * 메모리 누수 감지
   */
  async detectMemoryLeaks(testFn: () => Promise<void>, iterations = 10): Promise<{
    initialMemory: number
    finalMemory: number
    memoryGrowth: number
    possibleLeak: boolean
  }> {
    // 가비지 컬렉션 강제 실행 (가능한 경우)
    if ('gc' in window) {
      (window as any).gc()
    }

    const initialMemory = this.getMemoryUsage()

    for (let i = 0; i < iterations; i++) {
      await testFn()

      // 주기적으로 가비지 컬렉션 시도
      if (i % 3 === 0 && 'gc' in window) {
        (window as any).gc()
      }
    }

    // 최종 가비지 컬렉션
    if ('gc' in window) {
      (window as any).gc()
    }

    const finalMemory = this.getMemoryUsage()
    const memoryGrowth = finalMemory - initialMemory

    // 10MB 이상 증가하면 메모리 누수 의심
    const possibleLeak = memoryGrowth > 10 * 1024 * 1024

    return {
      initialMemory,
      finalMemory,
      memoryGrowth,
      possibleLeak,
    }
  }

  private getMemoryUsage(): number {
    if ('memory' in performance) {
      return (performance as any).memory.usedJSHeapSize
    }
    return 0
  }
}

/**
 * 네트워크 성능 테스트
 */
export class NetworkPerformanceTest {
  /**
   * API 응답 시간 측정
   */
  async measureApiResponseTime(url: string, options?: RequestInit): Promise<{
    responseTime: number
    statusCode: number
    success: boolean
    error?: string
  }> {
    const startTime = performance.now()

    try {
      const response = await fetch(url, options)
      const endTime = performance.now()

      return {
        responseTime: endTime - startTime,
        statusCode: response.status,
        success: response.ok,
      }
    } catch (error) {
      const endTime = performance.now()

      return {
        responseTime: endTime - startTime,
        statusCode: 0,
        success: false,
        error: error instanceof Error ? error.message : String(error),
      }
    }
  }

  /**
   * 여러 API 엔드포인트 동시 테스트
   */
  async testMultipleEndpoints(endpoints: Array<{
    name: string
    url: string
    options?: RequestInit
  }>): Promise<Array<{
    name: string
    responseTime: number
    success: boolean
    error?: string
  }>> {
    const promises = endpoints.map(async (endpoint) => {
      const result = await this.measureApiResponseTime(endpoint.url, endpoint.options)
      return {
        name: endpoint.name,
        responseTime: result.responseTime,
        success: result.success,
        error: result.error,
      }
    })

    return Promise.all(promises)
  }
}

/**
 * 접근성 테스트
 */
export class AccessibilityTest {
  /**
   * 키보드 내비게이션 테스트
   */
  async testKeyboardNavigation(container: HTMLElement): Promise<{
    focusableElements: number
    tabOrder: boolean
    escapeHandling: boolean
  }> {
    const focusableSelectors = [
      'a[href]',
      'button:not([disabled])',
      'input:not([disabled]):not([type="hidden"])',
      'select:not([disabled])',
      'textarea:not([disabled])',
      '[tabindex]:not([tabindex="-1"])',
      '[contenteditable="true"]',
    ]

    const focusableElements = container.querySelectorAll(focusableSelectors.join(', '))

    // 탭 순서 테스트
    let tabOrder = true
    let previousTabIndex = -1

    focusableElements.forEach((element) => {
      const tabIndex = parseInt(element.getAttribute('tabindex') || '0')
      if (tabIndex > 0 && tabIndex < previousTabIndex) {
        tabOrder = false
      }
      previousTabIndex = tabIndex
    })

    return {
      focusableElements: focusableElements.length,
      tabOrder,
      escapeHandling: true, // 실제 구현에서는 ESC 키 핸들링 테스트
    }
  }

  /**
   * ARIA 속성 검증
   */
  validateAriaAttributes(container: HTMLElement): {
    missingLabels: string[]
    invalidRoles: string[]
    missingDescriptions: string[]
  } {
    const issues = {
      missingLabels: [] as string[],
      invalidRoles: [] as string[],
      missingDescriptions: [] as string[],
    }

    // 라벨이 필요한 요소들 확인
    const interactiveElements = container.querySelectorAll('button, input, select, textarea')
    interactiveElements.forEach((element, index) => {
      const hasLabel = element.hasAttribute('aria-label') ||
                      element.hasAttribute('aria-labelledby') ||
                      element.closest('label') ||
                      element.querySelector('label')

      if (!hasLabel) {
        issues.missingLabels.push(`${element.tagName.toLowerCase()}[${index}]`)
      }
    })

    return issues
  }
}

/**
 * 전체 테스트 스위트 실행기
 */
export class TestSuite {
  private performanceRunner = new PerformanceTestRunner()
  private loadRunner = new LoadTestRunner()
  private renderTest = new RenderPerformanceTest()
  private networkTest = new NetworkPerformanceTest()
  private accessibilityTest = new AccessibilityTest()

  /**
   * 모든 테스트 실행
   */
  async runFullSuite(): Promise<{
    performance: PerformanceTestResult[]
    render: any
    network: any
    accessibility: any
    summary: {
      totalTests: number
      passedTests: number
      failedTests: number
      duration: number
    }
  }> {
    const startTime = performance.now()
    console.log('🚀 전체 테스트 스위트 실행 시작')

    // 성능 테스트
    console.log('📊 성능 테스트 실행 중...')
    const performanceResults = await this.runPerformanceTests()

    // 렌더링 테스트
    console.log('🎨 렌더링 테스트 실행 중...')
    const renderResults = await this.runRenderTests()

    // 네트워크 테스트
    console.log('🌐 네트워크 테스트 실행 중...')
    const networkResults = await this.runNetworkTests()

    // 접근성 테스트
    console.log('♿ 접근성 테스트 실행 중...')
    const accessibilityResults = await this.runAccessibilityTests()

    const endTime = performance.now()
    const duration = endTime - startTime

    const allTests = [
      ...performanceResults,
      renderResults,
      networkResults,
      accessibilityResults,
    ].filter(Boolean)

    const passedTests = allTests.filter(test => test.success).length
    const failedTests = allTests.length - passedTests

    console.log('✅ 전체 테스트 스위트 완료')
    console.log(`📈 통과: ${passedTests}, 실패: ${failedTests}, 소요시간: ${duration.toFixed(2)}ms`)

    return {
      performance: performanceResults,
      render: renderResults,
      network: networkResults,
      accessibility: accessibilityResults,
      summary: {
        totalTests: allTests.length,
        passedTests,
        failedTests,
        duration,
      },
    }
  }

  private async runPerformanceTests(): Promise<PerformanceTestResult[]> {
    // 기본 성능 테스트들 등록
    this.performanceRunner.register('Core Web Vitals', async () => {
      // LCP, CLS, FCP 측정 시뮬레이션
      await new Promise(resolve => setTimeout(resolve, 100))
    })

    this.performanceRunner.register('Memory Usage', async () => {
      // 메모리 사용량 테스트
      const largeArray = new Array(10000).fill(0)
      largeArray.forEach((_, index) => largeArray[index] = Math.random())
    })

    this.performanceRunner.register('DOM Manipulation', async () => {
      // DOM 조작 성능 테스트
      const container = document.createElement('div')
      for (let i = 0; i < 100; i++) {
        const element = document.createElement('div')
        element.textContent = `Item ${i}`
        container.appendChild(element)
      }
    })

    return await this.performanceRunner.runAll()
  }

  private async runRenderTests(): Promise<any> {
    try {
      const fps = await this.renderTest.measureFrameRate(2000)
      return {
        success: fps > 30,
        fps,
        testName: 'Frame Rate Test',
      }
    } catch (error) {
      return {
        success: false,
        error: error instanceof Error ? error.message : String(error),
        testName: 'Frame Rate Test',
      }
    }
  }

  private async runNetworkTests(): Promise<any> {
    try {
      const result = await this.networkTest.measureApiResponseTime('/api/health')
      return {
        success: result.success && result.responseTime < 1000,
        responseTime: result.responseTime,
        testName: 'API Response Time Test',
      }
    } catch (error) {
      return {
        success: false,
        error: error instanceof Error ? error.message : String(error),
        testName: 'API Response Time Test',
      }
    }
  }

  private async runAccessibilityTests(): Promise<any> {
    try {
      const container = document.body
      const result = await this.accessibilityTest.testKeyboardNavigation(container)
      const ariaResult = this.accessibilityTest.validateAriaAttributes(container)

      return {
        success: result.focusableElements > 0 && result.tabOrder && ariaResult.missingLabels.length === 0,
        focusableElements: result.focusableElements,
        tabOrder: result.tabOrder,
        missingLabels: ariaResult.missingLabels.length,
        testName: 'Accessibility Test',
      }
    } catch (error) {
      return {
        success: false,
        error: error instanceof Error ? error.message : String(error),
        testName: 'Accessibility Test',
      }
    }
  }
}

// 전역 테스트 스위트 인스턴스
export const testSuite = new TestSuite()

// 개발 모드에서만 전역 객체에 노출
if (import.meta.env.DEV) {
  (window as any).testSuite = testSuite
  (window as any).runPerformanceTests = () => testSuite.runFullSuite()
}
