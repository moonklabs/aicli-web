// 필터 컴포넌트 내보내기
export { default as BaseFilter } from './BaseFilter.vue'

// 필터 유틸리티 함수들
export function applyTableFilter(data: any[], filters: any[]): any[] {
  return data.filter(item => {
    return filters.every(filter => {
      const cellValue = item[filter.key]
      return evaluateFilter(cellValue, filter)
    })
  })
}

export function evaluateFilter(value: any, filter: any): boolean {
  if (filter.value === null || filter.value === undefined) {
    return true // 필터가 없으면 모든 항목 통과
  }

  const { operator, type, value: filterValue } = filter

  // null/undefined 값 처리
  if (value === null || value === undefined) {
    return operator === 'equals' && filterValue === null
  }

  switch (type) {
    case 'text':
      return evaluateTextFilter(value, filterValue, operator)
    case 'number':
      return evaluateNumberFilter(value, filterValue, operator)
    case 'date':
      return evaluateDateFilter(value, filterValue, operator)
    case 'select':
    case 'boolean':
      return evaluateSelectFilter(value, filterValue, operator)
    case 'multiSelect':
      return evaluateMultiSelectFilter(value, filterValue, operator)
    default:
      return true
  }
}

function evaluateTextFilter(value: any, filterValue: string, operator: string): boolean {
  const stringValue = String(value).toLowerCase()
  const stringFilterValue = String(filterValue).toLowerCase()

  switch (operator) {
    case 'equals':
      return stringValue === stringFilterValue
    case 'contains':
      return stringValue.includes(stringFilterValue)
    case 'startsWith':
      return stringValue.startsWith(stringFilterValue)
    case 'endsWith':
      return stringValue.endsWith(stringFilterValue)
    default:
      return stringValue.includes(stringFilterValue)
  }
}

function evaluateNumberFilter(value: any, filterValue: number | number[], operator: string): boolean {
  const numValue = Number(value)
  
  if (Array.isArray(filterValue) && operator === 'between') {
    const [min, max] = filterValue
    return numValue >= min && numValue <= max
  }

  const numFilterValue = Number(filterValue)

  switch (operator) {
    case 'equals':
      return numValue === numFilterValue
    case 'gt':
      return numValue > numFilterValue
    case 'gte':
      return numValue >= numFilterValue
    case 'lt':
      return numValue < numFilterValue
    case 'lte':
      return numValue <= numFilterValue
    default:
      return numValue === numFilterValue
  }
}

function evaluateDateFilter(value: any, filterValue: string | string[], operator: string): boolean {
  const dateValue = new Date(value)
  
  if (Array.isArray(filterValue) && operator === 'between') {
    const [startDate, endDate] = filterValue.map(d => new Date(d))
    return dateValue >= startDate && dateValue <= endDate
  }

  const filterDate = new Date(filterValue as string)

  switch (operator) {
    case 'equals':
      return dateValue.toDateString() === filterDate.toDateString()
    case 'gt':
      return dateValue > filterDate
    case 'gte':
      return dateValue >= filterDate
    case 'lt':
      return dateValue < filterDate
    case 'lte':
      return dateValue <= filterDate
    default:
      return dateValue.toDateString() === filterDate.toDateString()
  }
}

function evaluateSelectFilter(value: any, filterValue: any, operator: string): boolean {
  switch (operator) {
    case 'equals':
      return value === filterValue
    default:
      return value === filterValue
  }
}

function evaluateMultiSelectFilter(value: any, filterValue: any[], operator: string): boolean {
  if (!Array.isArray(filterValue) || filterValue.length === 0) {
    return true
  }

  switch (operator) {
    case 'in':
      return filterValue.includes(value)
    case 'notIn':
      return !filterValue.includes(value)
    default:
      return filterValue.includes(value)
  }
}

// 필터 상태 관리 헬퍼
export function createFilterState() {
  const filters = ref<any[]>([])
  
  const addFilter = (filter: any) => {
    const existingIndex = filters.value.findIndex(f => f.key === filter.key)
    if (existingIndex >= 0) {
      if (filter.value === null) {
        // 필터 제거
        filters.value.splice(existingIndex, 1)
      } else {
        // 필터 업데이트
        filters.value[existingIndex] = filter
      }
    } else if (filter.value !== null) {
      // 새 필터 추가
      filters.value.push(filter)
    }
  }
  
  const removeFilter = (key: string) => {
    const index = filters.value.findIndex(f => f.key === key)
    if (index >= 0) {
      filters.value.splice(index, 1)
    }
  }
  
  const clearAllFilters = () => {
    filters.value = []
  }
  
  const getFilter = (key: string) => {
    return filters.value.find(f => f.key === key)
  }
  
  const hasFilters = computed(() => filters.value.length > 0)
  
  return {
    filters: readonly(filters),
    addFilter,
    removeFilter,
    clearAllFilters,
    getFilter,
    hasFilters
  }
}

import { computed, readonly, ref } from 'vue'