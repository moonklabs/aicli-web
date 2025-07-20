---
task_id: T05B_S01
sprint_sequence_id: S01_M02
status: open
complexity: Medium
last_updated: 2025-07-21T06:31:00Z
github_issue: # Optional: GitHub issue number
---

# Task: Claude CLI 스트림 처리 시스템 구현

## Description
Claude CLI와의 입출력 스트림 처리 및 JSON 파싱 시스템을 구현합니다. 실시간 스트림 처리, 버퍼링, 기본 통신 인터페이스를 통해 안정적인 데이터 교환을 제공합니다.

## Goal / Objectives
- JSON 스트림 파싱 및 처리 시스템 구현
- 비동기 입출력 스트림 관리
- 버퍼링 및 타임아웃 처리
- 통신 인터페이스 및 이벤트 시스템

## Acceptance Criteria
- [ ] JSON 스트림 파서 구현
- [ ] 비동기 입출력 처리 구현
- [ ] 스트림 버퍼링 및 타임아웃 관리
- [ ] 이벤트 기반 통신 인터페이스
- [ ] 스트림 에러 감지 및 처리
- [ ] 백프레셰 및 플로우 제어

## Subtasks
- [ ] 스트림 핸들러 인터페이스 설계
- [ ] JSON 스트림 파서 구현
- [ ] 입출력 버퍼 관리 구현
- [ ] 이벤트 시스템 구현
- [ ] 타임아웃 및 에러 처리
- [ ] 스트림 테스트 작성

## 기술 가이드

### 스트림 핸들러 인터페이스
```go
type StreamHandler interface {
    Start(stdin io.WriteCloser, stdout, stderr io.ReadCloser) error
    SendMessage(msg *Message) error
    ReceiveMessage(timeout time.Duration) (*Response, error)
    Subscribe(eventType string, handler EventHandler) error
    Close() error
}

type Message struct {
    Type    string                 `json:"type"`
    Content string                 `json:"content"`
    Meta    map[string]interface{} `json:"meta,omitempty"`
    ID      string                 `json:"id"`
}

type Response struct {
    Type      string                 `json:"type"`
    Content   string                 `json:"content"`
    Metadata  map[string]interface{} `json:"metadata,omitempty"`
    MessageID string                 `json:"message_id,omitempty"`
    Error     *StreamError          `json:"error,omitempty"`
}

type StreamEvent struct {
    Type      string      `json:"type"`
    Data      interface{} `json:"data"`
    Timestamp time.Time   `json:"timestamp"`
}

type EventHandler func(event *StreamEvent) error
```

### JSON 스트림 파서 구현
```go
type JSONStreamParser struct {
    scanner   *bufio.Scanner
    decoder   *json.Decoder
    buffer    *bytes.Buffer
    mutex     sync.RWMutex
    logger    *logrus.Logger
}

func NewJSONStreamParser(reader io.Reader, logger *logrus.Logger) *JSONStreamParser {
    buffer := &bytes.Buffer{}
    return &JSONStreamParser{
        scanner: bufio.NewScanner(reader),
        decoder: json.NewDecoder(reader),
        buffer:  buffer,
        logger:  logger,
    }
}

func (p *JSONStreamParser) ParseNext() (*Response, error) {
    p.mutex.Lock()
    defer p.mutex.Unlock()
    
    // 스트림에서 다음 JSON 객체 읽기
    var response Response
    if err := p.decoder.Decode(&response); err != nil {
        if err == io.EOF {
            return nil, err
        }
        return nil, fmt.Errorf("failed to decode JSON: %w", err)
    }
    
    p.logger.WithFields(logrus.Fields{
        "type":       response.Type,
        "message_id": response.MessageID,
    }).Debug("Parsed JSON response")
    
    return &response, nil
}

func (p *JSONStreamParser) ParseStream(ctx context.Context) (<-chan *Response, <-chan error) {
    responseChan := make(chan *Response, 10)
    errorChan := make(chan error, 1)
    
    go func() {
        defer close(responseChan)
        defer close(errorChan)
        
        for {
            select {
            case <-ctx.Done():
                return
            default:
                response, err := p.ParseNext()
                if err != nil {
                    if err == io.EOF {
                        return
                    }
                    errorChan <- err
                    return
                }
                
                select {
                case responseChan <- response:
                case <-ctx.Done():
                    return
                }
            }
        }
    }()
    
    return responseChan, errorChan
}
```

