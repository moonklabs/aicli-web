import { apiDelete, apiGet, apiPost } from '@/api'
import type {
  CreateTerminalRequest,
  ExecuteCommandRequest,
  PaginatedResponse,
} from '@/types/api'
import type { TerminalSession } from '@/stores/terminal'

export const claudeApi = {
  /**
   * 터미널 세션 목록 조회
   */
  getSessions: async (params?: {
    workspaceId?: string
    status?: string
    page?: number
    limit?: number
  }): Promise<PaginatedResponse<TerminalSession>> => {
    const response = await apiGet<PaginatedResponse<TerminalSession>>('/claude/sessions', { params })
    return response.data.data
  },

  /**
   * 터미널 세션 상세 조회
   */
  getSession: async (id: string): Promise<TerminalSession> => {
    const response = await apiGet<TerminalSession>(`/claude/sessions/${id}`)
    return response.data.data
  },

  /**
   * 새 터미널 세션 생성
   */
  createSession: async (data: CreateTerminalRequest): Promise<TerminalSession> => {
    const response = await apiPost<TerminalSession>('/claude/sessions', data)
    return response.data.data
  },

  /**
   * 터미널 세션 삭제
   */
  deleteSession: async (id: string): Promise<void> => {
    await apiDelete(`/claude/sessions/${id}`)
  },

  /**
   * 명령 실행
   */
  executeCommand: async (sessionId: string, data: ExecuteCommandRequest): Promise<{
    commandId: string
    status: string
    message?: string
  }> => {
    const response = await apiPost(`/claude/sessions/${sessionId}/execute`, data)
    return response.data.data
  },

  /**
   * 세션 로그 조회
   */
  getSessionLogs: async (sessionId: string, params?: {
    from?: string
    to?: string
    limit?: number
    offset?: number
  }): Promise<{
    logs: Array<{
      id: string
      timestamp: string
      type: 'input' | 'output' | 'error' | 'system'
      content: string
      level?: 'info' | 'warn' | 'error' | 'debug'
    }>
    total: number
    hasMore: boolean
  }> => {
    const response = await apiGet(`/claude/sessions/${sessionId}/logs`, { params })
    return response.data.data
  },

  /**
   * 세션 로그 클리어
   */
  clearSessionLogs: async (sessionId: string): Promise<void> => {
    await apiDelete(`/claude/sessions/${sessionId}/logs`)
  },

  /**
   * Claude 작업 목록 조회
   */
  getTasks: async (params?: {
    workspaceId?: string
    status?: string
    type?: string
    page?: number
    limit?: number
  }): Promise<PaginatedResponse<{
    id: string
    type: string
    status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled'
    progress: number
    message?: string
    result?: any
    error?: string
    workspaceId?: string
    sessionId?: string
    createdAt: string
    updatedAt: string
    completedAt?: string
  }>> => {
    const response = await apiGet('/claude/tasks', { params })
    return response.data.data
  },

  /**
   * Claude 작업 상세 조회
   */
  getTask: async (id: string): Promise<{
    id: string
    type: string
    status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled'
    progress: number
    message?: string
    result?: any
    error?: string
    workspaceId?: string
    sessionId?: string
    logs: Array<{
      timestamp: string
      level: string
      message: string
    }>
    createdAt: string
    updatedAt: string
    completedAt?: string
  }> => {
    const response = await apiGet(`/claude/tasks/${id}`)
    return response.data.data
  },

  /**
   * Claude 작업 취소
   */
  cancelTask: async (id: string): Promise<void> => {
    await apiPost(`/claude/tasks/${id}/cancel`)
  },

  /**
   * Claude 설정 조회
   */
  getConfig: async (): Promise<{
    apiKey: string
    model: string
    maxTokens: number
    temperature: number
    timeout: number
    retryAttempts: number
    rateLimiting: {
      requestsPerMinute: number
      tokensPerMinute: number
    }
  }> => {
    const response = await apiGet('/claude/config')
    return response.data.data
  },

  /**
   * Claude 설정 업데이트
   */
  updateConfig: async (config: {
    model?: string
    maxTokens?: number
    temperature?: number
    timeout?: number
    retryAttempts?: number
    rateLimiting?: {
      requestsPerMinute?: number
      tokensPerMinute?: number
    }
  }): Promise<void> => {
    await apiPost('/claude/config', config)
  },

  /**
   * Claude API 상태 확인
   */
  getStatus: async (): Promise<{
    status: 'online' | 'offline' | 'error'
    latency: number
    rateLimits: {
      remaining: number
      reset: string
    }
    modelInfo: {
      name: string
      contextLength: number
      maxOutputTokens: number
    }
    lastChecked: string
  }> => {
    const response = await apiGet('/claude/status')
    return response.data.data
  },

  /**
   * Claude 모델 목록 조회
   */
  getModels: async (): Promise<Array<{
    id: string
    name: string
    description: string
    contextLength: number
    maxOutputTokens: number
    inputPrice: number
    outputPrice: number
    available: boolean
  }>> => {
    const response = await apiGet('/claude/models')
    return response.data.data
  },

  /**
   * 프롬프트 템플릿 목록 조회
   */
  getTemplates: async (): Promise<Array<{
    id: string
    name: string
    description: string
    category: string
    template: string
    variables: Array<{
      name: string
      type: string
      description: string
      required: boolean
      defaultValue?: string
    }>
    createdAt: string
    updatedAt: string
  }>> => {
    const response = await apiGet('/claude/templates')
    return response.data.data
  },

  /**
   * 프롬프트 템플릿 사용
   */
  useTemplate: async (templateId: string, variables: Record<string, string>): Promise<{
    prompt: string
    estimatedTokens: number
  }> => {
    const response = await apiPost(`/claude/templates/${templateId}/use`, { variables })
    return response.data.data
  },
}