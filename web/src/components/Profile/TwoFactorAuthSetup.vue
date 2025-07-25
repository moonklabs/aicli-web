<template>
  <div class="two-factor-auth">
    <div class="setup-header">
      <h4>2단계 인증 (2FA)</h4>
      <p class="setup-description">
        계정 보안을 강화하기 위해 2단계 인증을 설정하세요. 
        Google Authenticator, Authy 등의 앱을 사용할 수 있습니다.
      </p>
    </div>

    <!-- 2FA 상태 표시 -->
    <div class="status-section">
      <n-alert 
        :type="settings?.enabled ? 'success' : 'warning'" 
        :show-icon="true"
      >
        <template #header>
          <span>2단계 인증 {{ settings?.enabled ? '활성화됨' : '비활성화됨' }}</span>
        </template>
        <div v-if="settings?.enabled">
          <p>계정이 2단계 인증으로 보호되고 있습니다.</p>
          <p v-if="settings.lastUsed">
            마지막 사용: {{ formatDate(settings.lastUsed) }}
          </p>
        </div>
        <div v-else>
          <p>보안을 강화하기 위해 2단계 인증을 활성화하는 것을 권장합니다.</p>
        </div>
      </n-alert>
    </div>

    <!-- 2FA 비활성화 상태 - 설정하기 -->
    <div v-if="!settings?.enabled" class="setup-section">
      <div v-if="!showSetup" class="setup-start">
        <n-button
          type="primary"
          size="large"
          :loading="loading"
          @click="startSetup"
        >
          <template #icon>
            <n-icon><Shield /></n-icon>
          </template>
          2단계 인증 설정하기
        </n-button>
      </div>

      <!-- 설정 과정 -->
      <div v-else class="setup-process">
        <n-steps
          :current="currentStep"
          :status="stepStatus"
          size="small"
        >
          <n-step title="앱 설치" />
          <n-step title="QR 코드 스캔" />
          <n-step title="인증 코드 확인" />
          <n-step title="완료" />
        </n-steps>

        <!-- 단계 1: 앱 설치 안내 -->
        <div v-if="currentStep === 1" class="step-content">
          <h5>1단계: 인증 앱 설치</h5>
          <p>아래 앱 중 하나를 스마트폰에 설치하세요:</p>
          <div class="app-list">
            <div class="app-item">
              <n-icon size="24" color="#4285f4">
                <Smartphone />
              </n-icon>
              <div class="app-info">
                <strong>Google Authenticator</strong>
                <p>Google에서 제공하는 공식 인증 앱</p>
              </div>
            </div>
            <div class="app-item">
              <n-icon size="24" color="#ff6b35">
                <Key />
              </n-icon>
              <div class="app-info">
                <strong>Authy</strong>
                <p>백업 및 동기화 기능을 제공하는 인증 앱</p>
              </div>
            </div>
          </div>
          <div class="step-actions">
            <n-button type="primary" @click="nextStep">
              앱을 설치했습니다
            </n-button>
          </div>
        </div>

        <!-- 단계 2: QR 코드 스캔 -->
        <div v-if="currentStep === 2" class="step-content">
          <h5>2단계: QR 코드 스캔</h5>
          <p>인증 앱으로 아래 QR 코드를 스캔하세요:</p>
          
          <div class="qr-section">
            <div class="qr-code">
              <div v-if="settings?.qrCodeUrl" class="qr-image">
                <img :src="settings.qrCodeUrl" alt="2FA QR Code" />
              </div>
              <div v-else class="qr-loading">
                <n-spin size="large" />
                <p>QR 코드 생성 중...</p>
              </div>
            </div>
            
            <div class="manual-setup">
              <n-collapse>
                <n-collapse-item title="수동 설정 (QR 코드를 스캔할 수 없는 경우)">
                  <div class="manual-info">
                    <p><strong>계정 이름:</strong> {{ userEmail }}</p>
                    <p><strong>시크릿 키:</strong></p>
                    <n-input
                      :value="settings?.secret"
                      readonly
                      type="text"
                      size="small"
                    >
                      <template #suffix>
                        <n-button
                          text
                          @click="copySecret"
                        >
                          <n-icon><Copy /></n-icon>
                        </n-button>
                      </template>
                    </n-input>
                  </div>
                </n-collapse-item>
              </n-collapse>
            </div>
          </div>

          <div class="step-actions">
            <n-space>
              <n-button @click="prevStep">이전</n-button>
              <n-button type="primary" @click="nextStep">
                QR 코드를 스캔했습니다
              </n-button>
            </n-space>
          </div>
        </div>

        <!-- 단계 3: 인증 코드 확인 -->
        <div v-if="currentStep === 3" class="step-content">
          <h5>3단계: 인증 코드 확인</h5>
          <p>인증 앱에서 생성된 6자리 코드를 입력하세요:</p>
          
          <div class="verification-form">
            <n-form
              ref="verificationFormRef"
              :model="verificationForm"
              :rules="verificationRules"
            >
              <n-form-item path="token">
                <n-input
                  v-model:value="verificationForm.token"
                  placeholder="000000"
                  maxlength="6"
                  :style="{ fontSize: '18px', textAlign: 'center', letterSpacing: '4px' }"
                  :disabled="verifying"
                  @keyup.enter="verifyToken"
                />
              </n-form-item>
            </n-form>
            
            <div class="verification-help">
              <n-alert type="info" :show-icon="false">
                <p>인증 코드는 30초마다 변경됩니다. 새로운 코드를 기다려도 됩니다.</p>
              </n-alert>
            </div>
          </div>

          <div class="step-actions">
            <n-space>
              <n-button @click="prevStep" :disabled="verifying">이전</n-button>
              <n-button
                type="primary"
                :loading="verifying"
                :disabled="!verificationForm.token || verificationForm.token.length !== 6"
                @click="verifyToken"
              >
                인증 코드 확인
              </n-button>
            </n-space>
          </div>
        </div>

        <!-- 단계 4: 완료 및 백업 코드 -->
        <div v-if="currentStep === 4" class="step-content">
          <h5>4단계: 설정 완료</h5>
          <n-alert type="success" :show-icon="true">
            <template #header>2단계 인증이 활성화되었습니다!</template>
            <p>계정이 이제 2단계 인증으로 보호됩니다.</p>
          </n-alert>

          <!-- 백업 코드 -->
          <div class="backup-codes-section">
            <h6>백업 코드</h6>
            <p>인증 앱에 접근할 수 없을 때 사용할 수 있는 일회용 백업 코드입니다. 안전한 곳에 보관하세요.</p>
            
            <div class="backup-codes">
              <div class="codes-grid">
                <div
                  v-for="code in settings?.backupCodes"
                  :key="code"
                  class="backup-code"
                >
                  {{ code }}
                </div>
              </div>
              
              <div class="codes-actions">
                <n-space>
                  <n-button
                    type="primary"
                    ghost
                    @click="downloadBackupCodes"
                  >
                    <template #icon>
                      <n-icon><Download /></n-icon>
                    </template>
                    다운로드
                  </n-button>
                  <n-button
                    ghost
                    @click="copyBackupCodes"
                  >
                    <template #icon>
                      <n-icon><Copy /></n-icon>
                    </template>
                    복사
                  </n-button>
                </n-space>
              </div>
            </div>
          </div>

          <div class="step-actions">
            <n-button type="primary" @click="finishSetup">
              설정 완료
            </n-button>
          </div>
        </div>
      </div>
    </div>

    <!-- 2FA 활성화 상태 - 관리 옵션 -->
    <div v-else class="management-section">
      <div class="management-options">
        <n-space vertical size="large">
          <!-- 백업 코드 재생성 -->
          <n-card size="small">
            <template #header>
              <n-space align="center">
                <n-icon size="20">
                  <Key />
                </n-icon>
                <span>백업 코드</span>
              </n-space>
            </template>
            <p>백업 코드를 분실했거나 이미 사용한 경우 새로운 백업 코드를 생성할 수 있습니다.</p>
            <template #action>
              <n-button
                type="primary"
                ghost
                size="small"
                :loading="regeneratingCodes"
                @click="regenerateBackupCodes"
              >
                새 백업 코드 생성
              </n-button>
            </template>
          </n-card>

          <!-- 2FA 비활성화 -->
          <n-card size="small">
            <template #header>
              <n-space align="center">
                <n-icon size="20" color="#e74c3c">
                  <Warning />
                </n-icon>
                <span>2단계 인증 비활성화</span>
              </n-space>
            </template>
            <p>2단계 인증을 비활성화하면 계정 보안이 약해집니다. 신중하게 결정하세요.</p>
            <template #action>
              <n-button
                type="error"
                ghost
                size="small"
                @click="showDisableModal = true"
              >
                2FA 비활성화
              </n-button>
            </template>
          </n-card>
        </n-space>
      </div>
    </div>

    <!-- 2FA 비활성화 확인 모달 -->
    <n-modal
      v-model:show="showDisableModal"
      preset="card"
      title="2단계 인증 비활성화"
      size="medium"
      :bordered="false"
    >
      <div class="disable-form">
        <n-alert type="warning" :show-icon="true">
          <template #header>경고</template>
          <p>2단계 인증을 비활성화하면 계정 보안이 크게 약해집니다.</p>
        </n-alert>
        
        <n-form
          ref="disableFormRef"
          :model="disableForm"
          :rules="disableRules"
          style="margin-top: 16px;"
        >
          <n-form-item label="인증 코드 또는 백업 코드" path="token">
            <n-input
              v-model:value="disableForm.token"
              placeholder="6자리 인증 코드 또는 백업 코드"
              maxlength="10"
            />
          </n-form-item>
        </n-form>
      </div>

      <template #action>
        <n-space justify="end">
          <n-button @click="showDisableModal = false">취소</n-button>
          <n-button
            type="error"
            :loading="disabling"
            @click="disableTwoFactor"
          >
            비활성화
          </n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- 백업 코드 표시 모달 -->
    <n-modal
      v-model:show="showBackupCodesModal"
      preset="card"
      title="새 백업 코드"
      size="medium"
      :bordered="false"
      :closable="false"
      :mask-closable="false"
    >
      <div class="new-backup-codes">
        <n-alert type="warning" :show-icon="true">
          <template #header>중요</template>
          <p>새로운 백업 코드가 생성되었습니다. 이전 백업 코드는 더 이상 사용할 수 없습니다.</p>
        </n-alert>

        <div class="backup-codes">
          <div class="codes-grid">
            <div
              v-for="code in newBackupCodes"
              :key="code"
              class="backup-code"
            >
              {{ code }}
            </div>
          </div>
          
          <div class="codes-actions">
            <n-space>
              <n-button
                type="primary"
                ghost
                @click="downloadNewBackupCodes"
              >
                <template #icon>
                  <n-icon><Download /></n-icon>
                </template>
                다운로드
              </n-button>
              <n-button
                ghost
                @click="copyNewBackupCodes"
              >
                <template #icon>
                  <n-icon><Copy /></n-icon>
                </template>
                복사
              </n-button>
            </n-space>
          </div>
        </div>
      </div>

      <template #action>
        <n-space justify="end">
          <n-button type="primary" @click="closeBackupCodesModal">
            확인
          </n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useMessage } from 'naive-ui'
