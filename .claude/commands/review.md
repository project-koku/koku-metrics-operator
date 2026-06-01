Review $ARGUMENTS for:

1. **Correctness**: Does the logic match the reconciliation patterns in this codebase? Are errors wrapped with context (`fmt.Errorf("context: %w", err)`)? Are nil pointers handled?
2. **Idempotency**: Will this behave correctly if the reconciler runs it multiple times?
3. **Testing**: Are there Ginkgo tests covering success and failure paths? Are external dependencies mocked?
4. **CRD safety**: If CRD types changed, are they backward compatible? (No removed fields, no type changes, no optional-to-required)
5. **Upstream/downstream**: Are changes appropriate for the current branch? No `.tekton/` changes on upstream, no `.github/workflows/` changes on downstream.

Flag any issues found. Suggest specific fixes with code.
