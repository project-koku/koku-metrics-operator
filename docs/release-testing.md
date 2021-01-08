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
1. Create an opm index (test-catalog) for the most recent release and push to Quay.
2. Create a CatalogSource in OCP, and install the operator (this should be the version that will be upgraded).
3. Generate the new controller image, and push to Quay.
4. Generate the new bundle.
5. Build the bundle image and push to Quay.
6. Create an opm index (test-catalog) that contains the last release and the new release, and push to Quay.
7. Check that the test-catalog in OCP contains the new version.
8. Observe the installed operator, and ensure it upgrades automatically.


### Testing an operator upgrade

1. Create an opm index (test-catalog) for the most recent release and push to Quay:

Copy the previous release bundle to the testing directory:

```sh
$ PREVIOUS_VERSION=<version to upgrade from>
$ cp -r koku-metrics-operator/$PREVIOUS_VERSION testing
$ cd testing/$PREVIOUS_VERSION
```

Build the bundle and push to quay:

```sh
$ docker build -f Dockerfile . -t quay.io/$USERNAME/koku-metrics-operator-bundle:v$VERSION; docker push quay.io/$USERNAME/koku-metrics-operator-bundle:v$VERSION
```

Use `opm` to build a catalog image with the koku-metrics-operator and then push the image:

```sh
$ opm index add --bundles quay.io/$USERNAME/koku-metrics-operator-bundle:v$PREVIOUS_VERSION --tag quay.io/$USERNAME/test-catalog:latest --container-tool docker
$ docker push quay.io/$USERNAME/test-catalog:latest
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
  image: quay.io/$USERNAME/test-catalog:latest
  updateStrategy:
    registryPoll:
      interval: 1m
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
$ opm index add --bundles quay.io/$USERNAME/koku-metrics-operator-bundle:v$PREVIOUS_VERSION,quay.io/$USERNAME/koku-metrics-operator-bundle:v$VERSION --tag quay.io/$USERNAME/test-catalog:latest --container-tool docker
$ docker push quay.io/$USERNAME/test-catalog:latest
```

7. Check that the test-catalog in OCP contains the new version.
8. Observe the installed operator, and ensure it upgrades automatically.

Now the upgraded operator should be tested.
