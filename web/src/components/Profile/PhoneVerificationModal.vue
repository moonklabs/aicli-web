<template>
  <n-modal
    v-model:show="showModal"
    preset="card"
    title="전화번호 인증"
    size="medium"
    :bordered="false"
    :closable="!processing"
    :mask-closable="!processing"
  >
    <div class="phone-verification-content">
      <!-- 단계 진행 표시 -->
      <n-steps
        :current="currentStep"
        :status="stepStatus"
        size="small"
        style="margin-bottom: 24px;"
      >
        <n-step title="전화번호 확인" />
        <n-step title="인증번호 발송" />
        <n-step title="인증 완료" />
      </n-steps>

      <!-- 단계 1: 전화번호 확인 -->
      <div v-if="currentStep === 1" class="step-content">
        <div class="step-description">
          <p>인증할 전화번호를 확인하고 인증번호 발송을 요청하세요.</p>
        </div>

        <div class="phone-info">
          <n-form
            ref="phoneFormRef"
            :model="phoneForm"
            :rules="phoneFormRules"
            label-placement="top"
          >
            <n-form-item label="전화번호" path="phone">
              <n-input-group>
                <n-select
                  v-model:value="phoneForm.countryCode"
                  :options="countryCodeOptions"
                  style="width: 120px"
                  :disabled="processing"
                />
                <n-input
                  v-model:value="phoneForm.phoneNumber"
                  placeholder="전화번호를 입력하세요"
                  :disabled="processing"
                />
              </n-input-group>
            </n-form-item>
          </n-form>

          <div class="phone-display">
            <div class="formatted-phone">
              <span class="phone-label">인증할 번호:</span>
              <span class="phone-number">{{ formattedPhoneNumber }}</span>
            </div>
          </div>
        </div>

        <div class="verification-info">
          <n-alert type="info" :show-icon="true" size="small">
            <template #header>SMS 인증 안내</template>
            <ul>
              <li>입력한 전화번호로 6자리 인증번호가 발송됩니다</li>
              <li>인증번호는 3분간 유효합니다</li>
              <li>SMS 요금이 발생할 수 있습니다</li>
            </ul>
          </n-alert>
        </div>
      </div>

      <!-- 단계 2: 인증번호 입력 -->
      <div v-if="currentStep === 2" class="step-content">
        <div class="step-description">
          <n-result status="info" title="인증번호 발송됨" size="small">
            <template #footer>
              <div class="verification-form">
                <p>
                  <strong>{{ formattedPhoneNumber }}</strong>로 인증번호를 발송했습니다.
                </p>
                
                <n-form
                  ref="codeFormRef"
                  :model="codeForm"
                  :rules="codeFormRules"
                  style="margin: 20px 0;"
                >
                  <n-form-item path="code">
                    <n-input
                      v-model:value="codeForm.code"
                      placeholder="6자리 인증번호"
                      maxlength="6"
                      :style="{ 
                        fontSize: '18px', 
                        textAlign: 'center', 
                        letterSpacing: '4px' 
                      }"
                      :disabled="processing"
                      @keyup.enter="verifyCode"
                    />
                  </n-form-item>
                </n-form>

                <div class="timer-section">
                  <n-space align="center" justify="center">
                    <n-icon size="16" :color="timeLeft > 0 ? '#18a058' : '#e74c3c'">
                      <Timer />
                    </n-icon>
                    <span :class="{ 'expired': timeLeft <= 0 }">
                      {{ formatTime(timeLeft) }}
                    </span>
                  </n-space>
                </div>

                <div class="resend-section">
                  <n-space align="center" justify="center">
                    <span class="resend-text">인증번호를 받지 못하셨나요?</span>
                    <n-button
                      text
                      type="primary"
                      :disabled="resendCooldown > 0"
                      :loading="resending"
                      @click="resendVerificationCode"
                    >
                      {{ resendCooldown > 0 ? `${resendCooldown}초 후 재발송` : '다시 발송' }}
                    </n-button>
                  </n-space>
                </div>

                <div class="verification-help">
                  <n-alert type="warning" :show-icon="true" size="small">
                    <template #header">SMS를 받지 못하셨나요?</template>
                    <ul>
                      <li>전화번호가 정확한지 확인해주세요</li>
                      <li>스팸 차단 설정을 확인해주세요</li>
                      <li>통신 상태가 좋지 않은 경우 지연될 수 있습니다</li>
                    </ul>
                  </n-alert>
                </div>
              </div>
            </template>
          </n-result>
        </div>
      </div>

      <!-- 단계 3: 인증 완료 -->
      <div v-if="currentStep === 3" class="step-content">
        <div class="step-description">
          <n-result status="success" title="전화번호 인증 완료!" size="small">
            <template #footer>
              <div class="completion-info">
                <p>전화번호 인증이 성공적으로 완료되었습니다.</p>
                
                <div class="verified-phone">
                  <div class="phone-info-card">
                    <n-icon size="20" color="#18a058">
                      <CheckmarkCircle />
                    </n-icon>
                    <span class="phone-number">{{ formattedPhoneNumber }}</span>
                    <n-tag type="success" size="small">인증됨</n-tag>
                  </div>
                </div>

                <div class="next-steps">
                  <n-alert type="success" :show-icon="true" size="small">
                    <template #header">인증 완료</template>
                    <p>이제 SMS 알림을 받을 수 있고, 2단계 인증에서 SMS를 사용할 수 있습니다.</p>
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
            :disabled="!isPhoneFormValid"
            @click="sendVerificationCode"
          >
            인증번호 발송
          </n-button>
        </template>

        <!-- 단계 2 액션 -->
        <template v-if="currentStep === 2">
          <n-button @click="goBackToPhoneForm" :disabled="processing">번호 수정</n-button>
          <n-button
            type="primary"
            :loading="processing"
            :disabled="!isCodeFormValid"
            @click="verifyCode"
          >
            인증 확인
          </n-button>
        </template>

        <!-- 단계 3 액션 -->
        <template v-if="currentStep === 3">
          <n-button type="primary" @click="finishVerification">확인</n-button>
        </template>
      </n-space>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch, onUnmounted } from 'vue'
