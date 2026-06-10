//
// Copyright 2026 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package controller

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
