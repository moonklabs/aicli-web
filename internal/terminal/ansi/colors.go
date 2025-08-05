package ansi

// ColorType represents the type of color encoding
type ColorType int

const (
	ColorDefault ColorType = iota
	Color4Bit
	Color8Bit
	Color24Bit
)

// Color represents a terminal color
type Color struct {
	Type    ColorType
	R, G, B uint8
	Index   uint8
}

// Standard 4-bit color indices
const (
	Black = iota
	Red
	Green
	Yellow
	Blue
	Magenta
	Cyan
	White
	BrightBlack
	BrightRed
	BrightGreen
	BrightYellow
	BrightBlue
	BrightMagenta
	BrightCyan
	BrightWhite
)

// NewDefaultColor creates a default color
func NewDefaultColor() Color {
	return Color{Type: ColorDefault}
}

// New4BitColor creates a 4-bit color from an index
func New4BitColor(index uint8) Color {
	return Color{
		Type:  Color4Bit,
		Index: index,
	}
}

// New8BitColor creates an 8-bit color from an index
func New8BitColor(index uint8) Color {
	return Color{
		Type:  Color8Bit,
		Index: index,
	}
}

// New24BitColor creates a 24-bit RGB color
func New24BitColor(r, g, b uint8) Color {
	return Color{
		Type: Color24Bit,
		R:    r,
		G:    g,
		B:    b,
	}
}

// ToRGB converts any color type to RGB values
func (c Color) ToRGB() (r, g, b uint8) {
	switch c.Type {
	case Color4Bit:
		return convert4BitToRGB(c.Index)
	case Color8Bit:
		return convert8BitToRGB(c.Index)
	case Color24Bit:
		return c.R, c.G, c.B
	default:
		return 0, 0, 0
	}
}

// convert4BitToRGB converts a 4-bit color index to RGB
func convert4BitToRGB(index uint8) (r, g, b uint8) {
	// 표준 16색 팔레트
	colors := []struct{ r, g, b uint8 }{
		{0, 0, 0},       // Black
		{170, 0, 0},     // Red
		{0, 170, 0},     // Green
		{170, 85, 0},    // Yellow
		{0, 0, 170},     // Blue
		{170, 0, 170},   // Magenta
		{0, 170, 170},   // Cyan
		{170, 170, 170}, // White
		{85, 85, 85},    // Bright Black
		{255, 85, 85},   // Bright Red
		{85, 255, 85},   // Bright Green
		{255, 255, 85},  // Bright Yellow
		{85, 85, 255},   // Bright Blue
		{255, 85, 255},  // Bright Magenta
		{85, 255, 255},  // Bright Cyan
		{255, 255, 255}, // Bright White
	}

	if index < 16 {
		c := colors[index]
		return c.r, c.g, c.b
	}
	return 0, 0, 0
}

// convert8BitToRGB converts an 8-bit color index to RGB
func convert8BitToRGB(index uint8) (r, g, b uint8) {
	if index < 16 {
		// 표준 16색
		return convert4BitToRGB(index)
	} else if index < 232 {
		// 216색 큐브 (6x6x6)
		index -= 16
		r = (index / 36) * 51
		g = ((index / 6) % 6) * 51
		b = (index % 6) * 51
		return r, g, b
	} else {
		// 24개 그레이스케일
		gray := 8 + (index-232)*10
		return gray, gray, gray
	}
}

// IsDefault returns true if this is the default color
func (c Color) IsDefault() bool {
	return c.Type == ColorDefault
}