<template>
  <div v-if="isMobile" class="mobile-workflow-manager">
    <!-- 플로팅 액션 버튼 -->
    <FloatingActionButton
      v-if="settings.enableFloatingActionButton"
      :position="fabPosition"
      :expand-trigger="fabExpandTrigger"
      :show-labels="settings.showActionLabels"
      :hide-on-scroll="settings.hideOnScroll"
      @action-click="handleActionClick"
      @expand="handleFabExpand"
    />

    <!-- 빠른 액션 패널 -->
    <QuickActionPanel
      ref="quickActionPanelRef"
      :show-bottom-nav="settings.showBottomNav"
      :enable-recent-actions="settings.enableRecentActions"
      :drag-to-close="settings.enableDragToClose"
      @action-execute="handleActionExecute"
      @panel-open="handlePanelOpen"
      @panel-close="handlePanelClose"
    />

    <!-- 한 손 모드 토글 버튼 -->
    <Transition name="one-hand-toggle">
      <TouchButton
        v-if="settings.enableOneHandMode && showOneHandToggle"
        class="one-hand-toggle"
        :class="oneHandToggleClasses"
        variant="secondary"
        size="md"
        round
        :icon="oneHandModeIcon"
        :aria-label="oneHandModeLabel"
        :haptic-feedback="true"
        @click="toggleOneHandMode"
        @long-press="showOneHandSettings"
      >
        <component :is="oneHandModeIcon" />
      </TouchButton>
    </Transition>

    <!-- 접근성 도우미 -->
    <div v-if="settings.enableAccessibilityHelpers" class="accessibility-helpers">
      <!-- 큰 타겟 모드 토글 -->
      <TouchButton
        v-if="showAccessibilityToggles"
        class="accessibility-toggle"
        variant="tertiary"
        size="sm"
        round
        icon="AccessibilityIcon"
        aria-label="접근성 도우미"
        @click="toggleAccessibilityMode"
      />
    </div>

    <!-- 제스처 가이드 오버레이 -->
    <Transition name="gesture-guide">
      <div
        v-if="showGestureGuide"
        class="gesture-guide-overlay"
        @click="hideGestureGuide"
      >
        <div class="gesture-guide-content">
          <h3>제스처 가이드</h3>
          <div class="gesture-list">
            <div class="gesture-item">
              <span class="gesture-icon">👆</span>
              <span class="gesture-text">FAB 탭: 빠른 액션</span>
            </div>
            <div class="gesture-item">
              <span class="gesture-icon">✋</span>
              <span class="gesture-text">FAB 롱 프레스: 액션 패널</span>
            </div>
            <div class="gesture-item">
              <span class="gesture-icon">👈</span>
              <span class="gesture-text">왼쪽 스와이프: 뒤로 가기</span>
            </div>
            <div class="gesture-item">
              <span class="gesture-icon">👉</span>
              <span class="gesture-text">오른쪽 스와이프: 앞으로 가기</span>
            </div>
            <div class="gesture-item">
              <span class="gesture-icon">👋</span>
              <span class="gesture-text">기기 흔들기: 실행 취소</span>
            </div>
          </div>
          <TouchButton
            class="guide-close-button"
            variant="primary"
            size="md"
            @click="hideGestureGuide"
          >
            확인
          </TouchButton>
        </div>
      </div>
    </Transition>

    <!-- 워크플로우 설정 모달 -->
    <Teleport to="body">
      <div
        v-if="showSettings"
        class="workflow-settings-modal"
        @click="closeSettings"
      >
        <div class="settings-content" @click.stop>
          <div class="settings-header">
            <h3>모바일 워크플로우 설정</h3>
            <TouchButton
              variant="tertiary"
              size="sm"
              round
              icon="CloseIcon"
              @click="closeSettings"
            />
          </div>

          <div class="settings-body">
            <div class="setting-group">
              <h4>인터페이스</h4>
              <label class="setting-item">
                <input
                  v-model="settings.enableFloatingActionButton"
                  type="checkbox"
                />
                플로팅 액션 버튼
              </label>
              <label class="setting-item">
                <input
                  v-model="settings.showActionLabels"
                  type="checkbox"
                />
                액션 라벨 표시
              </label>
              <label class="setting-item">
                <input
                  v-model="settings.showBottomNav"
                  type="checkbox"
                />
                하단 네비게이션
              </label>
            </div>

            <div class="setting-group">
              <h4>제스처 및 인터랙션</h4>
              <label class="setting-item">
                <input
                  v-model="settings.enableSwipeNavigation"
                  type="checkbox"
                />
                스와이프 네비게이션
              </label>
              <label class="setting-item">
                <input
                  v-model="settings.enableDragToClose"
                  type="checkbox"
                />
                드래그로 닫기
              </label>
              <label class="setting-item">
                <input
                  v-model="settings.enableHapticFeedback"
                  type="checkbox"
                />
                햅틱 피드백
              </label>
            </div>

            <div class="setting-group">
              <h4>접근성</h4>
              <label class="setting-item">
                <input
                  v-model="settings.enableOneHandMode"
                  type="checkbox"
                />
                한 손 조작 모드
              </label>
              <label class="setting-item">
                <input
                  v-model="settings.enableLargeTargets"
                  type="checkbox"
                />
                큰 터치 타겟
              </label>
              <label class="setting-item">
                <input
                  v-model="settings.enableVoiceCommands"
                  type="checkbox"
                />
                음성 명령
              </label>
            </div>
          </div>

          <div class="settings-footer">
            <TouchButton
              variant="tertiary"
              size="md"
              @click="resetSettings"
            >
              초기화
            </TouchButton>
            <TouchButton
              variant="primary"
              size="md"
              @click="saveSettings"
            >
              저장
            </TouchButton>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch, onMounted, onBeforeUnmount } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMobileOptimization } from '@/composables/useMobileOptimization'
