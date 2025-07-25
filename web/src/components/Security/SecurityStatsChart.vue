<template>
  <div class="security-stats-chart">
    <div 
      ref="chartContainer" 
      class="chart-container"
      :style="{ height: chartHeight + 'px' }"
    />
    <div v-if="loading" class="chart-loading">
      <n-spin size="medium" />
    </div>
    <div v-if="error" class="chart-error">
      <n-result
        status="error"
        title="차트 로딩 실패"
        :description="error"
        size="small"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { NSpin, NResult } from 'naive-ui'
import * as echarts from 'echarts/core'
import {
  LineChart,
  BarChart,
  PieChart,
  ScatterChart
} from 'echarts/charts'
import {
  TitleComponent,
  TooltipComponent,
  GridComponent,
  LegendComponent,
  DataZoomComponent
} from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import type { SecurityStats } from '@/types/api'

// ECharts 컴포넌트 등록
echarts.use([
  LineChart,
  BarChart,
  PieChart,
  ScatterChart,
  TitleComponent,
  TooltipComponent,
  GridComponent,
  LegendComponent,
  DataZoomComponent,
  CanvasRenderer
])

interface Props {
  stats: SecurityStats | null
  chartType: 'login-trends' | 'risk-distribution' | 'security-events' | 'device-analysis'
  height?: number
  loading?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  height: 300,
  loading: false
})

// 상태 관리
const chartContainer = ref<HTMLElement>()
const chartInstance = ref<echarts.ECharts>()
const error = ref<string>('')

// 계산된 속성
const chartHeight = computed(() => props.height)

// 메소드
const initChart = async () => {
  if (!chartContainer.value || !props.stats) return

  try {
    error.value = ''
    
    if (chartInstance.value) {
      chartInstance.value.dispose()
    }

    chartInstance.value = echarts.init(chartContainer.value)
    
    const option = getChartOption()
    if (option) {
      chartInstance.value.setOption(option)
    }
    
    // 반응형 처리
    const resizeObserver = new ResizeObserver(() => {
      chartInstance.value?.resize()
    })
    resizeObserver.observe(chartContainer.value)
    
  } catch (err) {
    error.value = '차트 초기화에 실패했습니다'
    console.error('Chart initialization error:', err)
  }
}

const getChartOption = () => {
  if (!props.stats) return null

  switch (props.chartType) {
    case 'login-trends':
      return getLoginTrendsOption()
    case 'risk-distribution':
      return getRiskDistributionOption()
    case 'security-events':
      return getSecurityEventsOption()
    case 'device-analysis':
      return getDeviceAnalysisOption()
    default:
      return null
  }
}

const getLoginTrendsOption = () => {
  const stats = props.stats!
  
  // 가상 데이터 생성 (실제로는 API에서 받아와야 함)
  const dates = []
  const successData = []
  const failureData = []
  
  for (let i = 29; i >= 0; i--) {
    const date = new Date()
    date.setDate(date.getDate() - i)
    dates.push(date.toLocaleDateString('ko-KR', { month: 'short', day: 'numeric' }))
    
    // 가상 데이터
    successData.push(Math.floor(Math.random() * 50) + 10)
    failureData.push(Math.floor(Math.random() * 10))
  }

  return {
    title: {
      text: '로그인 추세 (최근 30일)',
      textStyle: {
        fontSize: 14,
        fontWeight: 'normal'
      }
    },
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'cross'
      }
    },
    legend: {
      data: ['성공', '실패'],
      bottom: 0
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '15%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      data: dates,
      axisLabel: {
        rotate: 45
      }
    },
    yAxis: {
      type: 'value'
    },
    series: [
      {
        name: '성공',
        type: 'line',
        data: successData,
        smooth: true,
        itemStyle: {
          color: '#27ae60'
        },
        areaStyle: {
          opacity: 0.3
        }
      },
      {
        name: '실패',
        type: 'line',
        data: failureData,
        smooth: true,
        itemStyle: {
          color: '#e74c3c'
        },
        areaStyle: {
          opacity: 0.3
        }
      }
    ]
  }
}

