import { ref, computed, nextTick } from 'vue'
import type { AxiosError } from 'axios'

export interface NotificationAction {
  label: string
  handler?: () => Promise<void> | void
  primary?: boolean
  dismiss?: boolean
}

export interface ErrorNotification {
  id: string
  type: 'error' | 'warning' | 'info' | 'success' | 'network'
  message: string
  details?: string
  actions: NotificationAction[]
  autoHide: boolean
  autoHideDelay: number
  hideProgress?: number
  isRetrying: boolean
  retryCount: number
  maxRetries: number
  retryProgress?: number
  timestamp: Date
}

export interface NotificationOptions {
  type?: ErrorNotification['type']
  autoHide?: boolean
  autoHideDelay?: number
  actions?: NotificationAction[]
  details?: string
  retryable?: boolean
  maxRetries?: number
}

// 전역 알림 상태
const notifications = ref<ErrorNotification[]>([])
const notificationId = ref(0)

// 기본 설정
const DEFAULT_AUTO_HIDE_DELAY = 5000 // 5초
const MAX_NOTIFICATIONS = 5

export function useErrorNotification() {
  // 알림 추가
  const addNotification = (
    message: string,
    options: NotificationOptions = {}
  ): string => {
    const id = `notification-${++notificationId.value}`
    
    const notification: ErrorNotification = {
      id,
      type: options.type || 'error',
      message,
      details: options.details,
      actions: options.actions || [],
      autoHide: options.autoHide !== false, // 기본값: true
      autoHideDelay: options.autoHideDelay || DEFAULT_AUTO_HIDE_DELAY,
      hideProgress: 100,
      isRetrying: false,
      retryCount: 0,
      maxRetries: options.maxRetries || 3,
      timestamp: new Date()
    }

    // 재시도 가능한 경우 재시도 액션 추가
    if (options.retryable && !notification.actions.some(a => a.label === '재시도')) {
      notification.actions.unshift({
        label: '재시도',
        primary: true,
        dismiss: false
      })
    }

    notifications.value.unshift(notification)

    // 최대 알림 수 제한
    if (notifications.value.length > MAX_NOTIFICATIONS) {
      notifications.value = notifications.value.slice(0, MAX_NOTIFICATIONS)
    }

    // 자동 숨김 처리
    if (notification.autoHide) {
      startAutoHide(id)
    }

    return id
  }

  // 자동 숨김 타이머 시작
  const startAutoHide = (id: string) => {
    const notification = notifications.value.find(n => n.id === id)
    if (!notification) return

    let progress = 100
    const interval = 50 // 50ms 간격
    const decrement = (100 / notification.autoHideDelay) * interval

    const timer = setInterval(() => {
      progress -= decrement
      
      const currentNotification = notifications.value.find(n => n.id === id)
      if (currentNotification) {
        currentNotification.hideProgress = Math.max(0, progress)
      }

      if (progress <= 0) {
        clearInterval(timer)
        removeNotification(id)
      }
    }, interval)

    // 마우스 호버 시 타이머 일시정지 (TODO: 구현 고려)
  }

  // 알림 제거
  const removeNotification = (id: string) => {
    const index = notifications.value.findIndex(n => n.id === id)
    if (index > -1) {
      notifications.value.splice(index, 1)
    }
  }

  // 모든 알림 제거
  const clearAllNotifications = () => {
    notifications.value = []
  }

  // 재시도 액션 실행
  const retryAction = async (id: string, handler: () => Promise<void> | void) => {
    const notification = notifications.value.find(n => n.id === id)
    if (!notification) return

    notification.isRetrying = true
    notification.retryCount++

    // 재시도 진행률 시뮬레이션
    let progress = 0
    const progressInterval = setInterval(() => {
      progress += 10
      notification.retryProgress = Math.min(progress, 90)
    }, 200)

    try {
      await handler()
      
      // 성공 시 성공 알림으로 변경
      notification.type = 'success'
      notification.message = '문제가 해결되었습니다'
      notification.actions = []
      notification.autoHide = true
      notification.autoHideDelay = 3000
      
      clearInterval(progressInterval)
      notification.retryProgress = 100
      
      setTimeout(() => {
        removeNotification(id)
      }, 3000)
      
    } catch (error) {
      clearInterval(progressInterval)
      notification.retryProgress = 0
      
      // 최대 재시도 횟수 도달 시
      if (notification.retryCount >= notification.maxRetries) {
        notification.message = '재시도 횟수를 초과했습니다. 잠시 후 다시 시도해주세요.'
        notification.actions = notification.actions.filter(a => a.label !== '재시도')
      }
    } finally {
      notification.isRetrying = false
    }
  }

  // Axios 에러에서 알림 생성
  const handleAxiosError = (error: AxiosError, customMessage?: string): string => {
    let message = customMessage || '요청 처리 중 오류가 발생했습니다'
    let type: ErrorNotification['type'] = 'error'
    let details = ''
    let retryable = false

    if (error.response) {
      // 서버 응답이 있는 경우
      const status = error.response.status
      const data = error.response.data as any

      switch (status) {
        case 401:
          message = '인증이 필요합니다. 다시 로그인해주세요.'
          type = 'warning'
          break
        case 403:
          message = '접근 권한이 없습니다.'
          type = 'warning'
          break
        case 404:
          message = '요청한 리소스를 찾을 수 없습니다.'
          break
        case 408:
          message = '요청 시간이 초과되었습니다.'
          retryable = true
          break
        case 429:
          message = '너무 많은 요청이 발생했습니다. 잠시 후 다시 시도해주세요.'
          retryable = true
          break
        case 500:
        case 502:
        case 503:
        case 504:
          message = '서버에 일시적인 문제가 발생했습니다.'
          retryable = true
          break
        default:
          if (data?.message) {
            message = data.message
          }
      }

      details = `HTTP ${status}: ${error.response.statusText}\nURL: ${error.config?.url}`
      
    } else if (error.request) {
      // 요청은 보냈지만 응답이 없는 경우
      message = '네트워크 연결을 확인해주세요.'
      type = 'network'
      retryable = true
      details = 'No response received from server'
      
    } else {
      // 요청 설정 중 오류
      details = error.message
    }

    // 개발 모드에서는 더 자세한 정보 포함
    if (import.meta.env.DEV) {
      details += `\n\nStack: ${error.stack}`
    }

    return addNotification(message, {
      type,
      details,
      retryable,
      actions: retryable ? [] : [
        {
          label: '새로고침',
          handler: () => window.location.reload(),
          dismiss: true
        }
      ]
    })
  }

  // 네트워크 오류 처리
  const handleNetworkError = (isOnline: boolean): string => {
    if (isOnline) {
      return addNotification('네트워크 연결이 복구되었습니다.', {
        type: 'success',
        autoHide: true,
        autoHideDelay: 3000
      })
    } else {
      return addNotification('네트워크 연결이 끊어졌습니다.', {
        type: 'network',
        autoHide: false,
        actions: [
          {
            label: '연결 확인',
            handler: async () => {
              // 네트워크 연결 테스트
              try {
                await fetch('/api/health', { method: 'HEAD' })
                addNotification('네트워크 연결이 복구되었습니다.', { type: 'success' })
              } catch {
                addNotification('여전히 네트워크에 연결할 수 없습니다.', { type: 'error' })
              }
            },
            primary: true,
            dismiss: false
          }
        ]
      })
    }
  }

  // 성공 알림 (간편 함수)
  const showSuccess = (message: string, autoHideDelay = 3000): string => {
    return addNotification(message, {
      type: 'success',
      autoHide: true,
      autoHideDelay
    })
  }

  // 경고 알림 (간편 함수)
  const showWarning = (message: string, autoHide = true): string => {
    return addNotification(message, {
      type: 'warning',
      autoHide
    })
  }

  // 정보 알림 (간편 함수)
  const showInfo = (message: string, autoHide = true): string => {
    return addNotification(message, {
      type: 'info',
      autoHide
    })
  }

  // 에러 알림 (간편 함수)
  const showError = (message: string, retryable = false): string => {
    return addNotification(message, {
      type: 'error',
      retryable,
      autoHide: !retryable
    })
  }

  // 계산된 속성들
  const hasNotifications = computed(() => notifications.value.length > 0)
  const errorCount = computed(() => 
    notifications.value.filter(n => n.type === 'error').length
  )
  const warningCount = computed(() => 
    notifications.value.filter(n => n.type === 'warning').length
  )

  return {
    // 상태
    notifications: computed(() => notifications.value),
    hasNotifications,
    errorCount,
    warningCount,

    // 메서드
    addNotification,
    removeNotification,
    clearAllNotifications,
    retryAction,

    // 에러 처리
    handleAxiosError,
    handleNetworkError,

    // 간편 함수들
    showSuccess,
    showWarning,
    showInfo,
    showError
  }
}

// 전역 인스턴스 (싱글톤 패턴)
let globalErrorNotification: ReturnType<typeof useErrorNotification> | null = null

export function useGlobalErrorNotification() {
  if (!globalErrorNotification) {
    globalErrorNotification = useErrorNotification()
  }
  return globalErrorNotification
}