<template>
  <div
    :class="[
      'app-textarea',
      sizeClasses,
      statusClasses,
      {
        'app-textarea--disabled': disabled,
        'app-textarea--readonly': readonly,
        'app-textarea--focused': focused,
        'app-textarea--clearable': clearable && clearVisible,
        'app-textarea--resizable': resizable,
        'app-textarea--autosize': autosize,
        'app-textarea--round': round
      }
    ]"
  >
    <!-- 실제 텍스트 영역 -->
    <textarea
      ref="textareaRef"
      :value="modelValue"
      :placeholder="placeholder"
      :disabled="disabled"
      :readonly="readonly"
      :rows="currentRows"
      :maxlength="maxlength"
      :aria-label="ariaLabel"
      :aria-describedby="ariaDescribedby"
      :aria-invalid="status === 'error'"
      :aria-required="required"
      :tabindex="disabled ? -1 : tabindex"
      class="app-textarea__field"
      v-bind="$attrs"
      @input="handleInput"
      @change="handleChange"
      @focus="handleFocus"
      @blur="handleBlur"
      @keydown="handleKeydown"
      @keyup="handleKeyup"
    ></textarea>

    <!-- 도구 모음 -->
    <div v-if="clearable || showCount" class="app-textarea__toolbar">
      <!-- 문자 수 표시 -->
      <span v-if="showCount" class="app-textarea__count" :class="countClasses">
        <span v-if="maxlength">{{ currentLength }}/{{ maxlength }}</span>
        <span v-else>{{ currentLength }}자</span>
      </span>

      <!-- 클리어 버튼 -->
      <button
        v-if="clearable && clearVisible"
        type="button"
        class="app-textarea__clear"
        :aria-label="'입력 내용 지우기'"
        :tabindex="disabled ? -1 : 0"
        @click="handleClear"
        @keydown="handleClearKeydown"
      >
        <svg viewBox="0 0 16 16" fill="currentColor">
          <path d="M8 1a7 7 0 1 0 0 14A7 7 0 0 0 8 1zM6.354 5.646a.5.5 0 1 0-.708.708L7.293 8l-1.647 1.646a.5.5 0 0 0 .708.708L8 8.707l1.646 1.647a.5.5 0 0 0 .708-.708L8.707 8l1.647-1.646a.5.5 0 0 0-.708-.708L8 7.293 6.354 5.646z"/>
        </svg>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'

interface Props {
  modelValue?: string;
  placeholder?: string;
  disabled?: boolean;
  readonly?: boolean;
  size?: 'small' | 'medium' | 'large';
  status?: 'default' | 'success' | 'warning' | 'error';
  rows?: number;
  minRows?: number;
  maxRows?: number;
  maxlength?: number;
  clearable?: boolean;
  showCount?: boolean;
  resizable?: boolean;
  autosize?: boolean;
  round?: boolean;
  required?: boolean;
  ariaLabel?: string;
  ariaDescribedby?: string;
  tabindex?: number;
}

const props = withDefaults(defineProps<Props>(), {
  size: 'medium',
  status: 'default',
  rows: 3,
  minRows: 3,
  maxRows: 10,
  disabled: false,
  readonly: false,
  clearable: false,
  showCount: false,
  resizable: true,
  autosize: false,
  round: false,
  required: false,
  tabindex: 0,
})

const emit = defineEmits<{
  'update:value': [value: string];
  'focus': [event: FocusEvent];
  'blur': [event: FocusEvent];
  'clear': [];
  'keydown': [event: KeyboardEvent];
  'keyup': [event: KeyboardEvent];
  'change': [event: Event];
  'input': [event: Event];
  'resize': [height: number];
}>()

// 반응형 상태
const textareaRef = ref<HTMLTextAreaElement>()
const focused = ref(false)
const currentRows = ref(props.rows)

// 현재 값의 길이
const currentLength = computed(() => {
  return String(props.modelValue || '').length
})

// 문자 수 제한 경고 클래스
const countClasses = computed(() => {
  if (!props.maxlength) return {}

  const ratio = currentLength.value / props.maxlength
  return {
    'app-textarea__count--warning': ratio >= 0.8,
    'app-textarea__count--error': ratio >= 1,
  }
})

// 클리어 버튼 표시 여부
const clearVisible = computed(() => {
  return props.modelValue && String(props.modelValue).length > 0 && !props.disabled && !props.readonly
})

