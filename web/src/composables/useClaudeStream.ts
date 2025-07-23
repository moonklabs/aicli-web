// Claude 스트림 전용 WebSocket 컴포저블
// Claude CLI 실시간 출력을 처리하는 특화된 WebSocket 연결 관리

import { computed, ref } from 'vue'
import { type UseWebSocketOptions, useWebSocket } from './useWebSocket'
// import { WS_EVENTS } from '@/constants'
import { processTerminalOutput } from '@/utils/terminal-utils'
import { AnsiParser } from '@/utils/ansi-parser'

export interface ClaudeStreamOptions extends UseWebSocketOptions {
  /**
   * ANSI 코드 자동 파싱 여부
   */
  parseAnsi?: boolean

  /**
   * 출력 버퍼링 지연 시간 (ms)
   */
  outputBufferDelay?: number

  /**
   * 최대 출력 라인 수
   */
  maxOutputLines?: number

  /**
   * 실시간 통계 수집 여부
   */
  collectStats?: boolean
}

export interface ClaudeMessage {
  id: string
  type: 'output' | 'error' | 'system' | 'progress' | 'completed' | 'failed'
  content: string
  timestamp: string
  sessionId: string
  userId?: string
  metadata?: Record<string, any>
}

export interface ClaudeStreamStats {
  totalMessages: number
  outputLines: number
  errorLines: number
  bytesReceived: number
  connectionTime: number
  lastActivity: string
}

export interface ParsedClaudeOutput {
  raw: string
  processed: string
  hasAnsi: boolean
  html: string
  plainText: string
}

