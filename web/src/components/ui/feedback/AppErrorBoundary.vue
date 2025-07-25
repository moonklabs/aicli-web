<template>
  <div class="app-error-boundary">
    <slot v-if="!hasError" />
    
    <!-- 에러 발생 시 표시할 내용 -->
    <div
      v-else
      class="app-error-boundary__error"
      :class="[
        sizeClasses,
        {
          'app-error-boundary__error--centered': centered,
          'app-error-boundary__error--card': showCard
        }
      ]"
    >
      <!-- 커스텀 에러 슬롯이 있으면 사용 -->
      <slot
        v-if="$slots.error"
        name="error"
        :error="error"
        :retry="retry"
        :reset="reset"
      />
      
      <!-- 기본 에러 UI -->
      <div v-else class="app-error-boundary__content">
        <!-- 에러 아이콘 -->
        <div class="app-error-boundary__icon">
          <svg
            width="48"
            height="48"
            viewBox="0 0 24 24"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
            aria-hidden="true"
          >
            <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="2" />
            <line x1="12" y1="8" x2="12" y2="12" stroke="currentColor" stroke-width="2" />
            <line x1="12" y1="16" x2="12.01" y2="16" stroke="currentColor" stroke-width="2" />
          </svg>
        </div>
        
        <!-- 에러 메시지 -->
        <div class="app-error-boundary__message">
          <h3 class="app-error-boundary__title">
            {{ title || '오류가 발생했습니다' }}
          </h3>
          
          <p class="app-error-boundary__description">
            {{ description || '예상치 못한 오류가 발생했습니다. 잠시 후 다시 시도해 주세요.' }}
          </p>
          
          <!-- 개발 환경에서 에러 상세 정보 표시 -->
          <details
            v-if="showDetails && isDevelopment"
            class="app-error-boundary__details"
          >
            <summary class="app-error-boundary__details-summary">
              에러 상세 정보
            </summary>
            <pre class="app-error-boundary__details-content">{{ errorDetails }}</pre>
          </details>
        </div>
        
        <!-- 액션 버튼들 -->
        <div class="app-error-boundary__actions">
          <button
            v-if="showRetry"
            class="app-error-boundary__button app-error-boundary__button--primary"
            @click="retry"
            :disabled="retrying"
          >
            <svg
              v-if="retrying"
              class="app-error-boundary__button-icon animate-spin"
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              xmlns="http://www.w3.org/2000/svg"
            >
              <path
                d="M12 2V6M12 18V22M4.93 4.93L7.76 7.76M16.24 16.24L19.07 19.07M2 12H6M18 12H22M4.93 19.07L7.76 16.24M16.24 7.76L19.07 4.93"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
            </svg>
            
            <svg
              v-else
              class="app-error-boundary__button-icon"
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              xmlns="http://www.w3.org/2000/svg"
            >
              <path
                d="M1 4V10H7M23 20V14H17"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
              <path
                d="M20.49 9A9 9 0 0 0 5.64 5.64L1 10M23 14L18.36 18.36A9 9 0 0 1 3.51 15"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
            </svg>
            
            {{ retrying ? '재시도 중...' : '다시 시도' }}
          </button>
          
          <button
            v-if="showReset"
            class="app-error-boundary__button app-error-boundary__button--secondary"
            @click="reset"
          >
            초기화
          </button>
          
          <button
            v-if="showHome"
            class="app-error-boundary__button app-error-boundary__button--secondary"
            @click="goHome"
          >
            홈으로 가기
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onErrorCaptured, nextTick } from 'vue';
import { useRouter } from 'vue-router';
import type { Size } from '@/types/ui';

interface Props {
  title?: string;
  description?: string;
  size?: Size;
  centered?: boolean;
  showCard?: boolean;
  showRetry?: boolean;
  showReset?: boolean;
  showHome?: boolean;
  showDetails?: boolean;
  onRetry?: () => void | Promise<void>;
  onReset?: () => void;
}

const props = withDefaults(defineProps<Props>(), {
  size: 'medium',
  centered: true,
  showCard: true,
  showRetry: true,
  showReset: false,
  showHome: false,
  showDetails: true
});

const emit = defineEmits<{
  error: [error: Error];
  retry: [];
  reset: [];
}>();

const router = useRouter();

// 에러 상태
const hasError = ref(false);
const error = ref<Error | null>(null);
const retrying = ref(false);

// 개발 환경 여부
const isDevelopment = computed(() => import.meta.env.DEV);

