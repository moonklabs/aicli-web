package performance

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("component", "performance")

// MemoryManager 고성능 메모리 관리자
type MemoryManager struct {
	pools       map[ObjectType]*ObjectPool
	bufferPool  *BufferPool
	allocator   *CustomAllocator
	gcOptimizer *GCOptimizer
	metrics     *MemoryMetrics
	config      *MemoryConfig
	monitor     *MemoryMonitor
	mutex       sync.RWMutex
}

// ObjectPool 객체 풀
type ObjectPool struct {
	poolType    ObjectType
	objects     chan interface{}
	factory     func() interface{}
	reset       func(interface{})
	maxSize     int
	currentSize int64
	created     int64
	reused      int64
	mutex       sync.RWMutex
}

// ObjectType 객체 타입
type ObjectType int

const (
	ObjectPTYSession ObjectType = iota
	ObjectWebSocketConn
	ObjectTerminalScreen
	ObjectANSICommand
	ObjectSnapshot
	ObjectFlowState
	ObjectStreamBuffer
)

// BufferPool 버퍼 풀
type BufferPool struct {
	pools     map[int]*sync.Pool // 크기별 버퍼 풀
	sizes     []int              // 지원되는 버퍼 크기들
	metrics   *BufferPoolMetrics
	maxSize   int
	alignment int
}

// MemoryConfig 메모리 설정
type MemoryConfig struct {
	MaxPoolSize      int
	BufferSizes      []int
	MaxBufferSize    int
	GCOptimization   bool
	MonitorInterval  time.Duration
	MemoryLimit      int64
	EnableProfiling  bool
}

// MemoryMetrics 메모리 메트릭
type MemoryMetrics struct {
	TotalAllocated   uint64
	TotalFreed       uint64
	CurrentUsage     uint64
	PoolHitRate      float64
	GCPauses         []time.Duration
	LastGCTime       time.Time
	ObjectsInUse     map[ObjectType]int64
	BuffersInUse     int64
}

// BufferPoolMetrics 버퍼 풀 메트릭
type BufferPoolMetrics struct {
	TotalGets    uint64
	TotalPuts    uint64
	TotalAllocs  uint64
	CurrentSize  int64
	HitRate      float64
}

// NewMemoryManager 새 메모리 관리자 생성
func NewMemoryManager(config *MemoryConfig) *MemoryManager {
	if config == nil {
		config = DefaultMemoryConfig()
	}
	
	mm := &MemoryManager{
		pools:       make(map[ObjectType]*ObjectPool),
		config:      config,
		metrics:     &MemoryMetrics{ObjectsInUse: make(map[ObjectType]int64)},
		gcOptimizer: NewGCOptimizer(config.GCOptimization),
	}
	
	// 객체 풀 초기화
	mm.initializeObjectPools()
	
	// 버퍼 풀 초기화
	mm.bufferPool = NewBufferPool(config.BufferSizes, config.MaxBufferSize)
	
	// 커스텀 할당자 초기화
	mm.allocator = NewCustomAllocator(config.MemoryLimit)
	
	// 모니터 초기화
	if config.MonitorInterval > 0 {
		mm.monitor = NewMemoryMonitor(mm, config.MonitorInterval)
		mm.monitor.Start()
	}
	
	return mm
}

// initializeObjectPools 객체 풀 초기화
func (mm *MemoryManager) initializeObjectPools() {
	// PTY 세션 풀
	mm.pools[ObjectPTYSession] = &ObjectPool{
		poolType: ObjectPTYSession,
		objects:  make(chan interface{}, mm.config.MaxPoolSize),
		factory:  mm.createPTYSession,
		reset:    mm.resetPTYSession,
		maxSize:  mm.config.MaxPoolSize,
	}
	
	// WebSocket 연결 풀
	mm.pools[ObjectWebSocketConn] = &ObjectPool{
		poolType: ObjectWebSocketConn,
		objects:  make(chan interface{}, mm.config.MaxPoolSize),
		factory:  mm.createWebSocketConn,
		reset:    mm.resetWebSocketConn,
		maxSize:  mm.config.MaxPoolSize,
	}
	
	// 터미널 스크린 풀
	mm.pools[ObjectTerminalScreen] = &ObjectPool{
		poolType: ObjectTerminalScreen,
		objects:  make(chan interface{}, mm.config.MaxPoolSize/2),
		factory:  mm.createTerminalScreen,
		reset:    mm.resetTerminalScreen,
		maxSize:  mm.config.MaxPoolSize / 2,
	}
	
	// ANSI 명령 풀
	mm.pools[ObjectANSICommand] = &ObjectPool{
		poolType: ObjectANSICommand,
		objects:  make(chan interface{}, mm.config.MaxPoolSize*2),
		factory:  mm.createANSICommand,
		reset:    mm.resetANSICommand,
		maxSize:  mm.config.MaxPoolSize * 2,
	}
	
	// 스냅샷 풀
	mm.pools[ObjectSnapshot] = &ObjectPool{
		poolType: ObjectSnapshot,
		objects:  make(chan interface{}, mm.config.MaxPoolSize/4),
		factory:  mm.createSnapshot,
		reset:    mm.resetSnapshot,
		maxSize:  mm.config.MaxPoolSize / 4,
	}
}

