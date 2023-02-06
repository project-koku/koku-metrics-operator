//
// Copyright 2021 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package collector

import "github.com/prometheus/common/model"

var (
	QueryMap = map[string]string{
		"koku_metrics:cost:node_allocatable_cpu_cores":    "kube_node_status_allocatable{resource='cpu'} * on(node) group_left(provider_id) max(kube_node_info) by (node, provider_id)",
		"koku_metrics:cost:node_allocatable_memory_bytes": "kube_node_status_allocatable{resource='memory'} * on(node) group_left(provider_id) max(kube_node_info) by (node, provider_id)",
		"koku_metrics:cost:node_capacity_cpu_cores":       "kube_node_status_capacity{resource='cpu'} * on(node) group_left(provider_id) max(kube_node_info) by (node, provider_id)",
		"koku_metrics:cost:node_capacity_memory_bytes":    "kube_node_status_capacity{resource='memory'} * on(node) group_left(provider_id) max(kube_node_info) by (node, provider_id)",

		"koku_metrics:cost:persistentvolume_pod_info":            "kube_pod_spec_volumes_persistentvolumeclaims_info * on(persistentvolumeclaim, namespace) group_left(volumename) kube_persistentvolumeclaim_info{volumename != ''}",
		"koku_metrics:cost:persistentvolumeclaim_capacity_bytes": "kubelet_volume_stats_capacity_bytes * on(persistentvolumeclaim, namespace) group_left(volumename) kube_persistentvolumeclaim_info{volumename != ''}",
		"koku_metrics:cost:persistentvolumeclaim_request_bytes":  "kube_persistentvolumeclaim_resource_requests_storage_bytes * on(persistentvolumeclaim, namespace) group_left(volumename) kube_persistentvolumeclaim_info{volumename != ''}",
		"koku_metrics:cost:persistentvolumeclaim_usage_bytes":    "kubelet_volume_stats_used_bytes * on(persistentvolumeclaim, namespace) group_left(volumename) kube_persistentvolumeclaim_info{volumename != ''}",

		"koku_metrics:cost:pod_limit_cpu_cores":      "sum by (pod, namespace, node) (kube_pod_container_resource_limits{pod!='', namespace!='', node!='', resource='cpu'} * on(pod, namespace) group_left() max by (pod, namespace) (kube_pod_status_phase{phase='Running'}))",
		"koku_metrics:cost:pod_request_cpu_cores":    "sum by (pod, namespace, node) (kube_pod_container_resource_requests{pod!='', namespace!='', node!='', resource='cpu'} * on(pod, namespace) group_left() max by (pod, namespace) (kube_pod_status_phase{phase='Running'}))",
		"koku_metrics:cost:pod_usage_cpu_cores":      "sum by (pod, namespace, node) (rate(container_cpu_usage_seconds_total{container!='', container!='POD', pod!='', namespace!='', node!=''}[5m]))",
		"koku_metrics:cost:pod_limit_memory_bytes":   "sum by (pod, namespace, node) (kube_pod_container_resource_limits{pod!='', namespace!='', node!='', resource='memory'} * on(pod, namespace) group_left() max by (pod, namespace) (kube_pod_status_phase{phase='Running'}))",
		"koku_metrics:cost:pod_request_memory_bytes": "sum by (pod, namespace, node) (kube_pod_container_resource_requests{pod!='', namespace!='', node!='', resource='memory'} * on(pod, namespace) group_left() max by (pod, namespace) (kube_pod_status_phase{phase='Running'}))",
		"koku_metrics:cost:pod_usage_memory_bytes":   "sum by (pod, namespace, node) (container_memory_usage_bytes{container!='', container!='POD', pod!='', namespace!='', node!=''})",

		"koku_metrics:ros:cpu_request_container_avg":      "avg by(container, pod, namespace) (kube_pod_container_resource_requests{container!='', container!='POD', pod!='', namespace!='', resource='cpu', unit='core'} * on(pod, namespace) group_left() max by (container, pod, namespace) (kube_pod_status_phase{phase='Running'}))",
		"koku_metrics:ros:cpu_request_container_sum":      "sum by(container, pod, namespace) (kube_pod_container_resource_requests{container!='', container!='POD', pod!='', namespace!='', resource='cpu', unit='core'} * on(pod, namespace) group_left() max by (container, pod, namespace) (kube_pod_status_phase{phase='Running'}))",
		"koku_metrics:ros:cpu_limit_container_avg":        "avg by(container, pod, namespace) (kube_pod_container_resource_limits{container!='', container!='POD', pod!='', namespace!='', resource='cpu', unit='core'} * on(pod, namespace) group_left() max by (container, pod, namespace) (kube_pod_status_phase{phase='Running'}))",
		"koku_metrics:ros:cpu_limit_container_sum":        "sum by(container, pod, namespace) (kube_pod_container_resource_limits{container!='', container!='POD', pod!='', namespace!='', resource='cpu', unit='core'} * on(pod, namespace) group_left() max by (container, pod, namespace) (kube_pod_status_phase{phase='Running'}))",
		"koku_metrics:ros:cpu_usage_container_avg":        "avg by(container, pod, namespace) (avg_over_time(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{container!='', container!='POD', pod!='', namespace!=''}[15m]))",
		"koku_metrics:ros:cpu_usage_container_min":        "min by(container, pod, namespace) (min_over_time(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{container!='', container!='POD', pod!='', namespace!=''}[15m]))",
		"koku_metrics:ros:cpu_usage_container_max":        "max by(container, pod, namespace) (max_over_time(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{container!='', container!='POD', pod!='', namespace!=''}[15m]))",
		"koku_metrics:ros:cpu_usage_container_sum":        "sum by(container, pod, namespace) (avg_over_time(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{container!='', container!='POD', pod!='', namespace!=''}[15m]))",
		"koku_metrics:ros:cpu_throttle_container_avg":     "avg by(container, pod, namespace) (rate(container_cpu_cfs_throttled_seconds_total{container!='', container!='POD', pod!='', namespace!=''}[15m]))",
		"koku_metrics:ros:cpu_throttle_container_max":     "max by(container, pod, namespace) (rate(container_cpu_cfs_throttled_seconds_total{container!='', container!='POD', pod!='', namespace!=''}[15m]))",
		"koku_metrics:ros:cpu_throttle_container_sum":     "sum by(container, pod, namespace) (rate(container_cpu_cfs_throttled_seconds_total{container!='', container!='POD', pod!='', namespace!=''}[15m]))",
		"koku_metrics:ros:memory_request_container_avg":   "avg by(container, pod, namespace) (kube_pod_container_resource_requests{container!='', container!='POD', pod!='', namespace!='', resource='memory', unit='byte'} * on(pod, namespace) group_left() max by (container, pod, namespace) (kube_pod_status_phase{phase='Running'}))",
		"koku_metrics:ros:memory_request_container_sum":   "sum by(container, pod, namespace) (kube_pod_container_resource_requests{container!='', container!='POD', pod!='', namespace!='', resource='memory', unit='byte'} * on(pod, namespace) group_left() max by (container, pod, namespace) (kube_pod_status_phase{phase='Running'}))",
		"koku_metrics:ros:memory_limit_container_avg":     "avg by(container, pod, namespace) (kube_pod_container_resource_limits{container!='', container!='POD', pod!='', namespace!='', resource='memory', unit='byte'} * on(pod, namespace) group_left() max by (container, pod, namespace) (kube_pod_status_phase{phase='Running'}))",
		"koku_metrics:ros:memory_limit_container_sum":     "sum by(container, pod, namespace) (kube_pod_container_resource_limits{container!='', container!='POD', pod!='', namespace!='', resource='memory', unit='byte'} * on(pod, namespace) group_left() max by (container, pod, namespace) (kube_pod_status_phase{phase='Running'}))",
		"koku_metrics:ros:memory_usage_container_avg":     "avg by(container, pod, namespace) (avg_over_time(container_memory_working_set_bytes{container!='', container!='POD', pod!='', namespace!=''}[15m]))",
		"koku_metrics:ros:memory_usage_container_min":     "min by(container, pod, namespace) (min_over_time(container_memory_working_set_bytes{container!='', container!='POD', pod!='', namespace!=''}[15m]))",
		"koku_metrics:ros:memory_usage_container_max":     "max by(container, pod, namespace) (max_over_time(container_memory_working_set_bytes{container!='', container!='POD', pod!='', namespace!=''}[15m]))",
		"koku_metrics:ros:memory_usage_container_sum":     "sum by(container, pod, namespace) (avg_over_time(container_memory_working_set_bytes{container!='', container!='POD', pod!='', namespace!=''}[15m]))",
		"koku_metrics:ros:memory_rss_usage_container_avg": "avg by(container, pod, namespace) (avg_over_time(container_memory_rss{container!='', container!='POD', pod!='', namespace!=''}[15m]))",
		"koku_metrics:ros:memory_rss_usage_container_min": "min by(container, pod, namespace) (min_over_time(container_memory_rss{container!='', container!='POD', pod!='', namespace!=''}[15m]))",
		"koku_metrics:ros:memory_rss_usage_container_max": "max by(container, pod, namespace) (max_over_time(container_memory_rss{container!='', container!='POD', pod!='', namespace!=''}[15m]))",
		"koku_metrics:ros:memory_rss_usage_container_sum": "sum by(container, pod, namespace) (avg_over_time(container_memory_rss{container!='', container!='POD', pod!='', namespace!=''}[15m]))",
	}
	nodeQueries = &querys{
		query{
			Name:        "node-allocatable-cpu-cores",
			QueryString: QueryMap["koku_metrics:cost:node_allocatable_cpu_cores"],
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
			QueryString: QueryMap["koku_metrics:cost:node_allocatable_memory_bytes"],
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
			QueryString: QueryMap["koku_metrics:cost:node_capacity_cpu_cores"],
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
			QueryString: QueryMap["koku_metrics:cost:node_capacity_memory_bytes"],
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
			Name:        "persistentvolume_pod_info",
			QueryString: QueryMap["koku_metrics:cost:persistentvolume_pod_info"],
			MetricKey:   staticFields{"namespace": "namespace", "pod": "pod"},
			RowKey:      []model.LabelName{"volumename"},
		},
		query{
			Name:        "persistentvolumeclaim-capacity-bytes",
			QueryString: QueryMap["koku_metrics:cost:persistentvolumeclaim_capacity_bytes"],
			QueryValue: &saveQueryValue{
				ValName:         "persistentvolumeclaim-capacity-bytes",
				Method:          "max",
				TransformedName: "persistentvolumeclaim-capacity-byte-seconds",
			},
			RowKey: []model.LabelName{"volumename"},
		},
		query{
			Name:        "persistentvolumeclaim-request-bytes",
			QueryString: QueryMap["koku_metrics:cost:persistentvolumeclaim_request_bytes"],
			QueryValue: &saveQueryValue{
				ValName:         "persistentvolumeclaim-request-bytes",
				Method:          "max",
				TransformedName: "persistentvolumeclaim-request-byte-seconds",
			},
			RowKey: []model.LabelName{"volumename"},
		},
		query{
			Name:        "persistentvolumeclaim-usage-bytes",
			QueryString: QueryMap["koku_metrics:cost:persistentvolumeclaim_usage_bytes"],
			QueryValue: &saveQueryValue{
				ValName:         "persistentvolumeclaim-usage-bytes",
				Method:          "sum",
				TransformedName: "persistentvolumeclaim-usage-byte-seconds",
			},
			RowKey: []model.LabelName{"volumename"},
		},
		query{
			Name:           "persistentvolume-labels",
			QueryString:    "kube_persistentvolume_labels * on(persistentvolume, namespace) group_left(storageclass) kube_persistentvolume_info",
			MetricKey:      staticFields{"storageclass": "storageclass", "persistentvolume": "persistentvolume"},
			MetricKeyRegex: regexFields{"persistentvolume_labels": "label_*"},
			RowKey:         []model.LabelName{"persistentvolume"},
		},
		query{
			Name:           "persistentvolumeclaim-labels",
			QueryString:    "kube_persistentvolumeclaim_labels * on(persistentvolumeclaim, namespace) group_left(volumename) kube_persistentvolumeclaim_info{volumename != ''}",
			MetricKey:      staticFields{"namespace": "namespace", "persistentvolumeclaim": "persistentvolumeclaim"},
			MetricKeyRegex: regexFields{"persistentvolumeclaim_labels": "label_"},
			RowKey:         []model.LabelName{"volumename"},
		},
	}
	podQueries = &querys{
		query{
			Name:        "pod-limit-cpu-cores",
			QueryString: QueryMap["koku_metrics:cost:pod_limit_cpu_cores"],
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
			QueryString: QueryMap["koku_metrics:cost:pod_limit_memory_bytes"],
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
			QueryString: QueryMap["koku_metrics:cost:pod_request_cpu_cores"],
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
			QueryString: QueryMap["koku_metrics:cost:pod_request_memory_bytes"],
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
			QueryString: QueryMap["koku_metrics:cost:pod_usage_cpu_cores"],
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
			QueryString: QueryMap["koku_metrics:cost:pod_usage_memory_bytes"],
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
	// resourceOptimizationQueries = &querys{
	// 	query{
	// 		Name:        "cpu-request-container-avg",
	// 		QueryString: QueryMap["koku_metrics:ros:cpu_request_container_avg"],
	// 		MetricKey:   staticFields{"container": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
	// 		QueryValue: &saveQueryValue{
	// 			ValName: "cpu-request-container-avg",
	// 			Method:  "sum",
	// 		},
	// 		RowKey: []model.LabelName{"container", "pod", "namespace"},
	// 	},
	// 	query{
	// 		Name:        "cpu-request-container-sum",
	// 		QueryString: QueryMap["koku_metrics:ros:cpu_request_container_sum"],
	// 		MetricKey:   staticFields{"container": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
	// 		QueryValue: &saveQueryValue{
	// 			ValName: "cpu-request-container-sum",
	// 			Method:  "sum",
	// 		},
	// 		RowKey: []model.LabelName{"container", "pod", "namespace"},
	// 	},
	// 	query{
	// 		Name:        "cpu-limit-container-avg",
	// 		QueryString: QueryMap["koku_metrics:ros:cpu_limit_container_avg"],
	// 		MetricKey:   staticFields{"container": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
	// 		QueryValue: &saveQueryValue{
	// 			ValName: "cpu-limit-container-avg",
	// 			Method:  "sum",
	// 		},
	// 		RowKey: []model.LabelName{"container", "pod", "namespace"},
	// 	},
	// 	query{
	// 		Name:        "cpu-limit-container-sum",
	// 		QueryString: QueryMap["koku_metrics:ros:cpu_limit_container_sum"],
	// 		MetricKey:   staticFields{"container": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
	// 		QueryValue: &saveQueryValue{
	// 			ValName: "cpu-limit-container-sum",
	// 			Method:  "sum",
	// 		},
	// 		RowKey: []model.LabelName{"container", "pod", "namespace"},
	// 	},
	// 	query{
	// 		Name:        "cpu-usage-container-avg",
	// 		QueryString: QueryMap["koku_metrics:ros:cpu_usage_container_avg"],
	// 		MetricKey:   staticFields{"container": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
	// 		QueryValue: &saveQueryValue{
	// 			ValName: "cpu-usage-container-avg",
	// 			Method:  "sum",
	// 		},
	// 		RowKey: []model.LabelName{"container", "pod", "namespace"},
	// 	},
	// 	query{
	// 		Name:        "cpu-usage-container-min",
	// 		QueryString: QueryMap["koku_metrics:ros:cpu_usage_container_min"],
	// 		MetricKey:   staticFields{"container": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
	// 		QueryValue: &saveQueryValue{
	// 			ValName: "cpu-usage-container-min",
	// 			Method:  "sum",
	// 		},
	// 		RowKey: []model.LabelName{"container", "pod", "namespace"},
	// 	},
	// 	query{
	// 		Name:        "cpu-usage-container-max",
	// 		QueryString: QueryMap["koku_metrics:ros:cpu_usage_container_max"],
	// 		MetricKey:   staticFields{"container": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
	// 		QueryValue: &saveQueryValue{
	// 			ValName: "cpu-usage-container-max",
	// 			Method:  "sum",
	// 		},
	// 		RowKey: []model.LabelName{"container", "pod", "namespace"},
	// 	},
	// 	query{
	// 		Name:        "cpu-usage-container-sum",
	// 		QueryString: QueryMap["koku_metrics:ros:cpu_usage_container_sum"],
	// 		MetricKey:   staticFields{"container": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
	// 		QueryValue: &saveQueryValue{
	// 			ValName: "cpu-usage-container-sum",
	// 			Method:  "sum",
	// 		},
	// 		RowKey: []model.LabelName{"container", "pod", "namespace"},
	// 	},
	// 	query{
	// 		Name:        "cpu-throttle-container-avg",
	// 		QueryString: QueryMap["koku_metrics:ros:cpu_throttle_container_avg"],
	// 		MetricKey:   staticFields{"container": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
	// 		QueryValue: &saveQueryValue{
	// 			ValName: "cpu-throttle-container-avg",
	// 			Method:  "sum",
	// 		},
	// 		RowKey: []model.LabelName{"container", "pod", "namespace"},
	// 	},
	// 	query{
	// 		Name:        "cpu-throttle-container-max",
	// 		QueryString: QueryMap["koku_metrics:ros:cpu_throttle_container_max"],
	// 		MetricKey:   staticFields{"container": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
	// 		QueryValue: &saveQueryValue{
	// 			ValName: "cpu-throttle-container-max",
	// 			Method:  "sum",
	// 		},
	// 		RowKey: []model.LabelName{"container", "pod", "namespace"},
	// 	},
	// 	query{
	// 		Name:        "cpu-throttle-container-sum",
	// 		QueryString: QueryMap["koku_metrics:ros:cpu_throttle_container_sum"],
	// 		MetricKey:   staticFields{"container": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
	// 		QueryValue: &saveQueryValue{
	// 			ValName: "cpu-throttle-container-sum",
	// 			Method:  "sum",
	// 		},
	// 		RowKey: []model.LabelName{"container", "pod", "namespace"},
	// 	},
	// 	query{
	// 		Name:        "memory-request-container-avg",
	// 		QueryString: QueryMap["koku_metrics:ros:memory_request_container_avg"],
	// 		MetricKey:   staticFields{"container": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
	// 		QueryValue: &saveQueryValue{
	// 			ValName: "memory-request-container-avg",
	// 			Method:  "sum",
	// 		},
	// 		RowKey: []model.LabelName{"container", "pod", "namespace"},
	// 	},
	// 	query{
	// 		Name:        "memory-request-container-sum",
	// 		QueryString: QueryMap["koku_metrics:ros:memory_request_container_sum"],
	// 		MetricKey:   staticFields{"container": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
	// 		QueryValue: &saveQueryValue{
	// 			ValName: "memory-request-container-sum",
	// 			Method:  "sum",
	// 		},
	// 		RowKey: []model.LabelName{"container", "pod", "namespace"},
	// 	},
	// 	query{
	// 		Name:        "memory-limit-container-avg",
	// 		QueryString: QueryMap["koku_metrics:ros:memory_limit_container_avg"],
	// 		MetricKey:   staticFields{"container": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
	// 		QueryValue: &saveQueryValue{
	// 			ValName: "memory-limit-container-avg",
	// 			Method:  "sum",
	// 		},
	// 		RowKey: []model.LabelName{"container", "pod", "namespace"},
	// 	},
	// 	query{
	// 		Name:        "memory-limit-container-sum",
	// 		QueryString: QueryMap["koku_metrics:ros:memory_limit_container_sum"],
	// 		MetricKey:   staticFields{"container": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
	// 		QueryValue: &saveQueryValue{
	// 			ValName: "memory-limit-container-sum",
	// 			Method:  "sum",
	// 		},
	// 		RowKey: []model.LabelName{"container", "pod", "namespace"},
	// 	},
	// 	query{
	// 		Name:        "memory-usage-container-avg",
	// 		QueryString: QueryMap["koku_metrics:ros:memory_usage_container_avg"],
	// 		MetricKey:   staticFields{"container": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
	// 		QueryValue: &saveQueryValue{
	// 			ValName: "memory-usage-container-avg",
	// 			Method:  "sum",
	// 		},
	// 		RowKey: []model.LabelName{"container", "pod", "namespace"},
	// 	},
	// 	query{
	// 		Name:        "memory-usage-container-min",
	// 		QueryString: QueryMap["koku_metrics:ros:memory_usage_container_min"],
	// 		MetricKey:   staticFields{"container": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
	// 		QueryValue: &saveQueryValue{
	// 			ValName: "memory-usage-container-min",
	// 			Method:  "sum",
	// 		},
	// 		RowKey: []model.LabelName{"container", "pod", "namespace"},
	// 	},
	// 	query{
	// 		Name:        "memory-usage-container-max",
	// 		QueryString: QueryMap["koku_metrics:ros:memory_usage_container_max"],
	// 		MetricKey:   staticFields{"container": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
	// 		QueryValue: &saveQueryValue{
	// 			ValName: "memory-usage-container-max",
	// 			Method:  "sum",
	// 		},
	// 		RowKey: []model.LabelName{"container", "pod", "namespace"},
	// 	},
	// 	query{
	// 		Name:        "memory-usage-container-sum",
	// 		QueryString: QueryMap["koku_metrics:ros:memory_usage_container_sum"],
	// 		MetricKey:   staticFields{"container": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
	// 		QueryValue: &saveQueryValue{
	// 			ValName: "memory-usage-container-sum",
	// 			Method:  "sum",
	// 		},
	// 		RowKey: []model.LabelName{"container", "pod", "namespace"},
	// 	},
	// 	query{
	// 		Name:        "memory-rss-usage-container-avg",
	// 		QueryString: QueryMap["koku_metrics:ros:memory_rss_usage_container_avg"],
	// 		MetricKey:   staticFields{"container": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
	// 		QueryValue: &saveQueryValue{
	// 			ValName: "memory-rss-usage-container-avg",
	// 			Method:  "sum",
	// 		},
	// 		RowKey: []model.LabelName{"container", "pod", "namespace"},
	// 	},
	// 	query{
	// 		Name:        "memory-rss-usage-container-min",
	// 		QueryString: QueryMap["koku_metrics:ros:memory_rss_usage_container_min"],
	// 		MetricKey:   staticFields{"container": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
	// 		QueryValue: &saveQueryValue{
	// 			ValName: "memory-rss-usage-container-min",
	// 			Method:  "sum",
	// 		},
	// 		RowKey: []model.LabelName{"container", "pod", "namespace"},
	// 	},
	// 	query{
	// 		Name:        "memory-rss-usage-container-max",
	// 		QueryString: QueryMap["koku_metrics:ros:memory_rss_usage_container_max"],
	// 		MetricKey:   staticFields{"container": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
	// 		QueryValue: &saveQueryValue{
	// 			ValName: "memory-rss-usage-container-max",
	// 			Method:  "sum",
	// 		},
	// 		RowKey: []model.LabelName{"container", "pod", "namespace"},
	// 	},
	// 	query{
	// 		Name:        "memory-rss-usage-container-sum",
	// 		QueryString: QueryMap["koku_metrics:ros:memory_rss_usage_container_sum"],
	// 		MetricKey:   staticFields{"container": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
	// 		QueryValue: &saveQueryValue{
	// 			ValName: "memory-rss-usage-container-sum",
	// 			Method:  "sum",
	// 		},
	// 		RowKey: []model.LabelName{"container", "pod", "namespace"},
	// 	},
	// }
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
