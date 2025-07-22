<template>
  <div class="dashboard-view">
    <!-- 헤더 -->
    <header class="dashboard-header">
      <div class="header-content">
        <h1 class="dashboard-title">AICLI Web Dashboard</h1>
        <div class="header-actions">
          <NButton
            type="primary"
            @click="createWorkspace"
            :loading="workspaceStore.isLoading"
          >
            새 워크스페이스
          </NButton>
          <NButton
            circle
            quaternary
            @click="refreshData"
            :loading="isRefreshing"
          >
            <template #icon>
              <NIcon><RefreshIcon /></NIcon>
            </template>
          </NButton>
        </div>
      </div>
    </header>

    <!-- 메인 콘텐츠 -->
    <main class="dashboard-main">
      <!-- 상태 카드들 -->
      <div class="stats-grid">
        <div class="stat-card">
          <div class="stat-icon">
            <NIcon size="24" color="#3182ce">
              <ServerIcon />
            </NIcon>
          </div>
          <div class="stat-content">
            <div class="stat-value">{{ workspaceStore.totalWorkspaces }}</div>
            <div class="stat-label">워크스페이스</div>
          </div>
        </div>

        <div class="stat-card">
          <div class="stat-icon">
            <NIcon size="24" color="#38a169">
              <PlayCircleIcon />
            </NIcon>
          </div>
          <div class="stat-content">
            <div class="stat-value">{{ workspaceStore.activeWorkspaces.length }}</div>
            <div class="stat-label">실행 중</div>
          </div>
        </div>

        <div class="stat-card">
          <div class="stat-icon">
            <NIcon size="24" color="#d69e2e">
              <TerminalIcon />
            </NIcon>
          </div>
          <div class="stat-content">
            <div class="stat-value">{{ terminalStore.activeSessions.length }}</div>
            <div class="stat-label">터미널 세션</div>
          </div>
        </div>

        <div class="stat-card">
          <div class="stat-icon">
            <NIcon size="24" color="#805ad5">
              <ContainerIcon />
            </NIcon>
          </div>
          <div class="stat-content">
            <div class="stat-value">{{ dockerStore.runningContainers.length }}</div>
            <div class="stat-label">실행 중인 컨테이너</div>
          </div>
        </div>
      </div>

      <!-- 콘텐츠 그리드 -->
      <div class="content-grid">
        <!-- 최근 워크스페이스 -->
        <div class="content-section">
          <div class="section-header">
            <h2 class="section-title">최근 워크스페이스</h2>
            <router-link to="/workspaces" class="section-link">
              전체 보기 →
            </router-link>
          </div>
          <div class="workspace-list">
            <div
              v-for="workspace in recentWorkspaces"
              :key="workspace.id"
              class="workspace-item"
              @click="openWorkspace(workspace.id)"
            >
              <div class="workspace-info">
                <div class="workspace-name">{{ workspace.name }}</div>
                <div class="workspace-path">{{ workspace.path }}</div>
              </div>
              <div class="workspace-status">
                <NTag
                  :type="getStatusType(workspace.status)"
                  size="small"
                >
                  {{ formatStatus(workspace.status) }}
                </NTag>
              </div>
            </div>
            <div v-if="recentWorkspaces.length === 0" class="empty-state">
              <NEmpty description="워크스페이스가 없습니다" />
            </div>
          </div>
        </div>

        <!-- 실행 중인 컨테이너 -->
        <div class="content-section">
          <div class="section-header">
            <h2 class="section-title">실행 중인 컨테이너</h2>
            <router-link to="/docker" class="section-link">
              전체 보기 →
            </router-link>
          </div>
          <div class="container-list">
            <div
              v-for="container in runningContainers"
              :key="container.id"
              class="container-item"
            >
              <div class="container-info">
                <div class="container-name">{{ container.name }}</div>
                <div class="container-image">{{ container.image }}</div>
              </div>
              <div class="container-actions">
                <NButton size="small" quaternary @click="stopContainer(container.id)">
                  중지
                </NButton>
              </div>
            </div>
            <div v-if="runningContainers.length === 0" class="empty-state">
              <NEmpty description="실행 중인 컨테이너가 없습니다" />
            </div>
          </div>
        </div>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  NButton,
  NEmpty,
  NIcon,
  NTag,
  useMessage,
} from 'naive-ui'
import {
  CubeOutline as ContainerIcon,
  PlayCircleOutline as PlayCircleIcon,
  RefreshOutline as RefreshIcon,
  ServerOutline as ServerIcon,
  TerminalOutline as TerminalIcon,
} from '@vicons/ionicons5'

import { useWorkspaceStore } from '@/stores/workspace'
import { useTerminalStore } from '@/stores/terminal'
import { useDockerStore } from '@/stores/docker'
import { formatStatusBadge } from '@/utils/format'

const router = useRouter()
const message = useMessage()

// 스토어
const workspaceStore = useWorkspaceStore()
const terminalStore = useTerminalStore()
const dockerStore = useDockerStore()

// 상태
const isRefreshing = ref(false)

// 계산된 속성
const recentWorkspaces = computed(() =>
  workspaceStore.workspaces.slice(0, 5),
)

