//
// Copyright 2021 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package collector

import (
	"strings"
	"time"

	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

type dateTimes struct {
	ReportPeriodStart string
	ReportPeriodEnd   string
	IntervalStart     string
	IntervalEnd       string
}

func newDates(ts *promv1.Range) *dateTimes {
	d := new(dateTimes)
	d.IntervalStart = ts.Start.String()
	d.IntervalEnd = ts.End.String()
	t := ts.Start
	d.ReportPeriodStart = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location()).String()
	d.ReportPeriodEnd = time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, t.Location()).String()
	return d
}

func (dt dateTimes) csvRow() []string {
	return []string{
		dt.ReportPeriodStart,
		dt.ReportPeriodEnd,
		dt.IntervalStart,
		dt.IntervalEnd,
	}
}

func (dt dateTimes) string() string { return strings.Join(dt.csvRow(), ",") }

type csvStruct interface {
	csvHeader() []string
	csvRow() []string
	string() string
}

func newNamespaceRow(ts *promv1.Range) namespaceRow { return namespaceRow{dateTimes: newDates(ts)} }
func newNodeRow(ts *promv1.Range) nodeRow           { return nodeRow{dateTimes: newDates(ts)} }
func newPodRow(ts *promv1.Range) podRow             { return podRow{dateTimes: newDates(ts)} }
func newStorageRow(ts *promv1.Range) storageRow     { return storageRow{dateTimes: newDates(ts)} }
func newVMRow(ts *promv1.Range) vmRow               { return vmRow{dateTimes: newDates(ts)} }
func newROSRow(ts *promv1.Range) resourceOptimizationRow {
	return resourceOptimizationRow{dateTimes: newDates(ts)}
}
func newROSNamespaceRow(ts *promv1.Range) rosNamespaceRow {
	return rosNamespaceRow{dateTimes: newDates(ts)}
}

type namespaceRow struct {
	*dateTimes
	Namespace       string `mapstructure:"namespace"`
	NamespaceLabels string `mapstructure:"namespace_labels"`
}

func (namespaceRow) csvHeader() []string {
	return []string{
		"report_period_start",
		"report_period_end",
		"interval_start",
		"interval_end",
		"namespace",
		"namespace_labels"}
}

func (row namespaceRow) csvRow() []string {
	return []string{
		row.ReportPeriodStart,
		row.ReportPeriodEnd,
		row.IntervalStart,
		row.IntervalEnd,
		row.Namespace,
		row.NamespaceLabels,
	}
}

func (row namespaceRow) string() string { return strings.Join(row.csvRow(), ",") }

type nodeRow struct {
	*dateTimes
	Node                          string `mapstructure:"node"`
	NodeCapacityCPUCores          string `mapstructure:"node-capacity-cpu-cores"`
	ModeCapacityCPUCoreSeconds    string `mapstructure:"node-capacity-cpu-core-seconds"`
	NodeCapacityMemoryBytes       string `mapstructure:"node-capacity-memory-bytes"`
	NodeCapacityMemoryByteSeconds string `mapstructure:"node-capacity-memory-byte-seconds"`
	NodeRole                      string `mapstructure:"node-role"`
	ResourceID                    string `mapstructure:"resource_id"`
	NodeLabels                    string `mapstructure:"node_labels"`
}

func (nodeRow) csvHeader() []string {
	return []string{
		"report_period_start",
		"report_period_end",
		"interval_start",
		"interval_end",
		"node",
		// "node_capacity_cpu_cores",  // if Node and Pod reports are ever separated, these lines can be uncommented
		// "node_capacity_cpu_core_seconds",
		// "node_capacity_memory_bytes",
		// "node_capacity_memory_byte_seconds",
		// "node_role",
		// "resource_id",
		"node_labels"}
}

