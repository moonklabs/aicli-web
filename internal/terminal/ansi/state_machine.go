package ansi

import (
	"fmt"
	"strconv"
	"strings"
)

// processEscape handles ESC sequence processing
func (ap *ANSIParser) processEscape(b byte) error {
	ap.buffer = append(ap.buffer, b)

	switch b {
	case '[': // CSI 시작
		ap.state = StateCSI
	case ']': // OSC 시작
		ap.state = StateOSC
	case 'P': // DCS 시작
		ap.state = StateDCS
	case '7': // DECSC - Save Cursor
		ap.addCommand(ANSICommand{Type: ANSISaveCursor, Raw: copyBytes(ap.buffer)})
		ap.resetParser()
	case '8': // DECRC - Restore Cursor
		ap.addCommand(ANSICommand{Type: ANSIRestoreCursor, Raw: copyBytes(ap.buffer)})
		ap.resetParser()
	case 'c': // RIS - Reset to Initial State
		ap.addCommand(ANSICommand{Type: ANSIResetAttributes, Raw: copyBytes(ap.buffer)})
		ap.resetParser()
	case 'D': // IND - Index (line feed)
		ap.addCommand(ANSICommand{Type: ANSILineFeed, Raw: copyBytes(ap.buffer)})
		ap.resetParser()
	case 'M': // RI - Reverse Index
		ap.addCommand(ANSICommand{Type: ANSIScrollDown, Raw: copyBytes(ap.buffer)})
		ap.resetParser()
	case 'E': // NEL - Next Line
		ap.addCommand(ANSICommand{Type: ANSILineFeed, Raw: copyBytes(ap.buffer)})
		ap.addCommand(ANSICommand{Type: ANSICarriageReturn})
		ap.resetParser()
	case '\\': // ST - String Terminator
		ap.resetParser()
	default:
		// 알려지지 않은 ESC 시퀀스는 무시하고 ground 상태로
		if ap.config.StrictMode {
			return fmt.Errorf("unknown escape sequence: ESC %c", b)
		}
		ap.resetParser()
	}

	return nil
}

// processCSI handles CSI (Control Sequence Introducer) sequences
func (ap *ANSIParser) processCSI(b byte) error {
	ap.buffer = append(ap.buffer, b)

	// 매개변수 수집 중
	if (b >= '0' && b <= '9') || b == ';' || b == '?' || b == ' ' || b == '!' || b == '>' || b == '<' {
		return nil
	}

	// 중간 바이트 (intermediate bytes)
	if b >= 0x20 && b <= 0x2F {
		return nil
	}

	// 종료 문자 도착 - 명령어 파싱
	if b >= '@' && b <= '~' {
		return ap.parseCSICommand(b)
	}

	// 잘못된 문자 - 시퀀스 무시
	if ap.config.StrictMode {
		return fmt.Errorf("invalid CSI sequence character: %c", b)
	}
	ap.resetParser()
	return nil
}

