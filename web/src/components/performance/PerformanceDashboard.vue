<template>
  <div class="performance-dashboard">
    <div class="dashboard-header">
      <h2>Performance Dashboard</h2>
      <div class="actions">
        <button 
          @click="refreshMetrics" 
          :disabled="isLoading"
          class="refresh-button"
        >
          <IconRefresh :class="{ rotating: isLoading }" />
          {{ isLoading ? '측정 중...' : '새로고침' }}
        </button>
        <button 
          @click="exportMetrics" 
          class="export-button"
          :disabled="!hasMetrics"
        >
          <IconDownload />
          내보내기
        </button>
      </div>
    </div>

    <div v-if="error" class="error-message">
      <IconWarning />
      {{ error }}
    </div>

    <div class="metrics-grid">
      <!-- Core Web Vitals -->
      <div class="metric-section core-vitals">
        <h3>Core Web Vitals</h3>
        <div class="metrics">
          <MetricCard
            title="LCP"
            subtitle="Largest Contentful Paint"
            :value="formatMetric(metrics.lcp, 'ms')"
            :status="getMetricStatus('lcp', metrics.lcp)"
            :threshold="thresholds.lcp"
            icon="paint"
          />
          <MetricCard
            title="FID"
            subtitle="First Input Delay"
            :value="formatMetric(metrics.fid, 'ms')"
            :status="getMetricStatus('fid', metrics.fid)"
            :threshold="thresholds.fid"
            icon="cursor"
          />
          <MetricCard
            title="CLS"
            subtitle="Cumulative Layout Shift"
            :value="formatMetric(metrics.cls)"
            :status="getMetricStatus('cls', metrics.cls)"
            :threshold="thresholds.cls"
            icon="layout"
          />
        </div>
      </div>

      <!-- Other Web Vitals -->
      <div class="metric-section other-vitals">
        <h3>Other Web Vitals</h3>
        <div class="metrics">
          <MetricCard
            title="FCP"
            subtitle="First Contentful Paint"
            :value="formatMetric(metrics.fcp, 'ms')"
            :status="getMetricStatus('fcp', metrics.fcp)"
            :threshold="thresholds.fcp"
            icon="eye"
          />
          <MetricCard
            title="INP"
            subtitle="Interaction to Next Paint"
            :value="formatMetric(metrics.inp, 'ms')"
            :status="getMetricStatus('inp', metrics.inp)"
            :threshold="thresholds.inp"
            icon="click"
          />
          <MetricCard
            title="TTFB"
            subtitle="Time to First Byte"
            :value="formatMetric(metrics.ttfb, 'ms')"
            :status="getMetricStatus('ttfb', metrics.ttfb)"
            :threshold="thresholds.ttfb"
            icon="server"
          />
        </div>
      </div>

      <!-- Score Overview -->
      <div class="metric-section score-overview">
        <h3>Performance Score</h3>
        <div class="score-display">
          <div class="score-circle" :class="getScoreClass(coreWebVitalsScore)">
            <svg viewBox="0 0 120 120">
              <circle
                cx="60"
                cy="60"
                r="54"
                fill="none"
                stroke="currentColor"
                stroke-width="12"
                opacity="0.2"
              />
              <circle
                cx="60"
                cy="60"
                r="54"
                fill="none"
                stroke="currentColor"
                stroke-width="12"
                :stroke-dasharray="`${scoreCircumference} ${scoreCircumference}`"
                :stroke-dashoffset="scoreOffset"
                transform="rotate(-90 60 60)"
              />
            </svg>
            <div class="score-value">{{ coreWebVitalsScore }}</div>
          </div>
          <div class="score-details">
            <p class="score-label">Core Web Vitals Score</p>
            <p class="score-description">
              {{ getScoreDescription(coreWebVitalsScore) }}
            </p>
          </div>
        </div>
      </div>

      <!-- Historical Trends -->
      <div class="metric-section trends">
        <h3>Performance Trends</h3>
        <div class="trend-charts">
          <TrendChart
            v-for="metric in ['lcp', 'fid', 'cls']"
            :key="metric"
            :metric="metric"
            :data="getTrendData(metric)"
            :threshold="thresholds[metric]"
          />
        </div>
      </div>
    </div>

    <!-- Recommendations -->
    <div v-if="recommendations.length > 0" class="recommendations">
      <h3>개선 권장사항</h3>
      <ul>
        <li v-for="(rec, index) in recommendations" :key="index">
          <span class="rec-icon">{{ rec.icon }}</span>
          <div>
            <strong>{{ rec.title }}</strong>
            <p>{{ rec.description }}</p>
          </div>
        </li>
      </ul>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useWebVitals } from '@/composables/useWebVitals'
