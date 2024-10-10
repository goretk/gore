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

//go:generate go run ./gen moduledata

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/goretk/gore/extern"
	"github.com/goretk/gore/extern/gover"
)

// Moduledata holds information about the layout of the executable image in memory.
type Moduledata interface {
	// Text returns the text secion.
	Text() ModuleDataSection
	// NoPtrData returns the noptrdata section.
	NoPtrData() ModuleDataSection
	// Data returns the data section.
	Data() ModuleDataSection
	// Bss returns the bss section.
	Bss() ModuleDataSection
	// NoPtrBss returns the noptrbss section.
	NoPtrBss() ModuleDataSection
	// Types returns the types section.
	Types() ModuleDataSection
	// PCLNTab returns the pclntab section.
	PCLNTab() ModuleDataSection
	// FuncTab returns the functab section.
	FuncTab() ModuleDataSection
	// ITabLinks returns the itablinks section.
	ITabLinks() ModuleDataSection
	// TypeLink returns the typelink section.
	TypeLink() ModuleDataSection
	// TypeLinkData returns the typelink section data.
	TypeLinkData() ([]int32, error)
	// GoFuncValue returns the value of the 'go:func.*' symbol.
	GoFuncValue() uint64
}

type moduledata struct {
	TextAddr, TextLen           uint64
	NoPtrDataAddr, NoPtrDataLen uint64
	DataAddr, DataLen           uint64
	BssAddr, BssLen             uint64
	NoPtrBssAddr, NoPtrBssLen   uint64

	TypesAddr, TypesLen       uint64
	TypelinkAddr, TypelinkLen uint64
	ITabLinkAddr, ITabLinkLen uint64
	FuncTabAddr, FuncTabLen   uint64
	PCLNTabAddr, PCLNTabLen   uint64

	GoFuncVal uint64

	fh fileHandler
}

// Text returns the text section.
func (m moduledata) Text() ModuleDataSection {
	return ModuleDataSection{
		Address: m.TextAddr,
		Length:  m.TextLen,
		fh:      m.fh,
	}
}

// NoPtrData returns the noptrdata section.
func (m moduledata) NoPtrData() ModuleDataSection {
	return ModuleDataSection{
		Address: m.NoPtrDataAddr,
		Length:  m.NoPtrDataLen,
		fh:      m.fh,
	}
}

// Data returns the data section.
func (m moduledata) Data() ModuleDataSection {
	return ModuleDataSection{
		Address: m.DataAddr,
		Length:  m.DataLen,
		fh:      m.fh,
	}
}

// Bss returns the bss section.
func (m moduledata) Bss() ModuleDataSection {
	return ModuleDataSection{
		Address: m.BssAddr,
		Length:  m.BssLen,
		fh:      m.fh,
	}
}

// NoPtrBss returns the noptrbss section.
func (m moduledata) NoPtrBss() ModuleDataSection {
	return ModuleDataSection{
		Address: m.NoPtrBssAddr,
		Length:  m.NoPtrBssLen,
		fh:      m.fh,
	}
}

// Types returns the types section.
func (m moduledata) Types() ModuleDataSection {
	return ModuleDataSection{
		Address: m.TypesAddr,
		Length:  m.TypesLen,
		fh:      m.fh,
	}
}

// PCLNTab returns the pclntab section.
func (m moduledata) PCLNTab() ModuleDataSection {
	return ModuleDataSection{
		Address: m.PCLNTabAddr,
		Length:  m.PCLNTabLen,
		fh:      m.fh,
	}
}

// FuncTab returns the functab section.
func (m moduledata) FuncTab() ModuleDataSection {
	return ModuleDataSection{
		Address: m.FuncTabAddr,
		Length:  m.FuncTabLen,
		fh:      m.fh,
	}
}

// ITabLinks returns the itablinks section.
func (m moduledata) ITabLinks() ModuleDataSection {
	return ModuleDataSection{
		Address: m.ITabLinkAddr,
		Length:  m.ITabLinkLen,
		fh:      m.fh,
	}
}

// TypeLink returns the typelink section.
func (m moduledata) TypeLink() ModuleDataSection {
	return ModuleDataSection{
		Address: m.TypelinkAddr,
		Length:  m.TypelinkLen,
		fh:      m.fh,
	}
}

// TypeLinkData returns the typelink section.
func (m moduledata) TypeLinkData() ([]int32, error) {
	base, data, err := m.fh.getSectionDataFromAddress(m.TypelinkAddr)

	if err != nil {
		return nil, fmt.Errorf("failed to get the typelink data section: %w", err)
	}

	r := bytes.NewReader(data[m.TypelinkAddr-base:])
	a := make([]int32, 0, m.TypelinkLen)
	bo := m.fh.getFileInfo().ByteOrder
	for i := uint64(0); i < m.TypelinkLen; i++ {
		// Type offsets are always int32
		var off int32
		err = binary.Read(r, bo, &off)
		if err != nil {
			return nil, fmt.Errorf("failed to read typelink item %d: %w", i, err)
		}
		a = append(a, off)
	}

	return a, nil
}

