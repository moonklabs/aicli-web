<template>
  <div class="time-picker" :class="{ 'time-picker--disabled': disabled }">
    <div class="time-picker__input-wrapper">
      <input
        ref="inputRef"
        v-model="displayValue"
        class="time-picker__input"
        :class="{
          'time-picker__input--focused': isOpen,
          'time-picker__input--disabled': disabled
        }"
        :placeholder="placeholder"
        :disabled="disabled"
        @click="togglePicker"
        @keydown="handleInputKeydown"
        :aria-label="ariaLabel"
        :aria-describedby="ariaDescribedby"
        :aria-expanded="isOpen"
        :aria-haspopup="true"
        role="combobox"
        readonly
      />
      
      <button
        v-if="clearable && modelValue && !disabled"
        @click="handleClear"
        class="time-picker__clear"
        :aria-label="clearAriaLabel"
        tabindex="-1"
      >
        <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
          <path d="M8 1a7 7 0 1 0 0 14A7 7 0 0 0 8 1zM4.646 4.646a.5.5 0 0 1 .708 0L8 7.293l2.646-2.647a.5.5 0 0 1 .708.708L8.707 8l2.647 2.646a.5.5 0 0 1-.708.708L8 8.707l-2.646 2.647a.5.5 0 0 1-.708-.708L7.293 8 4.646 5.354a.5.5 0 0 1 0-.708z"/>
        </svg>
      </button>
      
      <div class="time-picker__icon">
        <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
          <path d="M10 2a8 8 0 100 16 8 8 0 000-16zM10 4a6 6 0 110 12 6 6 0 010-12z"/>
          <path d="M10 5a.5.5 0 01.5.5V10h3a.5.5 0 010 1h-3.5a.5.5 0 01-.5-.5v-5A.5.5 0 0110 5z"/>
        </svg>
      </div>
    </div>

    <Transition name="time-picker-dropdown">
      <div
        v-if="isOpen"
        ref="pickerRef"
        class="time-picker__dropdown"
        @keydown="handlePickerKeydown"
        role="dialog"
        aria-modal="true"
        :aria-label="pickerAriaLabel"
      >
        <div class="time-picker__content">
          <!-- Time Display -->
          <div class="time-picker__display">
            <button
              class="time-picker__display-button"
              :class="{ 'time-picker__display-button--active': activeSection === 'hour' }"
              @click="activeSection = 'hour'"
              :aria-label="`Hour: ${displayHour}`"
            >
              {{ displayHour }}
            </button>
            <span class="time-picker__separator">:</span>
            <button
              class="time-picker__display-button"
              :class="{ 'time-picker__display-button--active': activeSection === 'minute' }"
              @click="activeSection = 'minute'"
              :aria-label="`Minute: ${displayMinute}`"
            >
              {{ displayMinute }}
            </button>
            <button
              v-if="!use24Hour"
              class="time-picker__display-button time-picker__display-button--period"
              :class="{ 'time-picker__display-button--active': activeSection === 'period' }"
              @click="activeSection = 'period'"
              :aria-label="`Period: ${period}`"
            >
              {{ period }}
            </button>
          </div>

          <!-- Selection Area -->
          <div class="time-picker__selection">
            <!-- Hour Selection -->
            <div
              v-show="activeSection === 'hour'"
              class="time-picker__grid"
              role="grid"
              :aria-label="hourGridAriaLabel"
            >
              <button
                v-for="h in hourOptions"
                :key="h"
                class="time-picker__grid-button"
                :class="{
                  'time-picker__grid-button--selected': h === hour,
                  'time-picker__grid-button--disabled': isHourDisabled(h)
                }"
                :disabled="isHourDisabled(h)"
                @click="selectHour(h)"
                :aria-label="`${h} hours`"
                :aria-selected="h === hour"
                role="gridcell"
              >
                {{ formatNumber(h) }}
              </button>
            </div>

            <!-- Minute Selection -->
            <div
              v-show="activeSection === 'minute'"
              class="time-picker__grid"
              role="grid"
              :aria-label="minuteGridAriaLabel"
            >
              <button
                v-for="m in minuteOptions"
                :key="m"
                class="time-picker__grid-button"
                :class="{
                  'time-picker__grid-button--selected': m === minute,
                  'time-picker__grid-button--disabled': isMinuteDisabled(m)
                }"
                :disabled="isMinuteDisabled(m)"
                @click="selectMinute(m)"
                :aria-label="`${m} minutes`"
                :aria-selected="m === minute"
                role="gridcell"
              >
                {{ formatNumber(m) }}
              </button>
            </div>

            <!-- Period Selection -->
            <div
              v-if="!use24Hour"
              v-show="activeSection === 'period'"
              class="time-picker__period"
            >
              <button
                class="time-picker__period-button"
                :class="{ 'time-picker__period-button--selected': period === 'AM' }"
                @click="selectPeriod('AM')"
                :aria-pressed="period === 'AM'"
              >
                AM
              </button>
              <button
                class="time-picker__period-button"
                :class="{ 'time-picker__period-button--selected': period === 'PM' }"
                @click="selectPeriod('PM')"
                :aria-pressed="period === 'PM'"
              >
                PM
              </button>
            </div>
          </div>

          <!-- Quick Actions -->
          <div class="time-picker__actions">
            <button
              @click="selectNow"
              class="time-picker__action-button"
              :disabled="isNowDisabled"
            >
              {{ nowButtonText }}
            </button>
            <button
              @click="handleConfirm"
              class="time-picker__action-button time-picker__action-button--primary"
            >
              {{ confirmButtonText }}
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'

