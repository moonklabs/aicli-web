import type { Meta, StoryObj } from '@storybook/vue3'
import LineChart from './LineChart.vue'
import type { ChartData } from '@/types/ui'

// 모킹 데이터 생성
const generateTimeSeriesData = (months = 12) => {
  const labels = []
  const salesData = []
  const revenueData = []
  const userGrowthData = []

  for (let i = 0; i < months; i++) {
    const date = new Date(2024, i, 1)
    labels.push(date.toLocaleDateString('ko-KR', { year: 'numeric', month: 'short' }))

    // 트렌드가 있는 데이터 생성
    const baseSales = 100 + i * 5
    const baseRevenue = 50 + i * 3
    const baseUsers = 200 + i * 15

    salesData.push(baseSales + Math.random() * 20 - 10)
    revenueData.push(baseRevenue + Math.random() * 15 - 7)
    userGrowthData.push(baseUsers + Math.random() * 30 - 15)
  }

  return { labels, salesData, revenueData, userGrowthData }
}

const { labels, salesData, revenueData, userGrowthData } = generateTimeSeriesData()

const basicLineData: ChartData = {
  labels,
  datasets: [
    {
      label: '매출',
      data: salesData,
      backgroundColor: 'rgba(59, 130, 246, 0.1)',
      borderColor: '#3b82f6',
      borderWidth: 2,
      tension: 0.2,
    },
  ],
}

const multiLineData: ChartData = {
  labels,
  datasets: [
    {
      label: '매출',
      data: salesData,
      backgroundColor: 'rgba(59, 130, 246, 0.1)',
      borderColor: '#3b82f6',
      borderWidth: 2,
      tension: 0.2,
    },
    {
      label: '수익',
      data: revenueData,
      backgroundColor: 'rgba(16, 185, 129, 0.1)',
      borderColor: '#10b981',
      borderWidth: 2,
      tension: 0.2,
    },
    {
      label: '사용자 증가',
      data: userGrowthData,
      backgroundColor: 'rgba(245, 158, 11, 0.1)',
      borderColor: '#f59e0b',
      borderWidth: 2,
      tension: 0.2,
    },
  ],
}

const steppedLineData: ChartData = {
  labels: ['1단계', '2단계', '3단계', '4단계', '5단계'],
  datasets: [
    {
      label: '프로세스 진행률',
      data: [0, 25, 50, 75, 100],
      backgroundColor: 'rgba(139, 92, 246, 0.1)',
      borderColor: '#8b5cf6',
      borderWidth: 3,
      stepped: true,
    },
  ],
}

const meta: Meta<typeof LineChart> = {
  title: 'UI/Charts/LineChart',
  component: LineChart,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component: `
라인 차트 컴포넌트입니다. Chart.js를 기반으로 구현되었습니다.

## 주요 기능
- 📈 시계열 데이터 시각화
- 🎨 다양한 스타일 옵션
- 📱 반응형 디자인
- ♿ 접근성 지원
- 🎯 인터랙티브 기능
- 🔄 실시간 데이터 업데이트
        `,
      },
    },
  },
  argTypes: {
    width: {
      control: { type: 'number', min: 200, max: 1200, step: 50 },
      description: '차트 너비',
    },
    height: {
      control: { type: 'number', min: 200, max: 800, step: 50 },
      description: '차트 높이',
    },
    responsive: {
      control: { type: 'boolean' },
      description: '반응형 여부',
    },
    maintainAspectRatio: {
      control: { type: 'boolean' },
      description: '종횡비 유지',
    },
    tension: {
      control: { type: 'range', min: 0, max: 0.5, step: 0.1 },
      description: '라인 곡률',
    },
    stepped: {
      control: { type: 'select' },
      options: [false, true, 'before', 'after', 'middle'],
      description: '단계별 라인',
    },
    spanGaps: {
      control: { type: 'boolean' },
      description: '데이터 간격 연결',
    },
    showLine: {
      control: { type: 'boolean' },
      description: '라인 표시',
    },
    loading: {
      control: { type: 'boolean' },
      description: '로딩 상태',
    },
  },
}

export default meta
type Story = StoryObj<typeof LineChart>

// 기본 스토리
export const Default: Story = {
  args: {
    data: basicLineData,
    width: 600,
    height: 400,
  },
}

