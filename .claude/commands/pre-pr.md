Run the pre-PR checklist for this branch:

1. `make fmt` -- format code
2. `make lint` -- run linters
3. `make test` -- run all tests
4. `make verify-manifests` -- check if CRD/RBAC manifests are up to date
5. `git status` -- check for uncommitted changes
6. Verify no secrets, tokens, or credentials in staged files
7. If CRD types were modified, confirm backward compatibility (no removed fields, no type changes)

Report the results and flag any failures.
