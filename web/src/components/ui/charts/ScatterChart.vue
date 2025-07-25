<template>
  <BaseChart
    v-bind="$attrs"
    chart-type="scatter"
    :data="processedData"
    :options="mergedOptions"
    v-on="$listeners"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import BaseChart from './BaseChart.vue'
import type { ChartData, ChartDataset, ScatterChartProps } from '@/types/ui'

interface Props extends ScatterChartProps {
  // ScatterChart 특화 속성들
}

const props = withDefaults(defineProps<Props>(), {
  showLine: false,
  pointRadius: 5,
  pointHoverRadius: 8,
})

// 스캐터 차트에 특화된 데이터 처리
const processedData = computed((): ChartData => {
  const data = { ...props.data }

  // 데이터셋에 스캐터 차트 특화 설정 적용
  data.datasets = data.datasets.map((dataset: ChartDataset, index: number) => {
    // 기본 색상 팔레트
    const colors = [
      '#3b82f6', '#ef4444', '#10b981', '#f59e0b',
      '#8b5cf6', '#06b6d4', '#ec4899', '#84cc16',
    ]

    const baseColor = colors[index % colors.length]

    return {
      ...dataset,
      // 스캐터 차트 기본 설정
      showLine: dataset.showLine ?? props.showLine,
      pointRadius: dataset.pointRadius ?? props.pointRadius,
      pointHoverRadius: dataset.pointHoverRadius ?? props.pointHoverRadius,
      backgroundColor: dataset.backgroundColor || `${baseColor}80`, // 투명도 추가
      borderColor: dataset.borderColor || baseColor,
      borderWidth: dataset.borderWidth ?? 2,
      pointBackgroundColor: dataset.pointBackgroundColor || baseColor,
      pointBorderColor: dataset.pointBorderColor || '#ffffff',
      pointBorderWidth: dataset.pointBorderWidth ?? 2,
      // 호버 효과
      pointHoverBackgroundColor: dataset.pointHoverBackgroundColor || baseColor,
      pointHoverBorderColor: dataset.pointHoverBorderColor || '#ffffff',
      pointHoverBorderWidth: dataset.pointHoverBorderWidth ?? 3,
      // 라인 설정 (showLine이 true일 때)
      tension: dataset.tension ?? 0,
      fill: false,
    }
  })

  return data
})

// 스캐터 차트에 특화된 옵션
const mergedOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  interaction: {
    mode: 'point' as const,
    intersect: false,
  },
  scales: {
    x: {
      type: 'linear',
      position: 'bottom',
      display: true,
      title: {
        display: true,
        text: 'X 축',
        font: {
          size: 14,
          weight: 'bold',
        },
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
        display: true,
        text: 'Y 축',
        font: {
          size: 14,
          weight: 'bold',
        },
      },
      grid: {
        display: true,
        color: 'rgba(0, 0, 0, 0.1)',
      },
    },
  },
  plugins: {
    tooltip: {
      enabled: true,
      mode: 'point' as const,
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
          const dataset = context[0]?.dataset
          return dataset?.label || '데이터 포인트'
        },
        label: (context: any) => {
          const point = context.parsed
          return `X: ${typeof point.x === 'number' ? point.x.toLocaleString() : point.x}, Y: ${typeof point.y === 'number' ? point.y.toLocaleString() : point.y}`
        },
        afterLabel: (context: any) => {
          // 추가 데이터가 있는 경우 표시
          const dataPoint = context.dataset.data[context.dataIndex]
          if (dataPoint && typeof dataPoint === 'object' && 'label' in dataPoint) {
            return `라벨: ${dataPoint.label}`
          }
          return ''
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
    point: {
      radius: props.pointRadius,
      hoverRadius: props.pointHoverRadius,
      borderWidth: 2,
      hoverBorderWidth: 3,
    },
    line: {
      tension: 0,
      borderWidth: 2,
      fill: false,
    },
  },
  animation: {
    duration: 750,
    easing: 'easeInOutQuart' as const,
  },
  ...props.options,
}))
</script>