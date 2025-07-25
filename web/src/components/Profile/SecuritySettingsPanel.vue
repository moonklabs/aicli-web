<template>
  <div class="security-settings">
    <div class="settings-header">
      <h3>보안 설정</h3>
      <p class="settings-description">
        계정 보안을 강화하고 민감한 작업에 대한 추가 보호를 설정할 수 있습니다.
      </p>
    </div>

    <div class="settings-content">
      <n-space vertical size="large">
        <!-- 비밀번호 변경 섹션 -->
        <n-card size="small">
          <PasswordChangeForm
            :last-password-change="profile?.lastPasswordChange"
            @success="handlePasswordChangeSuccess"
            @error="handlePasswordChangeError"
          />
        </n-card>

        <!-- 2단계 인증 섹션 -->
        <n-card size="small">
          <TwoFactorAuthSetup
            @setup-complete="handleTwoFactorSetupComplete"
            @disabled="handleTwoFactorDisabled"
          />
        </n-card>

        <!-- 세션 및 로그인 보안 섹션 -->
        <n-card size="small">
          <template #header>
            <n-space align="center">
              <NIcon size="20">
                <Shield />
              </NIcon>
              <span>세션 및 로그인 보안</span>
            </n-space>
          </template>

          <div class="session-security">
            <n-space vertical size="medium">
              <!-- 활성 세션 관리 -->
              <div class="security-item">
                <div class="item-info">
                  <h5>활성 세션 관리</h5>
                  <p>현재 로그인된 모든 기기와 세션을 확인하고 관리할 수 있습니다.</p>
                </div>
                <div class="item-action">
                  <n-button
                    type="primary"
                    ghost
                    @click="goToSessionManagement"
                  >
                    세션 관리
                  </n-button>
                </div>
              </div>

              <!-- 보안 알림 설정 -->
              <div class="security-item">
                <div class="item-info">
                  <h5>보안 알림</h5>
                  <p>의심스러운 로그인 활동이나 새로운 기기에서 로그인 시 알림을 받습니다.</p>
                </div>
                <div class="item-action">
                  <n-switch
                    v-model:value="securitySettings.notifyOnSuspiciousActivity"
                    @update:value="updateSecuritySetting('notifyOnSuspiciousActivity', $event)"
                  />
                </div>
              </div>

              <!-- 새 기기 로그인 알림 -->
              <div class="security-item">
                <div class="item-info">
                  <h5>새 기기 로그인 알림</h5>
                  <p>새로운 기기에서 로그인할 때 이메일 알림을 받습니다.</p>
                </div>
                <div class="item-action">
                  <n-switch
                    v-model:value="securitySettings.notifyOnNewDevice"
                    @update:value="updateSecuritySetting('notifyOnNewDevice', $event)"
                  />
                </div>
              </div>

              <!-- 민감한 작업 재인증 -->
              <div class="security-item">
                <div class="item-info">
                  <h5>민감한 작업 재인증</h5>
                  <p>민감한 작업(비밀번호 변경, 계정 삭제 등) 시 비밀번호를 다시 입력해야 합니다.</p>
                </div>
                <div class="item-action">
                  <n-switch
                    v-model:value="securitySettings.requireReauthForSensitiveActions"
                    @update:value="updateSecuritySetting('requireReauthForSensitiveActions', $event)"
                  />
                </div>
              </div>
            </n-space>
          </div>
        </n-card>

        <!-- 계정 접근 제한 섹션 -->
        <n-card size="small">
          <template #header>
            <n-space align="center">
              <NIcon size="20">
                <Lock />
              </NIcon>
              <span>계정 접근 제한</span>
            </n-space>
          </template>

          <div class="access-restrictions">
            <n-space vertical size="medium">
              <!-- 동시 세션 제한 -->
              <div class="security-item">
                <div class="item-info">
                  <h5>최대 동시 세션</h5>
                  <p>한 번에 로그인할 수 있는 최대 세션 수를 제한합니다.</p>
                </div>
                <div class="item-action">
                  <n-input-number
                    v-model:value="securitySettings.maxConcurrentSessions"
                    :min="1"
                    :max="10"
                    style="width: 120px"
                    @update:value="updateSecuritySetting('maxConcurrentSessions', $event)"
                  />
                </div>
              </div>

              <!-- 다중 기기 로그인 허용 -->
              <div class="security-item">
                <div class="item-info">
                  <h5>다중 기기 로그인</h5>
                  <p>여러 기기에서 동시에 로그인하는 것을 허용합니다.</p>
                </div>
                <div class="item-action">
                  <n-switch
                    v-model:value="securitySettings.allowMultipleDevices"
                    @update:value="updateSecuritySetting('allowMultipleDevices', $event)"
                  />
                </div>
              </div>

              <!-- 비활성 세션 자동 종료 -->
              <div class="security-item">
                <div class="item-info">
                  <h5>비활성 세션 자동 종료</h5>
                  <p>일정 시간 동안 활동이 없는 세션을 자동으로 종료합니다.</p>
                </div>
                <div class="item-action">
                  <n-switch
                    v-model:value="securitySettings.autoTerminateInactiveSessions"
                    @update:value="updateSecuritySetting('autoTerminateInactiveSessions', $event)"
                  />
                </div>
              </div>

              <!-- 비활성 시간 설정 -->
              <div v-if="securitySettings.autoTerminateInactiveSessions" class="security-item">
                <div class="item-info">
                  <h5>비활성 시간 제한</h5>
                  <p>세션이 자동으로 종료되기까지의 비활성 시간(분)입니다.</p>
                </div>
                <div class="item-action">
                  <n-select
                    v-model:value="securitySettings.inactivityTimeoutMinutes"
                    :options="inactivityTimeoutOptions"
                    style="width: 150px"
                    @update:value="updateSecuritySetting('inactivityTimeoutMinutes', $event)"
                  />
                </div>
              </div>

              <!-- 세션 타임아웃 설정 -->
              <div class="security-item">
                <div class="item-info">
                  <h5>세션 타임아웃</h5>
                  <p>로그인 세션이 유지되는 시간(분)입니다.</p>
                </div>
                <div class="item-action">
                  <n-select
                    v-model:value="securitySettings.sessionTimeoutMinutes"
                    :options="sessionTimeoutOptions"
                    style="width: 150px"
                    @update:value="updateSecuritySetting('sessionTimeoutMinutes', $event)"
                  />
                </div>
              </div>
            </n-space>
          </div>
        </n-card>

        <!-- 보안 활동 로그 섹션 -->
        <n-card size="small">
          <template #header>
            <n-space align="center">
              <NIcon size="20">
                <Document />
              </NIcon>
              <span>보안 활동 로그</span>
            </n-space>
          </template>

          <div class="activity-log">
            <n-space vertical size="medium">
              <!-- 로그인 기록 -->
              <div class="security-item">
                <div class="item-info">
                  <h5>로그인 기록</h5>
                  <p>최근 로그인 활동과 접속 기록을 확인할 수 있습니다.</p>
                </div>
                <div class="item-action">
                  <n-button
                    type="primary"
                    ghost
                    @click="viewLoginHistory"
                  >
                    기록 보기
                  </n-button>
                </div>
              </div>

              <!-- 계정 활동 로그 -->
              <div class="security-item">
                <div class="item-info">
                  <h5>계정 활동 로그</h5>
                  <p>프로파일 변경, 보안 설정 변경 등 계정 관련 활동 기록을 확인할 수 있습니다.</p>
                </div>
                <div class="item-action">
                  <n-button
                    type="primary"
                    ghost
                    @click="viewActivityLog"
                  >
                    기록 보기
                  </n-button>
                </div>
              </div>
            </n-space>
          </div>
        </n-card>
      </n-space>
    </div>

    <!-- 로그인 기록 모달 -->
    <n-modal
      v-model:show="showLoginHistoryModal"
      preset="card"
      title="로그인 기록"
      size="large"
      :bordered="false"
    >
      <div class="login-history">
        <n-data-table
          :columns="loginHistoryColumns"
          :data="loginHistory"
          :loading="loadingLoginHistory"
          :pagination="loginHistoryPagination"
          size="small"
        />
      </div>
    </n-modal>

    <!-- 활동 로그 모달 -->
    <n-modal
      v-model:show="showActivityLogModal"
      preset="card"
      title="계정 활동 로그"
      size="large"
      :bordered="false"
    >
      <div class="activity-log-content">
        <n-data-table
          :columns="activityLogColumns"
          :data="activityLog"
          :loading="loadingActivityLog"
          :pagination="activityLogPagination"
          size="small"
        />
      </div>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { h, onMounted, reactive, ref } from 'vue'
