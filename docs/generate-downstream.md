1. open PR to merge `main` into `downstream-base` and update vendored packages (example [PR](https://github.com/project-koku/koku-metrics-operator/pull/182)):
```
git checkout downstream-base
git pull -r
git checkout update-downstream-base
git rebase main
go mod vendor
git add vendor/
git commit -m 'revendor packages'
git push
```
2. have the above PR reviewed and **REBASE** into the `downstream-base`
3. branch `downstream-base` for the target downstream branch and push:
```
git checkout downstream-base
git checkout -b downstream-vX.Y.X (be sure to substitute the correct version for x.y.z, e.g. 2.0.0)
git push
```
4. again, using the newly updated `downstream-base`, make a new branch, and run the `make downstream` command:
```
git fetch origin
git switch --no-track -c new-downstream-vX.Y.X origin/downstream-base
make downstream
git add/commit/push
```
This will update all the necessary pieces of code. Commit these and open a new PR against the `downtream-vX.Y.Z` branch.
