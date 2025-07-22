<template>
  <div class="workspace-detail-view">
    <header class="detail-header">
      <NButton text @click="goBack" class="back-btn">
        <template #icon>
          <NIcon><BackIcon /></NIcon>
        </template>
        뒤로 가기
      </NButton>

      <div v-if="workspace" class="workspace-info">
        <h1 class="workspace-name">{{ workspace.name }}</h1>
        <NBadge
          :type="getStatusType(workspace.status)"
          :text="getStatusText(workspace.status)"
          class="workspace-status"
        />
      </div>

      <div class="header-actions">
        <NButton
          v-if="workspace?.status === 'inactive'"
          type="primary"
          @click="startWorkspace"
          :loading="isLoading"
        >
          시작
        </NButton>
        <NButton
          v-if="workspace?.status === 'active'"
          type="warning"
          @click="stopWorkspace"
          :loading="isLoading"
        >
          중지
        </NButton>
        <NButton
          type="error"
          @click="deleteWorkspace"
          :loading="isLoading"
        >
          삭제
        </NButton>
      </div>
    </header>

    <div v-if="workspace" class="detail-content">
      <!-- 워크스페이스 정보 -->
      <NCard class="info-card">
        <template #header>워크스페이스 정보</template>
        <NDescriptions :columns="2" bordered>
          <NDescriptionsItem label="이름">
            {{ workspace.name }}
          </NDescriptionsItem>
          <NDescriptionsItem label="상태">
            <NBadge
              :type="getStatusType(workspace.status)"
              :text="getStatusText(workspace.status)"
            />
          </NDescriptionsItem>
          <NDescriptionsItem label="경로">
            <NCode>{{ workspace.path }}</NCode>
          </NDescriptionsItem>
          <NDescriptionsItem label="컨테이너 ID">
            <NCode v-if="workspace.containerId">{{ workspace.containerId }}</NCode>
            <span v-else class="empty-value">없음</span>
          </NDescriptionsItem>
          <NDescriptionsItem label="Git 저장소">
            <NCode v-if="workspace.gitRemote">{{ workspace.gitRemote }}</NCode>
            <span v-else class="empty-value">설정되지 않음</span>
          </NDescriptionsItem>
          <NDescriptionsItem label="Git 브랜치">
            <NCode v-if="workspace.gitBranch">{{ workspace.gitBranch }}</NCode>
            <span v-else class="empty-value">설정되지 않음</span>
          </NDescriptionsItem>
          <NDescriptionsItem label="생성일">
            {{ formatDate(workspace.createdAt) }}
          </NDescriptionsItem>
          <NDescriptionsItem label="마지막 활동">
            {{ formatDate(workspace.lastActivity || workspace.updatedAt) }}
          </NDescriptionsItem>
        </NDescriptions>
        <NP v-if="workspace.description">
          <strong>설명:</strong><br />
          {{ workspace.description }}
        </NP>
      </NCard>

      <!-- 터미널 세션 -->
      <NCard class="terminal-card">
        <template #header>
          <div class="card-header">
            <span>터미널 세션</span>
            <NButton
              type="primary"
              size="small"
              @click="createTerminalSession"
              :disabled="workspace.status !== 'active'"
            >
              새 터미널
            </NButton>
          </div>
        </template>

        <div v-if="terminalSessions.length === 0" class="empty-section">
          <NEmpty description="터미널 세션이 없습니다">
            <template #extra>
              <NButton
                type="primary"
                @click="createTerminalSession"
                :disabled="workspace.status !== 'active'"
              >
                터미널 세션 생성
              </NButton>
            </template>
          </NEmpty>
        </div>

        <div v-else class="terminal-sessions">
          <div
            v-for="session in terminalSessions"
            :key="session.id"
            class="session-item"
            @click="openTerminalSession(session.id)"
          >
            <div class="session-info">
              <h4 class="session-title">{{ session.title }}</h4>
              <NBadge
                :type="getSessionStatusType(session.status)"
                :text="getSessionStatusText(session.status)"
              />
            </div>
            <div class="session-meta">
              <span class="session-time">{{ formatDate(session.lastActivity) }}</span>
              <span class="session-logs">{{ session.logs.length }}개 로그</span>
            </div>
          </div>
        </div>
      </NCard>

      <!-- 로그 및 활동 -->
      <NCard class="logs-card">
        <template #header>최근 활동</template>
        <NEmpty description="활동 로그가 없습니다" />
      </NCard>
    </div>

    <div v-else class="loading-state">
      <NSpin size="large">
        <template #description>워크스페이스 정보를 불러오는 중...</template>
      </NSpin>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import {
  NBadge,
  NButton,
  NCard,
  NCode,
  NDescriptions,
  NDescriptionsItem,
  NEmpty,
  NIcon,
  NP,
  NSpin,
} from 'naive-ui'
import { useWorkspaceStore } from '@/stores/workspace'
import { useTerminalStore } from '@/stores/terminal'
import type { Workspace } from '@/stores/workspace'
import type { TerminalSession } from '@/stores/terminal'
import { formatDate } from '@/utils/format'

