# AICode Manager 문서

Claude CLI를 웹 플랫폼으로 관리하는 통합 솔루션에 오신 것을 환영합니다.

## 개요

AICode Manager는 Anthropic의 Claude CLI를 웹 브라우저에서 쉽게 사용할 수 있도록 해주는 관리 플랫폼입니다. 개발자와 팀이 Claude AI를 활용하여 프로젝트를 효율적으로 관리하고 작업을 자동화할 수 있습니다.

## 주요 기능

### 🚀 **워크스페이스 관리**
- 각 프로젝트별 격리된 작업 환경 제공
- Docker 컨테이너 기반 안전한 실행 환경
- 실시간 로그 모니터링 및 상태 추적

### 🔒 **고급 인증 시스템**
- OAuth2 통합 (Google, GitHub)
- JWT 기반 세션 관리
- 역할 기반 접근 제어 (RBAC)
- 다단계 보안 정책

### 🌐 **웹 인터페이스**
- 직관적인 사용자 인터페이스
- 실시간 웹소켓 통신
- 반응형 디자인
- 다크/라이트 테마 지원

### ⚡ **고성능 아키텍처**
- Go 언어 기반 고성능 백엔드
- Redis 기반 분산 캐싱
- 데이터베이스 연결 풀링
- 자동 스케일링 지원

### 🔧 **개발자 친화적**
- RESTful API 제공
- OpenAPI/Swagger 문서화
- CLI 도구 지원
- 확장 가능한 플러그인 아키텍처

## 빠른 시작

### 1. 설치

=== "Docker Compose"
    ```bash
    git clone https://github.com/your-org/aicli-web.git
    cd aicli-web
    cp .env.example .env
    docker-compose up -d
    ```

=== "바이너리"
    ```bash
    wget https://github.com/your-org/aicli-web/releases/latest/aicli-web
    chmod +x aicli-web
    ./aicli-web --config config.yml
    ```

=== "소스 빌드"
    ```bash
    git clone https://github.com/your-org/aicli-web.git
    cd aicli-web
    make build
    ./bin/aicli-web
    ```

### 2. 초기 설정

1. 환경 변수 설정:
```bash
# .env 파일 편집
ANTHROPIC_API_KEY=your-claude-api-key
JWT_SECRET_KEY=your-jwt-secret
DATABASE_URL=postgres://user:pass@localhost/aicli
```

2. 데이터베이스 초기화:
```bash
./aicli-web migrate up
```

3. 관리자 계정 생성:
```bash
./aicli-web create-admin --email admin@example.com --password yourpassword
```

### 3. 첫 번째 워크스페이스 생성

1. 웹 브라우저에서 `http://localhost:8080` 접속
2. 로그인 후 "새 워크스페이스" 버튼 클릭
3. 프로젝트 디렉토리 선택 및 워크스페이스 설정
4. Claude CLI 작업 실행

## 아키텍처

```mermaid
graph TB
    A[웹 브라우저] --> B[Load Balancer]
    B --> C[API Server]
    C --> D[워크스페이스 관리자]
    C --> E[인증 서비스]
    C --> F[세션 관리]
    
    D --> G[Docker 컨테이너]
    G --> H[Claude CLI]
    
    E --> I[OAuth2 Provider]
    F --> J[Redis]
    C --> K[PostgreSQL]
    
    L[모니터링] --> C
    M[로깅] --> C
```

## 사용 사례

### 개발팀
- 코드 리뷰 자동화
- 문서 생성 및 업데이트
- 테스트 케이스 작성
- 리팩토링 지원

### 콘텐츠 팀
- 기술 문서 작성
- 블로그 포스트 생성
- 번역 작업
- SEO 최적화

### 프로젝트 관리
- 작업 계획 수립
- 진척 상황 보고서 생성
- 리스크 분석
- 의사결정 지원

## 지원되는 환경

| 구분 | 지원 버전 |
|------|-----------|
| **운영체제** | Linux, macOS, Windows |
| **Go** | 1.21+ |
| **PostgreSQL** | 13+ |
| **Redis** | 6.0+ |
| **Docker** | 20.10+ |
| **브라우저** | Chrome 90+, Firefox 88+, Safari 14+ |

## 라이선스

이 프로젝트는 MIT 라이선스 하에 배포됩니다. 자세한 내용은 [LICENSE](https://github.com/your-org/aicli-web/blob/main/LICENSE) 파일을 참조하세요.

## 기여하기

AICode Manager 프로젝트에 기여해 주셔서 감사합니다!

- 🐛 [버그 신고](https://github.com/your-org/aicli-web/issues/new?template=bug_report.md)
- 💡 [기능 제안](https://github.com/your-org/aicli-web/issues/new?template=feature_request.md)
- 📖 [문서 개선](https://github.com/your-org/aicli-web/edit/main/docs/)
- 🔧 [코드 기여](development/contributing.md)

## 지원

### 커뮤니티 지원
- [GitHub Discussions](https://github.com/your-org/aicli-web/discussions)
- [Discord 채널](https://discord.gg/aicli-web)
- [Stack Overflow](https://stackoverflow.com/questions/tagged/aicli-web)

### 상용 지원
엔터프라이즈급 지원이 필요하신 경우 [support@yourdomain.com](mailto:support@yourdomain.com)으로 연락주세요.

---

## 다음 단계

<div class="grid cards" markdown>

-   :material-rocket-launch:{ .lg .middle } **빠른 시작**

    ---

    5분 안에 AICode Manager를 설치하고 실행하는 방법을 배워보세요.

    [:octicons-arrow-right-24: 빠른 시작 가이드](introduction/quickstart.md)

-   :material-book-open:{ .lg .middle } **사용자 가이드**

    ---

    워크스페이스, 프로젝트, 태스크 관리 방법을 자세히 알아보세요.

    [:octicons-arrow-right-24: 사용자 가이드](user-guide/workspaces.md)

-   :material-api:{ .lg .middle } **API 참조**

    ---

    RESTful API와 WebSocket API 사용법을 확인하세요.

    [:octicons-arrow-right-24: API 문서](api/overview.md)

-   :material-cog:{ .lg .middle } **관리자 가이드**

    ---

    시스템 설치, 배포, 운영 방법을 알아보세요.

    [:octicons-arrow-right-24: 관리자 가이드](admin/architecture.md)

</div>