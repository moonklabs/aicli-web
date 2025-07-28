/**
 * 사용자 분석 및 행동 추적 시스템
 */

// 이벤트 타입 정의
export type AnalyticsEventType =
  | 'page_view'
  | 'user_interaction'
  | 'feature_usage'
  | 'error_occurrence'
  | 'performance_metric'
  | 'conversion'
  | 'custom'

// 사용자 속성 인터페이스
export interface UserProperties {
  userId?: string
  sessionId: string
  deviceType: 'desktop' | 'mobile' | 'tablet'
  browser: string
  os: string
  language: string
  timezone: string
  screenResolution: string
  colorDepth: number
  cookieEnabled: boolean
  javaEnabled: boolean
}

// 분석 이벤트 인터페이스
export interface AnalyticsEvent {
  eventType: AnalyticsEventType
  eventName: string
  timestamp: number
  sessionId: string
  userId?: string

  // 이벤트 데이터
  properties?: Record<string, any>

  // 페이지 정보
  page?: {
    url: string
    title: string
    referrer: string
    path: string
  }

  // 사용자 정보
  user?: Partial<UserProperties>

  // 기술적 정보
  technical?: {
    userAgent: string
    viewport: { width: number; height: number }
    connectionType?: string
    loadTime?: number
    scrollDepth?: number
  }
}

// GA4 설정 인터페이스
export interface GA4Config {
  measurementId: string
  apiSecret?: string
  debugMode?: boolean
  sendPageView?: boolean
  customDimensions?: Record<string, string>
}

// 분석 제공자 인터페이스
export interface AnalyticsProvider {
  name: string
  initialize(config?: any): Promise<void>
  track(event: AnalyticsEvent): Promise<void>
  identify(userId: string, properties?: Record<string, any>): Promise<void>
  setUserProperties(properties: Record<string, any>): Promise<void>
  flush?(): Promise<void>
}

/**
 * Google Analytics 4 제공자
 */
export class GA4Provider implements AnalyticsProvider {
  name = 'ga4'
  private config: GA4Config
  private isInitialized = false

  constructor(config: GA4Config) {
    this.config = config
  }

  async initialize(): Promise<void> {
    if (this.isInitialized) return

    try {
      // GA4 gtag 라이브러리 로드
      await this.loadGtagScript()

      // GA4 초기화
      window.gtag('config', this.config.measurementId, {
        debug_mode: this.config.debugMode || false,
        send_page_view: this.config.sendPageView !== false,
        custom_map: this.config.customDimensions || {},
      })

      this.isInitialized = true
      console.log('GA4 Provider initialized')
    } catch (error) {
      console.error('Failed to initialize GA4:', error)
      throw error
    }
  }

  async track(event: AnalyticsEvent): Promise<void> {
    if (!this.isInitialized) return

    try {
      const eventData: any = {
        event_category: event.eventType,
        event_label: event.eventName,
        session_id: event.sessionId,
        ...event.properties,
      }

      // 페이지 정보 추가
      if (event.page) {
        eventData.page_title = event.page.title
        eventData.page_location = event.page.url
        eventData.page_referrer = event.page.referrer
      }

      // 사용자 정보 추가
      if (event.user) {
        Object.assign(eventData, event.user)
      }

      // 기술적 정보 추가
      if (event.technical) {
        eventData.screen_resolution = `${event.technical.viewport.width}x${event.technical.viewport.height}`
        eventData.connection_type = event.technical.connectionType
        eventData.load_time = event.technical.loadTime
        eventData.scroll_depth = event.technical.scrollDepth
      }

      window.gtag('event', event.eventName, eventData)
    } catch (error) {
      console.error('Failed to track GA4 event:', error)
    }
  }

  async identify(userId: string, properties?: Record<string, any>): Promise<void> {
    if (!this.isInitialized) return

    try {
      window.gtag('config', this.config.measurementId, {
        user_id: userId,
        custom_map: properties || {},
      })
    } catch (error) {
      console.error('Failed to identify user in GA4:', error)
    }
  }

