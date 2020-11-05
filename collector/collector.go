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
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/mitchellh/mapstructure"
	costmgmtv1alpha1 "github.com/project-koku/korekuta-operator-go/api/v1alpha1"
	"github.com/project-koku/korekuta-operator-go/dirconfig"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

var (
	logger logr.Logger

	podFilePrefix       = "cm-openshift-pod-usage-"
	volFilePrefix       = "cm-openshift-storage-usage-"
	nodeFilePrefix      = "cm-openshift-node-usage-"
	namespaceFilePrefix = "cm-openshift-namespace-usage-"

	statusTimeFormat = "2006-01-02 15:04:05"
)

type prometheusConnection interface {
	QueryRange(ctx context.Context, query string, r promv1.Range) (model.Value, promv1.Warnings, error)
}

type mappedCSVStruct map[string]CSVStruct
type mappedQuery map[string]string
type mappedResults map[string]mappedValues
type mappedValues map[string]interface{}
type collector struct {
	context    context.Context
	promConn   prometheusConnection
	timeSeries promv1.Range
	log        logr.Logger
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

func getValue(query *SaveQueryValue, array []model.SamplePair) float64 {
	switch query.Method {
	case "sum":
		return sumSlice(array)
	case "max":
		return maxSlice(array)
	default:
		return 0
	}
}

func getStruct(val mappedValues, usage CSVStruct, rowResults mappedCSVStruct, key string) error {
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

func getQueryResults(q collector, queries Querys) (mappedResults, error) {
	results := mappedResults{}
	for _, query := range queries {
		matrix, err := performMatrixQuery(q, query.QueryString)
		if err != nil {
			return nil, fmt.Errorf("getQueryResults: %v", err)
		}
		results = iterateMatrix(matrix, query, results)
	}
	return results, nil
}

func iterateMatrix(matrix model.Matrix, q Query, results mappedResults) mappedResults {
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
	return results
}

// GenerateReports is responsible for querying prometheus and writing to report files
func GenerateReports(cost *costmgmtv1alpha1.CostManagement, dirCfg *dirconfig.DirectoryConfig, promconn promv1.API, ts promv1.Range, log logr.Logger) error {
	if logger == nil {
		logger = log
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	log = log.WithValues("costmanagement", "GenerateReports")
	defer cancel()

	querier := collector{
		context:    ctx,
		promConn:   promconn,
		timeSeries: ts,
		log:        log,
	}

	// yearMonth is used in filenames
	yearMonth := ts.Start.Format("200601") // this corresponds to YYYYMM format
	updateReportStatus(cost, ts)

	log.Info("querying for node metrics")
	nodeResults, err := getQueryResults(querier, nodeQueries)
	if err != nil {
		return err
	}

	if len(nodeResults) <= 0 {
		log.Info("no data to report")
		cost.Status.Reports.DataCollected = false
		cost.Status.Reports.DataCollectionMessage = "No data to report for the hour queried."
		// there is no data for the hour queried. Return nothing
		return nil
	}
	for node, val := range nodeResults {
		resourceID := getResourceID(val["provider_id"].(string))
		nodeResults[node]["resource_id"] = resourceID
	}

	log.Info("querying for pod metrics")
	podResults, err := getQueryResults(querier, podQueries)
	if err != nil {
		return err
	}

	log.Info("querying for storage metrics")
	volResults, err := getQueryResults(querier, volQueries)
	if err != nil {
		return err
	}

	log.Info("querying for namespaces")
	namespaceResults, err := getQueryResults(querier, namespaceQueries)
	if err != nil {
		return err
	}

	nodeRows := make(mappedCSVStruct)
	for node, val := range nodeResults {
		usage := NewNodeRow(ts)
		if err := getStruct(val, &usage, nodeRows, node); err != nil {
			return err
		}
	}
	nodeReport := reportFile{
		name:      nodeFilePrefix + yearMonth + ".csv",
		path:      dirCfg.Reports.Path,
		queryType: "node",
		queryData: nodeRows,
		headers:   NewNodeRow(ts),
	}
	if err := nodeReport.writeReport(); err != nil {
		return err
	}

	podRows := make(mappedCSVStruct)
	for pod, val := range podResults {
		usage := NewPodRow(ts)
		if err := getStruct(val, &usage, podRows, pod); err != nil {
			return err
		}
		if node, ok := val["node"]; ok {
			// Add the Node usage to the pod.
			if row, ok := nodeRows[node.(string)]; ok {
				usage.NodeRow = *row.(*NodeRow)
			} else {
				usage.NodeRow = NewNodeRow(ts)
			}
		}
	}
	podReport := reportFile{
		name:      podFilePrefix + yearMonth + ".csv",
		path:      dirCfg.Reports.Path,
		queryType: "pod",
		queryData: podRows,
		headers:   NewPodRow(ts),
	}
	if err := podReport.writeReport(); err != nil {
		return err
	}

	volRows := make(mappedCSVStruct)
	for pvc, val := range volResults {
		usage := NewStorageRow(ts)
		if err := getStruct(val, &usage, volRows, pvc); err != nil {
			return err
		}
	}
	volReport := reportFile{
		name:      volFilePrefix + yearMonth + ".csv",
		path:      dirCfg.Reports.Path,
		queryType: "volume",
		queryData: volRows,
		headers:   NewStorageRow(ts),
	}
	if err := volReport.writeReport(); err != nil {
		return err
	}

	namespaceRows := make(mappedCSVStruct)
	for namespace, val := range namespaceResults {
		usage := NewNamespaceRow(ts)
		if err := getStruct(val, &usage, namespaceRows, namespace); err != nil {
			return err
		}
	}
	namespaceReport := reportFile{
		name:      namespaceFilePrefix + yearMonth + ".csv",
		path:      dirCfg.Reports.Path,
		queryType: "namespace",
		queryData: namespaceRows,
		headers:   NewNamespaceRow(ts),
	}
	if err := namespaceReport.writeReport(); err != nil {
		return err
	}

	cost.Status.Reports.DataCollected = true
	cost.Status.Reports.DataCollectionMessage = ""

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

func updateReportStatus(cost *costmgmtv1alpha1.CostManagement, ts promv1.Range) {
	cost.Status.Reports.ReportMonth = ts.Start.Format("01")
	cost.Status.Reports.LastHourQueried = ts.Start.Format(statusTimeFormat) + " - " + ts.End.Format(statusTimeFormat)
}
