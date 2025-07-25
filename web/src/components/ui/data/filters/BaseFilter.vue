<template>
  <div
    :class="[
      'base-filter',
      `filter-${type}`,
      {
        'has-value': hasValue,
        'disabled': disabled
      }
    ]"
  >
    <!-- 텍스트 필터 -->
    <div v-if="type === 'text'" class="text-filter">
      <input
        v-model="filterValue"
        type="text"
        :placeholder="placeholder || '검색...'"
        :disabled="disabled"
        class="filter-input"
        @input="handleInputChange"
        @keydown.enter="applyFilter"
        @keydown.escape="clearFilter"
        :aria-label="`${column.title} 필터`"
      />
      <button
        v-if="hasValue"
        @click="clearFilter"
        class="clear-btn"
        :aria-label="'필터 지우기'"
      >
        ✕
      </button>
    </div>

    <!-- 숫자 필터 -->
    <div v-else-if="type === 'number'" class="number-filter">
      <select
        v-model="numberOperator"
        class="operator-select"
        :disabled="disabled"
        :aria-label="'숫자 필터 연산자'"
      >
        <option value="equals">=</option>
        <option value="gt">&gt;</option>
        <option value="gte">&gt;=</option>
        <option value="lt">&lt;</option>
        <option value="lte">&lt;=</option>
        <option value="between">범위</option>
      </select>
      
      <input
        v-model="filterValue"
        type="number"
        :placeholder="placeholder || '값 입력'"
        :disabled="disabled"
        class="filter-input"
        @input="handleInputChange"
        :aria-label="`${column.title} 필터 값`"
      />
      
      <input
        v-if="numberOperator === 'between'"
        v-model="filterValueEnd"
        type="number"
        placeholder="최대값"
        :disabled="disabled"
        class="filter-input"
        @input="handleInputChange"
        :aria-label="`${column.title} 필터 최대값`"
      />
      
      <button
        v-if="hasValue"
        @click="clearFilter"
        class="clear-btn"
        :aria-label="'필터 지우기'"
      >
        ✕
      </button>
    </div>

    <!-- 날짜 필터 -->
    <div v-else-if="type === 'date'" class="date-filter">
      <select
        v-model="dateOperator"
        class="operator-select"
        :disabled="disabled"
        :aria-label="'날짜 필터 연산자'"
      >
        <option value="equals">날짜</option>
        <option value="after">이후</option>
        <option value="before">이전</option>
        <option value="between">기간</option>
      </select>
      
      <input
        v-model="filterValue"
        type="date"
        :disabled="disabled"
        class="filter-input date-input"
        @input="handleInputChange"
        :aria-label="`${column.title} 필터 시작 날짜`"
      />
      
      <input
        v-if="dateOperator === 'between'"
        v-model="filterValueEnd"
        type="date"
        :disabled="disabled"
        class="filter-input date-input"
        @input="handleInputChange"
        :aria-label="`${column.title} 필터 종료 날짜`"
      />
      
      <button
        v-if="hasValue"
        @click="clearFilter"
        class="clear-btn"
        :aria-label="'필터 지우기'"
      >
        ✕
      </button>
    </div>

    <!-- 선택 필터 -->
    <div v-else-if="type === 'select'" class="select-filter">
      <select
        v-model="filterValue"
        :disabled="disabled"
        class="filter-select"
        @change="handleSelectChange"
        :aria-label="`${column.title} 필터 선택`"
      >
        <option value="">전체</option>
        <option
          v-for="option in selectOptions"
          :key="option.value"
          :value="option.value"
        >
          {{ option.label }}
        </option>
      </select>
    </div>

    <!-- 다중 선택 필터 -->
    <div v-else-if="type === 'multiSelect'" class="multi-select-filter">
      <div class="multi-select-container" @click="toggleDropdown">
        <div class="selected-display">
          <span v-if="selectedValues.length === 0" class="placeholder">
            {{ placeholder || '선택...' }}
          </span>
          <span v-else-if="selectedValues.length === 1">
            {{ getOptionLabel(selectedValues[0]) }}
          </span>
          <span v-else>
            {{ selectedValues.length }}개 선택됨
          </span>
        </div>
        <span class="dropdown-arrow">▼</span>
      </div>
      
      <div v-if="dropdownOpen" class="dropdown-menu" ref="dropdownMenu">
        <div class="dropdown-search">
          <input
            v-model="searchTerm"
            type="text"
            placeholder="검색..."
            class="search-input"
            @click.stop
          />
        </div>
        
        <div class="option-list">
          <label
            v-for="option in filteredOptions"
            :key="option.value"
            class="option-item"
            @click.stop
          >
            <input
              type="checkbox"
              :value="option.value"
              :checked="selectedValues.includes(option.value)"
              @change="handleOptionToggle(option.value)"
            />
            <span>{{ option.label }}</span>
          </label>
        </div>
        
        <div class="dropdown-actions">
          <button @click.stop="selectAll" class="action-btn">전체 선택</button>
          <button @click.stop="clearAll" class="action-btn">전체 해제</button>
        </div>
      </div>
    </div>

    <!-- 불린 필터 -->
    <div v-else-if="type === 'boolean'" class="boolean-filter">
      <select
        v-model="filterValue"
        :disabled="disabled"
        class="filter-select"
        @change="handleSelectChange"
        :aria-label="`${column.title} 필터`"
      >
        <option value="">전체</option>
        <option value="true">예</option>
        <option value="false">아니오</option>
      </select>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch, onMounted, onUnmounted } from 'vue'
