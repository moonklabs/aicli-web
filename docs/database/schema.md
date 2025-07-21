# AICode Manager Database Schema Documentation

## 개요

AICode Manager는 두 가지 데이터베이스 엔진을 지원합니다:
- **SQLite**: 관계형 데이터베이스, 복잡한 쿼리와 트랜잭션 지원
- **BoltDB**: Key-Value 스토어, 고성능과 간단한 운영

## 데이터 모델

### 1. Workspace (워크스페이스)

워크스페이스는 AICode Manager의 최상위 조직 단위입니다.

| Field | Type | Description | Constraints |
|-------|------|-------------|-------------|
| id | CHAR(36) | UUID v4 형식의 고유 식별자 | PRIMARY KEY |
| name | VARCHAR(100) | 워크스페이스 이름 | NOT NULL |
| project_path | VARCHAR(500) | 프로젝트 루트 경로 | NOT NULL |
| status | VARCHAR(20) | 상태 (active/inactive/archived) | NOT NULL, DEFAULT 'active' |
| owner_id | VARCHAR(50) | 소유자 ID | NOT NULL |
| claude_key | TEXT | 암호화된 Claude API 키 | NULLABLE |
| active_tasks | INTEGER | 현재 활성 태스크 수 | NOT NULL, DEFAULT 0 |
| created_at | DATETIME | 생성 시간 | NOT NULL |
| updated_at | DATETIME | 수정 시간 | NOT NULL |
| deleted_at | DATETIME | 삭제 시간 (soft delete) | NULLABLE |
| version | INTEGER | 낙관적 잠금용 버전 | NOT NULL, DEFAULT 1 |

### 2. Project (프로젝트)

프로젝트는 워크스페이스 내의 개별 코드베이스를 나타냅니다.

| Field | Type | Description | Constraints |
|-------|------|-------------|-------------|
| id | CHAR(36) | UUID v4 형식의 고유 식별자 | PRIMARY KEY |
| workspace_id | CHAR(36) | 소속 워크스페이스 ID | FOREIGN KEY, NOT NULL |
| name | VARCHAR(100) | 프로젝트 이름 | NOT NULL |
| path | VARCHAR(500) | 프로젝트 경로 | NOT NULL |
| description | TEXT | 프로젝트 설명 | NULLABLE |
| git_url | VARCHAR(500) | Git 리포지토리 URL | NULLABLE |
| git_branch | VARCHAR(100) | 현재 Git 브랜치 | NULLABLE |
| language | VARCHAR(50) | 주 개발 언어 | NULLABLE |
| status | VARCHAR(20) | 상태 (active/inactive/archived) | NOT NULL, DEFAULT 'active' |
| config | TEXT | 프로젝트 설정 (JSON) | NULLABLE |
| git_info | TEXT | Git 정보 캐시 (JSON) | NULLABLE |
| created_at | DATETIME | 생성 시간 | NOT NULL |
| updated_at | DATETIME | 수정 시간 | NOT NULL |
| deleted_at | DATETIME | 삭제 시간 (soft delete) | NULLABLE |
| version | INTEGER | 낙관적 잠금용 버전 | NOT NULL, DEFAULT 1 |

### 3. Session (세션)

세션은 Claude CLI 프로세스의 생명주기를 관리합니다.

| Field | Type | Description | Constraints |
|-------|------|-------------|-------------|
| id | CHAR(36) | UUID v4 형식의 고유 식별자 | PRIMARY KEY |
| project_id | CHAR(36) | 소속 프로젝트 ID | FOREIGN KEY, NOT NULL |
| process_id | INTEGER | OS 프로세스 ID | NULLABLE |
| status | VARCHAR(20) | 상태 (pending/active/idle/ending/ended/error) | NOT NULL, DEFAULT 'pending' |
| started_at | DATETIME | 시작 시간 | NULLABLE |
| ended_at | DATETIME | 종료 시간 | NULLABLE |
| last_active | DATETIME | 마지막 활동 시간 | NOT NULL |
| metadata | TEXT | 세션 메타데이터 (JSON) | NULLABLE |
| command_count | BIGINT | 실행된 명령어 수 | NOT NULL, DEFAULT 0 |
| bytes_in | BIGINT | 입력 바이트 수 | NOT NULL, DEFAULT 0 |
| bytes_out | BIGINT | 출력 바이트 수 | NOT NULL, DEFAULT 0 |
| error_count | BIGINT | 발생한 에러 수 | NOT NULL, DEFAULT 0 |
| max_idle_time | BIGINT | 최대 유휴 시간 (나노초) | NOT NULL, DEFAULT 1800000000000 |
| max_lifetime | BIGINT | 최대 생명 시간 (나노초) | NOT NULL, DEFAULT 14400000000000 |
| created_at | DATETIME | 생성 시간 | NOT NULL |
| updated_at | DATETIME | 수정 시간 | NOT NULL |
| version | INTEGER | 낙관적 잠금용 버전 | NOT NULL, DEFAULT 1 |

### 4. Task (태스크)

태스크는 세션 내에서 실행되는 개별 명령을 나타냅니다.

