import type { Meta, StoryObj } from '@storybook/vue3'
// action 함수를 간단한 console.log로 대체
const action = (name: string) => (...args: any[]) => {
  console.log(`[${name}]`, ...args)
}
import { ref } from 'vue'
import DatePicker from './DatePicker.vue'

const meta: Meta<typeof DatePicker> = {
  title: 'Forms/DateTimePicker/DatePicker',
  component: DatePicker,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component: `
### DatePicker 컴포넌트

사용자 친화적인 날짜 선택 컴포넌트입니다.

#### 주요 기능
- 달력 UI를 통한 직관적인 날짜 선택
- 키보드 네비게이션 완벽 지원
- 최소/최대 날짜 제한
- 다국어 지원 (한국어 기본)
- 커스터마이징 가능한 날짜 형식
- 오늘 날짜로 빠른 이동
- 월/년 빠른 선택
- 접근성 표준 준수 (ARIA)
- 반응형 디자인
        `,
      },
    },
  },
  argTypes: {
    modelValue: {
      control: { type: 'date' },
      description: '선택된 날짜',
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
      description: '주의 시작 요일 (0=일요일, 1=월요일)',
    },
    todayButtonText: {
      control: { type: 'text' },
      description: '오늘 버튼 텍스트',
    },
  },
  args: {
    clearable: true,
    placeholder: '날짜를 선택하세요',
    format: 'YYYY-MM-DD',
    locale: 'ko',
    firstDayOfWeek: 0,
    todayButtonText: '오늘',
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
    components: { DatePicker },
    setup() {
      const date = ref(args.modelValue)
      
      return {
        args,
        date,
        onUpdate: (value: Date | null) => {
          date.value = value
          action('update:modelValue')(value)
        },
        onChange: action('change'),
        onClear: action('clear'),
      }
    },
    template: `
      <div style="width: 320px;">
        <DatePicker
          v-bind="args"
          v-model="date"
          @update:modelValue="onUpdate"
          @change="onChange"
          @clear="onClear"
        />
        <div style="margin-top: 16px; font-size: 14px;">
          선택된 날짜: {{ date ? date.toLocaleDateString('ko-KR') : '없음' }}
        </div>
      </div>
    `,
  }),
}

// 초기값이 있는 경우
export const WithInitialValue: Story = {
  args: {
    modelValue: new Date(),
  },
  render: (args) => ({
    components: { DatePicker },
    setup() {
      const date = ref(args.modelValue)
      
      return {
        args,
        date,
        onUpdate: (value: Date | null) => {
          date.value = value
          action('update:modelValue')(value)
        },
      }
    },
    template: `
      <div style="width: 320px;">
        <DatePicker
          v-bind="args"
          v-model="date"
          @update:modelValue="onUpdate"
        />
      </div>
    `,
  }),
}

