## Releasing a new version of the koku-metrics-operator

Before releasing a new version of the operator, testing should be performed as described [here](release-testing.md).


### Create a github release and push the operator image
Create a GitHub release that corresponds with the operator release version. The [previous releases](https://github.com/project-koku/koku-metrics-operator/releases) can be used as a template. 

Update the release version at the top of the `Makefile` to match the release version of the operator: 

```
# Current Operator version
VERSION ?= <release-version>
```

Build and push the operator image to the project-koku quay repository: 

```
make docker-build
make docker-push
```

### Generate the release bundle 
Run the following command to generate the release bundle: 

```
make bundle DEFAULT_CHANNEL=alpha
```
This will generate a new `<release-version>` bundle inside of the `koku-metrics-operator` directory within the repository. 

Once the release bundle has been generated, fork & clone the [community-operators repository](https://github.com/operator-framework/community-operators). Create a branch, and copy the generated bundle to the `community-operators/community-operators/koku-metrics-operator/` directory in your cloned fork. 

For example, if the bundle was generated for a `1.0.0` release, the directory structure would look like the following: 

```
koku-metrics-operator/
├── 0.9.0
│   ├── manifests
│   │   ├── koku-metrics-cfg.openshift.io_kokumetricsconfigs.yaml
│   │   └── koku-metrics-operator.clusterserviceversion.yaml
│   ├── metadata
│   │   └── annotations.yaml
│   └── Dockerfile
├── 1.0.0
│   ├── manifests
│   │   ├── koku-metrics-cfg.v1.0.0.openshift.io_kokumetricsconfigs.yaml
│   │   └── koku-metrics-operator.v1.0.0.clusterserviceversion.yaml
│   ├── metadata
│   │   └── annotations.yaml
│   └── Dockerfile
```

### Create the release pull-request
Commit, sign, and push the branch to the fork of the community-operators repo. Once pushed, open a PR against the community-operators repo and fill out the resulting checklist: 

```
git commit -s -m "<commit-message>"
git push origin branch
```
The generated release bundle should also be committed to the `koku-metrics-operator` repo. 