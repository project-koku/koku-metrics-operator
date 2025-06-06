# Releasing a new version of the Koku Metrics Operator

This guide outlines the steps for releasing a new version of the Koku Metrics Operator, from creating a github release to submitting the bundle to the `community-operators-prod` repository for File-Based Catalog (FBC) auto-release.

Before proceeding, ensure all necessary testing has been performed as described in the [release-testing.md](release-testing.md) documentation.


## Create a github release and push the operator image

First, create a github release that corresponds to the new operator version. You can use [previous releases](https://github.com/project-koku/koku-metrics-operator/releases) as a template.

Make sure the `VERSION` variable at the top of your operator's `Makefile` matches the release version:

```makefile
# Current operator version
VERSION ?= <release-version>
```

After creating the github release and tag, a Quay hook should automatically pull and tag the new image. Verify this by checking for the new tag in the [Quay.io repository](https://quay.io/repository/project-koku/koku-metrics-operator?tab=tags). If the tag doesn't appear, you'll need to manually build and push the image:

```bash
make docker-build
make docker-push
```



## Generate the release bundle

**Note:** Ensure the operator image is available locally or pulled to your system so that the `operator-sdk` can correctly embed its reference within the bundle's manifests.

Run the following command to generate the OLM bundle for the new operator version:

```bash
make bundle CHANNELS=alpha,beta DEFAULT_CHANNEL=beta
```

This command generates a new bundle inside the `bundle/` directory within your repository.


## Submit the Generated bundle to `community-operators-prod`

After the bundle is generated, you need to contribute it to the `community-operators-prod` repository.

### 1. Fork, clone, and copy the bundle

1. Start by forking the [community-operators-prod repository](https://github.com/redhat-openshift-ecosystem/community-operators-prod/tree/main) and cloning your fork locally.
2. Create a new branch for your changes. For example `koku-metrics-operator-v4.0.0`.
3. Copy the contents of the generated `bundle/` directory into a new version-specific directory within your cloned fork. The path should be `community-operators-prod/operators/koku-metrics-operator/<VERSION>/`.
 For example, if you're releasing `version 4.0.0`, the directory structure in your `community-operators-prod` fork would look like this:

    ```
    community-operators-prod/
    └── operators/
        └── koku-metrics-operator/
            ├── 3.3.2/
            │   ├── manifests/
            │   │   ├── koku-metrics-cfg.openshift.io_kokumetricsconfigs.yaml
            │   │   └── koku-metrics-operator.clusterserviceversion.yaml
            │   └── metadata/
            │       └── annotations.yaml
            └── 4.0.0/  <-- New bundle Directory
                ├── manifests/
                │   ├── costmanagement-metrics-cfg.openshift.io_costmanagementmetricsconfigs.yaml
                │   └── koku-metrics-operator.clusterserviceversion.yaml
                ├── metadata/
                │   └── annotations.yaml
    ```

### 2. Configure File-Based Catalog (FBC) auto-release

To enable the auto-release feature for the File-Based Catalogs (FBCs), you must add a `release-config.yaml` file to the bundle directory. For more information, refer to the [File-Based Catalog auto-release documentation](https://redhat-openshift-ecosystem.github.io/operator-pipelines/users/fbc_autorelease/).

1. Create a file named `release-config.yaml` directly inside the new version directory. For example `community-operators-prod/operators/koku-metrics-operator/4.0.0/`.

    ```
    community-operators-prod/
    └── operators/
        └── koku-metrics-operator/
            └── 4.0.0/
                ├── manifests/
                ├── metadata/
                └── release-config.yaml  <-- New file
    ```

2. The contents of the `release-config.yaml` should be similar to the example below. This file tells the FBC generation automation how to include this bundle in specific channels and manage upgrade paths.

    ```yaml
    ---
    catalog_templates:
      - template_name: basic.yaml
        channels: [beta, alpha] # list of channels this bundle should be available in.
        replaces: koku-metrics-operator.v3.3.2 # the bundle this new version replaces in these channels.
    ```

### 3. Create the operator bundle pull request

Finally, commit your changes, sign the commit, and push your branch to your fork. Then, open a pull request against the main `redhat-openshift-ecosystem/community-operators-prod` repository.

```bash
git commit -s -m "<commit-message>"
git push origin your-branch-name
```

Once pushed, open a pull and complete the checklist provided. For an example, you can refer to [redhat-openshift-ecosystem/community-operators-prod#6824](https://github.com/redhat-openshift-ecosystem/community-operators-prod/pull/6824).

After your pull request merges, a new pull request will automatically be generated to update the catalog with the new bundle for all the supported OCP versions similar to [redhat-openshift-ecosystem/community-operators-prod#6825](https://github.com/redhat-openshift-ecosystem/community-operators-prod/pull/6825). Once this subsequent FBC pull request merges, the new version of the community release will be pushed out to the OperatorHub.
