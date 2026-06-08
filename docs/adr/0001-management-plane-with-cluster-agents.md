# Use a management plane with cluster agents

We will run the inference platform's Management Plane outside the Inference Clusters and deploy one Cluster Agent into each managed cluster. This separates product governance, audit history, and multi-cluster orchestration from GPU-serving failure domains while keeping Kubernetes-native deployment actions local to each cluster.

## Considered Options

- Deploy the whole backend inside each Inference Cluster: fastest for a single-cluster proof of concept, but couples product availability, upgrades, and permissions to scarce accelerator workloads.
- Let an external service directly access every cluster API server: reduces in-cluster components, but increases network exposure and centralizes high-privilege Kubernetes credentials.

## Consequences

- The Cluster Agent becomes the only component that applies or watches cluster-local serving resources.
- The Management Plane stores user intent, audit records, catalog metadata, and optimization history, not the authoritative runtime state.
