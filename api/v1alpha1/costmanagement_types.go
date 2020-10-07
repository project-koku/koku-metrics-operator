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

// CloudDotRedHatSourceSpec defines the desired state of CloudDotRedHatSource object in the CostManagementSpec
type CloudDotRedHatSourceSpec struct {

	// SourceName is a field of CostManagementSpec to represent the source name on cloud.redhat.com.
	// +optional
	SourceName string `json:"name,omitempty"`

	// CreateSource is a field of CostManagementSpec to represent if the source should be created if not found.
	// +optional
	CreateSource *bool `json:"create_source,omitempty"`
}

// CostManagementSpec defines the desired state of CostManagement
type CostManagementSpec struct {

	// ClusterID is a field of CostManagement to represent the cluster UUID.
	// +optional
	ClusterID string `json:"clusterID,omitempty"`

	// ValidateCert is a field of CostManagement to represent if the Ingress endpoint must be certificate validated.
	// +optional
	ValidateCert *bool `json:"validate_cert,omitempty"`

	// IngressURL is a field of CostManagement to represent the url of the ingress service.
	// +optional
	IngressURL string `json:"ingress_url,omitempty"`

	// Authentication is a field of CostManagement to represent the authentication object.
	// +optional
	Authentication AuthenticationSpec `json:"authentication,omitempty"`

	// +kubebuilder:validation:Minimum=0

	// UploadWait is a field of CostManagement to represent the time to wait before sending an upload.
	// +optional
	UploadWait *int64 `json:"upload_wait,omitempty"`

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

// CloudDotRedHatSourceStatus defines the observed state of CloudDotRedHatSource object in the CostManagementStatus
type CloudDotRedHatSourceStatus struct {

	// SourceName is a field of CostManagementStatus to represent the source name on cloud.redhat.com.
	// +optional
	SourceName string `json:"name,omitempty"`

	// SourceDefined is a field of CostManagementStatus to represent if the source exists as defined on cloud.redhat.com.
	// +optional
	SourceDefined *bool `json:"source_defined,omitempty"`

	// SourceError is a field of CostManagementStatus to represent the error encountered creating the source.
	// +optional
	SourceError string `json:"error,omitempty"`
}

// CostManagementStatus defines the observed state of CostManagement
type CostManagementStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ClusterID is a field of CostManagement to represent the cluster UUID.
	ClusterID string `json:"clusterID,omitempty"`

	// ValidateCert is a field of CostManagement to represent if the Ingress endpoint must be certificate validated.
	ValidateCert *bool `json:"validate_cert,omitempty"`

	// IngressURL is a field of CostManagement to represent the url of the ingress service.
	IngressURL string `json:"ingress_url,omitempty"`

	// Authentication is a field of CostManagement to represent the authentication status.
	Authentication AuthenticationStatus `json:"authentication,omitempty"`

	// UploadWait is a field of CostManagement to represent the time to wait before sending an upload.
	UploadWait *int64 `json:"upload_wait,omitempty"`

	// Last upload status
	LastUploadStatus string `json:"last_upload_status,omitempty"`

	// Last upload time
	LastUploadTime string `json:"last_upload_time,omitempty"`

	// Last successful upload time
	LastSuccessfulUploadTime string `json:"last_successful_upload_time,omitempty"`

	// Operator git commit Hash
	OperatorCommit string `json:"operator_commit,omitempty"`

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
