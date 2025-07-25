<template>
  <div
    :class="[
      'base-data-table',
      `size-${size}`,
      {
        'striped': striped,
        'bordered': bordered,
        'single-line': singleLine,
        'loading': loading
      }
    ]"
    role="table"
    :aria-label="ariaLabel"
    :aria-describedby="ariaDescribedby"
    :aria-busy="loading"
    :aria-rowcount="filteredAndSortedData.length"
  >
    <!-- 테이블 헤더 -->
    <div class="table-header">
      <!-- 글로벌 필터 -->
      <div v-if="globalSearch" class="global-search">
        <input
          v-model="globalSearchValue"
          type="text"
          :placeholder="globalSearchPlaceholder"
          class="search-input"
          @input="(e) => { globalSearchValue = e.target.value; handleGlobalSearch(e.target.value) }"
        />
      </div>

      <!-- 컬럼 설정 및 필터 버튼 -->
      <div class="table-actions">
        <div v-if="columnSettings" class="column-settings">
          <button @click="showColumnSettings = !showColumnSettings" class="settings-btn">
            컬럼 설정
          </button>
        </div>
        
        <div v-if="hasFilters" class="filter-actions">
          <span class="filter-count">필터: {{ activeFilters.length }}개</span>
          <button @click="clearAllTableFilters" class="clear-filters-btn">
            모든 필터 지우기
          </button>
        </div>
      </div>
    </div>

    <!-- 가상 스크롤 컨테이너 -->
    <div
      ref="scrollContainer"
      class="table-container"
      :style="{ height: scrollY ? `${scrollY}px` : 'auto' }"
      @scroll="handleScroll"
    >
      <!-- 테이블 헤더 -->
      <table 
        ref="tableElement"
        class="data-table" 
        :style="{ minWidth: scrollX ? `${scrollX}px` : 'auto' }"
        @keydown="handleTableKeydown"
      >
        <thead class="table-head" :class="{ sticky: stickyHeader }">
          <tr>
            <!-- 선택 체크박스 컬럼 -->
            <th v-if="selection?.type === 'checkbox'" class="selection-col">
              <input
                type="checkbox"
                :checked="isAllSelected"
                :indeterminate="isPartiallySelected"
                @change="handleSelectAll"
                :aria-label="'모든 행 선택'"
              />
            </th>

            <!-- 데이터 컬럼들 -->
            <th
              v-for="(column, index) in visibleColumns"
              :key="column.key"
              :class="[
                'table-header-cell',
                `align-${column.align || 'left'}`,
                {
                  'sortable': column.sortable,
                  'sorted': getSortDirection(column.key),
                  'resizable': column.resizable,
                  'fixed-left': column.fixed === 'left',
                  'fixed-right': column.fixed === 'right'
                }
              ]"
              :style="getColumnStyle(column)"
              @click="handleHeaderClick(column)"
              :tabindex="column.sortable ? 0 : -1"
              @keydown="(e) => handleHeaderKeydown(e, column)"
              :aria-sort="getAriaSortValue(column.key)"
            >
              <!-- 커스텀 헤더 렌더링 -->
              <component
                v-if="column.renderHeader"
                :is="column.renderHeader()"
              />
              <span v-else>{{ column.title }}</span>

              <!-- 정렬 인디케이터 -->
              <span
                v-if="column.sortable"
                class="sort-indicator"
                :class="getSortDirection(column.key)"
              >
                ↑↓
              </span>

              <!-- 리사이즈 핸들 -->
              <div
                v-if="column.resizable"
                class="resize-handle"
                @mousedown="(e) => handleResizeStart(e, index)"
              ></div>
            </th>
          </tr>

          <!-- 필터 행 -->
          <tr v-if="showFilters" class="filter-row">
            <th v-if="selection?.type === 'checkbox'" class="selection-col"></th>
            <th
              v-for="column in visibleColumns"
              :key="`filter-${column.key}`"
              class="filter-cell"
            >
              <BaseFilter
                v-if="column.filter"
                :column="column"
                :type="column.filter.type"
                :value="getFilterValue(column.key)"
                :placeholder="column.filter.placeholder"
                @filter-change="handleFilterChange"
              />
            </th>
          </tr>
        </thead>
      </table>

      <!-- 가상 스크롤 테이블 바디 -->
      <div
        v-if="virtualScroll?.enabled"
        ref="virtualContainer"
        class="virtual-container"
        :style="{ height: `${virtualTotalSize}px` }"
      >
        <table class="data-table" :style="{ minWidth: scrollX ? `${scrollX}px` : 'auto' }">
          <tbody ref="virtualBody" class="table-body">
            <tr
              v-for="item in visibleItems"
              :key="getRowKey(item, item.virtualItem?.index || 0)"
              :class="[
                'table-row',
                getRowClassName(item, item.virtualItem?.index || 0),
                {
                  'selected': isRowSelected(item),
                  'hover': hoveredRowIndex === (item.virtualItem?.index || 0)
                }
              ]"
              @click="(e) => handleRowClick(item, item.virtualItem?.index || 0, e)"
              @dblclick="(e) => handleRowDoubleClick(item, item.virtualItem?.index || 0, e)"
              @mouseenter="hoveredRowIndex = item.virtualItem?.index || 0"
              @mouseleave="hoveredRowIndex = -1"
              :style="{ 
                transform: `translateY(${item.virtualItem?.start || 0}px)`,
                height: `${item.virtualItem?.size || itemHeight}px`
              }"
              :data-index="item.virtualItem?.index"
            >
              <!-- 선택 체크박스/라디오 -->
              <td v-if="selection?.type" class="selection-col">
                <input
                  :type="selection.type"
                  :name="selection.type === 'radio' ? 'row-selection' : undefined"
                  :checked="isRowSelected(item)"
                  @change="(e) => handleRowSelection(item, e.target.checked)"
                  :aria-label="`행 ${rowIndex + 1} 선택`"
                />
              </td>

              <!-- 데이터 셀들 -->
              <td
                v-for="column in visibleColumns"
                :key="`${getRowKey(item, item.virtualItem?.index || 0)}-${column.key}`"
                :class="[
                  'table-cell',
                  `align-${column.align || 'left'}`,
                  {
                    'ellipsis': column.ellipsis,
                    'fixed-left': column.fixed === 'left',
                    'fixed-right': column.fixed === 'right'
                  }
                ]"
                :style="getColumnStyle(column)"
                @click="(e) => handleCellClick(getCellValue(item, column), item, column, e)"
                :title="column.ellipsis ? getCellDisplayValue(item, column) : undefined"
              >
                <!-- 커스텀 셀 렌더링 -->
                <component
                  v-if="column.render"
                  :is="column.render(item, item.virtualItem?.index || 0)"
                />
                <span v-else>{{ getCellDisplayValue(item, column) }}</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 일반 테이블 바디 (가상 스크롤 비활성화) -->
      <table v-else class="data-table" :style="{ minWidth: scrollX ? `${scrollX}px` : 'auto' }">
        <tbody class="table-body">
          <tr
            v-for="(item, rowIndex) in filteredAndSortedData"
            :key="getRowKey(item, rowIndex)"
            :class="[
              'table-row',
              getRowClassName(item, rowIndex),
              {
                'selected': isRowSelected(item),
                'hover': hoveredRowIndex === rowIndex
              }
            ]"
            @click="(e) => handleRowClick(item, rowIndex, e)"
            @dblclick="(e) => handleRowDoubleClick(item, rowIndex, e)"
            @mouseenter="hoveredRowIndex = rowIndex"
            @mouseleave="hoveredRowIndex = -1"
          >
            <!-- 선택 체크박스/라디오 -->
            <td v-if="selection?.type" class="selection-col">
              <input
                :type="selection.type"
                :name="selection.type === 'radio' ? 'row-selection' : undefined"
                :checked="isRowSelected(item)"
                @change="(e) => handleRowSelection(item, e.target.checked)"
                :aria-label="`행 ${rowIndex + 1} 선택`"
              />
            </td>

            <!-- 데이터 셀들 -->
            <td
              v-for="column in visibleColumns"
              :key="`${getRowKey(item, rowIndex)}-${column.key}`"
              :class="[
                'table-cell',
                `align-${column.align || 'left'}`,
                {
                  'ellipsis': column.ellipsis,
                  'fixed-left': column.fixed === 'left',
                  'fixed-right': column.fixed === 'right'
                }
              ]"
              :style="getColumnStyle(column)"
              @click="(e) => handleCellClick(getCellValue(item, column), item, column, e)"
              :title="column.ellipsis ? getCellDisplayValue(item, column) : undefined"
            >
              <!-- 커스텀 셀 렌더링 -->
              <component
                v-if="column.render"
                :is="column.render(item, rowIndex)"
              />
              <span v-else>{{ getCellDisplayValue(item, column) }}</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 로딩 오버레이 -->
    <div v-if="loading" class="loading-overlay">
      <div class="loading-spinner">로딩 중...</div>
    </div>

    <!-- 빈 상태 -->
    <div v-if="!loading && filteredAndSortedData.length === 0" class="empty-state">
      <slot name="empty">
        <div class="empty-content">
          <p>데이터가 없습니다</p>
        </div>
      </slot>
    </div>

    <!-- 페이지네이션 -->
    <div v-if="pagination" class="table-pagination">
      <slot name="pagination" :pagination="paginationState">
        <div class="pagination-info">
          총 {{ paginationState.total }}개 중 {{ paginationState.start }}-{{ paginationState.end }}
        </div>
        <div class="pagination-controls">
          <button
            :disabled="paginationState.page <= 1"
            @click="goToPage(paginationState.page - 1)"
          >
            이전
          </button>
          <span>{{ paginationState.page }} / {{ paginationState.totalPages }}</span>
          <button
            :disabled="paginationState.page >= paginationState.totalPages"
            @click="goToPage(paginationState.page + 1)"
          >
            다음
          </button>
        </div>
      </slot>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useVirtualizer } from '@tanstack/vue-virtual'
