import type { Meta, StoryObj } from '@storybook/vue3'
// action 함수를 간단한 console.log로 대체
const action = (name: string) => (...args: any[]) => {
  console.log(`[${name}]`, ...args)
}
import { ref } from 'vue'
import AutoComplete from './AutoComplete.vue'

const meta: Meta<typeof AutoComplete> = {
  title: 'Forms/AutoComplete',
  component: AutoComplete,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component: `
### AutoComplete 컴포넌트

사용자 입력에 따라 자동완성 제안을 제공하는 고급 입력 컴포넌트입니다.

#### 주요 기능
- 실시간 자동완성 제안
- 비동기 데이터 로딩 지원
- 커스터마이징 가능한 필터링
- 검색어 하이라이팅
- 새 항목 생성 옵션
- 디바운싱 지원
- 키보드 네비게이션
- 접근성 표준 준수 (ARIA)
- 커스텀 레이블/값 추출 함수
- 로딩 상태 표시
        `,
      },
    },
  },
  argTypes: {
    modelValue: {
      control: { type: 'text' },
      description: '현재 입력 값',
    },
    suggestions: {
      control: { type: 'object' },
      description: '자동완성 제안 목록',
    },
    labelKey: {
      control: { type: 'text' },
      description: '레이블 추출 키 또는 함수',
    },
    valueKey: {
      control: { type: 'text' },
      description: '값 추출 키 또는 함수',
    },
    placeholder: {
      control: { type: 'text' },
      description: '플레이스홀더 텍스트',
    },
    disabled: {
      control: { type: 'boolean' },
      description: '비활성화 여부',
    },
    clearable: {
      control: { type: 'boolean' },
      description: '지우기 버튼 표시 여부',
    },
    loading: {
      control: { type: 'boolean' },
      description: '로딩 상태',
    },
    minLength: {
      control: { type: 'number' },
      description: '자동완성 시작 최소 글자 수',
    },
    maxSuggestions: {
      control: { type: 'number' },
      description: '최대 제안 항목 수',
    },
    delay: {
      control: { type: 'number' },
      description: '입력 디바운스 지연 시간 (ms)',
    },
    showCreateOption: {
      control: { type: 'boolean' },
      description: '새 항목 생성 옵션 표시 여부',
    },
    createOptionText: {
      control: { type: 'text' },
      description: '새 항목 생성 텍스트',
    },
    emptyText: {
      control: { type: 'text' },
      description: '결과가 없을 때 표시할 텍스트',
    },
    caseSensitive: {
      control: { type: 'boolean' },
      description: '대소문자 구분 여부',
    },
    highlightMatches: {
      control: { type: 'boolean' },
      description: '매칭된 텍스트 하이라이트 여부',
    },
  },
  args: {
    placeholder: 'Type to search...',
    clearable: true,
    minLength: 1,
    maxSuggestions: 10,
    delay: 300,
    createOptionText: 'Create',
    emptyText: 'No results found',
    caseSensitive: false,
    highlightMatches: true,
  },
}

export default meta
type Story = StoryObj<typeof meta>

// 기본 제안 데이터
const countries = [
  'Afghanistan', 'Albania', 'Algeria', 'Andorra', 'Angola',
  'Argentina', 'Armenia', 'Australia', 'Austria', 'Azerbaijan',
  'Bahamas', 'Bahrain', 'Bangladesh', 'Barbados', 'Belarus',
  'Belgium', 'Belize', 'Benin', 'Bhutan', 'Bolivia',
  'Brazil', 'Bulgaria', 'Burkina Faso', 'Burundi', 'Cambodia',
  'Canada', 'Chile', 'China', 'Colombia', 'Costa Rica',
  'Croatia', 'Cuba', 'Cyprus', 'Czech Republic', 'Denmark',
  'Egypt', 'Estonia', 'Ethiopia', 'Finland', 'France',
  'Germany', 'Greece', 'Guatemala', 'Haiti', 'Honduras',
  'Hungary', 'Iceland', 'India', 'Indonesia', 'Iran',
  'Iraq', 'Ireland', 'Israel', 'Italy', 'Jamaica',
  'Japan', 'Jordan', 'Kazakhstan', 'Kenya', 'Kuwait',
  'Latvia', 'Lebanon', 'Libya', 'Lithuania', 'Luxembourg',
  'Malaysia', 'Maldives', 'Mali', 'Malta', 'Mexico',
  'Monaco', 'Mongolia', 'Morocco', 'Myanmar', 'Nepal',
  'Netherlands', 'New Zealand', 'Nicaragua', 'Niger', 'Nigeria',
  'Norway', 'Oman', 'Pakistan', 'Panama', 'Paraguay',
  'Peru', 'Philippines', 'Poland', 'Portugal', 'Qatar',
  'Romania', 'Russia', 'Saudi Arabia', 'Senegal', 'Serbia',
  'Singapore', 'Slovakia', 'Slovenia', 'Somalia', 'South Africa',
  'South Korea', 'Spain', 'Sri Lanka', 'Sudan', 'Sweden',
  'Switzerland', 'Syria', 'Taiwan', 'Tajikistan', 'Thailand',
  'Tunisia', 'Turkey', 'Uganda', 'Ukraine', 'United Arab Emirates',
  'United Kingdom', 'United States', 'Uruguay', 'Uzbekistan', 'Venezuela',
  'Vietnam', 'Yemen', 'Zambia', 'Zimbabwe'
]

