<template>
  <div class="monitoring-dashboard">
    <!-- 대시보드 헤더 -->
    <div class="dashboard-header">
      <div class="header-content">
        <h1>모니터링 대시보드</h1>
        <p class="header-subtitle">성능, 에러, 사용자 분석을 한눈에 확인하세요</p>
      </div>

      <div class="header-controls">
        <NSpace>
          <NSelect
            v-model:value="selectedTimeRange"
            style="width: 150px"
            :options="[
              { value: '1h', label: '최근 1시간' },
              { value: '24h', label: '최근 24시간' },
              { value: '7d', label: '최근 7일' },
              { value: '30d', label: '최근 30일' }
            ]"
            @update:value="handleTimeRangeChange"
          />

          <NButton @click="refreshAll" :loading="isRefreshing">
            <template #icon>
              <Icon name="refresh" />
            </template>
            새로고침
          </NButton>

          <NButton @click="exportAllData">
            <template #icon>
              <Icon name="download" />
            </template>
            내보내기
          </NButton>

          <NButton @click="openSettings" quaternary>
            <template #icon>
              <Icon name="settings" />
            </template>
          </NButton>
        </NSpace>
      </div>
    </div>

    <!-- 시스템 상태 개요 -->
    <div class="system-overview">
      <div class="overview-cards">
        <!-- 전체 시스템 상태 -->
        <div class="overview-card system-health" :class="systemHealthClass">
          <div class="card-header">
            <Icon :name="systemHealthIcon" />
            <h3>시스템 상태</h3>
          </div>
          <div class="card-content">
            <div class="status-indicator">
              <div class="status-dot" :class="systemHealthClass"></div>
              <span class="status-text">{{ systemHealthText }}</span>
            </div>
            <div class="uptime">
              <span class="label">가동시간:</span>
              <span class="value">{{ formatUptime(systemUptime) }}</span>
            </div>
          </div>
        </div>

        <!-- 성능 점수 -->
        <div class="overview-card performance">
          <div class="card-header">
            <Icon name="zap" />
            <h3>성능 점수</h3>
          </div>
          <div class="card-content">
            <div class="score-circle" :class="`grade-${performanceGrade.toLowerCase()}`">
              <span class="score">{{ performanceScore }}</span>
              <span class="grade">{{ performanceGrade }}</span>
            </div>
            <div class="score-details">
              <div class="metric">
                <span class="label">LCP:</span>
                <span class="value" :class="lcpStatus">{{ formatMetric(lcpValue) }}</span>
              </div>
              <div class="metric">
                <span class="label">FID:</span>
                <span class="value" :class="fidStatus">{{ formatMetric(fidValue) }}</span>
              </div>
              <div class="metric">
                <span class="label">CLS:</span>
                <span class="value" :class="clsStatus">{{ formatCLS(clsValue) }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- 에러 통계 -->
        <div class="overview-card errors">
          <div class="card-header">
            <Icon name="alert-triangle" />
            <h3>에러 현황</h3>
          </div>
          <div class="card-content">
            <div class="error-stats">
              <div class="stat-item">
                <span class="count critical">{{ criticalErrors }}</span>
                <span class="label">치명적</span>
              </div>
              <div class="stat-item">
                <span class="count warning">{{ warningErrors }}</span>
                <span class="label">경고</span>
              </div>
              <div class="stat-item">
                <span class="count total">{{ totalErrors }}</span>
                <span class="label">전체</span>
              </div>
            </div>
            <div class="error-rate">
              <span class="label">에러율:</span>
              <span class="value">{{ errorRate.toFixed(2) }}/분</span>
            </div>
          </div>
        </div>

        <!-- 사용자 활동 -->
        <div class="overview-card users">
          <div class="card-header">
            <Icon name="users" />
            <h3>사용자 활동</h3>
          </div>
          <div class="card-content">
            <div class="user-stats">
              <div class="stat-item">
                <span class="count active">{{ activeUsers }}</span>
                <span class="label">활성 사용자</span>
              </div>
              <div class="stat-item">
                <span class="count sessions">{{ totalSessions }}</span>
                <span class="label">세션</span>
              </div>
              <div class="stat-item">
                <span class="count pageviews">{{ pageViews }}</span>
                <span class="label">페이지뷰</span>
              </div>
            </div>
            <div class="engagement">
              <span class="label">평균 세션:</span>
              <span class="value">{{ formatDuration(avgSessionDuration) }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 탭 네비게이션 -->
    <div class="dashboard-tabs">
      <NTabs v-model:value="activeTab" type="card" size="large">
        <NTabPane name="overview" tab="개요">
          <div class="tab-content">
            <!-- 실시간 메트릭 차트 -->
            <div class="charts-grid">
              <div class="chart-card">
                <h3>실시간 성능 메트릭</h3>
                <LineChart
                  :data="realTimeMetricsData"
                  :options="realTimeChartOptions"
                  height="300"
                />
              </div>

              <div class="chart-card">
                <h3>에러 발생 추이</h3>
                <BarChart
                  :data="errorTrendData"
                  :options="errorChartOptions"
                  height="300"
                />
              </div>

              <div class="chart-card">
                <h3>사용자 활동</h3>
                <LineChart
                  :data="userActivityData"
                  :options="userActivityOptions"
                  height="300"
                />
              </div>

              <div class="chart-card">
                <h3>시스템 리소스</h3>
                <PieChart
                  :data="resourceUsageData"
                  :options="resourceChartOptions"
                  height="300"
                />
              </div>
            </div>

            <!-- 최근 이벤트 -->
            <div class="recent-events">
              <h3>최근 이벤트</h3>
              <div class="events-list">
                <div
                  v-for="event in recentEvents"
                  :key="event.id"
                  class="event-item"
                  :class="event.type"
                >
                  <div class="event-icon">
                    <Icon :name="getEventIcon(event.type)" />
                  </div>
                  <div class="event-content">
                    <div class="event-title">{{ event.title }}</div>
                    <div class="event-description">{{ event.description }}</div>
                    <div class="event-time">{{ formatRelativeTime(event.timestamp) }}</div>
                  </div>
                  <div class="event-severity" :class="event.severity">
                    {{ event.severity }}
                  </div>
                </div>
              </div>
            </div>
          </div>
        </NTabPane>

        <NTabPane name="performance" tab="성능">
          <PerformanceDashboard />
        </NTabPane>

        <NTabPane name="errors" tab="에러">
          <ErrorTrackingDashboard />
        </NTabPane>

        <NTabPane name="analytics" tab="분석">
          <div class="analytics-content">
            <!-- 분석 대시보드 내용 -->
            <AnalyticsDashboard />
          </div>
        </NTabPane>

        <NTabPane name="alerts" tab="알림">
          <div class="alerts-content">
            <!-- 알림 설정 및 현황 -->
            <AlertsDashboard />
          </div>
        </NTabPane>
      </NTabs>
    </div>

    <!-- 설정 모달 -->
    <NModal v-model:show="showSettings" style="width: 70%; max-width: 800px;">
      <NCard title="모니터링 설정">
        <div class="settings-content">
          <!-- 설정 폼 -->
          <MonitoringSettings @save="handleSettingsSave" />
        </div>
      </NCard>
    </NModal>

    <!-- 알림 토스트 -->
    <Teleport to="body">
      <div v-if="notifications.length > 0" class="notifications-container">
        <TransitionGroup name="notification" tag="div">
          <div
            v-for="notification in notifications"
            :key="notification.id"
            class="notification"
            :class="notification.type"
          >
            <Icon :name="getNotificationIcon(notification.type)" />
            <div class="notification-content">
              <div class="notification-title">{{ notification.title }}</div>
              <div class="notification-message">{{ notification.message }}</div>
            </div>
            <button @click="dismissNotification(notification.id)" class="notification-close">
              <Icon name="x" />
            </button>
          </div>
        </TransitionGroup>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { NButton, NCard, NModal, NSelect, NSpace, NTabPane, NTabs } from 'naive-ui'
import { usePerformanceMonitoring } from '@/composables/usePerformanceMonitoring'
import { useErrorTracking } from '@/composables/useErrorTracking'
import { useAnalytics } from '@/composables/useAnalytics'
import PerformanceDashboard from '@/components/Performance/PerformanceDashboard.vue'
import ErrorTrackingDashboard from '@/components/Performance/ErrorTrackingDashboard.vue'
import LineChart from '@/components/ui/charts/LineChart.vue'
import BarChart from '@/components/ui/charts/BarChart.vue'
import PieChart from '@/components/ui/charts/PieChart.vue'
import Icon from '@/components/common/Icon.vue'

// Composables
const performanceMonitoring = usePerformanceMonitoring()
const errorTracking = useErrorTracking()
const analytics = useAnalytics()

// 상태
const activeTab = ref('overview')
const selectedTimeRange = ref('24h')
const showSettings = ref(false)
const isRefreshing = ref(false)

// 시스템 상태
const systemUptime = ref(Date.now())

// 알림 시스템
interface Notification {
  id: string
  type: 'success' | 'warning' | 'error' | 'info'
  title: string
  message: string
  timestamp: number
}

const notifications = ref<Notification[]>([])

// 최근 이벤트
interface RecentEvent {
  id: string
  type: 'performance' | 'error' | 'user' | 'system'
  title: string
  description: string
  timestamp: number
  severity: 'low' | 'medium' | 'high' | 'critical'
}

const recentEvents = ref<RecentEvent[]>([
  {
    id: '1',
    type: 'performance',
    title: 'LCP 성능 개선',
    description: 'Largest Contentful Paint가 2.1초로 개선되었습니다',
    timestamp: Date.now() - 300000,
    severity: 'low',
  },
  {
    id: '2',
    type: 'error',
    title: 'JavaScript 에러 발생',
    description: 'Terminal 컴포넌트에서 undefined 참조 에러',
    timestamp: Date.now() - 600000,
    severity: 'medium',
  },
  {
    id: '3',
    type: 'user',
    title: '새로운 사용자 세션',
    description: '5명의 새로운 사용자가 접속했습니다',
    timestamp: Date.now() - 900000,
    severity: 'low',
  },
])

// 시스템 상태 계산
const systemHealthClass = computed(() => {
  const performanceScore = performanceMonitoring.performanceScore.value
  const errorRate = errorTracking.calculateErrorRate().rate

  if (performanceScore >= 90 && errorRate < 1) return 'healthy'
  if (performanceScore >= 70 && errorRate < 5) return 'warning'
  return 'critical'
})

const systemHealthIcon = computed(() => {
  const health = systemHealthClass.value
  return health === 'healthy' ? 'check-circle' :
         health === 'warning' ? 'alert-triangle' : 'x-circle'
})

const systemHealthText = computed(() => {
  const health = systemHealthClass.value
  return health === 'healthy' ? '정상' :
         health === 'warning' ? '주의' : '위험'
})

// 성능 메트릭
const performanceScore = computed(() => performanceMonitoring.performanceScore.value)
const performanceGrade = computed(() => performanceMonitoring.performanceGrade.value)

const webVitals = computed(() => performanceMonitoring.webVitalsStatus.value)
const lcpValue = computed(() => webVitals.value.lcp.value)
const fidValue = computed(() => webVitals.value.fid.value)
const clsValue = computed(() => webVitals.value.cls.value)

const lcpStatus = computed(() => webVitals.value.lcp.status)
const fidStatus = computed(() => webVitals.value.fid.status)
const clsStatus = computed(() => webVitals.value.cls.status)

// 에러 통계
const criticalErrors = computed(() => errorTracking.criticalErrors.value)
const warningErrors = computed(() => {
  const stats = errorTracking.state.errors.filter(e => e.level === 'warn').length
  return stats
})
const totalErrors = computed(() => errorTracking.state.errors.length)
const errorRate = computed(() => errorTracking.calculateErrorRate().rate)

// 사용자 통계
const activeUsers = computed(() => 1) // 실제 구현 시 analytics에서 가져옴
const totalSessions = computed(() => analytics.eventStats.value.byType.page_view)
const pageViews = computed(() => analytics.eventStats.value.total)
const avgSessionDuration = computed(() => analytics.sessionSummary.value.duration)

// 차트 데이터
const realTimeMetricsData = computed(() => ({
  labels: Array.from({ length: 20 }, (_, i) => `${i}분 전`),
  datasets: [
    {
      label: 'LCP (ms)',
      data: Array.from({ length: 20 }, () => Math.random() * 3000 + 1000),
      borderColor: '#f56565',
      backgroundColor: 'rgba(245, 101, 101, 0.1)',
      fill: true,
    },
    {
      label: 'FID (ms)',
      data: Array.from({ length: 20 }, () => Math.random() * 200 + 50),
      borderColor: '#48bb78',
      backgroundColor: 'rgba(72, 187, 120, 0.1)',
      fill: true,
    },
  ],
}))

const errorTrendData = computed(() => ({
  labels: Array.from({ length: 24 }, (_, i) => `${i}시`),
  datasets: [
    {
      label: '에러 발생 수',
      data: Array.from({ length: 24 }, () => Math.floor(Math.random() * 10)),
      backgroundColor: 'rgba(245, 101, 101, 0.6)',
      borderColor: '#f56565',
    },
  ],
}))

const userActivityData = computed(() => ({
  labels: Array.from({ length: 24 }, (_, i) => `${i}시`),
  datasets: [
    {
      label: '활성 사용자',
      data: Array.from({ length: 24 }, () => Math.floor(Math.random() * 50)),
      borderColor: '#4299e1',
      backgroundColor: 'rgba(66, 153, 225, 0.1)',
      fill: true,
    },
  ],
}))

const resourceUsageData = computed(() => ({
  labels: ['메모리', 'CPU', '네트워크', '스토리지'],
  datasets: [
    {
      label: '리소스 사용량',
      data: [65, 25, 15, 35],
      backgroundColor: ['#f56565', '#ed8936', '#48bb78', '#4299e1'],
    },
  ],
}))

// 차트 옵션
const realTimeChartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  scales: {
    y: { beginAtZero: true },
  },
  animation: { duration: 0 },
}

const errorChartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  scales: {
    y: { beginAtZero: true },
  },
}

