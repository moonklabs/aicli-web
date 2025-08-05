<template>
  <Teleport to="body">
    <div
      v-if="isMobile && isVisible"
      class="quick-action-panel"
      :class="panelClasses"
    >
      <!-- 드래그 핸들 -->
      <div
        ref="dragHandle"
        class="drag-handle"
        @touchstart="handleDragStart"
        @touchmove="handleDragMove"
        @touchend="handleDragEnd"
      >
        <div class="drag-indicator" />
      </div>

      <!-- 패널 헤더 -->
      <div class="panel-header">
        <h3 class="panel-title">빠른 액션</h3>
        <div class="header-actions">
          <TouchButton
            variant="tertiary"
            size="sm"
            round
            icon="SettingsIcon"
            aria-label="액션 설정"
            @click="openSettings"
          />
          <TouchButton
            variant="tertiary"
            size="sm"
            round
            icon="CloseIcon"
            aria-label="패널 닫기"
            @click="close"
          />
        </div>
      </div>

      <!-- 액션 그리드 -->
      <div class="action-grid" :class="gridClasses">
        <!-- 주요 액션들 -->
        <div class="action-section">
          <h4 class="section-title">주요 액션</h4>
          <div class="action-buttons">
            <TouchButton
              v-for="action in primaryActions"
              :key="action.id"
              class="action-button"
              :class="getActionButtonClass(action)"
              variant="secondary"
              size="lg"
              :icon="action.icon"
              :haptic-feedback="true"
              @click="() => executeAction(action)"
              @long-press="() => showActionMenu(action)"
            >
              <div class="action-content">
                <component v-if="action.icon" :is="action.icon" class="action-icon" />
                <span class="action-label">{{ action.label }}</span>
                <span v-if="action.shortcut" class="action-shortcut">
                  {{ action.shortcut }}
                </span>
              </div>
            </TouchButton>
          </div>
        </div>

        <!-- 컨텍스트 액션들 -->
        <div v-if="contextualActions.length > 0" class="action-section">
          <h4 class="section-title">{{ contextTitle }}</h4>
          <div class="action-buttons">
            <TouchButton
              v-for="action in contextualActions"
              :key="action.id"
              class="action-button contextual-action"
              variant="tertiary"
              size="md"
              :icon="action.icon"
              :haptic-feedback="true"
              @click="() => executeAction(action)"
            >
              <div class="action-content">
                <component v-if="action.icon" :is="action.icon" class="action-icon" />
                <span class="action-label">{{ action.label }}</span>
              </div>
            </TouchButton>
          </div>
        </div>

        <!-- 최근 액션들 -->
        <div v-if="recentActions.length > 0" class="action-section">
          <h4 class="section-title">최근 사용</h4>
          <div class="recent-actions">
            <TouchButton
              v-for="action in recentActions"
              :key="`recent-${action.id}`"
              class="recent-action-button"
              variant="tertiary"
              size="sm"
              :icon="action.icon"
              round
              :haptic-feedback="true"
              @click="() => executeAction(action)"
            >
              <component v-if="action.icon" :is="action.icon" />
            </TouchButton>
          </div>
        </div>
      </div>

      <!-- 하단 네비게이션 바 (한 손 모드용) -->
      <div v-if="showBottomNav" class="bottom-nav">
        <TouchButton
          v-for="navItem in bottomNavItems"
          :key="navItem.id"
          class="nav-button"
          :class="{ 'nav-button--active': navItem.active }"
          variant="tertiary"
          size="md"
          :icon="navItem.icon"
          @click="() => navigateToPage(navItem)"
        >
          <component v-if="navItem.icon" :is="navItem.icon" />
          <span class="nav-label">{{ navItem.label }}</span>
        </TouchButton>
      </div>

      <!-- 배경 오버레이 -->
      <div
        v-if="isExpanded"
        class="panel-backdrop"
        @click="close"
        @touchstart="close"
      />
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, ref, watch, onMounted, onBeforeUnmount } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMobileOptimization } from '@/composables/useMobileOptimization'
import { useMobileWorkflow } from '@/composables/useMobileWorkflow'
import TouchButton from '../form/TouchButton.vue'

