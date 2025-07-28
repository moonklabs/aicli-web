#!/bin/bash
# AICode Manager 테스트 서버 종료 스크립트

set -e

echo "🛑 AICode Manager 테스트 서버 종료..."

# 색상 정의
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PID_FILE=".test-server.pid"

# PID 파일 확인
if [ ! -f "$PID_FILE" ]; then
    echo -e "${YELLOW}⚠️  실행 중인 테스트 서버가 없습니다.${NC}"
    
    # Docker 컨테이너 확인
    if docker-compose ps 2>/dev/null | grep -q "Up"; then
        echo -e "${BLUE}🐳 Docker 컨테이너가 실행 중입니다.${NC}"
        read -p "Docker 컨테이너도 종료하시겠습니까? (y/N) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            docker-compose down
            echo -e "${GREEN}✅ Docker 컨테이너 종료 완료${NC}"
        fi
    fi
    exit 0
fi

# PID 파일에서 프로세스 종료
echo "프로세스 종료 중..."
while read pid; do
    if kill -0 "$pid" 2>/dev/null; then
        echo "  - PID $pid 종료"
        kill "$pid"
    fi
done < "$PID_FILE"

# PID 파일 삭제
rm -f "$PID_FILE"

echo -e "${GREEN}✅ 테스트 서버 종료 완료${NC}"