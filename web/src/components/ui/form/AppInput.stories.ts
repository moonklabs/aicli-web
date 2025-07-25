import type { Meta, StoryObj } from '@storybook/vue3'
import { action } from '@storybook/addon-actions'
import { ref } from 'vue'
import AppInput from './AppInput.vue'
import AppFormField from './AppFormField.vue'

const meta: Meta<typeof AppInput> = {
  title: 'UI/Form/AppInput',
  component: AppInput,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component: `
### AppInput 컴포넌트

다양한 타입과 기능을 지원하는 입력 필드 컴포넌트입니다.

#### 주요 기능
- 다양한 입력 타입 지원 (text, password, email, number, tel, url)
- 크기 조절 (small, medium, large)
- 상태 표시 (default, success, warning, error)
- 클리어 버튼 기능
- 비밀번호 표시/숨기기 기능
- 문자 수 표시 기능
- prefix/suffix 슬롯 지원
- 완전한 접근성 지원
- 다크 모드 지원
        `,
      },
    },
  },
  argTypes: {
    modelValue: {
      control: { type: 'text' },
      description: '입력 값',
    },
    type: {
      control: { type: 'select' },
      options: ['text', 'password', 'email', 'number', 'tel', 'url'],
      description: '입력 필드 타입',
    },
    size: {
      control: { type: 'select' },
      options: ['small', 'medium', 'large'],
      description: '입력 필드 크기',
    },
    status: {
      control: { type: 'select' },
      options: ['default', 'success', 'warning', 'error'],
      description: '입력 필드 상태',
    },
    placeholder: {
      control: { type: 'text' },
      description: '플레이스홀더 텍스트',
    },
    disabled: {
      control: { type: 'boolean' },
      description: '비활성화 상태',
    },
    readonly: {
      control: { type: 'boolean' },
      description: '읽기 전용 상태',
    },
    clearable: {
      control: { type: 'boolean' },
      description: '클리어 버튼 표시',
    },
    showCount: {
      control: { type: 'boolean' },
      description: '문자 수 표시',
    },
    maxlength: {
      control: { type: 'number' },
      description: '최대 문자 수',
    },
    round: {
      control: { type: 'boolean' },
      description: '둥근 모서리',
    },
  },
  args: {
    'onUpdate:value': action('update:value'),
    onFocus: action('focused'),
    onBlur: action('blurred'),
    onClear: action('cleared'),
    onChange: action('changed'),
    onInput: action('input'),
  },
}

export default meta
type Story = StoryObj<typeof meta>;

// 기본 스토리
export const Default: Story = {
  args: {
    placeholder: '텍스트를 입력하세요',
    size: 'medium',
  },
  render: (args) => ({
    components: { AppInput },
    setup() {
      const value = ref('')
      return { args, value }
    },
    template: '<AppInput v-model:value="value" v-bind="args" />',
  }),
}

// 타입별 입력 필드들
export const Types: Story = {
  render: () => ({
    components: { AppInput, AppFormField },
    setup() {
      const textValue = ref('')
      const passwordValue = ref('')
      const emailValue = ref('')
      const numberValue = ref('')
      const telValue = ref('')
      const urlValue = ref('')

      return {
        textValue,
        passwordValue,
        emailValue,
        numberValue,
        telValue,
        urlValue,
      }
    },
    template: `
      <div class="w-80 space-y-4">
        <AppFormField label="텍스트">
          <AppInput v-model:value="textValue" type="text" placeholder="일반 텍스트" />
        </AppFormField>
        
        <AppFormField label="비밀번호">
          <AppInput v-model:value="passwordValue" type="password" placeholder="비밀번호 입력" />
        </AppFormField>
        
        <AppFormField label="이메일">
          <AppInput v-model:value="emailValue" type="email" placeholder="이메일 주소" />
        </AppFormField>
        
        <AppFormField label="숫자">
          <AppInput v-model:value="numberValue" type="number" placeholder="숫자 입력" />
        </AppFormField>
        
        <AppFormField label="전화번호">
          <AppInput v-model:value="telValue" type="tel" placeholder="전화번호" />
        </AppFormField>
        
        <AppFormField label="URL">
          <AppInput v-model:value="urlValue" type="url" placeholder="웹사이트 주소" />
        </AppFormField>
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: '다양한 타입의 입력 필드들을 보여줍니다.',
      },
    },
  },
}

// 크기별 입력 필드들
export const Sizes: Story = {
  render: () => ({
    components: { AppInput },
    setup() {
      const smallValue = ref('')
      const mediumValue = ref('')
      const largeValue = ref('')

      return { smallValue, mediumValue, largeValue }
    },
    template: `
      <div class="w-80 space-y-4">
        <AppInput v-model:value="smallValue" size="small" placeholder="Small 크기" />
        <AppInput v-model:value="mediumValue" size="medium" placeholder="Medium 크기" />
        <AppInput v-model:value="largeValue" size="large" placeholder="Large 크기" />
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: '다양한 크기의 입력 필드들을 보여줍니다.',
      },
    },
  },
}

