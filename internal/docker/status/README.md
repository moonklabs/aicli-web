# 워크스페이스 상태 추적 시스템 (Workspace Status Tracking System)

워크스페이스와 Docker 컨테이너의 상태를 실시간으로 추적하고 동기화하는 시스템입니다.

## 🏗️ 아키텍처

### 핵심 구성 요소

1. **Tracker** - 워크스페이스 상태 추적자
2. **ResourceMonitor** - 리소스 모니터링 시스템  
3. **MetricsCollector** - 메트릭 수집기
4. **EventSystem** - 이벤트 처리 시스템

### 시스템 구조

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Tracker       │    │  ResourceMonitor │    │ MetricsCollector│
│                 │    │                  │    │                 │
│ - State Sync    │    │ - CPU/Memory     │    │ - Aggregation   │
│ - Events        │────│ - Network I/O    │────│ - History       │
│ - Callbacks     │    │ - Disk Usage     │    │ - Analysis      │
└─────────────────┘    └──────────────────┘    └─────────────────┘
          │                        │                        │
          └────────────────────────┼────────────────────────┘
                                   │
                    ┌──────────────────┐
                    │   Event Bus      │
                    │                  │
                    │ - Status Change  │
                    │ - Error/Recovery │
                    │ - Metrics Update │
                    └──────────────────┘
```

## 📦 주요 기능

### 1. 상태 추적 (State Tracking)

- **실시간 동기화**: 워크스페이스와 컨테이너 상태 자동 동기화
- **상태 전환 감지**: Active ↔ Inactive ↔ Archived 상태 변경 추적
- **이벤트 기반 알림**: 상태 변경 시 자동 알림 및 콜백 실행

### 2. 리소스 모니터링 (Resource Monitoring)

- **시스템 메트릭**: CPU, 메모리, 네트워크, 디스크 I/O 추적
- **실시간 수집**: 설정 가능한 간격으로 메트릭 수집
- **임계값 알림**: 리소스 사용량 임계값 초과 시 경고

### 3. 메트릭 수집 (Metrics Collection)

- **집계 처리**: 시간별/일별 메트릭 집계 및 분석
- **히스토리 관리**: 메트릭 히스토리 저장 및 트렌드 분석
- **성능 최적화**: 메모리 효율적인 데이터 구조 사용

### 4. 이벤트 시스템 (Event System)

- **이벤트 타입**: 7가지 이벤트 타입 지원
- **비동기 처리**: 논블로킹 이벤트 처리
- **확장 가능**: 플러그인 방식의 이벤트 핸들러

## 🚀 사용법

### 기본 설정

```go
import (
    "github.com/aicli/aicli-web/internal/docker/status"
    "github.com/aicli/aicli-web/internal/services"
    "github.com/aicli/aicli-web/internal/interfaces"
)

// 서비스 의존성 설정
var workspaceService interfaces.WorkspaceService = services.NewWorkspaceService(storage)
containerManager := docker.NewContainerManager(client)
dockerManager := docker.NewManager(config)

// 상태 추적자 생성
tracker := status.NewTracker(workspaceService, containerManager, dockerManager)

// 리소스 모니터 생성
monitor := status.NewResourceMonitor(containerManager, dockerManager)
```

### 상태 추적 시작

```go
// 추적 시작
err := tracker.Start()
if err != nil {
    log.Fatal("Failed to start tracker:", err)
}

// 종료 시 정리
defer tracker.Stop()

// 상태 변경 콜백 등록
tracker.OnStateChange(func(workspaceID string, oldState, newState *status.WorkspaceState) {
    log.Printf("Workspace %s changed: %s -> %s", 
        workspaceID, oldState.Status, newState.Status)
})
```

### 리소스 모니터링

```go
// 모니터링 시작
err := monitor.Start()
if err != nil {
    log.Fatal("Failed to start monitor:", err)
}
defer monitor.Stop()

