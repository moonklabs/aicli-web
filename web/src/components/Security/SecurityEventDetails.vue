<template>
  <div class="security-event-details">
    <NDescriptions :column="2" bordered>
      <!-- 기본 정보 -->
      <NDescriptionsItem label="이벤트 ID">
        <code class="event-id">{{ event.id }}</code>
      </NDescriptionsItem>

      <NDescriptionsItem label="발생 시간">
        {{ formatDateTime(event.createdAt) }}
      </NDescriptionsItem>

      <NDescriptionsItem label="심각도">
        <NTag :type="getSeverityType(event.severity)" size="small">
          {{ getSeverityText(event.severity) }}
        </NTag>
      </NDescriptionsItem>

      <NDescriptionsItem label="이벤트 유형">
        <div class="event-type">
          <NIcon :component="getEventIcon()" style="margin-right: 4px" />
          {{ getEventTypeText() }}
        </div>
      </NDescriptionsItem>

      <!-- 네트워크 정보 -->
      <NDescriptionsItem label="IP 주소">
        <code class="ip-address">{{ event.ipAddress }}</code>
      </NDescriptionsItem>

      <NDescriptionsItem label="위치" v-if="getLocationText()">
        <div class="location-info">
          <NIcon :component="MapPin" size="14" style="margin-right: 4px" />
          {{ getLocationText() }}
        </div>
      </NDescriptionsItem>

      <!-- 의심스러운 활동 전용 필드 -->
      <template v-if="isActivityType(event)">
        <NDescriptionsItem label="위험도 점수">
          <div class="risk-score">
            <NProgress
              type="line"
              :percentage="event.riskScore"
              :color="getRiskColor(event.riskScore)"
              :show-indicator="false"
              height="8"
              style="width: 120px"
            />
            <span class="score-text">{{ event.riskScore }}/100</span>
          </div>
        </NDescriptionsItem>

        <NDescriptionsItem label="활동 유형">
          <NTag type="warning" size="small">
            {{ getActivityTypeText(event.activityType) }}
          </NTag>
        </NDescriptionsItem>

        <NDescriptionsItem label="위험 지표" span="2" v-if="event.indicators.length > 0">
          <NSpace>
            <NTag
              v-for="indicator in event.indicators"
              :key="indicator"
              size="small"
              type="error"
              bordered
            >
              {{ indicator }}
            </NTag>
          </NSpace>
        </NDescriptionsItem>

        <NDescriptionsItem label="해결 상태">
          <NTag :type="event.isResolved ? 'success' : 'error'" size="small">
            {{ event.isResolved ? '해결됨' : '미해결' }}
          </NTag>
        </NDescriptionsItem>

        <NDescriptionsItem label="해결 정보" v-if="event.isResolved">
          <div class="resolution-info">
            <div>해결일시: {{ formatDateTime(event.resolvedAt) }}</div>
            <div v-if="event.resolvedBy">해결자: {{ event.resolvedBy }}</div>
            <div v-if="event.resolution" class="resolution-text">
              {{ event.resolution }}
            </div>
          </div>
        </NDescriptionsItem>
      </template>

      <!-- 세션 보안 이벤트 전용 필드 -->
      <template v-if="isSecurityEventType(event)">
        <NDescriptionsItem label="사용자 ID">
          <code>{{ event.userId }}</code>
        </NDescriptionsItem>

        <NDescriptionsItem label="세션 ID" v-if="event.sessionId">
          <code>{{ event.sessionId }}</code>
        </NDescriptionsItem>
      </template>

      <NDescriptionsItem label="User Agent" span="2">
        <code class="user-agent">{{ event.userAgent }}</code>
      </NDescriptionsItem>
    </NDescriptions>

    <!-- 이벤트 설명 -->
    <NDivider>이벤트 설명</NDivider>
    <div class="event-description">
      <p>{{ event.description || getDefaultDescription() }}</p>
    </div>

    <!-- 메타데이터 -->
    <div class="metadata-section" v-if="event.metadata && Object.keys(event.metadata).length > 0">
      <NDivider>추가 정보</NDivider>
      <NCollapse>
        <NCollapseItem title="메타데이터" name="metadata">
          <pre class="metadata-content">{{ JSON.stringify(event.metadata, null, 2) }}</pre>
        </NCollapseItem>
      </NCollapse>
    </div>

    <!-- 위치 정보 상세 (있는 경우) -->
    <div class="location-section" v-if="hasDetailedLocation()">
      <NDivider>위치 정보</NDivider>
      <NDescriptions :column="2" bordered>
        <NDescriptionsItem label="국가" v-if="getLocation()?.country">
          {{ getLocation()?.country }}
        </NDescriptionsItem>

        <NDescriptionsItem label="도시" v-if="getLocation()?.city">
          {{ getLocation()?.city }}
        </NDescriptionsItem>

        <NDescriptionsItem label="시간대" v-if="getLocation()?.timezone">
          {{ getLocation()?.timezone }}
        </NDescriptionsItem>
      </NDescriptions>
    </div>

    <!-- 권장 조치 -->
    <div class="recommendations-section" v-if="getRecommendations().length > 0">
      <NDivider>권장 조치</NDivider>
      <NAlert type="info" style="margin-bottom: 16px">
        <ul class="recommendations-list">
          <li v-for="recommendation in getRecommendations()" :key="recommendation">
            {{ recommendation }}
          </li>
        </ul>
      </NAlert>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import {
  NAlert,
  NCollapse,
  NCollapseItem,
  NDescriptions,
  NDescriptionsItem,
  NDivider,
  NIcon,
  NProgress,
  NSpace,
  NTag,
} from 'naive-ui'
import {
  AlertTriangle,
  DeviceDesktop,
  ExclamationCircle,
  Lock,
  Login,
  Logout,
  MapPin,
  Shield,
} from '@vicons/tabler'
import { format } from 'date-fns'
import { ko } from 'date-fns/locale'
import type { SessionSecurityEvent, SuspiciousActivity } from '@/types/api'