import MetricCard from './MetricCard.vue'
import TrendChart from './TrendChart.vue'
import IconRefresh from '@/components/icons/IconRefresh.vue'
import IconDownload from '@/components/icons/IconDownload.vue'
import IconWarning from '@/components/icons/IconWarning.vue'

const { metrics, isSupported, getCoreWebVitalsScore, evaluateMetric, thresholds } = useWebVitals()

const isLoading = ref(false)
const error = ref('')
const historicalData = ref<any[]>([])

// Core Web Vitals 점수
const coreWebVitalsScore = computed(() => getCoreWebVitalsScore())

// SVG 원 계산
const scoreCircumference = 2 * Math.PI * 54
const scoreOffset = computed(() => {
  const progress = coreWebVitalsScore.value / 100
  return scoreCircumference * (1 - progress)
})

// 메트릭 포맷팅
const formatMetric = (value?: number, unit = '') => {
  if (value === undefined) return '-'
  if (unit === 'ms') {
    return value >= 1000 ? `${(value / 1000).toFixed(2)}s` : `${Math.round(value)}ms`
  }
  return value.toFixed(3)
}

// 메트릭 상태 평가
const getMetricStatus = (metric: string, value?: number) => {
  if (!value) return 'unknown'
  return evaluateMetric(value, thresholds[metric])
}

// 점수 클래스
const getScoreClass = (score: number) => {
  if (score >= 90) return 'good'
  if (score >= 50) return 'needs-improvement'
  return 'poor'
}

// 점수 설명
const getScoreDescription = (score: number) => {
  if (score >= 90) return '성능이 매우 우수합니다'
  if (score >= 50) return '개선이 필요한 부분이 있습니다'
  return '성능 개선이 시급합니다'
}

// 메트릭이 있는지 확인
const hasMetrics = computed(() => {
  return Object.values(metrics.value).some(v => v !== undefined)
})

// 트렌드 데이터 가져오기
const getTrendData = (metric: string) => {
  return historicalData.value
    .map(entry => ({
      timestamp: entry.timestamp,
      value: entry.metrics[metric],
    }))
    .filter(item => item.value !== undefined)
}

// 권장사항 생성
const recommendations = computed(() => {
  const recs = []
  
  if (metrics.value.lcp && metrics.value.lcp > thresholds.lcp.good) {
    recs.push({
      icon: '🎨',
      title: 'LCP 개선',
      description: '이미지 최적화, 중요 리소스 사전 로드, 서버 응답 시간 개선을 고려하세요.',
    })
  }
  
  if (metrics.value.fid && metrics.value.fid > thresholds.fid.good) {
    recs.push({
      icon: '⚡',
      title: 'FID 개선',
      description: '메인 스레드 작업 최적화, 코드 분할, Web Worker 사용을 고려하세요.',
    })
  }
  
  if (metrics.value.cls && metrics.value.cls > thresholds.cls.good) {
    recs.push({
      icon: '📐',
      title: 'CLS 개선',
      description: '이미지/광고에 크기 속성 추가, 동적 콘텐츠 로딩 최적화를 고려하세요.',
    })
  }
  
  if (metrics.value.ttfb && metrics.value.ttfb > thresholds.ttfb.good) {
    recs.push({
      icon: '🚀',
      title: 'TTFB 개선',
      description: 'CDN 사용, 서버 최적화, 데이터베이스 쿼리 개선을 고려하세요.',
    })
  }
  
  return recs
})

// 메트릭 새로고침
const refreshMetrics = async () => {
  isLoading.value = true
  error.value = ''
  
  try {
    // 페이지 새로고침하여 새 메트릭 수집
    window.location.reload()
  } catch (err) {
    error.value = '메트릭 수집 중 오류가 발생했습니다.'
  } finally {
    isLoading.value = false
  }
}

// 메트릭 내보내기
const exportMetrics = () => {
  const data = {
    timestamp: new Date().toISOString(),
    metrics: metrics.value,
    score: coreWebVitalsScore.value,
    url: window.location.href,
    userAgent: navigator.userAgent,
  }
  
  const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `web-vitals-${Date.now()}.json`
  a.click()
  URL.revokeObjectURL(url)
}

// 로컬 스토리지에서 이력 로드
const loadHistoricalData = () => {
  const stored = localStorage.getItem('webVitalsHistory')
  if (stored) {
    try {
      historicalData.value = JSON.parse(stored)
    } catch (e) {
      console.error('Failed to load historical data:', e)
    }
  }
}

