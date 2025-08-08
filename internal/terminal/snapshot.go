package terminal

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// SnapshotManager 터미널 스냅샷 관리자
type SnapshotManager struct {
	snapshots  map[string]*Snapshot
	screens    map[string]*Screen
	mutex      sync.RWMutex
	config     *SnapshotConfig
	
	// 메트릭
	totalSnapshots uint64
	totalRestores  uint64
	totalBytes     uint64
}

// Snapshot 터미널 스냅샷
type Snapshot struct {
	ID          string                 `json:"id"`
	SessionID   string                 `json:"session_id"`
	Screen      *Screen                `json:"screen"`
	Cursor      CursorPosition         `json:"cursor"`
	ScrollBack  []string               `json:"scrollback"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata"`
	Compressed  bool                   `json:"compressed"`
	Size        int                    `json:"size"`
}

// Screen 터미널 화면
type Screen struct {
	Rows        int          `json:"rows"`
	Cols        int          `json:"cols"`
	Lines       []Line       `json:"lines"`
	CurrentLine int          `json:"current_line"`
	Buffer      *ScreenBuffer `json:"buffer"`
	Attributes  *ScreenAttrs  `json:"attributes"`
}

// Line 터미널 라인
type Line struct {
	Cells      []Cell `json:"cells"`
	Wrapped    bool   `json:"wrapped"`
	Attributes LineAttrs `json:"attributes"`
}

// Cell 터미널 셀
type Cell struct {
	Rune       rune      `json:"rune"`
	Attributes CellAttrs `json:"attributes"`
}

// CellAttrs 셀 속성
type CellAttrs struct {
	Foreground   Color  `json:"foreground"`
	Background   Color  `json:"background"`
	Bold         bool   `json:"bold"`
	Italic       bool   `json:"italic"`
	Underline    bool   `json:"underline"`
	Strikethrough bool  `json:"strikethrough"`
	Blink        bool   `json:"blink"`
	Reverse      bool   `json:"reverse"`
	Hidden       bool   `json:"hidden"`
}

// Color 색상
type Color struct {
	Type  ColorType `json:"type"`
	Value int       `json:"value"`
	RGB   *RGB      `json:"rgb,omitempty"`
}

// RGB RGB 색상
type RGB struct {
	R uint8 `json:"r"`
	G uint8 `json:"g"`
	B uint8 `json:"b"`
}

// ColorType 색상 타입
type ColorType int

const (
	ColorTypeDefault ColorType = iota
	ColorType16
	ColorType256
	ColorTypeRGB
)

// LineAttrs 라인 속성
type LineAttrs struct {
	DoubleWidth  bool `json:"double_width"`
	DoubleHeight bool `json:"double_height"`
}

// CursorPosition 커서 위치
type CursorPosition struct {
	Row     int  `json:"row"`
	Col     int  `json:"col"`
	Visible bool `json:"visible"`
	Blinking bool `json:"blinking"`
	Shape   CursorShape `json:"shape"`
}

// CursorShape 커서 모양
type CursorShape int

const (
	CursorShapeBlock CursorShape = iota
	CursorShapeUnderline
	CursorShapeBar
)

// ScreenBuffer 화면 버퍼
type ScreenBuffer struct {
	Primary   [][]Cell `json:"primary"`
	Alternate [][]Cell `json:"alternate"`
	Active    BufferType `json:"active"`
}

// BufferType 버퍼 타입
type BufferType int

const (
	BufferTypePrimary BufferType = iota
	BufferTypeAlternate
)

// ScreenAttrs 화면 속성
type ScreenAttrs struct {
	Title        string `json:"title"`
	IconTitle    string `json:"icon_title"`
	AutoWrapMode bool   `json:"auto_wrap_mode"`
	OriginMode   bool   `json:"origin_mode"`
	ReverseVideo bool   `json:"reverse_video"`
}

