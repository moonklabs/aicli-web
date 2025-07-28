import { ref, onMounted, onBeforeUnmount, nextTick, watch } from 'vue'
import { useDeviceDetection } from './useTouchPerformance'

interface AccessibilityOptions {
  enableFocusManagement?: boolean
  enableScreenReaderOptimization?: boolean
  enableKeyboardNavigation?: boolean
  enableHighContrastMode?: boolean
  enableTextScaling?: boolean
  enableReducedMotion?: boolean
  touchTargetMinSize?: number
}

export function useMobileAccessibility(options: AccessibilityOptions = {}) {
  const {
    enableFocusManagement = true,
    enableScreenReaderOptimization = true,
    enableKeyboardNavigation = true,
    enableHighContrastMode = true,
    enableTextScaling = true,
    enableReducedMotion = true,
    touchTargetMinSize = 44, // WCAG 권장 최소 터치 타겟 크기 (44px)
  } = options

  const { isTouchDevice, isMobile } = useDeviceDetection()

  // 접근성 상태
  const isScreenReaderActive = ref(false)
  const isHighContrastMode = ref(false)
  const isReducedMotionPreferred = ref(false)
  const textScaleFactor = ref(1)
  const focusedElement = ref<HTMLElement | null>(null)

  // 포커스 관리
  const focusHistory = ref<HTMLElement[]>([])
  const focusTrapStack = ref<HTMLElement[]>([])

  // 터치 타겟 크기 검증
  const validateTouchTargets = () => {
    if (!isTouchDevice.value) return

    const interactiveElements = document.querySelectorAll(
      'button, input, select, textarea, a, [role="button"], [role="link"], [tabindex]'
    )

    interactiveElements.forEach((element) => {
      const el = element as HTMLElement
      const rect = el.getBoundingClientRect()
      
      if (rect.width < touchTargetMinSize || rect.height < touchTargetMinSize) {
        // 터치 타겟이 작으면 패딩으로 확장
        const paddingH = Math.max(0, (touchTargetMinSize - rect.width) / 2)
        const paddingV = Math.max(0, (touchTargetMinSize - rect.height) / 2)
        
        el.style.paddingLeft = `${paddingH}px`
        el.style.paddingRight = `${paddingH}px`
        el.style.paddingTop = `${paddingV}px`
        el.style.paddingBottom = `${paddingV}px`
        el.style.minWidth = `${touchTargetMinSize}px`
        el.style.minHeight = `${touchTargetMinSize}px`
        
        // 접근성 개선을 위한 마킹
        el.setAttribute('data-touch-optimized', 'true')
      }
    })
  }

  // 스크린 리더 감지
  const detectScreenReader = () => {
    // 스크린 리더가 활성화되어 있을 때의 여러 신호들을 감지
    const hasScreenReader = 
      // NVDA, JAWS 등이 활성화되면 speechSynthesis가 활성화됨
      (window.speechSynthesis && window.speechSynthesis.getVoices().length > 0) ||
      // VoiceOver (iOS/macOS)
      /VoiceOver/i.test(navigator.userAgent) ||
      // 스크린 리더 전용 CSS가 로드되었는지 확인
      document.querySelector('[data-screen-reader]') !== null

    isScreenReaderActive.value = hasScreenReader

    if (hasScreenReader) {
      document.body.classList.add('screen-reader-active')
      
      // 스크린 리더용 최적화
      optimizeForScreenReader()
    }
  }

  // 스크린 리더 최적화
  const optimizeForScreenReader = () => {
    if (!enableScreenReaderOptimization) return

    // ARIA 라이브 영역 설정
    const liveRegion = document.createElement('div')
    liveRegion.setAttribute('aria-live', 'polite')
    liveRegion.setAttribute('aria-atomic', 'true')
    liveRegion.className = 'sr-only live-region'
    liveRegion.style.cssText = `
      position: absolute !important;
      width: 1px !important;
      height: 1px !important;
      padding: 0 !important;
      margin: -1px !important;
      overflow: hidden !important;
      clip: rect(0, 0, 0, 0) !important;
      white-space: nowrap !important;
      border: 0 !important;
    `
    document.body.appendChild(liveRegion)

    // 모든 이미지에 alt 속성 확인
    document.querySelectorAll('img').forEach((img) => {
      if (!img.alt && !img.getAttribute('aria-label')) {
        img.alt = '이미지'
      }
    })

    // 폼 요소 레이블 확인
    document.querySelectorAll('input, select, textarea').forEach((input) => {
      const element = input as HTMLInputElement
      if (!element.labels?.length && !element.getAttribute('aria-label') && !element.getAttribute('aria-labelledby')) {
        console.warn('폼 요소에 레이블이 없습니다:', element)
      }
    })
  }

  // 고대비 모드 감지
  const detectHighContrastMode = () => {
    if (!enableHighContrastMode) return

    // CSS 미디어 쿼리로 고대비 모드 감지
    const mediaQuery = window.matchMedia('(prefers-contrast: high)')
    
    const updateHighContrastMode = (e: MediaQueryListEvent | MediaQueryList) => {
      isHighContrastMode.value = e.matches
      
      if (e.matches) {
        document.body.classList.add('high-contrast-mode')
      } else {
        document.body.classList.remove('high-contrast-mode')
      }
    }

    mediaQuery.addListener(updateHighContrastMode)
    updateHighContrastMode(mediaQuery)

    return () => {
      mediaQuery.removeListener(updateHighContrastMode)
    }
  }

  // 애니메이션 감소 선호도 감지
  const detectReducedMotionPreference = () => {
    if (!enableReducedMotion) return

    const mediaQuery = window.matchMedia('(prefers-reduced-motion: reduce)')
    
    const updateReducedMotion = (e: MediaQueryListEvent | MediaQueryList) => {
      isReducedMotionPreferred.value = e.matches
      
      if (e.matches) {
        document.body.classList.add('reduced-motion')
        // 모든 애니메이션과 전환 효과 비활성화
        const style = document.createElement('style')
        style.textContent = `
          .reduced-motion *,
          .reduced-motion *::before,
          .reduced-motion *::after {
            animation-duration: 0.01ms !important;
            animation-iteration-count: 1 !important;
            transition-duration: 0.01ms !important;
            scroll-behavior: auto !important;
          }
        `
        document.head.appendChild(style)
      } else {
        document.body.classList.remove('reduced-motion')
      }
    }

    mediaQuery.addListener(updateReducedMotion)
    updateReducedMotion(mediaQuery)

    return () => {
      mediaQuery.removeListener(updateReducedMotion)
    }
  }

  // 텍스트 크기 조정
  const setupTextScaling = () => {
    if (!enableTextScaling) return

    const calculateTextScaleFactor = () => {
      // 시스템 글꼴 크기 설정을 감지
      const testElement = document.createElement('div')
      testElement.style.cssText = `
        position: absolute;
        visibility: hidden;
        font-size: 1rem;
        width: auto;
        height: auto;
      `
      testElement.textContent = 'Test'
      document.body.appendChild(testElement)
      
      const computedSize = window.getComputedStyle(testElement).fontSize
      const baseFontSize = 16 // 기본 폰트 크기 (16px)
      textScaleFactor.value = parseFloat(computedSize) / baseFontSize
      
      document.body.removeChild(testElement)
    }

    calculateTextScaleFactor()

    // 폰트 크기 변경 감지
    window.addEventListener('resize', calculateTextScaleFactor)

    return () => {
      window.removeEventListener('resize', calculateTextScaleFactor)
    }
  }

  // 포커스 관리
  const setupFocusManagement = () => {
    if (!enableFocusManagement) return

    const handleFocusChange = (event: FocusEvent) => {
      const target = event.target as HTMLElement
      
      if (target && target !== focusedElement.value) {
        if (focusedElement.value) {
          focusHistory.value.push(focusedElement.value)
          // 포커스 히스토리 크기 제한
          if (focusHistory.value.length > 10) {
            focusHistory.value = focusHistory.value.slice(-10)
          }
        }
        focusedElement.value = target
      }
    }

    document.addEventListener('focusin', handleFocusChange)
    document.addEventListener('focusout', handleFocusChange)

    return () => {
      document.removeEventListener('focusin', handleFocusChange)
      document.removeEventListener('focusout', handleFocusChange)
    }
  }

  // 키보드 네비게이션 최적화
  const setupKeyboardNavigation = () => {
    if (!enableKeyboardNavigation) return

    const handleKeyboardNavigation = (event: KeyboardEvent) => {
      // Tab 키 순환 개선
      if (event.key === 'Tab') {
        const focusableElements = getFocusableElements()
        const currentIndex = Array.from(focusableElements).indexOf(document.activeElement as HTMLElement)
        
        if (event.shiftKey) {
          // Shift+Tab (이전 요소)
          if (currentIndex === 0) {
            event.preventDefault()
            const lastElement = focusableElements[focusableElements.length - 1] as HTMLElement
            lastElement.focus()
          }
        } else {
          // Tab (다음 요소)
          if (currentIndex === focusableElements.length - 1) {
            event.preventDefault()
            const firstElement = focusableElements[0] as HTMLElement
            firstElement.focus()
          }
        }
      }

      // 방향키로 네비게이션 (모바일에서 외부 키보드 사용 시)
      if (['ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight'].includes(event.key)) {
        const activeElement = document.activeElement as HTMLElement
        const parent = activeElement.closest('[role="menu"], [role="listbox"], [role="grid"]')
        
        if (parent) {
          event.preventDefault()
          navigateWithArrowKeys(event.key, parent)
        }
      }

      // Escape 키로 모달/팝업 닫기
      if (event.key === 'Escape') {
        const modal = document.querySelector('[role="dialog"]:not([hidden])')
        if (modal) {
          const closeButton = modal.querySelector('[aria-label*="닫기"], [aria-label*="close"]') as HTMLElement
          closeButton?.click()
        }
      }
    }

    document.addEventListener('keydown', handleKeyboardNavigation)

    return () => {
      document.removeEventListener('keydown', handleKeyboardNavigation)
    }
  }

  // 포커스 가능한 요소들 찾기
  const getFocusableElements = () => {
    return document.querySelectorAll(
      'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'
    )
  }

  // 방향키 네비게이션
  const navigateWithArrowKeys = (key: string, container: Element) => {
    const items = container.querySelectorAll('[role="menuitem"], [role="option"], [role="gridcell"]')
    const currentIndex = Array.from(items).indexOf(document.activeElement!)
    
    let nextIndex = currentIndex
    
    switch (key) {
      case 'ArrowDown':
        nextIndex = (currentIndex + 1) % items.length
        break
      case 'ArrowUp':
        nextIndex = currentIndex === 0 ? items.length - 1 : currentIndex - 1
        break
      case 'ArrowRight':
        nextIndex = Math.min(currentIndex + 1, items.length - 1)
        break
      case 'ArrowLeft':
        nextIndex = Math.max(currentIndex - 1, 0)
        break
    }
    
    if (nextIndex !== currentIndex) {
      (items[nextIndex] as HTMLElement).focus()
    }
  }

  // 포커스 트랩 설정
  const trapFocus = (container: HTMLElement) => {
    focusTrapStack.value.push(container)
    
    const focusableElements = container.querySelectorAll(
      'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'
    )
    
    if (focusableElements.length > 0) {
      (focusableElements[0] as HTMLElement).focus()
    }
  }

  // 포커스 트랩 해제
  const releaseFocusTrap = () => {
    focusTrapStack.value.pop()
    
    // 이전 포커스 복원
    if (focusHistory.value.length > 0) {
      const previousElement = focusHistory.value.pop()
      previousElement?.focus()
    }
  }

  // 라이브 영역에 메시지 알림
  const announceToScreenReader = (message: string, priority: 'polite' | 'assertive' = 'polite') => {
    const liveRegion = document.querySelector('.live-region') as HTMLElement
    if (liveRegion) {
      liveRegion.setAttribute('aria-live', priority)
      liveRegion.textContent = message
      
      // 메시지를 읽은 후 정리
      setTimeout(() => {
        liveRegion.textContent = ''
      }, 1000)
    }
  }

  // 페이지 제목 업데이트 (스크린 리더 알림)
  const updatePageTitle = (title: string) => {
    document.title = title
    announceToScreenReader(`페이지가 ${title}로 변경되었습니다`)
  }

  // 접근성 검사
  const performAccessibilityAudit = () => {
    const issues: string[] = []

    // 이미지 alt 속성 검사
    document.querySelectorAll('img').forEach((img, index) => {
      if (!img.alt && !img.getAttribute('aria-label')) {
        issues.push(`이미지 ${index + 1}: alt 속성이 없습니다`)
      }
    })

    // 폼 레이블 검사
    document.querySelectorAll('input, select, textarea').forEach((input, index) => {
      const element = input as HTMLInputElement
      if (!element.labels?.length && !element.getAttribute('aria-label') && !element.getAttribute('aria-labelledby')) {
        issues.push(`폼 요소 ${index + 1}: 레이블이 없습니다`)
      }
    })

    // 색상 대비 검사 (간단한 검사)
    const elementsWithBgColor = document.querySelectorAll('[style*="background-color"], [style*="color"]')
    elementsWithBgColor.forEach((element, index) => {
      const styles = window.getComputedStyle(element)
      const bgColor = styles.backgroundColor
      const textColor = styles.color
      
      if (bgColor !== 'rgba(0, 0, 0, 0)' && textColor !== 'rgba(0, 0, 0, 0)') {
        // 실제 색상 대비 계산은 복잡하므로 여기서는 경고만 출력
        console.warn(`요소 ${index + 1}: 색상 대비를 확인하세요`, { bgColor, textColor })
      }
    })

    return issues
  }

  // 초기화 및 정리
  let cleanupFunctions: (() => void)[] = []

  onMounted(() => {
    nextTick(() => {
      detectScreenReader()
      validateTouchTargets()
      
      const highContrastCleanup = detectHighContrastMode()
      const reducedMotionCleanup = detectReducedMotionPreference()
      const textScalingCleanup = setupTextScaling()
      const focusCleanup = setupFocusManagement()
      const keyboardCleanup = setupKeyboardNavigation()

      if (highContrastCleanup) cleanupFunctions.push(highContrastCleanup)
      if (reducedMotionCleanup) cleanupFunctions.push(reducedMotionCleanup)
      if (textScalingCleanup) cleanupFunctions.push(textScalingCleanup)
      if (focusCleanup) cleanupFunctions.push(focusCleanup)
      if (keyboardCleanup) cleanupFunctions.push(keyboardCleanup)
    })
  })

  onBeforeUnmount(() => {
    cleanupFunctions.forEach(cleanup => cleanup())
    cleanupFunctions = []
  })

  return {
    // 상태
    isScreenReaderActive,
    isHighContrastMode,
    isReducedMotionPreferred,
    textScaleFactor,
    focusedElement,

    // 메서드
    trapFocus,
    releaseFocusTrap,
    announceToScreenReader,
    updatePageTitle,
    validateTouchTargets,
    performAccessibilityAudit,

    // 유틸리티
    focusHistory,
    getFocusableElements,
  }
}