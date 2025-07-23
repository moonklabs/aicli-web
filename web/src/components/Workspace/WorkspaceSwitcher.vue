<template>
  <div class="workspace-switcher">
    <!-- 메인 워크스페이스 선택 버튼 -->
    <n-dropdown
      :options="workspaceOptions"
      :show-arrow="true"
      @select="switchWorkspace"
      trigger="click"
      placement="bottom-start"
      :disabled="isSwitching"
    >
      <n-button
        class="workspace-selector"
        :loading="isSwitching"
        size="large"
        ghost
      >
        <template #icon>
          <n-icon><Layers /></n-icon>
        </template>
        
        <div class="workspace-info">
          <div class="workspace-name">
            {{ activeWorkspace?.name || '워크스페이스 선택' }}
          </div>
          <div class="workspace-path" v-if="activeWorkspace">
            {{ truncatePath(activeWorkspace.path) }}
          </div>
        </div>
        
        <template #suffix>
          <n-icon><ChevronDown /></n-icon>
        </template>
      </n-button>
    </n-dropdown>

    <!-- 빠른 전환 힌트 -->
    <div class="quick-switch-hint" v-if="workspaces.length > 1">
      <n-text depth="3" size="small">
        <kbd>Ctrl</kbd> + <kbd>`</kbd>로 빠른 전환
      </n-text>
    </div>

    <!-- 빠른 전환 모달 -->
    <n-modal
      v-model:show="showQuickSwitcher"
      preset="card"
      style="width: 600px;"
      title="워크스페이스 빠른 전환"
      :bordered="false"
      :closable="true"
      :mask-closable="true"
      @keydown.esc="closeQuickSwitcher"
    >
      <div class="quick-switcher">
        <n-input
          v-model:value="quickSearchQuery"
          placeholder="워크스페이스 검색..."
          size="large"
          clearable
          autofocus
          @keydown.enter="switchToSelectedWorkspace"
          @keydown.up.prevent="navigateUp"
          @keydown.down.prevent="navigateDown"
        >
          <template #prefix>
            <n-icon><Search /></n-icon>
          </template>
        </n-input>

        <div class="workspace-list">
          <div
            v-for="(workspace, index) in filteredWorkspaces"
            :key="workspace.id"
            class="workspace-item"
            :class="{ 
              'selected': selectedIndex === index,
              'active': workspace.id === activeWorkspace?.id 
            }"
            @click="switchWorkspace(workspace.id)"
            @mouseenter="selectedIndex = index"
          >
            <div class="workspace-main">
              <div class="workspace-header">
                <h4 class="workspace-name">{{ workspace.name }}</h4>
                <div class="workspace-shortcuts">
                  <kbd v-if="index < 9">{{ index + 1 }}</kbd>
                </div>
              </div>
              <p class="workspace-path">{{ workspace.path }}</p>

              <!-- 워크스페이스 상태 -->
              <div class="workspace-status">
                <n-space size="small">
                  <n-tag
                    :type="getWorkspaceStatusType(workspace.status)"
                    size="small"
                  >
                    {{ getWorkspaceStatusText(workspace.status) }}
                  </n-tag>
                  
                  <n-tag v-if="workspace.git?.branch" type="info" size="small">
                    <template #icon>
                      <n-icon><GitBranch /></n-icon>
                    </template>
                    {{ workspace.git.branch }}
                  </n-tag>
                  
                  <n-tag 
                    v-if="workspace.claudeSession" 
                    :type="workspace.claudeSession.status === 'active' ? 'success' : 'default'"
                    size="small"
                  >
                    <template #icon>
                      <n-icon><Terminal /></n-icon>
                    </template>
                    Claude {{ workspace.claudeSession.status === 'active' ? '활성' : '비활성' }}
                  </n-tag>
                </n-space>
              </div>
            </div>

            <div class="workspace-meta">
              <div class="last-accessed" v-if="workspace.lastAccessed">
                <n-time :time="new Date(workspace.lastAccessed)" relative />
              </div>
            </div>
          </div>

          <div v-if="filteredWorkspaces.length === 0" class="no-results">
            <n-empty description="검색 결과가 없습니다" size="small">
              <template #icon>
                <n-icon><Search /></n-icon>
              </template>
            </n-empty>
          </div>
        </div>

        <div class="quick-switcher-footer">
          <n-text depth="3" size="small">
            <kbd>↑</kbd><kbd>↓</kbd> 탐색 • <kbd>Enter</kbd> 선택 • <kbd>Esc</kbd> 닫기
          </n-text>
        </div>
      </div>
    </n-modal>

    <!-- 전환 진행 상태 -->
    <n-modal
      v-model:show="showSwitchProgress"
      preset="card"
      style="width: 400px;"
      title="워크스페이스 전환 중"
      :bordered="false"
      :closable="false"
      :mask-closable="false"
    >
      <div class="switch-progress">
        <n-progress
          type="line"
          :percentage="switchProgress"
          :show-indicator="false"
          :height="8"
        />
        
        <div class="progress-steps">
          <div 
            v-for="step in switchSteps"
            :key="step.key"
            class="progress-step"
            :class="{ 
              'completed': step.completed,
              'active': step.active 
            }"
          >
            <n-icon class="step-icon">
              <Check v-if="step.completed" />
              <LoaderCircle v-else-if="step.active" class="spin" />
              <Circle v-else />
            </n-icon>
            <span class="step-text">{{ step.text }}</span>
          </div>
        </div>
      </div>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, nextTick, h } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import {
  NButton,
  NDropdown,
  NIcon,
  NText,
  NModal,
  NInput,
  NSpace,
  NTag,
  NTime,
  NEmpty,
  NProgress
} from 'naive-ui'
import {
  Layers,
  ChevronDown,
  Search,
  GitBranch,
  Terminal,
  Check,
  Circle,
  LoaderCircle
} from '@vicons/lucide'

import { useWorkspaceStore } from '@/stores/workspace'

const workspaceStore = useWorkspaceStore()
const router = useRouter()
const message = useMessage()

// 로컬 상태
const isSwitching = ref(false)
const showQuickSwitcher = ref(false)
const showSwitchProgress = ref(false)
const quickSearchQuery = ref('')
const selectedIndex = ref(0)
const switchProgress = ref(0)

// 전환 단계
const switchSteps = ref([
  { key: 'save', text: '현재 상태 저장', completed: false, active: false },
  { key: 'validate', text: '워크스페이스 검증', completed: false, active: false },
  { key: 'load', text: '워크스페이스 로드', completed: false, active: false },
  { key: 'restore', text: '상태 복원', completed: false, active: false },
  { key: 'complete', text: '전환 완료', completed: false, active: false }
])

// 키보드 이벤트 리스너
const keyboardHandlers = new Map<string, () => void>()

// 계산된 속성
const workspaces = computed(() => workspaceStore.workspaces)
const activeWorkspace = computed(() => workspaceStore.activeWorkspace)

const workspaceOptions = computed(() => {
  return workspaces.value.map((workspace, index) => ({
    label: workspace.name,
    key: workspace.id,
    disabled: workspace.id === activeWorkspace.value?.id,
    icon: () => h(NIcon, null, { default: () => h(Layers) }),
    children: [
      {
        label: workspace.path,
        key: `${workspace.id}-path`,
        disabled: true,
        type: 'group'
      }
    ]
  }))
})

const filteredWorkspaces = computed(() => {
  if (!quickSearchQuery.value) return workspaces.value
  
  const query = quickSearchQuery.value.toLowerCase()
  return workspaces.value.filter(workspace => 
    workspace.name.toLowerCase().includes(query) ||
    workspace.path.toLowerCase().includes(query)
  )
})

// 유틸리티 함수
const truncatePath = (path: string, maxLength = 40): string => {
  if (path.length <= maxLength) return path
  
  const parts = path.split('/')
  if (parts.length <= 2) return path
  
  return `.../${parts.slice(-2).join('/')}`
}

const getWorkspaceStatusType = (status: string): 'success' | 'warning' | 'error' | 'info' => {
  switch (status) {
    case 'active': return 'success'
    case 'loading': return 'info'
    case 'error': return 'error'
    default: return 'info'
  }
}

const getWorkspaceStatusText = (status: string): string => {
  switch (status) {
    case 'active': return '활성'
    case 'idle': return '대기'
    case 'loading': return '로딩'
    case 'error': return '오류'
    case 'creating': return '생성중'
    case 'deleting': return '삭제중'
    default: return status
  }
}

// 키보드 단축키 등록
const registerKeyboardShortcuts = (): void => {
  // Ctrl + ` : 빠른 전환 모달
  keyboardHandlers.set('ctrl+`', showQuickSwitcher)
  
  // Ctrl + 1-9 : 직접 워크스페이스 전환
  for (let i = 1; i <= 9; i++) {
    keyboardHandlers.set(`ctrl+${i}`, () => switchToWorkspaceByIndex(i - 1))
  }
  
  // 이벤트 리스너 등록
  document.addEventListener('keydown', handleKeyDown)
}