// 상태별 입력 필드들
export const States: Story = {
  render: () => ({
    components: { AppInput, AppFormField },
    setup() {
      const defaultValue = ref('기본 상태')
      const successValue = ref('성공 상태')
      const warningValue = ref('경고 상태')
      const errorValue = ref('오류 상태')

      return { defaultValue, successValue, warningValue, errorValue }
    },
    template: `
      <div class="w-80 space-y-4">
        <AppFormField label="기본 상태">
          <AppInput v-model:value="defaultValue" status="default" />
        </AppFormField>
        
        <AppFormField label="성공 상태" status="success" feedback="입력이 성공적으로 완료되었습니다.">
          <AppInput v-model:value="successValue" status="success" />
        </AppFormField>
        
        <AppFormField label="경고 상태" status="warning" feedback="입력에 주의가 필요합니다.">
          <AppInput v-model:value="warningValue" status="warning" />
        </AppFormField>
        
        <AppFormField label="오류 상태" status="error" feedback="입력 형식이 올바르지 않습니다.">
          <AppInput v-model:value="errorValue" status="error" />
        </AppFormField>
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: '다양한 상태의 입력 필드들을 보여줍니다.',
      },
    },
  },
}

// 클리어 기능
export const Clearable: Story = {
  render: () => ({
    components: { AppInput, AppFormField },
    setup() {
      const value = ref('클리어 버튼을 테스트해보세요')

      return { value }
    },
    template: `
      <div class="w-80">
        <AppFormField label="클리어 가능한 입력" help="입력 필드 우측의 X 버튼을 클릭하면 내용이 지워집니다.">
          <AppInput v-model:value="value" :clearable="true" placeholder="텍스트를 입력하세요" />
        </AppFormField>
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: '클리어 버튼 기능을 가진 입력 필드입니다.',
      },
    },
  },
}

// 비밀번호 표시/숨기기
export const PasswordToggle: Story = {
  render: () => ({
    components: { AppInput, AppFormField },
    setup() {
      const password = ref('MySecurePassword123!')

      return { password }
    },
    template: `
      <div class="w-80">
        <AppFormField label="비밀번호" help="눈 모양 아이콘을 클릭하여 비밀번호를 표시하거나 숨길 수 있습니다.">
          <AppInput 
            v-model:value="password" 
            type="password" 
            placeholder="비밀번호를 입력하세요" 
          />
        </AppFormField>
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: '비밀번호 표시/숨기기 토글 기능을 가진 입력 필드입니다.',
      },
    },
  },
}

// 문자 수 표시
export const CharacterCount: Story = {
  render: () => ({
    components: { AppInput, AppFormField },
    setup() {
      const shortText = ref('짧은 텍스트')
      const longText = ref('문자 수 제한이 있는 긴 텍스트입니다.')

      return { shortText, longText }
    },
    template: `
      <div class="w-80 space-y-4">
        <AppFormField label="문자 수 표시" help="현재 입력된 문자 수가 표시됩니다.">
          <AppInput 
            v-model:value="shortText" 
            :show-count="true" 
            placeholder="텍스트를 입력하세요" 
          />
        </AppFormField>
        
        <AppFormField label="문자 수 제한" help="최대 50자까지 입력 가능합니다.">
          <AppInput 
            v-model:value="longText" 
            :show-count="true" 
            :maxlength="50" 
            placeholder="최대 50자까지 입력하세요" 
          />
        </AppFormField>
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: '문자 수 표시 및 제한 기능을 가진 입력 필드들입니다.',
      },
    },
  },
}

