package performance

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

// IOOptimizer I/O 성능 최적화 관리자
type IOOptimizer struct {
	readBuffer     *RingBuffer
	writeBuffer    *RingBuffer
	batchProcessor *BatchProcessor
	compressor     *StreamCompressor
	serializer     *FastSerializer
	config         *IOConfig
	metrics        *IOMetrics
	mutex          sync.RWMutex
}

// RingBuffer 링 버퍼
type RingBuffer struct {
	buffer   []byte
	readPos  int64
	writePos int64
	size     int64
	mask     int64
	mutex    sync.RWMutex
}

// BatchProcessor 배치 처리기
type BatchProcessor struct {
	batches     map[string]*MessageBatch
	ticker      *time.Ticker
	batchSize   int
	timeout     time.Duration
	processor   BatchProcessFunc
	mutex       sync.RWMutex
	stopCh      chan struct{}
	wg          sync.WaitGroup
}

// MessageBatch 메시지 배치
type MessageBatch struct {
	Messages  [][]byte
	Count     int
	CreatedAt time.Time
	SessionID string
}

// BatchProcessFunc 배치 처리 함수
type BatchProcessFunc func(batch *MessageBatch) error

// StreamCompressor 스트림 압축기
type StreamCompressor struct {
	compressionLevel int
	threshold        int
	enabled          bool
	stats            *CompressionStats
}

// FastSerializer 고속 직렬화기
type FastSerializer struct {
	bufferPool *sync.Pool
	encoder    Encoder
	decoder    Decoder
}

// Encoder 인코더 인터페이스
type Encoder interface {
	Encode(v interface{}) ([]byte, error)
}

// Decoder 디코더 인터페이스
type Decoder interface {
	Decode(data []byte, v interface{}) error
}

// IOConfig I/O 설정
type IOConfig struct {
	ReadBufferSize   int
	WriteBufferSize  int
	BatchSize        int
	BatchTimeout     time.Duration
	CompressionLevel int
	CompressionThreshold int
	EnableCompression bool
}

// IOMetrics I/O 메트릭
type IOMetrics struct {
	TotalReads       uint64
	TotalWrites      uint64
	BytesRead        uint64
	BytesWritten     uint64
	BatchesProcessed uint64
	CompressionRatio float64
	ReadLatency      time.Duration
	WriteLatency     time.Duration
}

// CompressionStats 압축 통계
type CompressionStats struct {
	TotalCompressed   uint64
	TotalUncompressed uint64
	BytesSaved        int64
	CompressionTime   time.Duration
	DecompressionTime time.Duration
}

// NewIOOptimizer 새 I/O 최적화기 생성
func NewIOOptimizer(config *IOConfig) *IOOptimizer {
	if config == nil {
		config = DefaultIOConfig()
	}
	
	ioo := &IOOptimizer{
		config:  config,
		metrics: &IOMetrics{},
	}
	
	// 링 버퍼 초기화
	ioo.readBuffer = NewRingBuffer(config.ReadBufferSize)
	ioo.writeBuffer = NewRingBuffer(config.WriteBufferSize)
	
	// 배치 처리기 초기화
	ioo.batchProcessor = NewBatchProcessor(config.BatchSize, config.BatchTimeout)
	
	// 압축기 초기화
	ioo.compressor = NewStreamCompressor(config.CompressionLevel, config.CompressionThreshold)
	ioo.compressor.enabled = config.EnableCompression
	
	// 직렬화기 초기화
	ioo.serializer = NewFastSerializer()
	
	return ioo
}

// NewRingBuffer 새 링 버퍼 생성
func NewRingBuffer(size int) *RingBuffer {
	// 크기를 2의 제곱수로 조정
	actualSize := 1
	for actualSize < size {
		actualSize <<= 1
	}
	
	return &RingBuffer{
		buffer: make([]byte, actualSize),
		size:   int64(actualSize),
		mask:   int64(actualSize - 1),
	}
}

// Write 링 버퍼에 쓰기
func (rb *RingBuffer) Write(data []byte) (int, error) {
	rb.mutex.Lock()
	defer rb.mutex.Unlock()
	
	dataLen := len(data)
	if dataLen == 0 {
		return 0, nil
	}
	
	// 사용 가능한 공간 확인
	available := rb.size - (rb.writePos - rb.readPos)
	if int64(dataLen) > available {
		return 0, fmt.Errorf("ring buffer full: need %d, available %d", dataLen, available)
	}
	
	// 데이터 쓰기
	for i := 0; i < dataLen; i++ {
		rb.buffer[rb.writePos&rb.mask] = data[i]
		rb.writePos++
	}
	
	return dataLen, nil
}

