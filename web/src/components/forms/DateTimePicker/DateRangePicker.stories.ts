import type { Meta, StoryObj } from '@storybook/vue3'
// action 함수를 간단한 console.log로 대체
const action = (name: string) => (...args: any[]) => {
  console.log(`[${name}]`, ...args)
}
import { ref } from 'vue'
import DateRangePicker from './DateRangePicker.vue'

interface DateRange {
  start: Date | null
  end: Date | null
}

const meta: Meta<typeof DateRangePicker> = {
  title: 'Forms/DateTimePicker/DateRangePicker',
  component: DateRangePicker,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component: `
### DateRangePicker 컴포넌트

날짜 범위를 선택할 수 있는 고급 컴포넌트입니다.

#### 주요 기능
- 듀얼 캘린더로 시작/종료 날짜 선택
- 미리 정의된 기간 프리셋
- 날짜 범위 하이라이팅
- 최소/최대 날짜 제한
- 커스터마이징 가능한 날짜 형식
- 다국어 지원
- 키보드 네비게이션
- 접근성 표준 준수 (ARIA)
- 반응형 디자인
        `,
      },
    },
  },
  argTypes: {
    modelValue: {
      control: { type: 'object' },
      description: '선택된 날짜 범위 {start: Date, end: Date}',
    },
    min: {
      control: { type: 'date' },
      description: '선택 가능한 최소 날짜',
    },
    max: {
      control: { type: 'date' },
      description: '선택 가능한 최대 날짜',
    },
    disabled: {
      control: { type: 'boolean' },
      description: '비활성화 여부',
    },
    clearable: {
      control: { type: 'boolean' },
      description: '지우기 버튼 표시 여부',
    },
    placeholder: {
      control: { type: 'text' },
      description: '플레이스홀더 텍스트',
    },
    format: {
      control: { type: 'text' },
      description: '날짜 표시 형식',
    },
    locale: {
      control: { type: 'select' },
      options: ['ko', 'en', 'ja', 'zh'],
      description: '언어 설정',
    },
    firstDayOfWeek: {
      control: { type: 'select' },
      options: [0, 1, 2, 3, 4, 5, 6],
      description: '주의 시작 요일',
    },
    showPresets: {
      control: { type: 'boolean' },
      description: '프리셋 표시 여부',
    },
    startPlaceholder: {
      control: { type: 'text' },
      description: '시작 날짜 플레이스홀더',
    },
    endPlaceholder: {
      control: { type: 'text' },
      description: '종료 날짜 플레이스홀더',
    },
  },
  args: {
    clearable: true,
    placeholder: '날짜 범위를 선택하세요',
    format: 'YYYY-MM-DD',
    locale: 'ko',
    firstDayOfWeek: 0,
    showPresets: true,
    startPlaceholder: '시작 날짜',
    endPlaceholder: '종료 날짜',
  },
}

export default meta
type Story = StoryObj<typeof meta>

// 기본 스토리
export const Default: Story = {
  args: {
    modelValue: null,
  },
  render: (args) => ({
    components: { DateRangePicker },
    setup() {
      const dateRange = ref<DateRange | null>(args.modelValue)

      return {
        args,
        dateRange,
        onUpdate: (value: DateRange | null) => {
          dateRange.value = value
          action('update:modelValue')(value)
        },
        onChange: action('change'),
        onClear: action('clear'),
      }
    },
    template: `
      <div style="width: 400px;">
        <DateRangePicker
          v-bind="args"
          v-model="dateRange"
          @update:modelValue="onUpdate"
          @change="onChange"
          @clear="onClear"
        />
        <div style="margin-top: 16px; font-size: 14px;">
          <div v-if="dateRange">
            시작: {{ dateRange.start?.toLocaleDateString('ko-KR') || '미선택' }}<br>
            종료: {{ dateRange.end?.toLocaleDateString('ko-KR') || '미선택' }}
          </div>
          <div v-else>날짜 범위가 선택되지 않았습니다.</div>
        </div>
      </div>
    `,
  }),
}

// 초기값이 있는 경우
export const WithInitialValue: Story = {
  args: {
    modelValue: {
      start: new Date(new Date().setDate(new Date().getDate() - 7)),
      end: new Date(),
    },
  },
  render: (args) => ({
    components: { DateRangePicker },
    setup() {
      const dateRange = ref<DateRange | null>(args.modelValue)

      return {
        args,
        dateRange,
        onUpdate: (value: DateRange | null) => {
          dateRange.value = value
          action('update:modelValue')(value)
        },
      }
    },
    template: `
      <div style="width: 400px;">
        <DateRangePicker
          v-bind="args"
          v-model="dateRange"
          @update:modelValue="onUpdate"
        />
        <div style="margin-top: 16px; font-size: 14px;">
          최근 7일 기간이 선택되어 있습니다.
        </div>
      </div>
    `,
  }),
}

