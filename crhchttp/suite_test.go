//
// Copyright 2023 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package crhchttp

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/project-koku/koku-metrics-operator/testutils"
)

var (
	testEnv *envtest.Environment

	sethttpgetmethod bool
	validMockTS      *httptest.Server
	badMockTS        *httptest.Server
	mockaccesstoken  = "mockAccessToken12345"
	tokenurlsuffix   = "/protocol/openid-connect/token"
)

func TestCrhchttp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Crhchttp Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(testutils.ZapLogger(true))

	validMockTS = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		mockTokenResponse := map[string]interface{}{
			"access_token":       mockaccesstoken,
			"expires_in":         3600,
			"refresh_expires_in": 1800,
			"token_type":         "Bearer",
			"not_before_policy":  0,
			"scope":              "user",
		}
		err := json.NewEncoder(w).Encode(mockTokenResponse)
		Expect(err).NotTo(HaveOccurred())
		w.WriteHeader(http.StatusOK)
	}))

	badMockTS = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, tokenurlsuffix) {
			http.NotFound(w, r)
		}

		if r.Method != http.MethodPost || sethttpgetmethod {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}

		// Read the body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read request body", http.StatusInternalServerError)
		}

		// Decode POST form data
		postData, err := url.ParseQuery(string(body))
		if err != nil {
			http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		}

		// Validate the fields
		if postData.Get("client_id") == "invalid client id" {
			invalidClientResponse := map[string]string{
				"error":             "invalid_client",
				"error_description": "Invalid client credentials",
			}
			w.WriteHeader(http.StatusBadRequest)
			err := json.NewEncoder(w).Encode(invalidClientResponse)
			Expect(err).NotTo(HaveOccurred())
		}
		if postData.Get("client_secret") == "invalid client secret" {
			invalidClientResponse := map[string]string{
				"error":             "unauthorized_client",
				"error_description": "Invalid client secret",
			}
			w.WriteHeader(http.StatusBadRequest)
			err := json.NewEncoder(w).Encode(invalidClientResponse)
			Expect(err).NotTo(HaveOccurred())
		}

		if postData.Get("grant_type") == "" {
			w.WriteHeader(http.StatusBadRequest)
		}
	}))

	By("bootstrapping test environment")
	t := true
	if os.Getenv("TEST_USE_EXISTING_CLUSTER") == "true" {
		testEnv = &envtest.Environment{
			UseExistingCluster: &t,
		}
	} else {
		testEnv = &envtest.Environment{
			CRDDirectoryPaths: []string{
				filepath.Join("..", "config", "crd", "bases"),
				filepath.Join("test_files", "crds"),
			},
		}
	}
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
	validMockTS.Close()
	badMockTS.Close()
})
