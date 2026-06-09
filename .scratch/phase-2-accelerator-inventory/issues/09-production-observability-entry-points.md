# Production Observability Entry Points

Status: ready-for-agent
Type: AFK

## Parent

.scratch/phase-2-accelerator-inventory/PRD.md

## What to build

Add production observability entry points around Accelerator Inventory and Tuning Records without turning the Management Plane into a raw metrics or log ingestion system. The Management Plane should expose selected Prometheus summaries, global Grafana links, log aggregation links, alert entries, and telemetry coverage warnings for fleet, cluster, model, deployment, and accelerator inventory views.

The completed slice should give SREs and operators a single product entry point for inventory-aware observability while preserving cluster-local drill-down and keeping serving traffic outside the Management Plane.

## Acceptance criteria

- [ ] Observability summaries can include selected Prometheus query results or graceful query failures without storing raw time series.
- [ ] Global Grafana dashboard links can be represented for fleet, cluster, model, deployment, and accelerator inventory views.
- [ ] Log aggregation entry points can be represented for Serving Applications and clusters without forwarding full logs through the Cluster Agent.
- [ ] Alert entries can represent inventory freshness, missing telemetry, compatibility risks, deployment health, and accelerator health warnings.
- [ ] Telemetry coverage warnings distinguish missing DCGM, missing Prometheus data, missing global dashboard link, and stale inventory.
- [ ] The Web Console exposes observability links and warnings from cluster, Serving Application, inventory, and tuning contexts.
- [ ] The implementation does not introduce Global Inference Gateway behavior, serving request routing, or Management Plane raw time-series storage.
- [ ] Tests cover selected Prometheus summary execution/failure, dashboard link rendering, alert entry exposure, telemetry coverage warnings, and no raw time-series persistence.

## Blocked by

- .scratch/phase-2-accelerator-inventory/issues/05-inventory-freshness-revision-history.md
- .scratch/phase-2-accelerator-inventory/issues/08-tuning-records-inventory-context.md
