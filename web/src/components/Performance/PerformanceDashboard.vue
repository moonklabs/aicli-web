<template>
  <div class="performance-dashboard">
    <!-- 대시보드 헤더 -->
    <div class="dashboard-header">
      <h2>성능 모니터링 대시보드</h2>
      <div class="dashboard-controls">
        <NButton
          v-if="!isMonitoring"
          @click="startMonitoring"
          type="primary"
          :loading="false"
        >
          모니터링 시작
        </NButton>
        <NButton
          v-else
          @click="stopMonitoring"
          type="error"
        >
          모니터링 중지
        </NButton>
        <NButton @click="resetMetrics" :disabled="!isReady">
          리셋
        </NButton>
        <NButton @click="exportMetrics" :disabled="!isReady">
          내보내기
        </NButton>
      </div>
    </div>

    <!-- 성능 점수 카드 -->
    <div class="performance-overview">
      <div class="score-card">
        <div class="score-circle" :class="`grade-${performanceGrade.toLowerCase()}`">
          <span class="score-value">{{ performanceScore }}</span>
          <span class="score-grade">{{ performanceGrade }}</span>
        </div>
        <div class="score-info">
          <h3>전체 성능 점수</h3>
          <p class="trend-indicator" :class="`trend-${performanceTrend}`">
            <Icon :name="getTrendIcon(performanceTrend)" />
            {{ getTrendText(performanceTrend) }}
          </p>
        </div>
      </div>

      <!-- 실시간 상태 -->
      <div class="status-cards">
        <div class="status-card">
          <Icon name="activity" />
          <div>
            <div class="status-label">모니터링 상태</div>
            <div class="status-value" :class="{ active: isMonitoring }">
              {{ isMonitoring ? '활성' : '비활성' }}
            </div>
          </div>
        </div>

        <div class="status-card">
          <Icon name="clock" />
          <div>
            <div class="status-label">마지막 업데이트</div>
            <div class="status-value">{{ formatTime(state.currentMetrics.timestamp) }}</div>
          </div>
        </div>

        <div class="status-card">
          <Icon name="smartphone" />
          <div>
            <div class="status-label">디바이스</div>
            <div class="status-value">{{ state.currentMetrics.deviceType || '-' }}</div>
          </div>
        </div>
      </div>
    </div>

    <!-- Core Web Vitals -->
    <div class="web-vitals-section">
      <h3>Core Web Vitals</h3>
      <div class="vitals-grid">
        <div
          v-for="(vital, key) in webVitalsStatus"
          :key="key"
          class="vital-card"
          :class="`status-${vital.status}`"
        >
          <div class="vital-header">
            <span class="vital-name">{{ getVitalName(key) }}</span>
            <Icon :name="getVitalIcon(vital.status)" />
          </div>
          <div class="vital-value">
            {{ formatVitalValue(key, vital.value) }}
          </div>
          <div class="vital-description">
            {{ getVitalDescription(key) }}
          </div>
          <div class="vital-threshold">
            임계값: {{ getVitalThreshold(key) }}
          </div>
        </div>
      </div>
    </div>

    <!-- 성능 차트 -->
    <div class="charts-section">
      <div class="chart-container">
        <h3>성능 트렌드</h3>
        <LineChart
          :data="performanceTrendData"
          :options="chartOptions"
          height="300"
        />
      </div>

      <div class="chart-container">
        <h3>메모리 사용량</h3>
        <LineChart
          :data="memoryUsageData"
          :options="memoryChartOptions"
          height="200"
        />
      </div>
    </div>

    <!-- 성능 메트릭 테이블 -->
    <div class="metrics-table-section">
      <h3>상세 메트릭</h3>
      <BaseDataTable
        :data="metricsTableData"
        :columns="metricsColumns"
        :loading="!isReady"
        :pagination="{ pageSize: 10 }"
      />
    </div>

    <!-- 권장사항 -->
    <div v-if="recommendations.length > 0" class="recommendations-section">
      <h3>성능 개선 권장사항</h3>
      <div class="recommendations-list">
        <div
          v-for="(recommendation, index) in recommendations"
          :key="index"
          class="recommendation-item"
        >
          <Icon name="lightbulb" />
          <span>{{ recommendation }}</span>
        </div>
      </div>
    </div>

    <!-- 오류 알림 -->
    <div v-if="hasErrors" class="errors-section">
      <NAlert type="error" title="모니터링 오류">
        <div v-for="(error, index) in state.errors" :key="index">
          {{ error.message }}
        </div>
      </NAlert>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'
