<template>
  <div class="security-settings">
    <div class="settings-header">
      <h3>보안 설정</h3>
      <p class="description">
        세션 타임아웃, 동시 접속 제한 등 계정 보안과 관련된 설정을 관리합니다.
      </p>
    </div>

    <n-form
      ref="formRef"
      :model="formData"
      :rules="formRules"
      label-placement="left"
      label-width="200px"
      :disabled="loading"
    >
      <!-- 세션 타임아웃 설정 -->
      <n-form-item label="세션 타임아웃" path="sessionTimeoutMinutes">
        <div class="setting-control">
          <n-input-number
            v-model:value="formData.sessionTimeoutMinutes"
            :min="15"
            :max="1440"
            :step="15"
            style="width: 200px"
          >
            <template #suffix>분</template>
          </n-input-number>
          <div class="setting-description">
            로그인 후 {{ formData.sessionTimeoutMinutes }}분간 비활성 상태가 지속되면 자동으로 로그아웃됩니다.
          </div>
        </div>
      </n-form-item>

      <!-- 최대 동시 세션 수 -->
      <n-form-item label="최대 동시 세션 수" path="maxConcurrentSessions">
        <div class="setting-control">
          <n-input-number
            v-model:value="formData.maxConcurrentSessions"
            :min="1"
            :max="10"
            style="width: 200px"
          >
            <template #suffix>개</template>
          </n-input-number>
          <div class="setting-description">
            동시에 로그인할 수 있는 세션의 최대 개수입니다. 초과 시 오래된 세션이 자동으로 종료됩니다.
          </div>
        </div>
      </n-form-item>

      <!-- 다중 디바이스 허용 -->
      <n-form-item label="다중 디바이스 허용" path="allowMultipleDevices">
        <div class="setting-control">
          <n-switch
            v-model:value="formData.allowMultipleDevices"
            :rail-style="railStyle"
          />
          <div class="setting-description">
            {{ formData.allowMultipleDevices 
              ? '여러 디바이스에서 동시에 로그인할 수 있습니다.' 
              : '한 번에 하나의 디바이스에서만 로그인할 수 있습니다.'
            }}
          </div>
        </div>
      </n-form-item>

      <!-- 비활성 세션 자동 종료 -->
      <n-form-item label="비활성 세션 자동 종료" path="autoTerminateInactiveSessions">
        <div class="setting-control">
          <n-switch
            v-model:value="formData.autoTerminateInactiveSessions"
            :rail-style="railStyle"
          />
          <div class="setting-description">
            일정 시간 동안 활동이 없는 세션을 자동으로 종료합니다.
          </div>
        </div>
      </n-form-item>

      <!-- 비활성 타임아웃 (비활성 세션 자동 종료가 활성화된 경우) -->
      <n-form-item 
        v-if="formData.autoTerminateInactiveSessions"
        label="비활성 타임아웃"
        path="inactivityTimeoutMinutes"
      >
        <div class="setting-control">
          <n-input-number
            v-model:value="formData.inactivityTimeoutMinutes"
            :min="30"
            :max="720"
            :step="30"
            style="width: 200px"
          >
            <template #suffix>분</template>
          </n-input-number>
          <div class="setting-description">
            {{ formData.inactivityTimeoutMinutes }}분간 활동이 없으면 해당 세션이 자동으로 종료됩니다.
          </div>
        </div>
      </n-form-item>

      <!-- 민감한 작업 재인증 요구 -->
      <n-form-item label="민감한 작업 재인증" path="requireReauthForSensitiveActions">
        <div class="setting-control">
          <n-switch
            v-model:value="formData.requireReauthForSensitiveActions"
            :rail-style="railStyle"
          />
          <div class="setting-description">
            비밀번호 변경, 계정 설정 변경 등 민감한 작업 시 비밀번호를 다시 입력해야 합니다.
          </div>
        </div>
      </n-form-item>

      <!-- 새 디바이스 알림 -->
      <n-form-item label="새 디바이스 알림" path="notifyOnNewDevice">
        <div class="setting-control">
          <n-switch
            v-model:value="formData.notifyOnNewDevice"
            :rail-style="railStyle"
          />
          <div class="setting-description">
            새로운 디바이스에서 로그인 시 이메일로 알림을 받습니다.
          </div>
        </div>
      </n-form-item>

      <!-- 의심스러운 활동 알림 -->
      <n-form-item label="의심스러운 활동 알림" path="notifyOnSuspiciousActivity">
        <div class="setting-control">
          <n-switch
            v-model:value="formData.notifyOnSuspiciousActivity"
            :rail-style="railStyle"
          />
          <div class="setting-description">
            의심스러운 로그인 시도나 비정상적인 활동이 감지되면 이메일로 알림을 받습니다.
          </div>
        </div>
      </n-form-item>
    </n-form>

    <!-- 저장 버튼 -->
    <div class="settings-actions">
      <n-space justify="end">
        <n-button @click="resetForm">초기화</n-button>
        <n-button
          type="primary"
          :loading="saving"
          :disabled="!hasChanges"
          @click="saveSettings"
        >
          설정 저장
        </n-button>
      </n-space>
    </div>

    <!-- 보안 권장사항 -->
    <div class="security-recommendations">
      <n-alert type="info" :show-icon="true">
        <template #header>보안 권장사항</template>
        <ul>
          <li>세션 타임아웃은 60분 이하로 설정하는 것을 권장합니다.</li>
          <li>최대 동시 세션 수는 3개 이하로 제한하는 것을 권장합니다.</li>
          <li>민감한 작업 재인증과 새 디바이스 알림을 활성화하는 것을 권장합니다.</li>
          <li>정기적으로 활성 세션을 확인하고 불필요한 세션을 종료하세요.</li>
        </ul>
      </n-alert>
    </div>

    <!-- 현재 설정 요약 -->
    <div class="settings-summary">
      <n-card title="현재 설정 요약" size="small">
        <n-descriptions :column="2" size="small">
          <n-descriptions-item label="세션 타임아웃">
            {{ formData.sessionTimeoutMinutes }}분
          </n-descriptions-item>
          <n-descriptions-item label="최대 동시 세션">
            {{ formData.maxConcurrentSessions }}개
          </n-descriptions-item>
          <n-descriptions-item label="다중 디바이스">
            {{ formData.allowMultipleDevices ? '허용' : '차단' }}
          </n-descriptions-item>
          <n-descriptions-item label="재인증 요구">
            {{ formData.requireReauthForSensitiveActions ? '활성화' : '비활성화' }}
          </n-descriptions-item>
          <n-descriptions-item label="새 디바이스 알림">
            {{ formData.notifyOnNewDevice ? '활성화' : '비활성화' }}
          </n-descriptions-item>
          <n-descriptions-item label="의심 활동 알림">
            {{ formData.notifyOnSuspiciousActivity ? '활성화' : '비활성화' }}
          </n-descriptions-item>
        </n-descriptions>
      </n-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useMessage } from 'naive-ui'
