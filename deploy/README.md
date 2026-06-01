# Deploy

This directory contains deployment assets for the phase-one single-cluster
inference platform.

## Components

- `gpu-operator`: NVIDIA GPU Operator installation and validation assets.
- `network-operator`: NVIDIA Network Operator installation and validation assets.
- `kube-prometheus-stack`: Prometheus, Alertmanager, Grafana, and dashboard assets.
- `dynamo`: NVIDIA Dynamo runtime deployment assets.
- `zhiliu`: Zhiliu platform service deployment assets.

## Suggested Order

1. Install cluster infrastructure operators.
2. Install observability addons.
3. Install Dynamo runtime components.
4. Install Zhiliu platform components.
5. Validate GPUs, networking, node-local storage, metrics, and inference endpoints.
