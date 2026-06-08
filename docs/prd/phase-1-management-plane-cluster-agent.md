# Phase 1 PRD: Management Plane and Cluster Agent

## Problem Statement

Operators need a product-facing way to deploy, observe, and retire large-model Serving Applications such as DeepSeek V4 Flash/Pro across one or more Inference Clusters without manually using kube-apiserver access, raw Dynamo CRDs, or cluster-local dashboards as the primary workflow.

The current deployment assets already support Dynamo, SGLang/vLLM, Prometheus, Grafana, and cluster-local examples, but they do not provide a Management Plane, Project-level ownership, Cluster Agent task execution, Endpoint Registry, or durable deployment lifecycle history.

## Solution

Build the Phase 1 platform loop around two deep modules:

1. **Management Plane**: the product control surface for Projects, Model Artifacts, Serving Applications, task orchestration, Endpoint Registry, audit history, and monitoring entry points.
2. **Cluster Agent**: the in-cluster whitelist task executor that registers with the Management Plane, polls for assigned tasks, applies platform-generated Dynamo resources through local RBAC, watches runtime status, and reports bounded status/events back.

Phase 1 is API-first: the Management API and domain services define the stable contract, while the Web Console starts with the minimum pages required to drive the deployment loop.

Implementation should prioritize the Cluster Agent path early because the end-to-end deployment loop depends on registration, polling, task execution, status watch, and safe redeploy behavior.

Phase 1 supports cluster-local serving URLs, cached Model Artifacts, local Prometheus summary queries, and Grafana deep links. It does not include a Global Inference Gateway, automatic model download/distribution, global metrics backend, iframe Grafana embedding, or arbitrary Kubernetes administration.

## User Stories

1. As a platform operator, I want to register an Inference Cluster, so that I can deploy Serving Applications to it from the Management Plane.
2. As a platform operator, I want each Inference Cluster to show its Agent health, so that I can trust whether deployment tasks can be executed.
3. As a platform operator, I want to see Cluster Agent version and capabilities, so that I can identify incompatible clusters before deployment.
4. As a platform operator, I want to group work under a Project, so that users, permissions, Serving Applications, and resource access have a product boundary.
5. As a platform operator, I want to grant a Project access to specific Inference Clusters, so that teams cannot deploy to unauthorized clusters.
6. As a platform operator, I want to grant a Project access to specific Accelerator Pools, so that scarce resources are controlled.
7. As a platform operator, I want to register cached Model Artifacts, so that deployments can reference known local model snapshots.
8. As a platform operator, I want Model Artifacts to record revision, cache path, PVC reference, quantization, and compatibility metadata, so that deployment validation catches obvious mismatches.
9. As a model operator, I want to create a Serving Application from DeepSeek V4 Flash/Pro, so that I can run a model service without hand-writing every CRD.
10. As a model operator, I want to choose an Inference Cluster and Accelerator Pool, so that I can control where the Serving Application runs.
11. As a model operator, I want to choose serving backend and Serving Topology, so that I can select SGLang/vLLM and aggregated or disaggregated serving paths.
12. As a model operator, I want to choose an optimization target, so that deployment intent captures throughput, latency, or SLA-oriented behavior.
13. As a model operator, I want the Management Plane to validate my intent before deployment, so that missing model cache, unsupported backend, or unauthorized cluster choices fail early.
14. As a model operator, I want to preview the generated deployment diff, so that I know what the Cluster Agent will apply.
15. As an approver, I want deployment changes to enter PendingApproval when required, so that risky GPU workload changes are reviewed.
16. As a model operator, I want approved deployments to create tasks for the Cluster Agent, so that I do not need direct kube-apiserver access.
17. As a cluster administrator, I want the Cluster Agent to use local RBAC, so that the Management Plane does not hold a broad kubeconfig.
18. As a security reviewer, I want Cluster Agent tasks to be whitelisted, so that arbitrary kubectl commands or user-provided YAML cannot be executed through the agent.
19. As a platform operator, I want Agent tasks to use leases, so that only one Agent execution owns a task at a time.
20. As a platform operator, I want Agent task results to be idempotent, so that retry after reconnect does not corrupt deployment state.
21. As a model operator, I want the Cluster Agent to support delete-before-apply for material redeployments, so that scarce GPU clusters do not deadlock during rolling updates.
22. As a model operator, I want to see deployment lifecycle states, so that I know whether a Serving Application is Draft, Validated, PendingApproval, Applying, Profiling, Deploying, Ready, Failed, or Retiring.
23. As a model operator, I want failed deployments to show actionable failure reasons, so that I can revise intent or inspect cluster-side details.
24. As a model operator, I want the Management Plane to show the serving endpoint URL when a Serving Application is Ready, so that callers know where to send inference requests.
25. As an application caller, I want the Phase 1 endpoint to be cluster-local, so that serving traffic goes directly to the target Inference Cluster.
26. As a platform operator, I want Endpoint Registry entries to be tied to Serving Applications, so that I can list and retire stale endpoints.
27. As a platform operator, I want local Prometheus summary metrics in the Management Plane, so that I can see basic health without opening Grafana immediately.
28. As a platform operator, I want Grafana dashboard deep links, so that I can drill down into existing cluster-local dashboards.
29. As a platform operator, I want the Cluster Agent to report bounded events and status summaries, so that the Management Plane stays responsive without becoming a metrics or log pipeline.
30. As a platform operator, I want bounded diagnostic log tail support, so that common failures can be inspected without full log aggregation in Phase 1.
31. As a platform operator, I want audit history for Serving Application changes, so that I can answer who changed what and why.
32. As a platform operator, I want previous Serving Application versions recorded, so that I can understand rollout history.
33. As a model operator, I want to retire a Serving Application, so that the Management Plane can coordinate deletion and endpoint cleanup.
34. As a platform operator, I want retired deployments to retain audit and tuning history, so that operational history is not lost.
35. As a platform operator, I want a clear unsupported-feature boundary, so that users do not expect Phase 1 to download models, globally route inference traffic, or provide arbitrary cluster administration.

