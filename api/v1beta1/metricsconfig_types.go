//
// Copyright 2021 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

const (
	// CertIgnore allows certificate validation to be bypassed.
	CertIgnore bool = false

	// CertCheck allows certificate validation to occur.
	CertCheck bool = true

	// UploadOn sets the operator to upload to console.redhat.com.
	UploadOn bool = true

	// UploadOff sets the operator to not upload to console.redhat.com.
	UploadOff bool = false

	//UploadCycle sets the default cycle to be 360 minutes (6 hours).
	UploadSchedule int64 = 360

	//SourceCheckSchedule sets the default cycle to be 1440 minutes (24 hours).
	SourceCheckSchedule int64 = 1440

	//PackagingMaxSize sets the default max file size to be 100 MB
	PackagingMaxSize int64 = 100
)

// AuthenticationType describes how the upload will be handled.
// Only one of the following authentication types may be specified.
// If none of the following types are specified, the default one
// is Token.
// +kubebuilder:validation:Enum=token;basic
type AuthenticationType string

const (
	// Basic allows upload of data using basic authentication.
	Basic AuthenticationType = "basic"

	// Token allows upload of data using token authentication.
	Token AuthenticationType = "token"
)

// EmbeddedObjectMetadata contains a subset of the fields included in k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta
// Only fields which are relevant to embedded resources are included.
type EmbeddedObjectMetadata struct {
	// Name must be unique within a namespace. Is required when creating resources, although
	// some resources may allow a client to request the generation of an appropriate name
	// automatically. Name is primarily intended for creation idempotence and configuration
	// definition.
	// Cannot be updated.
	// More info: http://kubernetes.io/docs/user-guide/identifiers#names
	// +optional
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`

	// Map of string keys and values that can be used to organize and categorize
	// (scope and select) objects. May match selectors of replication controllers
	// and services.
	// More info: http://kubernetes.io/docs/user-guide/labels
	// +optional
	Labels map[string]string `json:"labels,omitempty" protobuf:"bytes,11,rep,name=labels"`

	// Annotations is an unstructured key value map stored with a resource that may be
	// set by external tools to store and retrieve arbitrary metadata. They are not
	// queryable and should be preserved when modifying objects.
	// More info: http://kubernetes.io/docs/user-guide/annotations
	// +optional
	Annotations map[string]string `json:"annotations,omitempty" protobuf:"bytes,12,rep,name=annotations"`
}

// EmbeddedPersistentVolumeClaim is an embedded version of k8s.io/api/core/v1.PersistentVolumeClaim.
// It contains TypeMeta and a reduced ObjectMeta.
type EmbeddedPersistentVolumeClaim struct {
	metav1.TypeMeta `json:",inline"`

	// EmbeddedMetadata contains metadata relevant to an EmbeddedResource.
	EmbeddedObjectMetadata `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Spec defines the desired characteristics of a volume requested by a pod author.
	// More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims
	// +optional
	Spec corev1.PersistentVolumeClaimSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}

// AuthenticationSpec defines the desired state of Authentication object in the KokuMetricsConfigSpec.
type AuthenticationSpec struct {

	// AuthType is a field of KokuMetricsConfig to represent the authentication type to be used basic or token.
	// Valid values are:
	// - "basic" : Enables authentication using user and password from authentication secret.
	// - "token" (default): Uses cluster token for authentication.
	// +kubebuilder:default="token"
	AuthType AuthenticationType `json:"type"`

	// AuthenticationSecretName is a field of KokuMetricsConfig to represent the secret with the user and password used for uploads.
	// +optional
	AuthenticationSecretName string `json:"secret_name,omitempty"`
}

// PackagingSpec defines the desired state of the Packaging object in the KokuMetricsConfigSpec.
type PackagingSpec struct {

	// MaxSize is a field of KokuMetricsConfig to represent the max file size in megabytes that will be compressed for upload to Ingress.
	// The default is 100.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:default=100
	MaxSize int64 `json:"max_size_MB"`

	// MaxReports is a field of KokuMetricsConfig to represent the maximum number of reports to store.
	// The default is 30 reports which corresponds to approximately 7 days worth of data given the other default values.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=30
	MaxReports int64 `json:"max_reports_to_store"`
}