func (row nodeRow) csvRow() []string {
	return []string{
		row.ReportPeriodStart,
		row.ReportPeriodEnd,
		row.IntervalStart,
		row.IntervalEnd,
		row.Node,
		// row.NodeCapacityCPUCores,  // if Node and Pod reports are ever separated, these lines can be uncommented
		// row.ModeCapacityCPUCoreSeconds,
		// row.NodeCapacityMemoryBytes,
		// row.NodeCapacityMemoryByteSeconds,
		// row.NodeRole,
		// row.ResourceID,
		row.NodeLabels,
	}
}

func (row nodeRow) string() string { return strings.Join(row.csvRow(), ",") }

type podRow struct {
	*dateTimes
	nodeRow
	Namespace                   string `mapstructure:"namespace"`
	Pod                         string `mapstructure:"pod"`
	PodUsageCPUCoreSeconds      string `mapstructure:"pod-usage-cpu-core-seconds"`
	PodRequestCPUCoreSeconds    string `mapstructure:"pod-request-cpu-core-seconds"`
	PodLimitCPUCoreSeconds      string `mapstructure:"pod-limit-cpu-core-seconds"`
	PodUsageMemoryByteSeconds   string `mapstructure:"pod-usage-memory-byte-seconds"`
	PodRequestMemoryByteSeconds string `mapstructure:"pod-request-memory-byte-seconds"`
	PodLimitMemoryByteSeconds   string `mapstructure:"pod-limit-memory-byte-seconds"`
	PodLabels                   string `mapstructure:"pod_labels"`
}

func (podRow) csvHeader() []string {
	return []string{
		"report_period_start",
		"report_period_end",
		"interval_start",
		"interval_end",
		"node",
		"namespace",
		"pod",
		"pod_usage_cpu_core_seconds",
		"pod_request_cpu_core_seconds",
		"pod_limit_cpu_core_seconds",
		"pod_usage_memory_byte_seconds",
		"pod_request_memory_byte_seconds",
		"pod_limit_memory_byte_seconds",
		"node_capacity_cpu_cores",
		"node_capacity_cpu_core_seconds",
		"node_capacity_memory_bytes",
		"node_capacity_memory_byte_seconds",
		"node_role",
		"resource_id",
		"pod_labels"}
}

func (row podRow) csvRow() []string {
	return []string{
		row.ReportPeriodStart,
		row.ReportPeriodEnd,
		row.IntervalStart,
		row.IntervalEnd,
		row.Node,
		row.Namespace,
		row.Pod,
		row.PodUsageCPUCoreSeconds,
		row.PodRequestCPUCoreSeconds,
		row.PodLimitCPUCoreSeconds,
		row.PodUsageMemoryByteSeconds,
		row.PodRequestMemoryByteSeconds,
		row.PodLimitMemoryByteSeconds,
		row.NodeCapacityCPUCores,
		row.ModeCapacityCPUCoreSeconds,
		row.NodeCapacityMemoryBytes,
		row.NodeCapacityMemoryByteSeconds,
		row.NodeRole,
		row.ResourceID,
		row.PodLabels,
	}
}

func (row podRow) string() string { return strings.Join(row.csvRow(), ",") }

type storageRow struct {
	*dateTimes
	Namespace                                string
	Pod                                      string
	Node                                     string `mapstructure:"node"`
	PersistentVolumeClaim                    string `mapstructure:"persistentvolumeclaim"`
	PersistentVolume                         string `mapstructure:"persistentvolume"`
	StorageClass                             string `mapstructure:"storageclass"`
	CSIDriver                                string `mapstructure:"csi_driver"`
	CSIVolumeHandle                          string `mapstructure:"csi_volume_handle"`
	PersistentVolumeClaimCapacityBytes       string `mapstructure:"persistentvolumeclaim-capacity-bytes"`
	PersistentVolumeClaimCapacityByteSeconds string `mapstructure:"persistentvolumeclaim-capacity-byte-seconds"`
	VolumeRequestStorageByteSeconds          string `mapstructure:"persistentvolumeclaim-request-byte-seconds"`
	PersistentVolumeClaimUsageByteSeconds    string `mapstructure:"persistentvolumeclaim-usage-byte-seconds"`
	PersistentVolumeLabels                   string `mapstructure:"persistentvolume_labels"`
	PersistentVolumeClaimLabels              string `mapstructure:"persistentvolumeclaim_labels"`
}

