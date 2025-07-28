import type { Meta, StoryObj } from '@storybook/vue3'
import DataDashboard from './DataDashboard.vue'
import type { AdvancedTableColumn } from '@/types/ui'

// 샘플 데이터 생성 함수
const generateSalesData = (count: number) => {
  const products = ['노트북', '모니터', '키보드', '마우스', '헤드셋', '웹캠', '스피커', '태블릿']
  const regions = ['서울', '부산', '대구', '인천', '광주', '대전', '울산', '수원']
  const categories = ['전자기기', '컴퓨터', '액세서리', '모바일']

  return Array.from({ length: count }, (_, i) => ({
    id: i + 1,
    product: products[Math.floor(Math.random() * products.length)],
    category: categories[Math.floor(Math.random() * categories.length)],
    region: regions[Math.floor(Math.random() * regions.length)],
    sales: Math.floor(Math.random() * 1000000) + 100000,
    quantity: Math.floor(Math.random() * 100) + 1,
    profit: Math.floor(Math.random() * 500000) + 50000,
    date: new Date(2024, Math.floor(Math.random() * 12), Math.floor(Math.random() * 28) + 1).toISOString().split('T')[0],
    quarter: `Q${Math.floor(Math.random() * 4) + 1}`,
    salesRep: `담당자 ${i % 10 + 1}`,
    rating: Math.floor(Math.random() * 5) + 1,
    isPromoted: Math.random() > 0.7,
  }))
}

const generateEmployeeData = (count: number) => {
  const departments = ['개발팀', '디자인팀', '마케팅팀', '영업팀', '인사팀', '재무팀']
  const positions = ['인턴', '주니어', '시니어', '리드', '매니저', '디렉터']
  const skills = ['JavaScript', 'Python', 'Java', 'React', 'Vue', 'Angular', 'Node.js', 'SQL']

  return Array.from({ length: count }, (_, i) => ({
    id: i + 1,
    name: `직원 ${i + 1}`,
    department: departments[Math.floor(Math.random() * departments.length)],
    position: positions[Math.floor(Math.random() * positions.length)],
    salary: Math.floor(Math.random() * 5000) * 10000 + 3000000,
    experience: Math.floor(Math.random() * 15) + 1,
    age: Math.floor(Math.random() * 30) + 25,
    performance: Math.floor(Math.random() * 100) + 1,
    satisfaction: Math.floor(Math.random() * 10) + 1,
    skills: Math.floor(Math.random() * 3) + 1,
    joinDate: new Date(2020 + Math.floor(Math.random() * 4), Math.floor(Math.random() * 12), Math.floor(Math.random() * 28) + 1).toISOString().split('T')[0],
    isRemote: Math.random() > 0.6,
  }))
}

// 컬럼 정의
const salesColumns: AdvancedTableColumn[] = [
  {
    key: 'id',
    title: 'ID',
    width: 80,
    sortable: true,
    filter: { type: 'number' },
  },
  {
    key: 'product',
    title: '제품명',
    sortable: true,
    filter: { type: 'text', placeholder: '제품명 검색' },
  },
  {
    key: 'category',
    title: '카테고리',
    sortable: true,
    filter: {
      type: 'select',
      options: [
        { label: '전자기기', value: '전자기기' },
        { label: '컴퓨터', value: '컴퓨터' },
        { label: '액세서리', value: '액세서리' },
        { label: '모바일', value: '모바일' },
      ],
    },
  },
  {
    key: 'region',
    title: '지역',
    sortable: true,
    filter: {
      type: 'multiSelect',
      options: [
        { label: '서울', value: '서울' },
        { label: '부산', value: '부산' },
        { label: '대구', value: '대구' },
        { label: '인천', value: '인천' },
        { label: '광주', value: '광주' },
        { label: '대전', value: '대전' },
      ],
    },
  },
  {
    key: 'sales',
    title: '매출액',
    width: 120,
    sortable: true,
    align: 'right',
    filter: { type: 'number' },
    render: (row) => `₩${row.sales.toLocaleString()}`,
  },
  {
    key: 'quantity',
    title: '수량',
    width: 100,
    sortable: true,
    align: 'center',
    filter: { type: 'number' },
  },
  {
    key: 'profit',
    title: '수익',
    width: 120,
    sortable: true,
    align: 'right',
    filter: { type: 'number' },
    render: (row) => `₩${row.profit.toLocaleString()}`,
  },
  {
    key: 'date',
    title: '판매일',
    sortable: true,
    filter: { type: 'date' },
  },
  {
    key: 'quarter',
    title: '분기',
    width: 80,
    sortable: true,
    align: 'center',
    filter: {
      type: 'select',
      options: [
        { label: 'Q1', value: 'Q1' },
        { label: 'Q2', value: 'Q2' },
        { label: 'Q3', value: 'Q3' },
        { label: 'Q4', value: 'Q4' },
      ],
    },
  },
  {
    key: 'rating',
    title: '평점',
    width: 100,
    sortable: true,
    align: 'center',
    render: (row) => '⭐'.repeat(row.rating),
  },
]

