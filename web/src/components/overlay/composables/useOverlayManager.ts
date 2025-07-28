import { type InjectionKey, type Ref, computed, inject, onUnmounted, provide, ref } from 'vue'
import { allocateZIndex, getTopOverlayId, releaseZIndex } from '../utils/z-index'

export interface OverlayInstance {
  id: string
  type: 'modal' | 'drawer' | 'popover' | 'tooltip'
  zIndex: number
  closeOnEsc?: boolean
  closeOnClickOutside?: boolean
  onClose?: () => void
}

export interface OverlayManager {
  register: (overlay: Omit<OverlayInstance, 'zIndex'>) => number
  unregister: (id: string) => void
  isTopOverlay: (id: string) => boolean
  closeTopOverlay: () => void
  getActiveOverlays: () => OverlayInstance[]
}

// Injection key for overlay manager
const OverlayManagerKey: InjectionKey<OverlayManager> = Symbol('overlay-manager')

// 전역 오버레이 스택
const overlayStack = ref<OverlayInstance[]>([])

/**
 * 오버레이 매니저 생성 및 제공
 */
export function provideOverlayManager(): OverlayManager {
  const manager: OverlayManager = {
    register(overlay) {
      const zIndex = allocateZIndex(overlay.id, overlay.type.toUpperCase() as any)
      const instance: OverlayInstance = { ...overlay, zIndex }

      overlayStack.value.push(instance)

      return zIndex
    },

    unregister(id) {
      const index = overlayStack.value.findIndex(o => o.id === id)
      if (index > -1) {
        overlayStack.value.splice(index, 1)
        releaseZIndex(id)
      }
    },

    isTopOverlay(id) {
      return getTopOverlayId() === id
    },

    closeTopOverlay() {
      const topId = getTopOverlayId()
      if (topId) {
        const overlay = overlayStack.value.find(o => o.id === topId)
        if (overlay?.onClose) {
          overlay.onClose()
        }
      }
    },

    getActiveOverlays() {
      return [...overlayStack.value]
    },
  }

  provide(OverlayManagerKey, manager)

  return manager
}

/**
 * 오버레이 매니저 사용
 */
export function useOverlayManager(): OverlayManager | undefined {
  return inject(OverlayManagerKey)
}

/**
 * 개별 오버레이 인스턴스 관리
 */
export function useOverlay(options: {
  id: string
  type: OverlayInstance['type']
  closeOnEsc?: boolean
  closeOnClickOutside?: boolean
  onClose?: () => void
}) {
  const manager = useOverlayManager()
  const zIndex = ref<number>(0)
  const isTop = computed(() => manager?.isTopOverlay(options.id) ?? false)

  // 오버레이 등록
  if (manager) {
    zIndex.value = manager.register({
      id: options.id,
      type: options.type,
      closeOnEsc: options.closeOnEsc ?? true,
      closeOnClickOutside: options.closeOnClickOutside ?? false,
      onClose: options.onClose,
    })
  }

  // ESC 키 핸들러
  const handleEsc = (event: KeyboardEvent) => {
    if (event.key === 'Escape' && isTop.value && options.closeOnEsc) {
      options.onClose?.()
    }
  }

  // 전역 이벤트 리스너 등록
  if (typeof window !== 'undefined') {
    window.addEventListener('keydown', handleEsc)
  }

  // 정리
  onUnmounted(() => {
    manager?.unregister(options.id)
    if (typeof window !== 'undefined') {
      window.removeEventListener('keydown', handleEsc)
    }
  })

  return {
    zIndex,
    isTop,
  }
}