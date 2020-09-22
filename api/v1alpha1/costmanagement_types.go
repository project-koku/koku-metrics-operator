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

// CostManagementSpec defines the desired state of CostManagement
type CostManagementSpec struct {

	// ClusterID is a field of CostManagement to represent the cluster UUID.
	// +optional
	ClusterID string `json:"clusterID,omitempty"`

	// ValidateCert is a field of CostManagement to represent if the Ingress endpoint must be certificate validated.
	// +optional
	ValidateCert *bool `json:"validate_cert,omitempty"`

	// IngressUrl is a field of CostManagement to represent the url of the ingress service.
	// +optional
	IngressUrl string `json:"ingress_url,omitempty"`

	// Authentication is a field of CostManagement to represent the authentication type to be used basic or token.
	// Valid values are:
	// - "basic" : Enables authetication using user and password from authentication secret
	// - "token" (default): Uses cluster token for authentication
	// +optional
	Authentication AuthenticationType `json:"authentication,omitempty"`

	// AuthenticationSecretName is a field of CostManagement to represent the secret with the user and password used for uploads.
	// +optional
	AuthenticationSecretName string `json:"authentication_secret_name,omitempty"`

	// +kubebuilder:validation:Minimum=0

	// UploadWait is a field of CostManagement to represent the time to wait before sending an upload.
	// +optional
	UploadWait *int64 `json:"upload_wait,omitempty"`
}

// CostManagementStatus defines the observed state of CostManagement
type CostManagementStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ClusterID is a field of CostManagement to represent the cluster UUID.
	ClusterID string `json:"clusterID,omitempty"`

	// ValidateCert is a field of CostManagement to represent if the Ingress endpoint must be certificate validated.
	ValidateCert *bool `json:"validate_cert,omitempty"`

	// IngressUrl is a field of CostManagement to represent the url of the ingress service.
	IngressUrl string `json:"ingress_url,omitempty"`

	// Authentication is a field of CostManagement to represent the authentication type to be used basic or token.
	Authentication AuthenticationType `json:"authentication,omitempty"`

	// AuthenticationSecretName is a field of CostManagement to represent the secret with the user and password used for uploads.
	AuthenticationSecretName string `json:"authentication_secret_name,omitempty"`

	// UploadWait is a field of CostManagement to represent the time to wait before sending an upload.
	UploadWait *int64 `json:"upload_wait,omitempty"`

	// AuthenticationCredentialsFound is a field of CostManagement to represent if used for uploads were found.
	AuthenticationCredentialsFound *bool `json:"authentication_creds_found,omitempty"`
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