// 기본 스토리
export const Default: Story = {
  args: {
    modelValue: '',
    suggestions: countries,
  },
  render: (args) => ({
    components: { AutoComplete },
    setup() {
      const value = ref(args.modelValue)
      return {
        args,
        value,
        onUpdate: (val: string) => {
          value.value = val
          action('update:modelValue')(val)
        },
        onSelect: action('select'),
        onInput: action('input'),
        onSearch: action('search'),
      }
    },
    template: `
      <div style="width: 320px;">
        <AutoComplete
          v-bind="args"
          v-model="value"
          @update:modelValue="onUpdate"
          @select="onSelect"
          @input="onInput"
          @search="onSearch"
        />
        <div style="margin-top: 16px; font-size: 14px;">
          선택된 값: {{ value }}
        </div>
      </div>
    `,
  }),
}

// 객체 배열 사용
export const WithObjects: Story = {
  args: {
    modelValue: '',
    suggestions: [
      { id: 1, name: 'John Doe', email: 'john@example.com' },
      { id: 2, name: 'Jane Smith', email: 'jane@example.com' },
      { id: 3, name: 'Bob Johnson', email: 'bob@example.com' },
      { id: 4, name: 'Alice Brown', email: 'alice@example.com' },
      { id: 5, name: 'Charlie Davis', email: 'charlie@example.com' },
      { id: 6, name: 'Diana Evans', email: 'diana@example.com' },
      { id: 7, name: 'Edward Wilson', email: 'edward@example.com' },
      { id: 8, name: 'Fiona Garcia', email: 'fiona@example.com' },
    ],
    labelKey: (item: any) => `${item.name} (${item.email})`,
    valueKey: 'name',
  },
  render: (args) => ({
    components: { AutoComplete },
    setup() {
      const value = ref(args.modelValue)
      return {
        args,
        value,
        onUpdate: (val: string) => {
          value.value = val
          action('update:modelValue')(val)
        },
        onSelect: action('select'),
      }
    },
    template: `
      <div style="width: 320px;">
        <AutoComplete
          v-bind="args"
          v-model="value"
          @update:modelValue="onUpdate"
          @select="onSelect"
        />
      </div>
    `,
  }),
}

// 비동기 로딩
export const WithAsyncLoading: Story = {
  args: {
    modelValue: '',
    suggestions: [],
  },
  render: (args) => ({
    components: { AutoComplete },
    setup() {
      const value = ref(args.modelValue)
      const loading = ref(false)
      const suggestions = ref<string[]>([])

      const handleSearch = async (query: string) => {
        if (!query || query.length < 2) {
          suggestions.value = []
          return
        }

        loading.value = true
        action('search')(query)
        
        // API 호출 시뮬레이션
        await new Promise(resolve => setTimeout(resolve, 1000))
        
        // 필터링된 결과 반환
        suggestions.value = countries.filter(country =>
          country.toLowerCase().includes(query.toLowerCase())
        ).slice(0, 10)
        
        loading.value = false
      }

      return {
        args,
        value,
        loading,
        suggestions,
        handleSearch,
        onUpdate: (val: string) => {
          value.value = val
          action('update:modelValue')(val)
        },
      }
    },
    template: `
      <div style="width: 320px;">
        <AutoComplete
          v-bind="args"
          v-model="value"
          :suggestions="suggestions"
          :loading="loading"
          @update:modelValue="onUpdate"
          @search="handleSearch"
        />
        <div style="margin-top: 16px; font-size: 14px;">
          <div>입력값: {{ value }}</div>
          <div>로딩 상태: {{ loading ? '로딩 중...' : '대기' }}</div>
        </div>
      </div>
    `,
  }),
}

// 새 항목 생성 옵션
export const WithCreateOption: Story = {
  args: {
    modelValue: '',
    suggestions: ['Apple', 'Banana', 'Cherry', 'Date', 'Elderberry'],
    showCreateOption: true,
    createOptionText: '새 과일 추가',
  },
  render: (args) => ({
    components: { AutoComplete },
    setup() {
      const value = ref(args.modelValue)
      const suggestions = ref(args.suggestions)

      const handleCreate = (newValue: string) => {
        suggestions.value = [...suggestions.value, newValue]
        value.value = newValue
        action('create')(newValue)
      }

      return {
        args,
        value,
        suggestions,
        handleCreate,
        onUpdate: (val: string) => {
          value.value = val
          action('update:modelValue')(val)
        },
        onSelect: (item: any) => {
          if (item.__isCreateOption) {
            handleCreate(item.value)
          } else {
            action('select')(item)
          }
        },
      }
    },
    template: `
      <div style="width: 320px;">
        <AutoComplete
          v-bind="args"
          v-model="value"
          :suggestions="suggestions"
          @update:modelValue="onUpdate"
          @select="onSelect"
        />
        <div style="margin-top: 16px; font-size: 14px;">
          <div>선택된 값: {{ value }}</div>
          <div>전체 목록: {{ suggestions.join(', ') }}</div>
        </div>
      </div>
    `,
  }),
}