// GetObject 객체 풀에서 객체 가져오기
func (mm *MemoryManager) GetObject(objType ObjectType) interface{} {
	mm.mutex.RLock()
	pool, exists := mm.pools[objType]
	mm.mutex.RUnlock()
	
	if !exists || pool == nil {
		return mm.createObject(objType)
	}
	
	select {
	case obj := <-pool.objects:
		atomic.AddInt64(&pool.reused, 1)
		atomic.AddInt64(&mm.metrics.ObjectsInUse[objType], 1)
		return obj
	default:
		// 풀이 비어있으면 새로 생성
		obj := pool.factory()
		atomic.AddInt64(&pool.created, 1)
		atomic.AddInt64(&mm.metrics.ObjectsInUse[objType], 1)
		return obj
	}
}

// ReturnObject 객체를 풀에 반환
func (mm *MemoryManager) ReturnObject(objType ObjectType, obj interface{}) {
	mm.mutex.RLock()
	pool, exists := mm.pools[objType]
	mm.mutex.RUnlock()
	
	if !exists || pool == nil {
		return
	}
	
	// 객체 리셋
	if pool.reset != nil {
		pool.reset(obj)
	}
	
	// 풀 크기 확인
	if atomic.LoadInt64(&pool.currentSize) < int64(pool.maxSize) {
		select {
		case pool.objects <- obj:
			atomic.AddInt64(&pool.currentSize, 1)
			atomic.AddInt64(&mm.metrics.ObjectsInUse[objType], -1)
		default:
			// 풀이 가득 찬 경우 객체 버림
			atomic.AddInt64(&mm.metrics.ObjectsInUse[objType], -1)
		}
	} else {
		atomic.AddInt64(&mm.metrics.ObjectsInUse[objType], -1)
	}
}

// GetBuffer 버퍼 풀에서 버퍼 가져오기
func (mm *MemoryManager) GetBuffer(size int) []byte {
	return mm.bufferPool.GetBuffer(size)
}

// ReturnBuffer 버퍼를 풀에 반환
func (mm *MemoryManager) ReturnBuffer(buffer []byte) {
	mm.bufferPool.ReturnBuffer(buffer)
}

// Allocate 커스텀 메모리 할당
func (mm *MemoryManager) Allocate(size int) ([]byte, error) {
	return mm.allocator.Allocate(size)
}

// Free 커스텀 메모리 해제
func (mm *MemoryManager) Free(data []byte) {
	mm.allocator.Free(data)
}

// GetMetrics 메트릭 조회
func (mm *MemoryManager) GetMetrics() *MemoryMetrics {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()
	
	// 풀 히트율 계산
	var totalReused, totalCreated int64
	for _, pool := range mm.pools {
		totalReused += atomic.LoadInt64(&pool.reused)
		totalCreated += atomic.LoadInt64(&pool.created)
	}
	
	if totalCreated > 0 {
		mm.metrics.PoolHitRate = float64(totalReused) / float64(totalCreated+totalReused)
	}
	
	// 버퍼 메트릭 업데이트
	bufferMetrics := mm.bufferPool.GetMetrics()
	mm.metrics.BuffersInUse = bufferMetrics.CurrentSize
	
	// 현재 메모리 사용량
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	mm.metrics.CurrentUsage = memStats.Alloc
	mm.metrics.TotalAllocated = memStats.TotalAlloc
	
	return mm.metrics
}

// Optimize 메모리 최적화 실행
func (mm *MemoryManager) Optimize() {
	// GC 최적화
	if mm.gcOptimizer != nil {
		mm.gcOptimizer.Optimize()
	}
	
	// 풀 크기 조정
	mm.adjustPoolSizes()
	
	// 버퍼 풀 정리
	mm.bufferPool.Cleanup()
}

