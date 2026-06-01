# kube-prometheus-stack

Deployment assets for kube-prometheus-stack.

## Files

- `install.sh`: Fetches the kube-prometheus-stack Helm chart, installs or
  upgrades the `kube-prometheus-stack` release, and applies the included
  ServiceMonitor resources.
- `values.yaml`: Helm values for the target cluster.
- `servicemonitor.yaml`: ServiceMonitor for scraping DCGM exporter GPU metrics.

## Install

Run from this directory:

```bash
./install.sh
```

The script runs:

```bash
helm fetch https://github.com/prometheus-community/helm-charts/releases/download/kube-prometheus-stack-85.0.3/kube-prometheus-stack-85.0.3.tgz

helm upgrade --install kube-prometheus-stack -n monitoring \
  --create-namespace -n monitoring \
  kube-prometheus-stack-85.0.3.tgz \
  -f values.yaml

kubectl apply -f servicemonitor.yaml
```

## Validate

```bash
kubectl get pods -n monitoring
kubectl get servicemonitor -A
```
