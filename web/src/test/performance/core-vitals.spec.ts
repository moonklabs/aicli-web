import { type Page, expect, test } from '@playwright/test'

/**
 * Core Web Vitals 성능 테스트
 */
test.describe('Core Web Vitals 성능 테스트', () => {

  test.beforeEach(async ({ page }) => {
    // 성능 측정을 위한 기본 설정
    await page.addInitScript(() => {
      // Web Vitals 라이브러리 주입
      window.webVitalsData = []

      // 성능 옵저버 설정
      if ('PerformanceObserver' in window) {
        try {
          // LCP 측정
          const lcpObserver = new PerformanceObserver((list) => {
            const entries = list.getEntries()
            const lastEntry = entries[entries.length - 1]
            window.webVitalsData.push({
              name: 'LCP',
              value: lastEntry.startTime,
              timestamp: Date.now(),
            })
          })
          lcpObserver.observe({ type: 'largest-contentful-paint', buffered: true })

          // FID 측정
          const fidObserver = new PerformanceObserver((list) => {
            for (const entry of list.getEntries()) {
              if (entry.processingStart && entry.startTime) {
                window.webVitalsData.push({
                  name: 'FID',
                  value: entry.processingStart - entry.startTime,
                  timestamp: Date.now(),
                })
              }
            }
          })
          fidObserver.observe({ type: 'first-input', buffered: true })

          // CLS 측정
          let clsScore = 0
          const clsObserver = new PerformanceObserver((list) => {
            for (const entry of list.getEntries()) {
              if (!entry.hadRecentInput) {
                clsScore += entry.value
              }
            }
            window.webVitalsData.push({
              name: 'CLS',
              value: clsScore,
              timestamp: Date.now(),
            })
          })
          clsObserver.observe({ type: 'layout-shift', buffered: true })
        } catch (error) {
          console.warn('Performance Observer 설정 실패:', error)
        }
      }
    })
  })

  test('홈페이지 Core Web Vitals', async ({ page }) => {
    // 페이지 로드
    const startTime = Date.now()
    await page.goto('/')

    // LCP 대기 (최대 10초)
    await page.waitForFunction(() => {
      return window.webVitalsData?.some(data => data.name === 'LCP') || Date.now() - startTime > 10000
    }, {}, startTime)

    // 성능 메트릭 수집
    const webVitalsData = await page.evaluate(() => window.webVitalsData || [])
    const performanceEntries = await page.evaluate(() => {
      return {
        navigation: performance.getEntriesByType('navigation')[0],
        paint: performance.getEntriesByType('paint'),
        memory: 'memory' in performance ? (performance as any).memory : null,
      }
    })

    console.log('📊 성능 메트릭:', {
      webVitals: webVitalsData,
      performance: performanceEntries,
    })

    // LCP 검증 (2.5초 이하)
    const lcpData = webVitalsData.find(data => data.name === 'LCP')
    if (lcpData) {
      expect(lcpData.value).toBeLessThan(2500)
      console.log(`✅ LCP: ${lcpData.value.toFixed(2)}ms`)
    }

    // FCP 검증 (1.8초 이하)
    const fcpEntry = performanceEntries.paint?.find(entry => entry.name === 'first-contentful-paint')
    if (fcpEntry) {
      expect(fcpEntry.startTime).toBeLessThan(1800)
      console.log(`✅ FCP: ${fcpEntry.startTime.toFixed(2)}ms`)
    }

    // CLS 검증 (0.1 이하)
    const clsData = webVitalsData.find(data => data.name === 'CLS')
    if (clsData) {
      expect(clsData.value).toBeLessThan(0.1)
      console.log(`✅ CLS: ${clsData.value.toFixed(4)}`)
    }

    // 메모리 사용량 검증 (50MB 이하)
    if (performanceEntries.memory) {
      const memoryUsageMB = performanceEntries.memory.usedJSHeapSize / (1024 * 1024)
      expect(memoryUsageMB).toBeLessThan(50)
      console.log(`✅ Memory: ${memoryUsageMB.toFixed(2)}MB`)
    }
  })

  test('워크스페이스 페이지 성능', async ({ page }) => {
    await page.goto('/workspace')

    // 페이지 로딩 완료 대기
    await page.waitForLoadState('networkidle')

    // 성능 메트릭 수집
    const metrics = await page.evaluate(() => {
      const navigation = performance.getEntriesByType('navigation')[0] as PerformanceNavigationTiming
      return {
        domContentLoaded: navigation.domContentLoadedEventEnd - navigation.domContentLoadedEventStart,
        loadComplete: navigation.loadEventEnd - navigation.loadEventStart,
        domInteractive: navigation.domInteractive - navigation.fetchStart,
        firstByte: navigation.responseStart - navigation.requestStart,
      }
    })

    console.log('📊 워크스페이스 성능 메트릭:', metrics)

    // DOM 인터랙티브 시간 검증 (3초 이하)
    expect(metrics.domInteractive).toBeLessThan(3000)

    // TTFB 검증 (800ms 이하)
    expect(metrics.firstByte).toBeLessThan(800)

    // 로드 완료 시간 검증 (5초 이하)
    expect(metrics.loadComplete).toBeLessThan(5000)
  })

  test('터미널 페이지 성능', async ({ page }) => {
    await page.goto('/terminal')

    // 터미널 컴포넌트 로딩 대기
    await page.waitForSelector('[data-testid="terminal-container"]', { timeout: 10000 })

    // 성능 메트릭 측정
    const startTime = Date.now()

    // 터미널 인터랙션 시뮬레이션
    await page.click('[data-testid="terminal-input"]')
    await page.type('[data-testid="terminal-input"]', 'test command')

    const interactionTime = Date.now() - startTime

    // 인터랙션 응답 시간 검증 (100ms 이하)
    expect(interactionTime).toBeLessThan(100)
    console.log(`✅ 터미널 인터랙션 시간: ${interactionTime}ms`)
  })

  test('번들 크기 검증', async ({ page }) => {
    // 리소스 로딩 추적
    const resources: Array<{ url: string; size: number; type: string }> = []

    page.on('response', async (response) => {
      if (response.url().includes(page.url().split('/')[2])) {
        try {
          const contentLength = response.headers()['content-length']
          if (contentLength) {
            resources.push({
              url: response.url(),
              size: parseInt(contentLength, 10),
              type: response.request().resourceType(),
            })
          }
        } catch (error) {
          // 무시
        }
      }
    })

    await page.goto('/')
    await page.waitForLoadState('networkidle')

    // JavaScript 번들 크기 계산
    const jsResources = resources.filter(r => r.type === 'script')
    const totalJSSize = jsResources.reduce((sum, r) => sum + r.size, 0)
    const totalJSSizeMB = totalJSSize / (1024 * 1024)

    console.log('📦 번들 크기 분석:', {
      js: `${totalJSSizeMB.toFixed(2)}MB`,
      resources: jsResources.length,
      details: jsResources.map(r => ({
        url: r.url.split('/').pop(),
        size: `${(r.size / 1024).toFixed(2)}KB`,
      })),
    })

    // JavaScript 번들 크기 검증 (5MB 이하)
    expect(totalJSSizeMB).toBeLessThan(5)

    // 주요 청크 크기 검증
    const mainChunk = jsResources.find(r => r.url.includes('index') || r.url.includes('main'))
    if (mainChunk) {
      const mainChunkMB = mainChunk.size / (1024 * 1024)
      expect(mainChunkMB).toBeLessThan(2) // 메인 청크는 2MB 이하
    }
  })

  test('이미지 최적화 검증', async ({ page }) => {
    const imageResources: Array<{ url: string; size: number }> = []

    page.on('response', async (response) => {
      if (response.request().resourceType() === 'image') {
        const contentLength = response.headers()['content-length']
        if (contentLength) {
          imageResources.push({
            url: response.url(),
            size: parseInt(contentLength, 10),
          })
        }
      }
    })

    await page.goto('/')
    await page.waitForLoadState('networkidle')

    // 이미지 최적화 검증
    for (const image of imageResources) {
      const imageSizeKB = image.size / 1024

      // 개별 이미지 크기 검증 (500KB 이하)
      expect(imageSizeKB).toBeLessThan(500)

      // WebP 또는 AVIF 형식 권장
      const isOptimizedFormat = image.url.includes('.webp') ||
                               image.url.includes('.avif') ||
                               image.url.includes('.svg')

      if (!isOptimizedFormat && imageSizeKB > 50) {
        console.warn(`⚠️  최적화되지 않은 이미지: ${image.url} (${imageSizeKB.toFixed(2)}KB)`)
      }
    }

    console.log('🖼️  이미지 최적화 분석:', {
      총개수: imageResources.length,
      총크기: `${(imageResources.reduce((sum, img) => sum + img.size, 0) / 1024).toFixed(2)}KB`,
      평균크기: `${(imageResources.reduce((sum, img) => sum + img.size, 0) / imageResources.length / 1024).toFixed(2)}KB`,
    })
  })
})

// 글로벌 타입 확장
declare global {
  interface Window {
    webVitalsData: Array<{
      name: string
      value: number
      timestamp: number
    }>
  }
}