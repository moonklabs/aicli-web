<template>
  <Teleport to="body">
    <!-- PWA 설치 배너 (상단) -->
    <Transition name="install-banner">
      <div
        v-if="showBanner && shouldShowPrompt"
        class="fixed top-0 left-0 right-0 z-30 bg-gradient-to-r from-blue-500 to-purple-600 text-white py-3 px-4 shadow-lg"
      >
        <div class="flex items-center justify-between max-w-6xl mx-auto">
          <div class="flex items-center gap-3">
            <div class="text-2xl">📱</div>
            <div>
              <div class="font-medium text-sm">앱으로 설치하기</div>
              <div class="text-xs opacity-90">더 빠르고 편리한 접근을 위해 홈 화면에 추가하세요</div>
            </div>
          </div>
          
          <div class="flex items-center gap-2">
            <button
              @click="handleInstall"
              :disabled="isInstalling"
              class="bg-white text-blue-600 px-4 py-2 rounded-md text-sm font-medium hover:bg-gray-100 transition-colors disabled:opacity-50"
            >
              <span v-if="isInstalling">설치 중...</span>
              <span v-else>설치</span>
            </button>
            <button
              @click="dismissBanner"
              class="text-white hover:text-gray-200 p-1 transition-colors"
              aria-label="배너 닫기"
            >
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
              </svg>
            </button>
          </div>
        </div>
      </div>
    </Transition>

    <!-- PWA 설치 모달 -->
    <Transition name="install-modal">
      <div
        v-if="showModal"
        class="fixed inset-0 z-50 bg-black bg-opacity-50 flex items-center justify-center p-4"
        @click="closeModal"
      >
        <div
          class="bg-white dark:bg-gray-800 rounded-xl shadow-2xl max-w-md w-full p-6"
          @click.stop
        >
          <!-- 헤더 -->
          <div class="text-center mb-6">
            <div class="w-16 h-16 mx-auto mb-4 bg-gradient-to-br from-blue-500 to-purple-600 rounded-2xl flex items-center justify-center">
              <svg class="w-8 h-8 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 18h.01M8 21h8a2 2 0 002-2V5a2 2 0 00-2-2H8a2 2 0 00-2 2v14a2 2 0 002 2z"></path>
              </svg>
            </div>
            <h3 class="text-xl font-bold text-gray-900 dark:text-gray-100 mb-2">
              AICode Manager 설치
            </h3>
            <p class="text-gray-600 dark:text-gray-400 text-sm">
              홈 화면에 추가하여 앱처럼 사용하세요
            </p>
          </div>

          <!-- 기능 소개 -->
          <div class="mb-6">
            <h4 class="font-semibold text-gray-900 dark:text-gray-100 mb-3">설치 후 이용 가능한 기능:</h4>
            <ul class="space-y-2">
              <li class="flex items-center gap-3 text-sm text-gray-600 dark:text-gray-400">
                <div class="w-5 h-5 rounded-full bg-green-100 dark:bg-green-900 flex items-center justify-center">
                  <svg class="w-3 h-3 text-green-600 dark:text-green-400" fill="currentColor" viewBox="0 0 20 20">
                    <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd"></path>
                  </svg>
                </div>
                <span>오프라인에서도 캐시된 데이터 접근 가능</span>
              </li>
              <li class="flex items-center gap-3 text-sm text-gray-600 dark:text-gray-400">
                <div class="w-5 h-5 rounded-full bg-green-100 dark:bg-green-900 flex items-center justify-center">
                  <svg class="w-3 h-3 text-green-600 dark:text-green-400" fill="currentColor" viewBox="0 0 20 20">
                    <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd"></path>
                  </svg>
                </div>
                <span>더 빠른 앱 시작 시간</span>
              </li>
              <li class="flex items-center gap-3 text-sm text-gray-600 dark:text-gray-400">
                <div class="w-5 h-5 rounded-full bg-green-100 dark:bg-green-900 flex items-center justify-center">
                  <svg class="w-3 h-3 text-green-600 dark:text-green-400" fill="currentColor" viewBox="0 0 20 20">
                    <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd"></path>
                  </svg>
                </div>
                <span>전체 화면 앱 경험</span>
              </li>
              <li class="flex items-center gap-3 text-sm text-gray-600 dark:text-gray-400">
                <div class="w-5 h-5 rounded-full bg-green-100 dark:bg-green-900 flex items-center justify-center">
                  <svg class="w-3 h-3 text-green-600 dark:text-green-400" fill="currentColor" viewBox="0 0 20 20">
                    <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd"></path>
                  </svg>
                </div>
                <span>푸시 알림 지원 (향후 제공)</span>
              </li>
            </ul>
          </div>

          <!-- iOS 수동 설치 가이드 -->
          <div v-if="isIOS && !isInstallable" class="mb-6 p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
            <h4 class="font-semibold text-blue-900 dark:text-blue-100 mb-2">iOS 설치 가이드:</h4>
            <ol class="text-sm text-blue-800 dark:text-blue-200 space-y-1">
              <li v-for="(step, index) in manualInstallSteps" :key="index" class="flex gap-2">
                <span class="font-medium">{{ index + 1 }}.</span>
                <span>{{ step }}</span>
              </li>
            </ol>
          </div>

          <!-- 액션 버튼들 -->
          <div class="flex gap-3">
            <button
              v-if="isInstallable"
              @click="handleInstall"
              :disabled="isInstalling"
              class="flex-1 bg-gradient-to-r from-blue-500 to-purple-600 text-white py-3 px-4 rounded-lg font-medium hover:from-blue-600 hover:to-purple-700 transition-all disabled:opacity-50"
            >
              <span v-if="isInstalling" class="flex items-center justify-center gap-2">
                <svg class="animate-spin w-4 h-4" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
                설치 중...
              </span>
              <span v-else>지금 설치하기</span>
            </button>
            
            <button
              v-if="!isInstallable"
              @click="closeModal"
              class="flex-1 bg-blue-500 text-white py-3 px-4 rounded-lg font-medium hover:bg-blue-600 transition-colors"
            >
              확인
            </button>

            <button
              @click="dismissPermanently"
              class="px-4 py-3 text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 text-sm transition-colors"
            >
              다시 묻지 않기
            </button>
          </div>

          <!-- 닫기 버튼 -->
          <button
            @click="closeModal"
            class="absolute top-4 right-4 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 transition-colors"
            aria-label="모달 닫기"
          >
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
            </svg>
          </button>
        </div>
      </div>
    </Transition>

    <!-- 플로팅 설치 버튼 -->
    <Transition name="floating-button">
      <button
        v-if="showFloatingButton && shouldShowPrompt && !showBanner && !showModal"
        @click="openModal"
        class="fixed bottom-6 right-6 z-40 bg-gradient-to-r from-blue-500 to-purple-600 text-white p-4 rounded-full shadow-lg hover:shadow-xl transform hover:scale-105 transition-all duration-200"
        aria-label="앱 설치"
      >
        <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 18h.01M8 21h8a2 2 0 002-2V5a2 2 0 00-2-2H8a2 2 0 00-2 2v14a2 2 0 002 2z"></path>
        </svg>
      </button>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { usePWAInstall } from '@/composables/usePWAInstall'