export function useClaudeStream(sessionId?: string, options: ClaudeStreamOptions = {}) {
  const {
    parseAnsi = true,
    outputBufferDelay = 50,
    maxOutputLines = 1000,
    collectStats = true,
    ...wsOptions
  } = options

  // 기본 WebSocket 설정
  const baseUrl = import.meta.env.VITE_WS_BASE_URL || 'ws://localhost:8080'
  const url = sessionId ? `${baseUrl}/ws/claude-stream/${sessionId}` : undefined

  const ws = useWebSocket(url, {
    ...wsOptions,
    connectionName: sessionId ? `claude-stream-${sessionId}` : 'claude-stream',
    autoConnect: !!sessionId,
    reconnectInterval: 2000,
    maxReconnectAttempts: 10,
  })

  // Claude 스트림 상태
  const claudeStatus = ref<'idle' | 'running' | 'completed' | 'failed' | 'cancelled'>('idle')
  const currentCommand = ref<string>('')
  const outputBuffer = ref<ClaudeMessage[]>([])
  const sessionUsers = ref<any[]>([])
  const stats = ref<ClaudeStreamStats>({
    totalMessages: 0,
    outputLines: 0,
    errorLines: 0,
    bytesReceived: 0,
    connectionTime: 0,
    lastActivity: new Date().toISOString(),
  })

  // 출력 버퍼링용 타이머
  let bufferTimer: NodeJS.Timeout | null = null
  const pendingOutputs: ClaudeMessage[] = []

  // 계산된 속성
  const isClaudeRunning = computed(() => claudeStatus.value === 'running')
  const hasOutput = computed(() => outputBuffer.value.length > 0)
  const latestOutput = computed(() => outputBuffer.value[outputBuffer.value.length - 1])

  const errorMessages = computed(() =>
    outputBuffer.value.filter(msg => msg.type === 'error'),
  )

  const outputMessages = computed(() =>
    outputBuffer.value.filter(msg => msg.type === 'output'),
  )

  // 출력 처리 함수
  const processClaudeOutput = (content: string): ParsedClaudeOutput => {
    // 터미널 출력 전처리
    const processed = processTerminalOutput(content)

    // ANSI 파싱
    let html = processed
    let plainText = processed
    let hasAnsi = false

    if (parseAnsi) {
      const parsed = AnsiParser.parseAnsiEscapes(processed)
      hasAnsi = parsed.hasAnsiCodes
      html = AnsiParser.renderToHtml(parsed)
      plainText = AnsiParser.stripAnsiCodes(processed)
    }

    return {
      raw: content,
      processed,
      hasAnsi,
      html,
      plainText,
    }
  }

  // 버퍼링된 출력 플러시
  const flushOutputBuffer = () => {
    if (pendingOutputs.length === 0) return

    const outputs = [...pendingOutputs]
    pendingOutputs.length = 0

    outputs.forEach(output => {
      outputBuffer.value.push(output)

      // 통계 업데이트
      if (collectStats) {
        stats.value.totalMessages++
        stats.value.bytesReceived += output.content.length
        stats.value.lastActivity = output.timestamp

        if (output.type === 'output') {
          stats.value.outputLines++
        } else if (output.type === 'error') {
          stats.value.errorLines++
        }
      }
    })

    // 최대 라인 수 제한
    if (outputBuffer.value.length > maxOutputLines) {
      const excess = outputBuffer.value.length - maxOutputLines
      outputBuffer.value.splice(0, excess)
    }
  }

  // 출력 버퍼에 추가 (디바운싱)
  const addToOutputBuffer = (message: ClaudeMessage) => {
    pendingOutputs.push(message)

    if (bufferTimer) {
      clearTimeout(bufferTimer)
    }

    bufferTimer = setTimeout(flushOutputBuffer, outputBufferDelay)
  }

  // Claude 메시지 핸들러들
  const onClaudeMessage = (handler: (message: ClaudeMessage) => void) => {
    return ws.on('claude_message', (data: any) => {
      const message: ClaudeMessage = {
        id: data.message_id || `msg_${Date.now()}`,
        type: data.message_type || 'output',
        content: data.content || '',
        timestamp: data.timestamp || new Date().toISOString(),
        sessionId: data.session_id || sessionId || '',
        userId: data.user_id,
        metadata: data.meta,
      }

      handler(message)
    })
  }

  const onClaudeOutput = (handler: (output: ParsedClaudeOutput, message: ClaudeMessage) => void) => {
    return onClaudeMessage((message) => {
      if (message.type === 'output') {
        const parsed = processClaudeOutput(message.content)
        handler(parsed, message)
        addToOutputBuffer(message)
      }
    })
  }

  const onClaudeError = (handler: (error: string, message: ClaudeMessage) => void) => {
    return onClaudeMessage((message) => {
      if (message.type === 'error') {
        handler(message.content, message)
        addToOutputBuffer(message)
      }
    })
  }

  const onClaudeProgress = (handler: (progress: any, message: ClaudeMessage) => void) => {
    return onClaudeMessage((message) => {
      if (message.type === 'progress') {
        try {
          const progressData = typeof message.content === 'string'
            ? JSON.parse(message.content)
            : message.content
          handler(progressData, message)
        } catch (e) {
          console.warn('Failed to parse progress data:', e)
        }
      }
    })
  }

  const onClaudeComplete = (handler: (result: any, message: ClaudeMessage) => void) => {
    return onClaudeMessage((message) => {
      if (message.type === 'completed') {
        claudeStatus.value = 'completed'
        try {
          const result = typeof message.content === 'string'
            ? JSON.parse(message.content)
            : message.content
          handler(result, message)
        } catch (e) {
          handler({ status: 'completed', output: message.content }, message)
        }
      }
    })
  }

  const onClaudeFailed = (handler: (error: any, message: ClaudeMessage) => void) => {
    return onClaudeMessage((message) => {
      if (message.type === 'failed') {
        claudeStatus.value = 'failed'
        try {
          const error = typeof message.content === 'string'
            ? JSON.parse(message.content)
            : message.content
          handler(error, message)
        } catch (e) {
          handler({ error: message.content }, message)
        }
      }
    })
  }

  // 세션 이벤트 핸들러들
  const onSessionEvent = (handler: (event: any) => void) => {
    return ws.on('session_event', (data: any) => {
      const event = data.event || data
      handler(event)

      // 사용자 목록 업데이트
      if (event.type === 'user_joined' || event.type === 'user_left') {
        // 실제 구현에서는 서버에서 사용자 목록을 받아와야 함
        // 여기서는 시뮬레이션
      }
    })
  }

  const onUserJoined = (handler: (user: any) => void) => {
    return onSessionEvent((event) => {
      if (event.type === 'user_joined') {
        handler({
          userId: event.user_id,
          userName: event.user_name,
          timestamp: event.timestamp,
        })
      }
    })
  }

  const onUserLeft = (handler: (user: any) => void) => {
    return onSessionEvent((event) => {
      if (event.type === 'user_left') {
        handler({
          userId: event.user_id,
          userName: event.user_name,
          timestamp: event.timestamp,
        })
      }
    })
  }

  // Claude 명령 실행
  const executeClaudeCommand = (command: string, workingDir?: string, env?: Record<string, string>): boolean => {
    if (!ws.isConnected.value) {
      console.warn('WebSocket not connected')
      return false
    }

    claudeStatus.value = 'running'
    currentCommand.value = command

    return ws.send({
      type: 'claude_execute',
      payload: {
        command,
        working_dir: workingDir,
        environment: env,
        session_id: sessionId,
      },
    })
  }

  // Claude 프로세스 중단
  const stopClaudeExecution = (): boolean => {
    if (!ws.isConnected.value) {
      return false
    }

    claudeStatus.value = 'cancelled'

    return ws.send({
      type: 'claude_stop',
      payload: {
        session_id: sessionId,
      },
    })
  }

  // 사용자 입력 전송 (실행 중인 프로세스에)
  const sendUserInput = (input: string): boolean => {
    if (!ws.isConnected.value || !isClaudeRunning.value) {
      return false
    }

    return ws.send({
      type: 'user_input',
      payload: {
        input,
        session_id: sessionId,
      },
    })
  }

  // 출력 버퍼 클리어
  const clearOutput = () => {
    outputBuffer.value = []
    pendingOutputs.length = 0

    if (bufferTimer) {
      clearTimeout(bufferTimer)
      bufferTimer = null
    }

    // 통계 초기화
    if (collectStats) {
      stats.value = {
        totalMessages: 0,
        outputLines: 0,
        errorLines: 0,
        bytesReceived: 0,
        connectionTime: Date.now(),
        lastActivity: new Date().toISOString(),
      }
    }
  }

  // 출력을 텍스트로 내보내기
  const exportOutput = (format: 'text' | 'html' | 'json' = 'text'): string => {
    switch (format) {
      case 'html':
        return outputBuffer.value
          .map(msg => {
            const parsed = processClaudeOutput(msg.content)
            return `<div class="claude-output ${msg.type}" data-timestamp="${msg.timestamp}">${parsed.html}</div>`
          })
          .join('\n')

      case 'json':
        return JSON.stringify(outputBuffer.value, null, 2)

      case 'text':
      default:
        return outputBuffer.value
          .map(msg => {
            const timestamp = new Date(msg.timestamp).toLocaleTimeString()
            const content = AnsiParser.stripAnsiCodes(msg.content)
            return `[${timestamp}] ${msg.type.toUpperCase()}: ${content}`
          })
          .join('\n')
    }
  }

  return {
    // 기본 WebSocket 기능
    ...ws,

    // Claude 스트림 상태
    claudeStatus,
    currentCommand,
    outputBuffer,
    sessionUsers,
    stats,

    // 계산된 속성
    isClaudeRunning,
    hasOutput,
    latestOutput,
    errorMessages,
    outputMessages,

    // 이벤트 핸들러
    onClaudeMessage,
    onClaudeOutput,
    onClaudeError,
    onClaudeProgress,
    onClaudeComplete,
    onClaudeFailed,
    onSessionEvent,
    onUserJoined,
    onUserLeft,

    // 액션 메서드
    executeClaudeCommand,
    stopClaudeExecution,
    sendUserInput,
    clearOutput,
    exportOutput,
    processClaudeOutput,
  }
}

export default useClaudeStream