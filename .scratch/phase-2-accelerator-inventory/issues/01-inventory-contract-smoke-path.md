# Inventory Contract Smoke Path

Status: ready-for-agent
Type: AFK

## Parent

.scratch/phase-2-accelerator-inventory/PRD.md

## What to build

Build the first demoable Accelerator Inventory path from Cluster Agent to Management Plane to Web Console using fake inventory data. This slice establishes the versioned inventory contract, a report/read API, latest snapshot storage, heartbeat inventory revision linkage, and a minimal cluster inventory view that works without GPU nodes.

The completed slice should let a developer run the local Management API, run a fake Cluster Agent, report one Accelerator Inventory snapshot, fetch it through the API, and see freshness/status in the Web Console.

## Acceptance criteria

- [ ] A versioned Accelerator Inventory contract represents schema version, cluster identity, agent identity, revision, observed timestamp, per-node facts, vendor-specific facts, probe warnings/errors, and collection metadata.
- [ ] The Cluster Agent can produce and report fake Accelerator Inventory without requiring Kubernetes, NVIDIA devices, DCGM, or privileged access.
- [ ] The Management Plane accepts inventory reports through an authenticated Agent path and stores the latest snapshot per Inference Cluster.
- [ ] The Management API exposes the latest inventory and a freshness state for an Inference Cluster.
- [ ] Cluster Agent heartbeat or cluster health output references the latest inventory revision without embedding the full inventory payload.
- [ ] The Web Console shows a minimal cluster inventory section with revision, observed time, freshness, node count, and probe status from fake data.
- [ ] Older Phase 1 agents without inventory support still register, heartbeat, lease tasks, and appear as inventory missing or unsupported.
- [ ] Tests cover fake inventory reporting, API readback, idempotent latest snapshot behavior, backward compatibility, and Web Console/API parsing.

## Blocked by

None - can start immediately