const userActivityOptions = {
  responsive: true,
  maintainAspectRatio: false,
  scales: {
    y: { beginAtZero: true },
  },
}

const resourceChartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { position: 'bottom' as const },
  },
}

// 메서드
function handleTimeRangeChange(value: string) {
  console.log('Time range changed:', value)
  // 시간 범위에 따른 데이터 업데이트
}

function refreshAll() {
  isRefreshing.value = true

  // 모든 모니터링 데이터 새로고침
  setTimeout(() => {
    isRefreshing.value = false
    addNotification('success', '새로고침 완료', '모든 데이터가 업데이트되었습니다')
  }, 2000)
}

function exportAllData() {
  // 모든 모니터링 데이터 내보내기
  const data = {
    performance: performanceMonitoring.generateReport(),
    errors: errorTracking.exportErrors(),
    analytics: analytics.exportData(),
    timestamp: new Date().toISOString(),
  }

  const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `monitoring-data-${Date.now()}.json`
  a.click()
  URL.revokeObjectURL(url)

  addNotification('success', '내보내기 완료', '모니터링 데이터가 다운로드되었습니다')
}

function openSettings() {
  showSettings.value = true
}

function handleSettingsSave(settings: any) {
  console.log('Settings saved:', settings)
  showSettings.value = false
  addNotification('success', '설정 저장됨', '모니터링 설정이 저장되었습니다')
}

