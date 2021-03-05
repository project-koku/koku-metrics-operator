# v0.9.4 Koku Metrics Operator
## Introduction
The `koku-metrics-operator` is an OpenShift Operator used to collect OpenShift usage data and upload it to [cost managment](https://access.redhat.com/documentation/en-us/openshift_container_platform/4.5/html/getting_started_with_cost_management/assembly_introduction_cost_management). The operator queries Prometheus to create metric reports which are packaged and uploaded to cost management at [cloud.redhat.com](https://cloud.redhat.com).

This operator is capable of functioning within a disconnected/restricted network (aka air-gapped mode). In this mode, the operator will store the packaged reports for manual retrieval instead of being uploaded to cost management. Documentation for installing an operator within a restricted network can be found [here](https://docs.openshift.com/container-platform/4.5/operators/admin/olm-restricted-networks.html).

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

## Limitations and Pre-Requisites
#### Limitations (Potential for metrics data loss)
* A source **must** exist in cloud.redhat.com for an uploaded payload to be processed by cost management. The operator sends the payload to the Red Hat Insights Ingress service which usually returns successfully, but the operator does not currently confirm with cost management that the payload was processed. After Ingress accepts the uploaded payload, the payload is removed from the operator and is gone forever. If the data within the payload is not processed, a gap will be introduced in the usage metrics.

**Note** The following limitations are specific to operators configured to run in a restricted network:
* The `koku-metrics-operator` will not be able to generate new reports if the PVC storage is filled. If this occurs, the reports must be manually deleted from the PVC so that the operator can function as normal.
* The default report retention is 30 reports (about one week's worth of data). The reports must be manually downloaded and uploaded to cloud.redhat.com every week, or they will be deleted and the data will be lost.

#### Storage configuration prerequisite
The operator will attempt to create and use the following PVC when installed:

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

If a different PVC should be utilized, a valid PVC should be specified in the KokuMetricsConfig CR as described in the appropriate section below. The PVC to be used may exist already, or the operator will attempt to create it.

To use the default specification, the follow assumptions must be met:
1. A default StorageClass is defined.
2. Dynamic provisioning for that default StorageClass is enabled.

If these assumptions are not met, the operator will not deploy correctly. In these cases, storage must be manually configured. After configuring storage, a valid PVC template should be supplied in the `volume_claim_template` spec of the KokuMetricsConfig CR.

## Configure the koku-metrics-operator
**Note** There are separate instructions for configuring the `koku-metrics-operator` to run in a restricted network.
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
            name: <insert-name>
          spec:
            accessModes:
              - ReadWriteOnce
            resources:
              requests:
                storage: 10Gi
      ```

    **Note:** If using the YAML View, the `volume_claim_template` field must be added to the spec
5. Select `Create`.

# Restricted Network Usage (disconnected/air-gapped mode)
## Installation
To install the `koku-metrics-operator` in a restricted network, follow the [olm documentation](https://docs.openshift.com/container-platform/4.5/operators/admin/olm-restricted-networks.html). The operator is found in the `community-operators` Catalog in the `registry.redhat.io/redhat/community-operator-index:latest` Index. If pruning the index before pushing to the mirrored registry, keep the `koku-metrics-operator` package.

Within a restricted network, the operator queries prometheus to gather the necessary usage metrics, writes the query results to CSV files, and packages the reports for storage in the PVC. These reports then need to be manually downloaded from the cluster and uploaded to [cloud.redhat.com](https://cloud.redhat.com).

For more information, reach out to <cost-mgmt@redhat.com>.
## Configure the koku-metrics-operator for a restricted network
##### Create the KokuMetricsConfig
Configure the koku-metrics-operator by creating a `KokuMetricsConfig`.
1. On the left navigation pane, select `Operators` -> `Installed Operators` -> `koku-metrics-operator` -> `KokuMetricsConfig` -> `Create Instance`.
2. Specify the desired storage. If not specified, the operator will create a default Persistent Volume Claim called `koku-metrics-operator-data` with 10Gi of storage. To configure the koku-metrics-operator to use or create a different PVC, edit the following in the spec:
    * Add the desired configuration to the `volume_claim_template` field in the spec:

      ```
        volume_claim_template:
          apiVersion: v1
          kind: PersistentVolumeClaim
          metadata:
            name: <insert-name>
          spec:
            storageClassName: <insert-class-name>
            accessModes:
              - ReadWriteOnce
            resources:
              requests:
                storage: 10Gi
      ```

    **Note:** If using the YAML View, the `volume_claim_template` field must be added to the spec
3. (Optional) Specify the desired report retention. The operator will retain 30 reports by default. This corresponds to approximately one week's worth of data if using the default packaging cycle. To modify the number of retained reports:
    * Change the `packaging` spec field `max_reports_to_store` to the desired number of reports to retain. Once this max number is reached, the operator will start removing the oldest packages remaining on the PVC:

      ```
        packaging:
          max_size_MB: 100
          max_reports_to_store: 30
      ```

    **Note:** The number of retained reports directly affects the frequency that reports must be manually downloaded from the PVC. Take caution in setting this to a higher number of reports, as the operator cannot write data to the PVC if the storage is full.
4. To configure the operator to perform in a restricted network, set the `upload_toggle` to `false`:

  ```
    upload:
      upload_cycle: 360,
      upload_toggle: false
  ```

5. Select `Create`.

## Download reports from the Operator & clean up the PVC
If the `koku-metrics-operator` is configured to run in a restricted network, the metric reports will not automatically upload to cost managment. Instead, they need to be manually copied from the PVC for upload to [cloud.redhat.com](https://cloud.redhat.com). The default configuration saves one week of reports which means the process of downloading and uploading reports should be repeated weekly to prevent loss of metrics data. To download the reports, complete the following steps:
1. Create the following Pod, ensuring the `claimName` matches the PVC containing the report data:

  ```
    kind: Pod
    apiVersion: v1
    metadata:
      name: volume-shell
      namespace: koku-metrics-operator
    spec:
      volumes:
      - name: koku-metrics-operator-reports
        persistentVolumeClaim:
          claimName: koku-metrics-operator-data
      containers:
      - name: volume-shell
        image: busybox
        command: ['sleep', '3600']
        volumeMounts:
        - name: koku-metrics-operator-reports
          mountPath: /tmp/koku-metrics-operator-reports
  ```

2. Use rsync to copy all of the files ready for upload from the PVC to a local folder:

  ```
  $ oc rsync volume-shell:/tmp/koku-metrics-operator-reports/upload local/path/to/save/folder
  ```

3. Once confirming that the files have been successfully copied, use rsh to connect to the pod and delete the contents of the upload folder so that they are no longer in storage:

  ```
  $ oc rsh volume-shell
  $ rm /tmp/koku-metrics-operator-reports/upload/*
  ```

4. (Optional) Delete the pod that was used to connect to the PVC:

  ```
  $ oc delete -f volume-shell.yaml
  ```

## Create a source
In a restricted network, the `koku-metrics-operator` cannot automatically create a source. This process must be done manually. In the cloud.redhat.com platform, open the [Sources menu](https://cloud.redhat.com/settings/sources/) to begin adding an OpenShift source to cost management:

1. Navigate to the Sources menu
2. Select the `Red Hat sources` tab
3. Click `Add source` to open the Sources wizard.
4. Enter a name for the source and click `Next`.
5. Select `Red Hat Openshift Container Platform` as the source type and Cost Management as the application. Click `Next`.
6. Enter the cluster identifier into the cloud.redhat.com Sources wizard, and click `Next`.

    **Note:** The cluster identifier can be found in the KokuMetricsConfig CR, the cluster Overview page, or the cluster Help > About.

7. In the cloud.redhat.com Sources wizard, review the details and click `Finish` to create the Source.

## Upload the reports to cost managment
Uploading reports to cost managment is done through curl:

    $ curl -vvvv -F "file=@FILE_NAME.tar.gz;type=application/vnd.redhat.hccm.tar+tgz"  https://cloud.redhat.com/api/ingress/v1/upload -u USERNAME:PASS

where `USERNAME` and `PASS` correspond to the user credentials for [cloud.redhat.com](https://cloud.redhat.com), and `FILE_NAME` is the name of the report to upload.
