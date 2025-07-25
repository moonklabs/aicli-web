<template>
  <div
    :class="[
      'app-select',
      sizeClasses,
      statusClasses,
      {
        'app-select--disabled': disabled,
        'app-select--focused': focused,
        'app-select--opened': opened,
        'app-select--clearable': clearable && clearVisible,
        'app-select--multiple': multiple,
        'app-select--searchable': searchable,
        'app-select--round': round
      }
    ]"
    v-click-outside="handleClickOutside"
  >
    <!-- 선택 트리거 -->
    <div
      ref="triggerRef"
      class="app-select__trigger"
      :tabindex="disabled ? -1 : 0"
      :aria-expanded="opened"
      :aria-haspopup="'listbox'"
      :aria-labelledby="ariaLabelledby"
      :aria-describedby="ariaDescribedby"
      :aria-invalid="status === 'error'"
      :aria-required="required"
      role="combobox"
      @click="handleTriggerClick"
      @keydown="handleTriggerKeydown"
      @focus="handleFocus"
      @blur="handleBlur"
    >
      <!-- 선택된 값 표시 영역 -->
      <div class="app-select__value-container">
        <!-- 다중 선택 태그들 -->
        <div v-if="multiple && selectedItems.length > 0" class="app-select__tags">
          <div
            v-for="(item, index) in selectedItems"
            :key="getItemKey(item)"
            class="app-select__tag"
          >
            <span class="app-select__tag-text">{{ getItemLabel(item) }}</span>
            <button
              v-if="!disabled"
              type="button"
              class="app-select__tag-close"
              :aria-label="`${getItemLabel(item)} 제거`"
              @click.stop="handleTagRemove(item, index)"
              @keydown.stop="handleTagCloseKeydown($event, item, index)"
            >
              <svg viewBox="0 0 16 16" fill="currentColor">
                <path d="M4.646 4.646a.5.5 0 0 1 .708 0L8 7.293l2.646-2.647a.5.5 0 0 1 .708.708L8.707 8l2.647 2.646a.5.5 0 0 1-.708.708L8 8.707l-2.646 2.647a.5.5 0 0 1-.708-.708L7.293 8 4.646 5.354a.5.5 0 0 1 0-.708z"/>
              </svg>
            </button>
          </div>
        </div>

        <!-- 단일 선택 값 또는 검색 입력 -->
        <div v-if="!multiple || selectedItems.length === 0" class="app-select__input-wrapper">
          <input
            v-if="searchable"
            ref="searchInputRef"
            v-model="searchQuery"
            type="text"
            class="app-select__search-input"
            :placeholder="computedPlaceholder"
            :disabled="disabled"
            :readonly="!opened"
            @input="handleSearchInput"
            @keydown="handleSearchKeydown"
          />
          <div v-else class="app-select__display-value">
            {{ displayValue || computedPlaceholder }}
          </div>
        </div>
      </div>

      <!-- 클리어 버튼 -->
      <button
        v-if="clearable && clearVisible"
        type="button"
        class="app-select__clear"
        :aria-label="'선택 해제'"
        @click.stop="handleClear"
        @keydown.stop="handleClearKeydown"
      >
        <svg viewBox="0 0 16 16" fill="currentColor">
          <path d="M8 1a7 7 0 1 0 0 14A7 7 0 0 0 8 1zM6.354 5.646a.5.5 0 1 0-.708.708L7.293 8l-1.647 1.646a.5.5 0 0 0 .708.708L8 8.707l1.646 1.647a.5.5 0 0 0 .708-.708L8.707 8l1.647-1.646a.5.5 0 0 0-.708-.708L8 7.293 6.354 5.646z"/>
        </svg>
      </button>

      <!-- 드롭다운 화살표 -->
      <div class="app-select__arrow" :class="{ 'app-select__arrow--rotated': opened }">
        <svg viewBox="0 0 16 16" fill="currentColor">
          <path d="M4.427 6.573L8 10.146l3.573-3.573a.5.5 0 1 1 .708.708L8.354 11.208a.5.5 0 0 1-.708 0L3.719 7.281a.5.5 0 1 1 .708-.708z"/>
        </svg>
      </div>
    </div>

    <!-- 드롭다운 메뉴 -->
    <transition
      name="app-select-dropdown"
      @before-enter="onBeforeEnter"
      @after-leave="onAfterLeave"
    >
      <div
        v-show="opened"
        ref="dropdownRef"
        class="app-select__dropdown"
        :style="dropdownStyle"
        role="listbox"
        :aria-multiselectable="multiple"
        :aria-activedescendant="focusedOptionId"
      >
        <!-- 로딩 상태 -->
        <div v-if="loading" class="app-select__loading">
          <AppSpinner size="small" />
          <span>로딩 중...</span>
        </div>

        <!-- 옵션 목록 -->
        <div v-else-if="filteredOptions.length > 0" class="app-select__options">
          <div
            v-for="(option, index) in filteredOptions"
            :key="getItemKey(option)"
            :id="getOptionId(index)"
            class="app-select__option"
            :class="getOptionClasses(option, index)"
            role="option"
            :aria-selected="isSelected(option)"
            @click="handleOptionClick(option)"
            @mouseenter="focusedIndex = index"
          >
            <!-- 옵션 체크박스 (다중 선택) -->
            <div v-if="multiple" class="app-select__option-checkbox">
              <svg
                v-if="isSelected(option)"
                viewBox="0 0 16 16"
                fill="currentColor"
                class="app-select__option-check"
              >
                <path d="M13.854 3.646a.5.5 0 0 1 0 .708l-7 7a.5.5 0 0 1-.708 0l-3.5-3.5a.5.5 0 1 1 .708-.708L6.5 10.293l6.646-6.647a.5.5 0 0 1 .708 0z"/>
              </svg>
            </div>

            <!-- 옵션 내용 -->
            <div class="app-select__option-content">
              <slot name="option" :option="option" :index="index">
                {{ getItemLabel(option) }}
              </slot>
            </div>
          </div>
        </div>

        <!-- 옵션 없음 -->
        <div v-else class="app-select__empty">
          <slot name="empty">
            {{ searchQuery ? '검색 결과가 없습니다' : '옵션이 없습니다' }}
          </slot>
        </div>
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import AppSpinner from '../feedback/AppSpinner.vue'
import clickOutside from '@/directives/click-outside'

