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

const app = createApp(App)

// Pinia 설정 (스토어 사용을 위해)
const pinia = createPinia()
app.use(pinia)

// 라우터 설정
app.use(router)

// Naive UI 전역 컴포넌트 등록
app.component('NConfigProvider', NConfigProvider)
app.component('NMessageProvider', NMessageProvider)
app.component('NDialogProvider', NDialogProvider)
app.component('NNotificationProvider', NNotificationProvider)
app.component('NLoadingBarProvider', NLoadingBarProvider)

// 전역 API 설정 (선택적)
const { message, notification, dialog, loadingBar } = createDiscreteApi(
  ['message', 'dialog', 'notification', 'loadingBar'],
)

// 전역으로 사용할 수 있도록 설정
app.config.globalProperties.$message = message
app.config.globalProperties.$notification = notification
app.config.globalProperties.$dialog = dialog
app.config.globalProperties.$loadingBar = loadingBar

// 개발 환경에서 터미널 테스트 활성화
if (import.meta.env.DEV) {
  import('./utils/terminal-test').then(({ runDevelopmentTests }) => {
    runDevelopmentTests()
  }).catch(console.error)
}

// 앱 마운트
app.mount('#app')
