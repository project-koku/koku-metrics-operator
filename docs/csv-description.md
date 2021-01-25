# v0.9.1-alpha Koku Metrics Operator (Unsupported)
## Introduction
The `koku-metrics-operator` is an OpenShift Operator used to obtain OpenShift usage data and upload it to [cost managment](https://access.redhat.com/documentation/en-us/openshift_container_platform/4.5/html/getting_started_with_cost_management/assembly_introduction_cost_management). The Operator queries Prometheus to create metric reports, which are then packaged and uploaded to cost management at [cloud.redhat.com](https://cloud.redhat.com). For more information, reach out to <cost-mgmt@redhat.com>.
## Features and Capabilities
The Koku Metrics Operator (`koku-metrics-operator`) collects the metrics required for cost management by:
* Querying Prometheus to create reports specific to cost management.
* Packaging these reports as a tarball which is uploaded to cost management through cloud.redhat.com.
* The operator is also capable of creating a source in cloud.redhat.com. A source is required for cost management to process the upload.
* When the operator is configured, it will create a PVC to store the reports on. The operator can be configured to create a user-specified PVC or use a PVC that already exists. If not specified, the default PVC name is `koku-metrics-operator-data` and it is configured at 10GB of storage.  
* The operator can be installed and configured to support an air-gapped network. The reports are stored on a PVC and must be manually downloaded and uploaded to cloud.redhat.com to support restricted networks. 
#### Limitations (Potential for metrics data loss)
* A source **must** exist in cloud.redhat.com for an uploaded payload to be processed by cost management. The operator sends the payload to the Red Hat Insights Ingress service which usually returns successfully, but the operator does not currently confirm with cost management that the payload was processed. After Ingress accepts the uploaded payload, the payload is removed from the operator and is gone forever. If the data within the payload is not processed, a gap will be introduced in the usage metrics.
## Configure the koku-metrics-operator
##### Configure authentication
The default authentication for the operator is `token`. No further steps are required to configure token authentication. If `basic` is the preferred authentication method, a Secret must be created which holds username and password credentials:
1. On the left navigation pane, select `Workloads` -> `Secrets` -> select Project: `koku-metrics-operator` -> `Create` -> `Key/Value Secret`
2. Give the Secret a name and add 2 keys: `username` and `password` (all lowercase). The values for these keys correspond to cloud.redhat.com credentials.
3. Select `Create`.
##### Create the KokuMetricsConfig
Configure the koku-metrics-operator by creating a `KokuMetricsConfig`.
1. On the left navigation pane, select `Operators` -> `Installed Operators` -> `koku-metrics-operator` -> `KokuMetricsConfig` -> `Create Instance`.
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
4. If not specified, the operator will create a default Persistent Volume Claim called `koku-metrics-operator-data` with 10Gi of storage. To configure the koku-metrics-operator to use or create a different PVC, edit the following in the spec: 
    * Add the desired configuration to the `volume_claim_template` field in the spec:

        ```
          volume_claim_template:
            apiVersion: v1
            kind: PersistentVolumeClaim
            metadata:
              name: pvc-spec-definition
            spec:
              storageClassName: gp2
              accessModes:
                - ReadWriteOnce
              resources:
                requests:
                  storage: 10Gi
        ```

    **Note:** If using the YAML View, the `volume_claim_template` field must be added to the spec
5. Select `Create`.
