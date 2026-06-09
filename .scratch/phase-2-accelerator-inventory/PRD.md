# Phase 2 PRD: Accelerator Inventory, Production Observability, and Tuning

Status: ready-for-agent

## Problem Statement

Operators need a reliable way to understand whether an Inference Cluster can safely run and tune large-model Serving Applications before a deployment consumes scarce accelerator capacity. Phase 1 proves the Management Plane and Cluster Agent control loop, but the Cluster Agent still reports only a flat capability map and deployment status; it does not expose the observed accelerator hardware, node topology, or connectivity facts that determine whether DeepSeek V4 Flash/Pro and similar workloads can run well.

The missing facts include NVIDIA GPU model, memory size, GPU count, NVLink presence, InfiniBand/RDMA capability, driver and CUDA health, DCGM availability, Kubernetes node labels and taints, and the relationship between those facts and Accelerator Pools. Without an Accelerator Inventory, compatibility checks, tuning records, observability dashboards, and operational decisions rely on hand-written assumptions that drift from the real cluster.

## Solution

Build Phase 2 around **Accelerator Inventory** as the entry capability for production observability and tuning. The Cluster Agent should observe node-level accelerator facts from inside each Inference Cluster and report them to the Management Plane through a stable inventory contract. The Management Plane should store inventory snapshots, expose them through product APIs and the Web Console, connect them to Accelerator Pools, and use them as input for compatibility checks, tuning records, dashboards, alerts, and deployment readiness decisions.

The first implementation slice is NVIDIA-first, per ADR 0007. Phase 2 should support the hardware and connectivity signals needed by the current target clusters without pretending to solve all accelerator vendors. Domain language must remain **Accelerator Inventory**, **Accelerator Pool**, and **Inference Cluster** so Phase 4 can add vendor-neutral schemas and advanced scheduling without renaming the product model.

Phase 2 should also add the production observability backbone around this inventory: optional global metrics, global Grafana views, log aggregation entry points, alert aggregation, and Tuning Records that preserve profiling context and recommendations.

## User Stories

