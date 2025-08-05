#!/bin/bash
# 통합 테스트 실행 스크립트
# 전체 통합 테스트 스위트를 Docker Compose 환경에서 실행

set -euo pipefail

# 색상 정의
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 로그 함수들
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

# 스크립트 디렉토리 확인
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# 환경 변수 설정
export INTEGRATION_TEST=1
export CI=${CI:-false}
export TEST_TIMEOUT=${TEST_TIMEOUT:-30m}
export COMPOSE_PROJECT_NAME=${COMPOSE_PROJECT_NAME:-aicli-integration-test}

log_info "통합 테스트 실행 시작"
log_info "프로젝트 루트: ${PROJECT_ROOT}"
log_info "테스트 타임아웃: ${TEST_TIMEOUT}"

# 작업 디렉토리 변경
cd "${PROJECT_ROOT}"

# Docker와 Docker Compose 확인
check_docker() {
    log_info "Docker 환경 확인 중..."
    
    if ! command -v docker >/dev/null 2>&1; then
        log_error "Docker가 설치되지 않았습니다"
        exit 1
    fi
    
    if ! docker info >/dev/null 2>&1; then
        log_error "Docker 데몬이 실행되지 않았습니다"
        exit 1
    fi
    
    if ! command -v docker-compose >/dev/null 2>&1; then
        log_error "Docker Compose가 설치되지 않았습니다"
        exit 1
    fi
    
    log_success "Docker 환경 확인 완료"
}

# 테스트 환경 정리
cleanup() {
    log_info "테스트 환경 정리 중..."
    
    # Docker Compose 서비스 중지 및 제거
    docker-compose -f docker-compose.test.yml -p "${COMPOSE_PROJECT_NAME}" down -v --remove-orphans || true
    
    # 테스트 컨테이너 정리
    docker ps -a --filter "label=test.type=integration" --format "{{.ID}}" | xargs -r docker rm -f || true
    
    # 테스트 네트워크 정리
    docker network ls --filter "label=test.type=integration" --format "{{.ID}}" | xargs -r docker network rm || true
    
    # 테스트 볼륨 정리
    docker volume ls --filter "label=test.type=integration" --format "{{.Name}}" | xargs -r docker volume rm || true
    
    log_success "테스트 환경 정리 완료"
}

# 신호 핸들러 등록
trap cleanup EXIT INT TERM

# 빌드 확인
build_binaries() {
    log_info "바이너리 빌드 중..."
    
    if ! make build; then
        log_error "빌드 실패"
        exit 1
    fi
    
    log_success "바이너리 빌드 완료"
}

# 테스트 환경 준비
prepare_test_env() {
    log_info "테스트 환경 준비 중..."
    
    # 테스트 디렉토리 생성
    mkdir -p test/fixtures/git
    mkdir -p test/fixtures/db
    mkdir -p test/config
    mkdir -p reports
    
    # 필요한 이미지 확인 및 풀
    log_info "필요한 Docker 이미지 확인 중..."
    
    images=(
        "postgres:15-alpine"
        "redis:7-alpine"
        "docker:24-dind"
        "gitea/gitea:1.21"
        "prom/prometheus:latest"
        "alpine:latest"
    )
    
    for image in "${images[@]}"; do
        if ! docker image inspect "${image}" >/dev/null 2>&1; then
            log_info "이미지 풀링: ${image}"
            docker pull "${image}"
        fi
    done
    
    log_success "테스트 환경 준비 완료"
}

# 테스트 실행 함수들
run_unit_tests() {
    log_info "단위 테스트 실행 중..."
    
    if go test -v -race -cover -timeout="${TEST_TIMEOUT}" ./internal/... ./pkg/...; then
        log_success "단위 테스트 통과"
        return 0
    else
        log_error "단위 테스트 실패"
        return 1
    fi
}

run_integration_tests() {
    log_info "통합 테스트 환경 시작 중..."
    
    # Docker Compose로 테스트 환경 시작
    if ! docker-compose -f docker-compose.test.yml -p "${COMPOSE_PROJECT_NAME}" up -d test-db test-redis test-docker; then
        log_error "테스트 환경 시작 실패"
        return 1
    fi
    
    # 서비스들이 준비될 때까지 대기
    log_info "테스트 서비스 준비 대기 중..."
    sleep 30
    
    # 서비스 상태 확인
    for service in test-db test-redis test-docker; do
        if ! docker-compose -f docker-compose.test.yml -p "${COMPOSE_PROJECT_NAME}" ps "${service}" | grep -q "Up"; then
            log_error "서비스 ${service}가 실행되지 않았습니다"
            docker-compose -f docker-compose.test.yml -p "${COMPOSE_PROJECT_NAME}" logs "${service}"
            return 1
        fi
    done
    
    log_success "테스트 환경 준비 완료"
    
    # 통합 테스트 실행
    log_info "통합 테스트 실행 중..."
    
    # 환경 변수 설정
    export DB_HOST=localhost
    export DB_PORT=5434
    export DB_NAME=aicli_test
    export DB_USER=test_user
    export DB_PASSWORD=test_password
    export REDIS_HOST=localhost
    export REDIS_PORT=6380
    export REDIS_PASSWORD=test_redis_password
    export DOCKER_HOST=tcp://localhost:2376
    
    # 통합 테스트 실행
    if go test -v -race -tags=integration -timeout="${TEST_TIMEOUT}" ./test/...; then
        log_success "통합 테스트 통과"
        return 0
    else
        log_error "통합 테스트 실패"
        return 1
    fi
}

