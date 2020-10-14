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
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/project-koku/korekuta-operator-go/strset"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

var (
	logger logr.Logger

	dataPath            = "/tmp/cost-mgmt-operator-reports/data/"
	podFilePrefix       = "cm-openshift-usage-lookback-"
	volFilePrefix       = "cm-openshift-persistentvolumeclaim-lookback-"
	nodeFilePrefix      = "cm-openshift-node-labels-lookback-"
	namespaceFilePrefix = "cm-openshift-namespace-labels-lookback-"

	nodeQueries = mappedQuery{
		"node-allocatable-cpu-cores":    "kube_node_status_allocatable_cpu_cores * on(node) group_left(provider_id) max(kube_node_info) by (node, provider_id)",
		"node-allocatable-memory-bytes": "kube_node_status_allocatable_memory_bytes * on(node) group_left(provider_id) max(kube_node_info) by (node, provider_id)",
		"node-capacity-cpu-cores":       "kube_node_status_capacity_cpu_cores * on(node) group_left(provider_id) max(kube_node_info) by (node, provider_id)",
		"node-capacity-memory-bytes":    "kube_node_status_capacity_memory_bytes * on(node) group_left(provider_id) max(kube_node_info) by (node, provider_id)",
	}
	volQueries = mappedQuery{
		"persistentvolumeclaim-info":           "kube_persistentvolumeclaim_info",
		"persistentvolumeclaim-capacity-bytes": "kubelet_volume_stats_capacity_bytes",
		"persistentvolumeclaim-request-bytes":  "kube_persistentvolumeclaim_resource_requests_storage_bytes",
		"persistentvolumeclaim-usage-bytes":    "kubelet_volume_stats_used_bytes",
	}
	podQueries = mappedQuery{
		"pod-limit-cpu-cores":      "sum(kube_pod_container_resource_limits_cpu_cores) by (pod, namespace, node)",
		"pod-limit-memory-bytes":   "sum(kube_pod_container_resource_limits_memory_bytes) by (pod, namespace, node)",
		"pod-request-cpu-cores":    "sum(kube_pod_container_resource_requests_cpu_cores) by (pod, namespace, node)",
		"pod-request-memory-bytes": "sum(kube_pod_container_resource_requests_memory_bytes) by (pod, namespace, node)",
		"pod-usage-cpu-cores":      "sum(rate(container_cpu_usage_seconds_total{container!='POD',container!='',pod!=''}[5m])) BY (pod, namespace, node)",
		"pod-usage-memory-bytes":   "sum(container_memory_usage_bytes{container!='POD', container!='',pod!=''}) by (pod, namespace, node)",
	}
	labelQueries = map[string][]string{
		"namespace-labels":             {"namespace", "kube_namespace_labels"},
		"node-labels":                  {"node", "kube_node_labels"},
		"persistentvolume-labels":      {"persistentvolume", "kube_persistentvolume_labels"},
		"persistentvolumeclaim-labels": {"persistentvolumeclaim", "kube_persistentvolumeclaim_labels"},
		"pod-labels":                   {"pod", "kube_pod_labels"},
	}
)

type mappedCSVStruct map[string]CSVStruct
type mappedQuery map[string]string
type mappedResults map[string]mappedValues
type mappedValues map[string]interface{}
type collector struct {
	Context              context.Context
	PrometheusConnection promv1.API
	TimeSeries           promv1.Range
	Log                  logr.Logger
}

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

func sumSlice(array []model.SamplePair) float64 {
	var sum model.SampleValue
	for _, v := range array {
		sum += v.Value
	}
	return float64(sum)
}

func getValue(query string, array []model.SamplePair) float64 {
	switch {
	case strings.Contains(query, "usage"), strings.Contains(query, "limit"), strings.Contains(query, "request"):
		return sumSlice(array)
	default:
		return maxSlice(array)
	}
}

