<template>
  <n-card class="session-card" :class="{ 'current-session': session.isCurrentSession }">
    <!-- 세션 헤더 -->
    <template #header>
      <div class="session-header">
        <div class="device-info">
          <n-icon size="20" class="device-icon">
            <DeviceDesktop v-if="deviceType === 'desktop'" />
            <DeviceMobile v-else-if="deviceType === 'mobile'" />
            <DeviceTablet v-else-if="deviceType === 'tablet'" />
            <DeviceUnknown v-else />
          </n-icon>
          <div class="device-details">
            <h4>{{ session.deviceInfo.browser }} on {{ session.deviceInfo.os }}</h4>
            <span class="device-type">{{ session.deviceInfo.device }}</span>
          </div>
        </div>
        
        <div class="session-status">
          <n-tag
            v-if="session.isCurrentSession"
            type="success"
            size="small"
            round
          >
            현재 세션
          </n-tag>
          <n-tag
            v-else
            :type="sessionStatusType"
            size="small"
            round
          >
            {{ sessionStatusText }}
          </n-tag>
        </div>
      </div>
    </template>

    <!-- 세션 정보 -->
    <div class="session-content">
      <!-- 위치 정보 -->
      <div v-if="session.locationInfo" class="info-row">
        <n-icon size="16" class="info-icon">
          <MapPin />
        </n-icon>
        <span class="info-text">
          {{ formatLocation(session.locationInfo) }}
        </span>
      </div>

      <!-- IP 주소 -->
      <div class="info-row">
        <n-icon size="16" class="info-icon">
          <Network />
        </n-icon>
        <span class="info-text">{{ session.locationInfo?.ip || 'Unknown IP' }}</span>
      </div>

      <!-- 마지막 활동 시간 -->
      <div class="info-row">
        <n-icon size="16" class="info-icon">
          <Clock />
        </n-icon>
        <span class="info-text">
          마지막 활동: {{ formatLastActivity(session.lastActivityAt) }}
        </span>
      </div>

      <!-- 세션 생성 시간 -->
      <div class="info-row">
        <n-icon size="16" class="info-icon">
          <Calendar />
        </n-icon>
        <span class="info-text">
          로그인: {{ formatDate(session.createdAt) }}
        </span>
      </div>

      <!-- 만료 시간 -->
      <div class="info-row">
        <n-icon size="16" class="info-icon">
          <HourglassHigh />
        </n-icon>
        <span class="info-text">
          만료: {{ formatDate(session.expiresAt) }}
        </span>
      </div>

      <!-- 세션 ID (개발/디버깅용) -->
      <div class="info-row session-id">
        <n-icon size="16" class="info-icon">
          <Hash />
        </n-icon>
        <span class="info-text">{{ session.id.substring(0, 8) }}...</span>
      </div>
    </div>

    <!-- 액션 버튼들 -->
    <template #action>
      <div class="session-actions">
        <n-space>
          <!-- 의심스러운 활동 신고 버튼 -->
          <n-button
            v-if="!session.isCurrentSession"
            type="warning"
            ghost
            size="small"
            @click="handleReportSuspicious"
          >
            <template #icon>
              <n-icon><AlertTriangle /></n-icon>
            </template>
            신고
          </n-button>

          <!-- 세션 세부 정보 보기 -->
          <n-button
            type="info"
            ghost
            size="small"
            @click="showSessionDetails = true"
          >
            <template #icon>
              <n-icon><InfoCircle /></n-icon>
            </template>
            상세
          </n-button>

          <!-- 세션 종료 버튼 (현재 세션이 아닌 경우만) -->
          <n-button
            v-if="!session.isCurrentSession"
            type="error"
            ghost
            size="small"
            :loading="terminating"
            @click="handleTerminate"
          >
            <template #icon>
              <n-icon><LogOut /></n-icon>
            </template>
            종료
          </n-button>
        </n-space>
      </div>
    </template>

    <!-- 세션 상세 정보 모달 -->
    <n-modal
      v-model:show="showSessionDetails"
      preset="card"
      title="세션 상세 정보"
      size="medium"
      :bordered="false"
      :segmented="true"
    >
      <div class="session-details">
        <n-descriptions :column="1" bordered>
          <n-descriptions-item label="세션 ID">
            <n-text code>{{ session.id }}</n-text>
          </n-descriptions-item>
          <n-descriptions-item label="사용자 에이전트">
            <n-text>{{ session.deviceInfo.userAgent }}</n-text>
          </n-descriptions-item>
          <n-descriptions-item label="브라우저">
            {{ session.deviceInfo.browser }}
          </n-descriptions-item>
          <n-descriptions-item label="운영체제">
            {{ session.deviceInfo.os }}
          </n-descriptions-item>
          <n-descriptions-item label="디바이스">
            {{ session.deviceInfo.device }}
          </n-descriptions-item>
          <n-descriptions-item v-if="session.locationInfo" label="IP 주소">
            {{ session.locationInfo.ip }}
          </n-descriptions-item>
          <n-descriptions-item v-if="session.locationInfo?.country" label="국가">
            {{ session.locationInfo.country }}
          </n-descriptions-item>
          <n-descriptions-item v-if="session.locationInfo?.city" label="도시">
            {{ session.locationInfo.city }}
          </n-descriptions-item>
          <n-descriptions-item v-if="session.locationInfo?.timezone" label="시간대">
            {{ session.locationInfo.timezone }}
          </n-descriptions-item>
          <n-descriptions-item label="세션 생성">
            {{ formatDateTime(session.createdAt) }}
          </n-descriptions-item>
          <n-descriptions-item label="마지막 활동">
            {{ formatDateTime(session.lastActivityAt) }}
          </n-descriptions-item>
          <n-descriptions-item label="만료 시간">
            {{ formatDateTime(session.expiresAt) }}
          </n-descriptions-item>
          <n-descriptions-item label="상태">
            <n-tag :type="sessionStatusType" size="small">
              {{ sessionStatusText }}
            </n-tag>
          </n-descriptions-item>
        </n-descriptions>
      </div>
    </n-modal>

    <!-- 의심스러운 활동 신고 모달 -->
    <n-modal
      v-model:show="showReportModal"
      preset="card"
      title="의심스러운 활동 신고"
      size="medium"
      :bordered="false"
    >
      <div class="report-form">
        <n-form ref="reportFormRef" :model="reportForm" :rules="reportRules">
          <n-form-item label="신고 사유" path="reason">
            <n-select
              v-model:value="reportForm.reason"
              :options="suspiciousReasons"
              placeholder="신고 사유를 선택하세요"
            />
          </n-form-item>
          <n-form-item label="추가 설명" path="description">
            <n-input
              v-model:value="reportForm.description"
              type="textarea"
              :rows="3"
              placeholder="추가적인 설명을 입력하세요 (선택사항)"
            />
          </n-form-item>
        </n-form>
      </div>
      
      <template #action>
        <n-space justify="end">
          <n-button @click="showReportModal = false">취소</n-button>
          <n-button
            type="primary"
            :loading="reporting"
            @click="submitReport"
          >
            신고하기
          </n-button>
        </n-space>
      </template>
    </n-modal>
  </n-card>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useMessage } from 'naive-ui'
