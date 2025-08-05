import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'

interface OrientationConfig {
  // 자동 적응 설정
  enableAutoRotation?: boolean
  enableLayoutSwitching?: boolean
  enableContentReflow?: boolean
  
  // 애니메이션 설정
  enableTransitions?: boolean
  transitionDuration?: number
  
  // 레이아웃 설정
  portraitPreferences?: PortraitLayoutOptions
  landscapePreferences?: LandscapeLayoutOptions
  
  // 성능 설정
  debounceDelay?: number
  enableOptimizations?: boolean
}

interface PortraitLayoutOptions {
  navigationPosition?: 'top' | 'bottom'
  contentPadding?: number
  enableFullHeight?: boolean
  keyboardBehavior?: 'resize' | 'overlay' | 'scroll'
}

interface LandscapeLayoutOptions {
  navigationPosition?: 'left' | 'right' | 'top'
  enableSidePanel?: boolean
  contentColumns?: number
  compactMode?: boolean
}

interface OrientationState {
  current: 'portrait' | 'landscape'
  previous: 'portrait' | 'landscape' | null
  angle: number
  isChanging: boolean
  aspectRatio: number
  safeArea: {
    top: number
    bottom: number
    left: number
    right: number
  }
}

interface ViewportDimensions {
  width: number
  height: number
  availableWidth: number
  availableHeight: number
  devicePixelRatio: number
}