| Field | Type | Description | Constraints |
|-------|------|-------------|-------------|
| id | CHAR(36) | UUID v4 형식의 고유 식별자 | PRIMARY KEY |
| session_id | CHAR(36) | 소속 세션 ID | FOREIGN KEY, NOT NULL |
| command | TEXT | 실행할 명령어 | NOT NULL |
| status | VARCHAR(20) | 상태 (pending/running/completed/failed/cancelled) | NOT NULL, DEFAULT 'pending' |
| output | TEXT | 명령 실행 출력 | NULLABLE |
| error | TEXT | 에러 메시지 | NULLABLE |
| started_at | DATETIME | 시작 시간 | NULLABLE |
| completed_at | DATETIME | 완료 시간 | NULLABLE |
| bytes_in | BIGINT | 입력 바이트 수 | NOT NULL, DEFAULT 0 |
| bytes_out | BIGINT | 출력 바이트 수 | NOT NULL, DEFAULT 0 |
| duration | BIGINT | 실행 시간 (밀리초) | NOT NULL, DEFAULT 0 |
| created_at | DATETIME | 생성 시간 | NOT NULL |
| updated_at | DATETIME | 수정 시간 | NOT NULL |
| version | INTEGER | 낙관적 잠금용 버전 | NOT NULL, DEFAULT 1 |

## JSON 스키마

### ProjectConfig
```json
{
  "claude_api_key": "string (encrypted)",
  "environment": {
    "KEY": "VALUE"
  },
  "claude_options": {
    "model": "string",
    "max_tokens": "number",
    "temperature": "number",
    "system_prompt": "string",
    "exclude_paths": ["string"],
    "include_paths": ["string"]
  },
  "build_commands": ["string"],
  "test_commands": ["string"]
}
```

### GitInfo
```json
{
  "remote_url": "string",
  "current_branch": "string",
  "is_clean": "boolean",
  "last_commit": {
    "hash": "string",
    "author": "string",
    "message": "string",
    "timestamp": "datetime"
  },
  "status": {
    "modified": ["string"],
    "added": ["string"],
    "deleted": ["string"],
    "untracked": ["string"],
    "has_changes": "boolean"
  }
}
```

### SessionMetadata
```json
{
  "user_agent": "string",
  "client_ip": "string",
  "claude_version": "string",
  "docker_container_id": "string",
  "environment": {
    "KEY": "VALUE"
  }
}
```

## 인덱싱 전략

### Primary Indexes
- 모든 테이블의 `id` 필드는 Primary Key로 자동 인덱싱됩니다.

### Secondary Indexes

#### Workspace Indexes
- `idx_workspace_owner_id`: 사용자별 워크스페이스 조회
- `idx_workspace_status`: 상태별 필터링 (soft delete 제외)
- `idx_workspace_owner_status`: 복합 인덱스 (자주 사용되는 패턴)

#### Project Indexes
- `idx_project_workspace_id`: 워크스페이스별 프로젝트 조회
- `idx_project_status`: 상태별 필터링 (soft delete 제외)
- `idx_project_workspace_status`: 복합 인덱스

#### Session Indexes
- `idx_session_project_id`: 프로젝트별 세션 조회
- `idx_session_status`: 상태별 필터링
- `idx_session_last_active`: 유휴 세션 정리용
- `idx_session_active`: 활성 세션 빠른 조회

#### Task Indexes
- `idx_task_session_id`: 세션별 태스크 조회
- `idx_task_status`: 상태별 필터링
- `idx_task_created_at`: 시간순 정렬

## 데이터 무결성

### 1. Foreign Key Constraints
- CASCADE DELETE: 부모 엔티티 삭제 시 자식 엔티티도 함께 삭제
- 모든 외래 키는 NOT NULL로 설정

### 2. Check Constraints
- `status` 필드는 정의된 값만 허용
- 시간 필드는 논리적 순서 유지 (started_at < completed_at)

### 3. Soft Delete
- Workspace와 Project는 soft delete 지원
- 삭제된 레코드는 쿼리에서 기본적으로 제외
- 완전 삭제는 별도의 정리 작업으로 수행

### 4. Optimistic Locking
- `version` 필드를 통한 동시성 제어
- UPDATE 시 version 확인 및 증가

## 마이그레이션 전략

### SQLite
1. 마이그레이션 스크립트는 `internal/storage/schema/sqlite/` 디렉토리에 저장
2. 파일명 형식: `{version}_{description}.sql`
3. 순차적으로 실행되며, 롤백 스크립트도 함께 관리

### BoltDB
1. 스키마 변경이 상대적으로 유연함
2. 버킷 구조 변경 시 데이터 마이그레이션 함수 구현
3. 버전 정보는 metadata 버킷에 저장

## 성능 최적화

### 1. 쿼리 최적화
- 인덱스를 활용한 효율적인 조회
- N+1 쿼리 문제 방지 (JOIN 활용)
- 페이지네이션 구현

### 2. 캐싱 전략
- Git 정보는 주기적으로 갱신
- 활성 세션 정보는 메모리 캐시 활용
- 읽기 전용 데이터는 애플리케이션 레벨 캐싱

### 3. 연결 풀링
- SQLite: WAL 모드 활성화
- 적절한 연결 수 유지
- 장시간 트랜잭션 방지

## 보안 고려사항

### 1. API 키 암호화
- Claude API 키는 항상 암호화하여 저장
- 애플리케이션에서만 복호화 가능

### 2. 접근 제어
- owner_id를 통한 리소스 접근 제어
- 세션 격리를 통한 프로젝트 간 분리

### 3. 감사 로그
- 중요 작업은 별도 로그 테이블에 기록
- 변경 이력 추적 가능