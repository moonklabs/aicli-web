<template>
  <div class="file-tree">
    <div class="tree-header">
      <div class="tree-title">
        <n-icon class="title-icon"><Folder /></n-icon>
        <span class="title-text">{{ workspace?.name || 'Workspace' }}</span>
        <n-badge :value="visibleFileCount" type="info" size="small" />
      </div>
      
      <div class="tree-actions">
        <n-input
          v-model:value="searchQuery"
          placeholder="파일 검색..."
          size="small"
          clearable
          class="search-input"
        >
          <template #prefix>
            <n-icon><Search /></n-icon>
          </template>
        </n-input>
        
        <n-button size="small" quaternary @click="refreshTree" :loading="isLoading">
          <template #icon>
            <n-icon><RefreshCw /></n-icon>
          </template>
        </n-button>
        
        <n-button 
          size="small" 
          quaternary 
          @click="toggleHiddenFiles"
          :type="showHidden ? 'primary' : 'default'"
        >
          <template #icon>
            <n-icon><Eye :style="{ opacity: showHidden ? 1 : 0.5 }" /></n-icon>
          </template>
        </n-button>
        
        <n-dropdown 
          :options="filterOptions" 
          @select="handleFilterSelect"
          placement="bottom-end"
        >
          <n-button size="small" quaternary>
            <template #icon>
              <n-icon><Filter /></n-icon>
            </template>
          </n-button>
        </n-dropdown>
      </div>
    </div>

    <!-- Git 상태 요약 -->
    <div v-if="gitStatus" class="git-status-summary">
      <n-space size="small">
        <n-tag size="small" type="info">
          <template #icon>
            <n-icon><GitBranch /></n-icon>
          </template>
          {{ gitStatus.branch }}
        </n-tag>
        
        <n-tag 
          v-if="gitStatus.changedFiles.length > 0" 
          size="small" 
          type="warning"
        >
          <template #icon>
            <n-icon><FileEdit /></n-icon>
          </template>
          {{ gitStatus.changedFiles.length }}개 수정됨
        </n-tag>
        
        <n-tag 
          v-if="gitStatus.untrackedFiles.length > 0" 
          size="small" 
          type="info"
        >
          <template #icon>
            <n-icon><FilePlus /></n-icon>
          </template>
          {{ gitStatus.untrackedFiles.length }}개 추가됨
        </n-tag>
        
        <n-tag 
          v-if="gitStatus.stagedFiles.length > 0" 
          size="small" 
          type="success"
        >
          <template #icon>
            <n-icon><Check /></n-icon>
          </template>
          {{ gitStatus.stagedFiles.length }}개 스테이징됨
        </n-tag>
      </n-space>
    </div>

    <!-- 파일 트리 -->
    <div class="tree-content">
      <div v-if="isLoading" class="tree-loading">
        <n-spin size="small">
          <template #description>
            파일 트리를 불러오는 중...
          </template>
        </n-spin>
      </div>

      <div v-else-if="!treeData || treeData.length === 0" class="tree-empty">
        <n-empty description="파일이 없습니다" size="small">
          <template #icon>
            <n-icon size="48"><Folder /></n-icon>
          </template>
          <template #extra>
            <n-button size="small" @click="refreshTree">
              새로고침
            </n-button>
          </template>
        </n-empty>
      </div>

      <n-tree
        v-else
        :data="treeData"
        :selectable="true"
        :checkable="false"
        :show-line="true"
        :expand-on-click="false"
        :selected-keys="selectedKeys"
        :expanded-keys="expandedKeys"
        @update:selected-keys="handleSelect"
        @update:expanded-keys="handleExpand"
        class="workspace-tree"
        virtual-scroll
        :height="400"
      >
        <template #default="{ option }">
          <div 
            class="tree-node"
            :class="{ 
              'is-file': !option.isDirectory,
              'is-directory': option.isDirectory,
              'is-git-ignored': option.isGitIgnored,
              'is-modified': option.isModified,
              'is-selected': selectedKeys.includes(option.key)
            }"
            @contextmenu.prevent="showContextMenu($event, option)"
            @dblclick="handleDoubleClick(option)"
          >
            <div class="node-content">
              <n-icon class="node-icon" :class="{ 'expanded': option.isExpanded }">
                <component :is="getFileIcon(option)" />
              </n-icon>
              
              <span class="node-label">{{ option.name }}</span>
              
              <!-- Git 상태 표시 -->
              <n-tag
                v-if="option.gitStatus"
                :type="getGitStatusType(option.gitStatus)"
                size="tiny"
                class="git-status"
              >
                {{ getGitStatusText(option.gitStatus) }}
              </n-tag>
              
              <!-- 파일 크기 표시 -->
              <span 
                v-if="!option.isDirectory && option.size !== undefined"
                class="file-size"
              >
                {{ formatBytes(option.size) }}
              </span>

              <!-- 로딩 스피너 -->
              <n-spin v-if="option.isLoading" size="tiny" class="node-loading" />
            </div>
          </div>
        </template>
      </n-tree>
    </div>

    <!-- 컨텍스트 메뉴 -->
    <n-dropdown
      :show="showContextMenuFlag"
      :options="contextMenuOptions"
      :x="contextMenuX"
      :y="contextMenuY"
      @select="handleContextMenuSelect"
      @clickoutside="closeContextMenu"
      placement="bottom-start"
    />

    <!-- 파일 생성 모달 -->
    <n-modal
      v-model:show="showCreateModal"
      preset="dialog"
      title="새 파일/폴더 만들기"
      positive-text="만들기"
      negative-text="취소"
      @positive-click="createNewFile"
    >
      <n-form ref="createFormRef" :model="createForm" :rules="createFormRules">
        <n-form-item label="유형" path="type">
          <n-radio-group v-model:value="createForm.type">
            <n-radio value="file">파일</n-radio>
            <n-radio value="directory">폴더</n-radio>
          </n-radio-group>
        </n-form-item>
        
        <n-form-item label="이름" path="name">
          <n-input
            v-model:value="createForm.name"
            placeholder="파일 또는 폴더 이름을 입력하세요"
            @keydown.enter="createNewFile"
          />
        </n-form-item>
        
        <n-form-item label="위치">
          <n-input
            :value="createForm.parentPath"
            readonly
            placeholder="선택된 위치"
          />
        </n-form-item>
      </n-form>
    </n-modal>

    <!-- 이름 변경 모달 -->
    <n-modal
      v-model:show="showRenameModal"
      preset="dialog"
      title="이름 바꾸기"
      positive-text="변경"
      negative-text="취소"
      @positive-click="renameFile"
    >
      <n-form ref="renameFormRef" :model="renameForm" :rules="renameFormRules">
        <n-form-item label="새 이름" path="name">
          <n-input
            v-model:value="renameForm.name"
            placeholder="새 이름을 입력하세요"
            @keydown.enter="renameFile"
          />
        </n-form-item>
      </n-form>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, nextTick } from 'vue'
