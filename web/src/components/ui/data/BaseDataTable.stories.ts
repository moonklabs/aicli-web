import type { Meta, StoryObj } from '@storybook/vue3'
import BaseDataTable from './BaseDataTable.vue'
import type { AdvancedTableColumn } from '@/types/ui'

// 모킹 데이터 생성
const generateMockData = (count: number) => {
  const departments = ['개발', '디자인', '마케팅', '세일즈', '인사', '재무']
  const statuses = ['활성', '비활성', '대기', '완료']

  return Array.from({ length: count }, (_, i) => ({
    id: i + 1,
    name: `사용자 ${i + 1}`,
    email: `user${i + 1}@example.com`,
    age: 20 + Math.floor(Math.random() * 40),
    department: departments[Math.floor(Math.random() * departments.length)],
    salary: 30000000 + Math.floor(Math.random() * 50000000),
    status: statuses[Math.floor(Math.random() * statuses.length)],
    joinDate: new Date(2020 + Math.floor(Math.random() * 4), Math.floor(Math.random() * 12), Math.floor(Math.random() * 28) + 1).toISOString().split('T')[0],
    isActive: Math.random() > 0.3,
    score: Math.floor(Math.random() * 100),
  }))
}

const basicColumns: AdvancedTableColumn[] = [
  {
    key: 'id',
    title: 'ID',
    width: 80,
    sortable: true,
    filterable: true,
    filter: { type: 'number', placeholder: 'ID 검색' },
  },
  {
    key: 'name',
    title: '이름',
    sortable: true,
    filterable: true,
    filter: { type: 'text', placeholder: '이름 검색' },
  },
  {
    key: 'email',
    title: '이메일',
    sortable: true,
    filterable: true,
    ellipsis: true,
    filter: { type: 'text', placeholder: '이메일 검색' },
  },
  {
    key: 'age',
    title: '나이',
    width: 100,
    sortable: true,
    filterable: true,
    align: 'center',
    filter: { type: 'number', placeholder: '나이 검색' },
  },
  {
    key: 'department',
    title: '부서',
    sortable: true,
    filterable: true,
    filter: {
      type: 'select',
      options: [
        { label: '개발', value: '개발' },
        { label: '디자인', value: '디자인' },
        { label: '마케팅', value: '마케팅' },
        { label: '세일즈', value: '세일즈' },
        { label: '인사', value: '인사' },
        { label: '재무', value: '재무' },
      ],
    },
  },
  {
    key: 'salary',
    title: '연봉',
    width: 120,
    sortable: true,
    filterable: true,
    align: 'right',
    render: (row) => `${(row.salary / 10000).toLocaleString()}만원`,
  },
  {
    key: 'joinDate',
    title: '입사일',
    sortable: true,
    filterable: true,
    filter: { type: 'date' },
  },
  {
    key: 'isActive',
    title: '상태',
    width: 100,
    sortable: true,
    align: 'center',
    render: (row) => row.isActive ? '✅ 활성' : '❌ 비활성',
  },
  {
    key: 'score',
    title: '점수',
    width: 100,
    sortable: true,
    filterable: true,
    align: 'center',
    render: (row) => `${row.score}점`,
  },
]

const smallData = generateMockData(10)
const mediumData = generateMockData(100)
const largeData = generateMockData(10000)

const meta: Meta<typeof BaseDataTable> = {
  title: 'UI/Data/BaseDataTable',
  component: BaseDataTable,
  parameters: {
    layout: 'fullscreen',
    docs: {
      description: {
        component: `
고성능 데이터 테이블 컴포넌트입니다.

## 주요 기능
- ✨ 가상 스크롤링으로 대용량 데이터 처리
- 🔍 실시간 검색 및 필터링
- 📊 다중 컬럼 정렬
- ✅ 행 선택 (단일/다중)
- 📱 모바일 최적화
- ♿ 접근성 지원 (WCAG 2.1 AA)
- 🎨 커스터마이징 가능한 셀 렌더링
        `,
      },
    },
  },
  argTypes: {
    size: {
      control: { type: 'select' },
      options: ['small', 'medium', 'large'],
      description: '테이블 크기',
    },
    striped: {
      control: { type: 'boolean' },
      description: '줄무늬 스타일',
    },
    bordered: {
      control: { type: 'boolean' },
      description: '테두리 표시',
    },
    loading: {
      control: { type: 'boolean' },
      description: '로딩 상태',
    },
    pagination: {
      control: { type: 'boolean' },
      description: '페이지네이션 사용',
    },
    globalSearch: {
      control: { type: 'boolean' },
      description: '전역 검색 사용',
    },
    showFilters: {
      control: { type: 'boolean' },
      description: '컬럼 필터 표시',
    },
  },
}

