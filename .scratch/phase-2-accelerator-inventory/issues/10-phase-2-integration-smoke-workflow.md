# Phase 2 Integration Smoke Workflow

Status: ready-for-agent
Type: AFK

## Parent

.scratch/phase-2-accelerator-inventory/PRD.md

## What to build

Create a complete local smoke workflow that demonstrates the Phase 2 Accelerator Inventory path end to end: fake or fixture inventory is reported by the Cluster Agent, stored by the Management Plane, summarized through Accelerator Pools, used by Serving Application compatibility checks, referenced by a Tuning Record, and shown with observability entry points in the Web Console.

The completed slice should make the Phase 2 vertical thread verifiable by future agents and humans without requiring live GPU hardware.

## Acceptance criteria

- [ ] A documented local workflow starts the Management API, Web Console, and fake or fixture-backed Cluster Agent.
- [ ] The workflow reports representative Accelerator Inventory and verifies API readback.
- [ ] The workflow shows Accelerator Pool summary derived from the reported inventory.
- [ ] The workflow verifies at least one compatible Serving Application validation path and one incompatible path with an actionable reason.
- [ ] The workflow creates or reads a Tuning Record tied to the inventory revision.
- [ ] The workflow displays observability entry points and telemetry coverage warnings in the Web Console or API.
- [ ] Representative fixtures are included for at least one NVIDIA large-model cluster shape and one partial inventory shape.
- [ ] Regression tests or smoke scripts cover the workflow enough for AFK agents to validate future changes.
- [ ] Documentation explains what is fake, what requires real Kubernetes, and what requires real NVIDIA/DCGM/RDMA visibility.

## Blocked by

- .scratch/phase-2-accelerator-inventory/issues/07-inventory-backed-compatibility-checks.md
- .scratch/phase-2-accelerator-inventory/issues/08-tuning-records-inventory-context.md
- .scratch/phase-2-accelerator-inventory/issues/09-production-observability-entry-points.md