// SnapshotConfig 스냅샷 설정
type SnapshotConfig struct {
	MaxSnapshots      int
	MaxScrollBack     int
	EnableCompression bool
	CompressionLevel  int
	RetentionPeriod   time.Duration
	AutoSnapshot      bool
	SnapshotInterval  time.Duration
}

// DefaultSnapshotConfig 기본 스냅샷 설정
func DefaultSnapshotConfig() *SnapshotConfig {
	return &SnapshotConfig{
		MaxSnapshots:      100,
		MaxScrollBack:     1000,
		EnableCompression: true,
		CompressionLevel:  gzip.BestSpeed,
		RetentionPeriod:   24 * time.Hour,
		AutoSnapshot:      false,
		SnapshotInterval:  5 * time.Minute,
	}
}

// NewSnapshotManager 새 스냅샷 관리자 생성
func NewSnapshotManager(config *SnapshotConfig) *SnapshotManager {
	if config == nil {
		config = DefaultSnapshotConfig()
	}
	
	sm := &SnapshotManager{
		snapshots: make(map[string]*Snapshot),
		screens:   make(map[string]*Screen),
		config:    config,
	}
	
	// 자동 스냅샷 시작
	if config.AutoSnapshot {
		go sm.autoSnapshotLoop()
	}
	
	// 정리 작업 시작
	go sm.cleanupLoop()
	
	return sm
}

// CreateSnapshot 스냅샷 생성
func (sm *SnapshotManager) CreateSnapshot(sessionID string, screen *Screen) (*Snapshot, error) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	
	if len(sm.snapshots) >= sm.config.MaxSnapshots {
		return nil, fmt.Errorf("max snapshots reached: %d", sm.config.MaxSnapshots)
	}
	
	snapshot := &Snapshot{
		ID:        generateSnapshotID(),
		SessionID: sessionID,
		Screen:    sm.cloneScreen(screen),
		Cursor:    screen.GetCursorPosition(),
		ScrollBack: sm.getScrollBack(sessionID),
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
	
	// 압축 처리
	if sm.config.EnableCompression {
		if err := sm.compressSnapshot(snapshot); err != nil {
			log.Warnf("Failed to compress snapshot: %v", err)
		}
	}
	
	// 크기 계산
	snapshot.Size = sm.calculateSnapshotSize(snapshot)
	
	sm.snapshots[snapshot.ID] = snapshot
	sm.totalSnapshots++
	sm.totalBytes += uint64(snapshot.Size)
	
	log.Infof("Created snapshot %s for session %s (size: %d bytes)", 
		snapshot.ID, sessionID, snapshot.Size)
	
	return snapshot, nil
}

// RestoreSnapshot 스냅샷 복원
func (sm *SnapshotManager) RestoreSnapshot(snapshotID string) (*Screen, error) {
	sm.mutex.RLock()
	snapshot, exists := sm.snapshots[snapshotID]
	sm.mutex.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("snapshot %s not found", snapshotID)
	}
	
	// 압축 해제
	if snapshot.Compressed {
		if err := sm.decompressSnapshot(snapshot); err != nil {
			return nil, fmt.Errorf("failed to decompress snapshot: %w", err)
		}
	}
	
	// 화면 복원
	screen := sm.cloneScreen(snapshot.Screen)
	
	sm.mutex.Lock()
	sm.totalRestores++
	sm.mutex.Unlock()
	
	log.Infof("Restored snapshot %s", snapshotID)
	
	return screen, nil
}

// CaptureScreen 현재 화면 캡처
func (sm *SnapshotManager) CaptureScreen(sessionID string, rows, cols int) *Screen {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	
	screen := &Screen{
		Rows:        rows,
		Cols:        cols,
		Lines:       make([]Line, rows),
		CurrentLine: 0,
		Buffer:      sm.createScreenBuffer(rows, cols),
		Attributes:  &ScreenAttrs{},
	}
	
	// 라인 초기화
	for i := 0; i < rows; i++ {
		screen.Lines[i] = Line{
			Cells: make([]Cell, cols),
		}
		for j := 0; j < cols; j++ {
			screen.Lines[i].Cells[j] = Cell{
				Rune: ' ',
				Attributes: CellAttrs{
					Foreground: Color{Type: ColorTypeDefault},
					Background: Color{Type: ColorTypeDefault},
				},
			}
		}
	}
	
	sm.screens[sessionID] = screen
	
	return screen
}

