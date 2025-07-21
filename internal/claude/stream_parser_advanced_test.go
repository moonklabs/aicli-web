package claude

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONStreamParserAdvanced(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	t.Run("Multiline JSON Parsing", func(t *testing.T) {
		multilineJSON := `{
  "type": "text",
  "content": "This is a multiline
JSON message with line breaks",
  "message_id": "multi-123",
  "metadata": {
    "timestamp": "2024-01-01T00:00:00Z",
    "tags": ["tag1", "tag2"]
  }
}`

		reader := strings.NewReader(multilineJSON)
		parser := NewJSONStreamParser(reader, logger)

		// 라인별로 파싱
		var response *Response
		var err error
		
		for response == nil && err == nil {
			response, err = parser.ParseLineAdvanced()
		}

		require.NoError(t, err)
		require.NotNil(t, response)
		assert.Equal(t, "text", response.Type)
		assert.Contains(t, response.Content, "multiline")
		assert.Equal(t, "multi-123", response.MessageID)
	})

	t.Run("Mixed Single and Multiline", func(t *testing.T) {
		mixedJSON := `{"type":"simple","content":"Single line","message_id":"single-1"}
{
  "type": "complex",
  "content": "Multi line",
  "message_id": "multi-1"
}
{"type":"simple","content":"Another single","message_id":"single-2"}
{
  "type": "nested",
  "content": "Nested object",
  "metadata": {
    "nested": {
      "deep": "value"
    }
  },
  "message_id": "nested-1"
}`

		reader := strings.NewReader(mixedJSON)
		parser := NewJSONStreamParser(reader, logger)

		responses := make([]*Response, 0)
		
		for {
			resp, err := parser.ParseLineAdvanced()
			if err == io.EOF {
				break
			}
			require.NoError(t, err)
			if resp != nil {
				responses = append(responses, resp)
			}
		}

		assert.Len(t, responses, 4)
		assert.Equal(t, "single-1", responses[0].MessageID)
		assert.Equal(t, "multi-1", responses[1].MessageID)
		assert.Equal(t, "single-2", responses[2].MessageID)
		assert.Equal(t, "nested-1", responses[3].MessageID)
	})

	t.Run("Partial JSON Buffer", func(t *testing.T) {
		// 불완전한 JSON으로 시작
		partialJSON := `{
  "type": "partial",
  "content": "This JSON`

		reader := strings.NewReader(partialJSON)
		parser := NewJSONStreamParser(reader, logger)

		// 여러 번 파싱 시도
		for i := 0; i < 5; i++ {
			resp, err := parser.ParseLineAdvanced()
			if err == io.EOF {
				break
			}
			assert.NoError(t, err)
			assert.Nil(t, resp) // 아직 완성되지 않음
		}

		// 파서 상태 확인
		stats := parser.GetStats()
		assert.True(t, stats["in_multiline_json"].(bool))
		assert.Greater(t, stats["partial_buffer_size"].(int), 0)
	})

	t.Run("Complex Nested JSON", func(t *testing.T) {
		complexJSON := `{
  "type": "complex",
  "content": "Complex message",
  "metadata": {
    "arrays": [
      {"id": 1, "name": "item1"},
      {"id": 2, "name": "item2"}
    ],
    "nested": {
      "level1": {
        "level2": {
          "value": "deep"
        }
      }
    }
  },
  "message_id": "complex-123"
}`

		reader := strings.NewReader(complexJSON)
		parser := NewJSONStreamParser(reader, logger)

		var response *Response
		var lineCount int
		
		for response == nil {
			resp, err := parser.ParseLineAdvanced()
			if err == io.EOF {
				break
			}
			require.NoError(t, err)
			lineCount++
			if resp != nil {
				response = resp
			}
		}

		require.NotNil(t, response)
		assert.Equal(t, "complex", response.Type)
		assert.Equal(t, "complex-123", response.MessageID)
		assert.Greater(t, lineCount, 10) // 멀티라인이므로 여러 번 호출됨
	})

	t.Run("Error Recovery", func(t *testing.T) {
		// 잘못된 JSON 포함
		mixedData := `{"type":"valid","content":"Valid JSON","message_id":"valid-1"}
This is not JSON at all
{invalid json}
{"type":"valid","content":"Another valid","message_id":"valid-2"}
{
  "type": "multiline",
  "content": "Valid multiline",
  "message_id": "multi-1"
}`

		reader := strings.NewReader(mixedData)
		parser := NewJSONStreamParser(reader, logger)

		validResponses := make([]*Response, 0)
		errorCount := 0

		for {
			resp, err := parser.ParseLineAdvanced()
			if err == io.EOF {
				break
			}
			
			if err != nil {
				errorCount++
				parser.RecoverFromError()
				continue
			}
			
			if resp != nil {
				validResponses = append(validResponses, resp)
			}
		}

		assert.Len(t, validResponses, 3)
		assert.Greater(t, errorCount, 0)
	})

	t.Run("Empty Lines and Whitespace", func(t *testing.T) {
		dataWithSpaces := `

{"type":"first","content":"First message","message_id":"first-1"}

   

{"type":"second","content":"Second message","message_id":"second-1"}

`

		reader := strings.NewReader(dataWithSpaces)
		parser := NewJSONStreamParser(reader, logger)

		responses := make([]*Response, 0)
		
		for {
			resp, err := parser.ParseLineAdvanced()
			if err == io.EOF {
				break
			}
			require.NoError(t, err)
			if resp != nil {
				responses = append(responses, resp)
			}
		}

		assert.Len(t, responses, 2)
		assert.Equal(t, "first-1", responses[0].MessageID)
		assert.Equal(t, "second-1", responses[1].MessageID)
	})

	t.Run("Bracket Counting", func(t *testing.T) {
		// 배열을 포함한 JSON
		arrayJSON := `[
  {
    "type": "array_item",
    "content": "Item 1",
    "message_id": "item-1"
  },
  {
    "type": "array_item",
    "content": "Item 2",
    "message_id": "item-2"
  }
]`

		reader := strings.NewReader(arrayJSON)
		parser := NewJSONStreamParser(reader, logger)

		// 전체 배열을 하나의 응답으로 파싱 시도
		var completed bool
		for !completed {
			_, err := parser.ParseLineAdvanced()
			if err == io.EOF {
				break
			}
			
			stats := parser.GetStats()
			// 대괄호가 모두 닫혔는지 확인
			if stats["bracket_count"].(int) == 0 && stats["partial_buffer_size"].(int) > 0 {
				completed = true
			}
		}

		// 파서가 멀티라인 JSON을 추적했는지 확인
		stats := parser.GetStats()
		assert.False(t, stats["in_multiline_json"].(bool))
	})

	t.Run("Large Buffer Handling", func(t *testing.T) {
		// 큰 content를 가진 JSON
		largeContent := strings.Repeat("x", 100000) // 100KB
		largeJSON := `{
  "type": "large",
  "content": "` + largeContent + `",
  "message_id": "large-1"
}`

		reader := strings.NewReader(largeJSON)
		parser := NewJSONStreamParser(reader, logger)

		var response *Response
		for response == nil {
			resp, err := parser.ParseLineAdvanced()
			if err == io.EOF {
				break
			}
			require.NoError(t, err)
			if resp != nil {
				response = resp
			}
		}

		require.NotNil(t, response)
		assert.Equal(t, "large", response.Type)
		assert.Len(t, response.Content, 100000)
	})

	t.Run("Concurrent Parsing", func(t *testing.T) {
		// 동시에 여러 파서 실행
		jsonData := `{"type":"concurrent","content":"Message","message_id":"concurrent-1"}`
		
		numParsers := 10
		results := make(chan *Response, numParsers)
		errors := make(chan error, numParsers)

		for i := 0; i < numParsers; i++ {
			go func(id int) {
				reader := strings.NewReader(jsonData)
				parser := NewJSONStreamParser(reader, logger)
				
				resp, err := parser.ParseLineAdvanced()
				if err != nil && err != io.EOF {
					errors <- err
				} else if resp != nil {
					results <- resp
				}
			}(i)
		}

		// 결과 수집
		for i := 0; i < numParsers; i++ {
			select {
			case resp := <-results:
				assert.Equal(t, "concurrent", resp.Type)
			case err := <-errors:
				t.Errorf("Parser error: %v", err)
			}
		}
	})

	t.Run("Stream Context Cancellation", func(t *testing.T) {
		// 무한 스트림 시뮬레이션
		infiniteReader := &infiniteJSONReader{
			template: `{"type":"stream","content":"Message %d","message_id":"stream-%d"}`,
			delay:    10, // 10ms 지연
		}

		parser := NewJSONStreamParser(infiniteReader, logger)
		ctx, cancel := context.WithCancel(context.Background())

		responseChan, errorChan := parser.ParseStream(ctx)
		
		count := 0
		go func() {
			for range responseChan {
				count++
				if count >= 5 {
					cancel() // 5개 메시지 후 취소
				}
			}
		}()

		// 에러 채널 모니터링
		select {
		case <-errorChan:
			// 정상 종료
		case <-time.After(1 * time.Second):
			t.Error("Context cancellation timeout")
		}

		assert.GreaterOrEqual(t, count, 5)
	})
}

