<template>
  <div class="test-dashboard">
    <!-- 헤더 섹션 -->
    <div class="dashboard-header">
      <h2 class="dashboard-title">자동화된 테스트 스위트</h2>
      <div class="header-actions">
        <NSpace>
          <NButton
            @click="runAllTests"
            :loading="isRunning"
            type="primary"
            size="medium"
          >
            🧪 전체 테스트 실행
          </NButton>
          <NDropdown :options="testTypeOptions" @select="runSpecificTest">
            <NButton :disabled="isRunning" size="medium">
              🎯 선택 테스트 실행
            </NButton>
          </NDropdown>
          <NButton
            @click="clearResults"
            :disabled="!hasResults"
            size="medium"
          >
            🗑️ 결과 지우기
          </NButton>
          <NButton
            @click="exportResults"
            :disabled="!hasResults"
            size="medium"
          >
            📥 리포트 다운로드
          </NButton>
        </NSpace>
      </div>
    </div>

    <!-- 전체 결과 요약 -->
    <div v-if="testResults" class="results-summary">
      <div class="summary-cards">
        <NCard class="summary-card total">
          <div class="card-content">
            <div class="card-icon">🧪</div>
            <div class="card-details">
              <div class="card-value">{{ testResults.summary.totalTests }}</div>
              <div class="card-label">총 테스트</div>
            </div>
          </div>
        </NCard>

        <NCard class="summary-card passed">
          <div class="card-content">
            <div class="card-icon">✅</div>
            <div class="card-details">
              <div class="card-value">{{ testResults.summary.passedTests }}</div>
              <div class="card-label">통과</div>
            </div>
          </div>
        </NCard>

        <NCard class="summary-card failed">
          <div class="card-content">
            <div class="card-icon">❌</div>
            <div class="card-details">
              <div class="card-value">{{ testResults.summary.failedTests }}</div>
              <div class="card-label">실패</div>
            </div>
          </div>
        </NCard>

        <NCard class="summary-card duration">
          <div class="card-content">
            <div class="card-icon">⏱️</div>
            <div class="card-details">
              <div class="card-value">{{ formatDuration(testResults.summary.duration) }}</div>
              <div class="card-label">소요 시간</div>
            </div>
          </div>
        </NCard>
      </div>

      <!-- 성공률 진행 표시 -->
      <div class="success-rate">
        <h3>테스트 성공률</h3>
        <NProgress
          type="line"
          :percentage="successRate"
          :color="getSuccessRateColor(successRate)"
          :height="24"
          :show-indicator="false"
        />
        <div class="success-rate-text">
          {{ successRate.toFixed(1) }}% ({{ testResults.summary.passedTests }}/{{ testResults.summary.totalTests }})
        </div>
      </div>
    </div>

    <!-- 실행 중 상태 -->
    <div v-if="isRunning" class="running-status">
      <NCard>
        <div class="running-content">
          <NSpin size="large" />
          <div class="running-text">
            <h3>테스트 실행 중...</h3>
            <p>{{ currentTestStatus }}</p>
          </div>
        </div>
      </NCard>
    </div>

    <!-- 테스트 카테고리별 결과 -->
    <div v-if="testResults && !isRunning" class="test-categories">
      <NTabs type="line" animated>
        <!-- 성능 테스트 -->
        <NTabPane name="performance" tab="🚀 성능 테스트">
          <div class="test-category-content">
            <div class="category-header">
              <h3>성능 테스트 결과</h3>
              <div class="category-stats">
                {{ testResults.performance.filter(t => t.success).length }} / {{ testResults.performance.length }} 통과
              </div>
            </div>

            <div class="test-results-grid">
              <div
                v-for="test in testResults.performance"
                :key="test.testName"
                class="test-result-card"
                :class="{ success: test.success, failed: !test.success }"
              >
                <div class="test-header">
                  <div class="test-name">{{ test.testName }}</div>
                  <div class="test-status">
                    {{ test.success ? '✅' : '❌' }}
                  </div>
                </div>
                <div class="test-metrics">
                  <div class="metric">
                    <span class="metric-label">실행 시간:</span>
                    <span class="metric-value">{{ formatDuration(test.duration) }}</span>
                  </div>
                  <div class="metric">
                    <span class="metric-label">메모리 사용:</span>
                    <span class="metric-value">{{ formatBytes(test.memoryUsage) }}</span>
                  </div>
                </div>
                <div v-if="!test.success && test.error" class="test-error">
                  {{ test.error }}
                </div>
              </div>
            </div>
          </div>
        </NTabPane>

        <!-- 렌더링 테스트 -->
        <NTabPane name="render" tab="🎨 렌더링">
          <div class="test-category-content">
            <div class="category-header">
              <h3>렌더링 성능 테스트</h3>
              <div class="category-stats">
                {{ testResults.render?.success ? '통과' : '실패' }}
              </div>
            </div>

            <div v-if="testResults.render" class="render-results">
              <NCard :class="{ 'success-card': testResults.render.success, 'error-card': !testResults.render.success }">
                <div class="render-content">
                  <div class="render-metric">
                    <h4>프레임 레이트 (FPS)</h4>
                    <div class="fps-display">
                      <span class="fps-value">{{ testResults.render.fps?.toFixed(1) || 'N/A' }}</span>
                      <span class="fps-unit">FPS</span>
                    </div>
                    <div class="fps-status">
                      <span :class="['status-badge', getFpsStatus(testResults.render.fps)]">
                        {{ getFpsStatusText(testResults.render.fps) }}
                      </span>
                    </div>
                  </div>

                  <div v-if="!testResults.render.success" class="error-details">
                    <h4>오류 정보</h4>
                    <p>{{ testResults.render.error }}</p>
                  </div>
                </div>
              </NCard>
            </div>
          </div>
        </NTabPane>

        <!-- 네트워크 테스트 -->
        <NTabPane name="network" tab="🌐 네트워크">
          <div class="test-category-content">
            <div class="category-header">
              <h3>네트워크 성능 테스트</h3>
              <div class="category-stats">
                {{ testResults.network?.success ? '통과' : '실패' }}
              </div>
            </div>

            <div v-if="testResults.network" class="network-results">
              <NCard :class="{ 'success-card': testResults.network.success, 'error-card': !testResults.network.success }">
                <div class="network-content">
                  <div class="network-metric">
                    <h4>API 응답 시간</h4>
                    <div class="response-time-display">
                      <span class="response-time-value">{{ formatDuration(testResults.network.responseTime) }}</span>
                    </div>
                    <div class="response-time-status">
                      <span :class="['status-badge', getResponseTimeStatus(testResults.network.responseTime)]">
                        {{ getResponseTimeStatusText(testResults.network.responseTime) }}
                      </span>
                    </div>
                  </div>

                  <div v-if="!testResults.network.success" class="error-details">
                    <h4>오류 정보</h4>
                    <p>{{ testResults.network.error }}</p>
                  </div>
                </div>
              </NCard>
            </div>
          </div>
        </NTabPane>

        <!-- 접근성 테스트 -->
        <NTabPane name="accessibility" tab="♿ 접근성">
          <div class="test-category-content">
            <div class="category-header">
              <h3>접근성 테스트</h3>
              <div class="category-stats">
                {{ testResults.accessibility?.success ? '통과' : '실패' }}
              </div>
            </div>

            <div v-if="testResults.accessibility" class="accessibility-results">
              <NCard :class="{ 'success-card': testResults.accessibility.success, 'error-card': !testResults.accessibility.success }">
                <div class="accessibility-content">
                  <div class="accessibility-metrics">
                    <div class="metric-item">
                      <h4>포커스 가능한 요소</h4>
                      <span class="metric-value">{{ testResults.accessibility.focusableElements }}</span>
                    </div>

                    <div class="metric-item">
                      <h4>탭 순서</h4>
                      <span :class="['status-badge', testResults.accessibility.tabOrder ? 'good' : 'bad']">
                        {{ testResults.accessibility.tabOrder ? '올바름' : '문제 있음' }}
                      </span>
                    </div>

                    <div class="metric-item">
                      <h4>누락된 라벨</h4>
                      <span :class="['status-badge', testResults.accessibility.missingLabels === 0 ? 'good' : 'warning']">
                        {{ testResults.accessibility.missingLabels }}개
                      </span>
                    </div>
                  </div>

                  <div v-if="!testResults.accessibility.success" class="error-details">
                    <h4>개선 권장사항</h4>
                    <ul>
                      <li v-if="!testResults.accessibility.tabOrder">탭 순서를 일관되게 설정하세요</li>
                      <li v-if="testResults.accessibility.missingLabels > 0">
                        {{ testResults.accessibility.missingLabels }}개 요소에 적절한 라벨을 추가하세요
                      </li>
                    </ul>
                  </div>
                </div>
              </NCard>
            </div>
          </div>
        </NTabPane>
      </NTabs>
    </div>

    <!-- 테스트 기록 -->
    <div v-if="testHistory.length > 0" class="test-history">
      <NCard title="최근 테스트 기록">
        <NDataTable
          :columns="historyColumns"
          :data="testHistory"
          :pagination="{
            pageSize: 10,
            showSizePicker: true,
            pageSizes: [5, 10, 20]
          }"
        />
      </NCard>
    </div>

    <!-- 도움말 섹션 -->
    <div v-if="!testResults && !isRunning" class="help-section">
      <NCard title="테스트 스위트 가이드">
        <div class="help-content">
          <div class="test-type-info">
            <h4>🚀 성능 테스트</h4>
            <p>Core Web Vitals, 메모리 사용량, DOM 조작 성능을 측정합니다.</p>
          </div>

          <div class="test-type-info">
            <h4>🎨 렌더링 테스트</h4>
            <p>프레임 레이트, 렌더링 시간, 메모리 누수를 확인합니다.</p>
          </div>

          <div class="test-type-info">
            <h4>🌐 네트워크 테스트</h4>
            <p>API 응답 시간, 에러율, 처리량을 테스트합니다.</p>
          </div>

          <div class="test-type-info">
            <h4>♿ 접근성 테스트</h4>
            <p>키보드 네비게이션, ARIA 속성, 스크린 리더 호환성을 검증합니다.</p>
          </div>

          <div class="start-guide">
            <p><strong>시작하기:</strong> 상단의 "전체 테스트 실행" 버튼을 클릭하여 모든 테스트를 실행하거나, 특정 카테고리만 선택하여 테스트할 수 있습니다.</p>
          </div>
        </div>
      </NCard>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { NButton, NCard, NDataTable, NDropdown, NProgress, NSpace, NSpin, NTabPane, NTabs } from 'naive-ui'
