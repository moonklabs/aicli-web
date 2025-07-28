<template>
  <div class="date-picker" :class="{ 'date-picker--disabled': disabled }">
    <div class="date-picker__input-wrapper">
      <input
        ref="inputRef"
        v-model="displayValue"
        class="date-picker__input"
        :class="{
          'date-picker__input--focused': isOpen,
          'date-picker__input--disabled': disabled
        }"
        :placeholder="placeholder"
        :disabled="disabled"
        @click="toggleCalendar"
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
        class="date-picker__clear"
        :aria-label="clearAriaLabel"
        tabindex="-1"
      >
        <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
          <path d="M8 1a7 7 0 1 0 0 14A7 7 0 0 0 8 1zM4.646 4.646a.5.5 0 0 1 .708 0L8 7.293l2.646-2.647a.5.5 0 0 1 .708.708L8.707 8l2.647 2.646a.5.5 0 0 1-.708.708L8 8.707l-2.646 2.647a.5.5 0 0 1-.708-.708L7.293 8 4.646 5.354a.5.5 0 0 1 0-.708z"/>
        </svg>
      </button>
      
      <div class="date-picker__icon">
        <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
          <path d="M6 2a1 1 0 00-2 0v1H3a2 2 0 00-2 2v11a2 2 0 002 2h14a2 2 0 002-2V5a2 2 0 00-2-2h-1V2a1 1 0 10-2 0v1H6V2zM3 7h14v9H3V7z"/>
          <path d="M6 10a1 1 0 100 2 1 1 0 000-2zM10 10a1 1 0 100 2 1 1 0 000-2zM14 10a1 1 0 100 2 1 1 0 000-2z"/>
        </svg>
      </div>
    </div>

    <Transition name="date-picker-calendar">
      <div
        v-if="isOpen"
        ref="calendarRef"
        class="date-picker__calendar"
        @keydown="handleCalendarKeydown"
        role="dialog"
        aria-modal="true"
        :aria-label="calendarAriaLabel"
      >
        <!-- Calendar Header -->
        <div class="date-picker__header">
          <button
            @click="previousMonth"
            class="date-picker__nav-button"
            :aria-label="previousMonthAriaLabel"
          >
            <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
              <path d="M11.354 1.646a.5.5 0 0 1 0 .708L5.707 8l5.647 5.646a.5.5 0 0 1-.708.708l-6-6a.5.5 0 0 1 0-.708l6-6a.5.5 0 0 1 .708 0z"/>
            </svg>
          </button>
          
          <div class="date-picker__month-year">
            <select
              v-model="viewMonth"
              class="date-picker__select"
              :aria-label="monthSelectAriaLabel"
            >
              <option
                v-for="(month, index) in monthNames"
                :key="index"
                :value="index"
              >
                {{ month }}
              </option>
            </select>
            
            <select
              v-model="viewYear"
              class="date-picker__select"
              :aria-label="yearSelectAriaLabel"
            >
              <option
                v-for="year in yearOptions"
                :key="year"
                :value="year"
              >
                {{ year }}
              </option>
            </select>
          </div>
          
          <button
            @click="nextMonth"
            class="date-picker__nav-button"
            :aria-label="nextMonthAriaLabel"
          >
            <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
              <path d="M4.646 1.646a.5.5 0 0 1 .708 0l6 6a.5.5 0 0 1 0 .708l-6 6a.5.5 0 0 1-.708-.708L10.293 8 4.646 2.354a.5.5 0 0 1 0-.708z"/>
            </svg>
          </button>
        </div>

        <!-- Weekday Headers -->
        <div class="date-picker__weekdays">
          <div
            v-for="(day, index) in weekdayNames"
            :key="index"
            class="date-picker__weekday"
            role="columnheader"
          >
            {{ day }}
          </div>
        </div>

        <!-- Calendar Grid -->
        <div
          class="date-picker__grid"
          role="grid"
          :aria-label="gridAriaLabel"
        >
          <div
            v-for="(week, weekIndex) in calendarDays"
            :key="weekIndex"
            class="date-picker__week"
            role="row"
          >
            <button
              v-for="(day, dayIndex) in week"
              :key="`${weekIndex}-${dayIndex}`"
              class="date-picker__day"
              :class="{
                'date-picker__day--other-month': day.otherMonth,
                'date-picker__day--today': isToday(day),
                'date-picker__day--selected': isSelected(day),
                'date-picker__day--disabled': isDisabled(day),
                'date-picker__day--focused': isFocused(day)
              }"
              :disabled="isDisabled(day)"
              @click="selectDate(day)"
              @mouseenter="hoveredDate = day.date"
              @mouseleave="hoveredDate = null"
              :aria-label="getDayAriaLabel(day)"
              :aria-selected="isSelected(day)"
              role="gridcell"
              :tabindex="isFocused(day) ? 0 : -1"
            >
              {{ day.day }}
            </button>
          </div>
        </div>

        <!-- Calendar Footer -->
        <div class="date-picker__footer">
          <button
            @click="selectToday"
            class="date-picker__today-button"
            :disabled="isTodayDisabled"
          >
            {{ todayButtonText }}
          </button>
        </div>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import type { Ref } from 'vue'