// UploadSpec defines the desired state of Authentication object in the KokuMetricsConfigSpec.
type UploadSpec struct {

	// FOR DEVELOPMENT ONLY.
	// IngressAPIPath is a field of KokuMetricsConfig to represent the path of the Ingress API service.
	// The default is `/api/ingress/v1/upload`.
	// +kubebuilder:default=`/api/ingress/v1/upload`
	IngressAPIPath string `json:"ingress_path"`

	// UploadWait is a field of KokuMetricsConfig to represent the time to wait before sending an upload.
	// +optional
	// +kubebuilder:validation:Minimum=0
	UploadWait *int64 `json:"upload_wait,omitempty"`

	// UploadCycle is a field of KokuMetricsConfig to represent the number of minutes between each upload schedule.
	// The default is 360 min (6 hours).
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=360
	UploadCycle *int64 `json:"upload_cycle"`

	// UploadToggle is a field of KokuMetricsConfig to represent if the operator is installed in a restricted-network.
	// If `false`, the operator will not upload to console.redhat.com or check/create sources.
	// The default is true.
	// +kubebuilder:default=true
	UploadToggle *bool `json:"upload_toggle"`

	// ValidateCert is a field of KokuMetricsConfig to represent if the Ingress endpoint must be certificate validated.
	// +kubebuilder:default=true
	ValidateCert *bool `json:"validate_cert"`
}

// PrometheusSpec defines the desired state of PrometheusConfig object in the KokuMetricsConfigSpec.
type PrometheusSpec struct {

	// ContextTimeout is a field of KokuMetricsConfig to represent how long a query to prometheus should run in seconds before timing out.
	// The default is 120 seconds.
	// +kubebuilder:validation:Minimum=10
	// +kubebuilder:validation:Maximum=180
	// +kubebuilder:default=120
	ContextTimeout *int64 `json:"context_timeout,omitempty"`

	// CollectPreviousData is a field of KokuMetricsConfig to represent whether or not the operator will gather previous data upon KokuMetricsConfig
	// creation. This toggle only changes operator behavior when a new KokuMetricsConfig is created. When `true`, the operator will gather all
	// existing Prometheus data for the current month. The default is true.
	// +kubebuilder:default=true
	CollectPreviousData *bool `json:"collect_previous_data,omitempty"`

	// DisableMetricsCollectionCostManagement is a field of KokuMetricsConfig to represent whether or not the operator will generate
	// reports for cost-management metrics. The default is false.
	// +kubebuilder:default=false
	DisableMetricsCollectionCostManagement *bool `json:"disable_metrics_collection_cost_management,omitempty"`

	// DisableMetricsCollectionResourceOptimization is a field of KokuMetricsConfig to represent whether or not the operator will generate
	// reports for resource-optimization metrics. The default is false.
	// +kubebuilder:default=false
	DisableMetricsCollectionResourceOptimization *bool `json:"disable_metrics_collection_resource_optimization,omitempty"`

	// FOR DEVELOPMENT ONLY.
	// SvcAddress is a field of KokuMetricsConfig to represent the thanos-querier address.
	// The default is `https://thanos-querier.openshift-monitoring.svc:9091`.
	// +kubebuilder:default=`https://thanos-querier.openshift-monitoring.svc:9091`
	SvcAddress string `json:"service_address"`

	// FOR DEVELOPMENT ONLY.
	// SkipTLSVerification is a field of KokuMetricsConfig to represent if the thanos-querier endpoint must be certificate validated.
	// The default is false.
	// +kubebuilder:default=false
	SkipTLSVerification *bool `json:"skip_tls_verification"`
}

// CloudDotRedHatSourceSpec defines the desired state of CloudDotRedHatSource object in the KokuMetricsConfigSpec.
type CloudDotRedHatSourceSpec struct {

	// FOR DEVELOPMENT ONLY.
	// sources_path is the prefix of the Sources API on console.redhat.com.
	// The default is `/api/sources/v1.0/`.
	// +kubebuilder:default=`/api/sources/v1.0/`
	SourcesAPIPath string `json:"sources_path"`

	// name is the desired name of the integration to create on console.redhat.com.
	// +optional
	SourceName string `json:"name,omitempty"`

	// create_source toggles the creation of the integration on console.redhat.com.
	// +kubebuilder:default=false
	CreateSource *bool `json:"create_source"`

	// check_cycle is the number of minutes between each integration status check on console.redhat.com.
	// The default is 1440 min (24 hours).
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=1440
	CheckCycle *int64 `json:"check_cycle"`
}

