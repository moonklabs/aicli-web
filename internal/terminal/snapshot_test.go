package terminal

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestSnapshotManagerCreation 스냅샷 관리자 생성 테스트
func TestSnapshotManagerCreation(t *testing.T) {
	config := DefaultSnapshotConfig()
	sm := NewSnapshotManager(config)
	
	assert.NotNil(t, sm)
	assert.NotNil(t, sm.snapshots)
	assert.NotNil(t, sm.screens)
	assert.Equal(t, config.MaxSnapshots, sm.config.MaxSnapshots)
}

// TestCreateSnapshot 스냅샷 생성 테스트
func TestCreateSnapshot(t *testing.T) {
	sm := NewSnapshotManager(nil)
	
	// 화면 캡처
	screen := sm.CaptureScreen("test-session", 24, 80)
	assert.NotNil(t, screen)
	
	// 스냅샷 생성
	snapshot, err := sm.CreateSnapshot("test-session", screen)
	assert.NoError(t, err)
	assert.NotNil(t, snapshot)
	assert.NotEmpty(t, snapshot.ID)
	assert.Equal(t, "test-session", snapshot.SessionID)
	assert.Equal(t, 24, snapshot.Screen.Rows)
	assert.Equal(t, 80, snapshot.Screen.Cols)
}

// TestRestoreSnapshot 스냅샷 복원 테스트
func TestRestoreSnapshot(t *testing.T) {
	sm := NewSnapshotManager(nil)
	
	// 화면 생성 및 수정
	screen := sm.CaptureScreen("test-session", 24, 80)
	
	// 셀 업데이트
	testCell := Cell{
		Rune: 'A',
		Attributes: CellAttrs{
			Foreground: Color{Type: ColorType16, Value: 1},
			Bold:       true,
		},
	}
	
	err := sm.UpdateScreen("test-session", 0, 0, testCell)
	assert.NoError(t, err)
	
	// 스냅샷 생성
	snapshot, err := sm.CreateSnapshot("test-session", screen)
	assert.NoError(t, err)
	
	// 스냅샷 복원
	restoredScreen, err := sm.RestoreSnapshot(snapshot.ID)
	assert.NoError(t, err)
	assert.NotNil(t, restoredScreen)
	assert.Equal(t, screen.Rows, restoredScreen.Rows)
	assert.Equal(t, screen.Cols, restoredScreen.Cols)
}

// TestMaxSnapshots 최대 스냅샷 수 테스트
func TestMaxSnapshots(t *testing.T) {
	config := &SnapshotConfig{
		MaxSnapshots:      3,
		EnableCompression: false,
	}
	sm := NewSnapshotManager(config)
	
	screen := sm.CaptureScreen("test-session", 24, 80)
	
	// 최대 스냅샷 수만큼 생성
	for i := 0; i < 3; i++ {
		snapshot, err := sm.CreateSnapshot("test-session", screen)
		assert.NoError(t, err)
		assert.NotNil(t, snapshot)
	}
	
	// 추가 스냅샷 생성 시도 (실패해야 함)
	_, err := sm.CreateSnapshot("test-session", screen)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max snapshots reached")
}

// TestScreenUpdate 화면 업데이트 테스트
func TestScreenUpdate(t *testing.T) {
	sm := NewSnapshotManager(nil)
	
	// 화면 캡처
	screen := sm.CaptureScreen("test-session", 24, 80)
	
	// 여러 셀 업데이트
	testCells := []struct {
		row  int
		col  int
		cell Cell
	}{
		{0, 0, Cell{Rune: 'H'}},
		{0, 1, Cell{Rune: 'e'}},
		{0, 2, Cell{Rune: 'l'}},
		{0, 3, Cell{Rune: 'l'}},
		{0, 4, Cell{Rune: 'o'}},
	}
	
	for _, tc := range testCells {
		err := sm.UpdateScreen("test-session", tc.row, tc.col, tc.cell)
		assert.NoError(t, err)
	}
	
	// 업데이트 확인
	assert.Equal(t, 'H', screen.Lines[0].Cells[0].Rune)
	assert.Equal(t, 'e', screen.Lines[0].Cells[1].Rune)
	assert.Equal(t, 'l', screen.Lines[0].Cells[2].Rune)
	assert.Equal(t, 'l', screen.Lines[0].Cells[3].Rune)
	assert.Equal(t, 'o', screen.Lines[0].Cells[4].Rune)
}

