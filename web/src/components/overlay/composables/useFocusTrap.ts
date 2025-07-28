import { ref, watch, onMounted, onUnmounted, type Ref } from 'vue'

/**
 * 포커스 가능한 요소 선택자
 */
const FOCUSABLE_SELECTORS = [
  'a[href]:not([disabled])',
  'button:not([disabled])',
  'textarea:not([disabled])',
  'input:not([disabled])',
  'select:not([disabled])',
  '[tabindex]:not([tabindex="-1"]):not([disabled])',
].join(',')

/**
 * 포커스 트랩 훅
 * 오버레이 내에서 포커스가 순환하도록 관리
 */
export function useFocusTrap(
  containerRef: Ref<HTMLElement | null>,
  options: {
    initialFocus?: Ref<HTMLElement | null>
    restoreFocus?: boolean
    autoFocus?: boolean
  } = {}
) {
  const { initialFocus, restoreFocus = true, autoFocus = true } = options
  
  // 포커스 복원을 위한 이전 활성 요소 저장
  const previousActiveElement = ref<HTMLElement | null>(null)
  
  /**
   * 포커스 가능한 요소들 가져오기
   */
  function getFocusableElements(): HTMLElement[] {
    if (!containerRef.value) return []
    
    const elements = containerRef.value.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTORS)
    return Array.from(elements).filter(el => {
      // 실제로 포커스 가능한지 확인
      return el.offsetParent !== null && !el.hasAttribute('disabled')
    })
  }
  
  /**
   * 첫 번째 포커스 가능한 요소로 포커스 이동
   */
  function focusFirstElement(): void {
    const elements = getFocusableElements()
    if (elements.length > 0) {
      elements[0].focus()
    }
  }
  
  /**
   * 마지막 포커스 가능한 요소로 포커스 이동
   */
  function focusLastElement(): void {
    const elements = getFocusableElements()
    if (elements.length > 0) {
      elements[elements.length - 1].focus()
    }
  }
  
  /**
   * Tab 키 핸들러
   */
  function handleTab(event: KeyboardEvent): void {
    if (event.key !== 'Tab' || !containerRef.value) return
    
    const focusableElements = getFocusableElements()
    if (focusableElements.length === 0) {
      event.preventDefault()
      return
    }
    
    const activeElement = document.activeElement as HTMLElement
    const firstElement = focusableElements[0]
    const lastElement = focusableElements[focusableElements.length - 1]
    
    // Shift + Tab
    if (event.shiftKey) {
      if (activeElement === firstElement || !containerRef.value.contains(activeElement)) {
        event.preventDefault()
        lastElement.focus()
      }
    }
    // Tab
    else {
      if (activeElement === lastElement || !containerRef.value.contains(activeElement)) {
        event.preventDefault()
        firstElement.focus()
      }
    }
  }
  
  /**
   * 초기 포커스 설정
   */
  function setInitialFocus(): void {
    if (!autoFocus) return
    
    // 초기 포커스 요소가 지정된 경우
    if (initialFocus?.value) {
      initialFocus.value.focus()
      return
    }
    
    // 그렇지 않으면 첫 번째 포커스 가능한 요소로
    focusFirstElement()
  }
  
  /**
   * 포커스 트랩 활성화
   */
  function activate(): void {
    // 현재 활성 요소 저장
    if (restoreFocus) {
      previousActiveElement.value = document.activeElement as HTMLElement
    }
    
    // 초기 포커스 설정
    setInitialFocus()
    
    // 이벤트 리스너 등록
    document.addEventListener('keydown', handleTab)
  }
  
  /**
   * 포커스 트랩 비활성화
   */
  function deactivate(): void {
    // 이벤트 리스너 제거
    document.removeEventListener('keydown', handleTab)
    
    // 포커스 복원
    if (restoreFocus && previousActiveElement.value) {
      previousActiveElement.value.focus()
    }
  }
  
  // 컨테이너가 마운트되면 활성화
  onMounted(() => {
    if (containerRef.value) {
      activate()
    }
  })
  
  // 컨테이너가 변경되면 재활성화
  watch(containerRef, (newContainer, oldContainer) => {
    if (oldContainer) {
      deactivate()
    }
    if (newContainer) {
      activate()
    }
  })
  
  // 언마운트 시 비활성화
  onUnmounted(() => {
    deactivate()
  })
  
  return {
    activate,
    deactivate,
    focusFirstElement,
    focusLastElement,
    getFocusableElements,
  }
}