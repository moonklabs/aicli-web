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
- [x] BoltDB 초기화 및 버킷 생성
- [x] 직렬화/역직렬화 유틸리티 구현
- [x] WorkspaceStorage 구현
- [ ] ProjectStorage 구현
- [ ] SessionStorage 구현
- [ ] TaskStorage 구현
- [x] 인덱스 관리 시스템 구현
- [x] 쿼리 헬퍼 함수 작성

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

### 2025-07-21 16:30:00Z
**T05_S03 BoltDB Storage Implementation 진행 중**

#### 완료된 작업

1. **BoltDB 초기화 및 버킷 생성 완료** (`internal/storage/boltdb/storage.go`)
   - BoltDB 연결 및 설정 구조 구현
   - 필요한 모든 버킷 자동 생성 (데이터 + 인덱스 버킷)
   - 스토리지 팩토리 패턴 구현
   - 연결 생명주기 관리 및 건강성 확인
   - 트랜잭션 래퍼 통합
   - 백업 및 통계 기능 지원

2. **직렬화/역직렬화 유틸리티 구현 완료** (`internal/storage/boltdb/serializer.go`)
   - 모든 모델별 전용 직렬화기 구현 (Workspace, Project, Session, Task)
   - JSON 마샬링/언마샬링 헬퍼 함수
   - 민감한 정보 마스킹 처리 (API 키 등)
   - 인덱스 엔트리 직렬화 지원
   - 페이지네이션 데이터 직렬화
   - 에러 처리 및 타입 안전성 보장
   - 필수 필드 검증 및 타임스탬프 정규화

3. **인덱스 관리 시스템 구현 완료** (`internal/storage/boltdb/index.go`)
   - 유니크 인덱스 및 다중 인덱스 지원
   - 미리 정의된 인덱스들 (소유자, 이름, 워크스페이스, 프로젝트, 세션, 상태)
   - 배치 인덱스 업데이트 지원
   - 인덱스 통계 및 성능 모니터링
   - 프리픽스 기반 검색 지원
   - 인덱스 정리 및 재구축 기능

4. **쿼리 헬퍼 함수 작성 완료** (`internal/storage/boltdb/query.go`)
   - 통합 쿼리 시스템 구현
   - 페이지네이션 지원 (커서 및 오프셋 기반)
   - 정렬 및 필터링 지원
   - 텍스트 검색 기능
   - 모든 스토리지 타입별 전용 쿼리 메서드
   - 성능 최적화된 데이터 조회
   - 조건별 카운트 및 통계 기능

5. **트랜잭션 래퍼 구현 완료** (`internal/storage/boltdb/transaction.go`)
   - BoltDB 트랜잭션 생명주기 관리
   - 자동 커밋/롤백 지원
   - 트랜잭션별 스토리지 래퍼 구조
   - 워크스페이스 트랜잭션 메서드 완전 구현
   - 안전한 트랜잭션 컨텍스트 관리

6. **WorkspaceStorage 구현 완료** (`internal/storage/boltdb/workspace.go`)
   - 모든 WorkspaceStorage 인터페이스 메서드 구현
   - CRUD 작업: Create, GetByID, GetByOwnerID, Update, Delete, List
   - 인덱스 기반 효율적 쿼리
   - Soft Delete 구현
   - 중복 검사 및 데이터 무결성 보장
   - 검색 및 통계 기능
   - 여러 소유자 기반 조회 기능

#### 주요 구현 특징

- **Key-Value 최적화**: BoltDB 특성을 활용한 효율적인 데이터 구조
- **인덱스 시스템**: 보조 인덱스를 통한 빠른 쿼리 성능
- **JSON 직렬화**: 복잡한 구조체를 JSON으로 저장/복원
- **트랜잭션 안전성**: ACID 보장 및 자동 롤백
- **메모리 효율성**: 스트리밍 기반 대용량 데이터 처리
- **확장성**: 배치 작업 및 인덱스 최적화
- **타입 안전성**: 강타입 직렬화기 및 검증

#### 생성된 파일 구조

```
internal/storage/boltdb/
├── storage.go          # BoltDB 메인 스토리지 및 연결 관리
├── serializer.go       # JSON 직렬화/역직렬화 유틸리티
├── index.go            # 인덱스 관리 시스템
├── query.go            # 쿼리 헬퍼 및 검색 기능
├── transaction.go      # 트랜잭션 래퍼 구현
└── workspace.go        # WorkspaceStorage 구현
```

#### 다음 단계
- ProjectStorage 구현 (`project.go`)
- SessionStorage 구현 (`session.go`)  
- TaskStorage 구현 (`task.go`)
- 통합 테스트 작성
- 성능 벤치마크 테스트

#### 설계 결정사항

1. **인덱스 전략**: 별도 버킷을 사용한 보조 인덱스로 쿼리 성능 최적화
2. **직렬화 방식**: JSON 기반으로 스키마 유연성과 가독성 확보
3. **트랜잭션 모델**: BoltDB의 MVCC를 활용한 안전한 동시성 제어
4. **버킷 구조**: 엔티티별 메인 버킷 + 인덱스 버킷으로 명확한 분리
5. **에러 처리**: 표준 storage 에러로 변환하여 인터페이스 일관성 유지