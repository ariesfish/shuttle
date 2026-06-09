# Inventory Freshness and Revision History

Status: ready-for-agent
Type: AFK

## Parent

.scratch/phase-2-accelerator-inventory/PRD.md

## What to build

Make Accelerator Inventory trustworthy over time by adding freshness, revision history, idempotent updates, stale/missing/unsupported states, heartbeat linkage, and attribution. The Management Plane should retain the latest inventory plus bounded recent revisions and make inventory drift or staleness visible through API and Web Console.

The completed slice should let operators tell whether inventory is current, when it changed, which Cluster Agent reported it, and whether repeated reports are creating noisy history.

## Acceptance criteria

- [ ] Inventory reports generate stable revisions or content hashes so repeated identical reports are idempotent.
- [ ] The Management Plane stores the latest inventory snapshot and bounded recent revisions per Inference Cluster.
- [ ] Inventory freshness states include fresh, stale, missing, and unsupported.
- [ ] Cluster Agent heartbeat or cluster health output includes the latest inventory revision and freshness status.
- [ ] Inventory reports are attributed to Cluster Agent identity and observed/report timestamps.
- [ ] Inventory changes are auditable enough to explain hardware drift or validation decisions.
- [ ] The Management API exposes latest inventory, freshness, and bounded revision metadata.
- [ ] The Web Console shows stale/missing/unsupported warnings and recent inventory revision metadata.
- [ ] Tests cover idempotent update behavior, bounded history, stale detection, unsupported agents, attribution, and API readback.

## Blocked by

- .scratch/phase-2-accelerator-inventory/issues/01-inventory-contract-smoke-path.md
