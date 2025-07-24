<template>
  <div class="forbidden-view">
    <n-card class="forbidden-card">
      <div class="forbidden-content">
        <div class="forbidden-icon">
          <Icon name="shield-alert" size="64" />
        </div>
        
        <h1 class="forbidden-title">접근 권한이 없습니다</h1>
        
        <p class="forbidden-message">
          {{ errorMessage }}
        </p>
        
        <div class="forbidden-details" v-if="showDetails">
          <n-divider />
          <div class="details-section">
            <h3>접근 시도 정보</h3>
            <div class="info-grid">
              <div class="info-item">
                <strong>요청 경로:</strong>
                <span>{{ attemptedPath }}</span>
              </div>
              <div class="info-item">
                <strong>시간:</strong>
                <span>{{ formatTime(currentTime) }}</span>
              </div>
              <div class="info-item" v-if="userInfo">
                <strong>사용자:</strong>
                <span>{{ userInfo.username }} ({{ userInfo.email }})</span>
              </div>
              <div class="info-item" v-if="userRoles && userRoles.length > 0">
                <strong>현재 역할:</strong>
                <span>{{ userRoles.join(', ') }}</span>
              </div>
            </div>
          </div>
          
          <div class="debug-section" v-if="isDev">
            <h3>디버그 정보</h3>
            <pre class="debug-info">{{ debugInfo }}</pre>
          </div>
        </div>
        
        <div class="forbidden-actions">
          <n-space>
            <n-button 
              type="primary" 
              @click="goBack"
              :disabled="!canGoBack"
            >
              <template #icon>
                <Icon name="arrow-left" />
              </template>
              이전 페이지로
            </n-button>
            
            <n-button 
              type="default" 
              @click="goHome"
            >
              <template #icon>
                <Icon name="home" />
              </template>
              홈으로 가기
            </n-button>
            
            <n-button 
              type="default" 
              @click="toggleDetails"
            >
              <template #icon>
                <Icon :name="showDetails ? 'eye-off' : 'eye'" />
              </template>
              {{ showDetails ? '세부정보 숨기기' : '세부정보 보기' }}
            </n-button>
            
            <n-button 
              type="warning" 
              @click="requestAccess"
              v-if="!isAdmin"
            >
              <template #icon>
                <Icon name="mail" />
              </template>
              권한 요청하기
            </n-button>
          </n-space>
        </div>
        
        <div class="forbidden-help" v-if="!isAdmin">
          <n-alert type="info" title="권한이 필요하신가요?">
            이 페이지에 접근하려면 특별한 권한이 필요합니다. 
            관리자에게 권한 요청을 보내거나 시스템 관리자에게 문의하세요.
          </n-alert>
        </div>
      </div>
    </n-card>
    
    <!-- 권한 요청 모달 -->
    <n-modal 
      v-model:show="showRequestModal"
      preset="dialog"
      title="권한 요청"
      positive-text="요청 보내기"
      negative-text="취소"
      @positive-click="submitAccessRequest"
    >
      <n-form :model="accessRequest" ref="requestFormRef">
        <n-form-item label="요청 사유" path="reason">
          <n-input
            v-model:value="accessRequest.reason"
            type="textarea"
            placeholder="권한이 필요한 이유를 입력해주세요..."
            :rows="4"
          />
        </n-form-item>
        
        <n-form-item label="긴급도" path="urgency">
          <n-select
            v-model:value="accessRequest.urgency"
            :options="urgencyOptions"
          />
        </n-form-item>
      </n-form>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { PermissionUtils } from '@/utils/permission'
import { 
  NCard, 
  NButton, 
  NSpace, 
  NDivider, 
  NAlert, 
  NModal, 
  NForm, 
  NFormItem, 
  NInput, 
  NSelect,
  useMessage 
} from 'naive-ui'
import Icon from '@/components/common/Icon.vue'

// 컴포저블
const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const message = useMessage()

// 반응형 상태
const showDetails = ref(false)
const showRequestModal = ref(false)
const currentTime = ref(new Date())
const accessRequest = ref({
  reason: '',
  urgency: 'normal'
})

// 계산된 속성
const isDev = computed(() => import.meta.env.DEV)

const attemptedPath = computed(() => {
  return (route.query.from as string) || route.path
})

const errorMessage = computed(() => {
  const customReason = route.query.reason as string
  if (customReason) {
    return customReason
  }
  
  // 기본 메시지
  return '요청하신 페이지에 접근할 권한이 없습니다. 필요한 권한을 확인하고 관리자에게 문의하세요.'
})

const userInfo = computed(() => userStore.currentUser)

const userRoles = computed(() => {
  return userInfo.value?.roles || []
})

const isAdmin = computed(() => PermissionUtils.isAdmin())

