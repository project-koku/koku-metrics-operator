package collector

import (
	"context"
	"errors"
	"math"
	"reflect"
	"testing"

	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const epsilon = 0.00001

func nearlyEqual(a, b float64) bool {
	absA := math.Abs(a)
	absB := math.Abs(b)
	diff := math.Abs(a - b)

	if a == b { // shortcut, handles infinities
		return true
	} else if a == 0 || b == 0 || (absA+absB < math.SmallestNonzeroFloat64) {
		// a or b is zero or both are extremely close to it
		// relative error is less meaningful here
		return diff < (epsilon * math.MaxFloat64)
	} else { // use relative error
		return diff/math.Min((absA+absB), math.MaxFloat64) < epsilon
	}
}

type mappedMockPromResult map[string]mockPromResult
type mockPromResult struct {
	matrix   model.Matrix
	warnings promv1.Warnings
	err      error
}
type mockPrometheusConnection struct {
	mappedResults mappedMockPromResult
	t             *testing.T
}

func (m mockPrometheusConnection) QueryRange(ctx context.Context, query string, r promv1.Range) (model.Value, promv1.Warnings, error) {
	res, ok := m.mappedResults[query]
	if !ok {
		m.t.Fatalf("Could not find test result!")
	}
	if res.err != nil {
		return nil, nil, res.err
	}
	if res.warnings != nil {
		return res.matrix, res.warnings, nil
	}
	return res.matrix, nil, nil
}

func TestGetResourceID(t *testing.T) {
	getResourceIDTests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "with slashes", input: "gce://openshift-gce-devel/us-west1-a/metering-ci-3-ig-m-91kw", want: "metering-ci-3-ig-m-91kw"},
		{name: "without slashes", input: "metering-ci-3-ig-m-91kw", want: "metering-ci-3-ig-m-91kw"},
	}
	for _, tt := range getResourceIDTests {
		// using tt.name from the case to use it as the `t.Run` test name
		t.Run(tt.name, func(t *testing.T) {
			got := getResourceID(tt.input)
			if got != tt.want {
				t.Errorf("%s got %s want %s", tt.name, got, tt.want)
			}
		})
	}
}

func TestGetValue(t *testing.T) {
	getValueTests := []struct {
		name  string
		query SaveQueryValue
		array []model.SamplePair
		want  float64
	}{
		{
			name:  "sum",
			query: SaveQueryValue{Method: "sum"},
			array: []model.SamplePair{{Value: 1.3}, {Value: 2.3}, {Value: 3.3}},
			want:  6.9,
		},
		{
			name:  "sum inf",
			query: SaveQueryValue{Method: "sum"},
			array: []model.SamplePair{{Value: model.SampleValue(math.Inf(1))}, {Value: 2.3}, {Value: 3.3}},
			want:  math.Inf(1),
		},
		{
			name:  "max",
			query: SaveQueryValue{Method: "max"},
			array: []model.SamplePair{{Value: 1.3}, {Value: 2.3}, {Value: 3.3}},
			want:  3.3,
		},
		{
			name:  "max inf",
			query: SaveQueryValue{Method: "max"},
			array: []model.SamplePair{{Value: model.SampleValue(math.Inf(1))}, {Value: 2.3}, {Value: 3.3}},
			want:  math.Inf(1),
		},
		{
			name:  "unknown",
			query: SaveQueryValue{Method: "unknown"},
			array: []model.SamplePair{{Value: 1.3}, {Value: 2.3}, {Value: 3.3}},
			want:  0,
		},
	}
	for _, tt := range getValueTests {
		// using tt.name from the case to use it as the `t.Run` test name
		t.Run(tt.name, func(t *testing.T) {
			got := getValue(&tt.query, tt.array)
			if !nearlyEqual(got, tt.want) {
				t.Errorf("%s got %f want %f", tt.name, got, tt.want)
			}
		})
	}
}

func TestFloatToString(t *testing.T) {
	floatToStringTests := []struct {
		name  string
		input float64
		want  string
	}{
		{name: "decimal no rounding", input: 0.1234564567, want: "0.123456"},
		{name: "decimal needs rounding", input: 0.1234567890, want: "0.123457"},
		{name: "no decimal", input: 1234567890, want: "1234567890.000000"},
	}
	for _, tt := range floatToStringTests {
		// using tt.name from the case to use it as the `t.Run` test name
		t.Run(tt.name, func(t *testing.T) {
			got := floatToString(tt.input)
			if got != tt.want {
				t.Errorf("%s got %s want %s", tt.name, got, tt.want)
			}
		})
	}
}

