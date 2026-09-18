# Upstream Release — Community OperatorHub

> **Start here first:** [start-here.md](start-here.md) (when to release, Upstream vs Downstream).
>
> **Product:** `koku-metrics-operator` · **Branch:** `main` · **Image:** `quay.io/project-koku/koku-metrics-operator` · **Catalog:** [community-operators-prod](https://github.com/redhat-openshift-ecosystem/community-operators-prod)

This runbook ships a new community version to OperatorHub. It is independent of Downstream Konflux releases, but a full product cycle usually does Upstream first, then [downstream.md](downstream.md).

## Release order

1. Ensure intended features and fixes are merged into `main`.
2. Freeze **scope** (what is in / out). **Red Hat product CVE / UBI / RPM / certified-image work stays on Downstream** — see [cve.md](cve.md). Upstream may still ship ordinary dependency or Go toolchain updates on `main`.
3. Update OLM “New in” text in [csv-description.md](../csv-description.md) (`## New in vX.Y.Z`).
4. Run upgrade testing ([upstream-release-testing.md](../upstream-release-testing.md)).
5. Create the GitHub Release / tag `vX.Y.Z` (triggers the multi-arch image build).
6. Generate the OLM bundle and open a PR against `main`.
7. Submit the bundle to `community-operators-prod`.
8. After merge: auto FBC PR → community OperatorHub.

```
main (features) → CSV docs → upgrade test → GitHub Release (tag)
      → make bundle → PR main → community-operators-prod → OperatorHub
```

## Phase 1 — Scope and CSV description

### Decide what ships Upstream

| Include on Upstream | Keep for Downstream only |
|---------------------|---------------------------|
| Bugfixes and features already on `main` | Red Hat product CVE remediation (what clears `catalog.redhat.com`) |
| Dependency / Go toolchain updates on `main` (including ones that harden the *community* image) | UBI base bumps, `rpms.lock.yaml`, FIPS/OpenSSL packaging, certified Downstream Dockerfile |

Confirm with the release owner if a ticket is ambiguous. **Real example:** for **4.4.1**, Upstream took bugfixes and deps; Red Hat image security tickets stayed Downstream (see [CSV “New in v4.4.1” PR #957](https://github.com/project-koku/koku-metrics-operator/pull/957) and Upstream bundle [#960](https://github.com/project-koku/koku-metrics-operator/pull/960)).

### Update `docs/csv-description.md` (OLM / OperatorHub text)

Add a `## New in vX.Y.Z:` section (follow recent entries in [`docs/csv-description.md`](../csv-description.md)).

This is **not** the GitHub Release body. It is the prose that later gets embedded into the ClusterServiceVersion when you run `make bundle` (Phase 4). Operators and OperatorHub show this text.

### Phase 1 output

Open a PR against `main` of `koku-metrics-operator` (preferred: CSV-only) with `docs/csv-description.md` containing `## New in vX.Y.Z`, request review, and merge. Scope must be agreed and Red Hat product CVE/UBI/RPM work excluded. Done when that “New in” text is on `main` (or already on the branch you will use for `make bundle`).

Own PR is preferred (example: [#957](https://github.com/project-koku/koku-metrics-operator/pull/957) before the bundle PR). Same PR as the bundle is acceptable if you still write “New in” **before** `make bundle`.

### Two different “release notes”

| Artifact | When | What it is |
|----------|------|------------|
| `docs/csv-description.md` → “New in vX.Y.Z” | **Phase 1** (before bundle) | Feeds the **OLM CSV** / OperatorHub description |
| GitHub Release notes (UI form) | **Phase 3** (when you publish the tag) | Human release page on GitHub; often mirrors the “New in” bullets |

So: prepare the CSV “New in” text **before** the bundle. GitHub Release notes come later at the tag (you can copy the same bullets).

**Real order on 4.4.1:** CSV [#957](https://github.com/project-koku/koku-metrics-operator/pull/957) → GitHub Release [`v4.4.1`](https://github.com/project-koku/koku-metrics-operator/releases/tag/v4.4.1) → bundle [#960](https://github.com/project-koku/koku-metrics-operator/pull/960).

## Phase 2 — Upgrade testing

Validate OLM upgrade from the previous community version to the new one using a **personal** Quay image.

Full commands: [upstream-release-testing.md](../upstream-release-testing.md).

Summary:

```bash
PREVIOUS_VERSION=<last-community-version>   # e.g. 4.4.0
VERSION=<new-version>                       # e.g. 4.4.1
USERNAME=<your-quay-username>

oc new-project koku-metrics-operator
make bundle-deploy-previous
# Create a CostManagementMetricsConfig so the PVC exists and data can collect.

make docker-buildx IMG=quay.io/$USERNAME/koku-metrics-operator:v$VERSION
docker pull quay.io/$USERNAME/koku-metrics-operator:v$VERSION
make bundle CHANNELS=alpha,beta DEFAULT_CHANNEL=beta \
  IMG=quay.io/$USERNAME/koku-metrics-operator:v$VERSION
make bundle-build BUNDLE_IMG=quay.io/$USERNAME/koku-metrics-operator-bundle:v$VERSION bundle-push
make bundle-deploy-upgrade BUNDLE_IMG=quay.io/$USERNAME/koku-metrics-operator-bundle:v$VERSION
# Confirm OLM upgrades automatically; run smoke checks.
make bundle-deploy-cleanup
```

Testing on the latest supported OpenShift version is enough for a typical release unless a specific OCP regression is suspected.

## Phase 3 — GitHub Release and operator image

1. Open [Releases](https://github.com/project-koku/koku-metrics-operator/releases).
2. Draft a new release using a previous release as the template (e.g. [`v4.4.1`](https://github.com/project-koku/koku-metrics-operator/releases/tag/v4.4.1)).
3. Tag: `vX.Y.Z` (example: `v4.4.1`). Publish.

Publishing the tag triggers [`.github/workflows/build-and-publish.yaml`](../../.github/workflows/build-and-publish.yaml):

- Multi-arch operator image build
- Push to [quay.io/project-koku/koku-metrics-operator](https://quay.io/repository/project-koku/koku-metrics-operator?tab=tags)

The image is tagged with the **git tag name** (for example `v4.4.1`), not with a free-floating Makefile string alone.

If the tag does not appear on Quay, build and push manually from `main` at the release commit:

```bash
make docker-buildx
make docker-push
```

## Phase 4 — Generate the bundle and PR to `main`

This phase is entirely in the **operator repository**: [`project-koku/koku-metrics-operator`](https://github.com/project-koku/koku-metrics-operator) on branch **`main`**.  
(Do **not** use `community-operators-prod` yet — that is Phase 5.)

Once the org image is on Quay:

```bash
cd <path-to>/koku-metrics-operator
git checkout main && git pull origin main
git checkout -b release-bundle-vX.Y.Z
```

1. At the top of the `Makefile`, set:

   ```makefile
   PREVIOUS_VERSION ?= <last-version-on-community-OperatorHub>
   VERSION ?= <new-version>
   ```

   `PREVIOUS_VERSION` must be the last version **published** on community OperatorHub. It populates the CSV `replaces` field.

2. Pull the image so `operator-sdk` can embed the correct reference:

   ```bash
   docker pull quay.io/project-koku/koku-metrics-operator:v$VERSION
   # or: podman pull …
   ```

3. Generate the bundle:

   ```bash
   make bundle CHANNELS=alpha,beta DEFAULT_CHANNEL=beta
   ```

   This updates `bundle/` (version, channels, image reference, `replaces`).

4. **Open a PR** against **`main`** of **`koku-metrics-operator`** with `Makefile` + `bundle/` (and CSV description if not already merged). **Request review** and wait until it is **merged** before Phase 5.

**Real examples:** Upstream bundle PRs [#960](https://github.com/project-koku/koku-metrics-operator/pull/960) (`v4.4.1`), [#906](https://github.com/project-koku/koku-metrics-operator/pull/906) (`v4.4.0`).

Example commit message: `release: generate bundle for vX.Y.Z`.

### Phase 4 output

Open a PR against `main` of `koku-metrics-operator` with `Makefile` + `bundle/` for `vX.Y.Z`, request review, and merge. Done when the PR is merged, CSV `replaces` is `koku-metrics-operator.v<PREVIOUS_VERSION>`, and the image ref matches the Quay tag.

## Phase 5 — Submit to `community-operators-prod`

Repository: [redhat-openshift-ecosystem/community-operators-prod](https://github.com/redhat-openshift-ecosystem/community-operators-prod).

### 5.1 Copy the bundle

```bash
# Fork the repo on GitHub, then:
git clone git@github.com:<your-fork>/community-operators-prod.git
cd community-operators-prod
git checkout -b koku-metrics-operator-v$VERSION

mkdir -p operators/koku-metrics-operator/$VERSION
cp -r <path-to>/koku-metrics-operator/bundle/* \
  operators/koku-metrics-operator/$VERSION/
```

Expected layout:

```
community-operators-prod/operators/koku-metrics-operator/
├── 4.4.0/
│   ├── manifests/
│   ├── metadata/
│   └── release-config.yaml
└── 4.4.1/          ← new version
    ├── manifests/
    ├── metadata/
    └── release-config.yaml
```

### 5.2 Add `release-config.yaml`

Enables File-Based Catalog auto-release. Background: [FBC auto-release docs](https://redhat-openshift-ecosystem.github.io/operator-pipelines/users/fbc_autorelease/).

```yaml
---
catalog_templates:
  - template_name: basic.yaml
    channels: [beta, alpha]
    replaces: koku-metrics-operator.v<PREVIOUS_VERSION>
```

Place the file at `operators/koku-metrics-operator/<VERSION>/release-config.yaml`.

### 5.3 Signed commit and PR

```bash
git add operators/koku-metrics-operator/$VERSION
git commit -s -m "operator koku-metrics-operator ($VERSION)"
git push origin koku-metrics-operator-v$VERSION
```

**Open a PR** against `main` of `community-operators-prod` (using the access the team granted you), complete the repository checklist, **request review**, and **wait for merge**.

**Real examples:** browse merged PRs titled with `koku-metrics-operator` in [community-operators-prod](https://github.com/redhat-openshift-ecosystem/community-operators-prod/pulls?q=is%3Apr+is%3Amerged+koku-metrics-operator). Documented pattern: [#6824](https://github.com/redhat-openshift-ecosystem/community-operators-prod/pull/6824) with follow-up FBC [#6825](https://github.com/redhat-openshift-ecosystem/community-operators-prod/pull/6825).

### After your PR merges

1. An **automatic** follow-up PR updates FBCs for supported OCP versions (pattern like [#6825](https://github.com/redhat-openshift-ecosystem/community-operators-prod/pull/6825)). That bot PR also needs review/merge (you usually do not author it).
2. When that FBC PR merges, the new version appears on community OperatorHub.

### Phase 5 output

Open a PR on `community-operators-prod` with `operators/koku-metrics-operator/X.Y.Z/` (+ `release-config.yaml`), using a DCO signed commit (`git commit -s`). After it merges, a second usually automatic FBC PR must also merge. Done when both PRs are merged and OperatorHub shows `koku-metrics-operator` at `X.Y.Z`.

## Pitfalls

| Pitfall | What to do |
|---------|------------|
| Image missing on Quay after the GitHub Release | Wait for Actions; then `make docker-buildx` + `make docker-push` |
| Wrong `PREVIOUS_VERSION` | Must match last **published** community version (`replaces` / `release-config.yaml`) |
| Tag vs Makefile | Quay tag follows the **git tag** (`vX.Y.Z`) |
| Expecting Upstream alone to clear Red Hat product CVEs | Remediating `catalog.redhat.com` / `registry.redhat.io` requires Downstream ([cve.md](cve.md)) |
| Skipping upgrade testing | Always run Phase 2 before tagging when practical |
| Unsigned community-operators commit | Use `git commit -s` (DCO) |

## Checklist (copy/paste)

- [ ] Scope agreed; Red Hat product CVE / UBI / RPM work excluded from this Upstream release
- [ ] `docs/csv-description.md` has `## New in vX.Y.Z`
- [ ] Upgrade test passed ([upstream-release-testing.md](../upstream-release-testing.md))
- [ ] GitHub Release / tag `vX.Y.Z` published
- [ ] Image on `quay.io/project-koku/koku-metrics-operator:vX.Y.Z`
- [ ] Bundle PR merged on `main`
- [ ] `community-operators-prod` PR merged (signed commit + checklist)
- [ ] Auto FBC PR merged; OperatorHub shows the new version
- [ ] If Downstream will follow: hand off version + notes to [downstream.md](downstream.md)

## Related links

- [start-here.md](start-here.md)
- [downstream.md](downstream.md)
- [cve.md](cve.md)
- [upstream-release-testing.md](../upstream-release-testing.md)
- [csv-description.md](../csv-description.md)
- [community-operators-prod](https://github.com/redhat-openshift-ecosystem/community-operators-prod)
- Worked Upstream cycle (4.4.1): CSV [#957](https://github.com/project-koku/koku-metrics-operator/pull/957) → bundle [#960](https://github.com/project-koku/koku-metrics-operator/pull/960) → release [`v4.4.1`](https://github.com/project-koku/koku-metrics-operator/releases/tag/v4.4.1)
