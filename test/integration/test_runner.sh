#!/bin/bash
# AICode Manager 통합 테스트 실행 스크립트

set -e

# 색상 정의
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 테스트 설정
TEST_TIMEOUT="30m"
COVERAGE_FILE="integration_coverage.out"
REPORT_DIR="test_reports"

echo -e "${BLUE}🧪 AICode Manager 통합 테스트 실행기${NC}"
echo "=================================="

# 함수들
show_usage() {
    echo "사용법: $0 [옵션]"
    echo ""
    echo "옵션:"
    echo "  -h, --help          이 도움말 표시"
    echo "  -v, --verbose       상세 출력"
    echo "  -c, --coverage      커버리지 리포트 생성"
    echo "  -r, --report        HTML 리포트 생성"
    echo "  -s, --suite SUITE   특정 테스트 스위트만 실행"
    echo "  --no-docker         Docker 테스트 제외"
    echo "  --quick             빠른 테스트 (성능 테스트 제외)"
    echo ""
    echo "테스트 스위트:"
    echo "  agent      - 에이전트 통합 테스트"
    echo "  git        - Git 통합 테스트"  
    echo "  api        - API 통합 테스트"
    echo "  performance - 성능 및 부하 테스트"
    echo ""
    echo "예시:"
    echo "  $0 -c -r                    # 전체 테스트 + 커버리지 + 리포트"
    echo "  $0 -s agent                 # 에이전트 테스트만"
    echo "  $0 --quick                  # 빠른 테스트 (성능 제외)"
}

# 명령행 인수 파싱
VERBOSE=false
COVERAGE=false
REPORT=false
SUITE=""
NO_DOCKER=false
QUICK=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_usage
            exit 0
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -c|--coverage)
            COVERAGE=true
            shift
            ;;
        -r|--report)
            REPORT=true
            shift
            ;;
        -s|--suite)
            SUITE="$2"
            shift 2
            ;;
        --no-docker)
            NO_DOCKER=true
            shift
            ;;
        --quick)
            QUICK=true
            shift
            ;;
        *)
            echo -e "${RED}알 수 없는 옵션: $1${NC}"
            show_usage
            exit 1
            ;;
    esac
done

# 환경 확인
check_prerequisites() {
    echo -e "${BLUE}🔧 환경 확인 중...${NC}"
    
    # Go 설치 확인
    if ! command -v go &> /dev/null; then
        echo -e "${RED}❌ Go가 설치되지 않았습니다${NC}"
        exit 1
    fi
    
    # Docker 확인 (필요한 경우)
    if [ "$NO_DOCKER" = false ]; then
        if ! command -v docker &> /dev/null; then
            echo -e "${YELLOW}⚠️  Docker가 설치되지 않았습니다. --no-docker 옵션을 사용하세요${NC}"
            exit 1
        fi
        
        # Docker 서비스 확인
        if ! docker info &> /dev/null; then
            echo -e "${YELLOW}⚠️  Docker 서비스가 실행되지 않았습니다${NC}"
            exit 1
        fi
    fi
    
    echo -e "${GREEN}   ✅ 환경 확인 완료${NC}"
}

# 테스트 디렉토리 준비
prepare_test_environment() {
    echo -e "${BLUE}📁 테스트 환경 준비 중...${NC}"
    
    # 리포트 디렉토리 생성
    mkdir -p "$REPORT_DIR"
    
    # 이전 커버리지 파일 제거
    rm -f "$COVERAGE_FILE"
    
    echo -e "${GREEN}   ✅ 테스트 환경 준비 완료${NC}"
}