import { useMessage } from 'naive-ui'
import {
  NTree,
  NIcon,
  NButton,
  NBadge,
  NInput,
  NDropdown,
  NSpace,
  NTag,
  NSpin,
  NEmpty,
  NModal,
  NForm,
  NFormItem,
  NRadioGroup,
  NRadio
} from 'naive-ui'
import {
  Folder,
  FolderOpen,
  File,
  FileText,
  Image,
  Code,
  Database,
  Archive,
  RefreshCw,
  Eye,
  Search,
  Filter,
  GitBranch,
  FileEdit,
  FilePlus,
  Check,
  Plus,
  Edit,
  Trash2,
  Copy,
  Scissors,
  Download
} from '@vicons/lucide'

import { useFileTreeStore, type FileTreeNode } from '@/stores/fileTree'
import { useWorkspaceStore } from '@/stores/workspace'

interface Props {
  workspaceId?: string
}

const props = defineProps<Props>()

const fileTreeStore = useFileTreeStore()
const workspaceStore = useWorkspaceStore()
const message = useMessage()

// 로컬 상태
const selectedKeys = ref<string[]>([])
const expandedKeys = ref<string[]>([])
const searchQuery = ref('')
const showHidden = ref(false)
const showContextMenuFlag = ref(false)
const contextMenuX = ref(0)
const contextMenuY = ref(0)
const selectedNode = ref<FileTreeNode | null>(null)