import type { SessionSecuritySettings, UpdateSessionSettingsRequest } from '@/types/api'

// Props
interface Props {
  settings: SessionSecuritySettings | null
  loading?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  loading: false
})

// Emits
const emit = defineEmits<{
  update: [settings: UpdateSessionSettingsRequest]
}>()

// 컴포저블
const message = useMessage()

// 반응형 상태
const formRef = ref()
const saving = ref(false)
const originalSettings = ref<SessionSecuritySettings | null>(null)

// 폼 데이터
const formData = ref({
  sessionTimeoutMinutes: 60,
  maxConcurrentSessions: 3,
  allowMultipleDevices: true,
  requireReauthForSensitiveActions: true,
  notifyOnNewDevice: true,
  notifyOnSuspiciousActivity: true,
  autoTerminateInactiveSessions: false,
  inactivityTimeoutMinutes: 120
})

// 폼 유효성 검사 규칙
const formRules = {
  sessionTimeoutMinutes: {
    required: true,
    type: 'number',
    min: 15,
    max: 1440,
    message: '세션 타임아웃은 15분에서 1440분(24시간) 사이여야 합니다',
    trigger: ['blur', 'change']
  },
  maxConcurrentSessions: {
    required: true,
    type: 'number',
    min: 1,
    max: 10,
    message: '최대 동시 세션 수는 1개에서 10개 사이여야 합니다',
    trigger: ['blur', 'change']
  },
  inactivityTimeoutMinutes: {
    required: true,
    type: 'number',
    min: 30,
    max: 720,
    message: '비활성 타임아웃은 30분에서 720분(12시간) 사이여야 합니다',
    trigger: ['blur', 'change']
  }
}

