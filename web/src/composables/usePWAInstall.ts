import { ref, computed, onMounted, onUnmounted } from 'vue'

export interface PWAInstallPromptEvent extends Event {
  readonly platforms: string[]
  readonly userChoice: Promise<{
    outcome: 'accepted' | 'dismissed'
    platform: string
  }>
  prompt(): Promise<void>
}

export function usePWAInstall() {
  const deferredPrompt = ref<PWAInstallPromptEvent | null>(null)
  const isInstallable = ref(false)
  const isInstalled = ref(false)
  const isIOS = ref(false)
  const isStandalone = ref(false)
  const installPromptShown = ref(false)
  const userDismissedCount = ref(0)

  // 로컬 스토리지 키
  const STORAGE_KEYS = {
    DISMISSED_COUNT: 'pwa-install-dismissed-count',
    LAST_PROMPT: 'pwa-install-last-prompt',
    USER_REJECTED: 'pwa-install-user-rejected',
  }

  // iOS 감지
  const detectIOS = () => {
    return /iPad|iPhone|iPod/.test(navigator.userAgent) && !(window as any).MSStream
  }

  // 단독 모드 감지 (이미 설치된 PWA에서 실행 중인지)
  const detectStandalone = () => {
    return window.matchMedia('(display-mode: standalone)').matches ||
           (window.navigator as any).standalone === true
  }

  // 설치 가능 여부 확인
  const checkInstallability = computed(() => {
    if (isStandalone.value) return false // 이미 설치됨
    if (isIOS.value) return true // iOS는 수동 설치만 가능
    return isInstallable.value && deferredPrompt.value !== null
  })

  // 사용자가 설치를 거부했는지 확인
  const isUserRejected = computed(() => {
    const rejected = localStorage.getItem(STORAGE_KEYS.USER_REJECTED)
    return rejected === 'true'
  })

  // 프롬프트를 표시해야 하는지 확인
  const shouldShowPrompt = computed(() => {
    if (isStandalone.value || isUserRejected.value) return false
    
    const dismissedCount = parseInt(localStorage.getItem(STORAGE_KEYS.DISMISSED_COUNT) || '0')
    const lastPrompt = localStorage.getItem(STORAGE_KEYS.LAST_PROMPT)
    
    // 3번 이상 무시했으면 더 이상 표시하지 않음
    if (dismissedCount >= 3) return false
    
    // 마지막 프롬프트로부터 24시간이 지나지 않았으면 표시하지 않음
    if (lastPrompt) {
      const lastPromptTime = new Date(lastPrompt).getTime()
      const now = new Date().getTime()
      const hoursDiff = (now - lastPromptTime) / (1000 * 60 * 60)
      if (hoursDiff < 24) return false
    }
    
    return checkInstallability.value
  })

  // PWA 설치 프롬프트 표시
  const showInstallPrompt = async (): Promise<boolean> => {
    if (!checkInstallability.value) {
      console.warn('PWA 설치가 불가능한 상태입니다.')
      return false
    }

    try {
      if (isIOS.value) {
        // iOS는 수동 설치 가이드만 표시
        return true
      }

      if (deferredPrompt.value) {
        // beforeinstallprompt 이벤트가 있는 경우
        await deferredPrompt.value.prompt()
        const choiceResult = await deferredPrompt.value.userChoice
        
        console.log('PWA 설치 선택:', choiceResult.outcome)
        
        if (choiceResult.outcome === 'accepted') {
          console.log('사용자가 PWA 설치를 수락했습니다.')
          isInstalled.value = true
          return true
        } else {
          console.log('사용자가 PWA 설치를 거부했습니다.')
          incrementDismissedCount()
          return false
        }
      }
    } catch (error) {
      console.error('PWA 설치 프롬프트 오류:', error)
    }

    return false
  }

  // 수동 설치 가이드 표시 (주로 iOS용)
  const showManualInstallGuide = () => {
    return {
      isIOS: isIOS.value,
      steps: isIOS.value
        ? [
            '하단의 공유 버튼 (↗️)을 탭하세요',
            '"홈 화면에 추가" 옵션을 찾아 탭하세요',
            '"추가" 버튼을 탭하여 설치를 완료하세요',
          ]
        : [
            '브라우저 메뉴에서 "앱 설치" 또는 "홈 화면에 추가"를 찾으세요',
            '설치 버튼을 클릭하세요',
            '확인 대화상자에서 "설치"를 클릭하세요',
          ]
    }
  }

  // 거부 횟수 증가
  const incrementDismissedCount = () => {
    const count = parseInt(localStorage.getItem(STORAGE_KEYS.DISMISSED_COUNT) || '0')
    localStorage.setItem(STORAGE_KEYS.DISMISSED_COUNT, (count + 1).toString())
    localStorage.setItem(STORAGE_KEYS.LAST_PROMPT, new Date().toISOString())
    userDismissedCount.value = count + 1
  }

  // 사용자가 영구적으로 거부
  const markAsUserRejected = () => {
    localStorage.setItem(STORAGE_KEYS.USER_REJECTED, 'true')
  }

  // 설치 상태 초기화 (개발/테스트용)
  const resetInstallState = () => {
    localStorage.removeItem(STORAGE_KEYS.DISMISSED_COUNT)
    localStorage.removeItem(STORAGE_KEYS.LAST_PROMPT)
    localStorage.removeItem(STORAGE_KEYS.USER_REJECTED)
    userDismissedCount.value = 0
    installPromptShown.value = false
  }

  // PWA 업데이트 확인
  const checkForUpdates = async () => {
    if ('serviceWorker' in navigator) {
      try {
        const registration = await navigator.serviceWorker.getRegistration()
        if (registration) {
          await registration.update()
          console.log('PWA 업데이트 확인 완료')
        }
      } catch (error) {
        console.error('PWA 업데이트 확인 실패:', error)
      }
    }
  }

  // 설치 통계 수집
  const getInstallStats = () => {
    return {
      isInstallable: checkInstallability.value,
      isInstalled: isStandalone.value,
      isIOS: isIOS.value,
      dismissedCount: userDismissedCount.value,
      shouldShowPrompt: shouldShowPrompt.value,
      lastPrompt: localStorage.getItem(STORAGE_KEYS.LAST_PROMPT),
    }
  }

  // beforeinstallprompt 이벤트 핸들러
  const handleBeforeInstallPrompt = (e: Event) => {
    console.log('beforeinstallprompt 이벤트 감지됨')
    e.preventDefault()
    deferredPrompt.value = e as PWAInstallPromptEvent
    isInstallable.value = true
  }

  // 앱 설치 완료 이벤트 핸들러
  const handleAppInstalled = () => {
    console.log('PWA가 설치되었습니다')
    isInstalled.value = true
    isInstallable.value = false
    deferredPrompt.value = null
  }

  // 생명주기 관리
  onMounted(() => {
    // 플랫폼 감지
    isIOS.value = detectIOS()
    isStandalone.value = detectStandalone()
    
    // 로컬 스토리지에서 상태 복원
    userDismissedCount.value = parseInt(localStorage.getItem(STORAGE_KEYS.DISMISSED_COUNT) || '0')

    // 이벤트 리스너 등록
    window.addEventListener('beforeinstallprompt', handleBeforeInstallPrompt)
    window.addEventListener('appinstalled', handleAppInstalled)

    // 서비스 워커 상태 확인
    if ('serviceWorker' in navigator) {
      navigator.serviceWorker.ready.then(() => {
        console.log('Service Worker가 준비되었습니다')
      })
    }

    // 초기 설치 상태 확인
    if (isStandalone.value) {
      console.log('PWA가 이미 설치되어 실행 중입니다')
      isInstalled.value = true
    }
  })

  onUnmounted(() => {
    window.removeEventListener('beforeinstallprompt', handleBeforeInstallPrompt)
    window.removeEventListener('appinstalled', handleAppInstalled)
  })

  return {
    // 상태
    isInstallable: checkInstallability,
    isInstalled,
    isIOS,
    isStandalone,
    shouldShowPrompt,
    isUserRejected,
    userDismissedCount,

    // 메서드
    showInstallPrompt,
    showManualInstallGuide,
    incrementDismissedCount,
    markAsUserRejected,
    resetInstallState,
    checkForUpdates,
    getInstallStats,
  }
}

// 전역 PWA 설치 인스턴스
let globalPWAInstall: ReturnType<typeof usePWAInstall> | null = null

export function useGlobalPWAInstall() {
  if (!globalPWAInstall) {
    globalPWAInstall = usePWAInstall()
  }
  return globalPWAInstall
}