func (storageRow) csvHeader() []string {
	return []string{
		"report_period_start",
		"report_period_end",
		"interval_start",
		"interval_end",
		"namespace",
		"pod",
		"node",
		"persistentvolumeclaim",
		"persistentvolume",
		"storageclass",
		"csi_driver",
		"csi_volume_handle",
		"persistentvolumeclaim_capacity_bytes",
		"persistentvolumeclaim_capacity_byte_seconds",
		"volume_request_storage_byte_seconds",
		"persistentvolumeclaim_usage_byte_seconds",
		"persistentvolume_labels",
		"persistentvolumeclaim_labels"}
}

func (row storageRow) csvRow() []string {
	return []string{
		row.ReportPeriodStart,
		row.ReportPeriodEnd,
		row.IntervalStart,
		row.IntervalEnd,
		row.Namespace,
		row.Pod,
		row.Node,
		row.PersistentVolumeClaim,
		row.PersistentVolume,
		row.StorageClass,
		row.CSIDriver,
		row.CSIVolumeHandle,
		row.PersistentVolumeClaimCapacityBytes,
		row.PersistentVolumeClaimCapacityByteSeconds,
		row.VolumeRequestStorageByteSeconds,
		row.PersistentVolumeClaimUsageByteSeconds,
		row.PersistentVolumeLabels,
		row.PersistentVolumeClaimLabels,
	}
}

func (row storageRow) string() string { return strings.Join(row.csvRow(), ",") }

type vmRow struct {
	*dateTimes
	Node                      string `mapstructure:"node"`
	ResourceID                string `mapstructure:"resource_id"`
	Namespace                 string `mapstructure:"namespace"`
	VMName                    string `mapstructure:"name"`
	InstanceType              string `mapstructure:"instance_type"`
	OS                        string `mapstructure:"os"`
	GuestOSArch               string `mapstructure:"guest_os_arch"`
	GuestOSName               string `mapstructure:"guest_os_name"`
	GuestOSVersionId          string `mapstructure:"guest_os_version_id"`
	UptimeSeconds             string `mapstructure:"vm_uptime_total_seconds"`
	CPULimitCores             string `mapstructure:"vm_cpu_limit_cores"`
	CPULimitCoreSeconds       string `mapstructure:"vm_cpu_limit_core_seconds"`
	CPURequestCores           string `mapstructure:"vm_cpu_request_cores"`
	CPURequestCoreSeconds     string `mapstructure:"vm_cpu_request_core_seconds"`
	CPURequestSockets         string `mapstructure:"vm_cpu_request_sockets"`
	CPURequestSocketSeconds   string `mapstructure:"vm_cpu_request_socket_seconds"`
	CPURequestThreads         string `mapstructure:"vm_cpu_request_threads"`
	CPURequestThreadSeconds   string `mapstructure:"vm_cpu_request_thread_seconds"`
	CPUUsageSeconds           string `mapstructure:"vm_cpu_usage_total_seconds"`
	MemoryLimitBytes          string `mapstructure:"vm_memory_limit_bytes"`
	MemoryLimitByteSeconds    string `mapstructure:"vm_memory_limit_byte_seconds"`
	MemoryRequestBytes        string `mapstructure:"vm_memory_request_bytes"`
	MemoryRequestByteSeconds  string `mapstructure:"vm_memory_request_byte_seconds"`
	MemoryUsageBytes          string `mapstructure:"vm_memory_usage_byte_seconds"`
	Device                    string `mapstructure:"device"`
	VolumeMode                string `mapstructure:"volume_mode"`
	PersistentVolumeClaimName string `mapstructure:"persistentvolumeclaim_name"`
	DiskAllocatedSizeBytes    string `mapstructure:"vm_disk_allocated_size_byte_seconds"`
	VMLabels                  string `mapstructure:"vm_labels"`
}

