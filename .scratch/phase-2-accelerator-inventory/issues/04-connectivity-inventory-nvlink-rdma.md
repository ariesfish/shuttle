# Connectivity Inventory for NVLink and RDMA

Status: ready-for-agent
Type: AFK

## Parent

.scratch/phase-2-accelerator-inventory/PRD.md

## What to build

Extend Accelerator Inventory with optional connectivity facts for NVLink and InfiniBand/RDMA. The Cluster Agent should report whether high-bandwidth same-node GPU connectivity and RDMA-capable networking are observable, along with confidence and warnings when facts cannot be collected.

The completed slice should make connectivity assumptions visible for disaggregated serving and KV transfer without blocking inventory reporting when a cluster lacks DCGM, NVLink visibility, or RDMA probes.

## Acceptance criteria

- [ ] Inventory reports can represent NVLink presence, peer-connectivity summary when available, InfiniBand/RDMA capability, and probe confidence.
- [ ] Missing, incomplete, or unsupported connectivity probes produce warnings rather than failing the full inventory report.
- [ ] Connectivity facts remain optional and are clearly distinguishable from hard compatibility failures.
- [ ] Probe behavior is whitelisted, timeout-bounded, read-only, and redaction-safe.
- [ ] The Management API and Web Console show connectivity facts and warnings from the latest inventory snapshot.
- [ ] Tests cover NVLink present, NVLink unavailable, RDMA present, RDMA unavailable, incomplete confidence, and partial probe failures.

## Blocked by

- .scratch/phase-2-accelerator-inventory/issues/03-nvidia-accelerator-fact-probes.md
