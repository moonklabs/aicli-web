<template>
  <Teleport to="body">
    <div
      v-if="isMobile && showFab"
      class="floating-action-button"
      :class="fabClasses"
    >
      <!-- 메인 FAB 버튼 -->
      <TouchButton
        ref="mainFabRef"
        :class="mainButtonClasses"
        variant="primary"
        size="lg"
        round
        :icon="mainIcon"
        :aria-label="mainLabel"
        :haptic-feedback="true"
        :ripple-effect="true"
        @click="handleMainAction"
        @long-press="handleLongPress"
      >
        <component v-if="mainIcon" :is="mainIcon" />
      </TouchButton>

      <!-- 확장된 액션 버튼들 -->
      <Transition name="fab-expand">
        <div
          v-if="isExpanded"
          class="fab-actions"
          :class="actionsClasses"
        >
          <TouchButton
            v-for="(action, index) in visibleActions"
            :key="action.id"
            :class="actionButtonClasses"
            :style="getActionButtonStyle(index)"
            variant="secondary"
            size="md"
            round
            :icon="action.icon"
            :aria-label="action.label"
            :haptic-feedback="true"
            @click="() => handleActionClick(action)"
          >
            <component v-if="action.icon" :is="action.icon" />
            <span v-if="showLabels" class="action-label">
              {{ action.label }}
            </span>
          </TouchButton>
        </div>
      </Transition>

      <!-- 백드롭 (확장 시 배경 클릭으로 닫기) -->
      <div
        v-if="isExpanded"
        class="fab-backdrop"
        @click="collapse"
        @touchstart="collapse"
      />

      <!-- 액션 레이블 팝오버 -->
      <div
        v-if="hoveredAction && !isExpanded"
        class="action-tooltip"
        :style="tooltipStyle"
      >
        {{ hoveredAction.label }}
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, ref, watch, onMounted, onBeforeUnmount } from 'vue'
import { useMobileOptimization } from '@/composables/useMobileOptimization'
import { useMobileWorkflow } from '@/composables/useMobileWorkflow'
import { useAdvancedGestures } from '@/composables/useAdvancedGestures'
import TouchButton from '../form/TouchButton.vue'

interface FabAction {
  id: string
  label: string
  icon: string
  action: () => void
  category: 'primary' | 'secondary' | 'contextual'
  requiresAuth?: boolean
  hotkey?: string
}

interface Props {
  // 기본 설정
  position?: 'bottom-right' | 'bottom-left' | 'bottom-center'
  size?: 'sm' | 'md' | 'lg'
  variant?: 'floating' | 'docked' | 'extended'
  
  // 동작 설정
  expandDirection?: 'up' | 'left' | 'up-left' | 'fan'
  expandTrigger?: 'click' | 'longpress' | 'hover'
  autoCollapse?: boolean
  collapseDelay?: number
  
  // 표시 설정
  showLabels?: boolean
  maxActions?: number
  hideOnScroll?: boolean
  adaptToKeyboard?: boolean
  
  // 접근성
  ariaLabel?: string
  announceActions?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  position: 'bottom-right',
  size: 'lg',
  variant: 'floating',
  expandDirection: 'up',
  expandTrigger: 'click',
  autoCollapse: true,
  collapseDelay: 3000,
  showLabels: false,
  maxActions: 6,
  hideOnScroll: true,
  adaptToKeyboard: true,
  ariaLabel: '빠른 액션',
  announceActions: true,
})

const emit = defineEmits<{
  'action-click': [action: FabAction]
  'expand': []
  'collapse': []
}>()

// 컴포저블
const { isMobile, screenHeight, orientation } = useMobileOptimization()
const { workflowState, availableActions } = useMobileWorkflow()

// 상태
const isExpanded = ref(false)
const isVisible = ref(true)
const hoveredAction = ref<FabAction | null>(null)
const mainFabRef = ref<InstanceType<typeof TouchButton>>()
const lastScrollY = ref(0)
const keyboardHeight = ref(0)

// 메인 버튼 설정
const mainIcon = computed(() => isExpanded.value ? 'CloseIcon' : 'AddIcon')
const mainLabel = computed(() => isExpanded.value ? '액션 닫기' : '빠른 액션 열기')

// 표시 여부
const showFab = computed(() => {
  return isMobile.value && isVisible.value && !workflowState.value.bottomSheetOpen
})

// FAB 클래스
const fabClasses = computed(() => [
  `fab--${props.position}`,
  `fab--${props.size}`,
  `fab--${props.variant}`,
  `fab--${props.expandDirection}`,
  {
    'fab--expanded': isExpanded.value,
    'fab--one-hand': workflowState.value.isOneHandMode,
    'fab--landscape': orientation.value === 'landscape',
    'fab--keyboard-visible': keyboardHeight.value > 0,
  },
])

const mainButtonClasses = computed(() => [
  'fab-main-button',
  {
    'fab-main-button--expanded': isExpanded.value,
  },
])

const actionsClasses = computed(() => [
  `fab-actions--${props.expandDirection}`,
])

