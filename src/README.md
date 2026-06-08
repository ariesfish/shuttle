# Inference Platform Control Loop

This is the Phase 1 API-first skeleton for the Management Plane and Cluster Agent.

## Run Management API

```bash
cd src
go run ./cmd/management-api -addr :8080 -data data/management.json
```

## Run Cluster Agent

Create a cluster first, then run the agent with that cluster ID:

```bash
cd src
go run ./cmd/cluster-agent \
  -management-url http://localhost:8080 \
  -cluster-id cluster-2 \
  -capability dynamo=true,backend=vllm
```

## Smoke Test

```bash
curl -s localhost:8080/healthz

curl -s localhost:8080/v1/projects \
  -H 'Content-Type: application/json' \
  -d '{"name":"platform"}'

curl -s localhost:8080/v1/clusters \
  -H 'Content-Type: application/json' \
  -d '{"name":"h200-a"}'

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
curl -s localhost:8080/v1/model-artifacts \
  -H 'Content-Type: application/json' \
  -d '{"family":"deepseek-v4","variant":"flash","revision":"rev1","pvcMountPath":"/home/dynamo/.cache/huggingface","pvcModelPath":"models--deepseek-ai--DeepSeek-V4-Flash/snapshots/rev1","hostCachePath":"/data/cache/hub","quantization":"fp8"}'

curl -s localhost:8080/v1/serving-applications \
  -H 'Content-Type: application/json' \
  -d '{"projectId":"project-1","name":"DeepSeek V4 Flash","model":{"family":"deepseek-v4","variant":"flash","artifactId":"artifact-4","quantization":"fp8"},"placement":{"clusterId":"cluster-2","namespace":"dynamo-system"},"runtime":{"backend":"vllm","topology":"pd-disagg","recipe":"deepseek-v4-flash-vllm-dgd-disagg"},"service":{"endpointName":"deepseek-v4-flash","protocol":"openai-compatible","exposure":"cluster-local"},"optimization":{"target":"throughput","profilingMode":"disabled"}}'
```

Create a server-side dry-run preview task from the Serving Application:

```bash
curl -s -X POST localhost:8080/v1/serving-applications/app-5/preview-task
```

For `PreviewDeploymentDiff`, the Cluster Agent runs `kubectl apply --dry-run=server` against the target cluster.

Create an apply task from the same Serving Application:

```bash
curl -s -X POST localhost:8080/v1/serving-applications/app-5/apply-task
```

For `ApplyDeployment`, the Cluster Agent runs `kubectl apply` and then polls the rendered `DynamoGraphDeployment` status until it reaches a terminal phase or times out.

Create a delete-before-apply redeploy task:

```bash
curl -s -X POST localhost:8080/v1/serving-applications/app-5/redeploy-task
```

For `DeleteBeforeApplyRedeploy`, the Cluster Agent deletes the rendered `DynamoGraphDeployment`, waits for it to disappear, applies the rendered manifest, and watches status.

Create a retire task:

```bash
curl -s -X POST localhost:8080/v1/serving-applications/app-5/retire-task
```

For `RetireDeployment`, the Cluster Agent deletes the rendered `DynamoGraphDeployment` and waits for it to disappear.

IDs are generated sequentially in the JSON data file.
