//
// Copyright 2021 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package v1beta1

const (
	// DefaultAuthenticationType The default authencation type.
	DefaultAuthenticationType AuthenticationType = Token

	// DefaultAPIURL The default ingress path.
	DefaultAPIURL string = "https://console.redhat.com"

	// DefaultIngressPath The default ingress path.
	DefaultIngressPath string = "/api/ingress/v1/upload"

	// DefaultSourcesPath The default ingress path.
	DefaultSourcesPath string = "/api/sources/v1.0/"

	// DefaultPrometheusSvcAddress The default address to thanos-querier.
	DefaultPrometheusSvcAddress string = "https://thanos-querier.openshift-monitoring.svc:9091"

	// DefaultValidateCert The default cert validation setting
	DefaultValidateCert bool = CertIgnore

	// DefaultUploadToggle The default upload toggle
	DefaultUploadToggle bool = UploadOn

	// DefaultUploadCycle The default upload cycle
	DefaultUploadCycle int64 = UploadSchedule

	// DefaultSourceCheckCycle The default source check cycle
	DefaultSourceCheckCycle int64 = SourceCheckSchedule

	// DefaultMaxSize The default max size for report files
	DefaultMaxSize int64 = PackagingMaxSize

	// DefaultPrometheusContextTimeout The default context timeout for Prometheus Queries
	DefaultPrometheusContextTimeout int64 = 120

	// OldDefaultAPIURL The old default ingress path.
	OldDefaultAPIURL string = "https://cloud.redhat.com"

	// DefaultTokenURL The default path to obtain a service account access token
	DefaultTokenURL string = "https://sso.redhat.com/auth/realms/redhat-external/protocol/openid-connect/token"
)
