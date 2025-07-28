#!/usr/bin/env node

/**
 * 성능 모니터링 스크립트
 * CI/CD 파이프라인에서 성능 메트릭을 수집하고 분석합니다.
 */

import fs from 'fs'
import path from 'path'
import { fileURLToPath } from 'url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

// 성능 임계값 설정
const PERFORMANCE_THRESHOLDS = {
  lighthouse: {
    performance: 80,
    accessibility: 90,
    bestPractices: 80,
    seo: 80,
    pwa: 80,
  },
  webVitals: {
    lcp: 2500,      // ms
    fid: 100,       // ms
    cls: 0.1,       // score
    fcp: 1800,      // ms
    ttfb: 800,       // ms
  },
  bundle: {
    maxSizeMB: 5,
    maxChunks: 20,
  },
  tests: {
    minCoverage: 80,
    maxDuration: 300000, // 5분
  },
}

/**
 * Lighthouse 결과 분석
 */
function analyzeLighthouseResults() {
  console.log('🔍 Lighthouse 결과 분석 중...')

  const lighthouseDir = path.join(process.cwd(), '.lighthouseci')

  if (!fs.existsSync(lighthouseDir)) {
    console.warn('⚠️  Lighthouse 결과를 찾을 수 없습니다.')
    return null
  }

  try {
    const files = fs.readdirSync(lighthouseDir)
    const manifestFile = files.find(f => f.includes('manifest.json'))

    if (!manifestFile) {
      console.warn('⚠️  Lighthouse manifest 파일을 찾을 수 없습니다.')
      return null
    }

    const manifestPath = path.join(lighthouseDir, manifestFile)
    const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'))

    const results = []

    for (const report of manifest) {
      if (report.summary) {
        const summary = report.summary
        const result = {
          url: report.url,
          scores: {
            performance: Math.round(summary.performance * 100),
            accessibility: Math.round(summary.accessibility * 100),
            bestPractices: Math.round(summary['best-practices'] * 100),
            seo: Math.round(summary.seo * 100),
            pwa: summary.pwa ? Math.round(summary.pwa * 100) : null,
          },
          metrics: {
            lcp: summary['largest-contentful-paint'],
            fid: summary['first-input-delay'],
            cls: summary['cumulative-layout-shift'],
            fcp: summary['first-contentful-paint'],
            ttfb: summary['time-to-first-byte'],
          },
          passed: true,
          issues: [],
        }

        // 임계값 검사
        Object.entries(PERFORMANCE_THRESHOLDS.lighthouse).forEach(([key, threshold]) => {
          const score = result.scores[key]
          if (score !== null && score < threshold) {
            result.passed = false
            result.issues.push(`${key} 점수가 임계값보다 낮습니다: ${score} < ${threshold}`)
          }
        })

        Object.entries(PERFORMANCE_THRESHOLDS.webVitals).forEach(([key, threshold]) => {
          const metric = result.metrics[key]
          if (metric !== null && metric > threshold) {
            result.passed = false
            result.issues.push(`${key} 메트릭이 임계값을 초과했습니다: ${metric} > ${threshold}`)
          }
        })

        results.push(result)
      }
    }

    return results
  } catch (error) {
    console.error('❌ Lighthouse 결과 분석 실패:', error.message)
    return null
  }
}

/**
 * 번들 크기 분석
 */
function analyzeBundleSize() {
  console.log('📦 번들 크기 분석 중...')

  const distDir = path.join(process.cwd(), 'dist')

  if (!fs.existsSync(distDir)) {
    console.warn('⚠️  빌드 출력 디렉토리를 찾을 수 없습니다.')
    return null
  }

  try {
    const stats = getBundleStats(distDir)
    const result = {
      totalSize: stats.totalSize,
      totalSizeMB: stats.totalSizeMB,
      files: stats.files,
      passed: true,
      issues: [],
    }

    // 번들 크기 검사
    if (stats.totalSizeMB > PERFORMANCE_THRESHOLDS.bundle.maxSizeMB) {
      result.passed = false
      result.issues.push(`번들 크기가 임계값을 초과했습니다: ${stats.totalSizeMB.toFixed(2)}MB > ${PERFORMANCE_THRESHOLDS.bundle.maxSizeMB}MB`)
    }

    // 청크 수 검사
    const jsFiles = stats.files.filter(f => f.name.endsWith('.js'))
    if (jsFiles.length > PERFORMANCE_THRESHOLDS.bundle.maxChunks) {
      result.passed = false
      result.issues.push(`JS 파일 수가 임계값을 초과했습니다: ${jsFiles.length} > ${PERFORMANCE_THRESHOLDS.bundle.maxChunks}`)
    }

    return result
  } catch (error) {
    console.error('❌ 번들 크기 분석 실패:', error.message)
    return null
  }
}