// 아이콘
const BackIcon = {
  render: () => '←',
}

const route = useRoute()
const router = useRouter()
const message = useMessage()
const workspaceStore = useWorkspaceStore()
const terminalStore = useTerminalStore()

const workspaceId = route.params.id as string

// 상태
const workspace = computed(() =>
  workspaceStore.workspaceById(workspaceId) || null,
)
const isLoading = computed(() => workspaceStore.isLoading)
const terminalSessions = computed(() =>
  terminalStore.sessionsByWorkspace(workspaceId),
)

// 메서드
const goBack = () => {
  router.push({ name: 'workspaces' })
}

const getStatusType = (status: Workspace['status']) => {
  switch (status) {
    case 'active':
      return 'success'
    case 'inactive':
      return 'default'
    case 'error':
      return 'error'
    case 'creating':
    case 'deleting':
      return 'warning'
    default:
      return 'default'
  }
}

const getStatusText = (status: Workspace['status']) => {
  switch (status) {
    case 'active':
      return '활성'
    case 'inactive':
      return '비활성'
    case 'error':
      return '오류'
    case 'creating':
      return '생성 중'
    case 'deleting':
      return '삭제 중'
    default:
      return '알 수 없음'
  }
}

const getSessionStatusType = (status: TerminalSession['status']) => {
  switch (status) {
    case 'connected':
      return 'success'
    case 'disconnected':
      return 'default'
    case 'error':
      return 'error'
    case 'connecting':
      return 'warning'
    default:
      return 'default'
  }
}

const getSessionStatusText = (status: TerminalSession['status']) => {
  switch (status) {
    case 'connected':
      return '연결됨'
    case 'disconnected':
      return '연결 해제됨'
    case 'error':
      return '오류'
    case 'connecting':
      return '연결 중'
    default:
      return '알 수 없음'
  }
}

const startWorkspace = async () => {
  if (!workspace.value) return

  try {
    await workspaceStore.startWorkspace(workspace.value.id)
    message.success('워크스페이스가 시작되었습니다')
  } catch (_error) {
    message.error('워크스페이스 시작에 실패했습니다')
  }
}

const stopWorkspace = async () => {
  if (!workspace.value) return

  try {
    await workspaceStore.stopWorkspace(workspace.value.id)
    message.success('워크스페이스가 중지되었습니다')
  } catch (_error) {
    message.error('워크스페이스 중지에 실패했습니다')
  }
}

const deleteWorkspace = async () => {
  if (!workspace.value) return

  try {
    await workspaceStore.deleteWorkspace(workspace.value.id)
    message.success('워크스페이스가 삭제되었습니다')
    router.push({ name: 'workspaces' })
  } catch (_error) {
    message.error('워크스페이스 삭제에 실패했습니다')
  }
}

const createTerminalSession = async () => {
  if (!workspace.value) return

  try {
    const session = await terminalStore.createSession(workspace.value.id)
    if (session) {
      message.success('터미널 세션이 생성되었습니다')
      openTerminalSession(session.id)
    }
  } catch (_error) {
    message.error('터미널 세션 생성에 실패했습니다')
  }
}

const openTerminalSession = (sessionId: string) => {
  router.push({ name: 'terminal', params: { sessionId } })
}

// 라이프사이클
onMounted(() => {
  if (!workspace.value) {
    // 워크스페이스 정보가 없으면 목록으로 이동
    message.error('워크스페이스를 찾을 수 없습니다')
    router.push({ name: 'workspaces' })
  }
})
</script>

<style lang="scss" scoped>
.workspace-detail-view {
  padding: 1.5rem;
  max-width: 1200px;
  margin: 0 auto;
}

.detail-header {
  display: flex;
  align-items: center;
  margin-bottom: 2rem;
  gap: 1rem;

  .back-btn {
    color: var(--text-color-2);
    &:hover {
      color: var(--primary-color);
    }
  }

  .workspace-info {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 1rem;

    .workspace-name {
      margin: 0;
      font-size: 1.5rem;
      font-weight: 600;
      color: var(--text-color-1);
    }
  }

  .header-actions {
    display: flex;
    gap: 0.5rem;
  }
}

.detail-content {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.info-card,
.terminal-card,
.logs-card {
  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
}

.empty-value {
  color: var(--text-color-3);
  font-style: italic;
}

.empty-section {
  text-align: center;
  padding: 2rem;
}

.terminal-sessions {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.session-item {
  border: 1px solid var(--border-color);
  border-radius: 6px;
  padding: 1rem;
  cursor: pointer;
  transition: all 0.2s ease;

  &:hover {
    border-color: var(--primary-color);
    background-color: var(--hover-color);
  }

  .session-info {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 0.5rem;

    .session-title {
      margin: 0;
      font-size: 1rem;
      font-weight: 500;
    }
  }

  .session-meta {
    display: flex;
    justify-content: space-between;
    font-size: 0.875rem;
    color: var(--text-color-3);
  }
}

.loading-state {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 300px;
}
</style>