import type { AdvancedTableColumn, TableFilter } from '@/types/ui'

interface Props {
  column: AdvancedTableColumn
  type: 'text' | 'number' | 'date' | 'select' | 'multiSelect' | 'boolean'
  value?: any
  placeholder?: string
  disabled?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  disabled: false
})

const emit = defineEmits<{
  'update:value': [value: any]
  'filter-change': [filter: TableFilter]
}>()

// 반응형 상태
const filterValue = ref(props.value || '')
const filterValueEnd = ref('')
const numberOperator = ref('equals')
const dateOperator = ref('equals')
const selectedValues = ref<any[]>(Array.isArray(props.value) ? props.value : [])
const dropdownOpen = ref(false)
const searchTerm = ref('')
const dropdownMenu = ref<HTMLElement>()

// 계산된 속성
const hasValue = computed(() => {
  if (props.type === 'multiSelect') {
    return selectedValues.value.length > 0
  }
  return filterValue.value !== '' && filterValue.value !== null && filterValue.value !== undefined
})

const selectOptions = computed(() => {
  return props.column.filter?.options || []
})

const filteredOptions = computed(() => {
  if (!searchTerm.value) return selectOptions.value
  
  return selectOptions.value.filter(option =>
    option.label.toLowerCase().includes(searchTerm.value.toLowerCase())
  )
})

// 필터 값 처리
const getFilterValue = () => {
  switch (props.type) {
    case 'multiSelect':
      return selectedValues.value.length > 0 ? selectedValues.value : null
    case 'number':
      if (numberOperator.value === 'between') {
        return filterValue.value && filterValueEnd.value 
          ? [Number(filterValue.value), Number(filterValueEnd.value)]
          : null
      }
      return filterValue.value ? Number(filterValue.value) : null
    case 'date':
      if (dateOperator.value === 'between') {
        return filterValue.value && filterValueEnd.value 
          ? [filterValue.value, filterValueEnd.value]
          : null
      }
      return filterValue.value || null
    case 'boolean':
      return filterValue.value === '' ? null : filterValue.value === 'true'
    default:
      return filterValue.value || null
  }
}