  async setUserProperties(properties: Record<string, any>): Promise<void> {
    if (!this.isInitialized) return

    try {
      window.gtag('set', { user_properties: properties })
    } catch (error) {
      console.error('Failed to set user properties in GA4:', error)
    }
  }

  private async loadGtagScript(): Promise<void> {
    return new Promise((resolve, reject) => {
      if (window.gtag) {
        resolve()
        return
      }

      // gtag 함수 정의
      window.dataLayer = window.dataLayer || []
      window.gtag = function() {
        window.dataLayer.push(arguments)
      }
      window.gtag('js', new Date())

      // 스크립트 로드
      const script = document.createElement('script')
      script.async = true
      script.src = `https://www.googletagmanager.com/gtag/js?id=${this.config.measurementId}`
      script.onload = () => resolve()
      script.onerror = () => reject(new Error('Failed to load gtag script'))

      document.head.appendChild(script)
    })
  }
}

/**
 * 자체 분석 제공자 (로컬 수집)
 */
export class LocalAnalyticsProvider implements AnalyticsProvider {
  name = 'local'
  private events: AnalyticsEvent[] = []
  private maxEvents = 1000
  private apiEndpoint?: string

  constructor(config?: { maxEvents?: number; apiEndpoint?: string }) {
    this.maxEvents = config?.maxEvents || 1000
    this.apiEndpoint = config?.apiEndpoint
  }

  async initialize(): Promise<void> {
    // 저장된 이벤트 로드
    this.loadStoredEvents()
    console.log('Local Analytics Provider initialized')
  }

  async track(event: AnalyticsEvent): Promise<void> {
    try {
      this.events.push(event)

      // 최대 이벤트 수 제한
      if (this.events.length > this.maxEvents) {
        this.events.splice(0, this.events.length - this.maxEvents)
      }

      // 로컬 저장소에 저장
      this.saveToLocalStorage()

      // 서버로 전송 (설정된 경우)
      if (this.apiEndpoint) {
        this.sendToServer(event)
      }
    } catch (error) {
      console.error('Failed to track local event:', error)
    }
  }

  async identify(userId: string, properties?: Record<string, any>): Promise<void> {
    await this.track({
      eventType: 'custom',
      eventName: 'user_identified',
      timestamp: Date.now(),
      sessionId: this.generateSessionId(),
      userId,
      properties,
    })
  }

  async setUserProperties(properties: Record<string, any>): Promise<void> {
    await this.track({
      eventType: 'custom',
      eventName: 'user_properties_updated',
      timestamp: Date.now(),
      sessionId: this.generateSessionId(),
      properties,
    })
  }

  getEvents(): AnalyticsEvent[] {
    return [...this.events]
  }

  clearEvents(): void {
    this.events = []
    this.saveToLocalStorage()
  }

  exportEvents(): string {
    return JSON.stringify(this.events, null, 2)
  }

  private loadStoredEvents(): void {
    try {
      const stored = localStorage.getItem('analytics_events')
      if (stored) {
        this.events = JSON.parse(stored)
      }
    } catch (error) {
      console.warn('Failed to load stored events:', error)
    }
  }

  private saveToLocalStorage(): void {
    try {
      localStorage.setItem('analytics_events', JSON.stringify(this.events.slice(-100)))
    } catch (error) {
      console.warn('Failed to save events to localStorage:', error)
    }
  }