# 테스트 실행
run_tests() {
    echo -e "${BLUE}🚀 통합 테스트 실행 중...${NC}"
    
    # 기본 테스트 플래그
    TEST_FLAGS="-timeout $TEST_TIMEOUT"
    
    if [ "$VERBOSE" = true ]; then
        TEST_FLAGS="$TEST_FLAGS -v"
    fi
    
    if [ "$COVERAGE" = true ]; then
        TEST_FLAGS="$TEST_FLAGS -coverprofile=$COVERAGE_FILE"
    fi
    
    # 특정 스위트 실행
    if [ -n "$SUITE" ]; then
        case $SUITE in
            agent)
                TEST_PATTERN="TestAgentIntegrationSuite"
                ;;
            git)
                TEST_PATTERN="TestGitIntegrationSuite"
                ;;
            api)
                TEST_PATTERN="TestAPIIntegrationSuite"
                ;;
            performance)
                TEST_PATTERN="TestPerformanceSuite"
                ;;
            *)
                echo -e "${RED}❌ 알 수 없는 테스트 스위트: $SUITE${NC}"
                exit 1
                ;;
        esac
        TEST_FLAGS="$TEST_FLAGS -run $TEST_PATTERN"
        echo -e "${YELLOW}📋 특정 스위트 실행: $SUITE${NC}"
    fi
    
    # 빠른 테스트 모드
    if [ "$QUICK" = true ]; then
        echo -e "${YELLOW}⚡ 빠른 테스트 모드 (성능 테스트 제외)${NC}"
        if [ -z "$SUITE" ]; then
            TEST_FLAGS="$TEST_FLAGS -run 'Test(Agent|Git|API).*Suite'"
        fi
    fi
    
    # Docker 제외
    if [ "$NO_DOCKER" = true ]; then
        echo -e "${YELLOW}🐳 Docker 테스트 제외${NC}"
        export SKIP_DOCKER_TESTS=true
    fi
    
    # 테스트 실행
    echo -e "${BLUE}   실행 명령: go test $TEST_FLAGS ./test/integration${NC}"
    
    cd test/integration
    if go test $TEST_FLAGS .; then
        echo -e "${GREEN}✅ 모든 테스트가 성공했습니다!${NC}"
        TESTS_PASSED=true
    else
        echo -e "${RED}❌ 일부 테스트가 실패했습니다${NC}"
        TESTS_PASSED=false
    fi
    cd ../..
}

# 커버리지 리포트 생성
generate_coverage_report() {
    if [ "$COVERAGE" = true ] && [ -f "test/integration/$COVERAGE_FILE" ]; then
        echo -e "${BLUE}📊 커버리지 리포트 생성 중...${NC}"
        
        # 커버리지 HTML 리포트
        go tool cover -html="test/integration/$COVERAGE_FILE" -o "$REPORT_DIR/coverage.html"
        
        # 커버리지 요약
        COVERAGE_PERCENT=$(go tool cover -func="test/integration/$COVERAGE_FILE" | grep total: | awk '{print $3}')
        echo -e "${GREEN}   📈 통합 테스트 커버리지: $COVERAGE_PERCENT${NC}"
        
        # 커버리지 파일을 리포트 디렉토리로 복사
        cp "test/integration/$COVERAGE_FILE" "$REPORT_DIR/"
        
        echo -e "${GREEN}   ✅ 커버리지 리포트: $REPORT_DIR/coverage.html${NC}"
    fi
}