// 계산된 속성
const hasChanges = computed(() => {
  if (!originalSettings.value) return false
  
  return (
    formData.value.sessionTimeoutMinutes !== originalSettings.value.sessionTimeoutMinutes ||
    formData.value.maxConcurrentSessions !== originalSettings.value.maxConcurrentSessions ||
    formData.value.allowMultipleDevices !== originalSettings.value.allowMultipleDevices ||
    formData.value.requireReauthForSensitiveActions !== originalSettings.value.requireReauthForSensitiveActions ||
    formData.value.notifyOnNewDevice !== originalSettings.value.notifyOnNewDevice ||
    formData.value.notifyOnSuspiciousActivity !== originalSettings.value.notifyOnSuspiciousActivity ||
    formData.value.autoTerminateInactiveSessions !== originalSettings.value.autoTerminateInactiveSessions ||
    formData.value.inactivityTimeoutMinutes !== originalSettings.value.inactivityTimeoutMinutes
  )
})

// Switch 스타일링
const railStyle = ({ focused, checked }: { focused: boolean; checked: boolean }) => {
  const style: any = {}
  if (checked) {
    style.background = '#18a058'
    if (focused) {
      style.boxShadow = '0 0 0 2px #18a05840'
    }
  }
  return style
}

// 메서드
const loadSettings = () => {
  if (props.settings) {
    formData.value = {
      sessionTimeoutMinutes: props.settings.sessionTimeoutMinutes,
      maxConcurrentSessions: props.settings.maxConcurrentSessions,
      allowMultipleDevices: props.settings.allowMultipleDevices,
      requireReauthForSensitiveActions: props.settings.requireReauthForSensitiveActions,
      notifyOnNewDevice: props.settings.notifyOnNewDevice,
      notifyOnSuspiciousActivity: props.settings.notifyOnSuspiciousActivity,
      autoTerminateInactiveSessions: props.settings.autoTerminateInactiveSessions,
      inactivityTimeoutMinutes: props.settings.inactivityTimeoutMinutes
    }
    originalSettings.value = { ...props.settings }
  }
}

const resetForm = () => {
  if (originalSettings.value) {
    formData.value = {
      sessionTimeoutMinutes: originalSettings.value.sessionTimeoutMinutes,
      maxConcurrentSessions: originalSettings.value.maxConcurrentSessions,
      allowMultipleDevices: originalSettings.value.allowMultipleDevices,
      requireReauthForSensitiveActions: originalSettings.value.requireReauthForSensitiveActions,
      notifyOnNewDevice: originalSettings.value.notifyOnNewDevice,
      notifyOnSuspiciousActivity: originalSettings.value.notifyOnSuspiciousActivity,
      autoTerminateInactiveSessions: originalSettings.value.autoTerminateInactiveSessions,
      inactivityTimeoutMinutes: originalSettings.value.inactivityTimeoutMinutes
    }
  }
}