interface QuickAction {
  id: string
  label: string
  icon: string
  action: () => void
  category: 'primary' | 'secondary' | 'contextual'
  shortcut?: string
  requiresAuth?: boolean
  priority?: number
}

interface NavItem {
  id: string
  label: string
  icon: string
  path: string
  active: boolean
}

interface Props {
  // 표시 설정
  position?: 'bottom' | 'side'
  expandHeight?: 'half' | 'full' | 'auto'
  showOnScroll?: boolean
  persistState?: boolean
  
  // 동작 설정
  dragToClose?: boolean
  swipeToNavigate?: boolean
  autoHide?: boolean
  hideDelay?: number
  
  // 레이아웃 설정
  gridColumns?: number
  showBottomNav?: boolean
  enableRecentActions?: boolean
  maxRecentActions?: number
}

const props = withDefaults(defineProps<Props>(), {
  position: 'bottom',
  expandHeight: 'auto',
  showOnScroll: false,
  persistState: true,
  dragToClose: true,
  swipeToNavigate: true,
  autoHide: false,
  hideDelay: 5000,
  gridColumns: 2,
  showBottomNav: true,
  enableRecentActions: true,
  maxRecentActions: 6,
})

const emit = defineEmits<{
  'action-execute': [action: QuickAction]
  'panel-open': []
  'panel-close': []
  'settings-open': []
}>()

// 컴포저블
const route = useRoute()
const router = useRouter()
const { isMobile, screenHeight, orientation } = useMobileOptimization()
const { workflowState, quickActions, contextualActions } = useMobileWorkflow()

// 상태
const isVisible = ref(false)
const isExpanded = ref(false)
const dragPosition = ref(0)
const isDragging = ref(false)
const recentActions = ref<QuickAction[]>([])

// 드래그 상태
const dragHandle = ref<HTMLElement>()
const dragStartY = ref(0)
const dragCurrentY = ref(0)

// 주요 액션들
const primaryActions = computed(() => 
  (quickActions.value as QuickAction[]).filter(action => 
    action.category === 'primary'
  ).slice(0, 6)
)

// 컨텍스트 제목
const contextTitle = computed(() => {
  const contextMap: Record<string, string> = {
    workspace: '워크스페이스',
    terminal: '터미널',
    docker: 'Docker',
    profile: '프로필',
    settings: '설정',
  }
  
  return contextMap[workflowState.value.currentContext] || '컨텍스트 액션'
})

// 하단 네비게이션 아이템
const bottomNavItems = computed<NavItem[]>(() => [
  {
    id: 'dashboard',
    label: '홈',
    icon: 'HomeIcon',
    path: '/',
    active: route.path === '/',
  },
  {
    id: 'workspaces',
    label: '워크스페이스',
    icon: 'WorkspaceIcon',
    path: '/workspaces',
    active: route.path.startsWith('/workspaces'),
  },
  {
    id: 'terminal',
    label: '터미널',
    icon: 'TerminalIcon',
    path: '/terminal',
    active: route.path.startsWith('/terminal'),
  },
  {
    id: 'profile',
    label: '프로필',
    icon: 'UserIcon',
    path: '/profile',
    active: route.path.startsWith('/profile'),
  },
])

// 패널 클래스
const panelClasses = computed(() => [
  `panel--${props.position}`,
  `panel--${props.expandHeight}`,
  {
    'panel--expanded': isExpanded.value,
    'panel--dragging': isDragging.value,
    'panel--one-hand': workflowState.value.isOneHandMode,
    'panel--landscape': orientation.value === 'landscape',
  },
])

// 그리드 클래스
const gridClasses = computed(() => [
  `grid--columns-${props.gridColumns}`,
])

