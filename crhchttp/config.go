//
// Copyright 2021 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package crhchttp

import (
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kokumetricscfgv1beta1 "github.com/project-koku/koku-metrics-operator/api/v1beta1"
)

// AuthConfig provides the data for reconciling the CR with defaults
type AuthConfig struct {
	Client            client.Client
	ClusterID         string
	Authentication    kokumetricscfgv1beta1.AuthenticationType
	BearerTokenString string
	BasicAuthUser     string
	BasicAuthPassword string
	ValidateCert      bool
	OperatorCommit    string
	Log               logr.Logger
}
