# AGENTS.md

## Scope

These instructions apply to this repository. Follow them when changing Helm charts, deployment assets, or helper scripts.

## Helm chart workflow

When working with charts under `charts/`, use the scripts in `scripts/` instead of ad-hoc `helm package` or `helm push` commands.

The chart scripts define the relationship between chart sources and packages:

- `scripts/package-charts.sh` discovers top-level charts at `charts/*/Chart.yaml` and packages them into `charts/packages/`.
- `scripts/push-charts.sh` pushes packaged `*.tgz` files from `charts/packages/` to the configured OCI registry.
- `scripts/package-and-push-charts.sh` runs packaging first, then pushes the resulting packages.

Common commands:

```bash
# Package all top-level charts and push to the default OCI registry.
./scripts/package-and-push-charts.sh

# Package only selected chart directories by basename under charts/.
CHARTS="dynamo-system gpu-operator" ./scripts/package-charts.sh

# Push only selected package files from charts/packages/.
PACKAGES="dynamo-platform-1.2.0.tgz gpu-operator-v26.3.2.tgz" ./scripts/push-charts.sh

# Override the target registry when explicitly required.
REMOTE_REPO=oci://registry.example.com/helm-charts ./scripts/push-charts.sh
```

Rules:

- ALWAYS package charts through `scripts/package-charts.sh` or `scripts/package-and-push-charts.sh`.
- ALWAYS push charts through `scripts/push-charts.sh` or `scripts/package-and-push-charts.sh`.
- BEFORE adding or modifying deployment YAML under `deployment/`, read the relevant documentation under `docs/` first.
- NEVER commit files under `charts/packages/` or other generated `*.tgz` chart archives.

## Validation

Before committing chart/script changes, run targeted checks relevant to the change:

```bash
git diff --check
./scripts/package-charts.sh
```

For script-only changes, also run the changed script path directly enough to verify its argument parsing and defaults.
