# Local Development

## Pre-reqs

* Access to a supported version of an Openshift cluster
  * [Red Hat OpenShift Local](https://crc.dev/docs/installing/) can be used if the monitoring stack is enabled.
  * A cluster can also be provisioned [here](https://demo.redhat.com/catalog)
* A clone of [koku-metrics-operator](https://github.com/project-koku/koku-metrics-operator)
* [Go 1.13 or greater](https://golang.org/doc/install)
* [Openshift-CLI](https://docs.redhat.com/en/documentation/openshift_container_platform/latest/html/cli_tools/openshift-cli-oc#cli-about-cli_cli-developer-commands) (preferably a version that matches your Openshift cluster version)
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


5. Grant monitoring read access to the ServiceAccount:

    The local operator uses the `koku-metrics-controller-manager` ServiceAccount token to query the Prometheus/Thanos endpoint.
    Grant the ServiceAccount `cluster-monitoring-view` permissions:

    ```bash
    oc adm policy add-cluster-role-to-user \
      cluster-monitoring-view \
      system:serviceaccount:koku-metrics-operator:koku-metrics-controller-manager
    ```

    If this permission is missing, reconciliation can fail with:
    `prometheus test query failed: client_error: client error: 403`

6.  Retrieve ServiceAccount Token and CA Certificate:

    The operator's local environment needs a serviceAccount token and the cluster's service CA certificate to authenticate with the Kubernetes API. The `get-token-and-cert` Make command handles this retrieval:

    ```bash
    make get-token-and-cert
    ```

    This generates a new token for the `koku-metrics-controller-manager` ServiceAccount and retrieves the cluster's Service CA certificate from the `kube-root-ca.crt` ConfigMap.

    These files (`token` and `service-ca.crt`) will be placed in the `testing` directory but you can specify a different output location by setting the `SECRET_ABSPATH` environment variable:

    ```bash
    SECRET_ABSPATH=/absolute/path/to/local/secrets make get-token-and-cert
    ```

7. Deploy the operator

    ```
    make run ENABLE_WEBHOOKS=false SECRET_ABSPATH=/absolute/path/to/local/secrets
    ```

    At this point, you will see the operator spin up in your terminal. After a few seconds, you should see something similar to the following output:
    ```
    2020-10-21T09:31:37.195-0400    INFO    controller-runtime.controller   Starting workers        {"controller": "kokumetricsconfig", "worker count": 1}
    ```
    The operator is running but is not doing any work. We need to create a CR.

8. Deploy a CR. For local development, uses default token auth. However service-account authentication can be used with the following, which creates the appropriate authentication spec within the CR. This is the `client_id` and `client_secret` for your Red Hat Hybrid Cloud Console:

    ```
    $ make deploy-local-cr AUTH=service-account CLIENT_ID=<client_id> CLIENT_SECRET=<client_secret>
    ```
    This command copies `testing/costmanagementmetricsconfig-template.yaml` to `testing/costmanagement-metrics-cfg_v1beta1_costmanagementmetricsconfig.yaml`, adds an external prometheus route, disables TLS verification for the prometheus route, and (when `AUTH=service-account`) adds the authentication spec and applies `testing/authentication_secret.yaml`. The command then applies `testing/costmanagement-metrics-cfg_v1beta1_costmanagementmetricsconfig.yaml` to the cluster.

    After this CR has been created in the cluster, reconciliation will begin.

    Running `make deploy-local-cr` as-is will create the external prometheus route, disable TLS verification for prometheus, and use token authentication for console.redhat.com.

9. To continue development, make code changes. To apply those changes, stop the operator, and redeploy it. If changes are made to the api, the CRD needs to be re-registered, and the operator re-deployed.
