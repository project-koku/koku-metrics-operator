# Operator Release — Start Here

> **Single source of truth** for the release process. Accessory repos should **link here** instead of duplicating steps.
>
> For AI agents: read [AGENTS.md](../../AGENTS.md) and [AI agent notes](#ai-agent-notes) before proposing release commands.

## When should we release?

There is no fixed calendar. Ship when there is clear customer-facing value **or** a security SLA — and a Jira release epic (or equivalent) that defines scope.

### Versioning convention

| Version shape | What it usually means | Tracks |
|---------------|----------------------|--------|
| **`X.Y.0`** | Code changes / bugfixes (Upstream + Downstream **port**). May also include CVEs shipped in the same Downstream cycle. | Upstream then Downstream Path 1 |
| **`X.Y.1+`** | Downstream-only security / deps (CVE, UBI, RPM, certified-image). **Bundle-only** — no Upstream port. | Downstream Path 2 + [cve.md](cve.md) |

Examples: **4.4.1** = `X.Y.0`-style port; **4.4.2** = `X.Y.1+` Downstream-only CVE release.

| Situation | Ship… | Notes |
|-----------|-------|-------|
| Bugfixes, features, or dependency updates ready on `main` | **Upstream**, then **Downstream** (port) | Upstream may still ship dependency or Go toolchain updates on `main`; that does **not** remediate CVEs for the certified Downstream image customers see on `catalog.redhat.com`. Typical version: `X.Y.0`. |
| Red Hat product CVE / UBI / RPM / certified-image fixes | **Downstream only** (bundle-only) | Clears what customers see on `catalog.redhat.com`. A merged fix is **not** delivered until the Downstream operator image **and** the FBC catalog release for **every** supported OCP major succeed on `registry.redhat.io`. Typical version: `X.Y.1+`. |
| Both product code and Red Hat image security ready | **Upstream first**, then Downstream | Upstream is the simpler path; team practice is to land community when Upstream scope exists. Often an `X.Y.0` that also carries CVEs Downstream. |
| Nothing ready for customers | **Do not release** | Wait for agreed scope |

**Stop and answer before any branch work** (canonical questions — Downstream links here instead of repeating them):

1. Is this a **code change** (Upstream → Downstream **port**) or a **CVE / security-deps fix** (Downstream-only / **bundle-only**)?  
   - Code change → [upstream.md](upstream.md), then [downstream.md](downstream.md) Path 1.  
   - CVE / security-deps → [cve.md](cve.md) + [downstream.md](downstream.md) Path 2 (e.g. **4.4.2**).
2. For Downstream QE: **request testing**, or **info-only** (security/deps-only releases are often info-only — see [cve.md](cve.md))?

If unsure, ask the release owner before opening PRs.

## Choose your path

| Track | What you are doing | Document |
|-------|--------------------|----------|
| **Upstream** | Community OperatorHub release (`koku-metrics-operator`) | [upstream.md](upstream.md) |
| **Downstream** | Red Hat product release (`costmanagement-metrics-operator`) | [downstream.md](downstream.md) |
| **Downstream** | CVE intake → fix → list in Downstream prod YAMLs | [cve.md](cve.md) |

**Full cycle (when both apply):** Upstream → Downstream. CVE documentation feeds Downstream prod Release YAMLs ([cve.md](cve.md) + [downstream.md](downstream.md) Phase C).

### Worked examples (learn by reading real PRs)

See [versioning convention](#versioning-convention) for `X.Y.0` vs `X.Y.1+`.

| Cycle | Type | What to open |
|-------|------|----------------|
| Upstream **4.4.1** | Code (`X.Y.0`) | CSV [#957](https://github.com/project-koku/koku-metrics-operator/pull/957) → bundle [#960](https://github.com/project-koku/koku-metrics-operator/pull/960) → [`v4.4.1`](https://github.com/project-koku/koku-metrics-operator/releases/tag/v4.4.1) |
| Downstream **4.4.1** (port) | Code (`X.Y.0`) | Port [#965](https://github.com/project-koku/koku-metrics-operator/pull/965) → nudge [#978](https://github.com/project-koku/koku-metrics-operator/pull/978) → FBC catalog [#110](https://github.com/project-koku/cost-management-metrics-operator-fbc/pull/110) → prod YAMLs [#119](https://github.com/project-koku/cost-management-metrics-operator-fbc/pull/119) → [`v4.4.1-downstream`](https://github.com/project-koku/koku-metrics-operator/releases/tag/v4.4.1-downstream) → `operator_versions` [koku#6129](https://github.com/project-koku/koku/pull/6129) |
| Downstream **4.4.2** (bundle-only + CVEs) | CVE (`X.Y.1+`) | CVE [#1005](https://github.com/project-koku/koku-metrics-operator/pull/1005)/[#1008](https://github.com/project-koku/koku-metrics-operator/pull/1008) → bundle [#1017](https://github.com/project-koku/koku-metrics-operator/pull/1017) → nudge [#988](https://github.com/project-koku/koku-metrics-operator/pull/988) → FBC [#123](https://github.com/project-koku/cost-management-metrics-operator-fbc/pull/123) → prod YAMLs [#126](https://github.com/project-koku/cost-management-metrics-operator-fbc/pull/126) → retries [#128](https://github.com/project-koku/cost-management-metrics-operator-fbc/pull/128) → [`v4.4.2-downstream`](https://github.com/project-koku/koku-metrics-operator/releases/tag/v4.4.2-downstream) |

## Mental model

```
Upstream (main)                          Downstream (downstream branch)
───────────────                          ─────────────────────────────
GitHub Release + Quay image              Port or bundle-only → Konflux
     → make bundle → PR main                  → nudge → bundle digest
     → community-operators-prod               → FBC → stage → QE
     → OperatorHub                            → prod apply (operator, then FBC)
                                              → tag + announce
```

| | Upstream (`main`) | Downstream (`downstream`) |
|--|-------------------|---------------------------|
| Name | `koku-metrics-operator` | `costmanagement-metrics-operator` |
| Distribution | Community OperatorHub | `registry.redhat.io` (Red Hat product) |
| CI | GitHub Actions (`.github/workflows/`) | Konflux / Tekton (`.tekton/`) |
| Base images | Public registries (e.g. `docker.io`) | Red Hat UBI via internal registries (e.g. `brew.registry.redhat.io`) |
| Catalog / index | `community-operators-prod` | `cost-management-metrics-operator-fbc` |

**Golden rule:** develop features on **`main` first**. Downstream receives ported changes, plus Downstream-only Red Hat packaging (UBI, RPMs, FIPS, product CVE remediation).

Do **not** edit `.tekton/` on Upstream PRs or `.github/workflows/` on Downstream PRs. Makefile/Dockerfile differences between branches are intentional (do not “unify” them when porting).

## Prerequisites

### Everyone

- [ ] GitHub access to `project-koku/koku-metrics-operator`
- [ ] Quay account (personal username for upgrade testing)
- [ ] Local tools: Go (see `go.mod`), `oc`/`kubectl`, Operator SDK, Docker or Podman

### Upstream

- [ ] Permission to create GitHub Releases / tags on the operator repo
- [ ] Access to submit bundles to [community-operators-prod](https://github.com/redhat-openshift-ecosystem/community-operators-prod) (ask the release owner / org admins — the team typically **grants repo access** rather than relying only on a personal fork)
- [ ] PRs there require a **DCO sign-off**: use `git commit -s` so each commit gets a `Signed-off-by:` trailer (Developer Certificate of Origin — not related to GPG signing)

### Downstream / CVE

- [ ] Konflux login for tenant `cost-mgmt-dev-tenant` (via `koku-ci` / `koku-ci-management`)
- [ ] `yq` (Downstream port)
- [ ] VPN for `registry.redhat.io` pulls (FBC `make catalog`) and for internal Red Hat build/catalog UIs used in CVE work (not macOS Homebrew)
- [ ] Access to the release Jira epic and Slack channels used for QE heads-up

## Glossary (short)

| Term | Meaning |
|------|---------|
| CMMO | Cost Management Metrics Operator (product name) |
| Bundle | OLM package: manifests + metadata + `bundle.Dockerfile` |
| CSV | ClusterServiceVersion — OLM install manifest for the operator |
| FBC | File-Based Catalog — operator index, one catalog per OCP major |
| Nudge | Konflux bot PR that pins the new operator image digest into the bundle CSV |
| Stage | Pre-production registry path used for QE (`registry.stage.redhat.io`) |
| RHSA / RHBA / RHEA | Advisory types on Downstream prod `Release` objects (security / bug / enhancement) |
| DCO | Developer Certificate of Origin — `git commit -s` adds `Signed-off-by:` (required by some repos, e.g. community-operators-prod) |

## QE / IQE (pointer only)

Downstream QE tests **CatalogSource** images produced after FBC stage succeeds. Cluster setup and IQE execution live outside this hub:

- [FBC README — Gather FBC for QE / `configure_cluster.py`](https://github.com/project-koku/cost-management-metrics-operator-fbc)
- Team IQE / QE docs and Slack coordination (ask the release owner on day one)

Upstream upgrade testing (OLM previous → new) is part of the Upstream runbook and is detailed in [upstream-release-testing.md](../upstream-release-testing.md).

## AI agent notes

- Always resolve: **code change** (Upstream → Downstream port) or **CVE / security-deps** (Downstream-only)? Never mix CI folders or Dockerfiles across branches.
- Downstream production: apply the **operator** `Release` **before** FBC `Release`s.
- Never invent digests, snapshot names, or CVE IDs — copy from Konflux / `oc` / catalog.redhat.com.
- Prefer this documentation over chat history. If a step is missing or ambiguous, say so; do not invent process.
- If the release includes CRD changes: never remove fields, change types, or make optional fields required ([AGENTS.md](../../AGENTS.md)).
- Do not commit secrets. Do not log tokens or passwords in examples.

## Related documentation (all repos)

### This repository (`koku-metrics-operator`)

| Doc | Role |
|-----|------|
| [upstream.md](upstream.md) | Upstream release runbook |
| [downstream.md](downstream.md) | Downstream release runbook |
| [cve.md](cve.md) | CVE handling for releases |
| [upstream-release-testing.md](../upstream-release-testing.md) | OLM upgrade testing (detail) |
| [csv-description.md](../csv-description.md) | CSV text / “New in vX.Y.Z” |
| [AGENTS.md](../../AGENTS.md) | Conventions for humans and agents |
| [architecture.md](../architecture.md) | Design context (not release steps) |
| [local-development.md](../local-development.md) | Local setup context |

### Accessory repositories (link only — do not duplicate runbooks)

| Repo | Role |
|------|------|
| [cost-management-metrics-operator-fbc](https://github.com/project-koku/cost-management-metrics-operator-fbc) | Downstream FBC catalogs, `make catalog`, Gather FBC for QE |
| [community-operators-prod](https://github.com/redhat-openshift-ecosystem/community-operators-prod) | Upstream OperatorHub bundle submission |
| [koku](https://github.com/project-koku/koku) | `operator_versions` mapping after Downstream release |
| [koku-ci](https://github.com/project-koku/koku-ci) | Konflux login / kubeconfig helpers (`koku-ci-management`) |
| [konflux-release-data](https://gitlab.cee.redhat.com/releng/konflux-release-data) | ReleasePlan / RPA / tenant GitOps (platform) |

### External

| Link | Role |
|------|------|
| [Konflux user docs](https://konflux.pages.redhat.com/docs/users/) | Platform builds and releasing |
| [Releasing with an advisory](https://konflux.pages.redhat.com/docs/users/releasing/releasing-with-an-advisory.html) | RHSA / RHBA / RHEA guidance |
| [catalog.redhat.com — CMMO operator](https://catalog.redhat.com/software/containers/costmanagement/costmanagement-metrics-rhel9-operator/67168aeaef33c5ffaff8cd45) | Product image + Security / CVE tab |
| [Operator SDK](https://sdk.operatorframework.io/) | Bundle / OLM background |
| [FBC auto-release (community)](https://redhat-openshift-ecosystem.github.io/operator-pipelines/users/fbc_autorelease/) | `release-config.yaml` for community catalogs |
