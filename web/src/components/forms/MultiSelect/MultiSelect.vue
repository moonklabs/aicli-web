<template>
  <div class="multi-select" :class="{ 'multi-select--disabled': disabled }">
    <div 
      class="multi-select__trigger"
      :class="{ 
        'multi-select__trigger--focused': isFocused,
        'multi-select__trigger--disabled': disabled 
      }"
      @click="toggleDropdown"
      @keydown="handleTriggerKeydown"
      tabindex="0"
      role="combobox"
      :aria-expanded="isOpen"
      :aria-haspopup="listbox"
      :aria-labelledby="labelId"
      :aria-describedby="ariaDescribedby"
      :aria-disabled="disabled"
    >
      <div class="multi-select__values">
        <template v-if="selectedItems.length > 0">
          <span
            v-for="item in displayedItems"
            :key="getOptionValue(item)"
            class="multi-select__tag"
          >
            {{ getOptionLabel(item) }}
            <button
              v-if="!disabled"
              @click.stop="removeItem(item)"
              class="multi-select__tag-remove"
              :aria-label="`Remove ${getOptionLabel(item)}`"
              tabindex="-1"
            >
              ×
            </button>
          </span>
          <span v-if="hiddenCount > 0" class="multi-select__tag multi-select__tag--count">
            +{{ hiddenCount }}
          </span>
        </template>
        <span v-else class="multi-select__placeholder">
          {{ placeholder }}
        </span>
      </div>
      <div class="multi-select__arrow" :class="{ 'multi-select__arrow--open': isOpen }">
        <svg width="12" height="8" viewBox="0 0 12 8" fill="currentColor">
          <path d="M1 1.5L6 6.5L11 1.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" fill="none"/>
        </svg>
      </div>
    </div>

    <Transition name="multi-select-dropdown">
      <div
        v-if="isOpen"
        class="multi-select__dropdown"
        @keydown="handleDropdownKeydown"
      >
        <div v-if="searchable" class="multi-select__search">
          <input
            ref="searchInput"
            v-model="searchQuery"
            class="multi-select__search-input"
            placeholder="Search..."
            @input="handleSearch"
            @keydown.stop="handleSearchKeydown"
          />
        </div>

        <ul
          ref="optionsList"
          class="multi-select__options"
          role="listbox"
          :aria-multiselectable="true"
        >
          <li
            v-for="(option, index) in filteredOptions"
            :key="getOptionValue(option)"
            class="multi-select__option"
            :class="{
              'multi-select__option--selected': isSelected(option),
              'multi-select__option--highlighted': highlightedIndex === index,
              'multi-select__option--disabled': getOptionDisabled(option)
            }"
            @click="toggleOption(option)"
            @mouseenter="highlightedIndex = index"
            role="option"
            :aria-selected="isSelected(option)"
            :aria-disabled="getOptionDisabled(option)"
          >
            <span class="multi-select__option-checkbox">
              <svg v-if="isSelected(option)" width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
                <path d="M13.854 3.646a.5.5 0 0 1 0 .708l-7 7a.5.5 0 0 1-.708 0l-3.5-3.5a.5.5 0 1 1 .708-.708L6.5 10.293l6.646-6.647a.5.5 0 0 1 .708 0z"/>
              </svg>
            </span>
            <span class="multi-select__option-label">
              {{ getOptionLabel(option) }}
            </span>
          </li>
          <li v-if="filteredOptions.length === 0" class="multi-select__empty">
            {{ emptyText }}
          </li>
        </ul>

        <div v-if="showSelectAll && filteredOptions.length > 0" class="multi-select__actions">
          <button @click="selectAll" class="multi-select__action-button">
            Select All
          </button>
          <button @click="clearAll" class="multi-select__action-button">
            Clear All
          </button>
        </div>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'
import type { Ref } from 'vue'

interface Props {
  modelValue: any[]
  options: any[]
  labelKey?: string | ((option: any) => string)
  valueKey?: string | ((option: any) => any)
  disabledKey?: string | ((option: any) => boolean)
  placeholder?: string
  disabled?: boolean
  searchable?: boolean
  maxDisplay?: number
  showSelectAll?: boolean
  emptyText?: string
  labelId?: string
  ariaDescribedby?: string
}

const props = withDefaults(defineProps<Props>(), {
  labelKey: 'label',
  valueKey: 'value',
  placeholder: 'Select options...',
  disabled: false,
  searchable: true,
  maxDisplay: 3,
  showSelectAll: true,
  emptyText: 'No options available'
})

const emit = defineEmits<{
  'update:modelValue': [value: any[]]
  'change': [value: any[]]
  'search': [query: string]
}>()