func TestParseFields(t *testing.T) {
	parseFieldsTests := []struct {
		name  string
		input model.Metric
		str   string
		want  string
	}{
		{
			name: "no matches",
			input: model.Metric{
				"endpoint":   "https-main",
				"instance":   "10.131.0.11:8443",
				"job":        "kube-state-metrics",
				"namespace":  "openshift-infra",
				"pod":        "kube-state-metrics-b88767d9b-dljtf",
				"prometheus": "openshift-monitoring/k8s",
				"service":    "kube-state-metrics",
			},
			str:  "label_*",
			want: "",
		},
		{
			name: "one match",
			input: model.Metric{
				"endpoint":                              "https-main",
				"instance":                              "10.131.0.11:8443",
				"job":                                   "kube-state-metrics",
				"label_openshift_io_cluster_monitoring": "true",
				"namespace":                             "openshift-image-registry",
				"pod":                                   "kube-state-metrics-b88767d9b-dljtf",
				"prometheus":                            "openshift-monitoring/k8s",
				"service":                               "kube-state-metrics",
			},
			str:  "label_*",
			want: "label_openshift_io_cluster_monitoring:true",
		},
		{
			name: "multiple matches",
			input: model.Metric{
				"endpoint":                              "https-main",
				"instance":                              "10.131.0.11:8443",
				"job":                                   "kube-state-metrics",
				"label_controller_tools_k8s_io":         "1.0",
				"label_openshift_io_cluster_monitoring": "true",
				"namespace":                             "openshift-cloud-credential-operator",
				"pod":                                   "kube-state-metrics-b88767d9b-dljtf",
				"prometheus":                            "openshift-monitoring/k8s",
				"service":                               "kube-state-metrics",
			},
			str:  "label_*",
			want: "label_controller_tools_k8s_io:1.0|label_openshift_io_cluster_monitoring:true",
		},
	}
	for _, tt := range parseFieldsTests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseFields(tt.input, tt.str)
			if got != tt.want {
				t.Errorf("%s got %s want %s", tt.name, got, tt.want)
			}
		})
	}
}

