import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

export interface Workspace {
  id: string
  name: string
  path: string
  description?: string
  status: 'idle' | 'loading' | 'active' | 'error' | 'creating' | 'deleting'
  createdAt: string
  updatedAt: string
  lastModified: Date
  lastAccessed?: string
  containerId?: string
  
  // Git 정보
  git?: {
    branch: string
    hasChanges: boolean
    remoteUrl?: string
  }
  
  // Claude 세션 정보
  claudeSession?: {
    id: string
    status: 'active' | 'idle' | 'error'
  }
  
  // 통계 정보
  stats?: {
    fileCount: number
    lineCount: number
  }
  
  // 현재 진행 중인 태스크
  currentTask?: {
    id: string
    description: string
    progress: number
    status: 'running' | 'paused' | 'completed' | 'error'
  }
  
  // Docker 컨테이너 상태
  containerStatus?: {
    type: 'success' | 'warning' | 'error' | 'info'
    text: string
    containerId?: string
  }
  
  // 파일 트리
  fileTree?: FileTreeNode
}

export interface FileTreeNode {
  path: string
  name: string
  isDirectory: boolean
  isGitIgnored: boolean
  isModified: boolean
  gitStatus?: 'modified' | 'added' | 'deleted' | 'untracked'
  size?: number
  children?: FileTreeNode[]
}

export interface WorkspaceFilter {
  key: string
  label: string
  value: string
}

export interface WorkspaceConfig {
  baseImage?: string
  workingDir?: string
  environment?: Record<string, string>
  ports?: number[]
  volumes?: string[]
}

