package docker

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"
)

// TerminalSnapshot 터미널 스냅샷 인터페이스
type TerminalSnapshot interface {
	GetTimestamp() time.Time
	GetSize() (width, height int)
	GetCursor() (x, y int)
	GetBuffer() [][]Cell
	GetCompressedData() ([]byte, error)
	Restore() error
}

// Cell 터미널 셀 정보
type Cell struct {
	Char       rune           `json:"char"`
	Foreground Color          `json:"fg"`
	Background Color          `json:"bg"`
	Attributes CellAttributes `json:"attr"`
}

// Color ANSI 색상 정의
type Color struct {
	Type  ColorType `json:"type"`  // ANSI, RGB, Default
	Value uint32    `json:"value"` // 색상 값
}

// ColorType 색상 타입
type ColorType int

const (
	ColorDefault ColorType = iota
	ColorANSI
	ColorRGB
)

// CellAttributes 셀 속성
type CellAttributes struct {
	Bold      bool `json:"bold"`
	Italic    bool `json:"italic"`
	Underline bool `json:"underline"`
	Blink     bool `json:"blink"`
	Reverse   bool `json:"reverse"`
	Strikeout bool `json:"strikeout"`
}

// terminalSnapshot 터미널 스냅샷 구현
type terminalSnapshot struct {
	timestamp    time.Time   `json:"timestamp"`
	width        int         `json:"width"`
	height       int         `json:"height"`
	cursorX      int         `json:"cursor_x"`
	cursorY      int         `json:"cursor_y"`
	cursorVisible bool       `json:"cursor_visible"`
	buffer       [][]Cell    `json:"buffer"`
	compressed   []byte      `json:"-"` // 압축된 데이터 캐시
	mu           sync.RWMutex `json:"-"`
}

// NewTerminalSnapshot 새로운 터미널 스냅샷 생성
func NewTerminalSnapshot(width, height int) TerminalSnapshot {
	buffer := make([][]Cell, height)
	for i := range buffer {
		buffer[i] = make([]Cell, width)
		// 기본 셀로 초기화
		for j := range buffer[i] {
			buffer[i][j] = Cell{
				Char:       ' ',
				Foreground: Color{Type: ColorDefault},
				Background: Color{Type: ColorDefault},
			}
		}
	}

	return &terminalSnapshot{
		timestamp:     time.Now(),
		width:         width,
		height:        height,
		cursorX:       0,
		cursorY:       0,
		cursorVisible: true,
		buffer:        buffer,
	}
}

// GetTimestamp 타임스탬프 반환
func (ts *terminalSnapshot) GetTimestamp() time.Time {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.timestamp
}

// GetSize 터미널 크기 반환
func (ts *terminalSnapshot) GetSize() (width, height int) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.width, ts.height
}

// GetCursor 커서 위치 반환
func (ts *terminalSnapshot) GetCursor() (x, y int) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.cursorX, ts.cursorY
}

// GetBuffer 버퍼 반환
func (ts *terminalSnapshot) GetBuffer() [][]Cell {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	
	// 버퍼 복사 (thread-safe)
	buffer := make([][]Cell, ts.height)
	for i := range buffer {
		buffer[i] = make([]Cell, ts.width)
		copy(buffer[i], ts.buffer[i])
	}
	
	return buffer
}

// GetCompressedData 압축된 데이터 반환
func (ts *terminalSnapshot) GetCompressedData() ([]byte, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	// 캐시된 압축 데이터가 있으면 반환
	if ts.compressed != nil {
		return ts.compressed, nil
	}

	// JSON으로 직렬화
	data, err := json.Marshal(ts)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	// gzip 압축
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	
	_, err = gzipWriter.Write(data)
	if err != nil {
		gzipWriter.Close()
		return nil, fmt.Errorf("failed to compress data: %w", err)
	}

	if err := gzipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	ts.compressed = buf.Bytes()
	return ts.compressed, nil
}

// Restore 스냅샷 복원 (추후 구현)
func (ts *terminalSnapshot) Restore() error {
	// TODO: 스냅샷에서 터미널 상태 복원
	return nil
}

