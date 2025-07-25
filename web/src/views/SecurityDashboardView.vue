<template>
  <div class="security-dashboard">
    <!-- 헤더 섹션 -->
    <div class="header">
      <div class="title-section">
        <h1>보안 대시보드</h1>
        <p class="subtitle">계정 보안 상태를 모니터링하고 위험 요소를 관리합니다</p>
      </div>

      <!-- 보안 상태 요약 -->
      <div class="security-overview">
        <NSpace>
          <NStatistic
            v-if="securityStats"
            label="보안 위험도"
            :value="securityStats.riskScore"
            suffix="%"
            :value-style="getRiskScoreStyle(securityStats.riskScore)"
          />
          <NStatistic
            v-if="securityStats"
            label="총 로그인"
            :value="securityStats.totalLogins"
            :show-indicator="true"
          />
          <NStatistic
            v-if="securityStats && securityStats.suspiciousAttempts > 0"
            label="의심스러운 시도"
            :value="securityStats.suspiciousAttempts"
            :value-style="{ color: '#e74c3c' }"
          />
          <NStatistic
            v-if="securityStats"
            label="고유 디바이스"
            :value="securityStats.uniqueDevices"
          />
        </NSpace>
      </div>
    </div>

    <!-- 실시간 보안 알림 배너 -->
    <SecurityAlertBanner
      v-if="hasActiveAlerts"
      :alerts="realtimeAlerts"
      @dismiss="handleDismissAlert"
    />

    <!-- 메인 콘텐츠 -->
    <div class="content">
      <NTabs v-model:value="activeTab" type="line" animated>
        <!-- 보안 개요 탭 -->
        <NTabPane name="overview" tab="보안 개요">
          <NGrid :cols="24" :x-gap="16" :y-gap="16">
            <!-- 보안 통계 차트 -->
            <NGridItem :span="12">
              <NCard title="로그인 추세" embedded>
                <SecurityStatsChart
                  :stats="securityStats"
                  chart-type="login-trends"
                />
              </NCard>
            </NGridItem>

            <!-- 위험도 분포 차트 -->
            <NGridItem :span="12">
              <NCard title="위험도 분포" embedded>
                <SecurityStatsChart
                  :stats="securityStats"
                  chart-type="risk-distribution"
                />
              </NCard>
            </NGridItem>

            <!-- 최근 보안 이벤트 -->
            <NGridItem :span="24">
              <NCard title="최근 보안 이벤트" embedded>
                <template #header-extra>
                  <NButton
                    text
                    type="primary"
                    @click="$router.push('/security/events')"
                  >
                    전체 보기
                  </NButton>
                </template>
                <div class="recent-events">
                  <SecurityEventCard
                    v-for="event in recentSecurityEvents"
                    :key="event.id"
                    :event="event"
                    :compact="true"
                    @view-details="handleViewEventDetails"
                  />
                  <NEmpty
                    v-if="!recentSecurityEvents.length"
                    description="최근 보안 이벤트가 없습니다"
                  />
                </div>
              </NCard>
            </NGridItem>
          </NGrid>
        </NTabPane>

        <!-- 로그인 이력 탭 -->
        <NTabPane name="login-history" tab="로그인 이력">
          <div class="login-history-section">
            <div class="section-header">
              <h3>최근 로그인 이력</h3>
              <NSpace>
                <NButton
                  @click="refreshLoginHistory"
                  :loading="loadingLoginHistory"
                >
                  <template #icon>
                    <NIcon><Refresh /></NIcon>
                  </template>
                  새로고침
                </NButton>
                <NButton
                  type="primary"
                  @click="$router.push('/security/login-history')"
                >
                  전체 이력 보기
                </NButton>
              </NSpace>
            </div>

            <LoginHistoryTable
              :data="loginHistory"
              :loading="loadingLoginHistory"
              :limit="10"
              @refresh="refreshLoginHistory"
            />
          </div>
        </NTabPane>

        <!-- 의심스러운 활동 탭 -->
        <NTabPane name="suspicious" tab="의심스러운 활동">
          <div class="suspicious-activities-section">
            <div class="section-header">
              <h3>미해결 의심스러운 활동</h3>
              <NSpace>
                <NButton
                  @click="refreshSuspiciousActivities"
                  :loading="loadingSuspiciousActivities"
                >
                  <template #icon>
                    <NIcon><Refresh /></NIcon>
                  </template>
                  새로고침
                </NButton>
              </NSpace>
            </div>

            <div class="suspicious-activities-grid">
              <SecurityEventCard
                v-for="activity in suspiciousActivities"
                :key="activity.id"
                :event="activity"
                :show-resolve-button="true"
                @resolve="handleResolveSuspiciousActivity"
                @view-details="handleViewActivityDetails"
              />
              <NEmpty
                v-if="!suspiciousActivities.length"
                description="해결되지 않은 의심스러운 활동이 없습니다"
              />
            </div>
          </div>
        </NTabPane>

        <!-- 보안 설정 탭 -->
        <NTabPane name="settings" tab="보안 설정">
          <div class="security-settings-section">
            <NCard title="실시간 보안 알림 설정" embedded>
              <div class="alert-settings">
                <NForm
                  :model="alertSettings"
                  label-placement="left"
                  label-width="auto"
                >
                  <NFormItem label="실시간 알림 활성화">
                    <NSwitch
                      v-model:value="alertSettings.enableRealTimeAlerts"
                      @update:value="handleUpdateAlertSettings"
                    />
                  </NFormItem>

                  <NFormItem label="의심스러운 로그인 알림">
                    <NSwitch
                      v-model:value="alertSettings.notifyOnSuspiciousLogin"
                      :disabled="!alertSettings.enableRealTimeAlerts"
                      @update:value="handleUpdateAlertSettings"
                    />
                  </NFormItem>

                  <NFormItem label="새 디바이스 알림">
                    <NSwitch
                      v-model:value="alertSettings.notifyOnNewDevice"
                      :disabled="!alertSettings.enableRealTimeAlerts"
                      @update:value="handleUpdateAlertSettings"
                    />
                  </NFormItem>

                  <NFormItem label="위치 변경 알림">
                    <NSwitch
                      v-model:value="alertSettings.notifyOnLocationChange"
                      :disabled="!alertSettings.enableRealTimeAlerts"
                      @update:value="handleUpdateAlertSettings"
                    />
                  </NFormItem>

                  <NFormItem label="알림 임계값">
                    <NSelect
                      v-model:value="alertSettings.alertThreshold"
                      :options="alertThresholdOptions"
                      :disabled="!alertSettings.enableRealTimeAlerts"
                      @update:value="handleUpdateAlertSettings"
                    />
                  </NFormItem>
                </NForm>
              </div>
            </NCard>
          </div>
        </NTabPane>
      </NTabs>
    </div>

    <!-- 이벤트 상세 모달 -->
    <NModal
      v-model:show="showEventDetailModal"
      preset="card"
      title="보안 이벤트 상세"
      style="width: 600px"
    >
      <SecurityEventDetails
        v-if="selectedEvent"
        :event="selectedEvent"
      />
    </NModal>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import {
  NButton,
  NCard,
  NEmpty,
  NForm,
  NFormItem,
  NGrid,
  NGridItem,
  NIcon,
  NModal,
  NSelect,
  NSpace,
  NStatistic,
  NSwitch,
  NTabPane,
  NTabs,
} from 'naive-ui'
import { Refresh } from '@vicons/tabler'
import { authApi } from '@/api/services/auth'
import SecurityAlertBanner from '@/components/Security/SecurityAlertBanner.vue'
import SecurityEventCard from '@/components/Security/SecurityEventCard.vue'
import SecurityStatsChart from '@/components/Security/SecurityStatsChart.vue'
import LoginHistoryTable from '@/components/Security/LoginHistoryTable.vue'
import SecurityEventDetails from '@/components/Security/SecurityEventDetails.vue'
import type {
  LoginHistory,
  SecurityStats,
  SessionSecurityEvent,
  SuspiciousActivity,
} from '@/types/api'

