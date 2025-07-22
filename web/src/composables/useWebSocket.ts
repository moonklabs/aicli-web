// WebSocket 컴포저블

import { computed, onMounted, onUnmounted, ref } from 'vue'
import { type WebSocketMessage, type WebSocketOptions, webSocketService } from '@/utils/websocket'
import { WS_EVENTS } from '@/constants'

export interface UseWebSocketOptions extends WebSocketOptions {
  autoConnect?: boolean
  connectionName?: string
}

export function useWebSocket(url?: string, options: UseWebSocketOptions = {}) {
  const {
    autoConnect = false,
    connectionName = 'default',
    ...wsOptions
  } = options

  // 상태
  const status = ref<'connecting' | 'connected' | 'disconnected' | 'error'>('disconnected')
  const error = ref<Event | Error | null>(null)
  const lastMessage = ref<WebSocketMessage | null>(null)
  const messageHistory = ref<WebSocketMessage[]>([])

  // WebSocket 매니저 인스턴스
  let wsManager: ReturnType<typeof webSocketService.getConnection> | null = null

  // 계산된 속성
  const isConnected = computed(() => status.value === 'connected')
  const isConnecting = computed(() => status.value === 'connecting')
  const isDisconnected = computed(() => status.value === 'disconnected')
  const hasError = computed(() => status.value === 'error')

  // 이벤트 핸들러 정리 함수들
  const cleanupFunctions: (() => void)[] = []

  // 메서드
  const connect = async (): Promise<void> => {
    if (!url) {
      throw new Error('WebSocket URL is required')
    }

    try {
      wsManager = webSocketService.getConnection(connectionName, url, wsOptions)

      // 상태 변경 핸들러 등록
      const statusCleanup = wsManager.onStatusChange((newStatus) => {
        status.value = newStatus
      })
      cleanupFunctions.push(statusCleanup)

      // 에러 핸들러 등록
      const errorCleanup = wsManager.onError((err) => {
        error.value = err
      })
      cleanupFunctions.push(errorCleanup)

      // 모든 메시지 수신 핸들러
      const messageCleanup = wsManager.on('*', (message: WebSocketMessage) => {
        lastMessage.value = message
        messageHistory.value.push(message)

        // 메시지 히스토리 제한 (최대 100개)
        if (messageHistory.value.length > 100) {
          messageHistory.value = messageHistory.value.slice(-100)
        }
      })
      cleanupFunctions.push(messageCleanup)

      await wsManager.connect()
    } catch (err) {
      error.value = err instanceof Error ? err : new Error(String(err))
      throw err
    }
  }

  const disconnect = (): void => {
    wsManager?.disconnect()
    cleanup()
  }

  const send = (message: Omit<WebSocketMessage, 'timestamp' | 'id'>): boolean => {
    if (!wsManager) {
      console.warn('WebSocket not initialized')
      return false
    }
    return wsManager.send(message as WebSocketMessage)
  }

  const on = (event: string, handler: (data: any) => void): (() => void) => {
    if (!wsManager) {
      console.warn('WebSocket not initialized')
      return () => {}
    }

    const cleanup = wsManager.on(event, handler)
    cleanupFunctions.push(cleanup)
    return cleanup
  }

  const reconnect = async (): Promise<void> => {
    if (!wsManager) {
      throw new Error('WebSocket not initialized')
    }
    await wsManager.reconnect()
  }

  const cleanup = (): void => {
    cleanupFunctions.forEach(fn => fn())
    cleanupFunctions.length = 0
  }

  // 생명주기
  onMounted(() => {
    if (autoConnect && url) {
      connect().catch(console.error)
    }
  })

  onUnmounted(() => {
    disconnect()
  })

  return {
    // 상태
    status,
    error,
    lastMessage,
    messageHistory,

    // 계산된 속성
    isConnected,
    isConnecting,
    isDisconnected,
    hasError,

    // 메서드
    connect,
    disconnect,
    reconnect,
    send,
    on,
  }
}