// UpdateScreen 화면 업데이트
func (sm *SnapshotManager) UpdateScreen(sessionID string, row, col int, cell Cell) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	
	screen, exists := sm.screens[sessionID]
	if !exists {
		return fmt.Errorf("screen for session %s not found", sessionID)
	}
	
	if row < 0 || row >= screen.Rows || col < 0 || col >= screen.Cols {
		return fmt.Errorf("invalid position: (%d, %d)", row, col)
	}
	
	screen.Lines[row].Cells[col] = cell
	
	return nil
}

// GetDiff 스냅샷 차이 계산
func (sm *SnapshotManager) GetDiff(snapshot1ID, snapshot2ID string) (*SnapshotDiff, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	
	snap1, exists1 := sm.snapshots[snapshot1ID]
	snap2, exists2 := sm.snapshots[snapshot2ID]
	
	if !exists1 || !exists2 {
		return nil, fmt.Errorf("snapshot not found")
	}
	
	diff := &SnapshotDiff{
		Snapshot1ID: snapshot1ID,
		Snapshot2ID: snapshot2ID,
		Changes:     sm.calculateChanges(snap1.Screen, snap2.Screen),
		Timestamp:   time.Now(),
	}
	
	return diff, nil
}

// cloneScreen 화면 복제
func (sm *SnapshotManager) cloneScreen(screen *Screen) *Screen {
	if screen == nil {
		return nil
	}
	
	clone := &Screen{
		Rows:        screen.Rows,
		Cols:        screen.Cols,
		Lines:       make([]Line, len(screen.Lines)),
		CurrentLine: screen.CurrentLine,
		Buffer:      sm.cloneScreenBuffer(screen.Buffer),
		Attributes:  sm.cloneScreenAttrs(screen.Attributes),
	}
	
	for i, line := range screen.Lines {
		clone.Lines[i] = Line{
			Cells:      make([]Cell, len(line.Cells)),
			Wrapped:    line.Wrapped,
			Attributes: line.Attributes,
		}
		copy(clone.Lines[i].Cells, line.Cells)
	}
	
	return clone
}

// cloneScreenBuffer 화면 버퍼 복제
func (sm *SnapshotManager) cloneScreenBuffer(buffer *ScreenBuffer) *ScreenBuffer {
	if buffer == nil {
		return nil
	}
	
	clone := &ScreenBuffer{
		Primary:   sm.cloneCellBuffer(buffer.Primary),
		Alternate: sm.cloneCellBuffer(buffer.Alternate),
		Active:    buffer.Active,
	}
	
	return clone
}

// cloneCellBuffer 셀 버퍼 복제
func (sm *SnapshotManager) cloneCellBuffer(buffer [][]Cell) [][]Cell {
	if buffer == nil {
		return nil
	}
	
	clone := make([][]Cell, len(buffer))
	for i, row := range buffer {
		clone[i] = make([]Cell, len(row))
		copy(clone[i], row)
	}
	
	return clone
}

// cloneScreenAttrs 화면 속성 복제
func (sm *SnapshotManager) cloneScreenAttrs(attrs *ScreenAttrs) *ScreenAttrs {
	if attrs == nil {
		return nil
	}
	
	return &ScreenAttrs{
		Title:        attrs.Title,
		IconTitle:    attrs.IconTitle,
		AutoWrapMode: attrs.AutoWrapMode,
		OriginMode:   attrs.OriginMode,
		ReverseVideo: attrs.ReverseVideo,
	}
}

