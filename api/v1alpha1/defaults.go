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

	// DefaultIngressURL The default ingress url.
	DefaultIngressURL string = "https://cloud.redhat.com/api/ingress/v1/upload"

	// DefaultValidateCert The default cert validation setting
	DefaultValidateCert bool = CertIgnore
)
