# Air-Gapped Operator Instructions
## Installation
To install the `koku-metrics-operator` in a restricted network, follow the [olm documentation](https://docs.openshift.com/container-platform/4.5/operators/admin/olm-restricted-networks.html). The functionality of the Operator in a restricted network is to query Prometheus to create metric reports, which are then packaged and stored on a PVC. These reports can then be manually downloaded and uploaded to cost management at [cloud.redhat.com](https://cloud.redhat.com). For more information, reach out to <cost-mgmt@redhat.com>.
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
3. Specify the desired report retention. The operator will retain 30 reports by default. This is about one week's worth of data if using the default cycles. To edit the number of retained reports, edit the packaging field in the spec: 
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

## Downloading reports from the Operator & cleaning up the PVC
If the `koku-metrics-operator` is configured to run in a restricted network, the metric reports will not be automatically uploaded to cost managment. Instead, they will need to be manually copied from the PVC to later be uploaded to [cloud.redhat.com](https://cloud.redhat.com). The default report retention is set to 30 which means the process of downloading and uploading reports should be repeated weekly so as not to lose metric data. In order to connect to the PVC and download the reports, complete the following steps: 
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
2. Confirm that the `claimName` matches the PVC that the operator is configured to use. If no PVC was specified during operator configuration, this should remain `koku-metrics-operator-data`. 
3. Deploy the pod:
    ```
    $ oc create -f volume-shell.yaml
    ```
4. Use rsync to copy all of the files ready for upload from the PVC to a local folder: 
    ```
    $ oc rsync volume-shell:/tmp/koku-metrics-operator-reports/upload local/path/to/save/folder
    ```
5. Once confirming that the files have been successfully copied, use rsh to connect to the pod and delete the upload folder so that they are no longer taking up storage: 
    ```
    $ oc rsh volume-shell
    $ cd  /tmp/koku-metrics-operator-reports/
    $ rm -rf upload/
    ```
6. Delete the pod that was used to connect to the PVC: 
    ```
    $ oc delete -f volume-shell.yaml
    ```
## Creating a source 
In a restricted network, the `koku-metrics-operator` cannot automatically create a source. This process must be done manually. In the cloud.redhat.com platform, open the [Sources menu](https://cloud.redhat.com/settings/sources/) to begin adding an OpenShift source to cost management:

Navigate to Sources and click `Add source` to open the Sources wizard.
Enter a name for your source and click `Next`.
Select Cost Management as the application and OpenShift Container Platform as the source type. Click `Next`.

When prompted, enter the cluster identifier into the cloud.redhat.com Sources wizard, and click `Next`.

  **Note:** The cluster identifier can be found in Help > About in OpenShift.

In the cloud.redhat.com Sources wizard, review the details and click `Finish` to add the OpenShift Container Platform cluster to cost management.

## Uploading the reports to cost managment 
1. Uploading reports to cost managment can be done through curl: 
  * To use basic authentication, replace the `USERNAME` and `PASS` with your username and password for [cloud.redhat.com](https://cloud.redhat.com): 

    ```
    $ curl -vvvv -F "file=@FILE_NAME.tar.gz;type=application/vnd.redhat.hccm.tar+tgz"  https://cloud.redhat.com/api/ingress/v1/upload -u USERNAME:PASS
    ```

  * To use token authentication, log in to [access.redhat.com](https://access.redhat.com/management/api) and generate a token. Replace `GENERATED_TOKEN` with the token that you have generated and replace the `CLUSTER_ID` with your cluster ID in the following curl command: 

    ```
    $ curl -vvvv -F "file=@FILE_NAME.tar.gz;type=application/vnd.redhat.hccm.tar+tgz" https://cloud.redhat.com/api/ingress/v1/upload -H "Authorization: Bearer GENERATED_TOKEN" -H "User-Agent: cost-mgmt-operator/9ec0b9f48045ee0f9e4137e54dd01eddea2455c4 cluster/CLUSTER_ID"
    ```
  **Note:** Regardless of the authentication method you choose, replace the `FILE_NAME` with the file that you want to upload. 