import { NAlert, NButton } from 'naive-ui'
import { usePerformanceMonitoring } from '@/composables/usePerformanceMonitoring'
import LineChart from '@/components/ui/charts/LineChart.vue'
import BaseDataTable from '@/components/ui/data/BaseDataTable.vue'
import Icon from '@/components/common/Icon.vue'

// 성능 모니터링 composable 사용
const {
  state,
  performanceScore,
  performanceGrade,
  webVitalsStatus,
  performanceTrend,
  startMonitoring,
  stopMonitoring,
  resetMetrics,
  generateReport,
  exportMetrics,
  isMonitoring,
  hasErrors,
  isReady,
} = usePerformanceMonitoring()

// 권장사항 계산
const recommendations = computed(() => {
  const vitals = webVitalsStatus.value
  const recs: string[] = []

  if (vitals.lcp.status === 'poor' || vitals.lcp.status === 'needs-improvement') {
    recs.push('LCP 개선: 이미지 최적화 및 서버 응답 시간 단축')
  }

  if (vitals.fid.status === 'poor' || vitals.fid.status === 'needs-improvement') {
    recs.push('FID 개선: JavaScript 실행 최적화 및 코드 스플리팅')
  }

  if (vitals.cls.status === 'poor' || vitals.cls.status === 'needs-improvement') {
    recs.push('CLS 개선: 레이아웃 안정성 향상 및 리소스 크기 지정')
  }

  return recs
})

// 성능 트렌드 차트 데이터
const performanceTrendData = computed(() => {
  const data = state.historicalData.slice(-20) // 최근 20개

  return {
    labels: data.map((_, index) => `${index + 1}`),
    datasets: [
      {
        label: 'LCP (ms)',
        data: data.map(d => d.LCP || 0),
        borderColor: '#f56565',
        backgroundColor: 'rgba(245, 101, 101, 0.1)',
        fill: true,
      },
      {
        label: 'FID (ms)',
        data: data.map(d => d.FID || 0),
        borderColor: '#48bb78',
        backgroundColor: 'rgba(72, 187, 120, 0.1)',
        fill: true,
      },
      {
        label: 'CLS (×100)',
        data: data.map(d => (d.CLS || 0) * 100),
        borderColor: '#ed8936',
        backgroundColor: 'rgba(237, 137, 54, 0.1)',
        fill: true,
      },
    ],
  }
})

// 메모리 사용량 차트 데이터
const memoryUsageData = computed(() => {
  const data = state.historicalData.slice(-10)

  return {
    labels: data.map((_, index) => `${index + 1}`),
    datasets: [
      {
        label: '메모리 사용량 (MB)',
        data: data.map(d => d.memoryUsage || 0),
        borderColor: '#4299e1',
        backgroundColor: 'rgba(66, 153, 225, 0.1)',
        fill: true,
      },
    ],
  }
})

// 차트 옵션
const chartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  scales: {
    y: {
      beginAtZero: true,
      title: {
        display: true,
        text: '시간 (ms)',
      },
    },
    x: {
      title: {
        display: true,
        text: '측정 순서',
      },
    },
  },
  plugins: {
    legend: {
      position: 'top' as const,
    },
    tooltip: {
      mode: 'index' as const,
      intersect: false,
    },
  },
}

const memoryChartOptions = {
  ...chartOptions,
  scales: {
    ...chartOptions.scales,
    y: {
      ...chartOptions.scales.y,
      title: {
        display: true,
        text: '메모리 (MB)',
      },
    },
  },
}

// 메트릭 테이블 데이터
const metricsTableData = computed(() => {
  const metrics = state.currentMetrics
  return [
    {
      metric: 'Largest Contentful Paint',
      value: metrics.LCP ? `${metrics.LCP.toFixed(2)}ms` : '-',
      status: webVitalsStatus.value.lcp.status,
      threshold: '≤ 2.5s',
    },
    {
      metric: 'First Input Delay',
      value: metrics.FID ? `${metrics.FID.toFixed(2)}ms` : '-',
      status: webVitalsStatus.value.fid.status,
      threshold: '≤ 100ms',
    },
    {
      metric: 'Cumulative Layout Shift',
      value: metrics.CLS ? metrics.CLS.toFixed(4) : '-',
      status: webVitalsStatus.value.cls.status,
      threshold: '≤ 0.1',
    },
    {
      metric: 'First Contentful Paint',
      value: metrics.FCP ? `${metrics.FCP.toFixed(2)}ms` : '-',
      status: webVitalsStatus.value.fcp.status,
      threshold: '≤ 1.8s',
    },
    {
      metric: 'Time to First Byte',
      value: metrics.TTFB ? `${metrics.TTFB.toFixed(2)}ms` : '-',
      status: webVitalsStatus.value.ttfb.status,
      threshold: '≤ 800ms',
    },
    {
      metric: 'Memory Usage',
      value: metrics.memoryUsage ? `${metrics.memoryUsage.toFixed(2)}MB` : '-',
      status: (metrics.memoryUsage || 0) < 50 ? 'good' : 'needs-improvement',
      threshold: '< 50MB',
    },
  ]
})

