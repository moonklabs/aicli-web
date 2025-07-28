<template>
  <div
    :class="[
      'base-chart',
      {
        'loading': loading,
        'has-error': !!error
      }
    ]"
    :style="{
      width: width ? `${width}px` : '100%',
      height: height ? `${height}px` : '400px'
    }"
    role="img"
    :aria-label="accessibility?.description || `${chartType} 차트`"
    :aria-describedby="accessibility?.summary ? 'chart-summary' : undefined"
  >
    <!-- 차트 캔버스 -->
    <canvas
      ref="chartCanvas"
      :width="canvasWidth"
      :height="canvasHeight"
      @click="handleChartClick"
      @mousemove="handleChartHover"
      @mouseout="handleChartMouseOut"
    />

    <!-- 접근성 요약 -->
    <div
      v-if="accessibility?.summary"
      id="chart-summary"
      class="sr-only"
    >
      {{ accessibility.summary }}
    </div>

    <!-- 로딩 오버레이 -->
    <div v-if="loading" class="chart-loading">
      <div class="loading-spinner">
        <div class="spinner"></div>
        <span>차트를 로딩 중...</span>
      </div>
    </div>

    <!-- 에러 상태 -->
    <div v-if="error" class="chart-error">
      <div class="error-content">
        <h3>차트 로딩 실패</h3>
        <p>{{ error.message }}</p>
        <button @click="handleRetry" class="retry-btn">다시 시도</button>
      </div>
    </div>

    <!-- 툴바 (줌, 내보내기 등) -->
    <div v-if="showToolbar" class="chart-toolbar">
      <button
        v-if="zoom?.enabled"
        @click="resetZoom"
        :disabled="!isZoomed"
        class="toolbar-btn"
        title="줌 리셋"
      >
        🔍
      </button>

      <button
        v-if="exportOptions?.enabled"
        @click="showExportMenu = !showExportMenu"
        class="toolbar-btn"
        title="내보내기"
      >
        📁
      </button>

      <div v-if="showExportMenu" class="export-menu">
        <button
          v-for="format in availableExportFormats"
          :key="format"
          @click="exportChart(format)"
          class="export-option"
        >
          {{ format.toUpperCase() }}
        </button>
      </div>
    </div>

    <!-- 범례 (커스텀) -->
    <div v-if="showCustomLegend" class="custom-legend">
      <div
        v-for="(dataset, index) in chartData.datasets"
        :key="index"
        class="legend-item"
        @click="toggleDataset(index)"
        :class="{ hidden: hiddenDatasets.has(index) }"
      >
        <span
          class="legend-color"
          :style="{ backgroundColor: getDatasetColor(dataset, index) }"
        ></span>
        <span class="legend-label">{{ dataset.label }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  type PropType,
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  watch,
} from 'vue'
import {
  Chart,
  registerables,
} from 'chart.js'
import type { ChartConfiguration, ChartType } from 'chart.js'
import type {
  AdvancedChartProps,
  ChartData,
  ChartExportConfig,
  ChartTheme,
  ChartZoomConfig,
  RealTimeChartConfig,
} from '@/types/ui'

// Chart.js 플러그인 등록
Chart.register(...registerables)

interface Props extends AdvancedChartProps {
  chartType: ChartType
  showToolbar?: boolean
  showCustomLegend?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  responsive: true,
  maintainAspectRatio: false,
  redraw: false,
  showToolbar: true,
  showCustomLegend: false,
  loading: false,
  accessibility: () => ({ enabled: true }),
})

const emit = defineEmits<{
  'chart-create': [chart: Chart]
  'chart-update': [chart: Chart]
  'chart-destroy': [chart: Chart]
  'chart-click': [event: Event, elements: any[]]
  'chart-hover': [event: Event, elements: any[]]
  'error': [error: Error]
}>()

// 반응형 상태
const chartCanvas = ref<HTMLCanvasElement>()
const chartInstance = ref<Chart | null>(null)
const showExportMenu = ref(false)
const hiddenDatasets = ref(new Set<number>())
const isZoomed = ref(false)
const resizeObserver = ref<ResizeObserver | null>(null)

// 실시간 업데이트 관련
const realTimeInterval = ref<number | null>(null)
const dataBuffer = ref<any[]>([])

// 계산된 속성
const canvasWidth = computed(() => {
  return props.width || (chartCanvas.value?.parentElement?.clientWidth || 400)
})

const canvasHeight = computed(() => {
  return props.height || 400
})

