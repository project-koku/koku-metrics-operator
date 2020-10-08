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
	promapi "github.com/prometheus/client_golang/api"
	prom "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	dataPath            = "/tmp/cost-mgmt-operator-reports/data/"
	podFilePrefix       = "cm-openshift-usage-lookback-"
	volFilePrefix       = "cm-openshift-persistentvolumeclaim-lookback-"
	nodeFilePrefix      = "cm-openshift-node-labels-lookback-"
	namespaceFilePrefix = "cm-openshift-namespace-labels-lookback-"

	defaultPromHost = "https://thanos-querier.openshift-monitoring.svc:9091/"
	address         = defaultPromHost // the URL string for connecting to Prometheus
	nodeQueries     = map[string]string{
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
	// "cm-kube-namespace-labels":               "kube_namespace_labels",
	// "cm-kube-node-labels":                    "kube_node_labels",
	// "cm-kube-persistentvolume-labels":        "kube_persistentvolume_labels",
	// "cm-kube-persistentvolumeclaim-labels":   "kube_persistentvolumeclaim_labels",
	podLabelQuery = map[string]string{"cm-kube-pod-labels": "kube_pod_labels"}
	// "cm-kube-pod-persistentvolumeclaim-info": "kube_pod_spec_volumes_persistentvolumeclaims_info",

)

// PrometheusConfig provides the configuration options to set up a Prometheus connections from a URL.
type PrometheusConfig struct {
	// Address is the URL to reach Prometheus.
	Address string
	// BearerToken is the user auth token
	BearerToken config.Secret
	// CAFile is the ca file
	CAFile string

	log logr.Logger
}

func GetPromConn(ctx context.Context, log logr.Logger) (prom.API, error) {
	cfg := &PrometheusConfig{
		Address: "https://thanos-querier-openshift-monitoring.apps.cluster-9071.9071.sandbox1249.opentlc.com",
		// TODO: do not hardcode BearerToken, CAFile
		BearerToken: "eyJhbGciOiJSUzI1NiIsImtpZCI6IkFnTUphQU44Z1k3QzBDSmRndlJpV3h5Wm5jSVkyZlNQMDBDMG5BeGhjZGcifQ.eyJpc3MiOiJrdWJlcm5ldGVzL3NlcnZpY2VhY2NvdW50Iiwia3ViZXJuZXRlcy5pby9zZXJ2aWNlYWNjb3VudC9uYW1lc3BhY2UiOiJvcGVuc2hpZnQtY29zdCIsImt1YmVybmV0ZXMuaW8vc2VydmljZWFjY291bnQvc2VjcmV0Lm5hbWUiOiJkZWZhdWx0LXRva2VuLWZ3NzdoIiwia3ViZXJuZXRlcy5pby9zZXJ2aWNlYWNjb3VudC9zZXJ2aWNlLWFjY291bnQubmFtZSI6ImRlZmF1bHQiLCJrdWJlcm5ldGVzLmlvL3NlcnZpY2VhY2NvdW50L3NlcnZpY2UtYWNjb3VudC51aWQiOiI3ODBkMzUyNC01ODZjLTQ0MWUtODUwZC04ZmZmMTEyM2IzODkiLCJzdWIiOiJzeXN0ZW06c2VydmljZWFjY291bnQ6b3BlbnNoaWZ0LWNvc3Q6ZGVmYXVsdCJ9.PXKnhEQOAJTpA0snPKOegt0sN-3K9ppl6PJxebHGSH19_3Jv_CBdRbHfVizkTtQUGnxJwsEJbGL0GSkPAOJvEO5SZM16N3gBXU7F-ccpUTvy_h41U7fnP1abYsSs8xet6sSNx6BG2oRZg8LinXu8xisPLy3ShlcIflfDjn-y49rhDTHwcdAvQxqgwMyRlVGIRNBhsf_XHfHowt0rfggPnNhPQsPUhPICbKCWry27ALuhbXmaruLkL8HlAQE2ieccjj4HiwaPow6g_1a8_5U5zVkqQY3-w48TLADIJOd1vODwKsZ8RZLbWFjwKP04NS3x8Gb5o4y5P-2Toes4gBHrOA",
		CAFile:      "/etc/ssl/cert.pem", // this file is wrong
		log:         log,
	}
	promConn, err := newPrometheusConnFromCFG(*cfg)
	if err != nil {
		return nil, fmt.Errorf("can't connect to prometheus: %v", err)
	}

	log.Info("testing the ability to query prometheus")

	err = wait.Poll(3*time.Second, 15*time.Second, func() (bool, error) {
		_, _, err := promConn.Query(context.TODO(), "up", time.Now())
		if err != nil {
			return false, fmt.Errorf("failed to succesfully query prometheus: %v", err)
		}
		log.Info("prometheus queries are succeeding")
		return true, err
	})
	if err != nil {
		return nil, fmt.Errorf("prometheus queries are failing: %v", err)
	}

	return promConn, nil
}

