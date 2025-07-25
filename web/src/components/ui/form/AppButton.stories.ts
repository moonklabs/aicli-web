import type { Meta, StoryObj } from '@storybook/vue3'
import { action } from '@storybook/addon-actions'
import AppButton from './AppButton.vue'

const meta: Meta<typeof AppButton> = {
  title: 'UI/Form/AppButton',
  component: AppButton,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component: `
### AppButton 컴포넌트

고도로 커스터마이징 가능하고 접근성을 준수하는 버튼 컴포넌트입니다.

#### 주요 기능
- 다양한 크기 (small, medium, large)
- 다양한 변형 (solid, outline, ghost, text)
- 다양한 타입 (default, primary, success, warning, error, info)
- 로딩 상태 지원
- 아이콘 슬롯 지원
- 완전한 접근성 지원 (ARIA, 키보드 네비게이션)
- 다크 모드 지원
        `,
      },
    },
  },
  argTypes: {
    type: {
      control: { type: 'select' },
      options: ['default', 'primary', 'success', 'warning', 'error', 'info'],
      description: '버튼의 의미적 타입',
    },
    size: {
      control: { type: 'select' },
      options: ['small', 'medium', 'large'],
      description: '버튼 크기',
    },
    variant: {
      control: { type: 'select' },
      options: ['solid', 'outline', 'ghost', 'text'],
      description: '버튼 스타일 변형',
    },
    disabled: {
      control: { type: 'boolean' },
      description: '비활성화 상태',
    },
    loading: {
      control: { type: 'boolean' },
      description: '로딩 상태',
    },
    block: {
      control: { type: 'boolean' },
      description: '전체 너비 버튼',
    },
    round: {
      control: { type: 'boolean' },
      description: '둥근 모서리',
    },
    circle: {
      control: { type: 'boolean' },
      description: '원형 버튼',
    },
    htmlType: {
      control: { type: 'select' },
      options: ['button', 'submit', 'reset'],
      description: 'HTML 버튼 타입',
    },
    onClick: {
      action: 'clicked',
      description: '클릭 이벤트 핸들러',
    },
  },
  args: {
    onClick: action('clicked'),
    onFocus: action('focused'),
    onBlur: action('blurred'),
    onKeydown: action('keydown'),
  },
}

export default meta
type Story = StoryObj<typeof meta>;

// 기본 스토리
export const Default: Story = {
  args: {
    type: 'default',
    size: 'medium',
    variant: 'solid',
  },
  render: (args) => ({
    components: { AppButton },
    setup() {
      return { args }
    },
    template: '<AppButton v-bind="args">기본 버튼</AppButton>',
  }),
}

// 타입별 버튼들
export const Types: Story = {
  render: () => ({
    components: { AppButton },
    template: `
      <div class="flex flex-wrap gap-4">
        <AppButton type="default">Default</AppButton>
        <AppButton type="primary">Primary</AppButton>
        <AppButton type="success">Success</AppButton>
        <AppButton type="warning">Warning</AppButton>
        <AppButton type="error">Error</AppButton>
        <AppButton type="info">Info</AppButton>
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: '다양한 타입의 버튼들을 보여줍니다.',
      },
    },
  },
}

// 변형별 버튼들
export const Variants: Story = {
  render: () => ({
    components: { AppButton },
    template: `
      <div class="space-y-4">
        <div class="flex flex-wrap gap-4">
          <AppButton type="primary" variant="solid">Solid</AppButton>
          <AppButton type="primary" variant="outline">Outline</AppButton>
          <AppButton type="primary" variant="ghost">Ghost</AppButton>
          <AppButton type="primary" variant="text">Text</AppButton>
        </div>
        <div class="flex flex-wrap gap-4">
          <AppButton type="error" variant="solid">Error Solid</AppButton>
          <AppButton type="error" variant="outline">Error Outline</AppButton>
          <AppButton type="error" variant="ghost">Error Ghost</AppButton>
          <AppButton type="error" variant="text">Error Text</AppButton>
        </div>
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: '다양한 변형 스타일의 버튼들을 보여줍니다.',
      },
    },
  },
}

