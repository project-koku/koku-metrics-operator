//
// Copyright 2021 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package sources

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	metricscfgv1beta1 "github.com/project-koku/koku-metrics-operator/api/v1beta1"
	"github.com/project-koku/koku-metrics-operator/internal/crhchttp"
	"github.com/project-koku/koku-metrics-operator/internal/testutils"
)

var (
	auth    = &crhchttp.AuthConfig{ClusterID: "post-cluster-id"}
	handler = &SourceHandler{
		APIURL: "https://ci.console.redhat.com",
		Auth:   auth,
		Spec: metricscfgv1beta1.CloudDotRedHatSourceStatus{
			SourcesAPIPath: "/api/sources/v1.0/",
			SourceName:     "post-source-name",
		},
	}
	errSources = errors.New("test error")
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

type MockClientList struct {
	clients []MockClient
}

// Do is the mock client's `Do` func
func (ml *MockClientList) Do(req *http.Request) (*http.Response, error) {
	if len(ml.clients) <= 0 {
		return nil, fmt.Errorf("no more clients")
	}
	var m MockClient
	m, ml.clients = ml.clients[0], ml.clients[1:]
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

func getReqBody(t *testing.T, req *http.Request) []byte {
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	return bodyBytes
}

func TestMain(m *testing.M) {
	logf.SetLogger(testutils.ZapLogger(true))
	code := m.Run()
	os.Exit(code)
}

func TestGetSourceTypeID(t *testing.T) {
	expectedURL := "https://ci.console.redhat.com/api/sources/v1.0/source_types?filter[name]=openshift"
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
				Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":1},\"data\":[{\"id\":\"1\",\"name\":\"openshift\"}]}")), // type is io.ReadCloser,
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
				Body:       io.NopCloser(strings.NewReader("{\"errors\":[{\"status\":\"400\",\"detail\":\"ArgumentError: Failed to find definition for Name\"}]}")), // type is io.ReadCloser,
				Request:    &http.Request{Method: "GET", URL: &url.URL{}},
			},
			responseErr: nil,
			expected:    "",
			expectedErr: errSources,
		},
		{
			name: "parse error", // response body is bad json
			response: &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":1},\"data\":[{\"i:\"openshift\"}]}")), // type is io.ReadCloser,
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
				Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":2},\"data\":[{\"id\":\"1\",\"name\":\"openshift\"},{\"id\":\"2\",\"name\":\"amazon\"}]}")), // type is io.ReadCloser,
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
				Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":0},\"data\":[]}")), // type is io.ReadCloser,
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
			got, err := GetSourceTypeID(handler, clt)
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

func TestGetSources(t *testing.T) {
	getSourcesTests := []struct {
		name        string
		response    *http.Response
		responseErr error
		expected    []byte
		expectedErr error
	}{
		{
			name: "successful response with data",
			response: &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":1},\"data\":[{\"id\":\"1\",\"name\":\"openshift\"}]}")), // type is io.ReadCloser,
				Request:    &http.Request{Method: "GET", URL: &url.URL{}},
			},
			responseErr: nil,
			expected:    []byte("1"),
			expectedErr: nil,
		},
		{
			name:        "request failure",
			response:    &http.Response{},
			responseErr: errSources,
			expected:    nil,
			expectedErr: errSources,
		},
	}
	for _, tt := range getSourcesTests {
		t.Run(tt.name, func(t *testing.T) {
			clt := &MockClient{res: tt.response, err: tt.responseErr}
			got, err := GetSources(handler, clt)
			if tt.expectedErr != nil && err == nil {
				t.Errorf("%s expected error, got: %v", tt.name, err)
			}
			if tt.expectedErr == nil && err != nil {
				t.Errorf("%s got unexpected error: %v", tt.name, err)
			}
			if len(got) > 0 && len(tt.expected) <= 0 {
				t.Errorf("%s got %s want %s", tt.name, got, tt.expected)
			}
			if len(got) <= 0 && len(tt.expected) > 0 {
				t.Errorf("%s got %s want %s", tt.name, got, tt.expected)
			}
		})
	}
}