const metricsColumns = [
  { key: 'metric', title: '메트릭', width: 200 },
  { key: 'value', title: '현재 값', width: 100 },
  {
    key: 'status',
    title: '상태',
    width: 100,
    render: (row: any) => {
      const statusColors = {
        good: 'success',
        'needs-improvement': 'warning',
        poor: 'error',
        pending: 'info',
      }
      return `<span class="status-badge status-${row.status}">${getStatusText(row.status)}</span>`
    },
  },
  { key: 'threshold', title: '임계값', width: 100 },
]

// 헬퍼 함수들
function getVitalName(key: string): string {
  const names: Record<string, string> = {
    lcp: 'LCP',
    fid: 'FID',
    cls: 'CLS',
    fcp: 'FCP',
    ttfb: 'TTFB',
  }
  return names[key] || key.toUpperCase()
}

function getVitalDescription(key: string): string {
  const descriptions: Record<string, string> = {
    lcp: 'Largest Contentful Paint',
    fid: 'First Input Delay',
    cls: 'Cumulative Layout Shift',
    fcp: 'First Contentful Paint',
    ttfb: 'Time to First Byte',
  }
  return descriptions[key] || ''
}

function getVitalThreshold(key: string): string {
  const thresholds: Record<string, string> = {
    lcp: '≤ 2.5s',
    fid: '≤ 100ms',
    cls: '≤ 0.1',
    fcp: '≤ 1.8s',
    ttfb: '≤ 800ms',
  }
  return thresholds[key] || ''
}

function formatVitalValue(key: string, value: number): string {
  if (!value) return '-'

  if (key === 'cls') {
    return value.toFixed(4)
  }

  return `${value.toFixed(2)}ms`
}

function getVitalIcon(status: string): string {
  const icons: Record<string, string> = {
    good: 'checkmark-circle',
    'needs-improvement': 'warning',
    poor: 'close-circle',
    pending: 'time',
  }
  return icons[status] || 'help'
}

function getTrendIcon(trend: string): string {
  const icons: Record<string, string> = {
    improving: 'trending-up',
    declining: 'trending-down',
    stable: 'remove',
  }
  return icons[trend] || 'remove'
}

function getTrendText(trend: string): string {
  const texts: Record<string, string> = {
    improving: '개선 중',
    declining: '저하 중',
    stable: '안정',
  }
  return texts[trend] || '알 수 없음'
}

function getStatusText(status: string): string {
  const texts: Record<string, string> = {
    good: '양호',
    'needs-improvement': '개선 필요',
    poor: '나쁨',
    pending: '대기 중',
  }
  return texts[status] || status
}

function formatTime(timestamp?: number): string {
  if (!timestamp) return '-'

  const date = new Date(timestamp)
  return date.toLocaleTimeString('ko-KR')
}

// 자동 새로고침
watch(
  () => state.currentMetrics.timestamp,
  () => {
    // 메트릭이 업데이트될 때마다 차트 데이터 새로고침
  },
)
</script>

