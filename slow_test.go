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
// +build slow_test

package gore

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
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
	files   map[string]string
	filesMu sync.RWMutex
}

func (f *testFiles) get(os, arch string, pie bool) string {
	f.filesMu.Lock()
	defer f.filesMu.Unlock()
	if pie {
		return f.files[os+arch+"-pie"]
	}
	return f.files[os+arch]
}

func TestMain(m *testing.M) {
	fmt.Println("Creating test resources, this can take some time...")
	tmpDirs := make([]string, len(dynResources))
	fs := make(map[string]string)
	for i, r := range dynResources {
		fmt.Printf("Building resource file for %s_%s\n", r.os, r.arch)
		exe, dir := buildTestResource(testresourcesrc, r.os, r.arch, false)
		tmpDirs[i] = dir
		fs[r.os+r.arch] = exe

		// Build PIE version of the file. Not all host systems, particular macOS, appears to be able
		// to compile a PIE build of linux-386. In this case, we skip this combination.
		if !(r.arch == "386" && r.os == "linux") {
			exe, dir = buildTestResource(testresourcesrc, r.os, r.arch, true)
			tmpDirs[i] = dir
			fs[r.os+r.arch+"-pie"] = exe
		}
	}
	dynResourceFiles = &testFiles{files: fs}

	fmt.Println("Launching tests")
	code := m.Run()

	fmt.Println("Clean up test resources")
	for _, d := range tmpDirs {
		os.RemoveAll(d)
	}
	os.Exit(code)
}

func TestOpenAndCloseFile(t *testing.T) {
	for _, test := range dynResources {
		t.Run("open_"+test.os+"-"+test.arch, func(t *testing.T) {
			assert := assert.New(t)
			exe := dynResourceFiles.get(test.os, test.arch, false)

			f, err := Open(exe)
			assert.NoError(err)
			assert.NotNil(f)
			assert.NoError(f.Close())
		})

		t.Run("open_"+test.os+"-"+test.arch+"-pie", func(t *testing.T) {
			assert := assert.New(t)
			exe := dynResourceFiles.get(test.os, test.arch, true)
			if exe == "" {
				t.Skip("no PIE available")
			}

			f, err := Open(exe)
			assert.NoError(err)
			assert.NotNil(f)
			assert.NoError(f.Close())
		})
	}
}

func TestGetPackages(t *testing.T) {
	for _, test := range dynResources {
		t.Run("open_"+test.os+"-"+test.arch, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)
			exe := dynResourceFiles.get(test.os, test.arch, false)
			if exe == "" {
				t.Skip("no PIE available")
			}

			f, err := Open(exe)
			require.NoError(err)
			require.NotNil(f)
			defer f.Close()

			std, err := f.GetSTDLib()
			assert.NoError(err)
			assert.NotEmpty(std, "Should have a list of standard library packages.")

			_, err = f.GetGeneratedPackages()
			assert.NoError(err)
			// XXX: This check appears to be unstable. Sometimes files for unknown reason.
			// assert.NotEmpty(gen, "Should have a list of generated packages.")

			ven, err := f.GetVendors()
			assert.NoError(err)
			assert.Empty(ven, "Should not have a list of vendor packages.")

			_, err = f.GetUnknown()
			assert.NoError(err)
			// XXX: This check appears to be unstable. Sometimes files for unknown reason.
			// assert.Empty(unk, "Should not have a list of unknown packages")

			pkgs, err := f.GetPackages()
			assert.NoError(err)

			var mainpkg *Package
			for _, p := range pkgs {
				if p.Name == "main" {
					mainpkg = p
					break
				}
			}

			mp := false
			gd := false
			assert.NotNil(mainpkg, "Should include main package")
			for _, f := range mainpkg.Functions {
				if f.Name == "main" {
					mp = true
				} else if f.Name == "getData" {
					gd = true
				} else {
					assert.Fail("Unexpected function")
				}
			}
			assert.True(mp, "No main function found")
			assert.True(gd, "getData function not found")
		})

		t.Run("open_"+test.os+"-"+test.arch+"-pie", func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)
			exe := dynResourceFiles.get(test.os, test.arch, false)
			f, err := Open(exe)
			require.NoError(err)
			require.NotNil(f)
			defer f.Close()

			std, err := f.GetSTDLib()
			assert.NoError(err)
			assert.NotEmpty(std, "Should have a list of standard library packages.")

			_, err = f.GetGeneratedPackages()
			assert.NoError(err)
			// XXX: This check appears to be unstable. Sometimes files for unknown reason.
			// assert.NotEmpty(gen, "Should have a list of generated packages.")

			ven, err := f.GetVendors()
			assert.NoError(err)
			assert.Empty(ven, "Should not have a list of vendor packages.")

			_, err = f.GetUnknown()
			assert.NoError(err)
			// XXX: This check appears to be unstable. Sometimes files for unknown reason.
			// assert.Empty(unk, "Should not have a list of unknown packages")

			pkgs, err := f.GetPackages()
			assert.NoError(err)

			var mainpkg *Package
			for _, p := range pkgs {
				if p.Name == "main" {
					mainpkg = p
					break
				}
			}

			mp := false
			gd := false
			assert.NotNil(mainpkg, "Should include main package")
			for _, f := range mainpkg.Functions {
				if f.Name == "main" {
					mp = true
				} else if f.Name == "getData" {
					gd = true
				} else {
					assert.Fail("Unexpected function")
				}
			}
			assert.True(mp, "No main function found")
			assert.True(gd, "getData function not found")
		})
	}
}

