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
		"cost:persistentvolumeclaim_capacity_bytes": "kube_persistentvolume_capacity_bytes{persistentvolume != ''}",
		"cost:persistentvolumeclaim_request_bytes":  "kube_persistentvolumeclaim_resource_requests_storage_bytes * on(persistentvolumeclaim, namespace) group_left(volumename) max by(namespace, persistentvolumeclaim, volumename) (kube_persistentvolumeclaim_info{volumename != ''})",
		"cost:persistentvolumeclaim_usage_bytes":    "kubelet_volume_stats_used_bytes * on(persistentvolumeclaim, namespace) group_left(volumename) max by(namespace, persistentvolumeclaim, volumename) (kube_persistentvolumeclaim_info{volumename != ''})",
		"cost:persistentvolume_labels":              "kube_persistentvolume_labels * on(persistentvolume, namespace) group_left(storageclass, csi_driver, csi_volume_handle) max by(namespace, persistentvolume, storageclass, csi_driver, csi_volume_handle) (kube_persistentvolume_info)",
		"cost:persistentvolumeclaim_labels":         "kube_persistentvolumeclaim_labels * on(persistentvolumeclaim, namespace) group_left(volumename) max by(namespace, persistentvolumeclaim, volumename) (kube_persistentvolumeclaim_info{volumename != ''})",

		"cost:pod_limit_cpu_cores":      "sum by (pod, namespace, node) (kube_pod_container_resource_limits{pod!='', namespace!='', node!='', resource='cpu'} * on(pod, namespace) group_left max by (pod, namespace) (kube_pod_status_phase{phase='Running'}))",
		"cost:pod_request_cpu_cores":    "sum by (pod, namespace, node) (kube_pod_container_resource_requests{pod!='', namespace!='', node!='', resource='cpu'} * on(pod, namespace) group_left max by (pod, namespace) (kube_pod_status_phase{phase='Running'}))",
		"cost:pod_usage_cpu_cores":      "sum by (pod, namespace, node) (rate(container_cpu_usage_seconds_total{container!='', container!='POD', pod!='', namespace!='', node!=''}[5m]))",
		"cost:pod_limit_memory_bytes":   "sum by (pod, namespace, node) (kube_pod_container_resource_limits{pod!='', namespace!='', node!='', resource='memory'} * on(pod, namespace) group_left max by (pod, namespace) (kube_pod_status_phase{phase='Running'}))",
		"cost:pod_request_memory_bytes": "sum by (pod, namespace, node) (kube_pod_container_resource_requests{pod!='', namespace!='', node!='', resource='memory'} * on(pod, namespace) group_left max by (pod, namespace) (kube_pod_status_phase{phase='Running'}))",
		"cost:pod_usage_memory_bytes":   "sum by (pod, namespace, node) (container_memory_usage_bytes{container!='', container!='POD', pod!='', namespace!='', node!=''})",
		"cost:pod_labels":               "kube_pod_labels{namespace!='',pod!=''} * on(pod, namespace) group_left max by (pod, namespace) (kube_pod_status_phase{phase='Running'})",

		// virtual machine metrics queries
		"cost:vm_cpu_limit_cores":           "sum by (name, namespace) (kubevirt_vm_resource_limits{name!='', namespace!='', resource='cpu'}) * on (name, namespace) group_left max by (name, namespace) (kubevirt_vmi_info{phase='running'})",
		"cost:vm_cpu_request_cores":         "sum by (name, namespace) (kubevirt_vm_resource_requests{name!='', namespace!='', resource='cpu', unit='cores'}) * on (name, namespace) group_left max by (name, namespace) (kubevirt_vmi_info{phase='running'})",
		"cost:vm_cpu_request_sockets":       "sum by (name, namespace) (kubevirt_vm_resource_requests{name!='', namespace!='', resource='cpu', unit='sockets'}) * on (name, namespace) group_left max by (name, namespace) (kubevirt_vmi_info{phase='running'})",
		"cost:vm_cpu_request_threads":       "sum by (name, namespace) (kubevirt_vm_resource_requests{name!='', namespace!='', resource='cpu', unit='threads'}) * on (name, namespace) group_left max by (name, namespace) (kubevirt_vmi_info{phase='running'})",
		"cost:vm_cpu_usage":                 "sum by (name, namespace) (rate(kubevirt_vmi_cpu_usage_seconds_total{name!='', namespace!=''}[5m])) * on (name, namespace) group_left max by (name, namespace) (kubevirt_vmi_info{phase='running'})",
		"cost:vm_memory_limit_bytes":        "sum by (name, namespace) (kubevirt_vm_resource_limits{name!='', namespace!='', resource='memory'}) * on (name, namespace) group_left max by (name, namespace) (kubevirt_vmi_info{phase='running'})",
		"cost:vm_memory_request_bytes":      "sum by (name, namespace) (kubevirt_vm_resource_requests{name!='', namespace!='', resource='memory'}) * on (name, namespace) group_left max by (name, namespace) (kubevirt_vmi_info{phase='running'})",
		"cost:vm_memory_usage_bytes":        "sum by (name, namespace) (sum_over_time(kubevirt_vmi_memory_used_bytes{name!='', namespace!=''}[5m])) * on (name, namespace) group_left max by (name, namespace) (kubevirt_vmi_info{phase='running'})",
		"cost:vm_info":                      "sum by (name, namespace, node, os, instance_type, guest_os_name, guest_os_version_id, guest_os_arch) (kubevirt_vmi_info{phase='running'}) * on(node) group_left(provider_id) max by (node, provider_id) (kube_node_info)",
		"cost:vm_disk_allocated_size_bytes": "sum by (name, namespace, device, persistentvolumeclaim, volume_mode) (kubevirt_vm_disk_allocated_size_bytes{name!='', namespace!=''}) * on (name, namespace) group_left max by (name, namespace) (kubevirt_vmi_info{phase='running'})",
		"cost:vm_labels":                    "kubevirt_vm_labels{name!='', namespace!=''}",

		// cost NVIDIA GPU metrics queries
		"cost:nvidia_gpu_capacity_memory_mib": "sum by (pod, namespace, node, resource, label_nvidia_com_gpu_memory) ((kube_pod_container_resource_requests{pod!='', namespace!='', node!='', resource='nvidia_com_gpu'} * on(pod, namespace) group_left max by (pod, namespace) (kube_pod_status_phase{phase='Running'})) * on(node) group_left(label_nvidia_com_gpu_memory) (max by (node, label_nvidia_com_gpu_memory) (kube_node_labels)))",
		"cost:nvidia_gpu_utilization":         "sum by (exported_pod, exported_namespace, Hostname, UUID, modelName) (DCGM_FI_PROF_GR_ENGINE_ACTIVE) * on(exported_pod, exported_namespace) group_left(pod, namespace) max by (exported_pod, exported_namespace, pod, namespace) (label_replace(label_replace(kube_pod_status_phase{phase='Running'}, 'exported_pod', '$1', 'pod', '(.*)'), 'exported_namespace', '$1', 'namespace', '(.*)'))",

		// resource optimization container metrics queries
		"ros:namespace_filter":               "kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'} or kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'}",
		"ros:image_owners":                   "((max_over_time(kube_pod_container_info{container!='', container!='POD'}[15m]) * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'}) or (max_over_time(kube_pod_container_info{container!='', container!='POD'}[15m]) * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})) * on(pod, namespace) group_left(owner_kind, owner_name) max by(pod, namespace, owner_kind, owner_name) (max_over_time(kube_pod_owner{container!='', container!='POD', pod!=''}[15m]))",
		"ros:image_workloads":                "((max_over_time(kube_pod_container_info{container!='', container!='POD'}[15m]) * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'}) or (max_over_time(kube_pod_container_info{container!='', container!='POD'}[15m]) * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})) * on(pod, namespace) group_left(workload, workload_type) max by(pod, namespace, workload, workload_type) (max_over_time(namespace_workload_pod:kube_pod_owner:relabel{pod!=''}[15m]))",
		"ros:cpu_request_container_avg":      "((avg by(container, pod, namespace, node) (kube_pod_container_resource_requests{container!='', container!='POD', pod!='', resource='cpu', unit='core'} * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})) or (avg by(container, pod, namespace, node) (kube_pod_container_resource_requests{container!='', container!='POD', pod!='', resource='cpu', unit='core'} * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'}))) * on(pod, namespace) group_left max by (container, pod, namespace) (kube_pod_status_phase{phase='Running'})",
		"ros:cpu_request_container_sum":      "((sum by(container, pod, namespace, node) (kube_pod_container_resource_requests{container!='', container!='POD', pod!='', resource='cpu', unit='core'} * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})) or (sum by(container, pod, namespace, node) (kube_pod_container_resource_requests{container!='', container!='POD', pod!='', resource='cpu', unit='core'} * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'}))) * on(pod, namespace) group_left max by (container, pod, namespace) (kube_pod_status_phase{phase='Running'})",
		"ros:cpu_limit_container_avg":        "((avg by(container, pod, namespace, node) (kube_pod_container_resource_limits{container!='', container!='POD', pod!='', resource='cpu', unit='core'} * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})) or (avg by(container, pod, namespace, node) (kube_pod_container_resource_limits{container!='', container!='POD', pod!='', resource='cpu', unit='core'} * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'}))) * on(pod, namespace) group_left max by (container, pod, namespace) (kube_pod_status_phase{phase='Running'})",
		"ros:cpu_limit_container_sum":        "((sum by(container, pod, namespace, node) (kube_pod_container_resource_limits{container!='', container!='POD', pod!='', resource='cpu', unit='core'} * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})) or (sum by(container, pod, namespace, node) (kube_pod_container_resource_limits{container!='', container!='POD', pod!='', resource='cpu', unit='core'} * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'}))) * on(pod, namespace) group_left max by (container, pod, namespace) (kube_pod_status_phase{phase='Running'})",
		"ros:cpu_usage_container_avg":        "(avg by(container, pod, namespace, node) (avg_over_time(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{container!='', container!='POD', pod!=''}[15m]) * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})) or (avg by(container, pod, namespace, node) (avg_over_time(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{container!='', container!='POD', pod!=''}[15m]) * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'}))",
		"ros:cpu_usage_container_min":        "(min by(container, pod, namespace, node) (min_over_time(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{container!='', container!='POD', pod!=''}[15m]) * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})) or (min by(container, pod, namespace, node) (min_over_time(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{container!='', container!='POD', pod!=''}[15m]) * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'}))",
		"ros:cpu_usage_container_max":        "(max by(container, pod, namespace, node) (max_over_time(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{container!='', container!='POD', pod!=''}[15m]) * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})) or (max by(container, pod, namespace, node) (max_over_time(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{container!='', container!='POD', pod!=''}[15m]) * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'}))",
		"ros:cpu_usage_container_sum":        "(sum by(container, pod, namespace, node) (avg_over_time(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{container!='', container!='POD', pod!=''}[15m]) * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})) or (sum by(container, pod, namespace, node) (avg_over_time(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{container!='', container!='POD', pod!=''}[15m]) * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'}))",
		"ros:cpu_throttle_container_avg":     "(avg by(container, pod, namespace, node) (rate(container_cpu_cfs_throttled_seconds_total{container!='', container!='POD', pod!=''}[15m]) * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})) or (avg by(container, pod, namespace, node) (rate(container_cpu_cfs_throttled_seconds_total{container!='', container!='POD', pod!=''}[15m]) * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'}))",
		"ros:cpu_throttle_container_max":     "(max by(container, pod, namespace, node) (rate(container_cpu_cfs_throttled_seconds_total{container!='', container!='POD', pod!=''}[15m]) * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})) or (max by(container, pod, namespace, node) (rate(container_cpu_cfs_throttled_seconds_total{container!='', container!='POD', pod!=''}[15m]) * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'}))",
		"ros:cpu_throttle_container_min":     "(min by(container, pod, namespace, node) (rate(container_cpu_cfs_throttled_seconds_total{container!='', container!='POD', pod!=''}[15m]) * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})) or (min by(container, pod, namespace, node) (rate(container_cpu_cfs_throttled_seconds_total{container!='', container!='POD', pod!=''}[15m]) * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'}))",
		"ros:cpu_throttle_container_sum":     "(sum by(container, pod, namespace, node) (rate(container_cpu_cfs_throttled_seconds_total{container!='', container!='POD', pod!=''}[15m]) * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})) or (sum by(container, pod, namespace, node) (rate(container_cpu_cfs_throttled_seconds_total{container!='', container!='POD', pod!=''}[15m]) * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'}))",
		"ros:memory_request_container_avg":   "((avg by(container, pod, namespace, node) (kube_pod_container_resource_requests{container!='', container!='POD', pod!='', resource='memory', unit='byte'} * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})) or (avg by(container, pod, namespace, node) (kube_pod_container_resource_requests{container!='', container!='POD', pod!='', resource='memory', unit='byte'} * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'}))) * on(pod, namespace) group_left max by (container, pod, namespace) (kube_pod_status_phase{phase='Running'})",
		"ros:memory_request_container_sum":   "((sum by(container, pod, namespace, node) (kube_pod_container_resource_requests{container!='', container!='POD', pod!='', resource='memory', unit='byte'} * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})) or (sum by(container, pod, namespace, node) (kube_pod_container_resource_requests{container!='', container!='POD', pod!='', resource='memory', unit='byte'} * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'}))) * on(pod, namespace) group_left max by (container, pod, namespace) (kube_pod_status_phase{phase='Running'})",
		"ros:memory_limit_container_avg":     "((avg by(container, pod, namespace, node) (kube_pod_container_resource_limits{container!='', container!='POD', pod!='', resource='memory', unit='byte'} * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})) or (avg by(container, pod, namespace, node) (kube_pod_container_resource_limits{container!='', container!='POD', pod!='', resource='memory', unit='byte'} * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'}))) * on(pod, namespace) group_left max by (container, pod, namespace) (kube_pod_status_phase{phase='Running'})",
		"ros:memory_limit_container_sum":     "((sum by(container, pod, namespace, node) (kube_pod_container_resource_limits{container!='', container!='POD', pod!='', resource='memory', unit='byte'} * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})) or (sum by(container, pod, namespace, node) (kube_pod_container_resource_limits{container!='', container!='POD', pod!='', resource='memory', unit='byte'} * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'}))) * on(pod, namespace) group_left max by (container, pod, namespace) (kube_pod_status_phase{phase='Running'})",
		"ros:memory_usage_container_avg":     "(avg by(container, pod, namespace, node) (avg_over_time(container_memory_working_set_bytes{container!='', container!='POD', pod!=''}[15m]) * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})) or (avg by(container, pod, namespace, node) (avg_over_time(container_memory_working_set_bytes{container!='', container!='POD', pod!=''}[15m]) * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'}))",
		"ros:memory_usage_container_min":     "(min by(container, pod, namespace, node) (min_over_time(container_memory_working_set_bytes{container!='', container!='POD', pod!=''}[15m]) * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})) or (min by(container, pod, namespace, node) (min_over_time(container_memory_working_set_bytes{container!='', container!='POD', pod!=''}[15m]) * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'}))",
		"ros:memory_usage_container_max":     "(max by(container, pod, namespace, node) (max_over_time(container_memory_working_set_bytes{container!='', container!='POD', pod!=''}[15m]) * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})) or (max by(container, pod, namespace, node) (max_over_time(container_memory_working_set_bytes{container!='', container!='POD', pod!=''}[15m]) * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'}))",
		"ros:memory_usage_container_sum":     "(sum by(container, pod, namespace, node) (avg_over_time(container_memory_working_set_bytes{container!='', container!='POD', pod!=''}[15m]) * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})) or (sum by(container, pod, namespace, node) (avg_over_time(container_memory_working_set_bytes{container!='', container!='POD', pod!=''}[15m]) * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'}))",
		"ros:memory_rss_usage_container_avg": "(avg by(container, pod, namespace, node) (avg_over_time(container_memory_rss{container!='', container!='POD', pod!=''}[15m]) * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})) or (avg by(container, pod, namespace, node) (avg_over_time(container_memory_rss{container!='', container!='POD', pod!=''}[15m]) * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'}))",
		"ros:memory_rss_usage_container_min": "(min by(container, pod, namespace, node) (min_over_time(container_memory_rss{container!='', container!='POD', pod!=''}[15m]) * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})) or (min by(container, pod, namespace, node) (min_over_time(container_memory_rss{container!='', container!='POD', pod!=''}[15m]) * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'}))",
		"ros:memory_rss_usage_container_max": "(max by(container, pod, namespace, node) (max_over_time(container_memory_rss{container!='', container!='POD', pod!=''}[15m]) * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})) or (max by(container, pod, namespace, node) (max_over_time(container_memory_rss{container!='', container!='POD', pod!=''}[15m]) * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'}))",
		"ros:memory_rss_usage_container_sum": "(sum by(container, pod, namespace, node) (avg_over_time(container_memory_rss{container!='', container!='POD', pod!=''}[15m]) * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})) or (sum by(container, pod, namespace, node) (avg_over_time(container_memory_rss{container!='', container!='POD', pod!=''}[15m]) * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'}))",

		// resource optimization NVIDIA GPU container level metrics queries
		"ros:accelerator_core_usage_percentage_min":  "(min by (modelName, exported_container, exported_namespace, exported_pod, Hostname) (min_over_time(DCGM_FI_DEV_GPU_UTIL{exported_namespace != '', exported_container != '', exported_pod != ''}[15m])) * on(exported_namespace) group_left(namespace) label_replace(kube_namespace_labels{label_insights_cost_management_optimizations='true'}, 'exported_namespace', '$1', 'namespace', '(.*)') or min by (modelName,exported_container, exported_namespace,exported_pod, Hostname) (min_over_time(DCGM_FI_DEV_GPU_UTIL{exported_namespace != '', exported_container != '', exported_pod != ''}[15m])) * on(exported_namespace) group_left(namespace) label_replace(kube_namespace_labels{label_cost_management_optimizations='true'}, 'exported_namespace', '$1', 'namespace', '(.*)'))",
		"ros:accelerator_core_usage_percentage_max":  "(max by (modelName, exported_container, exported_namespace, exported_pod, Hostname) (max_over_time(DCGM_FI_DEV_GPU_UTIL{exported_namespace != '', exported_container != '', exported_pod != ''}[15m])) * on(exported_namespace) group_left(namespace) label_replace(kube_namespace_labels{label_insights_cost_management_optimizations='true'}, 'exported_namespace', '$1', 'namespace', '(.*)') or max by (modelName,exported_container,exported_namespace,exported_pod, Hostname) (max_over_time(DCGM_FI_DEV_GPU_UTIL{exported_namespace != '', exported_container != '', exported_pod != ''}[15m])) * on(exported_namespace) group_left(namespace) label_replace(kube_namespace_labels{label_cost_management_optimizations='true'}, 'exported_namespace', '$1', 'namespace', '(.*)'))",
		"ros:accelerator_core_usage_percentage_avg":  "(avg by (modelName, exported_container, exported_namespace, exported_pod, Hostname) (avg_over_time(DCGM_FI_DEV_GPU_UTIL{exported_namespace != '', exported_container != '', exported_pod != ''}[15m])) * on(exported_namespace) group_left(namespace) label_replace(kube_namespace_labels{label_insights_cost_management_optimizations='true'}, 'exported_namespace', '$1', 'namespace', '(.*)') or avg by (modelName, exported_container, exported_namespace, exported_pod, Hostname) (avg_over_time(DCGM_FI_DEV_GPU_UTIL{exported_namespace != '', exported_container != '', exported_pod != ''}[15m])) * on(exported_namespace) group_left(namespace) label_replace(kube_namespace_labels{label_cost_management_optimizations='true'}, 'exported_namespace', '$1', 'namespace', '(.*)'))",
		"ros:accelerator_memory_copy_percentage_min": "(min by (modelName, exported_container, exported_namespace, exported_pod, Hostname) (min_over_time(DCGM_FI_DEV_MEM_COPY_UTIL{exported_namespace != '', exported_container != '', exported_pod != ''}[15m])) * on(exported_namespace) group_left(namespace) label_replace(kube_namespace_labels{label_insights_cost_management_optimizations='true'}, 'exported_namespace', '$1', 'namespace', '(.*)') or min by (modelName,exported_container,exported_namespace,exported_pod, Hostname) (min_over_time(DCGM_FI_DEV_MEM_COPY_UTIL{exported_namespace != '', exported_container != '', exported_pod != ''}[15m])) * on(exported_namespace) group_left(namespace) label_replace(kube_namespace_labels{label_cost_management_optimizations='true'}, 'exported_namespace', '$1', 'namespace', '(.*)'))",
		"ros:accelerator_memory_copy_percentage_max": "(max by (modelName, exported_container, exported_namespace, exported_pod, Hostname) (max_over_time(DCGM_FI_DEV_MEM_COPY_UTIL{exported_namespace != '', exported_container != '', exported_pod != ''}[15m])) * on(exported_namespace) group_left(namespace) label_replace(kube_namespace_labels{label_insights_cost_management_optimizations='true'}, 'exported_namespace', '$1', 'namespace', '(.*)') or max by (modelName,exported_container,exported_namespace,exported_pod, Hostname) (max_over_time(DCGM_FI_DEV_MEM_COPY_UTIL{exported_namespace != '', exported_container != '', exported_pod != ''}[15m])) * on(exported_namespace) group_left(namespace) label_replace(kube_namespace_labels{label_cost_management_optimizations='true'}, 'exported_namespace', '$1', 'namespace', '(.*)'))",
		"ros:accelerator_memory_copy_percentage_avg": "(avg by (modelName, exported_container, exported_namespace, exported_pod, Hostname) (avg_over_time(DCGM_FI_DEV_MEM_COPY_UTIL{exported_namespace != '', exported_container != '', exported_pod != ''}[15m])) * on(exported_namespace) group_left(namespace) label_replace(kube_namespace_labels{label_insights_cost_management_optimizations='true'}, 'exported_namespace', '$1', 'namespace', '(.*)') or avg by (modelName, exported_container, exported_namespace, exported_pod, Hostname) (avg_over_time(DCGM_FI_DEV_MEM_COPY_UTIL{exported_namespace != '', exported_container != '', exported_pod != ''}[15m])) * on(exported_namespace) group_left(namespace) label_replace(kube_namespace_labels{label_cost_management_optimizations='true'}, 'exported_namespace', '$1', 'namespace', '(.*)'))",
		"ros:accelerator_frame_buffer_usage_min":     "(min by (modelName, GPU_I_PROFILE, exported_container, exported_namespace, exported_pod, Hostname) (min_over_time(DCGM_FI_DEV_FB_USED{exported_namespace != '', exported_container != '', exported_pod != ''}[15m])) * on(exported_namespace) group_left(namespace) label_replace(kube_namespace_labels{label_insights_cost_management_optimizations='true'}, 'exported_namespace', '$1', 'namespace', '(.*)') or min by (modelName, GPU_I_PROFILE, exported_container, exported_namespace, exported_pod, Hostname) (min_over_time(DCGM_FI_DEV_FB_USED{exported_namespace != '', exported_container != '', exported_pod != ''}[15m])) * on(exported_namespace) group_left(namespace) label_replace(kube_namespace_labels{label_cost_management_optimizations='true'}, 'exported_namespace', '$1', 'namespace', '(.*)'))",
		"ros:accelerator_frame_buffer_usage_max":     "(max by (modelName, GPU_I_PROFILE, exported_container, exported_namespace, exported_pod, Hostname) (max_over_time(DCGM_FI_DEV_FB_USED{exported_namespace != '', exported_container != '', exported_pod != ''}[15m])) * on(exported_namespace) group_left(namespace) label_replace(kube_namespace_labels{label_insights_cost_management_optimizations='true'}, 'exported_namespace', '$1', 'namespace', '(.*)') or max by (modelName, GPU_I_PROFILE, exported_container, exported_namespace, exported_pod, Hostname) (max_over_time(DCGM_FI_DEV_FB_USED{exported_namespace != '', exported_container != '', exported_pod != ''}[15m])) * on(exported_namespace) group_left(namespace) label_replace(kube_namespace_labels{label_cost_management_optimizations='true'}, 'exported_namespace', '$1', 'namespace', '(.*)'))",
		"ros:accelerator_frame_buffer_usage_avg":     "(avg by (modelName, GPU_I_PROFILE, exported_container, exported_namespace, exported_pod, Hostname) (avg_over_time(DCGM_FI_DEV_FB_USED{exported_namespace != '', exported_container != '', exported_pod != ''}[15m])) * on(exported_namespace) group_left(namespace) label_replace(kube_namespace_labels{label_insights_cost_management_optimizations='true'}, 'exported_namespace', '$1', 'namespace', '(.*)') or avg by (modelName, GPU_I_PROFILE, exported_container, exported_namespace, exported_pod, Hostname) (avg_over_time(DCGM_FI_DEV_FB_USED{exported_namespace != '', exported_container != '', exported_pod != ''}[15m])) * on(exported_namespace) group_left(namespace) label_replace(kube_namespace_labels{label_cost_management_optimizations='true'}, 'exported_namespace', '$1', 'namespace', '(.*)'))",

		// resource optimization namespace metrics queries
		"ros:cpu_request_namespace_sum":      "(sum by (namespace) (kube_resourcequota{resource='requests.cpu', type='hard'}) * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'} or sum by (namespace) (kube_resourcequota{resource='requests.cpu', type='hard'}) * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})",
		"ros:cpu_limit_namespace_sum":        "(sum by (namespace) (kube_resourcequota{ resource='limits.cpu', type='hard'}) * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'} or sum by (namespace) (kube_resourcequota{ resource='limits.cpu', type='hard'}) * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})",
		"ros:cpu_usage_namespace_avg":        "(avg_over_time(sum by(namespace) (node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{container!='', container!='POD', pod!=''})[15m:]) * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'} or avg_over_time(sum by(namespace) (node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{container!='', container!='POD', pod!=''})[15m:]) * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})",
		"ros:cpu_usage_namespace_max":        "(max_over_time(sum by(namespace) (node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{container!='', container!='POD', pod!=''})[15m:]) * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'} or max_over_time(sum by(namespace) (node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{container!='', container!='POD', pod!=''})[15m:]) * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})",
		"ros:cpu_usage_namespace_min":        "(min_over_time(sum by(namespace) (node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{container!='', container!='POD', pod!=''})[15m:]) * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'} or min_over_time(sum by(namespace) (node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{container!='', container!='POD', pod!=''})[15m:]) * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})",
		"ros:cpu_throttle_namespace_avg":     "(avg_over_time(sum by(namespace) (rate(container_cpu_cfs_throttled_seconds_total{container!='', container!='POD', pod!=''}[5m]))[15m:]) * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'} or avg_over_time(sum by(namespace) (rate(container_cpu_cfs_throttled_seconds_total{container!='', container!='POD', pod!=''}[5m]))[15m:]) * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})",
		"ros:cpu_throttle_namespace_max":     "(max_over_time(sum by(namespace) (rate(container_cpu_cfs_throttled_seconds_total{container!='', container!='POD', pod!=''}[5m]))[15m:]) * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'} or max_over_time(sum by(namespace) (rate(container_cpu_cfs_throttled_seconds_total{container!='', container!='POD', pod!=''}[5m]))[15m:]) * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})",
		"ros:cpu_throttle_namespace_min":     "(min_over_time(sum by(namespace) (rate(container_cpu_cfs_throttled_seconds_total{container!='', container!='POD', pod!=''}[5m]))[15m:]) * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'} or min_over_time(sum by(namespace) (rate(container_cpu_cfs_throttled_seconds_total{container!='', container!='POD', pod!=''}[5m]))[15m:]) * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})",
		"ros:memory_request_namespace_sum":   "(sum by (namespace) (kube_resourcequota{ resource='requests.memory', type='hard'}) * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'} or sum by (namespace) (kube_resourcequota{ resource='requests.memory', type='hard'}) * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})",
		"ros:memory_limit_namespace_sum":     "(sum by (namespace) (kube_resourcequota{ resource='limits.memory', type='hard'}) * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'} or sum by (namespace) (kube_resourcequota{ resource='limits.memory', type='hard'}) * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})",
		"ros:memory_usage_namespace_avg":     "(avg_over_time(sum by(namespace) (container_memory_working_set_bytes{container!='', container!='POD', pod!=''})[15m:]) * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'} or avg_over_time(sum by(namespace) (container_memory_working_set_bytes{container!='', container!='POD', pod!=''})[15m:]) * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})",
		"ros:memory_usage_namespace_max":     "(max_over_time(sum by(namespace) (container_memory_working_set_bytes{container!='', container!='POD', pod!=''})[15m:]) * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'} or max_over_time(sum by(namespace) (container_memory_working_set_bytes{container!='', container!='POD', pod!=''})[15m:]) * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})",
		"ros:memory_usage_namespace_min":     "(min_over_time(sum by(namespace) (container_memory_working_set_bytes{container!='', container!='POD', pod!=''})[15m:]) * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'} or min_over_time(sum by(namespace) (container_memory_working_set_bytes{container!='', container!='POD', pod!=''})[15m:]) * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})",
		"ros:memory_rss_usage_namespace_avg": "(avg_over_time(sum by(namespace) (container_memory_rss{container!='', container!='POD', pod!=''})[15m:]) * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'} or avg_over_time(sum by(namespace) (container_memory_rss{container!='', container!='POD', pod!=''})[15m:]) * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})",
		"ros:memory_rss_usage_namespace_max": "(max_over_time(sum by(namespace) (container_memory_rss{container!='', container!='POD', pod!=''})[15m:]) * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'} or max_over_time(sum by(namespace) (container_memory_rss{container!='', container!='POD', pod!=''})[15m:]) * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})",
		"ros:memory_rss_usage_namespace_min": "(min_over_time(sum by(namespace) (container_memory_rss{container!='', container!='POD', pod!=''})[15m:]) * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'} or min_over_time(sum by(namespace) (container_memory_rss{container!='', container!='POD', pod!=''})[15m:]) * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})",
		"ros:namespace_running_pods_max":     "(max_over_time(sum by(namespace) (kube_pod_status_phase{phase='Running'})[15m:]) * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'} or max_over_time(sum by(namespace) (kube_pod_status_phase{phase='Running'})[15m:]) * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})",
		"ros:namespace_running_pods_avg":     "(avg_over_time(sum by(namespace) (kube_pod_status_phase{phase='Running'})[15m:]) * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'} or avg_over_time(sum by(namespace) (kube_pod_status_phase{phase='Running'})[15m:]) * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})",
		"ros:namespace_total_pods_max":       "(max_over_time(sum by(namespace) (kube_pod_info)[15m:]) * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'} or max_over_time(sum by(namespace) (kube_pod_info)[15m:]) * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})",
		"ros:namespace_total_pods_avg":       "(avg_over_time(sum by(namespace) (kube_pod_info)[15m:]) * on(namespace) group_left kube_namespace_labels{label_insights_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'} or avg_over_time(sum by(namespace) (kube_pod_info)[15m:]) * on(namespace) group_left kube_namespace_labels{label_cost_management_optimizations='true', namespace!~'kube-.*|openshift|openshift-.*'})",
	}

	rosNamespaceFilter = query{
		Name:        "ros-namespace-filter",
		QueryString: QueryMap["ros:namespace_filter"],
		MetricKey:   staticFields{"namespace": "namespace"},
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
			RowKey: []model.LabelName{"persistentvolume"},
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
			MetricKey:   staticFields{"node": "node"},
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
			MetricKey:      staticFields{"storageclass": "storageclass", "persistentvolume": "persistentvolume", "csi_driver": "csi_driver", "csi_volume_handle": "csi_volume_handle"},
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
			QueryString:    QueryMap["cost:pod_labels"],
			MetricKey:      staticFields{"pod": "pod", "namespace": "namespace"},
			MetricKeyRegex: regexFields{"pod_labels": "label_*"},
			RowKey:         []model.LabelName{"pod", "namespace"},
		},
	}
	vmQueries = &querys{
		query{
			Name:        "vm_cpu_limit_cores",
			QueryString: QueryMap["cost:vm_cpu_limit_cores"],
			MetricKey: staticFields{
				"name":      "name",
				"namespace": "namespace",
			},
			QueryValue: &saveQueryValue{
				ValName:         "vm_cpu_limit_cores",
				Method:          "max",
				TransformedName: "vm_cpu_limit_core_seconds",
			},
			RowKey: []model.LabelName{"name", "namespace"},
		},
		query{
			Name:        "vm_cpu_request_cores",
			QueryString: QueryMap["cost:vm_cpu_request_cores"],
			MetricKey: staticFields{
				"name":      "name",
				"namespace": "namespace",
			},
			QueryValue: &saveQueryValue{
				ValName:         "vm_cpu_request_cores",
				Method:          "max",
				TransformedName: "vm_cpu_request_core_seconds",
			},
			RowKey: []model.LabelName{"name", "namespace"},
		},
		query{
			Name:        "vm_cpu_request_sockets",
			QueryString: QueryMap["cost:vm_cpu_request_sockets"],
			MetricKey: staticFields{
				"name":      "name",
				"namespace": "namespace",
			},
			QueryValue: &saveQueryValue{
				ValName:         "vm_cpu_request_sockets",
				Method:          "max",
				TransformedName: "vm_cpu_request_socket_seconds",
			},
			RowKey: []model.LabelName{"name", "namespace"},
		},
		query{
			Name:        "vm_cpu_request_threads",
			QueryString: QueryMap["cost:vm_cpu_request_threads"],
			MetricKey: staticFields{
				"name":      "name",
				"namespace": "namespace",
			},
			QueryValue: &saveQueryValue{
				ValName:         "vm_cpu_request_threads",
				Method:          "max",
				TransformedName: "vm_cpu_request_thread_seconds",
			},
			RowKey: []model.LabelName{"name", "namespace"},
		},
		query{
			Name:        "vm_cpu_usage",
			QueryString: QueryMap["cost:vm_cpu_usage"],
			MetricKey: staticFields{
				"name":      "name",
				"namespace": "namespace",
			},
			QueryValue: &saveQueryValue{
				ValName:         "vm_cpu_usage",
				Method:          "sum",
				TransformedName: "vm_cpu_usage_total_seconds",
			},
			RowKey: []model.LabelName{"name", "namespace"},
		},
		query{
			Name:        "vm_memory_limit_bytes",
			QueryString: QueryMap["cost:vm_memory_limit_bytes"],
			MetricKey: staticFields{
				"name":      "name",
				"namespace": "namespace",
			},
			QueryValue: &saveQueryValue{
				ValName:         "vm_memory_limit_bytes",
				Method:          "max",
				TransformedName: "vm_memory_limit_byte_seconds",
			},
			RowKey: []model.LabelName{"name", "namespace"},
		},
		query{
			Name:        "vm_memory_request_bytes",
			QueryString: QueryMap["cost:vm_memory_request_bytes"],
			MetricKey: staticFields{
				"name":      "name",
				"namespace": "namespace",
				"resource":  "resource",
			},
			QueryValue: &saveQueryValue{
				ValName:         "vm_memory_request_bytes",
				Method:          "max",
				TransformedName: "vm_memory_request_byte_seconds",
			},
			RowKey: []model.LabelName{"name", "namespace"},
		},
		query{
			Name:        "vm_memory_usage",
			QueryString: QueryMap["cost:vm_memory_usage_bytes"],
			MetricKey: staticFields{
				"name":      "name",
				"namespace": "namespace",
			},
			QueryValue: &saveQueryValue{
				ValName:         "vm_memory_usage_bytes",
				Method:          "sum",
				TransformedName: "vm_memory_usage_byte_seconds",
			},
			RowKey: []model.LabelName{"name", "namespace"},
		},
		query{
			Name:        "vm_info",
			QueryString: QueryMap["cost:vm_info"],
			MetricKey: staticFields{
				"node":                "node",
				"provider_id":         "provider_id",
				"name":                "name",
				"namespace":           "namespace",
				"instance_type":       "instance_type",
				"os":                  "os",
				"guest_os_arch":       "guest_os_arch",
				"guest_os_name":       "guest_os_name",
				"guest_os_version_id": "guest_os_version_id",
			},
			QueryValue: &saveQueryValue{
				Method:          "sum",
				TransformedName: "vm_uptime_total_seconds",
			},
			RowKey: []model.LabelName{"name", "namespace"},
		},
		query{
			Name:        "vm_disk_allocated_size",
			QueryString: QueryMap["cost:vm_disk_allocated_size_bytes"],
			MetricKey: staticFields{
				"name":                       "name",
				"namespace":                  "namespace",
				"device":                     "device",
				"volume_mode":                "volume_mode",
				"persistentvolumeclaim_name": "persistentvolumeclaim",
			},
			QueryValue: &saveQueryValue{
				ValName:         "vm_disk_allocated_size_bytes",
				Method:          "max",
				TransformedName: "vm_disk_allocated_size_byte_seconds",
			},
			RowKey: []model.LabelName{"name", "namespace"},
		},
		query{
			Name:        "vm_labels",
			QueryString: QueryMap["cost:vm_labels"],
			MetricKey: staticFields{
				"name":      "name",
				"namespace": "namespace",
			},
			MetricKeyRegex: regexFields{"vm_labels": "label_*"},
			RowKey:         []model.LabelName{"name", "namespace"},
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
	costNvidiaGpuQueries = &querys{
		query{
			Name:        "nvidia-gpu-memory-capacity-mib",
			QueryString: QueryMap["cost:nvidia_gpu_capacity_memory_mib"],
			MetricKey: staticFields{
				"pod":                     "pod",
				"namespace":               "namespace",
				"node":                    "node",
				"vendor_name":             "resource",
				"gpu_memory_capacity_mib": "label_nvidia_com_gpu_memory",
			},
			RowKey: []model.LabelName{"pod", "namespace", "node"},
		},
		query{
			Name:        "nvidia-gpu-utilization",
			QueryString: QueryMap["cost:nvidia_gpu_utilization"],
			MetricKey: staticFields{
				"node":       "Hostname",
				"namespace":  "exported_namespace",
				"pod":        "exported_pod",
				"gpu_uuid":   "UUID",
				"model_name": "modelName",
			},
			QueryValue: &saveQueryValue{
				Method:          "sum",
				TransformedName: "nvidia-gpu-pod-uptime-seconds",
			},
			RowKey: []model.LabelName{"exported_pod", "exported_namespace", "Hostname", "UUID"},
		},
	}
	rosContainerQueries = &querys{
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
			Name:        "cpu-throttle-container-min",
			QueryString: QueryMap["ros:cpu_throttle_container_min"],
			MetricKey:   staticFields{"container_name": "container", "pod": "pod", "namespace": "namespace", "node": "node"},
			QueryValue: &saveQueryValue{
				ValName: "cpu-throttle-container-min",
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
		query{
			Name:        "accelerator-core-usage-percentage-min",
			QueryString: QueryMap["ros:accelerator_core_usage_percentage_min"],
			MetricKey: staticFields{
				"accelerator_model_name": "modelName",
				"container":              "exported_container",
				"namespace":              "exported_namespace",
				"pod":                    "exported_pod",
				"node":                   "Hostname",
			},
			QueryValue: &saveQueryValue{
				ValName: "accelerator-core-usage-percentage-min",
			},
			RowKey: []model.LabelName{"exported_container", "exported_pod", "exported_namespace"},
		},
		query{
			Name:        "accelerator-core-usage-percentage-max",
			QueryString: QueryMap["ros:accelerator_core_usage_percentage_max"],
			MetricKey: staticFields{
				"accelerator_model_name": "modelName",
				"container":              "exported_container",
				"namespace":              "exported_namespace",
				"pod":                    "exported_pod",
				"node":                   "Hostname",
			},
			QueryValue: &saveQueryValue{
				ValName: "accelerator-core-usage-percentage-max",
			},
			RowKey: []model.LabelName{"exported_container", "exported_pod", "exported_namespace"},
		},
		query{
			Name:        "accelerator-core-usage-percentage-avg",
			QueryString: QueryMap["ros:accelerator_core_usage_percentage_avg"],
			MetricKey: staticFields{
				"accelerator_model_name": "modelName",
				"container":              "exported_container",
				"namespace":              "exported_namespace",
				"pod":                    "exported_pod",
				"node":                   "Hostname",
			},
			QueryValue: &saveQueryValue{
				ValName: "accelerator-core-usage-percentage-avg",
			},
			RowKey: []model.LabelName{"exported_container", "exported_pod", "exported_namespace"},
		},
		query{
			Name:        "accelerator-memory-copy-percentage-min",
			QueryString: QueryMap["ros:accelerator_memory_copy_percentage_min"],
			MetricKey: staticFields{
				"accelerator_model_name": "modelName",
				"container":              "exported_container",
				"namespace":              "exported_namespace",
				"pod":                    "exported_pod",
				"node":                   "Hostname",
			},
			QueryValue: &saveQueryValue{
				ValName: "accelerator-memory-copy-percentage-min",
			},
			RowKey: []model.LabelName{"exported_container", "exported_pod", "exported_namespace"},
		},
		query{
			Name:        "accelerator-memory-copy-percentage-max",
			QueryString: QueryMap["ros:accelerator_memory_copy_percentage_max"],
			MetricKey: staticFields{
				"accelerator_model_name": "modelName",
				"container":              "exported_container",
				"namespace":              "exported_namespace",
				"pod":                    "exported_pod",
				"node":                   "Hostname",
			},
			QueryValue: &saveQueryValue{
				ValName: "accelerator-memory-copy-percentage-max",
			},
			RowKey: []model.LabelName{"exported_container", "exported_pod", "exported_namespace"},
		},
		query{
			Name:        "accelerator-memory-copy-percentage-avg",
			QueryString: QueryMap["ros:accelerator_memory_copy_percentage_avg"],
			MetricKey: staticFields{
				"accelerator_model_name": "modelName",
				"container":              "exported_container",
				"namespace":              "exported_namespace",
				"pod":                    "exported_pod",
				"node":                   "Hostname",
			},
			QueryValue: &saveQueryValue{
				ValName: "accelerator-memory-copy-percentage-avg",
			},
			RowKey: []model.LabelName{"exported_container", "exported_pod", "exported_namespace"},
		},
		query{
			Name:        "accelerator-frame-buffer-usage-min",
			QueryString: QueryMap["ros:accelerator_frame_buffer_usage_min"],
			MetricKey: staticFields{
				"accelerator_model_name":   "modelName",
				"container":                "exported_container",
				"namespace":                "exported_namespace",
				"pod":                      "exported_pod",
				"node":                     "Hostname",
				"accelerator_profile_name": "GPU_I_PROFILE",
			},
			QueryValue: &saveQueryValue{
				ValName: "accelerator-frame-buffer-usage-min",
			},
			RowKey: []model.LabelName{"exported_container", "exported_pod", "exported_namespace"},
		},
		query{
			Name:        "accelerator-frame-buffer-usage-max",
			QueryString: QueryMap["ros:accelerator_frame_buffer_usage_max"],
			MetricKey: staticFields{
				"accelerator_model_name":   "modelName",
				"container":                "exported_container",
				"namespace":                "exported_namespace",
				"pod":                      "exported_pod",
				"node":                     "Hostname",
				"accelerator_profile_name": "GPU_I_PROFILE",
			},
			QueryValue: &saveQueryValue{
				ValName: "accelerator-frame-buffer-usage-max",
			},
			RowKey: []model.LabelName{"exported_container", "exported_pod", "exported_namespace"},
		},
		query{
			Name:        "accelerator-frame-buffer-usage-avg",
			QueryString: QueryMap["ros:accelerator_frame_buffer_usage_avg"],
			MetricKey: staticFields{
				"accelerator_model_name":   "modelName",
				"container":                "exported_container",
				"namespace":                "exported_namespace",
				"pod":                      "exported_pod",
				"node":                     "Hostname",
				"accelerator_profile_name": "GPU_I_PROFILE",
			},
			QueryValue: &saveQueryValue{
				ValName: "accelerator-frame-buffer-usage-avg",
			},
			RowKey: []model.LabelName{"exported_container", "exported_pod", "exported_namespace"},
		},
	}

	rosNamespaceQueries = &querys{
		query{
			Name:        "cpu-request-namespace-sum",
			QueryString: QueryMap["ros:cpu_request_namespace_sum"],
			MetricKey:   staticFields{"namespace": "namespace"},
			QueryValue: &saveQueryValue{
				ValName: "cpu-request-namespace-sum",
			},
			RowKey: []model.LabelName{"namespace"},
		},
		query{
			Name:        "cpu-limit-namespace-sum",
			QueryString: QueryMap["ros:cpu_limit_namespace_sum"],
			MetricKey:   staticFields{"namespace": "namespace"},
			QueryValue: &saveQueryValue{
				ValName: "cpu-limit-namespace-sum",
			},
			RowKey: []model.LabelName{"namespace"},
		},
		query{
			Name:        "cpu-usage-namespace-avg",
			QueryString: QueryMap["ros:cpu_usage_namespace_avg"],
			MetricKey:   staticFields{"namespace": "namespace"},
			QueryValue: &saveQueryValue{
				ValName: "cpu-usage-namespace-avg",
			},
			RowKey: []model.LabelName{"namespace"},
		},
		query{
			Name:        "cpu-usage-namespace-max",
			QueryString: QueryMap["ros:cpu_usage_namespace_max"],
			MetricKey:   staticFields{"namespace": "namespace"},
			QueryValue: &saveQueryValue{
				ValName: "cpu-usage-namespace-max",
			},
			RowKey: []model.LabelName{"namespace"},
		},
		query{
			Name:        "cpu-usage-namespace-min",
			QueryString: QueryMap["ros:cpu_usage_namespace_min"],
			MetricKey:   staticFields{"namespace": "namespace"},
			QueryValue: &saveQueryValue{
				ValName: "cpu-usage-namespace-min",
			},
			RowKey: []model.LabelName{"namespace"},
		},
		query{
			Name:        "cpu-throttle-namespace-avg",
			QueryString: QueryMap["ros:cpu_throttle_namespace_avg"],
			MetricKey:   staticFields{"namespace": "namespace"},
			QueryValue: &saveQueryValue{
				ValName: "cpu-throttle-namespace-avg",
			},
			RowKey: []model.LabelName{"namespace"},
		},
		query{
			Name:        "cpu-throttle-namespace-max",
			QueryString: QueryMap["ros:cpu_throttle_namespace_max"],
			MetricKey:   staticFields{"namespace": "namespace"},
			QueryValue: &saveQueryValue{
				ValName: "cpu-throttle-namespace-max",
			},
			RowKey: []model.LabelName{"namespace"},
		},
		query{
			Name:        "cpu-throttle-namespace-min",
			QueryString: QueryMap["ros:cpu_throttle_namespace_min"],
			MetricKey:   staticFields{"namespace": "namespace"},
			QueryValue: &saveQueryValue{
				ValName: "cpu-throttle-namespace-min",
			},
			RowKey: []model.LabelName{"namespace"},
		},
		query{
			Name:        "memory-request-namespace-sum",
			QueryString: QueryMap["ros:memory_request_namespace_sum"],
			MetricKey:   staticFields{"namespace": "namespace"},
			QueryValue: &saveQueryValue{
				ValName: "memory-request-namespace-sum",
			},
			RowKey: []model.LabelName{"namespace"},
		},
		query{
			Name:        "memory-limit-namespace-sum",
			QueryString: QueryMap["ros:memory_limit_namespace_sum"],
			MetricKey:   staticFields{"namespace": "namespace"},
			QueryValue: &saveQueryValue{
				ValName: "memory-limit-namespace-sum",
			},
			RowKey: []model.LabelName{"namespace"},
		},
		query{
			Name:        "memory-usage-namespace-avg",
			QueryString: QueryMap["ros:memory_usage_namespace_avg"],
			MetricKey:   staticFields{"namespace": "namespace"},
			QueryValue: &saveQueryValue{
				ValName: "memory-usage-namespace-avg",
			},
			RowKey: []model.LabelName{"namespace"},
		},
		query{
			Name:        "memory-usage-namespace-max",
			QueryString: QueryMap["ros:memory_usage_namespace_max"],
			MetricKey:   staticFields{"namespace": "namespace"},
			QueryValue: &saveQueryValue{
				ValName: "memory-usage-namespace-max",
			},
			RowKey: []model.LabelName{"namespace"},
		},
		query{
			Name:        "memory-usage-namespace-min",
			QueryString: QueryMap["ros:memory_usage_namespace_min"],
			MetricKey:   staticFields{"namespace": "namespace"},
			QueryValue: &saveQueryValue{
				ValName: "memory-usage-namespace-min",
			},
			RowKey: []model.LabelName{"namespace"},
		},
		query{
			Name:        "memory-rss-usage-namespace-avg",
			QueryString: QueryMap["ros:memory_rss_usage_namespace_avg"],
			MetricKey:   staticFields{"namespace": "namespace"},
			QueryValue: &saveQueryValue{
				ValName: "memory-rss-usage-namespace-avg",
			},
			RowKey: []model.LabelName{"namespace"},
		},
		query{
			Name:        "memory-rss-usage-namespace-max",
			QueryString: QueryMap["ros:memory_rss_usage_namespace_max"],
			MetricKey:   staticFields{"namespace": "namespace"},
			QueryValue: &saveQueryValue{
				ValName: "memory-rss-usage-namespace-max",
			},
			RowKey: []model.LabelName{"namespace"},
		},
		query{
			Name:        "memory-rss-usage-namespace-min",
			QueryString: QueryMap["ros:memory_rss_usage_namespace_min"],
			MetricKey:   staticFields{"namespace": "namespace"},
			QueryValue: &saveQueryValue{
				ValName: "memory-rss-usage-namespace-min",
			},
			RowKey: []model.LabelName{"namespace"},
		},
		query{
			Name:        "pods-running-namespace-max",
			QueryString: QueryMap["ros:namespace_running_pods_max"],
			MetricKey:   staticFields{"namespace": "namespace"},
			QueryValue: &saveQueryValue{
				ValName: "pods-running-namespace-max",
			},
			RowKey: []model.LabelName{"namespace"},
		},
		query{
			Name:        "pods-running-namespace-avg",
			QueryString: QueryMap["ros:namespace_running_pods_avg"],
			MetricKey:   staticFields{"namespace": "namespace"},
			QueryValue: &saveQueryValue{
				ValName: "pods-running-namespace-avg",
			},
			RowKey: []model.LabelName{"namespace"},
		},
		query{
			Name:        "pods-total-namespace-max",
			QueryString: QueryMap["ros:namespace_total_pods_max"],
			MetricKey:   staticFields{"namespace": "namespace"},
			QueryValue: &saveQueryValue{
				ValName: "pods-total-namespace-max",
			},
			RowKey: []model.LabelName{"namespace"},
		},
		query{
			Name:        "pods-total-namespace-avg",
			QueryString: QueryMap["ros:namespace_total_pods_avg"],
			MetricKey:   staticFields{"namespace": "namespace"},
			QueryValue: &saveQueryValue{
				ValName: "pods-total-namespace-avg",
			},
			RowKey: []model.LabelName{"namespace"},
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
