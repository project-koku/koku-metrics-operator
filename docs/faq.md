# Frequently asked questions

## Our operator is stuck upgrading from v0.9.4 to v0.9.5. How can we force the upgrade?

1. Log into the cluster via the CLI
2. Update the KokuMetricsConfig from v1alpha1 to v1beta1
    ```
    $ oc get kokumetricsconfigs (this step is to get the name of the CR. If you know it, this step can be skipped)
    NAME                    AGE
    kokumetricscfg-sample   41m
    $ oc get kokumetricsconfigs kokumetricscfg-sample -o json > tmp.json
    $ head -3 tmp.json
    {
    "apiVersion": "koku-metrics-cfg.openshift.io/v1beta1", <--------- ensure that this says `v1beta1`. Update if needed.
    "kind": "KokuMetricsConfig",
    $ oc apply -f tmp.json
    Warning: oc apply should be used on resource created by either oc create --save-config or oc apply
    kokumetricsconfig.koku-metrics-cfg.openshift.io/kokumetricscfg-sample configured
    ```
    **Note**: apply the tmp.json even if you did not make changes to the file.

3. Verify that the KokuMetricsConfig CustomResourceDefinition contains the v1alpha1 and v1beta1 in the storedVersions status:
    ```
    $ oc get crd kokumetricsconfigs.koku-metrics-cfg.openshift.io -o jsonpath='{.status.storedVersions}{"\n"}'
    [v1alpha1 v1beta1]
    ```
4. In a second terminal window, open a proxy to the cluster (you can choose any open port number):
    ```
    $ oc proxy --port=9000
    Starting to serve on 127.0.0.1:9000
    ```
5. Back in the first terminal, update the status of the CRD to remove the v1alpha1:
    ```
    $ curl -X PATCH -H 'Content-Type: application/strategic-merge-patch+json' --data '{"status":{"storedVersions":["v1beta1"]}}' localhost:9000/apis/apiextensions.k8s.io/v1/customresourcedefinitions/kokumetricsconfigs.koku-metrics-cfg.openshift.io/status
    ```
6. Verify that the CRD status only contains v1beta1:
    ```
    $ oc get crd kokumetricsconfigs.koku-metrics-cfg.openshift.io -o jsonpath='{.status.storedVersions}{"\n"}'
    [v1beta1]
    ```
7. Uninstall the operator and reinstall from OperatorHub. This should be installed in the same namespace in which it was previously installed. Ensure that the old CR appears under KokuMetricsConfigs.
8. Close the proxy to the cluster.