interface CalendarDay {
  date: Date
  day: number
  month: number
  year: number
  otherMonth: boolean
}

interface Props {
  modelValue: Date | null
  min?: Date
  max?: Date
  disabled?: boolean
  clearable?: boolean
  placeholder?: string
  format?: string
  locale?: string
  firstDayOfWeek?: number // 0 = Sunday, 1 = Monday, etc.
  ariaLabel?: string
  ariaDescribedby?: string
  clearAriaLabel?: string
  calendarAriaLabel?: string
  previousMonthAriaLabel?: string
  nextMonthAriaLabel?: string
  monthSelectAriaLabel?: string
  yearSelectAriaLabel?: string
  gridAriaLabel?: string
  todayButtonText?: string
}

const props = withDefaults(defineProps<Props>(), {
  disabled: false,
  clearable: true,
  placeholder: 'Select date',
  format: 'YYYY-MM-DD',
  locale: 'ko-KR',
  firstDayOfWeek: 0,
  clearAriaLabel: 'Clear date',
  calendarAriaLabel: 'Calendar',
  previousMonthAriaLabel: 'Previous month',
  nextMonthAriaLabel: 'Next month',
  monthSelectAriaLabel: 'Select month',
  yearSelectAriaLabel: 'Select year',
  gridAriaLabel: 'Calendar dates',
  todayButtonText: '오늘'
})

const emit = defineEmits<{
  'update:modelValue': [value: Date | null]
  'change': [value: Date | null]
  'clear': []
}>()

// State
const isOpen = ref(false)
const viewMonth = ref(0)
const viewYear = ref(new Date().getFullYear())
const focusedDate = ref<Date | null>(null)
const hoveredDate = ref<Date | null>(null)

// Refs
const inputRef = ref<HTMLInputElement>()
const calendarRef = ref<HTMLDivElement>()

// Computed
const monthNames = computed(() => {
  const formatter = new Intl.DateTimeFormat(props.locale, { month: 'long' })
  return Array.from({ length: 12 }, (_, i) => {
    const date = new Date(2000, i, 1)
    return formatter.format(date)
  })
})

const weekdayNames = computed(() => {
  const formatter = new Intl.DateTimeFormat(props.locale, { weekday: 'short' })
  const days = []
  for (let i = 0; i < 7; i++) {
    const date = new Date(2023, 0, 1 + i + props.firstDayOfWeek)
    days.push(formatter.format(date))
  }
  return days
})

const yearOptions = computed(() => {
  const currentYear = new Date().getFullYear()
  const startYear = props.min ? props.min.getFullYear() : currentYear - 100
  const endYear = props.max ? props.max.getFullYear() : currentYear + 100
  const years = []
  for (let year = startYear; year <= endYear; year++) {
    years.push(year)
  }
  return years
})

const displayValue = computed(() => {
  if (!props.modelValue) return ''
  return formatDate(props.modelValue)
})

const calendarDays = computed((): CalendarDay[][] => {
  const year = viewYear.value
  const month = viewMonth.value
  
  const firstDay = new Date(year, month, 1)
  const lastDay = new Date(year, month + 1, 0)
  
  const startDate = new Date(firstDay)
  const startOffset = (startDate.getDay() - props.firstDayOfWeek + 7) % 7
  startDate.setDate(startDate.getDate() - startOffset)
  
  const weeks: CalendarDay[][] = []
  const currentDate = new Date(startDate)
  
  for (let week = 0; week < 6; week++) {
    const days: CalendarDay[] = []
    for (let day = 0; day < 7; day++) {
      days.push({
        date: new Date(currentDate),
        day: currentDate.getDate(),
        month: currentDate.getMonth(),
        year: currentDate.getFullYear(),
        otherMonth: currentDate.getMonth() !== month
      })
      currentDate.setDate(currentDate.getDate() + 1)
    }
    weeks.push(days)
  }
  
  return weeks
})

