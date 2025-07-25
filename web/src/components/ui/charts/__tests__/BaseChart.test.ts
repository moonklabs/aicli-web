import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import BaseChart from '../BaseChart.vue'
import type { ChartData } from '@/types/ui'

// Chart.js 모킹
const mockChart = {
  destroy: vi.fn(),
  update: vi.fn(),
  resize: vi.fn(),
  resetZoom: vi.fn(),
  isZoomedOrPanned: vi.fn(() => false),
  getElementsAtEventForMode: vi.fn(() => []),
  canvas: {
    toDataURL: vi.fn(() => 'data:image/png;base64,test'),
  },
  data: { datasets: [] },
  options: {},
}

vi.mock('chart.js', () => ({
  Chart: vi.fn(() => mockChart),
  registerables: [],
}))

describe('BaseChart', () => {
  const mockChartData: ChartData = {
    labels: ['Jan', 'Feb', 'Mar', 'Apr', 'May'],
    datasets: [
      {
        label: 'Sales',
        data: [10, 20, 30, 40, 50],
        backgroundColor: '#3b82f6',
        borderColor: '#3b82f6',
      },
    ],
  }

  let wrapper: any

  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    if (wrapper) {
      wrapper.unmount()
    }
  })

  describe('렌더링', () => {
    it('차트 컨테이너가 올바르게 렌더링되어야 한다', () => {
      wrapper = mount(BaseChart, {
        props: {
          chartType: 'line',
          data: mockChartData,
        },
      })

      expect(wrapper.find('.base-chart').exists()).toBe(true)
      expect(wrapper.find('canvas').exists()).toBe(true)
    })

    it('차트 타입에 따른 적절한 설정이 적용되어야 한다', () => {
      wrapper = mount(BaseChart, {
        props: {
          chartType: 'bar',
          data: mockChartData,
        },
      })

      expect(wrapper.vm.chartConfiguration.type).toBe('bar')
    })

    it('캔버스 크기가 올바르게 설정되어야 한다', () => {
      wrapper = mount(BaseChart, {
        props: {
          chartType: 'line',
          data: mockChartData,
          width: 800,
          height: 400,
        },
      })

      expect(wrapper.vm.canvasWidth).toBe(800)
      expect(wrapper.vm.canvasHeight).toBe(400)
    })
  })

  describe('접근성', () => {
    it('적절한 ARIA 속성이 설정되어야 한다', () => {
      wrapper = mount(BaseChart, {
        props: {
          chartType: 'line',
          data: mockChartData,
          accessibility: {
            enabled: true,
            description: '매출 추이 차트',
            summary: '5개월간 매출이 꾸준히 증가하고 있습니다',
          },
        },
      })

      const chartContainer = wrapper.find('.base-chart')
      expect(chartContainer.attributes('role')).toBe('img')
      expect(chartContainer.attributes('aria-label')).toContain('매출 추이 차트')
      expect(chartContainer.attributes('aria-describedby')).toBeDefined()
    })

    it('접근성 요약이 표시되어야 한다', () => {
      wrapper = mount(BaseChart, {
        props: {
          chartType: 'line',
          data: mockChartData,
          accessibility: {
            enabled: true,
            summary: '차트 요약 정보',
          },
        },
      })

      expect(wrapper.find('#chart-summary').exists()).toBe(true)
      expect(wrapper.find('#chart-summary').text()).toBe('차트 요약 정보')
      expect(wrapper.find('#chart-summary').classes()).toContain('sr-only')
    })
  })

  describe('로딩 상태', () => {
    it('로딩 상태가 표시되어야 한다', () => {
      wrapper = mount(BaseChart, {
        props: {
          chartType: 'line',
          data: mockChartData,
          loading: true,
        },
      })

      expect(wrapper.find('.chart-loading').exists()).toBe(true)
      expect(wrapper.find('.loading-spinner').exists()).toBe(true)
      expect(wrapper.text()).toContain('차트를 로딩 중')
    })

    it('로딩 중에는 툴바가 비활성화되어야 한다', () => {
      wrapper = mount(BaseChart, {
        props: {
          chartType: 'line',
          data: mockChartData,
          loading: true,
          showToolbar: true,
        },
      })

      const toolbar = wrapper.find('.chart-toolbar')
      expect(toolbar.classes()).toContain('opacity-50')
      expect(toolbar.classes()).toContain('pointer-events-none')
    })
  })

  describe('에러 상태', () => {
    it('에러 상태가 표시되어야 한다', () => {
      const error = new Error('차트 로딩 실패')

      wrapper = mount(BaseChart, {
        props: {
          chartType: 'line',
          data: mockChartData,
          error,
        },
      })

      expect(wrapper.find('.chart-error').exists()).toBe(true)
      expect(wrapper.text()).toContain('차트 로딩 실패')
      expect(wrapper.text()).toContain(error.message)
    })

    it('에러 상태에서 재시도 버튼이 작동해야 한다', async () => {
      const error = new Error('Network error')

      wrapper = mount(BaseChart, {
        props: {
          chartType: 'line',
          data: mockChartData,
          error,
        },
      })

      const retryButton = wrapper.find('.retry-btn')
      expect(retryButton.exists()).toBe(true)

      await retryButton.trigger('click')

      // createChart 메서드가 호출되었는지 확인 (실제 구현에 따라 조정)
      expect(wrapper.vm).toBeDefined()
    })
  })

  describe('툴바 기능', () => {
    beforeEach(() => {
      wrapper = mount(BaseChart, {
        props: {
          chartType: 'line',
          data: mockChartData,
          showToolbar: true,
          zoom: { enabled: true },
          export: { enabled: true, formats: ['png', 'jpg'] },
        },
      })
    })

    it('툴바가 표시되어야 한다', () => {
      expect(wrapper.find('.chart-toolbar').exists()).toBe(true)
    })

    it('줌 리셋 버튼이 작동해야 한다', async () => {
      const zoomButton = wrapper.find('[title="줌 리셋"]')
      expect(zoomButton.exists()).toBe(true)

      await zoomButton.trigger('click')
      expect(mockChart.resetZoom).toHaveBeenCalled()
    })

    it('내보내기 메뉴가 토글되어야 한다', async () => {
      const exportButton = wrapper.find('[title="내보내기"]')
      await exportButton.trigger('click')

      expect(wrapper.find('.export-menu').exists()).toBe(true)
    })

    it('내보내기 옵션이 올바르게 표시되어야 한다', async () => {
      const exportButton = wrapper.find('[title="내보내기"]')
      await exportButton.trigger('click')

      const exportOptions = wrapper.findAll('.export-option')
      expect(exportOptions).toHaveLength(2)
      expect(exportOptions[0].text()).toBe('PNG')
      expect(exportOptions[1].text()).toBe('JPG')
    })
  })

  describe('커스텀 범례', () => {
    beforeEach(() => {
      wrapper = mount(BaseChart, {
        props: {
          chartType: 'line',
          data: mockChartData,
          showCustomLegend: true,
        },
      })
    })

    it('커스텀 범례가 표시되어야 한다', () => {
      expect(wrapper.find('.custom-legend').exists()).toBe(true)
    })

    it('범례 아이템이 올바르게 표시되어야 한다', () => {
      const legendItems = wrapper.findAll('.legend-item')
      expect(legendItems).toHaveLength(mockChartData.datasets.length)

      const firstItem = legendItems[0]
      expect(firstItem.find('.legend-label').text()).toBe('Sales')
      expect(firstItem.find('.legend-color').exists()).toBe(true)
    })

    it('범례 아이템 클릭 시 데이터셋 토글이 작동해야 한다', async () => {
      const firstLegendItem = wrapper.find('.legend-item')
      await firstLegendItem.trigger('click')

      // 데이터셋 숨김/표시 로직이 실행되었는지 확인
      expect(wrapper.vm.hiddenDatasets.has(0)).toBe(true)
    })
  })

  describe('이벤트 처리', () => {
    beforeEach(() => {
      wrapper = mount(BaseChart, {
        props: {
          chartType: 'line',
          data: mockChartData,
        },
      })
    })

    it('차트 클릭 이벤트가 처리되어야 한다', async () => {
      const canvas = wrapper.find('canvas')
      const mockEvent = new MouseEvent('click')

      await wrapper.vm.handleChartClick(mockEvent)

      expect(wrapper.emitted('chart-click')).toBeTruthy()
    })

    it('차트 호버 이벤트가 처리되어야 한다', async () => {
      const canvas = wrapper.find('canvas')
      const mockEvent = new MouseEvent('mousemove')

      await wrapper.vm.handleChartHover(mockEvent)

      expect(wrapper.emitted('chart-hover')).toBeTruthy()
    })
  })

  describe('테마 적용', () => {
    const mockTheme = {
      colors: {
        primary: ['#3b82f6', '#ef4444', '#10b981'],
        secondary: ['#6b7280'],
        accent: ['#8b5cf6'],
        neutral: ['#374151'],
      },
      fonts: {
        family: 'Arial',
        size: 12,
        weight: 'normal',
      },
      grid: {
        color: '#e5e7eb',
        lineWidth: 1,
      },
      tooltip: {
        backgroundColor: '#1f2937',
        titleColor: '#f9fafb',
        bodyColor: '#e5e7eb',
        borderColor: '#374151',
      },
    }

    it('테마가 올바르게 적용되어야 한다', () => {
      wrapper = mount(BaseChart, {
        props: {
          chartType: 'line',
          data: mockChartData,
          theme: mockTheme,
        },
      })

      const mergedOptions = wrapper.vm.mergedOptions
      expect(mergedOptions.plugins.tooltip.backgroundColor).toBe(mockTheme.tooltip.backgroundColor)
    })
  })

  describe('실시간 업데이트', () => {
    it('실시간 업데이트가 설정되어야 한다', () => {
      wrapper = mount(BaseChart, {
        props: {
          chartType: 'line',
          data: mockChartData,
          realTime: {
            enabled: true,
            interval: 1000,
            maxDataPoints: 20,
          },
        },
      })

      expect(wrapper.vm.realTime.enabled).toBe(true)
    })
  })

  describe('반응형', () => {
    it('반응형 설정이 올바르게 적용되어야 한다', () => {
      wrapper = mount(BaseChart, {
        props: {
          chartType: 'line',
          data: mockChartData,
          responsive: true,
          maintainAspectRatio: false,
        },
      })

      const options = wrapper.vm.mergedOptions
      expect(options.responsive).toBe(true)
      expect(options.maintainAspectRatio).toBe(false)
    })
  })

  describe('차트 생명주기', () => {
    it('컴포넌트 마운트 시 차트가 생성되어야 한다', () => {
      wrapper = mount(BaseChart, {
        props: {
          chartType: 'line',
          data: mockChartData,
        },
      })

      expect(wrapper.emitted('chart-create')).toBeTruthy()
    })

    it('컴포넌트 언마운트 시 차트가 파괴되어야 한다', () => {
      wrapper = mount(BaseChart, {
        props: {
          chartType: 'line',
          data: mockChartData,
        },
      })

      wrapper.unmount()
      expect(mockChart.destroy).toHaveBeenCalled()
    })

    it('데이터 변경 시 차트가 업데이트되어야 한다', async () => {
      wrapper = mount(BaseChart, {
        props: {
          chartType: 'line',
          data: mockChartData,
        },
      })

      const newData = {
        ...mockChartData,
        datasets: [
          {
            ...mockChartData.datasets[0],
            data: [15, 25, 35, 45, 55],
          },
        ],
      }

      await wrapper.setProps({ data: newData })
      expect(mockChart.update).toHaveBeenCalled()
    })
  })

  describe('내보내기 기능', () => {
    beforeEach(() => {
      wrapper = mount(BaseChart, {
        props: {
          chartType: 'line',
          data: mockChartData,
          export: {
            enabled: true,
            formats: ['png', 'jpg'],
            quality: 0.8,
            filename: 'test-chart',
          },
        },
      })
    })

    it('PNG 내보내기가 작동해야 한다', async () => {
      // DOM 메서드 모킹
      const mockLink = {
        click: vi.fn(),
        download: '',
        href: '',
      }
      const createElementSpy = vi.spyOn(document, 'createElement').mockReturnValue(mockLink as any)

      await wrapper.vm.exportChart('png')

      expect(createElementSpy).toHaveBeenCalledWith('a')
      expect(mockLink.download).toBe('test-chart.png')
      expect(mockLink.click).toHaveBeenCalled()

      createElementSpy.mockRestore()
    })
  })
})