func (vmRow) csvHeader() []string {
	return []string{
		"report_period_start",
		"report_period_end",
		"interval_start",
		"interval_end",
		"node",
		"resource_id",
		"namespace",
		"vm_name",
		"vm_instance_type",
		"vm_os",
		"vm_guest_os_arch",
		"vm_guest_os_name",
		"vm_guest_os_version",
		"vm_uptime_total_seconds",
		"vm_cpu_limit_cores",
		"vm_cpu_limit_core_seconds",
		"vm_cpu_request_cores",
		"vm_cpu_request_core_seconds",
		"vm_cpu_request_sockets",
		"vm_cpu_request_socket_seconds",
		"vm_cpu_request_threads",
		"vm_cpu_request_thread_seconds",
		"vm_cpu_usage_total_seconds",
		"vm_memory_limit_bytes",
		"vm_memory_limit_byte_seconds",
		"vm_memory_request_bytes",
		"vm_memory_request_byte_seconds",
		"vm_memory_usage_byte_seconds",
		"vm_device",
		"vm_volume_mode",
		"vm_persistentvolumeclaim_name",
		"vm_disk_allocated_size_byte_seconds",
		"vm_labels",
	}
}

func (row vmRow) csvRow() []string {
	return []string{
		row.ReportPeriodStart,
		row.ReportPeriodEnd,
		row.IntervalStart,
		row.IntervalEnd,
		row.Node,
		row.ResourceID,
		row.Namespace,
		row.VMName,
		row.InstanceType,
		row.OS,
		row.GuestOSArch,
		row.GuestOSName,
		row.GuestOSVersionId,
		row.UptimeSeconds,
		row.CPULimitCores,
		row.CPULimitCoreSeconds,
		row.CPURequestCores,
		row.CPURequestCoreSeconds,
		row.CPURequestSockets,
		row.CPURequestSocketSeconds,
		row.CPURequestThreads,
		row.CPURequestThreadSeconds,
		row.CPUUsageSeconds,
		row.MemoryLimitBytes,
		row.MemoryLimitByteSeconds,
		row.MemoryRequestBytes,
		row.MemoryRequestByteSeconds,
		row.MemoryUsageBytes,
		row.Device,
		row.VolumeMode,
		row.PersistentVolumeClaimName,
		row.DiskAllocatedSizeBytes,
		row.VMLabels,
	}
}

func (row vmRow) string() string { return strings.Join(row.csvRow(), ",") }

type resourceOptimizationRow struct {
	*dateTimes
	nodeRow
	ContainerName              string `mapstructure:"container_name"`
	Pod                        string `mapstructure:"pod"`
	OwnerName                  string `mapstructure:"owner_name"`
	OwnerKind                  string `mapstructure:"owner_kind"`
	Workload                   string `mapstructure:"workload"`
	WorkloadType               string `mapstructure:"workload_type"`
	Namespace                  string `mapstructure:"namespace"`
	ImageName                  string `mapstructure:"image_name"`
	CPURequestContainerAvg     string `mapstructure:"cpu-request-container-avg"`
	CPURequestContainerSum     string `mapstructure:"cpu-request-container-sum"`
	CPULimitContainerAvg       string `mapstructure:"cpu-limit-container-avg"`
	CPULimitContainerSum       string `mapstructure:"cpu-limit-container-sum"`
	CPUUsageContainerAvg       string `mapstructure:"cpu-usage-container-avg"`
	CPUUsageContainerMin       string `mapstructure:"cpu-usage-container-min"`
	CPUUsageContainerMax       string `mapstructure:"cpu-usage-container-max"`
	CPUUsageContainerSum       string `mapstructure:"cpu-usage-container-sum"`
	CPUThrottleContainerAvg    string `mapstructure:"cpu-throttle-container-avg"`
	CPUThrottleContainerMax    string `mapstructure:"cpu-throttle-container-max"`
	CPUThrottleContainerSum    string `mapstructure:"cpu-throttle-container-sum"`
	MemoryRequestContainerAvg  string `mapstructure:"memory-request-container-avg"`
	MemoryRequestContainerSum  string `mapstructure:"memory-request-container-sum"`
	MemoryLimitContainerAvg    string `mapstructure:"memory-limit-container-avg"`
	MemoryLimitContainerSum    string `mapstructure:"memory-limit-container-sum"`
	MemoryUsageContainerAvg    string `mapstructure:"memory-usage-container-avg"`
	MemoryUsageContainerMin    string `mapstructure:"memory-usage-container-min"`
	MemoryUsageContainerMax    string `mapstructure:"memory-usage-container-max"`
	MemoryUsageContainerSum    string `mapstructure:"memory-usage-container-sum"`
	MemoryRSSUsageContainerAvg string `mapstructure:"memory-rss-usage-container-avg"`
	MemoryRSSUsageContainerMin string `mapstructure:"memory-rss-usage-container-min"`
	MemoryRSSUsageContainerMax string `mapstructure:"memory-rss-usage-container-max"`
	MemoryRSSUsageContainerSum string `mapstructure:"memory-rss-usage-container-sum"`
}

