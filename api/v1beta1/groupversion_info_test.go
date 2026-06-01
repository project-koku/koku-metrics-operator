//
// Copyright 2021 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package v1beta1

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestAddToScheme(t *testing.T) {
	s := runtime.NewScheme()
	if err := AddToScheme(s); err != nil {
		t.Fatalf("AddToScheme() error = %v", err)
	}

	for _, gvk := range []schema.GroupVersionKind{
		GroupVersion.WithKind("CostManagementMetricsConfig"),
		GroupVersion.WithKind("CostManagementMetricsConfigList"),
	} {
		if !s.Recognizes(gvk) {
			t.Errorf("scheme does not recognize %v", gvk)
		}
	}
}