// 액션 버튼 클래스
const getActionButtonClass = (action: QuickAction) => [
  'primary-action',
  `priority-${action.priority || 1}`,
]

// 드래그 처리
const handleDragStart = (event: TouchEvent) => {
  if (!props.dragToClose) return
  
  isDragging.value = true
  dragStartY.value = event.touches[0].clientY
  dragCurrentY.value = dragStartY.value
}

const handleDragMove = (event: TouchEvent) => {
  if (!isDragging.value) return
  
  event.preventDefault()
  dragCurrentY.value = event.touches[0].clientY
  const deltaY = dragCurrentY.value - dragStartY.value
  
  // 아래로만 드래그 허용
  if (deltaY > 0) {
    dragPosition.value = deltaY
  }
}

const handleDragEnd = () => {
  if (!isDragging.value) return
  
  isDragging.value = false
  const deltaY = dragCurrentY.value - dragStartY.value
  
  // 일정 거리 이상 드래그하면 닫기
  if (deltaY > screenHeight.value * 0.3) {
    close()
  } else {
    // 원래 위치로 복원
    dragPosition.value = 0
  }
}

// 액션 실행
const executeAction = (action: QuickAction) => {
  action.action()
  emit('action-execute', action)
  
  // 최근 액션에 추가
  if (props.enableRecentActions) {
    addToRecentActions(action)
  }
  
  // 자동 숨김 모드라면 닫기
  if (props.autoHide) {
    setTimeout(() => {
      close()
    }, 100)
  }
}

// 액션 메뉴 표시
const showActionMenu = (action: QuickAction) => {
  console.log('Showing action menu for:', action.label)
  // 추가 옵션 메뉴 표시
}

// 최근 액션에 추가
const addToRecentActions = (action: QuickAction) => {
  // 중복 제거
  recentActions.value = recentActions.value.filter(a => a.id !== action.id)
  
  // 맨 앞에 추가
  recentActions.value.unshift(action)
  
  // 최대 개수 제한
  if (recentActions.value.length > props.maxRecentActions) {
    recentActions.value = recentActions.value.slice(0, props.maxRecentActions)
  }
  
  // 로컬 스토리지에 저장
  if (props.persistState) {
    localStorage.setItem('quickActionPanel.recentActions', JSON.stringify(recentActions.value))
  }
}

// 페이지 네비게이션
const navigateToPage = (navItem: NavItem) => {
  router.push(navItem.path)
  
  if (props.autoHide) {
    close()
  }
}

// 패널 열기/닫기
const open = () => {
  isVisible.value = true
  isExpanded.value = true
  emit('panel-open')
}

const close = () => {
  isExpanded.value = false
  
  setTimeout(() => {
    isVisible.value = false
    emit('panel-close')
  }, 300)
}

const toggle = () => {
  if (isExpanded.value) {
    close()
  } else {
    open()
  }
}

// 설정 열기
const openSettings = () => {
  emit('settings-open')
  close()
}

// 스크롤 감지
const handleScroll = () => {
  if (!props.showOnScroll) return
  
  const scrollY = window.scrollY
  
  // 스크롤이 일정 이상이면 표시
  if (scrollY > 200 && !isVisible.value) {
    isVisible.value = true
  }
}

// 키보드 단축키
const handleKeyPress = (event: KeyboardEvent) => {
  // Cmd/Ctrl + K로 패널 토글
  if ((event.metaKey || event.ctrlKey) && event.key === 'k') {
    event.preventDefault()
    toggle()
  }
  
  // ESC로 패널 닫기
  if (event.key === 'Escape' && isExpanded.value) {
    close()
  }
}

// 저장된 상태 복원
const restoreState = () => {
  if (!props.persistState) return
  
  try {
    const saved = localStorage.getItem('quickActionPanel.recentActions')
    if (saved) {
      recentActions.value = JSON.parse(saved)
    }
  } catch (error) {
    console.warn('Failed to restore quick action panel state:', error)
  }
}