const employeeColumns: AdvancedTableColumn[] = [
  {
    key: 'id',
    title: 'ID',
    width: 80,
    sortable: true,
  },
  {
    key: 'name',
    title: '이름',
    sortable: true,
    filter: { type: 'text' },
  },
  {
    key: 'department',
    title: '부서',
    sortable: true,
    filter: {
      type: 'select',
      options: [
        { label: '개발팀', value: '개발팀' },
        { label: '디자인팀', value: '디자인팀' },
        { label: '마케팅팀', value: '마케팅팀' },
        { label: '영업팀', value: '영업팀' },
        { label: '인사팀', value: '인사팀' },
        { label: '재무팀', value: '재무팀' },
      ],
    },
  },
  {
    key: 'position',
    title: '직급',
    sortable: true,
    filter: { type: 'text' },
  },
  {
    key: 'salary',
    title: '연봉',
    width: 120,
    sortable: true,
    align: 'right',
    filter: { type: 'number' },
    render: (row) => `${(row.salary / 10000).toLocaleString()}만원`,
  },
  {
    key: 'experience',
    title: '경력',
    width: 80,
    sortable: true,
    align: 'center',
    filter: { type: 'number' },
    render: (row) => `${row.experience}년`,
  },
  {
    key: 'age',
    title: '나이',
    width: 80,
    sortable: true,
    align: 'center',
    filter: { type: 'number' },
  },
  {
    key: 'performance',
    title: '성과 점수',
    width: 100,
    sortable: true,
    align: 'center',
    filter: { type: 'number' },
    render: (row) => `${row.performance}점`,
  },
  {
    key: 'satisfaction',
    title: '만족도',
    width: 100,
    sortable: true,
    align: 'center',
    render: (row) => '😊'.repeat(Math.ceil(row.satisfaction / 2)),
  },
  {
    key: 'isRemote',
    title: '재택근무',
    width: 100,
    sortable: true,
    align: 'center',
    filter: { type: 'boolean' },
    render: (row) => row.isRemote ? '🏠 재택' : '🏢 출근',
  },
]

// 데이터 생성
const salesData = generateSalesData(200)
const employeeData = generateEmployeeData(150)
const largeSalesData = generateSalesData(5000)

const meta: Meta<typeof DataDashboard> = {
  title: 'UI/Data/DataDashboard',
  component: DataDashboard,
  parameters: {
    layout: 'fullscreen',
    docs: {
      description: {
        component: `
데이터 시각화와 테이블이 통합된 대시보드 컴포넌트입니다.

## 주요 기능
- 📊 다양한 차트 타입 (선형, 막대, 파이, 산점도)
- 🔄 실시간 차트-테이블 연동
- 🎛️ 동적 레이아웃 전환
- 📱 반응형 디자인
- 📈 실시간 통계 정보
- 📥 데이터 내보내기
- 🖥️ 전체화면 모드

## 차트-테이블 연동
- 차트 클릭 시 해당 데이터가 테이블에서 선택됨
- 테이블 필터링 시 차트 데이터 자동 업데이트
- 호버 시 데이터 하이라이트
        `,
      },
    },
  },
  argTypes: {
    defaultLayout: {
      control: { type: 'select' },
      options: ['horizontal', 'vertical', 'sidebar', 'tabs'],
      description: '기본 레이아웃',
    },
    defaultChartType: {
      control: { type: 'select' },
      options: ['line', 'bar', 'pie', 'scatter'],
      description: '기본 차트 타입',
    },
    showChart: {
      control: { type: 'boolean' },
      description: '차트 표시',
    },
    showTable: {
      control: { type: 'boolean' },
      description: '테이블 표시',
    },
    showStats: {
      control: { type: 'boolean' },
      description: '통계 표시',
    },
    loading: {
      control: { type: 'boolean' },
      description: '로딩 상태',
    },
  },
}

