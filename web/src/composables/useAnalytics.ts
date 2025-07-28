import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { type AnalyticsEvent, type AnalyticsEventType, analytics } from '@/utils/analytics'

/**
 * 분석 상태 인터페이스
 */
interface AnalyticsState {
  isInitialized: boolean
  sessionId: string
  userId?: string
  events: AnalyticsEvent[]
  isTracking: boolean
}

/**
 * 이벤트 통계 인터페이스
 */
interface EventStats {
  total: number
  byType: Record<AnalyticsEventType, number>
  byHour: Array<{ hour: number; count: number }>
  topEvents: Array<{ name: string; count: number }>
}

/**
 * 사용자 세션 정보
 */
interface SessionInfo {
  sessionId: string
  startTime: number
  duration: number
  pageViews: number
  interactions: number
  lastActivity: number
}

/**
 * 분석을 위한 Composable
 */
export function useAnalytics() {
  // 반응형 상태
  const state = reactive<AnalyticsState>({
    isInitialized: false,
    sessionId: '',
    userId: undefined,
    events: [],
    isTracking: false,
  })

  // 세션 정보
  const sessionInfo = ref<SessionInfo>({
    sessionId: '',
    startTime: Date.now(),
    duration: 0,
    pageViews: 0,
    interactions: 0,
    lastActivity: Date.now(),
  })

  // 활동 타이머
  let activityTimer: NodeJS.Timeout | null = null
  let sessionTimer: NodeJS.Timeout | null = null

  /**
   * 분석 초기화
   */
  const initialize = async (options?: {
    userId?: string
    userProperties?: Record<string, any>
  }) => {
    try {
      await analytics.initialize()

      state.isInitialized = true
      state.isTracking = true

      if (options?.userId) {
        await identify(options.userId, options.userProperties)
      }

      // 세션 정보 초기화
      sessionInfo.value.sessionId = (analytics as any).sessionId
      sessionInfo.value.startTime = Date.now()

      // 주기적으로 세션 시간 업데이트
      sessionTimer = setInterval(updateSessionDuration, 1000)

      // 활동 추적 시작
      startActivityTracking()

      console.log('Analytics initialized')
    } catch (error) {
      console.error('Failed to initialize analytics:', error)
    }
  }

  /**
   * 사용자 식별
   */
  const identify = async (userId: string, properties?: Record<string, any>) => {
    if (!state.isInitialized) return

    state.userId = userId
    await analytics.identify(userId, properties)
  }

  /**
   * 사용자 속성 설정
   */
  const setUserProperties = async (properties: Record<string, any>) => {
    if (!state.isInitialized) return

    await analytics.setUserProperties(properties)
  }

  /**
   * 이벤트 추적
   */
  const track = async (
    eventType: AnalyticsEventType,
    eventName: string,
    properties?: Record<string, any>,
  ) => {
    if (!state.isInitialized || !state.isTracking) return

    await analytics.track(eventType, eventName, properties)
    updateActivity()
  }

  /**
   * 페이지 뷰 추적
   */
  const trackPageView = async (url?: string, title?: string) => {
    await track('page_view', 'page_view', {
      page_url: url || window.location.href,
      page_title: title || document.title,
    })

    sessionInfo.value.pageViews++
  }

  /**
   * 클릭 이벤트 추적
   */
  const trackClick = async (element: string, properties?: Record<string, any>) => {
    await track('user_interaction', 'click', {
      element,
      ...properties,
    })

    sessionInfo.value.interactions++
  }

  /**
   * 폼 제출 추적
   */
  const trackFormSubmit = async (formName: string, properties?: Record<string, any>) => {
    await track('user_interaction', 'form_submit', {
      form_name: formName,
      ...properties,
    })

    sessionInfo.value.interactions++
  }

  /**
   * 기능 사용 추적
   */
  const trackFeature = async (feature: string, action: string, properties?: Record<string, any>) => {
    await track('feature_usage', 'feature_used', {
      feature_name: feature,
      feature_action: action,
      ...properties,
    })
  }

  /**
   * 검색 추적
   */
  const trackSearch = async (query: string, results?: number, properties?: Record<string, any>) => {
    await track('user_interaction', 'search', {
      search_query: query,
      search_results: results,
      ...properties,
    })
  }

  /**
   * 다운로드 추적
   */
  const trackDownload = async (fileName: string, fileType?: string, properties?: Record<string, any>) => {
    await track('user_interaction', 'download', {
      file_name: fileName,
      file_type: fileType,
      ...properties,
    })
  }

  /**
   * 에러 추적
   */
  const trackError = async (error: string, details?: Record<string, any>) => {
    await track('error_occurrence', 'error', {
      error_message: error,
      ...details,
    })
  }

  /**
   * 성능 메트릭 추적
   */
  const trackPerformance = async (metric: string, value: number, unit?: string) => {
    await track('performance_metric', 'performance', {
      metric_name: metric,
      metric_value: value,
      metric_unit: unit || 'ms',
    })
  }

  /**
   * 변환 추적
   */
  const trackConversion = async (event: string, value?: number, currency?: string) => {
    await track('conversion', 'conversion', {
      conversion_event: event,
      conversion_value: value,
      currency: currency || 'KRW',
    })
  }

  /**
   * 사용자 정의 이벤트 추적
   */
  const trackCustom = async (eventName: string, properties?: Record<string, any>) => {
    await track('custom', eventName, properties)
  }

  /**
   * 추적 시작
   */
  const startTracking = () => {
    state.isTracking = true
  }

  /**
   * 추적 중지
   */
  const stopTracking = () => {
    state.isTracking = false
  }

  /**
   * 활동 추적 시작
   */
  const startActivityTracking = () => {
    // 마우스 이동 추적
    const trackMouseMove = () => updateActivity()

    // 키보드 입력 추적
    const trackKeyPress = () => updateActivity()

    // 스크롤 추적
    const trackScroll = () => updateActivity()

    // 클릭 추적
    const trackClickActivity = (event: MouseEvent) => {
      const element = event.target as HTMLElement
      const tagName = element.tagName.toLowerCase()
      const elementId = element.id
      const className = element.className

      trackClick(`${tagName}${elementId ? `#${elementId}` : ''}${className ? `.${className.split(' ')[0]}` : ''}`, {
        x: event.clientX,
        y: event.clientY,
        timestamp: Date.now(),
      })
    }

    // 이벤트 리스너 등록
    document.addEventListener('mousemove', trackMouseMove, { passive: true })
    document.addEventListener('keypress', trackKeyPress, { passive: true })
    document.addEventListener('scroll', trackScroll, { passive: true })
    document.addEventListener('click', trackClickActivity, { passive: true })

    // 언마운트 시 정리를 위한 함수 반환
    return () => {
      document.removeEventListener('mousemove', trackMouseMove)
      document.removeEventListener('keypress', trackKeyPress)
      document.removeEventListener('scroll', trackScroll)
      document.removeEventListener('click', trackClickActivity)
    }
  }

  /**
   * 활동 업데이트
   */
  const updateActivity = () => {
    sessionInfo.value.lastActivity = Date.now()

    // 활동 타이머 리셋
    if (activityTimer) {
      clearTimeout(activityTimer)
    }

    // 30분 후 비활성 상태로 간주
    activityTimer = setTimeout(() => {
      console.log('User inactive for 30 minutes')
    }, 30 * 60 * 1000)
  }

  /**
   * 세션 시간 업데이트
   */
  const updateSessionDuration = () => {
    sessionInfo.value.duration = Date.now() - sessionInfo.value.startTime
  }

  /**
   * 이벤트 통계 계산
   */
  const eventStats = computed<EventStats>(() => {
    const events = state.events
    const stats: EventStats = {
      total: events.length,
      byType: {
        page_view: 0,
        user_interaction: 0,
        feature_usage: 0,
        error_occurrence: 0,
        performance_metric: 0,
        conversion: 0,
        custom: 0,
      },
      byHour: [],
      topEvents: [],
    }

    // 타입별 통계
    events.forEach(event => {
      stats.byType[event.eventType]++
    })

    // 시간별 통계 (최근 24시간)
    const now = Date.now()
    const hours = Array.from({ length: 24 }, (_, i) => {
      const hour = new Date(now - (23 - i) * 60 * 60 * 1000).getHours()
      const hourStart = now - (23 - i) * 60 * 60 * 1000
      const hourEnd = hourStart + 60 * 60 * 1000

      const count = events.filter(event =>
        event.timestamp >= hourStart && event.timestamp < hourEnd,
      ).length

      return { hour, count }
    })
    stats.byHour = hours

    // 인기 이벤트
    const eventCounts: Record<string, number> = {}
    events.forEach(event => {
      eventCounts[event.eventName] = (eventCounts[event.eventName] || 0) + 1
    })

    stats.topEvents = Object.entries(eventCounts)
      .sort(([, a], [, b]) => b - a)
      .slice(0, 10)
      .map(([name, count]) => ({ name, count }))

    return stats
  })

  /**
   * 세션 요약
   */
  const sessionSummary = computed(() => {
    const duration = sessionInfo.value.duration
    const hours = Math.floor(duration / (1000 * 60 * 60))
    const minutes = Math.floor((duration % (1000 * 60 * 60)) / (1000 * 60))
    const seconds = Math.floor((duration % (1000 * 60)) / 1000)

    return {
      ...sessionInfo.value,
      formattedDuration: `${hours}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`,
      avgInteractionsPerMinute: duration > 0 ? sessionInfo.value.interactions / (duration / (1000 * 60)) : 0,
      isActive: Date.now() - sessionInfo.value.lastActivity < 5 * 60 * 1000, // 5분 이내 활동
    }
  })

  /**
   * 사용자 여정 추적
   */
  const userJourney = computed(() => {
    const pageViews = state.events.filter(e => e.eventType === 'page_view')
    return pageViews.map(event => ({
      url: event.properties?.page_url || 'unknown',
      title: event.properties?.page_title || 'unknown',
      timestamp: event.timestamp,
      timeSpent: 0, // 다음 페이지 뷰와의 시간 차이로 계산 가능
    }))
  })

  /**
   * 데이터 내보내기
   */
  const exportData = () => {
    const data = {
      session: sessionSummary.value,
      events: state.events,
      stats: eventStats.value,
      journey: userJourney.value,
      exportedAt: new Date().toISOString(),
    }

    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `analytics-data-${Date.now()}.json`
    a.click()
    URL.revokeObjectURL(url)
  }

  // 라이프사이클 훅
  onMounted(() => {
    initialize()
  })

  onUnmounted(() => {
    if (activityTimer) {
      clearTimeout(activityTimer)
    }
    if (sessionTimer) {
      clearInterval(sessionTimer)
    }
  })

  return {
    // 상태
    state: readonly(state),
    sessionInfo: readonly(sessionInfo),

    // 계산된 값
    eventStats,
    sessionSummary,
    userJourney,

    // 초기화 및 설정
    initialize,
    identify,
    setUserProperties,

    // 이벤트 추적
    track,
    trackPageView,
    trackClick,
    trackFormSubmit,
    trackFeature,
    trackSearch,
    trackDownload,
    trackError,
    trackPerformance,
    trackConversion,
    trackCustom,

    // 제어
    startTracking,
    stopTracking,

    // 유틸리티
    exportData,

    // 상태 확인
    isInitialized: computed(() => state.isInitialized),
    isTracking: computed(() => state.isTracking),
    hasData: computed(() => state.events.length > 0),
  }
}

