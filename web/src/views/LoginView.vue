<template>
  <div class="login-view">
    <div class="login-container">
      <!-- 로고 및 제목 -->
      <div class="login-header">
        <div class="logo-container">
          <NIcon size="48" color="#3182ce">
            <CodeIcon />
          </NIcon>
        </div>
        <h1 class="login-title">AICLI Web</h1>
        <p class="login-subtitle">AI Code Manager</p>
      </div>

      <!-- 로그인 폼 -->
      <div class="login-form-container">
        <NForm
          ref="formRef"
          :model="formData"
          :rules="formRules"
          size="large"
          @submit.prevent="handleLogin"
        >
          <!-- 사용자명/이메일 -->
          <NFormItem path="username">
            <NInput
              v-model:value="formData.username"
              placeholder="사용자명 또는 이메일"
              :disabled="isLoading"
              @keyup.enter="handleLogin"
            >
              <template #prefix>
                <NIcon>
                  <PersonIcon />
                </NIcon>
              </template>
            </NInput>
          </NFormItem>

          <!-- 비밀번호 -->
          <NFormItem path="password">
            <NInput
              v-model:value="formData.password"
              type="password"
              placeholder="비밀번호"
              :disabled="isLoading"
              show-password-on="mousedown"
              @keyup.enter="handleLogin"
            >
              <template #prefix>
                <NIcon>
                  <LockClosedIcon />
                </NIcon>
              </template>
            </NInput>
          </NFormItem>

          <!-- 추가 옵션 -->
          <div class="login-options">
            <NCheckbox v-model:checked="formData.rememberMe" :disabled="isLoading">
              로그인 상태 유지
            </NCheckbox>

            <NButton
              text
              type="primary"
              size="small"
              :disabled="isLoading"
              @click="showForgotPassword"
            >
              비밀번호를 잊으셨나요?
            </NButton>
          </div>

          <!-- 로그인 버튼 -->
          <NFormItem>
            <NButton
              type="primary"
              size="large"
              block
              :loading="isLoading"
              :disabled="!isFormValid"
              @click="handleLogin"
            >
              <template #icon>
                <NIcon>
                  <LogInIcon />
                </NIcon>
              </template>
              로그인
            </NButton>
          </NFormItem>
        </NForm>

        <!-- 구분선 -->
        <NDivider>또는</NDivider>

        <!-- OAuth 로그인 -->
        <div class="oauth-buttons">
          <NButton
            secondary
            size="large"
            block
            :loading="oauthLoading === 'google'"
            :disabled="isLoading || oauthLoading !== null"
            @click="handleOAuthLogin('google')"
          >
            <template #icon>
              <NIcon>
                <LogoGoogle />
              </NIcon>
            </template>
            Google로 로그인
          </NButton>
          
          <NButton
            secondary
            size="large"
            block
            :loading="oauthLoading === 'github'"
            :disabled="isLoading || oauthLoading !== null"
            @click="handleOAuthLogin('github')"
          >
            <template #icon>
              <NIcon>
                <LogoGithub />
              </NIcon>
            </template>
            GitHub으로 로그인
          </NButton>
        </div>

        <!-- 회원가입 링크 -->
        <div class="signup-link">
          <span>계정이 없으신가요?</span>
          <NButton
            text
            type="primary"
            :disabled="isLoading"
            @click="showSignup"
          >
            회원가입
          </NButton>
        </div>
      </div>

      <!-- 데모 계정 정보 -->
      <div class="demo-info">
        <NCard size="small" embedded>
          <template #header>
            <NIcon size="16">
              <InformationCircleIcon />
            </NIcon>
            데모 계정
          </template>
          <div class="demo-credentials">
            <p><strong>사용자명:</strong> admin</p>
            <p><strong>비밀번호:</strong> admin123</p>
          </div>
          <template #action>
            <NButton
              size="small"
              quaternary
              type="primary"
              @click="fillDemoCredentials"
            >
              데모 계정으로 채우기
            </NButton>
          </template>
        </NCard>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  type FormInst,
  type FormRules,
  NButton,
  NCard,
  NCheckbox,
  NDivider,
  NForm,
  NFormItem,
  NIcon,
  NInput,
  useMessage,
} from 'naive-ui'
import {
  CodeOutline as CodeIcon,
  InformationCircleOutline as InformationCircleIcon,
  LockClosedOutline as LockClosedIcon,
  LogInOutline as LogInIcon,
  LogoGithub,
  LogoGoogle,
  PersonOutline as PersonIcon,
} from '@vicons/ionicons5'