/**
 * 디렉토리 크기 계산
 */
function getBundleStats(dir) {
  const files = []
  let totalSize = 0

  function traverse(currentDir) {
    const items = fs.readdirSync(currentDir)

    for (const item of items) {
      const itemPath = path.join(currentDir, item)
      const stat = fs.statSync(itemPath)

      if (stat.isDirectory()) {
        traverse(itemPath)
      } else {
        const relativePath = path.relative(dir, itemPath)
        const fileInfo = {
          name: relativePath,
          size: stat.size,
          sizeMB: stat.size / (1024 * 1024),
        }
        files.push(fileInfo)
        totalSize += stat.size
      }
    }
  }

  traverse(dir)

  return {
    files: files.sort((a, b) => b.size - a.size),
    totalSize,
    totalSizeMB: totalSize / (1024 * 1024),
  }
}

/**
 * 테스트 결과 분석
 */
function analyzeTestResults() {
  console.log('🧪 테스트 결과 분석 중...')

  const testResultsPath = path.join(process.cwd(), 'test-results', 'e2e-results.json')
  const coverageReportPath = path.join(process.cwd(), 'coverage', 'coverage-summary.json')

  const result = {
    e2e: null,
    coverage: null,
    passed: true,
    issues: [],
  }

  // E2E 테스트 결과
  if (fs.existsSync(testResultsPath)) {
    try {
      const e2eResults = JSON.parse(fs.readFileSync(testResultsPath, 'utf8'))
      result.e2e = {
        total: e2eResults.stats?.tests || 0,
        passed: e2eResults.stats?.passed || 0,
        failed: e2eResults.stats?.failed || 0,
        duration: e2eResults.stats?.duration || 0,
      }

      if (result.e2e.failed > 0) {
        result.passed = false
        result.issues.push(`E2E 테스트 실패: ${result.e2e.failed}개`)
      }

      if (result.e2e.duration > PERFORMANCE_THRESHOLDS.tests.maxDuration) {
        result.passed = false
        result.issues.push(`테스트 실행 시간이 임계값을 초과했습니다: ${result.e2e.duration}ms > ${PERFORMANCE_THRESHOLDS.tests.maxDuration}ms`)
      }
    } catch (error) {
      console.warn('⚠️  E2E 테스트 결과를 읽을 수 없습니다:', error.message)
    }
  }

  // 커버리지 결과
  if (fs.existsSync(coverageReportPath)) {
    try {
      const coverageData = JSON.parse(fs.readFileSync(coverageReportPath, 'utf8'))
      const total = coverageData.total

      result.coverage = {
        lines: total.lines.pct,
        functions: total.functions.pct,
        branches: total.branches.pct,
        statements: total.statements.pct,
      }

      if (result.coverage.lines < PERFORMANCE_THRESHOLDS.tests.minCoverage) {
        result.passed = false
        result.issues.push(`코드 커버리지가 임계값보다 낮습니다: ${result.coverage.lines}% < ${PERFORMANCE_THRESHOLDS.tests.minCoverage}%`)
      }
    } catch (error) {
      console.warn('⚠️  커버리지 결과를 읽을 수 없습니다:', error.message)
    }
  }

  return result
}

/**
 * 성능 리포트 생성
 */
function generatePerformanceReport(lighthouseResults, bundleResults, testResults) {
  const timestamp = new Date().toISOString()
  const report = {
    timestamp,
    lighthouse: lighthouseResults,
    bundle: bundleResults,
    tests: testResults,
    overall: {
      passed: true,
      issues: [],
    },
  }

  // 전체 결과 계산
  const allResults = [lighthouseResults, bundleResults, testResults].filter(Boolean)

  for (const result of allResults) {
    if (Array.isArray(result)) {
      for (const item of result) {
        if (!item.passed) {
          report.overall.passed = false
          report.overall.issues.push(...item.issues)
        }
      }
    } else if (result && !result.passed) {
      report.overall.passed = false
      report.overall.issues.push(...result.issues)
    }
  }

  return report
}

