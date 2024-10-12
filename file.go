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
	"debug/dwarf"
	"debug/gosym"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"sync"

	macho2 "github.com/blacktop/go-macho"
	"github.com/blacktop/go-macho/pkg/fixupchains"
)

var (
	elfMagic       = []byte{0x7f, 0x45, 0x4c, 0x46}
	peMagic        = []byte{0x4d, 0x5a}
	maxMagicBufLen = 4
	machoMagic1    = []byte{0xfe, 0xed, 0xfa, 0xce}
	machoMagic2    = []byte{0xfe, 0xed, 0xfa, 0xcf}
	machoMagic3    = []byte{0xce, 0xfa, 0xed, 0xfe}
	machoMagic4    = []byte{0xcf, 0xfa, 0xed, 0xfe}
)

// Open opens a file and returns a handler to the file.
func Open(filePath string) (*GoFile, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	_, err = f.Seek(0, io.SeekStart)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, maxMagicBufLen)
	n, err := f.Read(buf)
	_ = f.Close()
	if err != nil {
		return nil, err
	}
	if n < maxMagicBufLen {
		return nil, ErrNotEnoughBytesRead
	}
	gofile := new(GoFile)
	if fileMagicMatch(buf, elfMagic) {
		elf, err := openELF(filePath)
		if err != nil {
			return nil, err
		}
		gofile.fh = elf
	} else if fileMagicMatch(buf, peMagic) {
		pe, err := openPE(filePath)
		if err != nil {
			return nil, err
		}
		gofile.fh = pe
	} else if fileMagicMatch(buf, machoMagic1) || fileMagicMatch(buf, machoMagic2) || fileMagicMatch(buf, machoMagic3) || fileMagicMatch(buf, machoMagic4) {
		macho, err := openMachO(filePath)
		if err != nil {
			return nil, err
		}
		gofile.fh = macho
	} else {
		return nil, ErrUnsupportedFile
	}
	gofile.FileInfo = gofile.fh.getFileInfo()

	// If the ID has been removed or tampered with, this will fail. If we can't
	// get a build ID, we skip it.
	buildID, err := gofile.fh.getBuildID()
	if err == nil {
		gofile.BuildID = buildID
	}

	// Try to extract build information.
	if bi, err := gofile.extractBuildInfo(); err == nil {
		// This error is a minor failure; it just means we don't have
		// this information.
		// So if fails, we just ignore it.
		gofile.BuildInfo = bi
		if bi.Compiler != nil {
			gofile.FileInfo.goversion = bi.Compiler
		}
	}

	return gofile, nil
}

// GoFile is a structure representing a go binary file.
type GoFile struct {
	// BuildInfo holds the data from the buildinfo structure.
	// This can be a nil because it's not always available.
	BuildInfo *BuildInfo
	// FileInfo holds information about the file.
	FileInfo *FileInfo
	// BuildID is the Go build ID hash extracted from the binary.
	BuildID string

	fh fileHandler

	stdPkgs   []*Package
	generated []*Package
	pkgs      []*Package
	vendors   []*Package
	unknown   []*Package

	pclntab *gosym.Table

	initPackagesOnce  sync.Once
	initPackagesError error

	runtimeText  uint64
	pclntabAddr  uint64
	pclntabBytes []byte
	pclntabOnce  sync.Once
	pclntabError error

	moduledata moduledata

	versionError error

	initModuleDataOnce  sync.Once
	initModuleDataError error
}

func (f *GoFile) initModuleData() error {
	f.initModuleDataOnce.Do(func() {
		err := f.ensureCompilerVersion()
		if err != nil {
			f.initModuleDataError = err
			return
		}
		f.moduledata, f.initModuleDataError = extractModuledata(f)
	})
	return f.initModuleDataError
}

// Moduledata extracts the file's moduledata.
func (f *GoFile) Moduledata() (Moduledata, error) {
	err := f.initModuleData()
	if err != nil {
		return moduledata{}, err
	}
	return f.moduledata, nil
}

