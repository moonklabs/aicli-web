package terminal

import (
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// BufferManager 터미널 버퍼 관리자
type BufferManager struct {
	buffers map[string]*TerminalBuffer
	mutex   sync.RWMutex
	config  *BufferConfig
}

// TerminalBuffer 터미널 버퍼
type TerminalBuffer struct {
	SessionID    string
	Lines        []BufferLine
	MaxLines     int
	CurrentLine  int
	ScrollOffset int
	mutex        sync.RWMutex
	
	// 버퍼 타입
	Primary      *CircularBuffer
	Alternate    *CircularBuffer
	ActiveBuffer BufferType
	
	// 통계
	totalLines   uint64
	totalBytes   uint64
}

// BufferLine 버퍼 라인
type BufferLine struct {
	Text       string
	Cells      []Cell
	Timestamp  int64
	Attributes LineAttrs
}

// CircularBuffer 순환 버퍼
type CircularBuffer struct {
	lines    []BufferLine
	capacity int
	head     int
	tail     int
	size     int
	mutex    sync.RWMutex
}

// BufferConfig 버퍼 설정
type BufferConfig struct {
	MaxBufferLines   int
	MaxScrollBack    int
	EnableAlternate  bool
	CircularCapacity int
}

// DefaultBufferConfig 기본 버퍼 설정
func DefaultBufferConfig() *BufferConfig {
	return &BufferConfig{
		MaxBufferLines:   10000,
		MaxScrollBack:    5000,
		EnableAlternate:  true,
		CircularCapacity: 1000,
	}
}

// NewBufferManager 새 버퍼 관리자 생성
func NewBufferManager(config *BufferConfig) *BufferManager {
	if config == nil {
		config = DefaultBufferConfig()
	}
	
	return &BufferManager{
		buffers: make(map[string]*TerminalBuffer),
		config:  config,
	}
}

// CreateBuffer 버퍼 생성
func (bm *BufferManager) CreateBuffer(sessionID string) *TerminalBuffer {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()
	
	buffer := &TerminalBuffer{
		SessionID:    sessionID,
		Lines:        make([]BufferLine, 0, bm.config.MaxBufferLines),
		MaxLines:     bm.config.MaxBufferLines,
		CurrentLine:  0,
		ScrollOffset: 0,
		Primary:      NewCircularBuffer(bm.config.CircularCapacity),
		ActiveBuffer: BufferTypePrimary,
	}
	
	if bm.config.EnableAlternate {
		buffer.Alternate = NewCircularBuffer(bm.config.CircularCapacity)
	}
	
	bm.buffers[sessionID] = buffer
	
	log.Infof("Created buffer for session %s", sessionID)
	return buffer
}

// GetBuffer 버퍼 조회
func (bm *BufferManager) GetBuffer(sessionID string) (*TerminalBuffer, error) {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()
	
	buffer, exists := bm.buffers[sessionID]
	if !exists {
		return nil, fmt.Errorf("buffer for session %s not found", sessionID)
	}
	
	return buffer, nil
}

// DeleteBuffer 버퍼 삭제
func (bm *BufferManager) DeleteBuffer(sessionID string) error {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()
	
	if _, exists := bm.buffers[sessionID]; !exists {
		return fmt.Errorf("buffer for session %s not found", sessionID)
	}
	
	delete(bm.buffers, sessionID)
	
	log.Infof("Deleted buffer for session %s", sessionID)
	return nil
}

// AppendLine 라인 추가
func (tb *TerminalBuffer) AppendLine(text string, cells []Cell) {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()
	
	line := BufferLine{
		Text:      text,
		Cells:     cells,
		Timestamp: timeNow(),
	}
	
	// 순환 버퍼에 추가
	activeBuffer := tb.getActiveBuffer()
	activeBuffer.Append(line)
	
	// 메인 버퍼에도 추가 (스크롤백용)
	if len(tb.Lines) >= tb.MaxLines {
		// 오래된 라인 제거
		tb.Lines = tb.Lines[1:]
	}
	tb.Lines = append(tb.Lines, line)
	
	tb.CurrentLine = len(tb.Lines) - 1
	tb.totalLines++
	tb.totalBytes += uint64(len(text))
}

// InsertLine 라인 삽입
func (tb *TerminalBuffer) InsertLine(index int, text string, cells []Cell) error {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()
	
	if index < 0 || index > len(tb.Lines) {
		return fmt.Errorf("invalid index: %d", index)
	}
	
	line := BufferLine{
		Text:      text,
		Cells:     cells,
		Timestamp: timeNow(),
	}
	
	// 라인 삽입
	tb.Lines = append(tb.Lines[:index], append([]BufferLine{line}, tb.Lines[index:]...)...)
	
	// 최대 라인 수 유지
	if len(tb.Lines) > tb.MaxLines {
		tb.Lines = tb.Lines[len(tb.Lines)-tb.MaxLines:]
	}
	
	tb.totalLines++
	tb.totalBytes += uint64(len(text))
	
	return nil
}

// DeleteLine 라인 삭제
func (tb *TerminalBuffer) DeleteLine(index int) error {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()
	
	if index < 0 || index >= len(tb.Lines) {
		return fmt.Errorf("invalid index: %d", index)
	}
	
	tb.Lines = append(tb.Lines[:index], tb.Lines[index+1:]...)
	
	if tb.CurrentLine >= len(tb.Lines) && tb.CurrentLine > 0 {
		tb.CurrentLine = len(tb.Lines) - 1
	}
	
	return nil
}

// ClearBuffer 버퍼 지우기
func (tb *TerminalBuffer) ClearBuffer() {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()
	
	tb.Lines = tb.Lines[:0]
	tb.CurrentLine = 0
	tb.ScrollOffset = 0
	
	// 순환 버퍼도 지우기
	tb.Primary.Clear()
	if tb.Alternate != nil {
		tb.Alternate.Clear()
	}
	
	log.Debugf("Cleared buffer for session %s", tb.SessionID)
}

// GetLines 라인 조회
func (tb *TerminalBuffer) GetLines(start, count int) []BufferLine {
	tb.mutex.RLock()
	defer tb.mutex.RUnlock()
	
	if start < 0 || start >= len(tb.Lines) {
		return []BufferLine{}
	}
	
	end := start + count
	if end > len(tb.Lines) {
		end = len(tb.Lines)
	}
	
	result := make([]BufferLine, end-start)
	copy(result, tb.Lines[start:end])
	
	return result
}

// GetVisibleLines 화면에 보이는 라인 조회
func (tb *TerminalBuffer) GetVisibleLines(rows int) []BufferLine {
	tb.mutex.RLock()
	defer tb.mutex.RUnlock()
	
	activeBuffer := tb.getActiveBuffer()
	return activeBuffer.GetRecent(rows)
}

// ScrollUp 위로 스크롤
func (tb *TerminalBuffer) ScrollUp(lines int) {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()
	
	tb.ScrollOffset += lines
	maxOffset := len(tb.Lines) - 1
	
	if tb.ScrollOffset > maxOffset {
		tb.ScrollOffset = maxOffset
	}
}

// ScrollDown 아래로 스크롤
func (tb *TerminalBuffer) ScrollDown(lines int) {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()
	
	tb.ScrollOffset -= lines
	
	if tb.ScrollOffset < 0 {
		tb.ScrollOffset = 0
	}
}

// ScrollToBottom 맨 아래로 스크롤
func (tb *TerminalBuffer) ScrollToBottom() {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()
	
	tb.ScrollOffset = 0
}

// ScrollToTop 맨 위로 스크롤
func (tb *TerminalBuffer) ScrollToTop() {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()
	
	tb.ScrollOffset = len(tb.Lines) - 1
}

// SwitchBuffer 버퍼 전환
func (tb *TerminalBuffer) SwitchBuffer() {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()
	
	if tb.Alternate == nil {
		return
	}
	
	if tb.ActiveBuffer == BufferTypePrimary {
		tb.ActiveBuffer = BufferTypeAlternate
	} else {
		tb.ActiveBuffer = BufferTypePrimary
	}
	
	log.Debugf("Switched to %v buffer for session %s", tb.ActiveBuffer, tb.SessionID)
}

// getActiveBuffer 활성 버퍼 조회
func (tb *TerminalBuffer) getActiveBuffer() *CircularBuffer {
	if tb.ActiveBuffer == BufferTypeAlternate && tb.Alternate != nil {
		return tb.Alternate
	}
	return tb.Primary
}

// GetScrollBack 스크롤백 조회
func (tb *TerminalBuffer) GetScrollBack(maxLines int) []string {
	tb.mutex.RLock()
	defer tb.mutex.RUnlock()
	
	start := 0
	if len(tb.Lines) > maxLines {
		start = len(tb.Lines) - maxLines
	}
	
	result := make([]string, 0, len(tb.Lines)-start)
	for i := start; i < len(tb.Lines); i++ {
		result = append(result, tb.Lines[i].Text)
	}
	
	return result
}

// Search 텍스트 검색
func (tb *TerminalBuffer) Search(pattern string, caseSensitive bool) []SearchResult {
	tb.mutex.RLock()
	defer tb.mutex.RUnlock()
	
	results := []SearchResult{}
	
	for i, line := range tb.Lines {
		matches := findMatches(line.Text, pattern, caseSensitive)
		for _, match := range matches {
			results = append(results, SearchResult{
				LineIndex: i,
				Column:    match.Start,
				Length:    match.Length,
				Text:      line.Text[match.Start:match.Start+match.Length],
			})
		}
	}
	
	return results
}

// GetStats 통계 조회
func (tb *TerminalBuffer) GetStats() map[string]interface{} {
	tb.mutex.RLock()
	defer tb.mutex.RUnlock()
	
	return map[string]interface{}{
		"session_id":     tb.SessionID,
		"total_lines":    tb.totalLines,
		"current_lines":  len(tb.Lines),
		"total_bytes":    tb.totalBytes,
		"scroll_offset":  tb.ScrollOffset,
		"active_buffer":  tb.ActiveBuffer,
		"primary_size":   tb.Primary.Size(),
		"alternate_size": 0,
	}
}

// NewCircularBuffer 새 순환 버퍼 생성
func NewCircularBuffer(capacity int) *CircularBuffer {
	return &CircularBuffer{
		lines:    make([]BufferLine, capacity),
		capacity: capacity,
		head:     0,
		tail:     0,
		size:     0,
	}
}

// Append 라인 추가
func (cb *CircularBuffer) Append(line BufferLine) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	cb.lines[cb.tail] = line
	cb.tail = (cb.tail + 1) % cb.capacity
	
	if cb.size < cb.capacity {
		cb.size++
	} else {
		cb.head = (cb.head + 1) % cb.capacity
	}
}