const actionButtonClasses = computed(() => [
  'fab-action-button',
])

// 표시할 액션들
const visibleActions = computed(() => {
  const actions = availableActions.value as FabAction[]
  return actions
    .filter(action => action.category === 'primary' || action.category === 'contextual')
    .slice(0, props.maxActions)
})

// 액션 버튼 스타일 계산
const getActionButtonStyle = (index: number) => {
  const total = visibleActions.value.length
  const baseDistance = 80 // 기본 거리
  const angle = props.expandDirection === 'fan' ? (index / (total - 1)) * 90 - 45 : 0

  let x = 0
  let y = 0

  switch (props.expandDirection) {
    case 'up':
      y = -(baseDistance * (index + 1))
      break
    case 'left':
      x = -(baseDistance * (index + 1))
      break
    case 'up-left':
      x = -(baseDistance * (index + 1) * 0.7)
      y = -(baseDistance * (index + 1) * 0.7)
      break
    case 'fan':
      const distance = baseDistance * (index + 1)
      x = Math.cos((angle * Math.PI) / 180) * distance
      y = -Math.sin((angle * Math.PI) / 180) * distance
      break
  }

  return {
    transform: `translate(${x}px, ${y}px)`,
    transitionDelay: `${index * 50}ms`,
  }
}

// 툴팁 스타일
const tooltipStyle = computed(() => {
  if (!hoveredAction.value) return {}
  
  return {
    bottom: '100px',
    right: '20px',
  }
})

// 스크롤 감지
const handleScroll = () => {
  if (!props.hideOnScroll) return

  const currentScrollY = window.scrollY
  const scrollDelta = currentScrollY - lastScrollY.value

  if (scrollDelta > 10) {
    // 아래로 스크롤 시 숨기기
    isVisible.value = false
    if (isExpanded.value) {
      collapse()
    }
  } else if (scrollDelta < -10) {
    // 위로 스크롤 시 보이기
    isVisible.value = true
  }

  lastScrollY.value = currentScrollY
}

// 키보드 감지
const handleKeyboardShow = () => {
  if (!props.adaptToKeyboard) return
  
  // iOS에서 visualViewport API 사용
  if ('visualViewport' in window) {
    const viewport = (window as any).visualViewport
    keyboardHeight.value = window.innerHeight - viewport.height
  }
}

// 메인 액션 처리
const handleMainAction = () => {
  if (props.expandTrigger === 'click') {
    toggle()
  } else {
    // 기본 액션 실행 (새 워크스페이스 등)
    const primaryAction = visibleActions.value.find(a => a.category === 'primary')
    if (primaryAction) {
      primaryAction.action()
      emit('action-click', primaryAction)
    }
  }
}

// 롱 프레스 처리
const handleLongPress = () => {
  if (props.expandTrigger === 'longpress') {
    toggle()
  } else {
    // 컨텍스트 메뉴 표시
    showContextMenu()
  }
}

// 액션 클릭 처리
const handleActionClick = (action: FabAction) => {
  action.action()
  emit('action-click', action)
  
  if (props.autoCollapse) {
    collapse()
  }
}

// 확장/축소
const expand = () => {
  isExpanded.value = true
  emit('expand')
  
  if (props.autoCollapse && props.collapseDelay > 0) {
    setTimeout(() => {
      if (isExpanded.value) {
        collapse()
      }
    }, props.collapseDelay)
  }
}

const collapse = () => {
  isExpanded.value = false
  emit('collapse')
}

const toggle = () => {
  if (isExpanded.value) {
    collapse()
  } else {
    expand()
  }
}

// 컨텍스트 메뉴 표시
const showContextMenu = () => {
  // 추가 옵션을 보여주는 컨텍스트 메뉴
  console.log('Showing context menu...')
}

// 제스처 처리
const setupGestures = () => {
  if (!mainFabRef.value?.$el) return

  const gestures = useAdvancedGestures(ref(mainFabRef.value.$el), {
    enableRotation: false,
    enableMultiFingerGestures: false,
    enableVelocityGestures: true,
    enableHapticFeedback: true,
  })

  // 빠른 스와이프로 액션 실행
  gestures.on('fastswipe', (event) => {
    const direction = event.velocity.x > 0 ? 'right' : 'left'
    
    if (direction === 'left' && visibleActions.value.length > 0) {
      // 첫 번째 액션 빠른 실행
      handleActionClick(visibleActions.value[0])
    }
  })
}

// 라이프사이클
onMounted(() => {
  if (props.hideOnScroll) {
    window.addEventListener('scroll', handleScroll, { passive: true })
  }
  
  if (props.adaptToKeyboard) {
    window.addEventListener('resize', handleKeyboardShow)
    
    if ('visualViewport' in window) {
      const viewport = (window as any).visualViewport
      viewport.addEventListener('resize', handleKeyboardShow)
    }
  }
  
  // 제스처 설정은 다음 틱에서
  setTimeout(() => {
    setupGestures()
  }, 100)
})

