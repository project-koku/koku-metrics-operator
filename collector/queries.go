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
	nodeQueries = &querys{
		query{
			Name:        "node-allocatable-cpu-cores",
			QueryString: "kube_node_status_allocatable_cpu_cores * on(node) group_left(provider_id) max(kube_node_info) by (node, provider_id)",
			MetricKey:   staticFields{"node": "node", "provider_id": "provider_id"},
			QueryValue: &saveQueryValue{
				ValName:         "node-allocatable-cpu-cores",
				Method:          "max",
				Factor:          maxFactor,
				TransformedName: "node-allocatable-cpu-core-seconds",
			},
			RowKey: "node",
		},
		query{
			Name:        "node-allocatable-memory-bytes",
			QueryString: "kube_node_status_allocatable_memory_bytes * on(node) group_left(provider_id) max(kube_node_info) by (node, provider_id)",
			MetricKey:   staticFields{"node": "node", "provider_id": "provider_id"},
			QueryValue: &saveQueryValue{
				ValName:         "node-allocatable-memory-bytes",
				Method:          "max",
				Factor:          maxFactor,
				TransformedName: "node-allocatable-memory-byte-seconds",
			},
			RowKey: "node",
		},
		query{
			Name:        "node-capacity-cpu-cores",
			QueryString: "kube_node_status_capacity_cpu_cores * on(node) group_left(provider_id) max(kube_node_info) by (node, provider_id)",
			MetricKey:   staticFields{"node": "node", "provider_id": "provider_id"},
			QueryValue: &saveQueryValue{
				ValName:         "node-capacity-cpu-cores",
				Method:          "max",
				Factor:          maxFactor,
				TransformedName: "node-capacity-cpu-core-seconds",
			},
			RowKey: "node",
		},
		query{
			Name:        "node-capacity-memory-bytes",
			QueryString: "kube_node_status_capacity_memory_bytes * on(node) group_left(provider_id) max(kube_node_info) by (node, provider_id)",
			MetricKey:   staticFields{"node": "node", "provider_id": "provider_id"},
			QueryValue: &saveQueryValue{
				ValName:         "node-capacity-memory-bytes",
				Method:          "max",
				Factor:          maxFactor,
				TransformedName: "node-capacity-memory-byte-seconds",
			},
			RowKey: "node",
		},
		query{
			Name:           "node-labels",
			QueryString:    "kube_node_labels",
			MetricKeyRegex: regexFields{"node_labels": "label_*"},
			RowKey:         "node",
		},
	}
	volQueries = &querys{
		query{
			Name:        "persistentvolume_pod_info",
			QueryString: "kube_pod_spec_volumes_persistentvolumeclaims_info * on(persistentvolumeclaim, namespace) group_left(volumename) kube_persistentvolumeclaim_info",
			MetricKey:   staticFields{"namespace": "namespace", "pod": "pod"},
			RowKey:      "volumename",
		},
		query{
			Name:        "persistentvolumeclaim-capacity-bytes",
			QueryString: "kubelet_volume_stats_capacity_bytes * on(persistentvolumeclaim, namespace) group_left(volumename) kube_persistentvolumeclaim_info",
			QueryValue: &saveQueryValue{
				ValName:         "persistentvolumeclaim-capacity-bytes",
				Method:          "max",
				Factor:          maxFactor,
				TransformedName: "persistentvolumeclaim-capacity-byte-seconds",
			},
			RowKey: "volumename",
		},
		query{
			Name:        "persistentvolumeclaim-request-bytes",
			QueryString: "kube_persistentvolumeclaim_resource_requests_storage_bytes * on(persistentvolumeclaim, namespace) group_left(volumename) kube_persistentvolumeclaim_info",
			QueryValue: &saveQueryValue{
				ValName:         "persistentvolumeclaim-request-bytes",
				Method:          "max",
				Factor:          maxFactor,
				TransformedName: "persistentvolumeclaim-request-byte-seconds",
			},
			RowKey: "volumename",
		},
		query{
			Name:        "persistentvolumeclaim-usage-bytes",
			QueryString: "kubelet_volume_stats_used_bytes * on(persistentvolumeclaim, namespace) group_left(volumename) kube_persistentvolumeclaim_info",
			QueryValue: &saveQueryValue{
				ValName:         "persistentvolumeclaim-usage-bytes",
				Method:          "sum",
				Factor:          sumFactor,
				TransformedName: "persistentvolumeclaim-usage-byte-seconds",
			},
			RowKey: "volumename",
		},
		query{
			Name:           "persistentvolume-labels",
			QueryString:    "kube_persistentvolume_labels * on(persistentvolume, namespace) group_left(storageclass) kube_persistentvolume_info",
			MetricKey:      staticFields{"storageclass": "storageclass", "persistentvolume": "persistentvolume"},
			MetricKeyRegex: regexFields{"persistentvolume_labels": "label_*"},
			RowKey:         "persistentvolume",
		},
		query{
			Name:           "persistentvolumeclaim-labels",
			QueryString:    "kube_persistentvolumeclaim_labels * on(persistentvolumeclaim, namespace) group_left(volumename) kube_persistentvolumeclaim_info",
			MetricKey:      staticFields{"namespace": "namespace", "persistentvolumeclaim": "persistentvolumeclaim"},
			MetricKeyRegex: regexFields{"persistentvolumeclaim_labels": "label_"},
			RowKey:         "volumename",
		},
	}
	podQueries = &querys{
		query{
			Name:        "pod-limit-cpu-cores",
			QueryString: "sum(kube_pod_container_resource_limits_cpu_cores) by (pod, namespace, node)",
			MetricKey:   staticFields{"pod": "pod", "namespace": "namespace", "node": "node"},
			QueryValue: &saveQueryValue{
				ValName:         "pod-limit-cpu-cores",
				Method:          "sum",
				Factor:          sumFactor,
				TransformedName: "pod-limit-cpu-core-seconds",
			},
			RowKey: "pod",
		},
		query{
			Name:        "pod-limit-memory-bytes",
			QueryString: "sum(kube_pod_container_resource_limits_memory_bytes) by (pod, namespace, node)",
			MetricKey:   staticFields{"pod": "pod", "namespace": "namespace", "node": "node"},
			QueryValue: &saveQueryValue{
				ValName:         "pod-limit-memory-bytes",
				Method:          "sum",
				Factor:          sumFactor,
				TransformedName: "pod-limit-memory-byte-seconds",
			},
			RowKey: "pod",
		},
		query{
			Name:        "pod-request-cpu-cores",
			QueryString: "sum(kube_pod_container_resource_requests_cpu_cores) by (pod, namespace, node)",
			MetricKey:   staticFields{"pod": "pod", "namespace": "namespace", "node": "node"},
			QueryValue: &saveQueryValue{
				ValName:         "pod-request-cpu-cores",
				Method:          "sum",
				Factor:          sumFactor,
				TransformedName: "pod-request-cpu-core-seconds",
			},
			RowKey: "pod",
		},
		query{
			Name:        "pod-request-memory-bytes",
			QueryString: "sum(kube_pod_container_resource_requests_memory_bytes) by (pod, namespace, node)",
			MetricKey:   staticFields{"pod": "pod", "namespace": "namespace", "node": "node"},
			QueryValue: &saveQueryValue{
				ValName:         "pod-request-memory-bytes",
				Method:          "sum",
				Factor:          sumFactor,
				TransformedName: "pod-request-memory-byte-seconds",
			},
			RowKey: "pod",
		},
		query{
			Name:        "pod-usage-cpu-cores",
			QueryString: "sum(rate(container_cpu_usage_seconds_total{container!='POD',container!='',pod!=''}[5m])) BY (pod, namespace, node)",
			MetricKey:   staticFields{"pod": "pod", "namespace": "namespace", "node": "node"},
			QueryValue: &saveQueryValue{
				ValName:         "pod-usage-cpu-cores",
				Method:          "sum",
				Factor:          sumFactor,
				TransformedName: "pod-usage-cpu-core-seconds",
			},
			RowKey: "pod",
		},
		query{
			Name:        "pod-usage-memory-bytes",
			QueryString: "sum(container_memory_usage_bytes{container!='POD', container!='',pod!=''}) by (pod, namespace, node)",
			MetricKey:   staticFields{"pod": "pod", "namespace": "namespace", "node": "node"},
			QueryValue: &saveQueryValue{
				ValName:         "pod-usage-memory-bytes",
				Method:          "sum",
				Factor:          sumFactor,
				TransformedName: "pod-usage-memory-byte-seconds",
			},
			RowKey: "pod",
		},
		query{
			Name:           "pod-labels",
			QueryString:    "kube_pod_labels",
			MetricKeyRegex: regexFields{"pod_labels": "label_*"},
			RowKey:         "pod",
		},
	}
	namespaceQueries = &querys{
		query{
			Name:           "namespace-labels",
			QueryString:    "kube_namespace_labels",
			MetricKey:      staticFields{"namespace": "namespace"},
			MetricKeyRegex: regexFields{"namespace_labels": "label_*"},
			RowKey:         "namespace",
		},
	}
)

type querys []query

type query struct {
	Name           string
	QueryString    string
	MetricKey      staticFields
	MetricKeyRegex regexFields
	QueryValue     *saveQueryValue
	RowKey         model.LabelName
}

type staticFields map[string]model.LabelName

type regexFields map[string]string

type saveQueryValue struct {
	ValName         string
	Method          string
	Factor          int
	TransformedName string
}
