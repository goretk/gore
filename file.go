// Copyright 2019 The GoRE.tk Authors. All rights reserved.
// Use of this source code is governed by the license that
// can be found in the LICENSE file.

package gore

import (
	"bytes"
	"context"
	"debug/gosym"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

var (
	elfMagic       = []byte{0x7f, 0x45, 0x4c, 0x46}
	elfMagicOffset = 0
	peMagic        = []byte{0x4d, 0x5a}
	peMagicOffset  = 0
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

	_, err = f.Seek(0, 0)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, maxMagicBufLen)
	n, err := f.Read(buf)
	f.Close()
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

	return gofile, nil
}

// GoFile is a structure representing a go binary file.
type GoFile struct {
	// FileInfo holds information about the file.
	FileInfo *FileInfo
	// BuildID is the Go build ID hash extracted from the binary.
	BuildID      string
	fh           fileHandler
	stdPkgs      []*Package
	generated    []*Package
	pkgs         []*Package
	vendors      []*Package
	unknown      []*Package
	pclntab      *gosym.Table
	initPackages sync.Once
}

func (f *GoFile) init() error {
	var returnVal error
	f.initPackages.Do(func() {
		tab, err := f.PCLNTab()
		if err != nil {
			returnVal = err
			return
		}
		f.pclntab = tab
		returnVal = f.enumPackages()
	})
	return returnVal
}

// GetCompilerVersion returns the Go compiler version of the compiler
// that was used to compile the binary.
func (f *GoFile) GetCompilerVersion() (*GoVersion, error) {
	return findGoCompilerVersion(f)
}

// GetGoRoot returns the Go Root path
// that was used to compile the binary.
func (f *GoFile) GetGoRoot() (string, error) {
	err := f.init()
	if err != nil {
		return "", err
	}
	return findGoRootPath(f)
}

// SetGoVersion sets the assumed compiler version that was used. This
// can be used to force a version if gore is not able to determine the
// compiler version used. The version string must match one of the strings
// normally extracted from the binary. For example to set the version to
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

// GetPackages returns the go packages that has been classified as part of the main
// project.
func (f *GoFile) GetPackages() ([]*Package, error) {
	err := f.init()
	return f.pkgs, err
}

// GetVendors returns the 3rd party packages used by the binary.
func (f *GoFile) GetVendors() ([]*Package, error) {
	err := f.init()
	return f.vendors, err
}

// GetSTDLib returns the standard library packages used by the binary.
func (f *GoFile) GetSTDLib() ([]*Package, error) {
	err := f.init()
	return f.stdPkgs, err
}

// GetGeneratedPackages returns the compiler generated packages used by the binary.
func (f *GoFile) GetGeneratedPackages() ([]*Package, error) {
	err := f.init()
	return f.generated, err
}

// GetUnknown returns unclassified packages used by the binary. This is a catch all
// category when the classification could not be determined.
func (f *GoFile) GetUnknown() ([]*Package, error) {
	err := f.init()
	return f.unknown, err
}

// findSourceLines walks from the entry of the function to the end and looks for the
// final source code line number. This function is pretty expensive to execute.
func findSourceLines(entry, end uint64, tab *gosym.Table) (int, int) {
	// We don't need the Func returned since we are operating within the same function.
	file, srcStart, _ := tab.PCToLine(entry)

	// We walk from entry to end and check the source code line number. If it's greater
	// then the current value, we set it as the new value. If the file is different, we
	// have entered an inlined function. In this case we skip it. There is a possibility
	// that we enter an inlined function that's defined in the same file. There is no way
	// for us to tell this is the case.
	srcEnd := srcStart

	// We take a shortcut and only check every 4 bytes. This isn't perfect, but it speeds
	// up the processes.
	for i := entry; i <= end; i = i + 4 {
		f, l, _ := tab.PCToLine(i)

		// If this line is a different file, it's an inlined function so just continue.
		if f != file {
			continue
		}

		// If the current line is less than the starting source line, we have entered
		// an inline function defined before this function.
		if l < srcStart {
			continue
		}

		// If the current line is greater, we assume it being closer to the end of the
		// function definition. So we take it as the current srcEnd value.
		if l > srcEnd {
			srcEnd = l
		}
	}

	return srcStart, srcEnd
}

