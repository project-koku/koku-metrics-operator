//
// Copyright 2021 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package crhchttp

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	metricscfgv1beta1 "github.com/project-koku/koku-metrics-operator/api/v1beta1"
)

// AuthConfig provides the data for reconciling the CR with defaults
type AuthConfig struct {
	Client            client.Client
	ClusterID         string
	Authentication    metricscfgv1beta1.AuthenticationType
	BearerTokenString string
	BasicAuthUser     string
	BasicAuthPassword string
	ValidateCert      bool
	OperatorCommit    string
}
