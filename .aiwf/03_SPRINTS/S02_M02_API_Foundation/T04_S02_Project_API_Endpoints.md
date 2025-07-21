---
task_id: T04_S02
task_name: Project Management API Endpoints
status: pending
complexity: medium
priority: medium
created_at: 2025-07-21
updated_at: 2025-07-21
sprint_id: S02_M02
---

# T04_S02: Project Management API Endpoints

## 태스크 개요

프로젝트 관리를 위한 API 엔드포인트를 구현합니다. 워크스페이스 내에서 프로젝트를 관리하고 Claude CLI와 연동할 수 있는 기능을 제공합니다.

## 목표

- 프로젝트 CRUD API 구현
- 프로젝트와 워크스페이스 관계 관리
- Git 리포지토리 정보 통합
- 프로젝트 설정 관리

## 수용 기준

- [ ] POST /workspaces/:id/projects가 새 프로젝트 생성
- [ ] GET /workspaces/:id/projects가 워크스페이스의 프로젝트 목록 반환
- [ ] GET /projects/:id가 프로젝트 상세 정보 반환
- [ ] PUT /projects/:id가 프로젝트 정보 업데이트
- [ ] DELETE /projects/:id가 프로젝트 삭제
- [ ] Git 상태 정보가 프로젝트 응답에 포함
- [ ] 프로젝트 설정 저장/조회 가능
- [ ] 단위 및 통합 테스트 작성

## 기술 가이드

### 주요 인터페이스 및 통합 지점

1. **라우터 통합**: `internal/server/router.go`에 프로젝트 라우트 추가
2. **워크스페이스 연동**: 기존 워크스페이스 모델과 관계 설정
3. **Git 통합**: go-git 라이브러리 활용하여 Git 정보 조회
4. **설정 관리**: 프로젝트별 `.aicli/config.yaml` 파일 관리

### 구현 구조

```
internal/
├── models/
│   └── project.go       # 프로젝트 모델 정의
├── api/
│   ├── controllers/
│   │   └── project.go   # 프로젝트 컨트롤러
│   └── handlers/
│       └── project.go   # 프로젝트 핸들러
├── services/
│   ├── project.go       # 프로젝트 비즈니스 로직
│   └── git.go          # Git 통합 서비스
└── storage/
    └── project.go       # 프로젝트 저장소
```

### 기존 패턴 참조

- 컨트롤러 패턴: `internal/api/controllers/workspace.go` 참조
- 모델 구조: 워크스페이스 모델과 일관성 유지
- 에러 처리: 기존 미들웨어 에러 처리 패턴 활용

## 구현 노트

### 단계별 접근법

1. Project 모델 정의 (워크스페이스 관계 포함)
2. 프로젝트 서비스 레이어 구현
3. Git 통합 서비스 구현
4. 프로젝트 컨트롤러 구현
5. API 라우트 추가
6. 프로젝트 설정 관리 기능
7. 테스트 작성

### 프로젝트 모델

```go
type Project struct {
    ID           string    `json:"id"`
    WorkspaceID  string    `json:"workspace_id"`
    Name         string    `json:"name" binding:"required"`
    Path         string    `json:"path" binding:"required"`
    GitURL       string    `json:"git_url,omitempty"`
    GitBranch    string    `json:"git_branch,omitempty"`
    Language     string    `json:"language"`
    Status       string    `json:"status"`
    Config       Config    `json:"config"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}
```

### Git 통합

- 리포지토리 상태 확인 (clean/dirty)
- 현재 브랜치 정보
- 최근 커밋 정보
- 원격 리포지토리 URL

### 프로젝트 설정

- Claude API 키 관리 (암호화)
- 프로젝트별 환경 변수
- Claude CLI 옵션
- 빌드/테스트 명령어

## 서브태스크

- [ ] 프로젝트 모델 정의
- [ ] 프로젝트 서비스 구현
- [ ] Git 통합 서비스 구현
- [ ] 프로젝트 컨트롤러 구현
- [ ] API 라우트 설정
- [ ] 프로젝트 설정 관리
- [ ] 테스트 작성

## 관련 링크

- go-git: https://github.com/go-git/go-git
- Git 상태 API 설계: https://docs.github.com/en/rest/repos