// 현재 메트릭을 이력에 저장
const saveToHistory = () => {
  if (!hasMetrics.value) return
  
  const entry = {
    timestamp: Date.now(),
    metrics: { ...metrics.value },
    score: coreWebVitalsScore.value,
  }
  
  historicalData.value.push(entry)
  
  // 최근 100개만 유지
  if (historicalData.value.length > 100) {
    historicalData.value = historicalData.value.slice(-100)
  }
  
  localStorage.setItem('webVitalsHistory', JSON.stringify(historicalData.value))
}

// 주기적으로 메트릭 저장
let saveInterval: number

onMounted(() => {
  if (!isSupported.value) {
    error.value = 'Web Vitals API가 지원되지 않는 브라우저입니다.'
  }
  
  loadHistoricalData()
  
  // 30초마다 메트릭 저장
  saveInterval = window.setInterval(() => {
    saveToHistory()
  }, 30000)
})

onUnmounted(() => {
  if (saveInterval) {
    clearInterval(saveInterval)
  }
})
</script>

<style scoped>
.performance-dashboard {
  padding: var(--spacing-2xl);
  background: var(--bg-secondary);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-md);
}

.dashboard-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--spacing-2xl);
}

.dashboard-header h2 {
  font-size: var(--text-2xl);
  font-weight: var(--font-bold);
  color: var(--text-primary);
  margin: 0;
}

.actions {
  display: flex;
  gap: var(--spacing-md);
}

.refresh-button,
.export-button {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  padding: var(--spacing-sm) var(--spacing-lg);
  background: var(--bg-primary);
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-md);
  color: var(--text-primary);
  font-size: var(--text-sm);
  cursor: pointer;
  transition: all var(--duration-fast);
}

.refresh-button:hover,
.export-button:hover {
  background: var(--bg-tertiary);
  border-color: var(--border-secondary);
}

.refresh-button:disabled,
.export-button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.rotating {
  animation: rotate 1s linear infinite;
}

@keyframes rotate {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.error-message {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  padding: var(--spacing-lg);
  background: var(--error-50);
  color: var(--error-700);
  border-radius: var(--radius-md);
  margin-bottom: var(--spacing-xl);
}

.metrics-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: var(--spacing-2xl);
}

@media (min-width: 1200px) {
  .metrics-grid {
    grid-template-columns: 2fr 1fr;
  }
  
  .metric-section.trends {
    grid-column: 1 / -1;
  }
}

.metric-section {
  background: var(--bg-primary);
  padding: var(--spacing-xl);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-primary);
}

.metric-section h3 {
  font-size: var(--text-lg);
  font-weight: var(--font-semibold);
  color: var(--text-primary);
  margin: 0 0 var(--spacing-lg) 0;
}

.metrics {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: var(--spacing-lg);
}

.score-display {
  display: flex;
  align-items: center;
  gap: var(--spacing-xl);
}

.score-circle {
  position: relative;
  width: 120px;
  height: 120px;
  flex-shrink: 0;
}

.score-circle svg {
  width: 100%;
  height: 100%;
  transform: rotate(-90deg);
}

.score-circle.good {
  color: var(--success-500);
}

.score-circle.needs-improvement {
  color: var(--warning-500);
}

.score-circle.poor {
  color: var(--error-500);
}

.score-value {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  font-size: var(--text-3xl);
  font-weight: var(--font-bold);
  color: var(--text-primary);
}

.score-details {
  flex: 1;
}

.score-label {
  font-size: var(--text-lg);
  font-weight: var(--font-semibold);
  color: var(--text-primary);
  margin: 0 0 var(--spacing-sm) 0;
}

.score-description {
  font-size: var(--text-base);
  color: var(--text-secondary);
  margin: 0;
}

.trend-charts {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: var(--spacing-lg);
}

.recommendations {
  margin-top: var(--spacing-2xl);
  padding: var(--spacing-xl);
  background: var(--bg-primary);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-primary);
}

.recommendations h3 {
  font-size: var(--text-lg);
  font-weight: var(--font-semibold);
  color: var(--text-primary);
  margin: 0 0 var(--spacing-lg) 0;
}

.recommendations ul {
  list-style: none;
  padding: 0;
  margin: 0;
}

.recommendations li {
  display: flex;
  gap: var(--spacing-md);
  padding: var(--spacing-md) 0;
  border-bottom: 1px solid var(--border-primary);
}

.recommendations li:last-child {
  border-bottom: none;
}

.rec-icon {
  font-size: var(--text-xl);
  flex-shrink: 0;
}

.recommendations strong {
  display: block;
  font-weight: var(--font-semibold);
  color: var(--text-primary);
  margin-bottom: var(--spacing-xs);
}

.recommendations p {
  margin: 0;
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

/* 다크 모드 지원 */
[data-theme='dark'] .error-message {
  background: rgba(239, 68, 68, 0.1);
  color: var(--error-400);
}
</style>