<template>
  <div class="notification-settings">
    <div class="settings-header">
      <h3>알림 설정</h3>
      <p class="settings-description">
        받고 싶은 알림 유형을 선택하고 알림 방법을 설정할 수 있습니다.
      </p>
    </div>

    <div class="settings-content">
      <n-space vertical size="large">
        <!-- 이메일 알림 섹션 -->
        <n-card size="small">
          <template #header>
            <n-space align="center">
              <n-icon size="20">
                <Mail />
              </n-icon>
              <span>이메일 알림</span>
            </n-space>
          </template>
          
          <div class="notification-section">
            <n-space vertical size="medium">
              <div class="notification-item">
                <div class="item-info">
                  <h5>로그인 알림</h5>
                  <p>새로운 기기에서 로그인하거나 의심스러운 로그인 활동이 감지될 때 알림을 받습니다.</p>
                </div>
                <div class="item-action">
                  <n-switch
                    v-model:value="localSettings.emailNotifications.loginNotifications"
                    @update:value="handleSettingChange"
                  />
                </div>
              </div>

              <div class="notification-item">
                <div class="item-info">
                  <h5>보안 경고</h5>
                  <p>계정 보안과 관련된 중요한 활동에 대한 알림을 받습니다.</p>
                </div>
                <div class="item-action">
                  <n-switch
                    v-model:value="localSettings.emailNotifications.securityAlerts"
                    @update:value="handleSettingChange"
                  />
                </div>
              </div>

              <div class="notification-item">
                <div class="item-info">
                  <h5>워크스페이스 업데이트</h5>
                  <p>워크스페이스 생성, 수정, 삭제 등 워크스페이스 관련 활동 알림을 받습니다.</p>
                </div>
                <div class="item-action">
                  <n-switch
                    v-model:value="localSettings.emailNotifications.workspaceUpdates"
                    @update:value="handleSettingChange"
                  />
                </div>
              </div>

              <div class="notification-item">
                <div class="item-info">
                  <h5>시스템 업데이트</h5>
                  <p>시스템 점검, 업데이트, 중요한 공지사항에 대한 알림을 받습니다.</p>
                </div>
                <div class="item-action">
                  <n-switch
                    v-model:value="localSettings.emailNotifications.systemUpdates"
                    @update:value="handleSettingChange"
                  />
                </div>
              </div>

              <div class="notification-item">
                <div class="item-info">
                  <h5>프로모션 이메일</h5>
                  <p>새로운 기능 소개, 이벤트, 프로모션 정보에 대한 이메일을 받습니다.</p>
                </div>
                <div class="item-action">
                  <n-switch
                    v-model:value="localSettings.emailNotifications.promotionalEmails"
                    @update:value="handleSettingChange"
                  />
                </div>
              </div>
            </n-space>
          </div>
        </n-card>

        <!-- 브라우저 푸시 알림 섹션 -->
        <n-card size="small">
          <template #header>
            <n-space align="center">
              <n-icon size="20">
                <Notifications />
              </n-icon>
              <span>브라우저 푸시 알림</span>
              <n-tag v-if="!pushNotificationSupported" type="warning" size="small">
                지원되지 않음
              </n-tag>
            </n-space>
          </template>
          
          <div class="notification-section">
            <n-space vertical size="medium">
              <!-- 푸시 알림 활성화 -->
              <div class="notification-item">
                <div class="item-info">
                  <h5>푸시 알림 활성화</h5>
                  <p>브라우저에서 실시간 알림을 받습니다. 브라우저 권한 허용이 필요합니다.</p>
                </div>
                <div class="item-action">
                  <n-switch
                    v-model:value="localSettings.pushNotifications.enabled"
                    :disabled="!pushNotificationSupported || requesting"
                    @update:value="handlePushNotificationToggle"
                  />
                </div>
              </div>

              <!-- 푸시 알림이 활성화된 경우에만 세부 설정 표시 -->
              <template v-if="localSettings.pushNotifications.enabled">
                <div class="notification-item">
                  <div class="item-info">
                    <h5>보안 경고</h5>
                    <p>긴급한 보안 관련 알림을 즉시 받습니다.</p>
                  </div>
                  <div class="item-action">
                    <n-switch
                      v-model:value="localSettings.pushNotifications.securityAlerts"
                      @update:value="handleSettingChange"
                    />
                  </div>
                </div>

                <div class="notification-item">
                  <div class="item-info">
                    <h5>워크스페이스 업데이트</h5>
                    <p>워크스페이스 상태 변경 시 실시간 알림을 받습니다.</p>
                  </div>
                  <div class="item-action">
                    <n-switch
                      v-model:value="localSettings.pushNotifications.workspaceUpdates"
                      @update:value="handleSettingChange"
                    />
                  </div>
                </div>

                <div class="notification-item">
                  <div class="item-info">
                    <h5>다이렉트 메시지</h5>
                    <p>다른 사용자로부터의 메시지나 멘션 알림을 받습니다.</p>
                  </div>
                  <div class="item-action">
                    <n-switch
                      v-model:value="localSettings.pushNotifications.directMessages"
                      @update:value="handleSettingChange"
                    />
                  </div>
                </div>
              </template>

              <!-- 푸시 알림 권한 상태 표시 -->
              <div v-if="pushNotificationSupported" class="permission-status">
                <n-alert 
                  :type="permissionAlertType" 
                  :show-icon="true"
                  size="small"
                >
                  <template #header>
                    푸시 알림 권한 상태
                  </template>
                  {{ permissionStatusText }}
                  <div v-if="notificationPermission === 'denied'" style="margin-top: 8px;">
                    <n-text depth="3" style="font-size: 12px;">
                      브라우저 설정에서 알림 권한을 허용해주세요.
                    </n-text>
                  </div>
                </n-alert>
              </div>
            </n-space>
          </div>
        </n-card>

        <!-- SMS 알림 섹션 -->
        <n-card size="small">
          <template #header>
            <n-space align="center">
              <n-icon size="20">
                <ChatBubble />
              </n-icon>
              <span>SMS 알림</span>
            </n-space>
          </template>
          
          <div class="notification-section">
            <n-space vertical size="medium">
              <!-- SMS 알림 활성화 -->
              <div class="notification-item">
                <div class="item-info">
                  <h5>SMS 알림 활성화</h5>
                  <p>중요한 알림을 문자 메시지로 받습니다. 전화번호 인증이 필요합니다.</p>
                </div>
                <div class="item-action">
                  <n-switch
                    v-model:value="localSettings.smsNotifications.enabled"
                    :disabled="!phoneVerified"
                    @update:value="handleSettingChange"
                  />
                </div>
              </div>

              <!-- 전화번호 미인증 안내 -->
              <div v-if="!phoneVerified" class="phone-verification-notice">
                <n-alert type="warning" :show-icon="true" size="small">
                  <template #header>전화번호 인증 필요</template>
                  <p>SMS 알림을 받으려면 먼저 전화번호를 인증해야 합니다.</p>
                  <template #action>
                    <n-button
                      size="small"
                      type="primary"
                      ghost
                      @click="goToProfileSettings"
                    >
                      인증하기
                    </n-button>
                  </template>
                </n-alert>
              </div>

              <!-- SMS 알림이 활성화된 경우에만 세부 설정 표시 -->
              <template v-if="localSettings.smsNotifications.enabled && phoneVerified">
                <div class="notification-item">
                  <div class="item-info">
                    <h5>보안 경고</h5>
                    <p>의심스러운 로그인이나 중요한 보안 이벤트 발생 시 SMS로 알림을 받습니다.</p>
                  </div>
                  <div class="item-action">
                    <n-switch
                      v-model:value="localSettings.smsNotifications.securityAlerts"
                      @update:value="handleSettingChange"
                    />
                  </div>
                </div>

                <div class="notification-item">
                  <div class="item-info">
                    <h5>중요 업데이트</h5>
                    <p>시스템 장애, 긴급 점검 등 중요한 업데이트를 SMS로 받습니다.</p>
                  </div>
                  <div class="item-action">
                    <n-switch
                      v-model:value="localSettings.smsNotifications.criticalUpdates"
                      @update:value="handleSettingChange"
                    />
                  </div>
                </div>
              </template>
            </n-space>
          </div>
        </n-card>

        <!-- 알림 시간 설정 섹션 -->
        <n-card size="small">
          <template #header>
            <n-space align="center">
              <n-icon size="20">
                <Time />
              </n-icon>
              <span>알림 시간 설정</span>
            </n-space>
          </template>
          
          <div class="notification-section">
            <n-space vertical size="medium">
              <div class="notification-item">
                <div class="item-info">
                  <h5>방해 금지 시간</h5>
                  <p>지정된 시간 동안은 긴급 알림을 제외하고 알림을 받지 않습니다.</p>
                </div>
                <div class="item-action">
                  <n-switch
                    v-model:value="doNotDisturbEnabled"
                    @update:value="handleDoNotDisturbToggle"
                  />
                </div>
              </div>

              <div v-if="doNotDisturbEnabled" class="time-range-setting">
                <div class="time-range">
                  <n-space align="center">
                    <span class="time-label">시작 시간:</span>
                    <n-time-picker
                      v-model:value="doNotDisturbStart"
                      format="HH:mm"
                      placeholder="시작 시간"
                      @update:value="handleTimeChange"
                    />
                    <span class="time-label">종료 시간:</span>
                    <n-time-picker
                      v-model:value="doNotDisturbEnd"
                      format="HH:mm"
                      placeholder="종료 시간"
                      @update:value="handleTimeChange"
                    />
                  </n-space>
                </div>
              </div>
            </n-space>
          </div>
        </n-card>
      </n-space>
    </div>

    <!-- 테스트 알림 전송 -->
    <div class="test-notification">
      <n-card size="small">
        <template #header>
          <n-space align="center">
            <n-icon size="20">
              <Flask />
            </n-icon>
            <span>알림 테스트</span>
          </n-space>
        </template>
        
        <div class="test-section">
          <p>설정한 알림이 정상적으로 작동하는지 테스트해볼 수 있습니다.</p>
          <n-space>
            <n-button
              type="primary"
              ghost
              :loading="sendingTestEmail"
              @click="sendTestEmail"
            >
              <template #icon>
                <n-icon><Mail /></n-icon>
              </template>
              테스트 이메일 발송
            </n-button>
            
            <n-button
              v-if="localSettings.pushNotifications.enabled"
              type="primary"
              ghost
              :loading="sendingTestPush"
              @click="sendTestPushNotification"
            >
              <template #icon>
                <n-icon><Notifications /></n-icon>
              </template>
              테스트 푸시 알림
            </n-button>
            
            <n-button
              v-if="localSettings.smsNotifications.enabled && phoneVerified"
              type="primary"
              ghost
              :loading="sendingTestSMS"
              @click="sendTestSMS"
            >
              <template #icon>
                <n-icon><ChatBubble /></n-icon>
              </template>
              테스트 SMS
            </n-button>
          </n-space>
        </div>
      </n-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch, onMounted } from 'vue'
