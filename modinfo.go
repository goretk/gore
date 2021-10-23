// This file is part of GoRE.
//
// Copyright (C) 2019-2021 GoRE Authors
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package gore

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/goretk/gore/extern"
)

var (
	// ErrNoBuildInfo is returned if the file has no build information available.
	ErrNoBuildInfo    = errors.New("no build info available")
	buildInfoMagic    = []byte{0xff, 0x20, 0x47, 0x6f, 0x20, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x69, 0x6e, 0x66, 0x3a}
	buildInfoSections = []string{".go.buildinfo", "__go_buildinfo", ".data"} // The order is important.
)

const (
	buildInfoSectionSize = 0x20 // This is enough for the data we need.
)

// BuildInfo that was extracted from the file.
type BuildInfo struct {
	// Compiler version. Can be nil.
	Compiler *GoVersion
	// ModInfo holds information about the Go modules in this file.
	// Can be nil.
	ModInfo *extern.BuildInfo
}

func (f *GoFile) extractBuildInfo() (*BuildInfo, error) {
	order := f.FileInfo.ByteOrder
	is32 := f.FileInfo.WordSize != 8

	var sectionData []byte
	// Find the section
	for _, v := range buildInfoSections {
		_, d, err := f.fh.getSectionData(v)
		if err != nil {
			if err == ErrSectionDoesNotExist {
				continue
			}
			return nil, fmt.Errorf("failed to get buildinfo section: %w", err)
		}

		// Check for the magic.
		i := bytes.Index(d, buildInfoMagic)
		if i == -1 {
			// Not the right section or doesn't exist, try next.
			continue
		}

		// Take a subslice of the section.
		sectionData = d[i : i+buildInfoSectionSize]

		break // We are done.
	}

	// Check if we found the build info.
	if sectionData == nil {
		return nil, ErrNoBuildInfo
	}

	buf := bytes.NewReader(sectionData)

	// Skip over the marker.
	_, err := buf.Seek(0x10, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("seek failed when skipping build info marker: %w", err)
	}

	// After the markers there are two string pointers. The first one points to the
	// compiler version. The second points to the module information.
	ptr1, err := readUIntTo64(buf, order, is32)
	if err != nil {
		return nil, fmt.Errorf("reading pointer to compiler version in buildinfo failed: %w", err)
	}

	ptr2, err := readUIntTo64(buf, order, is32)
	if err != nil {
		return nil, fmt.Errorf("reading pointer to module info in buildinfo failed: %w", err)
	}

	gv, err := extractStringFromPtr(f, ptr1)
	if err != nil {
		return nil, fmt.Errorf("extracting compiler version failed: %w", err)
	}

	modinfoData, err := extractStringFromPtr(f, ptr2)
	if err != nil {
		return nil, fmt.Errorf("extracting modinfo data failed: %w", err)
	}

	// Populate the result.

	bi := &BuildInfo{
		Compiler: ResolveGoVersion(gv),
	}

	if mi, ok := extern.ParseBuildInfo(modinfoData); ok {
		bi.ModInfo = mi
	}

	return bi, nil
}

func extractStringFromPtr(f *GoFile, offset uint64) (string, error) {
	// Check if the offset points
	order := f.FileInfo.ByteOrder
	is32 := f.FileInfo.WordSize != 8

	// Get enough bytes to handle 64-bit.
	buf, err := f.Bytes(offset, 0x20)
	if err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, ErrSectionDoesNotExist) {
			// The pointer probably points to a section that has 0 space no disk.
			// In this case, return an empty string.
			return "", nil
		}
		return "", err
	}

	r := bytes.NewReader(buf)

	// First ptr is to the data and the second is the length of the string.
	d, err := readUIntTo64(r, order, is32)
	if err != nil {
		return "", fmt.Errorf("error when reading the string's data offset: %w", err)
	}

	if d == uint64(0) || int64(d) == int64(-1) {
		return "", nil
	}

	l, err := readUIntTo64(r, order, is32)
	if err != nil {
		return "", fmt.Errorf("error when reading the string's length: %w", err)
	}

	// Get the string bytes.
	strBytes, err := f.Bytes(d, l)
	if err != nil {
		return "", fmt.Errorf("error when reading string bytes: %w", err)
	}

	return string(strBytes), err
}
