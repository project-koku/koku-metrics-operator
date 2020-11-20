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

package sources

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"

	logr "github.com/go-logr/logr/testing"
	"github.com/project-koku/korekuta-operator-go/crhchttp"
)

var (
	cost = &crhchttp.CostManagementConfig{
		APIURL:         "https://ci.cloud.redhat.com",
		SourcesAPIPath: "/api/sources/v1.0/",
		Log:            testLogger,
	}
	errSources = errors.New("test error")
	testLogger = logr.NullLogger{}
)

// https://www.thegreatcodeadventure.com/mocking-http-requests-in-golang/
type MockClient struct {
	req *http.Request
	res *http.Response
	err error
}

// Do is the mock client's `Do` func
func (m *MockClient) Do(req *http.Request) (*http.Response, error) {
	m.req = req
	return m.res, m.err
}

func escapeQuery(s string) string {
	slice := strings.SplitAfterN(s, "?", 2)
	if len(slice) < 2 {
		return s
	}
	q := slice[1]
	q = strings.ReplaceAll(q, "[", "%5B")
	q = strings.ReplaceAll(q, "]", "%5D")
	q = strings.ReplaceAll(q, "/", "%2F")
	return slice[0] + q
}

func TestGetSourceTypeID(t *testing.T) {
	expectedURL := "https://ci.cloud.redhat.com/api/sources/v1.0/source_types?filter[name]=openshift"
	getSourceTypeIDTests := []struct {
		name        string
		response    *http.Response
		responseErr error
		expected    string
		expectedErr error
	}{
		{
			name: "successful response with data",
			response: &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(strings.NewReader("{\"meta\":{\"count\":1},\"data\":[{\"id\":\"1\",\"name\":\"openshift\"}]}")), // type is io.ReadCloser,
				Request:    &http.Request{Method: "GET", URL: &url.URL{}},
			},
			responseErr: nil,
			expected:    "1",
			expectedErr: nil,
		},
		{
			name:        "request failure",
			response:    &http.Response{},
			responseErr: errSources,
			expected:    "",
			expectedErr: errSources,
		},
		{
			name: "400 bad response",
			response: &http.Response{
				StatusCode: 400,
				Body:       ioutil.NopCloser(strings.NewReader("{\"errors\":[{\"status\":\"400\",\"detail\":\"ArgumentError: Failed to find definition for Name\"}]}")), // type is io.ReadCloser,
				Request:    &http.Request{Method: "GET", URL: &url.URL{}},
			},
			responseErr: nil,
			expected:    "",
			expectedErr: errSources,
		},
		{
			name: "parse error",
			response: &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(strings.NewReader("{\"meta\":{\"count\":1},\"data\":[{\"i:\"openshift\"}]}")), // type is io.ReadCloser,
				Request:    &http.Request{Method: "GET", URL: &url.URL{}},
			},
			responseErr: nil,
			expected:    "",
			expectedErr: errSources,
		},
		{
			name: "too many count from response",
			response: &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(strings.NewReader("{\"meta\":{\"count\":2},\"data\":[{\"id\":\"1\",\"name\":\"openshift\"},{\"id\":\"2\",\"name\":\"amazon\"}]}")), // type is io.ReadCloser,
				Request:    &http.Request{Method: "GET", URL: &url.URL{}},
			},
			responseErr: nil,
			expected:    "",
			expectedErr: errSources,
		},
		{
			name: "no count from response",
			response: &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(strings.NewReader("{\"meta\":{\"count\":0},\"data\":[]}")), // type is io.ReadCloser,
				Request:    &http.Request{Method: "GET", URL: &url.URL{}},
			},
			responseErr: nil,
			expected:    "",
			expectedErr: errSources,
		},
	}
	for _, tt := range getSourceTypeIDTests {
		t.Run(tt.name, func(t *testing.T) {
			clt := &MockClient{res: tt.response, err: tt.responseErr}
			got, err := GetSourceTypeID(cost, clt)
			if tt.expectedErr != nil && err == nil {
				t.Errorf("%s expected error, got: %v", tt.name, err)
			}
			if tt.expectedErr == nil && err != nil {
				t.Errorf("%s got unexpected error: %v", tt.name, err)
			}
			if got != tt.expected {
				t.Errorf("%s got %s want %s", tt.name, got, tt.expected)
			}
			// check that the request query is correctly constructed
			got = clt.req.URL.String()
			want := escapeQuery(expectedURL)
			if got != want {
				t.Errorf("%s\n\tgot:\n\t\t%+v\n\twant:\n\t\t%s", tt.name, got, want)
			}
		})
	}
}

