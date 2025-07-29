#!/bin/bash
# Docker 에이전트 이미지 빌드 스크립트

set -e

# 스크립트 디렉토리
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
cd "$SCRIPT_DIR"

# 이미지 정보
IMAGE_NAME="aicli-agent"
IMAGE_TAG="${1:-latest}"
FULL_IMAGE_NAME="${IMAGE_NAME}:${IMAGE_TAG}"

echo "Building Docker image: ${FULL_IMAGE_NAME}"

# 빌드 인자
BUILD_ARGS=""
if [ -n "$HTTP_PROXY" ]; then
    BUILD_ARGS="${BUILD_ARGS} --build-arg HTTP_PROXY=${HTTP_PROXY}"
fi
if [ -n "$HTTPS_PROXY" ]; then
    BUILD_ARGS="${BUILD_ARGS} --build-arg HTTPS_PROXY=${HTTPS_PROXY}"
fi

# Docker 빌드
docker build \
    ${BUILD_ARGS} \
    --tag "${FULL_IMAGE_NAME}" \
    --file Dockerfile \
    .

# 빌드 확인
if [ $? -eq 0 ]; then
    echo "Successfully built ${FULL_IMAGE_NAME}"
    
    # 이미지 정보 출력
    docker images "${IMAGE_NAME}" --filter "label=tag=${IMAGE_TAG}"
    
    # 이미지 크기 확인
    IMAGE_SIZE=$(docker images "${FULL_IMAGE_NAME}" --format "{{.Size}}")
    echo "Image size: ${IMAGE_SIZE}"
else
    echo "Failed to build Docker image"
    exit 1
fi

# 개발 환경에서는 추가 태그 생성
if [ "${IMAGE_TAG}" != "latest" ]; then
    docker tag "${FULL_IMAGE_NAME}" "${IMAGE_NAME}:latest"
    echo "Also tagged as ${IMAGE_NAME}:latest"
fi