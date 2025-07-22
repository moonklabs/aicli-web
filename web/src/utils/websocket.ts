// WebSocket 클라이언트 매니저

export interface WebSocketOptions {
  reconnectInterval?: number
  maxReconnectAttempts?: number
  pingInterval?: number
  pongTimeout?: number
  debug?: boolean
}

export interface WebSocketMessage {
  type: string
  payload: any
  timestamp?: string
  id?: string
}

export type WebSocketEventHandler = (data: any) => void
export type WebSocketErrorHandler = (error: Event | Error) => void
export type WebSocketStatusHandler = (status: 'connecting' | 'connected' | 'disconnected' | 'error') => void

class WebSocketManager {
  private ws: WebSocket | null = null
  private url: string
  private options: Required<WebSocketOptions>
  private eventHandlers: Map<string, WebSocketEventHandler[]> = new Map()
  private errorHandlers: WebSocketErrorHandler[] = []
  private statusHandlers: WebSocketStatusHandler[] = []

  private reconnectCount = 0
  private reconnectTimer: NodeJS.Timeout | null = null
  private pingTimer: NodeJS.Timeout | null = null
  private pongTimer: NodeJS.Timeout | null = null

  private isManualClose = false
  private status: 'connecting' | 'connected' | 'disconnected' | 'error' = 'disconnected'

  constructor(url: string, options: WebSocketOptions = {}) {
    this.url = url
    this.options = {
      reconnectInterval: options.reconnectInterval || 5000,
      maxReconnectAttempts: options.maxReconnectAttempts || 10,
      pingInterval: options.pingInterval || 30000,
      pongTimeout: options.pongTimeout || 5000,
      debug: options.debug || false,
    }
  }

