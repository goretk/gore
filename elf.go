// Copyright 2019 The GoRE.tk Authors. All rights reserved.
// Use of this source code is governed by the license that
// can be found in the LICENSE file.

package gore

import (
	"debug/elf"
	"debug/gosym"
	"fmt"
)

func openELF(fp string) (*elfFile, error) {
	f, err := elf.Open(fp)
	if err != nil {
		return nil, err
	}
	return &elfFile{file: f}, nil
}

type elfFile struct {
	file *elf.File
}

func (e *elfFile) getPCLNTab() (*gosym.Table, error) {
	pclndat, err := e.file.Section(".gopclntab").Data()
	if err != nil {
		return nil, err
	}
	pcln := gosym.NewLineTable(pclndat, e.file.Section(".text").Addr)
	return gosym.NewTable(make([]byte, 0), pcln)
}

func (e *elfFile) Close() error {
	return e.file.Close()
}

func (e *elfFile) getRData() ([]byte, error) {
	section := e.file.Section(".rodata")
	if section == nil {
		return nil, ErrSectionDoesNotExist
	}
	return section.Data()
}

func (e *elfFile) getCodeSection() ([]byte, error) {
	section := e.file.Section(".text")
	if section == nil {
		return nil, ErrSectionDoesNotExist
	}
	return section.Data()
}

func (e *elfFile) getPCLNTABData() (uint64, []byte, error) {
	return e.getSectionData(".gopclntab")
}

func (e *elfFile) moduledataSection() string {
	return ".noptrdata"
}

func (e *elfFile) getSectionDataFromOffset(off uint64) (uint64, []byte, error) {
	for _, section := range e.file.Sections {
		if section.Addr <= off && off < (section.Addr+section.Size) {
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
	return &FileInfo{
		ByteOrder: e.file.FileHeader.ByteOrder,
		OS:        e.file.Machine.String(),
		WordSize:  wordSize,
	}
}

func (e *elfFile) getBuildID() (string, error) {
	_, data, err := e.getSectionData(".note.go.buildid")
	if err != nil {
		return "", fmt.Errorf("error when getting note section %w", err)
	}
	return parseBuildIDFromElf(data, e.file.ByteOrder)
}
