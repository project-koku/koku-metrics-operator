//
// Copyright 2021 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package collector

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	gologr "github.com/go-logr/logr"
	"github.com/mitchellh/mapstructure"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	logr "sigs.k8s.io/controller-runtime/pkg/log"

	metricscfgv1beta1 "github.com/project-koku/koku-metrics-operator/api/v1beta1"
	"github.com/project-koku/koku-metrics-operator/internal/dirconfig"
)

var (
	podFilePrefix          = "cm-openshift-pod-usage-"
	volFilePrefix          = "cm-openshift-storage-usage-"
	vmFilePrefix           = "cm-openshift-vm-usage-"
	nodeFilePrefix         = "cm-openshift-node-usage-"
	namespaceFilePrefix    = "cm-openshift-namespace-usage-"
	nvidiaGpuFilePrefix    = "cm-openshift-nvidia-gpu-usage-"
	rosContainerFilePrefix = "ros-openshift-container-"
	rosNamespaceFilePrefix = "ros-openshift-namespace-"

	statusTimeFormat = "2006-01-02 15:04:05"

	log = logr.Log.WithName("collector")

	ErrNoData                 = errors.New("no data to collect")
	ErrROSNoEnabledNamespaces = errors.New("no enabled namespaces for ROS")
)

type mappedCSVStruct map[string]csvStruct
type mappedResults map[string]mappedValues
type mappedValues map[string]interface{}

func floatToString(inputNum float64) string {
	// to convert a float number to a string
	return strconv.FormatFloat(inputNum, 'f', 6, 64)
}

func maxSlice(array []model.SamplePair) float64 {
	max := array[0].Value
	for _, v := range array {
		if v.Value > max {
			max = v.Value
		}
	}
	return float64(max)
}

func minSlice(array []model.SamplePair) float64 {
	min := array[0].Value
	for _, v := range array {
		if v.Value < min {
			min = v.Value
		}
	}
	return float64(min)
}

func avgSlice(array []model.SamplePair) float64 {
	length := len(array)
	if length <= 0 {
		return 0
	}
	sum := sumSlice(array)
	return sum / float64(length)
}

func sumSlice(array []model.SamplePair) float64 {
	var sum model.SampleValue
	for _, v := range array {
		sum += v.Value
	}
	return float64(sum)
}

func getValue(query *saveQueryValue, array []model.SamplePair) float64 {
	switch query.Method {
	case "sum":
		return sumSlice(array)
	case "max":
		return maxSlice(array)
	case "min":
		return minSlice(array)
	case "avg":
		return avgSlice(array)
	default:
		return 0
	}
}

func getStruct(val mappedValues, usage csvStruct, rowResults mappedCSVStruct, key string) error {
	if err := mapstructure.Decode(val, &usage); err != nil {
		return fmt.Errorf("getStruct: failed to convert map to struct: %v", err)
	}
	rowResults[key] = usage
	return nil
}

func getResourceID(input interface{}) string {
	input_string, ok := input.(string)
	if !ok {
		log.Info(fmt.Sprintf("failed to get resource-id from provider-id: %v", input))
		return ""
	}
	splitString := strings.Split(input_string, "/")
	return splitString[len(splitString)-1]
}

func generateKey(metric model.Metric, keys []model.LabelName) string {
	if len(keys) == 1 {
		return string(metric[keys[0]])
	}
	result := []string{}
	for _, key := range keys {
		result = append(result, string(metric[key]))
	}
	sort.Strings(result)

	return strings.Join(result, ",")
}

