#!/bin/bash

# AICode Manager 문서 빌드 스크립트
set -euo pipefail

# 색상 정의
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 로깅 함수
log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

warn() {
    echo -e "${YELLOW}[WARNING] $1${NC}"
}

error() {
    echo -e "${RED}[ERROR] $1${NC}"
}

success() {
    echo -e "${GREEN}[SUCCESS] $1${NC}"
}

# 도움말 표시
show_help() {
    cat << EOF
AICode Manager 문서 빌드 스크립트

사용법: $0 [옵션]

옵션:
    -h, --help          이 도움말 표시
    -c, --clean         빌드 전 기존 파일 정리
    -s, --serve         빌드 후 로컬 서버 시작
    -p, --port PORT     서버 포트 (기본값: 8000)
    -v, --verbose       상세 출력
    -w, --watch         파일 변경 감지 모드
    --production        프로덕션 빌드 (최적화 적용)
    --check-links       링크 검사 실행
    --no-git            Git 정보 없이 빌드

예시:
    $0                  기본 빌드
    $0 -cs              정리 후 빌드하고 서버 시작
    $0 --production     프로덕션 빌드
    $0 -w               개발 모드 (파일 감지)

EOF
}

# 기본 설정
CLEAN_BUILD=false
SERVE_DOCS=false
PORT=8000
VERBOSE=false
WATCH_MODE=false
PRODUCTION=false
CHECK_LINKS=false
USE_GIT=true

# 명령행 인자 파싱
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -c|--clean)
            CLEAN_BUILD=true
            shift
            ;;
        -s|--serve)
            SERVE_DOCS=true
            shift
            ;;
        -p|--port)
            PORT="$2"
            shift 2
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -w|--watch)
            WATCH_MODE=true
            SERVE_DOCS=true
            shift
            ;;
        --production)
            PRODUCTION=true
            shift
            ;;
        --check-links)
            CHECK_LINKS=true
            shift
            ;;
        --no-git)
            USE_GIT=false
            shift
            ;;
        *)
            error "알 수 없는 옵션: $1"
            show_help
            exit 1
            ;;
    esac
done

# 프로젝트 루트 디렉토리 확인
if [[ ! -f "mkdocs.yml" ]]; then
    error "mkdocs.yml 파일을 찾을 수 없습니다. 프로젝트 루트에서 실행하세요."
    exit 1
fi

# Python 및 pip 확인
check_dependencies() {
    log "의존성 확인 중..."
    
    if ! command -v python3 &> /dev/null; then
        error "Python 3이 설치되어 있지 않습니다."
        exit 1
    fi
    
    if ! command -v pip &> /dev/null; then
        error "pip가 설치되어 있지 않습니다."
        exit 1
    fi
    
    # 가상 환경 확인 (권장)
    if [[ -z "${VIRTUAL_ENV:-}" ]]; then
        warn "가상 환경을 사용하지 않고 있습니다. 가상 환경 사용을 권장합니다."
    fi
}

# MkDocs 의존성 설치
install_dependencies() {
    log "MkDocs 의존성 설치 중..."
    
    if [[ -f "requirements-docs.txt" ]]; then
        pip install -r requirements-docs.txt
    else
        warn "requirements-docs.txt 파일을 찾을 수 없습니다. 기본 MkDocs만 설치합니다."
        pip install mkdocs mkdocs-material
    fi
}

# 기존 빌드 파일 정리
clean_build() {
    if [[ "$CLEAN_BUILD" == true ]]; then
        log "기존 빌드 파일 정리 중..."
        rm -rf site/
        success "정리 완료"
    fi
}

# 설정 검증
validate_config() {
    log "MkDocs 설정 검증 중..."
    
    if ! mkdocs config-check; then
        error "MkDocs 설정에 오류가 있습니다."
        exit 1
    fi
    
    success "설정 검증 완료"
}

# 문서 구조 확인
check_docs_structure() {
    log "문서 구조 확인 중..."
    
    # 필수 파일 확인
    required_files=(
        "docs/index.md"
        "mkdocs.yml"
    )
    
    for file in "${required_files[@]}"; do
        if [[ ! -f "$file" ]]; then
            error "필수 파일이 없습니다: $file"
            exit 1
        fi
    done
    
    # 문서 파일 수 확인
    doc_count=$(find docs/ -name "*.md" | wc -l)
    log "발견된 마크다운 파일: ${doc_count}개"
    
    success "문서 구조 확인 완료"
}