interface SelectOption {
  label: string;
  value: any;
  disabled?: boolean;
  [key: string]: any;
}

interface Props {
  modelValue?: any;
  options?: SelectOption[];
  placeholder?: string;
  disabled?: boolean;
  size?: 'small' | 'medium' | 'large';
  status?: 'default' | 'success' | 'warning' | 'error';
  clearable?: boolean;
  searchable?: boolean;
  multiple?: boolean;
  loading?: boolean;
  round?: boolean;
  required?: boolean;
  labelField?: string;
  valueField?: string;
  ariaLabelledby?: string;
  ariaDescribedby?: string;
  maxTagCount?: number;
  remoteSearch?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  options: () => [],
  size: 'medium',
  status: 'default',
  disabled: false,
  clearable: false,
  searchable: false,
  multiple: false,
  loading: false,
  round: false,
  required: false,
  labelField: 'label',
  valueField: 'value',
  maxTagCount: -1,
  remoteSearch: false,
})

const emit = defineEmits<{
  'update:value': [value: any];
  'change': [value: any];
  'focus': [event: FocusEvent];
  'blur': [event: FocusEvent];
  'clear': [];
  'search': [query: string];
  'visibleChange': [visible: boolean];
}>()

// 반응형 상태
const triggerRef = ref<HTMLDivElement>()
const dropdownRef = ref<HTMLDivElement>()
const searchInputRef = ref<HTMLInputElement>()
const focused = ref(false)
const opened = ref(false)
const searchQuery = ref('')
const focusedIndex = ref(-1)
const dropdownStyle = ref({})

// 선택된 아이템들
const selectedItems = computed(() => {
  if (!props.multiple) {
    const item = props.options.find(option =>
      getItemValue(option) === props.modelValue,
    )
    return item ? [item] : []
  }

  const values = Array.isArray(props.modelValue) ? props.modelValue : []
  return props.options.filter(option =>
    values.includes(getItemValue(option)),
  )
})