// 크기별 버튼들
export const Sizes: Story = {
  render: () => ({
    components: { AppButton },
    template: `
      <div class="flex items-end gap-4">
        <AppButton type="primary" size="small">Small</AppButton>
        <AppButton type="primary" size="medium">Medium</AppButton>
        <AppButton type="primary" size="large">Large</AppButton>
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: '다양한 크기의 버튼들을 보여줍니다.',
      },
    },
  },
}

// 아이콘과 함께 사용
export const WithIcons: Story = {
  render: () => ({
    components: { AppButton },
    template: `
      <div class="flex flex-wrap gap-4">
        <AppButton type="primary">
          <template #icon>
            <svg viewBox="0 0 16 16" fill="currentColor" class="w-4 h-4">
              <path d="M8 4a.905.905 0 0 0-.9.995l.35 3.507a.552.552 0 0 0 1.1 0l.35-3.507A.905.905 0 0 0 8 4zm.002 6a1 1 0 1 0 0 2 1 1 0 0 0 0-2z"/>
            </svg>
          </template>
          정보
        </AppButton>
        
        <AppButton type="success">
          <template #icon>
            <svg viewBox="0 0 16 16" fill="currentColor" class="w-4 h-4">
              <path d="M13.854 3.646a.5.5 0 0 1 0 .708l-7 7a.5.5 0 0 1-.708 0l-3.5-3.5a.5.5 0 1 1 .708-.708L6.5 10.293l6.646-6.647a.5.5 0 0 1 .708 0z"/>
            </svg>
          </template>
          저장
        </AppButton>
        
        <AppButton type="error" variant="outline">
          삭제
          <template #suffix>
            <svg viewBox="0 0 16 16" fill="currentColor" class="w-4 h-4">
              <path d="M6.5 1h3a.5.5 0 0 1 .5.5v1H6v-1a.5.5 0 0 1 .5-.5zM11 2.5v-1A1.5 1.5 0 0 0 9.5 0h-3A1.5 1.5 0 0 0 5 1.5v1H2.506a.58.58 0 0 0-.01 1.152l.557 9.504A2 2 0 0 0 5.046 15h5.908a2 2 0 0 0 1.993-1.844l.557-9.504a.58.58 0 0 0-.01-1.152H11z"/>
            </svg>
          </template>
        </AppButton>
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: '아이콘과 함께 사용하는 버튼들입니다. prefix, suffix 슬롯을 활용할 수 있습니다.',
      },
    },
  },
}

// 아이콘 전용 버튼들
export const IconOnly: Story = {
  render: () => ({
    components: { AppButton },
    template: `
      <div class="flex gap-4">
        <AppButton type="primary" :circle="true" aria-label="설정">
          <template #icon>
            <svg viewBox="0 0 16 16" fill="currentColor" class="w-4 h-4">
              <path d="M8 4.754a3.246 3.246 0 1 0 0 6.492 3.246 3.246 0 0 0 0-6.492zM5.754 8a2.246 2.246 0 1 1 4.492 0 2.246 2.246 0 0 1-4.492 0z"/>
              <path d="M9.796 1.343c-.527-1.79-3.065-1.79-3.592 0l-.094.319a.873.873 0 0 1-1.255.52l-.292-.16c-1.64-.892-3.433.902-2.54 2.541l.159.292a.873.873 0 0 1-.52 1.255l-.319.094c-1.79.527-1.79 3.065 0 3.592l.319.094a.873.873 0 0 1 .52 1.255l-.16.292c-.892 1.64.901 3.434 2.541 2.54l.292-.159a.873.873 0 0 1 1.255.52l.094.319c.527 1.79 3.065 1.79 3.592 0l.094-.319a.873.873 0 0 1 1.255-.52l.292.16c1.64.893 3.434-.902 2.54-2.541l-.159-.292a.873.873 0 0 1 .52-1.255l.319-.094c1.79-.527 1.79-3.065 0-3.592l-.319-.094a.873.873 0 0 1-.52-1.255l.16-.292c.893-1.64-.902-3.433-2.541-2.54l-.292.159a.873.873 0 0 1-1.255-.52l-.094-.319z"/>
            </svg>
          </template>
        </AppButton>
        
        <AppButton type="success" size="small" :circle="true" aria-label="좋아요">
          <template #icon>
            <svg viewBox="0 0 16 16" fill="currentColor" class="w-3 h-3">
              <path d="m8 2.748-.717-.737C5.6.281 2.514.878 1.4 3.053c-.523 1.023-.641 2.5.314 4.385.92 1.815 2.834 3.989 6.286 6.357 3.452-2.368 5.365-4.542 6.286-6.357.955-1.886.838-3.362.314-4.385C13.486.878 10.4.28 8.717 2.01L8 2.748zM8 15C-7.333 4.868 3.279-3.04 7.824 1.143c.06.055.119.112.176.171a3.12 3.12 0 0 1 .176-.17C12.72-3.042 23.333 4.867 8 15z"/>
            </svg>
          </template>
        </AppButton>
        
        <AppButton type="error" variant="outline" size="large" :circle="true" aria-label="닫기">
          <template #icon>
            <svg viewBox="0 0 16 16" fill="currentColor" class="w-5 h-5">
              <path d="M4.646 4.646a.5.5 0 0 1 .708 0L8 7.293l2.646-2.647a.5.5 0 0 1 .708.708L8.707 8l2.647 2.646a.5.5 0 0 1-.708.708L8 8.707l-2.646 2.647a.5.5 0 0 1-.708-.708L7.293 8 4.646 5.354a.5.5 0 0 1 0-.708z"/>
            </svg>
          </template>
        </AppButton>
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: '아이콘만 있는 버튼들입니다. 접근성을 위해 aria-label을 반드시 제공해야 합니다.',
      },
    },
  },
}

// 로딩 상태
export const Loading: Story = {
  render: () => ({
    components: { AppButton },
    template: `
      <div class="flex flex-wrap gap-4">
        <AppButton type="primary" :loading="true">로딩 중...</AppButton>
        <AppButton type="success" variant="outline" :loading="true">저장 중...</AppButton>
        <AppButton type="warning" size="large" :loading="true">처리 중...</AppButton>
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: '로딩 상태의 버튼들입니다. 로딩 중에는 클릭이 비활성화됩니다.',
      },
    },
  },
}

