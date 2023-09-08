1. Create the `downstream-vX.Y.Z` branch based off main:
```
git fetch origin
git switch --no-track -c downstream-vX.Y.Z origin/main
git push downstream-vX.Y.Z
```

2. Branch `downstream-vX.Y.Z` so we can make the updates for the downstream code:
```
git checkout -b make-downstream-vX.Y.X (be sure to substitute the correct version for x.y.z, e.g. 2.0.0)
make downstream
git add/commit/push
```
3. Open PR against `downstream-vX.Y.Z` to merge the downstream code changes.