export function useOrientationAdaptation(config: OrientationConfig = {}) {
  const defaultConfig: Required<OrientationConfig> = {
    enableAutoRotation: true,
    enableLayoutSwitching: true,
    enableContentReflow: true,
    enableTransitions: true,
    transitionDuration: 300,
    portraitPreferences: {
      navigationPosition: 'bottom',
      contentPadding: 16,
      enableFullHeight: true,
      keyboardBehavior: 'resize',
    },
    landscapePreferences: {
      navigationPosition: 'left',
      enableSidePanel: true,
      contentColumns: 2,
      compactMode: false,
    },
    debounceDelay: 100,
    enableOptimizations: true,
  }

  const settings = { ...defaultConfig, ...config }

  // 상태 관리
  const orientationState = ref<OrientationState>({
    current: 'portrait',
    previous: null,
    angle: 0,
    isChanging: false,
    aspectRatio: 1,
    safeArea: {
      top: 0,
      bottom: 0,
      left: 0,
      right: 0,
    },
  })

  const viewport = ref<ViewportDimensions>({
    width: window.innerWidth,
    height: window.innerHeight,
    availableWidth: window.innerWidth,
    availableHeight: window.innerHeight,
    devicePixelRatio: window.devicePixelRatio,
  })

  // 디바운스 타이머
  let orientationTimer: number | null = null
  let transitionTimer: number | null = null

  // 계산된 속성들
  const isPortrait = computed(() => orientationState.value.current === 'portrait')
  const isLandscape = computed(() => orientationState.value.current === 'landscape')
  
  const isNarrowPortrait = computed(() => 
    isPortrait.value && viewport.value.width < 375
  )
  
  const isWidePortrait = computed(() => 
    isPortrait.value && viewport.value.width >= 414
  )
  
  const isCompactLandscape = computed(() => 
    isLandscape.value && viewport.value.height < 500
  )
  
  const isWideLandscape = computed(() => 
    isLandscape.value && viewport.value.width >= 834
  )

  // 현재 방향에 맞는 레이아웃 설정
  const currentLayoutOptions = computed(() => {
    return isPortrait.value 
      ? settings.portraitPreferences 
      : settings.landscapePreferences
  })

  // 안전 영역 계산
  const safeAreaInsets = computed(() => ({
    top: `${orientationState.value.safeArea.top}px`,
    bottom: `${orientationState.value.safeArea.bottom}px`,
    left: `${orientationState.value.safeArea.left}px`,
    right: `${orientationState.value.safeArea.right}px`,
  }))

  // 뷰포트 기반 CSS 사용자 정의 속성
  const viewportVariables = computed(() => ({
    '--viewport-width': `${viewport.value.width}px`,
    '--viewport-height': `${viewport.value.height}px`,
    '--available-width': `${viewport.value.availableWidth}px`,
    '--available-height': `${viewport.value.availableHeight}px`,
    '--aspect-ratio': orientationState.value.aspectRatio.toString(),
    '--safe-area-top': safeAreaInsets.value.top,
    '--safe-area-bottom': safeAreaInsets.value.bottom,
    '--safe-area-left': safeAreaInsets.value.left,
    '--safe-area-right': safeAreaInsets.value.right,
    '--orientation-transition-duration': `${settings.transitionDuration}ms`,
  }))

  // 방향 감지
  const detectOrientation = (): 'portrait' | 'landscape' => {
    const { width, height } = viewport.value
    
    // 너비가 높이보다 큰 경우 가로 모드
    if (width > height) {
      return 'landscape'
    }
    
    // 그 외의 경우 세로 모드
    return 'portrait'
  }

  // 방향 각도 계산
  const getOrientationAngle = (): number => {
    if ('screen' in window && 'orientation' in window.screen) {
      return (window.screen.orientation as any).angle || 0
    }
    
    // Fallback: orientation 속성 사용
    if ('orientation' in window) {
      return (window as any).orientation || 0
    }
    
    return 0
  }

  // 안전 영역 계산
  const calculateSafeArea = () => {
    // CSS env() 함수 지원 여부 확인
    if (CSS.supports('padding: env(safe-area-inset-top)')) {
      const computedStyle = getComputedStyle(document.documentElement)
      
      return {
        top: parseInt(computedStyle.getPropertyValue('env(safe-area-inset-top)')) || 0,
        bottom: parseInt(computedStyle.getPropertyValue('env(safe-area-inset-bottom)')) || 0,
        left: parseInt(computedStyle.getPropertyValue('env(safe-area-inset-left)')) || 0,
        right: parseInt(computedStyle.getPropertyValue('env(safe-area-inset-right)')) || 0,
      }
    }
    
    // iOS 감지 및 상태바 높이 추정
    const isIOS = /iPad|iPhone|iPod/.test(navigator.userAgent)
    if (isIOS) {
      const statusBarHeight = isLandscape.value ? 0 : 20
      const homeIndicatorHeight = 34 // iPhone X 이후 홈 인디케이터
      
      return {
        top: statusBarHeight,
        bottom: homeIndicatorHeight,
        left: 0,
        right: 0,
      }
    }
    
    return {
      top: 0,
      bottom: 0,
      left: 0,
      right: 0,
    }
  }

  // 뷰포트 업데이트
  const updateViewport = () => {
    const newViewport = {
      width: window.innerWidth,
      height: window.innerHeight,
      availableWidth: window.screen?.availWidth || window.innerWidth,
      availableHeight: window.screen?.availHeight || window.innerHeight,
      devicePixelRatio: window.devicePixelRatio,
    }
    
    viewport.value = newViewport
    
    // 종횡비 계산
    orientationState.value.aspectRatio = newViewport.width / newViewport.height
    
    // 안전 영역 업데이트
    orientationState.value.safeArea = calculateSafeArea()
  }

  // 방향 변경 처리
  const handleOrientationChange = () => {
    if (orientationTimer) {
      clearTimeout(orientationTimer)
    }
    
    orientationTimer = window.setTimeout(() => {
      const previousOrientation = orientationState.value.current
      const newOrientation = detectOrientation()
      const newAngle = getOrientationAngle()
      
      if (previousOrientation !== newOrientation) {
        // 방향 변경 시작
        orientationState.value.isChanging = true
        orientationState.value.previous = previousOrientation
        orientationState.value.current = newOrientation
        orientationState.value.angle = newAngle
        
        // 뷰포트 업데이트
        updateViewport()
        
        // CSS 변수 적용
        applyOrientationStyles()
        
        // 전환 애니메이션 완료 후 상태 업데이트
        if (transitionTimer) {
          clearTimeout(transitionTimer)
        }
        
        transitionTimer = window.setTimeout(() => {
          orientationState.value.isChanging = false
          orientationState.value.previous = null
        }, settings.transitionDuration)
        
        // 이벤트 발생
        onOrientationChange(newOrientation, previousOrientation)
      } else {
        // 방향은 같지만 크기가 변경된 경우 (키보드 등)
        updateViewport()
        applyOrientationStyles()
      }
    }, settings.debounceDelay)
  }

  // CSS 스타일 적용
  const applyOrientationStyles = () => {
    const root = document.documentElement
    
    // CSS 사용자 정의 속성 적용
    Object.entries(viewportVariables.value).forEach(([key, value]) => {
      root.style.setProperty(key, value)
    })
    
    // 방향 클래스 적용
    root.classList.remove('orientation-portrait', 'orientation-landscape')
    root.classList.add(`orientation-${orientationState.value.current}`)
    
    // 세부 클래스 적용
    if (isNarrowPortrait.value) {
      root.classList.add('orientation-portrait-narrow')
    } else {
      root.classList.remove('orientation-portrait-narrow')
    }
    
    if (isWidePortrait.value) {
      root.classList.add('orientation-portrait-wide')
    } else {
      root.classList.remove('orientation-portrait-wide')
    }
    
    if (isCompactLandscape.value) {
      root.classList.add('orientation-landscape-compact')
    } else {
      root.classList.remove('orientation-landscape-compact')
    }
    
    if (isWideLandscape.value) {
      root.classList.add('orientation-landscape-wide')
    } else {
      root.classList.remove('orientation-landscape-wide')
    }
    
    // 전환 클래스
    if (orientationState.value.isChanging && settings.enableTransitions) {
      root.classList.add('orientation-transitioning')
    } else {
      root.classList.remove('orientation-transitioning')
    }
  }

  // 방향별 레이아웃 조정
  const adjustLayoutForOrientation = () => {
    const options = currentLayoutOptions.value
    
    if (isPortrait.value) {
      adjustPortraitLayout(options as PortraitLayoutOptions)
    } else {
      adjustLandscapeLayout(options as LandscapeLayoutOptions)
    }
  }

  // 세로 모드 레이아웃 조정
  const adjustPortraitLayout = (options: PortraitLayoutOptions) => {
    const root = document.documentElement
    
    // 네비게이션 위치
    root.style.setProperty('--nav-position', options.navigationPosition || 'bottom')
    
    // 콘텐츠 패딩
    root.style.setProperty('--content-padding', `${options.contentPadding || 16}px`)
    
    // 전체 높이 사용 여부
    if (options.enableFullHeight) {
      root.classList.add('full-height-content')
    } else {
      root.classList.remove('full-height-content')
    }
    
    // 키보드 동작 설정
    root.setAttribute('data-keyboard-behavior', options.keyboardBehavior || 'resize')
  }

  // 가로 모드 레이아웃 조정
  const adjustLandscapeLayout = (options: LandscapeLayoutOptions) => {
    const root = document.documentElement
    
    // 네비게이션 위치
    root.style.setProperty('--nav-position', options.navigationPosition || 'left')
    
    // 사이드 패널 활성화
    if (options.enableSidePanel) {
      root.classList.add('enable-side-panel')
    } else {
      root.classList.remove('enable-side-panel')
    }
    
    // 콘텐츠 열 수
    root.style.setProperty('--content-columns', (options.contentColumns || 1).toString())
    
    // 컴팩트 모드
    if (options.compactMode) {
      root.classList.add('compact-landscape')
    } else {
      root.classList.remove('compact-landscape')
    }
  }

  // 키보드 인식 처리
  const handleKeyboardToggle = () => {
    // Visual Viewport API 지원 확인
    if ('visualViewport' in window) {
      const visualViewport = (window as any).visualViewport
      
      const updateKeyboardState = () => {
        const keyboardHeight = window.innerHeight - visualViewport.height
        const isKeyboardVisible = keyboardHeight > 50 // 임계값
        
        const root = document.documentElement
        root.style.setProperty('--keyboard-height', `${keyboardHeight}px`)
        
        if (isKeyboardVisible) {
          root.classList.add('keyboard-visible')
        } else {
          root.classList.remove('keyboard-visible')
        }
      }
      
      visualViewport.addEventListener('resize', updateKeyboardState)
      
      return () => {
        visualViewport.removeEventListener('resize', updateKeyboardState)
      }
    }
    
    return () => {}
  }

  // 방향 변경 이벤트 콜백
  const orientationChangeCallbacks = ref<((newOrientation: string, oldOrientation: string) => void)[]>([])
  
  const onOrientationChange = (newOrientation: string, oldOrientation: string) => {
    orientationChangeCallbacks.value.forEach(callback => {
      callback(newOrientation, oldOrientation)
    })
    
    // 레이아웃 조정
    if (settings.enableLayoutSwitching) {
      adjustLayoutForOrientation()
    }
    
    console.log(`Orientation changed: ${oldOrientation} → ${newOrientation}`)
  }

  // 방향 변경 리스너 등록
  const addOrientationChangeListener = (callback: (newOrientation: string, oldOrientation: string) => void) => {
    orientationChangeCallbacks.value.push(callback)
  }

  // 방향 변경 리스너 제거
  const removeOrientationChangeListener = (callback: (newOrientation: string, oldOrientation: string) => void) => {
    const index = orientationChangeCallbacks.value.indexOf(callback)
    if (index > -1) {
      orientationChangeCallbacks.value.splice(index, 1)
    }
  }

  // 방향 강제 설정 (테스트용)
  const forceOrientation = (orientation: 'portrait' | 'landscape') => {
    if ('screen' in window && 'orientation' in window.screen && 'lock' in (window.screen as any).orientation) {
      const orientationMap = {
        portrait: 'portrait-primary',
        landscape: 'landscape-primary',
      }
      
      return (window.screen as any).orientation.lock(orientationMap[orientation])
    }
    
    return Promise.reject(new Error('Screen orientation lock not supported'))
  }

  // 방향 잠금 해제
  const unlockOrientation = () => {
    if ('screen' in window && 'orientation' in window.screen && 'unlock' in (window.screen as any).orientation) {
      (window.screen as any).orientation.unlock()
    }
  }

  // 이벤트 리스너 설정
  const setupEventListeners = () => {
    // 방향 변경 이벤트
    window.addEventListener('orientationchange', handleOrientationChange)
    window.addEventListener('resize', handleOrientationChange)
    
    // 키보드 이벤트
    const removeKeyboardListener = handleKeyboardToggle()
    
    return () => {
      window.removeEventListener('orientationchange', handleOrientationChange)
      window.removeEventListener('resize', handleOrientationChange)
      removeKeyboardListener()
      
      if (orientationTimer) {
        clearTimeout(orientationTimer)
      }
      if (transitionTimer) {
        clearTimeout(transitionTimer)
      }
    }
  }

  // 초기 설정
  const initialize = () => {
    // 초기 상태 설정
    orientationState.value.current = detectOrientation()
    orientationState.value.angle = getOrientationAngle()
    
    // 뷰포트 초기 설정
    updateViewport()
    
    // 스타일 적용
    applyOrientationStyles()
    
    // 레이아웃 조정
    if (settings.enableLayoutSwitching) {
      adjustLayoutForOrientation()
    }
  }

  // 라이프사이클
  onMounted(() => {
    initialize()
    const cleanup = setupEventListeners()
    
    onBeforeUnmount(() => {
      cleanup()
    })
  })

  return {
    // 상태
    orientationState: computed(() => orientationState.value),
    viewport: computed(() => viewport.value),
    
    // 계산된 속성
    isPortrait,
    isLandscape,
    isNarrowPortrait,
    isWidePortrait,
    isCompactLandscape,
    isWideLandscape,
    currentLayoutOptions,
    safeAreaInsets,
    viewportVariables,
    
    // 메서드
    forceOrientation,
    unlockOrientation,
    addOrientationChangeListener,
    removeOrientationChangeListener,
    
    // 유틸리티
    detectOrientation,
    updateViewport,
    applyOrientationStyles,
    adjustLayoutForOrientation,
  }
}