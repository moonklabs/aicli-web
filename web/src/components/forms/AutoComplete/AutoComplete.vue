<template>
  <div class="auto-complete" :class="{ 'auto-complete--disabled': disabled }">
    <div class="auto-complete__input-wrapper">
      <input
        ref="inputRef"
        v-model="inputValue"
        class="auto-complete__input"
        :class="{
          'auto-complete__input--focused': isFocused,
          'auto-complete__input--disabled': disabled
        }"
        :placeholder="placeholder"
        :disabled="disabled"
        @input="handleInput"
        @focus="handleFocus"
        @blur="handleBlur"
        @keydown="handleKeydown"
        :aria-autocomplete="'list'"
        :aria-expanded="showSuggestions"
        :aria-controls="suggestionsId"
        :aria-activedescendant="activeDescendant"
        :aria-describedby="ariaDescribedby"
        role="combobox"
      />

      <div v-if="loading" class="auto-complete__loader">
        <svg class="auto-complete__spinner" viewBox="0 0 24 24">
          <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="3" fill="none" />
        </svg>
      </div>

      <button
        v-if="clearable && inputValue && !disabled"
        @click="handleClear"
        class="auto-complete__clear"
        :aria-label="clearAriaLabel"
        tabindex="-1"
      >
        <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
          <path d="M8 1a7 7 0 1 0 0 14A7 7 0 0 0 8 1zM4.646 4.646a.5.5 0 0 1 .708 0L8 7.293l2.646-2.647a.5.5 0 0 1 .708.708L8.707 8l2.647 2.646a.5.5 0 0 1-.708.708L8 8.707l-2.646 2.647a.5.5 0 0 1-.708-.708L7.293 8 4.646 5.354a.5.5 0 0 1 0-.708z"/>
        </svg>
      </button>
    </div>

    <Transition name="auto-complete-suggestions">
      <div
        v-if="showSuggestions && filteredSuggestions.length > 0"
        :id="suggestionsId"
        class="auto-complete__suggestions"
        role="listbox"
      >
        <div
          v-for="(suggestion, index) in filteredSuggestions"
          :key="getSuggestionKey(suggestion, index)"
          :id="`${suggestionsId}-${index}`"
          class="auto-complete__suggestion"
          :class="{
            'auto-complete__suggestion--highlighted': highlightedIndex === index,
            'auto-complete__suggestion--selected': isSelected(suggestion)
          }"
          @click="selectSuggestion(suggestion)"
          @mouseenter="highlightedIndex = index"
          role="option"
          :aria-selected="isSelected(suggestion)"
        >
          <slot name="suggestion" :suggestion="suggestion" :highlighted="highlightedIndex === index">
            <span v-html="highlightMatch(getSuggestionLabel(suggestion))"></span>
          </slot>
        </div>

        <div v-if="showCreateOption && inputValue" class="auto-complete__create-option">
          <button
            @click="createNewItem"
            class="auto-complete__create-button"
          >
            <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
              <path d="M8 2a.5.5 0 0 1 .5.5v5h5a.5.5 0 0 1 0 1h-5v5a.5.5 0 0 1-1 0v-5h-5a.5.5 0 0 1 0-1h5v-5A.5.5 0 0 1 8 2z"/>
            </svg>
            <span>{{ createOptionText || `Create "${inputValue}"` }}</span>
          </button>
        </div>
      </div>

      <div
        v-else-if="showSuggestions && inputValue && !loading"
        class="auto-complete__empty"
      >
        <slot name="empty">
          <span>{{ emptyText }}</span>
        </slot>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import type { Ref } from 'vue'

interface Props {
  modelValue: any
  suggestions: any[]
  labelKey?: string | ((item: any) => string)
  valueKey?: string | ((item: any) => any)
  placeholder?: string
  disabled?: boolean
  clearable?: boolean
  loading?: boolean
  minLength?: number
  maxSuggestions?: number
  delay?: number
  showCreateOption?: boolean
  createOptionText?: string
  emptyText?: string
  clearAriaLabel?: string
  ariaDescribedby?: string
  caseSensitive?: boolean
  filterMethod?: (query: string, item: any) => boolean
  highlightMatches?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  labelKey: 'label',
  valueKey: 'value',
  placeholder: 'Type to search...',
  disabled: false,
  clearable: true,
  loading: false,
  minLength: 1,
  maxSuggestions: 10,
  delay: 300,
  showCreateOption: false,
  emptyText: 'No results found',
  clearAriaLabel: 'Clear selection',
  caseSensitive: false,
  highlightMatches: true,
})

const emit = defineEmits<{
  'update:modelValue': [value: any]
  'input': [query: string]
  'select': [item: any]
  'create': [query: string]
  'clear': []
  'focus': [event: FocusEvent]
  'blur': [event: FocusEvent]
}>()

