<template>
  <div
    :class="[
      'app-input',
      sizeClasses,
      statusClasses,
      {
        'app-input--disabled': disabled,
        'app-input--readonly': readonly,
        'app-input--focused': focused,
        'app-input--clearable': clearable && clearVisible,
        'app-input--password': type === 'password',
        'app-input--with-prefix': $slots.prefix,
        'app-input--with-suffix': $slots.suffix || clearable || showPasswordToggle,
        'app-input--round': round
      }
    ]"
  >
    <!-- 접두사 슬롯 -->
    <div v-if="$slots.prefix" class="app-input__prefix">
      <slot name="prefix" />
    </div>

    <!-- 실제 입력 필드 -->
    <input
      ref="inputRef"
      :value="modelValue"
      :type="currentType"
      :placeholder="placeholder"
      :disabled="disabled"
      :readonly="readonly"
      :maxlength="maxlength"
      :autocomplete="autocomplete"
      :aria-label="ariaLabel"
      :aria-describedby="ariaDescribedby"
      :aria-invalid="status === 'error'"
      :aria-required="required"
      :tabindex="disabled ? -1 : tabindex"
      class="app-input__field"
      v-bind="$attrs"
      @input="handleInput"
      @change="handleChange"
      @focus="handleFocus"
      @blur="handleBlur"
      @keydown="handleKeydown"
      @keyup="handleKeyup"
    />

    <!-- 접미사 영역 -->
    <div v-if="$slots.suffix || clearable || showPasswordToggle || showCount" class="app-input__suffix">
      <!-- 문자 수 표시 -->
      <span v-if="showCount && maxlength" class="app-input__count">
        {{ currentLength }}/{{ maxlength }}
      </span>

      <!-- 클리어 버튼 -->
      <button
        v-if="clearable && clearVisible"
        type="button"
        class="app-input__clear"
        :aria-label="'입력 내용 지우기'"
        :tabindex="disabled ? -1 : 0"
        @click="handleClear"
        @keydown="handleClearKeydown"
      >
        <svg viewBox="0 0 16 16" fill="currentColor">
          <path d="M8 1a7 7 0 1 0 0 14A7 7 0 0 0 8 1zM6.354 5.646a.5.5 0 1 0-.708.708L7.293 8l-1.647 1.646a.5.5 0 0 0 .708.708L8 8.707l1.646 1.647a.5.5 0 0 0 .708-.708L8.707 8l1.647-1.646a.5.5 0 0 0-.708-.708L8 7.293 6.354 5.646z"/>
        </svg>
      </button>

      <!-- 비밀번호 표시/숨기기 버튼 -->
      <button
        v-if="showPasswordToggle"
        type="button"
        class="app-input__password-toggle"
        :aria-label="passwordVisible ? '비밀번호 숨기기' : '비밀번호 보기'"
        :tabindex="disabled ? -1 : 0"
        @click="togglePasswordVisibility"
        @keydown="handlePasswordToggleKeydown"
      >
        <svg v-if="passwordVisible" viewBox="0 0 16 16" fill="currentColor">
          <path d="M16 8s-3-5.5-8-5.5S0 8 0 8s3 5.5 8 5.5S16 8 16 8zM1.173 8a13.133 13.133 0 0 1 1.66-2.043C4.12 4.668 5.88 3.5 8 3.5c2.12 0 3.879 1.168 5.168 2.457A13.133 13.133 0 0 1 14.828 8c-.058.087-.122.183-.195.288-.335.48-.83 1.12-1.465 1.755C11.879 11.332 10.119 12.5 8 12.5c-2.12 0-3.879-1.168-5.168-2.457A13.134 13.134 0 0 1 1.172 8z"/>
          <path d="M8 5.5a2.5 2.5 0 1 0 0 5 2.5 2.5 0 0 0 0-5zM4.5 8a3.5 3.5 0 1 1 7 0 3.5 3.5 0 0 1-7 0z"/>
        </svg>
        <svg v-else viewBox="0 0 16 16" fill="currentColor">
          <path d="M13.359 11.238C15.06 9.72 16 8 16 8s-3-5.5-8-5.5a7.028 7.028 0 0 0-2.79.588l.77.771A5.944 5.944 0 0 1 8 3.5c2.12 0 3.879 1.168 5.168 2.457A13.134 13.134 0 0 1 14.828 8c-.058.087-.122.183-.195.288-.335.48-.83 1.12-1.465 1.755-.165.165-.337.328-.517.486l.708.709z"/>
          <path d="M11.297 9.176a3.5 3.5 0 0 0-4.474-4.474l.823.823a2.5 2.5 0 0 1 2.829 2.829l.822.822zm-2.943 1.299.822.822a3.5 3.5 0 0 1-4.474-4.474l.823.823a2.5 2.5 0 0 0 2.829 2.829z"/>
          <path d="M3.35 5.47c-.18.16-.353.322-.518.487A13.134 13.134 0 0 0 1.172 8l.195.288c.335.48.83 1.12 1.465 1.755C4.121 11.332 5.881 12.5 8 12.5c.716 0 1.39-.133 2.02-.36l.77.772A7.029 7.029 0 0 1 8 13.5C3 13.5 0 8 0 8s.939-1.721 2.641-3.238l.708.708zm10.296 8.884-12-12 .708-.708 12 12-.708.708z"/>
        </svg>
      </button>

      <!-- 접미사 슬롯 -->
      <div v-if="$slots.suffix" class="app-input__suffix-content">
        <slot name="suffix" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import type { InputBaseProps } from '@/types/ui'