### 스트림 핸들러 구현
```go
type claudeStreamHandler struct {
    stdin        io.WriteCloser
    stdout       io.ReadCloser
    stderr       io.ReadCloser
    parser       *JSONStreamParser
    eventBus     *EventBus
    buffer       *StreamBuffer
    isRunning    bool
    mutex        sync.RWMutex
    ctx          context.Context
    cancel       context.CancelFunc
    logger       *logrus.Logger
}

func NewStreamHandler(logger *logrus.Logger) StreamHandler {
    return &claudeStreamHandler{
        eventBus: NewEventBus(),
        buffer:   NewStreamBuffer(1024 * 1024), // 1MB 버퍼
        logger:   logger,
    }
}

func (sh *claudeStreamHandler) Start(stdin io.WriteCloser, stdout, stderr io.ReadCloser) error {
    sh.mutex.Lock()
    defer sh.mutex.Unlock()
    
    if sh.isRunning {
        return fmt.Errorf("stream handler is already running")
    }
    
    sh.stdin = stdin
    sh.stdout = stdout
    sh.stderr = stderr
    sh.parser = NewJSONStreamParser(stdout, sh.logger)
    sh.ctx, sh.cancel = context.WithCancel(context.Background())
    sh.isRunning = true
    
    // 스트림 처리 고루틴 시작
    go sh.processOutputStream()
    go sh.processErrorStream()
    
    sh.logger.Info("Stream handler started")
    return nil
}

func (sh *claudeStreamHandler) processOutputStream() {
    responseChan, errorChan := sh.parser.ParseStream(sh.ctx)
    
    for {
        select {
        case response := <-responseChan:
            if response == nil {
                return
            }
            sh.handleResponse(response)
            
        case err := <-errorChan:
            if err != nil {
                sh.handleStreamError(err)
                return
            }
            
        case <-sh.ctx.Done():
            return
        }
    }
}

func (sh *claudeStreamHandler) SendMessage(msg *Message) error {
    sh.mutex.RLock()
    defer sh.mutex.RUnlock()
    
    if !sh.isRunning {
        return fmt.Errorf("stream handler is not running")
    }
    
    // 메시지 ID 생성
    if msg.ID == "" {
        msg.ID = generateMessageID()
    }
    
    // JSON 인코딩
    data, err := json.Marshal(msg)
    if err != nil {
        return fmt.Errorf("failed to marshal message: %w", err)
    }
    
    // 스트림에 쓰기
    if _, err := sh.stdin.Write(append(data, '\n')); err != nil {
        return fmt.Errorf("failed to write to stdin: %w", err)
    }
    
    sh.logger.WithFields(logrus.Fields{
        "type": msg.Type,
        "id":   msg.ID,
    }).Debug("Message sent")
    
    return nil
}
```

### 스트림 버퍼 관리
```go
type StreamBuffer struct {
    buffer    *bytes.Buffer
    maxSize   int
    mutex     sync.RWMutex
    overflow  bool
}

func NewStreamBuffer(maxSize int) *StreamBuffer {
    return &StreamBuffer{
        buffer:  &bytes.Buffer{},
        maxSize: maxSize,
    }
}

func (sb *StreamBuffer) Write(data []byte) (int, error) {
    sb.mutex.Lock()
    defer sb.mutex.Unlock()
    
    if sb.buffer.Len()+len(data) > sb.maxSize {
        // 버퍼 오버플로우 처리
        sb.overflow = true
        excess := sb.buffer.Len() + len(data) - sb.maxSize
        
        // 오래된 데이터 제거
        sb.buffer.Next(excess)
    }
    
    return sb.buffer.Write(data)
}

func (sb *StreamBuffer) Read(data []byte) (int, error) {
    sb.mutex.RLock()
    defer sb.mutex.RUnlock()
    
    return sb.buffer.Read(data)
}

func (sb *StreamBuffer) Len() int {
    sb.mutex.RLock()
    defer sb.mutex.RUnlock()
    
    return sb.buffer.Len()
}

func (sb *StreamBuffer) HasOverflow() bool {
    sb.mutex.RLock()
    defer sb.mutex.RUnlock()
    
    return sb.overflow
}
```

### 이벤트 시스템
```go
type EventBus struct {
    subscribers map[string][]EventHandler
    mutex       sync.RWMutex
}

func NewEventBus() *EventBus {
    return &EventBus{
        subscribers: make(map[string][]EventHandler),
    }
}

func (eb *EventBus) Subscribe(eventType string, handler EventHandler) error {
    eb.mutex.Lock()
    defer eb.mutex.Unlock()
    
    eb.subscribers[eventType] = append(eb.subscribers[eventType], handler)
    return nil
}

func (eb *EventBus) Publish(event *StreamEvent) {
    eb.mutex.RLock()
    handlers := eb.subscribers[event.Type]
    eb.mutex.RUnlock()
    
    for _, handler := range handlers {
        go func(h EventHandler) {
            if err := h(event); err != nil {
                // 에러 로깅
                log.Printf("Event handler error: %v", err)
            }
        }(handler)
    }
}
```

### 타임아웃 및 에러 처리
```go
func (sh *claudeStreamHandler) ReceiveMessage(timeout time.Duration) (*Response, error) {
    ctx, cancel := context.WithTimeout(sh.ctx, timeout)
    defer cancel()
    
    select {
    case response := <-sh.responseChan:
        return response, nil
    case err := <-sh.errorChan:
        return nil, err
    case <-ctx.Done():
        return nil, fmt.Errorf("receive timeout after %v", timeout)
    }
}

type StreamError struct {
    Type    string `json:"type"`
    Message string `json:"message"`
    Code    int    `json:"code,omitempty"`
}

func (e *StreamError) Error() string {
    return fmt.Sprintf("stream error [%s]: %s", e.Type, e.Message)
}
```

### 성능 최적화
- **버퍼 풀링**: sync.Pool을 활용한 버퍼 재사용
- **고루틴 풀**: 워커 풀 패턴으로 고루틴 수 제한
- **백프레셔**: 채널 버퍼 크기 조정으로 플로우 제어
- **메모리 관리**: 큰 응답 데이터의 스트리밍 처리

## Output Log
*(This section is populated as work progresses on the task)*