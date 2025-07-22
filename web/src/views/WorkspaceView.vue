<template>
  <div class="workspace-view">
    <header class="workspace-header">
      <h1 class="workspace-title">워크스페이스 관리</h1>
      <p class="workspace-description">
        프로젝트별 독립된 환경을 생성하고 관리할 수 있습니다.
      </p>
    </header>

    <div class="workspace-content">
      <!-- 워크스페이스 통계 -->
      <div class="workspace-stats">
        <NCard class="stat-card">
          <NStatistic label="전체 워크스페이스" :value="totalWorkspaces" />
        </NCard>
        <NCard class="stat-card">
          <NStatistic
            label="활성 워크스페이스"
            :value="activeWorkspaces.length"
          />
        </NCard>
        <NCard class="stat-card">
          <NStatistic
            label="비활성 워크스페이스"
            :value="inactiveWorkspaces.length"
          />
        </NCard>
      </div>

      <!-- 워크스페이스 목록 -->
      <NCard class="workspace-list-card">
        <template #header>
          <div class="list-header">
            <h2>워크스페이스 목록</h2>
            <NButton type="primary" @click="showCreateModal = true">
              <template #icon>
                <NIcon><PlusIcon /></NIcon>
              </template>
              새 워크스페이스 생성
            </NButton>
          </div>
        </template>

        <div class="workspace-grid">
          <div
            v-for="workspace in workspaces"
            :key="workspace.id"
            class="workspace-card"
            @click="selectWorkspace(workspace)"
          >
            <div class="workspace-status">
              <NBadge
                :type="getStatusType(workspace.status)"
                :text="getStatusText(workspace.status)"
              />
            </div>
            <h3 class="workspace-name">{{ workspace.name }}</h3>
            <p class="workspace-path">{{ workspace.path }}</p>
            <p v-if="workspace.description" class="workspace-desc">
              {{ workspace.description }}
            </p>
            <div class="workspace-actions">
              <NButton
                v-if="workspace.status === 'inactive'"
                type="primary"
                size="small"
                @click.stop="startWorkspace(workspace.id)"
                :loading="isLoading"
              >
                시작
              </NButton>
              <NButton
                v-if="workspace.status === 'active'"
                type="warning"
                size="small"
                @click.stop="stopWorkspace(workspace.id)"
                :loading="isLoading"
              >
                중지
              </NButton>
              <NButton
                type="error"
                size="small"
                @click.stop="deleteWorkspace(workspace.id)"
                :loading="isLoading"
              >
                삭제
              </NButton>
            </div>
          </div>
        </div>

        <NEmpty v-if="workspaces.length === 0" description="워크스페이스가 없습니다">
          <template #extra>
            <NButton type="primary" @click="showCreateModal = true">
              첫 번째 워크스페이스 생성
            </NButton>
          </template>
        </NEmpty>
      </NCard>
    </div>

    <!-- 워크스페이스 생성 모달 -->
    <NModal v-model:show="showCreateModal" preset="dialog" title="새 워크스페이스 생성">
      <NForm
        ref="createFormRef"
        :model="createForm"
        :rules="createRules"
        label-placement="top"
      >
        <NFormItem label="워크스페이스 이름" path="name">
          <NInput
            v-model:value="createForm.name"
            placeholder="워크스페이스 이름을 입력하세요"
          />
        </NFormItem>
        <NFormItem label="프로젝트 경로" path="path">
          <NInput
            v-model:value="createForm.path"
            placeholder="/path/to/project"
          />
        </NFormItem>
        <NFormItem label="설명" path="description">
          <NInput
            v-model:value="createForm.description"
            type="textarea"
            placeholder="워크스페이스 설명 (선택사항)"
            :rows="3"
          />
        </NFormItem>
      </NForm>
      <template #action>
        <NSpace>
          <NButton @click="showCreateModal = false">취소</NButton>
          <NButton type="primary" @click="handleCreateWorkspace" :loading="isLoading">
            생성
          </NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import {
  NBadge,
  NButton,
  NCard,
  NEmpty,
  NForm,
  NFormItem,
  NIcon,
  NInput,
  NModal,
  NSpace,
  NStatistic,
} from 'naive-ui'
import { useWorkspaceStore } from '@/stores/workspace'
import type { Workspace } from '@/stores/workspace'

