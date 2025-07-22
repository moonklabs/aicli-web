import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

export interface TerminalSession {
  id: string
  workspaceId: string
  title: string
  status: 'connected' | 'disconnected' | 'error' | 'connecting'
  logs: TerminalLog[]
  createdAt: string
  lastActivity: string
  pid?: number
}

export interface TerminalLog {
  id: string
  timestamp: string
  type: 'input' | 'output' | 'error' | 'system'
  content: string
  level?: 'info' | 'warn' | 'error' | 'debug'
}

export interface TerminalCommand {
  command: string
  workingDir?: string
  environment?: Record<string, string>
}

export const useTerminalStore = defineStore('terminal', () => {
  // 상태
  const sessions = ref<TerminalSession[]>([])
  const activeSession = ref<TerminalSession | null>(null)
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  // 계산된 속성
  const activeSessions = computed(() =>
    sessions.value.filter(s => s.status === 'connected'),
  )

  const sessionsByWorkspace = computed(() => (workspaceId: string) =>
    sessions.value.filter(s => s.workspaceId === workspaceId),
  )

  const sessionById = computed(() => (id: string) =>
    sessions.value.find(s => s.id === id),
  )

  const totalSessions = computed(() => sessions.value.length)

  // 액션
  const setSessions = (sessionList: TerminalSession[]) => {
    sessions.value = sessionList
  }

  const addSession = (session: TerminalSession) => {
    const existingIndex = sessions.value.findIndex(s => s.id === session.id)
    if (existingIndex !== -1) {
      sessions.value[existingIndex] = session
    } else {
      sessions.value.push(session)
    }
  }

  const updateSession = (id: string, updates: Partial<TerminalSession>) => {
    const index = sessions.value.findIndex(s => s.id === id)
    if (index !== -1) {
      sessions.value[index] = { ...sessions.value[index], ...updates }

      // 활성 세션이 업데이트된 경우 동기화
      if (activeSession.value?.id === id) {
        activeSession.value = sessions.value[index]
      }
    }
  }

  const removeSession = (id: string) => {
    sessions.value = sessions.value.filter(s => s.id !== id)

    // 활성 세션이 삭제된 경우 클리어
    if (activeSession.value?.id === id) {
      activeSession.value = null
    }
  }

  const setActiveSession = (session: TerminalSession | null) => {
    activeSession.value = session
  }

  const setLoading = (loading: boolean) => {
    isLoading.value = loading
  }

  const setError = (errorMessage: string | null) => {
    error.value = errorMessage
  }

  // 로그 추가
  const addLog = (sessionId: string, log: TerminalLog) => {
    const session = sessions.value.find(s => s.id === sessionId)
    if (session) {
      session.logs.push(log)
      session.lastActivity = new Date().toISOString()

      // 로그가 너무 많이 쌓이는 것을 방지 (최대 1000개)
      if (session.logs.length > 1000) {
        session.logs = session.logs.slice(-900) // 최신 900개만 유지
      }

      // 활성 세션인 경우 업데이트
      if (activeSession.value?.id === sessionId) {
        activeSession.value = { ...session }
      }
    }
  }

  // 여러 로그 추가
  const addLogs = (sessionId: string, logs: TerminalLog[]) => {
    const session = sessions.value.find(s => s.id === sessionId)
    if (session) {
      session.logs.push(...logs)
      session.lastActivity = new Date().toISOString()

      // 로그 수 제한
      if (session.logs.length > 1000) {
        session.logs = session.logs.slice(-900)
      }

      if (activeSession.value?.id === sessionId) {
        activeSession.value = { ...session }
      }
    }
  }

  // 세션 로그 클리어
  const clearLogs = (sessionId: string) => {
    const session = sessions.value.find(s => s.id === sessionId)
    if (session) {
      session.logs = []

      if (activeSession.value?.id === sessionId) {
        activeSession.value = { ...session }
      }
    }
  }

  // 새 터미널 세션 생성
  const createSession = async (workspaceId: string, title?: string): Promise<TerminalSession | null> => {
    try {
      setLoading(true)
      setError(null)

      // TODO: API 호출로 터미널 세션 생성
      const newSession: TerminalSession = {
        id: `term_${Date.now()}`, // 임시 ID 생성
        workspaceId,
        title: title || `Terminal ${sessions.value.length + 1}`,
        status: 'connecting',
        logs: [],
        createdAt: new Date().toISOString(),
        lastActivity: new Date().toISOString(),
      }

      addSession(newSession)
      return newSession
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to create terminal session'
      setError(errorMessage)
      return null
    } finally {
      setLoading(false)
    }
  }

  // 명령 실행
  const executeCommand = async (sessionId: string, command: TerminalCommand): Promise<boolean> => {
    try {
      const inputLog: TerminalLog = {
        id: `log_${Date.now()}`,
        timestamp: new Date().toISOString(),
        type: 'input',
        content: command.command,
        level: 'info',
      }

      addLog(sessionId, inputLog)

      // TODO: WebSocket을 통한 실제 명령 실행
      console.log(`Executing command in session ${sessionId}:`, command.command)

      // 임시 응답 로그
      setTimeout(() => {
        const outputLog: TerminalLog = {
          id: `log_${Date.now()}`,
          timestamp: new Date().toISOString(),
          type: 'output',
          content: `Command executed: ${command.command}`,
          level: 'info',
        }
        addLog(sessionId, outputLog)
      }, 100)

      return true
    } catch (err) {
      const errorLog: TerminalLog = {
        id: `log_${Date.now()}`,
        timestamp: new Date().toISOString(),
        type: 'error',
        content: err instanceof Error ? err.message : 'Command execution failed',
        level: 'error',
      }
      addLog(sessionId, errorLog)
      return false
    }
  }

  // 세션 연결 해제
  const disconnectSession = async (sessionId: string): Promise<boolean> => {
    try {
      updateSession(sessionId, { status: 'disconnected' })

      // TODO: API 호출로 세션 연결 해제
      console.log(`Disconnecting session ${sessionId}`)
      return true
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to disconnect session'
      setError(errorMessage)
      return false
    }
  }

  return {
    // 상태
    sessions,
    activeSession,
    isLoading,
    error,

    // 계산된 속성
    activeSessions,
    sessionsByWorkspace,
    sessionById,
    totalSessions,

    // 액션
    setSessions,
    addSession,
    updateSession,
    removeSession,
    setActiveSession,
    setLoading,
    setError,
    addLog,
    addLogs,
    clearLogs,
    createSession,
    executeCommand,
    disconnectSession,
  }
})