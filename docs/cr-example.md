The following shows a complete CR and gives a brief description of each spec field. Every `spec` field is optional.

```
apiVersion: koku-metrics-cfg.openshift.io/v1alpha1
kind: KokuMetricsConfig
metadata:
  name: kokumetricsconfig-sample
spec:
  api_url: string # default=https://cloud.redhat.com, the url of the API endpoint for service interaction
  clusterID: string # The cluster ID -> the reconciler finds this value if not supplied
  validate_cert: bool # default=false, represent if the Ingress endpoint must be certificate validated
  authentication:
    type: choice (basic, token) # default=token
    secret_name: string # secret which contains user/password for basic auth
  packaging:
    max_size: int # default=100, max size in Megabytes for packaged files
  prometheus_config:
    service_address: string # default=https://thanos-querier.openshift-monitoring.svc:9091, route to thanos-querier
    skip_tls_verification: bool # default=false, do TLS verification for prometheus queries
  source:
    sources_path: string # default=/api/sources/v1.0/, path to sources API
    name: string # name of source in cloud.redhat.com
    create_source: bool # default=false, create the source or not
    check_cycle: int # default=1440, time in minutes to wait between source checks.
  upload: # optional
    ingress_path: string # default=/api/ingress/v1/upload/, the path of the Ingress API service
    upload_wait: int # time to wait before uploading
    upload_cycle: int # default=360 , time in minutes between uploads
    upload_toggle: bool # default=true, turn upload on or off -> true means upload, false means do not upload
```