import { nextTick, ref } from 'vue'

export type AriaLivePoliteness = 'off' | 'polite' | 'assertive';

export interface AriaLiveOptions {
  politeness?: AriaLivePoliteness;
  atomic?: boolean; // 전체 내용을 한 번에 읽을지 여부
  relevant?: 'additions' | 'removals' | 'text' | 'all'; // 어떤 변경사항을 알릴지
  busy?: boolean; // 업데이트 중인지 여부
  delay?: number; // 메시지 지연 시간 (ms)
}

class AriaLiveRegionManager {
  private regions = new Map<string, HTMLElement>()
  private messageQueue = new Map<string, string[]>()
  private timers = new Map<string, NodeJS.Timeout>()

  // 라이브 리전 생성 또는 가져오기
  private getOrCreateRegion(
    id: string,
    politeness: AriaLivePoliteness,
    options: AriaLiveOptions,
  ): HTMLElement {
    if (this.regions.has(id)) {
      return this.regions.get(id)!
    }

    // 라이브 리전 요소 생성
    const region = document.createElement('div')
    region.id = `aria-live-${id}`
    region.setAttribute('aria-live', politeness)

    if (options.atomic !== undefined) {
      region.setAttribute('aria-atomic', String(options.atomic))
    }

    if (options.relevant) {
      region.setAttribute('aria-relevant', options.relevant)
    }

    if (options.busy !== undefined) {
      region.setAttribute('aria-busy', String(options.busy))
    }

    // 스크린 리더 전용 스타일
    region.style.position = 'absolute'
    region.style.left = '-10000px'
    region.style.width = '1px'
    region.style.height = '1px'
    region.style.overflow = 'hidden'
    region.style.clipPath = 'inset(50%)'
    region.style.whiteSpace = 'nowrap'

    // DOM에 추가
    document.body.appendChild(region)

    this.regions.set(id, region)
    this.messageQueue.set(id, [])

    return region
  }

  // 메시지 알리기
  announce(
    message: string,
    id = 'default',
    options: AriaLiveOptions = {},
  ): void {
    const {
      politeness = 'polite',
      delay = 0,
    } = options

    if (!message.trim()) return

    const region = this.getOrCreateRegion(id, politeness, options)
    const queue = this.messageQueue.get(id)!

    // 기존 타이머 클리어
    if (this.timers.has(id)) {
      clearTimeout(this.timers.get(id)!)
    }

    // 메시지를 큐에 추가
    queue.push(message)

    // 지연 시간 후 메시지 표시
    const timer = setTimeout(() => {
      this.processQueue(id)
    }, delay)

    this.timers.set(id, timer)
  }

  // 큐에 있는 메시지 처리
  private async processQueue(id: string): Promise<void> {
    const region = this.regions.get(id)
    const queue = this.messageQueue.get(id)

    if (!region || !queue || queue.length === 0) return

    // 첫 번째 메시지 가져오기
    const message = queue.shift()!

    // 기존 내용 클리어 (스크린 리더가 새 내용을 읽도록)
    region.textContent = ''

    await nextTick()

    // 새 메시지 설정
    region.textContent = message

    // 남은 메시지가 있으면 계속 처리
    if (queue.length > 0) {
      setTimeout(() => {
        this.processQueue(id)
      }, 1000) // 메시지 간 간격
    }
  }

  // 특정 리전의 메시지 큐 클리어
  clear(id = 'default'): void {
    const queue = this.messageQueue.get(id)
    if (queue) {
      queue.length = 0
    }

    const region = this.regions.get(id)
    if (region) {
      region.textContent = ''
    }

    const timer = this.timers.get(id)
    if (timer) {
      clearTimeout(timer)
      this.timers.delete(id)
    }
  }

  // 모든 리전 클리어
  clearAll(): void {
    for (const id of this.regions.keys()) {
      this.clear(id)
    }
  }

  // 리전 제거
  removeRegion(id: string): void {
    this.clear(id)

    const region = this.regions.get(id)
    if (region && region.parentNode) {
      region.parentNode.removeChild(region)
    }

    this.regions.delete(id)
    this.messageQueue.delete(id)
  }

  // 모든 리전 제거
  cleanup(): void {
    for (const id of this.regions.keys()) {
      this.removeRegion(id)
    }
  }

  // 리전의 busy 상태 설정
  setBusy(busy: boolean, id = 'default'): void {
    const region = this.regions.get(id)
    if (region) {
      region.setAttribute('aria-busy', String(busy))
    }
  }
}

// 전역 매니저 인스턴스
const globalManager = new AriaLiveRegionManager()

// 브라우저 환경에서만 cleanup 등록
if (typeof window !== 'undefined') {
  window.addEventListener('beforeunload', () => {
    globalManager.cleanup()
  })
}

export function useAriaLive(
  regionId = 'default',
  defaultOptions: AriaLiveOptions = {},
) {
  const isAnnouncing = ref(false)

  // 메시지 알리기
  const announce = (message: string, options: AriaLiveOptions = {}): void => {
    const mergedOptions = { ...defaultOptions, ...options }

    isAnnouncing.value = true
    globalManager.announce(message, regionId, mergedOptions)

    // 일정 시간 후 상태 리셋
    setTimeout(() => {
      isAnnouncing.value = false
    }, mergedOptions.delay || 100)
  }

  // 성공 메시지
  const announceSuccess = (message: string): void => {
    announce(message, { politeness: 'polite' })
  }

  // 오류 메시지
  const announceError = (message: string): void => {
    announce(message, { politeness: 'assertive' })
  }

  // 경고 메시지
  const announceWarning = (message: string): void => {
    announce(message, { politeness: 'assertive' })
  }

  // 정보 메시지
  const announceInfo = (message: string): void => {
    announce(message, { politeness: 'polite' })
  }

  // 로딩 상태 알리기
  const announceLoading = (message = '로딩 중입니다'): void => {
    globalManager.setBusy(true, regionId)
    announce(message, { politeness: 'polite' })
  }

  // 로딩 완료 알리기
  const announceLoadingComplete = (message = '로딩이 완료되었습니다'): void => {
    globalManager.setBusy(false, regionId)
    announce(message, { politeness: 'polite' })
  }

  // 메시지 큐 클리어
  const clear = (): void => {
    globalManager.clear(regionId)
    isAnnouncing.value = false
  }

  // 리전 제거
  const cleanup = (): void => {
    globalManager.removeRegion(regionId)
    isAnnouncing.value = false
  }

  return {
    // 상태
    isAnnouncing,

    // 기본 메서드
    announce,
    clear,
    cleanup,

    // 편의 메서드
    announceSuccess,
    announceError,
    announceWarning,
    announceInfo,
    announceLoading,
    announceLoadingComplete,

    // 매니저 접근
    manager: globalManager,
  }
}

// 전역 유틸리티 함수들
export const ariaLive = {
  announce: (message: string, options?: AriaLiveOptions) =>
    globalManager.announce(message, 'default', options),

  success: (message: string) =>
    globalManager.announce(message, 'default', { politeness: 'polite' }),

  error: (message: string) =>
    globalManager.announce(message, 'default', { politeness: 'assertive' }),

  warning: (message: string) =>
    globalManager.announce(message, 'default', { politeness: 'assertive' }),

  clear: () => globalManager.clearAll(),

  cleanup: () => globalManager.cleanup(),
}