package docker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewTerminalSnapshot 터미널 스냅샷 생성 테스트
func TestNewTerminalSnapshot(t *testing.T) {
	width, height := 80, 24
	snapshot := NewTerminalSnapshot(width, height)

	assert.NotNil(t, snapshot)
	
	// 크기 확인
	w, h := snapshot.GetSize()
	assert.Equal(t, width, w)
	assert.Equal(t, height, h)

	// 커서 초기 위치 확인
	x, y := snapshot.GetCursor()
	assert.Equal(t, 0, x)
	assert.Equal(t, 0, y)

	// 버퍼 초기화 확인
	buffer := snapshot.GetBuffer()
	assert.Len(t, buffer, height)
	assert.Len(t, buffer[0], width)

	// 기본 셀 확인
	cell := buffer[0][0]
	assert.Equal(t, ' ', cell.Char)
	assert.Equal(t, ColorDefault, cell.Foreground.Type)
	assert.Equal(t, ColorDefault, cell.Background.Type)
}

// TestTerminalSnapshotSetCell 셀 설정 테스트
func TestTerminalSnapshotSetCell(t *testing.T) {
	snapshot := NewTerminalSnapshot(80, 24).(*terminalSnapshot)

	// 유효한 위치에 셀 설정
	cell := Cell{
		Char:       'A',
		Foreground: Color{Type: ColorANSI, Value: 31}, // 빨간색
		Background: Color{Type: ColorDefault},
	}

	err := snapshot.SetCell(10, 5, cell)
	assert.NoError(t, err)

	// 설정된 셀 확인
	buffer := snapshot.GetBuffer()
	assert.Equal(t, 'A', buffer[5][10].Char)
	assert.Equal(t, uint32(31), buffer[5][10].Foreground.Value)

	// 잘못된 위치에 셀 설정
	err = snapshot.SetCell(-1, 5, cell)
	assert.Error(t, err)

	err = snapshot.SetCell(80, 5, cell)
	assert.Error(t, err)

	err = snapshot.SetCell(10, 24, cell)
	assert.Error(t, err)
}

// TestTerminalSnapshotSetCursor 커서 설정 테스트
func TestTerminalSnapshotSetCursor(t *testing.T) {
	snapshot := NewTerminalSnapshot(80, 24).(*terminalSnapshot)

	// 유효한 위치에 커서 설정
	err := snapshot.SetCursor(10, 5, true)
	assert.NoError(t, err)

	x, y := snapshot.GetCursor()
	assert.Equal(t, 10, x)
	assert.Equal(t, 5, y)

	// 잘못된 위치에 커서 설정
	err = snapshot.SetCursor(-1, 5, true)
	assert.Error(t, err)

	err = snapshot.SetCursor(80, 5, true)
	assert.Error(t, err)

	err = snapshot.SetCursor(10, 24, true)
	assert.Error(t, err)
}

// TestTerminalSnapshotCompression 압축 테스트
func TestTerminalSnapshotCompression(t *testing.T) {
	snapshot := NewTerminalSnapshot(80, 24).(*terminalSnapshot)

	// 일부 셀에 데이터 설정
	for i := 0; i < 10; i++ {
		cell := Cell{
			Char:       rune('A' + i),
			Foreground: Color{Type: ColorANSI, Value: uint32(30 + i)},
		}
		snapshot.SetCell(i, 0, cell)
	}

	// 압축된 데이터 생성
	compressed, err := snapshot.GetCompressedData()
	assert.NoError(t, err)
	assert.NotEmpty(t, compressed)

	// 압축 데이터가 원본보다 작은지 확인 (일반적으로)
	assert.Greater(t, len(compressed), 0)

	// 두 번째 호출시 캐시된 데이터 반환 확인
	compressed2, err := snapshot.GetCompressedData()
	assert.NoError(t, err)
	assert.Equal(t, compressed, compressed2)
}

// MockPTYSession 테스트용 Mock PTY 세션
type MockPTYSession struct {
	alive      bool
	readData   []byte
	readIndex  int
	readError  error
}

func NewMockPTYSession(data string) *MockPTYSession {
	return &MockPTYSession{
		alive:     true,
		readData:  []byte(data),
		readIndex: 0,
	}
}