// State
const isOpen = ref(false)
const isFocused = ref(false)
const searchQuery = ref('')
const highlightedIndex = ref(-1)

// Refs
const searchInput = ref<HTMLInputElement>()
const optionsList = ref<HTMLUListElement>()

// Computed
const selectedItems = computed(() => {
  return props.options.filter(option => 
    props.modelValue.includes(getOptionValue(option))
  )
})

const displayedItems = computed(() => {
  return selectedItems.value.slice(0, props.maxDisplay)
})

const hiddenCount = computed(() => {
  return Math.max(0, selectedItems.value.length - props.maxDisplay)
})

const filteredOptions = computed(() => {
  if (!searchQuery.value) return props.options

  const query = searchQuery.value.toLowerCase()
  return props.options.filter(option => {
    const label = getOptionLabel(option).toLowerCase()
    return label.includes(query)
  })
})

// Methods
const getOptionLabel = (option: any): string => {
  if (typeof props.labelKey === 'function') {
    return props.labelKey(option)
  }
  return option[props.labelKey] || String(option)
}

const getOptionValue = (option: any): any => {
  if (typeof props.valueKey === 'function') {
    return props.valueKey(option)
  }
  return option[props.valueKey] || option
}

const getOptionDisabled = (option: any): boolean => {
  if (typeof props.disabledKey === 'function') {
    return props.disabledKey(option)
  }
  if (typeof props.disabledKey === 'string') {
    return option[props.disabledKey] || false
  }
  return false
}

const isSelected = (option: any): boolean => {
  return props.modelValue.includes(getOptionValue(option))
}

const toggleDropdown = () => {
  if (props.disabled) return
  isOpen.value = !isOpen.value
  
  if (isOpen.value) {
    nextTick(() => {
      if (props.searchable && searchInput.value) {
        searchInput.value.focus()
      }
    })
  }
}

const toggleOption = (option: any) => {
  if (props.disabled || getOptionDisabled(option)) return

  const value = getOptionValue(option)
  const newValue = isSelected(option)
    ? props.modelValue.filter(v => v !== value)
    : [...props.modelValue, value]

  emit('update:modelValue', newValue)
  emit('change', newValue)
}

const removeItem = (option: any) => {
  const value = getOptionValue(option)
  const newValue = props.modelValue.filter(v => v !== value)
  emit('update:modelValue', newValue)
  emit('change', newValue)
}

const selectAll = () => {
  const allValues = filteredOptions.value
    .filter(option => !getOptionDisabled(option))
    .map(option => getOptionValue(option))
  
  emit('update:modelValue', allValues)
  emit('change', allValues)
}

const clearAll = () => {
  emit('update:modelValue', [])
  emit('change', [])
}

const handleSearch = () => {
  emit('search', searchQuery.value)
  highlightedIndex.value = 0
}

// Keyboard navigation
const handleTriggerKeydown = (event: KeyboardEvent) => {
  switch (event.key) {
    case 'Enter':
    case ' ':
      event.preventDefault()
      toggleDropdown()
      break
    case 'ArrowDown':
      event.preventDefault()
      if (!isOpen.value) {
        isOpen.value = true
      } else {
        highlightedIndex.value = Math.min(highlightedIndex.value + 1, filteredOptions.value.length - 1)
      }
      break
    case 'ArrowUp':
      event.preventDefault()
      if (isOpen.value) {
        highlightedIndex.value = Math.max(highlightedIndex.value - 1, 0)
      }
      break
    case 'Escape':
      event.preventDefault()
      isOpen.value = false
      break
  }
}

const handleDropdownKeydown = (event: KeyboardEvent) => {
  switch (event.key) {
    case 'Enter':
    case ' ':
      event.preventDefault()
      if (highlightedIndex.value >= 0) {
        const option = filteredOptions.value[highlightedIndex.value]
        if (option && !getOptionDisabled(option)) {
          toggleOption(option)
        }
      }
      break
  }
}

const handleSearchKeydown = (event: KeyboardEvent) => {
  if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
    event.preventDefault()
    handleTriggerKeydown(event)
  }
}

// Click outside handler
const handleClickOutside = (event: MouseEvent) => {
  const target = event.target as HTMLElement
  const element = target.closest('.multi-select')
  if (!element) {
    isOpen.value = false
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})

// Reset search when dropdown closes
watch(isOpen, (newValue) => {
  if (!newValue) {
    searchQuery.value = ''
    highlightedIndex.value = -1
  }
})
</script>

