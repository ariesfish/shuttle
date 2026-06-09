# Tuning Records with Inventory Context

Status: ready-for-agent
Type: AFK

## Parent

.scratch/phase-2-accelerator-inventory/PRD.md

## What to build

Add Tuning Records as the Phase 2 product object for profiling, benchmark summaries, Planner settings, recommendations, and the Accelerator Inventory revision used to interpret those results. Tuning Records should preserve decision-relevant summaries and links to detailed artifacts without becoming raw benchmark artifact storage.

The completed slice should let a user create or view a Tuning Record tied to a Serving Application, Inference Cluster, Accelerator Pool, Model Artifact, Serving Recipe, and inventory revision.

## Acceptance criteria

- [ ] A Tuning Record can reference Serving Application, Inference Cluster, Accelerator Pool, Model Artifact, Serving Recipe, and Accelerator Inventory revision.
- [ ] Tuning Records can store profiling or benchmark summaries, Planner settings, recommendations, actor, timestamp, and reason.
- [ ] Tuning Records do not store raw benchmark artifacts or raw metrics time series in the Management Plane.
- [ ] The Management API exposes Tuning Record create/list/read behavior.
- [ ] The Web Console shows Tuning Records and the hardware context used when they were created.
- [ ] Serving Application detail views can show related Tuning Records and inventory context.
- [ ] Tests cover create/list/read behavior, required references, inventory revision linkage, and rendering/parsing of summary fields.

## Blocked by

- .scratch/phase-2-accelerator-inventory/issues/05-inventory-freshness-revision-history.md
- .scratch/phase-2-accelerator-inventory/issues/07-inventory-backed-compatibility-checks.md