1. As a platform operator, I want each Inference Cluster to report an Accelerator Inventory, so that I can see the real hardware facts behind deployment decisions.
2. As a platform operator, I want Accelerator Inventory to include node-level accelerator counts, so that I can estimate available serving capacity.
3. As a platform operator, I want Accelerator Inventory to include NVIDIA GPU model names, so that I can distinguish H100, H200, H800, A100, and other supported systems.
4. As a platform operator, I want Accelerator Inventory to include per-device memory size, so that large-model compatibility checks can fail early when memory is insufficient.
5. As a platform operator, I want Accelerator Inventory to include GPU count per node, so that tensor parallelism and multi-node serving recipes can be evaluated correctly.
6. As a platform operator, I want Accelerator Inventory to include NVLink presence when observable, so that high-bandwidth same-node assumptions are visible.
7. As a platform operator, I want Accelerator Inventory to include InfiniBand or RDMA capability when observable, so that disaggregated serving and KV transfer assumptions are visible.
8. As a platform operator, I want Accelerator Inventory to include NVIDIA driver version, so that runtime compatibility issues can be diagnosed before deployment.
9. As a platform operator, I want Accelerator Inventory to include CUDA runtime or node-level CUDA compatibility signals, so that backend/runtime constraints can be checked.
10. As a platform operator, I want Accelerator Inventory to include DCGM availability and health signals, so that GPU telemetry dashboards can explain missing metrics.
11. As a platform operator, I want Accelerator Inventory to include Kubernetes node labels, so that placement and Accelerator Pool membership can be audited.
12. As a platform operator, I want Accelerator Inventory to include Kubernetes taints, so that unschedulable or dedicated accelerator nodes are visible.
13. As a platform operator, I want Accelerator Inventory to include allocatable accelerator resources, so that I can compare observed capacity with Kubernetes scheduler capacity.
14. As a platform operator, I want Accelerator Inventory to include currently allocated accelerator resources when feasible, so that I can see capacity pressure.
15. As a platform operator, I want Accelerator Inventory to record observation time, so that stale hardware facts are obvious.
16. As a platform operator, I want the Management Plane to show whether an inventory snapshot is fresh, stale, or missing, so that I can trust or distrust placement decisions.
17. As a platform operator, I want each Cluster Agent heartbeat to indicate the latest inventory revision it knows about, so that inventory freshness is tied to Agent health.
18. As a cluster administrator, I want inventory collection to use local RBAC with read-only Kubernetes permissions, so that the Management Plane still does not hold a broad kubeconfig.
19. As a security reviewer, I want inventory collection to avoid arbitrary command execution through the Agent, so that the Cluster Agent remains a whitelist executor.
20. As a security reviewer, I want inventory data to exclude secrets and sensitive environment values, so that reporting hardware facts does not leak credentials.
21. As a model operator, I want Serving Application validation to compare deployment intent with observed Accelerator Inventory, so that unsupported cluster choices fail before apply.
22. As a model operator, I want validation errors to explain which observed hardware fact failed compatibility, so that I can pick a different cluster, pool, recipe, or Model Artifact.
23. As a model operator, I want the Web Console creation flow to show relevant inventory facts for candidate Inference Clusters, so that I can choose a viable deployment target.
24. As a model operator, I want Accelerator Pool details to show the inventory behind the pool, so that I know what capacity my Serving Application will consume.
25. As a model operator, I want the Serving Application detail page to show the hardware context used at deployment time, so that later performance results can be interpreted.
26. As a model operator, I want Tuning Records to capture the inventory snapshot used during profiling or benchmarking, so that recommendations remain explainable after hardware changes.
27. As a performance engineer, I want benchmark summaries to include GPU model, memory size, topology, and RDMA/NVLink facts, so that results are comparable across clusters.
28. As a performance engineer, I want Planner settings and recommendations to reference Accelerator Inventory, so that tuning advice is grounded in observed capacity.
29. As a performance engineer, I want inventory changes to be visible over time, so that regressions caused by node replacement or driver changes can be detected.
30. As an SRE, I want global dashboards to summarize accelerator fleet composition, so that I can see how many clusters and nodes have each accelerator type.
31. As an SRE, I want global dashboards to show accelerator health and telemetry coverage, so that missing DCGM or Prometheus data is visible.
32. As an SRE, I want global dashboards to correlate Serving Applications with accelerator types, so that incident impact can be assessed quickly.
33. As an SRE, I want alert aggregation to surface inventory-related risks, so that stale inventory, missing telemetry, or incompatible driver versions are actionable.
34. As an SRE, I want log aggregation entry points linked from Serving Application and cluster views, so that deployment failures can be investigated without full raw log forwarding through the Cluster Agent.
35. As a platform operator, I want global metrics to support fleet, cluster, model, deployment, and accelerator inventory views, so that Phase 2 observability moves beyond cluster-local links.
36. As a platform operator, I want the Management Plane to support optional Prometheus query execution for selected summaries, so that the Web Console can show live health without ingesting raw time series.
37. As a platform operator, I want the Management Plane to remain independent of serving traffic, so that global observability does not turn into a Global Inference Gateway.
38. As a platform operator, I want Accelerator Inventory to support multiple Inference Clusters, so that fleet-wide operations work across cluster boundaries.
39. As a platform operator, I want inventory collection failures to be recorded as bounded events, so that missing inventory is diagnosable.
40. As a cluster administrator, I want inventory collection to degrade gracefully when NVLink, RDMA, CUDA, or DCGM facts are not available, so that one missing probe does not break all Agent reporting.
41. As a cluster administrator, I want inventory probes to be configurable enough for clusters without DCGM or privileged device visibility, so that Phase 2 can run in restrictive environments.
42. As a platform engineer, I want the inventory contract to distinguish observed facts from operator-defined Accelerator Pool boundaries, so that runtime discovery does not override governance decisions.
43. As a platform engineer, I want Accelerator Pool membership to be derived from explicit rules or operator mappings, so that observed node facts do not create product access by accident.
44. As a platform engineer, I want inventory schema evolution to be versioned, so that older Cluster Agents can coexist with newer Management Plane releases.
45. As a platform engineer, I want the API to preserve unknown future inventory fields safely, so that Phase 4 vendor-neutral expansion does not require a disruptive migration.
46. As a platform engineer, I want inventory updates to be idempotent, so that repeated Agent reports do not create noisy history.
47. As a platform engineer, I want inventory history to be bounded or compacted, so that hardware observations do not grow without limit.
48. As a platform engineer, I want stale inventory to block or warn on deployments based on policy, so that users do not deploy against unreliable capacity facts.
49. As an approver, I want approval views to show the observed accelerator facts for risky deployments, so that I can review whether a large-model change is viable.
50. As an auditor, I want deployment audit records to include the inventory revision used for validation, so that I can explain why a deployment was allowed or rejected.
51. As an auditor, I want inventory changes to be attributable to a Cluster Agent and timestamp, so that hardware drift has an operational trail.
52. As a product user, I want terminology to say Accelerator Inventory and Accelerator Pool rather than GPU node info, so that the platform remains ready for future accelerator diversity.
53. As a future Phase 4 implementer, I want Phase 2 to avoid hard-coding the product model to NVIDIA-only names, so that vendor-neutral scheduling can be added without renaming public concepts.
54. As a future Phase 4 implementer, I want NVIDIA-specific details to live under vendor-specific inventory facts, so that non-NVIDIA accelerators can be added later.
55. As a maintainer, I want Phase 2 to keep the Cluster Agent as a deep module with narrow reporting interfaces, so that inventory probes can evolve without destabilizing deployment execution.
56. As a maintainer, I want Phase 2 to keep observability storage separate from deployment task orchestration, so that metrics/log growth does not threaten the control loop.
57. As a maintainer, I want Phase 2 to keep global metrics backend integration optional at first, so that Accelerator Inventory can ship before the full observability stack is finalized.
58. As a maintainer, I want a fake inventory mode for local development, so that Web Console and API workflows can be tested without GPU nodes.
59. As a developer, I want clear test fixtures for representative H200/H800/NVIDIA clusters, so that compatibility behavior can be verified without live hardware.
60. As a developer, I want API and Agent tests to cover partial inventory reports, so that production clusters with missing probes behave predictably.

