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

package v1alpha1

import (
	kokumetricscfgv1beta1 "github.com/project-koku/koku-metrics-operator/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// ConvertTo converts this KokuMetricsConfig to the Hub version (v1beta1).
func (src *KokuMetricsConfig) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*kokumetricscfgv1beta1.KokuMetricsConfig)
	return Convert_v1alpha1_KokuMetricsConfig_To_v1beta1_KokuMetricsConfig(src, dst, nil)
}

// ConvertFrom converts from the KokuMetricsConfig Hub version (v1beta1) to this version.
func (dst *KokuMetricsConfig) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*kokumetricscfgv1beta1.KokuMetricsConfig)
	return Convert_v1beta1_KokuMetricsConfig_To_v1alpha1_KokuMetricsConfig(src, dst, nil)
}

// ConvertTo converts this KokuMetricsConfigList to the Hub version (v1beta1).
func (src *KokuMetricsConfigList) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*kokumetricscfgv1beta1.KokuMetricsConfigList)
	return Convert_v1alpha1_KokuMetricsConfigList_To_v1beta1_KokuMetricsConfigList(src, dst, nil)
}

// ConvertFrom converts from the KokuMetricsConfigList Hub version (v1beta1) to this version.
func (dst *KokuMetricsConfigList) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*kokumetricscfgv1beta1.KokuMetricsConfigList)
	return Convert_v1beta1_KokuMetricsConfigList_To_v1alpha1_KokuMetricsConfigList(src, dst, nil)
}
