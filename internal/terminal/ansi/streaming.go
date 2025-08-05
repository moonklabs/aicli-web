package ansi

import (
	"io"
	"log"
	"sync"
	"time"
)

// StreamingParser provides streaming ANSI parsing capabilities
type StreamingParser struct {
	parser   *ANSIParser
	buffer   *RingBuffer
	commands chan ANSICommand
	config   *StreamConfig
	stats    *StreamStats
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// StreamConfig holds streaming parser configuration
type StreamConfig struct {
	BufferSize     int
	ReadBufferSize int
	MaxQueueSize   int
}

// StreamStats holds streaming statistics
type StreamStats struct {
	BytesRead         uint64
	CommandsProcessed uint64
	Errors            uint64
	StartTime         time.Time
	mutex             sync.RWMutex
}

// NewStreamingParser creates a new streaming parser
func NewStreamingParser(parser *ANSIParser, config *StreamConfig) *StreamingParser {
	if config == nil {
		config = &StreamConfig{
			BufferSize:     100,
			ReadBufferSize: 8192,
			MaxQueueSize:   1000,
		}
	}

	return &StreamingParser{
		parser: parser,
		config: config,
		stats: &StreamStats{
			StartTime: time.Now(),
		},
		stopCh: make(chan struct{}),
	}
}

// StartStream starts streaming from an io.Reader
func (sp *StreamingParser) StartStream(reader io.Reader) (<-chan ANSICommand, error) {
	sp.commands = make(chan ANSICommand, sp.config.BufferSize)

	sp.wg.Add(1)
	go sp.streamWorker(reader)

	return sp.commands, nil
}

// streamWorker is the main streaming worker routine
func (sp *StreamingParser) streamWorker(reader io.Reader) {
	defer sp.wg.Done()
	defer close(sp.commands)

	buffer := make([]byte, sp.config.ReadBufferSize)

	for {
		select {
		case <-sp.stopCh:
			return
		default:
			// 타임아웃을 설정하여 정기적으로 stop 신호 확인
			if readCloser, ok := reader.(io.ReadCloser); ok {
				// ReadCloser인 경우 나중에 닫을 수 있도록 저장
				defer readCloser.Close()
			}

			n, err := reader.Read(buffer)
			if err != nil {
				if err == io.EOF {
					return
				}
				log.Printf("Stream read error: %v", err)
				sp.incrementErrors()
				continue
			}

			if n > 0 {
				sp.incrementBytesRead(uint64(n))

				// 파싱
				commands, parseErr := sp.parser.Parse(buffer[:n])
				if parseErr != nil {
					log.Printf("Parse error: %v", parseErr)
					sp.incrementErrors()
					continue
				}

				// 명령어 전송
				for _, cmd := range commands {
					select {
					case sp.commands <- cmd:
						sp.incrementCommandsProcessed()
					case <-sp.stopCh:
						return
					default:
						// 채널이 가득 찬 경우 드롭
						log.Printf("Command channel full, dropping command")
					}
				}
			}
		}
	}
}

// Stop stops the streaming parser
func (sp *StreamingParser) Stop() {
	close(sp.stopCh)
	sp.wg.Wait()
}

// GetStats returns current streaming statistics
func (sp *StreamingParser) GetStats() StreamStats {
	sp.stats.mutex.RLock()
	defer sp.stats.mutex.RUnlock()
	return *sp.stats
}

// incrementBytesRead safely increments bytes read counter
func (sp *StreamingParser) incrementBytesRead(n uint64) {
	sp.stats.mutex.Lock()
	sp.stats.BytesRead += n
	sp.stats.mutex.Unlock()
}

// incrementCommandsProcessed safely increments commands processed counter
func (sp *StreamingParser) incrementCommandsProcessed() {
	sp.stats.mutex.Lock()
	sp.stats.CommandsProcessed++
	sp.stats.mutex.Unlock()
}

// incrementErrors safely increments error counter
func (sp *StreamingParser) incrementErrors() {
	sp.stats.mutex.Lock()
	sp.stats.Errors++
	sp.stats.mutex.Unlock()
}