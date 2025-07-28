<template>
  <Teleport to="body">
    <!-- 오프라인 배너 -->
    <Transition name="offline-banner">
      <div
        v-if="!isOnline"
        class="fixed top-0 left-0 right-0 z-40 bg-red-500 text-white py-2 px-4 text-center text-sm font-medium shadow-lg"
      >
        <div class="flex items-center justify-center gap-2">
          📶❌
          <span>인터넷 연결이 끊어졌습니다</span>
          <button
            @click="checkConnection"
            :disabled="isChecking"
            class="ml-2 px-2 py-1 bg-red-600 hover:bg-red-700 rounded text-xs transition-colors disabled:opacity-50"
          >
            <span v-if="isChecking">⏳</span>
            <span v-else>연결 확인</span>
          </button>
        </div>
      </div>
    </Transition>

    <!-- 오프라인 오버레이 (중요한 기능 차단시) -->
    <Transition name="offline-overlay">
      <div
        v-if="showOfflineOverlay"
        class="fixed inset-0 z-50 bg-black bg-opacity-50 flex items-center justify-center p-4"
        @click="showOfflineOverlay = false"
      >
        <div
          class="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-md w-full p-6 text-center"
          @click.stop
        >
          <div class="mb-4">
            <div class="text-red-500 mx-auto mb-2 text-5xl">📶❌</div>
            <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-2">
              오프라인 상태
            </h3>
            <p class="text-gray-600 dark:text-gray-400 text-sm">
              현재 인터넷 연결이 끊어진 상태입니다. 일부 기능이 제한될 수 있습니다.
            </p>
          </div>

          <!-- 네트워크 상태 정보 -->
          <div class="mb-4 p-3 bg-gray-50 dark:bg-gray-700 rounded text-sm">
            <div class="flex justify-between items-center mb-1">
              <span class="text-gray-600 dark:text-gray-400">연결 상태:</span>
              <span class="text-red-500 font-medium">오프라인</span>
            </div>
            <div class="flex justify-between items-center mb-1">
              <span class="text-gray-600 dark:text-gray-400">마지막 온라인:</span>
              <span class="text-gray-800 dark:text-gray-200">
                {{ formatLastOnlineTime }}
              </span>
            </div>
            <div class="flex justify-between items-center">
              <span class="text-gray-600 dark:text-gray-400">대기 중인 요청:</span>
              <span class="text-gray-800 dark:text-gray-200">
                {{ pendingRequests }}개
              </span>
            </div>
          </div>

          <!-- 사용 가능한 기능 안내 -->
          <div class="mb-4 text-left">
            <h4 class="font-medium text-gray-900 dark:text-gray-100 mb-2">오프라인에서 사용 가능:</h4>
            <ul class="text-sm text-gray-600 dark:text-gray-400 space-y-1">
              <li class="flex items-center gap-2">
                <span class="text-green-500">✓</span>
                <span>캐시된 데이터 조회</span>
              </li>
              <li class="flex items-center gap-2">
                <span class="text-green-500">✓</span>
                <span>로컬 설정 변경</span>
              </li>
              <li class="flex items-center gap-2">
                <span class="text-green-500">✓</span>
                <span>임시 저장 기능</span>
              </li>
            </ul>
          </div>

          <div class="mb-4 text-left">
            <h4 class="font-medium text-gray-900 dark:text-gray-100 mb-2">제한되는 기능:</h4>
            <ul class="text-sm text-gray-600 dark:text-gray-400 space-y-1">
              <li class="flex items-center gap-2">
                <span class="text-red-500">✗</span>
                <span>실시간 데이터 동기화</span>
              </li>
              <li class="flex items-center gap-2">
                <span class="text-red-500">✗</span>
                <span>파일 업로드/다운로드</span>
              </li>
              <li class="flex items-center gap-2">
                <span class="text-red-500">✗</span>
                <span>서버 API 호출</span>
              </li>
            </ul>
          </div>

          <!-- 액션 버튼들 -->
          <div class="flex gap-2">
            <button
              @click="checkConnection"
              :disabled="isChecking"
              class="flex-1 bg-blue-500 hover:bg-blue-600 text-white py-2 px-4 rounded transition-colors disabled:opacity-50"
            >
              <span v-if="isChecking" class="mr-2">⏳</span>
              {{ isChecking ? '확인 중...' : '연결 재시도' }}
            </button>
            <button
              @click="showOfflineOverlay = false"
              class="flex-1 bg-gray-500 hover:bg-gray-600 text-white py-2 px-4 rounded transition-colors"
            >
              계속 진행
            </button>
          </div>
        </div>
      </div>
    </Transition>

    <!-- 네트워크 복구 알림 -->
    <Transition name="reconnected-toast">
      <div
        v-if="showReconnected"
        class="fixed bottom-4 right-4 z-50 bg-green-500 text-white py-3 px-4 rounded-lg shadow-lg flex items-center gap-2"
      >
        📶
        <span class="font-medium">네트워크 연결이 복구되었습니다!</span>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