// KokuMetricsConfigSpec defines the desired state of KokuMetricsConfig.
type KokuMetricsConfigSpec struct {
	// +kubebuilder:validation:preserveUnknownFields=false

	// ClusterID is a field of KokuMetricsConfig to represent the cluster UUID. Normally this value should not be
	// specified. Only set this value if the clusterID cannot be obtained from the ClusterVersion.
	// +optional
	ClusterID string `json:"clusterID,omitempty"`

	// ClusterVersion is a field of KokuMetricsConfig to represent the cluster version. Normally this value should not be
	// specified. Only set this value if the clusterVersion cannot be obtained from the ClusterVersion.
	// +optional
	ClusterVersion string `json:"clusterVersion,omitempty"`

	// FOR DEVELOPMENT ONLY.
	// APIURL is a field of KokuMetricsConfig to represent the url of the API endpoint for service interaction.
	// The default is `https://console.redhat.com`.
	// +kubebuilder:default=`https://console.redhat.com`
	APIURL string `json:"api_url,omitempty"`

	// Authentication is a field of KokuMetricsConfig to represent the authentication object.
	Authentication AuthenticationSpec `json:"authentication"`

	//Packaging is a field of KokuMetricsConfig to represent the packaging object.
	Packaging PackagingSpec `json:"packaging"`

	// Upload is a field of KokuMetricsConfig to represent the upload object.
	Upload UploadSpec `json:"upload"`

	// PrometheusConfig is a field of KokuMetricsConfig to represent the configuration of Prometheus connection.
	PrometheusConfig PrometheusSpec `json:"prometheus_config"`

	// source represents the desired integration on console.redhat.com.
	Source CloudDotRedHatSourceSpec `json:"source"`

	// VolumeClaimTemplate is a field of KokuMetricsConfig to represent a PVC template.
	VolumeClaimTemplate *EmbeddedPersistentVolumeClaim `json:"volume_claim_template,omitempty"`
}

// AuthenticationStatus defines the desired state of Authentication object in the KokuMetricsConfigStatus.
type AuthenticationStatus struct {

	// AuthType is a field of KokuMetricsConfig to represent the authentication type to be used basic or token.
	AuthType AuthenticationType `json:"type,omitempty"`

	// AuthenticationSecretName is a field of KokuMetricsConfig to represent the secret with the user and password used for uploads.
	AuthenticationSecretName string `json:"secret_name,omitempty"`

	// AuthenticationCredentialsFound is a field of KokuMetricsConfig to represent if used for uploads were found.
	AuthenticationCredentialsFound *bool `json:"credentials_found,omitempty"`

	// ValidBasicAuth is a field of KokuMetricsConfig to represent if the given basic auth credentials are valid.
	ValidBasicAuth *bool `json:"valid_basic_auth,omitempty"`

	// AuthErrorMessage is a field of KokuMetricsConfig to represent an `invalid credentials` error message.
	AuthErrorMessage string `json:"error,omitempty"`

	// LastVerificationTime is a field of KokuMetricsConfig to represent the last time credentials were verified.
	// +nullable
	LastVerificationTime *metav1.Time `json:"last_credential_verification_time,omitempty"`
}

// PackagingStatus defines the observed state of the Packing object in the KokuMetricsConfigStatus.
type PackagingStatus struct {

	// LastSuccessfulPackagingTime is a field of KokuMetricsConfig that shows the time of the last successful file packaging.
	// +nullable
	LastSuccessfulPackagingTime metav1.Time `json:"last_successful_packaging_time,omitempty"`

	// MaxReports is a field of KokuMetricsConfig to represent the maximum number of reports to store.
	MaxReports *int64 `json:"max_reports_to_store,omitempty"`

	// MaxSize is a field of KokuMetricsConfig to represent the max file size in megabytes that will be compressed for upload to Ingress.
	MaxSize *int64 `json:"max_size_MB,omitempty"`

	// PackagedFiles is a field of KokuMetricsConfig to represent the list of file packages in storage.
	PackagedFiles []string `json:"packaged_files,omitempty"`

	// PackagingError is a field of KokuMetricsConfig to represent the error encountered packaging the reports.
	PackagingError string `json:"error,omitempty"`

	// ReportCount is a field of KokuMetricsConfig to represent the number of reports in storage.
	ReportCount *int64 `json:"number_reports_stored,omitempty"`
}