export default meta
type Story = StoryObj<typeof BaseDataTable>

// 기본 스토리
export const Default: Story = {
  args: {
    data: smallData,
    columns: basicColumns,
    rowKey: 'id',
  },
}

// 크기 변형
export const Small: Story = {
  args: {
    data: smallData,
    columns: basicColumns,
    size: 'small',
    rowKey: 'id',
  },
}

export const Large: Story = {
  args: {
    data: smallData,
    columns: basicColumns,
    size: 'large',
    rowKey: 'id',
  },
}

// 스타일 변형
export const Striped: Story = {
  args: {
    data: smallData,
    columns: basicColumns,
    striped: true,
    rowKey: 'id',
  },
}

export const Bordered: Story = {
  args: {
    data: smallData,
    columns: basicColumns,
    bordered: true,
    rowKey: 'id',
  },
}

// 기능별 스토리
export const WithGlobalSearch: Story = {
  args: {
    data: mediumData,
    columns: basicColumns,
    globalSearch: true,
    globalSearchPlaceholder: '이름, 이메일, 부서로 검색...',
    rowKey: 'id',
  },
  parameters: {
    docs: {
      description: {
        story: '전역 검색 기능이 활성화된 테이블입니다. 모든 컬럼을 대상으로 검색할 수 있습니다.',
      },
    },
  },
}

export const WithFilters: Story = {
  args: {
    data: mediumData,
    columns: basicColumns,
    showFilters: true,
    rowKey: 'id',
  },
  parameters: {
    docs: {
      description: {
        story: '컬럼별 필터링 기능이 활성화된 테이블입니다. 각 컬럼 하단에서 개별적으로 필터를 적용할 수 있습니다.',
      },
    },
  },
}

export const WithSelection: Story = {
  args: {
    data: smallData,
    columns: basicColumns,
    selection: {
      type: 'checkbox',
      selectedKeys: [],
    },
    rowKey: 'id',
  },
  parameters: {
    docs: {
      description: {
        story: '행 선택 기능이 있는 테이블입니다. 체크박스를 통해 다중 선택이 가능합니다.',
      },
    },
  },
}

export const WithRadioSelection: Story = {
  args: {
    data: smallData,
    columns: basicColumns,
    selection: {
      type: 'radio',
      selectedKeys: [],
    },
    rowKey: 'id',
  },
  parameters: {
    docs: {
      description: {
        story: '라디오 버튼을 통한 단일 선택이 가능한 테이블입니다.',
      },
    },
  },
}

export const WithPagination: Story = {
  args: {
    data: mediumData,
    columns: basicColumns,
    pagination: true,
    pageSize: 20,
    rowKey: 'id',
  },
  parameters: {
    docs: {
      description: {
        story: '페이지네이션이 적용된 테이블입니다. 대량의 데이터를 페이지 단위로 나누어 표시합니다.',
      },
    },
  },
}

export const VirtualScrolling: Story = {
  args: {
    data: largeData,
    columns: basicColumns,
    virtualScroll: {
      enabled: true,
      itemHeight: 40,
      overscan: 5,
    },
    rowKey: 'id',
  },
  parameters: {
    docs: {
      description: {
        story: '가상 스크롤링이 적용된 테이블입니다. 10,000개의 행을 부드럽게 스크롤할 수 있습니다.',
      },
    },
  },
}

export const Loading: Story = {
  args: {
    data: smallData,
    columns: basicColumns,
    loading: true,
    rowKey: 'id',
  },
  parameters: {
    docs: {
      description: {
        story: '로딩 상태를 표시하는 테이블입니다.',
      },
    },
  },
}

export const Empty: Story = {
  args: {
    data: [],
    columns: basicColumns,
    rowKey: 'id',
  },
  parameters: {
    docs: {
      description: {
        story: '데이터가 없을 때의 빈 상태를 표시하는 테이블입니다.',
      },
    },
  },
}

