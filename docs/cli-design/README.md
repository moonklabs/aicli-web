# AICode Manager - CLI 기반 설계

## 🚀 개요

이 문서는 Go/Rust 기반의 네이티브 CLI 도구로 설계된 AICode Manager의 아키텍처와 구현 방안을 설명합니다. Python subprocess 대신 더 효율적이고 성능이 뛰어난 네이티브 구현을 목표로 합니다.

## 🎯 설계 목표

1. **고성능**: 컴파일된 바이너리로 빠른 실행 속도
2. **낮은 리소스 사용량**: Python 대비 메모리 효율성
3. **쉬운 배포**: 단일 바이너리로 의존성 없이 배포
4. **우수한 동시성**: Go의 고루틴 또는 Rust의 async/await 활용
5. **안정성**: 타입 안전성과 메모리 안전성 보장

## 🏗️ 언어 선택: Go vs Rust

### Go 선택 이유
- **간단한 문법**: 빠른 개발과 유지보수
- **우수한 동시성**: 고루틴과 채널로 병렬 처리
- **Docker 친화적**: Docker와 같은 언어로 생태계 통합
- **빠른 컴파일**: 개발 생산성 향상
- **강력한 표준 라이브러리**: 웹 서버, JSON 처리 등 내장

### Rust 대안
- **최고 성능**: Zero-cost abstractions
- **메모리 안전성**: 컴파일 타임 보장
- **고급 타입 시스템**: 더 안전한 코드
- **WebAssembly 지원**: 브라우저 통합 가능

## 📋 주요 구성 요소

### 1. CLI 도구 (AICLI CLI)
- 사용자 친화적인 커맨드라인 인터페이스
- 로컬/원격 작업 관리
- 실시간 로그 스트리밍

### 2. 웹 서버
- Go: Gin/Echo/Fiber 프레임워크
- Rust: Actix-web/Rocket/Axum
- RESTful API + WebSocket 지원

### 3. Claude CLI 래퍼
- os/exec (Go) 또는 std::process (Rust) 사용
- 프로세스 생명주기 관리
- 스트림 처리 및 버퍼링

### 4. 컨테이너 관리
- Docker SDK 직접 사용
- 워크스페이스 격리
- 리소스 모니터링

## 📚 문서 구조

1. **architecture.md** - 시스템 아키텍처
2. **cli-implementation.md** - CLI 도구 구현 상세
3. **claude-wrapper.md** - Claude CLI 래핑 전략
4. **docker-integration.md** - Docker 통합 방법
5. **api-design.md** - API 설계 명세
6. **deployment.md** - 배포 가이드

## 🔍 Python 설계와의 차이점

| 항목 | Python 설계 | CLI 설계 |
|------|------------|----------|
| 실행 방식 | 인터프리터 | 컴파일된 바이너리 |
| 의존성 | pip 패키지 | 정적 링킹 |
| 배포 | Docker 필수 | 단일 실행 파일 |
| 성능 | 보통 | 뛰어남 |
| 메모리 | 높음 | 낮음 |
| 동시성 | asyncio | 고루틴/async |

## 🚀 빠른 시작

```bash
# Go 버전
go install github.com/yourusername/aicli-web/cmd/aicli@latest

# 또는 바이너리 다운로드
curl -L https://github.com/yourusername/aicli-web/releases/latest/download/aicli-$(uname -s)-$(uname -m) -o /usr/local/bin/aicli
chmod +x /usr/local/bin/aicli

# 사용
aicli workspace list
aicli task create --workspace my-project "Fix the bug in main.py"
aicli logs -f task-id
```