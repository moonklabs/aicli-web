<template>
  <div class="workspace-list-container">
    <!-- 상단 툴바 -->
    <div class="workspace-toolbar">
      <div class="toolbar-left">
        <h2 class="toolbar-title">워크스페이스</h2>
        <n-badge :value="totalWorkspaces" type="info" />
      </div>
      
      <div class="toolbar-right">
        <n-input-group>
          <n-input 
            v-model:value="workspaceStore.searchQuery.value"
            placeholder="워크스페이스 검색..."
            clearable
            style="width: 240px"
          >
            <template #prefix>
              <n-icon><Search /></n-icon>
            </template>
          </n-input>
          <n-select
            v-model:value="workspaceStore.sortBy.value"
            :options="sortOptions"
            style="width: 150px"
          />
        </n-input-group>
        
        <n-button type="primary" @click="showCreateModal = true">
          <template #icon>
            <n-icon><Plus /></n-icon>
          </template>
          새 워크스페이스
        </n-button>
      </div>
    </div>

    <!-- 필터 및 정렬 -->
    <div v-if="activeFilters.length > 0 || showFilterMenu" class="workspace-filters">
      <n-space>
        <n-tag
          v-for="filter in activeFilters"
          :key="filter.key"
          closable
          @close="workspaceStore.removeFilter(filter.key)"
        >
          {{ filter.label }}: {{ filter.value }}
        </n-tag>
        
        <n-dropdown 
          :options="filterOptions" 
          @select="handleAddFilter"
          trigger="click"
        >
          <n-button size="small" quaternary>
            <template #icon>
              <n-icon><Filter /></n-icon>
            </template>
            필터 추가
          </n-button>
        </n-dropdown>

        <n-button 
          v-if="activeFilters.length > 0"
          size="small" 
          quaternary 
          @click="workspaceStore.clearFilters()"
        >
          <template #icon>
            <n-icon><X /></n-icon>
          </template>
          필터 초기화
        </n-button>
      </n-space>
    </div>

    <!-- 로딩 상태 -->
    <div v-if="isLoading" class="workspace-loading">
      <n-space justify="center">
        <n-spin size="large">
          <template #description>
            워크스페이스 목록을 불러오는 중...
          </template>
        </n-spin>
      </n-space>
    </div>

    <!-- 워크스페이스 그리드 -->
    <div v-else-if="paginatedWorkspaces.length > 0" class="workspace-grid">
      <workspace-card
        v-for="workspace in paginatedWorkspaces"
        :key="workspace.id"
        :workspace="workspace"
        :is-active="workspace.id === currentActiveWorkspace?.id"
        @select="handleSelectWorkspace"
        @action="handleWorkspaceAction"
      />
    </div>

    <!-- 빈 상태 -->
    <n-empty
      v-else
      description="워크스페이스가 없습니다"
      class="workspace-empty"
    >
      <template #icon>
        <n-icon size="48"><FolderOpen /></n-icon>
      </template>
      <template #extra>
        <n-button type="primary" @click="showCreateModal = true">
          첫 워크스페이스 만들기
        </n-button>
      </template>
    </n-empty>

    <!-- 페이지네이션 -->
    <div v-if="totalPages > 1" class="workspace-pagination">
      <n-pagination
        v-model:page="workspaceStore.currentPage.value"
        :page-count="totalPages"
        :page-size="workspaceStore.pageSize.value"
        :show-size-picker="true"
        :page-sizes="[6, 12, 18, 24]"
        show-quick-jumper
        @update:page-size="handlePageSizeChange"
      />
    </div>

    <!-- 워크스페이스 생성 모달 -->
    <workspace-create-modal
      v-model:show="showCreateModal"
      @created="handleWorkspaceCreated"
    />

    <!-- 워크스페이스 설정 모달 -->
    <workspace-settings-modal
      v-model:show="showSettingsModal"
      :workspace="selectedWorkspace"
      @updated="handleWorkspaceUpdated"
    />

    <!-- 삭제 확인 대화상자 -->
    <n-modal
      v-model:show="showDeleteConfirm"
      preset="dialog"
      type="warning"
      title="워크스페이스 삭제"
      positive-text="삭제"
      negative-text="취소"
      @positive-click="confirmDelete"
    >
      <p>
        정말로 <strong>{{ selectedWorkspace?.name }}</strong> 워크스페이스를 삭제하시겠습니까?
      </p>
      <p class="delete-warning">
        이 작업은 되돌릴 수 없으며, 모든 데이터가 영구적으로 삭제됩니다.
      </p>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import {
  NButton,
  NInput,
  NInputGroup,
  NSelect,
  NBadge,
  NSpace,
  NTag,
  NDropdown,
  NIcon,
  NSpin,
  NEmpty,
  NPagination,
  NModal
} from 'naive-ui'
import {
  Search,
  Plus,
  Filter,
  X,
  FolderOpen
} from '@vicons/lucide'