import {
  ShieldSharp as Shield,
  KeySharp as Key,
  PhonePortraitSharp as Smartphone,
  CopySharp as Copy,
  DownloadSharp as Download,
  WarningSharp as Warning
} from '@vicons/ionicons5'
import { profileApi } from '@/api/services'
import { useUserStore } from '@/stores/user'
import type { TwoFactorAuthSettings } from '@/types/api'

// Emits
const emit = defineEmits<{
  setupComplete: [settings: TwoFactorAuthSettings]
  disabled: []
}>()

// 컴포저블
const message = useMessage()
const userStore = useUserStore()

// 반응형 상태
const loading = ref(false)
const verifying = ref(false)
const disabling = ref(false)
const regeneratingCodes = ref(false)
const settings = ref<TwoFactorAuthSettings | null>(null)
const showSetup = ref(false)
const currentStep = ref(1)
const stepStatus = ref<'process' | 'finish' | 'error' | 'wait'>('process')
const showDisableModal = ref(false)
const showBackupCodesModal = ref(false)
const newBackupCodes = ref<string[]>([])

// 폼 참조
const verificationFormRef = ref()
const disableFormRef = ref()

// 폼 데이터
const verificationForm = reactive({
  token: ''
})

const disableForm = reactive({
  token: ''
})

