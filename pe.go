// Copyright 2019 The GoRE.tk Authors. All rights reserved.
// Use of this source code is governed by the license that
// can be found in the LICENSE file.

package gore

import (
	"debug/gosym"
	"debug/pe"
	"encoding/binary"
)

func openPE(fp string) (*peFile, error) {
	f, err := pe.Open(fp)
	if err != nil {
		return nil, err
	}
	return &peFile{file: f}, nil
}

type peFile struct {
	file        *pe.File
	pclntabAddr uint64
	imageBase   uint64
}

func (p *peFile) getPCLNTab() (*gosym.Table, error) {
	addr, pclndat, err := searchFileForPCLNTab(p.file)
	if err != nil {
		return nil, err
	}
	pcln := gosym.NewLineTable(pclndat, uint64(p.file.Section(".text").VirtualAddress))
	p.pclntabAddr = uint64(addr) + p.imageBase
	return gosym.NewTable(make([]byte, 0), pcln)
}

func (p *peFile) Close() error {
	return p.file.Close()
}

func (p *peFile) getRData() ([]byte, error) {
	section := p.file.Section(".rdata")
	if section == nil {
		return nil, ErrSectionDoesNotExist
	}
	return section.Data()
}

func (p *peFile) getCodeSection() ([]byte, error) {
	section := p.file.Section(".text")
	if section == nil {
		return nil, ErrSectionDoesNotExist
	}
	return section.Data()
}

func (p *peFile) moduledataSection() string {
	return ".data"
}

func (p *peFile) getPCLNTABData() (uint64, []byte, error) {
	b, d, e := searchFileForPCLNTab(p.file)
	return p.imageBase + uint64(b), d, e
}

func (p *peFile) getSectionDataFromOffset(off uint64) (uint64, []byte, error) {
	for _, section := range p.file.Sections {
		if p.imageBase+uint64(section.VirtualAddress) <= off && off < p.imageBase+uint64(section.VirtualAddress+section.Size) {
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
	} else {
		fi.WordSize = intSize64
		optHdr := p.file.OptionalHeader.(*pe.OptionalHeader64)
		p.imageBase = optHdr.ImageBase
	}
	return fi
}
