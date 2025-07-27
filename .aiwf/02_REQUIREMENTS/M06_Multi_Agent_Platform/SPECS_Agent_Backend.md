# SPECS: Agent Backend

## 개요

멀티 에이전트 플랫폼의 백엔드 시스템 기술 사양

## 아키텍처

### 에이전트 모델
```go
type Agent struct {
    ID          string        `json:"id"`
    ProjectID   string        `json:"project_id"`
    Name        string        `json:"name"`
    Type        AgentType     `json:"type"`
    WorktreeID  string        `json:"worktree_id"`
    ContainerID string        `json:"container_id"`
    SessionID   string        `json:"session_id"`
    Status      AgentStatus   `json:"status"`
    Config      AgentConfig   `json:"config"`
    CreatedAt   time.Time     `json:"created_at"`
    UpdatedAt   time.Time     `json:"updated_at"`
}
```

### Git Worktree 관리
- go-git/go-git v5 사용
- 프로젝트별 독립 작업 공간
- 브랜치별 격리

### 스토리지
- SQLite 기반 에이전트 정보 저장
- AgentStorage 인터페이스 구현
- CRUD 작업 지원

### 서비스 로직
- 에이전트 생명주기 관리
- Docker 컨테이너 통합
- 상태 모니터링

## 구현 세부사항

### 파일 구조
```
internal/
├── models/agent.go           # 에이전트 모델
├── git/worktree_manager.go   # Git worktree 관리
├── storage/sqlite/agent.go   # SQLite 구현
└── services/agent.go         # 비즈니스 로직
```

### API 엔드포인트
```
POST   /api/projects/:id/agents          # 에이전트 생성
GET    /api/projects/:id/agents          # 에이전트 목록
GET    /api/agents/:id                   # 에이전트 상세
PUT    /api/agents/:id                   # 에이전트 수정
DELETE /api/agents/:id                   # 에이전트 삭제
POST   /api/agents/:id/start             # 에이전트 시작
POST   /api/agents/:id/stop              # 에이전트 중지
```

## 성능 요구사항

- 100개 이상 동시 에이전트 지원
- 에이전트 생성 시간 < 5초
- Git worktree 생성 시간 < 3초
- 메모리 사용량 최적화