export const CustomEmpty: Story = {
  args: {
    data: [],
    columns: basicColumns,
    rowKey: 'id',
  },
  render: (args) => ({
    components: { BaseDataTable },
    setup() {
      return { args }
    },
    template: `
      <BaseDataTable v-bind="args">
        <template #empty>
          <div style="padding: 2rem; text-align: center;">
            <div style="font-size: 3rem; margin-bottom: 1rem;">📊</div>
            <h3>데이터가 없습니다</h3>
            <p style="color: #6b7280;">새로운 데이터를 추가해보세요.</p>
            <button style="margin-top: 1rem; padding: 0.5rem 1rem; background: #3b82f6; color: white; border: none; border-radius: 4px; cursor: pointer;">
              데이터 추가하기
            </button>
          </div>
        </template>
      </BaseDataTable>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: '커스텀 빈 상태 슬롯을 사용한 테이블입니다.',
      },
    },
  },
}

export const AllFeatures: Story = {
  args: {
    data: mediumData,
    columns: basicColumns,
    globalSearch: true,
    showFilters: true,
    pagination: true,
    pageSize: 15,
    striped: true,
    selection: {
      type: 'checkbox',
      selectedKeys: [],
    },
    columnSettings: true,
    stickyHeader: true,
    rowKey: 'id',
  },
  parameters: {
    docs: {
      description: {
        story: '모든 기능이 활성화된 완전한 데이터 테이블입니다.',
      },
    },
  },
}

// 성능 테스트 스토리
export const PerformanceTest: Story = {
  args: {
    data: largeData,
    columns: basicColumns,
    virtualScroll: {
      enabled: true,
      itemHeight: 40,
      overscan: 5,
    },
    globalSearch: true,
    showFilters: true,
    performance: {
      debounceMs: 300,
      throttleMs: 100,
      lazyLoading: true,
    },
    rowKey: 'id',
  },
  parameters: {
    docs: {
      description: {
        story: '10,000개 행의 대용량 데이터로 성능을 테스트하는 테이블입니다.',
      },
    },
  },
}

// 접근성 테스트 스토리
export const AccessibilityTest: Story = {
  args: {
    data: smallData,
    columns: basicColumns,
    ariaLabel: '직원 정보 테이블',
    ariaDescribedby: 'table-description',
    rowKey: 'id',
  },
  render: (args) => ({
    components: { BaseDataTable },
    setup() {
      return { args }
    },
    template: `
      <div>
        <div id="table-description" style="margin-bottom: 1rem; padding: 1rem; background: #f3f4f6; border-radius: 6px;">
          <p><strong>키보드 네비게이션:</strong></p>
          <ul style="margin: 0.5rem 0; padding-left: 1.5rem;">
            <li>Tab/Shift+Tab: 포커스 이동</li>
            <li>화살표 키: 셀 간 이동</li>
            <li>Enter/Space: 정렬 또는 선택</li>
            <li>Home/End: 행의 시작/끝으로 이동</li>
            <li>Ctrl+Home/End: 테이블 시작/끝으로 이동</li>
          </ul>
        </div>
        <BaseDataTable v-bind="args" />
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: '접근성이 최적화된 테이블입니다. 키보드 네비게이션과 스크린 리더를 지원합니다.',
      },
    },
  },
}

// 모바일 최적화 스토리
export const MobileOptimized: Story = {
  args: {
    data: smallData,
    columns: basicColumns.slice(0, 4), // 모바일에서는 컬럼 수 제한
    responsive: {
      enabled: true,
      breakpoints: {
        sm: 640,
        md: 768,
        lg: 1024,
      },
      hideColumns: {
        sm: ['email', 'department'],
        md: ['department'],
      },
    },
    rowKey: 'id',
  },
  parameters: {
    viewport: {
      defaultViewport: 'mobile1',
    },
    docs: {
      description: {
        story: '모바일 디바이스에 최적화된 테이블입니다. 화면 크기에 따라 컬럼이 자동으로 숨겨집니다.',
      },
    },
  },
}

// 커스텀 렌더링 스토리
export const CustomRendering: Story = {
  args: {
    data: smallData,
    columns: [
      { key: 'id', title: 'ID', width: 80 },
      {
        key: 'name',
        title: '이름',
        render: (row) => `👤 ${row.name}`,
      },
      {
        key: 'age',
        title: '나이',
        render: (row) => {
          const ageGroup = row.age < 30 ? '🟢' : row.age < 40 ? '🟡' : '🔴'
          return `${ageGroup} ${row.age}세`
        },
      },
      {
        key: 'salary',
        title: '연봉',
        render: (row) => {
          const level = row.salary > 60000000 ? '💎' : row.salary > 40000000 ? '⭐' : '📍'
          return `${level} ${(row.salary / 10000).toLocaleString()}만원`
        },
      },
      {
        key: 'isActive',
        title: '상태',
        render: (row) => row.isActive
          ? '<span style="color: green; font-weight: bold;">🟢 활성</span>'
          : '<span style="color: red; font-weight: bold;">🔴 비활성</span>',
      },
    ],
    rowKey: 'id',
  },
  parameters: {
    docs: {
      description: {
        story: '커스텀 셀 렌더링 함수를 사용한 테이블입니다. 각 셀에 아이콘과 스타일을 적용할 수 있습니다.',
      },
    },
  },
}