import type {
  AdvancedDataTableProps,
  AdvancedTableColumn,
  TableFilter,
  TablePagination,
  TableSorter,
  VirtualScrollConfig,
} from '@/types/ui'
import { useTableAccessibility } from '@/composables/useTableAccessibility'
import { usePerformanceOptimization } from '@/composables/usePerformanceOptimization'
import { useTouchGestures } from '@/composables/useTouchGestures'

// Props 정의
interface Props extends AdvancedDataTableProps {
  globalSearch?: boolean
  globalSearchPlaceholder?: string
  columnSettings?: boolean
  stickyHeader?: boolean
  showFilters?: boolean
  ariaLabel?: string
  ariaDescribedby?: string
}

const props = withDefaults(defineProps<Props>(), {
  data: () => [],
  columns: () => [],
  size: 'medium',
  loading: false,
  pagination: false,
  pageSize: 20,
  striped: false,
  bordered: false,
  singleLine: false,
  globalSearch: false,
  globalSearchPlaceholder: '검색...',
  columnSettings: false,
  stickyHeader: true,
  showFilters: false,
  virtualScroll: () => ({ enabled: false, itemHeight: 40, overscan: 5 }),
})

// Emits 정의
const emit = defineEmits<{
  'update:checkedRowKeys': [keys: Array<string | number>]
  'update:page': [page: number]
  'update:pageSize': [pageSize: number]
  'update:sorter': [sorter: TableSorter | null]
  'update:filters': [filters: TableFilter[]]
  'row-click': [row: any, index: number, event: Event]
  'row-double-click': [row: any, index: number, event: Event]
  'cell-click': [cell: any, row: any, column: AdvancedTableColumn, event: Event]
}>()