// GetRecent 최근 라인 조회
func (cb *CircularBuffer) GetRecent(count int) []BufferLine {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	
	if count > cb.size {
		count = cb.size
	}
	
	result := make([]BufferLine, count)
	start := (cb.tail - count + cb.capacity) % cb.capacity
	
	for i := 0; i < count; i++ {
		idx := (start + i) % cb.capacity
		result[i] = cb.lines[idx]
	}
	
	return result
}

// GetAll 모든 라인 조회
func (cb *CircularBuffer) GetAll() []BufferLine {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	
	result := make([]BufferLine, cb.size)
	
	for i := 0; i < cb.size; i++ {
		idx := (cb.head + i) % cb.capacity
		result[i] = cb.lines[idx]
	}
	
	return result
}

// Clear 버퍼 지우기
func (cb *CircularBuffer) Clear() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	cb.head = 0
	cb.tail = 0
	cb.size = 0
}

// Size 버퍼 크기
func (cb *CircularBuffer) Size() int {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	
	return cb.size
}

// SearchResult 검색 결과
type SearchResult struct {
	LineIndex int    `json:"line_index"`
	Column    int    `json:"column"`
	Length    int    `json:"length"`
	Text      string `json:"text"`
}

// Match 매치 정보
type Match struct {
	Start  int
	Length int
}

// findMatches 매치 찾기
func findMatches(text, pattern string, caseSensitive bool) []Match {
	matches := []Match{}
	
	if !caseSensitive {
		text = toLower(text)
		pattern = toLower(pattern)
	}
	
	patternLen := len(pattern)
	textLen := len(text)
	
	for i := 0; i <= textLen-patternLen; i++ {
		if text[i:i+patternLen] == pattern {
			matches = append(matches, Match{
				Start:  i,
				Length: patternLen,
			})
			i += patternLen - 1
		}
	}
	
	return matches
}

// toLower 소문자 변환 (간단한 구현)
func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

// timeNow 현재 시간 (Unix 타임스탬프)
func timeNow() int64 {
	return time.Now().UnixNano() / 1e6
}