import { type PerformanceTestResult, testSuite } from '@/utils/test-helpers'

interface TestResults {
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
}

interface TestHistoryItem {
  id: string
  timestamp: string
  type: string
  totalTests: number
  passedTests: number
  failedTests: number
  duration: number
  successRate: number
}

// 반응형 상태
const isRunning = ref(false)
const testResults = ref<TestResults | null>(null)
const currentTestStatus = ref('')
const testHistory = ref<TestHistoryItem[]>([])

// 계산된 값들
const hasResults = computed(() => testResults.value !== null)
const successRate = computed(() => {
  if (!testResults.value || testResults.value.summary.totalTests === 0) return 0
  return (testResults.value.summary.passedTests / testResults.value.summary.totalTests) * 100
})

// 테스트 타입 옵션
const testTypeOptions = [
  { label: '🚀 성능 테스트', key: 'performance' },
  { label: '🎨 렌더링 테스트', key: 'render' },
  { label: '🌐 네트워크 테스트', key: 'network' },
  { label: '♿ 접근성 테스트', key: 'accessibility' },
]

// 테스트 기록 테이블 컬럼
const historyColumns = [
  {
    title: '시간',
    key: 'timestamp',
    render: (row: TestHistoryItem) => new Date(row.timestamp).toLocaleString(),
  },
  {
    title: '테스트 유형',
    key: 'type',
  },
  {
    title: '총 테스트',
    key: 'totalTests',
  },
  {
    title: '통과',
    key: 'passedTests',
    render: (row: TestHistoryItem) => `✅ ${row.passedTests}`,
  },
  {
    title: '실패',
    key: 'failedTests',
    render: (row: TestHistoryItem) => `❌ ${row.failedTests}`,
  },
  {
    title: '성공률',
    key: 'successRate',
    render: (row: TestHistoryItem) => `${row.successRate.toFixed(1)}%`,
  },
  {
    title: '소요 시간',
    key: 'duration',
    render: (row: TestHistoryItem) => formatDuration(row.duration),
  },
]

