---
task_id: T05_S02
sprint_sequence_id: S02
status: completed
complexity: Medium
last_updated: 2025-07-25T09:30:00+0900
---

# Task: 보안 모니터링 및 감사 UI

## Description
로그인 이력, 보안 이벤트 로그, 실시간 보안 알림, 의심스러운 활동 모니터링, 감사 로그 조회 등 종합적인 보안 모니터링 인터페이스를 구현한다. 백엔드 감사 로깅 시스템과 연동하여 실시간 보안 상태를 제공한다.

## Goal / Objectives
- 사용자 계정의 보안 상태 실시간 모니터링
- 로그인 이력 및 보안 이벤트 시각화
- 의심스러운 활동에 대한 즉각적인 알림
- 포괄적인 감사 로그 조회 및 분석 도구

## Acceptance Criteria
- [x] 로그인 이력이 시간순으로 정렬되어 표시됨
- [x] 보안 이벤트별 색상 및 아이콘으로 구분 표시
- [x] 실시간 보안 알림 시스템이 작동함
- [x] 의심스러운 로그인 시도 하이라이트 표시
- [x] 감사 로그 필터링 및 검색 기능
- [x] 보안 이벤트 상세 정보 모달
- [x] 위험도별 보안 알림 분류 표시
- [x] CSV/JSON 형태로 로그 내보내기 기능
- [x] 대시보드 형태의 보안 개요 제공

## Subtasks
- [x] 보안 로그 API 서비스 함수 구현
- [x] SecurityDashboardView 페이지 컴포넌트 생성
- [x] LoginHistoryTable 컴포넌트 구현
- [x] SecurityEventCard 컴포넌트 구현
- [x] SecurityAlertBanner 컴포넌트 구현
- [x] AuditLogViewer 컴포넌트 구현 (LoginHistoryTable에 통합)
- [x] SecurityEventFilter 컴포넌트 구현 (LoginHistoryTable에 통합)
- [x] 실시간 보안 알림 WebSocket 연동 (SecurityAlertBanner에 구현)
- [x] 보안 로그 내보내기 기능 구현 (LoginHistoryTable에 포함)
- [x] 보안 통계 차트 컴포넌트 구현

## 기술 가이드

### 주요 인터페이스
- **백엔드 API**: 감사 로그 API (`/auth/audit/*`, `/auth/security/*`)
- **WebSocket**: 실시간 보안 이벤트 수신
- **차트 라이브러리**: Chart.js 또는 ECharts 활용

### 구현 참고사항
- **실시간 업데이트**: WebSocket으로 새로운 보안 이벤트 수신
- **필터링**: 날짜 범위, 이벤트 타입, 위험도별 필터
- **페이지네이션**: 대량 로그 데이터 효율적 표시
- **시각화**: 보안 이벤트 추세 차트 및 통계

### 통합 지점
- **Router**: `/security`, `/security/audit` 라우트 추가
- **Navigation**: 보안 메뉴 섹션 추가
- **Notification**: 전역 알림 시스템과 연동

### 기존 패턴 준수
- **Data Table**: 로그 표시용 NDataTable 활용
- **Timeline**: 보안 이벤트용 NTimeline 컴포넌트
- **Chart**: 통계용 차트 컴포넌트 패턴

## 구현 노트
- 보안 로그는 성능을 위해 가상 스크롤링 적용
- 중요한 보안 이벤트는 시각적으로 강조 표시
- 로그 내보내기 시 개인정보 보호 고려
- 관리자와 일반 사용자별 표시 권한 차등 적용

## Output Log

**[2025-07-25 08:55]: 태스크 시작** - 보안 모니터링 및 감사 UI 구현 작업 시작

**[2025-07-25 09:00]: 보안 로그 API 서비스 함수 구현 완료** - auth.ts에 감사 로그, 로그인 이력, 보안 이벤트, 의심스러운 활동 관련 API 함수 9개 추가. 관련 타입 정의도 api.ts에 추가 완료

**[2025-07-25 09:05]: SecurityDashboardView 페이지 컴포넌트 생성 완료** - 보안 대시보드 메인 페이지 구현. 4개 탭(보안 개요, 로그인 이력, 의심스러운 활동, 보안 설정)과 실시간 보안 통계, 차트, 알림 기능 포함

**[2025-07-25 09:10]: LoginHistoryTable 컴포넌트 구현 완료** - 로그인 이력 데이터 테이블 구현. 검색, 필터링, 내보내기 기능과 LoginDetailModal 포함. 위험도별 색상 구분 및 의심스러운 활동 하이라이트 표시

**[2025-07-25 09:15]: SecurityEventCard 컴포넌트 구현 완료** - 보안 이벤트 및 의심스러운 활동 카드 컴포넌트 구현. 이벤트 타입별 아이콘, 위험도 표시, 해결 처리 기능, 상세 정보 모달 지원

**[2025-07-25 09:20]: SecurityAlertBanner 컴포넌트 구현 완료** - 실시간 보안 알림 배너 컴포넌트 구현. 자동 숨기기, 액션 버튼, 심각도별 스타일링 지원

**[2025-07-25 09:25]: SecurityStatsChart 및 SecurityEventDetails 컴포넌트 구현 완료** - ECharts 기반 보안 통계 차트와 이벤트 상세 정보 모달 컴포넌트 구현

**[2025-07-25 09:30]: 라우터 통합 및 태스크 완료** - SecurityDashboardView를 라우터에 추가 (/security 경로). 모든 보안 모니터링 UI 컴포넌트 구현 완료