func TestCheckSourceExists(t *testing.T) {
	// https://console.redhat.com/api/sources/v1.0/sources?filter[source_type_id]=1&filter[source_ref]=eb93b259-1369-4f90-88ce-e68c6ba879a9&filter[name]=OpenShift%20on%20Azure
	checkSourceExistsTests := []struct {
		name          string
		queryname     string
		sourceTypeID  string
		sourceRef     string
		response      *http.Response
		responseErr   error
		expected      *SourceItem
		expectedQuery string
		expectedErr   error
	}{
		{
			name:          "query test",
			queryname:     "name",
			sourceTypeID:  "1",
			sourceRef:     "12345",
			responseErr:   errSources,
			expectedQuery: "https://ci.console.redhat.com/api/sources/v1.0/sources?filter[name]=name&filter[source_ref]=12345&filter[source_type_id]=1",
			expectedErr:   errSources,
		},
		{
			name:          "query test",
			queryname:     "name",
			sourceRef:     "12345",
			responseErr:   errSources,
			expectedQuery: "https://ci.console.redhat.com/api/sources/v1.0/sources?filter[name]=name&filter[source_ref]=12345",
			expectedErr:   errSources,
		},
		{
			name:          "query test",
			sourceTypeID:  "1",
			sourceRef:     "12345",
			responseErr:   errSources,
			expectedQuery: "https://ci.console.redhat.com/api/sources/v1.0/sources?filter[source_ref]=12345&filter[source_type_id]=1",
			expectedErr:   errSources,
		},
		{
			name:          "query test",
			queryname:     "name",
			sourceTypeID:  "1",
			responseErr:   errSources,
			expectedQuery: "https://ci.console.redhat.com/api/sources/v1.0/sources?filter[name]=name&filter[source_type_id]=1",
			expectedErr:   errSources,
		},
		{
			name:          "query test",
			queryname:     "name",
			responseErr:   errSources,
			expectedQuery: "https://ci.console.redhat.com/api/sources/v1.0/sources?filter[name]=name",
			expectedErr:   errSources,
		},
		{
			name:          "query test",
			sourceTypeID:  "1",
			responseErr:   errSources,
			expectedQuery: "https://ci.console.redhat.com/api/sources/v1.0/sources?filter[source_type_id]=1",
			expectedErr:   errSources,
		},
		{
			name:          "query test",
			sourceRef:     "12345",
			responseErr:   errSources,
			expectedQuery: "https://ci.console.redhat.com/api/sources/v1.0/sources?filter[source_ref]=12345",
			expectedErr:   errSources,
		},
		{
			name:          "query test",
			responseErr:   errSources,
			expectedQuery: "https://ci.console.redhat.com/api/sources/v1.0/sources",
			expectedErr:   errSources,
		},
		{
			name: "successful response with data",
			response: &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":1},\"data\":[{\"id\":\"1\",\"name\":\"name\",\"source_type_id\":\"1\",\"source_ref\":\"12345\"}]}")), // type is io.ReadCloser,
				Request:    &http.Request{Method: "GET", URL: &url.URL{}},
			},
			responseErr:   nil,
			expected:      &SourceItem{ID: "1", Name: "name", SourceTypeID: "1", SourceRef: "12345"},
			expectedQuery: "https://ci.console.redhat.com/api/sources/v1.0/sources",
			expectedErr:   nil,
		},
		{
			name: "400 bad response",
			response: &http.Response{
				StatusCode: 400,
				Body:       io.NopCloser(strings.NewReader("{\"errors\":[{\"status\":\"400\",\"detail\":\"ArgumentError: Failed to find definition for Name\"}]}")), // type is io.ReadCloser,
				Request:    &http.Request{Method: "GET", URL: &url.URL{}},
			},
			expectedQuery: "https://ci.console.redhat.com/api/sources/v1.0/sources",
			responseErr:   nil,
			expectedErr:   errSources,
		},
		{
			name: "parse error", // response body is bad json
			response: &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":1},\"data\":[{\"i:\"openshift\"}]}")), // type is io.ReadCloser,
				Request:    &http.Request{Method: "GET", URL: &url.URL{}},
			},
			responseErr:   nil,
			expectedQuery: "https://ci.console.redhat.com/api/sources/v1.0/sources",
			expectedErr:   errSources,
		},
		{
			name:          "request failure",
			response:      &http.Response{},
			responseErr:   errSources,
			expectedQuery: "https://ci.console.redhat.com/api/sources/v1.0/sources",
			expectedErr:   errSources,
		},
		{
			name: "too many count from response",
			response: &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":2},\"data\":[{\"id\":\"1\",\"name\":\"name\",\"source_type_id\":\"1\",\"source_ref\":\"12345\"},{\"id\":\"2\",\"name\":\"name2\",\"source_type_id\":\"3\",\"source_ref\":\"67890\"}]}")), // type is io.ReadCloser,
				Request:    &http.Request{Method: "GET", URL: &url.URL{}},
			},
			responseErr:   nil,
			expectedQuery: "https://ci.console.redhat.com/api/sources/v1.0/sources",
			expectedErr:   nil,
		},
		{
			name: "no count from response",
			response: &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":0},\"data\":[]}")), // type is io.ReadCloser,
				Request:    &http.Request{Method: "GET", URL: &url.URL{}},
			},
			responseErr:   nil,
			expectedQuery: "https://ci.console.redhat.com/api/sources/v1.0/sources",
			expectedErr:   nil,
		},
	}
	for _, tt := range checkSourceExistsTests {
		t.Run(tt.name, func(t *testing.T) {
			clt := &MockClient{res: tt.response, err: tt.responseErr}
			got, err := CheckSourceExists(handler, clt, tt.sourceTypeID, tt.queryname, tt.sourceRef)
			if tt.expectedErr != nil && err == nil {
				t.Errorf("%s expected error, got: %v", tt.name, err)
			}
			if tt.expectedErr == nil && err != nil {
				t.Errorf("%s got unexpected error: %v", tt.name, err)
			}
			if got != nil && !reflect.DeepEqual(*got, *tt.expected) {
				t.Errorf("%s got %v want %v", tt.name, got, tt.expected)
			}
			if got == nil && !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("%s got %v want %v", tt.name, got, tt.expected)
			}
			// check that the request query is correctly constructed
			gotURL := clt.req.URL.String()
			wantURL := escapeQuery(tt.expectedQuery)
			if gotURL != wantURL {
				t.Errorf("%s\n\tgot:\n\t\t%+v\n\twant:\n\t\t%s", tt.name, gotURL, wantURL)
			}
		})
	}
}

