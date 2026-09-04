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
	"github.com/project-koku/koku-metrics-operator/internal/sources"
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
		Entry("on-prem URL", "https://on-prem-koku.example.com:8443", false),
		Entry("localhost URL", "http://localhost:8088", false),
		Entry("empty string", "", false),
		Entry("similar but wrong URL", "https://console.redhat.com.evil.com", false),
		Entry("default URL with trailing slash", "https://console.redhat.com/", true),
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

var _ = Describe("validateTokenURL", func() {
	DescribeTable("validates token_url based on api_url context",
		func(apiURL, tokenURL string, expectErrSubstring string) {
			err := validateTokenURL(apiURL, tokenURL)
			if expectErrSubstring == "" {
				Expect(err).ToNot(HaveOccurred())
				return
			}
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(expectErrSubstring))
		},
		Entry("RH api + default SSO",
			metricscfgv1beta1.DefaultAPIURL, metricscfgv1beta1.DefaultTokenURL, ""),
		Entry("RH api + default SSO with trailing slash",
			metricscfgv1beta1.DefaultAPIURL, metricscfgv1beta1.DefaultTokenURL+"/", ""),
		Entry("old RH api + default SSO",
			metricscfgv1beta1.OldDefaultAPIURL, metricscfgv1beta1.DefaultTokenURL, ""),
		Entry("RH api + attacker HTTPS token_url",
			metricscfgv1beta1.DefaultAPIURL, "https://evil.example.com/token", "Red Hat Cost Management endpoint"),
		Entry("RH api + lookalike SSO host",
			metricscfgv1beta1.DefaultAPIURL, "https://sso.redhat.com.evil.com/auth/realms/redhat-external/protocol/openid-connect/token", "Red Hat Cost Management endpoint"),
		Entry("RH api + http SSO",
			metricscfgv1beta1.DefaultAPIURL, "http://sso.redhat.com/auth/realms/redhat-external/protocol/openid-connect/token", "https"),
		Entry("custom api + custom HTTPS Keycloak",
			"https://on-prem-koku.example.com:8443", "https://keycloak.corp.internal/auth/realms/cost/protocol/openid-connect/token", ""),
		Entry("custom api + RH SSO token_url still allowed",
			"https://on-prem-koku.example.com:8443", metricscfgv1beta1.DefaultTokenURL, ""),
		Entry("custom api + http Keycloak",
			"https://on-prem-koku.example.com:8443", "http://keycloak.corp.internal/auth/token", "https"),
		Entry("empty token_url host",
			metricscfgv1beta1.DefaultAPIURL, "https://", "https"),
		Entry("malformed token_url",
			"https://on-prem-koku.example.com:8443", "://bad", "https"),
	)

	It("rejects RH api + invalid token_url in validateCredentials before token exchange", func() {
		r := &MetricsConfigReconciler{}
		handler := &sources.SourceHandler{
			Auth: &crhchttp.AuthConfig{
				Authentication: metricscfgv1beta1.ServiceAccount,
				ServiceAccountData: crhchttp.ServiceAccountData{
					ClientID:     "client",
					ClientSecret: "secret",
				},
			},
		}
		cr := &metricscfgv1beta1.MetricsConfig{}
		cr.Spec.Authentication.AuthType = metricscfgv1beta1.ServiceAccount
		cr.Status.APIURL = metricscfgv1beta1.DefaultAPIURL
		cr.Status.Authentication.TokenURL = "https://evil.example.com/token"

		err := r.validateCredentials(context.Background(), handler, cr, 0)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Red Hat Cost Management endpoint"))
		Expect(cr.Status.Authentication.AuthErrorMessage).To(ContainSubstring("Red Hat Cost Management endpoint"))
		Expect(handler.Auth.BearerTokenString).To(BeEmpty())
	})
})