// 커스텀 필터링
export const WithCustomFilter: Story = {
  args: {
    modelValue: '',
    suggestions: [
      { code: 'US', name: 'United States', dial: '+1' },
      { code: 'UK', name: 'United Kingdom', dial: '+44' },
      { code: 'CA', name: 'Canada', dial: '+1' },
      { code: 'AU', name: 'Australia', dial: '+61' },
      { code: 'DE', name: 'Germany', dial: '+49' },
      { code: 'FR', name: 'France', dial: '+33' },
      { code: 'JP', name: 'Japan', dial: '+81' },
      { code: 'KR', name: 'South Korea', dial: '+82' },
      { code: 'CN', name: 'China', dial: '+86' },
      { code: 'IN', name: 'India', dial: '+91' },
    ],
    labelKey: (item: any) => `${item.name} (${item.code}) ${item.dial}`,
    valueKey: 'code',
    filterMethod: (query: string, item: any) => {
      const q = query.toLowerCase()
      return item.name.toLowerCase().includes(q) ||
             item.code.toLowerCase().includes(q) ||
             item.dial.includes(q)
    },
  },
  render: (args) => ({
    components: { AutoComplete },
    setup() {
      const value = ref(args.modelValue)
      return {
        args,
        value,
        onUpdate: (val: string) => {
          value.value = val
          action('update:modelValue')(val)
        },
      }
    },
    template: `
      <div style="width: 320px;">
        <AutoComplete
          v-bind="args"
          v-model="value"
          @update:modelValue="onUpdate"
        />
        <div style="margin-top: 16px; font-size: 14px;">
          <p>국가 코드, 이름, 또는 전화번호로 검색할 수 있습니다.</p>
          <p>선택된 코드: {{ value }}</p>
        </div>
      </div>
    `,
  }),
}

// 하이라이팅 비활성화
export const WithoutHighlight: Story = {
  args: {
    modelValue: '',
    suggestions: countries,
    highlightMatches: false,
  },
  render: (args) => ({
    components: { AutoComplete },
    setup() {
      const value = ref(args.modelValue)
      return {
        args,
        value,
        onUpdate: (val: string) => {
          value.value = val
        },
      }
    },
    template: `
      <div style="width: 320px;">
        <AutoComplete
          v-bind="args"
          v-model="value"
          @update:modelValue="onUpdate"
        />
      </div>
    `,
  }),
}

// 비활성화된 상태
export const Disabled: Story = {
  args: {
    modelValue: 'South Korea',
    suggestions: countries,
    disabled: true,
  },
  render: (args) => ({
    components: { AutoComplete },
    setup() {
      const value = ref(args.modelValue)
      return {
        args,
        value,
      }
    },
    template: `
      <div style="width: 320px;">
        <AutoComplete
          v-bind="args"
          v-model="value"
        />
      </div>
    `,
  }),
}

// 최소 길이 설정
export const WithMinLength: Story = {
  args: {
    modelValue: '',
    suggestions: countries,
    minLength: 3,
    placeholder: 'Type at least 3 characters...',
  },
  render: (args) => ({
    components: { AutoComplete },
    setup() {
      const value = ref(args.modelValue)
      return {
        args,
        value,
        onUpdate: (val: string) => {
          value.value = val
        },
      }
    },
    template: `
      <div style="width: 320px;">
        <AutoComplete
          v-bind="args"
          v-model="value"
          @update:modelValue="onUpdate"
        />
        <div style="margin-top: 16px; font-size: 14px; color: #666;">
          최소 3글자 이상 입력해야 자동완성이 시작됩니다.
        </div>
      </div>
    `,
  }),
}

// 대소문자 구분
export const CaseSensitive: Story = {
  args: {
    modelValue: '',
    suggestions: ['JavaScript', 'javascript', 'JAVASCRIPT', 'TypeScript', 'typescript', 'TYPESCRIPT'],
    caseSensitive: true,
  },
  render: (args) => ({
    components: { AutoComplete },
    setup() {
      const value = ref(args.modelValue)
      return {
        args,
        value,
        onUpdate: (val: string) => {
          value.value = val
        },
      }
    },
    template: `
      <div style="width: 320px;">
        <AutoComplete
          v-bind="args"
          v-model="value"
          @update:modelValue="onUpdate"
        />
        <div style="margin-top: 16px; font-size: 14px; color: #666;">
          대소문자를 구분하여 검색합니다.
        </div>
      </div>
    `,
  }),
}