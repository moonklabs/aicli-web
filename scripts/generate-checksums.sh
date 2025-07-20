#!/bin/bash
# 릴리스 체크섬 생성 스크립트

set -e

# 색상 코드
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 사용법 출력
usage() {
    echo "Usage: $0 <directory>"
    echo "  directory: 바이너리 파일이 있는 디렉토리"
    echo ""
    echo "Example:"
    echo "  $0 ./dist"
    exit 1
}

# 인자 확인
if [ $# -ne 1 ]; then
    usage
fi

DIST_DIR="$1"

# 디렉토리 존재 확인
if [ ! -d "$DIST_DIR" ]; then
    echo -e "${RED}❌ 디렉토리를 찾을 수 없습니다: $DIST_DIR${NC}"
    exit 1
fi

echo -e "${BLUE}🔐 체크섬 생성 시작...${NC}"
echo ""

cd "$DIST_DIR"

# 기존 체크섬 파일 제거
rm -f checksums.txt checksums.sha256 *.sha256

# 바이너리 파일 목록
BINARIES=$(ls aicli-* 2>/dev/null | grep -v ".sha256" | grep -v "checksums" || true)

if [ -z "$BINARIES" ]; then
    echo -e "${RED}❌ 바이너리 파일을 찾을 수 없습니다${NC}"
    exit 1
fi

echo -e "${YELLOW}📋 발견된 바이너리 파일:${NC}"
echo "$BINARIES" | while read -r binary; do
    echo "  - $binary"
done
echo ""

# SHA256 체크섬 생성
echo -e "${BLUE}📝 SHA256 체크섬 생성 중...${NC}"

# 통합 체크섬 파일 생성
echo "# AICode Manager Release Checksums" > checksums.txt
echo "# Generated at: $(date -u '+%Y-%m-%d %H:%M:%S UTC')" >> checksums.txt
echo "" >> checksums.txt

# 각 바이너리에 대한 체크섬 생성
echo "$BINARIES" | while read -r binary; do
    if [ -f "$binary" ]; then
        # 통합 체크섬 파일에 추가
        sha256sum "$binary" >> checksums.txt
        
        # 개별 체크섬 파일 생성
        sha256sum "$binary" > "$binary.sha256"
        
        echo -e "${GREEN}✓${NC} $binary"
    fi
done

echo ""
echo -e "${BLUE}📊 체크섬 요약:${NC}"
echo ""

# 체크섬 파일 내용 표시 (포맷팅)
cat checksums.txt | grep -v "^#" | grep -v "^$" | while read -r checksum filename; do
    # 파일 크기 가져오기
    size=$(ls -lh "$filename" 2>/dev/null | awk '{print $5}')
    
    # 플랫폼 정보 추출
    platform=$(echo "$filename" | sed -E 's/aicli(-api)?-v[0-9.]+-(.*)/\2/')
    
    printf "%-50s %s (%-6s) %s\n" "$filename" "$checksum" "$size" "$platform"
done

echo ""

# 검증 스크립트 생성
cat > verify-checksums.sh << 'EOF'
#!/bin/bash
# 체크섬 검증 스크립트

set -e

# 색상 코드
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

echo -e "${YELLOW}🔍 체크섬 검증 시작...${NC}"
echo ""

# checksums.txt 파일 존재 확인
if [ ! -f "checksums.txt" ]; then
    echo -e "${RED}❌ checksums.txt 파일을 찾을 수 없습니다${NC}"
    exit 1
fi

# OS별 명령어 선택
if command -v sha256sum >/dev/null 2>&1; then
    CHECK_CMD="sha256sum -c"
elif command -v shasum >/dev/null 2>&1; then
    CHECK_CMD="shasum -a 256 -c"
else
    echo -e "${RED}❌ sha256sum 또는 shasum 명령어를 찾을 수 없습니다${NC}"
    exit 1
fi

# 체크섬 검증
if $CHECK_CMD checksums.txt; then
    echo ""
    echo -e "${GREEN}✅ 모든 파일의 체크섬이 일치합니다!${NC}"
else
    echo ""
    echo -e "${RED}❌ 체크섬 검증 실패!${NC}"
    exit 1
fi
EOF

chmod +x verify-checksums.sh

echo -e "${GREEN}✅ 체크섬 생성 완료!${NC}"
echo ""
echo -e "${YELLOW}💡 검증 방법:${NC}"
echo "  1. 통합 검증: sha256sum -c checksums.txt"
echo "  2. 개별 검증: sha256sum -c aicli-v1.0.0-linux-amd64.sha256"
echo "  3. 스크립트 사용: ./verify-checksums.sh"
echo ""

# 생성된 파일 목록
echo -e "${BLUE}📁 생성된 파일:${NC}"
ls -la checksums.txt verify-checksums.sh *.sha256 2>/dev/null | grep -v "^total"