const chartData = computed(() => {
  return props.data || { labels: [], datasets: [] }
})

const mergedOptions = computed(() => {
  const baseOptions = {
    responsive: props.responsive,
    maintainAspectRatio: props.maintainAspectRatio,
    plugins: {
      legend: {
        display: !props.showCustomLegend,
      },
      tooltip: {
        enabled: true,
        ...getTooltipTheme(),
      },
    },
    scales: getScalesConfiguration(),
    animation: getAnimationConfiguration(),
    ...props.options,
  }

  // 줌 플러그인 설정
  if (props.zoom?.enabled) {
    baseOptions.plugins.zoom = {
      pan: {
        enabled: true,
        mode: props.zoom.mode || 'xy',
      },
      zoom: {
        wheel: {
          enabled: true,
          speed: props.zoom.speed || 0.1,
        },
        pinch: {
          enabled: true,
        },
        mode: props.zoom.mode || 'xy',
        limits: {
          x: { min: props.zoom.rangeMin?.x, max: props.zoom.rangeMax?.x },
          y: { min: props.zoom.rangeMin?.y, max: props.zoom.rangeMax?.y },
        },
        onZoomComplete: ({ chart }) => {
          isZoomed.value = chart.isZoomedOrPanned()
          props.zoom?.onZoomComplete?.(chart)
        },
      },
    }
  }

  return baseOptions
})

const chartConfiguration = computed((): ChartConfiguration => ({
  type: props.chartType,
  data: chartData.value,
  options: mergedOptions.value,
  plugins: props.plugins || [],
}))

const availableExportFormats = computed(() => {
  return props.exportOptions?.formats || ['png', 'jpg', 'svg']
})

// 차트 생성 및 업데이트
const createChart = async () => {
  if (!chartCanvas.value) return

  try {
    // 기존 차트 제거
    if (chartInstance.value) {
      chartInstance.value.destroy()
    }

    // 새 차트 생성
    chartInstance.value = new Chart(chartCanvas.value, chartConfiguration.value)

    emit('chart-create', chartInstance.value)
    props.onChartCreate?.(chartInstance.value)

    // 실시간 업데이트 시작
    if (props.realTime?.enabled) {
      startRealTimeUpdates()
    }

  } catch (error) {
    console.error('Chart creation failed:', error)
    emit('error', error as Error)
    props.onError?.(error as Error)
  }
}

const updateChart = () => {
  if (!chartInstance.value) return

  try {
    // 데이터 업데이트
    chartInstance.value.data = chartData.value
    chartInstance.value.options = mergedOptions.value

    if (props.redraw) {
      chartInstance.value.update('none')
    } else {
      chartInstance.value.update()
    }

    emit('chart-update', chartInstance.value)
    props.onChartUpdate?.(chartInstance.value)

  } catch (error) {
    console.error('Chart update failed:', error)
    emit('error', error as Error)
    props.onError?.(error as Error)
  }
}

const destroyChart = () => {
  if (chartInstance.value) {
    emit('chart-destroy', chartInstance.value)
    props.onChartDestroy?.(chartInstance.value)

    chartInstance.value.destroy()
    chartInstance.value = null
  }

  stopRealTimeUpdates()
}

// 테마 관련 함수들
const getTooltipTheme = () => {
  if (!props.theme) return {}

  return {
    backgroundColor: props.theme.tooltip.backgroundColor,
    titleColor: props.theme.tooltip.titleColor,
    bodyColor: props.theme.tooltip.bodyColor,
    borderColor: props.theme.tooltip.borderColor,
    borderWidth: 1,
  }
}

const getScalesConfiguration = () => {
  if (!props.theme) return {}

  return {
    x: {
      grid: {
        color: props.theme.grid.color,
        lineWidth: props.theme.grid.lineWidth,
      },
      ticks: {
        color: props.theme.fonts.family,
        font: {
          family: props.theme.fonts.family,
          size: props.theme.fonts.size,
          weight: props.theme.fonts.weight,
        },
      },
    },
    y: {
      grid: {
        color: props.theme.grid.color,
        lineWidth: props.theme.grid.lineWidth,
      },
      ticks: {
        color: props.theme.fonts.family,
        font: {
          family: props.theme.fonts.family,
          size: props.theme.fonts.size,
          weight: props.theme.fonts.weight,
        },
      },
    },
  }
}

const getAnimationConfiguration = () => {
  return {
    duration: props.realTime?.animationDuration || 750,
    easing: 'easeInOutQuart',
  }
}