import { useUserStore } from '@/stores/user'
import { authApi } from '@/api/services/auth'

const router = useRouter()
const message = useMessage()
const userStore = useUserStore()

// 폼 참조
const formRef = ref<FormInst>()

// 상태
const isLoading = ref(false)
const oauthLoading = ref<string | null>(null)

// 폼 데이터
const formData = ref({
  username: '',
  password: '',
  rememberMe: false,
})

// 폼 유효성 검사 규칙
const formRules: FormRules = {
  username: [
    {
      required: true,
      message: '사용자명 또는 이메일을 입력해주세요',
      trigger: ['input', 'blur'],
    },
  ],
  password: [
    {
      required: true,
      message: '비밀번호를 입력해주세요',
      trigger: ['input', 'blur'],
    },
    {
      min: 6,
      message: '비밀번호는 최소 6자 이상이어야 합니다',
      trigger: 'blur',
    },
  ],
}

// 계산된 속성
const isFormValid = computed(() => {
  return formData.value.username.length > 0 && formData.value.password.length >= 6
})

// 메서드
const handleLogin = async () => {
  if (!formRef.value) return

  try {
    await formRef.value.validate()
    isLoading.value = true

    // 실제 API 호출은 추후 구현
    // const result = await authApi.login({
    //   username: formData.value.username,
    //   password: formData.value.password
    // })

    // 임시 로그인 시뮬레이션
    await new Promise(resolve => setTimeout(resolve, 1500))

    if (formData.value.username === 'admin' && formData.value.password === 'admin123') {
      // 임시 사용자 데이터
      const userData = {
        id: '1',
        username: 'admin',
        email: 'admin@aicli.dev',
        displayName: '관리자',
      }

      const authData = {
        token: 'demo-jwt-token',
        refreshToken: 'demo-refresh-token',
        expiresAt: Date.now() + 24 * 60 * 60 * 1000, // 24시간
      }

      userStore.setUser(userData)
      userStore.setAuth(authData)

      message.success('로그인되었습니다!')

      // 리다이렉트
      const redirect = router.currentRoute.value.query.redirect as string
      router.push(redirect || '/')
    } else {
      throw new Error('Invalid credentials')
    }
  } catch (error: any) {
    if (error?.message === 'Invalid credentials') {
      message.error('사용자명 또는 비밀번호가 올바르지 않습니다')
    } else {
      message.error('로그인 중 오류가 발생했습니다')
    }
  } finally {
    isLoading.value = false
  }
}

const fillDemoCredentials = () => {
  formData.value.username = 'admin'
  formData.value.password = 'admin123'
}

const showForgotPassword = () => {
  message.info('비밀번호 재설정 기능은 준비 중입니다')
}

const showSignup = () => {
  message.info('회원가입 기능은 준비 중입니다')
}

// OAuth 로그인 처리
const handleOAuthLogin = async (provider: string) => {
  try {
    oauthLoading.value = provider
    
    // 현재 경로를 세션에 저장 (리다이렉트용)
    const redirect = router.currentRoute.value.query.redirect as string
    if (redirect) {
      sessionStorage.setItem('oauth_redirect', redirect)
    }
    
    // OAuth 인증 URL 생성 및 요청
    const state = generateOAuthState()
    const response = await authApi.getOAuthAuthUrl({
      provider,
      state,
    })
    
    // 팝업으로 OAuth 인증 페이지 열기
    const popup = window.open(
      response.authUrl,
      `oauth-${provider}`,
      'width=500,height=600,scrollbars=yes,resizable=yes'
    )
    
    if (!popup) {
      throw new Error('팝업이 차단되었습니다. 팝업 차단을 해제하고 다시 시도해주세요.')
    }
    
    // 팝업 상태 모니터링
    const checkClosed = setInterval(() => {
      if (popup.closed) {
        clearInterval(checkClosed)
        oauthLoading.value = null
        
        // 팝업이 정상적으로 닫힌 경우 (OAuth 완료 또는 취소)
        // 별도 처리 없이 로딩 상태만 해제
      }
    }, 1000)
    
  } catch (error: any) {
    console.error(`OAuth ${provider} login error:`, error)
    message.error(error.message || `${getProviderDisplayName(provider)} 로그인 중 오류가 발생했습니다`)
    oauthLoading.value = null
  }
}