func TestGetApplicationTypeID(t *testing.T) {
	expectedURL := "https://ci.console.redhat.com/api/sources/v1.0/application_types?filter[name]=/insights/platform/cost-management"
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
				Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":1},\"data\":[{\"id\":\"1\",\"name\":\"openshift\"}]}")), // type is io.ReadCloser,
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
				Body:       io.NopCloser(strings.NewReader("{\"errors\":[{\"status\":\"400\",\"detail\":\"ArgumentError: Failed to find definition for Name\"}]}")), // type is io.ReadCloser,
				Request:    &http.Request{Method: "GET", URL: &url.URL{}},
			},
			responseErr: nil,
			expected:    "",
			expectedErr: errSources,
		},
		{
			name: "parse error", // response body is bad json
			response: &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":1},\"data\":[{\"i:\"openshift\"}]}")), // type is io.ReadCloser,
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
				Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":2},\"data\":[{\"id\":\"1\",\"name\":\"openshift\"},{\"id\":\"2\",\"name\":\"amazon\"}]}")), // type is io.ReadCloser,
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
				Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":0},\"data\":[]}")), // type is io.ReadCloser,
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
			got, err := GetApplicationTypeID(handler, clt)
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

func TestPostSource(t *testing.T) {
	expectedURL := "https://ci.console.redhat.com/api/sources/v1.0/sources"
	postSourceTests := []struct {
		name         string
		response     *http.Response
		responseErr  error
		sourceTypeID string
		expected     *SourceItem
		expectedBody []byte
		expectedErr  error
	}{
		{
			name: "successful response with data",
			response: &http.Response{
				StatusCode: 201,
				Body:       io.NopCloser(strings.NewReader("{\"id\":\"11\",\"name\":\"testSource01\",\"source_ref\":\"12345\",\"source_type_id\":\"1\",\"uid\":\"abcdef\"}")), // type is io.ReadCloser,
				Request:    &http.Request{Method: "POST", URL: &url.URL{}},
			},
			responseErr:  nil,
			sourceTypeID: "1",
			expected:     &SourceItem{ID: "11", Name: "testSource01", SourceTypeID: "1", SourceRef: "12345"},
			expectedBody: []byte(`{"name":"post-source-name","source_ref":"post-cluster-id","source_type_id":"1"}`),
			expectedErr:  nil,
		},
		{
			name:         "request failure",
			response:     &http.Response{},
			responseErr:  errSources,
			expected:     nil,
			expectedBody: []byte(`{"name":"post-source-name","source_ref":"post-cluster-id","source_type_id":""}`),
			expectedErr:  errSources,
		},
		{
			name: "400 bad response",
			response: &http.Response{
				StatusCode: 400,
				Body:       io.NopCloser(strings.NewReader("{\"errors\":[{\"status\":\"400\",\"detail\":\"Invalid parameter - Validation failed: Source type must exist\"}]}")), // type is io.ReadCloser,
				Request:    &http.Request{Method: "POST", URL: &url.URL{}},
			},
			responseErr:  nil,
			expected:     nil,
			expectedBody: []byte(`{"name":"post-source-name","source_ref":"post-cluster-id","source_type_id":""}`),
			expectedErr:  errSources,
		},
		{
			name: "parse error", // response body is bad json
			response: &http.Response{
				StatusCode: 201,
				Body:       io.NopCloser(strings.NewReader("{\"created_at\":\"2020-11-20T21:37:27Z\",\"id\":\"18292\"")), // type is io.ReadCloser,
				Request:    &http.Request{Method: "POST", URL: &url.URL{}},
			},
			responseErr:  nil,
			expected:     nil,
			expectedBody: []byte(`{"name":"post-source-name","source_ref":"post-cluster-id","source_type_id":""}`),
			expectedErr:  errSources,
		},
	}
	for _, tt := range postSourceTests {
		t.Run(tt.name, func(t *testing.T) {
			clt := &MockClient{res: tt.response, err: tt.responseErr}
			got, err := PostSource(handler, clt, tt.sourceTypeID)
			if tt.expectedErr != nil && err == nil {
				t.Errorf("%s expected error, got: %v", tt.name, err)
			}
			if tt.expectedErr == nil && err != nil {
				t.Errorf("%s got unexpected error: %v", tt.name, err)
			}
			if got != nil && !reflect.DeepEqual(*got, *tt.expected) {
				t.Errorf("%s got %v want %v", tt.name, got, tt.expected)
			}
			if got == nil && !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("%s got %v want %v", tt.name, got, tt.expected)
			}
			// check that the request url is correctly constructed
			gotURL := clt.req.URL.String()
			wantURL := escapeQuery(expectedURL)
			if gotURL != wantURL {
				t.Errorf("%s\n\tgot:\n\t\t%+v\n\twant:\n\t\t%s", tt.name, gotURL, wantURL)
			}

			// check that the request Body is correctly constructed
			gotBody := getReqBody(t, clt.req)
			if !reflect.DeepEqual(gotBody, tt.expectedBody) {
				t.Errorf("%s\n\tgot:\n\t\t%s\n\twant:\n\t\t%s", tt.name, gotBody, tt.expectedBody)
			}
		})
	}
}

