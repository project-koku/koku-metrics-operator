## Generating and testing release bundles

### Pre-requisites

* Access to an OpenShift cluster (any currently supported version)

### Testing an operator upgrade - Overview

Reference: https://sdk.operatorframework.io/docs/olm-integration/tutorial-bundle
Openshift comes with OLM enabled, so the "Enable OLM" steps can be omitted.

Testing an upgrade is composed of the following general steps:
1. Create a namespace (koku-metrics-operator)
2. Install the previous bundle (make bundle-deploy-previous)
3. Generate the new controller image, and push to Quay.
4. Generate the new bundle.
5. Build the bundle image and push to Quay.
6. Deploy the upgraded bundle (make bundle-deploy-upgrade)
7. Observe the installed operator, and ensure it upgrades automatically.


### Testing an operator upgrade
0. set version numbers in the Makefile:
```sh
$ PREVIOUS_VERSION=0.9.8
$ VERSION=0.9.9
```

1. Create the namespace in the cluster and deploy the previous bundle:
```sh
$ oc new-project koku-metrics-operator
$ make bundle-deploy-previous
```
Check the `koku-metrics-operator` namespace and ensure the previous version deployed correctly. Create a KokuMetricsConfig so that the PVC is created and data is collected, if available.

2. Generate the new controller image, and push to your Quay repo:

```sh
$ USERNAME=<quay-username>
$ VERSION=<release-version>
$ make docker-buildx IMG=quay.io/$USERNAME/koku-metrics-operator:v$VERSION
```

4. Generate the new bundle:

Run the following command to generate the bundle:

```sh
$ docker pull quay.io/$USERNAME/koku-metrics-operator:v$VERSION
$ make bundle CHANNELS=alpha,beta DEFAULT_CHANNEL=beta IMG=quay.io/$USERNAME/koku-metrics-operator:v$VERSION
```

This will generate a new `$VERSION` bundle inside of the `koku-metrics-operator` directory within the repository.

1. Build the bundle image and push to Quay:

Build and push bundle to your Quay repo:

```sh
$ make bundle-build BUNDLE_IMG=quay.io/$USERNAME/koku-metrics-operator-bundle:v$VERSION bundle-push
```

1. Once the bundle is available in Quay, deploy it to your cluster:
```sh
$ make bundle-deploy-upgrade BUNDLE_IMG=quay.io/$USERNAME/koku-metrics-operator-bundle:v$VERSION
```

8. Observe the installed operator, and ensure it upgrades automatically.

Once the operator has upgraded, run through manual tests as normal.
