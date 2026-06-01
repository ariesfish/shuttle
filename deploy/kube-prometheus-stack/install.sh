#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${SCRIPT_DIR}"

CHART_ARCHIVE="kube-prometheus-stack-85.0.3.tgz"
CHART_URL="https://github.com/prometheus-community/helm-charts/releases/download/kube-prometheus-stack-85.0.3/${CHART_ARCHIVE}"

if [[ ! -f "${CHART_ARCHIVE}" ]]; then
  helm fetch "${CHART_URL}"
else
  echo "Using existing chart archive: ${CHART_ARCHIVE}"
fi

helm upgrade --install kube-prometheus-stack -n monitoring \
  --create-namespace -n monitoring \
  "${CHART_ARCHIVE}" \
  -f values.yaml

kubectl apply -f servicemonitor.yaml