// 라이프사이클
onMounted(() => {
  restoreState()
  
  if (props.showOnScroll) {
    window.addEventListener('scroll', handleScroll, { passive: true })
  }
  
  document.addEventListener('keydown', handleKeyPress)
})

onBeforeUnmount(() => {
  window.removeEventListener('scroll', handleScroll)
  document.removeEventListener('keydown', handleKeyPress)
})

// 외부에서 호출할 수 있도록 expose
defineExpose({
  open,
  close,
  toggle,
  isVisible: computed(() => isVisible.value),
  isExpanded: computed(() => isExpanded.value),
})
</script>

<style lang="scss" scoped>
@use '@/styles/variables' as *;
@use '@/styles/mixins' as *;

.quick-action-panel {
  position: fixed;
  z-index: $z-modal;
  
  &.panel--bottom {
    bottom: 0;
    left: 0;
    right: 0;
    transform: translateY(100%);
    transition: transform $transition-normal;
    
    &.panel--expanded {
      transform: translateY(0);
    }
  }
  
  &.panel--side {
    top: 0;
    right: 0;
    bottom: 0;
    width: 320px;
    transform: translateX(100%);
    transition: transform $transition-normal;
    
    &.panel--expanded {
      transform: translateX(0);
    }
  }
  
  background: $light-bg-primary;
  border-radius: $border-radius-xl $border-radius-xl 0 0;
  box-shadow: $shadow-modal;
  
  .dark & {
    background: $dark-bg-secondary;
  }
  
  &.panel--one-hand {
    max-height: 60vh;
  }
  
  &.panel--landscape {
    max-height: 80vh;
  }
  
  &.panel--dragging {
    transition: none;
    transform: translateY(var(--drag-position, 0));
  }
}

.drag-handle {
  @include flex-center;
  padding: $spacing-3;
  cursor: grab;
  
  &:active {
    cursor: grabbing;
  }
  
  .drag-indicator {
    width: 40px;
    height: 4px;
    background: map-get($gray-colors, 300);
    border-radius: $border-radius-full;
    
    .dark & {
      background: $dark-bg-tertiary;
    }
  }
}

.panel-header {
  @include flex-between;
  padding: 0 $spacing-6 $spacing-4;
  border-bottom: 1px solid map-get($gray-colors, 200);
  
  .dark & {
    border-bottom-color: $dark-bg-tertiary;
  }
  
  .panel-title {
    font-size: $font-size-lg;
    font-weight: $font-weight-semibold;
    color: $light-text-primary;
    
    .dark & {
      color: $dark-text-primary;
    }
  }
  
  .header-actions {
    display: flex;
    gap: $spacing-2;
  }
}

.action-grid {
  padding: $spacing-4 $spacing-6;
  max-height: 50vh;
  overflow-y: auto;
  @include scrollbar-thin;
  
  &.grid--columns-1 {
    .action-buttons {
      grid-template-columns: 1fr;
    }
  }
  
  &.grid--columns-2 {
    .action-buttons {
      grid-template-columns: repeat(2, 1fr);
    }
  }
  
  &.grid--columns-3 {
    .action-buttons {
      grid-template-columns: repeat(3, 1fr);
    }
  }
}

.action-section {
  margin-bottom: $spacing-6;
  
  &:last-child {
    margin-bottom: 0;
  }
  
  .section-title {
    font-size: $font-size-sm;
    font-weight: $font-weight-semibold;
    color: $light-text-secondary;
    margin-bottom: $spacing-3;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    
    .dark & {
      color: $dark-text-secondary;
    }
  }
}

.action-buttons {
  display: grid;
  gap: $spacing-3;
}