// TestSnapshotCompression 스냅샷 압축 테스트
func TestSnapshotCompression(t *testing.T) {
	config := &SnapshotConfig{
		MaxSnapshots:      10,
		EnableCompression: true,
		CompressionLevel:  1,
	}
	sm := NewSnapshotManager(config)
	
	screen := sm.CaptureScreen("test-session", 24, 80)
	
	// 화면에 데이터 채우기
	for i := 0; i < 24; i++ {
		for j := 0; j < 80; j++ {
			sm.UpdateScreen("test-session", i, j, Cell{Rune: 'X'})
		}
	}
	
	// 압축된 스냅샷 생성
	snapshot, err := sm.CreateSnapshot("test-session", screen)
	assert.NoError(t, err)
	assert.True(t, snapshot.Compressed)
	assert.NotNil(t, snapshot.Metadata["compressed_screen"])
	
	// 복원 테스트
	restoredScreen, err := sm.RestoreSnapshot(snapshot.ID)
	assert.NoError(t, err)
	assert.NotNil(t, restoredScreen)
}

// TestBufferManager 버퍼 관리자 테스트
func TestBufferManager(t *testing.T) {
	bm := NewBufferManager(nil)
	
	// 버퍼 생성
	buffer := bm.CreateBuffer("test-session")
	assert.NotNil(t, buffer)
	assert.Equal(t, "test-session", buffer.SessionID)
	
	// 라인 추가
	buffer.AppendLine("Hello, World!", nil)
	buffer.AppendLine("Second line", nil)
	buffer.AppendLine("Third line", nil)
	
	assert.Equal(t, 3, len(buffer.Lines))
	
	// 라인 조회
	lines := buffer.GetLines(0, 2)
	assert.Equal(t, 2, len(lines))
	assert.Equal(t, "Hello, World!", lines[0].Text)
	assert.Equal(t, "Second line", lines[1].Text)
	
	// 버퍼 삭제
	err := bm.DeleteBuffer("test-session")
	assert.NoError(t, err)
}

// TestCircularBuffer 순환 버퍼 테스트
func TestCircularBuffer(t *testing.T) {
	cb := NewCircularBuffer(5)
	
	// 버퍼 용량보다 많은 라인 추가
	for i := 0; i < 10; i++ {
		cb.Append(BufferLine{
			Text: fmt.Sprintf("Line %d", i),
		})
	}
	
	// 크기 확인 (최대 용량)
	assert.Equal(t, 5, cb.Size())
	
	// 최근 라인 조회
	recent := cb.GetRecent(3)
	assert.Equal(t, 3, len(recent))
	assert.Equal(t, "Line 7", recent[0].Text)
	assert.Equal(t, "Line 8", recent[1].Text)
	assert.Equal(t, "Line 9", recent[2].Text)
	
	// 모든 라인 조회
	all := cb.GetAll()
	assert.Equal(t, 5, len(all))
	assert.Equal(t, "Line 5", all[0].Text)
	assert.Equal(t, "Line 9", all[4].Text)
}

// TestSerializer 직렬화기 테스트
func TestSerializer(t *testing.T) {
	serializer := NewSerializer(nil)
	sm := NewSnapshotManager(nil)
	
	// 화면 생성
	screen := sm.CaptureScreen("test-session", 10, 40)
	
	// 스냅샷 생성
	snapshot, err := sm.CreateSnapshot("test-session", screen)
	assert.NoError(t, err)
	
	// 직렬화
	data, err := serializer.SerializeSnapshot(snapshot)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)
	
	// 역직렬화
	restored, err := serializer.DeserializeSnapshot(data)
	assert.NoError(t, err)
	assert.NotNil(t, restored)
	assert.Equal(t, snapshot.ID, restored.ID)
	assert.Equal(t, snapshot.SessionID, restored.SessionID)
}

