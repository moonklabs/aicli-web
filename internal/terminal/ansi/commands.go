package ansi

// CommandType represents the type of ANSI command
type CommandType int

const (
	// 텍스트 및 제어
	ANSIText CommandType = iota
	ANSICarriageReturn
	ANSILineFeed
	ANSITab
	ANSIBackspace

	// 커서 제어
	ANSICursorUp
	ANSICursorDown
	ANSICursorForward
	ANSICursorBack
	ANSICursorPosition
	ANSISaveCursor
	ANSIRestoreCursor
	ANSIShowCursor
	ANSIHideCursor

	// 화면 조작
	ANSIClearScreen
	ANSIClearLine
	ANSIInsertLine
	ANSIDeleteLine
	ANSIScrollUp
	ANSIScrollDown

	// 색상 및 속성
	ANSISetGraphics
	ANSISetForeground
	ANSISetBackground
	ANSIResetAttributes

	// 모드 설정
	ANSISetMode
	ANSIResetMode
	ANSISetScrollRegion

	// 기타
	ANSIBell
	ANSISetTitle
)

// ANSICommand represents a parsed ANSI command
type ANSICommand struct {
	Type       CommandType
	Parameters []int
	Text       string
	X, Y       int
	Foreground Color
	Background Color
	Attributes TextAttributes
	Raw        []byte
}

// TextAttributes represents text display attributes
type TextAttributes struct {
	Bold          bool
	Dim           bool
	Italic        bool
	Underline     bool
	Blink         bool
	Reverse       bool
	Strikethrough bool
	Hidden        bool
}

// String returns the string representation of a command type
func (ct CommandType) String() string {
	switch ct {
	case ANSIText:
		return "Text"
	case ANSICarriageReturn:
		return "CarriageReturn"
	case ANSILineFeed:
		return "LineFeed"
	case ANSITab:
		return "Tab"
	case ANSIBackspace:
		return "Backspace"
	case ANSICursorUp:
		return "CursorUp"
	case ANSICursorDown:
		return "CursorDown"
	case ANSICursorForward:
		return "CursorForward"
	case ANSICursorBack:
		return "CursorBack"
	case ANSICursorPosition:
		return "CursorPosition"
	case ANSISaveCursor:
		return "SaveCursor"
	case ANSIRestoreCursor:
		return "RestoreCursor"
	case ANSIShowCursor:
		return "ShowCursor"
	case ANSIHideCursor:
		return "HideCursor"
	case ANSIClearScreen:
		return "ClearScreen"
	case ANSIClearLine:
		return "ClearLine"
	case ANSIInsertLine:
		return "InsertLine"
	case ANSIDeleteLine:
		return "DeleteLine"
	case ANSIScrollUp:
		return "ScrollUp"
	case ANSIScrollDown:
		return "ScrollDown"
	case ANSISetGraphics:
		return "SetGraphics"
	case ANSISetForeground:
		return "SetForeground"
	case ANSISetBackground:
		return "SetBackground"
	case ANSIResetAttributes:
		return "ResetAttributes"
	case ANSISetMode:
		return "SetMode"
	case ANSIResetMode:
		return "ResetMode"
	case ANSISetScrollRegion:
		return "SetScrollRegion"
	case ANSIBell:
		return "Bell"
	case ANSISetTitle:
		return "SetTitle"
	default:
		return "Unknown"
	}
}

// Reset resets all text attributes to default
func (ta *TextAttributes) Reset() {
	ta.Bold = false
	ta.Dim = false
	ta.Italic = false
	ta.Underline = false
	ta.Blink = false
	ta.Reverse = false
	ta.Strikethrough = false
	ta.Hidden = false
}