// UploadStatus defines the observed state of Upload object in the KokuMetricsConfigStatus.
type UploadStatus struct {

	// IngressAPIPath is a field of KokuMetricsConfig to represent the path of the Ingress API service.
	// +optional
	IngressAPIPath string `json:"ingress_path,omitempty"`

	// UploadToggle is a field of KokuMetricsConfig to represent if the operator should upload to console.redhat.com.
	// The default is true
	UploadToggle *bool `json:"upload,omitempty"`

	// UploadWait is a field of KokuMetricsConfig to represent the time to wait before sending an upload.
	UploadWait *int64 `json:"upload_wait,omitempty"`

	// UploadCycle is a field of KokuMetricsConfig to represent the number of minutes between each upload schedule.
	// The default is 360 min (6 hours).
	UploadCycle *int64 `json:"upload_cycle,omitempty"`

	// UploadError is a field of KokuMetricsConfigStatus to represent the error encountered uploading reports.
	// +optional
	UploadError string `json:"error,omitempty"`

	// LastUploadStatus is a field of KokuMetricsConfig that shows the http status of the last upload.
	LastUploadStatus string `json:"last_upload_status,omitempty"`

	// LastPayloadName is a field of KokuMetricsConfig that shows the name of the last payload file.
	LastPayloadName string `json:"last_payload_name,omitempty"`

	// LastPayloadManifest is a field of KokuMetricsConfig that shows the manifestID of the last payload.
	LastPayloadManifestID string `json:"last_payload_manifest_id,omitempty"`

	// LastPayloadRequestID is a field of KokuMetricsConfig that shows the insights request id of the last payload.
	LastPayloadRequestID string `json:"last_payload_request_id,omitempty"`

	// LastPayloadFiles is a field of KokuMetricsConfig to represent the list of files in the last payload that was sent.
	LastPayloadFiles []string `json:"last_payload_files,omitempty"`

	// LastSuccessfulUploadTime is a field of KokuMetricsConfig that shows the time of the last successful upload.
	// +nullable
	LastSuccessfulUploadTime metav1.Time `json:"last_successful_upload_time,omitempty"`

	// ValidateCert is a field of KokuMetricsConfig to represent if the Ingress endpoint must be certificate validated.
	ValidateCert *bool `json:"validate_cert,omitempty"`
}

// CloudDotRedHatSourceStatus defines the observed state of CloudDotRedHatSource object in the KokuMetricsConfigStatus.
type CloudDotRedHatSourceStatus struct {

	// sources_path is the prefix of the Sources API on console.redhat.com.
	// +optional
	SourcesAPIPath string `json:"sources_path,omitempty"`

	// name represents the name of the integration that the operator attempted to create on console.redhat.com.
	// +optional
	SourceName string `json:"name,omitempty"`

	// source_defined represents whether the defined integration name exists on console.redhat.com.
	// +optional
	SourceDefined *bool `json:"source_defined,omitempty"`

	// create_source represents the toggle used during the creation of the integration on console.redhat.com.
	// An Integration will not be created if upload_toggle is `false`.
	// +optional
	CreateSource *bool `json:"create_source,omitempty"`

	// error represents any errors encountered when creating the integration.
	// +optional
	SourceError string `json:"error,omitempty"`

	// last_check_time is the time that the last integration status check was attempted.
	// +nullable
	LastSourceCheckTime metav1.Time `json:"last_check_time,omitempty"`

	// check_cycle is the number of minutes between each integration status check on console.redhat.com.
	// The default is 1440 min (24 hours).
	CheckCycle *int64 `json:"check_cycle,omitempty"`
}

// PrometheusStatus defines the status for querying prometheus.
type PrometheusStatus struct {

	// PrometheusConfigured is a field of KokuMetricsConfigStatus to represent if the operator is configured to connect to prometheus.
	PrometheusConfigured bool `json:"prometheus_configured"`

	// ConfigError is a field of KokuMetricsConfigStatus to represent errors during prometheus configuration.
	ConfigError string `json:"configuration_error,omitempty"`

	// PrometheusConnected is a field of KokuMetricsConfigStatus to represent if prometheus can be queried.
	PrometheusConnected bool `json:"prometheus_connected"`

	// ContextTimeout is a field of KokuMetricsConfigState to represent how long a query to prometheus should run in seconds before timing out.
	ContextTimeout *int64 `json:"context_timeout,omitempty"`

	// ConnectionError is a field of KokuMetricsConfigStatus to represent errors during prometheus test query.
	ConnectionError string `json:"prometheus_connection_error,omitempty"`

	// LastQueryStartTime is a field of KokuMetricsConfigStatus to represent the last time queries were started.
	// +nullable
	LastQueryStartTime metav1.Time `json:"last_query_start_time,omitempty"`

	// LastQuerySuccessTime is a field of KokuMetricsConfigStatus to represent the last time queries were successful.
	// +nullable
	LastQuerySuccessTime metav1.Time `json:"last_query_success_time,omitempty"`

	// PreviousDataCollected is a field of KokuMetricsConfigStatus to represent whether or not the operator gathered the available Prometheus
	// data upon KokuMetricsConfig creation.
	// +kubebuilder:default=false
	PreviousDataCollected bool `json:"previous_data_collected,omitempty"`

	// DisabledMetricsCollectionCostManagement is a field of KokuMetricsConfigStatus to represent whether or not collecting
	// cost-management metrics is disabled. The default is false.
	// +kubebuilder:default=false
	DisabledMetricsCollectionCostManagement *bool `json:"disabled_metrics_collection_cost_management,omitempty"`

	// DisabledMetricsCollectionResourceOptimization is a field of KokuMetricsConfigStatus to represent whether or not collecting
	// resource-optimzation metrics is disabled. The default is true.
	// +kubebuilder:default=true
	DisabledMetricsCollectionResourceOptimization *bool `json:"disabled_metrics_collection_resource_optimization,omitempty"`

	// SvcAddress is the internal thanos-querier address.
	SvcAddress string `json:"service_address,omitempty"`

	// SkipTLSVerification is a field of KokuMetricsConfigStatus to represent if the thanos-querier endpoint must be certificate validated.
	SkipTLSVerification *bool `json:"skip_tls_verification,omitempty"`
}

