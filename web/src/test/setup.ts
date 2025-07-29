import { beforeAll, vi } from 'vitest'
import { config } from '@vue/test-utils'

// Chart.js 모킹
vi.mock('chart.js', () => ({
  Chart: vi.fn(() => ({
    destroy: vi.fn(),
    update: vi.fn(),
    resize: vi.fn(),
    getElementsAtEventForMode: vi.fn(() => []),
    resetZoom: vi.fn(),
    isZoomedOrPanned: vi.fn(() => false),
    canvas: {
      toDataURL: vi.fn(() => 'data:image/png;base64,test'),
    },
    data: { datasets: [] },
    options: {},
  })),
  registerables: [],
}))

// ResizeObserver 모킹
global.ResizeObserver = vi.fn(() => ({
  observe: vi.fn(),
  disconnect: vi.fn(),
  unobserve: vi.fn(),
}))

// IntersectionObserver 모킹
global.IntersectionObserver = vi.fn().mockImplementation(() => ({
  observe: vi.fn(),
  disconnect: vi.fn(),
  unobserve: vi.fn(),
  root: null,
  rootMargin: '',
  thresholds: [0],
  takeRecords: vi.fn().mockReturnValue([])
})) as any

// matchMedia 모킹
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: vi.fn().mockImplementation(query => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })),
})

// localStorage 모킹
const localStorageMock = {
  getItem: vi.fn(),
  setItem: vi.fn(),
  removeItem: vi.fn(),
  clear: vi.fn(),
}
Object.defineProperty(window, 'localStorage', {
  value: localStorageMock,
})

// requestAnimationFrame 모킹
global.requestAnimationFrame = vi.fn(cb => {
  setTimeout(cb, 0)
  return 1
})

global.cancelAnimationFrame = vi.fn()

// URL.createObjectURL 모킹
global.URL.createObjectURL = vi.fn(() => 'blob:test')
global.URL.revokeObjectURL = vi.fn()

// HTMLCanvasElement.getContext 모킹
HTMLCanvasElement.prototype.getContext = vi.fn((contextId: string) => {
  if (contextId === '2d') {
    return {
      // 메서드들
      fillRect: vi.fn(),
      clearRect: vi.fn(),
      getImageData: vi.fn(() => ({ data: new Array(4) })),
      putImageData: vi.fn(),
      createImageData: vi.fn(() => ({ data: new Array(4) })),
      setTransform: vi.fn(),
      drawImage: vi.fn(),
      save: vi.fn(),
      fillText: vi.fn(),
      restore: vi.fn(),
      beginPath: vi.fn(),
      moveTo: vi.fn(),
      lineTo: vi.fn(),
      closePath: vi.fn(),
      stroke: vi.fn(),
      translate: vi.fn(),
      scale: vi.fn(),
      rotate: vi.fn(),
      arc: vi.fn(),
      fill: vi.fn(),
      measureText: vi.fn(() => ({ width: 0 })),
      transform: vi.fn(),
      rect: vi.fn(),
      clip: vi.fn(),
      // 필수 속성들
      canvas: document.createElement('canvas'),
      globalAlpha: 1,
      globalCompositeOperation: 'source-over',
      strokeStyle: '#000000',
      fillStyle: '#000000',
      lineWidth: 1,
    } as any
  }
  return null
}) as any

// Vue Test Utils 글로벌 스텁 설정
config.global.stubs = {
  BaseFilter: true,
  NMessageProvider: true,
  NConfigProvider: true,
  NDialogProvider: true,
  NNotificationProvider: true,
  NLoadingBarProvider: true,
  NModal: true,
  NButton: true,
  NInput: true,
  NSelect: true,
  NSpace: true,
  NCard: true,
  NDataTable: true,
  NForm: true,
  NFormItem: true,
  NInputGroup: true,
  NInputGroupLabel: true,
  NCheckbox: true,
  NSpin: true,
  NEmpty: true,
  NResult: true,
  NTag: true,
  NBadge: true,
  NTooltip: true,
  NPopover: true,
  NDropdown: true,
  NMenu: true,
  NTabs: true,
  NTabPane: true,
  NProgress: true,
  NStatistic: true,
  NDescriptions: true,
  NDescriptionsItem: true,
  NAlert: true,
  NTree: true,
  NCollapse: true,
  NCollapseItem: true,
  NDivider: true,
  NBreadcrumb: true,
  NBreadcrumbItem: true,
  NPagination: true,
  NSwitch: true,
  NRadio: true,
  NRadioGroup: true,
  NDatePicker: true,
  NTimePicker: true,
  NUpload: true,
  NIcon: true,
  NAvatar: true,
  NGrid: true,
  NGi: true,
  NRow: true,
  NCol: true,
  NList: true,
  NListItem: true,
  NThing: true,
  NScrollbar: true,
  // 추가 커스텀 컴포넌트
  BaseTable: true,
  BaseDataGrid: true,
  BaseChart: true,
  BaseModal: true,
  BaseTooltip: true,
  BaseDropdown: true,
  BasePagination: true,
  BaseButton: true,
  BaseInput: true,
  BaseSelect: true,
  BaseForm: true,
  BaseFormItem: true,
  BaseLoading: true,
  BaseEmpty: true,
  BaseCard: true,
  BaseAlert: true,
  BaseBadge: true,
  BaseTag: true,
  BaseProgress: true,
  BaseStatistic: true,
  BaseTimeline: true,
  BaseTree: true,
  BaseCollapse: true,
  BaseDescription: true,
  BaseBreadcrumb: true,
  BaseMenu: true,
  BaseTabs: true,
  BaseDrawer: true,
  BasePopconfirm: true,
  BaseSteps: true,
  BaseCarousel: true,
  BaseRate: true,
  BaseSlider: true,
  BaseColorPicker: true,
  BaseCascader: true,
  BaseTransfer: true,
  BaseAutoComplete: true,
  BaseMention: true,
  BaseTreeSelect: true,
  BaseBackTop: true,
  BaseAffix: true,
  BaseAnchor: true,
  BaseSkeleton: true,
  BaseSpace: true,
  BaseTypography: true,
  BaseWatermark: true,
  BaseQrCode: true,
  BaseImagePreview: true,
  BaseVirtualList: true,
  BaseInfiniteScroll: true,
  BaseCountdown: true,
  BaseTextarea: true,
  BaseCode: true,
  BaseJson: true,
  BaseDiff: true,
  BaseMarkdown: true,
  BaseRichText: true,
  BaseCodeEditor: true,
  BaseJsonEditor: true,
  BaseSqlEditor: true,
  BaseTerminal: true,
  BaseFileTree: true,
  BaseFileViewer: true,
  BaseLogViewer: true,
  BasePerformance: true,
  BaseMonitor: true,
  BaseDebugger: true,
}

// NaiveUI useMessage 모킹
vi.mock('naive-ui', async () => {
  const actual = await vi.importActual('naive-ui')
  return {
    ...actual,
    useMessage: () => ({
      info: vi.fn(),
      success: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
      loading: vi.fn(),
      destroyAll: vi.fn(),
    }),
    useDialog: () => ({
      info: vi.fn(),
      success: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
      create: vi.fn(),
      destroyAll: vi.fn(),
    }),
    useNotification: () => ({
      info: vi.fn(),
      success: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
      create: vi.fn(),
      destroyAll: vi.fn(),
    }),
    useLoadingBar: () => ({
      start: vi.fn(),
      finish: vi.fn(),
      error: vi.fn(),
    }),
  }
})

beforeAll(() => {
  // 전역 설정 초기화
})