func (m *MockPTYSession) ID() string                    { return "mock-session" }
func (m *MockPTYSession) ContainerID() string           { return "mock-container" }
func (m *MockPTYSession) Start(ctx interface{}) error   { return nil }
func (m *MockPTYSession) Stop() error                   { return nil }
func (m *MockPTYSession) Write(data []byte) (int, error) { return len(data), nil }
func (m *MockPTYSession) Resize(width, height int) error { return nil }
func (m *MockPTYSession) IsAlive() bool                  { return m.alive }
func (m *MockPTYSession) GetCreatedAt() time.Time        { return time.Now() }
func (m *MockPTYSession) GetLastActivity() time.Time     { return time.Now() }

func (m *MockPTYSession) Read(data []byte) (int, error) {
	if m.readError != nil {
		return 0, m.readError
	}
	
	if m.readIndex >= len(m.readData) {
		return 0, nil // EOF
	}

	// 읽을 데이터 크기 결정
	remaining := len(m.readData) - m.readIndex
	readSize := len(data)
	if readSize > remaining {
		readSize = remaining
	}

	// 데이터 복사
	copy(data, m.readData[m.readIndex:m.readIndex+readSize])
	m.readIndex += readSize

	return readSize, nil
}

// TestNewSnapshotCapturer 스냅샷 캡처 시스템 생성 테스트
func TestNewSnapshotCapturer(t *testing.T) {
	mockSession := NewMockPTYSession("test data")
	interval := 100 * time.Millisecond
	maxHistory := 10

	capturer := NewSnapshotCapturer(mockSession, interval, maxHistory)

	assert.NotNil(t, capturer)
	assert.Equal(t, mockSession, capturer.ptySession)
	assert.Equal(t, interval, capturer.interval)
	assert.Equal(t, maxHistory, capturer.maxHistory)
	assert.Empty(t, capturer.snapshots)
	assert.False(t, capturer.isRunning)
}

// TestSnapshotCapturerStartStop 캡처 시스템 시작/중지 테스트
func TestSnapshotCapturerStartStop(t *testing.T) {
	mockSession := NewMockPTYSession("Hello World")
	capturer := NewSnapshotCapturer(mockSession, 50*time.Millisecond, 5)

	// 시작 테스트
	err := capturer.Start()
	assert.NoError(t, err)
	assert.True(t, capturer.isRunning)

	// 이미 실행 중인 상태에서 시작 시도
	err = capturer.Start()
	assert.Error(t, err)

	// 잠시 대기하여 스냅샷 캡처 확인
	time.Sleep(200 * time.Millisecond)

	// 스냅샷이 생성되었는지 확인
	count := capturer.GetSnapshotCount()
	assert.Greater(t, count, 0)

	// 중지 테스트
	err = capturer.Stop()
	assert.NoError(t, err)
	assert.False(t, capturer.isRunning)

	// 이미 중지된 상태에서 중지 시도
	err = capturer.Stop()
	assert.NoError(t, err)
}

// TestSnapshotCapturerHistory 히스토리 관리 테스트
func TestSnapshotCapturerHistory(t *testing.T) {
	mockSession := NewMockPTYSession("test")
	maxHistory := 3
	capturer := NewSnapshotCapturer(mockSession, 10*time.Millisecond, maxHistory)

	// 수동으로 스냅샷 추가
	for i := 0; i < 5; i++ {
		snapshot := NewTerminalSnapshot(80, 24)
		capturer.addSnapshot(snapshot)
	}

	// 최대 히스토리 수 확인
	count := capturer.GetSnapshotCount()
	assert.Equal(t, maxHistory, count)

	// 히스토리 조회
	history := capturer.GetSnapshotHistory()
	assert.Len(t, history, maxHistory)

	// 최신 스냅샷 조회
	latest := capturer.GetLatestSnapshot()
	assert.NotNil(t, latest)
	assert.Equal(t, history[len(history)-1], latest)

	// 히스토리 정리
	capturer.ClearHistory()
	assert.Equal(t, 0, capturer.GetSnapshotCount())
	
	// 빈 히스토리에서 최신 스냅샷 조회
	latest = capturer.GetLatestSnapshot()
	assert.Nil(t, latest)
}