func iterateMatrix(matrix model.Matrix, labelName model.LabelName, results mappedResults, qname string) mappedResults {
	for _, stream := range matrix {
		obj := string(stream.Metric[labelName])
		if results[obj] == nil {
			results[obj] = mappedValues{}
		}
		for labelName, labelValue := range stream.Metric {
			results[obj][string(labelName)] = string(labelValue)
		}
		value := getValue(qname, stream.Values)
		results[obj][qname] = floatToString(value)
		if strings.HasSuffix(qname, "-cores") || strings.HasSuffix(qname, "-bytes") {
			index := qname[:len(qname)-1] + "-seconds"
			results[obj][index] = floatToString(value * float64(len(stream.Values)))
		}
		if strings.HasPrefix(qname, "node-capacity") {
			index := qname[:len(qname)-1] + "-seconds"
			results[obj][index] = floatToString(value * 60 * float64(len(stream.Values)))
		}
	}
	return results
}

func getQueryResults(q collector, queries mappedQuery, key string) (mappedResults, error) {
	results := mappedResults{}
	for qname, query := range queries {
		matrix, err := performMatrixQuery(q, query)
		if err != nil {
			return nil, err
		}
		results = iterateMatrix(matrix, model.LabelName(key), results, qname)
	}
	return results, nil
}