  private async sendToServer(event: AnalyticsEvent): Promise<void> {
    if (!this.apiEndpoint) return

    try {
      await fetch(this.apiEndpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(event),
      })
    } catch (error) {
      console.warn('Failed to send event to server:', error)
    }
  }

  private generateSessionId(): string {
    return `session_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
  }
}

/**
 * 메인 분석 클래스
 */
export class Analytics {
  private providers: AnalyticsProvider[] = []
  private sessionId: string
  private userId?: string
  private userProperties: Partial<UserProperties>
  private isInitialized = false

  constructor() {
    this.sessionId = this.generateSessionId()
    this.userProperties = this.collectUserProperties()
  }

  /**
   * 분석 제공자 추가
   */
  addProvider(provider: AnalyticsProvider): void {
    this.providers.push(provider)
  }

  /**
   * 초기화
   */
  async initialize(): Promise<void> {
    if (this.isInitialized) return

    try {
      await Promise.all(this.providers.map(provider => provider.initialize()))
      this.isInitialized = true

      // 초기 페이지 뷰 추적
      this.trackPageView()

      // 스크롤 깊이 추적 설정
      this.setupScrollTracking()

      console.log('Analytics initialized with providers:', this.providers.map(p => p.name))
    } catch (error) {
      console.error('Failed to initialize analytics:', error)
    }
  }

  /**
   * 이벤트 추적
   */
  async track(
    eventType: AnalyticsEventType,
    eventName: string,
    properties?: Record<string, any>,
  ): Promise<void> {
    if (!this.isInitialized) return

    const event: AnalyticsEvent = {
      eventType,
      eventName,
      timestamp: Date.now(),
      sessionId: this.sessionId,
      userId: this.userId,
      properties,
      page: this.getCurrentPageInfo(),
      user: this.userProperties,
      technical: this.getTechnicalInfo(),
    }

    await Promise.all(this.providers.map(provider => provider.track(event)))
  }

  /**
   * 사용자 식별
   */
  async identify(userId: string, properties?: Record<string, any>): Promise<void> {
    this.userId = userId

    if (properties) {
      this.userProperties = { ...this.userProperties, ...properties }
    }

    await Promise.all(this.providers.map(provider => provider.identify(userId, properties)))
  }

  /**
   * 사용자 속성 설정
   */
  async setUserProperties(properties: Record<string, any>): Promise<void> {
    this.userProperties = { ...this.userProperties, ...properties }
    await Promise.all(this.providers.map(provider => provider.setUserProperties(properties)))
  }

  /**
   * 페이지 뷰 추적
   */
  async trackPageView(url?: string, title?: string): Promise<void> {
    await this.track('page_view', 'page_view', {
      page_url: url || window.location.href,
      page_title: title || document.title,
      page_path: window.location.pathname,
    })
  }

  /**
   * 사용자 상호작용 추적
   */
  async trackInteraction(element: string, action: string, properties?: Record<string, any>): Promise<void> {
    await this.track('user_interaction', 'interaction', {
      element,
      action,
      ...properties,
    })
  }

  /**
   * 기능 사용 추적
   */
  async trackFeatureUsage(feature: string, properties?: Record<string, any>): Promise<void> {
    await this.track('feature_usage', 'feature_used', {
      feature_name: feature,
      ...properties,
    })
  }

  /**
   * 에러 추적
   */
  async trackError(error: string, details?: Record<string, any>): Promise<void> {
    await this.track('error_occurrence', 'error', {
      error_message: error,
      ...details,
    })
  }

  /**
   * 성능 메트릭 추적
   */
  async trackPerformance(metric: string, value: number, unit?: string): Promise<void> {
    await this.track('performance_metric', 'performance', {
      metric_name: metric,
      metric_value: value,
      metric_unit: unit || 'ms',
    })
  }

  /**
   * 변환 추적
   */
  async trackConversion(event: string, value?: number, currency?: string): Promise<void> {
    await this.track('conversion', 'conversion', {
      conversion_event: event,
      conversion_value: value,
      currency: currency || 'KRW',
    })
  }

  /**
   * 스크롤 깊이 추적 설정
   */
  private setupScrollTracking(): void {
    let maxScrollDepth = 0
    let scrollTimer: NodeJS.Timeout

    const trackScrollDepth = () => {
      const scrollTop = window.pageYOffset || document.documentElement.scrollTop
      const documentHeight = document.documentElement.scrollHeight - window.innerHeight
      const scrollDepth = Math.round((scrollTop / documentHeight) * 100)

      if (scrollDepth > maxScrollDepth) {
        maxScrollDepth = scrollDepth

        clearTimeout(scrollTimer)
        scrollTimer = setTimeout(() => {
          this.track('user_interaction', 'scroll_depth', {
            scroll_depth: maxScrollDepth,
            page_url: window.location.href,
          })
        }, 1000)
      }
    }

    window.addEventListener('scroll', trackScrollDepth, { passive: true })
  }

  /**
   * 현재 페이지 정보 수집
   */
  private getCurrentPageInfo(): AnalyticsEvent['page'] {
    return {
      url: window.location.href,
      title: document.title,
      referrer: document.referrer,
      path: window.location.pathname,
    }
  }

  /**
   * 기술적 정보 수집
   */
  private getTechnicalInfo(): AnalyticsEvent['technical'] {
    return {
      userAgent: navigator.userAgent,
      viewport: {
        width: window.innerWidth,
        height: window.innerHeight,
      },
      connectionType: this.getConnectionType(),
      loadTime: performance.now(),
    }
  }

  /**
   * 사용자 속성 수집
   */
  private collectUserProperties(): UserProperties {
    return {
      sessionId: this.sessionId,
      deviceType: this.getDeviceType(),
      browser: this.getBrowser(),
      os: this.getOS(),
      language: navigator.language,
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
      screenResolution: `${screen.width}x${screen.height}`,
      colorDepth: screen.colorDepth,
      cookieEnabled: navigator.cookieEnabled,
      javaEnabled: false, // Java는 현대 브라우저에서 지원하지 않음
    }
  }

  /**
   * 세션 ID 생성
   */
  private generateSessionId(): string {
    return `analytics_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
  }

  /**
   * 디바이스 타입 감지
   */
  private getDeviceType(): 'desktop' | 'mobile' | 'tablet' {
    const userAgent = navigator.userAgent.toLowerCase()

    if (/tablet|ipad|playbook|silk/.test(userAgent)) {
      return 'tablet'
    }

    if (/mobile|iphone|ipod|android|blackberry|opera|mini|windows\sce|palm|smartphone|iemobile/.test(userAgent)) {
      return 'mobile'
    }

    return 'desktop'
  }

  /**
   * 브라우저 감지
   */
  private getBrowser(): string {
    const userAgent = navigator.userAgent

    if (userAgent.includes('Chrome')) return 'Chrome'
    if (userAgent.includes('Firefox')) return 'Firefox'
    if (userAgent.includes('Safari')) return 'Safari'
    if (userAgent.includes('Edge')) return 'Edge'
    if (userAgent.includes('Opera')) return 'Opera'

    return 'Unknown'
  }

  /**
   * 운영체제 감지
   */
  private getOS(): string {
    const userAgent = navigator.userAgent

    if (userAgent.includes('Windows')) return 'Windows'
    if (userAgent.includes('Mac')) return 'macOS'
    if (userAgent.includes('Linux')) return 'Linux'
    if (userAgent.includes('Android')) return 'Android'
    if (userAgent.includes('iOS')) return 'iOS'

    return 'Unknown'
  }

  /**
   * 연결 타입 감지
   */
  private getConnectionType(): string {
    if ('connection' in navigator) {
      const connection = (navigator as any).connection
      return connection.effectiveType || connection.type || 'unknown'
    }
    return 'unknown'
  }
}

/**
 * 전역 분석 인스턴스
 */
export const analytics = new Analytics()

// 글로벌 타입 확장
declare global {
  interface Window {
    gtag: (...args: any[]) => void
    dataLayer: any[]
  }
}

/**
 * Vue 플러그인
 */
export const analyticsPlugin = {
  install(app: any, options?: { ga4?: GA4Config; local?: boolean; apiEndpoint?: string }) {
    // GA4 제공자 추가
    if (options?.ga4) {
      analytics.addProvider(new GA4Provider(options.ga4))
    }

    // 로컬 제공자 추가
    if (options?.local !== false) {
      analytics.addProvider(new LocalAnalyticsProvider({
        apiEndpoint: options?.apiEndpoint,
      }))
    }

    // 전역 속성으로 analytics 제공
    app.config.globalProperties.$analytics = analytics
    app.provide('analytics', analytics)

    // 라우터 변경 시 페이지 뷰 추적
    const router = app.config.globalProperties.$router
    if (router) {
      router.afterEach((to: any) => {
        analytics.trackPageView(to.fullPath, to.meta?.title)
      })
    }
  },
}