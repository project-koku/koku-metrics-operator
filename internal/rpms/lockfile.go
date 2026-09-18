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
	return rpmvercmp(left, right)
}

func compareRelease(left, right string) int {
	return rpmvercmp(left, right)
}

// rpmvercmp compares RPM version/release strings using the same segment rules
// as rpm's rpmvercmp: numeric segments compare numerically, non-numeric
// lexicographically, '~' sorts before anything (including empty), '^' sorts
// after anything when the other side ends.
func rpmvercmp(left, right string) int {
	i, j := 0, 0
	for i < len(left) || j < len(right) {
		for i < len(left) && !isAlnum(left[i]) && left[i] != '~' && left[i] != '^' {
			i++
		}
		for j < len(right) && !isAlnum(right[j]) && right[j] != '~' && right[j] != '^' {
			j++
		}

		if i < len(left) && left[i] == '~' || j < len(right) && right[j] == '~' {
			if i >= len(left) || left[i] != '~' {
				return 1
			}
			if j >= len(right) || right[j] != '~' {
				return -1
			}
			i++
			j++
			continue
		}

		if i < len(left) && left[i] == '^' || j < len(right) && right[j] == '^' {
			if i >= len(left) || left[i] != '^' {
				return -1
			}
			if j >= len(right) || right[j] != '^' {
				return 1
			}
			i++
			j++
			continue
		}

		if i >= len(left) && j >= len(right) {
			return 0
		}
		if i >= len(left) {
			return -1
		}
		if j >= len(right) {
			return 1
		}

		startI, startJ := i, j
		if isDigit(left[i]) {
			for i < len(left) && isDigit(left[i]) {
				i++
			}
			for j < len(right) && isDigit(right[j]) {
				j++
			}
			lSeg := strings.TrimLeft(left[startI:i], "0")
			rSeg := strings.TrimLeft(right[startJ:j], "0")
			if len(lSeg) < len(rSeg) {
				return -1
			}
			if len(lSeg) > len(rSeg) {
				return 1
			}
			if lSeg < rSeg {
				return -1
			}
			if lSeg > rSeg {
				return 1
			}
			continue
		}

		for i < len(left) && isAlpha(left[i]) {
			i++
		}
		for j < len(right) && isAlpha(right[j]) {
			j++
		}
		lSeg := left[startI:i]
		rSeg := right[startJ:j]
		if lSeg < rSeg {
			return -1
		}
		if lSeg > rSeg {
			return 1
		}
	}
	return 0
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func isAlpha(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func isAlnum(b byte) bool {
	return isDigit(b) || isAlpha(b)
}