export default meta
type Story = StoryObj<typeof DataDashboard>

// 기본 스토리
export const Default: Story = {
  args: {
    title: '매출 대시보드',
    description: '제품별 매출 현황을 확인할 수 있습니다',
    data: salesData,
    columns: salesColumns,
    showChart: true,
    showTable: true,
    defaultLayout: 'horizontal',
    defaultChartType: 'bar',
  },
}

// 직원 데이터 대시보드
export const EmployeeDashboard: Story = {
  args: {
    title: '직원 현황 대시보드',
    description: '직원 정보 및 성과 분석',
    chartTitle: '부서별 직원 분포',
    tableTitle: '직원 목록',
    data: employeeData,
    columns: employeeColumns,
    showChart: true,
    showTable: true,
    showStats: true,
    defaultLayout: 'sidebar',
    defaultChartType: 'pie',
  },
}

// 레이아웃 변형
export const VerticalLayout: Story = {
  args: {
    title: '세로 레이아웃 대시보드',
    data: salesData,
    columns: salesColumns,
    defaultLayout: 'vertical',
    defaultChartType: 'line',
  },
}

export const SidebarLayout: Story = {
  args: {
    title: '사이드바 레이아웃 대시보드',
    data: salesData,
    columns: salesColumns,
    showStats: true,
    defaultLayout: 'sidebar',
    defaultChartType: 'bar',
  },
}

// 차트 타입별 스토리
export const LineChartDashboard: Story = {
  args: {
    title: '시계열 분석 대시보드',
    description: '시간에 따른 매출 추이',
    data: salesData,
    columns: salesColumns,
    defaultChartType: 'line',
  },
}

export const PieChartDashboard: Story = {
  args: {
    title: '카테고리별 분포 대시보드',
    description: '제품 카테고리별 매출 비중',
    data: salesData,
    columns: salesColumns,
    defaultChartType: 'pie',
  },
}

export const ScatterPlotDashboard: Story = {
  args: {
    title: '상관관계 분석 대시보드',
    description: '매출과 수익의 상관관계',
    data: salesData,
    columns: salesColumns,
    defaultChartType: 'scatter',
  },
}

// 차트 전용 모드
export const ChartOnly: Story = {
  args: {
    title: '차트 전용 대시보드',
    data: salesData,
    columns: salesColumns,
    showChart: true,
    showTable: false,
    defaultChartType: 'bar',
  },
}

// 테이블 전용 모드
export const TableOnly: Story = {
  args: {
    title: '테이블 전용 대시보드',
    data: salesData,
    columns: salesColumns,
    showChart: false,
    showTable: true,
  },
}

// 대용량 데이터
export const LargeDataset: Story = {
  args: {
    title: '대용량 데이터 대시보드',
    description: '5,000개 데이터 포인트',
    data: largeSalesData,
    columns: salesColumns,
    defaultChartType: 'line',
  },
  parameters: {
    docs: {
      description: {
        story: '5,000개의 대용량 데이터셋을 처리하는 대시보드입니다. 가상 스크롤링과 성능 최적화가 적용됩니다.',
      },
    },
  },
}

// 로딩 상태
export const LoadingState: Story = {
  args: {
    title: '로딩 중인 대시보드',
    data: salesData,
    columns: salesColumns,
    loading: true,
  },
}