import { useMessage } from 'naive-ui'
import { useRouter } from 'vue-router'
import { NIcon, NTag } from 'naive-ui'
import {
  TimeSharp as Clock,
  PhonePortraitSharp as Device,
  DocumentTextSharp as Document,
  LocationSharp as Location,
  LockClosedSharp as Lock,
  ShieldSharp as Shield,
} from '@vicons/ionicons5'
import { profileApi, sessionApi } from '@/api/services'
import { useUserStore } from '@/stores/user'
import type {
  SessionSecuritySettings,
  TwoFactorAuthSettings,
  UserProfile,
} from '@/types/api'

// 컴포넌트 import
import PasswordChangeForm from './PasswordChangeForm.vue'
import TwoFactorAuthSetup from './TwoFactorAuthSetup.vue'

// Props
interface Props {
  loading?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
})

// Emits
const emit = defineEmits<{
  passwordChange: [success: boolean]
  twoFactorSetup: [settings: TwoFactorAuthSettings]
}>()

// 컴포저블
const message = useMessage()
const router = useRouter()
const userStore = useUserStore()

// 반응형 상태
const loading = ref(false)
const loadingLoginHistory = ref(false)
const loadingActivityLog = ref(false)
const profile = ref<UserProfile | null>(null)
const showLoginHistoryModal = ref(false)
const showActivityLogModal = ref(false)
const loginHistory = ref([])
const activityLog = ref([])