// 커스텀 프리셋
export const WithCustomPresets: Story = {
  args: {
    modelValue: null,
    showPresets: true,
    presets: [
      {
        label: '오늘',
        value: () => ({
          start: new Date(),
          end: new Date(),
        }),
      },
      {
        label: '어제',
        value: () => {
          const yesterday = new Date()
          yesterday.setDate(yesterday.getDate() - 1)
          return {
            start: yesterday,
            end: yesterday,
          }
        },
      },
      {
        label: '최근 7일',
        value: () => ({
          start: new Date(new Date().setDate(new Date().getDate() - 6)),
          end: new Date(),
        }),
      },
      {
        label: '최근 30일',
        value: () => ({
          start: new Date(new Date().setDate(new Date().getDate() - 29)),
          end: new Date(),
        }),
      },
      {
        label: '이번 달',
        value: () => {
          const now = new Date()
          return {
            start: new Date(now.getFullYear(), now.getMonth(), 1),
            end: new Date(now.getFullYear(), now.getMonth() + 1, 0),
          }
        },
      },
      {
        label: '지난 달',
        value: () => {
          const now = new Date()
          return {
            start: new Date(now.getFullYear(), now.getMonth() - 1, 1),
            end: new Date(now.getFullYear(), now.getMonth(), 0),
          }
        },
      },
    ],
  },
  render: (args) => ({
    components: { DateRangePicker },
    setup() {
      const dateRange = ref<DateRange | null>(args.modelValue)

      return {
        args,
        dateRange,
        onUpdate: (value: DateRange | null) => {
          dateRange.value = value
        },
      }
    },
    template: `
      <div style="width: 400px;">
        <DateRangePicker
          v-bind="args"
          v-model="dateRange"
          @update:modelValue="onUpdate"
        />
      </div>
    `,
  }),
}

// 날짜 범위 제한
export const WithDateRestrictions: Story = {
  args: {
    modelValue: null,
    min: new Date(new Date().setMonth(new Date().getMonth() - 3)), // 3개월 전
    max: new Date(new Date().setMonth(new Date().getMonth() + 3)), // 3개월 후
  },
  render: (args) => ({
    components: { DateRangePicker },
    setup() {
      const dateRange = ref<DateRange | null>(args.modelValue)

      return {
        args,
        dateRange,
        onUpdate: (value: DateRange | null) => {
          dateRange.value = value
        },
      }
    },
    template: `
      <div style="width: 400px;">
        <DateRangePicker
          v-bind="args"
          v-model="dateRange"
          @update:modelValue="onUpdate"
        />
        <div style="margin-top: 16px; font-size: 14px; color: #666;">
          최근 3개월부터 향후 3개월까지만 선택 가능합니다.
        </div>
      </div>
    `,
  }),
}

// 과거 날짜만 선택 가능
export const PastDatesOnly: Story = {
  args: {
    modelValue: null,
    max: new Date(),
  },
  render: (args) => ({
    components: { DateRangePicker },
    setup() {
      const dateRange = ref<DateRange | null>(args.modelValue)

      return {
        args,
        dateRange,
        onUpdate: (value: DateRange | null) => {
          dateRange.value = value
        },
      }
    },
    template: `
      <div style="width: 400px;">
        <DateRangePicker
          v-bind="args"
          v-model="dateRange"
          @update:modelValue="onUpdate"
        />
        <div style="margin-top: 16px; font-size: 14px; color: #666;">
          과거 날짜만 선택 가능합니다 (데이터 분석용).
        </div>
      </div>
    `,
  }),
}

// 미래 날짜만 선택 가능
export const FutureDatesOnly: Story = {
  args: {
    modelValue: null,
    min: new Date(new Date().setDate(new Date().getDate() + 1)),
  },
  render: (args) => ({
    components: { DateRangePicker },
    setup() {
      const dateRange = ref<DateRange | null>(args.modelValue)

      return {
        args,
        dateRange,
        onUpdate: (value: DateRange | null) => {
          dateRange.value = value
        },
      }
    },
    template: `
      <div style="width: 400px;">
        <DateRangePicker
          v-bind="args"
          v-model="dateRange"
          @update:modelValue="onUpdate"
        />
        <div style="margin-top: 16px; font-size: 14px; color: #666;">
          미래 날짜만 선택 가능합니다 (예약/일정 관리용).
        </div>
      </div>
    `,
  }),
}

// 프리셋 없음
export const WithoutPresets: Story = {
  args: {
    modelValue: null,
    showPresets: false,
  },
  render: (args) => ({
    components: { DateRangePicker },
    setup() {
      const dateRange = ref<DateRange | null>(args.modelValue)

      return {
        args,
        dateRange,
        onUpdate: (value: DateRange | null) => {
          dateRange.value = value
        },
      }
    },
    template: `
      <div style="width: 400px;">
        <DateRangePicker
          v-bind="args"
          v-model="dateRange"
          @update:modelValue="onUpdate"
        />
      </div>
    `,
  }),
}