// 반응형 상태
const scrollContainer = ref<HTMLElement>()
const virtualContainer = ref<HTMLElement>()
const virtualBody = ref<HTMLElement>()
const tableElement = ref<HTMLElement>()

const globalSearchValue = ref('')
const showColumnSettings = ref(false)
const hoveredRowIndex = ref(-1)

const currentPage = ref(1)
const currentPageSize = ref(props.pageSize)
const currentSorters = ref<TableSorter[]>([])
const currentFilters = ref<TableFilter[]>(props.filters || [])
const selectedRowKeys = ref<Array<string | number>>([])

// 가상 스크롤 상태
const virtualStartIndex = ref(0)
const virtualEndIndex = ref(0)
const virtualHeight = ref(0)
const itemHeight = computed(() => props.virtualScroll?.itemHeight || 40)

// @tanstack/vue-virtual을 사용한 고성능 가상화
const virtualizer = computed(() => {
  if (!props.virtualScroll?.enabled || !scrollContainer.value) return null
  
  return useVirtualizer({
    count: paginatedData.value.length,
    getScrollElement: () => scrollContainer.value,
    estimateSize: () => itemHeight.value,
    overscan: props.virtualScroll?.overscan || 5,
    measureElement: (element) => {
      // 동적 높이 측정 지원
      if (typeof props.virtualScroll?.itemHeight === 'string' && props.virtualScroll.itemHeight === 'auto') {
        return element?.getBoundingClientRect().height ?? itemHeight.value
      }
      return itemHeight.value
    },
  })
})

