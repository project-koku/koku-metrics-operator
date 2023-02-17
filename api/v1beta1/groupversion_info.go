//
// Copyright 2021 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

// Package v1beta1 contains API Schema definitions for the koku-metrics-cfg v1beta1 API group
// +kubebuilder:object:generate=true
// +groupName=koku-metrics-cfg.openshift.io
package v1beta1

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// NamePrefix is the prefix used to distinguish upstream and downstream operators
	NamePrefix = "koku"

	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: fmt.Sprintf("%s-metrics-cfg.openshift.io", NamePrefix), Version: "v1beta1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)
