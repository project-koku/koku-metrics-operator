//
// Copyright 2021 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package collector

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	kokumetricscfgv1beta1 "github.com/project-koku/koku-metrics-operator/api/v1beta1"
	"github.com/project-koku/koku-metrics-operator/testutils"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
)

var trueDef = true
var defaultContextTimeout int64 = 90

type mappedMockPromResult map[string]*mockPromResult
type mockPromResult struct {
	value    model.Value
	warnings promv1.Warnings
	err      error
}
type mockPrometheusConnection struct {
	mappedResults *mappedMockPromResult
	singleResult  *mockPromResult
	t             *testing.T
}

func (m mockPrometheusConnection) QueryRange(ctx context.Context, query string, r promv1.Range) (model.Value, promv1.Warnings, error) {
	var res *mockPromResult
	var ok bool
	if m.mappedResults != nil {
		res, ok = (*m.mappedResults)[query]
		if !ok {
			m.t.Fatalf("Could not find test result!")
		}
	} else if m.singleResult != nil {
		res = m.singleResult
	} else {
		m.t.Fatalf("Could not find test result!")
	}
	return res.value, res.warnings, res.err
}

func (m mockPrometheusConnection) Query(ctx context.Context, query string, ts time.Time) (model.Value, promv1.Warnings, error) {
	res := m.singleResult
	return res.value, res.warnings, res.err
}

