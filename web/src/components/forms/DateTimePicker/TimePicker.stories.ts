import type { Meta, StoryObj } from '@storybook/vue3'
// action 함수를 간단한 console.log로 대체
const action = (name: string) => (...args: any[]) => {
  console.log(`[${name}]`, ...args)
}
import { ref } from 'vue'
import TimePicker from './TimePicker.vue'

const meta: Meta<typeof TimePicker> = {
  title: 'Forms/DateTimePicker/TimePicker',
  component: TimePicker,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component: `
### TimePicker 컴포넌트

사용자 친화적인 시간 선택 컴포넌트입니다.

#### 주요 기능
- 12시간/24시간 형식 지원
- 분 단위 간격 설정
- 최소/최대 시간 제한
- 현재 시간으로 빠른 설정
- 키보드 네비게이션 지원
- AM/PM 토글 (12시간 형식)
- 접근성 표준 준수 (ARIA)
- 반응형 디자인
        `,
      },
    },
  },
  argTypes: {
    modelValue: {
      control: { type: 'text' },
      description: '선택된 시간 (HH:mm 형식)',
    },
    min: {
      control: { type: 'text' },
      description: '선택 가능한 최소 시간',
    },
    max: {
      control: { type: 'text' },
      description: '선택 가능한 최대 시간',
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
    use24Hour: {
      control: { type: 'boolean' },
      description: '24시간 형식 사용 여부',
    },
    minuteStep: {
      control: { type: 'select' },
      options: [1, 5, 10, 15, 30],
      description: '분 선택 간격',
    },
    nowButtonText: {
      control: { type: 'text' },
      description: '현재 시간 버튼 텍스트',
    },
    confirmButtonText: {
      control: { type: 'text' },
      description: '확인 버튼 텍스트',
    },
  },
  args: {
    clearable: true,
    placeholder: '시간을 선택하세요',
    use24Hour: false,
    minuteStep: 5,
    nowButtonText: '현재',
    confirmButtonText: '확인',
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
    components: { TimePicker },
    setup() {
      const time = ref(args.modelValue)

      return {
        args,
        time,
        onUpdate: (value: string | null) => {
          time.value = value
          action('update:modelValue')(value)
        },
        onChange: action('change'),
        onClear: action('clear'),
      }
    },
    template: `
      <div style="width: 320px;">
        <TimePicker
          v-bind="args"
          v-model="time"
          @update:modelValue="onUpdate"
          @change="onChange"
          @clear="onClear"
        />
        <div style="margin-top: 16px; font-size: 14px;">
          선택된 시간: {{ time || '없음' }}
        </div>
      </div>
    `,
  }),
}

// 초기값이 있는 경우
export const WithInitialValue: Story = {
  args: {
    modelValue: '14:30',
  },
  render: (args) => ({
    components: { TimePicker },
    setup() {
      const time = ref(args.modelValue)

      return {
        args,
        time,
        onUpdate: (value: string | null) => {
          time.value = value
          action('update:modelValue')(value)
        },
      }
    },
    template: `
      <div style="width: 320px;">
        <TimePicker
          v-bind="args"
          v-model="time"
          @update:modelValue="onUpdate"
        />
      </div>
    `,
  }),
}

// 24시간 형식
export const Use24HourFormat: Story = {
  args: {
    modelValue: null,
    use24Hour: true,
  },
  render: (args) => ({
    components: { TimePicker },
    setup() {
      const time = ref(args.modelValue)

      return {
        args,
        time,
        onUpdate: (value: string | null) => {
          time.value = value
        },
      }
    },
    template: `
      <div style="width: 320px;">
        <TimePicker
          v-bind="args"
          v-model="time"
          @update:modelValue="onUpdate"
        />
        <div style="margin-top: 16px; font-size: 14px; color: #666;">
          24시간 형식을 사용합니다.
        </div>
      </div>
    `,
  }),
}

// 시간 범위 제한
export const WithTimeRange: Story = {
  args: {
    modelValue: null,
    min: '09:00',
    max: '18:00',
  },
  render: (args) => ({
    components: { TimePicker },
    setup() {
      const time = ref(args.modelValue)

      return {
        args,
        time,
        onUpdate: (value: string | null) => {
          time.value = value
        },
      }
    },
    template: `
      <div style="width: 320px;">
        <TimePicker
          v-bind="args"
          v-model="time"
          @update:modelValue="onUpdate"
        />
        <div style="margin-top: 16px; font-size: 14px; color: #666;">
          업무 시간(09:00 - 18:00)만 선택 가능합니다.
        </div>
      </div>
    `,
  }),
}

// 15분 간격
export const Minute15Step: Story = {
  args: {
    modelValue: null,
    minuteStep: 15,
  },
  render: (args) => ({
    components: { TimePicker },
    setup() {
      const time = ref(args.modelValue)

      return {
        args,
        time,
        onUpdate: (value: string | null) => {
          time.value = value
        },
      }
    },
    template: `
      <div style="width: 320px;">
        <TimePicker
          v-bind="args"
          v-model="time"
          @update:modelValue="onUpdate"
        />
        <div style="margin-top: 16px; font-size: 14px; color: #666;">
          15분 단위로 선택 가능합니다.
        </div>
      </div>
    `,
  }),
}