// 표시할 값
const displayValue = computed(() => {
  if (props.multiple) return ''

  const selected = selectedItems.value[0]
  return selected ? getItemLabel(selected) : ''
})

// 플레이스홀더 계산
const computedPlaceholder = computed(() => {
  if (props.multiple && selectedItems.value.length > 0) {
    return ''
  }
  return props.placeholder || '선택해주세요'
})

// 클리어 버튼 표시 여부
const clearVisible = computed(() => {
  if (props.multiple) {
    return selectedItems.value.length > 0
  }
  return props.modelValue != null && props.modelValue !== ''
})

// 필터링된 옵션들
const filteredOptions = computed(() => {
  if (!props.searchable || !searchQuery.value.trim()) {
    return props.options
  }

  const query = searchQuery.value.toLowerCase()
  return props.options.filter(option =>
    getItemLabel(option).toLowerCase().includes(query),
  )
})

// 포커스된 옵션 ID
const focusedOptionId = computed(() => {
  return focusedIndex.value >= 0 ? getOptionId(focusedIndex.value) : undefined
})

// 사이즈별 클래스
const sizeClasses = computed(() => ({
  'app-select--small': props.size === 'small',
  'app-select--medium': props.size === 'medium',
  'app-select--large': props.size === 'large',
}))

// 상태별 클래스
const statusClasses = computed(() => ({
  'app-select--default': props.status === 'default',
  'app-select--success': props.status === 'success',
  'app-select--warning': props.status === 'warning',
  'app-select--error': props.status === 'error',
}))

// 유틸리티 함수들
const getItemLabel = (item: SelectOption): string => {
  return String(item[props.labelField] || item.label || item)
}

const getItemValue = (item: SelectOption): any => {
  return item[props.valueField] || item.value || item
}

const getItemKey = (item: SelectOption): string => {
  return String(getItemValue(item))
}

const getOptionId = (index: number): string => {
  return `option-${index}`
}

const isSelected = (option: SelectOption): boolean => {
  if (props.multiple) {
    const values = Array.isArray(props.modelValue) ? props.modelValue : []
    return values.includes(getItemValue(option))
  }
  return getItemValue(option) === props.modelValue
}

const getOptionClasses = (option: SelectOption, index: number) => ({
  'app-select__option--selected': isSelected(option),
  'app-select__option--focused': index === focusedIndex.value,
  'app-select__option--disabled': option.disabled,
})

// 드롭다운 위치 계산
const updateDropdownPosition = () => {
  if (!triggerRef.value || !dropdownRef.value) return

  const triggerRect = triggerRef.value.getBoundingClientRect()
  const dropdownHeight = dropdownRef.value.offsetHeight
  const viewportHeight = window.innerHeight

  const spaceBelow = viewportHeight - triggerRect.bottom
  const spaceAbove = triggerRect.top

  // 아래쪽 공간이 충분하거나 위쪽보다 크면 아래에 표시
  const showBelow = spaceBelow >= dropdownHeight || spaceBelow >= spaceAbove

  dropdownStyle.value = {
    position: 'fixed',
    left: `${triggerRect.left}px`,
    width: `${triggerRect.width}px`,
    ...(showBelow
      ? { top: `${triggerRect.bottom}px` }
      : { bottom: `${viewportHeight - triggerRect.top}px` }
    ),
  }
}

// 이벤트 핸들러들
const handleTriggerClick = () => {
  if (props.disabled) return

  opened.value = !opened.value

  if (opened.value) {
    nextTick(() => {
      updateDropdownPosition()
      if (props.searchable && searchInputRef.value) {
        searchInputRef.value.focus()
      }
    })
  }

  emit('visibleChange', opened.value)
}

const handleTriggerKeydown = (event: KeyboardEvent) => {
  if (props.disabled) return

  switch (event.key) {
    case 'Enter':
    case ' ':
      event.preventDefault()
      handleTriggerClick()
      break
    case 'ArrowDown':
      event.preventDefault()
      if (!opened.value) {
        handleTriggerClick()
      } else {
        focusNextOption()
      }
      break
    case 'ArrowUp':
      event.preventDefault()
      if (opened.value) {
        focusPreviousOption()
      }
      break
    case 'Escape':
      if (opened.value) {
        opened.value = false
        emit('visibleChange', false)
      }
      break
  }
}