// 보안 설정
const securitySettings = reactive<SessionSecuritySettings>({
  userId: '',
  sessionTimeoutMinutes: 1440, // 24시간
  maxConcurrentSessions: 5,
  allowMultipleDevices: true,
  requireReauthForSensitiveActions: true,
  notifyOnNewDevice: true,
  notifyOnSuspiciousActivity: true,
  autoTerminateInactiveSessions: false,
  inactivityTimeoutMinutes: 60,
  updatedAt: '',
})

// 옵션 데이터
const sessionTimeoutOptions = [
  { label: '30분', value: 30 },
  { label: '1시간', value: 60 },
  { label: '2시간', value: 120 },
  { label: '6시간', value: 360 },
  { label: '12시간', value: 720 },
  { label: '24시간', value: 1440 },
  { label: '1주일', value: 10080 },
  { label: '1개월', value: 43200 },
]

const inactivityTimeoutOptions = [
  { label: '15분', value: 15 },
  { label: '30분', value: 30 },
  { label: '1시간', value: 60 },
  { label: '2시간', value: 120 },
  { label: '4시간', value: 240 },
  { label: '8시간', value: 480 },
]

// 테이블 컬럼 정의
const loginHistoryColumns = [
  {
    title: '시간',
    key: 'timestamp',
    width: 160,
    render: (row: any) => {
      return new Date(row.timestamp).toLocaleString('ko-KR')
    },
  },
  {
    title: 'IP 주소',
    key: 'ipAddress',
    width: 140,
  },
  {
    title: '위치',
    key: 'location',
    width: 160,
    render: (row: any) => {
      return row.location || '알 수 없음'
    },
  },
  {
    title: '기기',
    key: 'device',
    width: 200,
    render: (row: any) => {
      return h(NTag, { size: 'small' }, { default: () => row.device })
    },
  },
  {
    title: '상태',
    key: 'status',
    width: 100,
    render: (row: any) => {
      const type = row.status === 'success' ? 'success' : 'error'
      return h(NTag, { type, size: 'small' }, { default: () => row.status })
    },
  },
]

const activityLogColumns = [
  {
    title: '시간',
    key: 'timestamp',
    width: 160,
    render: (row: any) => {
      return new Date(row.timestamp).toLocaleString('ko-KR')
    },
  },
  {
    title: '활동',
    key: 'activity',
    width: 200,
  },
  {
    title: '설명',
    key: 'description',
    minWidth: 250,
  },
  {
    title: 'IP',
    key: 'ipAddress',
    width: 140,
  },
]

// 페이지네이션
const loginHistoryPagination = reactive({
  page: 1,
  pageSize: 10,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50],
  onUpdatePage: (page: number) => {
    loginHistoryPagination.page = page
    loadLoginHistory()
  },
})