export const useWorkspaceStore = defineStore('workspace', () => {
  // 상태
  const workspaces = ref<Workspace[]>([])
  const activeWorkspace = ref<Workspace | null>(null)
  const isLoading = ref(false)
  const error = ref<string | null>(null)
  
  // 검색 및 필터링
  const searchQuery = ref('')
  const sortBy = ref('name')
  const activeFilters = ref<WorkspaceFilter[]>([])
  
  // 페이지네이션
  const currentPage = ref(1)
  const pageSize = ref(12)

  // 계산된 속성
  const activeWorkspaces = computed(() =>
    workspaces.value.filter(w => w.status === 'active'),
  )

  const inactiveWorkspaces = computed(() =>
    workspaces.value.filter(w => w.status === 'idle'),
  )

  const workspaceById = computed(() => (id: string) =>
    workspaces.value.find(w => w.id === id),
  )

  const totalWorkspaces = computed(() => workspaces.value.length)

  // 필터링된 워크스페이스
  const filteredWorkspaces = computed(() => {
    let filtered = workspaces.value

    // 검색 필터링
    if (searchQuery.value) {
      const query = searchQuery.value.toLowerCase()
      filtered = filtered.filter(workspace => 
        workspace.name.toLowerCase().includes(query) ||
        workspace.path.toLowerCase().includes(query) ||
        workspace.description?.toLowerCase().includes(query)
      )
    }

    // 액티브 필터 적용
    activeFilters.value.forEach(filter => {
      switch (filter.key) {
        case 'status':
          filtered = filtered.filter(w => w.status === filter.value)
          break
        case 'hasGit':
          filtered = filtered.filter(w => !!w.git === (filter.value === 'true'))
          break
        case 'hasClaudeSession':
          filtered = filtered.filter(w => !!w.claudeSession === (filter.value === 'true'))
          break
        case 'containerStatus':
          filtered = filtered.filter(w => w.containerStatus?.type === filter.value)
          break
        case 'hasCurrentTask':
          filtered = filtered.filter(w => !!w.currentTask === (filter.value === 'true'))
          break
      }
    })

    // 정렬
    filtered.sort((a, b) => {
      switch (sortBy.value) {
        case 'name':
          return a.name.localeCompare(b.name)
        case 'lastModified':
          return new Date(b.lastModified).getTime() - new Date(a.lastModified).getTime()
        case 'path':
          return a.path.localeCompare(b.path)
        case 'status':
          return a.status.localeCompare(b.status)
        case 'fileCount':
          return (b.stats?.fileCount || 0) - (a.stats?.fileCount || 0)
        default:
          return 0
      }
    })

    return filtered
  })

  // 페이지네이션된 워크스페이스
  const paginatedWorkspaces = computed(() => {
    const start = (currentPage.value - 1) * pageSize.value
    const end = start + pageSize.value
    return filteredWorkspaces.value.slice(start, end)
  })

  const totalPages = computed(() => {
    return Math.ceil(filteredWorkspaces.value.length / pageSize.value)
  })

  // 현재 활성 워크스페이스 (activeWorkspace 대신)
  const currentActiveWorkspace = computed(() => {
    return workspaces.value.find(w => w.status === 'active') || null
  })

  // 액션
  const setWorkspaces = (workspaceList: Workspace[]) => {
    workspaces.value = workspaceList
  }

  const addWorkspace = (workspace: Workspace) => {
    const existingIndex = workspaces.value.findIndex(w => w.id === workspace.id)
    if (existingIndex !== -1) {
      workspaces.value[existingIndex] = workspace
    } else {
      workspaces.value.push(workspace)
    }
  }

  const updateWorkspace = (id: string, updates: Partial<Workspace>) => {
    const index = workspaces.value.findIndex(w => w.id === id)
    if (index !== -1) {
      workspaces.value[index] = { ...workspaces.value[index], ...updates }

      // 활성 워크스페이스가 업데이트된 경우 동기화
      if (activeWorkspace.value?.id === id) {
        activeWorkspace.value = workspaces.value[index]
      }
    }
  }

  const removeWorkspace = (id: string) => {
    workspaces.value = workspaces.value.filter(w => w.id !== id)

    // 활성 워크스페이스가 삭제된 경우 클리어
    if (activeWorkspace.value?.id === id) {
      activeWorkspace.value = null
    }
  }

  const setActiveWorkspace = (workspace: Workspace | null) => {
    activeWorkspace.value = workspace
  }

  const setLoading = (loading: boolean) => {
    isLoading.value = loading
  }

  const setError = (errorMessage: string | null) => {
    error.value = errorMessage
  }

  // 워크스페이스 목록 불러오기 (더미 데이터로 업데이트)
  const fetchWorkspaces = async (): Promise<void> => {
    try {
      setLoading(true)
      setError(null)

      // TODO: API 호출로 워크스페이스 목록 가져오기
      // const response = await workspaceApi.list()
      // setWorkspaces(response.data)
      
      // 임시 더미 데이터 (태스크 요구사항에 맞게)
      const dummyWorkspaces: Workspace[] = [
        {
          id: '1',
          name: 'aicli-web',
          path: '/workspace/aicli-web',
          description: 'AI CLI Web Platform',
          status: 'active',
          createdAt: '2025-07-20T10:00:00Z',
          updatedAt: '2025-07-23T12:30:00Z',
          lastModified: new Date('2025-07-23T12:30:00'),
          lastAccessed: '2025-07-23T12:30:00Z',
          git: {
            branch: 'main',
            hasChanges: true,
            remoteUrl: 'https://github.com/user/aicli-web.git'
          },
          claudeSession: {
            id: 'session-1',
            status: 'active'
          },
          stats: {
            fileCount: 156,
            lineCount: 12340
          },
          currentTask: {
            id: 'T03_S01',
            description: '워크스페이스 관리 UI 구현',
            progress: 30,
            status: 'running'
          },
          containerStatus: {
            type: 'success',
            text: 'Running',
            containerId: 'container-123'
          }
        },
        {
          id: '2',
          name: 'sample-project',
          path: '/workspace/sample-project',
          description: 'Sample Vue 3 Project',
          status: 'idle',
          createdAt: '2025-07-21T14:00:00Z',
          updatedAt: '2025-07-22T15:20:00Z',
          lastModified: new Date('2025-07-22T15:20:00'),
          lastAccessed: '2025-07-22T15:20:00Z',
          git: {
            branch: 'develop',
            hasChanges: false,
            remoteUrl: 'https://github.com/user/sample-project.git'
          },
          stats: {
            fileCount: 45,
            lineCount: 2800
          },
          containerStatus: {
            type: 'warning',
            text: 'Stopped'
          }
        },
        {
          id: '3',
          name: 'test-workspace',
          path: '/workspace/test-workspace',
          description: 'Testing Environment',
          status: 'error',
          createdAt: '2025-07-19T09:00:00Z',
          updatedAt: '2025-07-21T09:15:00Z',
          lastModified: new Date('2025-07-21T09:15:00'),
          lastAccessed: '2025-07-21T09:15:00Z',
          stats: {
            fileCount: 23,
            lineCount: 890
          },
          containerStatus: {
            type: 'error',
            text: 'Failed'
          }
        }
      ]
      
      setWorkspaces(dummyWorkspaces)
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : '워크스페이스를 불러오는데 실패했습니다'
      setError(errorMessage)
    } finally {
      setLoading(false)
    }
  }

  // 워크스페이스 생성
  const createWorkspace = async (workspaceData: {
    name: string
    path: string
    description?: string
  }): Promise<Workspace | null> => {
    try {
      setLoading(true)
      setError(null)

      // TODO: API 호출로 워크스페이스 생성
      const newWorkspace: Workspace = {
        ...workspaceData,
        id: `ws_${Date.now()}`, // 임시 ID 생성
        status: 'creating',
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
        lastModified: new Date(),
        stats: {
          fileCount: 0,
          lineCount: 0
        },
        containerStatus: {
          type: 'info',
          text: 'Not Started'
        }
      }

      addWorkspace(newWorkspace)
      return newWorkspace
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : '워크스페이스 생성에 실패했습니다'
      setError(errorMessage)
      return null
    } finally {
      setLoading(false)
    }
  }

  // 워크스페이스 시작
  const startWorkspace = async (id: string): Promise<boolean> => {
    try {
      setLoading(true)
      updateWorkspace(id, { status: 'active' })

      // TODO: API 호출로 워크스페이스 시작
      console.log(`Starting workspace ${id}`)
      return true
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to start workspace'
      setError(errorMessage)
      updateWorkspace(id, { status: 'error' })
      return false
    } finally {
      setLoading(false)
    }
  }

  // 워크스페이스 중지
  const stopWorkspace = async (id: string): Promise<boolean> => {
    try {
      setLoading(true)
      updateWorkspace(id, { status: 'inactive' })

      // TODO: API 호출로 워크스페이스 중지
      console.log(`Stopping workspace ${id}`)
      return true
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to stop workspace'
      setError(errorMessage)
      return false
    } finally {
      setLoading(false)
    }
  }

  // 워크스페이스 삭제
  const deleteWorkspace = async (id: string): Promise<boolean> => {
    try {
      setLoading(true)
      updateWorkspace(id, { status: 'deleting' })

      // TODO: API 호출로 워크스페이스 삭제
      console.log(`Deleting workspace ${id}`)
      removeWorkspace(id)
      return true
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to delete workspace'
      setError(errorMessage)
      return false
    } finally {
      setLoading(false)
    }
  }

  // 필터링 및 검색 액션
  const addFilter = (filter: WorkspaceFilter): void => {
    const existingIndex = activeFilters.value.findIndex(f => f.key === filter.key)
    if (existingIndex !== -1) {
      activeFilters.value[existingIndex] = filter
    } else {
      activeFilters.value.push(filter)
    }
  }

  const removeFilter = (key: string): void => {
    const index = activeFilters.value.findIndex(f => f.key === key)
    if (index !== -1) {
      activeFilters.value.splice(index, 1)
    }
  }

  const clearFilters = (): void => {
    activeFilters.value = []
    searchQuery.value = ''
  }

  const setSortBy = (sort: string): void => {
    sortBy.value = sort
  }

  // 워크스페이스 선택/활성화
  const selectWorkspace = (workspaceId: string): void => {
    const workspace = workspaces.value.find(w => w.id === workspaceId)
    if (workspace) {
      setActiveWorkspace(workspace)
    }
  }

  const activateWorkspace = async (workspaceId: string): Promise<void> => {
    const workspace = workspaces.value.find(w => w.id === workspaceId)
    if (!workspace) return

    workspace.status = 'loading'
    
    try {
      // TODO: API 호출 구현
      // await workspaceApi.activate(workspaceId)
      
      // 임시로 상태 변경
      workspace.status = 'active'
      setActiveWorkspace(workspace)
      
      // 다른 워크스페이스는 idle로 변경
      workspaces.value.forEach(w => {
        if (w.id !== workspaceId && w.status === 'active') {
          w.status = 'idle'
        }
      })
    } catch (err) {
      workspace.status = 'error'
      throw err
    }
  }

  // 파일 관련 액션
  const openFile = async (filePath: string): Promise<void> => {
    // TODO: 파일 열기 구현
    console.log('Opening file:', filePath)
  }

  // 워크스페이스 상태 저장
  const saveWorkspaceState = async (): Promise<void> => {
    if (!activeWorkspace.value) return
    
    try {
      // 현재 워크스페이스 상태를 로컬 스토리지에 저장
      const stateToSave = {
        workspaceId: activeWorkspace.value.id,
        lastAccessed: new Date().toISOString(),
        uiState: {
          // UI 관련 상태 저장 (예: 열린 파일, 스크롤 위치 등)
          openFiles: [], // TODO: 실제 열린 파일 목록
          selectedFiles: [], // TODO: 선택된 파일 목록
          expandedFolders: [], // TODO: 확장된 폴더 목록
        },
        claudeSession: activeWorkspace.value.claudeSession,
        lastTask: activeWorkspace.value.currentTask
      }
      
      localStorage.setItem(`workspace_state_${activeWorkspace.value.id}`, JSON.stringify(stateToSave))
      
      // 워크스페이스의 lastAccessed 시간 업데이트
      updateWorkspace(activeWorkspace.value.id, {
        lastAccessed: new Date().toISOString()
      })
      
      console.log(`Workspace state saved for: ${activeWorkspace.value.name}`)
    } catch (error) {
      console.error('Failed to save workspace state:', error)
      throw new Error('워크스페이스 상태 저장에 실패했습니다')
    }
  }

  // 워크스페이스 검증
  const validateWorkspace = async (workspaceId: string): Promise<boolean> => {
    try {
      const workspace = workspaces.value.find(w => w.id === workspaceId)
      if (!workspace) {
        throw new Error('워크스페이스를 찾을 수 없습니다')
      }
      
      // 워크스페이스 상태 검증
      if (workspace.status === 'error') {
        throw new Error('워크스페이스가 오류 상태입니다')
      }
      
      if (workspace.status === 'deleting') {
        throw new Error('삭제 중인 워크스페이스입니다')
      }
      
      // TODO: 실제 파일 시스템 검증
      // - 워크스페이스 디렉토리 존재 확인
      // - 필요한 권한 확인
      // - 디스크 공간 확인
      
      console.log(`Workspace validation passed for: ${workspace.name}`)
      return true
    } catch (error) {
      console.error('Workspace validation failed:', error)
      return false
    }
  }

  // 워크스페이스 상태 복원
  const restoreWorkspaceState = async (workspaceId: string): Promise<void> => {
    try {
      const workspace = workspaces.value.find(w => w.id === workspaceId)
      if (!workspace) {
        throw new Error('워크스페이스를 찾을 수 없습니다')
      }
      
      // 로컬 스토리지에서 저장된 상태 복원
      const savedStateJson = localStorage.getItem(`workspace_state_${workspaceId}`)
      if (savedStateJson) {
        const savedState = JSON.parse(savedStateJson)
        
        // UI 상태 복원
        if (savedState.uiState) {
          // TODO: 실제 UI 상태 복원 구현
          // - 열린 파일 복원
          // - 선택된 파일 복원  
          // - 확장된 폴더 복원
          console.log('Restoring UI state:', savedState.uiState)
        }
        
        // Claude 세션 상태 복원
        if (savedState.claudeSession) {
          updateWorkspace(workspaceId, {
            claudeSession: savedState.claudeSession
          })
        }
        
        // 마지막 태스크 복원
        if (savedState.lastTask) {
          updateWorkspace(workspaceId, {
            currentTask: savedState.lastTask
          })
        }
      }
      
      // lastAccessed 시간 업데이트
      updateWorkspace(workspaceId, {
        lastAccessed: new Date().toISOString()
      })
      
      console.log(`Workspace state restored for: ${workspace.name}`)
    } catch (error) {
      console.error('Failed to restore workspace state:', error)
      // 상태 복원 실패해도 전환은 계속 진행
    }
  }

  return {
    // 상태
    workspaces,
    activeWorkspace,
    isLoading,
    error,

    // 검색 및 필터링 상태
    searchQuery,
    sortBy,
    activeFilters,
    currentPage,
    pageSize,

    // 계산된 속성
    activeWorkspaces,
    inactiveWorkspaces,
    workspaceById,
    totalWorkspaces,
    filteredWorkspaces,
    paginatedWorkspaces,
    totalPages,
    currentActiveWorkspace,

    // 기본 액션
    setWorkspaces,
    addWorkspace,
    updateWorkspace,
    removeWorkspace,
    setActiveWorkspace,
    setLoading,
    setError,
    fetchWorkspaces,
    createWorkspace,
    startWorkspace,
    stopWorkspace,
    deleteWorkspace,

    // 새로운 액션
    addFilter,
    removeFilter,
    clearFilters,
    setSortBy,
    selectWorkspace,
    activateWorkspace,
    openFile,
    saveWorkspaceState,
    validateWorkspace,
    restoreWorkspaceState,
  }
})