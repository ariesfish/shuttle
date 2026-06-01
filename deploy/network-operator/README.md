# Network Operator

Deployment assets for NVIDIA Network Operator.

## Files

- `install.sh`: Fetches the Network Operator Helm chart from NVIDIA NGC,
  installs or upgrades the `network-operator` release, and applies the
  `NicClusterPolicy`.
- `values.yaml`: Helm values for the target cluster.
- `NicClusterPolicy.yaml`: NIC policy for RDMA shared device plugin resources.

## Install

Run from this directory:

```bash
./install.sh
```

The script runs:

```bash
helm fetch https://helm.ngc.nvidia.com/nvidia/charts/network-operator-26.1.0.tgz

helm upgrade --install network-operator \
  -n nvidia-network-operator \
  --create-namespace \
  --wait \
  network-operator-26.1.0.tgz \
  -f values.yaml

kubectl apply -f NicClusterPolicy.yaml -n nvidia-network-operator
```

## Validate

```bash
kubectl get pods -n nvidia-network-operator
kubectl get nicclusterpolicy -n nvidia-network-operator
```
