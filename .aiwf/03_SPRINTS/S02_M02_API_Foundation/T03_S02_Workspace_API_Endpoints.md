---
task_id: T03_S02
task_name: Workspace Management API Endpoints
status: done
complexity: medium
priority: high
created_at: 2025-07-21
updated_at: 2025-07-21 16:11
sprint_id: S02_M02
---

# T03_S02: Workspace Management API Endpoints

## 태스크 개요

워크스페이스 관리를 위한 핵심 비즈니스 API 엔드포인트를 구현합니다. 워크스페이스 CRUD 작업과 관련된 비즈니스 로직을 완성합니다.

## 목표

- 워크스페이스 생성/조회/수정/삭제 API 구현
- 워크스페이스 메타데이터 관리
- 프로젝트 경로 검증 및 관리
- 워크스페이스 상태 추적

## 수용 기준

- [x] POST /workspaces가 새 워크스페이스 생성
- [x] GET /workspaces가 페이지네이션과 함께 목록 반환
- [x] GET /workspaces/:id가 상세 정보 반환
- [x] PUT /workspaces/:id가 워크스페이스 정보 업데이트
- [x] DELETE /workspaces/:id가 워크스페이스 삭제 (soft delete)
- [x] 프로젝트 경로 유효성 검증 동작
- [x] 적절한 에러 응답과 상태 코드 반환
- [x] 통합 테스트 작성

## 기술 가이드

### 주요 인터페이스 및 통합 지점

1. **기존 컨트롤러**: `internal/api/controllers/workspace.go` 스텁 구현 완성
2. **모델 정의**: `internal/models/` 디렉토리에 Workspace 모델 정의
3. **스토리지 레이어**: `internal/storage/` 디렉토리에 저장소 인터페이스 구현
4. **검증 로직**: gin의 binding 태그와 custom validator 활용

### 구현 구조

```
internal/
├── models/
│   ├── workspace.go     # 워크스페이스 모델 정의
│   └── pagination.go    # 페이지네이션 모델
├── storage/
│   ├── interface.go     # 스토리지 인터페이스
│   ├── memory/          # 메모리 기반 구현 (개발용)
│   └── sqlite/          # SQLite 구현 (추후)
├── api/
│   └── controllers/
│       └── workspace.go # 워크스페이스 컨트롤러 완성
└── utils/
    └── validator.go     # 커스텀 검증 함수
```

### 기존 패턴 참조

- 컨트롤러 패턴: 기존 `workspace.go`의 인터페이스 유지
- 에러 처리: `internal/middleware/error.go` 패턴 활용
- 응답 포맷: 기존 핸들러들의 JSON 응답 구조 따르기

## 구현 노트

### 단계별 접근법

1. Workspace 모델 정의 (ID, 이름, 경로, 상태, 타임스탬프)
2. 스토리지 인터페이스 정의
3. 메모리 기반 스토리지 구현 (개발/테스트용)
4. 컨트롤러 비즈니스 로직 구현
5. 입력 검증 및 에러 처리
6. 페이지네이션 구현
7. 통합 테스트 작성

### 워크스페이스 모델

```go
type Workspace struct {
    ID          string    `json:"id"`
    Name        string    `json:"name" binding:"required"`
    ProjectPath string    `json:"project_path"`
    Status      string    `json:"status"`
    OwnerID     string    `json:"owner_id"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

### 비즈니스 로직

- 워크스페이스 이름 중복 검사
- 프로젝트 경로 존재 여부 확인
- 워크스페이스 상태 관리 (active, inactive, archived)
- Soft delete 구현 (실제 삭제 대신 상태 변경)

### 페이지네이션

- 쿼리 파라미터: page, limit, sort, order
- 기본값: page=1, limit=20
- 메타데이터 응답: total, page, limit, has_more

## 서브태스크

- [x] 워크스페이스 모델 정의
- [x] 스토리지 인터페이스 설계
- [x] 메모리 스토리지 구현
- [x] 컨트롤러 로직 구현
- [x] 입력 검증 로직 추가
- [x] 페이지네이션 구현
- [x] 통합 테스트 작성

## 관련 링크

- Gin Binding: https://gin-gonic.com/docs/examples/binding-and-validation/
- Go Validator: https://github.com/go-playground/validator

## 출력 로그

### 2025-07-21 16:11
- 워크스페이스 모델 정의 완료 (`internal/models/workspace.go`)
  - ID, 이름, 프로젝트 경로, 상태, 소유자 정보 포함
  - Gin binding 태그와 커스텀 validator 적용
  - WorkspaceStatus enum 타입으로 상태 관리
  
- 페이지네이션 모델 생성 (`internal/models/pagination.go`)
  - 재사용 가능한 페이지네이션 요청/응답 구조체
  - 정렬 및 필터링 옵션 지원
  
- 공통 응답 모델 생성 (`internal/models/common.go`)
  - 일관된 성공/에러 응답 포맷 정의
  
- 스토리지 인터페이스 설계 (`internal/storage/interface.go`)
  - Storage 인터페이스와 WorkspaceStorage 인터페이스 정의
  - CRUD 작업 및 페이지네이션 메서드 포함
  
- 메모리 기반 스토리지 구현 (`internal/storage/memory/`)
  - 개발/테스트용 인메모리 구현
  - 동시성 안전성을 위한 mutex 사용
  - 이름 기반 인덱싱으로 빠른 중복 검사
  - Soft delete 패턴 구현
  
- 커스텀 validator 추가 (`internal/utils/validator.go`)
  - 디렉토리 존재 여부 검증 (dir 태그)
  - 안전한 경로 검증 (safepath 태그)
  - 프로젝트 경로 유효성 검사 함수
  
- 워크스페이스 컨트롤러 완성 (`internal/api/controllers/workspace.go`)
  - 모든 CRUD 엔드포인트 구현
  - JWT 인증 통합
  - 권한 기반 접근 제어
  - 페이지네이션 지원
  - 에러 처리 및 검증
  
- 라우터 설정 업데이트 (`internal/server/router.go`)
  - 스토리지 인스턴스 주입
  
- 미들웨어 에러 헬퍼 추가 (`internal/middleware/error.go`)
  - ConflictError 헬퍼 함수 추가 (409 상태 코드)
  
- 통합 테스트 작성 (`internal/api/controllers/workspace_test.go`)
  - 모든 CRUD 작업 테스트
  - 권한 검증 테스트
  - 에러 케이스 처리 테스트

모든 서브태스크가 성공적으로 완료되었으며, 워크스페이스 관리 API가 완전히 구현되었습니다.