## Implementation Decisions

### Management Plane Modules

Phase 1 should build the Management Plane API-first. Web Console work should consume the same API and remain minimal until the deployment loop is stable.

- **Project Service**: owns Project identity, membership, authorization, and resource access grants.
- **Cluster Registry**: stores Inference Cluster identity, Agent registration state, capabilities, heartbeat, local observability URLs, and cluster-local endpoint patterns.
- **Model Artifact Registry**: stores cached Model Artifact metadata, including model family, revision, cache path, PVC reference, quantization, and compatibility metadata.
- **Serving Application Service**: owns the product lifecycle object, state machine, desired intent, active version, previous versions, endpoint reference, and observability references.
- **Deployment Intent Validator**: checks Project access, artifact availability metadata, cluster capability, backend/topology compatibility, and required fields before task creation.
- **Manifest Renderer**: converts Serving Application intent into platform-generated Dynamo resources for dry-run and apply tasks.
- **Task Orchestrator**: creates whitelist tasks, manages task state, leases, retries, idempotency keys, and task result records.
- **Agent Gateway**: authenticates Cluster Agents, receives heartbeats, serves polling requests, accepts status/event cursors, and records task results.
- **Endpoint Registry**: stores cluster-local serving URLs and their Serving Application ownership.
- **Observability Entry Service**: stores Grafana deep links and selected Prometheus query templates; provides summary metric queries without ingesting raw metrics.
- **Audit Service**: records user actions, approvals, generated diffs, task outcomes, and change reasons.

### Cluster Agent Modules

The Cluster Agent should be implemented in the first batch of Phase 1 work so the platform can validate the full control loop before polishing the Web Console.

- **Agent Runtime**: starts the agent, loads cluster identity, authenticates to the Management Plane, and sends heartbeats.
- **Capability Reporter**: reports Kubernetes version, Dynamo CRD availability, supported backend hints, observability endpoints, and resource labels available to the platform.
- **Task Poller**: polls assigned tasks from the Agent Gateway, renews leases, and handles reconnects.
- **Task Executor**: executes only whitelist task types such as validate, preview diff, apply deployment, delete-before-apply redeploy, inspect status, retire deployment, and bounded diagnostics.
- **Kubernetes Adapter**: applies platform-generated resources, performs server-side dry-run, watches CRD status/events, and reads selected pods/services.
- **Dynamo Adapter**: understands DGDR/DGD/DCD status and maps runtime state back to Serving Application lifecycle states.
- **Safe Redeploy Controller**: implements delete-before-apply sequencing for scarce accelerator clusters.
- **Status Sync**: sends bounded lifecycle status, events, endpoint readiness, and task results back to the Management Plane using cursors.
- **Diagnostics Provider**: returns bounded logs and selected Kubernetes events for troubleshooting.

### Phase 1 State Machine

Serving Application lifecycle states:

```text
Draft -> Validated -> PendingApproval -> Applying -> Profiling? -> Deploying -> Ready
                                                            \-> Failed
Ready -> PendingApproval -> Applying
Ready -> Retiring -> Retired
Failed -> Draft
```

The platform should record transition reason, actor, timestamp, active version, and related task ID for every transition.

### Phase 1 Task Types

- `RegisterCluster`
- `ValidateIntent`
- `PreviewDeploymentDiff`
- `ApplyDeployment`
- `DeleteBeforeApplyRedeploy`
- `InspectDeploymentStatus`
- `RetireDeployment`
- `FetchDiagnostics`
- `SyncEndpointReadiness`

Task payloads must be generated by platform services, not supplied as arbitrary user YAML.

### Phase 1 API Surface

The Management Plane API should expose product resources, not raw Kubernetes resources:

- Projects
- Inference Clusters
- Accelerator Pools
- Model Artifacts
- Serving Applications
- Deployment Tasks
- Endpoint Registry entries
- Observability entries
- Audit records

