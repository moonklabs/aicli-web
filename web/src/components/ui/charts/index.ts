// 차트 컴포넌트 Lazy Loading으로 내보내기
import { defineAsyncComponent } from 'vue'

export const BaseChart = defineAsyncComponent(() => import('./BaseChart.vue'))
export const LineChart = defineAsyncComponent(() => import('./LineChart.vue'))
export const BarChart = defineAsyncComponent(() => import('./BarChart.vue'))
export const PieChart = defineAsyncComponent(() => import('./PieChart.vue'))
export const ScatterChart = defineAsyncComponent(() => import('./ScatterChart.vue'))

// 타입 내보내기
export type {
  AdvancedChartProps,
  BaseChartProps,
  LineChartProps,
  BarChartProps,
  PieChartProps,
  ScatterChartProps,
  ChartData,
  ChartDataset,
  ChartDataPoint,
  ChartTheme,
  RealTimeChartConfig,
  ChartExportConfig,
  ChartZoomConfig,
  ChartTableIntegration,
} from '@/types/ui'