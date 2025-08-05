import { computed, ref, watch, onMounted, onBeforeUnmount } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMobileOptimization } from './useMobileOptimization'
import { useAdvancedGestures } from './useAdvancedGestures'

interface MobileWorkflowConfig {
  // 빠른 액션 설정
  enableQuickActions?: boolean
  enableSwipeNavigation?: boolean
  enableOneHandMode?: boolean
  enableContextualActions?: boolean
  
  // 레이아웃 최적화
  enableBottomSheetActions?: boolean
  enableFloatingActionButton?: boolean
  enablePullToRefresh?: boolean
  enableInfiniteScroll?: boolean
  
  // 접근성 최적화
  enableVoiceCommands?: boolean
  enableShakeToUndo?: boolean
  enableLargeTargets?: boolean
  
  // 성능 최적화
  enableLazyLoading?: boolean
  enablePrefetching?: boolean
  enableCaching?: boolean
}

interface QuickAction {
  id: string
  label: string
  icon: string
  action: () => void
  category: 'primary' | 'secondary' | 'contextual'
  shortcut?: string
  requiresAuth?: boolean
}

interface MobileWorkflowState {
  isOneHandMode: boolean
  currentContext: string
  quickActionsVisible: boolean
  bottomSheetOpen: boolean
  reachabilityZone: 'easy' | 'medium' | 'hard'
}