// parseCSICommand parses a complete CSI command
func (ap *ANSIParser) parseCSICommand(finalByte byte) error {
	// 매개변수 파싱
	paramStart := 2 // "ESC[" 이후부터
	paramEnd := len(ap.buffer) - 1

	// Private mode indicators 확인
	decMode := false
	if paramStart < len(ap.buffer) && ap.buffer[paramStart] == '?' {
		decMode = true
		paramStart++
	}

	paramStr := string(ap.buffer[paramStart:paramEnd])
	params := ap.parseParameters(paramStr)

	var cmd ANSICommand
	cmd.Parameters = params
	cmd.Raw = copyBytes(ap.buffer)

	switch finalByte {
	case 'A': // CUU - Cursor Up
		cmd.Type = ANSICursorUp
		cmd.Y = getParamOrDefault(params, 0, 1)
	case 'B': // CUD - Cursor Down
		cmd.Type = ANSICursorDown
		cmd.Y = getParamOrDefault(params, 0, 1)
	case 'C': // CUF - Cursor Forward
		cmd.Type = ANSICursorForward
		cmd.X = getParamOrDefault(params, 0, 1)
	case 'D': // CUB - Cursor Back
		cmd.Type = ANSICursorBack
		cmd.X = getParamOrDefault(params, 0, 1)
	case 'E': // CNL - Cursor Next Line
		cmd.Type = ANSICursorDown
		cmd.Y = getParamOrDefault(params, 0, 1)
		ap.addCommand(cmd)
		ap.addCommand(ANSICommand{Type: ANSICarriageReturn})
		ap.resetParser()
		return nil
	case 'F': // CPL - Cursor Previous Line
		cmd.Type = ANSICursorUp
		cmd.Y = getParamOrDefault(params, 0, 1)
		ap.addCommand(cmd)
		ap.addCommand(ANSICommand{Type: ANSICarriageReturn})
		ap.resetParser()
		return nil
	case 'G': // CHA - Cursor Horizontal Absolute
		cmd.Type = ANSICursorPosition
		cmd.X = getParamOrDefault(params, 0, 1)
		cmd.Y = -1 // Y position unchanged
	case 'H', 'f': // CUP - Cursor Position
		cmd.Type = ANSICursorPosition
		cmd.Y = getParamOrDefault(params, 0, 1)
		cmd.X = getParamOrDefault(params, 1, 1)
	case 'J': // ED - Erase Display
		cmd.Type = ANSIClearScreen
		cmd.Parameters = []int{getParamOrDefault(params, 0, 0)}
	case 'K': // EL - Erase Line
		cmd.Type = ANSIClearLine
		cmd.Parameters = []int{getParamOrDefault(params, 0, 0)}
	case 'L': // IL - Insert Line
		cmd.Type = ANSIInsertLine
		cmd.Parameters = []int{getParamOrDefault(params, 0, 1)}
	case 'M': // DL - Delete Line
		cmd.Type = ANSIDeleteLine
		cmd.Parameters = []int{getParamOrDefault(params, 0, 1)}
	case 'S': // SU - Scroll Up
		cmd.Type = ANSIScrollUp
		cmd.Parameters = []int{getParamOrDefault(params, 0, 1)}
	case 'T': // SD - Scroll Down
		cmd.Type = ANSIScrollDown
		cmd.Parameters = []int{getParamOrDefault(params, 0, 1)}
	case 'm': // SGR - Set Graphics Rendition
		cmd.Type = ANSISetGraphics
		ap.parseGraphicsParameters(&cmd, params)
	case 'n': // DSR - Device Status Report
		// 일반적으로 응답이 필요하지만 파서에서는 무시
		ap.resetParser()
		return nil
	case 'r': // DECSTBM - Set Scrolling Region
		cmd.Type = ANSISetScrollRegion
		if len(params) >= 2 {
			cmd.Parameters = []int{params[0], params[1]}
		} else {
			cmd.Parameters = []int{1, 24} // 기본값
		}
	case 's': // SCP - Save Cursor Position
		cmd.Type = ANSISaveCursor
	case 'u': // RCP - Restore Cursor Position
		cmd.Type = ANSIRestoreCursor
	case 'h': // SM - Set Mode
		if decMode {
			ap.parseDECMode(cmd, params, true)
		} else {
			cmd.Type = ANSISetMode
		}
	case 'l': // RM - Reset Mode
		if decMode {
			ap.parseDECMode(cmd, params, false)
		} else {
			cmd.Type = ANSIResetMode
		}
	default:
		if ap.config.StrictMode {
			return fmt.Errorf("unknown CSI command: %c", finalByte)
		}
		ap.resetParser()
		return nil
	}

	ap.addCommand(cmd)
	ap.resetParser()
	return nil
}