import { useWorkspaceStore } from '@/stores/workspace'
import type { Workspace, WorkspaceFilter } from '@/stores/workspace'
import WorkspaceCard from './WorkspaceCard.vue'
import WorkspaceCreateModal from './WorkspaceCreateModal.vue'
import WorkspaceSettingsModal from './WorkspaceSettingsModal.vue'

const workspaceStore = useWorkspaceStore()
const router = useRouter()
const message = useMessage()

// 로컬 상태
const showCreateModal = ref(false)
const showSettingsModal = ref(false)
const showDeleteConfirm = ref(false)
const showFilterMenu = ref(false)
const selectedWorkspace = ref<Workspace | null>(null)

// 계산된 속성
const totalWorkspaces = computed(() => workspaceStore.totalWorkspaces)
const filteredWorkspaces = computed(() => workspaceStore.filteredWorkspaces)
const paginatedWorkspaces = computed(() => workspaceStore.paginatedWorkspaces)
const activeFilters = computed(() => workspaceStore.activeFilters)
const currentActiveWorkspace = computed(() => workspaceStore.currentActiveWorkspace)
const isLoading = computed(() => workspaceStore.isLoading)
const totalPages = computed(() => workspaceStore.totalPages)

// 정렬 옵션
const sortOptions = [
  { label: '이름', value: 'name' },
  { label: '최근 수정일', value: 'lastModified' },
  { label: '경로', value: 'path' },
  { label: '상태', value: 'status' },
  { label: '파일 수', value: 'fileCount' }
]

// 필터 옵션
const filterOptions = computed(() => [
  {
    label: '상태',
    key: 'status-group',
    type: 'group',
    children: [
      { label: '활성', key: 'status-active', filterKey: 'status', filterValue: 'active' },
      { label: '대기', key: 'status-idle', filterKey: 'status', filterValue: 'idle' },
      { label: '오류', key: 'status-error', filterKey: 'status', filterValue: 'error' }
    ]
  },
  {
    label: 'Git',
    key: 'git-group', 
    type: 'group',
    children: [
      { label: 'Git 연결됨', key: 'has-git-true', filterKey: 'hasGit', filterValue: 'true' },
      { label: 'Git 연결 안됨', key: 'has-git-false', filterKey: 'hasGit', filterValue: 'false' }
    ]
  },
  {
    label: 'Claude 세션',
    key: 'claude-group',
    type: 'group', 
    children: [
      { label: 'Claude 활성', key: 'has-claude-true', filterKey: 'hasClaudeSession', filterValue: 'true' },
      { label: 'Claude 비활성', key: 'has-claude-false', filterKey: 'hasClaudeSession', filterValue: 'false' }
    ]
  },
  {
    label: '컨테이너 상태',
    key: 'container-group',
    type: 'group',
    children: [
      { label: '실행중', key: 'container-success', filterKey: 'containerStatus', filterValue: 'success' },
      { label: '중지됨', key: 'container-warning', filterKey: 'containerStatus', filterValue: 'warning' },
      { label: '오류', key: 'container-error', filterKey: 'containerStatus', filterValue: 'error' }
    ]
  },
  {
    label: '작업',
    key: 'task-group',
    type: 'group',
    children: [
      { label: '작업 진행중', key: 'has-task-true', filterKey: 'hasCurrentTask', filterValue: 'true' },
      { label: '작업 없음', key: 'has-task-false', filterKey: 'hasCurrentTask', filterValue: 'false' }
    ]
  }
])

// 이벤트 핸들러
const handleSelectWorkspace = async (workspaceId: string): Promise<void> => {
  try {
    await workspaceStore.activateWorkspace(workspaceId)
    await router.push(`/workspace/${workspaceId}`)
    message.success('워크스페이스가 활성화되었습니다')
  } catch (error) {
    message.error('워크스페이스 활성화에 실패했습니다')
    console.error('Workspace activation error:', error)
  }
}