// adjustPoolSizes 풀 크기 동적 조정
func (mm *MemoryManager) adjustPoolSizes() {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()
	
	for objType, pool := range mm.pools {
		currentSize := atomic.LoadInt64(&pool.currentSize)
		reused := atomic.LoadInt64(&pool.reused)
		created := atomic.LoadInt64(&pool.created)
		
		// 사용률 계산
		total := reused + created
		if total == 0 {
			continue
		}
		
		hitRate := float64(reused) / float64(total)
		
		// 히트율이 낮으면 풀 크기 증가
		if hitRate < 0.5 && currentSize < int64(pool.maxSize*2) {
			newSize := pool.maxSize * 2
			log.Debugf("Increasing pool size for %v: %d -> %d (hit rate: %.2f)",
				objType, pool.maxSize, newSize, hitRate)
			pool.maxSize = newSize
		}
		
		// 히트율이 매우 높고 크기가 크면 감소
		if hitRate > 0.9 && currentSize > int64(pool.maxSize/2) && pool.maxSize > 10 {
			newSize := pool.maxSize / 2
			log.Debugf("Decreasing pool size for %v: %d -> %d (hit rate: %.2f)",
				objType, pool.maxSize, newSize, hitRate)
			pool.maxSize = newSize
		}
	}
}

// Shutdown 메모리 관리자 종료
func (mm *MemoryManager) Shutdown() {
	if mm.monitor != nil {
		mm.monitor.Stop()
	}
	
	// 풀 정리
	mm.mutex.Lock()
	for _, pool := range mm.pools {
		close(pool.objects)
	}
	mm.mutex.Unlock()
	
	// 최종 GC 실행
	runtime.GC()
	runtime.Gosched()
}

// Factory 및 Reset 함수들

func (mm *MemoryManager) createPTYSession() interface{} {
	return &PTYSessionObject{}
}

func (mm *MemoryManager) resetPTYSession(obj interface{}) {
	if session, ok := obj.(*PTYSessionObject); ok {
		session.Reset()
	}
}

func (mm *MemoryManager) createWebSocketConn() interface{} {
	return &WebSocketConnObject{}
}

func (mm *MemoryManager) resetWebSocketConn(obj interface{}) {
	if conn, ok := obj.(*WebSocketConnObject); ok {
		conn.Reset()
	}
}

func (mm *MemoryManager) createTerminalScreen() interface{} {
	return &TerminalScreenObject{
		Buffer: make([][]rune, 24),
	}
}

func (mm *MemoryManager) resetTerminalScreen(obj interface{}) {
	if screen, ok := obj.(*TerminalScreenObject); ok {
		screen.Reset()
	}
}

func (mm *MemoryManager) createANSICommand() interface{} {
	return &ANSICommandObject{}
}

func (mm *MemoryManager) resetANSICommand(obj interface{}) {
	if cmd, ok := obj.(*ANSICommandObject); ok {
		cmd.Reset()
	}
}

func (mm *MemoryManager) createSnapshot() interface{} {
	return &SnapshotObject{}
}

func (mm *MemoryManager) resetSnapshot(obj interface{}) {
	if snap, ok := obj.(*SnapshotObject); ok {
		snap.Reset()
	}
}

func (mm *MemoryManager) createObject(objType ObjectType) interface{} {
	switch objType {
	case ObjectPTYSession:
		return mm.createPTYSession()
	case ObjectWebSocketConn:
		return mm.createWebSocketConn()
	case ObjectTerminalScreen:
		return mm.createTerminalScreen()
	case ObjectANSICommand:
		return mm.createANSICommand()
	case ObjectSnapshot:
		return mm.createSnapshot()
	default:
		return nil
	}
}

// NewBufferPool 새 버퍼 풀 생성
func NewBufferPool(sizes []int, maxSize int) *BufferPool {
	bp := &BufferPool{
		pools:     make(map[int]*sync.Pool),
		sizes:     sizes,
		maxSize:   maxSize,
		alignment: 64, // 캐시 라인 정렬
		metrics:   &BufferPoolMetrics{},
	}
	
	// 각 크기별 풀 초기화
	for _, size := range sizes {
		alignedSize := bp.alignSize(size)
		bp.pools[alignedSize] = &sync.Pool{
			New: func() interface{} {
				atomic.AddUint64(&bp.metrics.TotalAllocs, 1)
				return make([]byte, alignedSize)
			},
		}
	}
	
	return bp
}