const canGoBack = computed(() => {
  return window.history.length > 1 && route.query.from
})

const debugInfo = computed(() => {
  if (!isDev.value) return null
  
  return JSON.stringify({
    route: {
      path: route.path,
      params: route.params,
      query: route.query,
      meta: route.meta
    },
    user: {
      id: userInfo.value?.id,
      roles: userRoles.value,
      permissions: PermissionUtils.permissions
    },
    timestamp: currentTime.value.toISOString()
  }, null, 2)
})

const urgencyOptions = [
  { label: '일반', value: 'normal' },
  { label: '높음', value: 'high' },
  { label: '긴급', value: 'urgent' }
]

// 메서드
const toggleDetails = () => {
  showDetails.value = !showDetails.value
}

const goBack = () => {
  if (canGoBack.value) {
    router.back()
  } else {
    goHome()
  }
}

const goHome = () => {
  router.push({ name: 'dashboard' })
}

const requestAccess = () => {
  showRequestModal.value = true
}

const submitAccessRequest = async () => {
  try {
    // TODO: 실제 API 호출로 권한 요청 제출
    console.log('Access request submitted:', {
      path: attemptedPath.value,
      reason: accessRequest.reason,
      urgency: accessRequest.urgency,
      user: userInfo.value?.id,
      timestamp: new Date().toISOString()
    })
    
    message.success('권한 요청이 관리자에게 전송되었습니다.')
    showRequestModal.value = false
    
    // 폼 초기화
    accessRequest.value = {
      reason: '',
      urgency: 'normal'
    }
  } catch (error) {
    console.error('Failed to submit access request:', error)
    message.error('권한 요청 전송에 실패했습니다.')
  }
}

const formatTime = (date: Date) => {
  return date.toLocaleString('ko-KR', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

// 라이프사이클
onMounted(() => {
  // 페이지 접근 로그 기록 (개발 모드에서만)
  if (isDev.value) {
    console.warn('Forbidden access attempt:', {
      path: attemptedPath.value,
      user: userInfo.value?.username,
      reason: errorMessage.value,
      timestamp: currentTime.value.toISOString()
    })
  }
})
</script>

<style scoped lang="scss">
.forbidden-view {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.forbidden-card {
  max-width: 600px;
  width: 100%;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.1);
}

.forbidden-content {
  text-align: center;
  padding: 40px 24px;
}

.forbidden-icon {
  margin-bottom: 24px;
  color: #f56565;
}

.forbidden-title {
  font-size: 2rem;
  font-weight: 600;
  color: #2d3748;
  margin-bottom: 16px;
}

.forbidden-message {
  font-size: 1.1rem;
  color: #4a5568;
  line-height: 1.6;
  margin-bottom: 32px;
}

.forbidden-details {
  text-align: left;
  margin-bottom: 32px;
}

.details-section h3 {
  margin-bottom: 16px;
  color: #2d3748;
  font-weight: 600;
}

.info-grid {
  display: grid;
  gap: 12px;
}

.info-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 0;
  border-bottom: 1px solid #e2e8f0;
  
  strong {
    color: #4a5568;
    min-width: 100px;
  }
  
  span {
    color: #2d3748;
    text-align: right;
    word-break: break-all;
  }
}

.debug-section {
  margin-top: 24px;
  
  h3 {
    color: #e53e3e;
    margin-bottom: 12px;
  }
}

.debug-info {
  background: #f7fafc;
  padding: 16px;
  border-radius: 8px;
  font-size: 0.875rem;
  color: #2d3748;
  border: 1px solid #e2e8f0;
  overflow-x: auto;
}

.forbidden-actions {
  margin-bottom: 24px;
}

.forbidden-help {
  max-width: 500px;
  margin: 0 auto;
}

// 다크 모드 지원
.dark {
  .forbidden-title {
    color: #f7fafc;
  }
  
  .forbidden-message {
    color: #cbd5e0;
  }
  
  .details-section h3 {
    color: #f7fafc;
  }
  
  .info-item {
    border-bottom-color: #4a5568;
    
    strong {
      color: #cbd5e0;
    }
    
    span {
      color: #f7fafc;
    }
  }
  
  .debug-info {
    background: #2d3748;
    color: #f7fafc;
    border-color: #4a5568;
  }
}

// 반응형 디자인
@media (max-width: 640px) {
  .forbidden-content {
    padding: 24px 16px;
  }
  
  .forbidden-title {
    font-size: 1.5rem;
  }
  
  .forbidden-message {
    font-size: 1rem;
  }
  
  .info-item {
    flex-direction: column;
    align-items: flex-start;
    gap: 4px;
    
    span {
      text-align: left;
    }
  }
}
</style>