---
task_id: T08_S03
sprint_sequence_id: S03_M02
status: open
complexity: Medium
last_updated: 2025-07-21T16:00:00Z
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
- [ ] 모든 주요 쿼리에 적절한 인덱스 존재
- [ ] 쿼리 실행 시간 로깅 시스템 구현
- [ ] Slow 쿼리 감지 및 알림
- [ ] 쿼리 실행 계획 분석 가능
- [ ] 성능 벤치마크 테스트 작성

## Subtasks
- [ ] 쿼리 패턴 분석 및 인덱스 설계
- [ ] 쿼리 성능 모니터링 시스템 구현
- [ ] 쿼리 빌더 최적화
- [ ] 캐싱 레이어 구현
- [ ] 벤치마크 테스트 작성
- [ ] 성능 튜닝 가이드 문서화

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
*(작업 진행 시 업데이트)*