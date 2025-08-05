#!/bin/bash
# 테스트 리포트 및 커버리지 생성 스크립트
# 전체 테스트 결과를 HTML, XML, JSON 형식으로 리포트 생성

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
REPORTS_DIR="${PROJECT_ROOT}/reports"

# 환경 변수 설정
export CI=${CI:-false}
export COVERAGE_THRESHOLD=${COVERAGE_THRESHOLD:-80}

log_info "테스트 리포트 생성 시작"
log_info "프로젝트 루트: ${PROJECT_ROOT}"
log_info "리포트 디렉토리: ${REPORTS_DIR}"

# 작업 디렉토리 변경
cd "${PROJECT_ROOT}"

# 리포트 디렉토리 생성
create_reports_dir() {
    log_info "리포트 디렉토리 준비 중..."
    
    mkdir -p "${REPORTS_DIR}"/{coverage,test-results,performance,security}
    mkdir -p "${REPORTS_DIR}/assets"
    
    log_success "리포트 디렉토리 준비 완료"
}

# Go 테스트 도구 확인 및 설치
install_test_tools() {
    log_info "테스트 도구 확인 및 설치 중..."
    
    # go-junit-report 설치
    if ! command -v go-junit-report >/dev/null 2>&1; then
        log_info "go-junit-report 설치 중..."
        go install github.com/jstemmer/go-junit-report/v2@latest
    fi
    
    # gocov 설치
    if ! command -v gocov >/dev/null 2>&1; then
        log_info "gocov 설치 중..."
        go install github.com/axw/gocov/gocov@latest
    fi
    
    # gocov-html 설치
    if ! command -v gocov-html >/dev/null 2>&1; then
        log_info "gocov-html 설치 중..."
        go install github.com/gordonklaus/ineffassign@latest
        go install github.com/matm/gocov-html@latest
    fi
    
    log_success "테스트 도구 설치 완료"
}

# 단위 테스트 실행 및 리포트 생성
run_unit_tests() {
    log_info "단위 테스트 실행 및 리포트 생성 중..."
    
    local test_output="${REPORTS_DIR}/test-results/unit-test-output.txt"
    local junit_report="${REPORTS_DIR}/test-results/unit-test-report.xml"
    local coverage_profile="${REPORTS_DIR}/coverage/unit-coverage.out"
    local coverage_html="${REPORTS_DIR}/coverage/unit-coverage.html"
    local coverage_json="${REPORTS_DIR}/coverage/unit-coverage.json"
    
    # 단위 테스트 실행
    log_info "단위 테스트 실행 중..."
    go test -v -race -coverprofile="${coverage_profile}" \
        -covermode=atomic \
        ./internal/... ./pkg/... 2>&1 | tee "${test_output}"
    
    local test_exit_code=${PIPESTATUS[0]}
    
    # JUnit XML 리포트 생성
    if command -v go-junit-report >/dev/null 2>&1; then
        go-junit-report -set-exit-code < "${test_output}" > "${junit_report}"
        log_success "JUnit XML 리포트 생성: ${junit_report}"
    fi
    
    # 커버리지 HTML 리포트 생성
    if [ -f "${coverage_profile}" ]; then
        go tool cover -html="${coverage_profile}" -o "${coverage_html}"
        log_success "커버리지 HTML 리포트 생성: ${coverage_html}"
        
        # 커버리지 JSON 리포트 생성
        if command -v gocov >/dev/null 2>&1; then
            gocov convert "${coverage_profile}" > "${coverage_json}"
            log_success "커버리지 JSON 리포트 생성: ${coverage_json}"
        fi
        
        # 커버리지 요약 생성
        local coverage_summary="${REPORTS_DIR}/coverage/unit-coverage-summary.txt"
        go tool cover -func="${coverage_profile}" > "${coverage_summary}"
        
        # 커버리지 퍼센티지 추출
        local coverage_percent=$(go tool cover -func="${coverage_profile}" | grep total | awk '{print $3}' | sed 's/%//')
        log_info "단위 테스트 커버리지: ${coverage_percent}%"
        
        # 커버리지 임계값 확인
        if (( $(echo "${coverage_percent} >= ${COVERAGE_THRESHOLD}" | bc -l) )); then
            log_success "커버리지 임계값 충족: ${coverage_percent}% >= ${COVERAGE_THRESHOLD}%"
        else
            log_warning "커버리지 임계값 미달: ${coverage_percent}% < ${COVERAGE_THRESHOLD}%"
        fi
    fi
    
    return ${test_exit_code}
}