// Read 링 버퍼에서 읽기
func (rb *RingBuffer) Read(data []byte) (int, error) {
	rb.mutex.Lock()
	defer rb.mutex.Unlock()
	
	// 읽을 수 있는 데이터 확인
	available := rb.writePos - rb.readPos
	if available == 0 {
		return 0, io.EOF
	}
	
	// 읽을 크기 결정
	readSize := len(data)
	if int64(readSize) > available {
		readSize = int(available)
	}
	
	// 데이터 읽기
	for i := 0; i < readSize; i++ {
		data[i] = rb.buffer[rb.readPos&rb.mask]
		rb.readPos++
	}
	
	return readSize, nil
}

// Available 사용 가능한 데이터 크기
func (rb *RingBuffer) Available() int64 {
	rb.mutex.RLock()
	defer rb.mutex.RUnlock()
	
	return rb.writePos - rb.readPos
}

// Reset 링 버퍼 리셋
func (rb *RingBuffer) Reset() {
	rb.mutex.Lock()
	defer rb.mutex.Unlock()
	
	rb.readPos = 0
	rb.writePos = 0
}

// NewBatchProcessor 새 배치 처리기 생성
func NewBatchProcessor(batchSize int, timeout time.Duration) *BatchProcessor {
	bp := &BatchProcessor{
		batches:   make(map[string]*MessageBatch),
		batchSize: batchSize,
		timeout:   timeout,
		stopCh:    make(chan struct{}),
	}
	
	return bp
}

// Start 배치 처리기 시작
func (bp *BatchProcessor) Start(processor BatchProcessFunc) {
	bp.processor = processor
	bp.ticker = time.NewTicker(bp.timeout)
	
	bp.wg.Add(1)
	go bp.processingLoop()
}

// Stop 배치 처리기 중지
func (bp *BatchProcessor) Stop() {
	close(bp.stopCh)
	if bp.ticker != nil {
		bp.ticker.Stop()
	}
	bp.wg.Wait()
}

// AddMessage 메시지 추가
func (bp *BatchProcessor) AddMessage(sessionID string, message []byte) error {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()
	
	batch, exists := bp.batches[sessionID]
	if !exists {
		batch = &MessageBatch{
			Messages:  make([][]byte, 0, bp.batchSize),
			CreatedAt: time.Now(),
			SessionID: sessionID,
		}
		bp.batches[sessionID] = batch
	}
	
	// 메시지 추가
	batch.Messages = append(batch.Messages, message)
	batch.Count++
	
	// 배치 크기 도달 시 즉시 처리
	if batch.Count >= bp.batchSize {
		if err := bp.processBatch(batch); err != nil {
			return err
		}
		delete(bp.batches, sessionID)
	}
	
	return nil
}

// processingLoop 배치 처리 루프
func (bp *BatchProcessor) processingLoop() {
	defer bp.wg.Done()
	
	for {
		select {
		case <-bp.ticker.C:
			bp.processTimeoutBatches()
			
		case <-bp.stopCh:
			// 남은 배치 모두 처리
			bp.processAllBatches()
			return
		}
	}
}

// processTimeoutBatches 타임아웃된 배치 처리
func (bp *BatchProcessor) processTimeoutBatches() {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()
	
	now := time.Now()
	for sessionID, batch := range bp.batches {
		if now.Sub(batch.CreatedAt) >= bp.timeout {
			if err := bp.processBatch(batch); err != nil {
				log.Errorf("Failed to process batch for session %s: %v", sessionID, err)
			}
			delete(bp.batches, sessionID)
		}
	}
}

// processAllBatches 모든 배치 처리
func (bp *BatchProcessor) processAllBatches() {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()
	
	for sessionID, batch := range bp.batches {
		if err := bp.processBatch(batch); err != nil {
			log.Errorf("Failed to process batch for session %s: %v", sessionID, err)
		}
	}
	
	bp.batches = make(map[string]*MessageBatch)
}

// processBatch 배치 처리
func (bp *BatchProcessor) processBatch(batch *MessageBatch) error {
	if bp.processor == nil || batch.Count == 0 {
		return nil
	}
	
	return bp.processor(batch)
}

// NewStreamCompressor 새 스트림 압축기 생성
func NewStreamCompressor(level, threshold int) *StreamCompressor {
	return &StreamCompressor{
		compressionLevel: level,
		threshold:        threshold,
		stats:            &CompressionStats{},
	}
}

