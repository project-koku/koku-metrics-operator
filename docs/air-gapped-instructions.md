# Air-Gapped Operator Instructions
## Installation
To install the `koku-metrics-operator` in a restricted network, follow the [olm documentation](https://docs.openshift.com/container-platform/4.5/operators/admin/olm-restricted-networks.html). The functionality of the Operator in a restricted network is to query Prometheus to create metric reports, which are then packaged and stored on a PVC. These reports can then be manually downloaded and uploaded to cost management at [cloud.redhat.com](https://cloud.redhat.com). For more information, reach out to <cost-mgmt@redhat.com>.
## Configure the koku-metrics-operator for an air-gapped scenario
The operator can be configured through either the UI or CLI:
#### Configure through the UI
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
3. Specify the desired report retention. The operator will retain 30 reports by default. This is about one weeks worth of data if using the default cycles. To edit the number of retained reports, edit the packaging field in the spec: 
    * Edit the `max_reports_to_store` field within the `packaging` field in the spec to be the desired number of reports to retain. Once this max number is reached, the operator will delete the oldest report on the PVC:

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

#### Configure through the CLI
##### Create the KokuMetricsConfig
Configure the koku-metrics-operator by creating a `KokuMetricsConfig`. Note that for an air-gapped installation, the `upload_toggle` field must be set to false as it is in the following yaml. 
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
        max_reports_to_store: 30
      prometheus_config: {}
      source:
        check_cycle: 1440,
        create_source: false,
        name: INSERT-SOURCE-NAME
      upload:
        upload_cycle: 360,
        upload_toggle: false
    ```

2. Specify the desired storage configuration. If not specified, the operator will create a default Persistent Volume Claim called `koku-metrics-operator-data` with 10Gi of storage. To configure the koku-metrics-operator to use or create a different PVC, edit the following in the spec: 
    * Add the `volume_claim_template` field in the spec and specify the desired PVC configuration:

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
3. Specify the desired report retention. The operator will retain 30 reports by default. This is about one weeks worth of data if using the default cycles. 
    * To change the number of retained reports, edit the `max_reports_to_store` field within the `packaging` field in the spec to be the desired number of reports to retain. Once this max number is reached, the operator will delete the oldest report on the PVC:
4. Deploy the `KokuMetricsConfig` resource:
    ```
    $ oc create -f kokumetricsconfig.yaml
    ```
## Downloading reports from the Operator/ cleaning reports up
The operator must be installed in the `koku-metrics-operator` namespace. Installing the operator through OperatorHub will create the namespace automatically, or it can be created through either the UI or CLI:
1. Copy the following pod resource template and save it to a file called `volume-shell.yaml`:

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
2. If a PVC other than the default was specified within the operator configuration, change the claimName to the name of the specified PVC.
3. Deploy the pod resource:
    ```
    $ oc create -f volume-shell.yaml
    ```
4. Copy the files from the PVC to a local folder: 
  ```
  oc rsync volume-shell:/tmp/koku-metrics-operator-reports/upload local/path/to/save/folder
  ```
5. Once confirming that the files have been successfully copied, use rsh to connect to the pod and delete the upload folder: 
  ```
  oc rsh volume-shell
  cd  /tmp/koku-metrics-operator-reports/
  rm -rf upload/
  ```
## Uploading the reports to cost managment 
1. Use curl along with your username and password to upload the reports to cloud.redhat.com: 
```
curl -vvvv -F "file=@FILE_NAME.tar.gz;type=application/vnd.redhat.qpc.tar+tgz"  https://cloud.redhat.com/api/ingress/v1/upload -u USERNAME:PASS
```
 **Note:** Replace the `FILE_NAME` with the file that you want to upload. 
