//
// Copyright 2021 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package collector

import (
	"strings"
	"testing"
)

func TestStorageQueriesUseUniquePvcVolumenameJoin(t *testing.T) {
	storageQueryKeys := []string{
		"cost:persistentvolume_pod_info",
		"cost:persistentvolumeclaim_request_bytes",
		"cost:persistentvolumeclaim_usage_bytes",
		"cost:persistentvolumeclaim_labels",
	}

	legacyRHS := `max by(namespace, persistentvolumeclaim, volumename) (kube_persistentvolumeclaim_info{volumename != ''})`

	requiredJoinRHSClauses := []string{
		"max by(namespace, persistentvolumeclaim, volumename)",
		"kube_persistentvolumeclaim_info{volumename != ''}",
		"* on(volumename) group_left()",
		"label_replace(",
		`kube_persistentvolume_status_phase{phase='Bound'}`,
		`"persistentvolume", "(.+)"`,
	}
	for _, clause := range requiredJoinRHSClauses {
		if !strings.Contains(pvcVolumenameJoinRHS, clause) {
			t.Fatalf("pvcVolumenameJoinRHS missing required clause %q", clause)
		}
	}

	for _, key := range storageQueryKeys {
		query, ok := QueryMap[key]
		if !ok {
			t.Fatalf("missing QueryMap entry %q", key)
		}
		if !strings.Contains(query, "on(persistentvolumeclaim, namespace) group_left(volumename)") {
			t.Errorf("query %q must join on (persistentvolumeclaim, namespace) with group_left(volumename)", key)
		}
		if !strings.Contains(query, pvcVolumenameJoinRHS) {
			t.Errorf("query %q must use shared pvcVolumenameJoinRHS for a unique (namespace, persistentvolumeclaim) join", key)
		}
		if strings.Contains(query, legacyRHS) {
			t.Errorf("query %q still uses legacy PVC info join without Bound PV filter", key)
		}
	}
}