// 접근성 및 성능 최적화 컴포저블
const { 
  announce, 
  announceSortChange,
  announceFilterChange,
  announceDataChange,
  tableAttributes,
  getRowAttributes,
  getCellAttributes,
  focusCell,
  setInitialFocus
} = useTableAccessibility(
  {
    tableId: 'data-table',
    caption: props.ariaLabel || '데이터 테이블',
    enableKeyboardNavigation: true,
    enableCellNavigation: true,
    enableRowSelection: !!props.selection?.type,
    announceChanges: true,
    announceSelection: true,
    announceSort: true,
    announceFilter: true,
  },
  tableElement,
  computed(() => props.data || []),
  computed(() => props.columns || [])
)

const { 
  debounce: debounceFilter, 
  throttle: throttleScroll, 
  memoize 
} = usePerformanceOptimization({
  debounceMs: props.performance?.debounceMs || 300,
  throttleMs: props.performance?.throttleMs || 16,
  enableMemoization: true
})

// 필터 상태 관리
const {
  filters: activeFilters,
  addFilter,
  removeFilter,
  clearAllFilters,
  getFilter,
  hasFilters
} = createFilterState()

const { 
  setupTouchGestures 
} = useTouchGestures({
  element: tableElement,
  onSwipeLeft: () => handleHorizontalScroll('left'),
  onSwipeRight: () => handleHorizontalScroll('right'),
  onPinchZoom: handlePinchZoom
})

// 컬럼 관련 계산 속성
const visibleColumns = computed(() => {
  return props.columns?.filter(col => !col.hideable || col.hideable) || []
})

// 필터링된 데이터 (메모이제이션 적용)
const filteredData = computed(() => {
  // 메모이제이션을 위한 키 생성
  const cacheKey = JSON.stringify({
    dataLength: props.data?.length || 0,
    globalSearch: globalSearchValue.value,
    filters: activeFilters.value,
    dataHash: props.data?.slice(0, 5).map(item => Object.keys(item).length).join('') // 간단한 해시
  })
  
  const memoizedFilter = memoize((data: any[], search: string, filters: TableFilter[]) => {
    let result = [...data]

    // 글로벌 검색 적용
    if (search.trim()) {
      const searchTerm = search.toLowerCase()
      result = result.filter(item => {
        return visibleColumns.value.some(column => {
          const value = getCellDisplayValue(item, column)
          return String(value).toLowerCase().includes(searchTerm)
        })
      })
    }

    // 컬럼 필터 적용
    if (filters.length > 0) {
      result = applyTableFilter(result, filters)
    }

    return result
  }, () => cacheKey)
  
  return memoizedFilter(props.data || [], globalSearchValue.value, activeFilters.value)
})

// 정렬된 데이터
const filteredAndSortedData = computed(() => {
  const result = [...filteredData.value]

  if (currentSorters.value.length > 0) {
    result.sort((a, b) => {
      for (const sorter of currentSorters.value) {
        const aValue = a[sorter.key]
        const bValue = b[sorter.key]

        let comparison = 0

        if (typeof sorter.sorter === 'function') {
          comparison = sorter.sorter(aValue, bValue)
        } else {
          comparison = defaultCompare(aValue, bValue, sorter.sorter)
        }

        if (comparison !== 0) {
          return sorter.order === 'asc' ? comparison : -comparison
        }
      }
      return 0
    })
  }

  return result
})

// 페이지네이션 상태
const paginationState = computed(() => {
  const total = filteredAndSortedData.value.length
  const totalPages = Math.ceil(total / currentPageSize.value)
  const start = (currentPage.value - 1) * currentPageSize.value + 1
  const end = Math.min(currentPage.value * currentPageSize.value, total)

  return {
    page: currentPage.value,
    pageSize: currentPageSize.value,
    total,
    totalPages,
    start,
    end,
  }
})

// 현재 페이지 데이터
const paginatedData = computed(() => {
  if (!props.pagination) return filteredAndSortedData.value

  const start = (currentPage.value - 1) * currentPageSize.value
  const end = start + currentPageSize.value
  return filteredAndSortedData.value.slice(start, end)
})

