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
	"debug/gosym"
	"debug/pe"
	"encoding/binary"
	"fmt"
	"os"
)

func openPE(fp string) (peF *peFile, err error) {
	// Parsing by the file by debug/pe can panic if the PE file is malformed.
	// To prevent a crash, we recover the panic and return it as an error
	// instead.
	go func() {
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
	peF = &peFile{file: f, osFile: osFile}
	return
}

var _ fileHandler = (*peFile)(nil)

type peFile struct {
	file        *pe.File
	osFile      *os.File
	pclntabAddr uint64
	imageBase   uint64
}

func (p *peFile) getSymbolValue(s string) (uint64, error) {
	for _, sym := range p.file.Symbols {
		if sym.Name == s {
			return uint64(sym.Value), nil
		}
	}
	return 0, fmt.Errorf("symbol %s not found", s)
}

func (p *peFile) getParsedFile() any {
	return p.file
}

func (p *peFile) getFile() *os.File {
	return p.osFile
}

func (p *peFile) getPCLNTab(textStart uint64) (*gosym.Table, error) {
	addr, pclndat, err := p.searchForPCLNTab()
	if err != nil {
		return nil, err
	}
	pcln := gosym.NewLineTable(pclndat, textStart)
	p.pclntabAddr = uint64(addr) + p.imageBase
	return gosym.NewTable(make([]byte, 0), pcln)
}

// searchFileForPCLNTab will search the .rdata section for the
// PCLN table.
func (p *peFile) searchForPCLNTab() (uint32, []byte, error) {
	sec := p.file.Section(".rdata")
	if sec == nil {
		return 0, nil, ErrNoPCLNTab
	}
	secData, err := sec.Data()
	if err != nil {
		return 0, nil, err
	}

	tab, err := searchSectionForTab(secData, p.getFileInfo().ByteOrder)
	if err != nil {
		return 0, nil, err
	}

	addr := sec.VirtualAddress + uint32(len(secData)-len(tab))
	return addr, tab, err
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
	b, d, e := p.searchForPCLNTab()
	return p.imageBase + uint64(b), d, e
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
		optHdr := p.file.OptionalHeader.(*pe.OptionalHeader32)
		p.imageBase = uint64(optHdr.ImageBase)
		fi.Arch = Arch386
	} else {
		fi.WordSize = intSize64
		optHdr := p.file.OptionalHeader.(*pe.OptionalHeader64)
		p.imageBase = optHdr.ImageBase
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
