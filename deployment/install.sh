#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
CHARTS_DIR="${CHARTS_DIR:-${REPO_ROOT}/charts}"
RKE2=false

usage() {
  cat <<'EOF'
Usage:
  ./deployment/install.sh [--rke2]

Options:
  --rke2      Set fixed RKE2 containerd paths in GPU Operator toolkit.env.
              Default upstream Kubernetes install does not override toolkit.env.

Env:
  CHARTS_DIR=/path/to/charts       Helm chart root. Default: ../charts
EOF
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --rke2)
        RKE2=true
        shift
        ;;
      -h|--help)
        usage
        exit 0
        ;;
      *)
        echo "Unknown argument: $1" >&2
        usage >&2
        exit 1
        ;;
    esac
  done
}

gpu_toolkit_set_args() {
  if [[ "${RKE2}" != "true" ]]; then
    return
  fi

  printf '%s\0' \
    --set-string "toolkit.env[0].name=CONTAINERD_SOCKET" \
    --set-string "toolkit.env[0].value=/run/k3s/containerd/containerd.sock" \
    --set-string "toolkit.env[1].name=CONTAINERD_CONFIG" \
    --set-string "toolkit.env[1].value=/var/lib/rancher/rke2/agent/etc/containerd/config.toml"
}

install_gpu_operator() {
  if [[ "${RKE2}" == "true" ]]; then
    echo "==> Installing NVIDIA GPU Operator (RKE2)"
  else
    echo "==> Installing NVIDIA GPU Operator (upstream Kubernetes)"
  fi

  local toolkit_args=()
  mapfile -d '' -t toolkit_args < <(gpu_toolkit_set_args)

  helm upgrade --install gpu-operator \
    -n gpu-operator --create-namespace \
    "${CHARTS_DIR}/gpu-operator" \
    -f "${CHARTS_DIR}/gpu-operator/values.yaml" \
    "${toolkit_args[@]}"
}

install_network_operator() {
  echo "==> Installing NVIDIA Network Operator"

  helm upgrade --install network-operator \
    -n nvidia-network-operator \
    --create-namespace \
    --wait \
    "${CHARTS_DIR}/network-operator" \
    -f "${CHARTS_DIR}/network-operator/values.yaml"
}

install_kube_prometheus_stack() {
  echo "==> Installing kube-prometheus-stack"

  helm upgrade --install kube-prometheus-stack \
    -n monitoring \
    --create-namespace \
    "${CHARTS_DIR}/kube-prometheus-stack" \
    -f "${CHARTS_DIR}/kube-prometheus-stack/values.yaml"
}

install_dynamo_platform() {
  echo "==> Installing NVIDIA Dynamo Platform"

  helm upgrade --install dynamo-platform \
    --namespace dynamo-system --create-namespace \
    --set dynamo-operator.dynamo.metrics.prometheusEndpoint=http://kube-prometheus-stack-prometheus.monitoring.svc.cluster.local:9090 \
    "${CHARTS_DIR}/dynamo-system" \
    -f "${CHARTS_DIR}/dynamo-system/values.yaml"
}

apply_manifest() {
  local manifest="$1"

  echo "==> Applying ${manifest#"${SCRIPT_DIR}/"}"
  kubectl apply -f "${manifest}"
}

apply_extra_manifests() {
  echo "==> Applying additional manifests"

  apply_manifest "${SCRIPT_DIR}/nic-cluster-policy.yaml"
  apply_manifest "${SCRIPT_DIR}/dcgm-exporter-monitor.yaml"
  apply_manifest "${SCRIPT_DIR}/grafana-nodeport.yaml"
  apply_manifest "${SCRIPT_DIR}/nats-pv.yaml"
  apply_manifest "${SCRIPT_DIR}/model-cache-pvc.yaml"

  for manifest in "${SCRIPT_DIR}"/observability/*.yaml; do
    apply_manifest "${manifest}"
  done
}

main() {
  parse_args "$@"

  install_gpu_operator
  install_network_operator
  install_kube_prometheus_stack
  install_dynamo_platform
  apply_extra_manifests

  echo "==> Install complete"
  echo "Validate with:"
  echo "  kubectl get pods -n gpu-operator"
  echo "  kubectl get pods -n nvidia-network-operator"
  echo "  kubectl get pods -n monitoring"
  echo "  kubectl get pods -n dynamo-system"
}

main "$@"
