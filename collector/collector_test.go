package collector

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"reflect"
	"testing"
	"time"

	costmgmtv1alpha1 "github.com/project-koku/korekuta-operator-go/api/v1alpha1"
	"github.com/project-koku/korekuta-operator-go/dirconfig"
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

// Unmarshal is a function that unmarshals the data from the
// reader into the specified value.
var Unmarshal = func(r io.Reader, v interface{}) error {
	return json.NewDecoder(r).Decode(v)
}

// Load loads the file at path into v.
func Load(path string, v interface{}, t *testing.T) {
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}
	defer f.Close()
	if err := Unmarshal(f, v); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
}

var (
	fakeCost   = &costmgmtv1alpha1.CostManagement{}
	fakeDirCfg = &dirconfig.DirectoryConfig{
		Parent:  dirconfig.Directory{Path: "."},
		Upload:  dirconfig.Directory{Path: "./upload"},
		Staging: dirconfig.Directory{Path: "./expected_reports"},
		Reports: dirconfig.Directory{Path: "./test_reports"},
	}
	t, _          = time.Parse(time.RFC3339, "2020-11-06T16:43:23Z")
	fakeTimeRange = promv1.Range{
		Start: time.Date(t.Year(), t.Month(), t.Day(), t.Hour()-1, 0, 0, 0, t.Location()),
		End:   time.Date(t.Year(), t.Month(), t.Day(), t.Hour()-1, 59, 59, 0, t.Location()),
		Step:  time.Minute,
	}
)

type FakeCollector struct {
	Collector
	results map[*querys]mappedResults
	err     error
	t       *testing.T
}

func NewFakeCollector(c Collector) *PromCollector {
	return &PromCollector{collector: c}
}