// 아이콘 (실제로는 @vicons 패키지에서 import)
const PlusIcon = {
  render: () => '+',
}

const router = useRouter()
const message = useMessage()
const workspaceStore = useWorkspaceStore()

// 상태
const showCreateModal = ref(false)
const createFormRef = ref()
const createForm = ref({
  name: '',
  path: '',
  description: '',
})

// 폼 검증 규칙
const createRules = {
  name: {
    required: true,
    message: '워크스페이스 이름을 입력해주세요',
    trigger: 'blur',
  },
  path: {
    required: true,
    message: '프로젝트 경로를 입력해주세요',
    trigger: 'blur',
  },
}

// 계산된 속성
const workspaces = computed(() => workspaceStore.workspaces)
const activeWorkspaces = computed(() => workspaceStore.activeWorkspaces)
const inactiveWorkspaces = computed(() => workspaceStore.inactiveWorkspaces)
const totalWorkspaces = computed(() => workspaceStore.totalWorkspaces)
const isLoading = computed(() => workspaceStore.isLoading)

// 메서드
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

const selectWorkspace = (workspace: Workspace) => {
  workspaceStore.setActiveWorkspace(workspace)
  router.push({ name: 'workspace-detail', params: { id: workspace.id } })
}

const startWorkspace = async (id: string) => {
  try {
    await workspaceStore.startWorkspace(id)
    message.success('워크스페이스가 시작되었습니다')
  } catch (_error) {
    message.error('워크스페이스 시작에 실패했습니다')
  }
}

const stopWorkspace = async (id: string) => {
  try {
    await workspaceStore.stopWorkspace(id)
    message.success('워크스페이스가 중지되었습니다')
  } catch (_error) {
    message.error('워크스페이스 중지에 실패했습니다')
  }
}

const deleteWorkspace = async (id: string) => {
  try {
    await workspaceStore.deleteWorkspace(id)
    message.success('워크스페이스가 삭제되었습니다')
  } catch (_error) {
    message.error('워크스페이스 삭제에 실패했습니다')
  }
}

const handleCreateWorkspace = async () => {
  try {
    await createFormRef.value.validate()
    await workspaceStore.createWorkspace({
      name: createForm.value.name,
      path: createForm.value.path,
      description: createForm.value.description,
    })
    message.success('워크스페이스가 생성되었습니다')
    showCreateModal.value = false
    createForm.value = { name: '', path: '', description: '' }
  } catch (_error) {
    message.error('워크스페이스 생성에 실패했습니다')
  }
}
</script>

<style lang="scss" scoped>
.workspace-view {
  padding: 1.5rem;
  max-width: 1200px;
  margin: 0 auto;
}

.workspace-header {
  margin-bottom: 2rem;
  text-align: center;

  .workspace-title {
    font-size: 2rem;
    font-weight: 600;
    margin-bottom: 0.5rem;
    color: var(--text-color-1);
  }

  .workspace-description {
    color: var(--text-color-2);
    font-size: 1.1rem;
  }
}

.workspace-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1rem;
  margin-bottom: 2rem;

  .stat-card {
    text-align: center;
  }
}

.workspace-list-card {
  .list-header {
    display: flex;
    justify-content: space-between;
    align-items: center;

    h2 {
      margin: 0;
      font-size: 1.5rem;
      font-weight: 600;
    }
  }
}

.workspace-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 1.5rem;
  margin-top: 1rem;
}

.workspace-card {
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 1.5rem;
  cursor: pointer;
  transition: all 0.2s ease;
  background: var(--card-color);

  &:hover {
    border-color: var(--primary-color);
    box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
  }

  .workspace-status {
    margin-bottom: 1rem;
  }

  .workspace-name {
    margin: 0 0 0.5rem 0;
    font-size: 1.25rem;
    font-weight: 600;
    color: var(--text-color-1);
  }

  .workspace-path {
    margin: 0 0 0.5rem 0;
    font-size: 0.9rem;
    color: var(--text-color-3);
    font-family: monospace;
  }

  .workspace-desc {
    margin: 0 0 1rem 0;
    font-size: 0.9rem;
    color: var(--text-color-2);
    line-height: 1.4;
  }

  .workspace-actions {
    display: flex;
    gap: 0.5rem;
    margin-top: 1rem;
  }
}
</style>