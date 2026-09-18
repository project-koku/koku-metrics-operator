//
// Copyright 2026 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package rpms

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMinimumEVR(t *testing.T) {
	if got := MinimumEVR("openssl"); got != "1:3.5.8-1.el9_8" {
		t.Errorf("MinimumEVR(openssl) = %q, want %q", got, "1:3.5.8-1.el9_8")
	}
	if got := MinimumEVR("openssl-libs"); got != "1:3.5.8-1.el9_8" {
		t.Errorf("MinimumEVR(openssl-libs) = %q, want %q", got, "1:3.5.8-1.el9_8")
	}
	if got := MinimumEVR("unknown-package"); got != "" {
		t.Errorf("MinimumEVR(unknown-package) = %q, want empty", got)
	}
}

func TestCompareEVR(t *testing.T) {
	cases := []struct {
		left, right string
		want        int
	}{
		{left: "1:3.5.5-6.el9_8", right: "1:3.5.8-1.el9_8", want: -1},
		{left: "1:3.5.8-1.el9_8", right: "1:3.5.8-1.el9_8", want: 0},
		{left: "1:3.5.9-1.el9_8", right: "1:3.5.8-1.el9_8", want: 1},
		{left: "0:3.5.8-1.el9_8", right: "1:3.5.5-6.el9_8", want: -1},
		{left: "2:1.0.0-1", right: "1:9.0.0-1", want: 1},
		{left: "3.5.8-1.el9_8", right: "1:3.5.8-1.el9_8", want: -1},
		{left: "1:3.5.8-2.el9_8", right: "1:3.5.8-1.el9_8", want: 1},
		{left: "1:3.5.8-1.el9_8", right: "1:3.5.8-2.el9_8", want: -1},
		// Lexicographic string order wrongly treats "9" > "10"; RPM numeric segments do not.
		{left: "1:3.5.8-9.el9_8", right: "1:3.5.8-10.el9_8", want: -1},
		{left: "1:3.5.8-10.el9_8", right: "1:3.5.8-9.el9_8", want: 1},
		{left: "1:3.5-1", right: "1:3.5.0-1", want: -1}, // trailing .0 is a newer segment in rpmvercmp
		{left: "1:3.5.0.1-1", right: "1:3.5.0-1", want: 1},
		{left: "1:3.5-1", right: "1:3.5.1-1", want: -1},
		{left: "1:3.5.8", right: "1:3.5.8-1", want: -1},
		{left: "1:3.5.8~rc1-1", right: "1:3.5.8-1", want: -1},
	}

	for _, tc := range cases {
		got := compareEVR(tc.left, tc.right)
		if got != tc.want {
			t.Errorf("compareEVR(%q, %q) = %d, want %d", tc.left, tc.right, got, tc.want)
		}
	}
}

func TestCompareRelease(t *testing.T) {
	cases := []struct {
		left, right string
		want        int
	}{
		{left: "1.el9_8", right: "1.el9_8", want: 0},
		{left: "1.el9_8", right: "2.el9_8", want: -1},
		{left: "2.el9_8", right: "1.el9_8", want: 1},
		{left: "9.el9_8", right: "10.el9_8", want: -1},
		{left: "10.el9_8", right: "9.el9_8", want: 1},
	}

	for _, tc := range cases {
		got := compareRelease(tc.left, tc.right)
		if got != tc.want {
			t.Errorf("compareRelease(%q, %q) = %d, want %d", tc.left, tc.right, got, tc.want)
		}
	}
}

func TestRpmvercmp(t *testing.T) {
	cases := []struct {
		left, right string
		want        int
	}{
		{left: "1.0", right: "1.0", want: 0},
		{left: "1.0", right: "1.1", want: -1},
		{left: "1.10", right: "1.9", want: 1},
		{left: "1.0", right: "1.0.0", want: -1},
		{left: "1.0~rc1", right: "1.0", want: -1},
		{left: "1.0", right: "1.0~rc1", want: 1},
		{left: "1.0~rc1", right: "1.0~rc2", want: -1},
		{left: "1.0^1", right: "1.0", want: 1},
		{left: "1.0", right: "1.0^1", want: -1},
		{left: "1.0^1", right: "1.0^2", want: -1},
		{left: "", right: "", want: 0},
		{left: "abc", right: "abd", want: -1},
		{left: "abd", right: "abc", want: 1},
	}

	for _, tc := range cases {
		got := rpmvercmp(tc.left, tc.right)
		if got != tc.want {
			t.Errorf("rpmvercmp(%q, %q) = %d, want %d", tc.left, tc.right, got, tc.want)
		}
	}
}

func TestPackageEVR(t *testing.T) {
	packages := []packageEntry{
		{Name: "openssl", EVR: "1:3.5.8-1.el9_8"},
		{Name: "ca-certificates", EVR: "2025.1-1.el9"},
	}

	if evr, ok := packageEVR(packages, "openssl"); !ok || evr != "1:3.5.8-1.el9_8" {
		t.Errorf("packageEVR(openssl) = (%q, %v), want (1:3.5.8-1.el9_8, true)", evr, ok)
	}
	if evr, ok := packageEVR(packages, "missing"); ok || evr != "" {
		t.Errorf("packageEVR(missing) = (%q, %v), want (\"\", false)", evr, ok)
	}
}

// TestVerifyLockfile_OpenSSLMinimum guards the downstream RPM lockfile against
// regressions for RHSA-172172 after the MintMaker lockfile refresh (PR #1079).
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
	err := VerifyLockfile(lockfile)
	if err == nil {
		t.Fatal("expected below-minimum lockfile to fail verification")
	}
	if !strings.Contains(err.Error(), "below minimum") {
		t.Errorf("error = %v, want message containing %q", err, "below minimum")
	}
}

func TestVerifyLockfile_meetsMinimum(t *testing.T) {
	lockfile := filepath.Join("testdata", "openssl-meets-minimum.lock.yaml")
	if err := VerifyLockfile(lockfile); err != nil {
		t.Fatalf("expected meeting-minimum lockfile to pass: %v", err)
	}
}

func TestVerifyLockfile_missingPackage(t *testing.T) {
	lockfile := filepath.Join("testdata", "openssl-missing-package.lock.yaml")
	err := VerifyLockfile(lockfile)
	if err == nil {
		t.Fatal("expected missing-package lockfile to fail verification")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %v, want message containing %q", err, "not found")
	}
}

func TestVerifyLockfile_emptyArches(t *testing.T) {
	lockfile := filepath.Join("testdata", "empty-arches.lock.yaml")
	err := VerifyLockfile(lockfile)
	if err == nil {
		t.Fatal("expected empty-arches lockfile to fail verification")
	}
	if !strings.Contains(err.Error(), "no architectures found") {
		t.Errorf("error = %v, want message containing %q", err, "no architectures found")
	}
}

func TestVerifyLockfile_readError(t *testing.T) {
	err := VerifyLockfile(filepath.Join("testdata", "does-not-exist.lock.yaml"))
	if err == nil {
		t.Fatal("expected missing file to fail verification")
	}
	if !strings.Contains(err.Error(), "read lockfile") {
		t.Errorf("error = %v, want message containing %q", err, "read lockfile")
	}
}

func TestVerifyLockfile_parseError(t *testing.T) {
	err := VerifyLockfile(filepath.Join("testdata", "invalid.yaml"))
	if err == nil {
		t.Fatal("expected invalid YAML to fail verification")
	}
	if !strings.Contains(err.Error(), "parse lockfile") {
		t.Errorf("error = %v, want message containing %q", err, "parse lockfile")
	}
}