// import { Icon } from '@iconify/vue'
import { useNetworkStatus } from '@/composables/useNetworkStatus'
import { useGlobalErrorNotification } from '@/composables/useErrorNotification'

const {
  isOnline,
  lastOnlineTime,
  pendingRequests,
  forceReconnect,
} = useNetworkStatus()

// 사용하지 않는 변수들을 언더스코어로 표시
const { lastOfflineTime: _lastOfflineTime } = useNetworkStatus()
const { showSuccess: _showSuccess } = useGlobalErrorNotification()

const showOfflineOverlay = ref(false)
const showReconnected = ref(false)
const isChecking = ref(false)
const wasOffline = ref(false)

// 마지막 온라인 시간 포맷팅
const formatLastOnlineTime = computed(() => {
  if (!lastOnlineTime.value) return '알 수 없음'

  const now = new Date()
  const diff = now.getTime() - lastOnlineTime.value.getTime()

  if (diff < 60000) return '방금 전'
  if (diff < 3600000) return `${Math.floor(diff / 60000)}분 전`
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}시간 전`

  return lastOnlineTime.value.toLocaleDateString()
})

// 연결 확인
const checkConnection = async () => {
  isChecking.value = true
  try {
    const success = await forceReconnect()
    if (success) {
      showReconnected.value = true
      setTimeout(() => {
        showReconnected.value = false
      }, 3000)
    }
  } catch (error) {
    console.error('Connection check failed:', error)
  } finally {
    isChecking.value = false
  }
}

// 네트워크 상태 변화 감지
watch(isOnline, (online) => {
  if (!online && !wasOffline.value) {
    // 온라인에서 오프라인으로 변경
    wasOffline.value = true
    console.log('🔌 Network went offline')
  } else if (online && wasOffline.value) {
    // 오프라인에서 온라인으로 복구
    wasOffline.value = false
    showReconnected.value = true
    showOfflineOverlay.value = false

    console.log('🌐 Network reconnected')

    // 복구 알림 자동 숨김
    setTimeout(() => {
      showReconnected.value = false
    }, 3000)
  }
})

// 임계 기능 시도 시 오프라인 오버레이 표시
const showOfflineModal = () => {
  if (!isOnline.value) {
    showOfflineOverlay.value = true
  }
}

// 전역 함수로 노출 (다른 컴포넌트에서 호출 가능)
if (typeof window !== 'undefined') {
  (window as Record<string, unknown>).showOfflineModal = showOfflineModal
}

onMounted(() => {
  wasOffline.value = !isOnline.value
})

// 컴포넌트에서 직접 호출할 수 있도록 expose
defineExpose({
  showOfflineModal,
  checkConnection,
})
</script>

<style scoped>
.offline-banner-enter-active,
.offline-banner-leave-active {
  transition: transform 0.3s ease-in-out;
}

.offline-banner-enter-from {
  transform: translateY(-100%);
}

.offline-banner-leave-to {
  transform: translateY(-100%);
}

.offline-overlay-enter-active,
.offline-overlay-leave-active {
  transition: opacity 0.3s ease;
}

.offline-overlay-enter-from,
.offline-overlay-leave-to {
  opacity: 0;
}

.reconnected-toast-enter-active,
.reconnected-toast-leave-active {
  transition: all 0.3s ease;
}

.reconnected-toast-enter-from,
.reconnected-toast-leave-to {
  opacity: 0;
  transform: translateX(100%);
}

/* 오프라인 상태에서 특정 요소들 비활성화 스타일 */
:global(.offline-disabled) {
  opacity: 0.5;
  pointer-events: none;
  cursor: not-allowed;
}

/* 캐시된 데이터 표시 스타일 */
:global(.cached-data) {
  position: relative;
}

:global(.cached-data::after) {
  content: '캐시됨';
  position: absolute;
  top: -8px;
  right: -8px;
  background: #f59e0b;
  color: white;
  font-size: 10px;
  padding: 2px 6px;
  border-radius: 4px;
  z-index: 10;
}
</style>