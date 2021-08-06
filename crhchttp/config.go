//
// Copyright 2021 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

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
