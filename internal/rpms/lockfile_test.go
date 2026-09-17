//
// Copyright 2026 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package rpms

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompareEVR(t *testing.T) {
	cases := []struct {
		left, right string
		want        int
	}{
		{left: "1:3.5.5-6.el9_8", right: "1:3.5.8-1.el9_8", want: -1},
		{left: "1:3.5.8-1.el9_8", right: "1:3.5.8-1.el9_8", want: 0},
		{left: "1:3.5.9-1.el9_8", right: "1:3.5.8-1.el9_8", want: 1},
		{left: "0:3.5.8-1.el9_8", right: "1:3.5.5-6.el9_8", want: -1},
	}

	for _, tc := range cases {
		got := compareEVR(tc.left, tc.right)
		if got != tc.want {
			t.Errorf("compareEVR(%q, %q) = %d, want %d", tc.left, tc.right, got, tc.want)
		}
	}
}

// TestVerifyLockfile_OpenSSLMinimum guards the downstream RPM lockfile against
// regressions for RHSA-172172. Merge the MintMaker lockfile refresh (PR #1079)
// before this test can pass on downstream.
func TestVerifyLockfile_OpenSSLMinimum(t *testing.T) {
	lockfilePath := filepath.Join("..", "..", "rpms.lock.yaml")
	if _, err := os.Stat(lockfilePath); os.IsNotExist(err) {
		t.Skip("rpms.lock.yaml not present (downstream-only)")
	}

	if err := VerifyLockfile(lockfilePath); err != nil {
		t.Fatalf("expected rpms.lock.yaml to meet minimum OpenSSL EVRs for RHSA-172172: %v", err)
	}
}

func TestVerifyLockfile_detectsBelowMinimum(t *testing.T) {
	lockfile := filepath.Join("testdata", "openssl-below-minimum.lock.yaml")
	if err := VerifyLockfile(lockfile); err == nil {
		t.Fatal("expected below-minimum lockfile to fail verification")
	}
}
