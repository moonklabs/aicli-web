---
task_id: T05B_S01
sprint_sequence_id: S01_M02
status: completed
complexity: Medium
last_updated: 2025-07-21 09:44
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
- [x] JSON 스트림 파서 구현
- [x] 비동기 입출력 처리 구현
- [x] 스트림 버퍼링 및 타임아웃 관리
- [x] 이벤트 기반 통신 인터페이스
- [x] 스트림 에러 감지 및 처리
- [x] 백프레셔 및 플로우 제어

## Subtasks
- [x] 스트림 핸들러 인터페이스 설계
- [x] JSON 스트림 파서 구현
- [x] 입출력 버퍼 관리 구현
- [x] 이벤트 시스템 구현
- [x] 타임아웃 및 에러 처리
- [x] 스트림 테스트 작성

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

[2025-07-21 09:34]: T05B_S01 태스크 시작 - Claude CLI 스트림 처리 시스템 구현
[2025-07-21 09:35]: 스트림 핸들러 인터페이스 설계 완료 - StreamHandler 인터페이스와 기본 구조체 정의
[2025-07-21 09:36]: JSON 스트림 파서 구현 완료 - JSONStreamParser 구조체와 파싱 메서드들 구현
[2025-07-21 09:37]: 스트림 버퍼 관리 구현 완료 - StreamBuffer 구조체와 오버플로우 처리 구현
[2025-07-21 09:38]: 이벤트 시스템 구현 완료 - EventBus와 이벤트 발행/구독 시스템 구현
[2025-07-21 09:39]: 스트림 핸들러 구현 완료 - claudeStreamHandler 구조체와 메인 로직 구현
[2025-07-21 09:40]: 스트림 테스트 작성 완료 - 단위 테스트와 통합 테스트 구현
[2025-07-21 09:41]: 고급 통합 테스트 작성 완료 - 실시간 스트리밍, 에러 처리, 고부하 테스트 구현
[2025-07-21 09:42]: 사용 예제 작성 완료 - 기본 사용법, 이벤트 처리, 고처리량, 통계 모니터링 예제 구현
[2025-07-21 09:43]: Go 모듈 의존성 추가 완료 - google/uuid 라이브러리 추가
[2025-07-21 09:43]: 코드 리뷰 - 통과
결과: **통과** 모든 사양과 요구사항을 충족하는 완전한 구현
**범위:** T05B_S01 태스크 - Claude CLI 스트림 처리 시스템 구현 (7개 파일)
**발견사항:** 
  - 경미한 개선 사항 2개 발견 (심각도 1-2)
  - 테스트에서 mock 스트림 사용 (심각도: 2)
  - UUID 의존성 수동 추가 (심각도: 1)
  - 모든 핵심 기능이 사양대로 구현됨
**요약:** 스트림 핸들러, JSON 파서, 이벤트 버스, 버퍼 관리 시스템이 완전히 구현되었으며 포괄적인 테스트와 예제가 제공됨
**권장사항:** 구현이 완료되어 다음 태스크로 진행 가능