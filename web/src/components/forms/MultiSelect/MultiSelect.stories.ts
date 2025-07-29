import type { Meta, StoryObj } from '@storybook/vue3'
// action 함수를 간단한 console.log로 대체
const action = (name: string) => (...args: any[]) => {
  console.log(`[${name}]`, ...args)
}
import { ref } from 'vue'
import MultiSelect from './MultiSelect.vue'

const meta: Meta<typeof MultiSelect> = {
  title: 'Forms/MultiSelect',
  component: MultiSelect,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component: `
### MultiSelect 컴포넌트

다중 선택이 가능한 고급 드롭다운 컴포넌트입니다.

#### 주요 기능
- 다중 항목 선택 지원
- 실시간 검색 기능
- 전체 선택/해제 기능
- 선택된 항목 태그 표시
- 키보드 네비게이션 지원
- 접근성 표준 준수 (ARIA)
- 커스텀 레이블/값 추출 함수 지원
- 항목 비활성화 지원
- 반응형 디자인
        `,
      },
    },
  },
  argTypes: {
    modelValue: {
      control: { type: 'object' },
      description: '선택된 값들의 배열',
    },
    options: {
      control: { type: 'object' },
      description: '선택 가능한 옵션 목록',
    },
    labelKey: {
      control: { type: 'text' },
      description: '레이블 추출 키 또는 함수',
    },
    valueKey: {
      control: { type: 'text' },
      description: '값 추출 키 또는 함수',
    },
    disabledKey: {
      control: { type: 'text' },
      description: '비활성화 상태 추출 키 또는 함수',
    },
    placeholder: {
      control: { type: 'text' },
      description: '플레이스홀더 텍스트',
    },
    disabled: {
      control: { type: 'boolean' },
      description: '전체 비활성화 여부',
    },
    searchable: {
      control: { type: 'boolean' },
      description: '검색 기능 사용 여부',
    },
    maxDisplay: {
      control: { type: 'number' },
      description: '표시할 최대 태그 수',
    },
    showSelectAll: {
      control: { type: 'boolean' },
      description: '전체 선택 버튼 표시 여부',
    },
    emptyText: {
      control: { type: 'text' },
      description: '옵션이 없을 때 표시할 텍스트',
    },
  },
  args: {
    placeholder: 'Select options...',
    searchable: true,
    maxDisplay: 3,
    showSelectAll: true,
    emptyText: 'No options available',
  },
}

export default meta
type Story = StoryObj<typeof meta>

// 기본 옵션 데이터
const defaultOptions = [
  { label: 'Vue.js', value: 'vue', icon: '🟢' },
  { label: 'React', value: 'react', icon: '🔵' },
  { label: 'Angular', value: 'angular', icon: '🔴' },
  { label: 'Svelte', value: 'svelte', icon: '🟠' },
  { label: 'Next.js', value: 'nextjs', icon: '⚫' },
  { label: 'Nuxt.js', value: 'nuxtjs', icon: '🟢' },
  { label: 'SolidJS', value: 'solid', icon: '🔷' },
  { label: 'Remix', value: 'remix', icon: '💿' },
]

// 기본 스토리
export const Default: Story = {
  args: {
    modelValue: [],
    options: defaultOptions,
  },
  render: (args) => ({
    components: { MultiSelect },
    setup() {
      const selected = ref(args.modelValue)
      return {
        args,
        selected,
        onChange: action('change'),
        onUpdate: (value: any[]) => {
          selected.value = value
          action('update:modelValue')(value)
        },
      }
    },
    template: `
      <div style="width: 320px;">
        <MultiSelect
          v-bind="args"
          v-model="selected"
          @update:modelValue="onUpdate"
          @change="onChange"
        />
        <div style="margin-top: 16px; font-size: 14px;">
          선택된 값: {{ selected }}
        </div>
      </div>
    `,
  }),
}

// 미리 선택된 값이 있는 경우
export const WithInitialValues: Story = {
  args: {
    modelValue: ['vue', 'react'],
    options: defaultOptions,
  },
  render: (args) => ({
    components: { MultiSelect },
    setup() {
      const selected = ref(args.modelValue)
      return {
        args,
        selected,
        onUpdate: (value: any[]) => {
          selected.value = value
          action('update:modelValue')(value)
        },
      }
    },
    template: `
      <div style="width: 320px;">
        <MultiSelect
          v-bind="args"
          v-model="selected"
          @update:modelValue="onUpdate"
        />
      </div>
    `,
  }),
}

// 비활성화된 옵션이 있는 경우
export const WithDisabledOptions: Story = {
  args: {
    modelValue: [],
    options: [
      { label: 'Vue.js', value: 'vue', disabled: false },
      { label: 'React', value: 'react', disabled: true },
      { label: 'Angular', value: 'angular', disabled: false },
      { label: 'Svelte', value: 'svelte', disabled: true },
      { label: 'Next.js', value: 'nextjs', disabled: false },
    ],
  },
  render: (args) => ({
    components: { MultiSelect },
    setup() {
      const selected = ref(args.modelValue)
      return {
        args,
        selected,
        onUpdate: (value: any[]) => {
          selected.value = value
        },
      }
    },
    template: `
      <div style="width: 320px;">
        <MultiSelect
          v-bind="args"
          v-model="selected"
          @update:modelValue="onUpdate"
        />
      </div>
    `,
  }),
}

