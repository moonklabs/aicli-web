<template>
  <div class="trend-chart">
    <h4>{{ getMetricTitle(metric) }}</h4>
    <div class="chart-container" ref="chartContainer">
      <svg v-if="data.length > 0" :viewBox="viewBox">
        <!-- 그리드 라인 -->
        <g class="grid">
          <line
            v-for="i in 5"
            :key="`h-${i}`"
            :x1="padding"
            :y1="padding + (i - 1) * gridSpacing"
            :x2="width - padding"
            :y2="padding + (i - 1) * gridSpacing"
            stroke="var(--border-primary)"
            stroke-opacity="0.2"
          />
        </g>
        
        <!-- 임계값 라인 -->
        <g class="thresholds" v-if="threshold">
          <line
            :x1="padding"
            :y1="getY(threshold.good)"
            :x2="width - padding"
            :y2="getY(threshold.good)"
            stroke="var(--success-500)"
            stroke-width="2"
            stroke-dasharray="5,5"
            opacity="0.5"
          />
          <text
            :x="width - padding + 5"
            :y="getY(threshold.good)"
            fill="var(--success-500)"
            font-size="12"
            dominant-baseline="middle"
          >
            Good
          </text>
          
          <line
            :x1="padding"
            :y1="getY(threshold.poor)"
            :x2="width - padding"
            :y2="getY(threshold.poor)"
            stroke="var(--error-500)"
            stroke-width="2"
            stroke-dasharray="5,5"
            opacity="0.5"
          />
          <text
            :x="width - padding + 5"
            :y="getY(threshold.poor)"
            fill="var(--error-500)"
            font-size="12"
            dominant-baseline="middle"
          >
            Poor
          </text>
        </g>
        
        <!-- 데이터 라인 -->
        <path
          :d="linePath"
          fill="none"
          stroke="var(--primary-500)"
          stroke-width="2"
        />
        
        <!-- 데이터 포인트 -->
        <g class="data-points">
          <circle
            v-for="(point, index) in chartData"
            :key="index"
            :cx="point.x"
            :cy="point.y"
            r="4"
            :fill="getPointColor(point.value)"
            @mouseenter="showTooltip($event, point)"
            @mouseleave="hideTooltip"
          />
        </g>
      </svg>
      
      <div v-else class="no-data">
        데이터가 없습니다
      </div>
    </div>
    
    <!-- 툴팁 -->
    <div
      v-if="tooltip.visible"
      class="tooltip"
      :style="{
        left: `${tooltip.x}px`,
        top: `${tooltip.y}px`,
      }"
    >
      <div class="tooltip-content">
        <div class="tooltip-value">{{ formatValue(tooltip.value) }}</div>
        <div class="tooltip-time">{{ formatTime(tooltip.timestamp) }}</div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'

interface Props {
  metric: string
  data: Array<{
    timestamp: number
    value: number
  }>
  threshold?: {
    good: number
    poor: number
  }
}

const props = defineProps<Props>()

const chartContainer = ref<HTMLElement>()
const width = 400
const height = 200
const padding = 20
const viewBox = `0 0 ${width} ${height}`
const gridSpacing = (height - 2 * padding) / 4

const tooltip = ref({
  visible: false,
  x: 0,
  y: 0,
  value: 0,
  timestamp: 0,
})

// 메트릭 제목
const getMetricTitle = (metric: string) => {
  const titles: Record<string, string> = {
    lcp: 'LCP (Largest Contentful Paint)',
    fid: 'FID (First Input Delay)',
    cls: 'CLS (Cumulative Layout Shift)',
    fcp: 'FCP (First Contentful Paint)',
    inp: 'INP (Interaction to Next Paint)',
    ttfb: 'TTFB (Time to First Byte)',
  }
  return titles[metric] || metric.toUpperCase()
}

// 차트 데이터 계산
const chartData = computed(() => {
  if (props.data.length === 0) return []
  
  const sortedData = [...props.data].sort((a, b) => a.timestamp - b.timestamp)
  const minTime = sortedData[0].timestamp
  const maxTime = sortedData[sortedData.length - 1].timestamp
  const timeRange = maxTime - minTime || 1
  
  const values = sortedData.map(d => d.value)
  const minValue = Math.min(...values) * 0.9
  const maxValue = Math.max(...values) * 1.1
  const valueRange = maxValue - minValue || 1
  
  return sortedData.map(point => ({
    x: padding + ((point.timestamp - minTime) / timeRange) * (width - 2 * padding),
    y: height - padding - ((point.value - minValue) / valueRange) * (height - 2 * padding),
    value: point.value,
    timestamp: point.timestamp,
  }))
})