interface Props extends InputBaseProps {
  modelValue?: string | number;
  type?: 'text' | 'password' | 'email' | 'number' | 'tel' | 'url';
  autocomplete?: string;
  required?: boolean;
  round?: boolean;
  ariaLabel?: string;
  ariaDescribedby?: string;
  tabindex?: number;
  showPasswordOn?: 'click' | 'mousedown' | false;
}

const props = withDefaults(defineProps<Props>(), {
  type: 'text',
  size: 'medium',
  status: 'default',
  disabled: false,
  readonly: false,
  clearable: false,
  showCount: false,
  round: false,
  required: false,
  tabindex: 0,
  showPasswordOn: 'click',
})

const emit = defineEmits<{
  'update:value': [value: string | number];
  'focus': [event: FocusEvent];
  'blur': [event: FocusEvent];
  'clear': [];
  'keydown': [event: KeyboardEvent];
  'keyup': [event: KeyboardEvent];
  'change': [event: Event];
  'input': [event: Event];
}>()

// 반응형 상태
const inputRef = ref<HTMLInputElement>()
const focused = ref(false)
const passwordVisible = ref(false)

// 비밀번호 표시/숨기기 토글 표시 여부
const showPasswordToggle = computed(() => {
  return props.type === 'password' && props.showPasswordOn !== false
})

// 현재 입력 타입 (비밀번호 가시성에 따라 변경)
const currentType = computed(() => {
  if (props.type === 'password' && passwordVisible.value) {
    return 'text'
  }
  return props.type
})

// 현재 값의 길이
const currentLength = computed(() => {
  return String(props.modelValue || '').length
})

// 클리어 버튼 표시 여부
const clearVisible = computed(() => {
  return props.modelValue && String(props.modelValue).length > 0 && !props.disabled && !props.readonly
})

// 사이즈별 클래스
const sizeClasses = computed(() => ({
  'app-input--small': props.size === 'small',
  'app-input--medium': props.size === 'medium',
  'app-input--large': props.size === 'large',
}))

// 상태별 클래스
const statusClasses = computed(() => ({
  'app-input--default': props.status === 'default',
  'app-input--success': props.status === 'success',
  'app-input--warning': props.status === 'warning',
  'app-input--error': props.status === 'error',
}))

// 이벤트 핸들러
const handleInput = (event: Event) => {
  const target = event.target as HTMLInputElement
  const value = props.type === 'number' ? Number(target.value) : target.value

  emit('update:value', value)
  emit('input', event)

  // v-model 업데이트를 위한 onChange 호출
  props.onUpdate?.value?.(value)
  props.onChange?.(value)
}

const handleChange = (event: Event) => {
  emit('change', event)
}

const handleFocus = (event: FocusEvent) => {
  focused.value = true
  emit('focus', event)
  props.onFocus?.(event)
}

const handleBlur = (event: FocusEvent) => {
  focused.value = false
  emit('blur', event)
  props.onBlur?.(event)
}

const handleKeydown = (event: KeyboardEvent) => {
  emit('keydown', event)

  // Escape 키로 클리어
  if (event.key === 'Escape' && props.clearable && clearVisible.value) {
    handleClear()
  }
}

const handleKeyup = (event: KeyboardEvent) => {
  emit('keyup', event)
}

const handleClear = () => {
  const value = props.type === 'number' ? 0 : ''
  emit('update:value', value)
  emit('clear')

  props.onUpdate?.value?.(value)
  props.onChange?.(value)

  // 포커스를 input으로 이동
  nextTick(() => {
    inputRef.value?.focus()
  })
}

const handleClearKeydown = (event: KeyboardEvent) => {
  if (event.key === 'Enter' || event.key === ' ') {
    event.preventDefault()
    handleClear()
  }
}

const togglePasswordVisibility = () => {
  passwordVisible.value = !passwordVisible.value

  // 포커스를 input으로 유지
  nextTick(() => {
    inputRef.value?.focus()
  })
}

const handlePasswordToggleKeydown = (event: KeyboardEvent) => {
  if (event.key === 'Enter' || event.key === ' ') {
    event.preventDefault()
    togglePasswordVisibility()
  }
}

