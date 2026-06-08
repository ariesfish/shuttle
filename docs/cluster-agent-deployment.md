# Cluster Agent Deployment

The Cluster Agent runs inside each Inference Cluster as a Kubernetes Deployment. It initiates outbound connections to the Management Plane, registers itself for a pre-created `clusterId`, sends heartbeats, polls tasks, and executes platform-whitelisted operations through local Kubernetes RBAC.

## Runtime Shape

```text
Inference Cluster
  namespace/inference-platform-system
    ServiceAccount/cluster-agent
    Secret/cluster-agent-auth
    ConfigMap/cluster-agent-config
    Deployment/cluster-agent
      container: cluster-agent
        -> outbound HTTP(S) to Management Plane
        -> local kube-apiserver via in-cluster ServiceAccount token
```

The Management Plane does not store a broad kubeconfig for the Inference Cluster. The Cluster Agent uses its local ServiceAccount and RBAC to apply and inspect platform-owned Dynamo resources.

## Registration Flow

1. A platform admin creates an Inference Cluster record in the Management Plane.
2. The Management Plane returns a `clusterId`.
3. The admin installs `deployment/cluster-agent.yaml` into the target Inference Cluster with:
   - Management Plane URL.
   - `clusterId`.
   - Agent auth token.
   - capability metadata.
4. The Cluster Agent Pod starts and calls `POST /v1/agents/register`.
5. The Management Plane validates the token and `clusterId`.
6. The Management Plane creates or updates the Cluster Agent record and returns `agentId`.
7. The Agent starts heartbeat and task polling loops.

## Install

Create a cluster in the Web Console first, then copy its `clusterId`.

Edit `deployment/cluster-agent.yaml`:

```yaml
data:
  management-url: "https://management.example.com"
  cluster-id: "cluster-..."
  capabilities: "dynamo=true,backend=vllm"
  executor-mode: "kubectl"
```

Set the Secret value:

```yaml
stringData:
  auth-token: "<agent token>"
```

Apply it to the target Inference Cluster:

```bash
kubectl apply -f deployment/cluster-agent.yaml
kubectl -n inference-platform-system get pods
kubectl -n inference-platform-system logs deploy/cluster-agent -f
```

## Local Smoke Mode

For local UI smoke tests without a real Kubernetes cluster or Dynamo CRDs, set:

```yaml
data:
  executor-mode: "fake"
```

Fake mode simulates preview, apply, redeploy, and retire task success. Do not use fake mode in real Inference Clusters.

## RBAC Scope

The first manifest grants access to:

- Dynamo CRDs under `nvidia.com`:
  - `dynamographdeployments`
  - `dynamographdeploymentrequests`
  - `dynamocomponentdeployments`
  - `dynamomodels`
- Core resources needed for status, diagnostics, and delete-before-apply cleanup:
  - `pods`
  - `services`
  - `events`
- Apps resources used by operator-managed workloads:
  - `deployments`
  - `replicasets`

This is intentionally not a general-purpose cluster-admin role. Tighten the scope further once the exact managed namespaces and resource labels are stable.

## Production Notes

Phase 1 uses a static bearer token MVP. Before production, replace it with per-agent credentials or mTLS and rotate credentials per Inference Cluster.

Recommended hardening:

- Use a per-cluster Agent token.
- Restrict RBAC to managed namespaces when possible.
- Pin the Agent image by digest.
- Add NetworkPolicy allowing only outbound Management Plane access and kube-apiserver access.
- Add readiness/liveness probes once Agent health endpoints exist.