import { useMessage } from 'naive-ui'
import { useRouter } from 'vue-router'
import {
  MailSharp as Mail,
  NotificationsSharp as Notifications,
  ChatbubbleSharp as ChatBubble,
  TimeSharp as Time,
  FlaskSharp as Flask
} from '@vicons/ionicons5'
import { profileApi } from '@/api/services'
import { useUserStore } from '@/stores/user'
import type { NotificationSettings, UpdateNotificationSettingsRequest } from '@/types/api'

// Props
interface Props {
  settings?: NotificationSettings | null
  loading?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  loading: false
})

// Emits
const emit = defineEmits<{
  update: [settings: UpdateNotificationSettingsRequest]
}>()

// 컴포저블
const message = useMessage()
const router = useRouter()
const userStore = useUserStore()

// 반응형 상태
const requesting = ref(false)
const sendingTestEmail = ref(false)
const sendingTestPush = ref(false)
const sendingTestSMS = ref(false)
const notificationPermission = ref<NotificationPermission>('default')
const doNotDisturbEnabled = ref(false)
const doNotDisturbStart = ref<number | null>(null)
const doNotDisturbEnd = ref<number | null>(null)

// 로컬 설정 (실시간 변경을 위해)
const localSettings = reactive<NotificationSettings>({
  userId: '',
  emailNotifications: {
    loginNotifications: true,
    securityAlerts: true,
    workspaceUpdates: true,
    systemUpdates: true,
    promotionalEmails: false
  },
  pushNotifications: {
    enabled: false,
    securityAlerts: true,
    workspaceUpdates: false,
    directMessages: true
  },
  smsNotifications: {
    enabled: false,
    securityAlerts: true,
    criticalUpdates: true
  },
  updatedAt: ''
})