// infiniteJSONReader는 무한히 JSON을 생성하는 리더입니다.
type infiniteJSONReader struct {
	template string
	counter  int
	delay    int
	buffer   bytes.Buffer
}

func (r *infiniteJSONReader) Read(p []byte) (n int, err error) {
	if r.buffer.Len() == 0 {
		// 새 JSON 생성
		json := fmt.Sprintf(r.template, r.counter, r.counter) + "\n"
		r.buffer.WriteString(json)
		r.counter++
		
		if r.delay > 0 {
			time.Sleep(time.Duration(r.delay) * time.Millisecond)
		}
	}
	
	return r.buffer.Read(p)
}

func BenchmarkJSONParserAdvanced(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	b.Run("SingleLinePerformance", func(b *testing.B) {
		json := `{"type":"bench","content":"Benchmark message","message_id":"bench-123"}`
		data := strings.Repeat(json+"\n", b.N)
		reader := strings.NewReader(data)
		parser := NewJSONStreamParser(reader, logger)

		b.ResetTimer()
		
		count := 0
		for count < b.N {
			resp, err := parser.ParseLineAdvanced()
			if err == io.EOF {
				break
			}
			if err != nil {
				b.Fatal(err)
			}
			if resp != nil {
				count++
			}
		}
	})

	b.Run("MultilinePerformance", func(b *testing.B) {
		json := `{
  "type": "bench",
  "content": "Benchmark message",
  "metadata": {
    "field1": "value1",
    "field2": "value2"
  },
  "message_id": "bench-123"
}`
		data := strings.Repeat(json+"\n", b.N)
		reader := strings.NewReader(data)
		parser := NewJSONStreamParser(reader, logger)

		b.ResetTimer()
		
		count := 0
		for count < b.N {
			resp, err := parser.ParseLineAdvanced()
			if err == io.EOF {
				break
			}
			if err != nil {
				b.Fatal(err)
			}
			if resp != nil {
				count++
			}
		}
	})
}