const saveSettings = async () => {
  try {
    await formRef.value?.validate()
    
    saving.value = true
    
    const changedSettings: UpdateSessionSettingsRequest = {}
    
    if (formData.value.sessionTimeoutMinutes !== originalSettings.value?.sessionTimeoutMinutes) {
      changedSettings.sessionTimeoutMinutes = formData.value.sessionTimeoutMinutes
    }
    if (formData.value.maxConcurrentSessions !== originalSettings.value?.maxConcurrentSessions) {
      changedSettings.maxConcurrentSessions = formData.value.maxConcurrentSessions
    }
    if (formData.value.allowMultipleDevices !== originalSettings.value?.allowMultipleDevices) {
      changedSettings.allowMultipleDevices = formData.value.allowMultipleDevices
    }
    if (formData.value.requireReauthForSensitiveActions !== originalSettings.value?.requireReauthForSensitiveActions) {
      changedSettings.requireReauthForSensitiveActions = formData.value.requireReauthForSensitiveActions
    }
    if (formData.value.notifyOnNewDevice !== originalSettings.value?.notifyOnNewDevice) {
      changedSettings.notifyOnNewDevice = formData.value.notifyOnNewDevice
    }
    if (formData.value.notifyOnSuspiciousActivity !== originalSettings.value?.notifyOnSuspiciousActivity) {
      changedSettings.notifyOnSuspiciousActivity = formData.value.notifyOnSuspiciousActivity
    }
    if (formData.value.autoTerminateInactiveSessions !== originalSettings.value?.autoTerminateInactiveSessions) {
      changedSettings.autoTerminateInactiveSessions = formData.value.autoTerminateInactiveSessions
    }
    if (formData.value.inactivityTimeoutMinutes !== originalSettings.value?.inactivityTimeoutMinutes) {
      changedSettings.inactivityTimeoutMinutes = formData.value.inactivityTimeoutMinutes
    }
    
    emit('update', changedSettings)
  } catch (error) {
    console.error('설정 유효성 검사 실패:', error)
  } finally {
    saving.value = false
  }
}

// 와처
watch(() => props.settings, loadSettings, { immediate: true })

// 라이프사이클
onMounted(() => {
  loadSettings()
})
</script>

<style scoped lang="scss">
.security-settings {
  max-width: 800px;

  .settings-header {
    margin-bottom: 32px;
    padding-bottom: 16px;
    border-bottom: 1px solid var(--border-color);

    h3 {
      margin: 0 0 8px 0;
      font-size: 24px;
      font-weight: 600;
      color: var(--text-color-1);
    }

    .description {
      margin: 0;
      color: var(--text-color-2);
      font-size: 16px;
      line-height: 1.5;
    }
  }

  .setting-control {
    display: flex;
    flex-direction: column;
    gap: 8px;
    width: 100%;

    .setting-description {
      font-size: 14px;
      color: var(--text-color-3);
      line-height: 1.4;
    }
  }

  .settings-actions {
    margin: 32px 0;
    padding-top: 24px;
    border-top: 1px solid var(--border-color);
  }

  .security-recommendations {
    margin: 32px 0;

    ul {
      margin: 8px 0 0 0;
      padding-left: 20px;
      
      li {
        margin-bottom: 4px;
        font-size: 14px;
        line-height: 1.5;
      }
    }
  }

  .settings-summary {
    margin-top: 24px;
  }
}

// 반응형 디자인
@media (max-width: 768px) {
  .security-settings {
    .n-form {
      :deep(.n-form-item) {
        .n-form-item-label {
          width: 100% !important;
          text-align: left !important;
          padding-bottom: 8px;
        }

        .n-form-item-blank {
          margin-left: 0 !important;
        }
      }
    }

    .settings-summary {
      .n-descriptions {
        :deep(.n-descriptions-table) {
          .n-descriptions-table-wrapper {
            .n-descriptions-table-content {
              .n-descriptions-table-row {
                flex-direction: column;
                
                .n-descriptions-table-content-cell {
                  width: 100% !important;
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