import { useMessage } from 'naive-ui'
import {
  TimeSharp as Timer,
  CheckmarkCircleSharp as CheckmarkCircle
} from '@vicons/ionicons5'
import { profileApi } from '@/api/services'

// Props
interface Props {
  show: boolean
  phone?: string
}

const props = withDefaults(defineProps<Props>(), {
  phone: ''
})

// Emits
const emit = defineEmits<{
  'update:show': [value: boolean]
  verify: [code: string]
  success: [phone: string]
}>()

// 컴포저블
const message = useMessage()

// 반응형 상태
const processing = ref(false)
const resending = ref(false)
const currentStep = ref(1)
const stepStatus = ref<'process' | 'finish' | 'error' | 'wait'>('process')
const timeLeft = ref(180) // 3분 (180초)
const resendCooldown = ref(0)
const verificationTimer = ref<NodeJS.Timeout>()
const resendTimer = ref<NodeJS.Timeout>()

// 폼 참조
const phoneFormRef = ref()
const codeFormRef = ref()

// 폼 데이터
const phoneForm = reactive({
  countryCode: '+82',
  phoneNumber: ''
})

const codeForm = reactive({
  code: ''
})

// 국가 코드 옵션
const countryCodeOptions = [
  { label: '+82 (한국)', value: '+82' },
  { label: '+1 (미국)', value: '+1' },
  { label: '+81 (일본)', value: '+81' },
  { label: '+86 (중국)', value: '+86' },
  { label: '+44 (영국)', value: '+44' }
]

// 모달 표시 상태 (v-model)
const showModal = computed({
  get: () => props.show,
  set: (value) => emit('update:show', value)
})

