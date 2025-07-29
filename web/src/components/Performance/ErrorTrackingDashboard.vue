<template>
  <div class="error-tracking-dashboard">
    <!-- 대시보드 헤더 -->
    <div class="dashboard-header">
      <h2>에러 트래킹 대시보드</h2>
      <div class="dashboard-controls">
        <NButton
          v-if="!isTracking"
          @click="startTracking"
          type="primary"
        >
          트래킹 시작
        </NButton>
        <NButton
          v-else
          @click="stopTracking"
          type="error"
        >
          트래킹 중지
        </NButton>
        <NButton @click="clearErrors" :disabled="!hasErrors">
          모두 삭제
        </NButton>
        <NButton @click="exportErrors" :disabled="!hasErrors">
          내보내기
        </NButton>
        <NDropdown :options="testErrorOptions" @select="handleTestError">
          <NButton>테스트 에러</NButton>
        </NDropdown>
      </div>
    </div>

    <!-- 에러 통계 개요 -->
    <div class="error-overview">
      <div class="stats-cards">
        <div class="stat-card total">
          <div class="stat-icon">
            <Icon name="alert-circle" />
          </div>
          <div class="stat-content">
            <div class="stat-value">{{ errorStats.total }}</div>
            <div class="stat-label">총 에러</div>
          </div>
        </div>

        <div class="stat-card critical">
          <div class="stat-icon">
            <Icon name="x-circle" />
          </div>
          <div class="stat-content">
            <div class="stat-value">{{ criticalErrors }}</div>
            <div class="stat-label">치명적 에러</div>
          </div>
        </div>

        <div class="stat-card recent">
          <div class="stat-icon">
            <Icon name="clock" />
          </div>
          <div class="stat-content">
            <div class="stat-value">{{ recentErrors }}</div>
            <div class="stat-label">최근 1분</div>
          </div>
        </div>

        <div class="stat-card rate">
          <div class="stat-icon">
            <Icon name="trending-up" />
          </div>
          <div class="stat-content">
            <div class="stat-value">{{ errorRate.rate.toFixed(2) }}</div>
            <div class="stat-label">분당 에러율</div>
          </div>
        </div>
      </div>

      <!-- 레벨별 에러 분포 -->
      <div class="error-levels">
        <h3>에러 레벨 분포</h3>
        <div class="level-bars">
          <div
            v-for="(count, level) in errorStats"
            :key="level"
            v-show="level !== 'total' && level !== 'recentErrors'"
            class="level-bar"
            :class="`level-${level}`"
          >
            <div class="level-info">
              <span class="level-name">{{ getLevelName(level as ErrorLevel) }}</span>
              <span class="level-count">{{ count }}</span>
            </div>
            <div class="level-progress">
              <div
                class="level-fill"
                :style="{
                  width: `${(count / Math.max(errorStats.total, 1)) * 100}%`,
                  backgroundColor: errorLevelColors[level as ErrorLevel]
                }"
              ></div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 필터 섹션 -->
    <div class="filter-section">
      <div class="filter-controls">
        <NSelect
          v-model:value="selectedLevel"
          placeholder="에러 레벨"
          clearable
          style="width: 150px"
          :options="[
            { value: 'debug', label: 'Debug' },
            { value: 'info', label: 'Info' },
            { value: 'warn', label: 'Warning' },
            { value: 'error', label: 'Error' },
            { value: 'fatal', label: 'Fatal' }
          ]"
          @update:value="updateFilter"
        />

        <NInput
          v-model:value="searchQuery"
          placeholder="검색어 입력..."
          style="width: 200px"
          @update:value="updateFilter"
        >
          <template #prefix>
            <Icon name="search" />
          </template>
        </NInput>

        <NDatePicker
          v-model:value="timeRange"
          type="datetimerange"
          clearable
          style="width: 300px"
          @update:value="updateFilter"
        />

        <NButton @click="resetFilter" :disabled="!hasActiveFilter">
          필터 초기화
        </NButton>
      </div>
    </div>

    <!-- 에러 차트 -->
    <div class="charts-section">
      <div class="chart-container">
        <h3>시간대별 에러 발생</h3>
        <BarChart
          :data="timeChartData"
          :options="timeChartOptions"
          :height="250"
        />
      </div>

      <div class="chart-container">
        <h3>컴포넌트별 에러</h3>
        <PieChart
          :data="componentChartData"
          :options="componentChartOptions"
          :height="250"
        />
      </div>
    </div>

    <!-- 에러 목록 -->
    <div class="error-list-section">
      <div class="section-header">
        <h3>에러 목록 ({{ filteredErrors.length }})</h3>
        <div class="view-controls">
          <NRadioGroup v-model:value="viewMode">
            <NRadioButton value="card">카드</NRadioButton>
            <NRadioButton value="table">테이블</NRadioButton>
          </NRadioGroup>
        </div>
      </div>

      <!-- 카드 뷰 -->
      <div v-if="viewMode === 'card'" class="error-cards">
        <div
          v-for="error in paginatedErrors"
          :key="error.id"
          class="error-card"
          :class="`level-${error.level}`"
          @click="showErrorDetails(error)"
        >
          <div class="error-header">
            <div class="error-level">
              <Icon :name="getLevelIcon(error.level)" />
              <span>{{ getLevelName(error.level) }}</span>
            </div>
            <div class="error-time">
              {{ formatTime(error.timestamp) }}
            </div>
            <NButton
              size="small"
              @click.stop="removeError(error.id)"
              quaternary
              type="error"
            >
              <Icon name="close" />
            </NButton>
          </div>

          <div class="error-message">
            {{ error.message }}
          </div>

          <div class="error-meta">
            <span v-if="error.component" class="meta-item">
              <Icon name="layers" />
              {{ error.component }}
            </span>
            <span v-if="error.action" class="meta-item">
              <Icon name="zap" />
              {{ error.action }}
            </span>
            <span v-if="error.url" class="meta-item">
              <Icon name="link" />
              {{ getShortUrl(error.url) }}
            </span>
          </div>
        </div>
      </div>

      <!-- 테이블 뷰 -->
      <div v-else class="error-table">
        <BaseDataTable
          :data="paginatedErrors"
          :columns="errorTableColumns"
          :loading="false"
        />
      </div>

      <!-- 페이지네이션 -->
      <div v-if="totalPages > 1" class="pagination">
        <NPagination
          v-model:page="currentPage"
          :page-count="totalPages"
          :page-size="pageSize"
          show-size-picker
          :page-sizes="[10, 20, 50, 100]"
          @update:page-size="handlePageSizeChange"
        />
      </div>
    </div>

    <!-- 에러 상세 모달 -->
    <NModal v-model:show="showDetailModal" style="width: 80%; max-width: 900px;">
      <NCard title="에러 상세 정보">
        <div v-if="selectedError" class="error-details">
          <!-- 기본 정보 -->
          <div class="detail-section">
            <h4>기본 정보</h4>
            <div class="detail-grid">
              <div v-for="(value, key) in errorDetails.basicInfo" :key="key" class="detail-item">
                <label>{{ key }}</label>
                <span>{{ value }}</span>
              </div>
            </div>
          </div>

          <!-- 기술적 정보 -->
          <div class="detail-section">
            <h4>기술적 정보</h4>
            <div class="detail-grid">
              <div v-for="(value, key) in errorDetails.technicalInfo" :key="key" class="detail-item">
                <label>{{ key }}</label>
                <span>{{ value }}</span>
              </div>
            </div>
          </div>

          <!-- 스택 트레이스 -->
          <div v-if="errorDetails.stack" class="detail-section">
            <h4>스택 트레이스</h4>
            <pre class="stack-trace">{{ errorDetails.stack }}</pre>
          </div>

          <!-- 메타데이터 -->
          <div v-if="errorDetails.metadata" class="detail-section">
            <h4>메타데이터</h4>
            <pre class="metadata">{{ JSON.stringify(errorDetails.metadata, null, 2) }}</pre>
          </div>
        </div>

        <template #footer>
          <div class="modal-footer">
            <NButton @click="showDetailModal = false">닫기</NButton>
            <NButton
              v-if="selectedError"
              type="error"
              @click="removeError(selectedError.id); showDetailModal = false"
            >
              이 에러 삭제
            </NButton>
          </div>
        </template>
      </NCard>
    </NModal>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import {
  NButton, NCard, NDatePicker, NDropdown, NInput, NModal,
  NPagination, NRadioButton, NRadioGroup, NSelect,
} from 'naive-ui'
import { useErrorTracking } from '@/composables/useErrorTracking'
import { type ErrorEvent, type ErrorLevel } from '@/utils/error-tracker'
import BarChart from '@/components/ui/charts/BarChart.vue'
import PieChart from '@/components/ui/charts/PieChart.vue'
import BaseDataTable from '@/components/ui/data/BaseDataTable.vue'
import Icon from '@/components/common/Icon.vue'

