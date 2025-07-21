---
task_id: T08_S03
sprint_sequence_id: S03_M02
status: closed
complexity: Medium
last_updated: 2025-07-21T16:30:00Z
completed_at: 2025-07-21T16:30:00Z
---

# Task: Query Optimization and Performance

## Description
데이터베이스 쿼리 성능을 최적화하고 모니터링 시스템을 구축합니다. 인덱스 전략을 수립하고, 쿼리 분석 도구를 통합하며, 성능 메트릭을 수집합니다.

## Goal / Objectives
- 최적화된 인덱스 생성 및 관리
- 쿼리 성능 분석 도구 통합
- N+1 쿼리 문제 해결
- 쿼리 캐싱 전략 구현
- 성능 모니터링 대시보드

## Acceptance Criteria
- [x] 모든 주요 쿼리에 적절한 인덱스 존재
- [x] 쿼리 실행 시간 로깅 시스템 구현
- [x] Slow 쿼리 감지 및 알림
- [x] 쿼리 실행 계획 분석 가능
- [x] 성능 벤치마크 테스트 작성

## Subtasks
- [x] 쿼리 패턴 분석 및 인덱스 설계
- [x] 쿼리 성능 모니터링 시스템 구현
- [x] 쿼리 빌더 최적화
- [x] 캐싱 레이어 구현
- [x] 벤치마크 테스트 작성
- [x] 성능 튜닝 가이드 문서화

## 기술 가이드

### 주요 인터페이스 및 통합 지점
- SQLite: EXPLAIN QUERY PLAN 활용
- BoltDB: 버킷 구조 최적화
- 모니터링: Prometheus 메트릭 수집

### 인덱스 전략
```sql
-- SQLite 인덱스 생성
-- 워크스페이스 조회 최적화
CREATE INDEX idx_workspaces_owner_status 
ON workspaces(owner_id, status) 
WHERE deleted_at IS NULL;

-- 프로젝트 조회 최적화
CREATE INDEX idx_projects_workspace_status 
ON projects(workspace_id, status) 
WHERE deleted_at IS NULL;

-- 세션 조회 최적화
CREATE INDEX idx_sessions_project_status 
ON sessions(project_id, status);

-- 태스크 시간 기반 조회
CREATE INDEX idx_tasks_session_time 
ON tasks(session_id, created_at DESC);
```

### 쿼리 모니터링
```go
// internal/storage/monitoring/query.go
type QueryMonitor struct {
    threshold time.Duration
    logger    *zap.Logger
    metrics   *prometheus.HistogramVec
}

func (m *QueryMonitor) Wrap(query string, fn func() error) error {
    start := time.Now()
    
    err := fn()
    
    duration := time.Since(start)
    m.metrics.WithLabelValues(query).Observe(duration.Seconds())
    
    if duration > m.threshold {
        m.logger.Warn("slow query detected",
            zap.String("query", query),
            zap.Duration("duration", duration),
        )
    }
    
    return err
}
```

### 쿼리 빌더 최적화
```go
// internal/storage/query/builder.go
type QueryBuilder struct {
    selections []string
    joins      []string
    conditions []string
    params     []interface{}
}

// Eager Loading으로 N+1 방지
func (b *QueryBuilder) WithProjects() *QueryBuilder {
    b.joins = append(b.joins, 
        "LEFT JOIN projects ON projects.workspace_id = workspaces.id")
    return b
}

// 결과
query := builder.
    Select("workspaces.*", "projects.*").
    WithProjects().
    Where("workspaces.owner_id = ?", ownerID).
    Build()
```

### 캐싱 전략
```go
// internal/storage/cache/cache.go
type Cache interface {
    Get(key string, dest interface{}) error
    Set(key string, value interface{}, ttl time.Duration) error
    Delete(key string) error
    Clear() error
}

// 쿼리 결과 캐싱
func (s *Storage) GetWorkspaceWithCache(ctx context.Context, id string) (*models.Workspace, error) {
    key := fmt.Sprintf("workspace:%s", id)
    
    var ws models.Workspace
    if err := s.cache.Get(key, &ws); err == nil {
        return &ws, nil
    }
    
    // 캐시 미스 - DB 조회
    result, err := s.getWorkspaceFromDB(ctx, id)
    if err == nil {
        s.cache.Set(key, result, 5*time.Minute)
    }
    
    return result, err
}
```

## 구현 노트

### 성능 메트릭
```go
// Prometheus 메트릭
var (
    queryDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "aicli_query_duration_seconds",
            Help: "Query execution duration",
        },
        []string{"query_type", "table"},
    )
    
    queryCacheHits = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "aicli_query_cache_hits_total",
            Help: "Query cache hit count",
        },
        []string{"cache_type"},
    )
)
```

### 벤치마크 테스트
```go
// internal/storage/benchmark_test.go
func BenchmarkWorkspaceList(b *testing.B) {
    // 대량 데이터 준비
    // 다양한 쿼리 패턴 테스트
    // 인덱스 유무 비교
}
```

### 최적화 체크리스트
- [ ] 복합 인덱스 순서 최적화
- [ ] 불필요한 SELECT * 제거
- [ ] JOIN vs 서브쿼리 성능 비교
- [ ] 트랜잭션 범위 최소화
- [ ] 배치 작업 크기 조정

## Output Log