// 계산된 속성
const pushNotificationSupported = computed(() => {
  return 'Notification' in window && 'serviceWorker' in navigator
})

const phoneVerified = computed(() => {
  return userStore.user?.isPhoneVerified || false
})

const permissionAlertType = computed(() => {
  switch (notificationPermission.value) {
    case 'granted': return 'success'
    case 'denied': return 'error'
    default: return 'warning'
  }
})

const permissionStatusText = computed(() => {
  switch (notificationPermission.value) {
    case 'granted': return '푸시 알림 권한이 허용되었습니다.'
    case 'denied': return '푸시 알림 권한이 거부되었습니다.'
    default: return '푸시 알림 권한을 요청하지 않았습니다.'
  }
})

// 메서드
const checkNotificationPermission = () => {
  if (pushNotificationSupported.value) {
    notificationPermission.value = Notification.permission
  }
}

const requestNotificationPermission = async () => {
  if (!pushNotificationSupported.value) return false
  
  requesting.value = true
  try {
    const permission = await Notification.requestPermission()
    notificationPermission.value = permission
    return permission === 'granted'
  } catch (error) {
    console.error('푸시 알림 권한 요청 실패:', error)
    return false
  } finally {
    requesting.value = false
  }
}

const handlePushNotificationToggle = async (enabled: boolean) => {
  if (enabled && notificationPermission.value !== 'granted') {
    const granted = await requestNotificationPermission()
    if (!granted) {
      localSettings.pushNotifications.enabled = false
      message.error('푸시 알림 권한이 필요합니다')
      return
    }
  }
  
  handleSettingChange()
}

