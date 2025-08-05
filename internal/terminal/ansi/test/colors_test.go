package ansi_test

import (
	"testing"

	"github.com/aicli/aicli-web/internal/terminal/ansi"
	"github.com/stretchr/testify/assert"
)

func TestColorCreation(t *testing.T) {
	tests := []struct {
		name     string
		color    ansi.Color
		expected ansi.ColorType
	}{
		{
			name:     "Default color",
			color:    ansi.NewDefaultColor(),
			expected: ansi.ColorDefault,
		},
		{
			name:     "4-bit color",
			color:    ansi.New4BitColor(5),
			expected: ansi.Color4Bit,
		},
		{
			name:     "8-bit color",
			color:    ansi.New8BitColor(100),
			expected: ansi.Color8Bit,
		},
		{
			name:     "24-bit color",
			color:    ansi.New24BitColor(255, 128, 64),
			expected: ansi.Color24Bit,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.color.Type)
		})
	}
}

func TestColor4BitToRGB(t *testing.T) {
	tests := []struct {
		name     string
		index    uint8
		r, g, b  uint8
	}{
		{"Black", 0, 0, 0, 0},
		{"Red", 1, 170, 0, 0},
		{"Green", 2, 0, 170, 0},
		{"Yellow", 3, 170, 85, 0},
		{"Blue", 4, 0, 0, 170},
		{"Magenta", 5, 170, 0, 170},
		{"Cyan", 6, 0, 170, 170},
		{"White", 7, 170, 170, 170},
		{"Bright Black", 8, 85, 85, 85},
		{"Bright Red", 9, 255, 85, 85},
		{"Bright Green", 10, 85, 255, 85},
		{"Bright Yellow", 11, 255, 255, 85},
		{"Bright Blue", 12, 85, 85, 255},
		{"Bright Magenta", 13, 255, 85, 255},
		{"Bright Cyan", 14, 85, 255, 255},
		{"Bright White", 15, 255, 255, 255},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			color := ansi.New4BitColor(tt.index)
			r, g, b := color.ToRGB()
			assert.Equal(t, tt.r, r)
			assert.Equal(t, tt.g, g)
			assert.Equal(t, tt.b, b)
		})
	}
}

func TestColor8BitToRGB(t *testing.T) {
	tests := []struct {
		name    string
		index   uint8
		r, g, b uint8
	}{
		// 標準16色
		{"Standard Black", 0, 0, 0, 0},
		{"Standard Red", 1, 170, 0, 0},
		{"Standard White", 15, 255, 255, 255},

		// 216색キューブ
		{"Cube 16", 16, 0, 0, 0},     // 最初の色 (0,0,0)
		{"Cube 17", 17, 0, 0, 51},    // (0,0,1)
		{"Cube 21", 21, 0, 0, 255},   // (0,0,5)
		{"Cube 46", 46, 0, 255, 0},   // (0,5,0)
		{"Cube 196", 196, 255, 0, 0}, // (5,0,0)
		{"Cube 231", 231, 255, 255, 255}, // 最後の色 (5,5,5)

		// グレースケール
		{"Gray 232", 232, 8, 8, 8},
		{"Gray 243", 243, 118, 118, 118},
		{"Gray 255", 255, 238, 238, 238},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			color := ansi.New8BitColor(tt.index)
			r, g, b := color.ToRGB()
			assert.Equal(t, tt.r, r)
			assert.Equal(t, tt.g, g)
			assert.Equal(t, tt.b, b)
		})
	}
}

func TestColor24BitToRGB(t *testing.T) {
	tests := []struct {
		name    string
		r, g, b uint8
	}{
		{"Black", 0, 0, 0},
		{"White", 255, 255, 255},
		{"Red", 255, 0, 0},
		{"Green", 0, 255, 0},
		{"Blue", 0, 0, 255},
		{"Custom", 123, 45, 67},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			color := ansi.New24BitColor(tt.r, tt.g, tt.b)
			r, g, b := color.ToRGB()
			assert.Equal(t, tt.r, r)
			assert.Equal(t, tt.g, g)
			assert.Equal(t, tt.b, b)
		})
	}
}

func TestColorIsDefault(t *testing.T) {
	defaultColor := ansi.NewDefaultColor()
	assert.True(t, defaultColor.IsDefault())

	color4Bit := ansi.New4BitColor(1)
	assert.False(t, color4Bit.IsDefault())

	color8Bit := ansi.New8BitColor(100)
	assert.False(t, color8Bit.IsDefault())

	color24Bit := ansi.New24BitColor(255, 128, 64)
	assert.False(t, color24Bit.IsDefault())
}