// 가상 스크롤 아이템들 (개선된 버전)
const visibleItems = computed(() => {
  if (!props.virtualScroll?.enabled || !virtualizer.value) {
    return paginatedData.value
  }

  const items = virtualizer.value.getVirtualItems()
  return items.map(virtualItem => ({
    ...paginatedData.value[virtualItem.index],
    virtualItem
  }))
})

// 가상 스크롤 총 높이
const virtualTotalSize = computed(() => {
  return virtualizer.value?.getTotalSize() ?? 0
})

// 선택 상태 계산
const isAllSelected = computed(() => {
  const dataKeys = paginatedData.value.map((item, index) => getRowKey(item, index))
  return dataKeys.length > 0 && dataKeys.every(key => selectedRowKeys.value.includes(key))
})

const isPartiallySelected = computed(() => {
  const dataKeys = paginatedData.value.map((item, index) => getRowKey(item, index))
  const selected = dataKeys.filter(key => selectedRowKeys.value.includes(key))
  return selected.length > 0 && selected.length < dataKeys.length
})

// 유틸리티 함수들
const getRowKey = (item: any, index: number): string | number => {
  if (typeof props.rowKey === 'function') {
    return props.rowKey(item)
  }
  if (typeof props.rowKey === 'string') {
    return item[props.rowKey]
  }
  return index
}

const getRowClassName = (item: any, index: number): string => {
  if (typeof props.rowClassName === 'function') {
    return props.rowClassName(item, index)
  }
  return props.rowClassName || ''
}

const getCellValue = (item: any, column: AdvancedTableColumn): any => {
  return item[column.key]
}

const getCellDisplayValue = (item: any, column: AdvancedTableColumn): string => {
  const value = getCellValue(item, column)
  return value?.toString() || ''
}

const getColumnStyle = (column: AdvancedTableColumn) => {
  const style: Record<string, any> = {}

  if (column.width) style.width = typeof column.width === 'number' ? `${column.width}px` : column.width
  if (column.minWidth) style.minWidth = typeof column.minWidth === 'number' ? `${column.minWidth}px` : column.minWidth
  if (column.maxWidth) style.maxWidth = typeof column.maxWidth === 'number' ? `${column.maxWidth}px` : column.maxWidth

  return style
}

const getSortDirection = (key: string): string => {
  const sorter = currentSorters.value.find(s => s.key === key)
  return sorter?.order || ''
}

const getAriaSortValue = (key: string): string => {
  const direction = getSortDirection(key)
  if (direction === 'asc') return 'ascending'
  if (direction === 'desc') return 'descending'
  return 'none'
}

const isRowSelected = (item: any): boolean => {
  const key = getRowKey(item, paginatedData.value.indexOf(item))
  return selectedRowKeys.value.includes(key)
}

const getFilterValue = (key: string): any => {
  const filter = getFilter(key)
  return filter?.value
}

// 이벤트 핸들러들
const handleGlobalSearch = debounceFilter((value: string) => {
  // 글로벌 검색 결과 개수를 접근성 도구에 알림
  const resultCount = filteredData.value.length
  announce(`검색 결과: ${resultCount}개 항목이 발견되었습니다.`)
})

const handleScroll = throttleScroll((event: Event) => {
  if (!props.virtualScroll?.enabled) return

  updateVirtualScrollRange()
  
  // 스크롤 위치를 접근성 도구에 알림 (필요시)
  const target = event.target as HTMLElement
  if (target && Math.abs(target.scrollTop - target.scrollHeight + target.clientHeight) < 10) {
    announce('테이블 끝에 도달했습니다.')
  }
})

const handleHeaderClick = (column: AdvancedTableColumn) => {
  if (!column.sortable) return

  const existingSorter = currentSorters.value.find(s => s.key === column.key)
  let newOrder: 'asc' | 'desc' | null = null

  if (existingSorter) {
    if (existingSorter.order === 'asc') {
      existingSorter.order = 'desc'
      newOrder = 'desc'
    } else {
      // 정렬 제거
      currentSorters.value = currentSorters.value.filter(s => s.key !== column.key)
      newOrder = null
    }
  } else {
    // 새 정렬 추가 (멀티 정렬이 아니면 기존 정렬 제거)
    if (!column.sort?.multiple) {
      currentSorters.value = []
    }
    currentSorters.value.push({
      key: column.key,
      order: 'asc',
      sorter: column.sort?.compare || 'default',
    })
    newOrder = 'asc'
  }

  // 접근성 안내
  announceSortChange(column.title, newOrder)
  
  emit('update:sorter', currentSorters.value[0] || null)
}