func (f *GoFile) initPackages() error {
	f.initPackagesOnce.Do(func() {
		tab, err := f.PCLNTab()
		if err != nil {
			f.initPackagesError = err
			return
		}
		f.pclntab = tab
		f.initPackagesError = f.enumPackages()
	})
	return f.initPackagesError
}

// GetFile returns the raw file opened by the library.
func (f *GoFile) GetFile() *os.File {
	return f.fh.getFile()
}

// GetParsedFile returns the parsed file, should be cast based on the file type.
// Possible types are:
//   - *elf.File
//   - *pe.File
//   - *macho.File
//
// all from the debug package.
func (f *GoFile) GetParsedFile() any {
	return f.fh.getParsedFile()
}

// GetCompilerVersion returns the Go compiler version of the compiler
// that was used to compile the binary.
func (f *GoFile) GetCompilerVersion() (*GoVersion, error) {
	err := f.ensureCompilerVersion()
	if err != nil {
		return nil, err
	}
	return f.FileInfo.goversion, nil
}

func (f *GoFile) ensureCompilerVersion() error {
	if f.FileInfo.goversion == nil {
		f.tryExtractCompilerVersion()
	}
	return f.versionError
}

// tryExtractCompilerVersion tries to extract the compiler version from the binary.
// should only be called if FileInfo.goversion is nil.
func (f *GoFile) tryExtractCompilerVersion() {
	if f.FileInfo.goversion != nil {
		return
	}
	v, err := findGoCompilerVersion(f)
	if err != nil {
		f.versionError = err
	} else {
		f.FileInfo.goversion = v
	}
}

// SourceInfo returns the source code filename, starting line number
// and ending line number for the function.
func (f *GoFile) SourceInfo(fn *Function) (string, int, int) {
	srcFile, _, _ := f.pclntab.PCToLine(fn.Offset)
	start, end := findSourceLines(fn.Offset, fn.End, f.pclntab)
	return srcFile, start, end
}

// GetGoRoot returns the Go Root path used to compile the binary.
func (f *GoFile) GetGoRoot() (string, error) {
	err := f.initPackages()
	if err != nil {
		return "", err
	}
	return findGoRootPath(f)
}

// SetGoVersion sets the assumed compiler version that was used. This
// can be used to force a version if gore is not able to determine the
// compiler version used. The version string must match one of the strings
// normally extracted from the binary. For example, to set the version to
// go 1.12.0, use "go1.12". For 1.7.2, use "go1.7.2".
// If an incorrect version string or version not known to the library,
// ErrInvalidGoVersion is returned.
func (f *GoFile) SetGoVersion(version string) error {
	gv := ResolveGoVersion(version)
	if gv == nil {
		return ErrInvalidGoVersion
	}
	f.FileInfo.goversion = gv
	return nil
}

// GetPackages returns the go packages that have been classified as part of the main
// project.
func (f *GoFile) GetPackages() ([]*Package, error) {
	err := f.initPackages()
	return f.pkgs, err
}

// GetVendors returns the third party packages used by the binary.
func (f *GoFile) GetVendors() ([]*Package, error) {
	err := f.initPackages()
	return f.vendors, err
}

// GetSTDLib returns the standard library packages used by the binary.
func (f *GoFile) GetSTDLib() ([]*Package, error) {
	err := f.initPackages()
	return f.stdPkgs, err
}

// GetGeneratedPackages returns the compiler generated packages used by the binary.
func (f *GoFile) GetGeneratedPackages() ([]*Package, error) {
	err := f.initPackages()
	return f.generated, err
}

// GetUnknown returns unclassified packages used by the binary.
// This is a catch-all category when the classification could not be determined.
func (f *GoFile) GetUnknown() ([]*Package, error) {
	err := f.initPackages()
	return f.unknown, err
}

