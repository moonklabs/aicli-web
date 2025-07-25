<script setup lang="ts">
import { onMounted } from 'vue'
import { RouterView } from 'vue-router'
import {
  type GlobalTheme,
  NConfigProvider,
  NDialogProvider,
  NLoadingBarProvider,
  NMessageProvider,
  NNotificationProvider,
  darkTheme as _darkTheme,
} from 'naive-ui'
import { useUserStore } from '@/stores/user'
import ErrorNotification from '@/components/Common/ErrorNotification.vue'
import OfflineIndicator from '@/components/Common/OfflineIndicator.vue'
import ApiDebugPanel from '@/components/Debug/ApiDebugPanel.vue'

const userStore = useUserStore()

// 앱 초기화
onMounted(() => {
  // 인증 상태 복원
  userStore.initializeAuth()
})

// 테마 설정 (현재는 라이트 테마 고정, 추후 다크 모드 토글 구현)
const theme: GlobalTheme | null = null // null = 라이트 테마

</script>

<template>
  <!-- Naive UI 글로벌 프로바이더 설정 -->
  <NConfigProvider :theme="theme">
    <NLoadingBarProvider>
      <NDialogProvider>
        <NNotificationProvider>
          <NMessageProvider>
            <!-- 메인 앱 레이아웃 -->
            <div id="app">
              <RouterView />
              
              <!-- 전역 컴포넌트들 -->
              <ErrorNotification />
              <OfflineIndicator />
              <ApiDebugPanel />
            </div>
          </NMessageProvider>
        </NNotificationProvider>
      </NDialogProvider>
    </NLoadingBarProvider>
  </NConfigProvider>
</template>

<style lang="scss">
// 글로벌 스타일은 main.scss에서 처리되므로 여기서는 앱 전용 스타일만
#app {
  width: 100%;
  height: 100vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
</style>
