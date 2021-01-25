# v0.9.1-alpha Koku Metrics Operator (Unsupported)
## Introduction
The `koku-metrics-operator` is an OpenShift Operator used to collect OpenShift usage data and upload it to [cost managment](https://access.redhat.com/documentation/en-us/openshift_container_platform/4.5/html/getting_started_with_cost_management/assembly_introduction_cost_management). The operator queries Prometheus to create metric reports which are packaged and uploaded to cost management at [cloud.redhat.com](https://cloud.redhat.com).

This operator is capable of functioning within a restricted network. In this mode, the operator will store the packaged reports for manual retrieval instead of being uploaded to cost management. Documentation for installing an operator within a restricted network can be found [here](https://docs.openshift.com/container-platform/4.5/operators/admin/olm-restricted-networks.html).

For more information, reach out to <cost-mgmt@redhat.com>.
## Features and Capabilities
#### Metrics collection:
The Koku Metrics Operator (`koku-metrics-operator`) collects the metrics required for cost management by:
* Querying Prometheus to gather the necessary metrics for cost management.
* Writing Prometheus queries to CSV report files.
* Packaging the CSV report files into tarballs.

#### Additional Capabilities:
* The operator can be configured to automatically upload the packaged reports to cost management through Red Hat Insights Ingress service.
* The operator can create a source in cloud.redhat.com. A source is required for cost management to process the uploaded packages.
* PersistentVolumeClaim (PVC) configuration: The KokuMetricsConfig CR can accept a PVC definition and the operator will create and mount the PVC. If one is not provided, a default PVC will be created.
* Restricted network installation: this operator can function on a restricted network. In this mode, the operator stores the packaged reports for manual retrieval.

#### Limitations (Potential for metrics data loss)
* A source **must** exist in cloud.redhat.com for an uploaded payload to be processed by cost management. The operator sends the payload to the Red Hat Insights Ingress service which usually returns successfully, but the operator does not currently confirm with cost management that the payload was processed. After Ingress accepts the uploaded payload, the payload is removed from the operator and is gone forever. If the data within the payload is not processed, a gap will be introduced in the usage metrics.
* The `koku-metrics-operator` will not be able to generate new reports if the PVC storage is filled. If this occurs, the reports must be manually deleted from the PVC so that the operator can function as normal. 
* The default report retention is 30 reports (about one week's worth of data). If the operator is configured to run in restricted-network mode, the reports must be manually downloaded and uploaded to cloud.redhat.com every week, or they will be deleted and the data will be lost. 
## Configure the koku-metrics-operator
##### Configure authentication
The default authentication for the operator is `token`. No further steps are required to configure token authentication. If `basic` is the preferred authentication method, a Secret must be created which holds username and password credentials:
1. On the left navigation pane, select `Workloads` -> `Secrets` -> select Project: `koku-metrics-operator` -> `Create` -> `Key/Value Secret`
2. Give the Secret a name and add 2 keys: `username` and `password` (all lowercase). The values for these keys correspond to cloud.redhat.com credentials.
3. Select `Create`.
##### Storage configuration
The operator will attempt to create and use the following PVC when installed:
  ```
    volume_claim_template:
      apiVersion: v1
      kind: PersistentVolumeClaim
      metadata:
        name: koku-metrics-operator-data
      spec:
        accessModes:
          - ReadWriteOnce
        resources:
          requests:
            storage: 10Gi
  ```
If a different PVC should be utilized, a valid PVC should be specified in the KokuMetricsConfig CR as described in the following section. The PVC to be used may exist already, or the operator will attempt to create it.

To use the default specification, the follow assumptions must be met:
1. A default StorageClass is defined.
2. Dynamic provisioning for that default StorageClass is enabled.
If these assumptions are not met, the operator will not deploy correctly. In these cases, storage must be manually configured. For manual configuration, a valid PVC should be supplied in the `volume_claim_template` spec of the KokuMetricsConfig CR.

##### Create the KokuMetricsConfig
Configure the koku-metrics-operator by creating a `KokuMetricsConfig`.
1. On the left navigation pane, select `Operators` -> `Installed Operators` -> `koku-metrics-operator` -> `KokuMetricsConfig` -> `Create Instance`.
2. For `basic` authentication, edit the following values in the spec:
    * Replace `authentication: type:` with `basic`.
    * Add the `secret_name` field under `authentication`, and set it equal to the name of the authentication Secret that was created above. The spec should look similar to the following:
        ```
          authentication:
            secret_name: SECRET-NAME
            type: basic
        ```
3. To configure the koku-metrics-operator to create a cost management source, edit the following values in the `source` field:
    * Replace `INSERT-SOURCE-NAME` with the preferred name of the source to be created.
    * Replace the `create_source` field value with `true`.
    **Note:** if the source already exists, replace `INSERT-SOURCE-NAME` with the existing name, and leave `create_source` as false. This will allow the operator to confirm the source exists.
4. If not specified, the operator will create a default PersistentVolumeClaim called `koku-metrics-operator-data` with 10Gi of storage. To configure the koku-metrics-operator to use or create a different PVC, edit the following in the spec:
    * Add the desired configuration to the `volume_claim_template` field in the spec:
        ```
          volume_claim_template:
            apiVersion: v1
            kind: PersistentVolumeClaim
            metadata:
              name: pvc-spec-definition
            spec:
              accessModes:
                - ReadWriteOnce
              resources:
                requests:
                  storage: 10Gi
        ```
    **Note:** If using the YAML View, the `volume_claim_template` field must be added to the spec
5. Select `Create`.
