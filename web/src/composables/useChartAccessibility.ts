import { type Ref, computed, onBeforeUnmount, onMounted, ref } from 'vue'
import type { ChartData } from '@/types/ui'

interface ChartAccessibilityConfig {
  // 차트 기본 정보
  chartId: string
  title?: string
  description?: string

  // 데이터 테이블 제공 여부
  provideDataTable?: boolean
  showDataToggle?: boolean

  // 키보드 네비게이션
  enableKeyboardNavigation?: boolean

  // 스크린 리더 지원
  announceDataChanges?: boolean
  announceInteractions?: boolean

  // 색상 및 패턴
  highContrastMode?: boolean
  usePatterns?: boolean
}

export function useChartAccessibility(
  config: ChartAccessibilityConfig,
  chartRef: Ref<HTMLElement | undefined>,
  chartData: Ref<ChartData>,
) {
  // 상태 관리
  const showDataTable = ref(false)
  const currentFocus = ref<{ type: string; index: number } | null>(null)
  const liveRegion = ref<HTMLElement>()

  // 차트 속성 계산
  const chartAttributes = computed(() => ({
    'role': 'img',
    'aria-label': generateChartAriaLabel(),
    'aria-describedby': config.description ? `${config.chartId}-desc` : undefined,
    'tabindex': config.enableKeyboardNavigation ? '0' : undefined,
  }))

  // 차트 ARIA 라벨 생성
  const generateChartAriaLabel = (): string => {
    const baseLabel = config.title || '차트'
    const datasetCount = chartData.value.datasets?.length || 0
    const dataPointCount = chartData.value.labels?.length || 0

    return `${baseLabel}. ${datasetCount}개의 데이터셋과 ${dataPointCount}개의 데이터 포인트를 포함합니다.`
  }

  // 상세 차트 설명 생성
  const generateChartDescription = (): string => {
    if (!chartData.value.datasets || chartData.value.datasets.length === 0) {
      return '데이터가 없습니다.'
    }

    let description = ''

    // 데이터셋별 요약
    chartData.value.datasets.forEach((dataset, index) => {
      const label = dataset.label || `데이터셋 ${index + 1}`
      const dataCount = Array.isArray(dataset.data) ? dataset.data.length : 0

      if (Array.isArray(dataset.data) && dataset.data.length > 0) {
        // 숫자 데이터인 경우 통계 계산
        const numericData = dataset.data.filter(d => typeof d === 'number') as number[]

        if (numericData.length > 0) {
          const min = Math.min(...numericData)
          const max = Math.max(...numericData)
          const avg = numericData.reduce((a, b) => a + b, 0) / numericData.length

          description += `${label}: ${dataCount}개 데이터 포인트, 최솟값 ${min.toLocaleString()}, 최댓값 ${max.toLocaleString()}, 평균 ${avg.toLocaleString(undefined, { maximumFractionDigits: 2 })}. `
        }
      }
    })

    // 라벨 정보
    if (chartData.value.labels && chartData.value.labels.length > 0) {
      const firstLabel = chartData.value.labels[0]
      const lastLabel = chartData.value.labels[chartData.value.labels.length - 1]
      description += `X축은 ${firstLabel}부터 ${lastLabel}까지입니다.`
    }

    return description || '차트 데이터를 처리할 수 없습니다.'
  }

  // 데이터 테이블 생성
  const generateDataTable = () => {
    if (!chartData.value.datasets || chartData.value.datasets.length === 0) {
      return null
    }

    const headers = ['항목', ...chartData.value.datasets.map(d => d.label || '데이터')]
    const rows: string[][] = []

    // 라벨이 있는 경우
    if (chartData.value.labels) {
      chartData.value.labels.forEach((label, index) => {
        const row = [String(label)]

        chartData.value.datasets.forEach(dataset => {
          if (Array.isArray(dataset.data)) {
            const value = dataset.data[index]

            if (typeof value === 'number') {
              row.push(value.toLocaleString())
            } else if (typeof value === 'object' && value !== null) {
              // 객체 형태의 데이터 (예: {x, y})
              if ('y' in value) {
                row.push(String(value.y))
              } else if ('value' in value) {
                row.push(String(value.value))
              } else {
                row.push(JSON.stringify(value))
              }
            } else {
              row.push(String(value || ''))
            }
          }
        })

        rows.push(row)
      })
    } else {
      // 라벨이 없는 경우, 데이터셋의 첫 번째 것을 기준으로
      const maxLength = Math.max(...chartData.value.datasets.map(d =>
        Array.isArray(d.data) ? d.data.length : 0,
      ))

      for (let i = 0; i < maxLength; i++) {
        const row = [`항목 ${i + 1}`]

        chartData.value.datasets.forEach(dataset => {
          if (Array.isArray(dataset.data) && dataset.data[i] !== undefined) {
            const value = dataset.data[i]
            row.push(typeof value === 'number' ? value.toLocaleString() : String(value))
          } else {
            row.push('')
          }
        })

        rows.push(row)
      }
    }

    return { headers, rows }
  }

  // 키보드 네비게이션 핸들러
  const handleKeyDown = (event: KeyboardEvent) => {
    if (!config.enableKeyboardNavigation) return

    const { key, ctrlKey, altKey } = event
    let actionTaken = ''

    switch (key) {
      case 'Enter':
      case ' ':
        if (config.showDataToggle) {
          toggleDataTable()
          actionTaken = showDataTable.value ? '데이터 테이블 표시' : '데이터 테이블 숨김'
        }
        event.preventDefault()
        break

      case 'd':
      case 'D':
        if (altKey && config.provideDataTable) {
          showDataTable.value = true
          actionTaken = '데이터 테이블 표시'
          event.preventDefault()
        }
        break

      case 'Escape':
        if (showDataTable.value) {
          showDataTable.value = false
          actionTaken = '데이터 테이블 숨김'
          event.preventDefault()
        }
        break

      case 'i':
      case 'I':
        if (altKey) {
          announceChartInfo()
          actionTaken = '차트 정보 읽기'
          event.preventDefault()
        }
        break
    }

    if (actionTaken && config.announceInteractions) {
      announce(actionTaken)
    }
  }

  // 데이터 테이블 표시/숨김 토글
  const toggleDataTable = () => {
    showDataTable.value = !showDataTable.value

    if (config.announceInteractions) {
      announce(showDataTable.value ? '데이터 테이블이 표시되었습니다' : '데이터 테이블이 숨겨졌습니다')
    }
  }

  // 차트 정보 안내
  const announceChartInfo = () => {
    const description = generateChartDescription()
    announce(`차트 정보: ${description}`)
  }

  // 데이터 변경 안내
  const announceDataChange = () => {
    if (!config.announceDataChanges) return

    const datasetCount = chartData.value.datasets?.length || 0
    const message = `차트가 업데이트되었습니다. ${datasetCount}개의 데이터셋이 있습니다.`

    announce(message)
  }

  // 스크린 리더 안내
  const announce = (message: string) => {
    if (liveRegion.value) {
      liveRegion.value.textContent = message
    }
  }

  // 고대비 색상 팔레트
  const getHighContrastColors = () => [
    '#000000', '#FFFFFF', '#FF0000', '#00FF00',
    '#0000FF', '#FFFF00', '#FF00FF', '#00FFFF',
  ]

  // 패턴 정의 (SVG 패턴으로 활용 가능)
  const getPatternDefinitions = () => [
    'solid',
    'diagonal-lines',
    'dots',
    'vertical-lines',
    'horizontal-lines',
    'cross-hatch',
    'diamonds',
    'triangles',
  ]

  // 색상 접근성 개선
  const enhanceColorAccessibility = (originalColors: string[]) => {
    if (config.highContrastMode) {
      return getHighContrastColors()
    }

    // 색각 이상자를 위한 색상 조정
    return originalColors.map(color => {
      // 색상 대비 개선 로직 (실제로는 더 정교한 알고리즘 필요)
      return color
    })
  }

  // 차트 요소에 대한 설명 생성 (툴팁 등에서 사용)
  const getElementDescription = (elementType: string, dataIndex: number, datasetIndex?: number) => {
    const dataset = datasetIndex !== undefined ? chartData.value.datasets[datasetIndex] : null
    const label = chartData.value.labels?.[dataIndex] || `항목 ${dataIndex + 1}`

    if (dataset && Array.isArray(dataset.data)) {
      const value = dataset.data[dataIndex]
      const datasetLabel = dataset.label || `데이터셋 ${datasetIndex! + 1}`

      if (typeof value === 'number') {
        return `${datasetLabel}, ${label}: ${value.toLocaleString()}`
      } else if (typeof value === 'object' && value !== null) {
        if ('x' in value && 'y' in value) {
          return `${datasetLabel}, X: ${value.x}, Y: ${value.y}`
        }
      }
    }

    return `${elementType} ${dataIndex + 1}`
  }

  // 라이브 리전 생성
  const createLiveRegion = () => {
    const region = document.createElement('div')
    region.setAttribute('aria-live', 'polite')
    region.setAttribute('aria-atomic', 'true')
    region.className = 'sr-only'
    region.style.cssText = `
      position: absolute !important;
      width: 1px !important;
      height: 1px !important;
      padding: 0 !important;
      margin: -1px !important;
      overflow: hidden !important;
      clip: rect(0, 0, 0, 0) !important;
      white-space: nowrap !important;
      border: 0 !important;
    `

    document.body.appendChild(region)
    liveRegion.value = region
  }

  // 라이브 리전 제거
  const removeLiveRegion = () => {
    if (liveRegion.value) {
      document.body.removeChild(liveRegion.value)
      liveRegion.value = undefined
    }
  }

  // 이벤트 리스너 설정
  const setupEventListeners = () => {
    if (chartRef.value && config.enableKeyboardNavigation) {
      chartRef.value.addEventListener('keydown', handleKeyDown)
    }
  }

  // 이벤트 리스너 제거
  const removeEventListeners = () => {
    if (chartRef.value) {
      chartRef.value.removeEventListener('keydown', handleKeyDown)
    }
  }

  // 초기 포커스 설정
  const setInitialFocus = () => {
    if (chartRef.value && config.enableKeyboardNavigation) {
      chartRef.value.focus()
    }
  }

  // 생명주기 관리
  onMounted(() => {
    createLiveRegion()
    setupEventListeners()

    // 초기 안내
    if (config.announceDataChanges) {
      setTimeout(() => {
        announce(generateChartAriaLabel())
      }, 100)
    }
  })

  onBeforeUnmount(() => {
    removeEventListeners()
    removeLiveRegion()
  })

  return {
    // 상태
    showDataTable,
    currentFocus,

    // 속성
    chartAttributes,

    // 메서드
    generateChartDescription,
    generateDataTable,
    toggleDataTable,
    announceChartInfo,
    announceDataChange,
    announce,
    getElementDescription,
    enhanceColorAccessibility,
    getHighContrastColors,
    getPatternDefinitions,
    setInitialFocus,
  }
}