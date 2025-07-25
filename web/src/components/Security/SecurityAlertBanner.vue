<template>
  <div class="security-alert-banner" v-if="visibleAlerts.length > 0">
    <transition-group
      name="alert-slide"
      tag="div"
      class="alerts-container"
    >
      <NAlert
        v-for="alert in visibleAlerts"
        :key="alert.id"
        :type="getAlertType(alert.severity)"
        :show-icon="true"
        :closable="true"
        class="security-alert"
        @close="handleDismiss(alert.id)"
      >
        <template #icon>
          <NIcon :component="getAlertIcon(alert.type)" />
        </template>

        <div class="alert-content">
          <div class="alert-header">
            <span class="alert-title">{{ getAlertTitle(alert) }}</span>
            <NTag
              :type="getSeverityTagType(alert.severity)"
              size="small"
              class="severity-tag"
            >
              {{ getSeverityText(alert.severity) }}
            </NTag>
            <span class="alert-time">{{ formatTime(alert.timestamp) }}</span>
          </div>

          <div class="alert-message">
            {{ alert.message }}
          </div>

          <div class="alert-details" v-if="alert.details">
            <div class="detail-item" v-if="alert.details.ipAddress">
              <NIcon :component="MapPin" size="12" />
              <span>{{ alert.details.ipAddress }}</span>
              <span v-if="alert.details.location" class="location">
                ({{ alert.details.location }})
              </span>
            </div>

            <div class="detail-item" v-if="alert.details.device">
              <NIcon :component="DeviceDesktop" size="12" />
              <span>{{ alert.details.device }}</span>
            </div>

            <div class="detail-item" v-if="alert.details.riskScore">
              <NIcon :component="AlertTriangle" size="12" />
              <span>위험도: {{ alert.details.riskScore }}/100</span>
            </div>
          </div>

          <div class="alert-actions" v-if="alert.actions && alert.actions.length > 0">
            <NSpace size="small">
              <NButton
                v-for="action in alert.actions"
                :key="action.key"
                :type="action.type || 'default'"
                size="small"
                @click="handleAction(alert.id, action.key)"
              >
                {{ action.label }}
              </NButton>
            </NSpace>
          </div>
        </div>
      </NAlert>
    </transition-group>

    <!-- 더 많은 알림이 있을 때 요약 표시 -->
    <div class="more-alerts" v-if="hiddenAlertsCount > 0">
      <NButton
        text
        type="primary"
        size="small"
        @click="showAllAlerts"
      >
        +{{ hiddenAlertsCount }}개의 추가 알림 보기
      </NButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import {
  NAlert,
  NButton,
  NIcon,
  NSpace,
  NTag,
  useMessage,
} from 'naive-ui'
import {
  AlertTriangle,
  CheckCircle,
  DeviceDesktop,
  ExclamationCircle,
  InfoCircle,
  Lock,
  MapPin,
  Shield,
  XCircle,
} from '@vicons/tabler'
import { formatDistanceToNow } from 'date-fns'
import { ko } from 'date-fns/locale'

interface SecurityAlert {
  id: string
  type: 'suspicious_login' | 'new_device' | 'location_change' | 'brute_force' | 'account_lockout' | 'password_breach' | 'permission_change'
  severity: 'low' | 'medium' | 'high' | 'critical'
  title?: string
  message: string
  timestamp: string
  details?: {
    ipAddress?: string
    location?: string
    device?: string
    riskScore?: number
    attempts?: number
    [key: string]: any
  }
  actions?: Array<{
    key: string
    label: string
    type?: 'primary' | 'error' | 'warning' | 'success'
  }>
  dismissed?: boolean
  persistent?: boolean // true면 자동으로 사라지지 않음
}

interface Props {
  alerts: SecurityAlert[]
  maxVisible?: number
  autoHide?: boolean
  hideDelay?: number // 밀리초
}

const props = withDefaults(defineProps<Props>(), {
  alerts: () => [],
  maxVisible: 3,
  autoHide: true,
  hideDelay: 10000, // 10초
})

const emit = defineEmits<{
  dismiss: [alertId: string]
  action: [alertId: string, actionKey: string]
  showAll: []
}>()

const message = useMessage()

// 상태 관리
const dismissedAlerts = ref<Set<string>>(new Set())
const autoHideTimers = ref<Map<string, NodeJS.Timeout>>(new Map())

// 계산된 속성
const visibleAlerts = computed(() => {
  const filteredAlerts = props.alerts.filter(alert => !dismissedAlerts.value.has(alert.id))
  return filteredAlerts.slice(0, props.maxVisible)
})

const hiddenAlertsCount = computed(() => {
  const filteredAlerts = props.alerts.filter(alert => !dismissedAlerts.value.has(alert.id))
  return Math.max(0, filteredAlerts.length - props.maxVisible)
})

// 메소드
const getAlertType = (severity: string) => {
  const typeMap = {
    low: 'info',
    medium: 'warning',
    high: 'error',
    critical: 'error',
  }
  return typeMap[severity] || 'info'
}

const getAlertIcon = (type: string) => {
  const iconMap = {
    suspicious_login: AlertTriangle,
    new_device: DeviceDesktop,
    location_change: MapPin,
    brute_force: XCircle,
    account_lockout: Lock,
    password_breach: ExclamationCircle,
    permission_change: Shield,
  }
  return iconMap[type] || InfoCircle
}

