---
task_id: T05_S01
sprint_sequence_id: S01
status: open
complexity: Medium
last_updated: 2025-07-27T20:20:00+0900
---

# Task: 에이전트 API 엔드포인트 구현

## Description
에이전트 관리를 위한 RESTful API 엔드포인트를 구현합니다. 에이전트 CRUD 작업, 시작/중지 제어, 상태 조회, 로그 스트리밍 등의 기능을 제공하며, OpenAPI 문서화를 포함합니다.

## Goal / Objectives
- 에이전트 관리 RESTful API 설계 및 구현
- 에이전트 생명주기 제어 엔드포인트 구축
- 실시간 상태 조회 및 로그 스트리밍 API
- OpenAPI/Swagger 문서화 완성

## Acceptance Criteria
- [ ] 모든 에이전트 API 엔드포인트가 구현됨
- [ ] API 응답이 일관된 형식을 따름
- [ ] 적절한 HTTP 상태 코드 반환
- [ ] 인증 및 권한 검증이 적용됨
- [ ] OpenAPI 문서가 자동 생성됨
- [ ] API 테스트 커버리지 80% 이상

## Subtasks
- [ ] API 라우트 정의 및 컨트롤러 구조 설계
- [ ] 에이전트 CRUD 엔드포인트 구현
- [ ] 에이전트 제어 엔드포인트 구현 (start/stop/restart)
- [ ] 상태 조회 및 모니터링 엔드포인트 구현
- [ ] 로그 스트리밍 엔드포인트 구현
- [ ] 요청/응답 모델 정의 및 검증
- [ ] OpenAPI 주석 추가 및 문서 생성
- [ ] API 테스트 작성

## 기술 가이드

### 주요 인터페이스 및 통합 지점
- **기존 API 컨트롤러 패턴**: `internal/api/controllers/workspace.go`
- **라우터 설정**: `internal/server/routes.go`
- **미들웨어**: `internal/middleware/auth.go`, `internal/middleware/rbac.go`
- **WebSocket 핸들러**: `internal/api/websocket/`

### 특정 임포트 및 모듈 참조
```go
import (
    "github.com/gin-gonic/gin"
    "github.com/swaggo/swag"
    "aicli-web/internal/api/controllers"
    "aicli-web/internal/models"
    "aicli-web/internal/middleware"
)
```

### 따라야 할 기존 패턴
- 컨트롤러 구조: 의존성 주입, 메서드 바인딩
- 에러 응답: `middleware.ErrorResponse` 사용
- 페이지네이션: `models.PaginationParams` 활용
- 요청 검증: `gin.ShouldBindJSON`, 커스텀 검증자

### API 엔드포인트 구조
```
/api/v1/agents
├── GET    /                 # 에이전트 목록 조회
├── POST   /                 # 에이전트 생성
├── GET    /:id             # 에이전트 상세 조회
├── PUT    /:id             # 에이전트 수정
├── DELETE /:id             # 에이전트 삭제
├── POST   /:id/start       # 에이전트 시작
├── POST   /:id/stop        # 에이전트 중지
├── POST   /:id/restart     # 에이전트 재시작
├── GET    /:id/status      # 상태 조회
├── GET    /:id/logs        # 로그 조회
├── WS     /:id/logs/stream # 로그 스트리밍 (WebSocket)
└── GET    /:id/metrics     # 메트릭 조회
```

### 오류 처리 접근법
- 표준 HTTP 상태 코드 사용
- 구조화된 에러 응답 (code, message, details)
- 유효성 검증 에러 상세 정보 포함
- 클라이언트/서버 에러 구분

## 구현 노트

### 단계별 구현 접근법
1. `internal/api/controllers/agent.go` 생성
2. 요청/응답 모델 정의 (`internal/api/models/agent.go`)
3. 기본 CRUD 엔드포인트 구현
4. 제어 엔드포인트 구현
5. 로그 스트리밍 WebSocket 핸들러 구현
6. 라우터에 엔드포인트 등록
7. OpenAPI 주석 추가

### 주요 아키텍처 결정
- 서비스 계층을 통한 비즈니스 로직 처리
- 요청/응답 DTO와 도메인 모델 분리
- WebSocket을 통한 실시간 로그 스트리밍
- RBAC 기반 권한 검증

### 테스트 접근법
- httptest를 사용한 API 테스트
- Mock 서비스를 사용한 컨트롤러 테스트
- 인증/권한 시나리오 테스트
- WebSocket 연결 테스트

### 성능 고려사항
- 목록 조회 시 페이지네이션 필수
- 필터링 및 정렬 옵션 제공
- 응답 캐싱 고려
- 동시 요청 제한 (rate limiting)

### OpenAPI 문서화 예시
```go
// @Summary 에이전트 생성
// @Description 새로운 에이전트를 생성합니다
// @Tags agents
// @Accept json
// @Produce json
// @Param request body CreateAgentRequest true "에이전트 생성 요청"
// @Success 201 {object} AgentResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/agents [post]
```

## Output Log
*(This section is populated as work progresses on the task)*