func (f *GoFile) enumPackages() error {
	// Because finding the end source line of a function is costly, this function uses
	// a worker pool to analyze multiple functions in parallel.

	tab := f.pclntab
	packages := make(map[string]*Package)
	allPackages := sort.StringSlice{}

	type methodChanPayload struct {
		pkgName string
		name    string
		method  *Method
	}

	type functionChanPayload struct {
		pkgName  string
		name     string
		function *Function
	}

	var wg sync.WaitGroup
	work := make(chan gosym.Func)
	methodResultChan := make(chan methodChanPayload)
	functionResultChan := make(chan functionChanPayload)
	var pkgMutex sync.Mutex

	// Function executed by each worker.
	worker := func(wg *sync.WaitGroup, w <-chan gosym.Func, methodChan chan<- methodChanPayload, functionChan chan<- functionChanPayload) {
		defer wg.Done()

		for n := range w {
			srcStart, srcStop := findSourceLines(n.Entry, n.End, tab)
			name, _, _ := tab.PCToLine(n.Entry)

			if n.ReceiverName() != "" {
				m := &Method{
					Function: &Function{
						Name:          n.BaseName(),
						SrcLineLength: (srcStop - srcStart),
						SrcLineStart:  srcStart,
						SrcLineEnd:    srcStop,
						Offset:        n.Entry,
						End:           n.End,
						Filename:      filepath.Base(name),
						PackageName:   n.PackageName(),
					},
					Receiver: n.ReceiverName(),
				}

				// Send the method.
				methodChan <- methodChanPayload{pkgName: n.PackageName(), name: name, method: m}
			} else {
				f := &Function{
					Name:          n.BaseName(),
					SrcLineLength: (srcStop - srcStart),
					Offset:        n.Entry,
					End:           n.End,
					SrcLineStart:  srcStart,
					SrcLineEnd:    srcStop,
					Filename:      filepath.Base(name),
					PackageName:   n.PackageName(),
				}
				functionChan <- functionChanPayload{pkgName: n.PackageName(), name: name, function: f}
			}
		}
	}

	// Start workers. 10 workers appears to be enough. No extra performance above it.
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go worker(&wg, work, methodResultChan, functionResultChan)
	}

	// Context cancelation is used to signal the result routine that all workers
	// have completed their work.
	ctx, done := context.WithCancel(context.Background())

	// Result routine. This reads from all channels until the context has been canceled.
	go func() {
		// Locking the mutex prevents potential race condition when this routine still hasn't
		// finished and the main routine wants to start processing the result.
		pkgMutex.Lock()
		defer pkgMutex.Unlock()
		for {
			select {

			case <-ctx.Done():
				return

			case m := <-methodResultChan:
				p, ok := packages[m.pkgName]
				if !ok {
					p = &Package{
						Filepath:  filepath.Dir(m.name),
						Functions: make([]*Function, 0),
						Methods:   make([]*Method, 0),
					}
				}
				p.Methods = append(p.Methods, m.method)
				packages[m.pkgName] = p
				allPackages = append(allPackages, m.pkgName)

			case f := <-functionResultChan:
				p, ok := packages[f.pkgName]
				if !ok {
					p = &Package{
						Filepath:  filepath.Dir(f.name),
						Functions: make([]*Function, 0),
						Methods:   make([]*Method, 0),
					}
				}
				p.Functions = append(p.Functions, f.function)
				packages[f.pkgName] = p
				allPackages = append(allPackages, f.pkgName)

			}
		}
	}()

	// Send work to workers
	for _, n := range tab.Funcs {
		work <- n
	}

	// Close the work channel to indicate no more work is queued.
	close(work)

	// Wait for all workers to finish.
	wg.Wait()

	// Signal to the result routine to exit.
	done()

	// Get the lock, we wait here until the result routine has released the lock.
	pkgMutex.Lock()
	defer pkgMutex.Unlock()

	allPackages.Sort()

	mainPkg, ok := packages["main"]
	if !ok {
		return fmt.Errorf("no main package found")
	}

	classifier := NewPackageClassifier(mainPkg.Filepath)

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

// PCLNTab returns the PCLN table.
func (f *GoFile) PCLNTab() (*gosym.Table, error) {
	return f.fh.getPCLNTab()
}

// GetTypes returns a map of all types found in the binary file.
func (f *GoFile) GetTypes() ([]*GoType, error) {
	if f.FileInfo.goversion == nil {
		ver, err := f.GetCompilerVersion()
		if err != nil {
			return nil, err
		}
		f.FileInfo.goversion = ver
	}
	t, err := getTypes(f.FileInfo, f.fh)
	if err != nil {
		return nil, err
	}
	if err = f.init(); err != nil {
		return nil, err
	}
	return sortTypes(t), nil
}

// Bytes returns a slice of raw bytes with the length in the file from the address.
func (f *GoFile) Bytes(address uint64, length uint64) ([]byte, error) {
	base, section, err := f.fh.getSectionDataFromOffset(address)
	if err != nil {
		return nil, err
	}

	if address+length-base > uint64(len(section)) {
		return nil, errors.New("length out of bounds")
	}

	return section[address-base : address+length-base], nil
}

func sortTypes(types map[uint64]*GoType) []*GoType {
	sortedList := make([]*GoType, len(types), len(types))

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
	getPCLNTab() (*gosym.Table, error)
	getRData() ([]byte, error)
	getCodeSection() ([]byte, error)
	getSectionDataFromOffset(uint64) (uint64, []byte, error)
	getSectionData(string) (uint64, []byte, error)
	getFileInfo() *FileInfo
	getPCLNTABData() (uint64, []byte, error)
	moduledataSection() string
	getBuildID() (string, error)
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
	Arch386   = "i386"
	ArchMIPS  = "mips"
)
