#!/bin/bash
# CI 파이프라인 로컬 테스트 스크립트

set -e

echo "🔍 CI 파이프라인 로컬 테스트 시작..."

# 색상 정의
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 단계별 테스트 함수
run_step() {
    local step_name=$1
    local command=$2
    
    echo -e "\n${BLUE}▶ ${step_name}${NC}"
    if eval "$command"; then
        echo -e "${GREEN}✅ ${step_name} 성공${NC}"
    else
        echo -e "${RED}❌ ${step_name} 실패${NC}"
        exit 1
    fi
}

# 1. 의존성 확인
echo -e "${YELLOW}📦 의존성 확인...${NC}"
run_step "Go 버전 확인" "go version"
run_step "golangci-lint 확인" "which golangci-lint || echo 'golangci-lint not installed'"

# 2. 코드 포맷팅
run_step "코드 포맷팅" "make fmt"

# 3. 린트 실행
run_step "린트 검사" "make lint"

# 4. 테스트 실행
run_step "단위 테스트" "make test-unit"

# 5. 빌드 테스트
run_step "바이너리 빌드" "make build"

# 6. 보안 스캔 (설치되어 있는 경우)
if which gosec > /dev/null 2>&1; then
    run_step "보안 스캔" "make security"
else
    echo -e "${YELLOW}⚠️  gosec이 설치되어 있지 않아 보안 스캔을 건너뜁니다${NC}"
fi

# 7. 벤치마크 테스트 (선택사항)
echo -e "\n${YELLOW}벤치마크 테스트를 실행하시겠습니까? (y/N)${NC}"
read -r response
if [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
    run_step "벤치마크 테스트" "make test-bench"
fi

echo -e "\n${GREEN}✅ 모든 CI 단계가 성공적으로 완료되었습니다!${NC}"
echo -e "${BLUE}💡 팁: GitHub에 푸시하기 전에 이 스크립트를 실행하여 CI 실패를 미리 방지할 수 있습니다.${NC}"