import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

export interface Workspace {
  id: string
  name: string
  path: string
  description?: string
  status: 'active' | 'inactive' | 'error' | 'creating' | 'deleting'
  createdAt: string
  updatedAt: string
  containerId?: string
  gitRemote?: string
  gitBranch?: string
  lastActivity?: string
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

  // 계산된 속성
  const activeWorkspaces = computed(() =>
    workspaces.value.filter(w => w.status === 'active'),
  )

  const inactiveWorkspaces = computed(() =>
    workspaces.value.filter(w => w.status === 'inactive'),
  )

  const workspaceById = computed(() => (id: string) =>
    workspaces.value.find(w => w.id === id),
  )

  const totalWorkspaces = computed(() => workspaces.value.length)

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

  // 워크스페이스 생성
  const createWorkspace = async (workspaceData: Omit<Workspace, 'id' | 'createdAt' | 'updatedAt' | 'status'>): Promise<Workspace | null> => {
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
      }

      addWorkspace(newWorkspace)
      return newWorkspace
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to create workspace'
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

  return {
    // 상태
    workspaces,
    activeWorkspace,
    isLoading,
    error,

    // 계산된 속성
    activeWorkspaces,
    inactiveWorkspaces,
    workspaceById,
    totalWorkspaces,

    // 액션
    setWorkspaces,
    addWorkspace,
    updateWorkspace,
    removeWorkspace,
    setActiveWorkspace,
    setLoading,
    setError,
    createWorkspace,
    startWorkspace,
    stopWorkspace,
    deleteWorkspace,
  }
})