// TestExportImport 내보내기/가져오기 테스트
func TestExportImport(t *testing.T) {
	serializer := NewSerializer(nil)
	sm := NewSnapshotManager(nil)
	
	// 스냅샷 생성
	screen := sm.CaptureScreen("test-session", 5, 10)
	snapshot, _ := sm.CreateSnapshot("test-session", screen)
	
	// Base64 내보내기
	exported, err := serializer.ExportSnapshot(snapshot, ExportFormatBase64)
	assert.NoError(t, err)
	assert.NotEmpty(t, exported)
	
	// Base64 가져오기
	imported, err := serializer.ImportSnapshot(exported, ExportFormatBase64)
	assert.NoError(t, err)
	assert.NotNil(t, imported)
	assert.Equal(t, snapshot.ID, imported.ID)
}

// TestBufferSearch 버퍼 검색 테스트
func TestBufferSearch(t *testing.T) {
	buffer := &TerminalBuffer{
		SessionID: "test",
		Lines: []BufferLine{
			{Text: "Hello World"},
			{Text: "This is a test"},
			{Text: "Hello again"},
			{Text: "World of testing"},
		},
		MaxLines: 100,
	}
	
	// 대소문자 구분 검색
	results := buffer.Search("Hello", true)
	assert.Equal(t, 2, len(results))
	assert.Equal(t, 0, results[0].LineIndex)
	assert.Equal(t, 2, results[1].LineIndex)
	
	// 대소문자 무시 검색
	results = buffer.Search("world", false)
	assert.Equal(t, 2, len(results))
	assert.Equal(t, 0, results[0].LineIndex)
	assert.Equal(t, 3, results[1].LineIndex)
}

// TestSnapshotDiff 스냅샷 차이 테스트
func TestSnapshotDiff(t *testing.T) {
	sm := NewSnapshotManager(nil)
	
	// 첫 번째 스냅샷
	screen1 := sm.CaptureScreen("test-session", 5, 10)
	sm.UpdateScreen("test-session", 0, 0, Cell{Rune: 'A'})
	snapshot1, _ := sm.CreateSnapshot("test-session", screen1)
	
	// 두 번째 스냅샷 (변경 후)
	screen2 := sm.CaptureScreen("test-session", 5, 10)
	sm.UpdateScreen("test-session", 0, 0, Cell{Rune: 'B'})
	sm.UpdateScreen("test-session", 1, 1, Cell{Rune: 'C'})
	snapshot2, _ := sm.CreateSnapshot("test-session", screen2)
	
	// 차이 계산
	diff, err := sm.GetDiff(snapshot1.ID, snapshot2.ID)
	assert.NoError(t, err)
	assert.NotNil(t, diff)
	assert.Greater(t, len(diff.Changes), 0)
}

// BenchmarkSnapshotCreation 스냅샷 생성 벤치마크
func BenchmarkSnapshotCreation(b *testing.B) {
	sm := NewSnapshotManager(nil)
	screen := sm.CaptureScreen("bench-session", 24, 80)
	
	// 화면 채우기
	for i := 0; i < 24; i++ {
		for j := 0; j < 80; j++ {
			sm.UpdateScreen("bench-session", i, j, Cell{Rune: 'X'})
		}
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		sm.CreateSnapshot("bench-session", screen)
	}
}

// BenchmarkSerialization 직렬화 벤치마크
func BenchmarkSerialization(b *testing.B) {
	serializer := NewSerializer(nil)
	sm := NewSnapshotManager(nil)
	
	screen := sm.CaptureScreen("bench-session", 24, 80)
	snapshot, _ := sm.CreateSnapshot("bench-session", screen)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		data, _ := serializer.SerializeSnapshot(snapshot)
		serializer.DeserializeSnapshot(data)
	}
}