// 에러 트래킹 composable
const {
  state: _state,
  filteredErrors,
  errorStats,
  errorLevelColors,
  errorsByTime,
  errorsByComponent,
  startTracking,
  stopTracking,
  clearErrors,
  removeError,
  exportErrors,
  setFilter,
  resetFilter,
  formatErrorDetails,
  calculateErrorRate,
  generateTestError,
  isTracking,
  hasErrors,
  recentErrors,
  criticalErrors,
} = useErrorTracking()

// 뷰 상태
const viewMode = ref<'card' | 'table'>('card')
const currentPage = ref(1)
const pageSize = ref(20)
const showDetailModal = ref(false)
const selectedError = ref<ErrorEvent | null>(null)

// 필터 상태
const selectedLevel = ref<ErrorLevel | null>(null)
const searchQuery = ref('')
const timeRange = ref<[number, number] | null>(null)

// 에러율 계산
const errorRate = computed(() => calculateErrorRate())

// 페이지네이션
const totalPages = computed(() => Math.ceil(filteredErrors.value.length / pageSize.value))
const paginatedErrors = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return filteredErrors.value.slice(start, end)
})

// 활성 필터 확인
const hasActiveFilter = computed(() =>
  selectedLevel.value || searchQuery.value || timeRange.value,
)