const {
  isInstallable,
  isInstalled,
  isIOS,
  shouldShowPrompt,
  showInstallPrompt,
  showManualInstallGuide,
  incrementDismissedCount,
  markAsUserRejected,
} = usePWAInstall()

const showBanner = ref(false)
const showModal = ref(false)
const showFloatingButton = ref(false)
const isInstalling = ref(false)

// 수동 설치 가이드 단계
const manualInstallGuide = computed(() => showManualInstallGuide())
const manualInstallSteps = computed(() => manualInstallGuide.value.steps)

// Props
interface Props {
  autoShow?: boolean
  showBannerDefault?: boolean
  showFloatingDefault?: boolean
  delay?: number
}

const props = withDefaults(defineProps<Props>(), {
  autoShow: true,
  showBannerDefault: true,
  showFloatingDefault: true,
  delay: 3000, // 3초 후 표시
})

// 설치 처리
const handleInstall = async () => {
  if (!isInstallable.value) {
    if (isIOS.value) {
      // iOS는 수동 설치 가이드만 표시
      return
    }
    return
  }

  isInstalling.value = true
  try {
    const success = await showInstallPrompt()
    if (success) {
      console.log('PWA 설치 성공')
      showBanner.value = false
      showModal.value = false
    } else {
      console.log('PWA 설치 취소됨')
    }
  } catch (error) {
    console.error('PWA 설치 오류:', error)
  } finally {
    isInstalling.value = false
  }
}

// 배너 닫기
const dismissBanner = () => {
  showBanner.value = false
  incrementDismissedCount()
  
  // 플로팅 버튼 표시 (옵션)
  if (props.showFloatingDefault) {
    setTimeout(() => {
      showFloatingButton.value = true
    }, 5000) // 5초 후
  }
}

// 모달 열기
const openModal = () => {
  showModal.value = true
  showFloatingButton.value = false
}

// 모달 닫기
const closeModal = () => {
  showModal.value = false
  incrementDismissedCount()
  
  // 플로팅 버튼 다시 표시
  if (props.showFloatingDefault) {
    setTimeout(() => {
      showFloatingButton.value = true
    }, 10000) // 10초 후
  }
}

// 영구적으로 거부
const dismissPermanently = () => {
  markAsUserRejected()
  showBanner.value = false
  showModal.value = false
  showFloatingButton.value = false
}

// 자동 표시 로직
const initializeAutoShow = () => {
  if (!props.autoShow || !shouldShowPrompt.value) return

  setTimeout(() => {
    if (props.showBannerDefault) {
      showBanner.value = true
    } else if (props.showFloatingDefault) {
      showFloatingButton.value = true
    }
  }, props.delay)
}

// 설치 상태 변화 감지
watch([isInstalled, shouldShowPrompt], ([installed, shouldShow]) => {
  if (installed || !shouldShow) {
    showBanner.value = false
    showModal.value = false
    showFloatingButton.value = false
  }
})

onMounted(() => {
  initializeAutoShow()
})

// 컴포넌트 외부에서 호출할 수 있는 메서드들
defineExpose({
  openModal,
  closeModal,
  dismissBanner,
  dismissPermanently,
  handleInstall,
})
</script>

<style scoped>
.install-banner-enter-active,
.install-banner-leave-active {
  transition: transform 0.3s ease-in-out;
}

.install-banner-enter-from {
  transform: translateY(-100%);
}

.install-banner-leave-to {
  transform: translateY(-100%);
}

.install-modal-enter-active,
.install-modal-leave-active {
  transition: opacity 0.3s ease;
}

.install-modal-enter-from,
.install-modal-leave-to {
  opacity: 0;
}

.floating-button-enter-active,
.floating-button-leave-active {
  transition: all 0.3s ease;
}

.floating-button-enter-from,
.floating-button-leave-to {
  opacity: 0;
  transform: translateY(100px);
}

/* 펄스 애니메이션 */
@keyframes pulse {
  0%, 100% { transform: scale(1); }
  50% { transform: scale(1.05); }
}

.floating-button:hover {
  animation: pulse 2s infinite;
}
</style>