const runningContainers = computed(() =>
  dockerStore.runningContainers.slice(0, 5),
)

// 메서드
const createWorkspace = () => {
  router.push('/workspaces?action=create')
}

const openWorkspace = (id: string) => {
  router.push(`/workspace/${id}`)
}

const stopContainer = async (containerId: string) => {
  try {
    await dockerStore.stopContainer(containerId)
    message.success('컨테이너가 중지되었습니다')
  } catch (_error) {
    message.error('컨테이너 중지에 실패했습니다')
  }
}

const refreshData = async () => {
  isRefreshing.value = true
  try {
    await Promise.all([
      dockerStore.refreshContainers(),
      // workspaceStore.fetchWorkspaces() - 실제 API 연동 시 사용
    ])
    message.success('데이터가 새로고침되었습니다')
  } catch (_error) {
    message.error('데이터 새로고침에 실패했습니다')
  } finally {
    isRefreshing.value = false
  }
}

const getStatusType = (status: string): 'success' | 'warning' | 'error' | 'info' => {
  switch (status) {
    case 'active':
      return 'success'
    case 'inactive':
      return 'warning'
    case 'error':
      return 'error'
    default:
      return 'info'
  }
}

const formatStatus = (status: string): string => {
  return formatStatusBadge(status)
}

// 생명주기
onMounted(() => {
  refreshData()
})
</script>

<style lang="scss" scoped>
.dashboard-view {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: $light-bg-secondary;

  .dark & {
    background: $dark-bg-primary;
  }
}

.dashboard-header {
  background: $light-bg-primary;
  border-bottom: 1px solid map-get($gray-colors, 200);
  padding: $spacing-4 $spacing-6;

  .dark & {
    background: $dark-bg-secondary;
    border-bottom-color: $dark-bg-tertiary;
  }

  .header-content {
    @include flex-between;
    max-width: 1200px;
    margin: 0 auto;
  }

  .dashboard-title {
    font-size: $font-size-2xl;
    font-weight: $font-weight-semibold;
    margin: 0;
    color: $light-text-primary;

    .dark & {
      color: $dark-text-primary;
    }
  }

  .header-actions {
    display: flex;
    gap: $spacing-3;
  }
}

.dashboard-main {
  flex: 1;
  overflow: auto;
  padding: $spacing-6;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: $spacing-4;
  margin-bottom: $spacing-8;
  max-width: 1200px;
  margin-left: auto;
  margin-right: auto;

  @include mobile {
    grid-template-columns: repeat(2, 1fr);
    gap: $spacing-3;
  }
}

.stat-card {
  @include card-base;
  @include flex-between;
  padding: $spacing-4;

  .stat-icon {
    flex-shrink: 0;
  }

  .stat-content {
    text-align: right;
  }

  .stat-value {
    font-size: $font-size-2xl;
    font-weight: $font-weight-bold;
    color: $light-text-primary;

    .dark & {
      color: $dark-text-primary;
    }
  }

  .stat-label {
    font-size: $font-size-sm;
    color: $light-text-secondary;
    margin-top: $spacing-1;

    .dark & {
      color: $dark-text-secondary;
    }
  }
}

.content-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: $spacing-6;
  max-width: 1200px;
  margin: 0 auto;

  @include tablet {
    grid-template-columns: 1fr;
  }
}

.content-section {
  @include card-base;
}

.section-header {
  @include flex-between;
  margin-bottom: $spacing-4;
  padding-bottom: $spacing-3;
  border-bottom: 1px solid map-get($gray-colors, 200);

  .dark & {
    border-bottom-color: $dark-bg-tertiary;
  }

  .section-title {
    font-size: $font-size-lg;
    font-weight: $font-weight-semibold;
    margin: 0;
    color: $light-text-primary;

    .dark & {
      color: $dark-text-primary;
    }
  }

  .section-link {
    font-size: $font-size-sm;
    color: map-get($primary-colors, 600);
    text-decoration: none;

    &:hover {
      color: map-get($primary-colors, 700);
    }

    .dark & {
      color: map-get($primary-colors, 400);

      &:hover {
        color: map-get($primary-colors, 300);
      }
    }
  }
}

.workspace-list,
.container-list {
  .empty-state {
    padding: $spacing-8;
  }
}

.workspace-item,
.container-item {
  @include flex-between;
  padding: $spacing-3;
  border-radius: $border-radius-md;
  transition: $transition-base;
  cursor: pointer;

  &:hover {
    background: map-get($gray-colors, 50);

    .dark & {
      background: $dark-bg-tertiary;
    }
  }
}

.workspace-info,
.container-info {
  flex: 1;

  .workspace-name,
  .container-name {
    font-weight: $font-weight-medium;
    color: $light-text-primary;
    margin-bottom: $spacing-1;

    .dark & {
      color: $dark-text-primary;
    }
  }

  .workspace-path,
  .container-image {
    font-size: $font-size-sm;
    color: $light-text-secondary;

    .dark & {
      color: $dark-text-secondary;
    }
  }
}

.workspace-status,
.container-actions {
  flex-shrink: 0;
}
</style>