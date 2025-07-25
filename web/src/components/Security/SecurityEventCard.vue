<template>
  <n-card 
    :class="cardClass"
    hoverable
    size="small"
  >
    <template #header>
      <div class="event-header">
        <div class="event-title">
          <n-icon 
            :component="getEventIcon(event)" 
            :size="18"
            :color="getSeverityColor(event.severity)"
          />
          <span class="title-text">{{ getEventTitle(event) }}</span>
          <n-tag 
            :type="getSeverityType(event.severity)" 
            size="small"
            class="severity-tag"
          >
            {{ getSeverityText(event.severity) }}
          </n-tag>
        </div>
        
        <div class="event-meta">
          <span class="timestamp">
            {{ formatRelativeTime(event.createdAt) }}
          </span>
        </div>
      </div>
    </template>

    <template #header-extra v-if="!compact">
      <n-dropdown 
        :options="actionOptions" 
        @select="handleActionSelect"
        placement="bottom-end"
      >
        <n-button 
          quaternary 
          circle 
          size="small"
        >
          <template #icon>
            <n-icon><DotsVertical /></n-icon>
          </template>
        </n-button>
      </n-dropdown>
    </template>

    <div class="event-content">
      <!-- 이벤트 설명 -->
      <p class="event-description">
        {{ event.description || getDefaultDescription(event) }}
      </p>

      <!-- 위치 및 디바이스 정보 -->
      <div class="event-details">
        <div class="detail-row">
          <n-icon :component="MapPin" size="14" />
          <span class="detail-label">위치:</span>
          <span class="detail-value">
            <code class="ip-address">{{ event.ipAddress }}</code>
            <span v-if="getLocationText(event)" class="location">
              • {{ getLocationText(event) }}
            </span>
          </span>
        </div>

        <div class="detail-row" v-if="event.userAgent">
          <n-icon :component="DeviceDesktop" size="14" />
          <span class="detail-label">디바이스:</span>
          <span class="detail-value device-info">
            {{ getDeviceInfo(event.userAgent) }}
          </span>
        </div>

        <!-- 위험도 (SuspiciousActivity인 경우) -->
        <div class="detail-row" v-if="isActivityType(event) && event.riskScore">
          <n-icon :component="AlertTriangle" size="14" />
          <span class="detail-label">위험도:</span>
          <div class="risk-score">
            <n-progress
              type="line"
              :percentage="event.riskScore"
              :color="getRiskColor(event.riskScore)"
              :show-indicator="false"
              height="6"
              style="width: 80px"
            />
            <span class="score-text">{{ event.riskScore }}/100</span>
          </div>
        </div>

        <!-- 활동 유형 (SuspiciousActivity인 경우) -->
        <div class="detail-row" v-if="isActivityType(event)">
          <n-icon :component="Flag" size="14" />
          <span class="detail-label">유형:</span>
          <n-tag size="small" type="warning">
            {{ getActivityTypeText(event.activityType) }}
          </n-tag>
        </div>

        <!-- 해결 상태 (SuspiciousActivity인 경우) -->
        <div class="detail-row" v-if="isActivityType(event) && event.isResolved">
          <n-icon :component="Check" size="14" />
          <span class="detail-label">해결됨:</span>
          <span class="detail-value resolved-info">
            {{ formatDateTime(event.resolvedAt) }}
            <span v-if="event.resolvedBy" class="resolved-by">
              by {{ event.resolvedBy }}
            </span>
          </span>
        </div>
      </div>

      <!-- 메타데이터 (간단히 표시) -->
      <div class="metadata" v-if="event.metadata && Object.keys(event.metadata).length > 0 && !compact">
        <n-collapse>
          <n-collapse-item title="추가 정보" name="metadata">
            <pre class="metadata-content">{{ JSON.stringify(event.metadata, null, 2) }}</pre>
          </n-collapse-item>
        </n-collapse>
      </div>
    </div>

    <template #action v-if="!compact">
      <div class="event-actions">
        <n-space>
          <n-button 
            size="small" 
            @click="handleViewDetails"
          >
            상세 보기
          </n-button>
          
          <n-button 
            v-if="isActivityType(event) && !event.isResolved && showResolveButton"
            size="small"
            type="primary"
            @click="handleResolve"
          >
            해결 처리
          </n-button>
          
          <n-button 
            v-if="event.severity === 'critical' || event.severity === 'high'"
            size="small"
            type="warning"
            ghost
            @click="handleReport"
          >
            신고
          </n-button>
        </n-space>
      </div>
    </template>

    <!-- 해결 모달 -->
    <n-modal
      v-model:show="showResolveModal"
      preset="dialog"
      title="의심스러운 활동 해결"
      positive-text="해결"
      negative-text="취소"
      @positive-click="confirmResolve"
    >
      <div class="resolve-form">
        <p>이 의심스러운 활동을 어떻게 해결하시겠습니까?</p>
        <n-input
          v-model:value="resolutionText"
          type="textarea"
          placeholder="해결 방법이나 조치 내용을 입력하세요..."
          :rows="3"
        />
      </div>
    </n-modal>
  </n-card>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { 
  NCard, 
  NIcon, 
  NTag, 
  NButton, 
  NDropdown, 
  NSpace, 
  NProgress,
  NCollapse,
  NCollapseItem,
  NModal,
  NInput,
  useMessage
} from 'naive-ui'
import { 
  DotsVertical, 
  MapPin, 
  DeviceDesktop, 
  AlertTriangle, 
  Shield, 
  Lock, 
  Eye, 
  Flag,
  Check,
  Login,
  Logout,
  ExclamationCircle
} from '@vicons/tabler'
import { formatDistanceToNow, format } from 'date-fns'
import { ko } from 'date-fns/locale'
import type { SessionSecurityEvent, SuspiciousActivity } from '@/types/api'

