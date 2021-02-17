## Generating and testing release bundles

A full overview of community operator testing can be found [here](https://operator-framework.github.io/community-operators/testing-operators/). The following steps were parsed from the testing document and are specific to testing the `koku-metrics-operator`.

### Pre-requisites

* Access to a 4.5+ OpenShift cluster
* opm

To install opm, clone the operator-registry repository:

```
git clone https://github.com/operator-framework/operator-registry
```
Change into the directory and run `make build`. This will generate an `opm` executable in the `operator-registry/bin/` directory. Add the bin to your path, or substitute `opm` in the following commands with the full path to your `opm` executable.


### Testing an operator upgrade - Overview

Testing an upgrade is a complicated and an involved process. The general steps are as follows:
1. Create an opm index (test-catalog) for the most recent release and push to Quay (quay.io/project-koku/kmc-test-catalog contains previous versions and can be used instead of creating your own initial catalog).
2. Create a CatalogSource in OCP, and install the operator (this should be the version that will be upgraded).
3. Generate the new controller image, and push to Quay.
4. Generate the new bundle.
5. Build the bundle image and push to Quay.
6. Create an opm index (test-catalog) that contains the last release and the new release, and push to Quay.
7. Update the CatalogSource in OCP to point to the new test-catalog image.
8. Observe the installed operator, and ensure it upgrades automatically.


### Testing an operator upgrade

1. Check `quay.io/project-koku/kmc-test-catalog` for the most recent operator release. If the index does not exist, create it:

Check `quay.io/project-koku/koku-metrics-operator-bundle` for the most recent operator release bundle:

```sh
$ cd koku-metrics-operator/$PREVIOUS_VERSION
$ docker build -f Dockerfile . -t quay.io/project-koku/koku-metrics-operator-bundle:v$PREVIOUS_VERSION; docker push quay.io/project-koku/koku-metrics-operator-bundle:v$PREVIOUS_VERSION
```

Use `opm` to build a catalog image with the koku-metrics-operator and then push the image:

```sh
$ opm index add --bundles quay.io/project-koku/koku-metrics-operator-bundle:v$PREVIOUS_VERSION --tag quay.io/project-koku/kmc-test-catalog:v$PREVIOUS_VERSION --container-tool docker
$ docker push quay.io/project-koku/kmc-test-catalog:v$PREVIOUS_VERSION
```

2. Create a CatalogSource in OCP, and install the operator (this should be the version that will be upgraded):
Create a catalog source by copying the following into a file called `catalog-source.yaml`:

```sh
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: my-test-catalog
  namespace: openshift-marketplace
spec:
  sourceType: grpc
  image: quay.io/project-koku/kmc-test-catalog:v$PREVIOUS_VERSION
  updateStrategy:
    registryPoll:
      interval: 3m
```

Deploy the catalog source to the cluster:

```sh
$ oc apply -f catalog-source.yaml
```

Verify that the catalog source was created without errors in the `openshift-marketplace` project.

Search OperatorHub for the koku-metrics-operator. It should be available under the `custom` or `my-test-catalog` provider type depending on your OCP version.

Install the koku-metrics-operator in the `koku-metrics-operator` namespace, and test as normal.


3. Generate the new controller image, and push to Quay:

```sh
$ USERNAME=<quay-username>
$ VERSION=<release-version>
$ docker build . -t quay.io/$USERNAME/koku-metrics-operator:v$VERSION
$ docker push quay.io/$USERNAME/koku-metrics-operator:v$VERSION
```

4. Generate the new bundle:

Update the release versions at the top of the `Makefile` to match the release version of the operator:

```
# Current Operator version
PREVIOUS_VERSION ?= <version to upgrade from>
VERSION ?= <release-version>
```

Run the following command to generate the bundle:

```sh
$ make bundle DEFAULT_CHANNEL=alpha
```

This will generate a new `<release-version>` bundle inside of the `koku-metrics-operator` directory within the repository.

5. Build the bundle image and push to Quay:

Copy the generated bundle to the testing directory:

```sh
$ cp -r koku-metrics-operator/$VERSION testing
$ cd testing/$VERSION
```

Change the container images to the test controller image that was generated in step 3:

Search and replace `quay.io/project-koku/koku-metrics-operator:v$VERSION` with `quay.io/$USERNAME/koku-metrics-operator:v$VERSION` in the clusterserviceversion.

Build and push:

```sh
$ docker build -f Dockerfile . -t quay.io/$USERNAME/koku-metrics-operator-bundle:v$VERSION; docker push quay.io/$USERNAME/koku-metrics-operator-bundle:v$VERSION
```

6. Create an opm index (test-catalog) that contains the last release and the new release, and push to Quay:

```sh
$ opm index add --from-index quay.io/project-koku/kmc-test-catalog:v$PREVIOUS_VERSION --bundles quay.io/$USERNAME/koku-metrics-operator-bundle:v$VERSION --tag quay.io/$USERNAME/test-catalog:latest --container-tool docker
$ docker push quay.io/$USERNAME/test-catalog:v$VERSION
```

7. Update the image in the CatalogSource to point to the new test-catalog version (i.e. `image: quay.io/$USERNAME/test-catalog:v$VERSION`).
8. Observe the installed operator, and ensure it upgrades automatically.

Once the operator has upgraded, run through manual tests as normal.
