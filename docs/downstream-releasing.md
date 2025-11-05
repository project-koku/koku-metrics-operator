# Generate Downstream Release

This document outlines the process for generating the downstream version of the operator. The downstream version transforms all `koku` references to `costmanagement` for Red Hat's internal distribution.

## Prerequisites

* **rename** - Perl-based file rename utility ([install with Homebrew on OSX](https://formulae.brew.sh/formula/rename#default))
* **yq** - YAML processor ([installation guide](https://github.com/mikefarah/yq))
* **operator-sdk** - Operator SDK CLI tool
* **git** - Version control

## What the Downstream Target Does

The `make downstream` command performs the following transformations:
- Renames `koku` → `costmanagement` and `Koku` → `CostManagement` across the codebase
- Updates API group names, kinds, and project configuration
- Regenerates bundle manifests with downstream-specific metadata
- Appends OpenShift-specific labels to `bundle.Dockerfile`
- Sets `isCertified` flag to `true` for Red Hat certification

**Note:** The following files are excluded from transformations:
- `docs/downstream-releasing.md`
- `docs/report-fields-description.md`
- `docs/local-development.md`
- `docs/upstream-*.md`

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
- API references changed from `koku` to `costmanagement`
- `bundle.Dockerfile` contains `COPY bundle/manifests` (not `COPY manifests`)
- Bundle CSV has correct downstream metadata
- Excluded documentation files remain unchanged

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

Open a PR against the `downstream` branch to merge the downstream code changes.

**Important:** Ensure the PR base branch is set to `downstream`, not `main`.
