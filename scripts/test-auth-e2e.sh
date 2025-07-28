#!/bin/bash

# E2E Authentication Tests Runner Script
# 인증 시스템 E2E 테스트 실행 스크립트

set -euo pipefail

# 색상 정의
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 로그 함수
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 설정
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
TEST_DIR="$PROJECT_ROOT/_test_disabled/e2e/auth"
WEB_DIR="$PROJECT_ROOT/web"

# 기본값
RUN_BACKEND=true
RUN_FRONTEND=true
COVERAGE=false
VERBOSE=false
PARALLEL=true
TIMEOUT="10m"

# 도움말 출력
show_help() {
    cat << EOF
Usage: $0 [OPTIONS]

E2E Authentication Tests Runner

OPTIONS:
    -h, --help           Show this help message
    -b, --backend-only   Run only backend tests
    -f, --frontend-only  Run only frontend tests
    -c, --coverage       Generate coverage reports
    -v, --verbose        Verbose output
    -s, --sequential     Run tests sequentially (not parallel)
    -t, --timeout TIME   Test timeout (default: 10m)

EXAMPLES:
    $0                      # Run all tests
    $0 -b                   # Run only backend tests
    $0 -f -c               # Run frontend tests with coverage
    $0 -v -c               # Run all tests with verbose output and coverage
    $0 -t 15m              # Run tests with 15 minute timeout

EOF
}

# 옵션 파싱
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -b|--backend-only)
            RUN_FRONTEND=false
            shift
            ;;
        -f|--frontend-only)
            RUN_BACKEND=false
            shift
            ;;
        -c|--coverage)
            COVERAGE=true
            shift
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -s|--sequential)
            PARALLEL=false
            shift
            ;;
        -t|--timeout)
            TIMEOUT="$2"
            shift 2
            ;;
        *)
            log_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# 환경 확인
check_environment() {
    log_info "Checking environment..."
    
    # Go 설치 확인
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed or not in PATH"
        exit 1
    fi
    
    # Node.js 설치 확인 (프론트엔드 테스트 실행 시)
    if [[ "$RUN_FRONTEND" == "true" ]] && ! command -v node &> /dev/null; then
        log_error "Node.js is not installed or not in PATH"
        exit 1
    fi
    
    # 프로젝트 루트 확인
    if [[ ! -f "$PROJECT_ROOT/go.mod" ]]; then
        log_error "Not in a Go project root directory"
        exit 1
    fi
    
    log_success "Environment check passed"
}

# Go 의존성 확인
check_go_dependencies() {
    log_info "Checking Go dependencies..."
    
    cd "$PROJECT_ROOT"
    
    # 필수 테스트 패키지 확인
    if ! go list -m github.com/stretchr/testify &> /dev/null; then
        log_info "Installing testify..."
        go get github.com/stretchr/testify/assert
        go get github.com/stretchr/testify/require
    fi
    
    if ! go list -m github.com/gin-gonic/gin &> /dev/null; then
        log_error "Gin framework not found in dependencies"
        exit 1
    fi
    
    log_success "Go dependencies check passed"
}

# 프론트엔드 의존성 확인
check_frontend_dependencies() {
    if [[ "$RUN_FRONTEND" != "true" ]]; then
        return 0
    fi
    
    log_info "Checking frontend dependencies..."
    
    cd "$WEB_DIR"
    
    if [[ ! -f "package.json" ]]; then
        log_error "Frontend package.json not found"
        exit 1
    fi
    
    if [[ ! -d "node_modules" ]] || [[ ! -f "node_modules/.package-lock.json" ]]; then
        log_info "Installing frontend dependencies..."
        npm ci
    fi
    
    log_success "Frontend dependencies check passed"
}