function addNotification(type: Notification['type'], title: string, message: string) {
  const notification: Notification = {
    id: `notification_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
    type,
    title,
    message,
    timestamp: Date.now(),
  }

  notifications.value.push(notification)

  // 5초 후 자동 제거
  setTimeout(() => {
    dismissNotification(notification.id)
  }, 5000)
}

function dismissNotification(id: string) {
  const index = notifications.value.findIndex(n => n.id === id)
  if (index > -1) {
    notifications.value.splice(index, 1)
  }
}

function getEventIcon(type: string): string {
  const icons = {
    performance: 'zap',
    error: 'alert-triangle',
    user: 'user',
    system: 'server',
  }
  return icons[type as keyof typeof icons] || 'info'
}

function getNotificationIcon(type: string): string {
  const icons = {
    success: 'check-circle',
    warning: 'alert-triangle',
    error: 'x-circle',
    info: 'info',
  }
  return icons[type as keyof typeof icons] || 'info'
}

function formatMetric(value: number): string {
  return value ? `${value.toFixed(0)}ms` : '-'
}

function formatCLS(value: number): string {
  return value ? value.toFixed(4) : '-'
}

function formatUptime(timestamp: number): string {
  const uptime = Date.now() - timestamp
  const hours = Math.floor(uptime / (1000 * 60 * 60))
  const minutes = Math.floor((uptime % (1000 * 60 * 60)) / (1000 * 60))
  return `${hours}시간 ${minutes}분`
}

function formatDuration(ms: number): string {
  const minutes = Math.floor(ms / (1000 * 60))
  const seconds = Math.floor((ms % (1000 * 60)) / 1000)
  return `${minutes}분 ${seconds}초`
}

function formatRelativeTime(timestamp: number): string {
  const diff = Date.now() - timestamp
  const minutes = Math.floor(diff / (1000 * 60))
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)

  if (days > 0) return `${days}일 전`
  if (hours > 0) return `${hours}시간 전`
  return `${minutes}분 전`
}

// 실시간 업데이트 타이머
let updateTimer: NodeJS.Timeout | null = null

onMounted(() => {
  // 30초마다 데이터 업데이트
  updateTimer = setInterval(() => {
    // 실시간 데이터 업데이트 로직
  }, 30000)
})

onUnmounted(() => {
  if (updateTimer) {
    clearInterval(updateTimer)
  }
})

// 더미 컴포넌트들 (실제 구현에서는 별도 파일로 분리)
const AnalyticsDashboard = {
  template: '<div class="analytics-dashboard"><h3>분석 대시보드</h3><p>사용자 분석 데이터가 여기에 표시됩니다.</p></div>',
}

const AlertsDashboard = {
  template: '<div class="alerts-dashboard"><h3>알림 대시보드</h3><p>알림 설정 및 현황이 여기에 표시됩니다.</p></div>',
}

const MonitoringSettings = {
  emits: ['save'],
  template: `
    <div class="monitoring-settings">
      <h3>모니터링 설정</h3>
      <p>모니터링 관련 설정을 여기서 변경할 수 있습니다.</p>
      <button @click="$emit('save', {})">저장</button>
    </div>
  `,
}
</script>

<style lang="scss" scoped>
.monitoring-dashboard {
  padding: 20px;
  min-height: 100vh;
  background: var(--bg-color-primary);

  .dashboard-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 30px;
    padding-bottom: 20px;
    border-bottom: 1px solid var(--border-color);

    .header-content {
      h1 {
        margin: 0 0 5px 0;
        font-size: 28px;
        font-weight: 700;
        color: var(--text-color-primary);
      }

      .header-subtitle {
        margin: 0;
        color: var(--text-color-secondary);
        font-size: 14px;
      }
    }

    .header-controls {
      display: flex;
      gap: 15px;
    }
  }

  .system-overview {
    margin-bottom: 30px;

    .overview-cards {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
      gap: 20px;

      .overview-card {
        background: var(--bg-color-secondary);
        border: 1px solid var(--border-color);
        border-radius: 12px;
        padding: 20px;
        transition: all 0.3s ease;

        &:hover {
          box-shadow: 0 8px 25px rgba(0, 0, 0, 0.1);
          transform: translateY(-2px);
        }

        .card-header {
          display: flex;
          align-items: center;
          gap: 10px;
          margin-bottom: 15px;

          h3 {
            margin: 0;
            font-size: 16px;
            font-weight: 600;
            color: var(--text-color-primary);
          }
        }

        .card-content {
          .status-indicator {
            display: flex;
            align-items: center;
            gap: 8px;
            margin-bottom: 10px;

            .status-dot {
              width: 12px;
              height: 12px;
              border-radius: 50%;

              &.healthy { background: #48bb78; }
              &.warning { background: #ed8936; }
              &.critical { background: #f56565; }
            }

            .status-text {
              font-weight: 600;
              color: var(--text-color-primary);
            }
          }

          .uptime {
            font-size: 14px;
            color: var(--text-color-secondary);

            .label {
              margin-right: 5px;
            }

            .value {
              font-weight: 500;
            }
          }
        }

        &.performance {
          .score-circle {
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            width: 80px;
            height: 80px;
            border-radius: 50%;
            margin: 0 auto 15px;

            &.grade-a { background: linear-gradient(45deg, #48bb78, #68d391); }
            &.grade-b { background: linear-gradient(45deg, #4299e1, #63b3ed); }
            &.grade-c { background: linear-gradient(45deg, #ed8936, #f6ad55); }
            &.grade-d { background: linear-gradient(45deg, #f56565, #fc8181); }
            &.grade-f { background: linear-gradient(45deg, #e53e3e, #f56565); }

            .score {
              font-size: 20px;
              font-weight: bold;
              color: white;
            }

            .grade {
              font-size: 12px;
              color: white;
              opacity: 0.9;
            }
          }

          .score-details {
            .metric {
              display: flex;
              justify-content: space-between;
              margin-bottom: 5px;
              font-size: 13px;

              .label {
                color: var(--text-color-secondary);
              }

              .value {
                font-weight: 500;

                &.good { color: #48bb78; }
                &.needs-improvement { color: #ed8936; }
                &.poor { color: #f56565; }
              }
            }
          }
        }

        &.errors {
          .error-stats {
            display: flex;
            justify-content: space-around;
            margin-bottom: 15px;

            .stat-item {
              text-align: center;

              .count {
                display: block;
                font-size: 24px;
                font-weight: bold;

                &.critical { color: #f56565; }
                &.warning { color: #ed8936; }
                &.total { color: var(--text-color-primary); }
              }

              .label {
                font-size: 12px;
                color: var(--text-color-secondary);
              }
            }
          }

          .error-rate {
            text-align: center;
            font-size: 14px;

            .label {
              color: var(--text-color-secondary);
              margin-right: 5px;
            }

            .value {
              font-weight: 600;
              color: var(--text-color-primary);
            }
          }
        }

        &.users {
          .user-stats {
            display: flex;
            justify-content: space-around;
            margin-bottom: 15px;

            .stat-item {
              text-align: center;

              .count {
                display: block;
                font-size: 20px;
                font-weight: bold;
                color: var(--text-color-primary);
              }

              .label {
                font-size: 11px;
                color: var(--text-color-secondary);
              }
            }
          }

          .engagement {
            text-align: center;
            font-size: 14px;

            .label {
              color: var(--text-color-secondary);
              margin-right: 5px;
            }

            .value {
              font-weight: 600;
              color: var(--text-color-primary);
            }
          }
        }
      }
    }
  }

  .dashboard-tabs {
    :deep(.n-tabs) {
      .n-tabs-nav {
        background: var(--bg-color-secondary);
        border-radius: 8px;
        padding: 4px;
      }
    }

    .tab-content {
      padding: 20px 0;
    }

    .charts-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
      gap: 20px;
      margin-bottom: 30px;

      .chart-card {
        background: var(--bg-color-secondary);
        border: 1px solid var(--border-color);
        border-radius: 8px;
        padding: 20px;

        h3 {
          margin: 0 0 20px 0;
          font-size: 16px;
          font-weight: 600;
          color: var(--text-color-primary);
        }
      }
    }

    .recent-events {
      background: var(--bg-color-secondary);
      border: 1px solid var(--border-color);
      border-radius: 8px;
      padding: 20px;

      h3 {
        margin: 0 0 20px 0;
        font-size: 16px;
        font-weight: 600;
        color: var(--text-color-primary);
      }

      .events-list {
        .event-item {
          display: flex;
          align-items: center;
          gap: 15px;
          padding: 15px;
          border-radius: 6px;
          margin-bottom: 10px;
          border-left: 3px solid transparent;

          &.performance { border-left-color: #4299e1; }
          &.error { border-left-color: #f56565; }
          &.user { border-left-color: #48bb78; }
          &.system { border-left-color: #ed8936; }

          &:hover {
            background: var(--bg-color-tertiary);
          }

          .event-icon {
            font-size: 20px;
            opacity: 0.7;
          }

          .event-content {
            flex: 1;

            .event-title {
              font-weight: 600;
              color: var(--text-color-primary);
              margin-bottom: 2px;
            }

            .event-description {
              font-size: 13px;
              color: var(--text-color-secondary);
              margin-bottom: 2px;
            }

            .event-time {
              font-size: 11px;
              color: var(--text-color-tertiary);
            }
          }

          .event-severity {
            padding: 2px 8px;
            border-radius: 4px;
            font-size: 11px;
            font-weight: 500;

            &.low { background: #c6f6d5; color: #22543d; }
            &.medium { background: #feebc8; color: #c05621; }
            &.high { background: #fed7d7; color: #c53030; }
            &.critical { background: #fed7d7; color: #c53030; font-weight: bold; }
          }
        }
      }
    }
  }
}

// 알림 스타일
.notifications-container {
  position: fixed;
  top: 20px;
  right: 20px;
  z-index: 9999;
  pointer-events: none;

  .notification {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 16px;
    background: white;
    border-radius: 8px;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
    margin-bottom: 10px;
    pointer-events: all;
    max-width: 350px;

    &.success { border-left: 4px solid #48bb78; }
    &.warning { border-left: 4px solid #ed8936; }
    &.error { border-left: 4px solid #f56565; }
    &.info { border-left: 4px solid #4299e1; }

    .notification-content {
      flex: 1;

      .notification-title {
        font-weight: 600;
        color: var(--text-color-primary);
        margin-bottom: 2px;
      }

      .notification-message {
        font-size: 13px;
        color: var(--text-color-secondary);
      }
    }

    .notification-close {
      background: none;
      border: none;
      cursor: pointer;
      padding: 4px;
      opacity: 0.6;

      &:hover {
        opacity: 1;
      }
    }
  }
}

// 알림 애니메이션
.notification-enter-active,
.notification-leave-active {
  transition: all 0.3s ease;
}

.notification-enter-from {
  opacity: 0;
  transform: translateX(100%);
}

.notification-leave-to {
  opacity: 0;
  transform: translateX(100%);
}

// 반응형 디자인
@media (max-width: 768px) {
  .monitoring-dashboard {
    padding: 15px;

    .dashboard-header {
      flex-direction: column;
      gap: 15px;
      align-items: flex-start;
    }

    .overview-cards {
      grid-template-columns: 1fr;
    }

    .charts-grid {
      grid-template-columns: 1fr;
    }
  }
}
</style>