func (r *mappedResults) iterateVector(vector model.Vector, q query) {
	results := *r
	for _, sample := range vector {
		obj := generateKey(sample.Metric, q.RowKey)
		if results[obj] == nil {
			results[obj] = mappedValues{}
		}
		if q.MetricKey != nil {
			for key, field := range q.MetricKey {
				results[obj][key] = string(sample.Metric[field])
			}
		}
		if q.MetricKeyRegex != nil {
			for key, regexField := range q.MetricKeyRegex {
				results[obj][key] = findFields(sample.Metric, regexField)
			}
		}
		if q.QueryValue != nil {
			saveStruct := q.QueryValue
			value := float64(sample.Value)
			results[obj][saveStruct.ValName] = floatToString(value)
		}
	}
}

func (r *mappedResults) iterateMatrix(matrix model.Matrix, q query) {
	results := *r
	for _, stream := range matrix {
		obj := generateKey(stream.Metric, q.RowKey)
		if results[obj] == nil {
			results[obj] = mappedValues{}
		}
		if q.MetricKey != nil {
			for key, field := range q.MetricKey {
				results[obj][key] = string(stream.Metric[field])
			}
		}
		if q.MetricKeyRegex != nil {
			for key, regexField := range q.MetricKeyRegex {
				results[obj][key] = findFields(stream.Metric, regexField)
			}
		}
		if q.QueryValue != nil {
			saveStruct := q.QueryValue
			value := getValue(saveStruct, stream.Values)
			results[obj][saveStruct.ValName] = floatToString(value)
			if saveStruct.TransformedName != "" {
				factor := float64(60)
				if saveStruct.Method == "max" {
					factor *= float64(len(stream.Values))
				}
				results[obj][saveStruct.TransformedName] = floatToString(value * factor)
			}
		}
	}
}

// GenerateReports is responsible for querying prometheus and writing to report files
func GenerateReports(cr *metricscfgv1beta1.MetricsConfig, dirCfg *dirconfig.DirectoryConfig, c *PrometheusCollector) error {
	log := log.WithName("GenerateReports")
	log.Info(fmt.Sprintf("prometheus query timeout set to: %.0f seconds", c.ContextTimeout.Seconds()))

	// yearMonth is used in filenames
	yearMonth := c.TimeSeries.Start.Format("200601") // this corresponds to YYYYMM format
	updateReportStatus(cr, c.TimeSeries)

	// ################################################################################################################
	log.Info("querying for node metrics")
	nodeResults := mappedResults{}
	if err := c.getQueryRangeResults(nodeQueries, &nodeResults, MaxRetries); err != nil {
		return err
	}

	if len(nodeResults) <= 0 {
		log.Info("no data to report")
		// there is no data for the hour queried. Return nothing
		return ErrNoData
	}
	for node, val := range nodeResults {
		resourceID := getResourceID(val["provider_id"])
		nodeResults[node]["resource_id"] = resourceID
	}

	nodeRows := make(mappedCSVStruct)
	for node, val := range nodeResults {
		usage := newNodeRow(c.TimeSeries)
		if err := getStruct(val, &usage, nodeRows, node); err != nil {
			return err
		}
	}

	// ######## this actually generates the node report and the others for cost-management
	if cr.Spec.PrometheusConfig.DisableMetricsCollectionCostManagement != nil && !*cr.Spec.PrometheusConfig.DisableMetricsCollectionCostManagement {
		if err := generateCostManagementReports(log, c, dirCfg, nodeRows, yearMonth); err != nil {
			return err
		}
	}

	// ######## generate resource-optimization reports
	if cr.Spec.PrometheusConfig.DisableMetricsCollectionResourceOptimization != nil && !*cr.Spec.PrometheusConfig.DisableMetricsCollectionResourceOptimization {
		rosCollector := &PrometheusCollector{
			PromConn:           c.PromConn,
			PromCfg:            c.PromCfg,
			ContextTimeout:     c.ContextTimeout,
			serviceaccountPath: c.serviceaccountPath,
		}
		timeRange := c.TimeSeries
		start := timeRange.Start.Add(1 * time.Second)
		end := start.Add(14*time.Minute + 59*time.Second)
		var err error
		for i := 1; i < 5; i++ {
			timeRange.Start = start
			timeRange.End = end
			rosCollector.TimeSeries = timeRange
			if err = generateResourceOptimizationReports(log, rosCollector, dirCfg, nodeRows, yearMonth); err != nil {
				if !errors.Is(err, ErrROSNoEnabledNamespaces) {
					return err
				}
			}
			start = start.Add(15 * time.Minute)
			end = end.Add(15 * time.Minute)
		}

		if errors.Is(err, ErrROSNoEnabledNamespaces) {
			return ErrROSNoEnabledNamespaces
		}
	}

	//################################################################################################################

	return nil
}

