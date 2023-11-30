# Local Development

## Pre-reqs

* Access to a 4.3+ Openshift cluster
* A clone of [koku-metrics-operator](https://github.com/project-koku/koku-metrics-operator)
* [Go 1.13 or greater](https://golang.org/doc/install)
* [Openshift-CLI](https://docs.openshift.com/container-platform/4.5/cli_reference/openshift_cli/getting-started-cli.html) (preferably a version that matches your Openshift cluster version)
* [kubebuilder](https://book.kubebuilder.io/quick-start.html#installation)
* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
* [kustomize](https://kubectl.docs.kubernetes.io/installation/kustomize/) (before installing this separately, check that it was not already installed along with kubectl)
* [Docker Desktop](https://www.docker.com/products/docker-desktop)
* [quay.io](quay.io) account

## Running the operator locally

1. Log into your OCP cluster from a terminal, create an `koku-metrics-operator` namespace, and switch to the new namespace:

    ```
    $ oc login --token=<token> --server=<server>
    $ oc new-project koku-metrics-operator
    ```

2. Build the manager binary:

    ```
    $ make build
    ```

3. Register the CRD with the Kubernetes apiserver:

    ```
    $ make install
    ```

4. Deploy the ServiceAccount:

    ```
    $ oc apply -f testing/sa.yaml
    ```

5. The `token` and `service-ca.crt` need to be copied from one of the created `koku-metrics-manager-role-token-*` secrets.
A make command exists to help:

    ```
    $ make get-token-and-cert
    ```

This will place the `token` and `service-ca.crt` in the `testing` directory. To place these files somewhere else, you can also use the command like this:

    ```
    $ SECRET_ABSPATH=/absolute/path/to/local/secrets make get-token-and-cert
    ```

6. Deploy the operator

    ```
    make run ENABLE_WEBHOOKS=false SECRET_ABSPATH=/absolute/path/to/local/secrets
    ```

    At this point, you will see the operator spin up in your terminal. After a few seconds, you should see something similar to the following output:
    ```
    2020-10-21T09:31:37.195-0400    INFO    controller-runtime.controller   Starting workers        {"controller": "kokumetricsconfig", "worker count": 1}
    ```
    The operator is running but is not doing any work. We need to create a CR.

7. Deploy a CR. For local development, use basic authentication. The following creates the appropriate authentication spec within the CR. `username` and `password` correspond to the username (not email address) and password for the account you want to use at console.redhat.com:

    ```
    $ make deploy-local-cr AUTH=basic USER=<username> PASS=<password>
    ```
    This command uses the CR defined in `config/samples/koku-metrics-cfg_v1beta1_kokumetricsconfig.yaml`, adds an external prometheus route, disables TLS verification for the prometheus route, adds the authentication spec, and creates a CR in `testing/koku-metrics-cfg_v1beta1_kokumetricsconfig.yaml`. The command then deploys this CR to the cluster.

    After this CR has been created in the cluster, reconciliation will begin.

    Running `make deploy-local-cr` as-is will create the external prometheus route, disable TLS verification for prometheus, and use token authentication for console.redhat.com.

8. To continue development, make code changes. To apply those changes, stop the operator, and redeploy it. If changes are made to the api, the CRD needs to be re-registered, and the operator re-deployed.
