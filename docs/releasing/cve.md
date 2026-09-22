# CVE Handling for Operator Releases

> **Start here first:** [start-here.md](start-here.md) · **Ship the fix:** [downstream.md](downstream.md)
>
> **Red Hat product CVE remediation** (UBI, RPMs, FIPS packaging, what customers see on `catalog.redhat.com`) is **Downstream-only**. A merged Downstream fix does **not** reach customers until the Downstream operator image lands on `registry.redhat.io` **and** the FBC catalog release for **every** supported OCP major succeeds.
>
> Upstream may still ship dependency or Go toolchain updates on `main`; that does **not** remediate CVEs for the certified Downstream image customers see on `catalog.redhat.com`. Details: [start-here.md](start-here.md).

## When CVEs drive a release

Ship (or include in) a Downstream release when:

- Image-owner email / Jira security tasks require remediation within SLA (often ~3 weeks from notification)
- Base image, RPM lockfile, and/or Go builder bumps are ready on `downstream`
- The product advisory will typically be **RHSA** if any security fix is included ([downstream.md](downstream.md) Phase C)

Do not block Upstream feature work on Red Hat product CVE tickets. Clearing customer CVE reports for `costmanagement-metrics-rhel9-operator` requires a Downstream release (see callout above).

## How notifications arrive

1. Email to **image owners** (and related contacts) when the product image or its base is affected.
2. Release owner opens Jira tasks under the Downstream epic (for example *Operator: X.Y.Z Downstream Updates*), usually with:
   - Advisory link (`errata.engineering.redhat.com/advisory/…`)
   - Impacted image: `costmanagement/costmanagement-metrics-rhel9-operator`
   - SLA / due context
