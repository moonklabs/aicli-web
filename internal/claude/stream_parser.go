package claude

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	scanner   *bufio.Scanner
	decoder   *json.Decoder
	buffer    *bytes.Buffer
	mutex     sync.RWMutex
	logger    *logrus.Logger
	reader    io.Reader
}

// NewJSONStreamParser는 새로운 JSON 스트림 파서를 생성합니다.
func NewJSONStreamParser(reader io.Reader, logger *logrus.Logger) *JSONStreamParser {
	buffer := &bytes.Buffer{}
	return &JSONStreamParser{
		scanner: bufio.NewScanner(reader),
		decoder: json.NewDecoder(reader),
		buffer:  buffer,
		logger:  logger,
		reader:  reader,
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
		"buffer_size": p.buffer.Len(),
	}
}