const router = useRouter()
const message = useMessage()

// 상태 관리
const activeTab = ref('overview')
const securityStats = ref<SecurityStats | null>(null)
const recentSecurityEvents = ref<SessionSecurityEvent[]>([])
const loginHistory = ref<LoginHistory[]>([])
const suspiciousActivities = ref<SuspiciousActivity[]>([])
const realtimeAlerts = ref<any[]>([])
const selectedEvent = ref<any | null>(null)

// 로딩 상태
const loadingStats = ref(false)
const loadingEvents = ref(false)
const loadingLoginHistory = ref(false)
const loadingSuspiciousActivities = ref(false)

// 모달 상태
const showEventDetailModal = ref(false)

// 보안 알림 설정
const alertSettings = ref({
  enableRealTimeAlerts: true,
  notifyOnSuspiciousLogin: true,
  notifyOnNewDevice: true,
  notifyOnLocationChange: false,
  notifyOnHighRiskActivity: true,
  alertThreshold: 'medium' as 'low' | 'medium' | 'high' | 'critical',
})

// 옵션
const alertThresholdOptions = [
  { label: '낮음', value: 'low' },
  { label: '보통', value: 'medium' },
  { label: '높음', value: 'high' },
  { label: '매우 높음', value: 'critical' },
]

// 계산된 속성
const hasActiveAlerts = computed(() => realtimeAlerts.value.length > 0)

