# GPU Operator

Deployment assets for NVIDIA GPU Operator.

## Files

- `install.sh`: Fetches the GPU Operator Helm chart from NVIDIA NGC and installs
  or upgrades the `gpu-operator` release.
- `values.yaml`: Helm values for the target cluster.

## Install

Run from this directory:

```bash
./install.sh
```

The script runs:

```bash
helm fetch https://helm.ngc.nvidia.com/nvidia/charts/gpu-operator-v26.3.1.tgz

helm upgrade --install gpu-operator \
  -n gpu-operator --create-namespace \
  gpu-operator-v26.3.1.tgz \
  --set toolkit.hostPaths.runtimeDir=/run/k3s/containerd \
  -f values.yaml
```

## Validate

```bash
kubectl get pods -n gpu-operator
kubectl get nodes -l nvidia.com/gpu.present=true
```