# HTML 테스트 리포트 생성
generate_test_report() {
    if [ "$REPORT" = true ]; then
        echo -e "${BLUE}📄 테스트 리포트 생성 중...${NC}"
        
        # 기본 HTML 리포트 생성
        cat > "$REPORT_DIR/test_report.html" << EOF
<!DOCTYPE html>
<html lang="ko">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>AICode Manager 통합 테스트 리포트</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 0; padding: 20px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #2c3e50; border-bottom: 3px solid #3498db; padding-bottom: 10px; }
        h2 { color: #34495e; margin-top: 30px; }
        .status { padding: 10px 15px; border-radius: 5px; margin: 10px 0; }
        .success { background: #d4edda; color: #155724; border: 1px solid #c3e6cb; }
        .failure { background: #f8d7da; color: #721c24; border: 1px solid #f5c6cb; }
        .info { background: #d1ecf1; color: #0c5460; border: 1px solid #bee5eb; }
        .timestamp { color: #6c757d; font-size: 0.9em; }
        table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        th, td { padding: 12px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background-color: #f8f9fa; font-weight: 600; }
        .metric { text-align: center; margin: 20px; }
        .metric-value { font-size: 2em; font-weight: bold; color: #3498db; }
        .metric-label { color: #7f8c8d; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🧪 AICode Manager 통합 테스트 리포트</h1>
        <div class="timestamp">생성 시간: $(date '+%Y-%m-%d %H:%M:%S')</div>
        
        <div class="status $(if [ "$TESTS_PASSED" = true ]; then echo 'success'; else echo 'failure'; fi)">
            <strong>테스트 결과:</strong> $(if [ "$TESTS_PASSED" = true ]; then echo '✅ 성공'; else echo '❌ 실패'; fi)
        </div>
        
        <h2>📋 테스트 스위트</h2>
        <table>
            <tr>
                <th>스위트명</th>
                <th>설명</th>
                <th>상태</th>
            </tr>
            <tr>
                <td>Agent Integration</td>
                <td>에이전트 생명주기 E2E 테스트</td>
                <td>$(if [ -z "$SUITE" ] || [ "$SUITE" = "agent" ]; then echo '실행됨'; else echo '제외됨'; fi)</td>
            </tr>
            <tr>
                <td>Git Integration</td>
                <td>Git worktree 통합 테스트</td>
                <td>$(if [ -z "$SUITE" ] || [ "$SUITE" = "git" ]; then echo '실행됨'; else echo '제외됨'; fi)</td>
            </tr>
            <tr>
                <td>API Integration</td>
                <td>API 엔드포인트 통합 테스트</td>
                <td>$(if [ -z "$SUITE" ] || [ "$SUITE" = "api" ]; then echo '실행됨'; else echo '제외됨'; fi)</td>
            </tr>
            <tr>
                <td>Performance</td>
                <td>성능 및 부하 테스트</td>
                <td>$(if [ "$QUICK" = true ]; then echo '제외됨 (빠른 모드)'; elif [ -z "$SUITE" ] || [ "$SUITE" = "performance" ]; then echo '실행됨'; else echo '제외됨'; fi)</td>
            </tr>
        </table>
        
        <h2>🎯 성능 목표</h2>
        <div class="info">
            <strong>T06_S01 성능 최적화 목표:</strong>
            <ul>
                <li>100개 이상 동시 에이전트 지원</li>
                <li>에이전트 생성 시간 5초 이내 (P95 기준)</li>
                <li>메모리 사용량 선형 증가</li>
                <li>효율적인 CPU 사용률 분산</li>
                <li>자동 스케일링 동작</li>
            </ul>
        </div>
        
        $(if [ "$COVERAGE" = true ] && [ -f "$REPORT_DIR/coverage.html" ]; then echo "<h2>📊 커버리지 리포트</h2><p><a href='coverage.html'>커버리지 상세 보기</a></p>"; fi)
        
        <h2>🔧 환경 정보</h2>
        <table>
            <tr><td>Go 버전</td><td>$(go version)</td></tr>
            <tr><td>운영체제</td><td>$(uname -s) $(uname -r)</td></tr>
            <tr><td>Docker</td><td>$(if command -v docker &> /dev/null; then docker --version; else echo '사용하지 않음'; fi)</td></tr>
            <tr><td>테스트 실행 시간</td><td>$(date '+%Y-%m-%d %H:%M:%S')</td></tr>
        </table>
    </div>
</body>
</html>
EOF
        
        echo -e "${GREEN}   ✅ 테스트 리포트: $REPORT_DIR/test_report.html${NC}"
    fi
}

# 메인 실행
main() {
    echo -e "${BLUE}시작 시간: $(date '+%Y-%m-%d %H:%M:%S')${NC}"
    
    check_prerequisites
    prepare_test_environment
    run_tests
    generate_coverage_report
    generate_test_report
    
    echo ""
    echo "=================================="
    if [ "$TESTS_PASSED" = true ]; then
        echo -e "${GREEN}🎉 통합 테스트가 성공적으로 완료되었습니다!${NC}"
        exit 0
    else
        echo -e "${RED}⚠️  일부 테스트가 실패했습니다.${NC}"
        exit 1
    fi
}

# 스크립트 실행
main "$@"