import {
  Desktop,
  PhonePortrait as DeviceMobile,
  TabletPortrait as DeviceTablet,
  HelpCircle as DeviceUnknown,
  LocationSharp as MapPin,
  Globe as Network,
  TimeSharp as Clock,
  CalendarSharp as Calendar,
  Hourglass as HourglassHigh,
  PricetagSharp as Hash,
  WarningSharp as AlertTriangle,
  InformationCircleSharp as InfoCircle,
  LogOutSharp as LogOut
} from '@vicons/ionicons5'

// DeviceDesktop은 Desktop으로 별칭 설정
const DeviceDesktop = Desktop
// date-fns 대신 내장 함수 사용
import type { UserSession } from '@/types/api'

// Props
interface Props {
  session: UserSession
}

const props = defineProps<Props>()

// Emits
const emit = defineEmits<{
  terminate: [sessionId: string]
  reportSuspicious: [sessionId: string, reason: string]
}>()

// 컴포저블
const message = useMessage()

// 반응형 상태
const terminating = ref(false)
const reporting = ref(false)
const showSessionDetails = ref(false)
const showReportModal = ref(false)

// 신고 폼 데이터
const reportForm = ref({
  reason: '',
  description: ''
})

const reportFormRef = ref()

// 신고 사유 옵션
const suspiciousReasons = [
  { label: '본인이 사용하지 않는 디바이스', value: 'unknown_device' },
  { label: '알 수 없는 위치에서의 접근', value: 'unknown_location' },
  { label: '비정상적인 활동 패턴', value: 'abnormal_activity' },
  { label: '해킹 의심', value: 'suspected_hack' },
  { label: '기타', value: 'other' }
]

// 신고 폼 유효성 검사 규칙
const reportRules = {
  reason: {
    required: true,
    message: '신고 사유를 선택해주세요',
    trigger: ['blur', 'change']
  }
}

// 계산된 속성
const deviceType = computed(() => {
  const device = props.session.deviceInfo.device.toLowerCase()
  if (device.includes('mobile') || device.includes('phone')) {
    return 'mobile'
  }
  if (device.includes('tablet') || device.includes('ipad')) {
    return 'tablet'
  }
  return 'desktop'
})