func generateCostManagementReports(log gologr.Logger, c *PrometheusCollector, dirCfg *dirconfig.DirectoryConfig, nodeRows mappedCSVStruct, yearMonth string) error {

	// cost node metrics
	if err := generateCostNodeMetricsReport(log, c, dirCfg, nodeRows, yearMonth); err != nil {
		return err
	}

	// cost pod metrics
	if err := generateCostPodMetricsReport(log, c, dirCfg, nodeRows, yearMonth); err != nil {
		return err
	}

	// cost storage metrics
	if err := generateCostStorageMetricsReport(log, c, dirCfg, yearMonth); err != nil {
		return err
	}

	// cost vm metrics
	if err := generateCostVMMetricsReport(log, c, dirCfg, yearMonth); err != nil {
		return err
	}

	// cost namespace metric
	if err := generateCostNamespaceMetricsReport(log, c, dirCfg, yearMonth); err != nil {
		return err
	}

	//cost nvidia gpu metrics
	if err := generateCostNvidiaGpuMetricsReport(log, c, dirCfg, yearMonth); err != nil {
		return err
	}

	return nil
}

// generateCostNodeMetricsReport generates the report for node metrics.
func generateCostNodeMetricsReport(log gologr.Logger, c *PrometheusCollector, dirCfg *dirconfig.DirectoryConfig, nodeRows mappedCSVStruct, yearMonth string) error {
	log.Info("querying for node metrics")
	emptyNodeRow := newNodeRow(c.TimeSeries)
	nodeReport := report{
		file: &file{
			name: nodeFilePrefix + yearMonth + ".csv",
			path: dirCfg.Reports.Path,
		},
		data: &data{
			queryData: nodeRows,
			headers:   emptyNodeRow.csvHeader(),
			prefix:    emptyNodeRow.dateTimes.string(),
		},
	}
	log.WithName("writeResults").Info("writing node results to file", "filename", nodeReport.file.getName())
	if err := nodeReport.writeReport(); err != nil {
		return fmt.Errorf("failed to write node report: %v", err)
	}

	return nil
}

// generateCostNamespaceMetricsReport generates the report for namespace metrics.
func generateCostNamespaceMetricsReport(log gologr.Logger, c *PrometheusCollector, dirCfg *dirconfig.DirectoryConfig, yearMonth string) error {
	log.Info("querying for cost namespace metrics")
	namespaceResults := mappedResults{}
	if err := c.getQueryRangeResults(namespaceQueries, &namespaceResults, MaxRetries); err != nil {
		return err
	}

	namespaceRows := make(mappedCSVStruct)
	for ns, val := range namespaceResults {
		usage := newNamespaceRow(c.TimeSeries)
		if err := getStruct(val, &usage, namespaceRows, ns); err != nil {
			return err
		}
	}

	emptyNamespaceRow := newNamespaceRow(c.TimeSeries)
	namespaceReport := report{
		file: &file{
			name: namespaceFilePrefix + yearMonth + ".csv",
			path: dirCfg.Reports.Path,
		},
		data: &data{
			queryData: namespaceRows,
			headers:   emptyNamespaceRow.csvHeader(),
			prefix:    emptyNamespaceRow.dateTimes.string(),
		},
	}

	log.WithName("writeResults").Info("writing cost namespace results to file", "filename", namespaceReport.file.getName())
	if err := namespaceReport.writeReport(); err != nil {
		return fmt.Errorf("failed to write cost namespace report: %v", err)
	}
	return nil
}

