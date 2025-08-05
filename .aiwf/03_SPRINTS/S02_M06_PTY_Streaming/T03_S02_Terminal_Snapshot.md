---
task_id: T03_S02_Terminal_Snapshot
sprint_id: S02_M06_PTY_Streaming
milestone_id: M06
title: 터미널 스냅샷 캡처 시스템 구현
type: implementation
complexity: Medium
status: pending
assignee: unassigned
created: 2025-08-05T10:00:00+0900
last_updated: 2025-08-05T10:00:00+0900
depends_on: [T02_S02_WebSocket_Streaming, T05_S02_ANSI_Parser]
blocks: [T07_S02_Performance_Optimization]
epic: PTY_Streaming_System
---

# Task: 터미널 스냅샷 캡처 시스템 구현

## Task Summary
1초 간격으로 터미널 화면 상태를 캡처하고 관리하는 시스템을 구현합니다. ANSI 이스케이프 시퀀스를 파싱하여 터미널의 실제 화면 상태를 추적하고, 압축된 스냅샷 히스토리를 제공합니다.

## Acceptance Criteria

### 기능 요구사항
- [ ] 1초 간격 터미널 화면 상태 자동 캡처
- [ ] ANSI 이스케이프 시퀀스 파싱을 통한 화면 상태 재구성
- [ ] 터미널 크기(rows/cols) 동적 조정 지원
- [ ] 커서 위치 및 속성 추적
- [ ] 색상 정보 (foreground/background) 보존
- [ ] 스크롤 백 히스토리 관리
- [ ] 스냅샷 압축 및 저장 최적화

### 성능 요구사항
- [ ] 스냅샷 캡처 시간 < 10ms
- [ ] 메모리 사용량 세션당 < 100MB
- [ ] 압축률 > 70% 달성
- [ ] 히스토리 조회 시간 < 50ms
- [ ] 동시 100개 세션 스냅샷 처리

### 안정성 요구사항
- [ ] 스냅샷 누락 없는 연속 캡처
- [ ] 메모리 누수 방지
- [ ] 오래된 스냅샷 자동 정리
- [ ] 파싱 오류 시 우아한 복구

## Implementation Details

### 1. 터미널 스냅샷 관리자 구조

```go
// internal/terminal/snapshot_manager.go
type SnapshotManager struct {
    sessions    map[string]*TerminalSession
    config      *SnapshotConfig
    storage     SnapshotStorage
    parser      ANSIParser
    compressor  SnapshotCompressor
    mutex       sync.RWMutex
    stopCh      chan struct{}
}

type TerminalSession struct {
    SessionID     string
    Screen        *TerminalScreen
    History       *SnapshotHistory
    LastSnapshot  time.Time
    Ticker        *time.Ticker
    Parser        *ANSIParser
    stopCh        chan struct{}
}

type TerminalScreen struct {
    Rows          int
    Cols          int
    Buffer        [][]Cell
    CursorX       int
    CursorY       int
    CursorVisible bool
    ScrollTop     int
    ScrollBottom  int
    Attributes    CellAttributes
    Title         string
}

type Cell struct {
    Char       rune
    Foreground Color
    Background Color
    Attributes CellAttributes
}
```

### 2. 스냅샷 캡처 시스템

```go
// 스냅샷 캡처 인터페이스
type SnapshotInterface interface {
    StartCapture(sessionID string, initialScreen *TerminalScreen) error
    StopCapture(sessionID string) error
    GetSnapshot(sessionID string, timestamp time.Time) (*Snapshot, error)
    GetLatestSnapshot(sessionID string) (*Snapshot, error)
    GetSnapshotHistory(sessionID string, limit int) ([]*Snapshot, error)
    UpdateScreen(sessionID string, data []byte) error
}

// 스냅샷 데이터 구조
type Snapshot struct {
    SessionID   string
    Timestamp   time.Time
    Screen      *TerminalScreen
    Compressed  []byte
    Checksum    string
    Sequence    uint64
}

// 스냅샷 설정
type SnapshotConfig struct {
    CaptureInterval   time.Duration
    MaxHistorySize    int
    CompressionLevel  int
    RetentionPeriod   time.Duration
    StorageType       string
}
```

### 3. ANSI 파싱 및 화면 상태 업데이트