import { useMobileWorkflow } from '@/composables/useMobileWorkflow'
import { useAdvancedGestures } from '@/composables/useAdvancedGestures'
import FloatingActionButton from './FloatingActionButton.vue'
import QuickActionPanel from './QuickActionPanel.vue'
import TouchButton from '../form/TouchButton.vue'

interface WorkflowSettings {
  // UI 설정
  enableFloatingActionButton: boolean
  showActionLabels: boolean
  showBottomNav: boolean
  hideOnScroll: boolean
  
  // 제스처 설정
  enableSwipeNavigation: boolean
  enableDragToClose: boolean
  enableHapticFeedback: boolean
  
  // 접근성 설정
  enableOneHandMode: boolean
  enableLargeTargets: boolean
  enableVoiceCommands: boolean
  enableAccessibilityHelpers: boolean
  
  // 기능 설정
  enableRecentActions: boolean
  enableContextualActions: boolean
  enableGestureGuide: boolean
}

interface Props {
  // 자동 설정
  autoDetectUsage?: boolean
  adaptToContext?: boolean
  learnFromBehavior?: boolean
  
  // 초기 설정
  initialSettings?: Partial<WorkflowSettings>
  persistSettings?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  autoDetectUsage: true,
  adaptToContext: true,
  learnFromBehavior: true,
  persistSettings: true,
})

const emit = defineEmits<{
  'workflow-change': [settings: WorkflowSettings]
  'action-execute': [actionId: string]
  'settings-change': [settings: WorkflowSettings]
}>()

// 컴포저블
const route = useRoute()
const router = useRouter()
const { isMobile, orientation } = useMobileOptimization()
const { workflowState, toggleOneHandMode } = useMobileWorkflow()

// 상태
const showSettings = ref(false)
const showGestureGuide = ref(false)
const showOneHandToggle = ref(false)
const showAccessibilityToggles = ref(false)
const isFirstTime = ref(true)

// 참조
const quickActionPanelRef = ref<InstanceType<typeof QuickActionPanel>>()

