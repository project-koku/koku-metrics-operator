## Generating and testing release bundles

### Pre-requisites

* Access to an OpenShift cluster (any currently supported version)

### Testing an operator upgrade - Overview

Reference: https://sdk.operatorframework.io/docs/olm-integration/tutorial-bundle
OpenShift comes with OLM enabled, so the "Enable OLM" steps can be omitted.

Testing an upgrade is composed of the following general steps:
1. Create a namespace (koku-metrics-operator).
1. Install the previous bundle (make bundle-deploy-previous).
1. Generate the new controller image, and push to Quay.
1. Generate the new bundle.
1. Build the bundle image and push to Quay.
1. Deploy the upgraded bundle (make bundle-deploy-upgrade).
1. Observe the installed operator, and ensure it upgrades automatically.
1. Cleanup.


### Testing an operator upgrade
1. set version numbers in the Makefile:
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

1. Generate the new controller image, and push to your Quay repo:

    ```sh
    $ USERNAME=<quay-username>
    $ VERSION=<release-version>
    $ make docker-buildx IMG=quay.io/$USERNAME/koku-metrics-operator:v$VERSION
    ```

1. Generate the new bundle:

    Run the following command to generate the bundle:

    ```sh
    $ docker pull quay.io/$USERNAME/koku-metrics-operator:v$VERSION
    $ make bundle CHANNELS=alpha,beta DEFAULT_CHANNEL=beta IMG=quay.io/$USERNAME/koku-metrics-operator:v$VERSION
    ```

    This will generate a new `$VERSION` bundle inside of the `koku-metrics-operator` directory within the repository.

1. Build the bundle image and push to Quay:

    ```sh
    $ make bundle-build BUNDLE_IMG=quay.io/$USERNAME/koku-metrics-operator-bundle:v$VERSION bundle-push
    ```

1. Once the bundle is available in Quay, deploy it to your cluster:
    ```sh
    $ make bundle-deploy-upgrade BUNDLE_IMG=quay.io/$USERNAME/koku-metrics-operator-bundle:v$VERSION
    ```

1. Observe the installed operator, and ensure it upgrades automatically.

    Once the operator has upgraded, run through manual tests as normal.

1. When done with testing, the bundle can be deleted from the cluster with:

    ```sh
    $ make deploy-bundle-cleanup
    ```
