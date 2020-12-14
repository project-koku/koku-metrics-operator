## Releasing a new version of the koku-metrics-operator

Generate and test a release bundle following the [release testing documentation](release-testing.md). At the end of the documentation, there should be a new <release-version> bundle in a fork of the community operators repo. 

Create a GitHub release that corresponds with the operator release version. The [previous releases](https://github.com/project-koku/koku-metrics-operator/releases) can be used as a template. 

Build and push the operator image to the project-koku quay repository: 

```
make docker-build
make docker-push
```

### Edit the bundle csv to point back to the project-koku image
Search and replace `quay.io/$USERNAME/koku-metrics-operator:$VERSION` with `quay.io/project-koku/koku-metrics-operator:$VERSION` in the generated clusterserviceversion. 

Commit, sign, and push the branch to the fork of the community-operators repo. Once pushed, open a PR against the community-operators repo and fill out the resulting checklist: 

```
git commit -s -m "<commit-message>"
git push origin branch
```
