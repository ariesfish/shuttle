# Dynamo

Deployment assets for NVIDIA Dynamo.

## Files

- `install.sh`: Fetches the Dynamo Platform Helm chart from NVIDIA NGC and
  installs or upgrades the `dynamo-platform` release.
- `values.yaml`: Helm values for the target cluster.

## Install

Run from this directory:

```bash
./install.sh
```

The script runs:

```bash
helm fetch https://helm.ngc.nvidia.com/nvidia/ai-dynamo/charts/dynamo-platform-1.1.1.tgz

helm upgrade --install dynamo-platform \
  --namespace dynamo-system --create-namespace \
  --set dynamo-operator.dynamo.metrics.prometheusEndpoint=http://kube-prometheus-stack-prometheus.monitoring.svc.cluster.local:9090 \
  dynamo-platform-1.1.1.tgz \
  -f values.yaml
```

## Validate

```bash
kubectl get pods -n dynamo-system
kubectl get crd | grep -i dynamo
```
