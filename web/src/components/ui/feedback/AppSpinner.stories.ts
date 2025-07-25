import type { Meta, StoryObj } from '@storybook/vue3'
import AppSpinner from './AppSpinner.vue'

const meta: Meta<typeof AppSpinner> = {
  title: 'UI/Feedback/AppSpinner',
  component: AppSpinner,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component: '로딩 상태를 나타내는 스피너 컴포넌트입니다. 다양한 크기와 색상 변형을 지원하며, 설명 텍스트와 오버레이 모드를 제공합니다.',
      },
    },
  },
  tags: ['autodocs'],
  argTypes: {
    size: {
      control: { type: 'select' },
      options: ['small', 'medium', 'large'],
      description: '스피너의 크기',
    },
    variant: {
      control: { type: 'select' },
      options: ['default', 'primary', 'secondary', 'success', 'warning', 'error', 'info'],
      description: '스피너의 색상 변형',
    },
    description: {
      control: { type: 'text' },
      description: '스피너 하단에 표시될 설명 텍스트',
    },
    center: {
      control: { type: 'boolean' },
      description: '스피너를 중앙에 배치할지 여부',
    },
    overlay: {
      control: { type: 'boolean' },
      description: '전체 화면 오버레이로 표시할지 여부',
    },
    color: {
      control: { type: 'color' },
      description: '커스텀 색상 (CSS 색상 값)',
    },
    strokeWidth: {
      control: { type: 'number', min: 1, max: 10, step: 1 },
      description: '스피너 선의 두께',
    },
  },
}

export default meta
type Story = StoryObj<typeof meta>;

// 기본 스토리
export const Default: Story = {
  args: {
    size: 'medium',
    variant: 'primary',
  },
}

// 크기 변형
export const Sizes: Story = {
  render: () => ({
    components: { AppSpinner },
    template: `
      <div class="flex items-end gap-8">
        <div class="flex flex-col items-center gap-2">
          <AppSpinner size="small" />
          <span class="text-sm text-gray-600">Small</span>
        </div>
        <div class="flex flex-col items-center gap-2">
          <AppSpinner size="medium" />
          <span class="text-sm text-gray-600">Medium</span>
        </div>
        <div class="flex flex-col items-center gap-2">
          <AppSpinner size="large" />
          <span class="text-sm text-gray-600">Large</span>
        </div>
      </div>
    `,
  }),
}

// 색상 변형
export const Variants: Story = {
  render: () => ({
    components: { AppSpinner },
    template: `
      <div class="grid grid-cols-4 gap-6">
        <div class="flex flex-col items-center gap-2">
          <AppSpinner variant="default" />
          <span class="text-sm text-gray-600">Default</span>
        </div>
        <div class="flex flex-col items-center gap-2">
          <AppSpinner variant="primary" />
          <span class="text-sm text-gray-600">Primary</span>
        </div>
        <div class="flex flex-col items-center gap-2">
          <AppSpinner variant="secondary" />
          <span class="text-sm text-gray-600">Secondary</span>
        </div>
        <div class="flex flex-col items-center gap-2">
          <AppSpinner variant="success" />
          <span class="text-sm text-gray-600">Success</span>
        </div>
        <div class="flex flex-col items-center gap-2">
          <AppSpinner variant="warning" />
          <span class="text-sm text-gray-600">Warning</span>
        </div>
        <div class="flex flex-col items-center gap-2">
          <AppSpinner variant="error" />
          <span class="text-sm text-gray-600">Error</span>
        </div>
        <div class="flex flex-col items-center gap-2">
          <AppSpinner variant="info" />
          <span class="text-sm text-gray-600">Info</span>
        </div>
      </div>
    `,
  }),
}

// 설명 텍스트 포함
export const WithDescription: Story = {
  args: {
    size: 'medium',
    variant: 'primary',
    description: '데이터를 불러오는 중...',
  },
}