func TestIterateMatrix(t *testing.T) {
	testResult := mappedResults{}
	iterateMatrixTests := []struct {
		name    string
		query   Query
		matrix  model.Matrix
		results mappedResults
		want    mappedResults
	}{
		{
			name: "non-regex query",
			query: Query{
				Name:        "node-allocatable-cpu-cores",
				QueryString: "kube_node_status_allocatable_cpu_cores * on(node) group_left(provider_id) max(kube_node_info) by (node, provider_id)",
				MetricKey:   &StaticFields{MetricLabel: []model.LabelName{"node", "provider_id"}},
				QueryValue: &SaveQueryValue{
					ValName:         "node-allocatable-cpu-cores",
					Method:          "max",
					Factor:          maxFactor,
					TransformedName: "node-allocatable-cpu-core-seconds",
				},
				RowKey: "node",
			},
			matrix: model.Matrix{
				{
					Metric: model.Metric{
						"endpoint":    "https-main",
						"instance":    "10.131.0.11:8443",
						"job":         "kube-state-metrics",
						"namespace":   "openshift-monitoring",
						"node":        "ip-10-0-222-213.us-east-2.compute.internal",
						"pod":         "kube-state-metrics-b88767d9b-dljtf",
						"prometheus":  "openshift-monitoring/k8s",
						"provider_id": "aws:///us-east-2c/i-070043b2c0291bdc2",
						"service":     "kube-state-metrics",
					},
					Values: []model.SamplePair{
						{Timestamp: 1604339340, Value: 4},
						{Timestamp: 1604339400, Value: 4},
						{Timestamp: 1604339460, Value: 4},
					},
				},
			},
			results: testResult,
			want: mappedResults{
				"ip-10-0-222-213.us-east-2.compute.internal": {
					"node":                              "ip-10-0-222-213.us-east-2.compute.internal",
					"provider_id":                       "aws:///us-east-2c/i-070043b2c0291bdc2",
					"node-allocatable-cpu-cores":        "4.000000",
					"node-allocatable-cpu-core-seconds": "720.000000",
				},
			},
		},
		{
			name: "with-regex query",
			query: Query{
				Name:        "node-labels",
				QueryString: "kube_node_labels",
				MetricKeyRegex: &RegexFields{
					MetricRegex: []string{"label_*"},
					LabelMap:    []string{"node_labels"}},
				RowKey: "node",
			},
			matrix: model.Matrix{
				{
					Metric: model.Metric{
						"endpoint":                               "https-main",
						"instance":                               "10.131.0.11:8443",
						"job":                                    "kube-state-metrics",
						"label_beta_kubernetes_io_arch":          "amd64",
						"label_beta_kubernetes_io_instance_type": "m5.xlarge",
						"label_beta_kubernetes_io_os":            "linux",
						"label_failure_domain_beta_kubernetes_io_region": "us-east-2",
						"label_failure_domain_beta_kubernetes_io_zone":   "us-east-2c",
						"label_kubernetes_io_arch":                       "amd64",
						"label_kubernetes_io_hostname":                   "ip-10-0-222-213",
						"label_kubernetes_io_os":                         "linux",
						"label_node_kubernetes_io_instance_type":         "m5.xlarge",
						"label_node_openshift_io_os_id":                  "rhcos",
						"label_topology_kubernetes_io_region":            "us-east-2",
						"label_topology_kubernetes_io_zone":              "us-east-2c",
						"namespace":                                      "openshift-monitoring",
						"node":                                           "ip-10-0-222-213.us-east-2.compute.internal",
						"pod":                                            "kube-state-metrics-b88767d9b-dljtf",
						"prometheus":                                     "openshift-monitoring/k8s",
						"service":                                        "kube-state-metrics",
					},
					Values: []model.SamplePair{
						{Timestamp: 1604339340, Value: 4},
						{Timestamp: 1604339400, Value: 4},
						{Timestamp: 1604339460, Value: 4},
					},
				},
			},
			results: testResult,
			want: mappedResults{
				// since reusing the testResult, this query adds node labels
				"ip-10-0-222-213.us-east-2.compute.internal": {
					"node":                              "ip-10-0-222-213.us-east-2.compute.internal",
					"provider_id":                       "aws:///us-east-2c/i-070043b2c0291bdc2",
					"node-allocatable-cpu-cores":        "4.000000",
					"node-allocatable-cpu-core-seconds": "720.000000",
					"node_labels":                       "label_beta_kubernetes_io_arch:amd64|label_beta_kubernetes_io_instance_type:m5.xlarge|label_beta_kubernetes_io_os:linux|label_failure_domain_beta_kubernetes_io_region:us-east-2|label_failure_domain_beta_kubernetes_io_zone:us-east-2c|label_kubernetes_io_arch:amd64|label_kubernetes_io_hostname:ip-10-0-222-213|label_kubernetes_io_os:linux|label_node_kubernetes_io_instance_type:m5.xlarge|label_node_openshift_io_os_id:rhcos|label_topology_kubernetes_io_region:us-east-2|label_topology_kubernetes_io_zone:us-east-2c",
				},
			},
		},
		{
			name: "static field with label map query",
			query: Query{
				Name:        "node-capacity-cpu-cores",
				QueryString: "kube_node_status_capacity_cpu_cores * on(node) group_left(provider_id) max(kube_node_info) by (node, provider_id)",
				MetricKey: &StaticFields{
					MetricLabel: []model.LabelName{"node", "provider_id"},
					LabelMap:    []string{"node-renamed", "provider-id-renamed"},
				},
				QueryValue: &SaveQueryValue{
					ValName:         "node-capacity-cpu-cores",
					Method:          "max",
					Factor:          maxFactor,
					TransformedName: "node-capacity-cpu-core-seconds",
				},
				RowKey: "node",
			},
			matrix: model.Matrix{
				{
					Metric: model.Metric{
						"endpoint":    "https-main",
						"instance":    "10.131.0.11:8443",
						"job":         "kube-state-metrics",
						"namespace":   "openshift-monitoring",
						"node":        "ip-10-0-222-213.us-east-2.compute.internal",
						"pod":         "kube-state-metrics-b88767d9b-dljtf",
						"prometheus":  "openshift-monitoring/k8s",
						"provider_id": "aws:///us-east-2c/i-070043b2c0291bdc2",
						"service":     "kube-state-metrics",
					},
					Values: []model.SamplePair{
						{Timestamp: 1604339340, Value: 4},
						{Timestamp: 1604339400, Value: 4},
						{Timestamp: 1604339460, Value: 4},
					},
				},
			},
			results: testResult,
			want: mappedResults{
				// since reusing the testResult, this query adds node labels
				"ip-10-0-222-213.us-east-2.compute.internal": {
					"node":                              "ip-10-0-222-213.us-east-2.compute.internal",
					"provider_id":                       "aws:///us-east-2c/i-070043b2c0291bdc2",
					"node-allocatable-cpu-cores":        "4.000000",
					"node-allocatable-cpu-core-seconds": "720.000000",
					"node-capacity-cpu-cores":           "4.000000",
					"node-capacity-cpu-core-seconds":    "720.000000",
					"node-renamed":                      "ip-10-0-222-213.us-east-2.compute.internal",
					"provider-id-renamed":               "aws:///us-east-2c/i-070043b2c0291bdc2",
					"node_labels":                       "label_beta_kubernetes_io_arch:amd64|label_beta_kubernetes_io_instance_type:m5.xlarge|label_beta_kubernetes_io_os:linux|label_failure_domain_beta_kubernetes_io_region:us-east-2|label_failure_domain_beta_kubernetes_io_zone:us-east-2c|label_kubernetes_io_arch:amd64|label_kubernetes_io_hostname:ip-10-0-222-213|label_kubernetes_io_os:linux|label_node_kubernetes_io_instance_type:m5.xlarge|label_node_openshift_io_os_id:rhcos|label_topology_kubernetes_io_region:us-east-2|label_topology_kubernetes_io_zone:us-east-2c",
				},
			},
		},
	}
	for _, tt := range iterateMatrixTests {
		t.Run(tt.name, func(t *testing.T) {
			got := iterateMatrix(tt.matrix, tt.query, tt.results)
			eq := reflect.DeepEqual(got, tt.want)
			if !eq {
				t.Errorf("%s got:\n\t%s\n  want:\n\t%s", tt.name, got, tt.want)
			}
		})
	}
}