// State
const inputValue = ref('')
const showSuggestions = ref(false)
const isFocused = ref(false)
const highlightedIndex = ref(-1)
const debounceTimer = ref<ReturnType<typeof setTimeout>>()

// Refs
const inputRef = ref<HTMLInputElement>()
const suggestionsId = `auto-complete-suggestions-${Math.random().toString(36).substr(2, 9)}`

// Computed
const activeDescendant = computed(() => {
  if (highlightedIndex.value >= 0 && showSuggestions.value) {
    return `${suggestionsId}-${highlightedIndex.value}`
  }
  return undefined
})

const filteredSuggestions = computed(() => {
  if (!inputValue.value || inputValue.value.length < props.minLength) {
    return []
  }

  let filtered = props.suggestions

  if (props.filterMethod) {
    filtered = filtered.filter(item => props.filterMethod!(inputValue.value, item))
  } else {
    const query = props.caseSensitive ? inputValue.value : inputValue.value.toLowerCase()
    filtered = filtered.filter(item => {
      const label = getSuggestionLabel(item)
      const searchLabel = props.caseSensitive ? label : label.toLowerCase()
      return searchLabel.includes(query)
    })
  }

  return filtered.slice(0, props.maxSuggestions)
})

// Methods
const getSuggestionLabel = (item: any): string => {
  if (typeof props.labelKey === 'function') {
    return props.labelKey(item)
  }
  return item[props.labelKey] || String(item)
}

const getSuggestionValue = (item: any): any => {
  if (typeof props.valueKey === 'function') {
    return props.valueKey(item)
  }
  return item[props.valueKey] || item
}

const getSuggestionKey = (item: any, index: number): string => {
  const value = getSuggestionValue(item)
  return typeof value === 'object' ? `suggestion-${index}` : String(value)
}

const isSelected = (item: any): boolean => {
  return getSuggestionValue(item) === props.modelValue
}

const highlightMatch = (text: string): string => {
  if (!props.highlightMatches || !inputValue.value) {
    return text
  }

  const query = inputValue.value
  const regex = new RegExp(`(${query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')})`, props.caseSensitive ? 'g' : 'gi')
  return text.replace(regex, '<mark>$1</mark>')
}

const handleInput = () => {
  showSuggestions.value = true
  highlightedIndex.value = -1

  // Debounce input events
  clearTimeout(debounceTimer.value)
  debounceTimer.value = setTimeout(() => {
    emit('input', inputValue.value)
  }, props.delay)
}

const handleFocus = (event: FocusEvent) => {
  isFocused.value = true
  if (inputValue.value.length >= props.minLength) {
    showSuggestions.value = true
  }
  emit('focus', event)
}

const handleBlur = (event: FocusEvent) => {
  isFocused.value = false
  // Delay hiding to allow click events on suggestions
  setTimeout(() => {
    showSuggestions.value = false
  }, 200)
  emit('blur', event)
}

const handleKeydown = (event: KeyboardEvent) => {
  switch (event.key) {
    case 'ArrowDown':
      event.preventDefault()
      if (!showSuggestions.value && inputValue.value.length >= props.minLength) {
        showSuggestions.value = true
      } else if (filteredSuggestions.value.length > 0) {
        highlightedIndex.value = Math.min(
          highlightedIndex.value + 1,
          filteredSuggestions.value.length - 1,
        )
      }
      break

    case 'ArrowUp':
      event.preventDefault()
      if (showSuggestions.value && highlightedIndex.value > 0) {
        highlightedIndex.value--
      }
      break

    case 'Enter':
      event.preventDefault()
      if (highlightedIndex.value >= 0 && filteredSuggestions.value[highlightedIndex.value]) {
        selectSuggestion(filteredSuggestions.value[highlightedIndex.value])
      } else if (props.showCreateOption && inputValue.value) {
        createNewItem()
      }
      break

    case 'Escape':
      event.preventDefault()
      showSuggestions.value = false
      highlightedIndex.value = -1
      break

    case 'Tab':
      if (showSuggestions.value) {
        showSuggestions.value = false
      }
      break
  }
}

const selectSuggestion = (item: any) => {
  const value = getSuggestionValue(item)
  const label = getSuggestionLabel(item)

  inputValue.value = label
  emit('update:modelValue', value)
  emit('select', item)

  showSuggestions.value = false
  highlightedIndex.value = -1
}

const handleClear = () => {
  inputValue.value = ''
  emit('update:modelValue', null)
  emit('clear')
  nextTick(() => {
    inputRef.value?.focus()
  })
}

const createNewItem = () => {
  emit('create', inputValue.value)
  showSuggestions.value = false
}