// Prefix/Suffix 슬롯
export const WithPrefixSuffix: Story = {
  render: () => ({
    components: { AppInput, AppFormField },
    setup() {
      const amount = ref('1000')
      const username = ref('johndoe')
      const url = ref('mywebsite')

      return { amount, username, url }
    },
    template: `
      <div class="w-80 space-y-4">
        <AppFormField label="금액 입력">
          <AppInput v-model:value="amount" type="number" placeholder="0">
            <template #prefix>
              <span class="text-gray-500">₩</span>
            </template>
            <template #suffix>
              <span class="text-gray-500">원</span>
            </template>
          </AppInput>
        </AppFormField>
        
        <AppFormField label="사용자명">
          <AppInput v-model:value="username" placeholder="사용자명">
            <template #prefix>
              <svg class="w-4 h-4 text-gray-400" fill="currentColor" viewBox="0 0 16 16">
                <path d="M8 8a3 3 0 1 0 0-6 3 3 0 0 0 0 6zm2-3a2 2 0 1 1-4 0 2 2 0 0 1 4 0zm4 8c0 1-1 1-1 1H3s-1 0-1-1 1-4 6-4 6 3 6 4zm-1-.004c-.001-.246-.154-.986-.832-1.664C11.516 10.68 10.289 10 8 10c-2.29 0-3.516.68-4.168 1.332-.678.678-.83 1.418-.832 1.664h10z"/>
              </svg>
            </template>
          </AppInput>
        </AppFormField>
        
        <AppFormField label="웹사이트 URL">
          <AppInput v-model:value="url" placeholder="사이트명">
            <template #prefix>
              <span class="text-gray-500">https://</span>
            </template>
            <template #suffix>
              <span class="text-gray-500">.com</span>
            </template>
          </AppInput>
        </AppFormField>
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: 'prefix와 suffix 슬롯을 활용한 입력 필드들입니다.',
      },
    },
  },
}

// 비활성화 및 읽기 전용
export const DisabledAndReadonly: Story = {
  render: () => ({
    components: { AppInput, AppFormField },
    setup() {
      const normalValue = ref('일반 입력')
      const disabledValue = ref('비활성화된 입력')
      const readonlyValue = ref('읽기 전용 입력')

      return { normalValue, disabledValue, readonlyValue }
    },
    template: `
      <div class="w-80 space-y-4">
        <AppFormField label="일반 입력">
          <AppInput v-model:value="normalValue" placeholder="편집 가능" />
        </AppFormField>
        
        <AppFormField label="비활성화">
          <AppInput v-model:value="disabledValue" :disabled="true" />
        </AppFormField>
        
        <AppFormField label="읽기 전용">
          <AppInput v-model:value="readonlyValue" :readonly="true" />
        </AppFormField>
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: '비활성화 및 읽기 전용 상태의 입력 필드들입니다.',
      },
    },
  },
}

// 둥근 모서리
export const Rounded: Story = {
  render: () => ({
    components: { AppInput },
    setup() {
      const value = ref('')

      return { value }
    },
    template: `
      <div class="w-80">
        <AppInput 
          v-model:value="value" 
          :round="true" 
          placeholder="둥근 모서리 입력 필드" 
        />
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: '둥근 모서리를 가진 입력 필드입니다.',
      },
    },
  },
}

// 접근성 테스트
export const Accessibility: Story = {
  render: () => ({
    components: { AppInput, AppFormField },
    setup() {
      const firstName = ref('')
      const lastName = ref('')
      const email = ref('')
      const password = ref('')

      return { firstName, lastName, email, password }
    },
    template: `
      <div class="w-80 space-y-4">
        <div>
          <h3 class="text-lg font-semibold mb-4">접근성 테스트 폼</h3>
          <p class="text-sm text-gray-600 mb-4">
            Tab 키로 필드 간 이동, Escape 키로 클리어(clearable 필드), 
            스크린 리더 호환성 등을 테스트해보세요.
          </p>
        </div>
        
        <AppFormField 
          label="이름" 
          :required="true"
          help="필수 입력 항목입니다."
        >
          <AppInput 
            v-model:value="firstName" 
            placeholder="이름을 입력하세요"
            :required="true"
            aria-label="이름 입력"
          />
        </AppFormField>
        
        <AppFormField 
          label="성" 
          :required="true"
        >
          <AppInput 
            v-model:value="lastName" 
            placeholder="성을 입력하세요"
            :required="true"
            :clearable="true"
            aria-label="성 입력"
          />
        </AppFormField>
        
        <AppFormField 
          label="이메일 주소"
          :required="true"
          :status="email && !email.includes('@') ? 'error' : 'default'"
          :feedback="email && !email.includes('@') ? '올바른 이메일 형식이 아닙니다.' : ''"
        >
          <AppInput 
            v-model:value="email" 
            type="email" 
            placeholder="example@email.com"
            :required="true"
            :clearable="true"
            aria-label="이메일 주소 입력"
          />
        </AppFormField>
        
        <AppFormField 
          label="비밀번호"
          :required="true"
          help="8자 이상의 비밀번호를 입력해주세요."
        >
          <AppInput 
            v-model:value="password" 
            type="password" 
            placeholder="비밀번호"
            :required="true"
            aria-label="비밀번호 입력"
            :show-count="true"
            :maxlength="50"
          />
        </AppFormField>
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: '접근성 기능들을 테스트할 수 있는 입력 필드들입니다. 키보드 네비게이션, ARIA 라벨, 스크린 리더 지원 등을 확인해보세요.',
      },
    },
  },
}