func TestGetQueryResultsSuccess(t *testing.T) {
	col := PromCollector{
		TimeSeries: &promv1.Range{},
		Log:        testLogger,
	}
	getQueryResultsErrorsTests := []struct {
		name          string
		queries       *querys
		queriesResult mappedMockPromResult
		wantedResult  mappedResults
		wantedError   error
	}{
		{
			name: "get query results no errors",
			queries: &querys{
				query{
					Name:        "usage-cpu-cores",
					QueryString: "query1",
					MetricKey:   staticFields{"id": "id"},
					QueryValue: &saveQueryValue{
						ValName:         "usage-cpu-cores",
						Method:          "max",
						Factor:          maxFactor,
						TransformedName: "usage-cpu-core-seconds",
					},
					RowKey: []model.LabelName{"id"},
				},
				query{
					Name:        "capacity-cpu-cores",
					QueryString: "query2",
					MetricKey:   staticFields{"id": "id"},
					QueryValue: &saveQueryValue{
						ValName:         "capacity-cpu-cores",
						Method:          "max",
						Factor:          maxFactor,
						TransformedName: "capacity-cpu-core-seconds",
					},
					RowKey: []model.LabelName{"id"},
				},
				query{
					Name:           "labels",
					QueryString:    "query3",
					MetricKeyRegex: regexFields{"labels": "label_*"},
					RowKey:         []model.LabelName{"id"},
				},
			},
			queriesResult: mappedMockPromResult{
				"query1": &mockPromResult{
					value: model.Matrix{
						{
							Metric: model.Metric{
								"id":           "1",
								"random-field": "42",
							},
							Values: []model.SamplePair{
								{Timestamp: 1604339340, Value: 2},
								{Timestamp: 1604339400, Value: 2},
								{Timestamp: 1604339460, Value: 2},
							},
						}},
					warnings: nil,
					err:      nil,
				},
				"query2": &mockPromResult{
					value: model.Matrix{
						{
							Metric: model.Metric{"id": "1"},
							Values: []model.SamplePair{
								{Timestamp: 1604339340, Value: 4},
								{Timestamp: 1604339400, Value: 4},
								{Timestamp: 1604339460, Value: 4},
							},
						},
					},
					warnings: nil,
					err:      nil,
				},
				"query3": &mockPromResult{
					value: model.Matrix{
						{
							Metric: model.Metric{
								"id":            "1",
								"label_arch":    "amd64",
								"label_io_zone": "us-east-2c",
							},
							Values: []model.SamplePair{
								{Timestamp: 1604339340, Value: 1},
								{Timestamp: 1604339400, Value: 1},
								{Timestamp: 1604339460, Value: 1},
							},
						},
					},
					warnings: nil,
					err:      nil,
				},
			},
			wantedResult: mappedResults{
				"1": {
					"id":                        "1",
					"usage-cpu-cores":           "2.000000",
					"usage-cpu-core-seconds":    "360.000000",
					"capacity-cpu-cores":        "4.000000",
					"capacity-cpu-core-seconds": "720.000000",
					"labels":                    "label_arch:amd64|label_io_zone:us-east-2c",
				},
			},
			wantedError: nil,
		},
	}
	for _, tt := range getQueryResultsErrorsTests {
		t.Run(tt.name, func(t *testing.T) {
			col.PromConn = mockPrometheusConnection{
				mappedResults: &tt.queriesResult,
				t:             t,
			}
			got := mappedResults{}
			err := col.getQueryResults(tt.queries, &got)
			if tt.wantedError == nil && err != nil {
				t.Errorf("got unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.wantedResult) {
				t.Errorf("getQueryResults got:\n\t%s\n  want:\n\t%s", got, tt.wantedResult)
			}
		})
	}
}

func TestGetQueryResultsError(t *testing.T) {
	col := PromCollector{
		ContextTimeout: &defaultContextTimeout,
		TimeSeries:     &promv1.Range{},
		Log:            testLogger,
	}
	getQueryResultsErrorsTests := []struct {
		name         string
		queryResult  *mockPromResult
		wantedResult mappedResults
		wantedError  error
	}{
		{
			name:         "return incorrect type (model.Scalar)",
			queryResult:  &mockPromResult{value: &model.Scalar{}},
			wantedResult: mappedResults{},
			wantedError:  errTest,
		},
		{
			name:         "return incorrect type (model.Vector)",
			queryResult:  &mockPromResult{value: &model.Vector{}},
			wantedResult: mappedResults{},
			wantedError:  errTest,
		},
		{
			name:         "return incorrect type (model.String)",
			queryResult:  &mockPromResult{value: &model.String{}},
			wantedResult: mappedResults{},
			wantedError:  errTest,
		},
		{
			name: "warnings with no error",
			queryResult: &mockPromResult{
				value:    model.Matrix{},
				warnings: promv1.Warnings{"This is a warning."},
				err:      nil,
			},
			wantedResult: mappedResults{},
			wantedError:  nil,
		},
		{
			name: "error with no warnings",
			queryResult: &mockPromResult{
				value:    model.Matrix{},
				warnings: nil,
				err:      errTest,
			},
			wantedResult: mappedResults{},
			wantedError:  errTest,
		},
		{
			name: "error with warnings",
			queryResult: &mockPromResult{
				value:    model.Matrix{},
				warnings: promv1.Warnings{"This is another warning."},
				err:      errTest,
			},
			wantedResult: mappedResults{},
			wantedError:  errTest,
		},
	}
	for _, tt := range getQueryResultsErrorsTests {
		t.Run(tt.name, func(t *testing.T) {
			col.PromConn = mockPrometheusConnection{
				singleResult: tt.queryResult,
				t:            t,
			}
			got := mappedResults{}
			err := col.getQueryResults(&querys{query{QueryString: "fake-query"}}, &got)
			if tt.wantedError != nil && err == nil {
				t.Errorf("%s got: nil error, want: error", tt.name)
			}
			if !reflect.DeepEqual(got, tt.wantedResult) {
				t.Errorf("%s got: %s want: %s", tt.name, got, tt.wantedResult)
			}
		})
	}
}

func TestTestPrometheusConnection(t *testing.T) {
	col := PromCollector{
		TimeSeries: &promv1.Range{},
		Log:        testLogger,
	}
	testPrometheusConnectionTests := []struct {
		name        string
		wait        time.Duration
		queryResult *mockPromResult
		wantedError error
	}{
		{
			name:        "test query success",
			queryResult: &mockPromResult{err: nil},
			wantedError: nil,
		},
		{
			name:        "test query error",
			queryResult: &mockPromResult{err: errTest},
			wantedError: errTest,
		},
	}
	for _, tt := range testPrometheusConnectionTests {
		t.Run(tt.name, func(t *testing.T) {
			col.PromConn = mockPrometheusConnection{
				singleResult: tt.queryResult,
				t:            t,
			}
			err := testPrometheusConnection(col.PromConn)
			if tt.wantedError == nil && err != nil {
				t.Errorf("%s got unexpected error: %v", tt.name, err)
			}
			if tt.wantedError != nil && err == nil {
				t.Errorf("%s got: %v error, want: error", tt.name, err)
			}
		})
	}
}

func TestStatusHelper(t *testing.T) {
	statusHelperTests := []struct {
		name   string
		kmCfg  *kokumetricscfgv1beta1.KokuMetricsConfig
		status string
		want   bool
		err    error
	}{
		{
			name:   "config success",
			kmCfg:  &kokumetricscfgv1beta1.KokuMetricsConfig{},
			status: "configuration",
			want:   true,
			err:    nil,
		},
		{
			name:   "config failed",
			kmCfg:  &kokumetricscfgv1beta1.KokuMetricsConfig{},
			status: "configuration",
			want:   false,
			err:    errTest,
		},
		{
			name:   "connection success",
			kmCfg:  &kokumetricscfgv1beta1.KokuMetricsConfig{},
			status: "connection",
			want:   true,
			err:    nil,
		},
		{
			name:   "connection failed",
			kmCfg:  &kokumetricscfgv1beta1.KokuMetricsConfig{},
			status: "connection",
			want:   false,
			err:    errTest,
		},
	}
	for _, tt := range statusHelperTests {
		t.Run(tt.name, func(t *testing.T) {
			statusHelper(tt.kmCfg, tt.status, tt.err)
			var gotMsg string
			var gotBool bool
			switch tt.status {
			case "configuration":
				gotMsg = tt.kmCfg.Status.Prometheus.ConfigError
				gotBool = tt.kmCfg.Status.Prometheus.PrometheusConfigured
			case "connection":
				gotMsg = tt.kmCfg.Status.Prometheus.ConnectionError
				gotBool = tt.kmCfg.Status.Prometheus.PrometheusConnected
			}
			if tt.err != nil && gotMsg == "" {
				t.Errorf("%s got '' want %v", tt.name, tt.err)
			}
			if tt.err == nil && gotMsg != "" {
				t.Errorf("%s got %s want %v", tt.name, gotMsg, tt.err)
			}
			if tt.want != gotBool {
				t.Errorf("%s got %t want %t", tt.name, gotBool, tt.want)
			}
		})
	}
}

func TestGetPrometheusConnFromCfg(t *testing.T) {
	getPrometheusConnFromCfgTests := []struct {
		name        string
		cfg         *PrometheusConfig
		wantedError error
	}{
		{
			name: "wrong cert file config",
			cfg: &PrometheusConfig{
				CAFile: "not-a-real-file",
			},
			wantedError: errTest,
		},
		{
			name: "wrong svc address config",
			cfg: &PrometheusConfig{
				Address: "%gh&%ij", // this causes a url Parse error in promapi.NewClient
			},
			wantedError: errTest,
		},
		{
			name:        "empty config returns no errors",
			cfg:         &PrometheusConfig{},
			wantedError: nil,
		},
	}
	for _, tt := range getPrometheusConnFromCfgTests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := getPrometheusConnFromCfg(tt.cfg)
			if tt.wantedError == nil && err != nil {
				t.Errorf("%s got unexpected error: %v", tt.name, err)
			}
			if tt.wantedError != nil && err == nil {
				t.Errorf("%s expected error, got %v", tt.name, err)
			}
		})
	}
}

