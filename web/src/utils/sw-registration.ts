import { Workbox } from 'workbox-window'

export interface ServiceWorkerRegistrationOptions {
  onUpdate?: (registration: ServiceWorkerRegistration) => void
  onControllerChange?: (registration: ServiceWorkerRegistration) => void
  onInstalled?: (registration: ServiceWorkerRegistration) => void
  onError?: (error: Error) => void
}

export function registerServiceWorker(options: ServiceWorkerRegistrationOptions = {}) {
  if ('serviceWorker' in navigator) {
    const wb = new Workbox('/sw.js')

    // Service Worker 설치 완료 이벤트
    wb.addEventListener('installed', (event) => {
      console.log('🔧 Service Worker가 설치되었습니다')
      if (event.isUpdate) {
        console.log('📝 Service Worker가 업데이트되었습니다')
        options.onUpdate?.(event.sw as any)
      } else {
        console.log('✨ Service Worker가 처음 설치되었습니다')
        options.onInstalled?.(event.sw as any)
      }
    })

    // Service Worker 제어권 변경 이벤트
    wb.addEventListener('controlling', (event) => {
      console.log('🎮 Service Worker가 제어권을 갖게 되었습니다')
      options.onControllerChange?.(event.sw as any)
      
      // 페이지 새로고침 여부를 사용자에게 물어볼 수 있음
      if (event.isUpdate) {
        // 자동 새로고침 또는 사용자 선택 UI 표시
        console.log('🔄 앱이 업데이트되었습니다. 새로고침을 권장합니다.')
      }
    })

    // Service Worker 대기 상태 이벤트
    wb.addEventListener('waiting', (event) => {
      console.log('⏳ 새로운 Service Worker가 대기 중입니다')
      
      // 사용자에게 업데이트 알림을 표시할 수 있음
      showUpdateAvailable(event.sw as any)
    })

    // Service Worker 활성화 이벤트
    wb.addEventListener('activated', (event) => {
      console.log('🟢 Service Worker가 활성화되었습니다')
    })

    // 오프라인 대비 이벤트
    wb.addEventListener('message', (event) => {
      console.log('📨 Service Worker로부터 메시지:', event.data)
      
      if (event.data && event.data.type === 'CACHE_UPDATED') {
        console.log('📦 캐시가 업데이트되었습니다:', event.data.payload)
      }
    })

    // Service Worker 등록 및 시작
    wb.register()
      .then((registration) => {
        console.log('✅ Service Worker 등록 성공:', registration)
        
        // 주기적으로 업데이트 확인 (24시간마다)
        setInterval(() => {
          registration.update()
        }, 24 * 60 * 60 * 1000)
      })
      .catch((error) => {
        console.error('❌ Service Worker 등록 실패:', error)
        options.onError?.(error)
      })

    return wb
  } else {
    console.warn('⚠️ 이 브라우저는 Service Worker를 지원하지 않습니다')
    return null
  }
}

// 업데이트 알림 표시
function showUpdateAvailable(worker: ServiceWorker) {
  // 사용자에게 업데이트 알림을 표시
  // 이는 UI 컴포넌트에서 처리하거나 전역 이벤트로 발송할 수 있음
  console.log('📢 앱 업데이트가 준비되었습니다')
  
  // 커스텀 이벤트 발송
  window.dispatchEvent(new CustomEvent('sw-update-available', {
    detail: { worker }
  }))
}

// Service Worker 메시지 전송 유틸리티
export function sendMessageToSW(message: any): Promise<any> {
  return new Promise((resolve, reject) => {
    if (!navigator.serviceWorker.controller) {
      reject(new Error('Service Worker가 활성화되지 않았습니다'))
      return
    }

    const messageChannel = new MessageChannel()
    
    messageChannel.port1.onmessage = (event) => {
      if (event.data.error) {
        reject(event.data.error)
      } else {
        resolve(event.data)
      }
    }

    navigator.serviceWorker.controller.postMessage(message, [messageChannel.port2])
  })
}

// 캐시 클리어 유틸리티
export async function clearServiceWorkerCache(cacheName?: string): Promise<void> {
  try {
    if (cacheName) {
      const cache = await caches.open(cacheName)
      const requests = await cache.keys()
      await Promise.all(requests.map(request => cache.delete(request)))
      console.log(`🗑️ 캐시 '${cacheName}'이 삭제되었습니다`)
    } else {
      const cacheNames = await caches.keys()
      await Promise.all(cacheNames.map(name => caches.delete(name)))
      console.log('🗑️ 모든 캐시가 삭제되었습니다')
    }
  } catch (error) {
    console.error('❌ 캐시 삭제 실패:', error)
    throw error
  }
}

// Service Worker 업데이트 강제 적용
export async function skipWaiting(): Promise<void> {
  try {
    await sendMessageToSW({ type: 'SKIP_WAITING' })
    console.log('⏭️ Service Worker 업데이트를 강제 적용했습니다')
  } catch (error) {
    console.error('❌ Service Worker 업데이트 강제 적용 실패:', error)
    throw error
  }
}

// PWA 업데이트 관리자
export class PWAUpdateManager {
  private updateAvailable = false
  private waitingWorker: ServiceWorker | null = null

  constructor() {
    // 업데이트 이벤트 리스너 등록
    window.addEventListener('sw-update-available', (event: any) => {
      this.updateAvailable = true
      this.waitingWorker = event.detail.worker
      this.notifyUpdateAvailable()
    })
  }

  private notifyUpdateAvailable() {
    // 사용자에게 업데이트 알림
    console.log('🔔 앱 업데이트가 준비되었습니다')
    
    // UI에서 업데이트 프롬프트 표시
    // 예: 토스트 메시지, 배너, 모달 등
  }

  async applyUpdate(): Promise<void> {
    if (!this.waitingWorker) {
      throw new Error('대기 중인 Service Worker가 없습니다')
    }

    try {
      await this.waitingWorker.postMessage({ type: 'SKIP_WAITING' })
      window.location.reload()
    } catch (error) {
      console.error('❌ 업데이트 적용 실패:', error)
      throw error
    }
  }

  dismissUpdate(): void {
    this.updateAvailable = false
    this.waitingWorker = null
  }

  get hasUpdateAvailable(): boolean {
    return this.updateAvailable
  }
}

// 전역 업데이트 매니저 인스턴스
export const pwaUpdateManager = new PWAUpdateManager()