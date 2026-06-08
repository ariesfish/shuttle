# Build Phase 1 API-first and Agent-first

Phase 1 will stabilize the Management API and Cluster Agent control loop before investing heavily in Web Console polish. The platform's highest-risk assumptions are agent registration, polling, task leases, local RBAC, idempotent task execution, Dynamo status mapping, and delete-before-apply behavior, so those seams must be validated before UI breadth.

## Consequences

- Web Console work should consume the same Management API and stay minimal until the deployment loop is reliable.
- The first implementation slice should prove cluster registration, task polling, no-op execution, and status reporting before full deployment actions.
- UI mockups must not introduce product flows that the API and Cluster Agent cannot execute.