// 폼 검증 규칙
const phoneFormRules = {
  phoneNumber: [
    {
      required: true,
      message: '전화번호를 입력해주세요',
      trigger: ['blur', 'input']
    },
    {
      pattern: /^[0-9-]{8,15}$/,
      message: '올바른 전화번호 형식을 입력해주세요',
      trigger: ['blur', 'input']
    }
  ]
}

const codeFormRules = {
  code: [
    {
      required: true,
      message: '인증번호를 입력해주세요',
      trigger: ['blur', 'input']
    },
    {
      len: 6,
      pattern: /^\d{6}$/,
      message: '6자리 숫자를 입력해주세요',
      trigger: ['blur', 'input']
    }
  ]
}

// 계산된 속성
const formattedPhoneNumber = computed(() => {
  if (!phoneForm.phoneNumber) return ''
  return `${phoneForm.countryCode} ${phoneForm.phoneNumber}`
})

const isPhoneFormValid = computed(() => {
  return phoneForm.phoneNumber.length >= 8 &&
         /^[0-9-]{8,15}$/.test(phoneForm.phoneNumber)
})

const isCodeFormValid = computed(() => {
  return codeForm.code.length === 6 && /^\d{6}$/.test(codeForm.code)
})

// 메서드
const sendVerificationCode = async () => {
  try {
    await phoneFormRef.value?.validate()
    
    processing.value = true
    
    const fullPhoneNumber = formattedPhoneNumber.value
    await profileApi.requestPhoneVerification(fullPhoneNumber)
    
    currentStep.value = 2
    message.success('인증번호가 발송되었습니다')
    
    // 타이머 시작
    startVerificationTimer()
    
  } catch (error: any) {
    console.error('인증번호 발송 실패:', error)
    message.error(error.message || '인증번호 발송에 실패했습니다')
  } finally {
    processing.value = false
  }
}

const verifyCode = async () => {
  try {
    await codeFormRef.value?.validate()
    
    processing.value = true
    
    const fullPhoneNumber = formattedPhoneNumber.value
    await profileApi.confirmPhoneVerification(fullPhoneNumber, codeForm.code)
    
    currentStep.value = 3
    stepStatus.value = 'finish'
    
    // 타이머 정리
    clearTimers()
    
    message.success('전화번호 인증이 완료되었습니다')
    emit('verify', codeForm.code)
    
  } catch (error: any) {
    console.error('인증 코드 확인 실패:', error)
    message.error(error.message || '인증번호가 올바르지 않습니다')
  } finally {
    processing.value = false
  }
}

const resendVerificationCode = async () => {
  if (resendCooldown.value > 0) return
  
  resending.value = true
  try {
    const fullPhoneNumber = formattedPhoneNumber.value
    await profileApi.requestPhoneVerification(fullPhoneNumber)
    
    message.success('인증번호를 다시 발송했습니다')
    
    // 타이머 리셋
    timeLeft.value = 180
    startVerificationTimer()
    startResendCooldown()
    
  } catch (error: any) {
    console.error('인증번호 재발송 실패:', error)
    message.error(error.message || '인증번호 재발송에 실패했습니다')
  } finally {
    resending.value = false
  }
}

const startVerificationTimer = () => {
  clearTimeout(verificationTimer.value)
  
  verificationTimer.value = setInterval(() => {
    timeLeft.value--
    
    if (timeLeft.value <= 0) {
      clearInterval(verificationTimer.value)
      message.warning('인증번호가 만료되었습니다. 다시 발송해주세요.')
    }
  }, 1000)
}

