package claude

import (
	"bytes"
	"fmt"
	"sync"
	"time"
)

// StreamBuffer는 스트림 데이터를 버퍼링하는 구조체입니다.
type StreamBuffer struct {
	buffer    *bytes.Buffer
	maxSize   int
	mutex     sync.RWMutex
	overflow  bool
	written   int64
	read      int64
	createdAt time.Time
	lastWrite time.Time
}

// NewStreamBuffer는 새로운 스트림 버퍼를 생성합니다.
func NewStreamBuffer(maxSize int) *StreamBuffer {
	now := time.Now()
	return &StreamBuffer{
		buffer:    &bytes.Buffer{},
		maxSize:   maxSize,
		createdAt: now,
		lastWrite: now,
	}
}

// Write는 버퍼에 데이터를 씁니다.
func (sb *StreamBuffer) Write(data []byte) (int, error) {
	sb.mutex.Lock()
	defer sb.mutex.Unlock()

	if len(data) == 0 {
		return 0, nil
	}

	// 오버플로우 처리
	if sb.buffer.Len()+len(data) > sb.maxSize {
		sb.overflow = true
		excess := sb.buffer.Len() + len(data) - sb.maxSize

		// 오래된 데이터 제거
		if excess > 0 {
			discarded := make([]byte, excess)
			sb.buffer.Read(discarded)
		}
	}

	n, err := sb.buffer.Write(data)
	if err != nil {
		return n, fmt.Errorf("failed to write to buffer: %w", err)
	}

	sb.written += int64(n)
	sb.lastWrite = time.Now()

	return n, nil
}

// Read는 버퍼에서 데이터를 읽습니다.
func (sb *StreamBuffer) Read(data []byte) (int, error) {
	sb.mutex.RLock()
	defer sb.mutex.RUnlock()

	n, err := sb.buffer.Read(data)
	if err != nil {
		return n, fmt.Errorf("failed to read from buffer: %w", err)
	}

	sb.read += int64(n)
	return n, nil
}

// ReadLine은 버퍼에서 한 줄을 읽습니다.
func (sb *StreamBuffer) ReadLine() ([]byte, error) {
	sb.mutex.Lock()
	defer sb.mutex.Unlock()

	line, err := sb.buffer.ReadBytes('\n')
	if err != nil {
		return line, err
	}

	sb.read += int64(len(line))
	return line, nil
}

// Peek는 버퍼의 처음 n바이트를 읽지 않고 반환합니다.
func (sb *StreamBuffer) Peek(n int) []byte {
	sb.mutex.RLock()
	defer sb.mutex.RUnlock()

	data := sb.buffer.Bytes()
	if len(data) < n {
		n = len(data)
	}

	return data[:n]
}

// Len은 버퍼에 있는 데이터의 크기를 반환합니다.
func (sb *StreamBuffer) Len() int {
	sb.mutex.RLock()
	defer sb.mutex.RUnlock()

	return sb.buffer.Len()
}

// Cap은 버퍼의 최대 크기를 반환합니다.
func (sb *StreamBuffer) Cap() int {
	return sb.maxSize
}

// HasOverflow는 버퍼 오버플로우가 발생했는지 확인합니다.
func (sb *StreamBuffer) HasOverflow() bool {
	sb.mutex.RLock()
	defer sb.mutex.RUnlock()

	return sb.overflow
}

// IsEmpty는 버퍼가 비어있는지 확인합니다.
func (sb *StreamBuffer) IsEmpty() bool {
	sb.mutex.RLock()
	defer sb.mutex.RUnlock()

	return sb.buffer.Len() == 0
}

// IsFull은 버퍼가 가득 찼는지 확인합니다.
func (sb *StreamBuffer) IsFull() bool {
	sb.mutex.RLock()
	defer sb.mutex.RUnlock()

	return sb.buffer.Len() >= sb.maxSize
}

// Reset은 버퍼를 초기화합니다.
func (sb *StreamBuffer) Reset() {
	sb.mutex.Lock()
	defer sb.mutex.Unlock()

	sb.buffer.Reset()
	sb.overflow = false
	sb.written = 0
	sb.read = 0
	sb.lastWrite = time.Now()
}

// String은 버퍼의 내용을 문자열로 반환합니다.
func (sb *StreamBuffer) String() string {
	sb.mutex.RLock()
	defer sb.mutex.RUnlock()

	return sb.buffer.String()
}

// Bytes는 버퍼의 내용을 바이트 슬라이스로 반환합니다.
func (sb *StreamBuffer) Bytes() []byte {
	sb.mutex.RLock()
	defer sb.mutex.RUnlock()

	return sb.buffer.Bytes()
}

// GetStats는 버퍼의 통계 정보를 반환합니다.
func (sb *StreamBuffer) GetStats() map[string]interface{} {
	sb.mutex.RLock()
	defer sb.mutex.RUnlock()

	return map[string]interface{}{
		"size":         sb.buffer.Len(),
		"max_size":     sb.maxSize,
		"usage_ratio":  float64(sb.buffer.Len()) / float64(sb.maxSize),
		"overflow":     sb.overflow,
		"written":      sb.written,
		"read":         sb.read,
		"created_at":   sb.createdAt,
		"last_write":   sb.lastWrite,
		"age_seconds":  time.Since(sb.createdAt).Seconds(),
		"idle_seconds": time.Since(sb.lastWrite).Seconds(),
	}
}

// Resize는 버퍼의 최대 크기를 변경합니다.
func (sb *StreamBuffer) Resize(newMaxSize int) error {
	sb.mutex.Lock()
	defer sb.mutex.Unlock()

	if newMaxSize <= 0 {
		return fmt.Errorf("buffer size must be positive, got: %d", newMaxSize)
	}

	sb.maxSize = newMaxSize

	// 현재 버퍼가 새 크기보다 크면 잘라냄
	if sb.buffer.Len() > newMaxSize {
		sb.overflow = true
		excess := sb.buffer.Len() - newMaxSize
		discarded := make([]byte, excess)
		sb.buffer.Read(discarded)
	}

	return nil
}

// Clone은 버퍼의 복사본을 생성합니다.
func (sb *StreamBuffer) Clone() *StreamBuffer {
	sb.mutex.RLock()
	defer sb.mutex.RUnlock()

	newBuffer := NewStreamBuffer(sb.maxSize)
	newBuffer.buffer.Write(sb.buffer.Bytes())
	newBuffer.overflow = sb.overflow
	newBuffer.written = sb.written
	newBuffer.read = sb.read
	newBuffer.createdAt = sb.createdAt
	newBuffer.lastWrite = sb.lastWrite

	return newBuffer
}