interface Props {
  event: SessionSecurityEvent | SuspiciousActivity
}

const props = defineProps<Props>()

// 타입 가드
const isActivityType = (event: any): event is SuspiciousActivity => {
  return 'activityType' in event && 'riskScore' in event
}

const isSecurityEventType = (event: any): event is SessionSecurityEvent => {
  return 'eventType' in event && 'sessionId' in event
}

// 메소드
const formatDateTime = (dateString?: string) => {
  if (!dateString) return ''
  return format(new Date(dateString), 'yyyy년 MM월 dd일 HH:mm:ss', { locale: ko })
}

const getSeverityType = (severity: string) => {
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

const getEventIcon = () => {
  if (isActivityType(props.event)) {
    const iconMap = {
      unusual_location: MapPin,
      unusual_device: DeviceDesktop,
      brute_force: AlertTriangle,
      credential_stuffing: Lock,
      account_takeover: ExclamationCircle,
      privilege_escalation: Shield,
    }
    return iconMap[props.event.activityType] || AlertTriangle
  }

  const iconMap = {
    login: Login,
    logout: Logout,
    suspicious_activity: AlertTriangle,
    password_change: Lock,
    device_change: DeviceDesktop,
    location_change: MapPin,
  }
  return iconMap[props.event.eventType] || Shield
}

const getEventTypeText = () => {
  if (isActivityType(props.event)) {
    return getActivityTypeText(props.event.activityType)
  }

  const typeMap = {
    login: '로그인',
    logout: '로그아웃',
    suspicious_activity: '의심스러운 활동',
    password_change: '패스워드 변경',
    device_change: '디바이스 변경',
    location_change: '위치 변경',
  }
  return typeMap[props.event.eventType] || '보안 이벤트'
}

const getActivityTypeText = (activityType: string) => {
  const typeMap = {
    unusual_location: '비정상적인 위치',
    unusual_device: '비정상적인 디바이스',
    brute_force: '무차별 대입 공격',
    credential_stuffing: '크리덴셜 스터핑',
    account_takeover: '계정 탈취',
    privilege_escalation: '권한 상승',
  }
  return typeMap[activityType] || activityType
}

const getRiskColor = (score: number) => {
  if (score >= 80) return '#e74c3c'
  if (score >= 60) return '#f39c12'
  if (score >= 40) return '#3498db'
  return '#27ae60'
}

const getLocationText = () => {
  const location = getLocation()
  if (!location) return null
  return [location.city, location.country].filter(Boolean).join(', ')
}

const getLocation = () => {
  if ('location' in props.event) {
    return props.event.location
  }
  return null
}

const hasDetailedLocation = () => {
  const location = getLocation()
  return location && (location.country || location.city || location.timezone)
}

const getDefaultDescription = () => {
  if (isActivityType(props.event)) {
    return `${getActivityTypeText(props.event.activityType)} 활동이 감지되었습니다. 이 활동은 보안 위험을 나타낼 수 있으므로 주의가 필요합니다.`
  }
  return `${getEventTypeText()} 이벤트가 발생했습니다.`
}

const getRecommendations = () => {
  const recommendations: string[] = []

  if (isActivityType(props.event)) {
    const activity = props.event

    switch (activity.activityType) {
      case 'unusual_location':
        recommendations.push('새로운 위치에서의 접근이 본인이 맞는지 확인하세요')
        recommendations.push('해당 위치가 안전하지 않다면 세션을 종료하세요')
        break
      case 'unusual_device':
        recommendations.push('새로운 디바이스에서의 접근이 본인이 맞는지 확인하세요')
        recommendations.push('알 수 없는 디바이스라면 비밀번호를 변경하세요')
        break
      case 'brute_force':
        recommendations.push('즉시 비밀번호를 변경하세요')
        recommendations.push('2단계 인증을 활성화하세요')
        recommendations.push('비정상적인 로그인 시도가 계속되면 계정을 임시 잠금하세요')
        break
      case 'credential_stuffing':
        recommendations.push('비밀번호가 다른 서비스에서 유출되었을 가능성이 있습니다')
        recommendations.push('고유한 강력한 비밀번호로 변경하세요')
        recommendations.push('다른 서비스의 비밀번호도 확인하세요')
        break
      case 'account_takeover':
        recommendations.push('즉시 모든 세션을 종료하세요')
        recommendations.push('비밀번호를 변경하세요')
        recommendations.push('최근 계정 활동을 검토하세요')
        recommendations.push('관리자에게 문의하세요')
        break
      case 'privilege_escalation':
        recommendations.push('권한 변경이 승인된 것인지 확인하세요')
        recommendations.push('불법적인 권한 상승이라면 즉시 관리자에게 신고하세요')
        break
    }

    if (activity.riskScore >= 80) {
      recommendations.unshift('⚠️ 위험도가 매우 높습니다. 즉시 조치가 필요합니다.')
    }
  } else {
    const event = props.event

    if (event.severity === 'critical' || event.severity === 'high') {
      recommendations.push('이 이벤트는 심각한 보안 위험을 나타낼 수 있습니다')
      recommendations.push('추가 조사 및 모니터링이 필요합니다')
    }
  }

  return recommendations
}
</script>

<style scoped>
.security-event-details {
  padding: 8px 0;
}

.event-id {
  background: var(--code-color);
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 12px;
  font-family: 'Fira Code', monospace;
}

.event-type {
  display: flex;
  align-items: center;
}

.risk-score {
  display: flex;
  align-items: center;
  gap: 8px;
}

.score-text {
  font-size: 12px;
  font-weight: 500;
  min-width: 40px;
}

.ip-address {
  background: var(--code-color);
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 12px;
  font-family: 'Fira Code', monospace;
}

.user-agent {
  background: var(--code-color);
  padding: 8px;
  border-radius: 4px;
  font-size: 11px;
  font-family: 'Fira Code', monospace;
  word-break: break-all;
  line-height: 1.4;
  display: block;
}

.location-info {
  display: flex;
  align-items: center;
}

.resolution-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 13px;
}

.resolution-text {
  padding: 8px;
  background: var(--hover-color);
  border-radius: 4px;
  margin-top: 4px;
}

.event-description {
  color: var(--text-color-2);
  line-height: 1.5;
}

.metadata-content {
  font-size: 11px;
  background: var(--code-color);
  padding: 12px;
  border-radius: 4px;
  margin: 0;
  overflow-x: auto;
  max-height: 300px;
}

.recommendations-list {
  margin: 0;
  padding-left: 20px;
}

.recommendations-list li {
  margin-bottom: 4px;
  line-height: 1.4;
}
</style>