### 2025-07-21 구현 완료

#### 1. 쿼리 모니터링 시스템 구현
- **패키지**: `internal/storage/monitoring/`
- **파일들**:
  - `query.go`: 쿼리 실행 시간 측정, Slow 쿼리 감지, Prometheus 메트릭 수집
  - `analyzer.go`: SQLite 쿼리 실행 계획 분석, 최적화 제안 생성
  - `monitoring.go`: 통합 모니터링 시스템, 모니터링 실행기

**주요 기능**:
- 쿼리 실행 시간 측정 및 로깅
- 설정 가능한 Slow 쿼리 임계값 (기본: 100ms)
- Prometheus 메트릭 수집 (실행 시간, 에러 수, 캐시 히트율)
- SQLite EXPLAIN QUERY PLAN 분석
- 자동 최적화 제안 생성

#### 2. SQLite 인덱스 관리 시스템 구현
- **파일**: `internal/storage/sqlite/indexes.go`
- **통합**: `storage.go`에 IndexManager 및 최적화 메서드 추가

**주요 기능**:
- 동적 인덱스 생성/삭제/분석
- 테이블별 맞춤형 인덱스 제안
- 복합 인덱스 최적화 (owner_id + status 등)
- 부분 인덱스 지원 (WHERE deleted_at IS NULL)
- 인덱스 재구축 및 통계 수집

#### 3. BoltDB 최적화 시스템 구현
- **파일**: `internal/storage/boltdb/batch.go`
- **통합**: `storage.go`에 배치 처리기 및 대량 가져오기 도구 통합

**주요 기능**:
- 배치 처리기 (1000개 단위 배치, 1초 플러시 간격)
- 대량 데이터 가져오기 (5000개 단위 처리)
- 비동기 배치 쓰기 (2개 워커 고루틴)
- 인덱스 동시 업데이트
- 배치 통계 및 모니터링

#### 4. 쿼리 캐싱 시스템 구현
- **패키지**: `internal/storage/cache/`
- **파일들**:
  - `cache.go`: 인메모리 캐시 구현, LRU 제거, TTL 지원
  - `query.go`: 쿼리 특화 캐시, 엔티티별 캐시 관리

**주요 기능**:
- 인메모리 LRU 캐시 (64MB, 10,000개 항목 제한)
- 엔티티별 차별화된 TTL (워크스페이스: 10분, 태스크: 1분)
- 자동 만료 및 정리 (1분마다)
- 캐시 히트/미스 통계
- 패턴 기반 무효화 지원

#### 5. 쿼리 빌더 개선 및 최적화
- **패키지**: `internal/storage/query/`
- **파일들**:
  - `builder.go`: 고급 SQL 쿼리 빌더, 엔티티별 특화 빌더
  - `optimizer.go`: 쿼리 최적화 분석기, 자동 최적화 제안

**주요 기능**:
- 체인 방식 쿼리 빌더 (SELECT, JOIN, WHERE, ORDER BY 등)
- 엔티티별 특화 빌더 (WorkspaceQueryBuilder 등)
- 쿼리 최적화 분석 (SELECT *, N+1 쿼리, 카티젼 곱 감지)
- 자동 최적화 제안 (인덱스, 쿼리 재작성)
- 예상 성능 향상 계산

#### 6. 성능 벤치마크 테스트 작성
- **파일**: `internal/storage/benchmark_test.go`
- **범위**: 메모리/SQLite/BoltDB 스토리지 비교

**테스트 케이스**:
- 기본 CRUD 작업 벤치마크
- 동시성 테스트 (10개 고루틴)
- 스트레스 테스트 (10,000+ 레코드)
- 캐시 성능 테스트
- 모니터링 오버헤드 측정

#### 7. 배치 작업 지원 구현
- **파일**: `internal/storage/batch/scheduler.go`

**주요 기능**:
- 작업 스케줄러 (4개 워커, 100개 큐 크기)
- 진행률 추적 및 상태 관리
- 기본 작업 핸들러 (데이터 정리, 인덱스 재구축, 백업)
- 작업 통계 및 완료된 작업 자동 정리

### 성능 향상 결과 (예상)
- **SQLite 쿼리 성능**: 인덱스 최적화로 30-50% 향상
- **BoltDB 쓰기 성능**: 배치 처리로 10-20배 향상  
- **캐시 히트율**: 80% 이상 예상
- **Slow 쿼리 감소**: 모니터링 및 최적화 제안으로 90% 이상 감소

### 구현된 파일 목록
```
internal/storage/monitoring/
├── query.go          # 쿼리 모니터링 시스템
├── analyzer.go       # 쿼리 분석 도구  
└── monitoring.go     # 통합 모니터링

internal/storage/sqlite/
└── indexes.go        # SQLite 인덱스 관리

internal/storage/boltdb/
└── batch.go          # BoltDB 배치 처리

internal/storage/cache/
├── cache.go          # 인메모리 캐시
└── query.go          # 쿼리 캐시

internal/storage/query/
├── builder.go        # 쿼리 빌더
└── optimizer.go      # 쿼리 최적화

internal/storage/batch/
└── scheduler.go      # 배치 작업 스케줄러

internal/storage/
└── benchmark_test.go # 성능 벤치마크 테스트
```

모든 Acceptance Criteria와 Subtask가 성공적으로 완료되었습니다.