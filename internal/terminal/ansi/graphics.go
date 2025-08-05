package ansi

// parseGraphicsParameters parses SGR (Set Graphics Rendition) parameters
func (ap *ANSIParser) parseGraphicsParameters(cmd *ANSICommand, params []int) {
	if len(params) == 0 {
		// 매개변수 없음 = 모든 속성 리셋
		cmd.Type = ANSIResetAttributes
		return
	}

	// 초기 속성 설정
	cmd.Attributes = TextAttributes{}
	cmd.Foreground = NewDefaultColor()
	cmd.Background = NewDefaultColor()
	
	// 기본적으로 SetGraphics 타입 유지
	hasForegroundChange := false
	hasBackgroundChange := false

	for i := 0; i < len(params); i++ {
		param := params[i]

		switch param {
		case 0: // Reset all
			cmd.Type = ANSIResetAttributes
			cmd.Attributes.Reset()
			cmd.Foreground = NewDefaultColor()
			cmd.Background = NewDefaultColor()

		// 텍스트 속성
		case 1: // Bold
			cmd.Attributes.Bold = true
		case 2: // Dim
			cmd.Attributes.Dim = true
		case 3: // Italic
			cmd.Attributes.Italic = true
		case 4: // Underline
			cmd.Attributes.Underline = true
		case 5: // Blink
			cmd.Attributes.Blink = true
		case 7: // Reverse
			cmd.Attributes.Reverse = true
		case 8: // Hidden
			cmd.Attributes.Hidden = true
		case 9: // Strikethrough
			cmd.Attributes.Strikethrough = true

		// 속성 해제
		case 21: // Not bold (double underline in some terminals)
			cmd.Attributes.Bold = false
		case 22: // Normal intensity (not bold, not dim)
			cmd.Attributes.Bold = false
			cmd.Attributes.Dim = false
		case 23: // Not italic
			cmd.Attributes.Italic = false
		case 24: // Not underlined
			cmd.Attributes.Underline = false
		case 25: // Not blinking
			cmd.Attributes.Blink = false
		case 27: // Not reversed
			cmd.Attributes.Reverse = false
		case 28: // Not hidden
			cmd.Attributes.Hidden = false
		case 29: // Not strikethrough
			cmd.Attributes.Strikethrough = false

		// 표준 전경색 (30-37)
		case 30, 31, 32, 33, 34, 35, 36, 37:
			cmd.Foreground = New4BitColor(uint8(param - 30))
			hasForegroundChange = true

		// 확장 전경색
		case 38:
			if i+1 < len(params) {
				fg, skip := ap.parseExtendedColor(params[i+1:])
				cmd.Foreground = fg
				hasForegroundChange = true
				i += skip
			}

		// 기본 전경색으로 리셋
		case 39:
			cmd.Foreground = NewDefaultColor()
			hasForegroundChange = true

		// 표준 배경색 (40-47)
		case 40, 41, 42, 43, 44, 45, 46, 47:
			cmd.Background = New4BitColor(uint8(param - 40))
			hasBackgroundChange = true

		// 확장 배경색
		case 48:
			if i+1 < len(params) {
				bg, skip := ap.parseExtendedColor(params[i+1:])
				cmd.Background = bg
				hasBackgroundChange = true
				i += skip
			}

		// 기본 배경색으로 리셋
		case 49:
			cmd.Background = NewDefaultColor()
			hasBackgroundChange = true

		// 밝은 전경색 (90-97)
		case 90, 91, 92, 93, 94, 95, 96, 97:
			cmd.Foreground = New4BitColor(uint8(param - 90 + 8))
			hasForegroundChange = true

		// 밝은 배경색 (100-107)
		case 100, 101, 102, 103, 104, 105, 106, 107:
			cmd.Background = New4BitColor(uint8(param - 100 + 8))
			hasBackgroundChange = true
		}
	}
	
	// 타입 결정
	if hasForegroundChange && !hasBackgroundChange {
		cmd.Type = ANSISetForeground
	} else if hasBackgroundChange && !hasForegroundChange {
		cmd.Type = ANSISetBackground
	}
	// 그 외의 경우는 기본 ANSISetGraphics 유지
}

// parseExtendedColor parses extended color sequences (8-bit and 24-bit RGB)
func (ap *ANSIParser) parseExtendedColor(params []int) (Color, int) {
	if len(params) == 0 {
		return NewDefaultColor(), 0
	}

	switch params[0] {
	case 5: // 8-bit color
		if len(params) >= 2 {
			return New8BitColor(uint8(params[1])), 2
		}
	case 2: // 24-bit RGB color
		if len(params) >= 4 {
			return New24BitColor(
				uint8(params[1]),
				uint8(params[2]),
				uint8(params[3]),
			), 4
		}
	}

	return NewDefaultColor(), 1
}