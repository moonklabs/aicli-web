---
task_id: T03_S03
sprint_sequence_id: S03_M02
status: done
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
- [x] 마이그레이션 파일 자동 탐색 및 순서 보장
- [x] 현재 스키마 버전 추적 가능
- [x] Up/Down 마이그레이션 모두 지원
- [x] CLI에서 `aicli db migrate` 명령 실행 가능
- [x] 마이그레이션 실행 이력 저장

## Subtasks
- [x] 마이그레이션 인터페이스 정의
- [x] 마이그레이션 파일 로더 구현
- [x] SQLite 마이그레이션 실행기 구현
- [x] BoltDB 마이그레이션 실행기 구현
- [x] 버전 추적 테이블/버킷 구현
- [x] CLI 명령어 통합
- [x] 마이그레이션 생성 도구 작성

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

### 2025-07-21
**T03_S03 Database Migration System 구현 완료**

#### 완료된 작업

1. **마이그레이션 인터페이스 정의** (`internal/storage/migration/migration.go`)
   - Migration 핵심 인터페이스 정의 (Version, Description, Up, Down, CanRollback)
   - Migrator 실행기 인터페이스 정의 (Current, Migrate, Rollback, List, Status)
   - MigrationSource 소스 인터페이스 정의 (Load, Get, List)
   - MigrationTracker 추적기 인터페이스 정의 (Record 관리)
   - BaseMigration 기본 구현체 및 헬퍼 함수들
   - 마이그레이션 상태, 방향, 옵션, 이벤트 타입 정의
   - 버전 검증 및 비교 함수 (ValidateVersion, CompareVersions, SortVersions)

2. **마이그레이션 파일 로더 구현** (`internal/storage/migration/loader.go`)
   - SQLMigration 구조체 (UP/DOWN SQL 실행)
   - FileSystemSource 파일 시스템 기반 소스 구현
   - embed.FS 지원 (NewEmbedSource)
   - 마이그레이션 파일 스캔 및 파싱 (정규식 패턴 매칭)
   - 버전별 그룹화 및 정렬
   - MigrationLoader 멀티 소스 로더
   - 마이그레이션 순서 검증 (validateMigrationSequence)

3. **SQLite 마이그레이션 실행기 구현** (`internal/storage/migration/sqlite.go`)
   - SQLiteTracker 마이그레이션 추적기
   - schema_migrations 테이블 자동 생성 및 관리
   - 마이그레이션 실행 기록 저장/조회 (상태, 시간, 오류)
   - SQLiteMigrator 실행기 (트랜잭션 기반 실행)
   - 업그레이드/다운그레이드 지원 (planMigrations)
   - N단계 롤백 및 특정 버전 롤백
   - 자동 롤백 (실패 시)
   - DryRun 모드 지원

4. **BoltDB 마이그레이션 실행기 구현** (`internal/storage/migration/boltdb.go`)
   - BoltDBMigration 인터페이스 (Key-Value DB 전용)
   - BoltDBFuncMigration 함수 기반 구현체
   - BoltDBTracker JSON 기반 마이그레이션 기록 관리
   - _migrations 버킷 자동 생성
   - BoltDBMigrationSource 코드 기반 마이그레이션 소스
   - BoltDBMigrator 실행기 (bbolt.Tx 기반)
   - 트랜잭션 안전성 보장

5. **CLI 명령어 통합** (`internal/cli/commands/db.go`)
   - `aicli db migrate` - 마이그레이션 실행 (--version, --dry-run, --verbose 지원)
   - `aicli db rollback` - 롤백 실행 (--steps, --version 지원)
   - `aicli db status` - 마이그레이션 상태 표시 (테이블 형태)
   - `aicli db version` - 현재 스키마 버전 확인
   - `aicli db create` - 새 마이그레이션 파일 템플릿 생성
   - 설정 기반 스토리지 타입 자동 선택
   - 에러 처리 및 사용자 친화적 메시지

6. **마이그레이션 생성 도구 작성**
   - SQLite용 UP/DOWN SQL 파일 자동 생성
   - BoltDB용 Go 코드 템플릿 자동 생성
   - 자동 버전 번호 생성 (001, 002, 003...)
   - 파일명 정규화 및 설명 추가
   - 디렉토리 자동 생성

7. **CLI 통합 및 설정**
   - root.go에 DB 명령어 그룹 추가
   - 기존 config 시스템과 통합
   - 자동 데이터베이스 경로 설정 ($HOME/.aicli/)
   - 환경 변수 지원 (AICLI_STORAGE_TYPE, AICLI_STORAGE_DATA_SOURCE)

#### 주요 기능

- **멀티 백엔드 지원**: SQLite와 BoltDB 모두 지원
- **트랜잭션 안전성**: 실패 시 자동 롤백, 부분 적용 방지
- **버전 추적**: 완전한 마이그레이션 이력 관리
- **DryRun 모드**: 실제 실행 전 계획 확인 가능
- **유연한 롤백**: N단계 롤백 또는 특정 버전으로 롤백
- **자동 파일 생성**: 템플릿 기반 마이그레이션 파일 자동 생성
- **CLI 통합**: 직관적인 명령어 인터페이스

#### CLI 명령어 예시

```bash
# 마이그레이션 실행
aicli db migrate                    # 최신 버전으로
aicli db migrate --version 003      # 특정 버전으로
aicli db migrate --dry-run          # 계획만 확인

# 롤백
aicli db rollback --steps 1         # 1단계 롤백
aicli db rollback --version 002     # 특정 버전으로 롤백

# 상태 확인
aicli db status                     # 전체 상태
aicli db version                    # 현재 버전

# 마이그레이션 생성
aicli db create add_user_table      # 새 마이그레이션 생성
```

#### 생성된 파일 구조

```
internal/storage/migration/
├── migration.go          # 핵심 인터페이스 및 타입
├── loader.go            # 파일 시스템 로더
├── sqlite.go           # SQLite 마이그레이션 실행기
└── boltdb.go           # BoltDB 마이그레이션 실행기

internal/cli/commands/
└── db.go               # DB 관련 CLI 명령어

internal/storage/schema/
├── sqlite/             # SQLite 마이그레이션 파일 위치
│   ├── 001_*.up.sql
│   └── 001_*.down.sql
└── boltdb/             # BoltDB 마이그레이션 파일 위치
    └── 001_*.go
```

#### 의존성

기존 go.mod에 이미 포함된 라이브러리들을 활용:
- github.com/spf13/cobra (CLI)
- github.com/mattn/go-sqlite3 (SQLite 드라이버)
- go.etcd.io/bbolt (BoltDB)

#### 다음 단계 권장사항

1. **실제 마이그레이션 파일 작성**: T01_S03에서 설계한 스키마를 실제 마이그레이션 파일로 구현
2. **스토리지 구현체 통합**: T04_S03, T05_S03에서 SQLite/BoltDB 스토리지와 연동
3. **자동 마이그레이션**: 애플리케이션 시작 시 자동 마이그레이션 실행 옵션 추가
4. **백업/복원**: 마이그레이션 전 자동 백업 기능
5. **웹 UI**: 마이그레이션 상태를 웹에서 확인할 수 있는 대시보드