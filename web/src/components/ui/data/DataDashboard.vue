<template>
  <div class="data-dashboard" :class="{ 'loading': loading }">
    <!-- 대시보드 헤더 -->
    <div class="dashboard-header">
      <div class="title-section">
        <h2 v-if="title" class="dashboard-title">{{ title }}</h2>
        <p v-if="description" class="dashboard-description">{{ description }}</p>
      </div>

      <div class="controls-section">
        <!-- 레이아웃 토글 -->
        <div class="layout-controls">
          <button
            v-for="layout in availableLayouts"
            :key="layout.value"
            @click="currentLayout = layout.value"
            :class="['layout-btn', { active: currentLayout === layout.value }]"
            :title="layout.label"
          >
            {{ layout.icon }}
          </button>
        </div>

        <!-- 새로고침 버튼 -->
        <button
          @click="refreshData"
          :disabled="loading"
          class="refresh-btn"
          title="데이터 새로고침"
        >
          🔄
        </button>

        <!-- 전체화면 토글 -->
        <button
          @click="toggleFullscreen"
          class="fullscreen-btn"
          title="전체화면"
        >
          {{ isFullscreen ? '🗗' : '🗖' }}
        </button>
      </div>
    </div>

    <!-- 메인 콘텐츠 -->
    <div
      ref="dashboardContent"
      :class="['dashboard-content', `layout-${currentLayout}`]"
    >
      <!-- 차트 섹션 -->
      <div
        v-if="showChart"
        class="chart-section"
        :class="{ 'chart-loading': chartLoading }"
      >
        <div class="section-header">
          <h3>{{ chartTitle || '데이터 시각화' }}</h3>
          <div class="chart-controls">
            <!-- 차트 타입 선택 -->
            <select
              v-model="currentChartType"
              class="chart-type-selector"
              @change="handleChartTypeChange"
            >
              <option
                v-for="type in availableChartTypes"
                :key="type.value"
                :value="type.value"
              >
                {{ type.label }}
              </option>
            </select>

            <!-- 차트 설정 버튼 -->
            <button
              @click="showChartSettings = !showChartSettings"
              class="settings-btn"
              title="차트 설정"
            >
              ⚙️
            </button>
          </div>
        </div>

        <!-- 차트 설정 패널 -->
        <div v-if="showChartSettings" class="chart-settings-panel">
          <div class="settings-grid">
            <label class="setting-item">
              <span>X축 컬럼:</span>
              <select v-model="chartConfig.xAxis">
                <option
                  v-for="column in numericColumns"
                  :key="column.key"
                  :value="column.key"
                >
                  {{ column.title }}
                </option>
              </select>
            </label>

            <label class="setting-item">
              <span>Y축 컬럼:</span>
              <select v-model="chartConfig.yAxis">
                <option
                  v-for="column in numericColumns"
                  :key="column.key"
                  :value="column.key"
                >
                  {{ column.title }}
                </option>
              </select>
            </label>

            <label class="setting-item">
              <span>그룹 컬럼:</span>
              <select v-model="chartConfig.groupBy">
                <option value="">없음</option>
                <option
                  v-for="column in categoricalColumns"
                  :key="column.key"
                  :value="column.key"
                >
                  {{ column.title }}
                </option>
              </select>
            </label>

            <label class="setting-item">
              <input
                type="checkbox"
                v-model="chartConfig.animated"
              />
              <span>애니메이션</span>
            </label>
          </div>
        </div>

        <!-- 동적 차트 컴포넌트 -->
        <component
          :is="currentChartComponent"
          :data="chartData"
          :options="chartOptions"
          :loading="chartLoading"
          :theme="chartTheme"
          :table-integration="tableIntegration"
          @chart-click="handleChartClick"
          @chart-hover="handleChartHover"
          @error="handleChartError"
        />
      </div>

      <!-- 테이블 섹션 -->
      <div
        v-if="showTable"
        class="table-section"
        :class="{ 'table-loading': tableLoading }"
      >
        <div class="section-header">
          <h3>{{ tableTitle || '데이터 테이블' }}</h3>
          <div class="table-controls">
            <!-- 컬럼 표시/숨김 -->
            <button
              @click="showColumnManager = !showColumnManager"
              class="column-btn"
              title="컬럼 관리"
            >
              📋
            </button>

            <!-- 데이터 내보내기 -->
            <button
              @click="exportData"
              :disabled="!filteredData.length"
              class="export-btn"
              title="데이터 내보내기"
            >
              📥
            </button>
          </div>
        </div>

        <!-- 컬럼 관리 패널 -->
        <div v-if="showColumnManager" class="column-manager">
          <div class="column-list">
            <label
              v-for="column in tableColumns"
              :key="column.key"
              class="column-toggle"
            >
              <input
                type="checkbox"
                :checked="!hiddenColumns.has(column.key)"
                @change="toggleColumn(column.key, $event.target.checked)"
              />
              <span>{{ column.title }}</span>
            </label>
          </div>
        </div>

        <!-- 데이터 테이블 -->
        <BaseDataTable
          :data="filteredData"
          :columns="visibleColumns"
          :loading="tableLoading"
          :virtual-scroll="virtualScrollConfig"
          :selection="selectionConfig"
          :pagination="paginationConfig"
          :filters="currentFilters"
          :sorters="currentSorters"
          :global-search="true"
          :show-filters="true"
          :sticky-header="true"
          @update:checkedRowKeys="handleSelectionChange"
          @update:filters="handleFiltersChange"
          @update:sorter="handleSorterChange"
          @row-click="handleRowClick"
          @row-double-click="handleRowDoubleClick"
          @cell-click="handleCellClick"
        />
      </div>

      <!-- 통계 패널 (사이드바 레이아웃에서만) -->
      <div
        v-if="currentLayout === 'sidebar' && showStats"
        class="stats-section"
      >
        <div class="section-header">
          <h3>통계 정보</h3>
        </div>

        <div class="stats-grid">
          <div
            v-for="stat in computedStats"
            :key="stat.key"
            class="stat-card"
          >
            <div class="stat-label">{{ stat.label }}</div>
            <div class="stat-value">{{ stat.value }}</div>
            <div v-if="stat.change" class="stat-change" :class="stat.changeType">
              {{ stat.change }}
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 로딩 오버레이 -->
    <div v-if="loading" class="loading-overlay">
      <div class="loading-content">
        <div class="spinner"></div>
        <span>데이터를 로딩 중...</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  watch,
} from 'vue'
import BaseDataTable from './BaseDataTable.vue'
import { BarChart, LineChart, PieChart, ScatterChart } from '../charts'
import type {
  AdvancedTableColumn,
  ChartData,
  ChartTableIntegration,
  ChartTheme,
  TableFilter,
  TableSorter,
} from '@/types/ui'

