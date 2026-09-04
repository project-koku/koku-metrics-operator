# Downstream Release — Red Hat Product

> **Start here first:** [start-here.md](start-here.md) · **CVEs:** [cve.md](cve.md) · **FBC commands:** [FBC README](https://github.com/project-koku/cost-management-metrics-operator-fbc)
>
> **Product:** `costmanagement-metrics-operator` · **Branch:** `downstream` · **CI:** Konflux · **Customer registry:** `registry.redhat.io`


## Overview

```
Phase A: Path 1 (port from main) OR Path 2 (bundle-only) → Konflux builds operator → nudge PR → bundle digest
Phase B: FBC (VERSION / PREVIOUS_VERSION / REGISTRY_SHA) → stage → QE
Phase C–D: prod Release YAMLs → oc apply (operator FIRST, then FBC)
Phase E: tag + GitHub Release + operator_versions + Slack
```

**Phases** A–E are the sequential release stages. **Paths** 1 and 2 are only the two ways to start Phase A (pick one). Path 1 (port) is the usual case.

**Column of truth:** `downstream` → operator image → nudge → bundle digest → FBC → stage → QE → prod.

## Before you start

Answer the **canonical questions** in [start-here.md](start-here.md) (code change vs CVE fix; QE mode). Then pick the Phase A path:

| Answer | Use | Real example |
|--------|-----|--------------|
| Code change / port from `main` (usual — e.g. `X.Y.0`) | [Phase A — Path 1](#path-1--port-upstream--downstream) | [4.4.1 port PR #965](https://github.com/project-koku/koku-metrics-operator/pull/965) |
| CVE / security-deps bundle-only (e.g. `X.Y.1+`) | [Phase A — Path 2](#path-2--bundle-only) | [4.4.2 bundle PR #1017](https://github.com/project-koku/koku-metrics-operator/pull/1017) |

## Prerequisites

- [ ] Jira Downstream release epic (issues + security tasks linked)
- [ ] Tools: `yq`, Operator SDK, Docker/Podman, `oc`, `gh`
- [ ] Konflux CLI access to namespace `cost-mgmt-dev-tenant`
- [ ] Corporate **VPN** when running FBC `make catalog` (pulls `registry.redhat.io`) and when using internal Red Hat build/catalog UIs for CVE work


### Konflux login

Preferred (from [`koku-ci` / `koku-ci-management`](https://github.com/project-koku/koku-ci)):

```bash
cd <path-to>/koku-ci/koku-ci-management
make login
eval $(make env)
oc whoami          # must NOT be system:anonymous
oc project cost-mgmt-dev-tenant
```

UI: [Konflux — cost-mgmt-dev-tenant](https://konflux-ui.apps.stone-prd-rh01.pg1f.p1.openshiftapps.com/ns/cost-mgmt-dev-tenant/)

## Phase A — Operator + Bundle (`koku-metrics-operator` / `downstream`)

### Path 1 — Port upstream → downstream

Use when `main` has features/bugfixes to ship in the product build. This is the more common way to start Phase A.

**Real example:** Downstream **4.4.1** ported Upstream changes — [PR #965](https://github.com/project-koku/koku-metrics-operator/pull/965) (*downstream updates for v4.4.1*). Earlier pattern: [4.4.0 #909](https://github.com/project-koku/koku-metrics-operator/pull/909).

**Order (do not skip):**

1. Start from updated `main` → work branch → `make downstream` → **commit** (still on the `main` lineage)
2. Create a **backup** branch (rollback point before the messy merge)
3. `git merge origin/downstream` → resolve conflicts → commit the resolution
4. Squash so the PR against `downstream` is ~1 commit (not hundreds from `main`)
5. Open the PR → wait for checks / review → merge

```bash
# 1. From main → work branch → transform → commit on main lineage
git checkout main && git pull origin main
git checkout -b downstream-updates-vX.Y.Z
make downstream
```

What `make downstream` does (summary): renames `koku` → `costmanagement` in API/config/CSV description/RBAC annotations; regenerates Downstream bundle metadata; sets certified packaging; drops Upstream-only scorecard assets as configured in the Makefile target.

Review key files (bundle Dockerfile metadata, CSV naming, `certified: true` in packaging, “Cost Management” **with a space** in display strings), then commit:

```bash
git add -A
git commit -m "chore: apply make downstream for vX.Y.Z"
```

```bash
# 2. Backup BEFORE merging downstream — tip must include the commit above
git checkout -b downstream-updates-vX.Y.Z-backup
git checkout downstream-updates-vX.Y.Z

# Optional: reduce vendor conflicts — merge open Dependabot/Renovate deps into this branch first
git fetch origin

# 3. Bring in downstream (expect many conflicts)
git merge origin/downstream
```

#### Conflict resolution (guidance, not a rigid law)

After `git merge origin/downstream`, Git callers:

- **ours** = your current port branch (the one that ran `make downstream` on top of `main`)
- **theirs** = incoming `origin/downstream`

Keep the backup branch until conflicts are resolved and the squash looks right. If resolution goes sideways: reset the work branch to `downstream-updates-vX.Y.Z-backup` and retry from the merge.

The table below is the **usual** choice for CMMO ports. It is **not** guaranteed for every file on every release — judgment is required. When unsure: compare the hunk, prefer Downstream packaging/security files from `downstream`, prefer ported product code from the port branch, and ask the release owner if a hunk looks wrong.

| File path | Usual preference | Why |
|-----------|------------------|-----|
| `bundle.Dockerfile` | **ours** | Keep new version/release; leave build-commit REPLACE for nudge |
| `Dockerfile` | **theirs**, then bump version | Upstream Dockerfile differs; set Downstream version manually |
| `internal/` | **ours** | Ported Upstream code |
| `vendor/` | **theirs** after deps merge, or re-vendor | Prefer merging deps first to shrink conflicts |
| `renovate.json` | **theirs** | Downstream does not use Renovate |
| `Makefile` | **ours** | Downstream version bumps |
| `go.mod` / `go.sum` | **theirs** | Downstream Go / toolchain |
| `api/v1beta1/` | **theirs** (often) | Skip Upstream-only API churn not needed Downstream |
| `docs/` | mostly **ours** | Keep ported Upstream docs content as appropriate |
| CSV `containerImage` / `image` | **theirs** pinned `@sha256:…` | Never leave a floating tag — EC fails |
| CSV date field | accept both, keep **new** release date | |
| CSV `4.0.0 API` style sections | **theirs** when Upstream-only | Confirm with release owner if unsure |

Also:

- Fix “Costmanagement” → “Cost Management” where display names require a space.
- Ensure packaging stays `certified: true`.
- After resolving vendor conflicts, if OpenShift API `zz_generated.*` files disagree with `go.mod`, force those files from `origin/downstream` (same generation as Downstream Go).

When conflicts are resolved:

```bash
git add -A
git commit -m "chore: merge origin/downstream and resolve conflicts for vX.Y.Z"
```

#### Squash before the PR (avoid the “hundreds of commits” trap)

The port branch started from **`main`**, then merged `origin/downstream`. If you open a PR against `downstream` **without** squashing, GitHub’s compare view can list **every commit that exists on `main` but not on `downstream`** — often hundreds of unrelated commits. That is confusing for reviewers and has bitten the team (close/reopen or rewrite the branch).

**Fix:** after the conflict-resolution commit, reset softly to `origin/downstream` and create **one** (or very few) commit(s) that contain only the port diff:

```bash
# 4. Squash to a clean diff vs downstream
git fetch origin
git reset --soft origin/downstream
git commit -m "chore: port upstream changes to downstream for vX.Y.Z"
git push --force-with-lease origin downstream-updates-vX.Y.Z
```

**Stop and verify:** the branch “Commits” tab (once the PR exists) should show ~1–few commits, not the entire `main` history.

#### Path 1 output

Open a PR against `downstream` of `koku-metrics-operator` with the squashed Upstream → Downstream port, request review, wait for checks to pass, and merge. Done when the PR is merged (example: [#965](https://github.com/project-koku/koku-metrics-operator/pull/965)); continue at A2.

### Path 2 — Bundle-only

Use when the release is security/deps already merged on `downstream` and there is **no** Upstream port for this version.

**Real example:** Downstream **4.4.2** was bundle-only (CVE/UBI work already on `downstream`) — see [PR #1017](https://github.com/project-koku/koku-metrics-operator/pull/1017) (*downstream bundle for v4.4.2*). CVE fix PRs merged earlier include [#1005](https://github.com/project-koku/koku-metrics-operator/pull/1005) and [#1008](https://github.com/project-koku/koku-metrics-operator/pull/1008).

```bash
git checkout downstream && git pull origin downstream
git checkout -b downstream-bundle-vX.Y.Z

# Update docs/csv-description.md — add "## New in vX.Y.Z" (Downstream wording)
make downstream-bundle

git diff   # expect mostly bundle/ and bundle.Dockerfile
# Optional: make fmt && make vet && make vendor
```

#### Required fix before opening the PR — pin image digests

`make downstream-bundle` may write CSV `containerImage` / deployment `image` as a **tag** (`:X.Y.Z`). Enterprise Contract rejects that (`olm.unpinned_references`).

Pin both fields to the **current** Downstream SHA (previous released operator digest) before the PR. The nudge PR later updates them to the new digest:

```bash
git show origin/downstream:bundle/manifests/costmanagement-metrics-operator.clusterserviceversion.yaml \
  | grep containerImage
# → …@sha256:<previous>
```

Commit and push the pinned CSV (and related bundle files) on your branch.

#### Path 2 output

Open a PR against `downstream` of `koku-metrics-operator` with the Downstream bundle update (CSV pinned `@sha256:…`), request review, and merge. Done when the PR is merged (example: [#1017](https://github.com/project-koku/koku-metrics-operator/pull/1017)); continue at A2.

### A2 — Merge → operator image build

After the Path 1 or Path 2 PR is **reviewed and merged**, Konflux builds the operator (on-push). Do not skip the review/merge gate.

```bash
oc get pipelinerun -n cost-mgmt-dev-tenant | grep costmanagement-metrics-operator
oc get snapshot -l pac.test.appstudio.openshift.io/sha=<merge-commit-sha>
```

UI: Snapshots → component `costmanagement-metrics-operator`.

### A3 — Nudge PR (bundle digest update)

Konflux opens a PR on `downstream` (bot `red-hat-konflux`, label `konflux-nudge`) titled like:

`chore(deps): update costmanagement-metrics-operator to <digest>`

**Real examples:** [nudge #988](https://github.com/project-koku/koku-metrics-operator/pull/988) (4.4.2 era), [nudge #978](https://github.com/project-koku/koku-metrics-operator/pull/978) (4.4.1 era).

It updates the CSV image pins to the new operator digest. It does **not** reliably set the bundle build-commit labels.

#### Manual commit on the nudge branch

Edit `bundle.Dockerfile` to set **hardcoded** `LABEL`s (do **not** rely on `ARG COMMIT_REF` — Konflux may override build-args):

```dockerfile
LABEL io.openshift.build.commit.id="<merge-commit-sha-of-bundle-or-operator-PR>"
LABEL io.openshift.build.commit.url="https://github.com/project-koku/koku-metrics-operator/commit/<merge-commit-sha>"
```

Use the merge commit that represents the Downstream content being released (typically the bundle/port PR merge on `downstream`). Confirm with `git log origin/downstream`.

Commit and push that change onto the **existing** Konflux nudge PR branch (do not open a second PR).

### A4 — Final snapshot and bundle digest

The snapshot from the **merged nudge** (operator + bundle) is the release candidate.

```bash
oc get snapshot <snapshot-name> -n cost-mgmt-dev-tenant \
  -o jsonpath='{range .spec.components[*]}{.name}{" "}{.containerImage}{"\n"}{end}'
```

Copy the digest for **`costmanagement-metrics-operator-bundle`** → this is `REGISTRY_SHA` for Phase B.

The FBC Makefile expects the full form `sha256:<hex>` (see existing `REGISTRY_SHA` examples there). From the `containerImage` line (`…@sha256:<hex>`), copy either:

- `sha256:<hex>` as-is, or
- only the `<hex>` and prepend `sha256:` yourself

Do **not** paste a value that already starts with `sha256:` into a placeholder that also adds `sha256:` (that produces `sha256:sha256:…`).

## Phase B — FBC (`cost-management-metrics-operator-fbc`)

**Commands, Make targets, catalog layout, gathering CatalogSource images, and `configure_cluster.py`:** follow the **[FBC README](https://github.com/project-koku/cost-management-metrics-operator-fbc)** (SSOT for this repo’s tooling). This section only covers how FBC fits the Downstream release train.

### Release-train checklist

1. **B0 — Heads-up to QE** (before images) — notify IBM Power / IBM Z / ROS early (often when the Downstream PR is ready). IBM testing frequently takes ~1 week. Typical channels: `costmanagement-pz-collab` (Power/Z), `#finsights-dev` (ROS), `#forum-cost-mgmt` (announce later).
2. **B1 — Generate catalogs** — in the FBC repo, set `VERSION` / `PREVIOUS_VERSION` / `REGISTRY_SHA` (`REGISTRY_SHA` = **bundle** digest from A4, form `sha256:<hex>` — do not double the prefix). Then follow the FBC README (**How to update**). VPN is required for `make catalog`. **Open a PR** against `main`, request review, merge.
   - Examples: [4.4.2 #123](https://github.com/project-koku/cost-management-metrics-operator-fbc/pull/123), [4.4.1 #110](https://github.com/project-koku/cost-management-metrics-operator-fbc/pull/110) (rebuild [#111](https://github.com/project-koku/cost-management-metrics-operator-fbc/pull/111) when needed), [4.4.0 #99](https://github.com/project-koku/cost-management-metrics-operator-fbc/pull/99).
3. **B2 — Stage success for every OCP** — after merge, Konflux builds **one Component per OCP major** (`OCP_VERSIONS` in the FBC Makefile — today ~11+). Phase B is done only when **all** of those stage Releases succeed for your merge SHA (not a single `v4-XX`). Use the FBC README (**Gather FBC for QE**) or the [Konflux UI](https://konflux-ui.apps.stone-prd-rh01.pg1f.p1.openshiftapps.com/ns/cost-mgmt-dev-tenant/) Applications `…-fbc-v4-XX`.
4. **B3 — Keep snapshots and send images to QE** — annotate / list CatalogSource images per the FBC README, then send the list (see [start-here.md](start-here.md) QE question and [QE coordination](#qe-coordination)):

| Mode | What to do with the list |
|------|--------------------------|
| **Request testing** (functional changes) | Paste the image list + ask for testing (templates under [QE coordination](#qe-coordination)) |
| **Info-only** (security/deps-only) | Still share the images, but mark as information only |

#### B1 output

FBC catalog PR merged for `VERSION` (using `REGISTRY_SHA` from A4). Examples: [#123](https://github.com/project-koku/cost-management-metrics-operator-fbc/pull/123), [#110](https://github.com/project-koku/cost-management-metrics-operator-fbc/pull/110).

### Phase B pitfalls

| Pitfall | What to do |
|---------|------------|
| Looking at only one FBC Application / Component in the UI | Check **all** `…-fbc-v4-XX` apps / all CatalogSource lines for the merge SHA |
| Reusing old snapshots | Always use snapshots labeled with **this** FBC merge SHA |
| Skipping `keep-snapshot` | Images disappear; QE blocked (see FBC README) |
| Wrong `REGISTRY_SHA` | Must be the **bundle** digest from A4, not a random operator tag |
| Re-running pipelines ad hoc / duplicate snapshots | Prefer the builds from the **merged** FBC commit; do not mix SHAs when gathering images for QE |

IQE pipeline runs on FBC apps are useful signal but are **not** always a hard gate for stage auto-release — confirm current policy with the release owner if a run fails.

## Phase C — Production Release YAMLs

Store YAMLs under `releases/X.Y.Z/` in the **FBC repo** (Konflux does not keep Release objects forever; the folder is the team audit trail).

**Open a PR** against `main` of the FBC repo for review (**request reviewers**, address feedback). **Merging is not required** to `oc apply` — the PR is for peer check and history — but do not apply until someone else has looked at the snapshots/CVE list when practical.

### Start from a real previous release (do this)

**Do not invent the YAML from scratch.** Copy the latest `releases/<previous>/` pair and search/replace.

| Version | Operator YAML | FBC YAML | Add-releases PR |
|---------|---------------|----------|-----------------|
| **4.4.2** (best template; includes `prod-2` / `-000-2` retries) | [`costmanagement-metrics-operator-4.4.2.yaml`](https://github.com/project-koku/cost-management-metrics-operator-fbc/blob/main/releases/4.4.2/costmanagement-metrics-operator-4.4.2.yaml) | [`…-fbc-4.4.2.yaml`](https://github.com/project-koku/cost-management-metrics-operator-fbc/blob/main/releases/4.4.2/costmanagement-metrics-operator-fbc-4.4.2.yaml) | [FBC#126](https://github.com/project-koku/cost-management-metrics-operator-fbc/pull/126) |
| 4.4.1 | [`…-4.4.1.yaml`](https://github.com/project-koku/cost-management-metrics-operator-fbc/blob/main/releases/4.4.1/costmanagement-metrics-operator-4.4.1.yaml) | [`…-fbc-4.4.1.yaml`](https://github.com/project-koku/cost-management-metrics-operator-fbc/blob/main/releases/4.4.1/costmanagement-metrics-operator-fbc-4.4.1.yaml) | [FBC#119](https://github.com/project-koku/cost-management-metrics-operator-fbc/pull/119) |
| 4.4.0 | [`releases/4.4.0/`](https://github.com/project-koku/cost-management-metrics-operator-fbc/tree/main/releases/4.4.0) | same folder | [FBC#101](https://github.com/project-koku/cost-management-metrics-operator-fbc/pull/101) |

```bash
cd <path-to>/cost-management-metrics-operator-fbc
git checkout main && git pull
git checkout -b operator-vX.Y.Z-release
mkdir -p releases/X.Y.Z
cp releases/4.4.2/*.yaml releases/X.Y.Z/
# rename files to …-X.Y.Z.yaml, then edit fields below
```

### Advisory type

Follow [Releasing with an advisory](https://konflux.pages.redhat.com/docs/users/releasing/releasing-with-an-advisory.html):

| Type | When | Real example |
|------|------|--------------|
| **RHSA** | Any security fix (even if bugs/enhancements also ship) | 4.4.1 and 4.4.2 (`type: RHSA`) |
| **RHBA** | Bug fixes only | — |
| **RHEA** | Enhancements only | — |

### Operator Release YAML — fields and where to get them

Real shape (abbreviated from [4.4.2 operator YAML](https://github.com/project-koku/cost-management-metrics-operator-fbc/blob/main/releases/4.4.2/costmanagement-metrics-operator-4.4.2.yaml)):

```yaml
metadata:
  labels:
    release.appstudio.openshift.io/author: "rh-ee-lbacciot"   # your Konflux username
  name: costmanagement-metrics-operator-4.4.2-prod-2          # -prod-1 first try; -prod-2 after retry
spec:
  releasePlan: costmanagement-metrics-operator
  snapshot: costmanagement-metrics-operator-20260709-160540-000
  data:
    releaseNotes:
      type: RHSA
      topic: "Cost Management Metrics Operator version 4.4.2 release."
      issues:
        fixed:
          - id: COST-7461
            source: issues.redhat.com
      cves:
        - key: CVE-2026-5450
          component: costmanagement-metrics-operator
          packages:
            - glibc
```

| Field | Where the value comes from | 4.4.2 example |
|-------|----------------------------|---------------|
| `author` | Konflux / SSO username (`oc whoami` style id, often `rh-ee-…`) | `rh-ee-lbacciot` |
| `name` | `{operator}-{version}-prod-N` | `…-4.4.2-prod-2` (retry; first attempt was `prod-1`) |
| `releasePlan` | Fixed for this product | `costmanagement-metrics-operator` |
| `snapshot` | Phase A final snapshot (operator+bundle after nudge). Konflux UI → Snapshots, or `oc get snapshot …` | `costmanagement-metrics-operator-20260709-160540-000` |
| `type` | RHSA if any CVE; else RHBA/RHEA | `RHSA` |
| `topic` | One-line release blurb | `… version 4.4.2 release.` |
| `issues.fixed` | Linked COST tasks on the release epic that are **bugs/enhancements**, not the epic itself. Security/CVE cards may appear here when they are tracked as COST issues **and** also listed under `cves` — follow the previous YAML pattern for the release. | e.g. `COST-7461`, `COST-7781`, … |
| `cves` | [catalog.redhat.com Security tab](https://catalog.redhat.com/software/containers/costmanagement/costmanagement-metrics-rhel9-operator/67168aeaef33c5ffaff8cd45) + [cve.md](cve.md) | `CVE-2026-5450` / package `glibc`, etc. |
| `references` | Classification URL + one URL per CVE + product getting-started doc | see 4.4.2 file |

**How to find the operator snapshot name:** after Phase A4, in Konflux UI open the snapshot that contains **both** `costmanagement-metrics-operator` and `…-bundle`, or:

```bash
oc get snapshot -n cost-mgmt-dev-tenant \
  -l pac.test.appstudio.openshift.io/sha=<nudge-or-bundle-merge-sha>
```

For 4.4.1 the snapshot was `costmanagement-metrics-operator-20260609-072014-000` ([4.4.1 operator YAML](https://github.com/project-koku/cost-management-metrics-operator-fbc/blob/main/releases/4.4.1/costmanagement-metrics-operator-4.4.1.yaml)).

### FBC Release YAML — fields and where to get them

One `Release` document per OCP major (file is a multi-doc YAML). Real example — first and a **retry** entry from [4.4.2 FBC YAML](https://github.com/project-koku/cost-management-metrics-operator-fbc/blob/main/releases/4.4.2/costmanagement-metrics-operator-fbc-4.4.2.yaml):

```yaml
metadata:
  labels:
    release.appstudio.openshift.io/author: 'rh-ee-lbacciot'
    release.appstudio.openshift.io/version: '4.4.2'
    release.appstudio.openshift.io/sha: 'ceb7fcb6b2664eecc936867e35f95bcc2f0d110e'  # FBC merge commit
  name: costmanagement-metrics-operator-fbc-v4-12-20260710-125109-000-1
spec:
  releasePlan: costmanagement-metrics-operator-fbc-v4-12
  snapshot: costmanagement-metrics-operator-fbc-v4-12-20260710-125109-000
# …
# Retry example (v4-20 used -000-2):
# name: …-v4-20-20260710-125109-000-2
# snapshot: …-v4-20-20260710-125109-000   # same snapshot, new Release name
```

| Field | Where the value comes from | 4.4.2 example |
|-------|----------------------------|---------------|
| `author` / `version` | Same author; product version string | `4.4.2` |
| `sha` | Git commit that merged the FBC catalog PR (label on FBC snapshots) | `ceb7fcb6b2664eecc936867e35f95bcc2f0d110e` (from [FBC#123](https://github.com/project-koku/cost-management-metrics-operator-fbc/pull/123) merge) |
| `snapshot` | Exact FBC snapshot name per OCP from Phase B3 | `…-fbc-v4-12-20260710-125109-000` |
| `name` | `{snapshot}-1` first apply; `{snapshot}-2` on retry | v4-12 used `-000-1`; v4-20 used `-000-2` |
| `releasePlan` | `costmanagement-metrics-operator-fbc-v4-<OCP>` | `…-fbc-v4-12`, `…-v4-22`, … |

Confirm the `sha` label:

```bash
oc get snapshot <one-fbc-snapshot> -n cost-mgmt-dev-tenant \
  -o jsonpath='{.metadata.labels.pac\.test\.appstudio\.openshift\.io/sha}{"\n"}'
```

### Real PRs for this phase

| What | Link |
|------|------|
| Add 4.4.2 operator + FBC release YAMLs | [FBC#126](https://github.com/project-koku/cost-management-metrics-operator-fbc/pull/126) |
| Retry operator as `prod-2` (Kerberos/embargo transient) | [FBC#128](https://github.com/project-koku/cost-management-metrics-operator-fbc/pull/128) |
| Retry single FBC (`v4-20` `-000-2`) | commit `5327dac` on FBC `main` (*chore: retry FBC v4-20…*) |
| Add 4.4.1 release YAMLs | [FBC#119](https://github.com/project-koku/cost-management-metrics-operator-fbc/pull/119) |

Verify every snapshot against Konflux before asking for review. Mistakes are painful to unwind in customer environments.

#### Phase C output

Open a PR against `main` of `cost-management-metrics-operator-fbc` with operator + FBC prod Release YAMLs under `releases/X.Y.Z/`, and request review. Merge before apply is optional but recommended for history. Done when the reviewed YAMLs are ready, snapshots/CVE list are double-checked, and you can proceed to Phase D apply.

## Phase D — Apply to production

**Order is mandatory: operator first, then FBCs.**

```bash
# 1. Operator
oc apply -f releases/X.Y.Z/costmanagement-metrics-operator-X.Y.Z.yaml
oc get release costmanagement-metrics-operator-X.Y.Z-prod-1 -n cost-mgmt-dev-tenant -w

# 2. After operator Succeeded — FBCs
oc apply -f releases/X.Y.Z/costmanagement-metrics-operator-fbc-X.Y.Z.yaml

# Prefer filtering prod Release names (…-000-1 / …-000-2). A plain `grep fbc`
# also matches leftover stage Releases and is noisy.
oc get release -n cost-mgmt-dev-tenant | grep fbc | grep -- '-000-'
# or, first attempt only:
# oc get release -n cost-mgmt-dev-tenant | grep fbc | grep -- '-000-1'
```

Confirm the version appears on [catalog.redhat.com](https://catalog.redhat.com/software/containers/costmanagement/costmanagement-metrics-rhel9-operator/67168aeaef33c5ffaff8cd45).

Before apply, prefer confirming [catalog.stage.redhat.com](https://catalog.stage.redhat.com/en/software/containers/costmanagement/costmanagement-metrics-rhel9-operator/671692de3eefd9dc50b4f0e2) Security already reflects the fixes you are about to ship ([cve.md](cve.md)).

### Retries

Failed `Release` objects are not re-runnable. Create a new object with a new name:

| Failure | Action | Real example |
|---------|--------|--------------|
| Operator (e.g. transient Kerberos / embargo-check) | `prod-1` → `prod-2`, apply again, update the YAML PR | [FBC#128](https://github.com/project-koku/cost-management-metrics-operator-fbc/pull/128) (`4.4.2-prod-2`) |
| Individual FBC `publish-index-image` | That entry `-000-1` → `-000-2`, apply only what failed | 4.4.2 `v4-20` entry ends in `-000-2` in the [FBC YAML](https://github.com/project-koku/cost-management-metrics-operator-fbc/blob/main/releases/4.4.2/costmanagement-metrics-operator-fbc-4.4.2.yaml) |

Use a **fresh branch from `main`** for retry YAML fixes — do not reopen a huge FBC feature branch. **Open a small PR** for the rename (`prod-2` / `-000-2`) when you retry (example: [FBC#128](https://github.com/project-koku/cost-management-metrics-operator-fbc/pull/128)).

#### Phase D output

Prod `Release` objects Succeeded and the version appears on [catalog.redhat.com](https://catalog.redhat.com/software/containers/costmanagement/costmanagement-metrics-rhel9-operator/67168aeaef33c5ffaff8cd45). Done when the operator Release Succeeded, then all FBC prod Releases Succeeded, and the catalog shows `X.Y.Z`. If you need a retry YAML update (`prod-2` / `-000-2`), open a small PR for that rename.

## Phase E — Post-release

### E1 — Tag `vX.Y.Z-downstream`

Create an **annotated** tag on the commit that built the **operator image**, not necessarily the nudge merge.

The snapshot label `pac.test.appstudio.openshift.io/sha` often points at the nudge commit — that is frequently the **wrong** tag target.

Prefer the snapshot annotation / UI message that shows `rev=<operator-build-sha>`, for example:

```bash
oc get snapshot <snapshot-name> -n cost-mgmt-dev-tenant \
  -o jsonpath='{.metadata.annotations.pac\.test\.appstudio\.openshift\.io/sha-title}'
# look for rev=<sha>
```

**Real examples (why “nudge SHA ≠ tag SHA” matters):**

| Release | Tag | Commit tagged | Notes |
|---------|-----|---------------|-------|
| 4.4.2 | [`v4.4.2-downstream`](https://github.com/project-koku/koku-metrics-operator/releases/tag/v4.4.2-downstream) | `cdd836a1` ([#1017](https://github.com/project-koku/koku-metrics-operator/pull/1017) bundle) | Snapshot label may show nudge [#988](https://github.com/project-koku/koku-metrics-operator/pull/988) — do not tag that |
| 4.4.1 | [`v4.4.1-downstream`](https://github.com/project-koku/koku-metrics-operator/releases/tag/v4.4.1-downstream) | `53d63c3a` ([#977](https://github.com/project-koku/koku-metrics-operator/pull/977)) | Snapshot label may show nudge [#978](https://github.com/project-koku/koku-metrics-operator/pull/978) — do not tag that |

```bash
cd <path-to>/koku-metrics-operator
git tag -a vX.Y.Z-downstream <operator-build-sha> \
  -m "Cost Management Metrics Operator version X.Y.Z"
git push origin vX.Y.Z-downstream
```

### E2 — GitHub Release

```bash
gh release create vX.Y.Z-downstream \
  --title "costmanagement-metrics-operator:X.Y.Z" \
  --notes "..."
```

Use a previous Downstream release as the template (features blurb, “New in vX.Y.Z”, PR list, compare link).

```bash
git log vX.Y.(Z-1)-downstream..vX.Y.Z-downstream --oneline --no-merges
```

### E3 — `koku` `operator_versions`

Add `(version, downstream_commit, upstream_commit)` in `koku/masu/util/ocp/operator_versions.py` so the backend can map the SHA the operator sends to a human version.

- `downstream_commit`: same SHA as the Downstream tag
- `upstream_commit`: Upstream tag SHA when an Upstream release exists; for Downstream-only releases ask the release owner

**Real example:** [koku#6129](https://github.com/project-koku/koku/pull/6129) (*add operator 4.4.1 build commits to operator_versions*).

### E4 — Announce

Post in `#forum-cost-mgmt` that `costmanagement-metrics-operator X.Y.Z` is available (link release notes — e.g. [`v4.4.2-downstream`](https://github.com/project-koku/koku-metrics-operator/releases/tag/v4.4.2-downstream)).

Optionally confirm the demo cluster upgraded.

## QE coordination

| Release type | What to do |
|--------------|------------|
| Functional changes | Request testing from COST QE (x86), IBM Power, IBM Z, ROS as required |
| Security / deps only | Share CatalogSource images as **information only**; get team consensus before skipping deep QE |

**Heads-up (before images):**

```text
Heads up! We are preparing operator version X.Y.Z for release.
Expect catalog images for testing early next week / by <date>.
```

**Images ready (functional):**

```text
Catalog images for costmanagement-metrics-operator X.Y.Z are ready for QE.
FBC merge SHA: <sha>

CatalogSource images:
<paste oc get snapshot … output>
```

**Info-only (security):**

```text
Candidate FBC images for operator vX.Y.Z — information only.
This release contains security/dependency updates with no operator functionality changes.

<paste images>
```

Note: older OCP majors may be single-arch (x86_64 only) — call that out when relevant.

IBM Z blockers are often **Vault/access**, not test failures. Escalation / proceed-without-Z is a release-owner call when infra is the only blocker.

## End-to-end checklist

- [ ] Code change vs CVE fix decided ([start-here.md](start-here.md)); QE mode decided
- [ ] Phase A PR merged; operator built; nudge merged with build-commit LABELs
- [ ] Bundle digest recorded (`REGISTRY_SHA`)
- [ ] QE heads-up sent
- [ ] FBC PR merged; stage all Succeeded; `keep-snapshot`; images shared
- [ ] Prod YAMLs reviewed (advisory type, issues, CVEs); stage catalog Security checked
- [ ] `oc apply` operator → Succeeded → apply FBCs → Succeeded
- [ ] catalog.redhat.com shows `X.Y.Z`
- [ ] Tag `vX.Y.Z-downstream` on correct commit; GitHub Release; `operator_versions`; Slack

## Pitfalls index

| Symptom / risk | Cause / fix |
|----------------|-------------|
| EC `olm.unpinned_references` | CSV still has `:X.Y.Z` — pin `@sha256:` from current Downstream |
| Wrong tag commit | Tagged nudge SHA — use operator image `rev=` instead |
| Vendor compile errors after port | `zz_generated.*` mismatched with Downstream `go.mod` |
| Stage missing / wrong snapshots | Prefer builds from the **merged** FBC commit SHA; do not mix ad-hoc reruns |
| Prod apply order wrong | Always operator, then FBC |
| CVE PR merged, customers still flagged | Delivery only after Phase D |

## Related links

- [start-here.md](start-here.md)
- [upstream.md](upstream.md)
- [cve.md](cve.md)
- [FBC repository](https://github.com/project-koku/cost-management-metrics-operator-fbc)
- [Konflux — releasing with an advisory](https://konflux.pages.redhat.com/docs/users/releasing/releasing-with-an-advisory.html)
- [catalog.redhat.com — operator](https://catalog.redhat.com/software/containers/costmanagement/costmanagement-metrics-rhel9-operator/67168aeaef33c5ffaff8cd45)