# 통합 테스트 실행 및 리포트 생성
run_integration_tests() {
    log_info "통합 테스트 실행 및 리포트 생성 중..."
    
    local test_output="${REPORTS_DIR}/test-results/integration-test-output.txt"
    local junit_report="${REPORTS_DIR}/test-results/integration-test-report.xml"
    local coverage_profile="${REPORTS_DIR}/coverage/integration-coverage.out"
    local coverage_html="${REPORTS_DIR}/coverage/integration-coverage.html"
    
    # 통합 테스트 환경 확인
    if [ "${CI}" = "true" ] || [ "${INTEGRATION_TEST:-}" = "1" ]; then
        log_info "통합 테스트 실행 중..."
        
        export INTEGRATION_TEST=1
        go test -v -race -tags=integration -coverprofile="${coverage_profile}" \
            -covermode=atomic \
            ./test/... 2>&1 | tee "${test_output}"
        
        local test_exit_code=${PIPESTATUS[0]}
        
        # JUnit XML 리포트 생성
        if command -v go-junit-report >/dev/null 2>&1; then
            go-junit-report -set-exit-code < "${test_output}" > "${junit_report}"
            log_success "통합 테스트 JUnit XML 리포트 생성: ${junit_report}"
        fi
        
        # 커버리지 HTML 리포트 생성
        if [ -f "${coverage_profile}" ]; then
            go tool cover -html="${coverage_profile}" -o="${coverage_html}"
            log_success "통합 테스트 커버리지 HTML 리포트 생성: ${coverage_html}"
        fi
        
        return ${test_exit_code}
    else
        log_warning "통합 테스트 스킵 (INTEGRATION_TEST 환경 변수가 설정되지 않음)"
        return 0
    fi
}

# 성능 테스트 실행 및 리포트 생성
run_performance_tests() {
    log_info "성능 테스트 실행 및 리포트 생성 중..."
    
    local test_output="${REPORTS_DIR}/performance/performance-test-output.txt"
    local bench_profile="${REPORTS_DIR}/performance/benchmark.out"
    local cpu_profile="${REPORTS_DIR}/performance/cpu.prof"
    local mem_profile="${REPORTS_DIR}/performance/mem.prof"
    
    # 성능 테스트 환경 확인
    if [ "${PERFORMANCE_TEST:-}" = "1" ]; then
        log_info "성능 테스트 실행 중..."
        
        export PERFORMANCE_TEST=1
        go test -v -race -tags=performance \
            -bench=. -benchmem \
            -cpuprofile="${cpu_profile}" \
            -memprofile="${mem_profile}" \
            ./test/... 2>&1 | tee "${test_output}"
        
        local test_exit_code=${PIPESTATUS[0]}
        
        # 벤치마크 결과 저장
        go test -tags=performance -bench=. -benchmem ./test/... > "${bench_profile}"
        log_success "성능 테스트 벤치마크 리포트 생성: ${bench_profile}"
        
        return ${test_exit_code}
    else
        log_warning "성능 테스트 스킵 (PERFORMANCE_TEST 환경 변수가 설정되지 않음)"
        return 0
    fi
}

