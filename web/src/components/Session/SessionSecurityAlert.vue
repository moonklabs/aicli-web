<template>
  <div class="security-alerts">
    <!-- 보안 알림이 있을 때만 표시 -->
    <div v-if="hasAlerts" class="alerts-container">
      <!-- 높은 우선순위 알림 -->
      <n-alert
        v-for="alert in criticalAlerts"
        :key="alert.id"
        type="error"
        closable
        :show-icon="true"
        class="alert-item"
        @close="dismissAlert(alert.id)"
      >
        <template #header>
          <n-space align="center">
            <n-icon size="20">
              <ShieldExclamation />
            </n-icon>
            <span class="alert-title">{{ alert.title }}</span>
            <n-tag type="error" size="small">위험</n-tag>
          </n-space>
        </template>
        
        <div class="alert-content">
          <p>{{ alert.message }}</p>
          <div class="alert-details">
            <n-space size="small">
              <n-text depth="3" style="font-size: 12px;">
                {{ formatTime(alert.timestamp) }}
              </n-text>
              <n-text depth="3" style="font-size: 12px;">
                IP: {{ alert.ipAddress }}
              </n-text>
            </n-space>
          </div>
        </div>

        <template #action>
          <n-space>
            <n-button
              v-if="alert.actionable"
              type="error"
              size="small"
              @click="handleSecurityAction(alert)"
            >
              {{ alert.actionText || '세션 종료' }}
            </n-button>
            <n-button
              type="default"
              size="small"
              ghost
              @click="viewDetails(alert)"
            >
              상세보기
            </n-button>
          </n-space>
        </template>
      </n-alert>

      <!-- 중간 우선순위 알림 -->
      <n-alert
        v-for="alert in warningAlerts"
        :key="alert.id"
        type="warning"
        closable
        :show-icon="true"
        class="alert-item"
        @close="dismissAlert(alert.id)"
      >
        <template #header>
          <n-space align="center">
            <n-icon size="18">
              <Warning />
            </n-icon>
            <span class="alert-title">{{ alert.title }}</span>
            <n-tag type="warning" size="small">주의</n-tag>
          </n-space>
        </template>
        
        <div class="alert-content">
          <p>{{ alert.message }}</p>
          <div class="alert-details">
            <n-space size="small">
              <n-text depth="3" style="font-size: 12px;">
                {{ formatTime(alert.timestamp) }}
              </n-text>
              <n-text depth="3" style="font-size: 12px;">
                IP: {{ alert.ipAddress }}
              </n-text>
            </n-space>
          </div>
        </div>

        <template #action>
          <n-space>
            <n-button
              v-if="alert.actionable"
              type="warning"
              size="small"
              ghost
              @click="handleSecurityAction(alert)"
            >
              {{ alert.actionText || '확인' }}
            </n-button>
          </n-space>
        </template>
      </n-alert>

      <!-- 정보성 알림 -->
      <n-alert
        v-for="alert in infoAlerts"
        :key="alert.id"
        type="info"
        closable
        :show-icon="true"
        class="alert-item"
        @close="dismissAlert(alert.id)"
      >
        <template #header>
          <n-space align="center">
            <n-icon size="18">
              <InfoCircle />
            </n-icon>
            <span class="alert-title">{{ alert.title }}</span>
            <n-tag type="info" size="small">정보</n-tag>
          </n-space>
        </template>
        
        <div class="alert-content">
          <p>{{ alert.message }}</p>
          <div class="alert-details">
            <n-space size="small">
              <n-text depth="3" style="font-size: 12px;">
                {{ formatTime(alert.timestamp) }}
              </n-text>
            </n-space>
          </div>
        </div>
      </n-alert>
    </div>

    <!-- 알림이 없을 때 -->
    <n-empty
      v-else
      description="보안 알림이 없습니다"
      size="small"
      style="margin: 20px 0;"
    />

    <!-- 알림 상세 모달 -->
    <n-modal
      v-model:show="showDetailModal"
      preset="card"
      title="보안 알림 상세정보"
      size="medium"
      :bordered="false"
    >
      <div v-if="selectedAlert" class="alert-detail">
        <n-descriptions :column="1" bordered>
          <n-descriptions-item label="알림 유형">
            <n-tag :type="getAlertTagType(selectedAlert.severity)" size="small">
              {{ getSeverityLabel(selectedAlert.severity) }}
            </n-tag>
          </n-descriptions-item>
          <n-descriptions-item label="제목">
            {{ selectedAlert.title }}
          </n-descriptions-item>
          <n-descriptions-item label="메시지">
            {{ selectedAlert.message }}
          </n-descriptions-item>
          <n-descriptions-item label="발생 시간">
            {{ formatDateTime(selectedAlert.timestamp) }}
          </n-descriptions-item>
          <n-descriptions-item label="IP 주소">
            {{ selectedAlert.ipAddress }}
          </n-descriptions-item>
          <n-descriptions-item v-if="selectedAlert.userAgent" label="User Agent">
            <n-text style="word-break: break-all; font-size: 12px;">
              {{ selectedAlert.userAgent }}
            </n-text>
          </n-descriptions-item>
          <n-descriptions-item v-if="selectedAlert.sessionId" label="세션 ID">
            <n-text code>{{ selectedAlert.sessionId }}</n-text>
          </n-descriptions-item>
          <n-descriptions-item v-if="selectedAlert.metadata" label="추가 정보">
            <n-code :code="JSON.stringify(selectedAlert.metadata, null, 2)" language="json" />
          </n-descriptions-item>
        </n-descriptions>
      </div>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useMessage } from 'naive-ui'