const handleSearchKeydown = (event: KeyboardEvent) => {
  switch (event.key) {
    case 'ArrowDown':
      event.preventDefault()
      focusNextOption()
      break
    case 'ArrowUp':
      event.preventDefault()
      focusPreviousOption()
      break
    case 'Enter':
      event.preventDefault()
      if (focusedIndex.value >= 0) {
        const option = filteredOptions.value[focusedIndex.value]
        if (option && !option.disabled) {
          handleOptionClick(option)
        }
      }
      break
    case 'Escape':
      opened.value = false
      emit('visibleChange', false)
      break
  }
}

const handleSearchInput = () => {
  emit('search', searchQuery.value)
  focusedIndex.value = -1
}

const handleOptionClick = (option: SelectOption) => {
  if (option.disabled) return

  if (props.multiple) {
    const values = Array.isArray(props.modelValue) ? [...props.modelValue] : []
    const value = getItemValue(option)
    const index = values.indexOf(value)

    if (index >= 0) {
      values.splice(index, 1)
    } else {
      values.push(value)
    }

    emit('update:value', values)
    emit('change', values)
  } else {
    const value = getItemValue(option)
    emit('update:value', value)
    emit('change', value)

    opened.value = false
    emit('visibleChange', false)
  }

  // 검색어 초기화
  if (props.searchable && !props.multiple) {
    searchQuery.value = ''
  }
}

const handleTagRemove = (item: SelectOption, index: number) => {
  if (!props.multiple) return

  const values = Array.isArray(props.modelValue) ? [...props.modelValue] : []
  const value = getItemValue(item)
  const valueIndex = values.indexOf(value)

  if (valueIndex >= 0) {
    values.splice(valueIndex, 1)
    emit('update:value', values)
    emit('change', values)
  }
}

const handleTagCloseKeydown = (event: KeyboardEvent, item: SelectOption, index: number) => {
  if (event.key === 'Enter' || event.key === ' ') {
    event.preventDefault()
    handleTagRemove(item, index)
  }
}

const handleClear = () => {
  const value = props.multiple ? [] : null
  emit('update:value', value)
  emit('change', value)
  emit('clear')

  if (props.searchable) {
    searchQuery.value = ''
  }
}

const handleClearKeydown = (event: KeyboardEvent) => {
  if (event.key === 'Enter' || event.key === ' ') {
    event.preventDefault()
    handleClear()
  }
}

const handleFocus = (event: FocusEvent) => {
  focused.value = true
  emit('focus', event)
}

const handleBlur = (event: FocusEvent) => {
  focused.value = false
  emit('blur', event)
}

const handleClickOutside = () => {
  if (opened.value) {
    opened.value = false
    emit('visibleChange', false)
  }
}

// 포커스 관리
const focusNextOption = () => {
  const maxIndex = filteredOptions.value.length - 1

  do {
    focusedIndex.value = focusedIndex.value < maxIndex ? focusedIndex.value + 1 : 0
  } while (
    filteredOptions.value[focusedIndex.value]?.disabled &&
    focusedIndex.value !== 0
  )
}

const focusPreviousOption = () => {
  const maxIndex = filteredOptions.value.length - 1

  do {
    focusedIndex.value = focusedIndex.value > 0 ? focusedIndex.value - 1 : maxIndex
  } while (
    filteredOptions.value[focusedIndex.value]?.disabled &&
    focusedIndex.value !== maxIndex
  )
}

// 트랜지션 이벤트
const onBeforeEnter = () => {
  updateDropdownPosition()
}

const onAfterLeave = () => {
  focusedIndex.value = -1
  if (props.searchable) {
    searchQuery.value = ''
  }
}

// 포커스 메서드 노출
const focus = () => {
  triggerRef.value?.focus()
}

const blur = () => {
  triggerRef.value?.blur()
}

