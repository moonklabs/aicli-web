import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export interface FileTreeNode {
  key: string
  path: string
  name: string
  isDirectory: boolean
  isGitIgnored: boolean
  isModified: boolean
  gitStatus?: 'modified' | 'added' | 'deleted' | 'untracked' | 'renamed' | 'copied'
  size?: number
  lastModified?: Date
  children?: FileTreeNode[]
  isExpanded?: boolean
  isLoading?: boolean
  level: number
  parentPath?: string
}

export interface FileTreeFilter {
  showHidden: boolean
  showGitIgnored: boolean
  fileExtensions: string[]
  searchQuery: string
}

export interface GitStatus {
  branch: string
  hasChanges: boolean
  changedFiles: string[]
  untrackedFiles: string[]
  stagedFiles: string[]
}

export const useFileTreeStore = defineStore('fileTree', () => {
  // 상태
  const workspaceTrees = ref<Map<string, FileTreeNode>>(new Map())
  const expandedKeys = ref<Set<string>>(new Set())
  const selectedKeys = ref<Set<string>>(new Set())
  const loadingPaths = ref<Set<string>>(new Set())
  const gitStatus = ref<Map<string, GitStatus>>(new Map())
  const filter = ref<FileTreeFilter>({
    showHidden: false,
    showGitIgnored: true,
    fileExtensions: [],
    searchQuery: ''
  })
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  // 계산된 속성
  const getWorkspaceTree = computed(() => (workspaceId: string) => {
    return workspaceTrees.value.get(workspaceId)
  })

  const getGitStatus = computed(() => (workspaceId: string) => {
    return gitStatus.value.get(workspaceId)
  })

  const filteredTree = computed(() => (workspaceId: string) => {
    const tree = workspaceTrees.value.get(workspaceId)
    if (!tree) return null
    
    return filterTreeNode(tree, filter.value)
  })

  // 파일 트리 필터링
  const filterTreeNode = (node: FileTreeNode, filterOptions: FileTreeFilter): FileTreeNode | null => {
    const { showHidden, showGitIgnored, fileExtensions, searchQuery } = filterOptions

    // 숨김 파일 필터
    if (!showHidden && node.name.startsWith('.')) {
      return null
    }

    // Git ignore 필터
    if (!showGitIgnored && node.isGitIgnored) {
      return null
    }

    // 파일 확장자 필터
    if (fileExtensions.length > 0 && !node.isDirectory) {
      const ext = getFileExtension(node.name)
      if (ext && !fileExtensions.includes(ext)) {
        return null
      }
    }

    // 검색 쿼리 필터
    if (searchQuery && !node.name.toLowerCase().includes(searchQuery.toLowerCase())) {
      // 디렉토리의 경우 자식 중에 매칭되는 것이 있는지 확인
      if (node.isDirectory && node.children) {
        const hasMatchingChildren = node.children.some(child => 
          filterTreeNode(child, filterOptions) !== null
        )
        if (!hasMatchingChildren) return null
      } else {
        return null
      }
    }

    // 필터링된 자식들 처리
    let filteredChildren: FileTreeNode[] | undefined
    if (node.children) {
      filteredChildren = node.children
        .map(child => filterTreeNode(child, filterOptions))
        .filter((child): child is FileTreeNode => child !== null)
    }

    return {
      ...node,
      children: filteredChildren
    }
  }

  const getFileExtension = (filename: string): string | null => {
    const lastDot = filename.lastIndexOf('.')
    if (lastDot === -1 || lastDot === 0) return null
    return filename.substring(lastDot + 1).toLowerCase()
  }

  // 워크스페이스 파일 트리 로드
  const loadWorkspaceTree = async (workspaceId: string, path?: string): Promise<void> => {
    isLoading.value = true
    error.value = null
    
    const loadPath = path || '/'
    loadingPaths.value.add(loadPath)

    try {
      // TODO: API 호출로 파일 트리 가져오기
      // const response = await fileApi.getTree(workspaceId, path)
      
      // 더미 데이터 생성
      const dummyTree = generateDummyFileTree(workspaceId, loadPath)
      workspaceTrees.value.set(workspaceId, dummyTree)

      // 더미 Git 상태 생성
      gitStatus.value.set(workspaceId, {
        branch: 'main',
        hasChanges: true,
        changedFiles: ['/src/components/FileTree/FileTreeView.vue', '/src/stores/fileTree.ts'],
        untrackedFiles: ['/temp/cache.tmp', '/logs/debug.log'],
        stagedFiles: ['/src/stores/fileTree.ts']
      })

    } catch (err) {
      error.value = err instanceof Error ? err.message : '파일 트리를 불러오는데 실패했습니다'
    } finally {
      loadingPaths.value.delete(loadPath)
      isLoading.value = false
    }
  }

  // 더미 파일 트리 생성
  const generateDummyFileTree = (workspaceId: string, basePath: string): FileTreeNode => {
    const createNode = (name: string, path: string, isDirectory: boolean, level: number, options: Partial<FileTreeNode> = {}): FileTreeNode => ({
      key: path,
      path,
      name,
      isDirectory,
      isGitIgnored: false,
      isModified: false,
      level,
      parentPath: path.substring(0, path.lastIndexOf('/')),
      ...options
    })

    // 루트 노드
    const root = createNode('/', '/', true, 0, { isExpanded: true })

    // 더미 파일 구조
    root.children = [
      createNode('.git', '/.git', true, 1, { isGitIgnored: true, isExpanded: false }),
      createNode('.gitignore', '/.gitignore', false, 1, { size: 123, gitStatus: 'modified' }),
      createNode('README.md', '/README.md', false, 1, { size: 2048 }),
      createNode('package.json', '/package.json', false, 1, { size: 1024 }),
      createNode('src', '/src', true, 1, { 
        isExpanded: true,
        children: [
          createNode('components', '/src/components', true, 2, {
            isExpanded: true,
            children: [
              createNode('FileTree', '/src/components/FileTree', true, 3, {
                isExpanded: false,
                children: [
                  createNode('FileTreeView.vue', '/src/components/FileTree/FileTreeView.vue', false, 4, { 
                    size: 4096, 
                    gitStatus: 'modified',
                    isModified: true 
                  }),
                  createNode('FileTreeNode.vue', '/src/components/FileTree/FileTreeNode.vue', false, 4, { size: 2048 })
                ]
              }),
              createNode('Workspace', '/src/components/Workspace', true, 3, {
                isExpanded: false,
                children: [
                  createNode('WorkspaceCard.vue', '/src/components/Workspace/WorkspaceCard.vue', false, 4, { size: 3072 }),
                  createNode('WorkspaceList.vue', '/src/components/Workspace/WorkspaceList.vue', false, 4, { size: 5120 })
                ]
              })
            ]
          }),
          createNode('stores', '/src/stores', true, 2, {
            isExpanded: true,
            children: [
              createNode('fileTree.ts', '/src/stores/fileTree.ts', false, 3, { 
                size: 6144, 
                gitStatus: 'added',
                isModified: true 
              }),
              createNode('workspace.ts', '/src/stores/workspace.ts', false, 3, { size: 8192 }),
              createNode('docker.ts', '/src/stores/docker.ts', false, 3, { size: 4096 })
            ]
          }),
          createNode('views', '/src/views', true, 2, {
            isExpanded: false,
            children: [
              createNode('WorkspaceView.vue', '/src/views/WorkspaceView.vue', false, 3, { size: 2048 }),
              createNode('TerminalView.vue', '/src/views/TerminalView.vue', false, 3, { size: 3072 })
            ]
          })
        ]
      }),
      createNode('public', '/public', true, 1, {
        isExpanded: false,
        children: [
          createNode('favicon.ico', '/public/favicon.ico', false, 2, { size: 4096 }),
          createNode('index.html', '/public/index.html', false, 2, { size: 1024 })
        ]
      }),
      createNode('node_modules', '/node_modules', true, 1, { 
        isGitIgnored: true, 
        isExpanded: false 
      }),
      createNode('temp', '/temp', true, 1, {
        isExpanded: false,
        children: [
          createNode('cache.tmp', '/temp/cache.tmp', false, 2, { 
            size: 512, 
            gitStatus: 'untracked',
            isGitIgnored: true 
          })
        ]
      })
    ]

    return root
  }

  // 디렉토리 확장/축소
  const toggleExpanded = async (path: string, workspaceId: string): Promise<void> => {
    const tree = workspaceTrees.value.get(workspaceId)
    if (!tree) return

    const node = findNodeByPath(tree, path)
    if (!node || !node.isDirectory) return

    if (expandedKeys.value.has(path)) {
      expandedKeys.value.delete(path)
      node.isExpanded = false
    } else {
      expandedKeys.value.add(path)
      node.isExpanded = true

      // 자식이 로드되지 않았다면 로드
      if (!node.children || node.children.length === 0) {
        await loadDirectoryChildren(workspaceId, path)
      }
    }
  }

  // 디렉토리 자식 로드
  const loadDirectoryChildren = async (workspaceId: string, path: string): Promise<void> => {
    const tree = workspaceTrees.value.get(workspaceId)
    if (!tree) return

    const node = findNodeByPath(tree, path)
    if (!node || !node.isDirectory) return

    node.isLoading = true
    loadingPaths.value.add(path)

    try {
      // TODO: API 호출로 디렉토리 내용 가져오기
      // const response = await fileApi.getDirectoryContents(workspaceId, path)
      
      // 임시로 더미 자식 생성
      await new Promise(resolve => setTimeout(resolve, 500)) // 로딩 시뮬레이션
      
      // 자식이 이미 있는 경우 스킵
      if (node.children && node.children.length > 0) {
        return
      }

      // 더미 자식 생성 (실제로는 API 응답에서 가져옴)
      node.children = []

    } catch (err) {
      error.value = err instanceof Error ? err.message : '디렉토리 내용을 불러오는데 실패했습니다'
    } finally {
      node.isLoading = false
      loadingPaths.value.delete(path)
    }
  }

  // 경로로 노드 찾기
  const findNodeByPath = (tree: FileTreeNode, path: string): FileTreeNode | null => {
    if (tree.path === path) return tree
    
    if (tree.children) {
      for (const child of tree.children) {
        const found = findNodeByPath(child, path)
        if (found) return found
      }
    }
    
    return null
  }

  // 파일/폴더 선택
  const selectNode = (path: string): void => {
    selectedKeys.value.clear()
    selectedKeys.value.add(path)
  }

  const toggleSelection = (path: string): void => {
    if (selectedKeys.value.has(path)) {
      selectedKeys.value.delete(path)
    } else {
      selectedKeys.value.add(path)
    }
  }

  // 필터 업데이트
  const updateFilter = (updates: Partial<FileTreeFilter>): void => {
    filter.value = { ...filter.value, ...updates }
  }

  const resetFilter = (): void => {
    filter.value = {
      showHidden: false,
      showGitIgnored: true,
      fileExtensions: [],
      searchQuery: ''
    }
  }

  // Git 상태 새로고침
  const refreshGitStatus = async (workspaceId: string): Promise<void> => {
    try {
      // TODO: API 호출로 Git 상태 가져오기
      // const response = await gitApi.getStatus(workspaceId)
      
      // 더미 상태 업데이트
      const currentStatus = gitStatus.value.get(workspaceId)
      if (currentStatus) {
        gitStatus.value.set(workspaceId, {
          ...currentStatus,
          hasChanges: Math.random() > 0.5, // 랜덤으로 변경
        })
      }
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Git 상태를 새로고침하는데 실패했습니다'
    }
  }

  // 파일 생성/삭제/이름변경 (더미 구현)
  const createFile = async (workspaceId: string, parentPath: string, fileName: string, isDirectory: boolean): Promise<void> => {
    const tree = workspaceTrees.value.get(workspaceId)
    if (!tree) return

    const parentNode = findNodeByPath(tree, parentPath)
    if (!parentNode || !parentNode.isDirectory) return

    const newPath = `${parentPath}/${fileName}`.replace('//', '/')
    const newNode: FileTreeNode = {
      key: newPath,
      path: newPath,
      name: fileName,
      isDirectory,
      isGitIgnored: false,
      isModified: false,
      gitStatus: 'untracked',
      level: parentNode.level + 1,
      parentPath,
      size: isDirectory ? undefined : 0,
      children: isDirectory ? [] : undefined
    }

    if (!parentNode.children) {
      parentNode.children = []
    }
    
    parentNode.children.push(newNode)
    parentNode.children.sort((a, b) => {
      // 디렉토리가 파일보다 먼저, 그 다음 알파벳 순
      if (a.isDirectory !== b.isDirectory) {
        return a.isDirectory ? -1 : 1
      }
      return a.name.localeCompare(b.name)
    })
  }

  const deleteFile = async (workspaceId: string, path: string): Promise<void> => {
    const tree = workspaceTrees.value.get(workspaceId)
    if (!tree) return

    const node = findNodeByPath(tree, path)
    if (!node || !node.parentPath) return

    const parentNode = findNodeByPath(tree, node.parentPath)
    if (!parentNode || !parentNode.children) return

    const index = parentNode.children.findIndex(child => child.path === path)
    if (index !== -1) {
      parentNode.children.splice(index, 1)
    }

    // 선택된 키에서도 제거
    selectedKeys.value.delete(path)
    expandedKeys.value.delete(path)
  }

  const renameFile = async (workspaceId: string, oldPath: string, newName: string): Promise<void> => {
    const tree = workspaceTrees.value.get(workspaceId)
    if (!tree) return

    const node = findNodeByPath(tree, oldPath)
    if (!node || !node.parentPath) return

    const newPath = `${node.parentPath}/${newName}`.replace('//', '/')
    
    // 노드 정보 업데이트
    node.name = newName
    node.path = newPath
    node.key = newPath
    node.gitStatus = 'modified'
    node.isModified = true

    // 키 업데이트
    if (selectedKeys.value.has(oldPath)) {
      selectedKeys.value.delete(oldPath)
      selectedKeys.value.add(newPath)
    }
    if (expandedKeys.value.has(oldPath)) {
      expandedKeys.value.delete(oldPath)
      expandedKeys.value.add(newPath)
    }
  }

  // 정리
  const cleanup = (): void => {
    workspaceTrees.value.clear()
    expandedKeys.value.clear()
    selectedKeys.value.clear()
    loadingPaths.value.clear()
    gitStatus.value.clear()
    resetFilter()
    error.value = null
  }

  return {
    // 상태
    workspaceTrees: computed(() => workspaceTrees.value),
    expandedKeys: computed(() => expandedKeys.value),
    selectedKeys: computed(() => selectedKeys.value),
    loadingPaths: computed(() => loadingPaths.value),
    gitStatus: computed(() => gitStatus.value),
    filter: computed(() => filter.value),
    isLoading: computed(() => isLoading.value),
    error: computed(() => error.value),

    // 계산된 속성
    getWorkspaceTree,
    getGitStatus,
    filteredTree,

    // 액션
    loadWorkspaceTree,
    loadDirectoryChildren,
    toggleExpanded,
    selectNode,
    toggleSelection,
    updateFilter,
    resetFilter,
    refreshGitStatus,
    createFile,
    deleteFile,
    renameFile,
    cleanup
  }
})