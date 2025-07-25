<template>
  <n-modal
    v-model:show="showModal"
    preset="card"
    title="이메일 주소 변경"
    size="medium"
    :bordered="false"
    :closable="!processing"
    :mask-closable="!processing"
  >
    <div class="email-change-content">
      <!-- 단계 진행 표시 -->
      <n-steps
        :current="currentStep"
        :status="stepStatus"
        size="small"
        style="margin-bottom: 24px;"
      >
        <n-step title="새 이메일 입력" />
        <n-step title="인증 이메일 발송" />
        <n-step title="인증 완료" />
      </n-steps>

      <!-- 단계 1: 새 이메일 입력 -->
      <div v-if="currentStep === 1" class="step-content">
        <div class="step-description">
          <p>새로운 이메일 주소를 입력하고 현재 비밀번호를 확인해주세요.</p>
        </div>

        <n-form
          ref="emailFormRef"
          :model="emailForm"
          :rules="emailFormRules"
          label-placement="top"
        >
          <n-form-item label="현재 이메일" path="currentEmail">
            <n-input
              :value="currentEmail"
              readonly
              disabled
            />
          </n-form-item>

          <n-form-item label="새 이메일 주소" path="newEmail">
            <n-input
              v-model:value="emailForm.newEmail"
              type="email"
              placeholder="새로운 이메일 주소를 입력하세요"
              :disabled="processing"
            />
          </n-form-item>

          <n-form-item label="비밀번호 확인" path="password">
            <n-input
              v-model:value="emailForm.password"
              type="password"
              placeholder="현재 비밀번호를 입력하세요"
              show-password-on="mousedown"
              :disabled="processing"
            />
          </n-form-item>
        </n-form>

        <div class="security-notice">
          <n-alert type="info" :show-icon="true" size="small">
            <template #header>보안 알림</template>
            <p>이메일 변경 시 보안을 위해 다음과 같은 절차가 진행됩니다:</p>
            <ul>
              <li>새 이메일 주소로 인증 메일이 발송됩니다</li>
              <li>현재 이메일로도 변경 알림이 발송됩니다</li>
              <li>인증 완료 후 이메일이 변경됩니다</li>
            </ul>
          </n-alert>
        </div>
      </div>

      <!-- 단계 2: 인증 이메일 발송 -->
      <div v-if="currentStep === 2" class="step-content">
        <div class="step-description">
          <n-result status="info" title="인증 이메일 발송됨" size="small">
            <template #footer>
              <div class="email-sent-info">
                <p>
                  <strong>{{ emailForm.newEmail }}</strong>로 인증 이메일을 발송했습니다.
                </p>
                <p>이메일의 인증 링크를 클릭하여 이메일 변경을 완료해주세요.</p>

                <div class="verification-help">
                  <n-alert type="warning" :show-icon="true" size="small">
                    <template #header>이메일을 받지 못하셨나요?</template>
                    <ul>
                      <li>스팸 폴더를 확인해주세요</li>
                      <li>이메일 주소가 정확한지 확인해주세요</li>
                      <li>몇 분 후에도 받지 못하시면 다시 발송해주세요</li>
                    </ul>
                  </n-alert>
                </div>

                <div class="resend-section">
                  <n-space align="center">
                    <span class="resend-text">이메일을 받지 못하셨나요?</span>
                    <n-button
                      text
                      type="primary"
                      :disabled="resendCooldown > 0"
                      :loading="resending"
                      @click="resendVerificationEmail"
                    >
                      {{ resendCooldown > 0 ? `${resendCooldown}초 후 재발송` : '다시 발송' }}
                    </n-button>
                  </n-space>
                </div>

                <div class="manual-check">
                  <n-button
                    type="primary"
                    ghost
                    :loading="checking"
                    @click="checkVerificationStatus"
                  >
                    인증 완료 확인
                  </n-button>
                </div>
              </div>
            </template>
          </n-result>
        </div>
      </div>

      <!-- 단계 3: 인증 완료 -->
      <div v-if="currentStep === 3" class="step-content">
        <div class="step-description">
          <n-result status="success" title="이메일 변경 완료!" size="small">
            <template #footer>
              <div class="completion-info">
                <p>이메일 주소가 성공적으로 변경되었습니다.</p>
                <div class="email-info">
                  <div class="email-change-summary">
                    <span class="old-email">{{ currentEmail }}</span>
                    <n-icon size="16" style="margin: 0 8px;">
                      <ArrowForward />
                    </n-icon>
                    <span class="new-email">{{ emailForm.newEmail }}</span>
                  </div>
                </div>

                <div class="next-steps">
                  <n-alert type="success" :show-icon="true" size="small">
                    <template #header">다음 로그인부터 새 이메일을 사용하세요</template>
                    <p>보안을 위해 모든 기기에서 자동으로 로그아웃됩니다. 새 이메일로 다시 로그인해주세요.</p>
                  </n-alert>
                </div>
              </div>
            </template>
          </n-result>
        </div>
      </div>
    </div>

    <template #action>
      <n-space justify="end">
        <!-- 단계 1 액션 -->
        <template v-if="currentStep === 1">
          <n-button @click="closeModal" :disabled="processing">취소</n-button>
          <n-button
            type="primary"
            :loading="processing"
            :disabled="!isEmailFormValid"
            @click="submitEmailChange"
          >
            인증 이메일 발송
          </n-button>
        </template>

        <!-- 단계 2 액션 -->
        <template v-if="currentStep === 2">
          <n-button @click="goBackToEmailForm" :disabled="processing">이메일 수정</n-button>
          <n-button @click="closeModal" :disabled="processing">나중에 하기</n-button>
        </template>

        <!-- 단계 3 액션 -->
        <template v-if="currentStep === 3">
          <n-button type="primary" @click="finishEmailChange">확인</n-button>
        </template>
      </n-space>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useMessage } from 'naive-ui'