// 폼 검증 규칙
const verificationRules = {
  token: {
    required: true,
    len: 6,
    pattern: /^\d{6}$/,
    message: '6자리 숫자를 입력해주세요',
    trigger: ['blur', 'input']
  }
}

const disableRules = {
  token: {
    required: true,
    message: '인증 코드 또는 백업 코드를 입력해주세요',
    trigger: ['blur', 'input']
  }
}

// 계산된 속성
const userEmail = computed(() => userStore.user?.email || '')

// 메서드
const loadSettings = async () => {
  try {
    settings.value = await profileApi.getTwoFactorSettings()
  } catch (error) {
    console.error('2FA 설정 로드 실패:', error)
    message.error('2FA 설정을 불러오는데 실패했습니다')
  }
}

const startSetup = async () => {
  loading.value = true
  try {
    settings.value = await profileApi.setupTwoFactor()
    showSetup.value = true
    currentStep.value = 1
  } catch (error) {
    console.error('2FA 설정 시작 실패:', error)
    message.error('2FA 설정을 시작할 수 없습니다')
  } finally {
    loading.value = false
  }
}

const nextStep = () => {
  if (currentStep.value < 4) {
    currentStep.value++
  }
}

const prevStep = () => {
  if (currentStep.value > 1) {
    currentStep.value--
  }
}