/**
 * 자동 이벤트 추적을 위한 디렉티브
 */
export const vTrackClick = {
  mounted(el: HTMLElement, binding: any) {
    const handler = () => {
      const { event = 'click', properties = {} } = binding.value || {}
      analytics.track('user_interaction', event, {
        element: el.tagName.toLowerCase(),
        elementId: el.id,
        elementClass: el.className,
        elementText: el.textContent?.slice(0, 50),
        ...properties,
      })
    }

    el.addEventListener('click', handler)
    el._trackClickHandler = handler
  },

  unmounted(el: HTMLElement) {
    if (el._trackClickHandler) {
      el.removeEventListener('click', el._trackClickHandler)
      delete el._trackClickHandler
    }
  },
}

/**
 * 성능 측정 데코레이터
 */
export function withPerformanceTracking<T extends (...args: any[]) => any>(
  functionName: string,
  fn: T,
): T {
  return ((...args: any[]) => {
    const start = performance.now()

    try {
      const result = fn(...args)

      // Promise인 경우
      if (result && typeof result.then === 'function') {
        return result
          .then((value: any) => {
            const duration = performance.now() - start
            analytics.trackPerformance(`function_${functionName}`, duration)
            return value
          })
          .catch((error: any) => {
            const duration = performance.now() - start
            analytics.trackPerformance(`function_${functionName}`, duration)
            analytics.trackError(`Function error: ${functionName}`, { error: error.message })
            throw error
          })
      }

      // 동기 함수인 경우
      const duration = performance.now() - start
      analytics.trackPerformance(`function_${functionName}`, duration)
      return result
    } catch (error) {
      const duration = performance.now() - start
      analytics.trackPerformance(`function_${functionName}`, duration)
      analytics.trackError(`Function error: ${functionName}`, { error: (error as Error).message })
      throw error
    }
  }) as T
}

// 글로벌 타입 확장
declare module '@vue/runtime-core' {
  interface ComponentCustomProperties {
    $analytics: typeof analytics
  }
}

declare global {
  interface HTMLElement {
    _trackClickHandler?: EventListener
  }
}