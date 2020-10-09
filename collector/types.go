package collector

import (
	"encoding/csv"
	"io"
	"strings"
	"time"
)

type DateTimes struct {
	ReportPeriodStart string
	ReportPeriodEnd   string
	IntervalStart     string
	IntervalEnd       string
}

func NewDates(t time.Time) *DateTimes {
	d := new(DateTimes)
	d.IntervalEnd = t.String()
	start := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
	d.IntervalStart = start.String()
	d.ReportPeriodStart = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location()).String()
	d.ReportPeriodEnd = time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, t.Location()).String()
	return d
}

type CSVThing interface {
	CSVheader(w io.Writer) error
	CSVrow(w io.Writer) error
	RowString() []string
	String() string
}

type NamespaceRow struct {
	*DateTimes
	Namespace       string
	NamespaceLabels string `json:"namespace_labels"`
}

func NewNamespaceRow(t time.Time) NamespaceRow {
	row := NamespaceRow{}
	row.DateTimes = NewDates(t)
	return row
}

func (NamespaceRow) CSVheader(w io.Writer) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{
		"report_period_start",
		"report_period_end",
		"interval_start",
		"interval_end",
		"namespace",
		"namespace_labels"}); err != nil {
		return err
	}
	cw.Flush()
	return nil
}

func (row NamespaceRow) CSVrow(w io.Writer) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(row.RowString()); err != nil {
		return err
	}
	cw.Flush()
	return nil
}

func (row NamespaceRow) RowString() []string {
	return []string{
		row.ReportPeriodStart,
		row.ReportPeriodEnd,
		row.IntervalStart,
		row.IntervalEnd,
		row.Namespace,
		row.NamespaceLabels,
	}
}

func (row NamespaceRow) String() string {
	return strings.Join(row.RowString(), ",")
}

type NodeRow struct {
	*DateTimes
	Node                          string
	NodeCapacityCPUCores          string `json:"node-capacity-cpu-cores"`
	ModeCapacityCPUCoreSeconds    string `json:"node-capacity-cpu-core-seconds"`
	NodeCapacityMemoryBytes       string `json:"node-capacity-memory-bytes"`
	NodeCapacityMemoryByteSeconds string `json:"node-capacity-memory-byte-seconds"`
	ResourceID                    string `json:"resource_id"`
	NodeLabels                    string `json:"node_labels"`
}

func NewNodeRow(t time.Time) NodeRow {
	row := NodeRow{}
	row.DateTimes = NewDates(t)
	return row
}

func (NodeRow) CSVheader(w io.Writer) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{
		"report_period_start",
		"report_period_end",
		"interval_start",
		"interval_end",
		"node",
		"node_capacity_cpu_cores",
		"node_capacity_cpu_core_seconds",
		"node_capacity_memory_bytes",
		"node_capacity_memory_byte_seconds",
		"resource_id",
		"node_labels"}); err != nil {
		return err
	}
	cw.Flush()
	return nil
}

func (row NodeRow) CSVrow(w io.Writer) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(row.RowString()); err != nil {
		return err
	}
	cw.Flush()
	return nil
}

func (row NodeRow) RowString() []string {
	return []string{
		row.ReportPeriodStart,
		row.ReportPeriodEnd,
		row.IntervalStart,
		row.IntervalEnd,
		row.Node,
		row.NodeCapacityCPUCores,
		row.ModeCapacityCPUCoreSeconds,
		row.NodeCapacityMemoryBytes,
		row.NodeCapacityMemoryByteSeconds,
		row.ResourceID,
		row.NodeLabels,
	}
}

func (row NodeRow) String() string {
	return strings.Join(row.RowString(), ",")
}

type PodRow struct {
	*DateTimes
	Node                          string
	Namespace                     string
	Pod                           string
	PodUsageCPUCoreSeconds        string `json:"pod-usage-cpu-core-seconds"`
	PodRequestCPUCoreSeconds      string `json:"pod-request-cpu-core-seconds"`
	PodLimitCPUCoreSeconds        string `json:"pod-limit-cpu-core-seconds"`
	PodUsageMemoryByteSeconds     string `json:"pod-usage-memory-byte-seconds"`
	PodRequestMemoryByteSeconds   string `json:"pod-request-memory-byte-seconds"`
	PodLimitMemoryByteSeconds     string `json:"pod-limit-memory-byte-seconds"`
	NodeCapacityCPUCores          string `json:"node-capacity-cpu-cores"`
	ModeCapacityCPUCoreSeconds    string `json:"node-capacity-cpu-core-seconds"`
	NodeCapacityMemoryBytes       string `json:"node-capacity-memory-bytes"`
	NodeCapacityMemoryByteSeconds string `json:"node-capacity-memory-byte-seconds"`
	ResourceID                    string `json:"resource_id"`
	PodLabels                     string `json:"pod_labels"`
}

func NewPodRow(t time.Time) PodRow {
	row := PodRow{}
	row.DateTimes = NewDates(t)
	return row
}

func (PodRow) CSVheader(w io.Writer) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{
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
		"resource_id",
		"pod_labels"}); err != nil {
		return err
	}
	cw.Flush()
	return nil
}

func (row PodRow) CSVrow(w io.Writer) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(row.RowString()); err != nil {
		return err
	}
	cw.Flush()
	return nil
}

func (row PodRow) RowString() []string {
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
		row.ResourceID,
		row.PodLabels,
	}
}

func (row PodRow) String() string {
	return strings.Join(row.RowString(), ",")
}

type StorageRow struct {
	*DateTimes
	Namespace                                string
	Pod                                      string
	PersistentVolumeClaim                    string `json:"persistentvolumeclaim"`
	PersistentVolume                         string `json:"persistentvolume"`
	StorageClass                             string `json:"storageclass"`
	PersistentVolumeClaimCapacityBytes       string `json:"persistentvolumeclaim-capacity-bytes"`
	PersistentVolumeClaimCapacityByteSeconds string `json:"persistentvolumeclaim-capacity-byte-seconds"`
	VolumeRequestStorageByteSeconds          string `json:"persistentvolumeclaim-request-byte-seconds"`
	PersistentVolumeClaimUsageByteSeconds    string `json:"persistentvolumeclaim-usage-byte-seconds"`
	PersistentVolumeLabels                   string `json:"persistentvolume_labels"`
	PersistentVolumeClaimLabels              string `json:"persistentvolumeclaim_labels"`
}

func NewStorageRow(t time.Time) StorageRow {
	row := StorageRow{}
	row.DateTimes = NewDates(t)
	return row
}

func (StorageRow) CSVheader(w io.Writer) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{
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
		"persistentvolumeclaim_labels"}); err != nil {
		return err
	}
	cw.Flush()
	return nil
}

func (row StorageRow) CSVrow(w io.Writer) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(row.RowString()); err != nil {
		return err
	}
	cw.Flush()
	return nil
}

func (row StorageRow) RowString() []string {
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

func (row StorageRow) String() string {
	return strings.Join(row.RowString(), ",")
}
