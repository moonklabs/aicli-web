import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { useClaudeStream } from '@/composables/useClaudeStream'
import { generateSessionId, maskSensitiveCommand } from '@/utils/terminal-utils'

export interface TerminalSession {
  id: string
  workspaceId: string
  title: string
  status: 'connected' | 'disconnected' | 'error' | 'connecting'
  logs: TerminalLog[]
  createdAt: string
  lastActivity: string
  pid?: number
  claudeStream?: any
}

export interface TerminalLog {
  id: string
  timestamp: string
  type: 'input' | 'output' | 'error' | 'system'
  content: string
  level?: 'info' | 'warn' | 'error' | 'debug'
  parsed?: {
    raw: string
    html: string
    plainText: string
    hasAnsi: boolean
  }
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

      const sessionId = generateSessionId()
      const sessionTitle = title || `Terminal ${sessions.value.length + 1}`

      // Claude 스트림 WebSocket 연결 생성
      const claudeStream = useClaudeStream(sessionId, {
        parseAnsi: true,
        outputBufferDelay: 50,
        maxOutputLines: 1000,
        collectStats: true,
        autoConnect: true,
        reconnectInterval: 2000,
        maxReconnectAttempts: 5,
      })

      const newSession: TerminalSession = {
        id: sessionId,
        workspaceId,
        title: sessionTitle,
        status: 'connecting',
        logs: [],
        createdAt: new Date().toISOString(),
        lastActivity: new Date().toISOString(),
        claudeStream,
      }

      // WebSocket 이벤트 핸들러 설정
      setupSessionEventHandlers(newSession)

      addSession(newSession)

      // 연결 시도
      try {
        await claudeStream.connect()
        updateSession(sessionId, { status: 'connected' })

        // 시스템 메시지 추가
        addLog(sessionId, {
          id: `log_${Date.now()}`,
          timestamp: new Date().toISOString(),
          type: 'system',
          content: `터미널 세션이 연결되었습니다. (세션 ID: ${sessionId})`,
          level: 'info',
        })
      } catch (connectError) {
        updateSession(sessionId, { status: 'error' })
        const errorMessage = connectError instanceof Error ? connectError.message : 'Connection failed'

        addLog(sessionId, {
          id: `log_${Date.now()}`,
          timestamp: new Date().toISOString(),
          type: 'error',
          content: `연결 실패: ${errorMessage}`,
          level: 'error',
        })
      }

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
      const session = sessions.value.find(s => s.id === sessionId)
      if (!session || !session.claudeStream) {
        throw new Error('Session not found or not connected')
      }

      // 민감한 정보 마스킹하여 로그에 기록
      const maskedCommand = maskSensitiveCommand(command.command)
      const inputLog: TerminalLog = {
        id: `log_${Date.now()}`,
        timestamp: new Date().toISOString(),
        type: 'input',
        content: maskedCommand,
        level: 'info',
      }

      addLog(sessionId, inputLog)

      // Claude 스트림을 통해 명령 실행
      const success = session.claudeStream.executeClaudeCommand(
        command.command,
        command.workingDir,
        command.environment,
      )

      if (!success) {
        throw new Error('Failed to send command to Claude stream')
      }

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