func TestCheckSourceExists(t *testing.T) {
	// checkSourceExistsTests := []struct {
	// 	name         string
	// 	sourceTypeID string
	// 	sourceRef    string
	// 	response     *http.Response
	// 	responseErr  error
	// 	expected     *SourceItem
	// 	expectedErr  error
	// }{
	// 	{},
	// }
	// for _, tt := range checkSourceExistsTests {
	// 	t.Run(tt.name, func(t *testing.T) {
	// 		clt := &MockClient{res: tt.response, err: tt.responseErr}
	// 		got, err := CheckSourceExists(cost, clt, tt.sourceTypeID, tt.name, tt.sourceRef)
	// 		if tt.expectedErr != nil && err == nil {
	// 			t.Errorf("%s expected error, got: %v", tt.name, err)
	// 		}
	// 		if tt.expectedErr == nil && err != nil {
	// 			t.Errorf("%s got unexpected error: %v", tt.name, err)
	// 		}
	// 		if got != tt.expected {
	// 			t.Errorf("%s got %s want %s", tt.name, got, tt.expected)
	// 		}
	// 	})
	// }
}

func TestGetApplicationTypeID(t *testing.T) {
	expectedURL := "https://ci.cloud.redhat.com/api/sources/v1.0/application_types?filter[name]=/insights/platform/cost-management"
	getApplicationTypeIDTests := []struct {
		name        string
		response    *http.Response
		responseErr error
		expected    string
		expectedErr error
	}{
		{
			name: "successful response with data",
			response: &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(strings.NewReader("{\"meta\":{\"count\":1},\"data\":[{\"id\":\"1\",\"name\":\"openshift\"}]}")), // type is io.ReadCloser,
				Request:    &http.Request{Method: "GET", URL: &url.URL{}},
			},
			responseErr: nil,
			expected:    "1",
			expectedErr: nil,
		},
		{
			name:        "request failure",
			response:    &http.Response{},
			responseErr: errSources,
			expected:    "",
			expectedErr: errSources,
		},
		{
			name: "400 bad response",
			response: &http.Response{
				StatusCode: 400,
				Body:       ioutil.NopCloser(strings.NewReader("{\"errors\":[{\"status\":\"400\",\"detail\":\"ArgumentError: Failed to find definition for Name\"}]}")), // type is io.ReadCloser,
				Request:    &http.Request{Method: "GET", URL: &url.URL{}},
			},
			responseErr: nil,
			expected:    "",
			expectedErr: errSources,
		},
		{
			name: "parse error",
			response: &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(strings.NewReader("{\"meta\":{\"count\":1},\"data\":[{\"i:\"openshift\"}]}")), // type is io.ReadCloser,
				Request:    &http.Request{Method: "GET", URL: &url.URL{}},
			},
			responseErr: nil,
			expected:    "",
			expectedErr: errSources,
		},
		{
			name: "too many count from response",
			response: &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(strings.NewReader("{\"meta\":{\"count\":2},\"data\":[{\"id\":\"1\",\"name\":\"openshift\"},{\"id\":\"2\",\"name\":\"amazon\"}]}")), // type is io.ReadCloser,
				Request:    &http.Request{Method: "GET", URL: &url.URL{}},
			},
			responseErr: nil,
			expected:    "",
			expectedErr: errSources,
		},
		{
			name: "no count from response",
			response: &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(strings.NewReader("{\"meta\":{\"count\":0},\"data\":[]}")), // type is io.ReadCloser,
				Request:    &http.Request{Method: "GET", URL: &url.URL{}},
			},
			responseErr: nil,
			expected:    "",
			expectedErr: errSources,
		},
	}
	for _, tt := range getApplicationTypeIDTests {
		t.Run(tt.name, func(t *testing.T) {
			clt := &MockClient{res: tt.response, err: tt.responseErr}
			got, err := GetApplicationTypeID(cost, clt)
			if tt.expectedErr != nil && err == nil {
				t.Errorf("%s expected error, got: %v", tt.name, err)
			}
			if tt.expectedErr == nil && err != nil {
				t.Errorf("%s got unexpected error: %v", tt.name, err)
			}
			if got != tt.expected {
				t.Errorf("%s got %s want %s", tt.name, got, tt.expected)
			}
			// check that the request query is correctly constructed
			got = clt.req.URL.String()
			want := escapeQuery(expectedURL)
			if got != want {
				t.Errorf("%s\n\tgot:\n\t\t%+v\n\twant:\n\t\t%s", tt.name, got, want)
			}
		})
	}
}