func TestGetTypesFromDynamicBuiltResources(t *testing.T) {
	for _, test := range dynResources {
		t.Run("open_"+test.os+"-"+test.arch, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)
			exe := dynResourceFiles.get(test.os, test.arch, false)
			f, err := Open(exe)
			require.NoError(err)
			require.NotNil(f)
			defer f.Close()

			typs, err := f.GetTypes()
			require.NoError(err)

			var stringer *GoType
			for _, t := range typs {
				if t.PackagePath == "runtime" && t.Name == "runtime.g" {
					stringer = t
					break
				}
			}

			assert.NotNil(stringer, "the g type from runtime not found")
		})

		t.Run("open_"+test.os+"-"+test.arch+"-pie", func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)
			exe := dynResourceFiles.get(test.os, test.arch, true)
			if exe == "" {
				t.Skip(("PIE file not available"))
			}

			f, err := Open(exe)
			require.NoError(err)
			require.NotNil(f)
			defer f.Close()

			typs, err := f.GetTypes()
			require.NoError(err)

			var stringer *GoType
			for _, t := range typs {
				if t.PackagePath == "runtime" && t.Name == "runtime.g" {
					stringer = t
					break
				}
			}

			assert.NotNil(stringer, "the g type from runtime not found")
		})
	}
}

func TestGetCompilerVersion(t *testing.T) {
	testVersion := testCompilerVersion()
	expectedVersion := ResolveGoVersion(testVersion)

	// If the version could not be resolved, the version is new
	// and the library doesn't know about it. Use the version string
	// to created a new version.
	if expectedVersion == nil {
		expectedVersion = &GoVersion{Name: testVersion}
	}

	for _, test := range dynResources {
		t.Run("parsing_"+test.os+"-"+test.arch, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)
			exe := dynResourceFiles.get(test.os, test.arch, false)
			f, err := Open(exe)
			require.NoError(err)
			require.NotNil(f)
			defer f.Close()

			// Test
			version, err := f.GetCompilerVersion()
			assert.NoError(err)
			assert.Equal(expectedVersion, version)
		})

		t.Run("parsing_"+test.os+"-"+test.arch+"-pie", func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)
			exe := dynResourceFiles.get(test.os, test.arch, true)
			if exe == "" {
				t.Skip("no PIE available")
			}

			f, err := Open(exe)
			require.NoError(err)
			require.NotNil(f)
			defer f.Close()

			// Test
			version, err := f.GetCompilerVersion()
			assert.NoError(err)
			assert.Equal(expectedVersion, version)
		})
	}
}

func TestGetBuildID(t *testing.T) {
	for _, test := range dynResources {
		t.Run("buildID_"+test.os+"-"+test.arch, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)
			exe := dynResourceFiles.get(test.os, test.arch, false)
			f, err := Open(exe)
			require.NoError(err)
			require.NotNil(f)
			defer f.Close()

			assert.Equal(fixedBuildID, f.BuildID, "BuildID extracted doesn't match expected value.")
		})

		t.Run("buildID_"+test.os+"-"+test.arch+"-pie", func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)
			exe := dynResourceFiles.get(test.os, test.arch, true)
			if exe == "" {
				t.Skip("no PIE available")
			}

			f, err := Open(exe)
			require.NoError(err)
			require.NotNil(f)
			defer f.Close()

			assert.Equal(fixedBuildID, f.BuildID, "BuildID extracted doesn't match expected value.")
		})
	}
}

func TestSourceInfo(t *testing.T) {
	for _, test := range dynResources {
		t.Run("buildID_"+test.os+"-"+test.arch, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)
			exe := dynResourceFiles.get(test.os, test.arch, false)
			f, err := Open(exe)
			require.NoError(err)
			require.NotNil(f)
			defer f.Close()

			var testFn *Function
			pkgs, err := f.GetPackages()
			require.NoError(err)
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
			require.NotNil(testFn)

			file, start, end := f.SourceInfo(testFn)

			assert.NotEqual(0, start)
			assert.NotEqual(0, end)
			assert.NotEqual("", file)
		})

		t.Run("buildID_"+test.os+"-"+test.arch+"-pie", func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)
			exe := dynResourceFiles.get(test.os, test.arch, true)
			if exe == "" {
				t.Skip("no PIE available")
			}

			f, err := Open(exe)
			require.NoError(err)
			require.NotNil(f)
			defer f.Close()

			var testFn *Function
			pkgs, err := f.GetPackages()
			require.NoError(err)
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
			require.NotNil(testFn)

			file, start, end := f.SourceInfo(testFn)

			assert.NotEqual(0, start)
			assert.NotEqual(0, end)
			assert.NotEqual("", file)
		})
	}
}

func buildTestResource(body, goos, arch string, pie bool) (string, string) {
	goBin, err := exec.LookPath("go")
	if err != nil {
		panic("No go tool chain found: " + err.Error())
	}

	tmpdir, err := ioutil.TempDir("", "TestGORE")
	if err != nil {
		panic(err)
	}

	src := filepath.Join(tmpdir, "a.go")
	err = ioutil.WriteFile(src, []byte(body), 0644)
	if err != nil {
		panic(err)
	}

	exe := filepath.Join(tmpdir, "a")
	if pie {
		exe = exe + "-pie"
	}
	args := []string{"build", "-o", exe, "-ldflags", "-s -w -buildid=" + fixedBuildID}
	if pie {
		args = append(args, "-buildmode=pie")
	}
	args = append(args, src)

	cmd := exec.Command(goBin, args...)
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = tmpdir
	}

	cmd.Env = append(cmd.Env, "GOCACHE="+tmpdir, "GOARCH="+arch, "GOOS="+goos, "GOPATH="+gopath, "PATH="+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		panic("building test executable failed: " + string(out))
	}

	return exe, tmpdir
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
