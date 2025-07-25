<template>
  <BaseChart
    v-bind="$attrs"
    chart-type="line"
    :data="processedData"
    :options="mergedOptions"
    v-on="$listeners"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import BaseChart from './BaseChart.vue'
import type { ChartData, ChartDataset, LineChartProps } from '@/types/ui'

interface Props extends LineChartProps {
  // LineChart 특화 속성들
}

const props = withDefaults(defineProps<Props>(), {
  stepped: false,
  spanGaps: false,
  showLine: true,
  tension: 0.2,
})

// 라인 차트에 특화된 데이터 처리
const processedData = computed((): ChartData => {
  const data = { ...props.data }

  // 데이터셋에 라인 차트 특화 설정 적용
  data.datasets = data.datasets.map((dataset: ChartDataset) => ({
    ...dataset,
    // 라인 차트 기본 설정
    fill: dataset.fill ?? false,
    tension: dataset.tension ?? props.tension,
    stepped: props.stepped,
    spanGaps: props.spanGaps,
    showLine: props.showLine,
    pointRadius: dataset.pointRadius ?? 4,
    pointHoverRadius: dataset.pointHoverRadius ?? 6,
    borderWidth: dataset.borderWidth ?? 2,
    // 기본 색상 설정
    backgroundColor: dataset.backgroundColor || 'rgba(59, 130, 246, 0.1)',
    borderColor: dataset.borderColor || '#3b82f6',
    pointBackgroundColor: dataset.pointBackgroundColor || '#3b82f6',
    pointBorderColor: dataset.pointBorderColor || '#ffffff',
    pointBorderWidth: dataset.pointBorderWidth ?? 2,
  }))

  return data
})

// 라인 차트에 특화된 옵션
const mergedOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  interaction: {
    mode: 'index' as const,
    intersect: false,
  },
  scales: {
    x: {
      type: 'linear',
      display: true,
      title: {
        display: false,
      },
      grid: {
        display: true,
        color: 'rgba(0, 0, 0, 0.1)',
      },
    },
    y: {
      type: 'linear',
      display: true,
      title: {
        display: false,
      },
      grid: {
        display: true,
        color: 'rgba(0, 0, 0, 0.1)',
      },
      beginAtZero: true,
    },
  },
  plugins: {
    tooltip: {
      enabled: true,
      mode: 'index' as const,
      intersect: false,
      backgroundColor: 'rgba(0, 0, 0, 0.8)',
      titleColor: '#ffffff',
      bodyColor: '#ffffff',
      borderColor: '#e5e7eb',
      borderWidth: 1,
      cornerRadius: 6,
      displayColors: true,
      callbacks: {
        title: (context: any) => {
          return context[0]?.label || ''
        },
        label: (context: any) => {
          const dataset = context.dataset
          const value = context.parsed.y
          return `${dataset.label}: ${typeof value === 'number' ? value.toLocaleString() : value}`
        },
      },
    },
    legend: {
      display: true,
      position: 'top' as const,
      align: 'start' as const,
      labels: {
        usePointStyle: true,
        padding: 20,
        font: {
          size: 12,
        },
      },
    },
  },
  elements: {
    line: {
      tension: props.tension,
      borderCapStyle: 'round' as const,
      borderJoinStyle: 'round' as const,
    },
    point: {
      radius: 4,
      hoverRadius: 6,
      borderWidth: 2,
      hoverBorderWidth: 3,
    },
  },
  animation: {
    duration: 750,
    easing: 'easeInOutQuart' as const,
  },
  ...props.options,
}))
</script>