const handleSettingChange = () => {
  // 디바운스를 위한 지연
  clearTimeout(window.settingsUpdateTimeout)
  window.settingsUpdateTimeout = setTimeout(() => {
    updateSettings()
  }, 500)
}

const updateSettings = () => {
  const updateData: UpdateNotificationSettingsRequest = {
    emailNotifications: { ...localSettings.emailNotifications },
    pushNotifications: { ...localSettings.pushNotifications },
    smsNotifications: { ...localSettings.smsNotifications }
  }
  
  emit('update', updateData)
}

const handleDoNotDisturbToggle = (enabled: boolean) => {
  doNotDisturbEnabled.value = enabled
  if (enabled && !doNotDisturbStart.value) {
    // 기본값 설정 (22:00 - 08:00)
    doNotDisturbStart.value = 22 * 60 * 60 * 1000 // 22:00
    doNotDisturbEnd.value = 8 * 60 * 60 * 1000    // 08:00
  }
  // 실제 구현에서는 이 설정도 서버에 저장해야 함
}

const handleTimeChange = () => {
  // 실제 구현에서는 시간 변경을 서버에 저장해야 함
  console.log('방해 금지 시간 변경:', {
    start: doNotDisturbStart.value,
    end: doNotDisturbEnd.value
  })
}

const goToProfileSettings = () => {
  router.push('/profile?tab=basic')
}

const sendTestEmail = async () => {
  sendingTestEmail.value = true
  try {
    // 실제 구현에서는 테스트 이메일 발송 API 호출
    await new Promise(resolve => setTimeout(resolve, 1000))
    message.success('테스트 이메일이 발송되었습니다')
  } catch (error) {
    console.error('테스트 이메일 발송 실패:', error)
    message.error('테스트 이메일 발송에 실패했습니다')
  } finally {
    sendingTestEmail.value = false
  }
}