func TestPostApplication(t *testing.T) {
	expectedURL := "https://ci.console.redhat.com/api/sources/v1.0/applications"
	postSourceTests := []struct {
		name         string
		response     *http.Response
		responseErr  error
		source       *SourceItem
		appTypeID    string
		expectedBody []byte
		expectedErr  error
	}{
		{
			name: "successful response with data",
			response: &http.Response{
				StatusCode: 201,
				Body:       io.NopCloser(strings.NewReader("{\"created_at\":\"2020-11-20T21:37:27Z\",\"id\":\"18292\"}")), // type is io.ReadCloser,
				Request:    &http.Request{Method: "POST", URL: &url.URL{}},
			},
			responseErr:  nil,
			source:       &SourceItem{ID: "11", Name: "testSource01", SourceTypeID: "1", SourceRef: "12345"},
			appTypeID:    "1",
			expectedBody: []byte(`{"application_type_id":"1","source_id":"11"}`),
			expectedErr:  nil,
		},
		{
			name:         "request failure",
			response:     &http.Response{},
			responseErr:  errSources,
			source:       &SourceItem{ID: "11", Name: "testSource01", SourceTypeID: "1", SourceRef: "12345"},
			appTypeID:    "1",
			expectedBody: []byte(`{"application_type_id":"1","source_id":"11"}`),
			expectedErr:  errSources,
		},
		{
			name: "400 bad response",
			response: &http.Response{
				StatusCode: 400,
				Body:       io.NopCloser(strings.NewReader("{\"errors\":[{\"status\":\"400\",\"detail\":\"OpenAPIParser::InvalidPattern: #/components/schemas/ID pattern ^\\d+$ does not match value: source.ID\"}]}")), // type is io.ReadCloser,
				Request:    &http.Request{Method: "POST", URL: &url.URL{}},
			},
			responseErr:  nil,
			source:       &SourceItem{ID: "11", Name: "testSource01", SourceTypeID: "1", SourceRef: "12345"},
			appTypeID:    "",
			expectedBody: []byte(`{"application_type_id":"","source_id":"11"}`),
			expectedErr:  errSources,
		},
	}
	for _, tt := range postSourceTests {
		t.Run(tt.name, func(t *testing.T) {
			clt := &MockClient{res: tt.response, err: tt.responseErr}
			err := PostApplication(handler, clt, tt.source, tt.appTypeID)
			if tt.expectedErr != nil && err == nil {
				t.Errorf("%s expected error, got: %v", tt.name, err)
			}
			if tt.expectedErr == nil && err != nil {
				t.Errorf("%s got unexpected error: %v", tt.name, err)
			}
			// check that the request url is correctly constructed
			gotURL := clt.req.URL.String()
			wantURL := escapeQuery(expectedURL)
			if gotURL != wantURL {
				t.Errorf("%s\n\tgot:\n\t\t%+v\n\twant:\n\t\t%s", tt.name, gotURL, wantURL)
			}

			// check that the request Body is correctly constructed
			gotBody := getReqBody(t, clt.req)
			if !reflect.DeepEqual(gotBody, tt.expectedBody) {
				t.Errorf("%s\n\tgot:\n\t\t%s\n\twant:\n\t\t%s", tt.name, gotBody, tt.expectedBody)
			}
		})
	}
}

