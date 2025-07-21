---
task_id: T05_S03
sprint_sequence_id: S03_M02
status: open
complexity: Medium
last_updated: 2025-07-21T16:00:00Z
---

# Task: BoltDB Storage Implementation

## Description
BoltDB를 사용하는 Key-Value 기반 스토리지 구현체를 개발합니다. 문서 지향적 접근으로 JSON 직렬화를 사용하며, 효율적인 인덱싱과 쿼리를 지원합니다.

## Goal / Objectives
- BoltDB 버킷 구조 구현
- JSON 직렬화/역직렬화 로직
- 보조 인덱스 구현
- 트랜잭션 지원
- 쿼리 및 필터링 로직

## Acceptance Criteria
- [ ] 모든 스토리지 인터페이스 메서드 구현 완료
- [ ] 효율적인 인덱싱 시스템 구현
- [ ] 페이지네이션 지원
- [ ] 트랜잭션 처리 정상 작동
- [ ] 모든 단위 테스트 통과

## Subtasks
- [ ] BoltDB 초기화 및 버킷 생성
- [ ] 직렬화/역직렬화 유틸리티 구현
- [ ] WorkspaceStorage 구현
- [ ] ProjectStorage 구현
- [ ] SessionStorage 구현
- [ ] TaskStorage 구현
- [ ] 인덱스 관리 시스템 구현
- [ ] 쿼리 헬퍼 함수 작성

## 기술 가이드

### 주요 인터페이스 및 통합 지점
- `internal/storage/interface.go` - 구현해야 할 인터페이스
- 사용할 라이브러리: `go.etcd.io/bbolt`
- JSON 처리: 표준 `encoding/json`

### 버킷 구조 설계
```go
// 메인 버킷
const (
    bucketWorkspaces = "workspaces"
    bucketProjects   = "projects"
    bucketSessions   = "sessions"
    bucketTasks      = "tasks"
    
    // 인덱스 버킷
    bucketIndexOwner     = "idx_owner"      // owner_id -> workspace_ids
    bucketIndexWorkspace = "idx_workspace"  // workspace_id -> project_ids
    bucketIndexProject   = "idx_project"    // project_id -> session_ids
    bucketIndexSession   = "idx_session"    // session_id -> task_ids
)
```

### BoltDB 트랜잭션 패턴
```go
// internal/storage/boltdb/storage.go
type Storage struct {
    db *bbolt.DB
}

func (s *Storage) View(fn func(*bbolt.Tx) error) error {
    return s.db.View(fn)
}

func (s *Storage) Update(fn func(*bbolt.Tx) error) error {
    return s.db.Update(fn)
}
```

### 인덱싱 전략
```go
// 보조 인덱스 관리
func updateIndex(tx *bbolt.Tx, indexBucket, key, value string) error {
    b := tx.Bucket([]byte(indexBucket))
    
    // 기존 값 가져오기
    existing := b.Get([]byte(key))
    var ids []string
    if existing != nil {
        json.Unmarshal(existing, &ids)
    }
    
    // 새 값 추가
    ids = append(ids, value)
    
    // 저장
    data, _ := json.Marshal(ids)
    return b.Put([]byte(key), data)
}
```

### 페이지네이션 구현
```go
// Cursor 기반 페이지네이션
func (s *workspaceStorage) List(tx *bbolt.Tx, offset, limit int) ([]*models.Workspace, error) {
    b := tx.Bucket([]byte(bucketWorkspaces))
    c := b.Cursor()
    
    // offset까지 스킵
    for i := 0; i < offset && c.Next(); i++ {}
    
    var results []*models.Workspace
    for k, v := c.Next(); k != nil && len(results) < limit; k, v = c.Next() {
        var ws models.Workspace
        if err := json.Unmarshal(v, &ws); err == nil {
            results = append(results, &ws)
        }
    }
    
    return results, nil
}
```

## 구현 노트

### 파일 구조
```
internal/storage/boltdb/
├── storage.go          # 메인 스토리지 구현
├── workspace.go        # WorkspaceStorage 구현
├── project.go          # ProjectStorage 구현
├── session.go          # SessionStorage 구현
├── task.go             # TaskStorage 구현
├── transaction.go      # 트랜잭션 래퍼
├── index.go            # 인덱스 관리
├── serializer.go       # JSON 직렬화
└── query.go            # 쿼리 헬퍼
```

### 성능 최적화
- Batch 작업 시 단일 트랜잭션 사용
- 자주 사용되는 쿼리를 위한 인덱스 버킷
- Read 작업은 View 트랜잭션 사용
- 큰 데이터는 별도 버킷에 저장

### 에러 처리
- Key not found → `ErrNotFound`
- Duplicate key → `ErrAlreadyExists`
- 트랜잭션 실패 시 자동 롤백

## Output Log
*(작업 진행 시 업데이트)*