// generateCostPodMetricsReport generates the report for pod metrics.
func generateCostPodMetricsReport(log gologr.Logger, c *PrometheusCollector, dirCfg *dirconfig.DirectoryConfig, nodeRows mappedCSVStruct, yearMonth string) error {
	log.Info("querying for cost pod metrics")
	podResults := mappedResults{}
	if err := c.getQueryRangeResults(podQueries, &podResults, MaxRetries); err != nil {
		return err
	}

	podRows := make(mappedCSVStruct)
	for pod, val := range podResults {
		usage := newPodRow(c.TimeSeries)
		if err := getStruct(val, &usage, podRows, pod); err != nil {
			return err
		}
		if node, ok := val["node"]; ok {
			// Add the Node usage to the pod.
			if row, ok := nodeRows[node.(string)]; ok {
				usage.nodeRow = *row.(*nodeRow)
			} else {
				usage.nodeRow = newNodeRow(c.TimeSeries)
			}
		}
	}

	emptyPodRow := newPodRow(c.TimeSeries)
	podReport := report{
		file: &file{
			name: podFilePrefix + yearMonth + ".csv",
			path: dirCfg.Reports.Path,
		},
		data: &data{
			queryData: podRows,
			headers:   emptyPodRow.csvHeader(),
			prefix:    emptyPodRow.dateTimes.string(),
		},
	}

	log.WithName("writeResults").Info("writing pod results to file", "filename", podReport.file.getName())
	if err := podReport.writeReport(); err != nil {
		return fmt.Errorf("failed to write pod report: %v", err)
	}

	return nil
}

// generateCostStorageMetricsReport generates the report for storage metrics.
func generateCostStorageMetricsReport(log gologr.Logger, c *PrometheusCollector, dirCfg *dirconfig.DirectoryConfig, yearMonth string) error {
	log.Info("querying for cost storage metrics")
	volResults := mappedResults{}
	if err := c.getQueryRangeResults(volQueries, &volResults, MaxRetries); err != nil {
		return err
	}

	volRows := make(mappedCSVStruct)
	for pvc, val := range volResults {
		usage := newStorageRow(c.TimeSeries)
		if err := getStruct(val, &usage, volRows, pvc); err != nil {
			return err
		}
	}

	emptyVolRow := newStorageRow(c.TimeSeries)
	volReport := report{
		file: &file{
			name: volFilePrefix + yearMonth + ".csv",
			path: dirCfg.Reports.Path,
		},
		data: &data{
			queryData: volRows,
			headers:   emptyVolRow.csvHeader(),
			prefix:    emptyVolRow.dateTimes.string(),
		},
	}

	log.WithName("writeResults").Info("writing storage results to file", "filename", volReport.file.getName())
	if err := volReport.writeReport(); err != nil {
		return fmt.Errorf("failed to write storage report: %v", err)
	}

	return nil
}

// generateCostVMMetricsReport generates the report for VM metrics.
func generateCostVMMetricsReport(log gologr.Logger, c *PrometheusCollector, dirCfg *dirconfig.DirectoryConfig, yearMonth string) error {
	log.Info("querying for cost vm metrics")
	virtualMachineResults := mappedResults{}
	if err := c.getQueryRangeResults(vmQueries, &virtualMachineResults, MaxRetries); err != nil {
		return err
	}

	virtualMachineRows := make(mappedCSVStruct)
	for vm, val := range virtualMachineResults {
		usage := newVMRow(c.TimeSeries)
		if err := getStruct(val, &usage, virtualMachineRows, vm); err != nil {
			return err
		}
	}

	emptyVmRow := newVMRow(c.TimeSeries)
	virtualMachineReport := report{
		file: &file{
			name: vmFilePrefix + yearMonth + ".csv",
			path: dirCfg.Reports.Path,
		},
		data: &data{
			queryData: virtualMachineRows,
			headers:   emptyVmRow.csvHeader(),
			prefix:    emptyVmRow.dateTimes.string(),
		},
	}

	log.WithName("writeResults").Info("writing cost vm results to file", "filename", virtualMachineReport.file.getName())
	if err := virtualMachineReport.writeReport(); err != nil {
		return fmt.Errorf("failed to write cost vm report: %v", err)
	}

	return nil
}

