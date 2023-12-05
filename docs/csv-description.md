# Koku Metrics Operator
## Introduction
The `koku-metrics-operator` is a component of the [cost managment](https://access.redhat.com/documentation/en-us/cost_management_service) service for Openshift. The operator runs on the latest supported versions of Openshift. This operator obtains OpenShift usage data by querying Prometheus every hour to create metric reports that it uploads to Cost Management at [console.redhat.com](https://console.redhat.com) to be processed. For more information, reach out to <costmanagement@redhat.com>.

This operator is capable of functioning within a disconnected/restricted network (aka air-gapped mode). In this mode, the operator will store the packaged reports for manual retrieval instead of being uploaded to Cost Management. Documentation for installing an operator within a restricted network can be found [here](https://docs.openshift.com/container-platform/latest/operators/admin/olm-restricted-networks.html).

## Features and Capabilities
#### Metrics collection:
The Koku Metrics Operator (`koku-metrics-operator`) collects the metrics required for Cost Management by:
* Querying Prometheus to gather the necessary metrics for Cost Management.
* Writing the results of Prometheus queries to CSV report files.
* Packaging the CSV report files into tarballs.

#### Additional Capabilities:
* Resource Optimization metrics collection.
* The operator can be configured to gather all previous data within the configured retention period or a maximum of 90 days. The default data collection period is the 14 previous days. This setting is only applicable to newly created KokuMetricsConfigs.
* The operator can be configured to automatically upload the packaged reports to Cost Management through Red Hat Insights Ingress service.
* The operator can create an integration in console.redhat.com. An integration is required for Cost Management to process the uploaded packages.
* PersistentVolumeClaim (PVC) configuration: The KokuMetricsConfig CR can accept a PVC definition and the operator will create and mount the PVC. If one is not provided, a default PVC will be created.
* Restricted network installation: this operator can function on a restricted network. In this mode, the operator stores the packaged reports for manual retrieval.

## New in v3.1.0:
* Add service-account authentication type.
* __Deprecation Notice:__ Basic authentication is deprecated and will be removed in a future version of the operator.

## New in v3.0.0:
* Daily report generation: Operator versions prior to v3.0.0 generated sequential reports. Now, reports are generated starting at 0:00 UTC. Any payloads generated throughout a given day will contain all data starting from 0:00 UTC. Once the next day starts, the previous day's reports are packaged, and the new report again starts at 0:00 UTC for the current day.
* Failed query retry: In an attempt to prevent missing data, the operator will retry queries from the last successful query time, up to 5 times.

## New in v2.0.0:
* Adds metrics and report generation for resource optimization. This feature will collect additional usage metrics and create a new report in the payload. These metrics are enabled by default, but can be disabled by setting `disable_metrics_collection_resource_optimization` to `true`.
* Collect all available Prometheus data upon CR creation. This feature only applies to newly created KokuMetricsConfigs. The operator will check the monitoring stack configuration in the `openshift-monitoring` namespace. The operator will use the `retention` period set in the `cluster-monitoring-config` ConfigMap if defined, up to a maximum of 90 days. Otherwise it will fall back to collecting 14 days of data, if available. This data collection may be disabled by setting `collect_previous_data` to `false`. Turning this feature off results in the operator collecting metrics from the time the KokuMetricsConfig is created, forward.

## Limitations and Pre-Requisites
#### Limitations (Potential for metrics data loss)
* An integration **must** exist in console.redhat.com for an uploaded payload to be processed by Cost Management. The operator sends the payload to the Red Hat Insights Ingress service which usually returns successfully, but the operator does not currently confirm with Cost Management that the payload was processed. After Ingress accepts the uploaded payload, it is deleted from the operator. If the data within the payload is not processed, a gap will be introduced in the usage metrics. Data may be recollected by deleting the `KokuMetricsConfig`, creating a new `KokuMetricsConfig`, and setting `collect_previous_data: true`. This re-collection of data will gather all data stored in Prometheus, up to 90 days.

**Note** The following limitations are specific to operators configured to run in a restricted network:
* The `koku-metrics-operator` will not be able to generate new reports if the PVC storage is full. If this occurs, the reports must be manually deleted from the PVC so that the operator can function as normal.
* The default report retention is 30 reports (about one week's worth of data). The reports must be manually downloaded and uploaded to console.redhat.com every week, or they will be deleted and the data will be lost.

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

## Configurable parameters:
  * `authentication`:
    * `type: token` -> The authentication method for connecting to `console.redhat.com`. The default and preferred method is `token`. `basic` (deprecated) and `service-account` authentication methods are used when the openshift-config pull-secret does not contain a token for `console.redhat.com`.
    * `secret_name` -> The Secret used by the operator when the authentication type is `basic` (deprecated) or `service-account`. This parameter is required **only if** the authentication type is `basic` (deprecated) or `service-account`.
  * `packaging`:
    * `max_reports_to_store: 30` -> The number of reports to store when configured in air-gapped mode. The default is 30, with a minimum of 1 and no maximum. When the operator is not configured in air-gapped mode, this parameter has no effect. Reports are removed as soon as they are uploaded.
    * `max_size: 100` -> The maximum size for packaged files in Megabytes prior to compression. The default is 100, with a minimum of 1 and maximum of 100.
  * `prometheus_config`:
    * `collect_previous_data: true` -> Toggle for collecting all available data in Prometheus **upon KokuMetricsConfig creation** (This parameter will start to appear in KokuMetricsConfigs that were created prior to v2.0.0 but will not have any effect unless the KokuMetricsConfig is deleted and recreated). The default is `true`. The operator will first look for a `retention` period in the `cluster-monitoring-config` ConfigMap in the `openshift-monitoring` namespace and gather data over this time period up to a maximum of 90 days. If this configuration is not set, the default is 14 days. (New in v2.0.0)
    * `disable_metrics_collection_cost_management: false` -> Toggle for disabling the collection of metrics for Cost Management. The default is false. (New in v2.0.0)
    * `disable_metrics_collection_resource_optimization: false` -> Toggle for disabling the collection of metrics for Resource Optimization. The default is false. (New in v2.0.0)
    * `context_timeout: 120` -> The time in seconds before Prometheus queries timeout due to exceeding context timeout. The default is 120, with a minimum of 10 and maximum of 180.
  * `source`:
    * `name: ''` -> The name of the integration the operator will create in `console.redhat.com`. If the name value is empty, the default intergration name is the **cluster id**.
    * `create_source: false` -> Toggle for whether or not the operator will create the integration in `console.redhat.com`. The default is False. This parameter should be switched to True when an integration does not already exist in `console.redhat.com` for this cluster.
    * `check_cycle: 1440` -> The time in minutes to wait between checking if an integration exists for this cluster. The default is 1440 minutes (24 hrs).
  * `upload`:
    * `upload_cycle: 360` -> The time in minutes between payload uploads. The default is 360 (6 hours).
    * `upload_toggle: true` -> Toggle to turn upload on or off -> true means upload, false means do not upload (false == air-gapped mode). The default is `true`.
    * `upload_wait` -> The amount of time (in seconds) to pause before uploading a payload. The default is a random number between 0 and 35. This is used to decrease service load, but may be set to `0` if desired.
  * `volume_claim_template` -> see the "Storage configuration prerequisite" section above.

## Configure the koku-metrics-operator
**Note** There are separate instructions for configuring the `koku-metrics-operator` to run in a restricted network.
##### Configure authentication
The default authentication for the operator is `token`. No further steps are required to configure token authentication. If `basic` (deprecated) or `service-account` is the preferred authentication method, a Secret which holds the credentials must be created:
1. On the left navigation pane, select `Workloads` -> `Secrets` -> select Project: `koku-metrics-operator` -> `Create` -> `Key/Value Secret`
2. Give the Secret a name and add 2 keys (all lowercase) for the respective authentication type. The values for these keys correspond to console.redhat.com credentials:
    * basic auth (deprecated): `username` and `password`
    * service-account auth: `client_id` and `client_secret` 

3. Select `Create`.
##### Create the KokuMetricsConfig
Configure the koku-metrics-operator by creating a `KokuMetricsConfig`.
1. On the left navigation pane, select `Operators` -> `Installed Operators` -> `koku-metrics-operator` -> `KokuMetricsConfig` -> `Create Instance`.
2. For `basic` (deprecated) or `service-account` authentication, edit the following values in the spec:
    * Replace `authentication: type:` with `basic` or `service-account`.
    * Add the `secret_name` field under `authentication`, and set it equal to the name of the authentication Secret that was created above. The spec should look similar to the following:

        * for basic auth type (deprecated)
        ```
          authentication:
            secret_name: SECRET-NAME
            type: basic
        ```
          
        * for service-account auth type
        ```
          authentication:
            secret_name: SECRET-NAME
            type: service-account
        ```

3. To configure the koku-metrics-operator to create a cost management integration, edit the following values in the `source` field:
    * Replace the `name` field value with the preferred name of the integration to be created.
    * Replace the `create_source` field value with `true`.

    **Note:** if the integration already exists, replace the empty string value of the `name` field with the existing name, and leave `create_source` as false. This will allow the operator to confirm that the integration exists.

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
To install the `koku-metrics-operator` in a restricted network, follow the [olm documentation](https://docs.openshift.com/container-platform/latest/operators/admin/olm-restricted-networks.html). The operator is found in the `community-operators` Catalog in the `registry.redhat.io/redhat/community-operator-index:latest` Index. If pruning the index before pushing to the mirrored registry, keep the `koku-metrics-operator` package.

Within a restricted network, the operator queries prometheus to gather the necessary usage metrics, writes the query results to CSV files, and packages the reports for storage in the PVC. These reports then need to be manually downloaded from the cluster and uploaded to [console.redhat.com](https://console.redhat.com).

## Configure the koku-metrics-operator for a restricted network
##### Create the KokuMetricsConfig
Configure the koku-metrics-operator by creating a `KokuMetricsConfig`.
1. On the left navigation pane, select `Operators` -> `Installed Operators` -> `koku-metrics-operator` -> `KokuMetricsConfig` -> `Create Instance`.
2. Specify the desired storage. If not specified, the operator will create a default Persistent Volume Claim called `koku-metrics-operator-data` with 10Gi of storage. To configure the koku-metrics-operator to use or create a different PVC, edit the following in the spec:
    * Add the desired configuration to the `volume_claim_template` field in the spec (below is only a template. Any _valid_ `PersistentVolumeClaim` may be defined here):

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
      upload_cycle: 360
      upload_toggle: false
  ```

5. Select `Create`.

## Download reports from the Operator & clean up the PVC
If the `koku-metrics-operator` is configured to run in a restricted network, the metric reports will not automatically upload to cost managment. Instead, they need to be manually copied from the PVC for upload to [console.redhat.com](https://console.redhat.com). The default configuration saves one week of reports which means the process of downloading and uploading reports should be repeated weekly to prevent loss of metrics data. To download the reports, complete the following steps:
1. Create the following Pod, ensuring the `claimName` matches the PVC containing the report data:

  ```
    kind: Pod
    apiVersion: v1
    metadata:
      name: volume-shell
      namespace: koku-metrics-operator
      labels:
        app: koku-metrics-operator
    spec:
      volumes:
      - name: koku-metrics-operator-reports
        persistentVolumeClaim:
          claimName: koku-metrics-operator-data
      containers:
      - name: volume-shell
        image: busybox
        command: ['sleep', 'infinity']
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

## Create an Integration
In a restricted network, the `koku-metrics-operator` cannot automatically create an integration. This process must be done manually. In the console.redhat.com platform, open the [Integrations menu](https://console.redhat.com/settings/integrations/) to begin adding an OpenShift integration to Cost Management:

Prerequisites:
* The cluster identifier which can be found in the KokuMetricsConfig CR, the cluster Overview page, or the cluster Help > About.

Creating an integration:
1. Navigate to the Integrations menu
2. Select the `Red Hat` tab
3. Create a new `Red Hat Openshift Container Platform` integration:
    * give the integration a unique name
    * add the Cost Management application
    * add the cluster identifier
4. In the Source wizard, review the details and click `Finish` to create the source.

## Upload the reports to cost managment
Uploading reports to cost managment is done through curl:

    $ curl -vvvv -F "file=@FILE_NAME.tar.gz;type=application/vnd.redhat.hccm.tar+tgz"  https://console.redhat.com/api/ingress/v1/upload -u USERNAME:PASS

where `USERNAME` and `PASS` correspond to the user credentials for [console.redhat.com](https://console.redhat.com), and `FILE_NAME` is the name of the report to upload.