// 중앙 정렬
export const Centered: Story = {
  args: {
    size: 'large',
    variant: 'primary',
    description: '페이지를 로딩 중입니다',
    center: true,
  },
  parameters: {
    layout: 'fullscreen',
  },
  decorators: [
    () => ({
      template: '<div style="height: 300px; border: 1px dashed #ccc;"><story /></div>',
    }),
  ],
}

// 오버레이 모드
export const Overlay: Story = {
  args: {
    size: 'large',
    variant: 'primary',
    description: '처리 중입니다...',
    overlay: true,
  },
  parameters: {
    layout: 'fullscreen',
  },
  decorators: [
    () => ({
      template: `
        <div>
          <div class="p-8 space-y-4">
            <h1 class="text-2xl font-bold">샘플 페이지</h1>
            <p>이 페이지는 오버레이 스피너 데모를 위한 배경 콘텐츠입니다.</p>
            <div class="grid grid-cols-2 gap-4">
              <div class="h-32 bg-gray-100 rounded"></div>
              <div class="h-32 bg-gray-100 rounded"></div>
            </div>
          </div>
          <story />
        </div>
      `,
    }),
  ],
}

// 커스텀 색상
export const CustomColor: Story = {
  args: {
    size: 'medium',
    color: '#ff6b6b',
    description: '커스텀 색상 스피너',
  },
}

// 다양한 선 두께
export const StrokeWidths: Story = {
  render: () => ({
    components: { AppSpinner },
    template: `
      <div class="flex items-center gap-8">
        <div class="flex flex-col items-center gap-2">
          <AppSpinner :stroke-width="1" />
          <span class="text-sm text-gray-600">Stroke: 1</span>
        </div>
        <div class="flex flex-col items-center gap-2">
          <AppSpinner :stroke-width="3" />
          <span class="text-sm text-gray-600">Stroke: 3</span>
        </div>
        <div class="flex flex-col items-center gap-2">
          <AppSpinner :stroke-width="5" />
          <span class="text-sm text-gray-600">Stroke: 5</span>
        </div>
        <div class="flex flex-col items-center gap-2">
          <AppSpinner :stroke-width="8" />
          <span class="text-sm text-gray-600">Stroke: 8</span>
        </div>
      </div>
    `,
  }),
}

// 실제 사용 예제
export const InButton: Story = {
  render: () => ({
    components: { AppSpinner },
    template: `
      <div class="space-y-4">
        <button class="flex items-center px-4 py-2 bg-blue-600 text-white rounded-md disabled:opacity-50" disabled>
          <AppSpinner size="small" variant="secondary" class="mr-2" />
          처리 중...
        </button>
        
        <button class="flex items-center px-6 py-3 bg-green-600 text-white rounded-lg disabled:opacity-50" disabled>
          <AppSpinner size="small" variant="secondary" class="mr-2" />
          파일 업로드 중...
        </button>
        
        <button class="flex items-center px-3 py-1.5 text-sm bg-gray-600 text-white rounded disabled:opacity-50" disabled>
          <AppSpinner size="small" variant="secondary" class="mr-1.5" />
          저장 중
        </button>
      </div>
    `,
  }),
}

// 카드 내부 사용 예제
export const InCard: Story = {
  render: () => ({
    components: { AppSpinner },
    template: `
      <div class="max-w-md mx-auto bg-white border border-gray-200 rounded-lg shadow-sm">
        <div class="p-6">
          <h3 class="text-lg font-semibold mb-4">데이터 로딩</h3>
          <AppSpinner 
            size="medium" 
            variant="primary" 
            description="차트 데이터를 불러오는 중입니다..." 
            center 
          />
        </div>
      </div>
    `,
  }),
}

// 다크 모드 테스트
export const DarkMode: Story = {
  args: {
    size: 'medium',
    variant: 'primary',
    description: '다크 모드에서의 스피너',
  },
  parameters: {
    backgrounds: { default: 'dark' },
  },
  decorators: [
    () => ({
      template: '<div data-theme="dark" class="p-8"><story /></div>',
    }),
  ],
}