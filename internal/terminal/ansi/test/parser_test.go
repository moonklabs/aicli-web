package ansi_test

import (
	"bytes"
	"testing"

	"github.com/aicli/aicli-web/internal/terminal/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewANSIParser(t *testing.T) {
	parser := ansi.NewANSIParser()
	assert.NotNil(t, parser)
	assert.Equal(t, ansi.StateGround, parser.GetState())
}

func TestParseBasicText(t *testing.T) {
	parser := ansi.NewANSIParser()

	tests := []struct {
		name     string
		input    string
		expected []ansi.ANSICommand
	}{
		{
			name:  "Simple text",
			input: "Hello, World!",
			expected: []ansi.ANSICommand{
				{Type: ansi.ANSIText, Text: "Hello, World!"},
			},
		},
		{
			name:  "Text with newline",
			input: "Hello\nWorld",
			expected: []ansi.ANSICommand{
				{Type: ansi.ANSIText, Text: "Hello"},
				{Type: ansi.ANSILineFeed},
				{Type: ansi.ANSIText, Text: "World"},
			},
		},
		{
			name:  "Text with carriage return",
			input: "Hello\rWorld",
			expected: []ansi.ANSICommand{
				{Type: ansi.ANSIText, Text: "Hello"},
				{Type: ansi.ANSICarriageReturn},
				{Type: ansi.ANSIText, Text: "World"},
			},
		},
		{
			name:  "Text with tab",
			input: "Hello\tWorld",
			expected: []ansi.ANSICommand{
				{Type: ansi.ANSIText, Text: "Hello"},
				{Type: ansi.ANSITab},
				{Type: ansi.ANSIText, Text: "World"},
			},
		},
		{
			name:  "Text with backspace",
			input: "Hello\bWorld",
			expected: []ansi.ANSICommand{
				{Type: ansi.ANSIText, Text: "Hello"},
				{Type: ansi.ANSIBackspace},
				{Type: ansi.ANSIText, Text: "World"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commands, err := parser.Parse([]byte(tt.input))
			require.NoError(t, err)
			assert.Equal(t, len(tt.expected), len(commands))

			for i, cmd := range commands {
				assert.Equal(t, tt.expected[i].Type, cmd.Type)
				if cmd.Type == ansi.ANSIText {
					assert.Equal(t, tt.expected[i].Text, cmd.Text)
				}
			}
		})
	}
}

func TestParseCursorMovement(t *testing.T) {
	parser := ansi.NewANSIParser()

	tests := []struct {
		name     string
		input    string
		expected ansi.ANSICommand
	}{
		{
			name:     "Cursor up",
			input:    "\x1b[A",
			expected: ansi.ANSICommand{Type: ansi.ANSICursorUp, Y: 1},
		},
		{
			name:     "Cursor up 5",
			input:    "\x1b[5A",
			expected: ansi.ANSICommand{Type: ansi.ANSICursorUp, Y: 5},
		},
		{
			name:     "Cursor down",
			input:    "\x1b[B",
			expected: ansi.ANSICommand{Type: ansi.ANSICursorDown, Y: 1},
		},
		{
			name:     "Cursor forward",
			input:    "\x1b[C",
			expected: ansi.ANSICommand{Type: ansi.ANSICursorForward, X: 1},
		},
		{
			name:     "Cursor back",
			input:    "\x1b[D",
			expected: ansi.ANSICommand{Type: ansi.ANSICursorBack, X: 1},
		},
		{
			name:     "Cursor position",
			input:    "\x1b[10;20H",
			expected: ansi.ANSICommand{Type: ansi.ANSICursorPosition, X: 20, Y: 10},
		},
		{
			name:     "Cursor position default",
			input:    "\x1b[H",
			expected: ansi.ANSICommand{Type: ansi.ANSICursorPosition, X: 1, Y: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commands, err := parser.Parse([]byte(tt.input))
			require.NoError(t, err)
			require.Len(t, commands, 1)

			cmd := commands[0]
			assert.Equal(t, tt.expected.Type, cmd.Type)
			assert.Equal(t, tt.expected.X, cmd.X)
			assert.Equal(t, tt.expected.Y, cmd.Y)
		})
	}
}

func TestParseScreenManipulation(t *testing.T) {
	parser := ansi.NewANSIParser()

	tests := []struct {
		name     string
		input    string
		expected ansi.ANSICommand
	}{
		{
			name:     "Clear screen",
			input:    "\x1b[2J",
			expected: ansi.ANSICommand{Type: ansi.ANSIClearScreen, Parameters: []int{2}},
		},
		{
			name:     "Clear screen from cursor down",
			input:    "\x1b[J",
			expected: ansi.ANSICommand{Type: ansi.ANSIClearScreen, Parameters: []int{0}},
		},
		{
			name:     "Clear line",
			input:    "\x1b[K",
			expected: ansi.ANSICommand{Type: ansi.ANSIClearLine, Parameters: []int{0}},
		},
		{
			name:     "Clear entire line",
			input:    "\x1b[2K",
			expected: ansi.ANSICommand{Type: ansi.ANSIClearLine, Parameters: []int{2}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commands, err := parser.Parse([]byte(tt.input))
			require.NoError(t, err)
			require.Len(t, commands, 1)

			cmd := commands[0]
			assert.Equal(t, tt.expected.Type, cmd.Type)
			assert.Equal(t, tt.expected.Parameters, cmd.Parameters)
		})
	}
}

func TestParseColors(t *testing.T) {
	parser := ansi.NewANSIParser()

	tests := []struct {
		name           string
		input          string
		expectedFG     ansi.Color
		expectedBG     ansi.Color
		expectedAttrs  ansi.TextAttributes
	}{
		{
			name:       "Red foreground",
			input:      "\x1b[31m",
			expectedFG: ansi.New4BitColor(1),
		},
		{
			name:       "Green background",
			input:      "\x1b[42m",
			expectedBG: ansi.New4BitColor(2),
		},
		{
			name:       "Bold text",
			input:      "\x1b[1m",
			expectedAttrs: ansi.TextAttributes{Bold: true},
		},
		{
			name:       "Multiple attributes",
			input:      "\x1b[1;4;31m",
			expectedFG: ansi.New4BitColor(1),
			expectedAttrs: ansi.TextAttributes{Bold: true, Underline: true},
		},
		{
			name:       "8-bit color",
			input:      "\x1b[38;5;124m",
			expectedFG: ansi.New8BitColor(124),
		},
		{
			name:       "24-bit RGB color",
			input:      "\x1b[38;2;255;128;64m",
			expectedFG: ansi.New24BitColor(255, 128, 64),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commands, err := parser.Parse([]byte(tt.input))
			require.NoError(t, err)
			require.Len(t, commands, 1)

			cmd := commands[0]
			// 색상만 변경된 경우 타입이 ANSISetForeground 또는 ANSISetBackground일 수 있음
			if cmd.Type != ansi.ANSISetGraphics && cmd.Type != ansi.ANSISetForeground && cmd.Type != ansi.ANSISetBackground {
				t.Errorf("Expected command type to be ANSISetGraphics, ANSISetForeground, or ANSISetBackground, got %v", cmd.Type)
			}

			if tt.expectedFG.Type != ansi.ColorDefault {
				assert.Equal(t, tt.expectedFG, cmd.Foreground)
			}
			if tt.expectedBG.Type != ansi.ColorDefault {
				assert.Equal(t, tt.expectedBG, cmd.Background)
			}
			if tt.expectedAttrs != (ansi.TextAttributes{}) {
				assert.Equal(t, tt.expectedAttrs, cmd.Attributes)
			}
		})
	}
}