<style lang="scss" scoped>
.performance-dashboard {
  padding: 20px;

  .dashboard-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 30px;

    h2 {
      margin: 0;
      color: var(--text-color-primary);
    }

    .dashboard-controls {
      display: flex;
      gap: 10px;
    }
  }

  .performance-overview {
    display: grid;
    grid-template-columns: auto 1fr;
    gap: 30px;
    margin-bottom: 40px;

    .score-card {
      display: flex;
      align-items: center;
      gap: 20px;

      .score-circle {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        width: 120px;
        height: 120px;
        border-radius: 50%;
        position: relative;

        &.grade-a { background: linear-gradient(45deg, #48bb78, #68d391); }
        &.grade-b { background: linear-gradient(45deg, #4299e1, #63b3ed); }
        &.grade-c { background: linear-gradient(45deg, #ed8936, #f6ad55); }
        &.grade-d { background: linear-gradient(45deg, #f56565, #fc8181); }
        &.grade-f { background: linear-gradient(45deg, #e53e3e, #f56565); }

        .score-value {
          font-size: 28px;
          font-weight: bold;
          color: white;
        }

        .score-grade {
          font-size: 14px;
          color: white;
          opacity: 0.9;
        }
      }

      .score-info {
        h3 {
          margin: 0 0 10px 0;
          color: var(--text-color-primary);
        }

        .trend-indicator {
          display: flex;
          align-items: center;
          gap: 5px;
          font-size: 14px;

          &.trend-improving { color: #48bb78; }
          &.trend-declining { color: #f56565; }
          &.trend-stable { color: #718096; }
        }
      }
    }

    .status-cards {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
      gap: 20px;

      .status-card {
        display: flex;
        align-items: center;
        gap: 15px;
        padding: 20px;
        background: var(--bg-color-secondary);
        border-radius: 8px;
        border: 1px solid var(--border-color);

        .status-label {
          font-size: 12px;
          color: var(--text-color-secondary);
          margin-bottom: 5px;
        }

        .status-value {
          font-size: 16px;
          font-weight: 600;
          color: var(--text-color-primary);

          &.active {
            color: #48bb78;
          }
        }
      }
    }
  }

  .web-vitals-section {
    margin-bottom: 40px;

    h3 {
      margin-bottom: 20px;
      color: var(--text-color-primary);
    }

    .vitals-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
      gap: 20px;

      .vital-card {
        padding: 20px;
        background: var(--bg-color-secondary);
        border-radius: 8px;
        border: 1px solid var(--border-color);
        transition: all 0.3s ease;

        &.status-good {
          border-left: 4px solid #48bb78;
        }

        &.status-needs-improvement {
          border-left: 4px solid #ed8936;
        }

        &.status-poor {
          border-left: 4px solid #f56565;
        }

        &.status-pending {
          border-left: 4px solid #718096;
        }

        .vital-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 10px;

          .vital-name {
            font-weight: 600;
            color: var(--text-color-primary);
          }
        }

        .vital-value {
          font-size: 24px;
          font-weight: bold;
          color: var(--text-color-primary);
          margin-bottom: 5px;
        }

        .vital-description {
          font-size: 12px;
          color: var(--text-color-secondary);
          margin-bottom: 10px;
        }

        .vital-threshold {
          font-size: 11px;
          color: var(--text-color-tertiary);
          padding: 4px 8px;
          background: var(--bg-color-tertiary);
          border-radius: 4px;
          display: inline-block;
        }
      }
    }
  }

  .charts-section {
    display: grid;
    grid-template-columns: 2fr 1fr;
    gap: 30px;
    margin-bottom: 40px;

    .chart-container {
      background: var(--bg-color-secondary);
      padding: 20px;
      border-radius: 8px;
      border: 1px solid var(--border-color);

      h3 {
        margin: 0 0 20px 0;
        color: var(--text-color-primary);
      }
    }
  }

  .metrics-table-section,
  .recommendations-section {
    margin-bottom: 30px;

    h3 {
      margin-bottom: 20px;
      color: var(--text-color-primary);
    }
  }

  .recommendations-list {
    .recommendation-item {
      display: flex;
      align-items: center;
      gap: 10px;
      padding: 12px 16px;
      background: var(--bg-color-secondary);
      border-radius: 6px;
      margin-bottom: 10px;
      border-left: 3px solid #4299e1;

      span {
        color: var(--text-color-primary);
      }
    }
  }

  .errors-section {
    margin-bottom: 20px;
  }
}

// 상태 배지 스타일
:global(.status-badge) {
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;

  &.status-good {
    background: #c6f6d5;
    color: #22543d;
  }

  &.status-needs-improvement {
    background: #feebc8;
    color: #c05621;
  }

  &.status-poor {
    background: #fed7d7;
    color: #c53030;
  }

  &.status-pending {
    background: #e2e8f0;
    color: #4a5568;
  }
}

// 반응형 디자인
@media (max-width: 768px) {
  .performance-dashboard {
    padding: 15px;

    .performance-overview {
      grid-template-columns: 1fr;
      gap: 20px;

      .score-card {
        flex-direction: column;
        text-align: center;
        gap: 15px;
      }
    }

    .charts-section {
      grid-template-columns: 1fr;
      gap: 20px;
    }

    .vitals-grid {
      grid-template-columns: 1fr;
    }
  }
}
</style>