const handleHeaderKeydown = (event: KeyboardEvent, column: AdvancedTableColumn) => {
  if (event.key === 'Enter' || event.key === ' ') {
    event.preventDefault()
    handleHeaderClick(column)
  }
}

// 테이블 전체 키보드 이벤트 핸들러
const handleTableKeydown = (event: KeyboardEvent) => {
  // useTableAccessibility에서 처리하는 키보드 네비게이션
  // 이 함수는 컴포저블 내부에서 자동으로 처리됨
}

const handleFilterChange = (filter: TableFilter) => {
  const column = props.columns?.find(col => col.key === filter.key)
  addFilter(filter)
  
  // 접근성 안내
  if (column) {
    announceFilterChange(column.title, filter.value !== null)
  }
  
  emit('update:filters', activeFilters.value)
  
  // 필터 변경 시 첫 페이지로 이동
  if (currentPage.value > 1) {
    currentPage.value = 1
    emit('update:page', 1)
  }
}

// 필터 초기화
const clearAllTableFilters = () => {
  clearAllFilters()
  emit('update:filters', [])
  
  // 필터 초기화 시 첫 페이지로 이동
  if (currentPage.value > 1) {
    currentPage.value = 1
    emit('update:page', 1)
  }
}

const handleSelectAll = (event: Event) => {
  const target = event.target as HTMLInputElement
  const dataKeys = paginatedData.value.map((item, index) => getRowKey(item, index))

  if (target.checked) {
    // 모든 행 선택
    selectedRowKeys.value = [...new Set([...selectedRowKeys.value, ...dataKeys])]
  } else {
    // 현재 페이지 행 선택 해제
    selectedRowKeys.value = selectedRowKeys.value.filter(key => !dataKeys.includes(key))
  }

  emit('update:checkedRowKeys', selectedRowKeys.value)
}

const handleRowSelection = (item: any, checked: boolean) => {
  const key = getRowKey(item, paginatedData.value.indexOf(item))

  if (props.selection?.type === 'radio') {
    selectedRowKeys.value = checked ? [key] : []
  } else {
    if (checked) {
      selectedRowKeys.value.push(key)
    } else {
      selectedRowKeys.value = selectedRowKeys.value.filter(k => k !== key)
    }
  }

  emit('update:checkedRowKeys', selectedRowKeys.value)
  props.selection?.onSelectionChange?.(selectedRowKeys.value)
}

const handleRowClick = (item: any, index: number, event: Event) => {
  emit('row-click', item, index, event)
  props.onRowClick?.(item, index, event)
}

const handleRowDoubleClick = (item: any, index: number, event: Event) => {
  emit('row-double-click', item, index, event)
  props.onRowDoubleClick?.(item, index, event)
}

const handleCellClick = (cell: any, item: any, column: AdvancedTableColumn, event: Event) => {
  emit('cell-click', cell, item, column, event)
  props.onCellClick?.(cell, item, column, event)
}

const handleResizeStart = (event: MouseEvent, columnIndex: number) => {
  // 컬럼 리사이즈 로직 (나중에 구현)
}

const goToPage = (page: number) => {
  if (page < 1 || page > paginationState.value.totalPages) return

  currentPage.value = page
  emit('update:page', page)

  if (props.virtualScroll?.enabled) {
    updateVirtualScrollRange()
  }
}

// 유틸리티 함수들

const defaultCompare = (a: any, b: any, type: string): number => {
  if (a === b) return 0
  if (a === null || a === undefined) return -1
  if (b === null || b === undefined) return 1

  switch (type) {
    case 'numeric':
      return Number(a) - Number(b)
    case 'date':
      return new Date(a).getTime() - new Date(b).getTime()
    case 'alphanumeric':
      return String(a).localeCompare(String(b), undefined, { numeric: true })
    default:
      return String(a).localeCompare(String(b))
  }
}

const updateVirtualScrollRange = () => {
  // @tanstack/vue-virtual이 자동으로 처리하므로 더 이상 필요하지 않음
  // 하지만 레거시 호환성을 위해 유지
  if (!props.virtualScroll?.enabled) return
  
  // 가상화 라이브러리가 자동으로 처리
  virtualizer.value?.measure()
}

