# Inventory-backed Compatibility Checks

Status: ready-for-agent
Type: AFK

## Parent

.scratch/phase-2-accelerator-inventory/PRD.md

## What to build

Use observed Accelerator Inventory during Serving Application validation. Compatibility checks should compare deployment intent, Serving Recipe metadata, Model Artifact metadata, selected Inference Cluster, selected Accelerator Pool, inventory freshness, accelerator model, memory, GPU count, GPUs per node, connectivity facts, and runtime facts. Results should be actionable in both API responses and the Web Console.

The completed slice should fail incompatible deployments before preview/apply and explain whether the issue is missing inventory, stale inventory, insufficient memory, unsupported accelerator model, insufficient GPUs per node, missing RDMA/NVLink, or a runtime compatibility mismatch.

## Acceptance criteria

- [ ] Serving Application validation uses the latest relevant Accelerator Inventory revision when available.
- [ ] Validation records or returns the inventory revision used for the decision.
- [ ] Missing inventory, stale inventory, incompatible inventory, and partial-warning inventory are represented as distinct outcomes.
- [ ] Compatibility checks cover accelerator model, memory size, GPU count, GPUs per node, connectivity assumptions, and runtime facts when represented by the recipe or deployment intent.
- [ ] Validation errors are user-facing and name the specific observed fact that failed compatibility.
- [ ] The Web Console creation flow shows inventory-based target hints and actionable compatibility failures.
- [ ] Approval or review views expose the observed accelerator facts used for risky deployments.
- [ ] Audit records include enough inventory revision context to explain why validation allowed or rejected a deployment.
- [ ] Tests cover missing, stale, partial, and incompatible inventory plus at least one compatible DeepSeek V4 Flash/Pro path.

## Blocked by

- .scratch/phase-2-accelerator-inventory/issues/03-nvidia-accelerator-fact-probes.md
- .scratch/phase-2-accelerator-inventory/issues/04-connectivity-inventory-nvlink-rdma.md
- .scratch/phase-2-accelerator-inventory/issues/05-inventory-freshness-revision-history.md
- .scratch/phase-2-accelerator-inventory/issues/06-accelerator-pool-inventory-summaries.md
