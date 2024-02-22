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
	"debug/elf"
	"errors"
	"fmt"
	"os"
)

func openELF(fp string) (*elfFile, error) {
	osFile, err := os.Open(fp)
	if err != nil {
		return nil, fmt.Errorf("error when opening the ELF file: %w", err)
	}

	f, err := elf.NewFile(osFile)
	if err != nil {
		return nil, fmt.Errorf("error when parsing the ELF file: %w", err)
	}
	return &elfFile{
		file:   f,
		osFile: osFile,
		pcln:   newPclnTabOnce(),
		symtab: newSymbolTableOnce(),
	}, nil
}

var _ fileHandler = (*elfFile)(nil)

type elfFile struct {
	file   *elf.File
	osFile *os.File
	pcln   *pclntabOnce
	symtab *symbolTableOnce
}

func (e *elfFile) initSymTab() error {
	e.symtab.Do(func() {
		syms, err := e.file.Symbols()
		if err != nil {
			// If the error is ErrNoSymbols, we just ignore it.
			if !errors.Is(err, elf.ErrNoSymbols) {
				e.symtab.err = fmt.Errorf("error when getting the symbols: %w", err)
			}
			return
		}
		for _, sym := range syms {
			e.symtab.table[sym.Name] = symbol{
				Name:  sym.Name,
				Value: sym.Value,
				Size:  sym.Size,
			}
		}
	})
	return e.symtab.err
}

func (e *elfFile) hasSymbolTable() (bool, error) {
	err := e.initSymTab()
	if err != nil {
		return false, err
	}
	return len(e.symtab.table) > 0, nil
}

func (e *elfFile) getSymbol(name string) (uint64, uint64, error) {
	err := e.initSymTab()
	if err != nil {
		return 0, 0, err
	}
	sym, ok := e.symtab.table[name]
	if !ok {
		return 0, 0, ErrSymbolNotFound
	}
	return sym.Value, sym.Size, nil
}

func (e *elfFile) getParsedFile() any {
	return e.file
}

func (e *elfFile) getFile() *os.File {
	return e.osFile
}

func (e *elfFile) searchForPclnTab() (uint64, []byte, error) {
	// Only use linkmode=external and buildmode=pie will lead to this
	// Since the external linker merged all .data.rel.ro.* sections into .data.rel.ro
	// So we have to find .data.rel.ro.gopclntab manually
	sec := e.file.Section(".data.rel.ro")
	if sec == nil {
		return 0, nil, ErrNoPCLNTab
	}
	data, err := sec.Data()
	if err != nil {
		return 0, nil, fmt.Errorf("could not get the data for the pclntab: %w", err)
	}

	tab, err := searchSectionForTab(data, e.getFileInfo().ByteOrder)
	if err != nil {
		return 0, nil, fmt.Errorf("could not find the pclntab: %w", err)
	}
	addr := sec.Addr + sec.FileSize - uint64(len(tab))
	return addr, tab, nil
}

func (e *elfFile) symbolData(start, end string) (uint64, uint64, []byte) {
	elfSyms, err := e.file.Symbols()
	if err != nil {
		return 0, 0, nil
	}
	var addr, eaddr uint64
	for _, s := range elfSyms {
		if s.Name == start {
			addr = s.Value
		} else if s.Name == end {
			eaddr = s.Value
		}
		if addr != 0 && eaddr != 0 {
			break
		}
	}
	if addr == 0 || eaddr < addr {
		return 0, 0, nil
	}
	size := eaddr - addr
	data := make([]byte, size)
	for _, prog := range e.file.Progs {
		if prog.Vaddr <= addr && addr+size-1 <= prog.Vaddr+prog.Filesz-1 {
			if _, err := prog.ReadAt(data, int64(addr-prog.Vaddr)); err != nil {
				return 0, 0, nil
			}
			return addr, eaddr, data
		}
	}
	return 0, 0, nil
}

func (e *elfFile) Close() error {
	err := e.file.Close()
	if err != nil {
		return err
	}
	return e.osFile.Close()
}

func (e *elfFile) getRData() ([]byte, error) {
	section := e.file.Section(".rodata")
	if section == nil {
		return nil, ErrSectionDoesNotExist
	}
	return section.Data()
}

func (e *elfFile) getCodeSection() (uint64, []byte, error) {
	section := e.file.Section(".text")
	if section == nil {
		return 0, nil, ErrSectionDoesNotExist
	}
	data, err := section.Data()
	if err != nil {
		return 0, nil, fmt.Errorf("error when getting the code section: %w", err)
	}
	return section.Addr, data, nil
}

func (e *elfFile) getPCLNTABData() (start uint64, data []byte, err error) {
	return e.pcln.load(e.getPCLNTABDataImpl)
}

func (e *elfFile) getPCLNTABDataImpl() (start uint64, data []byte, err error) {
	pclnSection := e.file.Section(".gopclntab")
	if pclnSection == nil {
		// No section found. Check if the PIE section exists instead.
		pclnSection = e.file.Section(".data.rel.ro.gopclntab")
	}
	if pclnSection != nil {
		data, err = pclnSection.Data()
		if err != nil {
			return 0, nil, fmt.Errorf("could not get the data for the pclntab: %w", err)
		}
		return pclnSection.Addr, data, nil
	}

	// try to get data from symbol
	start, _, data = e.symbolData("runtime.pclntab", "runtime.epclntab")
	if data != nil {
		return start, data, nil
	}

	// try brute force searching for the pclntab
	start, data, err = e.searchForPclnTab()
	if err != nil {
		return 0, nil, fmt.Errorf("could not find the pclntab: %w", err)
	}
	return
}

func (e *elfFile) moduledataSection() string {
	return ".noptrdata"
}

func (e *elfFile) getSectionDataFromAddress(address uint64) (uint64, []byte, error) {
	for _, section := range e.file.Sections {
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

func (e *elfFile) getSectionData(name string) (uint64, []byte, error) {
	section := e.file.Section(name)
	if section == nil {
		return 0, nil, ErrSectionDoesNotExist
	}
	data, err := section.Data()
	return section.Addr, data, err
}

func (e *elfFile) getFileInfo() *FileInfo {
	var wordSize int
	class := e.file.FileHeader.Class
	if class == elf.ELFCLASS32 {
		wordSize = intSize32
	}
	if class == elf.ELFCLASS64 {
		wordSize = intSize64
	}

	var arch string
	switch e.file.Machine {
	case elf.EM_386:
		arch = Arch386
	case elf.EM_MIPS:
		arch = ArchMIPS
	case elf.EM_X86_64:
		arch = ArchAMD64
	case elf.EM_ARM:
		arch = ArchARM
	}

	return &FileInfo{
		ByteOrder: e.file.FileHeader.ByteOrder,
		OS:        e.file.Machine.String(),
		WordSize:  wordSize,
		Arch:      arch,
	}
}

func (e *elfFile) getBuildID() (string, error) {
	_, data, err := e.getSectionData(".note.go.buildid")
	// If the note section does not exist, we just ignore the build id.
	if errors.Is(err, ErrSectionDoesNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("error when getting note section: %w", err)
	}
	return parseBuildIDFromElf(data, e.file.ByteOrder)
}

func (e *elfFile) getDwarf() (*dwarf.Data, error) {
	return e.file.DWARF()
}