// 설정
const settings = reactive<WorkflowSettings>({
  enableFloatingActionButton: true,
  showActionLabels: false,
  showBottomNav: true,
  hideOnScroll: true,
  enableSwipeNavigation: true,
  enableDragToClose: true,
  enableHapticFeedback: true,
  enableOneHandMode: true,
  enableLargeTargets: false,
  enableVoiceCommands: false,
  enableAccessibilityHelpers: true,
  enableRecentActions: true,
  enableContextualActions: true,
  enableGestureGuide: true,
})

// FAB 설정
const fabPosition = computed(() => {
  if (workflowState.value.isOneHandMode) {
    return orientation.value === 'landscape' ? 'bottom-left' : 'bottom-right'
  }
  return 'bottom-right'
})

const fabExpandTrigger = computed(() => {
  return settings.enableSwipeNavigation ? 'longpress' : 'click'
})

// 한 손 모드 설정
const oneHandModeIcon = computed(() => 
  workflowState.value.isOneHandMode ? 'PhoneIcon' : 'HandIcon'
)

const oneHandModeLabel = computed(() => 
  workflowState.value.isOneHandMode ? '한 손 모드 해제' : '한 손 모드 활성화'
)

const oneHandToggleClasses = computed(() => [
  {
    'one-hand-toggle--active': workflowState.value.isOneHandMode,
    'one-hand-toggle--landscape': orientation.value === 'landscape',
  },
])

// 이벤트 핸들러
const handleActionClick = (action: any) => {
  emit('action-execute', action.id)
  
  // 사용 패턴 학습
  if (props.learnFromBehavior) {
    learnFromAction(action)
  }
}

const handleActionExecute = (action: any) => {
  emit('action-execute', action.id)
}

const handleFabExpand = () => {
  // FAB 확장 시 quick action panel 닫기
  if (quickActionPanelRef.value?.isExpanded.value) {
    quickActionPanelRef.value.close()
  }
}

const handlePanelOpen = () => {
  // 패널 열릴 때 한 손 토글 숨기기
  showOneHandToggle.value = false
}

const handlePanelClose = () => {
  // 패널 닫힐 때 한 손 토글 표시
  setTimeout(() => {
    showOneHandToggle.value = true
  }, 300)
}

// 한 손 모드 관련
const showOneHandSettings = () => {
  console.log('Showing one hand mode settings...')
  // 한 손 모드 상세 설정 표시
}

const toggleAccessibilityMode = () => {
  settings.enableLargeTargets = !settings.enableLargeTargets
  
  // DOM에 클래스 적용
  document.documentElement.setAttribute(
    'data-large-targets',
    settings.enableLargeTargets.toString()
  )
}

// 제스처 가이드
const showGestureGuideIfNeeded = () => {
  if (isFirstTime.value && settings.enableGestureGuide) {
    setTimeout(() => {
      showGestureGuide.value = true
    }, 2000)
    isFirstTime.value = false
  }
}

const hideGestureGuide = () => {
  showGestureGuide.value = false
  
  // 다시 표시하지 않도록 저장
  if (props.persistSettings) {
    localStorage.setItem('mobileWorkflow.gestureGuideShown', 'true')
  }
}

// 설정 관리
const saveSettings = () => {
  if (props.persistSettings) {
    localStorage.setItem('mobileWorkflow.settings', JSON.stringify(settings))
  }
  
  emit('settings-change', { ...settings })
  showSettings.value = false
}

const resetSettings = () => {
  Object.assign(settings, {
    enableFloatingActionButton: true,
    showActionLabels: false,
    showBottomNav: true,
    hideOnScroll: true,
    enableSwipeNavigation: true,
    enableDragToClose: true,
    enableHapticFeedback: true,
    enableOneHandMode: true,
    enableLargeTargets: false,
    enableVoiceCommands: false,
    enableAccessibilityHelpers: true,
    enableRecentActions: true,
    enableContextualActions: true,
    enableGestureGuide: true,
  })
}

const loadSettings = () => {
  if (!props.persistSettings) return
  
  try {
    const saved = localStorage.getItem('mobileWorkflow.settings')
    if (saved) {
      Object.assign(settings, JSON.parse(saved))
    }
    
    // 초기 설정 적용
    if (props.initialSettings) {
      Object.assign(settings, props.initialSettings)
    }
    
    // 제스처 가이드 표시 여부 확인
    const guideShown = localStorage.getItem('mobileWorkflow.gestureGuideShown')
    isFirstTime.value = !guideShown
  } catch (error) {
    console.warn('Failed to load mobile workflow settings:', error)
  }
}