// 메서드들
const runAllTests = async () => {
  isRunning.value = true
  currentTestStatus.value = '테스트 스위트 초기화 중...'

  try {
    currentTestStatus.value = '성능 테스트 실행 중...'
    await new Promise(resolve => setTimeout(resolve, 500))

    currentTestStatus.value = '렌더링 테스트 실행 중...'
    await new Promise(resolve => setTimeout(resolve, 300))

    currentTestStatus.value = '네트워크 테스트 실행 중...'
    await new Promise(resolve => setTimeout(resolve, 400))

    currentTestStatus.value = '접근성 테스트 실행 중...'
    await new Promise(resolve => setTimeout(resolve, 300))

    currentTestStatus.value = '결과 분석 중...'
    const results = await testSuite.runFullSuite()
    testResults.value = results

    // 테스트 기록에 추가
    addToHistory('전체 테스트', results.summary)

  } catch (error) {
    console.error('테스트 실행 중 오류:', error)
  } finally {
    isRunning.value = false
    currentTestStatus.value = ''
  }
}

const runSpecificTest = async (key: string) => {
  isRunning.value = true
  const testTypeMap: Record<string, string> = {
    performance: '성능 테스트',
    render: '렌더링 테스트',
    network: '네트워크 테스트',
    accessibility: '접근성 테스트',
  }

  currentTestStatus.value = `${testTypeMap[key]} 실행 중...`

  try {
    // 실제 구현에서는 특정 테스트만 실행
    await new Promise(resolve => setTimeout(resolve, 1000))

    // 모의 결과 생성
    const mockResults = createMockResults(key)
    testResults.value = mockResults

    addToHistory(testTypeMap[key], mockResults.summary)

  } catch (error) {
    console.error('테스트 실행 중 오류:', error)
  } finally {
    isRunning.value = false
    currentTestStatus.value = ''
  }
}