const handleWorkspaceAction = async (action: string, workspaceId: string): Promise<void> => {
  const workspace = workspaceStore.workspaceById(workspaceId)
  if (!workspace) return

  selectedWorkspace.value = workspace

  try {
    switch (action) {
      case 'activate':
        await workspaceStore.startWorkspace(workspaceId)
        message.success(`${workspace.name} 워크스페이스가 활성화되었습니다`)
        break
      case 'pause':
        // TODO: 일시정지 기능 구현
        message.info('일시정지 기능은 준비 중입니다')
        break
      case 'stop':
        await workspaceStore.stopWorkspace(workspaceId)
        message.success(`${workspace.name} 워크스페이스가 중지되었습니다`)
        break
      case 'settings':
        showSettingsModal.value = true
        break
      case 'open-browser':
        // TODO: 브라우저에서 열기 기능 구현
        window.open(`/workspace/${workspaceId}`, '_blank')
        break
      case 'delete':
        showDeleteConfirm.value = true
        break
    }
  } catch (error) {
    message.error(`작업 실행에 실패했습니다: ${error}`)
  }
}

const handleAddFilter = (key: string, option: any): void => {
  if (option.filterKey && option.filterValue) {
    const filter: WorkspaceFilter = {
      key: option.filterKey,
      label: option.label,
      value: option.filterValue
    }
    workspaceStore.addFilter(filter)
  }
}

const handlePageSizeChange = (pageSize: number): void => {
  workspaceStore.pageSize.value = pageSize
  workspaceStore.currentPage.value = 1
}

const handleWorkspaceCreated = (workspace: Workspace): void => {
  message.success(`${workspace.name} 워크스페이스가 생성되었습니다`)
  showCreateModal.value = false
}

const handleWorkspaceUpdated = (workspace: Workspace): void => {
  message.success(`${workspace.name} 워크스페이스가 업데이트되었습니다`)
  showSettingsModal.value = false
  selectedWorkspace.value = null
}

const confirmDelete = async (): Promise<void> => {
  if (!selectedWorkspace.value) return

  try {
    await workspaceStore.deleteWorkspace(selectedWorkspace.value.id)
    message.success(`${selectedWorkspace.value.name} 워크스페이스가 삭제되었습니다`)
    showDeleteConfirm.value = false
    selectedWorkspace.value = null
  } catch (error) {
    message.error('워크스페이스 삭제에 실패했습니다')
  }
}

// 생명주기
onMounted(async () => {
  try {
    await workspaceStore.fetchWorkspaces()
  } catch (error) {
    message.error('워크스페이스 목록을 불러오는데 실패했습니다')
  }
})
</script>

<style scoped>
.workspace-list-container {
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 20px;
}

.workspace-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 20px;
  margin-bottom: 16px;
}

.toolbar-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.toolbar-title {
  margin: 0;
  font-size: 24px;
  font-weight: 700;
  color: var(--n-text-color);
}

.toolbar-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.workspace-filters {
  padding: 16px;
  background-color: var(--n-color-hover);
  border-radius: 8px;
  border: 1px solid var(--n-border-color);
}

.workspace-loading {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 300px;
}

.workspace-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 20px;
  flex: 1;
}

.workspace-empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 400px;
}

.workspace-pagination {
  display: flex;
  justify-content: center;
  padding: 20px 0;
  border-top: 1px solid var(--n-border-color);
}

.delete-warning {
  color: var(--n-color-error);
  font-size: 12px;
  margin-top: 8px;
}

/* 반응형 디자인 */
@media (max-width: 1200px) {
  .workspace-grid {
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: 16px;
  }
}

@media (max-width: 768px) {
  .workspace-list-container {
    padding: 16px;
    gap: 16px;
  }

  .workspace-toolbar {
    flex-direction: column;
    align-items: stretch;
    gap: 16px;
  }

  .toolbar-right {
    flex-direction: column;
    gap: 12px;
  }

  .workspace-grid {
    grid-template-columns: 1fr;
    gap: 16px;
  }

  .toolbar-title {
    font-size: 20px;
  }
}

@media (max-width: 480px) {
  .workspace-list-container {
    padding: 12px;
  }

  .workspace-filters {
    padding: 12px;
  }
}
</style>