// 검색 기능 비활성화
export const WithoutSearch: Story = {
  args: {
    modelValue: [],
    options: defaultOptions,
    searchable: false,
  },
  render: (args) => ({
    components: { MultiSelect },
    setup() {
      const selected = ref(args.modelValue)
      return {
        args,
        selected,
        onUpdate: (value: any[]) => {
          selected.value = value
        },
      }
    },
    template: `
      <div style="width: 320px;">
        <MultiSelect
          v-bind="args"
          v-model="selected"
          @update:modelValue="onUpdate"
        />
      </div>
    `,
  }),
}

// 많은 옵션이 있는 경우
export const WithManyOptions: Story = {
  args: {
    modelValue: [],
    options: Array.from({ length: 100 }, (_, i) => ({
      label: `Option ${i + 1}`,
      value: `option-${i + 1}`,
      category: `Category ${Math.floor(i / 10) + 1}`,
    })),
  },
  render: (args) => ({
    components: { MultiSelect },
    setup() {
      const selected = ref(args.modelValue)
      return {
        args,
        selected,
        onUpdate: (value: any[]) => {
          selected.value = value
        },
      }
    },
    template: `
      <div style="width: 320px;">
        <MultiSelect
          v-bind="args"
          v-model="selected"
          @update:modelValue="onUpdate"
        />
        <div style="margin-top: 16px; font-size: 14px;">
          선택된 항목 수: {{ selected.length }}
        </div>
      </div>
    `,
  }),
}

// 커스텀 레이블/값 함수
export const WithCustomExtractors: Story = {
  args: {
    modelValue: [],
    options: [
      { name: 'JavaScript', id: 'js', type: 'language' },
      { name: 'TypeScript', id: 'ts', type: 'language' },
      { name: 'Python', id: 'py', type: 'language' },
      { name: 'Java', id: 'java', type: 'language' },
      { name: 'Go', id: 'go', type: 'language' },
    ],
    labelKey: (option: any) => `${option.name} (${option.id})`,
    valueKey: 'id',
  },
  render: (args) => ({
    components: { MultiSelect },
    setup() {
      const selected = ref(args.modelValue)
      return {
        args,
        selected,
        onUpdate: (value: any[]) => {
          selected.value = value
        },
      }
    },
    template: `
      <div style="width: 320px;">
        <MultiSelect
          v-bind="args"
          v-model="selected"
          @update:modelValue="onUpdate"
        />
      </div>
    `,
  }),
}

// 비활성화된 상태
export const Disabled: Story = {
  args: {
    modelValue: ['vue', 'react'],
    options: defaultOptions,
    disabled: true,
  },
  render: (args) => ({
    components: { MultiSelect },
    setup() {
      const selected = ref(args.modelValue)
      return {
        args,
        selected,
      }
    },
    template: `
      <div style="width: 320px;">
        <MultiSelect
          v-bind="args"
          v-model="selected"
        />
      </div>
    `,
  }),
}

// 전체 선택 버튼 숨김
export const WithoutSelectAll: Story = {
  args: {
    modelValue: [],
    options: defaultOptions,
    showSelectAll: false,
  },
  render: (args) => ({
    components: { MultiSelect },
    setup() {
      const selected = ref(args.modelValue)
      return {
        args,
        selected,
        onUpdate: (value: any[]) => {
          selected.value = value
        },
      }
    },
    template: `
      <div style="width: 320px;">
        <MultiSelect
          v-bind="args"
          v-model="selected"
          @update:modelValue="onUpdate"
        />
      </div>
    `,
  }),
}

// 비동기 검색 시뮬레이션
export const WithAsyncSearch: Story = {
  args: {
    modelValue: [],
    options: defaultOptions,
  },
  render: (args) => ({
    components: { MultiSelect },
    setup() {
      const selected = ref(args.modelValue)
      const loading = ref(false)
      const filteredOptions = ref(args.options)

      const handleSearch = async (query: string) => {
        loading.value = true
        action('search')(query)

        // 비동기 검색 시뮬레이션
        await new Promise(resolve => setTimeout(resolve, 500))

        if (query) {
          filteredOptions.value = args.options.filter(option =>
            option.label.toLowerCase().includes(query.toLowerCase()),
          )
        } else {
          filteredOptions.value = args.options
        }

        loading.value = false
      }

      return {
        args,
        selected,
        filteredOptions,
        loading,
        handleSearch,
        onUpdate: (value: any[]) => {
          selected.value = value
        },
      }
    },
    template: `
      <div style="width: 320px;">
        <MultiSelect
          v-bind="args"
          v-model="selected"
          :options="filteredOptions"
          @update:modelValue="onUpdate"
          @search="handleSearch"
        />
        <div v-if="loading" style="margin-top: 8px; font-size: 14px; color: #666;">
          검색 중...
        </div>
      </div>
    `,
  }),
}