<template>
  <div class="password-change-form">
    <div class="form-header">
      <h4>비밀번호 변경</h4>
      <p class="form-description">
        보안을 위해 정기적으로 비밀번호를 변경하는 것을 권장합니다.
      </p>
    </div>

    <n-form
      ref="formRef"
      :model="form"
      :rules="rules"
      label-placement="top"
      require-mark-placement="right-hanging"
    >
      <n-form-item label="현재 비밀번호" path="currentPassword">
        <n-input
          v-model:value="form.currentPassword"
          type="password"
          show-password-on="mousedown"
          placeholder="현재 비밀번호를 입력하세요"
          :disabled="loading"
        />
      </n-form-item>

      <n-form-item label="새 비밀번호" path="newPassword">
        <n-input
          v-model:value="form.newPassword"
          type="password"
          show-password-on="mousedown"
          placeholder="새 비밀번호를 입력하세요"
          :disabled="loading"
          @input="checkPasswordStrength"
        />
        
        <!-- 비밀번호 강도 표시 -->
        <template #feedback>
          <div v-if="passwordStrength && form.newPassword" class="password-strength">
            <div class="strength-meter">
              <div class="strength-label">비밀번호 강도:</div>
              <div class="strength-bar">
                <div 
                  class="strength-fill"
                  :class="strengthClass"
                  :style="{ width: `${(passwordStrength.score + 1) * 20}%` }"
                ></div>
              </div>
              <span class="strength-text" :class="strengthClass">
                {{ strengthText }}
              </span>
            </div>
            
            <!-- 경고 및 제안 -->
            <div v-if="passwordStrength.feedback.warning" class="strength-warning">
              <n-icon size="14" color="#f0a020">
                <Warning />
              </n-icon>
              {{ passwordStrength.feedback.warning }}
            </div>
            
            <div v-if="passwordStrength.feedback.suggestions.length > 0" class="strength-suggestions">
              <div class="suggestions-title">개선 제안:</div>
              <ul>
                <li v-for="suggestion in passwordStrength.feedback.suggestions" :key="suggestion">
                  {{ suggestion }}
                </li>
              </ul>
            </div>
          </div>
        </template>
      </n-form-item>

      <n-form-item label="새 비밀번호 확인" path="confirmPassword">
        <n-input
          v-model:value="form.confirmPassword"
          type="password"
          show-password-on="mousedown"
          placeholder="새 비밀번호를 다시 입력하세요"
          :disabled="loading"
        />
      </n-form-item>

      <div class="form-actions">
        <n-space justify="end">
          <n-button
            type="primary"
            :loading="loading"
            :disabled="!isFormValid"
            @click="handleSubmit"
          >
            <template #icon>
              <n-icon><Key /></n-icon>
            </template>
            비밀번호 변경
          </n-button>
        </n-space>
      </div>
    </n-form>

    <!-- 마지막 변경 정보 -->
    <div v-if="lastPasswordChange" class="last-change-info">
      <n-alert type="info" :show-icon="false">
        <template #header>
          <n-space align="center" size="small">
            <n-icon size="16">
              <Clock />
            </n-icon>
            <span>마지막 비밀번호 변경</span>
          </n-space>
        </template>
        {{ formatLastChange(lastPasswordChange) }}
      </n-alert>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch } from 'vue'
import { useMessage } from 'naive-ui'
import {
  KeySharp as Key,
  WarningSharp as Warning,
  TimeSharp as Clock
} from '@vicons/ionicons5'
import { profileApi } from '@/api/services'
import type { PasswordStrength } from '@/types/api'

// Props
interface Props {
  lastPasswordChange?: string
  loading?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  loading: false
})

// Emits
const emit = defineEmits<{
  success: []
  error: [error: Error]
}>()

// 컴포저블
const message = useMessage()

// 반응형 상태
const loading = ref(false)
const passwordStrength = ref<PasswordStrength | null>(null)
const strengthCheckTimeout = ref<NodeJS.Timeout>()

// 폼 참조
const formRef = ref()

// 폼 데이터
const form = reactive({
  currentPassword: '',
  newPassword: '',
  confirmPassword: ''
})

// 폼 검증 규칙
const rules = {
  currentPassword: {
    required: true,
    message: '현재 비밀번호를 입력해주세요',
    trigger: ['blur', 'input']
  },
  newPassword: [
    {
      required: true,
      message: '새 비밀번호를 입력해주세요',
      trigger: ['blur', 'input']
    },
    {
      min: 8,
      message: '비밀번호는 최소 8자 이상이어야 합니다',
      trigger: ['blur', 'input']
    },
    {
      validator: (rule: any, value: string) => {
        if (value === form.currentPassword) {
          return new Error('새 비밀번호는 현재 비밀번호와 달라야 합니다')
        }
        return true
      },
      trigger: ['blur', 'input']
    }
  ],
  confirmPassword: [
    {
      required: true,
      message: '비밀번호 확인을 입력해주세요',
      trigger: ['blur', 'input']
    },
    {
      validator: (rule: any, value: string) => {
        if (value !== form.newPassword) {
          return new Error('비밀번호가 일치하지 않습니다')
        }
        return true
      },
      trigger: ['blur', 'input']
    }
  ]
}

// 계산된 속성
const isFormValid = computed(() => {
  return form.currentPassword.length > 0 &&
         form.newPassword.length >= 8 &&
         form.confirmPassword === form.newPassword &&
         form.newPassword !== form.currentPassword
})

const strengthClass = computed(() => {
  if (!passwordStrength.value) return 'strength-weak'
  
  const score = passwordStrength.value.score
  if (score >= 4) return 'strength-very-strong'
  if (score >= 3) return 'strength-strong'
  if (score >= 2) return 'strength-medium'
  if (score >= 1) return 'strength-fair'
  return 'strength-weak'
})