// Compress 데이터 압축
func (sc *StreamCompressor) Compress(data []byte) ([]byte, bool, error) {
	if !sc.enabled || len(data) < sc.threshold {
		return data, false, nil
	}
	
	startTime := time.Now()
	
	var buf bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buf, sc.compressionLevel)
	if err != nil {
		return nil, false, err
	}
	
	if _, err := writer.Write(data); err != nil {
		writer.Close()
		return nil, false, err
	}
	
	if err := writer.Close(); err != nil {
		return nil, false, err
	}
	
	compressed := buf.Bytes()
	
	// 압축이 효과적인지 확인
	if len(compressed) >= len(data) {
		return data, false, nil
	}
	
	// 통계 업데이트
	atomic.AddUint64(&sc.stats.TotalCompressed, 1)
	atomic.AddInt64(&sc.stats.BytesSaved, int64(len(data)-len(compressed)))
	sc.stats.CompressionTime += time.Since(startTime)
	
	return compressed, true, nil
}

// Decompress 데이터 압축 해제
func (sc *StreamCompressor) Decompress(data []byte) ([]byte, error) {
	startTime := time.Now()
	
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		return nil, err
	}
	
	// 통계 업데이트
	atomic.AddUint64(&sc.stats.TotalUncompressed, 1)
	sc.stats.DecompressionTime += time.Since(startTime)
	
	return buf.Bytes(), nil
}

// NewFastSerializer 새 고속 직렬화기 생성
func NewFastSerializer() *FastSerializer {
	return &FastSerializer{
		bufferPool: &sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
		encoder: &BinaryEncoder{},
		decoder: &BinaryDecoder{},
	}
}

// Serialize 직렬화
func (fs *FastSerializer) Serialize(v interface{}) ([]byte, error) {
	return fs.encoder.Encode(v)
}

// Deserialize 역직렬화
func (fs *FastSerializer) Deserialize(data []byte, v interface{}) error {
	return fs.decoder.Decode(data, v)
}

// BinaryEncoder 바이너리 인코더
type BinaryEncoder struct{}

// Encode 인코딩
func (be *BinaryEncoder) Encode(v interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, v)
	return buf.Bytes(), err
}

// BinaryDecoder 바이너리 디코더
type BinaryDecoder struct{}

// Decode 디코딩
func (bd *BinaryDecoder) Decode(data []byte, v interface{}) error {
	buf := bytes.NewReader(data)
	return binary.Read(buf, binary.LittleEndian, v)
}

// OptimizeRead 읽기 최적화
func (ioo *IOOptimizer) OptimizeRead(reader io.Reader) ([]byte, error) {
	startTime := time.Now()
	
	// 링 버퍼로 읽기
	buffer := make([]byte, 4096)
	var totalRead int
	
	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			if _, writeErr := ioo.readBuffer.Write(buffer[:n]); writeErr != nil {
				return nil, writeErr
			}
			totalRead += n
			atomic.AddUint64(&ioo.metrics.BytesRead, uint64(n))
		}
		
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	
	// 버퍼에서 데이터 읽기
	result := make([]byte, totalRead)
	if _, err := ioo.readBuffer.Read(result); err != nil {
		return nil, err
	}
	
	atomic.AddUint64(&ioo.metrics.TotalReads, 1)
	ioo.metrics.ReadLatency = time.Since(startTime)
	
	return result, nil
}

// OptimizeWrite 쓰기 최적화
func (ioo *IOOptimizer) OptimizeWrite(writer io.Writer, data []byte) error {
	startTime := time.Now()
	
	// 압축 시도
	compressed, wasCompressed, err := ioo.compressor.Compress(data)
	if err != nil {
		return err
	}
	
	// 링 버퍼에 쓰기
	if _, err := ioo.writeBuffer.Write(compressed); err != nil {
		return err
	}
	
	// 버퍼에서 읽어서 실제 쓰기
	buffer := make([]byte, len(compressed))
	if _, err := ioo.writeBuffer.Read(buffer); err != nil {
		return err
	}
	
	n, err := writer.Write(buffer)
	if err != nil {
		return err
	}
	
	atomic.AddUint64(&ioo.metrics.BytesWritten, uint64(n))
	atomic.AddUint64(&ioo.metrics.TotalWrites, 1)
	
	if wasCompressed {
		ratio := float64(len(compressed)) / float64(len(data))
		ioo.metrics.CompressionRatio = ratio
	}
	
	ioo.metrics.WriteLatency = time.Since(startTime)
	
	return nil
}

// GetMetrics 메트릭 조회
func (ioo *IOOptimizer) GetMetrics() *IOMetrics {
	return ioo.metrics
}

// DefaultIOConfig 기본 I/O 설정
func DefaultIOConfig() *IOConfig {
	return &IOConfig{
		ReadBufferSize:       64 * 1024,  // 64KB
		WriteBufferSize:      64 * 1024,  // 64KB
		BatchSize:            100,
		BatchTimeout:         100 * time.Millisecond,
		CompressionLevel:     gzip.DefaultCompression,
		CompressionThreshold: 1024, // 1KB
		EnableCompression:    true,
	}
}