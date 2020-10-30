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
	Namespace       string `mapstructure:"namespace"`
	NamespaceLabels string `mapstructure:"namespace_labels"`
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
	Node                          string `mapstructure:"node"`
	NodeCapacityCPUCores          string `mapstructure:"node-capacity-cpu-cores"`
	ModeCapacityCPUCoreSeconds    string `mapstructure:"node-capacity-cpu-core-seconds"`
	NodeCapacityMemoryBytes       string `mapstructure:"node-capacity-memory-bytes"`
	NodeCapacityMemoryByteSeconds string `mapstructure:"node-capacity-memory-byte-seconds"`
	ResourceID                    string `mapstructure:"resource_id"`
	NodeLabels                    string `mapstructure:"node_labels"`
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
		// "node_capacity_cpu_cores",  // if Node and Pod reports are ever separated, these lines can be uncommented
		// "node_capacity_cpu_core_seconds",
		// "node_capacity_memory_bytes",
		// "node_capacity_memory_byte_seconds",
		// "resource_id",
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
		// row.NodeCapacityCPUCores,
		// row.ModeCapacityCPUCoreSeconds,
		// row.NodeCapacityMemoryBytes,
		// row.NodeCapacityMemoryByteSeconds,
		// row.ResourceID,
		row.NodeLabels,
	}
}

func (row NodeRow) String() string {
	return strings.Join(row.RowString(), ",")
}

type PodRow struct {
	*DateTimes
	NodeRow
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
