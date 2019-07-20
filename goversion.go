// Copyright 2019 The GoRE.tk Authors. All rights reserved.
// Use of this source code is governed by the license that
// can be found in the LICENSE file.

package gore

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"
)

var goVersionMatcher = regexp.MustCompile(`(go[\d+\.]*(beta|rc)?[\d*])`)
var versionMarker = []byte("go")

// GoVersion holds information about the compiler version.
type GoVersion struct {
	// Name is a string representation of the version.
	Name string
	// SHA is a digest of the git commit for the release.
	SHA string
	// Timestamp is a string of the timestamp when the commit was created.
	Timestamp string
}

// ResolveGoVersion tries to return the GoVersion for the given tag.
// For example the tag: go1 will return a GoVersion struct representing version 1.0 of the compiler.
// If no goversion for the given tag is found, nil is returned.
func ResolveGoVersion(tag string) *GoVersion {
	v, ok := goversions[tag]
	if !ok {
		return nil
	}
	return v
}

// GoVersionCompare compares two version strings.
// If a < b, -1 is returned.
// If a == b, 0 is returned.
// If a > b, 1 is returned.
func GoVersionCompare(a, b string) int {
	if a == b {
		return 0
	}

	aa := strings.Split(a, ".")
	ab := strings.Split(b, ".")

	if aa[0][:2] != "go" && ab[0][:2] != "go" {
		panic("Not a go version string")
	}
	amaj, err := strconv.Atoi(aa[0][2:])
	if err != nil {
		panic(err)
	}
	bmaj, err := strconv.Atoi(ab[0][2:])
	if err != nil {
		panic(err)
	}
	if amaj < bmaj {
		return -1
	}
	if amaj > bmaj {
		return 1
	}

	if len(aa) == 1 && amaj == bmaj {
		// Same major version but a is x.0.0
		return -1
	}

	if len(ab) == 1 && amaj == bmaj {
		// Same major version but b is x.0.0
		return 1
	}

	var min string
	var abeta int
	var arc int
	var bbeta int
	var brc int
	if strings.Contains(aa[1], "beta") {
		idx := strings.Index(aa[1], "beta")
		min = aa[1][:idx]
		abeta, err = strconv.Atoi(aa[1][idx+4:])
		if err != nil {
			panic(err)
		}
	} else if strings.Contains(aa[1], "rc") {
		idx := strings.Index(aa[1], "rc")
		min = aa[1][:idx]
		arc, err = strconv.Atoi(aa[1][idx+2:])
		if err != nil {
			panic(err)
		}
	} else {
		min = aa[1]
	}
	amin, err := strconv.Atoi(min)
	if err != nil {
		panic(err)
	}
	if strings.Contains(ab[1], "beta") {
		idx := strings.Index(ab[1], "beta")
		min = ab[1][:idx]
		bbeta, err = strconv.Atoi(ab[1][idx+4:])
		if err != nil {
			panic(err)
		}
	} else if strings.Contains(ab[1], "rc") {
		idx := strings.Index(ab[1], "rc")
		min = ab[1][:idx]
		brc, err = strconv.Atoi(ab[1][idx+2:])
		if err != nil {
			panic(err)
		}
	} else {
		min = ab[1]
	}
	bmin, err := strconv.Atoi(min)
	if err != nil {
		panic(err)
	}
	if amin < bmin {
		return -1
	}
	if amin > bmin {
		return 1
	}

	// At this point major and minor version are matching.
	if len(aa) > len(ab) {
		// a has patch version, b doesn't.
		return 1
	}
	if len(aa) < len(ab) {
		// b has patch version, a doesn't.
		return -1
	}

	// Compare patch versions.
	if len(aa) == 3 && len(ab) == 3 {
		apatch, err := strconv.Atoi(aa[2])
		if err != nil {
			panic(err)
		}
		bpatch, err := strconv.Atoi(ab[2])
		if err != nil {
			panic(err)
		}
		if apatch > bpatch {
			return 1
		}
		return -1
	}

	// Compare beta, rc and x.x.0 version.
	// x.x.0 version should have beta == 0 and rc == 0.
	if abeta < bbeta {
		if abeta != 0 {
			return -1
		}
		return 1
	}
	if abeta > bbeta {
		if bbeta != 0 {
			return 1
		}
		return -1
	}
	if arc < brc {
		if arc != 0 {
			return -1
		}
		return 1
	}
	if brc != 0 {
		return 1
	}
	return -1
}

func findGoCompilerVersion(f *GoFile) (*GoVersion, error) {
	data, err := f.fh.getRData()
	// If read only data section does not exist, try text.
	if err == ErrSectionDoesNotExist {
		data, err = f.fh.getCodeSection()
	}
	if err != nil {
		return nil, err
	}
	notfound := false
	for !notfound {
		version := matchGoVersionString(data)
		if version == "" {
			return nil, ErrNoGoVersionFound
		}
		ver := ResolveGoVersion(version)
		// Go before 1.4 does not have the version string so if we have found
		// a version string below 1.4beta1 it is a false positive.
		if ver == nil || GoVersionCompare(ver.Name, "go1.4beta1") < 0 {
			off := bytes.Index(data, []byte(version))
			// No match
			if off == -1 {
				break
			}
			data = data[off+2:]
			continue
		}
		return ver, nil
	}
	return nil, nil
}

func matchGoVersionString(data []byte) string {
	return string(goVersionMatcher.Find(data))
}