func TestSourceCreate(t *testing.T) {
	sourceCreateTests := []struct {
		name         string
		clts         MockClientList
		source       *SourceItem
		sourceTypeID string
		expectedErr  error
	}{
		{
			name: "GetApplicationTypeID failed",
			clts: MockClientList{clients: []MockClient{
				{
					res: &http.Response{},
					err: errSources,
				},
			}},
			source:       nil,
			sourceTypeID: "1",
			expectedErr:  errSources,
		},
		{
			name: "PostSource failed",
			clts: MockClientList{clients: []MockClient{
				{
					res: &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":1},\"data\":[{\"id\":\"1\",\"name\":\"openshift\"}]}")), // type is io.ReadCloser,
						Request:    &http.Request{Method: "GET", URL: &url.URL{}},
					},
					err: nil,
				},
				{
					res: &http.Response{},
					err: errSources,
				},
			}},
			source:       nil,
			sourceTypeID: "1",
			expectedErr:  errSources,
		},
		{
			name: "PostApplication failed",
			clts: MockClientList{clients: []MockClient{
				{
					res: &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":1},\"data\":[{\"id\":\"1\",\"name\":\"openshift\"}]}")), // type is io.ReadCloser,
						Request:    &http.Request{Method: "GET", URL: &url.URL{}},
					},
					err: nil,
				},
				{
					res: &http.Response{
						StatusCode: 201,
						Body:       io.NopCloser(strings.NewReader("{\"id\":\"11\",\"name\":\"testSource01\",\"source_ref\":\"12345\",\"source_type_id\":\"1\",\"uid\":\"abcdef\"}")), // type is io.ReadCloser,
						Request:    &http.Request{Method: "POST", URL: &url.URL{}},
					},
					err: nil,
				},
				{
					res: &http.Response{},
					err: errSources,
				},
			}},
			source:       &SourceItem{ID: "11", Name: "testSource01", SourceTypeID: "1", SourceRef: "12345"},
			sourceTypeID: "1",
			expectedErr:  errSources,
		},
		{
			name: "successful source and application create",
			clts: MockClientList{clients: []MockClient{
				{
					res: &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":1},\"data\":[{\"id\":\"1\",\"name\":\"openshift\"}]}")), // type is io.ReadCloser,
						Request:    &http.Request{Method: "GET", URL: &url.URL{}},
					},
					err: nil,
				},
				{
					res: &http.Response{
						StatusCode: 201,
						Body:       io.NopCloser(strings.NewReader("{\"id\":\"11\",\"name\":\"testSource01\",\"source_ref\":\"12345\",\"source_type_id\":\"1\",\"uid\":\"abcdef\"}")), // type is io.ReadCloser,
						Request:    &http.Request{Method: "POST", URL: &url.URL{}},
					},
					err: nil,
				},
				{
					res: &http.Response{
						StatusCode: 201,
						Body:       io.NopCloser(strings.NewReader("{\"created_at\":\"2020-11-20T21:37:27Z\",\"id\":\"18292\"}")), // type is io.ReadCloser,
						Request:    &http.Request{Method: "POST", URL: &url.URL{}},
					},
					err: nil,
				},
			}},
			source:       &SourceItem{ID: "11", Name: "testSource01", SourceTypeID: "1", SourceRef: "12345"},
			sourceTypeID: "1",
			expectedErr:  nil,
		},
	}
	for _, tt := range sourceCreateTests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SourceCreate(handler, &tt.clts, tt.sourceTypeID)
			if tt.expectedErr != nil && err == nil {
				t.Errorf("%s expected error, got: %v", tt.name, err)
			}
			if tt.expectedErr == nil && err != nil {
				t.Errorf("%s got unexpected error: %v", tt.name, err)
			}
			if got != nil && !reflect.DeepEqual(*got, *tt.source) {
				t.Errorf("%s got %v want %v", tt.name, got, tt.source)
			}
			if got == nil && !reflect.DeepEqual(got, tt.source) {
				t.Errorf("%s got %v want %v", tt.name, got, tt.source)
			}
		})
	}
}

