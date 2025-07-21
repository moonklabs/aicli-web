# BoltDB Bucket Structure

## Overview

BoltDB는 key-value 스토어로, 데이터를 버킷(bucket)이라는 네임스페이스로 구성합니다. AICode Manager에서는 각 엔티티별로 버킷을 생성하여 데이터를 관리합니다.

## Bucket Structure

### 1. workspaces
워크스페이스 정보를 저장하는 버킷

```
Key: workspace:{id}
Value: JSON encoded Workspace struct
```

예시:
```json
{
  "id": "ws_123e4567-e89b-12d3-a456-426614174000",
  "name": "My Project",
  "project_path": "/home/user/projects/myproject",
  "status": "active",
  "owner_id": "user_123",
  "claude_key": "encrypted_key_data",
  "active_tasks": 0,
  "created_at": "2025-07-21T10:00:00Z",
  "updated_at": "2025-07-21T10:00:00Z",
  "deleted_at": null,
  "version": 1
}
```

### 2. projects
프로젝트 정보를 저장하는 버킷

```
Key: project:{id}
Value: JSON encoded Project struct
```

### 3. sessions
세션 정보를 저장하는 버킷

```
Key: session:{id}
Value: JSON encoded Session struct
```

### 4. tasks
태스크 정보를 저장하는 버킷

```
Key: task:{id}
Value: JSON encoded Task struct
```

### 5. indexes
인덱스 정보를 저장하는 버킷 (빠른 조회를 위한 보조 인덱스)

#### 5.1 workspace_by_owner
```
Key: idx:workspace:owner:{owner_id}:{workspace_id}
Value: workspace_id (빈 값도 가능, 키 자체가 인덱스 역할)
```

#### 5.2 project_by_workspace
```
Key: idx:project:workspace:{workspace_id}:{project_id}
Value: project_id
```

#### 5.3 session_by_project
```
Key: idx:session:project:{project_id}:{session_id}
Value: session_id
```

#### 5.4 task_by_session
```
Key: idx:task:session:{session_id}:{task_id}
Value: task_id
```

#### 5.5 active_sessions
```
Key: idx:session:active:{project_id}
Value: session_id
```

### 6. metadata
시스템 메타데이터를 저장하는 버킷

```
Key: meta:schema_version
Value: "1"

Key: meta:last_migration
Value: "001_initial"
```

## Key Naming Convention

1. **Entity Keys**: `{entity_type}:{id}`
   - 예: `workspace:ws_123`, `project:proj_456`

2. **Index Keys**: `idx:{entity}:{index_type}:{value}:{id}`
   - 예: `idx:workspace:owner:user_123:ws_456`

3. **Metadata Keys**: `meta:{key_name}`
   - 예: `meta:schema_version`

## Transaction Support

BoltDB는 ACID 트랜잭션을 지원합니다. 다음과 같은 작업은 트랜잭션으로 처리되어야 합니다:

1. **Entity 생성**: Entity 저장 + 관련 인덱스 생성
2. **Entity 삭제**: Entity 삭제 + 관련 인덱스 삭제
3. **관계 업데이트**: 여러 Entity의 동시 업데이트

## Query Patterns

### 1. Get Workspace by ID
```go
bucket := tx.Bucket([]byte("workspaces"))
data := bucket.Get([]byte("workspace:" + id))
```

### 2. List Workspaces by Owner
```go
indexBucket := tx.Bucket([]byte("indexes"))
prefix := []byte("idx:workspace:owner:" + ownerID + ":")
c := indexBucket.Cursor()
for k, _ := c.Seek(prefix); bytes.HasPrefix(k, prefix); k, _ = c.Next() {
    // Extract workspace ID from key
}
```

### 3. Get Active Session for Project
```go
indexBucket := tx.Bucket([]byte("indexes"))
sessionID := indexBucket.Get([]byte("idx:session:active:" + projectID))
```

## Performance Considerations

1. **Bucket Size**: 각 버킷은 수백만 개의 키를 효율적으로 처리할 수 있습니다.

2. **Key Ordering**: 키는 바이트 순서로 정렬되므로, 범위 스캔이 효율적입니다.

3. **Index Management**: 보조 인덱스는 쿼리 성능을 향상시키지만, 쓰기 시 오버헤드가 있습니다.

4. **Transaction Size**: 대량의 쓰기 작업은 여러 트랜잭션으로 분할하는 것이 좋습니다.

## Backup Strategy

1. **Online Backup**: BoltDB는 온라인 백업을 지원합니다.
   ```go
   db.View(func(tx *bolt.Tx) error {
       return tx.CopyFile("backup.db", 0644)
   })
   ```

2. **Incremental Backup**: 트랜잭션 로그를 활용한 증분 백업 가능

3. **Export/Import**: JSON 형식으로 전체 데이터 내보내기/가져오기 구현 필요