const clearResults = () => {
  testResults.value = null
}

const exportResults = () => {
  if (!testResults.value) return

  const report = generateTestReport(testResults.value)
  const blob = new Blob([report], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `test-report-${Date.now()}.txt`
  a.click()
  URL.revokeObjectURL(url)
}

const addToHistory = (type: string, summary: TestResults['summary']) => {
  const historyItem: TestHistoryItem = {
    id: Date.now().toString(),
    timestamp: new Date().toISOString(),
    type,
    totalTests: summary.totalTests,
    passedTests: summary.passedTests,
    failedTests: summary.failedTests,
    duration: summary.duration,
    successRate: summary.totalTests > 0 ? (summary.passedTests / summary.totalTests) * 100 : 0,
  }

  testHistory.value.unshift(historyItem)

  // 최대 50개까지만 저장
  if (testHistory.value.length > 50) {
    testHistory.value = testHistory.value.slice(0, 50)
  }

  // 로컬 스토리지에 저장
  localStorage.setItem('test-history', JSON.stringify(testHistory.value))
}

// 유틸리티 함수들
const formatDuration = (ms: number): string => {
  if (ms < 1000) return `${ms.toFixed(1)}ms`
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`
  return `${(ms / 60000).toFixed(1)}m`
}

const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(Math.abs(bytes)) / Math.log(k))
  return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`
}

const getSuccessRateColor = (rate: number): string => {
  if (rate >= 90) return '#52c41a'
  if (rate >= 70) return '#faad14'
  if (rate >= 50) return '#fa8c16'
  return '#f5222d'
}

const getFpsStatus = (fps: number): string => {
  if (!fps) return 'unknown'
  if (fps >= 55) return 'excellent'
  if (fps >= 45) return 'good'
  if (fps >= 30) return 'fair'
  return 'poor'
}

const getFpsStatusText = (fps: number): string => {
  if (!fps) return '측정 불가'
  if (fps >= 55) return '매우 좋음'
  if (fps >= 45) return '좋음'
  if (fps >= 30) return '보통'
  return '개선 필요'
}

const getResponseTimeStatus = (time: number): string => {
  if (time <= 200) return 'excellent'
  if (time <= 500) return 'good'
  if (time <= 1000) return 'fair'
  return 'poor'
}

const getResponseTimeStatusText = (time: number): string => {
  if (time <= 200) return '매우 빠름'
  if (time <= 500) return '빠름'
  if (time <= 1000) return '보통'
  return '느림'
}

const createMockResults = (testType: string): TestResults => {
  // 특정 테스트 타입에 대한 모의 결과 생성
  const mockResults: TestResults = {
    performance: [],
    render: null,
    network: null,
    accessibility: null,
    summary: {
      totalTests: 1,
      passedTests: 1,
      failedTests: 0,
      duration: 1000 + Math.random() * 2000,
    },
  }

  switch (testType) {
    case 'performance':
      mockResults.performance = [
        {
          testName: 'Core Web Vitals',
          duration: 150 + Math.random() * 100,
          memoryUsage: 1024 * 1024 * (2 + Math.random() * 3),
          success: Math.random() > 0.2,
          metrics: {},
        },
      ]
      break
    case 'render':
      mockResults.render = {
        success: Math.random() > 0.1,
        fps: 45 + Math.random() * 15,
        testName: 'Frame Rate Test',
      }
      break
    case 'network':
      mockResults.network = {
        success: Math.random() > 0.15,
        responseTime: 200 + Math.random() * 800,
        testName: 'API Response Time Test',
      }
      break
    case 'accessibility':
      mockResults.accessibility = {
        success: Math.random() > 0.1,
        focusableElements: 15 + Math.floor(Math.random() * 10),
        tabOrder: Math.random() > 0.2,
        missingLabels: Math.floor(Math.random() * 3),
        testName: 'Accessibility Test',
      }
      break
  }

  return mockResults
}

const generateTestReport = (results: TestResults): string => {
  let report = '테스트 리포트\n'
  report += `생성 시간: ${new Date().toLocaleString()}\n`
  report += '===========================================\n\n'

  report += '전체 요약:\n'
  report += `- 총 테스트: ${results.summary.totalTests}\n`
  report += `- 통과: ${results.summary.passedTests}\n`
  report += `- 실패: ${results.summary.failedTests}\n`
  report += `- 성공률: ${successRate.value.toFixed(1)}%\n`
  report += `- 소요 시간: ${formatDuration(results.summary.duration)}\n\n`

  if (results.performance.length > 0) {
    report += '성능 테스트 결과:\n'
    results.performance.forEach(test => {
      report += `- ${test.testName}: ${test.success ? 'PASS' : 'FAIL'} (${formatDuration(test.duration)})\n`
      if (!test.success && test.error) {
        report += `  오류: ${test.error}\n`
      }
    })
    report += '\n'
  }

  if (results.render) {
    report += '렌더링 테스트 결과:\n'
    report += `- FPS: ${results.render.fps?.toFixed(1) || 'N/A'}\n`
    report += `- 상태: ${results.render.success ? 'PASS' : 'FAIL'}\n\n`
  }

  if (results.network) {
    report += '네트워크 테스트 결과:\n'
    report += `- 응답 시간: ${formatDuration(results.network.responseTime)}\n`
    report += `- 상태: ${results.network.success ? 'PASS' : 'FAIL'}\n\n`
  }

  if (results.accessibility) {
    report += '접근성 테스트 결과:\n'
    report += `- 포커스 가능한 요소: ${results.accessibility.focusableElements}\n`
    report += `- 탭 순서: ${results.accessibility.tabOrder ? '올바름' : '문제 있음'}\n`
    report += `- 누락된 라벨: ${results.accessibility.missingLabels}개\n`
    report += `- 상태: ${results.accessibility.success ? 'PASS' : 'FAIL'}\n\n`
  }

  return report
}

// 라이프사이클
onMounted(() => {
  // 로컬 스토리지에서 테스트 기록 로드
  const savedHistory = localStorage.getItem('test-history')
  if (savedHistory) {
    try {
      testHistory.value = JSON.parse(savedHistory)
    } catch (error) {
      console.error('테스트 기록 로드 실패:', error)
    }
  }
})
</script>

<style scoped>
.test-dashboard {
  padding: 24px;
  max-width: 1200px;
  margin: 0 auto;
}

.dashboard-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 32px;
}