interface DashboardProps {
  // 데이터
  data?: any[]
  columns?: AdvancedTableColumn[]

  // 제목 및 설명
  title?: string
  description?: string
  chartTitle?: string
  tableTitle?: string

  // 표시 옵션
  showChart?: boolean
  showTable?: boolean
  showStats?: boolean

  // 초기 설정
  defaultLayout?: 'horizontal' | 'vertical' | 'sidebar' | 'tabs'
  defaultChartType?: 'line' | 'bar' | 'pie' | 'scatter'

  // 로딩 상태
  loading?: boolean
  chartLoading?: boolean
  tableLoading?: boolean

  // 테마
  theme?: ChartTheme

  // 콜백
  onDataRefresh?: () => Promise<void>
  onDataExport?: (data: any[], format: string) => void
}

const props = withDefaults(defineProps<DashboardProps>(), {
  data: () => [],
  columns: () => [],
  showChart: true,
  showTable: true,
  showStats: false,
  defaultLayout: 'horizontal',
  defaultChartType: 'line',
  loading: false,
  chartLoading: false,
  tableLoading: false,
})

const emit = defineEmits<{
  'selection-change': [selectedRows: any[]]
  'filter-change': [filters: TableFilter[]]
  'chart-interaction': [event: string, data: any]
}>()

// 반응형 상태
const dashboardContent = ref<HTMLElement>()
const currentLayout = ref(props.defaultLayout)
const currentChartType = ref(props.defaultChartType)
const isFullscreen = ref(false)

