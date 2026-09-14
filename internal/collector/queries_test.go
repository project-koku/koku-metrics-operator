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

	if !strings.Contains(pvcVolumenameJoinRHS, "kube_persistentvolume_status_phase{phase='Bound'}") {
		t.Fatal("pvcVolumenameJoinRHS must filter to Bound persistent volumes")
	}

	for _, key := range storageQueryKeys {
		query, ok := QueryMap[key]
		if !ok {
			t.Fatalf("missing QueryMap entry %q", key)
		}
		if !strings.Contains(query, pvcVolumenameJoinRHS) {
			t.Errorf("query %q must use shared pvcVolumenameJoinRHS for a unique (namespace, persistentvolumeclaim) join", key)
		}
		if strings.Contains(query, legacyRHS) {
			t.Errorf("query %q still uses legacy PVC info join without Bound PV filter", key)
		}
	}
}