const getFilterOperator = () => {
  switch (props.type) {
    case 'number':
      return numberOperator.value
    case 'date':
      return dateOperator.value === 'after' ? 'gt' : 
             dateOperator.value === 'before' ? 'lt' :
             dateOperator.value
    case 'multiSelect':
      return 'in'
    case 'boolean':
      return 'equals'
    default:
      return 'contains'
  }
}

// 이벤트 핸들러
const handleInputChange = () => {
  applyFilter()
}

const handleSelectChange = () => {
  applyFilter()
}

const applyFilter = () => {
  const value = getFilterValue()
  const operator = getFilterOperator()
  
  emit('update:value', value)
  
  if (value !== null) {
    emit('filter-change', {
      key: props.column.key,
      value,
      operator,
      type: props.type
    })
  } else {
    // 빈 값일 때는 필터 제거
    emit('filter-change', {
      key: props.column.key,
      value: null,
      operator,
      type: props.type
    })
  }
}

const clearFilter = () => {
  filterValue.value = ''
  filterValueEnd.value = ''
  selectedValues.value = []
  applyFilter()
}

// 다중 선택 관련
const toggleDropdown = () => {
  dropdownOpen.value = !dropdownOpen.value
}

const handleOptionToggle = (value: any) => {
  const index = selectedValues.value.indexOf(value)
  if (index > -1) {
    selectedValues.value.splice(index, 1)
  } else {
    selectedValues.value.push(value)
  }
  applyFilter()
}

const getOptionLabel = (value: any): string => {
  const option = selectOptions.value.find(opt => opt.value === value)
  return option?.label || String(value)
}

const selectAll = () => {
  selectedValues.value = [...selectOptions.value.map(opt => opt.value)]
  applyFilter()
}

const clearAll = () => {
  selectedValues.value = []
  applyFilter()
}

// 드롭다운 외부 클릭 감지
const handleClickOutside = (event: Event) => {
  if (dropdownMenu.value && !dropdownMenu.value.contains(event.target as Node)) {
    dropdownOpen.value = false
  }
}

// 라이프사이클
onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})

// 초기값 설정
watch(
  () => props.value,
  (newValue) => {
    if (props.type === 'multiSelect') {
      selectedValues.value = Array.isArray(newValue) ? newValue : []
    } else {
      filterValue.value = newValue || ''
    }
  },
  { immediate: true }
)
</script>

<style scoped lang="scss">
.base-filter {
  position: relative;
  min-width: 120px;

  &.disabled {
    opacity: 0.6;
    pointer-events: none;
  }
}

.filter-input {
  width: 100%;
  padding: 0.25rem 0.5rem;
  border: 1px solid #d1d5db;
  border-radius: 4px;
  font-size: 0.875rem;
  background: white;

  &:focus {
    outline: none;
    border-color: #3b82f6;
    box-shadow: 0 0 0 1px #3b82f6;
  }

  &:disabled {
    background: #f9fafb;
    cursor: not-allowed;
  }
}

.filter-select {
  width: 100%;
  padding: 0.25rem 0.5rem;
  border: 1px solid #d1d5db;
  border-radius: 4px;
  font-size: 0.875rem;
  background: white;

  &:focus {
    outline: none;
    border-color: #3b82f6;
    box-shadow: 0 0 0 1px #3b82f6;
  }
}

.clear-btn {
  position: absolute;
  right: 0.25rem;
  top: 50%;
  transform: translateY(-50%);
  width: 18px;
  height: 18px;
  border: none;
  background: #6b7280;
  color: white;
  border-radius: 50%;
  cursor: pointer;
  font-size: 0.75rem;
  display: flex;
  align-items: center;
  justify-content: center;

  &:hover {
    background: #374151;
  }
}

// 텍스트 필터
.text-filter {
  position: relative;

  .filter-input {
    padding-right: 2rem;
  }
}

// 숫자 필터
.number-filter {
  display: flex;
  gap: 0.25rem;

  .operator-select {
    width: 60px;
    flex-shrink: 0;
  }

  .filter-input {
    flex: 1;
    min-width: 80px;
  }
}