func (resourceOptimizationRow) csvHeader() []string {
	return []string{
		"report_period_start",
		"report_period_end",
		"interval_start",
		"interval_end",
		"container_name",
		"pod",
		"owner_name",
		"owner_kind",
		"workload",
		"workload_type",
		"namespace",
		"image_name",
		"node",
		"resource_id",
		"cpu_request_container_avg",
		"cpu_request_container_sum",
		"cpu_limit_container_avg",
		"cpu_limit_container_sum",
		"cpu_usage_container_avg",
		"cpu_usage_container_min",
		"cpu_usage_container_max",
		"cpu_usage_container_sum",
		"cpu_throttle_container_avg",
		"cpu_throttle_container_max",
		"cpu_throttle_container_sum",
		"memory_request_container_avg",
		"memory_request_container_sum",
		"memory_limit_container_avg",
		"memory_limit_container_sum",
		"memory_usage_container_avg",
		"memory_usage_container_min",
		"memory_usage_container_max",
		"memory_usage_container_sum",
		"memory_rss_usage_container_avg",
		"memory_rss_usage_container_min",
		"memory_rss_usage_container_max",
		"memory_rss_usage_container_sum",
	}
}

func (row resourceOptimizationRow) csvRow() []string {
	return []string{
		row.ReportPeriodStart,
		row.ReportPeriodEnd,
		row.IntervalStart,
		row.IntervalEnd,
		row.ContainerName,
		row.Pod,
		row.OwnerName,
		row.OwnerKind,
		row.Workload,
		row.WorkloadType,
		row.Namespace,
		row.ImageName,
		row.Node,
		row.ResourceID,
		row.CPURequestContainerAvg,
		row.CPURequestContainerSum,
		row.CPULimitContainerAvg,
		row.CPULimitContainerSum,
		row.CPUUsageContainerAvg,
		row.CPUUsageContainerMin,
		row.CPUUsageContainerMax,
		row.CPUUsageContainerSum,
		row.CPUThrottleContainerAvg,
		row.CPUThrottleContainerMax,
		row.CPUThrottleContainerSum,
		row.MemoryRequestContainerAvg,
		row.MemoryRequestContainerSum,
		row.MemoryLimitContainerAvg,
		row.MemoryLimitContainerSum,
		row.MemoryUsageContainerAvg,
		row.MemoryUsageContainerMin,
		row.MemoryUsageContainerMax,
		row.MemoryUsageContainerSum,
		row.MemoryRSSUsageContainerAvg,
		row.MemoryRSSUsageContainerMin,
		row.MemoryRSSUsageContainerMax,
		row.MemoryRSSUsageContainerSum,
	}
}

func (row resourceOptimizationRow) string() string { return strings.Join(row.csvRow(), ",") }

