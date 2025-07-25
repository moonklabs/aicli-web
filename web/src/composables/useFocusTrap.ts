import { type Ref, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'

export interface FocusTrapOptions {
  initialFocus?: string | HTMLElement | (() => HTMLElement | null); // 초기 포커스 요소
  returnFocus?: HTMLElement | boolean; // 트랩 해제 시 포커스 복귀 요소
  allowOutsideClick?: boolean; // 외부 클릭 허용 여부
  escapeKeyDeactivates?: boolean; // Escape 키로 해제 여부
  clickOutsideDeactivates?: boolean; // 외부 클릭으로 해제 여부
  onActivate?: () => void; // 활성화 콜백
  onDeactivate?: () => void; // 비활성화 콜백
}

// 포커스 가능한 요소들의 셀렉터
const FOCUSABLE_SELECTOR = [
  'button:not([disabled])',
  'input:not([disabled])',
  'textarea:not([disabled])',
  'select:not([disabled])',
  'a[href]',
  'area[href]',
  'iframe',
  'object',
  'embed',
  '[tabindex]:not([tabindex="-1"])',
  '[contenteditable="true"]',
].join(', ')

export function useFocusTrap(
  target: Ref<HTMLElement | undefined>,
  active: Ref<boolean>,
  options: FocusTrapOptions = {},
) {
  const {
    initialFocus,
    returnFocus = true,
    allowOutsideClick = false,
    escapeKeyDeactivates = true,
    clickOutsideDeactivates = false,
    onActivate,
    onDeactivate,
  } = options

  const isActive = ref(false)
  const previousActiveElement = ref<HTMLElement | null>(null)
  const focusableElements = ref<HTMLElement[]>([])

  // 포커스 가능한 요소들 찾기
  const findFocusableElements = (): HTMLElement[] => {
    if (!target.value) return []

    const elements = Array.from(
      target.value.querySelectorAll(FOCUSABLE_SELECTOR),
    ) as HTMLElement[]

    return elements.filter(element => {
      // 숨겨진 요소 제외
      if (element.offsetWidth === 0 && element.offsetHeight === 0) {
        return false
      }

      // display: none 요소 제외
      const style = window.getComputedStyle(element)
      if (style.display === 'none' || style.visibility === 'hidden') {
        return false
      }

      return true
    })
  }

  // 초기 포커스 요소 결정
  const getInitialFocusElement = (): HTMLElement | null => {
    if (!target.value) return null

    // 옵션에서 지정된 초기 포커스
    if (initialFocus) {
      if (typeof initialFocus === 'string') {
        return target.value.querySelector(initialFocus) as HTMLElement
      } else if (typeof initialFocus === 'function') {
        return initialFocus()
      } else {
        return initialFocus
      }
    }

    // 첫 번째 포커스 가능한 요소
    const focusable = findFocusableElements()
    return focusable[0] || null
  }

  // 포커스 트랩 활성화
  const activate = async (): Promise<void> => {
    if (isActive.value || !target.value) return

    // 현재 포커스된 요소 저장
    previousActiveElement.value = document.activeElement as HTMLElement

    // 포커스 가능한 요소들 업데이트
    focusableElements.value = findFocusableElements()

    // 이벤트 리스너 등록
    document.addEventListener('keydown', handleKeydown, true)
    document.addEventListener('focusin', handleFocusIn, true)

    if (clickOutsideDeactivates) {
      document.addEventListener('mousedown', handleMouseDown, true)
      document.addEventListener('touchstart', handleMouseDown, true)
    }

    isActive.value = true

    // 초기 포커스 설정
    await nextTick()

    const initialElement = getInitialFocusElement()
    if (initialElement) {
      initialElement.focus()
    }

    onActivate?.()
  }

  // 포커스 트랩 비활성화
  const deactivate = async (): Promise<void> => {
    if (!isActive.value) return

    // 이벤트 리스너 제거
    document.removeEventListener('keydown', handleKeydown, true)
    document.removeEventListener('focusin', handleFocusIn, true)
    document.removeEventListener('mousedown', handleMouseDown, true)
    document.removeEventListener('touchstart', handleMouseDown, true)

    isActive.value = false

    // 포커스 복귀
    if (returnFocus && previousActiveElement.value) {
      await nextTick()
      previousActiveElement.value.focus()
    }

    previousActiveElement.value = null
    focusableElements.value = []

    onDeactivate?.()
  }

  // 키보드 이벤트 핸들러
  const handleKeydown = (event: KeyboardEvent): void => {
    if (!isActive.value || !target.value) return

    // Escape 키 처리
    if (event.key === 'Escape' && escapeKeyDeactivates) {
      event.preventDefault()
      deactivate()
      return
    }

    // Tab 키 처리
    if (event.key === 'Tab') {
      handleTabKey(event)
    }
  }

  // Tab 키 네비게이션 처리
  const handleTabKey = (event: KeyboardEvent): void => {
    const focusable = findFocusableElements()
    if (focusable.length === 0) {
      event.preventDefault()
      return
    }

    const firstElement = focusable[0]
    const lastElement = focusable[focusable.length - 1]
    const activeElement = document.activeElement as HTMLElement

    if (event.shiftKey) {
      // Shift+Tab (역방향)
      if (activeElement === firstElement) {
        event.preventDefault()
        lastElement.focus()
      }
    } else {
      // Tab (정방향)
      if (activeElement === lastElement) {
        event.preventDefault()
        firstElement.focus()
      }
    }
  }

  // 포커스 이벤트 핸들러
  const handleFocusIn = (event: FocusEvent): void => {
    if (!isActive.value || !target.value) return

    const focusedElement = event.target as HTMLElement

    // 트랩 영역 내부인지 확인
    if (target.value.contains(focusedElement)) {
      return
    }

    // 트랩 영역 외부로 포커스가 이동한 경우
    event.preventDefault()
    event.stopPropagation()

    // 첫 번째 포커스 가능한 요소로 포커스 이동
    const focusable = findFocusableElements()
    if (focusable.length > 0) {
      focusable[0].focus()
    }
  }

  // 마우스 이벤트 핸들러
  const handleMouseDown = (event: Event): void => {
    if (!isActive.value || !target.value) return

    const clickedElement = event.target as HTMLElement

    // 트랩 영역 외부 클릭 시 비활성화
    if (!target.value.contains(clickedElement)) {
      if (clickOutsideDeactivates) {
        deactivate()
      } else if (!allowOutsideClick) {
        event.preventDefault()
        event.stopPropagation()
      }
    }
  }

  // 포커스 가능한 요소 목록 업데이트
  const updateFocusableElements = (): void => {
    focusableElements.value = findFocusableElements()
  }

  // active 상태 변경 감지
  watch(active, async (newActive) => {
    if (newActive) {
      await activate()
    } else {
      await deactivate()
    }
  }, { immediate: true })

  // target 변경 감지
  watch(target, () => {
    if (isActive.value) {
      updateFocusableElements()
    }
  })

  // 컴포넌트 언마운트 시 정리
  onUnmounted(() => {
    if (isActive.value) {
      deactivate()
    }
  })

  return {
    // 상태
    isActive,
    focusableElements,

    // 메서드
    activate,
    deactivate,
    updateFocusableElements,
    findFocusableElements,

    // 유틸리티
    getInitialFocusElement,
  }
}