# 프로덕션 최적화 설정
setup_production() {
    if [[ "$PRODUCTION" == true ]]; then
        log "프로덕션 빌드 설정 적용 중..."
        
        export MKDOCS_ENV="production"
        export ENABLE_MINIFY="true"
        export ENABLE_SEARCH_INDEX="true"
        
        success "프로덕션 설정 적용 완료"
    fi
}

# 문서 빌드
build_docs() {
    log "문서 빌드 시작..."
    
    local build_args="--strict"
    
    if [[ "$VERBOSE" == true ]]; then
        build_args="$build_args --verbose"
    fi
    
    if [[ "$PRODUCTION" == true ]]; then
        log "프로덕션 모드로 빌드 중..."
    fi
    
    if ! mkdocs build $build_args; then
        error "문서 빌드에 실패했습니다."
        exit 1
    fi
    
    success "문서 빌드 완료"
}

# 빌드 결과 확인
verify_build() {
    log "빌드 결과 확인 중..."
    
    if [[ ! -d "site" ]]; then
        error "빌드 출력 디렉토리가 생성되지 않았습니다."
        exit 1
    fi
    
    if [[ ! -f "site/index.html" ]]; then
        error "메인 페이지가 생성되지 않았습니다."
        exit 1
    fi
    
    # 파일 수 및 크기 정보
    local file_count=$(find site/ -type f | wc -l)
    local total_size=$(du -sh site/ | cut -f1)
    
    log "생성된 파일: ${file_count}개"
    log "총 크기: ${total_size}"
    
    success "빌드 결과 확인 완료"
}

# 링크 검사
check_links() {
    if [[ "$CHECK_LINKS" == true ]]; then
        log "링크 검사 시작..."
        
        # htmlproof 설치 확인
        if command -v htmlproofer &> /dev/null; then
            htmlproof site/ --check-html --check-internal-hash --disable-external
        else
            warn "htmlproofer가 설치되어 있지 않습니다. 링크 검사를 건너뜁니다."
        fi
        
        success "링크 검사 완료"
    fi
}

# 개발 서버 시작
serve_docs() {
    if [[ "$SERVE_DOCS" == true ]]; then
        local serve_args="--dev-addr=0.0.0.0:$PORT"
        
        if [[ "$WATCH_MODE" == true ]]; then
            log "개발 모드로 서버 시작 중... (포트: $PORT)"
            log "파일 변경 감지 활성화"
            serve_args="$serve_args --livereload"
        else
            log "문서 서버 시작 중... (포트: $PORT)"
        fi
        
        if [[ "$VERBOSE" == true ]]; then
            serve_args="$serve_args --verbose"
        fi
        
        success "서버 시작됨: http://localhost:$PORT"
        log "서버를 중지하려면 Ctrl+C를 누르세요"
        
        # 브라우저 자동 열기 (macOS/Linux)
        if command -v open &> /dev/null; then
            open "http://localhost:$PORT"
        elif command -v xdg-open &> /dev/null; then
            xdg-open "http://localhost:$PORT"
        fi
        
        mkdocs serve $serve_args
    fi
}

# 빌드 정보 생성
generate_build_info() {
    log "빌드 정보 생성 중..."
    
    local build_info_file="site/build-info.json"
    local build_date=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    local git_commit=""
    local git_branch=""
    
    if [[ "$USE_GIT" == true ]] && command -v git &> /dev/null && git rev-parse --git-dir > /dev/null 2>&1; then
        git_commit=$(git rev-parse HEAD)
        git_branch=$(git rev-parse --abbrev-ref HEAD)
    fi
    
    cat > "$build_info_file" << EOF
{
    "build_date": "$build_date",
    "git_commit": "$git_commit",
    "git_branch": "$git_branch",
    "mkdocs_version": "$(mkdocs --version | cut -d' ' -f3)",
    "python_version": "$(python3 --version | cut -d' ' -f2)",
    "production_build": $PRODUCTION
}
EOF
    
    success "빌드 정보 생성 완료"
}

# 메인 실행 함수
main() {
    log "AICode Manager 문서 빌드 시작"
    
    check_dependencies
    install_dependencies
    clean_build
    validate_config
    check_docs_structure
    setup_production
    build_docs
    verify_build
    generate_build_info
    check_links
    
    success "문서 빌드 프로세스 완료"
    
    serve_docs
}

# 에러 발생 시 정리
cleanup() {
    error "빌드 중 오류가 발생했습니다."
    exit 1
}

# 트랩 설정
trap cleanup ERR

# 메인 함수 실행
main "$@"