const copySecret = async () => {
  if (settings.value?.secret) {
    try {
      await navigator.clipboard.writeText(settings.value.secret)
      message.success('시크릿 키가 클립보드에 복사되었습니다')
    } catch (error) {
      message.error('클립보드 복사에 실패했습니다')
    }
  }
}

const verifyToken = async () => {
  try {
    await verificationFormRef.value?.validate()
    
    verifying.value = true
    settings.value = await profileApi.enableTwoFactor({
      token: verificationForm.token
    })
    
    currentStep.value = 4
    message.success('2단계 인증이 활성화되었습니다')
    
  } catch (error: any) {
    console.error('토큰 검증 실패:', error)
    message.error(error.message || '인증 코드가 올바르지 않습니다')
  } finally {
    verifying.value = false
  }
}

const finishSetup = () => {
  showSetup.value = false
  currentStep.value = 1
  verificationForm.token = ''
  emit('setupComplete', settings.value!)
}

const regenerateBackupCodes = async () => {
  regeneratingCodes.value = true
  try {
    newBackupCodes.value = await profileApi.regenerateBackupCodes()
    showBackupCodesModal.value = true
    message.success('새로운 백업 코드가 생성되었습니다')
  } catch (error) {
    console.error('백업 코드 재생성 실패:', error)
    message.error('백업 코드 재생성에 실패했습니다')
  } finally {
    regeneratingCodes.value = false
  }
}

const disableTwoFactor = async () => {
  try {
    await disableFormRef.value?.validate()
    
    disabling.value = true
    await profileApi.disableTwoFactor(disableForm.token)
    
    settings.value = {
      enabled: false,
      setupComplete: false
    }
    
    showDisableModal.value = false
    disableForm.token = ''
    message.success('2단계 인증이 비활성화되었습니다')
    emit('disabled')
    
  } catch (error: any) {
    console.error('2FA 비활성화 실패:', error)
    message.error(error.message || '2FA 비활성화에 실패했습니다')
  } finally {
    disabling.value = false
  }
}

const downloadBackupCodes = () => {
  const codes = settings.value?.backupCodes || []
  downloadCodes(codes, 'backup-codes.txt')
}

const downloadNewBackupCodes = () => {
  downloadCodes(newBackupCodes.value, 'new-backup-codes.txt')
}