// GenerateReports is responsible for querying prometheus and writing to report files
func GenerateReports(promconn promv1.API, ts promv1.Range, log logr.Logger) error {
	if logger == nil {
		logger = log
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	log = log.WithValues("costmanagement", "GenerateReports")
	defer cancel()

	querier := collector{
		Context:              ctx,
		PrometheusConnection: promconn,
		TimeSeries:           ts,
		Log:                  log,
	}

	// yearMonth is used in filenames
	yearMonth := ts.Start.Format("200601") // this corresponds to YYYYMM format

	log.Info("querying for node metrics")
	nodeResults, err := getQueryResults(querier, nodeQueries, "node")
	if err != nil {
		return err
	}

	if len(nodeResults) <= 0 {
		log.Info("no data to report")
		// there is no data for the hour queried. Return nothing
		return nil
	}
	for node, val := range nodeResults {
		resourceID := getResourceID(val["provider_id"].(string))
		nodeResults[node]["resource_id"] = resourceID
	}

	log.Info("querying for pod metrics")
	podResults, err := getQueryResults(querier, podQueries, "pod")
	if err != nil {
		return err
	}

	log.Info("querying for storage metrics")
	volResults, err := getQueryResults(querier, volQueries, "persistentvolumeclaim")
	if err != nil {
		return err
	}

	log.Info("querying for labels")
	var labelResults = map[string]mappedResults{}
	for _, labelQuery := range labelQueries {
		label, query := labelQuery[0], labelQuery[1]
		if labelResults[label] == nil {
			labelResults[label] = mappedResults{}
		}
		results := labelResults[label]
		vector, err := performTheQuery(querier, query)
		if err != nil {
			return err
		}
		for _, val := range vector {
			label := string(val.Metric[model.LabelName(label)])
			labels := parseLabels(val.Metric)
			if results[label] == nil {
				results[label] = mappedValues{}
			}
			for labelName, val := range val.Metric {
				results[label][string(labelName)] = string(val)
			}
			results[label]["labels"] = labels
		}
	}

	podRows := make(mappedCSVStruct)
	for pod, val := range podResults {
		if node, ok := val["node"]; ok {
			// add the node queries into the pod results
			node := node.(string)
			dict, ok := nodeResults[string(node)]
			if !ok {
				return fmt.Errorf("node %s not found", node)
			}
			val["node-capacity-cpu-cores"] = dict["node-capacity-cpu-cores"]
			val["node-capacity-cpu-cores-seconds"] = dict["node-capacity-cpu-core-seconds"]
			val["node-capacity-memory-bytes"] = dict["node-capacity-memory-bytes"]
			val["node-capacity-memory-bytes-seconds"] = dict["node-capacity-memory-byte-seconds"]
			val["resource_id"] = dict["resource_id"]
		}

		val["pod_labels"] = labelResults["pod"][pod]["labels"]

		usage := NewPodRow(ts)
		if err := getStruct(val, &usage, podRows, pod); err != nil {
			return err
		}
	}
	if err := writeResults(podFilePrefix, yearMonth, "pod", podRows); err != nil {
		return err
	}

	volRows := make(mappedCSVStruct)
	for pvc, val := range volResults {
		pv := val["volumename"].(string)
		val["persistentvolume"] = pv
		val["persistentvolume_labels"] = labelResults["persistentvolume"][pv]["labels"]
		val["persistentvolumeclaim_labels"] = labelResults["persistentvolumeclaim"][pvc]["labels"]

		usage := NewStorageRow(ts)
		if err := getStruct(val, &usage, volRows, pvc); err != nil {
			return err
		}
	}
	if err := writeResults(volFilePrefix, yearMonth, "volume", volRows); err != nil {
		return err
	}

	nodeRows := make(mappedCSVStruct)
	for node, val := range nodeResults {
		val["node_labels"] = labelResults["node"][node]["labels"]

		usage := NewNodeRow(ts)
		if err := getStruct(val, &usage, nodeRows, node); err != nil {
			return err
		}
	}
	if err := writeResults(nodeFilePrefix, yearMonth, "node", nodeRows); err != nil {
		return err
	}

	namespaceRows := make(mappedCSVStruct)
	namespaces := labelResults["namespace"]
	for namespace, val := range namespaces {
		val["namespace_labels"] = namespaces[namespace]["labels"]

		usage := NewNamespaceRow(ts)
		if err := getStruct(val, &usage, namespaceRows, namespace); err != nil {
			return err
		}
	}
	if err := writeResults(namespaceFilePrefix, yearMonth, "namespace", namespaceRows); err != nil {
		return err
	}

	return nil
}

func getResourceID(input string) string {
	splitString := strings.Split(input, "/")
	return splitString[len(splitString)-1]
}

func parseLabels(input model.Metric) string {
	result := []string{}
	for name, val := range input {
		name := string(name)
		match, _ := regexp.MatchString("label_*", name)
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

func getStruct(val mappedValues, usage CSVStruct, rowResults mappedCSVStruct, key string) error {
	row, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("failed to marshal pod row")
	}
	if err := json.Unmarshal(row, &usage); err != nil {
		return fmt.Errorf("failed to unmarshal pod row")
	}
	rowResults[key] = usage
	return nil
}

func writeResults(prefix, yearMonth, key string, data mappedCSVStruct) error {
	csvFile, created, err := getOrCreateFile(dataPath, prefix+yearMonth+".csv")
	if err != nil {
		return fmt.Errorf("failed to get or create %s csv: %v", key, err)
	}
	defer csvFile.Close()
	logMsg := fmt.Sprintf("writing %s results to file", key)
	logger.WithValues("costmanagement", "writeResults").Info(logMsg, "filename", csvFile.Name(), "data set", key)
	if err := writeToFile(csvFile, data, created); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}
	return nil
}

func getOrCreateFile(path, filename string) (*os.File, bool, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			return nil, false, err
		}
	}
	filePath := filepath.Join(path, filename)
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		file, err := os.Create(filePath)
		return file, true, err
	}
	if err != nil {
		return nil, false, err
	}
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_RDWR, 0644)
	return file, false, err
}

// writeToFile compares the data to what is in the file and only adds new data to the file
func writeToFile(file *os.File, data mappedCSVStruct, created bool) error {
	set, err := readCsv(file, strset.NewSet())
	if err != nil {
		return fmt.Errorf("failed to read csv: %v", err)
	}
	if created {
		for _, row := range data {
			if err := row.CSVheader(file); err != nil {
				return err
			}
			break // write the headers using the first element in map
		}
	}

	for _, row := range data {
		if !set.Contains(row.String()) {
			if err := row.CSVrow(file); err != nil {
				return err
			}
		}
	}

	return file.Sync()
}

// readCsv reads the file and puts each row into a set
func readCsv(f *os.File, set *strset.Set) (*strset.Set, error) {
	lines, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return set, err
	}
	for _, line := range lines {
		set.Add(strings.Join(line, ","))
	}
	return set, nil
}