const showChartSettings = ref(false)
const showColumnManager = ref(false)
const hiddenColumns = ref(new Set<string>())
const selectedRowKeys = ref<Array<string | number>>([])

const currentFilters = ref<TableFilter[]>([])
const currentSorters = ref<TableSorter[]>([])

// 차트 설정
const chartConfig = ref({
  xAxis: '',
  yAxis: '',
  groupBy: '',
  animated: true,
})

// 레이아웃 옵션
const availableLayouts = [
  { value: 'horizontal', label: '가로 분할', icon: '⬌' },
  { value: 'vertical', label: '세로 분할', icon: '⬍' },
  { value: 'sidebar', label: '사이드바', icon: '◫' },
  { value: 'tabs', label: '탭', icon: '⎶' },
]

// 차트 타입 옵션
const availableChartTypes = [
  { value: 'line', label: '선 그래프' },
  { value: 'bar', label: '막대 그래프' },
  { value: 'pie', label: '파이 차트' },
  { value: 'scatter', label: '산점도' },
]

// 계산된 속성들
const filteredData = computed(() => {
  let result = [...props.data]

  // 필터 적용
  currentFilters.value.forEach(filter => {
    result = result.filter(item => {
      const cellValue = item[filter.key]
      return applyFilter(cellValue, filter)
    })
  })

  return result
})

const visibleColumns = computed(() => {
  return props.columns.filter(col => !hiddenColumns.value.has(col.key))
})

const numericColumns = computed(() => {
  return props.columns.filter(col => {
    const sample = props.data[0]
    return sample && typeof sample[col.key] === 'number'
  })
})

const categoricalColumns = computed(() => {
  return props.columns.filter(col => {
    const sample = props.data[0]
    return sample && typeof sample[col.key] === 'string'
  })
})

const selectedRows = computed(() => {
  return filteredData.value.filter((_, index) =>
    selectedRowKeys.value.includes(index),
  )
})

// 차트 관련 계산 속성
const currentChartComponent = computed(() => {
  const componentMap = {
    line: LineChart,
    bar: BarChart,
    pie: PieChart,
    scatter: ScatterChart,
  }
  return componentMap[currentChartType.value]
})

const chartData = computed((): ChartData => {
  const data = selectedRows.value.length > 0 ? selectedRows.value : filteredData.value

  if (!data.length || !chartConfig.value.xAxis || !chartConfig.value.yAxis) {
    return { labels: [], datasets: [] }
  }

  if (chartConfig.value.groupBy) {
    // 그룹별 데이터 처리
    return generateGroupedChartData(data)
  } else {
    // 단일 데이터셋 처리
    return generateSimpleChartData(data)
  }
})

const chartOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  animation: {
    duration: chartConfig.value.animated ? 750 : 0,
  },
}))

const chartTheme = computed(() => props.theme)

const tableIntegration = computed((): ChartTableIntegration => ({
  enabled: true,
  syncSelection: true,
  syncFiltering: true,
  highlightOnHover: true,
  onSelectionSync: (selection) => {
    // 차트에서 테이블 선택 동기화
  },
  onFilterSync: (filters) => {
    currentFilters.value = filters
  },
}))

// 가상 스크롤 설정
const virtualScrollConfig = computed(() => ({
  enabled: filteredData.value.length > 1000,
  itemHeight: 40,
  overscan: 5,
}))

// 선택 설정
const selectionConfig = computed(() => ({
  type: 'checkbox' as const,
  selectedKeys: selectedRowKeys.value,
  onSelectionChange: (keys: Array<string | number>) => {
    selectedRowKeys.value = keys
  },
}))

// 페이지네이션 설정
const paginationConfig = computed(() => ({
  page: 1,
  pageSize: 50,
  total: filteredData.value.length,
  showSizeChanger: true,
  pageSizes: [20, 50, 100, 200],
}))

