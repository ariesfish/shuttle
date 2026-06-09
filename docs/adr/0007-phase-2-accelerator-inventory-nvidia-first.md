# Build Phase 2 Accelerator Inventory NVIDIA-first

Phase 2 will add Accelerator Inventory as a first-class Cluster Agent report so production observability, tuning records, and compatibility checks can use observed accelerator facts instead of hand-written assumptions. The first slice will be NVIDIA-first because the current target clusters and large-model deployments depend on NVIDIA GPUs, NVLink, InfiniBand/RDMA, CUDA, drivers, and DCGM signals; vendor-neutral resource schemas and advanced scheduling remain Phase 4 concerns so Phase 2 does not overfit a premature abstraction or block on unsupported accelerator vendors.

## Consequences

- Cluster Agent capability reporting must grow from a flat capabilities map into an inventory contract for node-level accelerator facts.
- Phase 2 compatibility checks may depend on observed NVIDIA inventory, but must keep the domain language as Accelerator Inventory and Accelerator Pool rather than GPU node info.
- Phase 4 owns vendor-neutral schema expansion and automated placement across heterogeneous accelerator vendors.