.action-button {
  @include flex-center;
  flex-direction: column;
  padding: $spacing-4;
  border: 1px solid map-get($gray-colors, 200);
  border-radius: $border-radius-lg;
  background: map-get($gray-colors, 50);
  transition: all $transition-base;
  min-height: 80px;
  
  .dark & {
    border-color: $dark-bg-tertiary;
    background: $dark-bg-primary;
  }
  
  &:hover {
    @include no-touch {
      border-color: map-get($primary-colors, 300);
      background: map-get($primary-colors, 50);
      transform: translateY(-2px);
      box-shadow: $shadow-md;
      
      .dark & {
        border-color: map-get($primary-colors, 500);
        background: rgba(map-get($primary-colors, 500), 0.1);
      }
    }
  }
  
  &.primary-action {
    border-color: map-get($primary-colors, 200);
    background: map-get($primary-colors, 50);
    
    .dark & {
      border-color: map-get($primary-colors, 700);
      background: rgba(map-get($primary-colors, 500), 0.1);
    }
    
    &.priority-1 {
      background: map-get($primary-colors, 100);
      
      .dark & {
        background: rgba(map-get($primary-colors, 500), 0.2);
      }
    }
  }
  
  &.contextual-action {
    min-height: 60px;
  }
  
  .action-content {
    @include flex-center;
    flex-direction: column;
    gap: $spacing-2;
    text-align: center;
  }
  
  .action-icon {
    width: 24px;
    height: 24px;
    color: map-get($primary-colors, 600);
    
    .dark & {
      color: map-get($primary-colors, 400);
    }
  }
  
  .action-label {
    font-size: $font-size-sm;
    font-weight: $font-weight-medium;
    color: $light-text-primary;
    
    .dark & {
      color: $dark-text-primary;
    }
  }
  
  .action-shortcut {
    font-size: $font-size-xs;
    color: $light-text-secondary;
    font-family: monospace;
    
    .dark & {
      color: $dark-text-secondary;
    }
  }
}

.recent-actions {
  display: flex;
  gap: $spacing-3;
  overflow-x: auto;
  padding-bottom: $spacing-2;
  @include scrollbar-thin;
  
  .recent-action-button {
    flex-shrink: 0;
    width: 48px;
    height: 48px;
    border: 1px solid map-get($gray-colors, 200);
    background: map-get($gray-colors, 50);
    
    .dark & {
      border-color: $dark-bg-tertiary;
      background: $dark-bg-primary;
    }
    
    &:hover {
      border-color: map-get($primary-colors, 300);
      background: map-get($primary-colors, 50);
      
      .dark & {
        border-color: map-get($primary-colors, 500);
        background: rgba(map-get($primary-colors, 500), 0.1);
      }
    }
  }
}

.bottom-nav {
  display: flex;
  padding: $spacing-3 $spacing-6;
  border-top: 1px solid map-get($gray-colors, 200);
  background: $light-bg-secondary;
  
  .dark & {
    border-top-color: $dark-bg-tertiary;
    background: $dark-bg-primary;
  }
  
  .nav-button {
    flex: 1;
    @include flex-center;
    flex-direction: column;
    gap: $spacing-1;
    padding: $spacing-2;
    min-height: 60px;
    
    .nav-label {
      font-size: $font-size-xs;
      color: $light-text-secondary;
      
      .dark & {
        color: $dark-text-secondary;
      }
    }
    
    &.nav-button--active {
      .nav-label {
        color: map-get($primary-colors, 600);
        font-weight: $font-weight-semibold;
        
        .dark & {
          color: map-get($primary-colors, 400);
        }
      }
    }
  }
}

.panel-backdrop {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  z-index: -1;
  backdrop-filter: blur(4px);
}

// 접근성 개선
@include reduce-motion {
  .quick-action-panel {
    transition: none;
  }
  
  .action-button {
    transition: none;
    
    &:hover {
      transform: none;
    }
  }
}

// 모바일 최적화
@include mobile {
  .action-grid {
    padding: $spacing-3 $spacing-4;
  }
  
  .action-button {
    min-height: 70px;
    padding: $spacing-3;
    
    .action-icon {
      width: 20px;
      height: 20px;
    }
    
    .action-label {
      font-size: $font-size-xs;
    }
  }
}
</style>