/**
 * 접근성 컴포넌트 인덱스
 */

// 테마 및 접근성 설정
export { default as ThemeToggle } from './ThemeToggle.vue'

// 네비게이션 및 포커스 관리
export { default as SkipLinks } from './SkipLinks.vue'

// 개발자 도구
export { default as AccessibilityChecker } from './AccessibilityChecker.vue'

// 접근성 컴포넌트 타입
export interface AccessibilityComponentProps {
  'aria-label'?: string
  'aria-describedby'?: string
  'aria-labelledby'?: string
  'aria-hidden'?: boolean
  'aria-expanded'?: boolean
  'aria-disabled'?: boolean
  'aria-required'?: boolean
  'aria-invalid'?: boolean
  'aria-live'?: 'off' | 'polite' | 'assertive'
  role?: string
  tabindex?: number
}

// 접근성 유틸리티 함수
export const accessibilityUtils = {
  /**
   * 요소가 포커스 가능한지 확인
   */
  isFocusable: (element: HTMLElement): boolean => {
    if (!element || element.getAttribute('aria-hidden') === 'true') {
      return false
    }

    if (element.hasAttribute('disabled') || element.getAttribute('aria-disabled') === 'true') {
      return false
    }

    const style = window.getComputedStyle(element)
    if (style.display === 'none' || style.visibility === 'hidden' || style.opacity === '0') {
      return false
    }

    const focusableSelectors = [
      'a[href]',
      'button:not([disabled])',
      'input:not([disabled]):not([type="hidden"])',
      'select:not([disabled])',
      'textarea:not([disabled])',
      '[tabindex]:not([tabindex="-1"])',
      '[contenteditable="true"]',
    ]

    return focusableSelectors.some(selector => element.matches(selector))
  },

  /**
   * 고유한 ID 생성
   */
  generateId: (prefix = 'accessibility'): string => {
    return `${prefix}-${Math.random().toString(36).substr(2, 9)}`
  },

  /**
   * ARIA 라이브 메시지 알림
   */
  announce: (message: string, urgency: 'polite' | 'assertive' = 'polite'): void => {
    const announcer = document.createElement('div')
    announcer.setAttribute('aria-live', urgency)
    announcer.setAttribute('aria-atomic', 'true')
    announcer.style.position = 'absolute'
    announcer.style.left = '-10000px'
    announcer.style.width = '1px'
    announcer.style.height = '1px'
    announcer.style.overflow = 'hidden'

    document.body.appendChild(announcer)

    setTimeout(() => {
      announcer.textContent = message
    }, 100)

    setTimeout(() => {
      if (announcer.parentNode) {
        announcer.parentNode.removeChild(announcer)
      }
    }, 5000)
  },

  /**
   * 색상 대비 비율 계산 (간단한 버전)
   */
  getContrastRatio: (color1: string, color2: string): number => {
    // 실제 구현에서는 더 정확한 WCAG 대비 비율 계산이 필요
    // 여기서는 간단한 예시만 제공
    return 4.5 // WCAG AA 기준
  },

  /**
   * 키보드 이벤트가 액션 키인지 확인
   */
  isActionKey: (event: KeyboardEvent): boolean => {
    return event.key === 'Enter' || event.key === ' '
  },

  /**
   * 요소를 화면 중앙으로 스크롤
   */
  scrollIntoViewCentered: (element: HTMLElement): void => {
    element.scrollIntoView({
      behavior: 'smooth',
      block: 'center',
      inline: 'center',
    })
  },
}