# Claude Code Configuration for koku-metrics-operator

## Project Overview

The **koku-metrics-operator** is a Kubernetes operator built with the Operator SDK that collects OpenShift cluster usage metrics and uploads them to Koku (Red Hat's cost management service). It monitors cluster resources, generates reports, and sends data to console.redhat.com.

**Quick Links:**
- **[Contributing Guide](koku-metrics-operator/docs/contributing.md)** - Contribution guidelines (includes AI-assisted workflows)
- **[Architecture](../koku-metrics-operator/docs/architecture.md)** - Detailed system design and component relationships

### Repository Structure - Upstream & Downstream

This repository maintains **two versions** across different branches:

#### 🌐 main branch - Community Version (Upstream)
- **Name:** `koku-metrics-operator`
- **Purpose:** Open source community version
- **CI/CD:** GitHub Actions (see `.github/workflows/`)
- **Audience:** Community contributors, upstream development

#### 🔴 downstream branch - Red Hat Product Version
- **Name:** `costmanagement-metrics-operator`
- **Purpose:** Red Hat productized, supported version
- **CI/CD:** Konflux with Tekton pipelines (see `.tekton/` directory)
- **Audience:** Red Hat customers, product releases
- **URL:** https://github.com/project-koku/koku-metrics-operator/tree/downstream

**IMPORTANT:** When working on this repository, always verify which branch you're on:
- Use `git branch` to check current branch
- Understand the CI system for your target branch
- Naming conventions differ between upstream and downstream

**Quick Version Check:**
```bash
# Don't rely on branch names alone - check for version indicators:

# Check for upstream indicators
[ -d ".github/workflows" ] && echo "Upstream: koku-metrics-operator, GitHub Actions"

# Check for downstream indicators
[ -d ".tekton" ] && echo "Downstream: costmanagement-metrics-operator, Konflux"

# Or check Dockerfile registry
grep "^FROM" Dockerfile | head -1
```

**Dockerfile Differences:**

The Dockerfile base image registry is a clear indicator of which branch you're on:

**main branch (upstream):**
- Uses public Go images from `docker.io/library/golang`
- Example: `FROM docker.io/library/golang:X.XX.X AS builder`

**downstream branch (Red Hat product):**
- Uses Red Hat's internal brew registry with RHEL-based builders
- Example: `FROM brew.registry.redhat.io/rh-osbs/openshift-golang-builder:rhel_9_golang_X.XX AS builder`

**Key distinction:** upstream = `docker.io`, downstream = `brew.registry.redhat.io`

**Tech Stack:**
- Language: Go (see [go.mod](koku-metrics-operator/go.mod) for current version)
- Framework: Operator SDK / controller-runtime
- Testing: Ginkgo v2 + Gomega
- Platform: OpenShift (4.5 minimum, supports latest)
- Dependencies: Kubernetes, Prometheus, OpenShift APIs

**Key Components:**
- `api/v1beta1/`: Custom Resource Definitions (MetricsConfig CRD)
- `internal/controller/`: Reconciliation logic for Kubernetes resources
- `internal/collector/`: Prometheus metrics collection
- `internal/crhchttp/`: HTTP client for Red Hat console API
- `internal/sources/`: Sources API integration
- `internal/storage/`: Local storage management for reports

## Development Guidelines

### Code Conventions

1. **Go Standards:**
   - Follow standard Go formatting (gofmt, golint)
   - Use structured logging with `go.uber.org/zap`
   - Error handling: always check and wrap errors with context
   - Keep functions focused and testable

2. **Kubernetes Operator Patterns:**
   - Controllers should be idempotent
   - Use reconciliation loops properly (return ctrl.Result)
   - Handle resource deletions with finalizers
   - Respect Kubernetes API conventions (metadata, status, spec)

3. **Testing:**
   - Write tests using Ginkgo BDD style
   - Use Gomega matchers for assertions
   - Mock external dependencies (filesystem, HTTP, Kubernetes clients)
   - Test file location: `*_test.go` in same package
   - Run tests with: `make test`

4. **Error Handling:**
   - Don't panic in production code
   - Log errors with appropriate severity
   - Return errors up the stack with context
   - Use custom error types when appropriate

### Project Structure

```
koku-metrics-operator/
├── api/v1beta1/              # CRD definitions and types
├── cmd/main.go               # Application entry point
├── config/                   # Kubernetes manifests and kustomize
├── internal/
│   ├── controller/          # Reconciliation logic
│   ├── collector/           # Metrics collection from Prometheus
│   ├── crhchttp/           # Red Hat Console HTTP client
│   ├── sources/            # Sources API integration
│   ├── storage/            # Report storage logic
│   ├── packaging/          # Report packaging (tar.gz)
│   └── testutils/          # Testing utilities
└── vendor/                  # Vendored dependencies
```

### Working with This Codebase

**Before Making Changes:**
1. Read the relevant code to understand existing patterns
2. Check for existing tests that cover the area
3. Understand the controller reconciliation flow
4. Review CRD schema in `api/v1beta1/`

**When Adding Features:**
1. Update the CRD if needed (`metricsconfig_types.go`)
2. Implement reconciliation logic in controllers
3. Add/update tests (unit and integration)
4. Update documentation if user-facing
5. Run `make fmt` and `make lint` before committing

**When Fixing Bugs:**
1. Write a failing test that reproduces the bug
2. Fix the code to make the test pass
3. Ensure no regressions with `make test`
4. Add inline comments explaining non-obvious fixes

**Common Tasks:**
- Build operator: `make manager`
- Run tests: `make test`
- Run locally: `make run ENABLE_WEBHOOKS=false`
- Build image: `make docker-build`
- Install CRDs: `make install`

### Dependencies and Imports

**Preferred Packages:**
- Logging: `go.uber.org/zap` (structured logging)
- Testing: `github.com/onsi/ginkgo/v2`, `github.com/onsi/gomega`
- Kubernetes: `k8s.io/api`, `k8s.io/apimachinery`, `k8s.io/client-go`
- Controller: `sigs.k8s.io/controller-runtime`
- OpenShift: `github.com/openshift/api`
- Mocking: `github.com/golang/mock`

**Import Organization:**
1. Standard library
2. External packages (alphabetical)
3. Internal packages (alphabetical)
4. Blank line between groups

### Security Considerations

- **Never commit secrets** or credentials (.env files, tokens, certificates)
- **Validate user input** from CRDs before processing
- **Use RBAC properly** - respect cluster permissions
- **Handle sensitive data** carefully (cluster IDs, authentication tokens)
- **Check for command injection** when executing external commands
- **Validate paths** to prevent directory traversal

### Red Hat / OpenShift Specific

- This operator runs on **OpenShift 4.5+**
- Uses OpenShift APIs (`configv1.ClusterVersion`)
- Integrates with **console.redhat.com** APIs
- Follows Red Hat operator best practices
- Licensed under Apache 2.0

### Testing Philosophy

- **Unit tests** for business logic (pure functions, no I/O)
- **Integration tests** for controller reconciliation
- **Mock external dependencies** (filesystem, HTTP, Kubernetes API)
- **Test edge cases** (nil values, empty lists, errors)
- **Use table-driven tests** for multiple scenarios
- Coverage goal: critical paths must be tested

### CI/CD and Pre-commit

#### Pre-commit Hooks
- **Pre-commit hooks** are configured - install with `pre-commit install`
- **Never skip hooks** unless absolutely necessary
- **Run tests locally** before pushing

#### Continuous Integration

**main branch (upstream):**
- Uses **GitHub Actions** for CI/CD
- Workflows located in `.github/workflows/`
- Runs on PRs: CI, build, publish
- Public GitHub runners

**downstream branch (Red Hat product):**
- Uses **Konflux** build system with Tekton pipelines
- Pipeline definitions in `.tekton/` directory
- Red Hat internal infrastructure
- Different build and release process

**IMPORTANT:** Always verify which branch you're working on before modifying CI configurations!

### Documentation

- **Keep README.md updated** with setup instructions
- **Add godoc comments** to exported functions/types
- **Document controller logic** in reconciliation methods
- **Maintain CHANGELOG** for significant changes

## AI Assistant Behavior

### What to Do
✅ Read existing code before suggesting changes
✅ Follow Go and Kubernetes operator best practices
✅ Write tests for new code
✅ Use existing utilities and helpers when available
✅ Respect the project structure and conventions
✅ Ask for clarification when requirements are unclear
✅ Provide context for non-obvious code changes

### What NOT to Do
❌ Don't create new files unless necessary
❌ Don't modify vendor/ directory
❌ Don't change CRD schema without understanding impact
❌ Don't skip tests ("I'll write tests later")
❌ Don't introduce new dependencies without justification
❌ Don't make breaking changes to public APIs
❌ Don't commit secrets or sensitive data

### When Working on Controllers
- Understand the resource lifecycle (Create, Update, Delete)
- Use finalizers for cleanup logic
- Return proper ctrl.Result for requeueing
- Log state changes for debugging
- Handle not-found errors gracefully
- Respect owner references and garbage collection

### When Working on Tests
- Use descriptive test names (It blocks)
- Group related tests in Context blocks
- Use BeforeEach for common setup
- Clean up resources in AfterEach
- Mock external dependencies
- Test both success and failure paths

## Quick Reference

**Build Commands:**
```bash
make manager          # Build the operator binary
make docker-build     # Build container image
make test            # Run all tests
make fmt             # Format code
make lint            # Run linters
make install         # Install CRDs
make run             # Run locally
```

**File Paths for Common Tasks:**
- Add new controller: `internal/controller/`
- Modify CRD: `api/v1beta1/metricsconfig_types.go`
- Update HTTP client: `internal/crhchttp/`
- Change metrics collection: `internal/collector/`
- Update storage logic: `internal/storage/`

**Important Notes:**
- This is a Red Hat project - follow corporate guidelines
- Operator must be backward compatible with existing CRDs
- Changes may affect production OpenShift clusters
- Test thoroughly before releasing

## Branch-Specific Workflows

### Identifying Your Current Branch Context

**Don't assume branch names!** Instead, check for these indicators:

```bash
# Method 1: Check for CI directories
ls .github/workflows/ 2>/dev/null && echo "Upstream version"
ls .tekton/ 2>/dev/null && echo "Downstream version"

# Method 2: Check Dockerfile registry
grep "^FROM" Dockerfile | grep -q "docker.io" && echo "Upstream"
grep "^FROM" Dockerfile | grep -q "brew.registry.redhat.io" && echo "Downstream"

# Method 3: Check current branch (informational only)
git branch --show-current
```

### Working on Upstream Version
**Indicators:** `.github/workflows/` exists, Dockerfile uses `docker.io`

```bash
# CI uses GitHub Actions
ls .github/workflows/

# Run tests
make test

# Build
make manager
```

### Working on Downstream Version
**Indicators:** `.tekton/` exists, Dockerfile uses `brew.registry.redhat.io`

```bash
# CI uses Konflux/Tekton
ls .tekton/

# Same test/build commands apply
make test
make manager
```

### Common Pitfalls to Avoid
- ❌ Don't assume specific branch names (main, master, downstream, etc.)
- ❌ Don't modify `.github/workflows/` when working on downstream version
- ❌ Don't modify `.tekton/` when working on upstream version
- ❌ Don't mix Dockerfile registries (`docker.io` vs `brew.registry.redhat.io`)
- ❌ Don't assume operator name without checking version context
- ✅ Always check for CI directory indicators (`.github/workflows/` vs `.tekton/`)
- ✅ Verify Dockerfile registry to confirm upstream vs downstream
- ✅ Understand your target audience (community vs Red Hat customers)

## Contact & Support

- **Issues:** https://issues.redhat.com/projects/COST/
- **Repository (main):** https://github.com/project-koku/koku-metrics-operator
- **Repository (downstream):** https://github.com/project-koku/koku-metrics-operator/tree/downstream
- **Documentation:** See `docs/` directory for detailed guides

## Common Workflows

These workflows provide context for typical development tasks. For detailed step-by-step guides, see the `docs/` directory.

### Development Flow Context

**Always develop on upstream first**, then port to downstream.

**Version Indicators:**
```bash
# Identify which version you're working on
ls -d .github/workflows/ 2>/dev/null && echo "Upstream"
ls -d .tekton/ 2>/dev/null && echo "Downstream"
```

**Expected Differences (Intentional, Not Bugs):**
- **Upstream**: `.github/workflows/`, `docker.io` registry, `CGO_ENABLED=0`, distroless images
- **Downstream**: `.tekton/`, `brew.registry.redhat.io`, `CGO_ENABLED=1` (FIPS), UBI images
- **Makefile**: VERSION numbers differ, ENVTEST settings differ (dynamic vs hardcoded)
- **Dockerfile**: Different base images and registries (ported separately)

**When to Work Where:**
- ✅ New features, bug fixes → **upstream first**
- ✅ Red Hat product requirements (FIPS, Konflux) → downstream (after upstream merge)
- ❌ Don't develop features on downstream
- ❌ Don't treat Makefile/Dockerfile differences as bugs during porting

### Debugging Reconciliation Issues

**Common Issues and Quick Fixes:**

| Error | Cause | Fix |
|-------|-------|-----|
| `403 from Prometheus` | Missing cluster-monitoring-view role | `oc adm policy add-cluster-role-to-user cluster-monitoring-view system:serviceaccount:koku-metrics-operator:koku-metrics-controller-manager` |
| `connection refused` | Not logged into cluster | `oc login --token=<token> --server=<server>` |
| `CRD not found` | CRDs not installed | `make install` |
| Token expired | Token needs refresh | `make get-token-and-cert` |

**Key Files to Check:**
- `internal/controller/costmanagementmetricsconfig_controller.go` - Main reconciliation logic
- `internal/controller/prometheus.go` - Prometheus client setup, retention period logic
- `internal/crhchttp/` - HTTP client for console.redhat.com

**Full Setup Guide:** See `docs/local-development.md`

### Adding New Metrics Collection

**Work on upstream first.**

**Files to Modify:**
1. `internal/collector/queries.go` - Prometheus query definitions
2. `internal/collector/collector.go` - File prefixes (e.g., `cm-openshift-*-usage-`), report generation
3. `internal/collector/report.go` - CSV structures and processing logic
4. `internal/collector/collector_test.go` - Ginkgo/Gomega tests with mocked Prometheus responses
5. `docs/report-fields-description.md` - Document new fields

**Key Patterns:**
- Time ranges use UTC, truncated to hour boundaries (see `internal/controller/prometheus.go:86-88`)
- Aggregation methods: `getValue()` supports max, min, avg, sum (see `collector.go:91-100`)
- File prefix constants defined at top of `collector.go`
- Mock Prometheus responses in tests using existing test utilities

**After Upstream Merge:** Changes will be ported to downstream separately.

### Release Workflow

**Upstream Release (Primary):**
- See `docs/upstream-releasing.md` for detailed steps
- Update `VERSION` in Makefile
- Create GitHub release (triggers `.github/workflows/`)
- Generate bundle: `make bundle CHANNELS=alpha,beta DEFAULT_CHANNEL=beta`
- Submit to community-operators-prod

**Downstream Release:**
- See `docs/downstream-releasing.md`
- Managed by Konflux (`.tekton/` pipelines)
- Changes ported from upstream
- VERSION will lag behind upstream (expected)

**Critical Rules:**
- ❌ NEVER manually edit bundle manifests
- ✅ ALWAYS run `make bundle` to regenerate
- ✅ Pull operator image before generating bundle
- ✅ Verify VERSION in Makefile matches release tag

### Fixing Common Test Failures

**Ginkgo Test Patterns:**
- Nil pointer errors → Check `BeforeEach` mock initialization
- Use `Expect(value).NotTo(BeNil())` before dereferencing
- Group related tests in `Context` blocks

**Prometheus Query Tests:**
- Verify query syntax in `queries.go`
- Mock responses must match Prometheus format
- Time ranges: UTC hour boundaries
- Check aggregation method (max/min/avg/sum)

**Controller Tests:**
- Seed fake client with required resources in `BeforeEach`
- Default namespace: `koku-metrics-operator`
- Handle not-found with `apierrors.IsNotFound(err)` pattern

**Build Issues:**
- Package not found → `make vendor`
- ❌ Never manually edit `vendor/` directory

**ENVTEST Version Differences (Expected):**
- Upstream: Dynamic detection from `k8s.io/api`
- Downstream: Hardcoded `1.34.x` for stability
- Both settings are intentional

### Local Development Quick Start

See `docs/local-development.md` for full details.

**Quick Setup:**
```bash
oc new-project koku-metrics-operator
make build && make install
oc apply -f testing/sa.yaml
oc adm policy add-cluster-role-to-user cluster-monitoring-view \
  system:serviceaccount:koku-metrics-operator:koku-metrics-controller-manager
make get-token-and-cert
make run ENABLE_WEBHOOKS=false
make deploy-local-cr AUTH=service-account CLIENT_ID=<id> CLIENT_SECRET=<secret>
```

**Useful Make Targets:**
- `make help` - See all available targets
- `make deploy-local-cr` - Create test CR with external Prometheus route
- `make get-token-and-cert` - Retrieve ServiceAccount token and cluster CA cert
- `make test` - Run full test suite
- `make lint` - Run pre-commit hooks

### CI/CD Modifications

**Default to upstream CI** unless working on downstream-specific build requirements.

**Upstream (GitHub Actions):**
- Location: `.github/workflows/`
- Files: `ci.yaml`, `build-and-publish.yaml`, `ci-manual.yaml`
- Modifiable by contributors

**Downstream (Konflux):**
- Location: `.tekton/`
- Red Hat internal infrastructure
- Changes ported from upstream
- Requires Red Hat maintainer access

**Common Changes:**
- Add tests/linting → Modify upstream `.github/workflows/`
- Change Go version → Update Dockerfile (upstream first)
- Add build steps → Update appropriate CI based on version
