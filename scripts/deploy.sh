#!/bin/bash
# ==========================================
# AICLI-Web 프로덕션 배포 스크립트
# ==========================================

set -e  # 에러 발생 시 중단
set -u  # 미정의 변수 사용 시 에러

# 색상 정의
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 함수: 로그 출력
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

# 함수: 사전 요구사항 확인
check_requirements() {
    log_info "사전 요구사항 확인 중..."
    
    # Docker 확인
    if ! command -v docker &> /dev/null; then
        log_error "Docker가 설치되어 있지 않습니다."
        exit 1
    fi
    
    # Docker Compose 확인
    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose가 설치되어 있지 않습니다."
        exit 1
    fi
    
    # Git 확인
    if ! command -v git &> /dev/null; then
        log_warning "Git이 설치되어 있지 않습니다. 버전 정보를 가져올 수 없습니다."
    fi
    
    log_success "모든 요구사항이 충족되었습니다."
}

# 함수: 환경 설정
setup_environment() {
    log_info "환경 설정 중..."
    
    # .env 파일 확인
    if [ ! -f .env ]; then
        if [ -f .env.example ]; then
            log_warning ".env 파일이 없습니다. .env.example에서 생성합니다..."
            cp .env.example .env
            log_warning "⚠️  .env 파일을 편집하여 필수 설정을 입력하세요!"
            read -p "계속하시겠습니까? (y/n): " -n 1 -r
            echo
            if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                exit 1
            fi
        else
            log_error ".env 파일이 없습니다."
            exit 1
        fi
    fi
    
    # 필수 환경 변수 확인
    source .env
    
    if [ -z "${CLAUDE_API_KEY:-}" ]; then
        log_error "CLAUDE_API_KEY가 설정되지 않았습니다."
        exit 1
    fi
    
    # 워크스페이스 디렉토리 생성
    WORKSPACE_PATH=${WORKSPACE_HOST_PATH:-./workspace}
    if [ ! -d "$WORKSPACE_PATH" ]; then
        log_info "워크스페이스 디렉토리 생성: $WORKSPACE_PATH"
        mkdir -p "$WORKSPACE_PATH"
    fi
    
    # 데이터 디렉토리 생성
    if [ ! -d ./data ]; then
        log_info "데이터 디렉토리 생성..."
        mkdir -p ./data
    fi
    
    log_success "환경 설정 완료"
}

# 함수: Docker 이미지 빌드
build_images() {
    log_info "Docker 이미지 빌드 중..."
    
    # 빌드 인자 생성
    VERSION=${VERSION:-0.1.0}
    BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
    GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    
    # Docker Compose 빌드
    docker-compose -f docker-compose.prod.yml build \
        --build-arg VERSION=$VERSION \
        --build-arg BUILD_TIME=$BUILD_TIME \
        --build-arg GIT_COMMIT=$GIT_COMMIT
    
    log_success "Docker 이미지 빌드 완료"
}

# 함수: 서비스 시작
start_services() {
    log_info "서비스 시작 중..."
    
    # 기존 컨테이너 정리
    docker-compose -f docker-compose.prod.yml down
    
    # 서비스 시작
    docker-compose -f docker-compose.prod.yml up -d
    
    log_info "서비스 상태 확인 중..."
    sleep 5
    
    docker-compose -f docker-compose.prod.yml ps
    
    log_success "서비스 시작 완료"
}

# 함수: 헬스체크
health_check() {
    log_info "헬스체크 수행 중..."
    
    local max_retries=30
    local retry_count=0
    
    while [ $retry_count -lt $max_retries ]; do
        if curl -f -s http://localhost:${API_PORT:-8080}/health > /dev/null; then
            log_success "API 서버가 정상적으로 작동 중입니다."
            break
        fi
        
        retry_count=$((retry_count + 1))
        log_info "API 서버 시작 대기 중... ($retry_count/$max_retries)"
        sleep 2
    done
    
    if [ $retry_count -eq $max_retries ]; then
        log_error "API 서버가 시작되지 않았습니다."
        docker-compose -f docker-compose.prod.yml logs api
        exit 1
    fi
    
    log_success "헬스체크 완료"
}

# 함수: 배포 정보 출력
print_info() {
    echo
    echo "=========================================="
    echo "🎉 AICLI-Web 배포 완료!"
    echo "=========================================="
    echo
    echo "📌 접속 정보:"
    echo "  - Web UI: http://localhost:${WEB_PORT:-3000}"
    echo "  - API: http://localhost:${API_PORT:-8080}"
    echo "  - API Health: http://localhost:${API_PORT:-8080}/health"
    echo
    echo "📝 유용한 명령어:"
    echo "  - 로그 확인: docker-compose -f docker-compose.prod.yml logs -f"
    echo "  - 서비스 중지: docker-compose -f docker-compose.prod.yml down"
    echo "  - 서비스 재시작: docker-compose -f docker-compose.prod.yml restart"
    echo "  - 상태 확인: docker-compose -f docker-compose.prod.yml ps"
    echo
    echo "🔒 보안 권장사항:"
    echo "  1. 프로덕션 환경에서는 HTTPS를 설정하세요"
    echo "  2. 강력한 JWT_SECRET과 SESSION_SECRET을 사용하세요"
    echo "  3. 정기적으로 백업을 수행하세요"
    echo
    echo "=========================================="
}

# 함수: 클린업
cleanup() {
    log_info "클린업 수행 중..."
    
    # 미사용 이미지 제거
    docker image prune -f
    
    # 미사용 볼륨 제거 (주의: 데이터 손실 가능)
    # docker volume prune -f
    
    log_success "클린업 완료"
}

# 메인 실행
main() {
    echo "=========================================="
    echo "🚀 AICLI-Web 프로덕션 배포 시작"
    echo "=========================================="
    echo
    
    # 스크립트 실행 위치 확인
    if [ ! -f "docker-compose.prod.yml" ]; then
        log_error "프로젝트 루트 디렉토리에서 실행해주세요."
        exit 1
    fi
    
    # 단계별 실행
    check_requirements
    setup_environment
    build_images
    start_services
    health_check
    cleanup
    print_info
}

# 트랩 설정 (에러 발생 시 정리)
trap 'log_error "배포 중 오류가 발생했습니다."; exit 1' ERR

# 메인 함수 실행
main "$@"