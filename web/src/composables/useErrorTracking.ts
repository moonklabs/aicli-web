import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { type ErrorEvent, type ErrorLevel, errorTracker, logger } from '@/utils/error-tracker'

/**
 * 에러 트래킹 상태 인터페이스
 */
interface ErrorTrackingState {
  errors: ErrorEvent[]
  isTracking: boolean
  lastError?: ErrorEvent
}

/**
 * 에러 필터 옵션
 */
interface ErrorFilterOptions {
  level?: ErrorLevel
  component?: string
  timeRange?: {
    start: number
    end: number
  }
  searchQuery?: string
}

/**
 * 에러 트래킹을 위한 Composable
 */
export function useErrorTracking() {
  // 반응형 상태
  const state = reactive<ErrorTrackingState>({
    errors: [],
    isTracking: false,
    lastError: undefined,
  })

  // 필터 옵션
  const filterOptions = ref<ErrorFilterOptions>({})

  // 에러 리스너
  const errorListener = (error: ErrorEvent) => {
    state.errors.unshift(error) // 최신 에러를 맨 앞에 추가
    state.lastError = error

    // 최대 200개 에러만 유지
    if (state.errors.length > 200) {
      state.errors.splice(200)
    }
  }

  // 필터링된 에러 목록
  const filteredErrors = computed(() => {
    let filtered = state.errors

    const filter = filterOptions.value

    // 레벨 필터
    if (filter.level) {
      filtered = filtered.filter(error => error.level === filter.level)
    }

    // 컴포넌트 필터
    if (filter.component) {
      filtered = filtered.filter(error =>
        error.component?.toLowerCase().includes(filter.component!.toLowerCase()),
      )
    }

    // 시간 범위 필터
    if (filter.timeRange) {
      filtered = filtered.filter(error =>
        error.timestamp >= filter.timeRange!.start &&
        error.timestamp <= filter.timeRange!.end,
      )
    }

    // 검색 쿼리 필터
    if (filter.searchQuery) {
      const query = filter.searchQuery.toLowerCase()
      filtered = filtered.filter(error =>
        error.message.toLowerCase().includes(query) ||
        error.component?.toLowerCase().includes(query) ||
        error.action?.toLowerCase().includes(query),
      )
    }

    return filtered
  })

  // 에러 통계
  const errorStats = computed(() => {
    const stats = errorTracker.getStats()
    const total = Object.values(stats).reduce((sum, count) => sum + count, 0)

    return {
      ...stats,
      total,
      recentErrors: state.errors.filter(error =>
        Date.now() - error.timestamp < 60000, // 최근 1분
      ).length,
    }
  })

  // 에러 레벨별 색상
  const errorLevelColors = {
    debug: '#718096',
    info: '#4299e1',
    warn: '#ed8936',
    error: '#f56565',
    fatal: '#e53e3e',
  }

  // 심각도 순으로 정렬된 에러
  const errorsBySeverity = computed(() => {
    const severityOrder: ErrorLevel[] = ['fatal', 'error', 'warn', 'info', 'debug']

    return filteredErrors.value.sort((a, b) => {
      const severityA = severityOrder.indexOf(a.level)
      const severityB = severityOrder.indexOf(b.level)

      if (severityA !== severityB) {
        return severityA - severityB
      }

      return b.timestamp - a.timestamp // 최신순
    })
  })

  // 컴포넌트별 에러 통계
  const errorsByComponent = computed(() => {
    const componentStats: Record<string, { count: number; levels: Record<ErrorLevel, number> }> = {}

    state.errors.forEach(error => {
      const component = error.component || 'unknown'

      if (!componentStats[component]) {
        componentStats[component] = {
          count: 0,
          levels: { debug: 0, info: 0, warn: 0, error: 0, fatal: 0 },
        }
      }

      componentStats[component].count++
      componentStats[component].levels[error.level]++
    })

    return Object.entries(componentStats)
      .map(([component, stats]) => ({ component, ...stats }))
      .sort((a, b) => b.count - a.count)
  })

  // 시간대별 에러 분포
  const errorsByTime = computed(() => {
    const now = Date.now()
    const hours = 24
    const interval = (60 * 60 * 1000) // 1시간

    const timeSlots = Array.from({ length: hours }, (_, i) => {
      const startTime = now - (hours - i) * interval
      const endTime = startTime + interval

      return {
        hour: new Date(startTime).getHours(),
        count: state.errors.filter(error =>
          error.timestamp >= startTime && error.timestamp < endTime,
        ).length,
        startTime,
        endTime,
      }
    })

    return timeSlots
  })

  /**
   * 에러 트래킹 시작
   */
  const startTracking = () => {
    if (state.isTracking) return

    state.isTracking = true
    state.errors = errorTracker.getErrors()
    errorTracker.addListener(errorListener)

    logger.info('Error tracking started')
  }

  /**
   * 에러 트래킹 중지
   */
  const stopTracking = () => {
    if (!state.isTracking) return

    state.isTracking = false
    errorTracker.removeListener(errorListener)

    logger.info('Error tracking stopped')
  }

  /**
   * 에러 삭제
   */
  const clearErrors = () => {
    state.errors = []
    state.lastError = undefined
    errorTracker.clearErrors()

    logger.info('All errors cleared')
  }

  /**
   * 특정 에러 삭제
   */
  const removeError = (errorId: string) => {
    state.errors = state.errors.filter(error => error.id !== errorId)
    errorTracker.removeError(errorId)

    if (state.lastError?.id === errorId) {
      state.lastError = state.errors[0]
    }
  }

  /**
   * 필터 설정
   */
  const setFilter = (newFilter: Partial<ErrorFilterOptions>) => {
    filterOptions.value = { ...filterOptions.value, ...newFilter }
  }

  /**
   * 필터 리셋
   */
  const resetFilter = () => {
    filterOptions.value = {}
  }

  /**
   * 에러 내보내기
   */
  const exportErrors = () => {
    const data = errorTracker.exportErrors()
    const blob = new Blob([data], { type: 'application/json' })
    const url = URL.createObjectURL(blob)

    const a = document.createElement('a')
    a.href = url
    a.download = `error-log-${Date.now()}.json`
    a.click()

    URL.revokeObjectURL(url)
    logger.info('Error log exported')
  }

  /**
   * 테스트 에러 생성
   */
  const generateTestError = (level: ErrorLevel = 'error') => {
    const testMessages = {
      debug: 'Debug message for testing',
      info: 'Info message for testing',
      warn: 'Warning message for testing',
      error: 'Error message for testing',
      fatal: 'Fatal error message for testing',
    }

    logger[level](testMessages[level], {
      isTest: true,
      timestamp: Date.now(),
    })
  }

  /**
   * 에러 세부 정보 포맷
   */
  const formatErrorDetails = (error: ErrorEvent) => {
    return {
      basicInfo: {
        '메시지': error.message,
        '레벨': error.level,
        '시간': new Date(error.timestamp).toLocaleString('ko-KR'),
        '컴포넌트': error.component || '알 수 없음',
        '액션': error.action || '알 수 없음',
      },
      technicalInfo: {
        'URL': error.url,
        '사용자 에이전트': error.userAgent,
        '세션 ID': error.sessionId,
        '뷰포트': error.viewport ? `${error.viewport.width}x${error.viewport.height}` : '알 수 없음',
        '연결': error.connection || '알 수 없음',
        '언어': error.language || '알 수 없음',
        '메모리': error.memory ? `${error.memory.toFixed(2)}MB` : '알 수 없음',
      },
      stack: error.stack,
      metadata: error.metadata,
    }
  }

  /**
   * 에러 발생률 계산
   */
  const calculateErrorRate = (timeWindow = 3600000) => { // 기본 1시간
    const now = Date.now()
    const windowStart = now - timeWindow

    const errorsInWindow = state.errors.filter(error =>
      error.timestamp >= windowStart && error.level !== 'debug' && error.level !== 'info',
    )

    return {
      count: errorsInWindow.length,
      rate: errorsInWindow.length / (timeWindow / 60000), // 분당 에러 수
      period: timeWindow / 60000, // 분 단위
    }
  }

  // 라이프사이클 훅
  onMounted(() => {
    startTracking()
  })

  onUnmounted(() => {
    stopTracking()
  })

  return {
    // 상태
    state: readonly(state),
    filterOptions: readonly(filterOptions),

    // 계산된 값
    filteredErrors,
    errorStats,
    errorsBySeverity,
    errorsByComponent,
    errorsByTime,
    errorLevelColors,

    // 메서드
    startTracking,
    stopTracking,
    clearErrors,
    removeError,
    setFilter,
    resetFilter,
    exportErrors,
    generateTestError,
    formatErrorDetails,
    calculateErrorRate,

    // 유틸리티
    isTracking: computed(() => state.isTracking),
    hasErrors: computed(() => state.errors.length > 0),
    recentErrors: computed(() => errorStats.value.recentErrors),
    criticalErrors: computed(() =>
      state.errors.filter(error => error.level === 'fatal' || error.level === 'error').length,
    ),
  }
}

/**
 * 컴포넌트별 에러 트래킹 데코레이터
 */
export function withErrorTracking<T extends (...args: any[]) => any>(
  componentName: string,
  method: T,
  action?: string,
): T {
  return ((...args: any[]) => {
    try {
      const result = method(...args)

      // Promise인 경우 에러 처리
      if (result && typeof result.catch === 'function') {
        return result.catch((error: Error) => {
          logger.error(`Error in ${componentName}.${action || method.name}`, error, {
            args,
            component: componentName,
            action: action || method.name,
          })
          throw error
        })
      }

      return result
    } catch (error) {
      logger.error(`Error in ${componentName}.${action || method.name}`, error as Error, {
        args,
        component: componentName,
        action: action || method.name,
      })
      throw error
    }
  }) as T
}

/**
 * API 호출 에러 트래킹
 */
export function trackApiError(url: string, method: string, status?: number, error?: Error) {
  logger.error(`API Error: ${method} ${url}`, error, {
    component: 'api',
    action: method.toLowerCase(),
    url,
    status,
    isApiError: true,
  })
}