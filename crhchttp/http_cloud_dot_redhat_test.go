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
	"io"
	"io/ioutil"
	"mime/multipart"
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

func mocMultiPartWriter(err error) CreateMultiPartWriterFunc {
	return func(filename string) (*bytes.Buffer, *multipart.Writer, io.Writer, error) {
		buf := new(bytes.Buffer)
		mw := multipart.NewWriter(buf)
		return buf, mw, buf, err
	}
}

func TestGetMultiPartBodyAndHeaders(t *testing.T) {
	tempFile, err := ioutil.TempFile("/tmp", "bla.bla.txt")
	if err != nil {
		t.Errorf("failed creating a temp file: %s", err)
	}
	defer os.Remove(tempFile.Name())
	tcs := []struct {
		open           osOpenFunc
		createMpWriter CreateMultiPartWriterFunc
		expectedErr    error
		expectedBuf    *bytes.Buffer
	}{
		{
			mockOsOpen(nil, fmt.Errorf("bad file")),
			mocMultiPartWriter(nil),
			fmt.Errorf("failed to open file: bad file"),
			nil,
		},
		{
			mockOsOpen(nil, nil),
			mocMultiPartWriter(nil),
			fmt.Errorf("failed to copy file: invalid argument"),
			nil,
		},
		{
			mockOsOpen(tempFile, nil),
			createMultiPartWriter,
			nil,
			nil,
		},
		{
			mockOsOpen(nil, nil),
			mocMultiPartWriter(fmt.Errorf("bad buffer")),
			fmt.Errorf("failed to create part: bad buffer"),
			nil,
		},
	}

	for _, tc := range tcs {
		osOpen = tc.open
		tmpFnc := createMultiPartWriter
		createMultiPartWriter = tc.createMpWriter
		defer func() {
			osOpen = os.Open
			createMultiPartWriter = tmpFnc
		}()
		_, _, err := GetMultiPartBodyAndHeaders("file1.txt")
		if (err != nil && tc.expectedErr == nil) || (err == nil && tc.expectedErr != nil) {
			t.Errorf("Expected error to be %v but got %v", tc.expectedErr, err)
		}
		if err != nil && tc.expectedErr != nil && err.Error() != tc.expectedErr.Error() {
			t.Errorf("Expected error to be '%s' but got '%s'", tc.expectedErr, err)
		}
	}
}

func TestSetupRequest(t *testing.T) {
	tcs := []struct {
		reqFactory  RequestFactory
		auth        *AuthConfig
		expectedErr error
		isBasicAuth bool
	}{
		{
			generateRequest,
			&AuthConfig{
				ClusterID:         "cluster-id-1",
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
			generateRequest,
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
		{
			func(m, u string, b *bytes.Buffer) (*http.Request, error) {
				return nil, fmt.Errorf("something went wrong")
			},
			&AuthConfig{
				ClusterID:         "id",
				Authentication:    "",
				ValidateCert:      true,
				BearerTokenString: "",
				OperatorCommit:    "",
				Log:               testutils.TestLogger{},
			},
			fmt.Errorf("could not create request: something went wrong"),
			false,
		},
	}
	body := bytes.NewBuffer(nil)
	for _, tc := range tcs {
		tmpFnc := generateRequest
		generateRequest = tc.reqFactory
		defer func() {
			generateRequest = tmpFnc
		}()
		req, err := SetupRequest(tc.auth, "application/json", "GET", "/uri1", body)
		if err != nil && tc.expectedErr == nil {
			t.Errorf("Expected SetupRequest not to return error but got '%s'", err)
		}
		if req != nil {
			authValue := req.Header.Get("Authorization")
			if strings.HasPrefix(authValue, "Bearer ") && tc.isBasicAuth {
				t.Error("Expected request authorization to be Bearer but got Basic")
			}
			if strings.HasPrefix(authValue, "Basic ") && !tc.isBasicAuth {
				t.Error("Expected request authorization to be Basic but got Bearer")
			}
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
		tmpFnc := readCertFile
		readCertFile = tc.readCertFunc
		defer func() {
			readCertFile = tmpFnc
		}()
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
		dumper       ResponseDumper
		expectedData string
		expectedErr  error
	}{
		{http.MethodGet, http.StatusOK, "/bla", "req-id-1", bytes.NewBuffer(nil), responseDump, "", nil},
		{http.MethodPost, http.StatusBadRequest, "/bla/id/30302", "req-id-2", bytes.NewBufferString("{}"), responseDump, "", fmt.Errorf("error response: {}")},
		{http.MethodGet, http.StatusFound, "/bla/id/30302", "req-id-3", bytes.NewBufferString("{}"), responseDump, "", fmt.Errorf("error response: {}")},
		{http.MethodGet, http.StatusProcessing, "/bla/id/30302", "req-id-3", bytes.NewBufferString("{}"), responseDump, "", fmt.Errorf("error response: {}")},
		{http.MethodGet, http.StatusOK, "/bla", "req-id-1", bytes.NewBuffer(nil), func(r *http.Response, b bool) ([]byte, error) {
			return nil, fmt.Errorf("what are you doing?")
		}, "", fmt.Errorf("failed to dump response body: what are you doing?")},
		{http.MethodGet, http.StatusOK, "/bla", "req-id-1", bytes.NewBuffer(nil), func(r *http.Response, b bool) ([]byte, error) {
			return []byte("HEADERS"), nil
		}, "", fmt.Errorf("failed to read response body: DumpResponse split length does not equal 2")},
	}

	for _, tc := range tcs {
		tmpFunc := responseDump
		responseDump = tc.dumper
		defer func() {
			responseDump = tmpFunc
		}()
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

func TestDefaultReadCertFile(t *testing.T) {
	_, statErr := os.Stat(cacerts)
	doesExist := false
	if statErr == nil {
		doesExist = true
	}
	_, err := readCertFile()
	if err != nil && doesExist {
		t.Errorf("%s exists but reading it failed: %s", cacerts, err)
	} else if err == nil && !doesExist {
		t.Errorf("%s does not exist but reading was supposed to fails", cacerts)
	}
}