// 터미널 전용 WebSocket 컴포저블
export function useTerminalWebSocket(sessionId?: string, options: UseWebSocketOptions = {}) {
  const baseUrl = import.meta.env.VITE_WS_BASE_URL || 'ws://localhost:8080'
  const url = sessionId ? `${baseUrl}/ws/terminal/${sessionId}` : undefined

  const ws = useWebSocket(url, {
    ...options,
    connectionName: sessionId ? `terminal-${sessionId}` : 'terminal',
    autoConnect: !!sessionId,
  })

  // 터미널 특화 메서드
  const sendCommand = (command: string): boolean => {
    return ws.send({
      type: WS_EVENTS.TERMINAL_INPUT,
      payload: { command },
    })
  }

  const sendInput = (input: string): boolean => {
    return ws.send({
      type: WS_EVENTS.TERMINAL_INPUT,
      payload: { input },
    })
  }

  // 터미널 이벤트 핸들러
  const onOutput = (handler: (output: string) => void) => {
    return ws.on(WS_EVENTS.TERMINAL_OUTPUT, (data) => {
      handler(data.content || data.output || '')
    })
  }

  const onError = (handler: (error: string) => void) => {
    return ws.on(WS_EVENTS.TERMINAL_ERROR, (data) => {
      handler(data.error || data.message || 'Unknown error')
    })
  }

  return {
    ...ws,
    sendCommand,
    sendInput,
    onOutput,
    onError,
  }
}

// 워크스페이스 전용 WebSocket 컴포저블
export function useWorkspaceWebSocket(workspaceId?: string, options: UseWebSocketOptions = {}) {
  const baseUrl = import.meta.env.VITE_WS_BASE_URL || 'ws://localhost:8080'
  const url = workspaceId ? `${baseUrl}/ws/workspace/${workspaceId}` : undefined

  const ws = useWebSocket(url, {
    ...options,
    connectionName: workspaceId ? `workspace-${workspaceId}` : 'workspace',
    autoConnect: !!workspaceId,
  })

  // 워크스페이스 이벤트 핸들러
  const onStatusChange = (handler: (status: string) => void) => {
    return ws.on(WS_EVENTS.WORKSPACE_STATUS, (data) => {
      handler(data.status)
    })
  }

  const onUpdate = (handler: (data: any) => void) => {
    return ws.on(WS_EVENTS.WORKSPACE_UPDATED, handler)
  }

  return {
    ...ws,
    onStatusChange,
    onUpdate,
  }
}

// Docker 전용 WebSocket 컴포저블
export function useDockerWebSocket(options: UseWebSocketOptions = {}) {
  const baseUrl = import.meta.env.VITE_WS_BASE_URL || 'ws://localhost:8080'
  const url = `${baseUrl}/ws/docker`

  const ws = useWebSocket(url, {
    ...options,
    connectionName: 'docker',
    autoConnect: false, // 필요할 때만 연결
  })

  // Docker 이벤트 핸들러
  const onContainerStatus = (handler: (data: { containerId: string, status: string }) => void) => {
    return ws.on(WS_EVENTS.DOCKER_CONTAINER_STATUS, handler)
  }

  const onStats = (handler: (stats: any) => void) => {
    return ws.on(WS_EVENTS.DOCKER_STATS, handler)
  }

  return {
    ...ws,
    onContainerStatus,
    onStats,
  }
}

// 전역 이벤트 WebSocket 컴포저블
export function useGlobalWebSocket(options: UseWebSocketOptions = {}) {
  const baseUrl = import.meta.env.VITE_WS_BASE_URL || 'ws://localhost:8080'
  const url = `${baseUrl}/ws/events`

  const ws = useWebSocket(url, {
    ...options,
    connectionName: 'global',
    autoConnect: true,
    reconnectInterval: 3000,
    maxReconnectAttempts: 20,
  })

  // 전역 이벤트 핸들러들
  const onTaskUpdate = (handler: (task: any) => void) => {
    return ws.on(WS_EVENTS.TASK_UPDATE, handler)
  }

  const onTaskComplete = (handler: (task: any) => void) => {
    return ws.on(WS_EVENTS.TASK_COMPLETE, handler)
  }

  const onTaskError = (handler: (task: any) => void) => {
    return ws.on(WS_EVENTS.TASK_ERROR, handler)
  }

  return {
    ...ws,
    onTaskUpdate,
    onTaskComplete,
    onTaskError,
  }
}