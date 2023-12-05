# koku-metrics-operator

[![License: Apache-2.0](https://img.shields.io/github/license/project-koku/koku.svg)](https://www.apache.org/licenses/LICENSE-2.0)
[![Build and Publish Images](https://github.com/project-koku/koku-metrics-operator/actions/workflows/build-and-publish.yaml/badge.svg?branch=main)](https://github.com/project-koku/koku-metrics-operator/actions/workflows/build-and-publish.yaml)
[![Unit Tests](https://github.com/project-koku/koku-metrics-operator/workflows/Unit%20Tests/badge.svg)](https://github.com/project-koku/koku-metrics-operator/actions?query=branch%3Amain+workflow%3A%22Unit+Tests%22)

## About

Operator to obtain OCP usage data and upload it to koku. The operator utilizes [Golang](http://golang.org/) to collect usage data from an OCP cluster installation.

You must have access to an OpenShift v.4.5+ cluster.

To submit an issue please visit https://issues.redhat.com/projects/COST/


## Development

More specific documentation for local development can be found [here](docs/local-development.md).

This project was generated using Operator SDK. For a more in depth understanding of the structure of this repo, see the [user guide](https://sdk.operatorframework.io/docs/building-operators/golang/quickstart/) that was used to generate it.

This project requires Go 1.13 or greater if you plan on running the operator locally. To get started developing against `koku-metrics-operator` first clone a local copy of the git repository.

```
git clone https://github.com/project-koku/koku-metrics-operator.git
```

Run the following command to prevent changes to `config/manager/kustomization.yaml` from being committed to the repo:

```
git update-index --assume-unchanged config/manager/kustomization.yaml
```

Now install and configure pre-commit. You can choose your installation steps for pre-commit by using the [pre-commit installation instructions](https://pre-commit.com/#installation). Configure pre-commit for the local repository with the following command:

```
pre-commit install
```

Next, install the Operator SDK CLI using the following [documentation](https://sdk.operatorframework.io/docs/installation/). The operator is currently being built with the v0.19 release of the operator-sdk.

To build the manager binary you can execute the following make command:

```
make manager
```

To build the docker image you can execute the following make command:

```
make docker-build
```

Linting can be performed with the following make commands:

```
make fmt
make lint
```

## Testing

Execute test by running the following make command:

```
make test
```

## Deploying the Operator

First, create the `koku-metrics-operator` project. This is where we are going to deploy our Operator.

Before running the operator, the CRD must be registered with the Kubernetes apiserver:

```
make install
```

Once this is done, there are two ways to run the operator:

- As Go program outside a cluster
- As a Deployment inside a Kubernetes cluster

## Configuring your test environment

Projects are scaffolded with unit tests that utilize the [envtest](https://godoc.org/sigs.k8s.io/controller-runtime/pkg/envtest)
library, which requires certain Kubernetes server binaries be present locally.
Installation instructions can be found [here][env-test-setup].

### 1. Run locally outside the cluster

To run the operator locally execute the following command:

```sh
$ make run ENABLE_WEBHOOKS=false
```

### 2. Run as a Deployment inside the cluster

#### Build and push the image

Before building the operator image, ensure the generated Dockerfile references
the base image you want. You can change the default "runner" image `gcr.io/distroless/static:nonroot`
by replacing its tag with another, for example `alpine:latest`, and removing
the `USER: nonroot:nonroot` directive.

To build and push the operator image, use the following `make` commands.
Make sure to modify the `IMG` arg in the example below to reference a container repository that
you have access to. You can obtain an account for storing containers at
repository sites such quay.io or hub.docker.com. This example uses quay.

Build the image:
```sh
$ export USERNAME=<quay-username>

$ make docker-build IMG=quay.io/$USERNAME/koku-metrics-operator:v0.0.1
```

Push the image to a repository and make sure to set the repository to public:

```sh
$ make docker-push IMG=quay.io/$USERNAME/koku-metrics-operator:v0.0.1
```
**Note**:
The name and tag of the image (`IMG=<some-registry>/<project-name>:tag`) in both the commands can also be set in the Makefile. Modify the line which has `IMG ?= controller:latest` to set your desired default image name.

Branches of the repository are built automaticaly [here](https://quay.io/repository/project-koku/koku-metrics-operator).


#### Deploy the operator


```sh
$ cd config/default/ && kustomize edit set namespace "koku-metrics-operator" && cd ../..
```

Run the following to deploy the operator. This will also install the RBAC manifests from `config/rbac`.

```sh
$ make deploy IMG=quay.io/$USERNAME/koku-metrics-operator:v0.0.1
```

You can optionally build, push, and deploy the image all at the same time by running:

```sh
$ make build-and-deploy IMG=quay.io/$USERNAME/koku-metrics-operator:v0.0.1
```

You can also optionally substitute the USER arg for the IMG arg above for the build & deploy commands. Run `make help` to see all of the options.

*NOTE* If you have enabled webhooks in your deployments, you will need to have cert-manager already installed
in the cluster or `make deploy` will fail when creating the cert-manager resources.

If your current branch has been pushed to the origin repository then you can deploy your current branch with the following command.
```sh
$ make deploy-branch
```

Verify that the koku-metrics-operator is up and running:

```console
$ oc get deployment
```

## Create a KokuMetricsConfig CR

Create the CR with the default token authentication:

```sh
$ make deploy-cr
```

Create the CR with basic authentication specifying a username and password:

```sh
$ make deploy-cr USER=$YOUR_USERNAME PASS=$YOUR_PASSWORD AUTH=basic
```

Review the logs for the Koku Metrics operator.

### Cleanup

```sh
$ oc delete -f config/samples/koku-metrics-cfg_v1beta1_kokumetricsconfig.yaml
$ oc delete deployments,service -l control-plane=controller-manager
$ oc delete role,rolebinding --all
```