// GetBuffer 버퍼 가져오기
func (bp *BufferPool) GetBuffer(size int) []byte {
	atomic.AddUint64(&bp.metrics.TotalGets, 1)
	
	// 적절한 크기의 풀 찾기
	poolSize := bp.findPoolSize(size)
	
	pool, exists := bp.pools[poolSize]
	if !exists {
		atomic.AddUint64(&bp.metrics.TotalAllocs, 1)
		return make([]byte, size)
	}
	
	if buffer := pool.Get(); buffer != nil {
		buf := buffer.([]byte)
		atomic.AddInt64(&bp.metrics.CurrentSize, 1)
		return buf[:size] // 슬라이스 크기 조정
	}
	
	atomic.AddUint64(&bp.metrics.TotalAllocs, 1)
	return make([]byte, size)
}

// ReturnBuffer 버퍼 반환
func (bp *BufferPool) ReturnBuffer(buffer []byte) {
	if cap(buffer) > bp.maxSize {
		return // 너무 큰 버퍼는 풀에 반환하지 않음
	}
	
	atomic.AddUint64(&bp.metrics.TotalPuts, 1)
	
	// 용량에 맞는 풀 찾기
	poolSize := bp.findPoolSize(cap(buffer))
	pool, exists := bp.pools[poolSize]
	
	if exists && pool != nil {
		// 버퍼 초기화 (보안상 중요)
		for i := range buffer[:cap(buffer)] {
			buffer[i] = 0
		}
		pool.Put(buffer[:cap(buffer)])
		atomic.AddInt64(&bp.metrics.CurrentSize, -1)
	}
}

// findPoolSize 적절한 풀 크기 찾기
func (bp *BufferPool) findPoolSize(size int) int {
	for _, poolSize := range bp.sizes {
		if size <= poolSize {
			return bp.alignSize(poolSize)
		}
	}
	return bp.alignSize(bp.maxSize)
}

// alignSize 캐시 라인 정렬
func (bp *BufferPool) alignSize(size int) int {
	if size%bp.alignment == 0 {
		return size
	}
	return ((size / bp.alignment) + 1) * bp.alignment
}

// GetMetrics 메트릭 조회
func (bp *BufferPool) GetMetrics() *BufferPoolMetrics {
	gets := atomic.LoadUint64(&bp.metrics.TotalGets)
	allocs := atomic.LoadUint64(&bp.metrics.TotalAllocs)
	
	if gets > 0 {
		bp.metrics.HitRate = float64(gets-allocs) / float64(gets)
	}
	
	return bp.metrics
}

// Cleanup 버퍼 풀 정리
func (bp *BufferPool) Cleanup() {
	// sync.Pool은 GC가 자동으로 정리함
	// 명시적인 정리가 필요한 경우 여기에 구현
}

// DefaultMemoryConfig 기본 메모리 설정
func DefaultMemoryConfig() *MemoryConfig {
	return &MemoryConfig{
		MaxPoolSize:     100,
		BufferSizes:     []int{512, 1024, 2048, 4096, 8192, 16384, 32768, 65536},
		MaxBufferSize:   1024 * 1024, // 1MB
		GCOptimization:  true,
		MonitorInterval: 10 * time.Second,
		MemoryLimit:     2 * 1024 * 1024 * 1024, // 2GB
		EnableProfiling: false,
	}
}

// 플레이스홀더 객체 타입들 (실제 구현에서는 적절한 타입으로 교체)

type PTYSessionObject struct {
	ID   string
	Data []byte
}

func (p *PTYSessionObject) Reset() {
	p.ID = ""
	p.Data = nil
}

type WebSocketConnObject struct {
	ID   string
	Conn interface{}
}

func (w *WebSocketConnObject) Reset() {
	w.ID = ""
	w.Conn = nil
}

type TerminalScreenObject struct {
	Buffer [][]rune
	Width  int
	Height int
}

func (t *TerminalScreenObject) Reset() {
	for i := range t.Buffer {
		t.Buffer[i] = nil
	}
	t.Width = 0
	t.Height = 0
}

type ANSICommandObject struct {
	Command string
	Params  []int
}

func (a *ANSICommandObject) Reset() {
	a.Command = ""
	a.Params = nil
}

type SnapshotObject struct {
	Data      []byte
	Timestamp time.Time
}

func (s *SnapshotObject) Reset() {
	s.Data = nil
	s.Timestamp = time.Time{}
}

// String ObjectType 문자열 변환
func (o ObjectType) String() string {
	switch o {
	case ObjectPTYSession:
		return "PTYSession"
	case ObjectWebSocketConn:
		return "WebSocketConn"
	case ObjectTerminalScreen:
		return "TerminalScreen"
	case ObjectANSICommand:
		return "ANSICommand"
	case ObjectSnapshot:
		return "Snapshot"
	case ObjectFlowState:
		return "FlowState"
	case ObjectStreamBuffer:
		return "StreamBuffer"
	default:
		return fmt.Sprintf("Unknown(%d)", o)
	}
}