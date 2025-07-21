---
task_id: T04_S03
sprint_sequence_id: S03_M02
status: open
complexity: Medium
last_updated: 2025-07-21T16:00:00Z
---

# Task: SQLite Storage Implementation

## Description
SQLite 데이터베이스를 사용하는 스토리지 구현체를 개발합니다. 모든 CRUD 작업을 구현하고, 트랜잭션을 지원하며, 기존 스토리지 인터페이스와 완벽하게 호환되도록 합니다.

## Goal / Objectives
- SQLite 연결 관리 및 초기화
- 모든 엔티티(Workspace, Project, Session, Task)의 CRUD 구현
- Prepared Statement 사용으로 성능 최적화
- 트랜잭션 지원
- 에러 처리 및 로깅

## Acceptance Criteria
- [ ] 모든 스토리지 인터페이스 메서드 구현 완료
- [ ] 트랜잭션 처리 정상 작동
- [ ] Prepared Statement로 SQL 인젝션 방지
- [ ] 연결 풀링 구현
- [ ] 모든 단위 테스트 통과

## Subtasks
- [ ] SQLite 드라이버 설정 및 연결 관리
- [ ] WorkspaceStorage 구현
- [ ] ProjectStorage 구현
- [ ] SessionStorage 구현
- [ ] TaskStorage 구현
- [ ] 트랜잭션 래퍼 구현
- [ ] 통합 테스트 작성

## 기술 가이드

### 주요 인터페이스 및 통합 지점
- `internal/storage/interface.go` - 구현해야 할 인터페이스
- `internal/storage/memory/` - 참조할 기존 구현
- 사용할 드라이버: `github.com/mattn/go-sqlite3`

### SQLite 특화 구현 사항
```go
// internal/storage/sqlite/storage.go
type Storage struct {
    db         *sql.DB
    stmtCache  map[string]*sql.Stmt
    mu         sync.RWMutex
}

// 연결 설정
func New(dataSource string) (*Storage, error) {
    db, err := sql.Open("sqlite3", dataSource)
    // 연결 풀 설정
    db.SetMaxOpenConns(1) // SQLite는 단일 쓰기
    db.SetMaxIdleConns(1)
    
    // WAL 모드 활성화
    db.Exec("PRAGMA journal_mode=WAL")
    db.Exec("PRAGMA foreign_keys=ON")
}
```

### Prepared Statement 패턴
```go
// 성능을 위한 statement 캐싱
func (s *Storage) prepareStmt(query string) (*sql.Stmt, error) {
    s.mu.RLock()
    if stmt, ok := s.stmtCache[query]; ok {
        s.mu.RUnlock()
        return stmt, nil
    }
    s.mu.RUnlock()
    
    // 새 statement 준비 및 캐싱
}
```

### 페이지네이션 구현
```go
// LIMIT과 OFFSET 사용
query := `
    SELECT * FROM workspaces 
    WHERE owner_id = ? AND deleted_at IS NULL
    ORDER BY created_at DESC
    LIMIT ? OFFSET ?
`
```

### 트랜잭션 처리
```go
type sqliteTx struct {
    tx *sql.Tx
    storage *Storage
}

func (s *Storage) BeginTx(ctx context.Context) (storage.Transaction, error) {
    tx, err := s.db.BeginTx(ctx, nil)
    // 트랜잭션 래퍼 반환
}
```

## 구현 노트

### 파일 구조
```
internal/storage/sqlite/
├── storage.go          # 메인 스토리지 구현
├── workspace.go        # WorkspaceStorage 구현
├── project.go          # ProjectStorage 구현  
├── session.go          # SessionStorage 구현
├── task.go             # TaskStorage 구현
├── transaction.go      # 트랜잭션 래퍼
├── utils.go            # 공통 유틸리티
└── errors.go           # SQLite 에러 변환
```

### 에러 변환
- `UNIQUE constraint failed` → `ErrAlreadyExists`
- `no rows in result set` → `ErrNotFound`
- 기타 SQLite 에러 → 적절한 스토리지 에러로 매핑

### 성능 고려사항
- Batch INSERT 사용
- 인덱스 활용 쿼리 작성
- N+1 쿼리 방지
- 적절한 트랜잭션 범위 설정

## Output Log
*(작업 진행 시 업데이트)*