// 포커스 메서드 노출
const focus = () => {
  inputRef.value?.focus()
}

const blur = () => {
  inputRef.value?.blur()
}

const select = () => {
  inputRef.value?.select()
}

defineExpose({
  focus,
  blur,
  select,
  inputRef,
})

// 비밀번호 가시성 상태 초기화
watch(() => props.type, (newType) => {
  if (newType !== 'password') {
    passwordVisible.value = false
  }
})
</script>

<style lang="scss" scoped>
.app-input {
  @apply relative inline-flex items-center w-full;
  @apply bg-white border border-solid border-gray-300 rounded-md;
  @apply transition-colors duration-200;
  @apply focus-within:ring-2 focus-within:ring-blue-500 focus-within:ring-opacity-20;
  @apply focus-within:border-blue-500;

  // 비활성화 상태
  &--disabled {
    @apply bg-gray-50 border-gray-200 cursor-not-allowed;

    .app-input__field {
      @apply cursor-not-allowed;
    }
  }

  // 읽기 전용 상태
  &--readonly {
    @apply bg-gray-50;

    .app-input__field {
      @apply cursor-default;
    }
  }

  // 포커스 상태
  &--focused {
    @apply border-blue-500;
  }

  // 둥근 모서리
  &--round {
    @apply rounded-full;
  }

  // 사이즈 변형
  &--small {
    @apply text-sm min-h-[32px];

    .app-input__field {
      @apply py-1.5 px-3;
    }

    .app-input__prefix,
    .app-input__suffix {
      @apply px-2;
    }

    .app-input__clear,
    .app-input__password-toggle {
      @apply w-4 h-4;
    }

    .app-input__count {
      @apply text-xs;
    }
  }

  &--medium {
    @apply text-base min-h-[40px];

    .app-input__field {
      @apply py-2 px-3;
    }

    .app-input__prefix,
    .app-input__suffix {
      @apply px-3;
    }

    .app-input__clear,
    .app-input__password-toggle {
      @apply w-5 h-5;
    }

    .app-input__count {
      @apply text-sm;
    }
  }

  &--large {
    @apply text-lg min-h-[48px];

    .app-input__field {
      @apply py-3 px-4;
    }

    .app-input__prefix,
    .app-input__suffix {
      @apply px-4;
    }

    .app-input__clear,
    .app-input__password-toggle {
      @apply w-6 h-6;
    }

    .app-input__count {
      @apply text-base;
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

  // 접두사가 있는 경우
  &--with-prefix {
    .app-input__field {
      @apply pl-0;
    }
  }

  // 접미사가 있는 경우
  &--with-suffix {
    .app-input__field {
      @apply pr-0;
    }
  }

  // 컴포넌트 요소들
  &__prefix {
    @apply flex items-center text-gray-500 bg-gray-50 border-r border-gray-300;
    @apply rounded-l-md;
  }

  &__field {
    @apply flex-1 bg-transparent border-none outline-none;
    @apply text-gray-900 placeholder-gray-500;

    &:disabled {
      @apply text-gray-400;
    }

    // 숫자 입력 스피너 제거
    &[type="number"] {
      -moz-appearance: textfield;

      &::-webkit-outer-spin-button,
      &::-webkit-inner-spin-button {
        -webkit-appearance: none;
        margin: 0;
      }
    }
  }

  &__suffix {
    @apply flex items-center gap-1 text-gray-500;
  }

  &__count {
    @apply text-gray-400 select-none;
  }

  &__clear,
  &__password-toggle {
    @apply inline-flex items-center justify-center;
    @apply text-gray-400 hover:text-gray-600;
    @apply cursor-pointer transition-colors duration-200;
    @apply focus:outline-none focus:text-gray-600;
    @apply rounded;

    &:focus-visible {
      @apply ring-2 ring-blue-500 ring-opacity-50;
    }
  }

  &__suffix-content {
    @apply flex items-center;
  }
}

// 다크 모드
.dark .app-input {
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

  .app-input__prefix {
    @apply text-gray-400 bg-gray-900 border-gray-600;
  }

  .app-input__field {
    @apply text-gray-100 placeholder-gray-400;

    &:disabled {
      @apply text-gray-600;
    }
  }

  .app-input__count {
    @apply text-gray-500;
  }

  .app-input__clear,
  .app-input__password-toggle {
    @apply text-gray-500 hover:text-gray-300;

    &:focus {
      @apply text-gray-300;
    }
  }
}

// 접근성: 고대비 모드
@media (prefers-contrast: high) {
  .app-input {
    @apply border-2;

    &:focus-within {
      @apply ring-4;
    }
  }
}

// 접근성: 움직임 감소
@media (prefers-reduced-motion: reduce) {
  .app-input {
    @apply transition-none;
  }

  .app-input__clear,
  .app-input__password-toggle {
    @apply transition-none;
  }
}
</style>