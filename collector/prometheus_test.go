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

package collector

import (
	"context"
	"reflect"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	kokumetricscfgv1alpha1 "github.com/project-koku/koku-metrics-operator/api/v1alpha1"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	"github.com/xorcare/pointer"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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
					RowKey: "id",
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
					RowKey: "id",
				},
				query{
					Name:           "labels",
					QueryString:    "query3",
					MetricKeyRegex: regexFields{"labels": "label_*"},
					RowKey:         "id",
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
		TimeSeries: &promv1.Range{},
		Log:        testLogger,
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
		kmCfg  *kokumetricscfgv1alpha1.KokuMetricsConfig
		status string
		want   bool
		err    error
	}{
		{
			name:   "config success",
			kmCfg:  &kokumetricscfgv1alpha1.KokuMetricsConfig{},
			status: "configuration",
			want:   true,
			err:    nil,
		},
		{
			name:   "config failed",
			kmCfg:  &kokumetricscfgv1alpha1.KokuMetricsConfig{},
			status: "configuration",
			want:   false,
			err:    errTest,
		},
		{
			name:   "connection success",
			kmCfg:  &kokumetricscfgv1alpha1.KokuMetricsConfig{},
			status: "connection",
			want:   true,
			err:    nil,
		},
		{
			name:   "connection failed",
			kmCfg:  &kokumetricscfgv1alpha1.KokuMetricsConfig{},
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
		name        string
		cfg         *PrometheusConfig
		cfgErr      string
		con         prometheusConnection
		conErr      string
		wantedError error
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
			kmCfg := &kokumetricscfgv1alpha1.KokuMetricsConfig{}
			kmCfg.Status.Prometheus.ConfigError = tt.cfgErr
			kmCfg.Status.Prometheus.ConnectionError = tt.conErr
			kmCfg.Spec.PrometheusConfig.SkipTLSVerification = pointer.Bool(true)
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

var _ = Describe("Collector Tests", func() {

	BeforeEach(func() {
		// failed test runs that don't clean up leave resources behind.
		// &corev1.Pod{}, client.InNamespace("foo")
		Expect(k8sClient.DeleteAllOf(ctx, &corev1.ServiceAccount{}, client.InNamespace(kokuMetricsCfgNamespace))).Should(Succeed())
	})

	AfterEach(func() {

	})

	Describe("Get Bearer Token", func() {

		It("should not find service account", func() {
			createNamespace(kokuMetricsCfgNamespace)
			result, err := getBearerToken(k8sClient)
			Expect(err).Should(HaveOccurred())
			Expect(result).Should(BeEmpty())
		})
		It("should get the service account but not find secret", func() {
			createServiceAccount(kokuMetricsCfgNamespace, serviceAccountName, testSecretData)
			result, err := getBearerToken(k8sClient)
			Expect(err).Should(HaveOccurred())
			Expect(result).Should(BeEmpty())
		})
		It("should find secrets but not the default", func() {
			secrets := createListOfRandomSecrets(5, kokuMetricsCfgNamespace)
			sa := createServiceAccount(kokuMetricsCfgNamespace, serviceAccountName, testSecretData)
			addSecretsToSA(secrets, sa)
			result, err := getBearerToken(k8sClient)
			Expect(err).Should(HaveOccurred())
			Expect(result).Should(BeEmpty())
		})
		It("should get secret but token is not found", func() {
			secrets := createListOfRandomSecrets(5, kokuMetricsCfgNamespace)
			secrets = append(secrets, corev1.ObjectReference{
				Name: createPullSecret(kokuMetricsCfgNamespace, "default-token", "wrong-key", []byte{})})
			sa := createServiceAccount(kokuMetricsCfgNamespace, serviceAccountName, testSecretData)
			addSecretsToSA(secrets, sa)

			result, err := getBearerToken(k8sClient)
			Expect(err).Should(HaveOccurred())
			Expect(result).Should(BeEmpty())
		})
		It("should successfully find token but token is empty", func() {
			secrets := createListOfRandomSecrets(5, kokuMetricsCfgNamespace)
			secrets = append(secrets, corev1.ObjectReference{
				Name: createPullSecret(kokuMetricsCfgNamespace, "default-token", "token", []byte{})})
			sa := createServiceAccount(kokuMetricsCfgNamespace, serviceAccountName, testSecretData)
			addSecretsToSA(secrets, sa)

			result, err := getBearerToken(k8sClient)
			Expect(err).Should(HaveOccurred())
			Expect(result).Should(BeEmpty())
		})
		It("should successfully find token", func() {
			secrets := createListOfRandomSecrets(5, kokuMetricsCfgNamespace)
			secrets = append(secrets, corev1.ObjectReference{
				Name: createPullSecret(kokuMetricsCfgNamespace, "default-token", "token", fakeEncodedData(testSecretData))})
			sa := createServiceAccount(kokuMetricsCfgNamespace, serviceAccountName, testSecretData)
			addSecretsToSA(secrets, sa)

			result, err := getBearerToken(k8sClient)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(result).Should(Equal(config.Secret(fakeEncodedData(testSecretData))))
		})
	})
	Describe("Get Prometheus Config", func() {
		It("could not find bearer token", func() {
			kmCfg := &kokumetricscfgv1alpha1.PrometheusSpec{
				SvcAddress:          "svc-address",
				SkipTLSVerification: pointer.Bool(true),
			}
			result, err := getPrometheusConfig(kmCfg, k8sClient)
			Expect(err).Should(HaveOccurred())
			Expect(result).Should(BeNil())
		})
		It("could find bearer token", func() {
			secrets := createListOfRandomSecrets(5, kokuMetricsCfgNamespace)
			secrets = append(secrets, corev1.ObjectReference{
				Name: createPullSecret(kokuMetricsCfgNamespace, "default-token", "token", fakeEncodedData(testSecretData))})
			sa := createServiceAccount(kokuMetricsCfgNamespace, serviceAccountName, testSecretData)
			addSecretsToSA(secrets, sa)

			kmCfg := &kokumetricscfgv1alpha1.PrometheusSpec{
				SvcAddress:          "svc-address",
				SkipTLSVerification: pointer.Bool(true),
			}
			result, err := getPrometheusConfig(kmCfg, k8sClient)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(result.BearerToken).Should(Equal(config.Secret(fakeEncodedData(testSecretData))))
			Expect(result.Address).Should(Equal("svc-address"))
			Expect(result.SkipTLS).Should(BeTrue())
			Expect(result.CAFile).Should(Equal(certFile))
		})
	})
	Describe("Get Prometheus Connection", func() {
		BeforeEach(func() {
			secrets := createListOfRandomSecrets(5, kokuMetricsCfgNamespace)
			secrets = append(secrets, corev1.ObjectReference{
				Name: createPullSecret(kokuMetricsCfgNamespace, "default-token", "token", fakeEncodedData(testSecretData))})
			sa := createServiceAccount(kokuMetricsCfgNamespace, serviceAccountName, testSecretData)
			addSecretsToSA(secrets, sa)
		})
		It("Prom Spec NOT updated in CR", func() {
			promSpec = nil
			certFile = ""

			col := PromCollector{
				Client: k8sClient,
				Log:    testLogger,
			}
			kmCfg := &kokumetricscfgv1alpha1.KokuMetricsConfig{
				Spec: kokumetricscfgv1alpha1.KokuMetricsConfigSpec{
					PrometheusConfig: kokumetricscfgv1alpha1.PrometheusSpec{
						SvcAddress:          "svc-address",
						SkipTLSVerification: pointer.Bool(true),
					}}}
			err := col.GetPromConn(kmCfg)
			Expect(err).Should(HaveOccurred()) // error occurs because test connection fails
			Expect(col.PromCfg).ToNot(BeNil())
			Expect(col.PromCfg.Address).To(Equal("svc-address"))
			Expect(col.PromConn).ToNot(BeNil())
			Expect(kmCfg.Status.Prometheus.ConnectionError).ToNot(Equal(""))
			Expect(kmCfg.Status.Prometheus.PrometheusConnected).To(BeFalse())

			kmCfg.Spec.PrometheusConfig.SvcAddress = "svc-address"
			err = col.GetPromConn(kmCfg)
			Expect(err).Should(HaveOccurred()) // error occurs because test connection fails
			Expect(col.PromCfg.Address).To(Equal("svc-address"))
		})
		It("Update Prom Spec in CR", func() {
			promSpec = nil
			certFile = ""

			col := PromCollector{
				Client: k8sClient,
				Log:    testLogger,
			}
			kmCfg := &kokumetricscfgv1alpha1.KokuMetricsConfig{
				Spec: kokumetricscfgv1alpha1.KokuMetricsConfigSpec{
					PrometheusConfig: kokumetricscfgv1alpha1.PrometheusSpec{
						SvcAddress:          "svc-address",
						SkipTLSVerification: pointer.Bool(true),
					}}}
			err := col.GetPromConn(kmCfg)
			Expect(err).Should(HaveOccurred()) // error occurs because test connection fails
			Expect(col.PromCfg).ToNot(BeNil())
			Expect(col.PromCfg.Address).To(Equal("svc-address"))
			Expect(col.PromConn).ToNot(BeNil())
			Expect(kmCfg.Status.Prometheus.ConnectionError).ToNot(Equal(""))
			Expect(kmCfg.Status.Prometheus.PrometheusConnected).To(BeFalse())

			kmCfg.Spec.PrometheusConfig.SvcAddress = "new-svc-address"
			err = col.GetPromConn(kmCfg)
			Expect(err).Should(HaveOccurred()) // error occurs because test connection fails
			Expect(col.PromCfg.Address).To(Equal("new-svc-address"))
		})
	})
})