// parseDECMode parses DEC private mode sequences
func (ap *ANSIParser) parseDECMode(cmd ANSICommand, params []int, set bool) {
	for _, param := range params {
		switch param {
		case 25: // DECTCEM - Show/Hide Cursor
			if set {
				cmd.Type = ANSIShowCursor
			} else {
				cmd.Type = ANSIHideCursor
			}
			ap.addCommand(cmd)
		// 다른 DEC 모드들은 필요에 따라 추가
		}
	}
}

// processOSC handles OSC (Operating System Command) sequences
func (ap *ANSIParser) processOSC(b byte) error {
	ap.buffer = append(ap.buffer, b)

	// OSC 종료 확인
	if b == 0x07 { // BEL
		return ap.parseOSCCommand()
	}

	// ESC \ 체크 (ST - String Terminator)
	if len(ap.buffer) >= 2 && b == '\\' && ap.buffer[len(ap.buffer)-2] == 0x1B {
		return ap.parseOSCCommand()
	}

	return nil
}

// parseOSCCommand parses a complete OSC command
func (ap *ANSIParser) parseOSCCommand() error {
	// OSC 명령어 파싱
	// 예: ESC]0;Window Title\007
	start := 2 // "ESC]" 이후
	end := len(ap.buffer) - 1

	// 종료 문자가 ESC\인 경우
	if end > 0 && ap.buffer[end] == '\\' && ap.buffer[end-1] == 0x1B {
		end--
	}

	content := string(ap.buffer[start:end])
	parts := strings.SplitN(content, ";", 2)

	if len(parts) < 2 {
		ap.resetParser()
		return nil
	}

	num, err := strconv.Atoi(parts[0])
	if err != nil {
		if ap.config.StrictMode {
			return fmt.Errorf("invalid OSC number: %s", parts[0])
		}
		ap.resetParser()
		return nil
	}

	switch num {
	case 0, 2: // 윈도우 타이틀 설정
		ap.addCommand(ANSICommand{
			Type: ANSISetTitle,
			Text: parts[1],
			Raw:  copyBytes(ap.buffer),
		})
	// 다른 OSC 명령어는 필요에 따라 추가
	}

	ap.resetParser()
	return nil
}

// processDCS handles DCS (Device Control String) sequences
func (ap *ANSIParser) processDCS(b byte) error {
	ap.buffer = append(ap.buffer, b)

	// DCS는 보통 복잡하고 덜 사용되므로 기본적으로 무시
	// ESC \ (ST)로 종료
	if len(ap.buffer) >= 2 && b == '\\' && ap.buffer[len(ap.buffer)-2] == 0x1B {
		ap.resetParser()
	}

	return nil
}

// processString handles string state (OSC, DCS, etc.)
func (ap *ANSIParser) processString(b byte) error {
	return ap.processOSC(b)
}

// processIgnore handles ignore state
func (ap *ANSIParser) processIgnore(b byte) error {
	// ESC \ (ST)로 종료될 때까지 모든 바이트 무시
	if len(ap.buffer) >= 1 && b == '\\' && ap.buffer[len(ap.buffer)-1] == 0x1B {
		ap.resetParser()
	}
	return nil
}

// parseParameters parses CSI parameters
func (ap *ANSIParser) parseParameters(paramStr string) []int {
	if paramStr == "" {
		return []int{}
	}

	parts := strings.Split(paramStr, ";")
	params := make([]int, 0, len(parts))

	for _, part := range parts {
		if part == "" {
			params = append(params, 0)
		} else {
			if num, err := strconv.Atoi(part); err == nil {
				params = append(params, num)
			} else {
				params = append(params, 0)
			}
		}
	}

	return params
}

// getParamOrDefault returns parameter at index or default value
func getParamOrDefault(params []int, index, defaultValue int) int {
	if index < len(params) && params[index] > 0 {
		return params[index]
	}
	return defaultValue
}

// copyBytes creates a copy of a byte slice
func copyBytes(b []byte) []byte {
	if b == nil {
		return nil
	}
	c := make([]byte, len(b))
	copy(c, b)
	return c
}