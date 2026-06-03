# Deployment

This directory contains cluster-level deployment assets for the phase-one
single-cluster inference platform. Helm charts live in `../charts`; this
directory keeps the top-level installer and standalone Kubernetes manifests.

## Layout

- `install.sh`: Installs the local Helm charts from `../charts`, then applies the
  standalone YAML manifests in this directory.
- `nic-cluster-policy.yaml`: NVIDIA Network Operator `NicClusterPolicy` for RDMA
  shared device plugin resources.
- `dcgm-exporter-monitor.yaml`: `ServiceMonitor` for GPU metrics from DCGM
  exporter.
- `grafana-nodeport.yaml`: NodePort service for Grafana.
- `nats-pv.yaml`: HostPath persistent volume for Dynamo NATS JetStream data.
- `model-cache-pvc.yaml`: PVC for model cache storage in `dynamo-system`.
- `observability/`: Grafana dashboard ConfigMaps.
- `examples/`: Example Dynamo graph deployment manifests. These are samples and
  are not applied by `install.sh`.

## Install

Default upstream Kubernetes/containerd install:

```bash
./deployment/install.sh
```

RKE2 install:

```bash
./deployment/install.sh --rke2
```

`--rke2` sets fixed GPU Operator toolkit environment values:

```text
CONTAINERD_SOCKET=/run/k3s/containerd/containerd.sock
CONTAINERD_CONFIG=/var/lib/rancher/rke2/agent/etc/containerd/config.toml
```

The default upstream Kubernetes install does not override `toolkit.env`.

If the charts are not in `../charts`, set `CHARTS_DIR`:

```bash
CHARTS_DIR=/path/to/charts ./deployment/install.sh
```

## Install order

`install.sh` runs in this order:

1. NVIDIA GPU Operator chart: `../charts/gpu-operator`
2. NVIDIA Network Operator chart: `../charts/network-operator`
3. kube-prometheus-stack chart: `../charts/kube-prometheus-stack`
4. NVIDIA Dynamo Platform chart: `../charts/dynamo-system`
5. Standalone manifests:
   - `nic-cluster-policy.yaml`
   - `dcgm-exporter-monitor.yaml`
   - `grafana-nodeport.yaml`
   - `nats-pv.yaml`
   - `model-cache-pvc.yaml`
   - `observability/*.yaml`

## Validate

```bash
kubectl get pods -n gpu-operator
kubectl get pods -n nvidia-network-operator
kubectl get pods -n monitoring
kubectl get pods -n dynamo-system
kubectl get nicclusterpolicy -n nvidia-network-operator
kubectl get servicemonitor -A
kubectl get pv dynamo-platform-nats-js
kubectl get pvc -n dynamo-system model-cache
```

## Example workloads

After the platform is installed, apply example workloads manually as needed:

```bash
kubectl apply -f deployment/examples/qwen3-0-6b-vllm-dgdr-quickstart.yaml
```