const sendTestPushNotification = async () => {
  if (!pushNotificationSupported.value || notificationPermission.value !== 'granted') {
    message.error('푸시 알림 권한이 필요합니다')
    return
  }
  
  sendingTestPush.value = true
  try {
    // 브라우저 푸시 알림 테스트
    new Notification('AICLI 테스트 알림', {
      body: '푸시 알림 설정이 정상적으로 작동하고 있습니다.',
      icon: '/favicon.ico',
      badge: '/favicon.ico'
    })
    message.success('테스트 푸시 알림이 발송되었습니다')
  } catch (error) {
    console.error('테스트 푸시 알림 발송 실패:', error)
    message.error('테스트 푸시 알림 발송에 실패했습니다')
  } finally {
    sendingTestPush.value = false
  }
}

const sendTestSMS = async () => {
  sendingTestSMS.value = true
  try {
    // 실제 구현에서는 테스트 SMS 발송 API 호출
    await new Promise(resolve => setTimeout(resolve, 1000))
    message.success('테스트 SMS가 발송되었습니다')
  } catch (error) {
    console.error('테스트 SMS 발송 실패:', error)
    message.error('테스트 SMS 발송에 실패했습니다')
  } finally {
    sendingTestSMS.value = false
  }
}

// 설정 동기화
watch(() => props.settings, (newSettings) => {
  if (newSettings) {
    Object.assign(localSettings, newSettings)
  }
}, { immediate: true, deep: true })

// 라이프사이클
onMounted(() => {
  checkNotificationPermission()
})

// 전역 타입 확장 (TypeScript)
declare global {
  interface Window {
    settingsUpdateTimeout: number
  }
}
</script>

<style scoped lang="scss">
.notification-settings {
  .settings-header {
    margin-bottom: 24px;

    h3 {
      margin: 0 0 8px 0;
      font-size: 20px;
      font-weight: 500;
      color: var(--text-color-1);
    }

    .settings-description {
      margin: 0;
      color: var(--text-color-2);
      font-size: 14px;
      line-height: 1.4;
    }
  }

  .settings-content {
    .notification-section {
      .notification-item {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        gap: 16px;
        padding: 16px 0;
        border-bottom: 1px solid var(--border-color);

        &:last-child {
          border-bottom: none;
        }

        .item-info {
          flex: 1;

          h5 {
            margin: 0 0 4px 0;
            font-size: 16px;
            font-weight: 500;
            color: var(--text-color-1);
          }

          p {
            margin: 0;
            font-size: 14px;
            color: var(--text-color-2);
            line-height: 1.4;
          }
        }

        .item-action {
          flex-shrink: 0;
          display: flex;
          align-items: center;
        }
      }

      .permission-status {
        margin-top: 8px;
      }

      .phone-verification-notice {
        margin-top: 8px;
      }

      .time-range-setting {
        margin-top: 16px;
        padding-top: 16px;
        border-top: 1px solid var(--border-color);

        .time-range {
          .time-label {
            font-size: 14px;
            color: var(--text-color-2);
            white-space: nowrap;
          }
        }
      }
    }
  }

  .test-notification {
    margin-top: 24px;

    .test-section {
      p {
        margin: 0 0 16px 0;
        color: var(--text-color-2);
        font-size: 14px;
        line-height: 1.4;
      }
    }
  }
}

// 반응형 디자인
@media (max-width: 768px) {
  .notification-settings {
    .settings-content {
      .notification-section {
        .notification-item {
          flex-direction: column;
          align-items: flex-start;
          gap: 12px;

          .item-action {
            width: 100%;
            justify-content: flex-end;
          }
        }

        .time-range-setting {
          .time-range {
            .n-space {
              flex-direction: column;
              align-items: stretch;
              gap: 8px;
            }
          }
        }
      }
    }

    .test-notification {
      .test-section {
        .n-space {
          flex-direction: column;
          
          :deep(.n-space-item) {
            .n-button {
              width: 100%;
            }
          }
        }
      }
    }
  }
}

@media (max-width: 480px) {
  .notification-settings {
    .settings-content {
      .notification-section {
        .notification-item {
          .item-action {
            justify-content: flex-start;
          }
        }
      }
    }
  }
}
</style>