import { ArrowForwardSharp as ArrowForward } from '@vicons/ionicons5'
import { profileApi } from '@/api/services'
import { useUserStore } from '@/stores/user'

// Props
interface Props {
  show: boolean
  currentEmail?: string
}

const props = withDefaults(defineProps<Props>(), {
  currentEmail: '',
})

// Emits
const emit = defineEmits<{
  'update:show': [value: boolean]
  confirm: [newEmail: string, password: string]
  success: [newEmail: string]
}>()

// 컴포저블
const message = useMessage()
const userStore = useUserStore()

// 반응형 상태
const processing = ref(false)
const resending = ref(false)
const checking = ref(false)
const currentStep = ref(1)
const stepStatus = ref<'process' | 'finish' | 'error' | 'wait'>('process')
const resendCooldown = ref(0)
const resendTimer = ref<NodeJS.Timeout>()

// 폼 참조
const emailFormRef = ref()

// 폼 데이터
const emailForm = reactive({
  newEmail: '',
  password: '',
})

// 모달 표시 상태 (v-model)
const showModal = computed({
  get: () => props.show,
  set: (value) => emit('update:show', value),
})

// 폼 검증 규칙
const emailFormRules = {
  newEmail: [
    {
      required: true,
      message: '새 이메일 주소를 입력해주세요',
      trigger: ['blur', 'input'],
    },
    {
      type: 'email',
      message: '올바른 이메일 형식을 입력해주세요',
      trigger: ['blur', 'input'],
    },
    {
      validator: (rule: any, value: string) => {
        if (value === props.currentEmail) {
          return new Error('현재 이메일과 다른 주소를 입력해주세요')
        }
        return true
      },
      trigger: ['blur', 'input'],
    },
  ],
  password: {
    required: true,
    message: '현재 비밀번호를 입력해주세요',
    trigger: ['blur', 'input'],
  },
}

// 계산된 속성
const isEmailFormValid = computed(() => {
  return emailForm.newEmail.length > 0 &&
         emailForm.password.length > 0 &&
         emailForm.newEmail !== props.currentEmail &&
         /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(emailForm.newEmail)
})

// 메서드
const submitEmailChange = async () => {
  try {
    await emailFormRef.value?.validate()

    processing.value = true

    await profileApi.requestEmailChange(emailForm.newEmail, emailForm.password)

    currentStep.value = 2
    emit('confirm', emailForm.newEmail, emailForm.password)
    message.success('인증 이메일이 발송되었습니다')

    // 주기적으로 인증 상태 확인 (실제 구현에서는 WebSocket 사용 권장)
    startVerificationCheck()

  } catch (error: any) {
    console.error('이메일 변경 요청 실패:', error)
    message.error(error.message || '이메일 변경 요청에 실패했습니다')
  } finally {
    processing.value = false
  }
}

const resendVerificationEmail = async () => {
  if (resendCooldown.value > 0) return

  resending.value = true
  try {
    await profileApi.requestEmailChange(emailForm.newEmail, emailForm.password)
    message.success('인증 이메일을 다시 발송했습니다')

    // 재발송 쿨다운 시작
    startResendCooldown()

  } catch (error: any) {
    console.error('이메일 재발송 실패:', error)
    message.error(error.message || '이메일 재발송에 실패했습니다')
  } finally {
    resending.value = false
  }
}

const checkVerificationStatus = async () => {
  checking.value = true
  try {
    // 실제 구현에서는 인증 상태 확인 API 호출
    await new Promise(resolve => setTimeout(resolve, 1000))

    // 임시로 50% 확률로 성공 시뮬레이션
    const isVerified = Math.random() > 0.5

    if (isVerified) {
      currentStep.value = 3
      stepStatus.value = 'finish'

      // 사용자 스토어 업데이트
      userStore.updateUser({ email: emailForm.newEmail })

      message.success('이메일 인증이 완료되었습니다')
    } else {
      message.info('아직 인증이 완료되지 않았습니다. 이메일을 확인해주세요.')
    }

  } catch (error: any) {
    console.error('인증 상태 확인 실패:', error)
    message.error('인증 상태 확인에 실패했습니다')
  } finally {
    checking.value = false
  }
}

