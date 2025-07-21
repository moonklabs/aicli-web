---
task_id: T04_S03
sprint_sequence_id: S03_M02
status: in_progress
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
- [x] SQLite 드라이버 설정 및 연결 관리
- [x] WorkspaceStorage 구현
- [x] ProjectStorage 구현
- [x] SessionStorage 구현
- [x] TaskStorage 구현
- [x] 트랜잭션 래퍼 구현 (기본 구조)
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

### 2025-07-21 16:00:00Z
**T04_S03 SQLite Storage Implementation 진행 중**

#### 완료된 작업

1. **SQLite 드라이버 설정 및 연결 관리 완료** (`internal/storage/sqlite/storage.go`)
   - SQLite 드라이버 설정 및 초기화
   - 연결 풀 설정 (MaxOpenConns: 1, MaxIdleConns: 1)
   - PRAGMA 옵션 설정 (WAL 모드, 외래키 제약조건 등)
   - Prepared Statement 캐싱 메커니즘
   - 연결 생명주기 관리 및 건강성 확인
   - Context 지원 실행 메서드 (`execContext`, `queryContext`, `queryRowContext`)

2. **WorkspaceStorage 구현 완료** (`internal/storage/sqlite/workspace.go`)
   - 모든 WorkspaceStorage 인터페이스 메서드 구현
   - CRUD 작업: Create, GetByID, GetByOwnerID, Update, Delete, List
   - 페이지네이션 지원 (LIMIT, OFFSET 사용)
   - Soft Delete 구현
   - 이름 중복 검사 (ExistsByName)
   - SQL 인젝션 방지 (Prepared Statement 사용)
   - 에러 변환 및 처리
   - NULL 값 안전한 처리 (sql.NullString, sql.NullTime)

3. **트랜잭션 래퍼 기본 구조 구현** (`internal/storage/sqlite/transaction.go`)
   - Transaction 인터페이스 구현
   - 트랜잭션 생명주기 관리 (Commit, Rollback)
   - 트랜잭션 상태 추적 (IsClosed)
   - 트랜잭션용 스토리지 래퍼 구조
   - WorkspaceStorage 트랜잭션 메서드 일부 구현

4. **의존성 추가** (`go.mod`)
   - github.com/mattn/go-sqlite3 v1.14.22 추가
   - go.etcd.io/bbolt v1.3.10 추가 (향후 BoltDB 지원용)

#### 주요 구현 특징

- **성능 최적화**: Prepared Statement 캐싱으로 반복 쿼리 성능 향상
- **동시성 안전성**: sync.RWMutex를 사용한 statement 캐시 보호  
- **SQLite 최적화**: WAL 모드, 메모리 매핑, 캐시 크기 등 성능 최적화
- **에러 처리**: 데이터베이스별 에러를 표준 storage 에러로 변환
- **확장성**: 트랜잭션 래퍼를 통한 ACID 보장
- **타입 안전성**: models 구조체와 완벽 호환

5. **ProjectStorage 구현 완료** (`internal/storage/sqlite/project.go`)
   - 모든 ProjectStorage 인터페이스 메서드 구현
   - 복잡한 구조체 JSON 직렬화 (ProjectConfig, GitInfo)
   - CRUD 작업: Create, GetByID, GetByWorkspaceID, Update, Delete
   - 경로 기반 조회 (GetByPath)
   - 이름 중복 검사 (ExistsByName)
   - 페이지네이션 지원
   - JSON 마샬링/언마샬링 헬퍼 메서드

6. **SessionStorage 구현 완료** (`internal/storage/sqlite/session.go`)
   - 모든 SessionStorage 인터페이스 메서드 구현
   - 세션 상태 관리 및 필터링
   - 메타데이터 JSON 직렬화 처리
   - 활성 세션 수 조회
   - Duration 타입 나노초 변환 처리
   - 낙관적 잠금 (버전 필드) 지원
   - PagingRequest/PagingResponse 임시 구현

7. **TaskStorage 구현 완료** (`internal/storage/sqlite/task.go`)
   - 모든 TaskStorage 인터페이스 메서드 구현
   - 태스크 상태별 필터링 및 검색
   - 세션별 태스크 조회
   - 활성 태스크 수 집계
   - 실행 시간 및 통계 데이터 관리
   - 낙관적 잠금 지원

#### 구현된 파일 구조

```
internal/storage/sqlite/
├── storage.go          # 메인 스토리지 및 연결 관리
├── workspace.go        # WorkspaceStorage 구현
├── project.go          # ProjectStorage 구현
├── session.go          # SessionStorage 구현
├── task.go             # TaskStorage 구현
└── transaction.go      # 트랜잭션 래퍼 구현
```

#### 주요 설계 특징

- **타입 안전성**: 모든 models 구조체와 완벽 호환
- **JSON 직렬화**: 복잡한 구조체를 JSON으로 저장/복원
- **NULL 안전성**: sql.NullString, sql.NullTime 활용
- **성능 최적화**: Prepared Statement 캐싱
- **동시성 제어**: 낙관적 잠금 패턴 (version 필드)
- **확장성**: 페이지네이션 및 필터링 지원
- **트랜잭션 안전성**: ACID 보장 구조

#### 다음 단계
- 통합 테스트 작성
- 실제 마이그레이션 파일 작성 (테이블 스키마)
- 스토리지 팩토리 패턴 통합
- 성능 벤치마크 테스트