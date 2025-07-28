#!/bin/bash
# 웹 프론트엔드 시작 스크립트

echo "🌐 웹 프론트엔드 시작..."

cd web

# 의존성 확인
if [ ! -d "node_modules" ]; then
    echo "📦 의존성 설치 중..."
    pnpm install
fi

echo "🚀 개발 서버 시작 (포트: 5173)..."
echo ""
echo "🔗 접속 URL:"
echo "   - 웹 UI: http://localhost:5173"
echo ""

# 개발 서버 실행
pnpm dev