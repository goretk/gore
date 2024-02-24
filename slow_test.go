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
//go:build slow_test

package gore

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var dynResources = []struct {
	os   string
	arch string
}{
	{"linux", "386"},
	{"linux", "amd64"},
	{"windows", "386"},
	{"windows", "amd64"},
	{"darwin", "amd64"},
}

var dynResourceFiles *testFiles

type testFiles struct {
	files sync.Map
}

func (f *testFiles) get(os, arch string, pie, stripped bool) string {
	name := os + "-" + arch
	if pie {
		name += "-pie"
	}
	if !stripped {
		name += "-nostrip"
	}
	exe, ok := f.files.Load(name)
	if !ok {
		return ""
	}
	return exe.(string)
}

// pass nil to check means all combinations
func getMatrix(t *testing.T, checkPie, checkStrip *bool, prefix string, cb func(*testing.T, string)) {
	t.Helper()
	for _, r := range dynResources {
		var pieCases []bool
		if checkPie != nil {
			pieCases = append(pieCases, *checkPie)
		} else {
			pieCases = append(pieCases, true, false)
		}
		for _, pie := range pieCases {
			var strippedCases []bool
			if checkStrip != nil {
				strippedCases = append(strippedCases, *checkStrip)
			} else {
				strippedCases = append(strippedCases, true, false)
			}
			for _, stripped := range strippedCases {
				exe := dynResourceFiles.get(r.os, r.arch, pie, stripped)
				name := r.os + "-" + r.arch
				if pie {
					name += "-pie"
				}
				if !stripped {
					name += "-nostrip"
				}
				t.Run(prefix+"-"+name, func(tt *testing.T) {
					tt.Parallel()
					if exe == "" {
						tt.Skip("no executable available")
					}
					cb(tt, exe)
				})
			}
		}

	}
}

func TestMain(m *testing.M) {
	fmt.Println("Creating test resources, this can take some time...")
	var tmpDirs []string

	resultChan := make(chan buildResult)
	wg := &sync.WaitGroup{}

	dynResourceFiles = &testFiles{files: sync.Map{}}

	go func() {
		for r := range resultChan {
			tmpDirs = append(tmpDirs, r.dir)
			name := r.os + "-" + r.arch
			if r.pie {
				name += "-pie"
			}
			if !r.strip {
				name += "-nostrip"
			}
			dynResourceFiles.files.Store(name, r.exe)
		}
	}()

	for _, r := range dynResources {
		fmt.Printf("Building resource file for %s_%s\n", r.os, r.arch)

		for _, pie := range []bool{false, true} {
			for _, stripped := range []bool{false, true} {
				if pie && r.arch == "386" && r.os == "linux" {
					// seems impossible
					continue
				}
				wg.Add(1)
				go buildTestResource(testresourcesrc, r.os, r.arch, pie, stripped, wg, resultChan)
			}
		}
	}

	wg.Wait()
	close(resultChan)

	fmt.Println("Launching tests")
	code := m.Run()

	fmt.Println("Clean up test resources")

	for _, d := range tmpDirs {
		os.RemoveAll(d)
	}

	os.Exit(code)
}

func TestOpenAndCloseFile(t *testing.T) {
	getMatrix(t, nil, nil, "open", func(tt *testing.T, exe string) {
		a := assert.New(tt)

		f, err := Open(exe)
		a.NoError(err)
		a.NotNil(f)
		a.NoError(f.Close())
	})
}

func TestGetPackages(t *testing.T) {
	getMatrix(t, nil, nil, "getPackages", func(tt *testing.T, exe string) {
		a := assert.New(tt)
		r := require.New(tt)

		f, err := Open(exe)
		r.NoError(err)
		r.NotNil(f)
		defer f.Close()

		std, err := f.GetSTDLib()
		a.NoError(err)
		a.NotEmpty(std, "Should have a list of standard library packages.")

		_, err = f.GetGeneratedPackages()
		a.NoError(err)
		// XXX: This check appears to be unstable. Sometimes files for unknown reason.
		// assert.NotEmpty(gen, "Should have a list of generated packages.")

		ven, err := f.GetVendors()
		a.NoError(err)
		a.Empty(ven, "Should not have a list of vendor packages.")

		_, err = f.GetUnknown()
		a.NoError(err)
		// XXX: This check appears to be unstable. Sometimes files for unknown reason.
		// assert.Empty(unk, "Should not have a list of unknown packages")

		pkgs, err := f.GetPackages()
		a.NoError(err)

		var mainpkg *Package
		for _, p := range pkgs {
			if p.Name == "main" {
				mainpkg = p
				break
			}
		}

		mainPackageFound := false
		getDataFuncFound := false
		a.NotNil(mainpkg, "Should include main package")
		for _, f := range mainpkg.Functions {
			if f.Name == "main" {
				mainPackageFound = true
			} else if f.Name == "getData" {
				getDataFuncFound = true
			} else {
				a.Fail("Unexpected function")
			}
		}
		a.True(mainPackageFound, "No main function found")
		a.True(getDataFuncFound, "getData function not found")
	})
}

