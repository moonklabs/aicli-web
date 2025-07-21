---
task_id: T01_S03
sprint_sequence_id: S03_M02
status: completed
complexity: Medium
last_updated: 2025-07-21T16:30:00Z
---

# Task: Database Schema Design

## Description
AICode Manager의 핵심 엔티티들에 대한 데이터베이스 스키마를 설계합니다. SQLite와 BoltDB 모두를 지원할 수 있도록 추상화된 스키마를 정의하고, 엔티티 간의 관계를 명확히 설정합니다.

## Goal / Objectives
- 모든 핵심 엔티티(Workspace, Project, Session, Task)의 스키마 정의
- 엔티티 간 관계 및 제약조건 설계
- 인덱스 전략 수립
- SQLite DDL 스크립트 작성
- BoltDB 버킷 구조 설계

## Acceptance Criteria
- [x] ERD(Entity Relationship Diagram) 문서 작성 완료
- [x] SQLite 스키마 SQL 파일 생성
- [x] BoltDB 버킷 및 키 구조 문서화
- [x] 인덱스 전략 문서 작성
- [x] 데이터 타입 및 제약조건 명세 완료

## Subtasks
- [x] 기존 모델 분석 및 필요 필드 확인
- [x] ERD 다이어그램 작성 (Mermaid 사용)
- [x] SQLite 테이블 스키마 정의
- [x] BoltDB 버킷 구조 설계
- [x] 인덱스 및 성능 최적화 전략 수립
- [x] 스키마 문서화 (docs/database-schema.md)

## 기술 가이드

### 주요 인터페이스 및 통합 지점
- `internal/models/*.go` - 기존 Go 구조체 모델들
- `internal/storage/interface.go` - 스토리지 인터페이스 정의
- 새로 생성할 위치: `internal/storage/schema/`

### 고려해야 할 기존 모델
```
- models.Workspace: ID, Name, ProjectPath, Status, OwnerID, ClaudeKey, ActiveTasks, Timestamps
- models.Project: ID, WorkspaceID, Name, Path, GitBranch, Config, Status, Timestamps  
- models.Session: ID, ProjectID, Type, Status, Config, StartedAt, EndedAt, Metadata
- models.Task: ID, SessionID, Command, Status, Output, Error, StartedAt, CompletedAt
```

### 스키마 설계 원칙
- UUID를 Primary Key로 사용 (STRING 타입)
- 타임스탬프는 UTC로 저장
- Soft Delete 지원 (deleted_at 필드)
- 버전 관리를 위한 version 필드 추가 고려
- 전문 검색을 위한 필드는 별도 인덱싱

## 구현 노트

### SQLite 스키마 파일 경로
- `internal/storage/schema/sqlite/001_initial.sql`
- `internal/storage/schema/sqlite/indexes.sql`

### BoltDB 구조 문서 경로
- `internal/storage/schema/boltdb/buckets.md`

### 문서화 경로
- `docs/database/schema.md` - 전체 스키마 문서
- `docs/database/erd.md` - ERD 다이어그램

## Output Log

### 2025-07-21T16:30:00Z - 작업 완료

#### 수행한 작업:

1. **기존 모델 분석**
   - `/internal/models/` 디렉토리의 모든 모델 파일 검토
   - BaseModel 구조 누락 확인 및 생성
   - 4개의 핵심 엔티티 구조 파악 완료

2. **BaseModel 정의 추가**
   - `/internal/models/base.go` 파일 생성
   - 공통 필드 정의: ID, CreatedAt, UpdatedAt, Version
   - ProjectResponse 모델 및 ToResponse 메서드 구현

3. **ERD 다이어그램 작성**
   - `/docs/database/erd.md` 파일 생성
   - Mermaid 형식으로 엔티티 관계도 작성
   - 4개 엔티티 간의 관계 명확히 정의

4. **SQLite 스키마 생성**
   - `/internal/storage/schema/sqlite/001_initial.sql` - DDL 스크립트
   - `/internal/storage/schema/sqlite/indexes.sql` - 인덱스 정의
   - 트리거를 통한 updated_at 자동 업데이트 구현

5. **BoltDB 버킷 구조 설계**
   - `/internal/storage/schema/boltdb/buckets.md` 문서 작성
   - 6개의 주요 버킷 정의 (workspaces, projects, sessions, tasks, indexes, metadata)
   - 키 네이밍 컨벤션 및 쿼리 패턴 문서화

6. **통합 스키마 문서**
   - `/docs/database/schema.md` 종합 문서 작성
   - 전체 데이터 모델, JSON 스키마, 인덱싱 전략 포함
   - 성능 최적화 및 보안 고려사항 명시

#### 주요 설계 결정:

- **UUID v4**를 모든 테이블의 Primary Key로 사용
- **Soft Delete** 지원 (Workspace, Project)
- **낙관적 잠금** 구현 (version 필드)
- **JSON 필드** 활용 (config, metadata, git_info)
- **복합 인덱스** 설계로 쿼리 성능 최적화

#### 생성된 파일:
- `/internal/models/base.go`
- `/internal/storage/schema/sqlite/001_initial.sql`
- `/internal/storage/schema/sqlite/indexes.sql`
- `/internal/storage/schema/boltdb/buckets.md`
- `/docs/database/erd.md`
- `/docs/database/schema.md`