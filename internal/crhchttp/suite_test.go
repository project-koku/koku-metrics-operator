//
// Copyright 2023 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package crhchttp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/project-koku/koku-metrics-operator/internal/testutils"
)

var (
	testEnv *envtest.Environment

	validMockTS     *httptest.Server
	badMockTS       *httptest.Server
	mockaccesstoken = "mockAccessToken12345"
	tokenurlsuffix  = "/protocol/openid-connect/token"
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
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.Contains(r.URL.Path, "/bad-response"):
			w.Header().Set("Content-Length", "1")
			w.WriteHeader(http.StatusOK)
			w.(http.Flusher).Flush()
			return

		case strings.Contains(r.URL.Path, "/client-error"):
			clientErrorResponse := map[string]interface{}{
				"error":             "invalid_request",
				"error_description": "This request is missing a required parameter",
			}

			w.WriteHeader(http.StatusBadRequest)
			err := json.NewEncoder(w).Encode(clientErrorResponse)
			Expect(err).NotTo(HaveOccurred())
			return

		case strings.Contains(r.URL.Path, "/bad-json"):
			_, _ = w.Write([]byte(`{"invalid_json": "this is not a ServiceAccountToken}`))
			return

		case !strings.Contains(r.URL.Path, tokenurlsuffix):
			http.NotFound(w, r)
			return
		default:
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
