#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${SCRIPT_DIR}"

CHART_ARCHIVE="dynamo-platform-1.1.1.tgz"
CHART_URL="https://helm.ngc.nvidia.com/nvidia/ai-dynamo/charts/${CHART_ARCHIVE}"

if [[ ! -f "${CHART_ARCHIVE}" ]]; then
  helm fetch "${CHART_URL}"
else
  echo "Using existing chart archive: ${CHART_ARCHIVE}"
fi

helm upgrade --install dynamo-platform \
  --namespace dynamo-system --create-namespace \
  --set dynamo-operator.dynamo.metrics.prometheusEndpoint=http://kube-prometheus-stack-prometheus.monitoring.svc.cluster.local:9090 \
  "${CHART_ARCHIVE}" \
  -f values.yaml