// 사이즈별 클래스
const sizeClasses = computed(() => ({
  'app-textarea--small': props.size === 'small',
  'app-textarea--medium': props.size === 'medium',
  'app-textarea--large': props.size === 'large',
}))

// 상태별 클래스
const statusClasses = computed(() => ({
  'app-textarea--default': props.status === 'default',
  'app-textarea--success': props.status === 'success',
  'app-textarea--warning': props.status === 'warning',
  'app-textarea--error': props.status === 'error',
}))

// 자동 크기 조정
const adjustHeight = () => {
  if (!props.autosize || !textareaRef.value) return

  const textarea = textareaRef.value

  // 기본 높이로 초기화
  textarea.style.height = 'auto'

  // 내용에 맞는 높이 계산
  const scrollHeight = textarea.scrollHeight
  const lineHeight = parseInt(getComputedStyle(textarea).lineHeight, 10)
  const padding = parseInt(getComputedStyle(textarea).paddingTop, 10) +
                  parseInt(getComputedStyle(textarea).paddingBottom, 10)

  const minHeight = (props.minRows * lineHeight) + padding
  const maxHeight = (props.maxRows * lineHeight) + padding

  const newHeight = Math.min(Math.max(scrollHeight, minHeight), maxHeight)

  textarea.style.height = `${newHeight}px`

  // 행 수 업데이트
  const newRows = Math.round((newHeight - padding) / lineHeight)
  currentRows.value = newRows

  emit('resize', newHeight)
}

// 이벤트 핸들러
const handleInput = (event: Event) => {
  const target = event.target as HTMLTextAreaElement

  emit('update:value', target.value)
  emit('input', event)

  // 자동 크기 조정
  if (props.autosize) {
    nextTick(() => {
      adjustHeight()
    })
  }
}

const handleChange = (event: Event) => {
  emit('change', event)
}

const handleFocus = (event: FocusEvent) => {
  focused.value = true
  emit('focus', event)
}

const handleBlur = (event: FocusEvent) => {
  focused.value = false
  emit('blur', event)
}

const handleKeydown = (event: KeyboardEvent) => {
  emit('keydown', event)

  // Escape 키로 클리어
  if (event.key === 'Escape' && props.clearable && clearVisible.value) {
    handleClear()
  }

  // Ctrl+Enter로 폼 제출 (부모에서 처리)
  if (event.key === 'Enter' && (event.ctrlKey || event.metaKey)) {
    // 부모 컴포넌트에서 처리할 수 있도록 이벤트 전달
    event.preventDefault()
  }
}

const handleKeyup = (event: KeyboardEvent) => {
  emit('keyup', event)
}

const handleClear = () => {
  emit('update:value', '')
  emit('clear')

  // 포커스를 textarea로 이동
  nextTick(() => {
    textareaRef.value?.focus()
    adjustHeight()
  })
}

const handleClearKeydown = (event: KeyboardEvent) => {
  if (event.key === 'Enter' || event.key === ' ') {
    event.preventDefault()
    handleClear()
  }
}

// 포커스 메서드 노출
const focus = () => {
  textareaRef.value?.focus()
}

const blur = () => {
  textareaRef.value?.blur()
}

const select = () => {
  textareaRef.value?.select()
}

const setSelectionRange = (start: number, end: number) => {
  textareaRef.value?.setSelectionRange(start, end)
}

defineExpose({
  focus,
  blur,
  select,
  setSelectionRange,
  textareaRef,
  adjustHeight,
})

// 마운트 후 초기 높이 조정
onMounted(() => {
  if (props.autosize) {
    nextTick(() => {
      adjustHeight()
    })
  }
})

// 값 변경 시 높이 재조정
watch(() => props.modelValue, () => {
  if (props.autosize) {
    nextTick(() => {
      adjustHeight()
    })
  }
})

// autosize 속성 변경 시 처리
watch(() => props.autosize, (newAutosize) => {
  if (newAutosize) {
    nextTick(() => {
      adjustHeight()
    })
  } else if (textareaRef.value) {
    textareaRef.value.style.height = 'auto'
    currentRows.value = props.rows
  }
})
</script>

