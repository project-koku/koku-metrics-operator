# Contributing to koku-metrics-operator

Thank you for your interest in contributing to the koku-metrics-operator! This guide covers both traditional and AI-assisted development workflows.

## Quick Links

- **[Local Development Setup](local-development.md)** - How to run the operator locally
- **[Architecture](architecture.md)** - System design and component overview
- **[AI Development Context](../.claude/CLAUDE.md)** - For AI agents and Claude Code users
- **[Cursor Rules](../.cursor/.cursorrules)** - Quick reference for Cursor IDE
- **[Release Process](upstream-releasing.md)** - How to cut a release

## Before You Start

### Developer Certificate of Origin (DCO)

By contributing to this project, you agree to the Developer Certificate of Origin (DCO). See the [CONTRIBUTING](../CONTRIBUTING) file for details.

### Understand the Repository Structure

This repository has **two versions** across different branches:

- **main (upstream)**: Community version, develops features first
- **downstream**: Red Hat product version, receives ported changes

**Always develop on upstream (main) first** unless working on downstream-specific requirements (FIPS, Red Hat registries, etc.).

**Check your version:**
```bash
# Look for CI directories
ls -d .github/workflows/ 2>/dev/null && echo "Upstream"
ls -d .tekton/ 2>/dev/null && echo "Downstream"
```

## Getting Started

### 1. Fork and Clone

```bash
git clone https://github.com/<your-username>/koku-metrics-operator.git
cd koku-metrics-operator
```

### 2. Set Up Development Environment

**Prerequisites:**
- Go 1.13+ (see `go.mod` for current version)
- OpenShift 4.5+ cluster access
- kubectl/oc CLI
- Docker or Podman
- operator-sdk CLI

**Install pre-commit hooks:**
```bash
pre-commit install
```

**Prevent accidental commits to kustomization.yaml:**
```bash
git update-index --assume-unchanged config/manager/kustomization.yaml
```

See [local-development.md](local-development.md) for detailed setup instructions.

### 3. Create a Branch

```bash
git checkout -b feature/your-feature-name
```

**Branch naming conventions:**
- `feature/` - New features
- `fix/` - Bug fixes
- `docs/` - Documentation changes
- `refactor/` - Code refactoring

## Development Workflows

### Traditional Development

**Standard workflow:**
1. Read existing code in the area you're modifying
2. Write failing tests first (TDD)
3. Implement the feature/fix
4. Ensure tests pass: `make test`
5. Format and lint: `make fmt && make lint`
6. Update documentation if needed

**Code conventions:**
- Follow standard Go formatting (`gofmt`)
- Use structured logging with `zap`
- Handle all errors (never ignore)
- Keep functions focused and testable
- Write tests using Ginkgo BDD style

### AI-Assisted Development

**If using Claude Code, Cursor, or other AI coding assistants:**

1. **AI agents have context** from `.claude/CLAUDE.md` and `.cursor/.cursorrules`
2. **Let AI explore** the codebase before suggesting changes
3. **AI understands** upstream/downstream differences, testing patterns, and conventions
4. **Review AI changes carefully** - AI is a smart assistant, not a replacement for code review

**AI-friendly workflows in this repo:**
- Ask AI to explain architecture: "How does metric collection work?"
- Request debugging help: "Why am I getting 403 from Prometheus?"
- Generate tests: "Write Ginkgo tests for this function"
- Refactor code: "Simplify this reconciliation logic"

**What AI knows about this project:**
- Testing patterns (Ginkgo/Gomega, mocking strategies)
- Common debugging issues (RBAC, Prometheus connectivity)
- Release workflows (bundle generation, upstream vs downstream)
- File structure and component relationships

See [../.claude/CLAUDE.md](../.claude/CLAUDE.md) for comprehensive AI development context.

## Making Changes

### Code Guidelines

**Go Standards:**
- Use `go fmt` for formatting
- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Error wrapping: `fmt.Errorf("context: %w", err)`
- No panics in production code

**Kubernetes Operator Patterns:**
- Controllers must be idempotent
- Always check if resources exist before creating
- Use finalizers for cleanup logic
- Return appropriate `ctrl.Result` from reconciliation
- Handle `NotFound` errors gracefully