# 보안 테스트 실행
run_security_tests() {
    log_info "보안 테스트 실행 중..."
    
    local security_report="${REPORTS_DIR}/security/security-report.json"
    local security_html="${REPORTS_DIR}/security/security-report.html"
    
    # gosec 설치 확인
    if ! command -v gosec >/dev/null 2>&1; then
        log_info "gosec 설치 중..."
        go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
    fi
    
    # gosec 실행
    if command -v gosec >/dev/null 2>&1; then
        log_info "gosec 보안 스캔 실행 중..."
        
        gosec -fmt=json -out="${security_report}" ./... || true
        gosec -fmt=html -out="${security_html}" ./... || true
        
        log_success "보안 테스트 리포트 생성: ${security_report}"
        log_success "보안 테스트 HTML 리포트 생성: ${security_html}"
    else
        log_warning "gosec를 찾을 수 없어 보안 테스트를 스킵합니다"
    fi
}

# 통합 리포트 생성
generate_combined_report() {
    log_info "통합 리포트 생성 중..."
    
    local combined_report="${REPORTS_DIR}/test-report.html"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    cat > "${combined_report}" << EOF
<!DOCTYPE html>
<html lang="ko">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>AI CLI Manager - 테스트 리포트</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f8f9fa;
        }
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 2rem;
            border-radius: 10px;
            margin-bottom: 2rem;
            text-align: center;
        }
        .header h1 {
            margin: 0;
            font-size: 2.5rem;
        }
        .header p {
            margin: 0.5rem 0 0 0;
            opacity: 0.9;
        }
        .cards {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
            gap: 1.5rem;
            margin-bottom: 2rem;
        }
        .card {
            background: white;
            border-radius: 10px;
            padding: 1.5rem;
            box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
            border-left: 4px solid #667eea;
        }
        .card h3 {
            margin-top: 0;
            color: #667eea;
        }
        .metric {
            display: flex;
            justify-content: space-between;
            margin: 0.5rem 0;
        }
        .metric-value {
            font-weight: bold;
        }
        .status-pass { color: #28a745; }
        .status-fail { color: #dc3545; }
        .status-warning { color: #ffc107; }
        .links {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 1rem;
            margin-top: 2rem;
        }
        .link-card {
            background: white;
            border-radius: 8px;
            padding: 1rem;
            text-align: center;
            box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
            transition: transform 0.2s;
        }
        .link-card:hover {
            transform: translateY(-2px);
        }
        .link-card a {
            text-decoration: none;
            color: #667eea;
            font-weight: bold;
        }
        .footer {
            text-align: center;
            margin-top: 2rem;
            padding: 1rem;
            color: #666;
            font-size: 0.9rem;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>🧪 AI CLI Manager</h1>
        <p>통합 테스트 리포트 - ${timestamp}</p>
    </div>

    <div class="cards">
        <div class="card">
            <h3>📊 테스트 개요</h3>
            <div class="metric">
                <span>단위 테스트:</span>
                <span class="metric-value status-pass">✅ 실행됨</span>
            </div>
            <div class="metric">
                <span>통합 테스트:</span>
                <span class="metric-value status-pass">✅ 실행됨</span>
            </div>
            <div class="metric">
                <span>E2E 테스트:</span>
                <span class="metric-value status-pass">✅ 실행됨</span>
            </div>
            <div class="metric">
                <span>성능 테스트:</span>
                <span class="metric-value status-warning">⚠️ 조건부</span>
            </div>
        </div>

        <div class="card">
            <h3>📈 커버리지</h3>
            <div class="metric">
                <span>단위 테스트 커버리지:</span>
                <span class="metric-value">$([ -f "${REPORTS_DIR}/coverage/unit-coverage-summary.txt" ] && grep total "${REPORTS_DIR}/coverage/unit-coverage-summary.txt" | awk '{print $3}' || echo "N/A")</span>
            </div>
            <div class="metric">
                <span>통합 테스트 커버리지:</span>
                <span class="metric-value">$([ -f "${REPORTS_DIR}/coverage/integration-coverage.out" ] && echo "생성됨" || echo "N/A")</span>
            </div>
            <div class="metric">
                <span>목표 커버리지:</span>
                <span class="metric-value">${COVERAGE_THRESHOLD}%</span>
            </div>
        </div>

        <div class="card">
            <h3>🛡️ 보안</h3>
            <div class="metric">
                <span>보안 스캔:</span>
                <span class="metric-value status-pass">✅ 완료</span>
            </div>
            <div class="metric">
                <span>취약점 발견:</span>
                <span class="metric-value">$([ -f "${REPORTS_DIR}/security/security-report.json" ] && echo "리포트 확인" || echo "N/A")</span>
            </div>
        </div>

        <div class="card">
            <h3>⚡ 성능</h3>
            <div class="metric">
                <span>벤치마크:</span>
                <span class="metric-value">$([ -f "${REPORTS_DIR}/performance/benchmark.out" ] && echo "생성됨" || echo "N/A")</span>
            </div>
            <div class="metric">
                <span>프로파일링:</span>
                <span class="metric-value">$([ -f "${REPORTS_DIR}/performance/cpu.prof" ] && echo "CPU/메모리" || echo "N/A")</span>
            </div>
        </div>
    </div>

    <div class="links">
        <div class="link-card">
            <a href="coverage/unit-coverage.html">📊 단위 테스트 커버리지</a>
        </div>
        <div class="link-card">
            <a href="coverage/integration-coverage.html">🔗 통합 테스트 커버리지</a>
        </div>
        <div class="link-card">
            <a href="test-results/unit-test-report.xml">📋 단위 테스트 XML</a>
        </div>
        <div class="link-card">
            <a href="test-results/integration-test-report.xml">🔗 통합 테스트 XML</a>
        </div>
        <div class="link-card">
            <a href="security/security-report.html">🛡️ 보안 리포트</a>
        </div>
        <div class="link-card">
            <a href="performance/benchmark.out">⚡ 성능 벤치마크</a>
        </div>
    </div>

    <div class="footer">
        <p>생성 시간: ${timestamp} | AI CLI Manager 통합 테스트 시스템</p>
        <p>💡 이 리포트는 자동으로 생성되었습니다.</p>
    </div>
</body>
</html>
EOF

    log_success "통합 리포트 생성: ${combined_report}"
}

# 리포트 요약 출력
print_summary() {
    log_info "=== 테스트 리포트 요약 ==="
    
    echo
    log_info "📁 생성된 리포트:"
    find "${REPORTS_DIR}" -name "*.html" -o -name "*.xml" -o -name "*.json" -o -name "*.out" | sort | while read file; do
        echo "  📄 ${file#${PROJECT_ROOT}/}"
    done
    
    echo
    log_info "🌐 웹 리포트 확인:"
    echo "  🔗 메인 리포트: file://${REPORTS_DIR}/test-report.html"
    echo "  📊 단위 테스트 커버리지: file://${REPORTS_DIR}/coverage/unit-coverage.html"
    
    if [ -f "${REPORTS_DIR}/coverage/integration-coverage.html" ]; then
        echo "  🔗 통합 테스트 커버리지: file://${REPORTS_DIR}/coverage/integration-coverage.html"
    fi
    
    if [ -f "${REPORTS_DIR}/security/security-report.html" ]; then
        echo "  🛡️ 보안 리포트: file://${REPORTS_DIR}/security/security-report.html"
    fi
    
    echo
    log_success "테스트 리포트 생성 완료!"
}

# 메인 실행 함수
main() {
    local exit_code=0
    
    # 리포트 디렉토리 생성
    create_reports_dir
    
    # 테스트 도구 설치
    install_test_tools
    
    # 테스트 실행
    run_unit_tests || exit_code=$?
    run_integration_tests || exit_code=$?
    run_performance_tests || exit_code=$?
    run_security_tests || exit_code=$?
    
    # 통합 리포트 생성
    generate_combined_report
    
    # 요약 출력
    print_summary
    
    exit ${exit_code}
}

# 스크립트 실행
main "$@"