// 날짜 필터
.date-filter {
  display: flex;
  gap: 0.25rem;

  .operator-select {
    width: 60px;
    flex-shrink: 0;
  }

  .date-input {
    flex: 1;
    min-width: 120px;
  }
}

// 다중 선택 필터
.multi-select-container {
  position: relative;
  border: 1px solid #d1d5db;
  border-radius: 4px;
  padding: 0.25rem 0.5rem;
  background: white;
  cursor: pointer;
  display: flex;
  justify-content: space-between;
  align-items: center;
  min-height: 2rem;

  &:hover {
    border-color: #9ca3af;
  }

  .placeholder {
    color: #6b7280;
  }

  .dropdown-arrow {
    font-size: 0.75rem;
    color: #6b7280;
    transition: transform 0.2s;
  }

  &:focus-within {
    border-color: #3b82f6;
    box-shadow: 0 0 0 1px #3b82f6;

    .dropdown-arrow {
      transform: rotate(180deg);
    }
  }
}

.dropdown-menu {
  position: absolute;
  top: 100%;
  left: 0;
  right: 0;
  z-index: 50;
  background: white;
  border: 1px solid #d1d5db;
  border-radius: 4px;
  box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.1);
  max-height: 200px;
  overflow: hidden;

  .dropdown-search {
    padding: 0.5rem;
    border-bottom: 1px solid #e5e7eb;

    .search-input {
      width: 100%;
      padding: 0.25rem;
      border: 1px solid #d1d5db;
      border-radius: 4px;
      font-size: 0.875rem;
    }
  }

  .option-list {
    max-height: 120px;
    overflow-y: auto;
  }

  .option-item {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.5rem;
    cursor: pointer;
    font-size: 0.875rem;

    &:hover {
      background: #f3f4f6;
    }

    input[type="checkbox"] {
      margin: 0;
    }
  }

  .dropdown-actions {
    padding: 0.5rem;
    border-top: 1px solid #e5e7eb;
    display: flex;
    gap: 0.5rem;

    .action-btn {
      flex: 1;
      padding: 0.25rem;
      border: 1px solid #d1d5db;
      border-radius: 4px;
      background: white;
      font-size: 0.75rem;
      cursor: pointer;

      &:hover {
        background: #f3f4f6;
      }
    }
  }
}

// 반응형 스타일
@media (max-width: 768px) {
  .base-filter {
    min-width: 100px;
  }

  .number-filter,
  .date-filter {
    flex-direction: column;
    gap: 0.125rem;

    .operator-select {
      width: 100%;
    }
  }

  .dropdown-menu {
    .option-item {
      padding: 0.75rem 0.5rem;
    }
  }
}

// 다크 테마 지원
@media (prefers-color-scheme: dark) {
  .filter-input,
  .filter-select,
  .multi-select-container {
    background: #374151;
    border-color: #4b5563;
    color: #f3f4f6;

    &:focus {
      border-color: #60a5fa;
      box-shadow: 0 0 0 1px #60a5fa;
    }

    &:disabled {
      background: #1f2937;
    }
  }

  .dropdown-menu {
    background: #374151;
    border-color: #4b5563;

    .dropdown-search {
      border-bottom-color: #4b5563;

      .search-input {
        background: #1f2937;
        border-color: #4b5563;
        color: #f3f4f6;
      }
    }

    .option-item {
      &:hover {
        background: #4b5563;
      }
    }

    .dropdown-actions {
      border-top-color: #4b5563;

      .action-btn {
        background: #1f2937;
        border-color: #4b5563;
        color: #f3f4f6;

        &:hover {
          background: #4b5563;
        }
      }
    }
  }

  .clear-btn {
    background: #9ca3af;

    &:hover {
      background: #6b7280;
    }
  }
}
</style>