// 크기별 클래스
const sizeClasses = computed(() => ({
  'app-error-boundary__error--small': props.size === 'small',
  'app-error-boundary__error--medium': props.size === 'medium',
  'app-error-boundary__error--large': props.size === 'large'
}));

// 에러 상세 정보
const errorDetails = computed(() => {
  if (!error.value) return '';
  
  return {
    message: error.value.message,
    stack: error.value.stack,
    name: error.value.name,
    timestamp: new Date().toISOString()
  };
});

// 에러 캐치
onErrorCaptured((err: Error) => {
  console.error('Error caught by ErrorBoundary:', err);
  
  hasError.value = true;
  error.value = err;
  
  emit('error', err);
  
  // 에러를 상위로 전파하지 않음
  return false;
});

// 재시도
const retry = async (): Promise<void> => {
  if (retrying.value) return;
  
  try {
    retrying.value = true;
    
    if (props.onRetry) {
      await props.onRetry();
    }
    
    // 에러 상태 초기화
    reset();
    
    emit('retry');
  } catch (err) {
    console.error('Retry failed:', err);
    // 재시도 실패 시에도 에러 상태는 유지
  } finally {
    retrying.value = false;
  }
};

// 초기화
const reset = (): void => {
  hasError.value = false;
  error.value = null;
  retrying.value = false;
  
  if (props.onReset) {
    props.onReset();
  }
  
  emit('reset');
  
  // DOM 업데이트 후 슬롯 재렌더링
  nextTick();
};

// 홈으로 이동
const goHome = (): void => {
  router.push('/');
};

// 외부에서 사용할 수 있는 메서드들
defineExpose({
  retry,
  reset,
  hasError: computed(() => hasError.value),
  error: computed(() => error.value)
});
</script>

<style lang="scss" scoped>
.app-error-boundary {
  @apply w-full h-full;
  
  &__error {
    @apply flex items-center justify-center p-6;
    min-height: 200px;
    
    &--centered {
      @apply text-center;
    }
    
    &--card {
      @apply bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700;
    }
    
    &--small {
      @apply p-4;
      min-height: 150px;
    }
    
    &--large {
      @apply p-8;
      min-height: 300px;
    }
  }
  
  &__content {
    @apply max-w-md w-full space-y-6;
  }
  
  &__icon {
    @apply text-red-500 flex justify-center;
    
    svg {
      @apply w-12 h-12;
    }
    
    .app-error-boundary__error--small & svg {
      @apply w-8 h-8;
    }
    
    .app-error-boundary__error--large & svg {
      @apply w-16 h-16;
    }
  }
  
  &__message {
    @apply space-y-3;
  }
  
  &__title {
    @apply text-lg font-semibold text-gray-900 dark:text-gray-100;
    
    .app-error-boundary__error--small & {
      @apply text-base;
    }
    
    .app-error-boundary__error--large & {
      @apply text-xl;
    }
  }
  
  &__description {
    @apply text-gray-600 dark:text-gray-400 leading-relaxed;
    
    .app-error-boundary__error--small & {
      @apply text-sm;
    }
  }
  
  &__details {
    @apply mt-4 text-left;
  }
  
  &__details-summary {
    @apply cursor-pointer text-sm font-medium text-gray-700 dark:text-gray-300 hover:text-gray-900 dark:hover:text-gray-100;
  }
  
  &__details-content {
    @apply mt-2 p-3 bg-gray-50 dark:bg-gray-900 rounded text-xs font-mono text-gray-800 dark:text-gray-200 overflow-auto max-h-32;
    white-space: pre-wrap;
    word-break: break-all;
  }
  
  &__actions {
    @apply flex flex-wrap justify-center gap-3;
    
    .app-error-boundary__error--small & {
      @apply gap-2;
    }
  }
  
  &__button {
    @apply inline-flex items-center px-4 py-2 rounded-md font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed;
    
    &--primary {
      @apply bg-red-600 text-white hover:bg-red-700 focus:ring-red-500;
    }
    
    &--secondary {
      @apply bg-gray-200 text-gray-800 hover:bg-gray-300 focus:ring-gray-500 dark:bg-gray-700 dark:text-gray-200 dark:hover:bg-gray-600;
    }
    
    .app-error-boundary__error--small & {
      @apply px-3 py-1.5 text-sm;
    }
  }
  
  &__button-icon {
    @apply mr-2 flex-shrink-0;
    
    &.animate-spin {
      animation: spin 1s linear infinite;
    }
  }
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}
</style>