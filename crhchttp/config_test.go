//
// Copyright 2023 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package crhchttp

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GetAccessToken Functional Tests", func() {
	var authConfig *AuthConfig

	BeforeEach(func() {
		authConfig = &AuthConfig{
			Authentication: serviceaccount,
			ServiceAccountData: ServiceAccountData{
				ClientID:     "testClientId",
				ClientSecret: "testClientSecret",
				GrantType:    "testGrantType",
			},
		}
	})

	// Helper function to create an AuthConfig and call GetAccessToken.
	getTokenWithConfig := func(clientID, clientSecret, grantType string) error {
		config := &AuthConfig{
			Authentication: serviceaccount,
			ServiceAccountData: ServiceAccountData{
				ClientID:     clientID,
				ClientSecret: clientSecret,
				GrantType:    grantType,
			},
		}
		return config.GetAccessToken(badMockTS.URL + tokenurlsuffix)
	}

	It("Successfully retrieves and sets the access token", func() {
		err := authConfig.GetAccessToken(validMockTS.URL + tokenurlsuffix)
		Expect(err).NotTo(HaveOccurred())
		Expect(authConfig.BearerTokenString).To(Equal(mockaccesstoken))
	})

	Context("Negative Tests", func() {
		type TestCase struct {
			ClientID     string
			ClientSecret string
			GrantType    string
			ShouldError  bool
		}

		var testCases = []TestCase{
			{"testClientId", "testSecret", "", true},
			{"testClientId", "invalid client secret", "", true},
			{"invalid client id", "testSecret", "testGrant", true},
			{"testClientId", "invalid client secret", "testGrant", true},
		}

		for _, tc := range testCases {
			It("Should return error for invalid credentials", func() {
				err := getTokenWithConfig(tc.ClientID, tc.ClientSecret, tc.GrantType)
				if tc.ShouldError {
					Expect(err).To(HaveOccurred())
					Expect(authConfig.BearerTokenString).To(BeEmpty())
				} else {
					Expect(err).NotTo(HaveOccurred())
				}
			})
		}
	})

	It("should validate GetAccessToken method behavior", func() {
		err := authConfig.GetAccessToken(validMockTS.URL + tokenurlsuffix)
		Expect(err).NotTo(HaveOccurred())
		Expect(authConfig.BearerTokenString).To(Equal(mockaccesstoken))
	})

	It("should handle HTTP errors from token server", func() {
		sethttpgetmethod = true
		err := authConfig.GetAccessToken(badMockTS.URL + tokenurlsuffix)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Method Not Allowed"))
	})
})
