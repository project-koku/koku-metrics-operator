# Generate Downstream Release

This document outlines the process for generating the downstream version of the operator. The downstream version transforms all `koku` references to `costmanagement` for Red Hat's internal distribution.

## Prerequisites

* **yq** - YAML processor ([installation guide](https://github.com/mikefarah/yq))
* **operator-sdk** - Operator SDK CLI tool


## What the Downstream Target Does

The `make downstream` command performs the following transformations:
- Renames `koku` → `costmanagement` and `Koku` → `CostManagement` in:
  - `api/v1beta1/` - API type definitions
  - `config/` - Kubernetes manifests and kustomize configurations
  - `docs/csv-description.md` - ClusterServiceVersion description
  - `internal/` - RBAC kubebuilder annotations only
- Updates API group names, kinds, and project configuration
- Regenerates bundle manifests with downstream-specific metadata
- Appends OpenShift-specific labels to `bundle.Dockerfile`
- Sets `isCertified` flag to `true` for Red Hat certification

## Steps

### 1. Create a Feature Branch

Checkout a new branch from `main`:
```bash
git checkout main
git pull origin main
git checkout -b downstream-updates-vX.Y.Z
```

### 2. Generate Downstream Code Changes

Run the downstream target:
```bash
make downstream
```

This will:
- Remove upstream-specific directories (`koku-metrics-operator/`, `config/scorecard/`)
- Apply name transformations to relevant files
- Regenerate the bundle with downstream configuration
- Update the `bundle.Dockerfile` with OpenShift labels

### 3. Review Changes

Verify the transformations were applied correctly:
```bash
git status
git diff
```

Key things to check:
- References changed from `koku` to `costmanagement` in `api/v1beta1/` and `config/`
- `docs/csv-description.md` contains `costmanagement` references
- `bundle.Dockerfile` contains `COPY bundle/manifests`
- Bundle CSV has correct downstream metadata

### 4. Stage and Commit the Generated Changes

```bash
git add .
git commit -m "Generate downstream code changes for vX.Y.Z"
```

### 5. Merge with origin/downstream and Resolve Conflicts

Merge your changes with the existing downstream branch:

```bash
git fetch origin
git merge origin/downstream
```

**Resolving merge conflicts (if any):**

Common conflicts typically occur in:
- `vendor/` - Accept changes from the current branch (committed in step 4)
- `internal/` - Accept changes from the current branch (committed in step 4)
- `go.mod` and `go.sum` - Accept changes from the current branch (committed in step 4), then run `go mod tidy` if needed
- Bundle manifests - Review carefully. Keep the existing downstream `containerImage` reference, as it will be automatically updated by a Konflux nudge PR after these changes are merged.

After resolving all conflicts:
```bash
git add .
git commit -m "Merge with downstream and resolve conflicts"
```

### 6. Push Changes

```bash
git push origin downstream-updates-vX.Y.Z
```

### 7. Open Pull Request

Open a PR against the **`downstream`** branch to merge the downstream code changes.