// OAuth state 생성 (CSRF 보호)
const generateOAuthState = (): string => {
  const array = new Uint8Array(32)
  crypto.getRandomValues(array)
  return Array.from(array, byte => byte.toString(16).padStart(2, '0')).join('')
}

// 제공자 표시명 반환
const getProviderDisplayName = (provider: string): string => {
  const providerNames: Record<string, string> = {
    google: 'Google',
    github: 'GitHub',
  }
  return providerNames[provider] || provider
}
</script>

<style lang="scss" scoped>
.login-view {
  min-height: 100vh;
  @include flex-center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: $spacing-4;

  @include mobile {
    padding: $spacing-2;
  }
}

.login-container {
  width: 100%;
  max-width: 400px;
  background: $light-bg-primary;
  border-radius: $border-radius-xl;
  padding: $spacing-8;
  box-shadow: $shadow-xl;

  .dark & {
    background: $dark-bg-secondary;
  }

  @include mobile {
    padding: $spacing-6;
    border-radius: $border-radius-lg;
  }
}

.login-header {
  text-align: center;
  margin-bottom: $spacing-8;

  .logo-container {
    margin-bottom: $spacing-4;
  }

  .login-title {
    font-size: $font-size-3xl;
    font-weight: $font-weight-bold;
    color: $light-text-primary;
    margin: 0 0 $spacing-2 0;

    .dark & {
      color: $dark-text-primary;
    }
  }

  .login-subtitle {
    font-size: $font-size-base;
    color: $light-text-secondary;
    margin: 0;

    .dark & {
      color: $dark-text-secondary;
    }
  }
}

.login-form-container {
  margin-bottom: $spacing-6;
}

.login-options {
  @include flex-between;
  margin: $spacing-4 0;
  font-size: $font-size-sm;

  @include mobile {
    flex-direction: column;
    align-items: stretch;
    gap: $spacing-3;
  }
}

.oauth-buttons {
  margin-bottom: $spacing-6;
  display: flex;
  flex-direction: column;
  gap: $spacing-3;
  
  .n-button {
    transition: all 0.2s ease;
    
    &:not(:disabled):hover {
      transform: translateY(-1px);
      box-shadow: $shadow-md;
    }
  }
}

.signup-link {
  text-align: center;
  font-size: $font-size-sm;
  color: $light-text-secondary;

  .dark & {
    color: $dark-text-secondary;
  }

  span {
    margin-right: $spacing-2;
  }
}

.demo-info {
  margin-top: $spacing-6;
  padding-top: $spacing-6;
  border-top: 1px solid map-get($gray-colors, 200);

  .dark & {
    border-top-color: $dark-bg-tertiary;
  }

  .demo-credentials {
    font-size: $font-size-sm;

    p {
      margin: $spacing-1 0;

      strong {
        color: $light-text-primary;

        .dark & {
          color: $dark-text-primary;
        }
      }
    }
  }
}

// 폼 아이템 스타일 조정
:deep(.n-form-item .n-form-item-blank) {
  margin-bottom: $spacing-4;
}

:deep(.n-form-item:last-child .n-form-item-blank) {
  margin-bottom: 0;
}

// 카드 헤더 아이콘 정렬
:deep(.n-card-header) {
  display: flex;
  align-items: center;
  gap: $spacing-2;
}
</style>