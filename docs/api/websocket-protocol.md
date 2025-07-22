# WebSocket Protocol Specification

AICode Manager의 WebSocket 프로토콜 명세서입니다. 실시간 통신을 위한 메시지 형식, 연결 절차, 에러 처리 등을 상세히 설명합니다.

## 📋 목차

- [개요](#개요)
- [연결 및 인증](#연결-및-인증)
- [메시지 형식](#메시지-형식)
- [메시지 타입](#메시지-타입)
- [세션 관리](#세션-관리)
- [실시간 기능](#실시간-기능)
- [에러 처리](#에러-처리)
- [보안 고려사항](#보안-고려사항)
- [예제 구현](#예제-구현)

## 🎯 개요

### 프로토콜 특징

- **프로토콜**: WebSocket (RFC 6455)
- **서브프로토콜**: `claude-websocket-v1`
- **메시지 형식**: JSON
- **압축**: Per-message deflate 지원
- **인증**: JWT 토큰 기반
- **하트비트**: Ping/Pong (30초 간격)

### 연결 정보

```
WebSocket URL: wss://api.aicli.example.com/v1/ws
Subprotocol: claude-websocket-v1
Origin: https://app.aicli.example.com
```

## 🔐 연결 및 인증

### 연결 절차

1. **WebSocket 핸드셰이크**
2. **인증 토큰 전송**
3. **세션 바인딩**
4. **연결 확인**

#### 1. WebSocket 핸드셰이크

```http
GET /v1/ws HTTP/1.1
Host: api.aicli.example.com
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==
Sec-WebSocket-Version: 13
Sec-WebSocket-Protocol: claude-websocket-v1
Origin: https://app.aicli.example.com
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

```http
HTTP/1.1 101 Switching Protocols
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Accept: s3pPLMBiTxaQ9kYGzzhZRbK+xOo=
Sec-WebSocket-Protocol: claude-websocket-v1
```

#### 2. 인증 토큰 전송 (선택사항)

인증이 헤더에서 실패한 경우 WebSocket 연결 후 토큰 전송:

```json
{
  "type": "auth",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "token_type": "Bearer"
  },
  "timestamp": 1703123456789,
  "id": "msg_auth_001"
}
```

#### 3. 연결 확인 응답

```json
{
  "type": "connected",
  "data": {
    "connection_id": "conn_abc123def456",
    "user_id": "user_123",
    "session_id": "session_xyz789",
    "server_version": "1.2.0",
    "capabilities": [
      "streaming",
      "file_upload",
      "collaboration",
      "voice_chat"
    ]
  },
  "timestamp": 1703123456789,
  "id": "msg_connected_001"
}
```

## 📝 메시지 형식

### 기본 메시지 구조

```typescript
interface BaseMessage {
  type: string;           // 메시지 타입
  data: any;             // 메시지 데이터
  timestamp: number;     // Unix 타임스탬프 (ms)
  id: string;           // 고유 메시지 ID
  correlation_id?: string; // 응답 연결용 ID
  session_id?: string;   // 세션 ID (선택사항)
}
```

### 메시지 ID 형식

```
msg_{type}_{sequence}_{random}

예제:
- msg_claude_001_abc123
- msg_file_002_def456
- msg_error_003_ghi789
```

### 타임스탬프 형식

```javascript
// JavaScript
timestamp: Date.now()  // 1703123456789

// Go
timestamp: time.Now().UnixMilli()
```

## 🔄 메시지 타입

### 클라이언트 → 서버

#### 1. Claude 메시지 전송

```json
{
  "type": "claude_message",
  "data": {
    "content": "안녕하세요, Claude!",
    "session_id": "session_xyz789",
    "stream": true,
    "context": {
      "language": "ko",
      "format": "markdown"
    }
  },
  "timestamp": 1703123456789,
  "id": "msg_claude_001_abc123"
}
```

#### 2. 파일 업로드 시작

```json
{
  "type": "upload_start",
  "data": {
    "file_id": "file_abc123",
    "filename": "document.pdf",
    "content_type": "application/pdf",
    "size": 1048576,
    "total_chunks": 16,
    "session_id": "session_xyz789"
  },
  "timestamp": 1703123456789,
  "id": "msg_upload_001_def456"
}
```

#### 3. 파일 청크 전송

```json
{
  "type": "upload_chunk",
  "data": {
    "file_id": "file_abc123",
    "chunk_index": 0,
    "data": "base64EncodedChunkData...",
    "checksum": "sha256:a1b2c3d4...",
    "is_last": false
  },
  "timestamp": 1703123456789,
  "id": "msg_chunk_001_ghi789"
}
```

#### 4. 세션 생성

```json
{
  "type": "create_session",
  "data": {
    "project_path": "/path/to/project",
    "session_type": "development",
    "configuration": {
      "language": "go",
      "features": ["code_analysis", "debugging"]
    }
  },
  "timestamp": 1703123456789,
  "id": "msg_session_001_jkl012"
}
```

#### 5. 협업 참가

```json
{
  "type": "join_collaboration",
  "data": {
    "session_id": "session_xyz789",
    "user_id": "user_456",
    "role": "collaborator"
  },
  "timestamp": 1703123456789,
  "id": "msg_collab_001_mno345"
}
```

#### 6. 코드 변경

```json
{
  "type": "code_change",
  "data": {
    "session_id": "session_xyz789",
    "file_path": "main.go",
    "changes": [
      {
        "range": {
          "start": {"line": 10, "character": 5},
          "end": {"line": 10, "character": 15}
        },
        "text": "newFunction()"
      }
    ],
    "version": 42
  },
  "timestamp": 1703123456789,
  "id": "msg_code_001_pqr678"
}
```

#### 7. 커서 위치 업데이트

```json
{
  "type": "cursor_update",
  "data": {
    "session_id": "session_xyz789",
    "file_path": "main.go",
    "position": {
      "line": 15,
      "character": 8
    },
    "selection": {
      "start": {"line": 15, "character": 8},
      "end": {"line": 15, "character": 20}
    }
  },
  "timestamp": 1703123456789,
  "id": "msg_cursor_001_stu901"
}
```

### 서버 → 클라이언트

#### 1. Claude 응답 (스트리밍)

```json
{
  "type": "claude_response",
  "data": {
    "message_id": "msg_claude_001_abc123",
    "content": "안녕하세요! 무엇을 도와드릴까요?",
    "stream": true,
    "completed": false,
    "session_id": "session_xyz789"
  },
  "timestamp": 1703123456790,
  "id": "msg_response_001_vwx234",
  "correlation_id": "msg_claude_001_abc123"
}
```

#### 2. Claude 응답 완료

```json
{
  "type": "claude_response",
  "data": {
    "message_id": "msg_claude_001_abc123",
    "content": "",
    "stream": false,
    "completed": true,
    "session_id": "session_xyz789",
    "metadata": {
      "token_count": 45,
      "processing_time": 1250,
      "model": "claude-3-sonnet"
    }
  },
  "timestamp": 1703123457040,
  "id": "msg_complete_001_yzA567",
  "correlation_id": "msg_claude_001_abc123"
}
```

#### 3. 파일 업로드 확인

```json
{
  "type": "file_uploaded",
  "data": {
    "file_id": "file_abc123",
    "filename": "document.pdf",
    "content_type": "application/pdf",
    "size": 1048576,
    "checksum": "sha256:a1b2c3d4...",
    "url": "/api/v1/files/file_abc123",
    "session_id": "session_xyz789"
  },
  "timestamp": 1703123458000,
  "id": "msg_uploaded_001_BcD890",
  "correlation_id": "msg_upload_001_def456"
}
```

#### 4. 세션 생성 확인

```json
{
  "type": "session_created",
  "data": {
    "session_id": "session_new123",
    "project_path": "/path/to/project",
    "status": "active",
    "capabilities": ["code_analysis", "debugging"],
    "expires_at": 1703209856789
  },
  "timestamp": 1703123459000,
  "id": "msg_created_001_EfG123",
  "correlation_id": "msg_session_001_jkl012"
}
```

#### 5. 협업 이벤트

```json
{
  "type": "participant_joined",
  "data": {
    "session_id": "session_xyz789",
    "user_id": "user_456",
    "username": "jane_doe",
    "role": "collaborator",
    "joined_at": 1703123460000
  },
  "timestamp": 1703123460000,
  "id": "msg_joined_001_HiJ456"
}
```

#### 6. 코드 변경 알림

```json
{
  "type": "code_changed",
  "data": {
    "session_id": "session_xyz789",
    "user_id": "user_456",
    "file_path": "main.go",
    "changes": [
      {
        "range": {
          "start": {"line": 10, "character": 5},
          "end": {"line": 10, "character": 15}
        },
        "text": "newFunction()"
      }
    ],
    "version": 43
  },
  "timestamp": 1703123461000,
  "id": "msg_changed_001_KlM789"
}
```

#### 7. 시스템 상태

```json
{
  "type": "system_status",
  "data": {
    "status": "healthy",
    "active_sessions": 15,
    "queue_length": 3,
    "average_response_time": 850,
    "server_load": 0.65
  },
  "timestamp": 1703123462000,
  "id": "msg_status_001_NoP012"
}
```

## 🔧 세션 관리

### 세션 생성 플로우

```mermaid
sequenceDiagram
    participant C as Client
    participant S as Server
    participant CM as Claude Manager
    
    C->>S: create_session
    S->>CM: Initialize Claude Session
    CM-->>S: Session Ready
    S->>C: session_created
    
    C->>S: claude_message
    S->>CM: Forward Message
    CM-->>S: Stream Response
    S->>C: claude_response (streaming)
    CM-->>S: Response Complete
    S->>C: claude_response (completed)
```

### 세션 상태 관리

```json
{
  "type": "session_status",
  "data": {
    "session_id": "session_xyz789",
    "status": "active",        // active, idle, suspended, terminated
    "created_at": 1703123456789,
    "last_activity": 1703123461000,
    "participant_count": 3,
    "message_count": 25,
    "resource_usage": {
      "memory_mb": 128,
      "cpu_percent": 15
    }
  },
  "timestamp": 1703123462000,
  "id": "msg_status_001_QrS345"
}
```

### 세션 종료

```json
{
  "type": "close_session",
  "data": {
    "session_id": "session_xyz789",
    "reason": "user_request",    // user_request, timeout, error, server_shutdown
    "save_state": true
  },
  "timestamp": 1703123463000,
  "id": "msg_close_001_TuV678"
}
```

## 🚀 실시간 기능

### 스트리밍 응답

스트리밍 응답은 다음과 같은 패턴으로 전송됩니다:

1. **시작 메시지**: `stream: true, completed: false`
2. **중간 메시지들**: `stream: true, completed: false`
3. **완료 메시지**: `stream: false, completed: true`

```json
// 1. 스트리밍 시작
{
  "type": "claude_response",
  "data": {
    "message_id": "msg_claude_001",
    "content": "안녕하세요! ",
    "stream": true,
    "completed": false
  }
}

// 2. 스트리밍 계속
{
  "type": "claude_response",
  "data": {
    "message_id": "msg_claude_001",
    "content": "무엇을 도와드릴까요?",
    "stream": true,
    "completed": false
  }
}

// 3. 스트리밍 완료
{
  "type": "claude_response",
  "data": {
    "message_id": "msg_claude_001",
    "content": "",
    "stream": false,
    "completed": true,
    "metadata": {
      "total_tokens": 45,
      "processing_time": 1250
    }
  }
}
```

### 하트비트 (Ping/Pong)

서버는 30초마다 Ping 프레임을 전송하고, 클라이언트는 Pong 프레임으로 응답해야 합니다.

```javascript
// 클라이언트 측 Pong 핸들러
websocket.addEventListener('ping', () => {
  websocket.pong();
});

// 또는 자동 Pong 응답 (대부분의 브라우저에서 자동 처리)
```

### 연결 상태 확인

```json
{
  "type": "ping",
  "data": {
    "timestamp": 1703123464000
  },
  "timestamp": 1703123464000,
  "id": "msg_ping_001_WxY901"
}
```

클라이언트 응답:

```json
{
  "type": "pong",
  "data": {
    "timestamp": 1703123464050
  },
  "timestamp": 1703123464050,
  "id": "msg_pong_001_ZaB234",
  "correlation_id": "msg_ping_001_WxY901"
}
```

## ❌ 에러 처리

### 에러 메시지 형식

```json
{
  "type": "error",
  "data": {
    "code": "INVALID_SESSION",
    "message": "Session not found or expired",
    "details": {
      "session_id": "session_invalid123",
      "error_time": 1703123465000
    },
    "retry_after": 1000,      // 재시도 대기 시간 (ms)
    "recoverable": true       // 복구 가능 여부
  },
  "timestamp": 1703123465000,
  "id": "msg_error_001_CdE567",
  "correlation_id": "msg_claude_001_abc123"
}
```

### 에러 코드

| 코드 | 설명 | 복구 가능 | 권장 조치 |
|------|------|-----------|-----------|
| `INVALID_AUTH` | 인증 실패 | Yes | 토큰 갱신 후 재시도 |
| `INVALID_SESSION` | 유효하지 않은 세션 | Yes | 새 세션 생성 |
| `RATE_LIMITED` | 속도 제한 초과 | Yes | `retry_after` 후 재시도 |
| `SESSION_FULL` | 세션 참가자 수 초과 | No | 다른 세션 선택 |
| `FILE_TOO_LARGE` | 파일 크기 초과 | No | 파일 분할 또는 크기 감소 |
| `UNSUPPORTED_FORMAT` | 지원하지 않는 형식 | No | 지원 형식으로 변환 |
| `INTERNAL_ERROR` | 서버 내부 오류 | Yes | 잠시 후 재시도 |
| `SERVICE_UNAVAILABLE` | 서비스 일시 중단 | Yes | 서비스 복구 대기 |

### 재연결 로직

```javascript
class WebSocketClient {
  constructor() {
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = 5;
    this.reconnectInterval = 1000;
  }
  
  connect() {
    this.ws = new WebSocket('wss://api.aicli.example.com/v1/ws');
    
    this.ws.onclose = (event) => {
      if (!event.wasClean && this.reconnectAttempts < this.maxReconnectAttempts) {
        setTimeout(() => {
          this.reconnectAttempts++;
          console.log(`Reconnecting... (${this.reconnectAttempts}/${this.maxReconnectAttempts})`);
          this.connect();
        }, this.reconnectInterval * Math.pow(2, this.reconnectAttempts));
      }
    };
    
    this.ws.onopen = () => {
      this.reconnectAttempts = 0;
      console.log('WebSocket connected');
    };
  }
}
```

## 🔒 보안 고려사항

### 인증 및 권한

1. **JWT 토큰 검증**
   - 서명 검증
   - 만료 시간 확인
   - 발급자(issuer) 검증

2. **세션 기반 권한 제어**
   - 세션 소유자 확인
   - 협업 권한 검증
   - 파일 접근 권한 확인

### 메시지 검증

```go
// 서버 측 메시지 검증
func ValidateMessage(msg *Message, conn *Connection) error {
    // 1. 메시지 크기 제한
    if len(msg.Data) > MaxMessageSize {
        return ErrMessageTooLarge
    }
    
    // 2. 속도 제한 확인
    if !conn.RateLimiter.Allow() {
        return ErrRateLimited
    }
    
    // 3. 세션 권한 확인
    if msg.SessionID != "" {
        if !hasSessionAccess(conn.UserID, msg.SessionID) {
            return ErrUnauthorized
        }
    }
    
    // 4. 메시지 타입별 검증
    switch msg.Type {
    case "claude_message":
        return validateClaudeMessage(msg)
    case "upload_chunk":
        return validateUploadChunk(msg)
    default:
        return ErrInvalidMessageType
    }
}
```

### 입력 검증

```typescript
// 클라이언트 측 입력 검증
function validateMessageContent(content: string): boolean {
  // 1. 길이 제한
  if (content.length > 10000) {
    throw new Error('Message too long');
  }
  
  // 2. 금지된 패턴 확인
  const forbiddenPatterns = [
    /<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi,
    /javascript:/gi,
    /data:text\/html/gi
  ];
  
  for (const pattern of forbiddenPatterns) {
    if (pattern.test(content)) {
      throw new Error('Invalid content detected');
    }
  }
  
  return true;
}
```

### 파일 업로드 보안

```json
{
  "type": "upload_start",
  "data": {
    "filename": "document.pdf",
    "content_type": "application/pdf",
    "size": 1048576,
    "checksum": "sha256:a1b2c3d4e5f6...",  // 파일 무결성 검증
    "virus_scan": {
      "required": true,
      "engine": "clamav"
    }
  }
}
```

## 💡 예제 구현

### 완전한 클라이언트 구현

```typescript
// claude-websocket-client.ts
interface ClaudeWebSocketConfig {
  url: string;
  token?: string;
  reconnect?: boolean;
  maxReconnectAttempts?: number;
  pingInterval?: number;
}

class ClaudeWebSocketClient {
  private ws: WebSocket | null = null;
  private config: Required<ClaudeWebSocketConfig>;
  private messageHandlers = new Map<string, Function[]>();
  private pendingRequests = new Map<string, {resolve: Function, reject: Function, timeout: NodeJS.Timeout}>();
  
  constructor(config: ClaudeWebSocketConfig) {
    this.config = {
      reconnect: true,
      maxReconnectAttempts: 5,
      pingInterval: 30000,
      ...config
    };
  }
  
  async connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      const wsUrl = this.config.token 
        ? `${this.config.url}?token=${this.config.token}`
        : this.config.url;
        
      this.ws = new WebSocket(wsUrl, 'claude-websocket-v1');
      
      this.ws.onopen = () => {
        console.log('Connected to Claude WebSocket');
        this.startPingInterval();
        resolve();
      };
      
      this.ws.onmessage = (event) => {
        this.handleMessage(JSON.parse(event.data));
      };
      
      this.ws.onclose = (event) => {
        this.handleDisconnect(event);
      };
      
      this.ws.onerror = (error) => {
        console.error('WebSocket error:', error);
        reject(error);
      };
    });
  }
  
  // 메시지 전송 (Promise 기반)
  async sendMessage(type: string, data: any, timeout = 30000): Promise<any> {
    const messageId = this.generateMessageId();
    const message = {
      type,
      data,
      timestamp: Date.now(),
      id: messageId
    };
    
    return new Promise((resolve, reject) => {
      // 타임아웃 설정
      const timeoutHandle = setTimeout(() => {
        this.pendingRequests.delete(messageId);
        reject(new Error('Request timeout'));
      }, timeout);
      
      // 응답 대기 등록
      this.pendingRequests.set(messageId, {
        resolve,
        reject,
        timeout: timeoutHandle
      });
      
      // 메시지 전송
      if (this.ws && this.ws.readyState === WebSocket.OPEN) {
        this.ws.send(JSON.stringify(message));
      } else {
        reject(new Error('WebSocket not connected'));
      }
    });
  }
  
  // Claude에게 메시지 전송 (스트리밍)
  async sendToClaude(content: string, sessionId?: string): Promise<AsyncIterable<string>> {
    const messageId = this.generateMessageId();
    
    // 스트리밍 이벤트 리스너 등록
    const streamHandler = (message: any) => {
      if (message.data.message_id === messageId) {
        if (message.data.stream && !message.data.completed) {
          // 스트리밍 데이터
          return message.data.content;
        } else if (message.data.completed) {
          // 스트리밍 완료
          this.off('claude_response', streamHandler);
          return null; // 스트리밍 종료 신호
        }
      }
    };
    
    this.on('claude_response', streamHandler);
    
    // 메시지 전송
    await this.sendMessage('claude_message', {
      content,
      session_id: sessionId,
      stream: true
    });
    
    // AsyncIterable 반환
    return {
      async *[Symbol.asyncIterator]() {
        const chunks: string[] = [];
        let completed = false;
        
        const chunkHandler = (message: any) => {
          if (message.data.message_id === messageId) {
            if (message.data.stream && !message.data.completed) {
              chunks.push(message.data.content);
            } else if (message.data.completed) {
              completed = true;
            }
          }
        };
        
        this.on('claude_response', chunkHandler);
        
        try {
          while (!completed) {
            if (chunks.length > 0) {
              yield chunks.shift()!;
            } else {
              await new Promise(resolve => setTimeout(resolve, 10));
            }
          }
          
          // 남은 청크 처리
          while (chunks.length > 0) {
            yield chunks.shift()!;
          }
        } finally {
          this.off('claude_response', chunkHandler);
        }
      }
    };
  }
  
  // 파일 업로드
  async uploadFile(file: File, sessionId?: string): Promise<string> {
    const fileId = this.generateFileId();
    const chunkSize = 64 * 1024; // 64KB
    const totalChunks = Math.ceil(file.size / chunkSize);
    
    // 업로드 시작
    await this.sendMessage('upload_start', {
      file_id: fileId,
      filename: file.name,
      content_type: file.type,
      size: file.size,
      total_chunks: totalChunks,
      session_id: sessionId
    });
    
    // 청크 업로드
    for (let i = 0; i < totalChunks; i++) {
      const start = i * chunkSize;
      const end = Math.min(start + chunkSize, file.size);
      const chunk = file.slice(start, end);
      const chunkData = await this.fileToBase64(chunk);
      
      await this.sendMessage('upload_chunk', {
        file_id: fileId,
        chunk_index: i,
        data: chunkData,
        is_last: i === totalChunks - 1
      });
      
      // 진행 상황 이벤트 발생
      this.emit('upload_progress', {
        fileId,
        progress: ((i + 1) / totalChunks) * 100
      });
    }
    
    // 업로드 완료 대기
    return new Promise((resolve, reject) => {
      const uploadHandler = (message: any) => {
        if (message.data.file_id === fileId) {
          this.off('file_uploaded', uploadHandler);
          resolve(message.data.file_id);
        }
      };
      
      this.on('file_uploaded', uploadHandler);
      
      // 타임아웃 설정
      setTimeout(() => {
        this.off('file_uploaded', uploadHandler);
        reject(new Error('Upload timeout'));
      }, 60000);
    });
  }
  
  // 이벤트 리스너 관리
  on(eventType: string, handler: Function): void {
    if (!this.messageHandlers.has(eventType)) {
      this.messageHandlers.set(eventType, []);
    }
    this.messageHandlers.get(eventType)!.push(handler);
  }
  
  off(eventType: string, handler: Function): void {
    const handlers = this.messageHandlers.get(eventType);
    if (handlers) {
      const index = handlers.indexOf(handler);
      if (index > -1) {
        handlers.splice(index, 1);
      }
    }
  }
  
  emit(eventType: string, data: any): void {
    const handlers = this.messageHandlers.get(eventType) || [];
    handlers.forEach(handler => handler(data));
  }
  
  private handleMessage(message: any): void {
    // 응답 대기 중인 요청 확인
    if (message.correlation_id) {
      const pending = this.pendingRequests.get(message.correlation_id);
      if (pending) {
        clearTimeout(pending.timeout);
        this.pendingRequests.delete(message.correlation_id);
        pending.resolve(message);
        return;
      }
    }
    
    // 이벤트 핸들러 실행
    this.emit(message.type, message);
  }
  
  private handleDisconnect(event: CloseEvent): void {
    console.log('WebSocket disconnected:', event.code, event.reason);
    
    // 재연결 로직
    if (this.config.reconnect && !event.wasClean) {
      this.attemptReconnect();
    }
  }
  
  private async attemptReconnect(): Promise<void> {
    let attempts = 0;
    
    while (attempts < this.config.maxReconnectAttempts) {
      attempts++;
      const delay = Math.min(1000 * Math.pow(2, attempts - 1), 30000);
      
      console.log(`Reconnecting... (${attempts}/${this.config.maxReconnectAttempts})`);
      
      await new Promise(resolve => setTimeout(resolve, delay));
      
      try {
        await this.connect();
        console.log('Reconnected successfully');
        return;
      } catch (error) {
        console.error('Reconnection failed:', error);
      }
    }
    
    console.error('Max reconnection attempts reached');
  }
  
  private startPingInterval(): void {
    setInterval(() => {
      if (this.ws && this.ws.readyState === WebSocket.OPEN) {
        this.sendMessage('ping', { timestamp: Date.now() }).catch(console.error);
      }
    }, this.config.pingInterval);
  }
  
  private generateMessageId(): string {
    return `msg_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }
  
  private generateFileId(): string {
    return `file_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }
  
  private async fileToBase64(file: Blob): Promise<string> {
    return new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => {
        const result = reader.result as string;
        resolve(result.split(',')[1]);
      };
      reader.onerror = reject;
      reader.readAsDataURL(file);
    });
  }
  
  disconnect(): void {
    if (this.ws) {
      this.ws.close(1000, 'Client disconnect');
      this.ws = null;
    }
  }
}

// 사용 예제
async function example() {
  const client = new ClaudeWebSocketClient({
    url: 'wss://api.aicli.example.com/v1/ws',
    token: 'your-jwt-token'
  });
  
  await client.connect();
  
  // Claude와 대화
  const responseStream = await client.sendToClaude('안녕하세요, Claude!');
  
  for await (const chunk of responseStream) {
    console.log('Received:', chunk);
  }
  
  // 파일 업로드
  const fileInput = document.getElementById('file-input') as HTMLInputElement;
  const file = fileInput.files?.[0];
  
  if (file) {
    client.on('upload_progress', (data) => {
      console.log(`Upload progress: ${data.progress}%`);
    });
    
    const fileId = await client.uploadFile(file);
    console.log('File uploaded:', fileId);
  }
}
```

---

**다음 단계**: [에러 복구 시스템](../advanced/error-recovery-system.md)에서 장애 복구 메커니즘을 알아보세요.