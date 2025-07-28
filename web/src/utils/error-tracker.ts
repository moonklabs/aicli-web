/**
 * 에러 트래킹 및 로깅 시스템
 */

export type ErrorLevel = 'debug' | 'info' | 'warn' | 'error' | 'fatal'

export interface ErrorEvent {
  id: string
  timestamp: number
  level: ErrorLevel
  message: string
  stack?: string
  url: string
  userAgent: string
  userId?: string
  sessionId: string

  // 에러 컨텍스트
  component?: string
  action?: string
  metadata?: Record<string, any>

  // 성능 정보
  memory?: number
  timestamp_performance?: number

  // 브라우저 정보
  viewport?: { width: number; height: number }
  connection?: string
  language?: string
}

export interface ErrorTrackerConfig {
  apiEndpoint?: string
  maxErrors: number
  enableConsoleLog: boolean
  enableLocalStorage: boolean
  sendToServer: boolean
  filterErrors?: (error: ErrorEvent) => boolean
}

/**
 * 에러 트래커 클래스
 */
export class ErrorTracker {
  private config: ErrorTrackerConfig
  private errors: ErrorEvent[] = []
  private sessionId: string
  private listeners: Array<(error: ErrorEvent) => void> = []

  constructor(config: Partial<ErrorTrackerConfig> = {}) {
    this.config = {
      maxErrors: 100,
      enableConsoleLog: true,
      enableLocalStorage: true,
      sendToServer: false,
      ...config,
    }

    this.sessionId = this.generateSessionId()
    this.setupGlobalHandlers()
    this.loadStoredErrors()
  }

  /**
   * 에러 추가
   */
  addError(
    level: ErrorLevel,
    message: string,
    options: Partial<Omit<ErrorEvent, 'id' | 'timestamp' | 'level' | 'message' | 'url' | 'userAgent' | 'sessionId'>> = {},
  ): void {
    const error: ErrorEvent = {
      id: this.generateErrorId(),
      timestamp: Date.now(),
      level,
      message,
      url: window.location.href,
      userAgent: navigator.userAgent,
      sessionId: this.sessionId,
      viewport: {
        width: window.innerWidth,
        height: window.innerHeight,
      },
      connection: this.getConnectionType(),
      language: navigator.language,
      memory: this.getMemoryUsage(),
      ...options,
    }

    // 필터링
    if (this.config.filterErrors && !this.config.filterErrors(error)) {
      return
    }

    this.errors.push(error)

    // 최대 에러 수 제한
    if (this.errors.length > this.config.maxErrors) {
      this.errors.shift()
    }

    // 콘솔 출력
    if (this.config.enableConsoleLog) {
      this.logToConsole(error)
    }

    // 로컬 저장소에 저장
    if (this.config.enableLocalStorage) {
      this.saveToLocalStorage()
    }

    // 서버로 전송
    if (this.config.sendToServer && this.config.apiEndpoint) {
      this.sendToServer(error)
    }

    // 리스너들에게 알림
    this.notifyListeners(error)
  }

  /**
   * 리스너 추가
   */
  addListener(listener: (error: ErrorEvent) => void): void {
    this.listeners.push(listener)
  }

  /**
   * 리스너 제거
   */
  removeListener(listener: (error: ErrorEvent) => void): void {
    const index = this.listeners.indexOf(listener)
    if (index > -1) {
      this.listeners.splice(index, 1)
    }
  }

  /**
   * 에러 목록 반환
   */
  getErrors(level?: ErrorLevel): ErrorEvent[] {
    if (level) {
      return this.errors.filter(error => error.level === level)
    }
    return [...this.errors]
  }

  /**
   * 에러 통계
   */
  getStats(): Record<ErrorLevel, number> {
    const stats: Record<ErrorLevel, number> = {
      debug: 0,
      info: 0,
      warn: 0,
      error: 0,
      fatal: 0,
    }

    this.errors.forEach(error => {
      stats[error.level]++
    })

    return stats
  }

  /**
   * 에러 삭제
   */
  clearErrors(): void {
    this.errors = []
    this.saveToLocalStorage()
  }

  /**
   * 특정 에러 삭제
   */
  removeError(id: string): void {
    this.errors = this.errors.filter(error => error.id !== id)
    this.saveToLocalStorage()
  }

