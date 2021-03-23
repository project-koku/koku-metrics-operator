/*


Copyright 2021 Red Hat, Inc.

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CertValidationType describes how the certificate validation will be handled.
// Only one of the following certificate validation types may be specified.
// If none of the following types are specified, the default one
// is Token.
const (
	// CertIgnore allows certificate validation to be bypassed.
	CertIgnore bool = false

	// CertCheck allows certificate validation to occur.
	CertCheck bool = true

	//UploadOn sets the operator to upload to cloud.redhat.com.
	UploadOn bool = true

	//UploadOff sets the operator to not upload to cloud.redhat.com.
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

// AuthenticationSpec defines the desired state of Authentication object in the CostManagementMetricsConfigSpec.
type AuthenticationSpec struct {

	// AuthType is a field of CostManagementMetricsConfig to represent the authentication type to be used basic or token.
	// Valid values are:
	// - "basic" : Enables authentication using user and password from authentication secret.
	// - "token" (default): Uses cluster token for authentication.
	// +kubebuilder:default="token"
	AuthType AuthenticationType `json:"type"`

	// AuthenticationSecretName is a field of CostManagementMetricsConfig to represent the secret with the user and password used for uploads.
	// +optional
	AuthenticationSecretName string `json:"secret_name,omitempty"`
}

// PackagingSpec defines the desired state of the Packaging object in the CostManagementMetricsConfigSpec.
type PackagingSpec struct {

	// MaxSize is a field of CostManagementMetricsConfig to represent the max file size in megabytes that will be compressed for upload to Ingress.
	// The default is 100.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:default=100
	MaxSize int64 `json:"max_size_MB"`

	// MaxReports is a field of CostManagementMetricsConfig to represent the maximum number of reports to store.
	// The default is 30 reports which corresponds to approximately 7 days worth of data given the other default values.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=30
	MaxReports int64 `json:"max_reports_to_store"`
}

// UploadSpec defines the desired state of Authentication object in the CostManagementMetricsConfigSpec.
type UploadSpec struct {

	// FOR DEVELOPMENT ONLY.
	// IngressAPIPath is a field of CostManagementMetricsConfig to represent the path of the Ingress API service.
	// The default is `/api/ingress/v1/upload`.
	// +kubebuilder:default=`/api/ingress/v1/upload`
	IngressAPIPath string `json:"ingress_path"`

	// UploadWait is a field of CostManagementMetricsConfig to represent the time to wait before sending an upload.
	// +optional
	// +kubebuilder:validation:Minimum=0
	UploadWait *int64 `json:"upload_wait,omitempty"`

	// UploadCycle is a field of CostManagementMetricsConfig to represent the number of minutes between each upload schedule.
	// The default is 360 min (6 hours).
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=360
	UploadCycle *int64 `json:"upload_cycle"`

	// UploadToggle is a field of CostManagementMetricsConfig to represent if the operator is installed in a restricted-network.
	// If `false`, the operator will not upload to cloud.redhat.com or check/create sources.
	// The default is true.
	// +kubebuilder:default=true
	UploadToggle *bool `json:"upload_toggle"`

	// ValidateCert is a field of CostManagementMetricsConfig to represent if the Ingress endpoint must be certificate validated.
	// +kubebuilder:default=true
	ValidateCert *bool `json:"validate_cert"`
}

// PrometheusSpec defines the desired state of PrometheusConfig object in the CostManagementMetricsConfigSpec.
type PrometheusSpec struct {

	// FOR DEVELOPMENT ONLY.
	// SvcAddress is a field of CostManagementMetricsConfig to represent the thanos-querier address.
	// The default is `https://thanos-querier.openshift-monitoring.svc:9091`.
	// +kubebuilder:default=`https://thanos-querier.openshift-monitoring.svc:9091`
	SvcAddress string `json:"service_address"`

	// FOR DEVELOPMENT ONLY.
	// SkipTLSVerification is a field of CostManagementMetricsConfig to represent if the thanos-querier endpoint must be certificate validated.
	// The default is false.
	// +kubebuilder:default=false
	SkipTLSVerification *bool `json:"skip_tls_verification"`
}

// CloudDotRedHatSourceSpec defines the desired state of CloudDotRedHatSource object in the CostManagementMetricsConfigSpec.
type CloudDotRedHatSourceSpec struct {

	// FOR DEVELOPMENT ONLY.
	// SourcesAPIPath is a field of CostManagementMetricsConfig to represent the path of the Sources API service.
	// The default is `/api/sources/v1.0/`.
	// +kubebuilder:default=`/api/sources/v1.0/`
	SourcesAPIPath string `json:"sources_path"`

	// SourceName is a field of CostManagementMetricsConfigSpec to represent the source name on cloud.redhat.com.
	// +optional
	SourceName string `json:"name,omitempty"`

	// CreateSource is a field of CostManagementMetricsConfigSpec to represent if the source should be created if not found.
	// +kubebuilder:default=false
	CreateSource *bool `json:"create_source"`

	// CheckCycle is a field of CostManagementMetricsConfig to represent the number of minutes between each source check schedule
	// The default is 1440 min (24 hours).
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=1440
	CheckCycle *int64 `json:"check_cycle"`
}

// CostManagementMetricsConfigSpec defines the desired state of CostManagementMetricsConfig.
type CostManagementMetricsConfigSpec struct {
	// +kubebuilder:validation:preserveUnknownFields=false

	// ClusterID is a field of CostManagementMetricsConfig to represent the cluster UUID. Normally this value should not be
	// specified. Only set this value if the clusterID cannot be obtained from the ClusterVersion.
	// +optional
	ClusterID string `json:"clusterID,omitempty"`

	// FOR DEVELOPMENT ONLY.
	// APIURL is a field of CostManagementMetricsConfig to represent the url of the API endpoint for service interaction.
	// The default is `https://cloud.redhat.com`.
	// +kubebuilder:default=`https://cloud.redhat.com`
	APIURL string `json:"api_url,omitempty"`

	// Authentication is a field of CostManagementMetricsConfig to represent the authentication object.
	Authentication AuthenticationSpec `json:"authentication"`

	//Packaging is a field of CostManagementMetricsConfig to represent the packaging object.
	Packaging PackagingSpec `json:"packaging"`

	// Upload is a field of CostManagementMetricsConfig to represent the upload object.
	Upload UploadSpec `json:"upload"`

	// PrometheusConfig is a field of CostManagementMetricsConfig to represent the configuration of Prometheus connection.
	PrometheusConfig PrometheusSpec `json:"prometheus_config"`

	// Source is a field of CostManagementMetricsConfig to represent the desired source on cloud.redhat.com.
	Source CloudDotRedHatSourceSpec `json:"source"`

	// VolumeClaimTemplate is a field of CostManagementMetricsConfig to represent a PVC template.
	VolumeClaimTemplate *EmbeddedPersistentVolumeClaim `json:"volume_claim_template,omitempty"`
}

// AuthenticationStatus defines the desired state of Authentication object in the CostManagementMetricsConfigStatus.
type AuthenticationStatus struct {

	// AuthType is a field of CostManagementMetricsConfig to represent the authentication type to be used basic or token.
	AuthType AuthenticationType `json:"type,omitempty"`

	// AuthenticationSecretName is a field of CostManagementMetricsConfig to represent the secret with the user and password used for uploads.
	AuthenticationSecretName string `json:"secret_name,omitempty"`

	// AuthenticationCredentialsFound is a field of CostManagementMetricsConfig to represent if used for uploads were found.
	AuthenticationCredentialsFound *bool `json:"credentials_found,omitempty"`

	// ValidBasicAuth is a field of CostManagementMetricsConfig to represent if the given basic auth credentials are valid.
	ValidBasicAuth *bool `json:"valid_basic_auth,omitempty"`

	// AuthErrorMessage is a field of CostManagementMetricsConfig to represent an `invalid credentials` error message.
	AuthErrorMessage string `json:"error,omitempty"`

	// LastVerificationTime is a field of CostManagementMetricsConfig to represent the last time credentials were verified.
	// +nullable
	LastVerificationTime *metav1.Time `json:"last_credential_verification_time,omitempty"`
}

// PackagingStatus defines the observed state of the Packing object in the CostManagementMetricsConfigStatus.
type PackagingStatus struct {

	// LastSuccessfulPackagingTime is a field of CostManagementMetricsConfig that shows the time of the last successful file packaging.
	// +nullable
	LastSuccessfulPackagingTime metav1.Time `json:"last_successful_packaging_time,omitempty"`

	// MaxReports is a field of CostManagementMetricsConfig to represent the maximum number of reports to store.
	MaxReports *int64 `json:"max_reports_to_store,omitempty"`

	// MaxSize is a field of CostManagementMetricsConfig to represent the max file size in megabytes that will be compressed for upload to Ingress.
	MaxSize *int64 `json:"max_size_MB,omitempty"`

	// PackagedFiles is a field of CostManagementMetricsConfig to represent the list of file packages in storage.
	PackagedFiles []string `json:"packaged_files,omitempty"`

	// PackagingError is a field of CostManagementMetricsConfig to represent the error encountered packaging the reports.
	PackagingError string `json:"error,omitempty"`

	// ReportCount is a field of CostManagementMetricsConfig to represent the number of reports in storage.
	ReportCount *int64 `json:"number_reports_stored,omitempty"`
}

// UploadStatus defines the observed state of Upload object in the CostManagementMetricsConfigStatus.
type UploadStatus struct {

	// IngressAPIPath is a field of CostManagementMetricsConfig to represent the path of the Ingress API service.
	// +optional
	IngressAPIPath string `json:"ingress_path,omitempty"`

	// UploadToggle is a field of CostManagementMetricsConfig to represent if the operator should upload to cloud.redhat.com.
	// The default is true
	UploadToggle *bool `json:"upload,omitempty"`

	// UploadWait is a field of CostManagementMetricsConfig to represent the time to wait before sending an upload.
	UploadWait *int64 `json:"upload_wait,omitempty"`

	// UploadCycle is a field of CostManagementMetricsConfig to represent the number of minutes between each upload schedule.
	// The default is 360 min (6 hours).
	UploadCycle *int64 `json:"upload_cycle,omitempty"`

	// UploadError is a field of CostManagementMetricsConfigStatus to represent the error encountered uploading reports.
	// +optional
	UploadError string `json:"error,omitempty"`

	// LastUploadStatus is a field of CostManagementMetricsConfig that shows the http status of the last upload.
	LastUploadStatus string `json:"last_upload_status,omitempty"`

	// LastSuccessfulUploadTime is a field of CostManagementMetricsConfig that shows the time of the last successful upload.
	// +nullable
	LastSuccessfulUploadTime metav1.Time `json:"last_successful_upload_time,omitempty"`

	// ValidateCert is a field of CostManagementMetricsConfig to represent if the Ingress endpoint must be certificate validated.
	ValidateCert *bool `json:"validate_cert,omitempty"`
}

// CloudDotRedHatSourceStatus defines the observed state of CloudDotRedHatSource object in the CostManagementMetricsConfigStatus.
type CloudDotRedHatSourceStatus struct {

	// SourcesAPIPath is a field of CostManagementMetricsConfig to represent the path of the Sources API service.
	// +optional
	SourcesAPIPath string `json:"sources_path,omitempty"`

	// SourceName is a field of CostManagementMetricsConfigStatus to represent the source name on cloud.redhat.com.
	// +optional
	SourceName string `json:"name,omitempty"`

	// SourceDefined is a field of CostManagementMetricsConfigStatus to represent if the source exists as defined on cloud.redhat.com.
	// +optional
	SourceDefined *bool `json:"source_defined,omitempty"`

	// CreateSource is a field of CostManagementMetricsConfigStatus to represent if the source should be created if not found.
	// A source will not be created if upload_toggle is `false`.
	// +optional
	CreateSource *bool `json:"create_source,omitempty"`

	// SourceError is a field of CostManagementMetricsConfigStatus to represent the error encountered creating the source.
	// +optional
	SourceError string `json:"error,omitempty"`

	// LastSourceCheckTime is a field of CostManagementMetricsConfig that shows the time that the last check was attempted.
	// +nullable
	LastSourceCheckTime metav1.Time `json:"last_check_time,omitempty"`

	// CheckCycle is a field of CostManagementMetricsConfig to represent the number of minutes between each source check schedule.
	// The default is 1440 min (24 hours).
	CheckCycle *int64 `json:"check_cycle,omitempty"`
}

// PrometheusStatus defines the status for querying prometheus.
type PrometheusStatus struct {

	// PrometheusConfigured is a field of CostManagementMetricsConfigStatus to represent if the operator is configured to connect to prometheus.
	PrometheusConfigured bool `json:"prometheus_configured"`

	// ConfigError is a field of CostManagementMetricsConfigStatus to represent errors during prometheus configuration.
	ConfigError string `json:"configuration_error,omitempty"`

	// PrometheusConnected is a field of CostManagementMetricsConfigStatus to represent if prometheus can be queried.
	PrometheusConnected bool `json:"prometheus_connected"`

	// ConnectionError is a field of CostManagementMetricsConfigStatus to represent errors during prometheus test query.
	ConnectionError string `json:"prometheus_connection_error,omitempty"`

	// LastQueryStartTime is a field of CostManagementMetricsConfigStatus to represent the last time queries were started.
	// +nullable
	LastQueryStartTime metav1.Time `json:"last_query_start_time,omitempty"`

	// LastQuerySuccessTime is a field of CostManagementMetricsConfigStatus to represent the last time queries were successful.
	// +nullable
	LastQuerySuccessTime metav1.Time `json:"last_query_success_time,omitempty"`

	// SvcAddress is the internal thanos-querier address.
	SvcAddress string `json:"service_address,omitempty"`

	// SkipTLSVerification is a field of CostManagementMetricsConfigStatus to represent if the thanos-querier endpoint must be certificate validated.
	SkipTLSVerification *bool `json:"skip_tls_verification,omitempty"`
}

// ReportsStatus defines the status for generating reports.
type ReportsStatus struct {

	// ReportMonth is a field of CostManagementMetricsConfigStatus to represent the month for which reports are being generated.
	ReportMonth string `json:"report_month,omitempty"`

	// LastHourQueried is a field of CostManagementMetricsConfigStatus to represent the time range for which metrics were last queried.
	LastHourQueried string `json:"last_hour_queried,omitempty"`

	// DataCollected is a field of CostManagementMetricsConfigStatus to represent whether or not data was collected for the last query.
	DataCollected bool `json:"data_collected,omitempty"`

	// DataCollectionMessage is a field of CostManagementMetricsConfigStatus to represent a message associated with the data_collected status.
	DataCollectionMessage string `json:"data_collection_message,omitempty"`
}

// StorageStatus defines the status for storage.
type StorageStatus struct {

	// VolumeType is the string representation of the volume type.
	VolumeType string `json:"volume_type,omitempty"`

	// VolumeMounted is a bool to indicate if storage volume was mounted.
	VolumeMounted bool `json:"volume_mounted,omitempty"`
}

// CostManagementMetricsConfigStatus defines the observed state of CostManagementMetricsConfig.
type CostManagementMetricsConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ClusterID is a field of CostManagementMetricsConfig to represent the cluster UUID.
	ClusterID string `json:"clusterID,omitempty"`

	// APIURL is a field of CostManagementMetricsConfig to represent the url of the API endpoint for service interaction.
	// +optional
	APIURL string `json:"api_url,omitempty"`

	// Authentication is a field of CostManagementMetricsConfig to represent the authentication status.
	Authentication AuthenticationStatus `json:"authentication,omitempty"`

	// Packaging is a field of CostManagementMetricsConfig to represent the packaging status
	Packaging PackagingStatus `json:"packaging,omitempty"`

	// Upload is a field of CostManagementMetricsConfig to represent the upload object.
	Upload UploadStatus `json:"upload,omitempty"`

	// OperatorCommit is a field of CostManagementMetricsConfig that shows the commit hash of the operator.
	OperatorCommit string `json:"operator_commit,omitempty"`

	// Prometheus represents the status of premetheus queries.
	Prometheus PrometheusStatus `json:"prometheus,omitempty"`

	// Reports represents the status of report generation.
	Reports ReportsStatus `json:"reports,omitempty"`

	// Source is a field of CostManagementMetricsConfig to represent the observed state of the source on cloud.redhat.com.
	// +optional
	Source CloudDotRedHatSourceStatus `json:"source,omitempty"`

	// Storage is a field
	Storage StorageStatus `json:"storage,omitempty"`

	// PersistentVolumeClaim is a field of CostManagementMetricsConfig to represent a PVC.
	PersistentVolumeClaim *EmbeddedPersistentVolumeClaim `json:"persistent_volume_claim,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced

// CostManagementMetricsConfig is the Schema for the costmanagementmetricsconfig API
type CostManagementMetricsConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CostManagementMetricsConfigSpec   `json:"spec"`
	Status CostManagementMetricsConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CostManagementMetricsConfigList contains a list of CostManagementMetricsConfig
type CostManagementMetricsConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CostManagementMetricsConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CostManagementMetricsConfig{}, &CostManagementMetricsConfigList{})
}