  /**
   * WebSocket 연결 시작
   */
  connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      try {
        this.isManualClose = false
        this.setStatus('connecting')

        this.ws = new WebSocket(this.url)

        this.ws.onopen = () => {
          this.log('WebSocket connected')
          this.reconnectCount = 0
          this.setStatus('connected')
          this.startPing()
          resolve()
        }

        this.ws.onmessage = (event) => {
          this.handleMessage(event)
        }

        this.ws.onclose = (event) => {
          this.log('WebSocket closed', event.code, event.reason)
          this.stopPing()

          if (!this.isManualClose && this.shouldReconnect()) {
            this.setStatus('disconnected')
            this.scheduleReconnect()
          } else {
            this.setStatus('disconnected')
          }
        }

        this.ws.onerror = (error) => {
          this.log('WebSocket error', error)
          this.setStatus('error')
          this.notifyError(error)

          if (this.status === 'connecting') {
            reject(new Error('Failed to connect to WebSocket'))
          }
        }
      } catch (error) {
        this.log('Failed to create WebSocket', error)
        reject(error)
      }
    })
  }

  /**
   * WebSocket 연결 종료
   */
  disconnect(): void {
    this.isManualClose = true
    this.clearTimers()

    if (this.ws) {
      if (this.ws.readyState === WebSocket.OPEN) {
        this.ws.close(1000, 'Manual disconnect')
      }
      this.ws = null
    }

    this.setStatus('disconnected')
  }

  /**
   * 메시지 전송
   */
  send(message: WebSocketMessage): boolean {
    if (!this.isConnected()) {
      this.log('Cannot send message: WebSocket not connected')
      return false
    }

    try {
      const payload = {
        ...message,
        timestamp: message.timestamp || new Date().toISOString(),
        id: message.id || this.generateMessageId(),
      }

      this.ws!.send(JSON.stringify(payload))
      this.log('Message sent', payload)
      return true
    } catch (error) {
      this.log('Failed to send message', error)
      this.notifyError(error as Error)
      return false
    }
  }

  /**
   * 이벤트 핸들러 등록
   */
  on(event: string, handler: WebSocketEventHandler): () => void {
    if (!this.eventHandlers.has(event)) {
      this.eventHandlers.set(event, [])
    }

    this.eventHandlers.get(event)!.push(handler)

    // 구독 해제 함수 반환
    return () => {
      const handlers = this.eventHandlers.get(event)
      if (handlers) {
        const index = handlers.indexOf(handler)
        if (index > -1) {
          handlers.splice(index, 1)
        }
      }
    }
  }

  /**
   * 에러 핸들러 등록
   */
  onError(handler: WebSocketErrorHandler): () => void {
    this.errorHandlers.push(handler)

    return () => {
      const index = this.errorHandlers.indexOf(handler)
      if (index > -1) {
        this.errorHandlers.splice(index, 1)
      }
    }
  }

  /**
   * 상태 변경 핸들러 등록
   */
  onStatusChange(handler: WebSocketStatusHandler): () => void {
    this.statusHandlers.push(handler)

    return () => {
      const index = this.statusHandlers.indexOf(handler)
      if (index > -1) {
        this.statusHandlers.splice(index, 1)
      }
    }
  }

  /**
   * 현재 상태 반환
   */
  getStatus(): 'connecting' | 'connected' | 'disconnected' | 'error' {
    return this.status
  }

  /**
   * 연결 상태 확인
   */
  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN
  }

  /**
   * 재연결 시도
   */
  reconnect(): Promise<void> {
    this.disconnect()
    return this.connect()
  }

  // Private methods

  private handleMessage(event: MessageEvent): void {
    try {
      const data = JSON.parse(event.data)
      this.log('Message received', data)

      // pong 응답 처리
      if (data.type === 'pong') {
        this.handlePong()
        return
      }

      // 이벤트 핸들러 실행
      const handlers = this.eventHandlers.get(data.type)
      if (handlers) {
        handlers.forEach(handler => {
          try {
            handler(data.payload || data)
          } catch (error) {
            this.log('Error in event handler', error)
          }
        })
      }

      // 전체 메시지 핸들러 실행
      const allHandlers = this.eventHandlers.get('*')
      if (allHandlers) {
        allHandlers.forEach(handler => {
          try {
            handler(data)
          } catch (error) {
            this.log('Error in catch-all event handler', error)
          }
        })
      }
    } catch (error) {
      this.log('Failed to parse message', error)
    }
  }

  private setStatus(status: 'connecting' | 'connected' | 'disconnected' | 'error'): void {
    if (this.status !== status) {
      this.status = status
      this.statusHandlers.forEach(handler => {
        try {
          handler(status)
        } catch (error) {
          this.log('Error in status handler', error)
        }
      })
    }
  }

  private notifyError(error: Event | Error): void {
    this.errorHandlers.forEach(handler => {
      try {
        handler(error)
      } catch (err) {
        this.log('Error in error handler', err)
      }
    })
  }

  private shouldReconnect(): boolean {
    return this.reconnectCount < this.options.maxReconnectAttempts
  }

  private scheduleReconnect(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
    }

    const delay = Math.min(
      this.options.reconnectInterval * Math.pow(2, this.reconnectCount),
      30000, // 최대 30초
    )

    this.log(`Scheduling reconnect attempt ${this.reconnectCount + 1} in ${delay}ms`)

    this.reconnectTimer = setTimeout(() => {
      this.reconnectCount++
      this.connect().catch(() => {
        // 재연결 실패는 onclose에서 처리됨
      })
    }, delay)
  }

  private startPing(): void {
    this.stopPing()

    this.pingTimer = setInterval(() => {
      if (this.isConnected()) {
        this.send({ type: 'ping', payload: {} })

        // pong 타임아웃 설정
        this.pongTimer = setTimeout(() => {
          this.log('Pong timeout - closing connection')
          this.ws?.close()
        }, this.options.pongTimeout)
      }
    }, this.options.pingInterval)
  }

  private stopPing(): void {
    if (this.pingTimer) {
      clearInterval(this.pingTimer)
      this.pingTimer = null
    }

    if (this.pongTimer) {
      clearTimeout(this.pongTimer)
      this.pongTimer = null
    }
  }

  private handlePong(): void {
    if (this.pongTimer) {
      clearTimeout(this.pongTimer)
      this.pongTimer = null
    }
  }

  private clearTimers(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }

    this.stopPing()
  }

  private generateMessageId(): string {
    return Date.now().toString(36) + Math.random().toString(36).substr(2)
  }

  private log(...args: any[]): void {
    if (this.options.debug) {
      console.log('[WebSocket]', ...args)
    }
  }
}

// 전역 WebSocket 인스턴스 관리
class WebSocketService {
  private connections: Map<string, WebSocketManager> = new Map()

  /**
   * WebSocket 연결 생성 또는 기존 연결 반환
   */
  getConnection(name: string, url?: string, options?: WebSocketOptions): WebSocketManager {
    if (this.connections.has(name)) {
      return this.connections.get(name)!
    }

    if (!url) {
      throw new Error(`URL is required for new WebSocket connection: ${name}`)
    }

    const connection = new WebSocketManager(url, options)
    this.connections.set(name, connection)
    return connection
  }

  /**
   * WebSocket 연결 제거
   */
  removeConnection(name: string): void {
    const connection = this.connections.get(name)
    if (connection) {
      connection.disconnect()
      this.connections.delete(name)
    }
  }

  /**
   * 모든 연결 종료
   */
  disconnectAll(): void {
    this.connections.forEach((connection, _name) => {
      connection.disconnect()
    })
    this.connections.clear()
  }

  /**
   * 연결 목록 반환
   */
  getConnectionNames(): string[] {
    return Array.from(this.connections.keys())
  }
}

// 전역 WebSocket 서비스 인스턴스
export const webSocketService = new WebSocketService()

// 기본 내보내기
export { WebSocketManager }
export default WebSocketManager