func (f *GoFile) enumPackages() error {
	tab := f.pclntab
	packages := make(map[string]*Package)
	allPackages := sort.StringSlice{}

	for _, n := range tab.Funcs {
		p, ok := packages[n.PackageName()]
		if !ok {
			p = &Package{
				Filepath:  "", // to be filled later by dir(PCToLine())
				Functions: make([]*Function, 0),
				Methods:   make([]*Method, 0),
			}
			packages[n.PackageName()] = p
			allPackages = append(allPackages, n.PackageName())
		}

		if n.ReceiverName() != "" {
			m := &Method{
				Function: &Function{
					Name:        n.BaseName(),
					Offset:      n.Entry,
					End:         n.End,
					PackageName: n.PackageName(),
				},
				Receiver: n.ReceiverName(),
			}

			p.Methods = append(p.Methods, m)
		} else {
			f := &Function{
				Name:        n.BaseName(),
				Offset:      n.Entry,
				End:         n.End,
				PackageName: n.PackageName(),
			}
			p.Functions = append(p.Functions, f)
		}

		if p.Filepath == "" {
			fp, _, _ := tab.PCToLine(n.Entry)
			switch fp {
			case "<autogenerated>", "":
				pkg := n.PackageName()
				if pkg == "" {
					p.Filepath = fp
				}
			default:
				p.Filepath = path.Dir(fp)
			}
		}
	}

	allPackages.Sort()

	var classifier PackageClassifier

	if f.BuildInfo != nil && f.BuildInfo.ModInfo != nil {
		classifier = NewModPackageClassifier(f.BuildInfo.ModInfo)
	} else {
		mainPkg, ok := packages["main"]
		if !ok {
			return fmt.Errorf("no main package found")
		}

		classifier = NewPathPackageClassifier(mainPkg.Filepath)
	}

	for n, p := range packages {
		p.Name = n
		class := classifier.Classify(p)
		switch class {
		case ClassSTD:
			f.stdPkgs = append(f.stdPkgs, p)
		case ClassVendor:
			f.vendors = append(f.vendors, p)
		case ClassMain:
			f.pkgs = append(f.pkgs, p)
		case ClassUnknown:
			f.unknown = append(f.unknown, p)
		case ClassGenerated:
			f.generated = append(f.generated, p)
		}
	}
	return nil
}

// Close releases the file handler.
func (f *GoFile) Close() error {
	return f.fh.Close()
}

func (f *GoFile) getPCLNTABDataBySymbol() (uint64, []byte, error) {
	start, _, err := f.fh.getSymbol("runtime.pclntab")
	if err != nil {
		return 0, nil, err
	}
	end, _, err := f.fh.getSymbol("runtime.epclntab")
	if err != nil {
		return 0, nil, err
	}
	if end < start {
		return 0, nil, errors.New("invalid pclntab symbols")
	}
	sectStart, data, err := f.fh.getSectionDataFromAddress(start)
	if err != nil {
		return 0, nil, err
	}
	// ensure that the pclntab is within the same section
	if end-sectStart > uint64(len(data)) {
		return 0, nil, errors.New("pclntab out of bounds")
	}
	return start, data[start-sectStart : end-sectStart], nil
}