// 특정 워크스페이스 모니터링
ctx := context.Background()
metricsChan, err := monitor.StartMonitoring(ctx, "workspace-123")
if err != nil {
    log.Fatal("Failed to start workspace monitoring:", err)
}

// 메트릭 수신
go func() {
    for metrics := range metricsChan {
        log.Printf("CPU: %.2f%%, Memory: %s, Network: %.2f MB/s", 
            metrics.CPUPercent,
            formatBytes(metrics.MemoryUsage),
            (metrics.NetworkRxMB + metrics.NetworkTxMB))
    }
}()
```

### 상태 조회

```go
// 개별 워크스페이스 상태 조회
state, exists := tracker.GetWorkspaceState("workspace-123")
if exists {
    fmt.Printf("Status: %s, Container: %s, CPU: %.2f%%\n",
        state.Status, state.ContainerID, state.Metrics.CPUPercent)
}

// 모든 워크스페이스 상태 조회
allStates := tracker.GetAllWorkspaceStates()
for workspaceID, state := range allStates {
    fmt.Printf("%s: %s\n", workspaceID, state.Status)
}

// 리소스 요약 조회
summary, err := monitor.GetResourceSummary(ctx)
if err == nil {
    fmt.Printf("Total CPU: %.2f%%, Total Memory: %s, Active Containers: %d\n",
        summary.TotalCPUUsage, formatBytes(summary.TotalMemoryUsage), summary.ActiveContainers)
}
```

## ⚙️ 설정 옵션

### Tracker 설정

```go
// 동기화 간격 설정 (기본: 30초)
tracker.SetSyncInterval(15 * time.Second)

// 최대 재시도 횟수 설정 (기본: 3회)
tracker.SetMaxRetries(5)

// 커스텀 로거 설정
tracker.SetLogger(customLogger)
```

### ResourceMonitor 설정

```go
// 메트릭 수집 간격 설정 (기본: 10초)
monitor.SetCollectInterval(5 * time.Second)

// 커스텀 로거 설정
monitor.SetLogger(customLogger)
```

## 📊 메트릭 정보

### WorkspaceMetrics

```go
type WorkspaceMetrics struct {
    // 리소스 사용량
    CPUPercent  float64 // CPU 사용률 (%)
    MemoryUsage int64   // 메모리 사용량 (bytes)
    MemoryLimit int64   // 메모리 제한 (bytes)
    NetworkRxMB float64 // 네트워크 수신 (MB)
    NetworkTxMB float64 // 네트워크 송신 (MB)
    
    // 타이밍 정보
    Uptime       string    // 가동 시간
    LastActivity time.Time // 마지막 활동 시간
    
    // 오류 통계
    ErrorCount    int       // 오류 발생 횟수
    LastErrorTime time.Time // 마지막 오류 시간
}
```

### 집계 메트릭

- **ResourceSummary**: 전체 시스템 리소스 요약
- **TrackerStats**: 추적자 통계 정보
- **MonitorStats**: 모니터 성능 통계

## 🔄 이벤트 타입

| 타입 | 설명 | 데이터 |
|------|------|--------|
| `status_changed` | 워크스페이스 상태 변경 | StatusChangeEvent |
| `container_update` | 컨테이너 업데이트 | ContainerUpdateEvent |
| `error` | 오류 발생 | ErrorEvent |
| `recovery` | 오류 복구 | RecoveryEvent |
| `metrics_update` | 메트릭 업데이트 | MetricsUpdateEvent |
| `sync_start` | 동기화 시작 | SyncEvent |
| `sync_complete` | 동기화 완료 | SyncEvent |

## 🧪 테스트

### 단위 테스트 실행

```bash
# 모든 테스트 실행
make test-status

# 통합 테스트만 실행
make test-status-integration