interface Props {
  event: SessionSecurityEvent | SuspiciousActivity
  compact?: boolean
  showResolveButton?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  compact: false,
  showResolveButton: false
})

const emit = defineEmits<{
  viewDetails: [event: SessionSecurityEvent | SuspiciousActivity]
  resolve: [activityId: string, resolution: string]
  report: [event: SessionSecurityEvent | SuspiciousActivity]
}>()

const message = useMessage()

// 상태 관리
const showResolveModal = ref(false)
const resolutionText = ref('')

// 계산된 속성
const cardClass = computed(() => {
  const classes = ['security-event-card']
  if (props.event.severity === 'critical') classes.push('critical-event')
  if (props.event.severity === 'high') classes.push('high-severity')
  if (isActivityType(props.event) && !props.event.isResolved) classes.push('unresolved-activity')
  if (props.compact) classes.push('compact-card')
  return classes.join(' ')
})

const actionOptions = computed(() => [
  {
    label: '상세 보기',
    key: 'details',
    icon: () => h(NIcon, { component: Eye })
  },
  {
    label: '내보내기',
    key: 'export',
    icon: () => h(NIcon, { component: Download })
  },
  ...(isActivityType(props.event) && !props.event.isResolved ? [{
    label: '해결 처리',
    key: 'resolve',
    icon: () => h(NIcon, { component: Check })
  }] : [])
])

// 타입 가드
const isActivityType = (event: any): event is SuspiciousActivity => {
  return 'activityType' in event && 'riskScore' in event
}

// 메소드
const getEventIcon = (event: SessionSecurityEvent | SuspiciousActivity) => {
  if (isActivityType(event)) {
    const iconMap = {
      unusual_location: MapPin,
      unusual_device: DeviceDesktop,
      brute_force: AlertTriangle,
      credential_stuffing: Lock,
      account_takeover: ExclamationCircle,
      privilege_escalation: Shield
    }
    return iconMap[event.activityType] || AlertTriangle
  }
  
  const iconMap = {
    login: Login,
    logout: Logout,
    suspicious_activity: AlertTriangle,
    password_change: Lock,
    device_change: DeviceDesktop,
    location_change: MapPin
  }
  return iconMap[event.eventType] || Shield
}

const getEventTitle = (event: SessionSecurityEvent | SuspiciousActivity) => {
  if (isActivityType(event)) {
    return getActivityTypeText(event.activityType)
  }
  
  const titleMap = {
    login: '로그인',
    logout: '로그아웃',
    suspicious_activity: '의심스러운 활동',
    password_change: '패스워드 변경',
    device_change: '디바이스 변경',
    location_change: '위치 변경'
  }
  return titleMap[event.eventType] || '보안 이벤트'
}

const getActivityTypeText = (activityType: string) => {
  const typeMap = {
    unusual_location: '비정상적인 위치',
    unusual_device: '비정상적인 디바이스',
    brute_force: '무차별 대입 공격',
    credential_stuffing: '크리덴셜 스터핑',
    account_takeover: '계정 탈취',
    privilege_escalation: '권한 상승'
  }
  return typeMap[activityType] || activityType
}

const getSeverityColor = (severity: string) => {
  const colorMap = {
    low: '#27ae60',
    medium: '#f39c12',
    high: '#e67e22',
    critical: '#e74c3c'
  }
  return colorMap[severity] || '#666'
}

const getSeverityType = (severity: string) => {
  const typeMap = {
    low: 'success',
    medium: 'warning',
    high: 'error',
    critical: 'error'
  }
  return typeMap[severity] || 'default'
}

