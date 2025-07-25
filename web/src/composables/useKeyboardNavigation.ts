import { type Ref, nextTick, ref } from 'vue'

export interface KeyboardNavigationOptions {
  loop?: boolean; // 순환 네비게이션 여부
  vertical?: boolean; // 세로 방향 네비게이션
  horizontal?: boolean; // 가로 방향 네비게이션
  disabled?: Ref<boolean>; // 비활성화 상태
  onFocus?: (index: number, element: HTMLElement) => void; // 포커스 콜백
  onSelect?: (index: number, element: HTMLElement) => void; // 선택 콜백
}

export function useKeyboardNavigation(
  elements: Ref<HTMLElement[]>,
  options: KeyboardNavigationOptions = {},
) {
  const {
    loop = true,
    vertical = true,
    horizontal = false,
    disabled = ref(false),
    onFocus,
    onSelect,
  } = options

  const currentIndex = ref(-1)
  const isNavigating = ref(false)

  // 활성화된 요소 가져오기
  const getActiveElements = (): HTMLElement[] => {
    return elements.value.filter(el =>
      el &&
      !el.hasAttribute('disabled') &&
      el.tabIndex !== -1 &&
      !el.classList.contains('disabled'),
    )
  }

  // 다음 유효한 인덱스 계산
  const getNextIndex = (direction: 'next' | 'prev'): number => {
    const activeElements = getActiveElements()
    const maxIndex = activeElements.length - 1

    if (maxIndex < 0) return -1

    let nextIndex = currentIndex.value

    if (direction === 'next') {
      nextIndex = nextIndex < maxIndex ? nextIndex + 1 : (loop ? 0 : maxIndex)
    } else {
      nextIndex = nextIndex > 0 ? nextIndex - 1 : (loop ? maxIndex : 0)
    }

    return nextIndex
  }

  // 특정 인덱스로 포커스 이동
  const focusIndex = async (index: number): Promise<void> => {
    if (disabled.value) return

    const activeElements = getActiveElements()
    if (index < 0 || index >= activeElements.length) return

    const element = activeElements[index]
    if (!element) return

    currentIndex.value = index
    isNavigating.value = true

    await nextTick()

    try {
      element.focus()
      onFocus?.(index, element)
    } catch (error) {
      console.warn('Focus 설정 실패:', error)
    } finally {
      isNavigating.value = false
    }
  }

  // 다음 요소로 포커스 이동
  const focusNext = async (): Promise<void> => {
    const nextIndex = getNextIndex('next')
    await focusIndex(nextIndex)
  }

  // 이전 요소로 포커스 이동
  const focusPrevious = async (): Promise<void> => {
    const nextIndex = getNextIndex('prev')
    await focusIndex(nextIndex)
  }

  // 첫 번째 요소로 포커스 이동
  const focusFirst = async (): Promise<void> => {
    await focusIndex(0)
  }

  // 마지막 요소로 포커스 이동
  const focusLast = async (): Promise<void> => {
    const activeElements = getActiveElements()
    await focusIndex(activeElements.length - 1)
  }

  // 현재 포커스된 요소 선택
  const selectCurrent = (): void => {
    if (disabled.value || currentIndex.value < 0) return

    const activeElements = getActiveElements()
    const element = activeElements[currentIndex.value]

    if (element) {
      onSelect?.(currentIndex.value, element)
    }
  }

  // 키보드 이벤트 핸들러
  const handleKeydown = async (event: KeyboardEvent): Promise<void> => {
    if (disabled.value) return

    let handled = false

    switch (event.key) {
      case 'ArrowDown':
        if (vertical) {
          event.preventDefault()
          await focusNext()
          handled = true
        }
        break

      case 'ArrowUp':
        if (vertical) {
          event.preventDefault()
          await focusPrevious()
          handled = true
        }
        break

      case 'ArrowRight':
        if (horizontal) {
          event.preventDefault()
          await focusNext()
          handled = true
        }
        break

      case 'ArrowLeft':
        if (horizontal) {
          event.preventDefault()
          await focusPrevious()
          handled = true
        }
        break

      case 'Home':
        event.preventDefault()
        await focusFirst()
        handled = true
        break

      case 'End':
        event.preventDefault()
        await focusLast()
        handled = true
        break

      case 'Enter':
      case ' ':
        event.preventDefault()
        selectCurrent()
        handled = true
        break

      case 'Tab':
        // Tab 키는 기본 동작을 유지하되, 현재 네비게이션 상태 초기화
        currentIndex.value = -1
        break
    }

    return handled
  }

  // 요소의 현재 포커스 상태 확인
  const updateCurrentIndex = (): void => {
    const activeElements = getActiveElements()
    const focusedElement = document.activeElement as HTMLElement

    currentIndex.value = activeElements.findIndex(el => el === focusedElement)
  }

  // 포커스 이벤트 핸들러
  const handleFocus = (event: FocusEvent): void => {
    if (isNavigating.value) return // 프로그래밍 방식 포커스는 무시

    updateCurrentIndex()
  }

  // 특정 요소를 찾아서 포커스
  const focusElement = async (predicate: (element: HTMLElement, index: number) => boolean): Promise<boolean> => {
    const activeElements = getActiveElements()
    const targetIndex = activeElements.findIndex(predicate)

    if (targetIndex >= 0) {
      await focusIndex(targetIndex)
      return true
    }

    return false
  }

  // 텍스트 검색으로 포커스 (타이핑 네비게이션)
  const focusSearchResults = (() => {
    let searchString = ''
    let searchTimeout: NodeJS.Timeout | null = null

    return async (key: string): Promise<boolean> => {
      // 기존 타이머 클리어
      if (searchTimeout) {
        clearTimeout(searchTimeout)
      }

      // 검색 문자열 업데이트
      searchString += key.toLowerCase()

      // 검색 결과 찾기
      const found = await focusElement((element, index) => {
        const text = element.textContent?.toLowerCase() || ''
        return text.startsWith(searchString)
      })

      // 1초 후 검색 문자열 초기화
      searchTimeout = setTimeout(() => {
        searchString = ''
      }, 1000)

      return found
    }
  })()

  // 현재 상태 리셋
  const reset = (): void => {
    currentIndex.value = -1
    isNavigating.value = false
  }

  return {
    // 상태
    currentIndex,
    isNavigating,

    // 메서드
    focusIndex,
    focusNext,
    focusPrevious,
    focusFirst,
    focusLast,
    focusElement,
    focusSearchResults,
    selectCurrent,
    reset,
    updateCurrentIndex,

    // 이벤트 핸들러
    handleKeydown,
    handleFocus,

    // 유틸리티
    getActiveElements,
  }
}