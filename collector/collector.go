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
	costmgmtv1alpha1 "github.com/project-koku/korekuta-operator-go/api/v1alpha1"
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

	statusTimeFormat = "2006-01-02 15:04:05"
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

func iterateMatrix(matrix model.Matrix, q Query, results mappedResults) mappedResults {
	for _, stream := range matrix {
		obj := string(stream.Metric[q.RowKey])
		if results[obj] == nil {
			results[obj] = mappedValues{}
		}
		if q.MetricKey != nil {
			for i, field := range q.MetricKey.MetricLabel {
				index := string(field)
				if len(q.MetricKey.LabelMap) > 0 {
					index = q.MetricKey.LabelMap[i]
				}
				results[obj][index] = string(stream.Metric[field])
			}
		}
		if q.MetricKeyRegex != nil {
			for i, field := range q.MetricKeyRegex.LabelMap {
				results[obj][field] = parseFields(stream.Metric, q.MetricKeyRegex.MetricRegex[i])
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

func getQueryResults(q collector, queries Querys) (mappedResults, error) {
	results := mappedResults{}
	for _, query := range queries {
		matrix, err := performMatrixQuery(q, query.QueryString)
		if err != nil {
			return nil, err
		}
		results = iterateMatrix(matrix, query, results)
	}
	return results, nil
}

// GenerateReports is responsible for querying prometheus and writing to report files
func GenerateReports(cost *costmgmtv1alpha1.CostManagement, promconn promv1.API, ts promv1.Range, log logr.Logger) error {
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
	updateReportStatus(cost, ts)

	log.Info("querying for node metrics")
	nodeResults, err := getQueryResults(querier, nodeQueries)
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
	if err := writeResults(nodeFilePrefix, yearMonth, "node", nodeRows); err != nil {
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
	if err := writeResults(podFilePrefix, yearMonth, "pod", podRows); err != nil {
		return err
	}

	volRows := make(mappedCSVStruct)
	for pvc, val := range volResults {
		usage := NewStorageRow(ts)
		if err := getStruct(val, &usage, volRows, pvc); err != nil {
			return err
		}
	}
	if err := writeResults(volFilePrefix, yearMonth, "volume", volRows); err != nil {
		return err
	}

	namespaceRows := make(mappedCSVStruct)
	for namespace, val := range namespaceResults {
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

func parseFields(input model.Metric, str string) string {
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

func updateReportStatus(cost *costmgmtv1alpha1.CostManagement, ts promv1.Range) {
	cost.Status.Reports.ReportMonth = ts.Start.Format("01")
	cost.Status.Reports.LastHourQueried = ts.Start.Format(statusTimeFormat) + " - " + ts.End.Format(statusTimeFormat)
}