import {
  ShieldCheckmarkSharp as ShieldExclamation,
  WarningSharp as Warning,
  InformationCircleSharp as InfoCircle
} from '@vicons/ionicons5'

// 보안 알림 인터페이스
interface SecurityAlert {
  id: string
  title: string
  message: string
  severity: 'critical' | 'warning' | 'info'
  timestamp: string
  ipAddress: string
  userAgent?: string
  sessionId?: string
  actionable?: boolean
  actionText?: string
  metadata?: Record<string, any>
}

// Props
interface Props {
  alerts?: SecurityAlert[]
}

const props = withDefaults(defineProps<Props>(), {
  alerts: () => []
})

// Emits
const emit = defineEmits<{
  dismiss: [alertId: string]
  action: [alert: SecurityAlert]
}>()

// 컴포저블
const message = useMessage()

// 반응형 상태
const showDetailModal = ref(false)
const selectedAlert = ref<SecurityAlert | null>(null)

// 계산된 속성
const hasAlerts = computed(() => props.alerts.length > 0)

const criticalAlerts = computed(() => 
  props.alerts.filter(alert => alert.severity === 'critical')
)

const warningAlerts = computed(() => 
  props.alerts.filter(alert => alert.severity === 'warning')
)

const infoAlerts = computed(() => 
  props.alerts.filter(alert => alert.severity === 'info')
)

// 메서드
const formatTime = (timestamp: string) => {
  const date = new Date(timestamp)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMinutes = Math.floor(diffMs / (1000 * 60))
  
  if (diffMinutes < 1) return '방금 전'
  if (diffMinutes < 60) return `${diffMinutes}분 전`
  if (diffMinutes < 1440) return `${Math.floor(diffMinutes / 60)}시간 전`
  
  return date.toLocaleString('ko-KR', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}

const formatDateTime = (timestamp: string) => {
  return new Date(timestamp).toLocaleString('ko-KR', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

const getSeverityLabel = (severity: string) => {
  switch (severity) {
    case 'critical': return '위험'
    case 'warning': return '주의'
    case 'info': return '정보'
    default: return severity
  }
}

const getAlertTagType = (severity: string): 'error' | 'warning' | 'info' | 'success' => {
  switch (severity) {
    case 'critical': return 'error'
    case 'warning': return 'warning'
    case 'info': return 'info'
    default: return 'info'
  }
}

const dismissAlert = (alertId: string) => {
  emit('dismiss', alertId)
}

const handleSecurityAction = (alert: SecurityAlert) => {
  emit('action', alert)
}

const viewDetails = (alert: SecurityAlert) => {
  selectedAlert.value = alert
  showDetailModal.value = true
}
</script>

<style scoped lang="scss">
.security-alerts {
  .alerts-container {
    .alert-item {
      margin-bottom: 16px;

      &:last-child {
        margin-bottom: 0;
      }

      .alert-title {
        font-weight: 500;
        font-size: 14px;
      }

      .alert-content {
        p {
          margin: 0 0 8px 0;
          line-height: 1.5;
        }

        .alert-details {
          margin-top: 8px;
        }
      }
    }
  }

  .alert-detail {
    .n-descriptions {
      --n-th-color: var(--card-color);
    }
  }
}

// 반응형 디자인
@media (max-width: 480px) {
  .security-alerts {
    .alerts-container {
      .alert-item {
        :deep(.n-alert-body) {
          .n-alert__content {
            .alert-content {
              .alert-details {
                .n-space {
                  flex-direction: column;
                  align-items: flex-start;
                }
              }
            }
          }

          .n-alert__action {
            .n-space {
              flex-wrap: wrap;
              
              .n-space-item {
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