export const ChartLoading: Story = {
  args: {
    title: '차트 로딩 중',
    data: salesData,
    columns: salesColumns,
    chartLoading: true,
  },
}

export const TableLoading: Story = {
  args: {
    title: '테이블 로딩 중',
    data: salesData,
    columns: salesColumns,
    tableLoading: true,
  },
}

// 빈 데이터
export const EmptyData: Story = {
  args: {
    title: '빈 데이터 대시보드',
    description: '데이터가 없을 때의 상태',
    data: [],
    columns: salesColumns,
  },
}

// 실시간 업데이트 시뮬레이션
export const RealTimeSimulation: Story = {
  args: {
    title: '실시간 대시보드',
    description: '데이터가 실시간으로 업데이트됩니다',
    data: salesData.slice(0, 50),
    columns: salesColumns,
    defaultChartType: 'line',
  },
  render: (args) => ({
    components: { DataDashboard },
    setup() {
      const data = ref([...args.data])
      const isUpdating = ref(false)

      const addRandomData = () => {
        if (isUpdating.value) return

        isUpdating.value = true
        const newItem = generateSalesData(1)[0]
        newItem.id = data.value.length + 1
        data.value.push(newItem)

        // 최대 100개 데이터 유지
        if (data.value.length > 100) {
          data.value.shift()
        }

        setTimeout(() => {
          isUpdating.value = false
        }, 500)
      }

      const interval = setInterval(addRandomData, 2000)

      onBeforeUnmount(() => {
        clearInterval(interval)
      })

      return {
        args: { ...args, data },
        isUpdating,
      }
    },
    template: `
      <div>
        <div style="margin-bottom: 1rem; padding: 1rem; background: #f0f9ff; border-radius: 6px; border: 1px solid #0ea5e9;">
          <p style="margin: 0; font-weight: 500;">
            🔄 실시간 업데이트 
            <span v-if="isUpdating" style="color: #0ea5e9;">(업데이트 중...)</span>
          </p>
          <p style="margin: 0.5rem 0 0 0; font-size: 0.875rem; color: #64748b;">
            2초마다 새로운 데이터가 추가됩니다. 차트와 테이블이 자동으로 동기화됩니다.
          </p>
        </div>
        <DataDashboard v-bind="args" />
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: '실시간으로 데이터가 업데이트되는 대시보드입니다. 차트와 테이블이 자동으로 동기화됩니다.',
      },
    },
  },
}

// 차트-테이블 연동 데모
export const InteractiveDemo: Story = {
  args: {
    title: '인터랙티브 대시보드',
    description: '차트를 클릭하거나 테이블 행을 선택해보세요',
    data: salesData.slice(0, 30),
    columns: salesColumns,
    defaultChartType: 'bar',
  },
  render: (args) => ({
    components: { DataDashboard },
    setup() {
      const selectedData = ref([])
      const interactions = ref([])

      const handleSelectionChange = (selected) => {
        selectedData.value = selected
        interactions.value.unshift({
          type: 'selection',
          count: selected.length,
          time: new Date().toLocaleTimeString(),
        })
        if (interactions.value.length > 5) {
          interactions.value = interactions.value.slice(0, 5)
        }
      }

      const handleChartInteraction = (type, data) => {
        interactions.value.unshift({
          type: `chart-${type}`,
          data: data.elements?.length || 0,
          time: new Date().toLocaleTimeString(),
        })
        if (interactions.value.length > 5) {
          interactions.value = interactions.value.slice(0, 5)
        }
      }

      return {
        args,
        selectedData,
        interactions,
        handleSelectionChange,
        handleChartInteraction,
      }
    },
    template: `
      <div>
        <div style="margin-bottom: 1rem; display: flex; gap: 1rem;">
          <div style="flex: 1; padding: 1rem; background: #f8fafc; border-radius: 6px; border: 1px solid #e2e8f0;">
            <h4 style="margin: 0 0 0.5rem 0;">선택된 데이터</h4>
            <p style="margin: 0; font-size: 1.25rem; font-weight: 600; color: #3b82f6;">
              {{ selectedData.length }}개 항목
            </p>
          </div>
          <div style="flex: 2; padding: 1rem; background: #f8fafc; border-radius: 6px; border: 1px solid #e2e8f0;">
            <h4 style="margin: 0 0 0.5rem 0;">최근 상호작용</h4>
            <div v-if="interactions.length === 0" style="color: #64748b; font-size: 0.875rem;">
              차트나 테이블과 상호작용해보세요
            </div>
            <div v-else>
              <div 
                v-for="(interaction, index) in interactions" 
                :key="index"
                style="font-size: 0.875rem; margin-bottom: 0.25rem; color: #475569;"
              >
                {{ interaction.time }} - {{ interaction.type }}: {{ interaction.count || interaction.data }}
              </div>
            </div>
          </div>
        </div>
        <DataDashboard 
          v-bind="args" 
          @selection-change="handleSelectionChange"
          @chart-interaction="handleChartInteraction"
        />
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: '차트와 테이블의 상호작용을 시연하는 대시보드입니다. 클릭하고 선택하여 연동 기능을 확인해보세요.',
      },
    },
  },
}