// GoFuncValue returns the value of the "go:func.*" symbol.
func (m moduledata) GoFuncValue() uint64 {
	return m.GoFuncVal
}

// ModuleDataSection is a section defined in the Moduledata structure.
type ModuleDataSection struct {
	// Address is the virtual address where the section starts.
	Address uint64
	// Length is the byte length for the data in this section.
	Length uint64
	fh     fileHandler
}

// Data returns the data in the section.
func (m ModuleDataSection) Data() ([]byte, error) {
	// If we don't have any data, return an empty slice.
	if m.Length == 0 {
		return []byte{}, nil
	}
	base, data, err := m.fh.getSectionDataFromAddress(m.Address)
	if err != nil {
		return nil, fmt.Errorf("getting module data section failed: %w", err)
	}
	start := m.Address - base
	if uint64(len(data)) < start+m.Length {
		return nil, fmt.Errorf("the length of module data section is to big: address 0x%x, base 0x%x, length 0x%x", m.Address, base, m.Length)
	}
	buf := make([]byte, m.Length)
	copy(buf, data[start:start+m.Length])
	return buf, nil
}

func buildPclnTabAddrBinary(wordSize int, order binary.ByteOrder, addr uint64) []byte {
	buf := make([]byte, wordSize)
	if wordSize == intSize32 {
		order.PutUint32(buf, uint32(addr))
	} else {
		order.PutUint64(buf, addr)
	}
	return buf
}

func pickVersionedModuleData(info *FileInfo) (modulable, error) {
	var bits int
	if info.WordSize == intSize32 {
		bits = 32
	} else {
		bits = 64
	}

	if info.goversion == nil {
		return nil, ErrNoGoVersionFound
	}

	ver := gover.Parse(extern.StripGo(info.goversion.Name))
	zero := gover.Version{}
	if ver == zero {
		return nil, errors.New("could not parse the go version " + info.goversion.Name)
	}

	verBit, err := strconv.Atoi(ver.Minor)
	if err != nil {
		return nil, err
	}
	buf, err := selectModuleData(verBit, bits)
	if err != nil {
		return nil, fmt.Errorf("error when selecting the module data: %w", err)
	}

	return buf, nil
}

func extractModuledata(f *GoFile) (moduledata, error) {
	vmd, err := pickVersionedModuleData(f.FileInfo)
	if err != nil {
		return moduledata{}, err
	}

	vmdSize := binary.Size(vmd)

	// pre define these variables to follow the goto requirements
	var off int
	var magic []byte
	var tabAddr uint64

	secAddr, secData, err := f.fh.getSectionData(f.fh.moduledataSection())
	if err != nil {
		return moduledata{}, err
	}

	// if we can get the moduledata addr from the symbol, we have no need to search
	if ok, err := f.fh.hasSymbolTable(); ok && err == nil {
		addr, _, err := f.fh.getSymbol("runtime.firstmoduledata")
		if err == nil {
			off = int(addr - secAddr)
			goto load
		}
	}

	err = f.initPclntab()
	if err != nil {
		return moduledata{}, err
	}
	tabAddr = f.pclntabAddr

	magic = buildPclnTabAddrBinary(f.FileInfo.WordSize, f.FileInfo.ByteOrder, tabAddr)

search:
	off = bytes.Index(secData, magic)
load:
	if off == -1 {
		return moduledata{}, errors.New("could not find moduledata")
	}
	if len(secData) < off+vmdSize {
		return moduledata{}, fmt.Errorf("offset %d is out of bounds %d", off, len(secData))
	}

	data := secData[off : off+vmdSize]

	// Read the module struct from the file.
	r := bytes.NewReader(data)
	err = binary.Read(r, f.FileInfo.ByteOrder, vmd)
	if err != nil {
		return moduledata{}, fmt.Errorf("error when reading module data from file: %w", err)
	}

	// Convert the read struct to the type we return to the caller.
	md := vmd.toModuledata()

	// Take a simple validation step to ensure that the moduledata is valid.
	text := md.TextAddr
	etext := md.TextAddr + md.TextLen

	textSectAddr, textSect, err := f.fh.getCodeSection()
	if err != nil {
		return moduledata{}, err
	}
	if text > etext {
		goto invalidMD
	}

	if !(textSectAddr <= text && text < textSectAddr+uint64(len(textSect))) {
		goto invalidMD
	}

	// Add the file handler.
	md.fh = f.fh

	return md, nil

invalidMD:
	secData = secData[off+1:]
	goto search
}

func readUIntTo64(r io.Reader, byteOrder binary.ByteOrder, is32bit bool) (addr uint64, err error) {
	if is32bit {
		var addr32 uint32
		err = binary.Read(r, byteOrder, &addr32)
		addr = uint64(addr32)
	} else {
		err = binary.Read(r, byteOrder, &addr)
	}
	return
}

type modulable interface {
	toModuledata() moduledata
}