// 통계 정보
const computedStats = computed(() => {
  if (!filteredData.value.length) return []

  const stats = []

  // 총 행 수
  stats.push({
    key: 'total',
    label: '총 데이터',
    value: filteredData.value.length.toLocaleString(),
  })

  // 선택된 행 수
  if (selectedRows.value.length > 0) {
    stats.push({
      key: 'selected',
      label: '선택된 데이터',
      value: selectedRows.value.length.toLocaleString(),
    })
  }

  // 숫자 컬럼에 대한 통계
  numericColumns.value.forEach(column => {
    const values = filteredData.value
      .map(row => row[column.key])
      .filter(val => typeof val === 'number' && !isNaN(val))

    if (values.length > 0) {
      const sum = values.reduce((a, b) => a + b, 0)
      const avg = sum / values.length

      stats.push({
        key: `${column.key}_avg`,
        label: `${column.title} 평균`,
        value: avg.toLocaleString(undefined, { maximumFractionDigits: 2 }),
      })
    }
  })

  return stats
})

// 이벤트 핸들러들
const refreshData = async () => {
  if (props.onDataRefresh) {
    await props.onDataRefresh()
  }
}

const toggleFullscreen = () => {
  isFullscreen.value = !isFullscreen.value

  if (isFullscreen.value) {
    document.documentElement.requestFullscreen?.()
  } else {
    document.exitFullscreen?.()
  }
}

const handleChartTypeChange = () => {
  // 차트 타입에 따른 설정 초기화
  if (currentChartType.value === 'pie') {
    chartConfig.value.xAxis = ''
    chartConfig.value.yAxis = numericColumns.value[0]?.key || ''
  }
}

const toggleColumn = (columnKey: string, visible: boolean) => {
  if (visible) {
    hiddenColumns.value.delete(columnKey)
  } else {
    hiddenColumns.value.add(columnKey)
  }
}

const exportData = () => {
  if (props.onDataExport) {
    props.onDataExport(selectedRows.value.length > 0 ? selectedRows.value : filteredData.value, 'csv')
  } else {
    // 기본 CSV 내보내기
    downloadAsCSV(filteredData.value, 'data.csv')
  }
}

const handleSelectionChange = (keys: Array<string | number>) => {
  selectedRowKeys.value = keys
  emit('selection-change', selectedRows.value)
}

const handleFiltersChange = (filters: TableFilter[]) => {
  currentFilters.value = filters
  emit('filter-change', filters)
}

const handleSorterChange = (sorter: TableSorter | null) => {
  currentSorters.value = sorter ? [sorter] : []
}

const handleRowClick = (row: any, index: number, event: Event) => {
  // 행 클릭 처리
}

const handleRowDoubleClick = (row: any, index: number, event: Event) => {
  // 행 더블클릭 처리
}

const handleCellClick = (cell: any, row: any, column: AdvancedTableColumn, event: Event) => {
  // 셀 클릭 처리
}

const handleChartClick = (event: Event, elements: any[]) => {
  emit('chart-interaction', 'click', { event, elements })
}

const handleChartHover = (event: Event, elements: any[]) => {
  emit('chart-interaction', 'hover', { event, elements })
}

const handleChartError = (error: Error) => {
  console.error('Chart error:', error)
}

// 유틸리티 함수들
const generateSimpleChartData = (data: any[]): ChartData => {
  const xKey = chartConfig.value.xAxis
  const yKey = chartConfig.value.yAxis

  if (currentChartType.value === 'pie') {
    // 파이 차트의 경우 그룹별 집계
    const grouped = data.reduce((acc, item) => {
      const key = item[yKey] || 'Unknown'
      acc[key] = (acc[key] || 0) + 1
      return acc
    }, {})

    return {
      labels: Object.keys(grouped),
      datasets: [{
        label: '데이터 분포',
        data: Object.values(grouped),
      }],
    }
  }

  return {
    labels: data.map(item => item[xKey]),
    datasets: [{
      label: yKey,
      data: currentChartType.value === 'scatter'
        ? data.map(item => ({ x: item[xKey], y: item[yKey] }))
        : data.map(item => item[yKey]),
    }],
  }
}

