#!/usr/bin/env bash
set -uo pipefail
export PATH="/Applications/Docker.app/Contents/Resources/bin:$PATH"
TARGET_REG="cr.yichang.puhui.chengfengerlai.com"

images=(
  # gpu-operator current enabled/default operands
  "nvcr.io/nvidia/gpu-operator:v26.3.2"
  "registry.k8s.io/nfd/node-feature-discovery:v0.18.3"
  "nvcr.io/nvidia/k8s/container-toolkit:v1.19.1"
  "nvcr.io/nvidia/k8s-device-plugin:v0.19.2"
  "nvcr.io/nvidia/k8s/dcgm-exporter:4.5.3-4.8.2-distroless"
  "nvcr.io/nvidia/cloud-native/nvidia-fs:2.27.3"
  "nvcr.io/nvidia/cloud-native/gdrdrv:v2.5.2"

  # kube-prometheus-stack rendered/default images
  "ghcr.io/jkroepke/kube-webhook-certgen:1.8.3"
  "quay.io/prometheus-operator/prometheus-operator:v0.91.0"
  "quay.io/prometheus-operator/prometheus-config-reloader:v0.91.0"
  "quay.io/prometheus/alertmanager:v0.32.1"
  "quay.io/prometheus/prometheus:v3.12.0-distroless"
  "quay.io/thanos/thanos:v0.41.0"
  "docker.io/grafana/grafana:13.0.1-security-01"
  "quay.io/kiwigrid/k8s-sidecar:2.7.3"
  "quay.io/prometheus/node-exporter:v1.11.1-distroless"
  "registry.k8s.io/kube-state-metrics/kube-state-metrics:v2.19.0"

  # network-operator current rendered/operator default image refs
  "nvcr.io/nvidia/cloud-native/network-operator:v26.1.1"
  "nvcr.io/nvidia/mellanox/network-operator-init-container:network-operator-v26.1.1"

  # dynamo-system rendered/default images
  "nvcr.io/nvidia/ai-dynamo/kubernetes-operator:1.1.1"
  "docker.io/nats:2.10.21-alpine"
  "docker.io/natsio/nats-server-config-reloader:0.16.0"
  "docker.io/natsio/prometheus-nats-exporter:0.16.0"
)

ok=()
fail=()
for src in "${images[@]}"; do
  dst="$TARGET_REG/$src"
  echo "=== $src -> $dst (linux/amd64)"
  if docker buildx imagetools create --platform linux/amd64 -t "$dst" "$src"; then
    if docker buildx imagetools inspect "$dst" | grep -q 'Platform:.*linux/amd64'; then
      ok+=("$dst")
      echo "OK $dst"
    else
      fail+=("$src :: pushed but amd64 verification failed")
      echo "VERIFY_FAIL $dst"
    fi
  else
    fail+=("$src")
    echo "FAIL $src"
  fi
  echo
done

echo "===== SUMMARY OK ${#ok[@]} ====="
printf '%s\n' "${ok[@]}"
echo "===== SUMMARY FAIL ${#fail[@]} ====="
printf '%s\n' "${fail[@]}"
[[ ${#fail[@]} -eq 0 ]]