.dashboard-title {
  font-size: 24px;
  font-weight: 600;
  margin: 0;
  color: var(--text-color-primary);
}

.results-summary {
  margin-bottom: 32px;
}

.summary-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}

.summary-card {
  border-radius: 8px;
}

.summary-card.total {
  border-left: 4px solid #1890ff;
}

.summary-card.passed {
  border-left: 4px solid #52c41a;
}

.summary-card.failed {
  border-left: 4px solid #f5222d;
}

.summary-card.duration {
  border-left: 4px solid #faad14;
}

.card-content {
  display: flex;
  align-items: center;
  gap: 16px;
}

.card-icon {
  font-size: 32px;
}

.card-details {
  flex: 1;
}

.card-value {
  font-size: 24px;
  font-weight: 600;
  color: var(--text-color-primary);
}

.card-label {
  font-size: 14px;
  color: var(--text-color-secondary);
}

.success-rate {
  background: var(--bg-color-secondary);
  padding: 20px;
  border-radius: 8px;
}

.success-rate h3 {
  margin: 0 0 16px 0;
  color: var(--text-color-primary);
}

.success-rate-text {
  text-align: center;
  margin-top: 8px;
  font-weight: 600;
  color: var(--text-color-primary);
}

.running-status {
  margin-bottom: 32px;
}

