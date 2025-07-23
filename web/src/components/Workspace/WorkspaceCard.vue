<template>
  <n-card
    class="workspace-card"
    :class="{ 
      'active': isActive, 
      'loading': workspace.status === 'loading',
      'error': workspace.status === 'error' 
    }"
    hoverable
    @click="selectWorkspace"
  >
    <template #header>
      <div class="workspace-header">
        <div class="workspace-info">
          <h3 class="workspace-name">{{ workspace.name }}</h3>
          <p class="workspace-path">{{ workspace.path }}</p>
          <p v-if="workspace.description" class="workspace-description">
            {{ workspace.description }}
          </p>
        </div>
        <div class="workspace-actions">
          <n-dropdown :options="actionOptions" @select="handleAction">
            <n-button quaternary circle size="small">
              <template #icon>
                <n-icon><MoreHorizontal /></n-icon>
              </template>
            </n-button>
          </n-dropdown>
        </div>
      </div>
    </template>

    <div class="workspace-content">
      <!-- 상태 태그들 -->
      <div class="workspace-status">
        <n-space>
          <!-- Docker 컨테이너 상태 -->
          <n-tag
            v-if="workspace.containerStatus"
            :type="workspace.containerStatus.type"
            :bordered="false"
            size="small"
          >
            <template #icon>
              <n-icon><Docker /></n-icon>
            </template>
            {{ workspace.containerStatus.text }}
          </n-tag>

          <!-- Git 브랜치 정보 -->
          <n-tag v-if="workspace.git?.branch" type="info" size="small">
            <template #icon>
              <n-icon><GitBranch /></n-icon>
            </template>
            {{ workspace.git.branch }}
            <span v-if="workspace.git.hasChanges" class="git-changes">*</span>
          </n-tag>

          <!-- Claude 세션 상태 -->
          <n-tag
            v-if="workspace.claudeSession"
            :type="getClaudeSessionType(workspace.claudeSession.status)"
            size="small"
          >
            <template #icon>
              <n-icon><Terminal /></n-icon>
            </template>
            Claude {{ workspace.claudeSession.status }}
          </n-tag>

          <!-- 워크스페이스 상태 -->
          <n-tag
            :type="getWorkspaceStatusType(workspace.status)"
            size="small"
          >
            {{ getWorkspaceStatusText(workspace.status) }}
          </n-tag>
        </n-space>
      </div>

      <!-- 통계 정보 -->
      <div class="workspace-stats">
        <n-space>
          <n-statistic 
            label="파일 수" 
            :value="workspace.stats?.fileCount || 0" 
            size="small"
          />
          <n-statistic 
            label="라인 수" 
            :value="workspace.stats?.lineCount || 0" 
            size="small"
          />
          <n-statistic 
            label="마지막 수정" 
            :value="formatTimeAgo(workspace.lastModified)" 
            size="small"
          />
        </n-space>
      </div>

      <!-- 현재 진행 중인 태스크 -->
      <div v-if="workspace.currentTask" class="workspace-task">
        <div class="task-info">
          <span class="task-description">{{ workspace.currentTask.description }}</span>
          <n-tag 
            :type="getTaskStatusType(workspace.currentTask.status)" 
            size="tiny"
          >
            {{ workspace.currentTask.status }}
          </n-tag>
        </div>
        <n-progress
          :percentage="workspace.currentTask.progress"
          :status="getTaskProgressStatus(workspace.currentTask.status)"
          class="task-progress"
        />
      </div>
    </div>
  </n-card>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { NCard, NButton, NDropdown, NIcon, NTag, NSpace, NStatistic, NProgress } from 'naive-ui'
import {
  MoreHorizontal,
  Docker,
  GitBranch,
  Terminal,
  Play,
  Pause,
  Square,
  Settings,
  Trash2,
  ExternalLink
} from '@vicons/lucide'
import type { Workspace } from '@/stores/workspace'

interface Props {
  workspace: Workspace
  isActive?: boolean
}

interface Emits {
  (e: 'select', workspaceId: string): void
  (e: 'action', action: string, workspaceId: string): void
}

const props = withDefaults(defineProps<Props>(), {
  isActive: false
})

const emit = defineEmits<Emits>()