defineExpose({
  focus,
  blur,
  opened,
  triggerRef,
  dropdownRef,
})

// 스크롤 시 드롭다운 위치 업데이트
const handleScroll = () => {
  if (opened.value) {
    updateDropdownPosition()
  }
}

onMounted(() => {
  window.addEventListener('scroll', handleScroll, true)
  window.addEventListener('resize', handleScroll)
})

onUnmounted(() => {
  window.removeEventListener('scroll', handleScroll, true)
  window.removeEventListener('resize', handleScroll)
})

// 옵션 변경 시 포커스 초기화
watch(() => filteredOptions.value.length, () => {
  focusedIndex.value = -1
})
</script>

<style lang="scss" scoped>
.app-select {
  @apply relative inline-block w-full;

  &__trigger {
    @apply relative flex items-center w-full;
    @apply bg-white border border-solid border-gray-300 rounded-md;
    @apply cursor-pointer transition-colors duration-200;
    @apply focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-opacity-20;
    @apply focus:border-blue-500;
  }

  &--disabled {
    .app-select__trigger {
      @apply bg-gray-50 border-gray-200 cursor-not-allowed;
    }
  }

  &--focused {
    .app-select__trigger {
      @apply border-blue-500;
    }
  }

  &--opened {
    .app-select__trigger {
      @apply border-blue-500 ring-2 ring-blue-500 ring-opacity-20;
    }
  }

  &--round {
    .app-select__trigger {
      @apply rounded-full;
    }
  }

  // 사이즈 변형
  &--small {
    .app-select__trigger {
      @apply min-h-[32px] text-sm;
    }

    .app-select__value-container {
      @apply px-2 py-1;
    }

    .app-select__tag {
      @apply px-1.5 py-0.5 text-xs;
    }

    .app-select__clear,
    .app-select__arrow {
      @apply w-4 h-4 mx-1;
    }
  }

  &--medium {
    .app-select__trigger {
      @apply min-h-[40px] text-base;
    }

    .app-select__value-container {
      @apply px-3 py-2;
    }

    .app-select__tag {
      @apply px-2 py-1 text-sm;
    }

    .app-select__clear,
    .app-select__arrow {
      @apply w-5 h-5 mx-2;
    }
  }

  &--large {
    .app-select__trigger {
      @apply min-h-[48px] text-lg;
    }

    .app-select__value-container {
      @apply px-4 py-3;
    }

    .app-select__tag {
      @apply px-2.5 py-1.5 text-base;
    }

    .app-select__clear,
    .app-select__arrow {
      @apply w-6 h-6 mx-3;
    }
  }

  // 상태별 색상
  &--success {
    .app-select__trigger {
      @apply border-green-500;

      &:focus {
        @apply ring-green-500 ring-opacity-20 border-green-500;
      }
    }
  }

  &--warning {
    .app-select__trigger {
      @apply border-yellow-500;

      &:focus {
        @apply ring-yellow-500 ring-opacity-20 border-yellow-500;
      }
    }
  }

  &--error {
    .app-select__trigger {
      @apply border-red-500;

      &:focus {
        @apply ring-red-500 ring-opacity-20 border-red-500;
      }
    }
  }

  // 컴포넌트 요소들
  &__value-container {
    @apply flex-1 flex items-center flex-wrap gap-1;
  }

  &__tags {
    @apply flex items-center flex-wrap gap-1;
  }

  &__tag {
    @apply inline-flex items-center gap-1;
    @apply bg-blue-100 text-blue-800 rounded-md;
    @apply border border-blue-200;
  }

  &__tag-text {
    @apply truncate max-w-[120px];
  }

  &__tag-close {
    @apply inline-flex items-center justify-center;
    @apply text-blue-600 hover:text-blue-800;
    @apply rounded transition-colors duration-200;
    @apply focus:outline-none focus:ring-1 focus:ring-blue-500;

    svg {
      @apply w-3 h-3;
    }
  }

  &__input-wrapper {
    @apply flex-1 min-w-0;
  }

  &__search-input {
    @apply w-full bg-transparent border-none outline-none;
    @apply text-gray-900 placeholder-gray-500;
  }

  &__display-value {
    @apply truncate text-gray-900;

    &:empty::before {
      content: attr(data-placeholder);
      @apply text-gray-500;
    }
  }

  &__clear {
    @apply inline-flex items-center justify-center;
    @apply text-gray-400 hover:text-gray-600;
    @apply cursor-pointer transition-colors duration-200;
    @apply focus:outline-none focus:text-gray-600;
    @apply rounded;

    &:focus-visible {
      @apply ring-2 ring-blue-500 ring-opacity-50;
    }
  }

  &__arrow {
    @apply inline-flex items-center justify-center;
    @apply text-gray-400 transition-transform duration-200;

    &--rotated {
      transform: rotate(180deg);
    }
  }

  &__dropdown {
    @apply z-50 bg-white border border-gray-300 rounded-md shadow-lg;
    @apply max-h-60 overflow-hidden;
  }

  &__loading {
    @apply flex items-center gap-2 p-3 text-gray-500;
  }

  &__options {
    @apply overflow-y-auto max-h-60;
  }

  &__option {
    @apply flex items-center gap-2 px-3 py-2;
    @apply text-gray-900 cursor-pointer;
    @apply hover:bg-gray-50 transition-colors duration-200;

    &--selected {
      @apply bg-blue-50 text-blue-900;
    }

    &--focused {
      @apply bg-gray-100;
    }

    &--disabled {
      @apply text-gray-400 cursor-not-allowed bg-transparent;
    }
  }

  &__option-checkbox {
    @apply flex items-center justify-center w-4 h-4;
    @apply border border-gray-300 rounded;

    .app-select__option--selected & {
      @apply bg-blue-600 border-blue-600 text-white;
    }
  }

  &__option-check {
    @apply w-3 h-3;
  }

  &__option-content {
    @apply flex-1 truncate;
  }

  &__empty {
    @apply px-3 py-2 text-gray-500 text-center;
  }
}

