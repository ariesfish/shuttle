# Read-only Kubernetes Node Inventory

Status: ready-for-agent
Type: AFK

## Parent

.scratch/phase-2-accelerator-inventory/PRD.md

## What to build

Extend the Accelerator Inventory smoke path with read-only Kubernetes node discovery. The Cluster Agent should collect node names, placement-relevant labels, taints, capacity, allocatable resources, and accelerator resource names through bounded read-only probes, then report them through the existing inventory contract.

The completed slice should let an operator verify which Kubernetes nodes and accelerator resource capacities the Management Plane sees, without granting the Management Plane direct kube-apiserver access.

## Acceptance criteria

- [ ] The Cluster Agent has a read-only Kubernetes inventory probe for node labels, taints, capacity, allocatable resources, and accelerator resource names.
- [ ] Inventory reports include per-node Kubernetes facts without exposing secrets, environment dumps, kubeconfigs, or arbitrary command output.
- [ ] Probe failures, missing permissions, and API timeouts produce bounded warnings while preserving any successfully collected inventory facts.
- [ ] Agent deployment RBAC and deployment documentation describe the minimum read-only permissions required for node inventory.
- [ ] The Management API and Web Console show Kubernetes node inventory fields from a reported snapshot.
- [ ] Local development remains possible with fake inventory when no Kubernetes cluster is available.
- [ ] Tests cover successful node inventory mapping, partial probe failure, timeout behavior, redaction, and missing-permission behavior.

## Blocked by

- .scratch/phase-2-accelerator-inventory/issues/01-inventory-contract-smoke-path.md