// createScreenBuffer 화면 버퍼 생성
func (sm *SnapshotManager) createScreenBuffer(rows, cols int) *ScreenBuffer {
	buffer := &ScreenBuffer{
		Primary:   make([][]Cell, rows),
		Alternate: make([][]Cell, rows),
		Active:    BufferTypePrimary,
	}
	
	for i := 0; i < rows; i++ {
		buffer.Primary[i] = make([]Cell, cols)
		buffer.Alternate[i] = make([]Cell, cols)
		
		for j := 0; j < cols; j++ {
			defaultCell := Cell{
				Rune: ' ',
				Attributes: CellAttrs{
					Foreground: Color{Type: ColorTypeDefault},
					Background: Color{Type: ColorTypeDefault},
				},
			}
			buffer.Primary[i][j] = defaultCell
			buffer.Alternate[i][j] = defaultCell
		}
	}
	
	return buffer
}

// GetCursorPosition 커서 위치 조회
func (s *Screen) GetCursorPosition() CursorPosition {
	// TODO: 실제 커서 위치 추적
	return CursorPosition{
		Row:      0,
		Col:      0,
		Visible:  true,
		Blinking: true,
		Shape:    CursorShapeBlock,
	}
}

// getScrollBack 스크롤백 조회
func (sm *SnapshotManager) getScrollBack(sessionID string) []string {
	// TODO: 실제 스크롤백 버퍼 구현
	return []string{}
}

// compressSnapshot 스냅샷 압축
func (sm *SnapshotManager) compressSnapshot(snapshot *Snapshot) error {
	data, err := json.Marshal(snapshot.Screen)
	if err != nil {
		return err
	}
	
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	gw.SetConcurrency(256<<10, 1)
	
	if _, err := gw.Write(data); err != nil {
		return err
	}
	
	if err := gw.Close(); err != nil {
		return err
	}
	
	// Base64 인코딩
	compressed := base64.StdEncoding.EncodeToString(buf.Bytes())
	
	// 압축된 데이터를 메타데이터에 저장
	snapshot.Metadata["compressed_screen"] = compressed
	snapshot.Compressed = true
	
	// 원본 화면 데이터 제거 (메모리 절약)
	snapshot.Screen = nil
	
	return nil
}

// decompressSnapshot 스냅샷 압축 해제
func (sm *SnapshotManager) decompressSnapshot(snapshot *Snapshot) error {
	compressed, ok := snapshot.Metadata["compressed_screen"].(string)
	if !ok {
		return fmt.Errorf("compressed data not found")
	}
	
	// Base64 디코딩
	data, err := base64.StdEncoding.DecodeString(compressed)
	if err != nil {
		return err
	}
	
	// Gzip 압축 해제
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer gr.Close()
	
	var screen Screen
	if err := json.NewDecoder(gr).Decode(&screen); err != nil {
		return err
	}
	
	snapshot.Screen = &screen
	
	return nil
}

// calculateSnapshotSize 스냅샷 크기 계산
func (sm *SnapshotManager) calculateSnapshotSize(snapshot *Snapshot) int {
	data, _ := json.Marshal(snapshot)
	return len(data)
}

// calculateChanges 변경사항 계산
func (sm *SnapshotManager) calculateChanges(screen1, screen2 *Screen) []Change {
	changes := []Change{}
	
	if screen1 == nil || screen2 == nil {
		return changes
	}
	
	// 화면 크기 변경 확인
	if screen1.Rows != screen2.Rows || screen1.Cols != screen2.Cols {
		changes = append(changes, Change{
			Type: ChangeTypeResize,
			Data: map[string]interface{}{
				"old_rows": screen1.Rows,
				"old_cols": screen1.Cols,
				"new_rows": screen2.Rows,
				"new_cols": screen2.Cols,
			},
		})
	}
	
	// 셀 변경 확인
	minRows := min(screen1.Rows, screen2.Rows)
	minCols := min(screen1.Cols, screen2.Cols)
	
	for i := 0; i < minRows; i++ {
		for j := 0; j < minCols; j++ {
			if !cellsEqual(screen1.Lines[i].Cells[j], screen2.Lines[i].Cells[j]) {
				changes = append(changes, Change{
					Type: ChangeTypeCell,
					Row:  i,
					Col:  j,
					Data: map[string]interface{}{
						"old": screen1.Lines[i].Cells[j],
						"new": screen2.Lines[i].Cells[j],
					},
				})
			}
		}
	}
	
	return changes
}