<style lang="scss" scoped>
.multi-select {
  @apply relative w-full;

  &--disabled {
    @apply opacity-60 cursor-not-allowed;
  }

  &__trigger {
    @apply flex items-center justify-between;
    @apply w-full px-3 py-2;
    @apply bg-white border border-gray-300 rounded-md;
    @apply cursor-pointer;
    @apply transition-all duration-200;

    &:hover:not(&--disabled) {
      @apply border-gray-400;
    }

    &--focused {
      @apply outline-none ring-2 ring-blue-500 ring-opacity-25;
      @apply border-blue-500;
    }

    &--disabled {
      @apply cursor-not-allowed bg-gray-50;
    }
  }

  &__values {
    @apply flex-1 flex flex-wrap gap-1;
    @apply min-h-[24px];
  }

  &__tag {
    @apply inline-flex items-center gap-1;
    @apply px-2 py-0.5;
    @apply bg-blue-100 text-blue-800 text-sm;
    @apply rounded;

    &--count {
      @apply bg-gray-100 text-gray-700;
    }
  }

  &__tag-remove {
    @apply ml-1 text-blue-600 hover:text-blue-800;
    @apply focus:outline-none;
  }

  &__placeholder {
    @apply text-gray-400;
  }

  &__arrow {
    @apply ml-2 transition-transform duration-200;

    &--open {
      @apply transform rotate-180;
    }
  }

  &__dropdown {
    @apply absolute z-50 mt-1 w-full;
    @apply bg-white border border-gray-300 rounded-md shadow-lg;
    @apply overflow-hidden;
  }

  &__search {
    @apply p-2 border-b border-gray-200;
  }

  &__search-input {
    @apply w-full px-3 py-1.5;
    @apply border border-gray-300 rounded;
    @apply focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-opacity-25;
    @apply focus:border-blue-500;
  }

  &__options {
    @apply max-h-60 overflow-y-auto;
  }

  &__option {
    @apply flex items-center gap-2;
    @apply px-3 py-2;
    @apply cursor-pointer;
    @apply transition-colors duration-150;

    &:hover:not(&--disabled) {
      @apply bg-gray-50;
    }

    &--highlighted {
      @apply bg-gray-100;
    }

    &--selected {
      @apply bg-blue-50;
    }

    &--disabled {
      @apply opacity-50 cursor-not-allowed;
    }
  }

  &__option-checkbox {
    @apply flex items-center justify-center;
    @apply w-4 h-4;
    @apply border border-gray-300 rounded;
    @apply bg-white;

    svg {
      @apply text-blue-600;
    }
  }

  &__option-label {
    @apply flex-1;
  }

  &__empty {
    @apply px-3 py-4 text-center text-gray-500;
  }

  &__actions {
    @apply flex gap-2 p-2;
    @apply border-t border-gray-200;
  }

  &__action-button {
    @apply flex-1 px-3 py-1.5;
    @apply text-sm text-blue-600;
    @apply bg-white border border-blue-600 rounded;
    @apply hover:bg-blue-50;
    @apply focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-opacity-25;
    @apply transition-colors duration-150;
  }
}

// Dark mode
.dark .multi-select {
  &__trigger {
    @apply bg-gray-800 border-gray-600;

    &:hover:not(&--disabled) {
      @apply border-gray-500;
    }

    &--focused {
      @apply ring-blue-500 ring-opacity-25;
      @apply border-blue-500;
    }

    &--disabled {
      @apply bg-gray-900;
    }
  }

  &__tag {
    @apply bg-blue-900 text-blue-200;

    &--count {
      @apply bg-gray-700 text-gray-300;
    }
  }

  &__tag-remove {
    @apply text-blue-400 hover:text-blue-300;
  }

  &__placeholder {
    @apply text-gray-500;
  }

  &__dropdown {
    @apply bg-gray-800 border-gray-600;
  }

  &__search {
    @apply border-gray-700;
  }

  &__search-input {
    @apply bg-gray-700 border-gray-600 text-gray-100;
    @apply focus:border-blue-500;
  }

  &__option {
    &:hover:not(&--disabled) {
      @apply bg-gray-700;
    }

    &--highlighted {
      @apply bg-gray-700;
    }

    &--selected {
      @apply bg-blue-900;
    }
  }

  &__option-checkbox {
    @apply border-gray-600 bg-gray-700;
  }

  &__empty {
    @apply text-gray-400;
  }

  &__actions {
    @apply border-gray-700;
  }

  &__action-button {
    @apply text-blue-400 bg-gray-800 border-blue-400;
    @apply hover:bg-gray-700;
  }
}

// Animations
.multi-select-dropdown-enter-active,
.multi-select-dropdown-leave-active {
  @apply transition-all duration-200;
}

.multi-select-dropdown-enter-from,
.multi-select-dropdown-leave-to {
  @apply opacity-0 transform -translate-y-2;
}
</style>