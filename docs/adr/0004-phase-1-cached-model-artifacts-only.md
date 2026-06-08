# Support only cached model artifacts in Phase 1

Phase 1 will deploy only Model Artifacts that already exist in the target Inference Cluster's model cache or shared storage. Large models such as DeepSeek V4 are expensive to download, often run in offline or controlled-network clusters, and require cluster-specific capacity and credential handling that would expand the first deployment loop too much.

## Consequences

- The Management Plane records artifact identity, revision, cache path, PVC reference, quantization, and compatibility metadata.
- Automatic download, distribution, synchronization, and repair of model weights are out of scope for Phase 1.
- Operators remain responsible for preparing model cache contents before creating a Serving Application.
