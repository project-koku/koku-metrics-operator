//
// Copyright 2026 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package v1beta1

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestAddToSchemeRegistersMetricsConfigTypes(t *testing.T) {
	scheme := runtime.NewScheme()

	if err := AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme() error = %v", err)
	}

	checkRegisteredKind(t, scheme, &CostManagementMetricsConfig{}, GroupVersion.WithKind("CostManagementMetricsConfig"))
	checkRegisteredKind(t, scheme, &CostManagementMetricsConfigList{}, GroupVersion.WithKind("CostManagementMetricsConfigList"))
}

func checkRegisteredKind(t *testing.T, scheme *runtime.Scheme, obj runtime.Object, expected schema.GroupVersionKind) {
	t.Helper()

	gvks, unversioned, err := scheme.ObjectKinds(obj)
	if err != nil {
		t.Fatalf("ObjectKinds(%T) error = %v", obj, err)
	}
	if unversioned {
		t.Fatalf("ObjectKinds(%T) returned unversioned kind", obj)
	}
	if len(gvks) == 0 {
		t.Fatalf("ObjectKinds(%T) returned no kinds", obj)
	}

	for _, gvk := range gvks {
		if gvk == expected {
			return
		}
	}

	t.Fatalf("ObjectKinds(%T) = %v, want %v", obj, gvks, expected)
}