const getSeverityTagType = (severity: string) => {
  const typeMap = {
    low: 'success',
    medium: 'warning',
    high: 'error',
    critical: 'error',
  }
  return typeMap[severity] || 'default'
}

const getSeverityText = (severity: string) => {
  const textMap = {
    low: '낮음',
    medium: '보통',
    high: '높음',
    critical: '매우 높음',
  }
  return textMap[severity] || severity
}

const getAlertTitle = (alert: SecurityAlert) => {
  if (alert.title) return alert.title

  const titleMap = {
    suspicious_login: '의심스러운 로그인',
    new_device: '새 디바이스 접근',
    location_change: '위치 변경 감지',
    brute_force: '무차별 대입 공격',
    account_lockout: '계정 잠금',
    password_breach: '패스워드 유출',
    permission_change: '권한 변경',
  }
  return titleMap[alert.type] || '보안 알림'
}

const formatTime = (timestamp: string) => {
  return formatDistanceToNow(new Date(timestamp), { addSuffix: true, locale: ko })
}

const handleDismiss = (alertId: string) => {
  dismissedAlerts.value.add(alertId)
  clearAutoHideTimer(alertId)
  emit('dismiss', alertId)
}

const handleAction = (alertId: string, actionKey: string) => {
  emit('action', alertId, actionKey)

  // 액션 실행 후 자동으로 알림 제거 (설정에 따라)
  const alert = props.alerts.find(a => a.id === alertId)
  if (alert && !alert.persistent) {
    setTimeout(() => {
      handleDismiss(alertId)
    }, 1000)
  }
}

const showAllAlerts = () => {
  emit('showAll')
}

const setupAutoHide = (alert: SecurityAlert) => {
  if (!props.autoHide || alert.persistent) return

  const timer = setTimeout(() => {
    handleDismiss(alert.id)
  }, props.hideDelay)

  autoHideTimers.value.set(alert.id, timer)
}

const clearAutoHideTimer = (alertId: string) => {
  const timer = autoHideTimers.value.get(alertId)
  if (timer) {
    clearTimeout(timer)
    autoHideTimers.value.delete(alertId)
  }
}

const clearAllTimers = () => {
  autoHideTimers.value.forEach(timer => clearTimeout(timer))
  autoHideTimers.value.clear()
}

// 감시자
watch(() => props.alerts, (newAlerts, oldAlerts) => {
  // 새로운 알림에 대해 자동 숨기기 타이머 설정
  const newAlertIds = new Set(newAlerts.map(a => a.id))
  const oldAlertIds = new Set((oldAlerts || []).map(a => a.id))

  newAlerts.forEach(alert => {
    if (!oldAlertIds.has(alert.id)) {
      setupAutoHide(alert)
    }
  })

  // 제거된 알림의 타이머 정리
  oldAlertIds.forEach(alertId => {
    if (!newAlertIds.has(alertId)) {
      clearAutoHideTimer(alertId)
    }
  })
}, { deep: true })

// 생명주기
onMounted(() => {
  // 초기 알림에 대해 자동 숨기기 타이머 설정
  props.alerts.forEach(alert => {
    setupAutoHide(alert)
  })
})

onUnmounted(() => {
  clearAllTimers()
})
</script>

<style scoped>
.security-alert-banner {
  position: sticky;
  top: 0;
  z-index: 1000;
  background: var(--body-color);
  padding: 16px;
  border-bottom: 1px solid var(--border-color);
}

.alerts-container {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.security-alert {
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.alert-content {
  width: 100%;
}

.alert-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
  flex-wrap: wrap;
}

.alert-title {
  font-weight: 600;
  font-size: 14px;
}

.severity-tag {
  margin-left: auto;
}

.alert-time {
  font-size: 12px;
  color: var(--text-color-3);
  margin-left: auto;
}

.alert-message {
  font-size: 13px;
  color: var(--text-color-2);
  margin-bottom: 8px;
  line-height: 1.4;
}

.alert-details {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-bottom: 8px;
}

.detail-item {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: var(--text-color-3);
}

.location {
  font-style: italic;
}

.alert-actions {
  margin-top: 8px;
}

.more-alerts {
  margin-top: 8px;
  text-align: center;
  padding: 8px;
  background: var(--hover-color);
  border-radius: 6px;
}

/* 애니메이션 */
.alert-slide-enter-active,
.alert-slide-leave-active {
  transition: all 0.3s ease;
}

.alert-slide-enter-from {
  opacity: 0;
  transform: translateY(-20px);
}

.alert-slide-leave-to {
  opacity: 0;
  transform: translateX(100%);
}

.alert-slide-move {
  transition: transform 0.3s ease;
}

/* 반응형 */
@media (max-width: 768px) {
  .security-alert-banner {
    padding: 12px;
  }

  .alert-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 4px;
  }

  .severity-tag,
  .alert-time {
    margin-left: 0;
    align-self: flex-start;
  }

  .alert-details {
    flex-direction: column;
    gap: 4px;
  }
}

/* 심각도별 특별 스타일 */
.security-alert :deep(.n-alert--error-type) {
  border-left: 4px solid #e74c3c;
}

.security-alert :deep(.n-alert--warning-type) {
  border-left: 4px solid #f39c12;
}

/* 호버 효과 */
.security-alert:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}
</style>