```go
// ANSI 파싱을 통한 화면 업데이트
func (ts *TerminalSession) ProcessANSIData(data []byte) error {
    commands, err := ts.Parser.Parse(data)
    if err != nil {
        return fmt.Errorf("ANSI parsing failed: %w", err)
    }
    
    for _, cmd := range commands {
        switch cmd.Type {
        case ANSIText:
            ts.writeText(cmd.Text)
        case ANSICursorMove:
            ts.moveCursor(cmd.X, cmd.Y)
        case ANSIClearScreen:
            ts.clearScreen(cmd.Mode)
        case ANSISetColor:
            ts.setColor(cmd.Foreground, cmd.Background)
        case ANSIScrollRegion:
            ts.setScrollRegion(cmd.Top, cmd.Bottom)
        case ANSIInsertLine:
            ts.insertLines(cmd.Count)
        case ANSIDeleteLine:
            ts.deleteLines(cmd.Count)
        }
    }
    
    return nil
}

// 텍스트 쓰기 처리
func (ts *TerminalSession) writeText(text string) {
    for _, char := range text {
        if char == '\n' {
            ts.newLine()
        } else if char == '\r' {
            ts.carriageReturn()
        } else if char == '\t' {
            ts.tab()
        } else {
            ts.putChar(char)
        }
    }
}

// 문자 출력 처리
func (ts *TerminalSession) putChar(char rune) {
    if ts.Screen.CursorX >= ts.Screen.Cols {
        ts.newLine()
    }
    
    cell := Cell{
        Char:       char,
        Foreground: ts.Screen.Attributes.Foreground,
        Background: ts.Screen.Attributes.Background,
        Attributes: ts.Screen.Attributes,
    }
    
    ts.Screen.Buffer[ts.Screen.CursorY][ts.Screen.CursorX] = cell
    ts.Screen.CursorX++
}
```

### 4. 스냅샷 압축 및 저장

```go
// 스냅샷 압축 시스템
type SnapshotCompressor interface {
    Compress(snapshot *Snapshot) ([]byte, error)
    Decompress(data []byte) (*Snapshot, error)
    GetCompressionRatio() float64
}

type GzipCompressor struct {
    level int
}

func (gc *GzipCompressor) Compress(snapshot *Snapshot) ([]byte, error) {
    // 스냅샷을 JSON으로 직렬화
    jsonData, err := json.Marshal(snapshot.Screen)
    if err != nil {
        return nil, err
    }
    
    // Gzip 압축 적용
    var compressed bytes.Buffer
    writer, err := gzip.NewWriterLevel(&compressed, gc.level)
    if err != nil {
        return nil, err
    }
    
    if _, err := writer.Write(jsonData); err != nil {
        writer.Close()
        return nil, err
    }
    
    if err := writer.Close(); err != nil {
        return nil, err
    }
    
    return compressed.Bytes(), nil
}

// 차등 압축 (Delta Compression)
type DeltaCompressor struct {
    previousSnapshot *Snapshot
}

func (dc *DeltaCompressor) CompressDelta(current, previous *Snapshot) ([]byte, error) {
    changes := dc.computeChanges(current.Screen, previous.Screen)
    return json.Marshal(changes)
}
```

### 5. 스냅샷 히스토리 관리

```go
// 스냅샷 히스토리 관리
type SnapshotHistory struct {
    sessionID   string
    snapshots   []*Snapshot
    maxSize     int
    totalSize   int64
    mutex       sync.RWMutex
    storage     SnapshotStorage
}

func (sh *SnapshotHistory) AddSnapshot(snapshot *Snapshot) error {
    sh.mutex.Lock()
    defer sh.mutex.Unlock()
    
    // 용량 초과 시 오래된 스냅샷 제거
    if len(sh.snapshots) >= sh.maxSize {
        removed := sh.snapshots[0]
        sh.snapshots = sh.snapshots[1:]
        sh.totalSize -= int64(len(removed.Compressed))
        
        // 스토리지에서도 제거
        if err := sh.storage.DeleteSnapshot(removed.SessionID, removed.Timestamp); err != nil {
            return err
        }
    }
    
    sh.snapshots = append(sh.snapshots, snapshot)
    sh.totalSize += int64(len(snapshot.Compressed))
    
    return sh.storage.StoreSnapshot(snapshot)
}

func (sh *SnapshotHistory) GetSnapshotAt(timestamp time.Time) (*Snapshot, error) {
    sh.mutex.RLock()
    defer sh.mutex.RUnlock()
    
    // 이진 검색으로 가장 가까운 스냅샷 찾기
    idx := sort.Search(len(sh.snapshots), func(i int) bool {
        return sh.snapshots[i].Timestamp.After(timestamp)
    })
    
    if idx == 0 {
        return nil, fmt.Errorf("no snapshot found before timestamp")
    }
    
    return sh.snapshots[idx-1], nil
}
```

### 6. 주기적 캡처 및 정리 작업

