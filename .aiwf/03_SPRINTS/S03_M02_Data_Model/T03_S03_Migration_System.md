---
task_id: T03_S03
sprint_sequence_id: S03_M02
status: open
complexity: Medium
last_updated: 2025-07-21T16:00:00Z
---

# Task: Database Migration System

## Description
데이터베이스 스키마 버전 관리와 자동 마이그레이션을 위한 시스템을 구현합니다. SQLite와 BoltDB 모두를 지원하며, 업그레이드와 다운그레이드가 가능한 마이그레이션 시스템을 구축합니다.

## Goal / Objectives
- 마이그레이션 파일 구조 및 명명 규칙 정의
- 마이그레이션 실행 엔진 구현
- 버전 추적 및 상태 관리
- CLI 명령어 통합
- 롤백 기능 지원

## Acceptance Criteria
- [ ] 마이그레이션 파일 자동 탐색 및 순서 보장
- [ ] 현재 스키마 버전 추적 가능
- [ ] Up/Down 마이그레이션 모두 지원
- [ ] CLI에서 `aicli db migrate` 명령 실행 가능
- [ ] 마이그레이션 실행 이력 저장

## Subtasks
- [ ] 마이그레이션 인터페이스 정의
- [ ] 마이그레이션 파일 로더 구현
- [ ] SQLite 마이그레이션 실행기 구현
- [ ] BoltDB 마이그레이션 실행기 구현
- [ ] 버전 추적 테이블/버킷 구현
- [ ] CLI 명령어 통합
- [ ] 마이그레이션 생성 도구 작성

## 기술 가이드

### 주요 인터페이스 및 통합 지점
- `cmd/aicli/commands/` - CLI 명령어 추가 위치
- `internal/storage/` - 마이그레이션 시스템 위치
- golang-migrate 라이브러리 참고 (직접 구현 예정)

### 마이그레이션 인터페이스
```go
// internal/storage/migration/migration.go
type Migration interface {
    Version() string
    Description() string
    Up(ctx context.Context, db interface{}) error
    Down(ctx context.Context, db interface{}) error
}

type Migrator interface {
    Current() (string, error)
    Migrate(target string) error
    Rollback(steps int) error
    List() ([]MigrationInfo, error)
}
```

### 파일 명명 규칙
```
migrations/
├── sqlite/
│   ├── 001_initial_schema.up.sql
│   ├── 001_initial_schema.down.sql
│   ├── 002_add_indexes.up.sql
│   └── 002_add_indexes.down.sql
└── boltdb/
    ├── 001_initial_buckets.go
    └── 002_add_metadata.go
```

### CLI 명령어 구조
```bash
aicli db migrate                # 최신 버전으로 마이그레이션
aicli db migrate --version 3    # 특정 버전으로 마이그레이션
aicli db rollback --steps 1     # 1단계 롤백
aicli db status                 # 현재 마이그레이션 상태
aicli db create <name>          # 새 마이그레이션 파일 생성
```

## 구현 노트

### 버전 추적
- SQLite: `schema_migrations` 테이블 사용
- BoltDB: `_migrations` 버킷 사용
- 각 마이그레이션 실행 시 타임스탬프 기록

### 에러 처리
- 마이그레이션 실패 시 자동 롤백
- 부분 적용 방지를 위한 트랜잭션 사용
- 상세한 에러 로그 및 복구 가이드 제공

### embed 활용
```go
//go:embed migrations/sqlite/*.sql
var sqliteMigrations embed.FS

//go:embed migrations/boltdb/*.go
var boltdbMigrations embed.FS
```

## Output Log
*(작업 진행 시 업데이트)*