run_e2e_tests() {
    log_info "E2E 테스트 실행 중..."
    
    # E2E 테스트용 전체 환경 시작
    if ! docker-compose -f docker-compose.test.yml -p "${COMPOSE_PROJECT_NAME}" --profile e2e up -d; then
        log_error "E2E 테스트 환경 시작 실패"
        return 1
    fi
    
    # E2E 테스트 실행 대기
    log_info "E2E 테스트 환경 준비 대기 중..."
    sleep 60
    
    # E2E 테스트 실행
    if go test -v -race -tags=e2e -timeout="${TEST_TIMEOUT}" ./internal/claude/e2e_scenarios_test.go; then
        log_success "E2E 테스트 통과"
        return 0
    else
        log_error "E2E 테스트 실패"
        return 1
    fi
}

run_performance_tests() {
    log_info "성능 테스트 실행 중..."
    
    # 성능 테스트용 환경 시작
    if ! docker-compose -f docker-compose.test.yml -p "${COMPOSE_PROJECT_NAME}" --profile performance up -d; then
        log_error "성능 테스트 환경 시작 실패"
        return 1
    fi
    
    # 성능 테스트 실행 대기
    log_info "성능 테스트 환경 준비 대기 중..."
    sleep 30
    
    # 성능 테스트 실행
    export PERFORMANCE_TEST=1
    export MAX_CONCURRENT_AGENTS=${MAX_CONCURRENT_AGENTS:-50}
    export TEST_DURATION=${TEST_DURATION:-3m}
    
    if go test -v -race -tags=performance -timeout="${TEST_TIMEOUT}" ./test/performance_test_suite.go; then
        log_success "성능 테스트 통과"
        return 0
    else
        log_error "성능 테스트 실패"
        return 1
    fi
}

# 테스트 리포트 생성
generate_reports() {
    log_info "테스트 리포트 생성 중..."
    
    # 커버리지 리포트 생성
    if command -v go >/dev/null 2>&1; then
        go test -v -race -coverprofile=reports/coverage.out ./... || true
        go tool cover -html=reports/coverage.out -o reports/coverage.html || true
        log_success "커버리지 리포트 생성: reports/coverage.html"
    fi
    
    # Docker Compose 로그 수집
    docker-compose -f docker-compose.test.yml -p "${COMPOSE_PROJECT_NAME}" logs > reports/docker-compose.log 2>&1 || true
    log_success "Docker Compose 로그 저장: reports/docker-compose.log"
}

# 메인 실행 함수
main() {
    local test_type="${1:-all}"
    local exit_code=0
    
    log_info "통합 테스트 타입: ${test_type}"
    
    # 환경 확인
    check_docker
    
    # 환경 정리
    cleanup
    
    # 빌드
    build_binaries
    
    # 테스트 환경 준비
    prepare_test_env
    
    # 테스트 실행
    case "${test_type}" in
        "unit")
            run_unit_tests || exit_code=$?
            ;;
        "integration")
            run_integration_tests || exit_code=$?
            ;;
        "e2e")
            run_e2e_tests || exit_code=$?
            ;;
        "performance")
            run_performance_tests || exit_code=$?
            ;;
        "all")
            run_unit_tests || exit_code=$?
            run_integration_tests || exit_code=$?
            run_e2e_tests || exit_code=$?
            ;;
        *)
            log_error "지원되지 않는 테스트 타입: ${test_type}"
            log_info "사용법: $0 [unit|integration|e2e|performance|all]"
            exit 1
            ;;
    esac
    
    # 리포트 생성
    generate_reports
    
    # 결과 출력
    if [ ${exit_code} -eq 0 ]; then
        log_success "모든 테스트가 성공적으로 완료되었습니다"
    else
        log_error "일부 테스트가 실패했습니다 (종료 코드: ${exit_code})"
    fi
    
    exit ${exit_code}
}

# 스크립트 실행
main "$@"