// SetCell 셀 내용 설정
func (ts *terminalSnapshot) SetCell(x, y int, cell Cell) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if x < 0 || x >= ts.width || y < 0 || y >= ts.height {
		return fmt.Errorf("invalid cell position (%d, %d)", x, y)
	}

	ts.buffer[y][x] = cell
	ts.compressed = nil // 캐시 무효화
	return nil
}

// SetCursor 커서 위치 설정
func (ts *terminalSnapshot) SetCursor(x, y int, visible bool) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if x < 0 || x >= ts.width || y < 0 || y >= ts.height {
		return fmt.Errorf("invalid cursor position (%d, %d)", x, y)
	}

	ts.cursorX = x
	ts.cursorY = y
	ts.cursorVisible = visible
	ts.compressed = nil // 캐시 무효화
	return nil
}

// SnapshotCapturer 터미널 스냅샷 캡처 시스템
type SnapshotCapturer struct {
	ptySession   PTYSession
	interval     time.Duration
	maxHistory   int
	snapshots    []TerminalSnapshot
	ansiParser   *ANSIParser
	mu           sync.RWMutex
	stopChan     chan struct{}
	isRunning    bool
}

// NewSnapshotCapturer 새로운 스냅샷 캡처 시스템 생성
func NewSnapshotCapturer(ptySession PTYSession, interval time.Duration, maxHistory int) *SnapshotCapturer {
	return &SnapshotCapturer{
		ptySession: ptySession,
		interval:   interval,
		maxHistory: maxHistory,
		snapshots:  make([]TerminalSnapshot, 0, maxHistory),
		ansiParser: NewANSIParser(),
		stopChan:   make(chan struct{}),
		isRunning:  false,
	}
}

// Start 스냅샷 캡처 시작
func (sc *SnapshotCapturer) Start() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if sc.isRunning {
		return fmt.Errorf("snapshot capturer is already running")
	}

	sc.isRunning = true
	sc.stopChan = make(chan struct{})

	// 백그라운드 캡처 고루틴 시작
	go sc.captureLoop()

	return nil
}

// Stop 스냅샷 캡처 중지
func (sc *SnapshotCapturer) Stop() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if !sc.isRunning {
		return nil
	}

	sc.isRunning = false
	close(sc.stopChan)

	return nil
}

// captureLoop 캡처 루프
func (sc *SnapshotCapturer) captureLoop() {
	ticker := time.NewTicker(sc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-sc.stopChan:
			return
		case <-ticker.C:
			sc.captureSnapshot()
		}
	}
}

// captureSnapshot 스냅샷 캡처
func (sc *SnapshotCapturer) captureSnapshot() {
	if !sc.ptySession.IsAlive() {
		return
	}

	// 현재 터미널 크기 확인 (기본값 사용)
	width, height := 80, 24 // 기본 터미널 크기

	// 새 스냅샷 생성
	snapshot := NewTerminalSnapshot(width, height)

	// PTY 세션에서 현재 상태 읽기 시도
	buffer := make([]byte, 4096)
	n, err := sc.ptySession.Read(buffer)
	if err != nil && err != io.EOF {
		// 읽기 오류 시 빈 스냅샷 저장
	} else if n > 0 {
		// ANSI 파싱하여 스냅샷 업데이트
		sc.ansiParser.ParseToSnapshot(buffer[:n], snapshot.(*terminalSnapshot))
	}

	// 히스토리에 추가
	sc.addSnapshot(snapshot)
}

// addSnapshot 스냅샷 히스토리에 추가
func (sc *SnapshotCapturer) addSnapshot(snapshot TerminalSnapshot) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// 최대 히스토리 수 초과 시 오래된 것 제거
	if len(sc.snapshots) >= sc.maxHistory {
		// 첫 번째 요소 제거 (가장 오래된 것)
		sc.snapshots = sc.snapshots[1:]
	}

	// 새 스냅샷 추가
	sc.snapshots = append(sc.snapshots, snapshot)
}

// GetLatestSnapshot 최신 스냅샷 반환
func (sc *SnapshotCapturer) GetLatestSnapshot() TerminalSnapshot {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	if len(sc.snapshots) == 0 {
		return nil
	}

	return sc.snapshots[len(sc.snapshots)-1]
}

