<template>
  <div
    :class="[
      'app-form-field',
      {
        'app-form-field--required': required,
        'app-form-field--error': status === 'error',
        'app-form-field--success': status === 'success',
        'app-form-field--warning': status === 'warning',
        'app-form-field--disabled': disabled,
        'app-form-field--label-top': labelPlacement === 'top',
        'app-form-field--label-left': labelPlacement === 'left'
      }
    ]"
  >
    <!-- 라벨 -->
    <label
      v-if="label"
      :for="fieldId"
      class="app-form-field__label"
      :class="labelClasses"
    >
      {{ label }}
      <span v-if="required" class="app-form-field__required-mark" aria-label="필수">*</span>
    </label>

    <!-- 입력 필드 영역 -->
    <div class="app-form-field__control">
      <slot :field-id="fieldId" :aria-describedby="ariaDescribedby" />

      <!-- 도움말 텍스트 -->
      <div
        v-if="help && !showFeedback"
        :id="helpId"
        class="app-form-field__help"
      >
        {{ help }}
      </div>

      <!-- 피드백 메시지 -->
      <transition name="app-form-field-feedback">
        <div
          v-if="showFeedback && feedback"
          :id="feedbackId"
          class="app-form-field__feedback"
          :class="feedbackClasses"
          role="alert"
          :aria-live="status === 'error' ? 'assertive' : 'polite'"
        >
          <svg v-if="showFeedbackIcon" class="app-form-field__feedback-icon" viewBox="0 0 16 16" fill="currentColor">
            <!-- 성공 아이콘 -->
            <path
              v-if="status === 'success'"
              d="M13.854 3.646a.5.5 0 0 1 0 .708l-7 7a.5.5 0 0 1-.708 0l-3.5-3.5a.5.5 0 1 1 .708-.708L6.5 10.293l6.646-6.647a.5.5 0 0 1 .708 0z"
            />
            <!-- 경고 아이콘 -->
            <path
              v-else-if="status === 'warning'"
              d="M8 1a7 7 0 1 0 0 14A7 7 0 0 0 8 1zM7.5 4a.5.5 0 0 1 1 0v3a.5.5 0 0 1-1 0V4zM8 10a.75.75 0 1 1 0 1.5.75.75 0 0 1 0-1.5z"
            />
            <!-- 오류 아이콘 -->
            <path
              v-else-if="status === 'error'"
              d="M8 1a7 7 0 1 0 0 14A7 7 0 0 0 8 1zM6.646 5.646a.5.5 0 1 1 .708.708L8 7.293l.646-.647a.5.5 0 0 1 .708.708L8.707 8l.647.646a.5.5 0 0 1-.708.708L8 8.707l-.646.647a.5.5 0 0 1-.708-.708L7.293 8l-.647-.646z"
            />
          </svg>
          <span class="app-form-field__feedback-text">{{ feedback }}</span>
        </div>
      </transition>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, useId } from 'vue'
import type { ComponentStatus, FormFieldProps } from '@/types/ui'

interface Props extends FormFieldProps {
  help?: string;
  disabled?: boolean;
  showFeedbackIcon?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  labelPlacement: 'top',
  feedbackPlacement: 'bottom',
  showFeedback: true,
  showFeedbackIcon: true,
  required: false,
  disabled: false,
})

// 고유 ID 생성
const fieldId = useId()
const helpId = computed(() => `${fieldId}-help`)
const feedbackId = computed(() => `${fieldId}-feedback`)

// aria-describedby 속성 계산
const ariaDescribedby = computed(() => {
  const ids: string[] = []

  if (props.help && !props.showFeedback) {
    ids.push(helpId.value)
  }

  if (props.showFeedback && props.feedback) {
    ids.push(feedbackId.value)
  }

  return ids.length > 0 ? ids.join(' ') : undefined
})

// 라벨 클래스
const labelClasses = computed(() => ({
  'app-form-field__label--required': props.required,
  'app-form-field__label--disabled': props.disabled,
}))

// 피드백 클래스
const feedbackClasses = computed(() => ({
  'app-form-field__feedback--success': props.status === 'success',
  'app-form-field__feedback--warning': props.status === 'warning',
  'app-form-field__feedback--error': props.status === 'error',
}))
</script>

<style lang="scss" scoped>
.app-form-field {
  @apply mb-4;

  // 라벨 위치별 레이아웃
  &--label-top {
    @apply flex flex-col gap-1;
  }

  &--label-left {
    @apply flex items-start gap-4;

    .app-form-field__label {
      @apply flex-shrink-0 w-32 pt-2;
    }

    .app-form-field__control {
      @apply flex-1;
    }
  }

  // 상태별 스타일
  &--disabled {
    @apply opacity-60;
  }

  // 컴포넌트 요소들
  &__label {
    @apply block text-sm font-medium text-gray-700;
    @apply leading-5;

    &--required {
      @apply text-gray-900;
    }

    &--disabled {
      @apply text-gray-400;
    }
  }

  &__required-mark {
    @apply text-red-500 ml-1;
  }

  &__control {
    @apply relative;
  }

  &__help {
    @apply mt-1 text-sm text-gray-500;
    @apply leading-5;
  }

  &__feedback {
    @apply mt-1 flex items-start gap-1;
    @apply text-sm leading-5;

    &--success {
      @apply text-green-600;
    }

    &--warning {
      @apply text-yellow-600;
    }

    &--error {
      @apply text-red-600;
    }
  }

  &__feedback-icon {
    @apply flex-shrink-0 w-4 h-4 mt-0.5;
  }

  &__feedback-text {
    @apply flex-1;
  }
}

// 피드백 애니메이션
.app-form-field-feedback-enter-active,
.app-form-field-feedback-leave-active {
  @apply transition-all duration-200;
}

.app-form-field-feedback-enter-from,
.app-form-field-feedback-leave-to {
  @apply opacity-0 transform -translate-y-1;
}

// 다크 모드
.dark .app-form-field {
  &__label {
    @apply text-gray-300;

    &--required {
      @apply text-gray-100;
    }

    &--disabled {
      @apply text-gray-600;
    }
  }

  &__help {
    @apply text-gray-400;
  }

  &__feedback {
    &--success {
      @apply text-green-400;
    }

    &--warning {
      @apply text-yellow-400;
    }

    &--error {
      @apply text-red-400;
    }
  }
}

// 반응형: 작은 화면에서 라벨 위치 조정
@media (max-width: 640px) {
  .app-form-field {
    &--label-left {
      @apply flex-col gap-1;

      .app-form-field__label {
        @apply w-full pt-0;
      }
    }
  }
}

// 접근성: 움직임 감소
@media (prefers-reduced-motion: reduce) {
  .app-form-field-feedback-enter-active,
  .app-form-field-feedback-leave-active {
    @apply transition-none;
  }

  .app-form-field-feedback-enter-from,
  .app-form-field-feedback-leave-to {
    transform: none;
  }
}
</style>