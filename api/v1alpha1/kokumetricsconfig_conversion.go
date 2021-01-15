/*


Copyright 2021 Red Hat, Inc.

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

package v1alpha1

import (
	kokumetricscfgv1alpha2 "github.com/project-koku/koku-metrics-operator/api/v1alpha2"
	apiconversion "k8s.io/apimachinery/pkg/conversion"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// ConvertTo converts this KokuMetricsConfig to the Hub version (v1alpha2).
func (src *KokuMetricsConfig) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*kokumetricscfgv1alpha2.KokuMetricsConfig)
	if err := Convert_v1alpha1_KokuMetricsConfig_To_v1alpha2_KokuMetricsConfig(src, dst, nil); err != nil {
		return err
	}
	dst.Spec.ReportingCycle = src.Spec.Upload.UploadCycle
	dst.Status.ReportingCycle = src.Status.Upload.UploadCycle
	return nil
}

// ConvertFrom converts from the KokuMetricsConfig Hub version (v1alpha2) to this version.
func (dst *KokuMetricsConfig) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*kokumetricscfgv1alpha2.KokuMetricsConfig)
	return Convert_v1alpha2_KokuMetricsConfig_To_v1alpha1_KokuMetricsConfig(src, dst, nil)
}

// ConvertTo converts this KokuMetricsConfigList to the Hub version (v1alpha2).
func (src *KokuMetricsConfigList) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*kokumetricscfgv1alpha2.KokuMetricsConfigList)
	return Convert_v1alpha1_KokuMetricsConfigList_To_v1alpha2_KokuMetricsConfigList(src, dst, nil)
}

// ConvertFrom converts from the KokuMetricsConfigList Hub version (v1alpha2) to this version.
func (dst *KokuMetricsConfigList) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*kokumetricscfgv1alpha2.KokuMetricsConfigList)
	return Convert_v1alpha2_KokuMetricsConfigList_To_v1alpha1_KokuMetricsConfigList(src, dst, nil)
}

func Convert_v1alpha2_KokuMetricsConfigSpec_To_v1alpha1_KokuMetricsConfigSpec(in *kokumetricscfgv1alpha2.KokuMetricsConfigSpec, out *KokuMetricsConfigSpec, s apiconversion.Scope) error {
	if err := autoConvert_v1alpha2_KokuMetricsConfigSpec_To_v1alpha1_KokuMetricsConfigSpec(in, out, s); err != nil {
		return err
	}
	out.Upload.UploadCycle = in.ReportingCycle
	return nil
}

func Convert_v1alpha2_KokuMetricsConfigStatus_To_v1alpha1_KokuMetricsConfigStatus(in *kokumetricscfgv1alpha2.KokuMetricsConfigStatus, out *KokuMetricsConfigStatus, s apiconversion.Scope) error {
	if err := autoConvert_v1alpha2_KokuMetricsConfigStatus_To_v1alpha1_KokuMetricsConfigStatus(in, out, s); err != nil {
		return err
	}
	out.Upload.UploadCycle = in.ReportingCycle
	return nil
}

func Convert_v1alpha1_UploadSpec_To_v1alpha2_UploadSpec(in *UploadSpec, out *kokumetricscfgv1alpha2.UploadSpec, s apiconversion.Scope) error {
	out.IngressAPIPath = in.IngressAPIPath
	out.UploadWait = in.UploadWait
	out.UploadToggle = in.UploadToggle
	out.ValidateCert = in.ValidateCert
	return nil
}

func Convert_v1alpha1_UploadStatus_To_v1alpha2_UploadStatus(in *UploadStatus, out *kokumetricscfgv1alpha2.UploadStatus, s apiconversion.Scope) error {
	out.IngressAPIPath = in.IngressAPIPath
	out.UploadWait = in.UploadWait
	out.UploadToggle = in.UploadToggle
	out.ValidateCert = in.ValidateCert
	return nil
}
