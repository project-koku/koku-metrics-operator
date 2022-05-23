//
// Copyright 2022 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package v1beta1

// MetricsConfig inherits from KokuMetricsConfig to carry the configuration throughout the code.
type MetricsConfig struct {
	*KokuMetricsConfig
}

type ACompletelyRandomType struct {
	Thisisastring string
}
