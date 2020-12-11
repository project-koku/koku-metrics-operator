## Generating and testing release bundles 

More specific documentation for generating and testing bundles can be found [here](https://operator-framework.github.io/community-operators/testing-operators/).

### Pre-requisites 
Testing the bundle requires `opm`. To install opm, clone the operator-registry repository: 

```
git clone https://github.com/operator-framework/operator-registry
```
Next, cd into the directory and run `make build`. This will generate an `opm` executable in the `operator-registry/bin/` directory. Add that to your path, or substitue `opm` in the following commands with the full path to your `opm` executable. 

### Generating the release bundle 
First, create a GitHub release that corresponds with the operator release version. The [previous releases](https://github.com/project-koku/koku-metrics-operator/releases/tag/v0.9.0) can be used as a template. 

Next, build and push the updated operator image to the project-koku quay.io repository: 

```sh
$ export REL=<release-version>
docker build . -t quay.io/project-koku/koku-metrics-operator:$REL
docker push quay.io/project-koku/koku-metrics-operator:$REL   
```

Update the release version at the top of the `Makefile` to match the release version of the operator. 
Run the following command to generate the bundle: 

```
make bundle DEFAULT_CHANNEL=alpha
```
This should generate a new `1.0.0` bundle inside of the `koku-metrics-operator` directory within the repository. 

### Testing the release bundle 
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

Build the image inside of the version directory:
```sh
$ export USERNAME=<quay-username>
$ export REL=<release-version>
cd community-operators/koku-metrics-operator/$REL
$ docker build -f Dockerfile . -t quay.io/$USERNAME/koku-metrics-operator:$REL
```

Push the image to a repository and make sure to set the repository to public:

```sh
$ docker push quay.io/$USERNAME/koku-metrics-operator:$REL
```

Next, use `opm` to generate a catalog image with the koku-metrics-operator:

```sh
opm index add --bundles quay.io/$USERNAME/koku-metrics-operator:$REL --generate --out-dockerfile "my.Dockerfile"
```

Now, build and push the catalog image: 

```sh
docker build -f my.Dockerfile . --tag quay.io/$USERNAME/test-catalog:latest
docker push quay.io/$USERNAME/test-catalog:latest
```

Finally, create a catalog source by copying the following into a file called `catalog-source.yaml`: 

```sh
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: my-test-catalog
  namespace: openshift-marketplace
spec:
  sourceType: grpc
  image: quay.io/$USERNAME/test-catalog:latest
```

Deploy the catalog source to the cluster: 

```
oc apply -f catalog-source.yaml
```

Verify that the catalog source was created without errors by looking at the resulting pod in the `openshift-marketplace` project. 

Now search OperatorHub for the koku-metrics-operator. It should be available under the `custom` or `my-test-catalog` provider type depending on your OCP version.

Install the koku-metrics-operator in the `koku-metrics-operator` namespace, and test as normal. 

After testing, remove the `catalog-source.yaml` and any files that were generated from `opm` and submit the PR to the community-operators repo.