// generateCostNvidiaGpuMetricsReport generates the report for NVIDIA GPU metrics.
func generateCostNvidiaGpuMetricsReport(log gologr.Logger, c *PrometheusCollector, dirCfg *dirconfig.DirectoryConfig, yearMonth string) error {
	log.Info("querying for cost nvidia gpu metrics")
	nvidiaGpuResults := mappedResults{}
	if err := c.getQueryRangeResults(costNvidiaGpuQueries, &nvidiaGpuResults, MaxRetries); err != nil {
		return err
	}

	nvidiaGpuRows := make(mappedCSVStruct)
	gpuUtilizationResults := make(map[string]mappedValues)
	gpuResourceResults := make(map[string]mappedValues)

	// Separate the results
	for key, val := range nvidiaGpuResults {
		if _, ok := val["gpu_uuid"]; ok {
			gpuUtilizationResults[key] = val
		} else {
			gpuResourceResults[key] = val
		}
	}

	// For each gpu utilization result, find the matching memory & resource data and merge it
	for key, gpuVal := range gpuUtilizationResults {
		pod, _ := gpuVal["pod"].(string)
		namespace, _ := gpuVal["namespace"].(string)
		node, _ := gpuVal["node"].(string)

		// key generation to match `generateKey`
		resourceKeyParts := []string{pod, namespace, node}
		sort.Strings(resourceKeyParts)
		resourceKey := strings.Join(resourceKeyParts, ",")

		if resourceData, ok := gpuResourceResults[resourceKey]; ok {
			log.Info("found matching gpu resource data", "key", resourceKey)
			for dataKey, dataVal := range resourceData {
				gpuVal[dataKey] = dataVal
			}
		}

		usage := newNvidiaGpuRow(c.TimeSeries)
		if err := getStruct(gpuVal, &usage, nvidiaGpuRows, key); err != nil {
			return err
		}
	}
	emptyNvidiaGpuRow := newNvidiaGpuRow(c.TimeSeries)
	nvidiaGpuReport := report{
		file: &file{
			name: nvidiaGpuFilePrefix + yearMonth + ".csv",
			path: dirCfg.Reports.Path,
		},
		data: &data{
			queryData: nvidiaGpuRows,
			headers:   emptyNvidiaGpuRow.csvHeader(),
			prefix:    emptyNvidiaGpuRow.dateTimes.string(),
		},
	}

	log.WithName("writeResults").Info("writing cost nvidia gpu results to file", "filename", nvidiaGpuReport.file.getName())
	if err := nvidiaGpuReport.writeReport(); err != nil {
		return fmt.Errorf("failed to write cost nvidia gpu report: %v", err)
	}
	return nil
}

