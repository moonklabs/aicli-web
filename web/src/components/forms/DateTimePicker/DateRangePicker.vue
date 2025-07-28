<template>
  <div class="date-range-picker" :class="{ 'date-range-picker--disabled': disabled }">
    <div class="date-range-picker__input-wrapper">
      <input
        ref="inputRef"
        v-model="displayValue"
        class="date-range-picker__input"
        :class="{
          'date-range-picker__input--focused': isOpen,
          'date-range-picker__input--disabled': disabled
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
        class="date-range-picker__clear"
        :aria-label="clearAriaLabel"
        tabindex="-1"
      >
        <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
          <path d="M8 1a7 7 0 1 0 0 14A7 7 0 0 0 8 1zM4.646 4.646a.5.5 0 0 1 .708 0L8 7.293l2.646-2.647a.5.5 0 0 1 .708.708L8.707 8l2.647 2.646a.5.5 0 0 1-.708.708L8 8.707l-2.646 2.647a.5.5 0 0 1-.708-.708L7.293 8 4.646 5.354a.5.5 0 0 1 0-.708z"/>
        </svg>
      </button>
      
      <div class="date-range-picker__icon">
        <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
          <path d="M6 2a1 1 0 00-2 0v1H3a2 2 0 00-2 2v11a2 2 0 002 2h14a2 2 0 002-2V5a2 2 0 00-2-2h-1V2a1 1 0 10-2 0v1H6V2zM3 7h14v9H3V7z"/>
          <path d="M5 10a1 1 0 012 0v3a1 1 0 01-2 0v-3zM13 10a1 1 0 012 0v3a1 1 0 01-2 0v-3z"/>
        </svg>
      </div>
    </div>

    <Transition name="date-range-picker-calendar">
      <div
        v-if="isOpen"
        ref="calendarRef"
        class="date-range-picker__dropdown"
        @keydown="handleCalendarKeydown"
        role="dialog"
        aria-modal="true"
        :aria-label="calendarAriaLabel"
      >
        <div class="date-range-picker__content">
          <!-- Presets -->
          <div v-if="showPresets" class="date-range-picker__presets">
            <button
              v-for="preset in presets"
              :key="preset.label"
              @click="applyPreset(preset)"
              class="date-range-picker__preset-button"
            >
              {{ preset.label }}
            </button>
          </div>

          <!-- Calendars Container -->
          <div class="date-range-picker__calendars">
            <!-- Left Calendar -->
            <div class="date-range-picker__calendar">
              <div class="date-range-picker__header">
                <button
                  @click="previousMonth(0)"
                  class="date-range-picker__nav-button"
                  :aria-label="previousMonthAriaLabel"
                >
                  <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
                    <path d="M11.354 1.646a.5.5 0 0 1 0 .708L5.707 8l5.647 5.646a.5.5 0 0 1-.708.708l-6-6a.5.5 0 0 1 0-.708l6-6a.5.5 0 0 1 .708 0z"/>
                  </svg>
                </button>
                
                <div class="date-range-picker__month-year">
                  {{ monthNames[viewMonths[0].month] }} {{ viewMonths[0].year }}
                </div>
                
                <div class="date-range-picker__nav-spacer"></div>
              </div>

              <div class="date-range-picker__weekdays">
                <div
                  v-for="(day, index) in weekdayNames"
                  :key="index"
                  class="date-range-picker__weekday"
                >
                  {{ day }}
                </div>
              </div>

              <div class="date-range-picker__grid">
                <div
                  v-for="(week, weekIndex) in getCalendarDays(0)"
                  :key="weekIndex"
                  class="date-range-picker__week"
                >
                  <button
                    v-for="(day, dayIndex) in week"
                    :key="`${weekIndex}-${dayIndex}`"
                    class="date-range-picker__day"
                    :class="getDayClasses(day)"
                    :disabled="isDisabled(day)"
                    @click="selectDate(day)"
                    @mouseenter="hoveredDate = day.date"
                    @mouseleave="hoveredDate = null"
                    :aria-label="getDayAriaLabel(day)"
                  >
                    {{ day.day }}
                  </button>
                </div>
              </div>
            </div>

            <!-- Right Calendar -->
            <div class="date-range-picker__calendar">
              <div class="date-range-picker__header">
                <div class="date-range-picker__nav-spacer"></div>
                
                <div class="date-range-picker__month-year">
                  {{ monthNames[viewMonths[1].month] }} {{ viewMonths[1].year }}
                </div>
                
                <button
                  @click="nextMonth(1)"
                  class="date-range-picker__nav-button"
                  :aria-label="nextMonthAriaLabel"
                >
                  <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
                    <path d="M4.646 1.646a.5.5 0 0 1 .708 0l6 6a.5.5 0 0 1 0 .708l-6 6a.5.5 0 0 1-.708-.708L10.293 8 4.646 2.354a.5.5 0 0 1 0-.708z"/>
                  </svg>
                </button>
              </div>

              <div class="date-range-picker__weekdays">
                <div
                  v-for="(day, index) in weekdayNames"
                  :key="index"
                  class="date-range-picker__weekday"
                >
                  {{ day }}
                </div>
              </div>

              <div class="date-range-picker__grid">
                <div
                  v-for="(week, weekIndex) in getCalendarDays(1)"
                  :key="weekIndex"
                  class="date-range-picker__week"
                >
                  <button
                    v-for="(day, dayIndex) in week"
                    :key="`${weekIndex}-${dayIndex}`"
                    class="date-range-picker__day"
                    :class="getDayClasses(day)"
                    :disabled="isDisabled(day)"
                    @click="selectDate(day)"
                    @mouseenter="hoveredDate = day.date"
                    @mouseleave="hoveredDate = null"
                    :aria-label="getDayAriaLabel(day)"
                  >
                    {{ day.day }}
                  </button>
                </div>
              </div>
            </div>
          </div>

          <!-- Footer -->
          <div class="date-range-picker__footer">
            <div class="date-range-picker__selection">
              <span v-if="startDate">{{ formatDate(startDate) }}</span>
              <span v-else>{{ startPlaceholder }}</span>
              <span class="date-range-picker__separator">→</span>
              <span v-if="endDate">{{ formatDate(endDate) }}</span>
              <span v-else>{{ endPlaceholder }}</span>
            </div>
            <div class="date-range-picker__actions">
              <button
                @click="handleCancel"
                class="date-range-picker__action-button"
              >
                {{ cancelButtonText }}
              </button>
              <button
                @click="handleConfirm"
                class="date-range-picker__action-button date-range-picker__action-button--primary"
                :disabled="!startDate || !endDate"
              >
                {{ confirmButtonText }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'

interface CalendarDay {
  date: Date
  day: number
  month: number
  year: number
  otherMonth: boolean
}

interface DateRange {
  start: Date | null
  end: Date | null
}

interface Preset {
  label: string
  getValue: () => DateRange
}

interface Props {
  modelValue: DateRange | null
  min?: Date
  max?: Date
  disabled?: boolean
  clearable?: boolean
  placeholder?: string
  format?: string
  locale?: string
  firstDayOfWeek?: number
  showPresets?: boolean
  presets?: Preset[]
  ariaLabel?: string
  ariaDescribedby?: string
  clearAriaLabel?: string
  calendarAriaLabel?: string
  previousMonthAriaLabel?: string
  nextMonthAriaLabel?: string
  startPlaceholder?: string
  endPlaceholder?: string
  cancelButtonText?: string
  confirmButtonText?: string
}

const props = withDefaults(defineProps<Props>(), {
  disabled: false,
  clearable: true,
  placeholder: 'Select date range',
  format: 'YYYY-MM-DD',
  locale: 'ko-KR',
  firstDayOfWeek: 0,
  showPresets: true,
  clearAriaLabel: 'Clear date range',
  calendarAriaLabel: 'Date range picker',
  previousMonthAriaLabel: 'Previous month',
  nextMonthAriaLabel: 'Next month',
  startPlaceholder: '시작일',
  endPlaceholder: '종료일',
  cancelButtonText: '취소',
  confirmButtonText: '확인'
})

const emit = defineEmits<{
  'update:modelValue': [value: DateRange | null]
  'change': [value: DateRange | null]
  'clear': []
}>()

// State
const isOpen = ref(false)
const startDate = ref<Date | null>(null)
const endDate = ref<Date | null>(null)
const hoveredDate = ref<Date | null>(null)
const selecting = ref<'start' | 'end'>('start')
const viewMonths = ref([
  { month: new Date().getMonth(), year: new Date().getFullYear() },
  { month: new Date().getMonth() + 1, year: new Date().getFullYear() }
])

// Refs
const inputRef = ref<HTMLInputElement>()
const calendarRef = ref<HTMLDivElement>()

// Default presets
const defaultPresets: Preset[] = [
  {
    label: '오늘',
    getValue: () => {
      const today = new Date()
      today.setHours(0, 0, 0, 0)
      return { start: today, end: today }
    }
  },
  {
    label: '어제',
    getValue: () => {
      const yesterday = new Date()
      yesterday.setDate(yesterday.getDate() - 1)
      yesterday.setHours(0, 0, 0, 0)
      return { start: yesterday, end: yesterday }
    }
  },
  {
    label: '이번 주',
    getValue: () => {
      const now = new Date()
      const start = new Date(now)
      start.setDate(now.getDate() - now.getDay())
      start.setHours(0, 0, 0, 0)
      const end = new Date(start)
      end.setDate(start.getDate() + 6)
      return { start, end }
    }
  },
  {
    label: '이번 달',
    getValue: () => {
      const now = new Date()
      const start = new Date(now.getFullYear(), now.getMonth(), 1)
      const end = new Date(now.getFullYear(), now.getMonth() + 1, 0)
      return { start, end }
    }
  },
  {
    label: '지난 7일',
    getValue: () => {
      const end = new Date()
      end.setHours(0, 0, 0, 0)
      const start = new Date(end)
      start.setDate(end.getDate() - 6)
      return { start, end }
    }
  },
  {
    label: '지난 30일',
    getValue: () => {
      const end = new Date()
      end.setHours(0, 0, 0, 0)
      const start = new Date(end)
      start.setDate(end.getDate() - 29)
      return { start, end }
    }
  }
]

// Computed
const presets = computed(() => props.presets || defaultPresets)

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

const displayValue = computed(() => {
  if (!props.modelValue || !props.modelValue.start || !props.modelValue.end) return ''
  return `${formatDate(props.modelValue.start)} ~ ${formatDate(props.modelValue.end)}`
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

const isSameDay = (date1: Date, date2: Date): boolean => {
  return (
    date1.getDate() === date2.getDate() &&
    date1.getMonth() === date2.getMonth() &&
    date1.getFullYear() === date2.getFullYear()
  )
}

const isToday = (day: CalendarDay): boolean => {
  const today = new Date()
  return isSameDay(day.date, today)
}

const isSelected = (day: CalendarDay): boolean => {
  if (!startDate.value && !endDate.value) return false
  
  if (startDate.value && isSameDay(day.date, startDate.value)) return true
  if (endDate.value && isSameDay(day.date, endDate.value)) return true
  
  return false
}

const isInRange = (day: CalendarDay): boolean => {
  if (!startDate.value || !endDate.value) {
    // Show hover preview
    if (startDate.value && hoveredDate.value && !endDate.value) {
      const hoverStart = startDate.value < hoveredDate.value ? startDate.value : hoveredDate.value
      const hoverEnd = startDate.value < hoveredDate.value ? hoveredDate.value : startDate.value
      return day.date > hoverStart && day.date < hoverEnd
    }
    return false
  }
  
  return day.date > startDate.value && day.date < endDate.value
}

const isDisabled = (day: CalendarDay): boolean => {
  if (day.otherMonth) return true
  if (props.min && day.date < props.min) return true
  if (props.max && day.date > props.max) return true
  return false
}

const getDayClasses = (day: CalendarDay) => {
  return {
    'date-range-picker__day--other-month': day.otherMonth,
    'date-range-picker__day--today': isToday(day),
    'date-range-picker__day--selected': isSelected(day),
    'date-range-picker__day--in-range': isInRange(day),
    'date-range-picker__day--disabled': isDisabled(day),
    'date-range-picker__day--start': startDate.value && isSameDay(day.date, startDate.value),
    'date-range-picker__day--end': endDate.value && isSameDay(day.date, endDate.value)
  }
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

const getCalendarDays = (calendarIndex: number): CalendarDay[][] => {
  const { month, year } = viewMonths.value[calendarIndex]
  
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
}

const normalizeMonth = () => {
  // Ensure right calendar is always after left calendar
  if (viewMonths.value[1].year < viewMonths.value[0].year ||
      (viewMonths.value[1].year === viewMonths.value[0].year && 
       viewMonths.value[1].month <= viewMonths.value[0].month)) {
    viewMonths.value[1] = {
      year: viewMonths.value[0].month === 11 ? viewMonths.value[0].year + 1 : viewMonths.value[0].year,
      month: (viewMonths.value[0].month + 1) % 12
    }
  }
}

const previousMonth = (index: number) => {
  const vm = viewMonths.value[index]
  if (vm.month === 0) {
    vm.month = 11
    vm.year--
  } else {
    vm.month--
  }
  normalizeMonth()
}

const nextMonth = (index: number) => {
  const vm = viewMonths.value[index]
  if (vm.month === 11) {
    vm.month = 0
    vm.year++
  } else {
    vm.month++
  }
  normalizeMonth()
}

const toggleCalendar = () => {
  if (props.disabled) return
  isOpen.value = !isOpen.value
  
  if (isOpen.value) {
    // Initialize from model value
    if (props.modelValue?.start && props.modelValue?.end) {
      startDate.value = new Date(props.modelValue.start)
      endDate.value = new Date(props.modelValue.end)
      
      // Set view to show the selected range
      viewMonths.value[0] = {
        month: startDate.value.getMonth(),
        year: startDate.value.getFullYear()
      }
      normalizeMonth()
    } else {
      // Reset to current month
      const now = new Date()
      viewMonths.value[0] = {
        month: now.getMonth(),
        year: now.getFullYear()
      }
      normalizeMonth()
    }
    
    selecting.value = 'start'
    
    nextTick(() => {
      calendarRef.value?.focus()
    })
  }
}

const selectDate = (day: CalendarDay) => {
  if (isDisabled(day)) return
  
  const selectedDate = new Date(day.date)
  selectedDate.setHours(0, 0, 0, 0)
  
  if (selecting.value === 'start' || !startDate.value) {
    startDate.value = selectedDate
    endDate.value = null
    selecting.value = 'end'
  } else {
    if (selectedDate < startDate.value!) {
      // If end date is before start date, swap them
      endDate.value = startDate.value
      startDate.value = selectedDate
    } else {
      endDate.value = selectedDate
    }
    selecting.value = 'start'
  }
}

const applyPreset = (preset: Preset) => {
  const range = preset.getValue()
  startDate.value = range.start
  endDate.value = range.end
}

const handleConfirm = () => {
  if (!startDate.value || !endDate.value) return
  
  const value: DateRange = {
    start: new Date(startDate.value),
    end: new Date(endDate.value)
  }
  
  emit('update:modelValue', value)
  emit('change', value)
  isOpen.value = false
  
  nextTick(() => {
    inputRef.value?.focus()
  })
}

const handleCancel = () => {
  // Reset to original values
  if (props.modelValue?.start && props.modelValue?.end) {
    startDate.value = new Date(props.modelValue.start)
    endDate.value = new Date(props.modelValue.end)
  } else {
    startDate.value = null
    endDate.value = null
  }
  
  isOpen.value = false
  nextTick(() => {
    inputRef.value?.focus()
  })
}

const handleClear = () => {
  startDate.value = null
  endDate.value = null
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
  switch (event.key) {
    case 'Escape':
      event.preventDefault()
      handleCancel()
      break
  }
}

// Click outside handler
const handleClickOutside = (event: MouseEvent) => {
  const target = event.target as HTMLElement
  const element = target.closest('.date-range-picker')
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
.date-range-picker {
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

  &__presets {
    @apply flex flex-wrap gap-2 mb-4 pb-4 border-b border-gray-200;
  }

  &__preset-button {
    @apply px-3 py-1;
    @apply text-sm;
    @apply border border-gray-300 rounded;
    @apply hover:bg-gray-50;
    @apply focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-opacity-25;
    @apply transition-colors duration-150;
  }

  &__calendars {
    @apply flex gap-4;
  }

  &__calendar {
    @apply flex-1;
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

  &__nav-spacer {
    @apply w-8;
  }

  &__month-year {
    @apply text-sm font-medium;
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

    &--in-range {
      @apply bg-blue-100;

      &:hover {
        @apply bg-blue-200;
      }
    }

    &--start {
      @apply rounded-r-none;
    }

    &--end {
      @apply rounded-l-none;
    }

    &--disabled {
      @apply text-gray-300 cursor-not-allowed;
      
      &:hover {
        @apply bg-transparent;
      }
    }
  }

  &__footer {
    @apply mt-4 pt-4 border-t border-gray-200;
  }

  &__selection {
    @apply mb-3 text-center text-sm text-gray-600;
  }

  &__separator {
    @apply mx-2;
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

      &:disabled {
        @apply opacity-50 cursor-not-allowed;
        @apply hover:bg-blue-600;
      }
    }
  }
}

// Dark mode
.dark .date-range-picker {
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

  &__presets {
    @apply border-gray-700;
  }

  &__preset-button {
    @apply border-gray-600 text-gray-100;
    @apply hover:bg-gray-700;
  }

  &__nav-button {
    @apply text-gray-300 hover:bg-gray-700;
  }

  &__month-year {
    @apply text-gray-100;
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

    &--in-range {
      @apply bg-blue-900 bg-opacity-30;

      &:hover {
        @apply bg-blue-900 bg-opacity-50;
      }
    }

    &--disabled {
      @apply text-gray-600;
    }
  }

  &__footer {
    @apply border-gray-700;
  }

  &__selection {
    @apply text-gray-400;
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
.date-range-picker-calendar-enter-active,
.date-range-picker-calendar-leave-active {
  @apply transition-all duration-200;
}

.date-range-picker-calendar-enter-from,
.date-range-picker-calendar-leave-to {
  @apply opacity-0 transform scale-95;
}
</style>