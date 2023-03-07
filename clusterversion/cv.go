//
// Copyright 2021 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package clusterversion

import (
	"context"

	configv1 "github.com/openshift/api/config/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ClusterVersion interface {
	GetClusterVersion(context.Context) (*configv1.ClusterVersion, error)
}

type clusterVersionClient struct{ client.Client }

func NewCVClient(c client.Client) ClusterVersion {
	return &clusterVersionClient{c}
}

// GetClusterVersion gets the ClusterVersion CR
func (c *clusterVersionClient) GetClusterVersion(ctx context.Context) (*configv1.ClusterVersion, error) {
	cvList := &configv1.ClusterVersionList{}
	err := c.List(ctx, cvList)
	if err != nil {
		return nil, err
	}

	// ClusterVersion is a singleton
	for _, cv := range cvList.Items {
		return &cv, nil
	}

	return nil, errors.NewNotFound(schema.GroupResource{Group: configv1.GroupName, Resource: "ClusterVersion"}, "ClusterVersion")
}