// autoSnapshotLoop 자동 스냅샷 루프
func (sm *SnapshotManager) autoSnapshotLoop() {
	ticker := time.NewTicker(sm.config.SnapshotInterval)
	defer ticker.Stop()
	
	for range ticker.C {
		sm.mutex.RLock()
		screens := make(map[string]*Screen)
		for sessionID, screen := range sm.screens {
			screens[sessionID] = screen
		}
		sm.mutex.RUnlock()
		
		for sessionID, screen := range screens {
			if _, err := sm.CreateSnapshot(sessionID, screen); err != nil {
				log.Errorf("Auto snapshot failed for session %s: %v", sessionID, err)
			}
		}
	}
}

// cleanupLoop 정리 루프
func (sm *SnapshotManager) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	
	for range ticker.C {
		sm.mutex.Lock()
		
		cutoff := time.Now().Add(-sm.config.RetentionPeriod)
		toDelete := []string{}
		
		for id, snapshot := range sm.snapshots {
			if snapshot.Timestamp.Before(cutoff) {
				toDelete = append(toDelete, id)
			}
		}
		
		for _, id := range toDelete {
			delete(sm.snapshots, id)
			log.Debugf("Deleted expired snapshot %s", id)
		}
		
		sm.mutex.Unlock()
		
		if len(toDelete) > 0 {
			log.Infof("Cleaned up %d expired snapshots", len(toDelete))
		}
	}
}

// GetStats 통계 조회
func (sm *SnapshotManager) GetStats() map[string]interface{} {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	
	return map[string]interface{}{
		"total_snapshots": sm.totalSnapshots,
		"current_snapshots": len(sm.snapshots),
		"total_restores":   sm.totalRestores,
		"total_bytes":      sm.totalBytes,
		"active_screens":   len(sm.screens),
	}
}

// SnapshotDiff 스냅샷 차이
type SnapshotDiff struct {
	Snapshot1ID string    `json:"snapshot1_id"`
	Snapshot2ID string    `json:"snapshot2_id"`
	Changes     []Change  `json:"changes"`
	Timestamp   time.Time `json:"timestamp"`
}

// Change 변경사항
type Change struct {
	Type ChangeType             `json:"type"`
	Row  int                    `json:"row,omitempty"`
	Col  int                    `json:"col,omitempty"`
	Data map[string]interface{} `json:"data"`
}

// ChangeType 변경 타입
type ChangeType int

const (
	ChangeTypeCell ChangeType = iota
	ChangeTypeLine
	ChangeTypeResize
	ChangeTypeAttribute
)

// cellsEqual 셀 동등성 확인
func cellsEqual(c1, c2 Cell) bool {
	return c1.Rune == c2.Rune && 
		c1.Attributes.Foreground.Type == c2.Attributes.Foreground.Type &&
		c1.Attributes.Foreground.Value == c2.Attributes.Foreground.Value &&
		c1.Attributes.Background.Type == c2.Attributes.Background.Type &&
		c1.Attributes.Background.Value == c2.Attributes.Background.Value &&
		c1.Attributes.Bold == c2.Attributes.Bold &&
		c1.Attributes.Italic == c2.Attributes.Italic &&
		c1.Attributes.Underline == c2.Attributes.Underline
}

// min 최솟값
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// generateSnapshotID 스냅샷 ID 생성
func generateSnapshotID() string {
	return fmt.Sprintf("snap-%d", time.Now().UnixNano())
}