func (f *GoFile) initPclntab() error {
	f.pclntabOnce.Do(func() {
		addr, data, err := f.getPCLNTABDataBySymbol()
		if err != nil {
			addr, data, err = f.fh.getPCLNTABData()
			if err != nil {
				f.pclntabError = fmt.Errorf("error when getting pclntab: %w", err)
				return
			}
		}
		f.pclntabAddr = addr
		f.pclntabBytes = data

		// All the function address in the pclntab uses the symbol "runtime.text" as the base address.
		// This symbol is where the runtime uses as the start of the code section. While it should always
		// be located within the binary's text section, it may not be at the start of the section. For example,
		// external linkers may add additional code to the section before the "Go" code. We can find "runtime.text"
		// in the moduledata structure in the binary.
		// If we have the symbol table, just get it
		val, _, err := f.fh.getSymbol("runtime.text")
		if err == nil {
			f.runtimeText = val
			return
		}

		// Otherwise, we need to search it
		_, moddataSection, err := f.fh.getSectionData(f.fh.moduledataSection())
		if err != nil {
			f.pclntabError = fmt.Errorf("failed to get the section %s where the moduledata structure is stored: %w", f.fh.moduledataSection(), err)
			return
		}

		// At this point, we don't know what compiler version was used so we can't parse the moduledata structure.
		// We do know the field in different structure versions so we can check these offsets and see if the fall
		// within the text section.
		textStart, textData, err := f.fh.getCodeSection()
		if err != nil {
			f.pclntabError = fmt.Errorf("failed to get the file's text section: %w", err)
			return
		}

		// Since the moduledata starts with the address to the pclntab, we can use this to find the moduledata structure.
		runtimeText, err := f.findRuntimeText(textStart, textStart+uint64(len(textData)), f.pclntabAddr, moddataSection)
		if err != nil {
			if f.FileInfo.OS == "macOS" && f.FileInfo.Arch == ArchARM64 {
				t, err := f.findRuntimeTextMachoChainedFixups(f.pclntabAddr)
				if err != nil {
					f.pclntabError = fmt.Errorf("failed to find runtime.text symbol: %w", err)
					return
				}
				f.runtimeText = t
				return
			}

			f.pclntabError = fmt.Errorf("failed to find runtime.text symbol: %w", err)
			return
		}
		f.runtimeText = runtimeText
	})
	return f.pclntabError
}

// PCLNTab returns the PCLN table.
func (f *GoFile) PCLNTab() (*gosym.Table, error) {
	err := f.initPclntab()
	if err != nil {
		return nil, err
	}
	return gosym.NewTable(make([]byte, 0), gosym.NewLineTable(f.pclntabBytes, f.runtimeText))
}

func (f *GoFile) findRuntimeTextMachoChainedFixups(pclntabAddr uint64) (uint64, error) {
	of := f.fh.getFile()
	_, err := of.Seek(0, io.SeekStart)
	if err != nil {
		return 0, err
	}

	f2, err := macho2.NewFile(of)
	if err != nil {
		return 0, err
	}
	fixups, err := f2.DyldChainedFixups()
	if err != nil {
		return 0, err
	}
	baseAddr := f2.GetBaseAddress()
	var rebases []fixupchains.Rebase
	for _, start := range fixups.Starts {
		rebases = append(rebases, start.Rebases()...)
	}

	// First, we need to find the start of the moduledata
	var moduledataAddr uint64
	for _, rb := range rebases {
		if rb.Target()+baseAddr == pclntabAddr {
			moduledataAddr = baseAddr + rb.Offset()
			break
		}
	}
	// then, find field 22
	addr22 := moduledataAddr + 22*8
	for _, rb := range rebases {
		if rb.Offset()+baseAddr == addr22 {
			return baseAddr + rb.Target(), nil
		}
	}
	return 0, fmt.Errorf("failed to find runtime.text symbol")
}