## Implementation Decisions

### Product Boundary

Phase 2 starts with Accelerator Inventory as the first production observability and tuning capability. It is not a Phase 1 blocker and it is not the full Phase 4 heterogeneous scheduling system.

The canonical domain terms are **Accelerator Inventory**, **Accelerator Pool**, **Inference Cluster**, **Cluster Agent**, **Serving Application**, **Serving Recipe**, and **Optimization Profile**. Avoid product-facing names such as GPU node info, GPU cluster, hardware config, node group, or deployment preset.

The first inventory slice is NVIDIA-first per ADR 0007. It should collect NVIDIA GPU and connectivity facts required by current DeepSeek V4 Flash/Pro deployments while keeping public product concepts vendor-neutral.

### Deep Modules

- **Accelerator Inventory Reporter**: a Cluster Agent module that collects node-level accelerator facts through a stable interface and returns an inventory report independent of the task execution loop.
- **Inventory Probe Set**: a set of bounded, read-only probes for Kubernetes node resources, node labels, node taints, NVIDIA device facts, driver/CUDA/DCGM facts, NVLink signals, and InfiniBand/RDMA signals.
- **Inventory Contract**: a versioned API payload shared by Cluster Agent and Management Plane that distinguishes observed facts, collection errors, timestamps, and schema version.
- **Inventory Repository**: a Management Plane module that stores the latest inventory snapshot and bounded history for each Inference Cluster.
- **Accelerator Pool Service**: a Management Plane module that relates operator-defined Accelerator Pools to observed inventory without letting discovery create product access automatically.
- **Compatibility Matrix Service**: a Management Plane module that evaluates Serving Application intent, Serving Recipe metadata, Model Artifact metadata, Accelerator Pool access, and observed inventory facts.
- **Tuning Record Service**: a Management Plane module that records profiling inputs, benchmark summaries, Planner settings, recommendations, and the inventory revision used to interpret them.
- **Observability Summary Service**: a Management Plane module that queries selected Prometheus summaries and links global dashboards without ingesting raw time series.
- **Alert Entry Service**: a Management Plane module that stores or aggregates alert entry points related to clusters, deployments, accelerator telemetry, and inventory freshness.
- **Web Console Inventory Views**: UI surfaces for cluster inventory, Accelerator Pool capacity, Serving Application hardware context, tuning records, and global observability links.

