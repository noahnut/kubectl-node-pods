# kubectl-node-pods

A kubectl plugin that shows node-level pod distribution and resource request pressure (CPU/Memory) to help you identify scheduling imbalance.

## Output Example

```bash
NODE    STATUS  ROLES          VERSION  PODS  CPU_REQ/ALLOC             MEM_REQ/ALLOC                 VOLUMES  PRESSURE
node-1  Ready   control-plane  v1.29.1  32    6240m/8000m (78.0%)       10420Mi/16384Mi (63.6%)      12       medium
node-2  Ready   worker         v1.29.1  28    4320m/8000m (54.0%)       9800Mi/16384Mi (59.8%)       9        low
node-3  Ready   worker         v1.29.1  25    7100m/8000m (88.8%)       14010Mi/16384Mi (85.5%)      11       medium

TOTAL   -       -              -        85    17660m/24000m (73.6%)     34230Mi/49152Mi (69.6%)      all namespaces
```

## Installation

### Via krew

```bash
kubectl krew install node-pods
```

> Note: this works after your manifest is published to a krew index.

### Manual

```bash
make install
```

This copies the binary to `$GOPATH/bin`. Make sure that directory is in your `PATH`.

### Build from source

```bash
make build
```

## Usage

```bash
# Show pod/resource balance for all nodes (all namespaces)
kubectl node-pods

# Filter by namespace
kubectl node-pods -n kube-system

# Use a specific kubeconfig
kubectl node-pods --kubeconfig /path/to/config

# Use a specific context
kubectl node-pods --context my-cluster
```

## Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--kubeconfig` | | Path to the kubeconfig file |
| `--context` | | Kubernetes context to use |
| `--namespace` | `-n` | Filter pods by namespace (default: all) |

## How balance is evaluated

- `PODS`: Number of scheduled pods on each node.
- `CPU_REQ/ALLOC`: Sum of pod CPU requests divided by node allocatable CPU.
- `MEM_REQ/ALLOC`: Sum of pod Memory requests divided by node allocatable Memory.
- `PRESSURE`: `low` / `medium` / `high` based on the higher of CPU or Memory request ratio:
  - `< 70%`: low
  - `70% - 89.9%`: medium
  - `>= 90%`: high

## Cross-compile releases

```bash
make release
```

Outputs tarballs for darwin/linux/windows (amd64 & arm64) into `dist/`.

## CI/CD release flow

This project uses `.github/workflows/release.yml`:

1. Trigger on tag push (for example `v0.1.1`)
2. Run GoReleaser with `.goreleaser.yaml`
3. Build multi-platform archives (including `LICENSE`) and publish release assets
4. Run `krew-release-bot` to render `.krew.yaml` and open/update PR in krew-index

Tag and release:

```bash
git tag v0.1.1
git push origin v0.1.1
```

### Required GitHub secret

- `KREW_TOKEN`: Personal access token used by `krew-release-bot` to push to your fork and create PRs.

## Krew manifest template

- Template: `.krew.yaml` (Go template syntax)
- Function: `addURIAndSha` is provided by `krew-release-bot` to auto-fill `uri` and `sha256`

### Why this design

- Avoid manual `sha256` maintenance
- Keep release artifacts and krew-index PR generation in one flow
- Reduce YAML update mistakes for each new version
