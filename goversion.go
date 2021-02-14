// Copyright 2019 The GoRE.tk Authors. All rights reserved.
// Use of this source code is governed by the license that
// can be found in the LICENSE file.

package gore

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/arch/x86/x86asm"
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
	// Try to determine the version based on the schedinit function.
	if v := tryFromSchedInit(f); v != nil {
		return v, nil
	}

	// If no version was found, search the sections for the
	// version string.

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

// tryFromSchedInit tries to identify the version of the Go compiler that compiled the code.
// The function "schedinit" in the "runtime" package has the only reference to this string
// used to identify the version.
// The function returns nil if no version is found.
func tryFromSchedInit(f *GoFile) *GoVersion {
	// Check for non supported architectures.
	if f.FileInfo.Arch != Arch386 && f.FileInfo.Arch != ArchAMD64 {
		return nil
	}

	is32 := false
	if f.FileInfo.Arch == Arch386 {
		is32 = true
	}

	// Find shedinit function.
	var fcn *Function
	std, err := f.GetSTDLib()
	if err != nil {
		return nil
	}

pkgLoop:
	for _, v := range std {
		if v.Name != "runtime" {
			continue
		}
		for _, vv := range v.Functions {
			if vv.Name != "schedinit" {
				continue
			}
			fcn = vv
			break pkgLoop
		}
	}

	// Check if the functions was found
	if fcn == nil {
		// TODO: return an error type.
		return nil
	}

	// Get the raw hex.
	buf, err := f.Bytes(fcn.Offset, fcn.End-fcn.Offset)
	if err != nil {
		return nil
	}

	/*
		Disassemble the function until the loading of the Go version is found.
	*/

	// Counter for how many bytes has been read.
	s := 0
	mode := f.FileInfo.WordSize * 8

	for s < len(buf) {
		inst, err := x86asm.Decode(buf[s:], mode)
		if err != nil {
			return nil
		}

		// Update next instruction location.
		s = s + inst.Len

		// Check if it's a "lea" instruction.
		if inst.Op != x86asm.LEA {
			continue
		}

		// Check what it's loading and if it's pointing to the compiler version used.
		// First assume that the address is a direct addressing.
		arg := inst.Args[1].(x86asm.Mem)
		addr := arg.Disp
		if arg.Base == x86asm.EIP || arg.Base == x86asm.RIP {
			// If the addressing is based on the instruction pointer, fix the address.
			addr = addr + int64(fcn.Offset) + int64(s)
		}

		// If the addressing is based on the stack pointer, this is not the right
		// instruction.
		if arg.Base == x86asm.ESP || arg.Base == x86asm.RSP {
			continue
		}

		// Resolve the pointer to the string. If we get no data, this is not the
		// right instruction.
		b, _ := f.Bytes(uint64(addr), uint64(0x20))
		if b == nil {
			continue
		}

		r := bytes.NewReader(b)
		ptr, err := readUIntTo64(r, f.FileInfo.ByteOrder, is32)
		if err != nil {
			return nil
		}
		l, err := readUIntTo64(r, f.FileInfo.ByteOrder, is32)
		if err != nil {
			return nil
		}

		bstr, _ := f.Bytes(ptr, l)
		if bstr == nil {
			continue
		}

		if !bytes.HasPrefix(bstr, []byte("go1.")) {
			continue
		}

		// Likely the version string.
		ver := string(bstr)

		gover := ResolveGoVersion(ver)
		if gover != nil {
			return gover
		}

		// An unknown version.
		return &GoVersion{Name: ver}
	}

	return nil
}

func matchGoVersionString(data []byte) string {
	return string(goVersionMatcher.Find(data))
}
