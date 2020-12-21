/*


Copyright 2020 Red Hat, Inc.

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package crhchttp

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/project-koku/koku-metrics-operator/testutils"
)

type osOpenFunc = func(filename string) (*os.File, error)

func mockOsOpen(file *os.File, err error) osOpenFunc {
	return func(filename string) (*os.File, error) {
		return file, err
	}
}

func TestGetMultiPartBodyAndHeaders(t *testing.T) {
	tempFile, err := ioutil.TempFile("/tmp", "bla.bla.txt")
	if err != nil {
		t.Errorf("failed creating a temp file: %s", err)
	}
	defer os.Remove(tempFile.Name())
	tcs := []struct {
		open        osOpenFunc
		expectedErr error
		expectedBuf *bytes.Buffer
	}{
		{
			mockOsOpen(nil, fmt.Errorf("bad file")),
			fmt.Errorf("failed to open file: bad file"),
			nil,
		},
		{
			mockOsOpen(nil, nil),
			fmt.Errorf("failed to copy file: invalid argument"),
			nil,
		},
		{
			mockOsOpen(tempFile, nil),
			nil,
			nil,
		},
	}

	for _, tc := range tcs {
		osOpen = tc.open
		_, _, err := GetMultiPartBodyAndHeaders("file1.txt")
		if (err != nil && tc.expectedErr == nil) || (err == nil && tc.expectedErr != nil) {
			t.Errorf("Expected error to be %v but got %v", tc.expectedErr, err)
		}
		if err != nil && tc.expectedErr != nil && err.Error() != tc.expectedErr.Error() {
			t.Errorf("Expected error to be '%s' but got '%s'", tc.expectedErr, err)
		}
	}
	osOpen = os.Open
}

func TestSetupRequest(t *testing.T) {
	tcs := []struct {
		auth        *AuthConfig
		expectedErr error
		isBasicAuth bool
	}{
		{
			&AuthConfig{
				ClusterID:         "cluster-id",
				Authentication:    "basic",
				BasicAuthUser:     "masterOfPuppets",
				BasicAuthPassword: "gr33nH3LL",
				ValidateCert:      true,
				OperatorCommit:    "",
				Log:               testutils.TestLogger{},
			},
			nil,
			true,
		},
		{
			&AuthConfig{
				ClusterID:         "cluster-id",
				Authentication:    "something",
				ValidateCert:      true,
				BearerTokenString: "SL1PKN07",
				OperatorCommit:    "",
				Log:               testutils.TestLogger{},
			},
			nil,
			false,
		},
	}
	body := bytes.NewBuffer(nil)
	for _, tc := range tcs {
		req, err := SetupRequest(tc.auth, "application/json", "GET", "/uri1", body)
		if err != nil {
			t.Errorf("Expected SetupRequest not to return error but got '%s'", err)
		}
		authValue := req.Header.Get("Authorization")
		if strings.HasPrefix(authValue, "Bearer ") && tc.isBasicAuth {
			t.Error("Expected request authorization to be Bearer but got Basic")
		}
		if strings.HasPrefix(authValue, "Basic ") && !tc.isBasicAuth {
			t.Error("Expected request authorization to be Basic but got Bearer")
		}
	}
}

func TestSmokeGetClient(t *testing.T) {
	tcs := []struct {
		isValidCert  bool
		readCertFunc ReadCertFileFunc
	}{
		{false, func() ([]byte, error) {
			return nil, nil
		}},
		{true, func() ([]byte, error) {
			return nil, nil
		}},
		{true, func() ([]byte, error) {
			return nil, fmt.Errorf("bad file")
		}},
	}

	for _, tc := range tcs {
		authCfg := &AuthConfig{
			ClusterID:         "cluster-id",
			Authentication:    "bearer",
			BearerTokenString: "WRITEINGO",
			ValidateCert:      tc.isValidCert,
			OperatorCommit:    "",
			Log:               testutils.TestLogger{},
		}
		readCertFile = tc.readCertFunc
		GetClient(authCfg)
	}
}

func TestProcessResponse(t *testing.T) {
	tcs := []struct {
		method       string
		status       int
		url          string
		reqId        string
		body         *bytes.Buffer
		expectedData string
		expectedErr  error
	}{
		{http.MethodGet, http.StatusOK, "/bla", "req-id-1", bytes.NewBuffer(nil), "", nil},
		{http.MethodPost, http.StatusBadRequest, "/bla/id/30302", "req-id-2", bytes.NewBufferString("{}"), "", fmt.Errorf("error response: {}")},
		{http.MethodGet, http.StatusFound, "/bla/id/30302", "req-id-3", bytes.NewBufferString("{}"), "", fmt.Errorf("error response: {}")},
		{http.MethodGet, http.StatusProcessing, "/bla/id/30302", "req-id-3", bytes.NewBufferString("{}"), "", fmt.Errorf("error response: {}")},
	}

	for _, tc := range tcs {
		req, err := http.NewRequest(tc.method, tc.url, tc.body)
		if err != nil {
			t.Errorf("Failed to create a request from test case %s", err)
		}
		hdr := make(http.Header)
		hdr.Set("x-rh-insights-request-id", tc.reqId)
		resp := http.Response{
			Request:    req,
			StatusCode: tc.status,
			Header:     hdr,
			Body:       ioutil.NopCloser(tc.body),
		}
		data, err := ProcessResponse(testutils.TestLogger{}, &resp)
		if (err != nil && tc.expectedErr == nil) || (err == nil && tc.expectedErr != nil) {
			t.Errorf("Expected ProcessResponse to return error %s but got %s", tc.expectedErr, err)
		}
		if err != nil && tc.expectedErr != nil && err.Error() != tc.expectedErr.Error() {
			t.Errorf("Expected ProcessResponse to return error '%s' but got '%s'", tc.expectedErr, err)
		}
		if string(data) != tc.expectedData {
			t.Errorf("Expected ProcessResponse to return data '%s' but got '%s'", tc.expectedData, string(data))
		}
	}
}

func TestUpload(t *testing.T) {
	authCfg := &AuthConfig{
		ClusterID:         "cluster-id",
		Authentication:    "bearer",
		BearerTokenString: "WRITEINGO",
		ValidateCert:      false,
		OperatorCommit:    "",
		Log:               testutils.TestLogger{},
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(w, "Hello, client")
	}))
	defer ts.Close()
	status, _, err := Upload(authCfg, "application/json", "GET", ts.URL, bytes.NewBuffer(nil))
	if status != "200 OK" {
		t.Errorf("Expected Upload to return status '200 OK' but got '%s'", status)
	}
	if err != nil {
		t.Errorf("Expected Upload to return no error but got '%s'", err)
	}
}
