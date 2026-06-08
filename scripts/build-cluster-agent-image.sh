#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IMAGE_REPO="${IMAGE_REPO:-cr.yichang.puhui.chengfengerlai.com/aistudio/cluster-agent}"
IMAGE_TAG="${IMAGE_TAG:-dev}"
PLATFORM="${PLATFORM:-linux/amd64}"
PUSH="${PUSH:-false}"

IMAGE="${IMAGE_REPO}:${IMAGE_TAG}"

args=(
  buildx build
  --platform "${PLATFORM}"
  -f "${ROOT_DIR}/container/Dockerfile.cluster-agent"
  -t "${IMAGE}"
)

if [[ "${PUSH}" == "true" ]]; then
  args+=(--push)
else
  args+=(--load)
fi

args+=("${ROOT_DIR}/src")

echo "Building ${IMAGE} for ${PLATFORM}"
docker "${args[@]}"

echo "Image: ${IMAGE}"
