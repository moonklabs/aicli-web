<template>
  <BaseChart
    v-bind="$attrs"
    :chart-type="chartType"
    :data="processedData"
    :options="mergedOptions"
    v-on="$listeners"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import BaseChart from './BaseChart.vue'
import type { ChartData, ChartDataset, PieChartProps } from '@/types/ui'

interface Props extends PieChartProps {
  type?: 'pie' | 'doughnut'
}

const props = withDefaults(defineProps<Props>(), {
  type: 'pie',
  cutout: 0,
  circumference: 360,
  rotation: 0,
  animations: () => ({
    animateRotate: true,
    animateScale: true,
  }),
})

// 차트 타입 결정 (파이 또는 도넛)
const chartType = computed(() => {
  return props.type === 'doughnut' ? 'doughnut' : 'pie'
})

// 파이/도넛 차트에 특화된 데이터 처리
const processedData = computed((): ChartData => {
  const data = { ...props.data }

  // 데이터셋에 파이 차트 특화 설정 적용
  data.datasets = data.datasets.map((dataset: ChartDataset) => {
    // 기본 색상 팔레트 (더 다양한 색상)
    const colors = [
      '#3b82f6', '#ef4444', '#10b981', '#f59e0b',
      '#8b5cf6', '#06b6d4', '#ec4899', '#84cc16',
      '#f97316', '#8b5a3c', '#64748b', '#dc2626',
    ]

    // 데이터 포인트 수에 따라 색상 배열 생성
    const dataLength = Array.isArray(dataset.data) ? dataset.data.length : 0
    const backgroundColors = Array.isArray(dataset.backgroundColor)
      ? dataset.backgroundColor
      : Array.from({ length: dataLength }, (_, i) => colors[i % colors.length])

    const borderColors = Array.isArray(dataset.borderColor)
      ? dataset.borderColor
      : Array.from({ length: dataLength }, () => '#ffffff')

    return {
      ...dataset,
      backgroundColor: backgroundColors,
      borderColor: borderColors,
      borderWidth: dataset.borderWidth ?? 2,
      // 호버 효과
      hoverBackgroundColor: dataset.hoverBackgroundColor || backgroundColors.map((color: string) => `${color}dd`),
      hoverBorderColor: dataset.hoverBorderColor || borderColors,
      hoverBorderWidth: dataset.hoverBorderWidth ?? 3,
      // 도넛 차트의 경우 cutout 설정
      cutout: props.type === 'doughnut' ? (props.cutout || '60%') : 0,
    }
  })

  return data
})

// 파이/도넛 차트에 특화된 옵션
const mergedOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  circumference: props.circumference,
  rotation: props.rotation,
  plugins: {
    tooltip: {
      enabled: true,
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
          const value = context.parsed
          const total = dataset.data.reduce((sum: number, val: any) => {
            return sum + (typeof val === 'number' ? val : val.y || 0)
          }, 0)
          const percentage = total > 0 ? ((value / total) * 100).toFixed(1) : '0.0'
          return `${dataset.label || context.label}: ${value.toLocaleString()} (${percentage}%)`
        },
      },
    },
    legend: {
      display: true,
      position: 'right' as const,
      align: 'center' as const,
      labels: {
        usePointStyle: true,
        padding: 20,
        font: {
          size: 12,
        },
        generateLabels: (chart: any) => {
          const data = chart.data
          if (data.labels && data.datasets.length) {
            const dataset = data.datasets[0]
            return data.labels.map((label: string, i: number) => {
              const value = dataset.data[i]
              const total = dataset.data.reduce((sum: number, val: any) => sum + val, 0)
              const percentage = total > 0 ? ((value / total) * 100).toFixed(1) : '0.0'

              return {
                text: `${label} (${percentage}%)`,
                fillStyle: Array.isArray(dataset.backgroundColor)
                  ? dataset.backgroundColor[i]
                  : dataset.backgroundColor,
                strokeStyle: Array.isArray(dataset.borderColor)
                  ? dataset.borderColor[i]
                  : dataset.borderColor,
                lineWidth: dataset.borderWidth,
                pointStyle: 'circle',
                hidden: false,
                index: i,
              }
            })
          }
          return []
        },
      },
    },
  },
  animation: {
    animateRotate: props.animations?.animateRotate ?? true,
    animateScale: props.animations?.animateScale ?? true,
    duration: 750,
    easing: 'easeInOutQuart' as const,
  },
  ...props.options,
}))
</script>