interface Props {
  modelValue: string | null // HH:mm format
  min?: string
  max?: string
  disabled?: boolean
  clearable?: boolean
  placeholder?: string
  use24Hour?: boolean
  minuteStep?: number
  ariaLabel?: string
  ariaDescribedby?: string
  clearAriaLabel?: string
  pickerAriaLabel?: string
  hourGridAriaLabel?: string
  minuteGridAriaLabel?: string
  nowButtonText?: string
  confirmButtonText?: string
}

const props = withDefaults(defineProps<Props>(), {
  disabled: false,
  clearable: true,
  placeholder: 'Select time',
  use24Hour: false,
  minuteStep: 1,
  clearAriaLabel: 'Clear time',
  pickerAriaLabel: 'Time picker',
  hourGridAriaLabel: 'Select hour',
  minuteGridAriaLabel: 'Select minute',
  nowButtonText: '지금',
  confirmButtonText: '확인'
})

const emit = defineEmits<{
  'update:modelValue': [value: string | null]
  'change': [value: string | null]
  'clear': []
}>()

// State
const isOpen = ref(false)
const activeSection = ref<'hour' | 'minute' | 'period'>('hour')
const hour = ref(0)
const minute = ref(0)
const period = ref<'AM' | 'PM'>('AM')

// Refs
const inputRef = ref<HTMLInputElement>()
const pickerRef = ref<HTMLDivElement>()

// Computed
const hourOptions = computed(() => {
  if (props.use24Hour) {
    return Array.from({ length: 24 }, (_, i) => i)
  }
  return Array.from({ length: 12 }, (_, i) => i || 12)
})

const minuteOptions = computed(() => {
  const options = []
  for (let i = 0; i < 60; i += props.minuteStep) {
    options.push(i)
  }
  return options
})

const displayHour = computed(() => {
  return formatNumber(props.use24Hour ? hour.value : (hour.value % 12 || 12))
})

const displayMinute = computed(() => {
  return formatNumber(minute.value)
})

const displayValue = computed(() => {
  if (!props.modelValue) return ''
  
  const [h, m] = parseTime(props.modelValue)
  if (props.use24Hour) {
    return `${formatNumber(h)}:${formatNumber(m)}`
  } else {
    const displayH = h % 12 || 12
    const p = h >= 12 ? 'PM' : 'AM'
    return `${formatNumber(displayH)}:${formatNumber(m)} ${p}`
  }
})

const isNowDisabled = computed(() => {
  const now = new Date()
  const nowHour = now.getHours()
  const nowMinute = now.getMinutes()
  const nowTime = `${formatNumber(nowHour)}:${formatNumber(nowMinute)}`
  
  if (props.min && nowTime < props.min) return true
  if (props.max && nowTime > props.max) return true
  return false
})

// Methods
const formatNumber = (num: number): string => {
  return num.toString().padStart(2, '0')
}

const parseTime = (time: string): [number, number] => {
  const [h, m] = time.split(':').map(Number)
  return [h || 0, m || 0]
}

const formatTime = (h: number, m: number): string => {
  return `${formatNumber(h)}:${formatNumber(m)}`
}

const isHourDisabled = (h: number): boolean => {
  if (!props.min && !props.max) return false
  
  let actualHour = h
  if (!props.use24Hour) {
    if (period.value === 'PM' && h !== 12) actualHour = h + 12
    if (period.value === 'AM' && h === 12) actualHour = 0
  }
  
  const testTime = formatTime(actualHour, minute.value)
  
  if (props.min && testTime < props.min) return true
  if (props.max && testTime > props.max) return true
  
  return false
}