const isTodayDisabled = computed(() => {
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  return isDateDisabled(today)
})

// Methods
const formatDate = (date: Date): string => {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  
  return props.format
    .replace('YYYY', String(year))
    .replace('MM', month)
    .replace('DD', day)
}

const isToday = (day: CalendarDay): boolean => {
  const today = new Date()
  return (
    day.date.getDate() === today.getDate() &&
    day.date.getMonth() === today.getMonth() &&
    day.date.getFullYear() === today.getFullYear()
  )
}

const isSelected = (day: CalendarDay): boolean => {
  if (!props.modelValue) return false
  return (
    day.date.getDate() === props.modelValue.getDate() &&
    day.date.getMonth() === props.modelValue.getMonth() &&
    day.date.getFullYear() === props.modelValue.getFullYear()
  )
}

const isFocused = (day: CalendarDay): boolean => {
  if (!focusedDate.value) return false
  return (
    day.date.getDate() === focusedDate.value.getDate() &&
    day.date.getMonth() === focusedDate.value.getMonth() &&
    day.date.getFullYear() === focusedDate.value.getFullYear()
  )
}

const isDateDisabled = (date: Date): boolean => {
  if (props.min && date < props.min) return true
  if (props.max && date > props.max) return true
  return false
}

const isDisabled = (day: CalendarDay): boolean => {
  return day.otherMonth || isDateDisabled(day.date)
}

const getDayAriaLabel = (day: CalendarDay): string => {
  const formatter = new Intl.DateTimeFormat(props.locale, {
    weekday: 'long',
    year: 'numeric',
    month: 'long',
    day: 'numeric'
  })
  return formatter.format(day.date)
}

const toggleCalendar = () => {
  if (props.disabled) return
  isOpen.value = !isOpen.value
  
  if (isOpen.value) {
    const date = props.modelValue || new Date()
    viewMonth.value = date.getMonth()
    viewYear.value = date.getFullYear()
    focusedDate.value = date
    
    nextTick(() => {
      calendarRef.value?.focus()
    })
  }
}

const selectDate = (day: CalendarDay) => {
  if (isDisabled(day)) return
  
  const newDate = new Date(day.date)
  emit('update:modelValue', newDate)
  emit('change', newDate)
  isOpen.value = false
  
  nextTick(() => {
    inputRef.value?.focus()
  })
}

const selectToday = () => {
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  selectDate({
    date: today,
    day: today.getDate(),
    month: today.getMonth(),
    year: today.getFullYear(),
    otherMonth: false
  })
}

const handleClear = () => {
  emit('update:modelValue', null)
  emit('clear')
  nextTick(() => {
    inputRef.value?.focus()
  })
}

const previousMonth = () => {
  if (viewMonth.value === 0) {
    viewMonth.value = 11
    viewYear.value--
  } else {
    viewMonth.value--
  }
}

const nextMonth = () => {
  if (viewMonth.value === 11) {
    viewMonth.value = 0
    viewYear.value++
  } else {
    viewMonth.value++
  }
}

const handleInputKeydown = (event: KeyboardEvent) => {
  switch (event.key) {
    case 'Enter':
    case ' ':
    case 'ArrowDown':
      event.preventDefault()
      toggleCalendar()
      break
    case 'Escape':
      event.preventDefault()
      if (isOpen.value) {
        isOpen.value = false
      }
      break
  }
}

const handleCalendarKeydown = (event: KeyboardEvent) => {
  if (!focusedDate.value) {
    focusedDate.value = props.modelValue || new Date()
  }
  
  let newDate = new Date(focusedDate.value)
  let handled = true
  
  switch (event.key) {
    case 'ArrowLeft':
      newDate.setDate(newDate.getDate() - 1)
      break
    case 'ArrowRight':
      newDate.setDate(newDate.getDate() + 1)
      break
    case 'ArrowUp':
      newDate.setDate(newDate.getDate() - 7)
      break
    case 'ArrowDown':
      newDate.setDate(newDate.getDate() + 7)
      break
    case 'Home':
      newDate.setDate(1)
      break
    case 'End':
      newDate = new Date(newDate.getFullYear(), newDate.getMonth() + 1, 0)
      break
    case 'PageUp':
      if (event.shiftKey) {
        newDate.setFullYear(newDate.getFullYear() - 1)
      } else {
        newDate.setMonth(newDate.getMonth() - 1)
      }
      break
    case 'PageDown':
      if (event.shiftKey) {
        newDate.setFullYear(newDate.getFullYear() + 1)
      } else {
        newDate.setMonth(newDate.getMonth() + 1)
      }
      break
    case 'Enter':
    case ' ':
      event.preventDefault()
      if (!isDateDisabled(focusedDate.value)) {
        selectDate({
          date: focusedDate.value,
          day: focusedDate.value.getDate(),
          month: focusedDate.value.getMonth(),
          year: focusedDate.value.getFullYear(),
          otherMonth: false
        })
      }
      return
    case 'Escape':
      event.preventDefault()
      isOpen.value = false
      nextTick(() => {
        inputRef.value?.focus()
      })
      return
    default:
      handled = false
  }
  
  if (handled) {
    event.preventDefault()
    focusedDate.value = newDate
    
    // Update view if needed
    if (newDate.getMonth() !== viewMonth.value || newDate.getFullYear() !== viewYear.value) {
      viewMonth.value = newDate.getMonth()
      viewYear.value = newDate.getFullYear()
    }
  }
}

