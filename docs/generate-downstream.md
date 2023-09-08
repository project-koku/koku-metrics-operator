1. open PR to merge `main` into `downstream-base` and update vendored packages (example [PR](https://github.com/project-koku/koku-metrics-operator/pull/182)):
```
git fetch origin
git switch --no-track -c update-downstream-base-v3 origin/downstream-base
git merge origin/main
go mod vendor
```
1. have the above PR reviewed and merge
1. branch `main` for the target downstream branch and push:
```
git checkout main
git checkout -b downstream-VX.Y.X (be sure to substitute the correct version for x.y.z, e.g. 2.0.0)
git push
```
1. using the newly updated `downstream-base`, make a new branch, and run the `make downstream` command:
```
git fetch origin
git switch --no-track -c new-downstream-VX.Y.X origin/downstream-base
make downstream
```
This will update all the necessary pieces of code. Commit these and open a new PR against the `downtream-VX.Y.Z` branch.