```go
// 주기적 스냅샷 캡처
func (sm *SnapshotManager) startCaptureWorker(session *TerminalSession) {
    session.Ticker = time.NewTicker(sm.config.CaptureInterval)
    
    go func() {
        defer session.Ticker.Stop()
        
        for {
            select {
            case <-session.Ticker.C:
                if err := sm.captureSnapshot(session); err != nil {
                    log.Errorf("Failed to capture snapshot: %v", err)
                }
            case <-session.stopCh:
                return
            case <-sm.stopCh:
                return
            }
        }
    }()
}

func (sm *SnapshotManager) captureSnapshot(session *TerminalSession) error {
    snapshot := &Snapshot{
        SessionID: session.SessionID,
        Timestamp: time.Now(),
        Screen:    sm.cloneScreen(session.Screen),
        Sequence:  session.History.GetNextSequence(),
    }
    
    // 압축 수행
    compressed, err := sm.compressor.Compress(snapshot)
    if err != nil {
        return err
    }
    
    snapshot.Compressed = compressed
    snapshot.Checksum = sm.calculateChecksum(compressed)
    
    return session.History.AddSnapshot(snapshot)
}

// 정리 작업
func (sm *SnapshotManager) startCleanupWorker() {
    ticker := time.NewTicker(time.Hour)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            sm.cleanupExpiredSnapshots()
        case <-sm.stopCh:
            return
        }
    }
}
```

## 파일 구조

```
internal/terminal/
├── snapshot_manager.go    # 메인 스냅샷 관리자
├── terminal_session.go    # 터미널 세션 관리
├── screen.go             # 터미널 화면 상태
├── snapshot.go           # 스냅샷 데이터 구조
├── history.go            # 스냅샷 히스토리 관리
├── compressor.go         # 압축 시스템
├── storage.go            # 스냅샷 저장소
└── config.go             # 설정 관리

internal/terminal/ansi/
├── parser.go             # ANSI 파서 (T05에서 구현)
├── commands.go           # ANSI 명령어 정의
└── state.go              # 파서 상태 관리

internal/terminal/test/
├── snapshot_manager_test.go
├── screen_test.go
├── compressor_test.go
└── integration_test.go
```

## 핵심 구현 사항

### 1. 효율적인 메모리 관리
- 스크린 버퍼의 링 버퍼 구현
- 차등 압축을 통한 중복 데이터 제거
- 가비지 컬렉터 친화적인 객체 풀 사용

### 2. 실시간 성능 최적화
- 백그라운드 압축을 통한 캡처 지연 최소화
- 캐시를 활용한 빠른 히스토리 조회
- 비동기 I/O를 통한 스토리지 성능 향상

### 3. 데이터 무결성 보장
- 체크섬을 통한 압축 데이터 검증
- 트랜잭션 기반 스토리지 업데이트
- 복구 가능한 에러 처리

## Dependencies

### 필수 패키지
```go
import (
    "context"
    "sync"
    "time"
    "compress/gzip"
    "encoding/json"
    "crypto/sha256"
    "sort"
    
    // 로깅
    "github.com/sirupsen/logrus"
    
    // 메트릭
    "github.com/prometheus/client_golang/prometheus"
)
```

## 스토리지 인터페이스

```go
// 스냅샷 저장소 인터페이스
type SnapshotStorage interface {
    StoreSnapshot(snapshot *Snapshot) error
    LoadSnapshot(sessionID string, timestamp time.Time) (*Snapshot, error)
    DeleteSnapshot(sessionID string, timestamp time.Time) error
    ListSnapshots(sessionID string, limit int) ([]*Snapshot, error)
    GetStorageStats(sessionID string) (*StorageStats, error)
}

type StorageStats struct {
    TotalSnapshots   int64
    TotalSize        int64
    CompressionRatio float64
    OldestSnapshot   time.Time
    NewestSnapshot   time.Time
}
```

## 테스트 계획

### 단위 테스트
- ANSI 파싱 및 화면 업데이트 테스트
- 스냅샷 압축/해제 테스트
- 히스토리 관리 테스트
- 메모리 누수 테스트

### 통합 테스트
- 실제 터미널 출력과의 일치성 테스트
- 장시간 실행 안정성 테스트
- 다양한 터미널 크기 처리 테스트

### 성능 테스트
- 캡처 성능 벤치마크
- 압축률 및 압축 속도 측정
- 메모리 사용량 프로파일링

## Definition of Done
- [ ] 1초 간격 스냅샷 캡처 시스템 구현 완료
- [ ] ANSI 파싱을 통한 화면 상태 재구성 완료
- [ ] 압축 및 히스토리 관리 시스템 완료
- [ ] 단위 테스트 및 통합 테스트 통과
- [ ] 성능 요구사항 달성 확인
- [ ] 메모리 사용량 최적화 완료
- [ ] 코드 리뷰 완료

## Notes
- ANSI 파서는 T05_S02_ANSI_Parser에서 구현 예정
- 스냅샷 데이터는 향후 분석 도구에서 활용 가능
- 실시간 미리보기 기능을 위한 API 인터페이스 제공
- 스냅샷 데이터 암호화는 보안 요구사항에 따라 추가 고려