3. Multiple CVE tasks often ship together in one Downstream version. **Real example:** Downstream **4.4.2** bundled several UBI/RPM CVEs — see operator Release YAML [`releases/4.4.2/…-4.4.2.yaml`](https://github.com/project-koku/cost-management-metrics-operator-fbc/blob/main/releases/4.4.2/costmanagement-metrics-operator-4.4.2.yaml) (`type: RHSA`, many `cves:` entries) and fix PRs [#1005](https://github.com/project-koku/koku-metrics-operator/pull/1005) (coreutils / ubi bump), [#1008](https://github.com/project-koku/koku-metrics-operator/pull/1008) (glibc / ubi bump).

Ensure your contact data is correct in the product/image-owner configuration the platform uses for notifications (ask the release owner if you are not receiving mail).

## The two questions that decide the fix

### 1. Where does the vulnerable package live?

Downstream image build (simplified):

```
OpenShift golang builder
        ↓
ubi-micro (base OS packages, e.g. glibc)
   + ubi layer installing explicit RPMs (e.g. coreutils-single, openssl, …)
        ↓
final operator image
```

| Location | Typical fix |
|----------|-------------|
| Base OS content from **ubi-micro** | Bump `ubi-micro` (and related `ubi`) digests in the Downstream `Dockerfile` |
| Packages listed in **`rpms.in.yaml`** / pinned in **`rpms.lock.yaml`** | Refresh the lockfile (Konflux automation or manual) |
| **Go** toolchain / modules | Bump OpenShift golang **builder** tag and align `go.mod` / toolchain directives |

### 2. Is the package pinned in `rpms.lock.yaml`?

```bash
grep -E 'coreutils|glibc|openssl' rpms.lock.yaml
```

- If present → lockfile (and/or install path) is part of the fix.
- If absent → fix usually comes from the **base image** bump (or latest content pulled at image build time).

## Investigate the advisory

From the Jira card, open the **advisory** link. The **Builds** section lists fixed source RPMs, for example:

```text
coreutils-8.32-41.el9_8
glibc-2.34-272.el9_8
```

The release portion of the NEVRA identifies the fixed build. Confirm binary RPMs / publish dates in UBI content or the Container Catalog when you need a hard cutoff (“images built after date D include the fix”).

## Fix paths

Work on branch **`downstream`**. Use branch names like `cost-XXXX-bump-ubi-…` / `cost-XXXX-refresh-rpms-…`.

### A — Bump UBI base digests

1. Inspect current floating tags and pin **manifest list** digests:

   ```bash
   docker buildx imagetools inspect registry.redhat.io/ubi9/ubi-micro:9.8
   docker buildx imagetools inspect registry.redhat.io/ubi9/ubi:9.8
   ```

2. Update every relevant `FROM …@sha256:…` in the Downstream `Dockerfile` (`ubi-micro` and `ubi` stages as used today).
3. **Open a PR against `downstream`**, request review, and **merge** when green.

Optional: convert ubi-micro tag timestamps / compare Catalog RPM manifests if you must prove the base predates or postdates the advisory publish date.

### B — Refresh RPM lockfile

Konflux / Renovate is often configured to open PRs that refresh `rpms.lock.yaml` (common for OpenSSL and other FIPS-related packages pulled into the micro image).

- Prefer reviewing/merging the **automated PR** when it addresses the advisory (bot already opened it — you still **request review / merge**).
- If automation lags (~1–2 days after “shipped” advisories is common), wait briefly, then refresh manually using the team’s Red Hat subscription / activation-key flow (release owner can walk through; requires VPN and internal Konflux docs) and **open a PR against `downstream`**.

`rpms.in.yaml` declares **which** packages to install; `rpms.lock.yaml` pins **what** got resolved.

### C — Bump Go / OpenShift golang builder

When the advisory is against the builder image or Go toolchain:

1. Find the latest successful **OpenShift golang builder** image for the required stream (internal Red Hat builder catalog / UI; VPN). Prefer **RHEL 9** tags, not RHEL 8.
2. Update Downstream `Dockerfile` builder `FROM` and align `go.mod` (`go` / `toolchain` directives) with the guidance on that builder (minimum vs preferred patch).
3. Expect a large `vendor/` diff when modules move — review carefully; do not hand-edit `vendor/`.
4. **Open a PR against `downstream`**, request review, and **merge**.

## Verify the candidate image

After the Konflux **PR** pipeline succeeds (`downstream-operator-pr-*`):

1. Locate the PR build image in the tenant Quay repo (`redhat-user-workloads` / `costmanagement-metrics-operator`).
2. Compare **Vulnerabilities** (Quay UI) or Catalog Security for old vs new digests — advisory CVEs should clear on the new image.
3. For RPM-level proof, inspect the image’s RPM DB or Catalog RPM manifest for the expected NEVRA.
4. Smoke-test on a cluster when the change is non-trivial (build bundle from the PR image if needed).

## Fix merged ≠ delivered to customers

Delivery definition is in the callout at the top of this page (operator image **and** FBC for every supported OCP).

```
CVE PR merges on downstream → Konflux builds new operator image
        → nudge PR updates to the new digest (may auto-close older nudges)
        → nudge waits for the coordinated release cycle
        → Downstream release (downstream.md Phases A–D)
        → registry.redhat.io + FBC catalogs (all OCP majors)
```

| State | Jira / process |
|-------|----------------|
| Technical fix in git + green PR build | Move task to In Review (or equivalent) |
| Customers actually remediated | Only after Downstream release is **Released** — operator image on `registry.redhat.io` **and** FBC for every supported OCP |

**Do not merge the nudge PR early** just to “finish” a CVE. Nudge is part of the [downstream.md](downstream.md) release train, coordinated with the release owner. When Konflux refreshes a nudge, older nudge PRs are often auto-closed.

## Documenting CVEs in the prod Release YAML

When building Downstream prod YAMLs ([downstream.md](downstream.md) Phase C):

1. Open the **production** catalog image on [catalog.redhat.com](https://catalog.redhat.com/software/containers/costmanagement/costmanagement-metrics-rhel9-operator/67168aeaef33c5ffaff8cd45) → **Security** (shows what still affects the *currently published* image — useful for knowing what customers still see).
2. Cross-check CVE IDs with security tasks on the release epic (`implements` / linked issues). **A Jira security task is the usual source of truth, but not always required** — see [CVE without a Jira ticket](#cve-without-a-jira-ticket) below.
3. Include CVEs **fixed in this version**. Team practice: prioritize **Important** and **Moderate**; confirm whether **Low** is omitted with the release owner.
4. **Before `oc apply` (Phase D):** confirm the **stage** image already carries the fixes — open [catalog.stage.redhat.com](https://catalog.stage.redhat.com/en/software/containers/costmanagement/costmanagement-metrics-rhel9-operator/671692de3eefd9dc50b4f0e2) → **Security**. If stage shows the CVE cleared (or “No unapplied security updates”), the candidate build is good evidence to include that CVE in the prod YAML / proceed to apply.
5. For each CVE entry: `key`, `component` (`costmanagement-metrics-operator`), and `packages` (from catalog / errata).
6. Non-security bugfixes go under `issues.fixed` (COST links) — **not** under `cves`.
7. Do **not** list the release epic ticket itself as a fixed issue.
8. If any CVE is listed → advisory **`type: RHSA`**.

### CVE without a Jira ticket

Most CVEs arrive as epic-linked security tasks. Exception used on **4.4.2**: include a CVE **even without** a dedicated COST ticket when **all** of the following hold:

- The CVE appears as **fixable** / present for the product image on catalog Security (Important/Moderate per team practice), **and**
- The **stage** operator image already shows it remediated (stage Security tab), **and**
- You are shipping the Downstream release that contains that fixed image.

**Real example:** `CVE-2026-0915` (glibc) was listed in the [4.4.2 operator Release YAML](https://github.com/project-koku/cost-management-metrics-operator-fbc/blob/main/releases/4.4.2/costmanagement-metrics-operator-4.4.2.yaml) without a matching Jira security card, after stage catalog confirmed the fix. When in doubt, ask the release owner.

**Real examples to copy from:**

| Release | File | Notes |
|---------|------|-------|
| 4.4.2 | [`…-4.4.2.yaml`](https://github.com/project-koku/cost-management-metrics-operator-fbc/blob/main/releases/4.4.2/costmanagement-metrics-operator-4.4.2.yaml) | Large `cves:` list (openssl, glibc, coreutils, libcap, …) + COST issues; includes `CVE-2026-0915` |
| 4.4.1 | [`…-4.4.1.yaml`](https://github.com/project-koku/cost-management-metrics-operator-fbc/blob/main/releases/4.4.1/costmanagement-metrics-operator-4.4.1.yaml) | Smaller RHSA set |

Example shape (from 4.4.2):

```yaml
cves:
  - key: CVE-2026-5450
    component: costmanagement-metrics-operator
    packages:
      - glibc
  - key: CVE-2026-0915
    component: costmanagement-metrics-operator
    packages:
      - glibc
  - key: CVE-2025-5278
    component: costmanagement-metrics-operator
    packages:
      - coreutils
```

Copy structure from a recent `releases/X.Y.Z/` file in the FBC repo. One advisory may list many CVEs (for example OpenSSL) — include those that apply to **this** operator image build and were addressed in this release, not an unbounded dump of unrelated errata noise. When unsure, align with the release owner and the previous RHSA example.

## QE communication for security-only releases

| Mode | When |
|------|------|
| **Info-only** | Security/dependency-only Downstream release — share CatalogSource images; do not treat Power/Z full cycles as a hard gate without team consensus |
| **Request testing** | Operator functional changes are included |

Still send an early heads-up so ROS / IBM QE can plan ([downstream.md](downstream.md) QE section).

## Checklist

- [ ] Advisory understood; package location identified (base / RPM lock / Go builder)
- [ ] Fix PR against `downstream`; Konflux PR build green
- [ ] Candidate image shows CVE/NEVRA fixed
- [ ] Jira reflects “fix implemented” vs “released to customers”
- [ ] Nudge left for the release cycle (not merged ad hoc)
- [ ] Prod YAML: RHSA + `cves[]` + packages; bugs in `issues.fixed`
- [ ] Stage catalog Security checked before prod apply ([catalog.stage.redhat.com](https://catalog.stage.redhat.com/en/software/containers/costmanagement/costmanagement-metrics-rhel9-operator/671692de3eefd9dc50b4f0e2))
- [ ] Downstream release Phases A–D completed ([downstream.md](downstream.md)) — including FBC for every supported OCP

## Pitfalls

| Pitfall | What to do |
|---------|------------|
| “Fixed in Quay tenant” ≠ fixed for customers | Finish Downstream prod release (operator image **and** FBC for every OCP) |
| Merging nudge to “close” a CVE early | Coordinate with release cycle |
| Omitting a fixed CVE because there is no Jira card | If stage catalog shows it fixed in this build, include it (see [CVE without a Jira ticket](#cve-without-a-jira-ticket); e.g. `CVE-2026-0915` on 4.4.2) |
| Listing CVEs not fixed in **this** build | Cross-check **stage** catalog Security + epic tasks |
| Wrong RHEL stream on builder tags | Prefer RHEL 9 builder tags for this product image |
| Hand-editing `vendor/` | Use `make vendor` / deps PRs only |
| Expecting Upstream merge/release to clear product / UBI CVEs | Does not remediate the certified image — use Downstream (see callout at top) |

## Related links

- [start-here.md](start-here.md)
- [downstream.md](downstream.md)
- [Konflux — releasing with an advisory](https://konflux.pages.redhat.com/docs/users/releasing/releasing-with-an-advisory.html)
- [catalog.redhat.com — CMMO operator (Security tab)](https://catalog.redhat.com/software/containers/costmanagement/costmanagement-metrics-rhel9-operator/67168aeaef33c5ffaff8cd45)
- [catalog.stage.redhat.com — CMMO operator (pre-prod Security check)](https://catalog.stage.redhat.com/en/software/containers/costmanagement/costmanagement-metrics-rhel9-operator/671692de3eefd9dc50b4f0e2)
- [FBC repo `releases/`](https://github.com/project-koku/cost-management-metrics-operator-fbc/tree/main/releases) — prior RHSA YAML examples
- CVE fix PRs: [#1005](https://github.com/project-koku/koku-metrics-operator/pull/1005), [#1008](https://github.com/project-koku/koku-metrics-operator/pull/1008)
- OpenShift golang builder catalog and activation-key RPM refresh docs — internal Red Hat systems (VPN); ask release owner for current bookmarks