// Initialize input value from modelValue
const initializeInputValue = () => {
  if (props.modelValue && props.suggestions.length > 0) {
    const selected = props.suggestions.find(item =>
      getSuggestionValue(item) === props.modelValue,
    )
    if (selected) {
      inputValue.value = getSuggestionLabel(selected)
    }
  }
}

// Watchers
watch(() => props.modelValue, () => {
  initializeInputValue()
})

watch(() => props.suggestions, () => {
  if (props.modelValue) {
    initializeInputValue()
  }
})

// Lifecycle
onMounted(() => {
  initializeInputValue()
})

onUnmounted(() => {
  clearTimeout(debounceTimer.value)
})
</script>

<style lang="scss" scoped>
.auto-complete {
  @apply relative w-full;

  &--disabled {
    @apply opacity-60 cursor-not-allowed;
  }

  &__input-wrapper {
    @apply relative flex items-center;
  }

  &__input {
    @apply w-full px-3 py-2;
    @apply bg-white border border-gray-300 rounded-md;
    @apply transition-all duration-200;
    @apply focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-opacity-25;
    @apply focus:border-blue-500;

    &--focused {
      @apply border-blue-500;
    }

    &--disabled {
      @apply cursor-not-allowed bg-gray-50;
    }
  }

  &__loader {
    @apply absolute right-10 top-1/2 transform -translate-y-1/2;
  }

  &__spinner {
    @apply w-5 h-5 animate-spin;

    circle {
      stroke-dasharray: 62.83;
      stroke-dashoffset: 47.12;
      @apply animate-dash;
    }
  }

  &__clear {
    @apply absolute right-2 top-1/2 transform -translate-y-1/2;
    @apply p-1 rounded;
    @apply text-gray-400 hover:text-gray-600;
    @apply focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-opacity-25;
    @apply transition-colors duration-150;
  }

  &__suggestions {
    @apply absolute z-50 mt-1 w-full;
    @apply bg-white border border-gray-300 rounded-md shadow-lg;
    @apply max-h-60 overflow-y-auto;
  }

  &__suggestion {
    @apply px-3 py-2;
    @apply cursor-pointer;
    @apply transition-colors duration-150;

    &:hover {
      @apply bg-gray-50;
    }

    &--highlighted {
      @apply bg-gray-100;
    }

    &--selected {
      @apply bg-blue-50 text-blue-700;
    }

    :deep(mark) {
      @apply bg-yellow-200 text-inherit font-semibold;
    }
  }

  &__create-option {
    @apply border-t border-gray-200;
  }

  &__create-button {
    @apply w-full px-3 py-2;
    @apply flex items-center gap-2;
    @apply text-blue-600 hover:bg-blue-50;
    @apply transition-colors duration-150;
    @apply text-left;

    svg {
      @apply flex-shrink-0;
    }
  }

  &__empty {
    @apply absolute z-50 mt-1 w-full;
    @apply bg-white border border-gray-300 rounded-md shadow-lg;
    @apply px-3 py-4 text-center text-gray-500;
  }
}

// Dark mode
.dark .auto-complete {
  &__input {
    @apply bg-gray-800 border-gray-600 text-gray-100;

    &--disabled {
      @apply bg-gray-900;
    }
  }

  &__clear {
    @apply text-gray-500 hover:text-gray-300;
  }

  &__suggestions {
    @apply bg-gray-800 border-gray-600;
  }

  &__suggestion {
    @apply text-gray-100;

    &:hover {
      @apply bg-gray-700;
    }

    &--highlighted {
      @apply bg-gray-700;
    }

    &--selected {
      @apply bg-blue-900 text-blue-300;
    }

    :deep(mark) {
      @apply bg-yellow-700 text-yellow-100;
    }
  }

  &__create-option {
    @apply border-gray-700;
  }

  &__create-button {
    @apply text-blue-400 hover:bg-gray-700;
  }

  &__empty {
    @apply bg-gray-800 border-gray-600 text-gray-400;
  }
}

// Animations
.auto-complete-suggestions-enter-active,
.auto-complete-suggestions-leave-active {
  @apply transition-all duration-200;
}

.auto-complete-suggestions-enter-from,
.auto-complete-suggestions-leave-to {
  @apply opacity-0 transform -translate-y-2;
}

// Spinner animation
@keyframes dash {
  to {
    stroke-dashoffset: 0;
  }
}

@media (prefers-reduced-motion: reduce) {
  .auto-complete {
    &__spinner {
      @apply animate-none;
    }
  }

  .auto-complete-suggestions-enter-active,
  .auto-complete-suggestions-leave-active {
    @apply transition-none;
  }
}
</style>