// 영어 로케일
export const EnglishLocale: Story = {
  args: {
    modelValue: null,
    locale: 'en',
    placeholder: 'Select date range',
    startPlaceholder: 'Start date',
    endPlaceholder: 'End date',
  },
  render: (args) => ({
    components: { DateRangePicker },
    setup() {
      const dateRange = ref<DateRange | null>(args.modelValue)

      return {
        args,
        dateRange,
        onUpdate: (value: DateRange | null) => {
          dateRange.value = value
        },
      }
    },
    template: `
      <div style="width: 400px;">
        <DateRangePicker
          v-bind="args"
          v-model="dateRange"
          @update:modelValue="onUpdate"
        />
      </div>
    `,
  }),
}

// 커스텀 날짜 형식
export const CustomFormat: Story = {
  args: {
    modelValue: {
      start: new Date(),
      end: new Date(new Date().setDate(new Date().getDate() + 7)),
    },
    format: 'MM/DD/YYYY',
  },
  render: (args) => ({
    components: { DateRangePicker },
    setup() {
      const dateRange = ref<DateRange | null>(args.modelValue)

      return {
        args,
        dateRange,
        onUpdate: (value: DateRange | null) => {
          dateRange.value = value
        },
      }
    },
    template: `
      <div style="width: 400px;">
        <DateRangePicker
          v-bind="args"
          v-model="dateRange"
          @update:modelValue="onUpdate"
        />
      </div>
    `,
  }),
}

// 지우기 버튼 없음
export const NotClearable: Story = {
  args: {
    modelValue: {
      start: new Date(),
      end: new Date(new Date().setDate(new Date().getDate() + 3)),
    },
    clearable: false,
  },
  render: (args) => ({
    components: { DateRangePicker },
    setup() {
      const dateRange = ref<DateRange | null>(args.modelValue)

      return {
        args,
        dateRange,
        onUpdate: (value: DateRange | null) => {
          dateRange.value = value
        },
      }
    },
    template: `
      <div style="width: 400px;">
        <DateRangePicker
          v-bind="args"
          v-model="dateRange"
          @update:modelValue="onUpdate"
        />
      </div>
    `,
  }),
}

// 비활성화된 상태
export const Disabled: Story = {
  args: {
    modelValue: {
      start: new Date(new Date().setDate(new Date().getDate() - 7)),
      end: new Date(),
    },
    disabled: true,
  },
  render: (args) => ({
    components: { DateRangePicker },
    setup() {
      const dateRange = ref<DateRange | null>(args.modelValue)

      return {
        args,
        dateRange,
      }
    },
    template: `
      <div style="width: 400px;">
        <DateRangePicker
          v-bind="args"
          v-model="dateRange"
        />
      </div>
    `,
  }),
}

// 보고서 기간 선택
export const ReportPeriod: Story = {
  args: {
    modelValue: null,
    max: new Date(),
    showPresets: true,
    placeholder: '보고서 기간을 선택하세요',
    presets: [
      {
        label: '이번 주',
        value: () => {
          const now = new Date()
          const dayOfWeek = now.getDay()
          const start = new Date(now)
          start.setDate(now.getDate() - dayOfWeek)
          return { start, end: now }
        },
      },
      {
        label: '지난 주',
        value: () => {
          const now = new Date()
          const dayOfWeek = now.getDay()
          const end = new Date(now)
          end.setDate(now.getDate() - dayOfWeek - 1)
          const start = new Date(end)
          start.setDate(end.getDate() - 6)
          return { start, end }
        },
      },
      {
        label: '이번 분기',
        value: () => {
          const now = new Date()
          const quarter = Math.floor(now.getMonth() / 3)
          const start = new Date(now.getFullYear(), quarter * 3, 1)
          const end = new Date(now.getFullYear(), quarter * 3 + 3, 0)
          return { start, end: end > now ? now : end }
        },
      },
      {
        label: '올해',
        value: () => {
          const now = new Date()
          return {
            start: new Date(now.getFullYear(), 0, 1),
            end: now,
          }
        },
      },
    ],
  },
  render: (args) => ({
    components: { DateRangePicker },
    setup() {
      const dateRange = ref<DateRange | null>(args.modelValue)

      return {
        args,
        dateRange,
        onUpdate: (value: DateRange | null) => {
          dateRange.value = value
        },
      }
    },
    template: `
      <div style="width: 400px;">
        <DateRangePicker
          v-bind="args"
          v-model="dateRange"
          @update:modelValue="onUpdate"
        />
        <div v-if="dateRange" style="margin-top: 16px; font-size: 14px;">
          보고서 기간: {{ Math.ceil((dateRange.end.getTime() - dateRange.start.getTime()) / (1000 * 60 * 60 * 24)) + 1 }}일
        </div>
      </div>
    `,
  }),
}