// 다중 라인
export const MultiLine: Story = {
  args: {
    data: multiLineData,
    width: 600,
    height: 400,
  },
  parameters: {
    docs: {
      description: {
        story: '여러 데이터셋을 표시하는 다중 라인 차트입니다.',
      },
    },
  },
}

// 곡선 라인
export const CurvedLine: Story = {
  args: {
    data: basicLineData,
    tension: 0.4,
    width: 600,
    height: 400,
  },
  parameters: {
    docs: {
      description: {
        story: '부드러운 곡선으로 표시되는 라인 차트입니다.',
      },
    },
  },
}

// 직선 라인
export const StraightLine: Story = {
  args: {
    data: basicLineData,
    tension: 0,
    width: 600,
    height: 400,
  },
  parameters: {
    docs: {
      description: {
        story: '직선으로 연결되는 라인 차트입니다.',
      },
    },
  },
}

// 단계별 라인
export const SteppedLine: Story = {
  args: {
    data: steppedLineData,
    stepped: true,
    width: 600,
    height: 400,
  },
  parameters: {
    docs: {
      description: {
        story: '단계별로 표시되는 라인 차트입니다. 프로세스 진행률 등을 표시할 때 유용합니다.',
      },
    },
  },
}

// 영역 채우기
export const FilledArea: Story = {
  args: {
    data: {
      ...basicLineData,
      datasets: basicLineData.datasets.map(dataset => ({
        ...dataset,
        fill: true,
        backgroundColor: 'rgba(59, 130, 246, 0.2)',
      })),
    },
    width: 600,
    height: 400,
  },
  parameters: {
    docs: {
      description: {
        story: '라인 아래 영역이 채워진 차트입니다.',
      },
    },
  },
}

// 포인트 없는 라인
export const LineWithoutPoints: Story = {
  args: {
    data: {
      ...basicLineData,
      datasets: basicLineData.datasets.map(dataset => ({
        ...dataset,
        pointRadius: 0,
        pointHoverRadius: 5,
      })),
    },
    width: 600,
    height: 400,
  },
  parameters: {
    docs: {
      description: {
        story: '데이터 포인트가 숨겨진 라인 차트입니다. 호버 시에만 포인트가 표시됩니다.',
      },
    },
  },
}

// 대형 포인트
export const LargePoints: Story = {
  args: {
    data: {
      ...basicLineData,
      datasets: basicLineData.datasets.map(dataset => ({
        ...dataset,
        pointRadius: 8,
        pointHoverRadius: 12,
        pointBackgroundColor: '#3b82f6',
        pointBorderColor: '#ffffff',
        pointBorderWidth: 3,
      })),
    },
    width: 600,
    height: 400,
  },
  parameters: {
    docs: {
      description: {
        story: '큰 데이터 포인트가 표시되는 라인 차트입니다.',
      },
    },
  },
}

// 테마 적용
export const ThemedChart: Story = {
  args: {
    data: multiLineData,
    theme: {
      colors: {
        primary: ['#6366f1', '#8b5cf6', '#ec4899', '#f59e0b'],
        secondary: ['#64748b'],
        accent: ['#06b6d4'],
        neutral: ['#374151'],
      },
      fonts: {
        family: 'Inter, sans-serif',
        size: 12,
        weight: 'normal',
      },
      grid: {
        color: '#e2e8f0',
        lineWidth: 1,
      },
      tooltip: {
        backgroundColor: '#1e293b',
        titleColor: '#f1f5f9',
        bodyColor: '#cbd5e1',
        borderColor: '#475569',
      },
    },
    width: 600,
    height: 400,
  },
  parameters: {
    docs: {
      description: {
        story: '커스텀 테마가 적용된 라인 차트입니다.',
      },
    },
  },
}

// 실시간 업데이트
export const RealTimeUpdate: Story = {
  args: {
    data: basicLineData,
    realTime: {
      enabled: true,
      interval: 2000,
      maxDataPoints: 20,
      animationDuration: 500,
    },
    width: 600,
    height: 400,
  },
  parameters: {
    docs: {
      description: {
        story: '실시간으로 데이터가 업데이트되는 라인 차트입니다. (데모용 - 실제 데이터 소스 필요)',
      },
    },
  },
}

