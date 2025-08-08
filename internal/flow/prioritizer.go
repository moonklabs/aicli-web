package flow

import (
	"container/heap"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// MessagePrioritizer 메시지 우선순위 관리자
type MessagePrioritizer struct {
	queues     map[Priority]*PriorityQueue
	dropPolicy DropPolicy
	stats      *DropStatistics
	
	// 설정
	maxQueueSize      int
	maxMessageSize    int
	dropThreshold     float64
	
	mutex             sync.RWMutex
}

// PriorityQueue 우선순위 큐
type PriorityQueue struct {
	messages     []PriorityMessage
	maxSize      int
	currentSize  int64
	mutex        sync.RWMutex
	
	// 통계
	totalEnqueued uint64
	totalDequeued uint64
	totalDropped  uint64
}

// PriorityMessage 우선순위 메시지
type PriorityMessage struct {
	Data        []byte
	Priority    Priority
	Timestamp   time.Time
	SessionID   string
	MessageType MessageType
	Size        int
	index       int // heap 인덱스
}

// DropStatistics 드롭 통계
type DropStatistics struct {
	TotalDropped      uint64
	DroppedByPriority map[Priority]uint64
	DroppedByPolicy   map[DropPolicy]uint64
	DroppedBySize     uint64
	LastDrop          time.Time
	mutex             sync.RWMutex
}

// MessageHeap 우선순위 힙 구현
type MessageHeap []PriorityMessage

// NewMessagePrioritizer 새 메시지 우선순위 관리자 생성
func NewMessagePrioritizer(dropPolicy DropPolicy) *MessagePrioritizer {
	mp := &MessagePrioritizer{
		queues:         make(map[Priority]*PriorityQueue),
		dropPolicy:     dropPolicy,
		maxQueueSize:   10000,
		maxMessageSize: 1024 * 1024, // 1MB
		dropThreshold:  0.9,
		stats: &DropStatistics{
			DroppedByPriority: make(map[Priority]uint64),
			DroppedByPolicy:   make(map[DropPolicy]uint64),
		},
	}
	
	// 각 우선순위별 큐 초기화
	for priority := PriorityLow; priority <= PriorityCritical; priority++ {
		mp.queues[priority] = &PriorityQueue{
			messages: make([]PriorityMessage, 0, 1000),
			maxSize:  mp.maxQueueSize / 4, // 각 우선순위별 최대 크기
		}
	}
	
	return mp
}

// Enqueue 메시지 큐에 추가
func (mp *MessagePrioritizer) Enqueue(message PriorityMessage) error {
	mp.mutex.RLock()
	queue, exists := mp.queues[message.Priority]
	mp.mutex.RUnlock()
	
	if !exists {
		return fmt.Errorf("invalid priority: %d", message.Priority)
	}
	
	// 메시지 크기 확인
	message.Size = len(message.Data)
	if message.Size > mp.maxMessageSize {
		atomic.AddUint64(&mp.stats.DroppedBySize, 1)
		return fmt.Errorf("message too large: %d bytes (max: %d)", message.Size, mp.maxMessageSize)
	}
	
	return queue.Enqueue(message)
}

// Dequeue 메시지 큐에서 제거
func (mp *MessagePrioritizer) Dequeue() (*PriorityMessage, error) {
	// 우선순위 순서대로 확인
	for priority := PriorityCritical; priority >= PriorityLow; priority-- {
		mp.mutex.RLock()
		queue, exists := mp.queues[priority]
		mp.mutex.RUnlock()
		
		if !exists {
			continue
		}
		
		message := queue.Dequeue()
		if message != nil {
			return message, nil
		}
	}
	
	return nil, fmt.Errorf("all queues are empty")
}

// HandleOverflow 오버플로우 처리
func (mp *MessagePrioritizer) HandleOverflow(connectionID string, newMessage PriorityMessage) error {
	mp.mutex.Lock()
	defer mp.mutex.Unlock()
	
	queue := mp.queues[newMessage.Priority]
	if queue == nil {
		return fmt.Errorf("priority queue not found: %d", newMessage.Priority)
	}
	
	// 큐 사용률 확인
	utilization := float64(queue.currentSize) / float64(queue.maxSize)
	
	if utilization >= mp.dropThreshold {
		// 드롭 정책 적용
		droppedMsg, err := mp.applyDropPolicy(queue, newMessage)
		if err != nil {
			return err
		}
		
		if droppedMsg != nil {
			mp.updateDropStatistics(droppedMsg)
			
			log.Warnf("Message dropped for connection %s, priority %d, policy %s, size %d bytes",
				connectionID, droppedMsg.Priority, mp.dropPolicy.String(), droppedMsg.Size)
		}
	}
	
	// 새 메시지 추가
	return queue.Enqueue(newMessage)
}

// applyDropPolicy 드롭 정책 적용
func (mp *MessagePrioritizer) applyDropPolicy(queue *PriorityQueue, newMessage PriorityMessage) (*PriorityMessage, error) {
	switch mp.dropPolicy {
	case DropOldest:
		return mp.dropOldest(queue), nil
		
	case DropLowestPriority:
		return mp.dropLowestPriority(), nil
		
	case DropBySize:
		return mp.dropLargest(queue), nil
		
	case DropRandom:
		return mp.dropRandom(queue), nil
		
	default:
		return nil, fmt.Errorf("unknown drop policy: %d", mp.dropPolicy)
	}
}

// dropOldest 가장 오래된 메시지 드롭
func (mp *MessagePrioritizer) dropOldest(queue *PriorityQueue) *PriorityMessage {
	queue.mutex.Lock()
	defer queue.mutex.Unlock()
	
	if len(queue.messages) == 0 {
		return nil
	}
	
	// 가장 오래된 메시지 찾기
	oldestIdx := 0
	oldestTime := queue.messages[0].Timestamp
	
	for i, msg := range queue.messages {
		if msg.Timestamp.Before(oldestTime) {
			oldestIdx = i
			oldestTime = msg.Timestamp
		}
	}
	
	// 메시지 제거
	droppedMsg := queue.messages[oldestIdx]
	queue.messages = append(queue.messages[:oldestIdx], queue.messages[oldestIdx+1:]...)
	atomic.AddInt64(&queue.currentSize, -int64(droppedMsg.Size))
	atomic.AddUint64(&queue.totalDropped, 1)
	
	return &droppedMsg
}

// dropLowestPriority 가장 낮은 우선순위 메시지 드롭
func (mp *MessagePrioritizer) dropLowestPriority() *PriorityMessage {
	// 낮은 우선순위부터 확인
	for priority := PriorityLow; priority <= PriorityCritical; priority++ {
		queue := mp.queues[priority]
		if queue == nil {
			continue
		}
		
		queue.mutex.Lock()
		if len(queue.messages) > 0 {
			// 첫 번째 메시지 드롭
			droppedMsg := queue.messages[0]
			queue.messages = queue.messages[1:]
			atomic.AddInt64(&queue.currentSize, -int64(droppedMsg.Size))
			atomic.AddUint64(&queue.totalDropped, 1)
			queue.mutex.Unlock()
			return &droppedMsg
		}
		queue.mutex.Unlock()
	}
	
	return nil
}

// dropLargest 가장 큰 메시지 드롭
func (mp *MessagePrioritizer) dropLargest(queue *PriorityQueue) *PriorityMessage {
	queue.mutex.Lock()
	defer queue.mutex.Unlock()
	
	if len(queue.messages) == 0 {
		return nil
	}
	
	// 가장 큰 메시지 찾기
	largestIdx := 0
	largestSize := queue.messages[0].Size
	
	for i, msg := range queue.messages {
		if msg.Size > largestSize {
			largestIdx = i
			largestSize = msg.Size
		}
	}
	
	// 메시지 제거
	droppedMsg := queue.messages[largestIdx]
	queue.messages = append(queue.messages[:largestIdx], queue.messages[largestIdx+1:]...)
	atomic.AddInt64(&queue.currentSize, -int64(droppedMsg.Size))
	atomic.AddUint64(&queue.totalDropped, 1)
	
	return &droppedMsg
}

// dropRandom 랜덤 메시지 드롭
func (mp *MessagePrioritizer) dropRandom(queue *PriorityQueue) *PriorityMessage {
	queue.mutex.Lock()
	defer queue.mutex.Unlock()
	
	if len(queue.messages) == 0 {
		return nil
	}
	
	// 랜덤 인덱스 선택
	randomIdx := rand.Intn(len(queue.messages))
	
	// 메시지 제거
	droppedMsg := queue.messages[randomIdx]
	queue.messages = append(queue.messages[:randomIdx], queue.messages[randomIdx+1:]...)
	atomic.AddInt64(&queue.currentSize, -int64(droppedMsg.Size))
	atomic.AddUint64(&queue.totalDropped, 1)
	
	return &droppedMsg
}

// updateDropStatistics 드롭 통계 업데이트
func (mp *MessagePrioritizer) updateDropStatistics(droppedMsg *PriorityMessage) {
	mp.stats.mutex.Lock()
	defer mp.stats.mutex.Unlock()
	
	atomic.AddUint64(&mp.stats.TotalDropped, 1)
	mp.stats.DroppedByPriority[droppedMsg.Priority]++
	mp.stats.DroppedByPolicy[mp.dropPolicy]++
	mp.stats.LastDrop = time.Now()
}

// GetStatistics 통계 조회
func (mp *MessagePrioritizer) GetStatistics() *DropStatistics {
	mp.stats.mutex.RLock()
	defer mp.stats.mutex.RUnlock()
	
	// 통계 복사본 생성
	stats := &DropStatistics{
		TotalDropped:      atomic.LoadUint64(&mp.stats.TotalDropped),
		DroppedByPriority: make(map[Priority]uint64),
		DroppedByPolicy:   make(map[DropPolicy]uint64),
		DroppedBySize:     atomic.LoadUint64(&mp.stats.DroppedBySize),
		LastDrop:          mp.stats.LastDrop,
	}
	
	for k, v := range mp.stats.DroppedByPriority {
		stats.DroppedByPriority[k] = v
	}
	
	for k, v := range mp.stats.DroppedByPolicy {
		stats.DroppedByPolicy[k] = v
	}
	
	return stats
}

// GetQueueStatus 큐 상태 조회
func (mp *MessagePrioritizer) GetQueueStatus() map[Priority]QueueStatus {
	mp.mutex.RLock()
	defer mp.mutex.RUnlock()
	
	status := make(map[Priority]QueueStatus)
	
	for priority, queue := range mp.queues {
		queue.mutex.RLock()
		status[priority] = QueueStatus{
			Priority:      priority,
			MessageCount:  len(queue.messages),
			TotalSize:     atomic.LoadInt64(&queue.currentSize),
			MaxSize:       queue.maxSize,
			TotalEnqueued: atomic.LoadUint64(&queue.totalEnqueued),
			TotalDequeued: atomic.LoadUint64(&queue.totalDequeued),
			TotalDropped:  atomic.LoadUint64(&queue.totalDropped),
		}
		queue.mutex.RUnlock()
	}
	
	return status
}

// PriorityQueue methods

// Enqueue 메시지 큐에 추가
func (pq *PriorityQueue) Enqueue(message PriorityMessage) error {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()
	
	// 큐 크기 확인
	if len(pq.messages) >= pq.maxSize {
		return fmt.Errorf("queue is full: %d messages", len(pq.messages))
	}
	
	// 메시지 추가
	pq.messages = append(pq.messages, message)
	atomic.AddInt64(&pq.currentSize, int64(message.Size))
	atomic.AddUint64(&pq.totalEnqueued, 1)
	
	return nil
}

// Dequeue 메시지 큐에서 제거
func (pq *PriorityQueue) Dequeue() *PriorityMessage {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()
	
	if len(pq.messages) == 0 {
		return nil
	}
	
	// FIFO 방식으로 제거
	message := pq.messages[0]
	pq.messages = pq.messages[1:]
	atomic.AddInt64(&pq.currentSize, -int64(message.Size))
	atomic.AddUint64(&pq.totalDequeued, 1)
	
	return &message
}

// Peek 메시지 확인 (제거하지 않음)
func (pq *PriorityQueue) Peek() *PriorityMessage {
	pq.mutex.RLock()
	defer pq.mutex.RUnlock()
	
	if len(pq.messages) == 0 {
		return nil
	}
	
	return &pq.messages[0]
}

// Size 큐 크기 조회
func (pq *PriorityQueue) Size() int {
	pq.mutex.RLock()
	defer pq.mutex.RUnlock()
	
	return len(pq.messages)
}

// Clear 큐 비우기
func (pq *PriorityQueue) Clear() {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()
	
	pq.messages = pq.messages[:0]
	atomic.StoreInt64(&pq.currentSize, 0)
}

// QueueStatus 큐 상태
type QueueStatus struct {
	Priority      Priority
	MessageCount  int
	TotalSize     int64
	MaxSize       int
	TotalEnqueued uint64
	TotalDequeued uint64
	TotalDropped  uint64
}

// MessageHeap 인터페이스 구현

func (h MessageHeap) Len() int { return len(h) }

func (h MessageHeap) Less(i, j int) bool {
	// 우선순위가 높을수록 먼저 처리
	if h[i].Priority != h[j].Priority {
		return h[i].Priority > h[j].Priority
	}
	// 같은 우선순위면 타임스탬프 순
	return h[i].Timestamp.Before(h[j].Timestamp)
}

func (h MessageHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *MessageHeap) Push(x interface{}) {
	n := len(*h)
	message := x.(PriorityMessage)
	message.index = n
	*h = append(*h, message)
}

func (h *MessageHeap) Pop() interface{} {
	old := *h
	n := len(old)
	message := old[n-1]
	message.index = -1
	*h = old[0 : n-1]
	return message
}

// PriorityHeapQueue 힙 기반 우선순위 큐
type PriorityHeapQueue struct {
	heap  *MessageHeap
	mutex sync.RWMutex
}

// NewPriorityHeapQueue 새 힙 기반 우선순위 큐 생성
func NewPriorityHeapQueue() *PriorityHeapQueue {
	h := &MessageHeap{}
	heap.Init(h)
	return &PriorityHeapQueue{
		heap: h,
	}
}

// Push 메시지 추가
func (phq *PriorityHeapQueue) Push(message PriorityMessage) {
	phq.mutex.Lock()
	defer phq.mutex.Unlock()
	
	heap.Push(phq.heap, message)
}

// Pop 최고 우선순위 메시지 제거
func (phq *PriorityHeapQueue) Pop() *PriorityMessage {
	phq.mutex.Lock()
	defer phq.mutex.Unlock()
	
	if phq.heap.Len() == 0 {
		return nil
	}
	
	message := heap.Pop(phq.heap).(PriorityMessage)
	return &message
}

// String 드롭 정책 문자열 변환
func (d DropPolicy) String() string {
	switch d {
	case DropOldest:
		return "DropOldest"
	case DropLowestPriority:
		return "DropLowestPriority"
	case DropBySize:
		return "DropBySize"
	case DropRandom:
		return "DropRandom"
	default:
		return "Unknown"
	}
}

// String 우선순위 문자열 변환
func (p Priority) String() string {
	switch p {
	case PriorityLow:
		return "Low"
	case PriorityNormal:
		return "Normal"
	case PriorityHigh:
		return "High"
	case PriorityCritical:
		return "Critical"
	default:
		return "Unknown"
	}
}

// String 메시지 타입 문자열 변환
func (m MessageType) String() string {
	switch m {
	case MessageTypeData:
		return "Data"
	case MessageTypeControl:
		return "Control"
	case MessageTypeSystem:
		return "System"
	case MessageTypeError:
		return "Error"
	default:
		return "Unknown"
	}
}