const downloadCodes = (codes: string[], filename: string) => {
  const content = `2단계 인증 백업 코드\n생성일: ${new Date().toLocaleString()}\n\n${codes.join('\n')}\n\n주의: 이 코드들을 안전한 곳에 보관하세요.`
  const blob = new Blob([content], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
  message.success('백업 코드가 다운로드되었습니다')
}

const copyBackupCodes = async () => {
  const codes = settings.value?.backupCodes || []
  await copyCodes(codes)
}

const copyNewBackupCodes = async () => {
  await copyCodes(newBackupCodes.value)
}

const copyCodes = async (codes: string[]) => {
  try {
    await navigator.clipboard.writeText(codes.join('\n'))
    message.success('백업 코드가 클립보드에 복사되었습니다')
  } catch (error) {
    message.error('클립보드 복사에 실패했습니다')
  }
}

const closeBackupCodesModal = () => {
  showBackupCodesModal.value = false
  newBackupCodes.value = []
}

const formatDate = (dateString: string) => {
  return new Date(dateString).toLocaleString('ko-KR', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}

// 라이프사이클
onMounted(() => {
  loadSettings()
})
</script>

<style scoped lang="scss">
.two-factor-auth {
  .setup-header {
    margin-bottom: 24px;

    h4 {
      margin: 0 0 8px 0;
      font-size: 18px;
      font-weight: 500;
      color: var(--text-color-1);
    }

    .setup-description {
      margin: 0;
      font-size: 14px;
      color: var(--text-color-2);
      line-height: 1.5;
    }
  }

  .status-section {
    margin-bottom: 24px;
  }

  .setup-section {
    .setup-start {
      text-align: center;
      padding: 32px 0;
    }

    .setup-process {
      .n-steps {
        margin-bottom: 32px;
      }

      .step-content {
        max-width: 600px;
        margin: 0 auto;

        h5 {
          margin: 0 0 16px 0;
          font-size: 16px;
          font-weight: 500;
          color: var(--text-color-1);
        }

        h6 {
          margin: 0 0 8px 0;
          font-size: 14px;
          font-weight: 500;
          color: var(--text-color-1);
        }

        .app-list {
          margin: 16px 0;

          .app-item {
            display: flex;
            align-items: center;
            gap: 12px;
            padding: 12px;
            border: 1px solid var(--border-color);
            border-radius: 8px;
            margin-bottom: 8px;

            .app-info {
              strong {
                display: block;
                margin-bottom: 4px;
                color: var(--text-color-1);
              }

              p {
                margin: 0;
                font-size: 12px;
                color: var(--text-color-3);
              }
            }
          }
        }

        .qr-section {
          margin: 24px 0;

          .qr-code {
            text-align: center;
            margin-bottom: 24px;

            .qr-image {
              display: inline-block;
              padding: 16px;
              background: white;
              border-radius: 8px;
              border: 1px solid var(--border-color);

              img {
                width: 200px;
                height: 200px;
              }
            }

            .qr-loading {
              display: flex;
              flex-direction: column;
              align-items: center;
              gap: 12px;
              padding: 40px;

              p {
                margin: 0;
                color: var(--text-color-2);
              }
            }
          }

          .manual-setup {
            .manual-info {
              p {
                margin: 0 0 8px 0;
                font-size: 14px;

                strong {
                  color: var(--text-color-1);
                }
              }
            }
          }
        }

        .verification-form {
          max-width: 300px;
          margin: 24px auto;

          .verification-help {
            margin-top: 16px;
          }
        }

        .backup-codes-section {
          margin-top: 24px;

          .backup-codes {
            margin-top: 16px;

            .codes-grid {
              display: grid;
              grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
              gap: 8px;
              margin-bottom: 16px;

              .backup-code {
                padding: 8px 12px;
                background: var(--card-color);
                border: 1px solid var(--border-color);
                border-radius: 4px;
                font-family: 'Courier New', monospace;
                font-size: 14px;
                text-align: center;
                color: var(--text-color-1);
              }
            }

            .codes-actions {
              display: flex;
              justify-content: center;
            }
          }
        }

        .step-actions {
          margin-top: 32px;
          text-align: center;
        }
      }
    }
  }

  .management-section {
    .management-options {
      max-width: 600px;
    }
  }

  .disable-form,
  .new-backup-codes {
    .backup-codes {
      margin-top: 16px;

      .codes-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
        gap: 8px;
        margin-bottom: 16px;

        .backup-code {
          padding: 8px 12px;
          background: var(--card-color);
          border: 1px solid var(--border-color);
          border-radius: 4px;
          font-family: 'Courier New', monospace;
          font-size: 14px;
          text-align: center;
          color: var(--text-color-1);
        }
      }

      .codes-actions {
        display: flex;
        justify-content: center;
      }
    }
  }
}

// 반응형 디자인
@media (max-width: 768px) {
  .two-factor-auth {
    .setup-section {
      .setup-process {
        .step-content {
          .qr-section {
            .qr-code {
              .qr-image {
                img {
                  width: 160px;
                  height: 160px;
                }
              }
            }
          }

          .backup-codes-section {
            .backup-codes {
              .codes-grid {
                grid-template-columns: repeat(2, 1fr);
              }
            }
          }
        }
      }
    }
  }
}

@media (max-width: 480px) {
  .two-factor-auth {
    .setup-section {
      .setup-process {
        .step-content {
          .app-list {
            .app-item {
              flex-direction: column;
              text-align: center;
              gap: 8px;
            }
          }

          .qr-section {
            .qr-code {
              .qr-image {
                padding: 12px;

                img {
                  width: 140px;
                  height: 140px;
                }
              }
            }
          }

          .backup-codes-section {
            .backup-codes {
              .codes-grid {
                grid-template-columns: 1fr;
              }

              .codes-actions {
                .n-space {
                  width: 100%;
                  
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

          .step-actions {
            .n-space {
              width: 100%;
              
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
    }
  }
}
</style>