const isMinuteDisabled = (m: number): boolean => {
  if (!props.min && !props.max) return false
  
  let actualHour = hour.value
  if (!props.use24Hour) {
    if (period.value === 'PM' && hour.value !== 12) actualHour = hour.value + 12
    if (period.value === 'AM' && hour.value === 12) actualHour = 0
  }
  
  const testTime = formatTime(actualHour, m)
  
  if (props.min && testTime < props.min) return true
  if (props.max && testTime > props.max) return true
  
  return false
}

const togglePicker = () => {
  if (props.disabled) return
  isOpen.value = !isOpen.value
  
  if (isOpen.value) {
    if (props.modelValue) {
      const [h, m] = parseTime(props.modelValue)
      hour.value = props.use24Hour ? h : (h % 12 || 12)
      minute.value = m
      period.value = h >= 12 ? 'PM' : 'AM'
    } else {
      const now = new Date()
      hour.value = props.use24Hour ? now.getHours() : (now.getHours() % 12 || 12)
      minute.value = Math.floor(now.getMinutes() / props.minuteStep) * props.minuteStep
      period.value = now.getHours() >= 12 ? 'PM' : 'AM'
    }
    
    activeSection.value = 'hour'
    
    nextTick(() => {
      pickerRef.value?.focus()
    })
  }
}

const selectHour = (h: number) => {
  if (isHourDisabled(h)) return
  hour.value = h
  activeSection.value = 'minute'
}

const selectMinute = (m: number) => {
  if (isMinuteDisabled(m)) return
  minute.value = m
  if (!props.use24Hour) {
    activeSection.value = 'period'
  }
}

const selectPeriod = (p: 'AM' | 'PM') => {
  period.value = p
}

const selectNow = () => {
  const now = new Date()
  hour.value = props.use24Hour ? now.getHours() : (now.getHours() % 12 || 12)
  minute.value = Math.floor(now.getMinutes() / props.minuteStep) * props.minuteStep
  period.value = now.getHours() >= 12 ? 'PM' : 'AM'
}

const handleConfirm = () => {
  let actualHour = hour.value
  if (!props.use24Hour) {
    if (period.value === 'PM' && hour.value !== 12) actualHour = hour.value + 12
    if (period.value === 'AM' && hour.value === 12) actualHour = 0
  }
  
  const value = formatTime(actualHour, minute.value)
  emit('update:modelValue', value)
  emit('change', value)
  isOpen.value = false
  
  nextTick(() => {
    inputRef.value?.focus()
  })
}

const handleClear = () => {
  emit('update:modelValue', null)
  emit('clear')
  nextTick(() => {
    inputRef.value?.focus()
  })
}

const handleInputKeydown = (event: KeyboardEvent) => {
  switch (event.key) {
    case 'Enter':
    case ' ':
    case 'ArrowDown':
      event.preventDefault()
      togglePicker()
      break
    case 'Escape':
      event.preventDefault()
      if (isOpen.value) {
        isOpen.value = false
      }
      break
  }
}

const handlePickerKeydown = (event: KeyboardEvent) => {
  switch (event.key) {
    case 'Escape':
      event.preventDefault()
      isOpen.value = false
      nextTick(() => {
        inputRef.value?.focus()
      })
      break
    case 'Tab':
      // Allow tab navigation within picker
      break
    case 'Enter':
      if (event.target instanceof HTMLButtonElement) {
        // Let button handle its own click
      } else {
        event.preventDefault()
        handleConfirm()
      }
      break
  }
}

// Click outside handler
const handleClickOutside = (event: MouseEvent) => {
  const target = event.target as HTMLElement
  const element = target.closest('.time-picker')
  if (!element) {
    isOpen.value = false
  }
}

// Lifecycle
onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>

