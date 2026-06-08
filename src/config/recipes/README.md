# Serving Recipes

Serving Recipes are platform-managed serving configurations. They are reviewed and versioned with the Management API image, then exposed through `GET /v1/recipes` for Web Console selection.

## File layout

Use one YAML file per recipe:

```text
src/config/recipes/<recipe-id>.yaml
```

`metadata.id` is the stable API value used as `ServingApplication.runtime.recipe`.

## Support status

`spec.support.status` controls compatibility gates:

- `supported`: creation and deployment are allowed without extra warning.
- `experimental`: creation and deployment are allowed, but API/Web must show the warning.
- `blocked`: creation is rejected with the recipe warning or reason.

## Template binding

`spec.template.path` points to the Kubernetes manifest template relative to the repository root. `spec.template.renderer` selects the renderer implementation. The current renderer is `string-replacement-v1`.

## Validation

Run recipe and renderer checks from `src/`:

```bash
go test ./internal/management
```

Render snapshots are stored under `src/internal/management/testdata/render-snapshots/`. When an intentional recipe/template change alters rendered YAML, update snapshots with:

```bash
UPDATE_RENDER_SNAPSHOTS=1 go test ./internal/management -run TestRenderRecipeSnapshots
```