const closeSettings = () => {
  showSettings.value = false
}

// 사용 패턴 학습
const learnFromAction = (action: any) => {
  // 자주 사용하는 액션을 FAB에 우선 표시
  console.log('Learning from action:', action.id)
  
  // 실제 구현에서는 사용 빈도를 추적하고
  // 머신러닝 또는 통계 기반으로 UI를 최적화
}

// 컨텍스트 적응
const adaptToCurrentContext = () => {
  if (!props.adaptToContext) return
  
  const currentRoute = route.name as string
  
  // 라우트별 설정 조정
  switch (currentRoute) {
    case 'terminal':
      settings.showBottomNav = false // 터미널에서는 하단 네비 숨김
      break
    case 'workspaces':
      settings.enableContextualActions = true
      break
    default:
      // 기본 설정 유지
      break
  }
}

// 스와이프 네비게이션 설정
const setupSwipeNavigation = () => {
  if (!settings.enableSwipeNavigation) return
  
  // 전체 화면에 스와이프 제스처 적용
  const gestures = useAdvancedGestures(ref(document.documentElement), {
    enableSwipeNavigation: true,
    enableHapticFeedback: settings.enableHapticFeedback,
  })
  
  // 좌우 스와이프로 네비게이션
  gestures.on('swipeleft', () => {
    router.back()
  })
  
  gestures.on('swiperight', () => {
    router.forward()
  })
}

// 라이프사이클
onMounted(() => {
  loadSettings()
  adaptToCurrentContext()
  
  if (isMobile.value) {
    setupSwipeNavigation()
    showGestureGuideIfNeeded()
    
    // 초기 상태 설정
    setTimeout(() => {
      showOneHandToggle.value = settings.enableOneHandMode
      showAccessibilityToggles.value = settings.enableAccessibilityHelpers
    }, 1000)
  }
})

// 라우트 변경 감지
watch(() => route.name, () => {
  adaptToCurrentContext()
})

// 설정 변경 감지
watch(settings, (newSettings) => {
  emit('workflow-change', { ...newSettings })
  
  // 접근성 설정 적용
  if (newSettings.enableLargeTargets) {
    document.documentElement.setAttribute('data-large-targets', 'true')
  } else {
    document.documentElement.removeAttribute('data-large-targets')
  }
}, { deep: true })

// 외부에서 접근 가능한 메서드
defineExpose({
  openSettings: () => { showSettings.value = true },
  closeSettings,
  showGestureGuide: () => { showGestureGuide.value = true },
  hideGestureGuide,
  toggleOneHandMode,
  settings: computed(() => settings),
})
</script>

<style lang="scss" scoped>
@use '@/styles/variables' as *;
@use '@/styles/mixins' as *;

.mobile-workflow-manager {
  position: relative;
  z-index: $z-workflow;
}

.one-hand-toggle {
  position: fixed;
  top: 50%;
  right: $spacing-2;
  transform: translateY(-50%);
  z-index: $z-fab - 1;
  transition: all $transition-normal;
  
  &.one-hand-toggle--active {
    background: map-get($primary-colors, 500);
    color: white;
  }
  
  &.one-hand-toggle--landscape {
    top: 20%;
    right: $spacing-4;
  }
}

.accessibility-helpers {
  position: fixed;
  bottom: 120px;
  right: $spacing-6;
  z-index: $z-fab - 2;
  
  .accessibility-toggle {
    box-shadow: $shadow-md;
  }
}

