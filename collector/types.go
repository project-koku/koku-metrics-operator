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
	"encoding/csv"
	"io"
	"strings"
	"time"

	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

type DateTimes struct {
	ReportPeriodStart string
	ReportPeriodEnd   string
	IntervalStart     string
	IntervalEnd       string
}

func NewDates(ts promv1.Range) *DateTimes {
	d := new(DateTimes)
	d.IntervalStart = ts.Start.String()
	d.IntervalEnd = ts.End.String()
	t := ts.Start
	d.ReportPeriodStart = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location()).String()
	d.ReportPeriodEnd = time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, t.Location()).String()
	return d
}

type CSVStruct interface {
	CSVheader(w io.Writer) error
	CSVrow(w io.Writer) error
	RowString() []string
	String() string
}

type NamespaceRow struct {
	*DateTimes
	Namespace       string `json:"namespace"`
	NamespaceLabels string `json:"namespace_labels"`
}

func NewNamespaceRow(ts promv1.Range) NamespaceRow {
	row := NamespaceRow{}
	row.DateTimes = NewDates(ts)
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
	Node                          string `json:"node"`
	NodeCapacityCPUCores          string `json:"node-capacity-cpu-cores"`
	ModeCapacityCPUCoreSeconds    string `json:"node-capacity-cpu-core-seconds"`
	NodeCapacityMemoryBytes       string `json:"node-capacity-memory-bytes"`
	NodeCapacityMemoryByteSeconds string `json:"node-capacity-memory-byte-seconds"`
	ResourceID                    string `json:"resource_id"`
	NodeLabels                    string `json:"node_labels"`
}

func NewNodeRow(ts promv1.Range) NodeRow {
	row := NodeRow{}
	row.DateTimes = NewDates(ts)
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
	NodeRow
	Namespace                   string `json:"namespace"`
	Pod                         string `json:"pod"`
	PodUsageCPUCoreSeconds      string `json:"pod-usage-cpu-core-seconds"`
	PodRequestCPUCoreSeconds    string `json:"pod-request-cpu-core-seconds"`
	PodLimitCPUCoreSeconds      string `json:"pod-limit-cpu-core-seconds"`
	PodUsageMemoryByteSeconds   string `json:"pod-usage-memory-byte-seconds"`
	PodRequestMemoryByteSeconds string `json:"pod-request-memory-byte-seconds"`
	PodLimitMemoryByteSeconds   string `json:"pod-limit-memory-byte-seconds"`
	PodLabels                   string `json:"pod_labels"`
}

func NewPodRow(ts promv1.Range) PodRow {
	row := PodRow{}
	row.DateTimes = NewDates(ts)
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

func NewStorageRow(ts promv1.Range) StorageRow {
	row := StorageRow{}
	row.DateTimes = NewDates(ts)
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
