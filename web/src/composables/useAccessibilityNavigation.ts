/**
 * 접근성 네비게이션 컴포저블
 * 키보드 네비게이션, 포커스 관리, 스킵 링크 등을 제공
 */
import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue'

interface FocusableElement extends HTMLElement {
  tabIndex: number
}

interface FocusOptions {
  preventScroll?: boolean
  focusVisible?: boolean
}

interface SkipLinkTarget {
  id: string
  label: string
  element?: HTMLElement
}

// 포커스 가능한 요소 셀렉터
const FOCUSABLE_ELEMENTS = [
  'a[href]',
  'button:not([disabled])',
  'input:not([disabled]):not([type="hidden"])',
  'select:not([disabled])',
  'textarea:not([disabled])',
  '[tabindex]:not([tabindex="-1"])',
  '[contenteditable="true"]',
  'audio[controls]',
  'video[controls]',
  'details summary',
  'iframe',
].join(', ')

// 전역 상태
const currentFocusedElement = ref<HTMLElement | null>(null)
const focusHistory = ref<HTMLElement[]>([])
const skipLinkTargets = ref<SkipLinkTarget[]>([])
const keyboardNavigationEnabled = ref(true)

/**
 * 접근성 네비게이션 컴포저블
 */
export function useAccessibilityNavigation() {

  /**
   * 요소가 포커스 가능한지 확인
   */
  const isFocusable = (element: HTMLElement): boolean => {
    if (!element || element.getAttribute('aria-hidden') === 'true') {
      return false
    }

    // 비활성화된 요소 확인
    if (element.hasAttribute('disabled') || element.getAttribute('aria-disabled') === 'true') {
      return false
    }

    // 보이지 않는 요소 확인
    const style = window.getComputedStyle(element)
    if (style.display === 'none' || style.visibility === 'hidden' || style.opacity === '0') {
      return false
    }

    // 기본 포커스 가능 요소인지 확인
    if (element.matches(FOCUSABLE_ELEMENTS)) {
      return true
    }

    return false
  }

  /**
   * 컨테이너 내의 모든 포커스 가능한 요소 찾기
   */
  const getFocusableElements = (container: HTMLElement = document.body): HTMLElement[] => {
    const elements = Array.from(container.querySelectorAll(FOCUSABLE_ELEMENTS)) as HTMLElement[]
    return elements.filter(isFocusable)
  }

  /**
   * 요소에 포커스 설정
   */
  const focusElement = (element: HTMLElement, options: FocusOptions = {}): void => {
    if (!element || !isFocusable(element)) {
      return
    }

    // 이전 포커스 요소를 히스토리에 추가
    if (currentFocusedElement.value && currentFocusedElement.value !== element) {
      focusHistory.value.push(currentFocusedElement.value)
      // 히스토리 크기 제한 (최대 10개)
      if (focusHistory.value.length > 10) {
        focusHistory.value.shift()
      }
    }

    currentFocusedElement.value = element

    try {
      element.focus(options)

      // 포커스 가시성 강제 설정
      if (options.focusVisible) {
        element.setAttribute('data-focus-visible', 'true')
      }
    } catch (error) {
      console.warn('Failed to focus element:', error)
    }
  }

  /**
   * 이전 포커스로 되돌리기
   */
  const focusPrevious = (): void => {
    if (focusHistory.value.length > 0) {
      const previousElement = focusHistory.value.pop()
      if (previousElement && isFocusable(previousElement)) {
        focusElement(previousElement)
      }
    }
  }

  /**
   * 첫 번째 포커스 가능한 요소로 이동
   */
  const focusFirst = (container?: HTMLElement): void => {
    const focusableElements = getFocusableElements(container)
    if (focusableElements.length > 0) {
      focusElement(focusableElements[0])
    }
  }

  /**
   * 마지막 포커스 가능한 요소로 이동
   */
  const focusLast = (container?: HTMLElement): void => {
    const focusableElements = getFocusableElements(container)
    if (focusableElements.length > 0) {
      focusElement(focusableElements[focusableElements.length - 1])
    }
  }

  /**
   * 다음 포커스 가능한 요소로 이동
   */
  const focusNext = (currentElement?: HTMLElement, container?: HTMLElement): void => {
    const current = currentElement || currentFocusedElement.value || document.activeElement as HTMLElement
    if (!current) {
      focusFirst(container)
      return
    }

    const focusableElements = getFocusableElements(container)
    const currentIndex = focusableElements.indexOf(current)

    if (currentIndex >= 0 && currentIndex < focusableElements.length - 1) {
      focusElement(focusableElements[currentIndex + 1])
    } else {
      // 마지막 요소인 경우 첫 번째로 순환
      focusFirst(container)
    }
  }

  /**
   * 이전 포커스 가능한 요소로 이동
   */
  const focusPreviousElement = (currentElement?: HTMLElement, container?: HTMLElement): void => {
    const current = currentElement || currentFocusedElement.value || document.activeElement as HTMLElement
    if (!current) {
      focusLast(container)
      return
    }

    const focusableElements = getFocusableElements(container)
    const currentIndex = focusableElements.indexOf(current)

    if (currentIndex > 0) {
      focusElement(focusableElements[currentIndex - 1])
    } else {
      // 첫 번째 요소인 경우 마지막으로 순환
      focusLast(container)
    }
  }

  /**
   * 스킵 링크 대상 등록
   */
  const registerSkipTarget = (id: string, label: string, element?: HTMLElement): void => {
    const existingIndex = skipLinkTargets.value.findIndex(target => target.id === id)
    const target: SkipLinkTarget = { id, label, element }

    if (existingIndex >= 0) {
      skipLinkTargets.value[existingIndex] = target
    } else {
      skipLinkTargets.value.push(target)
    }
  }

  /**
   * 스킵 링크 대상 제거
   */
  const unregisterSkipTarget = (id: string): void => {
    const index = skipLinkTargets.value.findIndex(target => target.id === id)
    if (index >= 0) {
      skipLinkTargets.value.splice(index, 1)
    }
  }

  /**
   * 스킵 링크로 이동
   */
  const skipToTarget = (targetId: string): void => {
    const target = skipLinkTargets.value.find(t => t.id === targetId)
    if (!target) {
      return
    }

    let element = target.element
    if (!element) {
      element = document.getElementById(targetId)
    }

    if (element) {
      // 스킵 대상이 포커스 가능하지 않은 경우 tabindex 추가
      if (!isFocusable(element)) {
        element.tabIndex = -1
      }

      focusElement(element)

      // 화면 중앙으로 스크롤
      element.scrollIntoView({
        behavior: 'smooth',
        block: 'center',
      })
    }
  }

  /**
   * 포커스 트랩 생성
   */
  const createFocusTrap = (container: HTMLElement) => {
    let isActive = false
    const focusableElements = () => getFocusableElements(container)

    const handleKeydown = (event: KeyboardEvent): void => {
      if (!isActive || !keyboardNavigationEnabled.value) return

      if (event.key === 'Tab') {
        const elements = focusableElements()
        if (elements.length === 0) return

        const firstElement = elements[0]
        const lastElement = elements[elements.length - 1]
        const activeElement = document.activeElement as HTMLElement

        if (event.shiftKey) {
          // Shift + Tab
          if (activeElement === firstElement) {
            event.preventDefault()
            focusElement(lastElement)
          }
        } else {
          // Tab
          if (activeElement === lastElement) {
            event.preventDefault()
            focusElement(firstElement)
          }
        }
      } else if (event.key === 'Escape') {
        // ESC로 트랩 해제
        deactivate()
      }
    }

    const activate = (): void => {
      isActive = true
      document.addEventListener('keydown', handleKeydown)

      // 첫 번째 포커스 가능한 요소로 이동
      nextTick(() => {
        focusFirst(container)
      })
    }

    const deactivate = (): void => {
      isActive = false
      document.removeEventListener('keydown', handleKeydown)

      // 이전 포커스로 복원
      focusPrevious()
    }

    return {
      activate,
      deactivate,
      isActive: () => isActive,
    }
  }

  /**
   * 키보드 이벤트 핸들러 설정
   */
  const setupKeyboardNavigation = (): void => {
    const handleKeydown = (event: KeyboardEvent): void => {
      if (!keyboardNavigationEnabled.value) return

      // 전역 키보드 단축키
      switch (event.key) {
        case 'F6':
          // 영역 간 이동
          event.preventDefault()
          focusNext()
          break

        case 'Alt':
          if (event.altKey && event.key === '1') {
            // Alt + 1: 메인 콘텐츠로 스킵
            event.preventDefault()
            skipToTarget('main-content')
          }
          break
      }
    }

    document.addEventListener('keydown', handleKeydown)

    // 포커스 이벤트 추적
    const handleFocus = (event: FocusEvent): void => {
      const target = event.target as HTMLElement
      if (target && isFocusable(target)) {
        currentFocusedElement.value = target
      }
    }

    document.addEventListener('focusin', handleFocus)

    // 정리 함수 반환
    return () => {
      document.removeEventListener('keydown', handleKeydown)
      document.removeEventListener('focusin', handleFocus)
    }
  }

  /**
   * 접근성 어나운스먼트
   */
  const announce = (message: string, priority: 'polite' | 'assertive' = 'polite'): void => {
    const announcer = document.createElement('div')
    announcer.setAttribute('aria-live', priority)
    announcer.setAttribute('aria-atomic', 'true')
    announcer.style.position = 'absolute'
    announcer.style.left = '-10000px'
    announcer.style.width = '1px'
    announcer.style.height = '1px'
    announcer.style.overflow = 'hidden'

    document.body.appendChild(announcer)

    // 약간의 지연 후 메시지 설정 (스크린 리더가 읽을 수 있도록)
    setTimeout(() => {
      announcer.textContent = message
    }, 100)

    // 5초 후 제거
    setTimeout(() => {
      if (announcer.parentNode) {
        announcer.parentNode.removeChild(announcer)
      }
    }, 5000)
  }

  /**
   * 라우트 변경 어나운스먼트
   */
  const announceRouteChange = (routeName: string): void => {
    announce(`페이지가 ${routeName}(으)로 변경되었습니다.`, 'polite')
  }

  /**
   * 에러 어나운스먼트
   */
  const announceError = (error: string): void => {
    announce(`오류: ${error}`, 'assertive')
  }

  // 생명주기 관리
  let cleanup: (() => void) | null = null

  onMounted(() => {
    cleanup = setupKeyboardNavigation()

    // 기본 스킵 링크 대상 등록
    registerSkipTarget('main-content', '메인 콘텐츠')
    registerSkipTarget('navigation', '네비게이션')
    registerSkipTarget('search', '검색')
  })

  onUnmounted(() => {
    if (cleanup) {
      cleanup()
    }
  })

  return {
    // 상태
    currentFocusedElement: computed(() => currentFocusedElement.value),
    skipLinkTargets: computed(() => skipLinkTargets.value),
    keyboardNavigationEnabled: computed(() => keyboardNavigationEnabled.value),

    // 포커스 관리
    focusElement,
    focusPrevious,
    focusFirst,
    focusLast,
    focusNext,
    focusPreviousElement,
    isFocusable,
    getFocusableElements,

    // 스킵 링크
    registerSkipTarget,
    unregisterSkipTarget,
    skipToTarget,

    // 포커스 트랩
    createFocusTrap,

    // 어나운스먼트
    announce,
    announceRouteChange,
    announceError,

    // 설정
    enableKeyboardNavigation: () => { keyboardNavigationEnabled.value = true },
    disableKeyboardNavigation: () => { keyboardNavigationEnabled.value = false },
  }
}

/**
 * 전역 접근성 네비게이션 상태
 */
export const globalAccessibilityNavigation = {
  currentFocusedElement,
  skipLinkTargets,
  keyboardNavigationEnabled,
}