const sessionStatusType = computed(() => {
  if (props.session.isCurrentSession) return 'success'
  
  switch (props.session.status) {
    case 'active':
      return 'info'
    case 'expired':
      return 'warning'
    case 'terminated':
      return 'error'
    default:
      return 'default'
  }
})

const sessionStatusText = computed(() => {
  if (props.session.isCurrentSession) return '현재 세션'
  
  switch (props.session.status) {
    case 'active':
      return '활성'
    case 'expired':
      return '만료됨'
    case 'terminated':
      return '종료됨'
    default:
      return '알 수 없음'
  }
})

// 메서드
const formatLocation = (locationInfo: NonNullable<UserSession['locationInfo']>) => {
  const parts = []
  if (locationInfo.city) parts.push(locationInfo.city)
  if (locationInfo.country) parts.push(locationInfo.country)
  return parts.length > 0 ? parts.join(', ') : locationInfo.ip
}

const formatLastActivity = (dateString: string) => {
  const now = new Date()
  const date = new Date(dateString)
  const diffMs = now.getTime() - date.getTime()
  const diffMinutes = Math.floor(diffMs / (1000 * 60))
  const diffHours = Math.floor(diffMinutes / 60)
  const diffDays = Math.floor(diffHours / 24)

  if (diffMinutes < 1) return '방금 전'
  if (diffMinutes < 60) return `${diffMinutes}분 전`
  if (diffHours < 24) return `${diffHours}시간 전`
  if (diffDays < 7) return `${diffDays}일 전`
  return formatDate(dateString)
}

const formatDate = (dateString: string) => {
  const date = new Date(dateString)
  return date.toLocaleString('ko-KR', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

const formatDateTime = (dateString: string) => {
  const date = new Date(dateString)
  return date.toLocaleString('ko-KR', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

const handleTerminate = () => {
  emit('terminate', props.session.id)
}

const handleReportSuspicious = () => {
  showReportModal.value = true
}

const submitReport = async () => {
  try {
    await reportFormRef.value?.validate()
    
    reporting.value = true
    
    const reason = suspiciousReasons.find(r => r.value === reportForm.value.reason)?.label || reportForm.value.reason
    const fullReason = reportForm.value.description 
      ? `${reason}: ${reportForm.value.description}`
      : reason
    
    emit('reportSuspicious', props.session.id, fullReason)
    
    // 폼 초기화
    reportForm.value = { reason: '', description: '' }
    showReportModal.value = false
    
    message.success('의심스러운 활동을 신고했습니다')
  } catch (error) {
    console.error('신고 실패:', error)
  } finally {
    reporting.value = false
  }
}
</script>

<style scoped lang="scss">
.session-card {
  transition: all 0.3s ease;
  
  &:hover {
    transform: translateY(-2px);
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12);
  }

  &.current-session {
    border: 2px solid var(--primary-color);
    background: linear-gradient(145deg, 
      rgba(var(--primary-color-rgb), 0.02), 
      rgba(var(--primary-color-rgb), 0.05)
    );
  }

  .session-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;

    .device-info {
      display: flex;
      align-items: center;
      gap: 12px;

      .device-icon {
        color: var(--primary-color);
      }

      .device-details {
        h4 {
          margin: 0 0 4px 0;
          font-size: 16px;
          font-weight: 500;
          color: var(--text-color-1);
        }

        .device-type {
          font-size: 14px;
          color: var(--text-color-2);
        }
      }
    }

    .session-status {
      flex-shrink: 0;
    }
  }

  .session-content {
    .info-row {
      display: flex;
      align-items: center;
      gap: 8px;
      margin-bottom: 8px;

      &:last-child {
        margin-bottom: 0;
      }

      .info-icon {
        color: var(--text-color-3);
        flex-shrink: 0;
      }

      .info-text {
        font-size: 14px;
        color: var(--text-color-2);
        line-height: 1.4;
      }

      &.session-id {
        margin-top: 12px;
        padding-top: 8px;
        border-top: 1px solid var(--border-color);

        .info-text {
          font-family: 'Courier New', monospace;
          font-size: 12px;
          color: var(--text-color-3);
        }
      }
    }
  }

  .session-actions {
    margin-top: 16px;
  }
}

.session-details {
  .n-descriptions {
    --n-th-color: var(--card-color);
  }
}

.report-form {
  .n-form-item {
    margin-bottom: 16px;
  }
}

// 반응형 디자인
@media (max-width: 480px) {
  .session-card {
    .session-header {
      flex-direction: column;
      gap: 12px;
      align-items: flex-start;
    }

    .session-actions {
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
</style>