//
// Copyright 2021 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

// Package v1beta1 contains API Schema definitions for the costmanagement-metrics-cfg v1beta1 API group
// +kubebuilder:object:generate=true
// +groupName=costmanagement-metrics-cfg.openshift.io
package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// NamePrefix is the prefix used to distinguish upstream and downstream operators
	NamePrefix = "costmanagement"

	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: "costmanagement-metrics-cfg.openshift.io", Version: "v1beta1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

func addKnownTypes(s *runtime.Scheme) error {
	s.AddKnownTypes(GroupVersion,
		&CostManagementMetricsConfig{},
		&CostManagementMetricsConfigList{},
	)
	metav1.AddToGroupVersion(s, GroupVersion)
	return nil
}