// GetSnapshotHistory 스냅샷 히스토리 반환
func (sc *SnapshotCapturer) GetSnapshotHistory() []TerminalSnapshot {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	// 히스토리 복사
	history := make([]TerminalSnapshot, len(sc.snapshots))
	copy(history, sc.snapshots)

	return history
}

// GetSnapshotCount 스냅샷 개수 반환
func (sc *SnapshotCapturer) GetSnapshotCount() int {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return len(sc.snapshots)
}

// ClearHistory 히스토리 정리
func (sc *SnapshotCapturer) ClearHistory() {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.snapshots = sc.snapshots[:0]
}

// GetMemoryUsage 메모리 사용량 추정
func (sc *SnapshotCapturer) GetMemoryUsage() int64 {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	var totalSize int64
	for _, snapshot := range sc.snapshots {
		// 압축된 데이터 크기 사용
		if compressed, err := snapshot.GetCompressedData(); err == nil {
			totalSize += int64(len(compressed))
		} else {
			// 압축 실패 시 추정값 사용 (80x24 터미널 기준)
			width, height := snapshot.GetSize()
			totalSize += int64(width * height * 20) // 셀당 약 20바이트 추정
		}
	}

	return totalSize
}

// ANSIParser ANSI 이스케이프 시퀀스 파서
type ANSIParser struct {
	currentFG Color
	currentBG Color
	currentAttr CellAttributes
}

// NewANSIParser 새로운 ANSI 파서 생성
func NewANSIParser() *ANSIParser {
	return &ANSIParser{
		currentFG: Color{Type: ColorDefault},
		currentBG: Color{Type: ColorDefault},
	}
}

// ParseToSnapshot ANSI 데이터를 파싱하여 스냅샷에 적용
func (ap *ANSIParser) ParseToSnapshot(data []byte, snapshot *terminalSnapshot) error {
	// 간단한 텍스트 파싱 구현 (ANSI 이스케이프 시퀀스는 추후 확장)
	
	// 현재는 일반 텍스트만 처리
	text := string(data)
	x, y := snapshot.GetCursor()
	width, height := snapshot.GetSize()

	for _, char := range text {
		if char == '\n' {
			// 줄바꿈
			x = 0
			y++
			if y >= height {
				y = height - 1
				// 스크롤 처리 (추후 구현)
			}
		} else if char == '\r' {
			// 캐리지 리턴
			x = 0
		} else if char >= 32 { // 출력 가능한 문자
			// 문자 출력
			if x < width && y < height {
				cell := Cell{
					Char:       char,
					Foreground: ap.currentFG,
					Background: ap.currentBG,
					Attributes: ap.currentAttr,
				}
				snapshot.SetCell(x, y, cell)
			}
			x++
			if x >= width {
				x = 0
				y++
				if y >= height {
					y = height - 1
				}
			}
		}
		// 다른 제어 문자는 무시 (추후 ANSI 파싱 확장)
	}

	// 커서 위치 업데이트
	snapshot.SetCursor(x, y, true)

	return nil
}

// SnapshotCapturerStats 스냅샷 캡처 통계
type SnapshotCapturerStats struct {
	IsRunning       bool          `json:"is_running"`
	SnapshotCount   int           `json:"snapshot_count"`
	MaxHistory      int           `json:"max_history"`
	CaptureInterval time.Duration `json:"capture_interval"`
	MemoryUsage     int64         `json:"memory_usage"`
	LastCaptureTime time.Time     `json:"last_capture_time"`
}

// GetStats 통계 정보 반환
func (sc *SnapshotCapturer) GetStats() *SnapshotCapturerStats {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	var lastCaptureTime time.Time
	if len(sc.snapshots) > 0 {
		lastCaptureTime = sc.snapshots[len(sc.snapshots)-1].GetTimestamp()
	}

	return &SnapshotCapturerStats{
		IsRunning:       sc.isRunning,
		SnapshotCount:   len(sc.snapshots),
		MaxHistory:      sc.maxHistory,
		CaptureInterval: sc.interval,
		MemoryUsage:     sc.GetMemoryUsage(),
		LastCaptureTime: lastCaptureTime,
	}
}