#!/bin/bash
# Agent 컨테이너 엔트리포인트 스크립트

set -e

# 환경 변수 확인
echo "Starting AICode Manager Agent..."
echo "Workspace: ${WORKSPACE}"
echo "Agent Home: ${AGENT_HOME}"
echo "User: $(whoami)"
echo "UID: $(id -u)"
echo "GID: $(id -g)"

# Claude CLI PATH 설정
export PATH="$HOME/.local/bin:$PATH"

# Claude API 키 확인
if [ -n "$CLAUDE_API_KEY" ]; then
    echo "Claude API key is set"
    # API 키를 Claude CLI 설정에 저장
    mkdir -p $HOME/.config/claude
    echo "api_key: $CLAUDE_API_KEY" > $HOME/.config/claude/credentials.yaml
    chmod 600 $HOME/.config/claude/credentials.yaml
else
    echo "Warning: CLAUDE_API_KEY is not set"
fi

# 작업 디렉토리 권한 확인
if [ -w "$WORKSPACE" ]; then
    echo "Workspace is writable"
else
    echo "Warning: Workspace is not writable"
fi

# Git 설정 (필요한 경우)
if [ -n "$GIT_USER_NAME" ]; then
    git config --global user.name "$GIT_USER_NAME"
fi
if [ -n "$GIT_USER_EMAIL" ]; then
    git config --global user.email "$GIT_USER_EMAIL"
fi

# SSH 에이전트 설정 (필요한 경우)
if [ -n "$SSH_AUTH_SOCK" ]; then
    echo "SSH agent forwarding is enabled"
fi

# 프로세스 상태 파일
STATUS_FILE="${AGENT_HOME}/.agent_status"
echo "ready" > "$STATUS_FILE"

# 시그널 핸들러
cleanup() {
    echo "Shutting down agent..."
    echo "stopping" > "$STATUS_FILE"
    # 진행 중인 작업 정리
    exit 0
}

trap cleanup SIGTERM SIGINT

# Claude CLI 실행
if [ "$#" -eq 0 ]; then
    # 인자가 없으면 대화형 모드
    echo "Starting Claude CLI in interactive mode..."
    exec claude chat
elif [ "$1" = "sleep" ]; then
    # 컨테이너를 계속 실행 상태로 유지 (테스트용)
    echo "Agent container is running in sleep mode..."
    while true; do
        sleep 30
        echo "running" > "$STATUS_FILE"
    done
else
    # 명령어 실행
    echo "Executing: claude $@"
    exec claude "$@"
fi