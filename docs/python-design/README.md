# AICode Manager - Docker 기반 Claude CLI 웹 플랫폼

## 🚀 프로젝트 소개

AICode Manager는 Terragon에서 영감을 받아 개발된 Docker 기반의 Claude CLI 관리 플랫폼입니다. 여러 프로젝트의 AI 코딩 작업을 웹 인터페이스에서 병렬로 실행하고 모니터링할 수 있습니다.

### 주요 특징

- 🔒 **격리된 샌드박스**: 각 프로젝트는 독립적인 Docker 컨테이너에서 실행
- 🚦 **병렬 에이전트**: 여러 Claude 인스턴스를 동시에 실행하여 작업 효율성 극대화
- 🌐 **웹 기반 인터페이스**: 어디서든 브라우저로 접속하여 작업 관리
- 📁 **워크스페이스 통합**: 로컬 프로젝트들을 자동으로 인식하고 관리
- 🔐 **보안 환경**: Supabase Auth 기반 사용자 인증 및 권한 관리
- 💰 **Claude Max 지원**: API 비용 없이 Claude Max 구독 활용 가능

## 🎯 Terragon과의 비교

| 기능 | AICode Manager | Terragon |
|------|---------------|----------|
| 실행 환경 | 로컬 Docker | 클라우드 |
| 비용 | 무료 (자체 호스팅) | 유료 서비스 |
| Claude Max 지원 | ✅ | ❌ |
| Git 워크플로우 | ✅ | ✅ |
| 병렬 실행 | ✅ | ✅ |
| 웹 인터페이스 | ✅ | ✅ |

## 📋 요구사항

- Docker 및 Docker Compose
- Python 3.9+
- Node.js 18+ (Claude CLI용)
- Claude Max 구독 (선택사항)

## 🛠️ 빠른 시작

### 1. 저장소 클론
```bash
git clone https://github.com/yourusername/aicli-web.git
cd aicli-web
```

### 2. 환경 설정
```bash
cp .env.example .env
# .env 파일을 편집하여 필요한 설정 입력
```

### 3. Docker 컨테이너 실행
```bash
docker-compose up -d
```

### 4. 웹 인터페이스 접속
브라우저에서 `http://localhost:8000` 접속

## 🏗️ 프로젝트 구조

```
aicli-web/
├── docs/               # 프로젝트 문서
├── backend/           # Python FastAPI 백엔드
│   ├── api/          # API 엔드포인트
│   ├── services/     # 비즈니스 로직
│   └── models/       # 데이터 모델
├── frontend/          # Vue.js/React 프론트엔드
│   ├── src/
│   └── public/
├── docker/            # Docker 관련 파일
│   ├── claude/       # Claude CLI 컨테이너
│   └── workspace/    # 워크스페이스 컨테이너
└── docker-compose.yml
```

## 🔧 주요 기능

### 1. 워크스페이스 관리
- 로컬 프로젝트 자동 감지
- 프로젝트별 격리된 실행 환경
- 실시간 상태 모니터링

### 2. Claude CLI 통합
- Claude Max 구독 자동 인증
- 병렬 작업 실행
- 실시간 로그 스트리밍

### 3. Git 워크플로우
- 자동 브랜치 생성
- 커밋 및 PR 생성
- 병합 충돌 자동 해결

### 4. 실시간 모니터링
- 작업 진행 상황 추적
- 로그 실시간 확인
- 리소스 사용량 모니터링

## 📚 추가 문서

- [시스템 아키텍처](./architecture.md)
- [Claude CLI 통합 가이드](./claude-integration.md)
- [Docker 환경 설정](./docker-setup.md)
- [API 명세서](./api-specification.md)
- [보안 가이드](./security.md)
- [배포 가이드](./deployment.md)

## 🤝 기여하기

프로젝트에 기여하고 싶으시다면 [CONTRIBUTING.md](./CONTRIBUTING.md)를 참고해주세요.

## 📄 라이선스

이 프로젝트는 MIT 라이선스 하에 배포됩니다. 자세한 내용은 [LICENSE](../LICENSE) 파일을 참고하세요.

## 💬 문의

- 이슈 트래커: [GitHub Issues](https://github.com/yourusername/aicli-web/issues)
- 이메일: your-email@example.com