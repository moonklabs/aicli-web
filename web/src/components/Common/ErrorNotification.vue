<template>
  <Teleport to="body">
    <Transition name="notification-list" mode="default">
      <div
        v-if="notifications.length > 0"
        class="fixed top-4 right-4 z-50 flex flex-col gap-2 max-w-md"
      >
        <TransitionGroup name="notification" tag="div" class="flex flex-col gap-2">
          <div
            v-for="notification in notifications"
            :key="notification.id"
            :class="[
              'notification-card',
              'p-4 rounded-lg shadow-lg border-l-4 backdrop-blur-sm',
              getNotificationStyles(notification.type),
              { 'animate-pulse': notification.isRetrying }
            ]"
          >
            <!-- 상단 헤더 -->
            <div class="flex items-center justify-between mb-2">
              <div class="flex items-center gap-2">
                <Icon
                  :name="getNotificationIcon(notification.type)"
                  :class="getIconStyles(notification.type)"
                  size="16"
                />
                <span class="font-medium text-sm">
                  {{ getNotificationTitle(notification.type) }}
                </span>
              </div>

              <!-- 닫기 버튼 -->
              <button
                @click="removeNotification(notification.id)"
                class="text-gray-400 hover:text-gray-600 transition-colors"
              >
                <Icon name="x" size="14" />
              </button>
            </div>

            <!-- 메시지 내용 -->
            <p class="text-sm text-gray-700 dark:text-gray-300 mb-2">
              {{ notification.message }}
            </p>

            <!-- 추가 정보 (개발 모드에서만) -->
            <div v-if="isDev && notification.details" class="mb-2">
              <details class="text-xs">
                <summary class="cursor-pointer text-gray-500 hover:text-gray-700">
                  기술적 세부사항
                </summary>
                <pre class="mt-1 p-2 bg-gray-100 dark:bg-gray-800 rounded text-xs overflow-auto">{{ notification.details }}</pre>
              </details>
            </div>

            <!-- 액션 버튼들 -->
            <div v-if="notification.actions.length > 0" class="flex gap-2">
              <button
                v-for="action in notification.actions"
                :key="action.label"
                @click="handleAction(notification, action)"
                :disabled="notification.isRetrying"
                :class="[
                  'px-3 py-1 text-xs rounded border transition-colors',
                  action.primary
                    ? 'bg-blue-500 text-white border-blue-500 hover:bg-blue-600'
                    : 'bg-gray-100 text-gray-700 border-gray-300 hover:bg-gray-200',
                  { 'opacity-50 cursor-not-allowed': notification.isRetrying }
                ]"
              >
                <Icon v-if="notification.isRetrying && action.label === '재시도'" name="loading" class="animate-spin mr-1" size="12" />
                {{ action.label }}
              </button>
            </div>

            <!-- 진행률 바 (재시도 중일 때) -->
            <div v-if="notification.isRetrying" class="mt-2">
              <div class="w-full bg-gray-200 rounded-full h-1">
                <div
                  class="bg-blue-500 h-1 rounded-full transition-all duration-1000"
                  :style="{ width: `${notification.retryProgress || 0}%` }"
                ></div>
              </div>
              <p class="text-xs text-gray-500 mt-1">
                재시도 중... ({{ notification.retryCount }}/{{ notification.maxRetries }})
              </p>
            </div>

            <!-- 자동 사라짐 타이머 -->
            <div v-if="notification.autoHide && !notification.isRetrying" class="mt-2">
              <div class="w-full bg-gray-200 rounded-full h-0.5">
                <div
                  class="bg-gray-400 h-0.5 rounded-full transition-all linear"
                  :style="{
                    width: `${notification.hideProgress || 100}%`,
                    transitionDuration: `${notification.autoHideDelay}ms`
                  }"
                ></div>
              </div>
            </div>
          </div>
        </TransitionGroup>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Icon } from '@iconify/vue'
import { useErrorNotification } from '@/composables/useErrorNotification'

const {
  notifications,
  removeNotification,
  retryAction,
} = useErrorNotification()

const isDev = computed(() => import.meta.env.DEV)

// 알림 타입별 스타일
const getNotificationStyles = (type: string): string => {
  const styles = {
    error: 'bg-red-50 dark:bg-red-900/20 border-red-400 text-red-800 dark:text-red-200',
    warning: 'bg-yellow-50 dark:bg-yellow-900/20 border-yellow-400 text-yellow-800 dark:text-yellow-200',
    info: 'bg-blue-50 dark:bg-blue-900/20 border-blue-400 text-blue-800 dark:text-blue-200',
    success: 'bg-green-50 dark:bg-green-900/20 border-green-400 text-green-800 dark:text-green-200',
    network: 'bg-orange-50 dark:bg-orange-900/20 border-orange-400 text-orange-800 dark:text-orange-200',
  }
  return styles[type as keyof typeof styles] || styles.error
}

// 알림 타입별 아이콘
const getNotificationIcon = (type: string): string => {
  const icons = {
    error: 'mdi:alert-circle',
    warning: 'mdi:alert',
    info: 'mdi:information',
    success: 'mdi:check-circle',
    network: 'mdi:wifi-off',
  }
  return icons[type as keyof typeof icons] || icons.error
}

// 아이콘 스타일
const getIconStyles = (type: string): string => {
  const styles = {
    error: 'text-red-500',
    warning: 'text-yellow-500',
    info: 'text-blue-500',
    success: 'text-green-500',
    network: 'text-orange-500',
  }
  return styles[type as keyof typeof styles] || styles.error
}

// 알림 타입별 제목
const getNotificationTitle = (type: string): string => {
  const titles = {
    error: '오류 발생',
    warning: '주의',
    info: '알림',
    success: '성공',
    network: '네트워크 오류',
  }
  return titles[type as keyof typeof titles] || titles.error
}

// 액션 핸들러
const handleAction = async (notification: any, action: any) => {
  if (action.label === '재시도' && action.handler) {
    await retryAction(notification.id, action.handler)
  } else if (action.handler) {
    await action.handler()
  }

  if (action.dismiss !== false) {
    removeNotification(notification.id)
  }
}
</script>

<style scoped>
.notification-list-enter-active,
.notification-list-leave-active {
  transition: all 0.3s ease;
}

.notification-list-enter-from,
.notification-list-leave-to {
  opacity: 0;
  transform: translateY(-20px);
}

.notification-enter-active,
.notification-leave-active {
  transition: all 0.3s ease;
}

.notification-enter-from,
.notification-leave-to {
  opacity: 0;
  transform: translateX(100%);
}

.notification-move {
  transition: transform 0.3s ease;
}

.notification-card {
  animation: slideInRight 0.3s ease-out;
}

@keyframes slideInRight {
  from {
    opacity: 0;
    transform: translateX(100%);
  }
  to {
    opacity: 1;
    transform: translateX(0);
  }
}

/* 반응형 스타일 */
@media (max-width: 640px) {
  .notification-card {
    margin: 0 1rem;
    max-width: calc(100vw - 2rem);
  }
}
</style>