func (f *GoFile) findRuntimeText(textStart, textEnd, pclntabAddr uint64, modSectiondata []byte) (uint64, error) {
	var text, etext uint64
	magic := buildPclnTabAddrBinary(f.FileInfo.WordSize, f.FileInfo.ByteOrder, pclntabAddr)
	for {
		// Search for a potential match of the moduledata structure.
		offset := bytes.Index(modSectiondata, magic)

		// If we got -1 back, nothing was found. If the offset is close to the end of the section
		// it's not the correct match and we didn't find the structure.
		if offset == -1 || len(modSectiondata[offset:]) < 30*f.FileInfo.WordSize {
			return 0, fmt.Errorf("moduledata structure not found")
		}

		// We first check field 22 and 23 for runtime.text and runtime.etext. Current Go versions.
		if f.FileInfo.WordSize == intSize32 {
			text = uint64(f.FileInfo.ByteOrder.Uint32(modSectiondata[offset+22*f.FileInfo.WordSize:]))
			etext = uint64(f.FileInfo.ByteOrder.Uint32(modSectiondata[offset+23*f.FileInfo.WordSize:]))
		} else {
			text = f.FileInfo.ByteOrder.Uint64(modSectiondata[offset+22*f.FileInfo.WordSize:])
			etext = f.FileInfo.ByteOrder.Uint64(modSectiondata[offset+23*f.FileInfo.WordSize:])
		}
		if text >= textStart && text < textEnd && etext > textStart && etext <= textEnd {
			return text, nil
		}

		// If fields 22 and 23 didn't return what we expected, we check fields 12 and 13. These fields
		// are for older Go versions.
		if f.FileInfo.WordSize == intSize32 {
			text = uint64(f.FileInfo.ByteOrder.Uint32(modSectiondata[offset+12*f.FileInfo.WordSize:]))
			etext = uint64(f.FileInfo.ByteOrder.Uint32(modSectiondata[offset+13*f.FileInfo.WordSize:]))
		} else {
			text = f.FileInfo.ByteOrder.Uint64(modSectiondata[offset+12*f.FileInfo.WordSize:])
			etext = f.FileInfo.ByteOrder.Uint64(modSectiondata[offset+13*f.FileInfo.WordSize:])
		}
		if text >= textStart && text < textEnd && etext > textStart && etext <= textEnd {
			return text, nil
		}

		modSectiondata = modSectiondata[offset+1:]
	}
}

// GetTypes returns a map of all types found in the binary file.
func (f *GoFile) GetTypes() ([]*GoType, error) {
	err := f.initModuleData()
	if err != nil {
		return nil, err
	}
	md := f.moduledata

	t, err := getTypes(f.FileInfo, f.fh, md)
	if err != nil {
		return nil, err
	}
	if err = f.initPackages(); err != nil {
		return nil, err
	}
	return sortTypes(t), nil
}

// Bytes return a slice of raw bytes with the length in the file from the address.
func (f *GoFile) Bytes(address uint64, length uint64) ([]byte, error) {
	base, section, err := f.fh.getSectionDataFromAddress(address)
	if err != nil {
		return nil, err
	}

	if address+length-base > uint64(len(section)) {
		return nil, errors.New("length out of bounds")
	}

	return section[address-base : address+length-base], nil
}

func sortTypes(types map[uint64]*GoType) []*GoType {
	sortedList := make([]*GoType, len(types))

	i := 0
	for _, typ := range types {
		sortedList[i] = typ
		i++
	}
	sort.Slice(sortedList, func(i, j int) bool {
		if sortedList[i].PackagePath == sortedList[j].PackagePath {
			return sortedList[i].Name < sortedList[j].Name
		}
		return sortedList[i].PackagePath < sortedList[j].PackagePath
	})
	return sortedList
}

type fileHandler interface {
	io.Closer
	// returns the value, size and error
	getSymbol(name string) (uint64, uint64, error)
	getRData() ([]byte, error)
	getCodeSection() (uint64, []byte, error)
	getSectionDataFromAddress(uint64) (uint64, []byte, error)
	getSectionData(string) (uint64, []byte, error)
	getFileInfo() *FileInfo
	getPCLNTABData() (uint64, []byte, error)
	moduledataSection() string
	getBuildID() (string, error)
	getFile() *os.File
	getParsedFile() any
	getDwarf() (*dwarf.Data, error)
}

func fileMagicMatch(buf, magic []byte) bool {
	return bytes.HasPrefix(buf, magic)
}

// FileInfo holds information about the file.
type FileInfo struct {
	// Arch is the architecture the binary is compiled for.
	Arch string
	// OS is the operating system the binary is compiled for.
	OS string
	// ByteOrder is the byte order.
	ByteOrder binary.ByteOrder
	// WordSize is the natural integer size used by the file.
	WordSize  int
	goversion *GoVersion
}

const (
	ArchAMD64 = "amd64"
	ArchARM   = "arm"
	ArchARM64 = "arm64"
	Arch386   = "i386"
	ArchMIPS  = "mips"
)