// 에러 상세 정보
const errorDetails = computed(() => {
  if (!selectedError.value) return null
  return formatErrorDetails(selectedError.value)
})

// 차트 데이터
const timeChartData = computed(() => ({
  labels: errorsByTime.value.map(slot => `${slot.hour}시`),
  datasets: [
    {
      label: '에러 발생 수',
      data: errorsByTime.value.map(slot => slot.count),
      backgroundColor: 'rgba(245, 101, 101, 0.6)',
      borderColor: '#f56565',
      borderWidth: 1,
    },
  ],
}))

const componentChartData = computed(() => {
  const topComponents = errorsByComponent.value.slice(0, 8)
  return {
    labels: topComponents.map(item => item.component),
    datasets: [
      {
        label: '에러 발생 횟수',
        data: topComponents.map(item => item.count),
        backgroundColor: [
          '#f56565', '#ed8936', '#ecc94b', '#48bb78',
          '#38b2ac', '#4299e1', '#667eea', '#9f7aea',
        ],
      },
    ],
  }
})

// 차트 옵션
const timeChartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  scales: {
    y: {
      beginAtZero: true,
      title: {
        display: true,
        text: '에러 수',
      },
    },
    x: {
      title: {
        display: true,
        text: '시간',
      },
    },
  },
}

const componentChartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: {
      position: 'bottom' as const,
    },
  },
}

// 테이블 컬럼
const errorTableColumns = [
  {
    key: 'level',
    title: '레벨',
    width: 80,
    render: (row: ErrorEvent) => `<span class="level-badge level-${row.level}">${getLevelName(row.level)}</span>`,
  },
  {
    key: 'message',
    title: '메시지',
    ellipsis: true,
    width: 300,
  },
  {
    key: 'component',
    title: '컴포넌트',
    width: 120,
  },
  {
    key: 'timestamp',
    title: '시간',
    width: 150,
    render: (row: ErrorEvent) => formatTime(row.timestamp),
  },
  {
    key: 'actions',
    title: '작업',
    width: 100,
    render: (row: ErrorEvent) => `
      <button class="action-btn view" onclick="window.showErrorDetails('${row.id}')">
        상세
      </button>
      <button class="action-btn delete" onclick="window.removeError('${row.id}')">
        삭제
      </button>
    `,
  },
]

