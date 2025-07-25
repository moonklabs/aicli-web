import { type Ref, computed, onBeforeUnmount, onMounted, ref } from 'vue'

interface AccessibilityConfig {
  // 테이블 기본 정보
  tableId: string
  caption?: string
  summary?: string

  // 키보드 네비게이션 설정
  enableKeyboardNavigation?: boolean
  enableCellNavigation?: boolean
  enableRowSelection?: boolean

  // 스크린 리더 설정
  announceChanges?: boolean
  announceSelection?: boolean
  announceSort?: boolean
  announceFilter?: boolean

  // ARIA 설정
  ariaLabelledBy?: string
  ariaDescribedBy?: string
}

interface CellPosition {
  row: number
  col: number
}

export function useTableAccessibility(
  config: AccessibilityConfig,
  tableRef: Ref<HTMLElement | undefined>,
  data: Ref<any[]>,
  columns: Ref<any[]>,
) {
  // 현재 포커스된 셀 위치
  const focusedCell = ref<CellPosition>({ row: -1, col: -1 })
  const selectedRows = ref<Set<number>>(new Set())

  // ARIA live region for announcements
  const liveRegion = ref<HTMLElement>()
  const announcements = ref<string[]>([])

  // 키보드 이벤트 처리 상태
  const isNavigationActive = ref(false)

  // 접근성 속성 계산
  const tableAttributes = computed(() => ({
    'role': 'table',
    'aria-label': config.caption || '데이터 테이블',
    'aria-describedby': config.ariaDescribedBy,
    'aria-labelledby': config.ariaLabelledBy,
    'aria-rowcount': data.value.length + 1, // +1 for header
    'aria-colcount': columns.value.length,
  }))

  const headerAttributes = computed(() => ({
    'role': 'rowgroup',
  }))

  const bodyAttributes = computed(() => ({
    'role': 'rowgroup',
  }))

  // 행 속성 생성
  const getRowAttributes = (rowIndex: number, isHeader = false) => {
    const baseAttrs = {
      'role': 'row',
      'aria-rowindex': rowIndex + (isHeader ? 1 : 2), // +1 for 1-based, +1 for header
    }

    if (!isHeader) {
      return {
        ...baseAttrs,
        'aria-selected': selectedRows.value.has(rowIndex) ? 'true' : 'false',
        'tabindex': focusedCell.value.row === rowIndex ? '0' : '-1',
      }
    }

    return baseAttrs
  }

  // 셀 속성 생성
  const getCellAttributes = (rowIndex: number, colIndex: number, isHeader = false) => {
    const column = columns.value[colIndex]
    const baseAttrs = {
      'role': isHeader ? 'columnheader' : 'cell',
      'aria-colindex': colIndex + 1,
      'tabindex': focusedCell.value.row === rowIndex && focusedCell.value.col === colIndex ? '0' : '-1',
    }

    if (isHeader) {
      return {
        ...baseAttrs,
        'aria-sort': getSortAttribute(column),
        'aria-label': `${column.title} 컬럼${column.sortable ? ', 정렬 가능' : ''}`,
        'scope': 'col',
      }
    }

    return {
      ...baseAttrs,
      'aria-describedby': column.key ? `col-${column.key}-desc` : undefined,
    }
  }

  // 정렬 상태를 ARIA 속성으로 변환
  const getSortAttribute = (column: any): string => {
    if (!column.sortable) return 'none'

    // 정렬 상태에 따라 반환 (실제 정렬 상태는 부모 컴포넌트에서 관리)
    if (column.currentSort === 'asc') return 'ascending'
    if (column.currentSort === 'desc') return 'descending'
    return 'none'
  }

  // 키보드 네비게이션 핸들러
  const handleKeyDown = (event: KeyboardEvent) => {
    if (!config.enableKeyboardNavigation || !tableRef.value) return

    const { key, ctrlKey, shiftKey, metaKey } = event
    const maxRow = data.value.length - 1
    const maxCol = columns.value.length - 1

    let newPosition = { ...focusedCell.value }
    let shouldPreventDefault = true
    let actionTaken = ''

    switch (key) {
      case 'ArrowDown':
        if (newPosition.row < maxRow) {
          newPosition.row++
          actionTaken = `행 ${newPosition.row + 1}로 이동`
        }
        break

      case 'ArrowUp':
        if (newPosition.row > 0) {
          newPosition.row--
          actionTaken = `행 ${newPosition.row + 1}로 이동`
        }
        break

      case 'ArrowRight':
        if (newPosition.col < maxCol) {
          newPosition.col++
          actionTaken = `${columns.value[newPosition.col].title} 컬럼으로 이동`
        }
        break

      case 'ArrowLeft':
        if (newPosition.col > 0) {
          newPosition.col--
          actionTaken = `${columns.value[newPosition.col].title} 컬럼으로 이동`
        }
        break

      case 'Home':
        if (ctrlKey || metaKey) {
          newPosition = { row: 0, col: 0 }
          actionTaken = '테이블 시작으로 이동'
        } else {
          newPosition.col = 0
          actionTaken = '행의 시작으로 이동'
        }
        break

      case 'End':
        if (ctrlKey || metaKey) {
          newPosition = { row: maxRow, col: maxCol }
          actionTaken = '테이블 끝으로 이동'
        } else {
          newPosition.col = maxCol
          actionTaken = '행의 끝으로 이동'
        }
        break

      case 'PageDown':
        newPosition.row = Math.min(maxRow, newPosition.row + 10)
        actionTaken = `행 ${newPosition.row + 1}로 이동`
        break

      case 'PageUp':
        newPosition.row = Math.max(0, newPosition.row - 10)
        actionTaken = `행 ${newPosition.row + 1}로 이동`
        break

      case ' ':
      case 'Enter':
        if (config.enableRowSelection) {
          toggleRowSelection(newPosition.row)
          actionTaken = `행 ${newPosition.row + 1} ${selectedRows.value.has(newPosition.row) ? '선택됨' : '선택 해제됨'}`
        }
        break

      case 'Escape':
        clearSelection()
        actionTaken = '선택 해제됨'
        break

      default:
        shouldPreventDefault = false
    }

    if (shouldPreventDefault) {
      event.preventDefault()

      // 포커스 이동
      if (newPosition.row !== focusedCell.value.row || newPosition.col !== focusedCell.value.col) {
        focusCell(newPosition.row, newPosition.col)
      }

      // 스크린 리더 안내
      if (actionTaken && config.announceChanges) {
        announce(actionTaken)
      }
    }
  }

  // 셀에 포커스 설정
  const focusCell = (row: number, col: number) => {
    if (!tableRef.value) return

    const cell = tableRef.value.querySelector(
      `[data-row="${row}"][data-col="${col}"]`,
    ) as HTMLElement

    if (cell) {
      // 이전 셀의 tabindex 제거
      const prevCell = tableRef.value.querySelector('[tabindex="0"]')
      if (prevCell) {
        prevCell.setAttribute('tabindex', '-1')
      }

      // 새 셀에 포커스 설정
      cell.setAttribute('tabindex', '0')
      cell.focus()

      // 포커스 상태 업데이트
      focusedCell.value = { row, col }

      // 셀이 보이도록 스크롤
      scrollCellIntoView(cell)
    }
  }

  // 셀을 화면에 보이도록 스크롤
  const scrollCellIntoView = (cell: HTMLElement) => {
    cell.scrollIntoView({
      behavior: 'smooth',
      block: 'nearest',
      inline: 'nearest',
    })
  }

  // 행 선택 토글
  const toggleRowSelection = (rowIndex: number) => {
    if (selectedRows.value.has(rowIndex)) {
      selectedRows.value.delete(rowIndex)
    } else {
      selectedRows.value.add(rowIndex)
    }

    if (config.announceSelection) {
      const isSelected = selectedRows.value.has(rowIndex)
      announce(`행 ${rowIndex + 1} ${isSelected ? '선택됨' : '선택 해제됨'}`)
    }
  }

  // 모든 선택 해제
  const clearSelection = () => {
    selectedRows.value.clear()

    if (config.announceSelection) {
      announce('모든 선택이 해제되었습니다')
    }
  }

  // 스크린 리더 안내
  const announce = (message: string) => {
    if (!config.announceChanges) return

    announcements.value.push(message)

    // 중복 메시지 제거
    if (announcements.value.length > 5) {
      announcements.value.shift()
    }

    // ARIA live region 업데이트
    if (liveRegion.value) {
      liveRegion.value.textContent = message
    }
  }

  // 정렬 변경 안내
  const announceSortChange = (columnName: string, direction: 'asc' | 'desc' | null) => {
    if (!config.announceSort) return

    let message = ''
    if (direction === 'asc') {
      message = `${columnName} 컬럼이 오름차순으로 정렬되었습니다`
    } else if (direction === 'desc') {
      message = `${columnName} 컬럼이 내림차순으로 정렬되었습니다`
    } else {
      message = `${columnName} 컬럼의 정렬이 제거되었습니다`
    }

    announce(message)
  }

  // 필터 변경 안내
  const announceFilterChange = (columnName: string, hasFilter: boolean) => {
    if (!config.announceFilter) return

    const message = hasFilter
      ? `${columnName} 컬럼에 필터가 적용되었습니다`
      : `${columnName} 컬럼의 필터가 제거되었습니다`

    announce(message)
  }

  // 데이터 변경 안내
  const announceDataChange = (newRowCount: number) => {
    if (!config.announceChanges) return

    announce(`테이블이 업데이트되었습니다. 총 ${newRowCount}개의 행이 있습니다`)
  }

  // 테이블 요약 정보 생성
  const getTableSummary = () => {
    return config.summary ||
      `${columns.value.length}개 컬럼, ${data.value.length}개 행으로 구성된 데이터 테이블`
  }

  // 컬럼 설명 생성
  const getColumnDescription = (column: any) => {
    let description = `${column.title} 컬럼`

    if (column.sortable) {
      description += ', 정렬 가능'
    }

    if (column.filterable) {
      description += ', 필터 가능'
    }

    if (column.type) {
      description += `, ${column.type} 타입`
    }

    return description
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

  // 이벤트 리스너 등록
  const setupEventListeners = () => {
    if (tableRef.value && config.enableKeyboardNavigation) {
      tableRef.value.addEventListener('keydown', handleKeyDown)

      // 포커스 관리
      const cells = tableRef.value.querySelectorAll('[role="cell"], [role="columnheader"]')
      cells.forEach((cell, index) => {
        cell.addEventListener('focus', () => {
          const row = parseInt(cell.getAttribute('data-row') || '0')
          const col = parseInt(cell.getAttribute('data-col') || '0')
          focusedCell.value = { row, col }
        })
      })
    }
  }

  // 이벤트 리스너 제거
  const removeEventListeners = () => {
    if (tableRef.value) {
      tableRef.value.removeEventListener('keydown', handleKeyDown)
    }
  }

  // 초기 포커스 설정
  const setInitialFocus = () => {
    if (data.value.length > 0 && columns.value.length > 0) {
      focusCell(0, 0)
    }
  }

  // 생명주기 관리
  onMounted(() => {
    createLiveRegion()
    setupEventListeners()

    // 초기 안내
    if (config.announceChanges) {
      announce(getTableSummary())
    }
  })

  onBeforeUnmount(() => {
    removeEventListeners()
    removeLiveRegion()
  })

  return {
    // 상태
    focusedCell,
    selectedRows,
    announcements,

    // 속성 생성기
    tableAttributes,
    headerAttributes,
    bodyAttributes,
    getRowAttributes,
    getCellAttributes,

    // 메서드
    focusCell,
    toggleRowSelection,
    clearSelection,
    announce,
    announceSortChange,
    announceFilterChange,
    announceDataChange,
    getTableSummary,
    getColumnDescription,
    setInitialFocus,

    // 유틸리티
    scrollCellIntoView,
  }
}