# Restricted Network Usage
## Installation
To install the `koku-metrics-operator` in a restricted network, follow the [olm documentation](https://docs.openshift.com/container-platform/4.5/operators/admin/olm-restricted-networks.html). The operator is found in the `community-operators` Catalog in the `registry.redhat.io/redhat/community-operator-index:latest` Index. If pruning the index before pushing to the mirrored registry, keep the `koku-metrics-operator` package.

Within a restricted network, the operator queries prometheus to gather the necessary usage metrics, writes the query results to CSV files, and packages the reports for storage in the PVC. These reports then need to be manually downloaded from the cluster and uploaded to [cloud.redhat.com](https://cloud.redhat.com).

For more information, reach out to <cost-mgmt@redhat.com>.
## Configure the koku-metrics-operator for an air-gapped scenario
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
3. Specify the desired report retention. The operator will retain 30 reports by default. This corresponds to approximately one week's worth of data if using the default packaging cycle. To modify the number of retained reports:
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
  ```
  $ curl -vvvv -F "file=@FILE_NAME.tar.gz;type=application/vnd.redhat.hccm.tar+tgz"  https://cloud.redhat.com/api/ingress/v1/upload -u USERNAME:PASS
  ```
where `USERNAME` and `PASS` correspond to the user credentials for [cloud.redhat.com](https://cloud.redhat.com), and `FILE_NAME` is the name of the report to upload.
