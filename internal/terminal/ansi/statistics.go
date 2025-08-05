package ansi

import (
	"sync"
	"time"
)

// ParserStatistics holds parser performance statistics
type ParserStatistics struct {
	BytesProcessed    uint64
	CommandsParsed    uint64
	ErrorsEncountered uint64
	StartTime         time.Time
	LastReset         time.Time
	mutex             sync.RWMutex
}

// Reset resets the statistics
func (ps *ParserStatistics) Reset() {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	ps.BytesProcessed = 0
	ps.CommandsParsed = 0
	ps.ErrorsEncountered = 0
	ps.LastReset = time.Now()
}

// GetThroughput returns the throughput in bytes per second
func (ps *ParserStatistics) GetThroughput() float64 {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	duration := time.Since(ps.LastReset)
	if duration == 0 {
		return 0
	}

	return float64(ps.BytesProcessed) / duration.Seconds()
}

// GetCommandRate returns the command parsing rate per second
func (ps *ParserStatistics) GetCommandRate() float64 {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	duration := time.Since(ps.LastReset)
	if duration == 0 {
		return 0
	}

	return float64(ps.CommandsParsed) / duration.Seconds()
}

// GetErrorRate returns the error rate
func (ps *ParserStatistics) GetErrorRate() float64 {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	if ps.BytesProcessed == 0 {
		return 0
	}

	return float64(ps.ErrorsEncountered) / float64(ps.BytesProcessed)
}

// GetUptime returns the time since the parser was created
func (ps *ParserStatistics) GetUptime() time.Duration {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	return time.Since(ps.StartTime)
}

// IncrementBytesProcessed safely increments the bytes processed counter
func (ps *ParserStatistics) IncrementBytesProcessed(n uint64) {
	ps.mutex.Lock()
	ps.BytesProcessed += n
	ps.mutex.Unlock()
}

// IncrementCommandsParsed safely increments the commands parsed counter
func (ps *ParserStatistics) IncrementCommandsParsed(n uint64) {
	ps.mutex.Lock()
	ps.CommandsParsed += n
	ps.mutex.Unlock()
}

// IncrementErrors safely increments the error counter
func (ps *ParserStatistics) IncrementErrors() {
	ps.mutex.Lock()
	ps.ErrorsEncountered++
	ps.mutex.Unlock()
}