const getSeverityText = (severity: string) => {
  const textMap = {
    low: '낮음',
    medium: '보통',
    high: '높음',
    critical: '매우 높음'
  }
  return textMap[severity] || severity
}

const getRiskColor = (score: number) => {
  if (score >= 80) return '#e74c3c'
  if (score >= 60) return '#f39c12'
  if (score >= 40) return '#3498db'
  return '#27ae60'
}

const getLocationText = (event: SessionSecurityEvent | SuspiciousActivity) => {
  if ('location' in event && event.location) {
    return [event.location.city, event.location.country].filter(Boolean).join(', ')
  }
  return null
}

const getDeviceInfo = (userAgent: string) => {
  // 간단한 User Agent 파싱
  const browser = userAgent.match(/(Chrome|Firefox|Safari|Edge|Opera)\/[\d.]+/)?.[0] || '알 수 없음'
  const os = userAgent.match(/(Windows|Mac|Linux|Android|iOS)[\s\w\d.]*/)?.['1'] || '알 수 없음'
  return `${browser} on ${os}`
}

const getDefaultDescription = (event: SessionSecurityEvent | SuspiciousActivity) => {
  if (isActivityType(event)) {
    return `${getActivityTypeText(event.activityType)} 활동이 감지되었습니다.`
  }
  return `${getEventTitle(event)} 이벤트가 발생했습니다.`
}

const formatRelativeTime = (dateString: string) => {
  return formatDistanceToNow(new Date(dateString), { addSuffix: true, locale: ko })
}

const formatDateTime = (dateString?: string) => {
  if (!dateString) return ''
  return format(new Date(dateString), 'MM/dd HH:mm', { locale: ko })
}

const handleViewDetails = () => {
  emit('viewDetails', props.event)
}

const handleResolve = () => {
  showResolveModal.value = true
  resolutionText.value = ''
}

const handleReport = () => {
  emit('report', props.event)
}

const handleActionSelect = (key: string) => {
  switch (key) {
    case 'details':
      handleViewDetails()
      break
    case 'resolve':
      handleResolve()
      break
    case 'export':
      // 내보내기 로직
      message.info('내보내기 기능 구현 예정')
      break
  }
}

const confirmResolve = () => {
  if (!resolutionText.value.trim()) {
    message.error('해결 방법을 입력해주세요')
    return false
  }
  
  if (isActivityType(props.event)) {
    emit('resolve', props.event.id, resolutionText.value)
  }
  
  showResolveModal.value = false
  return true
}
</script>

<style scoped>
.security-event-card {
  margin-bottom: 16px;
  transition: all 0.3s ease;
}

.security-event-card.critical-event {
  border-left: 4px solid #e74c3c;
}

.security-event-card.high-severity {
  border-left: 4px solid #f39c12;
}

.security-event-card.unresolved-activity {
  background-color: rgba(231, 76, 60, 0.02);
}

.security-event-card.compact-card {
  margin-bottom: 8px;
}

.event-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

.event-title {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
}

.title-text {
  font-weight: 500;
  font-size: 14px;
}

.severity-tag {
  margin-left: auto;
}

.event-meta {
  text-align: right;
  margin-left: 16px;
}

.timestamp {
  font-size: 12px;
  color: var(--text-color-3);
}

.event-content {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.event-description {
  margin: 0;
  color: var(--text-color-2);
  font-size: 13px;
  line-height: 1.4;
}

.event-details {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.detail-row {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
}

.detail-label {
  font-weight: 500;
  color: var(--text-color-2);
  min-width: 50px;
}

.detail-value {
  color: var(--text-color-1);
  flex: 1;
}

.ip-address {
  background: var(--code-color);
  padding: 1px 4px;
  border-radius: 3px;
  font-family: 'Fira Code', monospace;
  font-size: 11px;
}

.location {
  color: var(--text-color-3);
  font-size: 11px;
}

.device-info {
  font-size: 11px;
}

.risk-score {
  display: flex;
  align-items: center;
  gap: 6px;
}

.score-text {
  font-size: 11px;
  font-weight: 500;
  min-width: 35px;
}

.resolved-info {
  font-size: 11px;
}

.resolved-by {
  color: var(--text-color-3);
}

.metadata {
  margin-top: 8px;
}

.metadata-content {
  font-size: 11px;
  background: var(--code-color);
  padding: 8px;
  border-radius: 4px;
  margin: 0;
  overflow-x: auto;
}

.event-actions {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--border-color);
}

.resolve-form {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.resolve-form p {
  margin: 0;
  color: var(--text-color-2);
}

@media (max-width: 768px) {
  .event-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }
  
  .event-meta {
    margin-left: 0;
    text-align: left;
  }
  
  .detail-row {
    flex-wrap: wrap;
  }
  
  .detail-label {
    min-width: auto;
  }
}
</style>