// 액션 메뉴 옵션
const actionOptions = computed(() => [
  {
    label: '활성화',
    key: 'activate',
    icon: () => h(NIcon, null, { default: () => h(Play) }),
    disabled: props.workspace.status === 'active'
  },
  {
    label: '일시정지',
    key: 'pause',
    icon: () => h(NIcon, null, { default: () => h(Pause) }),
    disabled: props.workspace.status !== 'active'
  },
  {
    label: '중지',
    key: 'stop',
    icon: () => h(NIcon, null, { default: () => h(Square) }),
    disabled: props.workspace.status === 'idle'
  },
  {
    type: 'divider',
    key: 'divider1'
  },
  {
    label: '설정',
    key: 'settings',
    icon: () => h(NIcon, null, { default: () => h(Settings) })
  },
  {
    label: '브라우저에서 열기',
    key: 'open-browser',
    icon: () => h(NIcon, null, { default: () => h(ExternalLink) })
  },
  {
    type: 'divider',
    key: 'divider2'
  },
  {
    label: '삭제',
    key: 'delete',
    icon: () => h(NIcon, null, { default: () => h(Trash2) }),
    props: {
      style: 'color: #d03050'
    }
  }
])

// 이벤트 핸들러
const selectWorkspace = (): void => {
  emit('select', props.workspace.id)
}

const handleAction = (key: string): void => {
  emit('action', key, props.workspace.id)
}

// 상태별 타입 결정 함수들
const getClaudeSessionType = (status: string): 'success' | 'warning' | 'error' | 'info' => {
  switch (status) {
    case 'active': return 'success'
    case 'idle': return 'info'
    case 'error': return 'error'
    default: return 'info'
  }
}

const getWorkspaceStatusType = (status: string): 'success' | 'warning' | 'error' | 'info' => {
  switch (status) {
    case 'active': return 'success'
    case 'loading': return 'warning'
    case 'error': return 'error'
    case 'creating': return 'info'
    case 'deleting': return 'warning'
    default: return 'info'
  }
}

const getWorkspaceStatusText = (status: string): string => {
  switch (status) {
    case 'active': return '활성'
    case 'idle': return '대기'
    case 'loading': return '로딩중'
    case 'error': return '오류'
    case 'creating': return '생성중'
    case 'deleting': return '삭제중'
    default: return status
  }
}

const getTaskStatusType = (status: string): 'success' | 'warning' | 'error' | 'info' => {
  switch (status) {
    case 'running': return 'info'
    case 'paused': return 'warning'
    case 'completed': return 'success'
    case 'error': return 'error'
    default: return 'info'
  }
}

const getTaskProgressStatus = (status: string): 'success' | 'warning' | 'error' | 'info' | undefined => {
  switch (status) {
    case 'completed': return 'success'
    case 'error': return 'error'
    default: return undefined
  }
}

// 시간 포맷팅
const formatTimeAgo = (date: Date): string => {
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  
  const minutes = Math.floor(diff / (1000 * 60))
  const hours = Math.floor(diff / (1000 * 60 * 60))
  const days = Math.floor(diff / (1000 * 60 * 60 * 24))
  
  if (minutes < 1) return '방금 전'
  if (minutes < 60) return `${minutes}분 전`
  if (hours < 24) return `${hours}시간 전`
  if (days < 7) return `${days}일 전`
  
  return date.toLocaleDateString('ko-KR')
}
</script>

<style scoped>
.workspace-card {
  min-width: 300px;
  max-width: 400px;
  transition: all 0.3s ease;
}

.workspace-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 20px rgba(0, 0, 0, 0.1);
}

.workspace-card.active {
  border: 2px solid var(--n-color-primary);
  box-shadow: 0 0 0 2px rgba(24, 160, 88, 0.2);
}

.workspace-card.loading {
  opacity: 0.7;
}

.workspace-card.error {
  border: 2px solid var(--n-color-error);
}

.workspace-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
}

.workspace-info {
  flex: 1;
  min-width: 0;
}

.workspace-name {
  margin: 0 0 4px 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--n-text-color);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.workspace-path {
  margin: 0 0 4px 0;
  font-size: 12px;
  color: var(--n-text-color-2);
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.workspace-description {
  margin: 0;
  font-size: 14px;
  color: var(--n-text-color-3);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.workspace-actions {
  flex-shrink: 0;
}

.workspace-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.workspace-status {
  display: flex;
  flex-wrap: wrap;
}

.git-changes {
  color: var(--n-color-warning);
  font-weight: bold;
  margin-left: 2px;
}

.workspace-stats {
  padding: 12px 0;
  border-top: 1px solid var(--n-border-color);
  border-bottom: 1px solid var(--n-border-color);
}

.workspace-task {
  background-color: var(--n-color-hover);
  padding: 12px;
  border-radius: 6px;
}

.task-info {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.task-description {
  font-size: 14px;
  color: var(--n-text-color);
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
  margin-right: 8px;
}

.task-progress {
  margin-top: 8px;
}

@media (max-width: 768px) {
  .workspace-card {
    min-width: 280px;
    max-width: 100%;
  }

  .workspace-header {
    flex-direction: column;
    align-items: stretch;
  }

  .workspace-actions {
    align-self: flex-end;
  }

  .workspace-stats {
    flex-direction: column;
    gap: 8px;
  }
}
</style>