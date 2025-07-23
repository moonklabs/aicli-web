/**
 * 키보드 단축키 관리를 위한 컴포저블
 */

import { onMounted, onUnmounted } from 'vue'

export interface KeyboardShortcut {
  key: string
  ctrlKey?: boolean
  metaKey?: boolean
  shiftKey?: boolean
  altKey?: boolean
  action: () => void
  description: string
  preventDefault?: boolean
}

export interface KeyboardShortcutOptions {
  enabled?: boolean
  preventDefault?: boolean
  stopPropagation?: boolean
}

export function useKeyboardShortcuts(
  shortcuts: KeyboardShortcut[],
  options: KeyboardShortcutOptions = {},
) {
  const {
    enabled = true,
    preventDefault = true,
    stopPropagation = false,
  } = options

  const handleKeyDown = (event: KeyboardEvent) => {
    if (!enabled) return

    // 입력 필드에서는 단축키 비활성화
    const target = event.target as HTMLElement
    if (
      target.tagName === 'INPUT' ||
      target.tagName === 'TEXTAREA' ||
      target.contentEditable === 'true'
    ) {
      // 특별한 경우만 허용 (Ctrl+C, Ctrl+V 등은 자연스럽게 동작)
      if (!event.ctrlKey && !event.metaKey) {
        return
      }
    }

    for (const shortcut of shortcuts) {
      const {
        key,
        ctrlKey = false,
        metaKey = false,
        shiftKey = false,
        altKey = false,
        action,
        preventDefault: shortcutPreventDefault = preventDefault,
      } = shortcut

      // 키 조합 매칭
      const keyMatches = event.key.toLowerCase() === key.toLowerCase()
      const ctrlMatches = event.ctrlKey === ctrlKey
      const metaMatches = event.metaKey === metaKey
      const shiftMatches = event.shiftKey === shiftKey
      const altMatches = event.altKey === altKey

      if (keyMatches && ctrlMatches && metaMatches && shiftMatches && altMatches) {
        if (shortcutPreventDefault) {
          event.preventDefault()
        }
        if (stopPropagation) {
          event.stopPropagation()
        }

        action()
        break
      }
    }
  }

  onMounted(() => {
    document.addEventListener('keydown', handleKeyDown)
  })

  onUnmounted(() => {
    document.removeEventListener('keydown', handleKeyDown)
  })

  return {
    shortcuts,
  }
}

// 터미널 전용 키보드 단축키
export function useTerminalKeyboardShortcuts(handlers: {
  onClear?: () => void
  onStop?: () => void
  onSearch?: () => void
  onFullscreen?: () => void
  onScrollToTop?: () => void
  onScrollToBottom?: () => void
  onCopy?: () => void
  onPaste?: () => void
  onSelectAll?: () => void
  onZoomIn?: () => void
  onZoomOut?: () => void
  onResetZoom?: () => void
}) {
  const shortcuts: KeyboardShortcut[] = [
    // 터미널 제어
    {
      key: 'l',
      ctrlKey: true,
      action: handlers.onClear || (() => {}),
      description: '터미널 클리어',
    },
    {
      key: 'c',
      ctrlKey: true,
      action: handlers.onStop || (() => {}),
      description: '실행 중지',
    },
    {
      key: 'f',
      ctrlKey: true,
      action: handlers.onSearch || (() => {}),
      description: '검색',
    },
    {
      key: 'F11',
      action: handlers.onFullscreen || (() => {}),
      description: '전체화면 토글',
    },

    // 스크롤 제어
    {
      key: 'Home',
      ctrlKey: true,
      action: handlers.onScrollToTop || (() => {}),
      description: '맨 위로 스크롤',
    },
    {
      key: 'End',
      ctrlKey: true,
      action: handlers.onScrollToBottom || (() => {}),
      description: '맨 아래로 스크롤',
    },

    // 클립보드
    {
      key: 'c',
      ctrlKey: true,
      shiftKey: true,
      action: handlers.onCopy || (() => {}),
      description: '선택된 텍스트 복사',
    },
    {
      key: 'v',
      ctrlKey: true,
      shiftKey: true,
      action: handlers.onPaste || (() => {}),
      description: '텍스트 붙여넣기',
    },
    {
      key: 'a',
      ctrlKey: true,
      action: handlers.onSelectAll || (() => {}),
      description: '모든 텍스트 선택',
    },

    // 확대/축소
    {
      key: '=',
      ctrlKey: true,
      action: handlers.onZoomIn || (() => {}),
      description: '확대',
    },
    {
      key: '-',
      ctrlKey: true,
      action: handlers.onZoomOut || (() => {}),
      description: '축소',
    },
    {
      key: '0',
      ctrlKey: true,
      action: handlers.onResetZoom || (() => {}),
      description: '확대/축소 리셋',
    },
  ]

  return useKeyboardShortcuts(shortcuts, {
    enabled: true,
    preventDefault: true,
  })
}

