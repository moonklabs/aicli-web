package ansi

import (
	"errors"
	"sync"
)

// RingBuffer is a circular buffer implementation for efficient streaming
type RingBuffer struct {
	data     []byte
	head     int
	tail     int
	size     int
	capacity int
	mutex    sync.RWMutex
}

// NewRingBuffer creates a new ring buffer with the specified capacity
func NewRingBuffer(capacity int) *RingBuffer {
	return &RingBuffer{
		data:     make([]byte, capacity),
		capacity: capacity,
		head:     0,
		tail:     0,
		size:     0,
	}
}

// Write writes data to the ring buffer
func (rb *RingBuffer) Write(data []byte) (int, error) {
	rb.mutex.Lock()
	defer rb.mutex.Unlock()

	if len(data) == 0 {
		return 0, nil
	}

	available := rb.capacity - rb.size
	if available == 0 {
		return 0, errors.New("buffer full")
	}

	// 쓸 수 있는 만큼만 쓰기
	toWrite := len(data)
	if toWrite > available {
		toWrite = available
	}

	written := 0
	for written < toWrite {
		// tail에서 버퍼 끝까지 쓸 수 있는 공간
		space := rb.capacity - rb.tail
		if space > toWrite-written {
			space = toWrite - written
		}

		copy(rb.data[rb.tail:rb.tail+space], data[written:written+space])
		rb.tail = (rb.tail + space) % rb.capacity
		written += space
	}

	rb.size += written
	return written, nil
}

// Read reads data from the ring buffer
func (rb *RingBuffer) Read(data []byte) (int, error) {
	rb.mutex.Lock()
	defer rb.mutex.Unlock()

	if rb.size == 0 {
		return 0, nil
	}

	toRead := len(data)
	if toRead > rb.size {
		toRead = rb.size
	}

	read := 0
	for read < toRead {
		// head에서 버퍼 끝까지 읽을 수 있는 데이터
		available := rb.capacity - rb.head
		
		// 버퍼가 래핑된 경우 처리
		if rb.head > rb.tail {
			// head부터 버퍼 끝까지만 읽을 수 있음
			// available은 이미 capacity - head로 설정되어 있음
		} else if rb.head < rb.tail {
			// head부터 tail까지만 읽을 수 있음
			available = rb.tail - rb.head
		}
		
		// 읽을 수 있는 데이터가 남은 읽기 요청보다 많은 경우
		if available > toRead-read {
			available = toRead - read
		}
		
		// available이 0 이하인 경우 처리
		if available <= 0 {
			break
		}

		copy(data[read:read+available], rb.data[rb.head:rb.head+available])
		rb.head = (rb.head + available) % rb.capacity
		read += available
	}

	rb.size -= read
	return read, nil
}

// Peek returns up to n bytes without removing them from the buffer
func (rb *RingBuffer) Peek(n int) []byte {
	rb.mutex.RLock()
	defer rb.mutex.RUnlock()

	if rb.size == 0 || n <= 0 {
		return nil
	}

	if n > rb.size {
		n = rb.size
	}

	result := make([]byte, n)
	head := rb.head
	for i := 0; i < n; i++ {
		result[i] = rb.data[head]
		head = (head + 1) % rb.capacity
	}

	return result
}

// Size returns the current size of data in the buffer
func (rb *RingBuffer) Size() int {
	rb.mutex.RLock()
	defer rb.mutex.RUnlock()
	return rb.size
}

// Capacity returns the capacity of the buffer
func (rb *RingBuffer) Capacity() int {
	return rb.capacity
}

// Available returns the available space in the buffer
func (rb *RingBuffer) Available() int {
	rb.mutex.RLock()
	defer rb.mutex.RUnlock()
	return rb.capacity - rb.size
}

// Clear clears the buffer
func (rb *RingBuffer) Clear() {
	rb.mutex.Lock()
	defer rb.mutex.Unlock()
	rb.head = 0
	rb.tail = 0
	rb.size = 0
}