func TestGetTypesFromDynamicBuiltResources(t *testing.T) {
	getMatrix(t, nil, nil, "getTypes", func(tt *testing.T, exe string) {
		a := assert.New(t)
		r := require.New(t)
		f, err := Open(exe)
		r.NoError(err)
		r.NotNil(f)
		defer f.Close()

		typs, err := f.GetTypes()
		r.NoError(err)

		var stringer *GoType
		for _, t := range typs {
			if t.PackagePath == "runtime" && t.Name == "runtime.g" {
				stringer = t
				break
			}
		}

		a.NotNil(stringer, "the g type from runtime not found")
	})
}

func TestGetCompilerVersion(t *testing.T) {
	testVersion := testCompilerVersion()
	expectedVersion := ResolveGoVersion(testVersion)

	// If the version could not be resolved, the version is new
	// and the library doesn't know about it. Use the version string
	// to create a new version.
	if expectedVersion == nil {
		expectedVersion = &GoVersion{Name: testVersion}
	}

	getMatrix(t, nil, nil, "compiler-version", func(t *testing.T, exe string) {
		a := assert.New(t)
		r := require.New(t)
		f, err := Open(exe)
		r.NoError(err)
		r.NotNil(f)
		defer f.Close()

		// Test
		version, err := f.GetCompilerVersion()
		a.NoError(err)
		a.Equal(expectedVersion, version)
	})
}

func TestGetBuildID(t *testing.T) {
	getMatrix(t, nil, nil, "buildID", func(t *testing.T, exe string) {
		a := assert.New(t)
		r := require.New(t)
		f, err := Open(exe)
		r.NoError(err)
		r.NotNil(f)
		defer f.Close()

		a.Equal(fixedBuildID, f.BuildID, "BuildID extracted doesn't match expected value.")
	})
}

func TestSourceInfo(t *testing.T) {
	getMatrix(t, nil, nil, "sourceInfo", func(t *testing.T, exe string) {
		a := assert.New(t)
		r := require.New(t)
		f, err := Open(exe)
		r.NoError(err)
		r.NotNil(f)
		defer f.Close()

		var testFn *Function
		pkgs, err := f.GetPackages()
		r.NoError(err)
		for _, pkg := range pkgs {
			if pkg.Name != "main" {
				continue
			}
			for _, fn := range pkg.Functions {
				if fn.Name != "getData" {
					continue
				}
				testFn = fn
				break
			}
		}
		r.NotNil(testFn)

		file, start, end := f.SourceInfo(testFn)

		a.NotEqual(0, start)
		a.NotEqual(0, end)
		a.NotEqual("", file)
	})
}

func TestDwarfString(t *testing.T) {
	noStrip := false
	getMatrix(t, nil, &noStrip, "dwarfString", func(t *testing.T, exe string) {
		r := require.New(t)

		f, err := Open(exe)
		r.NoError(err)
		r.NotNil(f)
		defer f.Close()

		gover, ok := getBuildVersionFromDwarf(f.fh)
		r.True(ok)
		r.Equal(gover, runtime.Version())

		goroot, ok := getGoRootFromDwarf(f.fh)
		r.True(ok)
		r.Equal(goroot, runtime.GOROOT())
	})
}

type buildResult struct {
	exe   string
	dir   string
	strip bool
	pie   bool
	os    string
	arch  string
}

func buildTestResource(body, goos, arch string, pie, stripped bool, wg *sync.WaitGroup, result chan buildResult) {
	defer wg.Done()
	goBin, err := exec.LookPath("go")
	if err != nil {
		panic("No go tool chain found: " + err.Error())
	}

	tmpdir, err := os.MkdirTemp("", "gore-test")
	if err != nil {
		panic(err)
	}

	src := filepath.Join(tmpdir, "a.go")
	err = os.WriteFile(src, []byte(body), 0644)
	if err != nil {
		panic(err)
	}

	exe := filepath.Join(tmpdir, "a")
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}

	var ldFlags string
	if stripped {
		ldFlags = "-s -w "
	}
	ldFlags += "-buildid=" + fixedBuildID
	args := []string{"build", "-o", exe, "-ldflags", ldFlags}
	if pie {
		args = append(args, "-buildmode=pie")
	}
	args = append(args, src)

	cmd := exec.Command(goBin, args...)
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = tmpdir
	}

	cmd.Env = append(cmd.Env, "GOCACHE="+tmpdir, "GOARCH="+arch, "GOOS="+goos, "GOPATH="+gopath, "GOTMPDIR="+gopath, "PATH="+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("building test executable failed: %s\n", string(out))
		os.Exit(1)
	}

	result <- buildResult{exe: exe, dir: tmpdir, strip: stripped, pie: pie, os: goos, arch: arch}
}

func testCompilerVersion() string {
	goBin, err := exec.LookPath("go")
	if err != nil {
		panic("No go tool chain found: " + err.Error())
	}
	out, err := exec.Command(goBin, "version").CombinedOutput()
	if err != nil {
		panic("Getting compiler version failed: " + string(out))
	}
	return strings.Split(string(out), " ")[2]
}
