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
	prom "github.com/prometheus/client_golang/api/prometheus/v1"
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
		"pod-usage-cpu-cores":      "sum(rate(container_cpu_usage_seconds_total{container!='POD',container!='',pod!=''}[60m])) BY (pod, namespace, node)",
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

func DoQuery(promconn prom.API, log logr.Logger) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	log = log.WithValues("costmanagement", "DoQuery")
	t := time.Now()
	start := time.Date(t.Year(), t.Month(), t.Day(), t.Hour()-1, 59, 59, 0, t.Location())
	yearMonth := start.Format("200601") // this corresponds to YYYYMM format
	defer cancel()

	var nodeResults = map[string]map[string]interface{}{}
	for qname, query := range nodeQueries {
		vector, err := performTheQuery(ctx, promconn, query, start, log)
		if err != nil {
			return err
		}

		for _, val := range vector {
			node := string(val.Metric["node"])
			if nodeResults[node] == nil {
				nodeResults[node] = map[string]interface{}{}
				resourceID := getResourceID(string(val.Metric["provider_id"]))
				nodeResults[node]["resource_id"] = resourceID
			}
			for labelName, val := range val.Metric {
				nodeResults[node][string(labelName)] = string(val)
			}
			nodeResults[node][qname] = floatToString(float64(val.Value))
			if strings.HasSuffix(qname, "-cores") || strings.HasSuffix(qname, "-bytes") {
				index := qname[:len(qname)-1] + "-seconds"
				nodeResults[node][index] = floatToString(float64(val.Value) * 3600)
			}

			nodeResults[node]["timestamp"] = val.Timestamp.Time()
		}
	}
	if len(nodeResults) <= 0 {
		log.Info("collector: no data to report")
		// there is no data for the hour queried. Return nothing
		return nil
	}

	var podResults = map[string]map[string]interface{}{}
	for qname, query := range podQueries {
		vector, err := performTheQuery(ctx, promconn, query, start, log)
		if err != nil {
			return err
		}

		for _, val := range vector {
			pod := string(val.Metric["pod"])
			if podResults[pod] == nil {
				podResults[pod] = map[string]interface{}{}
			}
			for labelName, val := range val.Metric {
				podResults[pod][string(labelName)] = string(val)
			}
			podResults[pod][qname] = floatToString(float64(val.Value))
			if strings.HasSuffix(qname, "-cores") || strings.HasSuffix(qname, "-bytes") {
				index := qname[:len(qname)-1] + "-seconds"
				podResults[pod][index] = floatToString(float64(val.Value) * 3600)
			}
			podResults[pod]["timestamp"] = val.Timestamp.Time()
		}
	}
	for name, res := range podResults {
		fmt.Printf("\nQuery: %s\n\tResult: %v | %v\n", name, res, res["node"])
	}

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

	var volResults = map[string]map[string]interface{}{}
	for qname, query := range volQueries {
		vector, err := performTheQuery(ctx, promconn, query, start, log)
		if err != nil {
			return err
		}

		for _, val := range vector {
			pvc := string(val.Metric["persistentvolumeclaim"])
			if volResults[pvc] == nil {
				volResults[pvc] = map[string]interface{}{}
			}
			for labelName, val := range val.Metric {
				volResults[pvc][string(labelName)] = string(val)
			}
			volResults[pvc][qname] = floatToString(float64(val.Value))
			if strings.HasSuffix(qname, "-cores") || strings.HasSuffix(qname, "-bytes") {
				index := qname[:len(qname)-1] + "-seconds"
				volResults[pvc][index] = floatToString(float64(val.Value) * 3600)
			}
			volResults[pvc]["timestamp"] = val.Timestamp.Time()
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

		row, err := json.Marshal(val)
		if err != nil {
			return fmt.Errorf("failed to marshal pod row")
		}
		usage := NewPodRow(start)
		if err := json.Unmarshal(row, &usage); err != nil {
			return fmt.Errorf("failed to unmarshal pod row")
		}
		podRows[pod] = usage
	}

	podCSVFile, created, err := getOrCreateFile(dataPath, podFilePrefix+yearMonth+".csv")
	if err != nil {
		return fmt.Errorf("failed to get or create pod csv: %v", err)
	}
	defer podCSVFile.Close()
	if err := writeToFile(podCSVFile, podRows, created); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	for pvc, val := range volResults {
		pv := val["volumename"].(string)
		val["persistentvolume"] = pv
		val["persistentvolume_labels"] = labelResults[pv]["labels"]
		val["persistentvolumeclaim_labels"] = labelResults[pvc]["labels"]
	}
	volRows := make(map[string]CSVThing)
	for vol, val := range volResults {
		row, err := json.Marshal(val)
		if err != nil {
			return fmt.Errorf("failed to marshal volume row")
		}
		usage := NewStorageRow(start)
		if err := json.Unmarshal(row, &usage); err != nil {
			return fmt.Errorf("failed to unmarshal volume row")
		}
		volRows[vol] = usage
	}
	volCSVFile, created, err := getOrCreateFile(dataPath, volFilePrefix+yearMonth+".csv")
	if err != nil {
		return fmt.Errorf("failed to get or create vol csv: %v", err)
	}
	defer volCSVFile.Close()
	if err := writeToFile(volCSVFile, volRows, created); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	nodeRows := make(map[string]CSVThing)
	for node, val := range nodeResults {
		val["node_labels"] = labelResults[node]["labels"]
		row, err := json.Marshal(val)
		if err != nil {
			return fmt.Errorf("failed to marshal node labels")
		}
		usage := NewNodeRow(start)
		if err := json.Unmarshal(row, &usage); err != nil {
			return fmt.Errorf("failed to unmarshal node row")
		}
		nodeRows[node] = usage
	}
	nodeCSVFile, created, err := getOrCreateFile(dataPath, nodeFilePrefix+yearMonth+".csv")
	if err != nil {
		return fmt.Errorf("failed to get or create node csv: %v", err)
	}
	defer nodeCSVFile.Close()
	if err := writeToFile(nodeCSVFile, nodeRows, created); err != nil {
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
