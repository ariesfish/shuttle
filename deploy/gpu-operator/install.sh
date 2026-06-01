#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${SCRIPT_DIR}"

CHART_ARCHIVE="gpu-operator-v26.3.1.tgz"
CHART_URL="https://helm.ngc.nvidia.com/nvidia/charts/${CHART_ARCHIVE}"

if [[ ! -f "${CHART_ARCHIVE}" ]]; then
  helm fetch "${CHART_URL}"
else
  echo "Using existing chart archive: ${CHART_ARCHIVE}"
fi

helm upgrade --install gpu-operator \
  -n gpu-operator --create-namespace \
  "${CHART_ARCHIVE}" \
  --set toolkit.hostPaths.runtimeDir=/run/k3s/containerd \
  -f values.yaml