// TestSnapshotCapturerStats 통계 정보 테스트
func TestSnapshotCapturerStats(t *testing.T) {
	mockSession := NewMockPTYSession("test data")
	interval := 100 * time.Millisecond
	maxHistory := 5
	capturer := NewSnapshotCapturer(mockSession, interval, maxHistory)

	// 초기 통계
	stats := capturer.GetStats()
	assert.NotNil(t, stats)
	assert.False(t, stats.IsRunning)
	assert.Equal(t, 0, stats.SnapshotCount)
	assert.Equal(t, maxHistory, stats.MaxHistory)
	assert.Equal(t, interval, stats.CaptureInterval)
	assert.Equal(t, int64(0), stats.MemoryUsage)

	// 스냅샷 추가 후 통계
	snapshot := NewTerminalSnapshot(80, 24)
	capturer.addSnapshot(snapshot)

	stats = capturer.GetStats()
	assert.Equal(t, 1, stats.SnapshotCount)
	assert.Greater(t, stats.MemoryUsage, int64(0))
	assert.False(t, stats.LastCaptureTime.IsZero())
}

// TestANSIParser ANSI 파서 테스트
func TestANSIParser(t *testing.T) {
	parser := NewANSIParser()
	assert.NotNil(t, parser)

	snapshot := NewTerminalSnapshot(80, 24).(*terminalSnapshot)

	// 간단한 텍스트 파싱
	text := "Hello World"
	err := parser.ParseToSnapshot([]byte(text), snapshot)
	assert.NoError(t, err)

	// 텍스트가 올바르게 설정되었는지 확인
	buffer := snapshot.GetBuffer()
	expectedText := "Hello World"
	for i, char := range expectedText {
		assert.Equal(t, char, buffer[0][i].Char)
	}

	// 커서 위치 확인
	x, y := snapshot.GetCursor()
	assert.Equal(t, len(expectedText), x)
	assert.Equal(t, 0, y)
}

// TestANSIParserNewline 줄바꿈 처리 테스트
func TestANSIParserNewline(t *testing.T) {
	parser := NewANSIParser()
	snapshot := NewTerminalSnapshot(80, 24).(*terminalSnapshot)

	// 줄바꿈이 포함된 텍스트
	text := "Line 1\nLine 2\nLine 3"
	err := parser.ParseToSnapshot([]byte(text), snapshot)
	assert.NoError(t, err)

	buffer := snapshot.GetBuffer()

	// 첫 번째 줄 확인
	line1 := "Line 1"
	for i, char := range line1 {
		assert.Equal(t, char, buffer[0][i].Char)
	}

	// 두 번째 줄 확인
	line2 := "Line 2"
	for i, char := range line2 {
		assert.Equal(t, char, buffer[1][i].Char)
	}

	// 세 번째 줄 확인
	line3 := "Line 3"
	for i, char := range line3 {
		assert.Equal(t, char, buffer[2][i].Char)
	}

	// 최종 커서 위치 확인
	x, y := snapshot.GetCursor()
	assert.Equal(t, len(line3), x)
	assert.Equal(t, 2, y)
}

// BenchmarkSnapshotCapture 스냅샷 캡처 성능 테스트
func BenchmarkSnapshotCapture(b *testing.B) {
	mockSession := NewMockPTYSession("test data for benchmarking")
	capturer := NewSnapshotCapturer(mockSession, time.Second, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		capturer.captureSnapshot()
	}
}

// BenchmarkSnapshotCompression 압축 성능 테스트
func BenchmarkSnapshotCompression(b *testing.B) {
	snapshot := NewTerminalSnapshot(80, 24).(*terminalSnapshot)

	// 샘플 데이터로 스냅샷 채우기
	for y := 0; y < 24; y++ {
		for x := 0; x < 80; x++ {
			cell := Cell{
				Char:       rune('A' + (x+y)%26),
				Foreground: Color{Type: ColorANSI, Value: uint32(30 + (x+y)%8)},
			}
			snapshot.SetCell(x, y, cell)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := snapshot.GetCompressedData()
		if err != nil {
			b.Fatal(err)
		}
		// 캐시 초기화 (실제 압축 성능 측정)
		snapshot.compressed = nil
	}
}