<style lang="scss" scoped>
.time-picker {
  @apply relative w-full;

  &--disabled {
    @apply opacity-60 cursor-not-allowed;
  }

  &__input-wrapper {
    @apply relative flex items-center;
  }

  &__input {
    @apply w-full px-3 py-2 pr-10;
    @apply bg-white border border-gray-300 rounded-md;
    @apply cursor-pointer;
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

  &__clear {
    @apply absolute right-8 top-1/2 transform -translate-y-1/2;
    @apply p-1 rounded;
    @apply text-gray-400 hover:text-gray-600;
    @apply focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-opacity-25;
    @apply transition-colors duration-150;
  }

  &__icon {
    @apply absolute right-2 top-1/2 transform -translate-y-1/2;
    @apply text-gray-400 pointer-events-none;
  }

  &__dropdown {
    @apply absolute z-50 mt-1;
    @apply bg-white border border-gray-300 rounded-md shadow-lg;
    @apply focus:outline-none;
  }

  &__content {
    @apply p-4;
  }

  &__display {
    @apply flex items-center justify-center mb-4;
    @apply text-2xl font-medium;
  }

  &__display-button {
    @apply px-2 py-1 rounded;
    @apply transition-colors duration-150;
    @apply focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-opacity-25;

    &:hover {
      @apply bg-gray-100;
    }

    &--active {
      @apply bg-blue-100 text-blue-700;
    }

    &--period {
      @apply ml-2 text-lg;
    }
  }

  &__separator {
    @apply mx-1;
  }

  &__selection {
    @apply mb-4;
  }

  &__grid {
    @apply grid grid-cols-4 gap-1;
  }

  &__grid-button {
    @apply w-full px-2 py-2;
    @apply text-sm;
    @apply rounded;
    @apply transition-colors duration-150;
    @apply focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-opacity-25;

    &:hover:not(:disabled) {
      @apply bg-gray-100;
    }

    &--selected {
      @apply bg-blue-500 text-white;

      &:hover {
        @apply bg-blue-600;
      }
    }

    &--disabled {
      @apply text-gray-300 cursor-not-allowed;
      
      &:hover {
        @apply bg-transparent;
      }
    }
  }

  &__period {
    @apply flex gap-2;
  }

  &__period-button {
    @apply flex-1 px-4 py-2;
    @apply border border-gray-300 rounded;
    @apply transition-colors duration-150;
    @apply focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-opacity-25;

    &:hover {
      @apply bg-gray-50;
    }

    &--selected {
      @apply bg-blue-500 text-white border-blue-500;

      &:hover {
        @apply bg-blue-600;
      }
    }
  }

  &__actions {
    @apply flex gap-2;
  }

  &__action-button {
    @apply flex-1 px-3 py-2;
    @apply text-sm font-medium;
    @apply border rounded-md;
    @apply focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-opacity-25;
    @apply transition-colors duration-150;
    
    @apply bg-white text-gray-700 border-gray-300;
    @apply hover:bg-gray-50;

    &--primary {
      @apply bg-blue-600 text-white border-blue-600;
      @apply hover:bg-blue-700;
    }

    &:disabled {
      @apply opacity-50 cursor-not-allowed;
      @apply hover:bg-white;
    }
  }
}

// Dark mode
.dark .time-picker {
  &__input {
    @apply bg-gray-800 border-gray-600 text-gray-100;

    &--disabled {
      @apply bg-gray-900;
    }
  }

  &__clear {
    @apply text-gray-500 hover:text-gray-300;
  }

  &__icon {
    @apply text-gray-500;
  }

  &__dropdown {
    @apply bg-gray-800 border-gray-600;
  }

  &__display {
    @apply text-gray-100;
  }

  &__display-button {
    &:hover {
      @apply bg-gray-700;
    }

    &--active {
      @apply bg-blue-900 text-blue-300;
    }
  }

  &__grid-button {
    @apply text-gray-100;

    &:hover:not(:disabled) {
      @apply bg-gray-700;
    }

    &--selected {
      @apply bg-blue-600 text-white;

      &:hover {
        @apply bg-blue-700;
      }
    }

    &--disabled {
      @apply text-gray-600;
    }
  }

  &__period-button {
    @apply bg-gray-700 border-gray-600 text-gray-100;

    &:hover {
      @apply bg-gray-600;
    }

    &--selected {
      @apply bg-blue-600 text-white border-blue-600;

      &:hover {
        @apply bg-blue-700;
      }
    }
  }

  &__action-button {
    @apply bg-gray-700 text-gray-100 border-gray-600;
    @apply hover:bg-gray-600;

    &--primary {
      @apply bg-blue-600 text-white border-blue-600;
      @apply hover:bg-blue-700;
    }
  }
}

// Animations
.time-picker-dropdown-enter-active,
.time-picker-dropdown-leave-active {
  @apply transition-all duration-200;
}

.time-picker-dropdown-enter-from,
.time-picker-dropdown-leave-to {
  @apply opacity-0 transform scale-95;
}
</style>