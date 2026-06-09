# Accelerator Pool Inventory Summaries

Status: ready-for-agent
Type: AFK

## Parent

.scratch/phase-2-accelerator-inventory/PRD.md

## What to build

Connect observed Accelerator Inventory to operator-defined Accelerator Pools without letting discovery create governance boundaries. The Management Plane should summarize observed capacity behind each pool and show whether pool definitions match real nodes, while preserving explicit Project access control and operator ownership of pool boundaries.

The completed slice should let an operator inspect an Accelerator Pool and understand which observed nodes and accelerator facts are behind it, without automatically creating pools or granting access from discovery.

## Acceptance criteria

- [ ] Accelerator Pool summaries are derived from observed inventory and explicit operator-defined pool rules or mappings.
- [ ] Inventory discovery does not automatically create Accelerator Pools, grant Project access, or override pool ownership boundaries.
- [ ] Pool summaries include node count, accelerator model summary, accelerator count, memory summary, relevant labels/taints, and freshness status.
- [ ] Invalid, empty, stale, or partially observed pool mappings are visible through API and Web Console warnings.
- [ ] Serving Application and Project authorization semantics remain unchanged by inventory discovery.
- [ ] Tests cover pool summary derivation, stale inventory, empty pools, partial node observations, and no automatic access grants.

## Blocked by

- .scratch/phase-2-accelerator-inventory/issues/02-read-only-kubernetes-node-inventory.md
- .scratch/phase-2-accelerator-inventory/issues/05-inventory-freshness-revision-history.md
