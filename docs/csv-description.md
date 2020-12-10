# v0.9.0-alpha Koku Metrics Operator (Unsupported)
## Introduction
The `koku-metrics-operator` is an OpenShift Operator used to obtain OpenShift usage data and upload it to [cost managment](https://access.redhat.com/documentation/en-us/openshift_container_platform/4.5/html/getting_started_with_cost_management/assembly_introduction_cost_management). The Operator queries Prometheus to create metric reports, which are then packaged and uploaded to cost management at [cloud.redhat.com](https://cloud.redhat.com). For more information, reach out to <cost-mgmt@redhat.com>.
## Features and Capabilities
The Koku Metrics Operator (`koku-metrics-operator`) collects the metrics required for cost management by:
* Querying Prometheus to create reports specific to cost management.
* Packaging these reports as a tarball which is uploaded to cost management through cloud.redhat.com.
* The operator is also capable of creating a source in cloud.redhat.com. A source is required for cost management to process the upload.
#### Limitations (Potential for metrics data loss)
* Report storage is not backed by a PersitentVolume. If the operator is redeployed, a gap may be introduced in the usage metrics.
* A source **must** exist in cloud.redhat.com for an uploaded payload to be processed by cost management. The operator sends the payload to the Red Hat Insights Ingress service which usually returns successfully, but the operator does not currently confirm with cost management that the payload was processed. After Ingress accepts the uploaded payload, the payload is removed from the operator and is gone forever. If the data within the payload is not processed, a gap will be introduced in the usage metrics.
## Installation
The operator must be installed in the `koku-metrics-operator` namespace. The namespace can be created through either the UI or CLI:
#### Namespace creation:
##### UI
1. On the left navigation pane, select `Administration` -> `Namespaces` -> `Create Namespace`.
2. Name the namespace `koku-metrics-operator`.
3. Select `Create`.
##### CLI
1. Run the following via the CLI to create and use the `koku-metrics-operator` project:
```
oc new-project koku-metrics-operator
```
#### Operator installation:
Ensure that the operator is installed into the `koku-metrics-operator` namespace.

## Configure the koku-metrics-operator
The operator can be configured through either the UI or CLI:
#### Configure through the UI
##### Configure authentication
The default authentication for the operator is `token`. No further steps are required to configure token authentication. If `basic` is the preferred authentication method, a Secret must be created which holds username and password credentials:
1. On the left navigation pane, select `Workloads` -> `Secrets` -> select Project: `koku-metrics-operator` -> `Create` -> `Key/Value Secret`
2. Give the Secret a name and add 2 keys: `username` and `password` (all lowercase). The values for these keys correspond to cloud.redhat.com credentials.
3. Select `Create`.
##### Create the KokuMetricsConfig
Configure the koku-metrics-operator by creating a `KokuMetricsConfig`.
1. On the left navigation pane, select `Operators` -> `Installed Operators` -> `koku-metrics-operator` -> `Create KokuMetricsConfig`.
2. For `basic` authentication, edit the following values in the spec:
    * Replace `authentication: type:` with `basic`.
    * Add the`secret_name` field under `authentication`, and set it equal to the name of the authentication Secret that was created above. The spec should look similar to the following:

        ```
          authentication:
            secret_name: SECRET-NAME
            type: basic
        ```

3. To configure the koku-metrics-operator to create a cost management source, edit the following values in the `source` field:
    * Replace `INSERT-SOURCE-NAME` with the preferred name of the source to be created.
    * Replace the `create_source` field value with `true`.

    **Note:** if the source already exists, replace `INSERT-SOURCE-NAME` with the existing name, and leave `create_source` as false. This will allow the operator to confirm the source exists.
4. Select `Create`.

#### Configure through the CLI
##### Configure authentication
The default configuration method for the operator to create sources and upload to [cloud.redhat.com](https://cloud.redhat.com/) is `token`. No further steps are required for configuring `token` authentication. If `basic` is the preferred authentication method, a Secret must be created which holds username and password credentials:
1. Copy the following into a file called `auth-secret.yaml`:

    ```
    kind: Secret
    apiVersion: v1
    metadata:
      name: authentication-secret
      namespace: koku-metrics-operator
    data:
    username: >-
      Y2xvdWQucmVkaGF0LmNvbSB1c2VybmFtZQ==
    password: >-
      Y2xvdWQucmVkaGF0LmNvbSBwYXNzd29yZA==
    ```

2. Replace the metadata.name with the preferred name for the authentication secret.
3. Replace the `username` and `password` values with the base64-encoded username and password credentials for logging into cloud.redhat.com.
4. Deploy the secret to the `koku-metrics-operator` namespace:
    ```
    $ oc create -f auth-secret.yaml
    ```

    **Note:** The name of the secret should match the `spec:  authentication:  secret_name` set in the KokuMetricsConfig that is going to be configured in the next steps.

##### Create the KokuMetricsConfig
Configure the koku-metrics-operator by creating a `KokuMetricsConfig`.
1. Copy the following `KokuMetricsConfig` resource template and save it to a file called `kokumetricsconfig.yaml`:

    ```
    apiVersion: koku-metrics-cfg.openshift.io/v1alpha1
    kind: KokuMetricsConfig
    metadata:
      name: kokumetricscfg-sample
    spec:
      authentication:
        type: token
      packaging:
        max_size_MB: 100
      prometheus_config: {}
      source:
        check_cycle: 1440,
        create_source: false,
        name: INSERT-SOURCE-NAME
      upload:
        upload_cycle: 360,
        upload_toggle: true
    ```

2. To configure the operator to use `basic` authentication, edit the following values in the `kokumetricsconfig.yaml` file:
    * Replace `authentication: type:` with `basic`.
    * Add a field called `secret_name` to the authentication field in the spec and set it equal to the name of the authentication secret that was created earlier. The authentication spec should look similar to the following:

        ```
          authentication:
            secret_name: SECRET-NAME
            type: basic
        ```
        
3. To configure the koku-metrics-operator to create a cost management source, edit the following values in the `kokumetricsconfig.yaml` file:
    * Replace `INSERT-SOURCE-NAME` with the preferred name of the source to be created.
    * Replace `create_source` field value with `true`.

    **Note:** if the source already exists, replace `INSERT-SOURCE-NAME` with the existing name, and leave `create_source` as false. This will allow the operator to confirm the source exists.
4. Deploy the `KokuMetricsConfig` resource:
    ```
    $ oc create -f kokumetricsconfig.yaml
    ```

The koku-metrics-operator will now create, package, and upload OpenShift usage reports to cost management.