func TestGetPromConn(t *testing.T) {
	getPromConnTests := []struct {
		name         string
		cfg          *PrometheusConfig
		createTokCrt bool
		cfgErr       string
		con          prometheusConnection
		conErr       string
		wantedError  error
	}{
		{
			name:        "nil promconn: return getConn error",
			cfg:         &PrometheusConfig{Address: "%gh&%ij"},
			cfgErr:      "",
			con:         nil,
			conErr:      "",
			wantedError: errTest,
		},
		{
			name:        "not empty ConfigError: return getConn error",
			cfg:         &PrometheusConfig{Address: "%gh&%ij"},
			cfgErr:      "error",
			con:         nil,
			conErr:      "",
			wantedError: errTest,
		},
		{
			name:         "not empty ConfigError: reconfig and succeed",
			cfg:          &PrometheusConfig{Address: "%gh&%ij"},
			createTokCrt: true,
			cfgErr:       "error",
			con: &mockPrometheusConnection{
				singleResult: &mockPromResult{err: nil},
			},
			conErr:      "",
			wantedError: nil,
		},
		{
			name:        "not empty ConnectionError: return getConn error",
			cfg:         &PrometheusConfig{Address: "%gh&%ij"},
			cfgErr:      "",
			con:         &mockPrometheusConnection{},
			conErr:      "not empty",
			wantedError: errTest,
		},
		{
			name:   "return getConn successed, test con fails",
			cfg:    &PrometheusConfig{Address: "%gh&%ij"},
			cfgErr: "",
			con: &mockPrometheusConnection{
				singleResult: &mockPromResult{err: errTest},
			},
			conErr:      "",
			wantedError: errTest,
		},
		{
			name:   "return getConn successed, test con succeeds",
			cfg:    &PrometheusConfig{Address: "%gh&%ij"},
			cfgErr: "",
			con: &mockPrometheusConnection{
				singleResult: &mockPromResult{err: nil},
			},
			conErr:      "",
			wantedError: nil,
		},
	}
	for _, tt := range getPromConnTests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.createTokCrt {
				tmpBase := serviceaccountPath
				serviceaccountPath = "./test_files/test_secrets"
				cert := testutils.CreateCertificate(serviceaccountPath, certKey)
				toke := testutils.CreateToken(serviceaccountPath, tokenKey)
				defer func() {
					serviceaccountPath = tmpBase
					os.Remove(cert)
					os.Remove(toke)
				}()
			}
			kmCfg := &kokumetricscfgv1beta1.KokuMetricsConfig{}
			kmCfg.Status.Prometheus.ConfigError = tt.cfgErr
			kmCfg.Status.Prometheus.ConnectionError = tt.conErr
			kmCfg.Spec.PrometheusConfig.SkipTLSVerification = &trueDef
			col := &PromCollector{
				PromConn: tt.con,
				PromCfg:  tt.cfg,
				Log:      testLogger,
			}
			promSpec = kmCfg.Spec.PrometheusConfig.DeepCopy()
			err := col.GetPromConn(kmCfg)
			if tt.wantedError == nil && err != nil {
				t.Errorf("%s got unexpected error: %v", tt.name, err)
			}
			if tt.wantedError != nil && err == nil {
				t.Errorf("%s expected error, got %v", tt.name, err)
			}
		})
	}
}