const unregisterKeyboardShortcuts = (): void => {
  document.removeEventListener('keydown', handleKeyDown)
  keyboardHandlers.clear()
}

const handleKeyDown = (event: KeyboardEvent): void => {
  const key = []
  if (event.ctrlKey) key.push('ctrl')
  if (event.altKey) key.push('alt')
  if (event.shiftKey) key.push('shift')
  key.push(event.key.toLowerCase())
  
  const shortcut = key.join('+')
  const handler = keyboardHandlers.get(shortcut)
  
  if (handler) {
    event.preventDefault()
    handler()
  }
}

// 워크스페이스 전환 관련
const switchWorkspace = async (workspaceId: string): Promise<void> => {
  if (isSwitching.value || workspaceId === activeWorkspace.value?.id) return
  
  const targetWorkspace = workspaces.value.find(w => w.id === workspaceId)
  if (!targetWorkspace) {
    message.error('워크스페이스를 찾을 수 없습니다')
    return
  }
  
  try {
    isSwitching.value = true
    showSwitchProgress.value = true
    switchProgress.value = 0
    
    // 전환 단계 초기화
    switchSteps.value.forEach(step => {
      step.completed = false
      step.active = false
    })
    
    // 단계 1: 현재 상태 저장
    switchSteps.value[0].active = true
    await workspaceStore.saveWorkspaceState()
    switchSteps.value[0].completed = true
    switchSteps.value[0].active = false
    switchProgress.value = 20
    
    await delay(300)
    
    // 단계 2: 워크스페이스 검증
    switchSteps.value[1].active = true
    const isValid = await workspaceStore.validateWorkspace(workspaceId)
    if (!isValid) {
      throw new Error('워크스페이스 검증에 실패했습니다')
    }
    switchSteps.value[1].completed = true
    switchSteps.value[1].active = false
    switchProgress.value = 40
    
    await delay(200)
    
    // 단계 3: 워크스페이스 로드
    switchSteps.value[2].active = true
    await workspaceStore.activateWorkspace(workspaceId)
    switchSteps.value[2].completed = true
    switchSteps.value[2].active = false
    switchProgress.value = 70
    
    await delay(200)
    
    // 단계 4: 상태 복원
    switchSteps.value[3].active = true
    await workspaceStore.restoreWorkspaceState(workspaceId)
    switchSteps.value[3].completed = true
    switchSteps.value[3].active = false
    switchProgress.value = 90
    
    await delay(200)
    
    // 단계 5: 라우터 업데이트 및 완료
    switchSteps.value[4].active = true
    await router.push(`/workspace/${workspaceId}`)
    switchSteps.value[4].completed = true
    switchSteps.value[4].active = false
    switchProgress.value = 100
    
    await delay(500)
    
    // 빠른 전환 모달 닫기
    closeQuickSwitcher()
    
    message.success(`'${targetWorkspace.name}' 워크스페이스로 전환되었습니다`)
    
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : '워크스페이스 전환에 실패했습니다'
    message.error(errorMessage)
    console.error('Workspace switch error:', error)
  } finally {
    isSwitching.value = false
    showSwitchProgress.value = false
    switchProgress.value = 0
  }
}

