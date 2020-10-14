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

var (
	nodeQueries = Querys{
		Query{
			Name:        "node-allocatable-cpu-cores",
			QueryString: "kube_node_status_allocatable_cpu_cores * on(node) group_left(provider_id) max(kube_node_info) by (node, provider_id)",
			Fields:      []model.LabelName{"namespace", "node", "provider_id"},
			MetricName:  "node-allocatable-cpu-cores",
			Key:         "node",
		},
		Query{
			Name:        "node-allocatable-memory-bytes",
			QueryString: "kube_node_status_allocatable_memory_bytes * on(node) group_left(provider_id) max(kube_node_info) by (node, provider_id)",
			Fields:      []model.LabelName{"namespace", "node", "provider_id"},
			MetricName:  "node-allocatable-memory-bytes",
			Key:         "node",
		},
		Query{
			Name:        "node-capacity-cpu-cores",
			QueryString: "kube_node_status_capacity_cpu_cores * on(node) group_left(provider_id) max(kube_node_info) by (node, provider_id)",
			Fields:      []model.LabelName{"namespace", "node", "provider_id"},
			MetricName:  "node-capacity-cpu-cores",
			Key:         "node",
		},
		Query{
			Name:        "node-capacity-memory-bytes",
			QueryString: "kube_node_status_capacity_memory_bytes * on(node) group_left(provider_id) max(kube_node_info) by (node, provider_id)",
			Fields:      []model.LabelName{"namespace", "node", "provider_id"},
			MetricName:  "node-capacity-memory-bytes",
			Key:         "node",
		},
		Query{
			Name:        "node-labels",
			QueryString: "kube_node_labels",
			Fields:      []model.LabelName{"label_*"},
			FieldsMap:   []string{"node_labels"},
			FieldRegex:  true,
			Key:         "node",
		},
	}
	volQueries = Querys{
		Query{
			Name:        "persistentvolume_pod_info",
			QueryString: "kube_pod_spec_volumes_persistentvolumeclaims_info * on(persistentvolumeclaim) group_left(storageclass, volumename) kube_persistentvolumeclaim_info",
			Fields:      []model.LabelName{"namespace", "persistentvolumeclaim", "pod", "storageclass", "volumename"},
			Key:         "volumename",
		},
		Query{
			Name:        "persistentvolumeclaim-capacity-bytes",
			QueryString: "kubelet_volume_stats_capacity_bytes * on(persistentvolumeclaim) group_left(volumename) kube_persistentvolumeclaim_info",
			MetricName:  "persistentvolumeclaim-capacity-bytes",
			Key:         "volumename",
		},
		Query{
			Name:        "persistentvolumeclaim-request-bytes",
			QueryString: "kube_persistentvolumeclaim_resource_requests_storage_bytes * on(persistentvolumeclaim) group_left(volumename) kube_persistentvolumeclaim_info",
			MetricName:  "persistentvolumeclaim-request-bytes",
			Key:         "volumename",
		},
		Query{
			Name:        "persistentvolumeclaim-usage-bytes",
			QueryString: "kubelet_volume_stats_used_bytes * on(persistentvolumeclaim) group_left(volumename) kube_persistentvolumeclaim_info",
			MetricName:  "persistentvolumeclaim-usage-bytes",
			Key:         "volumename",
		},
		Query{
			Name:        "persistentvolume-labels",
			QueryString: "kube_persistentvolume_labels",
			Fields:      []model.LabelName{"label_*"},
			FieldsMap:   []string{"persistentvolume_labels"},
			FieldRegex:  true,
			Key:         "persistentvolume",
		},
		Query{
			Name:        "persistentvolumeclaim-labels",
			QueryString: "kube_persistentvolumeclaim_labels * on(persistentvolumeclaim) group_left(volumename) kube_persistentvolumeclaim_info",
			Fields:      []model.LabelName{"label_*"},
			FieldsMap:   []string{"persistentvolumeclaim_labels"},
			FieldRegex:  true,
			Key:         "volumename",
		},
	}
	podQueries = Querys{
		Query{
			Name:        "pod-limit-cpu-cores",
			QueryString: "sum(kube_pod_container_resource_limits_cpu_cores) by (pod, namespace, node)",
			Fields:      []model.LabelName{"pod", "namespace", "node"},
			MetricName:  "pod-limit-cpu-cores",
			Key:         "pod",
		},
		Query{
			Name:        "pod-limit-memory-bytes",
			QueryString: "sum(kube_pod_container_resource_limits_memory_bytes) by (pod, namespace, node)",
			Fields:      []model.LabelName{"pod", "namespace", "node"},
			MetricName:  "pod-limit-cpu-cores",
			Key:         "pod",
		},
		Query{
			Name:        "pod-request-cpu-cores",
			QueryString: "sum(kube_pod_container_resource_requests_cpu_cores) by (pod, namespace, node)",
			Fields:      []model.LabelName{"pod", "namespace", "node"},
			MetricName:  "pod-request-cpu-cores",
			Key:         "pod",
		},
		Query{
			Name:        "pod-request-memory-bytes",
			QueryString: "sum(kube_pod_container_resource_requests_memory_bytes) by (pod, namespace, node)",
			Fields:      []model.LabelName{"pod", "namespace", "node"},
			MetricName:  "pod-request-memory-bytes",
			Key:         "pod",
		},
		Query{
			Name:        "pod-usage-cpu-cores",
			QueryString: "sum(rate(container_cpu_usage_seconds_total{container!='POD',container!='',pod!=''}[5m])) BY (pod, namespace, node)",
			Fields:      []model.LabelName{"pod", "namespace", "node"},
			MetricName:  "pod-usage-cpu-cores",
			Key:         "pod",
		},
		Query{
			Name:        "pod-usage-memory-bytes",
			QueryString: "sum(container_memory_usage_bytes{container!='POD', container!='',pod!=''}) by (pod, namespace, node)",
			Fields:      []model.LabelName{"pod", "namespace", "node"},
			MetricName:  "pod-usage-memory-bytes",
			Key:         "pod",
		},
		Query{
			Name:        "pod-labels",
			QueryString: "kube_pod_labels",
			Fields:      []model.LabelName{"label_*"},
			FieldsMap:   []string{"pod_labels"},
			FieldRegex:  true,
			Key:         "pod",
		},
	}
	namespaceQueries = Querys{
		Query{
			Name:        "namespace-labels",
			QueryString: "kube_namespace_labels",
			Fields:      []model.LabelName{"label_*", "namespace"},
			FieldsMap:   []string{"namespace_labels", "namespace"},
			FieldRegex:  true,
			Key:         "namespace",
		},
	}
)

type Query struct {
	Name        string
	QueryString string
	Fields      []model.LabelName
	FieldsMap   []string
	FieldRegex  bool
	MetricName  string
	Key         model.LabelName
}

type Querys []Query
