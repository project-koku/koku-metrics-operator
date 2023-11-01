1. Create the `downstream-vX.Y.Z` branch based off main:
```
git fetch origin
git switch --no-track -c downstream-vX.Y.Z origin/main
git push downstream-vX.Y.Z
```

2. Branch `downstream-vX.Y.Z` so we can make the updates for the downstream code:

When generating the downstream code, we must vendor the packages. This directory is very large and makes reviewing the PR in Github difficult. To make the review process easier, we should separate the code changes and the package vendoring into separate commits:

a. Checkout a branch that will be merged into the `downstream-vX.Y.Z` branch:
```
$ git checkout -b make-downstream-vX.Y.X (be sure to substitute the correct version for x.y.z, e.g. 2.0.0)
```

b. Generate the code changes:
```
$ make downstream
```

c. Vendor the packages:
```
$ make downstream-vendor
```

d. Add/commit/push:
```
$ git add/commit/push
```

3. Open PR against `downstream-vX.Y.Z` to merge the downstream code changes.
