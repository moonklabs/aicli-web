#!/bin/bash
# Claude CLI 설치 스크립트

set -e

echo "Installing Claude CLI..."

# Claude CLI 바이너리 다운로드 URL (예시)
# 실제 환경에서는 올바른 URL로 교체 필요
CLAUDE_CLI_VERSION="latest"
CLAUDE_CLI_URL="https://github.com/anthropics/claude-cli/releases/download/${CLAUDE_CLI_VERSION}/claude-linux-amd64"

# 임시 디렉토리 생성
TEMP_DIR=$(mktemp -d)
cd $TEMP_DIR

# Claude CLI 다운로드
echo "Downloading Claude CLI from ${CLAUDE_CLI_URL}..."
if command -v wget &> /dev/null; then
    wget -q -O claude "${CLAUDE_CLI_URL}" || {
        echo "Warning: Could not download Claude CLI. Using mock installation for development."
        # 개발 환경용 mock Claude CLI 생성
        cat > claude << 'EOF'
#!/bin/bash
# Mock Claude CLI for development
echo "Mock Claude CLI v0.1.0"
echo "This is a development placeholder."
echo "Arguments: $@"

# 기본 명령어 처리
case "$1" in
    --version)
        echo "claude version 0.1.0-mock"
        ;;
    --help)
        echo "Usage: claude [command] [options]"
        echo "Commands:"
        echo "  chat     Start an interactive chat"
        echo "  run      Run a single command"
        echo "  --help   Show this help message"
        echo "  --version Show version information"
        ;;
    chat)
        echo "Starting mock chat session..."
        echo "Type 'exit' to quit."
        while IFS= read -r line; do
            if [ "$line" = "exit" ]; then
                break
            fi
            echo "Mock response to: $line"
        done
        ;;
    run)
        shift
        echo "Mock execution of: $@"
        ;;
    *)
        echo "Unknown command: $1"
        echo "Try 'claude --help' for more information."
        exit 1
        ;;
esac
EOF
    }
else
    curl -sL -o claude "${CLAUDE_CLI_URL}" || {
        echo "Warning: Could not download Claude CLI. Using mock installation for development."
        # 위와 동일한 mock 스크립트 생성
        cat > claude << 'EOF'
#!/bin/bash
# Mock Claude CLI for development
echo "Mock Claude CLI v0.1.0"
echo "This is a development placeholder."
echo "Arguments: $@"

# 기본 명령어 처리
case "$1" in
    --version)
        echo "claude version 0.1.0-mock"
        ;;
    --help)
        echo "Usage: claude [command] [options]"
        echo "Commands:"
        echo "  chat     Start an interactive chat"
        echo "  run      Run a single command"
        echo "  --help   Show this help message"
        echo "  --version Show version information"
        ;;
    chat)
        echo "Starting mock chat session..."
        echo "Type 'exit' to quit."
        while IFS= read -r line; do
            if [ "$line" = "exit" ]; then
                break
            fi
            echo "Mock response to: $line"
        done
        ;;
    run)
        shift
        echo "Mock execution of: $@"
        ;;
    *)
        echo "Unknown command: $1"
        echo "Try 'claude --help' for more information."
        exit 1
        ;;
esac
EOF
    }
fi

# 실행 권한 추가
chmod +x claude

# 사용자 로컬 bin 디렉토리에 설치
mkdir -p $HOME/.local/bin
mv claude $HOME/.local/bin/

# PATH에 추가 (bashrc)
if ! grep -q "$HOME/.local/bin" $HOME/.bashrc; then
    echo 'export PATH="$HOME/.local/bin:$PATH"' >> $HOME/.bashrc
fi

# 정리
cd /
rm -rf $TEMP_DIR

# 설치 확인
export PATH="$HOME/.local/bin:$PATH"
if command -v claude &> /dev/null; then
    echo "Claude CLI installed successfully!"
    claude --version
else
    echo "Warning: Claude CLI installation may have failed."
    exit 1
fi