const generateGroupedChartData = (data: any[]): ChartData => {
  const xKey = chartConfig.value.xAxis
  const yKey = chartConfig.value.yAxis
  const groupKey = chartConfig.value.groupBy

  const grouped = data.reduce((acc, item) => {
    const group = item[groupKey] || 'Unknown'
    if (!acc[group]) acc[group] = []
    acc[group].push(item)
    return acc
  }, {})

  const labels = [...new Set(data.map(item => item[xKey]))].sort()
  const datasets = Object.entries(grouped).map(([group, items]: [string, any[]]) => ({
    label: group,
    data: labels.map(label => {
      const found = items.find(item => item[xKey] === label)
      return found ? found[yKey] : 0
    }),
  }))

  return { labels, datasets }
}

const applyFilter = (value: any, filter: TableFilter): boolean => {
  if (value === null || value === undefined) return false

  const filterValue = filter.value
  const stringValue = String(value).toLowerCase()
  const stringFilterValue = String(filterValue).toLowerCase()

  switch (filter.operator) {
    case 'equals':
      return value === filterValue
    case 'contains':
      return stringValue.includes(stringFilterValue)
    case 'startsWith':
      return stringValue.startsWith(stringFilterValue)
    case 'endsWith':
      return stringValue.endsWith(stringFilterValue)
    case 'gt':
      return Number(value) > Number(filterValue)
    case 'gte':
      return Number(value) >= Number(filterValue)
    case 'lt':
      return Number(value) < Number(filterValue)
    case 'lte':
      return Number(value) <= Number(filterValue)
    default:
      return true
  }
}

const downloadAsCSV = (data: any[], filename: string) => {
  const headers = visibleColumns.value.map(col => col.title).join(',')
  const rows = data.map(row =>
    visibleColumns.value.map(col => row[col.key] || '').join(','),
  ).join('\n')

  const csv = `${headers}\n${rows}`
  const blob = new Blob([csv], { type: 'text/csv' })
  const url = URL.createObjectURL(blob)

  const link = document.createElement('a')
  link.href = url
  link.download = filename
  link.click()

  URL.revokeObjectURL(url)
}

// 라이프사이클
onMounted(() => {
  // 초기 차트 설정
  if (numericColumns.value.length >= 2) {
    chartConfig.value.xAxis = numericColumns.value[0].key
    chartConfig.value.yAxis = numericColumns.value[1].key
  }
})
</script>

<style scoped lang="scss">
.data-dashboard {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: white;
  border-radius: 8px;
  overflow: hidden;
  position: relative;

  &.loading {
    .dashboard-content {
      opacity: 0.6;
      pointer-events: none;
    }
  }
}

.dashboard-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 1.5rem;
  border-bottom: 1px solid #e5e7eb;
  background: #f9fafb;
}

.title-section {
  .dashboard-title {
    margin: 0 0 0.5rem 0;
    font-size: 1.5rem;
    font-weight: 700;
    color: #1f2937;
  }

  .dashboard-description {
    margin: 0;
    color: #6b7280;
    font-size: 0.875rem;
  }
}