func newPrometheusConnFromCFG(cfg PrometheusConfig) (prom.API, error) {
	cfg.log.Info("configuring prometheus client")
	promconf := config.HTTPClientConfig{
		BearerToken: cfg.BearerToken,
		TLSConfig:   config.TLSConfig{CAFile: cfg.CAFile, InsecureSkipVerify: true},
	}
	roundTripper, err := config.NewRoundTripperFromConfig(promconf, "promconf", false, false)
	if err != nil {
		return nil, fmt.Errorf("can't create roundTripper: %v", err)
	}
	client, err := promapi.NewClient(promapi.Config{
		Address:      cfg.Address,
		RoundTripper: roundTripper,
	})
	if err != nil {
		return nil, fmt.Errorf("can't connect to prometheus: %v", err)
	}
	return prom.NewAPI(client), nil
}

func DoQuery(promconn prom.API) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t := time.Now()
	start := time.Date(t.Year(), t.Month(), t.Day(), t.Hour()-1, 59, 59, 0, t.Location())
	yearMonth := start.Format("200601") // because Go is weird, this corresponds to YYYYMM format
	defer cancel()
	// r := prom.Range{
	// 	Start: time.Now().Add(-time.Hour),
	// 	End:   time.Now(),
	// 	Step:  time.Minute,
	// }
	var nodeResults = map[string]map[string]interface{}{}
	for qname, query := range nodeQueries {
		vector, err := performTheQuery(ctx, promconn, query, start)
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
			nodeResults[node][qname+"-seconds"] = floatToString(float64(val.Value) * 3600)
			nodeResults[node]["timestamp"] = val.Timestamp.Time()
		}
	}
	if len(nodeResults) <= 0 {
		return fmt.Errorf("collector: no data to report")
	}

	var podResults = map[string]map[string]interface{}{}
	for qname, query := range podQueries {
		vector, err := performTheQuery(ctx, promconn, query, start)
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
			podResults[pod][qname+"-seconds"] = floatToString(float64(val.Value) * 3600)
			podResults[pod]["timestamp"] = val.Timestamp.Time()
		}
	}

	var labelResults = map[string]map[string]interface{}{}
	for _, labelQuery := range labelQueries {
		label, query := labelQuery[0], labelQuery[1]
		vector, err := performTheQuery(ctx, promconn, query, start)
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
		vector, err := performTheQuery(ctx, promconn, query, start)
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
			volResults[pvc][qname+"-seconds"] = floatToString(float64(val.Value) * 3600)
			volResults[pvc]["timestamp"] = val.Timestamp.Time()
		}
	}

	podRows := make(map[string]CSVThing)
	for pod, val := range podResults {
		node := val["node"].(string)
		dict, ok := nodeResults[string(node)]
		if !ok {
			fmt.Printf("\n%+v\n", val)
			return fmt.Errorf("node %s not found", node)
		}
		val["node-capacity-cpu-cores"] = dict["node-capacity-cpu-cores"]
		val["node-capacity-cpu-cores-seconds"] = dict["node-capacity-cpu-cores-seconds"]
		val["node-capacity-memory-bytes"] = dict["node-capacity-memory-bytes"]
		val["node-capacity-memory-bytes-seconds"] = dict["node-capacity-memory-bytes-seconds"]
		val["resource_id"] = dict["resource_id"]
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
		fmt.Printf("Result for PVC: %v\n\tValue: %v\n\n", pvc, val)
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

func found(set map[string]bool, val string) bool {
	return set[val]
}

func readCsv(f *os.File, set *set) (*set, error) {

	// Read File into a Variable
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
	set, err := readCsv(file, NewSet())
	if err != nil {
		return fmt.Errorf("failed to read csv: %v", err)
	}
	if created {
		for _, row := range data {
			row.CSVheader(file)
			break // just get the first item in the map and write the headers
		}
	}

	for id, row := range data {
		fmt.Printf("Result for: %v\n\tValue: %+v\n\n", id, row)
		if !set.Contains(strings.Join(row.RowString(), ",")) {
			row.CSVrow(file)
		} else {
			fmt.Println("line already exists in file")
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
	fmt.Printf("%s file `%s` exists\n", time.Now().String(), filePath)
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_RDWR, 0644)
	return file, false, err
}

func getResourceID(input string) string {
	splitString := strings.Split(input, "/")
	return splitString[len(splitString)-1]
}

func performTheQuery(ctx context.Context, promconn prom.API, query string, ts time.Time) (model.Vector, error) {
	result, warnings, err := promconn.Query(ctx, query, ts)
	if err != nil {
		return nil, fmt.Errorf("error querying prometheus: %v", err)
	}
	if len(warnings) > 0 {
		fmt.Printf("Warnings: %v\n", warnings)
	}
	vector, ok := result.(model.Vector)
	if !ok {
		return nil, fmt.Errorf("expected a vector in response to query, got a %v", result.Type())
	}
	return vector, nil
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

// 2020-10-01 00:00:00 -0400 EDT,2020-11-01 00:00:00 -0400 EDT,2020-10-07 15:00:00 -0400 EDT,2020-10-07 15:59:59 -0400 EDT,ip-10-0-147-111.us-east-2.compute.internal,4.000000,14400.000000,16326778880.000000,58776403968000.000000,i-0dd2eab633b37c882,label_failure_domain_beta_kubernetes_io_region:us-east-2|label_kubernetes_io_arch:amd64|label_kubernetes_io_os:linux|label_node_kubernetes_io_instance_type:m5.xlarge|label_node_openshift_io_os_id:rhcos|label_beta_kubernetes_io_arch:amd64|label_beta_kubernetes_io_os:linux|label_topology_kubernetes_io_region:us-east-2|label_beta_kubernetes_io_instance_type:m5.xlarge|label_failure_domain_beta_kubernetes_io_zone:us-east-2a|label_kubernetes_io_hostname:ip-10-0-147-111|label_topology_kubernetes_io_zone:us-east-2a
// 2020-10-01 00:00:00 -0400 EDT,2020-11-01 00:00:00 -0400 EDT,2020-10-07 15:00:00 -0400 EDT,2020-10-07 15:59:59 -0400 EDT,ip-10-0-147-111.us-east-2.compute.internal,4.000000,14400.000000,16326778880.000000,58776403968000.000000,i-0dd2eab633b37c882,label_kubernetes_io_os:linux|label_beta_kubernetes_io_arch:amd64|label_failure_domain_beta_kubernetes_io_zone:us-east-2a|label_node_kubernetes_io_instance_type:m5.xlarge|label_node_openshift_io_os_id:rhcos|label_topology_kubernetes_io_region:us-east-2|label_beta_kubernetes_io_os:linux|label_failure_domain_beta_kubernetes_io_region:us-east-2|label_kubernetes_io_arch:amd64|label_beta_kubernetes_io_instance_type:m5.xlarge|label_kubernetes_io_hostname:ip-10-0-147-111|label_topology_kubernetes_io_zone:us-east-2a
