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
	"debug/dwarf"
	"debug/macho"
	"errors"
	"fmt"
	"math"
	"os"
	"slices"
)

func openMachO(fp string) (*machoFile, error) {
	osFile, err := os.Open(fp)
	if err != nil {
		return nil, fmt.Errorf("error when opening the file: %w", err)
	}

	f, err := macho.NewFile(osFile)
	if err != nil {
		return nil, fmt.Errorf("error when parsing the Mach-O file: %w", err)
	}
	return &machoFile{file: f, osFile: osFile}, nil
}

var _ fileHandler = (*machoFile)(nil)

type machoFile struct {
	file   *macho.File
	osFile *os.File
}

func (m *machoFile) getSymbol(name string) (uint64, uint64, error) {
	var addrs []uint64

	foundedAddr := uint64(math.MaxUint64)

	const stabTypeMask = 0xe0

	for _, s := range m.file.Symtab.Syms {
		if s.Type&stabTypeMask != 0 || s.Sect == 0 {
			continue
		}

		addrs = append(addrs, s.Value)

		if s.Name == name {
			foundedAddr = s.Value
		}
	}

	if foundedAddr == math.MaxUint64 {
		return 0, 0, fmt.Errorf("symbol %s not found", name)
	}

	slices.Sort(addrs)

	index, _ := slices.BinarySearch(addrs, foundedAddr)

	if index == len(addrs)-1 {
		return foundedAddr, 0, errors.New("size not available")
	}
	return foundedAddr, addrs[index+1] - foundedAddr, nil
}

func (m *machoFile) getParsedFile() any {
	return m.file
}

func (m *machoFile) getFile() *os.File {
	return m.osFile
}

func (m *machoFile) Close() error {
	err := m.file.Close()
	if err != nil {
		return err
	}
	return m.osFile.Close()
}

func (m *machoFile) getRData() ([]byte, error) {
	_, data, err := m.getSectionData("__rodata")
	return data, err
}

func (m *machoFile) getCodeSection() (uint64, []byte, error) {
	return m.getSectionData("__text")
}

func (m *machoFile) getSectionDataFromAddress(address uint64) (uint64, []byte, error) {
	for _, section := range m.file.Sections {
		if section.Offset == 0 {
			// Only exist in memory
			continue
		}

		if section.Addr <= address && address < (section.Addr+section.Size) {
			data, err := section.Data()
			return section.Addr, data, err
		}
	}
	return 0, nil, ErrSectionDoesNotExist
}

func (m *machoFile) getSectionData(s string) (uint64, []byte, error) {
	section := m.file.Section(s)
	if section == nil {
		return 0, nil, ErrSectionDoesNotExist
	}
	data, err := section.Data()
	return section.Addr, data, err
}

func (m *machoFile) getFileInfo() *FileInfo {
	fi := &FileInfo{
		ByteOrder: m.file.ByteOrder,
		OS:        "macOS",
	}
	switch m.file.Cpu {
	case macho.Cpu386:
		fi.WordSize = intSize32
		fi.Arch = Arch386
	case macho.CpuAmd64:
		fi.WordSize = intSize64
		fi.Arch = ArchAMD64
	case macho.CpuArm64:
		fi.WordSize = intSize64
		fi.Arch = ArchARM64
	default:
		panic("Unsupported architecture")
	}
	return fi
}

func (m *machoFile) getPCLNTABData() (uint64, []byte, error) {
	return m.getSectionData("__gopclntab")
}

func (m *machoFile) moduledataSection() string {
	return "__noptrdata"
}

func (m *machoFile) getBuildID() (string, error) {
	_, data, err := m.getCodeSection()
	if err != nil {
		return "", fmt.Errorf("failed to get code section: %w", err)
	}
	return parseBuildIDFromRaw(data)
}

func (m *machoFile) getDwarf() (*dwarf.Data, error) {
	return m.file.DWARF()
}