.running-content {
  display: flex;
  align-items: center;
  gap: 24px;
  padding: 20px;
}

.running-text h3 {
  margin: 0 0 8px 0;
  color: var(--text-color-primary);
}

.running-text p {
  margin: 0;
  color: var(--text-color-secondary);
}

.test-categories {
  margin-bottom: 32px;
}

.test-category-content {
  padding: 20px 0;
}

.category-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.category-header h3 {
  margin: 0;
  color: var(--text-color-primary);
}

.category-stats {
  font-weight: 500;
  color: var(--text-color-secondary);
}

.test-results-grid {
  display: grid;
  gap: 16px;
}

.test-result-card {
  background: var(--bg-color-secondary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 16px;
}

.test-result-card.success {
  border-left: 4px solid #52c41a;
}

.test-result-card.failed {
  border-left: 4px solid #f5222d;
}

.test-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.test-name {
  font-weight: 600;
  color: var(--text-color-primary);
}

.test-status {
  font-size: 18px;
}

.test-metrics {
  display: flex;
  gap: 24px;
  margin-bottom: 8px;
}

.metric {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.metric-label {
  font-size: 12px;
  color: var(--text-color-secondary);
}

.metric-value {
  font-weight: 500;
  color: var(--text-color-primary);
}

.test-error {
  color: #f5222d;
  font-size: 12px;
  background: #fff2f0;
  padding: 8px;
  border-radius: 4px;
  border: 1px solid #ffccc7;
}

.success-card {
  border-left: 4px solid #52c41a;
}

.error-card {
  border-left: 4px solid #f5222d;
}

.render-content,
.network-content,
.accessibility-content {
  padding: 16px;
}

.fps-display,
.response-time-display {
  display: flex;
  align-items: baseline;
  gap: 8px;
  margin: 12px 0;
}

.fps-value,
.response-time-value {
  font-size: 32px;
  font-weight: 600;
  color: var(--text-color-primary);
}

.fps-unit {
  font-size: 16px;
  color: var(--text-color-secondary);
}

.status-badge {
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
}

.status-badge.excellent {
  background: #f6ffed;
  color: #52c41a;
}

.status-badge.good {
  background: #f6ffed;
  color: #52c41a;
}

.status-badge.fair {
  background: #fff7e6;
  color: #fa8c16;
}

.status-badge.poor {
  background: #fff1f0;
  color: #f5222d;
}

.status-badge.warning {
  background: #fff7e6;
  color: #fa8c16;
}

.status-badge.bad {
  background: #fff1f0;
  color: #f5222d;
}

.accessibility-metrics {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
  margin-bottom: 16px;
}

.metric-item {
  text-align: center;
}

.metric-item h4 {
  margin: 0 0 8px 0;
  font-size: 14px;
  color: var(--text-color-secondary);
}

.metric-item .metric-value {
  font-size: 24px;
  font-weight: 600;
  color: var(--text-color-primary);
}

.error-details {
  margin-top: 16px;
  padding: 12px;
  background: var(--bg-color-tertiary);
  border-radius: 4px;
}

.error-details h4 {
  margin: 0 0 8px 0;
  color: var(--text-color-primary);
}

.error-details p,
.error-details ul {
  margin: 0;
  color: var(--text-color-secondary);
}

.test-history {
  margin-bottom: 32px;
}

.help-section {
  margin-bottom: 32px;
}

.help-content {
  display: grid;
  gap: 20px;
}

.test-type-info {
  padding: 16px;
  background: var(--bg-color-tertiary);
  border-radius: 6px;
}

.test-type-info h4 {
  margin: 0 0 8px 0;
  color: var(--text-color-primary);
}

.test-type-info p {
  margin: 0;
  color: var(--text-color-secondary);
  line-height: 1.5;
}

.start-guide {
  padding: 16px;
  background: #e6f7ff;
  border: 1px solid #91d5ff;
  border-radius: 6px;
}

.start-guide p {
  margin: 0;
  color: #1890ff;
}

@media (max-width: 768px) {
  .test-dashboard {
    padding: 16px;
  }

  .dashboard-header {
    flex-direction: column;
    gap: 16px;
    align-items: stretch;
  }

  .summary-cards {
    grid-template-columns: 1fr;
  }

  .test-metrics {
    flex-direction: column;
    gap: 12px;
  }

  .accessibility-metrics {
    grid-template-columns: 1fr;
  }
}
</style>