// 새로운 성능 최적화 함수들
const handleHorizontalScroll = (direction: 'left' | 'right') => {
  if (!scrollContainer.value) return
  
  const scrollAmount = 200
  const currentScroll = scrollContainer.value.scrollLeft
  const newScroll = direction === 'left' 
    ? Math.max(0, currentScroll - scrollAmount)
    : currentScroll + scrollAmount
    
  scrollContainer.value.scrollTo({ left: newScroll, behavior: 'smooth' })
}

const handlePinchZoom = (scale: number) => {
  // 모바일 핀치 줌 처리 (나중에 구현)
  console.log('Pinch zoom:', scale)
}

// 라이프사이클
onMounted(() => {
  if (props.virtualScroll?.enabled) {
    updateVirtualScrollRange()
  }
  
  // 접근성 초기 포커스 설정
  nextTick(() => {
    setInitialFocus()
  })
  
  // 터치 제스처 설정 (모바일 지원)
  setupTouchGestures()
  
  // 성능 모니터링
  if (process.env.NODE_ENV === 'development') {
    console.log('BaseDataTable mounted with', {
      dataLength: props.data?.length || 0,
      virtualScrollEnabled: props.virtualScroll?.enabled,
      columnsCount: props.columns?.length || 0
    })
  }
})

// 반응형 업데이트
watch(
  () => props.data,
  (newData, oldData) => {
    if (props.virtualScroll?.enabled) {
      nextTick(() => {
        updateVirtualScrollRange()
      })
    }
    
    // 데이터 변경 시 접근성 안내
    if (newData && oldData && newData.length !== oldData.length) {
      announceDataChange(newData.length)
    }
  },
  { deep: true },
)

watch(
  () => props.selection?.selectedKeys,
  (newKeys) => {
    if (newKeys) {
      selectedRowKeys.value = [...newKeys]
    }
  },
  { immediate: true },
)
</script>

<style scoped lang="scss">
.base-data-table {
  --table-border-color: #e5e7eb;
  --table-header-bg: #f9fafb;
  --table-row-hover-bg: #f3f4f6;
  --table-selected-bg: #eff6ff;
  --table-text-color: #374151;
  --table-header-text-color: #1f2937;

  position: relative;
  border: 1px solid var(--table-border-color);
  border-radius: 6px;
  overflow: hidden;
  background: white;

  &.size-small {
    font-size: 0.875rem;

    .table-cell,
    .table-header-cell {
      padding: 0.5rem;
    }
  }

  &.size-medium {
    font-size: 0.9375rem;

    .table-cell,
    .table-header-cell {
      padding: 0.75rem;
    }
  }

  &.size-large {
    font-size: 1rem;

    .table-cell,
    .table-header-cell {
      padding: 1rem;
    }
  }

  &.loading {
    .table-container {
      opacity: 0.6;
      pointer-events: none;
    }
  }
}

.table-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem;
  border-bottom: 1px solid var(--table-border-color);
  background: var(--table-header-bg);
}

.global-search {
  .search-input {
    padding: 0.5rem;
    border: 1px solid var(--table-border-color);
    border-radius: 4px;
    min-width: 200px;

    &:focus {
      outline: none;
      border-color: #3b82f6;
      box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
    }
  }
}

.table-actions {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.column-settings {
  .settings-btn {
    padding: 0.5rem 1rem;
    border: 1px solid var(--table-border-color);
    border-radius: 4px;
    background: white;
    cursor: pointer;

    &:hover {
      background: #f3f4f6;
    }
  }
}

.filter-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;

  .filter-count {
    font-size: 0.875rem;
    color: #6b7280;
    padding: 0.25rem 0.5rem;
    background: #f3f4f6;
    border-radius: 4px;
  }

  .clear-filters-btn {
    padding: 0.25rem 0.75rem;
    border: 1px solid #dc2626;
    border-radius: 4px;
    background: white;
    color: #dc2626;
    cursor: pointer;
    font-size: 0.875rem;

    &:hover {
      background: #dc2626;
      color: white;
    }
  }
}

.table-container {
  overflow: auto;
  position: relative;
}

.data-table {
  width: 100%;
  border-collapse: collapse;
  table-layout: fixed;
}

.table-head {
  background: var(--table-header-bg);

  &.sticky {
    position: sticky;
    top: 0;
    z-index: 10;
  }
}

