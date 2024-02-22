// This file is part of GoRE.
//
// Copyright (C) 2019-2023 GoRE Authors
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

var ErrInvalidModuledata = errors.New("invalid moduledata")
var ErrNoEnoughDataForVMD = errors.New("no enough data to read moduledata")

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

func buildPclnTabAddrBinary(order binary.ByteOrder, addr uint64) ([]byte, error) {
	buf := make([]byte, intSize32)
	order.PutUint32(buf, uint32(addr))
	return buf, nil
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

func validateModuledata(md Moduledata, fileInfo *FileInfo, f fileHandler) (bool, error) {
	// Take a simple validation step to ensure that the moduledata is valid.
	text := md.Text()

	textSectAddr, textSect, err := f.getCodeSection()
	if err != nil {
		// this is not a failed validation, but a real error needs to be resolved
		return false, err
	}

	mdTextStart := text.Address
	mdTextEnd := text.Address + text.Length

	return textSectAddr <= mdTextStart && mdTextEnd <= textSectAddr+uint64(len(textSect)), nil
}

func searchModuledata(vmd modulable, fileInfo *FileInfo, f fileHandler) (moduledata, error) {
	_, secData, err := f.getSectionData(f.moduledataSection())
	if err != nil {
		return moduledata{}, err
	}
	tabAddr, _, err := f.getPCLNTABData()
	if err != nil {
		return moduledata{}, err
	}

	magic, err := buildPclnTabAddrBinary(fileInfo.ByteOrder, tabAddr)
	if err != nil {
		return moduledata{}, err
	}

	// If we have a versioned moduledata, we can skip the search.
	var candidates []modulable
	if vmd == nil {
		bits := 32
		if fileInfo.WordSize == intSize64 {
			bits = 64
		}
		candidates, _ = getModuleDataList(bits)
	}

	// Mark a recoverable error with position information.
	type offsetInvalidError struct {
		error
		offset int
	}

	// for search, we always need validation, so the behavior here is different from readModuledataFromSymbol
	trySearch := func(vmd modulable, data []byte) (moduledata, error) {
		off := bytes.Index(secData, magic)

		if off == -1 {
			return moduledata{}, errors.New("could not find pclntab address")
		}

		tryLoad := func(vmd modulable, off int) (moduledata, error) {
			vmdSize := binary.Size(vmd)
			if len(secData) < off+vmdSize {
				return moduledata{}, ErrNoEnoughDataForVMD
			}

			mdData := secData[off : off+vmdSize]

			// Read the module struct from the file.
			r := bytes.NewReader(mdData)
			err = binary.Read(r, fileInfo.ByteOrder, vmd)
			if err != nil {
				return moduledata{}, fmt.Errorf("error when reading module data: %w", err)
			}

			// Convert the read struct to the type we return to the caller.
			md := vmd.toModuledata()

			valid, err := validateModuledata(md, fileInfo, f)
			if err != nil {
				return moduledata{}, err
			}
			if !valid {
				return moduledata{}, offsetInvalidError{ErrInvalidModuledata, off}
			}
			return md, nil
		}

		if vmd != nil {
			return tryLoad(vmd, off)
		} else {
			minVmdSize := binary.Size(candidates[0])
			for _, candidateVmd := range candidates {
				minVmdSize = min(minVmdSize, binary.Size(candidateVmd))
			}

			for _, candidateVmd := range candidates {
				md, err := tryLoad(candidateVmd, off)
				if err == nil {
					return md, nil
				}
			}

			if len(secData) < off+minVmdSize {
				return moduledata{}, ErrNoEnoughDataForVMD
			}

			return moduledata{}, offsetInvalidError{errors.New("could not find moduledata with this match"), off}
		}

	}

	var offErr offsetInvalidError
	current := secData
	for {
		md, err := trySearch(vmd, current)
		if err == nil {
			md.fh = f
			return md, nil
		}
		if !errors.As(err, &offErr) {
			return moduledata{}, err
		}
		current = current[offErr.offset+1:]
	}
}

// Normally, we believe the info read from symbol
// is always correct, so no validation is needed.
// But without the goversion a brute force search is needed.
// And the moduledata can be malformed.
func readModuledataFromSymbol(vmd modulable, fileInfo *FileInfo, f fileHandler) (moduledata, error) {
	_, addr, err := f.getSymbol("runtime.firstmoduledata")
	if err != nil {
		return moduledata{}, err
	}

	base, data, err := f.getSectionDataFromAddress(addr)
	if err != nil {
		return moduledata{}, err
	}

	tryLoad := func(vmd modulable, validate bool) (moduledata, error) {
		vmdSize := binary.Size(vmd)
		if addr-base+uint64(vmdSize) > uint64(len(data)) {
			return moduledata{}, errors.New("moduledata is too big")
		}
		r := bytes.NewReader(data[addr-base : addr-base+uint64(vmdSize)])
		err = binary.Read(r, fileInfo.ByteOrder, vmd)
		if err != nil {
			return moduledata{}, fmt.Errorf("error when reading module data from file: %w", err)
		}
		md := vmd.toModuledata()

		if validate {
			valid, err := validateModuledata(md, fileInfo, f)
			if err != nil {
				return moduledata{}, err
			}
			if !valid {
				return moduledata{}, errors.New("moduledata is invalid")
			}
		}
		return md, nil
	}

	if vmd != nil {
		return tryLoad(vmd, true)
	} else {
		// cannot determine the version, so we have to traverse it
		var bits int
		if fileInfo.WordSize == intSize32 {
			bits = 32
		} else {
			bits = 64
		}

		candidates, _ := getModuleDataList(bits)
		for _, candidateVmd := range candidates {
			// can have error result, need to validate
			md, err := tryLoad(candidateVmd, true)
			if err == nil {
				return md, nil
			}
		}
		return moduledata{}, errors.New("could not find moduledata")
	}

}

func extractModuledata(fileInfo *FileInfo, f fileHandler) (moduledata, error) {
	vmd, err := pickVersionedModuleData(fileInfo)
	if err != nil {
		if !errors.Is(err, ErrNoGoVersionFound) {
			return moduledata{}, err
		}
	}

	hasSymbol, err := f.hasSymbolTable()
	if err != nil {
		return moduledata{}, err
	}
	if hasSymbol {
		md, err := readModuledataFromSymbol(vmd, fileInfo, f)
		if err == nil {
			md.fh = f
			return md, nil
		}
	}

	md, err := searchModuledata(vmd, fileInfo, f)
	if err != nil {
		return moduledata{}, err
	}
	md.fh = f
	return md, nil
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
