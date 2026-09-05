// Copyright (C) 2019-2026 GoRE Authors.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package gore

//go:generate go run ./gen moduledata

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"go/version"
	"io"
	"strconv"
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
	// ITabLinks returns the itablinks pointer array used before Go 1.27.
	// Length is the number of pointers. Go 1.27+ has no separate pointer array;
	// use ITabLinksData to read interface table addresses from either layout.
	ITabLinks() ModuleDataSection
	// ITabLinksData returns the virtual addresses of the static interface tables.
	ITabLinksData() ([]uint64, error)
	// TypeLink returns the typelink section.
	TypeLink() ModuleDataSection
	// TypeLinkData returns the typelink section data.
	TypeLinkData() ([]int32, error)
	// GoFuncValue returns the value of the 'go:func.*' symbol.
	GoFuncValue() uint64
	// ResolvePointer resolves a pointer value read from fileAddr in the binary.
	// On PIE binaries (Mach-O chained fixups, ELF RELATIVE relocations), raw
	// pointer values in data sections are fixup descriptors, not actual addresses.
	// Returns the resolved address, or val unchanged for non-PIE binaries.
	ResolvePointer(val uint64, fileAddr uint64) uint64
}

type moduledata struct {
	TextAddr, TextLen           uint64
	NoPtrDataAddr, NoPtrDataLen uint64
	DataAddr, DataLen           uint64
	BssAddr, BssLen             uint64
	NoPtrBssAddr, NoPtrBssLen   uint64

	TypesAddr, TypesLen       uint64
	TypeDescLen               uint64
	TypelinkAddr, TypelinkLen uint64
	ITabLinkAddr, ITabLinkLen uint64
	ITabOffset, ITabSize      uint64
	FuncTabAddr, FuncTabLen   uint64
	PCLNTabAddr, PCLNTabLen   uint64

	GoFuncVal uint64

	fh       fileHandler
	fileInfo *FileInfo
	resolver pointerResolver
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
	fileInfo := m.fileInfo
	if fileInfo == nil || fileInfo.goversion == nil {
		return nil, ErrNoGoVersionFound
	}
	if usesGo127TypeLayout(fileInfo.goversion.Name) {
		if m.TypeDescLen == 0 {
			return nil, errors.New("Go 1.27 moduledata has no type descriptor region")
		}
		types, err := m.Types().Data()
		if err != nil {
			return nil, fmt.Errorf("failed to get types data section: %w", err)
		}
		parser := newTypeParser(types, m.TypesAddr, fileInfo, m.fh, newPointerResolver(m.fh))
		offsets, err := parseTypeDescriptors(parser, m.TypeDescLen)
		if err != nil {
			return nil, fmt.Errorf("failed to parse type descriptors: %w", err)
		}
		return offsets, nil
	}

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

// ResolvePointer resolves a pointer value read from fileAddr in the binary.
func (m moduledata) ResolvePointer(val uint64, fileAddr uint64) uint64 {
	return resolvePointer(m.resolver, val, fileAddr)
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

	name := stripVersionSuffix(info.goversion.Name)
	if !version.IsValid(name) {
		return nil, errors.New("could not parse the go version " + info.goversion.Name)
	}

	lang := version.Lang(name) // e.g. "go1.26"
	if len(lang) <= len("go1.") {
		return nil, errors.New("could not parse minor version from " + name)
	}

	verBit, err := strconv.Atoi(lang[len("go1."):])
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
	var fromSymbol bool

	secAddr, secData, err := f.fh.getSectionData(f.fh.moduledataSection())
	if err != nil {
		return moduledata{}, err
	}
	secSize := uint64(len(secData))

	// if we can get the moduledata addr from the symbol, we have no need to search
	sym, err := f.fh.getSymbol("runtime.firstmoduledata")
	if err == nil {
		off = int(sym.Value - secAddr)
		fromSymbol = true
		goto load
	}

	err = f.initPclntab()
	if err != nil {
		return moduledata{}, err
	}
	tabAddr = f.pclntabAddr

	magic = buildPclnTabAddrBinary(f.FileInfo.WordSize, f.FileInfo.ByteOrder, tabAddr)

search:
	off = bytes.Index(secData, magic)

	if off == -1 {
		r := newPointerResolver(f.fh)
		off = findPointerValue(r, secAddr, secData, tabAddr, f.FileInfo.WordSize, f.FileInfo.ByteOrder)
	}

load:
	if off == -1 {
		return moduledata{}, errors.New("could not find moduledata")
	}

	available := len(secData) - off
	if available <= 0 {
		return moduledata{}, fmt.Errorf("offset %d is out of bounds %d", off, len(secData))
	}

	// When the section is slightly smaller than the struct (e.g. Go 1.26's __go_module
	// section may be 1 byte short), zero-fill the remainder. The trailing fields
	// (typically slice capacities) are not critical for analysis.
	var data []byte
	if available >= vmdSize {
		data = secData[off : off+vmdSize]
	} else {
		data = make([]byte, vmdSize)
		copy(data, secData[off:])
	}

	// On macOS/ELF PIE, pointer fields contain fixup descriptors instead of
	// virtual addresses. Resolve them before binary.Read so derived values
	// like TextLen = Etext - Text are computed from correct addresses.
	resolver := newPointerResolver(f.fh)
	if resolver != nil {
		baseAddr := secAddr + uint64(off)
		ws := f.FileInfo.WordSize
		bo := f.FileInfo.ByteOrder
		for _, ptrOff := range vmd.pointerOffsets() {
			if ptrOff+ws > len(data) {
				break
			}
			var raw uint64
			if ws == 4 {
				raw = uint64(bo.Uint32(data[ptrOff:]))
			} else {
				raw = bo.Uint64(data[ptrOff:])
			}
			val := resolver.ResolvePointer(raw, baseAddr+uint64(ptrOff))
			if val != raw {
				if ws == 4 {
					bo.PutUint32(data[ptrOff:], uint32(val))
				} else {
					bo.PutUint64(data[ptrOff:], val)
				}
			}
		}
	}

	r := bytes.NewReader(data)
	err = binary.Read(r, f.FileInfo.ByteOrder, vmd)
	if err != nil {
		return moduledata{}, fmt.Errorf("error when reading module data from file: %w", err)
	}

	md := vmd.toModuledata()
	usesTypeDescriptors := usesGo127TypeLayout(f.FileInfo.goversion.Name)

	// WebAssembly code addresses are function indices rather than ranges in
	// linear memory, so the native text-section validation does not apply. The
	// pclntab and types still have to point into linear memory; use those ranges
	// to reject false matches for the pclntab pointer. Go 1.27 stores typelink
	// descriptors inline at the beginning of the types region instead of in a
	// separate typelinks slice.
	if f.FileInfo.Arch == ArchWASM {
		if !validWasmModuledata(md, secSize, usesTypeDescriptors) {
			if fromSymbol {
				return moduledata{}, errors.New("WebAssembly moduledata at symbol address failed memory validation")
			}
			secData = secData[off+1:]
			goto search
		}
		md.fh = f.fh
		md.fileInfo = f.FileInfo
		return md, nil
	}

	// Take a simple validation step to ensure that the moduledata is valid.
	text := md.TextAddr
	etext := md.TextAddr + md.TextLen

	textSectAddr, textSect, err := f.fh.getCodeSection()
	if err != nil {
		return moduledata{}, err
	}

	if text > etext || !(textSectAddr <= text && text < textSectAddr+uint64(len(textSect))) {
		if fromSymbol {
			return moduledata{}, fmt.Errorf("moduledata at symbol address failed text validation")
		}
		secData = secData[off+1:]
		goto search
	}

	md.fh = f.fh
	md.fileInfo = f.FileInfo
	md.resolver = resolver

	return md, nil
}

func validWasmModuledata(md moduledata, memorySize uint64, usesTypeDescriptors bool) bool {
	if md.PCLNTabLen == 0 || md.TypesLen == 0 {
		return false
	}
	if !addressRangeWithin(md.PCLNTabAddr, md.PCLNTabLen, memorySize) ||
		!addressRangeWithin(md.TypesAddr, md.TypesLen, memorySize) {
		return false
	}
	if usesTypeDescriptors {
		return md.TypeDescLen != 0 && md.TypeDescLen <= md.TypesLen
	}
	if md.TypelinkLen == 0 {
		return false
	}
	if md.TypelinkLen > ^uint64(0)/4 {
		return false
	}
	return addressRangeWithin(md.TypelinkAddr, md.TypelinkLen*4, memorySize)
}

func addressRangeWithin(address, length, limit uint64) bool {
	return address <= limit && length <= limit-address
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
	pointerOffsets() []int
}