// 테스트 에러 옵션
const testErrorOptions = [
  { label: 'Debug', key: 'debug' },
  { label: 'Info', key: 'info' },
  { label: 'Warning', key: 'warn' },
  { label: 'Error', key: 'error' },
  { label: 'Fatal', key: 'fatal' },
]

// 메서드
function updateFilter() {
  setFilter({
    level: selectedLevel.value || undefined,
    searchQuery: searchQuery.value || undefined,
    timeRange: timeRange.value ? {
      start: timeRange.value[0],
      end: timeRange.value[1],
    } : undefined,
  })
  currentPage.value = 1
}

function handlePageSizeChange(newSize: number) {
  pageSize.value = newSize
  currentPage.value = 1
}

function showErrorDetails(error: ErrorEvent) {
  selectedError.value = error
  showDetailModal.value = true
}

function handleTestError(key: string) {
  generateTestError(key as ErrorLevel)
}

function getLevelName(level: ErrorLevel): string {
  const names: Record<ErrorLevel, string> = {
    debug: 'Debug',
    info: 'Info',
    warn: '경고',
    error: '에러',
    fatal: '치명적',
  }
  return names[level] || level
}

function getLevelIcon(level: ErrorLevel): string {
  const icons: Record<ErrorLevel, string> = {
    debug: 'bug',
    info: 'info',
    warn: 'warning',
    error: 'x-circle',
    fatal: 'alert-triangle',
  }
  return icons[level] || 'help'
}

function formatTime(timestamp: number): string {
  return new Date(timestamp).toLocaleString('ko-KR')
}

function getShortUrl(url: string): string {
  try {
    const urlObj = new URL(url)
    return urlObj.pathname.substring(0, 30) + (urlObj.pathname.length > 30 ? '...' : '')
  } catch {
    return url.substring(0, 30) + (url.length > 30 ? '...' : '')
  }
}

// 글로벌 메서드 등록 (테이블 액션용)
;(window as Record<string, unknown>).showErrorDetails = (errorId: string) => {
  const error = filteredErrors.value.find(e => e.id === errorId)
  if (error) showErrorDetails(error)
}

;(window as Record<string, unknown>).removeError = removeError

// 페이지 변경 감시
watch(filteredErrors, () => {
  if (currentPage.value > totalPages.value) {
    currentPage.value = Math.max(1, totalPages.value)
  }
})
</script>