// Click outside handler
const handleClickOutside = (event: MouseEvent) => {
  const target = event.target as HTMLElement
  const element = target.closest('.date-picker')
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

// Initialize focused date
watch(isOpen, (newValue) => {
  if (newValue) {
    focusedDate.value = props.modelValue || new Date()
  }
})
</script>

<style lang="scss" scoped>
.date-picker {
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

  &__calendar {
    @apply absolute z-50 mt-1;
    @apply bg-white border border-gray-300 rounded-md shadow-lg;
    @apply p-4;
    @apply focus:outline-none;
  }

  &__header {
    @apply flex items-center justify-between mb-2;
  }

  &__nav-button {
    @apply p-1 rounded;
    @apply text-gray-600 hover:bg-gray-100;
    @apply focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-opacity-25;
    @apply transition-colors duration-150;
  }

  &__month-year {
    @apply flex items-center gap-1;
  }

  &__select {
    @apply px-2 py-1;
    @apply bg-white border border-gray-300 rounded;
    @apply text-sm font-medium;
    @apply focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-opacity-25;
    @apply focus:border-blue-500;
  }

  &__weekdays {
    @apply grid grid-cols-7 mb-1;
  }

  &__weekday {
    @apply text-center text-xs font-medium text-gray-500;
    @apply py-1;
  }

  &__grid {
    @apply space-y-1;
  }

  &__week {
    @apply grid grid-cols-7 gap-1;
  }

  &__day {
    @apply w-8 h-8;
    @apply flex items-center justify-center;
    @apply text-sm;
    @apply rounded;
    @apply transition-colors duration-150;
    @apply focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-opacity-25;

    &:hover:not(:disabled) {
      @apply bg-gray-100;
    }

    &--other-month {
      @apply text-gray-400;
    }

    &--today {
      @apply font-semibold text-blue-600;
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

    &--focused {
      @apply ring-2 ring-blue-500 ring-opacity-50;
    }
  }

  &__footer {
    @apply mt-2 pt-2 border-t border-gray-200;
    @apply text-center;
  }

  &__today-button {
    @apply px-3 py-1;
    @apply text-sm text-blue-600;
    @apply bg-white border border-blue-600 rounded;
    @apply hover:bg-blue-50;
    @apply focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-opacity-25;
    @apply transition-colors duration-150;

    &:disabled {
      @apply opacity-50 cursor-not-allowed;
      @apply hover:bg-white;
    }
  }
}

// Dark mode
.dark .date-picker {
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

  &__calendar {
    @apply bg-gray-800 border-gray-600;
  }

  &__nav-button {
    @apply text-gray-300 hover:bg-gray-700;
  }

  &__select {
    @apply bg-gray-700 border-gray-600 text-gray-100;
  }

  &__weekday {
    @apply text-gray-400;
  }

  &__day {
    @apply text-gray-100;

    &:hover:not(:disabled) {
      @apply bg-gray-700;
    }

    &--other-month {
      @apply text-gray-500;
    }

    &--today {
      @apply text-blue-400;
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

  &__footer {
    @apply border-gray-700;
  }

  &__today-button {
    @apply text-blue-400 bg-gray-800 border-blue-400;
    @apply hover:bg-gray-700;
  }
}

// Animations
.date-picker-calendar-enter-active,
.date-picker-calendar-leave-active {
  @apply transition-all duration-200;
}

.date-picker-calendar-enter-from,
.date-picker-calendar-leave-to {
  @apply opacity-0 transform scale-95;
}
</style>