type rosNamespaceRow struct {
	*dateTimes
	Namespace         string `mapstructure:"namespace"`
	CPURequestSum     string `mapstructure:"cpu-request-namespace-sum"`
	CPULimitSum       string `mapstructure:"cpu-limit-namespace-sum"`
	CPUUsageAvg       string `mapstructure:"cpu-usage-namespace-avg"`
	CPUUsageMax       string `mapstructure:"cpu-usage-namespace-max"`
	CPUUsageMin       string `mapstructure:"cpu-usage-namespace-min"`
	CPUThrottleAvg    string `mapstructure:"cpu-throttle-namespace-avg"`
	CPUThrottleMax    string `mapstructure:"cpu-throttle-namespace-max"`
	CPUThrottleMin    string `mapstructure:"cpu-throttle-namespace-min"`
	MemoryRequestSum  string `mapstructure:"memory-request-namespace-sum"`
	MemoryLimitSum    string `mapstructure:"memory-limit-namespace-sum"`
	MemoryUsageAvg    string `mapstructure:"memory-usage-namespace-avg"`
	MemoryUsageMax    string `mapstructure:"memory-usage-namespace-max"`
	MemoryUsageMin    string `mapstructure:"memory-usage-namespace-min"`
	MemoryRSSUsageAvg string `mapstructure:"memory-rss-usage-namespace-avg"`
	MemoryRSSUsageMax string `mapstructure:"memory-rss-usage-namespace-max"`
	MemoryRSSUsageMin string `mapstructure:"memory-rss-usage-namespace-min"`
	PodsRunningMax    string `mapstructure:"pods-running-namespace-max"`
	PodsRunningAvg    string `mapstructure:"pods-running-namespace-avg"`
	PodsTotalMax      string `mapstructure:"pods-total-namespace-max"`
	PodsTotalAvg      string `mapstructure:"pods-total-namespace-avg"`
}

func (rosNamespaceRow) csvHeader() []string {
	return []string{
		"report_period_start",
		"report_period_end",
		"interval_start",
		"interval_end",
		"namespace",
		"cpu_request_namespace_sum",
		"cpu_limit_namespace_sum",
		"cpu_usage_namespace_avg",
		"cpu_usage_namespace_max",
		"cpu_usage_namespace_min",
		"cpu_throttle_namespace_avg",
		"cpu_throttle_namespace_max",
		"cpu_throttle_namespace_min",
		"memory_request_namespace_sum",
		"memory_limit_namespace_sum",
		"memory_usage_namespace_avg",
		"memory_usage_namespace_max",
		"memory_usage_namespace_min",
		"memory_rss_usage_namespace_avg",
		"memory_rss_usage_namespace_max",
		"memory_rss_usage_namespace_min",
		"namespace_running_pods_max",
		"namespace_running_pods_avg",
		"namespace_total_pods_max",
		"namespace_total_pods_avg",
	}
}

func (row rosNamespaceRow) csvRow() []string {
	return []string{
		row.ReportPeriodStart,
		row.ReportPeriodEnd,
		row.IntervalStart,
		row.IntervalEnd,
		row.Namespace,
		row.CPURequestSum,
		row.CPULimitSum,
		row.CPUUsageAvg,
		row.CPUUsageMax,
		row.CPUUsageMin,
		row.CPUThrottleAvg,
		row.CPUThrottleMax,
		row.CPUThrottleMin,
		row.MemoryRequestSum,
		row.MemoryLimitSum,
		row.MemoryUsageAvg,
		row.MemoryUsageMax,
		row.MemoryUsageMin,
		row.MemoryRSSUsageAvg,
		row.MemoryRSSUsageMax,
		row.MemoryRSSUsageMin,
		row.PodsRunningMax,
		row.PodsRunningAvg,
		row.PodsTotalMax,
		row.PodsTotalAvg,
	}
}

func (row rosNamespaceRow) string() string { return strings.Join(row.csvRow(), ",") }
