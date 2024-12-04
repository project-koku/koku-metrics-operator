## Releasing a new version of the koku-metrics-operator

Before releasing a new version of the operator, testing should be performed as described [here](release-testing.md).


### Create a github release and push the operator image
Create a GitHub release that corresponds with the operator release version. The [previous releases](https://github.com/project-koku/koku-metrics-operator/releases) can be used as a template.

Update the release version at the top of the `Makefile` to match the release version of the operator:

```
# Current Operator version
VERSION ?= <release-version>
```

> After creating the GitHub release/tag, a Quay hook should pull the image. Check the [quay repo](https://quay.io/repository/project-koku/koku-metrics-operator?tab=tags) and ensure the new tag was pulled. If the tag does not exist, the following should be run to build and push the image:
> ```
> make docker-build
> make docker-push
> ```

### Generate the release bundle
Run the following command to generate the release bundle:

```
make bundle CHANNELS=alpha,beta DEFAULT_CHANNEL=beta
```
This will generate a new bundle inside of the `bundle/` directory within the repository.

Once the release bundle has been generated, fork & clone the [community-operators-prod repository](https://github.com/redhat-openshift-ecosystem/community-operators-prod/tree/main). Create a branch, and copy the generated bundle to the `community-operators-prod/operators/koku-metrics-operator/<VERSION>/` directory in your cloned fork.

For example, if the bundle was generated for a `1.0.0` release, the directory structure would look like the following:

```
koku-metrics-operator/
├── 0.9.0
│   ├── manifests
│   │   ├── koku-metrics-cfg.openshift.io_kokumetricsconfigs.yaml
│   │   └── koku-metrics-operator.clusterserviceversion.yaml
│   ├── metadata
│   │   └── annotations.yaml
├── 1.0.0
│   ├── manifests
│   │   ├── koku-metrics-cfg.v1.0.0.openshift.io_kokumetricsconfigs.yaml
│   │   └── koku-metrics-operator.v1.0.0.clusterserviceversion.yaml
│   ├── metadata
│   │   └── annotations.yaml
```

### Create the operator-bundle pull-request
Commit, sign, and push the branch to the fork of the community-operators repo. Once pushed, open a PR against the community-operators repo and fill out the resulting checklist:

```
git commit -s -m "<commit-message>"
git push origin branch
```

Example PR: [redhat-openshift-ecosystem/community-operators-prod#5587](https://github.com/redhat-openshift-ecosystem/community-operators-prod/pull/5587)

Once this PR merges, a pipeline will kick off to generate the bundle. When the bundle is generated, the FBC needs to be updated to push out a release.

### Update the File Based Catalog (FBC)

A few make commands are available in the community-operators koku-metrics-operator directory. First, update the version at the top of the file:
```
PREVIOUS_VERSION ?= 3.3.1
VERSION ?= 3.3.2
```

The gather the new bundle pullspec. The following command will pull the latest bundle for the defined `VERSION` and output the sha of the pullspec:

```
$ make get-bundle
docker pull quay.io/community-operator-pipeline-prod/koku-metrics-operator:3.3.2
3.3.2: Pulling from community-operator-pipeline-prod/koku-metrics-operator
Digest: sha256:9114f72f6adca60e18616786019d680883bb1c1dc88d317e7adeeec787054469
Status: Image is up to date for quay.io/community-operator-pipeline-prod/koku-metrics-operator:3.3.2
quay.io/community-operator-pipeline-prod/koku-metrics-operator:3.3.2
docker inspect --format '{{.RepoDigests}}' quay.io/community-operator-pipeline-prod/koku-metrics-operator:3.3.2
[quay.io/community-operator-pipeline-prod/koku-metrics-operator@sha256:9114f72f6adca60e18616786019d680883bb1c1dc88d317e7adeeec787054469] <<<<
```

Copy the `PULLSPEC` and update the value in the Makefile:
```
PULLSPEC ?= quay.io/community-operator-pipeline-prod/koku-metrics-operator@sha256:9114f72f6adca60e18616786019d680883bb1c1dc88d317e7adeeec787054469
```

Then add the new version to the catalog template:
```
make add-new-version
```

Inspect the template to ensure the `name` and `replaces` are correct for both the `alpha` and `beta` channels, and ensure the `olm.bundle` image has been added:
```
...
      - name: koku-metrics-operator.v3.3.1
        replaces: koku-metrics-operator.v3.3.0
      - name: koku-metrics-operator.v3.3.2       <<<<
        replaces: koku-metrics-operator.v3.3.1   <<<<
    name: alpha
...
      - name: koku-metrics-operator.v3.3.1
        replaces: koku-metrics-operator.v3.3.0
      - name: koku-metrics-operator.v3.3.2       <<<<
        replaces: koku-metrics-operator.v3.3.1   <<<<
    name: beta
...
  - image: quay.io/community-operator-pipeline-prod/koku-metrics-operator@sha256:5c501ae285fe463608c4a2ca2d58bd9bb96b8faee7caef1076eee607eeaeb664
    schema: olm.bundle
  - image: quay.io/community-operator-pipeline-prod/koku-metrics-operator@sha256:9114f72f6adca60e18616786019d680883bb1c1dc88d317e7adeeec787054469  <<<<
    schema: olm.bundle                                                                                                                             <<<<
schema: olm.template.basic
```

Next, create and validate the catalogs:
```
$ make catalogs

...
v4.12 catalog validation passed
v4.13 catalog validation passed
v4.14 catalog validation passed
v4.15 catalog validation passed
v4.16 catalog validation passed
v4.17 catalog validation passed
```

Finally, create a PR in the community-operators-prod repo with the Makefile, catalog template, and catalogs changed above.

Example PR: [redhat-openshift-ecosystem/community-operators-prod#5588](https://github.com/redhat-openshift-ecosystem/community-operators-prod/pull/5588)

Once this PR is merged and the pipeline runs on the merged commit, the latest version of the operator will become available.
