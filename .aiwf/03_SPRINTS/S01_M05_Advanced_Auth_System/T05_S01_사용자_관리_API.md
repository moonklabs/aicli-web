---
task_id: T05_S01
sprint_sequence_id: S01
status: open
complexity: Medium
last_updated: 2025-07-22T18:30:00+0900
---

# Task: 포괄적인 사용자 관리 API

## Description
기존의 기본 인증 API를 확장하여 사용자 프로파일 관리, 계정 설정, 보안 설정, 사용자 검색 등을 포함하는 포괄적인 사용자 관리 API 시스템을 구현합니다.

## Goal / Objectives
- 사용자 프로파일 CRUD API 구현
- 계정 설정 및 보안 설정 관리 API
- 사용자 검색 및 필터링 시스템
- 사용자 활동 로그 및 통계 API
- 계정 복구 및 비밀번호 재설정 시스템

## Acceptance Criteria
- [ ] 사용자 프로파일 CRUD API 완전 동작
- [ ] 계정 설정 (비밀번호 변경, 이메일 변경 등) API 구현
- [ ] 보안 설정 (2FA, 로그인 알림 등) API 구현
- [ ] 관리자용 사용자 검색/필터링 API 구현
- [ ] 사용자 활동 로그 조회 API 구현
- [ ] 비밀번호 재설정 및 계정 복구 시스템 동작
- [ ] 사용자 통계 및 메트릭 API 제공
- [ ] OpenAPI/Swagger 문서화 완료

## Subtasks
- [ ] 사용자 프로파일 데이터 모델 확장
- [ ] 사용자 CRUD API 컨트롤러 구현
- [ ] 계정 설정 관리 API 구현
- [ ] 보안 설정 관리 API 구현
- [ ] 사용자 검색/필터링 시스템 구현
- [ ] 활동 로그 수집 및 조회 시스템
- [ ] 비밀번호 재설정 플로우 구현
- [ ] 이메일 알림 시스템 통합
- [ ] 관리자 전용 사용자 관리 API
- [ ] API 문서화 및 테스트 작성

## 기술 가이드

### 주요 인터페이스 및 통합 지점
- `internal/api/controllers/user.go` 새 컨트롤러: 사용자 관리 API
- `internal/models/user.go` 확장: 사용자 프로파일 필드 추가
- `internal/services/user.go` 새 서비스: 사용자 비즈니스 로직
- `internal/services/notification.go` 새 서비스: 이메일/알림 서비스
- 기존 `internal/api/handlers/auth.go` 확장: 계정 관리 기능

### 특정 임포트 및 모듈 참조
```go
// 이메일 발송
"gopkg.in/mail.v2"

// 비밀번호 해싱
"golang.org/x/crypto/bcrypt"

// 임시 토큰 생성
"crypto/rand"
"encoding/hex"

// 기존 시스템
"github.com/aicli/aicli-web/internal/auth"
"github.com/aicli/aicli-web/internal/models"
"github.com/aicli/aicli-web/internal/storage"
```

### 따라야 할 기존 패턴
- `internal/api/controllers/workspace.go`의 컨트롤러 구조
- `internal/services/workspace.go`의 서비스 레이어 패턴
- `internal/api/handlers/auth.go`의 응답 구조 일관성
- 기존 에러 처리 및 검증 패턴

### 작업할 데이터베이스 모델
```go
// 확장할 User 모델
type User struct {
    ID              string            `json:"id" db:"id"`
    Username        string            `json:"username" db:"username"`
    Email           string            `json:"email" db:"email"`
    PasswordHash    string            `json:"-" db:"password_hash"`
    FirstName       *string           `json:"first_name" db:"first_name"`
    LastName        *string           `json:"last_name" db:"last_name"`
    Avatar          *string           `json:"avatar" db:"avatar"`
    Bio             *string           `json:"bio" db:"bio"`
    Location        *string           `json:"location" db:"location"`
    Website         *string           `json:"website" db:"website"`
    TwoFactorEnabled bool             `json:"two_factor_enabled" db:"two_factor_enabled"`
    EmailVerified   bool              `json:"email_verified" db:"email_verified"`
    IsActive        bool              `json:"is_active" db:"is_active"`
    LastLoginAt     *time.Time        `json:"last_login_at" db:"last_login_at"`
    CreatedAt       time.Time         `json:"created_at" db:"created_at"`
    UpdatedAt       time.Time         `json:"updated_at" db:"updated_at"`
}

// 새로운 관련 모델들
type UserActivity struct {
    ID          string    `json:"id" db:"id"`
    UserID      string    `json:"user_id" db:"user_id"`
    Action      string    `json:"action" db:"action"`
    Resource    string    `json:"resource" db:"resource"`
    Details     string    `json:"details" db:"details"`
    IPAddress   string    `json:"ip_address" db:"ip_address"`
    UserAgent   string    `json:"user_agent" db:"user_agent"`
    Timestamp   time.Time `json:"timestamp" db:"timestamp"`
}

type PasswordResetToken struct {
    Token     string    `json:"token" db:"token"`
    UserID    string    `json:"user_id" db:"user_id"`
    ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
    UsedAt    *time.Time `json:"used_at" db:"used_at"`
}
```

### 에러 처리 접근법
- 중복 사용자명/이메일 시 409 Conflict 응답
- 권한 부족 시 403 Forbidden (타 사용자 정보 접근 제한)
- 유효하지 않은 토큰 시 400 Bad Request
- 존재하지 않는 사용자 시 404 Not Found

## 구현 노트

### 단계별 구현 접근법
1. **데이터 모델 확장**: User 모델에 프로파일 필드 추가
2. **기본 CRUD API**: 사용자 정보 조회/수정 API
3. **계정 관리**: 비밀번호/이메일 변경 API
4. **보안 기능**: 2FA, 로그인 알림 설정 API
5. **검색/필터링**: 관리자용 사용자 관리 기능
6. **복구 시스템**: 비밀번호 재설정 플로우

### 존중해야 할 주요 아키텍처 결정
- 기존 JWT 인증 시스템과의 완전한 호환성
- 프라이버시 보호: 민감한 정보는 적절한 권한 검사 후 노출
- RESTful API 설계 원칙 준수

### 기존 테스트 패턴을 바탕으로 한 테스트 접근법
- `internal/api/controllers/workspace_test.go` 패턴 활용
- 권한 검사 시나리오별 테스트
- 이메일 발송 기능 모킹 테스트

### 성능 고려사항
- 사용자 검색 시 인덱스 최적화
- 활동 로그 대량 데이터 처리 최적화
- 프로파일 이미지 업로드 시 용량 제한 및 최적화

## Output Log
*(This section is populated as work progresses on the task)*

[YYYY-MM-DD HH:MM:SS] Started task
[YYYY-MM-DD HH:MM:SS] Modified files: file1.js, file2.js
[YYYY-MM-DD HH:MM:SS] Completed subtask: Implemented feature X
[YYYY-MM-DD HH:MM:SS] Task completed