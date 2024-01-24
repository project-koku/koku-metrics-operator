//
// Copyright 2021 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package collector

import "github.com/prometheus/common/model"

var (
	QueryMap = map[string]string{
		"cost:node_allocatable_cpu_cores":    "kube_node_status_allocatable{resource='cpu'} * on(node) group_left(provider_id) max by (node, provider_id) (kube_node_info) ",
		"cost:node_allocatable_memory_bytes": "kube_node_status_allocatable{resource='memory'} * on(node) group_left(provider_id) max by (node, provider_id) (kube_node_info)",
		"cost:node_capacity_cpu_cores":       "kube_node_status_capacity{resource='cpu'} * on(node) group_left(provider_id) max by (node, provider_id) (kube_node_info)",
		"cost:node_capacity_memory_bytes":    "kube_node_status_capacity{resource='memory'} * on(node) group_left(provider_id) max by (node, provider_id) (kube_node_info)",

		"cost:persistentvolume_pod_info":            "kube_pod_spec_volumes_persistentvolumeclaims_info * on(persistentvolumeclaim, namespace) group_left(volumename) max by(namespace, persistentvolumeclaim, volumename) (kube_persistentvolumeclaim_info{volumename != ''})",
		"cost:persistentvolumeclaim_capacity_bytes": "kubelet_volume_stats_capacity_bytes * on(persistentvolumeclaim, namespace) group_left(volumename) max by(namespace, persistentvolumeclaim, volumename) (kube_persistentvolumeclaim_info{volumename != ''})",
		"cost:persistentvolumeclaim_request_bytes":  "kube_persistentvolumeclaim_resource_requests_storage_bytes * on(persistentvolumeclaim, namespace) group_left(volumename) max by(namespace, persistentvolumeclaim, volumename) (kube_persistentvolumeclaim_info{volumename != ''})",
		"cost:persistentvolumeclaim_usage_bytes":    "kubelet_volume_stats_used_bytes * on(persistentvolumeclaim, namespace) group_left(volumename) max by(namespace, persistentvolumeclaim, volumename) (kube_persistentvolumeclaim_info{volumename != ''})",
		"cost:persistentvolume_labels":              "kube_persistentvolume_labels * on(persistentvolume, namespace) group_left(storageclass) max by(namespace, persistentvolume, storageclass) (kube_persistentvolume_info)",
		"cost:persistentvolumeclaim_labels":         "kube_persistentvolumeclaim_labels * on(persistentvolumeclaim, namespace) group_left(volumename) max by(namespace, persistentvolumeclaim, volumename) (kube_persistentvolumeclaim_info{volumename != ''})",

		"cost:pod_limit_cpu_cores":      "sum by (pod, namespace, node) (kube_pod_container_resource_limits{pod!='', namespace!='', node!='', resource='cpu'} * on(pod, namespace) group_left max by (pod, namespace) (kube_pod_status_phase{phase='Running'}))",
		"cost:pod_request_cpu_cores":    "sum by (pod, namespace, node) (kube_pod_container_resource_requests{pod!='', namespace!='', node!='', resource='cpu'} * on(pod, namespace) group_left max by (pod, namespace) (kube_pod_status_phase{phase='Running'}))",
		"cost:pod_usage_cpu_cores":      "sum by (pod, namespace, node) (rate(container_cpu_usage_seconds_total{container!='', container!='POD', pod!='', namespace!='', node!=''}[5m]))",
		"cost:pod_limit_memory_bytes":   "sum by (pod, namespace, node) (kube_pod_container_resource_limits{pod!='', namespace!='', node!='', resource='memory'} * on(pod, namespace) group_left max by (pod, namespace) (kube_pod_status_phase{phase='Running'}))",
		"cost:pod_request_memory_bytes": "sum by (pod, namespace, node) (kube_pod_container_resource_requests{pod!='', namespace!='', node!='', resource='memory'} * on(pod, namespace) group_left max by (pod, namespace) (kube_pod_status_phase{phase='Running'}))",
		"cost:pod_usage_memory_bytes":   "sum by (pod, namespace, node) (container_memory_usage_bytes{container!='', container!='POD', pod!='', namespace!='', node!=''})",

		"ros:image_owners":                   "max_over_time(kube_pod_container_info{container!='', container!='POD', pod!='', namespace!='', namespace!~'kube-.*|openshift|openshift-.*'}[15m]) * on(pod) group_left(owner_kind, owner_name) max by(pod, owner_kind, owner_name) (max_over_time(kube_pod_owner{container!='', container!='POD', pod!='', namespace!='', namespace!~'kube-.*|openshift|openshift-.*'}[15m]))",
		"ros:image_workloads":                "max_over_time(kube_pod_container_info{container!='', container!='POD', pod!='', namespace!='', namespace!~'kube-.*|openshift|openshift-.*'}[15m]) * on(pod) group_left(workload, workload_type) max by(pod, workload, workload_type) (max_over_time(namespace_workload_pod:kube_pod_owner:relabel{pod!='', namespace!='', namespace!~'kube-.*|openshift|openshift-.*'}[15m]))",
		"ros:cpu_request_container_avg":      "avg by(container, pod, namespace, node) (kube_pod_container_resource_requests{container!='', container!='POD', pod!='', namespace!='', namespace!~'kube-.*|openshift|openshift-.*', resource='cpu', unit='core'} * on(pod, namespace) group_left max by (container, pod, namespace) (kube_pod_status_phase{phase='Running'}))",
		"ros:cpu_request_container_sum":      "sum by(container, pod, namespace, node) (kube_pod_container_resource_requests{container!='', container!='POD', pod!='', namespace!='', namespace!~'kube-.*|openshift|openshift-.*', resource='cpu', unit='core'} * on(pod, namespace) group_left max by (container, pod, namespace) (kube_pod_status_phase{phase='Running'}))",
		"ros:cpu_limit_container_avg":        "avg by(container, pod, namespace, node) (kube_pod_container_resource_limits{container!='', container!='POD', pod!='', namespace!='', namespace!~'kube-.*|openshift|openshift-.*', resource='cpu', unit='core'} * on(pod, namespace) group_left max by (container, pod, namespace) (kube_pod_status_phase{phase='Running'}))",
		"ros:cpu_limit_container_sum":        "sum by(container, pod, namespace, node) (kube_pod_container_resource_limits{container!='', container!='POD', pod!='', namespace!='', namespace!~'kube-.*|openshift|openshift-.*', resource='cpu', unit='core'} * on(pod, namespace) group_left max by (container, pod, namespace) (kube_pod_status_phase{phase='Running'}))",
		"ros:cpu_usage_container_avg":        "avg by(container, pod, namespace, node) (avg_over_time(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{container!='', container!='POD', pod!='', namespace!='', namespace!~'kube-.*|openshift|openshift-.*'}[15m]))",
		"ros:cpu_usage_container_min":        "min by(container, pod, namespace, node) (min_over_time(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{container!='', container!='POD', pod!='', namespace!='', namespace!~'kube-.*|openshift|openshift-.*'}[15m]))",
		"ros:cpu_usage_container_max":        "max by(container, pod, namespace, node) (max_over_time(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{container!='', container!='POD', pod!='', namespace!='', namespace!~'kube-.*|openshift|openshift-.*'}[15m]))",
		"ros:cpu_usage_container_sum":        "sum by(container, pod, namespace, node) (avg_over_time(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{container!='', container!='POD', pod!='', namespace!='', namespace!~'kube-.*|openshift|openshift-.*'}[15m]))",
		"ros:cpu_throttle_container_avg":     "avg by(container, pod, namespace, node) (rate(container_cpu_cfs_throttled_seconds_total{container!='', container!='POD', pod!='', namespace!='', namespace!~'kube-.*|openshift|openshift-.*'}[15m]))",
		"ros:cpu_throttle_container_max":     "max by(container, pod, namespace, node) (rate(container_cpu_cfs_throttled_seconds_total{container!='', container!='POD', pod!='', namespace!='', namespace!~'kube-.*|openshift|openshift-.*'}[15m]))",
		"ros:cpu_throttle_container_sum":     "sum by(container, pod, namespace, node) (rate(container_cpu_cfs_throttled_seconds_total{container!='', container!='POD', pod!='', namespace!='', namespace!~'kube-.*|openshift|openshift-.*'}[15m]))",
		"ros:memory_request_container_avg":   "avg by(container, pod, namespace, node) (kube_pod_container_resource_requests{container!='', container!='POD', pod!='', namespace!='', namespace!~'kube-.*|openshift|openshift-.*', resource='memory', unit='byte'} * on(pod, namespace) group_left max by (container, pod, namespace) (kube_pod_status_phase{phase='Running'}))",
		"ros:memory_request_container_sum":   "sum by(container, pod, namespace, node) (kube_pod_container_resource_requests{container!='', container!='POD', pod!='', namespace!='', namespace!~'kube-.*|openshift|openshift-.*', resource='memory', unit='byte'} * on(pod, namespace) group_left max by (container, pod, namespace) (kube_pod_status_phase{phase='Running'}))",
		"ros:memory_limit_container_avg":     "avg by(container, pod, namespace, node) (kube_pod_container_resource_limits{container!='', container!='POD', pod!='', namespace!='', namespace!~'kube-.*|openshift|openshift-.*', resource='memory', unit='byte'} * on(pod, namespace) group_left max by (container, pod, namespace) (kube_pod_status_phase{phase='Running'}))",
		"ros:memory_limit_container_sum":     "sum by(container, pod, namespace, node) (kube_pod_container_resource_limits{container!='', container!='POD', pod!='', namespace!='', namespace!~'kube-.*|openshift|openshift-.*', resource='memory', unit='byte'} * on(pod, namespace) group_left max by (container, pod, namespace) (kube_pod_status_phase{phase='Running'}))",
		"ros:memory_usage_container_avg":     "avg by(container, pod, namespace, node) (avg_over_time(container_memory_working_set_bytes{container!='', container!='POD', pod!='', namespace!='', namespace!~'kube-.*|openshift|openshift-.*'}[15m]))",
		"ros:memory_usage_container_min":     "min by(container, pod, namespace, node) (min_over_time(container_memory_working_set_bytes{container!='', container!='POD', pod!='', namespace!='', namespace!~'kube-.*|openshift|openshift-.*'}[15m]))",
		"ros:memory_usage_container_max":     "max by(container, pod, namespace, node) (max_over_time(container_memory_working_set_bytes{container!='', container!='POD', pod!='', namespace!='', namespace!~'kube-.*|openshift|openshift-.*'}[15m]))",
		"ros:memory_usage_container_sum":     "sum by(container, pod, namespace, node) (avg_over_time(container_memory_working_set_bytes{container!='', container!='POD', pod!='', namespace!='', namespace!~'kube-.*|openshift|openshift-.*'}[15m]))",
		"ros:memory_rss_usage_container_avg": "avg by(container, pod, namespace, node) (avg_over_time(container_memory_rss{container!='', container!='POD', pod!='', namespace!='', namespace!~'kube-.*|openshift|openshift-.*'}[15m]))",
		"ros:memory_rss_usage_container_min": "min by(container, pod, namespace, node) (min_over_time(container_memory_rss{container!='', container!='POD', pod!='', namespace!='', namespace!~'kube-.*|openshift|openshift-.*'}[15m]))",
		"ros:memory_rss_usage_container_max": "max by(container, pod, namespace, node) (max_over_time(container_memory_rss{container!='', container!='POD', pod!='', namespace!='', namespace!~'kube-.*|openshift|openshift-.*'}[15m]))",
		"ros:memory_rss_usage_container_sum": "sum by(container, pod, namespace, node) (avg_over_time(container_memory_rss{container!='', container!='POD', pod!='', namespace!='', namespace!~'kube-.*|openshift|openshift-.*'}[15m]))",
	}

	nodeQueries = &querys{
		query{
			Name:        "node-allocatable-cpu-cores",
			QueryString: QueryMap["cost:node_allocatable_cpu_cores"],
			MetricKey:   staticFields{"node": "node", "provider_id": "provider_id"},
			QueryValue: &saveQueryValue{
				ValName:         "node-allocatable-cpu-cores",
				Method:          "max",
				TransformedName: "node-allocatable-cpu-core-seconds",
			},
			RowKey: []model.LabelName{"node"},
		},
		query{
			Name:        "node-allocatable-memory-bytes",
			QueryString: QueryMap["cost:node_allocatable_memory_bytes"],
			MetricKey:   staticFields{"node": "node", "provider_id": "provider_id"},
			QueryValue: &saveQueryValue{
				ValName:         "node-allocatable-memory-bytes",
				Method:          "max",
				TransformedName: "node-allocatable-memory-byte-seconds",
			},
			RowKey: []model.LabelName{"node"},
		},
		query{
			Name:        "node-capacity-cpu-cores",
			QueryString: QueryMap["cost:node_capacity_cpu_cores"],
			MetricKey:   staticFields{"node": "node", "provider_id": "provider_id"},
			QueryValue: &saveQueryValue{
				ValName:         "node-capacity-cpu-cores",
				Method:          "max",
				TransformedName: "node-capacity-cpu-core-seconds",
			},
			RowKey: []model.LabelName{"node"},
		},
		query{
			Name:        "node-capacity-memory-bytes",
			QueryString: QueryMap["cost:node_capacity_memory_bytes"],
			MetricKey:   staticFields{"node": "node", "provider_id": "provider_id"},
			QueryValue: &saveQueryValue{
				ValName:         "node-capacity-memory-bytes",
				Method:          "max",
				TransformedName: "node-capacity-memory-byte-seconds",
			},
			RowKey: []model.LabelName{"node"},
		},
		query{
			Name:        "node-role",
			QueryString: "kube_node_role",
			MetricKey:   staticFields{"node": "node", "node-role": "role"},
			RowKey:      []model.LabelName{"node"},
		},
		query{
			Name:           "node-labels",
			QueryString:    "kube_node_labels",
			MetricKeyRegex: regexFields{"node_labels": "label_*"},
			RowKey:         []model.LabelName{"node"},
		},
	}
	volQueries = &querys{
		query{
			Name:        "persistentvolume-pod-info",
			QueryString: QueryMap["cost:persistentvolume_pod_info"],
			MetricKey:   staticFields{"namespace": "namespace", "pod": "pod"},
			RowKey:      []model.LabelName{"volumename"},
		},
		query{
			Name:        "persistentvolumeclaim-capacity-bytes",
			QueryString: QueryMap["cost:persistentvolumeclaim_capacity_bytes"],
			QueryValue: &saveQueryValue{
				ValName:         "persistentvolumeclaim-capacity-bytes",
				Method:          "max",
				TransformedName: "persistentvolumeclaim-capacity-byte-seconds",
			},
			RowKey: []model.LabelName{"volumename"},
		},
		query{
			Name:        "persistentvolumeclaim-request-bytes",
			QueryString: QueryMap["cost:persistentvolumeclaim_request_bytes"],
			QueryValue: &saveQueryValue{
				ValName:         "persistentvolumeclaim-request-bytes",
				Method:          "max",
				TransformedName: "persistentvolumeclaim-request-byte-seconds",
			},
			RowKey: []model.LabelName{"volumename"},
		},
		query{
			Name:        "persistentvolumeclaim-usage-bytes",
			QueryString: QueryMap["cost:persistentvolumeclaim_usage_bytes"],
			QueryValue: &saveQueryValue{
				ValName:         "persistentvolumeclaim-usage-bytes",
				Method:          "sum",
				TransformedName: "persistentvolumeclaim-usage-byte-seconds",
			},
			RowKey: []model.LabelName{"volumename"},
		},
		query{
			Name:           "persistentvolume-labels",
			QueryString:    QueryMap["cost:persistentvolume_labels"],
			MetricKey:      staticFields{"storageclass": "storageclass", "persistentvolume": "persistentvolume"},
			MetricKeyRegex: regexFields{"persistentvolume_labels": "label_*"},
			RowKey:         []model.LabelName{"persistentvolume"},
		},
		query{
			Name:           "persistentvolumeclaim-labels",
			QueryString:    QueryMap["cost:persistentvolumeclaim_labels"],
			MetricKey:      staticFields{"namespace": "namespace", "persistentvolumeclaim": "persistentvolumeclaim"},
			MetricKeyRegex: regexFields{"persistentvolumeclaim_labels": "label_"},
			RowKey:         []model.LabelName{"volumename"},
		},
	}
	podQueries = &querys{
		query{
			Name:        "pod-limit-cpu-cores",
			QueryString: QueryMap["cost:pod_limit_cpu_cores"],
			MetricKey:   staticFields{"pod": "pod", "namespace": "namespace", "node": "node"},
			QueryValue: &saveQueryValue{
				ValName:         "pod-limit-cpu-cores",
				Method:          "sum",
				TransformedName: "pod-limit-cpu-core-seconds",
			},
			RowKey: []model.LabelName{"pod", "namespace"},
		},
		query{
			Name:        "pod-limit-memory-bytes",
			QueryString: QueryMap["cost:pod_limit_memory_bytes"],
			MetricKey:   staticFields{"pod": "pod", "namespace": "namespace", "node": "node"},
			QueryValue: &saveQueryValue{
				ValName:         "pod-limit-memory-bytes",
				Method:          "sum",
				TransformedName: "pod-limit-memory-byte-seconds",
			},
			RowKey: []model.LabelName{"pod", "namespace"},
		},
		query{
			Name:        "pod-request-cpu-cores",
			QueryString: QueryMap["cost:pod_request_cpu_cores"],
			MetricKey:   staticFields{"pod": "pod", "namespace": "namespace", "node": "node"},
			QueryValue: &saveQueryValue{
				ValName:         "pod-request-cpu-cores",
				Method:          "sum",
				TransformedName: "pod-request-cpu-core-seconds",
			},
			RowKey: []model.LabelName{"pod", "namespace"},
		},
		query{
			Name:        "pod-request-memory-bytes",
			QueryString: QueryMap["cost:pod_request_memory_bytes"],
			MetricKey:   staticFields{"pod": "pod", "namespace": "namespace", "node": "node"},
			QueryValue: &saveQueryValue{
				ValName:         "pod-request-memory-bytes",
				Method:          "sum",
				TransformedName: "pod-request-memory-byte-seconds",
			},
			RowKey: []model.LabelName{"pod", "namespace"},
		},
		query{
			Name:        "pod-usage-cpu-cores",
			QueryString: QueryMap["cost:pod_usage_cpu_cores"],
			MetricKey:   staticFields{"pod": "pod", "namespace": "namespace", "node": "node"},
			QueryValue: &saveQueryValue{
				ValName:         "pod-usage-cpu-cores",
				Method:          "sum",
				TransformedName: "pod-usage-cpu-core-seconds",
			},
			RowKey: []model.LabelName{"pod", "namespace"},
		},
		query{
			Name:        "pod-usage-memory-bytes",
			QueryString: QueryMap["cost:pod_usage_memory_bytes"],
			MetricKey:   staticFields{"pod": "pod", "namespace": "namespace", "node": "node"},
			QueryValue: &saveQueryValue{
				ValName:         "pod-usage-memory-bytes",
				Method:          "sum",
				TransformedName: "pod-usage-memory-byte-seconds",
			},
			RowKey: []model.LabelName{"pod", "namespace"},
		},
		query{
			Name:           "pod-labels",
			QueryString:    "kube_pod_labels{namespace!='',pod!=''}",
			MetricKey:      staticFields{"pod": "pod", "namespace": "namespace"},
			MetricKeyRegex: regexFields{"pod_labels": "label_*"},
			RowKey:         []model.LabelName{"pod", "namespace"},
		},
	}
	namespaceQueries = &querys{
		query{
			Name:           "namespace-labels",
			QueryString:    "kube_namespace_labels",
			MetricKey:      staticFields{"namespace": "namespace"},
			MetricKeyRegex: regexFields{"namespace_labels": "label_*"},
			RowKey:         []model.LabelName{"namespace"},
		},
	}
	resourceOptimizationQueries = &querys{
		query{
			Name:        "container-image-owner",
			QueryString: QueryMap["ros:image_owners"],
			MetricKey:   staticFields{"image_name": "image", "owner_name": "owner_name", "owner_kind": "owner_kind", "container_name": "container", "pod": "pod", "namespace": "namespace"},
			RowKey:      []model.LabelName{"container", "pod", "namespace"},
		},
		query{
			Name:        "container-image-workload",
			QueryString: QueryMap["ros:image_workloads"],
			MetricKey:   staticFields{"image_name": "image", "workload": "workload", "workload_type": "workload_type", "container_name": "container", "pod": "pod", "namespace": "namespace"},
			RowKey:      []model.LabelName{"container", "pod", "namespace"},
		},
		query{
			Name:        "cpu-request-container-avg",
			QueryString: QueryMap["ros:cpu_request_container_avg"],
			MetricKey:   staticFields{"container_name": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
			QueryValue: &saveQueryValue{
				ValName: "cpu-request-container-avg",
			},
			RowKey: []model.LabelName{"container", "pod", "namespace"},
		},
		query{
			Name:        "cpu-request-container-sum",
			QueryString: QueryMap["ros:cpu_request_container_sum"],
			MetricKey:   staticFields{"container_name": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
			QueryValue: &saveQueryValue{
				ValName: "cpu-request-container-sum",
			},
			RowKey: []model.LabelName{"container", "pod", "namespace"},
		},
		query{
			Name:        "cpu-limit-container-avg",
			QueryString: QueryMap["ros:cpu_limit_container_avg"],
			MetricKey:   staticFields{"container_name": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
			QueryValue: &saveQueryValue{
				ValName: "cpu-limit-container-avg",
			},
			RowKey: []model.LabelName{"container", "pod", "namespace"},
		},
		query{
			Name:        "cpu-limit-container-sum",
			QueryString: QueryMap["ros:cpu_limit_container_sum"],
			MetricKey:   staticFields{"container_name": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
			QueryValue: &saveQueryValue{
				ValName: "cpu-limit-container-sum",
			},
			RowKey: []model.LabelName{"container", "pod", "namespace"},
		},
		query{
			Name:        "cpu-usage-container-avg",
			QueryString: QueryMap["ros:cpu_usage_container_avg"],
			MetricKey:   staticFields{"container_name": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
			QueryValue: &saveQueryValue{
				ValName: "cpu-usage-container-avg",
				Method:  "sum",
			},
			RowKey: []model.LabelName{"container", "pod", "namespace"},
		},
		query{
			Name:        "cpu-usage-container-min",
			QueryString: QueryMap["ros:cpu_usage_container_min"],
			MetricKey:   staticFields{"container_name": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
			QueryValue: &saveQueryValue{
				ValName: "cpu-usage-container-min",
			},
			RowKey: []model.LabelName{"container", "pod", "namespace"},
		},
		query{
			Name:        "cpu-usage-container-max",
			QueryString: QueryMap["ros:cpu_usage_container_max"],
			MetricKey:   staticFields{"container_name": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
			QueryValue: &saveQueryValue{
				ValName: "cpu-usage-container-max",
			},
			RowKey: []model.LabelName{"container", "pod", "namespace"},
		},
		query{
			Name:        "cpu-usage-container-sum",
			QueryString: QueryMap["ros:cpu_usage_container_sum"],
			MetricKey:   staticFields{"container_name": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
			QueryValue: &saveQueryValue{
				ValName: "cpu-usage-container-sum",
			},
			RowKey: []model.LabelName{"container", "pod", "namespace"},
		},
		query{
			Name:        "cpu-throttle-container-avg",
			QueryString: QueryMap["ros:cpu_throttle_container_avg"],
			MetricKey:   staticFields{"container_name": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
			QueryValue: &saveQueryValue{
				ValName: "cpu-throttle-container-avg",
			},
			RowKey: []model.LabelName{"container", "pod", "namespace"},
		},
		query{
			Name:        "cpu-throttle-container-max",
			QueryString: QueryMap["ros:cpu_throttle_container_max"],
			MetricKey:   staticFields{"container_name": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
			QueryValue: &saveQueryValue{
				ValName: "cpu-throttle-container-max",
			},
			RowKey: []model.LabelName{"container", "pod", "namespace"},
		},
		query{
			Name:        "cpu-throttle-container-sum",
			QueryString: QueryMap["ros:cpu_throttle_container_sum"],
			MetricKey:   staticFields{"container_name": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
			QueryValue: &saveQueryValue{
				ValName: "cpu-throttle-container-sum",
			},
			RowKey: []model.LabelName{"container", "pod", "namespace"},
		},
		query{
			Name:        "memory-request-container-avg",
			QueryString: QueryMap["ros:memory_request_container_avg"],
			MetricKey:   staticFields{"container_name": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
			QueryValue: &saveQueryValue{
				ValName: "memory-request-container-avg",
			},
			RowKey: []model.LabelName{"container", "pod", "namespace"},
		},
		query{
			Name:        "memory-request-container-sum",
			QueryString: QueryMap["ros:memory_request_container_sum"],
			MetricKey:   staticFields{"container_name": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
			QueryValue: &saveQueryValue{
				ValName: "memory-request-container-sum",
			},
			RowKey: []model.LabelName{"container", "pod", "namespace"},
		},
		query{
			Name:        "memory-limit-container-avg",
			QueryString: QueryMap["ros:memory_limit_container_avg"],
			MetricKey:   staticFields{"container_name": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
			QueryValue: &saveQueryValue{
				ValName: "memory-limit-container-avg",
			},
			RowKey: []model.LabelName{"container", "pod", "namespace"},
		},
		query{
			Name:        "memory-limit-container-sum",
			QueryString: QueryMap["ros:memory_limit_container_sum"],
			MetricKey:   staticFields{"container_name": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
			QueryValue: &saveQueryValue{
				ValName: "memory-limit-container-sum",
			},
			RowKey: []model.LabelName{"container", "pod", "namespace"},
		},
		query{
			Name:        "memory-usage-container-avg",
			QueryString: QueryMap["ros:memory_usage_container_avg"],
			MetricKey:   staticFields{"container_name": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
			QueryValue: &saveQueryValue{
				ValName: "memory-usage-container-avg",
			},
			RowKey: []model.LabelName{"container", "pod", "namespace"},
		},
		query{
			Name:        "memory-usage-container-min",
			QueryString: QueryMap["ros:memory_usage_container_min"],
			MetricKey:   staticFields{"container_name": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
			QueryValue: &saveQueryValue{
				ValName: "memory-usage-container-min",
			},
			RowKey: []model.LabelName{"container", "pod", "namespace"},
		},
		query{
			Name:        "memory-usage-container-max",
			QueryString: QueryMap["ros:memory_usage_container_max"],
			MetricKey:   staticFields{"container_name": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
			QueryValue: &saveQueryValue{
				ValName: "memory-usage-container-max",
			},
			RowKey: []model.LabelName{"container", "pod", "namespace"},
		},
		query{
			Name:        "memory-usage-container-sum",
			QueryString: QueryMap["ros:memory_usage_container_sum"],
			MetricKey:   staticFields{"container_name": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
			QueryValue: &saveQueryValue{
				ValName: "memory-usage-container-sum",
			},
			RowKey: []model.LabelName{"container", "pod", "namespace"},
		},
		query{
			Name:        "memory-rss-usage-container-avg",
			QueryString: QueryMap["ros:memory_rss_usage_container_avg"],
			MetricKey:   staticFields{"container_name": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
			QueryValue: &saveQueryValue{
				ValName: "memory-rss-usage-container-avg",
			},
			RowKey: []model.LabelName{"container", "pod", "namespace"},
		},
		query{
			Name:        "memory-rss-usage-container-min",
			QueryString: QueryMap["ros:memory_rss_usage_container_min"],
			MetricKey:   staticFields{"container_name": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
			QueryValue: &saveQueryValue{
				ValName: "memory-rss-usage-container-min",
			},
			RowKey: []model.LabelName{"container", "pod", "namespace"},
		},
		query{
			Name:        "memory-rss-usage-container-max",
			QueryString: QueryMap["ros:memory_rss_usage_container_max"],
			MetricKey:   staticFields{"container_name": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
			QueryValue: &saveQueryValue{
				ValName: "memory-rss-usage-container-max",
			},
			RowKey: []model.LabelName{"container", "pod", "namespace"},
		},
		query{
			Name:        "memory-rss-usage-container-sum",
			QueryString: QueryMap["ros:memory_rss_usage_container_sum"],
			MetricKey:   staticFields{"container_name": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
			QueryValue: &saveQueryValue{
				ValName: "memory-rss-usage-container-sum",
			},
			RowKey: []model.LabelName{"container", "pod", "namespace"},
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
	RowKey         []model.LabelName
}

type staticFields map[string]model.LabelName

type regexFields map[string]string

type saveQueryValue struct {
	ValName         string
	Method          string
	TransformedName string
}