.table-header-cell {
  position: relative;
  border-bottom: 1px solid var(--table-border-color);
  border-right: 1px solid var(--table-border-color);
  font-weight: 600;
  color: var(--table-header-text-color);
  text-align: left;
  user-select: none;

  &:last-child {
    border-right: none;
  }

  &.sortable {
    cursor: pointer;

    &:hover {
      background: #f3f4f6;
    }

    &:focus {
      outline: 2px solid #3b82f6;
      outline-offset: -2px;
    }
  }

  &.align-center {
    text-align: center;
  }

  &.align-right {
    text-align: right;
  }

  &.fixed-left {
    position: sticky;
    left: 0;
    z-index: 5;
    background: var(--table-header-bg);
  }

  &.fixed-right {
    position: sticky;
    right: 0;
    z-index: 5;
    background: var(--table-header-bg);
  }

  &.resizable {
    .resize-handle {
      position: absolute;
      top: 0;
      right: 0;
      width: 4px;
      height: 100%;
      cursor: col-resize;
      background: transparent;

      &:hover {
        background: #3b82f6;
      }
    }
  }
}

.sort-indicator {
  margin-left: 0.5rem;
  opacity: 0.3;

  &.asc {
    opacity: 1;

    &::after {
      content: ' ↑';
    }
  }

  &.desc {
    opacity: 1;

    &::after {
      content: ' ↓';
    }
  }
}

.filter-row {
  background: #f8fafc;

  .filter-cell {
    padding: 0.5rem;
    border-bottom: 1px solid var(--table-border-color);
    border-right: 1px solid var(--table-border-color);

    &:last-child {
      border-right: none;
    }
  }
}

.table-body {
  .table-row {
    &:hover {
      background: var(--table-row-hover-bg);
    }

    &.selected {
      background: var(--table-selected-bg);
    }

    &.striped:nth-child(even) {
      background: #f9fafb;
    }
  }
}

.table-cell {
  border-bottom: 1px solid var(--table-border-color);
  border-right: 1px solid var(--table-border-color);
  color: var(--table-text-color);

  &:last-child {
    border-right: none;
  }

  &.align-center {
    text-align: center;
  }

  &.align-right {
    text-align: right;
  }

  &.ellipsis {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &.fixed-left {
    position: sticky;
    left: 0;
    z-index: 3;
    background: white;
  }

  &.fixed-right {
    position: sticky;
    right: 0;
    z-index: 3;
    background: white;
  }
}

.selection-col {
  width: 48px;
  text-align: center;
}

.virtual-container {
  position: relative;
  overflow: hidden;
}

.loading-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(255, 255, 255, 0.8);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 20;
}

.loading-spinner {
  padding: 1rem 2rem;
  background: white;
  border-radius: 6px;
  box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
}

.empty-state {
  padding: 3rem;
  text-align: center;
  color: #6b7280;
}

.table-pagination {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem;
  border-top: 1px solid var(--table-border-color);
  background: var(--table-header-bg);
}

.pagination-controls {
  display: flex;
  align-items: center;
  gap: 1rem;

  button {
    padding: 0.5rem 1rem;
    border: 1px solid var(--table-border-color);
    border-radius: 4px;
    background: white;
    cursor: pointer;

    &:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }

    &:not(:disabled):hover {
      background: #f3f4f6;
    }
  }
}

// 반응형 스타일
@media (max-width: 768px) {
  .table-header {
    flex-direction: column;
    gap: 1rem;
    align-items: stretch;
  }

  .global-search .search-input {
    min-width: auto;
    width: 100%;
  }

  .table-pagination {
    flex-direction: column;
    gap: 1rem;
    align-items: stretch;
  }

  .pagination-controls {
    justify-content: center;
  }
}

// 다크 테마 지원
@media (prefers-color-scheme: dark) {
  .base-data-table {
    --table-border-color: #374151;
    --table-header-bg: #1f2937;
    --table-row-hover-bg: #374151;
    --table-selected-bg: #1e40af;
    --table-text-color: #f3f4f6;
    --table-header-text-color: #f9fafb;

    background: #111827;

    .table-cell.fixed-left,
    .table-cell.fixed-right {
      background: #111827;
    }

    .table-header-cell.fixed-left,
    .table-header-cell.fixed-right {
      background: #1f2937;
    }
  }
}
</style>