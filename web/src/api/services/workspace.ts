import { apiDelete, apiGet, apiPost, apiPut } from '@/api'
import type {
  CreateWorkspaceRequest,
  PaginatedResponse,
  UpdateWorkspaceRequest,
} from '@/types/api'
import type { Workspace } from '@/stores/workspace'

export const workspaceApi = {
  /**
   * 워크스페이스 목록 조회
   */
  getWorkspaces: async (params?: {
    page?: number
    limit?: number
    status?: string
    search?: string
  }): Promise<PaginatedResponse<Workspace>> => {
    const response = await apiGet<PaginatedResponse<Workspace>>('/workspaces', { params })
    return response.data.data
  },

  /**
   * 워크스페이스 상세 조회
   */
  getWorkspace: async (id: string): Promise<Workspace> => {
    const response = await apiGet<Workspace>(`/workspaces/${id}`)
    return response.data.data
  },

  /**
   * 워크스페이스 생성
   */
  createWorkspace: async (data: CreateWorkspaceRequest): Promise<Workspace> => {
    const response = await apiPost<Workspace>('/workspaces', data)
    return response.data.data
  },

  /**
   * 워크스페이스 업데이트
   */
  updateWorkspace: async (id: string, data: UpdateWorkspaceRequest): Promise<Workspace> => {
    const response = await apiPut<Workspace>(`/workspaces/${id}`, data)
    return response.data.data
  },

  /**
   * 워크스페이스 삭제
   */
  deleteWorkspace: async (id: string): Promise<void> => {
    await apiDelete(`/workspaces/${id}`)
  },

  /**
   * 워크스페이스 시작
   */
  startWorkspace: async (id: string): Promise<Workspace> => {
    const response = await apiPost<Workspace>(`/workspaces/${id}/start`)
    return response.data.data
  },

  /**
   * 워크스페이스 중지
   */
  stopWorkspace: async (id: string): Promise<Workspace> => {
    const response = await apiPost<Workspace>(`/workspaces/${id}/stop`)
    return response.data.data
  },

  /**
   * 워크스페이스 재시작
   */
  restartWorkspace: async (id: string): Promise<Workspace> => {
    const response = await apiPost<Workspace>(`/workspaces/${id}/restart`)
    return response.data.data
  },

  /**
   * 워크스페이스 로그 조회
   */
  getWorkspaceLogs: async (id: string, params?: {
    lines?: number
    since?: string
    follow?: boolean
  }): Promise<string[]> => {
    const response = await apiGet<string[]>(`/workspaces/${id}/logs`, { params })
    return response.data.data
  },

  /**
   * 워크스페이스 파일 목록 조회
   */
  getWorkspaceFiles: async (id: string, path?: string): Promise<Array<{
    name: string
    type: 'file' | 'directory'
    size?: number
    modifiedAt: string
    path: string
  }>> => {
    const response = await apiGet(`/workspaces/${id}/files`, {
      params: { path: path || '/' },
    })
    return response.data.data
  },

  /**
   * 워크스페이스 파일 내용 조회
   */
  getWorkspaceFile: async (id: string, filePath: string): Promise<{
    content: string
    encoding: string
    size: number
    modifiedAt: string
  }> => {
    const response = await apiGet(`/workspaces/${id}/files/content`, {
      params: { path: filePath },
    })
    return response.data.data
  },

  /**
   * 워크스페이스 파일 저장
   */
  saveWorkspaceFile: async (id: string, filePath: string, content: string): Promise<void> => {
    await apiPost(`/workspaces/${id}/files/content`, {
      path: filePath,
      content,
      encoding: 'utf-8',
    })
  },

  /**
   * 워크스페이스 파일/디렉토리 생성
   */
  createWorkspaceFile: async (id: string, path: string, type: 'file' | 'directory'): Promise<void> => {
    await apiPost(`/workspaces/${id}/files/create`, {
      path,
      type,
    })
  },

  /**
   * 워크스페이스 파일/디렉토리 삭제
   */
  deleteWorkspaceFile: async (id: string, path: string): Promise<void> => {
    await apiDelete(`/workspaces/${id}/files`, {
      params: { path },
    })
  },

  /**
   * 워크스페이스 Git 상태 조회
   */
  getWorkspaceGitStatus: async (id: string): Promise<{
    branch: string
    status: string
    staged: string[]
    modified: string[]
    untracked: string[]
    commits: Array<{
      hash: string
      message: string
      author: string
      date: string
    }>
  }> => {
    const response = await apiGet(`/workspaces/${id}/git/status`)
    return response.data.data
  },

  /**
   * 워크스페이스 Git 명령 실행
   */
  executeGitCommand: async (id: string, command: string): Promise<{
    output: string
    exitCode: number
  }> => {
    const response = await apiPost(`/workspaces/${id}/git/execute`, { command })
    return response.data.data
  },
}