  /**
   * 에러 내보내기
   */
  exportErrors(): string {
    return JSON.stringify({
      sessionId: this.sessionId,
      timestamp: Date.now(),
      errors: this.errors,
      stats: this.getStats(),
      metadata: {
        url: window.location.href,
        userAgent: navigator.userAgent,
        timestamp: new Date().toISOString(),
      },
    }, null, 2)
  }

  /**
   * 전역 에러 핸들러 설정
   */
  private setupGlobalHandlers(): void {
    // JavaScript 에러 처리
    window.addEventListener('error', (event) => {
      this.addError('error', event.message, {
        stack: event.error?.stack,
        component: 'global',
        metadata: {
          filename: event.filename,
          lineno: event.lineno,
          colno: event.colno,
        },
      })
    })

    // Promise rejection 처리
    window.addEventListener('unhandledrejection', (event) => {
      this.addError('error', `Unhandled Promise Rejection: ${event.reason}`, {
        stack: event.reason?.stack,
        component: 'promise',
        metadata: {
          reason: event.reason,
        },
      })
    })

    // Vue 에러 처리 (Vue 3)
    if (window.Vue && window.Vue.config) {
      const originalErrorHandler = window.Vue.config.errorHandler
      window.Vue.config.errorHandler = (error: Error, instance: any, info: string) => {
        this.addError('error', error.message, {
          stack: error.stack,
          component: instance?.$options.name || 'unknown',
          action: info,
          metadata: {
            componentData: instance?.$data,
            props: instance?.$props,
          },
        })

        if (originalErrorHandler) {
          originalErrorHandler(error, instance, info)
        }
      }
    }

    // 네트워크 에러 처리
    this.setupNetworkErrorHandling()
  }

  /**
   * 네트워크 에러 처리 설정
   */
  private setupNetworkErrorHandling(): void {
    // Fetch API 에러 처리
    const originalFetch = window.fetch
    window.fetch = async (...args) => {
      try {
        const response = await originalFetch(...args)

        if (!response.ok) {
          this.addError('warn', `HTTP ${response.status}: ${response.statusText}`, {
            component: 'network',
            action: 'fetch',
            metadata: {
              url: args[0],
              status: response.status,
              statusText: response.statusText,
            },
          })
        }

        return response
      } catch (error) {
        this.addError('error', `Network Error: ${error.message}`, {
          stack: error.stack,
          component: 'network',
          action: 'fetch',
          metadata: {
            url: args[0],
            error: error.message,
          },
        })
        throw error
      }
    }

    // XMLHttpRequest 에러 처리
    const originalOpen = XMLHttpRequest.prototype.open
    XMLHttpRequest.prototype.open = function(...args) {
      this.addEventListener('error', () => {
        errorTracker.addError('error', 'XMLHttpRequest Error', {
          component: 'network',
          action: 'xhr',
          metadata: {
            url: args[1],
            method: args[0],
          },
        })
      })

      return originalOpen.apply(this, args)
    }
  }

  /**
   * 콘솔에 로그 출력
   */
  private logToConsole(error: ErrorEvent): void {
    const style = this.getConsoleStyle(error.level)
    const prefix = `[${error.level.toUpperCase()}]`

    console.group(`%c${prefix} ${error.message}`, style)
    console.log('Timestamp:', new Date(error.timestamp).toISOString())
    console.log('Component:', error.component || 'unknown')
    console.log('Action:', error.action || 'unknown')
    console.log('Session ID:', error.sessionId)

    if (error.stack) {
      console.log('Stack:', error.stack)
    }

    if (error.metadata) {
      console.log('Metadata:', error.metadata)
    }

    console.groupEnd()
  }

  /**
   * 콘솔 스타일 반환
   */
  private getConsoleStyle(level: ErrorLevel): string {
    const styles: Record<ErrorLevel, string> = {
      debug: 'color: #718096; font-weight: normal;',
      info: 'color: #4299e1; font-weight: normal;',
      warn: 'color: #ed8936; font-weight: bold;',
      error: 'color: #f56565; font-weight: bold;',
      fatal: 'color: #e53e3e; font-weight: bold; background: #fed7d7; padding: 2px 4px;',
    }
    return styles[level]
  }

