# AGENTS.md

## Scope

These instructions apply to this repository. Follow them when changing Helm charts, deployment assets, or helper scripts.

## Agent skills

### Issue tracker

Engineering workflow issues and PRDs are tracked as Local Markdown files under `.scratch/`. See `docs/agents/issue-tracker.md`.

### Triage labels

Triage-aware skills use the canonical labels `needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, and `wontfix`. See `docs/agents/triage-labels.md`.

### Domain docs

This is a single-context repo: read root `CONTEXT.md`, `docs/adr/`, `docs/platform-architecture.md`, and `docs/prd/` as relevant. See `docs/agents/domain.md`.

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

## Deployment workflow for scarce GPU clusters

The shared test cluster does not have enough GPUs to run rolling updates for large LLM deployments. Do not rely on the Dynamo operator's default rolling update behavior for `DynamoGraphDeployment` changes.

When redeploying large GPU workloads under `deployment/`:

1. Inspect the current state first:

   ```bash
   kubectl get dynamographdeployment,dynamocomponentdeployment,deploy,rs,pod,svc -n <namespace> | grep <deployment-name>
   ```

2. Delete the existing graph deployment and wait for deletion:

   ```bash
   kubectl delete dynamographdeployment -n <namespace> <deployment-name> --wait=true --timeout=120s
   ```

3. Clean up any leftover operator-managed resources before applying the new spec. Use the deployment name label when available, and verify with `kubectl get`:

   ```bash
   kubectl delete dynamocomponentdeployment -n <namespace> -l nvidia.com/dynamo-graph-deployment=<deployment-name> --ignore-not-found --wait=true --timeout=60s
   kubectl delete pod,deploy,rs,svc -n <namespace> -l nvidia.com/dynamo-graph-deployment=<deployment-name> --ignore-not-found --wait=true --timeout=60s
   kubectl get dynamographdeployment,dynamocomponentdeployment,deploy,rs,pod,svc -n <namespace> | grep <deployment-name>
   ```

   If resources remain because labels are missing or stale, delete the exact resource names before continuing.

4. Only after old pods are gone, apply the new YAML:

   ```bash
   kubectl apply -f deployment/examples/<file>.yaml
   ```

5. Watch readiness and logs:

   ```bash
   kubectl get pods -n <namespace> | grep <deployment-name>
   kubectl logs -n <namespace> <pod> -c main --previous --tail=200 --timestamps
   kubectl logs -n <namespace> <pod> -c main --tail=200 --timestamps
   ```

Additional rules:

- Do not start a new deployment while old GPU pods for the same workload are still `Running`, `Pending`, `ContainerCreating`, or `Terminating`.
- Do not switch model IDs to uncached Hugging Face repositories when `HF_HUB_OFFLINE=1`; use the existing local snapshot path unless the user confirms the new checkpoint is present on every target node.
- For H200/H800 DeepSeek-V4 SGLang experiments, consult `docs/sglang/deepseek-v4.md` and adapt the H200 guidance, but keep the actual checkpoint path aligned with what is cached on the cluster.

## Validation

Before committing chart/script changes, run targeted checks relevant to the change:

```bash
git diff --check
./scripts/package-charts.sh
```

For script-only changes, also run the changed script path directly enough to verify its argument parsing and defaults.