// 모달 상태
const showCreateModal = ref(false)
const showRenameModal = ref(false)
const createFormRef = ref()
const renameFormRef = ref()

// 폼 데이터
const createForm = ref({
  type: 'file' as 'file' | 'directory',
  name: '',
  parentPath: '/'
})

const renameForm = ref({
  name: '',
  oldPath: ''
})

// 폼 규칙
const createFormRules = {
  name: [
    { required: true, message: '이름을 입력해주세요' },
    { pattern: /^[^/\\:*?"<>|]+$/, message: '파일명에 사용할 수 없는 문자가 포함되어 있습니다' }
  ]
}

const renameFormRules = {
  name: [
    { required: true, message: '이름을 입력해주세요' },
    { pattern: /^[^/\\:*?"<>|]+$/, message: '파일명에 사용할 수 없는 문자가 포함되어 있습니다' }
  ]
}

// 계산된 속성
const workspace = computed(() => {
  if (!props.workspaceId) return null
  return workspaceStore.workspaceById(props.workspaceId)
})

const isLoading = computed(() => fileTreeStore.isLoading)

const gitStatus = computed(() => {
  if (!props.workspaceId) return null
  return fileTreeStore.getGitStatus(props.workspaceId)
})

const filteredTree = computed(() => {
  if (!props.workspaceId) return null
  
  // 필터 업데이트
  fileTreeStore.updateFilter({
    showHidden: showHidden.value,
    searchQuery: searchQuery.value
  })
  
  return fileTreeStore.filteredTree(props.workspaceId)
})

const treeData = computed(() => {
  if (!filteredTree.value) return []
  return buildTreeData(filteredTree.value)
})

const visibleFileCount = computed(() => {
  if (!treeData.value) return 0
  return countVisibleFiles(treeData.value)
})

// 필터 옵션
const filterOptions = [
  {
    label: '파일 확장자별 필터링',
    key: 'extensions',
    children: [
      { label: 'JavaScript/TypeScript', key: 'js,ts,jsx,tsx' },
      { label: 'Vue 파일', key: 'vue' },
      { label: '스타일시트', key: 'css,scss,sass,less' },
      { label: '이미지', key: 'png,jpg,jpeg,gif,svg,webp' },
      { label: '문서', key: 'md,txt,json,yaml,yml' }
    ]
  },
  {
    label: '필터 초기화',
    key: 'reset'
  }
]

// 컨텍스트 메뉴 옵션
const contextMenuOptions = computed(() => {
  if (!selectedNode.value) return []
  
  const options = []
  
  if (selectedNode.value.isDirectory) {
    options.push(
      { label: '새 파일', key: 'create-file', icon: () => h(NIcon, null, { default: () => h(File) }) },
      { label: '새 폴더', key: 'create-folder', icon: () => h(NIcon, null, { default: () => h(Folder) }) },
      { type: 'divider' }
    )
  }
  
  options.push(
    { label: '이름 바꾸기', key: 'rename', icon: () => h(NIcon, null, { default: () => h(Edit) }) },
    { label: '복사', key: 'copy', icon: () => h(NIcon, null, { default: () => h(Copy) }) },
    { label: '잘라내기', key: 'cut', icon: () => h(NIcon, null, { default: () => h(Scissors) }) }
  )
  
  if (!selectedNode.value.isDirectory) {
    options.push(
      { label: '다운로드', key: 'download', icon: () => h(NIcon, null, { default: () => h(Download) }) }
    )
  }
  
  options.push(
    { type: 'divider' },
    { label: '삭제', key: 'delete', icon: () => h(NIcon, null, { default: () => h(Trash2) }) }
  )
  
  return options
})

// 트리 데이터 변환
const buildTreeData = (node: FileTreeNode): any[] => {
  if (!node.children) return []
  
  return node.children
    .sort((a, b) => {
      // 디렉토리 우선, 그 다음 알파벳 순
      if (a.isDirectory !== b.isDirectory) {
        return a.isDirectory ? -1 : 1
      }
      return a.name.localeCompare(b.name)
    })
    .map(child => ({
      key: child.key,
      label: child.name,
      name: child.name,
      path: child.path,
      isDirectory: child.isDirectory,
      isGitIgnored: child.isGitIgnored,
      isModified: child.isModified,
      gitStatus: child.gitStatus,
      size: child.size,
      isLoading: child.isLoading,
      isExpanded: child.isExpanded,
      children: child.isDirectory ? buildTreeData(child) : undefined
    }))
}

// 파일 개수 계산
const countVisibleFiles = (nodes: any[]): number => {
  let count = 0
  for (const node of nodes) {
    count++
    if (node.children) {
      count += countVisibleFiles(node.children)
    }
  }
  return count
}

// 파일 아이콘 결정
const getFileIcon = (node: any) => {
  if (node.isDirectory) {
    return node.isExpanded ? FolderOpen : Folder
  }
  
  const ext = getFileExtension(node.name)
  switch (ext) {
    case 'vue': return Code
    case 'js': case 'ts': case 'jsx': case 'tsx': return Code
    case 'css': case 'scss': case 'sass': case 'less': return Code
    case 'html': case 'htm': return Code
    case 'json': case 'yaml': case 'yml': return Database
    case 'md': case 'txt': return FileText
    case 'png': case 'jpg': case 'jpeg': case 'gif': case 'svg': case 'webp': return Image
    case 'zip': case 'tar': case 'gz': case '7z': return Archive
    default: return File
  }
}

const getFileExtension = (filename: string): string | null => {
  const lastDot = filename.lastIndexOf('.')
  if (lastDot === -1 || lastDot === 0) return null
  return filename.substring(lastDot + 1).toLowerCase()
}

// Git 상태 관련
const getGitStatusType = (status: string): 'success' | 'warning' | 'error' | 'info' => {
  switch (status) {
    case 'added': case 'staged': return 'success'
    case 'modified': return 'warning'
    case 'deleted': return 'error'
    case 'untracked': return 'info'
    default: return 'info'
  }
}

const getGitStatusText = (status: string): string => {
  switch (status) {
    case 'modified': return 'M'
    case 'added': return 'A'
    case 'deleted': return 'D'
    case 'untracked': return '?'
    case 'renamed': return 'R'
    case 'copied': return 'C'
    default: return status.charAt(0).toUpperCase()
  }
}

// 바이트 포맷팅
const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 B'
  
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

// 이벤트 핸들러
const handleSelect = (keys: string[]): void => {
  selectedKeys.value = keys
  if (keys.length > 0) {
    fileTreeStore.selectNode(keys[0])
  }
}

const handleExpand = async (keys: string[]): Promise<void> => {
  expandedKeys.value = keys
  
  // 새로 확장된 디렉토리 찾기
  const newlyExpanded = keys.filter(key => !fileTreeStore.expandedKeys.has(key))
  
  for (const key of newlyExpanded) {
    if (props.workspaceId) {
      await fileTreeStore.toggleExpanded(key, props.workspaceId)
    }
  }
}

const handleDoubleClick = (node: any): void => {
  if (node.isDirectory) {
    const isExpanded = expandedKeys.value.includes(node.key)
    if (isExpanded) {
      expandedKeys.value = expandedKeys.value.filter(key => key !== node.key)
    } else {
      expandedKeys.value.push(node.key)
    }
    handleExpand(expandedKeys.value)
  } else {
    // 파일 열기 (향후 에디터 통합)
    message.info(`파일 열기: ${node.name}`)
  }
}

const toggleHiddenFiles = (): void => {
  showHidden.value = !showHidden.value
  message.info(showHidden.value ? '숨김 파일 표시' : '숨김 파일 숨김')
}

const refreshTree = async (): Promise<void> => {
  if (!props.workspaceId) return
  
  try {
    await fileTreeStore.loadWorkspaceTree(props.workspaceId)
    await fileTreeStore.refreshGitStatus(props.workspaceId)
    message.success('파일 트리가 새로고침되었습니다')
  } catch (error) {
    message.error('파일 트리 새로고침에 실패했습니다')
  }
}

const handleFilterSelect = (key: string): void => {
  if (key === 'reset') {
    fileTreeStore.resetFilter()
    message.info('필터가 초기화되었습니다')
  } else if (key.includes(',')) {
    const extensions = key.split(',')
    fileTreeStore.updateFilter({ fileExtensions: extensions })
    message.info(`${extensions.length}개 확장자로 필터링되었습니다`)
  }
}

// 컨텍스트 메뉴 관련
const showContextMenu = (event: MouseEvent, node: any): void => {
  selectedNode.value = node
  contextMenuX.value = event.clientX
  contextMenuY.value = event.clientY
  showContextMenuFlag.value = true
}

const closeContextMenu = (): void => {
  showContextMenuFlag.value = false
  selectedNode.value = null
}

const handleContextMenuSelect = (key: string): void => {
  closeContextMenu()
  
  if (!selectedNode.value) return
  
  switch (key) {
    case 'create-file':
      openCreateModal('file')
      break
    case 'create-folder':
      openCreateModal('directory')
      break
    case 'rename':
      openRenameModal()
      break
    case 'copy':
      copyFile()
      break
    case 'cut':
      cutFile()
      break
    case 'download':
      downloadFile()
      break
    case 'delete':
      deleteFile()
      break
  }
}

// 파일 작업
const openCreateModal = (type: 'file' | 'directory'): void => {
  createForm.value = {
    type,
    name: '',
    parentPath: selectedNode.value?.path || '/'
  }
  showCreateModal.value = true
}

const openRenameModal = (): void => {
  if (!selectedNode.value) return
  
  renameForm.value = {
    name: selectedNode.value.name,
    oldPath: selectedNode.value.path
  }
  showRenameModal.value = true
}

const createNewFile = async (): Promise<void> => {
  if (!createFormRef.value || !props.workspaceId) return
  
  try {
    await createFormRef.value.validate()
    
    await fileTreeStore.createFile(
      props.workspaceId,
      createForm.value.parentPath,
      createForm.value.name,
      createForm.value.type === 'directory'
    )
    
    showCreateModal.value = false
    message.success(`${createForm.value.type === 'directory' ? '폴더' : '파일'}가 생성되었습니다`)
  } catch (error) {
    message.error('생성에 실패했습니다')
  }
}

const renameFile = async (): Promise<void> => {
  if (!renameFormRef.value || !props.workspaceId) return
  
  try {
    await renameFormRef.value.validate()
    
    await fileTreeStore.renameFile(
      props.workspaceId,
      renameForm.value.oldPath,
      renameForm.value.name
    )
    
    showRenameModal.value = false
    message.success('이름이 변경되었습니다')
  } catch (error) {
    message.error('이름 변경에 실패했습니다')
  }
}

const copyFile = (): void => {
  // TODO: 클립보드 기능 구현
  message.info('복사 기능은 준비 중입니다')
}

const cutFile = (): void => {
  // TODO: 클립보드 기능 구현
  message.info('잘라내기 기능은 준비 중입니다')
}

const downloadFile = (): void => {
  if (!selectedNode.value || selectedNode.value.isDirectory) return
  // TODO: 파일 다운로드 구현
  message.info('다운로드 기능은 준비 중입니다')
}

const deleteFile = async (): Promise<void> => {
  if (!selectedNode.value || !props.workspaceId) return
  
  try {
    await fileTreeStore.deleteFile(props.workspaceId, selectedNode.value.path)
    message.success('삭제되었습니다')
  } catch (error) {
    message.error('삭제에 실패했습니다')
  }
}

// 생명주기
onMounted(async () => {
  if (props.workspaceId) {
    await fileTreeStore.loadWorkspaceTree(props.workspaceId)
  }
})

// 워크스페이스 변경 감지
watch(() => props.workspaceId, async (newWorkspaceId) => {
  if (newWorkspaceId) {
    selectedKeys.value = []
    expandedKeys.value = []
    await fileTreeStore.loadWorkspaceTree(newWorkspaceId)
  }
})

// 검색 쿼리 변경 감지
watch(searchQuery, () => {
  // 검색 시 자동으로 필터 적용
})
</script>

<style scoped>
.file-tree {
  display: flex;
  flex-direction: column;
  height: 100%;
  border: 1px solid var(--n-border-color);
  border-radius: 8px;
  background-color: var(--n-card-color);
}

.tree-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid var(--n-border-color);
  background-color: var(--n-color-hover);
}

.tree-title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.title-icon {
  color: var(--n-color-primary);
}

.title-text {
  font-weight: 600;
  color: var(--n-text-color);
}

.tree-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.search-input {
  width: 180px;
}

.git-status-summary {
  padding: 8px 16px;
  border-bottom: 1px solid var(--n-border-color);
  background-color: var(--n-color-hover);
}

.tree-content {
  flex: 1;
  overflow: hidden;
}

.tree-loading,
.tree-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 200px;
  color: var(--n-text-color-3);
}

.workspace-tree {
  height: 100%;
  padding: 8px;
}

.tree-node {
  width: 100%;
  padding: 2px 4px;
  border-radius: 4px;
  transition: all 0.2s ease;
  cursor: pointer;
}

.tree-node:hover {
  background-color: var(--n-color-hover);
}

.tree-node.is-selected {
  background-color: var(--n-color-primary-hover);
  color: var(--n-color-primary);
}

.tree-node.is-git-ignored {
  opacity: 0.6;
}

.tree-node.is-modified {
  font-weight: 500;
}

.node-content {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
}

.node-icon {
  flex-shrink: 0;
  transition: transform 0.2s ease;
}

.node-icon.expanded {
  transform: rotate(90deg);
}

.node-label {
  flex: 1;
  font-size: 14px;
  line-height: 1.4;
  word-break: break-word;
}

.git-status {
  flex-shrink: 0;
  font-size: 10px;
  font-weight: bold;
  min-width: 16px;
  text-align: center;
}

.file-size {
  flex-shrink: 0;
  font-size: 11px;
  color: var(--n-text-color-3);
}

.node-loading {
  flex-shrink: 0;
}

/* 파일 유형별 색상 */
.tree-node.is-directory .node-label {
  color: var(--n-color-primary);
  font-weight: 500;
}

.tree-node.is-file .node-label {
  color: var(--n-text-color);
}

/* Git 상태별 색상 */
.tree-node.is-modified .node-label {
  color: var(--n-color-warning);
}

/* 반응형 디자인 */
@media (max-width: 768px) {
  .tree-header {
    flex-direction: column;
    align-items: stretch;
    gap: 12px;
  }

  .tree-actions {
    justify-content: space-between;
  }

  .search-input {
    width: 100%;
  }

  .git-status-summary {
    padding: 8px 12px;
  }
}

/* 스크롤바 스타일 */
.workspace-tree :deep(.n-scrollbar-content) {
  padding-right: 8px;
}

.workspace-tree :deep(.n-tree-node-content) {
  padding: 0 !important;
}

.workspace-tree :deep(.n-tree-node-content-wrapper) {
  padding: 0 !important;
}
</style>