const activityLogPagination = reactive({
  page: 1,
  pageSize: 10,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50],
  onUpdatePage: (page: number) => {
    activityLogPagination.page = page
    loadActivityLog()
  },
})

// 메서드
const loadProfile = async () => {
  try {
    profile.value = await profileApi.getProfile()
  } catch (error) {
    console.error('프로파일 로드 실패:', error)
  }
}

const loadSecuritySettings = async () => {
  try {
    const settings = await sessionApi.getSecuritySettings()
    Object.assign(securitySettings, settings)
  } catch (error) {
    console.error('보안 설정 로드 실패:', error)
    message.error('보안 설정을 불러오는데 실패했습니다')
  }
}

const updateSecuritySetting = async (key: keyof SessionSecuritySettings, value: any) => {
  try {
    const updateData = { [key]: value }
    const updatedSettings = await sessionApi.updateSecuritySettings(updateData)
    Object.assign(securitySettings, updatedSettings)
    message.success('보안 설정이 업데이트되었습니다')
  } catch (error) {
    console.error('보안 설정 업데이트 실패:', error)
    message.error('보안 설정 업데이트에 실패했습니다')
    // 에러 시 이전 값으로 복원
    loadSecuritySettings()
  }
}

const goToSessionManagement = () => {
  router.push('/sessions')
}

const viewLoginHistory = async () => {
  showLoginHistoryModal.value = true
  await loadLoginHistory()
}

const viewActivityLog = async () => {
  showActivityLogModal.value = true
  await loadActivityLog()
}

const loadLoginHistory = async () => {
  loadingLoginHistory.value = true
  try {
    const response = await profileApi.getLoginHistory({
      page: loginHistoryPagination.page,
      limit: loginHistoryPagination.pageSize,
    })
    loginHistory.value = response.items
    loginHistoryPagination.itemCount = response.total
  } catch (error) {
    console.error('로그인 기록 로드 실패:', error)
    message.error('로그인 기록을 불러오는데 실패했습니다')
  } finally {
    loadingLoginHistory.value = false
  }
}

const loadActivityLog = async () => {
  loadingActivityLog.value = true
  try {
    const response = await profileApi.getActivityLog({
      page: activityLogPagination.page,
      limit: activityLogPagination.pageSize,
    })
    activityLog.value = response.items
    activityLogPagination.itemCount = response.total
  } catch (error) {
    console.error('활동 로그 로드 실패:', error)
    message.error('활동 로그를 불러오는데 실패했습니다')
  } finally {
    loadingActivityLog.value = false
  }
}

const handlePasswordChangeSuccess = () => {
  emit('passwordChange', true)
  // 프로파일 다시 로드하여 마지막 비밀번호 변경 시간 업데이트
  loadProfile()
}

const handlePasswordChangeError = () => {
  emit('passwordChange', false)
}

const handleTwoFactorSetupComplete = (settings: TwoFactorAuthSettings) => {
  emit('twoFactorSetup', settings)
  // 프로파일 다시 로드하여 2FA 상태 업데이트
  loadProfile()
}

const handleTwoFactorDisabled = () => {
  // 프로파일 다시 로드하여 2FA 상태 업데이트
  loadProfile()
}

// 라이프사이클
onMounted(async () => {
  await Promise.all([
    loadProfile(),
    loadSecuritySettings(),
  ])
})
</script>

<style scoped lang="scss">
.security-settings {
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
    .session-security,
    .access-restrictions,
    .activity-log {
      .security-item {
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
    }
  }

  .login-history,
  .activity-log-content {
    min-height: 400px;
  }
}

// 반응형 디자인
@media (max-width: 768px) {
  .security-settings {
    .settings-content {
      .session-security,
      .access-restrictions,
      .activity-log {
        .security-item {
          flex-direction: column;
          align-items: flex-start;
          gap: 12px;

          .item-action {
            width: 100%;
            justify-content: flex-end;
          }
        }
      }
    }
  }
}

@media (max-width: 480px) {
  .security-settings {
    .settings-content {
      .session-security,
      .access-restrictions,
      .activity-log {
        .security-item {
          .item-action {
            justify-content: flex-start;

            .n-select,
            .n-input-number {
              width: 100% !important;
            }
          }
        }
      }
    }
  }
}
</style>