const strengthText = computed(() => {
  if (!passwordStrength.value) return ''
  
  const score = passwordStrength.value.score
  const texts = ['매우 약함', '약함', '보통', '강함', '매우 강함']
  return texts[score] || '알 수 없음'
})

// 메서드
const checkPasswordStrength = () => {
  // 기존 타이머 클리어
  if (strengthCheckTimeout.value) {
    clearTimeout(strengthCheckTimeout.value)
  }
  
  // 비밀번호가 너무 짧으면 체크하지 않음
  if (form.newPassword.length < 4) {
    passwordStrength.value = null
    return
  }
  
  // 디바운스로 API 호출 최적화
  strengthCheckTimeout.value = setTimeout(async () => {
    try {
      passwordStrength.value = await profileApi.checkPasswordStrength(form.newPassword)
    } catch (error) {
      console.error('비밀번호 강도 체크 실패:', error)
      // 에러 시 기본 강도 표시
      passwordStrength.value = {
        score: 0,
        feedback: {
          warning: '비밀번호 강도를 확인할 수 없습니다',
          suggestions: []
        },
        crackTime: {
          offlineSlowHashing: '알 수 없음',
          onlineThrottling: '알 수 없음'
        }
      }
    }
  }, 500)
}

const handleSubmit = async () => {
  try {
    await formRef.value?.validate()
    
    loading.value = true
    
    await profileApi.changePassword({
      currentPassword: form.currentPassword,
      newPassword: form.newPassword,
      confirmPassword: form.confirmPassword
    })
    
    // 폼 초기화
    form.currentPassword = ''
    form.newPassword = ''
    form.confirmPassword = ''
    passwordStrength.value = null
    
    message.success('비밀번호가 성공적으로 변경되었습니다')
    emit('success')
    
  } catch (error: any) {
    console.error('비밀번호 변경 실패:', error)
    message.error(error.message || '비밀번호 변경에 실패했습니다')
    emit('error', error)
  } finally {
    loading.value = false
  }
}

const formatLastChange = (dateString: string) => {
  const date = new Date(dateString)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24))
  
  if (diffDays === 0) {
    return '오늘'
  } else if (diffDays === 1) {
    return '어제'
  } else if (diffDays < 30) {
    return `${diffDays}일 전`
  } else if (diffDays < 365) {
    const diffMonths = Math.floor(diffDays / 30)
    return `${diffMonths}개월 전`
  } else {
    const diffYears = Math.floor(diffDays / 365)
    return `${diffYears}년 전`
  }
}

// 비밀번호 필드 변경 시 강도 체크 초기화
watch(() => form.newPassword, (newVal, oldVal) => {
  if (newVal !== oldVal && newVal.length === 0) {
    passwordStrength.value = null
  }
})
</script>

<style scoped lang="scss">
.password-change-form {
  .form-header {
    margin-bottom: 24px;

    h4 {
      margin: 0 0 8px 0;
      font-size: 18px;
      font-weight: 500;
      color: var(--text-color-1);
    }

    .form-description {
      margin: 0;
      font-size: 14px;
      color: var(--text-color-2);
      line-height: 1.4;
    }
  }

  .password-strength {
    margin-top: 8px;

    .strength-meter {
      display: flex;
      align-items: center;
      gap: 8px;
      margin-bottom: 8px;

      .strength-label {
        font-size: 12px;
        color: var(--text-color-2);
        white-space: nowrap;
      }

      .strength-bar {
        flex: 1;
        height: 4px;
        background: var(--border-color);
        border-radius: 2px;
        overflow: hidden;

        .strength-fill {
          height: 100%;
          transition: all 0.3s ease;
          border-radius: 2px;

          &.strength-weak {
            background: #e74c3c;
          }

          &.strength-fair {
            background: #f39c12;
          }

          &.strength-medium {
            background: #f1c40f;
          }

          &.strength-strong {
            background: #27ae60;
          }

          &.strength-very-strong {
            background: #2ecc71;
          }
        }
      }

      .strength-text {
        font-size: 12px;
        font-weight: 500;
        white-space: nowrap;

        &.strength-weak {
          color: #e74c3c;
        }

        &.strength-fair {
          color: #f39c12;
        }

        &.strength-medium {
          color: #f1c40f;
        }

        &.strength-strong {
          color: #27ae60;
        }

        &.strength-very-strong {
          color: #2ecc71;
        }
      }
    }

    .strength-warning {
      display: flex;
      align-items: center;
      gap: 6px;
      margin-bottom: 8px;
      font-size: 12px;
      color: #f0a020;
    }

    .strength-suggestions {
      font-size: 12px;
      color: var(--text-color-3);

      .suggestions-title {
        margin-bottom: 4px;
        font-weight: 500;
      }

      ul {
        margin: 0;
        padding-left: 16px;

        li {
          margin-bottom: 2px;
          line-height: 1.4;
        }
      }
    }
  }

  .form-actions {
    margin-top: 24px;
    padding-top: 16px;
    border-top: 1px solid var(--border-color);
  }

  .last-change-info {
    margin-top: 24px;
  }
}

// 반응형 디자인
@media (max-width: 480px) {
  .password-change-form {
    .password-strength {
      .strength-meter {
        flex-direction: column;
        align-items: stretch;
        gap: 4px;

        .strength-label {
          font-size: 11px;
        }

        .strength-text {
          font-size: 11px;
          text-align: center;
        }
      }

      .strength-suggestions {
        ul {
          padding-left: 12px;
        }
      }
    }

    .form-actions {
      .n-space {
        :deep(.n-space-item) {
          flex: 1;
          
          .n-button {
            width: 100%;
          }
        }
      }
    }
  }
}
</style>