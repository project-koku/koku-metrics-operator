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
	podFilePrefix       = "cm-openshift-pod-usage-"
	volFilePrefix       = "cm-openshift-storage-usage-"
	nodeFilePrefix      = "cm-openshift-node-usage-"
	namespaceFilePrefix = "cm-openshift-namespace-usage-"
	rosFilePrefix       = "ros-openshift-"

	statusTimeFormat = "2006-01-02 15:04:05"

	log = logr.Log.WithName("collector")

	ErrNoData = errors.New("no data to collect")
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
		for i := 1; i < 5; i++ {
			timeRange.Start = start
			timeRange.End = end
			rosCollector.TimeSeries = timeRange
			if err := generateResourceOpimizationReports(log, rosCollector, dirCfg, nodeRows, yearMonth); err != nil {
				return err
			}
			start = start.Add(15 * time.Minute)
			end = end.Add(15 * time.Minute)
		}

	}

	//################################################################################################################

	return nil
}

func generateCostManagementReports(log gologr.Logger, c *PrometheusCollector, dirCfg *dirconfig.DirectoryConfig, nodeRows mappedCSVStruct, yearMonth string) error {
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

	//################################################################################################################

	log.Info("querying for pod metrics")
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

	//################################################################################################################

	log.Info("querying for storage metrics")
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
	log.WithName("writeResults").Info("writing volume results to file", "filename", volReport.file.getName())
	if err := volReport.writeReport(); err != nil {
		return fmt.Errorf("failed to write volume report: %v", err)
	}

	//################################################################################################################

	log.Info("querying for namespaces")
	namespaceResults := mappedResults{}
	if err := c.getQueryRangeResults(namespaceQueries, &namespaceResults, MaxRetries); err != nil {
		return err
	}

	namespaceRows := make(mappedCSVStruct)
	for namespace, val := range namespaceResults {
		usage := newNamespaceRow(c.TimeSeries)
		if err := getStruct(val, &usage, namespaceRows, namespace); err != nil {
			return err
		}
	}
	emptyNameRow := newNamespaceRow(c.TimeSeries)
	namespaceReport := report{
		file: &file{
			name: namespaceFilePrefix + yearMonth + ".csv",
			path: dirCfg.Reports.Path,
		},
		data: &data{
			queryData: namespaceRows,
			headers:   emptyNameRow.csvHeader(),
			prefix:    emptyNameRow.dateTimes.string(),
		},
	}
	log.WithName("writeResults").Info("writing namespace results to file", "filename", namespaceReport.file.getName())
	if err := namespaceReport.writeReport(); err != nil {
		return fmt.Errorf("failed to write namespace report: %v", err)
	}

	return nil
}

func generateResourceOpimizationReports(log gologr.Logger, c *PrometheusCollector, dirCfg *dirconfig.DirectoryConfig, nodeRows mappedCSVStruct, yearMonth string) error {
	ts := c.TimeSeries.End
	log.Info(fmt.Sprintf("querying for resource-optimization for ts: %+v", ts))
	rosResults := mappedResults{}
	if err := c.getQueryResults(ts, resourceOptimizationQueries, &rosResults, MaxRetries); err != nil {
		return err
	}

	rosRows := make(mappedCSVStruct)
	for ros, val := range rosResults {
		usage := newROSRow(c.TimeSeries)
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
	emptyROSRow := newROSRow(c.TimeSeries)
	rosReport := report{
		file: &file{
			name: rosFilePrefix + yearMonth + ".csv",
			path: dirCfg.Reports.Path,
		},
		data: &data{
			queryData: rosRows,
			headers:   emptyROSRow.csvHeader(),
			prefix:    emptyROSRow.dateTimes.string(),
		},
	}
	log.WithName("writeResults").Info("writing resource-optimization results to file", "filename", rosReport.file.getName())
	if err := rosReport.writeReport(); err != nil {
		return fmt.Errorf("failed to write resource-optimization report: %v", err)
	}
	return nil
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
