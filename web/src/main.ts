import './styles/main.scss'

import { createApp } from 'vue'
import { createPinia } from 'pinia'

// Naive UI 설정
import {
  // 전역적으로 사용할 컴포넌트들
  NConfigProvider,
  NDialogProvider,
  NLoadingBarProvider,
  NMessageProvider,
  NNotificationProvider,
  // 메시지 API
  createDiscreteApi,
  // 다크 테마
  darkTheme as _darkTheme,
} from 'naive-ui'

import App from './App.vue'
import router from './router'
import { registerPermissionDirectives } from './directives/permission'
import { registerServiceWorker } from './utils/sw-registration'

const app = createApp(App)

// Pinia 설정 (스토어 사용을 위해)
const pinia = createPinia()
app.use(pinia)

// 라우터 설정
app.use(router)

// 권한 디렉티브 등록
registerPermissionDirectives(app)

// 플러그인들을 조건부로 동적 로드 (번들 크기 최적화)
if (import.meta.env.VITE_FEATURE_PERFORMANCE_MONITOR !== 'false') {
  import('./utils/performance').then(({ createPerformancePlugin }) => {
    app.use(createPerformancePlugin())
  })
}

if (import.meta.env.VITE_FEATURE_ERROR_TRACKING !== 'false') {
  import('./utils/error-tracker').then(({ errorTrackingPlugin }) => {
    app.use(errorTrackingPlugin)
  })
}

// 분석 플러그인은 환경 변수가 설정된 경우만 로드
if (import.meta.env.VITE_GA4_MEASUREMENT_ID || import.meta.env.VITE_ANALYTICS_ENDPOINT) {
  import('./utils/analytics').then(({ analyticsPlugin }) => {
    app.use(analyticsPlugin, {
      ga4: import.meta.env.VITE_GA4_MEASUREMENT_ID ? {
        measurementId: import.meta.env.VITE_GA4_MEASUREMENT_ID,
        debugMode: import.meta.env.DEV,
      } : undefined,
      local: true,
      apiEndpoint: import.meta.env.VITE_ANALYTICS_ENDPOINT,
    })
  })
}

// Naive UI 전역 컴포넌트 등록
app.component('NConfigProvider', NConfigProvider)
app.component('NMessageProvider', NMessageProvider)
app.component('NDialogProvider', NDialogProvider)
app.component('NNotificationProvider', NNotificationProvider)
app.component('NLoadingBarProvider', NLoadingBarProvider)

// 전역 API는 필요시에만 설정 (번들 크기 최적화)
if (import.meta.env.VITE_FEATURE_DISCRETE_API !== 'false') {
  const { message, notification, dialog, loadingBar } = createDiscreteApi(
    ['message', 'dialog', 'notification', 'loadingBar'],
  )
  
  // 전역으로 사용할 수 있도록 설정
  app.config.globalProperties.$message = message
  app.config.globalProperties.$notification = notification
  app.config.globalProperties.$dialog = dialog
  app.config.globalProperties.$loadingBar = loadingBar
}

// Service Worker 등록
registerServiceWorker({
  onUpdate: (registration) => {
    console.log('🔄 PWA 업데이트가 준비되었습니다')
    // 업데이트 알림을 위한 이벤트 발송
    window.dispatchEvent(new CustomEvent('pwa-update-available', {
      detail: { registration },
    }))
  },
  onInstalled: (registration) => {
    console.log('✨ PWA가 설치되었습니다')
  },
  onError: (error) => {
    console.error('❌ Service Worker 오류:', error)
  },
})

// 개발 환경에서 터미널 테스트 활성화
if (import.meta.env.DEV) {
  import('./utils/terminal-test').then(({ runDevelopmentTests }) => {
    runDevelopmentTests()
  }).catch(console.error)
}

// 앱 마운트
app.mount('#app')