export function useMobileWorkflow(config: MobileWorkflowConfig = {}) {
  const defaultConfig: Required<MobileWorkflowConfig> = {
    enableQuickActions: true,
    enableSwipeNavigation: true,
    enableOneHandMode: true,
    enableContextualActions: true,
    enableBottomSheetActions: true,
    enableFloatingActionButton: true,
    enablePullToRefresh: true,
    enableInfiniteScroll: true,
    enableVoiceCommands: false,
    enableShakeToUndo: true,
    enableLargeTargets: true,
    enableLazyLoading: true,
    enablePrefetching: true,
    enableCaching: true,
  }

  const settings = { ...defaultConfig, ...config }
  
  const route = useRoute()
  const router = useRouter()
  const { isMobile, screenHeight, screenWidth } = useMobileOptimization()

  // 상태 관리
  const workflowState = ref<MobileWorkflowState>({
    isOneHandMode: false,
    currentContext: 'general',
    quickActionsVisible: false,
    bottomSheetOpen: false,
    reachabilityZone: 'easy',
  })

  // 빠른 액션 목록
  const quickActions = ref<QuickAction[]>([])
  const contextualActions = ref<QuickAction[]>([])

  // 한 손 조작 모드 감지
  const reachabilityThreshold = computed(() => {
    // 화면 크기에 따른 한 손 조작 임계값
    if (screenHeight.value <= 667) return 0.7 // iPhone SE/8 크기
    if (screenHeight.value <= 736) return 0.6 // iPhone 8 Plus 크기
    if (screenHeight.value <= 812) return 0.5 // iPhone X/11 Pro 크기
    return 0.4 // 큰 화면
  })

  // 도달하기 쉬운 영역 계산
  const reachableArea = computed(() => {
    const threshold = reachabilityThreshold.value
    return {
      top: screenHeight.value * (1 - threshold),
      bottom: screenHeight.value,
      left: 0,
      right: screenWidth.value,
    }
  })

  // 현재 컨텍스트에 따른 액션 필터링
  const availableActions = computed(() => {
    const currentRoute = route.name as string
    const context = getContextFromRoute(currentRoute)
    
    return quickActions.value.filter(action => {
      if (action.category === 'contextual') {
        return contextualActions.value.some(ca => ca.id === action.id)
      }
      return true
    })
  })

  // 라우트에서 컨텍스트 추출
  const getContextFromRoute = (routeName: string): string => {
    const contextMap: Record<string, string> = {
      'dashboard': 'overview',
      'workspaces': 'workspace',
      'terminal': 'terminal',
      'docker': 'docker',
      'profile': 'profile',
      'settings': 'settings',
    }
    
    return contextMap[routeName] || 'general'
  }

  // 기본 빠른 액션 설정
  const setupDefaultQuickActions = () => {
    quickActions.value = [
      {
        id: 'new-workspace',
        label: '새 워크스페이스',
        icon: 'AddIcon',
        action: () => router.push('/workspaces/new'),
        category: 'primary',
        shortcut: 'Cmd+N',
      },
      {
        id: 'open-terminal',
        label: '터미널 열기',
        icon: 'TerminalIcon',
        action: () => router.push('/terminal'),
        category: 'primary',
        shortcut: 'Cmd+T',
      },
      {
        id: 'search',
        label: '검색',
        icon: 'SearchIcon',
        action: () => openSearch(),
        category: 'secondary',
        shortcut: 'Cmd+K',
      },
      {
        id: 'notifications',
        label: '알림',
        icon: 'BellIcon',
        action: () => openNotifications(),
        category: 'secondary',
      },
      {
        id: 'help',
        label: '도움말',
        icon: 'HelpIcon',
        action: () => openHelp(),
        category: 'secondary',
        shortcut: '?',
      },
    ]
  }

  // 컨텍스트별 액션 설정
  const setupContextualActions = () => {
    const context = workflowState.value.currentContext
    
    const contextActions: Record<string, QuickAction[]> = {
      workspace: [
        {
          id: 'run-command',
          label: '명령 실행',
          icon: 'PlayIcon',
          action: () => runCommand(),
          category: 'contextual',
        },
        {
          id: 'save-workspace',
          label: '워크스페이스 저장',
          icon: 'SaveIcon',
          action: () => saveWorkspace(),
          category: 'contextual',
          shortcut: 'Cmd+S',
        },
      ],
      terminal: [
        {
          id: 'clear-terminal',
          label: '터미널 지우기',
          icon: 'ClearIcon',
          action: () => clearTerminal(),
          category: 'contextual',
          shortcut: 'Cmd+K',
        },
        {
          id: 'new-tab',
          label: '새 탭',
          icon: 'TabIcon',
          action: () => newTerminalTab(),
          category: 'contextual',
          shortcut: 'Cmd+T',
        },
      ],
      docker: [
        {
          id: 'stop-container',
          label: '컨테이너 중지',
          icon: 'StopIcon',
          action: () => stopContainer(),
          category: 'contextual',
        },
        {
          id: 'view-logs',
          label: '로그 보기',
          icon: 'LogIcon',
          action: () => viewLogs(),
          category: 'contextual',
        },
      ],
    }

    contextualActions.value = contextActions[context] || []
  }

  // 한 손 조작 모드 토글
  const toggleOneHandMode = () => {
    workflowState.value.isOneHandMode = !workflowState.value.isOneHandMode
    
    // 한 손 모드 시 UI 조정
    if (workflowState.value.isOneHandMode) {
      // DOM 요소들을 아래쪽으로 이동
      adjustUIForOneHandMode()
    } else {
      // 원래 위치로 복원
      restoreUILayout()
    }
  }

  // UI를 한 손 조작에 맞게 조정
  const adjustUIForOneHandMode = () => {
    const rootElement = document.documentElement
    rootElement.classList.add('one-hand-mode')
    
    // CSS 커스텀 프로퍼티로 조정 값 설정
    rootElement.style.setProperty('--one-hand-offset', '30%')
    rootElement.style.setProperty('--one-hand-scale', '0.85')
  }

  // UI 레이아웃 복원
  const restoreUILayout = () => {
    const rootElement = document.documentElement
    rootElement.classList.remove('one-hand-mode')
    rootElement.style.removeProperty('--one-hand-offset')
    rootElement.style.removeProperty('--one-hand-scale')
  }

  // 스와이프 네비게이션 설정
  const setupSwipeNavigation = () => {
    if (!settings.enableSwipeNavigation) return

    // 좌우 스와이프로 페이지 이동
    const handleSwipeLeft = () => {
      // 다음 페이지로 이동 (히스토리 기반)
      if (window.history.state) {
        router.forward()
      }
    }

    const handleSwipeRight = () => {
      // 이전 페이지로 이동
      router.back()
    }

    // 스와이프 이벤트 리스너 등록 (실제 구현에서는 제스처 라이브러리 사용)
    return {
      handleSwipeLeft,
      handleSwipeRight,
    }
  }

  // 풀 투 리프레시 설정
  const setupPullToRefresh = () => {
    if (!settings.enablePullToRefresh) return

    const handlePullToRefresh = async () => {
      // 현재 페이지 데이터 새로고침
      try {
        await refreshCurrentPage()
        showRefreshSuccess()
      } catch (error) {
        showRefreshError()
      }
    }

    return { handlePullToRefresh }
  }

  // 무한 스크롤 설정
  const setupInfiniteScroll = () => {
    if (!settings.enableInfiniteScroll) return

    const handleLoadMore = async () => {
      // 추가 데이터 로드
      try {
        await loadMoreData()
      } catch (error) {
        console.error('Failed to load more data:', error)
      }
    }

    return { handleLoadMore }
  }

  // 음성 명령 설정
  const setupVoiceCommands = () => {
    if (!settings.enableVoiceCommands || !('webkitSpeechRecognition' in window)) return

    const recognition = new (window as any).webkitSpeechRecognition()
    recognition.continuous = false
    recognition.interimResults = false
    recognition.lang = 'ko-KR'

    recognition.onresult = (event: any) => {
      const command = event.results[0][0].transcript.toLowerCase()
      handleVoiceCommand(command)
    }

    const startListening = () => {
      recognition.start()
    }

    const stopListening = () => {
      recognition.stop()
    }

    return { startListening, stopListening }
  }

  // 디바이스 흔들기 감지 (실행 취소)
  const setupShakeToUndo = () => {
    if (!settings.enableShakeToUndo) return

    let acceleration = { x: 0, y: 0, z: 0 }
    let lastTime = 0

    const handleDeviceMotion = (event: DeviceMotionEvent) => {
      const current = Date.now()
      
      if (current - lastTime > 100) { // 100ms 간격으로 체크
        const deltaTime = current - lastTime
        lastTime = current

        if (event.accelerationIncludingGravity) {
          const { x, y, z } = event.accelerationIncludingGravity
          
          const deltaX = Math.abs(x! - acceleration.x)
          const deltaY = Math.abs(y! - acceleration.y)
          const deltaZ = Math.abs(z! - acceleration.z)
          
          acceleration = { x: x!, y: y!, z: z! }

          // 흔들기 감지 임계값
          if (deltaX + deltaY + deltaZ > 15) {
            handleShakeGesture()
          }
        }
      }
    }

    if (typeof DeviceMotionEvent !== 'undefined' && DeviceMotionEvent.requestPermission) {
      // iOS 13+ 권한 요청
      DeviceMotionEvent.requestPermission().then(response => {
        if (response === 'granted') {
          window.addEventListener('devicemotion', handleDeviceMotion)
        }
      })
    } else {
      window.addEventListener('devicemotion', handleDeviceMotion)
    }

    return () => {
      window.removeEventListener('devicemotion', handleDeviceMotion)
    }
  }

  // 도달성 분석
  const analyzeReachability = (x: number, y: number): 'easy' | 'medium' | 'hard' => {
    const reachable = reachableArea.value
    
    if (y >= reachable.top && y <= reachable.bottom) {
      if (x <= screenWidth.value * 0.7) {
        return 'easy' // 엄지로 쉽게 도달 가능
      } else {
        return 'medium' // 손가락을 뻗으면 도달 가능
      }
    }
    
    return 'hard' // 양손 또는 위치 조정 필요
  }

  // 액션 실행 함수들
  const openSearch = () => {
    // 검색 모달/페이지 열기
    console.log('Opening search...')
  }

  const openNotifications = () => {
    // 알림 패널 열기
    console.log('Opening notifications...')
  }

  const openHelp = () => {
    // 도움말 모달 열기
    console.log('Opening help...')
  }

  const runCommand = () => {
    // 명령 실행
    console.log('Running command...')
  }

  const saveWorkspace = () => {
    // 워크스페이스 저장
    console.log('Saving workspace...')
  }

  const clearTerminal = () => {
    // 터미널 지우기
    console.log('Clearing terminal...')
  }

  const newTerminalTab = () => {
    // 새 터미널 탭
    console.log('Creating new terminal tab...')
  }

  const stopContainer = () => {
    // 컨테이너 중지
    console.log('Stopping container...')
  }

  const viewLogs = () => {
    // 로그 보기
    console.log('Viewing logs...')
  }

  const refreshCurrentPage = async () => {
    // 현재 페이지 새로고침
    await new Promise(resolve => setTimeout(resolve, 1000))
  }

  const showRefreshSuccess = () => {
    console.log('Refresh successful')
  }

  const showRefreshError = () => {
    console.log('Refresh failed')
  }

  const loadMoreData = async () => {
    // 추가 데이터 로드
    await new Promise(resolve => setTimeout(resolve, 1000))
  }

  const handleVoiceCommand = (command: string) => {
    const commandMap: Record<string, () => void> = {
      '새 워크스페이스': () => router.push('/workspaces/new'),
      '터미널 열기': () => router.push('/terminal'),
      '검색': openSearch,
      '알림': openNotifications,
      '도움말': openHelp,
    }

    const action = commandMap[command]
    if (action) {
      action()
    }
  }

  const handleShakeGesture = () => {
    // 마지막 액션 실행 취소
    console.log('Shake detected - undoing last action...')
    // 실제 실행 취소 로직 구현
  }

  // 라이프사이클
  onMounted(() => {
    setupDefaultQuickActions()
    setupContextualActions()
    
    if (isMobile.value) {
      setupSwipeNavigation()
      setupPullToRefresh()
      setupInfiniteScroll()
      
      if (settings.enableVoiceCommands) {
        setupVoiceCommands()
      }
      
      if (settings.enableShakeToUndo) {
        setupShakeToUndo()
      }
    }
  })

  // 라우트 변경 시 컨텍스트 업데이트
  watch(() => route.name, (newRoute) => {
    if (newRoute) {
      workflowState.value.currentContext = getContextFromRoute(newRoute as string)
      setupContextualActions()
    }
  }, { immediate: true })

  return {
    // 상태
    workflowState: computed(() => workflowState.value),
    isMobile,
    reachableArea,
    availableActions,
    
    // 액션
    toggleOneHandMode,
    analyzeReachability,
    
    // 설정
    setupSwipeNavigation,
    setupPullToRefresh,
    setupInfiniteScroll,
    setupVoiceCommands,
    setupShakeToUndo,
    
    // 유틸리티
    getContextFromRoute,
    adjustUIForOneHandMode,
    restoreUILayout,
    
    // 데이터
    quickActions: computed(() => quickActions.value),
    contextualActions: computed(() => contextualActions.value),
  }
}