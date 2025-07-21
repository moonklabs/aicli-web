package claude

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Message는 Claude CLI로 전송할 메시지를 나타냅니다.
type Message struct {
	Type    string                 `json:"type"`
	Content string                 `json:"content"`
	Meta    map[string]interface{} `json:"meta,omitempty"`
	ID      string                 `json:"id"`
}

// Response는 Claude CLI로부터 받은 응답을 나타냅니다.
type Response struct {
	Type      string                 `json:"type"`
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	MessageID string                 `json:"message_id,omitempty"`
	Error     *StreamError          `json:"error,omitempty"`
}

// StreamError는 스트림 처리 중 발생한 오류를 나타냅니다.
type StreamError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Code    int    `json:"code,omitempty"`
}

func (e *StreamError) Error() string {
	return fmt.Sprintf("stream error [%s]: %s", e.Type, e.Message)
}

// JSONStreamParser는 JSON 스트림을 파싱하는 구조체입니다.
type JSONStreamParser struct {
	scanner     *bufio.Scanner
	decoder     *json.Decoder
	buffer      *bytes.Buffer
	mutex       sync.RWMutex
	logger      *logrus.Logger
	reader      io.Reader
	maxLineSize int
	
	// 부분 JSON 처리를 위한 필드
	partialBuffer   *bytes.Buffer
	inMultilineJSON bool
	braceCount      int
	bracketCount    int
}

// NewJSONStreamParser는 새로운 JSON 스트림 파서를 생성합니다.
func NewJSONStreamParser(reader io.Reader, logger *logrus.Logger) *JSONStreamParser {
	buffer := &bytes.Buffer{}
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // 최대 1MB 라인
	
	return &JSONStreamParser{
		scanner:       scanner,
		decoder:       json.NewDecoder(reader),
		buffer:        buffer,
		logger:        logger,
		reader:        reader,
		maxLineSize:   1024 * 1024, // 1MB
		partialBuffer: &bytes.Buffer{},
	}
}

// ParseNext는 스트림에서 다음 JSON 객체를 파싱합니다.
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

// ParseStream은 스트림을 지속적으로 파싱하여 채널로 응답을 전송합니다.
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

// ParseLine은 한 줄씩 읽어서 JSON 객체를 파싱합니다.
// 스트림에서 개행으로 구분된 JSON 객체들을 처리할 때 유용합니다.
func (p *JSONStreamParser) ParseLine() (*Response, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.scanner.Scan() {
		if err := p.scanner.Err(); err != nil {
			return nil, fmt.Errorf("scanner error: %w", err)
		}
		return nil, io.EOF
	}

	line := p.scanner.Text()
	if line == "" {
		return nil, fmt.Errorf("empty line")
	}

	var response Response
	if err := json.Unmarshal([]byte(line), &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON line: %w", err)
	}

	p.logger.WithFields(logrus.Fields{
		"type":       response.Type,
		"message_id": response.MessageID,
		"line":       line,
	}).Debug("Parsed JSON line")

	return &response, nil
}

// ParseLineStream은 라인 기반 스트림을 지속적으로 파싱합니다.
func (p *JSONStreamParser) ParseLineStream(ctx context.Context) (<-chan *Response, <-chan error) {
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
				response, err := p.ParseLine()
				if err != nil {
					if err == io.EOF {
						return
					}
					// 빈 라인이나 파싱 오류는 로깅만 하고 계속 진행
					p.logger.WithError(err).Debug("Failed to parse line, continuing")
					continue
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

// Reset은 파서를 리셋합니다.
func (p *JSONStreamParser) Reset(reader io.Reader) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.reader = reader
	p.scanner = bufio.NewScanner(reader)
	p.decoder = json.NewDecoder(reader)
	p.buffer.Reset()
}

// GetStats는 파서의 통계 정보를 반환합니다.
func (p *JSONStreamParser) GetStats() map[string]interface{} {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return map[string]interface{}{
		"buffer_size":        p.buffer.Len(),
		"partial_buffer_size": p.partialBuffer.Len(),
		"in_multiline_json":  p.inMultilineJSON,
		"brace_count":        p.braceCount,
		"bracket_count":      p.bracketCount,
	}
}

// ParseMultilineJSON은 멀티라인 JSON을 파싱합니다.
func (p *JSONStreamParser) ParseMultilineJSON(line string) (*Response, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// 라인을 부분 버퍼에 추가
	p.partialBuffer.WriteString(line)
	p.partialBuffer.WriteString("\n")

	// 중괄호와 대괄호 카운트
	for _, ch := range line {
		switch ch {
		case '{':
			p.braceCount++
		case '}':
			p.braceCount--
		case '[':
			p.bracketCount++
		case ']':
			p.bracketCount--
		}
	}

	// JSON이 완성되었는지 확인
	if p.braceCount == 0 && p.bracketCount == 0 && p.partialBuffer.Len() > 0 {
		// 완성된 JSON 파싱
		var response Response
		if err := json.Unmarshal(p.partialBuffer.Bytes(), &response); err != nil {
			// 파싱 실패 시 버퍼 리셋
			p.partialBuffer.Reset()
			p.inMultilineJSON = false
			return nil, fmt.Errorf("failed to unmarshal multiline JSON: %w", err)
		}

		// 성공적으로 파싱됨
		p.partialBuffer.Reset()
		p.inMultilineJSON = false
		
		p.logger.WithFields(logrus.Fields{
			"type":       response.Type,
			"message_id": response.MessageID,
		}).Debug("Parsed multiline JSON response")

		return &response, nil
	}

	// 아직 JSON이 완성되지 않음
	p.inMultilineJSON = true
	return nil, nil
}

// IsValidJSONStart는 라인이 JSON 객체의 시작인지 확인합니다.
func (p *JSONStreamParser) IsValidJSONStart(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[")
}

// ParseLineAdvanced는 향상된 라인 파싱을 수행합니다.
func (p *JSONStreamParser) ParseLineAdvanced() (*Response, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.scanner.Scan() {
		if err := p.scanner.Err(); err != nil {
			return nil, fmt.Errorf("scanner error: %w", err)
		}
		return nil, io.EOF
	}

	line := p.scanner.Text()
	if line == "" {
		return nil, nil // 빈 라인 무시
	}

	// 멀티라인 JSON 처리 중인 경우
	if p.inMultilineJSON {
		return p.ParseMultilineJSON(line)
	}

	// 단일 라인 JSON 시도
	var response Response
	if err := json.Unmarshal([]byte(line), &response); err == nil {
		// 성공적으로 파싱됨
		p.logger.WithFields(logrus.Fields{
			"type":       response.Type,
			"message_id": response.MessageID,
		}).Debug("Parsed single line JSON")
		return &response, nil
	}

	// JSON 시작인 경우 멀티라인 처리 시작
	if p.IsValidJSONStart(line) {
		return p.ParseMultilineJSON(line)
	}

	// JSON이 아닌 라인은 무시
	p.logger.WithField("line", line).Debug("Ignoring non-JSON line")
	return nil, nil
}

// RecoverFromError는 파싱 에러로부터 복구를 시도합니다.
func (p *JSONStreamParser) RecoverFromError() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.partialBuffer.Reset()
	p.inMultilineJSON = false
	p.braceCount = 0
	p.bracketCount = 0
	
	p.logger.Info("Parser recovered from error state")
}