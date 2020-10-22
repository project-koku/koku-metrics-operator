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

const (
	// DefaultAuthenticationType The default authencation type.
	DefaultAuthenticationType AuthenticationType = Token

	// DefaultAPIURL The default ingress path.
	DefaultAPIURL string = "https://cloud.redhat.com"

	// DefaultIngressPath The default ingress path.
	DefaultIngressPath string = "/api/ingress/v1/upload"

	// DefaultSourcesPath The default ingress path.
	DefaultSourcesPath string = "/api/sources/v1.0/"

	// DefaultPrometheusSvcAddress The default address to thanos-querier.
	DefaultPrometheusSvcAddress string = "https://thanos-querier.openshift-monitoring.svc:9091"

	// DefaultValidateCert The default cert validation setting
	DefaultValidateCert bool = CertIgnore

	//DefaultUploadToggle The default upload toggle
	DefaultUploadToggle bool = UploadOn

	//DefaultUploadCycle The default upload cycle
	DefaultUploadCycle int64 = UploadSchedule

	//DefaultSourceCheckCycle The default source check cycle
	DefaultSourceCheckCycle int64 = SourceCheckSchedule
)
