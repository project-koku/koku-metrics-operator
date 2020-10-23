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

package crhchttp

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	costmgmtv1alpha1 "github.com/project-koku/korekuta-operator-go/api/v1alpha1"
)

// CostManagementConfig provide the data for procesing the reconcile with defaults
type CostManagementConfig struct {
	ClusterID                string
	ValidateCert             bool
	APIURL                   string
	AuthenticationSecretName string
	Authentication           costmgmtv1alpha1.AuthenticationType
	UploadWait               int64
	UploadToggle             bool
	UploadCycle              int64
	IngressAPIPath           string
	BearerTokenString        string
	BasicAuthUser            string
	BasicAuthPassword        string
	LastUploadStatus         string
	LastUploadTime           metav1.Time
	LastSuccessfulUploadTime metav1.Time
	PrometheusSvcAddress     string
	SkipTLSVerification      bool
	PrometheusConnected      bool
	LastQueryStartTime       metav1.Time
	LastQuerySuccessTime     metav1.Time
	LastHourCollected        string
	ReportMonth              string
	OperatorCommit           string
	SourceName               string
	CreateSource             bool
	SourceCheckCycle         int64
	LastSourceCheckTime      metav1.Time
	SourcesAPIPath           string
}
