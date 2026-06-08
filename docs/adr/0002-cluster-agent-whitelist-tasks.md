# Keep Cluster Agent limited to whitelist tasks

The Cluster Agent will execute only platform-defined task types instead of acting as a general-purpose remote Kubernetes administration tunnel. This keeps customer or production Inference Clusters protected by local RBAC, makes audit semantics clear, and prevents the Management Plane from becoming an indirect high-privilege kubeconfig service.

## Consequences

- New cluster operations must be added as explicit task types with validation and audit behavior.
- Arbitrary `kubectl` command execution and arbitrary YAML apply are out of scope for the platform path.
- Emergency access, if needed later, must be designed as a separate break-glass mechanism rather than hidden inside normal agent tasks.
