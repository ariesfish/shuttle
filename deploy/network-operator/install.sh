#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${SCRIPT_DIR}"

CHART_ARCHIVE="network-operator-26.1.0.tgz"
CHART_URL="https://helm.ngc.nvidia.com/nvidia/charts/${CHART_ARCHIVE}"

if [[ ! -f "${CHART_ARCHIVE}" ]]; then
  helm fetch "${CHART_URL}"
else
  echo "Using existing chart archive: ${CHART_ARCHIVE}"
fi

helm upgrade --install network-operator \
  -n nvidia-network-operator \
  --create-namespace \
  --wait \
  "${CHART_ARCHIVE}" \
  -f values.yaml

kubectl apply -f NicClusterPolicy.yaml -n nvidia-network-operator
