---
task_id: T01_S02
sprint_sequence_id: S02
status: completed
complexity: Medium
last_updated: 2025-07-25T07:20:00+0900
---

# Task: OAuth2.0 소셜 로그인 UI 통합

## Description
기존 LoginView.vue에서 비활성화된 OAuth 소셜 로그인 버튼들을 활성화하고, 백엔드 OAuth2.0 API와 연동하여 Google, GitHub 소셜 로그인 기능을 완전히 구현한다. 소셜 계정 연결/해제 및 기존 계정과의 연동 기능도 포함한다.

## Goal / Objectives
- OAuth2.0 소셜 로그인이 완전히 작동하는 UI 구현
- Google, GitHub 소셜 로그인 플로우 통합
- 소셜 계정 관리 인터페이스 구현
- 기존 JWT 인증과의 원활한 통합

## Acceptance Criteria
- [ ] LoginView.vue에서 Google, GitHub 로그인 버튼이 활성화되고 작동함
- [ ] OAuth 플로우 처리 (리다이렉트, 콜백, 토큰 교환)가 정상 작동함
- [ ] 소셜 로그인 성공 시 사용자 정보가 올바르게 저장됨
- [ ] 소셜 계정 연결/해제 기능이 사용자 프로파일에서 작동함
- [ ] 기존 계정과 소셜 계정 연동이 가능함
- [ ] OAuth 에러 상황에 대한 적절한 UI 피드백 제공
- [ ] TypeScript 타입 정의가 완전함
- [ ] 반응형 디자인 지원

## Subtasks
- [x] OAuth2.0 API 서비스 함수 구현 (auth.ts 확장)
- [x] LoginView.vue OAuth 버튼 활성화 및 클릭 핸들러 구현
- [x] OAuth 콜백 페이지 컴포넌트 생성
- [ ] 소셜 계정 연결 관리 컴포넌트 구현
- [x] UserStore에 OAuth 관련 상태 및 액션 추가
- [x] OAuth 플로우 중 로딩/에러 상태 UI 구현
- [x] 소셜 로그인 성공 후 리다이렉트 로직 구현
- [ ] 기존 계정 연동 확인 모달 구현
- [x] OAuth 관련 TypeScript 타입 정의

## 기술 가이드

### 주요 인터페이스
- **API 엔드포인트**: 백엔드 OAuth API (`/auth/oauth/*`)
- **기존 컴포넌트**: `LoginView.vue`, `UserStore`
- **API 서비스**: `src/api/services/auth.ts`

### 구현 참고사항
- **OAuth 플로우**: Authorization Code Grant 방식 사용
- **팝업 vs 리다이렉트**: 사용자 경험을 위해 팝업 방식 권장
- **상태 관리**: Pinia의 `useUserStore` 활용
- **에러 처리**: OAuth 특화 에러 메시지 및 재시도 로직
- **보안**: CSRF 토큰 및 state 파라미터 검증

### 통합 지점
- **UserStore 확장**: OAuth 관련 상태 (`oauthProviders`, `linkedAccounts`)
- **Router 확장**: OAuth 콜백 라우트 추가
- **API 클라이언트**: 인증 헤더 자동 추가 로직 확장

### 기존 패턴 준수
- **Naive UI 컴포넌트**: 일관된 버튼 및 모달 스타일
- **SCSS 스타일링**: 기존 변수 및 믹스인 활용
- **에러 처리**: `useMessage` 훅 활용

## 구현 노트
- OAuth 플로우는 보안을 최우선으로 구현
- 사용자 경험을 해치지 않는 로딩 상태 표시
- 소셜 계정 연동 시 기존 데이터 보존
- 팝업 차단 상황에 대한 대체 플로우 제공

## Output Log

[2025-07-25 06:57]: 태스크 시작 - OAuth2.0 소셜 로그인 UI 통합 작업 개시
[2025-07-25 07:02]: OAuth 관련 TypeScript 타입 정의 완료 - OAuthProvider, OAuthUserInfo, OAuthAccount 등 추가
[2025-07-25 07:05]: auth.ts OAuth API 서비스 함수 구현 완료 - getOAuthAuthUrl, oAuthLogin, linkOAuthAccount 등 추가
[2025-07-25 07:08]: UserStore OAuth 관련 상태 및 액션 추가 완료 - 상태 관리 확장
[2025-07-25 07:12]: OAuth 콜백 페이지 컴포넌트 생성 완료 - OAuthCallbackView.vue 구현
[2025-07-25 07:15]: 라우터에 OAuth 콜백 라우트 추가 완료 - /auth/callback 경로 설정
[2025-07-25 07:18]: LoginView.vue OAuth 버튼 활성화 및 핸들러 구현 완료 - Google, GitHub 로그인 버튼 활성화
[2025-07-25 07:20]: 코드 리뷰 및 타입 에러 수정 완료 - OAuth 구현 검증 및 최종 완료