func TestGetQueryResults(t *testing.T) {
	mapResults := mappedMockPromResult{
		"kube_node_status_allocatable_cpu_cores * on(node) group_left(provider_id) max(kube_node_info) by (node, provider_id)": mockPromResult{
			matrix: model.Matrix{
				{
					Metric: model.Metric{
						"endpoint":    "https-main",
						"instance":    "10.131.0.11:8443",
						"job":         "kube-state-metrics",
						"namespace":   "openshift-monitoring",
						"node":        "ip-10-0-222-213.us-east-2.compute.internal",
						"pod":         "kube-state-metrics-b88767d9b-dljtf",
						"prometheus":  "openshift-monitoring/k8s",
						"provider_id": "aws:///us-east-2c/i-070043b2c0291bdc2",
						"service":     "kube-state-metrics",
					},
					Values: []model.SamplePair{
						{Timestamp: 1604339340, Value: 4},
						{Timestamp: 1604339400, Value: 4},
						{Timestamp: 1604339460, Value: 4},
					},
				}},
			warnings: nil,
			err:      nil,
		},
		"kube_node_status_capacity_cpu_cores * on(node) group_left(provider_id) max(kube_node_info) by (node, provider_id)": mockPromResult{
			matrix: model.Matrix{
				{
					Metric: model.Metric{
						"endpoint":    "https-main",
						"instance":    "10.131.0.11:8443",
						"job":         "kube-state-metrics",
						"namespace":   "openshift-monitoring",
						"node":        "ip-10-0-222-213.us-east-2.compute.internal",
						"pod":         "kube-state-metrics-b88767d9b-dljtf",
						"prometheus":  "openshift-monitoring/k8s",
						"provider_id": "aws:///us-east-2c/i-070043b2c0291bdc2",
						"service":     "kube-state-metrics",
					},
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
		"kube_node_labels": mockPromResult{
			matrix: model.Matrix{
				{
					Metric: model.Metric{
						"endpoint":                          "https-main",
						"instance":                          "10.131.0.11:8443",
						"job":                               "kube-state-metrics",
						"label_beta_kubernetes_io_arch":     "amd64",
						"label_topology_kubernetes_io_zone": "us-east-2c",
						"namespace":                         "openshift-monitoring",
						"node":                              "ip-10-0-222-213.us-east-2.compute.internal",
						"pod":                               "kube-state-metrics-b88767d9b-dljtf",
						"prometheus":                        "openshift-monitoring/k8s",
						"service":                           "kube-state-metrics",
					},
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
	}
	fakeCollector := collector{
		PromConn: mockPrometheusConnection{
			mappedResults: mapResults,
			t:             t,
		},
		TimeSeries: promv1.Range{},
		Log:        zap.New(),
	}
	queries := Querys{
		Query{
			Name:        "node-allocatable-cpu-cores",
			QueryString: "kube_node_status_allocatable_cpu_cores * on(node) group_left(provider_id) max(kube_node_info) by (node, provider_id)",
			MetricKey:   &StaticFields{MetricLabel: []model.LabelName{"node", "provider_id"}},
			QueryValue: &SaveQueryValue{
				ValName:         "node-allocatable-cpu-cores",
				Method:          "max",
				Factor:          maxFactor,
				TransformedName: "node-allocatable-cpu-core-seconds",
			},
			RowKey: "node",
		},
		Query{
			Name:        "node-capacity-cpu-cores",
			QueryString: "kube_node_status_capacity_cpu_cores * on(node) group_left(provider_id) max(kube_node_info) by (node, provider_id)",
			MetricKey:   &StaticFields{MetricLabel: []model.LabelName{"node", "provider_id"}},
			QueryValue: &SaveQueryValue{
				ValName:         "node-capacity-cpu-cores",
				Method:          "max",
				Factor:          maxFactor,
				TransformedName: "node-capacity-cpu-core-seconds",
			},
			RowKey: "node",
		},
		Query{
			Name:        "node-labels",
			QueryString: "kube_node_labels",
			MetricKeyRegex: &RegexFields{
				MetricRegex: []string{"label_*"},
				LabelMap:    []string{"node_labels"}},
			RowKey: "node",
		},
	}
	want := mappedResults{
		// since reusing the testResult, this query adds node labels
		"ip-10-0-222-213.us-east-2.compute.internal": {
			"node":                              "ip-10-0-222-213.us-east-2.compute.internal",
			"provider_id":                       "aws:///us-east-2c/i-070043b2c0291bdc2",
			"node-allocatable-cpu-cores":        "4.000000",
			"node-allocatable-cpu-core-seconds": "720.000000",
			"node-capacity-cpu-cores":           "4.000000",
			"node-capacity-cpu-core-seconds":    "720.000000",
			"node_labels":                       "label_beta_kubernetes_io_arch:amd64|label_topology_kubernetes_io_zone:us-east-2c",
		},
	}
	got, _ := getQueryResults(fakeCollector, queries)
	eq := reflect.DeepEqual(got, want)
	if !eq {
		t.Errorf("getQueryResults got:\n\t%s\n  want:\n\t%s", got, want)
	}
}

func TestGetQueryResultsError(t *testing.T) {
	mapResults := mappedMockPromResult{
		"kube_node_status_allocatable_cpu_cores * on(node) group_left(provider_id) max(kube_node_info) by (node, provider_id)": mockPromResult{
			matrix:   model.Matrix{},
			warnings: promv1.Warnings{"This is a warning."},
			err:      nil,
		},
		"kube_node_status_capacity_cpu_cores * on(node) group_left(provider_id) max(kube_node_info) by (node, provider_id)": mockPromResult{
			matrix:   model.Matrix{},
			warnings: nil,
			err:      errors.New("this is an error"),
		},
		"kube_node_labels": mockPromResult{
			matrix:   model.Matrix{},
			warnings: promv1.Warnings{"This is another warning."},
			err:      errors.New("this is another error"),
		},
	}
	fakeCollector := collector{
		PromConn: mockPrometheusConnection{
			mappedResults: mapResults,
			t:             t,
		},
		TimeSeries: promv1.Range{},
		Log:        zap.New(),
	}
	getQueryResultsErrorsTests := []struct {
		name         string
		collector    collector
		queries      Querys
		wantedResult mappedResults
		wantedError  error
	}{
		{
			name:      "warnings with no error",
			collector: fakeCollector,
			queries: Querys{
				Query{
					QueryString: "kube_node_status_allocatable_cpu_cores * on(node) group_left(provider_id) max(kube_node_info) by (node, provider_id)",
					RowKey:      "node",
				},
			},
			wantedResult: mappedResults{},
			wantedError:  nil,
		},
		{
			name:      "error with no warnings",
			collector: fakeCollector,
			queries: Querys{
				Query{
					QueryString: "kube_node_status_capacity_cpu_cores * on(node) group_left(provider_id) max(kube_node_info) by (node, provider_id)",
					RowKey:      "node",
				},
			},
			wantedResult: nil,
			wantedError:  errors.New("this is an error"),
		},
		{
			name:      "error with warnings",
			collector: fakeCollector,
			queries: Querys{
				Query{
					QueryString: "kube_node_labels",
					RowKey:      "node",
				},
			},
			wantedResult: nil,
			wantedError:  errors.New("this is another error"),
		},
	}
	for _, tt := range getQueryResultsErrorsTests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getQueryResults(tt.collector, tt.queries)
			if got != nil {
				eq := reflect.DeepEqual(got, tt.wantedResult)
				if !eq {
					t.Errorf("%s got: %s want: %s", tt.name, got, tt.wantedResult)
				}
			}
			if tt.wantedError != nil && err == nil {
				t.Errorf("%s got: nil error, want: error", tt.name)
			}
		})
	}
}
