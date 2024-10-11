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
	"debug/pe"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"slices"
	"sync"
)

func openPE(fp string) (peF *peFile, err error) {
	// Parsing by the file by debug/pe can panic if the PE file is malformed.
	// To prevent a crash, we recover the panic and return it as an error
	// instead.
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("error when processing PE file, probably corrupt: %s", r)
		}
	}()

	osFile, err := os.Open(fp)
	if err != nil {
		err = fmt.Errorf("error when opening the file: %w", err)
		return
	}

	f, err := pe.NewFile(osFile)
	if err != nil {
		err = fmt.Errorf("error when parsing the PE file: %w", err)
		return
	}

	imageBase := uint64(0)

	switch hdr := f.OptionalHeader.(type) {
	case *pe.OptionalHeader32:
		imageBase = uint64(hdr.ImageBase)
	case *pe.OptionalHeader64:
		imageBase = hdr.ImageBase
	default:
		err = errors.New("unknown optional header type")
		return
	}

	peF = &peFile{file: f, osFile: osFile, imageBase: imageBase}
	peF.getsymtab = sync.OnceValues(peF.initSymTab)
	return
}

var _ fileHandler = (*peFile)(nil)

type peFile struct {
	file      *pe.File
	osFile    *os.File
	imageBase uint64
	getsymtab func() (map[string]Symbol, error)
}

func (p *peFile) initSymTab() (map[string]Symbol, error) {
	var syms []Symbol
	for _, s := range p.file.Symbols {
		const (
			NUndef = 0  // An undefined (extern) symbol
			NAbs   = -1 // An absolute symbol (e_value is a constant, not an address)
			NDebug = -2 // A debugging symbol
		)
		sym := Symbol{Name: s.Name, Value: uint64(s.Value), Size: 0}
		switch s.SectionNumber {
		case NUndef, NAbs, NDebug: // do nothing
		default:
			if s.SectionNumber < 0 || len(p.file.Sections) < int(s.SectionNumber) {
				return nil, fmt.Errorf("invalid section number in symbol table")
			}
			sect := p.file.Sections[s.SectionNumber-1]
			sym.Value += p.imageBase + uint64(sect.VirtualAddress)
		}
		syms = append(syms, sym)
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

	return symm, nil
}

func (p *peFile) hasSymbolTable() (bool, error) {
	symm, err := p.getsymtab()
	if err != nil {
		return false, err
	}
	return len(symm) > 0, nil
}

func (p *peFile) getSymbol(name string) (uint64, uint64, error) {
	symm, err := p.getsymtab()
	if err != nil {
		return 0, 0, err
	}
	sym, ok := symm[name]
	if !ok {
		return 0, 0, ErrSymbolNotFound
	}
	return sym.Value, sym.Size, nil
}

func (p *peFile) getParsedFile() any {
	return p.file
}

func (p *peFile) getFile() *os.File {
	return p.osFile
}

func (p *peFile) Close() error {
	err := p.file.Close()
	if err != nil {
		return err
	}
	return p.osFile.Close()
}

func (p *peFile) getRData() ([]byte, error) {
	section := p.file.Section(".rdata")
	if section == nil {
		return nil, ErrSectionDoesNotExist
	}
	return section.Data()
}

func (p *peFile) getCodeSection() (uint64, []byte, error) {
	section := p.file.Section(".text")
	if section == nil {
		return 0, nil, ErrSectionDoesNotExist
	}
	data, err := section.Data()
	return p.imageBase + uint64(section.VirtualAddress), data, err
}

func (p *peFile) moduledataSection() string {
	return ".data"
}

func (p *peFile) getPCLNTABData() (uint64, []byte, error) {
	for _, v := range []string{".rdata", ".text"} {
		sec := p.file.Section(v)
		if sec == nil {
			continue
		}
		secData, err := sec.Data()
		if err != nil {
			continue
		}
		tab, err := searchSectionForTab(secData, p.getFileInfo().ByteOrder)
		if errors.Is(ErrNoPCLNTab, err) {
			continue
		}

		addr := uint64(sec.VirtualAddress) + uint64(len(secData)-len(tab))
		return p.imageBase + addr, tab, err
	}
	return 0, []byte{}, ErrNoPCLNTab
}

func (p *peFile) getSectionDataFromAddress(address uint64) (uint64, []byte, error) {
	for _, section := range p.file.Sections {
		if section.Offset == 0 {
			// Only exist in memory
			continue
		}

		if p.imageBase+uint64(section.VirtualAddress) <= address && address < p.imageBase+uint64(section.VirtualAddress+section.Size) {
			data, err := section.Data()
			return p.imageBase + uint64(section.VirtualAddress), data, err
		}
	}
	return 0, nil, ErrSectionDoesNotExist
}

func (p *peFile) getSectionData(name string) (uint64, []byte, error) {
	section := p.file.Section(name)
	if section == nil {
		return 0, nil, ErrSectionDoesNotExist
	}
	data, err := section.Data()
	return p.imageBase + uint64(section.VirtualAddress), data, err
}

func (p *peFile) getFileInfo() *FileInfo {
	fi := &FileInfo{ByteOrder: binary.LittleEndian, OS: "windows"}
	if p.file.Machine == pe.IMAGE_FILE_MACHINE_I386 {
		fi.WordSize = intSize32
		fi.Arch = Arch386
	} else {
		fi.WordSize = intSize64
		fi.Arch = ArchAMD64
	}
	return fi
}

func (p *peFile) getBuildID() (string, error) {
	_, data, err := p.getCodeSection()
	if err != nil {
		return "", fmt.Errorf("failed to get code section: %w", err)
	}
	return parseBuildIDFromRaw(data)
}

func (p *peFile) getDwarf() (*dwarf.Data, error) {
	return p.file.DWARF()
}
