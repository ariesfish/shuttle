# Use delete-before-apply for scarce accelerator redeployments

Material Serving Application changes on scarce accelerator clusters will use a delete-before-apply workflow instead of relying on default rolling updates. Large-model deployments may consume all available accelerators, so rolling updates can deadlock or strand workloads when the new runtime needs GPUs before the old runtime has released them.

## Consequences

- Redeployments may include planned service interruption unless a separate traffic-shift mechanism exists.
- The Management Plane must preview diffs, request approval when required, and show clear rollout progress.
- The Cluster Agent must verify old platform-owned GPU workloads are gone before applying the new deployment.
