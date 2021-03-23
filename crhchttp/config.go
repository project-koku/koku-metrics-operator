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
	"github.com/go-logr/logr"
	costmanagementmetricscfgv1beta1 "github.com/project-costmanagement/costmanagement-metrics-operator/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// AuthConfig provides the data for reconciling the CR with defaults
type AuthConfig struct {
	Client            client.Client
	ClusterID         string
	Authentication    costmanagementmetricscfgv1beta1.AuthenticationType
	BearerTokenString string
	BasicAuthUser     string
	BasicAuthPassword string
	ValidateCert      bool
	OperatorCommit    string
	Log               logr.Logger
}