### Cluster Agent Contract

The Cluster Agent should continue to register, heartbeat, poll tasks, execute whitelist tasks, and report bounded results. Phase 2 extends Agent reporting with an inventory contract instead of overloading the existing flat capabilities map.

Inventory reporting should include:

- Cluster identity and Agent identity.
- Inventory schema version.
- Inventory revision or content hash.
- Observation timestamp.
- Per-node accelerator facts.
- Per-node Kubernetes labels and taints relevant to placement.
- Per-node allocatable accelerator resources.
- NVIDIA-specific facts under a vendor-specific structure.
- Connectivity facts such as NVLink and InfiniBand/RDMA when observable.
- Probe-level warnings or errors for unavailable facts.
- Redaction-safe collection metadata.

Heartbeat should reference the latest inventory revision and freshness status. Full inventory payloads should be sent through a dedicated endpoint or bounded report path rather than repeated in every heartbeat.

The Agent must not expose arbitrary shell execution, arbitrary kubectl, or arbitrary user-provided probes. Probe behavior should be implemented as whitelisted code paths with predictable timeout and size limits.

### Management API Contract

The Management Plane should expose product resources and summaries rather than raw Kubernetes objects. Phase 2 API surface should include:

- Latest Accelerator Inventory for an Inference Cluster.
- Inventory freshness and last report status.
- Inventory history or revisions with bounded retention.
- Accelerator Pool inventory summary.
- Compatibility check result for a Serving Application intent.
- Tuning Records and associated inventory revision.
- Observability summaries and dashboard links.
- Alert entry points and inventory-related warnings.

Existing Cluster Agent registration and heartbeat APIs should remain backward-compatible for Phase 1 agents. Older agents without inventory support should appear as inventory missing rather than broken.

### Inventory Data Shape

The inventory model should distinguish the following categories:

- **Observed node facts**: node name, labels, taints, capacity, allocatable resources, accelerator resource names, and observation timestamp.
- **Observed accelerator facts**: vendor, product/model, device count, per-device memory, MIG or partitioning hints if observable, and health status if available.
- **Observed connectivity facts**: NVLink presence, peer connectivity if available, InfiniBand/RDMA capability, relevant network device hints, and probe confidence.
- **Observed runtime facts**: driver version, CUDA compatibility, container runtime device plugin signals, DCGM exporter or DCGM library availability.
- **Collection status**: successful probes, failed probes, skipped probes, warning messages, and stale data markers.
- **Pool mapping facts**: which observed nodes are candidates for which operator-defined Accelerator Pools.

NVIDIA-specific fields should be nested so the top-level product model remains Accelerator Inventory rather than GPU Inventory.

### Accelerator Pool Relationship

Accelerator Inventory describes observed capacity. Accelerator Pools remain operator-defined schedulable boundaries with governance semantics. Discovery may suggest pool membership or summarize capacity behind a pool, but it must not automatically grant Project access or silently create product ownership boundaries.

Project access to Accelerator Pools continues to be controlled by Management Plane authorization. Inventory can make invalid or stale pool definitions visible, but it should not override them.

### Compatibility and Deployment Validation

Serving Application validation should use observed inventory when available. Phase 2 compatibility checks should consider:

- Model Artifact family, variant, quantization, and cache metadata.
- Serving Recipe model/runtime/topology compatibility metadata.
- Requested Inference Cluster and Accelerator Pool.
- Observed accelerator model and memory size.
- Required GPU count and GPUs per node.
- Required topology or connectivity assumptions for disaggregated serving.
- Required driver/CUDA/DCGM conditions when represented by the recipe or runtime.
- Inventory freshness policy.

Validation should produce actionable user-facing failure reasons. Missing inventory should be distinguishable from incompatible inventory.

### Production Observability

