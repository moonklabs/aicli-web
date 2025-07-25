<template>
  <div class="login-detail-modal">
    <NDescriptions :column="2" bordered>
      <NDescriptionsItem label="로그인 시간">
        {{ formatDateTime(login.createdAt) }}
      </NDescriptionsItem>

      <NDescriptionsItem label="상태">
        <NTag :type="getStatusType(login.status)" size="small">
          {{ getStatusText(login.status) }}
        </NTag>
      </NDescriptionsItem>

      <NDescriptionsItem label="위험도 점수">
        <div class="risk-score">
          <NProgress
            type="line"
            :percentage="login.riskScore"
            :color="getRiskColor(login.riskScore)"
            :show-indicator="false"
            height="8"
          />
          <span class="score-text">{{ login.riskScore }}/100</span>
        </div>
      </NDescriptionsItem>

      <NDescriptionsItem label="의심스러운 활동">
        <NTag :type="login.isSuspicious ? 'error' : 'success'" size="small">
          {{ login.isSuspicious ? '예' : '아니오' }}
        </NTag>
      </NDescriptionsItem>

      <NDescriptionsItem label="IP 주소">
        <code class="ip-address">{{ login.ipAddress }}</code>
      </NDescriptionsItem>

      <NDescriptionsItem label="위치">
        <div v-if="login.location" class="location-info">
          <NIcon :component="MapPin" size="14" style="margin-right: 4px" />
          {{ [login.location.city, login.location.country].filter(Boolean).join(', ') }}
          <span v-if="login.location.timezone" class="timezone">
            ({{ login.location.timezone }})
          </span>
        </div>
        <span v-else class="no-data">위치 정보 없음</span>
      </NDescriptionsItem>

      <NDescriptionsItem label="로그인 방법">
        <NTag :type="getMethodType(login.loginMethod)" size="small">
          {{ getMethodText(login.loginMethod) }}
        </NTag>
        <span v-if="login.provider" class="provider-info">
          via {{ login.provider }}
        </span>
      </NDescriptionsItem>

      <NDescriptionsItem label="세션 ID">
        <code class="session-id">{{ login.sessionId }}</code>
      </NDescriptionsItem>
    </NDescriptions>

    <!-- 디바이스 정보 섹션 -->
    <NDivider>디바이스 정보</NDivider>
    <NDescriptions :column="2" bordered>
      <NDescriptionsItem label="브라우저">
        {{ login.deviceInfo.browser }}
      </NDescriptionsItem>

      <NDescriptionsItem label="운영체제">
        {{ login.deviceInfo.os }}
      </NDescriptionsItem>

      <NDescriptionsItem label="디바이스 유형">
        {{ login.deviceInfo.device }}
      </NDescriptionsItem>

      <NDescriptionsItem label="User Agent" span="2">
        <code class="user-agent">{{ login.userAgent }}</code>
      </NDescriptionsItem>
    </NDescriptions>

    <!-- 실패 사유 (실패한 경우에만) -->
    <div v-if="login.status === 'failure' && login.failureReason" class="failure-section">
      <NDivider>실패 사유</NDivider>
      <NAlert type="error" :show-icon="false">
        {{ login.failureReason }}
      </NAlert>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import {
  NAlert,
  NDescriptions,
  NDescriptionsItem,
  NDivider,
  NIcon,
  NProgress,
  NTag,
} from 'naive-ui'
import { MapPin } from '@vicons/tabler'
import { format } from 'date-fns'
import { ko } from 'date-fns/locale'
import type { LoginHistory } from '@/types/api'

interface Props {
  login: LoginHistory
}

const props = defineProps<Props>()

// 메소드
const formatDateTime = (dateString: string) => {
  return format(new Date(dateString), 'yyyy년 MM월 dd일 HH:mm:ss', { locale: ko })
}

const getStatusType = (status: string) => {
  const statusMap = {
    success: 'success',
    failure: 'error',
    blocked: 'warning',
  }
  return statusMap[status] || 'default'
}

const getStatusText = (status: string) => {
  const statusMap = {
    success: '성공',
    failure: '실패',
    blocked: '차단됨',
  }
  return statusMap[status] || status
}

const getMethodType = (method: string) => {
  const methodMap = {
    password: 'default',
    oauth: 'info',
    sso: 'warning',
    token: 'success',
  }
  return methodMap[method] || 'default'
}

const getMethodText = (method: string) => {
  const methodMap = {
    password: '비밀번호',
    oauth: 'OAuth',
    sso: 'SSO',
    token: '토큰',
  }
  return methodMap[method] || method
}

const getRiskColor = (score: number) => {
  if (score >= 80) return '#e74c3c'
  if (score >= 60) return '#f39c12'
  if (score >= 40) return '#3498db'
  return '#27ae60'
}
</script>

<style scoped>
.login-detail-modal {
  padding: 8px 0;
}

.risk-score {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}

.score-text {
  font-size: 12px;
  font-weight: 500;
  min-width: 40px;
}

.ip-address,
.session-id {
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
  gap: 4px;
}

.timezone {
  color: var(--text-color-3);
  font-size: 12px;
}

.provider-info {
  margin-left: 8px;
  color: var(--text-color-3);
  font-size: 12px;
}

.no-data {
  color: var(--text-color-3);
  font-style: italic;
}

.failure-section {
  margin-top: 16px;
}
</style>