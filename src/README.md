# Inference Platform Control Loop

This is the Phase 1 API-first skeleton for the Management Plane and Cluster Agent.

## Run Management API

```bash
cd src
go run ./cmd/management-api -addr :8080 -data data/management.json

# Or use Postgres-backed persistence:
go run ./cmd/management-api \
  -addr :8080 \
  -postgres-dsn 'postgres://user:pass@localhost:5432/inference?sslmode=disable'

# Enable static bearer-token auth:
go run ./cmd/management-api \
  -addr :8080 \
  -auth-token dev-secret
```

## Run Cluster Agent

Create a cluster first, then run the agent with that cluster ID:

```bash
cd src
go run ./cmd/cluster-agent \
  -management-url http://localhost:8080 \
  -cluster-id cluster-2 \
  -auth-token dev-secret \
  -capability dynamo=true,backend=vllm

# For local UI smoke without kubectl or Dynamo CRDs:
go run ./cmd/cluster-agent \
  -management-url http://localhost:8080 \
  -cluster-id cluster-2 \
  -auth-token dev-secret \
  -executor-mode fake \
  -capability dynamo=true,backend=vllm
```

In fake executor mode, the Cluster Agent also reports a synthetic Accelerator Inventory snapshot for local Phase 2 UI/API smoke testing. Kubectl/NVIDIA/RDMA discovery is intentionally not used in this mode.

## Smoke Test

```bash
curl -s localhost:8080/healthz

curl -s localhost:8080/v1/projects \
  -H 'Content-Type: application/json' \
  -d '{"name":"platform"}'

curl -s localhost:8080/v1/clusters \
  -H 'Content-Type: application/json' \
  -d '{"name":"h200-a","prometheusUrl":"http://prometheus.local","grafanaUrl":"http://grafana.local"}'

curl -s localhost:8080/v1/agents/register \
  -H 'Content-Type: application/json' \
  -d '{"clusterId":"cluster-2","version":"v0.1.0","capabilities":{"dynamo":"true"}}'

curl -s localhost:8080/v1/tasks \
  -H 'Content-Type: application/json' \
  -d '{"clusterId":"cluster-2","type":"InspectDeploymentStatus","payload":{"servingApplicationId":"sa-1"}}'

curl -s localhost:8080/v1/clusters/cluster-2/tasks:lease \
  -H 'Content-Type: application/json' \
  -d '{"agentId":"agent-3"}'
```

If the Cluster Agent is running, it will lease and complete pending tasks in no-op mode.

Register a cached model artifact and create a Serving Application:

```bash
curl -s localhost:8080/v1/artifacts \
  -H 'Content-Type: application/json' \
  -d '{"family":"deepseek-v4","variant":"flash","revision":"rev1","pvcMountPath":"/home/dynamo/.cache/huggingface","pvcModelPath":"models--deepseek-ai--DeepSeek-V4-Flash/snapshots/rev1","hostCachePath":"/data/cache/hub","quantization":"fp8"}'

curl -s localhost:8080/v1/apps \
  -H 'Content-Type: application/json' \
  -d '{"projectId":"project-1","name":"DeepSeek V4 Flash","model":{"family":"deepseek-v4","variant":"flash","artifactId":"artifact-4","quantization":"fp8"},"placement":{"clusterId":"cluster-2","namespace":"dynamo-system"},"runtime":{"backend":"vllm","topology":"pd-disagg","recipe":"deepseek-v4-flash-vllm-dgd-disagg"},"service":{"endpointName":"deepseek-v4-flash","protocol":"openai-compatible","exposure":"cluster-local"},"optimization":{"target":"throughput","profilingMode":"disabled"}}'
```

Create a server-side dry-run preview task from the Serving Application:

```bash
curl -s -X POST localhost:8080/v1/apps/app-5/tasks/preview
```

For `PreviewDeploymentDiff`, the Cluster Agent runs `kubectl apply --dry-run=server` against the target cluster.

Create an apply task from the same Serving Application:

```bash
curl -s -X POST localhost:8080/v1/apps/app-5/tasks/apply
```

For `ApplyDeployment`, the Cluster Agent runs `kubectl apply` and then polls the rendered `DynamoGraphDeployment` status until it reaches a terminal phase or times out.

Create a delete-before-apply redeploy task:

```bash
curl -s -X POST localhost:8080/v1/apps/app-5/tasks/redeploy
```

For `DeleteBeforeApplyRedeploy`, the Cluster Agent deletes the rendered `DynamoGraphDeployment`, waits for it to disappear, applies the rendered manifest, and watches status.

Create a retire task:

```bash
curl -s -X POST localhost:8080/v1/apps/app-5/tasks/retire
```

For `RetireDeployment`, the Cluster Agent deletes the rendered `DynamoGraphDeployment` and waits for it to disappear.

List Accelerator Inventory, registered endpoints, observability entry points, and audit records:

```bash
curl -s localhost:8080/v1/clusters/cluster-2/accelerator-inventory
curl -s localhost:8080/v1/endpoints
curl -s localhost:8080/v1/apps/app-5/observability
curl -s localhost:8080/v1/audit-records
```

Apply/redeploy task completion creates or updates the endpoint registry entry. Retire task completion removes it. Observability entries return Grafana deep links and Prometheus query templates; the Management Plane does not ingest raw metrics.

IDs are generated sequentially in the JSON data file.