func TestGetPrometheusConfig(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}
	trueDef := true
	kmCfg := &kokumetricscfgv1beta1.PrometheusSpec{
		SvcAddress:          "svc-address",
		SkipTLSVerification: &trueDef,
	}
	secretsPath := "./test_files/test_secrets"
	getPromCfgTests := []struct {
		name        string
		inCluster   bool
		basePath    string
		certKey     bool
		tokenKey    bool
		want        *PrometheusConfig
		wantedError error
	}{
		{
			name:      "successful config - in cluster",
			inCluster: true,
			basePath:  secretsPath,
			certKey:   true,
			tokenKey:  true,
			want: &PrometheusConfig{
				Address:     "svc-address",
				SkipTLS:     true,
				BearerToken: config.Secret([]byte("this-is-token-data")),
				CAFile:      filepath.Join(secretsPath, certKey),
			},
			wantedError: nil,
		},
		{
			name:        "missing token - in cluster",
			inCluster:   true,
			basePath:    secretsPath,
			certKey:     true,
			tokenKey:    false,
			want:        nil,
			wantedError: errTest,
		},
		{
			name:      "successful config - local",
			inCluster: false,
			basePath:  secretsPath,
			certKey:   true,
			tokenKey:  true,
			want: &PrometheusConfig{
				Address:     "svc-address",
				SkipTLS:     true,
				BearerToken: config.Secret([]byte("this-is-token-data")),
				CAFile:      filepath.Join(cwd, secretsPath, certKey),
			},
			wantedError: nil,
		},
		{
			name:        "missing token - local",
			inCluster:   false,
			basePath:    secretsPath,
			certKey:     true,
			tokenKey:    false,
			want:        nil,
			wantedError: errTest,
		},
		{
			name:        "local - no path to secrets",
			inCluster:   false,
			basePath:    "",
			tokenKey:    false,
			want:        nil,
			wantedError: errTest,
		},
	}
	for _, tt := range getPromCfgTests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.certKey {
				testutils.CreateCertificate(secretsPath, certKey)
			}
			if tt.tokenKey {
				testutils.CreateToken(secretsPath, tokenKey)
			}
			tmpBase := serviceaccountPath
			if tt.inCluster {
				serviceaccountPath = tt.basePath
			} else {
				if err := os.Setenv("SECRET_ABSPATH", filepath.Join(cwd, tt.basePath)); err != nil {
					t.Fatalf("failed to set SECRET_ABSPATH variable")
				}
			}
			defer func() {
				serviceaccountPath = tmpBase
				os.Unsetenv("SECRET_ABSPATH")
				os.Remove(filepath.Join(tt.basePath, "token"))
				os.Remove(filepath.Join(tt.basePath, "service-ca.crt"))
			}()
			got, err := getPrometheusConfig(kmCfg, tt.inCluster)
			if tt.wantedError == nil && err != nil {
				t.Errorf("%s got unexpected error: %v", tt.name, err)
			}
			if tt.wantedError != nil && err == nil {
				t.Errorf("%s expected error, got %v", tt.name, err)
			}
			if got != nil && !reflect.DeepEqual(*got, *tt.want) {
				t.Errorf("%s got %+v want %+v", tt.name, got, tt.want)
			}
			if got == nil && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("%s got %+v want %+v", tt.name, got, tt.want)
			}
		})
	}
}
