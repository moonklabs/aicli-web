/**
 * Z-index 관리 유틸리티
 * 오버레이 컴포넌트의 z-index를 자동으로 관리합니다
 */

// 기본 z-index 값들
export const Z_INDEX_BASE = {
  DROPDOWN: 1000,
  TOOLTIP: 2000,
  POPOVER: 3000,
  DRAWER: 4000,
  MODAL: 5000,
  NOTIFICATION: 6000,
} as const

// 현재 활성화된 오버레이들의 z-index 추적
const activeOverlays = new Map<string, number>()
let currentMaxZIndex = Math.max(...Object.values(Z_INDEX_BASE))

/**
 * 새로운 오버레이에 대한 z-index 할당
 */
export function allocateZIndex(overlayId: string, type: keyof typeof Z_INDEX_BASE): number {
  const baseZIndex = Z_INDEX_BASE[type]
  const zIndex = Math.max(baseZIndex, currentMaxZIndex + 1)
  
  activeOverlays.set(overlayId, zIndex)
  currentMaxZIndex = zIndex
  
  return zIndex
}

/**
 * 오버레이가 닫힐 때 z-index 해제
 */
export function releaseZIndex(overlayId: string): void {
  activeOverlays.delete(overlayId)
  
  // 현재 활성화된 오버레이 중 최대 z-index 재계산
  if (activeOverlays.size > 0) {
    currentMaxZIndex = Math.max(...activeOverlays.values())
  } else {
    currentMaxZIndex = Math.max(...Object.values(Z_INDEX_BASE))
  }
}

/**
 * 특정 오버레이의 z-index 가져오기
 */
export function getZIndex(overlayId: string): number | undefined {
  return activeOverlays.get(overlayId)
}

/**
 * 모든 활성 오버레이 ID 가져오기
 */
export function getActiveOverlayIds(): string[] {
  return Array.from(activeOverlays.keys())
}

/**
 * 가장 위에 있는 오버레이 ID 가져오기
 */
export function getTopOverlayId(): string | undefined {
  let topId: string | undefined
  let maxZ = -1
  
  activeOverlays.forEach((zIndex, id) => {
    if (zIndex > maxZ) {
      maxZ = zIndex
      topId = id
    }
  })
  
  return topId
}