// 날짜 범위 제한
export const WithDateRange: Story = {
  args: {
    modelValue: null,
    min: new Date(new Date().setDate(new Date().getDate() - 30)), // 30일 전
    max: new Date(new Date().setDate(new Date().getDate() + 30)), // 30일 후
  },
  render: (args) => ({
    components: { DatePicker },
    setup() {
      const date = ref(args.modelValue)
      
      return {
        args,
        date,
        onUpdate: (value: Date | null) => {
          date.value = value
        },
      }
    },
    template: `
      <div style="width: 320px;">
        <DatePicker
          v-bind="args"
          v-model="date"
          @update:modelValue="onUpdate"
        />
        <div style="margin-top: 16px; font-size: 14px; color: #666;">
          선택 가능한 날짜: 오늘 기준 ±30일
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
    components: { DatePicker },
    setup() {
      const date = ref(args.modelValue)
      
      return {
        args,
        date,
        onUpdate: (value: Date | null) => {
          date.value = value
        },
      }
    },
    template: `
      <div style="width: 320px;">
        <DatePicker
          v-bind="args"
          v-model="date"
          @update:modelValue="onUpdate"
        />
        <div style="margin-top: 16px; font-size: 14px; color: #666;">
          과거 날짜만 선택 가능합니다.
        </div>
      </div>
    `,
  }),
}

// 미래 날짜만 선택 가능
export const FutureDatesOnly: Story = {
  args: {
    modelValue: null,
    min: new Date(),
  },
  render: (args) => ({
    components: { DatePicker },
    setup() {
      const date = ref(args.modelValue)
      
      return {
        args,
        date,
        onUpdate: (value: Date | null) => {
          date.value = value
        },
      }
    },
    template: `
      <div style="width: 320px;">
        <DatePicker
          v-bind="args"
          v-model="date"
          @update:modelValue="onUpdate"
        />
        <div style="margin-top: 16px; font-size: 14px; color: #666;">
          오늘 이후 날짜만 선택 가능합니다.
        </div>
      </div>
    `,
  }),
}

// 영어 로케일
export const EnglishLocale: Story = {
  args: {
    modelValue: null,
    locale: 'en',
    placeholder: 'Select date',
    todayButtonText: 'Today',
  },
  render: (args) => ({
    components: { DatePicker },
    setup() {
      const date = ref(args.modelValue)
      
      return {
        args,
        date,
        onUpdate: (value: Date | null) => {
          date.value = value
        },
      }
    },
    template: `
      <div style="width: 320px;">
        <DatePicker
          v-bind="args"
          v-model="date"
          @update:modelValue="onUpdate"
        />
      </div>
    `,
  }),
}

// 월요일 시작
export const MondayFirst: Story = {
  args: {
    modelValue: null,
    firstDayOfWeek: 1,
  },
  render: (args) => ({
    components: { DatePicker },
    setup() {
      const date = ref(args.modelValue)
      
      return {
        args,
        date,
        onUpdate: (value: Date | null) => {
          date.value = value
        },
      }
    },
    template: `
      <div style="width: 320px;">
        <DatePicker
          v-bind="args"
          v-model="date"
          @update:modelValue="onUpdate"
        />
        <div style="margin-top: 16px; font-size: 14px; color: #666;">
          주의 시작이 월요일로 설정되었습니다.
        </div>
      </div>
    `,
  }),
}

// 커스텀 날짜 형식
export const CustomFormat: Story = {
  args: {
    modelValue: new Date(),
    format: 'YYYY년 MM월 DD일',
  },
  render: (args) => ({
    components: { DatePicker },
    setup() {
      const date = ref(args.modelValue)
      
      return {
        args,
        date,
        onUpdate: (value: Date | null) => {
          date.value = value
        },
      }
    },
    template: `
      <div style="width: 320px;">
        <DatePicker
          v-bind="args"
          v-model="date"
          @update:modelValue="onUpdate"
        />
      </div>
    `,
  }),
}

// 지우기 버튼 없음
export const NotClearable: Story = {
  args: {
    modelValue: new Date(),
    clearable: false,
  },
  render: (args) => ({
    components: { DatePicker },
    setup() {
      const date = ref(args.modelValue)
      
      return {
        args,
        date,
        onUpdate: (value: Date | null) => {
          date.value = value
        },
      }
    },
    template: `
      <div style="width: 320px;">
        <DatePicker
          v-bind="args"
          v-model="date"
          @update:modelValue="onUpdate"
        />
      </div>
    `,
  }),
}

// 비활성화된 상태
export const Disabled: Story = {
  args: {
    modelValue: new Date(),
    disabled: true,
  },
  render: (args) => ({
    components: { DatePicker },
    setup() {
      const date = ref(args.modelValue)
      
      return {
        args,
        date,
      }
    },
    template: `
      <div style="width: 320px;">
        <DatePicker
          v-bind="args"
          v-model="date"
        />
      </div>
    `,
  }),
}

// 생일 선택 (년도 범위)
export const BirthdayPicker: Story = {
  args: {
    modelValue: null,
    min: new Date(1900, 0, 1),
    max: new Date(),
    placeholder: '생년월일을 선택하세요',
  },
  render: (args) => ({
    components: { DatePicker },
    setup() {
      const date = ref(args.modelValue)
      
      return {
        args,
        date,
        onUpdate: (value: Date | null) => {
          date.value = value
        },
      }
    },
    template: `
      <div style="width: 320px;">
        <DatePicker
          v-bind="args"
          v-model="date"
          @update:modelValue="onUpdate"
        />
        <div v-if="date" style="margin-top: 16px; font-size: 14px;">
          나이: {{ new Date().getFullYear() - date.getFullYear() }}세
        </div>
      </div>
    `,
  }),
}

// 예약 날짜 선택
export const ReservationPicker: Story = {
  args: {
    modelValue: null,
    min: new Date(new Date().setDate(new Date().getDate() + 1)), // 내일부터
    max: new Date(new Date().setMonth(new Date().getMonth() + 3)), // 3개월 후까지
    placeholder: '예약 날짜를 선택하세요',
  },
  render: (args) => ({
    components: { DatePicker },
    setup() {
      const date = ref(args.modelValue)
      
      return {
        args,
        date,
        onUpdate: (value: Date | null) => {
          date.value = value
        },
      }
    },
    template: `
      <div style="width: 320px;">
        <DatePicker
          v-bind="args"
          v-model="date"
          @update:modelValue="onUpdate"
        />
        <div style="margin-top: 16px; font-size: 14px; color: #666;">
          내일부터 3개월 이내의 날짜만 예약 가능합니다.
        </div>
      </div>
    `,
  }),
}