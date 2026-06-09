# NVIDIA Accelerator Fact Probes

Status: ready-for-agent
Type: AFK

## Parent

.scratch/phase-2-accelerator-inventory/PRD.md

## What to build

Add NVIDIA-first accelerator fact collection to Accelerator Inventory. The Cluster Agent should report vendor-specific facts for NVIDIA GPU model, per-device memory, GPU count, driver version, CUDA compatibility signals, and DCGM availability or health. The product-facing model must remain Accelerator Inventory with NVIDIA-specific details nested under vendor-specific fields.

The completed slice should support fixture-driven local verification for representative H200, H800, H100, and A100 clusters without requiring live GPUs in tests.

## Acceptance criteria

- [ ] Inventory reports include NVIDIA-specific accelerator facts for GPU model, device count, per-device memory, driver version, CUDA compatibility signal, and DCGM availability or health when observable.
- [ ] NVIDIA-specific fields are nested under vendor-specific inventory structures and do not rename product concepts to GPU Inventory or GPU Pool.
- [ ] Missing NVIDIA probes produce bounded warnings and do not drop Kubernetes node inventory from the report.
- [ ] Probe behavior is whitelisted, timeout-bounded, and does not permit arbitrary shell commands or arbitrary user-provided probes.
- [ ] Test fixtures cover representative H200, H800, H100, and A100 inventory reports.
- [ ] Tests cover partial NVIDIA probe failure, unknown GPU models, missing DCGM, missing CUDA signal, redaction, and schema stability.
- [ ] The Web Console can render NVIDIA accelerator facts when present and a clear unsupported/missing state when absent.

## Blocked by

- .scratch/phase-2-accelerator-inventory/issues/01-inventory-contract-smoke-path.md
- .scratch/phase-2-accelerator-inventory/issues/02-read-only-kubernetes-node-inventory.md