func (fc *FakeCollector) getQueryResults(queries *querys) (mappedResults, error) {
	if fc.err != nil {
		return nil, fc.err
	}
	res, ok := fc.results[queries]
	if !ok {
		fc.t.Fatalf("FakeCollector: getQueryResults: failed to find query result")
	}
	return res, nil
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

func (m mockPrometheusConnection) Query(ctx context.Context, query string, ts time.Time) (model.Value, promv1.Warnings, error) {
	return nil, nil, nil
}

func TestLoadFile(t *testing.T) {
	fakeResults := make(map[*querys]mappedResults)
	resFileMap := map[*querys]string{
		namespaceQueries: "test_data/namespace-results.data",
		nodeQueries:      "test_data/node-results.data",
		podQueries:       "test_data/pod-results.data",
		volQueries:       "test_data/vol-results.data",
	}
	for q, s := range resFileMap {
		res := &mappedResults{}
		Load(s, res, t)
		fakeResults[q] = *res
	}

	fc := NewFakeCollector(
		&FakeCollector{
			results: fakeResults,
			t:       t,
			err:     nil,
		})
	fc.TimeSeries = &fakeTimeRange
	fc.Log = zap.New()

	if err := GenerateReports(fakeCost, fakeDirCfg, fc); err != nil {
		t.Errorf("Generating reports failed with err: %v", err)
	}
	fakeDirCfg.Reports.RemoveContents()
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
		query saveQueryValue
		array []model.SamplePair
		want  float64
	}{
		{
			name:  "sum",
			query: saveQueryValue{Method: "sum"},
			array: []model.SamplePair{{Value: 1.3}, {Value: 2.3}, {Value: 3.3}},
			want:  6.9,
		},
		{
			name:  "sum inf",
			query: saveQueryValue{Method: "sum"},
			array: []model.SamplePair{{Value: model.SampleValue(math.Inf(1))}, {Value: 2.3}, {Value: 3.3}},
			want:  math.Inf(1),
		},
		{
			name:  "max",
			query: saveQueryValue{Method: "max"},
			array: []model.SamplePair{{Value: 1.3}, {Value: 2.3}, {Value: 3.3}},
			want:  3.3,
		},
		{
			name:  "max inf",
			query: saveQueryValue{Method: "max"},
			array: []model.SamplePair{{Value: model.SampleValue(math.Inf(1))}, {Value: 2.3}, {Value: 3.3}},
			want:  math.Inf(1),
		},
		{
			name:  "unknown",
			query: saveQueryValue{Method: "unknown"},
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

func TestFindFields(t *testing.T) {
	findFieldsTests := []struct {
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
	for _, tt := range findFieldsTests {
		t.Run(tt.name, func(t *testing.T) {
			got := findFields(tt.input, tt.str)
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
		query   query
		matrix  model.Matrix
		results mappedResults
		want    mappedResults
	}{
		{
			name: "non-regex query",
			query: query{
				Name:        "node-allocatable-cpu-cores",
				QueryString: "kube_node_status_allocatable_cpu_cores * on(node) group_left(provider_id) max(kube_node_info) by (node, provider_id)",
				MetricKey:   staticFields{"node": "node", "provider_id": "provider_id"},
				QueryValue: &saveQueryValue{
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
			query: query{
				Name:           "node-labels",
				QueryString:    "kube_node_labels",
				MetricKeyRegex: regexFields{"node_labels": "label_*"},
				RowKey:         "node",
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
			query: query{
				Name:        "node-capacity-cpu-cores",
				QueryString: "kube_node_status_capacity_cpu_cores * on(node) group_left(provider_id) max(kube_node_info) by (node, provider_id)",
				MetricKey:   staticFields{"node-renamed": "node", "provider-id-renamed": "provider_id"},
				QueryValue: &saveQueryValue{
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
			tt.results.iterateMatrix(tt.matrix, tt.query)
			eq := reflect.DeepEqual(tt.results, tt.want)
			if !eq {
				t.Errorf("%s got:\n\t%s\n  want:\n\t%s", tt.name, tt.results, tt.want)
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
	fakeCollector := PromCollector{
		PromConn: mockPrometheusConnection{
			mappedResults: mapResults,
			t:             t,
		},
		TimeSeries: &promv1.Range{},
		Log:        zap.New(),
	}
	queries := &querys{
		query{
			Name:        "node-allocatable-cpu-cores",
			QueryString: "kube_node_status_allocatable_cpu_cores * on(node) group_left(provider_id) max(kube_node_info) by (node, provider_id)",
			MetricKey:   staticFields{"node": "node", "provider_id": "provider_id"},
			QueryValue: &saveQueryValue{
				ValName:         "node-allocatable-cpu-cores",
				Method:          "max",
				Factor:          maxFactor,
				TransformedName: "node-allocatable-cpu-core-seconds",
			},
			RowKey: "node",
		},
		query{
			Name:        "node-capacity-cpu-cores",
			QueryString: "kube_node_status_capacity_cpu_cores * on(node) group_left(provider_id) max(kube_node_info) by (node, provider_id)",
			MetricKey:   staticFields{"node": "node", "provider_id": "provider_id"},
			QueryValue: &saveQueryValue{
				ValName:         "node-capacity-cpu-cores",
				Method:          "max",
				Factor:          maxFactor,
				TransformedName: "node-capacity-cpu-core-seconds",
			},
			RowKey: "node",
		},
		query{
			Name:           "node-labels",
			QueryString:    "kube_node_labels",
			MetricKeyRegex: regexFields{"node_labels": "label_*"},
			RowKey:         "node",
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
	got, _ := fakeCollector.getQueryResults(queries)
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
	fakeCollector := PromCollector{
		PromConn: mockPrometheusConnection{
			mappedResults: mapResults,
			t:             t,
		},
		TimeSeries: &promv1.Range{},
		Log:        zap.New(),
	}
	getQueryResultsErrorsTests := []struct {
		name         string
		collector    PromCollector
		queries      *querys
		wantedResult mappedResults
		wantedError  error
	}{
		{
			name:      "warnings with no error",
			collector: fakeCollector,
			queries: &querys{
				query{
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
			queries: &querys{
				query{
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
			queries: &querys{
				query{
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
			got, err := tt.collector.getQueryResults(tt.queries)
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
