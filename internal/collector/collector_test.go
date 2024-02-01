package collector

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	metricscfgv1beta1 "github.com/project-koku/koku-metrics-operator/api/v1beta1"
	"github.com/project-koku/koku-metrics-operator/internal/dirconfig"
	"github.com/project-koku/koku-metrics-operator/internal/strset"
	"github.com/project-koku/koku-metrics-operator/internal/testutils"
)

const epsilon = 0.00001

func nearlyEqual(a, b float64) bool {
	absA := math.Abs(a)
	absB := math.Abs(b)
	diff := math.Abs(a - b)

	if a == b || (math.IsNaN(a) && math.IsNaN(b)) { // shortcut, handles infinities
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
	fakeCR = &metricscfgv1beta1.MetricsConfig{Spec: metricscfgv1beta1.MetricsConfigSpec{
		PrometheusConfig: metricscfgv1beta1.PrometheusSpec{
			DisableMetricsCollectionCostManagement:       &falseDef,
			DisableMetricsCollectionResourceOptimization: &falseDef,
		},
	}}
	fakeDirCfg = &dirconfig.DirectoryConfig{
		Parent:  dirconfig.Directory{Path: "."},
		Reports: dirconfig.Directory{Path: "./test_files/test_reports"},
	}
	localTime, _  = time.Parse(time.RFC3339, "2020-11-06T19:43:23Z")
	t             = localTime.UTC()
	fakeTimeRange = promv1.Range{
		Start: time.Date(t.Year(), t.Month(), t.Day(), t.Hour()-1, 0, 0, 0, t.Location()),
		End:   time.Date(t.Year(), t.Month(), t.Day(), t.Hour()-1, 59, 59, 0, t.Location()),
		Step:  time.Minute,
	}
)

func getFiles(dir string, t *testing.T) map[string]*os.File {
	fileMap := make(map[string]*os.File)
	filelist, err := os.ReadDir(filepath.Join("test_files", dir))
	if err != nil {
		t.Fatalf("Failed to read %s directory", dir)
	}
	for _, file := range filelist {
		f, err := os.Open(filepath.Join("test_files", dir, file.Name()))
		if err != nil {
			t.Fatalf("failed to open %s: %v", file.Name(), err)
		}
		fileMap[file.Name()] = f
	}
	return fileMap
}

func compareFiles(expected, generated *os.File) error {
	files := map[string]*os.File{"e": expected, "g": generated}
	sets := map[string]*strset.Set{"e": strset.NewSet(), "g": strset.NewSet()}
	for i, file := range files {
		var err error
		_, err = readCSV(file, sets[i], "")
		if err != nil {
			return fmt.Errorf("failed to readCSV %s: %v", file.Name(), err)
		}
	}

	var b strings.Builder
	b.WriteString("Missing Rows:\n")
	b.WriteString("\tGenerated is missing:\n")
	for val := range sets["e"].Range() {
		if !sets["g"].Contains(val) {
			fmt.Fprintf(&b, "\t\t%s\n", val)
		}
	}
	b.WriteString("\tGenerated has extra rows:\n")
	for val := range sets["g"].Range() {
		if !sets["e"].Contains(val) {
			fmt.Fprintf(&b, "\t\t%s\n", val)
		}
	}
	if b.String() != "Missing Rows:\n\tGenerated is missing:\n\tGenerated has extra rows:\n" {
		return errors.New(b.String())
	}

	return nil
}

func TestMain(m *testing.M) {
	logf.SetLogger(testutils.ZapLogger(true))
	code := m.Run()
	os.Exit(code)
}

func TestGenerateReports(t *testing.T) {
	mapResults := make(mappedMockPromResult)
	queryList := []*querys{nodeQueries, namespaceQueries, podQueries, volQueries}
	for _, q := range queryList {
		for _, query := range *q {
			res := &model.Matrix{}
			Load(filepath.Join("test_files", "test_data", query.Name), res, t)
			mapResults[query.QueryString] = &mockPromResult{value: *res}
		}
	}
	for _, query := range *resourceOptimizationQueries {
		res := &model.Vector{}
		Load(filepath.Join("test_files", "test_data", query.Name), res, t)
		mapResults[query.QueryString] = &mockPromResult{value: *res}
	}

	fakeCollector := &PrometheusCollector{
		PromConn: mockPrometheusConnection{
			mappedResults: &mapResults,
			t:             t,
		},
		TimeSeries: &fakeTimeRange,
	}
	if err := GenerateReports(fakeCR, fakeDirCfg, fakeCollector); err != nil {
		t.Errorf("Failed to generate reports: %v", err)
	}

	// ####### everything below compares the generated reports to the expected reports #######
	expectedMap := getFiles("expected_reports", t)
	generatedMap := getFiles("test_reports", t)

	if len(expectedMap) != len(generatedMap) {
		t.Errorf("incorrect number of reports generated")
	}

	for expected, expectedinfo := range expectedMap {
		generatedinfo, ok := generatedMap[expected]
		if !ok {
			t.Errorf("%s report file was not generated", expected)
		} else {
			if err := compareFiles(expectedinfo, generatedinfo); err != nil {
				t.Errorf("%s files do not compare: error: %v", expected, err)
			}
		}
	}

	if err := fakeDirCfg.Reports.RemoveContents(); err != nil {
		t.Fatal("failed to cleanup reports directory")
	}
}

func TestGenerateReportsNoROS(t *testing.T) {
	mapResults := make(mappedMockPromResult)
	queryList := []*querys{nodeQueries, namespaceQueries, podQueries, volQueries}
	for _, q := range queryList {
		for _, query := range *q {
			res := &model.Matrix{}
			Load(filepath.Join("test_files", "test_data", query.Name), res, t)
			mapResults[query.QueryString] = &mockPromResult{value: *res}
		}
	}
	for _, query := range *resourceOptimizationQueries {
		res := &model.Vector{}
		Load(filepath.Join("test_files", "test_data", query.Name), res, t)
		mapResults[query.QueryString] = &mockPromResult{value: *res}
	}

	fakeCollector := &PrometheusCollector{
		PromConn: mockPrometheusConnection{
			mappedResults: &mapResults,
			t:             t,
		},
		TimeSeries: &fakeTimeRange,
	}
	noRosCR := fakeCR.DeepCopy()
	noRosCR.Spec.PrometheusConfig.DisableMetricsCollectionResourceOptimization = &trueDef
	if err := GenerateReports(noRosCR, fakeDirCfg, fakeCollector); err != nil {
		t.Errorf("Failed to generate reports: %v", err)
	}

	// ####### everything below compares the generated reports to the expected reports #######
	expectedMap := getFiles("expected_reports", t)
	generatedMap := getFiles("test_reports", t)
	expectedDiff := 1 // The expected diff is equal to the number of ROS reports we generate. If we add or remove reports, this number should change

	if len(expectedMap)-len(generatedMap) != expectedDiff {
		t.Errorf("incorrect number of reports generated")
	}
	if err := fakeDirCfg.Reports.RemoveContents(); err != nil {
		t.Fatal("failed to cleanup reports directory")
	}
}

func TestGenerateReportsNoCost(t *testing.T) {
	mapResults := make(mappedMockPromResult)
	queryList := []*querys{nodeQueries, namespaceQueries, podQueries, volQueries}
	for _, q := range queryList {
		for _, query := range *q {
			res := &model.Matrix{}
			Load(filepath.Join("test_files", "test_data", query.Name), res, t)
			mapResults[query.QueryString] = &mockPromResult{value: *res}
		}
	}
	for _, query := range *resourceOptimizationQueries {
		res := &model.Vector{}
		Load(filepath.Join("test_files", "test_data", query.Name), res, t)
		mapResults[query.QueryString] = &mockPromResult{value: *res}
	}

	fakeCollector := &PrometheusCollector{
		PromConn: mockPrometheusConnection{
			mappedResults: &mapResults,
			t:             t,
		},
		TimeSeries: &fakeTimeRange,
	}
	noCostCR := fakeCR.DeepCopy()
	noCostCR.Spec.PrometheusConfig.DisableMetricsCollectionCostManagement = &trueDef
	if err := GenerateReports(noCostCR, fakeDirCfg, fakeCollector); err != nil {
		t.Errorf("Failed to generate reports: %v", err)
	}

	// ####### everything below compares the generated reports to the expected reports #######
	expectedMap := getFiles("expected_reports", t)
	generatedMap := getFiles("test_reports", t)
	expectedDiff := 4 // The expected diff is equal to the number of ROS reports we generate. If we add or remove reports, this number should change

	if len(expectedMap)-len(generatedMap) != expectedDiff {
		t.Errorf("incorrect number of reports generated")
	}
	if err := fakeDirCfg.Reports.RemoveContents(); err != nil {
		t.Fatal("failed to cleanup reports directory")
	}
}

func TestGenerateReportsQueryErrors(t *testing.T) {
	MaxRetries = 1
	mapResults := make(mappedMockPromResult)
	fakeCollector := &PrometheusCollector{
		PromConn: mockPrometheusConnection{
			mappedResults: &mapResults,
			t:             t,
		},
		TimeSeries: &fakeTimeRange,
	}

	queryList := []*querys{nodeQueries, podQueries, volQueries, namespaceQueries}
	for _, q := range queryList {
		for _, query := range *q {
			res := &model.Matrix{}
			Load(filepath.Join("test_files", "test_data", query.Name), res, t)
			mapResults[query.QueryString] = &mockPromResult{value: *res}
		}
	}
	for _, query := range *resourceOptimizationQueries {
		res := &model.Vector{}
		Load(filepath.Join("test_files", "test_data", query.Name), res, t)
		mapResults[query.QueryString] = &mockPromResult{value: *res}
	}

	resourceOptimizationError := "resourceOptimization error"
	for _, q := range *resourceOptimizationQueries {
		mapResults[q.QueryString] = &mockPromResult{err: errors.New(resourceOptimizationError)}
	}
	err := GenerateReports(fakeCR, fakeDirCfg, fakeCollector)
	if !strings.Contains(err.Error(), resourceOptimizationError) {
		t.Errorf("GenerateReports %s was expected, got %v", resourceOptimizationError, err)
	}

	namespaceError := "namespace error"
	for _, q := range *namespaceQueries {
		mapResults[q.QueryString] = &mockPromResult{err: errors.New(namespaceError)}
	}
	err = GenerateReports(fakeCR, fakeDirCfg, fakeCollector)
	if !strings.Contains(err.Error(), namespaceError) {
		t.Errorf("GenerateReports %s was expected, got %v", namespaceError, err)
	}
	storageError := "storage error"
	for _, q := range *volQueries {
		mapResults[q.QueryString] = &mockPromResult{err: errors.New(storageError)}
	}
	err = GenerateReports(fakeCR, fakeDirCfg, fakeCollector)
	if !strings.Contains(err.Error(), storageError) {
		t.Errorf("GenerateReports %s was expected, got %v", storageError, err)
	}
	podError := "pod error"
	for _, q := range *podQueries {
		mapResults[q.QueryString] = &mockPromResult{err: errors.New(podError)}
	}
	err = GenerateReports(fakeCR, fakeDirCfg, fakeCollector)
	if !strings.Contains(err.Error(), podError) {
		t.Errorf("GenerateReports %s was expected, got %v", podError, err)
	}
	nodeError := "node error"
	for _, q := range *nodeQueries {
		mapResults[q.QueryString] = &mockPromResult{err: errors.New(nodeError)}
	}
	err = GenerateReports(fakeCR, fakeDirCfg, fakeCollector)
	if !strings.Contains(err.Error(), nodeError) {
		t.Errorf("GenerateReports %s was expected, got %v", nodeError, err)
	}
	if err := fakeDirCfg.Reports.RemoveContents(); err != nil {
		t.Fatal("failed to cleanup reports directory")
	}
}

func TestGenerateReportsNoNodeData(t *testing.T) {
	mapResults := make(mappedMockPromResult)
	queryList := []*querys{nodeQueries}
	for _, q := range queryList {
		for _, query := range *q {
			res := &model.Matrix{}
			mapResults[query.QueryString] = &mockPromResult{value: *res}
		}
	}

	fakeCollector := &PrometheusCollector{
		PromConn: mockPrometheusConnection{
			mappedResults: &mapResults,
			t:             t,
		},
		TimeSeries: &fakeTimeRange,
	}
	if err := GenerateReports(fakeCR, fakeDirCfg, fakeCollector); err != nil && err != ErrNoData {
		t.Errorf("Failed to generate reports: %v", err)
	}
	filelist, err := os.ReadDir(filepath.Join("test_files", "test_reports"))
	if err != nil {
		t.Fatalf("Failed to read expected reports dir")
	}
	if len(filelist) != 0 {
		t.Errorf("unexpected report(s) generated: %#v", filelist)
	}
}

func TestGetResourceID(t *testing.T) {
	getResourceIDTests := []struct {
		name  string
		input interface{}
		want  string
	}{
		{name: "with slashes", input: "gce://openshift-gce-devel/us-west1-a/metering-ci-3-ig-m-91kw", want: "metering-ci-3-ig-m-91kw"},
		{name: "without slashes", input: "metering-ci-3-ig-m-91kw", want: "metering-ci-3-ig-m-91kw"},
		{name: "nil provider id", input: nil, want: ""},
		{name: "nonsense int", input: 35987345987, want: ""},
	}
	for _, tt := range getResourceIDTests {
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
			name:  "min",
			query: saveQueryValue{Method: "min"},
			array: []model.SamplePair{{Value: 1.3}, {Value: 2.3}, {Value: 3.3}},
			want:  1.3,
		},
		{
			name:  "min inf",
			query: saveQueryValue{Method: "min"},
			array: []model.SamplePair{{Value: model.SampleValue(math.Inf(1))}, {Value: 2.3}, {Value: model.SampleValue(math.Inf(-1))}},
			want:  math.Inf(-1),
		},
		{
			name:  "avg",
			query: saveQueryValue{Method: "avg"},
			array: []model.SamplePair{{Value: 1.3}, {Value: 2.3}, {Value: 3.3}},
			want:  2.3,
		},
		{
			name:  "avg inf",
			query: saveQueryValue{Method: "avg"},
			array: []model.SamplePair{{Value: model.SampleValue(math.Inf(1))}, {Value: 2.3}, {Value: 3.3}},
			want:  math.Inf(1),
		},
		{
			name:  "avg +/-inf",
			query: saveQueryValue{Method: "avg"},
			array: []model.SamplePair{{Value: model.SampleValue(math.Inf(1))}, {Value: 2.3}, {Value: model.SampleValue(math.Inf(-1))}},
			want:  math.NaN(),
		},
		{
			name:  "unknown",
			query: saveQueryValue{Method: "unknown"},
			array: []model.SamplePair{{Value: 1.3}, {Value: 2.3}, {Value: 3.3}},
			want:  0,
		},
	}
	for _, tt := range getValueTests {
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
				"endpoint": "https-main",
				"instance": "10.131.0.11:8443",
			},
			str:  "label_*",
			want: "",
		},
		{
			name: "one match",
			input: model.Metric{
				"endpoint":                              "https-main",
				"instance":                              "10.131.0.11:8443",
				"label_openshift_io_cluster_monitoring": "true",
			},
			str:  "label_*",
			want: "label_openshift_io_cluster_monitoring:true",
		},
		{
			name: "multiple matches",
			input: model.Metric{
				"endpoint":                              "https-main",
				"instance":                              "10.131.0.11:8443",
				"label_controller_tools_k8s_io":         "1.0",
				"label_openshift_io_cluster_monitoring": "true",
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
					TransformedName: "node-allocatable-cpu-core-seconds",
				},
				RowKey: []model.LabelName{"node"},
			},
			matrix: model.Matrix{
				{
					Metric: model.Metric{
						"endpoint":    "https-main",
						"instance":    "10.131.0.11:8443",
						"node":        "ip-10-0-222-213.us-east-2.compute.internal",
						"provider_id": "aws:///us-east-2c/i-070043b2c0291bdc2",
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
				RowKey:         []model.LabelName{"node"},
			},
			matrix: model.Matrix{
				{
					Metric: model.Metric{
						"endpoint":            "https-main",
						"instance":            "10.131.0.11:8443",
						"label_arch":          "amd64",
						"label_instance_type": "m5.xlarge",
						"label_os":            "linux",
						"node":                "ip-10-0-222-213.us-east-2.compute.internal",
					},
					Values: []model.SamplePair{
						{Timestamp: 1604339340, Value: 1},
						{Timestamp: 1604339400, Value: 1},
						{Timestamp: 1604339460, Value: 1},
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
					"node_labels":                       "label_arch:amd64|label_instance_type:m5.xlarge|label_os:linux",
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
					TransformedName: "node-capacity-cpu-core-seconds",
				},
				RowKey: []model.LabelName{"node"},
			},
			matrix: model.Matrix{
				{
					Metric: model.Metric{
						"endpoint":    "https-main",
						"instance":    "10.131.0.11:8443",
						"node":        "ip-10-0-222-213.us-east-2.compute.internal",
						"provider_id": "aws:///us-east-2c/i-070043b2c0291bdc2",
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
					"node_labels":                       "label_arch:amd64|label_instance_type:m5.xlarge|label_os:linux",
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