const startVerificationCheck = () => {
  // 실제 구현에서는 WebSocket이나 polling을 사용
  const checkInterval = setInterval(async () => {
    if (currentStep.value !== 2) {
      clearInterval(checkInterval)
      return
    }

    try {
      // 실제 구현에서는 인증 상태 확인 API 호출
      const isVerified = Math.random() > 0.9 // 낮은 확률로 자동 완료 시뮬레이션

      if (isVerified) {
        clearInterval(checkInterval)
        currentStep.value = 3
        stepStatus.value = 'finish'

        // 사용자 스토어 업데이트
        userStore.updateUser({ email: emailForm.newEmail })

        message.success('이메일 인증이 완료되었습니다')
      }
    } catch (error) {
      console.error('자동 인증 확인 실패:', error)
    }
  }, 5000) // 5초마다 확인

  // 10분 후 자동 확인 중지
  setTimeout(() => {
    clearInterval(checkInterval)
  }, 600000)
}

const startResendCooldown = () => {
  resendCooldown.value = 60 // 60초 쿨다운

  if (resendTimer.value) {
    clearInterval(resendTimer.value)
  }

  resendTimer.value = setInterval(() => {
    resendCooldown.value--

    if (resendCooldown.value <= 0) {
      clearInterval(resendTimer.value)
    }
  }, 1000)
}

const goBackToEmailForm = () => {
  currentStep.value = 1
  stepStatus.value = 'process'
}

const finishEmailChange = () => {
  emit('success', emailForm.newEmail)
  closeModal()

  // 모든 기기에서 로그아웃 처리 (실제 구현에서 필요)
  message.info('보안을 위해 모든 기기에서 로그아웃됩니다. 새 이메일로 다시 로그인해주세요.')
}

const closeModal = () => {
  // 쿨다운 타이머 정리
  if (resendTimer.value) {
    clearInterval(resendTimer.value)
  }

  emit('update:show', false)
}

const resetForm = () => {
  emailForm.newEmail = ''
  emailForm.password = ''
  currentStep.value = 1
  stepStatus.value = 'process'
  resendCooldown.value = 0

  if (resendTimer.value) {
    clearInterval(resendTimer.value)
  }
}

// 모달이 닫힐 때 폼 초기화
watch(() => props.show, (newShow) => {
  if (!newShow) {
    resetForm()
  }
})
</script>

<style scoped lang="scss">
.email-change-content {
  .step-content {
    .step-description {
      margin-bottom: 24px;

      p {
        margin: 0;
        font-size: 14px;
        color: var(--text-color-2);
        line-height: 1.5;
      }
    }

    .security-notice {
      margin-top: 16px;

      p {
        margin: 0 0 8px 0;
        font-size: 13px;
      }

      ul {
        margin: 0;
        padding-left: 16px;

        li {
          margin-bottom: 4px;
          font-size: 12px;
          color: var(--text-color-3);
        }
      }
    }

    .email-sent-info {
      text-align: center;

      p {
        margin: 0 0 16px 0;
        font-size: 14px;
        line-height: 1.5;

        strong {
          color: var(--primary-color);
        }
      }

      .verification-help {
        margin: 16px 0;

        ul {
          text-align: left;
          margin: 8px 0 0 0;
          padding-left: 16px;

          li {
            margin-bottom: 4px;
            font-size: 12px;
          }
        }
      }

      .resend-section {
        margin: 16px 0;
        padding: 12px;
        background: var(--card-color);
        border-radius: 6px;
        border: 1px solid var(--border-color);

        .resend-text {
          font-size: 13px;
          color: var(--text-color-2);
        }
      }

      .manual-check {
        margin-top: 16px;
      }
    }

    .completion-info {
      text-align: center;

      p {
        margin: 0 0 16px 0;
        font-size: 14px;
        color: var(--text-color-1);
      }

      .email-info {
        margin: 16px 0;

        .email-change-summary {
          display: flex;
          align-items: center;
          justify-content: center;
          padding: 12px;
          background: var(--card-color);
          border-radius: 6px;
          border: 1px solid var(--border-color);

          .old-email {
            color: var(--text-color-3);
            text-decoration: line-through;
          }

          .new-email {
            color: var(--primary-color);
            font-weight: 500;
          }
        }
      }

      .next-steps {
        margin-top: 16px;

        p {
          margin: 0;
          font-size: 12px;
        }
      }
    }
  }
}

// 반응형 디자인
@media (max-width: 480px) {
  .email-change-content {
    .step-content {
      .email-sent-info {
        .resend-section {
          .n-space {
            flex-direction: column;
            gap: 8px;
          }
        }
      }

      .completion-info {
        .email-info {
          .email-change-summary {
            flex-direction: column;
            gap: 8px;
          }
        }
      }
    }
  }
}
</style>