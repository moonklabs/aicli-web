# AICode Manager Agent Image
# Ubuntu 기반 Claude CLI 실행 환경

FROM ubuntu:22.04

# 환경 변수 설정
ENV DEBIAN_FRONTEND=noninteractive
ENV TZ=Asia/Seoul
ENV WORKSPACE=/workspace
ENV AGENT_HOME=/home/agent

# 기본 패키지 설치
RUN apt-get update && apt-get install -y \
    # 기본 도구
    curl \
    wget \
    git \
    vim \
    nano \
    # 빌드 도구
    build-essential \
    make \
    gcc \
    g++ \
    # Python 환경
    python3 \
    python3-pip \
    python3-venv \
    # Node.js 환경
    nodejs \
    npm \
    # 시스템 도구
    sudo \
    ca-certificates \
    gnupg \
    lsb-release \
    # 네트워크 도구
    net-tools \
    iputils-ping \
    # 프로세스 관리
    htop \
    psmisc \
    # 로케일 설정
    locales \
    && rm -rf /var/lib/apt/lists/*

# 로케일 설정
RUN locale-gen en_US.UTF-8 ko_KR.UTF-8
ENV LANG=en_US.UTF-8
ENV LC_ALL=en_US.UTF-8

# 시간대 설정
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# Agent 사용자 생성
RUN useradd -m -s /bin/bash -u 1000 agent && \
    echo "agent ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

# Go 1.21 설치
RUN wget -q https://go.dev/dl/go1.21.6.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz && \
    rm go1.21.6.linux-amd64.tar.gz

# Go 환경 변수 설정
ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPATH="/home/agent/go"
ENV PATH="${GOPATH}/bin:${PATH}"

# Claude CLI 설치 스크립트 복사
COPY scripts/install-claude-cli.sh /tmp/
RUN chmod +x /tmp/install-claude-cli.sh

# Claude CLI 설치 (agent 사용자로)
USER agent
RUN /tmp/install-claude-cli.sh

# 작업 디렉토리 생성
RUN mkdir -p ${WORKSPACE} ${AGENT_HOME}/.claude ${AGENT_HOME}/.config/claude

# 기본 설정 파일
COPY config/claude-config.yaml ${AGENT_HOME}/.config/claude/config.yaml

# 엔트리포인트 스크립트
COPY scripts/entrypoint.sh /usr/local/bin/entrypoint.sh
USER root
RUN chmod +x /usr/local/bin/entrypoint.sh

# 작업 디렉토리 설정
WORKDIR ${WORKSPACE}

# 볼륨 설정
VOLUME ["${WORKSPACE}", "${AGENT_HOME}/.claude"]

# 기본 사용자로 전환
USER agent

# 엔트리포인트
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["--help"]