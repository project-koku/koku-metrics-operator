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
	dataPath            = "/tmp/cost-mgmt-operator-reports/data/"
	podFilePrefix       = "cm-openshift-usage-lookback-"
	volFilePrefix       = "cm-openshift-persistentvolumeclaim-lookback-"
	nodeFilePrefix      = "cm-openshift-node-labels-lookback-"
	namespaceFilePrefix = "cm-openshift-namespace-labels-lookback-"

	nodeQueries = map[string]string{
		"node-allocatable-cpu-cores":    "kube_node_status_allocatable_cpu_cores * on(node) group_left(provider_id) max(kube_node_info) by (node, provider_id)",
		"node-allocatable-memory-bytes": "kube_node_status_allocatable_memory_bytes * on(node) group_left(provider_id) max(kube_node_info) by (node, provider_id)",
		"node-capacity-cpu-cores":       "kube_node_status_capacity_cpu_cores * on(node) group_left(provider_id) max(kube_node_info) by (node, provider_id)",
		"node-capacity-memory-bytes":    "kube_node_status_capacity_memory_bytes * on(node) group_left(provider_id) max(kube_node_info) by (node, provider_id)",
	}
	volQueries = map[string]string{
		"persistentvolumeclaim-info":           "kube_persistentvolumeclaim_info",                            // namespace,persistentvolumeclaim,pod,service,storageclass,volumename
		"persistentvolumeclaim-capacity-bytes": "kubelet_volume_stats_capacity_bytes",                        // namespace,node,persistentvolumeclaim
		"persistentvolumeclaim-request-bytes":  "kube_persistentvolumeclaim_resource_requests_storage_bytes", // namespace,persistentvolumeclaim,pod
		"persistentvolumeclaim-usage-bytes":    "kubelet_volume_stats_used_bytes",                            // namespace,node,persistentvolumeclaim
	}
	podQueries = map[string]string{
		"pod-limit-cpu-cores":      "sum(kube_pod_container_resource_limits_cpu_cores) by (pod, namespace, node)",
		"pod-limit-memory-bytes":   "sum(kube_pod_container_resource_limits_memory_bytes) by (pod, namespace, node)",
		"pod-request-cpu-cores":    "sum(kube_pod_container_resource_requests_cpu_cores) by (pod, namespace, node)",
		"pod-request-memory-bytes": "sum(kube_pod_container_resource_requests_memory_bytes) by (pod, namespace, node)",
		"pod-usage-cpu-cores":      "sum(rate(container_cpu_usage_seconds_total{container!='POD',container!='',pod!=''}[5m])) BY (pod, namespace, node)",
		"pod-usage-memory-bytes":   "sum(container_memory_usage_bytes{container!='POD', container!='',pod!=''}) by (pod, namespace, node)",
	}
	// # korekuta queries:
	labelQueries = map[string][]string{
		"namespace-labels":             {"namespace", "kube_namespace_labels"},
		"node-labels":                  {"node", "kube_node_labels"},
		"persistentvolume-labels":      {"persistentvolume", "kube_persistentvolume_labels"},           // namespace,persistentvolume,pod
		"persistentvolumeclaim-labels": {"persistentvolumeclaim", "kube_persistentvolumeclaim_labels"}, // namespace,persistentvolumeclaim,pod
		"pod-labels":                   {"pod", "kube_pod_labels"},
		// "pod-persistentvolumeclaim-info": {"pod", "kube_pod_spec_volumes_persistentvolumeclaims_info"},  // not used?
	}
)

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
	case strings.Contains(query, "usage"):
		return sumSlice(array)
	default:
		return maxSlice(array)
	}
}

