## Introduction
The `koku-metrics-operator` is an OpenShift Operator used to obtain OpenShift usage data and upload it to [cost managment](https://access.redhat.com/documentation/en-us/openshift_container_platform/4.3/html-single/getting_started_with_cost_management/index). The Operator queries Prometheus to create metrics reports, which are then packaged and uploaded to cost management. For more information, reach out to cost-mgmt@redhat.com.
## Features and Capabilities
The Cost Management Operator (`cost-mgmt-operator`) collects the metrics required for cost management by:
* Querying Prometheus to create usage reports specific to cost management.
* Packaging these reports as a tarball which is uploaded to cost management through cloud.redhat.com.
## Configuring the koku-metrics-operator
### Create the koku-metrics-operator namespace 
1. Run the following via the CLI to create and use the `koku-metrics-operator` project: 
```
oc new-project koku-metrics-operator
```
### Configuring authentication
Decide if you are going to use the default authentication method (token)to create sources and upload OpenShift data to [cloud.redhat.com](https://cloud.redhat.com/). If you are going to use token authentication, no further steps are required for configuring authentication. If you choose to use basic authentication, you need to complete the following steps. 
1. Copy the following into a file called `auth_secret.yaml`:
    ```
    ---
    kind: Secret
    apiVersion: v1
    metadata:
    name: dev-auth-secret
    namespace: koku-metrics-operator
    annotations:
        kubernetes.io/service-account.name: koku-metrics-operator
    data:
    username: >-
        Y2xvdWQucmVkaGF0LmNvbSB1c2VybmFtZQ==
    password: >-
        Y2xvdWQucmVkaGF0LmNvbSBwYXNzd29yZA==
    ```
2. Choose a name for your authentication secret and replace the metadata.name with it.
3. To use basic authentication, edit the secret to replace the username and password values with your base64-encoded username and password for connecting to cloud.redhat.com.
4. Deploy the secret to your OpenShift cluster in the `koku-metrics-operator` namespace:
    ```
    $ oc create -f auth-secret.yaml
    ```
     **NOTE**
    The name of the secret should match the `authentication_secret_name` set in the CostManagement custom resource configured in the next steps.
    ---
### Configuring the koku-metrics-operator
Configure the koku-metrics-operator by creating a `KokuMetricsConfig`. 
1. Copy the following `KokuMetricsConfig` resource template and save it to a file called `KokuMetricsConfig.yaml`:
    ```
    ---
    "apiVersion": "koku-metrics-cfg.openshift.io/v1alpha1",
    "kind": "KokuMetricsConfig",
    "metadata": {
        "name": "kokumetricscfg-sample"
    },
    "spec": {
        "authentication": {
            "type": "token"
        },
        "packaging": {
            "max_size_MB": 100
        },
        "prometheus_config": {},
        "source": {
            "check_cycle": 1440,
            "create_source": false
        },
        "upload": {
            "upload_cycle": 360,
            "upload_toggle": true
        }
    }
    ```
2. Edit the following values in your `KokuMetricsConfig.yaml` file:
    * If you are using `basic` authentication, you must change the authentication type within the spec to `basic` as `token` authentication is the default. For basic authentication, you must also add a field called `secret_name` to the authentication field in the spec and set it equal to the name of your authentication secret you created earlier.
    * If you would like the koku-metrics-operator to create your cost managment source for you, you will have to edit the source field in the spec. You will need to add a field to the source in the spec called `source_name` and set it equal to the name of the source that you would like to create. Then, you should change the `create_source` field value to `true` instead of `false`. 
3. Deploy the `KokuMetricsConfig` resource:
    ```
    $ oc create -f KokuMetricsConfig.yaml
    ```
The koku-metrics-operator will now create, package, and upload your OpenShift usage reports to cost management. 