// Y 좌표 계산 (임계값용)
const getY = (value: number) => {
  if (chartData.value.length === 0) return 0
  
  const values = props.data.map(d => d.value)
  const minValue = Math.min(...values) * 0.9
  const maxValue = Math.max(...values) * 1.1
  const valueRange = maxValue - minValue || 1
  
  return height - padding - ((value - minValue) / valueRange) * (height - 2 * padding)
}

// SVG 경로 생성
const linePath = computed(() => {
  if (chartData.value.length === 0) return ''
  
  return chartData.value
    .map((point, index) => {
      return `${index === 0 ? 'M' : 'L'} ${point.x} ${point.y}`
    })
    .join(' ')
})

// 포인트 색상
const getPointColor = (value: number) => {
  if (!props.threshold) return 'var(--primary-500)'
  
  if (value <= props.threshold.good) return 'var(--success-500)'
  if (value <= props.threshold.poor) return 'var(--warning-500)'
  return 'var(--error-500)'
}

// 값 포맷팅
const formatValue = (value: number) => {
  if (props.metric === 'cls') {
    return value.toFixed(3)
  }
  return value >= 1000 ? `${(value / 1000).toFixed(2)}s` : `${Math.round(value)}ms`
}

// 시간 포맷팅
const formatTime = (timestamp: number) => {
  const date = new Date(timestamp)
  return date.toLocaleTimeString('ko-KR', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

// 툴팁 표시
const showTooltip = (event: MouseEvent, point: any) => {
  const rect = chartContainer.value?.getBoundingClientRect()
  if (!rect) return
  
  tooltip.value = {
    visible: true,
    x: point.x,
    y: point.y - 10,
    value: point.value,
    timestamp: point.timestamp,
  }
}

// 툴팁 숨기기
const hideTooltip = () => {
  tooltip.value.visible = false
}

// 리사이즈 처리
let resizeObserver: ResizeObserver

onMounted(() => {
  if (chartContainer.value) {
    resizeObserver = new ResizeObserver(() => {
      // 차트 크기 재계산 로직
    })
    resizeObserver.observe(chartContainer.value)
  }
})

onUnmounted(() => {
  resizeObserver?.disconnect()
})
</script>

<style scoped>
.trend-chart {
  position: relative;
  background: var(--bg-secondary);
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-lg);
  padding: var(--spacing-lg);
}

.trend-chart h4 {
  font-size: var(--text-base);
  font-weight: var(--font-semibold);
  color: var(--text-primary);
  margin: 0 0 var(--spacing-md) 0;
}

.chart-container {
  position: relative;
  width: 100%;
  height: 200px;
}

.chart-container svg {
  width: 100%;
  height: 100%;
}

.data-points circle {
  cursor: pointer;
  transition: r var(--duration-fast);
}

.data-points circle:hover {
  r: 6;
}

.no-data {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--text-tertiary);
  font-size: var(--text-sm);
}

.tooltip {
  position: absolute;
  pointer-events: none;
  z-index: var(--z-tooltip);
  transform: translate(-50%, -100%);
}

.tooltip-content {
  background: var(--bg-elevated);
  border: 1px solid var(--border-secondary);
  border-radius: var(--radius-md);
  padding: var(--spacing-sm) var(--spacing-md);
  box-shadow: var(--shadow-lg);
}

.tooltip-value {
  font-size: var(--text-base);
  font-weight: var(--font-semibold);
  color: var(--text-primary);
}

.tooltip-time {
  font-size: var(--text-xs);
  color: var(--text-secondary);
  margin-top: var(--spacing-xs);
}

/* 다크 모드 */
[data-theme='dark'] .trend-chart {
  background: var(--bg-tertiary);
}

/* 반응형 */
@media (max-width: 768px) {
  .chart-container {
    height: 150px;
  }
}
</style>