const startResendCooldown = () => {
  resendCooldown.value = 30 // 30초 쿨다운
  
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

const clearTimers = () => {
  if (verificationTimer.value) {
    clearInterval(verificationTimer.value)
  }
  if (resendTimer.value) {
    clearInterval(resendTimer.value)
  }
}

const formatTime = (seconds: number) => {
  const minutes = Math.floor(seconds / 60)
  const remainingSeconds = seconds % 60
  return `${minutes}:${remainingSeconds.toString().padStart(2, '0')}`
}

const goBackToPhoneForm = () => {
  currentStep.value = 1
  stepStatus.value = 'process'
  clearTimers()
}

const finishVerification = () => {
  emit('success', formattedPhoneNumber.value)
  closeModal()
}

const closeModal = () => {
  clearTimers()
  emit('update:show', false)
}

const resetForm = () => {
  phoneForm.phoneNumber = ''
  phoneForm.countryCode = '+82'
  codeForm.code = ''
  currentStep.value = 1
  stepStatus.value = 'process'
  timeLeft.value = 180
  resendCooldown.value = 0
  clearTimers()
}

// Props 변경 시 폼 초기화
watch(() => props.phone, (newPhone) => {
  if (newPhone) {
    // 기존 전화번호가 있다면 파싱해서 설정
    if (newPhone.startsWith('+82')) {
      phoneForm.countryCode = '+82'
      phoneForm.phoneNumber = newPhone.replace('+82', '').trim()
    } else {
      phoneForm.phoneNumber = newPhone
    }
  }
}, { immediate: true })

// 모달이 닫힐 때 폼 초기화
watch(() => props.show, (newShow) => {
  if (!newShow) {
    resetForm()
  }
})

// 컴포넌트 언마운트 시 타이머 정리
onUnmounted(() => {
  clearTimers()
})
</script>

<style scoped lang="scss">
.phone-verification-content {
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

    .phone-info {
      margin-bottom: 16px;

      .phone-display {
        margin-top: 12px;
        padding: 12px;
        background: var(--card-color);
        border-radius: 6px;
        border: 1px solid var(--border-color);

        .formatted-phone {
          display: flex;
          align-items: center;
          gap: 8px;

          .phone-label {
            font-size: 14px;
            color: var(--text-color-2);
          }

          .phone-number {
            font-size: 16px;
            font-weight: 500;
            color: var(--primary-color);
          }
        }
      }
    }

    .verification-info {
      ul {
        margin: 8px 0 0 0;
        padding-left: 16px;

        li {
          margin-bottom: 4px;
          font-size: 12px;
          color: var(--text-color-3);
        }
      }
    }

    .verification-form {
      text-align: center;

      p {
        margin: 0 0 16px 0;
        font-size: 14px;
        line-height: 1.5;

        strong {
          color: var(--primary-color);
        }
      }

      .timer-section {
        margin: 16px 0;
        padding: 8px;
        background: var(--card-color);
        border-radius: 6px;
        border: 1px solid var(--border-color);

        span {
          font-size: 16px;
          font-weight: 500;
          color: var(--success-color);

          &.expired {
            color: var(--error-color);
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

      .verification-help {
        margin-top: 16px;

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
    }

    .completion-info {
      text-align: center;

      p {
        margin: 0 0 16px 0;
        font-size: 14px;
        color: var(--text-color-1);
      }

      .verified-phone {
        margin: 16px 0;

        .phone-info-card {
          display: flex;
          align-items: center;
          justify-content: center;
          gap: 8px;
          padding: 12px;
          background: var(--card-color);
          border-radius: 6px;
          border: 1px solid var(--success-color);

          .phone-number {
            font-size: 16px;
            font-weight: 500;
            color: var(--text-color-1);
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
  .phone-verification-content {
    .step-content {
      .phone-info {
        .phone-display {
          .formatted-phone {
            flex-direction: column;
            align-items: flex-start;
            gap: 4px;
          }
        }
      }

      .verification-form {
        .resend-section {
          .n-space {
            flex-direction: column;
            gap: 8px;
          }
        }
      }

      .completion-info {
        .verified-phone {
          .phone-info-card {
            flex-direction: column;
            gap: 8px;
          }
        }
      }
    }
  }
}
</style>