Raw DGDR/DGD/DCD details may be available as read-only debug views, but they are not the primary user workflow.

### Observability Decisions

- Local Prometheus remains the source for cluster-level and Serving Application summary metrics.
- Local Grafana remains the drill-down dashboard for Phase 1.
- Management Plane stores dashboard links and query templates.
- Management Plane does not store raw Prometheus time series.
- Cluster Agent does not forward full metrics or logs.
- Grafana is opened through external deep links, not iframe embedding.

### Serving Endpoint Decisions

- Phase 1 uses cluster-local serving URLs.
- Management Plane stores endpoint ownership and readiness in Endpoint Registry.
- Serving traffic does not flow through the Management Plane.
- Global Inference Gateway is deferred.

### Model Artifact Decisions

- Phase 1 supports only cached Model Artifacts.
- Operators prepare model cache contents before Serving Application creation.
- Management Plane records artifact metadata but does not download, distribute, synchronize, or repair model weights.

## Testing Decisions

Tests should verify external behavior and state transitions rather than implementation details.

### Management Plane Tests

- Project authorization tests verify users cannot deploy outside granted clusters or Accelerator Pools.
- Deployment validation tests cover missing artifacts, unsupported backend/topology combinations, unavailable clusters, and unauthorized resources.
- Manifest rendering tests snapshot generated intent-to-Dynamo-resource output for representative DeepSeek V4 Flash/Pro cases.
- Task orchestration tests cover leases, retries, idempotency, task completion, and failure transitions.
- Serving Application state machine tests cover valid transitions, invalid transitions, failure recovery, retirement, and audit metadata.
- Endpoint Registry tests cover endpoint creation, readiness updates, ownership, and cleanup on retirement.
- Observability entry tests verify dashboard links and selected summary query generation without raw metric ingestion.

### Cluster Agent Tests

- Agent registration tests verify identity, heartbeat, and capability reporting.
- Task polling tests cover lease renewal, reconnect, duplicate task handling, and idempotent result reporting.
- Whitelist enforcement tests verify arbitrary kubectl and arbitrary YAML apply are rejected.
- Kubernetes Adapter tests use fake clients or test clusters to verify dry-run, apply, watch, and event mapping behavior.
- Dynamo Adapter tests verify DGDR/DGD/DCD status maps to Serving Application phases.
- Safe Redeploy Controller tests verify old GPU workloads are deleted and gone before new resources are applied.
- Diagnostics Provider tests verify bounded logs/events and redaction rules.

### Integration Tests

- Single-cluster happy path: create Project, register cluster, register cached Model Artifact, create Serving Application, preview diff, approve, apply, reach Ready, expose endpoint.
- Failure path: invalid cached artifact or missing PVC fails validation before task execution.
- Agent disconnect path: task lease expires and is retried safely.
- Redeploy path: material change triggers delete-before-apply before new resources are applied.
- Retire path: Serving Application enters Retiring, cluster resources are deleted, endpoint is removed, audit remains.

## Out of Scope

- Global Inference Gateway.
- Unified global serving URL.
- Cross-cluster online traffic routing, failover, canary, shadow, or traffic shifting.
- Automatic model download, distribution, synchronization, or cache repair.
- Global metrics backend such as Thanos, VictoriaMetrics, or Mimir.
- iframe embedding of Grafana dashboards.
- Full log aggregation platform.
- Raw Prometheus time-series storage in the Management Plane.
- Arbitrary kubectl execution through Cluster Agent.
- Arbitrary user-supplied YAML apply through Cluster Agent.
- Cross-cluster automatic scheduling.
- Multi-tenant SaaS billing.
- Advanced accelerator vendor abstraction beyond the Phase 1 compatibility metadata needed for placement validation.

## Further Notes

Phase 1 should prioritize a complete DeepSeek V4 Flash/Pro deployment loop over breadth. The smallest useful version is:

1. Register one Inference Cluster through Cluster Agent polling.
2. Register one cached DeepSeek V4 Model Artifact.
3. Create one Project and grant cluster/resource access.
4. Create one Serving Application.
5. Preview generated Dynamo deployment diff.
6. Apply through the Cluster Agent.
7. Watch lifecycle status until Ready or Failed.
8. Show the cluster-local endpoint and Grafana dashboard link.
9. Retire the Serving Application safely.

The highest-risk seams are task idempotency, local RBAC design, delete-before-apply correctness, and the compatibility matrix between model artifact, backend, topology, and accelerator pool.

Recommended implementation order:

1. Management API skeleton and persistence for Project, Inference Cluster, Cluster Agent, and task records.
2. Cluster Agent registration, heartbeat, polling, leases, and no-op task completion.
3. Platform-generated manifest preview and server-side dry-run through the Agent.
4. Apply and status watch for one DeepSeek V4 Serving Application path.
5. Safe delete-before-apply redeploy and retirement.
6. Minimal Web Console pages that call the stable API.
