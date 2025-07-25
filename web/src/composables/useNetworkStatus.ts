import { ref, computed, onMounted, onUnmounted } from 'vue'
import { getNetworkStatus, cancelAllPendingRequests } from '@/api'

export interface NetworkStatus {
  isOnline: boolean
  isSlowConnection: boolean
  downlink?: number
  effectiveType?: string
  pendingRequests: number
  cachedResponses: number
  lastOnlineTime?: Date
  lastOfflineTime?: Date
}

export function useNetworkStatus() {
  const isOnline = ref(navigator.onLine)
  const isSlowConnection = ref(false)
  const downlink = ref<number | undefined>(undefined)
  const effectiveType = ref<string | undefined>(undefined)
  const pendingRequests = ref(0)
  const cachedResponses = ref(0)
  const lastOnlineTime = ref<Date | undefined>(undefined)
  const lastOfflineTime = ref<Date | undefined>(undefined)

  // 연결 정보 감지
  const updateConnectionInfo = () => {
    const connection = (navigator as any).connection || 
                     (navigator as any).mozConnection || 
                     (navigator as any).webkitConnection

    if (connection) {
      downlink.value = connection.downlink
      effectiveType.value = connection.effectiveType
      isSlowConnection.value = connection.effectiveType === 'slow-2g' || 
                              connection.effectiveType === '2g' ||
                              (connection.downlink && connection.downlink < 0.5)
    }

    // API 상태 업데이트
    const apiStatus = getNetworkStatus()
    pendingRequests.value = apiStatus.pendingRequests
    cachedResponses.value = apiStatus.cachedResponses
  }

  // 온라인 상태 변경 핸들러
  const handleOnline = () => {
    isOnline.value = true
    lastOnlineTime.value = new Date()
    updateConnectionInfo()
    
    console.log('🌐 네트워크 연결됨')
    
    // 필요시 대기 중인 요청들을 다시 시도할 수 있음
    // (현재는 API 클라이언트에서 자동 처리됨)
  }

  // 오프라인 상태 변경 핸들러
  const handleOffline = () => {
    isOnline.value = false
    lastOfflineTime.value = new Date()
    
    console.log('🔌 네트워크 연결 끊어짐')
    
    // 선택적으로 대기 중인 요청들 취소
    // cancelAllPendingRequests()
  }

  // 연결 정보 변경 핸들러
  const handleConnectionChange = () => {
    updateConnectionInfo()
    
    if (isSlowConnection.value) {
      console.warn('🐌 느린 네트워크 연결 감지됨')
    }
  }

  // 상태 리프레시
  const refreshStatus = () => {
    isOnline.value = navigator.onLine
    updateConnectionInfo()
  }

  // 네트워크 품질 테스트
  const testNetworkQuality = async (): Promise<{
    latency: number
    downloadSpeed: number
    isGoodConnection: boolean
  }> => {
    try {
      const startTime = performance.now()
      
      // 작은 이미지를 다운로드하여 네트워크 품질 측정
      const testUrl = `${import.meta.env.VITE_API_BASE_URL}/health?t=${Date.now()}`
      
      const response = await fetch(testUrl, {
        method: 'HEAD',
        cache: 'no-cache'
      })
      
      const endTime = performance.now()
      const latency = endTime - startTime
      
      // 간단한 다운로드 속도 측정 (정확하지 않음)
      const downloadSpeed = response.headers.get('content-length') ? 
        parseInt(response.headers.get('content-length')!) / (latency / 1000) / 1024 : 0
      
      const isGoodConnection = latency < 500 && downloadSpeed > 100 // 500ms 미만, 100KB/s 이상
      
      return {
        latency,
        downloadSpeed,
        isGoodConnection
      }
    } catch (error) {
      console.error('Network quality test failed:', error)
      return {
        latency: Infinity,
        downloadSpeed: 0,
        isGoodConnection: false
      }
    }
  }

  // 연결 복구 대기
  const waitForConnection = (): Promise<void> => {
    return new Promise((resolve) => {
      if (isOnline.value) {
        resolve()
        return
      }

      const handleReconnect = () => {
        if (navigator.onLine) {
          window.removeEventListener('online', handleReconnect)
          resolve()
        }
      }

      window.addEventListener('online', handleReconnect)
    })
  }

  // 수동 재연결 시도
  const forceReconnect = async (): Promise<boolean> => {
    try {
      // 간단한 핑 테스트
      const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/health`, {
        method: 'HEAD',
        cache: 'no-cache',
        signal: AbortSignal.timeout(5000) // 5초 타임아웃
      })
      
      if (response.ok) {
        isOnline.value = true
        lastOnlineTime.value = new Date()
        updateConnectionInfo()
        return true
      }
      
      return false
    } catch (error) {
      console.error('Reconnection failed:', error)
      return false
    }
  }

  // 라이프사이클 관리
  onMounted(() => {
    // 초기 상태 설정
    refreshStatus()

    // 이벤트 리스너 등록
    window.addEventListener('online', handleOnline)
    window.addEventListener('offline', handleOffline)

    // 연결 정보 변경 감지 (지원되는 브라우저에서만)
    const connection = (navigator as any).connection || 
                     (navigator as any).mozConnection || 
                     (navigator as any).webkitConnection

    if (connection) {
      connection.addEventListener('change', handleConnectionChange)
    }

    // 주기적으로 상태 업데이트 (30초마다)
    const interval = setInterval(updateConnectionInfo, 30000)

    onUnmounted(() => {
      window.removeEventListener('online', handleOnline)
      window.removeEventListener('offline', handleOffline)
      
      if (connection) {
        connection.removeEventListener('change', handleConnectionChange)
      }
      
      clearInterval(interval)
    })
  })

  return {
    // 상태
    isOnline,
    isSlowConnection,
    downlink,
    effectiveType,
    pendingRequests,
    cachedResponses,
    lastOnlineTime,
    lastOfflineTime,
    
    // 계산된 속성
    networkStatus: computed((): NetworkStatus => ({
      isOnline: isOnline.value,
      isSlowConnection: isSlowConnection.value,
      downlink: downlink.value,
      effectiveType: effectiveType.value,
      pendingRequests: pendingRequests.value,
      cachedResponses: cachedResponses.value,
      lastOnlineTime: lastOnlineTime.value,
      lastOfflineTime: lastOfflineTime.value,
    })),
    
    // 메서드
    refreshStatus,
    testNetworkQuality,
    waitForConnection,
    forceReconnect,
  }
}

// 전역 네트워크 상태 인스턴스 (선택사항)
let globalNetworkStatus: ReturnType<typeof useNetworkStatus> | null = null

export function useGlobalNetworkStatus() {
  if (!globalNetworkStatus) {
    globalNetworkStatus = useNetworkStatus()
  }
  return globalNetworkStatus
}