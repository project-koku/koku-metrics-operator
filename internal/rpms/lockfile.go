//
// Copyright 2026 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package rpms

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

// lockfile mirrors the subset of rpms.lock.yaml needed for verification.
type lockfile struct {
	Arches []archEntry `yaml:"arches"`
}

type archEntry struct {
	Arch     string         `yaml:"arch"`
	Packages []packageEntry `yaml:"packages"`
}

type packageEntry struct {
	Name string `yaml:"name"`
	EVR  string `yaml:"evr"`
}

// MinimumEVR returns the minimum acceptable EVR for a package name, or empty if
// no minimum is defined.
func MinimumEVR(packageName string) string {
	return minimumPackageEVRs[packageName]
}

var minimumPackageEVRs = map[string]string{
	// RHSA-172172 (RHSA-2026:67165): openssl security, bug fix, and enhancement update.
	"openssl":      "1:3.5.8-1.el9_8",
	"openssl-libs": "1:3.5.8-1.el9_8",
}

// VerifyLockfile reads path and ensures pinned packages meet configured minimum EVRs
// on every architecture present in the lockfile.
func VerifyLockfile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read lockfile %q: %w", path, err)
	}

	var lock lockfile
	if err := yaml.Unmarshal(data, &lock); err != nil {
		return fmt.Errorf("parse lockfile %q: %w", path, err)
	}

	if len(lock.Arches) == 0 {
		return fmt.Errorf("lockfile %q: no architectures found", path)
	}

	var violations []string
	for _, arch := range lock.Arches {
		for packageName, minimumEVR := range minimumPackageEVRs {
			pkgEVR, found := packageEVR(arch.Packages, packageName)
			if !found {
				violations = append(violations, fmt.Sprintf(
					"arch %s: package %q not found in lockfile",
					arch.Arch, packageName,
				))
				continue
			}

			if compareEVR(pkgEVR, minimumEVR) < 0 {
				violations = append(violations, fmt.Sprintf(
					"arch %s: package %q EVR %q is below minimum %q",
					arch.Arch, packageName, pkgEVR, minimumEVR,
				))
			}
		}
	}

	if len(violations) > 0 {
		return fmt.Errorf("lockfile %q failed minimum EVR checks:\n  %s",
			path, strings.Join(violations, "\n  "))
	}

	return nil
}

func packageEVR(packages []packageEntry, name string) (string, bool) {
	for _, pkg := range packages {
		if pkg.Name == name {
			return pkg.EVR, true
		}
	}
	return "", false
}

type parsedEVR struct {
	epoch   int
	version string
	release string
}

func parseEVR(evr string) parsedEVR {
	rest := evr
	epoch := 0

	if idx := strings.Index(evr, ":"); idx >= 0 {
		epoch, _ = strconv.Atoi(evr[:idx])
		rest = evr[idx+1:]
	}

	parts := strings.SplitN(rest, "-", 2)
	release := ""
	if len(parts) > 1 {
		release = parts[1]
	}

	return parsedEVR{
		epoch:   epoch,
		version: parts[0],
		release: release,
	}
}

// compareEVR returns -1 if left < right, 0 if equal, 1 if left > right.
func compareEVR(left, right string) int {
	l := parseEVR(left)
	r := parseEVR(right)

	if l.epoch != r.epoch {
		if l.epoch < r.epoch {
			return -1
		}
		return 1
	}

	if cmp := compareVersion(l.version, r.version); cmp != 0 {
		return cmp
	}

	return compareRelease(l.release, r.release)
}

func compareVersion(left, right string) int {
	lParts := strings.Split(left, ".")
	rParts := strings.Split(right, ".")
	maxLen := len(lParts)
	if len(rParts) > maxLen {
		maxLen = len(rParts)
	}

	for i := 0; i < maxLen; i++ {
		lVal := 0
		rVal := 0

		if i < len(lParts) {
			lVal, _ = strconv.Atoi(lParts[i])
		}
		if i < len(rParts) {
			rVal, _ = strconv.Atoi(rParts[i])
		}

		if lVal < rVal {
			return -1
		}
		if lVal > rVal {
			return 1
		}
	}

	return 0
}

func compareRelease(left, right string) int {
	if left == right {
		return 0
	}
	if left < right {
		return -1
	}
	return 1
}
