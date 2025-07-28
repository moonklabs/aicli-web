#\!/bin/bash
# 테스트 서버 시작 스크립트

echo "🚀 AICode Manager 테스트 서버 시작..."

# 환경변수 설정
export DISABLE_AUTH=true
export LOG_LEVEL=debug
export API_PORT=8080

# .env.test 파일 로드
if [ -f .env.test ]; then
    export $(cat .env.test | grep -v '^#' | xargs)
fi

# 데이터 디렉토리 생성
mkdir -p ./data

echo "📡 API 서버 시작 (포트: $API_PORT)..."
./build/aicli-api &
API_PID=$\!

echo "✅ API 서버 PID: $API_PID"
echo ""
echo "🔗 접속 URL:"
echo "   - API: http://localhost:8080"
echo "   - Health Check: http://localhost:8080/health"
echo ""
echo "종료하려면 Ctrl+C를 누르세요..."

# 종료 시그널 처리
trap "echo '🛑 서버 종료 중...'; kill $API_PID 2>/dev/null; exit" INT TERM

# 서버가 실행 중인 동안 대기
wait $API_PID
EOF < /dev/null