**Import organization:**
```go
import (
    // Standard library
    "context"
    "fmt"

    // External packages (alphabetical)
    "github.com/prometheus/client_golang/api"
    corev1 "k8s.io/api/core/v1"

    // Internal packages (alphabetical)
    "github.com/project-koku/koku-metrics-operator/api/v1beta1"
    "github.com/project-koku/koku-metrics-operator/internal/collector"
)
```

### Testing Requirements

**All changes must include tests.**

**Test structure:**
- Unit tests: Pure functions, business logic
- Integration tests: Controller reconciliation with fake client
- Mock external dependencies (HTTP, filesystem, Kubernetes API)

**Ginkgo patterns:**
```go
var _ = Describe("ComponentName", func() {
    Context("when doing something", func() {
        BeforeEach(func() {
            // Setup
        })

        It("should behave correctly", func() {
            result := DoSomething()
            Expect(result).To(Equal(expected))
        })

        AfterEach(func() {
            // Cleanup
        })
    })
})
```

**Run tests:**
```bash
make test                  # Run all tests
make clean-test-cache      # Clear test cache
```

**Coverage expectations:**
- All new code should have tests
- Critical paths must be covered
- Test both success and failure cases

### Common Contribution Scenarios

#### Adding New Metrics

**Files to modify:**
1. `internal/collector/queries.go` - Add Prometheus query
2. `internal/collector/collector.go` - Add file prefix constant, update report generation
3. `internal/collector/report.go` - Define CSV structure
4. `internal/collector/collector_test.go` - Add tests
5. `docs/report-fields-description.md` - Document fields

**Pattern:**
- Query Prometheus for time range (UTC hour boundaries)
- Aggregate data (max/min/avg/sum)
- Generate CSV with prefix: `cm-openshift-<type>-usage-*.csv`
- Test with mocked Prometheus responses

