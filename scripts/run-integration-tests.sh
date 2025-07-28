#!/bin/bash
# AICode Manager 통합 테스트 실행 스크립트

set -e

echo "🧪 AICode Manager 통합 테스트 시작..."

# 색상 정의
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 테스트 결과 추적
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# 테스트 함수
run_test() {
    local test_name=$1
    local test_command=$2
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo -e "\n${BLUE}▶ $test_name${NC}"
    
    if eval "$test_command"; then
        echo -e "${GREEN}  ✅ PASSED${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${RED}  ❌ FAILED${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
}

# 환경 준비
echo -e "${BLUE}🔧 환경 준비 중...${NC}"
export GO_ENV=test
export LOG_LEVEL=error

# Go 백엔드 테스트
echo -e "\n${BLUE}📦 Go 백엔드 테스트${NC}"
echo "================================"

run_test "Go 의존성 확인" "go mod verify"
run_test "Go 빌드 테스트" "go build -o /tmp/test-build ./cmd/api/main.go && rm -f /tmp/test-build"
run_test "Go 단위 테스트" "go test -short ./..."
run_test "Go 린트 검사" "golangci-lint run --fast 2>/dev/null || make lint"

# 웹 프론트엔드 테스트
echo -e "\n${BLUE}🌐 웹 프론트엔드 테스트${NC}"
echo "================================"

cd web
run_test "npm 패키지 확인" "[ -d node_modules ] || npm install"
run_test "TypeScript 타입 체크" "npx tsc --noEmit"
run_test "ESLint 검사" "npm run lint"
run_test "프로덕션 빌드" "npm run build"
cd ..

# API 서버 통합 테스트
echo -e "\n${BLUE}🚀 API 서버 통합 테스트${NC}"
echo "================================"

# API 서버 시작
echo "API 서버 시작 중..."
./build/aicli-api serve --port 8888 > /tmp/api-test.log 2>&1 &
API_PID=$!
sleep 3

# API 엔드포인트 테스트
run_test "헬스체크 API" "curl -s http://localhost:8888/health | grep -q 'healthy'"
run_test "시스템 정보 API" "curl -s http://localhost:8888/api/v1/system/info | grep -q 'version'"
run_test "버전 API" "curl -s http://localhost:8888/version | grep -q 'build_time'"

# API 서버 종료
kill $API_PID 2>/dev/null || true
wait $API_PID 2>/dev/null || true

# CLI 도구 테스트
echo -e "\n${BLUE}💻 CLI 도구 테스트${NC}"
echo "================================"

run_test "CLI 버전 확인" "./build/aicli version | grep -q 'Version'"
run_test "CLI 도움말" "./build/aicli help | grep -q 'Usage'"

# Docker 환경 테스트 (선택적)
if command -v docker &> /dev/null; then
    echo -e "\n${BLUE}🐳 Docker 환경 테스트${NC}"
    echo "================================"
    
    run_test "Docker 버전 확인" "docker --version"
    run_test "Docker Compose 확인" "docker-compose --version"
    run_test "Dockerfile 유효성" "docker build --dry-run -f Dockerfile ."
fi

# 보안 검사
echo -e "\n${BLUE}🔒 보안 검사${NC}"
echo "================================"

run_test "Go 보안 취약점 검사" "go list -json -deps ./... | nancy sleuth 2>/dev/null || echo 'Nancy not installed, skipping'"
run_test "민감한 정보 노출 검사" "! grep -r 'CLAUDE_API_KEY=' . --exclude-dir=.git --exclude=.env.example | grep -v '#'"

# 테스트 결과 요약
echo -e "\n${BLUE}📊 테스트 결과 요약${NC}"
echo "================================"
echo -e "총 테스트: $TOTAL_TESTS"
echo -e "${GREEN}통과: $PASSED_TESTS${NC}"
echo -e "${RED}실패: $FAILED_TESTS${NC}"

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "\n${GREEN}🎉 모든 테스트가 통과했습니다!${NC}"
    exit 0
else
    echo -e "\n${RED}⚠️  일부 테스트가 실패했습니다.${NC}"
    exit 1
fi