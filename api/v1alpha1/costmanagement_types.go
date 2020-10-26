/*


Copyright 2020 Red Hat, Inc.

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

package v1alpha1

import (
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

	//UploadOn sets the operator to upload to cloud.redhat.com
	UploadOn bool = true

	//UploadOff sets the operator to not upload to cloud.redhat.com
	UploadOff bool = false

	//UploadCycle sets the default cycle to be 360 minutes (6 hours)
	UploadSchedule int64 = 360

	//SourceCheckSchedule sets the default cycle to be 1440 minutes (24 hours)
	SourceCheckSchedule int64 = 1440
)

// AuthenticationType describes how the upload will be handled.
// Only one of the following authtication types may be specified.
// If none of the following types are specified, the default one
// is Token.
// +kubebuilder:validation:Enum=token;basic
type AuthenticationType string

const (
	// Basic allows upload of data using basic authencation.
	Basic AuthenticationType = "basic"

	// Token allows upload of data using token authencation.
	Token AuthenticationType = "token"
)

// AuthenticationSpec defines the desired state of Authentication object in the CostManagementSpec
type AuthenticationSpec struct {

	// AuthType is a field of CostManagement to represent the authentication type to be used basic or token.
	// Valid values are:
	// - "basic" : Enables authetication using user and password from authentication secret
	// - "token" (default): Uses cluster token for authentication
	// +optional
	AuthType AuthenticationType `json:"type,omitempty"`

	// AuthenticationSecretName is a field of CostManagement to represent the secret with the user and password used for uploads.
	// +optional
	AuthenticationSecretName string `json:"secret_name,omitempty"`
}

// UploadSpec defines the desired state of Authentication object in the CostManagementSpec
type UploadSpec struct {

	// IngressAPIPath is a field of CostManagement to represent the path of the Ingress API service.
	// The default is /api/ingress/v1/upload/
	// +optional
	IngressAPIPath string `json:"ingress_path,omitempty"`

	// UploadWait is a field of CostManagement to represent the time to wait before sending an upload.
	// +optional
	// +kubebuilder:validation:Minimum=0
	UploadWait *int64 `json:"upload_wait,omitempty"`

	// UploadCycle is a field of CostManagement to represent the number of minutes between each upload schedule
	// The default is 360 min (6 hours)
	// +optional
	// +kubebuilder:validation:Minimum=0
	UploadCycle *int64 `json:"upload_cycle,omitempty"`

	// UploadToggle is a field of CostManagement to represent if the operator should upload to cloud.redhat.com.
	// The default is true
	// +optional
	UploadToggle *bool `json:"upload_toggle,omitempty"`
}

// PrometheusSpec defines the desired state of PrometheusConfig object in the CostManagementSpec
type PrometheusSpec struct {

	// SvcAddress is a field of CostManagement to represent the thanos-querier address.
	SvcAddress string `json:"service_address,omitempty"`

	// SkipTLSVerification is a field of CostManagement to represent if the thanos-querier endpoint must be certificate validated.
	// The default is false.
	SkipTLSVerification *bool `json:"skip_tls_verification,omitempty"`
}

// CloudDotRedHatSourceSpec defines the desired state of CloudDotRedHatSource object in the CostManagementSpec
type CloudDotRedHatSourceSpec struct {

	// SourcesAPIPath is a field of CostManagement to represent the path of the Sources API service.
	// The default is /api/ingress/v1.0/sources/
	// +optional
	SourcesAPIPath string `json:"sources_path,omitempty"`

	// SourceName is a field of CostManagementSpec to represent the source name on cloud.redhat.com.
	// +optional
	SourceName string `json:"name,omitempty"`

	// CreateSource is a field of CostManagementSpec to represent if the source should be created if not found.
	// +optional
	CreateSource *bool `json:"create_source,omitempty"`

	// CheckCycle is a field of CostManagement to represent the number of minutes between each source check schedule
	// The default is 1440 min (24 hours)
	// +optional
	// +kubebuilder:validation:Minimum=0
	CheckCycle *int64 `json:"check_cycle,omitempty"`
}

// CostManagementSpec defines the desired state of CostManagement
type CostManagementSpec struct {

	// ClusterID is a field of CostManagement to represent the cluster UUID.
	// +optional
	ClusterID string `json:"clusterID,omitempty"`

	// ValidateCert is a field of CostManagement to represent if the Ingress endpoint must be certificate validated.
	// +optional
	ValidateCert *bool `json:"validate_cert,omitempty"`

	// APIURL is a field of CostManagement to represent the url of the API endpoint for service interaction.
	// +optional
	APIURL string `json:"api_url,omitempty"`

	// Authentication is a field of CostManagement to represent the authentication object.
	// +optional
	Authentication AuthenticationSpec `json:"authentication,omitempty"`

	// FileDirectory is a field of CostManagement to represent the location of stored files.
	// +optional
	FileDirectory string `json:"file_directory,omitempty"`

	// Upload is a field of CostManagement to represent the upload object.
	// +optional
	Upload UploadSpec `json:"upload,omitempty"`

	// PrometheusConfig is a field of CostManagement to represent the configuration of Prometheus connection.
	// +optional
	PrometheusConfig PrometheusSpec `json:"prometheus_config,omitempty"`

	// Source is a field of CostManagement to represent the desired source on cloud.redhat.com.
	// +optional
	Source CloudDotRedHatSourceSpec `json:"source,omitempty"`
}

// AuthenticationStatus defines the desired state of Authentication object in the CostManagementStatus
type AuthenticationStatus struct {

	// AuthType is a field of CostManagement to represent the authentication type to be used basic or token.
	AuthType AuthenticationType `json:"type,omitempty"`

	// AuthenticationSecretName is a field of CostManagement to represent the secret with the user and password used for uploads.
	AuthenticationSecretName string `json:"secret_name,omitempty"`

	// AuthenticationCredentialsFound is a field of CostManagement to represent if used for uploads were found.
	AuthenticationCredentialsFound *bool `json:"credentials_found,omitempty"`
}

// UploadStatus defines the desired state of Upload object in the CostManagementStatus
type UploadStatus struct {

	// IngressAPIPath is a field of CostManagement to represent the path of the Ingress API service.
	// +optional
	IngressAPIPath string `json:"ingress_path,omitempty"`

	// UploadToggle is a field of CostManagement to represent if the operator should upload to cloud.redhat.com.
	// The default is true
	UploadToggle *bool `json:"upload,omitempty"`

	// UploadWait is a field of CostManagement to represent the time to wait before sending an upload.
	UploadWait *int64 `json:"upload_wait,omitempty"`

	// UploadCycle is a field of CostManagement to represent the number of minutes between each upload schedule
	// The default is 360 min (6 hours)
	UploadCycle *int64 `json:"upload_cycle,omitempty"`

	// LastUploadStatus is a field of CostManagement that shows the http status of the last upload
	LastUploadStatus string `json:"last_upload_status,omitempty"`

	// LastUploadTime is a field of CostManagement that shows the time that the last upload was attempted
	// +nullable
	LastUploadTime metav1.Time `json:"last_upload_time,omitempty"`

	// LastSuccessfulUploadTime is a field of CostManagement that shows the time of the last successful upload
	// +nullable
	LastSuccessfulUploadTime metav1.Time `json:"last_successful_upload_time,omitempty"`
}

// CloudDotRedHatSourceStatus defines the observed state of CloudDotRedHatSource object in the CostManagementStatus
type CloudDotRedHatSourceStatus struct {

	// SourcesAPIPath is a field of CostManagement to represent the path of the Sources API service.
	// +optional
	SourcesAPIPath string `json:"sources_path,omitempty"`

	// SourceName is a field of CostManagementStatus to represent the source name on cloud.redhat.com.
	// +optional
	SourceName string `json:"name,omitempty"`

	// SourceDefined is a field of CostManagementStatus to represent if the source exists as defined on cloud.redhat.com.
	// +optional
	SourceDefined *bool `json:"source_defined,omitempty"`

	// SourceError is a field of CostManagementStatus to represent the error encountered creating the source.
	// +optional
	SourceError string `json:"error,omitempty"`

	// LastSourceCheckTime is a field of CostManagement that shows the time that the last check was attempted
	// +nullable
	LastSourceCheckTime metav1.Time `json:"last_check_time,omitempty"`

	// CheckCycle is a field of CostManagement to represent the number of minutes between each source check schedule
	// The default is 1440 min (24 hours)
	CheckCycle *int64 `json:"check_cycle,omitempty"`
}

// PrometheusStatus defines the status for querying prometheus
type PrometheusStatus struct {

	// PrometheusConfigured is a field of CostManagementStatus to represent if cost-management is configured to connect to prometheus.
	PrometheusConfigured bool `json:"prometheus_configured"`

	// ConfigError is a field of CostManagementStatus to represent errors during prometheus configuration.
	ConfigError string `json:"configuration_error,omitempty"`

	// PrometheusConnected is a field of CostManagementStatus to represent if prometheus can be queried.
	PrometheusConnected bool `json:"prometheus_connected"`

	// ConnectionError is a field of CostManagementStatus to represent errors during prometheus test query.
	ConnectionError string `json:"prometheus_connection_error,omitempty"`

	// LastQueryStartTime is a field of CostManagementStatus to represent the last time queries were started.
	// +nullable
	LastQueryStartTime metav1.Time `json:"last_query_start_time,omitempty"`

	// LastQuerySuccessTime is a field of CostManagementStatus to represent the last time queries were successful.
	// +nullable
	LastQuerySuccessTime metav1.Time `json:"last_query_success_time,omitempty"`

	// SvcAddress is the internal thanos-querier address.
	SvcAddress string `json:"service_address,omitempty"`

	// SkipTLSVerification is a field of CostManagementStatus to represent if the thanos-querier endpoint must be certificate validated
	SkipTLSVerification *bool `json:"skip_tls_verification,omitempty"`
}

// ReportsStatus defines the status for generating reports
type ReportsStatus struct {

	// ReportMonth is a field of CostManagementStatus to represent the month for which reports are being generated.
	ReportMonth string `json:"report_month,omit_empty"`

	// LastHourQueried is a field of CostManagementStatus to represent the time range for which metrics were last queried.
	LastHourQueried string `json:"last_hour_queried,omit_empty"`

	// DataCollected is a field of CostManagementStatus to represent whether or not data was collected for the last query.
	DataCollected bool `json:"data_collected,omit_empty"`

	// DataCollectionMessage is a field of CostManagementStatus to represent a message associated with the data_collected status.
	DataCollectionMessage string `json:"data_collection_message,omit_empty"`
}

// CostManagementStatus defines the observed state of CostManagement
type CostManagementStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ClusterID is a field of CostManagement to represent the cluster UUID.
	ClusterID string `json:"clusterID,omitempty"`

	// ValidateCert is a field of CostManagement to represent if the Ingress endpoint must be certificate validated.
	ValidateCert *bool `json:"validate_cert,omitempty"`

	// APIURL is a field of CostManagement to represent the url of the API endpoint for service interaction.
	// +optional
	APIURL string `json:"api_url,omitempty"`

	// Authentication is a field of CostManagement to represent the authentication status.
	Authentication AuthenticationStatus `json:"authentication,omitempty"`

	// FileDirectory is a field of CostManagement to represent the location of stored files.
	FileDirectory string `json:"file_directory,omitempty"`

	// FileDirectoryConfigured is a field of CostManagement to represent if the location of stored files has been configured.
	FileDirectoryConfigured *bool `json:"file_directory_configured,omitempty"`

	// Upload is a field of CostManagement to represent the upload object.
	Upload UploadStatus `json:"upload,omitempty"`

	// OperatorCommit is a field of CostManagement that shows the commit hash of the operator
	OperatorCommit string `json:"operator_commit,omitempty"`

	// Prometheus represents the status of premetheus queries
	Prometheus PrometheusStatus `json:"prometheus,omitempty"`

	// Reports represents the status of report generation
	Reports ReportsStatus `json:"reports,omitempty"`

	// Source is a field of CostManagement to represent the observed state of the source on cloud.redhat.com.
	// +optional
	Source CloudDotRedHatSourceStatus `json:"source,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// CostManagement is the Schema for the costmanagements API
type CostManagement struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CostManagementSpec   `json:"spec,omitempty"`
	Status CostManagementStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CostManagementList contains a list of CostManagement
type CostManagementList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CostManagement `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CostManagement{}, &CostManagementList{})
}
