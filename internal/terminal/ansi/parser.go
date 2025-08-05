// Package ansi provides ANSI escape sequence parsing for terminal emulation
package ansi

import (
	"fmt"
	"io"
	"sync"
	"time"
)

// ParserState represents the current state of the ANSI parser
type ParserState int

const (
	StateGround ParserState = iota
	StateEscape
	StateCSI
	StateOSC
	StateDCS
	StateString
	StateIgnore
)

// ANSIParser is the main ANSI escape sequence parser
type ANSIParser struct {
	state       ParserState
	buffer      []byte
	textBuffer  []byte
	commands    []ANSICommand
	config      *ParserConfig
	statistics  *ParserStatistics
	mutex       sync.RWMutex
}

// ParserConfig holds configuration for the ANSI parser
type ParserConfig struct {
	BufferSize        int
	MaxCommandBuffer  int
	StrictMode        bool // If true, errors on unknown sequences
	TrackStatistics   bool
}

// NewANSIParser creates a new ANSI parser with default configuration
func NewANSIParser() *ANSIParser {
	return NewANSIParserWithConfig(DefaultParserConfig())
}

// NewANSIParserWithConfig creates a new ANSI parser with custom configuration
func NewANSIParserWithConfig(config *ParserConfig) *ANSIParser {
	return &ANSIParser{
		state:      StateGround,
		buffer:     make([]byte, 0, config.BufferSize),
		textBuffer: make([]byte, 0, config.BufferSize),
		commands:   make([]ANSICommand, 0, config.MaxCommandBuffer),
		config:     config,
		statistics: &ParserStatistics{
			StartTime: time.Now(),
			LastReset: time.Now(),
		},
	}
}

// DefaultParserConfig returns default parser configuration
func DefaultParserConfig() *ParserConfig {
	return &ParserConfig{
		BufferSize:       4096,
		MaxCommandBuffer: 100,
		StrictMode:       false,
		TrackStatistics:  true,
	}
}

// Parse processes a byte slice and returns parsed ANSI commands
func (ap *ANSIParser) Parse(data []byte) ([]ANSICommand, error) {
	ap.mutex.Lock()
	defer ap.mutex.Unlock()

	// 통계 업데이트
	if ap.config.TrackStatistics {
		ap.statistics.BytesProcessed += uint64(len(data))
	}

	// 이전 명령어 버퍼 초기화
	ap.commands = ap.commands[:0]

	// 바이트 단위로 처리
	for _, b := range data {
		if err := ap.processByte(b); err != nil {
			if ap.config.StrictMode {
				return nil, fmt.Errorf("parsing error at byte %02x: %w", b, err)
			}
			// Non-strict mode에서는 오류 무시하고 계속
			if ap.config.TrackStatistics {
				ap.statistics.ErrorsEncountered++
			}
		}
	}

	// 마지막 텍스트 버퍼 플러시
	ap.flushTextBuffer()

	// 통계 업데이트
	if ap.config.TrackStatistics {
		ap.statistics.CommandsParsed += uint64(len(ap.commands))
	}

	// 결과 복사하여 반환
	result := make([]ANSICommand, len(ap.commands))
	copy(result, ap.commands)

	return result, nil
}

// processByte processes a single byte according to the current parser state
func (ap *ANSIParser) processByte(b byte) error {
	switch ap.state {
	case StateGround:
		return ap.processGround(b)
	case StateEscape:
		return ap.processEscape(b)
	case StateCSI:
		return ap.processCSI(b)
	case StateOSC:
		return ap.processOSC(b)
	case StateDCS:
		return ap.processDCS(b)
	case StateString:
		return ap.processString(b)
	case StateIgnore:
		return ap.processIgnore(b)
	default:
		return fmt.Errorf("unknown parser state: %d", ap.state)
	}
}

// processGround handles bytes in the ground state (normal text)
func (ap *ANSIParser) processGround(b byte) error {
	switch b {
	case 0x1B: // ESC
		ap.flushTextBuffer()
		ap.state = StateEscape
		ap.buffer = ap.buffer[:0]
		ap.buffer = append(ap.buffer, b)
	case 0x0A: // LF
		ap.flushTextBuffer()
		ap.addCommand(ANSICommand{Type: ANSILineFeed})
	case 0x0D: // CR
		ap.flushTextBuffer()
		ap.addCommand(ANSICommand{Type: ANSICarriageReturn})
	case 0x09: // TAB
		ap.flushTextBuffer()
		ap.addCommand(ANSICommand{Type: ANSITab})
	case 0x08: // BS
		ap.flushTextBuffer()
		ap.addCommand(ANSICommand{Type: ANSIBackspace})
	default:
		if b >= 0x20 && b <= 0x7E { // 인쇄 가능한 ASCII
			ap.addToTextBuffer(b)
		} else if b >= 0x80 { // UTF-8 멀티바이트
			ap.addToTextBuffer(b)
		}
		// 다른 제어 문자는 무시
	}
	return nil
}

// flushTextBuffer flushes accumulated text as a text command
func (ap *ANSIParser) flushTextBuffer() {
	if len(ap.textBuffer) > 0 {
		text := make([]byte, len(ap.textBuffer))
		copy(text, ap.textBuffer)
		ap.addCommand(ANSICommand{
			Type: ANSIText,
			Text: string(text),
		})
		ap.textBuffer = ap.textBuffer[:0]
	}
}

// addToTextBuffer adds a byte to the text buffer
func (ap *ANSIParser) addToTextBuffer(b byte) {
	ap.textBuffer = append(ap.textBuffer, b)
}

// addCommand adds a command to the command buffer
func (ap *ANSIParser) addCommand(cmd ANSICommand) {
	ap.commands = append(ap.commands, cmd)
}

// resetParser resets the parser to ground state
func (ap *ANSIParser) resetParser() {
	ap.state = StateGround
	ap.buffer = ap.buffer[:0]
}

// Reset resets the parser state completely
func (ap *ANSIParser) Reset() error {
	ap.mutex.Lock()
	defer ap.mutex.Unlock()

	ap.state = StateGround
	ap.buffer = ap.buffer[:0]
	ap.textBuffer = ap.textBuffer[:0]
	ap.commands = ap.commands[:0]

	if ap.config.TrackStatistics {
		ap.statistics.Reset()
	}

	return nil
}

// GetState returns the current parser state
func (ap *ANSIParser) GetState() ParserState {
	ap.mutex.RLock()
	defer ap.mutex.RUnlock()
	return ap.state
}

// GetStatistics returns parser statistics
func (ap *ANSIParser) GetStatistics() *ParserStatistics {
	if !ap.config.TrackStatistics {
		return nil
	}
	ap.mutex.RLock()
	defer ap.mutex.RUnlock()
	stats := *ap.statistics
	return &stats
}

// ParseStream creates a streaming parser that reads from an io.Reader
func (ap *ANSIParser) ParseStream(reader io.Reader) (<-chan ANSICommand, error) {
	// 스트리밍 파서 생성
	sp := NewStreamingParser(ap, &StreamConfig{
		BufferSize:     4096,
		ReadBufferSize: 8192,
	})

	return sp.StartStream(reader)
}

// String returns the string representation of a parser state
func (ps ParserState) String() string {
	switch ps {
	case StateGround:
		return "Ground"
	case StateEscape:
		return "Escape"
	case StateCSI:
		return "CSI"
	case StateOSC:
		return "OSC"
	case StateDCS:
		return "DCS"
	case StateString:
		return "String"
	case StateIgnore:
		return "Ignore"
	default:
		return fmt.Sprintf("Unknown(%d)", ps)
	}
}