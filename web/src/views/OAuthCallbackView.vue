<template>
  <div class="oauth-callback-view">
    <div class="callback-container">
      <div class="callback-content">
        <!-- 로딩 상태 -->
        <div v-if="isProcessing" class="callback-loading">
          <div class="loading-spinner">
            <NSpin size="large" />
          </div>
          <h2 class="loading-title">OAuth 로그인 처리 중...</h2>
          <p class="loading-message">잠시만 기다려주세요.</p>
        </div>

        <!-- 성공 상태 -->
        <div v-else-if="isSuccess" class="callback-success">
          <div class="success-icon">
            <NIcon size="64" color="#18a058">
              <CheckCircleIcon />
            </NIcon>
          </div>
          <h2 class="success-title">로그인 성공!</h2>
          <p class="success-message">{{ successMessage }}</p>
        </div>

        <!-- 에러 상태 -->
        <div v-else-if="hasError" class="callback-error">
          <div class="error-icon">
            <NIcon size="64" color="#d03050">
              <CloseCircleIcon />
            </NIcon>
          </div>
          <h2 class="error-title">로그인 실패</h2>
          <p class="error-message">{{ errorMessage }}</p>
          <div class="error-actions">
            <NButton
              type="primary"
              @click="returnToLogin"
            >
              로그인 페이지로 돌아가기
            </NButton>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NButton, NIcon, NSpin, useMessage } from 'naive-ui'
import {
  CheckmarkCircleOutline as CheckCircleIcon,
  CloseCircleOutline as CloseCircleIcon,
} from '@vicons/ionicons5'

import { authApi } from '@/api/services/auth'
import { useUserStore } from '@/stores/user'

const route = useRoute()
const router = useRouter()
const message = useMessage()
const userStore = useUserStore()

// 상태
const isProcessing = ref(true)
const isSuccess = ref(false)
const hasError = ref(false)
const errorMessage = ref('')
const successMessage = ref('')

// OAuth 콜백 처리
const handleOAuthCallback = async () => {
  try {
    const { provider, code, state, error } = route.query

    // 에러 파라미터가 있는 경우
    if (error) {
      throw new Error(`OAuth 인증이 취소되었거나 실패했습니다: ${error}`)
    }

    // 필수 파라미터 검증
    if (!provider || !code || !state) {
      throw new Error('OAuth 콜백 파라미터가 누락되었습니다.')
    }

    if (typeof provider !== 'string' || typeof code !== 'string' || typeof state !== 'string') {
      throw new Error('OAuth 콜백 파라미터가 올바르지 않습니다.')
    }

    // OAuth 로그인 처리
    const loginResponse = await authApi.oAuthLogin({
      provider,
      code,
      state,
    })

    // 사용자 정보 및 인증 정보 저장
    userStore.setUser(loginResponse.user)
    userStore.setAuth({
      token: loginResponse.token,
      refreshToken: loginResponse.refreshToken,
      expiresAt: Date.now() + (loginResponse.expiresIn * 1000),
    })

    isProcessing.value = false
    isSuccess.value = true
    successMessage.value = `${getProviderDisplayName(provider)} 계정으로 성공적으로 로그인되었습니다.`

    // 2초 후 메인 페이지로 리다이렉트
    setTimeout(() => {
      const redirect = sessionStorage.getItem('oauth_redirect') || '/'
      sessionStorage.removeItem('oauth_redirect')
      router.replace(redirect)
    }, 2000)

  } catch (error: any) {
    console.error('OAuth callback error:', error)
    
    isProcessing.value = false
    hasError.value = true
    errorMessage.value = error.message || 'OAuth 로그인 처리 중 오류가 발생했습니다.'
    
    message.error(errorMessage.value)
  }
}

// 제공자 표시명 반환
const getProviderDisplayName = (provider: string): string => {
  const providerNames: Record<string, string> = {
    google: 'Google',
    github: 'GitHub',
  }
  return providerNames[provider] || provider
}

// 로그인 페이지로 돌아가기
const returnToLogin = () => {
  router.replace('/login')
}

// 컴포넌트 마운트 시 콜백 처리
onMounted(() => {
  handleOAuthCallback()
})
</script>

<style lang="scss" scoped>
.oauth-callback-view {
  min-height: 100vh;
  @include flex-center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: $spacing-4;

  @include mobile {
    padding: $spacing-2;
  }
}

.callback-container {
  width: 100%;
  max-width: 480px;
  background: $light-bg-primary;
  border-radius: $border-radius-xl;
  padding: $spacing-8;
  box-shadow: $shadow-xl;
  text-align: center;

  .dark & {
    background: $dark-bg-secondary;
  }

  @include mobile {
    padding: $spacing-6;
    border-radius: $border-radius-lg;
  }
}

.callback-content {
  @include flex-center;
  flex-direction: column;
  gap: $spacing-6;
}

.callback-loading {
  .loading-spinner {
    margin-bottom: $spacing-4;
  }

  .loading-title {
    font-size: $font-size-2xl;
    font-weight: $font-weight-semibold;
    color: $light-text-primary;
    margin: 0 0 $spacing-2 0;

    .dark & {
      color: $dark-text-primary;
    }
  }

  .loading-message {
    font-size: $font-size-base;
    color: $light-text-secondary;
    margin: 0;

    .dark & {
      color: $dark-text-secondary;
    }
  }
}

.callback-success {
  .success-icon {
    margin-bottom: $spacing-4;
  }

  .success-title {
    font-size: $font-size-2xl;
    font-weight: $font-weight-semibold;
    color: $success-color;
    margin: 0 0 $spacing-2 0;
  }

  .success-message {
    font-size: $font-size-base;
    color: $light-text-secondary;
    margin: 0;

    .dark & {
      color: $dark-text-secondary;
    }
  }
}

.callback-error {
  .error-icon {
    margin-bottom: $spacing-4;
  }

  .error-title {
    font-size: $font-size-2xl;
    font-weight: $font-weight-semibold;
    color: $error-color;
    margin: 0 0 $spacing-2 0;
  }

  .error-message {
    font-size: $font-size-base;
    color: $light-text-secondary;
    margin: 0 0 $spacing-6 0;
    line-height: 1.5;

    .dark & {
      color: $dark-text-secondary;
    }
  }

  .error-actions {
    @include flex-center;
    gap: $spacing-3;
  }
}

// 애니메이션
.callback-loading,
.callback-success,
.callback-error {
  animation: fadeInUp 0.6s ease-out;
}

@keyframes fadeInUp {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>