Phase 2 should add a global metrics layer using Thanos, VictoriaMetrics, or Mimir when the deployment environment is ready. The Management Plane should use this layer for selected summaries and dashboard links, not as a raw time-series database embedded in the control loop.

Global Grafana dashboards should cover fleet, cluster, model, deployment, and accelerator inventory views. Local Grafana deep links remain useful drill-down entry points.

Log aggregation should be integrated through Loki, OpenSearch, or ClickHouse entry points. The Cluster Agent should continue to provide bounded diagnostics but should not become a full log forwarding pipeline.

Alert aggregation should surface inventory freshness, missing telemetry, compatibility risks, deployment health, and accelerator health signals.

### Tuning Records

Tuning Records are the Phase 2 durable product object for profiling, benchmark summaries, Planner settings, and recommendations. Each Tuning Record should reference:

- Serving Application.
- Inference Cluster.
- Accelerator Pool.
- Model Artifact.
- Serving Recipe.
- Inventory revision used during profiling or recommendation.
- Benchmark summary or profiling result.
- Planner settings and recommendations.
- Actor, timestamp, and reason when created by a user action.

Tuning Records should not be treated as raw benchmark artifact storage. They should preserve decision-relevant summaries and links to detailed artifacts when needed.

### Web Console

The Web Console should use the Management API and avoid direct Kubernetes access. Phase 2 UI should include:

- Cluster inventory tab or section.
- Accelerator Pool detail view with observed capacity summary.
- Serving Application creation hints based on inventory compatibility.
- Serving Application detail hardware context.
- Tuning Record list and detail views.
- Global observability links.
- Inventory freshness warnings.
- Missing telemetry warnings.

The UI should support fake inventory data for local development.

### Persistence

The current Postgres-backed persistence stores platform state as JSONB. Phase 2 can start from that adapter if needed, but inventory and tuning records should be designed as explicit domain records so they can later move to normalized tables.

Inventory history should be bounded through retention, compaction, or latest-plus-recent-revisions storage. Repeated identical inventory reports should be idempotent and should not create noisy history.

### Security and RBAC

Inventory collection should require read-only local RBAC. It should not broaden the Management Plane's cluster access model. The Cluster Agent remains the trusted in-cluster representative and must keep all collection paths whitelisted and bounded.

Inventory reports should be redaction-safe. The system should avoid reporting secrets, full environment dumps, credentials, kubeconfigs, or arbitrary device command output.

### Backward Compatibility

Phase 1 agents and clusters without inventory support should keep working. The Management Plane should show inventory as missing or unsupported, not fail cluster registration or task polling.

Inventory schema versions should be explicit. The Management Plane should tolerate unknown fields for forward compatibility and should expose unsupported required features through compatibility checks rather than transport failures.

## Testing Decisions

Tests should verify external behavior and domain contracts rather than implementation details. Probe internals may use fakes, fixtures, and command adapters, but the important tests should assert inventory reports, API behavior, compatibility results, and user-visible summaries.

### Cluster Agent Tests

- Inventory Reporter tests verify successful node and accelerator fact collection from fixture inputs.
- Inventory Reporter tests verify partial probe failure produces warnings without dropping the full report.
- Inventory Reporter tests verify probe timeouts and size limits produce bounded errors.
- Kubernetes node probe tests verify labels, taints, capacity, and allocatable resources are represented correctly.
- NVIDIA probe tests verify representative H200, H800, H100, and A100 fixtures map to normalized vendor-specific facts.
- Connectivity probe tests verify NVLink and InfiniBand/RDMA facts are optional and confidence-marked when incomplete.
- Redaction tests verify inventory reports do not include secrets, kubeconfigs, environment dumps, or arbitrary command output.
- Agent heartbeat tests verify the latest inventory revision is reported without embedding full inventory every time.
- Backward compatibility tests verify agents without inventory support still register and execute Phase 1 tasks.

Prior art: existing Cluster Agent tests cover client behavior, task execution, diagnostics planning, leases, and fake executor behavior.

### Management Plane Tests

