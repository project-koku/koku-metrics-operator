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

import "github.com/prometheus/common/model"

const (
	maxFactor int = 60
	sumFactor int = 1
)

var (
	nodeQueries = Querys{
		Query{
			Name:        "node-allocatable-cpu-cores",
			QueryString: "kube_node_status_allocatable_cpu_cores * on(node) group_left(provider_id) max(kube_node_info) by (node, provider_id)",
			MetricKey:   &StaticFields{MetricLabel: []model.LabelName{"namespace", "node", "provider_id"}},
			QueryValue: &SaveQueryValue{
				ValName:         "node-allocatable-cpu-cores",
				Method:          "max",
				Factor:          maxFactor,
				TransformedName: "node-allocatable-cpu-core-seconds",
			},
			RowKey: "node",
		},
		Query{
			Name:        "node-allocatable-memory-bytes",
			QueryString: "kube_node_status_allocatable_memory_bytes * on(node) group_left(provider_id) max(kube_node_info) by (node, provider_id)",
			MetricKey:   &StaticFields{MetricLabel: []model.LabelName{"namespace", "node", "provider_id"}},
			QueryValue: &SaveQueryValue{
				ValName:         "node-allocatable-memory-bytes",
				Method:          "max",
				Factor:          maxFactor,
				TransformedName: "node-allocatable-memory-byte-seconds",
			},
			RowKey: "node",
		},
		Query{
			Name:        "node-capacity-cpu-cores",
			QueryString: "kube_node_status_capacity_cpu_cores * on(node) group_left(provider_id) max(kube_node_info) by (node, provider_id)",
			MetricKey:   &StaticFields{MetricLabel: []model.LabelName{"namespace", "node", "provider_id"}},
			QueryValue: &SaveQueryValue{
				ValName:         "node-capacity-cpu-cores",
				Method:          "max",
				Factor:          maxFactor,
				TransformedName: "node-capacity-cpu-core-seconds",
			},
			RowKey: "node",
		},
		Query{
			Name:        "node-capacity-memory-bytes",
			QueryString: "kube_node_status_capacity_memory_bytes * on(node) group_left(provider_id) max(kube_node_info) by (node, provider_id)",
			MetricKey:   &StaticFields{MetricLabel: []model.LabelName{"namespace", "node", "provider_id"}},
			QueryValue: &SaveQueryValue{
				ValName:         "node-capacity-memory-bytes",
				Method:          "max",
				Factor:          maxFactor,
				TransformedName: "node-capacity-memory-byte-seconds",
			},
			RowKey: "node",
		},
		Query{
			Name:        "node-labels",
			QueryString: "kube_node_labels",
			MetricKeyRegex: &RegexFields{
				MetricRegex: []string{"label_*"},
				LabelMap:    []string{"node_labels"}},
			RowKey: "node",
		},
	}
	volQueries = Querys{
		Query{
			Name:        "persistentvolume_pod_info",
			QueryString: "kube_pod_spec_volumes_persistentvolumeclaims_info * on(persistentvolumeclaim) group_left(storageclass, volumename) kube_persistentvolumeclaim_info",
			MetricKey:   &StaticFields{MetricLabel: []model.LabelName{"namespace", "persistentvolumeclaim", "pod", "storageclass", "volumename"}},
			RowKey:      "volumename",
		},
		Query{
			Name:        "persistentvolumeclaim-capacity-bytes",
			QueryString: "kubelet_volume_stats_capacity_bytes * on(persistentvolumeclaim) group_left(volumename) kube_persistentvolumeclaim_info",
			QueryValue: &SaveQueryValue{
				ValName:         "persistentvolumeclaim-capacity-bytes",
				Method:          "max",
				Factor:          sumFactor,
				TransformedName: "persistentvolumeclaim-capacity-byte-seconds",
			},
			RowKey: "volumename",
		},
		Query{
			Name:        "persistentvolumeclaim-request-bytes",
			QueryString: "kube_persistentvolumeclaim_resource_requests_storage_bytes * on(persistentvolumeclaim) group_left(volumename) kube_persistentvolumeclaim_info",
			QueryValue: &SaveQueryValue{
				ValName:         "persistentvolumeclaim-request-bytes",
				Method:          "max",
				Factor:          sumFactor,
				TransformedName: "persistentvolumeclaim-request-byte-seconds",
			},
			RowKey: "volumename",
		},
		Query{
			Name:        "persistentvolumeclaim-usage-bytes",
			QueryString: "kubelet_volume_stats_used_bytes * on(persistentvolumeclaim) group_left(volumename) kube_persistentvolumeclaim_info",
			QueryValue: &SaveQueryValue{
				ValName:         "persistentvolumeclaim-usage-bytes",
				Method:          "sum",
				Factor:          sumFactor,
				TransformedName: "persistentvolumeclaim-usage-byte-seconds",
			},
			RowKey: "volumename",
		},
		Query{
			Name:        "persistentvolume-labels",
			QueryString: "kube_persistentvolume_labels",
			MetricKeyRegex: &RegexFields{
				MetricRegex: []string{"label_*"},
				LabelMap:    []string{"persistentvolume_labels"}},
			RowKey: "persistentvolume",
		},
		Query{
			Name:        "persistentvolumeclaim-labels",
			QueryString: "kube_persistentvolumeclaim_labels * on(persistentvolumeclaim) group_left(volumename) kube_persistentvolumeclaim_info",
			MetricKeyRegex: &RegexFields{
				MetricRegex: []string{"label_*"},
				LabelMap:    []string{"persistentvolumeclaim_labels"}},
			RowKey: "volumename",
		},
	}
	podQueries = Querys{
		Query{
			Name:        "pod-limit-cpu-cores",
			QueryString: "sum(kube_pod_container_resource_limits_cpu_cores) by (pod, namespace, node)",
			MetricKey:   &StaticFields{MetricLabel: []model.LabelName{"pod", "namespace", "node"}},
			QueryValue: &SaveQueryValue{
				ValName:         "pod-limit-cpu-cores",
				Method:          "sum",
				Factor:          sumFactor,
				TransformedName: "pod-limit-cpu-core-seconds",
			},
			RowKey: "pod",
		},
		Query{
			Name:        "pod-limit-memory-bytes",
			QueryString: "sum(kube_pod_container_resource_limits_memory_bytes) by (pod, namespace, node)",
			MetricKey:   &StaticFields{MetricLabel: []model.LabelName{"pod", "namespace", "node"}},
			QueryValue: &SaveQueryValue{
				ValName:         "pod-limit-memory-bytes",
				Method:          "sum",
				Factor:          sumFactor,
				TransformedName: "pod-limit-memory-byte-seconds",
			},
			RowKey: "pod",
		},
		Query{
			Name:        "pod-request-cpu-cores",
			QueryString: "sum(kube_pod_container_resource_requests_cpu_cores) by (pod, namespace, node)",
			MetricKey:   &StaticFields{MetricLabel: []model.LabelName{"pod", "namespace", "node"}},
			QueryValue: &SaveQueryValue{
				ValName:         "pod-request-cpu-cores",
				Method:          "sum",
				Factor:          sumFactor,
				TransformedName: "pod-request-cpu-core-seconds",
			},
			RowKey: "pod",
		},
		Query{
			Name:        "pod-request-memory-bytes",
			QueryString: "sum(kube_pod_container_resource_requests_memory_bytes) by (pod, namespace, node)",
			MetricKey:   &StaticFields{MetricLabel: []model.LabelName{"pod", "namespace", "node"}},
			QueryValue: &SaveQueryValue{
				ValName:         "pod-request-memory-bytes",
				Method:          "sum",
				Factor:          sumFactor,
				TransformedName: "pod-request-memory-byte-seconds",
			},
			RowKey: "pod",
		},
		Query{
			Name:        "pod-usage-cpu-cores",
			QueryString: "sum(rate(container_cpu_usage_seconds_total{container!='POD',container!='',pod!=''}[5m])) BY (pod, namespace, node)",
			MetricKey:   &StaticFields{MetricLabel: []model.LabelName{"pod", "namespace", "node"}},
			QueryValue: &SaveQueryValue{
				ValName:         "pod-usage-cpu-cores",
				Method:          "sum",
				Factor:          sumFactor,
				TransformedName: "pod-usage-cpu-core-seconds",
			},
			RowKey: "pod",
		},
		Query{
			Name:        "pod-usage-memory-bytes",
			QueryString: "sum(container_memory_usage_bytes{container!='POD', container!='',pod!=''}) by (pod, namespace, node)",
			MetricKey:   &StaticFields{MetricLabel: []model.LabelName{"pod", "namespace", "node"}},
			QueryValue: &SaveQueryValue{
				ValName:         "pod-usage-memory-bytes",
				Method:          "sum",
				Factor:          sumFactor,
				TransformedName: "pod-usage-memory-byte-seconds",
			},
			RowKey: "pod",
		},
		Query{
			Name:        "pod-labels",
			QueryString: "kube_pod_labels",
			MetricKeyRegex: &RegexFields{
				MetricRegex: []string{"label_*"},
				LabelMap:    []string{"pod_labels"}},
			RowKey: "pod",
		},
	}
	namespaceQueries = Querys{
		Query{
			Name:        "namespace-labels",
			QueryString: "kube_namespace_labels",
			MetricKey: &StaticFields{
				MetricLabel: []model.LabelName{"namespace"},
				LabelMap:    []string{"namespace"}},
			MetricKeyRegex: &RegexFields{
				MetricRegex: []string{"label_*"},
				LabelMap:    []string{"namespace_labels"}},
			RowKey: "namespace",
		},
	}
)

type Query struct {
	Name           string
	QueryString    string
	MetricKey      *StaticFields
	MetricKeyRegex *RegexFields
	QueryValue     *SaveQueryValue
	RowKey         model.LabelName
}

type Querys []Query

type StaticFields struct {
	MetricLabel []model.LabelName
	LabelMap    []string
}

type RegexFields struct {
	MetricRegex []string
	LabelMap    []string
}

type SaveQueryValue struct {
	ValName         string
	Method          string
	Factor          int
	TransformedName string
}
