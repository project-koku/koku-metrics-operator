Identify whether the current working tree is upstream or downstream:

1. Check for CI directory indicators:
   - `.github/workflows/` exists → upstream (koku-metrics-operator)
   - `.tekton/` exists → downstream (costmanagement-metrics-operator)

2. Check the Dockerfile base image registry:
   - `docker.io` → upstream
   - `brew.registry.redhat.io` → downstream

3. Check the current git branch name (informational only -- do not rely on this alone)

Report the results and warn about any files that don't belong on this branch.