onBeforeUnmount(() => {
  window.removeEventListener('scroll', handleScroll)
  window.removeEventListener('resize', handleKeyboardShow)
  
  if ('visualViewport' in window) {
    const viewport = (window as any).visualViewport
    viewport.removeEventListener('resize', handleKeyboardShow)
  }
})

// 키보드 단축키 처리
watch(() => workflowState.value.currentContext, () => {
  // 컨텍스트 변경 시 FAB 상태 초기화
  if (isExpanded.value) {
    collapse()
  }
})
</script>

<style lang="scss" scoped>
@use '@/styles/variables' as *;
@use '@/styles/mixins' as *;

.floating-action-button {
  position: fixed;
  z-index: $z-fab;
  
  // 위치별 스타일
  &.fab--bottom-right {
    bottom: $spacing-6;
    right: $spacing-6;
    
    .fab--one-hand & {
      bottom: 40%;
      right: $spacing-4;
    }
  }
  
  &.fab--bottom-left {
    bottom: $spacing-6;
    left: $spacing-6;
    
    .fab--one-hand & {
      bottom: 40%;
      left: $spacing-4;
    }
  }
  
  &.fab--bottom-center {
    bottom: $spacing-6;
    left: 50%;
    transform: translateX(-50%);
    
    .fab--one-hand & {
      bottom: 40%;
    }
  }
  
  // 가로 모드 조정
  &.fab--landscape {
    &.fab--bottom-right,
    &.fab--bottom-left {
      bottom: $spacing-4;
    }
  }
  
  // 키보드 표시 시 조정
  &.fab--keyboard-visible {
    bottom: calc(#{$spacing-6} + var(--keyboard-height, 0px));
  }
}

.fab-main-button {
  position: relative;
  transition: all $transition-normal;
  box-shadow: $shadow-fab;
  
  &:hover {
    transform: scale(1.05);
    box-shadow: $shadow-fab-hover;
  }
  
  &:active {
    transform: scale(0.95);
  }
  
  &.fab-main-button--expanded {
    transform: rotate(45deg);
    background: map-get($primary-colors, 600);
  }
}

.fab-actions {
  position: absolute;
  
  &.fab-actions--up {
    bottom: 100%;
    right: 0;
    margin-bottom: $spacing-4;
  }
  
  &.fab-actions--left {
    right: 100%;
    bottom: 0;
    margin-right: $spacing-4;
  }
  
  &.fab-actions--up-left {
    right: 100%;
    bottom: 100%;
    margin-right: $spacing-4;
    margin-bottom: $spacing-4;
  }
  
  &.fab-actions--fan {
    right: 0;
    bottom: 0;
  }
}

.fab-action-button {
  position: absolute;
  transition: all $transition-normal;
  box-shadow: $shadow-md;
  
  &:hover {
    transform: scale(1.1);
    box-shadow: $shadow-lg;
  }
  
  .action-label {
    position: absolute;
    right: 100%;
    margin-right: $spacing-2;
    background: $dark-bg-primary;
    color: $dark-text-primary;
    padding: $spacing-1 $spacing-2;
    border-radius: $border-radius-sm;
    font-size: $font-size-xs;
    white-space: nowrap;
    opacity: 0;
    transition: opacity $transition-fast;
  }
  
  &:hover .action-label {
    opacity: 1;
  }
}

.fab-backdrop {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.3);
  z-index: -1;
  backdrop-filter: blur(2px);
}

.action-tooltip {
  position: absolute;
  background: $dark-bg-primary;
  color: $dark-text-primary;
  padding: $spacing-2 $spacing-3;
  border-radius: $border-radius-md;
  font-size: $font-size-sm;
  white-space: nowrap;
  box-shadow: $shadow-lg;
  z-index: $z-tooltip;
  
  &::after {
    content: '';
    position: absolute;
    top: 100%;
    left: 50%;
    transform: translateX(-50%);
    border: 6px solid transparent;
    border-top-color: $dark-bg-primary;
  }
}

// 애니메이션
.fab-expand-enter-active,
.fab-expand-leave-active {
  transition: all $transition-normal;
}

.fab-expand-enter-from,
.fab-expand-leave-to {
  opacity: 0;
  transform: scale(0.8);
}

.fab-expand-enter-active .fab-action-button {
  transition: all $transition-normal;
}

.fab-expand-enter-from .fab-action-button {
  opacity: 0;
  transform: scale(0) !important;
}

// 한 손 모드 글로벌 스타일
:global(.one-hand-mode) {
  .floating-action-button {
    transform: translateY(var(--one-hand-offset, 0)) scale(var(--one-hand-scale, 1));
  }
}

// 접근성 개선
@include reduce-motion {
  .fab-main-button,
  .fab-action-button {
    transition: none;
  }
  
  .fab-main-button--expanded {
    transform: none;
  }
  
  .fab-expand-enter-active,
  .fab-expand-leave-active {
    transition: none;
  }
}

// 큰 타겟 모드
:global([data-large-targets="true"]) {
  .fab-main-button {
    min-width: 60px;
    min-height: 60px;
  }
  
  .fab-action-button {
    min-width: 50px;
    min-height: 50px;
  }
}
</style>