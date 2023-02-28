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
func newROSRow(ts *promv1.Range) resourceOptimizationRow {
	return resourceOptimizationRow{dateTimes: newDates(ts)}
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
	PersistentVolumeClaim                    string `mapstructure:"persistentvolumeclaim"`
	PersistentVolume                         string `mapstructure:"persistentvolume"`
	StorageClass                             string `mapstructure:"storageclass"`
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
		"persistentvolumeclaim",
		"persistentvolume",
		"storageclass",
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
		row.PersistentVolumeClaim,
		row.PersistentVolume,
		row.StorageClass,
		row.PersistentVolumeClaimCapacityBytes,
		row.PersistentVolumeClaimCapacityByteSeconds,
		row.VolumeRequestStorageByteSeconds,
		row.PersistentVolumeClaimUsageByteSeconds,
		row.PersistentVolumeLabels,
		row.PersistentVolumeClaimLabels,
	}
}

func (row storageRow) string() string { return strings.Join(row.csvRow(), ",") }

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
