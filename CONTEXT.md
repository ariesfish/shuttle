# Inference Platform

This context defines the domain language for managing large-model inference deployments across heterogeneous accelerator clusters.

## Language

**Management Plane**:
The product-facing control environment where users define, review, and govern inference services.
_Avoid_: Admin backend, control backend

**Inference Cluster**:
A Kubernetes cluster that runs model-serving workloads on accelerator resources.
_Avoid_: GPU cluster, compute cluster

**Cluster Agent**:
A trusted in-cluster representative of the Management Plane for a single Inference Cluster.
_Avoid_: Backend service, deployment service

**Project**:
A product boundary that groups users, permissions, Serving Applications, and allowed resource access.
_Avoid_: Kubernetes namespace, tenant

**Serving Application**:
A user-facing model service with deployment intent, runtime status, and operational history.
_Avoid_: Helm release, Kubernetes deployment, pod group

**Model Artifact**:
A concrete model checkpoint or local snapshot that can be served by an inference backend.
_Avoid_: Model name, Hugging Face repo

**Serving Topology**:
The runtime shape of a Serving Application, such as aggregated serving or prefill-decode disaggregation.
_Avoid_: Architecture, recipe

**Serving Recipe**:
A platform-managed serving configuration that binds supported model/runtime/topology combinations to compatibility status and deployment rendering metadata.
_Avoid_: User template, deployment preset, topology

**Accelerator Pool**:
A schedulable group of accelerator resources with shared operational characteristics.
_Avoid_: GPU pool, node group

**Optimization Profile**:
A record of performance targets, profiling results, and tuning observations for a Serving Application.
_Avoid_: Benchmark result, planner config

## Relationships

- A **Management Plane** manages one or more **Projects**.
- A **Project** owns zero or more **Serving Applications**.
- A **Project** may be granted access to one or more **Inference Clusters** and **Accelerator Pools**.
- Each **Inference Cluster** has one **Cluster Agent** trusted by the **Management Plane**.
- A **Serving Application** runs on exactly one **Inference Cluster** at a time.
- A **Serving Application** serves one **Model Artifact** using one **Serving Topology**.
- A **Serving Application** is created from one **Serving Recipe**.
- A **Serving Application** consumes capacity from one or more **Accelerator Pools**.
- A **Serving Application** may accumulate many **Optimization Profiles** over its lifetime.

## Example dialogue

> **Dev:** "Should the Management Plane directly create pods for DeepSeek V4 Pro?"
> **Domain expert:** "No — the Management Plane owns the Serving Application intent, while the Cluster Agent applies that intent inside the target Inference Cluster."

## Flagged ambiguities

- "管理后台" was used to mean both **Management Plane** and an in-cluster service — resolved: these are distinct concepts.
- "模型" may mean a catalog entry or a concrete checkpoint — resolved: deployable checkpoints are **Model Artifacts**.
- "GPU 集群" is too NVIDIA-specific for the target domain — resolved: use **Inference Cluster** and **Accelerator Pool** for vendor-neutral language.
- "namespace" may refer to Kubernetes placement or product ownership — resolved: use **Project** for the product boundary and Kubernetes namespace only as a cluster-side mapping.
