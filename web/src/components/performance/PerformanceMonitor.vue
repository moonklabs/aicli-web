<template>
  <div class="performance-monitor">
    <!-- 헤더 -->
    <div class="monitor-header">
      <h2>성능 모니터링</h2>
      <div class="monitor-controls">
        <button
          :class="['monitor-toggle', { active: isMonitoring }]"
          @click="toggleMonitoring"
          :aria-label="isMonitoring ? '모니터링 중지' : '모니터링 시작'"
        >
          {{ isMonitoring ? '중지' : '시작' }}
        </button>
        <button class="refresh-btn" @click="collectMetrics" :disabled="isMonitoring">
          새로고침
        </button>
      </div>
    </div>

    <!-- 성능 점수 요약 -->
    <div class="performance-score-card">
      <div class="score-circle" :class="performanceStatus">
        <span class="score-value">{{ performanceScore || '?' }}</span>
        <span class="score-label">점</span>
      </div>
      <div class="score-status">
        <h3>{{ getStatusLabel(performanceStatus) }}</h3>
        <p>{{ getStatusDescription(performanceStatus) }}</p>
      </div>
    </div>

    <!-- 메트릭 그리드 -->
    <div class="metrics-grid">
      <!-- Core Web Vitals -->
      <div class="metric-section">
        <h3>Core Web Vitals</h3>
        <div class="metric-cards">
          <MetricCard
            title="LCP"
            description="Largest Contentful Paint"
            :value="currentMetrics?.lcp"
            :target="2500"
            unit="ms"
            :status="getMetricStatus(currentMetrics?.lcp, 2500)"
          />
          <MetricCard
            title="FID"
            description="First Input Delay"
            :value="currentMetrics?.fid"
            :target="100"
            unit="ms"
            :status="getMetricStatus(currentMetrics?.fid, 100)"
          />
          <MetricCard
            title="CLS"
            description="Cumulative Layout Shift"
            :value="currentMetrics?.cls"
            :target="0.1"
            unit=""
            :status="getMetricStatus(currentMetrics?.cls, 0.1)"
          />
        </div>
      </div>

      <!-- 로딩 성능 -->
      <div class="metric-section">
        <h3>로딩 성능</h3>
        <div class="metric-cards">
          <MetricCard
            title="FCP"
            description="First Contentful Paint"
            :value="currentMetrics?.fcp"
            :target="1500"
            unit="ms"
            :status="getMetricStatus(currentMetrics?.fcp, 1500)"
          />
          <MetricCard
            title="TTI"
            description="Time to Interactive"
            :value="currentMetrics?.tti"
            :target="3000"
            unit="ms"
            :status="getMetricStatus(currentMetrics?.tti, 3000)"
          />
          <MetricCard
            title="TTFB"
            description="Time to First Byte"
            :value="currentMetrics?.ttfb"
            :target="800"
            unit="ms"
            :status="getMetricStatus(currentMetrics?.ttfb, 800)"
          />
        </div>
      </div>

      <!-- 리소스 정보 -->
      <div class="metric-section">
        <h3>리소스</h3>
        <div class="metric-cards">
          <MetricCard
            title="번들 크기"
            description="JavaScript Bundle Size"
            :value="currentMetrics?.bundleSize"
            :target="1024 * 1024"
            unit="KB"
            :status="getMetricStatus(currentMetrics?.bundleSize, 1024 * 1024)"
          />
          <MetricCard
            title="리소스 수"
            description="총 리소스 개수"
            :value="currentMetrics?.resourceCount"
            :target="50"
            unit="개"
            :status="getMetricStatus(currentMetrics?.resourceCount, 50)"
          />
          <MetricCard
            title="메모리"
            description="JavaScript 힙 사용량"
            :value="currentMetrics?.memoryUsage"
            :target="50 * 1024 * 1024"
            unit="MB"
            :status="getMetricStatus(currentMetrics?.memoryUsage, 50 * 1024 * 1024)"
          />
        </div>
      </div>
    </div>

    <!-- 성능 히스토리 (간단한 차트) -->
    <div class="performance-history" v-if="performanceHistory.length > 1">
      <h3>성능 히스토리</h3>
      <div class="history-chart">
        <div class="chart-placeholder">
          <!-- 실제 구현에서는 Chart.js나 다른 차트 라이브러리 사용 -->
          <p>성능 데이터: {{ performanceHistory.length }}개 기록</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { usePerformanceMonitoring } from '@/composables/usePerformanceMonitoring'
import MetricCard from './MetricCard.vue'

// 성능 모니터링 컴포저블
const {
  isMonitoring,
  currentMetrics,
  performanceHistory,
  performanceScore,
  performanceStatus,
  startMonitoring,
  stopMonitoring,
  collectMetrics,
} = usePerformanceMonitoring()

// 모니터링 토글
const toggleMonitoring = () => {
  if (isMonitoring.value) {
    stopMonitoring()
  } else {
    startMonitoring()
  }
}

// 성능 상태 라벨
const getStatusLabel = (status: string) => {
  const labels = {
    excellent: '우수',
    good: '양호',
    'needs-improvement': '개선 필요',
    poor: '불량',
    unknown: '측정 중'
  }
  return labels[status as keyof typeof labels] || '알 수 없음'
}

