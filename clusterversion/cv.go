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

package clusterversion

import (
	"context"

	configv1 "github.com/openshift/api/config/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ClusterVersion interface {
	GetClusterVersion() (*configv1.ClusterVersion, error)
}

type ClusterVersionBuilder interface {
	New(client.Client) ClusterVersion
}

type clusterVersionClient struct {
	client client.Client
}

type clusterVersionClientBuilder struct{}

func NewCVClient(c client.Client) ClusterVersion {
	return &clusterVersionClient{c}
}

func NewBuilder() ClusterVersionBuilder {
	return &clusterVersionClientBuilder{}
}

func (cvb *clusterVersionClientBuilder) New(c client.Client) ClusterVersion {
	return NewCVClient(c)
}

// GetClusterVersion gets the ClusterVersion CR
func (c *clusterVersionClient) GetClusterVersion() (*configv1.ClusterVersion, error) {
	cvList := &configv1.ClusterVersionList{}
	err := c.client.List(context.TODO(), cvList)
	if err != nil {
		return nil, err
	}

	// ClusterVersion is a singleton
	for _, cv := range cvList.Items {
		return &cv, nil
	}

	return nil, errors.NewNotFound(schema.GroupResource{Group: configv1.GroupName, Resource: "ClusterVersion"}, "ClusterVersion")
}