// 줌 및 팬 기능
export const ZoomAndPan: Story = {
  args: {
    data: {
      labels: Array.from({ length: 100 }, (_, i) => `Day ${i + 1}`),
      datasets: [
        {
          label: '데이터 포인트',
          data: Array.from({ length: 100 }, () => Math.random() * 100),
          borderColor: '#3b82f6',
          borderWidth: 1,
          pointRadius: 1,
        },
      ],
    },
    zoom: {
      enabled: true,
      mode: 'x',
      rangeMin: { x: 0 },
      rangeMax: { x: 100 },
    },
    width: 600,
    height: 400,
  },
  parameters: {
    docs: {
      description: {
        story: '줌과 팬 기능이 활성화된 라인 차트입니다. 마우스 휠로 줌, 드래그로 팬이 가능합니다.',
      },
    },
  },
}

// 내보내기 기능
export const WithExport: Story = {
  args: {
    data: multiLineData,
    export: {
      enabled: true,
      formats: ['png', 'jpg', 'svg'],
      quality: 0.9,
      filename: 'line-chart',
    },
    showToolbar: true,
    width: 600,
    height: 400,
  },
  parameters: {
    docs: {
      description: {
        story: '내보내기 기능이 있는 라인 차트입니다. 우상단 툴바에서 다양한 형식으로 내보낼 수 있습니다.',
      },
    },
  },
}

// 접근성 최적화
export const AccessibilityOptimized: Story = {
  args: {
    data: multiLineData,
    accessibility: {
      enabled: true,
      description: '2024년 월별 비즈니스 지표 추이',
      summary: '매출, 수익, 사용자 증가율이 모두 꾸준한 상승 추세를 보이고 있습니다. 매출은 1월 100에서 12월 155로 55% 증가했고, 수익은 50에서 83으로 66% 증가했습니다.',
    },
    showCustomLegend: true,
    width: 600,
    height: 400,
  },
  parameters: {
    docs: {
      description: {
        story: '접근성이 최적화된 라인 차트입니다. 스크린 리더를 위한 설명과 키보드 네비게이션을 지원합니다.',
      },
    },
  },
}

// 로딩 상태
export const Loading: Story = {
  args: {
    data: basicLineData,
    loading: true,
    width: 600,
    height: 400,
  },
  parameters: {
    docs: {
      description: {
        story: '로딩 상태를 표시하는 라인 차트입니다.',
      },
    },
  },
}

// 에러 상태
export const Error: Story = {
  args: {
    data: basicLineData,
    error: new Error('차트 데이터를 불러올 수 없습니다'),
    width: 600,
    height: 400,
  },
  parameters: {
    docs: {
      description: {
        story: '에러 상태를 표시하는 라인 차트입니다.',
      },
    },
  },
}

// 빈 데이터
export const EmptyData: Story = {
  args: {
    data: {
      labels: [],
      datasets: [],
    },
    width: 600,
    height: 400,
    fallbackContent: '표시할 데이터가 없습니다',
  },
  parameters: {
    docs: {
      description: {
        story: '데이터가 없을 때의 상태를 표시하는 라인 차트입니다.',
      },
    },
  },
}

// 모바일 최적화
export const MobileOptimized: Story = {
  args: {
    data: basicLineData,
    width: 350,
    height: 250,
    responsive: true,
    options: {
      plugins: {
        legend: {
          position: 'bottom' as const,
        },
      },
      scales: {
        x: {
          ticks: {
            maxTicksLimit: 6,
          },
        },
      },
    },
  },
  parameters: {
    viewport: {
      defaultViewport: 'mobile1',
    },
    docs: {
      description: {
        story: '모바일 디바이스에 최적화된 라인 차트입니다. 범례가 하단에 위치하고 틱 개수가 제한됩니다.',
      },
    },
  },
}

// 애니메이션 효과
export const AnimatedChart: Story = {
  args: {
    data: multiLineData,
    options: {
      animation: {
        duration: 2000,
        easing: 'easeInOutQuart' as const,
        delay: (context: any) => context.dataIndex * 100,
      },
    },
    width: 600,
    height: 400,
  },
  parameters: {
    docs: {
      description: {
        story: '애니메이션 효과가 적용된 라인 차트입니다. 데이터 포인트가 순차적으로 나타납니다.',
      },
    },
  },
}