.gesture-guide-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.8);
  @include flex-center;
  z-index: $z-modal + 1;
  backdrop-filter: blur(4px);
  
  .gesture-guide-content {
    background: $light-bg-primary;
    border-radius: $border-radius-xl;
    padding: $spacing-6;
    margin: $spacing-4;
    max-width: 400px;
    width: 100%;
    
    .dark & {
      background: $dark-bg-secondary;
    }
    
    h3 {
      font-size: $font-size-xl;
      font-weight: $font-weight-semibold;
      text-align: center;
      margin-bottom: $spacing-6;
      color: $light-text-primary;
      
      .dark & {
        color: $dark-text-primary;
      }
    }
    
    .gesture-list {
      margin-bottom: $spacing-6;
      
      .gesture-item {
        @include flex-center;
        gap: $spacing-3;
        padding: $spacing-3;
        border-radius: $border-radius-md;
        margin-bottom: $spacing-2;
        background: map-get($gray-colors, 50);
        
        .dark & {
          background: $dark-bg-primary;
        }
        
        .gesture-icon {
          font-size: $font-size-xl;
          flex-shrink: 0;
        }
        
        .gesture-text {
          color: $light-text-primary;
          font-size: $font-size-sm;
          
          .dark & {
            color: $dark-text-primary;
          }
        }
      }
    }
    
    .guide-close-button {
      width: 100%;
    }
  }
}

.workflow-settings-modal {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  @include flex-center;
  z-index: $z-modal;
  backdrop-filter: blur(4px);
  
  .settings-content {
    background: $light-bg-primary;
    border-radius: $border-radius-xl;
    margin: $spacing-4;
    max-width: 500px;
    width: 100%;
    max-height: 80vh;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    
    .dark & {
      background: $dark-bg-secondary;
    }
  }
  
  .settings-header {
    @include flex-between;
    padding: $spacing-6;
    border-bottom: 1px solid map-get($gray-colors, 200);
    
    .dark & {
      border-bottom-color: $dark-bg-tertiary;
    }
    
    h3 {
      font-size: $font-size-lg;
      font-weight: $font-weight-semibold;
      color: $light-text-primary;
      
      .dark & {
        color: $dark-text-primary;
      }
    }
  }
  
  .settings-body {
    flex: 1;
    overflow-y: auto;
    padding: $spacing-6;
    @include scrollbar-thin;
    
    .setting-group {
      margin-bottom: $spacing-6;
      
      &:last-child {
        margin-bottom: 0;
      }
      
      h4 {
        font-size: $font-size-base;
        font-weight: $font-weight-semibold;
        color: $light-text-primary;
        margin-bottom: $spacing-4;
        
        .dark & {
          color: $dark-text-primary;
        }
      }
      
      .setting-item {
        @include flex-center;
        gap: $spacing-3;
        padding: $spacing-3;
        margin-bottom: $spacing-2;
        border-radius: $border-radius-md;
        background: map-get($gray-colors, 50);
        cursor: pointer;
        transition: $transition-base;
        
        .dark & {
          background: $dark-bg-primary;
        }
        
        &:hover {
          background: map-get($gray-colors, 100);
          
          .dark & {
            background: $dark-bg-tertiary;
          }
        }
        
        input[type="checkbox"] {
          width: 18px;
          height: 18px;
        }
      }
    }
  }
  
  .settings-footer {
    @include flex-between;
    padding: $spacing-6;
    border-top: 1px solid map-get($gray-colors, 200);
    gap: $spacing-3;
    
    .dark & {
      border-top-color: $dark-bg-tertiary;
    }
  }
}

// 애니메이션
.one-hand-toggle-enter-active,
.one-hand-toggle-leave-active {
  transition: all $transition-normal;
}

.one-hand-toggle-enter-from,
.one-hand-toggle-leave-to {
  opacity: 0;
  transform: translateY(-50%) translateX(100%);
}

.gesture-guide-enter-active,
.gesture-guide-leave-active {
  transition: all $transition-normal;
}

.gesture-guide-enter-from,
.gesture-guide-leave-to {
  opacity: 0;
}

.gesture-guide-enter-from .gesture-guide-content,
.gesture-guide-leave-to .gesture-guide-content {
  transform: scale(0.9) translateY(20px);
}

// 접근성
@include reduce-motion {
  .one-hand-toggle,
  .gesture-guide-overlay,
  .workflow-settings-modal {
    transition: none;
  }
}
</style>