// 드롭다운 애니메이션
.app-select-dropdown-enter-active,
.app-select-dropdown-leave-active {
  @apply transition-all duration-200;
}

.app-select-dropdown-enter-from,
.app-select-dropdown-leave-to {
  @apply opacity-0 transform scale-95;
}

// 다크 모드
.dark .app-select {
  &__trigger {
    @apply bg-gray-800 border-gray-600;

    &:focus {
      @apply border-blue-400;
    }
  }

  &--disabled {
    .app-select__trigger {
      @apply bg-gray-900 border-gray-700;
    }
  }

  .app-select__tag {
    @apply bg-blue-900 text-blue-200 border-blue-800;
  }

  .app-select__tag-close {
    @apply text-blue-300 hover:text-blue-100;
  }

  .app-select__search-input,
  .app-select__display-value {
    @apply text-gray-100 placeholder-gray-400;
  }

  .app-select__clear,
  .app-select__arrow {
    @apply text-gray-500 hover:text-gray-300;
  }

  .app-select__dropdown {
    @apply bg-gray-800 border-gray-600;
  }

  .app-select__option {
    @apply text-gray-100 hover:bg-gray-700;

    &--selected {
      @apply bg-blue-900 text-blue-200;
    }

    &--focused {
      @apply bg-gray-700;
    }

    &--disabled {
      @apply text-gray-600;
    }
  }

  .app-select__option-checkbox {
    @apply border-gray-500;

    .app-select__option--selected & {
      @apply bg-blue-600 border-blue-600;
    }
  }

  .app-select__empty,
  .app-select__loading {
    @apply text-gray-400;
  }
}

// 접근성: 고대비 모드
@media (prefers-contrast: high) {
  .app-select {
    &__trigger {
      @apply border-2;

      &:focus {
        @apply ring-4;
      }
    }
  }
}

// 접근성: 움직임 감소
@media (prefers-reduced-motion: reduce) {
  .app-select {
    &__trigger,
    &__option,
    &__arrow,
    &__clear,
    &__tag-close {
      @apply transition-none;
    }
  }

  .app-select-dropdown-enter-active,
  .app-select-dropdown-leave-active {
    @apply transition-none;
  }
}
</style>