// 검색 전용 키보드 단축키
export function useSearchKeyboardShortcuts(handlers: {
  onNext?: () => void
  onPrevious?: () => void
  onClose?: () => void
  onCaseSensitive?: () => void
  onRegex?: () => void
}) {
  const shortcuts: KeyboardShortcut[] = [
    {
      key: 'Enter',
      action: handlers.onNext || (() => {}),
      description: '다음 결과',
    },
    {
      key: 'Enter',
      shiftKey: true,
      action: handlers.onPrevious || (() => {}),
      description: '이전 결과',
    },
    {
      key: 'Escape',
      action: handlers.onClose || (() => {}),
      description: '검색 닫기',
    },
    {
      key: 'i',
      ctrlKey: true,
      action: handlers.onCaseSensitive || (() => {}),
      description: '대소문자 구분 토글',
    },
    {
      key: 'r',
      ctrlKey: true,
      action: handlers.onRegex || (() => {}),
      description: '정규식 모드 토글',
    },
  ]

  return useKeyboardShortcuts(shortcuts, {
    enabled: true,
    preventDefault: true,
  })
}

// 단축키 도움말 텍스트 생성
export function generateShortcutHelp(shortcuts: KeyboardShortcut[]): string {
  return shortcuts
    .map(shortcut => {
      const keys = []
      if (shortcut.ctrlKey) keys.push('Ctrl')
      if (shortcut.metaKey) keys.push('Cmd')
      if (shortcut.shiftKey) keys.push('Shift')
      if (shortcut.altKey) keys.push('Alt')
      keys.push(shortcut.key)

      return `${keys.join('+')} - ${shortcut.description}`
    })
    .join('\n')
}

// 브라우저별 키 정규화
export function normalizeKey(event: KeyboardEvent): string {
  // 브라우저별 키 이름 차이 처리
  const keyMap: Record<string, string> = {
    ' ': 'Space',
    'Esc': 'Escape',
    'Del': 'Delete',
    'Up': 'ArrowUp',
    'Down': 'ArrowDown',
    'Left': 'ArrowLeft',
    'Right': 'ArrowRight',
  }

  return keyMap[event.key] || event.key
}

// 키 조합을 문자열로 변환
export function formatKeyCombo(shortcut: KeyboardShortcut): string {
  const parts = []

  if (shortcut.ctrlKey) parts.push('Ctrl')
  if (shortcut.metaKey) parts.push('Cmd')
  if (shortcut.shiftKey) parts.push('Shift')
  if (shortcut.altKey) parts.push('Alt')

  // 특수 키 이름 정리
  let keyName = shortcut.key
  const specialKeys: Record<string, string> = {
    ' ': 'Space',
    'Escape': 'Esc',
    'Delete': 'Del',
    'ArrowUp': '↑',
    'ArrowDown': '↓',
    'ArrowLeft': '←',
    'ArrowRight': '→',
    'F11': 'F11',
    'Home': 'Home',
    'End': 'End',
    'Enter': '⏎',
  }

  keyName = specialKeys[keyName] || keyName.toUpperCase()
  parts.push(keyName)

  return parts.join('+')
}

// 플랫폼별 modifier key 감지
export function getModifierKey(): 'Ctrl' | 'Cmd' {
  return navigator.platform.toLowerCase().includes('mac') ? 'Cmd' : 'Ctrl'
}

// 접근성을 위한 키보드 네비게이션 헬퍼
export function useKeyboardNavigation(elements: HTMLElement[]) {
  let currentIndex = 0

  const focusElement = (index: number) => {
    if (index >= 0 && index < elements.length) {
      elements[index]?.focus()
      currentIndex = index
    }
  }

  const handleKeyDown = (event: KeyboardEvent) => {
    switch (event.key) {
      case 'ArrowDown':
        event.preventDefault()
        focusElement((currentIndex + 1) % elements.length)
        break
      case 'ArrowUp':
        event.preventDefault()
        focusElement(currentIndex === 0 ? elements.length - 1 : currentIndex - 1)
        break
      case 'Home':
        event.preventDefault()
        focusElement(0)
        break
      case 'End':
        event.preventDefault()
        focusElement(elements.length - 1)
        break
    }
  }

  return {
    focusElement,
    handleKeyDown,
    currentIndex: () => currentIndex,
  }
}