const getDatasetColor = (dataset: any, index: number): string => {
  if (dataset.backgroundColor) {
    return Array.isArray(dataset.backgroundColor)
      ? dataset.backgroundColor[0]
      : dataset.backgroundColor
  }

  // 기본 색상 팔레트
  const colors = props.theme?.colors.primary || [
    '#3b82f6', '#ef4444', '#10b981', '#f59e0b',
    '#8b5cf6', '#06b6d4', '#ec4899', '#84cc16',
  ]

  return colors[index % colors.length]
}

// 이벤트 핸들러들
const handleChartClick = (event: Event) => {
  if (!chartInstance.value) return

  const elements = chartInstance.value.getElementsAtEventForMode(
    event as any,
    'nearest',
    { intersect: true },
    true,
  )

  emit('chart-click', event, elements)
  props.onClick?.(event, elements)
}

const handleChartHover = (event: Event) => {
  if (!chartInstance.value) return

  const elements = chartInstance.value.getElementsAtEventForMode(
    event as any,
    'nearest',
    { intersect: false },
    true,
  )

  emit('chart-hover', event, elements)
  props.onHover?.(event, elements)
}

const handleChartMouseOut = () => {
  // 호버 상태 리셋
}

const handleRetry = () => {
  createChart()
}

// 차트 기능들
const resetZoom = () => {
  if (chartInstance.value && props.zoom?.enabled) {
    chartInstance.value.resetZoom()
    isZoomed.value = false
  }
}

const toggleDataset = (index: number) => {
  if (!chartInstance.value) return

  const meta = chartInstance.value.getDatasetMeta(index)
  meta.hidden = !meta.hidden

  if (meta.hidden) {
    hiddenDatasets.value.add(index)
  } else {
    hiddenDatasets.value.delete(index)
  }

  chartInstance.value.update()
}

const exportChart = (format: string) => {
  if (!chartInstance.value) return

  try {
    const canvas = chartInstance.value.canvas
    const filename = props.exportOptions?.filename || `chart.${format}`

    if (format === 'png' || format === 'jpg') {
      const imageData = canvas.toDataURL(`image/${format}`, props.exportOptions?.quality || 0.8)
      downloadImage(imageData, filename)
    } else if (format === 'svg') {
      // SVG 내보내기는 추가 라이브러리 필요
      console.warn('SVG export not implemented yet')
    }

    showExportMenu.value = false
  } catch (error) {
    console.error('Export failed:', error)
    emit('error', error as Error)
  }
}

const downloadImage = (dataUrl: string, filename: string) => {
  const link = document.createElement('a')
  link.download = filename
  link.href = dataUrl
  link.click()
}

// 실시간 업데이트
const startRealTimeUpdates = () => {
  if (!props.realTime?.enabled || !props.realTime.interval) return

  realTimeInterval.value = setInterval(() => {
    // 실시간 데이터 업데이트 로직
    props.realTime?.onDataUpdate?.(chartInstance.value)

    // 최대 데이터 포인트 수 제한
    if (props.realTime.maxDataPoints && chartData.value.labels) {
      while (chartData.value.labels.length > props.realTime.maxDataPoints) {
        chartData.value.labels.shift()
        chartData.value.datasets.forEach(dataset => {
          if (Array.isArray(dataset.data)) {
            dataset.data.shift()
          }
        })
      }
    }

    updateChart()
  }, props.realTime.interval)
}

const stopRealTimeUpdates = () => {
  if (realTimeInterval.value) {
    clearInterval(realTimeInterval.value)
    realTimeInterval.value = null
  }
}

// 반응형 리사이즈
const setupResizeObserver = () => {
  if (!chartCanvas.value?.parentElement) return

  resizeObserver.value = new ResizeObserver(() => {
    nextTick(() => {
      if (chartInstance.value && props.responsive) {
        chartInstance.value.resize()
      }
    })
  })

  resizeObserver.value.observe(chartCanvas.value.parentElement)
}

const cleanupResizeObserver = () => {
  if (resizeObserver.value) {
    resizeObserver.value.disconnect()
    resizeObserver.value = null
  }
}

// 라이프사이클
onMounted(() => {
  nextTick(() => {
    createChart()
    setupResizeObserver()
  })
})

onBeforeUnmount(() => {
  destroyChart()
  cleanupResizeObserver()
})