.controls-section {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.layout-controls {
  display: flex;
  gap: 0.25rem;
  background: white;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  padding: 0.25rem;
}

.layout-btn {
  padding: 0.5rem;
  border: none;
  background: transparent;
  cursor: pointer;
  border-radius: 4px;
  font-size: 0.875rem;

  &:hover {
    background: #f3f4f6;
  }

  &.active {
    background: #3b82f6;
    color: white;
  }
}

.refresh-btn,
.fullscreen-btn {
  padding: 0.5rem;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  background: white;
  cursor: pointer;
  font-size: 0.875rem;

  &:hover {
    background: #f3f4f6;
  }

  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
}

.dashboard-content {
  flex: 1;
  display: flex;
  overflow: hidden;

  &.layout-horizontal {
    flex-direction: row;

    .chart-section,
    .table-section {
      flex: 1;
      min-width: 0;
    }
  }

  &.layout-vertical {
    flex-direction: column;

    .chart-section,
    .table-section {
      flex: 1;
      min-height: 0;
    }
  }

  &.layout-sidebar {
    flex-direction: row;

    .chart-section {
      flex: 2;
    }

    .table-section {
      flex: 3;
    }

    .stats-section {
      flex: 0 0 250px;
      border-left: 1px solid #e5e7eb;
    }
  }

  &.layout-tabs {
    // 탭 레이아웃은 추후 구현
  }
}

.chart-section,
.table-section,
.stats-section {
  display: flex;
  flex-direction: column;
  padding: 1rem;
  border-right: 1px solid #e5e7eb;

  &:last-child {
    border-right: none;
  }
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;

  h3 {
    margin: 0;
    font-size: 1.125rem;
    font-weight: 600;
    color: #1f2937;
  }
}

.chart-controls,
.table-controls {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.chart-type-selector {
  padding: 0.5rem;
  border: 1px solid #e5e7eb;
  border-radius: 4px;
  background: white;
  font-size: 0.875rem;
}

.settings-btn,
.column-btn,
.export-btn {
  padding: 0.5rem;
  border: 1px solid #e5e7eb;
  border-radius: 4px;
  background: white;
  cursor: pointer;
  font-size: 0.875rem;

  &:hover {
    background: #f3f4f6;
  }

  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
}

.chart-settings-panel,
.column-manager {
  padding: 1rem;
  background: #f9fafb;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  margin-bottom: 1rem;
}

.settings-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1rem;
}

.setting-item {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;

  span {
    font-size: 0.875rem;
    font-weight: 500;
    color: #374151;
  }

  select,
  input[type="checkbox"] {
    padding: 0.5rem;
    border: 1px solid #e5e7eb;
    border-radius: 4px;
  }

  input[type="checkbox"] {
    width: auto;
    margin-right: 0.5rem;
  }
}

.column-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 0.5rem;
}

.column-toggle {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.875rem;

  input[type="checkbox"] {
    margin: 0;
  }
}

.stats-grid {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.stat-card {
  padding: 1rem;
  background: white;
  border: 1px solid #e5e7eb;
  border-radius: 6px;

  .stat-label {
    font-size: 0.75rem;
    color: #6b7280;
    margin-bottom: 0.25rem;
  }

  .stat-value {
    font-size: 1.25rem;
    font-weight: 700;
    color: #1f2937;
  }

  .stat-change {
    font-size: 0.75rem;
    margin-top: 0.25rem;

    &.positive {
      color: #10b981;
    }

    &.negative {
      color: #ef4444;
    }
  }
}

.loading-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(255, 255, 255, 0.9);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 20;
}

.loading-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;

  .spinner {
    width: 40px;
    height: 40px;
    border: 4px solid #f3f4f6;
    border-top: 4px solid #3b82f6;
    border-radius: 50%;
    animation: spin 1s linear infinite;
  }

  span {
    font-size: 0.875rem;
    color: #6b7280;
  }
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

// 반응형 스타일
@media (max-width: 1024px) {
  .dashboard-content.layout-horizontal {
    flex-direction: column;
  }

  .dashboard-content.layout-sidebar {
    flex-direction: column;

    .stats-section {
      flex: none;
      border-left: none;
      border-top: 1px solid #e5e7eb;
    }
  }
}

@media (max-width: 768px) {
  .dashboard-header {
    flex-direction: column;
    gap: 1rem;
    align-items: stretch;
  }

  .controls-section {
    justify-content: space-between;
  }

  .settings-grid {
    grid-template-columns: 1fr;
  }

  .column-list {
    grid-template-columns: 1fr;
  }
}

// 다크 테마 지원
@media (prefers-color-scheme: dark) {
  .data-dashboard {
    background: #1f2937;
  }

  .dashboard-header {
    background: #374151;
    border-bottom-color: #4b5563;
  }

  .dashboard-title {
    color: #f9fafb;
  }

  .dashboard-description {
    color: #d1d5db;
  }

  .layout-controls {
    background: #374151;
    border-color: #4b5563;
  }

  .layout-btn {
    color: #e5e7eb;

    &:hover {
      background: #4b5563;
    }

    &.active {
      background: #3b82f6;
      color: white;
    }
  }

  .section-header h3 {
    color: #f9fafb;
  }

  .chart-settings-panel,
  .column-manager {
    background: #374151;
    border-color: #4b5563;
  }

  .stat-card {
    background: #374151;
    border-color: #4b5563;

    .stat-value {
      color: #f9fafb;
    }
  }
}
</style>