// This file is part of GoRE.
//
// Copyright (C) 2019-2024 GoRE Authors
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
	"cmp"
	"debug/dwarf"
	"debug/macho"
	"fmt"
	"os"
	"slices"
	"sync"
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
	ret := &machoFile{file: f, osFile: osFile}
	ret.getsymtab = sync.OnceValue(ret.initSymtab)
	return ret, nil
}

var _ fileHandler = (*machoFile)(nil)

type machoFile struct {
	file      *macho.File
	osFile    *os.File
	getsymtab func() map[string]Symbol
}

func (m *machoFile) initSymtab() map[string]Symbol {
	if m.file.Symtab == nil {
		// just do nothing, keep err nil and table empty
		return nil
	}

	const stabTypeMask = 0xe0
	// Build a sorted list of all symbols.
	// We infer the size of a symbol by looking at where the next symbol begins.
	syms := make([]Symbol, 0)
	for _, s := range m.file.Symtab.Syms {
		if s.Type&stabTypeMask != 0 {
			// Skip stab debug info.
			continue
		}
		syms = append(syms, Symbol{Name: s.Name, Value: s.Value})
	}

	slices.SortStableFunc(syms, func(a, b Symbol) int {
		return cmp.Compare(a.Value, b.Value)
	})

	for i := 0; i < len(syms)-1; i++ {
		syms[i].Size = syms[i+1].Value - syms[i].Value
	}

	symm := make(map[string]Symbol)
	for _, sym := range syms {
		symm[sym.Name] = sym
	}

	return symm
}

func (m *machoFile) getSymbol(name string) (Symbol, error) {
	sym, ok := m.getsymtab()[name]
	if !ok {
		return Symbol{}, ErrSymbolNotFound
	}
	return sym, nil
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