  /**
   * 로컬 저장소에 저장
   */
  private saveToLocalStorage(): void {
    try {
      const data = {
        sessionId: this.sessionId,
        errors: this.errors.slice(-50), // 최근 50개만 저장
      }
      localStorage.setItem('error_tracker_data', JSON.stringify(data))
    } catch (error) {
      console.warn('Failed to save errors to localStorage:', error)
    }
  }

  /**
   * 저장된 에러 로드
   */
  private loadStoredErrors(): void {
    try {
      const data = localStorage.getItem('error_tracker_data')
      if (data) {
        const parsed = JSON.parse(data)
        if (parsed.sessionId === this.sessionId) {
          this.errors = parsed.errors || []
        }
      }
    } catch (error) {
      console.warn('Failed to load errors from localStorage:', error)
    }
  }

  /**
   * 서버로 에러 전송
   */
  private async sendToServer(error: ErrorEvent): Promise<void> {
    if (!this.config.apiEndpoint) return

    try {
      await fetch(this.config.apiEndpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(error),
      })
    } catch (sendError) {
      console.warn('Failed to send error to server:', sendError)
    }
  }

  /**
   * 리스너들에게 알림
   */
  private notifyListeners(error: ErrorEvent): void {
    this.listeners.forEach(listener => {
      try {
        listener(error)
      } catch (listenerError) {
        console.warn('Error in error tracker listener:', listenerError)
      }
    })
  }

  /**
   * 세션 ID 생성
   */
  private generateSessionId(): string {
    return `session_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
  }

  /**
   * 에러 ID 생성
   */
  private generateErrorId(): string {
    return `error_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
  }

  /**
   * 연결 타입 반환
   */
  private getConnectionType(): string {
    if ('connection' in navigator) {
      const connection = (navigator as any).connection
      return connection.effectiveType || connection.type || 'unknown'
    }
    return 'unknown'
  }

  /**
   * 메모리 사용량 반환
   */
  private getMemoryUsage(): number | undefined {
    if ('memory' in performance) {
      const memory = (performance as any).memory
      return memory.usedJSHeapSize / (1024 * 1024) // MB 단위
    }
    return undefined
  }
}

/**
 * 전역 에러 트래커 인스턴스
 */
export const errorTracker = new ErrorTracker({
  enableConsoleLog: process.env.NODE_ENV === 'development',
  enableLocalStorage: true,
  sendToServer: process.env.NODE_ENV === 'production',
  apiEndpoint: '/api/errors',
})

/**
 * 로거 인터페이스
 */
export const logger = {
  debug: (message: string, metadata?: any) => {
    errorTracker.addError('debug', message, { metadata })
  },

  info: (message: string, metadata?: any) => {
    errorTracker.addError('info', message, { metadata })
  },

  warn: (message: string, metadata?: any) => {
    errorTracker.addError('warn', message, { metadata })
  },

  error: (message: string, error?: Error, metadata?: any) => {
    errorTracker.addError('error', message, {
      stack: error?.stack,
      metadata: {
        ...metadata,
        originalError: error?.message,
      },
    })
  },

  fatal: (message: string, error?: Error, metadata?: any) => {
    errorTracker.addError('fatal', message, {
      stack: error?.stack,
      metadata: {
        ...metadata,
        originalError: error?.message,
      },
    })
  },
}

/**
 * Vue 에러 핸들러 플러그인
 */
export const errorTrackingPlugin = {
  install(app: any) {
    app.config.errorHandler = (error: Error, instance: any, info: string) => {
      errorTracker.addError('error', error.message, {
        stack: error.stack,
        component: instance?.$?.type?.name || 'unknown',
        action: info,
        metadata: {
          componentProps: instance?.$?.props,
          componentData: instance?.$?.data,
        },
      })
    }

    // 전역 속성으로 logger 제공
    app.config.globalProperties.$logger = logger
    app.provide('logger', logger)
  },
}

// 글로벌 타입 확장
declare global {
  interface Window {
    Vue: any
    errorTracker: ErrorTracker
  }
}

// 전역 객체에 에러 트래커 추가 (디버깅용)
window.errorTracker = errorTracker