func TestSourceGetOrCreate(t *testing.T) {
	sourceGetOrCreateTests := []struct {
		name        string
		clts        MockClientList
		create      bool
		want        bool
		expectedErr error
	}{
		{
			name: "failed GetSourceTypeID",
			clts: MockClientList{clients: []MockClient{
				{
					res: &http.Response{},
					err: errSources,
				},
			}},
			create:      false,
			want:        false,
			expectedErr: errSources,
		},
		{
			name: "failed CheckSourceExists - `Check if Source exists already`",
			clts: MockClientList{clients: []MockClient{
				{
					res: &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":1},\"data\":[{\"id\":\"1\",\"name\":\"openshift\"}]}")), // type is io.ReadCloser,
						Request:    &http.Request{Method: "GET", URL: &url.URL{}},
					},
					err: nil,
				},
				{
					res: &http.Response{},
					err: errSources,
				},
			}},
			create:      false,
			want:        false,
			expectedErr: errSources,
		},
		{
			name: "successful CheckSourceExists - source already exists",
			clts: MockClientList{clients: []MockClient{
				{
					res: &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":1},\"data\":[{\"id\":\"1\",\"name\":\"openshift\"}]}")), // type is io.ReadCloser,
						Request:    &http.Request{Method: "GET", URL: &url.URL{}},
					},
					err: nil,
				},
				{
					res: &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":1},\"data\":[{\"id\":\"1\",\"name\":\"name\",\"source_type_id\":\"1\",\"source_ref\":\"12345\"}]}")), // type is io.ReadCloser,
						Request:    &http.Request{Method: "GET", URL: &url.URL{}},
					},
					err: nil,
				},
			}},
			create:      false,
			want:        true,
			expectedErr: nil,
		},
		{
			name: "successful source_create is false",
			clts: MockClientList{clients: []MockClient{
				{
					res: &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":1},\"data\":[{\"id\":\"1\",\"name\":\"openshift\"}]}")), // type is io.ReadCloser,
						Request:    &http.Request{Method: "GET", URL: &url.URL{}},
					},
					err: nil,
				},
				{
					res: &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":0},\"data\":[]}")), // type is io.ReadCloser,
						Request:    &http.Request{Method: "GET", URL: &url.URL{}},
					},
					err: nil,
				},
			}},
			create:      false,
			want:        false,
			expectedErr: errSources,
		},
		{
			name: "failed CheckSourceExists - `Check if cluster ID is registered`",
			clts: MockClientList{clients: []MockClient{
				{
					res: &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":1},\"data\":[{\"id\":\"1\",\"name\":\"openshift\"}]}")), // type is io.ReadCloser,
						Request:    &http.Request{Method: "GET", URL: &url.URL{}},
					},
					err: nil,
				},
				{
					res: &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":0},\"data\":[]}")), // type is io.ReadCloser,
						Request:    &http.Request{Method: "GET", URL: &url.URL{}},
					},
					err: nil,
				},
				{
					res: &http.Response{},
					err: errSources,
				},
			}},
			create:      true,
			want:        false,
			expectedErr: errSources,
		},
		{
			name: "successful CheckSourceExists - `cluster ID is already registered`",
			clts: MockClientList{clients: []MockClient{
				{
					res: &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":1},\"data\":[{\"id\":\"1\",\"name\":\"openshift\"}]}")), // type is io.ReadCloser,
						Request:    &http.Request{Method: "GET", URL: &url.URL{}},
					},
					err: nil,
				},
				{
					res: &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":0},\"data\":[]}")), // type is io.ReadCloser,
						Request:    &http.Request{Method: "GET", URL: &url.URL{}},
					},
					err: nil,
				},
				{
					res: &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":1},\"data\":[{\"id\":\"1\",\"name\":\"name\",\"source_type_id\":\"1\",\"source_ref\":\"12345\"}]}")), // type is io.ReadCloser,
						Request:    &http.Request{Method: "GET", URL: &url.URL{}},
					},
					err: nil,
				},
			}},
			create:      true,
			want:        false,
			expectedErr: errSources,
		},
		{
			name: "failed CheckSourceExists - `Check if source name is already in use`",
			clts: MockClientList{clients: []MockClient{
				{
					res: &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":1},\"data\":[{\"id\":\"1\",\"name\":\"openshift\"}]}")), // type is io.ReadCloser,
						Request:    &http.Request{Method: "GET", URL: &url.URL{}},
					},
					err: nil,
				},
				{
					res: &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":0},\"data\":[]}")), // type is io.ReadCloser,
						Request:    &http.Request{Method: "GET", URL: &url.URL{}},
					},
					err: nil,
				},
				{
					res: &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":0},\"data\":[]}")), // type is io.ReadCloser,
						Request:    &http.Request{Method: "GET", URL: &url.URL{}},
					},
					err: nil,
				},
				{
					res: &http.Response{},
					err: errSources,
				},
			}},
			create:      true,
			want:        false,
			expectedErr: errSources,
		},
		{
			name: "successful CheckSourceExists - `source name is already in use for non-Openshift source`",
			clts: MockClientList{clients: []MockClient{
				{
					res: &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":1},\"data\":[{\"id\":\"400\",\"name\":\"openshift\"}]}")), // type is io.ReadCloser,
						Request:    &http.Request{Method: "GET", URL: &url.URL{}},
					},
					err: nil,
				},
				{
					res: &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":0},\"data\":[]}")), // type is io.ReadCloser,
						Request:    &http.Request{Method: "GET", URL: &url.URL{}},
					},
					err: nil,
				},
				{
					res: &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":0},\"data\":[]}")), // type is io.ReadCloser,
						Request:    &http.Request{Method: "GET", URL: &url.URL{}},
					},
					err: nil,
				},
				{
					res: &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":1},\"data\":[{\"id\":\"1\",\"name\":\"name\",\"source_type_id\":\"1\",\"source_ref\":\"12345\"}]}")), // type is io.ReadCloser,
						Request:    &http.Request{Method: "GET", URL: &url.URL{}},
					},
					err: nil,
				},
			}},
			create:      true,
			want:        false,
			expectedErr: errSources,
		},
		{
			name: "successful CheckSourceExists - `source name is already in use for another Openshift source`",
			clts: MockClientList{clients: []MockClient{
				{
					res: &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":1},\"data\":[{\"id\":\"1\",\"name\":\"openshift\"}]}")), // type is io.ReadCloser,
						Request:    &http.Request{Method: "GET", URL: &url.URL{}},
					},
					err: nil,
				},
				{
					res: &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":0},\"data\":[]}")), // type is io.ReadCloser,
						Request:    &http.Request{Method: "GET", URL: &url.URL{}},
					},
					err: nil,
				},
				{
					res: &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":0},\"data\":[]}")), // type is io.ReadCloser,
						Request:    &http.Request{Method: "GET", URL: &url.URL{}},
					},
					err: nil,
				},
				{
					res: &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":1},\"data\":[{\"id\":\"1\",\"name\":\"name\",\"source_type_id\":\"1\",\"source_ref\":\"12345\"}]}")), // type is io.ReadCloser,
						Request:    &http.Request{Method: "GET", URL: &url.URL{}},
					},
					err: nil,
				},
			}},
			create:      true,
			want:        false,
			expectedErr: errSources,
		},

		{
			name: "failed source create",
			clts: MockClientList{clients: []MockClient{
				{
					res: &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":1},\"data\":[{\"id\":\"1\",\"name\":\"openshift\"}]}")), // type is io.ReadCloser,
						Request:    &http.Request{Method: "GET", URL: &url.URL{}},
					},
					err: nil,
				},
				{
					res: &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":0},\"data\":[]}")), // type is io.ReadCloser,
						Request:    &http.Request{Method: "GET", URL: &url.URL{}},
					},
					err: nil,
				},
				{
					res: &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":0},\"data\":[]}")), // type is io.ReadCloser,
						Request:    &http.Request{Method: "GET", URL: &url.URL{}},
					},
					err: nil,
				},
				{
					res: &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":0},\"data\":[]}")), // type is io.ReadCloser,
						Request:    &http.Request{Method: "GET", URL: &url.URL{}},
					},
					err: nil,
				},
				{
					res: &http.Response{},
					err: errSources,
				},
			}},
			create:      true,
			want:        false,
			expectedErr: errSources,
		},
		{
			name: "successful source create",
			clts: MockClientList{clients: []MockClient{
				{
					res: &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":1},\"data\":[{\"id\":\"1\",\"name\":\"openshift\"}]}")), // type is io.ReadCloser,
						Request:    &http.Request{Method: "GET", URL: &url.URL{}},
					},
					err: nil,
				},
				{
					res: &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":0},\"data\":[]}")), // type is io.ReadCloser,
						Request:    &http.Request{Method: "GET", URL: &url.URL{}},
					},
					err: nil,
				},
				{
					res: &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":0},\"data\":[]}")), // type is io.ReadCloser,
						Request:    &http.Request{Method: "GET", URL: &url.URL{}},
					},
					err: nil,
				},
				{
					res: &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":0},\"data\":[]}")), // type is io.ReadCloser,
						Request:    &http.Request{Method: "GET", URL: &url.URL{}},
					},
					err: nil,
				},
				{
					res: &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader("{\"meta\":{\"count\":1},\"data\":[{\"id\":\"1\",\"name\":\"openshift\"}]}")), // type is io.ReadCloser,
						Request:    &http.Request{Method: "GET", URL: &url.URL{}},
					},
					err: nil,
				},
				{
					res: &http.Response{
						StatusCode: 201,
						Body:       io.NopCloser(strings.NewReader("{\"id\":\"11\",\"name\":\"testSource01\",\"source_ref\":\"12345\",\"source_type_id\":\"1\",\"uid\":\"abcdef\"}")), // type is io.ReadCloser,
						Request:    &http.Request{Method: "POST", URL: &url.URL{}},
					},
					err: nil,
				},
				{
					res: &http.Response{
						StatusCode: 201,
						Body:       io.NopCloser(strings.NewReader("{\"created_at\":\"2020-11-20T21:37:27Z\",\"id\":\"18292\"}")), // type is io.ReadCloser,
						Request:    &http.Request{Method: "POST", URL: &url.URL{}},
					},
					err: nil,
				},
			}},
			create:      true,
			want:        true,
			expectedErr: nil,
		},
	}
	for _, tt := range sourceGetOrCreateTests {
		t.Run(tt.name, func(t *testing.T) {
			c := *handler
			c.Spec.CreateSource = &tt.create
			got, _, err := SourceGetOrCreate(&c, &tt.clts)
			if tt.expectedErr != nil && err == nil {
				t.Errorf("%s expected error, got: %v", tt.name, err)
			}
			if tt.expectedErr == nil && err != nil {
				t.Errorf("%s got unexpected error: %v", tt.name, err)
			}
			if tt.want != got {
				t.Errorf("%s got %v want %v", tt.name, got, tt.want)
			}
		})
	}
}
