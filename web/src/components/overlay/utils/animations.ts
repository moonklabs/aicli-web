/**
 * 오버레이 애니메이션 유틸리티
 * 일관된 애니메이션 효과를 위한 프리셋 제공
 */

export type AnimationType = 'fade' | 'scale' | 'slide' | 'zoom'
export type AnimationDirection = 'top' | 'right' | 'bottom' | 'left'

/**
 * 애니메이션 기본 설정
 */
export const ANIMATION_DURATION = {
  fast: 150,
  normal: 300,
  slow: 500,
} as const

/**
 * 애니메이션 이징 함수
 */
export const ANIMATION_EASING = {
  easeOut: 'cubic-bezier(0.25, 0.46, 0.45, 0.94)',
  easeInOut: 'cubic-bezier(0.645, 0.045, 0.355, 1)',
  elastic: 'cubic-bezier(0.68, -0.55, 0.265, 1.55)',
} as const

/**
 * Vue Transition 클래스 생성
 */
export function getTransitionClasses(
  type: AnimationType,
  direction?: AnimationDirection,
): Record<string, string> {
  const baseClass = `overlay-${type}`
  const directionClass = direction ? `-${direction}` : ''
  const prefix = `${baseClass}${directionClass}`

  return {
    enterActiveClass: `${prefix}-enter-active`,
    leaveActiveClass: `${prefix}-leave-active`,
    enterFromClass: `${prefix}-enter-from`,
    leaveToClass: `${prefix}-leave-to`,
  }
}

/**
 * 애니메이션 스타일 생성
 */
export function getAnimationStyles(
  type: AnimationType,
  duration: number = ANIMATION_DURATION.normal,
  easing: string = ANIMATION_EASING.easeInOut,
): Record<string, any> {
  return {
    transition: `all ${duration}ms ${easing}`,
  }
}

/**
 * 슬라이드 애니메이션을 위한 transform 값 계산
 */
export function getSlideTransform(direction: AnimationDirection): string {
  const transforms = {
    top: 'translateY(-100%)',
    right: 'translateX(100%)',
    bottom: 'translateY(100%)',
    left: 'translateX(-100%)',
  }

  return transforms[direction]
}