# 백엔드 테스트 실행
run_backend_tests() {
    if [[ "$RUN_BACKEND" != "true" ]]; then
        return 0
    fi
    
    log_info "Running backend E2E authentication tests..."
    
    cd "$PROJECT_ROOT"
    
    # 테스트 옵션 구성
    local test_args=()
    
    if [[ "$VERBOSE" == "true" ]]; then
        test_args+=("-v")
    fi
    
    if [[ "$PARALLEL" == "true" ]]; then
        test_args+=("-parallel" "4")
    fi
    
    test_args+=("-timeout" "$TIMEOUT")
    
    if [[ "$COVERAGE" == "true" ]]; then
        test_args+=("-coverprofile=coverage-auth-e2e.out")
        test_args+=("-covermode=atomic")
    fi
    
    # 테스트 실행
    local test_pattern="./$(basename "$TEST_DIR")"
    
    log_info "Executing: go test ${test_args[*]} $test_pattern"
    
    if go test "${test_args[@]}" "$test_pattern"; then
        log_success "Backend tests passed"
        
        # 커버리지 리포트 생성
        if [[ "$COVERAGE" == "true" ]] && [[ -f "coverage-auth-e2e.out" ]]; then
            log_info "Generating coverage report..."
            go tool cover -html=coverage-auth-e2e.out -o coverage-auth-e2e.html
            log_success "Coverage report generated: coverage-auth-e2e.html"
            
            # 커버리지 요약
            local coverage_percent=$(go tool cover -func=coverage-auth-e2e.out | tail -1 | awk '{print $3}')
            log_info "Overall coverage: $coverage_percent"
        fi
    else
        log_error "Backend tests failed"
        return 1
    fi
}

# 프론트엔드 테스트 실행
run_frontend_tests() {
    if [[ "$RUN_FRONTEND" != "true" ]]; then
        return 0
    fi
    
    log_info "Running frontend E2E authentication tests..."
    
    cd "$WEB_DIR"
    
    # 테스트 명령어 구성
    local test_cmd="npm run test:run"
    
    if [[ "$COVERAGE" == "true" ]]; then
        test_cmd="npm run test:coverage"
    fi
    
    # 특정 테스트 파일만 실행
    test_cmd="$test_cmd -- auth.test.ts"
    
    if [[ "$VERBOSE" == "true" ]]; then
        test_cmd="$test_cmd --reporter=verbose"
    fi
    
    log_info "Executing: $test_cmd"
    
    if eval "$test_cmd"; then
        log_success "Frontend tests passed"
    else
        log_error "Frontend tests failed"
        return 1
    fi
}

# 테스트 결과 요약
generate_summary() {
    log_info "Test Summary"
    echo "=================================="
    
    if [[ "$RUN_BACKEND" == "true" ]]; then
        echo "✓ Backend E2E Authentication Tests"
    fi
    
    if [[ "$RUN_FRONTEND" == "true" ]]; then
        echo "✓ Frontend E2E Authentication Tests"
    fi
    
    if [[ "$COVERAGE" == "true" ]]; then
        echo "✓ Coverage Reports Generated"
        
        if [[ -f "$PROJECT_ROOT/coverage-auth-e2e.html" ]]; then
            echo "  - Backend: file://$PROJECT_ROOT/coverage-auth-e2e.html"
        fi
        
        if [[ -d "$WEB_DIR/coverage" ]]; then
            echo "  - Frontend: file://$WEB_DIR/coverage/index.html"
        fi
    fi
    
    echo "=================================="
}

# 정리 함수
cleanup() {
    log_info "Cleaning up..."
    
    # 임시 파일 정리 (필요시)
    # rm -f /tmp/auth-e2e-*
    
    log_success "Cleanup completed"
}

# 시그널 핸들러 설정
trap cleanup EXIT INT TERM

# 메인 실행
main() {
    log_info "Starting E2E Authentication Tests"
    log_info "Configuration:"
    log_info "  - Backend tests: $RUN_BACKEND"
    log_info "  - Frontend tests: $RUN_FRONTEND"
    log_info "  - Coverage: $COVERAGE"
    log_info "  - Verbose: $VERBOSE"
    log_info "  - Parallel: $PARALLEL"
    log_info "  - Timeout: $TIMEOUT"
    echo ""
    
    # 환경 및 의존성 확인
    check_environment
    check_go_dependencies
    check_frontend_dependencies
    
    # 테스트 실행
    local backend_result=0
    local frontend_result=0
    
    if [[ "$RUN_BACKEND" == "true" ]]; then
        run_backend_tests || backend_result=$?
    fi
    
    if [[ "$RUN_FRONTEND" == "true" ]]; then
        run_frontend_tests || frontend_result=$?
    fi
    
    # 결과 확인
    if [[ $backend_result -eq 0 ]] && [[ $frontend_result -eq 0 ]]; then
        generate_summary
        log_success "All E2E authentication tests passed!"
        exit 0
    else
        log_error "Some tests failed"
        if [[ $backend_result -ne 0 ]]; then
            log_error "  - Backend tests failed"
        fi
        if [[ $frontend_result -ne 0 ]]; then
            log_error "  - Frontend tests failed"
        fi
        exit 1
    fi
}

# 스크립트 실행
main "$@"