<style lang="scss" scoped>
.app-textarea {
  @apply relative inline-flex flex-col w-full;
  @apply bg-white border border-solid border-gray-300 rounded-md;
  @apply transition-colors duration-200;
  @apply focus-within:ring-2 focus-within:ring-blue-500 focus-within:ring-opacity-20;
  @apply focus-within:border-blue-500;

  // 비활성화 상태
  &--disabled {
    @apply bg-gray-50 border-gray-200 cursor-not-allowed;

    .app-textarea__field {
      @apply cursor-not-allowed;
    }
  }

  // 읽기 전용 상태
  &--readonly {
    @apply bg-gray-50;

    .app-textarea__field {
      @apply cursor-default;
    }
  }

  // 포커스 상태
  &--focused {
    @apply border-blue-500;
  }

  // 둥근 모서리
  &--round {
    @apply rounded-xl;
  }

  // 크기 조정 불가
  &:not(.app-textarea--resizable) {
    .app-textarea__field {
      resize: none;
    }
  }

  // 자동 크기 조정
  &--autosize {
    .app-textarea__field {
      resize: none;
      overflow: hidden;
    }
  }

  // 사이즈 변형
  &--small {
    @apply text-sm;

    .app-textarea__field {
      @apply p-2;
    }

    .app-textarea__toolbar {
      @apply px-2 py-1.5;
    }

    .app-textarea__count {
      @apply text-xs;
    }

    .app-textarea__clear {
      @apply w-4 h-4;
    }
  }

  &--medium {
    @apply text-base;

    .app-textarea__field {
      @apply p-3;
    }

    .app-textarea__toolbar {
      @apply px-3 py-2;
    }

    .app-textarea__count {
      @apply text-sm;
    }

    .app-textarea__clear {
      @apply w-5 h-5;
    }
  }

  &--large {
    @apply text-lg;

    .app-textarea__field {
      @apply p-4;
    }

    .app-textarea__toolbar {
      @apply px-4 py-2.5;
    }

    .app-textarea__count {
      @apply text-base;
    }

    .app-textarea__clear {
      @apply w-6 h-6;
    }
  }

  // 상태별 색상
  &--success {
    @apply border-green-500;

    &:focus-within {
      @apply ring-green-500 ring-opacity-20 border-green-500;
    }
  }

  &--warning {
    @apply border-yellow-500;

    &:focus-within {
      @apply ring-yellow-500 ring-opacity-20 border-yellow-500;
    }
  }

  &--error {
    @apply border-red-500;

    &:focus-within {
      @apply ring-red-500 ring-opacity-20 border-red-500;
    }
  }

  // 컴포넌트 요소들
  &__field {
    @apply flex-1 w-full bg-transparent border-none outline-none;
    @apply text-gray-900 placeholder-gray-500;
    @apply resize-y;
    @apply rounded-md;

    &:disabled {
      @apply text-gray-400;
    }
  }

  &__toolbar {
    @apply flex items-center justify-between;
    @apply border-t border-gray-200 bg-gray-50;
    @apply rounded-b-md;
  }

  &__count {
    @apply text-gray-400 select-none font-mono;
    @apply transition-colors duration-200;

    &--warning {
      @apply text-yellow-600;
    }

    &--error {
      @apply text-red-600;
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
}

// 다크 모드
.dark .app-textarea {
  @apply bg-gray-800 border-gray-600;

  &:focus-within {
    @apply border-blue-400;
  }

  &--disabled {
    @apply bg-gray-900 border-gray-700;
  }

  &--readonly {
    @apply bg-gray-900;
  }

  .app-textarea__field {
    @apply text-gray-100 placeholder-gray-400;

    &:disabled {
      @apply text-gray-600;
    }
  }

  .app-textarea__toolbar {
    @apply border-gray-700 bg-gray-900;
  }

  .app-textarea__count {
    @apply text-gray-500;

    &--warning {
      @apply text-yellow-400;
    }

    &--error {
      @apply text-red-400;
    }
  }

  .app-textarea__clear {
    @apply text-gray-500 hover:text-gray-300;

    &:focus {
      @apply text-gray-300;
    }
  }
}

// 접근성: 고대비 모드
@media (prefers-contrast: high) {
  .app-textarea {
    @apply border-2;

    &:focus-within {
      @apply ring-4;
    }
  }
}

// 접근성: 움직임 감소
@media (prefers-reduced-motion: reduce) {
  .app-textarea {
    @apply transition-none;
  }

  .app-textarea__count,
  .app-textarea__clear {
    @apply transition-none;
  }
}
</style>