- Inventory API tests verify latest inventory, freshness status, and bounded history behavior.
- Inventory repository tests verify idempotent update behavior for identical inventory revisions.
- Inventory repository tests verify stale inventory detection.
- Accelerator Pool tests verify pool summaries are derived from observed inventory without creating access grants.
- Compatibility Matrix tests verify missing inventory, stale inventory, insufficient memory, unsupported GPU model, insufficient GPUs per node, missing RDMA, and unsupported driver/CUDA conditions.
- Serving Application validation tests verify compatibility failures are actionable and user-facing.
- Tuning Record tests verify records reference the inventory revision used during profiling or recommendations.
- Observability Summary tests verify selected Prometheus queries can be executed or fail gracefully without ingesting raw time series.
- Alert Entry tests verify inventory-related warnings are exposed without coupling to one alert backend.
- Audit tests verify deployment validation and Tuning Records preserve inventory revision references.

Prior art: existing Management Plane tests cover auth, commands, lifecycle, observability, Prometheus summaries, recipe loading, renderer snapshots, route aliases, and store behavior.

### Web Console Tests

- API adapter tests verify inventory, pool summaries, compatibility results, tuning records, and observability summaries are parsed correctly.
- Component tests or smoke tests verify cluster inventory, pool detail, Serving Application hardware context, and stale inventory warnings render from fake data.
- Build tests verify the TypeScript app compiles with the new API types.

Prior art: existing Web Console uses React, Vite, TanStack Query, and API-driven pages for Clusters, Projects, Model Artifacts, Serving Applications, Tasks, and Audit.

### Integration Tests

- Single-cluster inventory path: Agent reports NVIDIA inventory, Management Plane stores it, Web Console/API exposes it, and cluster freshness is healthy.
- Partial inventory path: Agent cannot observe RDMA or DCGM, Management Plane stores warnings, and compatibility checks distinguish warning from hard failure.
- Validation path: Serving Application creation against an incompatible Accelerator Pool fails before preview/apply with a specific reason.
- Tuning path: benchmark or profiling summary creates a Tuning Record tied to the inventory revision used for evaluation.
- Stale inventory path: inventory becomes stale and deployment validation warns or blocks according to policy.
- Backward compatibility path: Phase 1 Agent without inventory support can still lease and complete deployment tasks while inventory is shown as unsupported.

## Out of Scope

- Global Inference Gateway.
- Unified global serving URL.
- Cross-cluster online request routing, failover, canary, shadow, or traffic shifting.
- Automatic model download, distribution, synchronization, or cache repair.
- Arbitrary Kubernetes administration through Cluster Agent.
- Arbitrary user-provided YAML apply through Cluster Agent.
- Arbitrary shell command execution through Cluster Agent probes.
- Full vendor-neutral accelerator schema for non-NVIDIA vendors.
- Automated placement recommendations across heterogeneous accelerator vendors.
- Automatic Accelerator Pool creation or Project access grants from discovery.
- Full raw Prometheus time-series storage inside the Management Plane.
- Full log forwarding through Cluster Agent.
- Multi-tenant SaaS billing.
- Replacing local Grafana drill-down dashboards.
- Production identity provider integration unless required by the specific deployment environment.

## Further Notes

Phase 2 should start with the smallest useful Accelerator Inventory slice:

1. Define the Inventory Contract and Management API endpoints.
2. Add fake inventory support for local development and Web Console work.
3. Add Cluster Agent Kubernetes node inventory collection.
4. Add NVIDIA fixture-driven probes for model, memory, GPU count, driver/CUDA/DCGM, NVLink, and InfiniBand/RDMA facts.
5. Store latest inventory plus bounded revisions in the Management Plane.
6. Show inventory and freshness in the Web Console.
7. Connect inventory to Accelerator Pool summaries.
8. Use inventory in Serving Application compatibility checks.
9. Add Tuning Records that reference inventory revisions.
10. Add global observability and alert entry points after the inventory contract is stable.

The highest-risk seams are inventory schema design, safe and portable in-cluster probing, partial-data semantics, compatibility failure clarity, and keeping NVIDIA-first implementation details from leaking into product terminology.

Phase 2 should not wait for a final global metrics backend before shipping Accelerator Inventory. Inventory is the durable input that makes the later observability and tuning layers meaningful.
