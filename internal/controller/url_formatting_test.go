//
// Copyright 2026 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"

	metricscfgv1beta1 "github.com/project-koku/koku-metrics-operator/api/v1beta1"
	"github.com/project-koku/koku-metrics-operator/internal/crhchttp"
)

var _ = Describe("formatURLForDisplay", func() {
	DescribeTable("should format URLs correctly",
		func(input, expected string) {
			result := formatURLForDisplay(input)
			Expect(result).To(Equal(expected))
		},
		Entry("HTTPS URL without trailing slash", "https://console.redhat.com", "console.redhat.com"),
		Entry("HTTPS URL with trailing slash", "https://console.redhat.com/", "console.redhat.com"),
		Entry("HTTPS URL with port", "https://on-prem.example.com:8443", "on-prem.example.com:8443"),
		Entry("HTTPS URL with port and trailing slash", "https://on-prem.example.com:8443/", "on-prem.example.com:8443"),
		Entry("HTTP URL with port", "http://localhost:8088", "localhost:8088"),
		Entry("HTTP URL with port and trailing slash", "http://localhost:8088/", "localhost:8088"),
		Entry("HTTPS URL with internal domain", "https://koku.internal.corp", "koku.internal.corp"),
		Entry("URL without scheme", "console.redhat.com", "console.redhat.com"),
		// Fallback path - malformed URLs that url.Parse cannot handle
		// Note: url.Parse is quite permissive and succeeds on most strings,
		// so the fallback err != nil branch is essentially dead code in practice.
		// These test cases verify the fallback string manipulation works correctly.
		Entry("Invalid URL with colon in path - triggers fallback", "not a url ://", "not a url :/"),
		Entry("Invalid URL with spaces - triggers fallback", "https://host with spaces", "host with spaces"),
	)
})

var _ = Describe("isAllowedTokenAuthURL", func() {
	DescribeTable("should identify allowed Red Hat endpoints",
		func(url string, expected bool) {
			Expect(isAllowedApiUrl(url)).To(Equal(expected))
		},
		Entry("default API URL", metricscfgv1beta1.DefaultAPIURL, true),
		Entry("old default API URL", metricscfgv1beta1.OldDefaultAPIURL, true),
		Entry("stage API URL", metricscfgv1beta1.StageAPIURL, true),
		Entry("on-prem URL", "https://on-prem-koku.example.com:8443", false),
		Entry("localhost URL", "http://localhost:8088", false),
		Entry("empty string", "", false),
		Entry("similar but wrong URL", "https://console.redhat.com.evil.com", false),
		Entry("default URL with trailing slash", "https://console.redhat.com/", true),
		Entry("stage URL with trailing slash", "https://console.stage.redhat.com/", true),
		Entry("http instead of https", "http://console.redhat.com", false),
	)
})

var _ = Describe("setAuthentication token URL allowlist", func() {
	It("rejects non-approved URL and clears any stale bearer token without reading pull-secret", func() {
		r := &MetricsConfigReconciler{}
		authCfg := &crhchttp.AuthConfig{
			Authentication:    metricscfgv1beta1.Token,
			BearerTokenString: "stale-pull-secret-token",
		}
		cr := &metricscfgv1beta1.MetricsConfig{}
		cr.Status.APIURL = "https://evil.example.com"
		cr.Status.Authentication.AuthType = metricscfgv1beta1.Token

		err := r.setAuthentication(context.Background(), authCfg, cr, types.NamespacedName{})

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("token authentication is only permitted"))
		Expect(authCfg.BearerTokenString).To(BeEmpty())
		Expect(cr.Status.Authentication.AuthenticationCredentialsFound).ToNot(BeNil())
		Expect(*cr.Status.Authentication.AuthenticationCredentialsFound).To(BeFalse())
		Expect(cr.Status.Authentication.AuthErrorMessage).To(ContainSubstring("service-account"))
	})
})