/**
 * 리포트 출력
 */
function printReport(report) {
  console.log(`\n${'='.repeat(60)}`)
  console.log('📊 성능 모니터링 리포트')
  console.log('='.repeat(60))
  console.log(`⏰ 생성 시간: ${new Date(report.timestamp).toLocaleString('ko-KR')}`)
  console.log()

  // Lighthouse 결과
  if (report.lighthouse && report.lighthouse.length > 0) {
    console.log('🚀 Lighthouse 결과:')
    for (const result of report.lighthouse) {
      console.log(`  📄 ${result.url}`)
      console.log(`    성능: ${result.scores.performance}/100`)
      console.log(`    접근성: ${result.scores.accessibility}/100`)
      console.log(`    모범사례: ${result.scores.bestPractices}/100`)
      console.log(`    SEO: ${result.scores.seo}/100`)
      if (result.scores.pwa !== null) {
        console.log(`    PWA: ${result.scores.pwa}/100`)
      }
      console.log()
    }
  }

  // 번들 크기 결과
  if (report.bundle) {
    console.log('📦 번들 크기:')
    console.log(`  총 크기: ${report.bundle.totalSizeMB.toFixed(2)}MB`)
    console.log(`  파일 수: ${report.bundle.files.length}개`)
    console.log('  주요 파일:')
    report.bundle.files.slice(0, 5).forEach(file => {
      console.log(`    ${file.name}: ${file.sizeMB.toFixed(2)}MB`)
    })
    console.log()
  }

  // 테스트 결과
  if (report.tests) {
    console.log('🧪 테스트 결과:')
    if (report.tests.e2e) {
      console.log(`  E2E: ${report.tests.e2e.passed}/${report.tests.e2e.total} 통과`)
      console.log(`  실행시간: ${(report.tests.e2e.duration / 1000).toFixed(2)}초`)
    }
    if (report.tests.coverage) {
      console.log(`  커버리지: ${report.tests.coverage.lines}%`)
    }
    console.log()
  }

  // 전체 결과
  console.log('📋 전체 결과:')
  if (report.overall.passed) {
    console.log('✅ 모든 성능 검사를 통과했습니다!')
  } else {
    console.log('❌ 성능 검사에서 문제가 발견되었습니다:')
    report.overall.issues.forEach(issue => {
      console.log(`  • ${issue}`)
    })
  }
  console.log()
}

/**
 * 리포트 저장
 */
function saveReport(report) {
  const reportDir = path.join(process.cwd(), 'performance-reports')
  const reportFile = path.join(reportDir, `performance-report-${Date.now()}.json`)

  if (!fs.existsSync(reportDir)) {
    fs.mkdirSync(reportDir, { recursive: true })
  }

  fs.writeFileSync(reportFile, JSON.stringify(report, null, 2))
  console.log(`💾 리포트가 저장되었습니다: ${reportFile}`)
}

/**
 * 메인 실행 함수
 */
async function main() {
  console.log('🚀 성능 모니터링 시작...')
  console.log()

  try {
    // 각 분석 실행
    const lighthouseResults = analyzeLighthouseResults()
    const bundleResults = analyzeBundleSize()
    const testResults = analyzeTestResults()

    // 리포트 생성
    const report = generatePerformanceReport(lighthouseResults, bundleResults, testResults)

    // 리포트 출력
    printReport(report)

    // 리포트 저장
    saveReport(report)

    // 환경 변수로 결과 전달 (CI/CD용)
    if (process.env.GITHUB_ACTIONS) {
      console.log('📤 GitHub Actions 결과 설정 중...')
      console.log(`::set-output name=performance-passed::${report.overall.passed}`)
      console.log(`::set-output name=issues-count::${report.overall.issues.length}`)
    }

    // 종료 코드 설정
    process.exit(report.overall.passed ? 0 : 1)
  } catch (error) {
    console.error('❌ 성능 모니터링 실행 중 오류 발생:', error)
    process.exit(1)
  }
}

// 스크립트가 직접 실행될 때만 main 함수 호출
if (import.meta.url === `file://${process.argv[1]}`) {
  main()
}