// 커스텀 테마
export const DarkTheme: Story = {
  args: {
    title: '다크 테마 대시보드',
    data: salesData,
    columns: salesColumns,
    theme: {
      colors: {
        primary: ['#3b82f6', '#ef4444', '#10b981', '#f59e0b', '#8b5cf6', '#06b6d4'],
        secondary: ['#64748b', '#475569', '#334155'],
        accent: ['#f59e0b', '#ef4444'],
        neutral: ['#64748b', '#94a3b8', '#cbd5e1'],
      },
      fonts: {
        family: 'Inter, sans-serif',
        size: 12,
        weight: '400',
      },
      grid: {
        color: '#374151',
        lineWidth: 1,
      },
      tooltip: {
        backgroundColor: '#1f2937',
        titleColor: '#f9fafb',
        bodyColor: '#e5e7eb',
        borderColor: '#374151',
      },
    },
  },
  parameters: {
    backgrounds: {
      default: 'dark',
      values: [
        { name: 'dark', value: '#1f2937' },
      ],
    },
  },
}

// 내보내기 기능 데모
export const ExportDemo: Story = {
  args: {
    title: '데이터 내보내기 대시보드',
    description: '차트와 테이블 데이터를 다양한 형식으로 내보낼 수 있습니다',
    data: salesData.slice(0, 20),
    columns: salesColumns,
  },
  render: (args) => ({
    components: { DataDashboard },
    setup() {
      const exportHistory = ref([])

      const handleDataExport = (data, format) => {
        exportHistory.value.unshift({
          format,
          count: data.length,
          time: new Date().toLocaleTimeString(),
        })
        if (exportHistory.value.length > 3) {
          exportHistory.value = exportHistory.value.slice(0, 3)
        }

        // 실제 내보내기 로직 시뮬레이션
        console.log(`Exporting ${data.length} rows as ${format}`)
      }

      return {
        args,
        exportHistory,
        handleDataExport,
      }
    },
    template: `
      <div>
        <div style="margin-bottom: 1rem; padding: 1rem; background: #ecfdf5; border-radius: 6px; border: 1px solid #10b981;">
          <h4 style="margin: 0 0 0.5rem 0;">내보내기 기록</h4>
          <div v-if="exportHistory.length === 0" style="color: #059669; font-size: 0.875rem;">
            테이블 상단의 📥 버튼을 클릭하여 데이터를 내보내보세요
          </div>
          <div v-else>
            <div 
              v-for="(export_item, index) in exportHistory" 
              :key="index"
              style="font-size: 0.875rem; margin-bottom: 0.25rem; color: #047857;"
            >
              {{ export_item.time }} - {{ export_item.format.toUpperCase() }} ({{ export_item.count }}개 행)
            </div>
          </div>
        </div>
        <DataDashboard 
          v-bind="args" 
          @data-export="handleDataExport"
        />
      </div>
    `,
  }),
}

import { onBeforeUnmount, ref } from 'vue'