const getRiskDistributionOption = () => {
  const stats = props.stats!
  
  return {
    title: {
      text: '위험도 분포',
      textStyle: {
        fontSize: 14,
        fontWeight: 'normal'
      }
    },
    tooltip: {
      trigger: 'item',
      formatter: '{a} <br/>{b}: {c} ({d}%)'
    },
    legend: {
      bottom: 0,
      data: ['안전 (0-39)', '낮음 (40-59)', '보통 (60-79)', '높음 (80-100)']
    },
    series: [
      {
        name: '위험도',
        type: 'pie',
        radius: ['40%', '70%'],
        center: ['50%', '45%'],
        data: [
          { 
            value: stats.totalLogins - stats.suspiciousAttempts, 
            name: '안전 (0-39)',
            itemStyle: { color: '#27ae60' }
          },
          { 
            value: Math.floor(stats.suspiciousAttempts * 0.3), 
            name: '낮음 (40-59)',
            itemStyle: { color: '#3498db' }
          },
          { 
            value: Math.floor(stats.suspiciousAttempts * 0.5), 
            name: '보통 (60-79)',
            itemStyle: { color: '#f39c12' }
          },
          { 
            value: Math.floor(stats.suspiciousAttempts * 0.2), 
            name: '높음 (80-100)',
            itemStyle: { color: '#e74c3c' }
          }
        ],
        emphasis: {
          itemStyle: {
            shadowBlur: 10,
            shadowOffsetX: 0,
            shadowColor: 'rgba(0, 0, 0, 0.5)'
          }
        }
      }
    ]
  }
}

const getSecurityEventsOption = () => {
  const stats = props.stats!
  
  return {
    title: {
      text: '보안 이벤트 통계',
      textStyle: {
        fontSize: 14,
        fontWeight: 'normal'
      }
    },
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'shadow'
      }
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'value'
    },
    yAxis: {
      type: 'category',
      data: ['성공 로그인', '실패 로그인', '의심스러운 시도', '차단된 시도']
    },
    series: [
      {
        name: '이벤트 수',
        type: 'bar',
        data: [
          {
            value: stats.successfulLogins,
            itemStyle: { color: '#27ae60' }
          },
          {
            value: stats.failedLogins,
            itemStyle: { color: '#e74c3c' }
          },
          {
            value: stats.suspiciousAttempts,
            itemStyle: { color: '#f39c12' }
          },
          {
            value: stats.blockedAttempts,
            itemStyle: { color: '#8e44ad' }
          }
        ]
      }
    ]
  }
}

const getDeviceAnalysisOption = () => {
  const stats = props.stats!
  
  // 가상 디바이스 데이터
  const deviceData = [
    { name: 'Desktop', value: Math.floor(stats.uniqueDevices * 0.4) },
    { name: 'Mobile', value: Math.floor(stats.uniqueDevices * 0.45) },
    { name: 'Tablet', value: Math.floor(stats.uniqueDevices * 0.15) }
  ]
  
  return {
    title: {
      text: '디바이스 분석',
      textStyle: {
        fontSize: 14,
        fontWeight: 'normal'
      }
    },
    tooltip: {
      trigger: 'item'
    },
    legend: {
      bottom: 0
    },
    series: [
      {
        name: '디바이스 유형',
        type: 'pie',
        radius: '60%',
        center: ['50%', '45%'],
        data: deviceData,
        emphasis: {
          itemStyle: {
            shadowBlur: 10,
            shadowOffsetX: 0,
            shadowColor: 'rgba(0, 0, 0, 0.5)'
          }
        }
      }
    ]
  }
}

// 감시자
watch(() => props.stats, () => {
  nextTick(() => {
    initChart()
  })
}, { deep: true })

watch(() => props.chartType, () => {
  nextTick(() => {
    initChart()
  })
})

// 생명주기
onMounted(() => {
  nextTick(() => {
    initChart()
  })
})

onUnmounted(() => {
  if (chartInstance.value) {
    chartInstance.value.dispose()
  }
})
</script>

<style scoped>
.security-stats-chart {
  position: relative;
  width: 100%;
}

.chart-container {
  width: 100%;
}

.chart-loading {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  display: flex;
  align-items: center;
  justify-content: center;
}

.chart-error {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.9);
}
</style>