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
	"bytes"
	"cmp"
	"compress/zlib"
	"debug/dwarf"
	"encoding/binary"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"

	"github.com/blacktop/go-macho"
	"github.com/blacktop/go-macho/types"
)

func openMachO(r io.ReaderAt) (*machoFile, error) {
	f, err := macho.NewFile(r)
	if err != nil {
		return nil, fmt.Errorf("error when parsing the Mach-O file: %w", err)
	}
	ret := &machoFile{file: f, reader: r}
	ret.getsymtab = sync.OnceValue(ret.initSymtab)
	return ret, nil
}

var _ fileHandler = (*machoFile)(nil)

type machoFile struct {
	file      *macho.File
	reader    io.ReaderAt
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

func (m *machoFile) getReader() io.ReaderAt {
	return m.reader
}

func (m *machoFile) Close() error {
	err := m.file.Close()
	if err != nil {
		return err
	}
	return tryClose(m.reader)
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
	var section *types.Section
	for _, sect := range m.file.Sections {
		if sect.Name == s {
			section = sect
			break
		}
	}
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
	switch m.file.CPU {
	case types.CPUI386:
		fi.WordSize = intSize32
		fi.Arch = Arch386
	case types.CPUAmd64:
		fi.WordSize = intSize64
		fi.Arch = ArchAMD64
	case types.CPUArm64:
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

// getDwarf mostly a copy of github.com/blacktop/go-macho.File.DWARF() function
// removes dependency on github.com/blacktop/go-dwarf package
func (m *machoFile) getDwarf() (*dwarf.Data, error) {
	dwarfSuffix := func(s *types.Section) string {
		switch {
		case strings.HasPrefix(s.Name, "__debug_"):
			return s.Name[8:]
		case strings.HasPrefix(s.Name, "__zdebug_"):
			return s.Name[9:]
		default:
			return ""
		}
	}
	sectionData := func(s *types.Section) ([]byte, error) {
		b, err := s.Data()
		if err != nil && uint64(len(b)) < s.Size {
			return nil, err
		}

		if len(b) >= 12 && string(b[:4]) == "ZLIB" {
			dlen := binary.BigEndian.Uint64(b[4:12])
			dbuf := make([]byte, dlen)
			r, err := zlib.NewReader(bytes.NewBuffer(b[12:]))
			if err != nil {
				return nil, err
			}
			if _, err := io.ReadFull(r, dbuf); err != nil {
				return nil, err
			}
			if err := r.Close(); err != nil {
				return nil, err
			}
			b = dbuf
		}
		return b, nil
	}

	// There are many other DWARF sections, but these
	// are the ones the debug/dwarf package uses.
	// Don't bother loading others.
	var dat = map[string][]byte{"abbrev": nil, "info": nil, "str": nil, "line": nil, "ranges": nil}
	for _, s := range m.file.Sections {
		suffix := dwarfSuffix(s)
		if suffix == "" {
			continue
		}
		if _, ok := dat[suffix]; !ok {
			continue
		}
		b, err := sectionData(s)
		if err != nil {
			return nil, err
		}
		dat[suffix] = b
	}

	d, err := dwarf.New(dat["abbrev"], nil, nil, dat["info"], dat["line"], nil, dat["ranges"], dat["str"])
	if err != nil {
		return nil, err
	}

	// Look for DWARF4 .debug_types sections and DWARF5 sections.
	for i, s := range m.file.Sections {
		suffix := dwarfSuffix(s)
		if suffix == "" {
			continue
		}
		if _, ok := dat[suffix]; ok {
			// Already handled.
			continue
		}

		b, err := sectionData(s)
		if err != nil {
			return nil, err
		}

		if suffix == "types" {
			err = d.AddTypes(fmt.Sprintf("types-%d", i), b)
		} else {
			err = d.AddSection(".debug_"+suffix, b)
		}
		if err != nil {
			return nil, err
		}
	}

	return d, nil
}
