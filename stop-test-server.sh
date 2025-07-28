#\!/bin/bash
# 테스트 서버 중지 스크립트

echo "🛑 테스트 서버 중지 중..."

# API 서버 프로세스 찾아서 종료
API_PIDS=$(pgrep -f "aicli-api")
if [ -n "$API_PIDS" ]; then
    echo "API 서버 프로세스 종료: $API_PIDS"
    kill $API_PIDS 2>/dev/null
fi

# 웹 개발 서버 종료
WEB_PIDS=$(pgrep -f "vite")
if [ -n "$WEB_PIDS" ]; then
    echo "웹 개발 서버 종료: $WEB_PIDS"
    kill $WEB_PIDS 2>/dev/null
fi

# Docker 컨테이너 정리
if [ -f "docker-compose.test.yml" ]; then
    echo "Docker 컨테이너 정리..."
    docker-compose -f docker-compose.test.yml down 2>/dev/null
fi

echo "✅ 모든 테스트 서버가 종료되었습니다."