// 메소드
const getRiskScoreStyle = (score: number) => {
  if (score >= 80) return { color: '#e74c3c' }
  if (score >= 60) return { color: '#f39c12' }
  if (score >= 40) return { color: '#f1c40f' }
  return { color: '#27ae60' }
}

const loadSecurityStats = async () => {
  try {
    loadingStats.value = true
    securityStats.value = await authApi.getSecurityStats('30d')
  } catch (error) {
    message.error('보안 통계를 불러오는데 실패했습니다')
    console.error('Error loading security stats:', error)
  } finally {
    loadingStats.value = false
  }
}

const loadRecentSecurityEvents = async () => {
  try {
    loadingEvents.value = true
    const response = await authApi.getSecurityEvents({ limit: 5 })
    recentSecurityEvents.value = response.items
  } catch (error) {
    message.error('최근 보안 이벤트를 불러오는데 실패했습니다')
    console.error('Error loading security events:', error)
  } finally {
    loadingEvents.value = false
  }
}

const refreshLoginHistory = async () => {
  try {
    loadingLoginHistory.value = true
    const response = await authApi.getLoginHistory({ limit: 10 })
    loginHistory.value = response.items
  } catch (error) {
    message.error('로그인 이력을 불러오는데 실패했습니다')
    console.error('Error loading login history:', error)
  } finally {
    loadingLoginHistory.value = false
  }
}

const refreshSuspiciousActivities = async () => {
  try {
    loadingSuspiciousActivities.value = true
    const response = await authApi.getSuspiciousActivities({ limit: 20 })
    suspiciousActivities.value = response.items.filter(activity => !activity.isResolved)
  } catch (error) {
    message.error('의심스러운 활동을 불러오는데 실패했습니다')
    console.error('Error loading suspicious activities:', error)
  } finally {
    loadingSuspiciousActivities.value = false
  }
}

const loadSecurityAlertSettings = async () => {
  try {
    const settings = await authApi.getSecurityAlertSettings()
    alertSettings.value = { ...alertSettings.value, ...settings }
  } catch (error) {
    console.error('Error loading alert settings:', error)
  }
}

const handleViewEventDetails = (event: any) => {
  selectedEvent.value = event
  showEventDetailModal.value = true
}

const handleViewActivityDetails = (activity: any) => {
  selectedEvent.value = activity
  showEventDetailModal.value = true
}

const handleResolveSuspiciousActivity = async (activityId: string, resolution: string) => {
  try {
    await authApi.resolveSuspiciousActivity(activityId, resolution)
    message.success('의심스러운 활동이 해결되었습니다')
    await refreshSuspiciousActivities()
  } catch (error) {
    message.error('활동 해결에 실패했습니다')
    console.error('Error resolving suspicious activity:', error)
  }
}

const handleDismissAlert = (alertId: string) => {
  realtimeAlerts.value = realtimeAlerts.value.filter(alert => alert.id !== alertId)
}

const handleUpdateAlertSettings = async () => {
  try {
    await authApi.updateSecurityAlertSettings(alertSettings.value)
    message.success('보안 알림 설정이 업데이트되었습니다')
  } catch (error) {
    message.error('설정 업데이트에 실패했습니다')
    console.error('Error updating alert settings:', error)
  }
}

// 생명주기
onMounted(async () => {
  await Promise.all([
    loadSecurityStats(),
    loadRecentSecurityEvents(),
    refreshLoginHistory(),
    refreshSuspiciousActivities(),
    loadSecurityAlertSettings(),
  ])
})

onUnmounted(() => {
  // WebSocket 연결 정리 등
})
</script>

<style scoped>
.security-dashboard {
  padding: 24px;
  max-width: 1200px;
  margin: 0 auto;
}

.header {
  margin-bottom: 24px;
}

.title-section {
  margin-bottom: 16px;
}

.title-section h1 {
  margin: 0 0 8px 0;
  font-size: 28px;
  font-weight: 600;
  color: var(--text-color-1);
}

.subtitle {
  margin: 0;
  color: var(--text-color-2);
  font-size: 14px;
}

.security-overview {
  margin-bottom: 16px;
}

.content {
  background: var(--card-color);
  border-radius: 8px;
  overflow: hidden;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.section-header h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
}

.recent-events {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.suspicious-activities-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(400px, 1fr));
  gap: 16px;
}

.alert-settings {
  max-width: 600px;
}

.login-history-section,
.suspicious-activities-section,
.security-settings-section {
  padding: 16px;
}

@media (max-width: 768px) {
  .security-dashboard {
    padding: 16px;
  }

  .suspicious-activities-grid {
    grid-template-columns: 1fr;
  }
}
</style>