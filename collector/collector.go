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
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/mitchellh/mapstructure"
	kokumetricscfgv1beta1 "github.com/project-koku/koku-metrics-operator/api/v1beta1"
	"github.com/project-koku/koku-metrics-operator/dirconfig"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

var (
	podFilePrefix       = "cm-openshift-pod-usage-"
	volFilePrefix       = "cm-openshift-storage-usage-"
	nodeFilePrefix      = "cm-openshift-node-usage-"
	namespaceFilePrefix = "cm-openshift-namespace-usage-"

	statusTimeFormat = "2006-01-02 15:04:05"
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

func getResourceID(input string) string {
	splitString := strings.Split(input, "/")
	return splitString[len(splitString)-1]
}

func (r *mappedResults) iterateMatrix(matrix model.Matrix, q query) {
	results := *r
	for _, stream := range matrix {
		obj := string(stream.Metric[q.RowKey])
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
				results[obj][saveStruct.TransformedName] = floatToString(value * float64(len(stream.Values)*saveStruct.Factor))
			}
		}
	}
}

// GenerateReports is responsible for querying prometheus and writing to report files
func GenerateReports(kmCfg *kokumetricscfgv1beta1.CostManagementMetricsConfig, dirCfg *dirconfig.DirectoryConfig, c *PromCollector) error {
	log := c.Log.WithValues("costmanagementmetricsconfig", "GenerateReports")

	// yearMonth is used in filenames
	yearMonth := c.TimeSeries.Start.Format("200601") // this corresponds to YYYYMM format
	updateReportStatus(kmCfg, c.TimeSeries)

	// ################################################################################################################
	log.Info("querying for node metrics")
	nodeResults := mappedResults{}
	if err := c.getQueryResults(nodeQueries, &nodeResults); err != nil {
		return err
	}

	if len(nodeResults) <= 0 {
		log.Info("no data to report")
		kmCfg.Status.Reports.DataCollected = false
		kmCfg.Status.Reports.DataCollectionMessage = "No data to report for the hour queried."
		// there is no data for the hour queried. Return nothing
		return nil
	}
	for node, val := range nodeResults {
		resourceID := getResourceID(val["provider_id"].(string))
		nodeResults[node]["resource_id"] = resourceID
	}

	nodeRows := make(mappedCSVStruct)
	for node, val := range nodeResults {
		usage := newNodeRow(c.TimeSeries)
		if err := getStruct(val, &usage, nodeRows, node); err != nil {
			return err
		}
	}
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
	c.Log.WithValues("costmanagementmetricsconfig", "writeResults").Info("writing node results to file", "filename", nodeReport.file.getName())
	if err := nodeReport.writeReport(); err != nil {
		return fmt.Errorf("failed to write node report: %v", err)
	}

	//################################################################################################################

	log.Info("querying for pod metrics")
	podResults := mappedResults{}
	if err := c.getQueryResults(podQueries, &podResults); err != nil {
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
	c.Log.WithValues("costmanagementmetricsconfig", "writeResults").Info("writing pod results to file", "filename", podReport.file.getName())
	if err := podReport.writeReport(); err != nil {
		return fmt.Errorf("failed to write pod report: %v", err)
	}

	//################################################################################################################

	log.Info("querying for storage metrics")
	volResults := mappedResults{}
	if err := c.getQueryResults(volQueries, &volResults); err != nil {
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
	c.Log.WithValues("costmanagementmetricsconfig", "writeResults").Info("writing volume results to file", "filename", volReport.file.getName())
	if err := volReport.writeReport(); err != nil {
		return fmt.Errorf("failed to write volume report: %v", err)
	}

	//################################################################################################################

	log.Info("querying for namespaces")
	namespaceResults := mappedResults{}
	if err := c.getQueryResults(namespaceQueries, &namespaceResults); err != nil {
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
	c.Log.WithValues("costmanagementmetricsconfig", "writeResults").Info("writing namespace results to file", "filename", namespaceReport.file.getName())
	if err := namespaceReport.writeReport(); err != nil {
		return fmt.Errorf("failed to write namespace report: %v", err)
	}

	//################################################################################################################

	kmCfg.Status.Reports.DataCollected = true
	kmCfg.Status.Reports.DataCollectionMessage = ""

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

func updateReportStatus(kmCfg *kokumetricscfgv1beta1.CostManagementMetricsConfig, ts *promv1.Range) {
	kmCfg.Status.Reports.ReportMonth = ts.Start.Format("01")
	kmCfg.Status.Reports.LastHourQueried = ts.Start.Format(statusTimeFormat) + " - " + ts.End.Format(statusTimeFormat)
}
