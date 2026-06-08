# Use Git-managed Serving Recipes with ConfigMap overrides

Serving Recipes are platform release configuration, not tenant-managed runtime data. We will ship built-in recipes with the Management API image, allow cluster-specific additions or overrides through Helm-managed ConfigMaps, and apply updates through normal rollout rather than PVC mutation or hot reload. This keeps recipe compatibility metadata, renderer bindings, and deployment templates versioned, reviewable, and rollbackable while still giving cluster operators a controlled override path.

## Considered Options

- Web Console CRUD: flexible for operators, but turns deployment templates and compatibility gates into mutable production data that needs its own publishing, validation, rollback, and authorization model.
- PVC-mounted recipes: easy to mutate in-cluster, but weakens version traceability and makes multi-replica Management API behavior harder to reason about.
- Hot reload: reduces restart friction, but risks API instances serving different recipe catalogs during updates.

## Consequences

- The Web Console lists and selects Serving Recipes but does not author them.
- Recipe updates use image releases or Helm ConfigMap changes plus Deployment rollout.
- A recipe's `template.path` is the reviewed binding between platform compatibility metadata and the Kubernetes manifest renderer.
