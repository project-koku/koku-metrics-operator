# Local Development

## Pre-reqs

* Access to an Openshift cluster
* A clone of [korekuta-go-operator](https://github.com/project-koku/korekuta-operator-go)
* [Go 1.13 or greater](https://golang.org/doc/install)
* [Openshift-CLI](https://docs.openshift.com/container-platform/4.5/cli_reference/openshift_cli/getting-started-cli.html) (preferably a version that matches your Openshift cluster version)
* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
* [kustomize](https://kubernetes-sigs.github.io/kustomize/installation/) (before installing this separately, check that it was not already installed along with kubectl)
* [Docker Desktop](https://www.docker.com/products/docker-desktop)
* quay.io account

## Running the operator locally

1. Log into your OCP cluster from a terminal, create an `openshift-cost` namespace, and switch to the new namespace:

    ```
    $ oc login --token=<token> --server=<server>
    $ oc create namespace openshift-cost
    $ oc project openshift-cost
    ```

2. Build the manager binary:

    ```
    $ make manager
    ```

3. Register the CRD with the Kubernetes apiserver:

    ```
    $ make install
    ```

4. Deploy the operator

    ```
    make run ENABLE_WEBHOOKS=false
    ```

    At this point, you will see the operator spin up in your terminal. After a few seconds, you should see something similar to the following output:
    ```
    2020-10-21T09:31:37.195-0400    INFO    controller-runtime.controller   Starting workers        {"controller": "costmanagement", "worker count": 1}
    ```
    The operator is running but is not doing any work. We need to create a CR.

5. Deploy a CR. For now, use basic authentication. The following creates the appropriate spec. `username` and `password` correspond to the username (not email address) and password for the account you want to use at cloud.redhat.com:

    ```
    $ make deploy-cr AUTH=basic USER=<username> PASS=<password>
    ```
    This command uses the CR defined in `config/samples/cost-mgmt_v1alpha1_costmanagement.yaml`, adds the authentication spec, and creates a CR in `testing/cost-mgmt_v1alpha1_costmanagement.yaml`. The command then deploys this CR to the cluster.

    After this CR has been created in the cluster, reconciliation will begin. It will error because the URL for prometheus needs to be set in the CR.

6. In the cluster dashboard, under `Networks`, select `Routes`. Change the project to `openshift-monitoring` and copy the route for `thanos-querier`. In `testing/cost-mgmt_v1alpha1_costmanagement.yaml`, add the route and turn off TLS verification:

    ```
    spec:
        authentication:
            type: basic
            secret_name: dev-auth-secret
        prometheus_config:
            service_address: https://thanos-querier-openshift-monitoring.apps.cluster-....
            skip_tls_verification: true
    ```
    Redeploy the CR:
    ```
    $ testing/cost-mgmt_v1alpha1_costmanagement.yaml
    ```
    Once the CR is updated in the cluster, reconciliation will happen automatically. Now prometheus should be reachable.

7. To continue development, make code changes. To apply those changes, stop the operator, and redeploy it. If changes are made to the api, the CRD needs to be re-registered, and the operator re-deployed.
