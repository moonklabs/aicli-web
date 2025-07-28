#!/bin/bash
# AICode Manager 테스트 서버 실행 스크립트

set -e

echo "🚀 AICode Manager 테스트 서버 시작..."

# 색상 정의
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 환경 변수 설정
export GO_ENV=development
export API_PORT=8080
export LOG_LEVEL=debug

# .env 파일 확인
if [ ! -f .env ]; then
    echo -e "${YELLOW}⚠️  .env 파일이 없습니다. .env.example을 복사합니다...${NC}"
    if [ -f .env.example ]; then
        cp .env.example .env
        echo -e "${GREEN}✅ .env 파일 생성 완료${NC}"
    fi
fi

# Claude API 키 확인
if [ -z "$CLAUDE_API_KEY" ]; then
    echo -e "${YELLOW}⚠️  CLAUDE_API_KEY가 설정되지 않았습니다.${NC}"
    echo -e "${YELLOW}   .env 파일에 CLAUDE_API_KEY를 설정하거나 환경 변수로 설정하세요.${NC}"
fi

# 옵션 파싱
USE_DOCKER=false
USE_WEB=false
BUILD_ONLY=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --docker)
            USE_DOCKER=true
            shift
            ;;
        --with-web)
            USE_WEB=true
            shift
            ;;
        --build-only)
            BUILD_ONLY=true
            shift
            ;;
        --help)
            echo "사용법: ./start-test-server.sh [옵션]"
            echo ""
            echo "옵션:"
            echo "  --docker      Docker Compose로 실행"
            echo "  --with-web    웹 프론트엔드도 함께 실행"
            echo "  --build-only  빌드만 수행"
            echo "  --help        도움말 표시"
            exit 0
            ;;
        *)
            echo -e "${RED}❌ 알 수 없는 옵션: $1${NC}"
            exit 1
            ;;
    esac
done

# Docker로 실행
if [ "$USE_DOCKER" = true ]; then
    echo -e "${BLUE}🐳 Docker Compose로 실행합니다...${NC}"
    docker-compose up -d
    echo -e "${GREEN}✅ Docker 컨테이너 시작 완료${NC}"
    echo ""
    echo "서비스 접속 정보:"
    echo "  - API 서버: http://localhost:8080"
    echo "  - 헬스체크: http://localhost:8080/health"
    echo ""
    echo "로그 확인: docker-compose logs -f"
    echo "종료: docker-compose down"
    exit 0
fi

# 빌드 수행
echo -e "${BLUE}🔨 프로젝트 빌드 중...${NC}"

# Go 모듈 다운로드
echo "📦 Go 모듈 다운로드..."
go mod download

# API 서버 빌드 시도
echo "🏗️  API 서버 빌드..."
if go build -o ./build/aicli-api ./cmd/api/main.go 2>/dev/null; then
    echo -e "${GREEN}✅ API 서버 빌드 성공${NC}"
    API_BINARY="./build/aicli-api"
else
    echo -e "${YELLOW}⚠️  API 서버 빌드 실패. 직접 실행을 시도합니다...${NC}"
    API_BINARY=""
fi

# CLI 도구 빌드 시도
echo "🏗️  CLI 도구 빌드..."
if go build -o ./build/aicli ./cmd/aicli/main.go 2>/dev/null; then
    echo -e "${GREEN}✅ CLI 도구 빌드 성공${NC}"
else
    echo -e "${YELLOW}⚠️  CLI 도구 빌드 실패. simple_main.go로 빌드를 시도합니다...${NC}"
    if [ -f simple_main.go ]; then
        go build -o ./build/aicli-simple ./simple_main.go
        echo -e "${GREEN}✅ 간단한 CLI 빌드 성공${NC}"
    fi
fi

if [ "$BUILD_ONLY" = true ]; then
    echo -e "${GREEN}✅ 빌드 완료${NC}"
    exit 0
fi

# API 서버 실행
echo ""
echo -e "${BLUE}🚀 API 서버 시작...${NC}"

# PID 파일로 프로세스 관리
PID_FILE=".test-server.pid"

# 기존 프로세스 종료
if [ -f "$PID_FILE" ]; then
    OLD_PID=$(cat "$PID_FILE")
    if kill -0 "$OLD_PID" 2>/dev/null; then
        echo "기존 서버 프로세스(PID: $OLD_PID) 종료 중..."
        kill "$OLD_PID"
        sleep 2
    fi
    rm -f "$PID_FILE"
fi

# API 서버 실행
if [ -n "$API_BINARY" ]; then
    # 빌드된 바이너리 실행
    $API_BINARY serve --port 8080 &
else
    # go run으로 직접 실행
    go run ./cmd/api/main.go serve --port 8080 &
fi

API_PID=$!
echo $API_PID > "$PID_FILE"

# 서버 시작 대기
echo "서버 시작 대기 중..."
sleep 3

# 헬스체크
echo -e "${BLUE}🔍 헬스체크 수행...${NC}"
if curl -s http://localhost:8080/health > /dev/null 2>&1; then
    echo -e "${GREEN}✅ API 서버가 정상적으로 실행 중입니다!${NC}"
else
    echo -e "${YELLOW}⚠️  API 서버 헬스체크 실패. 로그를 확인하세요.${NC}"
fi

# 웹 프론트엔드 실행
if [ "$USE_WEB" = true ]; then
    echo ""
    echo -e "${BLUE}🌐 웹 프론트엔드 시작...${NC}"
    cd web
    
    # npm 설치 확인
    if ! command -v npm &> /dev/null; then
        echo -e "${RED}❌ npm이 설치되어 있지 않습니다.${NC}"
        exit 1
    fi
    
    # 의존성 설치
    if [ ! -d "node_modules" ]; then
        echo "📦 npm 패키지 설치 중..."
        npm install
    fi
    
    # 개발 서버 실행
    npm run dev &
    WEB_PID=$!
    echo $WEB_PID >> "../$PID_FILE"
    cd ..
    
    sleep 3
    echo -e "${GREEN}✅ 웹 프론트엔드가 실행 중입니다!${NC}"
fi

# 실행 정보 출력
echo ""
echo "========================================="
echo -e "${GREEN}✅ AICode Manager 테스트 서버 실행 완료!${NC}"
echo "========================================="
echo ""
echo "접속 정보:"
echo "  - API 서버: http://localhost:8080"
echo "  - API 문서: http://localhost:8080/swagger"
echo "  - 헬스체크: http://localhost:8080/health"
if [ "$USE_WEB" = true ]; then
    echo "  - 웹 UI: http://localhost:5173"
fi
echo ""
echo "주요 엔드포인트:"
echo "  - GET  /api/v1/system/info"
echo "  - GET  /api/v1/workspaces"
echo "  - POST /api/v1/auth/login"
echo ""
echo "프로세스 정보:"
echo "  - API 서버 PID: $API_PID"
if [ "$USE_WEB" = true ]; then
    echo "  - 웹 서버 PID: $WEB_PID"
fi
echo ""
echo -e "${YELLOW}종료하려면 Ctrl+C를 누르거나 ./stop-test-server.sh를 실행하세요.${NC}"
echo ""

# 시그널 핸들러 설정
cleanup() {
    echo ""
    echo "서버 종료 중..."
    if [ -f "$PID_FILE" ]; then
        while read pid; do
            if kill -0 "$pid" 2>/dev/null; then
                kill "$pid"
            fi
        done < "$PID_FILE"
        rm -f "$PID_FILE"
    fi
    echo "종료 완료"
    exit 0
}

trap cleanup SIGINT SIGTERM

# 서버 실행 유지
wait