func generateResourceOptimizationReports(log gologr.Logger, c *PrometheusCollector, dirCfg *dirconfig.DirectoryConfig, nodeRows mappedCSVStruct, yearMonth string) error {
	ts := c.TimeSeries.End
	namespacesAreEnabled, err := areNamespacesEnabled(c, ts)
	if err != nil {
		return err
	}
	if !namespacesAreEnabled {
		return ErrROSNoEnabledNamespaces
	}

	log.Info(fmt.Sprintf("querying for resource-optimization container metrics for ts: %+v", ts))
	rosResults := mappedResults{}

	if err := c.getQueryResults(ts, rosContainerQueries, &rosResults, MaxRetries); err != nil {
		return err
	}

	rosRows := make(mappedCSVStruct)
	for ros, val := range rosResults {
		usage := newROSContainerRow(c.TimeSeries)
		if err := getStruct(val, &usage, rosRows, ros); err != nil {
			return err
		}
		if node, ok := val["node"]; ok {
			// Add the Node usage to the pod.
			if row, ok := nodeRows[node.(string)]; ok {
				usage.nodeRow = *row.(*nodeRow)
			} else {
				usage.nodeRow = newNodeRow(c.TimeSeries)
			}
		}
	}
	emptyROSRow := newROSContainerRow(c.TimeSeries)
	rosReport := report{
		file: &file{
			name: rosContainerFilePrefix + yearMonth + ".csv",
			path: dirCfg.Reports.Path,
		},
		data: &data{
			queryData: rosRows,
			headers:   emptyROSRow.csvHeader(),
			prefix:    emptyROSRow.dateTimes.string(),
		},
	}
	log.WithName("writeResults").Info("writing resource-optimization container results to file", "filename", rosReport.file.getName())
	if err := rosReport.writeReport(); err != nil {
		return fmt.Errorf("failed to write resource-optimization container report: %v", err)
	}

	//resource optimization namespace reports
	log.Info(fmt.Sprintf("querying for resource-optimization namespace metrics for ts: %+v", ts))
	rosNamespaceResults := mappedResults{}

	if err := c.getQueryResults(ts, rosNamespaceQueries, &rosNamespaceResults, MaxRetries); err != nil {
		return err
	}

	rosNamespaceRows := make(mappedCSVStruct)
	for rosNs, val := range rosNamespaceResults {
		usage := newROSNamespaceRow(c.TimeSeries)
		if err := getStruct(val, &usage, rosNamespaceRows, rosNs); err != nil {
			return err
		}
	}
	emptyROSNamespaceRow := newROSNamespaceRow(c.TimeSeries)
	rosNamespaceReport := report{
		file: &file{
			name: rosNamespaceFilePrefix + yearMonth + ".csv",
			path: dirCfg.Reports.Path,
		},
		data: &data{
			queryData: rosNamespaceRows,
			headers:   emptyROSNamespaceRow.csvHeader(),
			prefix:    emptyROSNamespaceRow.dateTimes.string(),
		},
	}
	log.WithName("writeResults").Info("writing resource-optimization namespace results to file", "filename", rosNamespaceReport.file.getName())
	if err := rosNamespaceReport.writeReport(); err != nil {
		return fmt.Errorf("failed to write resource-optimization namespace report: %v", err)
	}

	return nil
}

func areNamespacesEnabled(c *PrometheusCollector, ts time.Time) (bool, error) {
	vector, err := c.getVectorQuerySimple(rosNamespaceFilter, ts)
	if err != nil {
		return false, fmt.Errorf("failed to query for namespaces: %v", err)
	}

	namespaces := []string{}
	for _, sample := range vector {
		for _, field := range rosNamespaceFilter.MetricKey {
			namespaces = append(namespaces, string(sample.Metric[field]))
		}
	}
	return len(namespaces) > 0, nil
}

func findFields(input model.Metric, str string) string {
	result := []string{}
	for name, val := range input {
		name := string(name)
		match, _ := regexp.MatchString(str, name)
		if match {
			result = append(result, name+":"+string(val))
		}
	}
	switch length := len(result); {
	case length > 0:
		sort.Strings(result)
		return strings.Join(result, "|")
	default:
		return ""
	}
}

func updateReportStatus(cr *metricscfgv1beta1.MetricsConfig, ts *promv1.Range) {
	cr.Status.Reports.ReportMonth = ts.Start.Format("01")
	cr.Status.Reports.LastHourQueried = ts.Start.Format(statusTimeFormat) + " - " + ts.End.Format(statusTimeFormat)
}