// ReportsStatus defines the status for generating reports.
type ReportsStatus struct {

	// ReportMonth is a field of KokuMetricsConfigStatus to represent the month for which reports are being generated.
	ReportMonth string `json:"report_month,omitempty"`

	// LastHourQueried is a field of KokuMetricsConfigStatus to represent the time range for which metrics were last queried.
	LastHourQueried string `json:"last_hour_queried,omitempty"`

	// DataCollected is a field of KokuMetricsConfigStatus to represent whether or not data was collected for the last query.
	DataCollected bool `json:"data_collected,omitempty"`

	// DataCollectionMessage is a field of KokuMetricsConfigStatus to represent a message associated with the data_collected status.
	DataCollectionMessage string `json:"data_collection_message,omitempty"`
}

// StorageStatus defines the status for storage.
type StorageStatus struct {

	// VolumeType is the string representation of the volume type.
	VolumeType string `json:"volume_type,omitempty"`

	// VolumeMounted is a bool to indicate if storage volume was mounted.
	VolumeMounted bool `json:"volume_mounted,omitempty"`
}

// KokuMetricsConfigStatus defines the observed state of KokuMetricsConfig.
type KokuMetricsConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ClusterID is a field of KokuMetricsConfig to represent the cluster UUID.
	ClusterID string `json:"clusterID,omitempty"`

	// ClusterVersion is a field of KokuMetricsConfig to represent the cluster version.
	ClusterVersion string `json:"clusterVersion,omitempty"`

	// APIURL is a field of KokuMetricsConfig to represent the url of the API endpoint for service interaction.
	// +optional
	APIURL string `json:"api_url,omitempty"`

	// Authentication is a field of KokuMetricsConfig to represent the authentication status.
	Authentication AuthenticationStatus `json:"authentication,omitempty"`

	// Packaging is a field of KokuMetricsConfig to represent the packaging status
	Packaging PackagingStatus `json:"packaging,omitempty"`

	// Upload is a field of KokuMetricsConfig to represent the upload object.
	Upload UploadStatus `json:"upload,omitempty"`

	// OperatorCommit is a field of KokuMetricsConfig that shows the commit hash of the operator.
	OperatorCommit string `json:"operator_commit,omitempty"`

	// Prometheus represents the status of premetheus queries.
	Prometheus PrometheusStatus `json:"prometheus,omitempty"`

	// Reports represents the status of report generation.
	Reports ReportsStatus `json:"reports,omitempty"`

	// source represents the observed state of the integration on console.redhat.com.
	// +optional
	Source CloudDotRedHatSourceStatus `json:"source,omitempty"`

	// Storage is a field
	Storage StorageStatus `json:"storage,omitempty"`

	// PersistentVolumeClaim is a field of KokuMetricsConfig to represent a PVC.
	PersistentVolumeClaim *EmbeddedPersistentVolumeClaim `json:"persistent_volume_claim,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced

// KokuMetricsConfig is the Schema for the kokumetricsconfig API
type KokuMetricsConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KokuMetricsConfigSpec   `json:"spec"`
	Status KokuMetricsConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KokuMetricsConfigList contains a list of KokuMetricsConfig
type KokuMetricsConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KokuMetricsConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KokuMetricsConfig{}, &KokuMetricsConfigList{})
}

// +kubebuilder:object:generate:=false
type MetricsConfig = KokuMetricsConfig

// +kubebuilder:object:generate:=false
type MetricsConfigSpec = KokuMetricsConfigSpec

// +kubebuilder:object:generate:=false
type MetricsConfigStatus = KokuMetricsConfigStatus