const switchToWorkspaceByIndex = (index: number): void => {
  if (index >= 0 && index < workspaces.value.length) {
    const workspace = workspaces.value[index]
    switchWorkspace(workspace.id)
  }
}

// 빠른 전환 모달 관련
const showQuickSwitcherModal = (): void => {
  showQuickSwitcher.value = true
  quickSearchQuery.value = ''
  selectedIndex.value = 0
  
  nextTick(() => {
    // 포커스 설정은 모달이 열린 후에
    const input = document.querySelector('.quick-switcher input') as HTMLInputElement
    input?.focus()
  })
}

const closeQuickSwitcher = (): void => {
  showQuickSwitcher.value = false
  quickSearchQuery.value = ''
  selectedIndex.value = 0
}

const navigateUp = (): void => {
  if (selectedIndex.value > 0) {
    selectedIndex.value--
  } else {
    selectedIndex.value = filteredWorkspaces.value.length - 1
  }
}

const navigateDown = (): void => {
  if (selectedIndex.value < filteredWorkspaces.value.length - 1) {
    selectedIndex.value++
  } else {
    selectedIndex.value = 0
  }
}

const switchToSelectedWorkspace = (): void => {
  if (filteredWorkspaces.value.length > 0 && selectedIndex.value >= 0) {
    const selectedWorkspace = filteredWorkspaces.value[selectedIndex.value]
    switchWorkspace(selectedWorkspace.id)
  }
}

// 유틸리티
const delay = (ms: number): Promise<void> => {
  return new Promise(resolve => setTimeout(resolve, ms))
}

// 생명주기
onMounted(() => {
  registerKeyboardShortcuts()
})

onUnmounted(() => {
  unregisterKeyboardShortcuts()
})

// 검색 쿼리 변경시 선택 인덱스 초기화
watch(quickSearchQuery, () => {
  selectedIndex.value = 0
})

// 필터된 워크스페이스 변경시 선택 인덱스 검증
watch(filteredWorkspaces, (newWorkspaces) => {
  if (selectedIndex.value >= newWorkspaces.length) {
    selectedIndex.value = Math.max(0, newWorkspaces.length - 1)
  }
})

