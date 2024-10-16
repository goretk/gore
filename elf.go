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
	"debug/dwarf"
	"debug/elf"
	"errors"
	"fmt"
	"io"
	"sync"
)

func openELF(r io.ReaderAt) (*elfFile, error) {
	f, err := elf.NewFile(r)
	if err != nil {
		return nil, fmt.Errorf("error when parsing the ELF file: %w", err)
	}
	ret := &elfFile{file: f, reader: r}
	ret.getsymtab = sync.OnceValues(ret.initSymTab)
	return ret, nil
}

var _ fileHandler = (*elfFile)(nil)

type elfFile struct {
	file      *elf.File
	reader    io.ReaderAt
	getsymtab func() (map[string]Symbol, error)
}

func (e *elfFile) initSymTab() (map[string]Symbol, error) {
	syms, err := e.file.Symbols()
	if err != nil {
		// If the error is ErrNoSymbols, we just ignore it.
		if !errors.Is(err, elf.ErrNoSymbols) {
			return nil, fmt.Errorf("error when getting the symbols: %w", err)
		}
		return nil, ErrSymbolNotFound
	}
	symm := make(map[string]Symbol)
	for _, sym := range syms {
		symm[sym.Name] = Symbol{
			Name:  sym.Name,
			Value: sym.Value,
			Size:  sym.Size,
		}
	}
	return symm, nil
}

func (e *elfFile) getSymbol(name string) (Symbol, error) {
	symm, err := e.getsymtab()
	if err != nil {
		return Symbol{}, err
	}
	sym, ok := symm[name]
	if !ok {
		return Symbol{}, ErrSymbolNotFound
	}
	return sym, nil
}

func (e *elfFile) getParsedFile() any {
	return e.file
}

func (e *elfFile) getReader() io.ReaderAt {
	return e.reader
}

func (e *elfFile) Close() error {
	err := e.file.Close()
	if err != nil {
		return err
	}
	return tryClose(e.reader)
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

func (e *elfFile) getPCLNTABData() (uint64, []byte, error) {
	// If the standard linker was used when linking the Go binary, the pclntab is located
	// in its own section in the ELF. We first check the section used when using the default
	// build mode. If the section doesn't exist, we check the section used when the PIE
	// build mode is used.
	for _, s := range []string{".gopclntab", ".data.rel.ro.gopclntab"} {
		start, data, err := e.getSectionData(s)
		if errors.Is(err, ErrSectionDoesNotExist) {
			continue
		}
		if err != nil {
			return 0, nil, fmt.Errorf("accessing section data for %s failed: %w", s, err)
		}
		// We found the pclntab section so we can return it to the caller.
		return start, data, nil
	}

	// For files that have been linked with an external linker, the table is located
	// in the .data.rel.ro section. Because it's not in its own section, we will have to
	// search for it in the section.
	start, data, err := e.getSectionData(".data.rel.ro")
	if err != nil {
		return 0, nil, fmt.Errorf("failed to get section: .data.rel.ro: %w", err)
	}

	buf, err := searchSectionForTab(data, e.file.FileHeader.ByteOrder)
	if err != nil {
		return 0, nil, fmt.Errorf("error when search for pclntab: %w", err)
	}

	// Calculate the virtual address of the PCLNTAB. We don't know the size of table so
	// we search from the end of section until we find the start of the table. Doing it
	// this way, we can use the difference between the size of the segment and the size
	// of the "tail" to get the offset where the table starts.
	vaddr := start + uint64(len(data)) - uint64(len(buf))

	return vaddr, buf, err
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
