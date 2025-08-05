package ansi_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/aicli/aicli-web/internal/terminal/ansi"
)

func BenchmarkParseSimpleText(b *testing.B) {
	parser := ansi.NewANSIParser()
	input := []byte("Hello, World! This is a simple text without any ANSI sequences.")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(input)
	}
}

func BenchmarkParseANSISequences(b *testing.B) {
	parser := ansi.NewANSIParser()
	input := []byte("\x1b[1;31mRed Bold\x1b[0m Normal \x1b[32mGreen\x1b[0m \x1b[2J\x1b[H")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(input)
	}
}

func BenchmarkParseComplexOutput(b *testing.B) {
	parser := ansi.NewANSIParser()
	
	// 실제 터미널 출력과 유사한 복잡한 시퀀스
	var buf bytes.Buffer
	for i := 0; i < 10; i++ {
		fmt.Fprintf(&buf, "\x1b[%d;1H", i+1) // 커서 위치
		fmt.Fprintf(&buf, "\x1b[K")          // 라인 지우기
		fmt.Fprintf(&buf, "\x1b[38;5;%dm", 16+i*10) // 8비트 색상
		fmt.Fprintf(&buf, "Line %d: ", i)
		fmt.Fprintf(&buf, "\x1b[1m")        // Bold
		fmt.Fprintf(&buf, "Important text")
		fmt.Fprintf(&buf, "\x1b[22m")       // Normal intensity
		fmt.Fprintf(&buf, " - Status: ")
		fmt.Fprintf(&buf, "\x1b[32mOK\x1b[0m\n") // Green OK
	}
	input := buf.Bytes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(input)
	}
}

func BenchmarkParseLargeText(b *testing.B) {
	parser := ansi.NewANSIParser()
	
	// 1MB의 텍스트 생성
	text := strings.Repeat("This is a line of text without ANSI sequences.\n", 20000)
	input := []byte(text)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(input)
	}
}

func BenchmarkParseColorSequences(b *testing.B) {
	parser := ansi.NewANSIParser()
	
	tests := []struct {
		name  string
		input string
	}{
		{"4-bit", "\x1b[31mRed\x1b[0m"},
		{"8-bit", "\x1b[38;5;196mRed\x1b[0m"},
		{"24-bit", "\x1b[38;2;255;0;0mRed\x1b[0m"},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			input := []byte(tt.input)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = parser.Parse(input)
			}
		})
	}
}

func BenchmarkStreamingParser(b *testing.B) {
	// 스트리밍 데이터 생성
	var buf bytes.Buffer
	for i := 0; i < 1000; i++ {
		fmt.Fprintf(&buf, "Line %d: \x1b[%dmColored text\x1b[0m\n", i, 31+(i%7))
	}
	data := buf.Bytes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser := ansi.NewANSIParser()
		reader := bytes.NewReader(data)
		cmdChan, _ := parser.ParseStream(reader)
		
		// 모든 명령어 소비
		for range cmdChan {
			// 처리
		}
	}
}

func BenchmarkRingBuffer(b *testing.B) {
	buffer := ansi.NewRingBuffer(4096)
	data := []byte("This is test data to write to the ring buffer.")

	b.Run("Write", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buffer.Write(data)
			buffer.Clear()
		}
	})

	b.Run("Read", func(b *testing.B) {
		buffer.Write(data)
		readBuf := make([]byte, len(data))
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buffer.Read(readBuf)
			buffer.Write(data)
		}
	})
}

func BenchmarkParseVimOutput(b *testing.B) {
	parser := ansi.NewANSIParser()
	
	// Vim과 유사한 출력 시뮬레이션
	var buf bytes.Buffer
	// 상태 바
	fmt.Fprintf(&buf, "\x1b[24;1H\x1b[7m-- INSERT --\x1b[0m")
	// 구문 강조된 코드
	for i := 0; i < 20; i++ {
		fmt.Fprintf(&buf, "\x1b[%d;1H", i+1)
		fmt.Fprintf(&buf, "\x1b[38;5;242m%3d \x1b[0m", i+1) // 라인 번호
		fmt.Fprintf(&buf, "\x1b[38;5;33mfunc\x1b[0m ")      // 키워드
		fmt.Fprintf(&buf, "\x1b[38;5;215mmain\x1b[0m() {\n") // 함수명
	}
	input := buf.Bytes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(input)
	}
}