// 빠른 전환 함수 노출
defineExpose({
  showQuickSwitcher: showQuickSwitcherModal,
  switchWorkspace
})
</script>

<style scoped>
.workspace-switcher {
  position: relative;
}

.workspace-selector {
  min-width: 250px;
  max-width: 400px;
  height: 48px;
  justify-content: space-between;
  border: 1px solid var(--n-border-color);
  background-color: var(--n-card-color);
}

.workspace-selector:hover {
  border-color: var(--n-color-primary-hover);
}

.workspace-info {
  flex: 1;
  text-align: left;
  margin: 0 12px;
}

.workspace-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--n-text-color);
  line-height: 1.2;
}

.workspace-path {
  font-size: 12px;
  color: var(--n-text-color-3);
  line-height: 1.2;
  margin-top: 2px;
}

.quick-switch-hint {
  margin-top: 8px;
  text-align: center;
}

.quick-switch-hint kbd {
  display: inline-block;
  padding: 2px 6px;
  background-color: var(--n-color-hover);
  border: 1px solid var(--n-border-color);
  border-radius: 4px;
  font-size: 11px;
  font-family: monospace;
  margin: 0 2px;
}

/* 빠른 전환 모달 */
.quick-switcher {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.workspace-list {
  max-height: 400px;
  overflow-y: auto;
  border: 1px solid var(--n-border-color);
  border-radius: 6px;
}

.workspace-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid var(--n-border-color);
  cursor: pointer;
  transition: all 0.2s ease;
}

.workspace-item:last-child {
  border-bottom: none;
}

.workspace-item:hover,
.workspace-item.selected {
  background-color: var(--n-color-hover);
}

.workspace-item.active {
  background-color: var(--n-color-primary-hover);
  border-left: 3px solid var(--n-color-primary);
}

.workspace-main {
  flex: 1;
}

.workspace-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
}

.workspace-item .workspace-name {
  font-size: 16px;
  font-weight: 600;
  color: var(--n-text-color);
  margin: 0;
}

.workspace-shortcuts kbd {
  background-color: var(--n-color-primary);
  color: white;
  padding: 2px 6px;
  border-radius: 3px;
  font-size: 10px;
  font-weight: bold;
}

.workspace-item .workspace-path {
  font-size: 13px;
  color: var(--n-text-color-2);
  margin-bottom: 8px;
}

.workspace-status {
  margin-bottom: 4px;
}

.workspace-meta {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4px;
}

.last-accessed {
  font-size: 12px;
  color: var(--n-text-color-3);
}

.no-results {
  padding: 40px 20px;
  text-align: center;
}

.quick-switcher-footer {
  text-align: center;
  padding: 8px;
  border-top: 1px solid var(--n-border-color);
  background-color: var(--n-color-hover);
  border-radius: 0 0 6px 6px;
}

.quick-switcher-footer kbd {
  display: inline-block;
  padding: 2px 4px;
  background-color: var(--n-card-color);
  border: 1px solid var(--n-border-color);
  border-radius: 2px;
  font-size: 10px;
  margin: 0 2px;
}

/* 전환 진행 상태 */
.switch-progress {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.progress-steps {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.progress-step {
  display: flex;
  align-items: center;
  gap: 12px;
  opacity: 0.5;
  transition: opacity 0.3s ease;
}

.progress-step.active,
.progress-step.completed {
  opacity: 1;
}

.step-icon {
  width: 16px;
  height: 16px;
  color: var(--n-text-color-3);
}

.progress-step.active .step-icon {
  color: var(--n-color-primary);
}

.progress-step.completed .step-icon {
  color: var(--n-color-success);
}

.step-text {
  font-size: 14px;
  color: var(--n-text-color-2);
}

.progress-step.active .step-text {
  color: var(--n-text-color);
  font-weight: 500;
}

.progress-step.completed .step-text {
  color: var(--n-text-color);
}

/* 애니메이션 */
@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.spin {
  animation: spin 1s linear infinite;
}

/* 반응형 디자인 */
@media (max-width: 768px) {
  .workspace-selector {
    min-width: 200px;
    max-width: 300px;
  }

  .workspace-item {
    flex-direction: column;
    align-items: stretch;
    gap: 8px;
  }

  .workspace-header {
    justify-content: space-between;
  }

  .workspace-meta {
    align-items: flex-start;
  }
}

/* 스크롤바 스타일 */
.workspace-list::-webkit-scrollbar {
  width: 6px;
}

.workspace-list::-webkit-scrollbar-track {
  background: var(--n-color-hover);
}

.workspace-list::-webkit-scrollbar-thumb {
  background: var(--n-border-color);
  border-radius: 3px;
}

.workspace-list::-webkit-scrollbar-thumb:hover {
  background: var(--n-text-color-3);
}
</style>