<template>
  <BaseChart
    v-bind="$attrs"
    chart-type="bar"
    :data="processedData"
    :options="mergedOptions"
    v-on="$listeners"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import BaseChart from './BaseChart.vue'
import type { BarChartProps, ChartData, ChartDataset } from '@/types/ui'

interface Props extends BarChartProps {
  // BarChart 특화 속성들
}

const props = withDefaults(defineProps<Props>(), {
  indexAxis: 'x',
  skipNull: false,
  grouped: true,
  barPercentage: 0.9,
  categoryPercentage: 0.8,
})

// 바 차트에 특화된 데이터 처리
const processedData = computed((): ChartData => {
  const data = { ...props.data }

  // 데이터셋에 바 차트 특화 설정 적용
  data.datasets = data.datasets.map((dataset: ChartDataset, index: number) => {
    // 기본 색상 팔레트
    const colors = [
      '#3b82f6', '#ef4444', '#10b981', '#f59e0b',
      '#8b5cf6', '#06b6d4', '#ec4899', '#84cc16',
    ]

    const baseColor = colors[index % colors.length]

    return {
      ...dataset,
      // 바 차트 기본 설정
      backgroundColor: dataset.backgroundColor || baseColor,
      borderColor: dataset.borderColor || baseColor,
      borderWidth: dataset.borderWidth ?? 1,
      borderRadius: dataset.borderRadius ?? 4,
      borderSkipped: dataset.borderSkipped ?? false,
      barPercentage: props.barPercentage,
      categoryPercentage: props.categoryPercentage,
      // 호버 효과
      hoverBackgroundColor: dataset.hoverBackgroundColor || `${baseColor}dd`,
      hoverBorderColor: dataset.hoverBorderColor || baseColor,
      hoverBorderWidth: dataset.hoverBorderWidth ?? 2,
    }
  })

  return data
})

// 바 차트에 특화된 옵션
const mergedOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  indexAxis: props.indexAxis,
  interaction: {
    mode: 'index' as const,
    intersect: false,
  },
  scales: {
    x: {
      type: props.indexAxis === 'y' ? 'linear' : 'category',
      display: true,
      title: {
        display: false,
      },
      grid: {
        display: props.indexAxis === 'y',
        color: 'rgba(0, 0, 0, 0.1)',
      },
      beginAtZero: props.indexAxis === 'y',
      stacked: !props.grouped,
    },
    y: {
      type: props.indexAxis === 'x' ? 'linear' : 'category',
      display: true,
      title: {
        display: false,
      },
      grid: {
        display: props.indexAxis === 'x',
        color: 'rgba(0, 0, 0, 0.1)',
      },
      beginAtZero: props.indexAxis === 'x',
      stacked: !props.grouped,
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
          const value = context.parsed[props.indexAxis === 'x' ? 'y' : 'x']
          return `${dataset.label}: ${typeof value === 'number' ? value.toLocaleString() : value}`
        },
      },
    },
    legend: {
      display: true,
      position: 'top' as const,
      align: 'start' as const,
      labels: {
        usePointStyle: false,
        padding: 20,
        font: {
          size: 12,
        },
      },
    },
  },
  animation: {
    duration: 750,
    easing: 'easeInOutQuart' as const,
  },
  ...props.options,
}))
</script>