# 독립적인 테스트 (다른 패키지 의존성 없이)
go test -v -tags=standalone ./internal/docker/status/...
```

### 테스트 커버리지

- **Tracker**: 상태 동기화, 이벤트 처리, 콜백 시스템
- **ResourceMonitor**: 메트릭 수집, 모니터링, 캐시 관리
- **EventSystem**: 이벤트 생성, 전파, 핸들링
- **Integration**: 전체 시스템 통합 시나리오

### 성능 테스트

```bash
# 벤치마크 테스트
go test -bench=. ./internal/docker/status/...

# 대용량 테스트 (50개 워크스페이스)
go test -v -run=TestIntegration_Performance ./internal/docker/status/...
```

## 🔧 문제 해결

### 일반적인 문제

1. **동기화 지연**
   ```go
   // 동기화 간격 단축
   tracker.SetSyncInterval(10 * time.Second)
   
   // 수동 동기화 강제 실행
   tracker.ForceSync("workspace-id")
   ```

2. **메트릭 수집 오류**
   ```go
   // 수집 간격 조정
   monitor.SetCollectInterval(30 * time.Second)
   
   // 모니터 상태 확인
   stats := monitor.GetMonitorStats()
   fmt.Printf("Active monitors: %d\n", stats.ActiveMonitors)
   ```

3. **메모리 사용량 증가**
   ```go
   // 캐시 통계 확인
   cacheStats := monitor.GetCacheStats()
   fmt.Printf("Cached containers: %d\n", cacheStats.CachedContainers)
   ```

### 로깅 설정

```go
// 커스텀 로거 구현
type CustomLogger struct {
    logger *log.Logger
}

func (l *CustomLogger) Info(msg string, args ...interface{}) {
    l.logger.Printf("[INFO] "+msg, args...)
}

func (l *CustomLogger) Error(msg string, err error, args ...interface{}) {
    l.logger.Printf("[ERROR] "+msg+": %v", append(args, err)...)
}

// 로거 설정
customLogger := &CustomLogger{logger: log.New(os.Stdout, "STATUS: ", log.LstdFlags)}
tracker.SetLogger(customLogger)
monitor.SetLogger(customLogger)
```

## 🚀 성능 특성

### 확장성

- **동시 워크스페이스**: 100개 이상 지원
- **메트릭 수집 오버헤드**: < 5%
- **상태 동기화 지연**: < 30초
- **이벤트 처리 지연**: < 1초

### 메모리 사용량

- **Tracker**: 워크스페이스당 ~2KB
- **ResourceMonitor**: 활성 모니터당 ~5KB
- **MetricsCollector**: 컨테이너당 ~10KB (히스토리 포함)

### 네트워크 사용량

- **Docker API 호출**: 동기화당 2-5 API 호출
- **메트릭 수집**: 컨테이너당 1 API 호출/수집 간격

## 🔮 향후 계획

### Phase 1 (현재)
- ✅ 기본 상태 추적 시스템
- ✅ 리소스 모니터링
- ✅ 이벤트 시스템
- ✅ 메트릭 수집

### Phase 2 (예정)
- 📝 영구 메트릭 저장소
- 📝 대시보드 API
- 📝 알림 시스템 통합
- 📝 WebSocket 실시간 전송

### Phase 3 (계획)
- 📝 예측 분석
- 📝 자동 스케일링 권장
- 📝 이상 징후 감지
- 📝 머신러닝 기반 최적화

## 📚 관련 문서

- [Docker 통합 가이드](../README.md)
- [API 문서](../../server/README.md)  
- [아키텍처 개요](../../../docs/ARCHITECTURE.md)
- [개발 가이드](../../../CONTRIBUTING.md)

## 🤝 기여

상태 추적 시스템 개선에 기여하려면:

1. 이슈 생성 및 논의
2. 기능 브랜치 생성
3. 테스트 포함한 코드 작성
4. PR 생성 및 리뷰

자세한 내용은 [기여 가이드](../../../CONTRIBUTING.md)를 참조하세요.