/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
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
// +kubebuilder:validation:Enum=false;true
type CertValidationType string

const (
	// CertIgnore allows certificate validation to be bypassed.
	CertIgnore CertValidationType = "false"

	// CertCheck allows certificate validation to occur.
	CertCheck CertValidationType = "true"
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

	// ValidateCert is a field of CostManagement to represent if the Ingress endpoint must be certifacte validated.
	// Valid values are:
	// - "true" : Enables validation of the upload endpoint
	// - "false" (default): Ignores validation of the upload endpoint
	// +optional
	ValidateCert CertValidationType `json:"validate_cert,omitempty"`

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