// 반응형 업데이트
watch(
  () => chartConfiguration.value,
  () => {
    if (chartInstance.value) {
      updateChart()
    }
  },
  { deep: true },
)

watch(
  () => props.loading,
  (newLoading) => {
    if (newLoading) {
      destroyChart()
    } else {
      nextTick(() => {
        createChart()
      })
    }
  },
)

// 외부에서 접근 가능한 메서드들
defineExpose({
  chartInstance: computed(() => chartInstance.value),
  resetZoom,
  exportChart,
  updateChart,
  createChart,
  destroyChart,
})
</script>

<style scoped lang="scss">
.base-chart {
  position: relative;
  display: flex;
  flex-direction: column;
  background: white;
  border-radius: 8px;
  overflow: hidden;

  &.loading {
    .chart-toolbar {
      opacity: 0.5;
      pointer-events: none;
    }
  }

  &.has-error {
    justify-content: center;
    align-items: center;
    min-height: 200px;
  }
}

canvas {
  display: block;
  width: 100% !important;
  height: 100% !important;
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}

.chart-loading {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(255, 255, 255, 0.9);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10;
}

.loading-spinner {
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

.chart-error {
  padding: 2rem;
  text-align: center;

  .error-content {
    max-width: 300px;
    margin: 0 auto;

    h3 {
      margin: 0 0 0.5rem 0;
      color: #dc2626;
      font-size: 1.125rem;
    }

    p {
      margin: 0 0 1rem 0;
      color: #6b7280;
      font-size: 0.875rem;
    }

    .retry-btn {
      padding: 0.5rem 1rem;
      background: #3b82f6;
      color: white;
      border: none;
      border-radius: 4px;
      cursor: pointer;
      font-size: 0.875rem;

      &:hover {
        background: #2563eb;
      }
    }
  }
}

.chart-toolbar {
  position: absolute;
  top: 1rem;
  right: 1rem;
  display: flex;
  gap: 0.5rem;
  z-index: 5;
}

.toolbar-btn {
  width: 36px;
  height: 36px;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  background: white;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1rem;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);

  &:hover {
    background: #f9fafb;
    border-color: #d1d5db;
  }

  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
}

.export-menu {
  position: absolute;
  top: 100%;
  right: 0;
  margin-top: 0.25rem;
  background: white;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.1);
  overflow: hidden;
  z-index: 10;

  .export-option {
    display: block;
    width: 100%;
    padding: 0.5rem 1rem;
    border: none;
    background: white;
    text-align: left;
    cursor: pointer;
    font-size: 0.875rem;

    &:hover {
      background: #f3f4f6;
    }
  }
}

.custom-legend {
  padding: 1rem;
  border-top: 1px solid #e5e7eb;
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  transition: background-color 0.2s;

  &:hover {
    background: #f3f4f6;
  }

  &.hidden {
    opacity: 0.5;
    text-decoration: line-through;
  }
}

.legend-color {
  width: 12px;
  height: 12px;
  border-radius: 2px;
  flex-shrink: 0;
}

.legend-label {
  font-size: 0.875rem;
  color: #374151;
}

// 반응형 스타일
@media (max-width: 768px) {
  .chart-toolbar {
    top: 0.5rem;
    right: 0.5rem;
  }

  .toolbar-btn {
    width: 32px;
    height: 32px;
    font-size: 0.875rem;
  }

  .custom-legend {
    padding: 0.75rem;
    font-size: 0.8125rem;
  }

  .legend-color {
    width: 10px;
    height: 10px;
  }
}

// 다크 테마 지원
@media (prefers-color-scheme: dark) {
  .base-chart {
    background: #1f2937;
  }

  .chart-loading {
    background: rgba(31, 41, 55, 0.9);
  }

  .loading-spinner span {
    color: #d1d5db;
  }

  .chart-error .error-content {
    h3 {
      color: #f87171;
    }

    p {
      color: #9ca3af;
    }
  }

  .toolbar-btn {
    background: #374151;
    border-color: #4b5563;
    color: #f3f4f6;

    &:hover {
      background: #4b5563;
      border-color: #6b7280;
    }
  }

  .export-menu {
    background: #374151;
    border-color: #4b5563;

    .export-option {
      background: #374151;
      color: #f3f4f6;

      &:hover {
        background: #4b5563;
      }
    }
  }

  .custom-legend {
    border-top-color: #4b5563;
  }

  .legend-item {
    &:hover {
      background: #374151;
    }
  }

  .legend-label {
    color: #e5e7eb;
  }
}
</style>
