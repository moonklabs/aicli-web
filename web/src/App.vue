<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'
import { RouterView, useRouter } from 'vue-router'
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
import { useTheme } from '@/composables/useTheme'
import { useAriaLive } from '@/composables/useAriaLive'
import ErrorNotification from '@/components/Common/ErrorNotification.vue'
import OfflineIndicator from '@/components/Common/OfflineIndicator.vue'
import PWAInstallPrompt from '@/components/Common/PWAInstallPrompt.vue'
import ApiDebugPanel from '@/components/Debug/ApiDebugPanel.vue'
import SkipLinks from '@/components/ui/accessibility/SkipLinks.vue'
import ThemeToggle from '@/components/ui/accessibility/ThemeToggle.vue'
import AccessibilityChecker from '@/components/ui/accessibility/AccessibilityChecker.vue'

const userStore = useUserStore()
const router = useRouter()

// 테마 관리
const { isDark, initTheme, accessibilitySettings, toggleTheme } = useTheme()
const { announce } = useAriaLive('app-navigation')

// Naive UI 테마 설정
const theme = computed<GlobalTheme | null>(() => {
  return isDark.value ? _darkTheme : null
})

// 라우터 변경 감지 및 접근성 알림
router.afterEach((to, from) => {
  if (accessibilitySettings.value.announcePageChanges && to.name !== from.name) {
    const pageName = (to.meta?.title as string) || (to.name as string) || '새 페이지'
    announce(`페이지가 ${pageName}으로 변경되었습니다`)
  }
})

// 키보드 단축키 처리
const handleKeyboardShortcuts = (event: KeyboardEvent): void => {
  // Cmd/Ctrl + Shift + T: 테마 토글
  if ((event.metaKey || event.ctrlKey) && event.shiftKey && event.key === 'T') {
    event.preventDefault()
    const previousTheme = isDark.value ? '다크 모드' : '라이트 모드'
    toggleTheme()
    
    // 접근성 알림
    if (accessibilitySettings.value.announcePageChanges) {
      const newTheme = isDark.value ? '다크 모드' : '라이트 모드'
      announce(`키보드 단축키로 테마가 ${previousTheme}에서 ${newTheme}(으)로 변경되었습니다`)
    }
  }
}

// 앱 초기화
onMounted(() => {
  // 테마 시스템 초기화
  initTheme()

  // 인증 상태 복원
  userStore.initializeAuth()

  // 키보드 단축키 이벤트 리스너 등록
  document.addEventListener('keydown', handleKeyboardShortcuts)
})

// 정리
onUnmounted(() => {
  // 키보드 단축키 이벤트 리스너 해제
  document.removeEventListener('keydown', handleKeyboardShortcuts)
})

</script>

<template>
  <!-- 접근성 스킵 링크 (페이지 최상단) -->
  <SkipLinks />

  <!-- Naive UI 글로벌 프로바이더 설정 -->
  <NConfigProvider :theme="theme">
    <NLoadingBarProvider>
      <NDialogProvider>
        <NNotificationProvider>
          <NMessageProvider>
            <!-- 메인 앱 레이아웃 -->
            <div id="app" class="app-container">
              <!-- 접근성 및 테마 컨트롤 -->
              <div class="app-controls">
                <ThemeToggle
                  :show-advanced-settings="true"
                  :compact="false"
                  :show-labels="true"
                />
              </div>

              <!-- 메인 콘텐츠 영역 (MainLayout이 전체 레이아웃 관리) -->
              <RouterView />

              <!-- 전역 컴포넌트들 -->
              <ErrorNotification />
              <OfflineIndicator />
              <PWAInstallPrompt />
              <ApiDebugPanel />

              <!-- 접근성 검사 도구 (개발 환경에서만) -->
              <AccessibilityChecker
                :auto-check="false"
                :live-monitoring="true"
              />
            </div>
          </NMessageProvider>
        </NNotificationProvider>
      </NDialogProvider>
    </NLoadingBarProvider>
  </NConfigProvider>
</template>

<style lang="scss">
// 글로벌 스타일은 main.scss에서 처리되므로 여기서는 앱 전용 스타일만
.app-container {
  width: 100%;
  height: 100vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--bg-primary);
  color: var(--text-primary);

  // 테마 전환 애니메이션
  transition: background-color var(--duration-normal) var(--ease-in-out),
              color var(--duration-normal) var(--ease-in-out);
}

.app-controls {
  position: fixed;
  top: var(--spacing-md);
  right: var(--spacing-md);
  z-index: var(--z-banner);

  @media (max-width: 640px) {
    top: var(--spacing-sm);
    right: var(--spacing-sm);
  }
}

// MainLayout이 콘텐츠 레이아웃을 관리하므로 기본 스타일만 제공
.app-container > .main-layout {
  width: 100%;
  height: 100%;
}

// 접근성 향상을 위한 전역 스타일
// 포커스 표시 강화
[data-force-focus-visible="true"] {
  * {
    &:focus {
      outline: 3px solid var(--border-focus) !important;
      outline-offset: 2px !important;
    }
  }
}

// 고대비 모드에서 그림자 제거
[data-accessibility-theme="high-contrast"] {
  * {
    box-shadow: none !important;
    text-shadow: none !important;
  }
}

// 애니메이션 감소 설정
[data-motion-preference="reduce"] {
  .app-container {
    transition: none !important;
  }

  * {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
  }
}

// 인쇄 친화적 스타일
@media print {
  .app-controls,
  .skip-links {
    display: none !important;
  }

  .main-content {
    padding: 0;
    overflow: visible;
  }
}
</style>