See [architecture.md](architecture.md#3-prometheus-collector) for details.

#### Fixing Reconciliation Issues

**Key files:**
- `internal/controller/costmanagementmetricsconfig_controller.go` - Main logic
- `internal/controller/prometheus.go` - Prometheus client setup

**Common issues:**
- RBAC: Ensure ServiceAccount has `cluster-monitoring-view`
- Authentication: Check token/secret validity
- Timing: Collection happens on schedule, not immediately

**Debugging:**
1. Check controller logs: `oc logs -n koku-metrics-operator deployment/...`
2. Inspect CR status: `oc get metricsconfig -o yaml`
3. Verify Prometheus connectivity

#### Updating CRD Schema

**CRITICAL: CRD changes can break existing clusters.**

1. Update `api/v1beta1/metricsconfig_types.go`
2. Add kubebuilder markers for validation
3. Run `make manifests` to regenerate CRD
4. Update default values if needed
5. Test backward compatibility
6. Document in PR description

**Never:**
- Remove fields (deprecated fields can stay)
- Change field types
- Make optional fields required

### Documentation

**When to update docs:**
- New features: Update relevant doc + architecture.md
- Bug fixes: Update troubleshooting in docs if pattern emerges
- API changes: Update CRD examples
- New workflows: Add to contributing.md

**Documentation files:**
- `README.md` - High-level overview
- `docs/architecture.md` - System design
- `docs/local-development.md` - Setup and testing
- `docs/report-fields-description.md` - CSV field definitions
- `.claude/CLAUDE.md` - AI development context

## Submitting Changes

### Before Creating a Pull Request

**Checklist:**
```bash
# 1. Ensure tests pass
make test

# 2. Format code
make fmt

# 3. Run linters and pre-commit hooks
make lint

# 4. Verify manifests are up-to-date (if CRD/RBAC changed)
make verify-manifests

# 5. Check git status
git status
```

**Do NOT:**
- Modify `vendor/` directory manually (use `make vendor`)
- Skip pre-commit hooks (unless absolutely necessary)
- Commit secrets or credentials
- Change downstream files when on upstream

### Pull Request Guidelines

**PR Title:**
- Use conventional commit style: `feat:`, `fix:`, `docs:`, `refactor:`
- Keep concise (under 70 characters)
- Example: `feat: add GPU metrics collection for NVIDIA cards`

**PR Description:**

Include:
1. **What** - What does this PR do?
2. **Why** - Why is this change needed?
3. **How** - Brief explanation of approach
4. **Testing** - How was this tested?
5. **Checklist** - Did you run tests, linting, etc.?

**Template:**
```markdown
## Description
Brief description of changes

## Motivation
Why is this change needed?

## Changes
- Added X
- Modified Y
- Fixed Z

## Testing
- [ ] Unit tests added/updated
- [ ] Integration tests pass
- [ ] Manually tested on OpenShift cluster
- [ ] Verified backward compatibility

## Checklist
- [ ] Code formatted (`make fmt`)
- [ ] Linters pass (`make lint`)
- [ ] Tests pass (`make test`)
- [ ] Documentation updated
- [ ] No secrets committed
```

### Review Process

**What to expect:**
1. Automated CI runs (tests, linting, build)
2. Code review from maintainers
3. Potential requested changes
4. Approval and merge

**Review criteria:**
- Code quality and style
- Test coverage
- Backward compatibility
- Documentation completeness
- Security considerations

**Be responsive:**
- Address review comments promptly
- Ask questions if feedback is unclear
- Push new commits (don't force-push unless requested)

## Development Best Practices

### Debugging Tips

**Common issues:**
| Problem | Solution |
|---------|----------|
| `403 from Prometheus` | Grant cluster-monitoring-view to ServiceAccount |
| `CRD not found` | Run `make install` |
| Tests fail with nil pointer | Check mock initialization in BeforeEach |
| Build fails with missing package | Run `make vendor` |

**Useful commands:**
```bash
# Check operator logs
oc logs -n koku-metrics-operator deployment/koku-metrics-operator-controller-manager -f

# Get CR status
oc get metricsconfig -o yaml

# Check RBAC
oc adm policy who-can get prometheuses.monitoring.coreos.com

# Run specific test
go test ./internal/collector -run TestGenerateReports
```

### Security Considerations

**Always:**
- Validate user input from CRDs
- Handle secrets securely (never log)
- Use RBAC appropriately
- Validate file paths (prevent traversal)

**Never:**
- Commit secrets, tokens, or credentials
- Log sensitive data (passwords, tokens)
- Execute arbitrary commands without validation
- Skip input validation

### Performance

**Keep in mind:**
- Prometheus queries are read-only, minimal cluster impact
- Collection happens hourly, not continuous
- Report size grows with cluster size
- Storage limits prevent unbounded growth

**Optimize:**
- Efficient Prometheus queries (avoid unbounded time ranges)
- Reasonable aggregation windows
- Clean up old reports

## Getting Help

**Questions?**
- Check [docs/faq.md](faq.md) for common questions
- Review [architecture.md](architecture.md) for system design
- Ask in GitHub Discussions or issues

**Found a bug?**
- Check existing issues: https://issues.redhat.com/projects/COST/
- Provide reproduction steps
- Include operator logs and CR status

**Need to discuss a feature?**
- Open an issue first for discussion
- Explain use case and proposed approach
- Get feedback before implementing

## Release Process

**Releasing is handled by maintainers.**

If you're a maintainer preparing a release:
- See [upstream-releasing.md](upstream-releasing.md) for detailed steps
- Update VERSION in Makefile
- Generate bundle: `make bundle`
- Submit to community-operators-prod

**Downstream releases:**
- See [downstream-releasing.md](downstream-releasing.md)
- Managed by Red Hat Konflux
- Changes ported from upstream

## Code of Conduct

**Be respectful:**
- Assume good intentions
- Provide constructive feedback
- Welcome newcomers
- Focus on the code, not the person

**This is a Red Hat project:**
- Follow Red Hat's community guidelines
- Changes may affect production clusters
- Prioritize stability and backward compatibility

## Additional Resources

### Documentation
- [Local Development](local-development.md)
- [Architecture](architecture.md)
- [FAQ](faq.md)
- [Report Fields](report-fields-description.md)
- [Upstream Releasing](upstream-releasing.md)
- [Downstream Releasing](downstream-releasing.md)

### AI Development
- [Claude Code Context](../.claude/CLAUDE.md)
- [Cursor Rules](../.cursor/.cursorrules)

### External
- [Operator SDK Documentation](https://sdk.operatorframework.io/)
- [Kubernetes Operator Patterns](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)
- [Ginkgo Testing Framework](https://onsi.github.io/ginkgo/)

## Thank You!

Your contributions help make cost management better for the OpenShift community. Whether you're fixing a typo, adding a feature, or helping with AI-assisted development workflows, your work is appreciated!

**Happy coding!** 🚀