// 비활성화 상태
export const Disabled: Story = {
  render: () => ({
    components: { AppButton },
    template: `
      <div class="flex flex-wrap gap-4">
        <AppButton type="primary" :disabled="true">비활성화</AppButton>
        <AppButton type="success" variant="outline" :disabled="true">비활성화</AppButton>
        <AppButton type="error" variant="ghost" :disabled="true">비활성화</AppButton>
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: '비활성화된 버튼들입니다.',
      },
    },
  },
}

// 블록 버튼
export const Block: Story = {
  render: () => ({
    components: { AppButton },
    template: `
      <div class="w-96 space-y-4">
        <AppButton type="primary" :block="true">전체 너비 버튼</AppButton>
        <AppButton type="success" variant="outline" :block="true">전체 너비 아웃라인</AppButton>
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: '전체 너비를 차지하는 블록 버튼들입니다.',
      },
    },
  },
}

// 둥근 버튼
export const Rounded: Story = {
  render: () => ({
    components: { AppButton },
    template: `
      <div class="flex flex-wrap gap-4">
        <AppButton type="primary" :round="true">둥근 버튼</AppButton>
        <AppButton type="success" variant="outline" :round="true">둥근 아웃라인</AppButton>
        <AppButton type="warning" variant="ghost" :round="true">둥근 고스트</AppButton>
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: '둥근 모서리를 가진 버튼들입니다.',
      },
    },
  },
}

// 상호작용 테스트
export const Interactive: Story = {
  args: {
    type: 'primary',
  },
  render: (args) => ({
    components: { AppButton },
    setup() {
      const handleClick = () => {
        alert('버튼이 클릭되었습니다!')
      }

      return { args, handleClick }
    },
    template: `
      <AppButton v-bind="args" @click="handleClick">
        클릭해보세요
      </AppButton>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: '클릭하면 알림이 표시되는 상호작용 버튼입니다.',
      },
    },
  },
}

// 접근성 테스트
export const Accessibility: Story = {
  render: () => ({
    components: { AppButton },
    template: `
      <div class="space-y-4">
        <div>
          <h3 class="text-lg font-semibold mb-2">키보드 네비게이션 테스트</h3>
          <p class="text-sm text-gray-600 mb-4">Tab 키로 이동하고 Enter 또는 Space 키로 활성화해보세요.</p>
          <div class="flex flex-wrap gap-4">
            <AppButton type="primary">첫 번째</AppButton>
            <AppButton type="success">두 번째</AppButton>
            <AppButton type="warning">세 번째</AppButton>
            <AppButton type="error">네 번째</AppButton>
          </div>
        </div>
        
        <div>
          <h3 class="text-lg font-semibold mb-2">ARIA 레이블 테스트</h3>
          <p class="text-sm text-gray-600 mb-4">스크린 리더가 올바르게 읽을 수 있도록 aria-label이 설정되었습니다.</p>
          <div class="flex flex-wrap gap-4">
            <AppButton type="primary" aria-label="사용자 프로필 편집하기" :circle="true">
              <template #icon>
                <svg viewBox="0 0 16 16" fill="currentColor" class="w-4 h-4">
                  <path d="M12.146.146a.5.5 0 0 1 .708 0l3 3a.5.5 0 0 1 0 .708L10.5 8.207l-3-3L12.146.146zM11.207 9.5L9 7.293L3.854 12.439a.5.5 0 0 1-.233.131L1.54 13.188a.5.5 0 0 1-.606-.606l.618-2.081a.5.5 0 0 1 .131-.232L6.707 5.5 9.5 8.293l1.707-1.793z"/>
                </svg>
              </template>
            </AppButton>
            <AppButton type="error" aria-label="항목 삭제하기" :circle="true">
              <template #icon>
                <svg viewBox="0 0 16 16" fill="currentColor" class="w-4 h-4">
                  <path d="M6.5 1h3a.5.5 0 0 1 .5.5v1H6v-1a.5.5 0 0 1 .5-.5zM11 2.5v-1A1.5 1.5 0 0 0 9.5 0h-3A1.5 1.5 0 0 0 5 1.5v1H2.506a.58.58 0 0 0-.01 1.152l.557 9.504A2 2 0 0 0 5.046 15h5.908a2 2 0 0 0 1.993-1.844l.557-9.504a.58.58 0 0 0-.01-1.152H11z"/>
                </svg>
              </template>
            </AppButton>
          </div>
        </div>
      </div>
    `,
  }),
  parameters: {
    docs: {
      description: {
        story: '접근성 기능들을 테스트할 수 있는 버튼들입니다. 키보드 네비게이션과 스크린 리더 지원을 확인해보세요.',
      },
    },
  },
}