// 성능 상태 설명
const getStatusDescription = (status: string) => {
  const descriptions = {
    excellent: '모든 성능 지표가 우수한 수준입니다',
    good: '대부분의 성능 지표가 양호한 수준입니다',
    'needs-improvement': '일부 성능 지표가 개선이 필요합니다',
    poor: '성능 지표가 전반적으로 불량합니다',
    unknown: '성능 데이터를 수집하고 있습니다'
  }
  return descriptions[status as keyof typeof descriptions] || ''
}

// 메트릭 상태 계산
const getMetricStatus = (value: number | null, target: number): 'good' | 'warning' | 'error' | 'unknown' => {
  if (value === null) return 'unknown'
  
  // CLS는 작을수록 좋음
  if (target <= 1) {
    if (value <= target * 0.5) return 'good'
    if (value <= target) return 'warning'
    return 'error'
  }
  
  // 시간 기반 메트릭은 작을수록 좋음
  if (value <= target * 0.7) return 'good'
  if (value <= target) return 'warning'
  return 'error'
}
</script>

<style scoped lang="scss">
.performance-monitor {
  padding: var(--spacing-lg);
  background: var(--bg-secondary);
  border-radius: var(--radius-xl);
  border: 1px solid var(--border-primary);
}

.monitor-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--spacing-xl);

  h2 {
    margin: 0;
    color: var(--text-primary);
    font-size: var(--text-xl);
    font-weight: var(--font-semibold);
  }
}

.monitor-controls {
  display: flex;
  gap: var(--spacing-sm);
}

.monitor-toggle {
  padding: var(--spacing-sm) var(--spacing-md);
  background: var(--bg-primary);
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-md);
  color: var(--text-primary);
  cursor: pointer;
  transition: var(--duration-normal) var(--ease-in-out);

  &:hover {
    background: var(--state-hover);
  }

  &.active {
    background: var(--primary-500);
    color: white;
    border-color: var(--primary-500);
  }
}

.refresh-btn {
  padding: var(--spacing-sm) var(--spacing-md);
  background: var(--bg-tertiary);
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-md);
  color: var(--text-secondary);
  cursor: pointer;
  transition: var(--duration-normal) var(--ease-in-out);

  &:hover:not(:disabled) {
    background: var(--state-hover);
  }

  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
}

.performance-score-card {
  display: flex;
  align-items: center;
  gap: var(--spacing-lg);
  padding: var(--spacing-lg);
  background: var(--bg-elevated);
  border-radius: var(--radius-lg);
  margin-bottom: var(--spacing-xl);
  border: 1px solid var(--border-primary);
}

.score-circle {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  width: 80px;
  height: 80px;
  border-radius: 50%;
  border: 4px solid;
  transition: var(--duration-normal) var(--ease-in-out);

  &.excellent {
    border-color: var(--success-500);
    background: var(--success-50);
    color: var(--success-700);
  }

  &.good {
    border-color: var(--primary-500);
    background: var(--primary-50);
    color: var(--primary-700);
  }

  &.needs-improvement {
    border-color: var(--warning-500);
    background: var(--warning-50);
    color: var(--warning-700);
  }

  &.poor {
    border-color: var(--error-500);
    background: var(--error-50);
    color: var(--error-700);
  }

  &.unknown {
    border-color: var(--gray-300);
    background: var(--gray-50);
    color: var(--gray-600);
  }
}

.score-value {
  font-size: var(--text-lg);
  font-weight: var(--font-bold);
  line-height: 1;
}

.score-label {
  font-size: var(--text-xs);
  font-weight: var(--font-medium);
}

.score-status {
  h3 {
    margin: 0 0 var(--spacing-xs) 0;
    color: var(--text-primary);
    font-size: var(--text-lg);
    font-weight: var(--font-semibold);
  }

  p {
    margin: 0;
    color: var(--text-secondary);
    font-size: var(--text-sm);
  }
}

.metrics-grid {
  display: grid;
  gap: var(--spacing-xl);
}

.metric-section {
  h3 {
    margin: 0 0 var(--spacing-md) 0;
    color: var(--text-primary);
    font-size: var(--text-base);
    font-weight: var(--font-semibold);
  }
}

.metric-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: var(--spacing-md);
}

.performance-history {
  margin-top: var(--spacing-xl);
  padding-top: var(--spacing-xl);
  border-top: 1px solid var(--border-primary);

  h3 {
    margin: 0 0 var(--spacing-md) 0;
    color: var(--text-primary);
    font-size: var(--text-base);
    font-weight: var(--font-semibold);
  }
}

.history-chart {
  height: 200px;
  background: var(--bg-elevated);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-primary);
  display: flex;
  align-items: center;
  justify-content: center;
}

.chart-placeholder {
  text-align: center;
  color: var(--text-secondary);
  font-size: var(--text-sm);
}

// 모바일 대응
@media (max-width: 768px) {
  .performance-monitor {
    padding: var(--spacing-md);
  }

  .monitor-header {
    flex-direction: column;
    gap: var(--spacing-md);
    align-items: stretch;
  }

  .performance-score-card {
    flex-direction: column;
    text-align: center;
    gap: var(--spacing-md);
  }

  .metric-cards {
    grid-template-columns: 1fr;
  }
}
</style>