func iterateMatrix(matrix model.Matrix, labelName model.LabelName, results map[string]map[string]interface{}, qname string) map[string]map[string]interface{} {
	for _, stream := range matrix {
		obj := string(stream.Metric[labelName])
		if results[obj] == nil {
			results[obj] = map[string]interface{}{}
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
	}
	return results
}

func DoQuery(promconn promv1.API, log logr.Logger) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	log = log.WithValues("costmanagement", "DoQuery")
	t := time.Now()
	start := time.Date(t.Year(), t.Month(), t.Day(), t.Hour()-1, 0, 0, 0, t.Location())
	end := time.Date(t.Year(), t.Month(), t.Day(), t.Hour()-1, 59, 59, 0, t.Location())
	timeRange := promv1.Range{
		Start: start,
		End:   end,
		Step:  time.Minute,
	}
	yearMonth := start.Format("200601") // this corresponds to YYYYMM format
	defer cancel()

	var nodeResults = map[string]map[string]interface{}{}
	for qname, query := range nodeQueries {
		matrix, err := performMatrixQuery(ctx, promconn, query, timeRange, log)
		if err != nil {
			return err
		}
		// if len(matrix) > 0 {
		// 	first := matrix[0]
		// 	fmt.Printf("\nMatrix Results:\n\tMETRICS: %+v\n\tVALUES: \n", first.Metric)
		// 	for name, v := range first.Values {
		// 		fmt.Printf("\t\t%v: %v\n", name, v)
		// 	}
		// 	fmt.Printf("LENGTH STREAM.VALUES: %v\n", len(first.Values))
		// }
		nodeResults = iterateMatrix(matrix, "node", nodeResults, qname)
	}

	if len(nodeResults) <= 0 {
		log.Info("collector: no data to report")
		// there is no data for the hour queried. Return nothing
		return nil
	}

	for node, val := range nodeResults {
		resourceID := getResourceID(val["provider_id"].(string))
		nodeResults[node]["resource_id"] = resourceID
	}

	var podResults = map[string]map[string]interface{}{}
	for qname, query := range podQueries {
		matrix, err := performMatrixQuery(ctx, promconn, query, timeRange, log)
		if err != nil {
			return err
		}
		podResults = iterateMatrix(matrix, "pod", podResults, qname)
	}

	var volResults = map[string]map[string]interface{}{}
	for qname, query := range volQueries {
		matrix, err := performMatrixQuery(ctx, promconn, query, timeRange, log)
		if err != nil {
			return err
		}
		volResults = iterateMatrix(matrix, "persistentvolumeclaim", volResults, qname)
	}
	// for name, res := range podResults {
	// 	fmt.Printf("\nQuery: %s\n\tResult: %v | %v\n", name, res, res["node"])
	// }

	var labelResults = map[string]map[string]interface{}{}
	for _, labelQuery := range labelQueries {
		label, query := labelQuery[0], labelQuery[1]
		vector, err := performTheQuery(ctx, promconn, query, start, log)
		if err != nil {
			return err
		}
		for _, val := range vector {
			label := string(val.Metric[model.LabelName(label)])
			labels := parseLabels(val.Metric)
			if labelResults[label] == nil {
				labelResults[label] = map[string]interface{}{}
			}
			for labelName, val := range val.Metric {
				labelResults[label][string(labelName)] = string(val)
			}
			labelResults[label]["labels"] = labels
			labelResults[label]["timestamp"] = val.Timestamp.Time()
		}
	}

	podRows := make(map[string]CSVThing)
	for pod, val := range podResults {
		if node, ok := val["node"]; ok {
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

		val["pod_labels"] = labelResults[pod]["labels"]

		usage := NewPodRow(timeRange)
		if err := getStruct(val, &usage, podRows, pod); err != nil {
			return err
		}

		// row, err := json.Marshal(val)
		// if err != nil {
		// 	return fmt.Errorf("failed to marshal pod row")
		// }
		// usage := NewPodRow(timeRange)
		// if err := json.Unmarshal(row, &usage); err != nil {
		// 	return fmt.Errorf("failed to unmarshal pod row")
		// }
		// podRows[pod] = usage
	}
	if err := writeResults(podFilePrefix, yearMonth, "pod", podRows); err != nil {
		return err
	}

	// podCSVFile, created, err := getOrCreateFile(dataPath, podFilePrefix+yearMonth+".csv")
	// if err != nil {
	// 	return fmt.Errorf("failed to get or create pod csv: %v", err)
	// }
	// defer podCSVFile.Close()
	// if err := writeToFile(podCSVFile, podRows, created); err != nil {
	// 	return fmt.Errorf("failed to write file: %v", err)
	// }

	volRows := make(map[string]CSVThing)
	for pvc, val := range volResults {
		pv := val["volumename"].(string)
		val["persistentvolume"] = pv
		val["persistentvolume_labels"] = labelResults[pv]["labels"]
		val["persistentvolumeclaim_labels"] = labelResults[pvc]["labels"]

		usage := NewStorageRow(timeRange)
		if err := getStruct(val, &usage, volRows, pvc); err != nil {
			return err
		}
	}
	if err := writeResults(volFilePrefix, yearMonth, "volume", volRows); err != nil {
		return err
	}

	nodeRows := make(map[string]CSVThing)
	for node, val := range nodeResults {
		val["node_labels"] = labelResults[node]["labels"]

		usage := NewNodeRow(timeRange)
		if err := getStruct(val, &usage, nodeRows, node); err != nil {
			return err
		}
	}
	if err := writeResults(nodeFilePrefix, yearMonth, "node", nodeRows); err != nil {
		return err
	}

	return nil
}

func getStruct(val map[string]interface{}, usage CSVThing, rowResults map[string]CSVThing, key string) error {
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

func writeResults(prefix, yearMonth, key string, data map[string]CSVThing) error {
	csvFile, created, err := getOrCreateFile(dataPath, prefix+yearMonth+".csv")
	if err != nil {
		return fmt.Errorf("failed to get or create %s csv: %v", key, err)
	}
	defer csvFile.Close()
	if err := writeToFile(csvFile, data, created); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}
	return nil
}

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

func writeToFile(file *os.File, data map[string]CSVThing, created bool) error {
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

func floatToString(inputNum float64) string {
	// to convert a float number to a string
	return strconv.FormatFloat(inputNum, 'f', 6, 64)
}

func sum(array []int) int {
	result := 0
	for _, v := range array {
		result += v
	}
	return result
}
