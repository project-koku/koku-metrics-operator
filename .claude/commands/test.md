Write tests for $ARGUMENTS using the existing style in the touched package.

Requirements:
- If the package uses Ginkgo, use `Describe`, `Context`, `It` blocks with descriptive names and Gomega matchers (`Expect`, `Eventually`, `Consistently`)
- If the package uses stdlib tests, use idiomatic `testing` patterns and table-driven tests where helpful
- Mock external dependencies (HTTP, filesystem, Kubernetes API)
- Cover success path, error conditions, and edge cases (nil, empty, not found)
- Use shared setup/cleanup patterns that match the package (`BeforeEach`/`AfterEach` for Ginkgo, helper functions or `t.Cleanup` for stdlib tests)
- Place tests in `*_test.go` in the same package
- Use nearby existing tests in the same package as the primary reference style

After writing, run `make test` to verify they pass.