<style lang="scss" scoped>
.error-tracking-dashboard {
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

  .error-overview {
    margin-bottom: 30px;

    .stats-cards {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
      gap: 20px;
      margin-bottom: 30px;

      .stat-card {
        display: flex;
        align-items: center;
        gap: 15px;
        padding: 20px;
        background: var(--bg-color-secondary);
        border-radius: 8px;
        border: 1px solid var(--border-color);

        &.total { border-left: 4px solid #4299e1; }
        &.critical { border-left: 4px solid #f56565; }
        &.recent { border-left: 4px solid #ed8936; }
        &.rate { border-left: 4px solid #48bb78; }

        .stat-icon {
          font-size: 24px;
          opacity: 0.7;
        }

        .stat-content {
          .stat-value {
            font-size: 24px;
            font-weight: bold;
            color: var(--text-color-primary);
          }

          .stat-label {
            font-size: 12px;
            color: var(--text-color-secondary);
          }
        }
      }
    }

    .error-levels {
      background: var(--bg-color-secondary);
      padding: 20px;
      border-radius: 8px;
      border: 1px solid var(--border-color);

      h3 {
        margin: 0 0 20px 0;
        color: var(--text-color-primary);
      }

      .level-bars {
        .level-bar {
          margin-bottom: 15px;

          .level-info {
            display: flex;
            justify-content: space-between;
            margin-bottom: 5px;

            .level-name {
              font-weight: 500;
              color: var(--text-color-primary);
            }

            .level-count {
              font-weight: bold;
              color: var(--text-color-secondary);
            }
          }

          .level-progress {
            height: 8px;
            background: var(--bg-color-tertiary);
            border-radius: 4px;
            overflow: hidden;

            .level-fill {
              height: 100%;
              transition: width 0.3s ease;
            }
          }
        }
      }
    }
  }

  .filter-section {
    margin-bottom: 30px;

    .filter-controls {
      display: flex;
      gap: 15px;
      flex-wrap: wrap;
      align-items: center;
    }
  }

  .charts-section {
    display: grid;
    grid-template-columns: 2fr 1fr;
    gap: 30px;
    margin-bottom: 30px;

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

  .error-list-section {
    .section-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 20px;

      h3 {
        margin: 0;
        color: var(--text-color-primary);
      }
    }

    .error-cards {
      display: grid;
      gap: 15px;

      .error-card {
        background: var(--bg-color-secondary);
        border: 1px solid var(--border-color);
        border-radius: 8px;
        padding: 20px;
        cursor: pointer;
        transition: all 0.2s ease;

        &:hover {
          box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
          transform: translateY(-2px);
        }

        &.level-debug { border-left: 4px solid #718096; }
        &.level-info { border-left: 4px solid #4299e1; }
        &.level-warn { border-left: 4px solid #ed8936; }
        &.level-error { border-left: 4px solid #f56565; }
        &.level-fatal { border-left: 4px solid #e53e3e; }

        .error-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 10px;

          .error-level {
            display: flex;
            align-items: center;
            gap: 5px;
            font-weight: 500;
            font-size: 12px;
            color: var(--text-color-secondary);
          }

          .error-time {
            font-size: 11px;
            color: var(--text-color-tertiary);
          }
        }

        .error-message {
          font-weight: 500;
          color: var(--text-color-primary);
          margin-bottom: 10px;
          line-height: 1.4;
        }

        .error-meta {
          display: flex;
          gap: 15px;
          flex-wrap: wrap;

          .meta-item {
            display: flex;
            align-items: center;
            gap: 5px;
            font-size: 11px;
            color: var(--text-color-tertiary);
          }
        }
      }
    }

    .pagination {
      margin-top: 30px;
      display: flex;
      justify-content: center;
    }
  }

  .error-details {
    .detail-section {
      margin-bottom: 30px;

      h4 {
        margin: 0 0 15px 0;
        color: var(--text-color-primary);
        border-bottom: 1px solid var(--border-color);
        padding-bottom: 5px;
      }

      .detail-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
        gap: 15px;

        .detail-item {
          label {
            display: block;
            font-size: 12px;
            color: var(--text-color-secondary);
            margin-bottom: 5px;
          }

          span {
            font-weight: 500;
            color: var(--text-color-primary);
            word-break: break-all;
          }
        }
      }

      .stack-trace,
      .metadata {
        background: var(--bg-color-tertiary);
        padding: 15px;
        border-radius: 6px;
        font-family: 'Monaco', 'Consolas', monospace;
        font-size: 12px;
        line-height: 1.4;
        color: var(--text-color-primary);
        overflow-x: auto;
        white-space: pre;
      }
    }
  }

  .modal-footer {
    display: flex;
    gap: 10px;
    justify-content: flex-end;
  }
}

// 레벨 배지 스타일
:global(.level-badge) {
  padding: 2px 6px;
  border-radius: 3px;
  font-size: 11px;
  font-weight: 500;

  &.level-debug { background: #e2e8f0; color: #4a5568; }
  &.level-info { background: #bee3f8; color: #2b6cb0; }
  &.level-warn { background: #feebc8; color: #c05621; }
  &.level-error { background: #fed7d7; color: #c53030; }
  &.level-fatal { background: #fed7d7; color: #c53030; font-weight: bold; }
}

// 액션 버튼 스타일
:global(.action-btn) {
  padding: 4px 8px;
  border: none;
  border-radius: 3px;
  font-size: 11px;
  cursor: pointer;
  margin-right: 5px;

  &.view {
    background: #4299e1;
    color: white;
  }

  &.delete {
    background: #f56565;
    color: white;
  }

  &:hover {
    opacity: 0.8;
  }
}

// 반응형 디자인
@media (max-width: 768px) {
  .error-tracking-dashboard {
    padding: 15px;

    .charts-section {
      grid-template-columns: 1fr;
    }

    .filter-controls {
      flex-direction: column;
      align-items: stretch;

      > * {
        width: 100% !important;
      }
    }
  }
}
</style>