  // WebSocket 이벤트 핸들러 설정
  const setupSessionEventHandlers = (session: TerminalSession) => {
    if (!session.claudeStream) return

    const { claudeStream } = session

    // Claude 출력 수신
    claudeStream.onClaudeOutput((parsed: any, message: any) => {
      const log: TerminalLog = {
        id: message.id,
        timestamp: message.timestamp,
        type: 'output',
        content: message.content,
        level: 'info',
        parsed: {
          raw: parsed.raw,
          html: parsed.html,
          plainText: parsed.plainText,
          hasAnsi: parsed.hasAnsi,
        },
      }
      addLog(session.id, log)
    })

    // Claude 에러 수신
    claudeStream.onClaudeError((error: any, message: any) => {
      const log: TerminalLog = {
        id: message.id,
        timestamp: message.timestamp,
        type: 'error',
        content: error,
        level: 'error',
      }
      addLog(session.id, log)
    })

    // Claude 실행 완료
    claudeStream.onClaudeComplete((result: any, message: any) => {
      const log: TerminalLog = {
        id: message.id,
        timestamp: message.timestamp,
        type: 'system',
        content: `명령 실행 완료: ${JSON.stringify(result)}`,
        level: 'info',
      }
      addLog(session.id, log)
    })

    // Claude 실행 실패
    claudeStream.onClaudeFailed((error: any, message: any) => {
      const log: TerminalLog = {
        id: message.id,
        timestamp: message.timestamp,
        type: 'error',
        content: `명령 실행 실패: ${JSON.stringify(error)}`,
        level: 'error',
      }
      addLog(session.id, log)
    })

    // 연결 상태 변경
    claudeStream.onStatusChange((status: any) => {
      const terminalStatus = mapWebSocketStatusToTerminalStatus(status)
      updateSession(session.id, { status: terminalStatus })

      const log: TerminalLog = {
        id: `status_${Date.now()}`,
        timestamp: new Date().toISOString(),
        type: 'system',
        content: `연결 상태 변경: ${status}`,
        level: 'info',
      }
      addLog(session.id, log)
    })

    // 에러 처리
    claudeStream.onError((error: any) => {
      const log: TerminalLog = {
        id: `error_${Date.now()}`,
        timestamp: new Date().toISOString(),
        type: 'error',
        content: `WebSocket 오류: ${error.message || error}`,
        level: 'error',
      }
      addLog(session.id, log)
      updateSession(session.id, { status: 'error' })
    })

    // 사용자 참여/퇴장 알림
    claudeStream.onUserJoined((user: any) => {
      const log: TerminalLog = {
        id: `user_join_${Date.now()}`,
        timestamp: new Date().toISOString(),
        type: 'system',
        content: `${user.userName}님이 세션에 참여했습니다.`,
        level: 'info',
      }
      addLog(session.id, log)
    })

    claudeStream.onUserLeft((user: any) => {
      const log: TerminalLog = {
        id: `user_left_${Date.now()}`,
        timestamp: new Date().toISOString(),
        type: 'system',
        content: `${user.userName}님이 세션에서 나갔습니다.`,
        level: 'info',
      }
      addLog(session.id, log)
    })
  }

  // WebSocket 상태를 터미널 상태로 매핑
  const mapWebSocketStatusToTerminalStatus = (wsStatus: string): TerminalSession['status'] => {
    switch (wsStatus) {
      case 'connected':
        return 'connected'
      case 'connecting':
        return 'connecting'
      case 'error':
        return 'error'
      case 'disconnected':
      default:
        return 'disconnected'
    }
  }

  // 세션 연결 해제
  const disconnectSession = async (sessionId: string): Promise<boolean> => {
    try {
      const session = sessions.value.find(s => s.id === sessionId)
      if (session?.claudeStream) {
        session.claudeStream.disconnect()
      }

      updateSession(sessionId, { status: 'disconnected' })

      const log: TerminalLog = {
        id: `disconnect_${Date.now()}`,
        timestamp: new Date().toISOString(),
        type: 'system',
        content: '세션 연결이 해제되었습니다.',
        level: 'info',
      }
      addLog(sessionId, log)

      return true
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to disconnect session'
      setError(errorMessage)
      return false
    }
  }

  // 세션에 사용자 입력 전송 (실행 중인 프로세스에)
  const sendUserInput = (sessionId: string, input: string): boolean => {
    const session = sessions.value.find(s => s.id === sessionId)
    if (!session?.claudeStream) {
      return false
    }

    return session.claudeStream.sendUserInput(input)
  }

  // Claude 실행 중단
  const stopExecution = (sessionId: string): boolean => {
    const session = sessions.value.find(s => s.id === sessionId)
    if (!session?.claudeStream) {
      return false
    }

    const success = session.claudeStream.stopClaudeExecution()

    if (success) {
      const log: TerminalLog = {
        id: `stop_${Date.now()}`,
        timestamp: new Date().toISOString(),
        type: 'system',
        content: '명령 실행이 중단되었습니다.',
        level: 'warn',
      }
      addLog(sessionId, log)
    }

    return success
  }

  // 세션 재연결
  const reconnectSession = async (sessionId: string): Promise<boolean> => {
    try {
      const session = sessions.value.find(s => s.id === sessionId)
      if (!session?.claudeStream) {
        return false
      }

      updateSession(sessionId, { status: 'connecting' })

      await session.claudeStream.reconnect()

      const log: TerminalLog = {
        id: `reconnect_${Date.now()}`,
        timestamp: new Date().toISOString(),
        type: 'system',
        content: '세션이 재연결되었습니다.',
        level: 'info',
      }
      addLog(sessionId, log)

      return true
    } catch (err) {
      updateSession(sessionId, { status: 'error' })

      const log: TerminalLog = {
        id: `reconnect_error_${Date.now()}`,
        timestamp: new Date().toISOString(),
        type: 'error',
        content: `재연결 실패: ${err instanceof Error ? err.message : 'Unknown error'}`,
        level: 'error',
      }
      addLog(sessionId, log)

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

    // 기본 액션
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

    // WebSocket 통합 액션
    createSession,
    executeCommand,
    disconnectSession,
    reconnectSession,
    sendUserInput,
    stopExecution,
    setupSessionEventHandlers,
  }
})