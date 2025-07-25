import type { Preview } from '@storybook/vue3-vite'
import { app } from '@storybook/vue3'

// Naive UI 설정
import {
  NAlert,
  NAvatar,
  NAvatarGroup,
  NBadge,
  NBreadcrumb,
  NButton,
  NCard,
  NCheckbox,
  NConfigProvider,
  NDataTable,
  NDatePicker,
  NDivider,
  NDrawer,
  NDropdown,
  NEmpty,
  NForm,
  NFormItem,
  NIcon,
  NInput,
  NInputNumber,
  NLayout,
  NLayoutContent,
  NLayoutFooter,
  NLayoutHeader,
  NLayoutSider,
  NMenu,
  NModal,
  NPopover,
  NProgress,
  NRadio,
  NResult,
  NSelect,
  NSkeleton,
  NSlider,
  NSpace,
  NSpin,
  NStep,
  NSteps,
  NSwitch,
  NTabPane,
  NTabs,
  NTag,
  NTooltip,
  create,
} from 'naive-ui'

// Naive UI 컴포넌트 등록
const naive = create({
  components: [
    NConfigProvider,
    NButton,
    NInput,
    NCard,
    NDataTable,
    NForm,
    NFormItem,
    NSelect,
    NCheckbox,
    NRadio,
    NSwitch,
    NSlider,
    NInputNumber,
    NDatePicker,
    NModal,
    NDrawer,
    NPopover,
    NDropdown,
    NTooltip,
    NAlert,
    NProgress,
    NSkeleton,
    NSpin,
    NEmpty,
    NResult,
    NTag,
    NBadge,
    NIcon,
    NAvatarGroup,
    NAvatar,
    NSpace,
    NDivider,
    NLayout,
    NLayoutHeader,
    NLayoutSider,
    NLayoutContent,
    NLayoutFooter,
    NMenu,
    NBreadcrumb,
    NTabs,
    NTabPane,
    NSteps,
    NStep,
  ],
})

app.use(naive)

// 전역 스타일 임포트
import '../src/assets/main.css'
import '../src/styles/main.scss'

const preview: Preview = {
  parameters: {
    controls: {
      matchers: {
        color: /(background|color)$/i,
        date: /Date$/i,
      },
    },
    docs: {
      extractComponentDescription: (component, { notes }) => {
        if (notes) {
          return typeof notes === 'string' ? notes : notes.markdown || notes.text
        }
        return null
      },
    },
    backgrounds: {
      default: 'light',
      values: [
        {
          name: 'light',
          value: '#ffffff',
        },
        {
          name: 'dark',
          value: '#101014',
        },
        {
          name: 'gray',
          value: '#f8f9fa',
        },
      ],
    },
    viewport: {
      viewports: {
        mobile: {
          name: 'Mobile',
          styles: {
            width: '375px',
            height: '667px',
          },
        },
        tablet: {
          name: 'Tablet',
          styles: {
            width: '768px',
            height: '1024px',
          },
        },
        desktop: {
          name: 'Desktop',
          styles: {
            width: '1200px',
            height: '800px',
          },
        },
        wide: {
          name: 'Wide',
          styles: {
            width: '1440px',
            height: '900px',
          },
        },
      },
    },
  },
}

export default preview