// 30분 간격
export const Minute30Step: Story = {
  args: {
    modelValue: null,
    minuteStep: 30,
  },
  render: (args) => ({
    components: { TimePicker },
    setup() {
      const time = ref(args.modelValue)

      return {
        args,
        time,
        onUpdate: (value: string | null) => {
          time.value = value
        },
      }
    },
    template: `
      <div style="width: 320px;">
        <TimePicker
          v-bind="args"
          v-model="time"
          @update:modelValue="onUpdate"
        />
        <div style="margin-top: 16px; font-size: 14px; color: #666;">
          30분 단위로 선택 가능합니다.
        </div>
      </div>
    `,
  }),
}

// 오전 시간대만
export const MorningOnly: Story = {
  args: {
    modelValue: null,
    min: '06:00',
    max: '12:00',
    placeholder: '오전 시간을 선택하세요',
  },
  render: (args) => ({
    components: { TimePicker },
    setup() {
      const time = ref(args.modelValue)

      return {
        args,
        time,
        onUpdate: (value: string | null) => {
          time.value = value
        },
      }
    },
    template: `
      <div style="width: 320px;">
        <TimePicker
          v-bind="args"
          v-model="time"
          @update:modelValue="onUpdate"
        />
      </div>
    `,
  }),
}

// 오후 시간대만
export const AfternoonOnly: Story = {
  args: {
    modelValue: null,
    min: '12:00',
    max: '23:59',
    placeholder: '오후 시간을 선택하세요',
  },
  render: (args) => ({
    components: { TimePicker },
    setup() {
      const time = ref(args.modelValue)

      return {
        args,
        time,
        onUpdate: (value: string | null) => {
          time.value = value
        },
      }
    },
    template: `
      <div style="width: 320px;">
        <TimePicker
          v-bind="args"
          v-model="time"
          @update:modelValue="onUpdate"
        />
      </div>
    `,
  }),
}

// 지우기 버튼 없음
export const NotClearable: Story = {
  args: {
    modelValue: '10:00',
    clearable: false,
  },
  render: (args) => ({
    components: { TimePicker },
    setup() {
      const time = ref(args.modelValue)

      return {
        args,
        time,
        onUpdate: (value: string | null) => {
          time.value = value
        },
      }
    },
    template: `
      <div style="width: 320px;">
        <TimePicker
          v-bind="args"
          v-model="time"
          @update:modelValue="onUpdate"
        />
      </div>
    `,
  }),
}

// 비활성화된 상태
export const Disabled: Story = {
  args: {
    modelValue: '15:45',
    disabled: true,
  },
  render: (args) => ({
    components: { TimePicker },
    setup() {
      const time = ref(args.modelValue)

      return {
        args,
        time,
      }
    },
    template: `
      <div style="width: 320px;">
        <TimePicker
          v-bind="args"
          v-model="time"
        />
      </div>
    `,
  }),
}

// 예약 시간 선택
export const AppointmentTime: Story = {
  args: {
    modelValue: null,
    min: '09:00',
    max: '17:00',
    minuteStep: 30,
    placeholder: '예약 시간을 선택하세요',
  },
  render: (args) => ({
    components: { TimePicker },
    setup() {
      const time = ref(args.modelValue)

      return {
        args,
        time,
        onUpdate: (value: string | null) => {
          time.value = value
        },
      }
    },
    template: `
      <div style="width: 320px;">
        <TimePicker
          v-bind="args"
          v-model="time"
          @update:modelValue="onUpdate"
        />
        <div style="margin-top: 16px; font-size: 14px; color: #666;">
          예약 가능 시간: 09:00 - 17:00 (30분 단위)
        </div>
      </div>
    `,
  }),
}

// 알람 시간 설정
export const AlarmTime: Story = {
  args: {
    modelValue: '07:00',
    use24Hour: true,
    minuteStep: 5,
    placeholder: '알람 시간 설정',
  },
  render: (args) => ({
    components: { TimePicker },
    setup() {
      const time = ref(args.modelValue)
      const isAlarmSet = ref(true)

      return {
        args,
        time,
        isAlarmSet,
        onUpdate: (value: string | null) => {
          time.value = value
          isAlarmSet.value = !!value
        },
      }
    },
    template: `
      <div style="width: 320px;">
        <TimePicker
          v-bind="args"
          v-model="time"
          @update:modelValue="onUpdate"
        />
        <div style="margin-top: 16px; font-size: 14px;">
          <span v-if="isAlarmSet">
            알람이 {{ time }}에 울립니다.
          </span>
          <span v-else style="color: #666;">
            알람이 설정되지 않았습니다.
          </span>
        </div>
      </div>
    `,
  }),
}