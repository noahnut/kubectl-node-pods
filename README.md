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

### Local krew test (reinstall plugin)

```bash
make test-release
```

This command will:
- build a tarball for your current OS/ARCH
- generate `node-pods.yaml` from `templates/krew-plugin.yaml.tmpl`
- uninstall old `node-pods` plugin if present
- install the new plugin using local manifest/archive

Quick verify:

```bash
kubectl krew list
kubectl node-pods --help
kubectl node-pods
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
2. Build multi-platform archives
3. Generate `dist/checksums.txt`
4. Create GitHub release and upload archives + checksums
5. Generate `krew-plugin.yaml` from the shared template

Tag and release:

```bash
git tag v0.1.1
git push origin v0.1.1
```

After workflow completes, download the generated `krew-plugin.yaml` artifact and use it to update your krew index.

## Shared krew manifest template

- Template: `templates/krew-plugin.yaml.tmpl`
- Generator: `cmd/gen-manifest`
- Local `make test-release` and GitHub Actions release workflow both use the same template/generator path.

### Why this design

- Avoid duplicate manifest logic in Makefile and CI scripts
- Keep local testing and CI output consistent
- Reduce manual YAML editing errors for `sha256`, `uri`, and platform entries
