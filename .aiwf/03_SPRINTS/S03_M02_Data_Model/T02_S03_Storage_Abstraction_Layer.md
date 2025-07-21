---
task_id: T02_S03
sprint_sequence_id: S03_M02
status: open
complexity: Medium
last_updated: 2025-07-21T16:00:00Z
---

# Task: Storage Abstraction Layer Implementation

## Description
SQLite와 BoltDB를 모두 지원할 수 있는 통합 스토리지 추상화 계층을 구현합니다. 기존 메모리 스토리지 인터페이스를 유지하면서 실제 데이터베이스 백엔드로 전환할 수 있는 구조를 만듭니다.

## Goal / Objectives
- 데이터베이스 독립적인 스토리지 팩토리 구현
- 설정 기반 스토리지 백엔드 선택 로직
- 공통 에러 처리 및 로깅 시스템
- 연결 풀링 및 생명주기 관리
- 트랜잭션 추상화 인터페이스

## Acceptance Criteria
- [ ] StorageFactory 구현으로 백엔드 선택 가능
- [ ] 기존 storage.Storage 인터페이스와 100% 호환
- [ ] 설정으로 SQLite/BoltDB/Memory 선택 가능
- [ ] 트랜잭션 인터페이스 정의 및 구현
- [ ] 에러 처리 및 로깅 표준화

## Subtasks
- [ ] StorageFactory 인터페이스 및 구현체 작성
- [ ] 설정 구조체에 스토리지 타입 추가
- [ ] 트랜잭션 인터페이스 정의
- [ ] 공통 유틸리티 함수 구현
- [ ] 연결 관리 로직 구현
- [ ] 단위 테스트 작성

## 기술 가이드

### 주요 인터페이스 및 통합 지점
- `internal/storage/interface.go` - 기존 스토리지 인터페이스
- `internal/storage/memory/` - 참조할 메모리 구현
- `internal/config/config.go` - 설정 구조체

### 새로 생성할 구조
```go
// internal/storage/factory.go
type StorageType string
const (
    StorageTypeMemory  StorageType = "memory"
    StorageTypeSQLite  StorageType = "sqlite"
    StorageTypeBoltDB  StorageType = "boltdb"
)

type StorageConfig struct {
    Type       StorageType
    DataSource string  // 파일 경로 또는 연결 문자열
    MaxConns   int
    // 기타 설정...
}

type StorageFactory interface {
    Create(config StorageConfig) (Storage, error)
}
```

### 트랜잭션 추상화
```go
// internal/storage/transaction.go
type Transaction interface {
    Commit() error
    Rollback() error
    
    // 각 스토리지 인터페이스의 트랜잭션 버전
    Workspace() WorkspaceStorage
    Project() ProjectStorage
    Session() SessionStorage
    Task() TaskStorage
}
```

### 에러 처리 패턴
- 기존 에러 변수 재사용 (ErrNotFound, ErrAlreadyExists 등)
- 데이터베이스별 에러를 공통 에러로 변환
- 컨텍스트 정보 포함한 에러 래핑

## 구현 노트

### 파일 구조
```
internal/storage/
├── factory.go          # StorageFactory 구현
├── transaction.go      # 트랜잭션 인터페이스
├── errors.go          # 에러 변환 유틸리티
├── sqlite/            # SQLite 구현 (T04에서)
├── boltdb/            # BoltDB 구현 (T05에서)
└── memory/            # 기존 메모리 구현
```

### 설정 통합
- `internal/config/config.go`에 Storage 섹션 추가
- 환경 변수: `AICLI_STORAGE_TYPE`, `AICLI_STORAGE_PATH`
- 기본값: memory (개발), sqlite (프로덕션)

## Output Log
*(작업 진행 시 업데이트)*