func TestComplexSequences(t *testing.T) {
	parser := ansi.NewANSIParser()

	// 복잡한 시퀀스 테스트
	input := "Hello \x1b[1;31mRed Bold\x1b[0m Normal \x1b[2J\x1b[H"
	commands, err := parser.Parse([]byte(input))
	require.NoError(t, err)

	expected := []ansi.CommandType{
		ansi.ANSIText,           // "Hello "
		ansi.ANSISetForeground,  // Bold + Red (색상 변경이 있으므로 SetForeground)
		ansi.ANSIText,           // "Red Bold"
		ansi.ANSIResetAttributes, // Reset
		ansi.ANSIText,           // " Normal "
		ansi.ANSIClearScreen,    // Clear screen
		ansi.ANSICursorPosition, // Home
	}

	require.Len(t, commands, len(expected))
	for i, cmd := range commands {
		assert.Equal(t, expected[i], cmd.Type)
	}
}

func TestStreamingParser(t *testing.T) {
	parser := ansi.NewANSIParser()
	input := bytes.NewReader([]byte("Hello\x1b[31mRed\x1b[0mNormal"))

	cmdChan, err := parser.ParseStream(input)
	require.NoError(t, err)

	var commands []ansi.ANSICommand
	for cmd := range cmdChan {
		commands = append(commands, cmd)
	}

	assert.Len(t, commands, 5) // "Hello", SetGraphics, "Red", Reset, "Normal"
}

func TestParserReset(t *testing.T) {
	parser := ansi.NewANSIParser()

	// 파싱 수행
	_, err := parser.Parse([]byte("Hello\x1b[31m"))
	require.NoError(t, err)

	// 리셋
	err = parser.Reset()
	require.NoError(t, err)

	// 상태 확인
	assert.Equal(t, ansi.StateGround, parser.GetState())

	// 새로운 파싱
	commands, err := parser.Parse([]byte("World"))
	require.NoError(t, err)
	assert.Len(t, commands, 1)
	assert.Equal(t, "World", commands[0].Text)
}

func TestUTF8Support(t *testing.T) {
	parser := ansi.NewANSIParser()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Korean text",
			input:    "안녕하세요",
			expected: "안녕하세요",
		},
		{
			name:     "Japanese text",
			input:    "こんにちは",
			expected: "こんにちは",
		},
		{
			name:     "Emoji",
			input:    "Hello 👋 World",
			expected: "Hello 👋 World",
		},
		{
			name:     "Mixed with ANSI",
			input:    "안녕\x1b[31m하세요\x1b[0m",
			expected: "안녕하세요",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commands, err := parser.Parse([]byte(tt.input))
			require.NoError(t, err)

			var text string
			for _, cmd := range commands {
				if cmd.Type == ansi.ANSIText {
					text += cmd.Text
				}
			}
			assert.Equal(t, tt.expected, text)
		})
	}
}